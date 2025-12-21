# Feature Definition: Custom Filenames for Epics & Features

## Summary

Extend custom filename support to epic and feature creation, providing feature parity with the task filename feature. Users can specify arbitrary file locations for epics and features using `--filename` and `--force` flags, enabling flexible documentation organization across the project.

## User Stories

### US-1: Standard Workflow with Custom Locations
As a product manager, I want to create epics and features that live in custom locations (not the default `docs/plan/` hierarchy), so that I can organize epics alongside strategic documentation or roadmaps.

### US-2: Flexible Documentation Organization
As a technical writer, I want to place feature specifications in shared documentation areas or custom folders, so that I can maintain documentation consistency across multiple projects.

### US-3: Consistent CLI Across All Entities
As a developer using the shark CLI, I want epics, features, and tasks to have identical `--filename` and `--force` flag behavior, so that I don't have to learn different commands for different entity types.

## Acceptance Criteria

- **AC-1**: `shark epic create --filename=<path>` accepts a custom relative path and creates the epic file at that location
- **AC-2**: `shark feature create --filename=<path>` accepts a custom relative path and creates the feature file at that location
- **AC-3**: Both commands validate filenames using the same rules as tasks:
  - Relative paths only (no absolute paths)
  - Must have `.md` extension
  - No path traversal (`..`) allowed
  - Must be within project boundaries
- **AC-4**: Both commands support `--force` flag to reassign a file if already claimed by another epic/feature
- **AC-5**: Without `--filename`, both commands use default locations: `docs/plan/<epic-key>/epic.md` and `docs/plan/<epic-key>/<feature-key>/feature.md`
- **AC-6**: File collision detection prevents multiple entities from claiming the same file (unless `--force` is used)
- **AC-7**: Error messages clearly indicate when a file is already claimed and suggest using `--force`
- **AC-8**: All validation and behavior reuses the existing `ValidateCustomFilename` logic from tasks (no duplication)

## MVP Scope

### In Scope
- Add `--filename` flag to `shark epic create`
- Add `--filename` flag to `shark feature create`
- Add `--force` flag to both commands
- Reuse `ValidateCustomFilename` from `taskcreation` package
- Implement file collision detection for epics/features
- Support file reassignment with `--force`
- Update help text and documentation

### Out of Scope
- Changing default locations for epics/features
- Custom templates for epics/features (defer to future enhancement)
- Batch operations or bulk filename assignment
- Modifying existing epic/feature files (this is for creation only)

## Value Proposition

**Why This Matters:**
1. **Consistency**: Epics and features now have feature parity with tasks - all support custom filenames
2. **Flexibility**: Users can organize documentation according to project conventions without rigid folder structures
3. **Enablement**: Supports use cases like shared documentation areas, roadmaps, and strategic planning documents
4. **Low Risk**: Reuses proven validation logic and patterns from the task filename feature
5. **No Breaking Changes**: Default behavior unchanged; custom filenames are purely additive

## Dependencies

- Task filename feature (E07-F05) - COMPLETED
  - Provides `ValidateCustomFilename` function to reuse
  - Provides collision detection and force reassignment patterns
  - No new dependencies required

## Success Metrics

- All three entity types (epic, feature, task) accept `--filename` with identical semantics
- Users can place epics/features in any `.md` file within the project
- File collision detection prevents accidental overwrites
- No documentation or tutorial updates needed beyond flag descriptions

## Technical Notes

### Implementation Pattern (from Tasks)

```go
// Input structure
type CreateEpicInput struct {
    Title       string
    Description string
    Filename    string  // Custom filepath (optional)
    Force       bool    // Force reassignment if claimed
}

// Validation
absPath, relPath, err := c.ValidateCustomFilename(input.Filename, projectRoot)

// Collision detection
existingEpic, err := epicRepo.GetByFilePath(ctx, relPath)
if existingEpic != nil && !input.Force {
    return fmt.Errorf("file is already claimed by epic %s", existingEpic.Key)
}

// Reassignment if --force
if existingEpic != nil && input.Force {
    epicRepo.UpdateFilePath(ctx, existingEpic.Key, nil)
}
```

### Validation Rules (Reused)

The `ValidateCustomFilename` function already enforces:
1. No absolute paths
2. `.md` extension required
3. No `..` path traversal
4. Within project boundaries
5. Non-empty filename after normalization

## Files to Modify

- `internal/cli/commands/epic.go` - Add `--filename` and `--force` flags to create command
- `internal/cli/commands/feature.go` - Add `--filename` and `--force` flags to create command
- `internal/repository/epic_repository.go` - Add `GetByFilePath` and `UpdateFilePath` methods
- `internal/repository/feature_repository.go` - Add `GetByFilePath` and `UpdateFilePath` methods
- CLI help text documentation

