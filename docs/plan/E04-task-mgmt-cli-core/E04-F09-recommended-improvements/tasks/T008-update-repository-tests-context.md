---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T002-update-task-repository-context.md, T003-update-epic-repository-context.md, T004-update-feature-repository-context.md, T005-update-taskhistory-repository-context.md, T006-update-http-handlers-request-context.md, T007-update-cli-commands-timeout-context.md]
estimated_time: 3 hours
---

# Task: Update All Repository Tests to Use Context

## Goal

Comprehensively update all repository test files to use context in test cases, including testing context cancellation behavior and ensuring test coverage is maintained or improved.

## Success Criteria

- [ ] All repository test files updated to pass context
- [ ] Context cancellation tests added for key operations
- [ ] All existing tests pass
- [ ] Test coverage maintained or improved (target: >70%)
- [ ] No test-only changes to production code
- [ ] Test output shows all tests passing

## Implementation Guidance

### Overview

Complete the context support implementation by updating all repository tests to use context. This includes both updating existing tests to pass context and adding new tests for context cancellation behavior.

### Key Requirements

- Update all repository test files to pass `context.Background()` to repository methods
- Add context cancellation tests for long-running operations
- Add context timeout tests for query operations
- Verify context errors are handled correctly in repositories
- Maintain or improve test coverage

Reference: [PRD - Test Strategy](../01-feature-prd.md#testing-strategy)

### Files to Create/Modify

**Test Files**:
- `internal/repository/task_repository_test.go` - Update all test cases
- `internal/repository/epic_repository_test.go` - Update all test cases
- `internal/repository/feature_repository_test.go` - Update all test cases
- `internal/repository/task_history_repository_test.go` - Update all test cases
- `cmd/server/main_test.go` - Update HTTP handler tests (if exists)
- `internal/cli/commands/*_test.go` - Update CLI command tests (if exist)

### Test Patterns to Implement

**Basic Context Tests**:
- Update all existing tests to pass `context.Background()`
- Verify tests still pass after context parameter added

**Context Cancellation Tests**:
- Test that cancelled context aborts operations
- Test that appropriate error is returned (`context.Canceled`)

**Context Timeout Tests**:
- Test that timeout context returns deadline exceeded error
- Test that operations complete within reasonable time

Reference: [PRD - Context Cancellation Tests](../01-feature-prd.md#context-cancellation-tests)

## Validation Gates

**Linting & Type Checking**:
- All tests compile without errors
- No unused context variables

**Unit Tests**:
- Run `go test ./internal/repository/...` - all pass
- Run `go test ./internal/cli/...` - all pass
- Run `go test ./cmd/...` - all pass

**Test Coverage**:
- Run `go test -cover ./internal/repository/...`
- Verify coverage is >70% for repository packages
- Generate coverage report: `go test -coverprofile=coverage.out ./...`

**Manual Verification**:
- Check that new context tests actually test cancellation behavior
- Verify test output is clean (no race conditions, no leaked goroutines)

## Context & Resources

- **PRD**: [Testing Strategy](../01-feature-prd.md#testing-strategy)
- **PRD**: [Context Cancellation Tests](../01-feature-prd.md#context-cancellation-tests)
- **Task Dependencies**: T002-T007 (all implementation must be complete before testing)
- **Go Testing**: [Go Blog - Testing](https://go.dev/blog/subtests)
- **Go Best Practices**: [Testing Patterns](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- This is a comprehensive testing task across many files
- Pattern for basic updates:
  ```go
  // Before
  task, err := repo.GetByID(1)

  // After
  ctx := context.Background()
  task, err := repo.GetByID(ctx, 1)
  ```
- Pattern for cancellation tests:
  ```go
  func TestTaskRepository_GetByID_ContextCancelled(t *testing.T) {
      ctx, cancel := context.WithCancel(context.Background())
      cancel() // Cancel immediately

      task, err := repo.GetByID(ctx, 1)

      assert.Nil(t, task)
      assert.Equal(t, context.Canceled, err)
  }
  ```
- Focus on high-value cancellation tests (don't need one for every method)
- Test should complete Phase 1 (Context Support) implementation
- After this task, all Phase 1 validation gates should pass
