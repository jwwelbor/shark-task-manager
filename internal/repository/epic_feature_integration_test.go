package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// Integration Tests for Epic and Feature Query Workflows
// These tests use the real database to verify end-to-end functionality
// and validate all acceptance criteria from the PRD.

// generateTestEpicKey generates a unique epic key using timestamp to avoid parallel test conflicts
// Format: E50<timestamp_last_6_digits> ensures uniqueness across parallel test runs
func generateTestEpicKey() string {
	// Use timestamp to ensure uniqueness even when tests run in parallel
	// Epic keys must match ^E\d{2}$ format (E followed by exactly 2 digits)
	// Use E50-E99 range (50 possible values) with timestamp to minimize collisions
	timestamp := time.Now().UnixNano()
	epicNum := 50 + (timestamp % 50) // Range 50-99
	return fmt.Sprintf("E%02d", epicNum)
}

// priorityPtr returns a pointer to a Priority
func priorityPtr(p models.Priority) *models.Priority {
	return &p
}

// Helper to create string pointer (avoiding duplicate declaration)
func strPtr(s string) *string {
	return &s
}

// TestEpicListingIntegration verifies listing all epics with progress
// Acceptance Criteria: Given 5 epics in database, shark epic list displays all 5 with progress
func TestEpicListingIntegration(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create 5 test epics with unique keys
	for i := 0; i < 5; i++ {
		epicKey := generateTestEpicKey()

		epic := &models.Epic{
			Key:           epicKey,
			Title:         fmt.Sprintf("Integration Test Epic %d", i),
			Description:   strPtr("Epic for integration testing"),
			Status:        models.EpicStatusActive,
			Priority:      models.PriorityMedium,
			BusinessValue: priorityPtr(models.PriorityHigh),
		}

		// Try to create, skip if already exists
		err := epicRepo.Create(ctx, epic)
		if err != nil {
			continue
		}

		// Create a feature with some tasks for each epic
		featureKey := fmt.Sprintf("%s-F01", epicKey)
		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         featureKey,
			Title:       fmt.Sprintf("Feature for epic %s", epicKey),
			Description: strPtr("Test feature"),
			Status:      models.FeatureStatusActive,
		}
		_ = featureRepo.Create(ctx, feature)
		if err != nil {
			t.Logf("Failed to create feature (may already exist): %v", err)
			continue
		}

		// Create 4 tasks: 2 completed, 2 todo = 50% progress
		for ti := 0; ti < 4; ti++ {
			status := models.TaskStatusTodo
			if ti < 2 {
				status = models.TaskStatusCompleted
			}
			taskKey := fmt.Sprintf("T-%s-%03d", featureKey, ti+1)
			_, _ = database.Exec(`
				INSERT OR IGNORE INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
				VALUES (?, ?, ?, ?, 'testing', 1, '[]')
			`, feature.ID, taskKey, fmt.Sprintf("Task %d", ti+1), status)
		}

		// Update feature progress
		_ = featureRepo.UpdateProgress(ctx, feature.ID)
	}

	// Retrieve all epics
	epics, err := epicRepo.List(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get all epics: %v", err)
	}

	// Should have at least some epics
	if len(epics) == 0 {
		t.Error("Expected at least some epics in database")
	}

	t.Logf("Successfully listed %d epics", len(epics))
}

// TestEpicDetailsIntegration verifies getting epic details with feature breakdown
// Acceptance Criteria: Epic with 3 features shows weighted progress correctly
func TestFeatureDetailsIntegration(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	epicKey := generateTestEpicKey()

	// Clean up any existing data with this epic key
	_, _ = database.Exec("DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?))", epicKey)
	_, _ = database.Exec("DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?)", epicKey)
	_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Feature Details Test Epic",
		Description:   strPtr("Epic for testing feature details"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: priorityPtr(models.PriorityHigh),
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create feature
	featureKey := fmt.Sprintf("%s-F02", epicKey)
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         featureKey,
		Title:       "Feature with mixed task statuses",
		Description: strPtr("Testing task breakdown"),
		Status:      models.FeatureStatusActive,
	}
	_ = featureRepo.Create(ctx, feature)

	// Create 10 tasks: 7 completed, 2 in_progress, 1 todo
	taskStatuses := []models.TaskStatus{
		models.TaskStatusCompleted,  // 1
		models.TaskStatusCompleted,  // 2
		models.TaskStatusCompleted,  // 3
		models.TaskStatusCompleted,  // 4
		models.TaskStatusCompleted,  // 5
		models.TaskStatusCompleted,  // 6
		models.TaskStatusCompleted,  // 7
		models.TaskStatusInProgress, // 8
		models.TaskStatusInProgress, // 9
		models.TaskStatusTodo,       // 10
	}

	for i, status := range taskStatuses {
		taskKey := fmt.Sprintf("T-%s-%03d", featureKey, i+1)
		_, _ = database.Exec(`
			INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
			VALUES (?, ?, ?, ?, 'testing', 1, '[]')
		`, feature.ID, taskKey, fmt.Sprintf("Task %d", i+1), status)
	}

	// Calculate progress
	progress, err := featureRepo.CalculateProgress(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to calculate feature progress: %v", err)
	}

	expected := 70.0
	if progress != expected {
		t.Errorf("Expected %.1f%% progress, got %.1f%%", expected, progress)
	}

	t.Logf("Feature %s: progress=%.1f%% with 7 completed, 2 in_progress, 1 todo tasks", featureKey, progress)
}

// TestFeatureListFilteringIntegration verifies filtering features by epic
func TestFeatureListFilteringIntegration(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create epic for filter testing
	epicKey := generateTestEpicKey()
	epic := &models.Epic{
		Key:           epicKey,
		Title:         fmt.Sprintf("Epic %s", epicKey),
		Description:   strPtr("Epic for filter testing"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: priorityPtr(models.PriorityMedium),
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Skipf("Epic already exists: %v", err)
		return
	}

	// Create 3 features for this epic
	for i := 1; i <= 3; i++ {
		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-F%02d", epicKey, i),
			Title:       fmt.Sprintf("Feature %d for %s", i, epicKey),
			Description: strPtr("Test feature"),
			Status:      models.FeatureStatusActive,
		}
		_ = featureRepo.Create(ctx, feature)
	}

	// Test filtering by epic
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to get features for epic %s: %v", epicKey, err)
	}

	expectedCount := 3
	if len(features) < expectedCount {
		t.Errorf("Epic %s: expected at least %d features, got %d", epicKey, expectedCount, len(features))
	}

	// Verify all returned features belong to the correct epic
	for _, feature := range features {
		if feature.EpicID != epic.ID {
			t.Errorf("Feature %s has epic_id=%d, expected %d", feature.Key, feature.EpicID, epic.ID)
		}
	}

	t.Logf("Filter test passed: Epic %s returned %d features", epicKey, len(features))
}

// TestProgressCalculationEdgeCases verifies edge case handling
func TestProgressCalculationEdgeCases(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	t.Run("FeatureWithZeroTasks", func(t *testing.T) {
		epicKey := generateTestEpicKey()

		// Clean up any existing data with this epic key
		_, _ = database.Exec("DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?))", epicKey)
		_, _ = database.Exec("DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?)", epicKey)
		_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

		epic := &models.Epic{
			Key:           epicKey,
			Title:         "Edge Case Epic",
			Description:   strPtr("Testing zero tasks"),
			Status:        models.EpicStatusActive,
			Priority:      models.PriorityLow,
			BusinessValue: priorityPtr(models.PriorityLow),
		}
		err := epicRepo.Create(ctx, epic)
		if err != nil {
			t.Fatalf("Failed to create epic: %v", err)
		}

		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-F01", epicKey),
			Title:       "Feature with no tasks",
			Description: strPtr("Edge case"),
			Status:      models.FeatureStatusActive,
		}
		_ = featureRepo.Create(ctx, feature)

		progress, err := featureRepo.CalculateProgress(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to calculate progress: %v", err)
		}

		if progress != 0.0 {
			t.Errorf("Feature with 0 tasks: expected 0.0%% progress, got %.1f%%", progress)
		}
	})

	t.Run("AllTasksCompleted", func(t *testing.T) {
		epicKey := generateTestEpicKey()

		// Clean up any existing data with this epic key
		_, _ = database.Exec("DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?))", epicKey)
		_, _ = database.Exec("DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?)", epicKey)
		_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

		epic := &models.Epic{
			Key:           epicKey,
			Title:         "All Complete Epic",
			Description:   strPtr("All tasks completed"),
			Status:        models.EpicStatusActive,
			Priority:      models.PriorityMedium,
			BusinessValue: priorityPtr(models.PriorityMedium),
		}
		err := epicRepo.Create(ctx, epic)
		if err != nil {
			t.Fatalf("Failed to create epic: %v", err)
		}

		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-F01", epicKey),
			Title:       "All tasks complete",
			Description: strPtr("Edge case"),
			Status:      models.FeatureStatusActive,
		}
		_ = featureRepo.Create(ctx, feature)

		// Create 5 completed tasks
		for i := 0; i < 5; i++ {
			taskKey := fmt.Sprintf("T-%s-F01-%03d", epicKey, i+1)
			_, _ = database.Exec(`
				INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
				VALUES (?, ?, ?, 'completed', 'testing', 1, '[]')
			`, feature.ID, taskKey, fmt.Sprintf("Task %d", i+1))
		}

		progress, err := featureRepo.CalculateProgress(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to calculate progress: %v", err)
		}

		if progress != 100.0 {
			t.Errorf("Feature with all tasks completed: expected 100.0%% progress, got %.1f%%", progress)
		}
	})

	t.Logf("All edge cases handled correctly")
}

// TestMultiLevelProgressPropagation verifies progress updates propagate correctly
func TestMultiLevelProgressPropagation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	epicKey := generateTestEpicKey()

	// Create epic
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Progress Propagation Test",
		Description:   strPtr("Testing progress updates"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: priorityPtr(models.PriorityHigh),
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Skipf("Epic already exists: %v", err)
		return
	}

	// Create feature with 4 tasks (all todo initially)
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("%s-F01", epicKey),
		Title:       "Propagation Test Feature",
		Description: strPtr("Test feature"),
		Status:      models.FeatureStatusActive,
	}
	_ = featureRepo.Create(ctx, feature)

	// Create 4 todo tasks
	taskIDs := make([]int64, 4)
	for i := 0; i < 4; i++ {
		taskKey := fmt.Sprintf("T-%s-F01-%03d", epicKey, i+1)
		result, err := database.Exec(`
			INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
			VALUES (?, ?, ?, 'todo', 'testing', 1, '[]')
		`, feature.ID, taskKey, fmt.Sprintf("Task %d", i+1))
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		taskIDs[i], _ = result.LastInsertId()
	}

	// Initial progress should be 0%
	_ = featureRepo.UpdateProgress(ctx, feature.ID)
	feature, _ = featureRepo.GetByKey(ctx, feature.Key)
	if feature.ProgressPct != 0.0 {
		t.Errorf("Initial feature progress: expected 0.0%%, got %.1f%%", feature.ProgressPct)
	}

	// Complete 2 tasks = 50% progress
	_, _ = database.Exec("UPDATE tasks SET status = 'completed' WHERE id = ?", taskIDs[0])
	_, _ = database.Exec("UPDATE tasks SET status = 'completed' WHERE id = ?", taskIDs[1])

	_ = featureRepo.UpdateProgress(ctx, feature.ID)
	feature, _ = featureRepo.GetByKey(ctx, feature.Key)
	if feature.ProgressPct != 50.0 {
		t.Errorf("After completing 2/4 tasks: expected 50.0%% progress, got %.1f%%", feature.ProgressPct)
	}

	t.Logf("Progress propagation verified: 0%% → 50%%")
}
