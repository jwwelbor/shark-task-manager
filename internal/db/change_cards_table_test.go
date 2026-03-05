package db

import (
	"os"
	"testing"
)

// TestChangeCardsTableCreation tests that the change_cards table is created successfully (TC-S2-01)
func TestChangeCardsTableCreation(t *testing.T) {
	tmpFile := "test_change_cards_creation.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='change_cards'").Scan(&tableName)
	if err != nil {
		t.Fatalf("change_cards table not found: %v", err)
	}
	if tableName != "change_cards" {
		t.Errorf("Expected table name 'change_cards', got '%s'", tableName)
	}
}

// TestChangeCardsTableSchema tests that all expected columns exist (TC-S2-02)
func TestChangeCardsTableSchema(t *testing.T) {
	tmpFile := "test_change_cards_schema.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	expectedColumns := []string{
		"id", "key", "title", "description",
		"status", "priority", "requested_by", "assigned_to",
		"epic_id", "feature_id", "related_task_id",
		"justification", "impact_analysis", "rollback_plan",
		"slug", "file_path", "created_at", "updated_at",
	}

	for _, colName := range expectedColumns {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('change_cards') WHERE name = ?", colName).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query column info for '%s': %v", colName, err)
		}
		if count == 0 {
			t.Errorf("Expected column '%s' not found in change_cards table", colName)
		}
	}
}

// TestChangeCardsTableIndexes tests that all expected indexes are created (TC-S2-03)
func TestChangeCardsTableIndexes(t *testing.T) {
	tmpFile := "test_change_cards_indexes.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	expectedIndexes := []string{
		"idx_change_cards_key",
		"idx_change_cards_status",
		"idx_change_cards_epic_id",
		"idx_change_cards_feature_id",
		"idx_change_cards_slug",
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

// TestChangeCardsTableDefaultValues tests that default status and priority are applied (TC-S2-04)
func TestChangeCardsTableDefaultValues(t *testing.T) {
	tmpFile := "test_change_cards_defaults.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO change_cards (key, title) VALUES (?, ?)`, "CC-001", "Test Change Card")
	if err != nil {
		t.Fatalf("Failed to insert minimal change_card: %v", err)
	}

	var status string
	var priority int
	err = db.QueryRow("SELECT status, priority FROM change_cards WHERE key = 'CC-001'").Scan(&status, &priority)
	if err != nil {
		t.Fatalf("Failed to query inserted change_card: %v", err)
	}

	if status != "proposed" {
		t.Errorf("Expected default status 'proposed', got '%s'", status)
	}
	if priority != 5 {
		t.Errorf("Expected default priority 5, got %d", priority)
	}
}

// TestChangeCardsTableInsertAndQuery tests basic CRUD operations (TC-S2-05)
func TestChangeCardsTableInsertAndQuery(t *testing.T) {
	tmpFile := "test_change_cards_insert.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO change_cards (key, title, description, status, priority, requested_by, justification, impact_analysis, rollback_plan, slug)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "CC-2026-001", "Add dark mode", "Support OS-level dark mode preference",
		"approved", 7, "product_owner", "User request from Q4 survey",
		"Affects all UI components", "Revert to light-only CSS", "add-dark-mode")
	if err != nil {
		t.Fatalf("Failed to insert test change card: %v", err)
	}

	var key, title, description, status, requestedBy, justification, impactAnalysis, rollbackPlan, slug string
	var priority int
	err = db.QueryRow(`
		SELECT key, title, description, status, priority, requested_by, justification, impact_analysis, rollback_plan, slug
		FROM change_cards WHERE key = ?`, "CC-2026-001").
		Scan(&key, &title, &description, &status, &priority, &requestedBy, &justification, &impactAnalysis, &rollbackPlan, &slug)
	if err != nil {
		t.Fatalf("Failed to query test change card: %v", err)
	}

	if key != "CC-2026-001" {
		t.Errorf("Expected key 'CC-2026-001', got '%s'", key)
	}
	if title != "Add dark mode" {
		t.Errorf("Expected title 'Add dark mode', got '%s'", title)
	}
	if status != "approved" {
		t.Errorf("Expected status 'approved', got '%s'", status)
	}
	if priority != 7 {
		t.Errorf("Expected priority 7, got %d", priority)
	}
	if requestedBy != "product_owner" {
		t.Errorf("Expected requested_by 'product_owner', got '%s'", requestedBy)
	}
	if slug != "add-dark-mode" {
		t.Errorf("Expected slug 'add-dark-mode', got '%s'", slug)
	}
}

// TestChangeCardsTableUniqueKeyConstraint tests that duplicate keys are rejected (TC-S2-06)
func TestChangeCardsTableUniqueKeyConstraint(t *testing.T) {
	tmpFile := "test_change_cards_unique_key.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO change_cards (key, title) VALUES (?, ?)`, "CC-DUP-001", "First Card")
	if err != nil {
		t.Fatalf("Failed to insert first change card: %v", err)
	}

	_, err = db.Exec(`INSERT INTO change_cards (key, title) VALUES (?, ?)`, "CC-DUP-001", "Duplicate Card")
	if err == nil {
		t.Error("Expected UNIQUE constraint error for duplicate key, got none")
	}
}

// TestChangeCardsTableUpdatedAtTrigger tests that the updated_at trigger is created (TC-S2-07)
func TestChangeCardsTableUpdatedAtTrigger(t *testing.T) {
	tmpFile := "test_change_cards_trigger.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	var triggerCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='change_cards_updated_at'").
		Scan(&triggerCount)
	if err != nil {
		t.Fatalf("Failed to query trigger: %v", err)
	}
	if triggerCount == 0 {
		t.Error("Expected trigger 'change_cards_updated_at' to exist")
	}
}

// TestChangeCardsTableIdempotentMigration tests that running migration twice does not error (TC-S2-08)
func TestChangeCardsTableIdempotentMigration(t *testing.T) {
	tmpFile := "test_change_cards_idempotent.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database on first call: %v", err)
	}
	db.Close()

	db2, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database on second call: %v", err)
	}
	defer db2.Close()

	var tableName string
	err = db2.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='change_cards'").Scan(&tableName)
	if err != nil {
		t.Fatalf("change_cards table not found after second migration: %v", err)
	}
	if tableName != "change_cards" {
		t.Errorf("Expected 'change_cards', got '%s'", tableName)
	}
}
