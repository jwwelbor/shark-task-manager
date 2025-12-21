package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestBulkCreate tests creating multiple tasks in a single transaction
func TestBulkCreate(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()

	// Get a feature to associate tasks with
	featureRepo := NewFeatureRepository(db)
	feature, err := featureRepo.GetByKey(ctx, "E04-F05")
	if err != nil {
		t.Fatalf("Failed to get feature: %v", err)
	}

	// Delete any existing test tasks first (for test isolation)
	database.Exec("DELETE FROM tasks WHERE key IN ('T-E04-F05-100', 'T-E04-F05-101', 'T-E04-F05-102')")

	// Create tasks for bulk insert
	tasks := []*models.Task{
		{
			FeatureID:   feature.ID,
			Key:         "T-E04-F05-100",
			Title:       "Bulk Test Task 1",
			Description: test.StringPtr("First bulk task"),
			Status:      models.TaskStatusTodo,
			Priority:    1,
			FilePath:    test.StringPtr("/path/to/task1.md"),
		},
		{
			FeatureID:   feature.ID,
			Key:         "T-E04-F05-101",
			Title:       "Bulk Test Task 2",
			Description: test.StringPtr("Second bulk task"),
			Status:      models.TaskStatusTodo,
			Priority:    2,
			FilePath:    test.StringPtr("/path/to/task2.md"),
		},
		{
			FeatureID:   feature.ID,
			Key:         "T-E04-F05-102",
			Title:       "Bulk Test Task 3",
			Description: test.StringPtr("Third bulk task"),
			Status:      models.TaskStatusTodo,
			Priority:    3,
			FilePath:    test.StringPtr("/path/to/task3.md"),
		},
	}

	// Test bulk create
	count, err := taskRepo.BulkCreate(ctx, tasks)
	if err != nil {
		t.Fatalf("BulkCreate failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected to create 3 tasks, created %d", count)
	}

	// Verify all tasks have IDs assigned
	for i, task := range tasks {
		if task.ID == 0 {
			t.Errorf("Task %d has no ID assigned", i)
		}
	}

	// Verify tasks are in database
	for _, task := range tasks {
		retrieved, err := taskRepo.GetByKey(ctx, task.Key)
		if err != nil {
			t.Errorf("Failed to retrieve task %s: %v", task.Key, err)
		}
		if retrieved.Title != task.Title {
			t.Errorf("Title mismatch for %s: expected %s, got %s", task.Key, task.Title, retrieved.Title)
		}
	}
}

// TestBulkCreateEmpty tests bulk create with empty slice
func TestBulkCreateEmpty(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	count, err := taskRepo.BulkCreate(ctx, []*models.Task{})
	if err != nil {
		t.Errorf("BulkCreate with empty slice should not error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

// TestBulkCreateValidationFailure tests that validation errors are caught
func TestBulkCreateValidationFailure(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	featureRepo := NewFeatureRepository(db)
	feature, _ := featureRepo.GetByKey(ctx, "E04-F05")

	// Create task with invalid key (empty)
	tasks := []*models.Task{
		{
			FeatureID: feature.ID,
			Key:       "", // Invalid: empty key
			Title:     "Invalid Task",
			Status:    models.TaskStatusTodo,
		},
	}

	_, err := taskRepo.BulkCreate(ctx, tasks)
	if err == nil {
		t.Error("Expected validation error, got nil")
	}
}

// TestBulkCreateRollback tests transaction rollback on error
func TestBulkCreateRollback(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	featureRepo := NewFeatureRepository(db)
	feature, _ := featureRepo.GetByKey(ctx, "E04-F05")

	// Get initial task count
	initialTasks, _ := taskRepo.ListByFeature(ctx, feature.ID)
	initialCount := len(initialTasks)

	// Create tasks with duplicate key (will cause error)
	tasks := []*models.Task{
		{
			FeatureID: feature.ID,
			Key:       "T-E04-F05-200",
			Title:     "First Task",
			Status:    models.TaskStatusTodo,
		},
		{
			FeatureID: feature.ID,
			Key:       "T-E04-F05-200", // Duplicate key
			Title:     "Duplicate Task",
			Status:    models.TaskStatusTodo,
		},
	}

	_, err := taskRepo.BulkCreate(ctx, tasks)
	if err == nil {
		t.Error("Expected error due to duplicate key")
	}

	// Verify no tasks were created (rollback worked)
	finalTasks, _ := taskRepo.ListByFeature(ctx, feature.ID)
	finalCount := len(finalTasks)

	if finalCount != initialCount {
		t.Errorf("Expected task count to remain %d after rollback, got %d", initialCount, finalCount)
	}
}

// TestGetByKeys tests bulk retrieval of tasks
func TestGetByKeys(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()

	// Test retrieving existing tasks
	keys := []string{"T-E99-F99-001", "T-E99-F99-002", "T-E99-F99-003"}
	result, err := taskRepo.GetByKeys(ctx, keys)
	if err != nil {
		t.Fatalf("GetByKeys failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(result))
	}

	// Verify each task is in the result
	for _, key := range keys {
		task, exists := result[key]
		if !exists {
			t.Errorf("Task %s not found in result", key)
		}
		if task.Key != key {
			t.Errorf("Key mismatch: expected %s, got %s", key, task.Key)
		}
	}
}

// TestGetByKeysPartial tests bulk retrieval with some missing keys
func TestGetByKeysPartial(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()

	// Mix of existing and non-existing keys
	keys := []string{"T-E99-F99-001", "T-E99-F99-999", "T-E99-F99-002"}
	result, err := taskRepo.GetByKeys(ctx, keys)
	if err != nil {
		t.Fatalf("GetByKeys failed: %v", err)
	}

	// Should only return existing tasks
	if len(result) != 2 {
		t.Errorf("Expected 2 tasks (missing keys omitted), got %d", len(result))
	}

	// Verify only existing tasks are present
	if _, exists := result["T-E99-F99-001"]; !exists {
		t.Error("Expected T-E99-F99-001 to be in result")
	}
	if _, exists := result["T-E99-F99-002"]; !exists {
		t.Error("Expected T-E99-F99-002 to be in result")
	}
	if _, exists := result["T-E99-F99-999"]; exists {
		t.Error("Did not expect T-E99-F99-999 to be in result")
	}
}

// TestGetByKeysEmpty tests bulk retrieval with empty keys list
func TestGetByKeysEmpty(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	result, err := taskRepo.GetByKeys(ctx, []string{})
	if err != nil {
		t.Errorf("GetByKeys with empty slice should not error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map, got %d items", len(result))
	}
}

// TestUpdateMetadata tests updating only metadata fields
func TestUpdateMetadata(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()

	// Get existing task
	task, err := taskRepo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	// Store original values of database-only fields
	originalStatus := task.Status
	originalPriority := task.Priority
	originalAgentType := task.AgentType

	// Update metadata fields
	task.Title = "Updated Title"
	task.Description = test.StringPtr("Updated Description")
	task.FilePath = test.StringPtr("/new/path/to/task.md")

	// Also try to change database-only fields (should be ignored)
	task.Status = models.TaskStatusCompleted
	task.Priority = 10 // Max valid priority
	newAgentType := models.AgentTypeBackend
	task.AgentType = &newAgentType

	// Update metadata
	err = taskRepo.UpdateMetadata(ctx, task)
	if err != nil {
		t.Fatalf("UpdateMetadata failed: %v", err)
	}

	// Retrieve and verify
	updated, err := taskRepo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated task: %v", err)
	}

	// Verify metadata fields were updated
	if updated.Title != "Updated Title" {
		t.Errorf("Title not updated: expected 'Updated Title', got '%s'", updated.Title)
	}
	if updated.Description == nil || *updated.Description != "Updated Description" {
		t.Error("Description not updated")
	}
	if updated.FilePath == nil || *updated.FilePath != "/new/path/to/task.md" {
		t.Error("FilePath not updated")
	}

	// Verify database-only fields were NOT updated
	if updated.Status != originalStatus {
		t.Errorf("Status should not change: expected %s, got %s", originalStatus, updated.Status)
	}
	if updated.Priority != originalPriority {
		t.Errorf("Priority should not change: expected %d, got %d", originalPriority, updated.Priority)
	}
	if originalAgentType != nil {
		if updated.AgentType == nil || *updated.AgentType != *originalAgentType {
			t.Error("AgentType should not change")
		}
	}
}

// TestUpdateMetadataNotFound tests updating non-existent task
func TestUpdateMetadataNotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	task := &models.Task{
		ID:       99999, // Non-existent ID
		Key:      "T-E99-F99-999",
		Title:    "Fake Task",
		Status:   models.TaskStatusTodo,
		Priority: 1,
	}

	err := taskRepo.UpdateMetadata(ctx, task)
	if err == nil {
		t.Error("Expected error when updating non-existent task")
	}
}

// TestEpicCreateIfNotExists tests idempotent epic creation
func TestEpicCreateIfNotExists(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)

	test.SeedTestData()

	// Test creating new epic
	businessValue := models.PriorityHigh
	newEpic := &models.Epic{
		Key:           "E98",
		Title:         "Test Epic",
		Description:   test.StringPtr("Test Description"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &businessValue,
	}

	// First call - may or may not be created depending on previous test runs
	epic1, created1, err := epicRepo.CreateIfNotExists(ctx, newEpic)
	if err != nil {
		t.Fatalf("CreateIfNotExists failed: %v", err)
	}
	if epic1.ID == 0 {
		t.Error("Expected epic to have ID")
	}

	// Second call - should always return existing epic (not created)
	epic2, created2, err := epicRepo.CreateIfNotExists(ctx, newEpic)
	if err != nil {
		t.Fatalf("CreateIfNotExists failed on second call: %v", err)
	}
	if created2 {
		t.Error("Expected epic to not be created on second call (already exists)")
	}
	if epic2.ID != epic1.ID {
		t.Errorf("Expected same epic ID: got %d and %d", epic1.ID, epic2.ID)
	}

	// Verify idempotency
	if created1 {
		t.Logf("Epic was created on first call")
	} else {
		t.Logf("Epic already existed on first call")
	}
}

// TestFeatureCreateIfNotExists tests idempotent feature creation
func TestFeatureCreateIfNotExists(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	test.SeedTestData()

	// Get existing epic
	epic, _ := epicRepo.GetByKey(ctx, "E04")

	// Test creating new feature
	newFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E04-F99",
		Title:       "Test Feature",
		Description: test.StringPtr("Test Description"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}

	// First call - may or may not be created depending on previous test runs
	feature1, created1, err := featureRepo.CreateIfNotExists(ctx, newFeature)
	if err != nil {
		t.Fatalf("CreateIfNotExists failed: %v", err)
	}
	if feature1.ID == 0 {
		t.Error("Expected feature to have ID")
	}

	// Second call - should always return existing feature (not created)
	feature2, created2, err := featureRepo.CreateIfNotExists(ctx, newFeature)
	if err != nil {
		t.Fatalf("CreateIfNotExists failed on second call: %v", err)
	}
	if created2 {
		t.Error("Expected feature to not be created on second call (already exists)")
	}
	if feature2.ID != feature1.ID {
		t.Errorf("Expected same feature ID: got %d and %d", feature1.ID, feature2.ID)
	}

	// Verify idempotency
	if created1 {
		t.Logf("Feature was created on first call")
	} else {
		t.Logf("Feature already existed on first call")
	}
}

// TestBulkCreatePerformance benchmarks bulk create performance
func TestBulkCreatePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	featureRepo := NewFeatureRepository(db)
	feature, _ := featureRepo.GetByKey(ctx, "E04-F05")

	// Clean up any existing performance test tasks
	database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E04-F05-2%'")

	// Create 100 tasks
	tasks := make([]*models.Task, 100)
	for i := 0; i < 100; i++ {
		tasks[i] = &models.Task{
			FeatureID:   feature.ID,
			Key:         test.GenerateUniqueKey("E04-F05", i+200),
			Title:       "Performance Test Task",
			Description: test.StringPtr("Performance testing"),
			Status:      models.TaskStatusTodo,
			Priority:    (i % 10) + 1, // Priority 1-10
		}
	}

	start := time.Now()
	count, err := taskRepo.BulkCreate(ctx, tasks)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("BulkCreate failed: %v", err)
	}
	if count != 100 {
		t.Errorf("Expected to create 100 tasks, created %d", count)
	}

	// PRD requirement: 100 tasks in <1 second
	if duration > time.Second {
		t.Errorf("BulkCreate took %v, expected <1s", duration)
	} else {
		t.Logf("BulkCreate of 100 tasks completed in %v", duration)
	}
}

// TestGetByKeysPerformance benchmarks bulk retrieval performance
func TestGetByKeysPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	featureRepo := NewFeatureRepository(db)
	feature, _ := featureRepo.GetByKey(ctx, "E04-F05")

	// First create 100 tasks
	tasks := make([]*models.Task, 100)
	for i := 0; i < 100; i++ {
		tasks[i] = &models.Task{
			FeatureID: feature.ID,
			Key:       test.GenerateUniqueKey("E04-F05", i+300),
			Title:     "Lookup Test Task",
			Status:    models.TaskStatusTodo,
			Priority:  5, // Valid priority
		}
	}
	taskRepo.BulkCreate(ctx, tasks)

	// Build keys list
	keys := make([]string, 100)
	for i, task := range tasks {
		keys[i] = task.Key
	}

	// Test bulk retrieval
	start := time.Now()
	result, err := taskRepo.GetByKeys(ctx, keys)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GetByKeys failed: %v", err)
	}
	if len(result) != 100 {
		t.Errorf("Expected 100 tasks, got %d", len(result))
	}

	// PRD requirement: 100 lookups in <100ms
	if duration > 100*time.Millisecond {
		t.Errorf("GetByKeys took %v, expected <100ms", duration)
	} else {
		t.Logf("GetByKeys of 100 tasks completed in %v", duration)
	}
}
