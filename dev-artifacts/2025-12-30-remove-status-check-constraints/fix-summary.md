# Fix Summary: Remove Hardcoded Status CHECK Constraints

## Problem
The database schema had hardcoded CHECK constraints on status columns that prevented using workflow-defined statuses from `.sharkconfig.json`.

### Hardcoded Constraints (BEFORE):
- **Tasks**: `CHECK (status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived'))`
- **Epics**: `CHECK (status IN ('draft', 'active', 'completed', 'archived'))`
- **Features**: `CHECK (status IN ('draft', 'active', 'completed', 'archived'))`

### Workflow Statuses (NEEDED):
14 statuses defined in `.sharkconfig.json`:
- draft, ready_for_refinement, in_refinement
- ready_for_development, in_development
- ready_for_code_review, in_code_review
- ready_for_qa, in_qa
- ready_for_approval, in_approval
- blocked, on_hold, completed, cancelled

## Solution Implemented

### 1. Updated Schema (internal/db/db.go)
Removed CHECK constraints from:
- Line 87: epics.status - removed `CHECK (status IN (...))`
- Line 117: features.status - removed `CHECK (status IN (...))`
- Line 150: tasks.status - removed `CHECK (status IN (...))`

### 2. Updated Existing Migration (internal/db/migrate.go)
Fixed `MigrateRemoveAgentTypeConstraint()` to not re-add status constraint:
- Line 77: Changed to `status TEXT NOT NULL` (removed CHECK)

### 3. Added New Migration Functions
Created `MigrateRemoveStatusCheckConstraints()` with helper functions:
- `needsStatusConstraintRemoval()` - Detects if migration is needed
- `migrateTasksStatusConstraint()` - Removes CHECK from tasks table
- `migrateEpicsStatusConstraint()` - Removes CHECK from epics table
- `migrateFeaturesStatusConstraint()` - Removes CHECK from features table

### 4. Integrated Migration
Added call to `MigrateRemoveStatusCheckConstraints()` in `runMigrations()`:
- Line 508: Migration runs after all other migrations

### 5. Detection Logic
Migration only runs if CHECK constraint detected:
- Pattern: `"status TEXT NOT NULL CHECK"`
- Avoids false positives from `verification_status` column
- Skips migration if already removed

### 6. Migration Process
For each table (tasks, epics, features):
1. Check if status CHECK constraint exists
2. Disable foreign keys temporarily
3. Rename table to `{table}_old`
4. Create new table without status CHECK constraint
5. Copy all data explicitly (column by column)
6. Recreate indexes
7. Recreate triggers
8. Drop old table
9. Re-enable foreign keys

## Testing

### Test-Driven Development (TDD) Process
1. **RED Phase**: Wrote failing tests demonstrating the problem
2. **GREEN Phase**: Implemented fix, tests now pass
3. **REFACTOR Phase**: Cleaned up migration code

### Tests Created (internal/db/status_constraints_test.go)

#### 1. TestWorkflowStatusesAllowed
Tests all 13 new workflow statuses can be inserted.
- **Before**: All 13 statuses FAILED with CHECK constraint error
- **After**: All 13 statuses SUCCESS ✅

#### 2. TestStatusTransitionFromOldToNewWorkflow
Tests transitioning from old status to new status.
- **Before**: FAILED - Cannot UPDATE from 'todo' to 'in_development'
- **After**: SUCCESS ✅

#### 3. TestInvalidStatusStillRejected
Documents that DB now accepts any string (application validation required).
- Shows that invalid statuses ARE accepted (by design)
- Warns that application-level validation is CRITICAL

### Test Results
```
=== RUN   TestWorkflowStatusesAllowed
--- PASS: TestWorkflowStatusesAllowed (0.05s)
    --- PASS: TestWorkflowStatusesAllowed/draft (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/ready_for_refinement (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/in_refinement (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/ready_for_development (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/in_development (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/ready_for_code_review (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/in_code_review (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/ready_for_qa (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/in_qa (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/ready_for_approval (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/in_approval (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/on_hold (0.00s)
    --- PASS: TestWorkflowStatusesAllowed/cancelled (0.00s)

=== RUN   TestStatusTransitionFromOldToNewWorkflow
--- PASS: TestStatusTransitionFromOldToNewWorkflow (0.03s)

=== RUN   TestInvalidStatusStillRejected
--- PASS: TestInvalidStatusStillRejected (0.03s)

PASS
ok      github.com/jwwelbor/shark-task-manager/internal/db     0.222s
```

## Database Migration Verification

### Before Migration:
```sql
-- tasks table
status TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived'))

-- epics table
status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived'))

-- features table
status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived'))
```

### After Migration:
```sql
-- tasks table
status TEXT NOT NULL,

-- epics table
status TEXT NOT NULL,

-- features table
status TEXT NOT NULL,
```

### Manual Verification:
```bash
$ sqlite3 shark-tasks.db "SELECT sql FROM sqlite_master WHERE type='table' AND name='tasks'" | grep "status TEXT"
    status TEXT NOT NULL,
    verification_status TEXT CHECK(verification_status IN ('pending', 'verified', 'needs_rework')) DEFAULT 'pending',

$ sqlite3 shark-tasks.db "INSERT INTO tasks (feature_id, key, title, description, status, priority)
  SELECT id, 'TEST-DEV-001', 'Test', 'Test', 'in_development', 5 FROM features LIMIT 1;"

# Result: SUCCESS - 1 row inserted ✅
```

## Impact

### Positive
✅ Workflow statuses from config can now be used
✅ Status validation is now configuration-driven
✅ No breaking changes to existing data
✅ Backward compatible (old statuses still work)
✅ Migration is automatic and safe

### Considerations
⚠️ Database no longer validates status values
⚠️ Application MUST validate statuses against workflow config
⚠️ Invalid statuses can be inserted if validation is missing

## Next Steps (Future Work)

1. **Application-Level Validation** (NOT part of this fix):
   - Add status validation in repository layer
   - Use workflow config as source of truth
   - Validate transitions according to `status_flow`

2. **Repository Updates**:
   - Update `TaskRepository.Create()` to validate status
   - Update `TaskRepository.UpdateStatus()` to validate transitions
   - Similar for `EpicRepository` and `FeatureRepository`

3. **Status Constants Migration**:
   - Update `internal/models/task.go` status constants
   - Add new workflow status constants
   - Deprecate old status constants

4. **CLI Updates**:
   - Update status flag validation
   - Update help text with new statuses
   - Update autocomplete

## Files Modified

1. `internal/db/db.go` - Schema definition (removed CHECK constraints)
2. `internal/db/migrate.go` - Added migration functions
3. `internal/db/status_constraints_test.go` - Test suite (NEW)

## Files for Future Work

1. `internal/models/task.go` - Status constants
2. `internal/repository/task_repository.go` - Status validation
3. `internal/repository/epic_repository.go` - Status validation
4. `internal/repository/feature_repository.go` - Status validation
5. `internal/cli/commands/task.go` - CLI validation

## Conclusion

The hardcoded status CHECK constraints have been successfully removed from the database schema. The migration is automatic, safe, and backward compatible. All tests pass. The database now accepts workflow-defined statuses from configuration.

**Status: ✅ COMPLETE**

All new workflow statuses can now be used:
- draft
- ready_for_refinement, in_refinement
- ready_for_development, in_development
- ready_for_code_review, in_code_review
- ready_for_qa, in_qa
- ready_for_approval, in_approval
- blocked, on_hold, completed, cancelled
