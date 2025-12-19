# T-E07-F10-003: Write integration tests for complete commands

**Task Type**: Testing
**Epic**: E07 - Enhancements
**Feature**: E07-F10 - Add complete method to epic and feature
**Agent Type**: backend / go-developer
**Complexity**: Medium
**Status**: ready_for_development

## Overview

Write comprehensive integration tests for both `shark feature complete` and `shark epic complete` commands. Tests should verify command functionality, error handling, transactional safety, and output formats.

## Acceptance Criteria

### Feature Complete Command Tests
- [ ] Test completing feature with all tasks already completed (no warning)
- [ ] Test completing feature with incomplete tasks (shows warning, requires --force)
- [ ] Test completing feature with --force (completes all regardless of status)
- [ ] Test blocking/unblocking behavior with blocked tasks
- [ ] Test feature progress calculation after completion (should be 100%)
- [ ] Test task history records created correctly
- [ ] Test JSON output format
- [ ] Test error handling (nonexistent feature, database errors)
- [ ] Test transactional rollback on error
- [ ] Test agent identifier handling

### Epic Complete Command Tests
- [ ] Test completing epic with all tasks already completed (no warning)
- [ ] Test completing epic with incomplete tasks (shows detailed warning, requires --force)
- [ ] Test completing epic with multiple features containing mixed statuses
- [ ] Test completing epic with --force (completes all across all features)
- [ ] Test highlighting of blocked tasks in warning output
- [ ] Test epic and all feature progress calculations (all should be 100%)
- [ ] Test task history records created correctly for all tasks
- [ ] Test JSON output format
- [ ] Test error handling (nonexistent epic, database errors)
- [ ] Test transactional rollback on error
- [ ] Test agent identifier handling

### Integration with Existing Commands
- [ ] Feature complete respects existing task state validation logic
- [ ] Epic complete respects existing task state validation logic
- [ ] Task history properly recorded (consistent with task start/complete/approve)
- [ ] Feature/epic progress calculation consistent with existing implementations

## Technical Requirements

### Test Structure
1. Add tests to existing test files:
   - `/internal/cli/commands/feature_test.go` for feature complete tests
   - `/internal/cli/commands/epic_test.go` for epic complete tests

2. Test patterns to follow:
   - Use test database setup from `internal/test/testdb.go`
   - Create test fixtures (epics, features, tasks) with various statuses
   - Test both human-readable and JSON output
   - Mock output where appropriate
   - Verify database state changes

3. Test helper functions:
   - Create helper to create test task with specific status
   - Create helper to verify task history records
   - Create helper to verify progress calculations

### Test Cases

#### Feature Complete - Successful Cases
```go
TestFeatureComplete_AllTasksCompleted
  Setup: Feature with 5 completed tasks
  Action: shark feature complete
  Verify: No warning, success message, progress = 100%

TestFeatureComplete_WithIncompleteTasksAndForce
  Setup: Feature with 2 todo, 1 in_progress, 2 ready_for_review tasks
  Action: shark feature complete --force
  Verify: All tasks completed, progress = 100%, history records created

TestFeatureComplete_IncompleteTasksNoForce
  Setup: Feature with incomplete tasks
  Action: shark feature complete (no --force)
  Verify: Warning shown, exit code 3, tasks NOT completed

TestFeatureComplete_JSONOutput
  Setup: Feature with mixed task statuses
  Action: shark feature complete --force --json
  Verify: Valid JSON with expected fields
```

#### Epic Complete - Successful Cases
```go
TestEpicComplete_AllTasksCompleted
  Setup: Epic with 3 features, all tasks completed
  Action: shark epic complete
  Verify: No warning, success message, all progress = 100%

TestEpicComplete_WithIncompleteTasksAndForce
  Setup: Epic with 3 features containing mixed task statuses
  Action: shark epic complete --force
  Verify: All tasks completed, all progress = 100%, history records created

TestEpicComplete_IncompleteTasksNoForce
  Setup: Epic with incomplete tasks
  Action: shark epic complete (no --force)
  Verify: Detailed warning shown, exit code 3, tasks NOT completed

TestEpicComplete_BlockedTasksHighlighted
  Setup: Epic with some blocked tasks
  Action: shark epic complete (no --force)
  Verify: Blocked tasks highlighted in warning output

TestEpicComplete_JSONOutput
  Setup: Epic with mixed task statuses
  Action: shark epic complete --force --json
  Verify: Valid JSON with expected fields including feature_count

TestEpicComplete_TransactionRollback
  Setup: Epic with mock database error on 3rd task
  Action: shark epic complete --force
  Verify: All tasks remain in original state (rollback occurred)
```

### Code References
- Existing command tests: `/internal/cli/commands/*_test.go`
- Test database helper: `/internal/test/testdb.go`
- Repository test patterns: `/internal/repository/*_test.go`
- Models: `/internal/models/` (Task, Feature, Epic, TaskStatus)

## Testing Requirements

### Manual Integration Testing
1. Build the binary: `make shark`
2. Test feature complete:
   - Create test feature with mixed task statuses
   - Run without --force (should show warning)
   - Run with --force (should complete)
   - Verify progress and history
3. Test epic complete:
   - Create test epic with multiple features
   - Run without --force (should show detailed warning)
   - Run with --force (should complete)
   - Verify all progress calculations

### Automated Test Coverage
- [ ] All test cases pass
- [ ] Code coverage >= 80% for complete command logic
- [ ] No race conditions or concurrency issues
- [ ] Database cleanup between tests

## Definition of Done

- [ ] All test cases implemented and passing
- [ ] Both feature and epic tests comprehensive
- [ ] Code follows existing test patterns
- [ ] Test output clear and actionable
- [ ] Integration with existing commands verified
- [ ] No breaking changes to existing functionality
- [ ] Ready for code review

## Related Documents
- Feature PRD: `/docs/plan/E07-enhancements/E07-F10-complete-commands/prd.md`
- Feature complete task: `T-E07-F10-001-IMPLEMENTATION.md`
- Epic complete task: `T-E07-F10-002-IMPLEMENTATION.md`
