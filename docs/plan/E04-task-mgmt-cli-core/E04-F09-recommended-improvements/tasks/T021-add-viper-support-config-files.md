---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T020-implement-config-loading-environment.md]
estimated_time: 2 hours
---

# Task: Add Viper Support for Config Files

## Goal

Extend the configuration loading to support optional config files (`.shark.yaml`, `.shark.json`, etc.) in addition to environment variables, providing flexibility for different deployment scenarios.

## Success Criteria

- [ ] Config loads from `.shark.yaml` file if present
- [ ] Config loads from `.shark.json` file if present (alternative format)
- [ ] Config searches for file in current directory and home directory
- [ ] Environment variables override config file values
- [ ] Config file is optional (defaults work without file)
- [ ] Example config file created in repository
- [ ] Documentation updated with config file examples

## Implementation Guidance

### Overview

Extend the `Load()` function from T020 to support optional configuration files. Users can provide config via file or environment variables, with environment variables taking precedence (12-factor app pattern).

### Key Requirements

- Support YAML and JSON config file formats
- Search for config file in: current directory, home directory, `/etc/shark/`
- Environment variables override config file values (highest precedence)
- Config file is completely optional
- Provide example config files in repository

Reference: [PRD - Config File Example](../01-feature-prd.md#fr-4-configuration-management)

### Files to Create/Modify

**Config Package**:
- `internal/config/config.go` - Update `Load()` to search for config files
- `internal/config/config_test.go` - Add tests for file loading

**Example Files** (in repository root):
- `.shark.yaml.example` - Example YAML config file
- `.shark.json.example` - Example JSON config file (alternative)

**Documentation**:
- Update README or docs with config file usage

### Implementation Pattern

**Extended Load() function**:
```go
func Load() (*Config, error) {
    v := viper.New()

    // Set config file search paths
    v.SetConfigName(".shark")
    v.SetConfigType("yaml")
    v.AddConfigPath(".")                      // Current directory
    v.AddConfigPath("$HOME")                  // Home directory
    v.AddConfigPath("/etc/shark/")            // System config

    // Set default values (same as T020)
    setDefaults(v)

    // Try to read config file (optional)
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            // Config file found but another error occurred
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
        // Config file not found is OK - will use defaults + env vars
    }

    // Environment variables (override config file)
    v.SetEnvPrefix("SHARK")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()

    // Unmarshal and validate (same as T020)
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return &cfg, nil
}
```

**Precedence order** (highest to lowest):
1. Environment variables (`SHARK_*`)
2. Config file (`.shark.yaml`)
3. Default values

### Example Config Files

**`.shark.yaml.example`**:
```yaml
# Shark Task Manager Configuration
# Copy this file to .shark.yaml and customize as needed

database:
  path: ./shark-tasks.db
  max_open_conns: 25
  max_idle_conns: 5
  timeout: 30s

server:
  port: "8080"
  read_timeout: 5s
  write_timeout: 10s

cli:
  default_format: table  # Options: json, table, text
  color_output: true
```

**`.shark.json.example`**:
```json
{
  "database": {
    "path": "./shark-tasks.db",
    "max_open_conns": 25,
    "max_idle_conns": 5,
    "timeout": "30s"
  },
  "server": {
    "port": "8080",
    "read_timeout": "5s",
    "write_timeout": "10s"
  },
  "cli": {
    "default_format": "table",
    "color_output": true
  }
}
```

### Integration Points

- **Environment Loading**: Built in T020
- **Config Struct**: Defined in T019
- **Viper**: Already configured for environment variables
- **User Workflow**: Optional convenience for users preferring config files

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Unit Tests**:
- Test loading from YAML file
- Test loading from JSON file
- Test config file in different search paths
- Test environment variables override file values
- Test missing config file uses defaults
- Test invalid YAML/JSON returns error
- Run: `go test ./internal/config/...`

**Manual Testing**:
- Create `.shark.yaml` and verify it's loaded
- Set environment variable and verify it overrides file
- Remove config file and verify defaults work
- Test in different directories (search path logic)

**Documentation**:
- Example config files are valid and well-commented
- README documents config file usage
- Precedence order clearly explained

## Context & Resources

- **PRD**: [Configuration Management](../01-feature-prd.md#fr-4-configuration-management)
- **PRD**: [Config File Example](../01-feature-prd.md#fr-4-configuration-management)
- **Task Dependency**: [T020 - Config Loading](./T020-implement-config-loading-environment.md)
- **Viper**: [Config Files](https://github.com/spf13/viper#reading-config-files)
- **12-Factor**: [Config](https://12factor.net/config)

## Notes for Agent

- Config file support is optional - don't require it
- Viper handles YAML/JSON parsing automatically
- `ConfigFileNotFoundError` is expected and OK (not an error)
- Search paths: current dir → home dir → /etc/shark/
- Environment variables must override file values (use `AutomaticEnv()` after `ReadInConfig()`)
- Provide good example files with comments explaining each option
- Example files should have `.example` extension (don't commit actual `.shark.yaml`)
- This gives users flexibility: env vars (Docker), config file (local dev), or defaults
- Config file is a convenience feature, not required functionality
