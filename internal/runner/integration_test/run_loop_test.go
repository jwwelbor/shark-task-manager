// Package integration_test contains end-to-end tests for the shark run loop.
// These tests spin up an isolated SQLite database and use mock agent dispatchers
// so they exercise real status-transition logic without any LLM calls or
// production database access.
package integration_test

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// taskTransitioner adapts TaskService to the runner.EntityTransitioner interface
// for use inside RunControllerDeps.
type taskTransitioner struct {
	svc *services.TaskService
}

func (a *taskTransitioner) TransitionStatus(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return a.svc.TransitionStatus(ctx, key, targetStatus, opts)
}

func (a *taskTransitioner) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return a.svc.GetNextStatus(ctx, key)
}

// noopPlaceholders returns an empty map — sufficient for the minimal workflow
// whose instruction templates contain only literal text.
type noopPlaceholders struct{}

func (n *noopPlaceholders) GeneratePlaceholders(_ context.Context, _ string) (map[string]string, error) {
	return map[string]string{}, nil
}

// newController is a helper that builds a RunController wired to env's
// action service, workflow service, and dispatchers.
func newController(t *testing.T, env *Env) *runner.RunController {
	t.Helper()

	actionSvc := env.NewActionService()
	taskSvc := env.NewTaskService()

	ctrl, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner: &taskTransitioner{svc: taskSvc},
		Placeholders: &noopPlaceholders{},
		ActionSvc:    actionSvc,
		WorkflowSvc:  env.WorkflowSvc,
		Dispatchers:  env.Dispatchers,
	})
	if err != nil {
		t.Fatalf("newController: %v", err)
	}
	return ctrl
}

// -------------------------------------------------------------------------
// TestRunLoop_SpawnAgent_SucceedsAndAdvancesToCompleted
// -------------------------------------------------------------------------

// TestRunLoop_SpawnAgent_SucceedsAndAdvancesToCompleted verifies that a task
// seeded in "todo" status is advanced to "completed" by the run loop when the
// mock dispatcher returns exit code 0.
//
// Workflow:
//
//	todo --[advance_status]--> in_progress --[spawn_agent]--> completed
func TestRunLoop_SpawnAgent_SucceedsAndAdvancesToCompleted(t *testing.T) {
	env := NewEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	// Seed a task in "todo" status.
	env.SeedTask(ctx, "IT-E01", "IT-E01-F01", "IT-E01-F01-001", "Implement feature", "todo")

	ctrl := newController(t, env)
	result, err := ctrl.Run(ctx, "IT-E01-F01-001", runner.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// The loop should have completed successfully.
	if result.Outcome != "completed" {
		t.Errorf("expected outcome 'completed', got %q (error: %s)", result.Outcome, result.Error)
	}

	// At least two stages: advance_status (todo→in_progress) + spawn_agent (in_progress→completed).
	if result.StagesCompleted < 2 {
		t.Errorf("expected at least 2 stages completed, got %d", result.StagesCompleted)
	}

	// The final status should be "completed".
	if result.FinalStatus != "completed" {
		t.Errorf("expected final status 'completed', got %q", result.FinalStatus)
	}

	// Verify the mock dispatcher was called exactly once (for the in_progress stage).
	disp := env.Dispatchers["anthropic"].(*MockDispatcher)
	if disp.DispatchCallCount != 1 {
		t.Errorf("expected dispatcher to be called once, got %d", disp.DispatchCallCount)
	}
}

// -------------------------------------------------------------------------
// TestRunLoop_DryRun_NoStatusChange
// -------------------------------------------------------------------------

// TestRunLoop_DryRun_NoStatusChange verifies that dry-run mode previews
// spawn_agent stages without modifying the entity's status or calling a real
// dispatcher.
//
// To avoid the infinite-loop scenario where advance_status dry-run does not
// update the DB (and subsequent GetNextStatus sees stale state), the task is
// seeded directly at "in_progress" — the status that triggers spawn_agent —
// bypassing the advance_status step entirely.
func TestRunLoop_DryRun_NoStatusChange(t *testing.T) {
	env := NewEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	// Seed at "in_progress" so the first (and only) action is spawn_agent.
	// After dry-run spawn_agent, GetNextStatus sees DB status "in_progress" and
	// returns the transition to "completed"; the loop stops cleanly.
	env.SeedTask(ctx, "IT-E02", "IT-E02-F01", "IT-E02-F01-001", "Dry-run task", "in_progress")

	ctrl := newController(t, env)
	result, err := ctrl.Run(ctx, "IT-E02-F01-001", runner.RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Run() dry-run error: %v", err)
	}

	// Dry-run must NOT call the real dispatcher.
	disp := env.Dispatchers["anthropic"].(*MockDispatcher)
	if disp.DispatchCallCount != 0 {
		t.Errorf("dry-run must not call dispatcher; got %d calls", disp.DispatchCallCount)
	}

	// The task must still be in its original status ("in_progress").
	taskSvc := env.NewTaskService()
	info, err := taskSvc.GetNextStatus(ctx, "IT-E02-F01-001")
	if err != nil {
		t.Fatalf("GetNextStatus after dry-run: %v", err)
	}
	if info.CurrentStatus != "in_progress" {
		t.Errorf("dry-run must not change status; expected 'in_progress', got %q", info.CurrentStatus)
	}

	// The dry-run result should describe the planned stage.
	if len(result.Stages) == 0 {
		t.Error("expected at least one stage in dry-run result")
	}

	// Outcome should reflect that the run finished without error.
	if result.Outcome == "failed" {
		t.Errorf("unexpected failed outcome in dry-run: %s", result.Error)
	}
}

// -------------------------------------------------------------------------
// TestRunLoop_AgentFails_StopsWithFailedOutcome
// -------------------------------------------------------------------------

// TestRunLoop_AgentFails_StopsWithFailedOutcome verifies that when the
// dispatcher returns a non-zero exit code, the loop stops and reports
// a "failed" outcome.
func TestRunLoop_AgentFails_StopsWithFailedOutcome(t *testing.T) {
	env := NewEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	env.SeedTask(ctx, "IT-E03", "IT-E03-F01", "IT-E03-F01-001", "Failing task", "in_progress")

	// Override the dispatcher to return non-zero exit.
	failDisp := NewMockDispatcher("default", &DispatchResult{
		ExitCode: 1,
		Stdout:   "",
		Stderr:   "agent encountered a fatal error",
	})
	env.Dispatchers[""] = failDisp
	env.Dispatchers["anthropic"] = failDisp

	ctrl := newController(t, env)
	result, err := ctrl.Run(ctx, "IT-E03-F01-001", runner.RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected Go error: %v", err)
	}

	// The outcome must be "failed" when the agent exits non-zero.
	if result.Outcome != "failed" {
		t.Errorf("expected outcome 'failed', got %q", result.Outcome)
	}

	// The error field should mention the exit code or failure.
	if result.Error == "" {
		t.Error("expected non-empty Error field in failed result")
	}

	// Dispatcher should have been called exactly once.
	if failDisp.DispatchCallCount != 1 {
		t.Errorf("expected dispatcher called once, got %d", failDisp.DispatchCallCount)
	}
}

// -------------------------------------------------------------------------
// TestRunLoop_AlreadyTerminal
// -------------------------------------------------------------------------

// TestRunLoop_AlreadyTerminal verifies that running an entity that is already
// in a terminal status returns the "already_terminal" outcome immediately
// without dispatching any agent or changing status.
func TestRunLoop_AlreadyTerminal(t *testing.T) {
	env := NewEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	env.SeedTask(ctx, "IT-E04", "IT-E04-F01", "IT-E04-F01-001", "Completed task", "completed")

	ctrl := newController(t, env)
	result, err := ctrl.Run(ctx, "IT-E04-F01-001", runner.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.Outcome != "already_terminal" {
		t.Errorf("expected outcome 'already_terminal', got %q", result.Outcome)
	}

	// No dispatcher should have been called.
	disp := env.Dispatchers["anthropic"].(*MockDispatcher)
	if disp.DispatchCallCount != 0 {
		t.Errorf("expected no dispatcher calls for terminal entity; got %d", disp.DispatchCallCount)
	}

	// Status must not have changed.
	if result.FinalStatus != "completed" {
		t.Errorf("expected final status 'completed', got %q", result.FinalStatus)
	}
}

// -------------------------------------------------------------------------
// TestRunLoop_DispatchInputPopulated
// -------------------------------------------------------------------------

// TestRunLoop_DispatchInputPopulated verifies that the RunController passes
// populated instruction strings and entity keys to the dispatcher so agents
// receive usable context.
func TestRunLoop_DispatchInputPopulated(t *testing.T) {
	env := NewEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	const taskKey = "IT-E05-F01-001"
	env.SeedTask(ctx, "IT-E05", "IT-E05-F01", taskKey, "Test dispatch input", "in_progress")

	ctrl := newController(t, env)
	_, err := ctrl.Run(ctx, taskKey, runner.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	disp := env.Dispatchers["anthropic"].(*MockDispatcher)
	if disp.LastInput == nil {
		t.Fatal("expected LastInput to be set after dispatch")
	}

	// The instruction must be non-empty.
	if disp.LastInput.Instruction == "" {
		t.Error("expected non-empty Instruction in DispatchInput")
	}

	// The entity key must be passed through.
	if disp.LastInput.EntityKey != taskKey {
		t.Errorf("expected EntityKey %q in DispatchInput, got %q", taskKey, disp.LastInput.EntityKey)
	}
}

// -------------------------------------------------------------------------
// TestRunLoop_MockDispatcher_WithError
// -------------------------------------------------------------------------

// TestRunLoop_MockDispatcher_WithError verifies the "failed" path when the
// dispatcher returns a Go error (e.g., process could not be started).
func TestRunLoop_MockDispatcher_WithError(t *testing.T) {
	env := NewEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	env.SeedTask(ctx, "IT-E06", "IT-E06-F01", "IT-E06-F01-001", "Error dispatch task", "in_progress")

	// Make the dispatcher return a hard error.
	errDisp := NewMockDispatcher("default", nil).WithError(
		&runner.ToolNotFoundError{Tool: "claude"},
	)
	env.Dispatchers[""] = errDisp
	env.Dispatchers["anthropic"] = errDisp

	ctrl := newController(t, env)
	result, err := ctrl.Run(ctx, "IT-E06-F01-001", runner.RunOptions{})
	if err != nil {
		t.Fatalf("Run() returned unexpected Go error (expected nil): %v", err)
	}

	if result.Outcome != "failed" {
		t.Errorf("expected outcome 'failed', got %q", result.Outcome)
	}

	if result.Error == "" {
		t.Error("expected non-empty Error in RunResult when dispatcher returns error")
	}
}
