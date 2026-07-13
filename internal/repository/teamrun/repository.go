// Package teamrun persists the normalized E38 team-run ledger.
package teamrun

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
)

const transactionAttempts = 3

// ErrDuplicateMembership identifies a second membership for the same run and
// typed child identity.
var ErrDuplicateMembership = errors.New("team-run item membership already exists")

// TeamRun is the SQL-facing durable run record. Domain validation belongs in
// internal/team; this package only stores the already validated snapshot.
type TeamRun struct {
	ID               int64
	RootKey          string
	RootType         string
	Status           string
	ExecutionMode    string
	ConcurrencyLimit int
	PlanHash         string
	AggregateOutcome *string
	NextAction       *string
	RootSessionID    *string
	StartedAt        *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// TeamRunItem is the SQL-facing durable membership record.
type TeamRunItem struct {
	ID               int64
	TeamRunID        int64
	ChildKey         string
	ChildType        string
	Wave             int
	ExecutionOrder   int
	DependencyKeys   string
	PlannedRole      *string
	PlannedAction    *string
	PlannedAgentType *string
	PlannedProvider  *string
	PlannedModel     *string
	PlannedEffort    *string
	ItemStatus       string
	ClaimSessionID   *string
	WorkerSessionID  *string
	Outcome          *string
	SkipReason       *string
	Evidence         *string
	Attempt          int
	StartedAt        *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Repository handles pure SQL access to the team-run ledger.
type Repository struct {
	db *dbconn.DB
}

// NewRepository creates a team-run repository backed by db.
func NewRepository(db *dbconn.DB) *Repository {
	return &Repository{db: db}
}

// NewTeamRunRepository is an explicit constructor alias for callers that
// prefer the domain name in wiring code.
func NewTeamRunRepository(db *dbconn.DB) *Repository {
	return NewRepository(db)
}

// CreateRunWithItems inserts a complete run snapshot in one short transaction.
// A failed item insert rolls back both the run and all earlier items. Busy
// errors retry the whole transaction, never an individual row insert.
func (r *Repository) CreateRunWithItems(ctx context.Context, run *TeamRun, items []*TeamRunItem) error {
	if run == nil {
		return errors.New("team-run is nil")
	}

	var runID int64
	itemIDs := make([]int64, len(items))
	err := r.withTransactionRetry(ctx, func(tx *sql.Tx) error {
		var err error
		runID, err = insertRunTx(ctx, tx, run)
		if err != nil {
			return err
		}
		for i, item := range items {
			if item == nil {
				return errors.New("team-run item is nil")
			}
			itemIDs[i], err = insertItemTx(ctx, tx, runID, item)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("create team run with items: %w", err)
	}

	run.ID = runID
	for i, item := range items {
		item.ID = itemIDs[i]
		item.TeamRunID = runID
	}
	return nil
}

// CreateRun inserts a run without item rows. CreateRunWithItems is preferred
// when a plan snapshot is being confirmed.
func (r *Repository) CreateRun(ctx context.Context, run *TeamRun) error {
	return r.CreateRunWithItems(ctx, run, nil)
}

// CreateRunTx inserts a run in a caller-owned transaction.
func (r *Repository) CreateRunTx(ctx context.Context, tx *sql.Tx, run *TeamRun) error {
	id, err := insertRunTx(ctx, tx, run)
	if err != nil {
		return fmt.Errorf("create team run in transaction: %w", err)
	}
	run.ID = id
	return nil
}

// CreateItemTx inserts a membership row in a caller-owned transaction.
func (r *Repository) CreateItemTx(ctx context.Context, tx *sql.Tx, item *TeamRunItem) error {
	if item == nil {
		return errors.New("team-run item is nil")
	}
	id, err := insertItemTx(ctx, tx, item.TeamRunID, item)
	if err != nil {
		return fmt.Errorf("create team-run item in transaction: %w", err)
	}
	item.ID = id
	return nil
}

// GetRun retrieves a run by its stable ID.
func (r *Repository) GetRun(ctx context.Context, id int64) (*TeamRun, error) {
	const query = `
		SELECT id, root_key, root_type, status, execution_mode, concurrency_limit,
			plan_hash, aggregate_outcome, next_action, root_session_id, started_at,
			completed_at, created_at, updated_at
		FROM team_runs WHERE id = ?`
	run, err := scanRun(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("team run %d not found: %w", id, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get team run %d: %w", id, err)
	}
	return run, nil
}

// ListItems returns a run's items in deterministic wave/order/key order.
func (r *Repository) ListItems(ctx context.Context, runID int64) ([]*TeamRunItem, error) {
	const query = `
		SELECT id, team_run_id, child_key, child_type, wave, execution_order,
			dependency_keys, planned_role, planned_action, planned_agent_type,
			planned_provider, planned_model, planned_effort, item_status,
			claim_session_id, worker_session_id, outcome, skip_reason, evidence,
			attempt, started_at, completed_at, created_at, updated_at
		FROM team_run_items
		WHERE team_run_id = ?
		ORDER BY wave, execution_order, child_type, child_key, id`
	rows, err := r.db.QueryContext(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("list team-run items for run %d: %w", runID, err)
	}
	defer rows.Close()

	items := make([]*TeamRunItem, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan team-run item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team-run items for run %d: %w", runID, err)
	}
	return items, nil
}

// UpdateRun persists the mutable run snapshot fields and refreshes updated_at.
func (r *Repository) UpdateRun(ctx context.Context, run *TeamRun) error {
	if run == nil {
		return errors.New("team-run is nil")
	}
	err := r.withTransactionRetry(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE team_runs
			SET status = ?, execution_mode = ?, concurrency_limit = ?, plan_hash = ?,
				aggregate_outcome = ?, next_action = ?, root_session_id = ?,
				started_at = ?, completed_at = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			run.Status, run.ExecutionMode, run.ConcurrencyLimit, run.PlanHash,
			stringValue(run.AggregateOutcome), stringValue(run.NextAction), stringValue(run.RootSessionID),
			sqlTimeValue(run.StartedAt), sqlTimeValue(run.CompletedAt), run.ID)
		if err != nil {
			return fmt.Errorf("update team run %d: %w", run.ID, err)
		}
		if affected, err := result.RowsAffected(); err != nil {
			return fmt.Errorf("update team run %d rows affected: %w", run.ID, err)
		} else if affected == 0 {
			return fmt.Errorf("team run %d not found: %w", run.ID, repoerr.ErrNotFound)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update team run %d: %w", run.ID, err)
	}
	return nil
}

// UpdateItem persists a membership row and refreshes updated_at.
func (r *Repository) UpdateItem(ctx context.Context, item *TeamRunItem) error {
	if item == nil {
		return errors.New("team-run item is nil")
	}
	err := r.withTransactionRetry(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE team_run_items
			SET wave = ?, execution_order = ?, dependency_keys = ?, planned_role = ?,
				planned_action = ?, planned_agent_type = ?, planned_provider = ?,
				planned_model = ?, planned_effort = ?, item_status = ?,
				claim_session_id = ?, worker_session_id = ?, outcome = ?, skip_reason = ?,
				evidence = ?, attempt = ?, started_at = ?, completed_at = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			item.Wave, item.ExecutionOrder, item.DependencyKeys,
			stringValue(item.PlannedRole), stringValue(item.PlannedAction), stringValue(item.PlannedAgentType),
			stringValue(item.PlannedProvider), stringValue(item.PlannedModel), stringValue(item.PlannedEffort),
			item.ItemStatus, stringValue(item.ClaimSessionID), stringValue(item.WorkerSessionID),
			stringValue(item.Outcome), stringValue(item.SkipReason), stringValue(item.Evidence), item.Attempt,
			sqlTimeValue(item.StartedAt), sqlTimeValue(item.CompletedAt), item.ID)
		if err != nil {
			return fmt.Errorf("update team-run item %d: %w", item.ID, err)
		}
		if affected, err := result.RowsAffected(); err != nil {
			return fmt.Errorf("update team-run item %d rows affected: %w", item.ID, err)
		} else if affected == 0 {
			return fmt.Errorf("team-run item %d not found: %w", item.ID, repoerr.ErrNotFound)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update team-run item %d: %w", item.ID, err)
	}
	return nil
}

func insertRunTx(ctx context.Context, tx *sql.Tx, run *TeamRun) (int64, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO team_runs (
			root_key, root_type, status, execution_mode, concurrency_limit, plan_hash,
			aggregate_outcome, next_action, root_session_id, started_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.RootKey, run.RootType, run.Status, run.ExecutionMode, run.ConcurrencyLimit,
		run.PlanHash, stringValue(run.AggregateOutcome), stringValue(run.NextAction),
		stringValue(run.RootSessionID), sqlTimeValue(run.StartedAt), sqlTimeValue(run.CompletedAt))
	if err != nil {
		return 0, fmt.Errorf("insert team run: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert team run last insert id: %w", err)
	}
	return id, nil
}

func insertItemTx(ctx context.Context, tx *sql.Tx, runID int64, item *TeamRunItem) (int64, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO team_run_items (
			team_run_id, child_key, child_type, wave, execution_order, dependency_keys,
			planned_role, planned_action, planned_agent_type, planned_provider,
			planned_model, planned_effort, item_status, claim_session_id,
			worker_session_id, outcome, skip_reason, evidence, attempt, started_at,
			completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID, item.ChildKey, item.ChildType, item.Wave, item.ExecutionOrder, item.DependencyKeys,
		stringValue(item.PlannedRole), stringValue(item.PlannedAction), stringValue(item.PlannedAgentType),
		stringValue(item.PlannedProvider), stringValue(item.PlannedModel), stringValue(item.PlannedEffort),
		item.ItemStatus, stringValue(item.ClaimSessionID), stringValue(item.WorkerSessionID),
		stringValue(item.Outcome), stringValue(item.SkipReason), stringValue(item.Evidence), item.Attempt,
		sqlTimeValue(item.StartedAt), sqlTimeValue(item.CompletedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateMembership
		}
		return 0, fmt.Errorf("insert team-run item %s/%s: %w", item.ChildType, item.ChildKey, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert team-run item last insert id: %w", err)
	}
	return id, nil
}

func (r *Repository) withTransactionRetry(ctx context.Context, fn func(*sql.Tx) error) error {
	for attempt := 0; attempt < transactionAttempts; attempt++ {
		tx, err := r.db.BeginTxContext(ctx)
		if err != nil {
			if !isBusyError(err) || attempt == transactionAttempts-1 {
				return err
			}
			if err := waitForRetry(ctx, attempt); err != nil {
				return err
			}
			continue
		}

		err = fn(tx)
		if err != nil {
			_ = tx.Rollback()
			if !isBusyError(err) || attempt == transactionAttempts-1 {
				return err
			}
			if err := waitForRetry(ctx, attempt); err != nil {
				return err
			}
			continue
		}
		if err := tx.Commit(); err != nil {
			// Do not retry a commit error: a driver may have committed before
			// reporting the error, and retrying could duplicate a run.
			return err
		}
		return nil
	}
	return errors.New("team-run transaction retry limit exceeded")
}

func waitForRetry(ctx context.Context, attempt int) error {
	timer := time.NewTimer(time.Duration(25*(1<<attempt)) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func scanRun(scanner interface{ Scan(...any) error }) (*TeamRun, error) {
	run := &TeamRun{}
	var startedAt, completedAt nullableTime
	var createdAt, updatedAt timeScanner
	createdAt.target = &run.CreatedAt
	updatedAt.target = &run.UpdatedAt
	err := scanner.Scan(
		&run.ID, &run.RootKey, &run.RootType, &run.Status, &run.ExecutionMode,
		&run.ConcurrencyLimit, &run.PlanHash, optionalStringValue(&run.AggregateOutcome),
		optionalStringValue(&run.NextAction), optionalStringValue(&run.RootSessionID),
		&startedAt, &completedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	run.StartedAt = startedAt.value
	run.CompletedAt = completedAt.value
	return run, nil
}

func scanItem(scanner interface{ Scan(...any) error }) (*TeamRunItem, error) {
	item := &TeamRunItem{}
	var startedAt, completedAt nullableTime
	var createdAt, updatedAt timeScanner
	createdAt.target = &item.CreatedAt
	updatedAt.target = &item.UpdatedAt
	err := scanner.Scan(
		&item.ID, &item.TeamRunID, &item.ChildKey, &item.ChildType, &item.Wave,
		&item.ExecutionOrder, &item.DependencyKeys, optionalStringValue(&item.PlannedRole),
		optionalStringValue(&item.PlannedAction), optionalStringValue(&item.PlannedAgentType),
		optionalStringValue(&item.PlannedProvider), optionalStringValue(&item.PlannedModel),
		optionalStringValue(&item.PlannedEffort), &item.ItemStatus,
		optionalStringValue(&item.ClaimSessionID), optionalStringValue(&item.WorkerSessionID),
		optionalStringValue(&item.Outcome), optionalStringValue(&item.SkipReason),
		optionalStringValue(&item.Evidence), &item.Attempt, &startedAt, &completedAt,
		&createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	item.StartedAt = startedAt.value
	item.CompletedAt = completedAt.value
	return item, nil
}

type optionalStringScanner struct {
	target **string
}

func optionalStringValue(target **string) optionalStringScanner {
	return optionalStringScanner{target: target}
}

func (s optionalStringScanner) Scan(src any) error {
	if src == nil {
		*s.target = nil
		return nil
	}
	var value string
	switch v := src.(type) {
	case string:
		value = v
	case []byte:
		value = string(v)
	default:
		return fmt.Errorf("cannot scan %T as nullable string", src)
	}
	*s.target = &value
	return nil
}

type nullableTime struct{ value *time.Time }

func (t *nullableTime) Scan(src any) error {
	if src == nil {
		t.value = nil
		return nil
	}
	value, err := parseTime(src)
	if err != nil {
		return err
	}
	t.value = &value
	return nil
}

type timeScanner struct{ target *time.Time }

func (t *timeScanner) Scan(src any) error {
	value, err := parseTime(src)
	if err != nil {
		return err
	}
	*t.target = value
	return nil
}

func parseTime(src any) (time.Time, error) {
	switch value := src.(type) {
	case time.Time:
		return value, nil
	case string:
		return parseTimeString(value)
	case []byte:
		return parseTimeString(string(value))
	default:
		return time.Time{}, fmt.Errorf("cannot scan %T as time", src)
	}
}

func parseTimeString(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid timestamp %q", value)
}

func stringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func sqlTimeValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return dbconn.FormatTime(*value)
}

func isBusyError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "sqlite_busy") ||
		strings.Contains(message, "sqlite_locked")
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "constraint failed: unique") ||
		strings.Contains(message, "sqlite_constraint")
}
