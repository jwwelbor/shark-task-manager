package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createLegacyRelationshipTablesForTest creates the three legacy relationship tables.
// This simulates a database that has the old schema before the E07-F39 migration.
func createLegacyRelationshipTablesForTest(t *testing.T, db *sql.DB) {
	t.Helper()

	ddl := `
CREATE TABLE IF NOT EXISTS task_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id INTEGER NOT NULL,
    to_task_id INTEGER NOT NULL,
    relationship_type TEXT NOT NULL DEFAULT 'depends_on',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS feature_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_feature_id INTEGER NOT NULL,
    to_feature_id INTEGER NOT NULL,
    relationship_type TEXT NOT NULL DEFAULT 'related_to',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS epic_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_epic_id INTEGER NOT NULL,
    to_epic_id INTEGER NOT NULL,
    relationship_type TEXT NOT NULL DEFAULT 'related_to',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`
	_, err := db.Exec(ddl)
	require.NoError(t, err, "failed to create legacy relationship tables")
}

// legacyTableExists checks whether a named table exists in the database.
func legacyTableExists(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`,
		tableName,
	).Scan(&count)
	require.NoError(t, err)
	return count > 0
}

// TestMigrateDropLegacyRelationshipTables_FreshDB validates AC-4 edge case (TC-4.1 fresh path):
// On a fresh database where the legacy tables never existed, the migration completes
// without error (DROP TABLE IF EXISTS semantics guarantee this).
func TestMigrateDropLegacyRelationshipTables_FreshDB(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Run migration on DB that never had legacy tables
	err := migrateDropLegacyRelationshipTables(db)
	require.NoError(t, err, "migration should succeed even when tables never existed")

	// Confirm tables do not exist (they were never there)
	assert.False(t, legacyTableExists(t, db, "task_relationships"))
	assert.False(t, legacyTableExists(t, db, "feature_relationships"))
	assert.False(t, legacyTableExists(t, db, "epic_relationships"))
}

// TestMigrateDropLegacyRelationshipTables_TablesPresent validates AC-4 (TC-4.1):
// After running the migration, tables that existed before are dropped.
func TestMigrateDropLegacyRelationshipTables_TablesPresent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	createLegacyRelationshipTablesForTest(t, db)

	// Verify tables exist before migration
	require.True(t, legacyTableExists(t, db, "task_relationships"), "task_relationships should exist before migration")
	require.True(t, legacyTableExists(t, db, "feature_relationships"), "feature_relationships should exist before migration")
	require.True(t, legacyTableExists(t, db, "epic_relationships"), "epic_relationships should exist before migration")

	// Run migration
	err := migrateDropLegacyRelationshipTables(db)
	require.NoError(t, err, "migration should succeed")

	// Verify tables are gone
	assert.False(t, legacyTableExists(t, db, "task_relationships"), "task_relationships should be absent after migration")
	assert.False(t, legacyTableExists(t, db, "feature_relationships"), "feature_relationships should be absent after migration")
	assert.False(t, legacyTableExists(t, db, "epic_relationships"), "epic_relationships should be absent after migration")
}

// TestMigrateDropLegacyRelationshipTables_Idempotent validates AC-8 (TC-8.2, TC-4.2):
// Running migration twice on the same DB does not error — tables are absent after
// the first run, and DROP TABLE IF EXISTS on the second run is a no-op.
func TestMigrateDropLegacyRelationshipTables_Idempotent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	createLegacyRelationshipTablesForTest(t, db)

	// First run
	err := migrateDropLegacyRelationshipTables(db)
	require.NoError(t, err, "first migration should succeed")

	// Confirm tables gone after first run
	assert.False(t, legacyTableExists(t, db, "task_relationships"))
	assert.False(t, legacyTableExists(t, db, "feature_relationships"))
	assert.False(t, legacyTableExists(t, db, "epic_relationships"))

	// Second run (tables already absent)
	err = migrateDropLegacyRelationshipTables(db)
	require.NoError(t, err, "second migration should succeed (idempotent)")

	// Confirm tables still absent
	assert.False(t, legacyTableExists(t, db, "task_relationships"))
	assert.False(t, legacyTableExists(t, db, "feature_relationships"))
	assert.False(t, legacyTableExists(t, db, "epic_relationships"))
}

// TestMigrateDropLegacyRelationshipTables_SchemaVersionBumped validates that
// CurrentSchemaVersion is at least 13 (the version at which the viewer_task_relationships
// view migration was added). Updated to 14 when E28-F01 added the tags/entity_tags migration.
func TestMigrateDropLegacyRelationshipTables_SchemaVersionBumped(t *testing.T) {
	assert.GreaterOrEqual(t, CurrentSchemaVersion, 13,
		"CurrentSchemaVersion must be >= 13 after viewer_task_relationships view migration is added")
}
