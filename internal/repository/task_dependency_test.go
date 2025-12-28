package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRepository_ValidateDependencies(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up test data first - only delete E99-F01 data used by this test
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F01'")

	// Get or create test epic E99
	var epicID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = 'E99'").Scan(&epicID)
	if err != nil {
		// Epic doesn't exist, create it
		epicResult, err := database.ExecContext(ctx, `
			INSERT INTO epics (key, title, description, status, priority)
			VALUES ('E99', 'Test Epic 99', 'Test epic', 'active', 'high')
		`)
		require.NoError(t, err)
		epicID, _ = epicResult.LastInsertId()
	}

	// Create test feature (will fail if exists, which is expected after cleanup)
	featureResult, err := database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E99-F01', 'Test Feature 99', 'Test feature', 'active')
	`, epicID)
	require.NoError(t, err)
	featureID, _ := featureResult.LastInsertId()

	// Cleanup is done at the beginning of the test, not at the end

	tests := []struct {
		name           string
		setupTasks     []*models.Task
		newTask        *models.Task
		expectValid    bool
		expectedErrMsg string
	}{
		{
			name: "valid - linear dependency chain",
			setupTasks: []*models.Task{
				{
					FeatureID:   featureID,
					Key:         "T-E99-F01-001",
					Title:       "Task 1",
					Status:      models.TaskStatusTodo,
					Priority:    5,
					DependsOn:   nil,
					Description: stringPtr("Task 1"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E99-F01-002",
					Title:       "Task 2",
					Status:      models.TaskStatusTodo,
					Priority:    5,
					DependsOn:   stringPtr(`["T-E99-F01-001"]`),
					Description: stringPtr("Task 2"),
				},
			},
			newTask: &models.Task{
				FeatureID:   featureID,
				Key:         "T-E99-F01-003",
				Title:       "Task 3",
				Status:      models.TaskStatusTodo,
				Priority:    5,
				DependsOn:   stringPtr(`["T-E99-F01-002"]`),
				Description: stringPtr("Task 3"),
			},
			expectValid: true,
		},
		{
			name: "invalid - would create simple cycle",
			setupTasks: []*models.Task{
				{
					FeatureID:   featureID,
					Key:         "T-E99-F01-004",
					Title:       "Task 4",
					Status:      models.TaskStatusTodo,
					Priority:    5,
					DependsOn:   nil, // Create without dependency first
					Description: stringPtr("Task 4"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E99-F01-005",
					Title:       "Task 5",
					Status:      models.TaskStatusTodo,
					Priority:    5,
					DependsOn:   stringPtr(`["T-E99-F01-004"]`), // This depends on T4
					Description: stringPtr("Task 5"),
				},
			},
			newTask:        nil, // We'll update T4 to depend on T5, creating a cycle
			expectValid:    false,
			expectedErrMsg: "would create circular dependency",
		},
		{
			name:       "invalid - self-reference",
			setupTasks: []*models.Task{},
			newTask: &models.Task{
				FeatureID:   featureID,
				Key:         "T-E99-F01-009",
				Title:       "Task 9",
				Status:      models.TaskStatusTodo,
				Priority:    5,
				DependsOn:   stringPtr(`["T-E99-F01-009"]`),
				Description: stringPtr("Task 9"),
			},
			expectValid:    false,
			expectedErrMsg: "task cannot depend on itself",
		},
		{
			name: "valid - diamond dependency (no cycle)",
			setupTasks: []*models.Task{
				{
					FeatureID:   featureID,
					Key:         "T-E99-F01-010",
					Title:       "Task 10",
					Status:      models.TaskStatusTodo,
					Priority:    5,
					DependsOn:   nil,
					Description: stringPtr("Task 10"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E99-F01-011",
					Title:       "Task 11",
					Status:      models.TaskStatusTodo,
					Priority:    5,
					DependsOn:   stringPtr(`["T-E99-F01-010"]`),
					Description: stringPtr("Task 11"),
				},
				{
					FeatureID:   featureID,
					Key:         "T-E99-F01-012",
					Title:       "Task 12",
					Status:      models.TaskStatusTodo,
					Priority:    5,
					DependsOn:   stringPtr(`["T-E99-F01-010"]`),
					Description: stringPtr("Task 12"),
				},
			},
			newTask: &models.Task{
				FeatureID:   featureID,
				Key:         "T-E99-F01-013",
				Title:       "Task 13",
				Status:      models.TaskStatusTodo,
				Priority:    5,
				DependsOn:   stringPtr(`["T-E99-F01-011", "T-E99-F01-012"]`),
				Description: stringPtr("Task 13"),
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			for _, task := range tt.setupTasks {
				_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", task.Key)
			}
			if tt.newTask != nil {
				_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", tt.newTask.Key)
			}

			// Create setup tasks
			for _, task := range tt.setupTasks {
				err := taskRepo.Create(ctx, task)
				require.NoError(t, err, "failed to create setup task")
			}

			// Special handling for cycle test - update T4 to depend on T5
			if tt.name == "invalid - would create simple cycle" {
				task4, err := taskRepo.GetByKey(ctx, "T-E99-F01-004")
				require.NoError(t, err)
				task4.DependsOn = stringPtr(`["T-E99-F01-005"]`)
				err = taskRepo.Update(ctx, task4)
				require.Error(t, err, "expected validation to fail")
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				// Try to create the task (validation happens inside Create now)
				err := taskRepo.Create(ctx, tt.newTask)

				if tt.expectValid {
					assert.NoError(t, err, "expected validation to pass and task creation to succeed")
				} else {
					require.Error(t, err, "expected validation to fail")
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			}

			// Clean up
			for _, task := range tt.setupTasks {
				_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
			}
			if tt.newTask != nil && tt.newTask.ID != 0 {
				_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", tt.newTask.ID)
			}
		})
	}
}

// This function has been moved to TaskRepository.ValidateTaskDependencies
// and is now integrated into Create and Update methods

func TestBuildDependencyGraph(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-%' OR key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E99-F99')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E99')")

	// Seed test data - use existing test infrastructure
	epicID, featureID := test.SeedTestData()
	require.NotZero(t, epicID, "epicID should not be zero")
	require.NotZero(t, featureID, "featureID should not be zero")

	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE feature_id = ?", featureID)

	// Create test tasks with dependencies
	tasks := []*models.Task{
		{
			FeatureID:   featureID,
			Key:         "T-E98-F01-001",
			Title:       "Base Task",
			Status:      models.TaskStatusTodo,
			Priority:    5,
			DependsOn:   nil,
			Description: stringPtr("Base task with no dependencies"),
		},
		{
			FeatureID:   featureID,
			Key:         "T-E98-F01-002",
			Title:       "Task 2",
			Status:      models.TaskStatusTodo,
			Priority:    5,
			DependsOn:   stringPtr(`["T-E98-F01-001"]`),
			Description: stringPtr("Depends on task 1"),
		},
		{
			FeatureID:   featureID,
			Key:         "T-E98-F01-003",
			Title:       "Task 3",
			Status:      models.TaskStatusTodo,
			Priority:    5,
			DependsOn:   stringPtr(`["T-E98-F01-002"]`),
			Description: stringPtr("Depends on task 2"),
		},
		{
			FeatureID:   featureID,
			Key:         "T-E98-F01-004",
			Title:       "Task 4",
			Status:      models.TaskStatusTodo,
			Priority:    5,
			DependsOn:   stringPtr(`["T-E98-F01-001", "T-E98-F01-002"]`),
			Description: stringPtr("Depends on task 1 and 2"),
		},
	}

	// Create all tasks
	for _, task := range tasks {
		err := taskRepo.Create(ctx, task)
		require.NoError(t, err, "failed to create task")
	}
	defer func() {
		for _, task := range tasks {
			database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
		}
	}()

	// Build dependency graph
	detector, err := taskRepo.BuildDependencyGraphForFeature(ctx, featureID)
	require.NoError(t, err)

	// Test cycle detection for valid graph
	hasCycle, cyclePath, err := detector.DetectCycle(ctx, "T-E98-F01-003")
	assert.False(t, hasCycle, "expected no cycle in valid graph")
	assert.NoError(t, err)
	assert.Empty(t, cyclePath)

	// Test dependency chain
	chain := detector.GetDependencyChain(ctx, "T-E98-F01-003")
	assert.Contains(t, chain, "T-E98-F01-003")
	assert.Contains(t, chain, "T-E98-F01-002")
	assert.Contains(t, chain, "T-E98-F01-001")
}

// This function has been moved to TaskRepository.BuildDependencyGraphForFeature
