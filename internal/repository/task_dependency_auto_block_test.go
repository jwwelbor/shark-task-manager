package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRepository_AutoBlockDependents_OnReopen(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E96-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E96-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E96'")

	// Create test epic and feature
	epicResult, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E96', 'Test Epic 96', 'Test epic for auto-block', 'active', 'high')
	`)
	require.NoError(t, err)
	epicID, _ := epicResult.LastInsertId()

	featureResult, err := database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E96-F01', 'Test Feature 96', 'Test feature', 'active')
	`, epicID)
	require.NoError(t, err)
	featureID, _ := featureResult.LastInsertId()

	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE feature_id = ?", featureID) }()
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID) }()
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID) }()

	tests := []struct {
		name                   string
		setupTasks             []*models.Task
		taskToReopen           string
		expectedBlockedTasks   []string
		expectedUnblockedTasks []string
	}{
		{
			name: "reopen task should block direct dependent",
			setupTasks: []*models.Task{
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-001",
					Title:       "Base Task",
					Status:      models.TaskStatusReadyForReview, // Will be reopened
					Priority:    5,
					DependsOn:   nil,
					Description: stringPtr("Base task"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-002",
					Title:       "Dependent Task",
					Status:      models.TaskStatusInProgress, // Should be blocked
					Priority:    5,
					DependsOn:   stringPtr(`["T-E96-F01-001"]`),
					Description: stringPtr("Depends on base task"),
				},
			},
			taskToReopen:           "T-E96-F01-001",
			expectedBlockedTasks:   []string{"T-E96-F01-002"},
			expectedUnblockedTasks: []string{},
		},
		{
			name: "reopen task should block transitive dependents",
			setupTasks: []*models.Task{
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-003",
					Title:       "Base Task",
					Status:      models.TaskStatusReadyForReview,
					Priority:    5,
					DependsOn:   nil,
					Description: stringPtr("Base task"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-004",
					Title:       "Middle Task",
					Status:      models.TaskStatusInProgress,
					Priority:    5,
					DependsOn:   stringPtr(`["T-E96-F01-003"]`),
					Description: stringPtr("Depends on base"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-005",
					Title:       "Leaf Task",
					Status:      models.TaskStatusTodo,
					Priority:    5,
					DependsOn:   stringPtr(`["T-E96-F01-004"]`),
					Description: stringPtr("Depends on middle"),
				},
			},
			taskToReopen:           "T-E96-F01-003",
			expectedBlockedTasks:   []string{"T-E96-F01-004", "T-E96-F01-005"},
			expectedUnblockedTasks: []string{},
		},
		{
			name: "reopen task should block only non-completed dependents",
			setupTasks: []*models.Task{
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-006",
					Title:       "Base Task",
					Status:      models.TaskStatusReadyForReview,
					Priority:    5,
					DependsOn:   nil,
					Description: stringPtr("Base task"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-007",
					Title:       "Completed Dependent",
					Status:      models.TaskStatusCompleted, // Already completed, shouldn't be blocked
					Priority:    5,
					DependsOn:   stringPtr(`["T-E96-F01-006"]`),
					Description: stringPtr("Completed dependent"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-008",
					Title:       "Active Dependent",
					Status:      models.TaskStatusInProgress, // Should be blocked
					Priority:    5,
					DependsOn:   stringPtr(`["T-E96-F01-006"]`),
					Description: stringPtr("Active dependent"),
				},
			},
			taskToReopen:           "T-E96-F01-006",
			expectedBlockedTasks:   []string{"T-E96-F01-008"},
			expectedUnblockedTasks: []string{"T-E96-F01-007"}, // Completed tasks stay completed
		},
		{
			name: "reopen task with no dependents should not block anything",
			setupTasks: []*models.Task{
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-009",
					Title:       "Isolated Task",
					Status:      models.TaskStatusReadyForReview,
					Priority:    5,
					DependsOn:   nil,
					Description: stringPtr("No dependents"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E96-F01-010",
					Title:       "Independent Task",
					Status:      models.TaskStatusInProgress,
					Priority:    5,
					DependsOn:   nil, // No dependency
					Description: stringPtr("Independent"),
				},
			},
			taskToReopen:           "T-E96-F01-009",
			expectedBlockedTasks:   []string{},
			expectedUnblockedTasks: []string{"T-E96-F01-010"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			for _, task := range tt.setupTasks {
				_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", task.Key)
			}

			// Create setup tasks
			for _, task := range tt.setupTasks {
				// Create task with todo status first
				originalStatus := task.Status
				task.Status = models.TaskStatusTodo
				err := taskRepo.Create(ctx, task)
				require.NoError(t, err, "failed to create setup task")

				// If task should have non-todo status, update it using forced method
				if originalStatus != models.TaskStatusTodo {
					err = taskRepo.UpdateStatusForced(ctx, task.ID, originalStatus, nil, nil, nil, true)
					require.NoError(t, err, "failed to update task to %s", originalStatus)

					// Refresh task to get updated status
					_, err = taskRepo.GetByKey(ctx, task.Key)
					require.NoError(t, err, "failed to refresh task")
				}
			}

			// Find the task to reopen
			taskToReopen, err := taskRepo.GetByKey(ctx, tt.taskToReopen)
			require.NoError(t, err, "failed to get task to reopen")

			// Reopen the task (this should trigger auto-blocking)
			err = taskRepo.ReopenTaskWithAutoBlock(ctx, taskToReopen.ID, nil, nil)
			require.NoError(t, err, "failed to reopen task")

			// Verify reopened task is now in_progress
			reopenedTask, err := taskRepo.GetByKey(ctx, tt.taskToReopen)
			require.NoError(t, err)
			assert.Equal(t, models.TaskStatusInProgress, reopenedTask.Status, "reopened task should be in_progress")

			// Verify expected blocked tasks are now blocked
			for _, expectedBlockedKey := range tt.expectedBlockedTasks {
				task, err := taskRepo.GetByKey(ctx, expectedBlockedKey)
				require.NoError(t, err, "failed to get task %s", expectedBlockedKey)
				assert.Equal(t, models.TaskStatusBlocked, task.Status,
					"task %s should be blocked after prerequisite was reopened", expectedBlockedKey)
				assert.NotNil(t, task.BlockedReason, "blocked task should have a reason")
				assert.Contains(t, *task.BlockedReason, tt.taskToReopen,
					"blocked reason should mention the reopened prerequisite")
			}

			// Verify expected unblocked tasks remain in their original status
			for _, expectedUnblockedKey := range tt.expectedUnblockedTasks {
				task, err := taskRepo.GetByKey(ctx, expectedUnblockedKey)
				require.NoError(t, err, "failed to get task %s", expectedUnblockedKey)
				assert.NotEqual(t, models.TaskStatusBlocked, task.Status,
					"task %s should not be blocked", expectedUnblockedKey)
			}

			// Clean up
			for _, task := range tt.setupTasks {
				_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
			}
		})
	}
}

func TestTaskRepository_ReopenTaskWithAutoBlock_TransitiveBlocking(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E95-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E95-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E95'")

	// Create test epic and feature
	epicResult, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E95', 'Test Epic 95', 'Test epic', 'active', 'high')
	`)
	require.NoError(t, err)
	epicID, _ := epicResult.LastInsertId()

	featureResult, err := database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E95-F01', 'Test Feature 95', 'Test feature', 'active')
	`, epicID)
	require.NoError(t, err)
	featureID, _ := featureResult.LastInsertId()

	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE feature_id = ?", featureID) }()
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID) }()
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID) }()

	// Create a complex dependency chain:
	// T1 (base)
	// T2 depends on T1
	// T3 depends on T2
	// T4 depends on T1 and T3 (diamond dependency)
	tasks := []*models.Task{
		{
			FeatureID:   featureID,
			Key:         "T-E95-F01-001",
			Title:       "Base Task",
			Status:      models.TaskStatusReadyForReview,
			Priority:    5,
			DependsOn:   nil,
			Description: stringPtr("Base task"),
		},
		{
			FeatureID:   featureID,
			Key:         "T-E95-F01-002",
			Title:       "Task 2",
			Status:      models.TaskStatusInProgress,
			Priority:    5,
			DependsOn:   stringPtr(`["T-E95-F01-001"]`),
			Description: stringPtr("Depends on T1"),
		},
		{
			FeatureID:   featureID,
			Key:         "T-E95-F01-003",
			Title:       "Task 3",
			Status:      models.TaskStatusTodo,
			Priority:    5,
			DependsOn:   stringPtr(`["T-E95-F01-002"]`),
			Description: stringPtr("Depends on T2"),
		},
		{
			FeatureID:   featureID,
			Key:         "T-E95-F01-004",
			Title:       "Task 4",
			Status:      models.TaskStatusTodo,
			Priority:    5,
			DependsOn:   stringPtr(`["T-E95-F01-001", "T-E95-F01-003"]`),
			Description: stringPtr("Depends on T1 and T3 (diamond)"),
		},
	}

	// Create all tasks
	for _, task := range tasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", task.Key)

		// Create with todo status first
		originalStatus := task.Status
		task.Status = models.TaskStatusTodo
		err := taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Update status if needed using forced method
		if originalStatus != models.TaskStatusTodo {
			err = taskRepo.UpdateStatusForced(ctx, task.ID, originalStatus, nil, nil, nil, true)
			require.NoError(t, err)
		}

		// Refresh to get updated status
		_, err = taskRepo.GetByKey(ctx, task.Key)
		require.NoError(t, err)
	}

	defer func() {
		for _, task := range tasks {
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
		}
	}()

	// Reopen T1 (base task)
	baseTask, err := taskRepo.GetByKey(ctx, "T-E95-F01-001")
	require.NoError(t, err)

	err = taskRepo.ReopenTaskWithAutoBlock(ctx, baseTask.ID, nil, nil)
	require.NoError(t, err)

	// Verify T1 is now in_progress
	t1, err := taskRepo.GetByKey(ctx, "T-E95-F01-001")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusInProgress, t1.Status)

	// Verify all transitive dependents are blocked
	t2, err := taskRepo.GetByKey(ctx, "T-E95-F01-002")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusBlocked, t2.Status, "T2 should be blocked (direct dependent of T1)")

	t3, err := taskRepo.GetByKey(ctx, "T-E95-F01-003")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusBlocked, t3.Status, "T3 should be blocked (transitive dependent via T2)")

	t4, err := taskRepo.GetByKey(ctx, "T-E95-F01-004")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusBlocked, t4.Status, "T4 should be blocked (depends on T1 and T3)")
}
