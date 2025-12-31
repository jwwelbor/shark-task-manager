# Task History Foreign Key Bug Fix

## Summary
Fixed critical bug where `task_history` table had a foreign key referencing `"tasks_old"(id)` instead of `tasks(id)`, causing task status updates to fail.

## Problem
**Error When Setting Task Status:**
```
Failed to update task status: failed to create history record: no such table: main.tasks_old
```

**Root Cause:**
The migration in `internal/db/migrate.go` that removed the status CHECK constraint recreated the `tasks` table but didn't update the `task_history` table's foreign key constraint.

**Affected Schema:**
```sql
CREATE TABLE task_history (
    ...
    FOREIGN KEY (task_id) REFERENCES "tasks_old"(id) ON DELETE CASCADE
)
```

Should be:
```sql
CREATE TABLE task_history (
    ...
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
)
```

## Solution
Implemented a two-part fix:

### 1. Updated Existing Migration (`migrateTasksStatusConstraint`)
Modified the migration that removes status CHECK constraints to also recreate the `task_history` table with the correct foreign key.

**Changes in `internal/db/migrate.go`:**
- Added Step 6: Rename `task_history` to `task_history_old`
- Created new `task_history` table with correct FK: `REFERENCES tasks(id)`
- Copied all existing history records
- Recreated indexes
- Dropped old tables

### 2. Added Dedicated Migration (`migrateTaskHistoryForeignKey`)
Created a new migration function to fix databases that were already migrated (where tasks table has no status constraint but task_history has the wrong FK).

**Changes in `internal/db/db.go`:**
- Added call to `migrateTaskHistoryForeignKey()` in `runMigrations()`
- This migration checks if `task_history` references `tasks_old` and fixes it
- Idempotent - safe to run multiple times

## Tests
Created comprehensive tests to ensure the fix works:

### 1. `TestMigrateTasksStatusConstraint_FixesTaskHistoryForeignKey`
Tests that the migration correctly recreates task_history when migrating from old schema with status constraints.

### 2. `TestMigrateTaskHistoryForeignKey_AlreadyMigratedDatabase`
Tests the dedicated migration function for databases where tasks was already migrated but task_history has the wrong FK.

**Test Coverage:**
- ✅ Foreign key correctly references `tasks(id)`
- ✅ All existing history records preserved
- ✅ Can insert new history records
- ✅ Foreign key constraint enforced (rejects invalid task_id)
- ✅ Migration is idempotent (safe to run multiple times)

## Verification
Tested on real database:

```bash
# Before fix:
$ shark task set-status T-E07-F11-002 in_development --force
# ERROR: no such table: main.tasks_old

# After fix:
$ shark task set-status T-E07-F11-002 in_development --force
# SUCCESS: Task T-E07-F11-002 status updated: todo → in_development

$ shark task set-status T-E07-F11-002 ready_for_code_review --force
# SUCCESS: Task T-E07-F11-002 status updated: in_development → ready_for_code_review
```

**Database Schema After Fix:**
```sql
CREATE TABLE task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    old_status TEXT,
    new_status TEXT NOT NULL,
    agent TEXT,
    notes TEXT,
    forced BOOLEAN DEFAULT FALSE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
```

**History Records Preserved:**
All 600 existing task_history records were successfully migrated and remain intact.

## Files Modified
- `/home/jwwelbor/projects/shark-task-manager/internal/db/migrate.go`
  - Updated `migrateTasksStatusConstraint()` to also recreate task_history
  - Added `migrateTaskHistoryForeignKey()` for already-migrated databases

- `/home/jwwelbor/projects/shark-task-manager/internal/db/db.go`
  - Added call to `migrateTaskHistoryForeignKey()` in `runMigrations()`

- `/home/jwwelbor/projects/shark-task-manager/internal/db/migrate_test.go`
  - Added `TestMigrateTasksStatusConstraint_FixesTaskHistoryForeignKey()`
  - Added `TestMigrateTaskHistoryForeignKey_AlreadyMigratedDatabase()`
  - Added `containsSubstring()` helper function

## Impact
- ✅ Task status updates now work correctly
- ✅ Task history is properly recorded
- ✅ Foreign key constraints properly enforced
- ✅ All existing data preserved
- ✅ Backwards compatible (migrations run automatically)

## Test Results
```
=== RUN   TestMigrateTasksStatusConstraint_FixesTaskHistoryForeignKey
--- PASS: TestMigrateTasksStatusConstraint_FixesTaskHistoryForeignKey (0.00s)
=== RUN   TestMigrateTaskHistoryForeignKey_AlreadyMigratedDatabase
--- PASS: TestMigrateTaskHistoryForeignKey_AlreadyMigratedDatabase (0.00s)

PASS
ok  	github.com/jwwelbor/shark-task-manager/internal/db	0.222s
```

## Date
2025-12-30
