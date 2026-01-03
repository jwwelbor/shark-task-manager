# Shark CLI Reference

Complete command reference for the Shark Task Manager CLI.

## Table of Contents

- [Global Flags](#global-flags)
- [Initialization](#initialization)
- [Epic Commands](#epic-commands)
- [Feature Commands](#feature-commands)
- [Task Commands](#task-commands)
- [Sync Commands](#sync-commands)
- [Configuration Commands](#configuration-commands)

---

## Global Flags

All commands support the following global flags:

- `--json`: Output results in machine-readable JSON format (required for AI agents)
- `--no-color`: Disable colored output
- `--verbose` / `-v`: Enable debug logging
- `--db <path>`: Override database path (default: `shark-tasks.db`)
- `--config <path>`: Override config file path (default: `.sharkconfig.json`)

**Example:**
```bash
shark task list --json --verbose
```

---

## Initialization

### `shark init`

Initialize Shark CLI infrastructure in the current project.

**Flags:**
- `--non-interactive`: Skip interactive prompts (recommended for automation)

**Example:**
```bash
# Interactive mode
shark init

# Non-interactive mode (for AI agents)
shark init --non-interactive
```

**Creates:**
- SQLite database (`shark-tasks.db`)
- Folder structure (`docs/plan/`)
- Configuration file (`.sharkconfig.json`)
- Templates directory (`shark-templates/`)

---

## Epic Commands

### `shark epic create`

Create a new epic.

**Required Flags:**
- `--title <string>`: Epic title

**Optional Flags:**
- `--file <path>`: Custom file path (relative to root, must include .md)
  - Aliases: `--filepath`, `--path`
- `--force`: Reassign file if already claimed by another epic or feature
- `--priority <1-10>`: Priority (1 = highest, 10 = lowest)
- `--business-value <1-10>`: Business value score
- `--json`: Output in JSON format

**Examples:**

```bash
# Create epic with default file path
shark epic create --title="User Management System"
# Creates: docs/plan/E07-user-management-system/epic.md

# Create epic with custom file path
shark epic create --title="Q1 2025 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# Create epic with priority and business value
shark epic create --title="Payment Integration" --priority=1 --business-value=10 --json

# Force reassign file (if already claimed)
shark epic create --title="Legacy Migration" --file="docs/legacy/epic.md" --force
```

**Aliases for --file flag:**
- `--filepath` (hidden)
- `--path` (hidden)

All three flags accept the same value: a complete file path including the `.md` extension.

---

### `shark epic list`

List all epics with progress information.

**Flags:**
- `--json`: Output in JSON format

**Examples:**

```bash
# List all epics (table format)
shark epic list

# List all epics (JSON format)
shark epic list --json
```

**JSON Output:**
```json
[
  {
    "id": 7,
    "key": "E07",
    "slug": "user-management-system",
    "title": "User Management System",
    "description": "Complete user management infrastructure",
    "file_path": "docs/plan/E07-user-management-system/epic.md",
    "priority": 1,
    "business_value": 10,
    "progress": 42.5,
    "created_at": "2026-01-02T10:00:00Z",
    "updated_at": "2026-01-02T15:30:00Z"
  }
]
```

---

### `shark epic get`

Get detailed information about a specific epic.

**Usage:**
```bash
shark epic get <epic-key> [--json]
```

**Supports:**
- Numeric keys: `E07`
- Slugged keys: `E07-user-management-system`

**Examples:**

```bash
# Get epic details (table format)
shark epic get E07

# Get epic details (JSON format)
shark epic get E07 --json

# Using slugged key
shark epic get E07-user-management-system --json
```

---

## Feature Commands

### `shark feature create`

Create a new feature within an epic.

**Required Flags:**
- `--epic <epic-key>`: Parent epic key
- `--title <string>`: Feature title

**Optional Flags:**
- `--file <path>`: Custom file path (relative to root, must include .md)
  - Aliases: `--filepath`, `--path`
- `--force`: Reassign file if already claimed by another feature or epic
- `--execution-order <number>`: Execution order within epic
- `--json`: Output in JSON format

**Examples:**

```bash
# Create feature with default file path
shark feature create --epic=E07 --title="Authentication"
# Creates: docs/plan/E07-user-management-system/E07-F01-authentication/feature.md

# Create feature with custom file path
shark feature create --epic=E07 --title="User Profiles" --file="docs/features/profiles/feature.md"

# Create feature with execution order
shark feature create --epic=E07 --title="Authorization" --execution-order=2 --json

# Force reassign file
shark feature create --epic=E07 --title="Legacy Auth" --file="docs/legacy/auth.md" --force
```

**Aliases for --file flag:**
- `--filepath` (hidden)
- `--path` (hidden)

---

### `shark feature list`

List features, optionally filtered by epic.

**Usage:**
```bash
shark feature list [EPIC] [--json]
# OR (flag syntax, backward compatible)
shark feature list [--epic=<epic-key>] [--json]
```

**Examples:**

```bash
# List all features
shark feature list

# List features in specific epic (positional argument)
shark feature list E07
shark feature list E07 --json

# List features in specific epic (flag syntax)
shark feature list --epic=E07 --json

# Using slugged epic key
shark feature list E07-user-management-system --json
```

---

### `shark feature get`

Get detailed information about a specific feature.

**Usage:**
```bash
shark feature get <feature-key> [--json]
```

**Supports:**
- Numeric keys: `E07-F01`, `F01`
- Slugged keys: `E07-F01-authentication`, `F01-authentication`

**Examples:**

```bash
# Get feature details
shark feature get E07-F01

# Get feature details (JSON)
shark feature get E07-F01 --json

# Using partial key
shark feature get F01

# Using slugged key
shark feature get E07-F01-authentication --json
```

---

## Task Commands

### `shark task create`

Create a new task within a feature.

**Required Flags:**
- `--epic <epic-key>`: Parent epic key
- `--feature <feature-key>`: Parent feature key
- `--title <string>`: Task title

**Optional Flags:**
- `--agent <type>`: Agent type (`frontend`, `backend`, `api`, `testing`, `devops`, `general`)
- `--priority <1-10>`: Priority (1 = highest, 10 = lowest, default: 5)
- `--description <string>`: Detailed description
- `--depends-on <task-keys>`: Comma-separated list of dependency task keys
- `--file <path>`: Custom file path (relative to root, must include .md)
  - Aliases: `--filepath`, `--filename`
- `--force`: Reassign file if already claimed by another task
- `--json`: Output in JSON format

**Examples:**

```bash
# Create task with defaults
shark task create --epic=E07 --feature=F01 --title="Implement JWT validation"

# Create task with agent and priority
shark task create \
  --epic=E07 \
  --feature=F01 \
  --title="Implement JWT validation" \
  --agent=backend \
  --priority=3

# Create task with dependencies
shark task create \
  --epic=E07 \
  --feature=F01 \
  --title="Add token refresh" \
  --agent=backend \
  --depends-on="T-E07-F01-001,T-E07-F01-002"

# Create task with custom file path
shark task create \
  --epic=E07 \
  --feature=F01 \
  --title="Legacy auth migration" \
  --file="docs/tasks/legacy/auth-migration.md" \
  --force
```

**Aliases for --file flag:**
- `--filepath` (hidden)
- `--filename` (hidden)

---

### `shark task list`

List tasks with optional filtering.

**Usage:**
```bash
shark task list [EPIC] [FEATURE] [--status=<status>] [--agent=<type>] [--json]
# OR (flag syntax, backward compatible)
shark task list [--epic=<epic-key>] [--feature=<feature-key>] [--status=<status>] [--agent=<type>] [--json]
```

**Filter Flags:**
- `--status <status>`: Filter by status (`todo`, `in_progress`, `ready_for_review`, `completed`, `blocked`)
- `--agent <type>`: Filter by agent type

**Examples:**

```bash
# List all tasks
shark task list

# List tasks in epic (positional)
shark task list E07

# List tasks in epic and feature (positional)
shark task list E07 F01
shark task list E07-F01  # Alternative format

# List tasks in epic and feature (flag syntax)
shark task list --epic=E07 --feature=F01

# Filter by status
shark task list --status=todo --json
shark task list --status=in_progress --json

# Filter by agent
shark task list --agent=backend --json

# Combine filters
shark task list E07 --agent=backend --status=todo --json
```

---

### `shark task get`

Get detailed information about a specific task.

**Usage:**
```bash
shark task get <task-key> [--json]
```

**Supports:**
- Numeric keys: `T-E07-F01-001`
- Slugged keys: `T-E07-F01-001-implement-jwt-validation`

**Examples:**

```bash
# Get task details
shark task get T-E07-F01-001

# Get task details (JSON)
shark task get T-E07-F01-001 --json

# Using slugged key
shark task get T-E07-F01-001-implement-jwt-validation --json
```

---

### `shark task next`

Find the next available task to work on.

**Flags:**
- `--agent <type>`: Filter by agent type
- `--epic <epic-key>`: Filter by epic
- `--json`: Output in JSON format

**Examples:**

```bash
# Get next available task (any agent)
shark task next --json

# Get next task for specific agent
shark task next --agent=backend --json
shark task next --agent=frontend --json

# Get next task in specific epic
shark task next --epic=E07 --json

# Combine filters
shark task next --epic=E07 --agent=backend --json
```

**Returns:**
- Tasks in `todo` status
- With all dependencies completed
- Sorted by priority (1 = highest)

---

### `shark task start`

Start working on a task (transition from `todo` to `in_progress`).

**Usage:**
```bash
shark task start <task-key> [--agent=<agent-id>] [--json]
```

**Examples:**

```bash
# Start task
shark task start T-E07-F01-001

# Start task with agent tracking
shark task start T-E07-F01-001 --agent="ai-agent-001" --json

# Using slugged key
shark task start T-E07-F01-001-implement-jwt-validation --json
```

---

### `shark task complete`

Mark task as ready for review (transition from `in_progress` to `ready_for_review`).

**Usage:**
```bash
shark task complete <task-key> [--notes="..."] [--json]
```

**Examples:**

```bash
# Mark task complete
shark task complete T-E07-F01-001

# Mark task complete with notes
shark task complete T-E07-F01-001 --notes="Implementation complete, all tests passing" --json
```

---

### `shark task approve`

Approve and mark task as completed (transition from `ready_for_review` to `completed`).

**Usage:**
```bash
shark task approve <task-key> [--notes="..."] [--json]
```

**Examples:**

```bash
# Approve task
shark task approve T-E07-F01-001

# Approve task with notes
shark task approve T-E07-F01-001 --notes="LGTM, approved" --json
```

---

### `shark task reopen`

Reopen task for rework (transition from `ready_for_review` to `in_progress`).

**Usage:**
```bash
shark task reopen <task-key> [--notes="..."] [--json]
```

**Examples:**

```bash
# Reopen task
shark task reopen T-E07-F01-001

# Reopen task with feedback
shark task reopen T-E07-F01-001 --notes="Need to add error handling for edge cases" --json
```

---

### `shark task block`

Block a task (transition to `blocked` status).

**Usage:**
```bash
shark task block <task-key> --reason="..." [--json]
```

**Examples:**

```bash
# Block task with reason
shark task block T-E07-F01-001 --reason="Waiting for API design approval"

# Block task with JSON output
shark task block T-E07-F01-001 --reason="Blocked by external dependency" --json
```

---

### `shark task unblock`

Unblock a task (transition from `blocked` to `todo`).

**Usage:**
```bash
shark task unblock <task-key> [--json]
```

**Examples:**

```bash
# Unblock task
shark task unblock T-E07-F01-001

# Unblock task with JSON output
shark task unblock T-E07-F01-001 --json
```

---

## Sync Commands

### `shark sync`

Synchronize markdown files with SQLite database.

**Flags:**
- `--dry-run`: Preview changes without applying them
- `--strategy <strategy>`: Conflict resolution strategy
  - `file-wins`: File system is source of truth
  - `database-wins`: Database is source of truth
  - `newer-wins`: Most recently modified wins
- `--create-missing`: Create missing epics/features from files
- `--cleanup`: Delete orphaned database records (files deleted)
- `--pattern <type>`: Sync specific pattern (`task`, `prp`)
- `--folder <path>`: Sync specific folder only
- `--json`: Output in JSON format

**Examples:**

```bash
# Preview sync changes
shark sync --dry-run --json

# Sync with file system as source of truth
shark sync --strategy=file-wins

# Sync with database as source of truth
shark sync --strategy=database-wins

# Sync with newest modification wins
shark sync --strategy=newer-wins

# Create missing epics/features
shark sync --create-missing

# Delete orphaned records
shark sync --cleanup

# Sync specific folder
shark sync --folder=docs/plan/E07-user-management-system

# Sync only task files
shark sync --pattern=task

# Sync only PRP files
shark sync --pattern=prp

# Sync both task and PRP files
shark sync --pattern=task --pattern=prp
```

**Important:**
- Status is managed exclusively in the database and is NOT synced from files
- This ensures atomic status transitions and audit trails

---

## Configuration Commands

### `shark config set`

Set a configuration value.

**Usage:**
```bash
shark config set <key> <value>
```

**Examples:**

```bash
# Set default agent type
shark config set default_agent backend

# Set default priority
shark config set default_priority 5
```

---

### `shark config get`

Get a configuration value.

**Usage:**
```bash
shark config get <key>
```

**Examples:**

```bash
# Get default agent type
shark config get default_agent

# Get default priority
shark config get default_priority
```

---

## File Path Organization

All entity creation commands (`epic create`, `feature create`, `task create`) support the `--file` flag for custom file paths.

### Default File Path Behavior

**Epics:**
- Default: `docs/plan/{epic-key}-{slug}/epic.md`
- Example: `docs/plan/E07-user-management-system/epic.md`

**Features:**
- Default: `docs/plan/{epic-key}-{epic-slug}/{feature-key}-{feature-slug}/feature.md`
- Example: `docs/plan/E07-user-management-system/E07-F01-authentication/feature.md`

**Tasks:**
- Default: `docs/plan/{epic-key}-{epic-slug}/{feature-key}-{feature-slug}/tasks/{task-key}.md`
- Example: `docs/plan/E07-user-management-system/E07-F01-authentication/tasks/T-E07-F01-001.md`

### Custom File Path Examples

```bash
# Epic with custom path
shark epic create "Q1 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# Feature with custom path
shark feature create --epic=E01 "User Growth" --file="docs/roadmap/features/growth.md"

# Task with custom path
shark task create --epic=E07 --feature=F01 "Migrate auth" --file="docs/migration/auth.md"
```

### Flag Aliases

The `--file` flag has hidden aliases for backward compatibility:

**Epic and Feature:**
- `--file` (primary, visible)
- `--filepath` (hidden alias)
- `--path` (hidden alias)

**Task:**
- `--file` (primary, visible)
- `--filepath` (hidden alias)
- `--filename` (hidden alias)

All aliases accept the same value: a complete file path including the `.md` extension.

---

## Dual Key Format Support

All `get`, `start`, `complete`, `approve`, `reopen`, `block`, and `unblock` commands support both numeric and slugged keys:

**Numeric Keys:**
- Epic: `E07`
- Feature: `E07-F01` or `F01`
- Task: `T-E07-F01-001`

**Slugged Keys:**
- Epic: `E07-user-management-system`
- Feature: `E07-F01-authentication` or `F01-authentication`
- Task: `T-E07-F01-001-implement-jwt-validation`

**Examples:**

```bash
# Using numeric keys
shark epic get E07
shark feature get E07-F01
shark task start T-E07-F01-001

# Using slugged keys (same entities)
shark epic get E07-user-management-system
shark feature get E07-F01-authentication
shark task start T-E07-F01-001-implement-jwt-validation
```

---

## JSON Output Format

All commands support `--json` flag for machine-readable output. This is required for AI agents.

**Epic JSON:**
```json
{
  "id": 7,
  "key": "E07",
  "slug": "user-management-system",
  "title": "User Management System",
  "description": "Complete user management infrastructure",
  "file_path": "docs/plan/E07-user-management-system/epic.md",
  "priority": 1,
  "business_value": 10,
  "progress": 42.5,
  "created_at": "2026-01-02T10:00:00Z",
  "updated_at": "2026-01-02T15:30:00Z"
}
```

**Feature JSON:**
```json
{
  "id": 1,
  "key": "E07-F01",
  "slug": "authentication",
  "epic_id": 7,
  "epic_key": "E07",
  "title": "Authentication",
  "description": "User authentication system",
  "file_path": "docs/plan/E07-user-management-system/E07-F01-authentication/feature.md",
  "execution_order": 1,
  "progress": 60.0,
  "task_count": 5,
  "created_at": "2026-01-02T10:00:00Z",
  "updated_at": "2026-01-02T15:30:00Z"
}
```

**Task JSON:**
```json
{
  "id": 1,
  "key": "T-E07-F01-001",
  "slug": "implement-jwt-validation",
  "feature_id": 1,
  "epic_id": 7,
  "title": "Implement JWT validation",
  "description": "Add JWT token validation middleware",
  "status": "in_progress",
  "priority": 3,
  "agent_type": "backend",
  "depends_on": ["T-E07-F01-000"],
  "dependency_status": {
    "T-E07-F01-000": "completed"
  },
  "file_path": "docs/plan/E07-user-management-system/E07-F01-authentication/tasks/T-E07-F01-001.md",
  "created_at": "2026-01-02T10:00:00Z",
  "updated_at": "2026-01-02T15:30:00Z"
}
```

---

## Exit Codes

Shark CLI uses standard exit codes:

- `0`: Success
- `1`: Not found (entity does not exist)
- `2`: Database error
- `3`: Invalid state (e.g., invalid status transition)

**Example usage in scripts:**

```bash
#!/bin/bash

shark task start T-E07-F01-001
if [ $? -eq 0 ]; then
  echo "Task started successfully"
else
  echo "Failed to start task"
  exit 1
fi
```

---

## AI Agent Best Practices

1. **Always use `--json` flag** for machine-readable output
2. **Check dependencies** before starting tasks via `shark task next --json`
3. **Use atomic operations** - each command is a single transaction
4. **Handle blocked tasks** - use `block` command with reasons
5. **Sync after Git operations** - run `shark sync` after pulls/checkouts
6. **Track work with agent identifier** - use `--agent` flag for audit trail
7. **Use priority effectively** - 1=highest, 10=lowest for task ordering
8. **Check exit codes** - Non-zero indicates errors

---

## Related Documentation

- [CLAUDE.md](../CLAUDE.md) - Development guidelines and project overview
- [README.md](../README.md) - Project introduction and quick start
- [MIGRATION_CUSTOM_PATHS.md](MIGRATION_CUSTOM_PATHS.md) - Migration guide for path changes
