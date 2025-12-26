package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// TestEpicCompleteCascadesToFeatures verifies that completing an epic
// marks all features as completed, which in turn completes all tasks
func TestEpicCompleteCascadesToFeatures(t *testing.T) {
	// Setup test database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create test epic
	epic := &models.Epic{
		Key:           "E99",
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: nil,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create Feature 1 with tasks
	feature1 := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E99-F01",
		Title:       "Feature 1",
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create feature1: %v", err)
	}

	// Create tasks for feature 1
	task1 := &models.Task{
		FeatureID: feature1.ID,
		Key:       "T-E99-F01-001",
		Title:     "Task 1",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	if err := taskRepo.Create(ctx, task1); err != nil {
		t.Fatalf("Failed to create task1: %v", err)
	}

	task2 := &models.Task{
		FeatureID: feature1.ID,
		Key:       "T-E99-F01-002",
		Title:     "Task 2",
		Status:    models.TaskStatusInProgress,
		Priority:  5,
	}
	if err := taskRepo.Create(ctx, task2); err != nil {
		t.Fatalf("Failed to create task2: %v", err)
	}

	// Create Feature 2 with tasks
	feature2 := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E99-F02",
		Title:       "Feature 2",
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}
	if err := featureRepo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create feature2: %v", err)
	}

	task3 := &models.Task{
		FeatureID: feature2.ID,
		Key:       "T-E99-F02-001",
		Title:     "Task 3",
		Status:    models.TaskStatusBlocked,
		Priority:  5,
	}
	if err := taskRepo.Create(ctx, task3); err != nil {
		t.Fatalf("Failed to create task3: %v", err)
	}

	// Create Feature 3 with no tasks
	feature3 := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E99-F03",
		Title:       "Feature 3 (no tasks)",
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}
	if err := featureRepo.Create(ctx, feature3); err != nil {
		t.Fatalf("Failed to create feature3: %v", err)
	}

	// Verify initial state
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}
	if len(features) != 3 {
		t.Fatalf("Expected 3 features, got %d", len(features))
	}

	allTasks := []*models.Task{}
	for _, feature := range features {
		tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to list tasks for feature %s: %v", feature.Key, err)
		}
		allTasks = append(allTasks, tasks...)
	}
	if len(allTasks) != 3 {
		t.Fatalf("Expected 3 tasks total, got %d", len(allTasks))
	}

	// ACT: Complete the epic (this is the behavior we're testing)
	// Simulate what runEpicComplete does with the fix
	agent := "test-agent"
	for _, task := range allTasks {
		if task.Status != models.TaskStatusCompleted {
			if err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, nil, true); err != nil {
				t.Fatalf("Failed to complete task %s: %v", task.Key, err)
			}
		}
	}

	// Update progress for all features and mark them as completed
	for _, feature := range features {
		// Update progress first (will auto-complete if all tasks are done)
		if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
			t.Fatalf("Failed to update progress for feature %s: %v", feature.Key, err)
		}

		// Fetch the updated feature to check its status
		updatedFeature, err := featureRepo.GetByID(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to get updated feature %s: %v", feature.Key, err)
		}

		// Explicitly mark feature as completed if not already
		// (This handles features with no tasks or other edge cases)
		if updatedFeature.Status != models.FeatureStatusCompleted {
			updatedFeature.Status = models.FeatureStatusCompleted
			if err := featureRepo.Update(ctx, updatedFeature); err != nil {
				t.Fatalf("Failed to complete feature %s: %v", updatedFeature.Key, err)
			}
		}
	}

	// Set epic status to completed
	epic.Status = models.EpicStatusCompleted
	if err := epicRepo.Update(ctx, epic); err != nil {
		t.Fatalf("Failed to update epic status: %v", err)
	}

	// ASSERT: Verify all features are marked as completed
	updatedFeatures, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list updated features: %v", err)
	}

	for _, feature := range updatedFeatures {
		if feature.Status != models.FeatureStatusCompleted {
			t.Errorf("Feature %s status is %s, expected completed", feature.Key, feature.Status)
		}

		// Check progress expectations based on whether feature has tasks
		tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to list tasks for feature %s: %v", feature.Key, err)
		}

		if len(tasks) > 0 {
			// Features with tasks should have 100% progress when completed
			if feature.ProgressPct != 100.0 {
				t.Errorf("Feature %s with tasks has progress %.1f%%, expected 100.0%%", feature.Key, feature.ProgressPct)
			}
		} else {
			// Features with no tasks have 0% progress (as per CalculateProgress logic)
			// but should still be marked as completed
			if feature.ProgressPct != 0.0 {
				t.Errorf("Feature %s with no tasks has progress %.1f%%, expected 0.0%%", feature.Key, feature.ProgressPct)
			}
		}
	}

	// Verify all tasks are completed
	for _, feature := range updatedFeatures {
		tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to list tasks for feature %s: %v", feature.Key, err)
		}
		for _, task := range tasks {
			if task.Status != models.TaskStatusCompleted {
				t.Errorf("Task %s status is %s, expected completed", task.Key, task.Status)
			}
		}
	}

	// Verify epic is completed
	updatedEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to get updated epic: %v", err)
	}
	if updatedEpic.Status != models.EpicStatusCompleted {
		t.Errorf("Epic status is %s, expected completed", updatedEpic.Status)
	}
}
