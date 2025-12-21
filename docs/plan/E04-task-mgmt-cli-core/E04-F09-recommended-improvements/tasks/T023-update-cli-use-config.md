---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T021-add-viper-support-config-files.md]
estimated_time: 1 hour
---

# Task: Update CLI to Use Config

## Goal

Update the CLI's main function and commands to load and use configuration, enabling user customization of database path, output format, and color preferences.

## Success Criteria

- [ ] CLI loads config on startup using `config.Load()`
- [ ] Database path comes from config, not hardcoded
- [ ] Default output format comes from config
- [ ] Color output setting comes from config
- [ ] Config errors are handled gracefully
- [ ] CLI works with defaults if no config provided
- [ ] CLI respects environment variable overrides
- [ ] CLI flags override config values (highest precedence)

## Implementation Guidance

### Overview

Replace hardcoded values in `cmd/pm/main.go` with configuration loaded from the config package. Enable users to customize CLI behavior through config file or environment variables.

### Key Requirements

- Load config at the start of CLI initialization
- Use config values for database initialization
- Use config values for default output formatting
- Use config values for color output
- CLI flags should override config (maintain current behavior)

Reference: [PRD - CLI Config Usage](../01-feature-prd.md#fr-4-configuration-management)

### Files to Create/Modify

**CLI Main**:
- `cmd/pm/main.go` - Replace hardcoded values with config
- May need to update command files if they reference global config

### Implementation Pattern

**Before (hardcoded values)**:
```go
func main() {
    // Hardcoded database path
    db, err := sql.Open("sqlite3", "shark-tasks.db")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    defer db.Close()

    // Execute commands
    rootCmd.Execute()
}
```

**After (using config)**:
```go
var cfg *config.Config

func init() {
    // Load configuration
    var err error
    cfg, err = config.Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
        fmt.Fprintf(os.Stderr, "Using default configuration.\n")
        cfg = config.DefaultConfig()
    }
}

func main() {
    // Open database with config
    db, err := sql.Open("sqlite3", cfg.Database.Path)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
        os.Exit(1)
    }
    defer db.Close()

    // Configure connection pool
    db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
    db.SetMaxIdleConns(cfg.Database.MaxIdleConns)

    // Initialize repositories
    taskRepo := sqlite.NewTaskRepository(db)
    // ... other repos

    // Execute commands
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

**Using config in commands**:
```go
// In command implementation
func runTaskList(cmd *cobra.Command, args []string) error {
    // Get output format from flag or config
    format, _ := cmd.Flags().GetString("format")
    if format == "" {
        format = cfg.CLI.DefaultFormat  // Use config default
    }

    // Use color output from config (if not overridden by flag)
    useColor := cfg.CLI.ColorOutput

    // ... rest of command logic
}
```

### Configuration Usage

**Database Configuration**:
- `cfg.Database.Path` - Database file path
- `cfg.Database.MaxOpenConns` - Connection pool size
- `cfg.Database.MaxIdleConns` - Idle connections

**CLI Configuration**:
- `cfg.CLI.DefaultFormat` - Default output format (json, table, text)
- `cfg.CLI.ColorOutput` - Enable colored output

**Precedence** (highest to lowest):
1. CLI flags (`--format=json`)
2. Environment variables (`SHARK_CLI_DEFAULT_FORMAT=json`)
3. Config file (`.shark.yaml`)
4. Default values

### Integration Points

- **Config Package**: Uses `internal/config.Load()`
- **Database**: Database path from config
- **Command Flags**: Flags override config values
- **Output Formatting**: Config provides defaults

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Build Verification**:
- CLI builds successfully: `go build ./cmd/shark`
- No hardcoded values remain (verify with grep)

**Manual Testing**:
- Run CLI with defaults: `shark task list`
- Test with environment variables:
  ```bash
  SHARK_DATABASE_PATH=/tmp/test.db shark task list
  ```
- Test with config file: Create `.shark.yaml`, verify settings used
- Test that CLI flags override config:
  ```bash
  shark task list --format=json  # Should use json even if config says table
  ```

**Integration Testing**:
- All CLI commands work with configured database
- Output formatting respects config
- Color output respects config
- Flags override config properly

## Context & Resources

- **PRD**: [Configuration Management](../01-feature-prd.md#fr-4-configuration-management)
- **PRD**: [CLI Config Example](../01-feature-prd.md#fr-4-configuration-management)
- **Task Dependencies**: T019, T020, T021 (config package complete)
- **Current Code**: `cmd/pm/main.go`
- **Config Package**: `internal/config/`

## Notes for Agent

- Load config in `init()` function, before `main()` runs
- Handle config loading errors gracefully (CLI should work with defaults)
- Create `DefaultConfig()` function for fallback if config loading fails
- CLI flags should always override config (highest precedence)
- Pattern: Check flag value, if empty use config value, if that's empty use hardcoded default
- Database path is most important config value for CLI
- Output format and color are nice-to-have customizations
- Test precedence order: flag > env var > config file > default
- Connection pool settings apply to CLI too (though less critical than server)
- This completes Phase 4 (Configuration Management)
