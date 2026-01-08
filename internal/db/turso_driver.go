package db

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// TursoDriver implements the Database interface for Turso (libSQL)
type TursoDriver struct {
	db *sql.DB
}

// NewTursoDriver creates a new Turso database driver
func NewTursoDriver() *TursoDriver {
	return &TursoDriver{}
}

// Connect opens a connection to a Turso database
// The DSN should include authToken: "libsql://db.turso.io?authToken=xxx"
func (t *TursoDriver) Connect(ctx context.Context, dsn string) error {
	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return err
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}

	t.db = db
	return nil
}

// Close closes the database connection
func (t *TursoDriver) Close() error {
	if t.db != nil {
		return t.db.Close()
	}
	return nil
}

// Ping verifies the database connection is alive
func (t *TursoDriver) Ping(ctx context.Context) error {
	if t.db == nil {
		return sql.ErrConnDone
	}
	return t.db.PingContext(ctx)
}

// Query executes a query that returns rows
func (t *TursoDriver) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	if t.db == nil {
		return nil, sql.ErrConnDone
	}
	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

// QueryRow executes a query that returns at most one row
func (t *TursoDriver) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	if t.db == nil {
		return &sqlRow{row: nil}
	}
	row := t.db.QueryRowContext(ctx, query, args...)
	return &sqlRow{row: row}
}

// Exec executes a query without returning rows
func (t *TursoDriver) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	if t.db == nil {
		return nil, sql.ErrConnDone
	}
	result, err := t.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlResult{result: result}, nil
}

// Begin starts a new transaction
func (t *TursoDriver) Begin(ctx context.Context) (Tx, error) {
	if t.db == nil {
		return nil, sql.ErrConnDone
	}
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx: tx}, nil
}

// DriverName returns the name of this driver
func (t *TursoDriver) DriverName() string {
	return "libsql"
}

// GetSQLDB returns the underlying *sql.DB for backward compatibility
// This is needed until repositories are updated to use Database interface
func (t *TursoDriver) GetSQLDB() (*sql.DB, error) {
	if t.db == nil {
		return nil, sql.ErrConnDone
	}
	return t.db, nil
}

// BuildTursoConnectionString builds a Turso connection string with auth token
// Example: BuildTursoConnectionString("libsql://db.turso.io", "token")
//
//	returns "libsql://db.turso.io?authToken=token"
func BuildTursoConnectionString(url, authToken string) string {
	if authToken == "" {
		return url
	}

	// Check if URL already has query parameters
	if strings.Contains(url, "?") {
		return url + "&authToken=" + authToken
	}
	return url + "?authToken=" + authToken
}
