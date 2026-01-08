package cli

import (
	"context"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

var (
	// globalDB holds the shared database connection for all commands
	globalDB *repository.DB

	// dbInitOnce ensures database is initialized exactly once
	dbInitOnce sync.Once

	// dbInitErr stores initialization error for propagation
	dbInitErr error
)

// GetDB returns the global database connection, initializing it if needed.
// This is the ONLY function commands should call to get database access.
//
// The database is initialized lazily on first call using the existing
// initDatabase() function which is cloud-aware and reads .sharkconfig.json.
//
// Usage:
//
//	repoDb, err := GetDB(cmd.Context())
//	if err != nil {
//	    return fmt.Errorf("failed to get database: %w", err)
//	}
func GetDB(ctx context.Context) (*repository.DB, error) {
	dbInitOnce.Do(func() {
		globalDB, dbInitErr = initDatabase(ctx)
	})

	if dbInitErr != nil {
		return nil, dbInitErr
	}

	return globalDB, nil
}

// CloseDB closes the global database connection.
// Called automatically by root command's PersistentPostRunE hook.
// It's safe to call multiple times (subsequent calls are no-ops).
func CloseDB() error {
	if globalDB != nil {
		err := globalDB.Close()
		// Reset state after close (allows reinitialization if needed)
		globalDB = nil
		dbInitErr = nil
		dbInitOnce = sync.Once{}
		return err
	}
	return nil
}

// ResetDB clears the global database state.
// This is intended for testing only - DO NOT use in production code.
// It allows tests to reset state between test cases.
func ResetDB() {
	if globalDB != nil {
		globalDB.Close()
	}
	globalDB = nil
	dbInitErr = nil
	dbInitOnce = sync.Once{}
}
