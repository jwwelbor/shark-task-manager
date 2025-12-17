package sync

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/taskfile"
	_ "github.com/mattn/go-sqlite3"
)

// SyncEngine orchestrates synchronization between filesystem and database
type SyncEngine struct {
	db          *sql.DB
	taskRepo    *repository.TaskRepository
	epicRepo    *repository.EpicRepository
	featureRepo *repository.FeatureRepository
	scanner     *FileScanner
	detector    *ConflictDetector
	resolver    *ConflictResolver
}

// NewSyncEngine creates a new SyncEngine instance with default patterns (task only)
func NewSyncEngine(dbPath string) (*SyncEngine, error) {
	return NewSyncEngineWithPatterns(dbPath, []PatternType{PatternTypeTask})
}

// NewSyncEngineWithPatterns creates a new SyncEngine instance with specific patterns enabled
func NewSyncEngineWithPatterns(dbPath string, patterns []PatternType) (*SyncEngine, error) {
	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create repository wrapper
	repoDb := repository.NewDB(db)

	return &SyncEngine{
		db:          db,
		taskRepo:    repository.NewTaskRepository(repoDb),
		epicRepo:    repository.NewEpicRepository(repoDb),
		featureRepo: repository.NewFeatureRepository(repoDb),
		scanner:     NewFileScannerWithPatterns(patterns),
		detector:    NewConflictDetector(),
		resolver:    NewConflictResolver(),
	}, nil
}

// Close closes the database connection
func (e *SyncEngine) Close() error {
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}

// Sync performs synchronization between filesystem and database
func (e *SyncEngine) Sync(ctx context.Context, opts SyncOptions) (*SyncReport, error) {
	report := &SyncReport{
		Warnings:  []string{},
		Errors:    []string{},
		Conflicts: []Conflict{},
	}

	// Step 1: Scan files
	files, err := e.scanner.Scan(opts.FolderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan files: %w", err)
	}
	report.FilesScanned = len(files)

	// If no files found, return early
	if len(files) == 0 {
		return report, nil
	}

	// Step 2: Parse files and build task metadata list
	taskDataList, parseWarnings := e.parseFiles(files)
	report.Warnings = append(report.Warnings, parseWarnings...)

	// If no valid tasks parsed, return early
	if len(taskDataList) == 0 {
		return report, nil
	}

	// Step 3: Query database for all task keys
	taskKeys := extractTaskKeys(taskDataList)
	dbTasks, err := e.taskRepo.GetByKeys(ctx, taskKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to query database tasks: %w", err)
	}

	// Step 4: Begin transaction (unless dry-run)
	var tx *sql.Tx
	if !opts.DryRun {
		tx, err = e.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		// Defer rollback - will be no-op if we commit successfully
		defer tx.Rollback()
	}

	// Step 5: Process each task
	for _, taskData := range taskDataList {
		if err := e.syncTask(ctx, tx, taskData, dbTasks, opts, report); err != nil {
			// Fatal error - rollback will happen via defer
			return report, fmt.Errorf("sync failed for task %s: %w", taskData.Key, err)
		}
	}

	// Step 6: Commit transaction (unless dry-run)
	if !opts.DryRun && tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return report, nil
}

// parseFiles parses all task files and returns task metadata with warnings
func (e *SyncEngine) parseFiles(files []TaskFileInfo) ([]*TaskMetadata, []string) {
	var taskDataList []*TaskMetadata
	var warnings []string

	// Create key generator for PRP files
	keyGen := taskcreation.NewKeyGenerator(e.taskRepo, e.featureRepo)

	for _, file := range files {
		// Parse task file
		taskFile, err := taskfile.ParseTaskFile(file.FilePath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse %s: %v", file.FilePath, err))
			continue
		}

		// Handle missing task_key (common for PRP files)
		if taskFile.Metadata.TaskKey == "" {
			// For PRP files, generate task key from epic/feature
			if file.PatternType == PatternTypePRP {
				// Validate epic/feature were inferred
				if file.EpicKey == "" || file.FeatureKey == "" {
					warnings = append(warnings, fmt.Sprintf("Cannot generate task_key for %s: missing epic/feature in path", file.FilePath))
					continue
				}

				// Generate task key
				ctx := context.Background()
				taskKey, err := keyGen.GenerateTaskKey(ctx, file.EpicKey, file.FeatureKey)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("Failed to generate task_key for %s: %v", file.FilePath, err))
					continue
				}

				// Update file frontmatter with generated key
				if err := taskfile.UpdateFrontmatterField(file.FilePath, "task_key", taskKey); err != nil {
					warnings = append(warnings, fmt.Sprintf("Generated key %s but couldn't write to %s: %v", taskKey, file.FilePath, err))
					// Continue anyway - we can still use the generated key for this sync
				}

				// Use generated key
				taskFile.Metadata.TaskKey = taskKey
			} else {
				// For task pattern files, task_key is required
				warnings = append(warnings, fmt.Sprintf("Missing task_key in %s", file.FilePath))
				continue
			}
		}

		// Build task metadata
		taskData := &TaskMetadata{
			Key:        taskFile.Metadata.TaskKey,
			Title:      taskFile.Metadata.Title,
			FilePath:   file.FilePath,
			ModifiedAt: file.ModifiedAt,
		}

		// Add description if present
		if taskFile.Metadata.Description != "" {
			taskData.Description = &taskFile.Metadata.Description
		}

		taskDataList = append(taskDataList, taskData)
	}

	return taskDataList, warnings
}

// syncTask synchronizes a single task
func (e *SyncEngine) syncTask(ctx context.Context, tx *sql.Tx, taskData *TaskMetadata,
	dbTasks map[string]*models.Task, opts SyncOptions, report *SyncReport) error {

	// Check if task exists in database
	dbTask, exists := dbTasks[taskData.Key]

	if !exists {
		// New task - import
		return e.importTask(ctx, tx, taskData, opts, report)
	}

	// Existing task - update if conflicts detected
	return e.updateTask(ctx, tx, taskData, dbTask, opts, report)
}

// importTask imports a new task into the database
func (e *SyncEngine) importTask(ctx context.Context, tx *sql.Tx, taskData *TaskMetadata,
	opts SyncOptions, report *SyncReport) error {

	// Extract epic and feature keys from task key
	// Task key format: T-E##-F##-###
	epicKey, featureKey, err := parseTaskKey(taskData.Key)
	if err != nil {
		return fmt.Errorf("invalid task key format %s: %w", taskData.Key, err)
	}

	// Get feature from database
	feature, err := e.featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		if err == sql.ErrNoRows {
			if opts.CreateMissing {
				// Auto-create feature and epic
				feature, err = e.createMissingFeature(ctx, tx, epicKey, featureKey)
				if err != nil {
					return fmt.Errorf("failed to create missing feature: %w", err)
				}
			} else {
				return fmt.Errorf("feature %s not found (use --create-missing to auto-create)", featureKey)
			}
		} else {
			return fmt.Errorf("failed to get feature: %w", err)
		}
	}

	// Create task model
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       taskData.Key,
		Title:     taskData.Title,
		Status:    models.TaskStatusTodo, // Default status for new tasks
		Priority:  5,                     // Default priority
	}

	// Add description if present
	if taskData.Description != nil {
		task.Description = taskData.Description
	}

	// Set file path
	task.FilePath = &taskData.FilePath

	// Create task (skip in dry-run mode)
	if !opts.DryRun {
		if err := e.taskRepo.Create(ctx, task); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		// Create task history record
		if err := e.createTaskHistory(ctx, task.ID, "Imported from file"); err != nil {
			// Log warning but don't fail sync
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("Failed to create history for task %s: %v", task.Key, err))
		}
	}

	report.TasksImported++
	return nil
}

// updateTask updates an existing task if conflicts are detected
func (e *SyncEngine) updateTask(ctx context.Context, tx *sql.Tx, taskData *TaskMetadata,
	dbTask *models.Task, opts SyncOptions, report *SyncReport) error {

	// Detect conflicts
	conflicts := e.detector.DetectConflicts(taskData, dbTask)

	// If no conflicts, nothing to update
	if len(conflicts) == 0 {
		return nil
	}

	// Resolve conflicts
	resolvedTask, err := e.resolver.ResolveConflicts(conflicts, taskData, dbTask, opts.Strategy)
	if err != nil {
		return fmt.Errorf("failed to resolve conflicts: %w", err)
	}

	// Update task (skip in dry-run mode)
	if !opts.DryRun {
		if err := e.taskRepo.UpdateMetadata(ctx, resolvedTask); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		// Create task history record
		historyMsg := fmt.Sprintf("Updated from file (%d conflicts resolved)", len(conflicts))
		if err := e.createTaskHistory(ctx, resolvedTask.ID, historyMsg); err != nil {
			// Log warning but don't fail sync
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("Failed to create history for task %s: %v", resolvedTask.Key, err))
		}
	}

	// Add conflicts to report
	for _, conflict := range conflicts {
		report.Conflicts = append(report.Conflicts, conflict)
	}

	report.TasksUpdated++
	report.ConflictsResolved += len(conflicts)
	return nil
}

// createMissingFeature creates a feature and epic if they don't exist
func (e *SyncEngine) createMissingFeature(ctx context.Context, tx *sql.Tx,
	epicKey, featureKey string) (*models.Feature, error) {

	// Get or create epic
	epic, err := e.epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create epic
			epic = &models.Epic{
				Key:      epicKey,
				Title:    fmt.Sprintf("Auto-created epic %s", epicKey),
				Status:   models.EpicStatusActive,
				Priority: models.PriorityMedium,
			}
			if err := e.epicRepo.Create(ctx, epic); err != nil {
				return nil, fmt.Errorf("failed to create epic: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get epic: %w", err)
		}
	}

	// Create feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    featureKey,
		Title:  fmt.Sprintf("Auto-created feature %s", featureKey),
		Status: models.FeatureStatusActive,
	}
	if err := e.featureRepo.Create(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to create feature: %w", err)
	}

	return feature, nil
}

// createTaskHistory creates a task history record
func (e *SyncEngine) createTaskHistory(ctx context.Context, taskID int64, message string) error {
	query := `
		INSERT INTO task_history (task_id, status_from, status_to, changed_by, change_description)
		VALUES (?, '', '', 'pm-sync', ?)
	`
	_, err := e.db.ExecContext(ctx, query, taskID, message)
	return err
}

// extractTaskKeys extracts all task keys from task data list
func extractTaskKeys(taskDataList []*TaskMetadata) []string {
	keys := make([]string, len(taskDataList))
	for i, taskData := range taskDataList {
		keys[i] = taskData.Key
	}
	return keys
}

// parseTaskKey parses epic and feature keys from task key
// Task key format: T-E##-F##-###
// Returns: epicKey, featureKey, error
func parseTaskKey(taskKey string) (string, string, error) {
	// Expected format: T-E04-F07-001 (13 characters)
	// Positions:       0123456789012
	if len(taskKey) < 13 {
		return "", "", fmt.Errorf("invalid task key format: %s", taskKey)
	}

	// Extract E## (positions 2-5, e.g., "E04")
	epicKey := taskKey[2:5]
	if len(epicKey) < 3 || epicKey[0] != 'E' {
		return "", "", fmt.Errorf("invalid epic key in task key: %s", taskKey)
	}

	// Extract E##-F## (positions 2-9, e.g., "E04-F07")
	if len(taskKey) < 9 {
		return "", "", fmt.Errorf("task key too short for feature: %s", taskKey)
	}
	featureKey := taskKey[2:9]
	if len(featureKey) != 7 || featureKey[3] != '-' || featureKey[4] != 'F' {
		return "", "", fmt.Errorf("invalid feature key in task key: %s", taskKey)
	}

	return epicKey, featureKey, nil
}
