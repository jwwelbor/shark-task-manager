package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskHistoryRepository_ListWithFilters tests filtering history records
func TestTaskHistoryRepository_ListWithFilters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	historyRepo := NewTaskHistoryRepository(db)
	taskRepo := NewTaskRepository(db)

	// Clean up existing test data - ensure clean state
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE agent LIKE 'test-agent%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E99-F99')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E99')")

	// Seed test data
	_, featureID := test.SeedTestData()
	require.NotZero(t, featureID, "featureID should not be zero after seeding")

	// Create test tasks
	agentBackend := models.AgentTypeBackend
	agentFrontend := models.AgentTypeFrontend
	filePath1 := "docs/test/task1.md"
	filePath2 := "docs/test/task2.md"
	dependsOn := "[]"

	task1 := &models.Task{
		FeatureID: featureID,
		Key:       "T-E99-F99-901",
		Title:     "History Test Task 1",
		Status:    models.TaskStatusTodo,
		AgentType: &agentBackend,
		Priority:  5,
		DependsOn: &dependsOn,
		FilePath:  &filePath1,
	}
	err := taskRepo.Create(ctx, task1)
	require.NoError(t, err)

	task2 := &models.Task{
		FeatureID: featureID,
		Key:       "T-E99-F99-902",
		Title:     "History Test Task 2",
		Status:    models.TaskStatusTodo,
		AgentType: &agentFrontend,
		Priority:  5,
		DependsOn: &dependsOn,
		FilePath:  &filePath2,
	}
	err = taskRepo.Create(ctx, task2)
	require.NoError(t, err)

	// Create history records with different attributes
	agent1 := "test-agent-1"
	agent2 := "test-agent-2"
	notes1 := "First transition"
	notes2 := "Second transition"

	// Task 1 history: todo -> in_progress -> completed
	history1 := &models.TaskHistory{
		TaskID:    task1.ID,
		OldStatus: nil,
		NewStatus: string(models.TaskStatusTodo),
		Agent:     &agent1,
		Notes:     &notes1,
	}
	err = historyRepo.Create(ctx, history1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	oldStatus1 := string(models.TaskStatusTodo)
	history2 := &models.TaskHistory{
		TaskID:    task1.ID,
		OldStatus: &oldStatus1,
		NewStatus: string(models.TaskStatusInProgress),
		Agent:     &agent1,
		Notes:     &notes2,
	}
	err = historyRepo.Create(ctx, history2)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	oldStatus2 := string(models.TaskStatusInProgress)
	history3 := &models.TaskHistory{
		TaskID:    task1.ID,
		OldStatus: &oldStatus2,
		NewStatus: string(models.TaskStatusCompleted),
		Agent:     &agent1,
		Notes:     nil,
	}
	err = historyRepo.Create(ctx, history3)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	// Task 2 history: todo -> in_progress
	history4 := &models.TaskHistory{
		TaskID:    task2.ID,
		OldStatus: nil,
		NewStatus: string(models.TaskStatusTodo),
		Agent:     &agent2,
		Notes:     nil,
	}
	err = historyRepo.Create(ctx, history4)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	oldStatus3 := string(models.TaskStatusTodo)
	history5 := &models.TaskHistory{
		TaskID:    task2.ID,
		OldStatus: &oldStatus3,
		NewStatus: string(models.TaskStatusInProgress),
		Agent:     &agent2,
		Notes:     nil,
	}
	err = historyRepo.Create(ctx, history5)
	require.NoError(t, err)

	// Test 1: No filters - should return all history records (with limit)
	t.Run("NoFilters", func(t *testing.T) {
		filters := HistoryFilters{
			Limit: 50,
		}
		histories, err := historyRepo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(histories), 5, "Should have at least 5 history records")
	})

	// Test 2: Filter by agent
	t.Run("FilterByAgent", func(t *testing.T) {
		filters := HistoryFilters{
			Agent: &agent1,
			Limit: 50,
		}
		histories, err := historyRepo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, 3, len(histories), "Should have 3 records for agent1")
		for _, h := range histories {
			require.NotNil(t, h.Agent)
			assert.Equal(t, agent1, *h.Agent)
		}
	})

	// Test 3: Filter by since timestamp
	t.Run("FilterBySince", func(t *testing.T) {
		// Get timestamp of history2
		since := history2.Timestamp
		filters := HistoryFilters{
			Since: &since,
			Limit: 50,
		}
		histories, err := historyRepo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		// Should have history2, history3, history4, history5 (and possibly others from other tests)
		assert.GreaterOrEqual(t, len(histories), 4, "Should have at least 4 records since timestamp")
		for _, h := range histories {
			assert.True(t, h.Timestamp.After(since) || h.Timestamp.Equal(since),
				"All timestamps should be >= since")
		}
	})

	// Test 4: Filter by epic
	t.Run("FilterByEpic", func(t *testing.T) {
		epicKey := "E99"
		filters := HistoryFilters{
			EpicKey: &epicKey,
			Limit:   50,
		}
		histories, err := historyRepo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, 5, len(histories), "Should have 5 records for E99")
	})

	// Test 5: Filter by status change (old -> new)
	t.Run("FilterByStatusChange", func(t *testing.T) {
		oldSt := string(models.TaskStatusTodo)
		newSt := string(models.TaskStatusInProgress)
		filters := HistoryFilters{
			OldStatus: &oldSt,
			NewStatus: &newSt,
			Limit:     50,
		}
		histories, err := historyRepo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, 2, len(histories), "Should have 2 todo->in_progress transitions")
		for _, h := range histories {
			require.NotNil(t, h.OldStatus)
			assert.Equal(t, oldSt, *h.OldStatus)
			assert.Equal(t, newSt, h.NewStatus)
		}
	})

	// Test 6: Combined filters (agent + epic)
	t.Run("CombinedFilters", func(t *testing.T) {
		epicKey := "E99"
		filters := HistoryFilters{
			Agent:   &agent1,
			EpicKey: &epicKey,
			Limit:   50,
		}
		histories, err := historyRepo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, 3, len(histories), "Should have 3 records for agent1 in E99")
	})

	// Test 7: Pagination with offset
	t.Run("Pagination", func(t *testing.T) {
		filters1 := HistoryFilters{
			Limit: 2,
		}
		page1, err := historyRepo.ListWithFilters(ctx, filters1)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page1), 2, "First page should have at most 2 records")

		filters2 := HistoryFilters{
			Offset: 2,
			Limit:  2,
		}
		page2, err := historyRepo.ListWithFilters(ctx, filters2)
		require.NoError(t, err)

		// Ensure no overlap between pages (different IDs)
		if len(page1) > 0 && len(page2) > 0 {
			assert.NotEqual(t, page1[0].ID, page2[0].ID, "Pages should have different records")
		}
	})

	// Test 8: Default limit (50)
	t.Run("DefaultLimit", func(t *testing.T) {
		filters := HistoryFilters{
			// No limit specified, should default to 50
		}
		histories, err := historyRepo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(histories), 50, "Should respect default limit of 50")
	})

	// Cleanup
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id IN (?, ?)", task1.ID, task2.ID) }()
}

// TestTaskHistoryRepository_ListWithFilters_EmptyResults tests that empty results are handled correctly
func TestTaskHistoryRepository_ListWithFilters_EmptyResults(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskHistoryRepository(db)

	// Filter by non-existent agent
	nonExistentAgent := "non-existent-agent-xyz"
	filters := HistoryFilters{
		Agent: &nonExistentAgent,
		Limit: 50,
	}

	histories, err := repo.ListWithFilters(ctx, filters)
	require.NoError(t, err)
	assert.Empty(t, histories, "Should return empty slice for non-existent agent")
}

// TestGetHistoryByTaskKey tests retrieving task history by task key
func TestGetHistoryByTaskKey(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	historyRepo := NewTaskHistoryRepository(db)
	taskRepo := NewTaskRepository(db)

	// Clean up existing test data - ensure clean state
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E99-%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E99-F99')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E99')")

	// Seed test data
	epicID, featureID := test.SeedTestData()
	require.NotZero(t, epicID, "epicID should not be zero after seeding")
	require.NotZero(t, featureID, "featureID should not be zero after seeding")

	// Get a test task
	task, err := taskRepo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create some history records for this task
	agent1 := "test-agent-1"
	notes1 := "Started working on the task"
	history1 := &models.TaskHistory{
		TaskID:    task.ID,
		OldStatus: nil,
		NewStatus: string(models.TaskStatusInProgress),
		Agent:     &agent1,
		Notes:     &notes1,
	}

	err = historyRepo.Create(ctx, history1)
	if err != nil {
		t.Fatalf("Failed to create history record 1: %v", err)
	}

	// Wait a moment to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Create second history record
	oldStatus := string(models.TaskStatusInProgress)
	agent2 := "test-agent-2"
	notes2 := "Completed the implementation"
	history2 := &models.TaskHistory{
		TaskID:    task.ID,
		OldStatus: &oldStatus,
		NewStatus: string(models.TaskStatusReadyForReview),
		Agent:     &agent2,
		Notes:     &notes2,
	}

	err = historyRepo.Create(ctx, history2)
	if err != nil {
		t.Fatalf("Failed to create history record 2: %v", err)
	}

	// Test GetHistoryByTaskKey - this method doesn't exist yet (RED phase)
	histories, err := historyRepo.GetHistoryByTaskKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get history by task key: %v", err)
	}

	// Verify we got both records
	assert.Equal(t, 2, len(histories), "Expected 2 history records")

	// Verify chronological order (oldest first for timeline display)
	assert.Equal(t, string(models.TaskStatusInProgress), histories[0].NewStatus)
	assert.Equal(t, string(models.TaskStatusReadyForReview), histories[1].NewStatus)

	// Verify agents and notes
	assert.NotNil(t, histories[0].Agent)
	assert.Equal(t, agent1, *histories[0].Agent)
	assert.NotNil(t, histories[0].Notes)
	assert.Equal(t, notes1, *histories[0].Notes)

	assert.NotNil(t, histories[1].Agent)
	assert.Equal(t, agent2, *histories[1].Agent)
	assert.NotNil(t, histories[1].Notes)
	assert.Equal(t, notes2, *histories[1].Notes)

	// Cleanup
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id = ?", task.ID) }()

	// Seed test data again for next test
	_ = epicID
	_ = featureID
}

// TestGetHistoryByTaskKeyNotFound tests getting history for non-existent task
func TestGetHistoryByTaskKeyNotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	historyRepo := NewTaskHistoryRepository(db)

	// Try to get history for non-existent task
	histories, err := historyRepo.GetHistoryByTaskKey(ctx, "T-E99-F99-999")

	// Should not error, just return empty slice
	assert.NoError(t, err)
	assert.Empty(t, histories)
}

// TestGetHistoryByTaskKeyEmptyHistory tests getting history for task with no history
func TestGetHistoryByTaskKeyEmptyHistory(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	historyRepo := NewTaskHistoryRepository(db)
	taskRepo := NewTaskRepository(db)

	// Seed test data first
	test.SeedTestData()

	// Clean up existing test data (history only, not the task)
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id IN (SELECT id FROM tasks WHERE key = 'T-E99-F99-002')")

	// Get the existing test task
	task, err := taskRepo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Get history - should be empty
	histories, err := historyRepo.GetHistoryByTaskKey(ctx, "T-E99-F99-002")
	assert.NoError(t, err)
	assert.Empty(t, histories)

	// No cleanup needed - using seeded data
	_ = task
}
