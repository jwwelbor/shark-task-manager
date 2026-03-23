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
)

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
