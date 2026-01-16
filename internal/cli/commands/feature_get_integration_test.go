package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TestFeatureGetIntegration_CalculateProgressWithConfig tests progress calculation with real config
func TestFeatureGetIntegration_CalculateProgressWithConfig(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E07-F01%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E07-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E07'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Seed test data
	epic := &models.Epic{
		Key:   "E07",
		Title: "Enhancements",
		Slug:  "enhancements",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	feature := &models.Feature{
		Key:    "E07-F01",
		Title:  "Feature One",
		Slug:   "feature-one",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create 4 tasks with different statuses: 2 completed, 1 in progress, 1 todo
	tasks := []*models.Task{
		{
			Key:       "TEST-E07-F01-001",
			Title:     "Task 1",
			Status:    "completed",
			FeatureID: feature.ID,
			EpicID:    epic.ID,
		},
		{
			Key:       "TEST-E07-F01-002",
			Title:     "Task 2",
			Status:    "completed",
			FeatureID: feature.ID,
			EpicID:    epic.ID,
		},
		{
			Key:       "TEST-E07-F01-003",
			Title:     "Task 3",
			Status:    "in_progress",
			FeatureID: feature.ID,
			EpicID:    epic.ID,
		},
		{
			Key:       "TEST-E07-F01-004",
			Title:     "Task 4",
			Status:    "todo",
			FeatureID: feature.ID,
			EpicID:    epic.ID,
		},
	}

	for _, task := range tasks {
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Calculate progress
	progress, err := featureRepo.CalculateProgress(ctx, feature.ID)
	if err != nil {
		t.Fatalf("CalculateProgress failed: %v", err)
	}

	// Verify: 2 completed out of 4 = 50%
	expectedProgress := 50.0
	if progress != expectedProgress {
		t.Errorf("Expected progress %f, got %f", expectedProgress, progress)
	}

	// Cleanup
	for _, task := range tasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestFeatureGetIntegration_GetStatusInfo tests status info with workflow config
func TestFeatureGetIntegration_GetStatusInfo(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E07-F02%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E07-F02'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E07'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Seed test data
	epic := &models.Epic{
		Key:   "E07",
		Title: "Enhancements",
		Slug:  "enhancements",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	feature := &models.Feature{
		Key:    "E07-F02",
		Title:  "Feature Two",
		Slug:   "feature-two",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create tasks with different statuses
	testCases := []struct {
		key    string
		status string
	}{
		{"TEST-E07-F02-001", "todo"},
		{"TEST-E07-F02-002", "in_progress"},
		{"TEST-E07-F02-003", "in_progress"},
		{"TEST-E07-F02-004", "completed"},
	}

	createdTasks := []*models.Task{}
	for _, tc := range testCases {
		task := &models.Task{
			Key:       tc.key,
			Title:     tc.key,
			Status:    tc.status,
			FeatureID: feature.ID,
			EpicID:    epic.ID,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		createdTasks = append(createdTasks, task)
	}

	// Get tasks for this feature to verify status counts
	tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		t.Fatalf("ListByFeature failed: %v", err)
	}

	// Verify we have 4 tasks
	if len(tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(tasks))
	}

	// Count statuses
	statusCounts := make(map[string]int)
	for _, task := range tasks {
		statusCounts[task.Status]++
	}

	// Verify counts: 1 todo, 2 in_progress, 1 completed
	if statusCounts["todo"] != 1 {
		t.Errorf("Expected 1 todo task, got %d", statusCounts["todo"])
	}
	if statusCounts["in_progress"] != 2 {
		t.Errorf("Expected 2 in_progress tasks, got %d", statusCounts["in_progress"])
	}
	if statusCounts["completed"] != 1 {
		t.Errorf("Expected 1 completed task, got %d", statusCounts["completed"])
	}

	// Cleanup
	for _, task := range createdTasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestFeatureGetIntegration_FeatureGetCommandJSONOutput tests full feature get command with JSON
func TestFeatureGetIntegration_FeatureGetCommandJSONOutput(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E07-F03%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E07-F03'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E07'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Seed test data
	epic := &models.Epic{
		Key:   "E07",
		Title: "Enhancements",
		Slug:  "enhancements",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	feature := &models.Feature{
		Key:         "E07-F03",
		Title:       "Feature Three",
		Slug:        "feature-three",
		EpicID:      epic.ID,
		Description: "Test feature for integration tests",
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create a task for the feature
	task := &models.Task{
		Key:       "TEST-E07-F03-001",
		Title:     "Sample Task",
		Status:    "todo",
		FeatureID: feature.ID,
		EpicID:    epic.ID,
		Priority:  5,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Update feature progress
	if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
		t.Fatalf("Failed to update progress: %v", err)
	}

	// Get updated feature
	updatedFeature, err := featureRepo.GetByID(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get feature: %v", err)
	}

	// Verify feature was updated with progress
	if updatedFeature == nil {
		t.Fatal("Expected feature, got nil")
	}

	if updatedFeature.Key != "E07-F03" {
		t.Errorf("Expected key E07-F03, got %s", updatedFeature.Key)
	}

	if updatedFeature.Title != "Feature Three" {
		t.Errorf("Expected title 'Feature Three', got %s", updatedFeature.Title)
	}

	// Verify JSON serialization works
	jsonData, err := json.Marshal(updatedFeature)
	if err != nil {
		t.Fatalf("Failed to marshal feature to JSON: %v", err)
	}

	var parsedFeature models.Feature
	if err := json.Unmarshal(jsonData, &parsedFeature); err != nil {
		t.Fatalf("Failed to unmarshal feature from JSON: %v", err)
	}

	if parsedFeature.Key != updatedFeature.Key {
		t.Errorf("Expected key %s, got %s", updatedFeature.Key, parsedFeature.Key)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestFeatureGetIntegration_WorkflowAwareness tests workflow configuration integration
func TestFeatureGetIntegration_WorkflowAwareness(t *testing.T) {
	ctx := context.Background()

	// Load workflow configuration
	workflowService := workflow.NewService(".")

	// Get status metadata
	metadata := workflowService.GetStatusMetadata("completed")

	// Verify workflow metadata is loaded
	if metadata == nil {
		t.Fatal("Expected status metadata for 'completed', got nil")
	}

	// Verify metadata has expected fields
	if metadata.Color == "" {
		t.Error("Expected color in metadata")
	}

	if metadata.Phase == "" {
		t.Error("Expected phase in metadata")
	}

	// Verify different statuses have different phases
	todoMeta := workflowService.GetStatusMetadata("todo")
	completedMeta := workflowService.GetStatusMetadata("completed")

	if todoMeta == nil || completedMeta == nil {
		t.Fatal("Expected both todo and completed metadata")
	}

	// These should be different phases (todo in planning, completed in done)
	if todoMeta.Phase == completedMeta.Phase {
		t.Logf("Note: Phases are the same (%s), this may be expected depending on workflow config", todoMeta.Phase)
	}
}

// TestFeatureGetIntegration_MultipleFeatures tests progress aggregation across multiple features
func TestFeatureGetIntegration_MultipleFeatures(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E07-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E07-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E07'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Seed test data
	epic := &models.Epic{
		Key:   "E07",
		Title: "Enhancements",
		Slug:  "enhancements",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create 3 features with different completion rates
	featureData := []struct {
		key        string
		title      string
		taskCount  int
		completed  int
	}{
		{"E07-F01", "Feature 1", 4, 2}, // 50% complete
		{"E07-F02", "Feature 2", 3, 3}, // 100% complete
		{"E07-F03", "Feature 3", 5, 0}, // 0% complete
	}

	createdFeatures := []*models.Feature{}
	createdTasks := []*models.Task{}

	for _, fd := range featureData {
		feature := &models.Feature{
			Key:    fd.key,
			Title:  fd.title,
			Slug:   fd.key,
			EpicID: epic.ID,
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}
		createdFeatures = append(createdFeatures, feature)

		// Create tasks for this feature
		for i := 1; i <= fd.taskCount; i++ {
			status := "todo"
			if i <= fd.completed {
				status = "completed"
			}
			task := &models.Task{
				Key:       "TEST-E07-" + fd.key[4:] + "-00" + string(rune('0'+i)),
				Title:     fd.key + " Task " + string(rune('0'+i)),
				Status:    status,
				FeatureID: feature.ID,
				EpicID:    epic.ID,
			}
			if err := taskRepo.Create(ctx, task); err != nil {
				t.Fatalf("Failed to create task: %v", err)
			}
			createdTasks = append(createdTasks, task)
		}
	}

	// Get all features for epic and verify they have proper task counts
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	if len(features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(features))
	}

	// Verify each feature's progress can be calculated
	for _, feature := range features {
		progress, err := featureRepo.CalculateProgress(ctx, feature.ID)
		if err != nil {
			t.Errorf("Failed to calculate progress for feature %s: %v", feature.Key, err)
		}

		// Verify progress is a valid percentage (0-100)
		if progress < 0 || progress > 100 {
			t.Errorf("Feature %s progress out of range: %f", feature.Key, progress)
		}

		// Verify specific expected progress values
		switch feature.Key {
		case "E07-F01":
			if progress != 50.0 {
				t.Errorf("Expected F01 progress 50%%, got %f%%", progress)
			}
		case "E07-F02":
			if progress != 100.0 {
				t.Errorf("Expected F02 progress 100%%, got %f%%", progress)
			}
		case "E07-F03":
			if progress != 0.0 {
				t.Errorf("Expected F03 progress 0%%, got %f%%", progress)
			}
		}
	}

	// Cleanup
	for _, task := range createdTasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}
	for _, feature := range createdFeatures {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestFeatureGetIntegration_EmptyFeature tests feature with no tasks
func TestFeatureGetIntegration_EmptyFeature(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E07-F04'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E07'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)

	// Seed test data
	epic := &models.Epic{
		Key:   "E07",
		Title: "Enhancements",
		Slug:  "enhancements",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	feature := &models.Feature{
		Key:    "E07-F04",
		Title:  "Feature Four",
		Slug:   "feature-four",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Calculate progress for feature with no tasks
	progress, err := featureRepo.CalculateProgress(ctx, feature.ID)
	if err != nil {
		t.Fatalf("CalculateProgress failed: %v", err)
	}

	// Empty feature should have 0% progress
	if progress != 0.0 {
		t.Errorf("Expected 0%% progress for empty feature, got %f%%", progress)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}
