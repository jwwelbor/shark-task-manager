package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

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
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create unique isolated test data
	epicNum := 10 + int((time.Now().UnixNano()%80))
	epicKey := fmt.Sprintf("E%02d", epicNum)
	featureKey := fmt.Sprintf("%s-F01", epicKey)

	// Clean up
	_, _ = database.Exec("DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic and feature for test
	epic := &models.Epic{Key: epicKey, Title: "Test Epic", Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	_ = epicRepo.Create(ctx, epic)
	feature := &models.Feature{EpicID: epic.ID, Key: featureKey, Title: "Test Feature", Status: models.FeatureStatusActive}
	_ = featureRepo.Create(ctx, feature)

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

	// Cleanup
	_, _ = database.Exec("DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.Exec("DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestBulkCreateRollback tests transaction rollback on error
func TestBulkCreateRollback(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create unique isolated test data
	epicNum := 10 + int((time.Now().UnixNano()%80))
	epicKey := fmt.Sprintf("E%02d", epicNum)
	featureKey := fmt.Sprintf("%s-F01", epicKey)

	// Clean up
	_, _ = database.Exec("DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE key = ?)", featureKey)
	_, _ = database.Exec("DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic and feature for test
	epic := &models.Epic{Key: epicKey, Title: "Test Epic", Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	_ = epicRepo.Create(ctx, epic)
	feature := &models.Feature{EpicID: epic.ID, Key: featureKey, Title: "Test Feature", Status: models.FeatureStatusActive}
	_ = featureRepo.Create(ctx, feature)

	// Get initial task count (should be 0)
	initialTasks, _ := taskRepo.ListByFeature(ctx, feature.ID)
	initialCount := len(initialTasks)

	// Create tasks with duplicate key (will cause error)
	tasks := []*models.Task{
		{
			FeatureID: feature.ID,
			Key:       fmt.Sprintf("T-%s-200", featureKey),
			Title:     "First Task",
			Status:    models.TaskStatusTodo,
			Priority:  1,
		},
		{
			FeatureID: feature.ID,
			Key:       fmt.Sprintf("T-%s-200", featureKey), // Duplicate key
			Title:     "Duplicate Task",
			Status:    models.TaskStatusTodo,
			Priority:  1,
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

	// Cleanup
	_, _ = database.Exec("DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.Exec("DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestGetByKeysPartial tests bulk retrieval with some missing keys
func TestGetByKeysPartial(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create unique isolated test data
	epicNum := 10 + int((time.Now().UnixNano()%80))
	epicKey := fmt.Sprintf("E%02d", epicNum)
	featureKey := fmt.Sprintf("%s-F01", epicKey)

	// Clean up
	_, _ = database.Exec("DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE key = ?)", featureKey)
	_, _ = database.Exec("DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic and feature for test
	epic := &models.Epic{Key: epicKey, Title: "Test Epic", Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	_ = epicRepo.Create(ctx, epic)
	feature := &models.Feature{EpicID: epic.ID, Key: featureKey, Title: "Test Feature", Status: models.FeatureStatusActive}
	_ = featureRepo.Create(ctx, feature)

	// Create 2 tasks with known keys
	task1 := &models.Task{FeatureID: feature.ID, Key: fmt.Sprintf("T-%s-001", featureKey), Title: "Task 1", Status: models.TaskStatusTodo, Priority: 1}
	task2 := &models.Task{FeatureID: feature.ID, Key: fmt.Sprintf("T-%s-002", featureKey), Title: "Task 2", Status: models.TaskStatusTodo, Priority: 1}
	_ = taskRepo.Create(ctx, task1)
	_ = taskRepo.Create(ctx, task2)

	// Mix of existing and non-existing keys
	keys := []string{task1.Key, fmt.Sprintf("T-%s-999", featureKey), task2.Key}
	result, err := taskRepo.GetByKeys(ctx, keys)
	if err != nil {
		t.Fatalf("GetByKeys failed: %v", err)
	}

	// Should only return existing tasks
	if len(result) != 2 {
		t.Errorf("Expected 2 tasks (missing keys omitted), got %d", len(result))
	}

	// Verify only existing tasks are present
	if _, exists := result[task1.Key]; !exists {
		t.Errorf("Expected %s to be in result", task1.Key)
	}
	if _, exists := result[task2.Key]; !exists {
		t.Errorf("Expected %s to be in result", task2.Key)
	}
	if _, exists := result[fmt.Sprintf("T-%s-999", featureKey)]; exists {
		t.Error("Did not expect non-existent task key to be in result")
	}

	// Cleanup
	_, _ = database.Exec("DELETE FROM tasks WHERE id IN (?, ?)", task1.ID, task2.ID)
	_, _ = database.Exec("DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.Exec("DELETE FROM epics WHERE id = ?", epic.ID)
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
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create unique isolated test data
	epicNum := 10 + int((time.Now().UnixNano()%80))
	epicKey := fmt.Sprintf("E%02d", epicNum)
	featureKey := fmt.Sprintf("%s-F01", epicKey)

	// Clean up
	_, _ = database.Exec("DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE key = ?)", featureKey)
	_, _ = database.Exec("DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.Exec("DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic and feature for test
	epic := &models.Epic{Key: epicKey, Title: "Test Epic", Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	_ = epicRepo.Create(ctx, epic)
	feature := &models.Feature{EpicID: epic.ID, Key: featureKey, Title: "Test Feature", Status: models.FeatureStatusActive}
	_ = featureRepo.Create(ctx, feature)

	// Create test task with initial agent type
	initialAgentType := models.AgentTypeFrontend
	task := &models.Task{
		FeatureID:   feature.ID,
		Key:         fmt.Sprintf("T-%s-001", featureKey),
		Title:       "Original Title",
		Description: test.StringPtr("Original Description"),
		Status:      models.TaskStatusTodo,
		Priority:    5,
		AgentType:   &initialAgentType,
	}
	_ = taskRepo.Create(ctx, task)

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
	err := taskRepo.UpdateMetadata(ctx, task)
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

	// Cleanup
	_, _ = database.Exec("DELETE FROM tasks WHERE id = ?", task.ID)
	_, _ = database.Exec("DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.Exec("DELETE FROM epics WHERE id = ?", epic.ID)
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

