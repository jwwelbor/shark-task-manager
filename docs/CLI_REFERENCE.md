# Shark CLI Reference

Complete command reference for the Shark Task Manager CLI.

## Table of Contents

- [Global Flags](#global-flags)
- [Key Format Improvements](#key-format-improvements)
- [Initialization](#initialization)
- [Epic Commands](#epic-commands)
- [Feature Commands](#feature-commands)
- [Task Commands](#task-commands)
- [Sync Commands](#sync-commands)
- [Configuration Commands](#configuration-commands)
- [Error Messages](#error-messages)

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

## Key Format Improvements

Shark CLI now supports flexible key formats for improved usability.

### Case Insensitive Keys

All entity keys are case insensitive. You can use any combination of uppercase and lowercase:

**Epics:**
```bash
shark epic get E07       # Standard
shark epic get e07       # Lowercase
shark epic get E07-user-management-system
shark epic get e07-user-management-system
```

**Features:**
```bash
shark feature get E07-F01        # Standard
shark feature get e07-f01        # Lowercase
shark feature get E07-f01        # Mixed case
shark feature get F01            # Short format
shark feature get f01            # Short format (lowercase)
```

**Tasks:**
```bash
shark task start E07-F20-001     # Short format
shark task start e07-f20-001     # Lowercase
shark task start T-E07-F20-001   # Traditional format
shark task start t-e07-f20-001   # Traditional lowercase
```

### Short Task Key Format

Task keys can now be referenced without the `T-` prefix:

**Traditional Format:**
```bash
shark task get T-E07-F20-001
shark task start T-E07-F20-001
shark task complete T-E07-F20-001
```

**Short Format (Recommended):**
```bash
shark task get E07-F20-001
shark task start E07-F20-001
shark task complete E07-F20-001
```

Both formats work identically. The CLI automatically normalizes keys internally.

### Positional Arguments

Feature and task creation commands now support cleaner positional argument syntax:

**Feature Creation:**
```bash
# New positional syntax (recommended)
shark feature create E07 "Feature Title"
shark feature create e07 "Feature Title"  # Case insensitive

# Traditional flag syntax (still supported)
shark feature create --epic=E07 --title="Feature Title"
```

**Task Creation:**
```bash
# New positional syntax - 3 arguments (epic, feature, title)
shark task create E07 F20 "Task Title"
shark task create e07 f20 "Task Title"  # Case insensitive

# New positional syntax - 2 arguments (combined epic-feature, title)
shark task create E07-F20 "Task Title"
shark task create e07-f20 "Task Title"  # Case insensitive

# Traditional flag syntax (still supported)
shark task create --epic=E07 --feature=F20 --title="Task Title"
```

### Syntax Compatibility

**All legacy syntax remains fully supported.** The new formats are additive improvements:

- ✅ Old commands continue to work unchanged
- ✅ Scripts don't need updates
- ✅ Mix and match syntaxes as preferred
- ✅ Case insensitivity works with all formats
- ✅ Backward compatibility guaranteed

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

**Positional Syntax (Recommended):**
```bash
shark feature create <epic-key> "<title>" [flags]
```

**Flag Syntax (Legacy, still supported):**
```bash
shark feature create --epic=<epic-key> --title="<title>" [flags]
```

**Optional Flags:**
- `--file <path>`: Custom file path (relative to root, must include .md)
- `--force`: Reassign file if already claimed by another feature or epic
- `--execution-order <number>`: Execution order within epic
- `--json`: Output in JSON format

**Examples:**

```bash
# Create feature with positional syntax (recommended)
shark feature create E07 "Authentication"
shark feature create e07 "Authentication"  # Case insensitive
# Creates: docs/plan/E07-user-management-system/E07-F01-authentication/feature.md

# Create feature with flag syntax (legacy)
shark feature create --epic=E07 --title="Authentication"

# Create feature with custom file path
shark feature create E07 "User Profiles" --file="docs/features/profiles/feature.md"

# Create feature with execution order
shark feature create E07 "Authorization" --execution-order=2 --json

# Force reassign file
shark feature create E07 "Legacy Auth" --file="docs/legacy/auth.md" --force
```

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

**Features:**
- **Workflow-aware status display**: Task statuses are colored according to workflow config
- **Phase information**: Status breakdown includes workflow phase (planning, development, review, etc.)
- **Completion message**: Shows "All tasks completed!" when progress reaches 100%

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

**Output includes:**
- Feature metadata (title, status, progress, path)
- Task status breakdown (status, count, phase) - ordered by workflow phase
- Task list with colored statuses
- Completion message if all tasks are done

**JSON Output:**
```json
{
  "id": 1,
  "epic_id": 7,
  "key": "E07-F01",
  "title": "Authentication",
  "status": "active",
  "progress_pct": 75.0,
  "tasks": [...],
  "status_breakdown": [
    {"status": "completed", "count": 3, "phase": "done", "color": "green"},
    {"status": "in_progress", "count": 1, "phase": "development", "color": "blue"}
  ]
}
```

---

## Task Commands

### `shark task create`

Create a new task within a feature.

**Positional Syntax (Recommended):**
```bash
# 3-argument format: epic, feature, title
shark task create <epic-key> <feature-key> "<title>" [flags]

# 2-argument format: combined epic-feature, title
shark task create <epic-feature-key> "<title>" [flags]
```

**Flag Syntax (Legacy, still supported):**
```bash
shark task create --epic=<epic-key> --feature=<feature-key> --title="<title>" [flags]
```

**Optional Flags:**
- `--agent <type>`: Agent type (`frontend`, `backend`, `api`, `testing`, `devops`, `general`)
- `--priority <1-10>`: Priority (1 = highest, 10 = lowest, default: 5)
- `--description <string>`: Detailed description
- `--depends-on <task-keys>`: Comma-separated list of dependency task keys
- `--file <path>`: Custom file path (relative to root, must include .md)
- `--force`: Reassign file if already claimed by another task
- `--json`: Output in JSON format

**Examples:**

```bash
# Create task with positional syntax - 3 arguments (recommended)
shark task create E07 F01 "Implement JWT validation"
shark task create e07 f01 "Implement JWT validation"  # Case insensitive

# Create task with positional syntax - 2 arguments
shark task create E07-F01 "Implement JWT validation"
shark task create e07-f01 "Implement JWT validation"  # Case insensitive

# Create task with flag syntax (legacy)
shark task create --epic=E07 --feature=F01 --title="Implement JWT validation"

# Create task with agent and priority
shark task create E07 F01 "Implement JWT validation" --agent=backend --priority=3

# Create task with dependencies
shark task create E07 F01 "Add token refresh" \
  --agent=backend \
  --depends-on="E07-F01-001,E07-F01-002"

# Create task with custom file path
shark task create E07 F01 "Legacy auth migration" \
  --file="docs/tasks/legacy/auth-migration.md" \
  --force
```

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
- Short format: `E07-F01-001` (recommended)
- Traditional format: `T-E07-F01-001`
- Slugged keys: `E07-F01-001-implement-jwt-validation`, `T-E07-F01-001-implement-jwt-validation`
- Case insensitive: `e07-f01-001`, `t-e07-f01-001`

**Examples:**

```bash
# Get task details (short format, recommended)
shark task get E07-F01-001
shark task get e07-f01-001  # Case insensitive

# Get task details (traditional format)
shark task get T-E07-F01-001

# Get task details (JSON)
shark task get E07-F01-001 --json

# Using slugged key
shark task get E07-F01-001-implement-jwt-validation --json
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
# Start task (short format, recommended)
shark task start E07-F01-001
shark task start e07-f01-001  # Case insensitive

# Start task (traditional format)
shark task start T-E07-F01-001

# Start task with agent tracking
shark task start E07-F01-001 --agent="ai-agent-001" --json

# Using slugged key
shark task start E07-F01-001-implement-jwt-validation --json
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
# Mark task complete (short format, recommended)
shark task complete E07-F01-001
shark task complete e07-f01-001  # Case insensitive

# Mark task complete with notes
shark task complete E07-F01-001 --notes="Implementation complete, all tests passing" --json
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
# Approve task (short format, recommended)
shark task approve E07-F01-001
shark task approve e07-f01-001  # Case insensitive

# Approve task with notes
shark task approve E07-F01-001 --notes="LGTM, approved" --json
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
# Reopen task (short format, recommended)
shark task reopen E07-F01-001
shark task reopen e07-f01-001  # Case insensitive

# Reopen task with feedback
shark task reopen E07-F01-001 --notes="Need to add error handling for edge cases" --json
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
# Block task with reason (short format, recommended)
shark task block E07-F01-001 --reason="Waiting for API design approval"
shark task block e07-f01-001 --reason="Waiting for API design approval"  # Case insensitive

# Block task with JSON output
shark task block E07-F01-001 --reason="Blocked by external dependency" --json
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
# Unblock task (short format, recommended)
shark task unblock E07-F01-001
shark task unblock e07-f01-001  # Case insensitive

# Unblock task with JSON output
shark task unblock E07-F01-001 --json
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

## Workflow Configuration

Shark supports customizable workflow configuration through `.sharkconfig.json`. This allows you to define custom status flows, colors, phases, and agent types.

### Configuration Structure

```json
{
  "status_flow": {
    "draft": ["ready_for_refinement", "cancelled"],
    "ready_for_refinement": ["in_refinement", "cancelled"],
    "in_refinement": ["ready_for_development", "draft"],
    "ready_for_development": ["in_development", "cancelled"],
    "in_development": ["ready_for_code_review", "blocked"],
    "ready_for_code_review": ["in_code_review", "in_development"],
    "in_code_review": ["ready_for_qa", "in_development"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["ready_for_approval", "in_development"],
    "ready_for_approval": ["in_approval"],
    "in_approval": ["completed", "ready_for_qa"],
    "completed": [],
    "blocked": ["ready_for_development"],
    "cancelled": []
  },
  "status_metadata": {
    "draft": {
      "color": "gray",
      "description": "Task created but not yet refined",
      "phase": "planning"
    },
    "in_development": {
      "color": "yellow",
      "description": "Code implementation in progress",
      "phase": "development",
      "agent_types": ["developer", "ai-coder"]
    },
    "completed": {
      "color": "green",
      "description": "Task finished and approved",
      "phase": "done"
    }
  },
  "special_statuses": {
    "_start_": ["draft", "ready_for_development"],
    "_complete_": ["completed", "cancelled"]
  }
}
```

### Configuration Options

**status_flow**: Defines valid transitions between statuses
- Key: Source status
- Value: Array of valid target statuses

**status_metadata**: Metadata for each status
- `color`: ANSI color name (red, green, yellow, blue, cyan, magenta, gray, white, orange, purple)
- `description`: Human-readable description
- `phase`: Workflow phase (planning, development, review, qa, approval, done, any)
- `agent_types`: Array of agent types that can work on tasks in this status

**special_statuses**: Special status markers
- `_start_`: Valid initial statuses for new tasks
- `_complete_`: Terminal statuses (no transitions out)

### Workflow Phases

Phases are used to order status displays:

1. **planning**: Draft, refinement stages (gray, cyan colors)
2. **development**: Active implementation (yellow colors)
3. **review**: Code review stages (magenta colors)
4. **qa**: Quality assurance (green colors)
5. **approval**: Final approval stages (purple colors)
6. **done**: Terminal states (white/green colors)
7. **any**: Status applicable to any phase (blocked, on_hold)

### Feature Get Display

The `shark feature get` command shows workflow-aware status information:
- Status breakdown ordered by workflow phase
- Statuses colored according to `status_metadata` colors
- Phase column shows which workflow stage each status belongs to
- "All tasks completed!" message when progress reaches 100%

### Example: Simple Workflow

For a simpler workflow with fewer statuses:

```json
{
  "status_flow": {
    "todo": ["in_progress"],
    "in_progress": ["review", "blocked"],
    "review": ["done", "in_progress"],
    "blocked": ["in_progress"],
    "done": []
  },
  "status_metadata": {
    "todo": {"color": "gray", "phase": "planning"},
    "in_progress": {"color": "yellow", "phase": "development"},
    "review": {"color": "magenta", "phase": "review"},
    "blocked": {"color": "red", "phase": "any"},
    "done": {"color": "green", "phase": "done"}
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["done"]
  }
}
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

## Error Messages

Shark CLI provides user-friendly error messages with context and examples to help you resolve issues quickly.

### Enhanced Error Format

When an error occurs, you'll see:
1. **Clear description** of what went wrong
2. **Context** about why it happened
3. **Example** showing the correct syntax
4. **Suggestions** for resolution

### Common Errors and Solutions

#### Invalid Epic Key Format

**Error:**
```
Error: invalid epic key format: "invalid"

Epic keys must follow format: E{number} or E{number}-{slug}

Valid examples:
  - E07
  - e07 (case insensitive)
  - E07-user-management
  - e07-user-management (case insensitive)
```

**Solution:** Use the correct epic key format with `E` prefix followed by a number.

---

#### Invalid Feature Key Format

**Error:**
```
Error: invalid feature key format: "invalid"

Feature keys must follow one of these formats:
  - E{epic}-F{feature} (full format)
  - F{feature} (short format)
  - With optional slug suffix

Valid examples:
  - E07-F01, e07-f01 (case insensitive)
  - F01, f01 (case insensitive)
  - E07-F01-authentication
  - F01-authentication
```

**Solution:** Use the correct feature key format.

---

#### Invalid Task Key Format

**Error:**
```
Error: invalid task key format: "invalid"

Task keys must follow one of these formats:
  - E{epic}-F{feature}-{number} (short format, recommended)
  - T-E{epic}-F{feature}-{number} (traditional format)
  - With optional slug suffix

Valid examples:
  - E07-F20-001, e07-f20-001 (case insensitive)
  - T-E07-F20-001, t-e07-f20-001
  - E07-F20-001-implement-jwt
  - T-E07-F20-001-implement-jwt
```

**Solution:** Use the correct task key format. The `T-` prefix is optional.

---

#### Task Not Found

**Error:**
```
Error: task not found: "E07-F20-999"

The task key was not found in the database.

Possible solutions:
  - Check the task key spelling
  - List tasks: shark task list E07 F20
  - Verify epic and feature exist
```

**Solution:** Verify the task exists using `shark task list` or check for typos.

---

#### Invalid Status Transition

**Error:**
```
Error: cannot transition from 'completed' to 'in_progress'

Valid transitions from 'completed':
  - No valid transitions (task is completed)

Task lifecycle:
  todo → in_progress → ready_for_review → completed
           ↓              ↓
        blocked ←────────┘
```

**Solution:** Follow the valid task lifecycle transitions. Use `shark task reopen` to return a task from review to in-progress.

---

#### Missing Required Arguments

**Error:**
```
Error: missing required arguments

Usage: shark task create <epic-key> <feature-key> "<title>" [flags]
   OR: shark task create <epic-feature-key> "<title>" [flags]
   OR: shark task create --epic=<key> --feature=<key> --title="<title>" [flags]

Examples:
  shark task create E07 F20 "Task Title"
  shark task create E07-F20 "Task Title"
  shark task create --epic=E07 --feature=F20 --title="Task Title"
```

**Solution:** Provide all required arguments in one of the supported syntaxes.

---

### Interpreting Error Messages

All error messages follow this structure:

```
Error: <brief description>

<detailed explanation>

<valid examples or solutions>
```

**Tips:**
- Read the entire error message for context
- Check the examples provided
- Verify your syntax matches one of the valid formats
- Use case insensitive keys (e07 works same as E07)
- Try the short format (E07-F20-001 instead of T-E07-F20-001)

---

## Related Documentation

- [CLAUDE.md](../CLAUDE.md) - Development guidelines and project overview
- [README.md](../README.md) - Project introduction and quick start
- [MIGRATION_F20.md](MIGRATION_F20.md) - Migration guide for CLI improvements (E07-F20)
- [MIGRATION_CUSTOM_PATHS.md](MIGRATION_CUSTOM_PATHS.md) - Migration guide for path changes
