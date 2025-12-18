# Shark CLI - Project Manager Command Line Interface

The PM (Project Manager) CLI is a command-line tool for managing tasks, epics, and features in multi-agent software development projects.

## Installation

### Build from Source

```bash
make pm
```

### Install to PATH

```bash
make install-pm
```

This installs the `shark` command to `~/go/bin/shark`. Ensure `~/go/bin` is in your PATH.

## Quick Start

```bash
# View available commands
pm --help

# View task commands
pm task --help

# List all tasks (when implemented)
pm task list

# Get JSON output
pm --json task list

# Disable colors
pm --no-color task list

# Enable verbose mode
pm --verbose task list
```

## Global Flags

These flags are available on all commands:

- `--json` - Output in JSON format (machine-readable, perfect for AI agents)
- `--no-color` - Disable colored output
- `--verbose` / `-v` - Enable verbose/debug output
- `--config <path>` - Specify custom config file (default: `.pmconfig.json`)
- `--db <path>` - Specify database file path (default: `shark-tasks.db`)

## Command Structure

The CLI follows a hierarchical command structure: `pm <resource> <action>`

### Resources

- `task` - Task lifecycle operations
- `epic` - Epic queries and management
- `feature` - Feature queries and management
- `config` - Configuration management

## Commands

### Task Commands

```bash
pm task list                    # List all tasks
pm task list --status=todo      # Filter by status
pm task list --epic=E04         # Filter by epic
pm task get T-E01-F01-001      # Get task details
pm task create                  # Create a new task
pm task start T-E01-F01-001    # Start working on a task
pm task complete T-E01-F01-001 # Mark task as complete
pm task next                    # Get next available task
pm task next --agent=frontend   # Get next task for frontend agent
```

### Epic Commands

```bash
pm epic list                    # List all epics
pm epic get E04                # Get epic details
pm epic status                  # Show status summary
```

See [Epic & Feature Query Documentation](EPIC_FEATURE_QUERIES.md) for detailed usage.

### Feature Commands

```bash
pm feature list                 # List all features
pm feature list --epic=E04     # List features in epic
pm feature list --status=active # Filter by status
pm feature get E04-F02         # Get feature details
```

See [Epic & Feature Query Documentation](EPIC_FEATURE_QUERIES.md) for detailed usage.

### Config Commands

```bash
pm config show                  # Show current configuration
pm config validate              # Validate config file
```

## Configuration File

Create a `.pmconfig.json` file in your project root:

```json
{
  "json": false,
  "no-color": false,
  "verbose": false,
  "db": "shark-tasks.db"
}
```

## Environment Variables

Override configuration with environment variables using the `PM_` prefix:

```bash
export PM_JSON=true
export PM_DB=/path/to/database.db
pm task list
```

## Output Formats

### Human-Readable (Default)

Tables with colored output for terminal use.

### JSON (--json flag)

Structured JSON output for programmatic consumption:

```bash
pm --json config show
{
  "db": "shark-tasks.db",
  "json": true,
  "no-color": false,
  "verbose": false
}
```

## Implementation Status

### ✅ Implemented (E04-F02)

- CLI framework with Cobra
- Hierarchical command structure
- Global flags (--json, --no-color, --verbose, --config, --db)
- Configuration system with Viper
- Output formatting with pterm
- Command groups (task, epic, feature, config)
- Help text generation
- Basic config commands

### 🚧 Coming Soon

- **E04-F03**: Task lifecycle operations (list, get, start, complete, next)
- **E04-F04**: Epic & feature queries with progress calculation
- **E04-F05**: File path management
- **E04-F06**: Task creation & templating
- **E04-F07**: Database initialization & sync
- **E05-F01**: Status dashboard
- **E05-F02**: Dependency management
- **E05-F03**: History & audit trail

## Development

### Running Tests

```bash
cd internal/cli
go test -v
```

### Building

```bash
make pm
```

### Code Structure

```
cmd/pm/              # CLI entry point
  main.go
internal/cli/        # CLI implementation
  root.go           # Root command and global flags
  commands/         # Command definitions
    task.go        # Task commands
    epic.go        # Epic commands
    feature.go     # Feature commands
    config.go      # Config commands
```

## For AI Agents

The CLI is optimized for AI agent usage:

1. **JSON Output**: Use `--json` flag for machine-readable output
2. **Exit Codes**: Commands return appropriate exit codes (0=success, 1=error)
3. **Structured Errors**: Error messages are clear and actionable
4. **No Interactive Prompts**: All commands work non-interactively
5. **Fast Startup**: CLI startup time <50ms

Example agent usage:

```bash
# Get next task as JSON
pm --json task next --agent=backend

# Parse and execute
TASK=$(pm --json task next --agent=backend | jq -r '.key')
pm task start $TASK
# ... do work ...
pm task complete $TASK
```

## Shell Completion

Cobra provides built-in shell completion. Generate completion scripts:

```bash
# Bash
pm completion bash > /etc/bash_completion.d/pm

# Zsh
pm completion zsh > "${fpath[1]}/_pm"

# Fish
pm completion fish > ~/.config/fish/completions/pm.fish
```

## Additional Documentation

### Epic & Feature Queries

Comprehensive documentation for querying epics and features:

- [Epic & Feature Query Guide](EPIC_FEATURE_QUERIES.md) - Complete guide with detailed explanations
- [Quick Reference](EPIC_FEATURE_QUICK_REFERENCE.md) - Fast lookup for common commands
- [Examples](EPIC_FEATURE_EXAMPLES.md) - Real-world usage scenarios and workflows

### Other Resources

- [Database Implementation](DATABASE_IMPLEMENTATION.md) - Database schema and design
- [Testing Guide](TESTING.md) - How to test the application
- [Development Guidelines](development/testing-guidelines.md) - Development best practices

## License

See LICENSE file for details.
