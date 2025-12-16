package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicRepository handles CRUD operations for epics
type EpicRepository struct {
	db *DB
}

// NewEpicRepository creates a new EpicRepository
func NewEpicRepository(db *DB) *EpicRepository {
	return &EpicRepository{db: db}
}

// Create creates a new epic
func (r *EpicRepository) Create(ctx context.Context, epic *models.Epic) error {
	if err := epic.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO epics (key, title, description, status, priority, business_value)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		epic.Key,
		epic.Title,
		epic.Description,
		epic.Status,
		epic.Priority,
		epic.BusinessValue,
	)
	if err != nil {
		return fmt.Errorf("failed to create epic: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	epic.ID = id
	return nil
}

// GetByID retrieves an epic by its ID
func (r *EpicRepository) GetByID(ctx context.Context, id int64) (*models.Epic, error) {
	query := `
		SELECT id, key, title, description, status, priority, business_value,
		       created_at, updated_at
		FROM epics
		WHERE id = ?
	`

	epic := &models.Epic{}
	err := r.db.QueryRow(query, id).Scan(
		&epic.ID,
		&epic.Key,
		&epic.Title,
		&epic.Description,
		&epic.Status,
		&epic.Priority,
		&epic.BusinessValue,
		&epic.CreatedAt,
		&epic.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("epic not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}

	return epic, nil
}

// GetByKey retrieves an epic by its key
func (r *EpicRepository) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	query := `
		SELECT id, key, title, description, status, priority, business_value,
		       created_at, updated_at
		FROM epics
		WHERE key = ?
	`

	epic := &models.Epic{}
	err := r.db.QueryRow(query, key).Scan(
		&epic.ID,
		&epic.Key,
		&epic.Title,
		&epic.Description,
		&epic.Status,
		&epic.Priority,
		&epic.BusinessValue,
		&epic.CreatedAt,
		&epic.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("epic not found with key %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}

	return epic, nil
}

// List retrieves all epics, optionally filtered by status
func (r *EpicRepository) List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
	query := `
		SELECT id, key, title, description, status, priority, business_value,
		       created_at, updated_at
		FROM epics
	`
	args := []interface{}{}

	if status != nil {
		query += " WHERE status = ?"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list epics: %w", err)
	}
	defer rows.Close()

	var epics []*models.Epic
	for rows.Next() {
		epic := &models.Epic{}
		err := rows.Scan(
			&epic.ID,
			&epic.Key,
			&epic.Title,
			&epic.Description,
			&epic.Status,
			&epic.Priority,
			&epic.BusinessValue,
			&epic.CreatedAt,
			&epic.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan epic: %w", err)
		}
		epics = append(epics, epic)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating epics: %w", err)
	}

	return epics, nil
}

// Update updates an existing epic
func (r *EpicRepository) Update(ctx context.Context, epic *models.Epic) error {
	if err := epic.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE epics
		SET title = ?, description = ?, status = ?, priority = ?, business_value = ?
		WHERE id = ?
	`

	result, err := r.db.Exec(query,
		epic.Title,
		epic.Description,
		epic.Status,
		epic.Priority,
		epic.BusinessValue,
		epic.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update epic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("epic not found with id %d", epic.ID)
	}

	return nil
}

// Delete deletes an epic (and all its features/tasks via CASCADE)
func (r *EpicRepository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM epics WHERE id = ?"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete epic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("epic not found with id %d", id)
	}

	return nil
}

// CalculateProgress calculates the progress of an epic based on its features
// Formula: weighted average = Σ(feature_progress × feature_task_count) / Σ(feature_task_count)
// Features are weighted by their task count, so a feature with 100 tasks has 10× weight of a feature with 10 tasks
func (r *EpicRepository) CalculateProgress(ctx context.Context, epicID int64) (float64, error) {
	query := `
		SELECT
		    COALESCE(SUM(f.progress_pct * (
		        SELECT COUNT(*) FROM tasks t WHERE t.feature_id = f.id
		    )), 0) as weighted_sum,
		    COALESCE(SUM((
		        SELECT COUNT(*) FROM tasks t WHERE t.feature_id = f.id
		    )), 0) as total_task_count
		FROM features f
		WHERE f.epic_id = ?
	`

	var weightedSum, totalTaskCount float64
	err := r.db.QueryRow(query, epicID).Scan(&weightedSum, &totalTaskCount)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate epic progress: %w", err)
	}

	// If epic has no features or all features have no tasks, return 0.0
	if totalTaskCount == 0 {
		return 0.0, nil
	}

	return weightedSum / totalTaskCount, nil
}

// CalculateProgressByKey calculates the progress of an epic by its key
func (r *EpicRepository) CalculateProgressByKey(ctx context.Context, key string) (float64, error) {
	epic, err := r.GetByKey(ctx, key)
	if err != nil {
		return 0, err
	}
	return r.CalculateProgress(ctx, epic.ID)
}
