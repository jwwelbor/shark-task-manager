package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestUpdateStatus_WorkflowValidation tests that status transitions are validated against workflow config
func TestUpdateStatus_WorkflowValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create custom workflow with strict transitions using existing DB statuses
	// Note: We use existing statuses because DB has CHECK constraint
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress"}, // Can only go to in_progress
			"in_progress":      {"completed"},   // Can only go to completed
			"completed":        {},              // Terminal status
			"blocked":          {},              // Terminal (for this test)
			"ready_for_review": {},              // Terminal (for this test)
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	// Clean up and seed test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-WORKFLOW-%'")
	test.SeedTestData()

	// Get test task and reset to todo
	task, err := repo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset task to todo status
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

	agent := "workflow-validation-test"

	// Test 1: Valid transition (todo -> in_progress)
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
	if err != nil {
		t.Errorf("Valid transition todo->in_progress should succeed, got error: %v", err)
	}

	// Verify status was updated
	updatedTask, _ := repo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status in_progress, got %s", updatedTask.Status)
	}

	// Test 2: Invalid transition (in_progress -> todo) - should fail
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusTodo, &agent, nil)
	if err == nil {
		t.Error("Invalid transition in_progress->todo should fail, but succeeded")
	}

	// Verify error message mentions the invalid transition
	if err != nil && !containsString(err.Error(), "invalid transition") {
		t.Errorf("Error message should mention 'invalid transition', got: %v", err)
	}

	// Test 3: Valid transition (in_progress -> completed)
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &agent, nil)
	if err != nil {
		t.Errorf("Valid transition in_progress->completed should succeed, got error: %v", err)
	}

	// Test 4: Invalid transition from terminal status (completed -> in_progress)
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
	if err == nil {
		t.Error("Transition from terminal status should fail, but succeeded")
	}
}

// TestUpdateStatus_DefaultWorkflowValidation tests validation with default workflow
func TestUpdateStatus_DefaultWorkflowValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Use default workflow (todo -> in_progress -> ready_for_review -> completed)
	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset to todo
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

	agent := "default-workflow-test"

	// Valid transition: todo -> in_progress
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
	if err != nil {
		t.Errorf("Valid transition should succeed: %v", err)
	}

	// Invalid transition: in_progress -> completed (skipping ready_for_review)
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &agent, nil)
	if err == nil {
		t.Error("Invalid transition in_progress->completed should fail (must go through ready_for_review)")
	}

	// Valid transition: in_progress -> ready_for_review
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusReadyForReview, &agent, nil)
	if err != nil {
		t.Errorf("Valid transition should succeed: %v", err)
	}

	// Valid transition: ready_for_review -> completed
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &agent, nil)
	if err != nil {
		t.Errorf("Valid transition should succeed: %v", err)
	}
}

// TestUpdateStatus_BlockedTransitions tests blocking transitions with workflow
func TestUpdateStatus_BlockedTransitions(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Default workflow allows blocking from todo and in_progress
	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-003")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset to todo
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

	agent := "blocked-workflow-test"

	// Valid: todo -> blocked
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusBlocked, &agent, nil)
	if err != nil {
		t.Errorf("Valid transition todo->blocked should succeed: %v", err)
	}

	// Valid: blocked -> in_progress
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
	if err != nil {
		t.Errorf("Valid transition blocked->in_progress should succeed: %v", err)
	}

	// Valid: in_progress -> blocked
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusBlocked, &agent, nil)
	if err != nil {
		t.Errorf("Valid transition in_progress->blocked should succeed: %v", err)
	}
}

// TestUpdateStatus_ErrorMessages tests that validation errors include helpful information
func TestUpdateStatus_ErrorMessages(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Use DB-compatible statuses
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress", "blocked"},
			"in_progress":      {"ready_for_review"},
			"ready_for_review": {"completed"},
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
	task, err := repo.GetByKey(ctx, "T-E99-F99-004")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Set task to todo status
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

	agent := "error-message-test"

	// Try invalid transition (todo -> completed, skipping in_progress and ready_for_review)
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &agent, nil)

	// Verify error contains helpful information
	if err == nil {
		t.Fatal("Expected error for invalid transition, got nil")
	}

	errMsg := err.Error()
	expectedParts := []string{"invalid transition", "todo", "completed"}
	for _, part := range expectedParts {
		if !containsString(errMsg, part) {
			t.Errorf("Error message should contain '%s', got: %s", part, errMsg)
		}
	}
}
