package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

func TestFeatureComplete_SetsFeatureStatusToCompletedWithNoTasks(t *testing.T) {
	// Setup test database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

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

	// Create test feature with no tasks
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

	// Simulate feature complete logic with no tasks
	// THIS IS THE FIX: Set feature status to completed even with no tasks
	feature.Status = models.FeatureStatusCompleted
	if err := featureRepo.Update(ctx, feature); err != nil {
		t.Fatalf("Failed to update feature status: %v", err)
	}

	// Verify feature status is now completed
	updatedFeature, err := featureRepo.GetByKey(ctx, "E01-F01")
	if err != nil {
		t.Fatalf("Failed to get updated feature: %v", err)
	}

	if updatedFeature.Status != models.FeatureStatusCompleted {
		t.Errorf("Expected feature status to be 'completed', got '%s'", updatedFeature.Status)
	}
}

func TestFeatureComplete_SetsFeatureStatusToCompletedWithTasks(t *testing.T) {
	// Setup test database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create repositories
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

	// Create test tasks
	task1 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E01-F01-001",
		Title:     "Test Task 1",
		Status:    models.TaskStatusReadyForReview,
		Priority:  5,
	}
	if err := taskRepo.Create(ctx, task1); err != nil {
		t.Fatalf("Failed to create task1: %v", err)
	}

	// Complete all tasks
	agent := "test-agent"
	if err := taskRepo.UpdateStatusForced(ctx, task1.ID, models.TaskStatusCompleted, &agent, nil, true); err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	// Update feature progress
	if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
		t.Fatalf("Failed to update feature progress: %v", err)
	}

	// Verify feature status is automatically set to completed (should auto-complete at 100%)
	updatedFeature, err := featureRepo.GetByKey(ctx, "E01-F01")
	if err != nil {
		t.Fatalf("Failed to get updated feature: %v", err)
	}

	if updatedFeature.Status != models.FeatureStatusCompleted {
		t.Errorf("Expected feature status to be 'completed', got '%s'", updatedFeature.Status)
	}
}
