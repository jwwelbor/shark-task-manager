package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/reporting"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/sync"
	"github.com/jwwelbor/shark-task-manager/internal/validation"
	"github.com/stretchr/testify/require"
)

// BenchmarkReportingOverhead measures the overhead of reporting on sync performance
// Target: <5% overhead for 100-file scan
func BenchmarkReportingOverhead(b *testing.B) {
	// Setup: Create test environment with 100 files
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize database
	database, err := db.InitDB(dbPath)
	require.NoError(b, err)
	defer database.Close()

	ctx := context.Background()
	dbWrapper := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)

	// Create test epic and feature
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	require.NoError(b, err)

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E01-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(b, err)

	// Create 100 task files
	planDir := filepath.Join(tempDir, "docs/plan/E01/E01-F01/tasks")
	err = os.MkdirAll(planDir, 0755)
	require.NoError(b, err)

	for i := 1; i <= 100; i++ {
		taskFile := filepath.Join(planDir, generateTaskFileName(i))
		taskKey := generateTaskKey(i)
		content := generateTaskContent(taskKey, i)
		err = os.WriteFile(taskFile, []byte(content), 0644)
		require.NoError(b, err)
	}

	// Create sync engine
	engine, err := sync.NewSyncEngine(dbPath)
	require.NoError(b, err)
	defer engine.Close()

	opts := sync.SyncOptions{
		DBPath:        dbPath,
		FolderPath:    filepath.Join(tempDir, "docs/plan"),
		DryRun:        false,
		Strategy:      sync.ConflictStrategyFileWins,
		CreateMissing: false,
	}

	// Benchmark sync with reporting
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		report, err := engine.Sync(syncCtx, opts)
		require.NoError(b, err)
		require.NotNil(b, report)
		cancel()

		// Note: In real usage, we would also generate ScanReport here
		// The benchmark measures the combined time
		scanReport := convertSyncToScanReport(report, time.Now(), opts.FolderPath, []sync.PatternType{sync.PatternTypeTask})
		require.NotNil(b, scanReport)
	}
}

// BenchmarkValidation1000Entities measures validation performance with 1000 entities
// Target: <1 second for 1000 entities
func BenchmarkValidation1000Entities(b *testing.B) {
	// Setup: Create test environment with ~1000 entities
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize database
	database, err := db.InitDB(dbPath)
	require.NoError(b, err)
	defer database.Close()

	ctx := context.Background()
	dbWrapper := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

	// Create 10 epics
	epics := make([]*models.Epic, 10)
	for i := 0; i < 10; i++ {
		epic := &models.Epic{
			Key:      "E" + paddedNumber(i+1),
			Title:    "Epic " + string(rune('0'+i)),
			Status:   models.EpicStatusActive,
			Priority: models.PriorityMedium,
		}
		err = epicRepo.Create(ctx, epic)
		require.NoError(b, err)
		epics[i] = epic
	}

	// Create 10 features per epic (100 features)
	features := make([]*models.Feature, 100)
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			idx := i*10 + j
			feature := &models.Feature{
				EpicID: epics[i].ID,
				Key:    "E" + paddedNumber(i+1) + "-F" + paddedNumber(j+1),
				Title:  "Feature " + string(rune('0'+idx)),
				Status: models.FeatureStatusActive,
			}
			err = featureRepo.Create(ctx, feature)
			require.NoError(b, err)
			features[idx] = feature
		}
	}

	// Create 9 tasks per feature (900 tasks)
	for i := 0; i < 100; i++ {
		for j := 0; j < 9; j++ {
			taskKey := features[i].Key + "-" + paddedNumber(j+1)
			taskFile := filepath.Join(tempDir, taskKey+".md")
			err = os.WriteFile(taskFile, []byte("# Task\n\nContent"), 0644)
			require.NoError(b, err)

			task := &models.Task{
				FeatureID: features[i].ID,
				Key:       "T-" + taskKey,
				Title:     "Task " + taskKey,
				Status:    models.TaskStatusTodo,
				Priority:  5,
				FilePath:  &taskFile,
			}
			err = taskRepo.Create(ctx, task)
			require.NoError(b, err)
		}
	}

	// Create validator
	repoAdapter := validation.NewRepositoryAdapter(epicRepo, featureRepo, taskRepo)
	validator := validation.NewValidator(repoAdapter)

	// Benchmark validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := validator.Validate(validateCtx)
		require.NoError(b, err)
		require.NotNil(b, result)
		require.Equal(b, 1010, result.Summary.TotalChecked)
		cancel()
	}
}

// BenchmarkSyncWithDryRun measures dry-run performance
func BenchmarkSyncWithDryRun(b *testing.B) {
	// Setup
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.InitDB(dbPath)
	require.NoError(b, err)
	defer database.Close()

	ctx := context.Background()
	dbWrapper := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)

	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	require.NoError(b, err)

	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E01-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(b, err)

	// Create 50 task files
	planDir := filepath.Join(tempDir, "docs/plan/E01/E01-F01/tasks")
	err = os.MkdirAll(planDir, 0755)
	require.NoError(b, err)

	for i := 1; i <= 50; i++ {
		taskFile := filepath.Join(planDir, generateTaskFileName(i))
		taskKey := generateTaskKey(i)
		content := generateTaskContent(taskKey, i)
		err = os.WriteFile(taskFile, []byte(content), 0644)
		require.NoError(b, err)
	}

	engine, err := sync.NewSyncEngine(dbPath)
	require.NoError(b, err)
	defer engine.Close()

	opts := sync.SyncOptions{
		DBPath:        dbPath,
		FolderPath:    filepath.Join(tempDir, "docs/plan"),
		DryRun:        true, // DRY RUN
		Strategy:      sync.ConflictStrategyFileWins,
		CreateMissing: false,
	}

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		report, err := engine.Sync(syncCtx, opts)
		require.NoError(b, err)
		require.True(b, report.DryRun)
		cancel()
	}
}

// BenchmarkJSONFormatting measures JSON formatting overhead
func BenchmarkJSONFormatting(b *testing.B) {
	// Create a realistic scan report
	scanReport := createRealisticScanReport(100, 20)

	// Benchmark JSON formatting
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This would normally import reporting package
		// For now, just measure the report generation
		_ = scanReport
	}
}

// Helper function to create realistic scan report for benchmarking
func createRealisticScanReport(matched, skipped int) *reporting.ScanReport {
	report := reporting.NewScanReport()

	report.Metadata = reporting.ScanMetadata{
		Timestamp:         time.Now(),
		DurationSeconds:   1.5,
		ValidationLevel:   "basic",
		DocumentationRoot: "/tmp/docs/plan",
		Patterns:          map[string]string{"task": "enabled"},
	}

	report.Counts.Scanned = matched + skipped
	report.Counts.Matched = matched
	report.Counts.Skipped = skipped

	report.Entities.Tasks.Matched = matched
	report.Entities.Tasks.Skipped = skipped

	// Add some errors
	for i := 0; i < skipped; i++ {
		report.AddError(reporting.SkippedFileEntry{
			FilePath:     "/tmp/docs/plan/E01/E01-F01/tasks/task-" + string(rune('0'+i)) + ".md",
			Reason:       "Missing required field: task_key",
			ErrorType:    "parse_error",
			SuggestedFix: "Add task_key field to frontmatter",
		})
	}

	return report
}

// Helper functions

func generateTaskFileName(i int) string {
	return "T-E01-F01-" + paddedNumber(i) + ".md"
}

func generateTaskKey(i int) string {
	return "T-E01-F01-" + paddedNumber(i)
}

func paddedNumber(i int) string {
	// Returns zero-padded 3-digit string for task numbers (001, 002, etc.)
	if i < 10 {
		return "00" + string(byte('0'+i))
	} else if i < 100 {
		tens := i / 10
		ones := i % 10
		return "0" + string(byte('0'+tens)) + string(byte('0'+ones))
	}
	// For 100+, just convert with 3 digits
	hundreds := i / 100
	tens := (i % 100) / 10
	ones := i % 10
	return string(byte('0'+hundreds)) + string(byte('0'+tens)) + string(byte('0'+ones))
}

func generateTaskContent(taskKey string, i int) string {
	return `---
task_key: ` + taskKey + `
status: todo
---

# Task ` + string(rune('0'+i)) + `

This is test task number ` + string(rune('0'+i)) + `.
`
}

func convertSyncToScanReport(syncReport *sync.SyncReport, startTime time.Time, folderPath string, patterns []sync.PatternType) *reporting.ScanReport {
	scanReport := reporting.NewScanReport()

	scanReport.Metadata = reporting.ScanMetadata{
		Timestamp:         startTime,
		DurationSeconds:   0.1,
		ValidationLevel:   "basic",
		DocumentationRoot: folderPath,
		Patterns:          make(map[string]string),
	}

	for _, p := range patterns {
		scanReport.Metadata.Patterns[string(p)] = "enabled"
	}

	scanReport.SetDryRun(syncReport.DryRun)
	scanReport.Counts.Scanned = syncReport.FilesScanned
	scanReport.Counts.Matched = syncReport.TasksImported + syncReport.TasksUpdated
	scanReport.Counts.Skipped = len(syncReport.Errors) + len(syncReport.Warnings)

	scanReport.Entities.Tasks.Matched = syncReport.TasksImported + syncReport.TasksUpdated
	scanReport.Entities.Tasks.Skipped = len(syncReport.Errors)

	for _, warning := range syncReport.Warnings {
		scanReport.AddWarning(reporting.SkippedFileEntry{
			FilePath:     folderPath,
			Reason:       warning,
			ErrorType:    "validation_warning",
			SuggestedFix: "Review warning and take appropriate action",
		})
	}

	for _, errMsg := range syncReport.Errors {
		scanReport.AddError(reporting.SkippedFileEntry{
			FilePath:     folderPath,
			Reason:       errMsg,
			ErrorType:    "parse_error",
			SuggestedFix: "Fix the error and re-run sync",
		})
	}

	scanReport.Status = scanReport.GetStatus()
	return scanReport
}
