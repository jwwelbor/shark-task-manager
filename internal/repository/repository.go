package repository

import (
	"context"
	"database/sql"
)

// DB wraps the database connection for repositories
type DB struct {
	*sql.DB
}

// NewDB creates a new DB instance
func NewDB(db *sql.DB) *DB {
	return &DB{db}
}

// BeginTxContext starts a new transaction with context
func (db *DB) BeginTxContext(ctx context.Context) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, nil)
}

// BeginTx starts a new transaction (deprecated: use BeginTxContext)
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}
