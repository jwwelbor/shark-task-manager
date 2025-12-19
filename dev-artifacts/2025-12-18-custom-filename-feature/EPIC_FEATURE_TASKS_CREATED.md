# E07-F01: Custom Filenames for Epics and Features - Implementation Tasks

**Date Created**: 2025-12-18
**Feature**: E07-F01-custom-filenames-for-epics-and-features
**Status**: Tasks Generated and Ready for Development

---

## Executive Summary

Generated 10 comprehensive implementation tasks for adding custom filename support to epic and feature creation commands. Tasks follow the exact same pattern as the successful task custom filename implementation (E07-F05 reference).

---

## Generated Tasks

### Phase 1: Repository Layer
**Task**: T-E07-F01-001 - Add GetByFilePath and UpdateFilePath methods
- Location: `docs/tasks/todo/T-E07-F01-001.md`
- Priority: 1 (Highest)
- Agent: backend
- Status: todo

**What It Does**:
- Adds repository methods to EpicRepository and FeatureRepository
- Methods for collision detection (GetByFilePath)
- Methods for file path updates and force reassignment (UpdateFilePath)
- Adds database indexes for performance

**Reference Implementation**: TaskRepository methods in internal/repository/task_repository.go:170-237

---

### Phase 2: Validation Logic
**Task**: T-E07-F01-002 - Extract ValidateCustomFilename to shared validation module
- Location: `docs/tasks/todo/T-E07-F01-002.md`
- Priority: 2
- Agent: backend
- Status: todo

**What It Does**:
- Moves ValidateCustomFilename from task creator to shared module
- Enables reuse by epic, feature, and task creators
- Zero code duplication across all entity types
- Comprehensive validation: absolute paths, traversal, extensions, boundaries

**Reference Implementation**: creator.go:229-273

---

### Phase 3: Data Models
**Task**: T-E07-F01-003 - Add FilePath field to Epic and Feature models
- Location: `docs/tasks/todo/T-E07-F01-003.md`
- Priority: 3
- Agent: backend
- Status: todo

**What It Does**:
- Adds FilePath *string field to Epic model
- Adds FilePath *string field to Feature model
- Proper database mapping with `db:"file_path"` tags
- Nullable fields for backward compatibility

---

### Phase 4: Epic Creator Function
**Task**: T-E07-F01-004 - Implement epic creator function with custom filename support
- Location: `docs/tasks/todo/T-E07-F01-004.md`
- Priority: 4
- Agent: backend
- Status: todo

**What It Does**:
- Creates CreateEpic function with custom filename support
- Implements collision detection (prevents overwriting other epic files)
- Implements force reassignment (--force flag)
- Handles file association (existing and new files)
- Atomic transactions for multi-step operations

**Reference Implementation**: Creator.CreateTask in creator.go:65-198

---

### Phase 5: Feature Creator Function
**Task**: T-E07-F01-005 - Implement feature creator function with custom filename support
- Location: `docs/tasks/todo/T-E07-F01-005.md`
- Priority: 5
- Agent: backend
- Status: todo

**What It Does**:
- Creates CreateFeature function mirroring epic creator
- Same collision detection, force reassignment, file handling
- Atomic transactions for data consistency
- Clear error messages matching epic creator

---

### Phase 6: Epic CLI Integration
**Task**: T-E07-F01-006 - Add --filename and --force flags to shark epic create command
- Location: `docs/tasks/todo/T-E07-F01-006.md`
- Priority: 6
- Agent: backend
- Status: todo

**What It Does**:
- Adds --filename string flag to `shark epic create` command
- Adds --force boolean flag to `shark epic create` command
- Parses flags and passes to epic creator
- Gets project root and passes to creator
- Handles all error cases with clear messaging

**Reference Implementation**: task.go epic create integration

---

### Phase 7: Feature CLI Integration
**Task**: T-E07-F01-007 - Add --filename and --force flags to shark feature create command
- Location: `docs/tasks/todo/T-E07-F01-007.md`
- Priority: 7
- Agent: backend
- Status: todo

**What It Does**:
- Adds --filename and --force flags to `shark feature create`
- Same implementation pattern as epic CLI integration
- Passes flags to feature creator function
- Complete error handling

---

### Phase 8: Database Schema
**Task**: T-E07-F01-008 - Add file_path columns and indexes to epics and features tables
- Location: `docs/tasks/todo/T-E07-F01-008.md`
- Priority: 8
- Agent: backend
- Status: todo

**What It Does**:
- Adds file_path TEXT column to epics table
- Adds file_path TEXT column to features table
- UNIQUE constraints prevent duplicate file assignments
- Performance indexes: idx_epics_file_path, idx_features_file_path
- Collision detection queries complete in < 100ms

---

### Phase 9: Documentation
**Task**: T-E07-F01-009 - Update CLI documentation for epic and feature custom filename support
- Location: `docs/tasks/todo/T-E07-F01-009.md`
- Priority: 9
- Agent: documentation
- Status: todo

**What It Does**:
- Updates docs/CLI_REFERENCE.md with custom filename sections
- Documents --filename and --force flags for both commands
- Provides practical usage examples
- Explains validation rules and error messages
- Updates CLAUDE.md command references

---

### Phase 10: Testing & Verification
**Task**: T-E07-F01-010 - Write integration tests and perform end-to-end verification
- Location: `docs/tasks/todo/T-E07-F01-010.md`
- Priority: 10
- Agent: qa
- Status: todo

**What It Does**:
- Unit tests for repository methods (GetByFilePath, UpdateFilePath)
- Validation tests for ValidateCustomFilename across all entities
- Integration tests for epic create CLI command
- Integration tests for feature create CLI command
- Tests for all edge cases and validation failures
- Manual end-to-end testing
- Comprehensive test coverage (90%+)

**Reference Implementation**: dev-artifacts/2025-12-18-custom-filename-feature/PROGRESS.md Phase 7

---

## Implementation Dependencies

```
T-E07-F01-001 (Repository Methods) ─┐
                                     ├─> T-E07-F01-003 (Models)
T-E07-F01-002 (Validation) ──────────┤   │
                                     │   │
                         ┌───────────┴─┬─┘
                         │             │
T-E07-F01-004 (Epic Creator) ────┐    │
                                 │    │
T-E07-F01-005 (Feature Creator) ─┤    │
                                 │    │
                         ┌───────┴────┴────────┐
                         │                    │
T-E07-F01-006 (Epic CLI) ──────────────────────┤
                                               │
T-E07-F01-007 (Feature CLI) ─────────────────────┤
                                                 │
T-E07-F01-008 (Database Schema) ─────────────────┤
                                                 │
T-E07-F01-009 (Documentation) ────────────────────┤
                                                 │
                            ┌────────────────────┘
                            │
T-E07-F01-010 (Testing) <───┘
```

**Execution Order**:
1. Start T-E07-F01-001, T-E07-F01-002 (Foundation)
2. Complete T-E07-F01-003, T-E07-F01-008 (Models + Schema)
3. Implement T-E07-F01-004, T-E07-F01-005 (Creators)
4. Add T-E07-F01-006, T-E07-F01-007 (CLI Integration)
5. Update T-E07-F01-009 (Documentation)
6. Execute T-E07-F01-010 (Testing)

---

## Key Design Decisions

### Code Reuse
- ValidateCustomFilename extracted to shared module (no duplication)
- Epic and feature creators follow same pattern as task creator
- CLI integration mirrors task create command structure

### Backward Compatibility
- Without --filename: epics use `docs/plan/{epic-key}/epic.md`
- Without --filename: features use `docs/plan/{epic-key}/{feature-key}/feature.md`
- Existing workflows completely unchanged (opt-in feature)

### Validation & Security
- Same validation rules for all entities
- Path traversal prevention (`..`)
- Absolute path rejection
- `.md` extension requirement
- Project boundary validation

### Error Handling
- Clear error messages matching task implementation
- Collision detection prevents data loss
- Force flag requires explicit opt-in
- Helpful output guides users

---

## Testing Strategy

### Unit Tests (Per Task 10)
- Repository methods (GetByFilePath, UpdateFilePath)
- ValidateCustomFilename across all entities
- Path normalization and special character handling

### Integration Tests (Per Task 10)
- Epic create with custom filename
- Feature create with custom filename
- Collision detection scenarios
- Force reassignment behavior
- File creation and association
- Validation failure cases

### End-to-End (Manual)
- CLI commands work with custom filenames
- Files created/associated correctly
- Database records match files
- Collision detection prevents overwrites
- Force reassignment updates both records
- Default behavior unchanged

---

## File Locations

### Feature Documentation
- PRD: `docs/plan/E07-enhancements/E07-F01-custom-filenames-for-epics-and-features/prd.md` (658 lines)

### Implementation Tasks
```
docs/tasks/todo/T-E07-F01-001.md - Repository Methods
docs/tasks/todo/T-E07-F01-002.md - Validation Logic
docs/tasks/todo/T-E07-F01-003.md - Data Models
docs/tasks/todo/T-E07-F01-004.md - Epic Creator
docs/tasks/todo/T-E07-F01-005.md - Feature Creator
docs/tasks/todo/T-E07-F01-006.md - Epic CLI
docs/tasks/todo/T-E07-F01-007.md - Feature CLI
docs/tasks/todo/T-E07-F01-008.md - Database Schema
docs/tasks/todo/T-E07-F01-009.md - Documentation
docs/tasks/todo/T-E07-F01-010.md - Testing
```

---

## Reference Implementation

All tasks reference the successful task custom filename implementation:
- **Progress file**: `dev-artifacts/2025-12-18-custom-filename-feature/PROGRESS.md`
- **Plan file**: `/home/jwwelbor/.claude/plans/ancient-purring-tower.md`
- **Source code**: `internal/taskcreation/creator.go`, `internal/repository/task_repository.go`, etc.

Tasks are designed to follow the exact same patterns, ensuring consistency and leveraging proven approaches.

---

## Expected Outcomes

Upon completion of all 10 tasks:

✅ Epic create command supports --filename and --force flags
✅ Feature create command supports --filename and --force flags
✅ Collision detection prevents file overwrites
✅ Force reassignment allows clearing old associations
✅ Validation prevents security issues (path traversal, absolute paths)
✅ File association works (existing and new files)
✅ All tests passing with 90%+ coverage
✅ Documentation complete
✅ Backward compatible (default behavior unchanged)

---

## Next Steps

1. **Start implementation**: Begin with T-E07-F01-001 (Repository Methods)
2. **Follow dependency order**: Tasks must be completed in sequence per dependency graph
3. **Reference implementation**: Use task filename feature as pattern
4. **Testing**: Run `make test` after each phase
5. **Manual verification**: Test CLI commands with real project

---

## Token Usage

- Generated: 10 comprehensive implementation tasks
- Included: Full requirements, implementation steps, acceptance criteria, testing strategy
- Reference: Task filename implementation pattern (PROGRESS.md)
- PRD: 658-line comprehensive feature specification

