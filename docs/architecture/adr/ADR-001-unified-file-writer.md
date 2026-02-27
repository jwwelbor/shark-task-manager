# ADR-001: Unified File Writer Package

**Status:** Accepted
**Date:** 2026-01-12
**Deciders:** Development Team
**Technical Story:** Consolidate duplicate file writing logic across epic, feature, and task creation

---

## Context and Problem Statement

Prior to this refactoring, epic, feature, and task creation commands each implemented their own file writing logic with significant code duplication:

- **Epic creation** (`internal/cli/commands/epic.go`): ~25 lines of file handling
- **Feature creation** (`internal/cli/commands/feature.go`): ~17 lines of similar code
- **Task creation** (`internal/taskcreation/creator.go`): ~40 lines including `writeFileExclusive()` method

This duplication led to:
- Maintenance burden (changes needed in 3 places)
- Inconsistent behavior (tasks used atomic writes, epics/features didn't)
- Code bloat (~80+ lines of duplicate file handling)
- Risk of bugs due to divergent implementations

**Key Question:** How can we consolidate file writing operations while maintaining existing behavior and improving code quality?

---

## Decision Drivers

- **Reduce code duplication** - DRY principle
- **Maintain existing behavior** - No breaking changes
- **Improve consistency** - Same logic for all entity types
- **Enable atomic writes** - Prevent race conditions
- **Simplify maintenance** - Single point of change
- **Comprehensive testing** - High test coverage for reliability

---

## Considered Options

### Option 1: Status Quo (Keep Duplicate Code)
**Pros:**
- No migration effort required
- No risk of introducing bugs

**Cons:**
- Continued maintenance burden
- Inconsistent implementations
- Technical debt accumulation
- Difficult to add new features

### Option 2: Shared Helper Function
**Pros:**
- Simple refactoring
- Minimal code changes

**Cons:**
- Still scattered across multiple files
- Hard to extend with new features
- Limited encapsulation

### Option 3: Unified File Operations Package (Selected)
**Pros:**
- Clear separation of concerns
- Single point of maintenance
- Easy to test in isolation
- Extensible for future features
- Consistent behavior across all entity types

**Cons:**
- Upfront migration effort
- Requires careful testing to avoid regressions

---

## Decision Outcome

**Chosen Option:** Option 3 - Create `internal/fileops` package with `EntityFileWriter`

### Implementation

Created a new package with the following components:

**Core Types:**
```go
type EntityFileWriter struct {}

type WriteOptions struct {
    Content         []byte
    ProjectRoot     string
    FilePath        string
    Force           bool
    CreateIfMissing bool  // Task-specific validation
    Verbose         bool
    EntityType      string
    UseAtomicWrite  bool
    Logger          func(string)
}

type WriteResult struct {
    Written      bool
    Linked       bool
    AbsolutePath string
    RelativePath string
}
```

**Key Method:**
```go
func (w *EntityFileWriter) WriteEntityFile(opts WriteOptions) (*WriteResult, error)
```

### Migration Phases

**Phase 1: Create Package** ✅
- Created `internal/fileops/writer.go` (214 lines)
- Created comprehensive test suite (594 lines, 26 tests)
- Achieved 87.1% test coverage
- All tests passing

**Phase 2: Epic Migration** ✅
- Updated `internal/cli/commands/epic.go`
- Reduced code from ~27 lines to ~19 lines
- Preserved all existing behavior
- All tests passing

**Phase 3: Feature Migration** ✅
- Updated `internal/cli/commands/feature.go`
- Reduced code from ~17 lines to ~14 lines
- Maintained file linking behavior
- All tests passing

**Phase 4: Task Migration** ✅
- Updated `internal/taskcreation/creator.go`
- Deleted `writeFileExclusive()` method (24 lines)
- Added verbose logging support
- Preserved `--create` flag behavior
- All tests passing

### Consequences

**Positive:**
- ✅ Eliminated ~50+ lines of duplicate code
- ✅ All entity types use identical file writing logic
- ✅ Atomic write protection available for all entities
- ✅ Single point of maintenance for file operations
- ✅ Comprehensive test coverage (87.1%)
- ✅ Consistent error handling and logging
- ✅ Easier to add features (backups, validation, etc.)

**Negative:**
- ⚠️ Initial migration effort (~6 hours development + testing)
- ⚠️ Additional package to maintain
- ⚠️ Learning curve for new developers (minimal)

**Neutral:**
- File behavior unchanged from user perspective
- No breaking changes to CLI interface
- Verbose logging works consistently across all commands

---

## Validation

### Test Coverage
- **Unit tests:** 26 tests (13 positive, 13 negative)
- **Coverage:** 87.1% (target: >85%)
- **Test time:** ~0.12s (fast)

### Test Categories
1. **Atomic write protection** - Prevents race conditions
2. **File existence handling** - Links vs creates
3. **Path resolution** - Absolute and relative paths
4. **Directory creation** - Parent directories
5. **Error handling** - Permission denied, invalid paths
6. **Verbose logging** - Optional debug output
7. **Force overwrite** - Update existing files
8. **Task-specific behavior** - CreateIfMissing validation

### Integration Testing
- All existing CLI tests pass without modification
- Manual smoke tests verified for epic/feature/task creation
- Verbose logging tested and working
- Custom file paths tested and working
- File linking behavior preserved

---

## Links

**Related Documentation:**
- Implementation Summary: `dev-artifacts/2026-01-11-unified-file-writer/IMPLEMENTATION_SUMMARY.md`
- Architecture Proposal: `dev-artifacts/2026-01-11-unified-file-writer/architecture-proposal.md`
- API Reference: `dev-artifacts/2026-01-11-unified-file-writer/api-reference.md`
- Migration Guide: `dev-artifacts/2026-01-11-unified-file-writer/migration-guide.md`

**Modified Files:**
- `internal/fileops/writer.go` (new)
- `internal/fileops/writer_test.go` (new)
- `internal/fileops/doc.go` (new)
- `internal/cli/commands/epic.go` (modified)
- `internal/cli/commands/feature.go` (modified)
- `internal/taskcreation/creator.go` (modified)
- `CLAUDE.md` (documented)

**Commits:**
- Phase 1: `b8f4a2c` - Create fileops package with tests
- Phase 2: `9c1d5e8` - Migrate epic creation
- Phase 3: `1a5d3f7` - Migrate feature creation
- Phase 4: `213266b` - Migrate task creation and remove writeFileExclusive

---

## Future Considerations

### Potential Enhancements
1. **File backups** - Automatic backup before overwrite
2. **Content validation** - Verify markdown frontmatter
3. **Dry-run mode** - Preview without writing
4. **Metrics collection** - Track file operations
5. **Concurrent writes** - Better handling of parallel operations
6. **Rollback support** - Undo failed operations

### Extension Points
The design allows easy extension through:
- Additional WriteOptions fields
- Custom logger implementations
- Entity-specific validations
- Pre/post write hooks

---

## Notes

This ADR represents a successful refactoring that improves code quality while maintaining backward compatibility. The phased migration approach minimized risk, and comprehensive testing ensured no regressions were introduced.

The unified file writer is now the standard approach for all file operations in the shark task manager, providing a solid foundation for future enhancements.
