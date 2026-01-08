package db

import (
	"context"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestAbstractionLayer_EndToEnd tests the complete abstraction layer
// from config → registry → driver → database operations
func TestAbstractionLayer_EndToEnd(t *testing.T) {
	ctx := context.Background()

	// Reset registry and register SQLite driver
	ResetRegistry()
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	// Create temporary database
	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	// Step 1: Initialize database using registry
	cfg := config.DatabaseConfig{
		Backend: "sqlite",
		URL:     tmpDB,
	}

	db, err := InitDatabase(ctx, cfg)
	if err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	defer db.Close()

	// Step 2: Create schema (simulate InitDB behavior)
	schema := `
		CREATE TABLE IF NOT EXISTS test_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'todo',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := db.Exec(ctx, schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Step 3: Insert data
	result, err := db.Exec(ctx,
		"INSERT INTO test_tasks (title, status) VALUES (?, ?)",
		"Test Task 1", "todo")
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get LastInsertId: %v", err)
	}

	// Step 4: Query single row
	row := db.QueryRow(ctx, "SELECT title, status FROM test_tasks WHERE id = ?", taskID)
	var title, status string
	if err := row.Scan(&title, &status); err != nil {
		t.Fatalf("Failed to scan row: %v", err)
	}

	if title != "Test Task 1" {
		t.Errorf("Expected title 'Test Task 1', got %q", title)
	}
	if status != "todo" {
		t.Errorf("Expected status 'todo', got %q", status)
	}

	// Step 5: Transaction test (commit)
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO test_tasks (title, status) VALUES (?, ?)", "Test Task 2", "in_progress")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Step 6: Verify transaction was committed
	rows, err := db.Query(ctx, "SELECT COUNT(*) FROM test_tasks")
	if err != nil {
		t.Fatalf("Failed to query count: %v", err)
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			t.Fatalf("Failed to scan count: %v", err)
		}
	}
	if count != 2 {
		t.Errorf("Expected 2 tasks after commit, got %d", count)
	}

	// Step 7: Transaction test (rollback)
	tx2, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin second transaction: %v", err)
	}

	_, err = tx2.Exec(ctx, "INSERT INTO test_tasks (title, status) VALUES (?, ?)", "Test Task 3", "completed")
	if err != nil {
		_ = tx2.Rollback()
		t.Fatalf("Failed to insert in second transaction: %v", err)
	}

	if err := tx2.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Step 8: Verify rollback
	row2 := db.QueryRow(ctx, "SELECT COUNT(*) FROM test_tasks")
	if err := row2.Scan(&count); err != nil {
		t.Fatalf("Failed to scan count after rollback: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 tasks after rollback, got %d", count)
	}
}

// TestAbstractionLayer_AutoDetection tests automatic backend detection
func TestAbstractionLayer_AutoDetection(t *testing.T) {
	ctx := context.Background()

	ResetRegistry()
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	// Config without explicit backend (should auto-detect)
	cfg := config.DatabaseConfig{
		URL: tmpDB,
	}

	db, err := InitDatabase(ctx, cfg)
	if err != nil {
		t.Fatalf("InitDatabase with auto-detection failed: %v", err)
	}
	defer db.Close()

	// Verify it's SQLite
	if db.DriverName() != "sqlite3" {
		t.Errorf("Expected auto-detected driver to be 'sqlite3', got %q", db.DriverName())
	}

	// Verify it works
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Ping failed on auto-detected database: %v", err)
	}
}

// TestAbstractionLayer_MultipleDrivers tests registering multiple backends
func TestAbstractionLayer_MultipleDrivers(t *testing.T) {
	ResetRegistry()

	// Register SQLite
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	// Register mock Turso driver
	RegisterDriver("turso", func() Database {
		return &mockDatabase{}
	})

	// Verify both are registered
	drivers := GetRegisteredDrivers()
	if len(drivers) != 2 {
		t.Fatalf("Expected 2 registered drivers, got %d", len(drivers))
	}

	// Test creating SQLite instance
	sqliteCfg := config.DatabaseConfig{
		Backend: "sqlite",
		URL:     "./test.db",
	}
	sqliteDB, err := NewDatabase(sqliteCfg)
	if err != nil {
		t.Errorf("Failed to create SQLite database: %v", err)
	}
	if sqliteDB.DriverName() != "sqlite3" {
		t.Errorf("Expected SQLite driver, got %q", sqliteDB.DriverName())
	}

	// Test creating Turso instance
	tursoCfg := config.DatabaseConfig{
		Backend: "turso",
		URL:     "libsql://test.turso.io",
	}
	tursoDB, err := NewDatabase(tursoCfg)
	if err != nil {
		t.Errorf("Failed to create Turso database: %v", err)
	}
	if tursoDB.DriverName() != "mock" {
		t.Errorf("Expected mock driver, got %q", tursoDB.DriverName())
	}
}

// TestAbstractionLayer_ConcurrentAccess tests thread-safety of registry
func TestAbstractionLayer_ConcurrentAccess(t *testing.T) {
	ResetRegistry()

	// Register driver
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	// Create multiple goroutines accessing registry
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			cfg := config.DatabaseConfig{
				Backend: "sqlite",
				URL:     ":memory:",
			}
			_, err := NewDatabase(cfg)
			if err != nil {
				t.Errorf("Concurrent NewDatabase failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestAbstractionLayer_InvalidConfig tests error handling for invalid configs
func TestAbstractionLayer_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	ResetRegistry()
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	tests := []struct {
		name   string
		config config.DatabaseConfig
	}{
		{
			name: "Empty backend",
			config: config.DatabaseConfig{
				Backend: "",
				URL:     "",
			},
		},
		{
			name: "Invalid backend and URL mismatch",
			config: config.DatabaseConfig{
				Backend: "turso",
				URL:     "./local.db", // File path for turso backend
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := InitDatabase(ctx, tt.config)
			if err == nil {
				t.Error("Expected error for invalid config, got nil")
			}
		})
	}
}

// TestAbstractionLayer_SchemaCreation tests that schema creation works through abstraction
func TestAbstractionLayer_SchemaCreation(t *testing.T) {
	ctx := context.Background()

	ResetRegistry()
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	tmpDB := t.TempDir() + "/test.db"
	defer os.Remove(tmpDB)

	cfg := config.DatabaseConfig{
		Backend: "sqlite",
		URL:     tmpDB,
	}

	db, err := InitDatabase(ctx, cfg)
	if err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	defer db.Close()

	// Create complex schema with foreign keys
	schema := `
		CREATE TABLE epics (
			id INTEGER PRIMARY KEY,
			title TEXT NOT NULL
		);

		CREATE TABLE features (
			id INTEGER PRIMARY KEY,
			epic_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
		);

		CREATE TABLE tasks (
			id INTEGER PRIMARY KEY,
			feature_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'todo',
			FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
		);
	`

	if _, err := db.Exec(ctx, schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Insert test data with foreign keys
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO epics (id, title) VALUES (1, 'Epic 1')")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("Failed to insert epic: %v", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO features (id, epic_id, title) VALUES (1, 1, 'Feature 1')")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("Failed to insert feature: %v", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO tasks (id, feature_id, title) VALUES (1, 1, 'Task 1')")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("Failed to insert task: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify foreign key cascade on delete
	_, err = db.Exec(ctx, "DELETE FROM epics WHERE id = 1")
	if err != nil {
		t.Fatalf("Failed to delete epic: %v", err)
	}

	// Check that cascade deleted features and tasks
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM tasks")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count tasks: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 tasks after cascade delete, got %d", count)
	}
}
