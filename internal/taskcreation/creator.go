package taskcreation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/fileops"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/pathresolver"
	"github.com/jwwelbor/shark-task-manager/internal/patterns"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// Creator orchestrates the complete task creation workflow
type Creator struct {
	db              *repository.DB
	keygen          *KeyGenerator
	validator       *Validator
	renderer        *templates.Renderer
	taskRepo        *repository.TaskRepository
	historyRepo     *repository.TaskHistoryRepository //nolint:staticcheck // Deprecated: will migrate to EntityHistoryRepository
	epicRepo        *repository.EpicRepository
	featureRepo     *repository.FeatureRepository
	pathResolver    *pathresolver.PathResolver
	projectRoot     string
	workflowService *workflow.Service
	verbose         bool
}

// NewCreator creates a new task creator.
// The workflowService is optional - if nil, it will be created using projectRoot.
func NewCreator(
	db *repository.DB,
	keygen *KeyGenerator,
	validator *Validator,
	renderer *templates.Renderer,
	taskRepo *repository.TaskRepository,
	historyRepo *repository.TaskHistoryRepository, //nolint:staticcheck // Deprecated: will migrate to EntityHistoryRepository
	epicRepo *repository.EpicRepository,
	featureRepo *repository.FeatureRepository,
	projectRoot string,
	workflowService *workflow.Service,
) *Creator {
	// Create workflow service if not provided
	if workflowService == nil {
		workflowService = workflow.NewService(projectRoot)
	}

	return &Creator{
		db:              db,
		keygen:          keygen,
		validator:       validator,
		renderer:        renderer,
		taskRepo:        taskRepo,
		historyRepo:     historyRepo,
		epicRepo:        epicRepo,
		featureRepo:     featureRepo,
		pathResolver:    pathresolver.NewPathResolver(epicRepo, featureRepo, nil, projectRoot),
		projectRoot:     projectRoot,
		workflowService: workflowService,
		verbose:         false,
	}
}

// SetVerbose enables or disables verbose logging
func (c *Creator) SetVerbose(verbose bool) {
	c.verbose = verbose
}

// CreateTaskInput holds the input for creating a task
type CreateTaskInput struct {
	EpicKey        string
	FeatureKey     string
	Title          string
	Description    string
	AgentType      string
	Priority       int
	DependsOn      string
	ExecutionOrder int
	CustomKey      string // Custom key override (optional)
	Filename       string // Custom filename path (relative to project root)
	Force          bool   // Force reassignment if file already claimed
	Create         bool   // Create file if it doesn't exist (when Filename is specified)
	Size           *int   // Optional size value (nil = unset)
	// Body, when non-empty, replaces the rendered placeholder body of the
	// task's markdown file (frontmatter is preserved). Empty string means
	// "use the rendered template body as-is".
	Body string
}

// CreateTaskResult holds the result of task creation
type CreateTaskResult struct {
	Task          *models.Task
	FilePath      string
	FileWasLinked bool // True if file existed and was linked, false if new file was created
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

	// 2. Begin database transaction before key allocation and writes.
	tx, err := c.db.BeginTxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 3. Generate or use custom task key within the transaction boundary.
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
		key, err = c.keygen.GenerateTaskKeyWithTx(ctx, tx, input.EpicKey, validated.NormalizedFeatureKey)
		if err != nil {
			return nil, err
		}
	}

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
		} else if os.IsNotExist(statErr) {
			// File doesn't exist - check if Create flag is set
			if !input.Create {
				return nil, fmt.Errorf("file '%s' does not exist. Use --create flag to create it", relPath)
			}
			// File doesn't exist but Create=true, so we'll create it later
			fileExists = false
		} else {
			// Other stat error (permission denied, etc.)
			return nil, fmt.Errorf("failed to check file status: %w", statErr)
		}
	} else {
		// Default: derive task path from feature's stored location via PathResolver
		featureBaseDir, err := c.pathResolver.ResolveTaskBaseDir(ctx, validated.NormalizedFeatureKey)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve task directory: %w", err)
		}

		// Task path: {featureBaseDir}/tasks/{task-key}.md
		taskFilename := key + ".md"
		filePath = filepath.Join(featureBaseDir, "tasks", taskFilename)
		fullFilePath = filepath.Join(c.projectRoot, filePath)

		// Create tasks directory if it doesn't exist (creates all parents)
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
		if err := c.taskRepo.UpdateFilePathWithTx(ctx, tx, existingTask.Key, nil); err != nil {
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

	// Determine initial status from workflow config via WorkflowService
	initialStatus := c.workflowService.GetInitialStatus()

	// Create task record
	task := &models.Task{BaseEntity: models.BaseEntity{
		Key:         key,
		Title:       input.Title,
		Description: description,
		FilePath:    &filePath,
		Size:        input.Size,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, FeatureID: validated.FeatureID,
		Status:         initialStatus,
		AgentType:      &validated.AgentType,
		Priority:       input.Priority,
		DependsOn:      dependsOnJSON,
		ExecutionOrder: executionOrder,
	}

	// 5. Insert task into database
	err = c.taskRepo.CreateWithTx(ctx, tx, task)
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

	markdown, err := c.renderer.Render(validated.AgentType, templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Apply caller-supplied body override (e.g. --content / piped stdin).
	if input.Body != "" {
		markdown = fileops.ReplaceBodyAfterFrontmatter(markdown, input.Body)
	}

	// 8. Write markdown file using unified file writer
	var writeResult *fileops.WriteResult
	if !fileExists {
		writer := fileops.NewEntityFileWriter()
		logFunc := func(msg string) {
			if c.verbose {
				slog.Debug("task-creator", "msg", msg)
			}
		}

		// Determine if we should create missing file:
		// - Always true for default paths (no custom filename)
		// - Respects input.Create flag for custom filenames
		createIfMissing := input.Filename == "" || input.Create

		writeResult, err = writer.WriteEntityFile(fileops.WriteOptions{
			Content:         []byte(markdown),
			ProjectRoot:     c.projectRoot,
			FilePath:        filePath,
			Verbose:         c.verbose,
			EntityType:      "task",
			UseAtomicWrite:  true,
			CreateIfMissing: createIfMissing,
			Logger:          logFunc,
		})
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

	// Determine if file was linked or created
	fileWasLinked := fileExists || (writeResult != nil && writeResult.Linked)

	return &CreateTaskResult{
		Task:          task,
		FilePath:      filePath,
		FileWasLinked: fileWasLinked,
	}, nil
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

// NOTE: getInitialTaskStatus has been removed and replaced with WorkflowService.GetInitialStatus()
// See T-E07-F16-012 for details on the refactoring.
