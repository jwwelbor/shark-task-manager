package dbinit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// Options controls how Init resolves the project root and config file.
type Options struct {
	// ProjectRoot is the explicit project root directory.
	// If empty, Init walks up from the current working directory looking
	// for .sharkconfig.json, shark-tasks.db, or .git/ (in that order of
	// preference), mirroring cli.FindProjectRoot behaviour.
	ProjectRoot string

	// ConfigPath is an optional override for the configuration file path.
	// If empty, defaults to filepath.Join(resolvedProjectRoot, ".sharkconfig.json").
	ConfigPath string
}

// Init returns a cloud-aware *repository.DB by reading .sharkconfig.json to
// choose between a local SQLite backend and a Turso cloud backend.
//
// It honours the following settings in the database section of .sharkconfig.json:
//   - backend: "sqlite" | "local" | "" → local SQLite via db.InitDB
//   - backend: "turso"                 → Turso cloud via the driver registry
//   - skip_migrations: true            → uses db.ApplySchemaIfNeeded (fast path)
//   - auth_token_file                  → loaded via db.LoadAuthToken
//
// Returns an error for any unsupported backend value.
func Init(ctx context.Context, opts Options) (*repository.DB, error) {
	projectRoot, configPath, err := resolveRoots(opts)
	if err != nil {
		return nil, err
	}

	dbConfig, err := loadDatabaseConfig(configPath, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}

	switch dbConfig.Backend {
	case "sqlite", "local", "":
		return initLocal(dbConfig)
	case "turso":
		return initTurso(ctx, dbConfig)
	default:
		return nil, fmt.Errorf("unsupported database backend: %q", dbConfig.Backend)
	}
}

// MustInit is like Init but panics on error, suitable for CLI entry points that
// use fail-fast semantics.
func MustInit(ctx context.Context, opts Options) *repository.DB {
	repoDb, err := Init(ctx, opts)
	if err != nil {
		panic(fmt.Sprintf("dbinit: failed to initialize database: %v", err))
	}
	return repoDb
}

// resolveRoots resolves the project root and config file path from opts.
func resolveRoots(opts Options) (projectRoot, configPath string, err error) {
	if opts.ProjectRoot != "" {
		projectRoot = opts.ProjectRoot
	} else {
		projectRoot, err = findProjectRoot("")
		if err != nil {
			return "", "", fmt.Errorf("failed to find project root: %w", err)
		}
	}

	if opts.ConfigPath != "" {
		configPath = opts.ConfigPath
	} else {
		configPath = filepath.Join(projectRoot, ".sharkconfig.json")
	}

	return projectRoot, configPath, nil
}

// loadDatabaseConfig reads db.DatabaseConfig from configPath.
// Falls back to a local SQLite default when the config file is absent or has
// no database section.
func loadDatabaseConfig(configPath, projectRoot string) (db.DatabaseConfig, error) {
	configDir := filepath.Dir(configPath)

	mgr := config.NewManager(configPath)
	cfg, err := mgr.Load()
	if err != nil {
		// Config file missing or unreadable — use local default.
		return db.DatabaseConfig{
			Backend: "sqlite",
			URL:     filepath.Join(projectRoot, "shark-tasks.db"),
		}, nil
	}

	if cfg.RawData == nil {
		return db.DatabaseConfig{
			Backend: "sqlite",
			URL:     filepath.Join(projectRoot, "shark-tasks.db"),
		}, nil
	}

	dbConfigRaw, ok := cfg.RawData["database"]
	if !ok {
		return db.DatabaseConfig{
			Backend: "sqlite",
			URL:     filepath.Join(projectRoot, "shark-tasks.db"),
		}, nil
	}

	dbConfigMap, ok := dbConfigRaw.(map[string]interface{})
	if !ok {
		return db.DatabaseConfig{}, fmt.Errorf("invalid database config format in %s", configPath)
	}

	dbConfig := db.DatabaseConfig{}

	if backend, ok := dbConfigMap["backend"].(string); ok {
		dbConfig.Backend = os.ExpandEnv(backend)
	}

	if url, ok := dbConfigMap["url"].(string); ok {
		dbConfig.URL = os.ExpandEnv(url)
	}

	if authTokenFile, ok := dbConfigMap["auth_token_file"].(string); ok {
		dbConfig.AuthTokenFile = os.ExpandEnv(authTokenFile)
	}

	if embeddedReplica, ok := dbConfigMap["embedded_replica"].(bool); ok {
		dbConfig.EmbeddedReplica = embeddedReplica
	}

	if skipMigrations, ok := dbConfigMap["skip_migrations"].(bool); ok {
		dbConfig.SkipMigrations = skipMigrations
	}

	// Apply defaults when fields are missing.
	if dbConfig.Backend == "" {
		dbConfig.Backend = "sqlite"
	}
	if dbConfig.URL == "" {
		dbConfig.URL = filepath.Join(configDir, "shark-tasks.db")
	}

	return dbConfig, nil
}

// initLocal initialises a local SQLite database and wraps it in *repository.DB.
func initLocal(dbConfig db.DatabaseConfig) (*repository.DB, error) {
	dbPath := dbConfig.URL
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local database at %q: %w", dbPath, err)
	}
	return repository.NewDB(sqlDB), nil
}

// initTurso initialises a Turso cloud database, applies schema/migrations
// (honouring skip_migrations), and wraps the result in *repository.DB.
func initTurso(ctx context.Context, dbConfig db.DatabaseConfig) (*repository.DB, error) {
	// Load auth token from file or env var before dialling.
	if dbConfig.AuthTokenFile != "" {
		authToken, err := db.LoadAuthToken(dbConfig.AuthTokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load Turso auth token: %w", err)
		}
		dbConfig.URL = db.BuildTursoConnectionString(dbConfig.URL, authToken)
	} else if token := os.Getenv("TURSO_AUTH_TOKEN"); token != "" {
		dbConfig.URL = db.BuildTursoConnectionString(dbConfig.URL, token)
	}

	if err := dbConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Turso database config: %w", err)
	}

	database, err := db.InitDatabase(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Turso database: %w", err)
	}

	tursoDriver, ok := database.(*db.TursoDriver)
	if !ok {
		return nil, fmt.Errorf("expected TursoDriver for turso backend, got %T", database)
	}

	sqlDB, err := tursoDriver.GetSQLDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from Turso driver: %w", err)
	}

	if dbConfig.SkipMigrations {
		if _, err := db.ApplySchemaIfNeeded(sqlDB); err != nil {
			return nil, fmt.Errorf("failed to apply schema (fast-path): %w", err)
		}
	} else {
		if err := db.ApplySchemaAndMigrations(sqlDB); err != nil {
			return nil, fmt.Errorf("failed to apply schema and migrations: %w", err)
		}
	}

	return repository.NewDB(sqlDB), nil
}
