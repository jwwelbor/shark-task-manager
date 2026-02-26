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

## Command Categories

### Quick Commands

Shorthand aliases for the most common task operations.

| Command | Description | Standard Equivalent |
|---------|-------------|---------------------|
| `shark next` | Get next available task | `shark task next` |
| `shark start <key>` | Start a task | `shark task start` |
| `shark done <key>` | Complete a task | `shark task complete` |
| `shark block <key>` | Block a task | `shark task block` |
| `shark unblock <key>` | Unblock a task | `shark task unblock` |

**Reference:** [Quick Commands](quick-commands.md)

### Core Commands

Entity-aware commands that auto-detect type from key format.

| Command | Description |
|---------|-------------|
| `shark get <key>` | Get entity details (epic/feature/task) |
| `shark list [epic] [feature]` | List entities with smart filtering |
| `shark create <type> [args]` | Create epic, feature, or task |
| `shark delete <key>` | Delete an entity |
| `shark view <key>` | View entity markdown file |

**Reference:** [Core Commands](core-commands.md)

### Status & Analytics

| Command | Description |
|---------|-------------|
| `shark status [key]` | Project dashboard or entity status |
| `shark status set <key> <status>` | Set entity status directly |
| `shark status advance <key>` | Advance to next status |
| `shark status options <key>` | Show valid next statuses |
| `shark status history <key>` | View status change history |
| `shark progress <key>` | Detailed progress breakdown |
| `shark analytics [key]` | Project or entity analytics |

**Reference:** [Status Commands](status-commands.md) | [Progress & Analytics](progress-analytics.md)

### Entity Management

Full CRUD and lifecycle commands for each entity type.

| Entity | Subcommands | Reference |
|--------|-------------|-----------|
| `shark task` | 26 subcommands (create, get, list, start, complete, block, deps, note, ...) | [Task Commands](task-commands.md) |
| `shark feature` | 13 subcommands (create, get, list, complete, context, note, ...) | [Feature Commands](feature-commands.md) |
| `shark epic` | 14 subcommands (create, get, list, complete, context, note, status, ...) | [Epic Commands](epic-commands.md) |
| `shark idea` | 6 subcommands (create, get, list, update, delete, promote) | [Idea Commands](idea-commands.md) |

### Details & Discovery

| Command | Description |
|---------|-------------|
| `shark context get/set/clear <key>` | Manage entity context fields |
| `shark search <query>` | Search across entities |
| `shark notes <key>` | View entity notes |
| `shark related-docs` | Manage related documents |

**Reference:** [Context Commands](context-commands.md) | [Discovery Commands](discovery-commands.md)

### Setup & Maintenance

| Command | Description |
|---------|-------------|
| `shark init` | Initialize project |
| `shark validate` | Validate project structure |
| `shark migrate` | Run database migrations |
| `shark cloud` | Cloud database management |
| `shark config` | Configuration management |

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
