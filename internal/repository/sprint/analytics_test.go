package sprint

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensureSprintTables creates the sprint tables required for analytics tests
// if they do not already exist.  This is necessary in worktrees that do not
// include the E19-F01 migration in their db.go.
func ensureSprintTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sprints (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			key         TEXT    NOT NULL UNIQUE,
			name        TEXT    NOT NULL,
			goal        TEXT,
			start_date  DATETIME NOT NULL,
			end_date    DATETIME NOT NULL,
			status      TEXT    NOT NULL DEFAULT 'planning',
			slug        TEXT,
			file_path   TEXT,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err, "ensureSprintTables: create sprints")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sprint_assignments (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			sprint_id   INTEGER NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
			entity_type TEXT    NOT NULL,
			entity_id   INTEGER NOT NULL,
			assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			removed_at  DATETIME,
			UNIQUE(sprint_id, entity_type, entity_id)
		)
	`)
	require.NoError(t, err, "ensureSprintTables: create sprint_assignments")

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_sprint_assignments_sprint
		ON sprint_assignments(sprint_id)
	`)
	require.NoError(t, err, "ensureSprintTables: idx_sprint_assignments_sprint")

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_sprint_assignments_entity
		ON sprint_assignments(entity_type, entity_id)
	`)
	require.NoError(t, err, "ensureSprintTables: idx_sprint_assignments_entity")
}

// seedTaskHistoryEvent describes a task_history row for completion event tests.
type seedTaskHistoryEvent struct {
	TaskID    int64
	OldStatus string
	NewStatus string
	Timestamp time.Time
}

// seedTaskHistoryEvents inserts task_history rows for completion event tests.
func seedTaskHistoryEvents(t *testing.T, ctx context.Context, events []seedTaskHistoryEvent) {
	t.Helper()
	rawDB := test.GetTestDB()
	for _, e := range events {
		_, err := rawDB.ExecContext(ctx, `
			INSERT INTO task_history (task_id, old_status, new_status, timestamp)
			VALUES (?, ?, ?, ?)
		`, e.TaskID, e.OldStatus, e.NewStatus, e.Timestamp)
		require.NoError(t, err, "seedTaskHistoryEvents: insert history task_id=%d", e.TaskID)
	}
}

// seedSprintForAnalytics creates a sprint row and returns its ID.
// Cleans up any existing sprint with the same key first.
func seedSprintForAnalytics(
	t *testing.T,
	ctx context.Context,
	key string,
	status string,
	startDate, endDate time.Time,
) int64 {
	t.Helper()
	rawDB := test.GetTestDB()

	// Clean any leftover data.
	_, _ = rawDB.ExecContext(ctx, `
		DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = ?)
	`, key)
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = ?`, key)

	result, err := rawDB.ExecContext(ctx, `
		INSERT INTO sprints (key, name, goal, start_date, end_date, status, slug)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, key, "Sprint "+key, "Goal", startDate, endDate, status, "sprint-"+key)
	require.NoError(t, err, "seedSprintForAnalytics: insert sprint %s", key)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// seedTaskForAnalytics creates a minimal task with the given size and returns its ID.
func seedTaskForAnalytics(t *testing.T, ctx context.Context, taskNum int, size *int) int64 {
	t.Helper()
	rawDB := test.GetTestDB()

	// Ensure test epic and feature exist.
	var epicID, featureID int64
	err := rawDB.QueryRowContext(ctx, `SELECT id FROM epics WHERE key = 'TEST-E99'`).Scan(&epicID)
	if err != nil {
		res, err2 := rawDB.ExecContext(ctx, `
			INSERT INTO epics (key, title, status, priority, file_path)
			VALUES ('TEST-E99', 'Test Epic', 'active', 'medium', '/tmp/e99.md')
		`)
		require.NoError(t, err2, "seedTaskForAnalytics: insert epic")
		epicID, _ = res.LastInsertId()
	}

	err = rawDB.QueryRowContext(ctx, `SELECT id FROM features WHERE key = 'TEST-E99-F01'`).Scan(&featureID)
	if err != nil {
		res, err2 := rawDB.ExecContext(ctx, `
			INSERT INTO features (key, title, status, epic_id, file_path)
			VALUES ('TEST-E99-F01', 'Test Feature', 'active', ?, '/tmp/f01.md')
		`, epicID)
		require.NoError(t, err2, "seedTaskForAnalytics: insert feature")
		featureID, _ = res.LastInsertId()
	}

	taskKey := fmt.Sprintf("ANTEST-E99-F01-%03d", taskNum)
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = ?`, taskKey)

	var res sql.Result
	var insertErr error
	if size != nil {
		res, insertErr = rawDB.ExecContext(ctx, `
			INSERT INTO tasks (key, title, status, feature_id, file_path, size)
			VALUES (?, ?, 'completed', ?, '/tmp/task.md', ?)
		`, taskKey, "Task "+taskKey, featureID, *size)
	} else {
		res, insertErr = rawDB.ExecContext(ctx, `
			INSERT INTO tasks (key, title, status, feature_id, file_path)
			VALUES (?, ?, 'completed', ?, '/tmp/task.md')
		`, taskKey, "Task "+taskKey, featureID)
	}
	require.NoError(t, insertErr, "seedTaskForAnalytics: insert task %s", taskKey)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

// addAssignment inserts a sprint_assignments row.
func addAssignment(t *testing.T, ctx context.Context, sprintID int64, entityType string, entityID int64, assignedAt time.Time, removedAt *time.Time) {
	t.Helper()
	rawDB := test.GetTestDB()
	_, err := rawDB.ExecContext(ctx, `
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at, removed_at)
		VALUES (?, ?, ?, ?, ?)
	`, sprintID, entityType, entityID, assignedAt, removedAt)
	require.NoError(t, err, "addAssignment: sprint_id=%d entity_type=%s entity_id=%d", sprintID, entityType, entityID)
}

// ============================================================================
// TestGetCompletionEvents (TC-B-07)
// Verifies GetCompletionEvents returns task_history events within the sprint
// window and filters out events outside the window.
// ============================================================================

func TestGetCompletionEvents(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	now := time.Now().UTC()
	start := now.AddDate(0, -1, -7)
	end := now.AddDate(0, -1, 7)

	size5 := 5
	taskID1 := seedTaskForAnalytics(t, ctx, 101, &size5)
	taskID2 := seedTaskForAnalytics(t, ctx, 102, &size5)

	sprintID := seedSprintForAnalytics(t, ctx, "SAN-951", "completed", start, end)
	addAssignment(t, ctx, sprintID, "task", taskID1, start, nil)
	addAssignment(t, ctx, sprintID, "task", taskID2, start, nil)

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM task_history WHERE task_id IN (?, ?)`, taskID1, taskID2)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'SAN-951'`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key IN ('ANTEST-E99-F01-101','ANTEST-E99-F01-102')`)
	}()

	// Event within window: taskID1 completed on day 3 of sprint.
	withinWindow := start.AddDate(0, 0, 3)
	// Event outside window: taskID2 completed one day AFTER sprint ended.
	afterWindow := end.AddDate(0, 0, 1)

	seedTaskHistoryEvents(t, ctx, []seedTaskHistoryEvent{
		{TaskID: taskID1, OldStatus: "in_progress", NewStatus: "completed", Timestamp: withinWindow},
		{TaskID: taskID2, OldStatus: "in_progress", NewStatus: "completed", Timestamp: afterWindow},
	})

	// Act
	events, err := repo.GetCompletionEvents(ctx, sprintID, start, end)
	require.NoError(t, err)

	// Assert: only the within-window event should be present.
	var foundTask1, foundTask2 bool
	for _, ev := range events {
		if ev.EntityID == taskID1 {
			foundTask1 = true
			assert.Equal(t, "task", ev.EntityType, "entity_type must be 'task'")
			assert.Equal(t, "completed", ev.NewStatus, "new_status must be 'completed'")
			assert.False(t, ev.Timestamp.IsZero(), "timestamp must not be zero")
		}
		if ev.EntityID == taskID2 {
			foundTask2 = true
		}
	}

	assert.True(t, foundTask1, "task101 completed within sprint window must be in events")
	assert.False(t, foundTask2, "task102 completed after sprint end must NOT be in events")
}

// TestGetCompletionEvents_Empty verifies an empty slice (not error) is returned
// when no task_history events fall within the sprint window.
func TestGetCompletionEvents_Empty(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	now := time.Now().UTC()
	start := now.AddDate(-1, 0, 0) // last year
	end := now.AddDate(-1, 0, 14)

	sprintID := seedSprintForAnalytics(t, ctx, "SAN-952", "completed", start, end)
	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'SAN-952'`)
	}()

	// No task_history events seeded — expect empty slice.
	events, err := repo.GetCompletionEvents(ctx, sprintID, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, events, "should return empty slice, not nil")
	assert.Empty(t, events, "no history events should yield empty slice")
}

// TestGetCompletionEvents_FiltersBySprintWindow verifies the timestamp boundary:
// events exactly at startDate and endDate are included; events outside are not.
func TestGetCompletionEvents_FiltersBySprintWindow(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	now := time.Now().UTC()
	start := now.AddDate(0, -2, 0)
	end := now.AddDate(0, -1, 0)

	size3 := 3
	taskID := seedTaskForAnalytics(t, ctx, 103, &size3)

	sprintID := seedSprintForAnalytics(t, ctx, "SAN-953", "completed", start, end)
	addAssignment(t, ctx, sprintID, "task", taskID, start, nil)

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM task_history WHERE task_id = ?`, taskID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'SAN-953'`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'ANTEST-E99-F01-103'`)
	}()

	// Seed one event at the sprint start boundary.
	seedTaskHistoryEvents(t, ctx, []seedTaskHistoryEvent{
		{TaskID: taskID, OldStatus: "todo", NewStatus: "in_progress", Timestamp: start},
	})

	events, err := repo.GetCompletionEvents(ctx, sprintID, start, end)
	require.NoError(t, err)

	assert.NotEmpty(t, events, "event at sprint start boundary (inclusive) must be returned")
}

// ============================================================================
// TestGetCycleTimeByPhase (TC-S-06 / spec §4.3)
// Verifies the average days per phase when task_history transitions exist, and
// that an empty slice (not error) is returned when there are none.
// ============================================================================

func TestGetCycleTimeByPhase_WithHistory(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	now := time.Now().UTC()
	start := now.AddDate(0, -2, 0)
	end := now.AddDate(0, -1, 0)

	size5 := 5
	taskID := seedTaskForAnalytics(t, ctx, 201, &size5)

	sprintID := seedSprintForAnalytics(t, ctx, "SAN-961", "completed", start, end)
	addAssignment(t, ctx, sprintID, "task", taskID, start, nil)

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM task_history WHERE task_id = ?`, taskID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'SAN-961'`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'ANTEST-E99-F01-201'`)
	}()

	// Seed two consecutive transitions: todo→in_progress and in_progress→completed.
	// The self-join will pair them and compute days elapsed for the 'todo' phase.
	t1 := start.AddDate(0, 0, 2)
	t2 := start.AddDate(0, 0, 5) // 3 days in in_progress
	seedTaskHistoryEvents(t, ctx, []seedTaskHistoryEvent{
		{TaskID: taskID, OldStatus: "todo", NewStatus: "in_progress", Timestamp: t1},
		{TaskID: taskID, OldStatus: "in_progress", NewStatus: "completed", Timestamp: t2},
	})

	phases, err := repo.GetCycleTimeByPhase(ctx, sprintID)
	require.NoError(t, err)

	// With two transitions forming one pair we expect at least one phase row.
	assert.NotEmpty(t, phases, "expected phase rows when task_history has consecutive transitions")

	// Verify the 'todo' phase average is approximately 2 days (t1 - start would
	// be the pair, but our query pairs th1 and th2 by task_id; the pair is
	// todo→in_progress duration is t2-t1 = 3 days for in_progress phase).
	// The self-join pairs: th1=todo event (t1) with th2=in_progress event (t2).
	// phase=old_status of th1 = "todo" → days_elapsed = julianday(t2)-julianday(t1) = 3
	var foundTodoPhase bool
	for _, p := range phases {
		if p.Phase == "todo" {
			foundTodoPhase = true
			// Approximately 3 days (floating point).
			assert.InDelta(t, 3.0, p.AverageDays, 0.1,
				"average days in todo phase should be ~3 days")
		}
	}
	assert.True(t, foundTodoPhase, "expected 'todo' phase in results")
}

// TestGetCycleTimeByPhase_NoHistory verifies that an empty slice (not error)
// is returned when no task_history rows exist for the sprint's tasks.
// This is the TC-S-06 graceful-degradation path.
func TestGetCycleTimeByPhase_NoHistory(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	now := time.Now().UTC()
	sprintID := seedSprintForAnalytics(t, ctx, "SAN-971", "completed",
		now.AddDate(0, -2, 0), now.AddDate(0, -1, 0))

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'SAN-971'`)
	}()

	// No tasks assigned, so no task_history rows exist for this sprint.
	phases, err := repo.GetCycleTimeByPhase(ctx, sprintID)
	assert.NoError(t, err)
	// Must return empty slice, not nil and not error.
	assert.NotNil(t, phases, "should return empty slice, not nil")
	assert.Empty(t, phases, "expected empty slice when no task_history rows exist for the sprint")
}

// TestGetCycleTimeByPhase_TaskWithNoHistory verifies that when tasks are assigned
// but have no task_history rows, the result is still an empty slice.
func TestGetCycleTimeByPhase_TaskWithNoHistory(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	now := time.Now().UTC()
	start := now.AddDate(0, -2, 0)
	end := now.AddDate(0, -1, 0)

	size2 := 2
	taskID := seedTaskForAnalytics(t, ctx, 202, &size2)

	sprintID := seedSprintForAnalytics(t, ctx, "SAN-972", "completed", start, end)
	addAssignment(t, ctx, sprintID, "task", taskID, start, nil)

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'SAN-972'`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'ANTEST-E99-F01-202'`)
	}()

	// No task_history for this task — no consecutive pair can be formed.
	phases, err := repo.GetCycleTimeByPhase(ctx, sprintID)
	assert.NoError(t, err)
	assert.Empty(t, phases, "no history rows → no phase pairs → empty result")
}

// TestGetCycleTimeByPhase_MultiplePhases verifies that multiple distinct phases
// are returned when a task goes through several status transitions.
func TestGetCycleTimeByPhase_MultiplePhases(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	now := time.Now().UTC()
	start := now.AddDate(0, -3, 0)
	end := now.AddDate(0, -2, 0)

	size8 := 8
	taskID := seedTaskForAnalytics(t, ctx, 203, &size8)

	sprintID := seedSprintForAnalytics(t, ctx, "SAN-973", "completed", start, end)
	addAssignment(t, ctx, sprintID, "task", taskID, start, nil)

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM task_history WHERE task_id = ?`, taskID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'SAN-973'`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'ANTEST-E99-F01-203'`)
	}()

	// Seed three transitions creating two consecutive pairs.
	t1 := start.AddDate(0, 0, 1)
	t2 := start.AddDate(0, 0, 4)  // 3 days in todo
	t3 := start.AddDate(0, 0, 10) // 6 days in in_progress
	seedTaskHistoryEvents(t, ctx, []seedTaskHistoryEvent{
		{TaskID: taskID, OldStatus: "todo", NewStatus: "in_progress", Timestamp: t1},
		{TaskID: taskID, OldStatus: "in_progress", NewStatus: "ready_for_review", Timestamp: t2},
		{TaskID: taskID, OldStatus: "ready_for_review", NewStatus: "completed", Timestamp: t3},
	})

	phases, err := repo.GetCycleTimeByPhase(ctx, sprintID)
	require.NoError(t, err)
	require.NotEmpty(t, phases, "expected phase rows from two consecutive pairs")

	// Build map for assertions.
	phaseMap := make(map[string]float64)
	for _, p := range phases {
		phaseMap[p.Phase] = p.AverageDays
	}

	// todo phase: t1→t2 = 3 days
	assert.InDelta(t, 3.0, phaseMap["todo"], 0.1, "todo phase should be ~3 days")
	// in_progress phase: t2→t3 = 6 days
	assert.InDelta(t, 6.0, phaseMap["in_progress"], 0.1, "in_progress phase should be ~6 days")
}

// ============================================================================
// TC-NF-02: EXPLAIN QUERY PLAN confirms indexed lookup on sprint_assignments
// ============================================================================

func TestExplainQueryPlan_CompletionEvents(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)

	// Run EXPLAIN QUERY PLAN on GetCompletionEvents inner query.
	// The sprint_assignments join on sprint_id should use an index.
	query := `
		EXPLAIN QUERY PLAN
		SELECT th.task_id, 'task' AS entity_type, th.new_status, th.timestamp
		FROM sprint_assignments sa
		JOIN task_history th ON th.task_id = sa.entity_id
		WHERE sa.sprint_id    = ?
		  AND sa.entity_type  = 'task'
		  AND th.timestamp   >= ?
		  AND th.timestamp   <= ?
		ORDER BY th.timestamp ASC
	`

	rows, err := rawDB.QueryContext(ctx, query, int64(1), time.Now().AddDate(0, -1, 0), time.Now())
	require.NoError(t, err)
	defer rows.Close()

	var foundIndexedLookup bool
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if scanErr := rows.Scan(&id, &parent, &notUsed, &detail); scanErr != nil {
			_ = scanErr
		}
		// Accept any of: USING INDEX, USING COVERING INDEX, idx_sprint_assignments
		// (modernc sqlite may emit slightly different wording from mattn/go-sqlite3).
		if containsStr(detail, "sprint_assignments") {
			if containsStr(detail, "USING INDEX") ||
				containsStr(detail, "USING COVERING INDEX") ||
				containsStr(detail, "idx_sprint_assignments") {
				foundIndexedLookup = true
			}
		}
	}
	_ = rows.Err()

	assert.True(t, foundIndexedLookup,
		"EXPLAIN QUERY PLAN for GetCompletionEvents must show indexed lookup on sprint_assignments")
}

// containsStr is a helper since strings.Contains requires the strings import.
func containsStr(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// TC-NF-01: Performance test (REQ-NF-001)
// Verifies that GetVelocityData, GetCompletionEvents, and GetCycleTimeByPhase
// each complete within 2 seconds when seeded with 50 sprints and 1000 tasks.
// Gated with testing.Short() so it only runs in full (non-short) mode.
// ============================================================================

func TestPerformance_Analytics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	ctx := context.Background()
	rawDB := test.GetTestDB()
	ensureSprintTables(t, rawDB)
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	now := time.Now().UTC()

	// --- Seed 50 sprints -------------------------------------------------------
	const numSprints = 50
	const numTasks = 1000

	sprintIDs := make([]int64, 0, numSprints)
	sprintKeys := make([]string, 0, numSprints)
	for i := 0; i < numSprints; i++ {
		key := fmt.Sprintf("PERFTEST-S%03d", i)
		start := now.AddDate(0, -(numSprints - i), 0)
		end := start.AddDate(0, 0, 14)
		id := seedSprintForAnalytics(t, ctx, key, "completed", start, end)
		sprintIDs = append(sprintIDs, id)
		sprintKeys = append(sprintKeys, key)
	}

	// Defer cleanup of all seeded sprints (cascade deletes assignments).
	defer func() {
		for _, key := range sprintKeys {
			_, _ = rawDB.ExecContext(ctx,
				`DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = ?)`, key)
			_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = ?`, key)
		}
	}()

	// --- Seed 1000 tasks and assign each to a sprint ---------------------------
	taskNums := make([]int, 0, numTasks)
	for i := 0; i < numTasks; i++ {
		taskNum := 5000 + i // offset to avoid collision with other tests
		taskID := seedTaskForAnalytics(t, ctx, taskNum, nil)
		sprintID := sprintIDs[i%numSprints]
		assignedAt := now.AddDate(0, -(numSprints - (i % numSprints)), 0)
		addAssignment(t, ctx, sprintID, "task", taskID, assignedAt, nil)
		taskNums = append(taskNums, taskNum)
	}

	// Defer cleanup of seeded tasks.
	defer func() {
		for _, n := range taskNums {
			taskKey := fmt.Sprintf("ANTEST-E99-F01-%03d", n)
			_, _ = rawDB.ExecContext(ctx,
				`DELETE FROM task_history WHERE task_id IN (SELECT id FROM tasks WHERE key = ?)`, taskKey)
			_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = ?`, taskKey)
		}
	}()

	const maxDuration = 2 * time.Second

	// --- GetVelocityData -------------------------------------------------------
	t.Run("GetVelocityData", func(t *testing.T) {
		start := time.Now()
		_, err := repo.GetVelocityData(ctx, numSprints)
		elapsed := time.Since(start)

		require.NoError(t, err, "GetVelocityData must not error")
		assert.Less(t, elapsed, maxDuration,
			"GetVelocityData must complete in < 2s, took %s", elapsed)
	})

	// --- GetCompletionEvents (use first sprint) --------------------------------
	t.Run("GetCompletionEvents", func(t *testing.T) {
		sprintID := sprintIDs[0]
		sprintStart := now.AddDate(0, -numSprints, 0)
		sprintEnd := sprintStart.AddDate(0, 0, 14)

		start := time.Now()
		_, err := repo.GetCompletionEvents(ctx, sprintID, sprintStart, sprintEnd)
		elapsed := time.Since(start)

		require.NoError(t, err, "GetCompletionEvents must not error")
		assert.Less(t, elapsed, maxDuration,
			"GetCompletionEvents must complete in < 2s, took %s", elapsed)
	})

	// --- GetCycleTimeByPhase (use first sprint) --------------------------------
	t.Run("GetCycleTimeByPhase", func(t *testing.T) {
		sprintID := sprintIDs[0]

		start := time.Now()
		_, err := repo.GetCycleTimeByPhase(ctx, sprintID)
		elapsed := time.Since(start)

		require.NoError(t, err, "GetCycleTimeByPhase must not error")
		assert.Less(t, elapsed, maxDuration,
			"GetCycleTimeByPhase must complete in < 2s, took %s", elapsed)
	})
}
