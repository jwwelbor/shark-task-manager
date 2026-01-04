package db

import (
	"os"
	"testing"
)

// TestIdeasTableCreation tests that the ideas table is created successfully
func TestIdeasTableCreation(t *testing.T) {
	// Create temporary database file
	tmpFile := "test_ideas_migration.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	// Initialize database with schema
	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify ideas table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='ideas'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Ideas table not found: %v", err)
	}
	if tableName != "ideas" {
		t.Errorf("Expected table name 'ideas', got '%s'", tableName)
	}
}

// TestIdeasTableSchema tests that all expected columns exist with correct constraints
func TestIdeasTableSchema(t *testing.T) {
	// Create temporary database file
	tmpFile := "test_ideas_schema.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	// Initialize database
	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Expected columns
	expectedColumns := []string{
		"id", "key", "title", "description", "created_date",
		"priority", "display_order", "notes", "related_docs", "dependencies",
		"status", "created_at", "updated_at",
		"converted_to_type", "converted_to_key", "converted_at",
	}

	for _, colName := range expectedColumns {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('ideas') WHERE name = ?", colName).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query column info for '%s': %v", colName, err)
		}
		if count == 0 {
			t.Errorf("Expected column '%s' not found in ideas table", colName)
		}
	}
}

// TestIdeasTableIndexes tests that all expected indexes are created
func TestIdeasTableIndexes(t *testing.T) {
	// Create temporary database file
	tmpFile := "test_ideas_indexes.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	// Initialize database
	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Expected indexes
	expectedIndexes := []string{
		"idx_ideas_key",
		"idx_ideas_status",
		"idx_ideas_created_date",
		"idx_ideas_priority",
	}

	for _, idxName := range expectedIndexes {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name = ?", idxName).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query index info for '%s': %v", idxName, err)
		}
		if count == 0 {
			t.Errorf("Expected index '%s' not found", idxName)
		}
	}
}

// TestIdeasTableInsertAndQuery tests basic insert and query operations
func TestIdeasTableInsertAndQuery(t *testing.T) {
	// Create temporary database file
	tmpFile := "test_ideas_insert.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	// Initialize database
	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Insert test idea
	_, err = db.Exec(`
		INSERT INTO ideas (key, title, description, created_date, priority, status)
		VALUES (?, ?, ?, datetime('now'), ?, ?)
	`, "I-2026-01-01-01", "Test Idea", "Test Description", 5, "new")
	if err != nil {
		t.Fatalf("Failed to insert test idea: %v", err)
	}

	// Query the idea back
	var key, title, description, status string
	var priority int
	err = db.QueryRow("SELECT key, title, description, priority, status FROM ideas WHERE key = ?", "I-2026-01-01-01").
		Scan(&key, &title, &description, &priority, &status)
	if err != nil {
		t.Fatalf("Failed to query test idea: %v", err)
	}

	// Verify values
	if key != "I-2026-01-01-01" {
		t.Errorf("Expected key 'I-2026-01-01-01', got '%s'", key)
	}
	if title != "Test Idea" {
		t.Errorf("Expected title 'Test Idea', got '%s'", title)
	}
	if description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got '%s'", description)
	}
	if priority != 5 {
		t.Errorf("Expected priority 5, got %d", priority)
	}
	if status != "new" {
		t.Errorf("Expected status 'new', got '%s'", status)
	}
}

// TestIdeasTableStatusConstraint tests that only valid statuses are accepted
func TestIdeasTableStatusConstraint(t *testing.T) {
	// Create temporary database file
	tmpFile := "test_ideas_status_constraint.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	// Initialize database
	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Try to insert idea with invalid status (should fail)
	_, err = db.Exec(`
		INSERT INTO ideas (key, title, created_date, status)
		VALUES (?, ?, datetime('now'), ?)
	`, "I-2026-01-01-02", "Test Idea", "invalid_status")

	if err == nil {
		t.Error("Expected error when inserting invalid status, but got none")
	}
}
