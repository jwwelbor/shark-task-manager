package db

import (
	"os"
	"strings"
	"testing"
)

// TestEntityNotesAcceptsBugEntityType tests that entity_notes accepts 'bug' as entity_type (TC-S5-01)
func TestEntityNotesAcceptsBugEntityType(t *testing.T) {
	tmpFile := "test_entity_notes_bug.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Insert a bug first so we have a valid entity to reference
	_, err = db.Exec(`INSERT INTO bugs (key, title) VALUES (?, ?)`, "BUG-EN-001", "Test Bug for Notes")
	if err != nil {
		t.Fatalf("Failed to insert bug: %v", err)
	}

	var bugID int64
	err = db.QueryRow("SELECT id FROM bugs WHERE key = 'BUG-EN-001'").Scan(&bugID)
	if err != nil {
		t.Fatalf("Failed to get bug ID: %v", err)
	}

	// Insert a note with entity_type = 'bug'
	_, err = db.Exec(`
		INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
		VALUES (?, ?, ?, ?)
	`, "bug", bugID, "comment", "This is a comment on a bug")
	if err != nil {
		t.Errorf("Expected entity_notes to accept entity_type='bug', got error: %v", err)
	}
}

// TestEntityNotesAcceptsChangeEntityType tests that entity_notes accepts 'change' as entity_type (TC-S5-02)
func TestEntityNotesAcceptsChangeEntityType(t *testing.T) {
	tmpFile := "test_entity_notes_change.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Insert a change_card first
	_, err = db.Exec(`INSERT INTO change_cards (key, title) VALUES (?, ?)`, "CC-EN-001", "Test Change Card for Notes")
	if err != nil {
		t.Fatalf("Failed to insert change_card: %v", err)
	}

	var ccID int64
	err = db.QueryRow("SELECT id FROM change_cards WHERE key = 'CC-EN-001'").Scan(&ccID)
	if err != nil {
		t.Fatalf("Failed to get change_card ID: %v", err)
	}

	// Insert a note with entity_type = 'change'
	_, err = db.Exec(`
		INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
		VALUES (?, ?, ?, ?)
	`, "change", ccID, "decision", "This is a decision note on a change card")
	if err != nil {
		t.Errorf("Expected entity_notes to accept entity_type='change', got error: %v", err)
	}
}

// TestEntityNotesRejectsInvalidEntityType originally asserted that the
// entity_notes.entity_type CHECK constraint rejected unknown entity types.
//
// B018 removed that CHECK and moved validation to the app-layer allowlist
// models.ValidEntityTypes (matching the bugs.linked_entity_type precedent).
// The test now asserts the inverse: raw SQL inserts of unknown entity types
// succeed at the DB layer, because validation happens in
// models.EntityNote.Validate() before reaching the repository. App-layer
// rejection is covered by internal/models/entity_test.go.
func TestEntityNotesRejectsInvalidEntityType(t *testing.T) {
	tmpFile := "test_entity_notes_invalid_type.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
		VALUES (?, ?, ?, ?)
	`, "invalid_type", 1, "comment", "Raw insert no longer rejected at DB layer post-B018")
	if err != nil {
		t.Errorf("Post-B018 the entity_type CHECK is removed; raw SQL insert "+
			"of an unknown entity_type should succeed at the DB layer "+
			"(app-layer validation rejects it instead). Got: %v", err)
	}
}

// TestEntityNotesStillAcceptsExistingEntityTypes tests that epic/feature/task still work (TC-S5-04)
func TestEntityNotesStillAcceptsExistingEntityTypes(t *testing.T) {
	tmpFile := "test_entity_notes_existing_types.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	existingTypes := []string{"epic", "feature", "task"}
	for i, entityType := range existingTypes {
		// Use a plausible entity_id value (1 is fine for testing constraint acceptance)
		_, err = db.Exec(`
			INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
			VALUES (?, ?, ?, ?)
		`, entityType, i+1, "comment", "Note for "+entityType)
		if err != nil {
			t.Errorf("Expected entity_notes to still accept entity_type='%s', got error: %v", entityType, err)
		}
	}
}

// TestEntityNotesSchemaContainsBugAndChange originally verified that the
// entity_notes CHECK constraint listed 'bug' and 'change'. After B018 the
// CHECK is gone entirely (validation lives in models.ValidEntityTypes),
// so this test now asserts the opposite: the column is declared NOT NULL
// with no inline CHECK, and the entity_type column itself still exists.
func TestEntityNotesSchemaContainsBugAndChange(t *testing.T) {
	tmpFile := "test_entity_notes_schema_check.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	var tableSql string
	err = db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='entity_notes'").Scan(&tableSql)
	if err != nil {
		t.Fatalf("Failed to get entity_notes schema: %v", err)
	}

	if !strings.Contains(tableSql, "entity_type TEXT NOT NULL") {
		t.Errorf("Expected entity_notes schema to declare entity_type as TEXT NOT NULL, got: %s", tableSql)
	}
	// Post-B018: the CHECK constraint is removed in favour of app-layer
	// validation. The schema must NOT contain an entity_type CHECK clause.
	if strings.Contains(tableSql, "entity_type IN") {
		t.Errorf("Expected entity_notes.entity_type CHECK to be removed post-B018, "+
			"but found 'entity_type IN' in schema: %s", tableSql)
	}
}

// TestEntityNotesCascadeDeleteBug tests that deleting a bug removes its notes (TC-S5-06)
func TestEntityNotesCascadeDeleteBug(t *testing.T) {
	tmpFile := "test_entity_notes_cascade_bug.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create a bug
	_, err = db.Exec(`INSERT INTO bugs (key, title) VALUES (?, ?)`, "BUG-CASCADE-001", "Bug to Delete")
	if err != nil {
		t.Fatalf("Failed to insert bug: %v", err)
	}

	var bugID int64
	err = db.QueryRow("SELECT id FROM bugs WHERE key = 'BUG-CASCADE-001'").Scan(&bugID)
	if err != nil {
		t.Fatalf("Failed to get bug ID: %v", err)
	}

	// Add a note to the bug
	_, err = db.Exec(`
		INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
		VALUES ('bug', ?, 'comment', 'Note that should be deleted')
	`, bugID)
	if err != nil {
		t.Fatalf("Failed to insert note for bug: %v", err)
	}

	// Verify note exists
	var noteCount int
	err = db.QueryRow("SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'bug' AND entity_id = ?", bugID).Scan(&noteCount)
	if err != nil {
		t.Fatalf("Failed to count notes: %v", err)
	}
	if noteCount != 1 {
		t.Fatalf("Expected 1 note before delete, got %d", noteCount)
	}

	// Delete the bug
	_, err = db.Exec("DELETE FROM bugs WHERE id = ?", bugID)
	if err != nil {
		t.Fatalf("Failed to delete bug: %v", err)
	}

	// Verify note is cascade deleted
	err = db.QueryRow("SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'bug' AND entity_id = ?", bugID).Scan(&noteCount)
	if err != nil {
		t.Fatalf("Failed to count notes after delete: %v", err)
	}
	if noteCount != 0 {
		t.Errorf("Expected 0 notes after bug deletion (cascade delete), got %d", noteCount)
	}
}

// TestEntityNotesCascadeDeleteChangeCard tests that deleting a change_card removes its notes (TC-S5-07)
func TestEntityNotesCascadeDeleteChangeCard(t *testing.T) {
	tmpFile := "test_entity_notes_cascade_change.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-shm")
	defer os.Remove(tmpFile + "-wal")

	db, err := InitDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create a change_card
	_, err = db.Exec(`INSERT INTO change_cards (key, title) VALUES (?, ?)`, "CC-CASCADE-001", "Change Card to Delete")
	if err != nil {
		t.Fatalf("Failed to insert change_card: %v", err)
	}

	var ccID int64
	err = db.QueryRow("SELECT id FROM change_cards WHERE key = 'CC-CASCADE-001'").Scan(&ccID)
	if err != nil {
		t.Fatalf("Failed to get change_card ID: %v", err)
	}

	// Add a note to the change_card
	_, err = db.Exec(`
		INSERT INTO entity_notes (entity_type, entity_id, note_type, content)
		VALUES ('change', ?, 'decision', 'Note that should be deleted')
	`, ccID)
	if err != nil {
		t.Fatalf("Failed to insert note for change_card: %v", err)
	}

	// Verify note exists
	var noteCount int
	err = db.QueryRow("SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'change' AND entity_id = ?", ccID).Scan(&noteCount)
	if err != nil {
		t.Fatalf("Failed to count notes: %v", err)
	}
	if noteCount != 1 {
		t.Fatalf("Expected 1 note before delete, got %d", noteCount)
	}

	// Delete the change_card
	_, err = db.Exec("DELETE FROM change_cards WHERE id = ?", ccID)
	if err != nil {
		t.Fatalf("Failed to delete change_card: %v", err)
	}

	// Verify note is cascade deleted
	err = db.QueryRow("SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'change' AND entity_id = ?", ccID).Scan(&noteCount)
	if err != nil {
		t.Fatalf("Failed to count notes after delete: %v", err)
	}
	if noteCount != 0 {
		t.Errorf("Expected 0 notes after change_card deletion (cascade delete), got %d", noteCount)
	}
}
