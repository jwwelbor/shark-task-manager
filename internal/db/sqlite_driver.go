package db

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDriver implements the Database interface for SQLite
type SQLiteDriver struct {
	db *sql.DB
}

// NewSQLiteDriver creates a new SQLite database driver
func NewSQLiteDriver() *SQLiteDriver {
	return &SQLiteDriver{}
}

// Connect opens a connection to a SQLite database
func (s *SQLiteDriver) Connect(ctx context.Context, dsn string) error {
	// Add foreign_keys parameter if not present
	if dsn != "" && !contains(dsn, "?") {
		dsn += "?_foreign_keys=on"
	} else if dsn != "" && !contains(dsn, "_foreign_keys") {
		dsn += "&_foreign_keys=on"
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}

	// Configure SQLite for optimal performance
	if err := s.configureSQLite(db); err != nil {
		db.Close()
		return err
	}

	s.db = db
	return nil
}

// Close closes the database connection
func (s *SQLiteDriver) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ping verifies the database connection is alive
func (s *SQLiteDriver) Ping(ctx context.Context) error {
	if s.db == nil {
		return sql.ErrConnDone
	}
	return s.db.PingContext(ctx)
}

// Query executes a query that returns rows
func (s *SQLiteDriver) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	if s.db == nil {
		return nil, sql.ErrConnDone
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

// QueryRow executes a query that returns at most one row
func (s *SQLiteDriver) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	if s.db == nil {
		return &sqlRow{row: nil}
	}
	row := s.db.QueryRowContext(ctx, query, args...)
	return &sqlRow{row: row}
}

// Exec executes a query without returning rows
func (s *SQLiteDriver) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	if s.db == nil {
		return nil, sql.ErrConnDone
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlResult{result: result}, nil
}

// Begin starts a new transaction
func (s *SQLiteDriver) Begin(ctx context.Context) (Tx, error) {
	if s.db == nil {
		return nil, sql.ErrConnDone
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx: tx}, nil
}

// DriverName returns the name of this driver
func (s *SQLiteDriver) DriverName() string {
	return "sqlite3"
}

// GetSQLDB returns the underlying *sql.DB for backward compatibility
// This is needed until repositories are updated to use Database interface
func (s *SQLiteDriver) GetSQLDB() (*sql.DB, error) {
	if s.db == nil {
		return nil, sql.ErrConnDone
	}
	return s.db, nil
}

// configureSQLite sets SQLite PRAGMA settings for optimal operation
func (s *SQLiteDriver) configureSQLite(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",       // Enable foreign key constraints
		"PRAGMA journal_mode = WAL;",      // Use Write-Ahead Logging for better concurrency
		"PRAGMA busy_timeout = 5000;",     // 5 second timeout for locks
		"PRAGMA synchronous = NORMAL;",    // Balance safety and performance
		"PRAGMA cache_size = -64000;",     // 64MB cache
		"PRAGMA temp_store = MEMORY;",     // Store temp tables in memory
		"PRAGMA mmap_size = 30000000000;", // Use memory-mapped I/O
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return err
		}
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fkEnabled); err != nil {
		return err
	}
	if fkEnabled != 1 {
		return sql.ErrConnDone // Use existing error type
	}

	return nil
}

// Adapter types that wrap sql.* types to implement our interfaces

type sqlRows struct {
	rows *sql.Rows
}

func (r *sqlRows) Close() error                   { return r.rows.Close() }
func (r *sqlRows) Next() bool                     { return r.rows.Next() }
func (r *sqlRows) Scan(dest ...interface{}) error { return r.rows.Scan(dest...) }
func (r *sqlRows) Err() error                     { return r.rows.Err() }
func (r *sqlRows) Columns() ([]string, error)     { return r.rows.Columns() }
func (r *sqlRows) ColumnTypes() ([]ColumnType, error) {
	types, err := r.rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	result := make([]ColumnType, len(types))
	for i, t := range types {
		result[i] = &sqlColumnType{ct: t}
	}
	return result, nil
}
func (r *sqlRows) NextResultSet() bool { return r.rows.NextResultSet() }

type sqlRow struct {
	row *sql.Row
}

func (r *sqlRow) Scan(dest ...interface{}) error {
	if r.row == nil {
		return sql.ErrNoRows
	}
	return r.row.Scan(dest...)
}

type sqlTx struct {
	tx *sql.Tx
}

func (t *sqlTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

func (t *sqlTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	row := t.tx.QueryRowContext(ctx, query, args...)
	return &sqlRow{row: row}
}

func (t *sqlTx) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlResult{result: result}, nil
}

func (t *sqlTx) Commit() error   { return t.tx.Commit() }
func (t *sqlTx) Rollback() error { return t.tx.Rollback() }

type sqlResult struct {
	result sql.Result
}

func (r *sqlResult) LastInsertId() (int64, error) { return r.result.LastInsertId() }
func (r *sqlResult) RowsAffected() (int64, error) { return r.result.RowsAffected() }

type sqlColumnType struct {
	ct *sql.ColumnType
}

func (c *sqlColumnType) Name() string                      { return c.ct.Name() }
func (c *sqlColumnType) DatabaseTypeName() string          { return c.ct.DatabaseTypeName() }
func (c *sqlColumnType) Length() (int64, bool)             { return c.ct.Length() }
func (c *sqlColumnType) DecimalSize() (int64, int64, bool) { return c.ct.DecimalSize() }
func (c *sqlColumnType) Nullable() (bool, bool)            { return c.ct.Nullable() }
func (c *sqlColumnType) ScanType() interface{}             { return c.ct.ScanType() }

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
