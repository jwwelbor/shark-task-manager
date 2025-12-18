package sync

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	_ "github.com/mattn/go-sqlite3"
)

// TestDryRunMode_NoChangesCommitted verifies that dry-run mode executes
// the full workflow but doesn't persist any database changes
func TestDryRunMode_NoChangesCommitted(t *testing.T) {
	// Create temporary test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize test database
	db := initTestDB(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	epic := createTestEpic(t, db, "E04", "Test Epic")
	_ = createTestFeature(t, db, epic.ID, "E04-F07", "Test Feature")

	// Create test task file
	taskDir := filepath.Join(tempDir, "docs", "plan", "E04-epic", "E04-F07-feature", "tasks")
	err := os.MkdirAll(taskDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create task directory: %v", err)
	}

	taskFilePath := filepath.Join(taskDir, "T-E04-F07-001.md")
	taskContent := `---
task_key: T-E04-F07-001
title: Test Task
status: todo
---

# Task: Test Task

This is a test task for dry-run mode.
`
	err = os.WriteFile(taskFilePath, []byte(taskContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create task file: %v", err)
	}

	// Get initial task count
	initialCount := getTaskCount(t, db)
	if initialCount != 0 {
		t.Fatalf("Expected 0 initial tasks, got %d", initialCount)
	}

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Run sync in dry-run mode
	opts := SyncOptions{
		FolderPath: filepath.Join(tempDir, "docs", "plan"),
		DryRun:     true,
		Strategy:   ConflictStrategyFileWins,
	}

	ctx := context.Background()
	report, err := engine.Sync(ctx, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify report shows what WOULD happen
	if report.FilesScanned != 1 {
		t.Errorf("Expected 1 file scanned, got %d", report.FilesScanned)
	}
	if report.TasksImported != 1 {
		t.Errorf("Expected 1 task imported in report, got %d", report.TasksImported)
	}
	if !report.DryRun {
		t.Error("Expected report.DryRun to be true")
	}

	// Verify NO changes were actually committed to database
	finalCount := getTaskCount(t, db)
	if finalCount != 0 {
		t.Errorf("Expected 0 tasks in database after dry-run, got %d", finalCount)
	}
}

// TestDryRunMode_ThenRealRun verifies that running dry-run followed by
// a real run produces the expected results
func TestDryRunMode_ThenRealRun(t *testing.T) {
	// Create temporary test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize test database
	db := initTestDB(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	epic := createTestEpic(t, db, "E04", "Test Epic")
	_ = createTestFeature(t, db, epic.ID, "E04-F07", "Test Feature")

	// Create test task file
	taskDir := filepath.Join(tempDir, "docs", "plan", "E04-epic", "E04-F07-feature", "tasks")
	err := os.MkdirAll(taskDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create task directory: %v", err)
	}

	taskFilePath := filepath.Join(taskDir, "T-E04-F07-001.md")
	taskContent := `---
task_key: T-E04-F07-001
title: Test Task
status: todo
---

# Task: Test Task

This is a test task.
`
	err = os.WriteFile(taskFilePath, []byte(taskContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create task file: %v", err)
	}

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	folderPath := filepath.Join(tempDir, "docs", "plan")

	// Run dry-run first
	dryRunOpts := SyncOptions{
		FolderPath: folderPath,
		DryRun:     true,
		Strategy:   ConflictStrategyFileWins,
	}

	ctx := context.Background()
	dryRunReport, err := engine.Sync(ctx, dryRunOpts)
	if err != nil {
		t.Fatalf("Dry-run sync failed: %v", err)
	}

	// Verify dry-run report
	if dryRunReport.TasksImported != 1 {
		t.Errorf("Dry-run: Expected 1 task imported, got %d", dryRunReport.TasksImported)
	}
	if !dryRunReport.DryRun {
		t.Error("Expected dryRunReport.DryRun to be true")
	}

	// Verify no changes committed
	if getTaskCount(t, db) != 0 {
		t.Error("Expected 0 tasks after dry-run")
	}

	// Now run real sync
	realOpts := SyncOptions{
		FolderPath: folderPath,
		DryRun:     false,
		Strategy:   ConflictStrategyFileWins,
	}

	realReport, err := engine.Sync(ctx, realOpts)
	if err != nil {
		t.Fatalf("Real sync failed: %v", err)
	}

	// Verify real sync report
	if realReport.TasksImported != 1 {
		t.Errorf("Real sync: Expected 1 task imported, got %d", realReport.TasksImported)
	}
	if realReport.DryRun {
		t.Error("Expected realReport.DryRun to be false")
	}

	// Verify changes were committed
	if getTaskCount(t, db) != 1 {
		t.Errorf("Expected 1 task after real sync, got %d", getTaskCount(t, db))
	}

	// Verify the reports are identical except for DryRun flag
	if dryRunReport.FilesScanned != realReport.FilesScanned {
		t.Errorf("FilesScanned mismatch: dry-run=%d, real=%d",
			dryRunReport.FilesScanned, realReport.FilesScanned)
	}
	if dryRunReport.TasksImported != realReport.TasksImported {
		t.Errorf("TasksImported mismatch: dry-run=%d, real=%d",
			dryRunReport.TasksImported, realReport.TasksImported)
	}
}

// TestDryRunMode_WithConflicts verifies that dry-run mode detects
// conflicts without resolving them in the database
func TestDryRunMode_WithConflicts(t *testing.T) {
	// Create temporary test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize test database
	db := initTestDB(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	epic := createTestEpic(t, db, "E04", "Test Epic")
	feature := createTestFeature(t, db, epic.ID, "E04-F07", "Test Feature")

	// Create task file with different title
	taskDir := filepath.Join(tempDir, "docs", "plan", "E04-epic", "E04-F07-feature", "tasks")
	err := os.MkdirAll(taskDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create task directory: %v", err)
	}

	// Create existing task in database (with file_path set to avoid extra conflict)
	taskFilePath := filepath.Join(taskDir, "T-E04-F07-001.md")
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E04-F07-001",
		Title:     "Original Title",
		Status:    models.TaskStatusTodo,
		Priority:  5,
		FilePath:  &taskFilePath,
	}
	repoDb := repository.NewDB(db)
	taskRepo := repository.NewTaskRepository(repoDb)
	err = taskRepo.Create(context.Background(), task)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	originalTitle := task.Title

	taskFilePath = filepath.Join(taskDir, "T-E04-F07-001.md")
	taskContent := `---
task_key: T-E04-F07-001
title: Modified Title
status: todo
---

# Task: Modified Title

This task has a different title.
`
	err = os.WriteFile(taskFilePath, []byte(taskContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create task file: %v", err)
	}

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Run sync in dry-run mode with file-wins strategy
	opts := SyncOptions{
		FolderPath: filepath.Join(tempDir, "docs", "plan"),
		DryRun:     true,
		Strategy:   ConflictStrategyFileWins,
	}

	ctx := context.Background()
	report, err := engine.Sync(ctx, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify conflict was detected
	if len(report.Conflicts) != 1 {
		t.Logf("Conflicts detected:")
		for i, c := range report.Conflicts {
			t.Logf("  [%d] Field: %s, DB: %q, File: %q", i, c.Field, c.DatabaseValue, c.FileValue)
		}
		t.Fatalf("Expected 1 conflict, got %d", len(report.Conflicts))
	}

	conflict := report.Conflicts[0]
	if conflict.Field != "title" {
		t.Errorf("Expected title conflict, got %s", conflict.Field)
	}
	if conflict.FileValue != "Modified Title" {
		t.Errorf("Expected file value 'Modified Title', got %s", conflict.FileValue)
	}
	if conflict.DatabaseValue != "Original Title" {
		t.Errorf("Expected database value 'Original Title', got %s", conflict.DatabaseValue)
	}

	// Verify report shows update would happen
	if report.TasksUpdated != 1 {
		t.Errorf("Expected 1 task updated in report, got %d", report.TasksUpdated)
	}
	if report.ConflictsResolved != 1 {
		t.Errorf("Expected 1 conflict resolved in report, got %d", report.ConflictsResolved)
	}

	// Verify database still has original title (no changes committed)
	updatedTask, err := taskRepo.GetByKey(ctx, "T-E04-F07-001")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	if updatedTask.Title != originalTitle {
		t.Errorf("Expected title to remain '%s', got '%s'", originalTitle, updatedTask.Title)
	}
}

// Helper functions

func initTestDB(t *testing.T, dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create schema
	schema := `
	CREATE TABLE epics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		status TEXT NOT NULL,
		priority TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE features (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		epic_id INTEGER NOT NULL,
		key TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		progress_pct INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (epic_id) REFERENCES epics(id)
	);

	CREATE TABLE tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		feature_id INTEGER NOT NULL,
		key TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		agent_type TEXT,
		priority INTEGER DEFAULT 5,
		depends_on TEXT,
		assigned_agent TEXT,
		file_path TEXT,
		blocked_reason TEXT,
		blocked_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (feature_id) REFERENCES features(id)
	);

	CREATE TABLE task_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL,
		status_from TEXT NOT NULL,
		status_to TEXT NOT NULL,
		changed_by TEXT NOT NULL,
		change_description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id)
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func createTestEpic(t *testing.T, db *sql.DB, key, title string) *models.Epic {
	epic := &models.Epic{
		Key:      key,
		Title:    title,
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}

	result, err := db.Exec(
		"INSERT INTO epics (key, title, status, priority) VALUES (?, ?, ?, ?)",
		epic.Key, epic.Title, epic.Status, epic.Priority,
	)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	id, _ := result.LastInsertId()
	epic.ID = id
	return epic
}

func createTestFeature(t *testing.T, db *sql.DB, epicID int64, key, title string) *models.Feature {
	feature := &models.Feature{
		EpicID: epicID,
		Key:    key,
		Title:  title,
		Status: models.FeatureStatusActive,
	}

	result, err := db.Exec(
		"INSERT INTO features (epic_id, key, title, status) VALUES (?, ?, ?, ?)",
		feature.EpicID, feature.Key, feature.Title, feature.Status,
	)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	id, _ := result.LastInsertId()
	feature.ID = id
	return feature
}

func getTaskCount(t *testing.T, db *sql.DB) int {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count tasks: %v", err)
	}
	return count
}
