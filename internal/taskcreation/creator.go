package taskcreation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
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
}

// NewCreator creates a new task creator
func NewCreator(
	db *repository.DB,
	keygen *KeyGenerator,
	validator *Validator,
	renderer *templates.Renderer,
	taskRepo *repository.TaskRepository,
	historyRepo *repository.TaskHistoryRepository,
) *Creator {
	return &Creator{
		db:          db,
		keygen:      keygen,
		validator:   validator,
		renderer:    renderer,
		taskRepo:    taskRepo,
		historyRepo: historyRepo,
	}
}

// CreateTaskInput holds the input for creating a task
type CreateTaskInput struct {
	EpicKey     string
	FeatureKey  string
	Title       string
	Description string
	AgentType   string
	Priority    int
	DependsOn   string
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

	// 2. Generate task key
	key, err := c.keygen.GenerateTaskKey(ctx, input.EpicKey, validated.NormalizedFeatureKey)
	if err != nil {
		return nil, err
	}

	// 3. Begin database transaction
	tx, err := c.db.BeginTxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 4. Prepare task data
	now := time.Now().UTC()
	filePath := filepath.Join("docs", "tasks", "todo", fmt.Sprintf("%s.md", key))

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

	// Create task record
	task := &models.Task{
		FeatureID:   validated.FeatureID,
		Key:         key,
		Title:       input.Title,
		Description: description,
		Status:      models.TaskStatusTodo,
		AgentType:   &validated.AgentType,
		Priority:    input.Priority,
		DependsOn:   dependsOnJSON,
		FilePath:    &filePath,
		CreatedAt:   now,
		UpdatedAt:   now,
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
		NewStatus: string(models.TaskStatusTodo),
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

	// 7. Render template
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

	// 8. Write markdown file
	fullFilePath := filepath.Join("/", "home", "jwwelbor", "projects", "shark-task-manager", filePath)
	err = c.writeFileExclusive(fullFilePath, []byte(markdown))
	if err != nil {
		return nil, fmt.Errorf("failed to write task file: %w", err)
	}

	// 9. Commit transaction
	if err := tx.Commit(); err != nil {
		// Try to delete the file if commit fails
		os.Remove(fullFilePath)
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
