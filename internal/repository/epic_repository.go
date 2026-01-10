package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/slug"
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

	// Generate slug from title
	generatedSlug := slug.Generate(epic.Title)
	epic.Slug = &generatedSlug

	query := `
		INSERT INTO epics (key, title, description, status, priority, business_value, slug, file_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		epic.Key,
		epic.Title,
		epic.Description,
		epic.Status,
		epic.Priority,
		epic.BusinessValue,
		epic.Slug,
		epic.FilePath,
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
		       slug, file_path, created_at, updated_at
		FROM epics
		WHERE id = ?
	`

	epic := &models.Epic{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&epic.ID,
		&epic.Key,
		&epic.Title,
		&epic.Description,
		&epic.Status,
		&epic.Priority,
		&epic.BusinessValue,
		&epic.Slug,
		&epic.FilePath,
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

// GetByKey retrieves an epic by its key, supporting both numeric (E04) and slugged (E04-epic-name) formats.
// It tries numeric lookup first for performance, then falls back to slug-based lookup if the key contains a hyphen.
func (r *EpicRepository) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	// Try direct numeric key lookup first (e.g., "E04")
	query := `
		SELECT id, key, title, description, status, priority, business_value,
		       slug, file_path, created_at, updated_at
		FROM epics
		WHERE key = ?
	`

	epic := &models.Epic{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&epic.ID,
		&epic.Key,
		&epic.Title,
		&epic.Description,
		&epic.Status,
		&epic.Priority,
		&epic.BusinessValue,
		&epic.Slug,
		&epic.FilePath,
		&epic.CreatedAt,
		&epic.UpdatedAt,
	)

	// If found by numeric key, return immediately
	if err == nil {
		return epic, nil
	}

	// If error is not "no rows", return the error
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}

	// Not found by numeric key - try slugged format if key contains hyphen
	// Slugged format: E04-epic-name (key + slug)
	if !containsHyphen(key) {
		// No hyphen means it's not a slugged key, return not found
		return nil, sql.ErrNoRows
	}

	// Parse slugged key: extract numeric key and slug
	// Format: E04-epic-name -> key="E04", slug="epic-name"
	parts := splitSluggedKey(key)
	if len(parts) < 2 {
		return nil, sql.ErrNoRows
	}

	numericKey := parts[0]
	slug := parts[1]

	// Query by numeric key and slug
	slugQuery := `
		SELECT id, key, title, description, status, priority, business_value,
		       slug, file_path, created_at, updated_at
		FROM epics
		WHERE key = ? AND slug = ?
	`

	err = r.db.QueryRowContext(ctx, slugQuery, numericKey, slug).Scan(
		&epic.ID,
		&epic.Key,
		&epic.Title,
		&epic.Description,
		&epic.Status,
		&epic.Priority,
		&epic.BusinessValue,
		&epic.Slug,
		&epic.FilePath,
		&epic.CreatedAt,
		&epic.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get epic by slug: %w", err)
	}

	return epic, nil
}

// containsHyphen checks if a string contains a hyphen
func containsHyphen(s string) bool {
	for _, c := range s {
		if c == '-' {
			return true
		}
	}
	return false
}

// splitSluggedKey splits a slugged key into [numericKey, slug]
// Example: "E04-epic-name" -> ["E04", "epic-name"]
func splitSluggedKey(key string) []string {
	// Find first hyphen position
	hyphenIdx := -1
	for i, c := range key {
		if c == '-' {
			hyphenIdx = i
			break
		}
	}

	if hyphenIdx == -1 {
		return []string{key}
	}

	numericKey := key[:hyphenIdx]
	slug := key[hyphenIdx+1:]

	return []string{numericKey, slug}
}

// GetByFilePath retrieves an epic by its file path for collision detection
func (r *EpicRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error) {
	query := `
		SELECT id, key, title, description, status, priority, business_value, slug, file_path, created_at, updated_at
		FROM epics
		WHERE file_path = ?
	`

	epic := &models.Epic{}
	err := r.db.QueryRowContext(ctx, query, filePath).Scan(
		&epic.ID,
		&epic.Key,
		&epic.Title,
		&epic.Description,
		&epic.Status,
		&epic.Priority,
		&epic.BusinessValue,
		&epic.Slug,
		&epic.FilePath,
		&epic.CreatedAt,
		&epic.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found is not an error
		}
		return nil, fmt.Errorf("get epic by file path: %w", err)
	}

	return epic, nil
}

// List retrieves all epics, optionally filtered by status
func (r *EpicRepository) List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
	query := `
		SELECT id, key, title, description, status, priority, business_value,
		       slug, file_path, created_at, updated_at
		FROM epics
	`
	args := []interface{}{}

	if status != nil {
		query += " WHERE status = ?"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
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
			&epic.Slug,
			&epic.FilePath,
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

	result, err := r.db.ExecContext(ctx, query,
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

	result, err := r.db.ExecContext(ctx, query, id)
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

// UpdateFilePath updates or clears the file path for an epic
func (r *EpicRepository) UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error {
	query := `
		UPDATE epics
		SET file_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query, newFilePath, epicKey)
	if err != nil {
		return fmt.Errorf("update epic file path: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("epic not found: %s", epicKey)
	}

	return nil
}

// CalculateProgress calculates the progress of an epic based on its features
// Formula: simple average = Σ(feature_progress) / total_features
// Feature progress is determined by:
//   - If feature status = "completed" OR "archived" → 100% (regardless of tasks)
//   - Otherwise → use feature's progress_pct field (calculated from tasks)
func (r *EpicRepository) CalculateProgress(ctx context.Context, epicID int64) (float64, error) {
	query := `
		SELECT
		    COALESCE(SUM(
		        CASE
		            WHEN f.status IN ('completed', 'archived') THEN 100.0
		            ELSE f.progress_pct
		        END
		    ), 0) as total_progress,
		    COUNT(*) as feature_count
		FROM features f
		WHERE f.epic_id = ?
	`

	var totalProgress float64
	var featureCount int
	err := r.db.QueryRowContext(ctx, query, epicID).Scan(&totalProgress, &featureCount)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate epic progress: %w", err)
	}

	// If epic has no features, return 0.0
	if featureCount == 0 {
		return 0.0, nil
	}

	return totalProgress / float64(featureCount), nil
}

// CalculateProgressByKey calculates the progress of an epic by its key
func (r *EpicRepository) CalculateProgressByKey(ctx context.Context, key string) (float64, error) {
	epic, err := r.GetByKey(ctx, key)
	if err != nil {
		return 0, err
	}
	return r.CalculateProgress(ctx, epic.ID)
}

// CreateIfNotExists creates epic only if it doesn't exist
// Returns epic (existing or newly created) and whether it was created
func (r *EpicRepository) CreateIfNotExists(ctx context.Context, epic *models.Epic) (*models.Epic, bool, error) {
	// Start transaction to prevent race conditions
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check if epic already exists
	existing, err := r.GetByKey(ctx, epic.Key)
	if err == nil {
		// Epic exists, return it
		return existing, false, nil
	}

	// Epic doesn't exist, create it
	if err := epic.Validate(); err != nil {
		return nil, false, fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO epics (key, title, description, status, priority, business_value)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		epic.Key,
		epic.Title,
		epic.Description,
		epic.Status,
		epic.Priority,
		epic.BusinessValue,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create epic: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get last insert id: %w", err)
	}

	epic.ID = id

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return epic, true, nil
}

// UpdateKey updates the key of an epic
func (r *EpicRepository) UpdateKey(ctx context.Context, oldKey string, newKey string) error {
	// Validate new key doesn't already exist
	existing, err := r.GetByKey(ctx, newKey)
	if err == nil && existing != nil {
		return fmt.Errorf("epic with key %s already exists", newKey)
	}

	query := `
		UPDATE epics
		SET key = ?, updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query, newKey, oldKey)
	if err != nil {
		return fmt.Errorf("update epic key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("epic not found: %s", oldKey)
	}

	return nil
}

// ============================================================================
// Cascading Status Calculation Methods (E07-F14)
// ============================================================================

// GetFeatureStatusBreakdown retrieves the count of features by status for an epic
// Used for deriving epic status from child features
func (r *EpicRepository) GetFeatureStatusBreakdown(ctx context.Context, epicID int64) (map[models.FeatureStatus]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM features
		WHERE epic_id = ?
		GROUP BY status
	`

	rows, err := r.db.QueryContext(ctx, query, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature status breakdown: %w", err)
	}
	defer rows.Close()

	counts := make(map[models.FeatureStatus]int)
	for rows.Next() {
		var status models.FeatureStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan feature status count: %w", err)
		}
		counts[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feature status counts: %w", err)
	}

	return counts, nil
}

// GetFeatureStatusBreakdownByKey retrieves the count of features by status for an epic by its key
func (r *EpicRepository) GetFeatureStatusBreakdownByKey(ctx context.Context, epicKey string) (map[models.FeatureStatus]int, error) {
	epic, err := r.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, err
	}
	return r.GetFeatureStatusBreakdown(ctx, epic.ID)
}

// UpdateStatus updates the status of an epic
func (r *EpicRepository) UpdateStatus(ctx context.Context, epicID int64, status models.EpicStatus) error {
	query := `UPDATE epics SET status = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, epicID)
	if err != nil {
		return fmt.Errorf("failed to update epic status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("epic not found with id %d", epicID)
	}

	return nil
}

// UpdateStatusByKey updates the status of an epic by its key
func (r *EpicRepository) UpdateStatusByKey(ctx context.Context, epicKey string, status models.EpicStatus) error {
	epic, err := r.GetByKey(ctx, epicKey)
	if err != nil {
		return err
	}
	return r.UpdateStatus(ctx, epic.ID, status)
}

// CascadeStatusToFeaturesAndTasks updates the status of all child features and their tasks
// Used when --force is specified to override workflow validation
func (r *EpicRepository) CascadeStatusToFeaturesAndTasks(ctx context.Context, epicID int64, targetFeatureStatus models.FeatureStatus, targetTaskStatus models.TaskStatus) error {
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// First update all features
	featureQuery := `UPDATE features SET status = ? WHERE epic_id = ?`

	_, err = tx.ExecContext(ctx, featureQuery, targetFeatureStatus, epicID)
	if err != nil {
		return fmt.Errorf("failed to cascade status to features: %w", err)
	}

	// Then update all tasks in those features
	taskQuery := `
		UPDATE tasks
		SET status = ?
		WHERE feature_id IN (SELECT id FROM features WHERE epic_id = ?)
	`

	_, err = tx.ExecContext(ctx, taskQuery, targetTaskStatus, epicID)
	if err != nil {
		return fmt.Errorf("failed to cascade status to tasks: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CascadeStatusToFeaturesAndTasksByKey is a convenience method that cascades status by epic key
func (r *EpicRepository) CascadeStatusToFeaturesAndTasksByKey(ctx context.Context, epicKey string, targetFeatureStatus models.FeatureStatus, targetTaskStatus models.TaskStatus) error {
	epic, err := r.GetByKey(ctx, epicKey)
	if err != nil {
		return err
	}
	return r.CascadeStatusToFeaturesAndTasks(ctx, epic.ID, targetFeatureStatus, targetTaskStatus)
}
