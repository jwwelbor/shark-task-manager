---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T015-define-domain-errors.md]
estimated_time: 3 hours
---

# Task: Update Repositories to Return Domain Errors

## Goal

Update all repository implementations to return typed domain errors instead of generic error strings, enabling better error handling and more helpful user messages in CLI commands.

## Success Criteria

- [ ] All "not found" cases return appropriate domain errors
- [ ] All validation errors return typed domain errors
- [ ] Foreign key violations return `ErrForeignKey`
- [ ] Duplicate key violations return `ErrDuplicateKey`
- [ ] Generic database errors are wrapped, not replaced
- [ ] All tests updated to check for domain errors
- [ ] All tests pass

## Implementation Guidance

### Overview

Replace generic error strings with typed domain errors throughout all repository implementations. This enables CLI commands to distinguish error types and provide context-specific help messages to users.

### Key Requirements

- Return `domain.ErrTaskNotFound`, `domain.ErrEpicNotFound`, etc. for not found cases
- Return `domain.ErrDuplicateKey` for constraint violations
- Return `domain.ErrForeignKey` for foreign key violations
- Return validation errors: `domain.ErrInvalidTaskKey`, `domain.ErrEmptyTitle`, etc.
- Wrap database errors for debugging: `fmt.Errorf("database error: %w", err)`
- Update error handling in all repositories

Reference: [PRD - Domain-Specific Errors](../01-feature-prd.md#fr-3-domain-specific-errors)

### Files to Create/Modify

**SQLite Repository Implementations**:
- `internal/repository/sqlite/task.go` - Return domain errors
- `internal/repository/sqlite/epic.go` - Return domain errors
- `internal/repository/sqlite/feature.go` - Return domain errors
- `internal/repository/sqlite/task_history.go` - Return domain errors

**Mock Implementations** (update for consistency):
- `internal/repository/mock/task.go` - Return domain errors
- `internal/repository/mock/epic.go` - Return domain errors
- `internal/repository/mock/feature.go` - Return domain errors
- `internal/repository/mock/task_history.go` - Return domain errors

**Test Files**:
- All repository test files - Update assertions to check for domain errors

### Error Mapping Pattern

**Before**:
```go
func (r *taskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("task not found with id %d", id)  // generic error
    }
    return task, err
}
```

**After**:
```go
func (r *taskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)
    if err == sql.ErrNoRows {
        return nil, domain.ErrTaskNotFound  // domain error
    }
    if err != nil {
        return nil, fmt.Errorf("database error: %w", err)  // wrap other errors
    }
    return task, nil
}
```

**Constraint violations**:
```go
err := r.db.ExecContext(ctx, query, ...)
if err != nil {
    if strings.Contains(err.Error(), "UNIQUE constraint failed") {
        return domain.ErrDuplicateKey
    }
    if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
        return domain.ErrForeignKey
    }
    return fmt.Errorf("database error: %w", err)
}
```

Reference: [PRD - Error Usage Example](../01-feature-prd.md#fr-3-domain-specific-errors)

### Integration Points

- **Domain Errors**: Use errors defined in `internal/domain/errors.go`
- **CLI Commands**: Will check error types with `errors.Is()` (implemented in T017)
- **Tests**: Update to use `errors.Is()` for assertions
- **Error Wrapping**: Use `%w` to maintain error chain

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Unit Tests**:
- Repository tests check for specific domain errors using `errors.Is()`
- Test pattern: `assert.True(t, errors.Is(err, domain.ErrTaskNotFound))`
- All existing tests updated and passing

**Integration Tests**:
- Test "not found" scenarios return correct errors
- Test constraint violations return correct errors
- Test that error wrapping preserves error chain

**Error Verification**:
- Grep for generic error strings in repository code
- Should find none (all replaced with domain errors)
- Verify error wrapping uses `%w` format

## Context & Resources

- **PRD**: [Domain-Specific Errors Requirements](../01-feature-prd.md#fr-3-domain-specific-errors)
- **PRD**: [Error Types List](../01-feature-prd.md#fr-3-domain-specific-errors)
- **PRD**: [Error Usage Example](../01-feature-prd.md#fr-3-domain-specific-errors)
- **Task Dependency**: [T015 - Define Domain Errors](./T015-define-domain-errors.md)
- **Domain Errors**: `internal/domain/errors.go`
- **Go Errors**: [Go Blog - Working with Errors](https://go.dev/blog/go1.13-errors)

## Notes for Agent

- Pattern: Map `sql.ErrNoRows` to appropriate `domain.ErrXxxNotFound`
- SQLite constraint errors are in error message strings, need to check with `strings.Contains()`
- Always wrap unexpected database errors: `fmt.Errorf("database error: %w", err)`
- Update both SQLite implementations and mock implementations
- Update tests to use `errors.Is()` instead of string matching
- This enables T017 (CLI error handling improvements)
- Common replacements:
  - "not found" → domain.ErrTaskNotFound
  - "UNIQUE constraint" → domain.ErrDuplicateKey
  - "FOREIGN KEY constraint" → domain.ErrForeignKey
- Keep validation errors (empty title, invalid key) as domain errors too
