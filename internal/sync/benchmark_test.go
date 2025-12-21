package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// BenchmarkIncrementalSync_10Files benchmarks incremental sync with 10 files
// Success criteria: <2 seconds for 1-10 file changes
func BenchmarkIncrementalSync_10Files(b *testing.B) {
	benchmarkIncrementalSync(b, 10, 10, 2.0)
}

// BenchmarkIncrementalSync_100Files benchmarks incremental sync with 100 files
// Success criteria: <5 seconds for 100 file changes
func BenchmarkIncrementalSync_100Files(b *testing.B) {
	benchmarkIncrementalSync(b, 100, 100, 5.0)
}

// BenchmarkIncrementalSync_500Files benchmarks incremental sync with 500 files
// Success criteria: <30 seconds for 500 file changes
func BenchmarkIncrementalSync_500Files(b *testing.B) {
	benchmarkIncrementalSync(b, 500, 500, 30.0)
}

// BenchmarkIncrementalSync_1000Files benchmarks incremental sync with 1000 files
// No specific requirement, but useful for scalability testing
func BenchmarkIncrementalSync_1000Files(b *testing.B) {
	benchmarkIncrementalSync(b, 1000, 1000, 60.0)
}

// BenchmarkIncrementalSync_NoChanges benchmarks incremental sync with no changes
// Success criteria: <1 second with no changes
func BenchmarkIncrementalSync_NoChanges(b *testing.B) {
	benchmarkIncrementalSync(b, 100, 0, 1.0)
}

// BenchmarkIncrementalSync_10Changed_100Total benchmarks selective sync
// 10 changed files out of 100 total
func BenchmarkIncrementalSync_10Changed_100Total(b *testing.B) {
	benchmarkIncrementalSyncSelective(b, 100, 10, 2.0)
}

// BenchmarkIncrementalSync_50Changed_500Total benchmarks selective sync
// 50 changed files out of 500 total
func BenchmarkIncrementalSync_50Changed_500Total(b *testing.B) {
	benchmarkIncrementalSyncSelective(b, 500, 50, 10.0)
}

// BenchmarkFullScan_100Files benchmarks traditional full scan for comparison
func BenchmarkFullScan_100Files(b *testing.B) {
	benchmarkFullScan(b, 100)
}

// BenchmarkFullScan_500Files benchmarks traditional full scan for comparison
func BenchmarkFullScan_500Files(b *testing.B) {
	benchmarkFullScan(b, 500)
}

// benchmarkIncrementalSync runs a benchmark where all files are changed
func benchmarkIncrementalSync(b *testing.B, totalFiles int, changedFiles int, maxSeconds float64) {
	// Setup test environment
	testDir := b.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")
	require.NoError(b, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(b, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(b, db)

	// Create task files
	taskFiles := createBenchmarkTaskFiles(b, docsPath, totalFiles)

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	require.NoError(b, err)
	defer engine.Close()

	// Initial sync to populate database
	initialOpts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  nil,
	}
	_, err = engine.Sync(context.Background(), initialOpts)
	require.NoError(b, err)

	// Set last sync time
	lastSyncTime := time.Now()
	time.Sleep(100 * time.Millisecond)

	// Modify specified number of files
	for i := 0; i < changedFiles; i++ {
		taskKey := fmt.Sprintf("T-E01-F01-%04d", i+1)
		updateTestTaskFile(b, taskFiles[i], taskKey, fmt.Sprintf("Updated Task %d", i+1))
	}

	// Reset timer before benchmark
	b.ResetTimer()

	// Run benchmark
	var totalDuration time.Duration
	for i := 0; i < b.N; i++ {
		opts := SyncOptions{
			DBPath:        dbPath,
			FolderPath:    docsPath,
			DryRun:        false,
			Strategy:      ConflictStrategyFileWins,
			CreateMissing: true,
			LastSyncTime:  &lastSyncTime,
		}

		start := time.Now()
		report, err := engine.Sync(context.Background(), opts)
		duration := time.Since(start)
		totalDuration += duration

		require.NoError(b, err)
		require.Equal(b, changedFiles, report.FilesFiltered, "Should process %d changed files", changedFiles)
	}

	b.StopTimer()

	// Calculate average duration
	avgDuration := totalDuration.Seconds() / float64(b.N)

	// Log results
	b.Logf("Average duration for %d changed files (out of %d total): %.3f seconds",
		changedFiles, totalFiles, avgDuration)

	// Validate performance requirement
	if avgDuration > maxSeconds {
		b.Errorf("Performance requirement failed: %.3f seconds (max: %.1f seconds)", avgDuration, maxSeconds)
	}
}

// benchmarkIncrementalSyncSelective runs a benchmark where only some files are changed
func benchmarkIncrementalSyncSelective(b *testing.B, totalFiles int, changedFiles int, maxSeconds float64) {
	// Setup test environment
	testDir := b.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")
	require.NoError(b, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(b, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(b, db)

	// Create task files
	taskFiles := createBenchmarkTaskFiles(b, docsPath, totalFiles)

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	require.NoError(b, err)
	defer engine.Close()

	// Initial sync to populate database
	initialOpts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  nil,
	}
	_, err = engine.Sync(context.Background(), initialOpts)
	require.NoError(b, err)

	// Set last sync time
	lastSyncTime := time.Now()
	time.Sleep(100 * time.Millisecond)

	// Modify only specified number of files
	for i := 0; i < changedFiles; i++ {
		taskKey := fmt.Sprintf("T-E01-F01-%04d", i+1)
		updateTestTaskFile(b, taskFiles[i], taskKey, fmt.Sprintf("Updated Task %d", i+1))
	}

	// Reset timer before benchmark
	b.ResetTimer()

	// Run benchmark
	var totalDuration time.Duration
	for i := 0; i < b.N; i++ {
		opts := SyncOptions{
			DBPath:        dbPath,
			FolderPath:    docsPath,
			DryRun:        false,
			Strategy:      ConflictStrategyFileWins,
			CreateMissing: true,
			LastSyncTime:  &lastSyncTime,
		}

		start := time.Now()
		report, err := engine.Sync(context.Background(), opts)
		duration := time.Since(start)
		totalDuration += duration

		require.NoError(b, err)
		require.Equal(b, totalFiles, report.FilesScanned, "Should scan all files")
		require.Equal(b, changedFiles, report.FilesFiltered, "Should filter %d changed files", changedFiles)
		require.Equal(b, totalFiles-changedFiles, report.FilesSkipped, "Should skip %d unchanged files", totalFiles-changedFiles)
	}

	b.StopTimer()

	// Calculate average duration
	avgDuration := totalDuration.Seconds() / float64(b.N)

	// Log results
	b.Logf("Average duration for %d changed files (out of %d total): %.3f seconds",
		changedFiles, totalFiles, avgDuration)
	b.Logf("Improvement: %.1f%% of files skipped", float64(totalFiles-changedFiles)/float64(totalFiles)*100)

	// Validate performance requirement
	if avgDuration > maxSeconds {
		b.Errorf("Performance requirement failed: %.3f seconds (max: %.1f seconds)", avgDuration, maxSeconds)
	}
}

// benchmarkFullScan runs a benchmark of traditional full scan (no incremental filtering)
func benchmarkFullScan(b *testing.B, totalFiles int) {
	// Setup test environment
	testDir := b.TempDir()
	dbPath := filepath.Join(testDir, "test.db")
	docsPath := filepath.Join(testDir, "docs", "plan")
	require.NoError(b, os.MkdirAll(docsPath, 0755))

	// Initialize database
	db := setupTestDatabase(b, dbPath)
	defer db.Close()

	// Create test epic and feature
	setupTestEpicAndFeature(b, db)

	// Create task files
	createBenchmarkTaskFiles(b, docsPath, totalFiles)

	// Create sync engine
	engine, err := NewSyncEngine(dbPath)
	require.NoError(b, err)
	defer engine.Close()

	// Initial sync to populate database
	initialOpts := SyncOptions{
		DBPath:        dbPath,
		FolderPath:    docsPath,
		DryRun:        false,
		Strategy:      ConflictStrategyFileWins,
		CreateMissing: true,
		LastSyncTime:  nil,
	}
	_, err = engine.Sync(context.Background(), initialOpts)
	require.NoError(b, err)

	// Reset timer before benchmark
	b.ResetTimer()

	// Run benchmark (no incremental filtering)
	var totalDuration time.Duration
	for i := 0; i < b.N; i++ {
		opts := SyncOptions{
			DBPath:        dbPath,
			FolderPath:    docsPath,
			DryRun:        false,
			Strategy:      ConflictStrategyFileWins,
			CreateMissing: true,
			LastSyncTime:  nil, // No incremental filtering
		}

		start := time.Now()
		report, err := engine.Sync(context.Background(), opts)
		duration := time.Since(start)
		totalDuration += duration

		require.NoError(b, err)
		require.Equal(b, totalFiles, report.FilesScanned, "Should scan all files")
	}

	b.StopTimer()

	// Calculate average duration
	avgDuration := totalDuration.Seconds() / float64(b.N)

	// Log results
	b.Logf("Average duration for full scan of %d files: %.3f seconds", totalFiles, avgDuration)
}

// Helper functions

// createBenchmarkTaskFiles creates multiple task files for benchmarking
func createBenchmarkTaskFiles(tb testing.TB, docsPath string, count int) []string {
	var taskFiles []string
	for i := 1; i <= count; i++ {
		taskKey := fmt.Sprintf("T-E01-F01-%04d", i)
		taskFile := filepath.Join(docsPath, taskKey+".md")
		createTestTaskFile(tb, taskFile, taskKey, fmt.Sprintf("Task %d", i))
		taskFiles = append(taskFiles, taskFile)
	}
	return taskFiles
}
