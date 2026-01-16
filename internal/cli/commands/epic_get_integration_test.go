package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestEpicGetIntegration_FeatureStatusRollup tests feature status rollup aggregation
func TestEpicGetIntegration_FeatureStatusRollup(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E07')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E07'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E07",
		Title: "Enhancements",
		Slug:  "enhancements",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create 3 features with different statuses
	features := []*models.Feature{
		{Key: "E07-F01", Title: "Feature 1", Slug: "feature-1", EpicID: epic.ID, Status: "active"},
		{Key: "E07-F02", Title: "Feature 2", Slug: "feature-2", EpicID: epic.ID, Status: "completed"},
		{Key: "E07-F03", Title: "Feature 3", Slug: "feature-3", EpicID: epic.ID, Status: "active"},
	}

	for _, feature := range features {
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}
	}

	// Get feature status rollup
	rollup, err := epicRepo.GetFeatureStatusRollup(ctx, epic.ID)
	if err != nil {
		t.Fatalf("GetFeatureStatusRollup failed: %v", err)
	}

	// Verify rollup counts
	if rollup == nil {
		t.Fatal("Expected rollup map, got nil")
	}

	if rollup["active"] != 2 {
		t.Errorf("Expected 2 active features, got %d", rollup["active"])
	}

	if rollup["completed"] != 1 {
		t.Errorf("Expected 1 completed feature, got %d", rollup["completed"])
	}

	// Cleanup
	for _, feature := range features {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicGetIntegration_TaskStatusRollup tests task status rollup aggregation
func TestEpicGetIntegration_TaskStatusRollup(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E08-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E08')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E08'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E08",
		Title: "Testing",
		Slug:  "testing",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create features
	feature1 := &models.Feature{
		Key:    "E08-F01",
		Title:  "Feature 1",
		Slug:   "feature-1",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	feature2 := &models.Feature{
		Key:    "E08-F02",
		Title:  "Feature 2",
		Slug:   "feature-2",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create tasks across both features with different statuses
	tasks := []*models.Task{
		// Feature 1 tasks
		{Key: "TEST-E08-F01-001", Title: "Task 1", Status: "completed", FeatureID: feature1.ID, EpicID: epic.ID},
		{Key: "TEST-E08-F01-002", Title: "Task 2", Status: "completed", FeatureID: feature1.ID, EpicID: epic.ID},
		{Key: "TEST-E08-F01-003", Title: "Task 3", Status: "in_progress", FeatureID: feature1.ID, EpicID: epic.ID},
		// Feature 2 tasks
		{Key: "TEST-E08-F02-001", Title: "Task 1", Status: "todo", FeatureID: feature2.ID, EpicID: epic.ID},
		{Key: "TEST-E08-F02-002", Title: "Task 2", Status: "in_progress", FeatureID: feature2.ID, EpicID: epic.ID},
		{Key: "TEST-E08-F02-003", Title: "Task 3", Status: "in_progress", FeatureID: feature2.ID, EpicID: epic.ID},
	}

	for _, task := range tasks {
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Get task status rollup
	rollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
	if err != nil {
		t.Fatalf("GetTaskStatusRollup failed: %v", err)
	}

	// Verify rollup counts across all features
	if rollup == nil {
		t.Fatal("Expected rollup map, got nil")
	}

	if rollup["completed"] != 2 {
		t.Errorf("Expected 2 completed tasks, got %d", rollup["completed"])
	}

	if rollup["in_progress"] != 3 {
		t.Errorf("Expected 3 in_progress tasks, got %d", rollup["in_progress"])
	}

	if rollup["todo"] != 1 {
		t.Errorf("Expected 1 todo task, got %d", rollup["todo"])
	}

	// Cleanup
	for _, task := range tasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature1.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature2.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicGetIntegration_ImpedimentsDetection tests blocked tasks detection
func TestEpicGetIntegration_ImpedimentsDetection(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E09-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E09')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E09'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E09",
		Title: "API Development",
		Slug:  "api-development",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create feature
	feature := &models.Feature{
		Key:    "E09-F01",
		Title:  "API Endpoints",
		Slug:   "api-endpoints",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create tasks with some blocked
	tasks := []*models.Task{
		{Key: "TEST-E09-F01-001", Title: "Design API", Status: "completed", FeatureID: feature.ID, EpicID: epic.ID},
		{Key: "TEST-E09-F01-002", Title: "Implement Endpoints", Status: "blocked", FeatureID: feature.ID, EpicID: epic.ID},
		{Key: "TEST-E09-F01-003", Title: "Write Tests", Status: "blocked", FeatureID: feature.ID, EpicID: epic.ID},
		{Key: "TEST-E09-F01-004", Title: "Deploy", Status: "todo", FeatureID: feature.ID, EpicID: epic.ID},
	}

	for _, task := range tasks {
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Get task status rollup to check for blocked tasks
	rollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
	if err != nil {
		t.Fatalf("GetTaskStatusRollup failed: %v", err)
	}

	// Verify blocked task count
	if rollup["blocked"] != 2 {
		t.Errorf("Expected 2 blocked tasks, got %d", rollup["blocked"])
	}

	// Blocked tasks are an impediment
	if rollup["blocked"] > 0 {
		t.Log("Impediment detected: Blocked tasks present in epic")
	}

	// Cleanup
	for _, task := range tasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicGetIntegration_EpicProgressCalculation tests overall epic progress
func TestEpicGetIntegration_EpicProgressCalculation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E10-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E10')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E10'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E10",
		Title: "Backend Services",
		Slug:  "backend-services",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create two features with different completion rates
	feature1 := &models.Feature{
		Key:    "E10-F01",
		Title:  "Auth Service",
		Slug:   "auth-service",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	feature2 := &models.Feature{
		Key:    "E10-F02",
		Title:  "User Service",
		Slug:   "user-service",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Feature 1: 2 completed, 2 total (50%)
	tasks1 := []*models.Task{
		{Key: "TEST-E10-F01-001", Title: "Task 1", Status: "completed", FeatureID: feature1.ID, EpicID: epic.ID},
		{Key: "TEST-E10-F01-002", Title: "Task 2", Status: "todo", FeatureID: feature1.ID, EpicID: epic.ID},
	}

	// Feature 2: 4 completed, 4 total (100%)
	tasks2 := []*models.Task{
		{Key: "TEST-E10-F02-001", Title: "Task 1", Status: "completed", FeatureID: feature2.ID, EpicID: epic.ID},
		{Key: "TEST-E10-F02-002", Title: "Task 2", Status: "completed", FeatureID: feature2.ID, EpicID: epic.ID},
		{Key: "TEST-E10-F02-003", Title: "Task 3", Status: "completed", FeatureID: feature2.ID, EpicID: epic.ID},
		{Key: "TEST-E10-F02-004", Title: "Task 4", Status: "completed", FeatureID: feature2.ID, EpicID: epic.ID},
	}

	allTasks := append(tasks1, tasks2...)
	for _, task := range allTasks {
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Calculate feature progress for each
	progress1, err := featureRepo.CalculateProgress(ctx, feature1.ID)
	if err != nil {
		t.Fatalf("Failed to calculate feature 1 progress: %v", err)
	}

	progress2, err := featureRepo.CalculateProgress(ctx, feature2.ID)
	if err != nil {
		t.Fatalf("Failed to calculate feature 2 progress: %v", err)
	}

	// Feature 1 should be 50%, Feature 2 should be 100%
	if progress1 != 50.0 {
		t.Errorf("Expected feature 1 progress 50%%, got %f%%", progress1)
	}

	if progress2 != 100.0 {
		t.Errorf("Expected feature 2 progress 100%%, got %f%%", progress2)
	}

	// Calculate epic progress (average of features)
	epicProgress, err := epicRepo.CalculateProgress(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to calculate epic progress: %v", err)
	}

	// Epic should be (50% + 100%) / 2 = 75%
	expectedEpicProgress := 75.0
	if epicProgress != expectedEpicProgress {
		t.Errorf("Expected epic progress %f%%, got %f%%", expectedEpicProgress, epicProgress)
	}

	// Cleanup
	for _, task := range allTasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature1.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature2.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicGetIntegration_JSONOutput tests JSON serialization of epic with rollups
func TestEpicGetIntegration_JSONOutput(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E11-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E11')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E11'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:         "E11",
		Title:       "JSON Test",
		Slug:        "json-test",
		Description: "Test epic for JSON serialization",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create feature
	feature := &models.Feature{
		Key:    "E11-F01",
		Title:  "Test Feature",
		Slug:   "test-feature",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create task
	task := &models.Task{
		Key:       "TEST-E11-F01-001",
		Title:     "Test Task",
		Status:    "completed",
		FeatureID: feature.ID,
		EpicID:    epic.ID,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Get feature rollup
	featureRollup, err := epicRepo.GetFeatureStatusRollup(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to get feature rollup: %v", err)
	}

	// Get task rollup
	taskRollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to get task rollup: %v", err)
	}

	// Verify JSON marshalling of rollups works
	featureRollupJSON, err := json.Marshal(featureRollup)
	if err != nil {
		t.Fatalf("Failed to marshal feature rollup: %v", err)
	}

	taskRollupJSON, err := json.Marshal(taskRollup)
	if err != nil {
		t.Fatalf("Failed to marshal task rollup: %v", err)
	}

	// Verify we can unmarshal back
	var parsedFeatureRollup map[string]int
	if err := json.Unmarshal(featureRollupJSON, &parsedFeatureRollup); err != nil {
		t.Fatalf("Failed to unmarshal feature rollup: %v", err)
	}

	var parsedTaskRollup map[string]int
	if err := json.Unmarshal(taskRollupJSON, &parsedTaskRollup); err != nil {
		t.Fatalf("Failed to unmarshal task rollup: %v", err)
	}

	// Verify epic JSON serialization
	epicJSON, err := json.Marshal(epic)
	if err != nil {
		t.Fatalf("Failed to marshal epic: %v", err)
	}

	var parsedEpic models.Epic
	if err := json.Unmarshal(epicJSON, &parsedEpic); err != nil {
		t.Fatalf("Failed to unmarshal epic: %v", err)
	}

	if parsedEpic.Key != "E11" {
		t.Errorf("Expected key E11, got %s", parsedEpic.Key)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicGetIntegration_MultipleFeatures tests epic with multiple features in different states
func TestEpicGetIntegration_MultipleFeatures(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E12-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E12')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E12'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E12",
		Title: "Multi-Feature Test",
		Slug:  "multi-feature-test",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create 4 features with different statuses and completion rates
	featureData := []struct {
		key       string
		status    string
		completed int
		total     int
	}{
		{"E12-F01", "completed", 5, 5},
		{"E12-F02", "active", 3, 5},
		{"E12-F03", "planning", 0, 3},
		{"E12-F04", "blocked", 1, 4},
	}

	createdFeatures := []*models.Feature{}
	for _, fd := range featureData {
		feature := &models.Feature{
			Key:    fd.key,
			Title:  "Feature " + fd.key,
			Slug:   fd.key,
			Status: fd.status,
			EpicID: epic.ID,
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}
		createdFeatures = append(createdFeatures, feature)

		// Create tasks for this feature
		for i := 1; i <= fd.total; i++ {
			taskStatus := "todo"
			if i <= fd.completed {
				taskStatus = "completed"
			}
			task := &models.Task{
				Key:       "TEST-" + fd.key + "-00" + string(rune('0'+i)),
				Title:     "Task " + string(rune('0'+i)),
				Status:    taskStatus,
				FeatureID: feature.ID,
				EpicID:    epic.ID,
			}
			if err := taskRepo.Create(ctx, task); err != nil {
				t.Fatalf("Failed to create task: %v", err)
			}
		}
	}

	// Get features for epic
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	if len(features) != 4 {
		t.Errorf("Expected 4 features, got %d", len(features))
	}

	// Get feature status rollup
	featureRollup, err := epicRepo.GetFeatureStatusRollup(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to get feature rollup: %v", err)
	}

	// Verify feature status counts
	if featureRollup["completed"] != 1 {
		t.Errorf("Expected 1 completed feature, got %d", featureRollup["completed"])
	}

	if featureRollup["active"] != 1 {
		t.Errorf("Expected 1 active feature, got %d", featureRollup["active"])
	}

	if featureRollup["planning"] != 1 {
		t.Errorf("Expected 1 planning feature, got %d", featureRollup["planning"])
	}

	if featureRollup["blocked"] != 1 {
		t.Errorf("Expected 1 blocked feature, got %d", featureRollup["blocked"])
	}

	// Get task status rollup
	taskRollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to get task rollup: %v", err)
	}

	// Total: 5+3+0+1 = 9 completed, 0+2+3+3 = 8 todo
	if taskRollup["completed"] != 9 {
		t.Errorf("Expected 9 completed tasks, got %d", taskRollup["completed"])
	}

	if taskRollup["todo"] != 8 {
		t.Errorf("Expected 8 todo tasks, got %d", taskRollup["todo"])
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E12-F%'")
	for _, feature := range createdFeatures {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}
