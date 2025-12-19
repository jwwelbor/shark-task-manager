package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// FeatureRepository handles CRUD operations for features
type FeatureRepository struct {
	db *DB
}

// NewFeatureRepository creates a new FeatureRepository
func NewFeatureRepository(db *DB) *FeatureRepository {
	return &FeatureRepository{db: db}
}

// Create creates a new feature
func (r *FeatureRepository) Create(ctx context.Context, feature *models.Feature) error {
	if err := feature.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO features (epic_id, key, title, description, status, progress_pct, execution_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		feature.EpicID,
		feature.Key,
		feature.Title,
		feature.Description,
		feature.Status,
		feature.ProgressPct,
		feature.ExecutionOrder,
	)
	if err != nil {
		return fmt.Errorf("failed to create feature: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	feature.ID = id
	return nil
}

// GetByID retrieves a feature by its ID
func (r *FeatureRepository) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, description, status, progress_pct,
		       execution_order, created_at, updated_at
		FROM features
		WHERE id = ?
	`

	feature := &models.Feature{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&feature.ID,
		&feature.EpicID,
		&feature.Key,
		&feature.Title,
		&feature.Description,
		&feature.Status,
		&feature.ProgressPct,
		&feature.ExecutionOrder,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feature not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	return feature, nil
}

// GetByKey retrieves a feature by its key
func (r *FeatureRepository) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, description, status, progress_pct,
		       execution_order, created_at, updated_at
		FROM features
		WHERE key = ?
	`

	feature := &models.Feature{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&feature.ID,
		&feature.EpicID,
		&feature.Key,
		&feature.Title,
		&feature.Description,
		&feature.Status,
		&feature.ProgressPct,
		&feature.ExecutionOrder,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	return feature, nil
}

// GetByFilePath retrieves a feature by its file path for collision detection
func (r *FeatureRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, description, status, progress_pct,
		       execution_order, file_path, created_at, updated_at
		FROM features
		WHERE file_path = ?
	`

	feature := &models.Feature{}
	err := r.db.QueryRowContext(ctx, query, filePath).Scan(
		&feature.ID,
		&feature.EpicID,
		&feature.Key,
		&feature.Title,
		&feature.Description,
		&feature.Status,
		&feature.ProgressPct,
		&feature.ExecutionOrder,
		&feature.FilePath,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found is not an error
		}
		return nil, fmt.Errorf("get feature by file path: %w", err)
	}

	return feature, nil
}

// ListByEpic retrieves all features for an epic
func (r *FeatureRepository) ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, description, status, progress_pct,
		       execution_order, created_at, updated_at
		FROM features
		WHERE epic_id = ?
		ORDER BY execution_order NULLS LAST, created_at
	`

	rows, err := r.db.QueryContext(ctx, query, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}
	defer rows.Close()

	var features []*models.Feature
	for rows.Next() {
		feature := &models.Feature{}
		err := rows.Scan(
			&feature.ID,
			&feature.EpicID,
			&feature.Key,
			&feature.Title,
			&feature.Description,
			&feature.Status,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.CreatedAt,
			&feature.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, feature)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating features: %w", err)
	}

	return features, nil
}

// List retrieves all features
func (r *FeatureRepository) List(ctx context.Context) ([]*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, description, status, progress_pct,
		       execution_order, created_at, updated_at
		FROM features
		ORDER BY execution_order NULLS LAST, created_at
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}
	defer rows.Close()

	var features []*models.Feature
	for rows.Next() {
		feature := &models.Feature{}
		err := rows.Scan(
			&feature.ID,
			&feature.EpicID,
			&feature.Key,
			&feature.Title,
			&feature.Description,
			&feature.Status,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.CreatedAt,
			&feature.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, feature)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating features: %w", err)
	}

	return features, nil
}

// Update updates an existing feature
func (r *FeatureRepository) Update(ctx context.Context, feature *models.Feature) error {
	if err := feature.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE features
		SET title = ?, description = ?, status = ?, progress_pct = ?, execution_order = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		feature.Title,
		feature.Description,
		feature.Status,
		feature.ProgressPct,
		feature.ExecutionOrder,
		feature.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update feature: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feature not found with id %d", feature.ID)
	}

	return nil
}

// Delete deletes a feature (and all its tasks via CASCADE)
func (r *FeatureRepository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM features WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete feature: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feature not found with id %d", id)
	}

	return nil
}

// UpdateFilePath updates or clears the file path for a feature
func (r *FeatureRepository) UpdateFilePath(ctx context.Context, featureKey string, newFilePath *string) error {
	query := `
		UPDATE features
		SET file_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query, newFilePath, featureKey)
	if err != nil {
		return fmt.Errorf("update feature file path: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feature not found: %s", featureKey)
	}

	return nil
}

// CalculateProgress calculates the progress of a feature based on its tasks
// Formula: (count of tasks with status='completed' OR status='archived') / (total tasks) × 100
func (r *FeatureRepository) CalculateProgress(ctx context.Context, featureID int64) (float64, error) {
	query := `
		SELECT
		    COUNT(*) as total_tasks,
		    COALESCE(SUM(CASE WHEN status IN ('completed', 'archived') THEN 1 ELSE 0 END), 0) as completed_tasks
		FROM tasks
		WHERE feature_id = ?
	`

	var totalTasks, completedTasks int
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&totalTasks, &completedTasks)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate feature progress: %w", err)
	}

	// If feature has no tasks, return 0.0 (not an error)
	if totalTasks == 0 {
		return 0.0, nil
	}

	return (float64(completedTasks) / float64(totalTasks)) * 100.0, nil
}

// CalculateProgressByKey calculates the progress of a feature by its key
func (r *FeatureRepository) CalculateProgressByKey(ctx context.Context, key string) (float64, error) {
	feature, err := r.GetByKey(ctx, key)
	if err != nil {
		return 0, err
	}
	return r.CalculateProgress(ctx, feature.ID)
}

// UpdateProgress recalculates and updates the cached progress_pct field
func (r *FeatureRepository) UpdateProgress(ctx context.Context, featureID int64) error {
	progress, err := r.CalculateProgress(ctx, featureID)
	if err != nil {
		return err
	}

	query := "UPDATE features SET progress_pct = ? WHERE id = ?"
	_, err = r.db.ExecContext(ctx, query, progress, featureID)
	if err != nil {
		return fmt.Errorf("failed to update feature progress: %w", err)
	}

	return nil
}

// UpdateProgressByKey recalculates and updates the cached progress_pct field by feature key
func (r *FeatureRepository) UpdateProgressByKey(ctx context.Context, key string) error {
	feature, err := r.GetByKey(ctx, key)
	if err != nil {
		return err
	}
	return r.UpdateProgress(ctx, feature.ID)
}

// ListByStatus retrieves all features with a specific status
func (r *FeatureRepository) ListByStatus(ctx context.Context, status models.FeatureStatus) ([]*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, description, status, progress_pct,
		       execution_order, created_at, updated_at
		FROM features
		WHERE status = ?
		ORDER BY execution_order NULLS LAST, created_at
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list features by status: %w", err)
	}
	defer rows.Close()

	var features []*models.Feature
	for rows.Next() {
		feature := &models.Feature{}
		err := rows.Scan(
			&feature.ID,
			&feature.EpicID,
			&feature.Key,
			&feature.Title,
			&feature.Description,
			&feature.Status,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.CreatedAt,
			&feature.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, feature)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating features: %w", err)
	}

	return features, nil
}

// ListByEpicAndStatus retrieves features filtered by both epic and status
func (r *FeatureRepository) ListByEpicAndStatus(ctx context.Context, epicID int64, status models.FeatureStatus) ([]*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, description, status, progress_pct,
		       execution_order, created_at, updated_at
		FROM features
		WHERE epic_id = ? AND status = ?
		ORDER BY execution_order NULLS LAST, created_at
	`

	rows, err := r.db.QueryContext(ctx, query, epicID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list features by epic and status: %w", err)
	}
	defer rows.Close()

	var features []*models.Feature
	for rows.Next() {
		feature := &models.Feature{}
		err := rows.Scan(
			&feature.ID,
			&feature.EpicID,
			&feature.Key,
			&feature.Title,
			&feature.Description,
			&feature.Status,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.CreatedAt,
			&feature.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, feature)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating features: %w", err)
	}

	return features, nil
}

// GetTaskCount returns the total number of tasks for a feature
func (r *FeatureRepository) GetTaskCount(ctx context.Context, featureID int64) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE feature_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get task count: %w", err)
	}

	return count, nil
}

// CreateIfNotExists creates feature only if it doesn't exist
// Returns feature (existing or newly created) and whether it was created
func (r *FeatureRepository) CreateIfNotExists(ctx context.Context, feature *models.Feature) (*models.Feature, bool, error) {
	// Start transaction to prevent race conditions
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if feature already exists
	existing, err := r.GetByKey(ctx, feature.Key)
	if err == nil {
		// Feature exists, return it
		return existing, false, nil
	}

	// Feature doesn't exist, create it
	if err := feature.Validate(); err != nil {
		return nil, false, fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO features (epic_id, key, title, description, status, progress_pct, execution_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		feature.EpicID,
		feature.Key,
		feature.Title,
		feature.Description,
		feature.Status,
		feature.ProgressPct,
		feature.ExecutionOrder,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create feature: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get last insert id: %w", err)
	}

	feature.ID = id

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return feature, true, nil
}
