// Package dbconn provides the shared database connection type used by all
// repository sub-packages. It is a thin wrapper around *sql.DB that adds
// convenience methods for transaction management.
//
// This package was extracted from the root repository package so that
// entity-specific sub-packages (task, feature, epic, note, worksession) can
// import the connection type without creating import cycles.
package dbconn

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// TimeFormat is the canonical layout for timestamp values bound as query
// parameters. It matches SQLite's preferred text datetime format (and the
// modernc driver's read-path parse formats), unlike the Go default
// time.Time.String() debug layout that drivers fall back to when handed a
// raw time.Time — which neither SQLite datetime() nor the driver read path
// can parse. Always store UTC.
const TimeFormat = "2006-01-02 15:04:05.999999999-07:00"

// FormatTime renders a timestamp in the canonical parameter layout (UTC).
// Repositories should bind FormatTime(t) instead of a raw time.Time so the
// stored text is identical across drivers (modernc sqlite, libsql/Turso).
func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeFormat)
}

// DB wraps the database connection for repositories.
type DB struct {
	*sql.DB
}

// NewDB creates a new DB instance.
func NewDB(db *sql.DB) *DB {
	return &DB{db}
}

// BeginTxContext starts a new transaction with context.
func (db *DB) BeginTxContext(ctx context.Context) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, nil)
}

// BeginTx starts a new transaction.
// Deprecated: use BeginTxContext.
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}

// ConditionalStatusUpdate atomically changes an entity's status only when it
// still matches expectedStatus. table must be one of Shark's fixed entity
// tables; keeping that allowlist here prevents identifiers from becoming a SQL
// interpolation surface while centralizing compare-and-swap semantics.
func ConditionalStatusUpdate(ctx context.Context, db *DB, table string, id int64, expectedStatus, newStatus string, touchUpdatedAt bool) (bool, error) {
	switch table {
	case "bugs", "change_cards", "epics", "features", "ideas", "sprints", "tech_debts":
	default:
		return false, fmt.Errorf("unsupported status table %q", table)
	}

	setClause := "status = ?"
	if touchUpdatedAt {
		setClause += ", updated_at = CURRENT_TIMESTAMP"
	}
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ? AND lower(status) = lower(?)", table, setClause)
	result, err := db.ExecContext(ctx, query, newStatus, id, expectedStatus)
	if err != nil {
		return false, fmt.Errorf("conditionally update status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("get conditional status update rows affected: %w", err)
	}
	return rows > 0, nil
}
