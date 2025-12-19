---
task_key: T-E07-F08-003
epic_key: E07
feature_key: E07-F08
title: Make ValidateCustomFilename accessible for epic and feature creation
status: created
priority: 2
agent_type: backend
depends_on: ["T-E07-F08-001"]
---

# Task: Make ValidateCustomFilename accessible for epic and feature creation

## Objective

Ensure the existing `ValidateCustomFilename` function from the task creation flow is accessible to epic and feature creation commands, enabling consistent validation across all entity types.

## Context

**Why this task exists**: E07-F05 implemented robust filename validation for tasks in `internal/taskcreation/creator.go`. This feature requires the same validation logic for epics and features to ensure consistency and security. Rather than duplicating code, we need to make the existing function reusable.

**Design reference**:
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (lines 133-153, 383-401)
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (lines 203-243)

## What to Build

**Option 1 (Recommended)**: Export the existing function

1. Verify `ValidateCustomFilename` in `internal/taskcreation/creator.go` is already exported (starts with uppercase)
2. If not exported, rename to uppercase: `ValidateCustomFilename`
3. Document that this function is shared across task, epic, and feature creation
4. No code duplication required

**Option 2**: Move to shared utility package (if refactoring is needed)

1. Create `internal/validation/filename.go`
2. Move `ValidateCustomFilename` from `internal/taskcreation/creator.go` to new package
3. Update import statements in:
   - `internal/taskcreation/creator.go`
   - `internal/cli/commands/epic.go` (after T-E07-F08-004)
   - `internal/cli/commands/feature.go` (after T-E07-F08-005)

**Recommendation**: Use Option 1 unless the function is currently private or there's a strong architectural reason to move it.

## Success Criteria

- [ ] `ValidateCustomFilename` function is accessible from epic and feature command handlers
- [ ] Function signature remains unchanged: `ValidateCustomFilename(filename string, projectRoot string) (absPath string, relPath string, error)`
- [ ] All validation rules are preserved:
  - No absolute paths
  - Must have `.md` extension
  - No path traversal (`..`)
  - Must resolve within project boundaries
- [ ] Existing task creation tests still pass (no regression)
- [ ] Function can be imported by packages outside `internal/taskcreation`

## Validation Gates

1. **Import Test**:
   - Create a temporary test file in `internal/cli/commands/` that imports and calls `ValidateCustomFilename`
   - Verify the import compiles without errors
   - Delete the temporary test file

2. **Regression Test**:
   - Run existing task creation tests: `go test -v ./internal/taskcreation/...`
   - All tests pass without modification

3. **Documentation**:
   - Add godoc comment to `ValidateCustomFilename` indicating it's shared across entity types
   - Example: `// ValidateCustomFilename validates custom file paths for tasks, epics, and features`

## Dependencies

**Prerequisite Tasks**:
- T-E07-F08-001 (models must exist, but not strictly required for validation function)

**Blocks**:
- T-E07-F08-004 (epic creation needs access to validation)
- T-E07-F08-005 (feature creation needs access to validation)

## Implementation Notes

### Current Function Location

The function is in `internal/taskcreation/creator.go`. Check if it's already exported:

```bash
grep -n "func ValidateCustomFilename" internal/taskcreation/creator.go
```

If the output shows `func ValidateCustomFilename(` (uppercase V), it's already exported and accessible. No changes needed beyond documentation.

If the output shows `func validateCustomFilename(` (lowercase v), rename to uppercase.

### Validation Rules Reference

The function enforces these rules (from PRD Section 3.2):

| Rule | Requirement | Error Message |
|------|-------------|---------------|
| No Absolute Paths | Filename must be relative | `filename must be relative to project root, got absolute path: {path}` |
| Extension Required | Must have `.md` extension | `invalid file extension: {ext} (must be .md)` |
| No Path Traversal | Cannot contain `..` | `invalid path: contains '..' (path traversal not allowed)` |
| Within Project | Must resolve inside project root | `path validation failed: path resolves outside project root` |

### Testing Strategy

Verify the function is accessible:

```go
package commands_test

import (
    "testing"
    "github.com/yourusername/shark/internal/taskcreation"
)

func TestValidateCustomFilenameAccessibility(t *testing.T) {
    projectRoot := "/home/user/project"

    // Valid path
    absPath, relPath, err := taskcreation.ValidateCustomFilename("docs/spec.md", projectRoot)
    if err != nil {
        t.Errorf("Valid path rejected: %v", err)
    }

    // Invalid path (absolute)
    _, _, err = taskcreation.ValidateCustomFilename("/absolute/path.md", projectRoot)
    if err == nil {
        t.Error("Absolute path should be rejected")
    }
}
```

Run with: `go test -v ./internal/cli/commands/...`

## References

- **PRD Section**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (Section 3.2 - Validation Rules)
- **Backend Design**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (Section: Validation Logic)
- **Current Implementation**: `internal/taskcreation/creator.go` (existing function to be reused)
