---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T012-create-mock-repository-implementations.md, T013-update-imports-use-interfaces.md]
estimated_time: 3 hours
---

# Task: Update Tests to Use Mock Implementations

## Goal

Update unit tests for CLI commands and HTTP handlers to use mock repository implementations instead of real database connections, making tests faster, more reliable, and easier to write.

## Success Criteria

- [ ] CLI command tests use mock repositories
- [ ] HTTP handler tests use mock repositories (if exist)
- [ ] Tests run without requiring database setup
- [ ] Tests are faster than before
- [ ] Test coverage maintained or improved
- [ ] All tests pass
- [ ] Example tests demonstrating mock usage provided

## Implementation Guidance

### Overview

Refactor existing unit tests to use the mock repository implementations created in T012. This makes tests faster (no database I/O), more reliable (no database state issues), and easier to write (simple in-memory mocks).

### Key Requirements

- Replace real database setup with mock repository initialization
- Update test cases to use mock repositories
- Remove database cleanup/teardown code from unit tests (keep in integration tests)
- Demonstrate mock patterns in example tests
- Keep integration tests using real SQLite database

Reference: [PRD - Testing Strategy](../01-feature-prd.md#testing-strategy)

### Files to Create/Modify

**CLI Command Tests**:
- `internal/cli/commands/task/*_test.go` - Use mock repositories
- `internal/cli/commands/epic/*_test.go` - Use mock repositories
- `internal/cli/commands/feature/*_test.go` - Use mock repositories

**Handler Tests** (if exist):
- `cmd/server/*_test.go` - Use mock repositories

**Integration Tests** (keep using real DB):
- `internal/repository/sqlite/*_test.go` - Continue using real database

**Example Tests**:
- Update or create example showing mock usage patterns

### Test Refactoring Pattern

**Before (using real database)**:
```go
func TestTaskList(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := repository.NewTaskRepository(db)

    // Test logic...
}
```

**After (using mocks)**:
```go
func TestTaskList(t *testing.T) {
    mockRepo := mock.NewTaskRepository()

    // Setup test data
    task := &models.Task{Key: "TEST-001", Title: "Test"}
    mockRepo.Create(context.Background(), task)

    // Test logic...
}
```

**Benefits**:
- No database setup/teardown
- Faster test execution
- Easier to test edge cases (just put data in mock)
- No database state pollution between tests

### Integration Points

- **Mock Repositories**: Use `internal/repository/mock` package
- **Domain Interfaces**: Tests work with interfaces, not concrete types
- **Integration Tests**: Keep using real database for end-to-end tests
- **Test Helpers**: May create helper functions for common mock setups

## Validation Gates

**Linting & Type Checking**:
- All tests compile without errors
- No unused database setup code in unit tests

**Unit Tests**:
- Run unit tests: `go test ./internal/cli/...` - all pass
- Tests run faster than before (measure execution time)
- Tests don't require database file

**Integration Tests**:
- Integration tests still use real database
- Run integration tests: `go test ./internal/repository/sqlite/...` - all pass

**Test Coverage**:
- Run coverage: `go test -cover ./...`
- Coverage maintained or improved
- Generate report: `go test -coverprofile=coverage.out ./...`

**Developer Experience**:
- Tests are easier to understand
- Tests are easier to write
- Test failures are easier to debug

## Context & Resources

- **PRD**: [Testing Strategy](../01-feature-prd.md#testing-strategy)
- **PRD**: [Unit Tests Section](../01-feature-prd.md#unit-tests)
- **Task Dependency**: [T012 - Create Mocks](./T012-create-mock-repository-implementations.md)
- **Mock Package**: `internal/repository/mock/`
- **Mock README**: `internal/repository/mock/README.md`
- **Go Testing**: [Go Blog - Table Driven Tests](https://go.dev/blog/subtests)

## Notes for Agent

- Focus on CLI command tests first (most benefit from mocks)
- Keep integration tests using real database (test SQLite implementation)
- Pattern: Create mock, populate with test data, run test, assert results
- Mocks make testing edge cases easier (just create specific mock state)
- Tests should be faster - no database I/O overhead
- Consider table-driven tests for multiple scenarios with mocks
- This completes Phase 2 (Repository Interfaces) implementation
- After this task, all Phase 2 validation gates should pass
