// Package repository provides data access layer with context support.
//
// All repository methods accept context.Context as the first parameter to support:
// - Request cancellation
// - Timeout management
// - Distributed tracing
// - Request-scoped values
//
// Callers should create contexts appropriately:
// - HTTP handlers: Use r.Context() from http.Request
// - CLI commands: Use context.WithTimeout(context.Background(), timeout)
// - Tests: Use context.Background() or context.WithTimeout()
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	task, err := repo.GetByID(ctx, taskID)
//	if err != nil {
//	    return err
//	}
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/slug"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TaskRepository handles CRUD operations for tasks
type TaskRepository struct {
	db       *DB
	workflow *config.WorkflowConfig
}

// NewTaskRepository creates a new TaskRepository with default workflow configuration
func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{
		db:       db,
		workflow: config.DefaultWorkflow(),
	}
}

// NewTaskRepositoryWithWorkflow creates a new TaskRepository with custom workflow configuration
func NewTaskRepositoryWithWorkflow(db *DB, workflow *config.WorkflowConfig) *TaskRepository {
	if workflow == nil {
		workflow = config.DefaultWorkflow()
	}
	return &TaskRepository{
		db:       db,
		workflow: workflow,
	}
}

// GetWorkflow returns the workflow configuration used by this repository
func (r *TaskRepository) GetWorkflow() *config.WorkflowConfig {
	return r.workflow
}

// Create creates a new task
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Validate dependencies before creating
	if err := r.ValidateTaskDependencies(ctx, task); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Generate slug from title if not already set
	if task.Slug == nil {
		generatedSlug := slug.Generate(task.Title)
		task.Slug = &generatedSlug
	}

	query := `
		INSERT INTO tasks (
			feature_id, key, title, slug, description, status, agent_type, priority,
			depends_on, assigned_agent, file_path, blocked_reason, execution_order
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		task.FeatureID,
		task.Key,
		task.Title,
		task.Slug,
		task.Description,
		task.Status,
		task.AgentType,
		task.Priority,
		task.DependsOn,
		task.AssignedAgent,
		task.FilePath,
		task.BlockedReason,
		task.ExecutionOrder,
	)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	task.ID = id
	return nil
}

// GetByID retrieves a task by its ID
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE id = ?
	`

	task := &models.Task{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Slug,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.ExecutionOrder,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
		&task.CompletedBy,
		&task.CompletionNotes,
		&task.FilesChanged,
		&task.TestsPassed,
		&task.VerificationStatus,
		&task.TimeSpentMinutes,
		&task.ContextData,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// GetByKey retrieves a task by its key, supporting both numeric and slugged formats.
// Supports two key formats:
// 1. Numeric: T-E04-F01-001
// 2. Slugged: T-E04-F01-001-task-name
//
// Lookup strategy:
// 1. Try exact match on the key column (handles legacy numeric keys)
// 2. If not found and key contains a slug suffix, parse and match numeric key + slug
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if key == "" {
		return nil, fmt.Errorf("task key cannot be empty")
	}

	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE key = ?
	`

	task := &models.Task{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Slug,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.ExecutionOrder,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
		&task.CompletedBy,
		&task.CompletionNotes,
		&task.FilesChanged,
		&task.TestsPassed,
		&task.VerificationStatus,
		&task.TimeSpentMinutes,
		&task.ContextData,
	)

	if err == nil {
		// Found by exact match on key column
		return task, nil
	}

	if err != sql.ErrNoRows {
		// Unexpected error
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Not found by exact match - try parsing as slugged key
	// Expected format: T-E##-F##-###-slug-text
	// Parse to extract numeric key and slug
	numericKey, slug, ok := parseSluggedKey(key)
	if !ok {
		// Cannot parse as slugged key, return not found
		return nil, fmt.Errorf("task not found with key %s", key)
	}

	// Try lookup by numeric key + slug match
	queryWithSlug := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE key = ? AND slug = ?
	`

	err = r.db.QueryRowContext(ctx, queryWithSlug, numericKey, slug).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Slug,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.ExecutionOrder,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
		&task.CompletedBy,
		&task.CompletionNotes,
		&task.FilesChanged,
		&task.TestsPassed,
		&task.VerificationStatus,
		&task.TimeSpentMinutes,
		&task.ContextData,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with key %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// parseSluggedKey parses a slugged task key into numeric key and slug components.
// Input format: T-E##-F##-###-slug-text
// Returns: numericKey (T-E##-F##-###), slug (slug-text), ok (true if valid format)
func parseSluggedKey(key string) (numericKey string, slug string, ok bool) {
	// Task key format: T-E##-F##-###
	// Minimum length: T-E1-F1-1 = 9 characters
	// With slug: T-E1-F1-1-slug = at least 14 characters
	if len(key) < 14 {
		return "", "", false
	}

	// Check prefix
	if !strings.HasPrefix(key, "T-") {
		return "", "", false
	}

	// Find the 4th hyphen which separates the numeric part from the slug
	// Format: T-E##-F##-###-slug
	//         ^  ^   ^   ^
	//         1  2   3   4
	hyphenCount := 0
	lastHyphenPos := -1

	for i, ch := range key {
		if ch == '-' {
			hyphenCount++
			if hyphenCount == 4 {
				lastHyphenPos = i
				break
			}
		}
	}

	if lastHyphenPos == -1 || lastHyphenPos >= len(key)-1 {
		// No 4th hyphen or nothing after it
		return "", "", false
	}

	numericKey = key[:lastHyphenPos]
	slug = key[lastHyphenPos+1:]

	// Validate numeric key format matches T-E##-F##-###
	// At minimum: T-E1-F1-1
	if len(numericKey) < 9 {
		return "", "", false
	}

	// Slug should be non-empty
	if slug == "" {
		return "", "", false
	}

	return numericKey, slug, true
}

// GetByFilePath retrieves a task by its file path
// Returns sql.ErrNoRows if no task found with that file path
func (r *TaskRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE file_path = ?
	`

	task := &models.Task{}
	err := r.db.QueryRowContext(ctx, query, filePath).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Slug,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.ExecutionOrder,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
		&task.CompletedBy,
		&task.CompletionNotes,
		&task.FilesChanged,
		&task.TestsPassed,
		&task.VerificationStatus,
		&task.TimeSpentMinutes,
		&task.ContextData,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task by file path: %w", err)
	}

	return task, nil
}

// UpdateFilePath updates the file_path for a task
// Pass nil to clear the file path
func (r *TaskRepository) UpdateFilePath(ctx context.Context, taskKey string, newFilePath *string) error {
	query := `
		UPDATE tasks
		SET file_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query, newFilePath, taskKey)
	if err != nil {
		return fmt.Errorf("failed to update file path: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", taskKey)
	}

	return nil
}

// ListByFeature retrieves all tasks for a feature
func (r *TaskRepository) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE feature_id = ?
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC
	`

	return r.queryTasks(ctx, query, featureID)
}

// ListByEpic retrieves all tasks for an epic (via features)
func (r *TaskRepository) ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason, t.execution_order,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at,
		       t.completed_by, t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data
		FROM tasks t
		INNER JOIN features f ON t.feature_id = f.id
		INNER JOIN epics e ON f.epic_id = e.id
		WHERE e.key = ?
		ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC
	`

	return r.queryTasks(ctx, query, epicKey)
}

// FilterByStatus retrieves tasks filtered by status
func (r *TaskRepository) FilterByStatus(ctx context.Context, status models.TaskStatus) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE status = ?
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC
	`

	return r.queryTasks(ctx, query, status)
}

// FilterByAgentType retrieves tasks filtered by agent type
func (r *TaskRepository) FilterByAgentType(ctx context.Context, agentType models.AgentType) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE agent_type = ?
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC
	`

	return r.queryTasks(ctx, query, agentType)
}

// FilterCombined retrieves tasks with multiple filter criteria
func (r *TaskRepository) FilterCombined(ctx context.Context, status *models.TaskStatus, epicKey *string, agentType *models.AgentType, maxPriority *int) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason, t.execution_order,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at,
		       t.completed_by, t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data
		FROM tasks t
	`

	args := []interface{}{}
	conditions := []string{}

	if epicKey != nil {
		query += `
		INNER JOIN features f ON t.feature_id = f.id
		INNER JOIN epics e ON f.epic_id = e.id
		`
		conditions = append(conditions, "e.key = ?")
		args = append(args, *epicKey)
	}

	if status != nil {
		conditions = append(conditions, "t.status = ?")
		args = append(args, *status)
	}

	if agentType != nil {
		conditions = append(conditions, "t.agent_type = ?")
		args = append(args, *agentType)
	}

	if maxPriority != nil {
		conditions = append(conditions, "t.priority <= ?")
		args = append(args, *maxPriority)
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC"

	return r.queryTasks(ctx, query, args...)
}

// List retrieves all tasks
func (r *TaskRepository) List(ctx context.Context) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC
	`

	return r.queryTasks(ctx, query)
}

// Update updates an existing task
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Validate dependencies before updating
	if err := r.ValidateTaskDependencies(ctx, task); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Check if execution_order is being changed - if so, cascade to other tasks
	var oldTask *models.Task
	var err error
	var needsCascade bool

	if task.ExecutionOrder != nil {
		oldTask, err = r.GetByID(ctx, task.ID)
		if err != nil {
			return fmt.Errorf("failed to get old task: %w", err)
		}

		// Check if order actually changed
		needsCascade = (oldTask.ExecutionOrder == nil) ||
			(oldTask.ExecutionOrder != nil && *oldTask.ExecutionOrder != *task.ExecutionOrder)
	}

	// Start transaction for cascade updates
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// If cascade is needed, get all tasks BEFORE updating, then resequence ALL tasks
	if needsCascade {
		// Get all tasks in the same feature (before any updates)
		allTasks, err := r.listByFeatureInTx(ctx, tx, task.FeatureID)
		if err != nil {
			return fmt.Errorf("failed to list tasks for cascade: %w", err)
		}

		// Convert to orderedItem format
		var items []orderedItem
		for _, t := range allTasks {
			items = append(items, orderedItem{
				ID:             t.ID,
				ExecutionOrder: t.ExecutionOrder,
			})
		}

		// Resequence
		resequenced := resequenceOrders(items, task.ID, task.ExecutionOrder)

		// Update ALL tasks with new orders
		updateQuery := "UPDATE tasks SET execution_order = ? WHERE id = ?"
		for _, item := range resequenced {
			_, err := tx.ExecContext(ctx, updateQuery, item.ExecutionOrder, item.ID)
			if err != nil {
				return fmt.Errorf("failed to cascade update order for task %d: %w", item.ID, err)
			}
		}

		// Now update the main task's other fields (execution_order already updated above)
		query := `
			UPDATE tasks
			SET title = ?, description = ?, status = ?, agent_type = ?, priority = ?,
			    depends_on = ?, assigned_agent = ?, file_path = ?, blocked_reason = ?, context_data = ?
			WHERE id = ?
		`

		result, err := tx.ExecContext(ctx, query,
			task.Title,
			task.Description,
			task.Status,
			task.AgentType,
			task.Priority,
			task.DependsOn,
			task.AssignedAgent,
			task.FilePath,
			task.BlockedReason,
			task.ContextData,
			task.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows == 0 {
			return fmt.Errorf("task not found with id %d", task.ID)
		}
	} else {
		// No cascade needed, just update the task normally
		query := `
			UPDATE tasks
			SET title = ?, description = ?, status = ?, agent_type = ?, priority = ?,
			    depends_on = ?, assigned_agent = ?, file_path = ?, blocked_reason = ?, execution_order = ?, context_data = ?
			WHERE id = ?
		`

		result, err := tx.ExecContext(ctx, query,
			task.Title,
			task.Description,
			task.Status,
			task.AgentType,
			task.Priority,
			task.DependsOn,
			task.AssignedAgent,
			task.FilePath,
			task.BlockedReason,
			task.ExecutionOrder,
			task.ContextData,
			task.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows == 0 {
			return fmt.Errorf("task not found with id %d", task.ID)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// listByFeatureInTx lists tasks by feature within a transaction
func (r *TaskRepository) listByFeatureInTx(ctx context.Context, tx *sql.Tx, featureID int64) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, updated_at, context_data
		FROM tasks
		WHERE feature_id = ?
		ORDER BY execution_order ASC
	`

	rows, err := tx.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID,
			&task.FeatureID,
			&task.Key,
			&task.Title,
			&task.Slug,
			&task.Description,
			&task.Status,
			&task.AgentType,
			&task.Priority,
			&task.DependsOn,
			&task.AssignedAgent,
			&task.FilePath,
			&task.BlockedReason,
			&task.ExecutionOrder,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.ContextData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// isValidStatusEnum checks if a status is valid according to the workflow configuration
func (r *TaskRepository) isValidStatusEnum(status models.TaskStatus) bool {
	// Check if status exists in workflow config
	if r.workflow != nil && r.workflow.StatusFlow != nil {
		_, exists := r.workflow.StatusFlow[string(status)]
		return exists
	}

	// Fallback to hardcoded statuses if no workflow config
	validStatuses := []models.TaskStatus{
		models.TaskStatusTodo,
		models.TaskStatusInProgress,
		models.TaskStatusBlocked,
		models.TaskStatusReadyForReview,
		models.TaskStatusCompleted,
		models.TaskStatusArchived,
	}
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// isValidTransition checks if a status transition is allowed according to workflow config.
// This method is now fully config-driven with no hardcoded fallback.
// If workflow config is missing, it uses the default workflow.
func (r *TaskRepository) isValidTransition(from models.TaskStatus, to models.TaskStatus) bool {
	// Workflow should always be initialized (either from config or default)
	if r.workflow == nil {
		// This should not happen as NewTaskRepository always sets workflow,
		// but use default workflow as safety fallback
		r.workflow = config.DefaultWorkflow()
	}

	// Validate transition using workflow config
	return config.ValidateTransition(r.workflow, string(from), string(to)) == nil
}

// UpdateStatus atomically updates task status, timestamps, and creates history record
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
	return r.UpdateStatusForced(ctx, taskID, newStatus, agent, notes, false)
}

// UpdateStatusForced atomically updates task status with optional validation bypass
func (r *TaskRepository) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, force bool) error {
	// Validate status is valid enum
	if !r.isValidStatusEnum(newStatus) {
		return fmt.Errorf("invalid status: %s", newStatus)
	}
	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current task state
	var currentStatus string
	var startedAt, completedAt, blockedAt sql.NullTime
	err = tx.QueryRowContext(ctx, "SELECT status, started_at, completed_at, blocked_at FROM tasks WHERE id = ?", taskID).
		Scan(&currentStatus, &startedAt, &completedAt, &blockedAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Validate transition if not forcing
	currentTaskStatus := models.TaskStatus(currentStatus)
	if force {
		// Log warning when force is used
		fmt.Printf("WARNING: Forced status update from %s to %s (taskID=%d)\n", currentStatus, newStatus, taskID)
	} else {
		// Check if transition is valid using workflow config
		if !r.isValidTransition(currentTaskStatus, newStatus) {
			// Generate helpful error message using workflow validator
			if r.workflow != nil {
				validationErr := config.ValidateTransition(r.workflow, string(currentTaskStatus), string(newStatus))
				if validationErr != nil {
					return validationErr
				}
			}
			return fmt.Errorf("invalid status transition from %s to %s", currentStatus, newStatus)
		}

		// Validate rejection reason for backward transitions
		if r.workflow != nil {
			isBackward, err := r.workflow.IsBackwardTransition(currentStatus, string(newStatus))
			if err != nil {
				return fmt.Errorf("failed to determine transition direction: %w", err)
			}

			if isBackward {
				// Backward transitions require a non-empty reason
				if notes == nil || strings.TrimSpace(*notes) == "" {
					return fmt.Errorf("rejection reason required for backward transition from %s to %s: use --reason flag or use --force to bypass", currentStatus, newStatus)
				}
			}
		}
	}

	// Update status and timestamps
	now := time.Now()
	query := "UPDATE tasks SET status = ?"
	args := []interface{}{newStatus}

	// Set appropriate timestamp based on new status
	if newStatus == models.TaskStatusInProgress && !startedAt.Valid {
		query += ", started_at = ?"
		args = append(args, now)
	} else if newStatus == models.TaskStatusCompleted && !completedAt.Valid {
		query += ", completed_at = ?"
		args = append(args, now)
	} else if newStatus == models.TaskStatusBlocked && !blockedAt.Valid {
		query += ", blocked_at = ?"
		args = append(args, now)
	}

	query += " WHERE id = ?"
	args = append(args, taskID)

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Create history record
	historyQuery := `
		INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, newStatus, agent, notes, force)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateStatusWithAction updates a task's status and returns the updated task with orchestrator action
// This method combines status update with retrieval of orchestrator action from workflow config
// Returns:
// - *models.Task: The updated task
// - *config.PopulatedAction: The orchestrator action for the new status (nil if not defined)
// - error: Any error that occurred during the update
func (r *TaskRepository) UpdateStatusWithAction(ctx context.Context, taskKey string, newStatus string) (*models.Task, *config.PopulatedAction, error) {
	// Get task by key
	task, err := r.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Update task status using existing method
	taskStatus := models.TaskStatus(newStatus)
	if err := r.UpdateStatus(ctx, task.ID, taskStatus, nil, nil); err != nil {
		return nil, nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Fetch updated task
	updatedTask, err := r.GetByID(ctx, task.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get updated task: %w", err)
	}

	// Get orchestrator action for new status from workflow config
	action, err := r.getOrchestratorAction(ctx, updatedTask, newStatus)
	if err != nil {
		// Log warning but don't fail - action is optional
		fmt.Printf("WARNING: Failed to get orchestrator action for status %s: %v\n", newStatus, err)
		action = nil
	}

	return updatedTask, action, nil
}

// getOrchestratorAction retrieves and populates orchestrator action for a status
// Returns nil if no action is defined for the status (not an error)
func (r *TaskRepository) getOrchestratorAction(ctx context.Context, task *models.Task, status string) (*config.PopulatedAction, error) {
	// Check if workflow is configured
	if r.workflow == nil || r.workflow.StatusMetadata == nil {
		return nil, nil // No workflow - no actions
	}

	// Get status metadata
	metadata, exists := r.workflow.StatusMetadata[status]
	if !exists {
		return nil, nil // Status not in config - no actions
	}

	// Check if action is defined
	if metadata.OrchestratorAction == nil {
		return nil, nil // No action for this status - OK
	}

	// Populate template with task ID
	instruction := metadata.OrchestratorAction.PopulateTemplate(task.Key)

	// Return populated action
	populatedAction := &config.PopulatedAction{
		Action:      metadata.OrchestratorAction.Action,
		AgentType:   metadata.OrchestratorAction.AgentType,
		Skills:      metadata.OrchestratorAction.Skills,
		Instruction: instruction,
	}

	return populatedAction, nil
}

// BlockTask marks a task as blocked with a reason
func (r *TaskRepository) BlockTask(ctx context.Context, taskID int64, reason string, agent *string) error {
	return r.BlockTaskForced(ctx, taskID, reason, agent, false)
}

// BlockTaskForced marks a task as blocked with optional validation bypass
func (r *TaskRepository) BlockTaskForced(ctx context.Context, taskID int64, reason string, agent *string, force bool) error {
	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current task state
	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Validate transition if not forcing
	currentTaskStatus := models.TaskStatus(currentStatus)
	if force {
		fmt.Printf("WARNING: Forced block from %s status (taskID=%d)\n", currentStatus, taskID)
	} else {
		// Validate transition using workflow config
		if !r.isValidTransition(currentTaskStatus, models.TaskStatusBlocked) {
			if r.workflow != nil {
				validationErr := config.ValidateTransition(r.workflow, string(currentTaskStatus), string(models.TaskStatusBlocked))
				if validationErr != nil {
					return validationErr
				}
			}
			return fmt.Errorf("invalid status transition from %s to blocked", currentStatus)
		}
	}

	// Update status, blocked_at, and blocked_reason
	now := time.Now()
	query := `UPDATE tasks SET status = ?, blocked_at = ?, blocked_reason = ? WHERE id = ?`
	_, err = tx.ExecContext(ctx, query, models.TaskStatusBlocked, now, reason, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, models.TaskStatusBlocked, agent, reason, force)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UnblockTask unblocks a task and returns it to todo status
func (r *TaskRepository) UnblockTask(ctx context.Context, taskID int64, agent *string) error {
	return r.UnblockTaskForced(ctx, taskID, agent, false)
}

// UnblockTaskForced unblocks a task with optional validation bypass
func (r *TaskRepository) UnblockTaskForced(ctx context.Context, taskID int64, agent *string, force bool) error {
	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current task state
	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Validate transition if not forcing
	currentTaskStatus := models.TaskStatus(currentStatus)
	if force {
		fmt.Printf("WARNING: Forced unblock from %s status (taskID=%d)\n", currentStatus, taskID)
	} else {
		// Validate transition using workflow config
		if !r.isValidTransition(currentTaskStatus, models.TaskStatusTodo) {
			if r.workflow != nil {
				validationErr := config.ValidateTransition(r.workflow, string(currentTaskStatus), string(models.TaskStatusTodo))
				if validationErr != nil {
					return validationErr
				}
			}
			return fmt.Errorf("invalid status transition from %s to todo", currentStatus)
		}
	}

	// Update status and clear blocked fields
	query := `UPDATE tasks SET status = ?, blocked_at = NULL, blocked_reason = NULL WHERE id = ?`
	_, err = tx.ExecContext(ctx, query, models.TaskStatusTodo, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, models.TaskStatusTodo, agent, nil, force)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ReopenTask reopens a task from ready_for_review back to in_progress
func (r *TaskRepository) ReopenTask(ctx context.Context, taskID int64, agent *string, notes *string) error {
	return r.ReopenTaskForced(ctx, taskID, agent, notes, false)
}

// ReopenTaskForced reopens a task with optional validation bypass
func (r *TaskRepository) ReopenTaskForced(ctx context.Context, taskID int64, agent *string, notes *string, force bool) error {
	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current task state
	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Validate transition if not forcing
	currentTaskStatus := models.TaskStatus(currentStatus)
	if force {
		fmt.Printf("WARNING: Forced reopen from %s status (taskID=%d)\n", currentStatus, taskID)
	} else {
		// Validate transition using workflow config
		if !r.isValidTransition(currentTaskStatus, models.TaskStatusInProgress) {
			if r.workflow != nil {
				validationErr := config.ValidateTransition(r.workflow, string(currentTaskStatus), string(models.TaskStatusInProgress))
				if validationErr != nil {
					return validationErr
				}
			}
			return fmt.Errorf("invalid status transition from %s to in_progress", currentStatus)
		}
	}

	// Update status and clear completed_at
	query := `UPDATE tasks SET status = ?, completed_at = NULL WHERE id = ?`
	_, err = tx.ExecContext(ctx, query, models.TaskStatusInProgress, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, models.TaskStatusInProgress, agent, notes, force)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete deletes a task (and its history via CASCADE)
func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM tasks WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found with id %d", id)
	}

	return nil
}

// GetStatusBreakdown returns a count of tasks by status for a feature.
// Returns workflow.StatusCount slice ordered by workflow phase, with metadata.
// Only includes statuses that have non-zero counts.
func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) ([]workflow.StatusCount, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE feature_id = ?
		GROUP BY status
	`

	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status breakdown: %w", err)
	}
	defer rows.Close()

	// Build map of actual counts
	actualCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status breakdown: %w", err)
		}
		actualCounts[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating status breakdown: %w", err)
	}

	// Get all statuses from workflow config (in order)
	allStatuses := r.getOrderedStatuses()

	// Track which statuses we've already added
	addedStatuses := make(map[string]bool)

	// Build result with workflow metadata, only including non-zero counts
	var result []workflow.StatusCount
	for _, status := range allStatuses {
		count, exists := actualCounts[status]
		if !exists || count == 0 {
			continue // Skip zero counts
		}

		meta := r.getStatusMetadata(status)
		result = append(result, workflow.StatusCount{
			Status: status,
			Count:  count,
			Phase:  meta.Phase,
			Color:  meta.Color,
		})
		addedStatuses[status] = true
	}

	// Add any statuses from data that weren't in the workflow config (at the end)
	for status, count := range actualCounts {
		if addedStatuses[status] || count == 0 {
			continue
		}
		// These are statuses not in workflow config (e.g., legacy statuses)
		meta := r.getStatusMetadata(status)
		result = append(result, workflow.StatusCount{
			Status: status,
			Count:  count,
			Phase:  meta.Phase,
			Color:  meta.Color,
		})
	}

	return result, nil
}

// GetStatusBreakdownMap returns status counts as a map for backward compatibility.
// Deprecated: Use GetStatusBreakdown for workflow-aware status displays.
func (r *TaskRepository) GetStatusBreakdownMap(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE feature_id = ?
		GROUP BY status
	`

	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status breakdown: %w", err)
	}
	defer rows.Close()

	// Initialize breakdown with common statuses set to 0
	breakdown := map[models.TaskStatus]int{
		models.TaskStatusTodo:           0,
		models.TaskStatusInProgress:     0,
		models.TaskStatusBlocked:        0,
		models.TaskStatusReadyForReview: 0,
		models.TaskStatusCompleted:      0,
		models.TaskStatusArchived:       0,
	}

	// Fill in actual counts from query
	for rows.Next() {
		var status models.TaskStatus
		var count int
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status breakdown: %w", err)
		}
		breakdown[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating status breakdown: %w", err)
	}

	return breakdown, nil
}

// getOrderedStatuses returns all statuses in workflow order
func (r *TaskRepository) getOrderedStatuses() []string {
	if r.workflow == nil {
		// Fallback to default statuses
		return []string{
			"draft", "ready_for_refinement", "in_refinement", "ready_for_development",
			"in_development", "ready_for_code_review", "in_code_review", "ready_for_qa",
			"in_qa", "ready_for_approval", "in_approval", "completed", "cancelled", "blocked", "on_hold",
		}
	}

	// Group by phase and sort
	phases := []string{"planning", "development", "review", "qa", "approval", "done", "any"}
	statusesByPhase := make(map[string][]string)

	for status := range r.workflow.StatusFlow {
		meta, found := r.workflow.GetStatusMetadata(status)
		phase := "other"
		if found && meta.Phase != "" {
			phase = meta.Phase
		}
		statusesByPhase[phase] = append(statusesByPhase[phase], status)
	}

	// Sort within each phase
	for _, statuses := range statusesByPhase {
		sort.Strings(statuses)
	}

	// Concatenate in phase order
	var result []string
	for _, phase := range phases {
		result = append(result, statusesByPhase[phase]...)
	}
	if otherStatuses := statusesByPhase["other"]; len(otherStatuses) > 0 {
		result = append(result, otherStatuses...)
	}

	return result
}

// getStatusMetadata returns metadata for a status
func (r *TaskRepository) getStatusMetadata(status string) config.StatusMetadata {
	if r.workflow == nil {
		return config.StatusMetadata{}
	}
	meta, _ := r.workflow.GetStatusMetadata(status)
	return meta
}

// GetTaskCountForFeature returns the total number of tasks for a given feature
func (r *TaskRepository) GetTaskCountForFeature(ctx context.Context, featureID int64) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE feature_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get task count for feature: %w", err)
	}

	return count, nil
}

// BulkCreate creates multiple tasks in a single transaction
// Returns number of tasks created and error
func (r *TaskRepository) BulkCreate(ctx context.Context, tasks []*models.Task) (int, error) {
	if len(tasks) == 0 {
		return 0, nil
	}

	// Validate all tasks before inserting
	for i, task := range tasks {
		if err := task.Validate(); err != nil {
			return 0, fmt.Errorf("validation failed for task %d: %w", i, err)
		}
	}

	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Prepare statement for efficiency
	query := `
		INSERT INTO tasks (
			feature_id, key, title, description, status, agent_type, priority,
			depends_on, assigned_agent, file_path, blocked_reason
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert all tasks
	count := 0
	for _, task := range tasks {
		result, err := stmt.ExecContext(ctx,
			task.FeatureID,
			task.Key,
			task.Title,
			task.Description,
			task.Status,
			task.AgentType,
			task.Priority,
			task.DependsOn,
			task.AssignedAgent,
			task.FilePath,
			task.BlockedReason,
		)
		if err != nil {
			return count, fmt.Errorf("failed to insert task %s: %w", task.Key, err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return count, fmt.Errorf("failed to get last insert id for task %s: %w", task.Key, err)
		}

		task.ID = id
		count++
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return count, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

// GetByKeys retrieves multiple tasks by their keys
// Returns map of key -> task, missing keys are omitted
func (r *TaskRepository) GetByKeys(ctx context.Context, keys []string) (map[string]*models.Task, error) {
	if len(keys) == 0 {
		return make(map[string]*models.Task), nil
	}

	// Build dynamic IN clause
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE key IN (?` + strings.Repeat(", ?", len(keys)-1) + `)`

	// Convert keys to []interface{} for query
	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by keys: %w", err)
	}
	defer rows.Close()

	// Build result map
	result := make(map[string]*models.Task)
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID,
			&task.FeatureID,
			&task.Key,
			&task.Title,
			&task.Slug,
			&task.Description,
			&task.Status,
			&task.AgentType,
			&task.Priority,
			&task.DependsOn,
			&task.AssignedAgent,
			&task.FilePath,
			&task.BlockedReason,
			&task.ExecutionOrder,
			&task.CreatedAt,
			&task.StartedAt,
			&task.CompletedAt,
			&task.BlockedAt,
			&task.UpdatedAt,
			&task.CompletedBy,
			&task.CompletionNotes,
			&task.FilesChanged,
			&task.TestsPassed,
			&task.VerificationStatus,
			&task.TimeSpentMinutes,
			&task.ContextData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		result[task.Key] = task
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return result, nil
}

// UpdateMetadata updates only metadata fields (title, description, file_path)
// Does NOT update status, priority, agent_type (database-only fields)
func (r *TaskRepository) UpdateMetadata(ctx context.Context, task *models.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE tasks
		SET title = ?, description = ?, file_path = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.FilePath,
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task metadata: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found with id %d", task.ID)
	}

	return nil
}

// GetMaxSequenceForFeature gets the maximum task sequence number for a feature
// Returns 0 if no tasks exist for the feature
func (r *TaskRepository) GetMaxSequenceForFeature(ctx context.Context, featureKey string) (int, error) {
	// Task keys are in format: T-E##-F##-###
	// We need to extract the sequence number (###) from the key
	// Use SQL to parse the key and find the maximum sequence
	query := `
		SELECT COALESCE(MAX(CAST(SUBSTR(t.key, -3) AS INTEGER)), 0) as max_sequence
		FROM tasks t
		INNER JOIN features f ON t.feature_id = f.id
		WHERE f.key = ? AND t.key LIKE 'T-' || ? || '-%'
	`

	var maxSequence int
	err := r.db.QueryRowContext(ctx, query, featureKey, featureKey).Scan(&maxSequence)
	if err != nil {
		return 0, fmt.Errorf("failed to get max sequence for feature %s: %w", featureKey, err)
	}

	return maxSequence, nil
}

// UpdateKey updates the key of a task
func (r *TaskRepository) UpdateKey(ctx context.Context, oldKey string, newKey string) error {
	// Validate new key doesn't already exist
	existing, err := r.GetByKey(ctx, newKey)
	if err == nil && existing != nil {
		return fmt.Errorf("task with key %s already exists", newKey)
	}

	query := `
		UPDATE tasks
		SET key = ?, updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query, newKey, oldKey)
	if err != nil {
		return fmt.Errorf("update task key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found: %s", oldKey)
	}

	return nil
}

// queryTasks is a helper function to execute task queries
func (r *TaskRepository) queryTasks(ctx context.Context, query string, args ...interface{}) ([]*models.Task, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID,
			&task.FeatureID,
			&task.Key,
			&task.Title,
			&task.Slug,
			&task.Description,
			&task.Status,
			&task.AgentType,
			&task.Priority,
			&task.DependsOn,
			&task.AssignedAgent,
			&task.FilePath,
			&task.BlockedReason,
			&task.ExecutionOrder,
			&task.CreatedAt,
			&task.StartedAt,
			&task.CompletedAt,
			&task.BlockedAt,
			&task.UpdatedAt,
			&task.CompletedBy,
			&task.CompletionNotes,
			&task.FilesChanged,
			&task.TestsPassed,
			&task.VerificationStatus,
			&task.TimeSpentMinutes,
			&task.ContextData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// UpdateCompletionMetadata updates completion metadata for a task
func (r *TaskRepository) UpdateCompletionMetadata(ctx context.Context, taskKey string, metadata *models.CompletionMetadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Convert files_changed array to JSON
	filesJSON, err := metadata.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert files_changed to JSON: %w", err)
	}

	query := `
		UPDATE tasks
		SET completed_by = ?,
		    completion_notes = ?,
		    files_changed = ?,
		    tests_passed = ?,
		    verification_status = ?,
		    time_spent_minutes = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		metadata.CompletedBy,
		metadata.CompletionNotes,
		filesJSON,
		metadata.TestsPassed,
		metadata.VerificationStatus,
		metadata.TimeSpentMinutes,
		taskKey,
	)
	if err != nil {
		return fmt.Errorf("failed to update completion metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", taskKey)
	}

	return nil
}

// GetCompletionMetadata retrieves completion metadata for a task
func (r *TaskRepository) GetCompletionMetadata(ctx context.Context, taskKey string) (*models.CompletionMetadata, error) {
	query := `
		SELECT completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, completed_at
		FROM tasks
		WHERE key = ?
	`

	metadata := models.NewCompletionMetadata()
	var filesJSON *string
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, taskKey).Scan(
		&metadata.CompletedBy,
		&metadata.CompletionNotes,
		&filesJSON,
		&metadata.TestsPassed,
		&metadata.VerificationStatus,
		&metadata.TimeSpentMinutes,
		&completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with key %s", taskKey)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get completion metadata: %w", err)
	}

	// Parse files_changed JSON
	if filesJSON != nil && *filesJSON != "" {
		if err := metadata.FromJSON(*filesJSON); err != nil {
			return nil, fmt.Errorf("failed to parse files_changed JSON: %w", err)
		}
	}

	// Set completed_at if valid
	if completedAt.Valid {
		metadata.CompletedAt = &completedAt.Time
	}

	return metadata, nil
}

// FindByFileChanged searches for tasks that created or modified a specific file
func (r *TaskRepository) FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE files_changed IS NOT NULL
		  AND files_changed LIKE ?
		ORDER BY completed_at DESC NULLS LAST
	`

	// Use SQL LIKE pattern for partial matching
	pattern := "%" + filePath + "%"

	return r.queryTasks(ctx, query, pattern)
}

// GetUnverifiedTasks retrieves all tasks with verification_status != 'verified'
func (r *TaskRepository) GetUnverifiedTasks(ctx context.Context) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE verification_status != 'verified'
		  AND status IN ('ready_for_review', 'completed')
		ORDER BY completed_at DESC NULLS LAST
	`

	return r.queryTasks(ctx, query)
}

// FilterByMetadataAgentType retrieves tasks filtered by agent type from workflow metadata
// Uses status metadata to find statuses that include the specified agent type,
// then returns all tasks in those statuses
func (r *TaskRepository) FilterByMetadataAgentType(ctx context.Context, agentType string, workflow *config.WorkflowConfig) ([]*models.Task, error) {
	if workflow == nil {
		workflow = r.workflow
	}

	// Get statuses that include this agent type
	statuses := workflow.GetStatusesByAgentType(agentType)
	if len(statuses) == 0 {
		// No statuses match this agent type - return empty list
		return []*models.Task{}, nil
	}

	// Build SQL query with IN clause for multiple statuses
	placeholders := make([]string, len(statuses))
	args := make([]interface{}, len(statuses))
	for i, status := range statuses {
		placeholders[i] = "?"
		args[i] = status
	}

	query := fmt.Sprintf(`
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE status IN (%s)
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC
	`, strings.Join(placeholders, ", "))

	return r.queryTasks(ctx, query, args...)
}

// FilterByMetadataPhase retrieves tasks filtered by workflow phase from metadata
// Uses status metadata to find statuses in the specified phase,
// then returns all tasks in those statuses
func (r *TaskRepository) FilterByMetadataPhase(ctx context.Context, phase string, workflow *config.WorkflowConfig) ([]*models.Task, error) {
	if workflow == nil {
		workflow = r.workflow
	}

	// Get statuses in this phase
	statuses := workflow.GetStatusesByPhase(phase)
	if len(statuses) == 0 {
		// No statuses in this phase - return empty list
		return []*models.Task{}, nil
	}

	// Build SQL query with IN clause for multiple statuses
	placeholders := make([]string, len(statuses))
	args := make([]interface{}, len(statuses))
	for i, status := range statuses {
		placeholders[i] = "?"
		args[i] = status
	}

	query := fmt.Sprintf(`
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE status IN (%s)
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC
	`, strings.Join(placeholders, ", "))

	return r.queryTasks(ctx, query, args...)
}
