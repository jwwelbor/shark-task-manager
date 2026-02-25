package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestUpdateStatusForced_BypassValidation tests that UpdateStatusForced works
// for any transition now that validation is handled by the service layer.
func TestUpdateStatusForced_BypassValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create strict workflow (kept for backward compatibility with constructor)
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress"},
			"in_progress":      {"completed"},
			"completed":        {},
			"blocked":          {},
			"ready_for_review": {},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset to todo
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)

	agent := "force-test-agent"

	// Test 1: Any transition succeeds now (validation in service layer)
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("completed"), &agent, nil)
	if err != nil {
		t.Errorf("Transition should succeed (validation moved to service): %v", err)
	}

	// Reset and clean history to avoid timestamp ordering issues
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id = ? AND new_status = ?", task.ID, models.TaskStatus("completed"))

	// Test 2: Same transition WITH force should also succeed
	err = repo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("completed"), &agent, nil, nil, nil, true)
	if err != nil {
		t.Errorf("Forced transition should succeed, got error: %v", err)
	}

	// Verify status was updated
	updatedTask, _ := repo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatus("completed") {
		t.Errorf("Expected status completed after forced transition, got %s", updatedTask.Status)
	}

	// Test 3: Verify history record has forced=true
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND new_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatus("completed")).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if !forced {
		t.Error("Expected history record to have forced=true")
	}
}

// TestUpdateStatusForced_BlockTransition tests forced block transitions via UpdateStatusForced.
// NOTE: BlockTask/BlockTaskForced were removed from the repository (validation moved to service layer).
// Block transitions now go through UpdateStatusForced directly.
func TestUpdateStatusForced_BlockTransition(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing history for this task from previous test runs
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id = ? AND new_status = ?", task.ID, models.TaskStatus("blocked"))

	// Set task to completed
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("completed"), task.ID)

	agent := "block-force-test"
	reason := "Emergency rollback needed"

	// Block transition with force=true via UpdateStatusForced
	err = repo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("blocked"), &agent, &reason, nil, nil, true)
	if err != nil {
		t.Errorf("Forced block should succeed: %v", err)
	}

	// Verify blocked
	blockedTask, _ := repo.GetByID(ctx, task.ID)
	if blockedTask.Status != models.TaskStatus("blocked") {
		t.Errorf("Expected status blocked, got %s", blockedTask.Status)
	}

	// Verify history has forced=true
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND new_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatus("blocked")).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if !forced {
		t.Error("Expected forced block to have forced=true in history")
	}
}

// TestUpdateStatusForced_ReopenTransition tests forced reopen transitions via UpdateStatusForced.
// NOTE: ReopenTask/ReopenTaskForced were removed from the repository (validation moved to service layer).
// Reopen transitions now go through UpdateStatusForced directly.
func TestUpdateStatusForced_ReopenTransition(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-003")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Set to ready_for_review
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("ready_for_review"), task.ID)

	agent := "reopen-force-test"
	notes := "Found critical bug, needs rework"

	// Reopen transition with force=true via UpdateStatusForced
	err = repo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("in_progress"), &agent, &notes, nil, nil, true)
	if err != nil {
		t.Errorf("Forced reopen should succeed: %v", err)
	}

	// Verify reopened
	reopenedTask, _ := repo.GetByID(ctx, task.ID)
	if reopenedTask.Status != models.TaskStatus("in_progress") {
		t.Errorf("Expected status in_progress after reopen, got %s", reopenedTask.Status)
	}

	// Verify history has forced=true
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND new_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatus("in_progress")).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if !forced {
		t.Error("Expected forced reopen to have forced=true in history")
	}
}

// TestUpdateStatusForced_UnblockTransition tests forced unblock transitions via UpdateStatusForced.
// NOTE: UnblockTask/UnblockTaskForced were removed from the repository (validation moved to service layer).
// Unblock transitions now go through UpdateStatusForced directly.
func TestUpdateStatusForced_UnblockTransition(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-004")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Set to blocked
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ?, blocked_reason = ? WHERE id = ?",
		models.TaskStatus("blocked"), "Test blocking", task.ID)

	agent := "unblock-force-test"

	// Unblock transition with force=true via UpdateStatusForced
	err = repo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("todo"), &agent, nil, nil, nil, true)
	if err != nil {
		t.Errorf("Forced unblock should succeed: %v", err)
	}

	// Verify unblocked
	unblockedTask, _ := repo.GetByID(ctx, task.ID)
	if unblockedTask.Status != models.TaskStatus("todo") {
		t.Errorf("Expected status todo after unblock, got %s", unblockedTask.Status)
	}

	// Verify history has forced=true
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND old_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatus("blocked")).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if !forced {
		t.Error("Expected forced unblock to have forced=true in history")
	}
}

// TestUpdateStatusForced_NormalTransitionsNotForced tests that normal transitions have forced=false
func TestUpdateStatusForced_NormalTransitionsNotForced(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	repo := NewTaskRepository(db) // Use default workflow

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset to todo
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)

	agent := "normal-transition-test"

	// Perform normal valid transition
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agent, nil)
	if err != nil {
		t.Fatalf("Valid transition should succeed: %v", err)
	}

	// Verify history has forced=false
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND new_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatus("in_progress")).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if forced {
		t.Error("Expected normal transition to have forced=false, got forced=true")
	}
}
