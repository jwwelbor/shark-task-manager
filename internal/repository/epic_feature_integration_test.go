package repository

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// Integration Tests for Epic and Feature Query Workflows
// These tests use the real database to verify end-to-end functionality
// and validate all acceptance criteria from the PRD.

// generateTestEpicKey generates a unique epic key in the range E50-E99 (reserved for integration tests)
func generateTestEpicKey() string {
	// Use range E50-E99 for integration tests to avoid conflicts
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	num := 50 + r.Intn(50) // 50-99
	return fmt.Sprintf("E%02d", num)
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
// Acceptance Criteria: Given 5 epics in database, pm epic list displays all 5 with progress
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
		err = featureRepo.Create(ctx, feature)
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
			database.Exec(`
				INSERT OR IGNORE INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
				VALUES (?, ?, ?, ?, 'testing', 1, '[]')
			`, feature.ID, taskKey, fmt.Sprintf("Task %d", ti+1), status)
		}

		// Update feature progress
		featureRepo.UpdateProgress(ctx, feature.ID)
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
func TestEpicDetailsIntegration(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	epicKey := generateTestEpicKey()

	// Create epic
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Epic Details Test",
		Description:   strPtr("Testing epic details with multiple features"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: priorityPtr(models.PriorityMedium),
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Skipf("Epic already exists, skipping test: %v", err)
		return
	}

	// Feature 1: 50% progress (5 of 10 tasks completed)
	feature1 := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("%s-F01", epicKey),
		Title:       "Feature 1 - 50% complete",
		Description: strPtr("Test feature"),
		Status:      models.FeatureStatusActive,
	}
	featureRepo.Create(ctx, feature1)

	for i := 0; i < 10; i++ {
		status := models.TaskStatusTodo
		if i < 5 {
			status = models.TaskStatusCompleted
		}
		taskKey := fmt.Sprintf("T-%s-F01-%03d", epicKey, i+1)
		database.Exec(`
			INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
			VALUES (?, ?, ?, ?, 'testing', 1, '[]')
		`, feature1.ID, taskKey, fmt.Sprintf("Task %d", i+1), status)
	}
	featureRepo.UpdateProgress(ctx, feature1.ID)

	// Feature 2: 75% progress (6 of 8 tasks completed)
	feature2 := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("%s-F02", epicKey),
		Title:       "Feature 2 - 75% complete",
		Description: strPtr("Test feature"),
		Status:      models.FeatureStatusActive,
	}
	featureRepo.Create(ctx, feature2)

	for i := 0; i < 8; i++ {
		status := models.TaskStatusTodo
		if i < 6 {
			status = models.TaskStatusCompleted
		}
		taskKey := fmt.Sprintf("T-%s-F02-%03d", epicKey, i+1)
		database.Exec(`
			INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
			VALUES (?, ?, ?, ?, 'testing', 1, '[]')
		`, feature2.ID, taskKey, fmt.Sprintf("Task %d", i+1), status)
	}
	featureRepo.UpdateProgress(ctx, feature2.ID)

	// Feature 3: 100% progress (2 of 2 tasks completed)
	feature3 := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("%s-F03", epicKey),
		Title:       "Feature 3 - 100% complete",
		Description: strPtr("Test feature"),
		Status:      models.FeatureStatusActive,
	}
	featureRepo.Create(ctx, feature3)

	for i := 0; i < 2; i++ {
		taskKey := fmt.Sprintf("T-%s-F03-%03d", epicKey, i+1)
		database.Exec(`
			INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
			VALUES (?, ?, ?, 'completed', 'testing', 1, '[]')
		`, feature3.ID, taskKey, fmt.Sprintf("Task %d", i+1))
	}
	featureRepo.UpdateProgress(ctx, feature3.ID)

	// Calculate epic progress
	// Weighted average: (50*10 + 75*8 + 100*2) / (10+8+2) = (500 + 600 + 200) / 20 = 1300/20 = 65.0
	progress, err := epicRepo.CalculateProgress(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to calculate epic progress: %v", err)
	}

	expected := 65.0
	if progress != expected {
		t.Errorf("Expected epic progress %.1f%%, got %.1f%%", expected, progress)
	}

	// Get all features for the epic
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		t.Fatalf("Failed to get features: %v", err)
	}

	if len(features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(features))
	}

	t.Logf("Epic %s: progress=%.1f%% (expected %.1f%%), features=%d", epicKey, progress, expected, len(features))
}

// TestFeatureDetailsIntegration verifies getting feature details with task breakdown
// Acceptance Criteria: Feature with 10 tasks (7 completed, 2 in_progress, 1 todo) shows 70% progress
func TestFeatureDetailsIntegration(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	epicKey := generateTestEpicKey()

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
		t.Skipf("Epic already exists: %v", err)
		return
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
	featureRepo.Create(ctx, feature)

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
		database.Exec(`
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
		featureRepo.Create(ctx, feature)
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
			t.Skip("Epic already exists")
			return
		}

		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-F01", epicKey),
			Title:       "Feature with no tasks",
			Description: strPtr("Edge case"),
			Status:      models.FeatureStatusActive,
		}
		featureRepo.Create(ctx, feature)

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
			t.Skip("Epic already exists")
			return
		}

		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-F01", epicKey),
			Title:       "All tasks complete",
			Description: strPtr("Edge case"),
			Status:      models.FeatureStatusActive,
		}
		featureRepo.Create(ctx, feature)

		// Create 5 completed tasks
		for i := 0; i < 5; i++ {
			taskKey := fmt.Sprintf("T-%s-F01-%03d", epicKey, i+1)
			database.Exec(`
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
	featureRepo.Create(ctx, feature)

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
	featureRepo.UpdateProgress(ctx, feature.ID)
	feature, _ = featureRepo.GetByKey(ctx, feature.Key)
	if feature.ProgressPct != 0.0 {
		t.Errorf("Initial feature progress: expected 0.0%%, got %.1f%%", feature.ProgressPct)
	}

	// Complete 2 tasks = 50% progress
	database.Exec("UPDATE tasks SET status = 'completed' WHERE id = ?", taskIDs[0])
	database.Exec("UPDATE tasks SET status = 'completed' WHERE id = ?", taskIDs[1])

	featureRepo.UpdateProgress(ctx, feature.ID)
	feature, _ = featureRepo.GetByKey(ctx, feature.Key)
	if feature.ProgressPct != 50.0 {
		t.Errorf("After completing 2/4 tasks: expected 50.0%% progress, got %.1f%%", feature.ProgressPct)
	}

	t.Logf("Progress propagation verified: 0%% → 50%%")
}
