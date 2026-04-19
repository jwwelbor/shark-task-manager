// Package runner
//
// Stage-error coverage tests.
//
// These tests enforce the full observability contract for stage events:
//
//  1. run.stage.error MUST be emitted at every non-empty result.Error branch
//     in controller.go.  For each branch we drive a fixture that forces that
//     exact error and assert the event was emitted with the mandatory phase
//     + error fields.
//
//  2. Every run.stage.error event MUST carry "phase" and "error" attributes
//     in addition to the existing fields.
//
//  3. run.stage.dispatch MUST carry the REAL resolved CLI command (e.g. the
//     exact "claude ..." or "codex ..." string), NOT a placeholder.
//
//  4. run.stage.complete MUST include agent_type and provider.
//
//  5. The "truncated" field in run.stage.error MUST be OMITTED when no
//     truncation occurred, and present (and true) when truncation did occur.
//
// Golden rule (re-affirmed): no real database, no real agent, no filesystem.
// Pure mocks.
package runner

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// ---------------------------------------------------------------------------
// Local helpers
// ---------------------------------------------------------------------------

// firstStageError returns the first run.stage.error event in buf. Fatals the
// test if no such event is found.
func firstStageError(t *testing.T, events []map[string]any) map[string]any {
	t.Helper()
	errs := eventsByMsg(events, "run.stage.error")
	if len(errs) < 1 {
		t.Fatalf("expected at least 1 run.stage.error event, got 0 (all events: %+v)", events)
	}
	return errs[0]
}

// requireNonEmptyString asserts that key is present on ev and is a non-empty
// string. Returns the value on success; calls t.Errorf and returns "" on failure.
func requireNonEmptyString(t *testing.T, ev map[string]any, key, context string) string {
	t.Helper()
	v, ok := ev[key]
	if !ok {
		t.Errorf("%s: attribute %q missing", context, key)
		return ""
	}
	s, ok := v.(string)
	if !ok {
		t.Errorf("%s: attribute %q has wrong type %T, want string", context, key, v)
		return ""
	}
	if s == "" {
		t.Errorf("%s: attribute %q is empty string", context, key)
	}
	return s
}

// ---------------------------------------------------------------------------
// Silent-branch coverage — each test below forces a specific result.Error
// branch in controller.go to fire, and asserts run.stage.error was emitted
// with the right phase.
// ---------------------------------------------------------------------------

// TestController_EmitsStageError_ContextCancelled exercises controller.go line
// 230 — the top-of-loop ctx.Done() branch. Previously: silent.
func TestController_EmitsStageError_ContextCancelled(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	ctrl := makeController(t, transitioner, &MockActionService{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel BEFORE Run, so the ctx.Done() branch fires

	opts := RunOptions{RunID: "run-ctx", Observability: obsEnabled(0)}
	result, err := ctrl.Run(ctx, "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (ctx)")
	if phase != "context" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "context")
	}
	requireNonEmptyString(t, ev, "error", "stage.error (ctx)")
}

// TestController_EmitsStageError_PlaceholderFailure exercises controller.go line
// 254 — the placeholder-generation failure branch.
func TestController_EmitsStageError_PlaceholderFailure(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	placeholders := &MockPlaceholderGen{
		GenerateFunc: func(ctx context.Context, key string) (map[string]string, error) {
			return nil, fmt.Errorf("boom: placeholder generation failed")
		},
	}
	deps := RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: placeholders,
		ActionSvc:    &MockActionService{},
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  map[string]AgentDispatcher{"": &MockDispatcher{}},
	}
	ctrl, err := NewRunController(deps)
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	opts := RunOptions{RunID: "run-ph", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (placeholder)")
	if phase != "placeholders" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "placeholders")
	}
	errStr := requireNonEmptyString(t, ev, "error", "stage.error (placeholder)")
	if !strings.Contains(errStr, "boom") {
		t.Errorf("stage.error: error = %q, want substring %q", errStr, "boom")
	}
}

// TestController_EmitsStageError_ActionLookupFailure exercises controller.go line
// 265 — the GetStatusActionPopulated failure branch.
func TestController_EmitsStageError_ActionLookupFailure(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return nil, fmt.Errorf("action-lookup-failed")
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, nil)

	opts := RunOptions{RunID: "run-al", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (action lookup)")
	if phase != "action_lookup" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "action_lookup")
	}
	errStr := requireNonEmptyString(t, ev, "error", "stage.error (action lookup)")
	if !strings.Contains(errStr, "action-lookup-failed") {
		t.Errorf("stage.error: error = %q, want substring %q", errStr, "action-lookup-failed")
	}
}

// TestController_EmitsStageError_UnknownAction exercises controller.go line
// 308 — the default case (unknown action type).
func TestController_EmitsStageError_UnknownAction(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: "completely_made_up_action"}, nil
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, nil)

	opts := RunOptions{RunID: "run-ua", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (unknown action)")
	if phase != "unknown_action" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "unknown_action")
	}
	requireNonEmptyString(t, ev, "error", "stage.error (unknown action)")
}

// TestController_EmitsStageError_AdvanceStatusDryRun_NextStatusFailure exercises
// controller.go line 332 — the dry-run GetNextStatus failure inside
// handleAdvanceStatus.
func TestController_EmitsStageError_AdvanceStatusDryRun_NextStatusFailure(t *testing.T) {
	buf := captureSlog(t)

	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				// Outer Run() call: succeed so we reach the loop.
				return &services.NextStatusInfo{CurrentStatus: "ready_for_review", IsTerminal: false}, nil
			}
			// Nested call inside handleAdvanceStatus: fail.
			return nil, fmt.Errorf("ns-lookup-failed")
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionAdvanceStatus}, nil
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, nil)

	opts := RunOptions{RunID: "run-as-dry", DryRun: true, Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (advance_status dry-run)")
	if phase != "advance_status" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "advance_status")
	}
	requireNonEmptyString(t, ev, "error", "stage.error (advance_status dry-run)")
}

// TestController_EmitsStageError_AdvanceStatusLive_NextStatusFailure exercises
// controller.go line 357 — the live-mode GetNextStatus failure inside
// handleAdvanceStatus.
func TestController_EmitsStageError_AdvanceStatusLive_NextStatusFailure(t *testing.T) {
	buf := captureSlog(t)

	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				return &services.NextStatusInfo{CurrentStatus: "ready_for_review", IsTerminal: false}, nil
			}
			return nil, fmt.Errorf("ns-live-failed")
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionAdvanceStatus}, nil
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, nil)

	opts := RunOptions{RunID: "run-as-live", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (advance_status live)")
	if phase != "advance_status" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "advance_status")
	}
	errStr := requireNonEmptyString(t, ev, "error", "stage.error (advance_status live)")
	if !strings.Contains(errStr, "ns-live-failed") {
		t.Errorf("stage.error: error = %q, want substring %q", errStr, "ns-live-failed")
	}
}

// TestController_EmitsStageError_AdvanceStatusTransitionFailure exercises
// controller.go line 374 — the TransitionStatus failure inside handleAdvanceStatus.
func TestController_EmitsStageError_AdvanceStatusTransitionFailure(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "ready_for_review",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return nil, fmt.Errorf("transition-refused")
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionAdvanceStatus}, nil
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, nil)

	opts := RunOptions{RunID: "run-as-tr", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (advance transition)")
	if phase != "transition" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "transition")
	}
	errStr := requireNonEmptyString(t, ev, "error", "stage.error (advance transition)")
	if !strings.Contains(errStr, "transition-refused") {
		t.Errorf("stage.error: error = %q, want substring %q", errStr, "transition-refused")
	}
}

// TestController_EmitsStageError_DispatcherSelectionFailure exercises
// controller.go line 416 — selectDispatcher returning an error.
func TestController_EmitsStageError_DispatcherSelectionFailure(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:    config.ActionSpawnAgent,
				AgentType: "developer",
				Provider:  "nonexistent-provider",
			}, nil
		},
	}
	// Only "" is wired, so "nonexistent-provider" will miss.
	dispatchers := map[string]AgentDispatcher{"": &MockDispatcher{}}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-disp", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (dispatcher selection)")
	if phase != "dispatcher_selection" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "dispatcher_selection")
	}
	errStr := requireNonEmptyString(t, ev, "error", "stage.error (dispatcher selection)")
	if !strings.Contains(errStr, "nonexistent-provider") {
		t.Errorf("stage.error: error = %q, want substring %q", errStr, "nonexistent-provider")
	}
}

// TestController_EmitsStageError_PostDispatchNextStatusFailure exercises
// controller.go line 550 — the GetNextStatus failure AFTER a successful agent
// dispatch inside handleSpawnAgent.
func TestController_EmitsStageError_PostDispatchNextStatusFailure(t *testing.T) {
	buf := captureSlog(t)

	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
			}
			// Post-dispatch: fail.
			return nil, fmt.Errorf("post-dispatch-ns-failed")
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 0, Command: "claude --ok"}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-post-ns", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (post-dispatch next_status)")
	if phase != "post_dispatch" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "post_dispatch")
	}
	errStr := requireNonEmptyString(t, ev, "error", "stage.error (post-dispatch next_status)")
	if !strings.Contains(errStr, "post-dispatch-ns-failed") {
		t.Errorf("stage.error: error = %q, want substring %q", errStr, "post-dispatch-ns-failed")
	}
}

// TestController_DryRun_PostDispatchFailure_EmitsError exercises the dry-run
// branch of handleSpawnAgent (controller.go ~line 486) when the post-dispatch
// GetNextStatus call fails. Prior to this rework, the dry-run branch silently
// swallowed that failure as Outcome="completed", hiding the transitioner
// error from operators. The fix mirrors the real-path behaviour (line ~632):
//
//   - emit exactly one run.stage.error with phase="post_dispatch" and a
//     non-empty error attribute carrying the transitioner failure text;
//   - set result.Outcome="failed" and result.Error to the failure text;
//   - NOT invoke dispatcher.Dispatch — dry-run never dispatches.
//
// This test guards against regressing to the silent-swallow behaviour.
func TestController_DryRun_PostDispatchFailure_EmitsError(t *testing.T) {
	buf := captureSlog(t)

	dispatchCalls := 0
	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				// First call (top of the run loop) must succeed so the
				// controller enters handleSpawnAgent and reaches the
				// dry-run post-dispatch GetNextStatus call.
				return &services.NextStatusInfo{
					CurrentStatus: "in_development",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					},
				}, nil
			}
			// Second call (post-dispatch, inside the dry-run branch) fails.
			return nil, fmt.Errorf("dry-run-post-dispatch-ns-failed")
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:    config.ActionSpawnAgent,
				AgentType: "developer",
				Provider:  "anthropic",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{
			// Dry-run must NEVER call Dispatch. If this runs, the counter
			// trips the assertion below.
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				dispatchCalls++
				return &DispatchResult{ExitCode: 0, Command: "claude --ok"}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{
		RunID:         "run-dryrun-post-ns",
		DryRun:        true,
		Observability: obsEnabled(0),
	}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Dry-run must NOT dispatch.
	if dispatchCalls != 0 {
		t.Errorf("Dispatch was called %d time(s), want 0 — dry-run must not invoke the dispatcher", dispatchCalls)
	}

	// Run result must reflect the post-dispatch failure (previously this
	// was the silent Outcome="completed" bug the rework fixes).
	if result.Outcome != "failed" {
		t.Errorf("result.Outcome = %q, want %q (dry-run must surface post-dispatch next_status failure)", result.Outcome, "failed")
	}
	if !strings.Contains(result.Error, "dry-run-post-dispatch-ns-failed") {
		t.Errorf("result.Error = %q, want substring %q", result.Error, "dry-run-post-dispatch-ns-failed")
	}

	events := parseEvents(t, buf)

	// Exactly one run.stage.error event with phase="post_dispatch".
	errorEvents := eventsByMsg(events, "run.stage.error")
	if len(errorEvents) != 1 {
		t.Fatalf("expected 1 run.stage.error event, got %d (events: %+v)", len(errorEvents), events)
	}
	ev := errorEvents[0]
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (dry-run post_dispatch)")
	if phase != "post_dispatch" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "post_dispatch")
	}
	errStr := requireNonEmptyString(t, ev, "error", "stage.error (dry-run post_dispatch)")
	if !strings.Contains(errStr, "dry-run-post-dispatch-ns-failed") {
		t.Errorf("stage.error: error = %q, want substring %q", errStr, "dry-run-post-dispatch-ns-failed")
	}
}

// TestController_EmitsStageError_PostDispatchTransitionFailure exercises
// controller.go line 610 — the TransitionStatus failure AFTER a successful
// agent dispatch inside handleSpawnAgent.
func TestController_EmitsStageError_PostDispatchTransitionFailure(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return nil, fmt.Errorf("post-dispatch-transition-refused")
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 0, Command: "claude --ok"}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-post-tr", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (post-dispatch transition)")
	if phase != "transition" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "transition")
	}
	errStr := requireNonEmptyString(t, ev, "error", "stage.error (post-dispatch transition)")
	if !strings.Contains(errStr, "post-dispatch-transition-refused") {
		t.Errorf("stage.error: error = %q, want substring %q", errStr, "post-dispatch-transition-refused")
	}
}

// ---------------------------------------------------------------------------
// run.stage.error MUST carry phase + error on the agent-failure paths.
// ---------------------------------------------------------------------------

// TestController_StageError_AgentFailure_HasPhaseAndError verifies the existing
// AgentFailedError path (dispatcher returns both result + *AgentFailedError)
// now includes phase + error attributes.
func TestController_StageError_AgentFailure_HasPhaseAndError(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 7, Stderr: "oops", Command: "c --x"},
					&AgentFailedError{ExitCode: 7, Stderr: "oops", Command: "c --x"}
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-agf", Observability: obsEnabled(0)}
	_, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (agent fail)")
	if phase != "dispatch" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "dispatch")
	}
	requireNonEmptyString(t, ev, "error", "stage.error (agent fail)")
}

// TestController_StageError_NonZeroExit_HasPhaseAndError verifies the mock-style
// path (dispatcher returns result with non-zero exit, no error) also includes
// phase + error attributes.
func TestController_StageError_NonZeroExit_HasPhaseAndError(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 3, Stderr: "err3", Command: "c --y"}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-nz", Observability: obsEnabled(0)}
	_, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (non-zero exit)")
	if phase != "dispatch" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "dispatch")
	}
	requireNonEmptyString(t, ev, "error", "stage.error (non-zero exit)")
}

// ---------------------------------------------------------------------------
// run.stage.dispatch MUST carry the REAL resolved CLI command,
// NOT a placeholder.
// ---------------------------------------------------------------------------

// TestController_StageDispatch_UsesRealCommand verifies that run.stage.dispatch
// carries the real command string returned by dispatcher.BuildCommand(input),
// not a placeholder string.
func TestController_StageDispatch_UsesRealCommand(t *testing.T) {
	buf := captureSlog(t)

	const wantCmd = "claude -p do-work --output-format json"

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do-work",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{
			BuildCommandFunc: func(input DispatchInput) (string, error) { return wantCmd, nil },
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 0, Command: wantCmd}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-realcmd", Observability: obsEnabled(0)}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	dispatchEvents := eventsByMsg(parseEvents(t, buf), "run.stage.dispatch")
	if len(dispatchEvents) != 1 {
		t.Fatalf("expected 1 run.stage.dispatch event, got %d", len(dispatchEvents))
	}
	gotCmd, _ := dispatchEvents[0]["command"].(string)
	if gotCmd != wantCmd {
		t.Errorf("stage.dispatch: command = %q, want %q", gotCmd, wantCmd)
	}
	if strings.Contains(gotCmd, "dispatch ") && strings.Contains(gotCmd, " for ") && strings.Contains(gotCmd, "@") {
		t.Errorf("stage.dispatch: command %q still looks like a placeholder", gotCmd)
	}
}

// ---------------------------------------------------------------------------
// BuildCommand failures (e.g. NUL byte in argv) MUST surface as a structured
// run.stage.error with phase="shell_quote" and MUST NOT invoke
// dispatcher.Dispatch. os/exec would reject NUL-bearing argv with EINVAL
// anyway, so the dispatch is doomed — but without this error path the run
// controller would silently fall through to Dispatch and surface a generic
// subprocess failure instead of a self-describing signal.
// ---------------------------------------------------------------------------

// TestController_BuildCommandFailure_EmitsShellQuoteError verifies that when
// dispatcher.BuildCommand returns an error, the controller:
//   - Emits a run.stage.error with phase="shell_quote" and a non-empty
//     error attribute derived from the BuildCommand error.
//   - Does NOT emit a run.stage.dispatch (the dispatch is aborted before
//     logging the invocation — the command is unrunnable).
//   - Does NOT call dispatcher.Dispatch.
//   - Sets the run result.Outcome="failed" and result.Error to the build
//     error message, so the caller sees the same failure the logs describe.
func TestController_BuildCommandFailure_EmitsShellQuoteError(t *testing.T) {
	buf := captureSlog(t)

	dispatchCalls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do-work\x00contains-nul", // triggers NUL via real dispatcher; mock just returns error below
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{
			BuildCommandFunc: func(input DispatchInput) (string, error) {
				return "", errShellQuoteNUL
			},
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				dispatchCalls++
				return &DispatchResult{ExitCode: 0}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-nul", Observability: obsEnabled(0)}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Dispatch MUST NOT be called.
	if dispatchCalls != 0 {
		t.Errorf("Dispatch was called %d time(s), want 0 — controller must skip Dispatch when BuildCommand fails", dispatchCalls)
	}

	// Run result reflects the shell-quote failure.
	if result.Outcome != "failed" {
		t.Errorf("result.Outcome = %q, want %q", result.Outcome, "failed")
	}
	if !strings.Contains(result.Error, "NUL") {
		t.Errorf("result.Error = %q, want substring %q", result.Error, "NUL")
	}

	events := parseEvents(t, buf)

	// No run.stage.dispatch event — BuildCommand failed, no invocation to log.
	dispatchEvents := eventsByMsg(events, "run.stage.dispatch")
	if len(dispatchEvents) != 0 {
		t.Errorf("expected 0 run.stage.dispatch events (BuildCommand failed), got %d", len(dispatchEvents))
	}

	// Exactly one run.stage.error event with phase="shell_quote".
	errorEvents := eventsByMsg(events, "run.stage.error")
	if len(errorEvents) != 1 {
		t.Fatalf("expected 1 run.stage.error event, got %d", len(errorEvents))
	}
	ev := errorEvents[0]
	phase := requireNonEmptyString(t, ev, "phase", "stage.error (shell_quote)")
	if phase != "shell_quote" {
		t.Errorf("stage.error: phase = %q, want %q", phase, "shell_quote")
	}
	errAttr := requireNonEmptyString(t, ev, "error", "stage.error (shell_quote)")
	if !strings.Contains(errAttr, "NUL") {
		t.Errorf("stage.error: error = %q, want substring %q", errAttr, "NUL")
	}
}

// ---------------------------------------------------------------------------
// run.stage.complete MUST include agent_type and provider.
// ---------------------------------------------------------------------------

// TestController_StageComplete_IncludesAgentTypeAndProvider verifies the
// run.stage.complete event carries both agent_type and provider, enabling
// dashboards to group stage completions by agent.
func TestController_StageComplete_IncludesAgentTypeAndProvider(t *testing.T) {
	buf := captureSlog(t)

	ctrl := happyPathFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		return &DispatchResult{ExitCode: 0, Command: "claude --ok"}, nil
	})
	opts := RunOptions{RunID: "run-cmplt", Observability: obsEnabled(0)}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	events := eventsByMsg(parseEvents(t, buf), "run.stage.complete")
	if len(events) != 1 {
		t.Fatalf("expected 1 run.stage.complete event, got %d", len(events))
	}
	ev := events[0]
	if at, _ := ev["agent_type"].(string); at != "developer" {
		t.Errorf("stage.complete: agent_type = %q, want %q", at, "developer")
	}
	if p, _ := ev["provider"].(string); p != "anthropic" {
		t.Errorf("stage.complete: provider = %q, want %q", p, "anthropic")
	}
}

// ---------------------------------------------------------------------------
// truncated field MUST be omitted when no truncation occurred.
// ---------------------------------------------------------------------------

// TestController_StageError_TruncatedOmittedWhenNoTruncation verifies that for
// short stderr/stdout payloads, the "truncated" attribute is absent from the
// emitted event. Emitting "truncated": false unconditionally would force log
// consumers to reason about a signal that conveys no information.
func TestController_StageError_TruncatedOmittedWhenNoTruncation(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{
						ExitCode: 1,
						Stdout:   "short",
						Stderr:   "short",
						Command:  "c",
					}, &AgentFailedError{
						ExitCode: 1,
						Stdout:   "short",
						Stderr:   "short",
						Command:  "c",
					}
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-notrunc", Observability: obsEnabled(0)}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	if _, ok := ev["truncated"]; ok {
		t.Errorf("stage.error: attribute \"truncated\" MUST be omitted when no truncation occurred, got present")
	}
}

// TestController_StageError_TruncatedPresentAndTrueWhenTruncated verifies that
// when the stderr/stdout payloads exceed LogTruncateBytes, the "truncated"
// attribute is present and equals true.
func TestController_StageError_TruncatedPresentAndTrueWhenTruncated(t *testing.T) {
	buf := captureSlog(t)

	const limit = 32
	big := strings.Repeat("y", 4096)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{
						ExitCode: 1,
						Stdout:   big,
						Stderr:   big,
						Command:  "c",
					}, &AgentFailedError{
						ExitCode: 1,
						Stdout:   big,
						Stderr:   big,
						Command:  "c",
					}
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-trunc2", Observability: obsEnabled(limit)}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	ev := firstStageError(t, parseEvents(t, buf))
	v, ok := ev["truncated"]
	if !ok {
		t.Fatal("stage.error: attribute \"truncated\" must be present when truncation occurred")
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("stage.error: attribute \"truncated\" has wrong type %T, want bool", v)
	}
	if !b {
		t.Error("stage.error: attribute \"truncated\" must be true when truncation occurred")
	}
}

// ---------------------------------------------------------------------------
// Shell-safety of the logged dispatch command.
//
// The command emitted on the run.stage.dispatch event must be not merely the
// real command (covered by TestController_StageDispatch_UsesRealCommand above)
// but SHELL-EQUIVALENT: pasting the string into a POSIX shell must reproduce
// exactly the argv that exec.Command passes to the OS.
//
// These subtests exercise the controller end-to-end with real
// ClaudeDispatcher / CodexDispatcher instances (no MockDispatcher) and
// inspect the `command` field on the emitted run.stage.dispatch event. We
// shell-tokenize it with shellSplitForTest and assert the recovered argv.
// ---------------------------------------------------------------------------

// runStageDispatchCapturesCommand runs the controller once against a real
// AgentDispatcher (claude or codex) with the given instruction and returns
// the value of the `command` attribute emitted on run.stage.dispatch. Any
// test-infrastructure failure (missing event, wrong type) is reported via t.
func runStageDispatchCapturesCommand(t *testing.T, d AgentDispatcher, provider, instruction string) string {
	t.Helper()

	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    provider,
				Instruction: instruction,
			}, nil
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, map[string]AgentDispatcher{
		provider: d,
	})

	opts := RunOptions{RunID: "run-shellsafe", Observability: obsEnabled(0)}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	dispatchEvents := eventsByMsg(parseEvents(t, buf), "run.stage.dispatch")
	if len(dispatchEvents) != 1 {
		t.Fatalf("expected 1 run.stage.dispatch event, got %d", len(dispatchEvents))
	}
	cmd, ok := dispatchEvents[0]["command"].(string)
	if !ok {
		t.Fatalf("stage.dispatch: command attribute missing or wrong type: %+v", dispatchEvents[0])
	}
	return cmd
}

// TestController_StageDispatch_ShellSafe_Claude exercises the full controller
// pipeline with the real ClaudeDispatcher and verifies that the logged
// command on run.stage.dispatch is POSIX-shell-equivalent for instructions
// containing spaces, quotes, and metacharacters. The MockDispatcher path
// cannot catch a regression in joinCommand; this path can.
func TestController_StageDispatch_ShellSafe_Claude(t *testing.T) {
	// Use real ClaudeDispatcher. No lookPath / cmdFactory overrides are
	// needed because the controller calls BuildCommand BEFORE Dispatch;
	// the logged string comes solely from BuildCommand. We substitute a
	// lookPathFunc so the (real) Dispatch path that follows does not
	// require the claude binary on the test host.
	newClaude := func() *ClaudeDispatcher {
		c := NewClaudeDispatcher()
		var cap capturedCmd
		c.cmdFactory = recordingFactory(&cap)
		c.lookPathFunc = successLookPath
		return c
	}

	t.Run("instruction with spaces round-trips", func(t *testing.T) {
		instruction := "do work now"
		got := runStageDispatchCapturesCommand(t, newClaude(), "anthropic", instruction)

		if !strings.Contains(got, "'do work now'") {
			t.Errorf("expected instruction to be single-quoted in logged command, got: %q", got)
		}
		tokens, err := shellSplitForTest(got)
		if err != nil {
			t.Fatalf("logged command is not shell-tokenizable: %v (%q)", err, got)
		}
		if !sliceContainsOrdered(tokens, []string{"-p", instruction}) {
			t.Errorf("argv recovered from logged command did not contain %q as a single %q arg.\n  tokens: %q\n  cmd:    %s",
				instruction, "-p", tokens, got)
		}
	})

	t.Run("instruction with single quote round-trips", func(t *testing.T) {
		instruction := "it's fine"
		got := runStageDispatchCapturesCommand(t, newClaude(), "anthropic", instruction)

		if !strings.Contains(got, `'it'\''s fine'`) {
			t.Errorf("expected single quote to be POSIX-escaped as '\\'' in logged command, got: %q", got)
		}
		tokens, err := shellSplitForTest(got)
		if err != nil {
			t.Fatalf("logged command is not shell-tokenizable: %v (%q)", err, got)
		}
		if !sliceContainsOrdered(tokens, []string{"-p", instruction}) {
			t.Errorf("argv recovered from logged command did not contain %q as a single %q arg.\n  tokens: %q\n  cmd:    %s",
				instruction, "-p", tokens, got)
		}
	})

	t.Run("instruction with shell metacharacters round-trips", func(t *testing.T) {
		instruction := `rm -rf /; echo "pwned" && cat $HOME`
		got := runStageDispatchCapturesCommand(t, newClaude(), "anthropic", instruction)

		tokens, err := shellSplitForTest(got)
		if err != nil {
			t.Fatalf("logged command is not shell-tokenizable: %v (%q)", err, got)
		}
		if !sliceContainsOrdered(tokens, []string{"-p", instruction}) {
			t.Errorf("argv recovered from logged command did not contain the raw instruction as a single %q arg.\n  tokens: %q\n  cmd:    %s",
				"-p", tokens, got)
		}
		// Defense-in-depth: the metacharacters must not appear at token
		// boundaries (which would indicate unquoted splitting).
		for _, meta := range []string{";", "&&", "|"} {
			for _, tok := range tokens {
				if tok == meta {
					t.Errorf("metacharacter %q appears as its own token — argv was split: tokens=%q cmd=%s",
						meta, tokens, got)
				}
			}
		}
	})
}

// TestController_StageDispatch_ShellSafe_Codex is the codex equivalent of
// the claude test above.
func TestController_StageDispatch_ShellSafe_Codex(t *testing.T) {
	newCodex := func() *CodexDispatcher {
		c := NewCodexDispatcher()
		var cap capturedCmd
		c.cmdFactory = recordingFactory(&cap)
		c.lookPathFunc = successLookPath
		return c
	}

	t.Run("instruction with spaces round-trips", func(t *testing.T) {
		instruction := "do work now"
		got := runStageDispatchCapturesCommand(t, newCodex(), "codex", instruction)

		if !strings.Contains(got, "'do work now'") {
			t.Errorf("expected instruction to be single-quoted in logged command, got: %q", got)
		}
		tokens, err := shellSplitForTest(got)
		if err != nil {
			t.Fatalf("logged command is not shell-tokenizable: %v (%q)", err, got)
		}
		// codex places the instruction as the LAST argv element.
		if len(tokens) < 2 || tokens[len(tokens)-1] != instruction {
			t.Errorf("last argv element = %q, want %q.\n  tokens: %q\n  cmd:    %s",
				tokens[len(tokens)-1], instruction, tokens, got)
		}
	})

	t.Run("instruction with single quote round-trips", func(t *testing.T) {
		instruction := "it's fine"
		got := runStageDispatchCapturesCommand(t, newCodex(), "codex", instruction)

		if !strings.Contains(got, `'it'\''s fine'`) {
			t.Errorf("expected single quote to be POSIX-escaped as '\\'' in logged command, got: %q", got)
		}
		tokens, err := shellSplitForTest(got)
		if err != nil {
			t.Fatalf("logged command is not shell-tokenizable: %v (%q)", err, got)
		}
		if len(tokens) < 2 || tokens[len(tokens)-1] != instruction {
			t.Errorf("last argv element = %q, want %q.\n  tokens: %q\n  cmd:    %s",
				tokens[len(tokens)-1], instruction, tokens, got)
		}
	})

	t.Run("instruction with shell metacharacters round-trips", func(t *testing.T) {
		instruction := `rm -rf /; echo "pwned" && cat $HOME`
		got := runStageDispatchCapturesCommand(t, newCodex(), "codex", instruction)

		tokens, err := shellSplitForTest(got)
		if err != nil {
			t.Fatalf("logged command is not shell-tokenizable: %v (%q)", err, got)
		}
		if len(tokens) < 2 || tokens[len(tokens)-1] != instruction {
			t.Errorf("last argv element = %q, want %q.\n  tokens: %q\n  cmd:    %s",
				tokens[len(tokens)-1], instruction, tokens, got)
		}
	})
}

// sliceContainsOrdered reports whether sub appears in s as a contiguous
// subsequence, matching exact element equality. Used to locate a specific
// flag/value pair in a larger argv.
func sliceContainsOrdered(s, sub []string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := range sub {
			if s[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
