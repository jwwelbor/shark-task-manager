package commands

// Covers T-E34-F01-005 (docs/plan/E34-prompt-and-skill-improvements/E34-F01-harness-aware-prompt-rendering/tasks/T-E34-F01-005.md):
// TC-009, TC-010, TC-011 from the feature test-plan.md — AC-08 next/run
// parity at all three precedence tiers (flag, claim, env).
//
// Per test-plan.md's Caller-Path Contract for these TCs: mock
// EntityTransitioner, PlaceholderGenerator, and ClaimReader only — never
// HarnessResolver.Resolve itself — and drive each surface through its own
// real entrypoint: `runNext` (next.go, via runHarnessNextCommand) for
// `shark next`, and `runner.RunController.Run` directly for `shark run`.
// This file is the only place in the test suite where both entrypoints are
// reachable from the same package, since runNext lives in package commands
// and the parity assertion needs to call both.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/require"
)

// parityDispatcher captures the assembled prompt handed to DispatchInput and
// then fails the dispatch, so RunController.Run stops after exactly one
// Step-3/4 pass without needing TransitionStatus or a second iteration to be
// mocked — the assembled prompt is already captured by the time Dispatch is
// called (controller.go's handleSpawnAgent builds it before dispatching).
type parityDispatcher struct {
	captured *string
}

func (d *parityDispatcher) Dispatch(ctx context.Context, input runner.DispatchInput) (*runner.DispatchResult, error) {
	*d.captured = input.Instruction
	return nil, parityDispatchStop("parity test: stop after capturing the assembled prompt")
}

func (d *parityDispatcher) Name() string { return "parity-mock" }

func (d *parityDispatcher) BuildCommand(input runner.DispatchInput) (string, error) {
	return "parity-mock-cmd", nil
}

type parityDispatchStop string

func (e parityDispatchStop) Error() string { return string(e) }

var _ runner.AgentDispatcher = (*parityDispatcher)(nil)

// renderViaRunController drives runner.RunController.Run directly (the real
// `shark run` entrypoint per the Caller-Path Contract) against the same
// entity/claim/step fixture harnessTestCache builds for the `next` side, and
// returns the assembled prompt captured from DispatchInput.Instruction.
func renderViaRunController(t *testing.T, claims services.ClaimReader, templateBody string, override services.HarnessIdentity) string {
	t.Helper()
	renderer, templateName := harnessDispatchRenderer(t, templateBody)

	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*action.PopulatedAction, error) {
			rendered, err := renderer.Render(templateName, vars)
			if err != nil {
				return nil, err
			}
			return &action.PopulatedAction{Action: "spawn_agent", Instruction: rendered}, nil
		},
	}

	var captured string
	ctrl, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner: fixedNextTransitioner{info: &services.NextStatusInfo{
			CurrentStatus: "in_progress",
			IsTerminal:    false,
		}},
		Placeholders: fixedNextPlaceholders{vars: map[string]string{}},
		ActionSvc:    actionSvc,
		WorkflowSvc:  workflow.NewService(""),
		Dispatchers:  map[string]runner.AgentDispatcher{"": &parityDispatcher{captured: &captured}},
		PromptAssembler: runner.PromptAssemblerFunc(func(ctx context.Context, input runner.PromptAssemblyInput) (string, error) {
			return assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
		}),
		HarnessResolver: services.NewHarnessResolver(claims),
	})
	require.NoError(t, err)

	_, err = ctrl.Run(context.Background(), "E01-F01-001", runner.RunOptions{
		EntityType:      "task",
		HarnessOverride: override,
	})
	// parityDispatchStop is expected: it's how the loop stops after one pass.
	// Any other error means the harness-resolution/render path itself failed.
	if err != nil {
		t.Fatalf("RunController.Run() returned unexpected error: %v", err)
	}
	require.NotEmpty(t, captured, "dispatcher must have captured an assembled prompt before RunController.Run returned")
	return captured
}

// renderViaNext drives runNext (the real `shark next` entrypoint) against the
// harnessTestCache fixture and returns the rendered prompt from NextResponse.
func renderViaNext(t *testing.T, claims services.ClaimReader, templateBody string, args []string) string {
	t.Helper()
	cache := harnessTestCache(t, claims, templateBody)
	stdout, err := runHarnessNextCommand(t, cache, args)
	require.NoError(t, err)

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	return resp.Prompt
}

// TestTC009_NextRunParity_FlagTier covers TC-009: `--harness=claude
// --harness-version=2.1.0 --harness-model=opus` passed to both surfaces
// produces byte-identical rendered prompts.
func TestTC009_NextRunParity_FlagTier(t *testing.T) {
	unsetHarnessEnv(t)
	claims := &harnessMockClaimReader{claim: nil}
	override := services.HarnessIdentity{Type: "claude", Version: "2.1.0", Model: "opus"}

	nextPrompt := renderViaNext(t, claims, harnessIfTemplate, []string{
		"E01-F01-001", "--harness=claude", "--harness-version=2.1.0", "--harness-model=opus",
	})
	runPrompt := renderViaRunController(t, claims, harnessIfTemplate, override)

	require.Equal(t, nextPrompt, runPrompt, "AC-08: next/run prompts must be byte-identical at the flag tier")
	// Pin the branch, not just equality: two surfaces that both degraded to
	// the zero identity would render identical (branch B) prompts and pass
	// the Equal check above without either surface having resolved the flag.
	require.Contains(t, nextPrompt, harnessBranchA,
		"flag tier passes --harness=claude; both surfaces must render the claude branch")
}

// TestTC010_NextRunParity_ClaimTier covers TC-010: no override flags on
// either surface, harness sourced entirely from a claim (harness=codex).
func TestTC010_NextRunParity_ClaimTier(t *testing.T) {
	unsetHarnessEnv(t)
	claims := &harnessMockClaimReader{claim: &models.EntityClaim{Harness: "codex"}}

	nextPrompt := renderViaNext(t, claims, harnessIfTemplate, []string{"E01-F01-001"})
	runPrompt := renderViaRunController(t, claims, harnessIfTemplate, services.HarnessIdentity{})

	require.Equal(t, nextPrompt, runPrompt, "AC-08: next/run prompts must be byte-identical at the claim tier")
	require.Contains(t, nextPrompt, harnessBranchB, "claim tier fixture uses harness=codex, must render the generic branch")
}

// TestTC011_NextRunParity_EnvTier covers TC-011: entity unclaimed on both
// surfaces, SHARK_HARNESS=claude sourced from env for the duration of both
// calls.
func TestTC011_NextRunParity_EnvTier(t *testing.T) {
	unsetHarnessEnv(t)
	t.Setenv("SHARK_HARNESS", "claude")
	claims := &harnessMockClaimReader{claim: nil}

	nextPrompt := renderViaNext(t, claims, harnessIfTemplate, []string{"E01-F01-001"})
	runPrompt := renderViaRunController(t, claims, harnessIfTemplate, services.HarnessIdentity{})

	require.Equal(t, nextPrompt, runPrompt, "AC-08: next/run prompts must be byte-identical at the env tier")
	require.Contains(t, nextPrompt, harnessBranchA, "env tier fixture uses SHARK_HARNESS=claude, must render the claude branch")
}
