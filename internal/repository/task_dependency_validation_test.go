package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRepository_ValidateTaskDependencies_Create(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up test data first - delete tasks, features, and epics to ensure clean state
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E99-F99')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E99')")

	// Seed test data
	_, featureID := test.SeedTestData()
	require.NotZero(t, featureID, "featureID should not be zero after seeding")

	tests := []struct {
		name           string
		setupTasks     []*models.Task
		newTask        *models.Task
		expectValid    bool
		expectedErrMsg string
	}{
		{
			name:       "valid - no dependencies",
			setupTasks: []*models.Task{},
			newTask: &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F01-001",
				Title: "Task 1",

				Description: stringPtr("Task 1")}, FeatureID: featureID,

				Status:    models.TaskStatus("todo"),
				Priority:  5,
				DependsOn: nil,
			},
			expectValid: true,
		},
		{
			name: "valid - linear dependency chain",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-002",
						Title:       "Task 2",
						Description: stringPtr("Task 2"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil,
				},
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-003",
						Title:       "Task 3",
						Description: stringPtr("Task 3"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: stringPtr(`["T-E99-F01-002"]`),
				},
			},
			newTask: &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F01-004",
				Title: "Task 4",

				Description: stringPtr("Task 4")}, FeatureID: featureID,

				Status:    models.TaskStatus("todo"),
				Priority:  5,
				DependsOn: stringPtr(`["T-E99-F01-003"]`),
			},
			expectValid: true,
		},
		{
			name: "invalid - dependency does not exist",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-005",
						Title:       "Task 5",
						Description: stringPtr("Task 5"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil,
				},
			},
			newTask: &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F01-006",
				Title: "Task 6",

				Description: stringPtr("Task 6")}, FeatureID: featureID,

				Status:    models.TaskStatus("todo"),
				Priority:  5,
				DependsOn: stringPtr(`["T-E99-F01-999"]`),
			},
			expectValid:    false,
			expectedErrMsg: "dependency does not exist",
		},
		{
			name: "invalid - would create simple cycle",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-007",
						Title:       "Task 7",
						Description: stringPtr("Task 7"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil, // Create without dependency first
				},
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-008",
						Title:       "Task 8",
						Description: stringPtr("Task 8"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: stringPtr(`["T-E99-F01-007"]`), // T8 depends on T7
				},
			},
			newTask:        nil, // We'll test by updating T7 to depend on T8
			expectValid:    false,
			expectedErrMsg: "would create circular dependency",
		},
		{
			name: "invalid - complex cycle",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-009",
						Title:       "Task 9",
						Description: stringPtr("Task 9"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil, // Create without dependency first
				},
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-010",
						Title:       "Task 10",
						Description: stringPtr("Task 10"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: stringPtr(`["T-E99-F01-009"]`), // T10 depends on T9
				},
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-011",
						Title:       "Task 11",
						Description: stringPtr("Task 11"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: stringPtr(`["T-E99-F01-010"]`), // T11 depends on T10
				},
			},
			newTask:        nil, // We'll update T9 to depend on T11, creating a cycle
			expectValid:    false,
			expectedErrMsg: "would create circular dependency",
		},
		{
			name:       "invalid - self-reference",
			setupTasks: []*models.Task{},
			newTask: &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F01-012",
				Title: "Task 12",

				Description: stringPtr("Task 12")}, FeatureID: featureID,

				Status:    models.TaskStatus("todo"),
				Priority:  5,
				DependsOn: stringPtr(`["T-E99-F01-012"]`),
			},
			expectValid:    false,
			expectedErrMsg: "task cannot depend on itself",
		},
		{
			name: "valid - diamond dependency (no cycle)",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-013",
						Title:       "Task 13",
						Description: stringPtr("Task 13"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil,
				},
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-014",
						Title:       "Task 14",
						Description: stringPtr("Task 14"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: stringPtr(`["T-E99-F01-013"]`),
				},
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-015",
						Title:       "Task 15",
						Description: stringPtr("Task 15"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: stringPtr(`["T-E99-F01-013"]`),
				},
			},
			newTask: &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F01-016",
				Title: "Task 16",

				Description: stringPtr("Task 16")}, FeatureID: featureID,

				Status:    models.TaskStatus("todo"),
				Priority:  5,
				DependsOn: stringPtr(`["T-E99-F01-014", "T-E99-F01-015"]`),
			},
			expectValid: true,
		},
		{
			name: "invalid - malformed JSON dependencies",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E99-F01-017",
						Title:       "Task 17",
						Description: stringPtr("Task 17"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil,
				},
			},
			newTask: &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F01-018",
				Title: "Task 18",

				Description: stringPtr("Task 18")}, FeatureID: featureID,

				Status:    models.TaskStatus("todo"),
				Priority:  5,
				DependsOn: stringPtr(`invalid json`),
			},
			expectValid:    false,
			expectedErrMsg: "invalid depends_on: must be a valid JSON array",
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

			// Test validation
			if tt.newTask == nil {
				// Special cycle test - update first task to create cycle
				var taskToUpdate *models.Task
				var err error
				if tt.name == "invalid - would create simple cycle" {
					taskToUpdate, err = taskRepo.GetByKey(ctx, "T-E99-F01-007")
					require.NoError(t, err)
					taskToUpdate.DependsOn = stringPtr(`["T-E99-F01-008"]`)
				} else if tt.name == "invalid - complex cycle" {
					taskToUpdate, err = taskRepo.GetByKey(ctx, "T-E99-F01-009")
					require.NoError(t, err)
					taskToUpdate.DependsOn = stringPtr(`["T-E99-F01-011"]`)
				}

				if taskToUpdate != nil {
					err = taskRepo.Update(ctx, taskToUpdate)
					require.Error(t, err, "expected validation to fail")
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				// Regular create test
				err := taskRepo.Create(ctx, tt.newTask)

				if tt.expectValid {
					assert.NoError(t, err, "expected validation to pass")
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

func TestTaskRepository_ValidateTaskDependencies_Update(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up test data first - delete tasks, features, and epics to ensure clean state
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-%' OR key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E99-F99')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E99')")

	// Seed test data
	_, featureID := test.SeedTestData()
	require.NotZero(t, featureID, "featureID should not be zero after seeding")

	tests := []struct {
		name           string
		setupTasks     []*models.Task
		updateTaskKey  string             // Which task to update
		updateTask     func(*models.Task) // Modify task before validation
		expectValid    bool
		expectedErrMsg string
	}{
		{
			name: "valid - add dependency to existing task",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E98-F01-001",
						Title:       "Task 1",
						Description: stringPtr("Task 1"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil,
				},
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E98-F01-002",
						Title:       "Task 2",
						Description: stringPtr("Task 2"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil,
				},
			},
			updateTaskKey: "T-E98-F01-002", // Update Task 2 to depend on Task 1
			updateTask: func(task *models.Task) {
				task.DependsOn = stringPtr(`["T-E98-F01-001"]`)
			},
			expectValid: true,
		},
		{
			name: "invalid - update would create cycle",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E98-F01-003",
						Title:       "Task 3",
						Description: stringPtr("Task 3"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil,
				},
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E98-F01-004",
						Title:       "Task 4",
						Description: stringPtr("Task 4"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: stringPtr(`["T-E98-F01-003"]`),
				},
			},
			updateTaskKey: "T-E98-F01-003", // Update Task 3
			updateTask: func(task *models.Task) {
				// Task 3 now depends on Task 4, creating a cycle
				task.DependsOn = stringPtr(`["T-E98-F01-004"]`)
			},
			expectValid:    false,
			expectedErrMsg: "would create circular dependency",
		},
		{
			name: "invalid - update adds non-existent dependency",
			setupTasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{
						Key:         "T-E98-F01-005",
						Title:       "Task 5",
						Description: stringPtr("Task 5"),
					},
					FeatureID: featureID,
					Status:    models.TaskStatus("todo"),
					Priority:  5,
					DependsOn: nil,
				},
			},
			updateTaskKey: "T-E98-F01-005", // Update Task 5
			updateTask: func(task *models.Task) {
				task.DependsOn = stringPtr(`["T-E98-F01-999"]`)
			},
			expectValid:    false,
			expectedErrMsg: "dependency does not exist",
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
				err := taskRepo.Create(ctx, task)
				require.NoError(t, err, "failed to create setup task")
			}

			// Get the task to update using the specified key
			// We need to get a fresh copy from DB to ensure we have all fields
			taskToUpdate, err := taskRepo.GetByKey(ctx, tt.updateTaskKey)
			require.NoError(t, err, "failed to get task to update")

			// Apply update
			tt.updateTask(taskToUpdate)

			// Validate updated task dependencies
			err = taskRepo.ValidateTaskDependencies(ctx, taskToUpdate)

			if tt.expectValid {
				assert.NoError(t, err, "expected validation to pass")
			} else {
				require.Error(t, err, "expected validation to fail")
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			}

			// Clean up
			for _, task := range tt.setupTasks {
				_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
			}
		})
	}
}

func TestTaskRepository_GetTaskDependents(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E97-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create test epic and feature directly
	epicResult, err := database.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E97', 'Test Epic 97', 'Test epic for dependents', 'active', 'high')
	`)
	require.NoError(t, err)
	epicID, _ := epicResult.LastInsertId()

	featureResult, err := database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E97-F01', 'Test Feature 97', 'Test feature', 'active')
	`, epicID)
	require.NoError(t, err)
	featureID, _ := featureResult.LastInsertId()

	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE feature_id = ?", featureID) }()
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID) }()
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID) }()

	// Create test tasks with dependency structure
	// Task 1 (no dependencies)
	// Task 2 depends on Task 1
	// Task 3 depends on Task 1
	// Task 4 depends on Task 2 and Task 3
	tasks := []*models.Task{
		{
			BaseEntity: models.BaseEntity{
				Key:         "T-E97-F01-001",
				Title:       "Base Task",
				Description: stringPtr("Base task"),
			},
			FeatureID: featureID,
			Status:    models.TaskStatus("todo"),
			Priority:  5,
			DependsOn: nil,
		},
		{
			BaseEntity: models.BaseEntity{
				Key:         "T-E97-F01-002",
				Title:       "Task 2",
				Description: stringPtr("Depends on task 1"),
			},
			FeatureID: featureID,
			Status:    models.TaskStatus("todo"),
			Priority:  5,
			DependsOn: stringPtr(`["T-E97-F01-001"]`),
		},
		{
			BaseEntity: models.BaseEntity{
				Key:         "T-E97-F01-003",
				Title:       "Task 3",
				Description: stringPtr("Depends on task 1"),
			},
			FeatureID: featureID,
			Status:    models.TaskStatus("todo"),
			Priority:  5,
			DependsOn: stringPtr(`["T-E97-F01-001"]`),
		},
		{
			BaseEntity: models.BaseEntity{
				Key:         "T-E97-F01-004",
				Title:       "Task 4",
				Description: stringPtr("Depends on task 2 and 3"),
			},
			FeatureID: featureID,
			Status:    models.TaskStatus("todo"),
			Priority:  5,
			DependsOn: stringPtr(`["T-E97-F01-002", "T-E97-F01-003"]`),
		},
	}

	// Clean up and create tasks
	for _, task := range tasks {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", task.Key)
		err := taskRepo.Create(ctx, task)
		require.NoError(t, err)
	}
	defer func() {
		for _, task := range tasks {
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
		}
	}()

	tests := []struct {
		name             string
		taskKey          string
		expectedCount    int
		expectedTaskKeys []string
	}{
		{
			name:             "get dependents of base task",
			taskKey:          "T-E97-F01-001",
			expectedCount:    2,
			expectedTaskKeys: []string{"T-E97-F01-002", "T-E97-F01-003"},
		},
		{
			name:             "get dependents of middle task",
			taskKey:          "T-E97-F01-002",
			expectedCount:    1,
			expectedTaskKeys: []string{"T-E97-F01-004"},
		},
		{
			name:             "get dependents of leaf task",
			taskKey:          "T-E97-F01-004",
			expectedCount:    0,
			expectedTaskKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dependents, err := taskRepo.GetTaskDependents(ctx, tt.taskKey)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(dependents))

			// Check that expected tasks are in the dependents list
			foundKeys := make(map[string]bool)
			for _, dep := range dependents {
				foundKeys[dep.Key] = true
			}

			for _, expectedKey := range tt.expectedTaskKeys {
				assert.True(t, foundKeys[expectedKey], "expected to find dependent task %s", expectedKey)
			}
		})
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO entity_relationships
			(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
		VALUES ('task', ?, 'task', ?, 'depends_on')
	`, tasks[2].ID, tasks[1].ID)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, `
			DELETE FROM entity_relationships
			WHERE from_entity_type = 'task' AND from_entity_id = ?
			  AND to_entity_type = 'task' AND to_entity_id = ?
			  AND relationship_type = 'depends_on'
		`, tasks[2].ID, tasks[1].ID)
	}()

	dependencyTests := []struct {
		name             string
		taskKey          string
		expectedTaskKeys []string
	}{
		{
			name:             "get dependencies of base task",
			taskKey:          "T-E97-F01-001",
			expectedTaskKeys: []string{},
		},
		{
			name:             "get dependencies from legacy depends_on",
			taskKey:          "T-E97-F01-004",
			expectedTaskKeys: []string{"T-E97-F01-002", "T-E97-F01-003"},
		},
		{
			name:             "get dependencies from legacy and entity relationships",
			taskKey:          "T-E97-F01-003",
			expectedTaskKeys: []string{"T-E97-F01-001", "T-E97-F01-002"},
		},
	}

	for _, tt := range dependencyTests {
		t.Run(tt.name, func(t *testing.T) {
			dependencies, err := taskRepo.GetTaskDependencies(ctx, tt.taskKey)
			require.NoError(t, err)
			assert.Len(t, dependencies, len(tt.expectedTaskKeys))

			foundKeys := make(map[string]bool)
			for _, dep := range dependencies {
				foundKeys[dep.Key] = true
			}

			for _, expectedKey := range tt.expectedTaskKeys {
				assert.True(t, foundKeys[expectedKey], "expected to find dependency task %s", expectedKey)
			}
		})
	}
}
