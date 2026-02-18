package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureRepository_ListByStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create test epic with unique key
	// Use nanosecond timestamp modulo 1000 for better uniqueness
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano())%1000/10)

	// Clean up any existing data from previous test runs
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE ?", fmt.Sprintf("E%s-F%%", suffix))
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", fmt.Sprintf("E%s", suffix))

	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           fmt.Sprintf("E%s", suffix),
		Title:         "Test Epic",
		Description:   stringPtr("Test Description"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create features with different statuses
	activeFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("E%s-F01", suffix),
		Title:       "Active Feature",
		Description: stringPtr("Active"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
	}
	err = featureRepo.Create(ctx, activeFeature)
	require.NoError(t, err)

	completedFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("E%s-F02", suffix),
		Title:       "Completed Feature",
		Description: stringPtr("Completed"),
		Status:      models.FeatureStatusCompleted,
		ProgressPct: 100.0,
	}
	err = featureRepo.Create(ctx, completedFeature)
	require.NoError(t, err)

	draftFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("E%s-F03", suffix),
		Title:       "Draft Feature",
		Description: stringPtr("Draft"),
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
	}
	err = featureRepo.Create(ctx, draftFeature)
	require.NoError(t, err)

	// Test filtering by active status
	t.Run("filter by active status", func(t *testing.T) {
		features, err := featureRepo.ListByStatus(ctx, models.FeatureStatusActive)
		require.NoError(t, err)

		// Find our test feature in results
		found := false
		for _, f := range features {
			if f.Key == activeFeature.Key {
				found = true
				assert.Equal(t, models.FeatureStatusActive, f.Status)
				assert.Equal(t, "Active Feature", f.Title)
			}
		}
		assert.True(t, found, "Should find active feature in results")
	})

	// Test filtering by completed status
	t.Run("filter by completed status", func(t *testing.T) {
		features, err := featureRepo.ListByStatus(ctx, models.FeatureStatusCompleted)
		require.NoError(t, err)

		// Find our test feature in results
		found := false
		for _, f := range features {
			if f.Key == completedFeature.Key {
				found = true
				assert.Equal(t, models.FeatureStatusCompleted, f.Status)
				assert.Equal(t, "Completed Feature", f.Title)
			}
		}
		assert.True(t, found, "Should find completed feature in results")
	})

	// Test filtering by draft status
	t.Run("filter by draft status", func(t *testing.T) {
		features, err := featureRepo.ListByStatus(ctx, models.FeatureStatusDraft)
		require.NoError(t, err)

		// Find our test feature in results
		found := false
		for _, f := range features {
			if f.Key == draftFeature.Key {
				found = true
				assert.Equal(t, models.FeatureStatusDraft, f.Status)
				assert.Equal(t, "Draft Feature", f.Title)
			}
		}
		assert.True(t, found, "Should find draft feature in results")
	})
}

func TestFeatureRepository_ListByEpicAndStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create test epic with unique key
	// Use nanosecond timestamp modulo 1000 for better uniqueness
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano())%1000/10)

	// Clean up any existing data from previous test runs
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE ?", fmt.Sprintf("E%s-F%%", suffix))
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", fmt.Sprintf("E%s", suffix))

	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           fmt.Sprintf("E%s", suffix),
		Title:         "Test Epic",
		Description:   stringPtr("Test Description"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create features with different statuses
	activeFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("E%s-F01", suffix),
		Title:       "Active Feature",
		Description: stringPtr("Active"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
	}
	err = featureRepo.Create(ctx, activeFeature)
	require.NoError(t, err)

	completedFeature := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("E%s-F02", suffix),
		Title:       "Completed Feature",
		Description: stringPtr("Completed"),
		Status:      models.FeatureStatusCompleted,
		ProgressPct: 100.0,
	}
	err = featureRepo.Create(ctx, completedFeature)
	require.NoError(t, err)

	// Test filtering by epic and status
	t.Run("filter by epic and active status", func(t *testing.T) {
		features, err := featureRepo.ListByEpicAndStatus(ctx, epic.ID, models.FeatureStatusActive)
		require.NoError(t, err)

		// Find our test feature in results
		found := false
		for _, f := range features {
			if f.Key == activeFeature.Key {
				found = true
				assert.Equal(t, epic.ID, f.EpicID)
				assert.Equal(t, models.FeatureStatusActive, f.Status)
			}
		}
		assert.True(t, found, "Should find active feature in epic")
	})
}

func TestFeatureRepository_GetTaskCount(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	// Create test epic with unique key
	// Epic keys must be E\d{2} format, so use E10-E99 range with timestamp
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)
	featureKey := fmt.Sprintf("%s-F01", epicKey)

	// Clean up any existing data from previous test runs
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE ?", fmt.Sprintf("T-%s-%%", featureKey))
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Test Epic",
		Description:   stringPtr("Test Description"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create feature
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         featureKey,
		Title:       "Test Feature",
		Description: stringPtr("Test"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 0.0,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Create tasks
	task1 := &models.Task{
		FeatureID:   feature.ID,
		Key:         fmt.Sprintf("T-%s-001", featureKey),
		Title:       "Task 1",
		Description: stringPtr("Task 1"),
		Status:      models.TaskStatus("completed"),
		Priority:    1,
	}
	err = taskRepo.Create(ctx, task1)
	require.NoError(t, err)

	task2 := &models.Task{
		FeatureID:   feature.ID,
		Key:         fmt.Sprintf("T-%s-002", featureKey),
		Title:       "Task 2",
		Description: stringPtr("Task 2"),
		Status:      models.TaskStatus("in_progress"),
		Priority:    2,
	}
	err = taskRepo.Create(ctx, task2)
	require.NoError(t, err)

	task3 := &models.Task{
		FeatureID:   feature.ID,
		Key:         fmt.Sprintf("T-%s-003", featureKey),
		Title:       "Task 3",
		Description: stringPtr("Task 3"),
		Status:      models.TaskStatus("todo"),
		Priority:    3,
	}
	err = taskRepo.Create(ctx, task3)
	require.NoError(t, err)

	// Test getting task count
	t.Run("get task count for feature", func(t *testing.T) {
		count, err := featureRepo.GetTaskCount(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	// Test feature with no tasks
	t.Run("get task count for feature with no tasks", func(t *testing.T) {
		emptyFeature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-F02", epicKey),
			Title:       "Empty Feature",
			Description: stringPtr("Empty"),
			Status:      models.FeatureStatusActive,
			ProgressPct: 0.0,
		}
		err = featureRepo.Create(ctx, emptyFeature)
		require.NoError(t, err)

		count, err := featureRepo.GetTaskCount(ctx, emptyFeature.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
