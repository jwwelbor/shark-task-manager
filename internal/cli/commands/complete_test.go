package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestFeatureComplete_CompletedTasks tests completing a feature with all completed tasks
func TestFeatureComplete_CompletedTasks(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up test data
	database.Exec("DELETE FROM tasks WHERE key LIKE 'T-E71-F01%'")
	database.Exec("DELETE FROM features WHERE key = 'E71-F01'")
	database.Exec("DELETE FROM epics WHERE key = 'E71'")

	// Get repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create test epic
	epic := &models.Epic{
		Key:      "E71",
		Title:    "Test Feature Complete",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E71-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create completed tasks
	for i := 0; i < 3; i++ {
		task := &models.Task{
			FeatureID: feature.ID,
			Key:       "T-E71-F01-00" + string(rune(49+i)),
			Title:     "Task " + string(rune(49+i)),
			Status:    models.TaskStatusCompleted,
			Priority:  5,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Update progress
	if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
		t.Fatalf("Failed to update progress: %v", err)
	}

	// Get updated feature
	feature, err := featureRepo.GetByID(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get feature: %v", err)
	}

	// Verify progress is 100%
	if feature.ProgressPct != 100.0 {
		t.Errorf("Expected feature progress to be 100.0%%, got %.1f%%", feature.ProgressPct)
	}
}

// TestFeatureComplete_MixedStatuses tests feature complete with mixed task statuses
func TestFeatureComplete_MixedStatuses(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	database.Exec("DELETE FROM tasks WHERE key LIKE 'T-E72-F02%'")
	database.Exec("DELETE FROM features WHERE key = 'E72-F02'")
	database.Exec("DELETE FROM epics WHERE key = 'E72'")

	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create epic
	epic := &models.Epic{
		Key:      "E72",
		Title:    "Feature Complete Test",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E72-F02",
		Title:  "Mixed Status Feature",
		Status: models.FeatureStatusActive,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create tasks with different statuses
	statuses := []models.TaskStatus{
		models.TaskStatusTodo,
		models.TaskStatusInProgress,
		models.TaskStatusCompleted,
	}

	for i, status := range statuses {
		task := &models.Task{
			FeatureID: feature.ID,
			Key:       "T-E72-F02-00" + string(rune(49+i)),
			Title:     "Task",
			Status:    status,
			Priority:  5,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Get status breakdown
	breakdown, err := taskRepo.GetStatusBreakdown(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get status breakdown: %v", err)
	}

	// Verify breakdown
	if breakdown[models.TaskStatusTodo] != 1 {
		t.Errorf("Expected 1 todo task, got %d", breakdown[models.TaskStatusTodo])
	}
	if breakdown[models.TaskStatusInProgress] != 1 {
		t.Errorf("Expected 1 in_progress task, got %d", breakdown[models.TaskStatusInProgress])
	}
	if breakdown[models.TaskStatusCompleted] != 1 {
		t.Errorf("Expected 1 completed task, got %d", breakdown[models.TaskStatusCompleted])
	}

	// Complete remaining tasks
	agent := "test-agent"
	allTasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	for _, task := range allTasks {
		if task.Status != models.TaskStatusCompleted {
			if err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, nil, true); err != nil {
				t.Fatalf("Failed to complete task: %v", err)
			}
		}
	}

	// Update progress
	if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
		t.Fatalf("Failed to update progress: %v", err)
	}

	// Verify all tasks are completed and progress is 100%
	feature, err = featureRepo.GetByID(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get feature: %v", err)
	}

	if feature.ProgressPct != 100.0 {
		t.Errorf("Expected feature progress to be 100.0%%, got %.1f%%", feature.ProgressPct)
	}
}

// TestEpicComplete_MultipleFeatures tests epic complete across multiple features
func TestEpicComplete_MultipleFeatures(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	database.Exec("DELETE FROM tasks WHERE key LIKE 'T-E73-F%'")
	database.Exec("DELETE FROM features WHERE key LIKE 'E73-F%'")
	database.Exec("DELETE FROM epics WHERE key = 'E73'")

	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create epic
	epic := &models.Epic{
		Key:      "E73",
		Title:    "Epic Complete Test",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create multiple features with tasks
	numFeatures := 2
	tasksPerFeature := 2

	for f := 0; f < numFeatures; f++ {
		feature := &models.Feature{
			EpicID: epic.ID,
			Key:    "E73-F0" + string(rune(49+f)),
			Title:  "Feature",
			Status: models.FeatureStatusActive,
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}

		// Create tasks for feature
		for taskIdx := 0; taskIdx < tasksPerFeature; taskIdx++ {
			task := &models.Task{
				FeatureID: feature.ID,
				Key:       "T-E73-F0" + string(rune(49+f)) + "-00" + string(rune(49+taskIdx)),
				Title:     "Task",
				Status:    models.TaskStatusTodo,
				Priority:  5,
			}
			if err := taskRepo.Create(ctx, task); err != nil {
				t.Fatalf("Failed to create task: %v", err)
			}
		}
	}

	// Get all features and complete tasks
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	if len(features) != numFeatures {
		t.Errorf("Expected %d features, got %d", numFeatures, len(features))
	}

	agent := "test-agent"
	totalCompleted := 0

	for _, feature := range features {
		featureTasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to list feature tasks: %v", err)
		}

		for _, task := range featureTasks {
			if err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, nil, true); err != nil {
				t.Fatalf("Failed to complete task: %v", err)
			}
			totalCompleted++
		}

		// Update feature progress
		if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
			t.Fatalf("Failed to update feature progress: %v", err)
		}
	}

	// Verify all tasks completed
	expectedTotal := numFeatures * tasksPerFeature
	if totalCompleted != expectedTotal {
		t.Errorf("Expected %d completed tasks, got %d", expectedTotal, totalCompleted)
	}

	// Calculate epic progress
	epicProgress, err := epicRepo.CalculateProgress(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to calculate epic progress: %v", err)
	}

	if epicProgress != 100.0 {
		t.Errorf("Expected epic progress 100.0%%, got %.1f%%", epicProgress)
	}
}

// TestFeatureComplete_JSONOutput tests JSON output serialization
func TestFeatureComplete_JSONOutput(t *testing.T) {
	result := map[string]interface{}{
		"feature_key":     "E-TEST-F01",
		"tasks_completed": 5,
		"total_tasks":     5,
		"progress":        100.0,
	}

	// Serialize to JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Deserialize and verify
	var output map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &output); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if output["feature_key"] != "E-TEST-F01" {
		t.Errorf("Expected feature_key E-TEST-F01, got %v", output["feature_key"])
	}

	if output["progress"].(float64) != 100.0 {
		t.Errorf("Expected progress 100.0, got %v", output["progress"])
	}
}

// TestEpicComplete_StatusBreakdown tests status breakdown aggregation
func TestEpicComplete_StatusBreakdown(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	database.Exec("DELETE FROM tasks WHERE key LIKE 'T-E74-F%'")
	database.Exec("DELETE FROM features WHERE key LIKE 'E74-F%'")
	database.Exec("DELETE FROM epics WHERE key = 'E74'")

	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create epic
	epic := &models.Epic{
		Key:      "E74",
		Title:    "Status Breakdown Test",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create 2 features with mixed statuses
	featureStatuses := [][]models.TaskStatus{
		{models.TaskStatusTodo, models.TaskStatusCompleted},
		{models.TaskStatusInProgress, models.TaskStatusBlocked},
	}

	totalStatuses := make(map[models.TaskStatus]int)

	for f, statuses := range featureStatuses {
		feature := &models.Feature{
			EpicID: epic.ID,
			Key:    "E74-F0" + string(rune(49+f)),
			Title:  "Feature",
			Status: models.FeatureStatusActive,
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}

		// Create tasks
		for taskIdx, status := range statuses {
			task := &models.Task{
				FeatureID: feature.ID,
				Key:       "T-E74-F0" + string(rune(49+f)) + "-00" + string(rune(49+taskIdx)),
				Title:     "Task",
				Status:    status,
				Priority:  5,
			}
			if err := taskRepo.Create(ctx, task); err != nil {
				t.Fatalf("Failed to create task: %v", err)
			}
			totalStatuses[status]++
		}
	}

	// Verify aggregated status breakdown
	if totalStatuses[models.TaskStatusTodo] != 1 {
		t.Errorf("Expected 1 todo task, got %d", totalStatuses[models.TaskStatusTodo])
	}
	if totalStatuses[models.TaskStatusInProgress] != 1 {
		t.Errorf("Expected 1 in_progress task, got %d", totalStatuses[models.TaskStatusInProgress])
	}
	if totalStatuses[models.TaskStatusCompleted] != 1 {
		t.Errorf("Expected 1 completed task, got %d", totalStatuses[models.TaskStatusCompleted])
	}
	if totalStatuses[models.TaskStatusBlocked] != 1 {
		t.Errorf("Expected 1 blocked task, got %d", totalStatuses[models.TaskStatusBlocked])
	}
}
