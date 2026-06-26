package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfgworkflow "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// routeBasedTaskWorkflowSvc builds a workflow.Service from a temp project whose
// task workflow uses only canonical route step names. Legacy status migration
// must use Shark's built-in repair map, not workflow aliases.
func routeBasedTaskWorkflowSvc(t *testing.T) *workflow.Service {
	t.Helper()
	projectRoot := t.TempDir()
	workflowDir := filepath.Join(projectRoot, "shark-data", "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	taskYAML := `version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    is_planning: true
    action: advance_status
    outcomes:
      pass: development
      fail: draft
      blocked: blocked
  development:
    phase: development
    action: spawn_agent
    agent: developer
    outcomes:
      pass: completed
      fail: draft
      blocked: blocked
  blocked:
    phase: blocked
    parking: true
  completed:
    phase: done
    terminal: true
`
	if err := os.WriteFile(filepath.Join(workflowDir, "task.yaml"), []byte(taskYAML), 0o644); err != nil {
		t.Fatalf("write task.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".sharkconfig.json"),
		[]byte(`{"workflow_config": "shark-data/workflow"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgworkflow.ClearWorkflowCache()
	t.Cleanup(cfgworkflow.ClearWorkflowCache)
	return workflow.NewService(projectRoot)
}

func TestMigrateStatuses_DryRunAndApply(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)

	// Clean any prior test rows.
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-MIG-%'")

	_, featureID := test.SeedTestData()

	// Insert three tasks with distinct statuses straight into the DB so we can
	// stage a legacy status value (raw SQL bypasses Go-level status validation).
	insert := func(key, status string) int64 {
		res, err := database.ExecContext(ctx,
			`INSERT INTO tasks (key, title, status, feature_id) VALUES (?, ?, ?, ?)`,
			key, "Migration test", status, featureID)
		if err != nil {
			t.Fatalf("insert %s: %v", key, err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	legacyID := insert("TEST-MIG-001", "ready_for_development") // non-identity alias -> rewritten
	currentID := insert("TEST-MIG-002", "development")          // already current -> untouched
	blockedID := insert("TEST-MIG-003", "blocked")              // identity step name -> untouched
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-MIG-%'")
	})

	// Seed a task_history row for the legacy task; the migration must not touch it.
	if _, err := database.ExecContext(ctx,
		`INSERT INTO task_history (task_id, old_status, new_status) VALUES (?, ?, ?)`,
		legacyID, "draft", "ready_for_development"); err != nil {
		t.Fatalf("seed task_history: %v", err)
	}

	wfSvc := routeBasedTaskWorkflowSvc(t)

	statusOf := func(id int64) string {
		var s string
		if err := database.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", id).Scan(&s); err != nil {
			t.Fatalf("status of %d: %v", id, err)
		}
		return s
	}

	// --- Plan (dry-run): must NOT mutate the status column ---
	planned, err := collectStatusRewrites(ctx, db, wfSvc)
	if err != nil {
		t.Fatalf("collectStatusRewrites: %v", err)
	}
	var found *statusRewrite
	for i := range planned {
		if planned[i].Table == "tasks" && planned[i].Old == "ready_for_development" {
			found = &planned[i]
		}
		if planned[i].Old == planned[i].New {
			t.Errorf("plan contains an identity rewrite: %+v", planned[i])
		}
	}
	if found == nil {
		t.Fatal("expected a tasks ready_for_development->development rewrite in the plan")
	}
	if found.New != "development" || found.Count != 1 {
		t.Errorf("unexpected plan entry: %+v", found)
	}
	if got := statusOf(legacyID); got != "ready_for_development" {
		t.Errorf("dry-run mutated the status column: got %q", got)
	}

	// --- Apply: rewrites only the non-identity alias ---
	if err := applyStatusRewrites(ctx, db, planned); err != nil {
		t.Fatalf("applyStatusRewrites: %v", err)
	}
	if got := statusOf(legacyID); got != "development" {
		t.Errorf("legacy task not rewritten: got %q, want development", got)
	}
	if got := statusOf(currentID); got != "development" {
		t.Errorf("already-current task changed: got %q", got)
	}
	if got := statusOf(blockedID); got != "blocked" {
		t.Errorf("blocked task changed: got %q", got)
	}

	// --- task_history preserved (audit trail untouched) ---
	var histNew string
	if err := database.QueryRowContext(ctx,
		"SELECT new_status FROM task_history WHERE task_id = ?", legacyID).Scan(&histNew); err != nil {
		t.Fatalf("read task_history: %v", err)
	}
	if histNew != "ready_for_development" {
		t.Errorf("task_history was rewritten: got %q, want ready_for_development", histNew)
	}
}

func TestMigrateStatuses_UnmappedLegacyStatusFails(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	db := repository.NewDB(database)

	epicRes, err := database.ExecContext(ctx,
		`INSERT INTO epics (key, title, description, status, priority) VALUES (?, ?, ?, ?, ?)`,
		"E98", "Migration Epic", "Migration test epic", "active", "high")
	if err != nil {
		t.Fatalf("insert epic: %v", err)
	}
	epicID, err := epicRes.LastInsertId()
	if err != nil {
		t.Fatalf("epic last insert id: %v", err)
	}
	featureRes, err := database.ExecContext(ctx,
		`INSERT INTO features (epic_id, key, title, description, status) VALUES (?, ?, ?, ?, ?)`,
		epicID, "E98-F01", "Migration Feature", "Migration test feature", "active")
	if err != nil {
		t.Fatalf("insert feature: %v", err)
	}
	featureID, err := featureRes.LastInsertId()
	if err != nil {
		t.Fatalf("feature last insert id: %v", err)
	}
	res, err := database.ExecContext(ctx,
		`INSERT INTO tasks (key, title, status, feature_id) VALUES (?, ?, ?, ?)`,
		"TEST-MIG-UNMAPPED-001", "Unmapped legacy status", "ready_for_unknown", featureID)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
	if _, err := res.RowsAffected(); err != nil {
		t.Fatalf("rows affected: %v", err)
	}

	wfSvc := routeBasedTaskWorkflowSvc(t)
	_, err = collectStatusRewrites(ctx, db, wfSvc)
	if err == nil {
		t.Fatal("expected unmapped legacy status to fail")
	}
	if !strings.Contains(err.Error(), "ready_for_unknown") {
		t.Fatalf("expected error to mention ready_for_unknown, got %v", err)
	}
}
