# Current Implementation Analysis

## Database Schema Status

### Epics Table
```sql
file_path TEXT                  -- Stores full path (e.g., "docs/plan/E01/epic.md")
custom_folder_path TEXT         -- Exists but UNUSED
slug TEXT                       -- For human-readable keys
```

### Features Table
```sql
file_path TEXT                  -- Stores full path (e.g., "docs/plan/E01/F01/feature.md")
custom_folder_path TEXT         -- Exists but UNUSED
slug TEXT                       -- For human-readable keys
```

### Tasks Table
```sql
file_path TEXT                  -- Stores full path (e.g., "docs/plan/E01/F01/tasks/T-E01-F01-001.md")
-- NO custom_folder_path column! ❌
slug TEXT                       -- For human-readable keys
```

**Problem 1:** Tasks don't have `custom_folder_path` column at all
**Problem 2:** `custom_folder_path` exists for epics/features but is never used in path calculation

## Command Flag Analysis

### Epic Create (`shark epic create`)
- `--path` flag: Sets `custom_folder_path` in database ✓
- `--filename` flag: Sets full file path (WRONG - should only be filename)
- File creation: Uses default logic, ignores `custom_folder_path` ❌

### Epic Update (`shark epic update`)
- `--path` flag: Updates `custom_folder_path` in database ✓
- File modification: Does NOT move/update the actual file ❌
- No cascade to child features ❌

### Feature Create (`shark feature create`)
- `--path` flag: Sets `custom_folder_path` in database ✓
- `--filename` flag: Sets full file path (WRONG - should only be filename)
- File creation: Uses default logic, ignores `custom_folder_path` ❌
- Inheritance: Does NOT inherit epic's `custom_folder_path` ❌

### Feature Update (`shark feature update`)
- `--path` flag: Updates `custom_folder_path` in database ✓
- File modification: Does NOT move/update the actual file ❌
- No cascade to child tasks ❌

### Task Create (`shark task create`)
- `--path` flag: Does NOT exist ❌
- `--filename` flag: Sets full file path (WRONG - should only be filename)
- File creation: Uses default logic only
- Inheritance: Does NOT inherit from feature/epic paths ❌

### Task Update (`shark task update`)
- `--path` flag: Does NOT exist ❌
- File modification: No path update capability ❌

## Code Flow Analysis

### Feature Create Path Logic (feature.go:828-1125)

```
1. Parse --path flag → customFolderPath variable (line 922-930)
2. Parse --filename flag → customFilePath variable (line 945-1007)
3. Calculate file path:
   IF --filename provided:
     Use --filename (full path) → featureFilePath
   ELSE:
     Use DEFAULT logic (lines 1009-1040):
       - Find epic directory: docs/plan/{epic-key}-*/
       - Create feature directory: {epic-dir}/{feature-slug}/
       - Set file: {feature-dir}/feature.md
     ⚠️ NEVER checks or uses customFolderPath!
4. Create feature object:
   - FilePath = featureFilePath (line 1102)
   - CustomFolderPath = customFolderPath (line 1103)
```

**THE BUG:** Lines 1009-1040 calculate the file path using ONLY the default structure. The `customFolderPath` variable is set but never referenced in the path calculation logic!

### Feature Update Path Logic (feature.go:1585-1599)

```
1. Parse --path flag → relPath (lines 1585-1599)
2. Set feature.CustomFolderPath = &relPath
3. Call featureRepo.Update(ctx, feature)
4. Update database ✓
5. ⚠️ File on disk is NEVER moved or updated
6. ⚠️ Child tasks are NEVER notified of path change
```

**THE BUG:** Updates the database field but has no effect on the actual file location or child entities.

## What's Actually Working

✅ Database columns exist (for epics and features)
✅ Flags are parsed correctly
✅ Values are stored in database
✅ Display now shows custom_folder_path (after recent fix)

## What's Broken

❌ **Path calculation ignores custom_folder_path**
  - Feature create uses default path even when --path is provided
  - Epic create uses default path even when --path is provided

❌ **No filename separation**
  - --filename takes full path, not just filename
  - file_path stores full path (mixing path + filename)

❌ **No inheritance**
  - Features don't inherit epic's custom path
  - Tasks don't inherit feature's custom path
  - No cascade when parent path changes

❌ **No file movement**
  - Update --path changes database but not actual files
  - No option to move/rename files when path changes

❌ **Tasks missing custom_folder_path**
  - No database column
  - No --path flag in commands
  - No way to customize task file locations

❌ **No centralized path resolution**
  - Path logic duplicated across create/update commands
  - No single source of truth for "where should this file be?"

## Example: What Happens vs What Should Happen

### Example 1: Create feature with custom path

**Command:**
```bash
shark feature create --epic=E01 "Authentication" --path="docs/security"
```

**What happens now:**
1. Parses --path → customFolderPath = "docs/security" ✓
2. Calculates file path using DEFAULT: docs/plan/E01-*/E01-F01-authentication/feature.md ❌
3. Creates file at default location ❌
4. Stores in database:
   - file_path = "docs/plan/E01-.../E01-F01-authentication/feature.md" ❌
   - custom_folder_path = "docs/security" (UNUSED!) ✓

**What SHOULD happen:**
1. Parses --path → custom_path = "security" ✓
2. Resolves epic's path → "docs/plan/E01-epic-name" (or epic's custom path)
3. Computes full path: {epic-path}/security/feature.md ✓
4. Creates file at computed location ✓
5. Stores in database:
   - custom_path = "security" ✓
   - filename = "feature.md" ✓
   - file_path = COMPUTED (or cached) ✓

### Example 2: Update feature path

**Command:**
```bash
shark feature update E01-F01 --path="docs/archived"
```

**What happens now:**
1. Parses --path → relPath = "docs/archived" ✓
2. Updates database: custom_folder_path = "docs/archived" ✓
3. File stays at old location ❌
4. Next `feature get` shows custom_folder_path but file_path is unchanged ❌

**What SHOULD happen:**
1. Parses --path → new_path = "archived" ✓
2. Computes new full path → "docs/plan/E01-epic-name/archived/feature.md" ✓
3. Confirms with user: "Move file from X to Y?" ✓
4. Moves file on disk ✓
5. Updates database ✓
6. Cascades to all child tasks (recalculate their paths) ✓

## Impact Assessment

### Severity: High
- Feature is completely non-functional
- Advertised in documentation (CLAUDE.md) but doesn't work
- Users who tried to use it have incorrect data in database

### Data Integrity: Medium
- Database has custom_folder_path values that don't match actual file locations
- No data loss, but data is misleading/incorrect

### Backward Compatibility: Low Risk
- Feature was never working, so no users depend on current behavior
- Can fix without breaking existing functionality

## Root Cause Summary

1. **Incomplete implementation**: `custom_folder_path` column added but never integrated into path logic
2. **No path resolution abstraction**: Each command calculates paths independently
3. **Design confusion**: Mixing full paths with path segments
4. **Missing features**: Tasks don't have custom path support at all
5. **No testing**: No tests verify that --path flag actually affects file creation

## Recommended Fix Strategy

See `PATH_FILENAME_ARCHITECTURE.md` for detailed design and `REFACTORING_PLAN.md` for implementation steps.

### High-Level Approach

1. **Add missing schema**: Add custom_folder_path to tasks table
2. **Separate concerns**: Split file_path into custom_path + filename columns
3. **Centralize logic**: Create path resolution function with inheritance
4. **Fix commands**: Update create/update to use new path resolution
5. **Add tests**: Comprehensive tests for all path scenarios
6. **Migrate data**: Backfill new columns from existing file_path values
7. **Document**: Update CLI reference and migration guide

### Estimated Complexity

- **Database changes**: Medium (schema migration, data backfill)
- **Path resolution**: High (complex inheritance logic)
- **Command updates**: High (8+ commands to modify)
- **Testing**: High (many edge cases and scenarios)
- **Total effort**: Large refactoring (3-5 days)

## Files Needing Changes

### Core Logic
- `internal/db/db.go` - Schema migration for new columns
- `internal/utils/path_resolver.go` - NEW: Centralized path resolution
- `internal/models/epic.go` - Add path methods
- `internal/models/feature.go` - Add path methods
- `internal/models/task.go` - Add path methods

### Repository Layer
- `internal/repository/epic_repository.go` - Support new columns
- `internal/repository/feature_repository.go` - Support new columns
- `internal/repository/task_repository.go` - Support new columns

### Command Layer
- `internal/cli/commands/epic.go` - Fix create/update
- `internal/cli/commands/feature.go` - Fix create/update
- `internal/cli/commands/task.go` - Fix create/update, add --path flag

### Tests
- `internal/utils/path_resolver_test.go` - NEW: Test path resolution
- `internal/repository/*_test.go` - Update for new columns
- `internal/cli/commands/*_test.go` - Add path flag tests

### Documentation
- `docs/CLI_REFERENCE.md` - Update flag documentation
- `docs/MIGRATION_CUSTOM_PATHS.md` - Update migration guide
- `CLAUDE.md` - Update architecture documentation
