package entityhistory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// ListRecentAcrossEntitiesOptions configures the ListRecentAcrossEntities query.
// All fields are optional — zero values mean "no filter / use default".
type ListRecentAcrossEntitiesOptions struct {
	// EntityType restricts results to a single entity type (e.g. "task", "epic").
	// Empty string means all entity types are included.
	EntityType string

	// Since restricts results to history rows recorded after this time.
	// nil means no lower-bound time filter.
	Since *time.Time

	// Limit caps the number of rows returned, ordered DESC by changed_at.
	// A value of 0 or negative returns an empty slice (caller is responsible for
	// clamping to a useful value before calling). Max meaningful value is
	// effectively unlimited; callers should apply their own cap.
	Limit int
}

// RecentActivityRow is a single result row from ListRecentAcrossEntities.
// It combines columns from the entity_history table with the key and title
// of the parent entity, resolved via INNER JOIN.
type RecentActivityRow struct {
	// EntityType identifies the kind of entity (e.g. "epic", "feature", "task", "bug", "change").
	EntityType string

	// Key is the business key of the parent entity (e.g. "E07", "E07-F01", "T-E07-F01-001").
	Key string

	// Title is the human-readable title of the parent entity.
	Title string

	// FromStatus is the status the entity transitioned from (may be empty for initial creation).
	FromStatus string

	// ToStatus is the status the entity transitioned to.
	ToStatus string

	// ChangedAt is when the status change was recorded.
	ChangedAt time.Time
}

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

// ListRecentAcrossEntities returns recent status-change activity across all entity
// types (epic, feature, task, bug, change) in a single SQL round-trip.
//
// Each SELECT arm in the UNION ALL emits: entity_type (literal string), key,
// title, from_status, to_status, changed_at. An INNER JOIN to the parent entity
// table ensures orphaned history rows (whose parent was deleted) are omitted.
//
// Results are ordered DESC by changed_at (most recent first) and capped by
// opts.Limit. When opts.Limit is 0 or negative, an empty slice is returned
// immediately without hitting the database.
func (r *EntityHistoryRepository) ListRecentAcrossEntities(ctx context.Context, opts ListRecentAcrossEntitiesOptions) ([]*RecentActivityRow, error) {
	if opts.Limit <= 0 {
		return []*RecentActivityRow{}, nil
	}

	// Build the UNION ALL query. Each arm joins entity_history to its parent
	// entity table using INNER JOIN so orphaned rows are excluded.
	//
	// Columns in each SELECT arm (must match scan order below):
	//   entity_type, key, title, from_status, to_status, changed_at
	const unionQuery = `
		SELECT 'epic'    AS entity_type, e.key, e.title, eh.from_status, eh.to_status, eh.changed_at
		FROM entity_history eh
		INNER JOIN epics e ON e.id = eh.entity_id AND eh.entity_type = 'epic'
		UNION ALL
		SELECT 'feature' AS entity_type, f.key, f.title, eh.from_status, eh.to_status, eh.changed_at
		FROM entity_history eh
		INNER JOIN features f ON f.id = eh.entity_id AND eh.entity_type = 'feature'
		UNION ALL
		SELECT 'task'    AS entity_type, t.key, t.title, eh.from_status, eh.to_status, eh.changed_at
		FROM entity_history eh
		INNER JOIN tasks t ON t.id = eh.entity_id AND eh.entity_type = 'task'
		UNION ALL
		SELECT 'bug'     AS entity_type, b.key, b.title, eh.from_status, eh.to_status, eh.changed_at
		FROM entity_history eh
		INNER JOIN bugs b ON b.id = eh.entity_id AND eh.entity_type = 'bug'
		UNION ALL
		SELECT 'change'  AS entity_type, cc.key, cc.title, eh.from_status, eh.to_status, eh.changed_at
		FROM entity_history eh
		INNER JOIN change_cards cc ON cc.id = eh.entity_id AND eh.entity_type = 'change'
	`

	// Wrap in an outer SELECT so we can apply WHERE filters and ORDER BY / LIMIT
	// after the UNION ALL is evaluated.
	var sb strings.Builder
	sb.WriteString("SELECT entity_type, key, title, from_status, to_status, changed_at FROM (")
	sb.WriteString(unionQuery)
	sb.WriteString(") AS combined WHERE 1=1")

	args := make([]interface{}, 0, 3)

	if opts.EntityType != "" {
		sb.WriteString(" AND entity_type = ?")
		args = append(args, opts.EntityType)
	}

	if opts.Since != nil {
		sb.WriteString(" AND changed_at > ?")
		args = append(args, opts.Since.UTC())
	}

	sb.WriteString(" ORDER BY changed_at DESC LIMIT ?")
	args = append(args, opts.Limit)

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent activity across entities: %w", err)
	}
	defer rows.Close()

	var results []*RecentActivityRow
	for rows.Next() {
		row := &RecentActivityRow{}
		var fromStatus sql.NullString
		err := rows.Scan(
			&row.EntityType,
			&row.Key,
			&row.Title,
			&fromStatus,
			&row.ToStatus,
			&row.ChangedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recent activity row: %w", err)
		}
		if fromStatus.Valid {
			row.FromStatus = fromStatus.String
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent activity rows: %w", err)
	}

	if results == nil {
		results = []*RecentActivityRow{}
	}

	return results, nil
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
