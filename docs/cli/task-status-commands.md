# Task Status Commands

## Overview

The Shark CLI provides commands to manage task status through their lifecycle. All status update commands support the `--force` flag for administrative overrides.

## Basic Workflow

The normal task workflow follows this progression:

```
todo → in_progress → ready_for_review → completed → archived
       ↓               ↓
    blocked         (reopen)
       ↓
     todo
```

## Commands

### shark task start

Start working on a task (transition from `todo` to `in_progress`).

**Usage:**
```bash
shark task start <task-key> [flags]
```

**Flags:**
- `--agent string`: Agent identifier (defaults to $USER env var)
- `--force`: Force status change bypassing validation (use with caution)

**Examples:**
```bash
# Normal start (requires task in 'todo' status)
shark task start T-E07-F01-001

# Force start from any status
shark task start T-E07-F01-001 --force

# Start with specific agent
shark task start T-E07-F01-001 --agent="alice"
```

**Normal Behavior:**
- Task must be in `todo` status
- Sets `status` to `in_progress`
- Sets `started_at` timestamp
- Records status change in history

**With --force:**
- Accepts task in ANY status
- Transitions to `in_progress`
- Logs warning
- Records `forced=true` in history

---

### shark task complete

Mark a task as ready for review (transition from `in_progress` to `ready_for_review`).

**Usage:**
```bash
shark task complete <task-key> [flags]
```

**Flags:**
- `--agent string`: Agent identifier (defaults to $USER env var)
- `--notes, -n string`: Completion notes
- `--force`: Force status change bypassing validation

**Examples:**
```bash
# Normal completion (requires task in 'in_progress' status)
shark task complete T-E07-F01-001

# With notes
shark task complete T-E07-F01-001 --notes="All tests passing"

# Force complete from any status
shark task complete T-E07-F01-001 --force
```

**Normal Behavior:**
- Task must be in `in_progress` status
- Sets `status` to `ready_for_review`
- Records status change with optional notes

**With --force:**
- Accepts task in ANY status
- Transitions to `ready_for_review`
- Logs warning
- Records `forced=true` in history

---

### shark task approve

Approve a task for completion (transition from `ready_for_review` to `completed`).

**Usage:**
```bash
shark task approve <task-key> [flags]
```

**Flags:**
- `--agent string`: Agent identifier (defaults to $USER env var)
- `--notes, -n string`: Approval notes
- `--force`: Force status change bypassing validation

**Examples:**
```bash
# Normal approval (requires task in 'ready_for_review' status)
shark task approve T-E07-F01-001

# With approval notes
shark task approve T-E07-F01-001 --notes="Code review passed"

# Force approve from any status
shark task approve T-E07-F01-001 --force
```

**Normal Behavior:**
- Task must be in `ready_for_review` status
- Sets `status` to `completed`
- Sets `completed_at` timestamp
- Records status change with optional notes

**With --force:**
- Accepts task in ANY status
- Transitions to `completed`
- Logs warning
- Records `forced=true` in history

---

### shark task block

Block a task with a required reason (transition to `blocked` status).

**Usage:**
```bash
shark task block <task-key> --reason="<reason>" [flags]
```

**Flags:**
- `--reason, -r string`: Reason for blocking (required)
- `--agent string`: Agent identifier (defaults to $USER env var)
- `--force`: Force status change bypassing validation

**Examples:**
```bash
# Normal block (requires task in 'todo' or 'in_progress' status)
shark task block T-E07-F01-001 --reason="Waiting for API design"

# With agent identifier
shark task block T-E07-F01-001 --reason="Blocked by dependency" --agent="bob"

# Force block from any status
shark task block T-E07-F01-001 --reason="Emergency freeze" --force
```

**Normal Behavior:**
- Task must be in `todo` or `in_progress` status
- Sets `status` to `blocked`
- Sets `blocked_at` timestamp
- Sets `blocked_reason` field
- Records reason in history

**With --force:**
- Accepts task in ANY status
- Transitions to `blocked`
- Sets blocked_reason
- Logs warning
- Records `forced=true` in history

---

### shark task unblock

Unblock a task and return it to the queue (transition from `blocked` to `todo`).

**Usage:**
```bash
shark task unblock <task-key> [flags]
```

**Flags:**
- `--agent string`: Agent identifier (defaults to $USER env var)
- `--force`: Force status change bypassing validation

**Examples:**
```bash
# Normal unblock (requires task in 'blocked' status)
shark task unblock T-E07-F01-001

# Force unblock from any status
shark task unblock T-E07-F01-001 --force
```

**Normal Behavior:**
- Task must be in `blocked` status
- Sets `status` to `todo`
- Clears `blocked_at` timestamp
- Clears `blocked_reason` field
- Records status change in history

**With --force:**
- Accepts task in ANY status
- Transitions to `todo`
- Clears blocked fields
- Logs warning
- Records `forced=true` in history

---

### shark task reopen

Reopen a task for additional work (transition from `ready_for_review` back to `in_progress`).

**Usage:**
```bash
shark task reopen <task-key> [flags]
```

**Flags:**
- `--agent string`: Agent identifier (defaults to $USER env var)
- `--notes, -n string`: Rework notes
- `--force`: Force status change bypassing validation

**Examples:**
```bash
# Normal reopen (requires task in 'ready_for_review' status)
shark task reopen T-E07-F01-001

# With rework notes
shark task reopen T-E07-F01-001 --notes="Need to fix edge case"

# Force reopen from any status
shark task reopen T-E07-F01-001 --force
```

**Normal Behavior:**
- Task must be in `ready_for_review` status
- Sets `status` to `in_progress`
- Clears `completed_at` timestamp (if set)
- Records status change with optional notes

**With --force:**
- Accepts task in ANY status
- Transitions to `in_progress`
- Logs warning
- Records `forced=true` in history

---

## Status Transition Rules

### Without --force

| From Status        | Allowed Transitions                          |
|-------------------|---------------------------------------------|
| todo              | in_progress, blocked                        |
| in_progress       | ready_for_review, blocked                   |
| blocked           | todo                                        |
| ready_for_review  | completed, in_progress (reopen)             |
| completed         | archived                                    |
| archived          | (no transitions)                            |

### With --force

| From Status     | Allowed Transitions |
|-----------------|---------------------|
| ANY             | ANY (except archived → other statuses should be rare) |

**Note:** The `--force` flag bypasses transition validation but still requires the target status to be a valid status enum value.

## Error Handling

### Normal Validation Errors

```bash
$ shark task approve T-E07-F01-001
Error: Invalid state transition from in_progress to completed. Task must be in 'ready_for_review' status.
Use --force to bypass this validation
```

### Invalid Status

```bash
$ shark task start T-E07-F01-001 --force
Error: invalid status: invalid_status
```

### Task Not Found

```bash
$ shark task start T-INVALID-001
Error: Task not found: T-INVALID-001
```

## Best Practices

### When to Use Each Command

1. **start**: When beginning work on a task from the backlog
2. **complete**: When you've finished implementation and it's ready for review
3. **approve**: After reviewing a task and confirming it meets requirements
4. **block**: When you can't proceed due to external dependency or issue
5. **unblock**: When the blocker is resolved and work can resume
6. **reopen**: When review identifies issues that need to be fixed

### Agent Identifier

The `--agent` flag defaults to the `$USER` environment variable. Specify it explicitly when:
- Running commands on behalf of another user
- In automation where `$USER` might not be set correctly
- Tracking which automated process made the change

### Using Notes

Always provide notes with `--notes` when:
- Blocking a task (explain what's blocking it)
- Reopening a task (explain what needs to be fixed)
- Approving a task (document what was reviewed)
- Any change that might not be obvious to others

### Using --force

See [Force Status Updates](../features/force-status-updates.md) for comprehensive guidance on when and how to use the `--force` flag.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0    | Success |
| 1    | General error (task not found, database error, etc.) |
| 2    | Database error |
| 3    | Invalid status transition |

## Related Commands

- [`shark task list`](./task-list.md) - List and filter tasks
- [`shark task get`](./task-get.md) - View task details
- [`shark task create`](./task-create.md) - Create new tasks
- [`shark task delete`](./task-delete.md) - Delete tasks

## See Also

- [Force Status Updates](../features/force-status-updates.md)
- [Task Lifecycle](../concepts/task-lifecycle.md)
- [Status Transitions](../concepts/status-transitions.md)
