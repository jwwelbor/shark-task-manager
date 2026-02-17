# Task CLI Commands

Complete reference for task management commands.

## Overview

Tasks are atomic work units in Shark's hierarchy (Epic → Feature → **Task**). They represent single implementable changes with clear acceptance criteria.

**Task Lifecycle:**
```
draft → refinement_ba → refinement_tech → ready_for_development →
in_development → ready_for_code_review → in_code_review →
ready_for_qa → in_qa → ready_for_approval → in_approval → completed
```

## Commands

### `shark task create`

Create a new task in a feature.

#### Syntax (Positional - Recommended)

```bash
# 3-argument format
shark task create <epic> <feature> "<title>" [flags]

# 2-argument format (combined epic-feature)
shark task create <epic-feature> "<title>" [flags]
```

#### Syntax (Flag-Based - Legacy)

```bash
shark task create --epic=<epic> --feature=<feature> --title="<title>" [flags]
```

#### Arguments

- `<epic>` - Epic key (e.g., E07, e07)
- `<feature>` - Feature key (e.g., F01, f01)
- `<epic-feature>` - Combined format (e.g., E07-F01)
- `<title>` - Task title (quoted string)

#### Optional Flags

- `--agent=<type>` - Agent type (e.g., backend, frontend, qa)
- `--priority=<1-10>` - Task priority (default: 5)
- `--order=<n>` - Execution order (primary sequencing)
- `--depends-on=<task-key>` - Dependency task keys (comma-separated)
- `--file=<path>` - Custom file path
- `--force` - Force reassignment if file already claimed
- `--json` - Output JSON response

#### Examples

**3-argument format:**
```bash
shark task create E07 F01 "Implement token generation"
```

**2-argument format:**
```bash
shark task create E07-F01 "Implement token validation"
```

**With agent type:**
```bash
shark task create E07 F01 "Design token API" --agent=backend
```

**With execution order (recommended for sequencing):**
```bash
shark task create E07 F01 "Setup JWT library" --order=1
shark task create E07 F01 "Implement token generation" --order=2 --agent=backend
```

**With priority (secondary to order):**
```bash
shark task create E07 F01 "Critical security fix" --order=1 --priority=10
```

**With dependencies:**
```bash
shark task create E07 F01 "Test token refresh" --depends-on=T-E07-F01-001,T-E07-F01-002
```

**Case insensitive:**
```bash
shark task create e07 f01 "Task Title"
```

**JSON output:**
```bash
shark task create E07 F01 "Token generation" --json
```

```json
{
  "id": 5001,
  "key": "T-E07-F01-001",
  "slug": "token-generation",
  "title": "Token generation",
  "epic_id": 7,
  "epic_key": "E07",
  "feature_id": 201,
  "feature_key": "E07-F01",
  "status": "draft",
  "agent_type": null,
  "priority": 5,
  "execution_order": 1,
  "file_path": "docs/plan/.../tasks/T-E07-F01-001.md",
  "created_at": "2026-02-16T11:00:00Z",
  "updated_at": "2026-02-16T11:00:00Z"
}
```

---

### `shark task list`

List tasks with filtering.

#### Syntax (Positional - Recommended)

```bash
shark task list [EPIC] [FEATURE] [flags]
```

#### Syntax (Flag-Based - Legacy)

```bash
shark task list [--epic=<epic>] [--feature=<feature>] [flags]
```

#### Optional Arguments

- `[EPIC]` - Epic key filter
- `[FEATURE]` - Feature key filter (short or full format)

#### Optional Flags

- `--status=<status>` - Filter by status
- `--agent=<type>` - Filter by agent type
- `--show-all` - Include completed tasks (default: hide completed)
- `--json` - Output JSON response

#### Examples

**All non-completed tasks:**
```bash
shark task list
```

**All tasks (including completed):**
```bash
shark task list --show-all
```

**Tasks in epic:**
```bash
shark task list E07
```

**Tasks in feature (positional):**
```bash
shark task list E07 F01
shark task list E07-F01      # Combined format
```

**Tasks by status:**
```bash
shark task list --status=ready_for_development
shark task list --status=completed --show-all
```

**Tasks by agent:**
```bash
shark task list --agent=backend
shark task list E07 --agent=frontend
```

**Output:**
```
Key              Title                            Status                  Agent     Priority
──────────────────────────────────────────────────────────────────────────────────────────
T-E07-F01-001    Implement token generation       in_development          backend          5
T-E07-F01-002    Token validation endpoint        ready_for_development   backend          5
T-E07-F01-003    Token refresh logic              draft                   backend          3
T-E07-F01-004    Design token API                 ready_for_code_review   backend          8
T-E07-F01-005    Token expiry tests               ready_for_qa            qa               5
```

---

### `shark task get`

Get detailed task information.

#### Syntax

```bash
shark task get <task-key> [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive, short or traditional format)

#### Optional Flags

- `--json` - Output JSON response

#### Examples

**Short format:**
```bash
shark task get E07-F01-001
```

**Traditional format:**
```bash
shark task get T-E07-F01-001
```

**Case insensitive:**
```bash
shark task get e07-f01-001
```

**Human-readable output:**
```
═══════════════════════════════════════════════════════════════════
Task: T-E07-F01-001 - Implement token generation
═══════════════════════════════════════════════════════════════════

Feature: E07-F01 - JWT Token Management
Epic: E07 - User Authentication System

Status: in_development
Phase: development
Agent Type: backend
Priority: 5
Execution Order: 2

File: docs/plan/.../tasks/T-E07-F01-001.md

Dependencies: None
Blocks: T-E07-F01-003, T-E07-F01-004

Created: 2026-01-20 10:00:00
Updated: 2026-02-16 11:30:00
Started: 2026-02-16 09:15:00

───────────────────────────────────────────────────────────────────
Progress: 50%
Time in Status: 2h 15m

Related Documents:
- docs/plan/.../02-architecture.md
- docs/plan/.../09-test-plan.md
```

**JSON output:**
```bash
shark task get E07-F01-001 --json
```

```json
{
  "task": {
    "id": 5001,
    "key": "T-E07-F01-001",
    "slug": "implement-token-generation",
    "title": "Implement token generation",
    "epic_id": 7,
    "epic_key": "E07",
    "feature_id": 201,
    "feature_key": "E07-F01",
    "status": "in_development",
    "agent_type": "backend",
    "priority": 5,
    "execution_order": 2,
    "file_path": "docs/plan/.../T-E07-F01-001.md",
    "created_at": "2026-01-20T10:00:00Z",
    "updated_at": "2026-02-16T11:30:00Z",
    "started_at": "2026-02-16T09:15:00Z"
  },
  "dependencies": [],
  "blocks": [
    "T-E07-F01-003",
    "T-E07-F01-004"
  ],
  "related_docs": [
    "docs/plan/.../02-architecture.md",
    "docs/plan/.../09-test-plan.md"
  ],
  "status_metadata": {
    "phase": "development",
    "color": "yellow",
    "progress_weight": 0.5,
    "responsibility": "agent"
  }
}
```

---

### `shark task next`

Get next available task for an agent.

#### Syntax

```bash
shark task next [flags]
```

#### Optional Flags

- `--agent=<type>` - Agent type filter (e.g., backend, frontend, qa)
- `--epic=<epic>` - Epic filter
- `--json` - Output JSON response

#### Examples

**Next task for any agent:**
```bash
shark task next
```

**Next backend task:**
```bash
shark task next --agent=backend
```

**Next task in epic:**
```bash
shark task next --epic=E07
```

**Next backend task in epic:**
```bash
shark task next --agent=backend --epic=E07
```

**Output:**
```
Next Task: T-E07-F01-002
Title: Token validation endpoint
Feature: E07-F01 - JWT Token Management
Status: ready_for_development
Agent: backend
Priority: 5

To start: shark task start T-E07-F01-002
```

**JSON output:**
```bash
shark task next --agent=backend --json
```

```json
{
  "key": "T-E07-F01-002",
  "title": "Token validation endpoint",
  "feature_key": "E07-F01",
  "feature_title": "JWT Token Management",
  "epic_key": "E07",
  "status": "ready_for_development",
  "agent_type": "backend",
  "priority": 5,
  "execution_order": 3,
  "file_path": "docs/plan/.../T-E07-F01-002.md"
}
```

#### Selection Logic

1. **Filter by agent type** (if specified)
2. **Filter by epic** (if specified)
3. **Filter by status**: Only "ready_for_*" statuses
4. **Sort by**:
   - Execution order (primary, ascending - lower values first)
   - Priority (secondary, descending - higher values first)
   - Created date (tertiary, oldest first)
5. **Return first match**

---

### `shark task start`

Start working on a task.

#### Syntax

```bash
shark task start <task-key> [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)

#### Optional Flags

- `--agent=<agent-id>` - Agent ID starting the task
- `--json` - Output JSON response

#### Examples

**Start task:**
```bash
shark task start T-E07-F01-001
```

**Start with agent ID:**
```bash
shark task start T-E07-F01-001 --agent=dev-agent-123
```

**Output:**
```
Task T-E07-F01-001 started
Status: ready_for_development → in_development
```

#### Behavior

1. Validates current status allows starting
2. Updates status to `in_development` (or appropriate "in_*" status)
3. Records transition in task history
4. Sets `started_at` timestamp
5. Auto-unblocks dependent tasks (if this was a blocker)

---

### `shark task complete`

Mark task as complete (ready for review).

#### Syntax

```bash
shark task complete <task-key> [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)

#### Optional Flags

- `--notes="Completion notes"` - Completion notes
- `--json` - Output JSON response

#### Examples

**Mark complete:**
```bash
shark task complete T-E07-F01-001
```

**With notes:**
```bash
shark task complete T-E07-F01-001 --notes="Implementation complete, all tests passing"
```

**Output:**
```
Task T-E07-F01-001 marked complete
Status: in_development → ready_for_code_review
```

#### Behavior

1. Validates task is in active development status
2. Advances to next review status (e.g., `ready_for_code_review`)
3. Records notes in task history
4. Triggers status cascade (updates feature/epic progress)

---

### `shark task approve`

Approve a task (mark as completed).

#### Syntax

```bash
shark task approve <task-key> [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)

#### Optional Flags

- `--notes="Approval notes"` - Approval notes
- `--json` - Output JSON response

#### Examples

**Approve task:**
```bash
shark task approve T-E07-F01-001
```

**With notes:**
```bash
shark task approve T-E07-F01-001 --notes="Code review passed, UAT approved"
```

**Output:**
```
Task T-E07-F01-001 approved
Status: in_approval → completed
```

---

### `shark task reopen`

Reopen a task (back to in_progress).

#### Syntax

```bash
shark task reopen <task-key> [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)

#### Optional Flags

- `--notes="Reopen reason"` - Required if `require_rejection_reason: true` in config
- `--json` - Output JSON response

#### Examples

**Reopen task:**
```bash
shark task reopen T-E07-F01-001 --notes="Test failures in token expiry logic"
```

**Output:**
```
Task T-E07-F01-001 reopened
Status: ready_for_approval → in_development
```

---

### `shark task block`

Block a task due to external dependency.

#### Syntax

```bash
shark task block <task-key> --reason="<reason>" [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)

#### Required Flags

- `--reason="<reason>"` - Blocking reason

#### Optional Flags

- `--json` - Output JSON response

#### Examples

**Block task:**
```bash
shark task block T-E07-F01-003 --reason="Waiting for API spec from E07-F02"
```

**Output:**
```
Task T-E07-F01-003 blocked
Status: in_development → blocked
Reason: Waiting for API spec from E07-F02
```

---

### `shark task unblock`

Unblock a task.

#### Syntax

```bash
shark task unblock <task-key> [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)

#### Optional Flags

- `--notes="Resolution notes"` - Notes on how blocker was resolved
- `--json` - Output JSON response

#### Examples

**Unblock task:**
```bash
shark task unblock T-E07-F01-003 --notes="API spec received, ready to resume"
```

**Output:**
```
Task T-E07-F01-003 unblocked
Status: blocked → in_development
```

---

### `shark task update`

Update task fields.

#### Syntax

```bash
shark task update <task-key> [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)

#### Optional Flags

- `--title="New Title"` - Update title
- `--status=<status>` - Update status (must follow workflow)
- `--agent=<type>` - Update agent type
- `--priority=<1-10>` - Update priority
- `--order=<n>` - Update execution order
- `--note="Update reason"` - Add note to history
- `--json` - Output JSON response

#### Examples

**Update title:**
```bash
shark task update T-E07-F01-001 --title="Implement JWT token generation with RSA"
```

**Update agent:**
```bash
shark task update T-E07-F01-001 --agent=backend
```

**Update priority:**
```bash
shark task update T-E07-F01-001 --priority=10
```

**Update execution order:**
```bash
shark task update T-E07-F01-001 --order=1
```

**Multiple fields:**
```bash
shark task update T-E07-F01-001 --priority=10 --order=1 --note="Elevated to highest priority"
```

---

### `shark task history`

View task change history.

#### Syntax

```bash
shark task history <task-key> [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)

#### Optional Flags

- `--json` - Output JSON response
- `--limit=<n>` - Limit results (default: 50)

#### Examples

```bash
shark task history T-E07-F01-001
```

**Output:**
```
────────────────────────────────────────────────────────────────────
Task T-E07-F01-001 Change History
────────────────────────────────────────────────────────────────────

Date                  From Status         To Status           Notes
────────────────────────────────────────────────────────────────────
2026-02-16 11:30:00   in_development      ready_for_code_r... Implementation complete
2026-02-16 09:15:00   ready_for_develop.. in_development      Started work
2026-02-15 16:45:00   in_refinement_tech  ready_for_develop.. Architecture approved
2026-02-15 10:30:00   ready_for_refinem.. in_refinement_tech  Architect assigned
2026-02-15 09:00:00   draft               ready_for_refinem.. Initial triage
```

---

### `shark task note`

Add a note to task.

#### Syntax

```bash
shark task note <task-key> "<note-text>" [flags]
```

#### Arguments

- `<task-key>` - Task key (case-insensitive)
- `<note-text>` - Note content (quoted string)

#### Optional Flags

- `--metadata <key>=<value>` - Add structured metadata
- `--json` - Output JSON response

#### Examples

**Simple note:**
```bash
shark task note T-E07-F01-001 "Waiting on code review from tech lead"
```

**Note with metadata:**
```bash
shark task note T-E07-F01-001 "Test coverage at 95%" --metadata test_coverage=95
```

---

## Key Concepts

### Task Status Flow

**Planning Phase:**
1. `draft` - Initial creation
2. `ready_for_refinement_ba` - Ready for BA spec
3. `in_refinement_ba` - BA specifying
4. `ready_for_refinement_tech` - Ready for tech design
5. `in_refinement_tech` - Architect designing

**Development Phase:**
6. `ready_for_development` - Spec complete, ready to implement
7. `in_development` - Implementation in progress

**Review Phase:**
8. `ready_for_code_review` - Code complete, awaiting review
9. `in_code_review` - Tech lead reviewing

**QA Phase:**
10. `ready_for_qa` - Ready for testing
11. `in_qa` - QA testing

**Approval Phase:**
12. `ready_for_approval` - Ready for final approval
13. `in_approval` - Product owner reviewing

**Terminal Phases:**
14. `completed` - Work done
15. `cancelled` - Task abandoned
16. `blocked` - External blocker
17. `on_hold` - Paused

### Execution Order vs Priority

**Execution Order (Primary):**
- Determines sequence of implementation
- Lower values run first
- Used by `shark task next` as primary sort
- Example: Setup (order=1) → Implementation (order=2) → Tests (order=3)

**Priority (Secondary):**
- Determines importance when order is equal
- Higher values are more important (1-10 scale)
- Used as tiebreaker when execution_order is the same
- Example: Critical bug (priority=10) vs Nice-to-have (priority=3)

**Sorting logic:**
```
ORDER BY execution_order ASC, priority DESC, created_at ASC
```

### Dependencies

**Hard dependencies** (`depends_on`):
- Task cannot start until dependencies complete
- Enforced by workflow validation
- Prevents circular dependencies

**Soft dependencies** (blocking):
- Task B depends on Task A
- If Task A blocks, Task B auto-blocks
- When Task A unblocks, Task B auto-unblocks

### Agent Types

**Standard agents:**
- `backend` - Backend/API development
- `frontend` - UI/UX development
- `qa` - Testing and quality assurance
- `devops` - Infrastructure and deployment

**Custom agents:**
- `business-analyst` - Requirements specification
- `architect` - Technical design
- `tech-lead` - Code review
- `product-manager` - Feature coordination
- `researcher` - Codebase research

**Agent routing:**
- Status metadata defines `agent_types` for each status
- `shark task next --agent=<type>` filters to that agent
- Multiple agent types allowed per status (e.g., ["tech-lead", "code-reviewer"])

### Progress Weight

Each status has a progress_weight (0.0-1.0) reflecting completion:

```json
{
  "draft": 0.0,
  "ready_for_refinement": 0.05,
  "in_development": 0.5,
  "ready_for_code_review": 0.75,
  "ready_for_qa": 0.8,
  "ready_for_approval": 0.9,
  "completed": 1.0
}
```

**Feature progress calculation:**
```
Feature Progress = Σ(task_progress_weight) / task_count * 100%
```

## Common Workflows

### Pick Up Next Task

```bash
# 1. Get next task
shark task next --agent=backend

# 2. Start work
shark task start T-E07-F01-002

# 3. Verify status
shark task get T-E07-F01-002
```

### Complete Task Through Review

```bash
# 1. Mark complete (ready for code review)
shark task complete T-E07-F01-001 --notes="Implementation complete, all tests passing"

# 2. Code review passes (agent or manual)
shark task approve T-E07-F01-001 --notes="LGTM"

# 3. Verify completion
shark task get T-E07-F01-001
```

### Handle Blocked Task

```bash
# 1. Block task
shark task block T-E07-F01-003 --reason="Waiting for API spec from E07-F02"

# 2. Check blocked tasks
shark task list --status=blocked

# 3. Unblock when ready
shark task unblock T-E07-F01-003 --notes="API spec received"

# 4. Resume work
shark task start T-E07-F01-003
```

### Prioritize and Sequence Tasks

```bash
# Create tasks with execution order
shark task create E07 F01 "Setup JWT library" --order=1 --priority=10
shark task create E07 F01 "Generate tokens" --order=2 --agent=backend
shark task create E07 F01 "Validate tokens" --order=3 --depends-on=T-E07-F01-002

# Get next task (respects order)
shark task next --agent=backend
# Returns: T-E07-F01-001 (order=1)
```

## Related Documentation

- **[epic-cli.md](epic-cli.md)** - Epic commands
- **[feature-cli.md](feature-cli.md)** - Feature commands
- **[workflow-configuration.md](workflow-configuration.md)** - Status flows
- **[template-system.md](template-system.md)** - Agent instructions
