package integration_test

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/runner"
)

// -------------------------------------------------------------------------
// TestRunLoop_ZeroConfig_ResolvesEmbeddedWorkflowAndCompletes
// -------------------------------------------------------------------------

// TestRunLoop_ZeroConfig_ResolvesEmbeddedWorkflowAndCompletes covers the gap
// left by TestRunLoop_SpawnAgent_SucceedsAndAdvancesToCompleted, which relies
// on an on-disk shark-data/workflow/task.yaml fixture. A project created by a
// plain `shark admin init` has no shark-data/ on disk and no workflow_config
// field in .sharkconfig.json — this test proves that shape still resolves a
// real, route-based workflow (through workflow.Service/GetWorkflowForLevel,
// the same path config.ActionService and every workflow.Service consumer use)
// from the embedded canonical bundle, and that a task seeded in the embedded
// workflow's initial status ("draft") can run to completion.
//
// Embedded task workflow (internal/sharkdata/default_data/workflow/task.yaml):
//
//	draft --[advance_status]--> development --[spawn_agent]--> completed
func TestRunLoop_ZeroConfig_ResolvesEmbeddedWorkflowAndCompletes(t *testing.T) {
	env := NewZeroConfigEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	// Sanity check: the workflow service resolved from the embedded bundle
	// alone must report "draft" as the initial status, not "todo" — this is
	// what breaks if a hardcoded legacy default ever creeps back in.
	if got := env.WorkflowSvc.GetInitialStatusString(); got != "draft" {
		t.Fatalf("expected embedded task workflow initial status 'draft', got %q", got)
	}

	// Seed a task in "draft" status (the embedded workflow's start status).
	env.SeedTask(ctx, "IT-E01", "IT-E01-F01", "IT-E01-F01-001", "Implement feature", "draft")

	ctrl := newController(t, env)
	result, err := ctrl.Run(ctx, "IT-E01-F01-001", runner.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.Outcome != "completed" {
		t.Errorf("expected outcome 'completed', got %q (error: %s)", result.Outcome, result.Error)
	}

	// Two stages: draft->development, then developer dispatch.
	if result.StagesCompleted < 2 {
		t.Errorf("expected at least 2 stages completed, got %d", result.StagesCompleted)
	}

	if result.FinalStatus != "completed" {
		t.Errorf("expected final status 'completed', got %q", result.FinalStatus)
	}

	disp := env.Dispatchers["anthropic"].(*MockDispatcher)
	if disp.DispatchCallCount != 1 {
		t.Errorf("expected only developer dispatch, got %d calls", disp.DispatchCallCount)
	}
}

func TestRunLoop_ZeroConfig_LegacyResearchStatusResumesDevelopment(t *testing.T) {
	env := NewZeroConfigEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	env.SeedTask(ctx, "IT-E01", "IT-E01-F01", "IT-E01-F01-002", "Resume legacy research task", "research")

	ctrl := newController(t, env)
	result, err := ctrl.Run(ctx, "IT-E01-F01-002", runner.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.Outcome != "completed" || result.FinalStatus != "completed" {
		t.Fatalf("legacy research task result = outcome %q, final status %q; want completed", result.Outcome, result.FinalStatus)
	}

	disp := env.Dispatchers["anthropic"].(*MockDispatcher)
	if disp.DispatchCallCount != 1 {
		t.Errorf("expected legacy research task to dispatch only developer, got %d calls", disp.DispatchCallCount)
	}
}
