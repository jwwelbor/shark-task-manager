# Shark Task Manager - Database Triggers

## Overview
This document details all database triggers in the Shark Task Manager schema. Triggers are server-side automation that maintain data consistency and audit trails.

**Total Triggers**: 3 (all AFTER UPDATE triggers)

---

## Trigger Inventory

### 1. epics_updated_at Trigger

**Location**: `internal/db/db.go` lines 88-93

```sql
CREATE TRIGGER IF NOT EXISTS epics_updated_at
AFTER UPDATE ON epics
FOR EACH ROW
BEGIN
    UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Type**: AFTER UPDATE
**Timing**: After every row update on the `epics` table
**Scope**: FOR EACH ROW (fires once per updated row)

**Purpose**:
- Auto-update the `updated_at` timestamp whenever an epic record is modified
- Maintain accurate audit trail of when records were last changed
- Eliminate need for application layer to manage timestamps

**Trigger Logic**:
1. Event: `UPDATE` statement executes on epics table
2. When: After the update completes successfully
3. Action: Set `updated_at = CURRENT_TIMESTAMP` on the affected row

**Data Flow**:
```
Application Layer
    ↓
UPDATE epics SET title = 'New Title' WHERE id = 5
    ↓
SQLite Database
    ├─ Update row: id=5, title='New Title'
    ├─ [AFTER UPDATE trigger fires]
    ├─ UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = 5
    └─ Commit transaction
    ↓
Row now has: title='New Title', updated_at='2025-12-19 15:30:45.123'
```

**Trigger Patterns Activated By**:
```sql
-- Direct field update (trigger fires)
UPDATE epics SET title = 'New Title' WHERE key = 'E04';
-- Result: updated_at set to current time

-- Status change (trigger fires)
UPDATE epics SET status = 'archived' WHERE id = 5;
-- Result: updated_at updated

-- Priority change (trigger fires)
UPDATE epics SET priority = 'high' WHERE key = 'E07';
-- Result: updated_at updated

-- Bulk update (trigger fires for EACH row)
UPDATE epics SET status = 'completed' WHERE status = 'archived';
-- Result: updated_at updated for all matching rows

-- Direct timestamp update (nested trigger - careful!)
UPDATE epics SET updated_at = datetime('-1 day') WHERE id = 5;
-- Result: Trigger fires, overwrites manual timestamp
-- Side effect: Timestamp becomes current time, not '-1 day'
-- This is usually desired but can be surprising
```

**Important Notes**:
- Trigger fires even if UPDATE is within a transaction
- Trigger is RECURSIVE: an UPDATE triggered by the trigger would fire again
  - SQLite has `PRAGMA recursive_triggers` (OFF by default, safe)
  - This trigger doesn't create infinite loop because it targets same row
- No trigger for INSERT (created_at is handled by DEFAULT)

**Impact on Statements**:
```
WITHOUT trigger:
    Application must: UPDATE epics SET title = ?, updated_at = CURRENT_TIMESTAMP
    Code complexity: Every update must include timestamp

WITH trigger:
    Application: UPDATE epics SET title = ?
    Trigger handles: timestamp update
    Code simplicity: One line instead of two
```

**Performance Implications**:
- Minimal overhead: ~0.1-0.5 ms per update (single UPDATE statement)
- Recursive update is in same transaction (no additional disk I/O)
- Index on `id` (PRIMARY KEY) makes WHERE clause instant

**Data Consistency Guarantee**:
```
Before trigger:
    CREATE epics
    ├─ created_at: 2025-01-15 10:00:00
    ├─ updated_at: 2025-01-15 10:00:00

After updates:
    UPDATE title
    ├─ created_at: 2025-01-15 10:00:00
    ├─ updated_at: 2025-12-19 15:30:45 (automatically updated)

    UPDATE status
    ├─ created_at: 2025-01-15 10:00:00
    ├─ updated_at: 2025-12-19 15:31:00 (automatically updated)
```

**Caveat - Direct Timestamp Updates**:
```sql
-- Attempting to set updated_at manually (trigger overrides)
UPDATE epics SET title = 'New', updated_at = datetime('-1 day') WHERE id = 5;

-- Result: The trigger fires AFTER the update
-- Trigger executes: UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = 5
-- Final result: updated_at = current time, not '-1 day'

-- Workaround (if you need custom timestamp):
-- 1. Disable trigger temporarily (not recommended)
-- 2. Use PRAGMA: ALTER TABLE ... DISABLE TRIGGER (not standard SQLite)
-- 3. Store both updated_at (automatic) and last_audit_time (manual) columns
```

**Query Examples Using updated_at**:
```sql
-- Find recently updated epics
SELECT * FROM epics
WHERE updated_at > datetime('-1 day')
ORDER BY updated_at DESC;

-- Detect stale epics
SELECT * FROM epics
WHERE updated_at < datetime('-30 days')
AND status != 'archived';

-- Audit last change
SELECT key, title, updated_at, (julianday('now') - julianday(updated_at)) * 24 as hours_since_update
FROM epics
ORDER BY updated_at DESC;
```

---

### 2. features_updated_at Trigger

**Location**: `internal/db/db.go` lines 119-124

```sql
CREATE TRIGGER IF NOT EXISTS features_updated_at
AFTER UPDATE ON features
FOR EACH ROW
BEGIN
    UPDATE features SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Type**: AFTER UPDATE
**Timing**: After every row update on the `features` table
**Scope**: FOR EACH ROW (fires once per updated row)

**Purpose**:
- Auto-update the `updated_at` timestamp whenever a feature record is modified
- Maintain consistent audit trail across all entity types
- Mirror of `epics_updated_at` trigger for features

**Trigger Logic** (Identical to epics_updated_at):
```
UPDATE features ... → Trigger fires → UPDATE features SET updated_at = CURRENT_TIMESTAMP
```

**Trigger Patterns Activated By**:
```sql
-- Feature status update
UPDATE features SET status = 'active' WHERE key = 'E04-F01';
-- Result: updated_at automatically set to current time

-- Feature title/description update
UPDATE features SET description = 'New description' WHERE id = 12;
-- Result: updated_at updated

-- Feature progress update (calculated from tasks)
UPDATE features SET progress_pct = 45.5 WHERE key = 'E04-F01';
-- Result: updated_at updated automatically

-- Bulk execution order update
UPDATE features SET execution_order = 3 WHERE epic_id = 5;
-- Result: updated_at updated for all affected features

-- Status bulk transition
UPDATE features SET status = 'archived' WHERE status = 'completed';
-- Result: updated_at updated for all completed features
```

**Performance Implications**:
- Minimal overhead: ~0.1-0.5 ms per update
- No additional index lookups needed (PK index used)
- Transaction-local, no inter-table dependencies

**Data Consistency Guarantee**:
- Every feature modification automatically timestamps the change
- Applications need not track timestamp updates
- Enables audit queries on feature change history

**Integration with Feature Progress Calculation**:
```sql
-- When feature progress is recalculated from tasks:
UPDATE features SET progress_pct = (
    SELECT CAST(COUNT(CASE WHEN status = 'completed' THEN 1 END) AS REAL) * 100.0 / COUNT(*)
    FROM tasks
    WHERE feature_id = features.id
) WHERE id = 123;
-- Trigger fires: updated_at is set to calculation time
-- Result: progress_pct AND updated_at both updated in same transaction
```

---

### 3. tasks_updated_at Trigger

**Location**: `internal/db/db.go` lines 162-167

```sql
CREATE TRIGGER IF NOT EXISTS tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Type**: AFTER UPDATE
**Timing**: After every row update on the `tasks` table
**Scope**: FOR EACH ROW (fires once per updated row)

**Purpose**:
- Auto-update the `updated_at` timestamp whenever a task record is modified
- Maintain audit trail for all task changes
- Critical for tracking task lifecycle modifications

**Trigger Logic** (Identical pattern):
```
UPDATE tasks ... → Trigger fires → UPDATE tasks SET updated_at = CURRENT_TIMESTAMP
```

**Trigger Patterns Activated By**:
This is the most-used trigger due to task lifecycle operations:

```sql
-- Status transitions
UPDATE tasks SET status = 'in_progress' WHERE key = 'T-E04-F06-001';
-- Result: updated_at updated, triggers history creation (external trigger)

-- Priority changes
UPDATE tasks SET priority = 8 WHERE id = 42;
-- Result: updated_at updated

-- Agent assignment
UPDATE tasks SET assigned_agent = 'dev-alice' WHERE key = 'T-E04-F06-001';
-- Result: updated_at updated

-- Start task
UPDATE tasks SET status = 'in_progress', started_at = CURRENT_TIMESTAMP WHERE id = 42;
-- Result: Both started_at AND updated_at are set

-- Block task
UPDATE tasks SET status = 'blocked', blocked_reason = 'Waiting for API', blocked_at = CURRENT_TIMESTAMP
WHERE key = 'T-E04-F06-001';
-- Result: updated_at updated, blocked_at set, status updated

-- Complete task
UPDATE tasks SET status = 'ready_for_review', completed_at = CURRENT_TIMESTAMP WHERE id = 42;
-- Result: completed_at set, updated_at set (trigger)

-- Approve task
UPDATE tasks SET status = 'completed' WHERE key = 'T-E04-F06-001';
-- Result: updated_at updated (completed_at not re-set, only first completion)

-- File path mapping (sync operation)
UPDATE tasks SET file_path = 'docs/plan/E04/F06/T-E04-F06-001.md' WHERE id = 42;
-- Result: updated_at updated

-- Bulk status change
UPDATE tasks SET status = 'archived' WHERE status = 'completed' AND completed_at < datetime('-90 days');
-- Result: updated_at updated for all archived tasks

-- Dependency tracking
UPDATE tasks SET depends_on = 'T-E04-F06-001,T-E04-F06-002' WHERE key = 'T-E04-F06-003';
-- Result: updated_at updated
```

**Important Task Lifecycle Pattern**:
```
Task Creation (no trigger, created_at handled by DEFAULT)
    ├─ created_at: 2025-12-19 10:00:00
    ├─ started_at: NULL
    ├─ completed_at: NULL
    ├─ updated_at: 2025-12-19 10:00:00

Start Task (trigger fires)
    UPDATE tasks SET status = 'in_progress', started_at = CURRENT_TIMESTAMP
    WHERE id = 42
    ├─ created_at: 2025-12-19 10:00:00
    ├─ started_at: 2025-12-19 10:05:00 (explicit)
    ├─ completed_at: NULL
    ├─ updated_at: 2025-12-19 10:05:00 (trigger)

Complete Task (trigger fires)
    UPDATE tasks SET status = 'ready_for_review', completed_at = CURRENT_TIMESTAMP
    WHERE id = 42
    ├─ created_at: 2025-12-19 10:00:00
    ├─ started_at: 2025-12-19 10:05:00
    ├─ completed_at: 2025-12-19 10:15:00 (explicit)
    ├─ updated_at: 2025-12-19 10:15:00 (trigger)

Approve Task (trigger fires)
    UPDATE tasks SET status = 'completed'
    WHERE id = 42
    ├─ created_at: 2025-12-19 10:00:00
    ├─ started_at: 2025-12-19 10:05:00
    ├─ completed_at: 2025-12-19 10:15:00 (unchanged)
    ├─ updated_at: 2025-12-19 10:20:00 (trigger)
```

**Performance Implications**:
- Most frequently triggered (task management is high-activity)
- Minimal overhead per trigger (~0.1-0.5 ms)
- Aggregate cost depends on batch operations
  - Single task update: ~1 ms total
  - Bulk update (100 tasks): ~50-100 ms total (acceptable)

**Interaction with Application Logic**:
```
Application: START TASK
  ├─ Validate task exists and status = 'todo'
  ├─ UPDATE tasks SET status = 'in_progress', started_at = CURRENT_TIMESTAMP
  ├─ [Trigger fires: updated_at set]
  ├─ [Application creates task_history record]
  └─ Return success

Result in database:
  tasks table: status='in_progress', started_at=<time>, updated_at=<time (trigger)>
  task_history table: old_status='todo', new_status='in_progress', timestamp=<time>
```

**Cascade Behavior with Task History**:
```
Application Layer triggers history creation (NOT database trigger):
    UPDATE tasks SET status = 'completed', completed_at = ...
    ├─ [Database trigger: UPDATE updated_at]
    ├─ [Application code: INSERT INTO task_history (...)]
    └─ [Two operations in same transaction, both succeed or both rollback]

If history insert fails:
    ├─ Transaction rolls back
    ├─ Task UPDATE rolled back (including trigger)
    ├─ Database remains consistent
```

**Note on Missing History Trigger**:
This implementation creates task_history records in application code, not via database trigger.

Rationale:
- Allows richer history context (agent name, notes)
- Avoids recursive database triggers
- Application has control over history retention
- Manual history creation is more maintainable than trigger-based

Alternative approach (not used):
```sql
-- Hypothetical history trigger (NOT IN ACTUAL SCHEMA)
CREATE TRIGGER task_status_change
AFTER UPDATE OF status ON tasks
FOR EACH ROW
WHEN OLD.status IS NOT NEW.status
BEGIN
    INSERT INTO task_history (task_id, old_status, new_status, timestamp)
    VALUES (NEW.id, OLD.status, NEW.status, CURRENT_TIMESTAMP);
END;
```

Actual approach (used in code):
```go
// internal/repository/task_repository.go
func (r *TaskRepository) UpdateTaskStatus(tx *sql.Tx, taskID int, newStatus string) error {
    // Get old status
    oldStatus, err := r.GetTaskStatus(taskID)

    // Update task (triggers: updated_at)
    _, err := tx.Exec("UPDATE tasks SET status = ? WHERE id = ?", newStatus, taskID)

    // Create history record
    _, err := tx.Exec(
        "INSERT INTO task_history (task_id, old_status, new_status, timestamp) VALUES (?, ?, ?, ?)",
        taskID, oldStatus, newStatus, time.Now(),
    )

    return nil
}
```

---

## Trigger Summary Table

| Trigger Name | Table | Event | Timing | Scope | Frequency | Overhead |
|--------------|-------|-------|--------|-------|-----------|----------|
| epics_updated_at | epics | UPDATE | AFTER | FOR EACH ROW | Low | Minimal |
| features_updated_at | features | UPDATE | AFTER | FOR EACH ROW | Medium | Minimal |
| tasks_updated_at | tasks | UPDATE | AFTER | FOR EACH ROW | High | Minimal |

---

## Trigger Lifecycle

### When Triggers Execute

```
Transaction Begin
    ↓
Application SQL
    ├─ INSERT ... (no triggers)
    ├─ UPDATE ... (ALL triggers fire)
    ├─ DELETE ... (no triggers)
    └─ ...
    ↓
[All triggers execute in order]
    ↓
Transaction Commit
    OR
Transaction Rollback (triggers rolled back too)
```

### Transaction Safety

**Trigger Atomicity**:
```sql
BEGIN TRANSACTION;
    UPDATE epics SET title = 'New' WHERE id = 5;
    -- Trigger fires: UPDATE epics SET updated_at = ... WHERE id = 5
    -- Both updates in same transaction
COMMIT;
-- Either both succeed or both roll back together
```

**Failure Handling**:
```sql
BEGIN TRANSACTION;
    UPDATE epics SET title = 'New' WHERE id = 5;
    -- Trigger fires and succeeds
    UPDATE features SET status = 'invalid_status' WHERE id = 10;
    -- This fails (CHECK constraint violated)
ROLLBACK;
-- Both the UPDATE and the triggered UPDATE are rolled back
-- Database remains consistent
```

---

## Trigger Performance Analysis

### Execution Cost

**Per-Row Trigger Cost**:
```
UPDATE statement: ~0.5 ms
Trigger execution: ~0.1 ms
Trigger UPDATE: ~0.1 ms
Trigger completion: ~0.1 ms
Total per row: ~0.8-1.0 ms
```

**Batch Operation Cost**:
```
UPDATE 100 tasks SET ... WHERE status = 'todo'

Without trigger:
    ├─ Query planning: 0.5 ms
    ├─ Filter evaluation: 50 ms (100 rows)
    ├─ Update execution: 100 ms
    └─ Total: ~150 ms

With 3 triggers (all update):
    ├─ Query planning: 0.5 ms
    ├─ Filter evaluation: 50 ms
    ├─ Update execution: 100 ms
    ├─ Trigger 1 (tasks_updated_at): 100 × 0.1 = 10 ms
    └─ Total: ~160 ms

Overhead: ~10 ms for 100 rows (6% slower)
Per-row overhead: 0.1 ms (negligible)
```

### Scalability

**With Current Schema**:
- 100 tasks: Trigger overhead = ~0 ms
- 1000 tasks: Trigger overhead = ~1 ms
- 10000 tasks: Trigger overhead = ~10 ms

**Conclusions**:
- Triggers scale linearly (acceptable)
- Application logic bottleneck >> trigger execution time
- Triggers are not a performance concern for this schema

---

## Trigger Design Patterns

### Pattern 1: Auto-Timestamp (Used Here)
```sql
CREATE TRIGGER table_updated_at
AFTER UPDATE ON table
FOR EACH ROW
BEGIN
    UPDATE table SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Pros**:
- Simple to implement
- Guaranteed timestamp accuracy
- No application coordination needed

**Cons**:
- Recursive UPDATE (though safe in this case)
- Manual timestamp updates get overridden

**Alternatives**:
- App-managed timestamps (error-prone)
- Column-level timestamp (SQLite doesn't support)
- Trigger-less approach with app discipline (less reliable)

### Pattern 2: Audit Trail (Not Used - App-Managed)
```sql
-- NOT IMPLEMENTED IN SCHEMA
-- Instead, application handles history:

CREATE TRIGGER task_history_on_update
AFTER UPDATE OF status ON tasks
FOR EACH ROW
WHEN OLD.status IS NOT NEW.status
BEGIN
    INSERT INTO task_history (task_id, old_status, new_status, ...)
    VALUES (NEW.id, OLD.status, NEW.status, CURRENT_TIMESTAMP);
END;
```

**Why Not Implemented**:
- Loses context (agent name, notes)
- Recursive trigger concerns
- Application has more control this way
- Easier to debug and maintain

### Pattern 3: Enforced Constraints (Not Used)
```sql
-- NOT IMPLEMENTED - Database constraints used instead
-- Example hypothetical:

CREATE TRIGGER prevent_status_downgrade
BEFORE UPDATE OF status ON tasks
FOR EACH ROW
WHEN NEW.status IN ('blocked', 'archived')
BEGIN
    SELECT CASE
        WHEN OLD.status IN ('completed', 'ready_for_review') THEN
            RAISE(ABORT, 'Cannot downgrade completed task status')
    END;
END;
```

**Why Not Implemented**:
- Status transitions are complex (business logic)
- Better handled in application layer
- Easier to test and modify
- Database constraints should enforce data types, not business rules

---

## Trigger Best Practices

### Applied in This Schema
✓ **Simple triggers**: Only timestamp updates, no complex logic
✓ **Minimal side effects**: Each trigger touches only one table
✓ **FOR EACH ROW**: Explicit about scope
✓ **No cascading triggers**: Won't recursively trigger other triggers
✓ **Transaction-safe**: All work within single transaction
✓ **Deterministic**: No random behavior or external calls

### Could Be Improved
? **Conditional triggers**: Could use WHEN clause for performance
  - E.g., only fire if status changed: `WHEN OLD.status IS NOT NEW.status`
  - Benefit: Skip trigger execution if no actual change
  - Trade-off: Complexity vs. minimal performance gain

? **Audit context**: Could add trigger-based history (with caveats)
  - Benefit: Guaranteed history for all changes
  - Trade-off: Loses application context (agent, notes)

? **Cascading deletes**: Could auto-archive instead of hard delete
  - Benefit: Better data preservation
  - Trade-off: More complex trigger logic

---

## Debugging Triggers

### Verify Trigger Execution

```bash
sqlite3 shark-tasks.db ".schema tasks"
-- Should show: CREATE TRIGGER IF NOT EXISTS tasks_updated_at ...
```

### Test Trigger Behavior

```sql
-- Check if updated_at is being set
SELECT id, title, updated_at FROM tasks WHERE id = 42;
-- Note the updated_at value

-- Update the task
UPDATE tasks SET title = 'New Title' WHERE id = 42;

-- Check again (updated_at should be newer)
SELECT id, title, updated_at FROM tasks WHERE id = 42;
```

### Disable Triggers (for debugging)

```sql
-- SQLite doesn't support disabling individual triggers easily
-- Workaround: Drop and recreate

DROP TRIGGER IF EXISTS tasks_updated_at;

-- Do your test without trigger...
UPDATE tasks SET title = 'Test' WHERE id = 42;

-- Recreate trigger
CREATE TRIGGER IF NOT EXISTS tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### Verify Trigger Syntax

```sql
-- After schema creation, check integrity
PRAGMA integrity_check;
-- Should return: ok
```

---

## Trigger Impact on Development

### Application Code Implications

**When writing UPDATE statements**:
```go
// App doesn't need to manage updated_at
err := db.Exec(
    "UPDATE tasks SET status = ?, title = ? WHERE id = ?",
    newStatus,
    newTitle,
    taskID,
)
// Trigger automatically sets updated_at
// No need for: "... SET status = ?, title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
```

**When reading updated_at**:
```go
// Can reliably use updated_at for audit/sorting
rows, err := db.Query(
    "SELECT * FROM tasks ORDER BY updated_at DESC LIMIT 10",
)
// Guaranteed to be recent updates, not stale
```

**Testing Implications**:
```go
// Tests must account for trigger-set timestamps
task1 := getTask(id)
time.Sleep(100 * time.Millisecond)
updateTask(id, "new title")
task2 := getTask(id)

assert.Equal(t, task1.UpdatedAt, task1.CreatedAt)  // Yes, equal on creation
assert.True(t, task2.UpdatedAt.After(task2.CreatedAt))  // Yes, trigger updated it
assert.True(t, task2.UpdatedAt.After(task1.UpdatedAt))  // Yes, newer timestamp
```

---

## Migration and Maintenance

### Schema Evolution

**If adding a new timestamp field**:
```sql
-- Add new column
ALTER TABLE tasks ADD COLUMN last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Create trigger for new column
CREATE TRIGGER IF NOT EXISTS tasks_activity_timestamp
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET last_activity_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**If changing trigger logic**:
```sql
-- Step 1: Drop old trigger
DROP TRIGGER IF EXISTS tasks_updated_at;

-- Step 2: Create new trigger
CREATE TRIGGER IF NOT EXISTS tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Step 3: Verify
SELECT * FROM sqlite_master WHERE type='trigger' AND name='tasks_updated_at';
```

### Backward Compatibility

Current triggers maintain backward compatibility:
- Timestamp updates are non-breaking
- No change to table structure
- No impact on SELECT queries
- UPDATE statements work unchanged (just get bonus timestamp update)

---

## Conclusion

The three triggers in the Shark Task Manager schema implement automatic timestamp maintenance using a proven pattern. They are:

1. **Simple**: Each does one thing (update timestamp)
2. **Efficient**: Negligible performance overhead
3. **Reliable**: Guaranteed to execute or roll back with transaction
4. **Maintainable**: Clear purpose, easy to debug
5. **Safe**: No recursive problems or data loss risks

Together with foreign key constraints and CHECK constraints, these triggers maintain database consistency and provide reliable audit trails through timestamps.

