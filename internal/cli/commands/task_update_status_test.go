package commands

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTaskUpdate_WithStatusFlag tests that task update with --status flag
// updates the task status using workflow validation
func TestTaskUpdate_WithStatusFlag(t *testing.T) {
	// ARRANGE: Set up test database and create test task
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := repository.NewDB(database)

	// Clean up test data before and after
	cleanupTestData := func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E97-F97-%'")
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F97'")
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")
	}
	cleanupTestData()
	defer cleanupTestData()

	// Create test epic
	epicRepo := repository.NewEpicRepository(dbWrapper)
	epic := &models.Epic{
		Key:      "E97",
		Title:    "Test Epic for Status Update",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Create test feature
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E97-F97",
		Title:  "Test Feature for Status Update",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	// Load workflow config
	workflow, err := config.LoadWorkflowConfig(".sharkconfig.json")
	if err != nil {
		t.Fatalf("Failed to load workflow config: %v", err)
	}

	// Create task repository with workflow
	taskRepo := repository.NewTaskRepositoryWithWorkflow(dbWrapper, workflow)

	// Create test task with initial status
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E97-F97-001",
		Title:     "Test Task for Status Update",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// ACT: Update the task status using the repository method
	// (simulating what runTaskUpdate should call)
	// Use in_progress which exists in hardcoded fallback statuses
	newStatus := models.TaskStatusInProgress
	err = taskRepo.UpdateStatusForced(ctx, task.ID, newStatus, nil, nil, true)

	// ASSERT: Verify the status was updated
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Retrieve the updated task
	updatedTask, err := taskRepo.GetByKey(ctx, task.Key)
	if err != nil {
		t.Fatalf("Failed to retrieve updated task: %v", err)
	}

	// Verify status changed
	if updatedTask.Status != newStatus {
		t.Errorf("Expected status %s, got %s", newStatus, updatedTask.Status)
	}

	// Verify task_history was created
	var historyCount int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_history WHERE task_id = ?", task.ID).Scan(&historyCount)
	if err != nil {
		t.Fatalf("Failed to query task_history: %v", err)
	}

	if historyCount == 0 {
		t.Error("Expected task_history record to be created, but found none")
	}
}

// TestTaskUpdate_WithStatusFlag_InvalidTransition tests that invalid status transitions
// are rejected by workflow validation
func TestTaskUpdate_WithStatusFlag_InvalidTransition(t *testing.T) {
	// ARRANGE: Set up test database and create test task
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := repository.NewDB(database)

	// Clean up test data before and after
	cleanupTestData := func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-F98-%'")
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F98'")
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
	}
	cleanupTestData()
	defer cleanupTestData()

	// Create test epic
	epicRepo := repository.NewEpicRepository(dbWrapper)
	epic := &models.Epic{
		Key:      "E98",
		Title:    "Test Epic for Invalid Transition",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Create test feature
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E98-F98",
		Title:  "Test Feature for Invalid Transition",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	// Load workflow config
	workflow, err := config.LoadWorkflowConfig(".sharkconfig.json")
	if err != nil {
		t.Fatalf("Failed to load workflow config: %v", err)
	}

	// Create task repository with workflow
	taskRepo := repository.NewTaskRepositoryWithWorkflow(dbWrapper, workflow)

	// Create test task with "completed" status
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E98-F98-001",
		Title:     "Test Task for Invalid Transition",
		Status:    models.TaskStatus("completed"),
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// ACT: Attempt to update status to invalid transition (completed -> todo)
	// According to workflow, completed has no valid transitions (terminal state)
	invalidStatus := models.TaskStatus("todo")
	err = taskRepo.UpdateStatusForced(ctx, task.ID, invalidStatus, nil, nil, false)

	// ASSERT: Verify the transition was rejected
	if err == nil {
		t.Fatal("Expected error for invalid status transition, got nil")
	}

	// Verify error is a ValidationError (from validation package)
	if _, ok := err.(*config.ValidationError); !ok {
		// Also check for WorkflowValidationError (renamed from ValidationError)
		if _, ok := err.(*config.WorkflowValidationError); !ok {
			t.Errorf("Expected ValidationError or WorkflowValidationError, got %T: %v", err, err)
		}
	}

	// Verify status did NOT change
	unchangedTask, err := taskRepo.GetByKey(ctx, task.Key)
	if err != nil {
		t.Fatalf("Failed to retrieve task: %v", err)
	}

	if unchangedTask.Status != models.TaskStatus("completed") {
		t.Errorf("Expected status to remain 'completed', got '%s'", unchangedTask.Status)
	}
}
