package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

// TestSQLiteDriver_Interface verifies SQLiteDriver implements Database interface
func TestSQLiteDriver_Interface(t *testing.T) {
	var _ Database = (*SQLiteDriver)(nil) // Compile-time check
}

// TestSQLiteDriver_Connect tests connecting to a SQLite database
func TestSQLiteDriver_Connect(t *testing.T) {
	ctx := context.Background()
	driver := NewSQLiteDriver()
	defer driver.Close()

	// Create temporary database
	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	// Connect
	err := driver.Connect(ctx, tmpDB)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Verify connection with Ping
	if err := driver.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Verify driver name
	if driver.DriverName() != "sqlite3" {
		t.Errorf("Expected driver name 'sqlite3', got %q", driver.DriverName())
	}
}

// TestSQLiteDriver_QueryOperations tests basic query operations
func TestSQLiteDriver_QueryOperations(t *testing.T) {
	ctx := context.Background()
	driver := NewSQLiteDriver()
	defer driver.Close()

	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	if err := driver.Connect(ctx, tmpDB); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create test table
	_, err := driver.Exec(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert data
	result, err := driver.Exec(ctx, "INSERT INTO test (name) VALUES (?)", "test1")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Verify LastInsertId
	id, err := result.LastInsertId()
	if err != nil {
		t.Errorf("Failed to get LastInsertId: %v", err)
	}
	if id != 1 {
		t.Errorf("Expected LastInsertId 1, got %d", id)
	}

	// Query single row
	row := driver.QueryRow(ctx, "SELECT name FROM test WHERE id = ?", id)
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("Failed to scan row: %v", err)
	}
	if name != "test1" {
		t.Errorf("Expected name 'test1', got %q", name)
	}

	// Insert more data
	_, _ = driver.Exec(ctx, "INSERT INTO test (name) VALUES (?)", "test2")
	_, _ = driver.Exec(ctx, "INSERT INTO test (name) VALUES (?)", "test3")

	// Query multiple rows
	rows, err := driver.Query(ctx, "SELECT id, name FROM test ORDER BY id")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Errorf("Failed to scan row: %v", err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Errorf("Rows error: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 rows, got %d", count)
	}
}

// TestSQLiteDriver_Transactions tests transaction support
func TestSQLiteDriver_Transactions(t *testing.T) {
	ctx := context.Background()
	driver := NewSQLiteDriver()
	defer driver.Close()

	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	if err := driver.Connect(ctx, tmpDB); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create test table
	_, err := driver.Exec(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Test commit
	tx, err := driver.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO test (name) VALUES (?)", "committed")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify data was committed
	row := driver.QueryRow(ctx, "SELECT COUNT(*) FROM test")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row after commit, got %d", count)
	}

	// Test rollback
	tx2, err := driver.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin second transaction: %v", err)
	}

	_, err = tx2.Exec(ctx, "INSERT INTO test (name) VALUES (?)", "rolled_back")
	if err != nil {
		_ = tx2.Rollback()
		t.Fatalf("Failed to insert in second transaction: %v", err)
	}

	if err := tx2.Rollback(); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify data was NOT committed
	row2 := driver.QueryRow(ctx, "SELECT COUNT(*) FROM test")
	if err := row2.Scan(&count); err != nil {
		t.Fatalf("Failed to count rows after rollback: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row after rollback, got %d", count)
	}
}

// TestSQLiteDriver_ForeignKeys tests that foreign keys are enforced
func TestSQLiteDriver_ForeignKeys(t *testing.T) {
	ctx := context.Background()
	driver := NewSQLiteDriver()
	defer driver.Close()

	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	if err := driver.Connect(ctx, tmpDB); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create tables with foreign key
	_, err := driver.Exec(ctx, `
		CREATE TABLE parent (id INTEGER PRIMARY KEY);
		CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id));
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Try to insert child with non-existent parent (should fail)
	_, err = driver.Exec(ctx, "INSERT INTO child (id, parent_id) VALUES (1, 999)")
	if err == nil {
		t.Error("Expected foreign key constraint violation, but insert succeeded")
	}

	// Insert parent first
	_, err = driver.Exec(ctx, "INSERT INTO parent (id) VALUES (1)")
	if err != nil {
		t.Fatalf("Failed to insert parent: %v", err)
	}

	// Now insert child should succeed
	_, err = driver.Exec(ctx, "INSERT INTO child (id, parent_id) VALUES (1, 1)")
	if err != nil {
		t.Errorf("Failed to insert child with valid parent: %v", err)
	}
}

// TestSQLiteDriver_ErrorHandling tests error handling for invalid operations
func TestSQLiteDriver_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Test operations on unconnected driver
	driver := NewSQLiteDriver()

	if err := driver.Ping(ctx); err != sql.ErrConnDone {
		t.Errorf("Expected ErrConnDone from Ping on unconnected driver, got %v", err)
	}

	_, err := driver.Query(ctx, "SELECT 1")
	if err != sql.ErrConnDone {
		t.Errorf("Expected ErrConnDone from Query on unconnected driver, got %v", err)
	}

	_, err = driver.Exec(ctx, "SELECT 1")
	if err != sql.ErrConnDone {
		t.Errorf("Expected ErrConnDone from Exec on unconnected driver, got %v", err)
	}

	_, err = driver.Begin(ctx)
	if err != sql.ErrConnDone {
		t.Errorf("Expected ErrConnDone from Begin on unconnected driver, got %v", err)
	}

	// Test invalid SQL
	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	driver2 := NewSQLiteDriver()
	defer driver2.Close()

	if err := driver2.Connect(ctx, tmpDB); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	_, err = driver2.Exec(ctx, "INVALID SQL")
	if err == nil {
		t.Error("Expected error from invalid SQL, got nil")
	}
}

// TestSQLiteDriver_CloseIdempotent tests that Close can be called multiple times
func TestSQLiteDriver_CloseIdempotent(t *testing.T) {
	ctx := context.Background()
	driver := NewSQLiteDriver()

	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	if err := driver.Connect(ctx, tmpDB); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Close multiple times should not error
	if err := driver.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}

	if err := driver.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}
