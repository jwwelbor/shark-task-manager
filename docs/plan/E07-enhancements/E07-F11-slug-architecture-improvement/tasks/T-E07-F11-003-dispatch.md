# Task Dispatch: T-E07-F11-003

**Task**: Create migration CLI command for slug columns
**Agent**: Backend Developer
**Status**: Dispatched
**Dispatched**: 2025-12-30

---

## Context

T-E07-F11-001 (database schema) and T-E07-F11-002 (backfill logic) are complete. The backfill function `BackfillSlugsFromFilePaths` exists in `/home/jwwelbor/projects/shark-task-manager/internal/db/migrate_slug_backfill.go` and has been tested.

Now we need to create a CLI command that:
1. Exposes this backfill functionality to users
2. Provides dry-run mode to preview changes
3. Provides stats/reporting on migration results
4. Follows shark CLI patterns and conventions

---

## Requirements

### Command Structure

Create a new command: `shark migrate backfill-slugs`

**Flags:**
- `--dry-run` - Preview changes without applying them (default: false)
- `--verbose` / `-v` - Show detailed output (default: false)
- `--json` - Output results in JSON format (default: false)

**Example Usage:**
```bash
# Preview changes
shark migrate backfill-slugs --dry-run

# Apply migration
shark migrate backfill-slugs

# Apply with verbose output
shark migrate backfill-slugs --verbose

# Get JSON output for automation
shark migrate backfill-slugs --json
```

### Implementation Plan

1. **Create migrate command group** (if doesn't exist)
   - File: `internal/cli/commands/migrate.go`
   - Parent command: `shark migrate`
   - Subcommands will register under it

2. **Create backfill subcommand**
   - File: `internal/cli/commands/migrate_backfill_slugs.go`
   - Command: `backfill-slugs`
   - Calls `db.BackfillSlugsFromFilePaths(db)`

3. **Add dry-run support**
   - Modify `BackfillSlugsFromFilePaths` to accept a `dryRun bool` parameter
   - When `dryRun=true`, don't commit transaction (rollback instead)
   - Return statistics about what WOULD be updated

4. **Add reporting/statistics**
   - Return struct with migration stats:
     ```go
     type MigrationStats struct {
         EpicsUpdated    int
         FeaturesUpdated int
         TasksUpdated    int
         TotalEpics      int
         TotalFeatures   int
         TotalTasks      int
     }
     ```

5. **Output formatting**
   - Table format (default): Show before/after counts
   - JSON format (`--json`): Machine-readable stats
   - Verbose format (`--verbose`): Show individual updates

### Files to Create/Modify

**Create:**
- `internal/cli/commands/migrate.go` - Parent command
- `internal/cli/commands/migrate_backfill_slugs.go` - Subcommand implementation

**Modify:**
- `internal/db/migrate_slug_backfill.go` - Add dry-run support and return stats

**Test:**
- `internal/cli/commands/migrate_backfill_slugs_test.go` - Command tests with mocks

---

## Acceptance Criteria

- [ ] Command `shark migrate backfill-slugs` exists and runs without error
- [ ] `--dry-run` flag works (no database changes, shows preview)
- [ ] `--verbose` flag shows detailed migration output
- [ ] `--json` flag outputs machine-readable results
- [ ] Command shows before/after statistics for all entity types
- [ ] Migration is idempotent (can run multiple times safely)
- [ ] Tests cover dry-run mode, stats reporting, and error handling
- [ ] Help text explains all flags and usage examples

---

## Testing Approach

### Unit Tests (internal/cli/commands/migrate_backfill_slugs_test.go)
- Mock repository and database
- Test command argument parsing
- Test output formatting (table vs JSON)
- Test error handling

### Integration Tests (manual validation)
- Run on test database with `--dry-run` first
- Verify stats are accurate
- Apply migration without `--dry-run`
- Verify all slugs populated correctly
- Run again to verify idempotency

---

## Implementation Notes

### Dry-Run Pattern
```go
func BackfillSlugsFromFilePaths(db *sql.DB, dryRun bool) (*MigrationStats, error) {
    tx, err := db.Begin()
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    // ... perform updates ...

    stats := &MigrationStats{...}

    if dryRun {
        // Don't commit - just return stats
        return stats, nil
    }

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    return stats, nil
}
```

### Statistics Collection
Modify backfill logic to count updates:
- Count records with NULL/empty slugs before update
- Count records updated successfully
- Count total records in each table

### CLI Output Example (Table Format)
```
Backfilling slugs from file paths...

Results:
  Entity    Before  After  Updated
  ------    ------  -----  -------
  Epics     0/7     7/7    7
  Features  0/11    11/11  11
  Tasks     0/278   278/278 278

Migration completed successfully!
```

### CLI Output Example (Dry-Run)
```
DRY RUN MODE - No changes will be applied

Preview of slug backfill:

Results:
  Entity    Current  Will Update
  ------    -------  -----------
  Epics     0/7      7
  Features  0/11     11
  Tasks     0/278    278

Run without --dry-run to apply changes
```

---

## Related Files

**Existing Backfill Logic:**
- `/home/jwwelbor/projects/shark-task-manager/internal/db/migrate_slug_backfill.go`
- `/home/jwwelbor/projects/shark-task-manager/internal/db/migrate_slug_backfill_test.go`

**Test Utility (for reference, not production):**
- `/home/jwwelbor/projects/shark-task-manager/cmd/test-backfill/main.go`

**Existing CLI Command Patterns:**
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go`
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/sync.go`

---

## Dependencies

**Prerequisite Tasks:**
- ✅ T-E07-F11-001 - Database schema with slug columns
- ✅ T-E07-F11-002 - Backfill migration logic

**Blocks:**
- T-E07-F11-004 - Generate and store slug during epic creation
- T-E07-F11-005 - Generate and store slug during feature creation
- T-E07-F11-006 - Generate and store slug during task creation

---

## Estimated Effort

**Size**: Medium (M)
**Estimated Time**: 2-3 hours

**Breakdown:**
- Create migrate command structure: 30 min
- Add dry-run support to backfill logic: 45 min
- Implement stats reporting: 30 min
- Create CLI command: 45 min
- Write tests: 45 min
- Manual testing and validation: 30 min

---

## Success Metrics

- Command runs without errors
- Dry-run mode accurately predicts changes
- Stats are accurate (verified against database queries)
- Migration is idempotent (running twice produces same result)
- All tests pass
- User can confidently run migration with preview first
