---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T001-add-context-to-repository-interfaces.md]
estimated_time: 1 hour
---

# Task: Update TaskHistoryRepository to Use Context

## Goal

Update the TaskHistoryRepository implementation to accept and use `context.Context` in all database operations, enabling cancellation support for history tracking operations.

## Success Criteria

- [ ] All TaskHistoryRepository methods accept `context.Context` as first parameter
- [ ] All database queries use `QueryContext()`, `QueryRowContext()`, and `ExecContext()`
- [ ] Context cancellation is checked before database operations
- [ ] All tests updated to pass context
- [ ] All tests pass
- [ ] No breaking changes to method signatures beyond adding context parameter

## Implementation Guidance

### Overview

Update the TaskHistoryRepository to follow the context-aware pattern defined in T001. This is the simplest repository update as TaskHistory is primarily append-only with read operations for audit trails.

### Key Requirements

- Add `ctx context.Context` as first parameter to all methods - See [PRD - Context Support](../01-feature-prd.md#fr-1-context-support)
- Replace all `db.Query()` calls with `db.QueryContext(ctx, ...)`
- Replace all `db.QueryRow()` calls with `db.QueryRowContext(ctx, ...)`
- Replace all `db.Exec()` calls with `db.ExecContext(ctx, ...)`
- Check `ctx.Err()` at the start of each method for early cancellation detection

### Files to Create/Modify

**Backend**:
- `internal/repository/task_history_repository.go` - Update all methods to use context
- `internal/repository/task_history_repository_test.go` - Update test calls to pass context (if exists)

### Integration Points

- **TaskRepository**: History entries are created when tasks change status
- **CLI Commands**: `pm task history` commands will pass timeout context
- **HTTP Handlers**: Server handlers will pass request context for audit queries

Reference: [PRD - Affected Components](../01-feature-prd.md#affected-components)

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Unit Tests**:
- All existing task history repository tests pass with context
- Context cancellation is respected (operations abort on cancelled context)
- Test coverage maintained at current level

**Integration Tests**:
- Task history operations work with timeout context
- History entries are created successfully with context

**Manual Testing**:
- Change task status and verify history entry created
- Run `pm task history <key>` and verify it works
- Verify no performance degradation

## Context & Resources

- **PRD**: [Context Support Requirements](../01-feature-prd.md#fr-1-context-support)
- **Task Reference**: [T001 - Add Context Signatures](./T001-add-context-to-repository-interfaces.md)
- **Task Reference**: [T002 - TaskRepository Example](./T002-update-task-repository-context.md)
- **Architecture**: [ARCHITECTURE_REVIEW.md](../../../../architecture/ARCHITECTURE_REVIEW.md)
- **Go Best Practices**: [GO_BEST_PRACTICES.md](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- Follow the exact pattern used in T002 for consistency
- Context parameter is always first: `func (r *TaskHistoryRepository) Method(ctx context.Context, ...)`
- Check `ctx.Err()` at the start of methods for early cancellation
- TaskHistoryRepository is the simplest repository - mostly insert and list operations
- Update all callers to pass `context.Background()` initially (will be improved in T006/T007)
- This task should be quick - estimated 1 hour
