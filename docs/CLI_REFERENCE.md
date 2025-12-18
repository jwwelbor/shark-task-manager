# Shark CLI Reference

Complete command reference for the Shark task management CLI. Designed for AI agents and developers managing hierarchical work (Epics → Features → Tasks).

## Overview

Shark is a task management CLI for AI-driven development workflows.

**Hierarchy:** Epics → Features → Tasks

**Storage:** SQLite database with markdown file synchronization

**Design principles:**
- JSON output for machine parsing (`--json` flag)
- Atomic operations (each command is single transaction)
- Dependency-aware task selection
- Bidirectional file-database sync

**Key concepts:**
- Auto-generated keys: E01, E01-F01, T-E01-F01-001
- Task statuses: `todo`, `in_progress`, `blocked`, `ready_for_review`, `completed`, `archived`
- Epic/Feature statuses: `draft`, `active`, `completed`, `archived`
- Priority ordering: 1-10 (1 = highest priority)
- Task dependencies: tasks can depend on other tasks
- Status transitions: enforced state machine (e.g., todo → in_progress → ready_for_review → completed)

## Installation & Setup

### `shark init`

Initialize Shark CLI infrastructure. Creates database, folders, config, and templates.

**Flags:**
- `--non-interactive` - Skip prompts, use defaults (for automation)
- `--force` - Overwrite existing config and templates

**Creates:**
- SQLite database: `shark-tasks.db`
- Folder structure: `docs/plan/`
- Config file: `.sharkconfig.json`
- Templates: `shark-templates/` (epic.md, feature.md, task.md)

**Examples:**

```bash
# Interactive initialization
shark init

# Non-interactive (for CI/automation)
shark init --non-interactive

# Force overwrite existing files
shark init --force
```

## Complete Workflow Example

End-to-end workflow: initialize → create epic → create feature → create task → work task through lifecycle.

```bash
# Initialize (first time only)
shark init --non-interactive

# Create epic
shark epic create "New Project"
#  SUCCESS  Epic created successfully!
#
# Epic Key:  E01-new-project
# Directory: docs/plan/E01-new-project
# File:      docs/plan/E01-new-project/epic.md
# Database:  ✓ Epic record created (ID: 1)
#
# Next steps:
# 1. Edit the epic.md file to add details
# 2. Create features with: shark feature create --epic=E01 "Feature title"

# Create feature in epic
shark feature create --epic=E01 "Core Functionality"
#  SUCCESS  Feature created successfully!
#
# Feature Key: E01-F01-core-functionality
# Epic:        E01
# Directory:   docs/plan/E01-new-project/E01-F01-core-functionality
# File:        docs/plan/E01-new-project/E01-F01-core-functionality/prd.md
#
# Next steps:
# 1. Edit the prd.md file to add details
# 2. Create tasks with: shark task create --epic=E01 --feature=F01 --title="Task title" --agent=backend

# Create task in feature
shark task create \
  --epic=E01 \
  --feature=F01 \
  --title="Implement validation logic" \
  --agent=backend \
  --priority=1
#  SUCCESS  Created task T-E01-F01-001: Implement validation logic
# File created at: docs/tasks/todo/T-E01-F01-001.md
# Start work with: pm task start T-E01-F01-001

# Discover next available task (JSON output)
shark task next --agent=backend --json
# {
#   "agent_type": "backend",
#   "dependencies": null,
#   "dependency_status": {},
#   "file_path": "docs/tasks/todo/T-E01-F01-001.md",
#   "key": "T-E01-F01-001",
#   "priority": 1,
#   "title": "Implement validation logic"
# }

# Start task (todo → in_progress)
shark task start T-E01-F01-001 --agent="ai-agent-001"
#  SUCCESS  Task T-E01-F01-001 started. Status changed to in_progress.

# ... do implementation work ...

# Mark ready for review (in_progress → ready_for_review)
shark task complete T-E01-F01-001 --agent="ai-agent-001" --notes="Implementation complete"
#  SUCCESS  Task T-E01-F01-001 marked ready for review. Status changed to ready_for_review.

# Approve and complete (ready_for_review → completed)
shark task approve T-E01-F01-001 --agent="reviewer-001" --notes="LGTM"
#  SUCCESS  Task T-E01-F01-001 approved and completed.

# Sync database with filesystem
shark sync
#  SUCCESS  Sync completed:
#   Files scanned:      33
#   New tasks imported: 0
#   Tasks updated:      0
#   Conflicts resolved: 0
#   Warnings:           0
#   Errors:             0
```

## Epic Commands

### `shark epic create <title>`

Create a new epic with auto-assigned key, folder, and database entry.

**Flags:**
- `--description <text>` - Epic description (optional)

**Examples:**

```bash
# Basic epic creation
shark epic create "User Authentication System"

# With description
shark epic create "User Auth" --description="Add OAuth and MFA"
```

### `shark epic list`

List all epics with progress information.

**Flags:**
- `--status <status>` - Filter by: `draft`, `active`, `completed`, `archived`
- `--sort-by <field>` - Sort by: `key`, `progress`, `status` (default: `key`)

**Examples:**

```bash
# List all epics
shark epic list

# Filter by status
shark epic list --status=active

# Sort by progress
shark epic list --sort-by=progress

# JSON output
shark epic list --json
```

### `shark epic get <epic-key>`

Get detailed epic information with all features and progress.

**Examples:**

```bash
# Human-readable output
shark epic get E04

# JSON output
shark epic get E04 --json
```

### `shark epic delete <epic-key>`

Delete an epic (CASCADE deletes all features and tasks).

**Flags:**
- `--force` - Force deletion even if epic has features

**Examples:**

```bash
# Delete epic with no features
shark epic delete E05

# Force delete epic with features/tasks
shark epic delete E05 --force
```

**Warning:** Cannot be undone. All features and tasks are deleted.

## Feature Commands

### `shark feature create --epic=<key> <title>`

Create a new feature with auto-assigned key, folder, and database entry.

**Flags:**
- `--epic <key>` - Epic key (required, e.g., `E01`)
- `--description <text>` - Feature description (optional)

**Examples:**

```bash
# Basic feature creation
shark feature create --epic=E01 "OAuth Login Integration"

# With description
shark feature create --epic=E01 "OAuth Login" --description="Add OAuth 2.0 support"
```

### `shark feature list`

List features with optional filtering.

**Flags:**
- `-e, --epic <key>` - Filter by epic key
- `--status <status>` - Filter by: `draft`, `active`, `completed`, `archived`
- `--sort-by <field>` - Sort by: `key`, `progress`, `status` (default: `key`)

**Examples:**

```bash
# List all features
shark feature list

# Features in specific epic
shark feature list --epic=E04

# Filter by status
shark feature list --status=active

# Sort by progress
shark feature list --sort-by=progress

# JSON output
shark feature list --json
```

### `shark feature get <feature-key>`

Get detailed feature information with all tasks and progress.

**Examples:**

```bash
# Human-readable output
shark feature get E04-F02

# JSON output
shark feature get E04-F02 --json
```

### `shark feature delete <feature-key>`

Delete a feature (CASCADE deletes all tasks).

**Flags:**
- `--force` - Force deletion even if feature has tasks

**Examples:**

```bash
# Delete feature with no tasks
shark feature delete E04-F02

# Force delete feature with tasks
shark feature delete E04-F02 --force
```

**Warning:** Cannot be undone. All tasks are deleted.

## Task Commands

### `shark task create`

Create a new task with auto-generated key and file.

**Flags (required):**
- `-e, --epic <key>` - Epic key (e.g., `E01`)
- `-f, --feature <key>` - Feature key (e.g., `F02` or `E01-F02`)
- `-t, --title <text>` - Task title
- `-a, --agent <type>` - Agent type: `frontend`, `backend`, `api`, `testing`, `devops`, `general`

**Flags (optional):**
- `-d, --description <text>` - Detailed description
- `-p, --priority <n>` - Priority 1-10 (default: 5, where 1 = highest)
- `--depends-on <keys>` - Comma-separated dependency task keys

**Examples:**

```bash
# Basic task
shark task create --epic=E01 --feature=F02 --title="Build login form" --agent=frontend

# With all options
shark task create \
  -e E01 \
  -f F02 \
  -t "User authentication service" \
  -a backend \
  -p 3 \
  -d "Implement JWT-based auth" \
  --depends-on="T-E01-F01-001,T-E01-F01-002"
```

### `shark task list`

List tasks with optional filtering.

**Flags:**
- `-s, --status <status>` - Filter by: `todo`, `in_progress`, `completed`, `blocked`, `ready_for_review`, `archived`
- `-e, --epic <key>` - Filter by epic key
- `-f, --feature <key>` - Filter by feature key
- `-a, --agent <type>` - Filter by agent type
- `--priority-min <n>` - Minimum priority (1-10)
- `--priority-max <n>` - Maximum priority (1-10)
- `-b, --blocked` - Show only blocked tasks

**Examples:**

```bash
# List all tasks
shark task list

# Filter by status
shark task list --status=todo

# Filter by epic
shark task list --epic=E04

# Filter by agent
shark task list --agent=backend

# High priority tasks (1-3)
shark task list --priority-min=1 --priority-max=3

# Only blocked tasks
shark task list --blocked

# JSON output
shark task list --json
```

### `shark task get <task-key>`

Get detailed task information including dependencies.

**Examples:**

```bash
# Human-readable output
shark task get T-E01-F01-001

# JSON output with dependency status
shark task get T-E01-F01-001 --json
```

### `shark task next`

Find next available task (dependency-aware, priority-sorted).

**Selection criteria:**
- Status = `todo`
- All dependencies completed or archived
- Sorted by priority (1 = highest)

**Flags:**
- `-a, --agent <type>` - Filter by agent type
- `-e, --epic <key>` - Filter by epic key

**Examples:**

```bash
# Next task (any agent)
shark task next

# Next backend task
shark task next --agent=backend

# Next task in specific epic
shark task next --epic=E04

# JSON output
shark task next --json
```

### `shark task delete <task-key>`

Delete a task (CASCADE deletes history).

**Examples:**

```bash
# Delete task
shark task delete T-E04-F01-001
```

**Warning:** Cannot be undone. Task history is also deleted.

## Task Lifecycle

Task status flow:

```
todo → in_progress → ready_for_review → completed
  ↓         ↓
blocked    (reopen)
```

### `shark task start <task-key>`

Start working on a task (todo → in_progress).

**Validations:**
- Current status must be `todo`
- Warns if dependencies incomplete

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)

**Examples:**

```bash
# Start task
shark task start T-E04-F06-001

# With agent identifier
shark task start T-E04-F06-001 --agent="ai-agent-001"
```

### `shark task complete <task-key>`

Mark task ready for review (in_progress → ready_for_review).

**Validations:**
- Current status must be `in_progress`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Completion notes

**Examples:**

```bash
# Mark ready for review
shark task complete T-E04-F06-001

# With notes
shark task complete T-E04-F06-001 --notes="All tests passing"
```

### `shark task approve <task-key>`

Approve and complete task (ready_for_review → completed).

**Validations:**
- Current status must be `ready_for_review`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Approval notes

**Examples:**

```bash
# Approve task
shark task approve T-E04-F06-001

# With approval notes
shark task approve T-E04-F06-001 --agent="reviewer-001" --notes="LGTM"
```

### `shark task reopen <task-key>`

Reopen for rework (ready_for_review → in_progress).

**Validations:**
- Current status must be `ready_for_review`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Rework notes

**Examples:**

```bash
# Reopen task
shark task reopen T-E04-F06-001

# With rework reason
shark task reopen T-E04-F06-001 --notes="Need to add error handling"
```

### `shark task block <task-key>`

Block a task (todo/in_progress → blocked).

**Validations:**
- Current status must be `todo` or `in_progress`

**Flags:**
- `-r, --reason <text>` - Reason for blocking (REQUIRED)
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)

**Examples:**

```bash
# Block task with reason
shark task block T-E04-F06-001 --reason="Waiting for API design approval"

# Short form
shark task block T-E04-F06-001 -r "Missing credentials"
```

### `shark task unblock <task-key>`

Unblock a task (blocked → todo).

**Validations:**
- Current status must be `blocked`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)

**Examples:**

```bash
# Unblock task
shark task unblock T-E04-F06-001
```

## Sync Commands

### `shark sync`

Synchronize task files with database bidirectionally.

**Important:** Status is managed EXCLUSIVELY in the database and is NOT synced from files.

**Flags:**
- `--folder <path>` - Sync specific folder (default: `docs/plan`)
- `--dry-run` - Preview changes without applying
- `--strategy <strategy>` - Conflict resolution: `file-wins`, `database-wins`, `newer-wins` (default: `file-wins`)
- `--create-missing` - Auto-create missing epics/features from files
- `--cleanup` - Delete orphaned database tasks (files deleted)
- `--pattern <type>` - File patterns to scan: `task`, `prp` (can specify multiple, default: `task`)

**Examples:**

```bash
# Sync all (task files only)
shark sync

# Preview changes
shark sync --dry-run

# Sync PRP files only
shark sync --pattern=prp

# Sync both task and PRP files
shark sync --pattern=task --pattern=prp

# Sync specific folder
shark sync --folder=docs/plan/E04-task-mgmt-cli-core

# Database overrides files
shark sync --strategy=database-wins

# Create missing epics/features
shark sync --create-missing

# Delete orphaned tasks
shark sync --cleanup

# JSON output
shark sync --json
```

**Use cases:**
- After `git pull` or `git checkout`
- After manual file edits
- To clean up after deleting files
- To import tasks from file system

## Global Flags

Available on all commands:

- `--json` - Output in JSON format (machine-readable)
- `--no-color` - Disable colored output
- `-v, --verbose` - Enable verbose/debug output
- `--config <path>` - Config file path (default: `.sharkconfig.json`)
- `--db <path>` - Database file path (default: `shark-tasks.db`)

**Examples:**

```bash
# JSON output (for automation/parsing)
shark task list --json
shark epic get E04 --json

# Disable colors (for logging/CI)
shark task list --no-color

# Verbose output (debugging)
shark sync --verbose

# Custom database location
shark task list --db=/path/to/custom.db

# Custom config location
shark init --config=/path/to/custom-config.json
```
