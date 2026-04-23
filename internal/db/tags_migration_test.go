package db

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration_TagsAndEntityTags_TablesCreated verifies that the tags and entity_tags
// tables are created by the migrateAddTagsAndEntityTags migration (E28-F01).
func TestMigration_TagsAndEntityTags_TablesCreated(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	// Verify tags table exists
	var tagsExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tags'`).Scan(&tagsExists)
	require.NoError(t, err, "failed to query for tags table")
	assert.Equal(t, 1, tagsExists, "tags table should exist after migration")

	// Verify entity_tags table exists
	var entityTagsExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='entity_tags'`).Scan(&entityTagsExists)
	require.NoError(t, err, "failed to query for entity_tags table")
	assert.Equal(t, 1, entityTagsExists, "entity_tags table should exist after migration")
}

// TestMigration_TagsTable_Schema verifies the tags table has the correct columns.
func TestMigration_TagsTable_Schema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	// Verify tags table schema
	rows, err := db.Query(`SELECT name, type FROM pragma_table_info('tags') ORDER BY cid`)
	require.NoError(t, err, "failed to query tags table info")
	defer rows.Close()

	type col struct{ name, typ string }
	var cols []col
	for rows.Next() {
		var c col
		require.NoError(t, rows.Scan(&c.name, &c.typ))
		cols = append(cols, c)
	}
	require.NoError(t, rows.Err())

	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.name
	}

	assert.Contains(t, colNames, "id", "tags table should have id column")
	assert.Contains(t, colNames, "name", "tags table should have name column")
	assert.Contains(t, colNames, "created_at", "tags table should have created_at column")
	assert.Contains(t, colNames, "updated_at", "tags table should have updated_at column")
}

// TestMigration_EntityTagsTable_Schema verifies the entity_tags table has the correct columns.
func TestMigration_EntityTagsTable_Schema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	// Verify entity_tags table schema
	rows, err := db.Query(`SELECT name, type FROM pragma_table_info('entity_tags') ORDER BY cid`)
	require.NoError(t, err, "failed to query entity_tags table info")
	defer rows.Close()

	type col struct{ name, typ string }
	var cols []col
	for rows.Next() {
		var c col
		require.NoError(t, rows.Scan(&c.name, &c.typ))
		cols = append(cols, c)
	}
	require.NoError(t, rows.Err())

	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.name
	}

	assert.Contains(t, colNames, "id", "entity_tags table should have id column")
	assert.Contains(t, colNames, "entity_type", "entity_tags table should have entity_type column")
	assert.Contains(t, colNames, "entity_id", "entity_tags table should have entity_id column")
	assert.Contains(t, colNames, "tag_id", "entity_tags table should have tag_id column")
	assert.Contains(t, colNames, "created_at", "entity_tags table should have created_at column")
}

// TestMigration_TagsIndexes verifies that all three expected indexes are created on the tags
// and entity_tags tables.
func TestMigration_TagsIndexes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	expectedIndexes := []string{
		"idx_tags_name",
		"idx_tags_created_at",
		"idx_entity_tags_entity",
		"idx_entity_tags_tag",
		"idx_entity_tags_tag_entity",
	}

	for _, indexName := range expectedIndexes {
		var count int
		err = db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`,
			indexName,
		).Scan(&count)
		require.NoError(t, err, "failed to query index %s", indexName)
		assert.Equal(t, 1, count, "index %s should exist after migration", indexName)
	}
}

// TestMigration_EntityTagsCascadeDeleteTriggers verifies that all six cascade-delete
// triggers for entity_tags are created, one per parent entity table.
func TestMigration_EntityTagsCascadeDeleteTriggers(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	expectedTriggers := []string{
		"entity_tags_cascade_delete_epic",
		"entity_tags_cascade_delete_feature",
		"entity_tags_cascade_delete_task",
		"entity_tags_cascade_delete_bug",
		"entity_tags_cascade_delete_change",
		"entity_tags_cascade_delete_idea",
	}

	for _, triggerName := range expectedTriggers {
		var count int
		err = db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?`,
			triggerName,
		).Scan(&count)
		require.NoError(t, err, "failed to query trigger %s", triggerName)
		assert.Equal(t, 1, count, "trigger %s should exist after migration", triggerName)
	}
}

// TestMigration_TagsAndEntityTags_Idempotent verifies that running the migration
// multiple times (by reopening the database) does not produce errors.
func TestMigration_TagsAndEntityTags_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// First init
	db, err := InitDB(dbPath)
	require.NoError(t, err, "first InitDB should succeed")
	db.Close()

	// Second init — schema is already at CurrentSchemaVersion, ApplySchemaIfNeeded
	// must not error even though migrateAddTagsAndEntityTags uses IF NOT EXISTS.
	db, err = InitDB(dbPath)
	require.NoError(t, err, "second InitDB should succeed (migration is idempotent)")
	defer db.Close()

	// Confirm tables still exist after second run
	var tagsExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tags'`).Scan(&tagsExists)
	require.NoError(t, err)
	assert.Equal(t, 1, tagsExists, "tags table should still exist after second migration run")
}

// TestMigration_TagsAndEntityTags_SchemaVersion verifies that the schema version is
// bumped to 14 (CurrentSchemaVersion) after the migration.
func TestMigration_TagsAndEntityTags_SchemaVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	var version int
	err = db.QueryRow(`SELECT version FROM schema_version ORDER BY version DESC LIMIT 1`).Scan(&version)
	require.NoError(t, err, "failed to read schema_version")
	assert.Equal(t, CurrentSchemaVersion, version,
		"schema_version should equal CurrentSchemaVersion (%d) after migration", CurrentSchemaVersion)
}

// TestMigration_EntityTagsCascadeDeleteTriggers_FunctionalEpic verifies that
// deleting an epic removes its associated entity_tags rows (cascade behavior).
func TestMigration_EntityTagsCascadeDeleteTriggers_FunctionalEpic(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	// Insert a test epic (priority has a CHECK constraint: 'high', 'medium', 'low')
	res, err := db.Exec(`INSERT INTO epics (key, title, status, priority) VALUES ('E99', 'Test Epic', 'todo', 'medium')`)
	require.NoError(t, err, "failed to insert test epic")
	epicID, err := res.LastInsertId()
	require.NoError(t, err)

	// Insert a test tag
	res, err = db.Exec(`INSERT INTO tags (name) VALUES ('test-tag')`)
	require.NoError(t, err, "failed to insert test tag")
	tagID, err := res.LastInsertId()
	require.NoError(t, err)

	// Insert entity_tags row linking epic to tag
	_, err = db.Exec(`INSERT INTO entity_tags (entity_type, entity_id, tag_id) VALUES ('epic', ?, ?)`,
		epicID, tagID)
	require.NoError(t, err, "failed to insert entity_tags row")

	// Verify row exists
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM entity_tags WHERE entity_type='epic' AND entity_id=?`, epicID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "entity_tags row should exist before delete")

	// Delete the epic — trigger should cascade
	_, err = db.Exec(`DELETE FROM epics WHERE id=?`, epicID)
	require.NoError(t, err, "failed to delete epic")

	// Verify entity_tags row is gone
	err = db.QueryRow(`SELECT COUNT(*) FROM entity_tags WHERE entity_type='epic' AND entity_id=?`, epicID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "entity_tags row should be deleted by cascade trigger after epic delete")
}

// TestMigration_EntityTagsTable_CheckConstraint verifies that entity_tags rejects
// entity_type values outside the allowed set.
func TestMigration_EntityTagsTable_CheckConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	// Insert a tag to reference
	res, err := db.Exec(`INSERT INTO tags (name) VALUES ('constraint-test-tag')`)
	require.NoError(t, err)
	tagID, err := res.LastInsertId()
	require.NoError(t, err)

	// Attempt to insert an invalid entity_type — should fail CHECK constraint
	_, err = db.Exec(`INSERT INTO entity_tags (entity_type, entity_id, tag_id) VALUES ('invalid_type', 1, ?)`, tagID)
	require.Error(t, err, "inserting an invalid entity_type into entity_tags should fail")
	assert.True(t, strings.Contains(err.Error(), "CHECK constraint") ||
		strings.Contains(err.Error(), "UNIQUE constraint") ||
		strings.Contains(err.Error(), "constraint"),
		"error should mention a constraint failure, got: %v", err)
}

// TestMigration_TagsTable_UniqueNameConstraint verifies that tags rejects
// duplicate names (UNIQUE COLLATE NOCASE constraint).
func TestMigration_TagsTable_UniqueNameConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	_, err = db.Exec(`INSERT INTO tags (name) VALUES ('duplicate-tag')`)
	require.NoError(t, err, "first insert should succeed")

	_, err = db.Exec(`INSERT INTO tags (name) VALUES ('duplicate-tag')`)
	require.Error(t, err, "second insert with same name should fail (UNIQUE constraint)")
}

// TestMigration_DirectMigrateAddTagsAndEntityTags verifies that calling the migration
// function directly on a fully-initialized DB works correctly and is idempotent.
// The function references parent entity tables (epics, bugs, etc.) via triggers, so
// it must be called on a DB that has the full schema already applied.
func TestMigration_DirectMigrateAddTagsAndEntityTags(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Use InitDB so the parent tables exist before we call the function directly.
	db, err := InitDB(dbPath)
	require.NoError(t, err, "InitDB should succeed")
	defer db.Close()

	// Call the migration function a second time — must be idempotent (IF NOT EXISTS).
	err = migrateAddTagsAndEntityTags(db)
	require.NoError(t, err, "migrateAddTagsAndEntityTags should be idempotent when called a second time")

	// Verify both tables still exist
	for _, tbl := range []string{"tags", "entity_tags"} {
		var cnt int
		err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&cnt)
		require.NoError(t, err)
		assert.Equal(t, 1, cnt, "table %s should exist after second call", tbl)
	}
}
