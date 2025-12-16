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
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskRepository handles CRUD operations for tasks
type TaskRepository struct {
	db *DB
}

// NewTaskRepository creates a new TaskRepository
func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create creates a new task
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO tasks (
			feature_id, key, title, description, status, agent_type, priority,
			depends_on, assigned_agent, file_path, blocked_reason
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
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
		SELECT id, feature_id, key, title, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason,
		       created_at, started_at, completed_at, blocked_at, updated_at
		FROM tasks
		WHERE id = ?
	`

	task := &models.Task{}
	err := r.db.QueryRow(query, id).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// GetByKey retrieves a task by its key
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason,
		       created_at, started_at, completed_at, blocked_at, updated_at
		FROM tasks
		WHERE key = ?
	`

	task := &models.Task{}
	err := r.db.QueryRow(query, key).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with key %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// ListByFeature retrieves all tasks for a feature
func (r *TaskRepository) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason,
		       created_at, started_at, completed_at, blocked_at, updated_at
		FROM tasks
		WHERE feature_id = ?
		ORDER BY priority ASC, created_at ASC
	`

	return r.queryTasks(ctx, query, featureID)
}

// ListByEpic retrieves all tasks for an epic (via features)
func (r *TaskRepository) ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at
		FROM tasks t
		INNER JOIN features f ON t.feature_id = f.id
		INNER JOIN epics e ON f.epic_id = e.id
		WHERE e.key = ?
		ORDER BY t.priority ASC, t.created_at ASC
	`

	return r.queryTasks(ctx, query, epicKey)
}

// FilterByStatus retrieves tasks filtered by status
func (r *TaskRepository) FilterByStatus(ctx context.Context, status models.TaskStatus) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason,
		       created_at, started_at, completed_at, blocked_at, updated_at
		FROM tasks
		WHERE status = ?
		ORDER BY priority ASC, created_at ASC
	`

	return r.queryTasks(ctx, query, status)
}

// FilterByAgentType retrieves tasks filtered by agent type
func (r *TaskRepository) FilterByAgentType(ctx context.Context, agentType models.AgentType) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason,
		       created_at, started_at, completed_at, blocked_at, updated_at
		FROM tasks
		WHERE agent_type = ?
		ORDER BY priority ASC, created_at ASC
	`

	return r.queryTasks(ctx, query, agentType)
}

// FilterCombined retrieves tasks with multiple filter criteria
func (r *TaskRepository) FilterCombined(ctx context.Context, status *models.TaskStatus, epicKey *string, agentType *models.AgentType, maxPriority *int) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at
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

	query += " ORDER BY t.priority ASC, t.created_at ASC"

	return r.queryTasks(ctx, query, args...)
}

// List retrieves all tasks
func (r *TaskRepository) List(ctx context.Context) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason,
		       created_at, started_at, completed_at, blocked_at, updated_at
		FROM tasks
		ORDER BY priority ASC, created_at ASC
	`

	return r.queryTasks(ctx, query)
}

// Update updates an existing task
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, agent_type = ?, priority = ?,
		    depends_on = ?, assigned_agent = ?, file_path = ?, blocked_reason = ?
		WHERE id = ?
	`

	result, err := r.db.Exec(query,
		task.Title,
		task.Description,
		task.Status,
		task.AgentType,
		task.Priority,
		task.DependsOn,
		task.AssignedAgent,
		task.FilePath,
		task.BlockedReason,
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

	return nil
}

// UpdateStatus atomically updates task status, timestamps, and creates history record
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
	// Start transaction
	tx, err := r.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current task state
	var currentStatus string
	var startedAt, completedAt, blockedAt sql.NullTime
	err = tx.QueryRow("SELECT status, started_at, completed_at, blocked_at FROM tasks WHERE id = ?", taskID).
		Scan(&currentStatus, &startedAt, &completedAt, &blockedAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
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

	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Create history record
	historyQuery := `
		INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(historyQuery, taskID, currentStatus, newStatus, agent, notes)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BlockTask marks a task as blocked with a reason
func (r *TaskRepository) BlockTask(ctx context.Context, taskID int64, reason string, agent *string) error {
	// Start transaction
	tx, err := r.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current task state
	var currentStatus string
	err = tx.QueryRow("SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Update status, blocked_at, and blocked_reason
	now := time.Now()
	query := `UPDATE tasks SET status = ?, blocked_at = ?, blocked_reason = ? WHERE id = ?`
	_, err = tx.Exec(query, models.TaskStatusBlocked, now, reason, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes) VALUES (?, ?, ?, ?, ?)`
	_, err = tx.Exec(historyQuery, taskID, currentStatus, models.TaskStatusBlocked, agent, reason)
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
	// Start transaction
	tx, err := r.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current task state
	var currentStatus string
	err = tx.QueryRow("SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Update status and clear blocked fields
	query := `UPDATE tasks SET status = ?, blocked_at = NULL, blocked_reason = NULL WHERE id = ?`
	_, err = tx.Exec(query, models.TaskStatusTodo, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes) VALUES (?, ?, ?, ?, ?)`
	_, err = tx.Exec(historyQuery, taskID, currentStatus, models.TaskStatusTodo, agent, nil)
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
	// Start transaction
	tx, err := r.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current task state
	var currentStatus string
	err = tx.QueryRow("SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Update status and clear completed_at
	query := `UPDATE tasks SET status = ?, completed_at = NULL WHERE id = ?`
	_, err = tx.Exec(query, models.TaskStatusInProgress, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes) VALUES (?, ?, ?, ?, ?)`
	_, err = tx.Exec(historyQuery, taskID, currentStatus, models.TaskStatusInProgress, agent, notes)
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

	result, err := r.db.Exec(query, id)
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

// GetStatusBreakdown returns a count of tasks by status for a feature
func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE feature_id = ?
		GROUP BY status
	`

	rows, err := r.db.Query(query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status breakdown: %w", err)
	}
	defer rows.Close()

	// Initialize breakdown with all statuses set to 0
	breakdown := map[models.TaskStatus]int{
		models.TaskStatusTodo:          0,
		models.TaskStatusInProgress:    0,
		models.TaskStatusBlocked:       0,
		models.TaskStatusReadyForReview: 0,
		models.TaskStatusCompleted:     0,
		models.TaskStatusArchived:      0,
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

// queryTasks is a helper function to execute task queries
func (r *TaskRepository) queryTasks(ctx context.Context, query string, args ...interface{}) ([]*models.Task, error) {
	rows, err := r.db.Query(query, args...)
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
			&task.Description,
			&task.Status,
			&task.AgentType,
			&task.Priority,
			&task.DependsOn,
			&task.AssignedAgent,
			&task.FilePath,
			&task.BlockedReason,
			&task.CreatedAt,
			&task.StartedAt,
			&task.CompletedAt,
			&task.BlockedAt,
			&task.UpdatedAt,
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
