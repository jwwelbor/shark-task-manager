package db

import (
	"context"
	"os"
	"testing"
)

// TestTursoDriver_Interface verifies TursoDriver implements Database interface
func TestTursoDriver_Interface(t *testing.T) {
	var _ Database = (*TursoDriver)(nil) // Compile-time check
}

// TestTursoDriver_NewDriver tests creating a new Turso driver instance
func TestTursoDriver_NewDriver(t *testing.T) {
	driver := NewTursoDriver()
	if driver == nil {
		t.Fatal("NewTursoDriver returned nil")
	}
	if driver.DriverName() != "libsql" {
		t.Errorf("Expected driver name 'libsql', got %q", driver.DriverName())
	}
}

// TestTursoDriver_Connect tests connection to Turso (requires credentials)
func TestTursoDriver_Connect(t *testing.T) {
	// Skip if no credentials available
	dbURL := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	if dbURL == "" || authToken == "" {
		t.Skip("Skipping Turso connection test: TURSO_DATABASE_URL or TURSO_AUTH_TOKEN not set")
	}

	ctx := context.Background()
	driver := NewTursoDriver()
	defer driver.Close()

	// Build connection string
	dsn := BuildTursoConnectionString(dbURL, authToken)

	// Connect
	err := driver.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Failed to connect to Turso: %v", err)
	}

	// Verify connection
	if err := driver.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

// TestTursoDriver_BasicOperations tests basic CRUD operations (requires credentials)
func TestTursoDriver_BasicOperations(t *testing.T) {
	dbURL := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	if dbURL == "" || authToken == "" {
		t.Skip("Skipping Turso operations test: credentials not set")
	}

	ctx := context.Background()
	driver := NewTursoDriver()
	defer driver.Close()

	dsn := BuildTursoConnectionString(dbURL, authToken)
	if err := driver.Connect(ctx, dsn); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create test table
	_, err := driver.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_turso (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Clean up any existing data
	_, err = driver.Exec(ctx, "DELETE FROM test_turso")
	if err != nil {
		t.Fatalf("Failed to clean table: %v", err)
	}

	// Insert data
	result, err := driver.Exec(ctx, "INSERT INTO test_turso (id, name) VALUES (?, ?)", 1, "Test Item")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Errorf("Failed to get RowsAffected: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	// Query data
	row := driver.QueryRow(ctx, "SELECT name FROM test_turso WHERE id = ?", 1)
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("Failed to scan row: %v", err)
	}

	if name != "Test Item" {
		t.Errorf("Expected name 'Test Item', got %q", name)
	}

	// Cleanup
	_, err = driver.Exec(ctx, "DROP TABLE test_turso")
	if err != nil {
		t.Logf("Warning: failed to drop test table: %v", err)
	}
}

// TestTursoDriver_Transactions tests transaction support (requires credentials)
func TestTursoDriver_Transactions(t *testing.T) {
	dbURL := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	if dbURL == "" || authToken == "" {
		t.Skip("Skipping Turso transaction test: credentials not set")
	}

	ctx := context.Background()
	driver := NewTursoDriver()
	defer driver.Close()

	dsn := BuildTursoConnectionString(dbURL, authToken)
	if err := driver.Connect(ctx, dsn); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create test table
	_, err := driver.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx (
			id INTEGER PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Clean up
	defer func() { _, _ = driver.Exec(ctx, "DROP TABLE test_tx") }()
	_, _ = driver.Exec(ctx, "DELETE FROM test_tx")

	// Test commit
	tx, err := driver.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO test_tx (id, value) VALUES (?, ?)", 1, "committed")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify commit
	row := driver.QueryRow(ctx, "SELECT COUNT(*) FROM test_tx")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row after commit, got %d", count)
	}

	// Test rollback
	tx2, err := driver.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin second transaction: %v", err)
	}

	_, err = tx2.Exec(ctx, "INSERT INTO test_tx (id, value) VALUES (?, ?)", 2, "rolled_back")
	if err != nil {
		_ = tx2.Rollback()
		t.Fatalf("Failed to insert in second transaction: %v", err)
	}

	if err := tx2.Rollback(); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify rollback
	row2 := driver.QueryRow(ctx, "SELECT COUNT(*) FROM test_tx")
	if err := row2.Scan(&count); err != nil {
		t.Fatalf("Failed to count after rollback: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row after rollback, got %d", count)
	}
}

// TestBuildTursoConnectionString tests DSN builder
func TestBuildTursoConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		token    string
		expected string
	}{
		{
			name:     "libsql URL with token",
			url:      "libsql://db.turso.io",
			token:    "test-token",
			expected: "libsql://db.turso.io?authToken=test-token",
		},
		{
			name:     "https URL with token",
			url:      "https://db.turso.io",
			token:    "test-token",
			expected: "https://db.turso.io?authToken=test-token",
		},
		{
			name:     "URL with query params",
			url:      "libsql://db.turso.io?foo=bar",
			token:    "test-token",
			expected: "libsql://db.turso.io?foo=bar&authToken=test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildTursoConnectionString(tt.url, tt.token)
			if result != tt.expected {
				t.Errorf("BuildTursoConnectionString(%q, %q) = %q, want %q",
					tt.url, tt.token, result, tt.expected)
			}
		})
	}
}
