package feature

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
	"github.com/jwwelbor/shark-task-manager/internal/slug"
)

var tracer = repoutil.NewTracer("internal/repository/feature")

// FeatureRepository handles CRUD operations for features
type FeatureRepository struct {
	db *dbconn.DB
}

// NewFeatureRepository creates a new FeatureRepository
func NewFeatureRepository(db *dbconn.DB) *FeatureRepository {
	return &FeatureRepository{db: db}
}

// Create creates a new feature
func (r *FeatureRepository) Create(ctx context.Context, feature *models.Feature) (retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.Create",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "features"),
			attribute.String("db.key", feature.Key),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	if err := feature.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Generate slug from title
	generatedSlug := slug.Generate(feature.Title)
	feature.Slug = &generatedSlug

	query := `
		INSERT INTO features (epic_id, key, title, slug, description, status, status_override, progress_pct, execution_order, file_path, context_data, size)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		feature.ContextData,
		feature.Size,
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
func (r *FeatureRepository) GetByID(ctx context.Context, id int64) (_ *models.Feature, retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.GetByID",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "features"),
			attribute.Int64("db.id", id),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, context_data, size, created_at, updated_at
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
		&feature.ContextData,
		&feature.Size,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feature not found with id %d: %w", id, repoerr.ErrNotFound)
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
func (r *FeatureRepository) GetByKey(ctx context.Context, key string) (_ *models.Feature, retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.GetByKey",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "features"),
			attribute.String("db.key", key),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

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
		       execution_order, file_path, context_data, size, created_at, updated_at
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
		&feature.ContextData,
		&feature.Size,
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
		       execution_order, file_path, context_data, size, created_at, updated_at
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
		&feature.ContextData,
		&feature.Size,
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

		// If the slug part is purely numeric (e.g., "015"), this is a task key format
		// like "E15-F11-015". Look up the parent feature instead of treating it as a slug.
		if repoutil.IsNumeric(slugPart) {
			featureKey := parts[0] + "-" + numericPart // e.g., "E15-F11"
			return r.getByExactKey(ctx, featureKey)
		}
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
		       execution_order, file_path, context_data, size, created_at, updated_at
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
		&feature.ContextData,
		&feature.Size,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	return feature, err
}

// GetByFilePath retrieves a feature by its file path for collision detection
func (r *FeatureRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, context_data, size, created_at, updated_at
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
		&feature.ContextData,
		&feature.Size,
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
func (r *FeatureRepository) ListByEpic(ctx context.Context, epicID int64) (_ []*models.Feature, retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.ListByEpic",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "features"),
			attribute.Int64("db.epic_id", epicID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, context_data, size, created_at, updated_at
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
			&feature.ContextData,
			&feature.Size,
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
func (r *FeatureRepository) List(ctx context.Context) (_ []*models.Feature, retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.List",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "features"),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, context_data, size, created_at, updated_at
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
			&feature.ContextData,
			&feature.Size,
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

// BeginTx starts a new database transaction for use by the service layer.
// Services own transaction boundaries per Standard 8; repositories participate
// in service-owned transactions by accepting *sql.Tx parameters.
func (r *FeatureRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTxContext(ctx)
}

// Update updates an existing feature.
// It starts an internal transaction to handle execution_order cascade atomically.
// For service-owned transactions, use UpdateWithTx directly after calling BeginTx.
func (r *FeatureRepository) Update(ctx context.Context, feature *models.Feature) (retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.Update",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "features"),
			attribute.String("db.key", feature.Key),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	return r.updateInternal(ctx, feature, false)
}

// UpdateNoResequence updates a feature without cascading execution_order changes
// to siblings. Used to preserve intentional duplicate-order groups (parallel
// work) when callers re-assign a feature's order via
// `shark feature update --parallel`. Equivalent to Update in every other respect.
func (r *FeatureRepository) UpdateNoResequence(ctx context.Context, feature *models.Feature) (retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.UpdateNoResequence",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "features"),
			attribute.String("db.key", feature.Key),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	return r.updateInternal(ctx, feature, true)
}

// updateInternal performs the feature update. When forceSkipCascade is true the
// execution_order resequence is suppressed regardless of whether the order has
// actually changed.
//
// In the skip-cascade path (TD-008), no transaction is opened — the operation
// is a single non-cascading row update, so BEGIN/COMMIT add latency
// (meaningful on Turso, negligible on local SQLite) without any atomicity
// benefit.
func (r *FeatureRepository) updateInternal(ctx context.Context, feature *models.Feature, forceSkipCascade bool) error {
	if err := feature.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Skip-cascade fast path: single-row UPDATE, no transaction. Used by
	// UpdateNoResequence (--parallel renumber).
	if forceSkipCascade {
		return r.updateRowDirect(ctx, feature)
	}

	// Check if execution_order is being changed - if so, cascade to other features.
	var needsCascade bool
	if feature.ExecutionOrder != nil {
		oldFeature, err := r.GetByID(ctx, feature.ID)
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

	if err := r.updateWithTx(ctx, tx, feature, needsCascade); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// updateRowDirect performs a single-row feature UPDATE outside any transaction.
// Used by the --parallel update path (forceSkipCascade=true) where no sibling
// cascade is needed and atomicity across multiple rows is irrelevant. See
// TD-008 for the rationale.
func (r *FeatureRepository) updateRowDirect(ctx context.Context, feature *models.Feature) error {
	query := `
		UPDATE features
		SET title = ?, description = ?, status = ?, progress_pct = ?, execution_order = ?, file_path = ?, size = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		feature.Title,
		feature.Description,
		feature.Status,
		feature.ProgressPct,
		feature.ExecutionOrder,
		feature.FilePath,
		feature.Size,
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
		return fmt.Errorf("feature not found with id %d: %w", feature.ID, repoerr.ErrNotFound)
	}

	return nil
}

// updateWithTx performs the feature update within a caller-provided transaction.
// The needsCascade flag indicates whether execution_order cascade is required.
// Validation must be performed by the caller before invoking this method.
func (r *FeatureRepository) updateWithTx(ctx context.Context, tx *sql.Tx, feature *models.Feature, needsCascade bool) error {
	// If cascade is needed, get all features BEFORE updating, then resequence ALL features
	if needsCascade {
		// Get all features in the same epic (before any updates)
		allFeatures, err := r.listByEpicInTx(ctx, tx, feature.EpicID)
		if err != nil {
			return fmt.Errorf("failed to list features for cascade: %w", err)
		}

		// Convert to repoutil.OrderedItem format
		var items []repoutil.OrderedItem
		for _, f := range allFeatures {
			items = append(items, repoutil.OrderedItem{
				ID:             f.ID,
				ExecutionOrder: f.ExecutionOrder,
			})
		}

		// Resequence
		resequenced := repoutil.ResequenceOrders(items, feature.ID, feature.ExecutionOrder)

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
			SET title = ?, description = ?, status = ?, progress_pct = ?, file_path = ?, size = ?
			WHERE id = ?
		`

		result, err := tx.ExecContext(ctx, query,
			feature.Title,
			feature.Description,
			feature.Status,
			feature.ProgressPct,
			feature.FilePath,
			feature.Size,
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
			return fmt.Errorf("feature not found with id %d: %w", feature.ID, repoerr.ErrNotFound)
		}
	} else {
		// No cascade needed, just update the feature normally
		query := `
			UPDATE features
			SET title = ?, description = ?, status = ?, progress_pct = ?, execution_order = ?, file_path = ?, size = ?
			WHERE id = ?
		`

		result, err := tx.ExecContext(ctx, query,
			feature.Title,
			feature.Description,
			feature.Status,
			feature.ProgressPct,
			feature.ExecutionOrder,
			feature.FilePath,
			feature.Size,
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
			return fmt.Errorf("feature not found with id %d: %w", feature.ID, repoerr.ErrNotFound)
		}
	}

	return nil
}

// listByEpicInTx lists features by epic within a transaction
func (r *FeatureRepository) listByEpicInTx(ctx context.Context, tx *sql.Tx, epicID int64) ([]*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, slug, description, status, progress_pct, execution_order,
		       file_path, context_data, size, created_at, updated_at
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
			&feature.FilePath,
			&feature.ContextData,
			&feature.Size,
			&feature.CreatedAt,
			&feature.UpdatedAt,
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
func (r *FeatureRepository) Delete(ctx context.Context, id int64) (retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.Delete",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "DELETE"),
			attribute.String("db.table", "features"),
			attribute.Int64("db.id", id),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

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
		return fmt.Errorf("feature not found with id %d: %w", id, repoerr.ErrNotFound)
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
		return fmt.Errorf("feature not found: %s: %w", featureKey, repoerr.ErrNotFound)
	}

	return nil
}

// ListByStatus retrieves all features with a specific status
func (r *FeatureRepository) ListByStatus(ctx context.Context, status models.FeatureStatus) ([]*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, context_data, size, created_at, updated_at
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
			&feature.ContextData,
			&feature.Size,
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
		       execution_order, file_path, context_data, size, created_at, updated_at
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
			&feature.ContextData,
			&feature.Size,
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

// CreateIfNotExists creates feature only if it doesn't exist.
// Returns feature (existing or newly created) and whether it was created.
func (r *FeatureRepository) CreateIfNotExists(ctx context.Context, feature *models.Feature) (*models.Feature, bool, error) {
	// Start transaction to prevent race conditions
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, created, err := r.createIfNotExistsWithTx(ctx, tx, feature)
	if err != nil {
		return nil, false, err
	}

	if !created {
		// Feature already existed; no need to commit
		return result, false, nil
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, true, nil
}

// createIfNotExistsWithTx performs the create-if-not-exists operation within a
// caller-provided transaction. This allows the service layer to own the transaction
// boundary when this operation must be composed with other atomic operations.
func (r *FeatureRepository) createIfNotExistsWithTx(ctx context.Context, tx *sql.Tx, feature *models.Feature) (*models.Feature, bool, error) {
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
		return fmt.Errorf("feature not found: %s: %w", oldKey, repoerr.ErrNotFound)
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
		return fmt.Errorf("feature not found with id %d: %w", featureID, repoerr.ErrNotFound)
	}

	return nil
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

// GetContextData retrieves the context data JSON string for a feature by its ID
func (r *FeatureRepository) GetContextData(ctx context.Context, featureID int64) (*string, error) {
	query := `SELECT context_data FROM features WHERE id = ?`
	var contextData *string
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&contextData)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feature not found with id %d: %w", featureID, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feature context data: %w", err)
	}
	return contextData, nil
}

// UpdateContextData updates the context data JSON string for a feature
func (r *FeatureRepository) UpdateContextData(ctx context.Context, featureID int64, contextData *string) error {
	query := `UPDATE features SET context_data = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, contextData, featureID)
	if err != nil {
		return fmt.Errorf("failed to update feature context data: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feature not found with id %d: %w", featureID, repoerr.ErrNotFound)
	}
	return nil
}

// UpdateStatus updates only the status field of a feature.
func (r *FeatureRepository) UpdateStatus(ctx context.Context, featureID int64, status models.FeatureStatus) (retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.UpdateStatus",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "features"),
			attribute.Int64("db.id", featureID),
			attribute.String("db.new_status", string(status)),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `UPDATE features SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, status, featureID)
	if err != nil {
		return fmt.Errorf("failed to update feature status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feature not found with id %d: %w", featureID, repoerr.ErrNotFound)
	}
	return nil
}

// UpdateStatusIfCurrent atomically updates feature status only when the current
// stored status still matches expectedStatus (case-insensitive).
func (r *FeatureRepository) UpdateStatusIfCurrent(ctx context.Context, featureID int64, expectedStatus models.FeatureStatus, newStatus models.FeatureStatus) (bool, error) {
	updated, err := dbconn.ConditionalStatusUpdate(ctx, r.db, "features", featureID, string(expectedStatus), string(newStatus), true)
	if err != nil {
		return false, fmt.Errorf("conditionally update feature status: %w", err)
	}
	return updated, nil
}

// GetByIDTx retrieves a feature by its ID within an existing transaction.
// Provides snapshot isolation for cascade idempotency checks (REQ-F-008):
// reading the entity inside the cascade transaction ensures the in-tx re-fetch
// reflects the most-recent committed state at the moment the transaction started.
func (r *FeatureRepository) GetByIDTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Feature, error) {
	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, context_data, size, created_at, updated_at
		FROM features
		WHERE id = ?
	`

	feature := &models.Feature{}
	err := tx.QueryRowContext(ctx, query, id).Scan(
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
		&feature.ContextData,
		&feature.Size,
		&feature.CreatedAt,
		&feature.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feature not found with id %d: %w", id, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feature by id in transaction: %w", err)
	}

	return feature, nil
}

// UpdateStatusTx updates a feature's status inside an existing transaction.
// Mirrors UpdateStatus() but accepts *sql.Tx so the cascade can own the transaction
// and roll back atomically across feature + epic updates.
// The agent and notes parameters are accepted for interface symmetry but are not
// written to the features table (which has no agent/notes columns); they are
// intended for caller use when writing accompanying history rows.
func (r *FeatureRepository) UpdateStatusTx(
	ctx context.Context,
	tx *sql.Tx,
	id int64,
	status string,
	_ *string, // agent — reserved for interface symmetry, not persisted to features table
	_ *string, // notes — reserved for interface symmetry, not persisted to features table
) error {
	query := `UPDATE features SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := tx.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update feature status in transaction: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feature not found with id %d: %w", id, repoerr.ErrNotFound)
	}
	return nil
}

// CascadeStatusToTasks updates the status of all child tasks to match a target task status
// Used when --force is specified to override workflow validation
func (r *FeatureRepository) CascadeStatusToTasks(ctx context.Context, featureID int64, targetTaskStatus models.TaskStatus) error {
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	historyQuery := `
		INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced)
		SELECT id, status, ?, ?, ?, ?
		FROM tasks
		WHERE feature_id = ? AND status <> ?
	`
	agent := "shark-cli"
	notes := "Force-completed via feature cascade"
	if _, err := tx.ExecContext(ctx, historyQuery, targetTaskStatus, agent, notes, true, featureID, targetTaskStatus); err != nil {
		return fmt.Errorf("failed to create cascade task history: %w", err)
	}

	query := `UPDATE tasks SET status = ?`
	args := []interface{}{targetTaskStatus}
	if targetTaskStatus == models.TaskStatus("completed") {
		query += `, completed_at = COALESCE(completed_at, CURRENT_TIMESTAMP)`
	}
	query += ` WHERE feature_id = ? AND status <> ?`
	args = append(args, featureID, targetTaskStatus)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to cascade status to tasks: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the number of tasks updated (optional, for debugging)
	_ = rows

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit task status cascade: %w", err)
	}

	return nil
}

// FeatureDisplayDataRaw holds the raw JSON strings from the feature_display_data view.
// The service layer is responsible for unmarshaling these into domain types.
type FeatureDisplayDataRaw struct {
	TasksJSON         string
	TaskBreakdownJSON string
	DocumentsJSON     string
	NotesJSON         string
}

// GetFeatureDisplayDataRaw fetches all display data for a feature in a single query
// using the feature_display_data view. Returns raw JSON strings that the service
// layer unmarshals into domain types.
func (r *FeatureRepository) GetFeatureDisplayDataRaw(ctx context.Context, featureID int64) (*FeatureDisplayDataRaw, error) {
	query := `SELECT tasks_json, task_breakdown_json, documents_json, notes_json
		FROM feature_display_data WHERE id = ?`

	raw := &FeatureDisplayDataRaw{}
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(
		&raw.TasksJSON,
		&raw.TaskBreakdownJSON,
		&raw.DocumentsJSON,
		&raw.NotesJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query feature_display_data for feature %d: %w", featureID, err)
	}

	return raw, nil
}

// CountByStatus returns feature counts grouped by status.
// GetRecent returns the most recently created features, ordered by created_at DESC.
// limit must be positive; the caller (service) is responsible for bounds-checking.
// Returns an empty (non-nil) slice if no rows exist.
func (r *FeatureRepository) GetRecent(ctx context.Context, limit int) (_ []*models.Feature, retErr error) {
	ctx, span := tracer.Start(ctx, "FeatureRepository.GetRecent",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "features"),
			attribute.Int("db.limit", limit),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, epic_id, key, title, slug, description, status, COALESCE(status_override, 0) as status_override, progress_pct,
		       execution_order, file_path, context_data, size, created_at, updated_at
		FROM features
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent features: %w", err)
	}
	defer rows.Close()

	features := []*models.Feature{}
	for rows.Next() {
		f := &models.Feature{}
		err := rows.Scan(
			&f.ID,
			&f.EpicID,
			&f.Key,
			&f.Title,
			&f.Slug,
			&f.Description,
			&f.Status,
			&f.StatusOverride,
			&f.ProgressPct,
			&f.ExecutionOrder,
			&f.FilePath,
			&f.ContextData,
			&f.Size,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating features: %w", err)
	}

	return features, nil
}

func (r *FeatureRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM features GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("failed to count features by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan feature status count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}
