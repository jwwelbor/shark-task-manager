# Test Plan: E21-F17 Task Deps Service Layer Cleanup

**Feature:** E21-F17 — Task Deps Service Layer Cleanup
**Spec Reference:** `spec.md` (REQ-1, REQ-2, REQ-3)
**Epic UAT Reference:** `../uat-acceptance-plan.md` (P0: zero behavioral regression)
**Date:** 2026-03-22

---

## 1. AC Test Matrix

Each row traces to a specific acceptance criterion from `spec.md`.

---

### REQ-1: Move Three Helper Functions from CLI to EntityRelationshipService

**Acceptance Criteria (from spec.md §3.1):**
- AC-1.1: `GetTaskRelationships(ctx, taskID int64, typeFilter []string) ([]RelationshipWithTask, error)` exists on `EntityRelationshipService`
- AC-1.2: `GetTaskBlockedBy(ctx, taskID int64) ([]RelationshipWithTask, error)` exists on `EntityRelationshipService`
- AC-1.3: `GetTaskBlocks(ctx, taskID int64) ([]RelationshipWithTask, error)` exists on `EntityRelationshipService`
- AC-1.4: The three helper functions are removed from `internal/cli/commands/task_deps.go`
- AC-1.5: CLI command handlers call service methods instead of inline logic

---

#### TC-1.1: GetTaskRelationships returns all relationships for a task (no type filter)

| Field | Value |
|-------|-------|
| **AC** | AC-1.1 |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Service unit test (mocked repositories) |
| **Priority** | High |

**Preconditions:**
- `MockEntityRelationshipRepository` returns 2 relationships for task ID 10 (one `depends_on`, one `blocks`)
- `MockTaskByIDResolver` returns valid tasks for the related task IDs

**Test Steps:**
1. Construct `EntityRelationshipService` with mock repos
2. Call `GetTaskRelationships(ctx(), 10, nil)`
3. Assert no error
4. Assert returned slice contains 2 `RelationshipWithTask` entries
5. Assert each entry has populated `.Task` field (not nil)

**Expected Result:** 2 results returned; Task fields hydrated from resolver

**Edge Cases:**
- typeFilter=nil returns all relationship types
- Task with no relationships returns empty slice (not error)
- RelationshipRepository error propagates as error return

---

#### TC-1.2: GetTaskRelationships filters by relationship type

| Field | Value |
|-------|-------|
| **AC** | AC-1.1 |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Service unit test |
| **Priority** | High |

**Preconditions:**
- `MockEntityRelationshipRepository` returns 3 relationships: 2 `depends_on`, 1 `blocks`

**Test Steps:**
1. Call `GetTaskRelationships(ctx(), 10, []string{"depends_on"})`
2. Assert only 2 results returned (blocks relationship excluded)
3. Assert all returned relationships have type `depends_on`

**Expected Result:** Filtering by type reduces result set correctly

**Edge Cases:**
- Unknown type filter returns empty slice (not error)
- Multiple types in filter (["depends_on", "blocks"]) returns union

---

#### TC-1.3: GetTaskBlockedBy returns tasks that block the given task

| Field | Value |
|-------|-------|
| **AC** | AC-1.2 |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Service unit test |
| **Priority** | High |

**Preconditions:**
- Task 10 is blocked by tasks 20 and 30
- `MockEntityRelationshipRepository.ListByTaskID` returns relationships where task 10 is the target of `blocks` relationships
- `MockTaskByIDResolver` resolves IDs 20 and 30 to mock task structs

**Test Steps:**
1. Call `GetTaskBlockedBy(ctx(), 10)`
2. Assert no error
3. Assert 2 results returned
4. Assert result[0].Task.ID == 20 and result[1].Task.ID == 30

**Expected Result:** Returns tasks that hold a `blocks` relationship pointing to task 10

**Edge Cases:**
- Task not blocked by anything returns empty slice
- Resolver error for one ID propagates immediately (fail-fast)

---

#### TC-1.4: GetTaskBlocks returns tasks that the given task blocks

| Field | Value |
|-------|-------|
| **AC** | AC-1.3 |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Service unit test |
| **Priority** | High |

**Preconditions:**
- Task 10 blocks tasks 40 and 50
- Relationships exist with task 10 as source of `blocks` type

**Test Steps:**
1. Call `GetTaskBlocks(ctx(), 10)`
2. Assert no error
3. Assert 2 results returned
4. Assert Task fields for IDs 40 and 50 are populated

**Expected Result:** Returns tasks that task 10 is blocking

**Edge Cases:**
- Task blocking nothing returns empty slice
- Same task blocks multiple others — all returned, order stable

---

#### TC-1.5: Repository error in GetTaskRelationships propagates correctly

| Field | Value |
|-------|-------|
| **AC** | AC-1.1, AC-1.2, AC-1.3 |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Service unit test (error path) |
| **Priority** | High |

**Preconditions:**
- `MockEntityRelationshipRepository.ListByTaskID` returns `fmt.Errorf("db error")`

**Test Steps:**
1. Call `GetTaskRelationships(ctx(), 10, nil)`
2. Assert error is non-nil
3. Assert error message wraps "db error"
4. Assert returned slice is nil or empty

**Expected Result:** Repository error surfaces to caller with business context wrapping

---

#### TC-1.6: Task resolver error in GetTaskRelationships propagates correctly

| Field | Value |
|-------|-------|
| **AC** | AC-1.1 |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Service unit test (error path) |
| **Priority** | Medium |

**Preconditions:**
- Repository returns 1 relationship with related task ID 99
- `MockTaskByIDResolver.GetTaskByID(ctx, 99)` returns `fmt.Errorf("task 99 not found")`

**Test Steps:**
1. Call `GetTaskRelationships(ctx(), 10, nil)`
2. Assert error contains "task 99 not found"

**Expected Result:** Resolver errors propagate; partial results not returned

---

#### TC-1.7: CLI task_deps.go no longer contains inline business logic

| Field | Value |
|-------|-------|
| **AC** | AC-1.4, AC-1.5 |
| **Test File** | Static code verification (not a runtime test) |
| **Test Type** | Code structure assertion |
| **Priority** | High |

**Verification Steps:**
1. `grep -n "getTaskRelationshipsViaEntityRel\|getTaskBlockedByViaEntityRel\|getTaskBlocksViaEntityRel" internal/cli/commands/task_deps.go` returns no matches
2. Command handlers call service methods (e.g., `svc.GetTaskRelationships(...)`) not inline loops
3. `make build` succeeds

**Expected Result:** Three helper functions absent from CLI file; service calls present

---

### REQ-2: Add GetByIDs Batch Query to TaskRepository

**Acceptance Criteria (from spec.md §3.2):**
- AC-2.1: `GetByIDs(ctx context.Context, ids []int64) ([]*models.Task, error)` exists on `TaskRepository`
- AC-2.2: Implementation uses a single `WHERE id IN (...)` query (not N loops)
- AC-2.3: Empty input slice returns empty slice without a DB query
- AC-2.4: Result order matches input ID order OR is deterministic

---

#### TC-2.1: GetByIDs returns all tasks for provided IDs

| Field | Value |
|-------|-------|
| **AC** | AC-2.1, AC-2.2 |
| **Test File** | `internal/repository/task_repository_test.go` |
| **Test Type** | Repository integration test (real DB, cleanup required) |
| **Priority** | High |

**Preconditions:**
- Test epic `E95`, feature `E95-F01` exist (created and deferred for cleanup)
- 3 tasks with known keys `T-E95-F01-011`, `T-E95-F01-012`, `T-E95-F01-013` created and their IDs captured

**Test Steps:**
1. Call `repo.GetByIDs(ctx, []int64{task1.ID, task2.ID, task3.ID})`
2. Assert no error
3. Assert 3 results returned
4. Assert all keys match expected task keys

**Expected Result:** All 3 tasks returned in single call

**Cleanup:** DELETE tasks by ID in defer; DELETE feature; DELETE epic

**Edge Cases:**
- Only some IDs exist in DB: returns only found tasks (no error for missing IDs)

---

#### TC-2.2: GetByIDs with empty slice returns empty result

| Field | Value |
|-------|-------|
| **AC** | AC-2.3 |
| **Test File** | `internal/repository/task_repository_test.go` |
| **Test Type** | Repository integration test |
| **Priority** | High |

**Test Steps:**
1. Call `repo.GetByIDs(ctx, []int64{})`
2. Assert no error
3. Assert returned slice is non-nil and length 0

**Expected Result:** Empty input → empty output, no DB query executed

**Why This Matters:** The `IN ()` SQL clause with zero elements is invalid in SQLite and would panic/error without an early-return guard.

---

#### TC-2.3: GetByIDs with nil slice behaves like empty slice

| Field | Value |
|-------|-------|
| **AC** | AC-2.3 |
| **Test File** | `internal/repository/task_repository_test.go` |
| **Test Type** | Repository integration test |
| **Priority** | Medium |

**Test Steps:**
1. Call `repo.GetByIDs(ctx, nil)`
2. Assert no error
3. Assert returned slice is empty (length 0)

**Expected Result:** nil input treated safely; no DB error

---

#### TC-2.4: GetByIDs with nonexistent IDs returns empty slice

| Field | Value |
|-------|-------|
| **AC** | AC-2.1 |
| **Test File** | `internal/repository/task_repository_test.go` |
| **Test Type** | Repository integration test |
| **Priority** | Medium |

**Test Steps:**
1. Call `repo.GetByIDs(ctx, []int64{-1, -2, -3})`
2. Assert no error
3. Assert returned slice is empty

**Expected Result:** Non-matching IDs produce empty slice (not error), consistent with `ListByFeature` behavior

---

#### TC-2.5: GetByIDs with single ID returns single task

| Field | Value |
|-------|-------|
| **AC** | AC-2.1 |
| **Test File** | `internal/repository/task_repository_test.go` |
| **Test Type** | Repository integration test |
| **Priority** | Medium |

**Preconditions:** One test task created

**Test Steps:**
1. Call `repo.GetByIDs(ctx, []int64{task1.ID})`
2. Assert 1 result returned with correct key

**Expected Result:** Single-element IN clause works correctly

---

### REQ-3: Unify resolveTaskRelationships Private Helper

**Acceptance Criteria (from spec.md §3.3):**
- AC-3.1: Private `resolveTaskRelationships(ctx, rels []models.EntityRelationship, selfTaskID int64) ([]RelationshipWithTask, error)` helper exists in `entity_relationship_service.go`
- AC-3.2: `GetTaskRelationships`, `GetTaskBlockedBy`, `GetTaskBlocks` all delegate to this helper
- AC-3.3: Helper calls `TaskByIDResolver.GetByIDs()` (or equivalent batch) once per invocation, not once per relationship
- AC-3.4: The helper filters out the self-task (selfTaskID) from results

---

#### TC-3.1: resolveTaskRelationships hydrates Task field for all relationships

| Field | Value |
|-------|-------|
| **AC** | AC-3.1, AC-3.3 |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Service unit test (tested indirectly via public methods) |
| **Priority** | High |

**Preconditions:**
- MockTaskByIDResolver captures call count
- 3 relationships passed to the public method that delegates to helper

**Test Steps:**
1. Call `GetTaskRelationships(ctx(), taskID, nil)` with 3 mock relationships
2. Assert resolver `GetByIDs` was called exactly once (not 3 times)
3. Assert all 3 returned `RelationshipWithTask.Task` fields are non-nil

**Expected Result:** Batch resolution — single GetByIDs call regardless of relationship count

---

#### TC-3.2: resolveTaskRelationships excludes self-task from results

| Field | Value |
|-------|-------|
| **AC** | AC-3.4 |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Service unit test |
| **Priority** | High |

**Preconditions:**
- Relationships include one pointing back to the selfTaskID (circular reference scenario)
- Mock resolver returns the self-task as one of the resolved tasks

**Test Steps:**
1. Call `GetTaskRelationships(ctx(), selfTaskID, nil)` where one relationship's related ID == selfTaskID
2. Assert the self-task is NOT present in results

**Expected Result:** Self-reference filtered out; remaining tasks returned normally

**Edge Cases:**
- All relationships point to self: empty results returned (no error)

---

#### TC-3.3: Delegation pattern — all three public methods use shared helper

| Field | Value |
|-------|-------|
| **AC** | AC-3.2 |
| **Test File** | Static code verification |
| **Test Type** | Code structure assertion |
| **Priority** | High |

**Verification:**
1. `grep -n "resolveTaskRelationships" internal/services/entity_relationship_service.go` shows 4 occurrences: 1 definition + 3 call sites
2. No relationship-hydration loop exists outside `resolveTaskRelationships`

**Expected Result:** DRY implementation — single hydration path

---

#### TC-3.4: Existing EntityRelationshipService tests still pass after constructor change

| Field | Value |
|-------|-------|
| **AC** | AC-3.1 (no regression) |
| **Test File** | `internal/services/entity_relationship_service_test.go` |
| **Test Type** | Regression (existing tests) |
| **Priority** | Critical |

**Background:** After F17, `NewEntityRelationshipService` takes a second parameter `TaskByIDResolver`. All existing tests using the old single-argument constructor must be updated.

**Test Steps:**
1. Update all `NewEntityRelationshipService(mockRepo)` calls in existing tests to `NewEntityRelationshipService(mockRepo, mockTaskResolver)` or `NewEntityRelationshipService(mockRepo, nil)` where task resolution is not needed
2. Run `go test ./internal/services/... -v`
3. Assert all previously passing tests still pass

**Expected Result:** Zero new test failures from constructor signature change

---

## 2. Integration Scenarios

These scenarios test cross-component boundaries — verifying that the refactoring does not break end-to-end behavior observable by a CLI user.

---

### IS-1: shark task deps <key> output identical before and after refactoring

**Boundary:** CLI command handler → EntityRelationshipService → Repository

**Scenario:** A task with 2 `depends_on` relationships and 1 `blocks` relationship.

**Before (current behavior):** `task_deps.go` helpers call `GetTaskByID` in a loop, producing a tree output.

**After (target behavior):** `task_deps.go` calls `entityRelSvc.GetTaskRelationships(...)`, produces identical output.

**Validation Method:**
1. Capture `shark task deps <key> --json` output before the refactoring commit (or from spec examples)
2. After refactoring, run `shark task deps <key> --json` against same data
3. Assert JSON output is byte-for-byte identical

**Components Crossing Boundary:**
- `task_deps.go` → `EntityRelationshipService.GetTaskRelationships`
- `EntityRelationshipService` → `EntityRelationshipRepository.ListByTaskID`
- `EntityRelationshipService` → `TaskByIDResolver.GetByIDs` (new) or `GetTaskByID` (existing loop)

---

### IS-2: shark task blocked-by <key> output identical before and after

**Boundary:** CLI command handler → EntityRelationshipService → Repository

**Scenario:** Task blocked by 2 other tasks.

**Validation Method:**
1. Capture `shark task blocked-by <key>` output before change
2. After refactoring, assert identical output
3. Verify no extra DB round-trips with `--verbose` flag

---

### IS-3: shark task blocks <key> output identical before and after

**Boundary:** CLI command handler → EntityRelationshipService → Repository

**Scenario:** Task blocks 3 other tasks.

**Validation Method:**
1. Capture `shark task blocks <key>` output
2. Assert identical output after refactoring
3. Verify output order is stable (deterministic)

---

### IS-4: Zero regression on make test after full implementation

**Boundary:** All packages — full test suite

**Scenario:** After all 3 tasks (T-E21-F17-001, T-E21-F17-002, T-E21-F17-003) are implemented.

**Validation Method:**
```bash
make fmt && make lint && make test
```

All three must exit 0. This is the mandatory quality gate from `uat-acceptance-plan.md`.

---

### IS-5: EntityRelationshipService constructor wiring in CLI services_global.go

**Boundary:** `services_global.go` → `EntityRelationshipService` constructor

**Scenario:** After F17, `GetEntityRelationshipService()` in `services_global.go` must pass a `TaskByIDResolver` implementation (the concrete `*repository.TaskRepository`) as the second argument.

**Validation Method:**
1. Build succeeds: `make shark`
2. `shark task deps <key>` runs without panic at runtime
3. Compiler rejects build if second constructor argument is missing

---

### IS-6: N+1 elimination — single DB round-trip for batch resolve

**Boundary:** `EntityRelationshipService.resolveTaskRelationships` → `TaskRepository.GetByIDs`

**Scenario:** Task with 5 relationships. Before: 5 DB calls (N+1). After: 1 call.

**Validation Method:**
- In service test: `MockTaskByIDResolver` with call counter asserts `GetByIDs` called once
- Not a DB integration test — mock-level verification is sufficient
- Corresponds to TC-3.1

---

## 3. Test Infrastructure

### 3.1 Existing Infrastructure (Do Not Recreate)

| Component | Location | Status |
|-----------|----------|--------|
| `MockEntityRelationshipRepository` | `internal/services/entity_relationship_service_test.go` | Exists — use and extend |
| `ctx()` helper function | `internal/services/entity_relationship_service_test.go` | Exists (`func ctx() context.Context { return context.Background() }`) |
| `NewEntityRelationshipService(repo)` calls | `entity_relationship_service_test.go` | Exists — must update to 2-arg form |
| Real DB setup: `test.GetTestDB()` | `internal/test/testdb.go` | Exists — use in repository tests |
| Test epic/feature creation pattern | `internal/repository/task_repository_test.go` | Exists — follow `E95` pattern with dedicated test prefix |
| `require.NoError` / `assert.Equal` | `testify` library already imported | Exists |
| `MockTaskRepositoryForTree` / `MockRelationshipRepositoryForTree` | `internal/cli/commands/task_deps_tree_test.go` | Exists — reference only; tree tests are out of scope for F17 |

### 3.2 New Infrastructure to Create

#### MockTaskByIDResolver

**Location:** `internal/services/entity_relationship_service_test.go`

**Purpose:** Satisfies the `TaskByIDResolver` interface defined in `entity_relationship_service.go` for service-layer tests. Must NOT use real DB.

**Pattern (function-field mock, matching existing `MockEntityRelationshipRepository`):**

```go
type MockTaskByIDResolver struct {
    GetByIDsFunc func(ctx context.Context, ids []int64) ([]*models.Task, error)
}

func (m *MockTaskByIDResolver) GetByIDs(ctx context.Context, ids []int64) ([]*models.Task, error) {
    if m.GetByIDsFunc != nil {
        return m.GetByIDsFunc(ctx, ids)
    }
    return nil, fmt.Errorf("GetByIDs not implemented in mock")
}
```

**Usage pattern in tests:**

```go
mockResolver := &MockTaskByIDResolver{
    GetByIDsFunc: func(ctx context.Context, ids []int64) ([]*models.Task, error) {
        var tasks []*models.Task
        for _, id := range ids {
            tasks = append(tasks, &models.Task{BaseEntity: models.BaseEntity{
                ID: id, Key: fmt.Sprintf("T-MOCK-%03d", id),
            }})
        }
        return tasks, nil
    },
}
svc := NewEntityRelationshipService(mockRelRepo, mockResolver)
```

#### TaskByIDResolver Interface

**Location:** `internal/services/entity_relationship_service.go` (to be defined by developer in T-E21-F17-001)

**Expected signature:**
```go
type TaskByIDResolver interface {
    GetByIDs(ctx context.Context, ids []int64) ([]*models.Task, error)
}
```

**Satisfied by:** `*repository.TaskRepository` after T-E21-F17-002 adds `GetByIDs` method

#### Test Data Prefix for Repository Tests

**Convention:** Use `T-E95-F17-0XX` keys for GetByIDs repository tests to avoid conflicts with existing `T-E95-F01-001` slug test data.

Example:
- `T-E95-F17-011` — batch test task 1
- `T-E95-F17-012` — batch test task 2
- `T-E95-F17-013` — batch test task 3

Cleanup pattern (following existing repository test conventions):
```go
// BEFORE test
_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E95-F17-%'")

// AFTER test (defer)
defer database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E95-F17-%'")
```

### 3.3 Existing Tests Requiring Updates

The following existing tests will require modification when `NewEntityRelationshipService` gains a second constructor parameter:

| Test Function | File | Change Required |
|--------------|------|-----------------|
| All tests calling `NewEntityRelationshipService(mockRepo)` | `entity_relationship_service_test.go` | Add second argument: `NewEntityRelationshipService(mockRepo, nil)` or a mock resolver |

When `TaskByIDResolver` is not needed by the test (e.g., cycle detection tests, CRUD tests not exercising relationship hydration), pass `nil` as the second argument. The service must handle `nil` resolver gracefully for these existing test paths, OR the existing tests must each provide a minimal mock resolver.

**Recommendation:** Pass `nil` for tests that don't test the hydration path, and guard in the service: `if s.taskResolver != nil { ... }`.

---

## 4. Exit Gate Verification

Before marking QA complete:

| Gate | Check |
|------|-------|
| Every AC in spec.md has at least one test case | REQ-1: TC-1.1 through TC-1.7; REQ-2: TC-2.1 through TC-2.5; REQ-3: TC-3.1 through TC-3.4 |
| Edge cases identified for each AC | Empty/nil IDs (TC-2.2, TC-2.3), nonexistent IDs (TC-2.4), self-reference (TC-3.2), repository errors (TC-1.5), resolver errors (TC-1.6) |
| Integration scenarios cover cross-component boundaries | IS-1 through IS-6 cover CLI→Service→Repo boundary, constructor wiring, and N+1 elimination |
| Test patterns reference existing infrastructure | MockEntityRelationshipRepository extended; `test.GetTestDB()` used for repo tests; function-field mock pattern followed throughout |
| `make fmt && make lint && make test` passes | IS-4 |
| Zero behavioral regression at CLI output level | IS-1, IS-2, IS-3 |
