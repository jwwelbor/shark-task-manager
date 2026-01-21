package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/progress"
	"github.com/jwwelbor/shark-task-manager/internal/slug"
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

	// Generate slug from title
	generatedSlug := slug.Generate(feature.Title)
	feature.Slug = &generatedSlug

	query := `
		INSERT INTO features (epic_id, key, title, slug, description, status, status_override, progress_pct, execution_order, file_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		feature.EpicID,
		feature.Key,
		feature.Title,
		feature.Slug,
		feature.Description,
		feature.Status,
		feature.StatusOverride,
		feature.ProgressPct,
		feature.ExecutionOrder,
		feature.FilePath,
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
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, created_at, updated_at
		FROM features
		WHERE id = ?
	`

	feature := &models.Feature{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&feature.ID,
		&feature.EpicID,
		&feature.Key,
		&feature.Title,
		&feature.Slug,
		&feature.Description,
		&feature.Status,
		&feature.StatusOverride,
		&feature.ProgressPct,
		&feature.ExecutionOrder,
		&feature.FilePath,
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

// GetByKey retrieves a feature by its key with support for multiple key formats:
// - Full key: "E07-F11"
// - Numeric key: "F11" or "f11"
// - Slugged key: "F11-slug-name" or "f11-slug-name"
// - Full key with slug: "E07-F11-slug-name"
//
// The method tries lookups in this order:
// 1. Exact match on key column
// 2. Pattern match for numeric key (key LIKE '%F11')
// 3. Pattern match for slugged key (key || '-' || slug matches input)
func (r *FeatureRepository) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	// Normalize key to uppercase for comparison
	normalizedKey := strings.ToUpper(key)

	// Try 1: Exact match on key column
	feature, err := r.getByExactKey(ctx, normalizedKey)
	if err == nil {
		return feature, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get feature by exact key: %w", err)
	}

	// Try 2: Numeric key pattern (F11, f11) -> match features with key ending in -F11
	if strings.HasPrefix(normalizedKey, "F") {
		feature, err = r.getByNumericKey(ctx, normalizedKey)
		if err == nil {
			return feature, nil
		}
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to get feature by numeric key: %w", err)
		}
	}

	// Try 3: Slugged key pattern (F11-slug-name or E07-F11-slug-name)
	// Extract the numeric part and slug, then match against key and slug columns
	feature, err = r.getBySluggedKey(ctx, normalizedKey)
	if err == nil {
		return feature, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get feature by slugged key: %w", err)
	}

	// No match found
	return nil, sql.ErrNoRows
}

// getByExactKey performs exact match lookup on the key column
func (r *FeatureRepository) getByExactKey(ctx context.Context, key string) (*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, created_at, updated_at
		FROM features
		WHERE key = ?
	`

	feature := &models.Feature{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&feature.ID,
		&feature.EpicID,
		&feature.Key,
		&feature.Title,
		&feature.Slug,
		&feature.Description,
		&feature.Status,
		&feature.StatusOverride,
		&feature.ProgressPct,
		&feature.ExecutionOrder,
		&feature.FilePath,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	return feature, err
}

// getByNumericKey matches features where the key ends with the numeric part
// Example: "F11" matches "E07-F11", "E05-F11", etc.
func (r *FeatureRepository) getByNumericKey(ctx context.Context, numericKey string) (*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, created_at, updated_at
		FROM features
		WHERE key LIKE ?
	`

	// Match pattern: any epic prefix followed by the numeric key
	// E.g., "F11" -> "%F11" which matches "E07-F11", "E99-F11", etc.
	pattern := "%-" + numericKey

	feature := &models.Feature{}
	err := r.db.QueryRowContext(ctx, query, pattern).Scan(
		&feature.ID,
		&feature.EpicID,
		&feature.Key,
		&feature.Title,
		&feature.Slug,
		&feature.Description,
		&feature.Status,
		&feature.StatusOverride,
		&feature.ProgressPct,
		&feature.ExecutionOrder,
		&feature.FilePath,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	return feature, err
}

// getBySluggedKey matches features by parsing slugged key formats
// Formats: "F11-slug-name", "f11-slug-name", "E07-F11-slug-name"
func (r *FeatureRepository) getBySluggedKey(ctx context.Context, sluggedKey string) (*models.Feature, error) {
	// Parse slugged key to extract numeric part and slug
	// Possible formats:
	// - F11-user-auth-feature
	// - E07-F11-user-auth-feature

	parts := strings.Split(sluggedKey, "-")
	if len(parts) < 2 {
		return nil, sql.ErrNoRows
	}

	var numericPart string
	var slugPart string

	// Check if first part is epic (E##) or feature (F##)
	if strings.HasPrefix(parts[0], "E") && len(parts) >= 3 {
		// Format: E07-F11-slug-name
		numericPart = parts[1]                  // F11
		slugPart = strings.Join(parts[2:], "-") // slug-name
	} else if strings.HasPrefix(parts[0], "F") {
		// Format: F11-slug-name
		numericPart = parts[0]                  // F11
		slugPart = strings.Join(parts[1:], "-") // slug-name
	} else {
		return nil, sql.ErrNoRows
	}

	// Query for features where key ends with numeric part AND slug matches
	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, created_at, updated_at
		FROM features
		WHERE key LIKE ? AND slug = ?
	`

	pattern := "%-" + numericPart
	slugLower := strings.ToLower(slugPart)

	feature := &models.Feature{}
	err := r.db.QueryRowContext(ctx, query, pattern, slugLower).Scan(
		&feature.ID,
		&feature.EpicID,
		&feature.Key,
		&feature.Title,
		&feature.Slug,
		&feature.Description,
		&feature.Status,
		&feature.StatusOverride,
		&feature.ProgressPct,
		&feature.ExecutionOrder,
		&feature.FilePath,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	return feature, err
}

// GetByFilePath retrieves a feature by its file path for collision detection
func (r *FeatureRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
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
		&feature.Slug,
		&feature.Description,
		&feature.Status,
		&feature.StatusOverride,
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
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, created_at, updated_at
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
			&feature.Slug,
			&feature.Description,
			&feature.Status,
			&feature.StatusOverride,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.FilePath,
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
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, created_at, updated_at
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
			&feature.Slug,
			&feature.Description,
			&feature.Status,
			&feature.StatusOverride,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.FilePath,
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

	// Check if execution_order is being changed - if so, cascade to other features
	var oldFeature *models.Feature
	var err error
	var needsCascade bool

	if feature.ExecutionOrder != nil {
		oldFeature, err = r.GetByID(ctx, feature.ID)
		if err != nil {
			return fmt.Errorf("failed to get old feature: %w", err)
		}

		// Check if order actually changed
		needsCascade = (oldFeature.ExecutionOrder == nil) ||
			(oldFeature.ExecutionOrder != nil && *oldFeature.ExecutionOrder != *feature.ExecutionOrder)
	}

	// Start transaction for cascade updates
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// If cascade is needed, get all features BEFORE updating, then resequence ALL features
	if needsCascade {
		// Get all features in the same epic (before any updates)
		allFeatures, err := r.listByEpicInTx(ctx, tx, feature.EpicID)
		if err != nil {
			return fmt.Errorf("failed to list features for cascade: %w", err)
		}

		// Convert to orderedItem format
		var items []orderedItem
		for _, f := range allFeatures {
			items = append(items, orderedItem{
				ID:             f.ID,
				ExecutionOrder: f.ExecutionOrder,
			})
		}

		// Resequence
		resequenced := resequenceOrders(items, feature.ID, feature.ExecutionOrder)

		// Update ALL features with new orders
		updateQuery := "UPDATE features SET execution_order = ? WHERE id = ?"
		for _, item := range resequenced {
			_, err := tx.ExecContext(ctx, updateQuery, item.ExecutionOrder, item.ID)
			if err != nil {
				return fmt.Errorf("failed to cascade update order for feature %d: %w", item.ID, err)
			}
		}

		// Now update the main feature's other fields (execution_order already updated above)
		query := `
			UPDATE features
			SET title = ?, description = ?, status = ?, progress_pct = ?
			WHERE id = ?
		`

		result, err := tx.ExecContext(ctx, query,
			feature.Title,
			feature.Description,
			feature.Status,
			feature.ProgressPct,
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
	} else {
		// No cascade needed, just update the feature normally
		query := `
			UPDATE features
			SET title = ?, description = ?, status = ?, progress_pct = ?, execution_order = ?
			WHERE id = ?
		`

		result, err := tx.ExecContext(ctx, query,
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
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// listByEpicInTx lists features by epic within a transaction
func (r *FeatureRepository) listByEpicInTx(ctx context.Context, tx *sql.Tx, epicID int64) ([]*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, slug, description, status, progress_pct, execution_order,
		       created_at, updated_at, file_path
		FROM features
		WHERE epic_id = ?
		ORDER BY execution_order ASC
	`

	rows, err := tx.QueryContext(ctx, query, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to query features: %w", err)
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
			&feature.Slug,
			&feature.Description,
			&feature.Status,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.CreatedAt,
			&feature.UpdatedAt,
			&feature.FilePath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, feature)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating features: %w", err)
	}

	return features, nil
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

// CalculateProgress calculates the weighted progress of a feature based on task status weights
// Uses workflow config to apply progress weights to each task status
func (r *FeatureRepository) CalculateProgress(ctx context.Context, featureID int64) (float64, error) {
	// Get task status breakdown
	query := `
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE feature_id = ?
		GROUP BY status
	`

	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return 0, fmt.Errorf("failed to get task status breakdown: %w", err)
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var taskStatus string
		var count int
		if err := rows.Scan(&taskStatus, &count); err != nil {
			return 0, fmt.Errorf("failed to scan status count: %w", err)
		}
		statusCounts[taskStatus] = count
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating status counts: %w", err)
	}

	// If feature has no tasks, return 0.0 (not an error)
	if len(statusCounts) == 0 {
		return 0.0, nil
	}

	// Load workflow config for weighted progress calculation
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	var cfg *config.WorkflowConfig
	if cwd != "" {
		configPath := cwd + "/.sharkconfig.json"
		cfg, err = config.LoadWorkflowConfig(configPath)
		if err != nil {
			// If config load fails, use default weights (completion-based)
			cfg = nil
		}
	}

	// Calculate weighted progress using progress package
	progressInfo := progress.CalculateProgress(statusCounts, cfg)
	return progressInfo.WeightedPct, nil
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
// Automatically sets feature status to "completed" when progress reaches 100%
func (r *FeatureRepository) UpdateProgress(ctx context.Context, featureID int64) error {
	progress, err := r.CalculateProgress(ctx, featureID)
	if err != nil {
		return err
	}

	// Auto-complete feature when all tasks are completed
	var newStatus models.FeatureStatus
	if progress >= 100.0 {
		newStatus = models.FeatureStatusCompleted
	} else {
		// Keep existing status but update progress
		// For features that are not yet 100% complete, don't change their status
		query := "UPDATE features SET progress_pct = ? WHERE id = ?"
		_, err = r.db.ExecContext(ctx, query, progress, featureID)
		if err != nil {
			return fmt.Errorf("failed to update feature progress: %w", err)
		}
		return nil
	}

	// Update both progress and status when reaching 100%
	query := "UPDATE features SET progress_pct = ?, status = ? WHERE id = ?"
	_, err = r.db.ExecContext(ctx, query, progress, newStatus, featureID)
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
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, created_at, updated_at
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
			&feature.Slug,
			&feature.Description,
			&feature.Status,
			&feature.StatusOverride,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.FilePath,
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
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, created_at, updated_at
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
			&feature.Slug,
			&feature.Description,
			&feature.Status,
			&feature.StatusOverride,
			&feature.ProgressPct,
			&feature.ExecutionOrder,
			&feature.FilePath,
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
	defer func() { _ = tx.Rollback() }()

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

// UpdateKey updates the key of a feature
func (r *FeatureRepository) UpdateKey(ctx context.Context, oldKey string, newKey string) error {
	// Validate new key doesn't already exist
	existing, err := r.GetByKey(ctx, newKey)
	if err == nil && existing != nil {
		return fmt.Errorf("feature with key %s already exists", newKey)
	}

	query := `
		UPDATE features
		SET key = ?, updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query, newKey, oldKey)
	if err != nil {
		return fmt.Errorf("update feature key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feature not found: %s", oldKey)
	}

	return nil
}

// ============================================================================
// Cascading Status Calculation Methods (E07-F14)
// ============================================================================

// GetTaskStatusBreakdown retrieves the count of tasks by status for a feature
// Used for deriving feature status from child tasks
func (r *FeatureRepository) GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE feature_id = ?
		GROUP BY status
	`

	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown: %w", err)
	}
	defer rows.Close()

	counts := make(map[models.TaskStatus]int)
	for rows.Next() {
		var status models.TaskStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan task status count: %w", err)
		}
		counts[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task status counts: %w", err)
	}

	return counts, nil
}

// GetTaskStatusBreakdownByKey retrieves the count of tasks by status for a feature by its key
func (r *FeatureRepository) GetTaskStatusBreakdownByKey(ctx context.Context, featureKey string) (map[models.TaskStatus]int, error) {
	feature, err := r.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, err
	}
	return r.GetTaskStatusBreakdown(ctx, feature.ID)
}

// SetStatusOverride enables or disables status override for a feature
// When override=true, automatic status calculation is disabled
func (r *FeatureRepository) SetStatusOverride(ctx context.Context, featureID int64, override bool) error {
	query := `UPDATE features SET status_override = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, override, featureID)
	if err != nil {
		return fmt.Errorf("failed to set status override: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feature not found with id %d", featureID)
	}

	return nil
}

// SetStatusOverrideByKey enables or disables status override for a feature by its key
func (r *FeatureRepository) SetStatusOverrideByKey(ctx context.Context, featureKey string, override bool) error {
	feature, err := r.GetByKey(ctx, featureKey)
	if err != nil {
		return err
	}
	return r.SetStatusOverride(ctx, feature.ID, override)
}

// UpdateStatusIfNotOverridden updates the status only if status_override is false
// Returns true if the status was updated, false if skipped due to override
func (r *FeatureRepository) UpdateStatusIfNotOverridden(ctx context.Context, featureID int64, newStatus models.FeatureStatus) (bool, error) {
	query := `
		UPDATE features
		SET status = ?
		WHERE id = ? AND (status_override = 0 OR status_override IS NULL)
	`

	result, err := r.db.ExecContext(ctx, query, newStatus, featureID)
	if err != nil {
		return false, fmt.Errorf("failed to update status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rows > 0, nil
}

// UpdateStatusIfNotOverriddenByKey updates the status only if status_override is false
func (r *FeatureRepository) UpdateStatusIfNotOverriddenByKey(ctx context.Context, featureKey string, newStatus models.FeatureStatus) (bool, error) {
	feature, err := r.GetByKey(ctx, featureKey)
	if err != nil {
		return false, err
	}
	return r.UpdateStatusIfNotOverridden(ctx, feature.ID, newStatus)
}

// CascadeStatusToTasks updates the status of all child tasks to match a target task status
// Used when --force is specified to override workflow validation
func (r *FeatureRepository) CascadeStatusToTasks(ctx context.Context, featureID int64, targetTaskStatus models.TaskStatus) error {
	query := `UPDATE tasks SET status = ? WHERE feature_id = ?`

	result, err := r.db.ExecContext(ctx, query, targetTaskStatus, featureID)
	if err != nil {
		return fmt.Errorf("failed to cascade status to tasks: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the number of tasks updated (optional, for debugging)
	_ = rows

	return nil
}

// CascadeStatusToTasksByKey is a convenience method that cascades status by feature key
func (r *FeatureRepository) CascadeStatusToTasksByKey(ctx context.Context, featureKey string, targetTaskStatus models.TaskStatus) error {
	feature, err := r.GetByKey(ctx, featureKey)
	if err != nil {
		return err
	}
	return r.CascadeStatusToTasks(ctx, feature.ID, targetTaskStatus)
}
