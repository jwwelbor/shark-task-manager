package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestBackfillSlugsFromFilePaths tests the slug backfill migration
// RED PHASE: This test should FAIL because BackfillSlugsFromFilePaths doesn't exist yet
func TestBackfillSlugsFromFilePaths(t *testing.T) {
	// Create in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create tables with slug columns
	_, err = db.Exec(`
		CREATE TABLE epics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			file_path TEXT,
			slug TEXT
		);

		CREATE TABLE features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			epic_id INTEGER NOT NULL,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			file_path TEXT,
			slug TEXT,
			FOREIGN KEY (epic_id) REFERENCES epics(id)
		);

		CREATE TABLE tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feature_id INTEGER NOT NULL,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			file_path TEXT,
			slug TEXT,
			FOREIGN KEY (feature_id) REFERENCES features(id)
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Insert test data with file paths (no slugs)
	// Epic: Extract "task-mgmt-cli-capabilities" from file path
	_, err = db.Exec(`
		INSERT INTO epics (id, key, title, file_path)
		VALUES (1, 'E05', 'Task Management CLI Capabilities', 'docs/plan/E05-task-mgmt-cli-capabilities/epic.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test epic: %v", err)
	}

	// Feature: Extract "incremental-sync-engine" from file path
	_, err = db.Exec(`
		INSERT INTO features (id, epic_id, key, title, file_path)
		VALUES (1, 1, 'E06-F04', 'Incremental Sync Engine', 'docs/plan/E06-intelligent-scanning/E06-F04-incremental-sync-engine/prd.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test feature: %v", err)
	}

	// Task: Extract "some-task-description" from file path
	_, err = db.Exec(`
		INSERT INTO tasks (id, feature_id, key, title, file_path)
		VALUES (1, 1, 'T-E04-F01-001', 'Some Task Description', 'docs/plan/E04-epic/E04-F01-feature/tasks/T-E04-F01-001-some-task-description.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test task: %v", err)
	}

	// Insert entities with NULL file_path (should remain NULL slug)
	_, err = db.Exec(`
		INSERT INTO epics (id, key, title, file_path) VALUES (2, 'E08', 'Epic Without Path', NULL);
		INSERT INTO features (id, epic_id, key, title, file_path) VALUES (2, 1, 'E08-F01', 'Feature Without Path', NULL);
		INSERT INTO tasks (id, feature_id, key, title, file_path) VALUES (2, 1, 'T-E08-F01-001', 'Task Without Path', NULL);
	`)
	if err != nil {
		t.Fatalf("Failed to insert entities with NULL file_path: %v", err)
	}

	// Verify slugs are NULL before migration
	var epicSlug, featureSlug, taskSlug sql.NullString
	err = db.QueryRow("SELECT slug FROM epics WHERE id = 1").Scan(&epicSlug)
	if err != nil {
		t.Fatalf("Failed to query epic slug: %v", err)
	}
	if epicSlug.Valid {
		t.Errorf("Epic slug should be NULL before migration, got: %s", epicSlug.String)
	}

	// Run the backfill migration
	// This will FAIL because the function doesn't exist yet
	err = BackfillSlugsFromFilePaths(db)
	if err != nil {
		t.Fatalf("BackfillSlugsFromFilePaths failed: %v", err)
	}

	// Verify epic slug was extracted correctly
	err = db.QueryRow("SELECT slug FROM epics WHERE id = 1").Scan(&epicSlug)
	if err != nil {
		t.Fatalf("Failed to query epic slug after migration: %v", err)
	}
	if !epicSlug.Valid || epicSlug.String != "task-mgmt-cli-capabilities" {
		t.Errorf("Expected epic slug 'task-mgmt-cli-capabilities', got: %v", epicSlug)
	}

	// Verify feature slug was extracted correctly
	err = db.QueryRow("SELECT slug FROM features WHERE id = 1").Scan(&featureSlug)
	if err != nil {
		t.Fatalf("Failed to query feature slug after migration: %v", err)
	}
	if !featureSlug.Valid || featureSlug.String != "incremental-sync-engine" {
		t.Errorf("Expected feature slug 'incremental-sync-engine', got: %v", featureSlug)
	}

	// Verify task slug was extracted correctly
	err = db.QueryRow("SELECT slug FROM tasks WHERE id = 1").Scan(&taskSlug)
	if err != nil {
		t.Fatalf("Failed to query task slug after migration: %v", err)
	}
	if !taskSlug.Valid || taskSlug.String != "some-task-description" {
		t.Errorf("Expected task slug 'some-task-description', got: %v", taskSlug)
	}

	// Verify entities with NULL file_path have NULL slug
	err = db.QueryRow("SELECT slug FROM epics WHERE id = 2").Scan(&epicSlug)
	if err != nil {
		t.Fatalf("Failed to query epic with NULL file_path: %v", err)
	}
	if epicSlug.Valid {
		t.Errorf("Epic with NULL file_path should have NULL slug, got: %s", epicSlug.String)
	}

	err = db.QueryRow("SELECT slug FROM features WHERE id = 2").Scan(&featureSlug)
	if err != nil {
		t.Fatalf("Failed to query feature with NULL file_path: %v", err)
	}
	if featureSlug.Valid {
		t.Errorf("Feature with NULL file_path should have NULL slug, got: %s", featureSlug.String)
	}

	err = db.QueryRow("SELECT slug FROM tasks WHERE id = 2").Scan(&taskSlug)
	if err != nil {
		t.Fatalf("Failed to query task with NULL file_path: %v", err)
	}
	if taskSlug.Valid {
		t.Errorf("Task with NULL file_path should have NULL slug, got: %s", taskSlug.String)
	}
}

// TestExtractEpicSlugFromPath tests epic slug extraction edge cases
func TestExtractEpicSlugFromPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "standard epic path",
			filePath: "docs/plan/E05-task-mgmt-cli-capabilities/epic.md",
			expected: "task-mgmt-cli-capabilities",
		},
		{
			name:     "epic with single word slug",
			filePath: "docs/plan/E07-enhancements/epic.md",
			expected: "enhancements",
		},
		{
			name:     "epic with multiple hyphens",
			filePath: "docs/plan/E10-advanced-task-intelligence-context-management/epic.md",
			expected: "advanced-task-intelligence-context-management",
		},
		{
			name:     "empty file path",
			filePath: "",
			expected: "",
		},
		{
			name:     "malformed path - no epic marker",
			filePath: "docs/plan/some-folder/epic.md",
			expected: "",
		},
		{
			name:     "malformed path - no epic.md",
			filePath: "docs/plan/E05-task-mgmt/other.md",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEpicSlugFromPath(tt.filePath)
			if result != tt.expected {
				t.Errorf("extractEpicSlugFromPath(%q) = %q, want %q", tt.filePath, result, tt.expected)
			}
		})
	}
}

// TestExtractFeatureSlugFromPath tests feature slug extraction edge cases
func TestExtractFeatureSlugFromPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "standard feature path with prd.md",
			filePath: "docs/plan/E06-intelligent-scanning/E06-F04-incremental-sync-engine/prd.md",
			expected: "incremental-sync-engine",
		},
		{
			name:     "feature path with feature.md",
			filePath: "docs/plan/E10-context/E10-F01-task-activity-notes-system/feature.md",
			expected: "task-activity-notes-system",
		},
		{
			name:     "feature with single word slug",
			filePath: "docs/plan/E07-enhancements/E07-F01-migrations/prd.md",
			expected: "migrations",
		},
		{
			name:     "empty file path",
			filePath: "",
			expected: "",
		},
		{
			name:     "malformed path - no feature marker",
			filePath: "docs/plan/E06-scanning/some-folder/prd.md",
			expected: "",
		},
		{
			name:     "malformed path - no prd.md or feature.md",
			filePath: "docs/plan/E06-scanning/E06-F04-sync/other.md",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFeatureSlugFromPath(tt.filePath)
			if result != tt.expected {
				t.Errorf("extractFeatureSlugFromPath(%q) = %q, want %q", tt.filePath, result, tt.expected)
			}
		})
	}
}

// TestExtractTaskSlugFromPath tests task slug extraction edge cases
func TestExtractTaskSlugFromPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "standard task path with slug",
			filePath: "docs/plan/E04-epic/E04-F01-feature/tasks/T-E04-F01-001-some-task-description.md",
			expected: "some-task-description",
		},
		{
			name:     "task without slug (key only)",
			filePath: "docs/plan/E04-epic/E04-F01-feature/tasks/T-E04-F01-001.md",
			expected: "",
		},
		{
			name:     "task with multi-word slug",
			filePath: "docs/plan/E05/E05-F01/tasks/T-E05-F01-002-implement-database-schema.md",
			expected: "implement-database-schema",
		},
		{
			name:     "absolute path task",
			filePath: "/home/user/projects/shark/docs/plan/E04-epic/E04-F01-feature/tasks/T-E04-F01-003-fix-bug.md",
			expected: "fix-bug",
		},
		{
			name:     "empty file path",
			filePath: "",
			expected: "",
		},
		{
			name:     "malformed path - no .md extension",
			filePath: "docs/plan/E04-epic/tasks/T-E04-F01-001-slug",
			expected: "",
		},
		{
			name:     "malformed path - invalid task key format",
			filePath: "docs/plan/E04-epic/tasks/INVALID-KEY-slug.md",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTaskSlugFromPath(tt.filePath)
			if result != tt.expected {
				t.Errorf("extractTaskSlugFromPath(%q) = %q, want %q", tt.filePath, result, tt.expected)
			}
		})
	}
}
