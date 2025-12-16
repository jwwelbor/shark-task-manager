package repository

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskHistoryRepository handles CRUD operations for task history
type TaskHistoryRepository struct {
	db *DB
}

// NewTaskHistoryRepository creates a new TaskHistoryRepository
func NewTaskHistoryRepository(db *DB) *TaskHistoryRepository {
	return &TaskHistoryRepository{db: db}
}

// Create creates a new task history record
func (r *TaskHistoryRepository) Create(ctx context.Context, history *models.TaskHistory) error {
	if err := history.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		history.TaskID,
		history.OldStatus,
		history.NewStatus,
		history.Agent,
		history.Notes,
	)
	if err != nil {
		return fmt.Errorf("failed to create task history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	history.ID = id
	return nil
}

// ListByTask retrieves all history records for a task
func (r *TaskHistoryRepository) ListByTask(ctx context.Context, taskID int64) ([]*models.TaskHistory, error) {
	query := `
		SELECT id, task_id, old_status, new_status, agent, notes, timestamp
		FROM task_history
		WHERE task_id = ?
		ORDER BY timestamp DESC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list task history: %w", err)
	}
	defer rows.Close()

	var histories []*models.TaskHistory
	for rows.Next() {
		history := &models.TaskHistory{}
		err := rows.Scan(
			&history.ID,
			&history.TaskID,
			&history.OldStatus,
			&history.NewStatus,
			&history.Agent,
			&history.Notes,
			&history.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task history: %w", err)
		}
		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task history: %w", err)
	}

	return histories, nil
}

// ListRecent retrieves recent history records across all tasks
func (r *TaskHistoryRepository) ListRecent(ctx context.Context, limit int) ([]*models.TaskHistory, error) {
	query := `
		SELECT id, task_id, old_status, new_status, agent, notes, timestamp
		FROM task_history
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent history: %w", err)
	}
	defer rows.Close()

	var histories []*models.TaskHistory
	for rows.Next() {
		history := &models.TaskHistory{}
		err := rows.Scan(
			&history.ID,
			&history.TaskID,
			&history.OldStatus,
			&history.NewStatus,
			&history.Agent,
			&history.Notes,
			&history.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task history: %w", err)
		}
		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task history: %w", err)
	}

	return histories, nil
}
