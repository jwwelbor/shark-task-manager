# Feature PRD: Custom Filenames for Epics & Features

**Epic**: E07-Enhancements
**Feature**: E07-F08-Custom Filenames for Epics & Features
**Version**: 1.0
**Status**: Ready for Implementation
**Last Updated**: 2025-12-18

---

## 1. Overview

### Problem Statement

The Shark Task Manager now supports custom filenames for tasks (E07-F05), allowing users to place task files at arbitrary locations within the project. However, epics and features are still locked to a rigid default directory structure:

- Epics: Always `docs/plan/<epic-key>/epic.md`
- Features: Always `docs/plan/<epic-key>/<feature-key>/feature.md`

This inconsistency limits documentation flexibility. Users cannot organize epics and features alongside strategic roadmaps, shared documentation, or custom project hierarchies without creating the entire default structure.

### Solution

Extend custom filename support to epic and feature creation, providing **feature parity** with the task filename feature. Users gain:

- `--filename` flag to specify arbitrary `.md` file locations (relative to project root)
- `--force` flag to reassign files when collisions occur
- File collision detection to prevent unintended overwrites
- Same validation rules and patterns as tasks (proven, tested, secure)

### Why Now

- **Dependencies complete**: Task filename feature (E07-F05) provides proven patterns and validation logic
- **Low risk**: Reuses existing `ValidateCustomFilename` function—no new validation code needed
- **High consistency**: Makes CLI uniform across all entity types (epic, feature, task)
- **User demand**: Enables real-world use cases:
  - Roadmap documents in `/docs/roadmap/E07-overview.md`
  - Shared specifications in `/docs/shared/authentication.md`
  - Architecture decisions in `/docs/adr/E04-api-design.md`

---

## 2. User Stories

### User Story 1: Custom Epic Locations

**ID**: US-E07-F08-001

As a **product manager**, I want to create an epic at a custom file location (e.g., `/docs/roadmap/2025-platform.md`), so that I can organize strategic epics with other roadmap documents without maintaining the rigid `docs/plan/` structure.

#### Acceptance Criteria

- **AC-1.1**: `shark epic create --title="2025 Platform" --filename="docs/roadmap/2025-platform.md"` creates the epic key (e.g., E99) and writes the file at the specified path
- **AC-1.2**: The epic file contains standard YAML frontmatter with `epic_key` and `title` fields
- **AC-1.3**: Without `--filename`, the epic uses the default location: `docs/plan/E99/epic.md`
- **AC-1.4**: The command returns the epic key and confirmation message
- **AC-1.5**: If the custom file path already exists and another epic claims it, the command fails with a clear error message (unless `--force` is used)

---

### User Story 2: Custom Feature Locations

**ID**: US-E07-F08-002

As a **technical writer**, I want to place feature specifications in custom locations (e.g., `/docs/specs/E04-authentication.md`), so that I can maintain feature documentation in dedicated specification areas instead of following the default epic/feature folder hierarchy.

#### Acceptance Criteria

- **AC-2.1**: `shark feature create --epic=E04 --title="OAuth Login" --filename="docs/specs/E04-authentication.md"` creates the feature and writes at the custom path
- **AC-2.2**: The feature file contains standard YAML frontmatter with `feature_key`, `epic_key`, and `title`
- **AC-2.3**: Without `--filename`, the feature uses the default location: `docs/plan/E04/<generated-feature-key>/feature.md`
- **AC-2.4**: The command returns the feature key and confirmation message
- **AC-2.5**: File collision detection prevents multiple features from claiming the same file (error message shows which feature currently owns it, unless `--force` is used)

---

### User Story 3: CLI Consistency Across Entity Types

**ID**: US-E07-F08-003

As a **developer using the CLI**, I want epics, features, and tasks to have identical `--filename` and `--force` flag behavior, so that I don't need to learn different syntax or rules for different entity types.

#### Acceptance Criteria

- **AC-3.1**: All three commands accept `--filename` with the same validation rules:
  - Relative paths only (no absolute paths)
  - Must have `.md` extension
  - No path traversal (`..`) allowed
  - Must be within project boundaries
- **AC-3.2**: All three commands accept `--force` to override file collisions with identical semantics
- **AC-3.3**: Error messages follow the same format across all entity types
- **AC-3.4**: Help text for `shark epic create --help` and `shark feature create --help` documents both flags with examples

---

## 3. Feature Specification

### 3.1 CLI Flags

#### `shark epic create` (New Flags)

```bash
shark epic create --title="My Epic" [--priority=<high|medium|low>] [--business-value=<high|medium|low>] [--description="..."] [--filename=<path>] [--force]
```

**Flags**:
- `--filename` (string, optional): Custom file path relative to project root. Must end in `.md`. Example: `docs/roadmap/2025.md`
- `--force` (bool, optional): If set, reassign the file from any epic/feature currently claiming it. Default: `false`

**Behavior**:
- Without `--filename`: Creates file at `docs/plan/<epic-key>/epic.md` (current default)
- With `--filename`: Creates file at the specified path relative to project root
- Returns the generated epic key (e.g., `E09`) and confirmation message
- File is created only if it doesn't already exist; existing files are silently associated

#### `shark feature create` (New Flags)

```bash
shark feature create --epic=<epic-key> --title="My Feature" [--description="..."] [--execution-order=<N>] [--filename=<path>] [--force]
```

**Flags**:
- `--filename` (string, optional): Custom file path relative to project root. Must end in `.md`. Example: `docs/specs/auth-service.md`
- `--force` (bool, optional): If set, reassign the file from any feature currently claiming it. Default: `false`

**Behavior**:
- Without `--filename`: Creates file at `docs/plan/<epic-key>/<feature-key>/feature.md` (current default)
- With `--filename`: Creates file at the specified path
- Returns the generated feature key (e.g., `E04-F12`) and confirmation message
- File is created only if it doesn't already exist; existing files are silently associated

---

### 3.2 Validation Rules

All validation rules are **reused directly from the task filename feature** (E07-F05) via the existing `ValidateCustomFilename` function. No new validation code is needed.

**Rules Enforced**:

| Rule | Requirement | Error Message |
|------|-------------|---------------|
| **No Absolute Paths** | Filename must be relative to project root | `filename must be relative to project root, got absolute path: /absolute/path` |
| **Extension Required** | File must have `.md` extension | `invalid file extension: .txt (must be .md)` |
| **No Path Traversal** | Path cannot contain `..` | `invalid path: contains '..' (path traversal not allowed)` |
| **Within Project** | Path must resolve inside project root | `path validation failed: path resolves outside project root` |
| **Non-Empty Filename** | After normalization, filename must be valid | `invalid filename: resolved to empty or invalid path` |

**Implementation**:
- Function location: `internal/taskcreation/creator.go` → `ValidateCustomFilename(filename, projectRoot)`
- This function is **made public** (capitalize first letter) so epic/feature creators can reuse it
- Returns `(absPath string, relPath string, err error)` for both file operations and database storage

---

### 3.3 Collision Detection & Force Reassignment

#### Collision Detection Rules

When creating an epic or feature with a `--filename`:

1. **Check database** for any epic or feature claiming that file path
2. **If no collision**: Proceed normally
3. **If collision exists**:
   - Without `--force`: Return error showing which entity currently owns the file
   - With `--force`: Clear the file path from the existing entity and proceed

#### Error Messages

**Without --force** (collision detected):

```
Error: file 'docs/specs/auth.md' is already claimed by epic E04 ('OAuth Integration'). Use --force to reassign.
```

```
Error: file 'docs/roadmap/2025.md' is already claimed by feature E04-F01 ('User Authentication'). Use --force to reassign.
```

#### Force Reassignment Behavior

When `--force` is used and a collision is detected:

1. Update the existing epic/feature: Set `file_path` to NULL
2. Create the new epic/feature with the specified file path
3. Log the reassignment (if verbose mode enabled)
4. Return success message indicating the file was reassigned

**Success Message Example**:
```
Created epic E09 'Platform Roadmap' at docs/roadmap/2025.md (reassigned from epic E04)
```

---

### 3.4 File Existence Handling

**Behavior when `--filename` points to an existing file**:

- **File exists, no collision in DB**: Silently associate the file (file is created by someone else, we just claim it)
- **File exists, collision in DB**: Fail with collision error (unless `--force` is used)
- **File doesn't exist, no collision**: Create the file with standard YAML frontmatter
- **File doesn't exist, collision in DB**: Fail with collision error (unless `--force` is used)

This matches the task filename feature behavior exactly.

---

### 3.5 Default Behavior (No Flags)

When `--filename` is **not provided**, use the existing default behavior:

**For Epics**:
```
docs/plan/<epic-key>/epic.md
```
Example: `docs/plan/E09/epic.md`

**For Features**:
```
docs/plan/<epic-key>/<feature-key>/feature.md
```
Example: `docs/plan/E04/E04-F12/feature.md`

This ensures **backward compatibility**—existing workflows and scripts continue to work unchanged.

---

### 3.6 Edge Cases & Error Handling

| Edge Case | Input | Expected Behavior |
|-----------|-------|------------------|
| Absolute path | `--filename=/absolute/path/epic.md` | Reject: "filename must be relative to project root" |
| Path with `..` | `--filename=../docs/epic.md` | Reject: "contains '..' (path traversal not allowed)" |
| Wrong extension | `--filename=docs/epic.txt` | Reject: "invalid file extension: .txt (must be .md)" |
| Empty filename | `--filename=""` | Use default behavior (no error) |
| Path outside project | `--filename=../../outside/epic.md` | Reject: "path resolves outside project root" |
| File already owned by same entity | Create E09, then try again with same filename | Idempotent—succeed (no update needed) |
| File owned by different epic | E09 owns `docs/roadmap/2025.md`, try E10 same path | Fail with error (unless `--force`) |
| File owned by feature, create epic with same path | F01 owns `docs/shared.md`, create epic with same | Fail with error (unless `--force`) |
| Both `--force` and no collision | File unclaimed, use `--force` | Succeed normally (no-op) |
| Concurrent creation of two epics, same file | Race condition in DB | Database foreign key constraint + unique file_path index prevents both |

---

## 4. Implementation Notes

### 4.1 Database Changes

**Schema**: No new columns needed. The `file_path` field already exists in both `epics` and `features` tables based on the feature-definition.md. However, the current schema in `/home/jwwelbor/projects/shark-task-manager/internal/db/db.go` does NOT include `file_path` columns for epics and features.

**Required Schema Updates**:

1. **Add `file_path` column to `epics` table**:
   ```sql
   ALTER TABLE epics ADD COLUMN file_path TEXT;
   ```

2. **Add `file_path` column to `features` table**:
   ```sql
   ALTER TABLE features ADD COLUMN file_path TEXT;
   ```

3. **Add indexes for performance** (in the schema creation function):
   ```sql
   CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);
   CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);
   ```

These indexes ensure collision detection queries are efficient.

---

### 4.2 Model Changes

**Files to Update**:
- `internal/models/epic.go`: Add `FilePath *string` field to `Epic` struct with `db:"file_path"` tag
- `internal/models/feature.go`: Add `FilePath *string` field to `Feature` struct with `db:"file_path"` tag

**Nullable Rationale**: Use `*string` (pointer) to allow NULL values in database. When an epic/feature uses the default location, `FilePath` can be NULL, indicating "no custom path."

---

### 4.3 Repository Layer Changes

**File**: `internal/repository/epic_repository.go`

Add two new methods:

```go
// GetByFilePath retrieves an epic by its file path
// Returns nil if no epic found (not an error condition)
func (r *EpicRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error) {
    // SQL query: SELECT ... FROM epics WHERE file_path = ?
    // Returns sql.ErrNoRows wrapped if not found
}

// UpdateFilePath updates the file_path for an epic
// Pass nil to clear the file path (set to NULL)
func (r *EpicRepository) UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error {
    // SQL query: UPDATE epics SET file_path = ? WHERE key = ?
}
```

**File**: `internal/repository/feature_repository.go`

Add the same two methods for features:

```go
// GetByFilePath retrieves a feature by its file path
func (r *FeatureRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error)

// UpdateFilePath updates the file_path for a feature
func (r *FeatureRepository) UpdateFilePath(ctx context.Context, featureKey string, newFilePath *string) error
```

---

### 4.4 Creator/Creator Functions

**Note**: The task filename feature uses a centralized `Creator` struct in `internal/taskcreation/creator.go`. For epics and features, we need similar creation logic. Options:

**Option A (Recommended)**: Extend the `Creator` struct to handle epics and features, or create separate `EpicCreator` and `FeatureCreator` structs with similar patterns.

**Option B**: Duplicate the logic in CLI command handlers (less maintainable).

The existing `ValidateCustomFilename` function is already public and reusable, so both options can leverage it.

**Key Functions to Create/Modify**:

1. **Make `ValidateCustomFilename` public** (rename to `ValidateCustomFilename` with uppercase, it's already there)
2. **Extract validation into a utility** (or reuse existing) that both epic/feature creators and task creator call
3. **Epic creation flow**:
   - Validate inputs
   - Generate epic key
   - Validate custom filename (if provided)
   - Check for file collision
   - Handle force reassignment
   - Create file (if doesn't exist)
   - Save to database

4. **Feature creation flow**: Same as epic, but with feature-specific generation

---

### 4.5 CLI Command Changes

**File**: `internal/cli/commands/epic.go`

Locate the epic create command handler and add flags:

```go
epicCreateCmd.Flags().String("filename", "", "Custom filename path (relative to project root)")
epicCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another epic")
```

Parse and pass to creator:

```go
filename, _ := cmd.Flags().GetString("filename")
force, _ := cmd.Flags().GetBool("force")

input := epic.CreateEpicInput{
    Title: title,
    // ... other fields ...
    Filename: filename,
    Force: force,
}

creator := epic.NewCreator(repositories, projectRoot)
result, err := creator.Create(ctx, input)
```

**File**: `internal/cli/commands/feature.go`

Same pattern for feature creation:

```go
featureCreateCmd.Flags().String("filename", "", "Custom filename path (relative to project root)")
featureCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another feature")
```

---

### 4.6 Code Reuse Opportunities

1. **ValidateCustomFilename**: Already exists in `internal/taskcreation/creator.go`—make it available to epic/feature creators
   - Option: Move to a shared utility package (e.g., `internal/validation/filename.go`)
   - Option: Export from `taskcreation` package
   - Preferred: Move to `internal/patterns/` or create new shared validation module

2. **Collision detection pattern**: Replicate the logic from task creator:
   ```go
   existing, err := repo.GetByFilePath(ctx, filePath)
   if existing != nil && !input.Force {
       return fmt.Errorf("file '%s' is already claimed by %s", filePath, existing.Key)
   }
   if existing != nil && input.Force {
       repo.UpdateFilePath(ctx, existing.Key, nil)
   }
   ```

3. **Error message format**: Consistent across all entity types

---

## 5. Test Strategy

### 5.1 Unit Tests

**File**: New file `internal/repository/epic_repository_file_path_test.go` (or add to existing test file)

- `TestEpicRepository_GetByFilePath`: Retrieve epic by custom file path
- `TestEpicRepository_GetByFilePath_NotFound`: Return nil when no epic found
- `TestEpicRepository_UpdateFilePath`: Update file path for existing epic
- `TestEpicRepository_UpdateFilePath_Clear`: Set file path to NULL

**File**: New file `internal/repository/feature_repository_file_path_test.go`

- `TestFeatureRepository_GetByFilePath`: Retrieve feature by custom file path
- `TestFeatureRepository_GetByFilePath_NotFound`: Return nil when no feature found
- `TestFeatureRepository_UpdateFilePath`: Update file path for existing feature
- `TestFeatureRepository_UpdateFilePath_Clear`: Set file path to NULL

### 5.2 Integration Tests

**File**: `internal/cli/commands/epic_test.go` (or new file if doesn't exist)

- `TestEpicCreate_CustomFilename`: Create epic with custom path
- `TestEpicCreate_CustomFilename_ExistingFile`: Associate with existing file
- `TestEpicCreate_CustomFilename_Collision`: Detect and reject collision without --force
- `TestEpicCreate_ForceReassignment_FromEpic`: Reassign file from another epic
- `TestEpicCreate_ForceReassignment_FromFeature`: Reassign file from a feature (force override)
- `TestEpicCreate_InvalidFilename_AbsolutePath`: Reject absolute path
- `TestEpicCreate_InvalidFilename_PathTraversal`: Reject `..` in path
- `TestEpicCreate_InvalidFilename_WrongExtension`: Reject non-.md files
- `TestEpicCreate_DefaultFilename_Backward_Compatible`: Without --filename, use default location

**File**: `internal/cli/commands/feature_test.go`

- Similar test suite for feature creation

### 5.3 Validation Tests

Test the ValidateCustomFilename function specifically:

**File**: `internal/taskcreation/creator_test.go` (add to existing or enhance)

- `TestValidateCustomFilename_ValidPath`: Accept `docs/spec/auth.md`
- `TestValidateCustomFilename_AbsolutePath`: Reject `/absolute/path/epic.md`
- `TestValidateCustomFilename_PathTraversal`: Reject `docs/../../outside.md`
- `TestValidateCustomFilename_WrongExtension`: Reject `docs/spec/auth.txt`
- `TestValidateCustomFilename_OutsideProject`: Reject paths that resolve outside project root
- `TestValidateCustomFilename_ReturnsAbsAndRelPaths`: Verify both absolute and relative paths are correct

### 5.4 End-to-End Scenarios

**Test Case 1: Complete Workflow - Epic with Custom Path**
```
1. Run: shark epic create --title="Platform Roadmap" --filename="docs/roadmap/2025.md"
2. Verify: Epic E09 created, file exists at docs/roadmap/2025.md
3. Verify: Database stores file_path = "docs/roadmap/2025.md" for E09
4. Run: shark epic get E09 --json
5. Verify: JSON response includes file_path
```

**Test Case 2: Collision Detection**
```
1. Create epic E09 with filename="docs/shared/spec.md"
2. Try to create feature E04-F20 with filename="docs/shared/spec.md"
3. Verify: Command fails with error "file '...' is already claimed by epic E09"
4. Run with --force
5. Verify: Feature created, epic E09 file_path cleared (set to NULL)
6. Verify: Feature E04-F20 now claims the file
```

**Test Case 3: Default Behavior Unchanged**
```
1. Run: shark epic create --title="Feature Set" (no --filename)
2. Verify: Epic E10 created at docs/plan/E10/epic.md (default)
3. Verify: file_path is NULL in database
4. Run: shark feature create --epic=E10 --title="Login" (no --filename)
5. Verify: Feature E10-F01 created at docs/plan/E10/E10-F01/feature.md (default)
6. Verify: file_path is NULL in database
```

---

## 6. Success Criteria

The feature is complete when:

### Functional Criteria
- [ ] `shark epic create --filename=<path>` works and creates files at custom locations
- [ ] `shark feature create --filename=<path>` works and creates files at custom locations
- [ ] Both commands validate filenames with identical rules (no absolute paths, no `..`, `.md` only)
- [ ] File collision detection prevents multiple entities from claiming the same file (unless --force)
- [ ] `--force` flag reassigns files from one entity to another
- [ ] Error messages clearly indicate which entity owns a file when collisions occur
- [ ] Without `--filename`, both commands use default locations (backward compatible)

### Code Quality Criteria
- [ ] ValidateCustomFilename is reused (not duplicated) across task/epic/feature creators
- [ ] All tests pass (unit, integration, validation)
- [ ] Code follows existing patterns (dependency injection, error handling, repository design)
- [ ] No new validation code written—reuse existing ValidateCustomFilename function
- [ ] Database migrations applied (add file_path columns and indexes)

### Documentation Criteria
- [ ] `docs/CLI_REFERENCE.md` updated with `--filename` and `--force` flag documentation
- [ ] `CLAUDE.md` updated with new command syntax
- [ ] Help text in commands shows both flags with examples
- [ ] README.md mentions feature parity across entity types (if documentation update section exists)

### Backward Compatibility Criteria
- [ ] Existing workflows without `--filename` continue to work unchanged
- [ ] Default file paths remain the same
- [ ] No breaking changes to CLI syntax
- [ ] Database schema migration handled gracefully (ALTER TABLE with NOT NULL constraint should use default values)

---

## 7. Assumptions & Constraints

### Assumptions

1. **ValidateCustomFilename is reusable**: The existing function in `internal/taskcreation/creator.go` can be exported and reused by epic/feature creators without modification
2. **Database schema migration is possible**: The production database will be migrated to add `file_path` columns to epics and features
3. **Epic and feature keys are already generated**: Similar to tasks, epic and feature key generation logic already exists
4. **YAML frontmatter parsing**: Epic and feature files already use YAML frontmatter (e.g., `epic_key`, `feature_key`); no new parsing needed
5. **File writing utilities exist**: Existing file writing patterns (from task creation) can be reused

### Constraints

1. **Cannot modify existing default paths**: Backward compatibility requires that epics and features without `--filename` always use `docs/plan/` hierarchy
2. **File path uniqueness**: Only one entity (epic or feature) can claim a single file (unless `--force` is used)
3. **Relative paths only**: Absolute paths are rejected for portability (same as tasks)
4. **Markdown files only**: `.md` extension is required (consistent with all project documentation)
5. **No cross-database collisions**: Collision detection only checks within the current database; external files are not tracked

### Known Limitations

1. **File path not unique across entity types initially**: An epic and feature could potentially claim the same file if collision detection is cross-entity (clarification: the feature-definition.md suggests this should work, but implementation needs to verify)
2. **Migration required**: If the codebase already has epics/features in production, a database migration script is needed to add the `file_path` columns

---

## 8. Dependencies & Sequencing

### External Dependencies
- **E07-F05 (Task Filename Feature)**: COMPLETE ✓
  - Provides proven patterns and validation logic
  - `ValidateCustomFilename` function available for reuse
  - Collision detection and force reassignment patterns established

### Prerequisite Work
1. ✓ Database schema modified (add `file_path` columns and indexes)
2. ✓ Models updated (Epic and Feature structs with FilePath field)
3. ✓ Repositories updated (GetByFilePath and UpdateFilePath methods)
4. ✓ ValidateCustomFilename exported for reuse
5. CLI commands updated with new flags
6. Integration tests written and passing

### Sequencing
This feature should be implemented **after** E07-F05 is complete and proven. No parallelization needed; the feature is straightforward and follows established patterns.

---

## 9. References

- **Feature Definition**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/feature-definition.md`
- **Task Filename Implementation Plan**: `/home/jwwelbor/.claude/plans/ancient-purring-tower.md`
- **Current Database Schema**: `internal/db/db.go` (lines 67-191)
- **Task Filename Implementation**: `internal/taskcreation/creator.go` (ValidateCustomFilename, CreateTask)
- **CLI Reference**: `docs/CLI_REFERENCE.md`

---

## Appendix A: Example User Workflows

### Example 1: Create Epic at Custom Roadmap Path

```bash
$ shark epic create --title="2025 Platform Roadmap" --filename="docs/roadmap/2025-platform.md"

Created epic E09 'Platform Roadmap'
File: docs/roadmap/2025-platform.md
Key: E09

$ cat docs/roadmap/2025-platform.md
---
epic_key: E09
title: 2025 Platform Roadmap
description: null
---

# 2025 Platform Roadmap
...
```

### Example 2: Create Feature at Custom Spec Path

```bash
$ shark feature create --epic=E04 --title="OAuth Integration" --filename="docs/specs/auth-oauth.md"

Created feature E04-F15 'OAuth Integration'
File: docs/specs/auth-oauth.md
Key: E04-F15

$ cat docs/specs/auth-oauth.md
---
feature_key: E04-F15
epic_key: E04
title: OAuth Integration
description: null
---

# OAuth Integration
...
```

### Example 3: Collision Detection and Force Reassignment

```bash
# First, E04 owns the file
$ shark epic create --title="Authentication" --filename="docs/shared/auth.md"
Created epic E04 'Authentication'
File: docs/shared/auth.md

# Try to create feature with same path (no --force)
$ shark feature create --epic=E05 --title="SSO" --filename="docs/shared/auth.md"
Error: file 'docs/shared/auth.md' is already claimed by epic E04 ('Authentication'). Use --force to reassign.

# Use --force to reassign
$ shark feature create --epic=E05 --title="SSO" --filename="docs/shared/auth.md" --force
Created feature E05-F01 'SSO' (reassigned from epic E04)
File: docs/shared/auth.md

# Verify reassignment
$ shark epic get E04
...
file_path: null  # Now null because reassigned
```

### Example 4: Backward Compatibility (Default Paths)

```bash
# Without --filename, uses default locations
$ shark epic create --title="Core Features"
Created epic E06 'Core Features'
File: docs/plan/E06/epic.md

$ shark feature create --epic=E06 --title="Database"
Created feature E06-F01 'Database'
File: docs/plan/E06/E06-F01/feature.md

# Identical to existing behavior—no breaking changes
```

---

## Appendix B: Database Schema Changes

### Current State (from internal/db/db.go)

```sql
CREATE TABLE epics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (...),
    priority TEXT NOT NULL CHECK (...),
    business_value TEXT CHECK (...),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (...),
    progress_pct REAL NOT NULL DEFAULT 0.0 CHECK (...),
    execution_order INTEGER NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
);
```

### Required Changes

```sql
-- Add file_path column to epics
ALTER TABLE epics ADD COLUMN file_path TEXT;

-- Add file_path column to features
ALTER TABLE features ADD COLUMN file_path TEXT;

-- Add indexes for collision detection queries
CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);
CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);
```

### Updated Schema (for new projects)

Both tables gain:
```sql
file_path TEXT,  -- NULL if using default location
```

And both gain corresponding indexes.

---

**Document End**
