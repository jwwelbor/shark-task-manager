package repository

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestCompleteWorkflow tests a full task lifecycle
func TestCompleteWorkflow(t *testing.T) {
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	task, _ := taskRepo.GetByKey("T-TEST-002") // Todo task
	agent := "workflow-test-agent"

	// Workflow: todo -> in_progress -> ready_for_review -> completed

	// Step 1: Start task
	err := taskRepo.UpdateStatus(task.ID, models.TaskStatusInProgress, &agent, nil)
	if err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}

	updatedTask, _ := taskRepo.GetByID(task.ID)
	if updatedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status in_progress, got %s", updatedTask.Status)
	}
	if !updatedTask.StartedAt.Valid {
		t.Error("Expected started_at to be set")
	}

	// Step 2: Complete task
	notes := "Implementation finished"
	err = taskRepo.UpdateStatus(task.ID, models.TaskStatusReadyForReview, &agent, &notes)
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	updatedTask, _ = taskRepo.GetByID(task.ID)
	if updatedTask.Status != models.TaskStatusReadyForReview {
		t.Errorf("Expected status ready_for_review, got %s", updatedTask.Status)
	}

	// Step 3: Approve task
	approvalNotes := "LGTM"
	err = taskRepo.UpdateStatus(task.ID, models.TaskStatusCompleted, &agent, &approvalNotes)
	if err != nil {
		t.Fatalf("Failed to approve task: %v", err)
	}

	updatedTask, _ = taskRepo.GetByID(task.ID)
	if updatedTask.Status != models.TaskStatusCompleted {
		t.Errorf("Expected status completed, got %s", updatedTask.Status)
	}
	if !updatedTask.CompletedAt.Valid {
		t.Error("Expected completed_at to be set")
	}

	// Verify history records were created (3 transitions)
	var historyCount int
	err = database.QueryRow("SELECT COUNT(*) FROM task_history WHERE task_id = ?", task.ID).Scan(&historyCount)
	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}
	if historyCount < 3 {
		t.Errorf("Expected at least 3 history records, got %d", historyCount)
	}
}

// TestBlockUnblockWorkflow tests blocking and unblocking
func TestBlockUnblockWorkflow(t *testing.T) {
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	task, _ := taskRepo.GetByKey("T-TEST-002")
	agent := "block-test-agent"

	// Start the task
	taskRepo.UpdateStatus(task.ID, models.TaskStatusInProgress, &agent, nil)

	// Block it
	reason := "Waiting for API specification"
	err := taskRepo.BlockTask(task.ID, reason, &agent)
	if err != nil {
		t.Fatalf("Failed to block task: %v", err)
	}

	blockedTask, _ := taskRepo.GetByID(task.ID)
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
	err = taskRepo.UnblockTask(task.ID, &agent)
	if err != nil {
		t.Fatalf("Failed to unblock task: %v", err)
	}

	unblockedTask, _ := taskRepo.GetByID(task.ID)
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
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	task, _ := taskRepo.GetByKey("T-TEST-002")
	agent := "reopen-test-agent"

	// Complete workflow to ready_for_review
	taskRepo.UpdateStatus(task.ID, models.TaskStatusInProgress, &agent, nil)
	taskRepo.UpdateStatus(task.ID, models.TaskStatusReadyForReview, &agent, nil)

	// Reopen for rework
	reworkNotes := "Need to add error handling"
	err := taskRepo.ReopenTask(task.ID, &agent, &reworkNotes)
	if err != nil {
		t.Fatalf("Failed to reopen task: %v", err)
	}

	reopenedTask, _ := taskRepo.GetByID(task.ID)
	if reopenedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status in_progress after reopen, got %s", reopenedTask.Status)
	}
	if reopenedTask.CompletedAt.Valid {
		t.Error("Expected completed_at to be cleared after reopen")
	}

	// Can complete again after rework
	err = taskRepo.UpdateStatus(task.ID, models.TaskStatusReadyForReview, &agent, nil)
	if err != nil {
		t.Errorf("Should be able to complete task again after reopen: %v", err)
	}
}

