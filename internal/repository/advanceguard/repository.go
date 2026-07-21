// Package advanceguard persists replay-protection records for guarded advances.
package advanceguard

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

var ErrAlreadyConsumed = models.ErrAdvanceGuardAlreadyConsumed

// Repository persists replay-protection records for guarded status advances.
type Repository struct {
	db *dbconn.DB
}

// NewRepository creates a repository backed by db.
func NewRepository(db *dbconn.DB) *Repository {
	return &Repository{db: db}
}

// WasConsumed reports whether the same entity/session/from_status/outcome tuple
// was already recorded.
func (r *Repository) WasConsumed(ctx context.Context, entityType string, entityID int64, sessionID, fromStatus, outcome string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM advance_guard_consumptions
		WHERE entity_type = ? AND entity_id = ? AND session_id = ? AND from_status = ? AND outcome = ?
	`
	var count int
	if err := r.db.QueryRowContext(ctx, query, entityType, entityID, sessionID, fromStatus, outcome).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to query guarded advance consumption: %w", err)
	}
	return count > 0, nil
}

// RecordConsumed stores a consumed guarded advance. It returns ErrAlreadyConsumed
// when the same entity/session/from_status/outcome combination was already used.
func (r *Repository) RecordConsumed(ctx context.Context, entityType string, entityID int64, sessionID, fromStatus, outcome string) error {
	query := `
		INSERT INTO advance_guard_consumptions (
			entity_type, entity_id, session_id, from_status, outcome
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(entity_type, entity_id, session_id, from_status, outcome) DO NOTHING
	`

	result, err := r.db.ExecContext(ctx, query, entityType, entityID, sessionID, fromStatus, outcome)
	if err != nil {
		return fmt.Errorf("failed to record guarded advance: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get guarded advance rows affected: %w", err)
	}
	if rows == 0 {
		return ErrAlreadyConsumed
	}
	return nil
}

// DeleteConsumed removes a previously recorded consumption. Used to compensate
// when RecordConsumed succeeds but the guarded status update that followed it
// fails (e.g. a genuine concurrent race), so a phantom consumption doesn't
// block a later legitimate replay of the same session/from_status/outcome.
func (r *Repository) DeleteConsumed(ctx context.Context, entityType string, entityID int64, sessionID, fromStatus, outcome string) error {
	query := `
		DELETE FROM advance_guard_consumptions
		WHERE entity_type = ? AND entity_id = ? AND session_id = ? AND from_status = ? AND outcome = ?
	`
	if _, err := r.db.ExecContext(ctx, query, entityType, entityID, sessionID, fromStatus, outcome); err != nil {
		return fmt.Errorf("failed to delete guarded advance consumption: %w", err)
	}
	return nil
}

// WasConsumedWithTx reports guarded-advance consumption inside tx.
func (r *Repository) WasConsumedWithTx(ctx context.Context, tx *sql.Tx, entityType string, entityID int64, sessionID, fromStatus, outcome string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM advance_guard_consumptions
		WHERE entity_type = ? AND entity_id = ? AND session_id = ? AND from_status = ? AND outcome = ?
	`
	var count int
	if err := tx.QueryRowContext(ctx, query, entityType, entityID, sessionID, fromStatus, outcome).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to query guarded advance consumption in transaction: %w", err)
	}
	return count > 0, nil
}

// RecordConsumedWithTx stores a guarded-advance consumption inside tx.
func (r *Repository) RecordConsumedWithTx(ctx context.Context, tx *sql.Tx, entityType string, entityID int64, sessionID, fromStatus, outcome string) error {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO advance_guard_consumptions (
			entity_type, entity_id, session_id, from_status, outcome
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(entity_type, entity_id, session_id, from_status, outcome) DO NOTHING`,
		entityType, entityID, sessionID, fromStatus, outcome,
	)
	if err != nil {
		return fmt.Errorf("failed to record guarded advance in transaction: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get guarded advance rows affected in transaction: %w", err)
	}
	if rows == 0 {
		return ErrAlreadyConsumed
	}
	return nil
}
