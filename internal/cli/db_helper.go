package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
)

// GetDatabaseConfig reads database configuration from .sharkconfig.json
// Returns config with fallback to local SQLite if no database section exists
func GetDatabaseConfig(configPath string) (config.DatabaseConfig, error) {
	// Get directory containing config
	configDir := filepath.Dir(configPath)

	// Load config using manager
	mgr := config.NewManager(configPath)
	cfg, err := mgr.Load()
	if err != nil {
		return config.DatabaseConfig{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Check if database config exists in raw data
	if cfg.RawData == nil {
		// No config data - fall back to local database
		return config.DatabaseConfig{
			Backend: "sqlite",
			URL:     filepath.Join(configDir, "shark-tasks.db"),
		}, nil
	}

	dbConfigRaw, ok := cfg.RawData["database"]
	if !ok {
		// No database section - fall back to local database
		return config.DatabaseConfig{
			Backend: "sqlite",
			URL:     filepath.Join(configDir, "shark-tasks.db"),
		}, nil
	}

	// Parse database config
	dbConfigMap, ok := dbConfigRaw.(map[string]interface{})
	if !ok {
		return config.DatabaseConfig{}, fmt.Errorf("invalid database config format")
	}

	// Extract fields
	dbConfig := config.DatabaseConfig{}

	if backend, ok := dbConfigMap["backend"].(string); ok {
		dbConfig.Backend = backend
	}

	if url, ok := dbConfigMap["url"].(string); ok {
		dbConfig.URL = url
	}

	if authTokenFile, ok := dbConfigMap["auth_token_file"].(string); ok {
		dbConfig.AuthTokenFile = authTokenFile
	}

	if embeddedReplica, ok := dbConfigMap["embedded_replica"].(bool); ok {
		dbConfig.EmbeddedReplica = embeddedReplica
	}

	// Fall back to local if backend/URL not specified
	if dbConfig.Backend == "" {
		dbConfig.Backend = "sqlite"
	}
	if dbConfig.URL == "" {
		dbConfig.URL = filepath.Join(configDir, "shark-tasks.db")
	}

	return dbConfig, nil
}

// InitializeDatabaseFromConfig initializes a database connection using config from .sharkconfig.json
// This is the cloud-aware replacement for db.InitDB()
func InitializeDatabaseFromConfig(ctx context.Context, configPath string) (db.Database, error) {
	// Get database config
	dbConfig, err := GetDatabaseConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database config: %w", err)
	}

	// Validate config
	if err := dbConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	// Load auth token if Turso and auth_token_file is specified
	if dbConfig.Backend == "turso" && dbConfig.AuthTokenFile != "" {
		authToken, err := db.LoadAuthToken(dbConfig.AuthTokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load auth token: %w", err)
		}

		// Build connection string with auth token
		dbConfig.URL = db.BuildTursoConnectionString(dbConfig.URL, authToken)
	} else if dbConfig.Backend == "turso" {
		// Try environment variable
		authToken := os.Getenv("TURSO_AUTH_TOKEN")
		if authToken != "" {
			dbConfig.URL = db.BuildTursoConnectionString(dbConfig.URL, authToken)
		}
	}

	// Initialize database using driver registry
	database, err := db.InitDatabase(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return database, nil
}
