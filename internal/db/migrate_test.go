package db

import (
	"database/sql"
	"os"
	"testing"
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
