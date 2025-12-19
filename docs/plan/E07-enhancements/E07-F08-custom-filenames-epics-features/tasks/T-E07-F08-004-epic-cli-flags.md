---
task_key: T-E07-F08-004
epic_key: E07
feature_key: E07-F08
title: Add --filename and --force flags to shark epic create
status: created
priority: 3
agent_type: backend
depends_on: ["T-E07-F08-001", "T-E07-F08-002", "T-E07-F08-003"]
---

# Task: Add --filename and --force flags to shark epic create

## Objective

Extend the `shark epic create` command to accept `--filename` and `--force` flags, enabling custom file path assignment with collision detection and force reassignment.

## Context

**Why this task exists**: Epic files are currently locked to `docs/plan/{epic-key}/epic.md`. Users need flexibility to organize epics in custom locations (e.g., `docs/roadmap/2025.md`) to align with their documentation structure.

**Design reference**:
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (lines 43-119)
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (lines 98-131)

## What to Build

Modify `internal/cli/commands/epic.go` to add custom filename support:

### 1. CLI Flags

Add two new flags to the `shark epic create` command:

```go
epicCreateCmd.Flags().String("filename", "", "Custom filename path (relative to project root, must end in .md)")
epicCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another epic or feature")
```

### 2. Command Handler Logic

Update the epic creation handler to:

1. **Parse Flags**:
   - Read `--filename` and `--force` flags
   - If `--filename` is empty, use default behavior (NULL in database, create at `docs/plan/{epic-key}/epic.md`)

2. **Validate Custom Filename** (if provided):
   - Call `ValidateCustomFilename(filename, projectRoot)`
   - Return validation errors to user with clear messages

3. **Collision Detection**:
   - Query `epicRepo.GetByFilePath(ctx, validatedRelPath)`
   - If collision found and `--force` is false: return error with details
   - If collision found and `--force` is true: call `epicRepo.UpdateFilePath(ctx, conflictingEpicKey, nil)` to clear

4. **Create Epic**:
   - Set `epic.FilePath` to validated relative path (or `nil` for default)
   - Insert into database via existing epic creation flow
   - Create markdown file at the specified location (or default)

5. **Output**:
   - Success message: `Created epic {key} '{title}' at {file_path}`
   - If reassigned: `Created epic {key} '{title}' at {file_path} (reassigned from {entity_type} {entity_key})`

### 3. Help Text

Update command help to document both flags:

```
Flags:
  --filename string    Custom file path relative to project root (must end in .md)
  --force              Force reassignment if file already claimed by another entity
  --description string Epic description
  --priority string    Priority: high, medium, low (default: medium)
  --business-value string Business value: high, medium, low
```

## Success Criteria

- [ ] `shark epic create --filename="docs/roadmap/2025.md" "Platform"` creates epic at custom location
- [ ] Without `--filename`, epic is created at default location: `docs/plan/{epic-key}/epic.md`
- [ ] Collision detection prevents duplicate file claims without `--force`
- [ ] `--force` flag reassigns files from conflicting epics or features
- [ ] Error messages clearly indicate which entity owns a file during collision
- [ ] All validation rules from `ValidateCustomFilename` are enforced
- [ ] Help text documents both new flags with examples

## Validation Gates

1. **Manual Testing**:
   ```bash
   # Build the CLI
   make build

   # Test default behavior (backward compatibility)
   ./bin/shark epic create "Test Epic 1"
   # Verify: File created at docs/plan/E##/epic.md

   # Test custom filename
   ./bin/shark epic create --filename="docs/roadmap/test.md" "Test Epic 2"
   # Verify: File created at docs/roadmap/test.md

   # Test collision detection
   ./bin/shark epic create --filename="docs/roadmap/test.md" "Test Epic 3"
   # Verify: Error message shows Epic E## already claims the file

   # Test force reassignment
   ./bin/shark epic create --filename="docs/roadmap/test.md" --force "Test Epic 4"
   # Verify: Epic created, previous epic's file_path cleared

   # Test validation errors
   ./bin/shark epic create --filename="/absolute/path.md" "Test Epic 5"
   # Verify: Error about absolute paths

   ./bin/shark epic create --filename="docs/../outside.md" "Test Epic 6"
   # Verify: Error about path traversal

   ./bin/shark epic create --filename="docs/spec.txt" "Test Epic 7"
   # Verify: Error about .md extension requirement
   ```

2. **Integration Tests**:
   - Write tests in `internal/cli/commands/epic_test.go`:
     - `TestEpicCreate_CustomFilename`
     - `TestEpicCreate_CustomFilename_Collision`
     - `TestEpicCreate_ForceReassignment`
     - `TestEpicCreate_InvalidFilename_AbsolutePath`
     - `TestEpicCreate_InvalidFilename_PathTraversal`
     - `TestEpicCreate_DefaultBehavior`

3. **Regression Tests**:
   - Run all existing epic tests: `go test -v ./internal/cli/commands/... -run Epic`
   - Ensure no existing functionality breaks

## Dependencies

**Prerequisite Tasks**:
- T-E07-F08-001 (database schema with `file_path` column)
- T-E07-F08-002 (repository methods for collision detection and updates)
- T-E07-F08-003 (access to `ValidateCustomFilename`)

**Blocks**:
- T-E07-F08-006 (documentation updates reference this implementation)

## Implementation Notes

### Error Handling Pattern

Follow existing error patterns in the codebase:

```go
// Collision error without --force
if existingEpic != nil && !force {
    return fmt.Errorf("file '%s' is already claimed by epic %s ('%s'). Use --force to reassign",
        validatedRelPath, existingEpic.Key, existingEpic.Title)
}

// Validation error
if err := validateError; err != nil {
    return fmt.Errorf("invalid filename: %w", err)
}
```

### Transaction Safety

Wrap collision check, reassignment, and creation in a single transaction to ensure atomicity:

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// 1. Check collision
existingEpic, err := epicRepo.GetByFilePath(ctx, validatedRelPath)

// 2. Handle force reassignment
if existingEpic != nil && force {
    epicRepo.UpdateFilePath(ctx, existingEpic.Key, nil)
}

// 3. Create new epic
epicRepo.Create(ctx, newEpic)

// Commit transaction
tx.Commit()
```

### File Writing

Reuse existing file creation patterns from epic creation:

- Create parent directories if they don't exist
- Write YAML frontmatter with `epic_key`, `title`, `description`, etc.
- If file already exists, associate it (don't overwrite)

### Testing Database State

After reassignment, verify in database:

```sql
-- Check old epic has file_path = NULL
SELECT key, file_path FROM epics WHERE key = 'E04';

-- Check new epic has file_path set
SELECT key, file_path FROM epics WHERE key = 'E10';
```

## References

- **Backend Design**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (Section: shark epic create)
- **PRD Section**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (Section 3.1 - CLI Flags)
- **Existing Epic Command**: `internal/cli/commands/epic.go`
- **Task Filename Reference**: `internal/cli/commands/task.go` (for patterns to follow)
