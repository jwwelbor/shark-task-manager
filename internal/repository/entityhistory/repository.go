package entityhistory

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// EntityHistoryRepository handles CRUD operations for the entity_history table.
// It provides polymorphic history recording and querying for all entity types
// (epic, feature, task, bug, change).
type EntityHistoryRepository struct {
	db *dbconn.DB
}

// NewEntityHistoryRepository creates a new EntityHistoryRepository.
func NewEntityHistoryRepository(db *dbconn.DB) *EntityHistoryRepository {
	return &EntityHistoryRepository{db: db}
}

// Create inserts a new entity history record into the entity_history table.
// It calls history.Validate() before the INSERT. On success, history.ID is
// set to the auto-generated row ID.
func (r *EntityHistoryRepository) Create(ctx context.Context, history *models.EntityHistory) error {
	if err := history.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO entity_history (
			entity_type, entity_id, from_status, to_status,
			changed_by, notes, forced, rejection_reason, changed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		history.EntityType,
		history.EntityID,
		history.FromStatus,
		history.ToStatus,
		history.ChangedBy,
		history.Notes,
		history.Forced,
		history.RejectionReason,
		history.ChangedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create entity history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	history.ID = id
	return nil
}

// ListByEntity retrieves all history records for a specific entity type and ID,
// ordered by changed_at DESC (most recent first).
func (r *EntityHistoryRepository) ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
	query := `
		SELECT id, entity_type, entity_id, from_status, to_status,
		       changed_by, notes, forced, rejection_reason, changed_at
		FROM entity_history
		WHERE entity_type = ? AND entity_id = ?
		ORDER BY changed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity history: %w", err)
	}
	defer rows.Close()

	histories, err := scanEntityHistories(rows)
	if err != nil {
		return nil, err
	}

	if histories == nil {
		histories = make([]*models.EntityHistory, 0)
	}

	return histories, nil
}

// scanEntityHistories scans rows into a slice of EntityHistory pointers.
// Shared helper for all List methods.
func scanEntityHistories(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]*models.EntityHistory, error) {
	var histories []*models.EntityHistory
	for rows.Next() {
		h := &models.EntityHistory{}
		err := rows.Scan(
			&h.ID,
			&h.EntityType,
			&h.EntityID,
			&h.FromStatus,
			&h.ToStatus,
			&h.ChangedBy,
			&h.Notes,
			&h.Forced,
			&h.RejectionReason,
			&h.ChangedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity history: %w", err)
		}
		histories = append(histories, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity history: %w", err)
	}

	return histories, nil
}
