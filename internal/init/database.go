package init

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

// createDatabase creates database schema if it doesn't exist
// Returns true if database was created, false if already existed
// Returns error if database exists with data (user must manually delete it)
func (i *Initializer) createDatabase(ctx context.Context, dbPath string) (bool, error) {
	// Check if database already exists
	if _, err := os.Stat(dbPath); err == nil {
		// Database file exists, check if it contains any data
		if hasData, err := i.databaseHasData(dbPath); err != nil {
			return false, fmt.Errorf("failed to check database contents: %w", err)
		} else if hasData {
			// Database has data, refuse to initialize
			return false, fmt.Errorf("database already contains data. To reset, manually delete: %s", dbPath)
		}
		// Database file exists but is empty, skip creation
		return false, nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, err
	}

	// Create database with schema
	database, err := db.InitDB(dbPath)
	if err != nil {
		return false, err
	}
	defer database.Close()

	// Set file permissions (Unix only)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dbPath, 0600); err != nil {
			return false, err
		}
	}

	return true, nil
}

// databaseHasData checks if database contains any data
func (i *Initializer) databaseHasData(dbPath string) (bool, error) {
	// Open database without initializing schema
	database, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return false, fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Test the connection
	if err := database.Ping(); err != nil {
		return false, fmt.Errorf("failed to ping database: %w", err)
	}

	// Check if tables exist and have data
	tables := []string{"epics", "features", "tasks", "task_history"}
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		// Use sqlite3 error handling - table might not exist
		if err := database.QueryRow(query).Scan(&count); err != nil {
			// Table doesn't exist, continue checking
			continue
		}
		if count > 0 {
			// Found data in this table
			return true, nil
		}
	}

	// No data found in any tables
	return false, nil
}
