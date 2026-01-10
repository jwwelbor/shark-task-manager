package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskRepository_Create_GeneratesAndStoresSlug verifies slug generation during task creation
func TestTaskRepository_Create_GeneratesAndStoresSlug(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E95-F01-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E95-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E95'")

	// Create dedicated epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E95",
		Title:         "Test Epic for Slug Generation",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create dedicated feature for this test
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E95-F01",
		Title:  "Test Feature for Slug Generation",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create task
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E95-F01-001",
		Title:     "Implement User Authentication System",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}

	err = repo.Create(ctx, task)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Verify slug was generated and stored
	assert.NotNil(t, task.Slug, "Slug should be generated")
	assert.Equal(t, "implement-user-authentication-system", *task.Slug)

	// Verify slug is persisted in database
	retrieved, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.Slug, "Slug should be persisted")
	assert.Equal(t, "implement-user-authentication-system", *retrieved.Slug)
}

// TestTaskRepository_Create_SlugHandlesSpecialCharacters verifies slug handles special characters
func TestTaskRepository_Create_SlugHandlesSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key IN ('T-E97-F01-001', 'T-E97-F01-002', 'T-E97-F01-003')")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create a dedicated test epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E97",
		Title:         "Test Epic for Task Slug Special Characters",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create a dedicated test feature
	featureRepo := NewFeatureRepository(db)
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E97-F01",
		Title:  "Test Feature for Task Slugs",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	testCases := []struct {
		title        string
		expectedSlug string
	}{
		{
			title:        "Fix Bug: Memory Leak in Worker Pool",
			expectedSlug: "fix-bug-memory-leak-in-worker-pool",
		},
		{
			title:        "Upgrade PostgreSQL -> MongoDB",
			expectedSlug: "upgrade-postgresql-mongodb",
		},
		{
			title:        "Add Support for UTF-8 & Unicode 测试",
			expectedSlug: "add-support-for-utf-8-unicode",
		},
	}

	for i, tc := range testCases {
		task := &models.Task{
			FeatureID: testFeature.ID,
			Key:       fmt.Sprintf("T-E97-F01-%03d", i+1),
			Title:     tc.title,
			Status:    models.TaskStatusTodo,
			Priority:  5,
		}

		err := repo.Create(ctx, task)
		require.NoError(t, err, "Failed to create task with key %s, title: %s", task.Key, tc.title)

		assert.NotNil(t, task.Slug, "Slug should be generated for: %s", tc.title)
		assert.Equal(t, tc.expectedSlug, *task.Slug, "Slug mismatch for: %s", tc.title)

		// Cleanup
		defer func(id int64) {
			if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}(task.ID)
	}
}

// TestTaskRepository_UpdateCascadesOrder verifies that updating a task's execution order
// automatically resequences all other tasks in the same feature
func TestTaskRepository_UpdateCascadesOrder(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E98",
		Title:         "Test Epic for Order Cascade",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create test feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E98-F01",
		Title:  "Test Feature for Order Cascade",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	// Create four tasks with sequential orders: a-1, b-2, c-3, d-4
	order1, order2, order3, order4 := 1, 2, 3, 4
	taskA := &models.Task{
		FeatureID:      testFeature.ID,
		Key:            "T-E98-F01-001",
		Title:          "Task A",
		Status:         models.TaskStatusTodo,
		Priority:       5,
		ExecutionOrder: &order1,
	}
	taskB := &models.Task{
		FeatureID:      testFeature.ID,
		Key:            "T-E98-F01-002",
		Title:          "Task B",
		Status:         models.TaskStatusTodo,
		Priority:       5,
		ExecutionOrder: &order2,
	}
	taskC := &models.Task{
		FeatureID:      testFeature.ID,
		Key:            "T-E98-F01-003",
		Title:          "Task C",
		Status:         models.TaskStatusTodo,
		Priority:       5,
		ExecutionOrder: &order3,
	}
	taskD := &models.Task{
		FeatureID:      testFeature.ID,
		Key:            "T-E98-F01-004",
		Title:          "Task D",
		Status:         models.TaskStatusTodo,
		Priority:       5,
		ExecutionOrder: &order4,
	}

	err = taskRepo.Create(ctx, taskA)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskA.ID) }()

	err = taskRepo.Create(ctx, taskB)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskB.ID) }()

	err = taskRepo.Create(ctx, taskC)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskC.ID) }()

	err = taskRepo.Create(ctx, taskD)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskD.ID) }()

	// When: Update task D's order from 4 to 2
	newOrder := 2
	taskD.ExecutionOrder = &newOrder
	err = taskRepo.Update(ctx, taskD)
	require.NoError(t, err, "Failed to update task D's order")

	// Then: Verify cascade - expected order: a-1, d-2, b-3, c-4
	// Get all tasks for this feature
	tasks, err := taskRepo.ListByFeature(ctx, testFeature.ID)
	require.NoError(t, err, "Failed to list tasks by feature ID")
	require.Len(t, tasks, 4, "Should have 4 tasks")

	// Build a map for easy verification
	taskOrders := make(map[string]int)
	for _, task := range tasks {
		if task.ExecutionOrder != nil {
			taskOrders[task.Title] = *task.ExecutionOrder
		}
	}

	// Verify expected orders
	assert.Equal(t, 1, taskOrders["Task A"], "Task A should be at order 1")
	assert.Equal(t, 2, taskOrders["Task D"], "Task D should be at order 2 (moved)")
	assert.Equal(t, 3, taskOrders["Task B"], "Task B should be at order 3 (shifted)")
	assert.Equal(t, 4, taskOrders["Task C"], "Task C should be at order 4 (shifted)")
}
