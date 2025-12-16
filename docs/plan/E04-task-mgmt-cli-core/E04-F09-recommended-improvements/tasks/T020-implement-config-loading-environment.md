---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T019-create-config-package.md]
estimated_time: 3 hours
---

# Task: Implement Config Loading from Environment

## Goal

Implement configuration loading from environment variables using the Viper library, enabling environment-specific configuration and 12-factor app compliance.

## Success Criteria

- [ ] Config loads from environment variables with `SHARK_` prefix
- [ ] Default values are sensible and well-documented
- [ ] Config struct is fully populated from environment
- [ ] Config validation catches invalid values
- [ ] Config loading handles missing values gracefully
- [ ] Unit tests for config loading pass
- [ ] Documentation for all config options created

## Implementation Guidance

### Overview

Implement the configuration loading logic that reads from environment variables and populates the config struct defined in T019. Use Viper for environment variable handling and provide sensible defaults for all values.

### Key Requirements

- Use Viper to load environment variables with `SHARK_` prefix
- Provide default values for all configuration options
- Validate configuration after loading (e.g., port number range, file paths)
- Support duration parsing for timeout values (e.g., `30s`, `5m`)
- Return config struct and error from Load() function

Reference: [PRD - Configuration Management](../01-feature-prd.md#fr-4-configuration-management)

### Files to Create/Modify

**Config Package**:
- `internal/config/config.go` - Implement `Load()` function
- `internal/config/config_test.go` - Unit tests for config loading
- `internal/config/validation.go` (optional) - Config validation logic
- `internal/config/defaults.go` (optional) - Default values

### Implementation Pattern

**Config Loading**:
```go
func Load() (*Config, error) {
    v := viper.New()

    // Set default values
    v.SetDefault("database.path", "shark-tasks.db")
    v.SetDefault("database.max_open_conns", 25)
    v.SetDefault("database.max_idle_conns", 5)
    v.SetDefault("database.timeout", "30s")
    v.SetDefault("server.port", "8080")
    v.SetDefault("server.read_timeout", "5s")
    v.SetDefault("server.write_timeout", "10s")
    v.SetDefault("cli.default_format", "table")
    v.SetDefault("cli.color_output", true)

    // Environment variables
    v.SetEnvPrefix("SHARK")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()

    // Unmarshal into config struct
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return &cfg, nil
}
```

**Validation**:
```go
func (c *Config) Validate() error {
    // Database validation
    if c.Database.Path == "" {
        return fmt.Errorf("database path cannot be empty")
    }
    if c.Database.MaxOpenConns < 1 {
        return fmt.Errorf("max_open_conns must be at least 1")
    }
    if c.Database.Timeout < time.Second {
        return fmt.Errorf("database timeout must be at least 1 second")
    }

    // Server validation
    if c.Server.Port == "" {
        return fmt.Errorf("server port cannot be empty")
    }
    port, err := strconv.Atoi(c.Server.Port)
    if err != nil || port < 1 || port > 65535 {
        return fmt.Errorf("invalid server port: %s (must be 1-65535)", c.Server.Port)
    }

    // CLI validation
    validFormats := map[string]bool{"json": true, "table": true, "text": true}
    if !validFormats[c.CLI.DefaultFormat] {
        return fmt.Errorf("invalid cli format: %s (must be json, table, or text)", c.CLI.DefaultFormat)
    }

    return nil
}
```

Reference: [PRD - Config Structure](../01-feature-prd.md#fr-4-configuration-management)

### Environment Variable Mapping

- `SHARK_DATABASE_PATH` → `config.Database.Path`
- `SHARK_DATABASE_MAX_OPEN_CONNS` → `config.Database.MaxOpenConns`
- `SHARK_DATABASE_MAX_IDLE_CONNS` → `config.Database.MaxIdleConns`
- `SHARK_DATABASE_TIMEOUT` → `config.Database.Timeout`
- `SHARK_SERVER_PORT` → `config.Server.Port`
- `SHARK_SERVER_READ_TIMEOUT` → `config.Server.ReadTimeout`
- `SHARK_SERVER_WRITE_TIMEOUT` → `config.Server.WriteTimeout`
- `SHARK_CLI_DEFAULT_FORMAT` → `config.CLI.DefaultFormat`
- `SHARK_CLI_COLOR_OUTPUT` → `config.CLI.ColorOutput`

### Integration Points

- **Config Struct**: Defined in T019
- **Main Functions**: Will use `config.Load()` in T022/T023
- **Viper Library**: Already in go.mod
- **Environment**: Standard Go environment variable handling

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Unit Tests**:
- Test config loading with environment variables set
- Test config loading with defaults (no env vars)
- Test config validation catches invalid values
- Test duration parsing works correctly
- Run: `go test ./internal/config/...`

**Manual Testing**:
- Load config with no env vars (uses defaults)
- Set env vars and verify they override defaults
- Test invalid values trigger validation errors
- Verify all fields populated correctly

**Documentation**:
- All config options documented
- Default values documented
- Environment variable names documented

## Context & Resources

- **PRD**: [Configuration Management](../01-feature-prd.md#fr-4-configuration-management)
- **PRD**: [Environment Variables List](../01-feature-prd.md#fr-4-configuration-management)
- **Task Dependency**: [T019 - Config Package](./T019-create-config-package.md)
- **Viper**: [Viper Documentation](https://github.com/spf13/viper)
- **12-Factor**: [Config Section](https://12factor.net/config)

## Notes for Agent

- Viper is already in go.mod (used by Cobra)
- Pattern: Set defaults, then load from environment, then validate
- Use `SetEnvPrefix("SHARK")` to add SHARK_ prefix automatically
- Use `SetEnvKeyReplacer` to convert dots to underscores (e.g., `database.path` → `DATABASE_PATH`)
- `AutomaticEnv()` enables automatic environment variable reading
- Duration parsing: Viper handles `30s`, `5m`, etc. automatically
- Validation should be strict but helpful (clear error messages)
- Default values should be production-ready
- This enables T022/T023 (using config in main functions)
