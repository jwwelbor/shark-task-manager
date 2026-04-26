# Shark CLI Reference

Complete command reference for the Shark Task Manager CLI.

---

## Quick Start

### For AI Agents (DevAgent)

```bash
# Advance workflow and set status
shark status advance E07-F01-001 --json             # Advance task to next workflow status
shark status set E07-F01-001 in_development --json  # Set task status directly
shark status advance B001 --json                    # Advance bug to next workflow status
shark status advance CC-001 --json                  # Advance change-card to next workflow status

# Extract specific fields
shark get E07-F01-001 --field status       # Just the status
shark get E07-F01-001 --field key          # Just the key
```

### For Human Developers

```bash
# Daily workflow
shark status                               # Project dashboard
shark status advance E07-F01-001           # Advance to next workflow status
shark status set E07-F01-001 completed     # Set status directly

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
| `shark status [key]` | Project dashboard or entity status | - |
| `shark status set <key> <status>` | Set entity status directly | `shark task set-status` |
| `shark status advance <key>` | Advance to next workflow status | `shark task next-status` |
| `shark status options <key>` | Show valid next statuses | - |
| `shark status history <key>` | View status change history | - |

**Reference:** [Status Commands](status-commands.md)

### Inspect

Commands for viewing and searching project data.

| Command | Description |
|---------|-------------|
| `shark get <key>` | Get entity details (epic/feature/task) |
| `shark list [epic] [feature]` | List entities with smart filtering |
| `shark view <key>` | View entity markdown file |
| `shark search <query>` | Search across entities |

**Reference:** [Core Commands](core-commands.md) | [Progress & Analytics](progress-analytics.md) | [Discovery Commands](discovery-commands.md)

### Manage

Commands for creating, updating, and organizing entities.

| Command | Description |
|---------|-------------|
| `shark create <type> [args]` | Create epic, feature, or task |
| `shark update <key> [flags]` | Update entity (auto-detects epic/feature/task from key) |
| `shark delete <key>` | Delete an entity |
| `shark context get/set/clear <key>` | Manage entity context fields |
| `shark notes <key>` | View entity notes |
| `shark related-docs` | Manage related documents |
| `shark docs` | Alias for `shark related-docs` |
| `shark history <key>` | View entity history |

**Note:** `shark update` does not accept a `--status` flag. Use `shark status set <key> <status>` instead.

**Reference:** [Core Commands](core-commands.md) | [Context Commands](context-commands.md) | [Discovery Commands](discovery-commands.md)

### Vocabulary

Commands for managing the closed tag vocabulary and applying registered tags to entities.

| Command | Description | Reference |
|---------|-------------|-----------|
| `shark tags list` | List all registered tag names | [Tags](tags.md) |
| `shark tags add <name>` | Register a new tag name (auth required) | [Tags](tags.md) |
| `shark tags rm <name>` | Remove a tag from the vocabulary (auth required) | [Tags](tags.md) |
| `shark tags rename <old> <new>` | Rename a tag across all entities (auth required) | [Tags](tags.md) |
| `shark <entity> tag add <key> <name>` | Retroactively attach a tag to an existing entity (task, feature, epic, bug, change, idea) | [Tags](tags.md#retroactive-tagging-shark-entity-tag-addrm) |
| `shark <entity> tag rm <key> <name>` | Retroactively detach a tag from an existing entity | [Tags](tags.md#retroactive-tagging-shark-entity-tag-addrm) |

All six `create` and `update` commands also accept a repeatable `--tag <name>` flag. See [Applying Tags During Create/Update](tags.md#applying-tags-during-createupdate). Tags can be made mandatory on creation per entity type via the [`tag_required_for`](configuration.md#tag_required_for) config field.

**Reference:** [Tags Commands](tags.md)

### Advanced

Full entity-specific subcommands, analytics, and administrative tools.

| Command | Description |
|---------|-------------|
| `shark idea` | Manage ideas (create, list, get, update, delete, promote) |
| `shark analytics [key]` | Project or entity analytics |
| `shark progress <key>` | Detailed progress breakdown |

#### Entity Commands

All six entity families support the `--tag <name>` repeatable flag on `create` and `update`, plus a `tag add|rm <key> <name>` subcommand for retroactive tagging. See [Tags](tags.md).

| Entity | Subcommands | Reference |
|--------|-------------|-----------|
| `shark task` | 20 subcommands (create, get, list, update, delete, next-status, set-status, tag, context, deps, blocked-by, blocks, history, link, note, notes, timeline, resume, sessions, unlink) | [Task Commands](task-commands.md) |
| `shark feature` | 13 subcommands (create, get, list, complete, context, note, tag, ...) | [Feature Commands](feature-commands.md) |
| `shark epic` | 14 subcommands (create, get, list, complete, context, note, status, tag, ...) | [Epic Commands](epic-commands.md) |
| `shark bug` | 10 subcommands (create, get, list, update, delete, triage, note, notes, context, tag) | [Bug Commands](bug-commands.md) |
| `shark change` | 10 subcommands (create, get, list, update, delete, approve, note, notes, context, tag) | [Change Commands](change-commands.md) |
| `shark idea` | 7 subcommands (create, get, list, update, delete, convert, tag) | [Idea Commands](idea-commands.md) |

**Note:** `shark task update`, `shark feature update`, `shark epic update`, `shark bug update`, and `shark change update` do not accept `--status`. Use `shark status set <key> <status>` instead.

#### Admin Commands (`shark admin`)

Setup, maintenance, and configuration commands are grouped under `shark admin`.

| Command | Description |
|---------|-------------|
| `shark admin init [--non-interactive] [--force]` | Initialize project / re-sync templates |
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

Bug (B001)           ← standalone, optionally linked to any entity
Change-Card (CC-001) ← standalone, optionally linked to epic or feature
```

### Key Format Auto-Detection

Shark detects entity type from key format:

| Pattern | Entity | Example |
|---------|--------|---------|
| `E##` | Epic | `E07`, `E07-user-management` |
| `E##-F##` | Feature | `E07-F01`, `F01` |
| `E##-F##-###` | Task | `E07-F01-001`, `T-E07-F01-001` |
| `B###` | Bug | `B001`, `B042` |
| `CC-###` | Change-Card | `CC-001`, `CC-042` |

All keys are **case insensitive**.

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Not found |
| `2` | Database error |
| `3` | Invalid state |
| `4` | Field not found |
