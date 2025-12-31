# Analysis: Remove Hardcoded Status CHECK Constraints

## Problem Identified

The database schema has hardcoded CHECK constraints on status columns that prevent using the new workflow statuses defined in `.sharkconfig.json`.

### Current CHECK Constraints

1. **Tasks table** (line 150 in `internal/db/db.go`):
   ```sql
   status TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived'))
   ```

2. **Epics table** (line 87):
   ```sql
   status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived'))
   ```

3. **Features table** (line 117):
   ```sql
   status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived'))
   ```

4. **Migration file** (`internal/db/migrate.go` line 77):
   Has the same hardcoded constraints when recreating tasks table

### New Workflow Statuses

The `.sharkconfig.json` defines 14 statuses:
- draft
- ready_for_refinement
- in_refinement
- ready_for_development
- in_development
- ready_for_code_review
- in_code_review
- ready_for_qa
- in_qa
- ready_for_approval
- in_approval
- blocked
- on_hold
- completed
- cancelled

## Why CHECK Constraints Exist

Based on code review, the CHECK constraints were added for:
1. **Data integrity** - Prevent invalid status values
2. **Documentation** - Self-documenting valid states in schema
3. **Performance** - Database-level validation is fast

## Why They Need to Be Removed

1. **Flexibility** - Workflow statuses should be configurable, not hardcoded
2. **Extensibility** - Users should be able to define custom workflows
3. **Configuration-driven** - Status validation should use `.sharkconfig.json` as source of truth
4. **Migration** - Existing databases can't use new workflow without schema changes

## Solution Approach

### 1. Remove CHECK constraints from schema
- Update `createSchema()` in `internal/db/db.go`
- Remove CHECK constraints on status columns for all tables

### 2. Add migration to remove constraints from existing databases
- SQLite doesn't support `ALTER TABLE DROP CONSTRAINT`
- Must recreate tables without constraints
- Use same pattern as `MigrateRemoveAgentTypeConstraint()` in `migrate.go`

### 3. Add application-level validation
- Status validation should happen in repository layer
- Use workflow config to determine valid statuses
- Validate transitions according to `status_flow` in config

### 4. Update existing migration code
- `MigrateRemoveAgentTypeConstraint()` also has hardcoded status constraint
- Must be updated to remove status CHECK constraint too

## Implementation Steps

1. **Write tests first** (TDD) - ✅ DONE
   - Test that workflow statuses can be used
   - Test that transitions work
   - Document behavior change for invalid statuses

2. **Create migration function**
   - `MigrateRemoveStatusCheckConstraints()` for all three tables
   - Handle tasks, epics, features tables

3. **Update schema creation**
   - Remove CHECK constraints from `createSchema()`

4. **Update existing migration**
   - Fix `MigrateRemoveAgentTypeConstraint()` to not add status constraint back

5. **Run migrations in `runMigrations()`**
   - Call new migration function

6. **Verify tests pass** (GREEN phase)

7. **Add application-level validation** (future work)
   - Not part of this fix
   - Should be separate task/PR

## Risk Assessment

### Low Risk
- Migration pattern already used successfully for `agent_type` constraint
- Tests demonstrate correct behavior
- Backward compatible (all old statuses still work)

### Medium Risk
- Database accepts any string after migration
- Application MUST validate statuses to prevent corruption
- Missing validation is not caught at DB level

### Mitigation
- Add clear documentation about validation requirement
- Repository layer should validate against workflow config
- Consider adding status validation to all Create/Update operations

## Files to Modify

1. `internal/db/db.go` - Remove CHECK constraints from schema
2. `internal/db/migrate.go` - Add migration function
3. `internal/db/status_constraints_test.go` - Tests already written
4. Documentation - Update migration notes

## Test Results (RED phase)

All workflow status tests fail with:
```
CHECK constraint failed: status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived')
```

This confirms the problem and demonstrates expected behavior after fix.
