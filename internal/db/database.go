package db

import "context"

// Database interface abstracts database operations across different backends (SQLite, Turso)
// This allows repositories to work with any database implementation without modification
type Database interface {
	// Connection management
	Connect(ctx context.Context, dsn string) error
	Close() error
	Ping(ctx context.Context) error

	// Query operations - matches sql.DB API for compatibility
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)

	// Transaction support
	Begin(ctx context.Context) (Tx, error)

	// Metadata
	DriverName() string
}

// Tx represents a database transaction
// Matches sql.Tx API for compatibility with existing repository code
type Tx interface {
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Commit() error
	Rollback() error
}

// Rows represents the result of a query that returns multiple rows
// Matches sql.Rows API for compatibility
type Rows interface {
	Close() error
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
	Columns() ([]string, error)
	ColumnTypes() ([]ColumnType, error)
	NextResultSet() bool
}

// Row represents the result of a query that returns a single row
// Matches sql.Row API for compatibility
type Row interface {
	Scan(dest ...interface{}) error
}

// Result represents the result of an Exec operation
// Matches sql.Result API for compatibility
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// ColumnType represents metadata about a column
// Matches sql.ColumnType API for compatibility
type ColumnType interface {
	Name() string
	DatabaseTypeName() string
	Length() (length int64, ok bool)
	DecimalSize() (precision, scale int64, ok bool)
	Nullable() (nullable, ok bool)
	ScanType() interface{}
}
