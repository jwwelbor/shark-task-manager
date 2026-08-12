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

	// Clean up any leftover test data (entity_relationships first due to FK)
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE (from_entity_type = 'task' AND from_entity_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%')) OR (to_entity_type = 'task' AND to_entity_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%'))")
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
		_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE (from_entity_type = 'task' AND from_entity_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%')) OR (to_entity_type = 'task' AND to_entity_id IN (SELECT id FROM tasks WHERE key LIKE 'T-E97-F01-%'))")
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
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001",
		Title: "Prerequisite Task",

		Description: stringPtr("No dependencies")}, FeatureID: featureID,

		Status:    models.TaskStatus("completed"),
		Priority:  5,
		DependsOn: nil,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002",
		Title: "Dependent Task",

		Description: stringPtr("Depends on task 1")}, FeatureID: featureID,

		Status:    models.TaskStatus("blocked"),
		Priority:  5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`),
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
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

func TestTaskStatusPolicy_UsesConfiguredDependencyTargetsAndTimestamps(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	prerequisite := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Prerequisite", Description: stringPtr("")}, FeatureID: featureID, Status: "queued", Priority: 5}
	dependent := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Dependent", Description: stringPtr("")}, FeatureID: featureID, Status: "stalled", Priority: 5, DependsOn: stringPtr(`["T-E97-F01-001"]`)}
	require.NoError(t, repo.Create(ctx, prerequisite))
	require.NoError(t, repo.Create(ctx, dependent))
	_, err := database.ExecContext(ctx, "UPDATE tasks SET blocked_reason = ? WHERE id = ?", "Prerequisite task T-E97-F01-001 was reopened", dependent.ID)
	require.NoError(t, err)

	_, err = repo.StatusUpdateRaw(ctx, models.StatusUpdateParams{
		TaskID: prerequisite.ID, TaskKey: prerequisite.Key, OldStatus: "queued", NewStatus: "shipped",
		TerminalStatuses: []string{"shipped"}, BlockedStatuses: []string{"stalled"}, ExecutionStatuses: []string{"working"}, UnblockedStatus: "queued",
	})
	require.NoError(t, err)
	updatedDependent, err := repo.GetByID(ctx, dependent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("queued"), updatedDependent.Status)

	_, err = repo.StatusUpdateRaw(ctx, models.StatusUpdateParams{
		TaskID: prerequisite.ID, TaskKey: prerequisite.Key, OldStatus: "shipped", NewStatus: "working",
		TerminalStatuses: []string{"shipped"}, BlockedStatuses: []string{"stalled"}, ExecutionStatuses: []string{"working"}, UnblockedStatus: "queued",
	})
	require.NoError(t, err)
	updatedPrerequisite, err := repo.GetByID(ctx, prerequisite.ID)
	require.NoError(t, err)
	assert.True(t, updatedPrerequisite.StartedAt.Valid)

	require.NoError(t, repo.ReopenTaskWithAutoBlockWithPolicy(ctx, prerequisite.ID, nil, nil, models.TaskDependencyStatusPolicy{
		TerminalStatuses: []string{"shipped"}, BlockedStatuses: []string{"stalled"}, ReopenStatus: "working", UnblockedStatus: "queued",
	}))
	updatedPrerequisite, err = repo.GetByID(ctx, prerequisite.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("working"), updatedPrerequisite.Status)
	updatedDependent, err = repo.GetByID(ctx, dependent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("stalled"), updatedDependent.Status)
}

func TestAutoUnblock_MultipleDeps_PartialCompletion(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// T-001 completed, T-002 in_progress, T-003 depends on both
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("in_progress"), Priority: 5,
	}
	task3 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-003", Title: "Task 3",

		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001", "T-E97-F01-002"]`),
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
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
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task3 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-003", Title: "Task 3",

		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001", "T-E97-F01-002"]`),
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-002", nil)
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
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",

		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`),
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
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
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}

	require.NoError(t, taskRepo.Create(ctx, task1))

	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
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
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",

		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("in_progress"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`),
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
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

	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",

		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`),
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
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

	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",

		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`),
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
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
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("in_progress"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",

		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`),
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
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("todo"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",

		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`),
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

// Tests for entity_relationships table support

func TestAutoUnblock_TaskRelationships_SingleDependency(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// Create T-001 (completed) and T-002 (blocked, depends on T-001 via entity_relationships)
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		// No depends_on JSON field — dependency is via entity_relationships only
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))

	// Create entity_relationship: task2 depends_on task1
	_, err := database.ExecContext(ctx,
		"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('task', ?, 'task', ?, 'depends_on')",
		task2.ID, task1.ID)
	require.NoError(t, err)

	// Set dependency-pattern blocked reason
	_, err = database.ExecContext(ctx,
		"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
		"Prerequisite task T-E97-F01-001 was reopened", task2.ID)
	require.NoError(t, err)

	// Auto-unblock: task1 is completed, task2 depends only on task1 via entity_relationships
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
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

	// T-001 completed, T-002 in_progress, T-003 depends on both via entity_relationships
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("in_progress"), Priority: 5,
	}
	task3 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-003", Title: "Task 3",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// Create relationships: task3 depends_on task1 and task2
	_, err := database.ExecContext(ctx,
		"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('task', ?, 'task', ?, 'depends_on')",
		task3.ID, task1.ID)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx,
		"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('task', ?, 'task', ?, 'depends_on')",
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Empty(t, unblocked, "should NOT unblock when not all entity_relationships deps are completed")
}

func TestAutoUnblock_TaskRelationships_AllCompleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	_, featureID, cleanup := setupAutoUnblockTest(t)
	defer cleanup()

	// Both T-001 and T-002 completed, T-003 depends on both via entity_relationships
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task3 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-003", Title: "Task 3",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// Create relationships: task3 depends_on task1 and task2
	_, err := database.ExecContext(ctx,
		"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('task', ?, 'task', ?, 'depends_on')",
		task3.ID, task1.ID)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx,
		"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('task', ?, 'task', ?, 'depends_on')",
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-002", nil)
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
	// T-003 depends on T-001 via depends_on JSON AND on T-002 via entity_relationships
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task3 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-003", Title: "Task 3",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`), // Legacy depends_on
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// Also add entity_relationships: task3 depends_on task2
	_, err := database.ExecContext(ctx,
		"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('task', ?, 'task', ?, 'depends_on')",
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-002", nil)
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
	// T-003 depends on T-001 via depends_on JSON AND on T-002 via entity_relationships
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task 1",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("completed"), Priority: 5,
	}
	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task 2",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("in_progress"), Priority: 5,
	}
	task3 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-003", Title: "Task 3",
		Description: stringPtr("")}, FeatureID: featureID,
		Status: models.TaskStatus("blocked"), Priority: 5,
		DependsOn: stringPtr(`["T-E97-F01-001"]`), // Legacy dep satisfied
	}

	require.NoError(t, taskRepo.Create(ctx, task1))
	require.NoError(t, taskRepo.Create(ctx, task2))
	require.NoError(t, taskRepo.Create(ctx, task3))

	// entity_relationships dep NOT satisfied (task2 in_progress)
	_, err := database.ExecContext(ctx,
		"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('task', ?, 'task', ?, 'depends_on')",
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

	unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", nil)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.Empty(t, unblocked, "should NOT unblock when entity_relationships dep is unsatisfied")

	t3, err := taskRepo.GetByKey(ctx, "T-E97-F01-003")
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("blocked"), t3.Status)
}

// TestAutoUnblock_UsesCallerTerminalStatuses proves dependency satisfaction is
// decided by the caller-supplied (service-resolved) terminal-status list rather
// than a hardcoded completed/archived pair. Both subtests invert under the
// hardcoded set, so reintroducing the literal turns them red.
func TestAutoUnblock_UsesCallerTerminalStatuses(t *testing.T) {
	// shippedOnly is a custom workflow's terminal set: "shipped" is terminal and
	// "completed" is not.
	shippedOnly := []string{"shipped"}

	t.Run("custom terminal status satisfies the dependency", func(t *testing.T) {
		ctx := context.Background()
		database := test.GetTestDB()
		db := NewDB(database)
		taskRepo := NewTaskRepository(db)

		_, featureID, cleanup := setupAutoUnblockTest(t)
		defer cleanup()

		dep := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Prerequisite",
			Description: stringPtr("")}, FeatureID: featureID,
			Status: models.TaskStatus("shipped"), Priority: 5,
		}
		dependent := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Dependent",
			Description: stringPtr("")}, FeatureID: featureID,
			Status: models.TaskStatus("blocked"), Priority: 5,
			DependsOn: stringPtr(`["T-E97-F01-001"]`),
		}
		require.NoError(t, taskRepo.Create(ctx, dep))
		require.NoError(t, taskRepo.Create(ctx, dependent))
		_, err := database.ExecContext(ctx,
			"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
			"Prerequisite task T-E97-F01-001 was reopened", dependent.ID)
		require.NoError(t, err)

		tx, err := db.BeginTxContext(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", shippedOnly)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())

		assert.Equal(t, []string{"T-E97-F01-002"}, unblocked,
			`"shipped" is terminal for this caller, so the dependency is satisfied; a hardcoded completed/archived check would leave it blocked`)
	})

	t.Run("status outside the caller terminal set does not satisfy", func(t *testing.T) {
		ctx := context.Background()
		database := test.GetTestDB()
		db := NewDB(database)
		taskRepo := NewTaskRepository(db)

		_, featureID, cleanup := setupAutoUnblockTest(t)
		defer cleanup()

		dep := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Prerequisite",
			Description: stringPtr("")}, FeatureID: featureID,
			Status: models.TaskStatus("completed"), Priority: 5,
		}
		dependent := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Dependent",
			Description: stringPtr("")}, FeatureID: featureID,
			Status: models.TaskStatus("blocked"), Priority: 5,
			DependsOn: stringPtr(`["T-E97-F01-001"]`),
		}
		require.NoError(t, taskRepo.Create(ctx, dep))
		require.NoError(t, taskRepo.Create(ctx, dependent))
		_, err := database.ExecContext(ctx,
			"UPDATE tasks SET blocked_reason = ?, blocked_at = CURRENT_TIMESTAMP WHERE id = ?",
			"Prerequisite task T-E97-F01-001 was reopened", dependent.ID)
		require.NoError(t, err)

		tx, err := db.BeginTxContext(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		unblocked, err := taskRepo.AutoUnblockDependents(ctx, tx, "T-E97-F01-001", shippedOnly)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())

		assert.Empty(t, unblocked,
			`"completed" is NOT terminal for this caller, so the dependency is unsatisfied; a hardcoded completed/archived check would wrongly unblock`)
	})
}
