package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigrateAddExecutionOrder(t *testing.T) {
	// Create a temporary database file
	tmpDB := "test_migrate_exec_order.db"
	defer os.Remove(tmpDB)
	defer os.Remove(tmpDB + "-shm")
	defer os.Remove(tmpDB + "-wal")

	// Initialize database with existing schema (without execution_order)
	db, err := InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// First, we need to remove execution_order if it exists (since our schema now includes it)
	// For testing, let's check if column exists before migration
	var hasExecutionOrder bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('features')
		WHERE name = 'execution_order'
	`).Scan(&hasExecutionOrder)
	if err != nil {
		t.Fatalf("Failed to check for execution_order column: %v", err)
	}

	// If execution_order already exists (new schema), skip migration test
	if hasExecutionOrder {
		t.Log("execution_order column already exists in schema, skipping migration test")

		// Test NULL handling
		testNullHandling(t, db)
		return
	}

	// Run migration
	if err := MigrateAddExecutionOrder(db); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify features table has execution_order column
	var featuresHasColumn bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('features')
		WHERE name = 'execution_order'
	`).Scan(&featuresHasColumn)
	if err != nil {
		t.Fatalf("Failed to check features table: %v", err)
	}
	if !featuresHasColumn {
		t.Error("execution_order column not added to features table")
	}

	// Verify tasks table has execution_order column
	var tasksHasColumn bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('tasks')
		WHERE name = 'execution_order'
	`).Scan(&tasksHasColumn)
	if err != nil {
		t.Fatalf("Failed to check tasks table: %v", err)
	}
	if !tasksHasColumn {
		t.Error("execution_order column not added to tasks table")
	}

	// Test NULL handling
	testNullHandling(t, db)
}

func testNullHandling(t *testing.T, db *sql.DB) {
	// Create test epic
	_, err := db.Exec(`
		INSERT INTO epics (key, title, status, priority)
		VALUES ('E99', 'Test Epic', 'active', 'medium')
	`)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Create features with and without execution_order
	_, err = db.Exec(`
		INSERT INTO features (epic_id, key, title, status, execution_order)
		VALUES
			(1, 'F99-01', 'Feature 1', 'active', 1),
			(1, 'F99-02', 'Feature 2', 'active', NULL),
			(1, 'F99-03', 'Feature 3', 'active', 2)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test features: %v", err)
	}

	// Query features ordered by execution_order NULLS LAST
	rows, err := db.Query(`
		SELECT key, execution_order
		FROM features
		WHERE epic_id = 1
		ORDER BY execution_order NULLS LAST, created_at
	`)
	if err != nil {
		t.Fatalf("Failed to query features: %v", err)
	}
	defer rows.Close()

	var results []struct {
		key            string
		executionOrder sql.NullInt64
	}

	for rows.Next() {
		var r struct {
			key            string
			executionOrder sql.NullInt64
		}
		if err := rows.Scan(&r.key, &r.executionOrder); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		results = append(results, r)
	}

	// Verify order: F99-01 (1), F99-03 (2), F99-02 (NULL)
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	if results[0].key != "F99-01" || !results[0].executionOrder.Valid || results[0].executionOrder.Int64 != 1 {
		t.Errorf("Expected first result to be F99-01 with execution_order=1, got %s with %v", results[0].key, results[0].executionOrder)
	}

	if results[1].key != "F99-03" || !results[1].executionOrder.Valid || results[1].executionOrder.Int64 != 2 {
		t.Errorf("Expected second result to be F99-03 with execution_order=2, got %s with %v", results[1].key, results[1].executionOrder)
	}

	if results[2].key != "F99-02" || results[2].executionOrder.Valid {
		t.Errorf("Expected third result to be F99-02 with NULL execution_order, got %s with %v", results[2].key, results[2].executionOrder)
	}

	// Cleanup test data
	_, _ = db.Exec(`DELETE FROM features WHERE epic_id = 1`)
	_, _ = db.Exec(`DELETE FROM epics WHERE id = 1`)
}

func TestMigrateAddSlugColumns(t *testing.T) {
	// Create a temporary database file
	tmpDB := "test_migrate_slug.db"
	defer os.Remove(tmpDB)
	defer os.Remove(tmpDB + "-shm")
	defer os.Remove(tmpDB + "-wal")

	// Initialize database with existing schema
	db, err := InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify slug columns exist in all three tables
	tables := []string{"epics", "features", "tasks"}
	for _, table := range tables {
		var hasSlugColumn bool
		err = db.QueryRow(`
			SELECT COUNT(*) > 0
			FROM pragma_table_info(?)
			WHERE name = 'slug'
		`, table).Scan(&hasSlugColumn)
		if err != nil {
			t.Fatalf("Failed to check %s table for slug column: %v", table, err)
		}
		if !hasSlugColumn {
			t.Errorf("slug column not found in %s table", table)
		}
	}

	// Verify indexes exist for slug columns
	for _, table := range tables {
		indexName := "idx_" + table + "_slug"
		var indexExists bool
		err = db.QueryRow(`
			SELECT COUNT(*) > 0
			FROM sqlite_master
			WHERE type = 'index' AND name = ?
		`, indexName).Scan(&indexExists)
		if err != nil {
			t.Fatalf("Failed to check for %s index: %v", indexName, err)
		}
		if !indexExists {
			t.Errorf("Index %s not found", indexName)
		}
	}

	// Test that slug column accepts NULL values (for backwards compatibility)
	_, err = db.Exec(`
		INSERT INTO epics (key, title, status, priority, slug)
		VALUES ('E98', 'Test Epic', 'active', 'medium', NULL)
	`)
	if err != nil {
		t.Errorf("Failed to insert epic with NULL slug: %v", err)
	}

	// Test that slug column accepts text values
	_, err = db.Exec(`
		INSERT INTO epics (key, title, status, priority, slug)
		VALUES ('E97', 'Test Epic 2', 'active', 'medium', 'test-epic-2')
	`)
	if err != nil {
		t.Errorf("Failed to insert epic with slug value: %v", err)
	}

	// Verify we can query by slug
	var epicKey string
	err = db.QueryRow(`SELECT key FROM epics WHERE slug = 'test-epic-2'`).Scan(&epicKey)
	if err != nil {
		t.Errorf("Failed to query epic by slug: %v", err)
	}
	if epicKey != "E97" {
		t.Errorf("Expected epic key 'E97', got '%s'", epicKey)
	}

	// Cleanup test data
	_, _ = db.Exec(`DELETE FROM epics WHERE key IN ('E98', 'E97')`)
}

// TestMigrateTasksStatusConstraint_FixesTaskHistoryForeignKey tests that the migration
// correctly recreates the task_history table with the proper foreign key reference
func TestMigrateTasksStatusConstraint_FixesTaskHistoryForeignKey(t *testing.T) {
	// Create a test database in memory
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create old schema with CHECK constraint on status (simulating pre-migration state)
	_, err = db.Exec(`
		CREATE TABLE features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			status TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create features table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feature_id INTEGER NOT NULL,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'completed')),
			agent_type TEXT,
			priority INTEGER NOT NULL DEFAULT 5,
			depends_on TEXT,
			assigned_agent TEXT,
			file_path TEXT,
			blocked_reason TEXT,
			execution_order INTEGER NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			blocked_at TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_by TEXT,
			completion_notes TEXT,
			files_changed TEXT,
			tests_passed BOOLEAN DEFAULT 0,
			verification_status TEXT CHECK(verification_status IN ('pending', 'verified', 'needs_rework')) DEFAULT 'pending',
			time_spent_minutes INTEGER,
			context_data TEXT,
			slug TEXT,
			FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tasks table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE task_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL,
			old_status TEXT,
			new_status TEXT NOT NULL,
			agent TEXT,
			notes TEXT,
			forced BOOLEAN DEFAULT FALSE,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create task_history table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`INSERT INTO features (id, key, title, status) VALUES (1, 'F01', 'Test Feature', 'active')`)
	if err != nil {
		t.Fatalf("Failed to insert test feature: %v", err)
	}

	_, err = db.Exec(`INSERT INTO tasks (id, feature_id, key, title, status) VALUES (1, 1, 'T-001', 'Test Task', 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert test task: %v", err)
	}

	_, err = db.Exec(`INSERT INTO task_history (task_id, old_status, new_status) VALUES (1, NULL, 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert test history: %v", err)
	}

	// Run the migration
	err = migrateTasksStatusConstraint(db)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify the task_history table has the correct foreign key
	var schema string
	err = db.QueryRow(`
		SELECT sql FROM sqlite_master WHERE type='table' AND name='task_history'
	`).Scan(&schema)
	if err != nil {
		t.Fatalf("Failed to get task_history schema: %v", err)
	}

	// Check that foreign key references 'tasks', not 'tasks_old'
	if containsSubstring(schema, "tasks_old") {
		t.Errorf("task_history still references tasks_old. Schema:\n%s", schema)
	}

	if !containsSubstring(schema, "REFERENCES tasks(id)") && !containsSubstring(schema, `REFERENCES "tasks"(id)`) {
		t.Errorf("task_history does not reference tasks(id). Schema:\n%s", schema)
	}

	// Verify data integrity - task_history records should still exist
	var historyCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM task_history WHERE task_id = 1`).Scan(&historyCount)
	if err != nil {
		t.Fatalf("Failed to query task_history: %v", err)
	}

	if historyCount != 1 {
		t.Errorf("Expected 1 history record, got %d", historyCount)
	}

	// Verify we can insert new history records with foreign key constraint working
	_, err = db.Exec(`INSERT INTO task_history (task_id, old_status, new_status) VALUES (1, 'todo', 'in_progress')`)
	if err != nil {
		t.Errorf("Failed to insert new history record after migration: %v", err)
	}

	// Verify foreign key constraint is enforced (should fail with non-existent task_id)
	_, err = db.Exec(`INSERT INTO task_history (task_id, old_status, new_status) VALUES (999, 'todo', 'in_progress')`)
	if err == nil {
		t.Error("Expected foreign key constraint error when inserting history for non-existent task, but got none")
	}
}

// Helper function to check if string contains substring
func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMigrateTaskHistoryForeignKey_AlreadyMigratedDatabase tests that the dedicated
// migration function fixes task_history in databases where tasks was already migrated
func TestMigrateTaskHistoryForeignKey_AlreadyMigratedDatabase(t *testing.T) {
	// Create a test database in memory
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Temporarily disable foreign keys for test setup
	// (We need to create task_history with a bad FK reference)
	_, err = db.Exec("PRAGMA foreign_keys = OFF;")
	if err != nil {
		t.Fatalf("Failed to disable foreign keys: %v", err)
	}

	// Create schema simulating a database that has ALREADY been migrated
	// (tasks table has no status constraint, but task_history has wrong FK)
	_, err = db.Exec(`
		CREATE TABLE features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			status TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create features table: %v", err)
	}

	// Create tasks table WITHOUT status constraint (already migrated)
	_, err = db.Exec(`
		CREATE TABLE tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feature_id INTEGER NOT NULL,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 5,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tasks table: %v", err)
	}

	// Create task_history with WRONG foreign key (references tasks_old)
	// This simulates the bug state
	_, err = db.Exec(`
		CREATE TABLE task_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL,
			old_status TEXT,
			new_status TEXT NOT NULL,
			agent TEXT,
			notes TEXT,
			forced BOOLEAN DEFAULT FALSE,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES "tasks_old"(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create task_history table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`INSERT INTO features (id, key, title, status) VALUES (1, 'F01', 'Test Feature', 'active')`)
	if err != nil {
		t.Fatalf("Failed to insert test feature: %v", err)
	}

	_, err = db.Exec(`INSERT INTO tasks (id, feature_id, key, title, status) VALUES (1, 1, 'T-001', 'Test Task', 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert test task: %v", err)
	}

	_, err = db.Exec(`INSERT INTO task_history (task_id, old_status, new_status) VALUES (1, NULL, 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert test history: %v", err)
	}

	// Verify the bug exists - task_history references tasks_old
	var schema string
	err = db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='task_history'`).Scan(&schema)
	if err != nil {
		t.Fatalf("Failed to get task_history schema: %v", err)
	}

	if !containsSubstring(schema, "tasks_old") {
		t.Fatal("Test setup failed: task_history should reference tasks_old before migration")
	}

	// Run the dedicated migration function
	err = migrateTaskHistoryForeignKey(db)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify the fix - task_history should now reference tasks
	err = db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='task_history'`).Scan(&schema)
	if err != nil {
		t.Fatalf("Failed to get task_history schema after migration: %v", err)
	}

	if containsSubstring(schema, "tasks_old") {
		t.Errorf("task_history still references tasks_old after migration. Schema:\n%s", schema)
	}

	if !containsSubstring(schema, "REFERENCES tasks(id)") && !containsSubstring(schema, `REFERENCES "tasks"(id)`) {
		t.Errorf("task_history does not reference tasks(id) after migration. Schema:\n%s", schema)
	}

	// Verify data integrity
	var historyCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM task_history WHERE task_id = 1`).Scan(&historyCount)
	if err != nil {
		t.Fatalf("Failed to query task_history: %v", err)
	}

	if historyCount != 1 {
		t.Errorf("Expected 1 history record, got %d", historyCount)
	}

	// Verify we can insert new history records
	_, err = db.Exec(`INSERT INTO task_history (task_id, old_status, new_status) VALUES (1, 'todo', 'in_progress')`)
	if err != nil {
		t.Errorf("Failed to insert new history record after migration: %v", err)
	}

	// Verify running migration again is idempotent (no-op)
	err = migrateTaskHistoryForeignKey(db)
	if err != nil {
		t.Errorf("Migration should be idempotent but failed on second run: %v", err)
	}
}
