package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// HistoryFilters defines filters for querying task history
type HistoryFilters struct {
	Agent      *string    // Filter by agent ID
	Since      *time.Time // Filter by timestamp (>= since)
	EpicKey    *string    // Filter by epic key
	FeatureKey *string    // Filter by feature key
	OldStatus  *string    // Filter by old status
	NewStatus  *string    // Filter by new status
	Limit      int        // Maximum number of records to return (default 50)
	Offset     int        // Number of records to skip for pagination
}

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

// ListWithFilters retrieves history records with optional filters
func (r *TaskHistoryRepository) ListWithFilters(ctx context.Context, filters HistoryFilters) ([]*models.TaskHistory, error) {
	// Set default limit if not specified
	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	// Build query with filters
	query := `
		SELECT DISTINCT th.id, th.task_id, th.old_status, th.new_status, th.agent, th.notes, th.timestamp
		FROM task_history th
	`

	var joins []string
	var conditions []string
	var args []interface{}

	// Join with tasks if we need epic or feature filtering
	if filters.EpicKey != nil || filters.FeatureKey != nil {
		joins = append(joins, "INNER JOIN tasks t ON th.task_id = t.id")
	}

	// Join with features if we need epic filtering
	if filters.EpicKey != nil {
		joins = append(joins, "INNER JOIN features f ON t.feature_id = f.id")
	}

	// Add joins to query
	if len(joins) > 0 {
		query += "\n" + strings.Join(joins, "\n")
	}

	// Add WHERE conditions
	if filters.Agent != nil {
		conditions = append(conditions, "th.agent = ?")
		args = append(args, *filters.Agent)
	}

	if filters.Since != nil {
		conditions = append(conditions, "th.timestamp >= ?")
		args = append(args, *filters.Since)
	}

	if filters.EpicKey != nil {
		conditions = append(conditions, "f.key LIKE ?")
		args = append(args, *filters.EpicKey+"%")
	}

	if filters.FeatureKey != nil {
		conditions = append(conditions, "t.feature_id IN (SELECT id FROM features WHERE key = ?)")
		args = append(args, *filters.FeatureKey)
	}

	if filters.OldStatus != nil {
		conditions = append(conditions, "th.old_status = ?")
		args = append(args, *filters.OldStatus)
	}

	if filters.NewStatus != nil {
		conditions = append(conditions, "th.new_status = ?")
		args = append(args, *filters.NewStatus)
	}

	// Add WHERE clause if we have conditions
	if len(conditions) > 0 {
		query += "\nWHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering, limit, and offset
	query += "\nORDER BY th.timestamp DESC"
	query += "\nLIMIT ? OFFSET ?"
	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list history with filters: %w", err)
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

// GetHistoryByTaskKey retrieves all history records for a task by its key
func (r *TaskHistoryRepository) GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
	query := `
		SELECT th.id, th.task_id, th.old_status, th.new_status, th.agent, th.notes, th.timestamp
		FROM task_history th
		INNER JOIN tasks t ON th.task_id = t.id
		WHERE t.key = ?
		ORDER BY th.timestamp ASC
	`

	rows, err := r.db.QueryContext(ctx, query, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get history by task key: %w", err)
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
