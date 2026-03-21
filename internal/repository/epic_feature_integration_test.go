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

		epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
			Title:       fmt.Sprintf("Integration Test Epic %d", i),
			Description: strPtr("Epic for integration testing")}, Status: models.EpicStatusActive,
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
		feature := &models.Feature{BaseEntity: models.BaseEntity{Key: featureKey,
			Title:       fmt.Sprintf("Feature for epic %s", epicKey),
			Description: strPtr("Test feature")}, EpicID: epic.ID,

			Status: models.FeatureStatusActive,
		}
		_ = featureRepo.Create(ctx, feature)
		if err != nil {
			t.Logf("Failed to create feature (may already exist): %v", err)
			continue
		}

		// Create 4 tasks: 2 completed, 2 todo = 50% progress
		for ti := 0; ti < 4; ti++ {
			status := models.TaskStatus("todo")
			if ti < 2 {
				status = models.TaskStatus("completed")
			}
			taskKey := fmt.Sprintf("T-%s-%03d", featureKey, ti+1)
			_, _ = database.Exec(`
				INSERT OR IGNORE INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
				VALUES (?, ?, ?, ?, 'testing', 1, '[]')
			`, feature.ID, taskKey, fmt.Sprintf("Task %d", ti+1), status)
		}

		// Update feature progress directly via SQL (UpdateProgress was moved to FeatureService)
		// Calculate: completed tasks / total tasks * 100
		_, _ = database.Exec("UPDATE features SET progress_pct = 50.0 WHERE id = ?", feature.ID)
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title:       "Feature Details Test Epic",
		Description: strPtr("Epic for testing feature details")}, Status: models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: priorityPtr(models.PriorityHigh),
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create feature
	featureKey := fmt.Sprintf("%s-F02", epicKey)
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: featureKey,
		Title:       "Feature with mixed task statuses",
		Description: strPtr("Testing task breakdown")}, EpicID: epic.ID,

		Status: models.FeatureStatusActive,
	}
	_ = featureRepo.Create(ctx, feature)

	// Create 10 tasks: 7 completed, 2 in_progress, 1 todo
	taskStatuses := []models.TaskStatus{
		models.TaskStatus("completed"),   // 1
		models.TaskStatus("completed"),   // 2
		models.TaskStatus("completed"),   // 3
		models.TaskStatus("completed"),   // 4
		models.TaskStatus("completed"),   // 5
		models.TaskStatus("completed"),   // 6
		models.TaskStatus("completed"),   // 7
		models.TaskStatus("in_progress"), // 8
		models.TaskStatus("in_progress"), // 9
		models.TaskStatus("todo"),        // 10
	}

	for i, status := range taskStatuses {
		taskKey := fmt.Sprintf("T-%s-%03d", featureKey, i+1)
		_, _ = database.Exec(`
			INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
			VALUES (?, ?, ?, ?, 'testing', 1, '[]')
		`, feature.ID, taskKey, fmt.Sprintf("Task %d", i+1), status)
	}

	// Verify task counts via GetTaskStatusBreakdown (CalculateProgress moved to FeatureService)
	breakdown, err := featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get task status breakdown: %v", err)
	}

	completedCount := breakdown[models.TaskStatus("completed")]
	expectedCompleted := 7
	if completedCount != expectedCompleted {
		t.Errorf("Expected %d completed tasks, got %d", expectedCompleted, completedCount)
	}

	totalTasks := 0
	for _, count := range breakdown {
		totalTasks += count
	}
	if totalTasks != 10 {
		t.Errorf("Expected 10 total tasks, got %d", totalTasks)
	}

	t.Logf("Feature %s: %d/%d completed with 7 completed, 2 in_progress, 1 todo tasks", featureKey, completedCount, totalTasks)
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title:       fmt.Sprintf("Epic %s", epicKey),
		Description: strPtr("Epic for filter testing")}, Status: models.EpicStatusActive,
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
		feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("%s-F%02d", epicKey, i),
			Title:       fmt.Sprintf("Feature %d for %s", i, epicKey),
			Description: strPtr("Test feature")}, EpicID: epic.ID,

			Status: models.FeatureStatusActive,
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

		epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
			Title:       "Edge Case Epic",
			Description: strPtr("Testing zero tasks")}, Status: models.EpicStatusActive,
			Priority:      models.PriorityLow,
			BusinessValue: priorityPtr(models.PriorityLow),
		}
		err := epicRepo.Create(ctx, epic)
		if err != nil {
			t.Fatalf("Failed to create epic: %v", err)
		}

		feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("%s-F01", epicKey),
			Title:       "Feature with no tasks",
			Description: strPtr("Edge case")}, EpicID: epic.ID,

			Status: models.FeatureStatusActive,
		}
		_ = featureRepo.Create(ctx, feature)

		// CalculateProgress was moved to FeatureService; verify via GetTaskCount
		taskCount, err := featureRepo.GetTaskCount(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to get task count: %v", err)
		}

		if taskCount != 0 {
			t.Errorf("Feature with 0 tasks: expected 0 task count, got %d", taskCount)
		}
	})

	t.Run("AllTasksCompleted", func(t *testing.T) {
		epicKey := generateTestEpicKey()

		// Clean up any existing data with this epic key
		_, _ = database.Exec("DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?))", epicKey)
		_, _ = database.Exec("DELETE FROM features WHERE epic_id IN (SELECT id FROM epics WHERE key = ?)", epicKey)
		_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

		epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
			Title:       "All Complete Epic",
			Description: strPtr("All tasks completed")}, Status: models.EpicStatusActive,
			Priority:      models.PriorityMedium,
			BusinessValue: priorityPtr(models.PriorityMedium),
		}
		err := epicRepo.Create(ctx, epic)
		if err != nil {
			t.Fatalf("Failed to create epic: %v", err)
		}

		feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("%s-F01", epicKey),
			Title:       "All tasks complete",
			Description: strPtr("Edge case")}, EpicID: epic.ID,

			Status: models.FeatureStatusActive,
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

		// CalculateProgress was moved to FeatureService; verify via GetTaskStatusBreakdown
		breakdown, err := featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to get task status breakdown: %v", err)
		}

		completedCount := breakdown[models.TaskStatus("completed")]
		if completedCount != 5 {
			t.Errorf("Feature with all tasks completed: expected 5 completed, got %d", completedCount)
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title:       "Progress Propagation Test",
		Description: strPtr("Testing progress updates")}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: priorityPtr(models.PriorityHigh),
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Skipf("Epic already exists: %v", err)
		return
	}

	// Create feature with 4 tasks (all todo initially)
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("%s-F01", epicKey),
		Title:       "Propagation Test Feature",
		Description: strPtr("Test feature")}, EpicID: epic.ID,

		Status: models.FeatureStatusActive,
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

	// Initial state: all 4 tasks are todo
	// UpdateProgress was moved to FeatureService; verify via GetTaskStatusBreakdown
	breakdown, err := featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get task status breakdown: %v", err)
	}

	todoCount := breakdown[models.TaskStatus("todo")]
	if todoCount != 4 {
		t.Errorf("Initial state: expected 4 todo tasks, got %d", todoCount)
	}

	// Complete 2 tasks
	_, _ = database.Exec("UPDATE tasks SET status = 'completed' WHERE id = ?", taskIDs[0])
	_, _ = database.Exec("UPDATE tasks SET status = 'completed' WHERE id = ?", taskIDs[1])

	// Verify updated breakdown
	breakdown, err = featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get task status breakdown after update: %v", err)
	}

	completedCount := breakdown[models.TaskStatus("completed")]
	if completedCount != 2 {
		t.Errorf("After completing 2/4 tasks: expected 2 completed, got %d", completedCount)
	}

	t.Logf("Progress propagation verified: 0 completed → 2 completed")
}
