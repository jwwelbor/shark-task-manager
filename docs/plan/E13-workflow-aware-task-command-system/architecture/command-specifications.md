# Command Specifications: claim, finish, reject

**Epic**: [E13 Workflow-Aware Task Command System](../epic.md)

**Last Updated**: 2026-01-11

---

## Overview

This document provides detailed specifications for the three new phase-aware commands that replace hardcoded workflow assumptions with dynamic workflow-based transitions.

---

## Command: shark task claim

### Purpose

Claims a task for an agent, transitioning from `ready_for_X` → `in_X` and starting a work session.

### Syntax

```bash
shark task claim <task-key> [--agent=<type>] [--json]
```

### Arguments

| Argument | Required | Type | Description |
|----------|----------|------|-------------|
| task-key | Yes | string | Task identifier (T-E##-F##-###, E##-F##-###, or slugged) |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --agent | string | (auto) | Agent type/identifier claiming task |
| --json | bool | false | Output as JSON |

### Agent Auto-Detection

**Priority Order**:
1. `--agent` flag value (explicit)
2. `SHARK_AGENT_TYPE` environment variable
3. `~/.sharkconfig` user default agent
4. `$USER` environment variable
5. "unknown"

### Behavior

**Preconditions**:
- Task exists in database
- Task status matches `ready_for_*` pattern
- Workflow allows transition to corresponding `in_*` status

**Actions** (atomic transaction):
1. Validate task status is `ready_for_*`
2. Determine target status: `ready_for_X` → `in_X`
3. Validate transition via workflow config
4. UPDATE tasks SET status='in_X', assigned_agent='<agent>'
5. INSERT INTO task_history (transition record)
6. INSERT INTO task_sessions (start work session)
7. COMMIT

**Postconditions**:
- Task status is `in_X`
- Task has assigned agent
- Active work session exists
- History record created

### Output

**Human-Readable**:
```
Task T-E07-F20-001 claimed by backend
Status: in_development
Started: 2026-01-11 10:30:00
Next phase: code_review (after finish, tech-lead or code-reviewer will claim)
```

**JSON**:
```json
{
  "task_key": "T-E07-F20-001",
  "previous_status": "ready_for_development",
  "new_status": "in_development",
  "agent": "backend",
  "session": {
    "id": 123,
    "started_at": "2026-01-11T10:30:00Z"
  },
  "next_phase": {
    "phase": "code_review",
    "status": "ready_for_code_review",
    "agent_types": ["tech-lead", "code-reviewer"]
  }
}
```

### Error Cases

**Task Not Found**:
```
Error: Task T-E07-F20-999 not found
Exit Code: 1
```

**Already Claimed**:
```
Error: Cannot claim task T-E07-F20-001
Task is already in 'in_development' status (claimed by backend at 2026-01-11 09:00)
Use 'shark task finish' to complete work or 'shark task reject' to send back
Exit Code: 3
```

**Invalid Workflow Transition**:
```
Error: Cannot transition from 'completed' to 'in_completed'
Task is in terminal status. Use 'shark task reopen' if rework needed.
Exit Code: 3
```

**Agent Type Mismatch** (warning only):
```
Warning: Agent 'frontend' claiming 'ready_for_development' task
Expected agent types: developer, ai-coder
Proceeding with claim (use --force to suppress warning)
Task T-E07-F20-001 claimed successfully
```

### Examples

```bash
# Developer claims task
shark task claim T-E07-F20-001 --agent=backend

# AI orchestrator claims with JSON output
shark task claim T-E07-F20-001 --agent=ai-coder --json

# Claim with auto-detected agent (from $USER)
shark task claim T-E07-F20-001

# Claim using short key format
shark task claim E07-F20-001 --agent=developer

# Claim using slugged key
shark task claim T-E07-F20-001-implement-auth --agent=backend
```

### Database Changes

**tasks table**:
```sql
UPDATE tasks
SET status = 'in_development',
    assigned_agent = 'backend',
    started_at = NOW(),
    updated_at = NOW()
WHERE key = 'T-E07-F20-001'
```

**task_history table**:
```sql
INSERT INTO task_history
  (task_id, from_status, to_status, changed_by, changed_at, notes)
VALUES
  (123, 'ready_for_development', 'in_development', 'backend', NOW(), 'Task claimed')
```

**task_sessions table**:
```sql
INSERT INTO task_sessions
  (task_id, agent_id, started_at)
VALUES
  (123, 'backend', NOW())
```

---

## Command: shark task finish

### Purpose

Completes current phase and advances task to next workflow stage. Ends active work session.

### Syntax

```bash
shark task finish <task-key> [--notes="..."] [--json]
```

### Arguments

| Argument | Required | Type | Description |
|----------|----------|------|-------------|
| task-key | Yes | string | Task identifier |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --notes | string | "" | Completion notes/summary |
| --json | bool | false | Output as JSON |

### Behavior

**Preconditions**:
- Task exists in database
- Task status matches `in_*` pattern
- Workflow defines valid next status

**Transition Logic**:
1. Read current status (e.g., `in_development`)
2. Get valid transitions from workflow config
3. Select next status using priority:
   - Priority 1: First `ready_for_*` status (standard forward)
   - Priority 2: Terminal status (`completed`, `cancelled`)
   - Priority 3: First valid transition
4. Validate selected transition
5. Execute transition

**Actions** (atomic transaction):
1. Validate task status is `in_*`
2. Determine next status via workflow
3. UPDATE tasks SET status='<next>', completed_at=NOW()
4. INSERT INTO task_history (completion record)
5. UPDATE task_sessions SET ended_at=NOW(), outcome='completed'
6. COMMIT

**Postconditions**:
- Task status is next phase (or terminal)
- Work session ended
- Completion notes recorded
- History record created

### Output

**Human-Readable**:
```
Task T-E07-F20-001 completed
Status: in_development → ready_for_code_review
Work session: 2h 30m (2026-01-11 10:30 - 13:00)
Next phase: code_review (tech-lead or code-reviewer can claim)
```

**JSON**:
```json
{
  "task_key": "T-E07-F20-001",
  "previous_status": "in_development",
  "new_status": "ready_for_code_review",
  "notes": "API implementation complete",
  "session": {
    "id": 123,
    "started_at": "2026-01-11T10:30:00Z",
    "ended_at": "2026-01-11T13:00:00Z",
    "duration_minutes": 150,
    "outcome": "completed"
  },
  "next_phase": {
    "phase": "code_review",
    "status": "ready_for_code_review",
    "agent_types": ["tech-lead", "code-reviewer"]
  }
}
```

### Multiple Next States (Interactive)

If workflow has multiple valid forward transitions:

```bash
$ shark task finish T-E07-F20-001

Task T-E07-F20-001 is in: in_development

Multiple next states available:
  1. ready_for_code_review (standard flow) [RECOMMENDED]
  2. ready_for_qa (skip code review)
  3. ready_for_refinement (needs rework)

Selection [1]: _
```

**Non-Interactive Mode** (--json or --auto):
- Always selects first `ready_for_*` status
- Logs decision to history

### Error Cases

**Not In Progress**:
```
Error: Cannot finish task in 'ready_for_development' status
Use 'shark task claim' to start work first
Exit Code: 3
```

**Terminal Status**:
```
Error: Cannot finish task in 'completed' status
Task is already in terminal state
Exit Code: 3
```

**No Valid Transitions**:
```
Error: Workflow error - 'in_qa' has no outgoing transitions
This is a configuration error. Fix .sharkconfig.json or contact admin.
Exit Code: 2
```

### Examples

```bash
# Finish task with notes
shark task finish T-E07-F20-001 --notes="API endpoints implemented and tested"

# Finish with JSON output (AI orchestrator)
shark task finish T-E07-F20-001 --json

# Finish with automatic selection (non-interactive)
shark task finish T-E07-F20-001 --auto

# Finish at end of QA phase (to approval)
shark task finish T-E07-F20-005 --notes="All tests passed"
# Status: in_qa → ready_for_approval
```

### Database Changes

**tasks table**:
```sql
UPDATE tasks
SET status = 'ready_for_code_review',
    completed_at = NOW(),
    updated_at = NOW()
WHERE key = 'T-E07-F20-001'
```

**task_history table**:
```sql
INSERT INTO task_history
  (task_id, from_status, to_status, changed_by, changed_at, notes)
VALUES
  (123, 'in_development', 'ready_for_code_review', 'backend', NOW(),
   'API endpoints implemented and tested')
```

**task_sessions table**:
```sql
UPDATE task_sessions
SET ended_at = NOW(),
    outcome = 'completed',
    notes = 'API endpoints implemented and tested'
WHERE task_id = 123
  AND ended_at IS NULL
```

---

## Command: shark task reject

### Purpose

Sends task backward in workflow for rework. Ends current work session with rejection outcome.

### Syntax

```bash
shark task reject <task-key> --reason="..." [--to=<status>] [--json]
```

### Arguments

| Argument | Required | Type | Description |
|----------|----------|------|-------------|
| task-key | Yes | string | Task identifier |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --reason | string | (required) | Rejection reason (why task isn't ready) |
| --to | string | (auto) | Target status (if multiple backward paths exist) |
| --json | bool | false | Output as JSON |

### Behavior

**Preconditions**:
- Task exists in database
- Task has valid backward transition in workflow
- Reason provided (required)

**Transition Logic**:
1. Read current status (e.g., `in_development`)
2. If `--to` specified: validate it's a valid backward transition
3. If `--to` not specified: auto-determine using priority:
   - Priority 1: Refinement phase (`in_refinement`, `ready_for_refinement`)
   - Priority 2: Earlier phase in workflow order
   - Priority 3: Any valid backward transition
4. Validate selected transition
5. Execute transition

**Actions** (atomic transaction):
1. Validate reason is provided
2. Determine target status (--to or auto)
3. Validate transition is backward (earlier phase)
4. UPDATE tasks SET status='<target>'
5. INSERT INTO task_history (rejection record with reason)
6. INSERT INTO task_notes (rejection note)
7. UPDATE task_sessions SET ended_at=NOW(), outcome='rejected'
8. COMMIT

**Postconditions**:
- Task status is earlier phase
- Rejection reason recorded
- Work session ended with 'rejected' outcome
- Visible in task history

### Output

**Human-Readable**:
```
Task T-E07-F20-001 rejected
Status: in_development → in_refinement
Reason: Acceptance criteria incomplete
Work session: 1h 15m (2026-01-11 10:30 - 11:45)
Task returned to: refinement phase (business-analyst or architect can claim)
```

**JSON**:
```json
{
  "task_key": "T-E07-F20-001",
  "previous_status": "in_development",
  "new_status": "in_refinement",
  "reason": "Acceptance criteria incomplete",
  "session": {
    "id": 123,
    "started_at": "2026-01-11T10:30:00Z",
    "ended_at": "2026-01-11T11:45:00Z",
    "duration_minutes": 75,
    "outcome": "rejected"
  },
  "next_phase": {
    "phase": "planning",
    "status": "in_refinement",
    "agent_types": ["business-analyst", "architect"]
  }
}
```

### Error Cases

**Missing Reason**:
```
Error: --reason is required when rejecting a task
Provide an explanation of why the task cannot proceed
Example: shark task reject T-E07-F20-001 --reason="Missing database schema"
Exit Code: 1
```

**Invalid Target Status**:
```
Error: Cannot reject from 'in_development' to 'ready_for_qa'
Rejection must move to earlier workflow phase
Current phase: development
Target phase: qa (later than current)
Valid backward transitions: in_refinement, ready_for_refinement
Exit Code: 3
```

**No Backward Transitions**:
```
Error: No backward transition available from 'draft'
Task is in initial status. Use 'shark task delete' if not needed.
Exit Code: 3
```

### Examples

```bash
# Reject with auto-determined target (to refinement)
shark task reject T-E07-F20-001 --reason="Acceptance criteria incomplete"

# Reject to specific status
shark task reject T-E07-F20-001 --reason="Missing DB schema" --to=in_refinement

# Code reviewer rejects back to development
shark task reject T-E07-F20-001 --reason="Unit tests missing" --to=in_development

# QA rejects back to development
shark task reject T-E07-F20-005 --reason="API returns 500 error" --to=in_development

# JSON output
shark task reject T-E07-F20-001 --reason="Requirements unclear" --json
```

### Database Changes

**tasks table**:
```sql
UPDATE tasks
SET status = 'in_refinement',
    updated_at = NOW()
WHERE key = 'T-E07-F20-001'
```

**task_history table**:
```sql
INSERT INTO task_history
  (task_id, from_status, to_status, changed_by, changed_at, notes, rejection_reason)
VALUES
  (123, 'in_development', 'in_refinement', 'backend', NOW(),
   'Task rejected', 'Acceptance criteria incomplete')
```

**task_notes table** (or notes column):
```sql
INSERT INTO task_notes
  (task_id, type, content, created_by, created_at)
VALUES
  (123, 'rejection', 'Acceptance criteria incomplete - missing edge case handling',
   'backend', NOW())
```

**task_sessions table**:
```sql
UPDATE task_sessions
SET ended_at = NOW(),
    outcome = 'rejected',
    notes = 'Acceptance criteria incomplete'
WHERE task_id = 123
  AND ended_at IS NULL
```

---

## Common Patterns

### Transaction Safety

All commands use same transaction pattern:

```go
func executeCommand(ctx context.Context, db *DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback() // Rollback if commit not reached

    // 1. Validate preconditions
    // 2. Update task status
    // 3. Record history
    // 4. Update/create session

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }
    return nil
}
```

### Workflow Integration

All commands follow same workflow pattern:

```go
// 1. Load workflow service
workflow := workflow.NewService(projectRoot)

// 2. Get current task
task, err := taskRepo.GetByKey(ctx, taskKey)

// 3. Determine target status (command-specific logic)
targetStatus, err := workflow.GetNextPhaseStatus(task.Status) // finish
// or
targetStatus, err := workflow.GetClaimStatus(task.Status) // claim
// or
targetStatus := flagValue // reject with explicit --to

// 4. Validate transition
if err := workflow.ValidateTransition(task.Status, targetStatus, "finish"); err != nil {
    return err
}

// 5. Execute transition
```

### Error Message Guidelines

**User-Actionable Errors**:
- State what went wrong
- Explain why it failed
- Suggest how to fix
- Show relevant commands

**Example**:
```
Error: Cannot claim task in 'in_development' status
Task is already claimed by backend (since 2026-01-11 09:00)

Suggested actions:
  • If you are the owner: shark task finish T-E07-F20-001
  • If wrong agent claimed: shark task reject T-E07-F20-001 --reason="Reassign"
  • To force claim: shark task claim T-E07-F20-001 --force (admin only)
```

---

## Implementation Checklist

### Per Command

- [ ] Create command file (`internal/cli/commands/task_<cmd>.go`)
- [ ] Implement command handler with Cobra
- [ ] Add argument parsing and validation
- [ ] Integrate workflow service
- [ ] Implement transaction logic
- [ ] Add JSON and human-readable output
- [ ] Write unit tests (mocked repositories)
- [ ] Write integration tests (real database)
- [ ] Update help text
- [ ] Add to command categorization

### Repository Methods

- [ ] Add `ClaimTask(ctx, taskID, agentID) error`
- [ ] Add `FinishTask(ctx, taskID, notes) error`
- [ ] Add `RejectTask(ctx, taskID, targetStatus, reason) error`
- [ ] Ensure transaction support
- [ ] Add workflow validation integration

---

## References

- [System Architecture](./system-architecture.md)
- [Workflow Config Reader](./workflow-config-reader.md)
- [Session Tracking](./session-tracking.md)
- [Transition Validation](./transition-validation.md)
- [Epic Requirements](../requirements.md) - REQ-F-001, REQ-F-002, REQ-F-003
