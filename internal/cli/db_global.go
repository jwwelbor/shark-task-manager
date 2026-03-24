package cli

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// dbContainer holds the database connection and its initialization state.
// Using a container struct makes ResetDB() safe: we swap the entire
// container atomically instead of reassigning individual sync.Once values
// (which would be a data race if any goroutine is mid-initialization).
type dbContainer struct {
	db       *repository.DB
	initOnce sync.Once
	initErr  error
}

// globalDBContainer is accessed only through loadDBContainer / storeDBContainer.
// Using atomic pointer operations ensures that a call to ResetDB()
// is immediately visible to any goroutine that subsequently calls
// GetDB(), without requiring a separate mutex.
//
//nolint:gochecknoglobals // Intentional package-level singleton for CLI entry points.
var globalDBContainer unsafe.Pointer // *dbContainer

func init() {
	storeDBContainer(new(dbContainer))
}

func loadDBContainer() *dbContainer {
	return (*dbContainer)(atomic.LoadPointer(&globalDBContainer))
}

func storeDBContainer(c *dbContainer) {
	atomic.StorePointer(&globalDBContainer, unsafe.Pointer(c))
}

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
	// Handle nil context (Cobra commands don't set context by default)
	if ctx == nil {
		ctx = context.Background()
	}

	c := loadDBContainer()
	c.initOnce.Do(func() {
		c.db, c.initErr = initDatabase(ctx)
	})

	if c.initErr != nil {
		return nil, c.initErr
	}

	return c.db, nil
}

// CloseDB closes the global database connection.
// Called automatically by root command's PersistentPostRunE hook.
// It's safe to call multiple times (subsequent calls are no-ops).
func CloseDB() error {
	c := loadDBContainer()
	if c.db != nil {
		err := c.db.Close()
		// Swap to a fresh container so subsequent calls see clean state
		storeDBContainer(new(dbContainer))
		return err
	}
	return nil
}

// ResetDB clears the global database state.
// This is intended for testing only - DO NOT use in production code.
// It allows tests to reset state between test cases.
func ResetDB() {
	c := loadDBContainer()
	if c.db != nil {
		c.db.Close()
	}
	storeDBContainer(new(dbContainer))
}
