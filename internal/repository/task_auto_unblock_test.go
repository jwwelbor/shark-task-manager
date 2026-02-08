package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAutoUnblockTest creates a clean E97 epic/feature for auto-unblock tests.
// Returns epicID, featureID, and a cleanup function.
func setupAutoUnblockTest(t *testing.T) (int64, int64, func()) {
	t.Helper()
	ctx := context.Background()
	database := test.GetTestDB()

	// Clean up any leftover test data (task_relationships first due to FK)
	_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships WHERE from_task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%') OR to_task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E97-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F01'")

	// Get or create E97 epic
	var epicID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM epics WHERE key = 'E97'").Scan(&epicID)
	if err != nil {
		result, err := database.ExecContext(ctx, `
			INSERT INTO epics (key, title, description, status, priority)
			VALUES ('E97', 'Auto-Unblock Test Epic', 'Test epic for auto-unblock', 'active', 'medium')
		`)
		require.NoError(t, err)
		epicID, _ = result.LastInsertId()
	}

	// Create feature
	featureResult, err := database.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, 'E97-F01', 'Auto-Unblock Test Feature', 'Test feature', 'active')
	`, epicID)
	require.NoError(t, err)
	featureID, _ := featureResult.LastInsertId()

	cleanup := func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships WHERE from_task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%') OR to_task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%')")
		_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%')")
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E97-F01-%'")
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F01'")
	}

	return epicID, featureID, cleanup
}

func TestAutoUnblock_SingleDependency(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// Create T-001 (no deps) and T-002 (depends on T-001)
	task1 := &models.Task{
		FeatureID:   featureID,
		Key:         "T-E97-F01-001",
		Title:       "Prerequisite Task",
		Status:      models.TaskStatus("completed"),
		Priority:    5,
		DependsOn:   nil,
		Description: stringPtr("No dependencies"),
	}
	task2 := &models.Task{
		FeatureID:   featureID,
		Key:         "T-E97-F01-002",
		Title:       "Dependent Task",
		Status:      models.TaskStatus("blocked"),
		Priority:    5,
		DependsOn:   stringPtr(`["T-E97-F01-001"]`),
		Description: stringPtr("Depends on task 1"),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	// Set blocked_reason to dependency pattern (simulate ReopenTaskWithAutoBlock)
	_, err := database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task2.ID)
	require.NoError(t, err)

	// Now auto-unblock: task1 is completed, task2 depends only on task1
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// Verify task2 was unblocked
	assert.Equal(t, []string{"T-E97-F01-002"}, unblocked)

	// Verify DB state
	updated, err := taskRepo.GetByKey(ctx, "T-E97-F01-002")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("todo"), updated.Status)
	assert.Nil(t, updated.BlockedReason)
}

func TestAutoUnblock_MultipleDeps_PartialCompletion(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 completed, T-002 in_progress, T-003 depends on both
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("in_progress"), Priority: 5, Description: stringPtr(""),
	}
	task3 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-003", Title: "Task 3",
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn:   stringPtr(`["T-E97-F01-001", "T-E97-F01-002"]`),
		Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// Set dependency-pattern blocked reason
	_, err := database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task3.ID)
	require.NoError(t, err)

	// Completing task1 should NOT unblock task3 (task2 still in_progress)
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Empty(t, unblocked, "should NOT unblock when not all deps are completed")

	// Verify task3 is still blocked
	t3, err := taskRepo.GetByKey(ctx, "T-E97-F01-003")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("blocked"), t3.Status)
}

func TestAutoUnblock_MultipleDeps_AllCompleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// Both T-001 and T-002 completed, T-003 depends on both
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task3 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-003", Title: "Task 3",
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn:   stringPtr(`["T-E97-F01-001", "T-E97-F01-002"]`),
		Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// Set dependency-pattern blocked reason
	_, err := database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task3.ID)
	require.NoError(t, err)

	// Completing task2 (last dep) SHOULD unblock task3
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-002")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Equal(t, []string{"T-E97-F01-003"}, unblocked)

	// Verify DB state
	t3, err := taskRepo.GetByKey(ctx, "T-E97-F01-003")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("todo"), t3.Status)
	assert.Nil(t, t3.BlockedReason)
}

func TestAutoUnblock_ManualBlockSkipped(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 completed, T-002 blocked manually (not dependency-blocked)
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn:   stringPtr(`["T-E97-F01-001"]`),
		Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	// Set MANUAL blocked reason (not a dependency pattern)
	_, err := database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Waiting on API key from infrastructure team", task2.ID)
	require.NoError(t, err)

	// Auto-unblock should skip task2 because it was manually blocked
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Empty(t, unblocked, "manually blocked task should NOT be auto-unblocked")

	// Verify task2 is still blocked with original reason
	t2, err := taskRepo.GetByKey(ctx, "T-E97-F01-002")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("blocked"), t2.Status)
	assert.NotNil(t, t2.BlockedReason)
	assert.Equal(t, "Waiting on API key from infrastructure team", *t2.BlockedReason)
}

func TestAutoUnblock_NoDependents(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 completed, no dependents
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))

	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Empty(t, unblocked, "no dependents means nothing to unblock")
}

func TestAutoUnblock_NonBlockedDependentSkipped(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 completed, T-002 depends on T-001 but is in_progress (not blocked)
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("in_progress"), Priority: 5,
		DependsOn:   stringPtr(`["T-E97-F01-001"]`),
		Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Empty(t, unblocked, "non-blocked dependents should be skipped")
}

func TestAutoUnblock_HistoryRecorded(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn:   stringPtr(`["T-E97-F01-001"]`),
		Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	_, err := database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task2.ID)
	require.NoError(t, err)

	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	require.Len(t, unblocked, 1)

	// Verify task_history was created
	var historyCount int
	err = database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM task_history WHERE task_id = ? AND old_status = 'blocked' AND new_status = 'todo' AND notes = 'Auto-unblocked: all dependencies satisfied'",
		task2.ID).Scan(&historyCount)
	require.NoError(t, err)
	assert.Equal(t, 1, historyCount, "expected one history record for auto-unblock")
}

func TestAutoUnblock_AutoBlockedPrefixPattern(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn:   stringPtr(`["T-E97-F01-001"]`),
		Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	// Use "Auto-blocked:" prefix pattern (future-proofing)
	_, err := database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Auto-blocked: dependency chain from T-E97-F01-001", task2.ID)
	require.NoError(t, err)

	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Equal(t, []string{"T-E97-F01-002"}, unblocked, "Auto-blocked: prefix should be recognized")
}

func TestAutoUnblock_ViaUpdateStatusForcedWithUnblock(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 in_progress, T-002 blocked (depends on T-001)
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("in_progress"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn:   stringPtr(`["T-E97-F01-001"]`),
		Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	_, err := database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task2.ID)
	require.NoError(t, err)

	// Use the high-level method: complete task1, expect task2 to auto-unblock
	unblockedKeys, err := taskRepo.UpdateStatusForcedWithUnblock(
		ctx, task1.ID, models.TaskStatus("completed"), nil, nil, nil, nil, true,
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"T-E97-F01-002"}, unblockedKeys)

	// Verify task2 is now todo
	t2, err := taskRepo.GetByKey(ctx, "T-E97-F01-002")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("todo"), t2.Status)

	// Verify task1 is completed
	t1, err := taskRepo.GetByKey(ctx, "T-E97-F01-001")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("completed"), t1.Status)
}

func TestAutoUnblock_UpdateStatusForced_NoUnblockForNonCompletionStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 todo, T-002 blocked (depends on T-001)
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("todo"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn:   stringPtr(`["T-E97-F01-001"]`),
		Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	_, err := database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task2.ID)
	require.NoError(t, err)

	// Move task1 to in_progress (not completed) - should NOT trigger auto-unblock
	unblockedKeys, err := taskRepo.UpdateStatusForcedWithUnblock(
		ctx, task1.ID, models.TaskStatus("in_progress"), nil, nil, nil, nil, true,
	)
	require.NoError(t, err)

	assert.Empty(t, unblockedKeys, "non-completion status should not trigger auto-unblock")

	// Verify task2 is still blocked
	t2, err := taskRepo.GetByKey(ctx, "T-E97-F01-002")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("blocked"), t2.Status)
}

// Tests for task_relationships table support

func TestAutoUnblock_TaskRelationships_SingleDependency(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// Create T-001 (completed) and T-002 (blocked, depends on T-001 via task_relationships)
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("blocked"), Priority: 5, Description: stringPtr(""),
		// No depends_on JSON field — dependency is via task_relationships only
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	// Create task_relationship: task2 depends_on task1
	_, err := database.ExecContext(ctx,
		"INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, 'depends_on')",
		task2.ID, task1.ID)
	require.NoError(t, err)

	// Set dependency-pattern blocked reason
	_, err = database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task2.ID)
	require.NoError(t, err)

	// Auto-unblock: task1 is completed, task2 depends only on task1 via task_relationships
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Equal(t, []string{"T-E97-F01-002"}, unblocked)

	// Verify DB state
	updated, err := taskRepo.GetByKey(ctx, "T-E97-F01-002")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("todo"), updated.Status)
}

func TestAutoUnblock_TaskRelationships_MultipleDeps_PartialCompletion(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 completed, T-002 in_progress, T-003 depends on both via task_relationships
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("in_progress"), Priority: 5, Description: stringPtr(""),
	}
	task3 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-003", Title: "Task 3",
		Status: models.TaskStatus("blocked"), Priority: 5, Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// Create relationships: task3 depends_on task1 and task2
	_, err := database.ExecContext(ctx,
		"INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, 'depends_on')",
		task3.ID, task1.ID)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx,
		"INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, 'depends_on')",
		task3.ID, task2.ID)
	require.NoError(t, err)

	// Set dependency-pattern blocked reason
	_, err = database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task3.ID)
	require.NoError(t, err)

	// Completing task1 should NOT unblock task3 (task2 still in_progress)
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Empty(t, unblocked, "should NOT unblock when not all task_relationships deps are completed")
}

func TestAutoUnblock_TaskRelationships_AllCompleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// Both T-001 and T-002 completed, T-003 depends on both via task_relationships
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task3 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-003", Title: "Task 3",
		Status: models.TaskStatus("blocked"), Priority: 5, Description: stringPtr(""),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// Create relationships: task3 depends_on task1 and task2
	_, err := database.ExecContext(ctx,
		"INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, 'depends_on')",
		task3.ID, task1.ID)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx,
		"INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, 'depends_on')",
		task3.ID, task2.ID)
	require.NoError(t, err)

	// Set dependency-pattern blocked reason
	_, err = database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task3.ID)
	require.NoError(t, err)

	// Completing task2 (last dep) SHOULD unblock task3
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-002")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Equal(t, []string{"T-E97-F01-003"}, unblocked)

	// Verify DB state
	t3, err := taskRepo.GetByKey(ctx, "T-E97-F01-003")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("todo"), t3.Status)
}

func TestAutoUnblock_MixedDependencies_LegacyAndRelationships(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 completed, T-002 completed
	// T-003 depends on T-001 via depends_on JSON AND on T-002 via task_relationships
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task3 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-003", Title: "Task 3",
		Status: models.TaskStatus("blocked"), Priority: 5, Description: stringPtr(""),
		DependsOn: stringPtr(`["T-E97-F01-001"]`), // Legacy depends_on
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// Also add task_relationships: task3 depends_on task2
	_, err := database.ExecContext(ctx,
		"INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, 'depends_on')",
		task3.ID, task2.ID)
	require.NoError(t, err)

	// Set dependency-pattern blocked reason
	_, err = database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task3.ID)
	require.NoError(t, err)

	// Both dependencies satisfied — should unblock
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-002")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Equal(t, []string{"T-E97-F01-003"}, unblocked)

	t3, err := taskRepo.GetByKey(ctx, "T-E97-F01-003")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("todo"), t3.Status)
}

func TestAutoUnblock_MixedDependencies_PartialSatisfied(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 completed, T-002 in_progress
	// T-003 depends on T-001 via depends_on JSON AND on T-002 via task_relationships
	task1 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-001", Title: "Task 1",
		Status: models.TaskStatus("completed"), Priority: 5, Description: stringPtr(""),
	}
	task2 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-002", Title: "Task 2",
		Status: models.TaskStatus("in_progress"), Priority: 5, Description: stringPtr(""),
	}
	task3 := &models.Task{
		FeatureID: featureID, Key: "T-E97-F01-003", Title: "Task 3",
		Status: models.TaskStatus("blocked"), Priority: 5, Description: stringPtr(""),
		DependsOn: stringPtr(`["T-E97-F01-001"]`), // Legacy dep satisfied
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// task_relationships dep NOT satisfied (task2 in_progress)
	_, err := database.ExecContext(ctx,
		"INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, 'depends_on')",
		task3.ID, task2.ID)
	require.NoError(t, err)

	// Set dependency-pattern blocked reason
	_, err = database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task3.ID)
	require.NoError(t, err)

	// Legacy dep satisfied but relationship dep not — should NOT unblock
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Empty(t, unblocked, "should NOT unblock when task_relationships dep is unsatisfied")

	t3, err := taskRepo.GetByKey(ctx, "T-E97-F01-003")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("blocked"), t3.Status)
}
