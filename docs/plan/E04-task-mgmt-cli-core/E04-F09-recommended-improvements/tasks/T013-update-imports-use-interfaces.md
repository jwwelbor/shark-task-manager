---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T011-move-sqlite-implementations.md]
estimated_time: 2 hours
---

# Task: Update All Imports to Use Interfaces

## Goal

Update all code throughout the codebase to use domain interfaces instead of concrete repository types, completing the dependency inversion pattern and enabling flexible implementation swapping.

## Success Criteria

- [ ] All `main.go` files use domain interfaces for repository variables
- [ ] All CLI commands accept domain interfaces, not concrete types
- [ ] All HTTP handlers accept domain interfaces, not concrete types
- [ ] Constructor calls use `sqlite.NewTaskRepository()` returning interface
- [ ] No direct references to concrete repository types outside of main functions
- [ ] All code compiles without errors
- [ ] All tests pass

## Implementation Guidance

### Overview

Update all code to depend on domain interfaces rather than concrete implementations. This completes the dependency inversion and enables easy testing with mocks and potential future implementation swapping (e.g., SQLite to PostgreSQL).

### Key Requirements

- Change variable types from `*repository.TaskRepository` to `domain.TaskRepository`
- Update function parameters to accept interfaces: `func foo(repo domain.TaskRepository)`
- Keep constructor calls in `main()` functions: `taskRepo := sqlite.NewTaskRepository(db)`
- Remove all imports of concrete repository packages outside of main files
- Inject dependencies through constructors or function parameters

Reference: [PRD - Package Dependencies](../01-feature-prd.md#design-overview)

### Files to Create/Modify

**Main Functions**:
- `cmd/server/main.go` - Change repository variable types to interfaces
- `cmd/pm/main.go` - Change repository variable types to interfaces

**CLI Commands**:
- `internal/cli/commands/task/*.go` - Accept domain interfaces in functions
- `internal/cli/commands/epic/*.go` - Accept domain interfaces in functions
- `internal/cli/commands/feature/*.go` - Accept domain interfaces in functions

**HTTP Handlers** (if separate files):
- Update handler functions to accept interfaces

**Helper Functions**:
- Any utility functions that accept repositories - change to interfaces

### Dependency Injection Pattern

**Before**:
```go
// cmd/server/main.go
import "internal/repository"

func main() {
    db := initDB()
    taskRepo := repository.NewTaskRepository(db)  // concrete type
    // ...
}
```

**After**:
```go
// cmd/server/main.go
import (
    "internal/domain"
    "internal/repository/sqlite"
)

func main() {
    db := initDB()
    var taskRepo domain.TaskRepository = sqlite.NewTaskRepository(db)  // interface type
    // OR simply:
    taskRepo := sqlite.NewTaskRepository(db)  // returns interface
    // ...
}
```

**CLI Command Pattern**:
```go
// Before
func runTaskList(repo *repository.TaskRepository) error { ... }

// After
func runTaskList(repo domain.TaskRepository) error { ... }
```

### Integration Points

- **Main Functions**: Only place where concrete implementations are constructed
- **Business Logic**: All other code uses interfaces
- **Tests**: Can now use mock implementations easily
- **Future**: Can swap implementations without changing business logic

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Dependency Verification**:
- Grep for concrete repository imports outside of main files
- Should only find imports in `cmd/*/main.go` files
- All business logic uses `internal/domain` imports

**Build Verification**:
- `go build ./cmd/server` succeeds
- `go build ./cmd/shark` succeeds

**Test Verification**:
- Run full test suite: `go test ./...`
- All tests pass
- Integration tests still work with SQLite implementations

**Runtime Verification**:
- Run CLI: `shark task list` works
- Run server: `go run cmd/server/main.go` works
- No runtime errors or panics

## Context & Resources

- **PRD**: [Package Dependencies](../01-feature-prd.md#design-overview)
- **PRD**: [Repository Interfaces](../01-feature-prd.md#fr-2-repository-interfaces)
- **Task Dependency**: [T011 - Move SQLite Implementations](./T011-move-sqlite-implementations.md)
- **Domain Interfaces**: `internal/domain/repositories.go`
- **Architecture**: [Dependency Inversion](../../../../architecture/ARCHITECTURE_REVIEW.md)
- **Go Best Practices**: [Interface-Based Design](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- This is primarily a refactoring task - change types, update imports
- Pattern: Accept interfaces, return concrete implementations from constructors
- Only `main()` functions should import concrete implementations (`internal/repository/sqlite`)
- All other code imports interfaces (`internal/domain`)
- Use IDE refactoring or systematic find/replace for type changes
- After this task, dependency graph is clean: business logic → interfaces, not implementations
- This enables easy testing (use mocks) and future implementation swapping
- Verify with `go mod graph` or manual inspection that imports are correct
