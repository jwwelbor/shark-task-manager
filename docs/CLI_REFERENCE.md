# Shark CLI Reference

Complete command reference for the Shark Task Manager CLI.

## Table of Contents

- [Global Flags](#global-flags)
- [Key Format Improvements](#key-format-improvements)
- [Initialization](#initialization)
- [Epic Commands](#epic-commands)
- [Feature Commands](#feature-commands)
- [Task Commands](#task-commands)
- [Task Update API Response Format](#task-update-api-response-format)
- [Enhanced JSON API Response Fields](#enhanced-json-api-response-fields)
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

#### Epic Rollups

Epic get now includes comprehensive rollup information for visibility across all features:

**Feature Status Rollup:**
Shows the distribution of features by status across the epic:
- Counts features in each workflow status
- Helps understand feature progression
- Example: `In Planning: 2, In Development: 3, In Review: 1, Completed: 4`

**Task Status Rollup:**
Aggregates task counts across all features:
- Total task count across entire epic
- Breakdown by status (todo, in_progress, ready_for_*, completed, blocked)
- Provides full workflow visibility
- Example: `Total: 47 tasks | Todo: 10 | In Progress: 15 | Ready for Review: 8 | Completed: 12 | Blocked: 2`

**Impediments:**
Lists all blocked tasks that are impeding progress:
- Blocked task key and title
- Parent feature
- Blocker reason
- Age of blockage
- Enables quick identification and resolution of blockers

**Example Table Output:**
```
Feature Status Rollup
  In Planning: 1 | In Development: 2 | In Review: 1 | Completed: 3

Task Status Rollup
  Total: 47 tasks
  Todo: 10 | In Progress: 15 | Ready for Review: 8 | Completed: 12 | Blocked: 2

Impediments
  🔴 T-E07-F01-005 "Setup OAuth providers" (Feature: Authentication, 2 days)
  🔴 T-E07-F03-012 "Configure Postgres replication" (Feature: Database, 5 days)
```

**JSON Output:** Enhanced with rollup data:
```json
{
  "id": 7,
  "key": "E07",
  "title": "User Management System",
  "progress": 55.0,
  "feature_status_rollup": {
    "in_planning": 1,
    "in_development": 2,
    "in_review": 1,
    "completed": 3
  },
  "task_status_rollup": {
    "total": 47,
    "todo": 10,
    "in_progress": 15,
    "ready_for_approval": 5,
    "ready_for_qa": 3,
    "completed": 12,
    "blocked": 2
  },
  "impediments": [
    {
      "task_key": "T-E07-F01-005",
      "task_title": "Setup OAuth providers",
      "feature_key": "E07-F01",
      "feature_title": "Authentication",
      "reason": "Waiting for OAuth provider approval",
      "blocked_since": "2026-01-14T10:00:00Z",
      "age_days": 2
    }
  ]
}
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

#### Health Indicators

Feature list now displays health indicators in table format:

- **🔴 Red**: Feature has blocked tasks
- **🟡 Yellow**: Feature has tasks in `ready_for_approval` status for more than 3 days
- **🟢 Green**: No issues detected

**Progress Format:** Shows both weighted and completion progress:
- `70.5% | 50%` = 70.5% weighted progress, 50% completion progress
- Helps identify tasks with high weight that impact overall progress

**Notes Column:** Shows count of action items:
- Number of tasks awaiting action (in ready_for_* statuses)
- Quick indicator of workload needing attention

**Example Table Output:**
```
Epic    Feature                     Progress         Status  Notes
E07     Authentication              70.5% | 50%      🟡      4 awaiting
E07     User Management             100% | 100%      🟢      0 awaiting
E07     Permission System           45.0% | 20%      🔴      2 blocked
```

**JSON Output:** Enhanced with health indicators:
```json
{
  "id": 1,
  "key": "E07-F01",
  "title": "Authentication",
  "epic_key": "E07",
  "progress": 70.5,
  "weighted_progress": 70.5,
  "completion_progress": 50.0,
  "health_status": "warning",
  "action_items_count": 4,
  "blocked_count": 0,
  "ready_for_approval_count": 4
}
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

#### Enhanced Status Information

The feature get command now includes three additional sections for improved visibility and tracking:

**Progress Breakdown:**
Shows weighted and completion progress metrics:
- **Weighted Progress**: Calculated based on configured `progress_weight` for each status
- **Completion Progress**: Raw percentage of completed tasks
- **Total Tasks**: Count of all tasks in feature

**Work Summary:**
Categorizes tasks by responsibility:
- **Completed**: Finished and approved tasks
- **Agent Work**: Tasks assigned to AI agents
- **Human Work**: Tasks requiring human engineers
- **QA Work**: Tasks for quality assurance team
- **Blocked Work**: Tasks blocked by dependencies
- **Not Started**: Todo tasks

**Action Items:**
Lists tasks awaiting action, grouped by status:
- Tasks in `ready_for_approval` status
- Tasks in `ready_for_qa` status
- Other actionable statuses from workflow config

**Example Output (Table Format):**
```
Progress Breakdown
  Weighted: 70.5% | Completion: 9.1% | Total: 11 tasks

Work Summary
  Completed: 1 | Agent Work: 0 | Human Work: 7 | QA Work: 0 | Blocked: 0 | Not Started: 3

Action Items
  Ready for Approval (4 tasks)
    - T-E07-F23-001
    - T-E07-F23-002
    - T-E07-F23-003
    - T-E07-F23-004
```

**JSON Output:**
```json
{
  "id": 1,
  "epic_id": 7,
  "key": "E07-F01",
  "title": "Authentication",
  "status": "active",
  "progress_pct": 75.0,
  "progress_info": {
    "weighted_progress_pct": 70.5,
    "completion_progress_pct": 9.1,
    "total_tasks": 11
  },
  "work_summary": {
    "total_tasks": 11,
    "completed_tasks": 1,
    "agent_work": 0,
    "human_work": 7,
    "qa_work": 0,
    "blocked_work": 0,
    "not_started": 3
  },
  "action_items": {
    "ready_for_approval": ["T-E07-F23-001", "T-E07-F23-002", "T-E07-F23-003", "T-E07-F23-004"],
    "ready_for_qa": []
  },
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
shark task list [EPIC] [FEATURE] [--status=<status>] [--agent=<type>] [--with-actions] [--json]
# OR (flag syntax, backward compatible)
shark task list [--epic=<epic-key>] [--feature=<feature-key>] [--status=<status>] [--agent=<type>] [--with-actions] [--json]
```

**Filter Flags:**
- `--status <status>`: Filter by status (`todo`, `in_progress`, `ready_for_review`, `completed`, `blocked`)
- `--agent <type>`: Filter by agent type
- `--with-actions`: Include orchestrator actions with each task (optional, for batch orchestrator polling)

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

# List with orchestrator actions (for orchestrator polling)
shark task list --status=ready_for_development --with-actions --json
shark task list E07 F01 --with-actions --json

# Combine all filters
shark task list E07 --agent=backend --status=ready_for_development --with-actions --json
```

**About the --with-actions Flag:**

The `--with-actions` flag includes `orchestrator_action` metadata with each task. This is useful for orchestrators that need to batch-fetch ready tasks and their execution instructions.

**Without flag (default):**
```json
[
  {
    "id": 123,
    "key": "E01-F03-002",
    "status": "ready_for_development",
    "title": "Implement feature X"
  }
]
```

**With --with-actions flag:**
```json
[
  {
    "id": 123,
    "key": "E01-F03-002",
    "status": "ready_for_development",
    "title": "Implement feature X",
    "orchestrator_action": {
      "action": "spawn_agent",
      "agent_type": "developer",
      "skills": ["test-driven-development", "implementation"],
      "instruction": "Launch a developer agent to implement task E01-F03-002..."
    }
  }
]
```

See [Task Update API Response Format](#task-update-api-response-format) section for complete action field documentation.

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

### `shark task next-status`

Transition a task to the next valid status in the workflow.

**Usage:**
```bash
shark task next-status <task-key> [--status=<status>] [--json]
```

**Flags:**
- `--status <status>`: Explicitly specify target status (skips selection)
- `--json`: Output in JSON format

**Behavior:**

When multiple valid transitions are available, behavior depends on the `interactive_mode` configuration:

**Non-Interactive Mode (Default):**
- Automatically selects the first valid transition from workflow configuration
- Prints info message showing which status was selected
- Example: `ℹ Auto-selected next status: in_qa (from 2 options)`

**Interactive Mode (Opt-In):**
- Displays interactive prompt with numbered options
- Waits for user input (1-N or Ctrl+C to cancel)
- Requires `interactive_mode: true` in `.sharkconfig.json`

**When Only One Transition Available:**
- Always auto-selects the single option (both modes)
- No prompt or selection message needed

**Examples:**

```bash
# Non-interactive mode (default) - auto-selects first transition
shark task next-status E07-F23-006
# Output:
# ℹ Auto-selected next status: in_qa (from 2 options)
# ✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa

# Interactive mode (when enabled in config)
# Requires: { "interactive_mode": true } in .sharkconfig.json
shark task next-status E07-F23-006
# Output:
# Task: T-E07-F23-006
# Current status: ready_for_qa
#
# Available transitions:
#   1) in_qa
#   2) on_hold
#
# Enter selection [1-2]: 1
# ✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa

# Explicit status (skips selection in both modes)
shark task next-status E07-F23-006 --status=in_qa
# ✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa

# JSON output
shark task next-status E07-F23-006 --json
# Returns available transitions if multiple options

# Case insensitive
shark task next-status e07-f23-006
shark task next-status T-E07-F23-006  # Traditional format also works
```

**Auto-Selection Logic:**

The first transition in the workflow configuration is selected:

```json
{
  "status_flow": {
    "ready_for_qa": ["in_qa", "on_hold"]
    //               ^^^^^^^^ <- This is auto-selected (non-interactive mode)
  }
}
```

**Configuration Impact:**

| Config Setting | Multiple Transitions | Single Transition |
|----------------|---------------------|-------------------|
| `interactive_mode: false` (default) | Auto-selects first option | Auto-selects only option |
| `interactive_mode: true` | Shows interactive prompt | Auto-selects only option |
| `--status` flag provided | Uses specified status | Uses specified status |

**Use Cases:**

- **Agent/Automation Workflows:** Use default non-interactive mode
- **CI/CD Pipelines:** Use default non-interactive mode
- **Human Manual Operations:** Enable interactive mode in config
- **Explicit Control:** Use `--status` flag to specify exact transition

**Related Configuration:**
- See [Interactive Mode Configuration](#interactive-mode-configuration) for details on `interactive_mode` setting
- See [Workflow Configuration](#workflow-configuration) for status flow definitions

---

## Task Update API Response Format

This section documents the enhanced task update API response format that includes `orchestrator_action` metadata for AI Agent Orchestrators.

### Overview

When tasks transition status (via `shark task start`, `shark task complete`, `shark task approve`, etc.), the API response includes optional `orchestrator_action` metadata. This metadata tells orchestrators what action to take next and which agent should be spawned (if applicable).

**Key Features:**
- Atomic response: Task state + action metadata in single API call
- Backward compatible: Missing actions don't break existing code
- Flexible: Actions are defined per-status in configuration
- Optional: Clients can safely ignore missing actions

### JSON Response Structure

#### Complete Task Response with Orchestrator Action

```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "slug": "implement-feature-x",
  "feature_id": 45,
  "epic_id": 7,
  "title": "Implement feature X",
  "description": "Detailed task description",
  "status": "ready_for_development",
  "priority": 5,
  "agent_type": "developer",
  "depends_on": ["T-E01-F03-001"],
  "file_path": "docs/plan/E01/E01-F03/tasks/T-E01-F03-002.md",
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-15T12:30:00Z",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation", "shark-task-management"],
    "instruction": "Launch a developer agent to implement task T-E01-F03-002. Write tests first, then implement to pass tests following the technical specifications."
  }
}
```

#### Response Without Orchestrator Action

When no action is defined for a status, the `orchestrator_action` field is **omitted entirely** (not null):

```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "in_progress",
  "title": "Implement feature X"
  // NO orchestrator_action field
}
```

### OrchestratorAction Object Schema

The `orchestrator_action` object contains metadata to guide orchestrator behavior:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `action` | String | Yes | Action type: `spawn_agent`, `pause`, `wait_for_triage`, or `archive` |
| `agent_type` | String | Conditional | Required if `action=spawn_agent`. Type of agent to spawn (e.g., `developer`, `architect`, `reviewer`) |
| `skills` | String Array | Conditional | Required if `action=spawn_agent`. Array of skills the agent should have (e.g., `["test-driven-development", "implementation"]`) |
| `instruction` | String | Conditional | Required if `action=spawn_agent` or `pause`. Human-readable instruction template with template variables populated (e.g., `{task_id}` replaced with actual task key) |

### Action Types

#### spawn_agent

Launch a new agent to work on the task.

**Required Fields:**
- `action`: `"spawn_agent"`
- `agent_type`: Agent role/type
- `skills`: Array of required skills
- `instruction`: Detailed instruction for the agent

**Example:**
```json
{
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation", "shark-task-management"],
    "instruction": "Launch a developer agent to implement task T-E01-F03-002. Write tests first, then implement to pass tests. Reference the technical specifications in the task file."
  }
}
```

**Common agent_type values:**
- `developer`: Backend/general development
- `frontend-developer`: Frontend-specific work
- `architect`: System design and architecture
- `reviewer`: Code review and quality assurance
- `test-engineer`: Testing and QA
- `devops`: Infrastructure and deployment

#### pause

Wait before taking action (e.g., waiting for external dependencies).

**Required Fields:**
- `action`: `"pause"`
- `instruction`: Reason for pause and what to wait for

**Example:**
```json
{
  "orchestrator_action": {
    "action": "pause",
    "instruction": "Task T-E01-F03-002 is blocked waiting for API specification from architect. Check back when ready_for_development status is reached."
  }
}
```

#### wait_for_triage

Task requires manual review and assignment before proceeding.

**Required Fields:**
- `action`: `"wait_for_triage"`
- `instruction`: Triage instructions

**Example:**
```json
{
  "orchestrator_action": {
    "action": "wait_for_triage",
    "instruction": "Task T-E01-F03-002 requires manual triage to assign to appropriate team. Review dependencies and priority before assigning."
  }
}
```

#### archive

Task is complete and should be archived/ignored.

**Required Fields:**
- `action`: `"archive"`
- `instruction`: Reason for archival

**Example:**
```json
{
  "orchestrator_action": {
    "action": "archive",
    "instruction": "Task T-E01-F03-002 is completed and archived. No further action needed."
  }
}
```

### Template Variables

The `instruction` field may contain template variables that are automatically populated at runtime:

| Variable | Description | Example |
|----------|-------------|---------|
| `{task_id}` | Task key (normalized) | `T-E01-F03-002` or `E01-F03-002` |

**Example Template:**
```
"instruction": "Launch a developer agent to implement task {task_id}. Follow test-driven development practices."
```

**After Substitution:**
```
"instruction": "Launch a developer agent to implement task T-E01-F03-002. Follow test-driven development practices."
```

### Response Examples by Command

#### Task Start Command
```bash
$ shark task start E01-F03-002 --json
```

Response (task transitions to `in_progress`):
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "in_progress",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation"],
    "instruction": "Launch a developer agent to implement task T-E01-F03-002..."
  }
}
```

#### Task Complete Command
```bash
$ shark task complete E01-F03-002 --json
```

Response (task transitions to `ready_for_review`):
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "ready_for_review",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "reviewer",
    "skills": ["code-review", "quality-assurance"],
    "instruction": "Launch a code reviewer agent to review task T-E01-F03-002..."
  }
}
```

#### Task Approve Command
```bash
$ shark task approve E01-F03-002 --json
```

Response (task transitions to `completed`):
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "completed",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "archive",
    "instruction": "Task T-E01-F03-002 is completed. Archive and prepare final report."
  }
}
```

#### Task Block Command
```bash
$ shark task block E01-F03-002 --reason="Waiting for API design" --json
```

Response (task transitions to `blocked`):
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "blocked",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "pause",
    "instruction": "Task T-E01-F03-002 is blocked: Waiting for API design. Check back when unblocked."
  }
}
```

### Integration Guide

#### Parsing Orchestrator Action in Go

```go
// Define response structure
type TaskResponse struct {
    ID                   int64                `json:"id"`
    Key                  string               `json:"key"`
    Status               string               `json:"status"`
    Title                string               `json:"title"`
    OrchestratorAction   *OrchestratorAction  `json:"orchestrator_action,omitempty"`
}

type OrchestratorAction struct {
    Action      string   `json:"action"`
    AgentType   string   `json:"agent_type,omitempty"`
    Skills      []string `json:"skills,omitempty"`
    Instruction string   `json:"instruction"`
}

// Parse response
func parseTaskResponse(jsonData []byte) (*TaskResponse, error) {
    var task TaskResponse
    if err := json.Unmarshal(jsonData, &task); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    return &task, nil
}

// Handle orchestrator action
func handleTaskResponse(task *TaskResponse) error {
    // Action is optional - check presence before accessing
    if task.OrchestratorAction == nil {
        log.Printf("Task %s updated, no action defined", task.Key)
        return nil
    }

    action := task.OrchestratorAction
    switch action.Action {
    case "spawn_agent":
        return spawnAgent(task.Key, action.AgentType, action.Skills, action.Instruction)
    case "pause":
        log.Printf("Pausing: %s", action.Instruction)
        return nil
    case "wait_for_triage":
        log.Printf("Waiting for triage: %s", action.Instruction)
        return nil
    case "archive":
        return archiveTask(task.Key)
    default:
        return fmt.Errorf("unknown action type: %s", action.Action)
    }
}

// Spawn agent based on action
func spawnAgent(taskID, agentType string, skills []string, instruction string) error {
    agent := &Agent{
        Type:        agentType,
        Skills:      skills,
        TaskID:      taskID,
        Instruction: instruction,
    }
    return orchestrator.Spawn(agent)
}
```

#### Parsing in Python

```python
import json
import logging

def parse_task_response(json_data):
    """Parse task response from shark CLI."""
    task = json.loads(json_data)
    return task

def handle_task_response(task):
    """Process orchestrator action from task response."""
    action = task.get('orchestrator_action')

    # Action is optional
    if action is None:
        logging.info(f"Task {task['key']} updated, no action defined")
        return

    action_type = action['action']

    if action_type == 'spawn_agent':
        spawn_agent(
            task_id=task['key'],
            agent_type=action['agent_type'],
            skills=action['skills'],
            instruction=action['instruction']
        )
    elif action_type == 'pause':
        logging.info(f"Pausing: {action['instruction']}")
    elif action_type == 'wait_for_triage':
        logging.info(f"Waiting for triage: {action['instruction']}")
    elif action_type == 'archive':
        archive_task(task['key'])
    else:
        raise ValueError(f"Unknown action type: {action_type}")

def spawn_agent(task_id, agent_type, skills, instruction):
    """Spawn an agent to work on the task."""
    # Implementation depends on your orchestrator
    pass
```

### Backward Compatibility

**Guarantee**: The `orchestrator_action` field is optional and omitted when not defined. Existing code continues to work without changes.

**Migration Path** (if you have existing orchestrators):

1. **No changes required** - Code ignoring `orchestrator_action` continues working
2. **Optional enhancement** - Add handling for `orchestrator_action` when available
3. **Gradual rollout** - Enable actions per-status in configuration as needed

**Before (no actions):**
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "ready_for_development",
  "title": "Implement feature X"
}
```

**After (with actions):**
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "ready_for_development",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation"],
    "instruction": "Launch a developer agent..."
  }
}
```

Existing orchestrators ignore the new field. New orchestrators can opt-in to using it.

### Error Handling

#### Missing Actions

Missing actions are **not errors**. The `orchestrator_action` field is simply omitted from the response:

```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "in_progress",
  "title": "Implement feature X"
  // No orchestrator_action field
}
```

**Recommended handling:**
```go
if action := task.OrchestratorAction; action != nil {
    // Handle action
} else {
    // No action defined - orchestrator should have fallback logic
    log.Warn("No action defined for task", "task_id", task.Key)
}
```

#### Invalid Actions

Invalid actions are caught at configuration load time (not at response time). The CLI will fail to start with helpful error messages.

#### Template Variable Errors

If template variable population fails, the template is returned as-is (with unpopulated variables). Log a warning and continue:

```bash
# If {task_id} fails to populate:
"instruction": "Launch agent for task {task_id}"  # Unpopulated placeholder

# Orchestrator should handle gracefully:
if strings.Contains(instruction, "{") {
    log.Warn("Template not fully populated", "instruction", instruction)
}
```

### Debugging

#### View Orchestrator Action for a Task

Get the current action for a task without modifying status:

```bash
# Get task details (no status change)
shark task get E01-F03-002 --json

# Check the orchestrator_action field (if present)
shark task get E01-F03-002 --json | jq '.orchestrator_action'
```

#### View Actions in Task List

Retrieve multiple ready tasks with their actions (Story 5 feature):

```bash
# List tasks in ready_for_development status WITH actions
shark task list --status=ready_for_development --with-actions --json

# View actions for specific epic/feature
shark task list E01 F03 --with-actions --json
```

### Configuration

Orchestrator actions are defined in the `.sharkconfig.json` workflow configuration:

```json
{
  "status_metadata": {
    "ready_for_development": {
      "color": "yellow",
      "phase": "development",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["test-driven-development", "implementation", "shark-task-management"],
        "instruction_template": "Launch a developer agent to implement task {task_id}. Write tests first, then implement to pass tests following the technical specifications."
      }
    },
    "ready_for_code_review": {
      "color": "magenta",
      "phase": "review",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "reviewer",
        "skills": ["code-review", "quality-assurance"],
        "instruction_template": "Launch a reviewer agent to review task {task_id}. Check code quality, tests, and compliance with specifications."
      }
    }
  }
}
```

### Related Commands

- `shark task start` - Transition task to in_progress (may return action)
- `shark task complete` - Transition task to ready_for_review (may return action)
- `shark task approve` - Transition task to completed (may return action)
- `shark task reopen` - Revert task from review (may return action)
- `shark task block` - Transition task to blocked (may return action)
- `shark task unblock` - Transition task from blocked (may return action)
- `shark task list --with-actions` - Batch-fetch multiple tasks with actions

---

## Enhanced JSON API Response Fields

This section documents the enhanced JSON response fields added for improved status tracking and visibility across features and epics.

### Feature Get Enhanced Fields

The `shark feature get` command returns additional fields for comprehensive status visibility:

**Progress Information:**
```json
"progress_info": {
  "weighted_progress_pct": 70.5,
  "completion_progress_pct": 9.1,
  "total_tasks": 11
}
```

**Work Summary (by responsibility):**
```json
"work_summary": {
  "total_tasks": 11,
  "completed_tasks": 1,
  "agent_work": 0,
  "human_work": 7,
  "qa_work": 0,
  "blocked_work": 0,
  "not_started": 3
}
```

**Action Items (by status):**
```json
"action_items": {
  "ready_for_approval": ["T-E07-F23-001", "T-E07-F23-002"],
  "ready_for_qa": ["T-E07-F23-006"]
}
```

**Complete Feature Response:**
```json
{
  "id": 1,
  "key": "E07-F01",
  "title": "Authentication",
  "status": "active",
  "progress_pct": 75.0,
  "progress_info": {
    "weighted_progress_pct": 70.5,
    "completion_progress_pct": 9.1,
    "total_tasks": 11
  },
  "work_summary": {
    "total_tasks": 11,
    "completed_tasks": 1,
    "agent_work": 0,
    "human_work": 7,
    "qa_work": 0,
    "blocked_work": 0,
    "not_started": 3
  },
  "action_items": {
    "ready_for_approval": ["T-E07-F23-001", "T-E07-F23-002"],
    "ready_for_qa": ["T-E07-F23-006"]
  }
}
```

### Feature List Enhanced Fields

The `shark feature list` command returns health indicators and dual progress metrics:

```json
{
  "id": 1,
  "key": "E07-F01",
  "title": "Authentication",
  "progress": 70.5,
  "weighted_progress": 70.5,
  "completion_progress": 50.0,
  "health_status": "warning",
  "action_items_count": 4,
  "blocked_count": 0,
  "ready_for_approval_count": 4
}
```

**Health Status Values:**
- `"healthy"`: No blockers, all approval tasks < 3 days
- `"warning"`: Ready for approval tasks > 3 days, or minor blockers
- `"critical"`: Multiple blockers or high-priority tasks blocked

### Epic Get Enhanced Fields

The `shark epic get` command returns rollup information across all features:

**Feature Status Rollup:**
```json
"feature_status_rollup": {
  "in_planning": 1,
  "in_development": 2,
  "in_review": 1,
  "completed": 3
}
```

**Task Status Rollup:**
```json
"task_status_rollup": {
  "total": 47,
  "todo": 10,
  "in_progress": 15,
  "ready_for_approval": 5,
  "ready_for_qa": 3,
  "completed": 12,
  "blocked": 2
}
```

**Impediments (Blocked Tasks):**
```json
"impediments": [
  {
    "task_key": "T-E07-F01-005",
    "task_title": "Setup OAuth providers",
    "feature_key": "E07-F01",
    "feature_title": "Authentication",
    "reason": "Waiting for OAuth provider approval",
    "blocked_since": "2026-01-14T10:00:00Z",
    "age_days": 2
  }
]
```

**Complete Epic Response:**
```json
{
  "id": 7,
  "key": "E07",
  "title": "User Management System",
  "progress": 55.0,
  "feature_status_rollup": {
    "in_planning": 1,
    "in_development": 2,
    "in_review": 1,
    "completed": 3
  },
  "task_status_rollup": {
    "total": 47,
    "todo": 10,
    "in_progress": 15,
    "ready_for_approval": 5,
    "ready_for_qa": 3,
    "completed": 12,
    "blocked": 2
  },
  "impediments": [
    {
      "task_key": "T-E07-F01-005",
      "task_title": "Setup OAuth providers",
      "feature_key": "E07-F01",
      "feature_title": "Authentication",
      "reason": "Waiting for OAuth provider approval",
      "blocked_since": "2026-01-14T10:00:00Z",
      "age_days": 2
    }
  ]
}
```

### Configuration-Driven Calculations

All enhanced fields are calculated based on status configuration in `.sharkconfig.json`:

**Status Metadata Configuration:**
```json
{
  "status_metadata": {
    "completed": {
      "color": "green",
      "phase": "done",
      "progress_weight": 100,
      "responsibility": "none",
      "blocks_feature": false
    },
    "ready_for_approval": {
      "color": "magenta",
      "phase": "review",
      "progress_weight": 75,
      "responsibility": "human",
      "blocks_feature": true
    },
    "in_development": {
      "color": "yellow",
      "phase": "development",
      "progress_weight": 50,
      "responsibility": "agent",
      "blocks_feature": false
    }
  }
}
```

**Field Calculations:**

1. **Weighted Progress**: `(sum of progress_weight * task_count) / total_tasks * 100`
2. **Completion Progress**: `(completed_tasks / total_tasks) * 100`
3. **Work Breakdown**: Grouped by `responsibility` field (agent, human, qa_team, none)
4. **Health Status**: Based on `blocks_feature` statuses and age of approval tasks
5. **Action Items**: Tasks in statuses with `blocks_feature: true`

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

### Interactive Mode Configuration

The `interactive_mode` configuration field controls whether commands prompt for user input when multiple options are available.

**Configuration Field:**
- **Name:** `interactive_mode`
- **Type:** Boolean
- **Default:** `false` (non-interactive)
- **Purpose:** Controls interactive prompts in status transition commands

**Default Behavior (Non-Interactive):**
- Commands automatically select the first valid option from workflow configuration
- Ideal for agent/automation workflows
- Prints clear message showing which option was selected
- Never blocks waiting for user input

**Interactive Mode (Opt-In):**
- Commands display interactive prompts for user selection
- Requires manual input when multiple options available
- Suitable for human users who want explicit control
- Enable by setting `interactive_mode: true` in `.sharkconfig.json`

**Example Configuration:**

```json
{
  "interactive_mode": false,
  "status_flow": {
    "ready_for_qa": ["in_qa", "on_hold"],
    "in_qa": ["ready_for_approval", "in_development"]
  },
  "status_metadata": {
    "in_qa": {
      "color": "yellow",
      "phase": "qa",
      "progress_weight": 80
    }
  }
}
```

**Usage Examples:**

```bash
# Non-interactive mode (default)
$ shark task next-status E07-F23-006
ℹ Auto-selected next status: in_qa (from 2 options)
✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa

# Interactive mode (when enabled in config)
# .sharkconfig.json: { "interactive_mode": true }
$ shark task next-status E07-F23-006
Task: T-E07-F23-006
Current status: ready_for_qa

Available transitions:
  1) in_qa
  2) on_hold

Enter selection [1-2]: 1
✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa
```

**When to Use Each Mode:**

| Use Case | Recommended Mode | Reason |
|----------|------------------|--------|
| AI Agent workflows | Non-interactive (default) | Agents can't provide interactive input |
| CI/CD pipelines | Non-interactive (default) | Automation requires predictable behavior |
| Scripts/batch operations | Non-interactive (default) | Background processes need non-blocking execution |
| Human users (manual) | Interactive (opt-in) | Explicit control over status transitions |
| Development/debugging | Interactive (opt-in) | Review options before selecting |

**Auto-Selection Behavior:**

When `interactive_mode` is `false` (default) and multiple transitions are available, the command automatically selects the first transition defined in the workflow configuration:

```json
{
  "status_flow": {
    "ready_for_qa": ["in_qa", "on_hold"]
    //               ^^^^^^^^  <- This is auto-selected
  }
}
```

The order in `status_flow` determines selection priority. Place the most common/preferred transition first.

---

## Workflow Configuration

Shark supports customizable workflow configuration through `.sharkconfig.json`. This allows you to define custom status flows, colors, phases, and agent types.

### Configuration Structure

```json
{
  "interactive_mode": false,
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

**interactive_mode**: Controls interactive prompts for status transitions (optional, default: `false`)
- `false` (default): Auto-select first transition when multiple options available (ideal for agents/automation)
- `true`: Show interactive prompt for user selection when multiple options available
- See [Interactive Mode Configuration](#interactive-mode-configuration) for detailed documentation

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
