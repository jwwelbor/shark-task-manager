package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestCompleteWorkflow tests a full task lifecycle
func TestCompleteWorkflow(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := taskRepo.GetByKey(ctx, "T-E99-F99-002") // Todo task
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset task to todo status in case a previous test modified it
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)

	agent := "workflow-test-agent"

	// Workflow: todo -> in_progress -> ready_for_review -> completed

	// Step 1: Start task
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agent, nil)
	if err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}

	updatedTask, _ := taskRepo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatus("in_progress") {
		t.Errorf("Expected status in_progress, got %s", updatedTask.Status)
	}
	if !updatedTask.StartedAt.Valid {
		t.Error("Expected started_at to be set")
	}

	// Step 2: Complete task
	notes := "Implementation finished"
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), &agent, &notes)
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	updatedTask, _ = taskRepo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatus("ready_for_review") {
		t.Errorf("Expected status ready_for_review, got %s", updatedTask.Status)
	}

	// Step 3: Approve task
	approvalNotes := "LGTM"
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("completed"), &agent, &approvalNotes)
	if err != nil {
		t.Fatalf("Failed to approve task: %v", err)
	}

	updatedTask, _ = taskRepo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatus("completed") {
		t.Errorf("Expected status completed, got %s", updatedTask.Status)
	}
	if !updatedTask.CompletedAt.Valid {
		t.Error("Expected completed_at to be set")
	}

	// Verify history records were created (3 transitions)
	var historyCount int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_history WHERE task_id = ?", task.ID).Scan(&historyCount)
	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}
	if historyCount < 3 {
		t.Errorf("Expected at least 3 history records, got %d", historyCount)
	}
}

// TestBlockUnblockWorkflow tests blocking and unblocking
func TestBlockUnblockWorkflow(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := taskRepo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset task to todo status in case a previous test modified it
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)

	agent := "block-test-agent"

	// Start the task
	if err := taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agent, nil); err != nil {
		t.Fatalf("Failed to update task status to in_progress: %v", err)
	}

	// Block it via UpdateStatusForced (BlockTask removed - validation moved to service layer)
	reason := "Waiting for API specification"
	err = taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("blocked"), &agent, &reason, nil, nil, false)
	if err != nil {
		t.Fatalf("Failed to block task: %v", err)
	}

	blockedTask, _ := taskRepo.GetByID(ctx, task.ID)
	if blockedTask.Status != models.TaskStatus("blocked") {
		t.Errorf("Expected status blocked, got %s", blockedTask.Status)
	}
	// NOTE: blocked_reason and blocked_at field management has moved to the service layer.
	// UpdateStatusForced only updates status and timestamps, not blocked_reason.

	// Unblock it via UpdateStatusForced (UnblockTask removed - validation moved to service layer)
	err = taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("todo"), &agent, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("Failed to unblock task: %v", err)
	}

	unblockedTask, _ := taskRepo.GetByID(ctx, task.ID)
	if unblockedTask.Status != models.TaskStatus("todo") {
		t.Errorf("Expected status todo after unblock, got %s", unblockedTask.Status)
	}
	// NOTE: blocked_reason and blocked_at clearing has moved to the service layer.
	// UpdateStatusForced only updates status, not blocked_reason/blocked_at fields.
}

// TestReopenWorkflow tests reopening a task for rework
func TestReopenWorkflow(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := taskRepo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset task to todo status in case a previous test modified it
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)

	agent := "reopen-test-agent"

	// Complete workflow to ready_for_review
	if err := taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agent, nil); err != nil {
		t.Fatalf("Failed to update task status to in_progress: %v", err)
	}
	if err := taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), &agent, nil); err != nil {
		t.Fatalf("Failed to update task status to ready_for_review: %v", err)
	}

	// Reopen for rework via UpdateStatusForced (ReopenTask removed - validation moved to service layer)
	reworkNotes := "Need to add error handling"
	err = taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("in_progress"), &agent, &reworkNotes, nil, nil, false)
	if err != nil {
		t.Fatalf("Failed to reopen task: %v", err)
	}

	reopenedTask, _ := taskRepo.GetByID(ctx, task.ID)
	if reopenedTask.Status != models.TaskStatus("in_progress") {
		t.Errorf("Expected status in_progress after reopen, got %s", reopenedTask.Status)
	}
	if reopenedTask.CompletedAt.Valid {
		t.Error("Expected completed_at to be cleared after reopen")
	}

	// Can complete again after rework
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), &agent, nil)
	if err != nil {
		t.Errorf("Should be able to complete task again after reopen: %v", err)
	}
}
