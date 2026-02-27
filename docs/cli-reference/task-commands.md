# Task Commands

Complete reference for all task management commands in Shark Task Manager.

## Overview

Tasks are atomic work units in Shark's hierarchy: **Epic > Feature > Task**. Each task represents a single implementable change with clear acceptance criteria, status tracking, dependency management, and work session history.

Tasks flow through a configurable workflow. The basic profile uses 5 statuses, while the advanced profile supports 19 statuses across planning, development, review, QA, and approval phases. See [Workflow Profiles](../guides/workflow-profiles.md) for details.

**Key format:** All task commands accept case-insensitive keys in multiple formats:
- Short format (recommended): `E07-F01-001`
- Traditional format: `T-E07-F01-001`
- Slugged format: `E07-F01-001-implement-jwt-validation`

## Quick Reference

| Command | Description | Category |
|---------|-------------|----------|
| `shark task start` | Start working on a task | Lifecycle |
| `shark task complete` | Mark task as ready for review | Lifecycle |
| `shark task approve` | Approve task for completion | Lifecycle |
| `shark task reopen` | Reopen a task for rework | Lifecycle |
| `shark task block` | Block a task with a reason | Lifecycle |
| `shark task unblock` | Unblock a task | Lifecycle |
| `shark task next` | Get next available task | Lifecycle |
| `shark task next-status` | Advance to next workflow status | Lifecycle |
| `shark task set-status` | Set task to a specific status | Lifecycle |
| `shark task create` | Create a new task | CRUD |
| `shark task get` | Get task details | CRUD |
| `shark task list` | List tasks with filtering | CRUD |
| `shark task delete` | Delete a task | CRUD |
| `shark task update` | Update task properties | CRUD |
| `shark task blocked-by` | Show incoming dependencies | Dependencies |
| `shark task blocks` | Show outgoing blockers | Dependencies |
| `shark task deps` | Show all relationships | Dependencies |
| `shark task link` | Create typed relationships | Dependencies |
| `shark task unlink` | Remove typed relationships | Dependencies |
| `shark task context` | Manage structured context data | Context & Notes |
| `shark task note` | Add typed notes | Context & Notes |
| `shark task notes` | List notes for a task | Context & Notes |
| `shark task criteria` | Manage acceptance criteria | Context & Notes |
| `shark task resume` | Get full context for resuming work | Context & Notes |
| `shark task history` | Show status change history | History & Sessions |
| `shark task sessions` | View work sessions | History & Sessions |
| `shark task timeline` | Show chronological timeline | History & Sessions |

---

## Lifecycle Commands

Commands for transitioning tasks through the workflow.

### `shark task start`

Start working on a task. Transitions from a ready status to an active development status.

**Usage:**
```bash
shark task start <task-key> [flags]
```

**Flags:**
- `--agent <string>` - Agent identifier (defaults to USER env var)
- `--force` - Force status change bypassing validation
- `--json` - Output in JSON format

**Examples:**
```bash
# Start a task
shark task start E07-F01-001

# Start with agent tracking
shark task start E07-F01-001 --agent=dev-agent-001

# Force start from any status (administrative override)
shark task start E07-F01-001 --force
```

**Behavior:**
1. Validates current status allows starting
2. Updates status to `in_development` (or appropriate active status)
3. Records transition in task history
4. Sets `started_at` timestamp

---

### `shark task complete`

Mark a task as complete (ready for review). Records completion metadata including files changed, test status, and time spent.

**Usage:**
```bash
shark task complete <task-key> [flags]
```

**Flags:**
- `-n, --notes <string>` - Completion notes
- `--summary <string>` - Completion summary describing what was delivered
- `--agent <string>` - Agent identifier
- `--agent-id <string>` - Agent execution ID for traceability
- `--files-created <strings>` - Files created during task (repeatable)
- `--files-modified <strings>` - Files modified during task (repeatable)
- `--tests <string>` - Test status summary (e.g., "16/16 passing")
- `--time-spent <int>` - Time spent in minutes
- `--verified` - Mark task as verified
- `--force` - Force status change bypassing validation
- `--json` - Output in JSON format

**Examples:**
```bash
# Mark complete with notes
shark task complete E07-F01-001 --notes="Implementation complete, all tests passing"

# Complete with full metadata
shark task complete E07-F01-001 \
  --summary="Added JWT token validation endpoint" \
  --tests="24/24 passing" \
  --files-modified=internal/auth/jwt.go,internal/auth/jwt_test.go \
  --time-spent=120

# Complete with JSON output
shark task complete E07-F01-001 --notes="Done" --json
```

**Behavior:**
1. Validates task is in an active development status
2. Advances to the next review status (e.g., `ready_for_code_review`)
3. Stores completion metadata (files, tests, time spent)
4. Records notes in task history

---

### `shark task approve`

Approve a task for completion. Transitions from a review status to `completed`.

**Usage:**
```bash
shark task approve <task-key> [flags]
```

**Flags:**
- `-n, --notes <string>` - Approval notes
- `--agent <string>` - Agent identifier
- `--rejection-reason <string>` - Reason for rejection (if rejecting instead)
- `--reason-doc <string>` - Path to rejection reason document
- `--force` - Force status change bypassing validation
- `--json` - Output in JSON format

**Examples:**
```bash
# Approve a task
shark task approve E07-F01-001

# Approve with notes
shark task approve E07-F01-001 --notes="LGTM, code review passed"

# Approve with JSON output
shark task approve E07-F01-001 --notes="Approved" --json
```

---

### `shark task reopen`

Reopen a task for rework. Sends a task backward in the workflow, typically from a review status back to development.

**Usage:**
```bash
shark task reopen <task-key> [flags]
```

**Flags:**
- `-n, --notes <string>` - Rework notes
- `--rejection-reason <string>` - Reason for rejection (required for backward transitions)
- `--reason-doc <string>` - Path to rejection reason document
- `--agent <string>` - Agent identifier
- `--force` - Force status change bypassing validation
- `--json` - Output in JSON format

**Examples:**
```bash
# Reopen with rejection reason
shark task reopen E07-F01-001 \
  --rejection-reason="Missing error handling on line 67"

# Reopen with linked review document
shark task reopen E07-F01-001 \
  --rejection-reason="Found 3 issues in review" \
  --reason-doc="docs/reviews/E07-F01-001-review.md"

# Force reopen without rejection reason (not recommended)
shark task reopen E07-F01-001 --force
```

**About Rejection Reasons:**
- Required for backward transitions in the workflow
- Stored as rejection history and visible in `shark task get`
- Helps developers understand exactly what needs fixing
- See [Rejection Reasons](rejection-reasons.md) for detailed documentation

---

### `shark task block`

Block a task due to an external dependency or impediment.

**Usage:**
```bash
shark task block <task-key> [flags]
```

**Flags:**
- `-r, --reason <string>` - Reason for blocking (required)
- `--agent <string>` - Agent identifier
- `--force` - Force status change bypassing validation
- `--json` - Output in JSON format

**Examples:**
```bash
# Block with reason
shark task block E07-F01-001 --reason="Waiting for API spec from E07-F02"

# Block with short flag
shark task block E07-F01-001 -r "Dependencies not ready"

# Block with JSON output
shark task block E07-F01-001 --reason="Blocked by external dependency" --json
```

---

### `shark task unblock`

Unblock a task and return it to its previous status (typically `draft` or the prior active status).

**Usage:**
```bash
shark task unblock <task-key> [flags]
```

**Flags:**
- `--agent <string>` - Agent identifier
- `--force` - Force status change bypassing validation
- `--json` - Output in JSON format

**Examples:**
```bash
# Unblock a task
shark task unblock E07-F01-001

# Unblock with JSON output
shark task unblock E07-F01-001 --json
```

---

### `shark task next`

Find the next available task to work on based on dependencies, priority, and agent type.

**Usage:**
```bash
shark task next [flags]
```

**Flags:**
- `-a, --agent <string>` - Agent type to match (e.g., backend, frontend, qa)
- `-e, --epic <string>` - Filter by epic key
- `--json` - Output in JSON format

**Examples:**
```bash
# Get next available task
shark task next

# Next task for a specific agent type
shark task next --agent=backend

# Next task in a specific epic
shark task next --epic=E07

# Combine filters with JSON
shark task next --agent=backend --epic=E07 --json
```

**Selection Logic:**
1. Filter by agent type (if specified)
2. Filter by epic (if specified)
3. Filter by status: only "ready_for_*" statuses
4. Sort by execution order (ascending), then priority (descending), then created date (oldest first)
5. Return first match

---

### `shark task next-status`

Advance a task to the next valid status in the workflow. Supports both interactive and non-interactive mode.

**Usage:**
```bash
shark task next-status <task-key> [flags]
```

**Flags:**
- `--status <string>` - Target status for direct transition (non-interactive)
- `--preview` - Show available transitions without making changes
- `--force` - Bypass workflow validation
- `--reason <string>` - Rejection reason for backward transitions
- `--reason-doc <string>` - Path to document detailing rejection reason
- `--json` - Output in JSON format

**Examples:**
```bash
# Auto-advance to next status (non-interactive default)
shark task next-status E07-F01-001

# Preview available transitions
shark task next-status E07-F01-001 --preview

# Transition to a specific status
shark task next-status E07-F01-001 --status=ready_for_code_review

# JSON output for scripting
shark task next-status E07-F01-001 --json
```

**Behavior:**
- **Non-interactive mode (default):** Auto-selects the first valid transition
- **Interactive mode (opt-in via config):** Shows numbered prompt for selection
- **Single transition available:** Auto-selects in both modes
- Use `--status` flag for explicit control in automation

---

### `shark task set-status`

Set a task to a specific status with workflow validation.

**Usage:**
```bash
shark task set-status <task-key> <status> [flags]
```

**Flags:**
- `--notes <string>` - Notes to record with status transition
- `--force` - Force status change bypassing workflow validation
- `--json` - Output in JSON format

**Examples:**
```bash
# Set task status
shark task set-status E07-F01-001 in_development

# Set with notes
shark task set-status E07-F01-001 ready_for_qa --notes="Code review approved"

# Force backward transition
shark task set-status E07-F01-001 draft --force
```

---

## CRUD Commands

Commands for creating, reading, updating, and deleting tasks.

### `shark task create`

Create a new task within a feature.

**Positional Syntax (Recommended):**
```bash
# 3-argument format
shark task create <epic-key> <feature-key> "<title>" [flags]

# 2-argument format
shark task create <epic-feature-key> "<title>" [flags]
```

**Flag Syntax (Legacy):**
```bash
shark task create --epic=<epic-key> --feature=<feature-key> --title="<title>" [flags]
```

**Flags:**
- `-a, --agent <string>` - Agent type (e.g., backend, frontend, qa, architect)
- `-p, --priority <int>` - Priority level 1-10 (1=highest, default: 5)
- `--order <int>` - Execution order within feature (lower runs first)
- `-d, --description <string>` - Detailed description
- `--depends-on <string>` - Comma-separated dependency task keys
- `--key <string>` - Custom task key
- `--file <string>` - Custom file path (relative to root, must include .md)
- `--create` - Create file if it does not exist
- `--force` - Force reassignment if file already claimed
- `--json` - Output in JSON format

**Examples:**
```bash
# 3-argument format (recommended)
shark task create E07 F01 "Implement JWT validation"

# 2-argument format
shark task create E07-F01 "Implement JWT validation"

# With execution order and agent
shark task create E07 F01 "Setup JWT library" --order=1 --agent=backend

# With dependencies
shark task create E07 F01 "Add token refresh" \
  --depends-on="E07-F01-001,E07-F01-002"

# Case insensitive
shark task create e07 f01 "Task Title"
```

---

### `shark task get`

Get detailed information about a specific task including status, dependencies, related documents, and completion metadata.

**Usage:**
```bash
shark task get <task-key> [flags]
```

**Flags:**
- `--completion-details` - Display completion metadata details (files changed, test results)
- `--json` - Output in JSON format

**Examples:**
```bash
# Get task details
shark task get E07-F01-001

# Get with completion metadata
shark task get E07-F01-001 --completion-details

# JSON output
shark task get E07-F01-001 --json
```

**JSON Response Fields:**
```json
{
  "task": {
    "id": 5001,
    "key": "T-E07-F01-001",
    "slug": "implement-token-generation",
    "title": "Implement token generation",
    "status": "in_development",
    "agent_type": "backend",
    "priority": 5,
    "execution_order": 2,
    "file_path": "docs/plan/.../T-E07-F01-001.md"
  },
  "dependencies": [],
  "blocks": ["T-E07-F01-003"],
  "related_docs": ["docs/plan/.../02-architecture.md"],
  "status_metadata": {
    "phase": "development",
    "color": "yellow",
    "progress_weight": 0.5,
    "responsibility": "agent"
  }
}
```

---

### `shark task list`

List tasks with optional filtering by epic, feature, status, agent, and more.

**Usage:**
```bash
shark task list [EPIC] [FEATURE] [flags]
```

**Flags:**
- `-e, --epic <string>` - Filter by epic key
- `-f, --feature <string>` - Filter by feature key
- `-s, --status <string>` - Filter by status
- `-a, --agent <string>` - Filter by assigned agent
- `-b, --blocked` - Show only blocked tasks
- `--all` - Show all tasks including completed (completed hidden by default)
- `--has-rejections` - Filter tasks that have rejections
- `--priority-min <int>` - Minimum priority (1=highest)
- `--priority-max <int>` - Maximum priority (10=lowest)
- `--with-actions` - Include orchestrator actions with each task
- `--json` - Output in JSON format

**Examples:**
```bash
# List all non-completed tasks
shark task list

# List tasks in an epic
shark task list E07

# List tasks in a feature (positional)
shark task list E07 F01

# Filter by status and agent
shark task list --status=in_development --agent=backend

# Show blocked tasks only
shark task list --blocked

# Include orchestrator actions for automation
shark task list --status=ready_for_development --with-actions --json
```

**The `--with-actions` Flag:**

When included, each task in the JSON response gains an `orchestrator_action` field containing agent dispatch instructions. See [Orchestrator Actions](orchestrator-actions.md) for details.

---

### `shark task delete`

Delete a task permanently. This cannot be undone; task history is deleted via CASCADE.

**Usage:**
```bash
shark task delete <task-key> [flags]
```

**Flags:**
- `--json` - Output in JSON format

**Examples:**
```bash
# Delete a task
shark task delete E07-F01-001

# Delete with JSON confirmation
shark task delete E07-F01-001 --json
```

---

### `shark task update`

Update task properties such as title, description, priority, agent, execution order, and status.

**Usage:**
```bash
shark task update <task-key> [flags]
```

**Flags:**
- `--title <string>` - New title
- `-d, --description <string>` - New description
- `-a, --agent <string>` - New agent type
- `-p, --priority <int>` - New priority (1-10, -1=no change)
- `--order <int>` - New execution order (-1=no change)
- `--status <string>` - New status (uses workflow validation)
- `--depends-on <string>` - New comma-separated dependency task keys
- `--key <string>` - New key
- `--filename <string>` - New file path (relative to project root)
- `--reason <string>` - Reason for backward status transitions
- `--reason-doc <string>` - Path to rejection reason document
- `--force` - Force reassignment or bypass workflow validation
- `--json` - Output in JSON format

**Examples:**
```bash
# Update title
shark task update E07-F01-001 --title="Implement JWT token generation with RSA"

# Update priority and agent
shark task update E07-F01-001 --priority=1 --agent=backend

# Update execution order
shark task update E07-F01-001 --order=1

# Update multiple fields
shark task update E07-F01-001 --priority=10 --order=1
```

---

## Dependency Commands

Commands for managing relationships between tasks.

### `shark task blocked-by`

Show all tasks that this task depends on (incoming dependencies). These are the tasks that must complete before this task can start.

**Usage:**
```bash
shark task blocked-by <task-key> [flags]
```

**Flags:**
- `--json` - Output in JSON format

**Examples:**
```bash
# Show what blocks this task
shark task blocked-by E07-F01-004

# JSON output
shark task blocked-by E07-F01-004 --json
```

---

### `shark task blocks`

Show all tasks that depend on this task completing (outgoing blockers). These are the tasks waiting for this task to finish.

**Usage:**
```bash
shark task blocks <task-key> [flags]
```

**Flags:**
- `--json` - Output in JSON format

**Examples:**
```bash
# Show what this task blocks
shark task blocks E07-F01-003

# JSON output
shark task blocks E07-F01-003 --json
```

---

### `shark task deps`

Show all relationships for a task including dependencies, blockers, related tasks, and other typed relationships. Supports tree visualization.

**Usage:**
```bash
shark task deps <task-key> [flags]
```

**Flags:**
- `--tree` - Show dependency tree visualization
- `--upstream` - Show upstream dependencies (prerequisites)
- `--downstream` - Show downstream dependents (tasks waiting on this)
- `--type <string>` - Filter by relationship types (comma-separated)
- `--max-depth <int>` - Maximum tree depth (default: 10)
- `--json` - Output in JSON format

**Examples:**
```bash
# Show all relationships
shark task deps E07-F01-004

# Show as dependency tree
shark task deps E07-F01-004 --tree

# Show upstream dependencies only
shark task deps E07-F01-004 --tree --upstream

# Show downstream dependents only
shark task deps E07-F01-004 --tree --downstream

# Filter by relationship type
shark task deps E07-F01-004 --type depends_on,blocks
```

---

### `shark task link`

Create typed relationships between tasks to track dependencies, blockers, and related work.

**Usage:**
```bash
shark task link <task-key> [flags]
```

**Relationship Types:**
- `depends_on` - Task depends on another completing (hard dependency)
- `blocks` - Task blocks another from proceeding
- `related_to` - Tasks share common code/concerns
- `follows` - Task naturally follows another (soft ordering)
- `spawned_from` - Task was created from UAT/bugs in another
- `duplicates` - Tasks represent duplicate work
- `references` - Task consults/uses output of another

**Flags:**
- `--depends-on <string>` - Create depends_on relationships (comma-separated task keys)
- `--blocks <string>` - Create blocks relationships
- `--related-to <string>` - Create related_to relationships
- `--follows <string>` - Create follows relationships
- `--spawned-from <string>` - Create spawned_from relationships
- `--duplicates <string>` - Create duplicates relationships
- `--references <string>` - Create references relationships
- `--json` - Output in JSON format

**Examples:**
```bash
# Single dependency
shark task link E07-F01-004 --depends-on T-E07-F01-003

# Multiple dependencies
shark task link E07-F01-004 --depends-on T-E07-F01-003,T-E07-F01-001

# Multiple relationship types
shark task link E07-F01-004 --depends-on T-E07-F01-003 --related-to T-E07-F01-002

# Spawned task from UAT findings
shark task link E07-F01-008 --spawned-from T-E07-F01-002
```

---

### `shark task unlink`

Remove typed relationships between tasks.

**Usage:**
```bash
shark task unlink <task-key> [flags]
```

**Flags:**
- `--depends-on <string>` - Remove depends_on relationships (comma-separated task keys)
- `--blocks <string>` - Remove blocks relationships
- `--related-to <string>` - Remove related_to relationships
- `--follows <string>` - Remove follows relationships
- `--spawned-from <string>` - Remove spawned_from relationships
- `--duplicates <string>` - Remove duplicates relationships
- `--references <string>` - Remove references relationships
- `--type <string>` - Relationship type to remove (use with `--all`)
- `--all` - Remove all relationships of the specified type
- `--json` - Output in JSON format

**Examples:**
```bash
# Remove specific dependency
shark task unlink E07-F01-004 --depends-on T-E07-F01-003

# Remove multiple relationships
shark task unlink E07-F01-004 --depends-on T-E07-F01-003,T-E07-F01-001

# Remove all relationships of a type
shark task unlink E07-F01-004 --type depends_on --all
```

---

## Context & Notes Commands

Commands for managing structured context, notes, acceptance criteria, and resume data.

### `shark task context`

Manage structured resume context data for tasks. Context data includes progress tracking, implementation decisions, open questions, blockers, and acceptance criteria status.

**Subcommands:**
- `get` - Get context data (default when no subcommand)
- `set` - Set or update a context field
- `clear` - Clear all context data

**Usage:**
```bash
shark task context <task-key> [flags]
shark task context get <task-key> [flags]
shark task context set <task-key> --field <field> --value <value> [flags]
shark task context clear <task-key> [flags]
```

**Supported Fields for `set`:**
- `current_step` - String describing current work step
- `completed_steps` - JSON array of completed steps
- `remaining_steps` - JSON array of remaining steps
- `implementation_decisions` - JSON object with decision key-value pairs
- `open_questions` - JSON array of question strings
- `blockers` - JSON array of blocker objects
- `acceptance_criteria_status` - JSON array of criterion objects

**Examples:**
```bash
# Get task context
shark task context E07-F01-001

# Set current step
shark task context set E07-F01-001 --field current_step --value "Implementing API endpoint"

# Set completed steps
shark task context set E07-F01-001 --field completed_steps --value '["Setup project","Write tests"]'

# Set open questions
shark task context set E07-F01-001 --field open_questions --value '["What API version?"]'

# Clear all context
shark task context clear E07-F01-001

# JSON output
shark task context E07-F01-001 --json
```

---

### `shark task note`

Add typed notes to a task for context, decisions, blockers, and documentation.

**Subcommands:**
- `add` - Add a typed note to a task

**Usage:**
```bash
shark task note add <task-key> --type <type> "<content>" [flags]
```

**Note Types:**
- `comment` - General observation
- `decision` - Why we chose X over Y
- `blocker` - What is blocking progress
- `solution` - How we solved a problem
- `reference` - External links, documentation
- `implementation` - What we actually built
- `testing` - Test results, coverage
- `future` - Future improvements / TODO
- `question` - Unanswered questions

**Flags:**
- `-t, --type <string>` - Note type (required)
- `-c, --created-by <string>` - Creator name (optional)
- `--json` - Output in JSON format

**Examples:**
```bash
# Decision note
shark task note add E07-F01-001 --type decision "Used bcrypt for password hashing"

# Blocker note with author
shark task note add E07-F01-001 --type blocker "Waiting for API spec" --created-by alice

# Solution note
shark task note add E07-F01-001 --type solution "Fixed by adding null check"

# Reference note
shark task note add E07-F01-001 --type reference "https://jwt.io/introduction"
```

---

### `shark task notes`

List all notes for a task, optionally filtered by type.

**Usage:**
```bash
shark task notes <task-key> [flags]
```

**Flags:**
- `-t, --type <string>` - Filter by note type (comma-separated for multiple)
- `--json` - Output in JSON format

**Examples:**
```bash
# List all notes
shark task notes E07-F01-001

# List decision notes only
shark task notes E07-F01-001 --type decision

# List multiple types
shark task notes E07-F01-001 --type decision,solution

# JSON output
shark task notes E07-F01-001 --json
```

---

### `shark task criteria`

Import, view, and manage acceptance criteria for tasks. Criteria are extracted from markdown checklist format in task files.

**Subcommands:**
- `list` - List acceptance criteria with status summary
- `import` - Import criteria from task markdown file
- `check` - Mark a criterion as complete (verified)
- `fail` - Mark a criterion as failed

**Usage:**
```bash
shark task criteria list <task-key> [flags]
shark task criteria import <task-key> [flags]
shark task criteria check <task-key> <criterion-id> [flags]
shark task criteria fail <task-key> <criterion-id> [flags]
```

**Flags (check):**
- `-n, --note <string>` - Verification notes (optional)

**Flags (fail):**
- `-n, --note <string>` - Failure reason (required)

**Examples:**
```bash
# Import criteria from task markdown
shark task criteria import E07-F01-001

# List criteria with status summary
shark task criteria list E07-F01-001

# Mark criterion as complete
shark task criteria check E07-F01-001 5 --note "Verified with unit tests"

# Mark criterion as failed
shark task criteria fail E07-F01-001 3 --note "Performance threshold not met"

# JSON output
shark task criteria list E07-F01-001 --json
```

**Criteria Display:**
- Total, complete, pending, failed, in_progress, and n/a counts
- Status icons: check (complete), x (failed), circle (pending), half-circle (in_progress), dash (n/a)
- Percentage calculation (e.g., "85% complete (6/7 criteria)")

---

### `shark task resume`

Get all context needed to resume work on a task in a single command. This is the recommended way for agents to pick up work on an existing task.

**Usage:**
```bash
shark task resume <task-key> [flags]
```

**Flags:**
- `--json` - Output in JSON format

**Includes:**
- Task details (title, description, status, priority, dependencies)
- Context data (progress, decisions, questions, blockers, acceptance criteria)
- Task notes (chronologically ordered)
- Completion metadata (if task is completed)
- Work sessions (all sessions with durations and outcomes)

**Examples:**
```bash
# Resume a task (human-readable)
shark task resume E07-F01-001

# Resume with JSON output (for agents)
shark task resume E07-F01-001 --json
```

---

## History & Sessions Commands

Commands for viewing task history, work sessions, and timelines.

### `shark task history`

Display the complete lifecycle history of a task showing all status transitions with timestamps, agents, and notes.

**Usage:**
```bash
shark task history <task-key> [flags]
```

**Flags:**
- `--format <string>` - Output format: `csv` or `json`
- `--json` - Output in JSON format

**Examples:**
```bash
# Show task history
shark task history E07-F01-001

# CSV format
shark task history E07-F01-001 --format=csv

# JSON format
shark task history E07-F01-001 --json
```

**Sample Output:**
```
Task T-E07-F01-001 Change History

Date                  From Status           To Status             Notes
------------------------------------------------------------------------
2026-02-16 11:30:00   in_development        ready_for_code_rev... Implementation complete
2026-02-16 09:15:00   ready_for_development in_development        Started work
2026-02-15 16:45:00   in_refinement_tech    ready_for_development Architecture approved
```

---

### `shark task sessions`

View all work sessions for a task with durations, outcomes, and aggregate statistics.

**Usage:**
```bash
shark task sessions <task-key> [flags]
```

**Flags:**
- `--json` - Output in JSON format

**Shows:**
- Session start/end times
- Duration for each session
- Session outcome (completed, paused, blocked)
- Session notes
- Total time spent
- Average session duration

**Examples:**
```bash
# View work sessions
shark task sessions E07-F01-001

# JSON output
shark task sessions E07-F01-001 --json
```

---

### `shark task timeline`

Show a unified chronological timeline of status changes and notes for a task. Interleaves task history entries with notes to provide a complete picture of what happened.

**Usage:**
```bash
shark task timeline <task-key> [flags]
```

**Flags:**
- `--json` - Output in JSON format

**Examples:**
```bash
# Show timeline
shark task timeline E07-F01-001

# JSON output
shark task timeline E07-F01-001 --json
```

---

## Common Workflows

### Pick Up Next Task (Agent Workflow)

```bash
# 1. Get next task for your agent type
shark task next --agent=backend --json

# 2. Resume context (if previously worked on)
shark task resume E07-F01-002

# 3. Start the task
shark task start E07-F01-002 --agent=dev-agent-001

# 4. Track progress with context
shark task context set E07-F01-002 --field current_step --value "Writing unit tests"

# 5. Add notes as you work
shark task note add E07-F01-002 --type decision "Used bcrypt for password hashing"

# 6. Mark complete when done
shark task complete E07-F01-002 --notes="All tests passing" --tests="24/24 passing"

# 7. Advance to next status
shark task next-status E07-F01-002
```

### Complete Task Through Review

```bash
# 1. Developer marks complete
shark task complete E07-F01-001 --notes="Implementation complete"

# 2. Code reviewer approves
shark task approve E07-F01-001 --notes="LGTM"

# 3. Or reviewer requests changes
shark task reopen E07-F01-001 \
  --rejection-reason="Missing error handling on line 67"
```

### Handle Blocked Task

```bash
# 1. Block the task
shark task block E07-F01-003 --reason="Waiting for API spec from E07-F02"

# 2. Check blocked tasks
shark task list --blocked

# 3. Unblock when ready
shark task unblock E07-F01-003

# 4. Resume work
shark task start E07-F01-003
```

### Manage Dependencies

```bash
# Create tasks with execution order
shark task create E07 F01 "Setup JWT library" --order=1
shark task create E07 F01 "Generate tokens" --order=2 --agent=backend
shark task create E07 F01 "Validate tokens" --order=3

# Add dependency
shark task link E07-F01-003 --depends-on T-E07-F01-002

# View dependency tree
shark task deps E07-F01-003 --tree

# Check what blocks a task
shark task blocked-by E07-F01-003
```

---

## Key Concepts

### Execution Order vs Priority

**Execution Order (Primary):**
- Determines sequence of implementation within a feature
- Lower values run first
- Used by `shark task next` as primary sort
- Example: Setup (order=1) then Implementation (order=2) then Tests (order=3)

**Priority (Secondary):**
- Determines importance when execution order is equal
- Scale 1-10 where 1 is highest priority
- Used as tiebreaker when execution_order matches

**Sorting logic:** `ORDER BY execution_order ASC, priority DESC, created_at ASC`

### Agent Types

Shark supports flexible agent type assignment. Any non-empty string works.

**Standard types** (have specific templates): `frontend`, `backend`, `api`, `testing`, `devops`, `general`

**Custom types** (use general template): `architect`, `business-analyst`, `qa`, `tech-lead`, `product-manager`, `ux-designer`, or any custom string

```bash
# Query by agent type
shark task list --agent=architect
shark task next --agent=business-analyst --json
```

### Progress Weight

Each status has a `progress_weight` (0.0-1.0) reflecting completion percentage. Feature progress is calculated as the average of all task weights.

---

## JSON Output

All task commands support `--json` for machine-readable output. Additionally, the global `--field` flag can extract a single field:

```bash
# Get just the status
shark task get E07-F01-001 --json --field status

# Get just the key
shark task next --agent=backend --json --field key
```

---

## Related Documentation

- [Feature Commands](feature-commands.md) - Parent entity management
- [Epic Commands](epic-commands.md) - Top-level entity management
- [Rejection Reasons](rejection-reasons.md) - Rejection workflow documentation
- [Orchestrator Actions](orchestrator-actions.md) - API response format for AI orchestrators
- [JSON API Fields](json-api-fields.md) - Enhanced JSON response fields
- [Key Formats](key-formats.md) - Case insensitive and slugged key formats
- [Interactive Mode](interactive-mode.md) - Configure interactive prompts
- [Workflow Configuration](workflow-config.md) - Status flows and phases
- [Workflow Profiles](../guides/workflow-profiles.md) - Basic vs advanced workflow profiles
