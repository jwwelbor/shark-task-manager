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
		Status:    models.TaskStatus("todo"),
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
			Status:    models.TaskStatus("todo"),
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
		Status:         models.TaskStatus("todo"),
		Priority:       5,
		ExecutionOrder: &order1,
	}
	taskB := &models.Task{
		FeatureID:      testFeature.ID,
		Key:            "T-E98-F01-002",
		Title:          "Task B",
		Status:         models.TaskStatus("todo"),
		Priority:       5,
		ExecutionOrder: &order2,
	}
	taskC := &models.Task{
		FeatureID:      testFeature.ID,
		Key:            "T-E98-F01-003",
		Title:          "Task C",
		Status:         models.TaskStatus("todo"),
		Priority:       5,
		ExecutionOrder: &order3,
	}
	taskD := &models.Task{
		FeatureID:      testFeature.ID,
		Key:            "T-E98-F01-004",
		Title:          "Task D",
		Status:         models.TaskStatus("todo"),
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

// TestTaskRepository_UpdateStatus_BackwardTransitionRequiresReason tests rejection reason validation
func TestTaskRepository_UpdateStatus_BackwardTransitionRequiresReason(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first (use unique numbers to avoid conflicts)
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-F98%%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-F98%%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create test epic
	highValue := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E98",
		Title:         "Test Rejection Reasons Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highValue,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create test feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E98-F98",
		Title:  "Test Rejection Reasons Feature",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	// Create a test task in in_progress status (development phase)
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E98-F98-001",
		Title:     "Test Rejection Reason Task",
		Status:    models.TaskStatus("in_progress"),
		Priority:  5,
	}
	err = repo.Create(ctx, task)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID) }()

	// Get the initial task status
	initialTask, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(initialTask.Status))

	t.Run("backward transition without reason succeeds at repo layer", func(t *testing.T) {
		// NOTE: Backward transition validation (reason requirement) has moved to the service layer.
		// The repository layer no longer validates transitions or requires reasons.

		// Ensure task starts in in_progress
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(current.Status))

		// Update to ready_for_review
		reason := "Test review"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err, "Forward transition should succeed")

		// Verify status changed
		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("ready_for_review"), models.TaskStatus(updated.Status))

		// Backward transition without reason now succeeds at repo layer
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, nil)
		assert.NoError(t, err, "Backward transition without reason should succeed at repo layer (validation moved to service)")

		// Verify status changed
		updated, err = repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(updated.Status))
	})

	t.Run("backward transition with reason should succeed", func(t *testing.T) {
		// Reset task to ready_for_review first
		reason := "Initial review"
		err := repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err, "Forward transition should succeed")

		// Verify task is in correct status
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, models.TaskStatus("ready_for_review"), models.TaskStatus(current.Status))

		// Now try backward transition WITH reason (ready_for_review -> in_progress)
		rejectionReason := "Missing error handling"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &rejectionReason)
		assert.NoError(t, err, "Backward transition with reason should succeed")

		// Verify status changed
		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(updated.Status))
	})

	t.Run("backward transition with force flag bypasses reason requirement", func(t *testing.T) {
		// Ensure task is in in_progress first
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		if current.Status != models.TaskStatus("in_progress") {
			resetReason := "Reset for test"
			_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &resetReason)
		}

		// Reset task to ready_for_review
		reason := "Review"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err)

		// Try backward transition with force but no reason
		err = repo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("in_progress"), nil, nil, nil, nil, true)
		assert.NoError(t, err, "Backward transition with force should bypass reason requirement")

		// Verify status changed
		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(updated.Status))
	})

	t.Run("forward transition without reason should succeed", func(t *testing.T) {
		// Ensure task is in in_progress first
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		if current.Status != models.TaskStatus("in_progress") {
			resetReason := "Reset for test"
			_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &resetReason)
		}

		// Forward transitions (planning -> development) should not require reason
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, nil)
		assert.NoError(t, err, "Forward transition should succeed without reason")

		// Verify status changed
		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("ready_for_review"), models.TaskStatus(updated.Status))
	})

	t.Run("empty reason string succeeds at repo layer for backward transition", func(t *testing.T) {
		// NOTE: Backward transition reason validation has moved to the service layer.
		// The repository layer no longer validates reasons.
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		if current.Status != models.TaskStatus("in_progress") {
			resetReason := "Reset for test"
			_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &resetReason)
		}

		// Reset to ready_for_review
		reason := "Review"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err)

		// Try with empty reason string - repo allows it now
		emptyReason := ""
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &emptyReason)
		assert.NoError(t, err, "Backward transition with empty reason should succeed at repo layer (validation moved to service)")
	})

	t.Run("whitespace-only reason succeeds at repo layer for backward transition", func(t *testing.T) {
		// NOTE: Backward transition reason validation has moved to the service layer.
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		if current.Status != models.TaskStatus("in_progress") {
			resetReason := "Reset for test"
			_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &resetReason)
		}

		// Reset to ready_for_review
		reason := "Review"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err)

		// Try with whitespace-only reason - repo allows it now
		whitespacedReason := "   \t\n  "
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &whitespacedReason)
		assert.NoError(t, err, "Backward transition with whitespace-only reason should succeed at repo layer (validation moved to service)")
	})
}

// TestTaskRepository_UpdateStatusForced_StoresRejectionReason verifies that rejection reasons are stored in task_history
func TestTaskRepository_UpdateStatusForced_StoresRejectionReason(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	historyRepo := NewTaskHistoryRepository(db)

	// Seed test data
	_, featureID := test.SeedTestData()

	// Create task in ready_for_review status with unique key
	timestamp := time.Now().UnixNano() % 1000
	taskKey := fmt.Sprintf("T-E99-F99-%03d", timestamp)
	priority := 5
	task := &models.Task{
		Key:       taskKey,
		Title:     "Test Rejection Reason Storage",
		Status:    models.TaskStatus("ready_for_review"),
		FeatureID: featureID,
		Priority:  priority,
	}

	err := repo.Create(ctx, task)
	require.NoError(t, err, "Failed to create test task")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}()

	// Update status with rejection reason (backward transition)
	rejectionReason := "Missing error handling on line 67"
	agent := "test-agent"
	notes := "This needs to be fixed"

	// Call UpdateStatusForced with rejection reason
	err = repo.UpdateStatusForced(
		ctx,
		task.ID,
		models.TaskStatus("in_progress"),
		&agent,
		&notes,
		&rejectionReason, // Now passing rejection reason
		nil,              // documentPath
		false,
	)
	require.NoError(t, err, "UpdateStatusForced should succeed")

	// Verify task status was updated
	updatedTask, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err, "Failed to get updated task")
	require.Equal(t, models.TaskStatus("in_progress"), updatedTask.Status, "Task status should be updated")

	// Verify rejection reason was stored in history
	history, err := historyRepo.ListByTask(ctx, task.ID)
	require.NoError(t, err, "Failed to retrieve task history")
	require.NotEmpty(t, history, "History should have at least one entry")

	// Get the most recent history entry (most recent first in list)
	lastEntry := history[len(history)-1]
	require.NotNil(t, lastEntry.OldStatus, "Old status should be present")
	require.Equal(t, string(models.TaskStatus("ready_for_review")), *lastEntry.OldStatus, "Old status should match")
	require.Equal(t, string(models.TaskStatus("in_progress")), lastEntry.NewStatus, "New status should match")

	// THIS IS THE CRITICAL ASSERTION - rejection reason should be stored
	// This will FAIL until we implement Step 1-2
	require.NotNil(t, lastEntry.RejectionReason, "Rejection reason should be stored in history")
	require.Equal(t, rejectionReason, *lastEntry.RejectionReason, "Rejection reason should match what was provided")
	require.NotNil(t, lastEntry.Agent, "Agent should be stored")
	require.Equal(t, agent, *lastEntry.Agent, "Agent should be stored")
	require.NotNil(t, lastEntry.Notes, "Notes should be stored")
	require.Equal(t, notes, *lastEntry.Notes, "Notes should be stored")
}

// TestTaskRepository_GetByIDs tests the batch GetByIDs method
func TestTaskRepository_GetByIDs(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E95-F17-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E95-F17'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E95'")

	// Create dedicated epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E95",
		Title:         "Test Epic for GetByIDs",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)
	}()

	// Create dedicated feature for this test
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E95-F17",
		Title:  "Test Feature for GetByIDs",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID)
	}()

	// Create 3 test tasks
	task1 := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E95-F17-011",
		Title:     "Batch Test Task One",
		Status:    models.TaskStatus("todo"),
		Priority:  5,
	}
	task2 := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E95-F17-012",
		Title:     "Batch Test Task Two",
		Status:    models.TaskStatus("in_progress"),
		Priority:  3,
	}
	task3 := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E95-F17-013",
		Title:     "Batch Test Task Three",
		Status:    models.TaskStatus("todo"),
		Priority:  7,
	}

	for _, task := range []*models.Task{task1, task2, task3} {
		err = repo.Create(ctx, task)
		require.NoError(t, err, "Failed to create test task %s", task.Key)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E95-F17-%'")
	}()

	t.Run("MultipleTasks", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{task1.ID, task2.ID, task3.ID})
		require.NoError(t, err)
		assert.Len(t, tasks, 3)

		// Build a map for easy lookup
		tasksByKey := make(map[string]*models.Task)
		for _, task := range tasks {
			tasksByKey[task.Key] = task
		}
		assert.Contains(t, tasksByKey, "T-E95-F17-011")
		assert.Contains(t, tasksByKey, "T-E95-F17-012")
		assert.Contains(t, tasksByKey, "T-E95-F17-013")
	})

	t.Run("EmptyInput", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{})
		require.NoError(t, err)
		assert.NotNil(t, tasks)
		assert.Len(t, tasks, 0)
	})

	t.Run("NilInput", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, tasks)
		assert.Len(t, tasks, 0)
	})

	t.Run("SomeMissing", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{task1.ID, -999, task3.ID})
		require.NoError(t, err)
		assert.Len(t, tasks, 2)

		tasksByKey := make(map[string]*models.Task)
		for _, task := range tasks {
			tasksByKey[task.Key] = task
		}
		assert.Contains(t, tasksByKey, "T-E95-F17-011")
		assert.Contains(t, tasksByKey, "T-E95-F17-013")
	})

	t.Run("AllMissing", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{-1, -2, -3})
		require.NoError(t, err)
		assert.NotNil(t, tasks)
		assert.Len(t, tasks, 0)
	})

	t.Run("SingleID", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{task2.ID})
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "T-E95-F17-012", tasks[0].Key)
		assert.Equal(t, "Batch Test Task Two", tasks[0].Title)
		assert.Equal(t, models.TaskStatus("in_progress"), tasks[0].Status)
	})
}
