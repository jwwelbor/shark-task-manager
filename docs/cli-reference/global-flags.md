# Global Flags

All Shark CLI commands support the following global flags:

## Available Flags

- `--json`: Output results in machine-readable JSON format (required for AI agents)
- `--no-color`: Disable colored output
- `--verbose` / `-v`: Enable debug logging
- `--db <path>`: Override database path (default: `shark-tasks.db`)
- `--config <path>`: Override config file path (default: `.sharkconfig.json`)

## Examples

```bash
# JSON output with verbose logging
shark task list --json --verbose

# Use custom database
shark task list --db=/path/to/custom.db

# Use custom config
shark task list --config=/path/to/.sharkconfig.json

# Disable colors (useful for logs)
shark task list --no-color
```

## When to Use

- **--json**: Always use for AI agents and automated scripts
- **--verbose**: Use for debugging and troubleshooting
- **--no-color**: Use in CI/CD pipelines or when piping output
- **--db**: Use to work with multiple databases or custom locations
- **--config**: Use to switch between different project configurations

## Related Documentation

- [Best Practices](best-practices.md) - AI agent best practices
- [JSON Output Format](json-output.md) - JSON response structures
