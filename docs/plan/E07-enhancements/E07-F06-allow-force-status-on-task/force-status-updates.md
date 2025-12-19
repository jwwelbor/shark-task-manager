# Force Status Updates (E07-F06)

## Overview

The `--force` flag allows administrators to bypass normal status transition validation when updating task statuses. This is an administrative override for exceptional cases where the normal workflow restrictions need to be bypassed.

## When to Use --force

Use `--force` when:
- Correcting data entry errors (e.g., task marked as wrong status)
- Recovering from workflow issues (e.g., task stuck in wrong state)
- Administrative cleanup operations
- Special circumstances that don't fit normal workflows

**WARNING**: Force updates bypass safety checks and can create inconsistent state if misused. Use with caution and only when necessary.

## Commands with --force Support

All task status update commands support the `--force` flag:

### shark task start
```bash
# Normal usage (requires task to be in 'todo' status)
shark task start T-E07-F06-001

# Force start from any status
shark task start T-E07-F06-001 --force
```

### shark task complete
```bash
# Normal usage (requires task to be in 'in_progress' status)
shark task complete T-E07-F06-001

# Force complete from any status
shark task complete T-E07-F06-001 --force
```

### shark task approve
```bash
# Normal usage (requires task to be in 'ready_for_review' status)
shark task approve T-E07-F06-001

# Force approve from any status
shark task approve T-E07-F06-001 --force
```

### shark task block
```bash
# Normal usage (requires task to be in 'todo' or 'in_progress' status)
shark task block T-E07-F06-001 --reason="Waiting for API"

# Force block from any status
shark task block T-E07-F06-001 --reason="Waiting for API" --force
```

### shark task unblock
```bash
# Normal usage (requires task to be in 'blocked' status)
shark task unblock T-E07-F06-001

# Force unblock from any status
shark task unblock T-E07-F06-001 --force
```

### shark task reopen
```bash
# Normal usage (requires task to be in 'ready_for_review' status)
shark task reopen T-E07-F06-001 --notes="Needs more work"

# Force reopen from any status
shark task reopen T-E07-F06-001 --notes="Needs more work" --force
```

## Normal Status Transitions

Without `--force`, only these transitions are allowed:

```
todo → in_progress
todo → blocked
in_progress → ready_for_review
in_progress → blocked
blocked → todo
ready_for_review → completed
ready_for_review → in_progress (reopen)
completed → archived
```

## Force Status Behavior

With `--force`:
- **Any status → Any status** (except must be valid status enum)
- Validation is bypassed
- Warning logged to console
- Audit trail records `forced=true` in history

## Audit Trail

All forced status updates are tracked in the task_history table:

```sql
SELECT * FROM task_history WHERE forced = 1;
```

The history record includes:
- `task_id`: Which task was updated
- `old_status`: Status before update
- `new_status`: Status after update
- `agent`: Who performed the update
- `notes`: Optional notes
- `forced`: TRUE when --force was used
- `timestamp`: When the update occurred

## Examples

### Example 1: Correcting an Accidental Approval

Task was accidentally approved but needs more work:

```bash
# Check current status
shark task get T-E07-F06-001

# Force reopen from completed status (normally not allowed)
shark task reopen T-E07-F06-001 --notes="Accidentally approved, needs fixes" --force
```

### Example 2: Administrative Cleanup

Multiple tasks need to be reset to todo:

```bash
# Force all these tasks back to todo
shark task unblock T-E07-F06-001 --force
shark task unblock T-E07-F06-002 --force
shark task unblock T-E07-F06-003 --force
```

### Example 3: Emergency Bypass

Critical task needs to skip review:

```bash
# Force directly from in_progress to completed (skipping ready_for_review)
shark task approve T-E07-F06-001 --notes="Emergency hotfix" --force
```

## Safety Considerations

### When NOT to Use --force

- **Normal workflow operations**: If the status transition makes sense, follow the normal workflow
- **Uncertain about state**: If you're not sure what status a task should be in, investigate first
- **Automated scripts**: Don't use --force in automation unless absolutely necessary
- **Learning/training**: Don't use --force to work around learning the proper workflow

### Best Practices

1. **Always provide notes**: Use `--notes` to explain WHY you're forcing the status change
2. **Document in tickets**: Record force operations in your ticketing system
3. **Review audit trail**: Periodically review forced operations to identify process issues
4. **Communicate with team**: Let team members know when you force status changes on shared tasks
5. **Fix root causes**: If you find yourself using --force frequently, fix the underlying process issue

## Implementation Details

### Validation Logic

```go
if force {
    // Skip transition validation, only check enum
    if !isValidStatusEnum(newStatus) {
        return ErrInvalidStatus
    }
    // Log warning
    log.Warn("Forced status update", "from", currentStatus, "to", newStatus)
} else {
    if !isValidTransition(currentStatus, newStatus) {
        return ErrInvalidTransition
    }
}
```

### Database Schema

The `forced` column was added to the `task_history` table:

```sql
ALTER TABLE task_history ADD COLUMN forced BOOLEAN DEFAULT FALSE;
```

### Repository Methods

All status update methods have a forced variant:
- `UpdateStatus()` → `UpdateStatusForced(force bool)`
- `BlockTask()` → `BlockTaskForced(force bool)`
- `UnblockTask()` → `UnblockTaskForced(force bool)`
- `ReopenTask()` → `ReopenTaskForced(force bool)`

## Testing

To test force functionality:

```bash
# Create a test task
shark task create "Test Force" --epic=E07 --feature=F06

# Start it normally
shark task start T-E07-F06-TEST

# Try to approve directly (should fail)
shark task approve T-E07-F06-TEST
# Error: Invalid state transition from in_progress to completed

# Force approve (should succeed)
shark task approve T-E07-F06-TEST --force
# Success with warning

# Check history
sqlite3 shark-tasks.db "SELECT * FROM task_history WHERE task_id = (SELECT id FROM tasks WHERE key = 'T-E07-F06-TEST');"
```

## Related

- [Status Transitions](status-transitions.md)
- [Task Lifecycle](task-lifecycle.md)
- [Audit Trail](audit-trail.md)

## Changelog

- **2025-12-18**: Initial implementation (E07-F06)
  - Added --force flag to all status update commands
  - Implemented validation bypass logic
  - Added audit trail with forced column
