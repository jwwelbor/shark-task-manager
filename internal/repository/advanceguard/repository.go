package advanceguard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

var ErrAlreadyConsumed = errors.New("guarded advance already consumed")

// Repository persists replay-protection records for guarded status advances.
type Repository struct {
	db *dbconn.DB
}

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
	`

	_, err := r.db.ExecContext(ctx, query, entityType, entityID, sessionID, fromStatus, outcome)
	if err == nil {
		return nil
	}
	if isUniqueConstraint(err) {
		return ErrAlreadyConsumed
	}
	return fmt.Errorf("failed to record guarded advance: %w", err)
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "constraint failed: UNIQUE")
}
