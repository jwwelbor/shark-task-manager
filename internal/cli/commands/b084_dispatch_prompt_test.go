package commands

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Exercise the shipped routes and templates through keyed next and the
// in-process runner. The regression is the instruction the worker receives.
func TestBugDispatchRequiresDecisionCoverage(t *testing.T) {
	const key = "B123"
	resetB036TemplateState()
	t.Cleanup(resetB036TemplateState)
	t.Cleanup(config.ClearWorkflowCache)

	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{}\n"), 0600))
	output, err := exec.Command("git", "init", "--quiet", projectDir).CombinedOutput()
	require.NoError(t, err, "%s", output)
	t.Chdir(projectDir)

	actionRoot, err := config.NewActionService(configPath)
	require.NoError(t, err)
	actionSvc := actionRoot.ForEntity("bug")
	workflowSvc := workflow.NewService(projectDir)
	bugWorkflow := workflowSvc.ForLevel("bug").GetWorkflow()
	plan := runner.ApprovedPlan{
		Pointer: "docs/plan/bugs/B123.repair-plan.md",
		Bytes:   []byte("Coverage index: happy-path regression only.\n"),
	}
	loadPlan := func(_ context.Context, bugKey, _ string) (runner.ApprovedPlan, error) {
		require.Equal(t, key, bugKey)
		return plan, nil
	}

	for _, tc := range []struct {
		status string
		want   []string
	}{
		{
			status: "repair_plan",
			want:   []string{"Add every applicable decision constraint to the coverage index", "testable assertion or an authorized disposition"},
		},
		{
			status: "plan_review",
			want:   []string{"Independently reconcile the decision sources with the coverage index", "Return `replan` for an omitted applicable constraint"},
		},
		{
			status: "development",
			want: []string{
				"new, existing, and resumed fixes",
				"Record the assertion and observed result for each applicable constraint",
				"Return `replan` when an omitted or changed constraint requires a material plan update",
				"even if the regression tests and project quality gate pass",
				"Every applicable decision constraint has passing verification evidence or an authorized disposition",
			},
		},
	} {
		t.Run(tc.status, func(t *testing.T) {
			step, ok := bugWorkflow.GetStep(tc.status)
			require.True(t, ok)
			if tc.status == "development" || tc.status == "plan_review" {
				require.Equal(t, "repair_plan", step.Outcomes["replan"])
			}
			transitioner := fixedNextTransitioner{info: &services.NextStatusInfo{
				CurrentStatus: tc.status, ResultContract: step.ResultContract,
			}}
			generator := fixedNextPlaceholders{vars: map[string]string{
				"id": key, "title": "Fixture bug", "file_path": "docs/plan/bugs/B123.md",
			}}
			checkPrompt := func(t *testing.T, prompt string) {
				t.Helper()
				want := []string{
					"`shark related-docs list " + key + " --json`",
					"Use the parent project command context",
					"Follow explicit references in the bug, research report, and repair plan when available",
					"Read the decision briefs and relevant prior findings",
					"An empty related-document list still requires checking explicit references",
					"If you find no applicable constraints, continue with the other requirements for this stage",
					"Return `blocked` if you cannot complete the lookup or read a required source",
				}
				for _, instruction := range append(want, tc.want...) {
					assert.Truef(t, strings.Contains(prompt, instruction), "dispatch is missing %q", instruction)
				}
				if tc.status == "development" {
					assert.Contains(t, prompt, string(plan.Bytes))
				}
			}

			t.Run("next", func(t *testing.T) {
				cache := &nextAdapterCache{
					entries: map[string]*nextAdapters{
						"bug": {transitioner: transitioner, generator: generator, actionSvc: actionSvc},
					},
					bugWorkflow:     bugWorkflow,
					approvedBugPlan: loadPlan,
				}
				out, err := runHarnessNextCommand(t, cache, []string{key})
				require.NoError(t, err)
				var resp NextResponse
				require.NoError(t, json.Unmarshal([]byte(out), &resp))
				checkPrompt(t, resp.Prompt)
			})

			t.Run("run", func(t *testing.T) {
				dispatcher := &b036RecordingDispatcher{}
				controller, err := runner.NewRunController(runner.RunControllerDeps{
					Transitioner:    transitioner,
					Placeholders:    generator,
					ActionSvc:       actionSvc,
					WorkflowSvc:     workflowSvc,
					Dispatchers:     map[string]runner.AgentDispatcher{step.Provider: dispatcher},
					ApprovedBugPlan: loadPlan,
					PromptAssembler: runner.PromptAssemblerFunc(func(_ context.Context, input runner.PromptAssemblyInput) (string, error) {
						prompt, err := assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
						if err != nil {
							return "", err
						}
						return bugParentProjectInstructions(prompt)
					}),
				})
				require.NoError(t, err)
				_, err = controller.Run(context.Background(), key, runner.RunOptions{EntityType: "bug", WorkingDir: projectDir})
				require.NoError(t, err)
				require.Len(t, dispatcher.inputs, 1)
				checkPrompt(t, dispatcher.inputs[0].Instruction)
			})
		})
	}
}
