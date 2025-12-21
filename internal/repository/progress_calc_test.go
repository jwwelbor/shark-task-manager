package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// NOTE: This file tests repository methods which directly interact with the database.
// Repository tests SHOULD use the real database to verify SQL queries work correctly.
//
// For other layers (services, CLI, etc.), MOCK the repository methods instead of using
// the real database. You already know what the repository will return, so mock it.
//
// Testing Philosophy:
// - Repository layer: Real database (tests SQL correctness)
// - Service layer: Mock repositories (tests business logic)
// - CLI layer: Mock services (tests command handling)
// - Integration tests: Real database (tests end-to-end)

// Helper to create epic, feature, and tasks for testing.
// Uses INSERT OR IGNORE pattern like SeedTestData() to handle shared test database.
// Epics E90-E99 are reserved for progress tests.
func setupProgressTest(t *testing.T, epicNum int, featureNum int, taskStatuses []models.TaskStatus) (int64, int64) {
	database := test.GetTestDB()

	// Use E90-E99 range for progress tests
	epicKey := fmt.Sprintf("E%02d", epicNum)
	featureKey := fmt.Sprintf("E%02d-F%02d", epicNum, featureNum)

	// Create epic via SQL with INSERT OR IGNORE
	_, err := database.Exec(`
		INSERT OR IGNORE INTO epics (key, title, description, status, priority)
		VALUES (?, 'Progress Test Epic', 'Test epic', 'active', 'medium')
	`, epicKey)
	if err != nil {
		t.Fatalf("Failed to create epic %s: %v", epicKey, err)
	}

	// Always query for the epic ID since INSERT OR IGNORE may return unreliable LastInsertId()
	var epicID int64
	err = database.QueryRow("SELECT id FROM epics WHERE key = ?", epicKey).Scan(&epicID)
	if err != nil {
		t.Fatalf("Failed to get epic ID for %s: %v", epicKey, err)
	}
	t.Logf("Using epic %s with ID=%d", epicKey, epicID)

	// Verify epic exists before creating feature
	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM epics WHERE id = ?", epicID).Scan(&count)
	if err != nil || count == 0 {
		t.Fatalf("Epic with ID %d (key=%s) does not exist in database", epicID, epicKey)
	}

	// Clean up any existing feature with this specific key from previous test runs to ensure test isolation
	_, _ = database.Exec("DELETE FROM features WHERE key = ?", featureKey)

	// Create feature via SQL with INSERT OR IGNORE
	if epicID == 0 {
		t.Fatalf("Cannot create feature %s: epicID is 0 (epic %s does not exist)", featureKey, epicKey)
	}
	_, err = database.Exec(`
		INSERT OR IGNORE INTO features (epic_id, key, title, description, status)
		VALUES (?, ?, 'Progress Test Feature', 'Test feature', 'active')
	`, epicID, featureKey)
	if err != nil {
		t.Fatalf("Failed to create feature %s with epicID=%d: %v", featureKey, epicID, err)
	}

	// Always query for the feature ID since INSERT OR IGNORE may return unreliable LastInsertId()
	var featureID int64
	err = database.QueryRow("SELECT id FROM features WHERE key = ?", featureKey).Scan(&featureID)
	if err != nil {
		t.Fatalf("Failed to get feature ID for %s: %v", featureKey, err)
	}

	// Delete and recreate tasks for this feature (so we can control task statuses)
	_, _ = database.Exec("DELETE FROM tasks WHERE feature_id = ?", featureID)

	// Create tasks
	for i, status := range taskStatuses {
		taskKey := fmt.Sprintf("%s-T%03d", featureKey, i+1)
		_, err := database.Exec(`
			INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
			VALUES (?, ?, ?, ?, 'testing', 1, '[]')
		`, featureID, taskKey, fmt.Sprintf("Task %d", i+1), status)
		if err != nil {
			t.Fatalf("Failed to create task %s with status %s: %v", taskKey, status, err)
		}
	}

	return epicID, featureID
}

// TestFeatureProgress_NoTasks verifies 0% progress when feature has no tasks
func TestFeatureProgress_NoTasks(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)

	_, featureID := setupProgressTest(t, 90, 1, []models.TaskStatus{})

	progress, err := featureRepo.CalculateProgress(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to calculate progress: %v", err)
	}

	if progress != 0.0 {
		t.Errorf("Expected 0.0%% progress for feature with no tasks, got %.1f%%", progress)
	}
}

// TestFeatureProgress_CompletedTasks verifies counting completed tasks
func TestFeatureProgress_CompletedTasks(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)

	// 7 completed, 2 in_progress, 1 todo = 70% progress
	statuses := []models.TaskStatus{
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusInProgress,
		models.TaskStatusInProgress,
		models.TaskStatusTodo,
	}

	_, featureID := setupProgressTest(t, 91, 1, statuses)

	progress, err := featureRepo.CalculateProgress(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to calculate progress: %v", err)
	}

	expected := 70.0
	if progress != expected {
		t.Errorf("Expected %.1f%% progress, got %.1f%%", expected, progress)
	}
}

// TestFeatureProgress_CompletedAndArchived verifies both completed and archived count as done
func TestFeatureProgress_CompletedAndArchived(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)

	// 5 completed + 2 archived = 7 done out of 10 = 70%
	statuses := []models.TaskStatus{
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusArchived,
		models.TaskStatusArchived,
		models.TaskStatusTodo,
		models.TaskStatusTodo,
		models.TaskStatusTodo,
	}

	_, featureID := setupProgressTest(t, 92, 1, statuses)

	progress, err := featureRepo.CalculateProgress(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to calculate progress: %v", err)
	}

	expected := 70.0
	if progress != expected {
		t.Errorf("Expected %.1f%% progress (completed + archived), got %.1f%%", expected, progress)
	}
}

// TestFeatureProgress_AllCompleted verifies 100% progress
func TestFeatureProgress_AllCompleted(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)

	statuses := []models.TaskStatus{
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
	}

	_, featureID := setupProgressTest(t, 93, 1, statuses)

	progress, err := featureRepo.CalculateProgress(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to calculate progress: %v", err)
	}

	expected := 100.0
	if progress != expected {
		t.Errorf("Expected %.1f%% progress, got %.1f%%", expected, progress)
	}
}

// TestFeatureProgress_NoneCompleted verifies 0% with tasks present
func TestFeatureProgress_NoneCompleted(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)

	statuses := []models.TaskStatus{
		models.TaskStatusTodo,
		models.TaskStatusTodo,
		models.TaskStatusTodo,
		models.TaskStatusInProgress,
		models.TaskStatusInProgress,
	}

	_, featureID := setupProgressTest(t, 94, 1, statuses)

	progress, err := featureRepo.CalculateProgress(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to calculate progress: %v", err)
	}

	expected := 0.0
	if progress != expected {
		t.Errorf("Expected %.1f%% progress, got %.1f%%", expected, progress)
	}
}

// TestFeatureProgress_BlockedNotCounted verifies blocked tasks don't count as completed
func TestFeatureProgress_BlockedNotCounted(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)

	// 1 completed, 2 blocked, 1 todo = 25% (only completed counts)
	statuses := []models.TaskStatus{
		models.TaskStatusCompleted,
		models.TaskStatusBlocked,
		models.TaskStatusBlocked,
		models.TaskStatusTodo,
	}

	_, featureID := setupProgressTest(t, 95, 1, statuses)

	progress, err := featureRepo.CalculateProgress(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to calculate progress: %v", err)
	}

	expected := 25.0
	if progress != expected {
		t.Errorf("Expected %.1f%% progress, got %.1f%%", expected, progress)
	}
}

// TestEpicProgress_NoFeatures verifies 0% when epic has no features
func TestEpicProgress_NoFeatures(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	epicRepo := NewEpicRepository(db)

	// Create epic with no features
	epicID, _ := setupProgressTest(t, 96, 1, []models.TaskStatus{})

	// Delete the feature so epic has none
	_, _ = test.GetTestDB().Exec("DELETE FROM features WHERE epic_id = ?", epicID)

	progress, err := epicRepo.CalculateProgress(ctx, epicID)
	if err != nil {
		t.Fatalf("Failed to calculate epic progress: %v", err)
	}

	if progress != 0.0 {
		t.Errorf("Expected 0.0%% progress for epic with no features, got %.1f%%", progress)
	}
}

// TestEpicProgress_WeightedAverage verifies epic progress is weighted by task count
func TestEpicProgress_WeightedAverage(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	// Feature 1: 50% with 10 tasks (E97-F01)
	statuses1 := make([]models.TaskStatus, 10)
	for i := 0; i < 10; i++ {
		if i < 5 {
			statuses1[i] = models.TaskStatusCompleted
		} else {
			statuses1[i] = models.TaskStatusTodo
		}
	}
	epicID, feature1ID := setupProgressTest(t, 97, 1, statuses1)
	_ = featureRepo.UpdateProgress(ctx, feature1ID)

	// Feature 2: 100% with 10 tasks (E97-F02)
	statuses2 := make([]models.TaskStatus, 10)
	for i := 0; i < 10; i++ {
		statuses2[i] = models.TaskStatusCompleted
	}
	_, feature2ID := setupProgressTest(t, 97, 2, statuses2)
	_ = featureRepo.UpdateProgress(ctx, feature2ID)

	// Weighted average: (50×10 + 100×10) / (10+10) = 1500/20 = 75.0
	progress, err := epicRepo.CalculateProgress(ctx, epicID)
	if err != nil {
		t.Fatalf("Failed to calculate epic progress: %v", err)
	}

	expected := 75.0
	if progress != expected {
		t.Errorf("Expected %.1f%% epic progress (weighted average), got %.1f%%", expected, progress)
	}
}

// TestEpicProgress_TaskCountWeighting verifies small complete feature doesn't dominate large incomplete one
func TestEpicProgress_TaskCountWeighting(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	// Feature 1: 100% but only 1 task (E98-F01)
	epicID, feature1ID := setupProgressTest(t, 98, 1, []models.TaskStatus{
		models.TaskStatusCompleted,
	})
	_ = featureRepo.UpdateProgress(ctx, feature1ID)

	// Feature 2: 0% with 9 tasks (E98-F02)
	statuses2 := make([]models.TaskStatus, 9)
	for i := 0; i < 9; i++ {
		statuses2[i] = models.TaskStatusTodo
	}
	_, feature2ID := setupProgressTest(t, 98, 2, statuses2)
	_ = featureRepo.UpdateProgress(ctx, feature2ID)

	// Weighted average: (100×1 + 0×9) / (1+9) = 100/10 = 10.0
	// NOT simple average of (100+0)/2 = 50.0
	progress, err := epicRepo.CalculateProgress(ctx, epicID)
	if err != nil {
		t.Fatalf("Failed to calculate epic progress: %v", err)
	}

	expected := 10.0
	if progress != expected {
		t.Errorf("Expected %.1f%% epic progress (weighted by task count), got %.1f%%", expected, progress)
	}
}

// TestFeatureProgressByKey verifies calculating progress by feature key
func TestFeatureProgressByKey(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)

	statuses := make([]models.TaskStatus, 10)
	for i := 0; i < 10; i++ {
		if i < 5 {
			statuses[i] = models.TaskStatusCompleted
		} else {
			statuses[i] = models.TaskStatusTodo
		}
	}

	setupProgressTest(t, 80, 1, statuses)

	// Calculate progress by key
	progress, err := featureRepo.CalculateProgressByKey(ctx, "E80-F01")
	if err != nil {
		t.Fatalf("Failed to calculate progress by key: %v", err)
	}

	expected := 50.0
	if progress != expected {
		t.Errorf("Expected %.1f%% progress, got %.1f%%", expected, progress)
	}
}

// TestEpicProgressByKey verifies calculating progress by epic key
func TestEpicProgressByKey(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	statuses := make([]models.TaskStatus, 10)
	for i := 0; i < 10; i++ {
		if i < 6 {
			statuses[i] = models.TaskStatusCompleted
		} else {
			statuses[i] = models.TaskStatusTodo
		}
	}

	_, featureID := setupProgressTest(t, 81, 1, statuses)

	// Update feature progress first
	_ = featureRepo.UpdateProgress(ctx, featureID)

	// Calculate epic progress by key
	progress, err := epicRepo.CalculateProgressByKey(ctx, "E81")
	if err != nil {
		t.Fatalf("Failed to calculate epic progress by key: %v", err)
	}

	expected := 60.0
	if progress != expected {
		t.Errorf("Expected %.1f%% epic progress, got %.1f%%", expected, progress)
	}
}

// TestUpdateProgressByKey verifies updating cached progress using key
func TestUpdateProgressByKey(t *testing.T) {
	ctx := context.Background()
	db := NewDB(test.GetTestDB())
	featureRepo := NewFeatureRepository(db)

	statuses := []models.TaskStatus{
		models.TaskStatusCompleted,
		models.TaskStatusCompleted,
		models.TaskStatusTodo,
		models.TaskStatusTodo,
	}

	setupProgressTest(t, 82, 1, statuses)

	// Update progress by key
	err := featureRepo.UpdateProgressByKey(ctx, "E82-F01")
	if err != nil {
		t.Fatalf("Failed to update progress by key: %v", err)
	}

	// Retrieve feature and verify progress was updated
	feature, err := featureRepo.GetByKey(ctx, "E82-F01")
	if err != nil {
		t.Fatalf("Failed to get feature: %v", err)
	}

	expected := 50.0
	if feature.ProgressPct != expected {
		t.Errorf("Expected cached progress %.1f%%, got %.1f%%", expected, feature.ProgressPct)
	}
}
