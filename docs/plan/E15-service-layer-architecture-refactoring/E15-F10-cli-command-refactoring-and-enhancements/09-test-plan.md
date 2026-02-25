# Test Plan: E15-F10 CLI Command Refactoring and Enhancements

**Feature**: E15-F10-cli-command-refactoring-and-enhancements
**Date**: 2026-02-18
**Author**: QA Agent

---

## 1. Overview

This test plan covers the verification strategy for E15-F10, the final phase of the E15 service layer architecture refactoring. The feature has two primary goals:

1. Create `IdeaService` and extend `TaskService`, `EpicService`, and `FeatureService` with missing methods (T-E15-F10-004)
2. Refactor seven fat-controller CLI command files to thin wrappers that call services (T-E15-F10-001, T-E15-F10-002, T-E15-F10-003)

The primary quality concern is **architecture correctness**: after all tasks are complete, no production CLI command handler may call `repository.New*()` or `cli.GetDB()` directly.

---

## 2. Testing Strategy

### 2.1 Test Layer Decisions

| Layer | Test Approach | Why |
|-------|--------------|-----|
| Service methods (new/extended) | Unit tests with mocked repositories | Services must be testable without a database |
| CLI command handlers (refactored) | Unit tests with mocked services | Commands must be testable without a database |
| Architecture compliance | Static grep assertion | Automated check that no fat-controller pattern remains |
| Regression | `make test` (full suite) | Ensure nothing broken by refactoring |
| Smoke tests | Manual CLI invocation | Verify end-to-end behavior unchanged |

### 2.2 Golden Rule Application

- **IdeaService tests** (`internal/services/idea_service_test.go`): Use `MockIdeaRepository` — no real database
- **TaskService extension tests** (`internal/services/task_service_test.go`): Use `MockTaskRepository` — no real database
- **EpicService/FeatureService extension tests**: Use existing mock infrastructure — no real database
- **CLI command tests** (`internal/cli/commands/idea_test.go`, etc.): Use mocked services — no real database
- **Repository tests** (`internal/repository/*_test.go`): The only layer that may use real database (pre-existing tests, not modified by this feature)

### 2.3 Coverage Targets

| Component | Target Coverage | Priority |
|-----------|----------------|----------|
| `IdeaService` (all methods) | >= 80% line coverage | Must-Have |
| `TaskService` new methods (`AddDependency`, `RemoveDependency`, `ListDependencies`, `UnlinkFile`) | >= 80% line coverage | Must-Have |
| `EpicService.LinkDocument`, `EpicService.UnlinkDocument` | >= 80% line coverage | Should-Have |
| `FeatureService.LinkDocument`, `FeatureService.UnlinkDocument` | >= 80% line coverage | Should-Have |
| CLI command handlers (refactored) | Argument parsing and output path coverage | Should-Have |

---

## 3. Service Layer Tests

### 3.1 IdeaService Tests (`internal/services/idea_service_test.go`)

#### Mock Infrastructure Required

```go
// MockIdeaRepository implements IdeaRepository interface for testing
type MockIdeaRepository struct {
    CreateFunc                func(ctx context.Context, idea *models.Idea) error
    GetByIDFunc               func(ctx context.Context, id int64) (*models.Idea, error)
    GetByKeyFunc              func(ctx context.Context, key string) (*models.Idea, error)
    ListFunc                  func(ctx context.Context, filter *IdeaFilters) ([]*models.Idea, error)
    UpdateFunc                func(ctx context.Context, idea *models.Idea) error
    DeleteFunc                func(ctx context.Context, id int64) error
    MarkAsConvertedFunc       func(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error
    GetNextSequenceForDateFunc func(ctx context.Context, dateStr string) (int, error)
}
```

#### TC-IDEA-001: CreateIdea — Key Generation (First of Day)

**Priority**: High — Core business logic
**Test type**: Unit with mocked repository

**Preconditions**: `GetNextSequenceForDate` returns 1 (no ideas for today)

**Steps**:
1. Create `MockIdeaRepository` with `GetNextSequenceForDateFunc` returning 1
2. Create `IdeaService` with the mock
3. Call `CreateIdea(ctx, CreateIdeaInput{Title: "Test Idea", Description: "..."})`

**Expected Results**:
- Returned idea has key matching `I-YYYY-MM-DD-01` pattern (today's date)
- `Create` called on repository with the correct idea
- No error returned

#### TC-IDEA-002: CreateIdea — Key Generation (Subsequent on Same Day)

**Priority**: High
**Test type**: Unit with mocked repository

**Preconditions**: `GetNextSequenceForDate` returns 3 (two ideas already exist today)

**Steps**:
1. Mock returns sequence 3
2. Call `CreateIdea`

**Expected Results**:
- Returned idea has key `I-YYYY-MM-DD-03`
- Sequence padding uses two digits (01, 02, ... 09, 10)

#### TC-IDEA-003: CreateIdea — Empty Title Validation

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Call `CreateIdea(ctx, CreateIdeaInput{Title: "", Description: "test"})`

**Expected Results**:
- Returns non-nil error
- Error message contains "title" or "required"
- `Create` not called on repository

#### TC-IDEA-004: CreateIdea — Repository Failure

**Priority**: High
**Test type**: Unit

**Steps**:
1. `GetNextSequenceForDate` returns 1
2. `Create` returns `fmt.Errorf("database error")`
3. Call `CreateIdea`

**Expected Results**:
- Returns error wrapping the repository error
- Error message contains "create" or "idea"

#### TC-IDEA-005: GetIdea — Happy Path

**Priority**: High
**Test type**: Unit

**Steps**:
1. Mock `GetByKey` returns `&models.Idea{Key: "I-2026-02-18-01", Title: "Test"}`
2. Call `GetIdea(ctx, "I-2026-02-18-01")`

**Expected Results**:
- Returns idea with matching key
- No error

#### TC-IDEA-006: GetIdea — Not Found

**Priority**: High
**Test type**: Unit

**Steps**:
1. Mock `GetByKey` returns `nil, &NotFoundError{...}`
2. Call `GetIdea(ctx, "I-2026-02-18-99")`

**Expected Results**:
- Returns nil idea
- Returns error (not found)

#### TC-IDEA-007: ListIdeas — No Filter

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Mock `List` returns slice of 3 ideas
2. Call `ListIdeas(ctx, IdeaFilters{})`

**Expected Results**:
- Returns all 3 ideas
- No error

#### TC-IDEA-008: ListIdeas — Status Filter

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Mock `List` returns ideas, with filter passed through
2. Call `ListIdeas(ctx, IdeaFilters{Status: "active"})`

**Expected Results**:
- Filter is passed to repository (verify mock receives correct filter)
- Returns ideas from mock

#### TC-IDEA-009: UpdateIdea — Happy Path

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. `GetByKey` returns existing idea
2. `Update` returns nil
3. Call `UpdateIdea(ctx, "I-2026-02-18-01", UpdateIdeaInput{Title: ptr("Updated")})`

**Expected Results**:
- Repository `Update` called with modified idea
- Returned idea has updated title
- No error

#### TC-IDEA-010: UpdateIdea — Not Found

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. `GetByKey` returns not-found error
2. Call `UpdateIdea`

**Expected Results**:
- Error returned, update not attempted

#### TC-IDEA-011: DeleteIdea — Happy Path

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. `GetByKey` returns existing idea with ID=5
2. `Delete` returns nil
3. Call `DeleteIdea(ctx, "I-2026-02-18-01")`

**Expected Results**:
- `Delete` called with correct ID (5)
- No error

#### TC-IDEA-012: ConvertIdea — Happy Path

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. `GetByKey` returns existing idea
2. `MarkAsConverted` returns nil
3. Call `ConvertIdea(ctx, "I-2026-02-18-01", "task", "E15-F10-001")`

**Expected Results**:
- `MarkAsConverted` called with correct parameters
- No error

#### TC-IDEA-013: ConvertIdea — Already Converted

**Priority**: Low
**Test type**: Unit

**Steps**:
1. `GetByKey` returns idea with `ConvertedToType` already set
2. Call `ConvertIdea`

**Expected Results**:
- Returns error if service enforces "cannot convert twice" rule
- OR succeeds if the service allows reconversion (verify architecture document intent)

#### TC-IDEA-014: Key Generation — Sequence Padding Boundary

**Priority**: Medium
**Test type**: Unit (table-driven)

**Test cases**:
- Sequence 1 → key ends in `-01`
- Sequence 9 → key ends in `-09`
- Sequence 10 → key ends in `-10`
- Sequence 99 → key ends in `-99`

---

### 3.2 TaskService Extension Tests (`internal/services/task_service_test.go`)

The existing `MockTaskRepository` in `task_service_test.go` must be extended to include methods for dependency management:

```go
// Additional fields to add to MockTaskRepository:
AddDependencyFunc    func(ctx context.Context, taskID, depID int64) error
RemoveDependencyFunc func(ctx context.Context, taskID, depID int64) error
ListDependenciesFunc func(ctx context.Context, taskID int64) ([]*models.Task, error)
UnlinkFileFunc       func(ctx context.Context, taskID int64) error
```

#### TC-TASK-DEP-001: AddDependency — Happy Path

**Priority**: High
**Test type**: Unit

**Steps**:
1. Mock `GetByKey` for both task and dependency keys — both exist
2. Mock `AddDependency` returns nil
3. Call `TaskService.AddDependency(ctx, "E15-F10-001", "E15-F10-002")`

**Expected Results**:
- `AddDependency` called on repository with correct IDs
- No error returned

#### TC-TASK-DEP-002: AddDependency — Task Not Found

**Priority**: High
**Test type**: Unit

**Steps**:
1. `GetByKey` for the task key returns not-found error
2. Call `AddDependency`

**Expected Results**:
- Error returned, dependency not added
- Error message indicates task not found

#### TC-TASK-DEP-003: AddDependency — Dependency Task Not Found

**Priority**: High
**Test type**: Unit

**Steps**:
1. `GetByKey` for task key succeeds
2. `GetByKey` for dep key returns not-found error
3. Call `AddDependency`

**Expected Results**:
- Error returned, dependency not added
- Error indicates dependency task not found

#### TC-TASK-DEP-004: AddDependency — Self-Dependency Prevented

**Priority**: High
**Test type**: Unit

**Steps**:
1. Both `GetByKey` calls return the same task (same ID)
2. Call `AddDependency(ctx, "E15-F10-001", "E15-F10-001")`

**Expected Results**:
- Error returned with message like "task cannot depend on itself"
- Repository `AddDependency` not called

#### TC-TASK-DEP-005: RemoveDependency — Happy Path

**Priority**: High
**Test type**: Unit

**Steps**:
1. Both tasks exist
2. `RemoveDependency` returns nil
3. Call `TaskService.RemoveDependency(ctx, "E15-F10-001", "E15-F10-002")`

**Expected Results**:
- `RemoveDependency` called on repository
- No error

#### TC-TASK-DEP-006: RemoveDependency — Dependency Does Not Exist

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Tasks exist but `RemoveDependency` returns not-found error
2. Call `RemoveDependency`

**Expected Results**:
- Error returned
- Error message indicates dependency not found

#### TC-TASK-DEP-007: ListDependencies — Happy Path

**Priority**: High
**Test type**: Unit

**Steps**:
1. `GetByKey` returns task with ID=1
2. `ListDependencies` returns 2 dependency tasks
3. Call `TaskService.ListDependencies(ctx, "E15-F10-001")`

**Expected Results**:
- Returns slice of 2 tasks
- No error

#### TC-TASK-DEP-008: ListDependencies — No Dependencies

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. `GetByKey` succeeds
2. `ListDependencies` returns empty slice
3. Call `ListDependencies`

**Expected Results**:
- Returns empty (not nil) slice
- No error

#### TC-TASK-UNLINK-001: UnlinkFile — Happy Path

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. `GetByKey` returns task with `FilePath` set
2. `UnlinkFile` (or `Update`) returns nil
3. Call `TaskService.UnlinkFile(ctx, "E15-F10-001")`

**Expected Results**:
- Task's `FilePath` cleared in repository
- No error

#### TC-TASK-UNLINK-002: UnlinkFile — Task Not Found

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. `GetByKey` returns not-found error
2. Call `UnlinkFile`

**Expected Results**:
- Error returned

---

### 3.3 EpicService Extension Tests (`internal/services/epic_service_test.go`)

#### TC-EPIC-LINK-001: LinkDocument — Happy Path

**Priority**: Should-Have
**Test type**: Unit

**Steps**:
1. `GetByKey` returns epic with ID=1
2. Document linking succeeds
3. Call `EpicService.LinkDocument(ctx, "E15", "docs/plan/spec.md")`

**Expected Results**:
- Document linked to epic
- No error

#### TC-EPIC-LINK-002: LinkDocument — Epic Not Found

**Priority**: Should-Have
**Test type**: Unit

**Steps**:
1. `GetByKey` returns not-found error

**Expected Results**:
- Error returned

#### TC-EPIC-UNLINK-001: UnlinkDocument — Happy Path

**Priority**: Should-Have
**Test type**: Unit

**Steps**:
1. Epic and document both found
2. Unlink succeeds

**Expected Results**:
- Document unlinked
- No error

#### TC-EPIC-UNLINK-002: UnlinkDocument — Document Not Linked

**Priority**: Low
**Test type**: Unit

**Expected Results**:
- Error or no-op (verify service design intent)

---

### 3.4 FeatureService Extension Tests (`internal/services/feature_service_test.go`)

Mirror of EpicService tests above, applied to `FeatureService.LinkDocument` and `FeatureService.UnlinkDocument`:

- TC-FEAT-LINK-001: LinkDocument — Happy Path
- TC-FEAT-LINK-002: LinkDocument — Feature Not Found
- TC-FEAT-UNLINK-001: UnlinkDocument — Happy Path
- TC-FEAT-UNLINK-002: UnlinkDocument — Document Not Linked

---

### 3.5 EpicService Integration Verification Tests (T-E15-F10-002)

These tests verify the existing EpicService integration is complete and clean. Add to `internal/services/epic_service_test.go`.

#### TC-EPIC-COMPAT-001: CompleteEpic — All Tasks Completed

**Priority**: High
**Test type**: Unit

**Steps**:
1. Mock returns epic with all features/tasks completed
2. `force=false`
3. Call `CompleteEpic(ctx, "E15", false)`

**Expected Results**:
- Epic status updated
- No error

#### TC-EPIC-COMPAT-002: CompleteEpic — Incomplete Tasks, No Force

**Priority**: High
**Test type**: Unit

**Steps**:
1. Mock returns epic with incomplete tasks
2. `force=false`
3. Call `CompleteEpic`

**Expected Results**:
- Error returned
- Error indicates incomplete tasks exist

#### TC-EPIC-COMPAT-003: CompleteEpic — Incomplete Tasks, Force=true

**Priority**: High
**Test type**: Unit

**Steps**:
1. Mock returns epic with incomplete tasks
2. `force=true`
3. Call `CompleteEpic(ctx, "E15", true)`

**Expected Results**:
- Epic completed despite incomplete tasks
- No error

#### TC-EPIC-COMPAT-004: GetImpediments — No Blocked Tasks

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. No blocked tasks in mock
2. Call `GetImpediments(ctx, "E15")`

**Expected Results**:
- Returns empty slice
- No error

#### TC-EPIC-COMPAT-005: GetImpediments — With Blocked Tasks

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Mock returns 2 blocked tasks
2. Call `GetImpediments`

**Expected Results**:
- Returns 2 impediments with task details and age

#### TC-EPIC-COMPAT-006: CascadeStatusToFeaturesAndTasks — Propagates Correctly

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Mock verifies cascade called with correct statuses
2. Call `CascadeStatusToFeaturesAndTasks`

**Expected Results**:
- Repository cascade method called with expected parameters

---

## 4. CLI Command Tests

### 4.1 Testing Pattern for Refactored Commands

After refactoring, all CLI command tests must use **mocked services**, not mocked repositories or real databases. The pattern established in `task_workflow_test.go` and similar files should be followed.

For each refactored command, the test verifies:
1. **Argument parsing**: Correct values extracted from `cmd.Args` and flags
2. **Service call**: Correct service method called with correct parameters
3. **Output formatting**: JSON and human-readable paths both covered
4. **Error handling**: Service errors propagated or formatted correctly

### 4.2 idea.go Command Tests (`internal/cli/commands/idea_test.go`)

The existing `idea_test.go` tests `generateIdeaKey` using a real repository. After refactoring, these tests must be updated: key generation logic moves to `IdeaService`, so `idea_test.go` should instead test that the CLI command calls `GetIdeaService().CreateIdea(...)` with correct input.

**Note**: Existing tests using `repository.IdeaRepository` directly will need updating when `idea.go` is refactored to remove the local `IdeaRepository` interface.

#### TC-CLI-IDEA-001: runIdeaCreate — Calls Service with Correct Input

**Priority**: High
**Test type**: Unit with mocked service

**Mock setup**: `MockIdeaService.CreateIdeaFunc` captures input and returns stub idea

**Steps**:
1. Execute idea create command with title, description, tags
2. Verify mock was called with correct `CreateIdeaInput`
3. Verify success message output

**Expected Results**:
- Service called once with expected input fields
- Output contains idea key

#### TC-CLI-IDEA-002: runIdeaCreate — JSON Output

**Priority**: High
**Test type**: Unit

**Steps**:
1. Set `cli.GlobalConfig.JSON = true`
2. Execute idea create command
3. Verify JSON output format

**Expected Results**:
- Valid JSON output containing idea fields
- No human-readable messages mixed in

#### TC-CLI-IDEA-003: runIdeaList — Calls Service with Filters

**Priority**: High
**Test type**: Unit

**Steps**:
1. Execute `idea list --status=active`
2. Verify `ListIdeas` called with `IdeaFilters{Status: "active"}`

#### TC-CLI-IDEA-004: runIdeaGet — Happy Path

**Priority**: High
**Test type**: Unit

**Steps**:
1. Execute `idea get I-2026-02-18-01`
2. Verify `GetIdea` called with correct key

#### TC-CLI-IDEA-005: runIdeaGet — Not Found

**Priority**: High
**Test type**: Unit

**Steps**:
1. Mock returns not-found error
2. Execute `idea get I-2026-02-18-99`

**Expected Results**:
- Error output or non-zero exit
- No panic

#### TC-CLI-IDEA-006: runIdeaConvert — Calls Service

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Execute `idea convert I-2026-02-18-01 --type=task --key=E15-F10-001`
2. Verify `ConvertIdea` called with correct parameters

---

### 4.3 task_deps.go Command Tests (`internal/cli/commands/task_deps_test.go`)

**New test file** — does not exist yet, must be created.

#### TC-CLI-DEP-001: runTaskDepAdd — Calls TaskService.AddDependency

**Priority**: High
**Test type**: Unit with mocked service

**Steps**:
1. Execute `task dep add E15-F10-001 E15-F10-002`
2. Verify `AddDependency(ctx, "E15-F10-001", "E15-F10-002")` called

#### TC-CLI-DEP-002: runTaskDepAdd — Self-Dependency Error Display

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Service returns self-dependency error
2. Execute command
3. Verify error displayed appropriately

#### TC-CLI-DEP-003: runTaskDepRemove — Calls TaskService.RemoveDependency

**Priority**: High
**Test type**: Unit

**Steps**:
1. Execute `task dep remove E15-F10-001 E15-F10-002`
2. Verify `RemoveDependency` called with correct keys

#### TC-CLI-DEP-004: runTaskDepList — Calls TaskService.ListDependencies

**Priority**: High
**Test type**: Unit

**Steps**:
1. Execute `task dep list E15-F10-001`
2. Verify `ListDependencies` called
3. Verify output contains dependency task keys

#### TC-CLI-DEP-005: runTaskDepList — JSON Output

**Priority**: Medium
**Test type**: Unit

**Steps**:
1. Set JSON mode
2. Execute `task dep list E15-F10-001`
3. Verify valid JSON with dependency array

---

### 4.4 related_docs.go Command Tests (`internal/cli/commands/related_docs_test.go`)

Existing tests use mocked repositories directly (e.g., `NewMockDocumentRepository()`). After refactoring to use service layer, these tests must be updated to use mocked services instead.

#### TC-CLI-RELDOC-001: Link Document to Epic — Calls EpicService.LinkDocument

**Priority**: High
**Test type**: Unit with mocked service

**Steps**:
1. Execute `related-docs link --epic=E15 "OAuth Spec" docs/oauth.md`
2. Verify `GetEpicService().LinkDocument(ctx, "E15", "docs/oauth.md")` called

#### TC-CLI-RELDOC-002: Link Document to Feature — Calls FeatureService.LinkDocument

**Priority**: High
**Test type**: Unit

**Steps**:
1. Execute with `--feature=E15-F10`
2. Verify `GetFeatureService().LinkDocument` called

#### TC-CLI-RELDOC-003: Link Document to Task — Calls TaskService method

**Priority**: High
**Test type**: Unit

#### TC-CLI-RELDOC-004: Unlink Document from Epic — Calls EpicService.UnlinkDocument

**Priority**: Medium
**Test type**: Unit

#### TC-CLI-RELDOC-005: List Documents for Entity

**Priority**: Medium
**Test type**: Unit

---

### 4.5 Other Refactored Commands

For the remaining refactored commands, write the following test cases as a minimum. Full test files should be created for each:

**search.go** (`internal/cli/commands/search_test.go`):
- TC-CLI-SEARCH-001: Search with query — calls service search method
- TC-CLI-SEARCH-002: JSON output format

**task_sessions.go** (`internal/cli/commands/task_sessions_test.go` or update `task_work_session_test.go`):
- Verify existing `task_work_session_test.go` still passes after refactoring
- TC-CLI-SESSION-001: StartSession — calls TaskService.StartSession
- TC-CLI-SESSION-002: EndSession — calls TaskService.EndSession

**task_unlink.go** (`internal/cli/commands/task_unlink_test.go`):
- TC-CLI-UNLINK-001: Calls TaskService.UnlinkFile with correct key
- TC-CLI-UNLINK-002: Task not found — error displayed

**status.go** (`internal/cli/commands/status_test.go`):
- Existing `status_test.go` exists — verify tests still pass after refactoring
- TC-CLI-STATUS-001: Epic key routes to EpicService
- TC-CLI-STATUS-002: Feature key routes to FeatureService
- TC-CLI-STATUS-003: Task key routes to TaskService

---

## 5. Architecture Compliance Tests

### 5.1 Fat-Controller Elimination Check

This is the primary acceptance criterion for T-E15-F10-001. It is verified by running the following grep assertion after all refactoring is complete:

```bash
# Must return zero matches in production code
grep -rn "repository\.New\|cli\.GetDB" internal/cli/commands/ | grep -v "_test.go"
```

**Gate**: This command must produce no output. Any match is a failing test.

Run this check:
1. After each individual file refactoring (partial validation)
2. After all files are refactored (final validation)
3. As part of the CI pipeline via `make lint` or a dedicated make target

### 5.2 Import Compliance Check

After refactoring, no production CLI command file should import the repository package:

```bash
# Must return zero matches
grep -rn "\"github.com/jwwelbor/shark-task-manager/internal/repository\"" internal/cli/commands/ | grep -v "_test.go"
```

### 5.3 Partial Violation Check (T-E15-F10-002, T-E15-F10-003)

Verify the partial violations in helper files are resolved:

```bash
# epic_helpers.go — must return zero matches
grep -n "repository\.New\|cli\.GetDB\|\*repository\.EpicRepository\|\*repository\.FeatureRepository" internal/cli/commands/epic_helpers.go

# feature_helpers.go — must return zero matches
grep -n "repository\.New\|cli\.GetDB\|\*repository\.FeatureRepository\|\*repository\.EpicRepository" internal/cli/commands/feature_helpers.go
```

---

## 6. Regression Tests

### 6.1 Full Test Suite

After each task is complete, run the full test suite:

```bash
make fmt && make lint && make test
```

All three commands must exit with code 0. No exceptions.

**Critical**: Run this after each individual file refactoring, not only at the end. Regressions caught early are cheaper to fix.

### 6.2 Existing Test Preservation

The following existing test files test behavior that must not change after refactoring. Verify these continue to pass:

| Test File | Covers |
|-----------|--------|
| `internal/cli/commands/idea_test.go` | Idea key generation (tests must be updated to use service mock) |
| `internal/cli/commands/idea_convert_test.go` | Idea conversion behavior |
| `internal/cli/commands/related_docs_test.go` | Document linking behavior (tests must be updated to use service mock) |
| `internal/cli/commands/task_work_session_test.go` | Session command behavior |
| `internal/cli/commands/task_deps_tree_test.go` | Dependency tree display |
| `internal/cli/commands/status_test.go` | Status dispatch behavior |
| `internal/cli/commands/status_priority_test.go` | Status priority behavior |
| `internal/services/epic_service_test.go` | EpicService behavior |
| `internal/services/feature_service_test.go` | FeatureService behavior |
| `internal/services/task_service_test.go` | TaskService behavior |

**Note on test file migration**: Existing CLI tests that currently use mocked repositories (e.g., `NewMockDocumentRepository()`) will need to be updated to use mocked services after the CLI commands are refactored. This is expected and is part of the test work for T-E15-F10-001.

### 6.3 Service Test Coverage Baseline

Before starting implementation, record the baseline coverage:

```bash
go test -cover ./internal/services/ 2>&1 | tail -5
```

After implementation, rerun and confirm coverage improved (never decreased) for existing service files, and new service files meet the 80% target.

---

## 7. Smoke Tests (Manual)

After all automated tests pass, run the following manual smoke tests to verify end-to-end behavior is unchanged.

### 7.1 Idea Commands

```bash
# List ideas
./bin/shark idea list

# Create an idea
./bin/shark idea create "Test idea from E15-F10 smoke test"

# Get the idea (use the key from previous output)
./bin/shark idea get I-2026-02-18-01

# JSON output
./bin/shark idea list --json | jq '.[0].key'

# Convert idea (if applicable)
./bin/shark idea convert I-2026-02-18-01 --type=task --key=E15-F10-001
```

### 7.2 Task Dependency Commands

```bash
# List dependencies for a task
./bin/shark task dep list E15-F10-001

# JSON output
./bin/shark task dep list E15-F10-001 --json
```

### 7.3 Related Documents Commands

```bash
# List related docs for an epic
./bin/shark related-docs list --epic=E15

# List related docs for a feature
./bin/shark related-docs list --feature=E15-F10
```

### 7.4 Status Command

```bash
# Epic status (routes through EpicService)
./bin/shark status E15

# Feature status (routes through FeatureService)
./bin/shark status E15-F10

# Task status (routes through TaskService)
./bin/shark status E15-F10-001
```

### 7.5 Regression Smoke Tests

Run the most commonly used commands to verify no regressions:

```bash
./bin/shark epic list
./bin/shark epic get E15
./bin/shark feature list E15
./bin/shark feature get E15-F10
./bin/shark task list E15
./bin/shark task list E15 F10
./bin/shark task get E15-F10-001
```

---

## 8. Quality Gates

A task may only be marked complete when ALL of the following are true:

### 8.1 Per-Task Gates

| Gate | T-E15-F10-004 | T-E15-F10-001 | T-E15-F10-002 | T-E15-F10-003 |
|------|:---:|:---:|:---:|:---:|
| `make fmt && make lint && make test` passes | Required | Required | Required | Required |
| New service files have >= 80% coverage | Required | N/A | Required | Required |
| New service tests use mocked repositories only | Required | N/A | Required | Required |
| `grep` fat-controller check returns empty | N/A | Required | Required | Required |
| No `repository.New*` imports in production CLI files | N/A | Required | N/A | N/A |
| Refactored command handler functions <= 30 lines each | N/A | Required | N/A | N/A |
| Smoke tests pass manually | Required | Required | Required | Required |

### 8.2 Feature-Level Gate

Before E15-F10 can be marked complete:

1. `grep -rn "repository\.New\|cli\.GetDB" internal/cli/commands/ | grep -v "_test.go"` returns empty
2. `make fmt && make lint && make test` passes with zero failures
3. `go test -cover ./internal/services/ | grep idea_service` shows >= 80% coverage
4. All 7 fat-controller files confirmed refactored (line counts in target range per architecture doc Section 7)
5. EpicService and FeatureService test coverage at minimum 70% (T-E15-F10-002 and T-E15-F10-003)

---

## 9. Test File Inventory

### 9.1 New Test Files to Create

| File | Task | Priority |
|------|------|----------|
| `internal/services/idea_service_test.go` | T-E15-F10-004 | Must-Have |
| `internal/cli/commands/task_deps_test.go` | T-E15-F10-001 | Should-Have |
| `internal/cli/commands/task_unlink_test.go` | T-E15-F10-001 | Should-Have |
| `internal/cli/commands/search_test.go` | T-E15-F10-001 | Should-Have |

### 9.2 Existing Test Files to Update

| File | What Changes | Task |
|------|-------------|------|
| `internal/services/task_service_test.go` | Add mock methods for dependency/unlink; add new test cases | T-E15-F10-004 |
| `internal/services/epic_service_test.go` | Add tests for LinkDocument, UnlinkDocument, CompleteEpic variants, GetImpediments | T-E15-F10-002 / T-E15-F10-004 |
| `internal/services/feature_service_test.go` | Add tests for LinkDocument, UnlinkDocument | T-E15-F10-003 / T-E15-F10-004 |
| `internal/cli/commands/idea_test.go` | Replace repository mock with service mock; update test cases for refactored commands | T-E15-F10-001 |
| `internal/cli/commands/related_docs_test.go` | Replace repository mocks with service mocks | T-E15-F10-001 |
| `internal/cli/commands/task_work_session_test.go` | Verify still passes; update if session methods move to service layer | T-E15-F10-001 |

### 9.3 Test Files to Verify (No Changes Expected)

These files should continue to pass unchanged — run them as regression verification:

- `internal/cli/commands/idea_convert_test.go`
- `internal/cli/commands/task_deps_tree_test.go`
- `internal/cli/commands/status_test.go`
- `internal/cli/commands/status_priority_test.go`
- All `epic_*_test.go`, `feature_*_test.go`, `task_*_test.go` files

---

## 10. Risk Areas and Mitigations

| Risk | Likelihood | Impact | Test Mitigation |
|------|-----------|--------|-----------------|
| `idea_test.go` tightly coupled to old `IdeaRepository` interface in `idea.go` | High | Medium | Must update tests when interface moves to service layer |
| `related_docs_test.go` uses `createRelatedDocsAddCmd()` factory with repo injection — may break after refactor | High | Medium | Update factory pattern to accept service mocks; see if factory can be replaced |
| Session management model (`models.Session`) may not exist | Medium | High | Check before writing session tests; skip or stub if model is absent |
| Circular dependency detection in `AddDependency` is complex — may be deferred | Medium | Low | Test what exists; note any deferred functionality |
| Helpers in `epic_helpers.go` accepting `*repository.EpicRepository` — may not compile after refactor | Medium | High | Run `make build` after each change, not just `make test` |
| `idea.go` has a locally-defined `IdeaRepository` interface — may conflict with service layer interface | Medium | Medium | Explicitly test that the interface moved correctly; compilation catches this |

---

## 11. Test Execution Order

Recommended execution sequence to detect issues early:

1. **Before any implementation**: Run `make test` to establish a green baseline
2. **After T-E15-F10-004** (IdeaService + service extensions):
   - Run new `idea_service_test.go`
   - Run extended `task_service_test.go`, `epic_service_test.go`, `feature_service_test.go`
   - Run full `make test`
3. **After each file refactored in T-E15-F10-001** (one file at a time):
   - Run `make test` immediately
   - Run fat-controller grep check on refactored file
4. **After T-E15-F10-002** (EpicService integration verification):
   - Run epic_helpers.go partial violation grep check
   - Run `make test`
5. **After T-E15-F10-003** (FeatureService integration verification):
   - Run feature_helpers.go partial violation grep check
   - Run `make test`
6. **Final validation** (all tasks complete):
   - Run full architecture compliance grep checks (Section 5.1, 5.2, 5.3)
   - Run `make fmt && make lint && make test`
   - Run manual smoke tests (Section 7)
   - Run coverage report: `make test-coverage`

---

*Test Plan Version*: 1.0
*Last Updated*: 2026-02-18
