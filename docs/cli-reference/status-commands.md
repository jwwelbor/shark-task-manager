# Status Commands

Commands for managing entity status transitions. Use `shark progress` for project
or epic dashboards and `shark get <key>` for feature or other entity details.

## Quick Reference

- `shark status` - Status operation namespace
- `shark status set` - Set an entity to a specific status
- `shark status advance` - Advance an entity to the next workflow status
- `shark status transitions` - Show available status transitions (read-only)
- `shark status history` - Show status change history for a task

## shark status

`shark status` is a subcommand namespace for status operations. Running it
without a subcommand displays help; it does not display a dashboard or inspect
an entity.

Use `shark progress [EPIC]` for project or epic dashboards.
Use `shark get <key> --field status` for an entity's current status, or
`shark get <key>` for its full details.

### Usage

```
shark status <set|advance|transitions|history> ...
```

### Related inspection commands

```bash
# Project or epic dashboard
shark progress
shark progress E05

# Entity details or current status
shark get E05-F02
shark get E05-F02 --field status
```

## shark status set

Set an epic, feature, or task to a specific status. Entity type is auto-detected from the key format.

This command is **idempotent**: if the entity is already at the target status, it returns exit code 0 with `"changed": false` in JSON output.

> **Note:** unlike `shark status advance`, this command does **not** check for an
> open blocking Question on the target entity. Use `shark status advance` when a
> Question-blocking gate must be enforced.

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

Advance an epic, feature, or task through its configured workflow.

When an entity has multiple valid next statuses, this command auto-selects the first one. Route-based workflows may also release semantic outcomes with `--outcome`. When `advance_guard.enabled` is `true` in `.sharkconfig.json`, parent-loop usage must include `--session` and `--from-status`.

### Usage

```
shark status advance <key> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--agent <name>` | Agent or user performing the transition |
| `--force-repeat` | Override a replay rejection when `advance_guard.allow_repeat_with_force` is enabled; requires `--reason` |
| `--from-status <status>` | Expected current status for guarded advances |
| `--outcome <name>` | Release a route-based outcome such as `pass`, `fail`, or `blocked` |
| `--reason <text>` | Reason for backward or forced transitions |
| `--session <sid>` | Lease/session id for guarded advances |
| `-h, --help` | Help for advance |

### Examples

```bash
# Auto-advance a task to its next workflow status
shark status advance E07-F01-001

# Release a route-based outcome
shark status advance E07-F01-001 --outcome pass

# Guarded parent-loop advance (required when advance_guard.enabled=true)
shark status advance E07-F01-001 --outcome fail --session "$SID" --from-status code_review

# Audited replay override when allow_repeat_with_force=true
shark status advance E07-F01-001 --outcome fail --session "$SID" --from-status code_review --force-repeat --reason="manual override"
```

## shark status transitions

Show the available status transitions for an epic, feature, or task without making any changes. This is a read-only command useful for understanding what transitions are currently valid before performing one.

### Usage

```
shark status transitions <key> [flags]
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
shark status transitions E07-F01-001

# Show available transitions for an epic
shark status transitions E07

# JSON output for scripting
shark status transitions E07-F01 --json
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
