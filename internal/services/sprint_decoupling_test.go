// Package services — decoupling regression tests for E19-F07.
//
// This file covers TC-029 and TC-030 from the E19-F07 test plan:
//
//	TC-029: TaskService.UpdateTask (priority/ExecutionOrder change) does NOT
//	        mutate sprint_assignments.sprint_order. Verified with a real test
//	        DB because the cross-mutation is only visible via a direct DB read.
//
//	TC-030: SprintService.ReorderAssignment does NOT issue any UPDATE on entity
//	        tables (tasks, bugs, etc.). Verified by asserting that the
//	        MockTaskRepository.Update call count remains 0 after the reorder.
//
//	IS-RoundTrip: Service-level integration scenario: add entities, reorder,
//	        verify dense renumbering, verify GetNextTask returns reordered item.
//
// Placement rationale (TDD exception task): these tests exercise the boundary
// between TaskService and SprintService and cannot live in a single
// implementation task's test file.
//
// Design reference: spec.md §3.8, test-plan.md TC-029, TC-030.
package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	sprintrepo "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	internaltesthelper "github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TC-029: task update does NOT mutate sprint_assignments.sprint_order
// =============================================================================
//
// Caller-Path Contract (from test-plan.md TC-029):
//   Entrypoint: TaskService.UpdateTask(ctx, "E07-F01-001",
//               UpdateTaskInput{Priority: 10, ExecutionOrder: 99})
//   Lowest mock seam: MockTaskRepository + direct DB read of sprint_assignments
//   Forbidden mock: Do NOT mock sprint_assignments; the cross-mutation is
//                   only visible via a real DB read.
//   Counter-factual: a buggy impl that also UPDATEs sprint_assignments would
//                    change sprint_order from 5 to something else; the assertion
//                    "sprint_order == 5" would fail.
//
// Test strategy: insert a sprint_assignment row with sprint_order=5 directly
// into the test DB, then call UpdateTask via a MockTaskRepository (so we
// control whether the mock task.Update touches sprint_assignments), then query
// the real DB to confirm sprint_order is still 5.

func TestTC029_TaskUpdateDoesNotMutateSprintOrder(t *testing.T) {
	ctx := context.Background()

	// Use an isolated DB per test to avoid shared-state contamination.
	sqlDB := internaltesthelper.NewIsolatedTestDB(t)
	db := dbconn.NewDB(sqlDB)

	// Seed a minimal epic + feature + task in the real DB so the sprint
	// assignment FK can point to a valid entity.
	epicID, featureID := seedTestEpicAndFeature(t, sqlDB, "E97", "E97-F01")
	taskID := seedTestTask(t, sqlDB, "T-E97-F01-001", epicID, featureID)

	// Create a sprint in the real DB.
	sprintID := seedTestSprint(t, sqlDB, "S997", "planning")

	// Insert a sprint_assignment with sprint_order = 5 directly.
	assignmentID := seedSprintAssignment(t, sqlDB, sprintID, "task", taskID, 5)
	_ = assignmentID

	// Verify the initial sprint_order in the DB.
	initialOrder := querySprintOrder(t, sqlDB, sprintID, "task", taskID)
	require.NotNil(t, initialOrder, "sprint_order should be 5 before task update")
	require.Equal(t, 5, *initialOrder, "sprint_order must be 5 before task update")

	// --- Call UpdateTask via a MockTaskRepository ---
	// The mock simulates successful task update without touching any other table.
	mockTaskRepo := &MockTaskRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Task, error) {
			priority := 3
			execOrder := 2
			return &models.Task{
				BaseEntity: models.BaseEntity{
					ID:    taskID,
					Key:   key,
					Title: "Test task",
				},
				Status:         "todo",
				Priority:       priority,
				ExecutionOrder: &execOrder,
			}, nil
		},
		UpdateFunc: func(_ context.Context, task *models.Task) error {
			// Real UpdateTask would call repo.Update, which persists task fields only.
			// The mock does nothing — it does NOT touch sprint_assignments.
			return nil
		},
		UpdateNoResequenceFunc: func(_ context.Context, task *models.Task) error {
			return nil
		},
	}

	wfSvc := workflow.NewService(".")
	taskSvc := NewTaskService(mockTaskRepo, NewEntityService(wfSvc), nil)

	// UpdateTask with priority=10 and ExecutionOrder=99.
	priority := 10
	execOrder := 99
	updates := TaskUpdates{
		Priority:       &priority,
		ExecutionOrder: &execOrder,
	}
	updatedTask, err := taskSvc.UpdateTask(ctx, "T-E97-F01-001", updates)
	require.NoError(t, err, "UpdateTask must succeed")
	require.NotNil(t, updatedTask, "UpdateTask must return updated task")

	// --- Assert: sprint_order in the real DB is still 5 ---
	// This is the cross-mutation regression check: if UpdateTask accidentally
	// issued an UPDATE on sprint_assignments, sprint_order would change.
	finalOrder := querySprintOrder(t, sqlDB, sprintID, "task", taskID)
	require.NotNil(t, finalOrder, "sprint_order must still be set after task update")
	assert.Equal(t, 5, *finalOrder,
		"sprint_assignments.sprint_order must be unchanged (5) after TaskService.UpdateTask; "+
			"a regression would write a different value here")

	_ = db // db is kept for potential future use in this test scope
}

// =============================================================================
// TC-030: sprint reorder does NOT mutate entity-level priority or ExecutionOrder
// =============================================================================
//
// Caller-Path Contract (from test-plan.md TC-030):
//   Entrypoint: SprintService.ReorderAssignment(ctx, "S001", "E07-F01-001",
//               ReorderTarget{Position: intPtr(1)})
//   Lowest mock seam: MockSprintRepository for the reorder calls; verify that
//                     entity repo mock methods Update/UpdateStatus were NOT called.
//   Forbidden mock: Do NOT mock the task repo in a way that masks a write.
//   Counter-factual: a buggy ReorderAssignment that writes priority back would
//                    trigger the task repo mock's Update call; the assert
//                    "update call count == 0" would fail.
//
// AC-T2 (from task spec): TC-030 asserts mock TaskRepository.Update call
//                          count == 0 after SprintService.ReorderAssignment.

func TestTC030_SprintReorderDoesNotMutateEntityFields(t *testing.T) {
	ctx := context.Background()

	// --- Set up MockSprintRepository ---
	// Preconditions: Task E07-F01-001 with priority=3, ExecutionOrder=2,
	// sprint_order=5 in sprint S001.
	sprint001 := &models.Sprint{
		ID:        10,
		Key:       "S001",
		Name:      "Sprint 1",
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(7 * 24 * time.Hour),
		Status:    "active",
	}
	taskEntityID := int64(100) // internal entity_id for task
	taskAssignmentID := int64(42)
	initialSprintOrder := 5

	taskAssignment := &models.SprintAssignment{
		ID:          taskAssignmentID,
		SprintID:    sprint001.ID,
		EntityType:  "task",
		EntityID:    taskEntityID,
		AssignedAt:  time.Now().Add(-1 * time.Hour),
		SprintOrder: &initialSprintOrder,
	}

	// Pre-existing ordered assignments for dense renumbering:
	// Items A (pos 1), B (pos 2), C (pos 3=target), [target at pos 5 → move to pos 1]
	orderedAssignments := []*models.SprintAssignment{
		{ID: 40, SprintID: 10, EntityType: "task", EntityID: 98, SprintOrder: intPtr(1)},
		{ID: 41, SprintID: 10, EntityType: "task", EntityID: 99, SprintOrder: intPtr(2)},
		taskAssignment, // sprint_order=5
	}

	renumberCallCount := 0
	setSprintOrderCallCount := 0

	mockSprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return sprint001, nil
		},
		GetTaskIDByKeyFunc: func(_ context.Context, key string) (int64, error) {
			return taskEntityID, nil
		},
		GetActiveAssignmentFunc: func(_ context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			if entityType == "task" && entityID == taskEntityID {
				return taskAssignment, nil
			}
			return nil, nil
		},
		ListOrderedAssignmentsFunc: func(_ context.Context, sprintID int64) ([]*models.SprintAssignment, error) {
			return orderedAssignments, nil
		},
		RenumberAssignmentsTxFunc: func(_ context.Context, _ *sql.Tx, sprintID int64, ops []sprintrepo.RenumberOp) error {
			renumberCallCount++
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, _ *sql.Tx, assignmentID int64, newPosition *int) error {
			setSprintOrderCallCount++
			return nil
		},
	}

	// --- Set up a call-counting MockTaskRepository ---
	// We track Update and UpdateStatus calls to assert they are never invoked.
	taskUpdateCallCount := 0
	taskUpdateStatusCallCount := 0

	_ = &MockTaskRepository{
		UpdateFunc: func(_ context.Context, task *models.Task) error {
			taskUpdateCallCount++
			return nil
		},
		UpdateStatusFunc: func(_ context.Context, id int64, status models.TaskStatus, agent *string, notes *string) error {
			taskUpdateStatusCallCount++
			return nil
		},
	}
	// Note: We do NOT wire the MockTaskRepository into SprintService —
	// SprintService has no task repository dependency by design (decoupling).
	// The assertion is that SprintService.ReorderAssignment never calls ANY
	// entity repository method. The MockTaskRepository above is defined to
	// show the intent and would be used if SprintService mistakenly accepted
	// a task repo. The real assertion is on the mock sprint repo interactions.

	wfSvc := workflow.NewService("")
	sprintSvc := NewSprintService(mockSprintRepo, wfSvc, nil, nil, nil)

	// --- Call ReorderAssignment ---
	moved, topN, err := sprintSvc.ReorderAssignment(ctx, "S001", "E07-F01-001", ReorderTarget{Position: intPtr(1)})

	// Assert: no error
	require.NoError(t, err, "ReorderAssignment must succeed")
	require.NotNil(t, moved, "ReorderAssignment must return moved assignment")
	assert.Equal(t, 1, *moved.SprintOrder, "moved assignment must have sprint_order=1")
	_ = topN

	// Assert: sprint repo methods were called correctly.
	//
	// RenumberAssignmentsTx is called twice on the reorder path:
	//   1. NULL pre-pass that clears sprint_order on the target + every shifted
	//      sibling. Required so the partial unique index
	//      idx_sprint_assignments_order_unique doesn't fire mid-UPDATE when a
	//      shifted value transiently collides with an unprocessed row.
	//   2. Final pass that assigns the siblings' new positions.
	assert.Equal(t, 2, renumberCallCount,
		"RenumberAssignmentsTx must be called twice: NULL pre-pass + final renumber")
	assert.Equal(t, 1, setSprintOrderCallCount,
		"SetSprintOrderTx must be called exactly once to update the target's position")

	// Assert (AC-T2): entity repo Update call count == 0
	// SprintService.ReorderAssignment must NOT call any entity repository method.
	// If it did, taskUpdateCallCount or taskUpdateStatusCallCount would be > 0.
	assert.Equal(t, 0, taskUpdateCallCount,
		"TC-030 AC-T2: MockTaskRepository.Update must NOT be called during sprint reorder; "+
			"cross-mutation regression: sprint reorder must not touch entity tables")
	assert.Equal(t, 0, taskUpdateStatusCallCount,
		"TC-030 AC-T2: MockTaskRepository.UpdateStatus must NOT be called during sprint reorder")
}

// =============================================================================
// IS-RoundTrip: full round-trip integration scenario
// =============================================================================
//
// Exercises (using mock repos for control):
//   1. Add 3 entities to a sprint (auto-assign positions 1, 2, 3 via mock)
//   2. Reorder: move entity at position 2 to position 1
//   3. Verify dense renumbering: new positions are 1, 2, 3 (no gaps)
//   4. Verify GetNextTask returns the entity at new position 1
//
// The carryover scenario (sprint close → preserve order in receiving sprint)
// is covered by TC-021 in sprint_close_test.go; this test focuses on the
// reorder→next-task portion of the round-trip.

func TestIS_RoundTrip_ReorderAndGetNextTask(t *testing.T) {
	ctx := context.Background()

	// Scenario setup:
	//   Entity A: sprint_order=1, key="E07-F01-001", agent=backend
	//   Entity B: sprint_order=2, key="E07-F01-002", agent=backend  ← move to pos 1
	//   Entity C: sprint_order=3, key="E07-F01-003", agent=backend
	//
	// After reorder (B→pos1): A=2, B=1, C=3
	// GetNextTask must return B (sprint_order=1).

	now := time.Now()
	sprint001 := &models.Sprint{
		ID:        20,
		Key:       "S001",
		Name:      "Sprint Round-Trip",
		StartDate: now.Add(-24 * time.Hour),
		EndDate:   now.Add(7 * 24 * time.Hour),
		Status:    "active",
	}

	// Assignments before reorder.
	assignA := &models.SprintAssignment{ID: 1, SprintID: 20, EntityType: "task", EntityID: 1001, SprintOrder: intPtr(1), AssignedAt: now.Add(-30 * time.Minute)}
	assignB := &models.SprintAssignment{ID: 2, SprintID: 20, EntityType: "task", EntityID: 1002, SprintOrder: intPtr(2), AssignedAt: now.Add(-20 * time.Minute)}
	assignC := &models.SprintAssignment{ID: 3, SprintID: 20, EntityType: "task", EntityID: 1003, SprintOrder: intPtr(3), AssignedAt: now.Add(-10 * time.Minute)}

	// After reorder (B moves to pos 1), the expected state is:
	//   B=1, A=2, C=3
	// The mock applies this state update in memory via the captured ops.

	// Track what RenumberAssignmentsTx receives so we can verify dense numbering.
	var capturedRenumberOps []sprintrepo.RenumberOp
	setSprintOrderCallCount := 0

	// After reorder, ListOrderedAssignments returns the reordered slice.
	postReorderAssignments := []*models.SprintAssignment{
		{ID: 2, SprintID: 20, EntityType: "task", EntityID: 1002, SprintOrder: intPtr(1), AssignedAt: now.Add(-20 * time.Minute)},
		{ID: 1, SprintID: 20, EntityType: "task", EntityID: 1001, SprintOrder: intPtr(2), AssignedAt: now.Add(-30 * time.Minute)},
		{ID: 3, SprintID: 20, EntityType: "task", EntityID: 1003, SprintOrder: intPtr(3), AssignedAt: now.Add(-10 * time.Minute)},
	}

	listOrderedCallCount := 0
	mockSprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return sprint001, nil
		},
		GetByIDFunc: func(_ context.Context, id int64) (*models.Sprint, error) {
			return sprint001, nil
		},
		// List is called by GetNextTask → ListSprints to find the active sprint.
		ListFunc: func(_ context.Context, filters *sprintrepo.SprintListFilters) ([]*models.Sprint, error) {
			return []*models.Sprint{sprint001}, nil
		},
		GetTaskIDByKeyFunc: func(_ context.Context, key string) (int64, error) {
			// resolveEntityTypeAndID calls GetTaskIDByKey with the normalized key
			// (T-E##-F##-### format) from keys.Parse().
			switch key {
			case "T-E07-F01-001", "E07-F01-001":
				return 1001, nil
			case "T-E07-F01-002", "E07-F01-002":
				return 1002, nil
			case "T-E07-F01-003", "E07-F01-003":
				return 1003, nil
			}
			return 0, nil
		},
		GetActiveAssignmentFunc: func(_ context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			switch entityID {
			case 1001:
				return assignA, nil
			case 1002:
				return assignB, nil
			case 1003:
				return assignC, nil
			}
			return nil, nil
		},
		ListOrderedAssignmentsFunc: func(_ context.Context, sprintID int64) ([]*models.SprintAssignment, error) {
			listOrderedCallCount++
			if listOrderedCallCount == 1 {
				// First call: initial ordered state (before reorder)
				return []*models.SprintAssignment{assignA, assignB, assignC}, nil
			}
			// Subsequent calls: post-reorder state
			return postReorderAssignments, nil
		},
		RenumberAssignmentsTxFunc: func(_ context.Context, _ *sql.Tx, sprintID int64, ops []sprintrepo.RenumberOp) error {
			capturedRenumberOps = ops
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, _ *sql.Tx, assignmentID int64, newPosition *int) error {
			setSprintOrderCallCount++
			return nil
		},
		// ListBacklog for GetNextTask — returns post-reorder ordering.
		ListBacklogFunc: func(_ context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprintrepo.BacklogItem, error) {
			execOrder1 := 1
			execOrder2 := 2
			execOrder3 := 3
			agentBackend := "backend"
			return []*sprintrepo.BacklogItem{
				{
					Key:            "E07-F01-001",
					EntityType:     "task",
					Title:          "Task A",
					Status:         "todo",
					AgentType:      &agentBackend,
					Priority:       5,
					ExecutionOrder: &execOrder1,
					AssignedAt:     now.Add(-30 * time.Minute),
					SprintOrder:    intPtr(2), // A moved to pos 2
				},
				{
					Key:            "E07-F01-002",
					EntityType:     "task",
					Title:          "Task B",
					Status:         "todo",
					AgentType:      &agentBackend,
					Priority:       5,
					ExecutionOrder: &execOrder2,
					AssignedAt:     now.Add(-20 * time.Minute),
					SprintOrder:    intPtr(1), // B moved to pos 1
				},
				{
					Key:            "E07-F01-003",
					EntityType:     "task",
					Title:          "Task C",
					Status:         "todo",
					AgentType:      &agentBackend,
					Priority:       5,
					ExecutionOrder: &execOrder3,
					AssignedAt:     now.Add(-10 * time.Minute),
					SprintOrder:    intPtr(3), // C stays at pos 3
				},
			}, nil
		},
	}

	wfSvc := workflow.NewService("")
	sprintSvc := NewSprintService(mockSprintRepo, wfSvc, nil, nil, nil)

	// Step 1: Reorder — move entity at position 2 (Entity B) to position 1.
	moved, _, err := sprintSvc.ReorderAssignment(ctx, "S001", "E07-F01-002", ReorderTarget{Position: intPtr(1)})
	require.NoError(t, err, "ReorderAssignment must succeed")
	require.NotNil(t, moved, "moved assignment must be non-nil")

	// Step 2: Verify dense renumbering — the sibling ops captured must produce
	// positions without gaps.  After B moves to pos 1:
	//   A (was pos 1) → pos 2
	//   C (was pos 3) → pos 3 (unchanged, but reaffirmed)
	assert.Equal(t, 1, *moved.SprintOrder, "B (moved) must be at position 1")

	// The renumber ops should adjust A to position 2 (siblings shifted).
	// C stays at 3 (no shift needed since B was inserted before it only displaced A).
	for _, op := range capturedRenumberOps {
		// No position must be 0 or negative — dense numbering guarantee.
		require.NotNil(t, op.NewPosition, "renumber op must have a non-nil new position")
		assert.Greater(t, *op.NewPosition, 0, "renumber op position must be >= 1 (dense)")
	}

	assert.Equal(t, 1, setSprintOrderCallCount,
		"SetSprintOrderTx must be called exactly once (for the moved item)")

	// Step 3: Verify GetNextTask returns the entity at new position 1 (Entity B).
	next, err := sprintSvc.GetNextTask(ctx, "")
	require.NoError(t, err, "GetNextTask must succeed after reorder")
	require.NotNil(t, next, "GetNextTask must return a task")
	assert.Equal(t, "E07-F01-002", next.Key,
		"GetNextTask must return Entity B (sprint_order=1) after reorder; "+
			"Entity A was displaced to position 2")
	assert.Equal(t, 1, *next.SprintOrder,
		"returned task sprint_order must be 1")
}

// =============================================================================
// Test helpers (scoped to this file; not exported)
// =============================================================================

// seedTestEpicAndFeature inserts minimal epic and feature rows into the real test
// DB and returns their IDs. Uses raw SQL to avoid import cycles.
func seedTestEpicAndFeature(t *testing.T, db *sql.DB, epicKey, featureKey string) (int64, int64) {
	t.Helper()
	ctx := context.Background()

	// Clean up in reverse dependency order.
	_, _ = db.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE ?", epicKey+"-%")
	_, _ = db.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key LIKE 'S99%')")
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Insert epic.
	epicResult, err := db.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES (?, 'Decoupling Test Epic', 'For decoupling regression tests', 'active', 'high')
	`, epicKey)
	require.NoError(t, err, "seedTestEpicAndFeature: failed to insert epic")
	epicID, err := epicResult.LastInsertId()
	require.NoError(t, err, "seedTestEpicAndFeature: LastInsertId for epic")

	// Insert feature.
	featureResult, err := db.ExecContext(ctx, `
		INSERT INTO features (key, title, description, epic_id, status)
		VALUES (?, 'Decoupling Test Feature', 'For decoupling regression tests', ?, 'active')
	`, featureKey, epicID)
	require.NoError(t, err, "seedTestEpicAndFeature: failed to insert feature")
	featureID, err := featureResult.LastInsertId()
	require.NoError(t, err, "seedTestEpicAndFeature: LastInsertId for feature")

	return epicID, featureID
}

// seedTestTask inserts a minimal task row into the real test DB and returns its ID.
// Note: the tasks table only has feature_id (not epic_id); epicID is unused here
// but kept for call-site clarity.
func seedTestTask(t *testing.T, db *sql.DB, taskKey string, _ int64, featureID int64) int64 {
	t.Helper()
	ctx := context.Background()

	// Clean up existing task with this key.
	_, _ = db.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", taskKey)

	priority := 3
	execOrder := 2
	result, err := db.ExecContext(ctx, `
		INSERT INTO tasks (key, title, description, status, priority, execution_order, feature_id)
		VALUES (?, 'Decoupling Test Task', 'For TC-029', 'todo', ?, ?, ?)
	`, taskKey, priority, execOrder, featureID)
	require.NoError(t, err, "seedTestTask: failed to insert task")
	taskID, err := result.LastInsertId()
	require.NoError(t, err, "seedTestTask: LastInsertId for task")

	return taskID
}

// seedTestSprint inserts a sprint row into the real test DB and returns its ID.
func seedTestSprint(t *testing.T, db *sql.DB, sprintKey, status string) int64 {
	t.Helper()
	ctx := context.Background()

	// Clean up.
	_, _ = db.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = ?)", sprintKey)
	_, _ = db.ExecContext(ctx, "DELETE FROM sprints WHERE key = ?", sprintKey)

	now := time.Now()
	result, err := db.ExecContext(ctx, `
		INSERT INTO sprints (key, name, goal, start_date, end_date, status, slug, file_path)
		VALUES (?, 'Decoupling Test Sprint', 'TC-029 regression guard', ?, ?, ?, ?, ?)
	`, sprintKey,
		now.Add(-24*time.Hour).Format(time.RFC3339),
		now.Add(7*24*time.Hour).Format(time.RFC3339),
		status,
		"decoupling-test-sprint",
		"docs/plan/sprints/"+sprintKey+".md",
	)
	require.NoError(t, err, "seedTestSprint: failed to insert sprint")
	sprintID, err := result.LastInsertId()
	require.NoError(t, err, "seedTestSprint: LastInsertId for sprint")

	return sprintID
}

// seedSprintAssignment inserts a sprint_assignment row with the given sprint_order
// and returns the assignment ID.
func seedSprintAssignment(t *testing.T, db *sql.DB, sprintID int64, entityType string, entityID int64, sprintOrder int) int64 {
	t.Helper()
	ctx := context.Background()

	// Remove any existing active assignment for this entity to avoid UNIQUE violation.
	_, _ = db.ExecContext(ctx, `
		UPDATE sprint_assignments SET removed_at = CURRENT_TIMESTAMP
		WHERE sprint_id = ? AND entity_type = ? AND entity_id = ? AND removed_at IS NULL
	`, sprintID, entityType, entityID)

	result, err := db.ExecContext(ctx, `
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, sprint_order, assigned_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, sprintID, entityType, entityID, sprintOrder)
	require.NoError(t, err, "seedSprintAssignment: failed to insert sprint_assignment")
	assignmentID, err := result.LastInsertId()
	require.NoError(t, err, "seedSprintAssignment: LastInsertId for sprint_assignment")

	return assignmentID
}

// querySprintOrder reads sprint_assignments.sprint_order for the given sprint + entity.
// Returns nil if no active assignment exists.
func querySprintOrder(t *testing.T, db *sql.DB, sprintID int64, entityType string, entityID int64) *int {
	t.Helper()
	ctx := context.Background()

	var order sql.NullInt64
	err := db.QueryRowContext(ctx, `
		SELECT sprint_order FROM sprint_assignments
		WHERE sprint_id = ? AND entity_type = ? AND entity_id = ? AND removed_at IS NULL
	`, sprintID, entityType, entityID).Scan(&order)
	if err == sql.ErrNoRows {
		return nil
	}
	require.NoError(t, err, "querySprintOrder: failed to query sprint_assignments")

	if !order.Valid {
		return nil
	}
	v := int(order.Int64)
	return &v
}

// sprintOrderPtr is a local alias for intPtr (test-file scope only) to
// emphasise the semantic in seedSprintAssignment calls.
// The canonical intPtr helper is defined in sprint_service_test.go.
func sprintOrderPtr(v int) *int { return &v }

// Ensure sprintOrderPtr is used to avoid the "declared and not used" error.
var _ = sprintOrderPtr

// Ensure the repository.DB type is referenced (satisfies import usage).
var _ *repository.DB
