# Status Commands

Commands for viewing status dashboards and managing entity status transitions.

## Quick Reference

- `shark status` - Display status dashboard for project, epic, or feature
- `shark status set` - Set an entity to a specific status
- `shark status advance` - Advance an entity to the next workflow status
- `shark status options` - Show available status transitions (read-only)
- `shark status history` - Show status change history for a task

## shark status

Display a comprehensive status dashboard showing project progress, active tasks, and blocked items. The scope is determined by positional arguments: no arguments shows the full project, an epic key scopes to that epic, and adding a feature key scopes to that feature.

### Usage

```
shark status [EPIC] [FEATURE] [flags]
```

### Positional Arguments

| Argument | Description |
|----------|-------------|
| _(none)_ | Show full project dashboard |
| `EPIC` | Show status for a specific epic (e.g., `E05`) |
| `EPIC FEATURE` | Show status for a specific feature (e.g., `E05 F02` or `E05-F02`) |

### Flags

| Flag | Description |
|------|-------------|
| `--epic <key>` | Filter by epic key (flag syntax, still supported) |
| `--include-archived` | Include archived epics and features |
| `--recent <window>` | Recent completion window (`24h`, `7d`, `30d`, `90d`) |
| `-h, --help` | Help for status |

### Examples

```bash
# Show full project status dashboard
shark status

# Show status for epic E05
shark status E05

# Show status for feature E05-F02 (two positional args)
shark status E05 F02

# Show status for feature E05-F02 (combined format)
shark status E05-F02

# Include completions from the last 7 days
shark status --recent=7d

# JSON output for scripting
shark status --json
```

### Relationship to `shark progress`

The `shark status` command focuses on current status, active work, and blocked items. The separate `shark progress` command provides a progress-oriented dashboard with health indicators and completion metrics. Both accept the same positional arguments and filtering flags. Use `status` when you want a snapshot of current state; use `progress` when you want completion percentages and health summaries.

## shark status set

Set an epic, feature, or task to a specific status. Entity type is auto-detected from the key format.

This command is **idempotent**: if the entity is already at the target status, it returns exit code 0 with `"changed": false` in JSON output.

### Usage

```
shark status set <key> <status> [flags]
```

### Key Formats

| Format | Entity Type |
|--------|-------------|
| `E07` | Epic |
| `E07-F01` | Feature |
| `E07-F01-001` | Task |

### Flags

| Flag | Description |
|------|-------------|
| `--agent <name>` | Agent performing the transition |
| `--force` | Bypass workflow validation |
| `--notes <text>` | Transition notes |
| `--reason <text>` | Reason for backward or forced transitions |
| `-h, --help` | Help for set |

### Examples

```bash
# Set a task to in_development
shark status set E07-F01-001 in_development

# Force a backward transition with a reason
shark status set E07-F01-001 todo --reason="Needs rework" --force

# Set an epic status with JSON output
shark status set E07 active --json
```

## shark status advance

Advance an epic, feature, or task through its configured workflow by selecting from available transitions.

When an entity has multiple valid next statuses, this command auto-selects the first one. For automation and scripting, use `--status` to specify the target directly. Use `--preview` to see available transitions without making changes.

### Usage

```
shark status advance <key> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--status <name>` | Target status for direct transition (non-interactive) |
| `--preview` | Show available transitions without making changes |
| `--force` | Bypass workflow validation (administrative override) |
| `--agent <name>` | Agent or user performing the transition |
| `--reason <text>` | Reason for backward or forced transitions |
| `-h, --help` | Help for advance |

### Examples

```bash
# Auto-advance a task to its next workflow status
shark status advance E07-F01-001

# Preview available transitions without changing anything
shark status advance E07-F01-001 --preview

# Advance directly to a specific status
shark status advance E07-F01-001 --status=in_development
```

## shark status options

Show the available status transitions for an epic, feature, or task without making any changes. This is a read-only command useful for understanding what transitions are currently valid before performing one.

### Usage

```
shark status options <key> [flags]
```

### Key Formats

| Format | Entity Type |
|--------|-------------|
| `E07` | Epic |
| `E07-F01` | Feature |
| `E07-F01-001` | Task |

### Flags

| Flag | Description |
|------|-------------|
| `-h, --help` | Help for options |

### Examples

```bash
# Show available transitions for a task
shark status options E07-F01-001

# Show available transitions for an epic
shark status options E07

# JSON output for scripting
shark status options E07-F01 --json
```

## shark status history

Show the status change history for a task. Only tasks have history records; epics and features do not track status change history.

### Usage

```
shark status history <task-key> [flags]
```

### Key Formats

| Format | Entity Type |
|--------|-------------|
| `E07-F01-001` | Task (short format) |
| `T-E07-F01-001` | Task (traditional format) |

### Flags

| Flag | Description |
|------|-------------|
| `--limit <n>` | Maximum number of history entries to show (default: 50) |
| `-h, --help` | Help for history |

### Examples

```bash
# Show full status history for a task
shark status history E07-F01-001

# Show only the last 10 status changes
shark status history E07-F01-001 --limit=10

# JSON output for scripting
shark status history E07-F01-001 --json
```

## Global Flags

All status commands support the standard global flags:

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (machine-readable) |
| `--field <name>` | Extract a single field from JSON output (e.g., `--field status`) |
| `--config <path>` | Config file path (default: `.sharkconfig.json`) |
| `--db <path>` | Database file path (default: `shark-tasks.db`) |
| `--no-color` | Disable colored output |
| `-v, --verbose` | Enable verbose/debug output |

## Related Documentation

- [Task Commands](task-commands.md) - Task lifecycle commands (`start`, `complete`, `approve`, `block`)
- [Workflow Configuration](workflow-config.md) - Customize status flows and transitions
- [Workflow Profiles](../guides/workflow-profiles.md) - Basic and advanced workflow profiles
- [Key Formats](key-formats.md) - Case insensitive keys, short formats
- [JSON Output](json-output.md) - JSON response structures
