# Shark CLI Reference

Complete command reference for the Shark Task Manager CLI.

---

## Quick Start

### For AI Agents (DevAgent)

```bash
# Get next task and start working
shark next --json                          # Get next task (machine-readable)
shark start E07-F01-001 --json             # Start task
shark done E07-F01-001 --notes="Done" --json  # Complete task

# Extract specific fields
shark next --field key                     # Just the task key
shark get E07-F01-001 --field status       # Just the status
```

### For Human Developers

```bash
# Daily workflow
shark status                               # Project dashboard
shark next                                 # What should I work on?
shark start E07-F01-001                    # Start working
shark done E07-F01-001 --notes="Finished"  # Mark complete

# Browse project
shark list                                 # List epics
shark list E07                             # List features in epic
shark progress E07                         # Epic progress
```

---

## Command Groups (E17)

The CLI is organized into 4 groups. Commands are categorized by purpose, not by entity type.

### Workflow

Commands for day-to-day task lifecycle operations.

| Command | Description | Standard Equivalent |
|---------|-------------|---------------------|
| `shark next` | Get next available task | `shark task next` |
| `shark start <key>` | Start a task | `shark task start` |
| `shark done <key>` | Complete a task | `shark task complete` |
| `shark block <key>` | Block a task | `shark task block` |
| `shark unblock <key>` | Unblock a task | `shark task unblock` |
| `shark status [key]` | Project dashboard or entity status | - |
| `shark status set <key> <status>` | Set entity status directly | - |
| `shark status advance <key>` | Advance to next status | - |
| `shark status options <key>` | Show valid next statuses | - |
| `shark status history <key>` | View status change history | - |

**Reference:** [Quick Commands](quick-commands.md) | [Status Commands](status-commands.md)

### Inspect

Commands for viewing and searching project data.

| Command | Description |
|---------|-------------|
| `shark get <key>` | Get entity details (epic/feature/task) |
| `shark list [epic] [feature]` | List entities with smart filtering |
| `shark view <key>` | View entity markdown file |
| `shark progress <key>` | Detailed progress breakdown |
| `shark search <query>` | Search across entities |

**Reference:** [Core Commands](core-commands.md) | [Progress & Analytics](progress-analytics.md) | [Discovery Commands](discovery-commands.md)

### Manage

Commands for creating, updating, and organizing entities.

| Command | Description |
|---------|-------------|
| `shark create <type> [args]` | Create epic, feature, or task |
| `shark update <key> [flags]` | Update entity (auto-detects epic/feature/task from key) |
| `shark delete <key>` | Delete an entity |
| `shark idea` | Manage ideas (create, list, get, update, delete, promote) |
| `shark context get/set/clear <key>` | Manage entity context fields |
| `shark notes <key>` | View entity notes |
| `shark related-docs` | Manage related documents |
| `shark docs` | Alias for `shark related-docs` |
| `shark analytics [key]` | Project or entity analytics |
| `shark history <key>` | View entity history |

**Note:** `shark update` does not accept a `--status` flag. Use `shark status set <key> <status>` instead.

**Reference:** [Core Commands](core-commands.md) | [Context Commands](context-commands.md) | [Discovery Commands](discovery-commands.md)

### Advanced

Full entity-specific subcommands and administrative tools.

#### Entity Commands

| Entity | Subcommands | Reference |
|--------|-------------|-----------|
| `shark task` | 26 subcommands (create, get, list, start, complete, block, deps, note, ...) | [Task Commands](task-commands.md) |
| `shark feature` | 13 subcommands (create, get, list, complete, context, note, ...) | [Feature Commands](feature-commands.md) |
| `shark epic` | 14 subcommands (create, get, list, complete, context, note, status, ...) | [Epic Commands](epic-commands.md) |

**Note:** `shark task update`, `shark feature update`, and `shark epic update` no longer accept `--status`. Use `shark status set <key> <status>` instead.

#### Admin Commands (`shark admin`)

Setup, maintenance, and configuration commands are grouped under `shark admin`.

| Command | Description |
|---------|-------------|
| `shark admin init [--non-interactive]` | Initialize project |
| `shark admin init update [--workflow=basic\|advanced]` | Update workflow configuration |
| `shark admin validate` | Validate project structure |
| `shark admin migrate slugs` | Backfill slugs for all entities |
| `shark admin cloud init/status` | Cloud database management |
| `shark admin config show` | Show full configuration |
| `shark admin config validate` | Validate config file |
| `shark admin config get-format` | Get output format |
| `shark admin config get-status-action` | Debug workflow status actions |
| `shark admin workflow` | Workflow management |

**Reference:** [Setup Commands](setup-commands.md) | [Configuration](configuration.md)

---

## Global Flags

Available on all commands:

| Flag | Description |
|------|-------------|
| `--json` | Machine-readable JSON output |
| `--field <name>` | Extract single field (implies `--json`) |
| `--no-color` | Disable colored output |
| `--verbose` / `-v` | Debug logging |
| `--db <path>` | Override database path |
| `--config <path>` | Override config path |

**Reference:** [Global Flags](global-flags.md)

---

## Reference Documentation

| Topic | Description |
|-------|-------------|
| [Key Formats](key-formats.md) | Case-insensitive keys, slug format, short format |
| [JSON Output](json-output.md) | JSON response structures |
| [Error Messages](error-messages.md) | Common errors, exit codes, JSON errors |
| [Best Practices](best-practices.md) | AI agent and human developer patterns |
| [Workflow Configuration](workflow-configuration.md) | Status flows, phases, agent routing |
| [Configuration](configuration.md) | `.sharkconfig.json` and `shark config` commands |
| [Template System](template-system.md) | Entity file templates |
| [Orchestrator Actions](orchestrator-actions.md) | Status-based orchestrator routing |
| [Interactive Mode](interactive-mode.md) | Interactive prompt configuration |
| [File Paths](file-paths.md) | Custom file path organization |
| [Rejection Reasons](rejection-reasons.md) | Task rejection workflow |

---

## Key Concepts

### Entity Hierarchy

```
Epic (E07)
  └── Feature (E07-F01)
        └── Task (E07-F01-001 or T-E07-F01-001)
```

### Key Format Auto-Detection

Shark detects entity type from key format:

| Pattern | Entity | Example |
|---------|--------|---------|
| `E##` | Epic | `E07`, `E07-user-management` |
| `E##-F##` | Feature | `E07-F01`, `F01` |
| `E##-F##-###` | Task | `E07-F01-001`, `T-E07-F01-001` |

All keys are **case insensitive**.

### Dual Command Style

Every quick command has a standard equivalent:

```bash
shark next          # Quick style
shark task next     # Standard style (identical behavior)
```

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Not found |
| `2` | Database error |
| `3` | Invalid state |
| `4` | Field not found |
