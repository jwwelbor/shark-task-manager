package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration_RejectionReason verifies that database migration adds rejection_reason
// column to task_history table and creates the appropriate index (E07-F22)
func TestMigration_RejectionReason(t *testing.T) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Initialize database with migrations
	db, err := InitDB(dbPath)
	require.NoError(t, err, "migration failed")
	defer db.Close()

	// Verify rejection_reason column exists in task_history table
	var columnCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM pragma_table_info('task_history')
		WHERE name = 'rejection_reason'
	`).Scan(&columnCount)
	require.NoError(t, err, "failed to query rejection_reason column")
	assert.Equal(t, 1, columnCount, "rejection_reason column not found in task_history table")

	// Verify column is TEXT type
	var columnType string
	err = db.QueryRow(`
		SELECT type
		FROM pragma_table_info('task_history')
		WHERE name = 'rejection_reason'
	`).Scan(&columnType)
	require.NoError(t, err, "failed to query rejection_reason column type")
	assert.Equal(t, "TEXT", columnType, "rejection_reason column should be TEXT type")

	// Verify idx_task_history_rejection_reason index exists
	var indexCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type='index' AND name='idx_task_history_rejection_reason'
	`).Scan(&indexCount)
	require.NoError(t, err, "failed to query index")
	assert.Equal(t, 1, indexCount, "idx_task_history_rejection_reason index not found")

	// Verify task_notes table has 'rejection' in note_type constraint
	var noteTypesCheckSQL string
	err = db.QueryRow(`
		SELECT sql FROM sqlite_master
		WHERE type='table' AND name='task_notes'
	`).Scan(&noteTypesCheckSQL)
	require.NoError(t, err, "failed to query task_notes table schema")
	assert.Contains(t, noteTypesCheckSQL, "'rejection'", "task_notes table should allow 'rejection' note type")
}

// TestMigration_RejectionReason_Idempotent verifies that the migration can be run
// multiple times safely without errors (E07-F22)
func TestMigration_RejectionReason_Idempotent(t *testing.T) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// First migration
	db, err := InitDB(dbPath)
	require.NoError(t, err, "first migration failed")
	defer db.Close()

	// Get initial rejection_reason column count
	var initialCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM pragma_table_info('task_history')
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
		FROM pragma_table_info('task_history')
		WHERE name = 'rejection_reason'
	`).Scan(&finalCount)
	require.NoError(t, err, "failed to query final column count")
	assert.Equal(t, 1, finalCount, "rejection_reason column should still exist exactly once after second migration")
}

// TestMigration_TaskNotesNoteTypeConstraint verifies that task_notes table
// has been updated to include 'rejection' in the note_type CHECK constraint
func TestMigration_TaskNotesNoteTypeConstraint(t *testing.T) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Initialize database with migrations
	db, err := InitDB(dbPath)
	require.NoError(t, err, "migration failed")
	defer db.Close()

	// Get table schema
	var tableSchema string
	err = db.QueryRow(`
		SELECT sql FROM sqlite_master
		WHERE type='table' AND name='task_notes'
	`).Scan(&tableSchema)
	require.NoError(t, err, "failed to query task_notes schema")

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
		assert.Contains(t, tableSchema, noteType, "task_notes CHECK constraint should include note type: %s", noteType)
	}

	// Verify CHECK constraint syntax
	assert.Contains(t, tableSchema, "CHECK (note_type IN", "task_notes should have CHECK constraint on note_type")
}
