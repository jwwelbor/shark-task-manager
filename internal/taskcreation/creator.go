package taskcreation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/patterns"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

// Creator orchestrates the complete task creation workflow
type Creator struct {
	db          *repository.DB
	keygen      *KeyGenerator
	validator   *Validator
	renderer    *templates.Renderer
	taskRepo    *repository.TaskRepository
	historyRepo *repository.TaskHistoryRepository
	epicRepo    *repository.EpicRepository
	featureRepo *repository.FeatureRepository
	projectRoot string
}

// NewCreator creates a new task creator
func NewCreator(
	db *repository.DB,
	keygen *KeyGenerator,
	validator *Validator,
	renderer *templates.Renderer,
	taskRepo *repository.TaskRepository,
	historyRepo *repository.TaskHistoryRepository,
	epicRepo *repository.EpicRepository,
	featureRepo *repository.FeatureRepository,
	projectRoot string,
) *Creator {
	return &Creator{
		db:          db,
		keygen:      keygen,
		validator:   validator,
		renderer:    renderer,
		taskRepo:    taskRepo,
		historyRepo: historyRepo,
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		projectRoot: projectRoot,
	}
}

// CreateTaskInput holds the input for creating a task
type CreateTaskInput struct {
	EpicKey        string
	FeatureKey     string
	Title          string
	Description    string
	AgentType      string
	CustomTemplate string
	Priority       int
	DependsOn      string
	ExecutionOrder int
	CustomKey      string // Custom key override (optional)
	Filename       string // Custom filename path (relative to project root)
	Force          bool   // Force reassignment if file already claimed
}

// CreateTaskResult holds the result of task creation
type CreateTaskResult struct {
	Task     *models.Task
	FilePath string
}

// CreateTask orchestrates the complete task creation workflow
func (c *Creator) CreateTask(ctx context.Context, input CreateTaskInput) (*CreateTaskResult, error) {
	// 1. Validate all inputs
	validated, err := c.validator.ValidateTaskInput(ctx, TaskInput{
		EpicKey:     input.EpicKey,
		FeatureKey:  input.FeatureKey,
		Title:       input.Title,
		Description: input.Description,
		AgentType:   input.AgentType,
		Priority:    input.Priority,
		DependsOn:   input.DependsOn,
	})
	if err != nil {
		return nil, err
	}

	// 2. Generate or use custom task key
	var key string
	if input.CustomKey != "" {
		// Validate custom key doesn't already exist
		existing, err := c.taskRepo.GetByKey(ctx, input.CustomKey)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("task with key %s already exists", input.CustomKey)
		}
		key = input.CustomKey
	} else {
		// Auto-generate task key
		var err error
		key, err = c.keygen.GenerateTaskKey(ctx, input.EpicKey, validated.NormalizedFeatureKey)
		if err != nil {
			return nil, err
		}
	}

	// 3. Begin database transaction
	tx, err := c.db.BeginTxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 4. Prepare task data
	now := time.Now().UTC()

	// Determine file path based on custom filename or default
	var filePath string     // Relative path for database
	var fullFilePath string // Absolute path for file operations
	var fileExists bool

	if input.Filename != "" {
		// Custom filename - validate it
		absPath, relPath, err := ValidateCustomFilename(input.Filename, c.projectRoot)
		if err != nil {
			return nil, fmt.Errorf("invalid filename: %w", err)
		}

		filePath = relPath
		fullFilePath = absPath

		// Check if file exists
		if _, statErr := os.Stat(fullFilePath); statErr == nil {
			fileExists = true
		}
	} else {
		// Default: derive task path from feature's actual location
		// 1. Fetch feature from database
		feature, err := c.featureRepo.GetByKey(ctx, validated.NormalizedFeatureKey)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch feature: %w", err)
		}

		// 2. Resolve task path based on feature's base path
		// Note: We use PathResolver's logic but can't call it directly since task doesn't exist yet
		if feature.FilePath != nil && *feature.FilePath != "" {
			// Feature has a file path - derive task path from it
			// Example: feature.FilePath = "docs/plan/E10-advanced-task.../E10-F01-task-notes/feature.md"
			// Task path should be:      "docs/plan/E10-advanced-task.../E10-F01-task-notes/tasks/T-E10-F01-001.md"
			featureDir := filepath.Dir(*feature.FilePath)
			relPath := filepath.Join(featureDir, "tasks", key+".md")
			fullFilePath = filepath.Join(c.projectRoot, relPath)
			filePath = relPath
		} else {
			// Default: compute path based on feature's default location
			// We need to manually construct since task doesn't exist yet
			epic, err := c.epicRepo.GetByID(ctx, feature.EpicID)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch epic: %w", err)
			}

			// Default: docs/plan/{epic-key}/{feature-key}
			epicSlug := ""
			if epic.Slug != nil && *epic.Slug != "" {
				epicSlug = *epic.Slug
			} else {
				epicSlug = epic.Key
			}
			featureSlug := ""
			if feature.Slug != nil && *feature.Slug != "" {
				featureSlug = *feature.Slug
			} else {
				featureSlug = feature.Key
			}
			epicFolder := epic.Key + "-" + epicSlug
			featureFolder := feature.Key + "-" + featureSlug
			featureBaseDir := filepath.Join("docs", "plan", epicFolder, featureFolder)

			// Task path: {featureBaseDir}/tasks/{task-key}.md
			taskFilename := key + ".md"
			relPath := filepath.Join(featureBaseDir, "tasks", taskFilename)
			fullFilePath = filepath.Join(c.projectRoot, relPath)
			filePath = relPath
		}

		// 3. Create tasks directory if it doesn't exist (creates all parents)
		tasksDir := filepath.Dir(fullFilePath)
		if err := os.MkdirAll(tasksDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create tasks directory: %w", err)
		}
	}

	// Check for file collision (another task already claims this file)
	existingTask, err := c.taskRepo.GetByFilePath(ctx, filePath)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check file collision: %w", err)
	}

	if existingTask != nil {
		// Another task already uses this file
		if !input.Force {
			return nil, fmt.Errorf(
				"file '%s' is already claimed by task %s ('%s'). Use --force to reassign",
				filePath, existingTask.Key, existingTask.Title,
			)
		}

		// Force mode: clear file path from old task
		if err := c.taskRepo.UpdateFilePath(ctx, existingTask.Key, nil); err != nil {
			return nil, fmt.Errorf("failed to unassign file from %s: %w", existingTask.Key, err)
		}
	}

	// Convert dependencies to JSON
	var dependsOnJSON *string
	if len(validated.ValidatedDependencies) > 0 {
		depsBytes, err := json.Marshal(validated.ValidatedDependencies)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal dependencies: %w", err)
		}
		depsStr := string(depsBytes)
		dependsOnJSON = &depsStr
	}

	// Prepare description
	var description *string
	if input.Description != "" {
		description = &input.Description
	}

	// Prepare execution_order
	var executionOrder *int
	if input.ExecutionOrder > 0 {
		executionOrder = &input.ExecutionOrder
	}

	// Determine initial status from workflow config
	initialStatus := c.getInitialTaskStatus()

	// Create task record
	task := &models.Task{
		FeatureID:      validated.FeatureID,
		Key:            key,
		Title:          input.Title,
		Description:    description,
		Status:         initialStatus,
		AgentType:      &validated.AgentType,
		Priority:       input.Priority,
		DependsOn:      dependsOnJSON,
		FilePath:       &filePath,
		ExecutionOrder: executionOrder,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// 5. Insert task into database
	err = c.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task in database: %w", err)
	}

	// 6. Create task history record
	agent := getCurrentUser()
	history := &models.TaskHistory{
		TaskID:    task.ID,
		OldStatus: nil,
		NewStatus: string(initialStatus),
		Agent:     &agent,
		Notes:     stringPtr("Task created"),
		Timestamp: now,
	}

	historyQuery := `
		INSERT INTO task_history (task_id, old_status, new_status, agent, notes, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = tx.ExecContext(ctx, historyQuery, history.TaskID, history.OldStatus, history.NewStatus, history.Agent, history.Notes, history.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to create history record: %w", err)
	}

	// 7. Render template with selection priority: custom > agent > general
	templateData := templates.TemplateData{
		Key:         key,
		Title:       input.Title,
		Description: input.Description,
		Epic:        input.EpicKey,
		Feature:     validated.NormalizedFeatureKey,
		AgentType:   validated.AgentType,
		Priority:    input.Priority,
		DependsOn:   validated.ValidatedDependencies,
		CreatedAt:   now,
	}

	markdown, err := c.renderer.RenderWithSelection(validated.AgentType, input.CustomTemplate, templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// 8. Write markdown file (only if it doesn't already exist)
	if !fileExists {
		err = c.writeFileExclusive(fullFilePath, []byte(markdown))
		if err != nil {
			return nil, fmt.Errorf("failed to write task file: %w", err)
		}
	}

	// 9. Commit transaction
	if err := tx.Commit(); err != nil {
		// Try to delete the file only if we created it
		if !fileExists {
			os.Remove(fullFilePath)
		}
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &CreateTaskResult{
		Task:     task,
		FilePath: filePath,
	}, nil
}

// writeFileExclusive writes a file only if it doesn't exist
func (c *Creator) writeFileExclusive(path string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file exclusively (fails if exists)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("file already exists: %s", path)
		}
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write data
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ValidateCustomFilename validates custom file paths for tasks, epics, and features.
// It enforces several security and naming constraints:
// - Filenames must be relative to the project root (no absolute paths)
// - Files must have a .md extension
// - Path traversal attempts (containing "..") are rejected
// - Resolved paths must stay within project boundaries
//
// Returns:
// - absPath: Absolute path for file system operations
// - relPath: Relative path for database storage (portable across systems)
// - error: Validation error, if any
//
// This function is shared across task, epic, and feature creation to ensure
// consistent filename validation across all entity types.
func ValidateCustomFilename(filename string, projectRoot string) (absPath string, relPath string, err error) {
	// 1. Reject absolute paths
	if filepath.IsAbs(filename) {
		return "", "", fmt.Errorf("filename must be relative to project root, got absolute path: %s", filename)
	}

	// 2. Clean the path (resolves ./ and normalizes separators)
	cleanPath := filepath.Clean(filename)

	// 3. Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", "", fmt.Errorf("invalid path: contains '..' (path traversal not allowed)")
	}

	// 4. Ensure path is within project boundaries
	fullPath := filepath.Join(projectRoot, cleanPath)
	if err := patterns.ValidatePathWithinProject(fullPath, projectRoot); err != nil {
		return "", "", fmt.Errorf("path validation failed: %w", err)
	}

	// 5. Validate file extension
	ext := filepath.Ext(cleanPath)
	if ext != ".md" {
		return "", "", fmt.Errorf("invalid file extension: %s (must be .md)", ext)
	}

	// 6. Ensure filename is not empty after cleaning
	base := filepath.Base(cleanPath)
	if base == "" || base == "." || base == ".." {
		return "", "", fmt.Errorf("invalid filename: resolved to empty or invalid path")
	}

	// 7. Convert to absolute path for file operations
	absPath, err = filepath.Abs(fullPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// 8. Store relative path for database (portable across systems)
	relPath = cleanPath

	return absPath, relPath, nil
}

// getCurrentUser returns the current user identifier
func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "system"
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// getInitialTaskStatus returns the initial status for new tasks from workflow config.
// It reads the first entry status from special_statuses._start_ in .sharkconfig.json.
// Falls back to TaskStatusTodo if workflow config is not found or doesn't define entry statuses.
func (c *Creator) getInitialTaskStatus() models.TaskStatus {
	// Load workflow config from project root
	configPath := filepath.Join(c.projectRoot, ".sharkconfig.json")
	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil || workflow == nil {
		// Config not found or failed to load - use default
		return models.TaskStatusTodo
	}

	// Get entry statuses from special_statuses._start_
	startStatuses, exists := workflow.SpecialStatuses[config.StartStatusKey]
	if !exists || len(startStatuses) == 0 {
		// No entry statuses defined - use default
		return models.TaskStatusTodo
	}

	// Return first entry status
	return models.TaskStatus(startStatuses[0])
}
