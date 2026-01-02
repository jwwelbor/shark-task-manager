# E07-F12 Task Breakdown Summary

**Date**: 2026-01-01
**Tech Director**: Claude Opus 4.5
**Feature**: E07-F12 - Parameter Consistency Across Create and Update Commands

---

## Overview

Phase 1 is **COMPLETE**. This document provides the breakdown for Phases 2 and 3.

### Current Status

- **Phase 1**: COMPLETE ✅
  - Missing flags added to epic and feature create commands
  - All tests passing, 100% backward compatible
  - See: `phase1-implementation-complete.md`

- **Phase 2**: TODO (5 tasks created)
  - Create shared modules for DRY architecture
  - Foundation for refactoring

- **Phase 3**: TODO (4 tasks created)
  - Refactor commands to use shared modules
  - Achieve 20%+ code reduction

---

## Task List

### Phase 2: Create Shared Modules (Priority 8)

All Phase 2 tasks must be completed before Phase 3 can begin.

#### T-E07-F12-001: Create shared_flags.go
- **Agent**: backend
- **Goal**: Flag registration module with composable flag sets
- **Key Functions**:
  - `AddFlagSet(cmd, flagSet, opts...)`
  - Flag sets: metadata, path, epic_status, feature_status, custom_key
  - Options: WithDefaults, WithRequired
- **Tests**: 7 test cases, 95%+ coverage
- **Dependencies**: None

#### T-E07-F12-002: Create validators.go
- **Agent**: backend
- **Goal**: Centralized validation logic
- **Key Functions**:
  - `ValidateCustomPath(cmd, flagName)`
  - `ValidateCustomFilename(cmd, flagName, projectRoot)`
  - `ValidateNoSpaces(key, entityType)`
  - `ValidateStatus(status, entityType)`
  - `ValidatePriority(priority, entityType)`
- **Tests**: 11 test cases, 95%+ coverage
- **Dependencies**: None

#### T-E07-F12-003: Create file_assignment.go
- **Agent**: backend
- **Goal**: File collision detection and reassignment
- **Key Functions**:
  - `DetectFileCollision(ctx, filePath, epicRepo, featureRepo)`
  - `HandleFileReassignment(ctx, collision, force, epicRepo, featureRepo)`
  - `CreateBackupIfForce(force, dbPath, operation)`
- **Tests**: 7 test cases, 90%+ coverage
- **Dependencies**: None

#### T-E07-F12-004: Create status_priority.go
- **Agent**: backend
- **Goal**: Status/priority parsing with defaults
- **Key Functions**:
  - `ParseEpicStatus(cmd, defaultStatus)`
  - `ParseEpicPriority(cmd, defaultPriority)`
  - `ParseEpicBusinessValue(cmd)`
  - `ParseFeatureStatus(cmd, defaultStatus)`
- **Tests**: 7 test cases, 95%+ coverage
- **Dependencies**: None

#### T-E07-F12-005: Write comprehensive tests for all Phase 2 modules
- **Agent**: backend
- **Goal**: Achieve 90%+ test coverage for all shared modules
- **Test Files**:
  - `shared_flags_test.go`
  - `validators_test.go`
  - `file_assignment_test.go`
  - `status_priority_test.go`
- **Coverage Target**: 90%+ for all Phase 2 modules
- **Dependencies**: T-E07-F12-001, T-E07-F12-002, T-E07-F12-003, T-E07-F12-004

---

### Phase 3: Refactor Commands (Priority 7)

Phase 3 tasks depend on Phase 2 completion.

#### T-E07-F12-006: Refactor epic.go to use shared modules
- **Agent**: backend
- **Goal**: Replace duplicate code in epic.go
- **Changes**:
  - Replace flag registration with `AddFlagSet` calls
  - Replace validation with shared validators
  - Replace collision detection with shared functions
  - Remove duplicate `backupDatabaseOnForce` function
- **Target**: 20%+ code reduction in epic.go
- **Dependencies**: All Phase 2 tasks (T-E07-F12-001 through T-E07-F12-005)

#### T-E07-F12-007: Refactor feature.go to use shared modules
- **Agent**: backend
- **Goal**: Replace duplicate code in feature.go
- **Changes**:
  - Replace flag registration with `AddFlagSet` calls
  - Replace validation with shared validators
  - Replace collision detection with shared functions
  - Remove duplicate `backupDatabaseOnForceFeature` function
- **Target**: 20%+ code reduction in feature.go
- **Dependencies**: All Phase 2 tasks (T-E07-F12-001 through T-E07-F12-005)

#### T-E07-F12-008: Update tests for refactored commands
- **Agent**: backend
- **Goal**: Update tests to work with refactored code
- **Changes**:
  - Update `epic_create_test.go`
  - Update `epic_update_test.go`
  - Update `feature_update_test.go`
  - Ensure all existing tests still pass
- **Coverage Target**: Maintain or improve existing coverage
- **Dependencies**: T-E07-F12-006, T-E07-F12-007

#### T-E07-F12-009: Verify backward compatibility and run full regression testing
- **Agent**: testing
- **Goal**: Ensure 100% backward compatibility
- **Testing**:
  - Run full test suite (`make test`)
  - Manual testing of all command combinations
  - Verify default behavior unchanged
  - Verify all existing flags work
  - Performance testing (no regression)
- **Deliverable**: Regression test report
- **Dependencies**: T-E07-F12-008

---

## Success Metrics

### Code Quality

| Metric | Before | Target |
|--------|--------|--------|
| Lines in epic.go | ~1500 | <1200 (-20%) |
| Lines in feature.go | ~1600 | <1300 (-20%) |
| Duplicate code blocks | 5+ | 0 |
| Test coverage (commands) | ~75% | >85% |
| Test coverage (shared modules) | N/A | >90% |

### Functional

| Metric | Target |
|--------|--------|
| Backward compatibility | 100% |
| Existing tests passing | 100% |
| New features working | 100% |
| Command execution time | No regression |

---

## Implementation Sequence

### Recommended Order

1. **Phase 2a**: Create shared modules (T-E07-F12-001 through T-E07-F12-004)
   - Can be done in parallel
   - No dependencies on each other
   - DO NOT modify epic.go or feature.go

2. **Phase 2b**: Write comprehensive tests (T-E07-F12-005)
   - Depends on Phase 2a completion
   - Ensures all modules work correctly
   - Achieve 90%+ coverage

3. **Phase 3a**: Refactor commands (T-E07-F12-006, T-E07-F12-007)
   - Can be done in parallel
   - Both depend on Phase 2 completion
   - Update command files to use shared modules

4. **Phase 3b**: Update tests (T-E07-F12-008)
   - Depends on Phase 3a completion
   - Ensure refactored code works

5. **Phase 3c**: Regression testing (T-E07-F12-009)
   - Depends on Phase 3b completion
   - Final validation before feature completion

---

## Key Dependencies

```
Phase 2: Create Shared Modules
├── T-E07-F12-001 (shared_flags.go)      [no deps]
├── T-E07-F12-002 (validators.go)        [no deps]
├── T-E07-F12-003 (file_assignment.go)   [no deps]
├── T-E07-F12-004 (status_priority.go)   [no deps]
└── T-E07-F12-005 (comprehensive tests)  [depends on 001-004]

Phase 3: Refactor Commands
├── T-E07-F12-006 (refactor epic.go)     [depends on all Phase 2]
├── T-E07-F12-007 (refactor feature.go)  [depends on all Phase 2]
├── T-E07-F12-008 (update tests)         [depends on 006, 007]
└── T-E07-F12-009 (regression testing)   [depends on 008]
```

---

## Quality Gates

### Phase 2 Complete When:
- [ ] All 4 shared modules created
- [ ] All shared modules have 90%+ test coverage
- [ ] All tests pass
- [ ] No changes to epic.go or feature.go
- [ ] Code review approved

### Phase 3 Complete When:
- [ ] epic.go and feature.go refactored
- [ ] 20%+ code reduction achieved
- [ ] All duplicate code eliminated
- [ ] All tests pass
- [ ] Backward compatibility 100%
- [ ] Performance no regression
- [ ] Code review approved

### Feature Complete When:
- [ ] All 9 tasks completed
- [ ] All acceptance criteria met
- [ ] UAT passed
- [ ] Documentation updated (if needed)

---

## Related Documentation

- **Research Findings**: `research-findings.md`
- **Implementation Plan**: `implementation-plan.md`
- **Phase 1 Complete**: `phase1-implementation-complete.md`
- **Source Code**:
  - `internal/cli/commands/epic.go`
  - `internal/cli/commands/feature.go`
- **Tests**:
  - `internal/cli/commands/epic_create_test.go`
  - `internal/cli/commands/feature_create_test.go`

---

## Next Steps

1. **Review** this task breakdown with the team
2. **Dispatch** Phase 2 tasks to backend developers
3. **Monitor** progress via shark task status
4. **Quality gate** after Phase 2 before starting Phase 3
5. **Quality gate** after Phase 3 before feature completion
6. **Present UAT** to user after all tasks complete

---

**Status**: Ready for Phase 2 implementation
**Phase 1**: ✅ COMPLETE
**Phase 2**: TODO (9 tasks created)
**Phase 3**: TODO (dependent on Phase 2)
