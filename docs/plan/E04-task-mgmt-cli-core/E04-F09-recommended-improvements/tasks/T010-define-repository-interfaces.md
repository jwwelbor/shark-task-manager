---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: backend-architect
dependencies: [T009-create-domain-package.md]
estimated_time: 2 hours
---

# Task: Define Repository Interfaces in Domain Package

## Goal

Define explicit interfaces for all repositories in the `internal/domain/` package to enable dependency inversion, easier testing, and the ability to swap implementations (e.g., SQLite to PostgreSQL).

## Success Criteria

- [ ] `TaskRepository` interface defined with all methods
- [ ] `EpicRepository` interface defined with all methods
- [ ] `FeatureRepository` interface defined with all methods
- [ ] `TaskHistoryRepository` interface defined with all methods
- [ ] All interfaces use `context.Context` as first parameter
- [ ] Interfaces documented with godoc comments
- [ ] No dependencies on concrete implementations
- [ ] Code compiles without errors

## Implementation Guidance

### Overview

Create the repository interfaces that define the contracts for data access operations. These interfaces live in `internal/domain/` and have zero dependencies on concrete implementations, following the Dependency Inversion Principle.

### Key Requirements

- Define interfaces in `internal/domain/repositories.go`
- Each interface method matches current repository implementation signatures
- All methods accept `context.Context` as first parameter
- Methods return domain types (`*models.Task`, etc.) and errors
- Include comprehensive godoc comments
- No implementation details in interfaces (pure contracts)

Reference: [PRD - Repository Interfaces](../01-feature-prd.md#fr-2-repository-interfaces)

### Files to Create/Modify

**Domain Package**:
- `internal/domain/repositories.go` - Define all four repository interfaces

### Interface Specifications

**TaskRepository Interface**:
- All CRUD operations: Create, GetByID, GetByKey, Update, Delete
- Query operations: List, ListByEpic, ListByFeature, ListByStatus
- Status operations: UpdateStatus, GetHistory
- Context-aware: All methods accept `context.Context`

**EpicRepository Interface**:
- CRUD operations for epics
- Query operations: List, GetByKey
- Context-aware: All methods accept `context.Context`

**FeatureRepository Interface**:
- CRUD operations for features
- Query operations: List, GetByKey, ListByEpic
- Context-aware: All methods accept `context.Context`

**TaskHistoryRepository Interface**:
- Create history entry
- Query operations: ListByTask, GetByID
- Context-aware: All methods accept `context.Context`

Reference: [PRD - Interface Example](../01-feature-prd.md#fr-2-repository-interfaces)

### Integration Points

- **Current Repositories**: Interface signatures must match current repository method signatures
- **Domain Types**: Interfaces use `*models.Task`, `*models.Epic`, etc.
- **Future Implementations**: Interfaces enable SQLite, PostgreSQL, mock, in-memory implementations

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors
- Godoc comments present for all interfaces and methods

**Interface Verification**:
- Compare interface signatures with current repository implementations
- Verify all public methods from current repos are in interfaces
- Verify all methods have `context.Context` as first parameter

**Documentation**:
- Each interface has godoc comment explaining its purpose
- Each method has godoc comment (brief description)

## Context & Resources

- **PRD**: [Repository Interfaces Requirements](../01-feature-prd.md#fr-2-repository-interfaces)
- **PRD**: [Package Structure](../01-feature-prd.md#fr-2-repository-interfaces)
- **Task Dependency**: [T009 - Domain Package Structure](./T009-create-domain-package.md)
- **Current Implementation**: `internal/repository/*.go` files
- **Architecture**: [ARCHITECTURE_REVIEW.md](../../../../architecture/ARCHITECTURE_REVIEW.md)
- **Go Best Practices**: [Interface Design](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- Examine current repository implementations to ensure complete interface coverage
- Pattern for interfaces:
  ```go
  // TaskRepository defines the data access contract for tasks
  type TaskRepository interface {
      Create(ctx context.Context, task *models.Task) error
      GetByID(ctx context.Context, id int64) (*models.Task, error)
      GetByKey(ctx context.Context, key string) (*models.Task, error)
      // ... all other methods
  }
  ```
- Keep interfaces in single file: `internal/domain/repositories.go`
- No implementation details, just method signatures
- Interfaces enable testing with mocks (created in T012)
- Interfaces are implemented by SQLite concrete types (moved in T011)
- This is pure interface definition - no logic, no dependencies
