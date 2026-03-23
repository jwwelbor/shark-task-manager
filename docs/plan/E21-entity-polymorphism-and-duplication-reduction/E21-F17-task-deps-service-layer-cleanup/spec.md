# Specification: E21-F17 Task Deps Service Layer Cleanup

**Date:** 2026-03-22
**Status:** Complete
**Complexity:** STANDARD (11/27)

---

## 1. Problem Statement

The task dependencies CLI command (`internal/cli/commands/task_deps.go`) contains approximately 150 lines of business logic directly in CLI command handlers, violating the thin-controller architecture established in E15. Three issues need resolution:

1. **Fat controller anti-pattern**: Functions `getTaskRelationshipsViaEntityRel`, `getTaskBlockedByViaEntityRel`, and `getTaskBlocksViaEntityRel` contain relationship resolution, filtering, and task enrichment logic that belongs in the service layer.

2. **N+1 query pattern**: Each of the three helper functions calls `taskSvc.GetTaskByID()` inside a loop -- one database round-trip per relationship. For a task with 10 relationships, this means 10 individual queries instead of 1 batch query.

3. **Code duplication**: The three helper functions follow nearly identical patterns (get relationships, filter by entity type, loop to resolve task details, build `RelationshipWithTask` slice) with only minor differences in direction and relationship type filtering.

---

## 2. Requirements

### REQ-1: Move relationship helpers to EntityRelationshipService

**Description:** Extract `getTaskRelationshipsViaEntityRel`, `getTaskBlockedByViaEntityRel`, and `getTaskBlocksViaEntityRel` from `task_deps.go` into service methods on `EntityRelationshipService`.

**Acceptance Criteria:**
- `EntityRelationshipService` gains three new methods:
  - `GetTaskRelationships(ctx, taskID int64, typeFilter []string) ([]RelationshipWithTask, error)`
  - `GetTaskBlockedBy(ctx, taskID int64) ([]RelationshipWithTask, error)`
  - `GetTaskBlocks(ctx, taskID int64) ([]RelationshipWithTask, error)`
- All three methods accept `context.Context` as first parameter and task ID (not task key) since the CLI already resolves the task before calling these.
- The `EntityRelationshipService` requires a new dependency: a `TaskByIDResolver` interface for resolving task IDs to task details.
- CLI commands (`runTaskDeps`, `runTaskBlockedBy`, `runTaskBlocks`) become thin wrappers: parse args, call service, format output.
- The private helper functions are removed from `task_deps.go`.
- Existing CLI behavior (output format, error handling) is preserved.

### REQ-2: Add batch GetTasksByIDs repository method

**Description:** Add a `GetByIDs(ctx, ids []int64) ([]*models.Task, error)` method to `TaskRepository` that fetches multiple tasks in a single SQL query.

**Acceptance Criteria:**
- `TaskRepository` gains `GetByIDs(ctx context.Context, ids []int64) ([]*models.Task, error)`.
- Uses a single SQL query with `WHERE id IN (?, ?, ...)` parameterization.
- Returns tasks in no guaranteed order (caller sorts if needed).
- Handles empty ID slice gracefully (returns empty slice, no error).
- Handles IDs not found gracefully (returns only found tasks, no error for missing IDs).
- The `TaskByIDResolver` interface used by `EntityRelationshipService` includes this batch method.

### REQ-3: Unify duplicate resolve patterns

**Description:** The three service methods from REQ-1 share a common pattern: get relationships from repository, filter task-to-task only, resolve related task IDs to task details, build `RelationshipWithTask` results. Extract this shared logic into a private helper.

**Acceptance Criteria:**
- A private method `resolveTaskRelationships(ctx, rels []*models.EntityRelationship, selfTaskID int64) ([]RelationshipWithTask, error)` handles the shared logic.
- `GetTaskRelationships`, `GetTaskBlockedBy`, and `GetTaskBlocks` delegate to this helper after fetching relationships from the repository.
- The helper uses batch `GetByIDs` (from REQ-2) instead of per-ID lookups.
- Total unique lines of relationship resolution logic is reduced by approximately 50% compared to the current three separate functions.

---

## 3. Architecture

### 3.1 New Interface: TaskByIDResolver

Defined in `internal/services/entity_relationship_service.go`, at the point of use:

```go
// TaskByIDResolver resolves task IDs to task models.
// Used by EntityRelationshipService for enriching relationships with task details.
type TaskByIDResolver interface {
    GetByID(ctx context.Context, id int64) (*models.Task, error)
    GetByIDs(ctx context.Context, ids []int64) ([]*models.Task, error)
}
```

This interface is satisfied by `*repository.TaskRepository`.

### 3.2 Updated EntityRelationshipService

```go
type EntityRelationshipService struct {
    repo         EntityRelationshipRepository
    taskResolver TaskByIDResolver  // NEW: for enriching task relationships
}

func NewEntityRelationshipService(
    repo EntityRelationshipRepository,
    taskResolver TaskByIDResolver,    // NEW parameter (can be nil for non-task usage)
) *EntityRelationshipService
```

**New public methods:**

```go
// GetTaskRelationships returns all task-to-task relationships for a task,
// enriched with task details, optionally filtered by relationship type.
func (s *EntityRelationshipService) GetTaskRelationships(
    ctx context.Context, taskID int64, typeFilter []string,
) ([]RelationshipWithTask, error)

// GetTaskBlockedBy returns tasks that this task depends on (outgoing depends_on).
func (s *EntityRelationshipService) GetTaskBlockedBy(
    ctx context.Context, taskID int64,
) ([]RelationshipWithTask, error)

// GetTaskBlocks returns tasks that depend on this task (incoming depends_on + outgoing blocks).
func (s *EntityRelationshipService) GetTaskBlocks(
    ctx context.Context, taskID int64,
) ([]RelationshipWithTask, error)
```

**New private helper:**

```go
// resolveTaskRelationships filters entity relationships to task-to-task only,
// batch-resolves task IDs, and returns enriched RelationshipWithTask results.
func (s *EntityRelationshipService) resolveTaskRelationships(
    ctx context.Context,
    rels []*models.EntityRelationship,
    selfTaskID int64,
) ([]RelationshipWithTask, error)
```

### 3.3 New Repository Method

In `internal/repository/task_repository.go`:

```go
// GetByIDs retrieves multiple tasks by their IDs in a single query.
// Returns only found tasks; missing IDs are silently skipped.
// Returns empty slice (not nil) for empty input.
func (r *TaskRepository) GetByIDs(ctx context.Context, ids []int64) ([]*models.Task, error)
```

SQL pattern:
```sql
SELECT id, key, title, status, ...
FROM tasks
WHERE id IN (?, ?, ...)
```

### 3.4 Updated CLI Commands

After refactoring, the CLI command handlers become thin wrappers:

```go
func runTaskDeps(cmd *cobra.Command, args []string) error {
    taskKey := args[0]
    opts := parseTaskDepsOptions(cmd)

    taskSvc := cli.GetTaskService()
    relSvc := cli.GetEntityRelationshipService()

    task, err := taskSvc.GetTask(cmd.Context(), taskKey)
    if err != nil { ... }

    if opts.showTree {
        // Tree mode unchanged (already uses interfaces)
        ...
    }

    // Flat mode: service handles relationship resolution
    rels, err := relSvc.GetTaskRelationships(cmd.Context(), task.ID, opts.typeFilter)
    if err != nil { ... }

    // Format output (presentation logic stays in CLI)
    ...
}
```

### 3.5 Backward Compatibility

- The `entityRelRepoAdapter`, `TaskRepositoryInterfaceWithID`, and `RelationshipRepositoryInterface` types in `task_deps.go` remain for tree-building code. Tree building is not in scope for this feature.
- The `DependencyTree` struct and `buildDependencyTree`/`buildDependentsTree`/`renderTree` functions remain in `task_deps.go` as presentation-adjacent code. Moving them is out of scope.
- The `RelationshipWithTask` type already exists in `internal/services/task_service.go` and is reused.

### 3.6 Wiring Changes

In `internal/cli/services_global.go`, the `GetEntityRelationshipService()` accessor needs updating to inject the `TaskByIDResolver`:

```go
func GetEntityRelationshipService() *services.EntityRelationshipService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(...)
    }
    relRepo := repository.NewEntityRelationshipRepository(db)
    taskRepo := repository.NewTaskRepository(db)  // NEW
    return services.NewEntityRelationshipService(relRepo, taskRepo)  // UPDATED
}
```

---

## 4. Files Affected

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/services/entity_relationship_service.go` | Modified | Add TaskByIDResolver interface, 3 public methods, 1 private helper, update constructor |
| `internal/repository/task_repository.go` | Modified | Add GetByIDs batch method |
| `internal/cli/commands/task_deps.go` | Modified | Remove 3 helper functions, simplify command handlers |
| `internal/cli/services_global.go` | Modified | Update GetEntityRelationshipService wiring |
| `internal/services/entity_relationship_service_test.go` | Modified | Add tests for new methods with mocks |
| `internal/repository/task_repository_test.go` | Modified | Add tests for GetByIDs |

---

## 5. Task Breakdown

### Task 1: Move task relationship helpers from CLI to service layer (T-E21-F17-001)
- Add `TaskByIDResolver` interface to `entity_relationship_service.go`
- Update `EntityRelationshipService` constructor to accept optional `TaskByIDResolver`
- Implement `GetTaskRelationships`, `GetTaskBlockedBy`, `GetTaskBlocks` on service
- Implement `resolveTaskRelationships` private helper (initially using per-ID lookups)
- Update `services_global.go` wiring
- Refactor CLI commands to call service methods
- Remove `getTaskRelationshipsViaEntityRel`, `getTaskBlockedByViaEntityRel`, `getTaskBlocksViaEntityRel` from `task_deps.go`
- Add service tests with mocked repository and resolver

### Task 2: Add batch GetTasksByIDs repository method (T-E21-F17-002)
- Implement `GetByIDs(ctx, ids []int64) ([]*models.Task, error)` on `TaskRepository`
- Handle edge cases: empty input, missing IDs, SQL parameter limits
- Add `GetByIDs` to `TaskByIDResolver` interface
- Update `resolveTaskRelationships` to use batch fetching
- Add repository tests with real database

### Task 3: Unify near-duplicate helper functions (T-E21-F17-003)
- Verify `resolveTaskRelationships` handles all three use cases correctly
- Ensure `GetTaskRelationships`, `GetTaskBlockedBy`, `GetTaskBlocks` all delegate to shared helper
- Verify no duplicate resolution logic remains
- Add comprehensive tests covering all three methods via the shared helper

---

## 6. Test Plan

### Service Tests (mocked repositories)

| Test Case | Method | Scenario |
|-----------|--------|----------|
| GetTaskRelationships_NoRelationships | GetTaskRelationships | Task with no relationships returns empty slice |
| GetTaskRelationships_WithTypeFilter | GetTaskRelationships | Only matching types returned |
| GetTaskRelationships_SkipsNonTaskRelationships | GetTaskRelationships | Epic-to-task relationships filtered out |
| GetTaskRelationships_ResolverReturnsError | GetTaskRelationships | Graceful handling when task resolution fails |
| GetTaskBlockedBy_ReturnsDependencies | GetTaskBlockedBy | Returns outgoing depends_on relationships |
| GetTaskBlocks_ReturnsIncomingAndOutgoing | GetTaskBlocks | Returns both incoming depends_on and outgoing blocks |
| GetTaskBlocks_NoDuplicates | GetTaskBlocks | No duplicate entries when same task appears in both directions |

### Repository Tests (real database)

| Test Case | Method | Scenario |
|-----------|--------|----------|
| GetByIDs_MultipleTasks | GetByIDs | Returns all matching tasks |
| GetByIDs_EmptyInput | GetByIDs | Returns empty slice for empty input |
| GetByIDs_SomeMissing | GetByIDs | Returns only found tasks, no error for missing |
| GetByIDs_AllMissing | GetByIDs | Returns empty slice when no IDs match |
| GetByIDs_SingleID | GetByIDs | Works correctly with single ID |

### Integration Verification

- Run `shark task deps <key>` and verify output matches current behavior
- Run `shark task blocked-by <key>` and verify output matches current behavior
- Run `shark task blocks <key>` and verify output matches current behavior
- Run all three with `--json` flag and verify JSON structure is preserved

---

## 7. Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking existing CLI output | Low | High | Preserve exact output format; test with real commands |
| Constructor change breaks callers | Medium | Medium | Search all call sites of `NewEntityRelationshipService`; update wiring |
| Batch query SQL injection | Low | High | Use parameterized queries only |
| Performance regression in tree mode | Low | Low | Tree mode code unchanged; only flat mode refactored |

---

## 8. Design Decisions

**ADR-1: Add TaskByIDResolver to EntityRelationshipService (not TaskService)**

The relationship resolution logic belongs in `EntityRelationshipService` because:
- It orchestrates relationship data with task enrichment
- `TaskService` should not depend on `EntityRelationshipService` (avoids circular dependency)
- The resolver interface keeps the dependency lightweight and mockable

**ADR-2: Optional TaskByIDResolver (nil-safe)**

The `TaskByIDResolver` parameter is optional (can be nil) to avoid breaking existing callers that only use basic relationship CRUD. The new methods return an error if called without a resolver.

**ADR-3: Batch fetch in shared helper, not individual methods**

The batch optimization is applied once in `resolveTaskRelationships` rather than in each public method. This ensures all three methods benefit automatically and reduces maintenance surface.
