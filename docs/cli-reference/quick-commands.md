# Quick Commands

Shark provides shortcut aliases for the most commonly used task operations. These quick commands save keystrokes by removing the `task` subcommand from frequently used workflows.

Both the quick command and its full equivalent behave identically -- same flags, same output, same behavior. Use whichever style you prefer.

## Command Reference

### shark next

Get the next available task based on priority, dependencies, and agent type.

**Full equivalent:** `shark task next`

**Usage:**

```
shark next [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | `-a` | Agent type to match |
| `--epic` | `-e` | Filter by epic key |
| `--help` | `-h` | Help for next |

**Examples:**

```bash
# Get the next available task
shark next

# Get the next task for a specific agent type
shark next --agent=frontend

# Get the next task within a specific epic
shark next --epic=E07
```

---

### shark start

Mark a task as in_progress and update timestamps.

**Full equivalent:** `shark task start`

Use `--force` to bypass status transition validation. This allows starting a task from any status (not just 'todo'). Use with caution as this is an administrative override.

Supports multiple key formats (numeric, full, or slugged).

**Usage:**

```
shark start <task-key> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--agent` | Agent identifier (defaults to USER env var) |
| `--force` | Force status change bypassing validation (use with caution) |
| `--help` / `-h` | Help for start |

**Examples:**

```bash
# Start a task using short key format
shark start E07-F01-001

# Start a task using full key format
shark start T-E04-F01-001

# Start a task using slugged key format
shark start T-E04-F01-001-user-auth
```

---

### shark done

Mark a task as ready_for_review and update timestamps.

**Full equivalent:** `shark task complete`

Use `--force` to bypass status transition validation. This allows marking a task complete from any status (not just 'in_progress'). Use with caution as this is an administrative override.

Supports multiple key formats (numeric, full, or slugged).

**Usage:**

```
shark done <task-key> [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | | Agent identifier (defaults to USER env var) |
| `--agent-id` | | Agent execution ID for traceability |
| `--files-created` | | Files created during task (repeatable) |
| `--files-modified` | | Files modified during task (repeatable) |
| `--force` | | Force status change bypassing validation (use with caution) |
| `--help` | `-h` | Help for done |
| `--notes` | `-n` | Completion notes |
| `--summary` | | Completion summary describing what was delivered |
| `--tests` | | Test status summary (e.g., '16/16 passing') |
| `--time-spent` | | Time spent in minutes |
| `--verified` | | Mark task as verified |

**Examples:**

```bash
# Mark a task as done
shark done E07-F01-001

# Mark done with completion notes
shark done T-E04-F01-001 --notes "Implemented feature X"

# Mark done with a delivery summary
shark done T-E04-F01-001-user-auth --summary "Added JWT authentication"
```

---

### shark block

Mark a task as blocked with a required reason.

**Full equivalent:** `shark task block`

Use `--force` to bypass status transition validation. This allows blocking a task from any status (not just 'todo' or 'in_progress'). Use with caution as this is an administrative override.

**Usage:**

```
shark block <task-key> [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | | Agent identifier (defaults to USER env var) |
| `--force` | | Force status change bypassing validation (use with caution) |
| `--help` | `-h` | Help for block |
| `--reason` | `-r` | Reason for blocking (required) |

**Examples:**

```bash
# Block a task with a reason
shark block E07-F01-001 --reason "Waiting for API spec"

# Block using the short flag
shark block T-E04-F01-001 -r "Dependencies not ready"

# Force-block a task regardless of current status
shark block E07-F01-001 --reason "Pausing work" --force
```

---

### shark unblock

Unblock a task and return it to draft status.

**Full equivalent:** `shark task unblock`

Use `--force` to bypass status transition validation. This allows unblocking a task from any status (not just 'blocked'). Use with caution as this is an administrative override.

**Usage:**

```
shark unblock <task-key> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--agent` | Agent identifier (defaults to USER env var) |
| `--force` | Force status change bypassing validation (use with caution) |
| `--help` / `-h` | Help for unblock |

**Examples:**

```bash
# Unblock a task
shark unblock E07-F01-001

# Unblock using the full key format
shark unblock T-E04-F01-001

# Force-unblock a task regardless of current status
shark unblock E07-F01-001 --force
```

## Quick Command to Full Command Mapping

| Quick Command | Full Command | Description |
|---------------|--------------|-------------|
| `shark next` | `shark task next` | Get next available task |
| `shark start <key>` | `shark task start <key>` | Start a task (set to in_progress) |
| `shark done <key>` | `shark task complete <key>` | Complete a task (set to ready_for_review) |
| `shark block <key>` | `shark task block <key>` | Block a task with a reason |
| `shark unblock <key>` | `shark task unblock <key>` | Unblock a blocked task |

Both styles accept the same flags and produce identical output. Quick commands are purely syntactic shortcuts -- there is no difference in behavior.

## Global Flags

All quick commands support the standard global flags:

| Flag | Description |
|------|-------------|
| `--config` | Config file path (default: .sharkconfig.json) |
| `--db` | Database file path (default: shark-tasks.db) |
| `--field` | Extract a single field from JSON output (e.g., `--field status`) |
| `--json` | Output in JSON format (machine-readable) |
| `--no-color` | Disable colored output |
| `--verbose` / `-v` | Enable verbose/debug output |

## Related Documentation

- [Task Commands](task-commands.md) - Full task command reference
- [Key Formats](key-formats.md) - Supported key formats and case sensitivity
- [Global Flags](global-flags.md) - All global flags
- [Best Practices](best-practices.md) - AI agent best practices
