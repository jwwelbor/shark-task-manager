package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTaskUpdate_PriorityAndDependencies verifies that task update correctly updates priority and dependencies
// and that task get displays the updated values
func TestTaskUpdate_PriorityAndDependencies(t *testing.T) {
	// ARRANGE: Set up test database and create test tasks
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := repository.NewDB(database)

	// Clean up test data before and after
	cleanupTestData := func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E90-F90-%'")
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E90-F90'")
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E90'")
	}
	cleanupTestData()
	defer cleanupTestData()

	// Create test epic
	epicRepo := repository.NewEpicRepository(dbWrapper)
	epic := &models.Epic{
		Key:      "E90",
		Title:    "Test Epic for Update",
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
		Key:    "E90-F90",
		Title:  "Test Feature for Update",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	// Create dependency task (must be created first)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	depTask := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E90-F90-001",
		Title:     "Dependency Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, depTask)
	if err != nil {
		t.Fatalf("Failed to create dependency task: %v", err)
	}

	// Create main task
	mainTask := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E90-F90-002",
		Title:     "Main Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, mainTask)
	if err != nil {
		t.Fatalf("Failed to create main task: %v", err)
	}

	// ACT: Update the task with new priority and dependency
	newPriority := 2
	dependsOnJSON := `["T-E90-F90-001"]`

	updatedTask, err := taskRepo.GetByKey(ctx, mainTask.Key)
	if err != nil {
		t.Fatalf("Failed to get task for update: %v", err)
	}

	updatedTask.Priority = newPriority
	updatedTask.DependsOn = &dependsOnJSON

	err = taskRepo.Update(ctx, updatedTask)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	// ASSERT: Verify the task was updated
	fetchedTask, err := taskRepo.GetByKey(ctx, mainTask.Key)
	if err != nil {
		t.Fatalf("Failed to fetch updated task: %v", err)
	}

	// Verify priority was updated
	if fetchedTask.Priority != newPriority {
		t.Errorf("Expected priority %d, got %d", newPriority, fetchedTask.Priority)
	}

	// Verify dependencies were updated
	if fetchedTask.DependsOn == nil {
		t.Error("Expected dependencies to be set, got nil")
	} else {
		var deps []string
		err = json.Unmarshal([]byte(*fetchedTask.DependsOn), &deps)
		if err != nil {
			t.Errorf("Failed to parse dependencies: %v", err)
		} else if len(deps) != 1 || deps[0] != "T-E90-F90-001" {
			t.Errorf("Expected dependencies [T-E90-F90-001], got %v", deps)
		}
	}
}
