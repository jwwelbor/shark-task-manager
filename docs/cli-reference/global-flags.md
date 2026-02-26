# Global Flags

All Shark CLI commands support the following global flags:

## Available Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--json` | | Output in JSON format (machine-readable) | `false` |
| `--field <name>` | | Extract a single field from JSON output | |
| `--no-color` | | Disable colored output | `false` |
| `--verbose` | `-v` | Enable debug logging | `false` |
| `--db <path>` | | Override database file path | `shark-tasks.db` |
| `--config <path>` | | Override config file path | `.sharkconfig.json` |

## Flag Details

### `--json`

Output results in machine-readable JSON format. Required for AI agents and automated scripts.

```bash
shark list --json
shark get E07 --json
shark task list --json
```

### `--field`

Extract a single field from JSON output. Implicitly enables `--json`. Useful for scripting and piping.

```bash
# Get just the status of a task
shark get E07-F01-001 --field status
# Output: "in_progress"

# Get just the key of the next task
shark next --field key
# Output: "E17-F11-001"

# Get progress percentage
shark progress E07 --field weighted_progress
```

### `--no-color`

Disable ANSI color codes in output. Useful in CI/CD pipelines, log files, or when piping output.

```bash
shark status --no-color
shark list E07 --no-color | tee output.txt
```

### `--verbose` / `-v`

Enable verbose/debug logging. Shows internal operations, database queries, and timing information.

```bash
shark task start E07-F01-001 -v
shark validate --verbose
```

### `--db`

Override the database file path. Defaults to `shark-tasks.db` in the project root.

```bash
shark task list --db=/path/to/custom.db
shark status --db=./test-shark-tasks.db
```

### `--config`

Override the configuration file path. Defaults to `.sharkconfig.json` in the project root.

```bash
shark task list --config=/path/to/.sharkconfig.json
shark status --config=.sharkconfig.prod.json
```

## Environment Variables

| Variable | Equivalent Flag | Description |
|----------|----------------|-------------|
| `SHARK_OUTPUT` | `--json` | Set to `json` for JSON output by default |

```bash
# Set JSON output globally
export SHARK_OUTPUT=json
shark task list  # Now outputs JSON without --json flag
```

## Usage Patterns

### AI Agent Configuration

AI agents should always use `--json` for reliable parsing:

```bash
shark next --json
shark get E07-F01-001 --json --field status
shark task start E07-F01-001 --json
```

### CI/CD Pipelines

```bash
shark validate --json --no-color
shark task list --json --no-color --db="$CI_DB_PATH"
```

### Human Developer

```bash
# Default colored output
shark status
shark list E07

# Quick field extraction
shark get E07-F01-001 --field status
```

## Related Documentation

- [Best Practices](best-practices.md) - AI agent best practices
- [JSON Output Format](json-output.md) - JSON response structures
- [Configuration](configuration.md) - Configuration file reference
