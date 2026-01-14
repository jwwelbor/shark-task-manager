package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// initDatabase initializes a cloud-aware database connection
// This replaces the old pattern of calling db.InitDB() directly
// It reads .sharkconfig.json to determine whether to use local SQLite or Turso cloud
func initDatabase(ctx context.Context) (*repository.DB, error) {
	// Find project root
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	// Get config path
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")

	// Get database config
	dbConfig, err := GetDatabaseConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database config: %w", err)
	}

	// For local/sqlite backend, use the old path for now (backward compatibility)
	// This avoids updating the entire repository layer in one go
	if dbConfig.Backend == "sqlite" || dbConfig.Backend == "local" || dbConfig.Backend == "" {
		// Use old InitDB for local SQLite
		dbPath := dbConfig.URL
		if dbPath == "" {
			dbPath = filepath.Join(projectRoot, "shark-tasks.db")
		}

		database, err := db.InitDB(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize database: %w", err)
		}

		return repository.NewDB(database), nil
	}

	// For Turso cloud, use the new driver system
	database, err := InitializeDatabaseFromConfig(ctx, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Turso driver needs to be converted to *sql.DB for repository layer
	// This is a temporary solution until repositories are updated to use Database interface
	tursoDriver, ok := database.(*db.TursoDriver)
	if !ok {
		return nil, fmt.Errorf("expected TursoDriver for turso backend")
	}

	// Get the underlying *sql.DB from Turso driver
	sqlDB, err := tursoDriver.GetSQLDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from Turso driver: %w", err)
	}

	// Apply schema and migrations to the Turso database
	// This ensures tables like 'ideas' are created on cloud databases
	if err := db.ApplySchemaAndMigrations(sqlDB); err != nil {
		return nil, fmt.Errorf("failed to apply schema and migrations: %w", err)
	}

	return repository.NewDB(sqlDB), nil
}
