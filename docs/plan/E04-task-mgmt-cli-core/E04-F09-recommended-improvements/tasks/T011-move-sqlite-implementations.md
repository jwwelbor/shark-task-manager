---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T010-define-repository-interfaces.md]
estimated_time: 3 hours
---

# Task: Move SQLite Implementations to repository/sqlite Package

## Goal

Refactor the existing repository implementations to the `internal/repository/sqlite/` package, making them explicit implementations of the domain interfaces and improving code organization.

## Success Criteria

- [ ] New `internal/repository/sqlite/` package created
- [ ] TaskRepository implementation moved to `sqlite/task.go`
- [ ] EpicRepository implementation moved to `sqlite/epic.go`
- [ ] FeatureRepository implementation moved to `sqlite/feature.go`
- [ ] TaskHistoryRepository implementation moved to `sqlite/task_history.go`
- [ ] All implementations explicitly implement domain interfaces
- [ ] All tests still pass
- [ ] Old repository files removed (if fully migrated)

## Implementation Guidance

### Overview

Move the current SQLite-based repository implementations from `internal/repository/` to `internal/repository/sqlite/` to make them explicit implementations of the domain interfaces. This improves code organization and makes it clear these are SQLite-specific implementations.

### Key Requirements

- Create `internal/repository/sqlite/` package
- Move each repository implementation to its own file in sqlite package
- Update struct types to be private (lowercase): `taskRepository`, not `TaskRepository`
- Update constructor functions to return domain interface types
- Verify implementations satisfy domain interfaces (use compile-time check)
- Update all imports throughout codebase

Reference: [PRD - Package Structure](../01-feature-prd.md#fr-2-repository-interfaces)

### Files to Create/Modify

**New SQLite Package**:
- `internal/repository/sqlite/task.go` - TaskRepository SQLite implementation
- `internal/repository/sqlite/epic.go` - EpicRepository SQLite implementation
- `internal/repository/sqlite/feature.go` - FeatureRepository SQLite implementation
- `internal/repository/sqlite/task_history.go` - TaskHistoryRepository SQLite implementation
- `internal/repository/sqlite/database.go` - Database interface/wrapper (if needed)

**Files to Update**:
- `cmd/server/main.go` - Update imports to `internal/repository/sqlite`
- `cmd/pm/main.go` - Update imports to `internal/repository/sqlite`
- All CLI command files - Update imports
- Test files - Update imports

**Files to Remove** (after migration):
- `internal/repository/task_repository.go`
- `internal/repository/epic_repository.go`
- `internal/repository/feature_repository.go`
- `internal/repository/task_history_repository.go`

### Implementation Pattern

**Private struct, public constructor returning interface**:
```go
// sqlite/task.go
package sqlite

import "internal/domain"

type taskRepository struct {
    db Database
}

// NewTaskRepository creates a new SQLite-backed TaskRepository
func NewTaskRepository(db Database) domain.TaskRepository {
    return &taskRepository{db: db}
}

// Compile-time interface check
var _ domain.TaskRepository = (*taskRepository)(nil)
```

Reference: [PRD - Interface Example](../01-feature-prd.md#fr-2-repository-interfaces)

### Integration Points

- **Domain Interfaces**: Implementations must satisfy interface contracts
- **Main Functions**: Update imports and constructor calls in `cmd/`
- **Tests**: Update imports to use new sqlite package
- **Database Type**: May need to define Database interface in sqlite package

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors
- Interface satisfaction verified: `var _ domain.TaskRepository = (*taskRepository)(nil)`

**Unit Tests**:
- Run `go test ./internal/repository/sqlite/...` - all pass
- Test coverage maintained

**Integration Tests**:
- Run full test suite: `go test ./...` - all pass
- CLI commands still work
- Server still works

**Manual Verification**:
- Build project: `go build ./cmd/...`
- Run CLI: `pm task list`
- Run server: `go run cmd/server/main.go`
- Verify no regressions

## Context & Resources

- **PRD**: [Repository Interfaces](../01-feature-prd.md#fr-2-repository-interfaces)
- **PRD**: [Package Structure](../01-feature-prd.md#fr-2-repository-interfaces)
- **Task Dependency**: [T010 - Define Interfaces](./T010-define-repository-interfaces.md)
- **Current Code**: `internal/repository/*.go`
- **Architecture**: [SYSTEM_DESIGN.md](../../../../architecture/SYSTEM_DESIGN.md)
- **Go Best Practices**: [Package Organization](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- This is primarily a refactoring/moving task with import updates
- Keep implementation code identical, just move files and update package
- Pattern: Private struct (`taskRepository`), public constructor returning interface
- Use compile-time check to verify interface implementation: `var _ domain.TaskRepository = (*taskRepository)(nil)`
- Update imports systematically (use IDE refactoring or find/replace)
- May need to define `Database` interface in sqlite package if using interface for *sql.DB
- Remove old files only after verifying all imports updated
- Test thoroughly after move - this touches many files
