package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration_RejectionReason verifies that the database schema includes rejection_reason
// column in entity_history table (originally E07-F22 for task_history, now in
// polymorphic entity_history via E21-F08).
func TestMigration_RejectionReason(t *testing.T) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Initialize database with migrations
	db, err := InitDB(dbPath)
	require.NoError(t, err, "migration failed")
	defer db.Close()

	// Verify rejection_reason column exists in entity_history table
	// (post E21-F08: task_history is replaced by entity_history)
	var columnCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM pragma_table_info('entity_history')
		WHERE name = 'rejection_reason'
	`).Scan(&columnCount)
	require.NoError(t, err, "failed to query rejection_reason column")
	assert.Equal(t, 1, columnCount, "rejection_reason column not found in entity_history table")

	// Verify column is TEXT type
	var columnType string
	err = db.QueryRow(`
		SELECT type
		FROM pragma_table_info('entity_history')
		WHERE name = 'rejection_reason'
	`).Scan(&columnType)
	require.NoError(t, err, "failed to query rejection_reason column type")
	assert.Equal(t, "TEXT", columnType, "rejection_reason column should be TEXT type")

	// Verify entity_notes table has 'rejection' in note_type constraint
	// (task_notes is renamed to task_notes_backup after entity_notes migration)
	var noteTypesCheckSQL string
	err = db.QueryRow(`
		SELECT sql FROM sqlite_master
		WHERE type='table' AND name='entity_notes'
	`).Scan(&noteTypesCheckSQL)
	require.NoError(t, err, "failed to query entity_notes table schema")
	assert.Contains(t, noteTypesCheckSQL, "'rejection'", "entity_notes table should allow 'rejection' note type")
}

// TestMigration_RejectionReason_Idempotent verifies that the migration can be run
// multiple times safely without errors (E07-F22, updated for E21-F08 polymorphic schema)
func TestMigration_RejectionReason_Idempotent(t *testing.T) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// First migration
	db, err := InitDB(dbPath)
	require.NoError(t, err, "first migration failed")
	defer db.Close()

	// Verify rejection_reason column exists in entity_history
	var initialCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM pragma_table_info('entity_history')
		WHERE name = 'rejection_reason'
	`).Scan(&initialCount)
	require.NoError(t, err, "failed to query initial column count")
	assert.Equal(t, 1, initialCount, "rejection_reason column should exist after first migration")

	// Second migration (should not fail)
	// Close and reopen to trigger migration again
	db.Close()
	db, err = InitDB(dbPath)
	require.NoError(t, err, "second migration failed - migration should be idempotent")
	defer db.Close()

	// Verify column still exists (shouldn't be duplicated)
	var finalCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM pragma_table_info('entity_history')
		WHERE name = 'rejection_reason'
	`).Scan(&finalCount)
	require.NoError(t, err, "failed to query final column count")
	assert.Equal(t, 1, finalCount, "rejection_reason column should still exist exactly once after second migration")
}

// TestMigration_EntityNotesNoteTypeConstraint verifies that entity_notes table
// includes 'rejection' in the note_type CHECK constraint (E16-F04 migration from task_notes)
func TestMigration_EntityNotesNoteTypeConstraint(t *testing.T) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Initialize database with migrations
	db, err := InitDB(dbPath)
	require.NoError(t, err, "migration failed")
	defer db.Close()

	// Get entity_notes table schema (task_notes is renamed to task_notes_backup after migration)
	var tableSchema string
	err = db.QueryRow(`
		SELECT sql FROM sqlite_master
		WHERE type='table' AND name='entity_notes'
	`).Scan(&tableSchema)
	require.NoError(t, err, "failed to query entity_notes schema")

	// Verify all expected note types are in the constraint
	expectedTypes := []string{
		"'comment'",
		"'decision'",
		"'blocker'",
		"'solution'",
		"'reference'",
		"'implementation'",
		"'testing'",
		"'future'",
		"'question'",
		"'rejection'",
	}

	for _, noteType := range expectedTypes {
		assert.Contains(t, tableSchema, noteType, "entity_notes CHECK constraint should include note type: %s", noteType)
	}

	// Verify CHECK constraint syntax
	assert.Contains(t, tableSchema, "CHECK (note_type IN", "entity_notes should have CHECK constraint on note_type")
}

// TestMigration_DisplayViewsExistAfterEntityNotesMigration verifies that the
// epic_display_data, feature_display_data, and task_display_data views exist
// after all migrations complete, including migrateEntityNotesExpandEntityTypes
// which drops them (E18-F02 regression fix).
func TestMigration_DisplayViewsExistAfterEntityNotesMigration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "migration failed")
	defer db.Close()

	for _, viewName := range []string{"epic_display_data", "feature_display_data", "task_display_data"} {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name=?
		`, viewName).Scan(&count)
		require.NoError(t, err, "failed to query sqlite_master for view %s", viewName)
		assert.Equal(t, 1, count, "view %s should exist after all migrations complete", viewName)
	}
}

// TestMigration_ApplySchemaAndMigrationsRestoresDroppedViews verifies that
// ApplySchemaAndMigrations (used by the Turso/cloud path) restores display
// views that were previously dropped by migrateEntityNotesExpandEntityTypes
// (E18-F02 regression fix).
func TestMigration_ApplySchemaAndMigrationsRestoresDroppedViews(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Initialize fresh database (creates all views)
	db, err := InitDB(dbPath)
	require.NoError(t, err, "initial migration failed")

	// Simulate the bug: manually drop the display views as migrateEntityNotesExpandEntityTypes did
	for _, view := range []string{"epic_display_data", "feature_display_data", "task_display_data"} {
		_, err := db.Exec("DROP VIEW IF EXISTS " + view)
		require.NoError(t, err, "failed to drop view %s", view)
	}

	// Verify views are gone
	for _, viewName := range []string{"epic_display_data", "feature_display_data", "task_display_data"} {
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name=?`, viewName).Scan(&count)
		assert.Equal(t, 0, count, "setup: view %s should be dropped", viewName)
	}

	// ApplySchemaAndMigrations simulates what the Turso/cloud path does on re-migration
	err = ApplySchemaAndMigrations(db)
	require.NoError(t, err, "ApplySchemaAndMigrations failed")
	db.Close()

	// Reopen to verify views persisted
	db2, err := InitDB(dbPath)
	require.NoError(t, err)
	defer db2.Close()

	for _, viewName := range []string{"epic_display_data", "feature_display_data", "task_display_data"} {
		var count int
		err := db2.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name=?`, viewName).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "view %s should be restored by ApplySchemaAndMigrations", viewName)
	}
}

// TestMigration_ApplySchemaIfNeeded_RestoresViewsWhenVersionBehind verifies that
// ApplySchemaIfNeeded (the skip_migrations Turso path) restores dropped display
// views when the recorded schema version is behind CurrentSchemaVersion.
// This ensures that bumping CurrentSchemaVersion forces re-migration on Turso
// databases that had views dropped by the E18-F01 bug (E18-F02 regression fix).
func TestMigration_ApplySchemaIfNeeded_RestoresViewsWhenVersionBehind(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Create fully-migrated database
	db, err := InitDB(dbPath)
	require.NoError(t, err, "initial migration failed")

	// Simulate bug state: drop display views
	for _, view := range []string{"epic_display_data", "feature_display_data", "task_display_data"} {
		_, err := db.Exec("DROP VIEW IF EXISTS " + view)
		require.NoError(t, err)
	}

	// Simulate old schema version (pre-bump) so ApplySchemaIfNeeded triggers.
	// Clear the table first since setSchemaVersion inserts and getSchemaVersion reads the max.
	_, err = db.Exec(`DELETE FROM schema_version`)
	require.NoError(t, err, "failed to clear schema_version")
	err = setSchemaVersion(db, CurrentSchemaVersion-1)
	require.NoError(t, err, "failed to set old schema version")

	// ApplySchemaIfNeeded should detect version is behind and re-apply
	applied, err := ApplySchemaIfNeeded(db)
	require.NoError(t, err, "ApplySchemaIfNeeded failed")
	assert.True(t, applied, "ApplySchemaIfNeeded should have applied schema since version was behind")

	// Verify all display views are restored
	for _, viewName := range []string{"epic_display_data", "feature_display_data", "task_display_data"} {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name=?`, viewName).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "view %s should be restored after ApplySchemaIfNeeded", viewName)
	}

	db.Close()
}
