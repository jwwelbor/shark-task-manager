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
