package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	_ "github.com/mattn/go-sqlite3"
)

// TestSyncEngine_NewSyncEngine tests creating a new sync engine
func TestSyncEngine_NewSyncEngine(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	database.Close()

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Verify engine was created
	if engine.db == nil {
		t.Error("Database connection is nil")
	}
	if engine.taskRepo == nil {
		t.Error("TaskRepository is nil")
	}
	if engine.epicRepo == nil {
		t.Error("EpicRepository is nil")
	}
	if engine.featureRepo == nil {
		t.Error("FeatureRepository is nil")
	}
	if engine.scanner == nil {
		t.Error("FileScanner is nil")
	}
	if engine.detector == nil {
		t.Error("ConflictDetector is nil")
	}
	if engine.resolver == nil {
		t.Error("ConflictResolver is nil")
	}
}

// TestSyncEngine_Sync_EmptyDirectory tests syncing an empty directory
func TestSyncEngine_Sync_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Setup database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	database.Close()

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Create empty directory for scanning
	scanDir := filepath.Join(tempDir, "empty")
	if err := os.MkdirAll(scanDir, 0755); err != nil {
		t.Fatalf("Failed to create scan directory: %v", err)
	}

	// Run sync
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    scanDir,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: false,
		Cleanup:       false,
	}

	ctx := context.Background()
	report, err := engine.Sync(ctx, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify report
	if report.FilesScanned != 0 {
		t.Errorf("Expected 0 files scanned, got %d", report.FilesScanned)
	}
	if report.TasksImported != 0 {
		t.Errorf("Expected 0 tasks imported, got %d", report.TasksImported)
	}
	if report.TasksUpdated != 0 {
		t.Errorf("Expected 0 tasks updated, got %d", report.TasksUpdated)
	}
}

// TestSyncEngine_Sync_NewTasksImport tests importing new tasks
func TestSyncEngine_Sync_NewTasksImport(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Setup database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create test epic and feature
	ctx := context.Background()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	desc := "Test epic description"
	epic := &models.Epic{
		Key:         "E04",
		Title:       "Test Epic",
		Description: &desc,
		Status:      models.EpicStatusActive,
		Priority:    models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	featureDesc := "Test feature description"
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         "E04-F07",
		Title:       "Test Feature",
		Description: &featureDesc,
		Status:      models.FeatureStatusActive,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create test task files
	scanDir := filepath.Join(tempDir, "docs", "plan", "E04-test-epic", "E04-F07-test-feature")
	if err := os.MkdirAll(scanDir, 0755); err != nil {
		t.Fatalf("Failed to create scan directory: %v", err)
	}

	taskFile1 := filepath.Join(scanDir, "T-E04-F07-001.md")
	taskContent1 := `---
task_key: T-E04-F07-001
title: First test task
description: Test description 1
---

# Task Content
Test content here.
`
	if err := os.WriteFile(taskFile1, []byte(taskContent1), 0644); err != nil {
		t.Fatalf("Failed to write task file: %v", err)
	}

	taskFile2 := filepath.Join(scanDir, "T-E04-F07-002.md")
	taskContent2 := `---
task_key: T-E04-F07-002
title: Second test task
---

# Task Content
More test content.
`
	if err := os.WriteFile(taskFile2, []byte(taskContent2), 0644); err != nil {
		t.Fatalf("Failed to write task file: %v", err)
	}

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Run sync
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    scanDir,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: false,
		Cleanup:       false,
	}

	report, err := engine.Sync(ctx, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify report
	if report.FilesScanned != 2 {
		t.Errorf("Expected 2 files scanned, got %d", report.FilesScanned)
	}
	if report.TasksImported != 2 {
		t.Errorf("Expected 2 tasks imported, got %d", report.TasksImported)
	}
	if report.TasksUpdated != 0 {
		t.Errorf("Expected 0 tasks updated, got %d", report.TasksUpdated)
	}

	// Verify tasks in database
	taskRepo := repository.NewTaskRepository(repoDb)
	task1, err := taskRepo.GetByKey(ctx, "T-E04-F07-001")
	if err != nil {
		t.Fatalf("Failed to get task 1: %v", err)
	}
	if task1.Title != "First test task" {
		t.Errorf("Expected title 'First test task', got '%s'", task1.Title)
	}
	if task1.Status != models.TaskStatusTodo {
		t.Errorf("Expected status 'todo', got '%s'", task1.Status)
	}

	task2, err := taskRepo.GetByKey(ctx, "T-E04-F07-002")
	if err != nil {
		t.Fatalf("Failed to get task 2: %v", err)
	}
	if task2.Title != "Second test task" {
		t.Errorf("Expected title 'Second test task', got '%s'", task2.Title)
	}
}

// TestSyncEngine_Sync_UpdateExistingTasks tests updating existing tasks
func TestSyncEngine_Sync_UpdateExistingTasks(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Setup database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create test epic, feature, and task
	ctx := context.Background()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	epic := &models.Epic{
		Key:      "E04",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E04-F07",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	desc := "Original description"
	task := &models.Task{
		FeatureID:   feature.ID,
		Key:         "T-E04-F07-001",
		Title:       "Original title",
		Description: &desc,
		Status:      models.TaskStatusInProgress,
		Priority:    5,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Create task file with updated metadata
	scanDir := filepath.Join(tempDir, "docs", "plan", "E04-test-epic", "E04-F07-test-feature")
	if err := os.MkdirAll(scanDir, 0755); err != nil {
		t.Fatalf("Failed to create scan directory: %v", err)
	}

	taskFile := filepath.Join(scanDir, "T-E04-F07-001.md")
	taskContent := `---
task_key: T-E04-F07-001
title: Updated title
description: Updated description
---

# Task Content
Updated content.
`
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("Failed to write task file: %v", err)
	}

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Run sync with file-wins strategy
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    scanDir,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: false,
		Cleanup:       false,
	}

	report, err := engine.Sync(ctx, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify report
	if report.FilesScanned != 1 {
		t.Errorf("Expected 1 file scanned, got %d", report.FilesScanned)
	}
	if report.TasksImported != 0 {
		t.Errorf("Expected 0 tasks imported, got %d", report.TasksImported)
	}
	if report.TasksUpdated != 1 {
		t.Errorf("Expected 1 task updated, got %d", report.TasksUpdated)
	}
	if report.ConflictsResolved != 3 {
		t.Errorf("Expected 3 conflicts resolved (title, description, file_path), got %d", report.ConflictsResolved)
	}

	// Verify task was updated
	updatedTask, err := taskRepo.GetByKey(ctx, "T-E04-F07-001")
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}
	if updatedTask.Title != "Updated title" {
		t.Errorf("Expected title 'Updated title', got '%s'", updatedTask.Title)
	}
	if updatedTask.Description == nil || *updatedTask.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%v'", updatedTask.Description)
	}
	// Status should NOT be updated (database-only field)
	if updatedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status 'in_progress' (preserved), got '%s'", updatedTask.Status)
	}
}

// TestSyncEngine_Sync_DryRun tests dry-run mode
func TestSyncEngine_Sync_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Setup database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create test epic and feature
	ctx := context.Background()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	epic := &models.Epic{
		Key:      "E04",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E04-F07",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create task file
	scanDir := filepath.Join(tempDir, "docs", "plan", "E04-test-epic", "E04-F07-test-feature")
	if err := os.MkdirAll(scanDir, 0755); err != nil {
		t.Fatalf("Failed to create scan directory: %v", err)
	}

	taskFile := filepath.Join(scanDir, "T-E04-F07-001.md")
	taskContent := `---
task_key: T-E04-F07-001
title: Test task
---

# Content
`
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("Failed to write task file: %v", err)
	}

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Run sync in dry-run mode
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    scanDir,
		DryRun:        true,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: false,
		Cleanup:       false,
	}

	report, err := engine.Sync(ctx, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify report shows what would happen
	if report.FilesScanned != 1 {
		t.Errorf("Expected 1 file scanned, got %d", report.FilesScanned)
	}

	// Verify task was NOT created in database
	taskRepo := repository.NewTaskRepository(repoDb)
	_, err = taskRepo.GetByKey(ctx, "T-E04-F07-001")
	if err == nil {
		t.Error("Expected task not to exist (dry-run), but it was found")
	}
}

// TestSyncEngine_Sync_ConflictStrategies tests different conflict resolution strategies
func TestSyncEngine_Sync_ConflictStrategies(t *testing.T) {
	strategies := []struct {
		name          string
		strategy      ConflictStrategy
		expectedTitle string
	}{
		{"file-wins", ConflictStrategyFileWins, "File title"},
		{"database-wins", ConflictStrategyDatabaseWins, "DB title"},
	}

	for _, tc := range strategies {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "test.db")

			// Setup database
			database, err := db.InitDB(dbPath)
			if err != nil {
				t.Fatalf("Failed to initialize database: %v", err)
			}
			defer database.Close()

			ctx := context.Background()
			repoDb := repository.NewDB(database)
			epicRepo := repository.NewEpicRepository(repoDb)
			featureRepo := repository.NewFeatureRepository(repoDb)
			taskRepo := repository.NewTaskRepository(repoDb)

			epic := &models.Epic{
		Key:      "E04",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
			if err := epicRepo.Create(ctx, epic); err != nil {
				t.Fatalf("Failed to create epic: %v", err)
			}

			feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E04-F07",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
			if err := featureRepo.Create(ctx, feature); err != nil {
				t.Fatalf("Failed to create feature: %v", err)
			}

			task := &models.Task{
				FeatureID: feature.ID,
				Key:       "T-E04-F07-001",
				Title:     "DB title",
				Status:    models.TaskStatusTodo,
				Priority:  5,
			}
			if err := taskRepo.Create(ctx, task); err != nil {
				t.Fatalf("Failed to create task: %v", err)
			}

			// Create task file with different title
			scanDir := filepath.Join(tempDir, "docs", "plan", "E04-test-epic", "E04-F07-test-feature")
			if err := os.MkdirAll(scanDir, 0755); err != nil {
				t.Fatalf("Failed to create scan directory: %v", err)
			}

			taskFile := filepath.Join(scanDir, "T-E04-F07-001.md")
			taskContent := `---
task_key: T-E04-F07-001
title: File title
---
`
			if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
				t.Fatalf("Failed to write task file: %v", err)
			}

			// Wait a bit to ensure file has newer timestamp
			time.Sleep(10 * time.Millisecond)

			// Create sync engine
			engine, err := NewSyncEngine(dbPath)
			if err != nil {
				t.Fatalf("Failed to create sync engine: %v", err)
			}
			defer engine.Close()

			// Run sync with specified strategy
			opts := SyncOptions{
				DBPath:        dbPath,
				FolderPath:    scanDir,
				DryRun:        false,
				Strategy:      tc.strategy,
				CreateMissing: false,
				Cleanup:       false,
			}

			_, err = engine.Sync(ctx, opts)
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}

			// Verify task title matches expected
			updatedTask, err := taskRepo.GetByKey(ctx, "T-E04-F07-001")
			if err != nil {
				t.Fatalf("Failed to get updated task: %v", err)
			}
			if updatedTask.Title != tc.expectedTitle {
				t.Errorf("Expected title '%s', got '%s'", tc.expectedTitle, updatedTask.Title)
			}
		})
	}
}

// TestSyncEngine_Sync_InvalidYAML tests handling of invalid YAML
func TestSyncEngine_Sync_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Setup database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create task file with invalid YAML
	scanDir := filepath.Join(tempDir, "docs", "plan", "E04-test-epic", "E04-F07-test-feature")
	if err := os.MkdirAll(scanDir, 0755); err != nil {
		t.Fatalf("Failed to create scan directory: %v", err)
	}

	taskFile := filepath.Join(scanDir, "T-E04-F07-001.md")
	taskContent := `---
invalid yaml: [unclosed bracket
title: Test
---
`
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("Failed to write task file: %v", err)
	}

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Run sync
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    scanDir,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: false,
		Cleanup:       false,
	}

	ctx := context.Background()
	report, err := engine.Sync(ctx, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify report contains warning
	if len(report.Warnings) == 0 {
		t.Error("Expected warnings for invalid YAML")
	}
	if report.TasksImported != 0 {
		t.Errorf("Expected 0 tasks imported (invalid YAML), got %d", report.TasksImported)
	}
}

// TestSyncEngine_Sync_TransactionRollback tests transaction rollback on error
func TestSyncEngine_Sync_TransactionRollback(t *testing.T) {
	// This test would require injecting an error during sync
	// For now, we'll test that valid operations commit properly
	t.Skip("Transaction rollback test requires error injection mechanism")
}
