package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestUpdateStatus_NoValidation tests that the repository layer no longer validates
// status transitions. All validation is now handled by the service layer.
func TestUpdateStatus_NoValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create custom workflow (kept for constructor compatibility)
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
		StatusMetadata: map[string]config.StatusMetadata{
			"todo":             {ProgressWeight: 0, Phase: "planning"},
			"in_progress":      {ProgressWeight: 50, Phase: "development"},
			"completed":        {ProgressWeight: 100, Phase: "done"},
			"blocked":          {ProgressWeight: 25, Phase: "planning"},
			"ready_for_review": {ProgressWeight: 75, Phase: "review"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-WORKFLOW-%'")
	test.SeedTestData()

	task, err := repo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

	agent := "workflow-validation-test"

	// Test 1: Forward transition succeeds
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agent, nil)
	if err != nil {
		t.Errorf("Transition todo->in_progress should succeed: %v", err)
	}

	updatedTask, _ := repo.GetByID(ctx, task.ID)
	if updatedTask.Status != models.TaskStatus("in_progress") {
		t.Errorf("Expected status in_progress, got %s", updatedTask.Status)
	}

	// Test 2: Previously invalid transition now succeeds (validation moved to service)
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("todo"), &agent, nil)
	if err != nil {
		t.Errorf("Transition in_progress->todo should succeed at repo layer (validation moved to service): %v", err)
	}

	// Test 3: Any transition succeeds at repo layer
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "in_progress", task.ID)
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("completed"), &agent, nil)
	if err != nil {
		t.Errorf("Transition in_progress->completed should succeed at repo layer: %v", err)
	}

	// Test 4: Transition from terminal status also succeeds at repo layer
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agent, nil)
	if err != nil {
		t.Errorf("Transition completed->in_progress should succeed at repo layer (validation moved to service): %v", err)
	}
}

// TestUpdateStatus_DefaultWorkflowAcceptsAll tests that default workflow repo accepts all transitions
func TestUpdateStatus_DefaultWorkflowAcceptsAll(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)

	agent := "default-workflow-test"

	// All transitions should succeed at repo layer
	transitions := []models.TaskStatus{"in_progress", "completed", "ready_for_review", "todo", "blocked"}
	for _, target := range transitions {
		err = repo.UpdateStatus(ctx, task.ID, target, &agent, nil)
		if err != nil {
			t.Errorf("Transition to %s should succeed at repo layer: %v", target, err)
		}
	}
}

// TestUpdateStatus_BlockedTransitions tests blocking transitions work without validation
func TestUpdateStatus_BlockedTransitions(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-003")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)

	agent := "blocked-workflow-test"

	// todo -> blocked
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("blocked"), &agent, nil)
	if err != nil {
		t.Errorf("Transition todo->blocked should succeed: %v", err)
	}

	// blocked -> in_progress
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agent, nil)
	if err != nil {
		t.Errorf("Transition blocked->in_progress should succeed: %v", err)
	}

	// in_progress -> blocked
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("blocked"), &agent, nil)
	if err != nil {
		t.Errorf("Transition in_progress->blocked should succeed: %v", err)
	}
}

// TestUpdateStatus_HistoryRecordsCreated tests that history records are created for all transitions
func TestUpdateStatus_HistoryRecordsCreated(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-004")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatus("todo"), task.ID)
	// Clean up any existing history for this task
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id = ?", task.ID)

	agent := "history-test"

	// Perform a transition
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agent, nil)
	if err != nil {
		t.Fatalf("Transition should succeed: %v", err)
	}

	// Verify history record was created
	var count int
	err = database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM task_history WHERE task_id = ? AND old_status = ? AND new_status = ?",
		task.ID, "todo", "in_progress").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}
	if count == 0 {
		t.Error("Expected history record to be created for transition")
	}
}
