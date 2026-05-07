package sprint

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedEntity describes an entity to be assigned to a sprint in tests.
type seedEntity struct {
	EntityType string
	EntityID   int64
	Size       *int
	AssignedAt time.Time
	RemovedAt  *time.Time
}

// seedTaskHistoryEvent describes a task_history row for completion event tests.
type seedTaskHistoryEvent struct {
	TaskID    int64
	OldStatus string
	NewStatus string
	Timestamp time.Time
}

// seedSprintWithAssignments creates a sprint and populates sprint_assignments.
// Returns the sprint ID. Cleans up any existing sprint with the same key first.
func seedSprintWithAssignments(
	t *testing.T,
	database interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (interface{ LastInsertId() (int64, error) }, error)
	},
	ctx context.Context,
	sprintKey string,
	status models.SprintStatus,
	startDate, endDate time.Time,
	entities []seedEntity,
) int64 {
	t.Helper()

	// Use the underlying *sql.DB via test.GetTestDB() directly.
	// We accept the raw DB for direct SQL operations.
	rawDB := test.GetTestDB()

	// Clean up existing sprint with this key.
	_, _ = rawDB.ExecContext(ctx, `
		DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = ?)
	`, sprintKey)
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = ?`, sprintKey)

	// Insert the sprint.
	result, err := rawDB.ExecContext(ctx, `
		INSERT INTO sprints (key, name, goal, start_date, end_date, status, slug)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, sprintKey, "Sprint "+sprintKey, "Goal", startDate, endDate, status, "sprint-"+sprintKey)
	require.NoError(t, err, "seedSprintWithAssignments: insert sprint")

	sprintID, err := result.LastInsertId()
	require.NoError(t, err, "seedSprintWithAssignments: last insert id")

	// Insert assignments.
	for _, e := range entities {
		assignedAt := e.AssignedAt
		if assignedAt.IsZero() {
			assignedAt = startDate
		}
		_, err := rawDB.ExecContext(ctx, `
			INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at, removed_at)
			VALUES (?, ?, ?, ?, ?)
		`, sprintID, e.EntityType, e.EntityID, assignedAt, e.RemovedAt)
		require.NoError(t, err, "seedSprintWithAssignments: insert assignment entity_type=%s entity_id=%d", e.EntityType, e.EntityID)
	}

	return sprintID
}

// seedTaskWithSize creates a task with the given size and returns its ID.
// Requires epic and feature to exist. Uses a synthetic key based on taskNum.
func seedTaskWithSize(t *testing.T, ctx context.Context, taskNum int, size *int) int64 {
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
		require.NoError(t, err2, "seedTaskWithSize: insert epic")
		epicID, _ = res.LastInsertId()
	}

	err = rawDB.QueryRowContext(ctx, `SELECT id FROM features WHERE key = 'TEST-E99-F01'`).Scan(&featureID)
	if err != nil {
		res, err2 := rawDB.ExecContext(ctx, `
			INSERT INTO features (key, title, status, epic_id, file_path)
			VALUES ('TEST-E99-F01', 'Test Feature', 'active', ?, '/tmp/f01.md')
		`, epicID)
		require.NoError(t, err2, "seedTaskWithSize: insert feature")
		featureID, _ = res.LastInsertId()
	}

	taskKey := "TEST-E99-F01-" + intToStr(taskNum)
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = ?`, taskKey)

	var res interface{ LastInsertId() (int64, error) }
	var insertErr error
	if size != nil {
		res, insertErr = rawDB.ExecContext(ctx, `
			INSERT INTO tasks (key, title, status, feature_id, file_path, size)
			VALUES (?, ?, 'completed', ?, '/tmp/task.md', ?)
		`, taskKey, "Task "+taskKey, featureID, *size)
	} else {
		res, insertErr = rawDB.ExecContext(ctx, `
			INSERT INTO tasks (key, title, status, feature_id, file_path, size)
			VALUES (?, ?, 'completed', ?, '/tmp/task.md', NULL)
		`, taskKey, "Task "+taskKey, featureID)
	}
	require.NoError(t, insertErr, "seedTaskWithSize: insert task")
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

// intToStr converts an int to a 3-digit string (e.g. 1 → "001").
func intToStr(n int) string {
	if n < 10 {
		return "00" + string(rune('0'+n))
	}
	if n < 100 {
		return "0" + string(rune('0'+n/10)) + string(rune('0'+n%10))
	}
	return string(rune('0'+n/100)) + string(rune('0'+n/10%10)) + string(rune('0'+n%10))
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

// ensureSprintTables creates the sprint tables required for analytics tests
// if they do not already exist.
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

// seedSprintForAnalytics creates a sprint row and returns its ID.
func seedSprintForAnalytics(
	t *testing.T,
	ctx context.Context,
	key, status string,
	start, end time.Time,
) int64 {
	t.Helper()
	rawDB := test.GetTestDB()
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = ?`, key)

	result, err := rawDB.ExecContext(ctx, `
		INSERT INTO sprints (key, name, goal, start_date, end_date, status, slug)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, key, "Sprint "+key, "Goal", start, end, status, "sprint-"+key)
	require.NoError(t, err, "seedSprintForAnalytics: insert sprint %s", key)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// seedTaskForAnalytics creates a minimal task with the given size and returns its ID.
func seedTaskForAnalytics(t *testing.T, ctx context.Context, taskNum int, size *int) int64 {
	t.Helper()
	rawDB := test.GetTestDB()

	var epicID, featureID int64
	err := rawDB.QueryRowContext(ctx, `SELECT id FROM epics WHERE key = 'ANTEST-E99'`).Scan(&epicID)
	if err != nil {
		res, err2 := rawDB.ExecContext(ctx, `
			INSERT INTO epics (key, title, status, priority, file_path)
			VALUES ('ANTEST-E99', 'Analytics Test Epic', 'active', 'medium', '/tmp/antest-e99.md')
		`)
		require.NoError(t, err2, "seedTaskForAnalytics: insert epic")
		epicID, _ = res.LastInsertId()
	}

	err = rawDB.QueryRowContext(ctx, `SELECT id FROM features WHERE key = 'ANTEST-E99-F01'`).Scan(&featureID)
	if err != nil {
		res, err2 := rawDB.ExecContext(ctx, `
			INSERT INTO features (key, title, status, epic_id, file_path)
			VALUES ('ANTEST-E99-F01', 'Analytics Test Feature', 'active', ?, '/tmp/antest-f01.md')
		`, epicID)
		require.NoError(t, err2, "seedTaskForAnalytics: insert feature")
		featureID, _ = res.LastInsertId()
	}

	taskKey := fmt.Sprintf("ANTEST-E99-F01-%03d", taskNum)
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = ?`, taskKey)

	var insertErr error
	var res interface{ LastInsertId() (int64, error) }
	if size != nil {
		res, insertErr = rawDB.ExecContext(ctx, `
			INSERT INTO tasks (key, title, status, feature_id, file_path, size)
			VALUES (?, ?, 'completed', ?, '/tmp/antest-task.md', ?)
		`, taskKey, "Task "+taskKey, featureID, *size)
	} else {
		res, insertErr = rawDB.ExecContext(ctx, `
			INSERT INTO tasks (key, title, status, feature_id, file_path, size)
			VALUES (?, ?, 'completed', ?, '/tmp/antest-task.md', NULL)
		`, taskKey, "Task "+taskKey, featureID)
	}
	require.NoError(t, insertErr, "seedTaskForAnalytics: insert task %s", taskKey)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

// addAssignment inserts a sprint_assignments row.
func addAssignment(
	t *testing.T,
	ctx context.Context,
	sprintID int64,
	entityType string,
	entityID int64,
	assignedAt time.Time,
	removedAt *time.Time,
) {
	t.Helper()
	rawDB := test.GetTestDB()
	_, err := rawDB.ExecContext(ctx, `
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at, removed_at)
		VALUES (?, ?, ?, ?, ?)
	`, sprintID, entityType, entityID, assignedAt, removedAt)
	require.NoError(t, err, "addAssignment: sprint_id=%d entity_type=%s entity_id=%d", sprintID, entityType, entityID)
}

// ============================================================================
// TestGetVelocityData_CompletedSprints
// TC-NF-02 (partial): verifies GetVelocityData returns correct Σ size per sprint
// for completed sprints, with unsized entities counted separately.
// ============================================================================

func TestGetVelocityData_CompletedSprints(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	// Clean up test sprints.
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key IN ('S901', 'S902')`)

	now := time.Now().UTC()
	start := now.AddDate(0, -3, 0)
	end := now.AddDate(0, -2, 0)

	// Sprint S901: 2 tasks with sizes 5 and 8, 1 task unsized.
	size5 := 5
	size8 := 8
	taskID1 := seedTaskWithSize(t, ctx, 1, &size5)
	taskID2 := seedTaskWithSize(t, ctx, 2, &size8)
	taskID3 := seedTaskWithSize(t, ctx, 3, nil) // unsized

	sprintID1 := seedSprintWithAssignments(t, nil, ctx, "S901", "completed", start, end, []seedEntity{
		{EntityType: "task", EntityID: taskID1, Size: &size5},
		{EntityType: "task", EntityID: taskID2, Size: &size8},
		{EntityType: "task", EntityID: taskID3, Size: nil},
	})
	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID1)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S901'`)
	}()

	// Sprint S902: 1 task with size 10.
	size10 := 10
	taskID4 := seedTaskWithSize(t, ctx, 4, &size10)

	start2 := now.AddDate(0, -2, 0)
	end2 := now.AddDate(0, -1, 0)
	sprintID2 := seedSprintWithAssignments(t, nil, ctx, "S902", "completed", start2, end2, []seedEntity{
		{EntityType: "task", EntityID: taskID4, Size: &size10},
	})
	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID2)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S902'`)
	}()

	rows, err := repo.GetVelocityData(ctx, 10)
	require.NoError(t, err)

	// Find our test sprints in the results.
	var row901, row902 *VelocityRow
	for i := range rows {
		switch rows[i].SprintKey {
		case "S901":
			row901 = &rows[i]
		case "S902":
			row902 = &rows[i]
		}
	}

	require.NotNil(t, row901, "expected S901 in velocity results")
	assert.Equal(t, 13, row901.CompletedSize, "S901 completed size should be 5+8=13")
	assert.Equal(t, 1, row901.UnsizedCompleted, "S901 should have 1 unsized completed task")

	require.NotNil(t, row902, "expected S902 in velocity results")
	assert.Equal(t, 10, row902.CompletedSize, "S902 completed size should be 10")
	assert.Equal(t, 0, row902.UnsizedCompleted, "S902 should have 0 unsized completed tasks")

	// Cleanup tasks.
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key IN ('TEST-E99-F01-001','TEST-E99-F01-002','TEST-E99-F01-003','TEST-E99-F01-004')`)
}

// ============================================================================
// TestGetVelocityData_Limit
// Verifies that the limit parameter is respected and results are oldest-first.
// ============================================================================

func TestGetVelocityData_Limit(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	// Clean up test sprints.
	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key IN ('S911', 'S912', 'S913')`)

	size3 := 3
	taskID5 := seedTaskWithSize(t, ctx, 5, &size3)
	taskID6 := seedTaskWithSize(t, ctx, 6, &size3)
	taskID7 := seedTaskWithSize(t, ctx, 7, &size3)

	now := time.Now().UTC()

	// Three completed sprints in chronological order.
	sprintID1 := seedSprintWithAssignments(t, nil, ctx, "S911", "completed",
		now.AddDate(0, -6, 0), now.AddDate(0, -5, 0),
		[]seedEntity{{EntityType: "task", EntityID: taskID5, Size: &size3}})
	sprintID2 := seedSprintWithAssignments(t, nil, ctx, "S912", "completed",
		now.AddDate(0, -4, 0), now.AddDate(0, -3, 0),
		[]seedEntity{{EntityType: "task", EntityID: taskID6, Size: &size3}})
	sprintID3 := seedSprintWithAssignments(t, nil, ctx, "S913", "completed",
		now.AddDate(0, -2, 0), now.AddDate(0, -1, 0),
		[]seedEntity{{EntityType: "task", EntityID: taskID7, Size: &size3}})

	defer func() {
		for _, id := range []int64{sprintID1, sprintID2, sprintID3} {
			_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, id)
		}
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key IN ('S911', 'S912', 'S913')`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key IN ('TEST-E99-F01-005','TEST-E99-F01-006','TEST-E99-F01-007')`)
	}()

	// Request only 2 results — should return S912 and S913 (the 2 most recent completed).
	rows, err := repo.GetVelocityData(ctx, 2)
	require.NoError(t, err)

	// Filter to our test sprints.
	var testRows []VelocityRow
	for _, r := range rows {
		if r.SprintKey == "S911" || r.SprintKey == "S912" || r.SprintKey == "S913" {
			testRows = append(testRows, r)
		}
	}

	assert.Len(t, testRows, 2, "limit=2 should return 2 of our 3 test sprints")

	// Oldest-first: the 2 most recent are S912 then S913.
	if len(testRows) == 2 {
		keys := []string{testRows[0].SprintKey, testRows[1].SprintKey}
		assert.Contains(t, keys, "S912")
		assert.Contains(t, keys, "S913")
		assert.NotContains(t, keys, "S911", "S911 is not among the 2 most recent")
	}
}

// ============================================================================
// TestGetVelocityData_Empty
// Verifies that GetVelocityData returns an empty slice (not error) when no
// completed sprints exist with the test keys.
// ============================================================================

func TestGetVelocityData_Empty(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	// No completed sprints with key prefix TEST-EMPTY-S in DB.
	// Call with limit=0 — should return empty or only existing completed sprints.
	// Since we can't guarantee no other sprints exist, we call with a very small
	// limit and verify no error is returned.
	rows, err := repo.GetVelocityData(ctx, 0)
	assert.NoError(t, err)
	assert.NotNil(t, rows, "should return empty slice, not nil, on no results")
}

// ============================================================================
// TestGetVelocityData_OnlyCompletedSprints
// Verifies that sprints in planning/active/closing status are excluded.
// ============================================================================

func TestGetVelocityData_OnlyCompletedSprints(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key IN ('S921', 'S922')`)

	size2 := 2
	taskID8 := seedTaskWithSize(t, ctx, 8, &size2)
	taskID9 := seedTaskWithSize(t, ctx, 9, &size2)

	now := time.Now().UTC()

	// One completed sprint and one active sprint.
	sprintID1 := seedSprintWithAssignments(t, nil, ctx, "S921", "completed",
		now.AddDate(0, -2, 0), now.AddDate(0, -1, 0),
		[]seedEntity{{EntityType: "task", EntityID: taskID8, Size: &size2}})
	sprintID2 := seedSprintWithAssignments(t, nil, ctx, "S922", "active",
		now.AddDate(0, 0, -7), now.AddDate(0, 0, 7),
		[]seedEntity{{EntityType: "task", EntityID: taskID9, Size: &size2}})

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id IN (?, ?)`, sprintID1, sprintID2)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key IN ('S921', 'S922')`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key IN ('TEST-E99-F01-008','TEST-E99-F01-009')`)
	}()

	rows, err := repo.GetVelocityData(ctx, 10)
	require.NoError(t, err)

	var sawS921, sawS922 bool
	for _, r := range rows {
		if r.SprintKey == "S921" {
			sawS921 = true
		}
		if r.SprintKey == "S922" {
			sawS922 = true
		}
	}

	assert.True(t, sawS921, "completed sprint S921 should be in velocity results")
	assert.False(t, sawS922, "active sprint S922 must NOT be in velocity results")
}

// ============================================================================
// TestGetSprintAssignedEntities
// Verifies that GetSprintAssignedEntities returns all assignments including
// soft-deleted (removed_at IS NOT NULL) for a sprint.
// ============================================================================

func TestGetSprintAssignedEntities(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S931'`)

	size5 := 5
	taskID10 := seedTaskWithSize(t, ctx, 10, &size5)
	taskID11 := seedTaskWithSize(t, ctx, 11, nil) // unsized

	now := time.Now().UTC()
	start := now.AddDate(0, -1, 0)
	end := now.AddDate(0, 0, 14)
	removedAt := now.AddDate(0, 0, -3)

	sprintID := seedSprintWithAssignments(t, nil, ctx, "S931", "active",
		start, end,
		[]seedEntity{
			// Active assignment with size.
			{EntityType: "task", EntityID: taskID10, Size: &size5, AssignedAt: start},
			// Soft-deleted assignment (removed mid-sprint), unsized.
			{EntityType: "task", EntityID: taskID11, Size: nil, AssignedAt: start, RemovedAt: &removedAt},
		})

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S931'`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key IN ('TEST-E99-F01-010','TEST-E99-F01-011')`)
	}()

	entities, err := repo.GetSprintAssignedEntities(ctx, sprintID)
	require.NoError(t, err)
	require.Len(t, entities, 2, "should return both active and removed assignments")

	// Find each entity.
	var active, removed *AssignedEntity
	for i := range entities {
		if entities[i].EntityID == taskID10 {
			active = &entities[i]
		}
		if entities[i].EntityID == taskID11 {
			removed = &entities[i]
		}
	}

	require.NotNil(t, active, "active assignment should be returned")
	assert.Nil(t, active.RemovedAt, "active assignment has no removed_at")
	require.NotNil(t, active.Size, "active assignment has size")
	assert.Equal(t, 5, *active.Size)

	require.NotNil(t, removed, "removed assignment should be returned")
	require.NotNil(t, removed.RemovedAt, "removed assignment has removed_at set")
	assert.Nil(t, removed.Size, "removed assignment has nil size (unsized)")
}

// ============================================================================
// TestGetSprintAssignedEntities_Empty
// Verifies that an empty slice (not error) is returned for a sprint with no
// assignments.
// ============================================================================

func TestGetSprintAssignedEntities_Empty(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S941'`)

	now := time.Now().UTC()
	result, err := rawDB.ExecContext(ctx, `
		INSERT INTO sprints (key, name, goal, start_date, end_date, status, slug)
		VALUES ('S941', 'Empty Sprint', 'none', ?, ?, 'planning', 'empty-sprint')
	`, now.AddDate(0, 1, 0), now.AddDate(0, 2, 0))
	require.NoError(t, err)
	sprintID, _ := result.LastInsertId()

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S941'`)
	}()

	entities, err := repo.GetSprintAssignedEntities(ctx, sprintID)
	assert.NoError(t, err)
	assert.NotNil(t, entities, "should return empty slice, not nil")
	assert.Len(t, entities, 0)
}

// ============================================================================
// TestGetCompletionEvents
// Verifies that GetCompletionEvents returns task_history events within the
// sprint window, and filters out events outside the window.
// ============================================================================

func TestGetCompletionEvents(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S951'`)

	size5 := 5
	taskID12 := seedTaskWithSize(t, ctx, 12, &size5)
	taskID13 := seedTaskWithSize(t, ctx, 13, &size5)

	now := time.Now().UTC()
	start := now.AddDate(0, -1, -7)
	end := now.AddDate(0, -1, 7)

	sprintID := seedSprintWithAssignments(t, nil, ctx, "S951", "completed",
		start, end,
		[]seedEntity{
			{EntityType: "task", EntityID: taskID12, Size: &size5, AssignedAt: start},
			{EntityType: "task", EntityID: taskID13, Size: &size5, AssignedAt: start},
		})

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM task_history WHERE task_id IN (?, ?)`, taskID12, taskID13)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S951'`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key IN ('TEST-E99-F01-012','TEST-E99-F01-013')`)
	}()

	// Event within window: task12 completed on day 3 of the sprint.
	withinWindow := start.AddDate(0, 0, 3)
	// Event outside window: task13 completed after the sprint ended.
	afterWindow := end.AddDate(0, 0, 1)

	seedTaskHistoryEvents(t, ctx, []seedTaskHistoryEvent{
		{TaskID: taskID12, OldStatus: "in_progress", NewStatus: "completed", Timestamp: withinWindow},
		{TaskID: taskID13, OldStatus: "in_progress", NewStatus: "completed", Timestamp: afterWindow},
	})

	events, err := repo.GetCompletionEvents(ctx, sprintID, start, end)
	require.NoError(t, err)

	// Only the within-window event should be returned.
	var foundTask12, foundTask13 bool
	for _, ev := range events {
		if ev.EntityID == taskID12 {
			foundTask12 = true
		}
		if ev.EntityID == taskID13 {
			foundTask13 = true
		}
	}
	assert.True(t, foundTask12, "task12 (completed within window) should be in events")
	assert.False(t, foundTask13, "task13 (completed after window) must NOT be in events")
}

// ============================================================================
// TestGetCycleTimeByPhase
// Verifies that GetCycleTimeByPhase returns non-empty slice when task_history
// has transitions, and empty slice (not error) when there are none.
// ============================================================================

func TestGetCycleTimeByPhase_WithHistory(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S961'`)

	size5 := 5
	taskID14 := seedTaskWithSize(t, ctx, 14, &size5)

	now := time.Now().UTC()
	start := now.AddDate(0, -2, 0)
	end := now.AddDate(0, -1, 0)

	sprintID := seedSprintWithAssignments(t, nil, ctx, "S961", "completed",
		start, end,
		[]seedEntity{
			{EntityType: "task", EntityID: taskID14, Size: &size5, AssignedAt: start},
		})

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM task_history WHERE task_id = ?`, taskID14)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprint_assignments WHERE sprint_id = ?`, sprintID)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S961'`)
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'TEST-E99-F01-014'`)
	}()

	// Seed two task_history events so we have phase data.
	t1 := start.AddDate(0, 0, 2)
	t2 := start.AddDate(0, 0, 5)
	seedTaskHistoryEvents(t, ctx, []seedTaskHistoryEvent{
		{TaskID: taskID14, OldStatus: "todo", NewStatus: "in_progress", Timestamp: t1},
		{TaskID: taskID14, OldStatus: "in_progress", NewStatus: "completed", Timestamp: t2},
	})

	phases, err := repo.GetCycleTimeByPhase(ctx, sprintID)
	require.NoError(t, err)
	// With task history, we should get at least one phase row.
	assert.NotEmpty(t, phases, "expected non-empty phase rows when task_history exists")
}

func TestGetCycleTimeByPhase_NoHistory(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()
	db := dbconn.NewDB(rawDB)
	repo := NewSprintAnalyticsRepository(db)

	_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S971'`)

	now := time.Now().UTC()
	result, err := rawDB.ExecContext(ctx, `
		INSERT INTO sprints (key, name, goal, start_date, end_date, status, slug)
		VALUES ('S971', 'No History Sprint', 'none', ?, ?, 'completed', 'no-history')
	`, now.AddDate(0, -2, 0), now.AddDate(0, -1, 0))
	require.NoError(t, err)
	sprintID, _ := result.LastInsertId()

	defer func() {
		_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = 'S971'`)
	}()

	phases, err := repo.GetCycleTimeByPhase(ctx, sprintID)
	assert.NoError(t, err)
	// No history = empty slice, not an error.
	assert.Empty(t, phases, "expected empty slice when no task_history rows exist for the sprint")
}

// ============================================================================
// TC-NF-02: EXPLAIN QUERY PLAN confirms indexed lookups for core analytics queries
// ============================================================================

func TestExplainQueryPlan_VelocityData(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()

	// Run EXPLAIN QUERY PLAN on the inner query that hits sprint_assignments.
	// The presence of "sprint_assignments" with USING INDEX (not SCAN TABLE)
	// validates TC-NF-02.
	query := `
		EXPLAIN QUERY PLAN
		SELECT s.key, s.name,
			COALESCE(SUM(CASE WHEN t.size IS NOT NULL THEN t.size ELSE 0 END), 0) AS completed_size,
			COALESCE(SUM(CASE WHEN t.size IS NULL THEN 1 ELSE 0 END), 0) AS unsized_completed
		FROM sprints s
		JOIN sprint_assignments sa ON sa.sprint_id = s.id
		LEFT JOIN tasks t ON t.id = sa.entity_id AND sa.entity_type = 'task'
		WHERE s.status = 'completed'
		GROUP BY s.id, s.key, s.name
		ORDER BY s.end_date ASC
		LIMIT ?
	`

	rows, err := rawDB.QueryContext(ctx, query, 5)
	require.NoError(t, err)
	defer rows.Close()

	var plan string
	var foundIndexedLookup bool
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			// SQLite EXPLAIN QUERY PLAN columns vary — just scan detail.
			_ = rows.Scan(&detail)
		}
		plan += detail + "\n"
		// An index scan on sprint_assignments will contain "USING INDEX".
		if containsStr(detail, "sprint_assignments") {
			if containsStr(detail, "USING INDEX") || containsStr(detail, "idx_sprint_assignments") {
				foundIndexedLookup = true
			}
		}
	}
	_ = rows.Err()

	_ = plan // for debugging
	assert.True(t, foundIndexedLookup,
		"EXPLAIN QUERY PLAN must show indexed lookup on sprint_assignments, got:\n%s", plan)
}

func TestExplainQueryPlan_AssignedEntities(t *testing.T) {
	ctx := context.Background()
	rawDB := test.GetTestDB()

	query := `
		EXPLAIN QUERY PLAN
		SELECT sa.entity_type, sa.entity_id, sa.assigned_at, sa.removed_at,
			CASE sa.entity_type
				WHEN 'task' THEN t.size
				WHEN 'bug' THEN b.size
				WHEN 'change_card' THEN cc.size
				WHEN 'tech_debt' THEN td.size
				ELSE NULL
			END AS size
		FROM sprint_assignments sa
		LEFT JOIN tasks t ON t.id = sa.entity_id AND sa.entity_type = 'task'
		LEFT JOIN bugs b ON b.id = sa.entity_id AND sa.entity_type = 'bug'
		LEFT JOIN change_cards cc ON cc.id = sa.entity_id AND sa.entity_type = 'change_card'
		LEFT JOIN tech_debts td ON td.id = sa.entity_id AND sa.entity_type = 'tech_debt'
		WHERE sa.sprint_id = ?
	`

	rows, err := rawDB.QueryContext(ctx, query, int64(1))
	require.NoError(t, err)
	defer rows.Close()

	var foundIndexedLookup bool
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			_ = rows.Scan(&detail)
		}
		if containsStr(detail, "sprint_assignments") {
			if containsStr(detail, "USING INDEX") || containsStr(detail, "idx_sprint_assignments") {
				foundIndexedLookup = true
			}
		}
	}
	_ = rows.Err()

	assert.True(t, foundIndexedLookup,
		"EXPLAIN QUERY PLAN for GetSprintAssignedEntities must show indexed lookup on sprint_assignments")
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

	defer func() {
		for _, key := range sprintKeys {
			_, _ = rawDB.ExecContext(ctx,
				`DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = ?)`, key)
			_, _ = rawDB.ExecContext(ctx, `DELETE FROM sprints WHERE key = ?`, key)
		}
	}()

	taskNums := make([]int, 0, numTasks)
	for i := 0; i < numTasks; i++ {
		taskNum := 5000 + i
		taskID := seedTaskForAnalytics(t, ctx, taskNum, nil)
		sprintID := sprintIDs[i%numSprints]
		assignedAt := now.AddDate(0, -(numSprints - (i % numSprints)), 0)
		addAssignment(t, ctx, sprintID, "task", taskID, assignedAt, nil)
		taskNums = append(taskNums, taskNum)
	}

	defer func() {
		for _, n := range taskNums {
			taskKey := fmt.Sprintf("ANTEST-E99-F01-%03d", n)
			_, _ = rawDB.ExecContext(ctx,
				`DELETE FROM task_history WHERE task_id IN (SELECT id FROM tasks WHERE key = ?)`, taskKey)
			_, _ = rawDB.ExecContext(ctx, `DELETE FROM tasks WHERE key = ?`, taskKey)
		}
	}()

	const maxDuration = 2 * time.Second

	t.Run("GetVelocityData", func(t *testing.T) {
		start := time.Now()
		_, err := repo.GetVelocityData(ctx, numSprints)
		elapsed := time.Since(start)

		require.NoError(t, err, "GetVelocityData must not error")
		assert.Less(t, elapsed, maxDuration,
			"GetVelocityData must complete in < 2s, took %s", elapsed)
	})

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

	t.Run("GetCycleTimeByPhase", func(t *testing.T) {
		sprintID := sprintIDs[0]
		taskID := seedTaskForAnalytics(t, ctx, 9999, nil)
		t1 := now.AddDate(0, -2, 0)
		t2 := t1.AddDate(0, 0, 3)
		seedTaskHistoryEvents(t, ctx, []seedTaskHistoryEvent{
			{TaskID: taskID, OldStatus: "todo", NewStatus: "in_progress", Timestamp: t1},
			{TaskID: taskID, OldStatus: "in_progress", NewStatus: "completed", Timestamp: t2},
		})

		addAssignment(t, ctx, sprintID, "task", taskID, t1, nil)

		start := time.Now()
		_, err := repo.GetCycleTimeByPhase(ctx, sprintID)
		elapsed := time.Since(start)

		require.NoError(t, err, "GetCycleTimeByPhase must not error")
		assert.Less(t, elapsed, maxDuration,
			"GetCycleTimeByPhase must complete in < 2s, took %s", elapsed)
	})
}

// containsStr is a helper since strings.Contains requires importing strings.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}
