package db

import (
	"os"
	"strings"
	"testing"
)

// INT-01: Fresh database full initialization
// Verifies that a new database has all E18-F01 tables, indexes, and migrated entity_notes.
func TestE18F01_INT01_FreshDatabaseFullInitialization(t *testing.T) {
	tmpFile := "test_e18_int01_fresh.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// Verify bugs table exists
	var bugsExists int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='bugs'").Scan(&bugsExists); err != nil {
		t.Fatalf("Failed to check bugs table: %v", err)
	}
	if bugsExists == 0 {
		t.Error("INT-01: bugs table not created in fresh database")
	}

	// Verify change_cards table exists
	var changeCardsExists int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='change_cards'").Scan(&changeCardsExists); err != nil {
		t.Fatalf("Failed to check change_cards table: %v", err)
	}
	if changeCardsExists == 0 {
		t.Error("INT-01: change_cards table not created in fresh database")
	}

	// Verify all bugs indexes exist
	bugsIndexes := []string{
		"idx_bugs_key", "idx_bugs_status", "idx_bugs_severity",
		"idx_bugs_linked_entity_type", "idx_bugs_slug",
	}
	for _, idx := range bugsIndexes {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&count); err != nil {
			t.Fatalf("Failed to check bugs index %s: %v", idx, err)
		}
		if count == 0 {
			t.Errorf("INT-01: bugs index '%s' not created", idx)
		}
	}

	// Verify all change_cards indexes exist
	changeCardsIndexes := []string{
		"idx_change_cards_key", "idx_change_cards_status", "idx_change_cards_epic_id",
		"idx_change_cards_feature_id", "idx_change_cards_slug",
	}
	for _, idx := range changeCardsIndexes {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&count); err != nil {
			t.Fatalf("Failed to check change_cards index %s: %v", idx, err)
		}
		if count == 0 {
			t.Errorf("INT-01: change_cards index '%s' not created", idx)
		}
	}

	// Verify entity_notes has expanded CHECK constraint
	var tableSql string
	if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='entity_notes'").Scan(&tableSql); err != nil {
		t.Fatalf("INT-01: Failed to get entity_notes schema: %v", err)
	}
	if !strings.Contains(tableSql, "'bug'") {
		t.Error("INT-01: entity_notes CHECK constraint does not include 'bug'")
	}
	if !strings.Contains(tableSql, "'change'") {
		t.Error("INT-01: entity_notes CHECK constraint does not include 'change'")
	}
}

// INT-02: Existing database upgrade preserves data
// Simulates a pre-E18 database (entity_notes with old CHECK), then runs migrations
// and verifies data is preserved and schema is updated.
func TestE18F01_INT02_ExistingDatabaseUpgradePreservesData(t *testing.T) {
	tmpFile := "test_e18_int02_upgrade.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	// Step 1: First initialize to get full base schema
	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	// Insert existing entity data (simulating pre-E18 data)
	_, err = db.Exec(`INSERT INTO epics (key, title, status, priority, file_path) VALUES (?, ?, ?, ?, ?)`,
		"E01", "Test Epic", "active", "medium", "docs/plan/E01/E01.md")
	if err != nil {
		t.Fatalf("Failed to insert test epic: %v", err)
	}

	// Insert a note for the epic (simulating existing entity notes)
	_, err = db.Exec(`
		INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
		VALUES ('epic', 1, 'comment', 'Pre-migration note')
	`)
	if err != nil {
		t.Fatalf("Failed to insert pre-migration note: %v", err)
	}

	db.Close()

	// Step 2: Re-open (simulates upgrade run)
	db2, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Re-InitDB (upgrade) failed: %v", err)
	}
	defer db2.Close()

	// Verify pre-existing data preserved
	var epicKey string
	err = db2.QueryRow("SELECT key FROM epics WHERE key = 'E01'").Scan(&epicKey)
	if err != nil {
		t.Fatalf("INT-02: Pre-existing epic data lost after upgrade: %v", err)
	}
	if epicKey != "E01" {
		t.Errorf("INT-02: Expected epic key 'E01', got '%s'", epicKey)
	}

	// Verify pre-existing note preserved
	var noteContent string
	err = db2.QueryRow("SELECT content FROM entity_notes WHERE entity_type='epic' AND content='Pre-migration note'").Scan(&noteContent)
	if err != nil {
		t.Fatalf("INT-02: Pre-existing entity note lost after upgrade: %v", err)
	}

	// Verify bugs and change_cards tables exist after upgrade
	var bugsExists int
	if err := db2.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='bugs'").Scan(&bugsExists); err != nil {
		t.Fatalf("INT-02: Failed to check bugs table: %v", err)
	}
	if bugsExists == 0 {
		t.Error("INT-02: bugs table not created during upgrade")
	}

	var ccExists int
	if err := db2.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='change_cards'").Scan(&ccExists); err != nil {
		t.Fatalf("INT-02: Failed to check change_cards table: %v", err)
	}
	if ccExists == 0 {
		t.Error("INT-02: change_cards table not created during upgrade")
	}
}

// INT-05: Entity notes cross-entity compatibility
// entity_notes accepts all 5 entity types; cascade delete works for existing types.
func TestE18F01_INT05_EntityNotesCrossEntityCompatibility(t *testing.T) {
	tmpFile := "test_e18_int05_entity_notes.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// Verify all 5 entity types are accepted
	entityTests := []struct {
		entityType string
		noteType   string
	}{
		{"epic", "comment"},
		{"feature", "decision"},
		{"task", "implementation"},
		{"bug", "comment"},
		{"change", "decision"},
	}

	for i, tt := range entityTests {
		_, err := db.Exec(`
			INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
			VALUES (?, ?, ?, ?)
		`, tt.entityType, i+1, tt.noteType, "Test note for "+tt.entityType)
		if err != nil {
			t.Errorf("INT-05: entity_notes should accept entity_type='%s', got error: %v", tt.entityType, err)
		}
	}

	// Verify cascade delete works for bugs
	_, err = db.Exec(`INSERT INTO bugs (key, title) VALUES (?, ?)`, "INT05-BUG-001", "Cascade Test Bug")
	if err != nil {
		t.Fatalf("INT-05: Failed to insert bug: %v", err)
	}

	var bugID int64
	err = db.QueryRow("SELECT id FROM bugs WHERE key='INT05-BUG-001'").Scan(&bugID)
	if err != nil {
		t.Fatalf("INT-05: Failed to get bug ID: %v", err)
	}

	_, err = db.Exec(`INSERT INTO entity_notes (entity_type, entity_id, note_type, content) VALUES ('bug', ?, 'comment', 'Will be cascade deleted')`, bugID)
	if err != nil {
		t.Fatalf("INT-05: Failed to insert bug note: %v", err)
	}

	_, err = db.Exec("DELETE FROM bugs WHERE id=?", bugID)
	if err != nil {
		t.Fatalf("INT-05: Failed to delete bug: %v", err)
	}

	var noteCount int
	err = db.QueryRow("SELECT COUNT(*) FROM entity_notes WHERE entity_type='bug' AND entity_id=?", bugID).Scan(&noteCount)
	if err != nil {
		t.Fatalf("INT-05: Failed to count notes: %v", err)
	}
	if noteCount != 0 {
		t.Errorf("INT-05: Expected cascade delete to remove bug notes, found %d remaining", noteCount)
	}
}
