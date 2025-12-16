---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T001-add-context-to-repository-interfaces.md]
estimated_time: 2 hours
---

# Task: Update EpicRepository to Use Context

## Goal

Update the EpicRepository implementation to accept and use `context.Context` in all database operations, enabling cancellation support and preparing for distributed tracing.

## Success Criteria

- [ ] All EpicRepository methods accept `context.Context` as first parameter
- [ ] All database queries use `QueryContext()`, `QueryRowContext()`, and `ExecContext()`
- [ ] Context cancellation is checked before database operations
- [ ] All tests updated to pass context
- [ ] All tests pass
- [ ] No breaking changes to method signatures beyond adding context parameter

## Implementation Guidance

### Overview

Update the EpicRepository to follow the context-aware pattern defined in T001. This enables HTTP handlers to pass request context and CLI commands to use timeout context for epic operations.

### Key Requirements

- Add `ctx context.Context` as first parameter to all methods - See [PRD - Context Support](../01-feature-prd.md#fr-1-context-support)
- Replace all `db.Query()` calls with `db.QueryContext(ctx, ...)`
- Replace all `db.QueryRow()` calls with `db.QueryRowContext(ctx, ...)`
- Replace all `db.Exec()` calls with `db.ExecContext(ctx, ...)`
- Check `ctx.Err()` at the start of each method for early cancellation detection

### Files to Create/Modify

**Backend**:
- `internal/repository/epic_repository.go` - Update all methods to use context
- `internal/repository/epic_repository_test.go` - Update test calls to pass context

### Integration Points

- **TaskRepository**: Epic operations may be called alongside task operations
- **FeatureRepository**: Epics contain features, context propagates through queries
- **CLI Commands**: `pm epic` commands will pass timeout context
- **HTTP Handlers**: Server handlers will pass request context

Reference: [PRD - Affected Components](../01-feature-prd.md#affected-components)

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Unit Tests**:
- All existing epic repository tests pass with context
- Context cancellation is respected (operations abort on cancelled context)
- Test coverage maintained at current level

**Integration Tests**:
- Epic CRUD operations work with timeout context
- Database queries use `*Context()` variants

**Manual Testing**:
- Run `pm epic list` and verify it works
- Run `pm epic create` and verify it works
- Verify no performance degradation

## Context & Resources

- **PRD**: [Context Support Requirements](../01-feature-prd.md#fr-1-context-support)
- **Task Reference**: [T001 - Add Context Signatures](./T001-add-context-to-repository-interfaces.md)
- **Task Reference**: [T002 - TaskRepository Example](./T002-update-task-repository-context.md)
- **Architecture**: [ARCHITECTURE_REVIEW.md](../../../../architecture/ARCHITECTURE_REVIEW.md)
- **Go Best Practices**: [GO_BEST_PRACTICES.md](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- Follow the exact pattern used in T002 for consistency
- Context parameter is always first: `func (r *EpicRepository) Method(ctx context.Context, ...)`
- Check `ctx.Err()` at the start of methods for early cancellation
- For methods that loop over results, check context periodically: `if ctx.Err() != nil { return ctx.Err() }`
- Update all callers to pass `context.Background()` initially (will be improved in T006/T007)
- EpicRepository is simpler than TaskRepository, should take less time
