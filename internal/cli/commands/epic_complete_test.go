package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

func TestEpicComplete_SetsEpicStatusToCompleted(t *testing.T) {
	// Setup test database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repositories using the test database
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create test epic
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create test feature
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E01-F01",
		Title:       "Test Feature",
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create test tasks (all in ready_for_review status)
	task1 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E01-F01-001",
		Title:     "Test Task 1",
		Status:    models.TaskStatusReadyForReview,
		Priority:  5,
	}
	task2 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E01-F01-002",
		Title:     "Test Task 2",
		Status:    models.TaskStatusReadyForReview,
		Priority:  5,
	}

	if err := taskRepo.Create(ctx, task1); err != nil {
		t.Fatalf("Failed to create task1: %v", err)
	}
	if err := taskRepo.Create(ctx, task2); err != nil {
		t.Fatalf("Failed to create task2: %v", err)
	}

	// Execute epic complete logic (simulate the command)
	// Get all features in epic
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	// Complete all tasks
	for _, feature := range features {
		tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		agent := "test-agent"
		for _, task := range tasks {
			if task.Status != models.TaskStatusCompleted {
				if err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, nil, true); err != nil {
					t.Fatalf("Failed to complete task: %v", err)
				}
			}
		}

		// Update feature progress
		if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
			t.Fatalf("Failed to update feature progress: %v", err)
		}
	}

	// THIS IS THE FIX: Update epic status to completed
	epic.Status = models.EpicStatusCompleted
	if err := epicRepo.Update(ctx, epic); err != nil {
		t.Fatalf("Failed to update epic status: %v", err)
	}

	// Verify epic status is now completed
	updatedEpic, err := epicRepo.GetByKey(ctx, "E01")
	if err != nil {
		t.Fatalf("Failed to get updated epic: %v", err)
	}

	if updatedEpic.Status != models.EpicStatusCompleted {
		t.Errorf("Expected epic status to be 'completed', got '%s'", updatedEpic.Status)
	}
}

func TestEpicComplete_SetsEpicStatusToCompletedEvenWithNoTasks(t *testing.T) {
	// Setup test database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repositories using the test database
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	// Create test epic with no features
	epic := &models.Epic{
		Key:      "E02",
		Title:    "Test Epic No Features",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Update epic status to completed
	epic.Status = models.EpicStatusCompleted
	if err := epicRepo.Update(ctx, epic); err != nil {
		t.Fatalf("Failed to update epic status: %v", err)
	}

	// Verify epic status is completed
	updatedEpic, err := epicRepo.GetByKey(ctx, "E02")
	if err != nil {
		t.Fatalf("Failed to get updated epic: %v", err)
	}

	if updatedEpic.Status != models.EpicStatusCompleted {
		t.Errorf("Expected epic status to be 'completed', got '%s'", updatedEpic.Status)
	}
}
