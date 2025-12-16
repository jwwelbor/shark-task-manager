# Task: Create internal/config Package

**Task ID**: E04-F09-T019
**Feature**: E04-F09 - Recommended Architecture Improvements
**Priority**: P3 (Medium)
**Estimated Effort**: 2 hours
**Status**: Todo

---

## Objective

Create the `internal/config` package to manage application configuration from environment variables and config files, eliminating hardcoded values.

## Background

Currently, configuration values are hardcoded:
- Database path: `"shark-tasks.db"` in main.go
- Server port: `"8080"` in main.go
- No way to change without recompiling

This prevents:
- Environment-specific configuration (dev, staging, prod)
- User customization
- Container deployment (12-factor app compliance)
- Testing with different configurations

## Acceptance Criteria

- [ ] `internal/config/` package created
- [ ] Configuration struct defined with all settings
- [ ] Load from environment variables (with `SHARK_` prefix)
- [ ] Load from config file (`.shark.yaml`) using Viper
- [ ] Sensible defaults for all settings
- [ ] Validation on configuration load
- [ ] Configuration is immutable after load
- [ ] Compiles successfully
- [ ] Unit tests for config loading

## Implementation Details

### File: internal/config/config.go

```go
package config

import (
    "fmt"
    "os"
    "time"

    "github.com/spf13/viper"
)

// Config holds all application configuration.
//
// Configuration is loaded from three sources in order of precedence:
//   1. Environment variables (highest priority)
//   2. Config file (.shark.yaml)
//   3. Default values (lowest priority)
//
// Environment variables use the SHARK_ prefix, e.g.:
//   SHARK_DB_PATH=/path/to/db
//   SHARK_SERVER_PORT=8080
//
// Example config file (.shark.yaml):
//   database:
//     path: ./shark-tasks.db
//     max_open_conns: 25
//   server:
//     port: 8080
//
// Usage:
//   cfg := config.Load()
//   db, err := db.InitDB(cfg.Database.Path)
type Config struct {
    Database DatabaseConfig `mapstructure:"database"`
    Server   ServerConfig   `mapstructure:"server"`
    CLI      CLIConfig      `mapstructure:"cli"`
}

// DatabaseConfig contains database-specific settings.
type DatabaseConfig struct {
    // Path to the SQLite database file.
    // Default: "shark-tasks.db"
    // Env: SHARK_DB_PATH
    Path string `mapstructure:"path"`

    // Maximum number of open database connections.
    // Default: 25
    // Env: SHARK_DB_MAX_OPEN_CONNS
    MaxOpenConns int `mapstructure:"max_open_conns"`

    // Maximum number of idle database connections.
    // Default: 5
    // Env: SHARK_DB_MAX_IDLE_CONNS
    MaxIdleConns int `mapstructure:"max_idle_conns"`

    // Maximum lifetime of a database connection.
    // Default: 1 hour
    // Env: SHARK_DB_CONN_MAX_LIFETIME
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`

    // Query timeout for database operations.
    // Default: 30 seconds
    // Env: SHARK_DB_QUERY_TIMEOUT
    QueryTimeout time.Duration `mapstructure:"query_timeout"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
    // Port for the HTTP server.
    // Default: "8080"
    // Env: SHARK_SERVER_PORT
    Port string `mapstructure:"port"`

    // Read timeout for HTTP requests.
    // Default: 5 seconds
    // Env: SHARK_SERVER_READ_TIMEOUT
    ReadTimeout time.Duration `mapstructure:"read_timeout"`

    // Write timeout for HTTP responses.
    // Default: 10 seconds
    // Env: SHARK_SERVER_WRITE_TIMEOUT
    WriteTimeout time.Duration `mapstructure:"write_timeout"`

    // Idle timeout for keep-alive connections.
    // Default: 120 seconds
    // Env: SHARK_SERVER_IDLE_TIMEOUT
    IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

// CLIConfig contains CLI-specific settings.
type CLIConfig struct {
    // Default output format for CLI commands.
    // Valid values: "table", "json", "yaml", "text"
    // Default: "table"
    // Env: SHARK_CLI_FORMAT
    DefaultFormat string `mapstructure:"default_format"`

    // Enable colored output in terminal.
    // Default: true
    // Env: SHARK_CLI_COLOR
    ColorOutput bool `mapstructure:"color_output"`

    // Maximum number of items to display without pagination.
    // Default: 50
    // Env: SHARK_CLI_MAX_DISPLAY
    MaxDisplay int `mapstructure:"max_display"`

    // Default timeout for CLI operations.
    // Default: 30 seconds
    // Env: SHARK_CLI_TIMEOUT
    Timeout time.Duration `mapstructure:"timeout"`
}

// Load loads configuration from environment variables and config file.
//
// Configuration precedence (highest to lowest):
//   1. Environment variables (SHARK_DB_PATH, etc.)
//   2. Config file (.shark.yaml in current directory)
//   3. Default values
//
// Returns a Config struct with all values populated.
// Panics if configuration is invalid (use MustLoad in main.go).
func Load() *Config {
    v := viper.New()

    // Set defaults
    setDefaults(v)

    // Read from config file (optional)
    v.SetConfigName(".shark")
    v.SetConfigType("yaml")
    v.AddConfigPath(".")           // Current directory
    v.AddConfigPath("$HOME")       // Home directory
    v.AddConfigPath("/etc/shark/") // System config

    // Ignore error if config file doesn't exist (defaults will be used)
    _ = v.ReadInConfig()

    // Read from environment variables (highest priority)
    v.SetEnvPrefix("SHARK")
    v.AutomaticEnv()

    // Unmarshal into config struct
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        panic(fmt.Sprintf("failed to unmarshal config: %v", err))
    }

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        panic(fmt.Sprintf("invalid configuration: %v", err))
    }

    return &cfg
}

// MustLoad is an alias for Load() that panics on error.
// Use this in main.go for fail-fast behavior.
func MustLoad() *Config {
    return Load()
}

// LoadOrDefault loads configuration or returns default config on error.
// Use this in tests or when graceful degradation is acceptable.
func LoadOrDefault() *Config {
    defer func() {
        if r := recover(); r != nil {
            // Return default config if load fails
            return
        }
    }()

    return Load()
}

// setDefaults sets default values for all configuration options.
func setDefaults(v *viper.Viper) {
    // Database defaults
    v.SetDefault("database.path", "shark-tasks.db")
    v.SetDefault("database.max_open_conns", 25)
    v.SetDefault("database.max_idle_conns", 5)
    v.SetDefault("database.conn_max_lifetime", time.Hour)
    v.SetDefault("database.query_timeout", 30*time.Second)

    // Server defaults
    v.SetDefault("server.port", "8080")
    v.SetDefault("server.read_timeout", 5*time.Second)
    v.SetDefault("server.write_timeout", 10*time.Second)
    v.SetDefault("server.idle_timeout", 120*time.Second)

    // CLI defaults
    v.SetDefault("cli.default_format", "table")
    v.SetDefault("cli.color_output", true)
    v.SetDefault("cli.max_display", 50)
    v.SetDefault("cli.timeout", 30*time.Second)
}

// Validate validates the configuration and returns error if invalid.
func (c *Config) Validate() error {
    // Validate database config
    if c.Database.Path == "" {
        return fmt.Errorf("database.path cannot be empty")
    }
    if c.Database.MaxOpenConns < 1 {
        return fmt.Errorf("database.max_open_conns must be at least 1")
    }
    if c.Database.MaxIdleConns < 1 {
        return fmt.Errorf("database.max_idle_conns must be at least 1")
    }
    if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
        return fmt.Errorf("database.max_idle_conns cannot exceed max_open_conns")
    }
    if c.Database.QueryTimeout < time.Second {
        return fmt.Errorf("database.query_timeout must be at least 1 second")
    }

    // Validate server config
    if c.Server.Port == "" {
        return fmt.Errorf("server.port cannot be empty")
    }
    if c.Server.ReadTimeout < time.Second {
        return fmt.Errorf("server.read_timeout must be at least 1 second")
    }
    if c.Server.WriteTimeout < time.Second {
        return fmt.Errorf("server.write_timeout must be at least 1 second")
    }

    // Validate CLI config
    validFormats := map[string]bool{
        "table": true,
        "json":  true,
        "yaml":  true,
        "text":  true,
    }
    if !validFormats[c.CLI.DefaultFormat] {
        return fmt.Errorf("cli.default_format must be one of: table, json, yaml, text")
    }
    if c.CLI.MaxDisplay < 1 {
        return fmt.Errorf("cli.max_display must be at least 1")
    }
    if c.CLI.Timeout < time.Second {
        return fmt.Errorf("cli.timeout must be at least 1 second")
    }

    return nil
}

// String returns a string representation of the configuration (for debugging).
// Sensitive values are redacted.
func (c *Config) String() string {
    return fmt.Sprintf(
        "Config{Database: {Path: %s, MaxOpenConns: %d, MaxIdleConns: %d}, "+
            "Server: {Port: %s, ReadTimeout: %s, WriteTimeout: %s}, "+
            "CLI: {Format: %s, Color: %v}}",
        c.Database.Path,
        c.Database.MaxOpenConns,
        c.Database.MaxIdleConns,
        c.Server.Port,
        c.Server.ReadTimeout,
        c.Server.WriteTimeout,
        c.CLI.DefaultFormat,
        c.CLI.ColorOutput,
    )
}
```

### Example Config File

Create `.shark.yaml.example`:

```yaml
# Shark Task Manager Configuration
# Copy this file to .shark.yaml and customize

database:
  # Path to SQLite database file
  path: ./shark-tasks.db

  # Connection pool settings
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 1h

  # Query timeout
  query_timeout: 30s

server:
  # HTTP server port
  port: 8080

  # Timeouts
  read_timeout: 5s
  write_timeout: 10s
  idle_timeout: 120s

cli:
  # Default output format: table, json, yaml, text
  default_format: table

  # Enable colored terminal output
  color_output: true

  # Maximum items to display without pagination
  max_display: 50

  # Default timeout for CLI operations
  timeout: 30s
```

### Environment Variables Reference

Create `docs/CONFIGURATION.md`:

```markdown
# Configuration Reference

## Environment Variables

All environment variables use the `SHARK_` prefix.

### Database Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SHARK_DB_PATH` | string | `shark-tasks.db` | Path to SQLite database file |
| `SHARK_DB_MAX_OPEN_CONNS` | int | `25` | Maximum open connections |
| `SHARK_DB_MAX_IDLE_CONNS` | int | `5` | Maximum idle connections |
| `SHARK_DB_CONN_MAX_LIFETIME` | duration | `1h` | Maximum connection lifetime |
| `SHARK_DB_QUERY_TIMEOUT` | duration | `30s` | Query timeout |

### Server Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SHARK_SERVER_PORT` | string | `8080` | HTTP server port |
| `SHARK_SERVER_READ_TIMEOUT` | duration | `5s` | Read timeout |
| `SHARK_SERVER_WRITE_TIMEOUT` | duration | `10s` | Write timeout |
| `SHARK_SERVER_IDLE_TIMEOUT` | duration | `120s` | Idle timeout |

### CLI Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SHARK_CLI_FORMAT` | string | `table` | Output format (table, json, yaml, text) |
| `SHARK_CLI_COLOR` | bool | `true` | Enable colored output |
| `SHARK_CLI_MAX_DISPLAY` | int | `50` | Max items before pagination |
| `SHARK_CLI_TIMEOUT` | duration | `30s` | Operation timeout |

## Config File

Create `.shark.yaml` in your project directory or home directory.

See `.shark.yaml.example` for a complete example.

## Configuration Precedence

1. **Environment variables** (highest priority)
2. **Config file** (`.shark.yaml`)
3. **Default values** (lowest priority)

## Examples

### Using Environment Variables

```bash
# Use custom database path
export SHARK_DB_PATH=/var/lib/shark/tasks.db

# Change server port
export SHARK_SERVER_PORT=3000

# Run server
./shark-server
```

### Using Config File

```bash
# Create config file
cp .shark.yaml.example .shark.yaml

# Edit values
vim .shark.yaml

# Run server (reads .shark.yaml)
./shark-server
```

### Docker/Container

```dockerfile
ENV SHARK_DB_PATH=/data/tasks.db
ENV SHARK_SERVER_PORT=8080
```
```

## Testing

### Unit Tests

```go
// internal/config/config_test.go
package config

import (
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestConfig_Defaults(t *testing.T) {
    cfg := Load()

    assert.Equal(t, "shark-tasks.db", cfg.Database.Path)
    assert.Equal(t, 25, cfg.Database.MaxOpenConns)
    assert.Equal(t, "8080", cfg.Server.Port)
    assert.Equal(t, "table", cfg.CLI.DefaultFormat)
}

func TestConfig_FromEnvironment(t *testing.T) {
    // Set environment variables
    os.Setenv("SHARK_DB_PATH", "/custom/path.db")
    os.Setenv("SHARK_SERVER_PORT", "3000")
    defer os.Unsetenv("SHARK_DB_PATH")
    defer os.Unsetenv("SHARK_SERVER_PORT")

    cfg := Load()

    assert.Equal(t, "/custom/path.db", cfg.Database.Path)
    assert.Equal(t, "3000", cfg.Server.Port)
}

func TestConfig_Validation(t *testing.T) {
    tests := []struct {
        name    string
        modify  func(*Config)
        wantErr string
    }{
        {
            name: "empty database path",
            modify: func(c *Config) {
                c.Database.Path = ""
            },
            wantErr: "database.path cannot be empty",
        },
        {
            name: "invalid max open conns",
            modify: func(c *Config) {
                c.Database.MaxOpenConns = 0
            },
            wantErr: "database.max_open_conns must be at least 1",
        },
        {
            name: "idle > open conns",
            modify: func(c *Config) {
                c.Database.MaxIdleConns = 100
                c.Database.MaxOpenConns = 10
            },
            wantErr: "database.max_idle_conns cannot exceed max_open_conns",
        },
        {
            name: "invalid CLI format",
            modify: func(c *Config) {
                c.CLI.DefaultFormat = "invalid"
            },
            wantErr: "cli.default_format must be one of",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := &Config{
                Database: DatabaseConfig{
                    Path:            "test.db",
                    MaxOpenConns:    25,
                    MaxIdleConns:    5,
                    ConnMaxLifetime: time.Hour,
                    QueryTimeout:    30 * time.Second,
                },
                Server: ServerConfig{
                    Port:         "8080",
                    ReadTimeout:  5 * time.Second,
                    WriteTimeout: 10 * time.Second,
                    IdleTimeout:  120 * time.Second,
                },
                CLI: CLIConfig{
                    DefaultFormat: "table",
                    ColorOutput:   true,
                    MaxDisplay:    50,
                    Timeout:       30 * time.Second,
                },
            }

            tt.modify(cfg)
            err := cfg.Validate()

            require.Error(t, err)
            assert.Contains(t, err.Error(), tt.wantErr)
        })
    }
}
```

## Dependencies

### Depends On
- None (independent task)

### Blocks
- E04-F09-T020: Implement config loading from environment
- E04-F09-T021: Add Viper support for config files
- E04-F09-T022: Update server main.go to use config
- E04-F09-T023: Update CLI to use config

## Success Criteria

- ✅ Config package created with all structs
- ✅ Load() function with environment variable support
- ✅ Config file support with Viper
- ✅ Sensible defaults for all values
- ✅ Validation logic implemented
- ✅ Unit tests pass
- ✅ Example config file created
- ✅ Configuration documentation written

## Completion Checklist

- [ ] Create `internal/config/config.go`
- [ ] Define Config, DatabaseConfig, ServerConfig, CLIConfig structs
- [ ] Implement Load() function with Viper
- [ ] Implement setDefaults() function
- [ ] Implement Validate() method
- [ ] Create `.shark.yaml.example`
- [ ] Write `docs/CONFIGURATION.md`
- [ ] Write unit tests in `config_test.go`
- [ ] Verify compilation: `go build ./internal/config/`
- [ ] Run tests: `go test ./internal/config/`
- [ ] Git commit: "Add configuration package for environment-based config"
