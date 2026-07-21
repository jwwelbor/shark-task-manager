package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
)

// ProgressMutationRepository provides the narrow transaction-aware data access
// used to keep feature progress caches and epic status rollups coherent after a
// progress-affecting write. It deliberately owns SQL only; workflow policy lives
// in services.AggregateMutationCoordinator.
type ProgressMutationRepository struct{}

// NewProgressMutationRepository creates a transaction-aware aggregate repository.
func NewProgressMutationRepository() *ProgressMutationRepository {
	return &ProgressMutationRepository{}
}

// GetFeatureForProgressTx reads the fields required to recalculate a feature.
func (r *ProgressMutationRepository) GetFeatureForProgressTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Feature, error) {
	feature := &models.Feature{}
	err := tx.QueryRowContext(ctx, `
		SELECT id, epic_id, key, status, COALESCE(status_override, 0), progress_pct
		FROM features WHERE id = ?`, id,
	).Scan(&feature.ID, &feature.EpicID, &feature.Key, &feature.Status, &feature.StatusOverride, &feature.ProgressPct)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("feature not found with id %d: %w", id, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get feature for progress: %w", err)
	}
	return feature, nil
}

// GetTaskStatusBreakdownTx returns each task status count for a feature.
func (r *ProgressMutationRepository) GetTaskStatusBreakdownTx(ctx context.Context, tx *sql.Tx, featureID int64) (map[models.TaskStatus]int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT status, COUNT(*) FROM tasks WHERE feature_id = ? GROUP BY status`, featureID)
	if err != nil {
		return nil, fmt.Errorf("get task status breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := make(map[models.TaskStatus]int)
	for rows.Next() {
		var status models.TaskStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan task status breakdown: %w", err)
		}
		breakdown[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task status breakdown: %w", err)
	}
	return breakdown, nil
}

// UpdateFeatureProgressAndStatusTx writes the coherent cached progress and status.
func (r *ProgressMutationRepository) UpdateFeatureProgressAndStatusTx(ctx context.Context, tx *sql.Tx, featureID int64, progress float64, status models.FeatureStatus) error {
	result, err := tx.ExecContext(ctx, `UPDATE features SET progress_pct = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, progress, status, featureID)
	if err != nil {
		return fmt.Errorf("update feature progress: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get feature progress update rows: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("feature not found with id %d: %w", featureID, repoerr.ErrNotFound)
	}
	return nil
}

// GetEpicForStatusTx reads the current epic status for rollup calculation.
func (r *ProgressMutationRepository) GetEpicForStatusTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Epic, error) {
	epic := &models.Epic{}
	err := tx.QueryRowContext(ctx, `SELECT id, key, status FROM epics WHERE id = ?`, id).Scan(&epic.ID, &epic.Key, &epic.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("epic not found with id %d: %w", id, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get epic for status: %w", err)
	}
	return epic, nil
}

// GetFeatureStatusBreakdownTx returns child feature status counts for an epic.
func (r *ProgressMutationRepository) GetFeatureStatusBreakdownTx(ctx context.Context, tx *sql.Tx, epicID int64) (map[models.FeatureStatus]int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT status, COUNT(*) FROM features WHERE epic_id = ? GROUP BY status`, epicID)
	if err != nil {
		return nil, fmt.Errorf("get feature status breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := make(map[models.FeatureStatus]int)
	for rows.Next() {
		var status models.FeatureStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan feature status breakdown: %w", err)
		}
		breakdown[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature status breakdown: %w", err)
	}
	return breakdown, nil
}

// UpdateEpicStatusTx writes a changed derived epic status.
func (r *ProgressMutationRepository) UpdateEpicStatusTx(ctx context.Context, tx *sql.Tx, epicID int64, status models.EpicStatus) error {
	result, err := tx.ExecContext(ctx, `UPDATE epics SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, epicID)
	if err != nil {
		return fmt.Errorf("update epic status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get epic status update rows: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("epic not found with id %d: %w", epicID, repoerr.ErrNotFound)
	}
	return nil
}

// GetLastNonTerminalStatusTx returns the most recent non-terminal status for an
// entity using the caller-owned aggregate transaction.
func (r *ProgressMutationRepository) GetLastNonTerminalStatusTx(
	ctx context.Context,
	tx *sql.Tx,
	entityType models.EntityType,
	entityID int64,
	terminalStatuses []string,
) (string, bool, error) {
	query := `
		SELECT to_status FROM entity_history
		WHERE entity_type = ? AND entity_id = ?`
	args := []interface{}{entityType, entityID}
	if len(terminalStatuses) > 0 {
		placeholders := make([]string, len(terminalStatuses))
		for i, status := range terminalStatuses {
			placeholders[i] = "?"
			args = append(args, status)
		}
		query += fmt.Sprintf(" AND to_status NOT IN (%s)", strings.Join(placeholders, ", "))
	}
	query += " ORDER BY changed_at DESC LIMIT 1"

	var status string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("get last non-terminal status: %w", err)
	}
	return status, true, nil
}

// CreateEntityHistoryTx records an aggregate-derived status change inside the
// caller-owned transaction.
func (r *ProgressMutationRepository) CreateEntityHistoryTx(ctx context.Context, tx *sql.Tx, history *models.EntityHistory) error {
	if err := history.Validate(); err != nil {
		return fmt.Errorf("validate entity history: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO entity_history (
			entity_type, entity_id, from_status, to_status,
			changed_by, notes, forced, rejection_reason, changed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		history.EntityType,
		history.EntityID,
		history.FromStatus,
		history.ToStatus,
		history.ChangedBy,
		history.Notes,
		history.Forced,
		history.RejectionReason,
		dbconn.FormatTime(history.ChangedAt),
	)
	if err != nil {
		return fmt.Errorf("create entity history: %w", err)
	}
	history.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get entity history id: %w", err)
	}
	return nil
}
