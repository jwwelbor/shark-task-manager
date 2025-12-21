---
task_key: T-E07-F08-005
epic_key: E07
feature_key: E07-F08
title: Add --filename and --force flags to shark feature create
status: created
priority: 3
agent_type: backend
depends_on: ["T-E07-F08-001", "T-E07-F08-002", "T-E07-F08-003"]
---

# Task: Add --filename and --force flags to shark feature create

## Objective

Extend the `shark feature create` command to accept `--filename` and `--force` flags, enabling custom file path assignment with collision detection and force reassignment.

## Context

**Why this task exists**: Feature files are currently locked to `docs/plan/{epic-key}/{feature-key}/feature.md`. Users need flexibility to organize features in custom locations (e.g., `docs/specs/auth-service.md`) to align with their documentation structure.

**Design reference**:
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (lines 121-198)
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (lines 116-131)

## What to Build

Modify `internal/cli/commands/feature.go` to add custom filename support:

### 1. CLI Flags

Add two new flags to the `shark feature create` command:

```go
featureCreateCmd.Flags().String("filename", "", "Custom filename path (relative to project root, must end in .md)")
featureCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another feature or epic")
```

### 2. Command Handler Logic

Update the feature creation handler to:

1. **Parse Flags**:
   - Read `--filename` and `--force` flags
   - If `--filename` is empty, use default behavior (NULL in database, create at `docs/plan/{epic-key}/{feature-key}/feature.md`)

2. **Validate Custom Filename** (if provided):
   - Call `ValidateCustomFilename(filename, projectRoot)`
   - Return validation errors to user with clear messages

3. **Collision Detection**:
   - Query `featureRepo.GetByFilePath(ctx, validatedRelPath)`
   - If collision found and `--force` is false: return error with details
   - If collision found and `--force` is true: call `featureRepo.UpdateFilePath(ctx, conflictingFeatureKey, nil)` to clear

4. **Create Feature**:
   - Set `feature.FilePath` to validated relative path (or `nil` for default)
   - Insert into database via existing feature creation flow
   - Create markdown file at the specified location (or default)

5. **Output**:
   - Success message: `Created feature {key} '{title}' at {file_path}`
   - If reassigned: `Created feature {key} '{title}' at {file_path} (reassigned from {entity_type} {entity_key})`

### 3. Help Text

Update command help to document both flags:

```
Flags:
  --epic string        Parent epic key (required)
  --filename string    Custom file path relative to project root (must end in .md)
  --force              Force reassignment if file already claimed by another entity
  --description string Feature description
  --execution-order int Optional execution order within epic
```

## Success Criteria

- [ ] `shark feature create --epic=E04 --filename="docs/specs/auth.md" "OAuth"` creates feature at custom location
- [ ] Without `--filename`, feature is created at default location: `docs/plan/{epic-key}/{feature-key}/feature.md`
- [ ] Collision detection prevents duplicate file claims without `--force`
- [ ] `--force` flag reassigns files from conflicting features or epics
- [ ] Error messages clearly indicate which entity owns a file during collision
- [ ] All validation rules from `ValidateCustomFilename` are enforced
- [ ] Help text documents both new flags with examples

## Validation Gates

1. **Manual Testing**:
   ```bash
   # Build the CLI
   make build

   # Create an epic first
   ./bin/shark epic create "Test Epic"
   # Assume epic key is E99

   # Test default behavior (backward compatibility)
   ./bin/shark feature create --epic=E99 "Test Feature 1"
   # Verify: File created at docs/plan/E99/E99-F01/feature.md

   # Test custom filename
   ./bin/shark feature create --epic=E99 --filename="docs/specs/test.md" "Test Feature 2"
   # Verify: File created at docs/specs/test.md

   # Test collision detection
   ./bin/shark feature create --epic=E99 --filename="docs/specs/test.md" "Test Feature 3"
   # Verify: Error message shows Feature E99-F02 already claims the file

   # Test force reassignment
   ./bin/shark feature create --epic=E99 --filename="docs/specs/test.md" --force "Test Feature 4"
   # Verify: Feature created, previous feature's file_path cleared

   # Test validation errors
   ./bin/shark feature create --epic=E99 --filename="/absolute/path.md" "Test Feature 5"
   # Verify: Error about absolute paths

   ./bin/shark feature create --epic=E99 --filename="docs/../outside.md" "Test Feature 6"
   # Verify: Error about path traversal

   ./bin/shark feature create --epic=E99 --filename="docs/spec.txt" "Test Feature 7"
   # Verify: Error about .md extension requirement
   ```

2. **Integration Tests**:
   - Write tests in `internal/cli/commands/feature_test.go`:
     - `TestFeatureCreate_CustomFilename`
     - `TestFeatureCreate_CustomFilename_Collision`
     - `TestFeatureCreate_ForceReassignment`
     - `TestFeatureCreate_InvalidFilename_AbsolutePath`
     - `TestFeatureCreate_InvalidFilename_PathTraversal`
     - `TestFeatureCreate_DefaultBehavior`

3. **Regression Tests**:
   - Run all existing feature tests: `go test -v ./internal/cli/commands/... -run Feature`
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

Follow the same error patterns as epic creation:

```go
// Collision error without --force
if existingFeature != nil && !force {
    return fmt.Errorf("file '%s' is already claimed by feature %s ('%s'). Use --force to reassign",
        validatedRelPath, existingFeature.Key, existingFeature.Title)
}

// Validation error
if err := validateError; err != nil {
    return fmt.Errorf("invalid filename: %w", err)
}
```

### Transaction Safety

Wrap collision check, reassignment, and creation in a single transaction:

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// 1. Check collision
existingFeature, err := featureRepo.GetByFilePath(ctx, validatedRelPath)

// 2. Handle force reassignment
if existingFeature != nil && force {
    featureRepo.UpdateFilePath(ctx, existingFeature.Key, nil)
}

// 3. Create new feature
featureRepo.Create(ctx, newFeature)

// Commit transaction
tx.Commit()
```

### File Writing

Reuse existing file creation patterns from feature creation:

- Create parent directories if they don't exist
- Write YAML frontmatter with `feature_key`, `epic_key`, `title`, `description`, etc.
- If file already exists, associate it (don't overwrite)

### Cross-Entity Collision

Features can also collide with epics (different entity type claiming same file). Check both:

```go
// Check features
existingFeature, _ := featureRepo.GetByFilePath(ctx, validatedRelPath)

// Check epics (cross-entity collision)
existingEpic, _ := epicRepo.GetByFilePath(ctx, validatedRelPath)

if existingFeature != nil && !force {
    return fmt.Errorf("file '%s' is already claimed by feature %s ('%s'). Use --force to reassign",
        validatedRelPath, existingFeature.Key, existingFeature.Title)
}

if existingEpic != nil && !force {
    return fmt.Errorf("file '%s' is already claimed by epic %s ('%s'). Use --force to reassign",
        validatedRelPath, existingEpic.Key, existingEpic.Title)
}

// Force reassignment handles either case
if existingFeature != nil && force {
    featureRepo.UpdateFilePath(ctx, existingFeature.Key, nil)
}
if existingEpic != nil && force {
    epicRepo.UpdateFilePath(ctx, existingEpic.Key, nil)
}
```

### Testing Database State

After reassignment, verify in database:

```sql
-- Check old feature has file_path = NULL
SELECT key, file_path FROM features WHERE key = 'E04-F12';

-- Check new feature has file_path set
SELECT key, file_path FROM features WHERE key = 'E05-F03';
```

## References

- **Backend Design**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (Section: shark feature create)
- **PRD Section**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (Section 3.1 - CLI Flags)
- **Existing Feature Command**: `internal/cli/commands/feature.go`
- **Task Filename Reference**: `internal/cli/commands/task.go` (for patterns to follow)
