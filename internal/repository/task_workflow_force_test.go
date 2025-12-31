package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestUpdateStatusForced_BypassValidation tests that force flag bypasses workflow validation
func TestUpdateStatusForced_BypassValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create strict workflow
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress"}, // Can ONLY go to in_progress
			"in_progress":      {"completed"},   // Can ONLY go to completed
			"completed":        {},              // Terminal
			"blocked":          {},              // Terminal (unreachable in this workflow)
			"ready_for_review": {},              // Terminal (unreachable in this workflow)
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
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

	agent := "force-test-agent"

	// Test 1: Invalid transition WITHOUT force should fail
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &agent, nil)
	if err == nil {
		t.Error("Invalid transition todo->completed should fail without force flag")
	}

	// Test 2: Same invalid transition WITH force should succeed
	err = repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, nil, true)
	if err != nil {
		t.Errorf("Forced transition should succeed, got error: %v", err)
	}

	// Verify status was updated
	updatedTask, _ := repo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatusCompleted {
		t.Errorf("Expected status completed after forced transition, got %s", updatedTask.Status)
	}

	// Test 3: Verify history record has forced=true
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND new_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatusCompleted).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if !forced {
		t.Error("Expected history record to have forced=true")
	}
}

// TestUpdateStatusForced_BlockTaskForced tests forcing block transitions
func TestUpdateStatusForced_BlockTaskForced(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Workflow that doesn't allow blocking from completed
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress", "blocked"},
			"in_progress":      {"completed", "blocked"},
			"completed":        {}, // No transitions allowed
			"blocked":          {"todo"},
			"ready_for_review": {"completed"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean up any existing history for this task from previous test runs
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id = ? AND new_status = ?", task.ID, models.TaskStatusBlocked)

	// Set task to completed
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusCompleted, task.ID)

	agent := "block-force-test"
	reason := "Emergency rollback needed"

	// Test 1: Blocking from completed should fail without force
	err = repo.BlockTask(ctx, task.ID, reason, &agent)
	if err == nil {
		t.Error("Blocking from completed should fail without force")
	}

	// Test 2: Blocking from completed WITH force should succeed
	err = repo.BlockTaskForced(ctx, task.ID, reason, &agent, true)
	if err != nil {
		t.Errorf("Forced block should succeed: %v", err)
	}

	// Verify blocked
	blockedTask, _ := repo.GetByID(ctx, task.ID)
	if blockedTask.Status != models.TaskStatusBlocked {
		t.Errorf("Expected status blocked, got %s", blockedTask.Status)
	}

	// Verify history has forced=true
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND new_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatusBlocked).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if !forced {
		t.Error("Expected forced block to have forced=true in history")
	}
}

// TestUpdateStatusForced_ReopenTaskForced tests forcing reopen transitions
func TestUpdateStatusForced_ReopenTaskForced(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Workflow that doesn't allow reopening from completed
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress"},
			"in_progress":      {"ready_for_review"},
			"ready_for_review": {"completed"}, // No reopen path
			"completed":        {},
			"blocked":          {"todo"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-003")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Set to ready_for_review
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusReadyForReview, task.ID)

	agent := "reopen-force-test"
	notes := "Found critical bug, needs rework"

	// Test 1: Reopen should fail without force (workflow doesn't allow it)
	err = repo.ReopenTask(ctx, task.ID, &agent, &notes)
	if err == nil {
		t.Error("Reopen should fail when workflow doesn't allow it")
	}

	// Test 2: Reopen WITH force should succeed
	err = repo.ReopenTaskForced(ctx, task.ID, &agent, &notes, true)
	if err != nil {
		t.Errorf("Forced reopen should succeed: %v", err)
	}

	// Verify reopened
	reopenedTask, _ := repo.GetByID(ctx, task.ID)
	if reopenedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status in_progress after reopen, got %s", reopenedTask.Status)
	}

	// Verify history has forced=true
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND new_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatusInProgress).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if !forced {
		t.Error("Expected forced reopen to have forced=true in history")
	}
}

// TestUpdateStatusForced_UnblockTaskForced tests forcing unblock transitions
func TestUpdateStatusForced_UnblockTaskForced(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Workflow where unblocking is restricted
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress"},
			"in_progress":      {"completed", "blocked"},
			"completed":        {},
			"blocked":          {}, // No unblock path
			"ready_for_review": {"completed"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-004")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Set to blocked
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ?, blocked_reason = ? WHERE id = ?",
		models.TaskStatusBlocked, "Test blocking", task.ID)

	agent := "unblock-force-test"

	// Test 1: Unblock should fail without force
	err = repo.UnblockTask(ctx, task.ID, &agent)
	if err == nil {
		t.Error("Unblock should fail when workflow doesn't allow transitions from blocked")
	}

	// Test 2: Unblock WITH force should succeed
	err = repo.UnblockTaskForced(ctx, task.ID, &agent, true)
	if err != nil {
		t.Errorf("Forced unblock should succeed: %v", err)
	}

	// Verify unblocked
	unblockedTask, _ := repo.GetByID(ctx, task.ID)
	if unblockedTask.Status != models.TaskStatusTodo {
		t.Errorf("Expected status todo after unblock, got %s", unblockedTask.Status)
	}

	// Verify history has forced=true
	var forced bool
	var newStatus string
	err = database.QueryRowContext(ctx,
		"SELECT forced, new_status FROM task_history WHERE task_id = ? AND old_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatusBlocked).Scan(&forced, &newStatus)

	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("Failed to query history: %v", err)
	}

	if err == nil && !forced {
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
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

	agent := "normal-transition-test"

	// Perform normal valid transition
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
	if err != nil {
		t.Fatalf("Valid transition should succeed: %v", err)
	}

	// Verify history has forced=false
	var forced bool
	err = database.QueryRowContext(ctx,
		"SELECT forced FROM task_history WHERE task_id = ? AND new_status = ? ORDER BY timestamp DESC LIMIT 1",
		task.ID, models.TaskStatusInProgress).Scan(&forced)

	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}

	if forced {
		t.Error("Expected normal transition to have forced=false, got forced=true")
	}
}
