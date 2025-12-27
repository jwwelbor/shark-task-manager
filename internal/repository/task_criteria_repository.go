package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskCriteriaRepository handles CRUD operations for task acceptance criteria
type TaskCriteriaRepository struct {
	db *DB
}

// NewTaskCriteriaRepository creates a new TaskCriteriaRepository
func NewTaskCriteriaRepository(db *DB) *TaskCriteriaRepository {
	return &TaskCriteriaRepository{db: db}
}

// Create creates a new task criterion
func (r *TaskCriteriaRepository) Create(ctx context.Context, criteria *models.TaskCriteria) error {
	if err := criteria.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO task_criteria (
			task_id, criterion, status, verified_at, verification_notes
		)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		criteria.TaskID,
		criteria.Criterion,
		criteria.Status,
		criteria.VerifiedAt,
		criteria.VerificationNotes,
	)
	if err != nil {
		return fmt.Errorf("failed to create task criterion: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	criteria.ID = id
	return nil
}

// GetByID retrieves a task criterion by its ID
func (r *TaskCriteriaRepository) GetByID(ctx context.Context, id int64) (*models.TaskCriteria, error) {
	query := `
		SELECT id, task_id, criterion, status, verified_at, verification_notes, created_at
		FROM task_criteria
		WHERE id = ?
	`

	criteria := &models.TaskCriteria{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&criteria.ID,
		&criteria.TaskID,
		&criteria.Criterion,
		&criteria.Status,
		&criteria.VerifiedAt,
		&criteria.VerificationNotes,
		&criteria.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task criterion not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task criterion: %w", err)
	}

	return criteria, nil
}

// GetByTaskID retrieves all criteria for a task
func (r *TaskCriteriaRepository) GetByTaskID(ctx context.Context, taskID int64) ([]*models.TaskCriteria, error) {
	query := `
		SELECT id, task_id, criterion, status, verified_at, verification_notes, created_at
		FROM task_criteria
		WHERE task_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task criteria: %w", err)
	}
	defer rows.Close()

	var criteria []*models.TaskCriteria
	for rows.Next() {
		criterion := &models.TaskCriteria{}
		err := rows.Scan(
			&criterion.ID,
			&criterion.TaskID,
			&criterion.Criterion,
			&criterion.Status,
			&criterion.VerifiedAt,
			&criterion.VerificationNotes,
			&criterion.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task criterion: %w", err)
		}
		criteria = append(criteria, criterion)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task criteria: %w", err)
	}

	return criteria, nil
}

// Update updates a task criterion
func (r *TaskCriteriaRepository) Update(ctx context.Context, criteria *models.TaskCriteria) error {
	if err := criteria.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE task_criteria
		SET criterion = ?, status = ?, verified_at = ?, verification_notes = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		criteria.Criterion,
		criteria.Status,
		criteria.VerifiedAt,
		criteria.VerificationNotes,
		criteria.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task criterion: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task criterion not found with id %d", criteria.ID)
	}

	return nil
}

// UpdateStatus updates the status of a criterion and optionally sets verification fields
func (r *TaskCriteriaRepository) UpdateStatus(ctx context.Context, id int64, status models.CriteriaStatus, notes *string) error {
	if err := models.ValidateCriteriaStatus(string(status)); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	var verifiedAt *time.Time
	if status == models.CriteriaStatusComplete || status == models.CriteriaStatusFailed {
		now := time.Now()
		verifiedAt = &now
	}

	query := `
		UPDATE task_criteria
		SET status = ?, verified_at = ?, verification_notes = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, status, verifiedAt, notes, id)
	if err != nil {
		return fmt.Errorf("failed to update criterion status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task criterion not found with id %d", id)
	}

	return nil
}

// Delete deletes a task criterion
func (r *TaskCriteriaRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM task_criteria WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task criterion: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task criterion not found with id %d", id)
	}

	return nil
}

// DeleteByTaskID deletes all criteria for a task
func (r *TaskCriteriaRepository) DeleteByTaskID(ctx context.Context, taskID int64) error {
	query := `DELETE FROM task_criteria WHERE task_id = ?`

	_, err := r.db.ExecContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task criteria: %w", err)
	}

	return nil
}

// GetSummary returns a summary of criteria statuses for a task
type CriteriaSummary struct {
	TaskID          int64
	TotalCount      int
	PendingCount    int
	InProgressCount int
	CompleteCount   int
	FailedCount     int
	NACount         int
	CompletionPct   float64
}

// GetSummaryByTaskID calculates a summary of criteria for a task
func (r *TaskCriteriaRepository) GetSummaryByTaskID(ctx context.Context, taskID int64) (*CriteriaSummary, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) as in_progress,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as complete,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			SUM(CASE WHEN status = 'na' THEN 1 ELSE 0 END) as na
		FROM task_criteria
		WHERE task_id = ?
	`

	summary := &CriteriaSummary{TaskID: taskID}
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&summary.TotalCount,
		&summary.PendingCount,
		&summary.InProgressCount,
		&summary.CompleteCount,
		&summary.FailedCount,
		&summary.NACount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get criteria summary: %w", err)
	}

	// Calculate completion percentage (complete + na) / total
	if summary.TotalCount > 0 {
		summary.CompletionPct = float64(summary.CompleteCount+summary.NACount) / float64(summary.TotalCount) * 100.0
	}

	return summary, nil
}
