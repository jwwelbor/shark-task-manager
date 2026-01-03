# Tech Director Handoff: E07-F19 File Path Flag Standardization

**Date:** 2026-01-02
**From:** Tech Director
**To:** Backend Developer Agent
**Status:** Partial completion - needs developer to finish

---

## Summary

I've completed the initial database migration and partial model updates for E07-F19. The migration successfully drops the `custom_folder_path` columns from epics and features tables. However, several source files still reference `CustomFolderPath`, preventing compilation.

---

## Completed Work

### ✅ T-E07-F19-001: Database Schema Migration (DONE)

**Files Modified:**
- `internal/db/db.go` - Added `migrateDropCustomFolderPath()` function
- `internal/db/drop_custom_folder_path_migration_test.go` - TDD tests (passing)

**What was done:**
1. Created migration function that:
   - Checks if `custom_folder_path` column exists in epics/features tables
   - Drops indexes: `idx_epics_custom_folder_path` and `idx_features_custom_folder_path`
   - Drops columns using `ALTER TABLE ... DROP COLUMN`
   - Is idempotent (safe to run multiple times)

2. Removed code that ADDS custom_folder_path columns (lines 467-499 in db.go)

3. Removed custom_folder_path from:
   - Schema creation in `createSchema()` function
   - Index creation in `newIndexes` array

4. Tests pass: `go test -v ./internal/db -run TestMigrateDropCustomFolderPath`

**Database Status:**
- Migration runs successfully when shark CLI is invoked
- Columns are properly dropped from both tables
- Database has been restored from backup (`shark-tasks.db.backup-before-e07-f19`) because code isn't ready yet

---

### ⚠️ T-E07-F19-005: Update Data Models (PARTIALLY DONE)

**Files Modified:**
- `internal/models/epic.go` - Removed `CustomFolderPath` field ✅
- `internal/models/feature.go` - Removed `CustomFolderPath` field ✅
- `internal/repository/epic_repository.go` - Removed all custom_folder_path references ✅
  - Updated Create, GetByID, GetByKey (both numeric and slugged), Update
  - Removed `GetCustomFolderPath()` function entirely
  - Removed custom_folder_path from all SQL queries
- `internal/repository/feature_repository.go` - Removed all custom_folder_path references ✅
  - Updated all SELECT and INSERT queries
  - Removed `GetCustomFolderPath()` function
  - Updated Update method

**Files NOT Modified (compilation errors):**
```
internal/pathresolver/resolver.go - 7+ references to CustomFolderPath
internal/taskcreation/creator.go - 4 references to CustomFolderPath
internal/sync/discovery.go - 4 references to CustomFolderPath
```

Plus discovery package files:
```
internal/discovery/folder_scanner.go
internal/discovery/types.go
internal/discovery/frontmatter_parser.go
```

And migrate.go:
```
internal/db/migrate.go
```

And CLI commands (these are for flag standardization in T-E07-F19-002/003/004):
```
internal/cli/commands/epic.go
internal/cli/commands/feature.go
```

---

## What Needs to Be Done

### Immediate: Fix Compilation Errors

**Priority 1: Remove CustomFolderPath from pathresolver**

File: `internal/pathresolver/resolver.go`

The path resolver has precedence logic that checks custom_folder_path. According to the architecture decision, this entire precedence level should be REMOVED. Only `file_path` should be used.

**Current precedence:**
1. Explicit `file_path` (keep)
2. `custom_folder_path` (REMOVE - this is the bug!)
3. Default path calculation (keep)

**Required changes:**
- Remove all checks for `epic.CustomFolderPath`
- Remove all checks for `feature.CustomFolderPath`
- Simplify logic to only check `file_path` vs default

**Priority 2: Update taskcreation**

File: `internal/taskcreation/creator.go`

Similar to pathresolver - remove custom_folder_path logic, use only file_path.

**Priority 3: Update sync/discovery**

Files:
- `internal/sync/discovery.go`
- `internal/discovery/folder_scanner.go`
- `internal/discovery/types.go`
- `internal/discovery/frontmatter_parser.go`

Remove CustomFolderPath from:
- Struct literals
- Comparisons
- Assignments

**Priority 4: Check migrate.go**

File: `internal/db/migrate.go`

Verify no lingering references to custom_folder_path.

---

### Next: CLI Flag Standardization (T-E07-F19-002/003/004)

Once compilation works, implement flag standardization per architecture decision:

**Standard flags for epic/feature/task create/update commands:**
```go
// Primary flag (shown in help)
cmd.Flags().String("file", "", "Full file path (e.g., docs/custom/epic.md)")

// Aliases (hidden in help but functional)
cmd.Flags().String("filepath", "", "Alias for --file")
cmd.Flags().String("path", "", "Alias for --file")

// Hide aliases
cmd.Flags().MarkHidden("filepath")
cmd.Flags().MarkHidden("path")
```

**Files to update:**
- `internal/cli/commands/epic.go` - epic create, epic update
- `internal/cli/commands/feature.go` - feature create, feature update
- `internal/cli/commands/task.go` - task create (if it has custom path support)

**Validation:**
- All three flags (`--file`, `--filepath`, `--path`) should set the same value
- Flag processing should merge them (last one wins)
- Update help text to show only `--file`

---

## Testing Requirements

**After fixing compilation:**

1. **Run existing tests:**
   ```bash
   make test
   ```

2. **Test migration manually:**
   ```bash
   # Restore backup that still has custom_folder_path columns
   cp shark-tasks.db.backup-before-e07-f19 shark-tasks.db

   # Run any shark command to trigger migration
   ./bin/shark epic list

   # Verify columns dropped
   sqlite3 shark-tasks.db "PRAGMA table_info(epics)" | grep custom_folder_path
   # Should return nothing
   ```

3. **Test path resolution:**
   ```bash
   # Create epic with explicit file path
   ./bin/shark epic create "Test Epic" --file="docs/custom/test.md"

   # Verify it uses that path (not default)
   ./bin/shark epic get E##
   ```

4. **Integration tests (T-E07-F19-007):**
   - Create epics/features/tasks with --file, --filepath, --path flags
   - Verify all three work identically
   - Update entities and verify file paths change correctly

---

## Architecture Reference

See: `dev-artifacts/2026-01-02-feature-update-path-bug/ARCHITECTURE_DECISION.md`

**Key principles:**
- Use single `file_path` column storing full file path
- No path hierarchy, no path segments, no inheritance
- Default path computation unchanged (backward compatible)
- `--file` flag specifies full file path
- Simple beats complex (1 concept vs 6+ concepts)

---

## Files Backup

Before continuing, key backups exist:
- `shark-tasks.db.backup-before-e07-f19` - Database before migration
- Git has all code changes uncommitted (use `git diff` to see)

---

## Recommended Approach

1. **Fix compilation errors** (pathresolver, taskcreation, sync, discovery)
   - Search and remove all `CustomFolderPath` references
   - Simplify path logic to check only `file_path` vs default
   - Update tests that reference custom_folder_path

2. **Rebuild and verify** existing tests pass

3. **Implement flag standardization** (T-E07-F19-002/003/004)
   - Add `--file`, `--filepath`, `--path` flags
   - Hide aliases
   - Test all three work identically

4. **Update documentation** (T-E07-F19-006)
   - Update CLI_REFERENCE.md
   - Mark MIGRATION_CUSTOM_PATHS.md as deprecated
   - Update CLAUDE.md

5. **Integration testing** (T-E07-F19-007)
   - Test epic/feature/task creation with custom paths
   - Test updates change file paths correctly
   - Verify migration works on existing databases

---

## Questions?

Reach out to tech-director or review the architecture decision document for clarification.

**Good luck!** 🚀
