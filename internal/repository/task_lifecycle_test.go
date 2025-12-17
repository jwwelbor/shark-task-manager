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
	agent := "workflow-test-agent"

	// Workflow: todo -> in_progress -> ready_for_review -> completed

	// Step 1: Start task
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
	if err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}

	updatedTask, _ := taskRepo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status in_progress, got %s", updatedTask.Status)
	}
	if !updatedTask.StartedAt.Valid {
		t.Error("Expected started_at to be set")
	}

	// Step 2: Complete task
	notes := "Implementation finished"
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusReadyForReview, &agent, &notes)
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	updatedTask, _ = taskRepo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatusReadyForReview {
		t.Errorf("Expected status ready_for_review, got %s", updatedTask.Status)
	}

	// Step 3: Approve task
	approvalNotes := "LGTM"
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &agent, &approvalNotes)
	if err != nil {
		t.Fatalf("Failed to approve task: %v", err)
	}

	updatedTask, _ = taskRepo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatusCompleted {
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
	agent := "block-test-agent"

	// Start the task
	taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)

	// Block it
	reason := "Waiting for API specification"
	err = taskRepo.BlockTask(ctx, task.ID, reason, &agent)
	if err != nil {
		t.Fatalf("Failed to block task: %v", err)
	}

	blockedTask, _ := taskRepo.GetByID(ctx, task.ID)
	if blockedTask.Status != models.TaskStatusBlocked {
		t.Errorf("Expected status blocked, got %s", blockedTask.Status)
	}
	if blockedTask.BlockedReason == nil || *blockedTask.BlockedReason != reason {
		t.Errorf("Expected blocked_reason '%s'", reason)
	}
	if !blockedTask.BlockedAt.Valid {
		t.Error("Expected blocked_at to be set")
	}

	// Unblock it
	err = taskRepo.UnblockTask(ctx, task.ID, &agent)
	if err != nil {
		t.Fatalf("Failed to unblock task: %v", err)
	}

	unblockedTask, _ := taskRepo.GetByID(ctx, task.ID)
	if unblockedTask.Status != models.TaskStatusTodo {
		t.Errorf("Expected status todo after unblock, got %s", unblockedTask.Status)
	}
	if unblockedTask.BlockedReason != nil {
		t.Error("Expected blocked_reason to be cleared")
	}
	if unblockedTask.BlockedAt.Valid {
		t.Error("Expected blocked_at to be cleared")
	}
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
	agent := "reopen-test-agent"

	// Complete workflow to ready_for_review
	taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
	taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusReadyForReview, &agent, nil)

	// Reopen for rework
	reworkNotes := "Need to add error handling"
	err = taskRepo.ReopenTask(ctx, task.ID, &agent, &reworkNotes)
	if err != nil {
		t.Fatalf("Failed to reopen task: %v", err)
	}

	reopenedTask, _ := taskRepo.GetByID(ctx, task.ID)
	if reopenedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status in_progress after reopen, got %s", reopenedTask.Status)
	}
	if reopenedTask.CompletedAt.Valid {
		t.Error("Expected completed_at to be cleared after reopen")
	}

	// Can complete again after rework
	err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusReadyForReview, &agent, nil)
	if err != nil {
		t.Errorf("Should be able to complete task again after reopen: %v", err)
	}
}

