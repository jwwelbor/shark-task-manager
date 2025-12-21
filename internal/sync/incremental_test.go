package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	_ "github.com/mattn/go-sqlite3"
)

// TestFilter_ForceFullScan verifies that --force-full-scan bypasses incremental filtering
func TestFilter_ForceFullScan(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	taskRepo := repository.NewTaskRepository(db)
	filter := NewIncrementalFilter(taskRepo)

	// Create test files
	files := createTestFiles(t, tmpDir, 10)

	// Add files to database so they're not "new"
	ctx := context.Background()
	for i, f := range files {
		task := &models.Task{
			FeatureID: 1,
			Key:       fmt.Sprintf("T-E04-F07-%03d", i+1),
			Title:     "Test Task",
			Status:    models.TaskStatusTodo,
			Priority:  5,
			FilePath:  &f.FilePath,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Set last sync time to now (all files should be filtered out in incremental mode)
	now := time.Now()

	// Test 1: Without force full scan (should filter based on mtime)
	result1, _, err := filter.Filter(context.Background(), files, FilterOptions{
		LastSyncTime:  &now,
		ForceFullScan: false,
	})
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// All files created before 'now', so should be filtered out
	if len(result1) != 0 {
		t.Errorf("Expected 0 files without force scan, got %d", len(result1))
	}

	// Test 2: With force full scan (should include all files)
	result2, stats2, err := filter.Filter(context.Background(), files, FilterOptions{
		LastSyncTime:  &now,
		ForceFullScan: true,
	})
	if err != nil {
		t.Fatalf("Filter with force scan failed: %v", err)
	}

	if len(result2) != 10 {
		t.Errorf("Expected 10 files with force scan, got %d", len(result2))
	}

	if stats2.FilteredFiles != 10 {
		t.Errorf("Expected stats.FilteredFiles=10, got %d", stats2.FilteredFiles)
	}

	if stats2.SkippedFiles != 0 {
		t.Errorf("Expected stats.SkippedFiles=0, got %d", stats2.SkippedFiles)
	}
}

// TestFilter_NilLastSyncTime verifies fallback to full scan when last_sync_time is nil
func TestFilter_NilLastSyncTime(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	taskRepo := repository.NewTaskRepository(db)
	filter := NewIncrementalFilter(taskRepo)

	// Create test files
	files := createTestFiles(t, tmpDir, 5)

	// Test with nil last sync time (should perform full scan)
	result, stats, err := filter.Filter(context.Background(), files, FilterOptions{
		LastSyncTime:  nil,
		ForceFullScan: false,
	})
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	if len(result) != 5 {
		t.Errorf("Expected 5 files with nil lastSyncTime, got %d", len(result))
	}

	if stats.FilteredFiles != 5 {
		t.Errorf("Expected stats.FilteredFiles=5, got %d", stats.FilteredFiles)
	}

	if stats.SkippedFiles != 0 {
		t.Errorf("Expected stats.SkippedFiles=0, got %d", stats.SkippedFiles)
	}
}

// TestFilter_MtimeComparison verifies mtime-based filtering logic
func TestFilter_MtimeComparison(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	taskRepo := repository.NewTaskRepository(db)
	filter := NewIncrementalFilter(taskRepo)

	// Create test files with specific mtimes
	baseTime := time.Now().Add(-2 * time.Hour)

	file1 := createTestFileWithMtime(t, tmpDir, "file1.md", baseTime.Add(-1*time.Hour)) // 3 hours ago
	file2 := createTestFileWithMtime(t, tmpDir, "file2.md", baseTime.Add(1*time.Hour))  // 1 hour ago
	file3 := createTestFileWithMtime(t, tmpDir, "file3.md", baseTime.Add(3*time.Hour))  // 1 hour in future from baseTime

	files := []TaskFileInfo{file1, file2, file3}

	// Add files to database (so they're not "new")
	ctx := context.Background()
	for i, f := range files {
		task := &models.Task{
			FeatureID: 1,
			Key:       fmt.Sprintf("T-E04-F07-%03d", i+1),
			Title:     "Test Task",
			Status:    models.TaskStatusTodo,
			Priority:  5,
			FilePath:  &f.FilePath,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Set last sync time to baseTime (2 hours ago)
	lastSyncTime := baseTime

	// Filter files
	result, stats, err := filter.Filter(ctx, files, FilterOptions{
		LastSyncTime:  &lastSyncTime,
		ForceFullScan: false,
	})
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// file1 mtime (3h ago) < lastSyncTime (2h ago) -> excluded
	// file2 mtime (1h ago) > lastSyncTime (2h ago) -> included
	// file3 mtime (1h future) > lastSyncTime (2h ago) -> included
	if len(result) != 2 {
		t.Errorf("Expected 2 files after filtering, got %d", len(result))
	}

	if stats.FilteredFiles != 2 {
		t.Errorf("Expected stats.FilteredFiles=2, got %d", stats.FilteredFiles)
	}

	if stats.SkippedFiles != 1 {
		t.Errorf("Expected stats.SkippedFiles=1, got %d", stats.SkippedFiles)
	}

	// Verify correct files were included
	resultPaths := make(map[string]bool)
	for _, f := range result {
		resultPaths[f.FilePath] = true
	}

	if !resultPaths[file2.FilePath] {
		t.Errorf("Expected file2 to be included")
	}

	if !resultPaths[file3.FilePath] {
		t.Errorf("Expected file3 to be included")
	}

	if resultPaths[file1.FilePath] {
		t.Errorf("Expected file1 to be excluded")
	}
}

// TestFilter_NewFiles verifies that new files (not in database) are always included
func TestFilter_NewFiles(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	taskRepo := repository.NewTaskRepository(db)
	filter := NewIncrementalFilter(taskRepo)

	// Create test files
	oldFile := createTestFileWithMtime(t, tmpDir, "old.md", time.Now().Add(-2*time.Hour))
	newFile := createTestFileWithMtime(t, tmpDir, "new.md", time.Now().Add(-2*time.Hour))

	files := []TaskFileInfo{oldFile, newFile}

	// Add only oldFile to database
	ctx := context.Background()
	task := &models.Task{
		FeatureID: 1,
		Key:       "T-E04-F07-001",
		Title:     "Old Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
		FilePath:  &oldFile.FilePath,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Set last sync time to now (oldFile should be filtered out based on mtime)
	now := time.Now()

	// Filter files
	result, stats, err := filter.Filter(ctx, files, FilterOptions{
		LastSyncTime:  &now,
		ForceFullScan: false,
	})
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// oldFile: in DB, mtime old -> excluded
	// newFile: NOT in DB -> included regardless of mtime
	if len(result) != 1 {
		t.Errorf("Expected 1 file (new file), got %d", len(result))
	}

	if result[0].FilePath != newFile.FilePath {
		t.Errorf("Expected newFile to be included, got %s", result[0].FilePath)
	}

	if stats.NewFiles != 1 {
		t.Errorf("Expected stats.NewFiles=1, got %d", stats.NewFiles)
	}

	if stats.FilteredFiles != 1 {
		t.Errorf("Expected stats.FilteredFiles=1, got %d", stats.FilteredFiles)
	}

	if stats.SkippedFiles != 1 {
		t.Errorf("Expected stats.SkippedFiles=1, got %d", stats.SkippedFiles)
	}
}

// TestFilter_ClockSkewTolerance verifies clock skew handling
func TestFilter_ClockSkewTolerance(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	taskRepo := repository.NewTaskRepository(db)
	filter := NewIncrementalFilter(taskRepo)

	// Create files with different future mtimes
	now := time.Now()
	file1 := createTestFileWithMtime(t, tmpDir, "file1.md", now.Add(30*time.Second)) // Small skew (30s)
	file2 := createTestFileWithMtime(t, tmpDir, "file2.md", now.Add(90*time.Second)) // Large skew (90s)

	files := []TaskFileInfo{file1, file2}

	// Add files to database
	ctx := context.Background()
	for i, f := range files {
		task := &models.Task{
			FeatureID: 1,
			Key:       fmt.Sprintf("T-E04-F07-%03d", i+1),
			Title:     "Test Task",
			Status:    models.TaskStatusTodo,
			Priority:  5,
			FilePath:  &f.FilePath,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Set last sync time to 1 hour ago
	lastSyncTime := now.Add(-1 * time.Hour)

	// Filter files
	result, stats, err := filter.Filter(ctx, files, FilterOptions{
		LastSyncTime:  &lastSyncTime,
		ForceFullScan: false,
	})
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// Both files should be included (mtime > lastSyncTime)
	if len(result) != 2 {
		t.Errorf("Expected 2 files, got %d", len(result))
	}

	// file1 (30s future): no warning (within tolerance)
	// file2 (90s future): warning (exceeds tolerance)
	hasWarning := false
	for _, warning := range stats.Warnings {
		if strings.Contains(warning, "file2.md") && strings.Contains(warning, "clock skew") {
			hasWarning = true
			break
		}
	}

	if !hasWarning {
		t.Errorf("Expected clock skew warning for file2.md, got warnings: %v", stats.Warnings)
	}

	// Should NOT have warning for file1
	for _, warning := range stats.Warnings {
		if strings.Contains(warning, "file1.md") {
			t.Errorf("Unexpected warning for file1.md: %s", warning)
		}
	}
}

// TestFilter_Performance verifies <100ms overhead for 500 files
func TestFilter_Performance(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	taskRepo := repository.NewTaskRepository(db)
	filter := NewIncrementalFilter(taskRepo)

	// Create 500 test files
	files := createTestFiles(t, tmpDir, 500)

	// Add files to database
	ctx := context.Background()
	for i, f := range files {
		task := &models.Task{
			FeatureID: 1,
			Key:       fmt.Sprintf("T-E04-F07-%03d", i+1),
			Title:     "Test Task",
			Status:    models.TaskStatusTodo,
			Priority:  5,
			FilePath:  &f.FilePath,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Set last sync time
	lastSyncTime := time.Now().Add(-1 * time.Hour)

	// Measure filtering performance
	start := time.Now()
	_, _, err := filter.Filter(ctx, files, FilterOptions{
		LastSyncTime:  &lastSyncTime,
		ForceFullScan: false,
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// Performance requirement: <100ms for 500 files
	if elapsed > 100*time.Millisecond {
		t.Errorf("Filter took %v, expected <100ms for 500 files", elapsed)
	}

	t.Logf("Filter performance: %v for 500 files (requirement: <100ms)", elapsed)
}

// TestFilter_EmptyFileList verifies handling of empty file list
func TestFilter_EmptyFileList(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	taskRepo := repository.NewTaskRepository(db)
	filter := NewIncrementalFilter(taskRepo)

	// Test with empty file list
	now := time.Now()
	result, stats, err := filter.Filter(context.Background(), []TaskFileInfo{}, FilterOptions{
		LastSyncTime:  &now,
		ForceFullScan: false,
	})
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 files, got %d", len(result))
	}

	if stats.TotalFiles != 0 {
		t.Errorf("Expected stats.TotalFiles=0, got %d", stats.TotalFiles)
	}
}

// TestFilter_MissingFile verifies handling of files that can't be stat'ed
func TestFilter_MissingFile(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	taskRepo := repository.NewTaskRepository(db)
	filter := NewIncrementalFilter(taskRepo)

	// Create file info for non-existent file
	missingFile := TaskFileInfo{
		FilePath:   filepath.Join(tmpDir, "missing.md"),
		FileName:   "missing.md",
		ModifiedAt: time.Now(),
	}

	files := []TaskFileInfo{missingFile}

	// Add to database (so it's not "new")
	ctx := context.Background()
	task := &models.Task{
		FeatureID: 1,
		Key:       "T-E04-F07-001",
		Title:     "Test Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
		FilePath:  &missingFile.FilePath,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Set last sync time
	lastSyncTime := time.Now().Add(-1 * time.Hour)

	// Filter files
	result, stats, err := filter.Filter(ctx, files, FilterOptions{
		LastSyncTime:  &lastSyncTime,
		ForceFullScan: false,
	})
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// Missing file should be skipped with warning
	if len(result) != 0 {
		t.Errorf("Expected 0 files (missing file skipped), got %d", len(result))
	}

	if stats.SkippedFiles != 1 {
		t.Errorf("Expected stats.SkippedFiles=1, got %d", stats.SkippedFiles)
	}

	// Should have warning about missing file
	hasWarning := false
	for _, warning := range stats.Warnings {
		if strings.Contains(warning, "missing.md") && strings.Contains(warning, "Cannot stat") {
			hasWarning = true
			break
		}
	}

	if !hasWarning {
		t.Errorf("Expected warning for missing file, got warnings: %v", stats.Warnings)
	}
}

// Helper functions

// setupTestDB initializes a test database and returns repository wrapper + cleanup function
func setupTestDB(t *testing.T, dbPath string) (*repository.DB, func()) {
	// Initialize database with schema
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create repository wrapper
	repoDb := repository.NewDB(database)

	// Create test epic and feature for foreign key constraints
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	epic := &models.Epic{
		Key:      "E04",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(context.Background(), epic); err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E04-F07",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	if err := featureRepo.Create(context.Background(), feature); err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	cleanup := func() {
		database.Close()
	}

	return repoDb, cleanup
}

func createTestFiles(t *testing.T, dir string, count int) []TaskFileInfo {
	var files []TaskFileInfo
	for i := 0; i < count; i++ {
		filename := fmt.Sprintf("T-E04-F07-%03d.md", i+1)
		filePath := filepath.Join(dir, filename)

		// Create file with some content
		if err := os.WriteFile(filePath, []byte("# Test Task\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Get file info for mtime
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat test file: %v", err)
		}

		files = append(files, TaskFileInfo{
			FilePath:   filePath,
			FileName:   filename,
			ModifiedAt: info.ModTime(),
		})
	}
	return files
}

func createTestFileWithMtime(t *testing.T, dir, filename string, mtime time.Time) TaskFileInfo {
	filePath := filepath.Join(dir, filename)

	// Create file
	if err := os.WriteFile(filePath, []byte("# Test Task\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set mtime using Chtimes
	if err := os.Chtimes(filePath, mtime, mtime); err != nil {
		t.Fatalf("Failed to set mtime: %v", err)
	}

	return TaskFileInfo{
		FilePath:   filePath,
		FileName:   filename,
		ModifiedAt: mtime,
	}
}
