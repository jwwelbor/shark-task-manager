package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestBackfillSlugsFromFilePaths tests the slug backfill migration
// This test verifies the three-phase approach: task paths, feature paths, own paths
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
	// Epic 1: Has epic.md file path (will be extracted from own path - Phase 3)
	_, err = db.Exec(`
		INSERT INTO epics (id, key, title, file_path)
		VALUES (1, 'E05', 'Task Management CLI Capabilities', 'docs/plan/E05-task-mgmt-cli-capabilities/epic.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test epic 1: %v", err)
	}

	// Epic 2: NO file_path (will be extracted from feature path - Phase 2)
	_, err = db.Exec(`
		INSERT INTO epics (id, key, title, file_path)
		VALUES (2, 'E06', 'Intelligent Scanning', NULL)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test epic 2: %v", err)
	}

	// Epic 3: NO file_path (will be extracted from task path - Phase 1)
	_, err = db.Exec(`
		INSERT INTO epics (id, key, title, file_path)
		VALUES (3, 'E04', 'Core CLI', NULL)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test epic 3: %v", err)
	}

	// Feature 1: Has file_path with epic slug (epic 2 will get slug from this - Phase 2)
	_, err = db.Exec(`
		INSERT INTO features (id, epic_id, key, title, file_path)
		VALUES (1, 2, 'E06-F04', 'Incremental Sync Engine', 'docs/plan/E06-intelligent-scanning/E06-F04-incremental-sync-engine/prd.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test feature 1: %v", err)
	}

	// Feature 2: NO file_path (will be extracted from task path - Phase 1)
	_, err = db.Exec(`
		INSERT INTO features (id, epic_id, key, title, file_path)
		VALUES (2, 3, 'E04-F01', 'Database Schema', NULL)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test feature 2: %v", err)
	}

	// Task 1: Has slug in filename (will extract task slug - Phase 3)
	// Path contains correct epic and feature slugs for extraction in Phase 1
	_, err = db.Exec(`
		INSERT INTO tasks (id, feature_id, key, title, file_path)
		VALUES (1, 1, 'T-E06-F04-001', 'Some Task Description', 'docs/plan/E06-intelligent-scanning/E06-F04-incremental-sync-engine/tasks/T-E06-F04-001-some-task-description.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test task 1: %v", err)
	}

	// Task 2: NO slug in filename, but path contains epic and feature slugs (Phase 1)
	_, err = db.Exec(`
		INSERT INTO tasks (id, feature_id, key, title, file_path)
		VALUES (2, 2, 'T-E04-F01-002', 'Another Task', '/home/user/docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/tasks/T-E04-F01-002.md')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test task 2: %v", err)
	}

	// Insert entities with NULL file_path (should remain NULL slug)
	_, err = db.Exec(`
		INSERT INTO epics (id, key, title, file_path) VALUES (4, 'E08', 'Epic Without Path', NULL);
		INSERT INTO features (id, epic_id, key, title, file_path) VALUES (3, 4, 'E08-F01', 'Feature Without Path', NULL);
		INSERT INTO tasks (id, feature_id, key, title, file_path) VALUES (3, 3, 'T-E08-F01-001', 'Task Without Path', NULL);
	`)
	if err != nil {
		t.Fatalf("Failed to insert entities with NULL file_path: %v", err)
	}

	// Verify all slugs are NULL before migration
	var epicSlug sql.NullString
	err = db.QueryRow("SELECT slug FROM epics WHERE id = 1").Scan(&epicSlug)
	if err != nil {
		t.Fatalf("Failed to query epic slug: %v", err)
	}
	if epicSlug.Valid {
		t.Errorf("Epic slug should be NULL before migration, got: %s", epicSlug.String)
	}

	// Run the backfill migration
	stats, err := BackfillSlugsFromFilePaths(db, false)
	if err != nil {
		t.Fatalf("BackfillSlugsFromFilePaths failed: %v", err)
	}

	// Verify stats are correct
	if stats.EpicsTotal != 4 {
		t.Errorf("Expected 4 total epics, got %d", stats.EpicsTotal)
	}
	if stats.FeaturesTotal != 3 {
		t.Errorf("Expected 3 total features, got %d", stats.FeaturesTotal)
	}
	if stats.TasksTotal != 3 {
		t.Errorf("Expected 3 total tasks, got %d", stats.TasksTotal)
	}

	// Verify Epic 1 slug (extracted from own epic.md path - Phase 3)
	err = db.QueryRow("SELECT slug FROM epics WHERE id = 1").Scan(&epicSlug)
	if err != nil {
		t.Fatalf("Failed to query epic 1 slug after migration: %v", err)
	}
	if !epicSlug.Valid || epicSlug.String != "task-mgmt-cli-capabilities" {
		t.Errorf("Expected epic 1 slug 'task-mgmt-cli-capabilities', got: %v", epicSlug)
	}

	// Verify Epic 2 slug (extracted from feature path - Phase 2)
	err = db.QueryRow("SELECT slug FROM epics WHERE id = 2").Scan(&epicSlug)
	if err != nil {
		t.Fatalf("Failed to query epic 2 slug after migration: %v", err)
	}
	if !epicSlug.Valid || epicSlug.String != "intelligent-scanning" {
		t.Errorf("Expected epic 2 slug 'intelligent-scanning', got Valid=%v String='%s'", epicSlug.Valid, epicSlug.String)
	}

	// Verify Epic 3 slug (extracted from task path - Phase 1)
	err = db.QueryRow("SELECT slug FROM epics WHERE id = 3").Scan(&epicSlug)
	if err != nil {
		t.Fatalf("Failed to query epic 3 slug after migration: %v", err)
	}
	if !epicSlug.Valid || epicSlug.String != "task-mgmt-cli-core" {
		t.Errorf("Expected epic 3 slug 'task-mgmt-cli-core', got: %v", epicSlug)
	}

	// Verify Feature 1 slug (extracted from own path - Phase 3)
	var featureSlug sql.NullString
	err = db.QueryRow("SELECT slug FROM features WHERE id = 1").Scan(&featureSlug)
	if err != nil {
		t.Fatalf("Failed to query feature 1 slug after migration: %v", err)
	}
	if !featureSlug.Valid || featureSlug.String != "incremental-sync-engine" {
		t.Errorf("Expected feature 1 slug 'incremental-sync-engine', got Valid=%v String='%s'", featureSlug.Valid, featureSlug.String)
	}

	// Verify Feature 2 slug (extracted from task path - Phase 1)
	err = db.QueryRow("SELECT slug FROM features WHERE id = 2").Scan(&featureSlug)
	if err != nil {
		t.Fatalf("Failed to query feature 2 slug after migration: %v", err)
	}
	if !featureSlug.Valid || featureSlug.String != "database-schema" {
		t.Errorf("Expected feature 2 slug 'database-schema', got: %v", featureSlug)
	}

	// Verify Task 1 slug (extracted from own filename - Phase 3)
	var taskSlug sql.NullString
	err = db.QueryRow("SELECT slug FROM tasks WHERE id = 1").Scan(&taskSlug)
	if err != nil {
		t.Fatalf("Failed to query task 1 slug after migration: %v", err)
	}
	if !taskSlug.Valid || taskSlug.String != "some-task-description" {
		t.Errorf("Expected task 1 slug 'some-task-description', got: %v", taskSlug)
	}

	// Verify Task 2 has NO slug (task key only, no slug in filename)
	err = db.QueryRow("SELECT slug FROM tasks WHERE id = 2").Scan(&taskSlug)
	if err != nil {
		t.Fatalf("Failed to query task 2 slug after migration: %v", err)
	}
	// Task 2 should have NO slug because filename is just T-E04-F01-002.md
	if taskSlug.Valid {
		t.Errorf("Expected task 2 to have NULL slug (no slug in filename), got: %s", taskSlug.String)
	}

	// Verify entities with NULL file_path have NULL slug
	err = db.QueryRow("SELECT slug FROM epics WHERE id = 4").Scan(&epicSlug)
	if err != nil {
		t.Fatalf("Failed to query epic 4 with NULL file_path: %v", err)
	}
	if epicSlug.Valid {
		t.Errorf("Epic 4 with NULL file_path should have NULL slug, got: %s", epicSlug.String)
	}

	err = db.QueryRow("SELECT slug FROM features WHERE id = 3").Scan(&featureSlug)
	if err != nil {
		t.Fatalf("Failed to query feature 3 with NULL file_path: %v", err)
	}
	if featureSlug.Valid {
		t.Errorf("Feature 3 with NULL file_path should have NULL slug, got: %s", featureSlug.String)
	}

	err = db.QueryRow("SELECT slug FROM tasks WHERE id = 3").Scan(&taskSlug)
	if err != nil {
		t.Fatalf("Failed to query task 3 with NULL file_path: %v", err)
	}
	if taskSlug.Valid {
		t.Errorf("Task 3 with NULL file_path should have NULL slug, got: %s", taskSlug.String)
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
			name:     "extract epic slug from feature path",
			filePath: "docs/plan/E10-advanced-task-intelligence-context-management/E10-F01-task-activity-notes-system/feature.md",
			expected: "advanced-task-intelligence-context-management",
		},
		{
			name:     "extract epic slug from task path (absolute)",
			filePath: "/home/user/docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/tasks/T-E04-F01-001.md",
			expected: "task-mgmt-cli-core",
		},
		{
			name:     "extract epic slug from task path (relative)",
			filePath: "docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/tasks/T-E04-F01-002.md",
			expected: "task-mgmt-cli-core",
		},
		{
			name:     "epic folder without slug - should not extract feature key",
			filePath: "docs/plan/E08/E08-F01/tasks/T-E08-F01-001.md",
			expected: "",
		},
		{
			name:     "epic folder without slug - feature path",
			filePath: "docs/plan/E05/E05-F01-migrations/prd.md",
			expected: "",
		},
		{
			name:     "epic folder without slug - should not extract F05",
			filePath: "docs/plan/E07/E07-F05-slug-architecture-improvement/tasks/T-E07-F05-001.md",
			expected: "",
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
			name:     "extract feature slug from task path (absolute)",
			filePath: "/home/user/docs/plan/E04-epic/E04-F01-database-schema/tasks/T-E04-F01-001.md",
			expected: "database-schema",
		},
		{
			name:     "extract feature slug from task path (relative)",
			filePath: "docs/plan/E04-epic/E04-F01-database-schema/tasks/T-E04-F01-002.md",
			expected: "database-schema",
		},
		{
			name:     "extract feature slug from task path with multi-part epic",
			filePath: "docs/plan/E10-advanced-task-intelligence-context-management/E10-F05-work-sessions-resume-context/tasks/T-E10-F05-001.md",
			expected: "work-sessions-resume-context",
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
