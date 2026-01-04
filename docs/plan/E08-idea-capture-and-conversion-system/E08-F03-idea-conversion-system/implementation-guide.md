# E08-F03 Implementation Guide: Idea Conversion System

## Overview

This feature implements conversion of lightweight ideas into structured entities (epic, feature, task). When an idea is converted, it:
1. Changes the idea status to "converted"
2. Records what it was converted to (type, key, timestamp)
3. Creates the target entity with copied metadata
4. Returns the new entity's key

## Architecture

### Repository Layer Changes

**IdeaRepository** needs new conversion tracking method:
```go
// MarkAsConverted updates an idea's conversion tracking fields
func (r *IdeaRepository) MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error
```

This method:
- Updates idea status to "converted"
- Sets converted_to_type, converted_to_key, converted_at
- All in a single UPDATE statement
- Returns error if idea not found

### CLI Command Structure

New command hierarchy under `idea`:
```
shark idea convert <idea-key> epic
shark idea convert <idea-key> feature --epic=E##
shark idea convert <idea-key> task --epic=E## --feature=F##
```

Implementation location: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/idea.go`

Add subcommand `ideaConvertCmd` with three subcommands:
- `ideaConvertEpicCmd`
- `ideaConvertFeatureCmd`
- `ideaConvertTaskCmd`

### Conversion Workflow (all types)

1. **Validate idea exists and is not already converted**
   - Fetch idea by key
   - Check status is NOT "converted"
   - Return error if already converted

2. **Create target entity**
   - Copy title from idea
   - Copy description from idea (if present)
   - Copy related_docs from idea (if present, parse JSON array)
   - Use existing creation logic (EpicRepository.Create, FeatureRepository.Create, TaskRepository.Create)
   - Auto-generate key using existing logic

3. **Mark idea as converted**
   - Call IdeaRepository.MarkAsConverted
   - Pass entity type and key

4. **Return success with new key**
   - JSON output: `{"idea_key": "I-2026-01-01-01", "converted_to": "E15", "type": "epic"}`
   - Human output: `Idea I-2026-01-01-01 converted to epic E15`

### Error Handling

All conversion commands must handle:
- Idea not found (exit code 1)
- Idea already converted (exit code 3, message: "Idea already converted to {type} {key} on {date}")
- Target entity creation fails (exit code 2)
- Conversion tracking update fails (exit code 2)

## Task Implementation Details

### T-E08-F03-001: Convert Idea to Epic

**Command**: `shark idea convert <idea-key> epic`

**Flow**:
1. Parse idea key from args[0]
2. Fetch idea from IdeaRepository
3. Validate idea not already converted
4. Create epic using EpicRepository.Create:
   - Title: idea.Title
   - Description: idea.Description (if present)
   - RelatedDocuments: parse idea.RelatedDocs JSON (if present)
   - Priority: "medium" (default)
   - BusinessValue: "medium" (default)
5. Mark idea as converted (type="epic", key=newEpicKey)
6. Output result

**Test Cases** (write FIRST):
- Convert valid idea to epic (success)
- Convert already-converted idea (error)
- Convert non-existent idea (error)
- Convert idea with description and related docs (metadata copied)
- JSON output format validation

### T-E08-F03-002: Convert Idea to Feature

**Command**: `shark idea convert <idea-key> feature --epic=E##`

**Flags**:
- `--epic` (required): Target epic key

**Flow**:
1. Parse idea key from args[0]
2. Parse epic key from --epic flag (required, error if missing)
3. Validate epic exists (fetch from EpicRepository)
4. Fetch idea from IdeaRepository
5. Validate idea not already converted
6. Create feature using FeatureRepository.Create:
   - EpicID: from validated epic
   - Title: idea.Title
   - Description: idea.Description (if present)
   - RelatedDocuments: parse idea.RelatedDocs JSON (if present)
   - ExecutionOrder: auto-assigned (next available in epic)
   - Status: "draft" (default)
7. Mark idea as converted (type="feature", key=newFeatureKey)
8. Output result

**Test Cases** (write FIRST):
- Convert valid idea to feature (success)
- Convert without --epic flag (error)
- Convert with non-existent epic (error)
- Convert already-converted idea (error)
- Convert idea with metadata (copied correctly)
- JSON output format validation

### T-E08-F03-003: Convert Idea to Task

**Command**: `shark idea convert <idea-key> task --epic=E## --feature=F##`

**Flags**:
- `--epic` (required): Target epic key
- `--feature` (required): Target feature key

**Flow**:
1. Parse idea key from args[0]
2. Parse epic and feature keys from flags (both required)
3. Validate epic exists
4. Validate feature exists and belongs to epic
5. Fetch idea from IdeaRepository
6. Validate idea not already converted
7. Create task using TaskRepository.Create:
   - EpicKey: from flag
   - FeatureKey: from flag
   - Title: idea.Title
   - Description: idea.Description (if present)
   - RelatedDocs: parse idea.RelatedDocs JSON (if present)
   - Priority: idea.Priority (if present, else default 5)
   - Status: "todo" (default)
   - AgentType: "general" (default)
8. Mark idea as converted (type="task", key=newTaskKey)
9. Output result

**Test Cases** (write FIRST):
- Convert valid idea to task (success)
- Convert without required flags (error)
- Convert with non-existent epic/feature (error)
- Convert with feature not in epic (error)
- Convert already-converted idea (error)
- Convert idea with priority (preserved in task)
- JSON output format validation

### T-E08-F03-004: Conversion Tracking and History

This task enhances the system to display conversion information.

**Changes Required**:

1. **IdeaRepository.MarkAsConverted** (new method):
```go
func (r *IdeaRepository) MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error {
    query := `
        UPDATE ideas
        SET status = 'converted',
            converted_to_type = ?,
            converted_to_key = ?,
            converted_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    result, err := r.db.ExecContext(ctx, query, convertedToType, convertedToKey, ideaID)
    // ... error handling
}
```

2. **Update `shark idea get` command** to display conversion info:
   - If idea.Status == "converted", show:
     - "Converted to: {type} {key}"
     - "Converted at: {timestamp}"
   - In JSON output, include conversion fields

3. **Add `--original-idea` flag to entity get commands** (optional enhancement):
   - `shark epic get E15 --original-idea`: Show linked idea if epic was converted from idea
   - Requires reverse lookup: search ideas WHERE converted_to_type='epic' AND converted_to_key='E15'

**Test Cases** (write FIRST):
- MarkAsConverted updates all fields correctly
- MarkAsConverted returns error for non-existent idea
- `shark idea get` displays conversion info for converted idea
- `shark idea get` JSON output includes conversion fields
- Reverse lookup finds original idea (if implementing --original-idea)

## TDD Requirements

### Test Order (Critical)
1. Write repository tests FIRST (using real database with cleanup)
2. Write CLI command tests SECOND (using mocked repositories)
3. Implement repository methods
4. Implement CLI commands
5. Run all tests (must pass before proceeding)
6. Manual verification

### Repository Tests
- Location: `/home/jwwelbor/projects/shark-task-manager/internal/repository/idea_repository_test.go`
- Use real database: `test.GetTestDB()`
- Clean up before each test: `DELETE FROM ideas WHERE key LIKE 'TEST-%'`
- Seed test data: Create test idea, epic, feature as needed
- Cleanup after test: Delete created records

### CLI Tests
- Location: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/idea_test.go`
- Use MOCKED repositories (DO NOT use real database)
- Create mock IdeaRepository, EpicRepository, FeatureRepository, TaskRepository
- Test command logic, argument parsing, output formatting
- Verify mock methods called with correct parameters

## Quality Gates

Each task must meet these criteria before marking complete:
1. All tests written FIRST and passing
2. Repository methods implemented and tested
3. CLI commands implemented and tested
4. Manual verification successful:
   - Create test idea
   - Convert to each entity type
   - Verify conversion tracking
   - Verify metadata copied correctly
5. JSON output validated
6. Error cases tested

## Dependencies Between Tasks

**Execution Order**:
- T-E08-F03-004 should be implemented FIRST (provides MarkAsConverted method)
- T-E08-F03-001, T-E08-F03-002, T-E08-F03-003 can run in parallel after T-E08-F03-004

**Rationale**: All three conversion commands need `IdeaRepository.MarkAsConverted`, which is added in T-E08-F03-004. Implementing this first unblocks the other three tasks.

## Reference Code Locations

- Idea model: `/home/jwwelbor/projects/shark-task-manager/internal/models/idea.go`
- IdeaRepository: `/home/jwwelbor/projects/shark-task-manager/internal/repository/idea_repository.go`
- Idea CLI commands: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/idea.go`
- Epic creation: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/epic.go`
- Feature creation: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/feature.go`
- Task creation: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go`

## Success Criteria

Feature is complete when:
1. All 4 tasks marked complete in shark
2. All tests passing (`make test`)
3. Manual end-to-end verification:
   ```bash
   # Create test idea
   ./bin/shark idea create "Test Idea for Conversion" --description="Testing conversion system"

   # Convert to epic
   ./bin/shark idea convert I-2026-01-01-01 epic --json

   # Create another idea
   ./bin/shark idea create "Test Feature Idea"

   # Convert to feature
   ./bin/shark idea convert I-2026-01-01-02 feature --epic=E15 --json

   # Create another idea
   ./bin/shark idea create "Test Task Idea" --priority=8

   # Convert to task
   ./bin/shark idea convert I-2026-01-01-03 task --epic=E15 --feature=E15-F01 --json

   # Verify conversion tracking
   ./bin/shark idea get I-2026-01-01-01 --json
   ```
4. Code committed and pushed
5. Feature marked complete in shark
