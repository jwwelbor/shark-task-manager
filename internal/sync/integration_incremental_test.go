package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIncrementalSync_FirstSync verifies first sync performs full scan and sets timestamp
func TestIncrementalSync_FirstSync(t *testing.T) {
	// Arrange: Create test environment
	testDir := t.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")

	// Create test directory structure
	require.NoError(t, os.MkdirAll(docsPath, 0755))

	// Create .sharkconfig.json (no last_sync_time)
	configPath := filepath.Join(testDir, ".sharkconfig.json")
	configManager := config.NewManager(configPath)
	require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0644))
	_, err := configManager.Load()
	require.NoError(t, err)

	// Initialize database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(t, db)

	// Create task files
	taskFile1 := filepath.Join(docsPath, "T-E01-F01-001.md")
	taskFile2 := filepath.Join(docsPath, "T-E01-F01-002.md")
	createTestTaskFile(t, taskFile1, "T-E01-F01-001", "First Task")
	createTestTaskFile(t, taskFile2, "T-E01-F01-002", "Second Task")

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	require.NoError(t, err)
	defer engine.Close()

	startTime := time.Now()

	// Act: Run first sync (no LastSyncTime = full scan)
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		Cleanup:       false,
		LastSyncTime:  nil, // First sync
	}

	report, err := engine.Sync(context.Background(), opts)
	require.NoError(t, err)

	// Assert: Full scan performed
	assert.Equal(t, 2, report.FilesScanned, "Should scan all files")
	assert.Equal(t, 2, report.FilesFiltered, "Should filter all files (full scan)")
	assert.Equal(t, 0, report.FilesSkipped, "Should skip no files on first sync")
	assert.Equal(t, 2, report.TasksImported, "Should import both tasks")
	assert.Equal(t, 0, report.TasksUpdated, "Should update no tasks")

	// Update last_sync_time in config
	err = configManager.UpdateLastSyncTime(startTime)
	require.NoError(t, err)

	// Verify last_sync_time was written
	reloadedConfig, err := configManager.Load()
	require.NoError(t, err)
	require.NotNil(t, reloadedConfig.LastSyncTime)
	assert.True(t, reloadedConfig.LastSyncTime.After(startTime.Add(-1*time.Second)))
}

// TestIncrementalSync_NoChanges verifies sync with no changes completes in <1s
func TestIncrementalSync_NoChanges(t *testing.T) {
	// Arrange: Create test environment with existing sync
	testDir := t.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")

	require.NoError(t, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(t, db)

	// Create and import task files
	taskFile1 := filepath.Join(docsPath, "T-E01-F01-001.md")
	createTestTaskFile(t, taskFile1, "T-E01-F01-001", "First Task")

	// Create sync engine and run initial sync
	engine, err := NewSyncEngine(dbPath)
	require.NoError(t, err)
	defer engine.Close()

	// Initial sync
	initialOpts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  nil,
	}
	_, err = engine.Sync(context.Background(), initialOpts)
	require.NoError(t, err)

	// Set last sync time to now
	lastSyncTime := time.Now()

	// Act: Run incremental sync with no changes
	startTime := time.Now()
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  &lastSyncTime,
	}

	report, err := engine.Sync(context.Background(), opts)
	require.NoError(t, err)
	duration := time.Since(startTime)

	// Assert: No files processed, completes quickly
	assert.Equal(t, 1, report.FilesScanned, "Should scan all files")
	assert.Equal(t, 0, report.FilesFiltered, "Should filter no files (no changes)")
	assert.Equal(t, 1, report.FilesSkipped, "Should skip unchanged file")
	assert.Equal(t, 0, report.TasksImported, "Should import no tasks")
	assert.Equal(t, 0, report.TasksUpdated, "Should update no tasks")
	assert.Less(t, duration.Seconds(), 1.0, "Should complete in <1 second")
}

// TestIncrementalSync_FewFilesChanged verifies sync with 5 files changed completes in <2s
func TestIncrementalSync_FewFilesChanged(t *testing.T) {
	// Arrange: Create test environment
	testDir := t.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")

	require.NoError(t, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(t, db)

	// Create 10 task files
	var taskFiles []string
	for i := 1; i <= 10; i++ {
		taskKey := fmt.Sprintf("T-E01-F01-%03d", i)
		taskFile := filepath.Join(docsPath, taskKey+".md")
		createTestTaskFile(t, taskFile, taskKey, fmt.Sprintf("Task %d", i))
		taskFiles = append(taskFiles, taskFile)
	}

	// Create sync engine and run initial sync
	engine, err := NewSyncEngine(dbPath)
	require.NoError(t, err)
	defer engine.Close()

	// Initial sync
	initialOpts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  nil,
	}
	_, err = engine.Sync(context.Background(), initialOpts)
	require.NoError(t, err)

	// Set last sync time
	lastSyncTime := time.Now()
	time.Sleep(100 * time.Millisecond) // Ensure file mtime is after last sync

	// Modify 5 files
	for i := 0; i < 5; i++ {
		taskKey := fmt.Sprintf("T-E01-F01-%03d", i+1)
		updateTestTaskFile(t, taskFiles[i], taskKey, fmt.Sprintf("Updated Task %d", i+1))
	}

	// Act: Run incremental sync
	startTime := time.Now()
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  &lastSyncTime,
	}

	report, err := engine.Sync(context.Background(), opts)
	require.NoError(t, err)
	duration := time.Since(startTime)

	// Assert: Only changed files processed
	assert.Equal(t, 10, report.FilesScanned, "Should scan all files")
	assert.Equal(t, 5, report.FilesFiltered, "Should filter 5 changed files")
	assert.Equal(t, 5, report.FilesSkipped, "Should skip 5 unchanged files")
	assert.Equal(t, 0, report.TasksImported, "Should import no new tasks")
	assert.Equal(t, 5, report.TasksUpdated, "Should update 5 tasks")
	assert.Less(t, duration.Seconds(), 2.0, "Should complete in <2 seconds")
}

// TestIncrementalSync_ConflictResolution verifies conflict detection with incremental sync
func TestIncrementalSync_ConflictResolution(t *testing.T) {
	// Arrange: Create test environment
	testDir := t.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")

	require.NoError(t, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(t, db)

	// Create task file
	taskFile := filepath.Join(docsPath, "T-E01-F01-001.md")
	createTestTaskFile(t, taskFile, "T-E01-F01-001", "Original Title")

	// Create sync engine and run initial sync
	engine, err := NewSyncEngine(dbPath)
	require.NoError(t, err)
	defer engine.Close()

	// Initial sync
	initialOpts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  nil,
	}
	_, err = engine.Sync(context.Background(), initialOpts)
	require.NoError(t, err)

	lastSyncTime := time.Now()
	time.Sleep(100 * time.Millisecond)

	// Modify database directly (simulate concurrent DB update)
	taskRepo := repository.NewTaskRepository(repository.NewDB(db))
	tasks, err := taskRepo.GetByKeys(context.Background(), []string{"T-E01-F01-001"})
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	dbTask := tasks["T-E01-F01-001"]
	dbTask.Title = "Database Updated Title"
	err = taskRepo.UpdateMetadata(context.Background(), dbTask)
	require.NoError(t, err)

	// Modify file
	updateTestTaskFile(t, taskFile, "T-E01-F01-001", "File Updated Title")

	// Act: Run incremental sync with file-wins strategy
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  &lastSyncTime,
	}

	report, err := engine.Sync(context.Background(), opts)
	require.NoError(t, err)

	// Assert: Conflict detected and resolved
	assert.Equal(t, 1, report.TasksUpdated, "Should update task")
	assert.Greater(t, report.ConflictsResolved, 0, "Should resolve conflicts")
	assert.NotEmpty(t, report.Conflicts, "Should report conflicts")

	// Verify file wins (title from file)
	tasks, err = taskRepo.GetByKeys(context.Background(), []string{"T-E01-F01-001"})
	require.NoError(t, err)
	assert.Equal(t, "File Updated Title", tasks["T-E01-F01-001"].Title)
}

// TestIncrementalSync_ForceFullScan verifies --force-full-scan ignores last_sync_time
func TestIncrementalSync_ForceFullScan(t *testing.T) {
	// Arrange: Create test environment
	testDir := t.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")

	require.NoError(t, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(t, db)

	// Create task files
	for i := 1; i <= 5; i++ {
		taskKey := fmt.Sprintf("T-E01-F01-%03d", i)
		taskFile := filepath.Join(docsPath, taskKey+".md")
		createTestTaskFile(t, taskFile, taskKey, fmt.Sprintf("Task %d", i))
	}

	// Create sync engine and run initial sync
	engine, err := NewSyncEngine(dbPath)
	require.NoError(t, err)
	defer engine.Close()

	// Initial sync
	initialOpts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  nil,
	}
	_, err = engine.Sync(context.Background(), initialOpts)
	require.NoError(t, err)

	// Set last sync time to now (no files modified after this)
	lastSyncTime := time.Now()

	// Act: Run sync with ForceFullScan = true
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  &lastSyncTime,
		ForceFullScan: true, // Force full scan despite LastSyncTime
	}

	report, err := engine.Sync(context.Background(), opts)
	require.NoError(t, err)

	// Assert: Full scan performed (all files filtered)
	assert.Equal(t, 5, report.FilesScanned, "Should scan all files")
	assert.Equal(t, 5, report.FilesFiltered, "Should filter all files (force full scan)")
	assert.Equal(t, 0, report.FilesSkipped, "Should skip no files")
}

// TestIncrementalSync_TransactionRollback verifies transaction rollback behavior
func TestIncrementalSync_TransactionRollback(t *testing.T) {
	// Arrange: Create test environment
	testDir := t.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")

	require.NoError(t, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	// Create test epic (but not feature - will cause error)
	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(repository.NewDB(db))
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create task file (will fail due to missing feature)
	taskFile := filepath.Join(docsPath, "T-E01-F01-001.md")
	createTestTaskFile(t, taskFile, "T-E01-F01-001", "Task Title")

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	require.NoError(t, err)
	defer engine.Close()

	lastSyncTime := time.Now().Add(-1 * time.Hour)

	// Act: Run sync (should fail due to missing feature)
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: false, // Don't auto-create
		LastSyncTime:  &lastSyncTime,
	}

	report, err := engine.Sync(ctx, opts)

	// Assert: Sync fails, transaction rolled back
	assert.Error(t, err, "Should return error")
	assert.Contains(t, err.Error(), "feature", "Error should mention missing feature")

	// Verify no tasks were created (transaction rolled back)
	taskRepo := repository.NewTaskRepository(repository.NewDB(db))
	tasks, err := taskRepo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, tasks, "No tasks should be created after rollback")

	// Report should still be returned with partial stats
	assert.NotNil(t, report, "Report should be returned even on error")
}

// TestIncrementalSync_BackwardCompatibility verifies non-incremental sync still works
func TestIncrementalSync_BackwardCompatibility(t *testing.T) {
	// Arrange: Create test environment
	testDir := t.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")

	require.NoError(t, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(t, db)

	// Create task files
	taskFile := filepath.Join(docsPath, "T-E01-F01-001.md")
	createTestTaskFile(t, taskFile, "T-E01-F01-001", "Task Title")

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	require.NoError(t, err)
	defer engine.Close()

	// Act: Run sync WITHOUT LastSyncTime (traditional E04-F07 behavior)
	opts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  nil, // No incremental sync
		ForceFullScan: false,
	}

	report, err := engine.Sync(context.Background(), opts)
	require.NoError(t, err)

	// Assert: Works exactly as E04-F07 (full scan)
	assert.Equal(t, 1, report.FilesScanned, "Should scan all files")
	assert.Equal(t, 1, report.FilesFiltered, "Should filter all files")
	assert.Equal(t, 0, report.FilesSkipped, "Should skip no files")
	assert.Equal(t, 1, report.TasksImported, "Should import task")
	assert.Equal(t, 0, report.TasksUpdated, "Should update no tasks")
}
