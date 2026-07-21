package integration_test

import (
	"context"
	"os"
	"path/filepath"
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
//	draft --[advance_status]--> research --[advance_status]--> development --[spawn_agent]--> completed
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
	writeTaskResearchArtifacts(t, env, "IT-E01-F01-001")

	ctrl := newController(t, env)
	result, err := ctrl.Run(ctx, "IT-E01-F01-001", runner.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.Outcome != "completed" {
		t.Errorf("expected outcome 'completed', got %q (error: %s)", result.Outcome, result.Error)
	}

	// Three stages: draft->research, validated research->development, then dispatch.
	if result.StagesCompleted < 3 {
		t.Errorf("expected at least 3 stages completed, got %d", result.StagesCompleted)
	}

	if result.FinalStatus != "completed" {
		t.Errorf("expected final status 'completed', got %q", result.FinalStatus)
	}

	disp := env.Dispatchers["anthropic"].(*MockDispatcher)
	if disp.DispatchCallCount != 2 {
		t.Errorf("expected researcher and developer dispatches, got %d calls", disp.DispatchCallCount)
	}
}

func writeTaskResearchArtifacts(t *testing.T, env *Env, taskKey string) {
	t.Helper()
	filePath := filepath.Join("tasks", taskKey+".md")
	if _, err := env.DB.DB.Exec(`UPDATE tasks SET file_path = ? WHERE key = ?`, filePath, taskKey); err != nil {
		t.Fatalf("set task file path: %v", err)
	}
	dir := filepath.Join(env.Dir, "tasks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create task artifact directory: %v", err)
	}
	frontMatter := "---\nresearch_schema: 2\nentity_key: " + taskKey + "\nentity_type: task\nrecipe: universal\nrigor: simple\ncategories: [backend]\nrelated_work: false\n---\n"
	report := frontMatter + "# Research report\n\n## Scope\nRunner task transition.\n\n## Research checklist\n- [x] `scope_vocabulary` — Evidence: `tasks/" + taskKey + ".md`.\n- [x] `affected_implementation_or_contract` — Evidence: `internal/runner` transition path.\n\n## Findings\nThe embedded workflow advances through research and cites the parent Capability map at `docs/plan/IT-E01/IT-E01-F01/research-report.md`.\n\n## Decisions\nUse the existing runner path.\n\n## Sources\n- `internal/runner`\n- `docs/plan/IT-E01/IT-E01-F01/research-report.md` (parent Capability map)\n"
	if err := os.WriteFile(filepath.Join(dir, taskKey+".research-report.md"), []byte(report), 0o644); err != nil {
		t.Fatalf("write research report: %v", err)
	}
}
