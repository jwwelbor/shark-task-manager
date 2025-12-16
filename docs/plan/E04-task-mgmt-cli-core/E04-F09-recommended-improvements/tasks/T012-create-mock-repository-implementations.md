---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T010-define-repository-interfaces.md]
estimated_time: 4 hours
---

# Task: Create Mock Repository Implementations

## Goal

Create mock implementations of all repository interfaces in `internal/repository/mock/` package to enable easy testing without requiring a real database.

## Success Criteria

- [ ] Mock implementations created for all four repositories
- [ ] Mocks implement domain interfaces completely
- [ ] Mocks use in-memory storage (maps) for data
- [ ] Mocks support all CRUD operations
- [ ] Mock behavior is predictable and testable
- [ ] Example test demonstrating mock usage provided
- [ ] All mocks include compile-time interface checks

## Implementation Guidance

### Overview

Create in-memory mock implementations of all repository interfaces to enable fast unit testing without database dependencies. These mocks should be simple, predictable, and sufficient for testing business logic in CLI commands and HTTP handlers.

### Key Requirements

- Create `internal/repository/mock/` package
- Implement all four domain interfaces with in-memory storage
- Use Go maps for storage: `map[int64]*models.Task`, `map[string]*models.Task`, etc.
- Support basic CRUD operations with realistic behavior
- Thread-safe if used concurrently (use `sync.RWMutex`)
- Return domain errors (e.g., `domain.ErrTaskNotFound`) appropriately

Reference: [PRD - Mock Implementations](../01-feature-prd.md#fr-2-repository-interfaces)

### Files to Create/Modify

**Mock Package**:
- `internal/repository/mock/task.go` - Mock TaskRepository
- `internal/repository/mock/epic.go` - Mock EpicRepository
- `internal/repository/mock/feature.go` - Mock FeatureRepository
- `internal/repository/mock/task_history.go` - Mock TaskHistoryRepository
- `internal/repository/mock/README.md` - Mock usage documentation

**Example Test** (optional but recommended):
- `internal/repository/mock/example_test.go` - Example showing mock usage

### Mock Implementation Pattern

**Basic structure**:
```go
// mock/task.go
package mock

type TaskRepository struct {
    mu         sync.RWMutex
    tasks      map[int64]*models.Task
    tasksByKey map[string]*models.Task
    nextID     int64
}

func NewTaskRepository() *TaskRepository {
    return &TaskRepository{
        tasks:      make(map[int64]*models.Task),
        tasksByKey: make(map[string]*models.Task),
        nextID:     1,
    }
}

func (m *TaskRepository) Create(ctx context.Context, task *models.Task) error {
    if ctx.Err() != nil {
        return ctx.Err()
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    task.ID = m.nextID
    m.nextID++
    m.tasks[task.ID] = task
    m.tasksByKey[task.Key] = task
    return nil
}

// GetByID retrieves task by ID
func (m *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    if ctx.Err() != nil {
        return nil, ctx.Err()
    }

    m.mu.RLock()
    defer m.mu.RUnlock()

    task, exists := m.tasks[id]
    if !exists {
        return nil, domain.ErrTaskNotFound
    }
    return task, nil
}

// ... implement all other methods
```

**Interface satisfaction check**:
```go
var _ domain.TaskRepository = (*TaskRepository)(nil)
```

### Integration Points

- **Domain Interfaces**: Mocks implement domain.TaskRepository, etc.
- **Domain Errors**: Mocks return appropriate domain errors
- **Tests**: Used in unit tests for CLI commands, HTTP handlers
- **Context Support**: Mocks respect context cancellation

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors
- All interface checks pass

**Unit Tests**:
- Write tests for mock implementations themselves
- Test CRUD operations work correctly
- Test that domain errors are returned appropriately
- Test context cancellation is respected

**Integration with Existing Tests**:
- Update 1-2 existing tests to use mocks as proof of concept
- Verify mocks are easier to use than real database in tests

**Example Test**:
```go
func TestTaskCommand_WithMock(t *testing.T) {
    mockRepo := mock.NewTaskRepository()

    // Create test task
    task := &models.Task{Key: "TEST-001", Title: "Test"}
    err := mockRepo.Create(context.Background(), task)
    require.NoError(t, err)

    // Retrieve task
    retrieved, err := mockRepo.GetByKey(context.Background(), "TEST-001")
    require.NoError(t, err)
    assert.Equal(t, "Test", retrieved.Title)
}
```

## Context & Resources

- **PRD**: [Repository Interfaces](../01-feature-prd.md#fr-2-repository-interfaces)
- **PRD**: [Testing Strategy](../01-feature-prd.md#testing-strategy)
- **Task Dependency**: [T010 - Define Interfaces](./T010-define-repository-interfaces.md)
- **Domain Interfaces**: `internal/domain/repositories.go`
- **Domain Errors**: `internal/domain/errors.go` (will be created in T015)
- **Go Best Practices**: [Testing and Mocks](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- Mocks are in-memory implementations - no database, no persistence
- Use maps for storage: `map[int64]*models.Task`, `map[string]*models.Task`
- Thread-safe: Use `sync.RWMutex` for concurrent access
- Respect context: Check `ctx.Err()` at method start
- Return domain errors: `domain.ErrTaskNotFound`, not `fmt.Errorf("not found")`
- Mocks should be simple - don't need to implement every edge case
- Goal: Make testing easier, not to be production-grade implementations
- Include README.md explaining how to use mocks in tests
- This enables T014 (updating tests to use mocks)
