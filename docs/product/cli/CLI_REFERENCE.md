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
# 2. Create tasks with: shark task create "Task title" --epic=E01 --feature=F01 --agent=backend

# Create task in feature
shark task create "Implement validation logic" \
  --epic=E01 \
  --feature=F01 \
  --agent=backend \
  --priority=1
#  SUCCESS  Created task T-E01-F01-001: Implement validation logic
# File created at: docs/tasks/todo/T-E01-F01-001.md
# Start work with: shark task start T-E01-F01-001

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

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--path` | string | (empty) | Custom folder base path for organizing epic and child features. Relative to project root. Example: `docs/roadmap/2025-q1` |
| `--filename` | string | (empty) | Custom file path relative to project root. Must end in `.md`. Example: `docs/roadmap/2025.md`. **Note:** Takes precedence over `--path` |
| `--force` | bool | false | Force reassignment if file path already claimed by another epic or feature |
| `--description` | string | (empty) | Epic description |
| `--priority` | string | medium | Priority: `high`, `medium`, or `low` |
| `--business-value` | string | (empty) | Business value: `high`, `medium`, or `low` |

**Custom Filename:**

By default, epics are created in `docs/plan/{epic-key}/` with filename `epic.md`.

Use `--filename` to specify a custom path:

```bash
shark epic create "Platform Roadmap" --filename="docs/roadmap/2025.md"
```

**Rules:**
- Path must be relative to project root
- Must include `.md` extension
- Existing files are automatically associated (not overwritten)
- Use `--force` to reassign files from other epics

**Examples:**

```bash
# Default location (backward compatible)
shark epic create "User Authentication System"
# Created epic E01 at docs/plan/E01-user-authentication-system/epic.md

# With description
shark epic create "User Auth" --description="Add OAuth and MFA"

# Custom location
shark epic create "Platform Roadmap" --filename="docs/roadmap/2025-platform.md"
# Created epic E02 at docs/roadmap/2025-platform.md

# Associate existing file
shark epic create "Q1 Roadmap" --filename="docs/shared-roadmap.md"
# Created epic E03 at docs/shared-roadmap.md

# Force reassignment from another epic
shark epic create "SSO Integration" --filename="docs/roadmap/2025-platform.md" --force
# Created epic E04 at docs/roadmap/2025-platform.md (reassigned from epic E02)
```

**Custom Folder Path (`--path`):**

Use `--path` to define a base folder where epic files and child features will be organized:

```bash
shark epic create "Q1 2025 Roadmap" --path="docs/roadmap/2025-q1"
```

This allows flexible organization of your project structure:

```
docs/
├── roadmap/                    # Custom folder base path
│   ├── 2025-q1/                # Epic's custom path
│   │   ├── epic.md             # Epic file
│   │   ├── user-growth/        # Feature inherits parent path
│   │   │   └── feature.md
│   │   └── retention/
│   │       └── feature.md
│   └── 2025-q2/
│       └── epic.md
└── plan/                       # Default location (backward compatible)
    └── E03-other-epic/
        └── epic.md
```

**Path Resolution Order:**

1. `--filename` - Explicit file path (highest priority)
2. `--path` - Custom folder base path
3. Default: `docs/plan/{epic-key}/`

**Inheritance Rules:**

- Features created under an epic inherit the epic's custom folder path
- Features can override the inherited path with their own `--path`
- Tasks inherit the feature's path, which inherits from the epic
- Path override only applies to direct path, not concatenated

**Path Normalization:**

- Paths are stored relative to project root
- Trailing slashes are normalized
- Empty strings are treated as NULL (use default)
- Paths must not contain `..` (path traversal protection)

**Examples:**

```bash
# Organize by quarter/year
shark epic create "Q1 2025 Initiative" --path="docs/roadmap/2025-q1"

# Feature inherits epic path
shark feature create --epic=E01 "User Growth"
# Stored in docs/roadmap/2025-q1/ directory

# Feature overrides epic path
shark feature create --epic=E01 "Legacy API" --path="docs/legacy-api"
# Stored in docs/legacy-api/ directory

# Mix organization styles in same project
shark epic create "Core Product" --path="docs/core"
shark epic create "Platform Services"  # Uses default: docs/plan/E02-platform-services/
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

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | (required) | Epic key (e.g., `E01`) |
| `--path` | string | (empty) | Custom folder base path for this feature (inherited from epic if not specified). Relative to project root. Example: `docs/features/auth` |
| `--filename` | string | (empty) | Custom file path relative to project root. Must end in `.md`. Example: `docs/specs/auth.md`. **Note:** Takes precedence over `--path` |
| `--force` | bool | false | Force reassignment if file path already claimed by another feature or epic |
| `--description` | string | (empty) | Feature description |
| `--execution-order` | int | 0 | Execution order (0 = not set) |

**Custom Filename:**

By default, features are created in `docs/plan/{epic-key}/{feature-key}/` with filename `feature.md`.

Use `--filename` to specify a custom path:

```bash
shark feature create --epic=E01 --filename="docs/specs/auth.md" "OAuth Login"
```

**Rules:**
- Path must be relative to project root
- Must include `.md` extension
- Existing files are automatically associated (not overwritten)
- Use `--force` to reassign files from other features

**Examples:**

```bash
# Default location (backward compatible)
shark feature create --epic=E01 "OAuth Login Integration"
# Feature Key: E01-F01-oauth-login-integration
# File: docs/plan/E01-user-authentication-system/E01-F01-oauth-login-integration/feature.md

# With description
shark feature create --epic=E01 "OAuth Login" --description="Add OAuth 2.0 support"

# Custom location
shark feature create --epic=E01 --filename="docs/specs/auth.md" "Security Hardening"
# Feature Key: E01-F02-security-hardening
# File: docs/specs/auth.md

# Associate existing file
shark feature create --epic=E01 --filename="docs/shared/integration.md" "Payment Integration"
# Feature Key: E01-F03-payment-integration
# File: docs/shared/integration.md

# Force reassignment from another feature
shark feature create --epic=E01 --filename="docs/specs/auth.md" --force "MFA Implementation"
# Feature Key: E01-F04-mfa-implementation
# File: docs/specs/auth.md (reassigned from E01-F02)
```

**Custom Folder Path (`--path`):**

Features inherit the epic's custom folder path but can override it:

```bash
# Epic with custom path
shark epic create "Q1 Initiative" --path="docs/roadmap/2025-q1"

# Feature inherits epic's custom path
shark feature create --epic=E01 "User Growth"
# Stored in docs/roadmap/2025-q1/ directory (inherited)

# Feature overrides epic's custom path
shark feature create --epic=E01 "Mobile App" --path="docs/mobile/app"
# Stored in docs/mobile/app/ directory (override)
```

**Examples:**

```bash
# Default location (no custom paths)
shark feature create --epic=E01 "Payment Integration"

# Inherit epic's custom path
shark epic create "Mobile Strategy" --path="docs/mobile/2025"
shark feature create --epic=E02 "iOS App"
# Stored in docs/mobile/2025/ directory

# Mix --path and --filename (filename takes precedence)
shark feature create --epic=E02 "Backend API" \
  --path="docs/mobile/backend" \
  --filename="docs/api/spec.md"
# File stored at docs/api/spec.md (filename wins)
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

**Arguments:**
- `<title>` - Task title (positional argument)

**Flags (required):**
- `-e, --epic <key>` - Epic key (e.g., `E01`)
- `-f, --feature <key>` - Feature key (e.g., `F02` or `E01-F02`)
- `-a, --agent <type>` - Agent type: `frontend`, `backend`, `api`, `testing`, `devops`, `general`

**Flags (optional):**
- `-d, --description <text>` - Detailed description
- `-p, --priority <n>` - Priority 1-10 (default: 5, where 1 = highest)
- `--depends-on <keys>` - Comma-separated dependency task keys
- `--filename <path>` - Custom filename path (relative to project root, must include .md extension)
- `--force` - Force reassignment if file already claimed by another task

**Custom Filename:**

By default, tasks are created in `docs/tasks/todo/` with filename pattern `T-E{epic}-F{feature}-{number}.md`.

Use `--filename` to specify a custom path:

```bash
shark task create "API Design" --epic=E04 --feature=F06 --filename="docs/plan/E04/E04-F06/api-design.md"
```

**Rules:**
- Path must be relative to project root
- Must include `.md` extension
- Existing files are automatically associated (not overwritten)
- Use `--force` to reassign files from other tasks

**Examples:**

```bash
# Basic task
shark task create "Build login form" --epic=E01 --feature=F02 --agent=frontend

# With all options
shark task create "User authentication service" \
  -e E01 \
  -f F02 \
  -a backend \
  -p 3 \
  -d "Implement JWT-based auth" \
  --depends-on="T-E01-F01-001,T-E01-F01-002"

# Custom path in plan directory
shark task create "API Design" \
  --epic=E04 \
  --feature=F06 \
  --filename="docs/plan/E04/E04-F06/api-design.md"

# Associate existing file
shark task create "Review" \
  --epic=E04 \
  --feature=F06 \
  --filename="docs/plan/E04/existing-doc.md"

# Force reassignment from another task
shark task create "New Task" \
  --epic=E04 \
  --feature=F06 \
  --filename="docs/shared.md" \
  --force
```

### `shark task list`

List tasks with optional filtering.

**Default Behavior:** Completed tasks are hidden by default to focus on active work. Use `--show-all` to include them.

**Flags:**
- `-s, --status <status>` - Filter by: `todo`, `in_progress`, `completed`, `blocked`, `ready_for_review`, `archived`
- `-e, --epic <key>` - Filter by epic key
- `-f, --feature <key>` - Filter by feature key
- `-a, --agent <type>` - Filter by agent type
- `--priority-min <n>` - Minimum priority (1-10)
- `--priority-max <n>` - Maximum priority (1-10)
- `-b, --blocked` - Show only blocked tasks
- `--show-all` - Show all tasks including completed (by default, completed tasks are hidden)

**Examples:**

```bash
# List all non-completed tasks (default)
shark task list

# List ALL tasks including completed
shark task list --show-all

# Filter by status (explicit filter overrides default hiding)
shark task list --status=todo
shark task list --status=completed  # Shows only completed tasks

# Filter by epic (still hides completed by default)
shark task list --epic=E04

# Filter by epic, show all including completed
shark task list --epic=E04 --show-all

# Filter by agent
shark task list --agent=backend

# High priority tasks (1-3)
shark task list --priority-min=1 --priority-max=3

# Only blocked tasks
shark task list --blocked

# JSON output (respects --show-all flag)
shark task list --json
shark task list --show-all --json
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

**Auto-tracking:**
- Automatically creates a work session to track time spent
- Session is associated with the task and agent
- Use `shark task sessions <task-key>` to view work session history

**Examples:**

```bash
# Start task
shark task start T-E04-F06-001

# With agent identifier
shark task start T-E04-F06-001 --agent="ai-agent-001"

# Output:
#  SUCCESS  Task T-E04-F06-001 started
# Status changed: todo → in_progress
# Work session started (ID: 42)
```

### `shark task complete <task-key>`

Mark task ready for review (in_progress → ready_for_review).

**Validations:**
- Current status must be `in_progress`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Completion notes
- `--files-created <files>` - Comma-separated list of files created
- `--files-modified <files>` - Comma-separated list of files modified
- `--tests` - Mark that tests were written/updated
- `--verified` - Mark that implementation was verified

**Auto-tracking:**
- Automatically ends the active work session
- Records session outcome as "completed"
- Stores completion metadata for future reference

**Examples:**

```bash
# Mark ready for review
shark task complete T-E04-F06-001

# With notes
shark task complete T-E04-F06-001 --notes="All tests passing"

# With completion metadata (recommended for AI agents)
shark task complete T-E04-F06-001 \
  --files-created="internal/api/login.go,internal/api/login_test.go" \
  --files-modified="internal/router/routes.go" \
  --tests \
  --verified \
  --notes="Implemented login endpoint with JWT auth"

# Output:
#  SUCCESS  Task T-E04-F06-001 marked ready for review
# Status changed: in_progress → ready_for_review
# Work session ended (Duration: 2h 15m)
# Completion metadata saved
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

## Task Intelligence Commands (E10 Features)

The following commands provide advanced task intelligence, context management, and progress tracking capabilities.

### Task Notes & Activity Tracking

#### `shark task note add <task-key> <note-text>`

Add a categorized note to a task for tracking progress, blockers, decisions, questions, or context.

**Flags:**
- `-c, --category <type>` - Note category: `progress`, `blocker`, `question`, `decision`, `context` (default: `progress`)

**Examples:**

```bash
# Add progress note
shark task note add T-E01-F01-001 "Implemented user validation logic"

# Add blocker note
shark task note add T-E01-F01-001 "Waiting for API spec" --category=blocker

# Add question note
shark task note add T-E01-F01-001 "Should we support OAuth 2.1?" --category=question

# Add decision note
shark task note add T-E01-F01-001 "Decided to use bcrypt for password hashing" --category=decision

# Add context note
shark task note add T-E01-F01-001 "Related to security audit findings" --category=context
```

#### `shark task notes <task-key>`

View all notes for a task, grouped by category.

**Examples:**

```bash
# View all notes
shark task notes T-E01-F01-001

# JSON output
shark task notes T-E01-F01-001 --json
```

#### `shark task timeline <task-key>`

View chronological timeline of task activity (status changes + notes).

**Examples:**

```bash
# View task timeline
shark task timeline T-E01-F01-001

# Output:
# Task Timeline: T-E01-F01-001
#
# 2025-12-24 10:00:00 | STATUS  | todo → in_progress
# 2025-12-24 10:15:00 | PROGRESS| Implemented user validation logic
# 2025-12-24 11:30:00 | BLOCKER | Waiting for API spec
# 2025-12-24 14:00:00 | DECISION| Decided to use bcrypt for password hashing
# 2025-12-24 16:00:00 | STATUS  | in_progress → ready_for_review

# JSON output
shark task timeline T-E01-F01-001 --json
```

#### `shark notes search <query>`

Search all task notes across the project using full-text search.

**Flags:**
- `-c, --category <type>` - Filter by category: `progress`, `blocker`, `question`, `decision`, `context`
- `-e, --epic <key>` - Filter by epic
- `-f, --feature <key>` - Filter by feature

**Examples:**

```bash
# Search all notes
shark notes search "validation"

# Search only blockers
shark notes search "API" --category=blocker

# Search within epic
shark notes search "authentication" --epic=E01

# JSON output
shark notes search "bcrypt" --json
```

### Task Relationships & Dependencies

#### `shark task link <source-key> <target-key>`

Create a relationship between two tasks.

**Flags:**
- `-t, --type <relationship>` - Relationship type: `depends-on`, `blocks`, `relates-to`, `duplicates` (default: `depends-on`)

**Examples:**

```bash
# Create dependency (source depends on target)
shark task link T-E01-F01-002 T-E01-F01-001

# Specify relationship type
shark task link T-E01-F01-003 T-E01-F01-001 --type=blocks

# Create related-to relationship
shark task link T-E01-F01-004 T-E01-F01-002 --type=relates-to

# Mark as duplicate
shark task link T-E01-F02-001 T-E01-F01-001 --type=duplicates
```

**Cycle Detection:**
- Automatically prevents circular dependencies
- Returns error if relationship would create a cycle

#### `shark task unlink <source-key> <target-key>`

Remove a relationship between two tasks.

**Examples:**

```bash
# Remove relationship
shark task unlink T-E01-F01-002 T-E01-F01-001
```

#### `shark task deps <task-key>`

View all dependencies for a task (tasks this task depends on).

**Examples:**

```bash
# View dependencies
shark task deps T-E01-F01-003

# Output:
# Dependencies for T-E01-F01-003:
#
# T-E01-F01-001 | completed  | Implement login API endpoint
# T-E01-F01-002 | in_progress| Design login UI components

# JSON output
shark task deps T-E01-F01-003 --json
```

#### `shark task blocked-by <task-key>`

View all tasks that block this task.

**Examples:**

```bash
# View blockers
shark task blocked-by T-E01-F01-005

# JSON output
shark task blocked-by T-E01-F01-005 --json
```

#### `shark task blocks <task-key>`

View all tasks that this task blocks.

**Examples:**

```bash
# View blocked tasks
shark task blocks T-E01-F01-001

# JSON output
shark task blocks T-E01-F01-001 --json
```

### Acceptance Criteria

#### `shark task criteria import <task-key>`

Import acceptance criteria from task markdown file.

**Examples:**

```bash
# Import criteria from task file
shark task criteria import T-E01-F01-001

# Output:
#  SUCCESS  Imported 5 acceptance criteria from T-E01-F01-001
```

**Task file format:**

```markdown
---
task_key: T-E01-F01-001
---

## Acceptance Criteria

- [ ] User can log in with email and password
- [ ] Invalid credentials return 401 error
- [ ] JWT token is returned on successful login
- [ ] Password is validated against bcrypt hash
- [ ] Login attempts are rate limited
```

#### `shark task criteria check <task-key> <criterion-text>`

Mark an acceptance criterion as passed.

**Examples:**

```bash
# Mark criterion as passed
shark task criteria check T-E01-F01-001 "User can log in with email and password"

# Output:
#  SUCCESS  Criterion marked as passed
```

#### `shark task criteria fail <task-key> <criterion-text>`

Mark an acceptance criterion as failed.

**Examples:**

```bash
# Mark criterion as failed
shark task criteria fail T-E01-F01-001 "Login attempts are rate limited"

# Output:
#  SUCCESS  Criterion marked as failed
```

#### `shark feature criteria <feature-key>`

View aggregated acceptance criteria status for all tasks in a feature.

**Examples:**

```bash
# View feature criteria
shark feature criteria E01-F01

# Output:
# Acceptance Criteria for Feature E01-F01:
#
# Task T-E01-F01-001 (3/5 passed):
#   ✓ User can log in with email and password
#   ✓ Invalid credentials return 401 error
#   ✓ JWT token is returned on successful login
#   ✗ Password is validated against bcrypt hash
#   ○ Login attempts are rate limited

# JSON output
shark feature criteria E01-F01 --json
```

### Work Sessions & Resume Context

#### `shark task context set <task-key>`

Save structured resume context for a task (progress, decisions, questions, blockers, acceptance criteria status).

**Flags:**
- `--progress <text>` - Current progress description
- `--decisions <text>` - Key decisions made (JSON array or newline-separated)
- `--questions <text>` - Open questions (JSON array or newline-separated)
- `--blockers <text>` - Current blockers (JSON array or newline-separated)
- `--acceptance-criteria <text>` - AC status summary

**Examples:**

```bash
# Set progress only
shark task context set T-E01-F01-001 --progress="Implemented 3 of 5 endpoints"

# Set multiple fields
shark task context set T-E01-F01-001 \
  --progress="Login API complete, UI 50% done" \
  --decisions='["Using JWT with 24h expiry", "Bcrypt rounds set to 12"]' \
  --questions='["Should we support refresh tokens?"]' \
  --blockers='["Waiting for UI mockups"]'

# Context is automatically merged (doesn't overwrite other fields)
shark task context set T-E01-F01-001 --acceptance-criteria="3 of 5 criteria passing"
```

#### `shark task context get <task-key>`

Retrieve saved context for a task.

**Examples:**

```bash
# Get context
shark task context get T-E01-F01-001

# Output:
# Context for T-E01-F01-001:
#
# Progress: Login API complete, UI 50% done
#
# Decisions:
#   - Using JWT with 24h expiry
#   - Bcrypt rounds set to 12
#
# Questions:
#   - Should we support refresh tokens?
#
# Blockers:
#   - Waiting for UI mockups

# JSON output
shark task context get T-E01-F01-001 --json
```

#### `shark task context clear <task-key>`

Clear saved context for a task.

**Examples:**

```bash
# Clear context
shark task context clear T-E01-F01-001
```

#### `shark task resume <task-key>`

Get comprehensive resume context for a task (aggregates all context for quick resumption).

**Includes:**
1. Task details (title, status, priority, agent)
2. Saved context data (progress, decisions, questions, blockers)
3. Recent notes (last 10)
4. Last work session details
5. Dependencies status
6. Related tasks
7. Acceptance criteria summary
8. Files touched (from completion metadata)
9. Timeline highlights
10. Suggested next actions

**Examples:**

```bash
# Get resume context
shark task resume T-E01-F01-001

# Output:
# ═══════════════════════════════════════════════════════════
# Resume Context: T-E01-F01-001
# Implement login API endpoint
# ═══════════════════════════════════════════════════════════
#
# [1] TASK OVERVIEW
# Status:       in_progress
# Priority:     8 (high)
# Agent:        backend
# Epic:         E01 - User Authentication System
# Feature:      E01-F01 - Login Flow
#
# [2] CURRENT PROGRESS
# Login API complete, UI 50% done
#
# [3] KEY DECISIONS MADE
#   • Using JWT with 24h expiry
#   • Bcrypt rounds set to 12
#
# [4] OPEN QUESTIONS ⚠️
#   ? Should we support refresh tokens?
#
# [5] BLOCKERS ⚠️
#   ! Waiting for UI mockups
#
# [6] LAST WORK SESSION
# Started: 2025-12-24 14:00:00
# Duration: 2h 15m (ongoing)
# Agent: backend-agent-1
#
# [7] RECENT NOTES (last 5)
# 2025-12-24 15:30 | PROGRESS | JWT validation working
# 2025-12-24 14:45 | DECISION | Using bcrypt for passwords
# 2025-12-24 14:15 | PROGRESS | Started API implementation
#
# [8] DEPENDENCIES
# ✓ T-E01-F01-001 (completed) - Set up authentication database
# ○ T-E01-F01-002 (todo) - Design login UI
#
# [9] ACCEPTANCE CRITERIA (3/5 passing)
# ✓ User can log in with email and password
# ✓ Invalid credentials return 401 error
# ✓ JWT token is returned on successful login
# ✗ Password is validated against bcrypt hash
# ○ Login attempts are rate limited
#
# [10] FILES TOUCHED
# Created: internal/api/login.go, internal/api/login_test.go
# Modified: internal/router/routes.go
#
# [11] SUGGESTED NEXT ACTIONS
#   • Complete remaining acceptance criteria (2 pending)
#   • Resolve blocker: Waiting for UI mockups
#   • Answer question: Should we support refresh tokens?
#   • Add tests for rate limiting

# JSON output (all data structured)
shark task resume T-E01-F01-001 --json
```

#### `shark task sessions <task-key>`

View all work sessions for a task with statistics.

**Examples:**

```bash
# View sessions
shark task sessions T-E01-F01-001

# Output:
# Work Sessions for T-E01-F01-001:
#
# Session #1
# Started:  2025-12-24 10:00:00
# Ended:    2025-12-24 12:30:00
# Duration: 2h 30m
# Agent:    backend-agent-1
# Outcome:  paused
#
# Session #2
# Started:  2025-12-24 14:00:00
# Ended:    2025-12-24 16:15:00
# Duration: 2h 15m
# Agent:    backend-agent-1
# Outcome:  completed
#
# Statistics:
# Total sessions: 2
# Total time:     4h 45m
# Average:        2h 22m
# Outcomes:       1 completed, 1 paused

# JSON output
shark task sessions T-E01-F01-001 --json
```

### Analytics

#### `shark analytics sessions`

Analyze work session patterns for estimation and planning.

**Flags:**
- `--session-duration` - Analyze session duration patterns
- `--pause-frequency` - Analyze pause/resume frequency
- `-e, --epic <key>` - Filter by epic
- `-f, --feature <key>` - Filter by feature
- `-a, --agent <type>` - Filter by agent type

**Examples:**

```bash
# Session duration analysis
shark analytics sessions --session-duration

# Output:
# ═══════════════════════════════════════════════════════════
# Session Duration Analysis
# ═══════════════════════════════════════════════════════════
#
# Overall Metrics:
#   Total Sessions:        47
#   Tasks with Sessions:   23
#   Sessions per Task:     2.0
#
# Time Investment:
#   Total Time:            87h 15m
#   Average Session:       1h 51m
#   Median Session:        1h 30m
#
# Distribution:
#   < 1 hour:     12 sessions (26%)
#   1-2 hours:    18 sessions (38%)
#   2-4 hours:    14 sessions (30%)
#   > 4 hours:     3 sessions (6%)
#
# Estimation Insights:
#   • Most tasks require 1-2 hour sessions
#   • Consider breaking tasks into <2 hour chunks

# Pause frequency analysis
shark analytics sessions --pause-frequency --epic=E01

# Filter by agent type
shark analytics sessions --session-duration --agent=backend

# JSON output
shark analytics sessions --session-duration --json
```

### Enhanced Search

#### `shark search --file <file-path>`

Search tasks by files they touched (created or modified).

**Examples:**

```bash
# Find tasks that touched a file
shark search --file="internal/api/login.go"

# Output:
# Tasks that touched internal/api/login.go:
#
# T-E01-F01-001 | completed | Implement login API endpoint
# T-E01-F01-005 | completed | Add rate limiting to login

# JSON output
shark search --file="internal/api/login.go" --json
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
