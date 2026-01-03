package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestMigrateDropCustomFolderPath tests the migration that removes custom_folder_path columns
func TestMigrateDropCustomFolderPath(t *testing.T) {
	// Create in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create minimal schema with custom_folder_path columns
	schema := `
		CREATE TABLE epics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
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

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Insert test data
	testData := `
		INSERT INTO epics (key, title, status, priority, file_path, custom_folder_path)
		VALUES ('E01', 'Test Epic', 'draft', 'high', 'docs/plan/E01/epic.md', 'custom/path');

		INSERT INTO features (epic_id, key, title, status, file_path, custom_folder_path)
		VALUES (1, 'E01-F01', 'Test Feature', 'draft', 'docs/plan/E01/F01/feature.md', 'custom/feature/path');
	`

	if _, err := db.Exec(testData); err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Verify custom_folder_path columns exist before migration
	var epicColumnExists, featureColumnExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'custom_folder_path'`).Scan(&epicColumnExists)
	if err != nil {
		t.Fatalf("Failed to check epics schema: %v", err)
	}
	if epicColumnExists != 1 {
		t.Fatal("custom_folder_path column should exist in epics table before migration")
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'custom_folder_path'`).Scan(&featureColumnExists)
	if err != nil {
		t.Fatalf("Failed to check features schema: %v", err)
	}
	if featureColumnExists != 1 {
		t.Fatal("custom_folder_path column should exist in features table before migration")
	}

	// Run migration
	if err := migrateDropCustomFolderPath(db); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify custom_folder_path columns are removed
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'custom_folder_path'`).Scan(&epicColumnExists)
	if err != nil {
		t.Fatalf("Failed to check epics schema after migration: %v", err)
	}
	if epicColumnExists != 0 {
		t.Error("custom_folder_path column should be removed from epics table after migration")
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'custom_folder_path'`).Scan(&featureColumnExists)
	if err != nil {
		t.Fatalf("Failed to check features schema after migration: %v", err)
	}
	if featureColumnExists != 0 {
		t.Error("custom_folder_path column should be removed from features table after migration")
	}

	// Verify indexes are also removed
	var epicIndexExists, featureIndexExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_epics_custom_folder_path'`).Scan(&epicIndexExists)
	if err != nil {
		t.Fatalf("Failed to check epics index: %v", err)
	}
	if epicIndexExists != 0 {
		t.Error("idx_epics_custom_folder_path index should be removed after migration")
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_features_custom_folder_path'`).Scan(&featureIndexExists)
	if err != nil {
		t.Fatalf("Failed to check features index: %v", err)
	}
	if featureIndexExists != 0 {
		t.Error("idx_features_custom_folder_path index should be removed after migration")
	}

	// Verify data integrity - file_path should still be present
	var epicFilePath, featureFilePath string
	err = db.QueryRow(`SELECT file_path FROM epics WHERE key = 'E01'`).Scan(&epicFilePath)
	if err != nil {
		t.Fatalf("Failed to read epic file_path after migration: %v", err)
	}
	if epicFilePath != "docs/plan/E01/epic.md" {
		t.Errorf("Epic file_path should be preserved, got: %s", epicFilePath)
	}

	err = db.QueryRow(`SELECT file_path FROM features WHERE key = 'E01-F01'`).Scan(&featureFilePath)
	if err != nil {
		t.Fatalf("Failed to read feature file_path after migration: %v", err)
	}
	if featureFilePath != "docs/plan/E01/F01/feature.md" {
		t.Errorf("Feature file_path should be preserved, got: %s", featureFilePath)
	}

	// Verify migration is idempotent (can be run multiple times safely)
	if err := migrateDropCustomFolderPath(db); err != nil {
		t.Fatalf("Migration should be idempotent, but failed on second run: %v", err)
	}
}

// TestMigrateDropCustomFolderPath_AlreadyMigrated tests migration on database that doesn't have the column
func TestMigrateDropCustomFolderPath_AlreadyMigrated(t *testing.T) {
	// Create in-memory database without custom_folder_path
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create minimal schema WITHOUT custom_folder_path columns
	schema := `
		CREATE TABLE epics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			priority TEXT NOT NULL,
			file_path TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			epic_id INTEGER NOT NULL,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			file_path TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Run migration - should be no-op
	if err := migrateDropCustomFolderPath(db); err != nil {
		t.Fatalf("Migration should succeed on already-migrated database: %v", err)
	}

	// Verify columns still don't exist
	var epicColumnExists, featureColumnExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'custom_folder_path'`).Scan(&epicColumnExists)
	if err != nil {
		t.Fatalf("Failed to check epics schema: %v", err)
	}
	if epicColumnExists != 0 {
		t.Error("custom_folder_path column should not exist in epics table")
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'custom_folder_path'`).Scan(&featureColumnExists)
	if err != nil {
		t.Fatalf("Failed to check features schema: %v", err)
	}
	if featureColumnExists != 0 {
		t.Error("custom_folder_path column should not exist in features table")
	}
}
