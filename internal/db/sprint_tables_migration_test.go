package db

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

// setupSprintTestDB initializes a fresh test database, then runs migrateSprintTables
// directly (it is not yet wired into runMigrations until T-E19-F01-007). The returned
// cleanup must be deferred. All sprint-table tests share this setup helper because
// the migration is intentionally not invoked by InitDB until T-007 lands.
func setupSprintTestDB(t *testing.T, tmpFile string) (*sql.DB, func()) {
	t.Helper()

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Run the sprint migration directly. It uses CREATE TABLE/INDEX/TRIGGER IF NOT
	// EXISTS so it is idempotent.
	if err := migrateSprintTables(db); err != nil {
		_ = db.Close()
		t.Fatalf("migrateSprintTables failed: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.Remove(tmpFile)
		_ = os.Remove(tmpFile + "-shm")
		_ = os.Remove(tmpFile + "-wal")
	}
	return db, cleanup
}

// TestMigrateSprintTables_CreatesSprintAssignmentsTable verifies the sprint_assignments
// table is created with the polymorphic columns from spec §3.3 (T-E19-F01-005 AC).
func TestMigrateSprintTables_CreatesSprintAssignmentsTable(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_table.db")
	defer cleanup()

	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sprint_assignments'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sqlite_master: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected sprint_assignments table to exist, got count=%d", count)
	}

	// Verify the polymorphic column set: sprint_id, entity_type, entity_id,
	// assigned_at, removed_at — and explicitly NO task_id / bug_id / etc. columns.
	rows, err := db.Query(`PRAGMA table_info(sprint_assignments);`)
	if err != nil {
		t.Fatalf("PRAGMA table_info failed: %v", err)
	}
	defer rows.Close()

	got := map[string]bool{}
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan PRAGMA row: %v", err)
		}
		got[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	want := []string{"id", "sprint_id", "entity_type", "entity_id", "assigned_at", "removed_at"}
	for _, col := range want {
		if !got[col] {
			t.Errorf("Expected sprint_assignments to have column %q, got columns: %v", col, got)
		}
	}
	// AC-3: polymorphic — must NOT have entity-specific FK columns.
	forbidden := []string{"task_id", "bug_id", "change_card_id", "tech_debt_id"}
	for _, col := range forbidden {
		if got[col] {
			t.Errorf("sprint_assignments must be polymorphic; column %q must not exist (AC-3)", col)
		}
	}
}

// TestMigrateSprintTables_CreatesSprintAssignmentsIndexes verifies the three required
// indexes on sprint_assignments exist (T-E19-F01-005 AC).
func TestMigrateSprintTables_CreatesSprintAssignmentsIndexes(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_indexes.db")
	defer cleanup()

	wanted := []string{
		"idx_sprint_assignments_sprint",
		"idx_sprint_assignments_entity",
		"idx_sprint_assignments_active_one",
	}
	for _, idx := range wanted {
		var count int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`,
			idx,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query for index %s: %v", idx, err)
		}
		if count != 1 {
			t.Errorf("Expected index %s to exist, got count=%d", idx, count)
		}
	}

	// Verify the partial unique index has the WHERE removed_at IS NULL clause.
	var idxSQL string
	err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='index' AND name='idx_sprint_assignments_active_one'`,
	).Scan(&idxSQL)
	if err != nil {
		t.Fatalf("query partial index sql: %v", err)
	}
	if !strings.Contains(strings.ToUpper(idxSQL), "WHERE REMOVED_AT IS NULL") {
		t.Errorf("Expected idx_sprint_assignments_active_one to be a partial index "+
			"with 'WHERE removed_at IS NULL', got SQL: %s", idxSQL)
	}
	if !strings.Contains(strings.ToUpper(idxSQL), "UNIQUE") {
		t.Errorf("Expected idx_sprint_assignments_active_one to be UNIQUE, got SQL: %s", idxSQL)
	}
}

// TestMigrateSprintTables_SingleActiveSprintIndex verifies the sprint table
// enforces exactly one active sprint at the database layer.
func TestMigrateSprintTables_SingleActiveSprintIndex(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_single_active.db")
	defer cleanup()

	// First active sprint succeeds.
	if _, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date, status)
		VALUES (?, ?, ?, ?, ?)
	`, "S100", "Sprint 100", "2026-04-01", "2026-04-15", "active"); err != nil {
		t.Fatalf("first active sprint insert should succeed, got: %v", err)
	}

	// Second active sprint must fail because of the partial unique index.
	_, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date, status)
		VALUES (?, ?, ?, ?, ?)
	`, "S101", "Sprint 101", "2026-04-16", "2026-04-30", "active")
	if err == nil {
		t.Fatal("expected unique constraint failure on second active sprint, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Errorf("expected UNIQUE constraint error, got: %v", err)
	}

	// Non-active sprint remains allowed.
	if _, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date, status)
		VALUES (?, ?, ?, ?, ?)
	`, "S102", "Sprint 102", "2026-05-01", "2026-05-15", "planning"); err != nil {
		t.Errorf("planning sprint insert should succeed, got: %v", err)
	}
}

// TestMigrateSprintTables_PartialUniqueIndex verifies the one-active-sprint-per-entity
// integrity guarantee — the most critical assertion for downstream features.
//
// The index is partial: it constrains uniqueness only over rows with
// removed_at IS NULL. Two active rows for the same (entity_type, entity_id) must
// conflict; the same insert with one removed_at set must succeed (T-E19-F01-005 AC).
func TestMigrateSprintTables_PartialUniqueIndex(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_partial_unique.db")
	defer cleanup()

	// Insert a sprint to satisfy the FK on sprint_assignments.sprint_id.
	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S001", "Sprint One", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}

	// First active row — succeeds.
	if _, err := db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "task", 42); err != nil {
		t.Fatalf("first active insert should succeed, got: %v", err)
	}

	// Second active row with the SAME (entity_type, entity_id) — must fail.
	_, err = db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "task", 42)
	if err == nil {
		t.Fatal("expected UNIQUE constraint failure on second active row, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Errorf("expected UNIQUE constraint error, got: %v", err)
	}

	// Soft-delete the existing row, then re-insert — must succeed (the partial
	// index only constrains rows where removed_at IS NULL).
	if _, err := db.Exec(`
		UPDATE sprint_assignments SET removed_at = CURRENT_TIMESTAMP
		WHERE entity_type='task' AND entity_id=42
	`); err != nil {
		t.Fatalf("soft-delete update: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "task", 42); err != nil {
		t.Errorf("insert after soft-delete should succeed (partial unique only "+
			"applies where removed_at IS NULL), got: %v", err)
	}
}

// TestMigrateSprintTables_NoEntityTypeCheckConstraint verifies the post-B018
// convention is honoured: the DB has no CHECK on entity_type, so raw inserts of
// arbitrary strings succeed at the DB layer. App-layer validation
// (ValidateSprintAssignmentEntityType in models/validation.go) is the sole gate.
func TestMigrateSprintTables_NoEntityTypeCheckConstraint(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_no_check.db")
	defer cleanup()

	// Verify the schema text contains no `entity_type IN (` clause.
	var tableSQL string
	err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='sprint_assignments'`,
	).Scan(&tableSQL)
	if err != nil {
		t.Fatalf("query sprint_assignments schema: %v", err)
	}
	if strings.Contains(tableSQL, "entity_type IN") {
		t.Errorf("Per the post-B018 convention, sprint_assignments must NOT have an "+
			"entity_type CHECK constraint. Found 'entity_type IN' in schema: %s", tableSQL)
	}

	// Insert a sprint to satisfy the FK.
	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S002", "Sprint Two", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}

	// Raw insert with an arbitrary entity_type — succeeds at DB layer (app validates).
	if _, err := db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "whatever", 1); err != nil {
		t.Errorf("Post-B018 the DB has no entity_type CHECK; raw insert with "+
			"arbitrary entity_type should succeed at the DB layer "+
			"(app-layer validation rejects it instead). Got: %v", err)
	}
}

// TestMigrateSprintTables_CascadeDeleteFromTask verifies the task-parent cascade
// trigger removes sprint_assignments rows when the parent task is deleted.
func TestMigrateSprintTables_CascadeDeleteFromTask(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_cascade_task.db")
	defer cleanup()

	// Need an epic + feature to satisfy task FKs.
	if _, err := db.Exec(
		`INSERT INTO epics (key, title, status, priority) VALUES (?, ?, ?, ?)`,
		"E99", "Test Epic", "todo", "medium"); err != nil {
		t.Fatalf("insert epic: %v", err)
	}
	var epicID int64
	if err := db.QueryRow(`SELECT id FROM epics WHERE key='E99'`).Scan(&epicID); err != nil {
		t.Fatalf("get epic id: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO features (key, epic_id, title, status) VALUES (?, ?, ?, ?)`,
		"E99-F01", epicID, "Test Feature", "todo"); err != nil {
		t.Fatalf("insert feature: %v", err)
	}
	var featureID int64
	if err := db.QueryRow(`SELECT id FROM features WHERE key='E99-F01'`).Scan(&featureID); err != nil {
		t.Fatalf("get feature id: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO tasks (key, feature_id, title, status) VALUES (?, ?, ?, ?)`,
		"T-E99-F01-001", featureID, "Test Task", "todo"); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	var taskID int64
	if err := db.QueryRow(
		`SELECT id FROM tasks WHERE key='T-E99-F01-001'`).Scan(&taskID); err != nil {
		t.Fatalf("get task id: %v", err)
	}

	// Sprint and assignment.
	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S010", "Cascade Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "task", taskID); err != nil {
		t.Fatalf("insert sprint_assignment: %v", err)
	}

	// Sanity check before deletion.
	var before int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sprint_assignments WHERE entity_type='task' AND entity_id=?`,
		taskID,
	).Scan(&before); err != nil {
		t.Fatalf("count before: %v", err)
	}
	if before != 1 {
		t.Fatalf("expected 1 sprint_assignment row before delete, got %d", before)
	}

	// Delete the parent task — trigger should cascade.
	if _, err := db.Exec(`DELETE FROM tasks WHERE id=?`, taskID); err != nil {
		t.Fatalf("delete task: %v", err)
	}

	var after int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sprint_assignments WHERE entity_type='task' AND entity_id=?`,
		taskID,
	).Scan(&after); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != 0 {
		t.Errorf("Expected cascade-delete trigger to remove sprint_assignments for "+
			"deleted task; got %d rows remaining", after)
	}
}

// TestMigrateSprintTables_CascadeDeleteFromBug verifies the bug-parent cascade trigger.
func TestMigrateSprintTables_CascadeDeleteFromBug(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_cascade_bug.db")
	defer cleanup()

	if _, err := db.Exec(
		`INSERT INTO bugs (key, title) VALUES (?, ?)`,
		"B901", "Cascade Bug"); err != nil {
		t.Fatalf("insert bug: %v", err)
	}
	var bugID int64
	if err := db.QueryRow(
		`SELECT id FROM bugs WHERE key='B901'`).Scan(&bugID); err != nil {
		t.Fatalf("get bug id: %v", err)
	}

	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S011", "Bug Cascade Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "bug", bugID); err != nil {
		t.Fatalf("insert sprint_assignment: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM bugs WHERE id=?`, bugID); err != nil {
		t.Fatalf("delete bug: %v", err)
	}

	var after int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sprint_assignments WHERE entity_type='bug' AND entity_id=?`,
		bugID,
	).Scan(&after); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != 0 {
		t.Errorf("Expected cascade-delete trigger to remove sprint_assignments for "+
			"deleted bug; got %d rows remaining", after)
	}
}

// TestMigrateSprintTables_CascadeDeleteFromChangeCard verifies the change-card-parent
// cascade trigger.
func TestMigrateSprintTables_CascadeDeleteFromChangeCard(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_cascade_change.db")
	defer cleanup()

	if _, err := db.Exec(
		`INSERT INTO change_cards (key, title) VALUES (?, ?)`,
		"CC-901", "Cascade Change"); err != nil {
		t.Fatalf("insert change_card: %v", err)
	}
	var ccID int64
	if err := db.QueryRow(
		`SELECT id FROM change_cards WHERE key='CC-901'`).Scan(&ccID); err != nil {
		t.Fatalf("get change_card id: %v", err)
	}

	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S012", "Change Cascade Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "change_card", ccID); err != nil {
		t.Fatalf("insert sprint_assignment: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM change_cards WHERE id=?`, ccID); err != nil {
		t.Fatalf("delete change_card: %v", err)
	}

	var after int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sprint_assignments WHERE entity_type='change_card' AND entity_id=?`,
		ccID,
	).Scan(&after); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != 0 {
		t.Errorf("Expected cascade-delete trigger to remove sprint_assignments for "+
			"deleted change_card; got %d rows remaining", after)
	}
}

// TestMigrateSprintTables_CascadeDeleteFromTechDebt verifies the tech_debt-parent
// cascade trigger. Easy to miss — tech_debts is the most-recently-added parent.
func TestMigrateSprintTables_CascadeDeleteFromTechDebt(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_cascade_techdebt.db")
	defer cleanup()

	if _, err := db.Exec(
		`INSERT INTO tech_debts (key, title) VALUES (?, ?)`,
		"TD-901", "Cascade Tech Debt"); err != nil {
		t.Fatalf("insert tech_debt: %v", err)
	}
	var tdID int64
	if err := db.QueryRow(
		`SELECT id FROM tech_debts WHERE key='TD-901'`).Scan(&tdID); err != nil {
		t.Fatalf("get tech_debt id: %v", err)
	}

	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S013", "Tech Debt Cascade Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "tech_debt", tdID); err != nil {
		t.Fatalf("insert sprint_assignment: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM tech_debts WHERE id=?`, tdID); err != nil {
		t.Fatalf("delete tech_debt: %v", err)
	}

	var after int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sprint_assignments WHERE entity_type='tech_debt' AND entity_id=?`,
		tdID,
	).Scan(&after); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != 0 {
		t.Errorf("Expected cascade-delete trigger to remove sprint_assignments for "+
			"deleted tech_debt; got %d rows remaining", after)
	}
}

// TestMigrateSprintTables_CascadeFKOnSprintDelete verifies that deleting a sprint
// cascades to its sprint_assignments rows via the table-level FK
// (FOREIGN KEY (sprint_id) REFERENCES sprints(id) ON DELETE CASCADE).
// This is the only FK-driven cascade on this table; the four entity_type
// triggers handle the polymorphic side.
func TestMigrateSprintTables_CascadeFKOnSprintDelete(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_fk_cascade.db")
	defer cleanup()

	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S020", "FK Cascade Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id)
		VALUES (?, ?, ?)
	`, sprintID, "task", 1); err != nil {
		t.Fatalf("insert sprint_assignment: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM sprints WHERE id=?`, sprintID); err != nil {
		t.Fatalf("delete sprint: %v", err)
	}

	var remaining int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sprint_assignments WHERE sprint_id=?`,
		sprintID,
	).Scan(&remaining); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if remaining != 0 {
		t.Errorf("Expected FK ON DELETE CASCADE to remove sprint_assignments for "+
			"deleted sprint; got %d rows remaining", remaining)
	}
}

// TestMigrateSprintTables_CreatesSprintCapacityTable verifies the sprint_capacity
// table is created with the columns from spec §3.3 (T-E19-F01-006 AC).
//
// Required columns (per AC and spec §3.3): id, sprint_id, agent_type,
// capacity_points, allocated_points, created_at, updated_at.
// allocated_points must be intentionally nullable — NULL means "not yet computed".
func TestMigrateSprintTables_CreatesSprintCapacityTable(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_capacity_table.db")
	defer cleanup()

	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sprint_capacity'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sqlite_master: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected sprint_capacity table to exist, got count=%d", count)
	}

	// Verify the column set matches spec §3.3 exactly.
	rows, err := db.Query(`PRAGMA table_info(sprint_capacity);`)
	if err != nil {
		t.Fatalf("PRAGMA table_info failed: %v", err)
	}
	defer rows.Close()

	type colInfo struct {
		notnull int
	}
	got := map[string]colInfo{}
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan PRAGMA row: %v", err)
		}
		got[name] = colInfo{notnull: notnull}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	want := []string{"id", "sprint_id", "agent_type", "capacity_points", "allocated_points", "created_at", "updated_at"}
	for _, col := range want {
		if _, ok := got[col]; !ok {
			t.Errorf("Expected sprint_capacity to have column %q, got columns: %v", col, got)
		}
	}

	// allocated_points must be nullable (notnull=0). NULL means "not yet computed".
	if info, ok := got["allocated_points"]; ok {
		if info.notnull != 0 {
			t.Errorf("Expected allocated_points to be nullable (notnull=0), got notnull=%d", info.notnull)
		}
	}
}

// TestMigrateSprintTables_CreatesSprintCapacityIndex verifies the
// idx_sprint_capacity_sprint index exists (T-E19-F01-006 AC).
func TestMigrateSprintTables_CreatesSprintCapacityIndex(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_capacity_index.db")
	defer cleanup()

	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_sprint_capacity_sprint'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query for index idx_sprint_capacity_sprint: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected index idx_sprint_capacity_sprint to exist, got count=%d", count)
	}
}

// TestMigrateSprintTables_SprintCapacityUpdatedAtTrigger verifies the
// sprint_capacity_updated_at trigger exists and updates the column on UPDATE.
func TestMigrateSprintTables_SprintCapacityUpdatedAtTrigger(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_capacity_updated_at.db")
	defer cleanup()

	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='sprint_capacity_updated_at'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query for trigger sprint_capacity_updated_at: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected trigger sprint_capacity_updated_at to exist, got count=%d", count)
	}

	// Functional check: insert a row, read updated_at, sleep, update, read again — must change.
	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S100", "Capacity Trigger Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	if _, err := db.Exec(
		`INSERT INTO sprint_capacity (sprint_id, agent_type, capacity_points) VALUES (?, ?, ?)`,
		sprintID, "developer", 10.0,
	); err != nil {
		t.Fatalf("insert sprint_capacity: %v", err)
	}

	var beforeUpdated string
	if err := db.QueryRow(
		`SELECT updated_at FROM sprint_capacity WHERE sprint_id=? AND agent_type='developer'`,
		sprintID,
	).Scan(&beforeUpdated); err != nil {
		t.Fatalf("read updated_at before: %v", err)
	}

	// SQLite's CURRENT_TIMESTAMP is second-resolution. Force a different timestamp
	// so the trigger's effect is observable without a real-time sleep.
	if _, err := db.Exec(
		`UPDATE sprint_capacity SET capacity_points = ?, updated_at = '2000-01-01 00:00:00'
		 WHERE sprint_id=? AND agent_type='developer'`,
		20.0, sprintID,
	); err != nil {
		t.Fatalf("update sprint_capacity: %v", err)
	}

	var afterUpdated string
	if err := db.QueryRow(
		`SELECT updated_at FROM sprint_capacity WHERE sprint_id=? AND agent_type='developer'`,
		sprintID,
	).Scan(&afterUpdated); err != nil {
		t.Fatalf("read updated_at after: %v", err)
	}

	// The trigger fires AFTER UPDATE, so even though the UPDATE explicitly set
	// updated_at to a 2000-era timestamp, the trigger overwrites it back to
	// CURRENT_TIMESTAMP. afterUpdated must NOT be the 2000 stamp we wrote.
	if afterUpdated == "2000-01-01 00:00:00" {
		t.Errorf("Expected sprint_capacity_updated_at trigger to overwrite manual "+
			"updated_at='2000-01-01 00:00:00' with CURRENT_TIMESTAMP, but got %q (trigger did not fire)",
			afterUpdated)
	}
}

// TestMigrateSprintTables_SprintCapacityUniqueConstraint verifies the
// UNIQUE(sprint_id, agent_type) constraint rejects duplicate (sprint, agent) pairs
// and that the constraint is declared in-table (not as a separate index).
func TestMigrateSprintTables_SprintCapacityUniqueConstraint(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_capacity_unique.db")
	defer cleanup()

	// AC: the UNIQUE is a TABLE-LEVEL constraint declared inline in CREATE TABLE,
	// not a separate CREATE UNIQUE INDEX statement. Verify by inspecting the table SQL.
	var tableSQL string
	if err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='sprint_capacity'`,
	).Scan(&tableSQL); err != nil {
		t.Fatalf("query sprint_capacity schema: %v", err)
	}
	upper := strings.ToUpper(tableSQL)
	if !strings.Contains(upper, "UNIQUE") || !strings.Contains(upper, "SPRINT_ID") || !strings.Contains(upper, "AGENT_TYPE") {
		t.Errorf("Expected sprint_capacity CREATE TABLE statement to include "+
			"UNIQUE(sprint_id, agent_type) inline, got SQL: %s", tableSQL)
	}

	// Functional check: duplicate (sprint_id, agent_type) must fail.
	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S101", "Unique Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	if _, err := db.Exec(
		`INSERT INTO sprint_capacity (sprint_id, agent_type, capacity_points) VALUES (?, ?, ?)`,
		sprintID, "developer", 10.0,
	); err != nil {
		t.Fatalf("first insert should succeed: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO sprint_capacity (sprint_id, agent_type, capacity_points) VALUES (?, ?, ?)`,
		sprintID, "developer", 20.0,
	)
	if err == nil {
		t.Fatal("expected UNIQUE constraint failure on duplicate (sprint_id, agent_type), got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Errorf("expected UNIQUE constraint error, got: %v", err)
	}
	// AC: error message should reference the constraint columns.
	if !strings.Contains(err.Error(), "sprint_capacity.sprint_id") ||
		!strings.Contains(err.Error(), "sprint_capacity.agent_type") {
		t.Errorf("expected error to reference 'sprint_capacity.sprint_id' and "+
			"'sprint_capacity.agent_type', got: %v", err)
	}

	// Different agent_type for the same sprint — must succeed.
	if _, err := db.Exec(
		`INSERT INTO sprint_capacity (sprint_id, agent_type, capacity_points) VALUES (?, ?, ?)`,
		sprintID, "qa", 5.0,
	); err != nil {
		t.Errorf("different agent_type for same sprint should succeed, got: %v", err)
	}
}

// TestMigrateSprintTables_SprintCapacityFKCascade verifies that deleting a sprint
// cascades to its sprint_capacity rows via FOREIGN KEY (sprint_id) ON DELETE CASCADE.
func TestMigrateSprintTables_SprintCapacityFKCascade(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_capacity_fk_cascade.db")
	defer cleanup()

	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S102", "FK Cascade Capacity Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	for _, agent := range []string{"developer", "qa", "ba"} {
		if _, err := db.Exec(
			`INSERT INTO sprint_capacity (sprint_id, agent_type, capacity_points) VALUES (?, ?, ?)`,
			sprintID, agent, 10.0,
		); err != nil {
			t.Fatalf("insert sprint_capacity for %s: %v", agent, err)
		}
	}

	var before int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sprint_capacity WHERE sprint_id=?`, sprintID,
	).Scan(&before); err != nil {
		t.Fatalf("count before: %v", err)
	}
	if before != 3 {
		t.Fatalf("expected 3 sprint_capacity rows before delete, got %d", before)
	}

	if _, err := db.Exec(`DELETE FROM sprints WHERE id=?`, sprintID); err != nil {
		t.Fatalf("delete sprint: %v", err)
	}

	var after int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sprint_capacity WHERE sprint_id=?`, sprintID,
	).Scan(&after); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != 0 {
		t.Errorf("Expected FK ON DELETE CASCADE to remove sprint_capacity rows for "+
			"deleted sprint; got %d rows remaining", after)
	}
}

// TestMigrateSprintTables_SprintCapacityAllocatedPointsNullable verifies
// allocated_points accepts NULL (intentional nullable design — see spec §3.3
// and AC: "allocated_points REAL (nullable)").
func TestMigrateSprintTables_SprintCapacityAllocatedPointsNullable(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_capacity_nullable.db")
	defer cleanup()

	res, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S103", "Nullable Sprint", "2026-04-01", "2026-04-15")
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	sprintID, _ := res.LastInsertId()

	// Insert without specifying allocated_points — must succeed.
	if _, err := db.Exec(
		`INSERT INTO sprint_capacity (sprint_id, agent_type, capacity_points) VALUES (?, ?, ?)`,
		sprintID, "developer", 10.0,
	); err != nil {
		t.Fatalf("insert without allocated_points should succeed: %v", err)
	}

	var allocated sql.NullFloat64
	if err := db.QueryRow(
		`SELECT allocated_points FROM sprint_capacity WHERE sprint_id=? AND agent_type='developer'`,
		sprintID,
	).Scan(&allocated); err != nil {
		t.Fatalf("read allocated_points: %v", err)
	}
	if allocated.Valid {
		t.Errorf("Expected allocated_points to be NULL when not specified, got %v", allocated.Float64)
	}
}

// TestMigrateSprintTables_Idempotent verifies the migration is safe to run twice
// — required for the migration framework's re-run semantics.
func TestMigrateSprintTables_Idempotent(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_assignments_idempotent.db")
	defer cleanup()

	// Setup helper already ran the migration once; run it again and assert no error.
	if err := migrateSprintTables(db); err != nil {
		t.Fatalf("Second run of migrateSprintTables should be idempotent, got: %v", err)
	}

	// Verify the schema is still consistent after the second run.
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sprint_assignments'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected exactly 1 sprint_assignments table after re-run, got %d", count)
	}

	// And the cascade + updated_at triggers should still exist.
	for _, tr := range []string{
		"sprint_assignments_cascade_delete_task",
		"sprint_assignments_cascade_delete_bug",
		"sprint_assignments_cascade_delete_change_card",
		"sprint_assignments_cascade_delete_tech_debt",
		"sprint_capacity_updated_at",
	} {
		var trCount int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?`,
			tr,
		).Scan(&trCount); err != nil {
			t.Fatalf("query trigger %s: %v", tr, err)
		}
		if trCount != 1 {
			t.Errorf("Expected trigger %s to exist after re-run, got count=%d", tr, trCount)
		}
	}
}

// TestMigrateSprintTables_CreatesAllThreeTables is the consolidated AC check
// for spec §6.1 — verifies that all three sprint-related tables (sprints,
// sprint_assignments, sprint_capacity) exist after migrateSprintTables runs.
// Per-table column checks live in the dedicated _CreatesSprintAssignmentsTable
// and _CreatesSprintCapacityTable tests; this test exists specifically because
// the AC requires a single test that confirms presence of ALL three.
func TestMigrateSprintTables_CreatesAllThreeTables(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_all_three_tables.db")
	defer cleanup()

	wanted := []string{"sprints", "sprint_assignments", "sprint_capacity"}
	for _, table := range wanted {
		var count int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`,
			table,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query sqlite_master for table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Expected table %s to exist after migrateSprintTables, got count=%d", table, count)
		}
	}
}

// TestMigrateSprintTables_CreatesAllIndexes is the consolidated AC check for
// spec §6.1 — verifies all eight indexes created by migrateSprintTables exist
// by their exact names. The `idx_sprints_key` index is UNIQUE; the others are
// non-unique except `idx_sprints_active_one` and
// `idx_sprint_assignments_active_one`, which are partial UNIQUE indexes
// validated separately by dedicated tests.
func TestMigrateSprintTables_CreatesAllIndexes(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_all_indexes.db")
	defer cleanup()

	wanted := []string{
		"idx_sprints_key",
		"idx_sprints_status",
		"idx_sprints_slug",
		"idx_sprints_active_one",
		"idx_sprint_assignments_sprint",
		"idx_sprint_assignments_entity",
		"idx_sprint_assignments_active_one",
		"idx_sprint_capacity_sprint",
	}
	for _, idx := range wanted {
		var count int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`,
			idx,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query sqlite_master for index %s: %v", idx, err)
		}
		if count != 1 {
			t.Errorf("Expected index %s to exist after migrateSprintTables, got count=%d", idx, count)
		}
	}
}

// TestMigrateSprintTables_StartEndDateCheck verifies the table-level
// `CHECK (start_date < end_date)` constraint on the sprints table rejects
// rows whose end_date is on or before the start_date. The fixture uses
// start='2026-04-01', end='2026-03-18' (end before start) — the standard
// case enumerated in spec §6.1. SQLite returns an error containing
// "CHECK constraint failed" when a CHECK is violated.
func TestMigrateSprintTables_StartEndDateCheck(t *testing.T) {
	db, cleanup := setupSprintTestDB(t, "test_sprint_start_end_date_check.db")
	defer cleanup()

	// end_date is BEFORE start_date — must violate the CHECK constraint.
	_, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S500", "Bad Date Sprint", "2026-04-01", "2026-03-18")
	if err == nil {
		t.Fatal("expected CHECK constraint failure when end_date < start_date, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "check constraint failed") {
		t.Errorf("expected error containing 'CHECK constraint failed', got: %v", err)
	}

	// Sanity: a valid sprint where start_date < end_date succeeds (confirms the
	// failure above is from the CHECK, not a different schema problem).
	if _, err := db.Exec(`
		INSERT INTO sprints (key, name, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, "S501", "Good Date Sprint", "2026-04-01", "2026-04-15"); err != nil {
		t.Errorf("control insert with start_date < end_date should succeed, got: %v", err)
	}
}

// TestSchemaVersionBumpedTo18 is the AC-named alias for the schema-version
// bump check from T-E19-F01-008. It calls ApplySchemaAndMigrations on a fresh
// test database and asserts the recorded schema_version is at least 18 — the
// value pinned by E19-F01 (sprints, sprint_assignments, sprint_capacity).
//
// NOTE: The assertion uses GreaterOrEqual(18) rather than Equal(18) so that
// subsequent feature migrations (e.g., E19-F03 bumped to 19) do not break this
// E19-F01 guard. The AC for E19-F01 is "schema_version >= 18 after the sprint
// tables migration" — not "exactly 18". A more specific version-exact check is
// provided by TestMigration_SprintCompletions_SchemaVersion in db_test.go.
//
// A more general test (TestMigration_SchemaVersion in db_test.go) covers the
// same logic via InitDB; this version is named per the task spec and uses
// the explicit ApplySchemaAndMigrations path so the assertion fires whether
// or not the InitDB-level migration framework changes shape.
func TestSchemaVersionBumpedTo18(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test_schema_v18.db"

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// Re-run ApplySchemaAndMigrations to exercise the explicit path the AC
	// names. This is idempotent (every migration uses IF NOT EXISTS guards
	// or pre-checks) and re-bumps schema_version to CurrentSchemaVersion.
	if err := ApplySchemaAndMigrations(db); err != nil {
		t.Fatalf("ApplySchemaAndMigrations failed: %v", err)
	}

	version, err := getSchemaVersion(db)
	if err != nil {
		t.Fatalf("getSchemaVersion failed: %v", err)
	}
	// Assert >= 18: E19-F01 established version 18; subsequent features (E19-F03+)
	// will bump further. The E19-F01 AC is satisfied as long as the sprint tables
	// migration ran and the version is at or above 18.
	if version < 18 {
		t.Errorf("Expected schema_version >= 18 after ApplySchemaAndMigrations "+
			"(E19-F01 bumped CurrentSchemaVersion to 18 for sprints, "+
			"sprint_assignments, sprint_capacity); got %d", version)
	}
	if CurrentSchemaVersion < 18 {
		t.Errorf("Expected CurrentSchemaVersion constant >= 18, got %d", CurrentSchemaVersion)
	}
}
