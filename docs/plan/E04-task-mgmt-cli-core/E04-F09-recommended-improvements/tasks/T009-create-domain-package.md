# Task: Create internal/domain Package Structure

**Task ID**: E04-F09-T009
**Feature**: E04-F09 - Recommended Architecture Improvements
**Priority**: P1 (High)
**Estimated Effort**: 1 hour
**Status**: Todo

---

## Objective

Create the `internal/domain` package to house repository interfaces and domain errors, establishing a dependency-free domain layer following clean architecture principles.

## Background

Currently, repositories are concrete implementations with no interfaces. Creating a domain package provides:
- Clean separation between interface (contract) and implementation
- Zero dependencies (pure Go code)
- Foundation for multiple implementations (SQLite, PostgreSQL, mocks)
- Testability through interfaces

## Acceptance Criteria

- [ ] `internal/domain/` package created
- [ ] Package has zero external dependencies (only stdlib + internal/models)
- [ ] Package documentation explains purpose and usage
- [ ] Ready to receive interface definitions (T010) and error definitions (T015)
- [ ] Compiles successfully

## Implementation Details

### Directory Structure

Create the following structure:

```
internal/
├── domain/
│   ├── doc.go              # Package documentation
│   ├── repositories.go     # Repository interface definitions (empty for now)
│   └── errors.go           # Domain error definitions (empty for now)
├── models/                 # Existing - unchanged
├── repository/             # Existing - will be refactored later
├── db/                     # Existing - unchanged
└── cli/                    # Existing - unchanged
```

### File: internal/domain/doc.go

```go
// Package domain defines the core business domain interfaces and types.
//
// This package establishes the domain layer following clean architecture principles.
// It contains:
//   - Repository interfaces (contracts for data access)
//   - Domain errors (business-specific error types)
//   - Domain services (if needed in the future)
//
// Design Principles:
//   1. Zero Dependencies: This package depends ONLY on stdlib and internal/models
//   2. Interface-First: Define contracts, not implementations
//   3. Technology Agnostic: No database, HTTP, or CLI dependencies
//   4. Testable: All types can be mocked easily
//
// Package Relationships:
//
//   cmd/server, cmd/pm  →  internal/domain (interfaces)
//                      ↓
//   internal/repository/sqlite  →  internal/domain (implements)
//   internal/repository/mock    →  internal/domain (implements)
//
// The domain package sits at the center of the architecture with no outward dependencies.
//
// Example Usage:
//
//   // Define interface in domain
//   type TaskRepository interface {
//       Create(ctx context.Context, task *models.Task) error
//       GetByID(ctx context.Context, id int64) (*models.Task, error)
//   }
//
//   // Implement in repository/sqlite
//   type sqliteTaskRepository struct { ... }
//   func (r *sqliteTaskRepository) Create(...) { ... }
//
//   // Use interface in application
//   func ProcessTasks(repo domain.TaskRepository) {
//       task, err := repo.GetByID(ctx, 1)
//       // ...
//   }
//
// This design allows:
//   - Easy testing with mock implementations
//   - Swapping database backends (SQLite → PostgreSQL)
//   - Clear API contracts
//   - Compile-time type safety
package domain
```

### File: internal/domain/repositories.go (Placeholder)

```go
package domain

import (
    "context"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// Repository interface definitions will be added in task E04-F09-T010.
//
// This file will contain:
//   - TaskRepository interface
//   - EpicRepository interface
//   - FeatureRepository interface
//   - TaskHistoryRepository interface
//   - Database interface (for dependency injection)
//
// Example structure (to be implemented in T010):
//
//   type TaskRepository interface {
//       Create(ctx context.Context, task *models.Task) error
//       GetByID(ctx context.Context, id int64) (*models.Task, error)
//       // ... more methods
//   }

// Placeholder comment - interfaces will be added in T010
```

### File: internal/domain/errors.go (Placeholder)

```go
package domain

// Domain error definitions will be added in task E04-F09-T015.
//
// This file will contain:
//   - ErrTaskNotFound
//   - ErrEpicNotFound
//   - ErrFeatureNotFound
//   - ErrInvalidStatus
//   - ErrDependencyNotMet
//   - And other business-specific errors
//
// Example structure (to be implemented in T015):
//
//   var (
//       ErrTaskNotFound = errors.New("task not found")
//       ErrEpicNotFound = errors.New("epic not found")
//       // ... more errors
//   )

// Placeholder comment - errors will be added in T015
```

### Package Dependencies

**What domain CAN depend on**:
- ✅ `context` (stdlib)
- ✅ `errors` (stdlib)
- ✅ `time` (stdlib)
- ✅ `internal/models` (domain entities)

**What domain CANNOT depend on**:
- ❌ `database/sql` (implementation detail)
- ❌ `internal/db` (infrastructure)
- ❌ `internal/repository` (implementations)
- ❌ `github.com/mattn/go-sqlite3` (database driver)
- ❌ `github.com/spf13/cobra` (CLI framework)
- ❌ Any external packages (except models)

**Verification**:
```bash
# Check dependencies - should only see stdlib + models
go list -f '{{ join .Imports "\n" }}' ./internal/domain/
```

### Implementation Steps

1. **Create directory**:
   ```bash
   mkdir -p internal/domain
   ```

2. **Create doc.go** with package documentation

3. **Create repositories.go** with placeholder comment

4. **Create errors.go** with placeholder comment

5. **Verify compilation**:
   ```bash
   go build ./internal/domain/
   ```

6. **Verify zero external dependencies**:
   ```bash
   go list -f '{{ join .Deps "\n" }}' ./internal/domain/ | grep -v "^internal" | grep -v "^github.com/jwwelbor" | head -10
   # Should only see stdlib packages
   ```

7. **Run tests** (none yet, but verify testability):
   ```bash
   go test ./internal/domain/
   # Should pass with no tests
   ```

## Design Decisions

### Why a Separate Domain Package?

**Benefits**:
- **Dependency Inversion**: High-level code depends on abstractions, not implementations
- **Testability**: Can mock entire repository layer
- **Flexibility**: Can swap implementations (SQLite → PostgreSQL) without changing business logic
- **Clear Contracts**: Interfaces document expected behavior
- **Clean Architecture**: Domain layer has no infrastructure dependencies

**Trade-offs**:
- More files and packages
- Slightly more boilerplate
- Need to update two places when adding methods (interface + implementation)

**Decision**: Benefits outweigh costs for a project that may grow.

### Why Zero Dependencies?

The domain package represents pure business logic and should:
- Be testable without database
- Be understandable without knowing implementation details
- Be portable across different infrastructures
- Compile extremely fast (no external dependencies)

### Why Not Use Existing Repository Types?

Current repository structs (`TaskRepository`, etc.) are concrete implementations tied to SQLite. Interfaces allow:
- Multiple implementations (SQLite, PostgreSQL, in-memory)
- Easy mocking for tests
- Clear API boundaries

## Testing

### Verification Tests

```bash
# 1. Package compiles
go build ./internal/domain/

# 2. No external dependencies
go list -f '{{ join .Imports "\n" }}' ./internal/domain/ | grep -v "^context$" | grep -v "^github.com/jwwelbor/shark-task-manager/internal/models$"
# Should output nothing (no other imports)

# 3. Package documentation exists
go doc internal/domain
# Should show package comment

# 4. Files exist
ls -la internal/domain/
# Should show: doc.go, repositories.go, errors.go
```

### Future Test Pattern

Once interfaces are defined (T010), tests will look like:

```go
// internal/domain/repositories_test.go
package domain_test  // Note: _test suffix for black-box testing

import (
    "testing"

    "github.com/jwwelbor/shark-task-manager/internal/domain"
)

// Verify interfaces can be implemented
type mockTaskRepo struct{}

func (m *mockTaskRepo) Create(ctx context.Context, task *models.Task) error {
    return nil
}
// ... implement other methods

// Verify interface can be used
func TestTaskRepositoryInterface(t *testing.T) {
    var repo domain.TaskRepository = &mockTaskRepo{}

    // Verify it compiles and satisfies interface
    _ = repo
}
```

## Dependencies

### Depends On
- None (foundational task)

### Blocks
- E04-F09-T010: Define repository interfaces in domain package
- E04-F09-T015: Define domain errors in domain/errors.go
- E04-F09-T011: Move SQLite implementations to repository/sqlite

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Package import cycle | Low | Domain has no internal imports except models |
| Confusion about package purpose | Medium | Comprehensive documentation in doc.go |
| Premature abstraction | Low | Based on architecture review recommendation |

## Success Criteria

- ✅ `internal/domain/` directory created
- ✅ `doc.go` with comprehensive package documentation
- ✅ `repositories.go` placeholder created
- ✅ `errors.go` placeholder created
- ✅ Package compiles: `go build ./internal/domain/`
- ✅ Zero external dependencies verified
- ✅ Package documentation accessible: `go doc internal/domain`
- ✅ Ready for T010 (interface definitions)
- ✅ Ready for T015 (error definitions)

## Completion Checklist

- [ ] Create `internal/domain/` directory
- [ ] Write `doc.go` with package documentation
- [ ] Create `repositories.go` with placeholder
- [ ] Create `errors.go` with placeholder
- [ ] Verify compilation: `go build ./internal/domain/`
- [ ] Verify zero external dependencies
- [ ] Test package documentation: `go doc internal/domain`
- [ ] Git commit: "Create domain package for interfaces and errors"

## Notes

- This task sets up the structure only
- No actual interfaces or errors yet (those come in T010 and T015)
- Keep package documentation clear and comprehensive
- Domain package is the foundation for clean architecture
- Review package structure with team before proceeding to T010
