# Parallel Execution Plan: Phase 2 Tasks (002-004)

**Date**: 2026-01-01
**Feature**: E07-F12 Parameter Consistency
**Phase**: Phase 2 - Shared Module Creation

## Overview

Executing three independent backend tasks in parallel:
- T-E07-F12-002: validators.go
- T-E07-F12-003: file_assignment.go  
- T-E07-F12-004: status_priority.go

These tasks have NO dependencies on each other and can run simultaneously.

## Task Specifications

### T-E07-F12-002: validators.go
- **Priority**: 8
- **Agent**: backend
- **File**: internal/cli/commands/validators.go
- **Test File**: internal/cli/commands/validators_test.go
- **Coverage Target**: 95%+
- **Dependencies**: None
- **API**:
  - ValidateCustomPath
  - ValidateCustomFilename
  - ValidateNoSpaces
  - ValidateStatus
  - ValidatePriority

### T-E07-F12-003: file_assignment.go
- **Priority**: 8
- **Agent**: backend
- **File**: internal/cli/commands/file_assignment.go
- **Test File**: internal/cli/commands/file_assignment_test.go
- **Coverage Target**: 95%+
- **Dependencies**: None (but will use validators.go in Phase 3)
- **API**:
  - DetectFileCollision
  - HandleFileReassignment
  - CreateBackupIfForce

### T-E07-F12-004: status_priority.go
- **Priority**: 8
- **Agent**: backend
- **File**: internal/cli/commands/status_priority.go
- **Test File**: internal/cli/commands/status_priority_test.go
- **Coverage Target**: 95%+
- **Dependencies**: validators.go (for validation functions)
- **API**:
  - ParseEpicStatus
  - ParseEpicPriority
  - ParseEpicBusinessValue
  - ParseFeatureStatus

## Execution Strategy

### TDD Workflow (Each Task)

1. **RED Phase**: Write failing tests
   - Create test file
   - Write comprehensive test cases
   - Verify tests fail (no implementation yet)

2. **GREEN Phase**: Implement minimal code
   - Create implementation file
   - Write code to pass tests
   - Verify all tests pass

3. **Coverage Phase**: Verify coverage
   - Run coverage report
   - Ensure 95%+ coverage
   - Add tests if needed

### Coordination Points

**Task 004 Note**: status_priority.go uses validation functions from validators.go (Task 002). However, Task 004 can proceed in parallel by:
- Starting with tests and implementation structure
- Using the validators.go API contract (defined in specs)
- The actual import will work once Task 002 completes

### Success Criteria

Each task complete when:
- [ ] Implementation file created
- [ ] Test file created with comprehensive tests
- [ ] All tests passing
- [ ] Coverage >= 95%
- [ ] No changes to epic.go or feature.go (Phase 3)
- [ ] Task status updated in shark

## Shark Commands for Monitoring

```bash
# Start tasks
./bin/shark task start T-E07-F12-002
./bin/shark task start T-E07-F12-003
./bin/shark task start T-E07-F12-004

# Check progress
./bin/shark task list E07-F12 --json

# Complete tasks
./bin/shark task complete T-E07-F12-002 --notes="..."
./bin/shark task complete T-E07-F12-003 --notes="..."
./bin/shark task complete T-E07-F12-004 --notes="..."
```

## Next Steps After Completion

After all three tasks complete:
- T-E07-F12-005: Write comprehensive integration tests for all Phase 2 modules
- Then Phase 3: Refactor epic.go and feature.go to use shared modules

---

## Execution Progress

### T-E07-F12-002: validators.go ✅ COMPLETE

**Status**: ready_for_code_review
**Completion Time**: 2026-01-01

**Implementation Summary**:
- Created `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/validators.go`
- Created `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/validators_test.go`
- Followed strict TDD methodology (RED-GREEN-REFACTOR)
- All tests passing (100%)
- Coverage: 96% average across all functions
  - ValidateCustomPath: 90.0%
  - ValidateCustomFilename: 100.0%
  - ValidateNoSpaces: 100.0%
  - ValidateStatus: 90.0%
  - ValidatePriority: 100.0%

**API Delivered**:
```go
type PathValidationResult struct {
    RelativePath string
    AbsolutePath string
}

func ValidateCustomPath(cmd *cobra.Command, flagName string) (*PathValidationResult, error)
func ValidateCustomFilename(cmd *cobra.Command, flagName string, projectRoot string) (*PathValidationResult, error)
func ValidateNoSpaces(key string, entityType string) error
func ValidateStatus(status string, entityType string) error
func ValidatePriority(priority string, entityType string) error
```

**Test Coverage**:
- 11 test functions
- 45+ test cases
- All edge cases covered (empty values, invalid values, entity type variations)

**Next Steps**:
- Move to T-E07-F12-003 (file_assignment.go)
- Move to T-E07-F12-004 (status_priority.go)

