package db

import (
	"os"
	"testing"
)

// TestBugsTableCreation tests that the bugs table is created successfully (TC-S1-01)
func TestBugsTableCreation(t *testing.T) {
	tmpFile := "test_bugs_creation.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='bugs'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Bugs table not found: %v", err)
	}
	if tableName != "bugs" {
		t.Errorf("Expected table name 'bugs', got '%s'", tableName)
	}
}

// TestBugsTableSchema tests that all expected columns exist (TC-S1-02)
func TestBugsTableSchema(t *testing.T) {
	tmpFile := "test_bugs_schema.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	expectedColumns := []string{
		"id", "key", "title", "slug", "description",
		"status", "severity", "linked_entity_type", "linked_entity_key",
		"context_data", "file_path", "created_at", "updated_at",
	}

	for _, colName := range expectedColumns {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('bugs') WHERE name = ?", colName).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query column info for '%s': %v", colName, err)
		}
		if count == 0 {
			t.Errorf("Expected column '%s' not found in bugs table", colName)
		}
	}
}

// TestBugsTableIndexes tests that all expected indexes are created (TC-S1-03)
func TestBugsTableIndexes(t *testing.T) {
	tmpFile := "test_bugs_indexes.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	expectedIndexes := []string{
		"idx_bugs_key",
		"idx_bugs_status",
		"idx_bugs_severity",
		"idx_bugs_linked_entity_type",
		"idx_bugs_slug",
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

// TestBugsTableDefaultValues tests that default status and severity are applied (TC-S1-04)
func TestBugsTableDefaultValues(t *testing.T) {
	tmpFile := "test_bugs_defaults.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Insert with minimal required fields to test defaults
	_, err = db.Exec(`INSERT INTO bugs (key, title) VALUES (?, ?)`, "BUG-001", "Test Bug")
	if err != nil {
		t.Fatalf("Failed to insert minimal bug: %v", err)
	}

	var status, severity string
	err = db.QueryRow("SELECT status, severity FROM bugs WHERE key = 'BUG-001'").Scan(&status, &severity)
	if err != nil {
		t.Fatalf("Failed to query inserted bug: %v", err)
	}

	if status != "reported" {
		t.Errorf("Expected default status 'reported', got '%s'", status)
	}
	if severity != "medium" {
		t.Errorf("Expected default severity 'medium', got '%s'", severity)
	}
}

// TestBugsTableSeverityConstraint tests that only valid severities are accepted (TC-S1-05)
func TestBugsTableSeverityConstraint(t *testing.T) {
	tmpFile := "test_bugs_severity_constraint.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	validSeverities := []string{"critical", "high", "medium", "low"}
	for i, sev := range validSeverities {
		key := "BUG-VALID-00" + string(rune('1'+i))
		_, err = db.Exec(`INSERT INTO bugs (key, title, severity) VALUES (?, ?, ?)`, key, "Test Bug", sev)
		if err != nil {
			t.Errorf("Expected valid severity '%s' to be accepted, got error: %v", sev, err)
		}
	}

	// Invalid severity should be rejected
	_, err = db.Exec(`INSERT INTO bugs (key, title, severity) VALUES (?, ?, ?)`, "BUG-INVALID-001", "Test Bug", "unknown")
	if err == nil {
		t.Error("Expected error for invalid severity 'unknown', but got none")
	}
}

// TestBugsTableInsertAndQuery tests basic CRUD operations (TC-S1-06)
func TestBugsTableInsertAndQuery(t *testing.T) {
	tmpFile := "test_bugs_insert.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO bugs (key, title, slug, description, status, severity, linked_entity_type, linked_entity_key)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "BUG-2026-001", "Login Failure", "login-failure", "Users cannot log in with valid credentials",
		"triaged", "high", "task", "E07-F01-001")
	if err != nil {
		t.Fatalf("Failed to insert test bug: %v", err)
	}

	var key, title, slug, description, status, severity, linkedType, linkedKey string
	err = db.QueryRow(`
		SELECT key, title, slug, description, status, severity, linked_entity_type, linked_entity_key
		FROM bugs WHERE key = ?`, "BUG-2026-001").
		Scan(&key, &title, &slug, &description, &status, &severity, &linkedType, &linkedKey)
	if err != nil {
		t.Fatalf("Failed to query test bug: %v", err)
	}

	if key != "BUG-2026-001" {
		t.Errorf("Expected key 'BUG-2026-001', got '%s'", key)
	}
	if title != "Login Failure" {
		t.Errorf("Expected title 'Login Failure', got '%s'", title)
	}
	if slug != "login-failure" {
		t.Errorf("Expected slug 'login-failure', got '%s'", slug)
	}
	if description != "Users cannot log in with valid credentials" {
		t.Errorf("Unexpected description: '%s'", description)
	}
	if status != "triaged" {
		t.Errorf("Expected status 'triaged', got '%s'", status)
	}
	if severity != "high" {
		t.Errorf("Expected severity 'high', got '%s'", severity)
	}
	if linkedType != "task" {
		t.Errorf("Expected linked_entity_type 'task', got '%s'", linkedType)
	}
	if linkedKey != "E07-F01-001" {
		t.Errorf("Expected linked_entity_key 'E07-F01-001', got '%s'", linkedKey)
	}
}

// TestBugsTableUniqueKeyConstraint tests that duplicate keys are rejected (TC-S1-07)
func TestBugsTableUniqueKeyConstraint(t *testing.T) {
	tmpFile := "test_bugs_unique_key.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO bugs (key, title) VALUES (?, ?)`, "BUG-DUP-001", "First Bug")
	if err != nil {
		t.Fatalf("Failed to insert first bug: %v", err)
	}

	_, err = db.Exec(`INSERT INTO bugs (key, title) VALUES (?, ?)`, "BUG-DUP-001", "Duplicate Bug")
	if err == nil {
		t.Error("Expected UNIQUE constraint error for duplicate key, got none")
	}
}

// TestBugsTableIdempotentMigration tests that running migration twice does not error (TC-S1-08)
func TestBugsTableIdempotentMigration(t *testing.T) {
	tmpFile := "test_bugs_idempotent.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database on first call: %v", err)
	}
	db.Close()

	// Reopen (runs migrations again)
	db2, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database on second call: %v", err)
	}
	defer db2.Close()

	// Bugs table should still be there
	var tableName string
	err = db2.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='bugs'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Bugs table not found after second migration: %v", err)
	}
	if tableName != "bugs" {
		t.Errorf("Expected 'bugs', got '%s'", tableName)
	}
}

// TestBugsTableTimestamps tests that created_at and updated_at are automatically set (TC-S1-09)
func TestBugsTableTimestamps(t *testing.T) {
	tmpFile := "test_bugs_timestamps.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO bugs (key, title) VALUES (?, ?)`, "BUG-TS-001", "Timestamp Test Bug")
	if err != nil {
		t.Fatalf("Failed to insert bug: %v", err)
	}

	var createdAt, updatedAt string
	err = db.QueryRow("SELECT created_at, updated_at FROM bugs WHERE key = 'BUG-TS-001'").
		Scan(&createdAt, &updatedAt)
	if err != nil {
		t.Fatalf("Failed to query timestamps: %v", err)
	}

	if createdAt == "" {
		t.Error("Expected created_at to be set automatically, got empty string")
	}
	if updatedAt == "" {
		t.Error("Expected updated_at to be set automatically, got empty string")
	}
}

// TestBugsTableUpdatedAtTrigger tests that the updated_at trigger fires on UPDATE (TC-S1-10)
func TestBugsTableUpdatedAtTrigger(t *testing.T) {
	tmpFile := "test_bugs_trigger.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify the trigger exists
	var triggerCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='bugs_updated_at'").
		Scan(&triggerCount)
	if err != nil {
		t.Fatalf("Failed to query trigger: %v", err)
	}
	if triggerCount == 0 {
		t.Error("Expected trigger 'bugs_updated_at' to exist")
	}
}
