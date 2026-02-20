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
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/slug"
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
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
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
		ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC
	`

	return r.queryTasks(ctx, query, epicKey)
}

// ListBlockedTasksByEpic retrieves all blocked tasks for an epic.
// This method is more efficient than loading all tasks and filtering client-side.
func (r *TaskRepository) ListBlockedTasksByEpic(ctx context.Context, epicKey string) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason, t.execution_order,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at,
		       t.completed_by, t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data
		FROM tasks t
		INNER JOIN features f ON t.feature_id = f.id
		INNER JOIN epics e ON f.epic_id = e.id
		WHERE e.key = ? AND t.status = ?
		ORDER BY t.blocked_at DESC NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC
	`

	return r.queryTasks(ctx, query, epicKey, models.TaskStatus("blocked"))
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
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`

	return r.queryTasks(ctx, query, status)
}

// FilterByAgentType retrieves tasks filtered by agent type
func (r *TaskRepository) FilterByAgentType(ctx context.Context, agentType string) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE agent_type = ?
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`

	return r.queryTasks(ctx, query, agentType)
}

// FilterCombined retrieves tasks with multiple filter criteria
func (r *TaskRepository) FilterCombined(ctx context.Context, status *models.TaskStatus, epicKey *string, agentType *string, maxPriority *int) ([]*models.Task, error) {
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

	query += " ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC"

	tasks, err := r.queryTasks(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// ListByKeyPrefix retrieves all tasks whose key starts with the given prefix.
// Used by key generation to find globally unique keys regardless of feature_id.
func (r *TaskRepository) ListByKeyPrefix(ctx context.Context, prefix string) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE key LIKE ?
		ORDER BY key ASC
	`

	return r.queryTasks(ctx, query, prefix+"%")
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
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`

	tasks, err := r.queryTasks(ctx, query)
	if err != nil {
		return nil, err
	}

	return tasks, nil
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
		models.TaskStatus("todo"),
		models.TaskStatus("in_progress"),
		models.TaskStatus("blocked"),
		models.TaskStatus("ready_for_review"),
		models.TaskStatus("completed"),
		models.TaskStatus("archived"),
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
	// For backward transitions, use notes as rejection reason if provided
	// This maintains API compatibility while supporting the rejection reason requirement
	return r.UpdateStatusForced(ctx, taskID, newStatus, agent, notes, notes, nil, false)
}

// UpdateStatusForced atomically updates task status with optional validation bypass
func (r *TaskRepository) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
	_, err := r.updateStatusForcedInternal(ctx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
	return err
}

// UpdateStatusForcedWithUnblock atomically updates task status and, when the new
// status is completed or archived, auto-unblocks dependent tasks whose dependencies
// are all satisfied. Returns the keys of auto-unblocked tasks.
func (r *TaskRepository) UpdateStatusForcedWithUnblock(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error) {
	return r.updateStatusForcedInternal(ctx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
}

// updateStatusForcedInternal is the shared implementation for UpdateStatusForced
// and UpdateStatusForcedWithUnblock. It performs the status update and auto-unblock
// in a single transaction, returning any auto-unblocked task keys.
func (r *TaskRepository) updateStatusForcedInternal(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error) {
	// Validate status is valid enum (skip when force=true to allow any status string)
	if !force && !r.isValidStatusEnum(newStatus) {
		return nil, fmt.Errorf("invalid status: %s", newStatus)
	}
	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current task state
	var currentStatus string
	var taskKey string
	var startedAt, completedAt, blockedAt sql.NullTime
	err = tx.QueryRowContext(ctx, "SELECT key, status, started_at, completed_at, blocked_at FROM tasks WHERE id = ?", taskID).
		Scan(&taskKey, &currentStatus, &startedAt, &completedAt, &blockedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get current task status: %w", err)
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
					return nil, validationErr
				}
			}
			return nil, fmt.Errorf("invalid status transition from %s to %s", currentStatus, newStatus)
		}

		// Validate rejection reason for backward transitions
		if r.workflow != nil {
			isBackward, err := r.workflow.IsBackwardTransition(currentStatus, string(newStatus))
			if err != nil {
				return nil, fmt.Errorf("failed to determine transition direction: %w", err)
			}

			if isBackward {
				// Backward transitions require a non-empty reason
				if rejectionReason == nil || strings.TrimSpace(*rejectionReason) == "" {
					return nil, fmt.Errorf("rejection reason required for backward transition from %s to %s: use --reason flag or use --force to bypass", currentStatus, newStatus)
				}
			}
		}
	}

	// Update status and timestamps
	now := time.Now()
	query := "UPDATE tasks SET status = ?"
	args := []interface{}{newStatus}

	// Set appropriate timestamp based on new status
	if newStatus == models.TaskStatus("in_progress") && !startedAt.Valid {
		query += ", started_at = ?"
		args = append(args, now)
	} else if newStatus == models.TaskStatus("completed") && !completedAt.Valid {
		query += ", completed_at = ?"
		args = append(args, now)
	} else if newStatus == models.TaskStatus("blocked") && !blockedAt.Valid {
		query += ", blocked_at = ?"
		args = append(args, now)
	}

	query += " WHERE id = ?"
	args = append(args, taskID)

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Create history record with rejection reason support
	historyQuery := `
		INSERT INTO task_history (task_id, old_status, new_status, agent, notes, rejection_reason, forced)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, historyQuery, taskID, currentStatus, newStatus, agent, notes, rejectionReason, force)
	if err != nil {
		return nil, fmt.Errorf("failed to create history record: %w", err)
	}

	historyID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get history record id: %w", err)
	}

	// Create rejection note if rejection reason is provided and transition is backward
	if rejectionReason != nil && strings.TrimSpace(*rejectionReason) != "" {
		isBackward := false
		if r.workflow != nil {
			var checkErr error
			isBackward, checkErr = r.workflow.IsBackwardTransition(currentStatus, string(newStatus))
			if checkErr != nil {
				// Log but don't fail - the transition already succeeded
				fmt.Printf("WARNING: Failed to check backward transition for rejection note: %v\n", checkErr)
			}
		}

		if isBackward {
			noteRepo := NewEntityNoteRepository(r.db)
			rejectedBy := "system"
			if agent != nil && *agent != "" {
				rejectedBy = *agent
			}

			_, err := noteRepo.CreateRejectionNoteWithTx(
				ctx, tx, models.EntityTypeTask, taskID, historyID,
				currentStatus, string(newStatus),
				*rejectionReason, rejectedBy, documentPath,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create rejection note: %w", err)
			}
		}
	}

	// Auto-unblock dependents when transitioning to completed or archived
	var unblockedKeys []string
	if newStatus == models.TaskStatus("completed") || newStatus == models.TaskStatus("archived") {
		unblockedKeys, err = r.AutoUnblockDependents(ctx, tx, taskKey)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-unblock dependents: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return unblockedKeys, nil
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

	// Populate template with task placeholders
	instruction := metadata.OrchestratorAction.PopulateTemplate(config.TaskPlaceholders(task))

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
		if !r.isValidTransition(currentTaskStatus, models.TaskStatus("blocked")) {
			if r.workflow != nil {
				validationErr := config.ValidateTransition(r.workflow, string(currentTaskStatus), string(models.TaskStatus("blocked")))
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
	_, err = tx.ExecContext(ctx, query, models.TaskStatus("blocked"), now, reason, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record with rejection_reason support
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, rejection_reason, forced) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, models.TaskStatus("blocked"), agent, &reason, nil, force)
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
		if !r.isValidTransition(currentTaskStatus, models.TaskStatus("todo")) {
			if r.workflow != nil {
				validationErr := config.ValidateTransition(r.workflow, string(currentTaskStatus), string(models.TaskStatus("todo")))
				if validationErr != nil {
					return validationErr
				}
			}
			return fmt.Errorf("invalid status transition from %s to todo", currentStatus)
		}
	}

	// Update status and clear blocked fields
	query := `UPDATE tasks SET status = ?, blocked_at = NULL, blocked_reason = NULL WHERE id = ?`
	_, err = tx.ExecContext(ctx, query, models.TaskStatus("todo"), taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, rejection_reason, forced) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, models.TaskStatus("todo"), agent, nil, nil, force)
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
	// For backward compatibility, treat notes as rejection reason for the reopen
	// If no notes provided, use a default message or require force
	if notes == nil || strings.TrimSpace(*notes) == "" {
		// Default rejection reason when reopening without explicit notes
		defaultReason := "Task returned for rework"
		notes = &defaultReason
	}
	return r.ReopenTaskForced(ctx, taskID, agent, notes, notes, nil, false)
}

// ReopenTaskForced reopens a task with optional validation bypass
// Use rejectionReason for backward transitions to capture why task was rejected
func (r *TaskRepository) ReopenTaskForced(ctx context.Context, taskID int64, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
	return r.UpdateStatusForced(ctx, taskID, models.TaskStatus("in_progress"), agent, notes, rejectionReason, documentPath, force)
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

// GetStatusBreakdownMapBatch returns status counts as maps for multiple features in a single query.
// This method avoids N+1 query problems when fetching breakdowns for many features.
func (r *TaskRepository) GetStatusBreakdownMapBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error) {
	if len(featureIDs) == 0 {
		return make(map[int64]map[models.TaskStatus]int), nil
	}

	// Build the query with IN clause
	placeholders := make([]string, len(featureIDs))
	args := make([]interface{}, len(featureIDs))
	for i, id := range featureIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT feature_id, status, COUNT(*) as count
		FROM tasks
		WHERE feature_id IN (%s)
		GROUP BY feature_id, status
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch status breakdown: %w", err)
	}
	defer rows.Close()

	// Initialize result map with empty maps for each feature
	result := make(map[int64]map[models.TaskStatus]int)
	for _, featureID := range featureIDs {
		result[featureID] = make(map[models.TaskStatus]int)
	}

	// Fill in actual counts from query
	for rows.Next() {
		var featureID int64
		var status models.TaskStatus
		var count int

		if err := rows.Scan(&featureID, &status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan batch status breakdown: %w", err)
		}

		result[featureID][status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch status breakdown: %w", err)
	}

	return result, nil
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
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
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
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`, strings.Join(placeholders, ", "))

	return r.queryTasks(ctx, query, args...)
}

// GetRejectionCounts returns rejection counts and last rejection timestamps for given tasks
// Uses efficient LEFT JOIN with COUNT aggregation to avoid N+1 queries
// Returns two maps: taskID -> rejection count and taskID -> last rejection time
func (r *TaskRepository) GetRejectionCounts(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
	if len(taskIDs) == 0 {
		return make(map[int64]int), make(map[int64]*time.Time), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(taskIDs))
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT
			t.id,
			COALESCE(COUNT(tn.id), 0) as rejection_count,
			datetime(MAX(tn.created_at)) as last_rejection_at
		FROM tasks t
		LEFT JOIN entity_notes tn ON t.id = tn.entity_id AND tn.entity_type = 'task' AND tn.note_type = 'rejection'
		WHERE t.id IN (` + strings.Join(placeholders, ",") + `)
		GROUP BY t.id
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query rejection counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[int64]int)
	lastTimes := make(map[int64]*time.Time)

	for rows.Next() {
		var taskID int64
		var rejectionCount int
		var lastRejectionAtStr sql.NullString

		if err := rows.Scan(&taskID, &rejectionCount, &lastRejectionAtStr); err != nil {
			return nil, nil, fmt.Errorf("failed to scan rejection counts: %w", err)
		}

		counts[taskID] = rejectionCount
		if lastRejectionAtStr.Valid {
			// Parse the timestamp string
			parsedTime, err := time.Parse("2006-01-02 15:04:05", lastRejectionAtStr.String)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse last_rejection_at: %w", err)
			}
			lastTimes[taskID] = &parsedTime
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating rejection counts: %w", err)
	}

	// Ensure all requested task IDs are in the result maps (even if 0 rejections)
	for _, taskID := range taskIDs {
		if _, ok := counts[taskID]; !ok {
			counts[taskID] = 0
		}
	}

	return counts, lastTimes, nil
}

// GetTaskCountsForFeatures returns the total task count for each of the given feature IDs
// in a single batch query. This replaces N individual GetTaskCount calls with one query.
//
// Returns a map from featureID to count. Feature IDs with no tasks are not included in
// the map; callers should treat a missing key as a count of zero.
func (r *TaskRepository) GetTaskCountsForFeatures(ctx context.Context, featureIDs []int64) (map[int64]int, error) {
	if len(featureIDs) == 0 {
		return map[int64]int{}, nil
	}

	// Build parameterised IN clause
	placeholders := make([]string, len(featureIDs))
	args := make([]interface{}, len(featureIDs))
	for i, id := range featureIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT feature_id, COUNT(*) FROM tasks WHERE feature_id IN (%s) GROUP BY feature_id`,
		strings.Join(placeholders, ", "),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get task counts for features: %w", err)
	}
	defer rows.Close()

	counts := make(map[int64]int, len(featureIDs))
	for rows.Next() {
		var featureID int64
		var count int
		if err := rows.Scan(&featureID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan task count row: %w", err)
		}
		counts[featureID] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task counts: %w", err)
	}

	return counts, nil
}
