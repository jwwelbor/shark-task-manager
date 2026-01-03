package commands

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestConvertIdeaToEpic_Success tests successfully converting an idea to epic
func TestConvertIdeaToEpic_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	var createdEpic *models.Epic
	var markedIdeaID int64
	var markedType string
	var markedKey string

	mockIdeaRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			if key == "I-2026-01-01-01" {
				return &models.Idea{
					ID:          1,
					Key:         "I-2026-01-01-01",
					Title:       "Test Idea",
					Description: stringPtr("Test description"),
					CreatedDate: now,
					Status:      models.IdeaStatusNew,
				}, nil
			}
			return nil, fmt.Errorf("idea not found")
		},
		MarkAsConvertedFunc: func(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error {
			markedIdeaID = ideaID
			markedType = convertedToType
			markedKey = convertedToKey
			return nil
		},
	}

	mockEpicRepo := &MockEpicRepository{
		CreateFunc: func(ctx context.Context, epic *models.Epic) error {
			createdEpic = epic
			epic.ID = 1
			epic.Key = "E15"
			return nil
		},
	}

	// Call conversion function (will be implemented)
	newKey, err := convertIdeaToEpic(ctx, mockIdeaRepo, mockEpicRepo, "I-2026-01-01-01")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify epic was created
	if createdEpic == nil {
		t.Fatal("Epic was not created")
	}
	if createdEpic.Title != "Test Idea" {
		t.Errorf("Expected epic title 'Test Idea', got %s", createdEpic.Title)
	}
	if createdEpic.Description == nil || *createdEpic.Description != "Test description" {
		t.Error("Epic description was not copied from idea")
	}

	// Verify idea was marked as converted
	if markedIdeaID != 1 {
		t.Errorf("Expected idea ID 1 to be marked, got %d", markedIdeaID)
	}
	if markedType != "epic" {
		t.Errorf("Expected type 'epic', got %s", markedType)
	}
	if markedKey != "E15" {
		t.Errorf("Expected key 'E15', got %s", markedKey)
	}

	// Verify return value
	if newKey != "E15" {
		t.Errorf("Expected return key 'E15', got %s", newKey)
	}
	if markedKey != "E15" {
		t.Errorf("Unused variable check: markedKey should be E15, got %s", markedKey)
	}
}

// TestConvertIdeaToEpic_AlreadyConverted tests error when idea already converted
func TestConvertIdeaToEpic_AlreadyConverted(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	convertedToType := "feature"
	convertedToKey := "E10-F05"

	mockIdeaRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{
				ID:              1,
				Key:             "I-2026-01-01-01",
				Title:           "Already Converted",
				CreatedDate:     now,
				Status:          models.IdeaStatusConverted,
				ConvertedToType: &convertedToType,
				ConvertedToKey:  &convertedToKey,
			}, nil
		},
	}

	mockEpicRepo := &MockEpicRepository{}

	_, err := convertIdeaToEpic(ctx, mockIdeaRepo, mockEpicRepo, "I-2026-01-01-01")

	if err == nil {
		t.Fatal("Expected error for already-converted idea, got none")
	}
}

// TestConvertIdeaToEpic_IdeaNotFound tests error when idea doesn't exist
func TestConvertIdeaToEpic_IdeaNotFound(t *testing.T) {
	ctx := context.Background()

	mockIdeaRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return nil, fmt.Errorf("idea not found with key %q", key)
		},
	}

	mockEpicRepo := &MockEpicRepository{}

	_, err := convertIdeaToEpic(ctx, mockIdeaRepo, mockEpicRepo, "I-9999-99-99-99")

	if err == nil {
		t.Fatal("Expected error for non-existent idea, got none")
	}
}

// TestConvertIdeaToFeature_Success tests successfully converting an idea to feature
func TestConvertIdeaToFeature_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	var createdFeature *models.Feature
	var markedIdeaID int64
	var markedType string
	var markedKey string

	mockIdeaRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{
				ID:          1,
				Key:         "I-2026-01-01-01",
				Title:       "Test Feature Idea",
				Description: stringPtr("Feature description"),
				CreatedDate: now,
				Status:      models.IdeaStatusNew,
			}, nil
		},
		MarkAsConvertedFunc: func(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error {
			markedIdeaID = ideaID
			markedType = convertedToType
			markedKey = convertedToKey
			return nil
		},
	}

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			if key == "E10" {
				return &models.Epic{
					ID:    10,
					Key:   "E10",
					Title: "Existing Epic",
				}, nil
			}
			return nil, fmt.Errorf("epic not found")
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		CreateFunc: func(ctx context.Context, feature *models.Feature) error {
			createdFeature = feature
			feature.ID = 1
			feature.Key = "E10-F03"
			return nil
		},
	}

	newKey, err := convertIdeaToFeature(ctx, mockIdeaRepo, mockEpicRepo, mockFeatureRepo, "I-2026-01-01-01", "E10")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify feature was created
	if createdFeature == nil {
		t.Fatal("Feature was not created")
	}
	if createdFeature.Title != "Test Feature Idea" {
		t.Errorf("Expected feature title 'Test Feature Idea', got %s", createdFeature.Title)
	}
	if createdFeature.EpicID != 10 {
		t.Errorf("Expected epic ID 10, got %d", createdFeature.EpicID)
	}

	// Verify idea was marked as converted
	if markedIdeaID != 1 {
		t.Errorf("Expected idea ID 1 to be marked, got %d", markedIdeaID)
	}
	if markedType != "feature" {
		t.Errorf("Expected type 'feature', got %s", markedType)
	}

	// Verify return value
	if newKey != "E10-F03" {
		t.Errorf("Expected return key 'E10-F03', got %s", newKey)
	}
	if markedKey != "E10-F03" {
		t.Errorf("Unused variable check: markedKey should be E10-F03, got %s", markedKey)
	}
}

// TestConvertIdeaToFeature_EpicNotFound tests error when epic doesn't exist
func TestConvertIdeaToFeature_EpicNotFound(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	mockIdeaRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{
				ID:          1,
				Key:         "I-2026-01-01-01",
				Title:       "Test Idea",
				CreatedDate: now,
				Status:      models.IdeaStatusNew,
			}, nil
		},
	}

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("epic not found with key %q", key)
		},
	}

	mockFeatureRepo := &MockFeatureRepository{}

	_, err := convertIdeaToFeature(ctx, mockIdeaRepo, mockEpicRepo, mockFeatureRepo, "I-2026-01-01-01", "E99")

	if err == nil {
		t.Fatal("Expected error for non-existent epic, got none")
	}
}

// TestConvertIdeaToTask_Success tests successfully converting an idea to task
func TestConvertIdeaToTask_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	priority := 8

	var createdTask *models.Task
	var markedIdeaID int64
	var markedType string
	var markedKey string

	mockIdeaRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{
				ID:          1,
				Key:         "I-2026-01-01-01",
				Title:       "Test Task Idea",
				Description: stringPtr("Task description"),
				Priority:    &priority,
				CreatedDate: now,
				Status:      models.IdeaStatusNew,
			}, nil
		},
		MarkAsConvertedFunc: func(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error {
			markedIdeaID = ideaID
			markedType = convertedToType
			markedKey = convertedToKey
			return nil
		},
	}

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 10, Key: "E10", Title: "Epic"}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				ID:     5,
				EpicID: 10,
				Key:    "E10-F02",
				Title:  "Feature",
			}, nil
		},
	}

	mockTaskRepo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			createdTask = task
			task.ID = 1
			task.Key = "T-E10-F02-005"
			return nil
		},
	}

	newKey, err := convertIdeaToTask(ctx, mockIdeaRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo, "I-2026-01-01-01", "E10", "E10-F02")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify task was created
	if createdTask == nil {
		t.Fatal("Task was not created")
	}
	if createdTask.Title != "Test Task Idea" {
		t.Errorf("Expected task title 'Test Task Idea', got %s", createdTask.Title)
	}
	if createdTask.Priority != 8 {
		t.Errorf("Expected priority 8, got %d", createdTask.Priority)
	}

	// Verify idea was marked as converted
	if markedIdeaID != 1 {
		t.Errorf("Expected idea ID 1 to be marked, got %d", markedIdeaID)
	}
	if markedType != "task" {
		t.Errorf("Expected type 'task', got %s", markedType)
	}

	// Verify return value
	if newKey != "T-E10-F02-005" {
		t.Errorf("Expected return key 'T-E10-F02-005', got %s", newKey)
	}
	if markedKey != "T-E10-F02-005" {
		t.Errorf("Unused variable check: markedKey should be T-E10-F02-005, got %s", markedKey)
	}
}

// TestConvertIdeaToTask_FeatureNotInEpic tests error when feature doesn't belong to epic
func TestConvertIdeaToTask_FeatureNotInEpic(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	mockIdeaRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{
				ID:          1,
				Key:         "I-2026-01-01-01",
				Title:       "Test Idea",
				CreatedDate: now,
				Status:      models.IdeaStatusNew,
			}, nil
		},
	}

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 10, Key: "E10"}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			// Feature belongs to different epic (ID 5 instead of 10)
			return &models.Feature{
				ID:     1,
				EpicID: 5,
				Key:    "E05-F01",
			}, nil
		},
	}

	mockTaskRepo := &MockTaskRepository{}

	_, err := convertIdeaToTask(ctx, mockIdeaRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo, "I-2026-01-01-01", "E10", "E05-F01")

	if err == nil {
		t.Fatal("Expected error for feature not in epic, got none")
	}
}
