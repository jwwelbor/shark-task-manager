---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T021-add-viper-support-config-files.md]
estimated_time: 1 hour
---

# Task: Update Server main.go to Use Config

## Goal

Update the HTTP server's main function to load and use configuration instead of hardcoded values, enabling environment-specific server configuration and proper production deployment.

## Success Criteria

- [ ] Server loads config on startup using `config.Load()`
- [ ] Database path comes from config, not hardcoded
- [ ] Server port comes from config, not hardcoded
- [ ] Timeouts come from config
- [ ] Connection pool settings come from config
- [ ] Config errors are handled gracefully
- [ ] Server starts successfully with defaults
- [ ] Server respects environment variable overrides

## Implementation Guidance

### Overview

Replace hardcoded values in `cmd/server/main.go` with configuration loaded from the config package. This enables flexible deployment with environment-specific settings.

### Key Requirements

- Load config at the start of `main()`
- Use config values for database initialization
- Use config values for HTTP server configuration
- Handle config loading errors gracefully
- Log configuration at startup (for debugging)

Reference: [PRD - Server Config Usage](../01-feature-prd.md#fr-4-configuration-management)

### Files to Create/Modify

**Server Main**:
- `cmd/server/main.go` - Replace hardcoded values with config

### Implementation Pattern

**Before (hardcoded values)**:
```go
func main() {
    // Hardcoded database path
    db, err := sql.Open("sqlite3", "shark-tasks.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Hardcoded server port
    log.Println("Starting server on :8080")
    http.ListenAndServe(":8080", router)
}
```

**After (using config)**:
```go
func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Log configuration (for debugging)
    log.Printf("Config: DB=%s, Port=%s", cfg.Database.Path, cfg.Server.Port)

    // Open database with config
    db, err := sql.Open("sqlite3", cfg.Database.Path)
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()

    // Configure connection pool
    db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
    db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.Database.Timeout)

    // Initialize repositories
    taskRepo := sqlite.NewTaskRepository(db)
    // ... other repos

    // Configure HTTP server with timeouts
    server := &http.Server{
        Addr:         ":" + cfg.Server.Port,
        Handler:      router,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
    }

    log.Printf("Starting server on :%s", cfg.Server.Port)
    if err := server.ListenAndServe(); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

### Configuration Usage

**Database Configuration**:
- `cfg.Database.Path` - Database file path
- `cfg.Database.MaxOpenConns` - Connection pool size
- `cfg.Database.MaxIdleConns` - Idle connections
- `cfg.Database.Timeout` - Query timeout

**Server Configuration**:
- `cfg.Server.Port` - HTTP port (without ":" prefix in config)
- `cfg.Server.ReadTimeout` - Read timeout for requests
- `cfg.Server.WriteTimeout` - Write timeout for responses

### Integration Points

- **Config Package**: Uses `internal/config.Load()`
- **Database**: Database path and connection pool from config
- **HTTP Server**: Port and timeouts from config
- **Repositories**: Initialized with configured database

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Build Verification**:
- Server builds successfully: `go build ./cmd/server`
- No hardcoded values remain (verify with grep)

**Manual Testing**:
- Start server with defaults: `go run cmd/server/main.go`
- Verify server starts on default port (8080)
- Test with environment variables:
  ```bash
  SHARK_SERVER_PORT=9090 go run cmd/server/main.go
  ```
- Verify server starts on configured port (9090)
- Test with config file: Create `.shark.yaml`, verify settings used

**Integration Testing**:
- Test API endpoints still work with configured server
- Verify database operations work with configured path
- Verify timeouts are respected

## Context & Resources

- **PRD**: [Configuration Management](../01-feature-prd.md#fr-4-configuration-management)
- **PRD**: [Server Config Example](../01-feature-prd.md#fr-4-configuration-management)
- **Task Dependencies**: T019, T020, T021 (config package complete)
- **Current Code**: `cmd/server/main.go`
- **Config Package**: `internal/config/`

## Notes for Agent

- Load config at the very start of `main()`
- Use `log.Fatalf()` for config loading errors (can't proceed without config)
- Log config values at startup for debugging (but not sensitive values)
- Add ":" prefix to port when creating server address
- Set all http.Server fields for proper timeout handling
- Connection pool settings are important for production
- Test both default config and environment variable overrides
- No hardcoded values should remain - grep for "8080", "shark-tasks.db", etc.
- This makes server production-ready and cloud-friendly
