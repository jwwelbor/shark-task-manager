---
feature_key: E07-F08-custom-filenames-for-epics-and-features
epic_key: E07
title: Custom Filenames for Epics and Features
description: Add --filename and --force flags to epic and feature create commands, reusing task filename validation logic
---

# Custom Filenames for Epics and Features

**Feature Key**: E07-F08-custom-filenames-for-epics-and-features

---

## Epic

- **Epic PRD**: [E07 Enhancements](../../epic.md)
- **Related Feature**: [E07-F05: Task Custom Filenames](../../E07-F05-task-custom-filenames/) (Reference implementation)

---

## Goal

### Problem
Currently, epics and features are locked into a rigid directory structure (`docs/plan/{epicKey}/{featureKey}/`), while tasks already support custom filenames via `--filename` and `--force` flags. This inconsistency limits documentation flexibility and prevents epics/features from being placed alongside other project documentation (roadmaps, strategic initiatives, shared specifications). Users must either follow the convention or manually manage file associations outside the CLI.

### Solution
Extend custom filename support to both `shark epic create` and `shark feature create` commands using the same `--filename` and `--force` flags available for tasks. Reuse the existing `ValidateCustomFilename` validation logic to ensure consistent, secure filename handling across all entity types. This provides users with the flexibility to organize their documentation while maintaining safety guarantees.

### Impact
- **Feature Parity**: All entity types (epics, features, tasks) support custom filenames
- **Documentation Flexibility**: Users can organize epics/features anywhere in their project
- **Zero Breaking Changes**: Backward compatible; existing workflows unchanged
- **Code Reuse**: Leverages proven, tested patterns from task implementation

---

## User Personas

### Persona 1: Technical Lead / Project Architect

**Profile**:
- **Role/Title**: Technical leader managing multi-team project
- **Experience Level**: 5+ years in role, deep technical proficiency
- **Key Characteristics**:
  - Manages project structure and documentation standards
  - Works across multiple teams and documentation areas
  - Uses Shark CLI for task orchestration and planning

**Goals Related to This Feature**:
1. Organize epics/features alongside architectural documentation in shared areas
2. Maintain consistent file naming conventions across project documentation
3. Support team members with different documentation preferences

**Pain Points This Feature Addresses**:
- Epic/feature documentation locked into `docs/plan/` hierarchy
- Cannot colocate epic docs with architecture/design docs
- Manual workarounds needed for shared documentation

**Success Looks Like**:
Epic and feature documentation can be placed flexibly in the project structure, enabling better organization and alignment with existing documentation practices. Team members have consistent CLI experience across all entity types.

### Persona 2: Developer / Task Executor

**Profile**:
- **Role/Title**: Developer implementing features and tasks
- **Experience Level**: Varies, but comfortable with CLI
- **Key Characteristics**:
  - Executes tasks created by architects/leads
  - References epic and feature documentation regularly
  - Values consistency in tooling

**Goals Related to This Feature**:
1. Navigate documentation using familiar patterns
2. Have consistent CLI experience across entity types
3. Find documentation in logical locations

**Pain Points This Feature Addresses**:
- Learning curve when epic/feature behavior differs from tasks
- Documentation scattered across multiple locations
- Confusion from inconsistent entity handling

**Success Looks Like**:
The CLI behaves predictably; developers can use the same mental model (`--filename` and `--force`) for all entity types.

---

## User Stories

### Must-Have Stories

**Story 1: Custom Epic Filename**

As a technical lead, I want to create an epic with a custom filename path so that I can organize strategic epics alongside roadmap documentation rather than in the standard `docs/plan/` directory.

**Acceptance Criteria**:
- [ ] `shark epic create` accepts `--filename` flag with relative path
- [ ] Custom filename must include `.md` extension
- [ ] Path must be relative to project root
- [ ] Existing files are automatically associated (not overwritten)
- [ ] Default behavior unchanged: without `--filename`, epic uses `docs/plan/{epic-key}/epic.md`
- [ ] Error message clearly indicates file collision and shows which epic owns the file
- [ ] `--force` flag reassigns file from another epic (with warning in output)

**Story 2: Custom Feature Filename**

As a technical lead, I want to create a feature with a custom filename path so that feature specifications can live alongside relevant domain documentation or shared implementation guides.

**Acceptance Criteria**:
- [ ] `shark feature create` accepts `--filename` flag with relative path
- [ ] Custom filename must include `.md` extension
- [ ] Path must be relative to project root
- [ ] Existing files are automatically associated (not overwritten)
- [ ] Default behavior unchanged: without `--filename`, feature uses `docs/plan/{epic-key}/{feature-key}/feature.md`
- [ ] Error message clearly indicates file collision and shows which feature owns the file
- [ ] `--force` flag reassigns file from another feature (with warning in output)

**Story 3: Validation Consistency**

As a developer, I want the same validation rules and error messages across epic, feature, and task filename specifications so that I have a consistent mental model for the CLI.

**Acceptance Criteria**:
- [ ] All entity types reject absolute paths with same error message
- [ ] All entity types reject path traversal (`..`) with same error message
- [ ] All entity types require `.md` extension with same error message
- [ ] All entity types validate paths within project boundaries
- [ ] Collision detection uses consistent messaging format
- [ ] Force reassignment behavior identical across entity types

---

### Should-Have Stories

**Story 4: Documentation Updates**

As a new user, I want clear documentation of the `--filename` and `--force` flags for epic and feature creation so that I understand the capabilities and use cases.

**Acceptance Criteria**:
- [ ] `docs/CLI_REFERENCE.md` documents `--filename` and `--force` for both commands
- [ ] Examples show custom epic locations and custom feature locations
- [ ] Documentation explains collision detection and force reassignment
- [ ] Examples highlight backward compatibility (default behavior unchanged)

---

### Edge Case & Error Stories

**Error Story 1: Path Traversal Prevention**

When I attempt to create an epic with a filename containing `..`, the system rejects it with a clear error message.

**Acceptance Criteria**:
- [ ] Path `../../../etc/passwd` is rejected
- [ ] Error message: "invalid filename: path contains '..' (path traversal not allowed)"
- [ ] No file is created or modified

**Error Story 2: Absolute Path Rejection**

When I attempt to create a feature with an absolute path, the system rejects it with a clear error message.

**Acceptance Criteria**:
- [ ] Path `/home/user/docs/feature.md` is rejected
- [ ] Error message: "invalid filename: absolute paths not allowed, use relative paths"
- [ ] No file is created or modified

**Error Story 3: Wrong Extension**

When I attempt to create an epic with a non-Markdown extension, the system rejects it.

**Acceptance Criteria**:
- [ ] Path `docs/my-epic.txt` is rejected
- [ ] Error message: "invalid filename: file must have .md extension"
- [ ] No file is created or modified

**Error Story 4: File Collision Without Force**

When I attempt to create an epic using a filename already claimed by another epic, the system fails gracefully and shows which epic owns the file.

**Acceptance Criteria**:
- [ ] Command fails with exit code 1
- [ ] Error message: "file 'docs/roadmap.md' is already claimed by epic E04 ('Q1 Roadmap'). Use --force to reassign"
- [ ] No files are modified
- [ ] Original epic's file_path remains unchanged

**Error Story 5: Force Reassignment**

When I use `--force` to reassign a file from one epic to another, the old epic's file reference is cleared and the new epic claims the file.

**Acceptance Criteria**:
- [ ] Old epic's file_path column set to NULL
- [ ] New epic's file_path references the file path
- [ ] File content is not modified
- [ ] Output confirms reassignment with message like "Reassigned 'docs/roadmap.md' from E04 to E07"

---

## Requirements

### Functional Requirements

**Category: CLI Command Flags**

1. **REQ-F-001**: Epic Create --filename Flag
   - **Description**: `shark epic create --filename=<path>` accepts custom file path relative to project root
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag accepts string value (path to .md file)
     - [ ] Flag is optional (defaults to empty string)
     - [ ] Path can contain nested directories (e.g., `docs/roadmap/initiatives.md`)

2. **REQ-F-002**: Epic Create --force Flag
   - **Description**: `shark epic create --force` allows reassignment of files from other epics
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag is boolean (true if present, false otherwise)
     - [ ] Only applies when `--filename` is provided
     - [ ] Clears file_path from existing epic that owns the file

3. **REQ-F-003**: Feature Create --filename Flag
   - **Description**: `shark feature create --filename=<path>` accepts custom file path relative to project root
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag accepts string value (path to .md file)
     - [ ] Flag is optional (defaults to empty string)
     - [ ] Path can contain nested directories

4. **REQ-F-004**: Feature Create --force Flag
   - **Description**: `shark feature create --force` allows reassignment of files from other features
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag is boolean (true if present, false otherwise)
     - [ ] Only applies when `--filename` is provided
     - [ ] Clears file_path from existing feature that owns the file

**Category: Validation Logic**

5. **REQ-F-005**: Filename Validation
   - **Description**: Reuse `ValidateCustomFilename()` from task creator to validate epic/feature filenames
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Rejects absolute paths (e.g., `/home/user/file.md`)
     - [ ] Rejects path traversal (e.g., paths containing `..`)
     - [ ] Requires `.md` extension
     - [ ] Validates path within project boundaries via `patterns.ValidatePathWithinProject()`
     - [ ] Returns both absolute path (for file ops) and relative path (for database)

6. **REQ-F-006**: File Collision Detection
   - **Description**: Check if another epic/feature claims file before allowing creation
   - **User Story**: Stories 1-2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Query database for existing epic/feature with same file_path
     - [ ] Fail with clear error if collision detected (unless --force)
     - [ ] Error message includes existing entity key and title
     - [ ] Format: "file 'X' is already claimed by epic E04 ('Title'). Use --force to reassign"

7. **REQ-F-007**: File Association
   - **Description**: Associate existing files without overwriting; create new files if they don't exist
   - **User Story**: Stories 1-2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] If file exists: associate it (set file_path, don't modify content)
     - [ ] If file doesn't exist: create it with appropriate header markdown
     - [ ] If file exists and collision detected: fail (don't overwrite)

8. **REQ-F-008**: Force Reassignment Logic
   - **Description**: When --force is used, clear file_path from existing entity and assign to new entity
   - **User Story**: Stories 1-2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Update existing epic/feature to set file_path = NULL
     - [ ] Create new epic/feature with file_path set to custom filename
     - [ ] Transaction wraps both updates (atomic)
     - [ ] Output confirms reassignment

**Category: Database**

9. **REQ-F-009**: Epic file_path Column
   - **Description**: Add `file_path` column to epics table (if not already present)
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Column is TEXT type, nullable
     - [ ] Column has UNIQUE constraint (no two epics share same file)
     - [ ] Index `idx_epics_file_path` exists for query performance

10. **REQ-F-010**: Feature file_path Column
    - **Description**: Add `file_path` column to features table (if not already present)
    - **User Story**: Story 2
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] Column is TEXT type, nullable
      - [ ] Column has UNIQUE constraint (no two features share same file)
      - [ ] Index `idx_features_file_path` exists for query performance

11. **REQ-F-011**: Repository Methods
    - **Description**: Add GetByFilePath() and UpdateFilePath() to epic/feature repositories
    - **User Story**: Stories 1-2
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] `EpicRepository.GetByFilePath(ctx, path)` returns epic or nil
      - [ ] `FeatureRepository.GetByFilePath(ctx, path)` returns feature or nil
      - [ ] `EpicRepository.UpdateFilePath(ctx, epicKey, filePath)` updates file_path
      - [ ] `FeatureRepository.UpdateFilePath(ctx, featureKey, filePath)` updates file_path
      - [ ] Methods use parameterized queries for security
      - [ ] Methods properly handle transactions

### Non-Functional Requirements

**Security**

1. **REQ-NF-001**: Path Traversal Prevention
   - **Description**: Prevent directory traversal attacks via filename parameter
   - **Implementation**: ValidateCustomFilename rejects `..` in path
   - **Compliance**: OWASP Path Traversal Prevention
   - **Risk Mitigation**: Prevents writing/reading files outside project directory

2. **REQ-NF-002**: Arbitrary File Write Prevention
   - **Description**: Restrict filenames to `.md` extension only
   - **Implementation**: ValidateCustomFilename requires `.md` extension
   - **Compliance**: OWASP Input Validation
   - **Risk Mitigation**: Prevents creating executable files or system files

3. **REQ-NF-003**: Absolute Path Prevention
   - **Description**: Only allow relative paths, not absolute paths
   - **Implementation**: ValidateCustomFilename rejects absolute paths
   - **Compliance**: Principle of Least Privilege
   - **Risk Mitigation**: Ensures database portability and prevents escaping project

**Performance**

1. **REQ-NF-004**: Collision Detection Query Performance
   - **Description**: Collision detection queries complete in < 100ms
   - **Measurement**: Database query execution time
   - **Target**: < 100ms for collision detection queries
   - **Justification**: CLI responsiveness; users expect immediate feedback on file availability

2. **REQ-NF-005**: Index Coverage
   - **Description**: Database indexes on file_path columns for fast collision detection
   - **Measurement**: Query plan uses index (verified via EXPLAIN)
   - **Target**: No full table scans for file_path lookups
   - **Justification**: Performance scales with database size

**Backward Compatibility**

1. **REQ-NF-006**: Default Behavior Unchanged
   - **Description**: Epic/feature creation without --filename uses standard locations
   - **Target**: 100% of existing workflows unchanged
   - **Measurement**: No breaking changes to CLI contracts; all existing commands work identically
   - **Justification**: Non-disruptive feature; users opt-in to custom filenames

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Create Epic with Custom Filename**
- **Given** a valid Shark project with E07 epic initialized
- **When** user runs: `shark epic create "Q1 Strategic Goals" --epic=E07 --filename="docs/roadmap/q1.md"`
- **Then** epic E07-CUSTOM-KEY is created with file_path set to "docs/roadmap/q1.md"
- **And** if file doesn't exist, it's created with epic markdown header
- **And** if file exists, it's associated without modification
- **And** CLI outputs: "Epic created: E07-CUSTOM-KEY (custom file: docs/roadmap/q1.md)"

**Scenario 2: Create Feature with Custom Filename**
- **Given** a valid Shark project with E07 epic and E07-F08 feature initialized
- **When** user runs: `shark feature create "API Design" --epic=E07 --filename="docs/api/design.md"`
- **Then** feature E07-F08-CUSTOM-KEY is created with file_path set to "docs/api/design.md"
- **And** if file exists, it's associated; if not, it's created
- **And** CLI outputs: "Feature created: E07-F08-CUSTOM-KEY (custom file: docs/api/design.md)"

**Scenario 3: Collision Detection**
- **Given** epic E07 already claims file path "docs/roadmap.md"
- **When** user runs: `shark feature create "New Feature" --epic=E07 --filename="docs/roadmap.md"`
- **Then** command fails with exit code 1
- **And** error message: "file 'docs/roadmap.md' is already claimed by epic E07 ('Epic Title'). Use --force to reassign"
- **And** no files are modified
- **And** no new epic/feature is created

**Scenario 4: Force Reassignment**
- **Given** epic E04 currently owns file "docs/shared.md" (file_path = "docs/shared.md")
- **When** user runs: `shark epic create "New Epic" --epic=E07 --filename="docs/shared.md" --force`
- **Then** epic E04's file_path is set to NULL
- **And** new epic E07-NEW is created with file_path = "docs/shared.md"
- **And** file content is not modified
- **And** CLI outputs confirmation: "Reassigned 'docs/shared.md' from E04 to E07-NEW"

**Scenario 5: Validation Failure - Absolute Path**
- **Given** a Shark project initialized
- **When** user runs: `shark epic create "Test" --epic=E07 --filename="/etc/passwd.md"`
- **Then** command fails with exit code 1
- **And** error message: "invalid filename: absolute paths not allowed, use relative paths"
- **And** no files are created or modified

**Scenario 6: Validation Failure - Path Traversal**
- **Given** a Shark project initialized
- **When** user runs: `shark feature create "Test" --epic=E07 --filename="../../../etc/secret.md"`
- **Then** command fails with exit code 1
- **And** error message: "invalid filename: path contains '..' (path traversal not allowed)"
- **And** no files are created or modified

**Scenario 7: Validation Failure - Wrong Extension**
- **Given** a Shark project initialized
- **When** user runs: `shark epic create "Test" --epic=E07 --filename="docs/notes.txt"`
- **Then** command fails with exit code 1
- **And** error message: "invalid filename: file must have .md extension"
- **And** no files are created or modified

**Scenario 8: Default Behavior (No --filename)**
- **Given** a Shark project initialized with E07 epic
- **When** user runs: `shark epic create "Standard Epic" --epic=E07` (no --filename)
- **Then** epic is created with file_path = "docs/plan/E07/epic.md"
- **And** no change from existing behavior

---

## Out of Scope

### Explicitly Excluded

1. **Default File Path Changes**
   - **Why**: Would break backward compatibility; users rely on current standard locations
   - **Future**: Could be made configurable via `.sharkconfig.json` in future release
   - **Workaround**: Use `--filename` explicitly to change location per entity

2. **Bulk Reassignment Operations**
   - **Why**: Adds complexity; single entity operations sufficient for MVP
   - **Future**: Could add `shark sync --reassign-files` command later
   - **Workaround**: Reassign files one-by-one with `--force` flag

3. **File Deletion on Reassignment**
   - **Why**: Preserves data; reassignment only changes association, not files
   - **Future**: Could add `--cleanup` option to future sync command
   - **Workaround**: Users manually delete unreferenced files if desired

4. **Custom Epic Naming Format**
   - **Why**: Keeps task generation logic simple and predictable
   - **Future**: Could add naming schemes in future
   - **Workaround**: Use standard naming or rename after creation via database

---

## Implementation Notes

### Database Schema Changes

**File**: `internal/db/db.go`

If not already present, add to schema:
```sql
-- Check if file_path column exists before adding
-- (For epics table)
ALTER TABLE epics ADD COLUMN file_path TEXT UNIQUE;
CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);

-- (For features table)
ALTER TABLE features ADD COLUMN file_path TEXT UNIQUE;
CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);
```

### Model Changes

**File**: `internal/models/epic.go`
```go
type Epic struct {
    // ... existing fields ...
    FilePath *string `db:"file_path"` // NEW: nullable, for custom filename
}
```

**File**: `internal/models/feature.go`
```go
type Feature struct {
    // ... existing fields ...
    FilePath *string `db:"file_path"` // NEW: nullable, for custom filename
}
```

### Repository Methods

Both `EpicRepository` and `FeatureRepository` need:

```go
// GetByFilePath retrieves an epic/feature by its file path
func (r *EpicRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error)

// UpdateFilePath updates the file_path for an epic/feature
// Pass nil to clear the file path
func (r *EpicRepository) UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error
```

### Code Reuse

The `ValidateCustomFilename()` function from `internal/taskcreation/creator.go` should be:
1. Extracted to `internal/taskcreation/validation.go` (or similar shared location)
2. Imported by all creator functions (epic creator, feature creator, task creator)
3. No duplication of validation logic

### CLI Command Changes

**File**: `internal/cli/commands/epic.go`
- Add flags: `--filename`, `--force`
- Parse flags in `runEpicCreate`
- Pass to epic creator function

**File**: `internal/cli/commands/feature.go`
- Add flags: `--filename`, `--force`
- Parse flags in `runFeatureCreate`
- Pass to feature creator function

### Creator Function Changes

Need to create or update:
- **Epic creator** (may live in `internal/taskcreation/` or separate): Accepts custom filename, validates, detects collisions
- **Feature creator** (may live in `internal/taskcreation/` or separate): Accepts custom filename, validates, detects collisions

Both should follow the same pattern as task creator (see `ancient-purring-tower.md` plan).

---

## Success Criteria

### Functional Completeness
- [ ] `shark epic create --filename=<path>` works with validation
- [ ] `shark feature create --filename=<path>` works with validation
- [ ] `--force` flag works for both commands
- [ ] Collision detection prevents overwrites
- [ ] File association works (existing files and new files)
- [ ] Validation rejects absolute paths, path traversal, wrong extensions

### Code Quality
- [ ] No code duplication: ValidateCustomFilename is shared
- [ ] All repository methods follow existing patterns
- [ ] All CLI flags follow existing conventions
- [ ] Error messages consistent across entity types
- [ ] Transaction handling ensures atomicity

### Documentation
- [ ] `docs/CLI_REFERENCE.md` updated with new flags and examples
- [ ] `CLAUDE.md` updated with command syntax
- [ ] Examples show common use cases
- [ ] Error conditions documented

### Testing
- [ ] Unit tests for repository methods (GetByFilePath, UpdateFilePath)
- [ ] Integration tests for CLI commands with custom filenames
- [ ] Validation tests (absolute paths, traversal, extensions)
- [ ] Collision detection tests (with and without --force)
- [ ] Backward compatibility tests (default behavior unchanged)
- [ ] File association tests (existing vs. new files)

### Backward Compatibility
- [ ] All existing epic creation workflows still work
- [ ] All existing feature creation workflows still work
- [ ] Default file paths unchanged (docs/plan/...)
- [ ] No breaking changes to database contracts

---

## Assumptions & Constraints

### Assumptions
1. **ValidateCustomFilename exists and works** - Proven by E07-F05 (task feature) implementation
2. **Database supports UNIQUE constraints** - SQLite enforces uniqueness on file_path columns
3. **Project root available at runtime** - Via `os.Getwd()` in command handlers
4. **File system is accessible** - Can check file existence, create files (same as tasks)
5. **Transactions work correctly** - Both UpdateFilePath calls can be wrapped atomically

### Constraints
1. **Relative paths only** - Absolute paths rejected by validation; ensures portability
2. **Unique file paths** - Database enforces: no two epics/features can claim same file
3. **`.md` extension required** - Only markdown files supported; security boundary
4. **No file overwrite** - Existing files associated but not modified; data preservation
5. **Force requires explicit opt-in** - `--force` must be provided explicitly; prevents accidents

### Known Limitations
1. **No auto-cleanup** - Reassigned files remain on disk; users must manually delete if desired (future enhancement)
2. **No batch operations** - One entity at a time; scaling to 100+ entities requires multiple commands (future enhancement)
3. **No migration for existing entities** - Old epics/features have NULL file_path; they continue to use default locations (acceptable, backward compatible)

---

## Testing Strategy

### Unit Tests

**File**: `internal/repository/epic_repository_test.go`
- `TestEpicRepository_GetByFilePath` - Find epic by file path
- `TestEpicRepository_GetByFilePath_NotFound` - Return nil for missing path
- `TestEpicRepository_UpdateFilePath` - Update file path
- `TestEpicRepository_UpdateFilePath_ClearPath` - Set to NULL

**File**: `internal/repository/feature_repository_test.go`
- `TestFeatureRepository_GetByFilePath` - Find feature by file path
- `TestFeatureRepository_GetByFilePath_NotFound` - Return nil for missing path
- `TestFeatureRepository_UpdateFilePath` - Update file path
- `TestFeatureRepository_UpdateFilePath_ClearPath` - Set to NULL

**File**: `internal/taskcreation/validation_test.go` (if extracted)
- `TestValidateCustomFilename_ValidPaths` - Accept valid relative paths
- `TestValidateCustomFilename_RejectAbsolutePath` - Reject `/home/...`
- `TestValidateCustomFilename_RejectTraversal` - Reject `../...`
- `TestValidateCustomFilename_RejectWrongExtension` - Reject non-.md files
- `TestValidateCustomFilename_RejectPathsOutsideProject` - Validate boundaries

### Integration Tests

**File**: `internal/cli/commands/epic_test.go`
- `TestEpicCreate_CustomFilename` - Create epic with custom path
- `TestEpicCreate_CustomFilename_ExistingFile` - Associate existing file
- `TestEpicCreate_CustomFilename_Collision` - Detect collision, fail without --force
- `TestEpicCreate_ForceReassignment` - Reassign file from another epic
- `TestEpicCreate_InvalidFilename_AbsolutePath` - Reject absolute path
- `TestEpicCreate_InvalidFilename_Traversal` - Reject path traversal
- `TestEpicCreate_InvalidFilename_WrongExtension` - Reject non-.md
- `TestEpicCreate_DefaultBehavior` - No --filename uses standard location

**File**: `internal/cli/commands/feature_test.go`
- Same set of tests as epic (TestFeatureCreate_*)

### Scenario Tests (End-to-End)

Script: `scripts/test-custom-filenames.sh`
- Create epic with custom location, verify file exists and database record matches
- Create feature with custom location in nested directory
- Attempt collision, verify error message and database unchanged
- Force reassignment, verify old entity has NULL file_path
- Test all validation failures
- Verify default behavior unchanged (no --filename flag)

---

## Dependencies

- **E07-F05 (Task Custom Filenames)**: Reference implementation; provides ValidateCustomFilename and patterns
- **Existing epic/feature repository code**: Build upon existing CRUD methods
- **Existing database schema**: Add file_path columns and indexes

No blocking dependencies; E07-F05 is already complete.

---

## References

- Task filename implementation plan: `/home/jwwelbor/.claude/plans/ancient-purring-tower.md`
- Task creator source: `internal/taskcreation/creator.go`
- Epic repository: `internal/repository/epic_repository.go`
- Feature repository: `internal/repository/feature_repository.go`
- CLI commands: `internal/cli/commands/epic.go`, `internal/cli/commands/feature.go`

---

*Last Updated*: 2025-12-18
