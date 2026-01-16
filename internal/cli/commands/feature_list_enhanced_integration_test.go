package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestFeatureListIntegration_ProgressFormatValidation tests progress formatting
func TestFeatureListIntegration_ProgressFormatValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E13-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E13')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E13'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E13",
		Title: "Progress Test",
		Slug:  "progress-test",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create features with various progress levels
	featureData := []struct {
		key        string
		completed  int
		total      int
		expected   float64
	}{
		{"E13-F01", 0, 4, 0.0},      // 0%
		{"E13-F02", 1, 4, 25.0},     // 25%
		{"E13-F03", 2, 4, 50.0},     // 50%
		{"E13-F04", 3, 4, 75.0},     // 75%
		{"E13-F05", 4, 4, 100.0},    // 100%
		{"E13-F06", 0, 0, 0.0},      // Empty (0%)
	}

	features := make([]*models.Feature, 0)
	createdTasks := make([]*models.Task, 0)

	for _, fd := range featureData {
		feature := &models.Feature{
			Key:    fd.key,
			Title:  "Feature " + fd.key,
			Slug:   fd.key,
			EpicID: epic.ID,
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}
		features = append(features, feature)

		// Create tasks
		for i := 1; i <= fd.total; i++ {
			status := "todo"
			if i <= fd.completed {
				status = "completed"
			}
			task := &models.Task{
				Key:       "TEST-" + fd.key + "-00" + string(rune('0'+i)),
				Title:     "Task " + string(rune('0'+i)),
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

	// Verify progress calculation for each feature
	for i, fd := range featureData {
		progress, err := featureRepo.CalculateProgress(ctx, features[i].ID)
		if err != nil {
			t.Fatalf("Failed to calculate progress for %s: %v", fd.key, err)
		}

		if progress != fd.expected {
			t.Errorf("Feature %s: expected progress %.1f%%, got %.1f%%", fd.key, fd.expected, progress)
		}

		// Verify progress is valid (0-100)
		if progress < 0 || progress > 100 {
			t.Errorf("Feature %s: progress out of valid range (0-100): %f", fd.key, progress)
		}
	}

	// Cleanup
	for _, task := range createdTasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}
	for _, feature := range features {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestFeatureListIntegration_JSONOutputValidation tests JSON serialization in list context
func TestFeatureListIntegration_JSONOutputValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E14-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E14')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E14'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E14",
		Title: "JSON Output Test",
		Slug:  "json-output-test",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create multiple features
	featureCount := 3
	features := make([]*models.Feature, 0)

	for i := 1; i <= featureCount; i++ {
		keyNum := i
		feature := &models.Feature{
			Key:         "E14-F0" + string(rune('0'+keyNum)),
			Title:       "Feature " + string(rune('0'+keyNum)),
			Slug:        "feature-" + string(rune('0'+keyNum)),
			EpicID:      epic.ID,
			Description: "Test feature for JSON output",
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}
		features = append(features, feature)

		// Create tasks for each feature
		for j := 1; j <= 2; j++ {
			task := &models.Task{
				Key:       "TEST-E14-F0" + string(rune('0'+keyNum)) + "-00" + string(rune('0'+j)),
				Title:     "Task " + string(rune('0'+j)),
				Status:    "todo",
				FeatureID: feature.ID,
				EpicID:    epic.ID,
			}
			if err := taskRepo.Create(ctx, task); err != nil {
				t.Fatalf("Failed to create task: %v", err)
			}
		}
	}

	// List features for epic
	listFeatures, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to list features: %v", err)
	}

	if len(listFeatures) != featureCount {
		t.Errorf("Expected %d features, got %d", featureCount, len(listFeatures))
	}

	// Verify each feature can be JSON marshalled
	for _, feature := range listFeatures {
		featureJSON, err := json.Marshal(feature)
		if err != nil {
			t.Fatalf("Failed to marshal feature %s to JSON: %v", feature.Key, err)
		}

		var parsedFeature models.Feature
		if err := json.Unmarshal(featureJSON, &parsedFeature); err != nil {
			t.Fatalf("Failed to unmarshal feature %s from JSON: %v", feature.Key, err)
		}

		if parsedFeature.Key != feature.Key {
			t.Errorf("Feature key mismatch: expected %s, got %s", feature.Key, parsedFeature.Key)
		}
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E14-F%'")
	for _, feature := range features {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestFeatureListIntegration_HealthIndicatorCalculation tests health indicator generation
func TestFeatureListIntegration_HealthIndicatorCalculation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E15-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E15')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E15'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E15",
		Title: "Health Test",
		Slug:  "health-test",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create features with different health indicators
	// Feature 1: Healthy (on track)
	feature1 := &models.Feature{
		Key:    "E15-F01",
		Title:  "Healthy Feature",
		Slug:   "healthy-feature",
		Status: "active",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create feature1: %v", err)
	}

	// Feature 2: At Risk (some blocking tasks)
	feature2 := &models.Feature{
		Key:    "E15-F02",
		Title:  "At Risk Feature",
		Slug:   "at-risk-feature",
		Status: "active",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create feature2: %v", err)
	}

	// Feature 3: Blocked (all tasks blocked)
	feature3 := &models.Feature{
		Key:    "E15-F03",
		Title:  "Blocked Feature",
		Slug:   "blocked-feature",
		Status: "blocked",
		EpicID: epic.ID,
	}
	if err := featureRepo.Create(ctx, feature3); err != nil {
		t.Fatalf("Failed to create feature3: %v", err)
	}

	// Feature 1: Add healthy tasks (mix of completed and in progress)
	for i := 1; i <= 4; i++ {
		status := "completed"
		if i > 2 {
			status = "in_progress"
		}
		task := &models.Task{
			Key:       "TEST-E15-F01-00" + string(rune('0'+i)),
			Title:     "Task " + string(rune('0'+i)),
			Status:    status,
			FeatureID: feature1.ID,
			EpicID:    epic.ID,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Feature 2: Add at-risk tasks (some blocked)
	tasks2 := []string{"completed", "in_progress", "blocked", "todo"}
	for i, status := range tasks2 {
		task := &models.Task{
			Key:       "TEST-E15-F02-00" + string(rune('0'+i+1)),
			Title:     "Task " + string(rune('0'+i+1)),
			Status:    status,
			FeatureID: feature2.ID,
			EpicID:    epic.ID,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Feature 3: Add all blocked tasks
	for i := 1; i <= 3; i++ {
		task := &models.Task{
			Key:       "TEST-E15-F03-00" + string(rune('0'+i)),
			Title:     "Task " + string(rune('0'+i)),
			Status:    "blocked",
			FeatureID: feature3.ID,
			EpicID:    epic.ID,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Get task counts for each feature to calculate health
	for _, feature := range []*models.Feature{feature1, feature2, feature3} {
		tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to list tasks for feature %s: %v", feature.Key, err)
		}

		// Count statuses
		statusCounts := make(map[string]int)
		for _, task := range tasks {
			statusCounts[task.Status]++
		}

		// Determine health based on blocked task percentage
		var health string
		blockedCount := statusCounts["blocked"]
		totalCount := len(tasks)

		if totalCount == 0 {
			health = "unknown"
		} else {
			blockedPercent := float64(blockedCount) / float64(totalCount) * 100
			if blockedPercent == 0 {
				health = "healthy"
			} else if blockedPercent < 50 {
				health = "at-risk"
			} else {
				health = "blocked"
			}
		}

		// Verify expected health status
		switch feature.Key {
		case "E15-F01":
			if health != "healthy" {
				t.Errorf("Feature %s expected health 'healthy', got '%s'", feature.Key, health)
			}
		case "E15-F02":
			if health != "at-risk" {
				t.Errorf("Feature %s expected health 'at-risk', got '%s'", feature.Key, health)
			}
		case "E15-F03":
			if health != "blocked" {
				t.Errorf("Feature %s expected health 'blocked', got '%s'", feature.Key, health)
			}
		}
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E15-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id IN (?, ?, ?)", feature1.ID, feature2.ID, feature3.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestFeatureListIntegration_MultipleEpicsFeatures tests listing across multiple epics
func TestFeatureListIntegration_MultipleEpicsFeatures(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-E16-F% OR TEST-E17-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key IN ('E16', 'E17'))")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E16', 'E17')")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	// Create two epics
	epic1 := &models.Epic{
		Key:   "E16",
		Title: "Epic 1",
		Slug:  "epic-1",
	}
	if err := epicRepo.Create(ctx, epic1); err != nil {
		t.Fatalf("Failed to create epic1: %v", err)
	}

	epic2 := &models.Epic{
		Key:   "E17",
		Title: "Epic 2",
		Slug:  "epic-2",
	}
	if err := epicRepo.Create(ctx, epic2); err != nil {
		t.Fatalf("Failed to create epic2: %v", err)
	}

	// Create features for each epic
	features1 := []*models.Feature{}
	for i := 1; i <= 2; i++ {
		feature := &models.Feature{
			Key:    "E16-F0" + string(rune('0'+i)),
			Title:  "Feature " + string(rune('0'+i)),
			Slug:   "feature-" + string(rune('0'+i)),
			EpicID: epic1.ID,
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}
		features1 = append(features1, feature)
	}

	features2 := []*models.Feature{}
	for i := 1; i <= 3; i++ {
		feature := &models.Feature{
			Key:    "E17-F0" + string(rune('0'+i)),
			Title:  "Feature " + string(rune('0'+i)),
			Slug:   "feature-" + string(rune('0'+i)),
			EpicID: epic2.ID,
		}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create feature: %v", err)
		}
		features2 = append(features2, feature)
	}

	// List features for each epic and verify isolation
	list1, err := featureRepo.ListByEpic(ctx, epic1.ID)
	if err != nil {
		t.Fatalf("Failed to list features for epic1: %v", err)
	}

	list2, err := featureRepo.ListByEpic(ctx, epic2.ID)
	if err != nil {
		t.Fatalf("Failed to list features for epic2: %v", err)
	}

	// Verify counts
	if len(list1) != 2 {
		t.Errorf("Expected 2 features for epic1, got %d", len(list1))
	}

	if len(list2) != 3 {
		t.Errorf("Expected 3 features for epic2, got %d", len(list2))
	}

	// Verify no cross-contamination
	for _, feature := range list1 {
		if feature.EpicID != epic1.ID {
			t.Errorf("Feature %s belongs to wrong epic (got epic %d, want %d)", feature.Key, feature.EpicID, epic1.ID)
		}
	}

	for _, feature := range list2 {
		if feature.EpicID != epic2.ID {
			t.Errorf("Feature %s belongs to wrong epic (got epic %d, want %d)", feature.Key, feature.EpicID, epic2.ID)
		}
	}

	// Cleanup
	for _, feature := range features1 {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}
	for _, feature := range features2 {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	}
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id IN (?, ?)", epic1.ID, epic2.ID)
}

// TestFeatureListIntegration_StatusOverrideDetection tests detection of status overrides
func TestFeatureListIntegration_StatusOverrideDetection(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E18-F%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E18'")

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)

	// Create epic
	epic := &models.Epic{
		Key:   "E18",
		Title: "Status Override Test",
		Slug:  "status-override-test",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create feature with status_override field
	feature := &models.Feature{
		Key:            "E18-F01",
		Title:          "Feature with Override",
		Slug:           "feature-with-override",
		EpicID:         epic.ID,
		Status:         "active",
		StatusOverride: true,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Retrieve and verify
	retrieved, err := featureRepo.GetByID(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	if retrieved.StatusOverride != true {
		t.Errorf("Expected status override to be true, got %v", retrieved.StatusOverride)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}
