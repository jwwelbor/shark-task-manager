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
		Status:      models.TaskStatusCompleted,
		Priority:    1,
	}
	err = taskRepo.Create(ctx, task1)
	require.NoError(t, err)

	task2 := &models.Task{
		FeatureID:   feature.ID,
		Key:         fmt.Sprintf("T-%s-002", featureKey),
		Title:       "Task 2",
		Description: stringPtr("Task 2"),
		Status:      models.TaskStatusInProgress,
		Priority:    2,
	}
	err = taskRepo.Create(ctx, task2)
	require.NoError(t, err)

	task3 := &models.Task{
		FeatureID:   feature.ID,
		Key:         fmt.Sprintf("T-%s-003", featureKey),
		Title:       "Task 3",
		Description: stringPtr("Task 3"),
		Status:      models.TaskStatusTodo,
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

func TestTaskRepository_GetStatusBreakdown(t *testing.T) {
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

	// Clean up any stale test data with this suffix
	_, _ = database.ExecContext(ctx, fmt.Sprintf("DELETE FROM tasks WHERE key LIKE 'T-%s-%%'", featureKey))
	_, _ = database.ExecContext(ctx, fmt.Sprintf("DELETE FROM features WHERE key = '%s'", featureKey))
	_, _ = database.ExecContext(ctx, fmt.Sprintf("DELETE FROM epics WHERE key = '%s'", epicKey))

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

	// Create tasks with different statuses
	tasks := []*models.Task{
		{
			FeatureID:   feature.ID,
			Key:         fmt.Sprintf("T-%s-001", featureKey),
			Title:       "Completed Task 1",
			Description: stringPtr("Completed"),
			Status:      models.TaskStatusCompleted,
			Priority:    1,
		},
		{
			FeatureID:   feature.ID,
			Key:         fmt.Sprintf("T-%s-002", featureKey),
			Title:       "Completed Task 2",
			Description: stringPtr("Completed"),
			Status:      models.TaskStatusCompleted,
			Priority:    1,
		},
		{
			FeatureID:   feature.ID,
			Key:         fmt.Sprintf("T-%s-003", featureKey),
			Title:       "In Progress Task 1",
			Description: stringPtr("In Progress"),
			Status:      models.TaskStatusInProgress,
			Priority:    2,
		},
		{
			FeatureID:   feature.ID,
			Key:         fmt.Sprintf("T-%s-004", featureKey),
			Title:       "In Progress Task 2",
			Description: stringPtr("In Progress"),
			Status:      models.TaskStatusInProgress,
			Priority:    2,
		},
		{
			FeatureID:   feature.ID,
			Key:         fmt.Sprintf("T-%s-005", featureKey),
			Title:       "Todo Task",
			Description: stringPtr("Todo"),
			Status:      models.TaskStatusTodo,
			Priority:    3,
		},
		{
			FeatureID:   feature.ID,
			Key:         fmt.Sprintf("T-%s-006", featureKey),
			Title:       "Blocked Task",
			Description: stringPtr("Blocked"),
			Status:      models.TaskStatusBlocked,
			Priority:    4,
		},
	}

	for _, task := range tasks {
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)
	}

	// Test getting status breakdown (new workflow.StatusCount format)
	t.Run("get status breakdown for feature", func(t *testing.T) {
		breakdown, err := taskRepo.GetStatusBreakdown(ctx, feature.ID)
		require.NoError(t, err)

		// Build map from StatusCount slice for easier assertions
		countMap := make(map[string]int)
		for _, sc := range breakdown {
			countMap[sc.Status] = sc.Count
		}

		// Only non-zero counts should be in the result
		assert.Equal(t, 2, countMap["completed"])
		assert.Equal(t, 2, countMap["in_progress"])
		assert.Equal(t, 1, countMap["todo"])
		assert.Equal(t, 1, countMap["blocked"])
		// Zero counts should not be present
		_, hasReadyForReview := countMap["ready_for_review"]
		assert.False(t, hasReadyForReview, "Zero count statuses should not be in result")
	})

	// Test feature with no tasks
	t.Run("get status breakdown for feature with no tasks", func(t *testing.T) {
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

		breakdown, err := taskRepo.GetStatusBreakdown(ctx, emptyFeature.ID)
		require.NoError(t, err)

		// Empty result for feature with no tasks
		assert.Empty(t, breakdown, "Feature with no tasks should have empty breakdown")
	})

	// Test GetStatusBreakdownMap for backward compatibility
	t.Run("get status breakdown map for feature (backward compat)", func(t *testing.T) {
		breakdown, err := taskRepo.GetStatusBreakdownMap(ctx, feature.ID)
		require.NoError(t, err)

		assert.Equal(t, 2, breakdown[models.TaskStatusCompleted])
		assert.Equal(t, 2, breakdown[models.TaskStatusInProgress])
		assert.Equal(t, 1, breakdown[models.TaskStatusTodo])
		assert.Equal(t, 1, breakdown[models.TaskStatusBlocked])
		assert.Equal(t, 0, breakdown[models.TaskStatusReadyForReview])
		assert.Equal(t, 0, breakdown[models.TaskStatusArchived])
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
