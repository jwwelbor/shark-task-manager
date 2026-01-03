package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestInitDB_RunsAllMigrations verifies that InitDB correctly runs all migrations
// including the E07-F19 custom_folder_path column drop
func TestInitDB_RunsAllMigrations(t *testing.T) {
	// Create a temporary database file
	tmpDB := "test_initdb_migrations.db"
	defer os.Remove(tmpDB)
	defer os.Remove(tmpDB + "-shm")
	defer os.Remove(tmpDB + "-wal")

	// Initialize database - should run all migrations
	db, err := InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify custom_folder_path columns DO NOT exist (E07-F19 migration)
	var epicColumnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'custom_folder_path'
	`).Scan(&epicColumnExists)
	if err != nil {
		t.Fatalf("Failed to check epics schema: %v", err)
	}
	if epicColumnExists != 0 {
		t.Error("custom_folder_path column should not exist in epics table after InitDB")
	}

	var featureColumnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'custom_folder_path'
	`).Scan(&featureColumnExists)
	if err != nil {
		t.Fatalf("Failed to check features schema: %v", err)
	}
	if featureColumnExists != 0 {
		t.Error("custom_folder_path column should not exist in features table after InitDB")
	}

	// Verify file_path columns exist
	var epicFilePathExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'file_path'
	`).Scan(&epicFilePathExists)
	if err != nil {
		t.Fatalf("Failed to check epics schema for file_path: %v", err)
	}
	if epicFilePathExists != 1 {
		t.Error("file_path column should exist in epics table")
	}

	var featureFilePathExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'file_path'
	`).Scan(&featureFilePathExists)
	if err != nil {
		t.Fatalf("Failed to check features schema for file_path: %v", err)
	}
	if featureFilePathExists != 1 {
		t.Error("file_path column should exist in features table")
	}

	// Verify slug columns exist (E07-F11 migration)
	var epicSlugExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'slug'
	`).Scan(&epicSlugExists)
	if err != nil {
		t.Fatalf("Failed to check epics schema for slug: %v", err)
	}
	if epicSlugExists != 1 {
		t.Error("slug column should exist in epics table")
	}

	// Verify execution_order columns exist
	var featureExecOrderExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'execution_order'
	`).Scan(&featureExecOrderExists)
	if err != nil {
		t.Fatalf("Failed to check features schema for execution_order: %v", err)
	}
	if featureExecOrderExists != 1 {
		t.Error("execution_order column should exist in features table")
	}

	// Verify status_override column exists (E07-F14 migration)
	var statusOverrideExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'status_override'
	`).Scan(&statusOverrideExists)
	if err != nil {
		t.Fatalf("Failed to check features schema for status_override: %v", err)
	}
	if statusOverrideExists != 1 {
		t.Error("status_override column should exist in features table")
	}

	// Verify custom_folder_path indexes DO NOT exist
	var epicIndexExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_epics_custom_folder_path'
	`).Scan(&epicIndexExists)
	if err != nil {
		t.Fatalf("Failed to check epics index: %v", err)
	}
	if epicIndexExists != 0 {
		t.Error("idx_epics_custom_folder_path index should not exist after InitDB")
	}

	var featureIndexExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_features_custom_folder_path'
	`).Scan(&featureIndexExists)
	if err != nil {
		t.Fatalf("Failed to check features index: %v", err)
	}
	if featureIndexExists != 0 {
		t.Error("idx_features_custom_folder_path index should not exist after InitDB")
	}

	// Test data insertion with file_path (but not custom_folder_path)
	_, err = db.Exec(`
		INSERT INTO epics (key, title, status, priority, file_path)
		VALUES ('E99', 'Test Epic', 'draft', 'high', 'docs/plan/E99/epic.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test epic: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO features (epic_id, key, title, status, file_path)
		VALUES (1, 'E99-F01', 'Test Feature', 'draft', 'docs/plan/E99/F01/feature.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test feature: %v", err)
	}

	// Verify data was inserted correctly
	var epicFilePath string
	err = db.QueryRow(`SELECT file_path FROM epics WHERE key = 'E99'`).Scan(&epicFilePath)
	if err != nil {
		t.Fatalf("Failed to read epic file_path: %v", err)
	}
	if epicFilePath != "docs/plan/E99/epic.md" {
		t.Errorf("Expected epic file_path 'docs/plan/E99/epic.md', got: %s", epicFilePath)
	}

	// Cleanup test data
	_, _ = db.Exec(`DELETE FROM features WHERE key = 'E99-F01'`)
	_, _ = db.Exec(`DELETE FROM epics WHERE key = 'E99'`)
}

// TestInitDB_ExistingDatabaseWithCustomFolderPath tests that InitDB correctly
// migrates an existing database that has custom_folder_path columns
func TestInitDB_ExistingDatabaseWithCustomFolderPath(t *testing.T) {
	// Create a temporary database file
	tmpDB := "test_initdb_existing.db"
	defer os.Remove(tmpDB)
	defer os.Remove(tmpDB + "-shm")
	defer os.Remove(tmpDB + "-wal")

	// Create database with old schema (with custom_folder_path)
	db, err := sql.Open("sqlite3", tmpDB+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create old schema with custom_folder_path columns
	oldSchema := `
		CREATE TABLE epics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL,
			priority TEXT NOT NULL,
			file_path TEXT,
			custom_folder_path TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_epics_custom_folder_path ON epics(custom_folder_path);

		CREATE TABLE features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			epic_id INTEGER NOT NULL,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			file_path TEXT,
			custom_folder_path TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
		);

		CREATE INDEX idx_features_custom_folder_path ON features(custom_folder_path);
	`

	if _, err := db.Exec(oldSchema); err != nil {
		t.Fatalf("Failed to create old schema: %v", err)
	}

	// Insert test data with custom_folder_path
	_, err = db.Exec(`
		INSERT INTO epics (key, title, status, priority, file_path, custom_folder_path)
		VALUES ('E88', 'Old Epic', 'draft', 'high', 'docs/plan/E88/epic.md', 'custom/epic/path')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test epic: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO features (epic_id, key, title, status, file_path, custom_folder_path)
		VALUES (1, 'E88-F01', 'Old Feature', 'draft', 'docs/plan/E88/F01/feature.md', 'custom/feature/path')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test feature: %v", err)
	}

	db.Close()

	// Now run InitDB on the existing database - should migrate it
	db, err = InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize existing database: %v", err)
	}
	defer db.Close()

	// Verify custom_folder_path columns are removed
	var epicColumnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'custom_folder_path'
	`).Scan(&epicColumnExists)
	if err != nil {
		t.Fatalf("Failed to check epics schema: %v", err)
	}
	if epicColumnExists != 0 {
		t.Error("custom_folder_path column should be removed from epics table after migration")
	}

	var featureColumnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'custom_folder_path'
	`).Scan(&featureColumnExists)
	if err != nil {
		t.Fatalf("Failed to check features schema: %v", err)
	}
	if featureColumnExists != 0 {
		t.Error("custom_folder_path column should be removed from features table after migration")
	}

	// Verify data is preserved - file_path should still exist
	var epicFilePath string
	err = db.QueryRow(`SELECT file_path FROM epics WHERE key = 'E88'`).Scan(&epicFilePath)
	if err != nil {
		t.Fatalf("Failed to read epic file_path after migration: %v", err)
	}
	if epicFilePath != "docs/plan/E88/epic.md" {
		t.Errorf("Expected epic file_path 'docs/plan/E88/epic.md', got: %s", epicFilePath)
	}

	var featureFilePath string
	err = db.QueryRow(`SELECT file_path FROM features WHERE key = 'E88-F01'`).Scan(&featureFilePath)
	if err != nil {
		t.Fatalf("Failed to read feature file_path after migration: %v", err)
	}
	if featureFilePath != "docs/plan/E88/F01/feature.md" {
		t.Errorf("Expected feature file_path 'docs/plan/E88/F01/feature.md', got: %s", featureFilePath)
	}

	// Verify indexes are removed
	var epicIndexExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_epics_custom_folder_path'
	`).Scan(&epicIndexExists)
	if err != nil {
		t.Fatalf("Failed to check epics index: %v", err)
	}
	if epicIndexExists != 0 {
		t.Error("idx_epics_custom_folder_path index should be removed after migration")
	}

	var featureIndexExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_features_custom_folder_path'
	`).Scan(&featureIndexExists)
	if err != nil {
		t.Fatalf("Failed to check features index: %v", err)
	}
	if featureIndexExists != 0 {
		t.Error("idx_features_custom_folder_path index should be removed after migration")
	}
}
