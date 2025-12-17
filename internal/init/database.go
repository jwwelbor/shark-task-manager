package init

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

// createDatabase creates database schema if it doesn't exist
// Returns true if database was created, false if already existed
func (i *Initializer) createDatabase(ctx context.Context, dbPath string) (bool, error) {
	// Check if database already exists
	if _, err := os.Stat(dbPath); err == nil {
		// Database exists, skip creation
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
