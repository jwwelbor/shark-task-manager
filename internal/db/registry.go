package db

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

var (
	// drivers stores registered database driver factories
	drivers = make(map[string]DriverFactory)
	// mu protects concurrent access to drivers map
	mu sync.RWMutex
)

// DriverFactory is a function that creates a new Database instance
type DriverFactory func() Database

// RegisterDriver registers a database driver with the given name
// This should be called during package initialization (init functions)
func RegisterDriver(name string, factory DriverFactory) {
	mu.Lock()
	defer mu.Unlock()
	drivers[name] = factory
}

// NewDatabase creates a new database instance based on the provided configuration
// It automatically detects the backend from the URL if backend is not specified
func NewDatabase(config config.DatabaseConfig) (Database, error) {
	backend := config.Backend
	if backend == "" {
		// Auto-detect from URL
		backend = DetectBackend(config.URL)
	}

	mu.RLock()
	factory, exists := drivers[backend]
	mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown database backend: %s (available: %v)", backend, GetRegisteredDrivers())
	}

	return factory(), nil
}

// DetectBackend automatically detects the database backend from a URL
// Returns "turso" for libsql:// or https:// URLs, "sqlite" for file paths
func DetectBackend(url string) string {
	if strings.HasPrefix(url, "libsql://") || strings.HasPrefix(url, "https://") {
		return "turso"
	}
	return "sqlite"
}

// GetRegisteredDrivers returns a list of registered driver names
func GetRegisteredDrivers() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}
	return names
}

// ResetRegistry clears all registered drivers (used for testing)
func ResetRegistry() {
	mu.Lock()
	defer mu.Unlock()
	drivers = make(map[string]DriverFactory)
}

// InitDatabase is a high-level function to initialize a database connection
// It creates a database instance, connects to it, and verifies the connection
func InitDatabase(ctx context.Context, config config.DatabaseConfig) (Database, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	// Create database instance
	db, err := NewDatabase(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Connect to database
	if err := db.Connect(ctx, config.URL); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Verify connection
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// init registers the SQLite and Turso drivers by default
func init() {
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})
	RegisterDriver("turso", func() Database {
		return NewTursoDriver()
	})
}
