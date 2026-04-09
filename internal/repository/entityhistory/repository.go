package entityhistory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

// GetLastNonTerminalStatus returns the to_status of the most recent entity_history
// row for (entityType, entityID) where to_status is not in terminalStatuses.
//
// Returns (status, true, nil) on hit, ("", false, nil) on no rows, ("", false, err) on DB error.
//
// The caller-supplied terminalStatuses slice is used to build a parameterized NOT IN clause;
// no status names are hardcoded in this method.
func (r *EntityHistoryRepository) GetLastNonTerminalStatus(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	terminalStatuses []string,
) (string, bool, error) {
	// Build the NOT IN clause from the caller-supplied slice.
	// When terminalStatuses is empty, the WHERE clause has no NOT IN filter,
	// so all rows are considered non-terminal.
	var query string
	var args []interface{}

	if len(terminalStatuses) == 0 {
		query = `
			SELECT to_status FROM entity_history
			WHERE entity_type = ? AND entity_id = ?
			ORDER BY changed_at DESC LIMIT 1
		`
		args = []interface{}{entityType, entityID}
	} else {
		placeholders := make([]string, len(terminalStatuses))
		for i := range terminalStatuses {
			placeholders[i] = "?"
		}
		query = fmt.Sprintf(`
			SELECT to_status FROM entity_history
			WHERE entity_type = ? AND entity_id = ?
			  AND to_status NOT IN (%s)
			ORDER BY changed_at DESC LIMIT 1
		`, strings.Join(placeholders, ", "))
		args = make([]interface{}, 0, 2+len(terminalStatuses))
		args = append(args, entityType, entityID)
		for _, s := range terminalStatuses {
			args = append(args, s)
		}
	}

	var status string
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&status)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to get last non-terminal status: %w", err)
	}
	return status, true, nil
}

// CreateTx writes a new entity_history row inside an existing transaction.
// Mirrors Create() but accepts *sql.Tx for cascade-owned transaction use.
// Calls history.Validate() before the INSERT. On success, history.ID is set.
func (r *EntityHistoryRepository) CreateTx(
	ctx context.Context,
	tx *sql.Tx,
	history *models.EntityHistory,
) error {
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

	result, err := tx.ExecContext(ctx, query,
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
		return fmt.Errorf("failed to create entity history in transaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	history.ID = id
	return nil
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
