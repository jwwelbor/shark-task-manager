package status

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *repository.DB {
	sqlDB, err := db.InitDB(":memory:")
	require.NoError(t, err)
	return &repository.DB{DB: sqlDB}
}

func createTestWorkflowConfig() *config.WorkflowConfig {
	return &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"draft": {
				Phase:          "planning",
				ProgressWeight: 0.0,
			},
			"todo": {
				Phase:          "planning",
				ProgressWeight: 0.0,
			},
			"in_progress": {
				Phase:          "development",
				ProgressWeight: 0.5,
			},
			"ready_for_review": {
				Phase:          "review",
				ProgressWeight: 0.75,
			},
			"completed": {
				Phase:          "done",
				ProgressWeight: 1.0,
			},
			"blocked": {
				Phase:          "any",
				ProgressWeight: 0.0,
				BlocksFeature:  true,
			},
		},
	}
}

func TestCalculationService_RecalculateFeatureStatus(t *testing.T) {
	ctx := context.Background()
	testDB := setupTestDB(t)
	defer testDB.Close()

	featureRepo := repository.NewFeatureRepository(testDB)
	epicRepo := repository.NewEpicRepository(testDB)
	taskRepo := repository.NewTaskRepository(testDB)
	cfg := createTestWorkflowConfig()
	calcService := NewCalculationService(testDB, cfg)

	// Create test epic
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E01-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	t.Run("empty_feature_stays_draft", func(t *testing.T) {
		result, err := calcService.RecalculateFeatureStatus(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, "draft", result.NewStatus)
		assert.False(t, result.WasChanged) // Already draft
	})

	// Track task IDs for later updates
	var taskIDs []int64

	t.Run("all_todo_stays_draft", func(t *testing.T) {
		// Add todo tasks
		for i := 1; i <= 3; i++ {
			task := &models.Task{
				FeatureID: feature.ID,
				Key:       "T-E01-F01-00" + string(rune('0'+i)),
				Title:     "Todo Task",
				Status:    models.TaskStatusTodo,
				Priority:  5,
			}
			err := taskRepo.Create(ctx, task)
			require.NoError(t, err)
			taskIDs = append(taskIDs, task.ID)
		}

		result, err := calcService.RecalculateFeatureStatus(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, "draft", result.NewStatus)
	})

	t.Run("in_progress_activates_feature", func(t *testing.T) {
		// Update a task to in_progress
		err := taskRepo.UpdateStatusForced(ctx, taskIDs[0], models.TaskStatusInProgress, nil, nil, nil, true)
		require.NoError(t, err)

		result, err := calcService.RecalculateFeatureStatus(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, "active", result.NewStatus)
		assert.True(t, result.WasChanged)
	})

	t.Run("all_completed_completes_feature", func(t *testing.T) {
		// Complete all tasks
		for _, id := range taskIDs {
			err := taskRepo.UpdateStatusForced(ctx, id, models.TaskStatusCompleted, nil, nil, nil, true)
			require.NoError(t, err)
		}

		result, err := calcService.RecalculateFeatureStatus(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.NewStatus)
		assert.True(t, result.WasChanged)
	})

	t.Run("override_prevents_update", func(t *testing.T) {
		// Set override
		err := featureRepo.SetStatusOverride(ctx, feature.ID, true)
		require.NoError(t, err)

		// Reopen a task
		err = taskRepo.UpdateStatusForced(ctx, taskIDs[0], models.TaskStatusTodo, nil, nil, nil, true)
		require.NoError(t, err)

		result, err := calcService.RecalculateFeatureStatus(ctx, feature.ID)
		require.NoError(t, err)
		assert.True(t, result.WasSkipped)
		assert.Equal(t, "status_override enabled", result.SkipReason)

		// Verify status didn't change
		updatedFeature, err := featureRepo.GetByID(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FeatureStatusCompleted, updatedFeature.Status)
	})
}

func TestCalculationService_RecalculateEpicStatus(t *testing.T) {
	ctx := context.Background()
	testDB := setupTestDB(t)
	defer testDB.Close()

	featureRepo := repository.NewFeatureRepository(testDB)
	epicRepo := repository.NewEpicRepository(testDB)
	cfg := createTestWorkflowConfig()
	calcService := NewCalculationService(testDB, cfg)

	// Create test epic
	epic := &models.Epic{
		Key:      "E02",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	t.Run("empty_epic_stays_draft", func(t *testing.T) {
		result, err := calcService.RecalculateEpicStatus(ctx, epic.ID)
		require.NoError(t, err)
		assert.Equal(t, "draft", result.NewStatus)
	})

	t.Run("all_draft_features_keeps_draft", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			feature := &models.Feature{
				EpicID: epic.ID,
				Key:    "E02-F0" + string(rune('0'+i)),
				Title:  "Draft Feature",
				Status: models.FeatureStatusDraft,
			}
			err := featureRepo.Create(ctx, feature)
			require.NoError(t, err)
		}

		result, err := calcService.RecalculateEpicStatus(ctx, epic.ID)
		require.NoError(t, err)
		assert.Equal(t, "draft", result.NewStatus)
	})

	t.Run("active_feature_activates_epic", func(t *testing.T) {
		// Update feature to active
		feature, err := featureRepo.GetByKey(ctx, "E02-F01")
		require.NoError(t, err)
		feature.Status = models.FeatureStatusActive
		err = featureRepo.Update(ctx, feature)
		require.NoError(t, err)

		result, err := calcService.RecalculateEpicStatus(ctx, epic.ID)
		require.NoError(t, err)
		assert.Equal(t, "active", result.NewStatus)
		assert.True(t, result.WasChanged)
	})

	t.Run("all_completed_features_completes_epic", func(t *testing.T) {
		features := []string{"E02-F01", "E02-F02", "E02-F03"}
		for _, key := range features {
			feature, err := featureRepo.GetByKey(ctx, key)
			require.NoError(t, err)
			feature.Status = models.FeatureStatusCompleted
			err = featureRepo.Update(ctx, feature)
			require.NoError(t, err)
		}

		result, err := calcService.RecalculateEpicStatus(ctx, epic.ID)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.NewStatus)
		assert.True(t, result.WasChanged)
	})
}

func TestCalculationService_CascadeFromTask(t *testing.T) {
	ctx := context.Background()
	testDB := setupTestDB(t)
	defer testDB.Close()

	featureRepo := repository.NewFeatureRepository(testDB)
	epicRepo := repository.NewEpicRepository(testDB)
	taskRepo := repository.NewTaskRepository(testDB)
	cfg := createTestWorkflowConfig()
	calcService := NewCalculationService(testDB, cfg)

	// Create test epic
	epic := &models.Epic{
		Key:      "E03",
		Title:    "Cascade Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E03-F01",
		Title:  "Cascade Test Feature",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Create test task
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E03-F01-001",
		Title:     "Cascade Test Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)

	t.Run("cascade_updates_both_feature_and_epic", func(t *testing.T) {
		// Start the task (make it in_progress)
		err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatusInProgress, nil, nil, nil, true)
		require.NoError(t, err)

		results, err := calcService.CascadeFromTask(ctx, "T-E03-F01-001")
		require.NoError(t, err)
		require.Len(t, results, 2)

		// Check feature update
		assert.Equal(t, "feature", results[0].EntityType)
		assert.Equal(t, "active", results[0].NewStatus)

		// Check epic update
		assert.Equal(t, "epic", results[1].EntityType)
		assert.Equal(t, "active", results[1].NewStatus)

		// Verify in database
		updatedFeature, err := featureRepo.GetByID(ctx, feature.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FeatureStatusActive, updatedFeature.Status)

		updatedEpic, err := epicRepo.GetByID(ctx, epic.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EpicStatusActive, updatedEpic.Status)
	})

	t.Run("complete_task_completes_feature_and_epic", func(t *testing.T) {
		// Complete the only task
		err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, nil, nil, nil, true)
		require.NoError(t, err)

		results, err := calcService.CascadeFromTask(ctx, "T-E03-F01-001")
		require.NoError(t, err)

		// Feature should be completed
		assert.Equal(t, "completed", results[0].NewStatus)
		// Epic should be completed too
		assert.Equal(t, "completed", results[1].NewStatus)
	})
}

func TestCalculationService_RecalculateAll(t *testing.T) {
	ctx := context.Background()
	testDB := setupTestDB(t)
	defer testDB.Close()

	featureRepo := repository.NewFeatureRepository(testDB)
	epicRepo := repository.NewEpicRepository(testDB)
	taskRepo := repository.NewTaskRepository(testDB)
	cfg := createTestWorkflowConfig()
	calcService := NewCalculationService(testDB, cfg)

	// Track first task ID for status update
	var firstTaskID int64

	// Create test epics and features
	for i := 1; i <= 2; i++ {
		epic := &models.Epic{
			Key:      "E0" + string(rune('0'+i)),
			Title:    "Test Epic " + string(rune('0'+i)),
			Status:   models.EpicStatusDraft,
			Priority: models.PriorityMedium,
		}
		err := epicRepo.Create(ctx, epic)
		require.NoError(t, err)

		for j := 1; j <= 2; j++ {
			feature := &models.Feature{
				EpicID: epic.ID,
				Key:    "E0" + string(rune('0'+i)) + "-F0" + string(rune('0'+j)),
				Title:  "Test Feature " + string(rune('0'+j)),
				Status: models.FeatureStatusDraft,
			}
			err = featureRepo.Create(ctx, feature)
			require.NoError(t, err)

			// Add a task
			task := &models.Task{
				FeatureID: feature.ID,
				Key:       "T-E0" + string(rune('0'+i)) + "-F0" + string(rune('0'+j)) + "-001",
				Title:     "Test Task",
				Status:    models.TaskStatusTodo,
				Priority:  5,
			}
			err = taskRepo.Create(ctx, task)
			require.NoError(t, err)

			// Track the first task (E01-F01-001) for status update
			if i == 1 && j == 1 {
				firstTaskID = task.ID
			}
		}
	}

	// Make one task in_progress to see changes
	err := taskRepo.UpdateStatusForced(ctx, firstTaskID, models.TaskStatusInProgress, nil, nil, nil, true)
	require.NoError(t, err)

	summary, err := calcService.RecalculateAll(ctx)
	require.NoError(t, err)

	// Should have 4 features + 2 epics = 6 changes processed
	assert.Len(t, summary.Changes, 6)
	// At least E01-F01 should be updated to active
	assert.GreaterOrEqual(t, summary.FeaturesUpdated, 1)
	// At least E01 should be updated to active
	assert.GreaterOrEqual(t, summary.EpicsUpdated, 1)
	// Duration may be 0 for fast operations (sub-millisecond)
	assert.GreaterOrEqual(t, summary.DurationMs, int64(0))
}
