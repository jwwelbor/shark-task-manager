# UAT Test Guide - TaskService Implementation

**Feature:** E15-F08 - TaskService Implementation
**Epic:** E15 - Service Layer Architecture Refactoring
**Generated:** 2026-02-17
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Transform Shark's architecture from a fat-controller pattern (business logic in CLI commands) to a clean three-layer architecture with dedicated service layer.

**Current Problem:**
- CLI commands contain 40-45% business logic across 43,590 lines of code
- Monolithic 2,000+ line command files
- Business rules duplicated across CLI and future HTTP API
- Difficult to test in isolation (requires real database)

**Target Architecture:**
- **Layer 1 (Entry Points)**: CLI commands and HTTP handlers - thin wrappers (parse → call → format)
- **Layer 2 (Service Layer)**: TaskService, FeatureService, EpicService - all business logic
- **Layer 3 (Repository Layer)**: Pure data access, no business rules

**This Feature's Role:**

E15-F08 implements the TaskService with five core CRUD operations (CreateTask, GetTask, UpdateTask, DeleteTask, ListTasks) that form the foundation for task management in the service layer. This feature:

1. **Builds on E15-F01 Foundation**: Implements the TaskService interface, dependency injection patterns, and constructor design established in F01
2. **Enables Future Refactoring**: Provides service methods that CLI commands (F07) will call instead of direct repository access
3. **Establishes Testing Pattern**: Demonstrates mock-based service testing without real database
4. **Validates Architecture**: Proves the service layer pattern works for real CRUD operations

**Related Features:**

- **E15-F01 (Completed)**: Service Interface Design and Foundation
  - Status: Completed (100%)
  - Dependency: F08 implements interfaces designed in F01
  - Integration: F08 uses TaskRepository interface, DI patterns, and constructor design from F01

- **E15-F02 (Cancelled)**: TaskService CRUD Operations (merged into F08)
- **E15-F03 (Cancelled)**: TaskService Lifecycle Operations (merged into F08)
- **E15-F04 (Draft)**: TaskService Querying and Dependencies - future extension
- **E15-F05 (Draft)**: Epic and Feature Service Expansion - parallel service implementation
- **E15-F06 (Draft)**: Repository Layer Cleanup - depends on F08 service layer
- **E15-F07 (Draft)**: CLI Commands as Thin Wrappers - will call F08 service methods

**Integration Points:**

1. **With E15-F01 (Foundation)**:
   - Uses TaskRepository interface defined in F01
   - Follows constructor injection pattern from F01
   - Implements service design principles documented in F01
   - Adheres to godoc standards from F01

2. **For E15-F07 (CLI Refactoring)**:
   - Provides TaskService.CreateTask() for CLI create command
   - Provides TaskService.GetTask() for CLI get command
   - Provides TaskService.ListTasks() for CLI list command with filters
   - Provides TaskService.UpdateTask() for CLI update command
   - Provides TaskService.DeleteTask() for CLI delete command

3. **No Direct UI Integration**: Backend service layer, no frontend components
4. **No Data Sharing**: Service layer orchestrates repository calls, no shared state
5. **Workflow Handoffs**: None - standalone service implementation

---

## Design Intent

**From Epic PRD (docs/plan/E15-service-layer-architecture-refactoring/epic.md):**

> **FR1 - Service Layer**: Introduce service/application layer (internal/services/) with TaskService, FeatureService, EpicService that owns business logic, orchestration, validation.

> **Problem Statement**: Shark's CLI commands contain significant business logic (40-45% of code), making them fat controllers. With 43,590 lines across commands averaging 2,000+ lines each, business rules are scattered, duplicated between CLI and potential HTTP API, and difficult to test in isolation.

> **Success Metrics**:
> - Reduce average CLI command file from 2,000+ lines to 200-400 lines
> - Service layer contains 80%+ of business logic
> - 100% service methods tested with mocked repositories (no real DB)

**From Task Specification (T-E15-F08-001.md):**

> **Objective**: Implement the five core CRUD methods in TaskService (CreateTask, GetTask, UpdateTask, DeleteTask, ListTasks) with comprehensive test coverage using mocked repositories.

**Key Design Decisions to Validate:**

1. **Interface-Based Dependency Injection**: Service depends on TaskRepository interface (not concrete *repository.TaskRepository), enabling mock-based testing
2. **Business-Level Inputs**: Methods accept task keys (not database IDs), priority ranges (not raw integers)
3. **Error Wrapping**: All errors wrapped with business context at service layer
4. **Workflow Validation**: Status transitions validated via workflow.Service
5. **Graceful Degradation**: Optional dependencies (creatorSvc, noteRepo) handled with nil checks
6. **Mock-Based Testing**: Zero database dependency in service tests - all use MockTaskRepository
7. **DTOs for Complex Inputs**: CreateTaskInput, TaskUpdates, TaskFilters to avoid parameter explosion

---

## Cross-Feature Integration Tests

### Integration Scenario 1: Service Interface Compliance with F01 Foundation
**Features:** E15-F01 (Foundation) + E15-F08 (Implementation)
**Scenario:** Verify TaskService implements the interface designed in F01 and follows DI patterns

**Steps:**
1. Read TaskService constructor in internal/services/task_service.go
2. Verify constructor signature matches F01 DI pattern: `NewTaskService(repo TaskRepository, workflowSvc *workflow.Service, ...)`
3. Read TaskRepository interface definition
4. Verify all five CRUD methods exist: CreateTask, GetTask, UpdateTask, DeleteTask, ListTasks
5. Check that service depends on interface (TaskRepository) not concrete type (*repository.TaskRepository)

**Expected Result:**
- ✅ Constructor uses dependency injection (no globals, no direct DB access)
- ✅ TaskRepository interface defined with all required methods
- ✅ Service struct holds interface reference: `repo TaskRepository`
- ✅ All five CRUD methods implemented and callable
- ✅ Service can be instantiated with mock repositories (proven by tests)

**Success Criteria:**
- [ ] Constructor follows DI pattern from F01
- [ ] Service depends on TaskRepository interface (not concrete type)
- [ ] All five CRUD methods are present and functional
- [ ] Tests use MockTaskRepository successfully (no real DB)

---

### Integration Scenario 2: Workflow Service Integration
**Features:** E15-F08 (TaskService) + Workflow Package
**Scenario:** Verify TaskService validates status transitions via workflow.Service

**Steps:**
1. Review CreateTask method - should use workflow.GetDefaultStatus()
2. Check that service validates status via workflow.ValidateStatus()
3. Verify error messages include business context when workflow validation fails

**Expected Result:**
- ✅ CreateTask uses workflow service to get default status (not hardcoded "todo")
- ✅ Service validates status values against workflow configuration
- ✅ Invalid status transitions produce clear error messages

**Success Criteria:**
- [ ] Default status comes from workflow service (not hardcoded)
- [ ] Status validation uses workflow.ValidateStatus()
- [ ] Tests verify workflow integration (mock workflow service)

---

## Epic Acceptance Validation

| Epic AC | Description | Feature Contribution | Status |
|---------|-------------|---------------------|--------|
| **FR1** | Service layer owns business logic | F08 implements TaskService with all business logic for task CRUD operations | [ ] |
| **FR9** | Service tests use mocked repositories | F08 has 52 tests, all using MockTaskRepository (0 real DB tests) | [ ] |
| **Success Metric** | Service layer contains 80%+ business logic | F08 service layer has ~350 lines of business logic vs ~15 lines per CLI command wrapper | [ ] |
| **Success Metric** | 100% service tests use mocks | F08: 52/52 tests use mocked repositories | [ ] |

---

## Feature Acceptance Validation

| Feature AC | Description | Status |
|------------|-------------|--------|
| **AC-001** | All five CRUD methods implemented and functional | [ ] |
| **AC-002** | Each method handles context properly (cancellation, timeout) | [ ] |
| **AC-003** | Error messages include task keys for debugging | [ ] |
| **AC-004** | CreateTask validates inputs (empty title, out-of-range priority, missing epic/feature) | [ ] |
| **AC-005** | UpdateTask applies only non-nil updates via TaskUpdates pointer fields | [ ] |
| **AC-006** | ListTasks filters by epic, feature, status, agent_type; excludes completed by default | [ ] |
| **AC-007** | ListTasks sorts by execution_order ascending, then priority descending | [ ] |
| **AC-008** | All methods wrap errors with business context | [ ] |
| **AC-009** | No real database used in tests (mocks only) | [ ] |
| **AC-010** | Service tests have >80% code coverage | [ ] |
| **AC-011** | `make fmt` produces no changes | [ ] |
| **AC-012** | `make lint` produces no violations | [ ] |
| **AC-013** | `make test` passes with all tests green | [ ] |
| **AC-014** | No new external dependencies added | [ ] |
| **AC-015** | Methods work with both mocked and real repositories (interface-based) | [ ] |

---

## Test Scenarios

### Scenario 1: CreateTask - Happy Path
**Tasks covered:** T-E15-F08-001
**Test File:** internal/services/task_service_test.go (TestTaskService_CreateTask_Happy_Path)

**Preconditions:**
- Service instantiated with MockTaskRepository
- Epic E07 exists in mock data
- Feature E07-F01 exists in mock data

**Steps:**
1. Call `svc.CreateTask(ctx, CreateTaskInput{EpicKey: "E07", FeatureKey: "F01", Title: "Test Task", Priority: 8})`
2. Verify mock repository Create method was called
3. Check returned task has generated key (e.g., "T-E07-F01-001")
4. Verify task title, priority, status are correct

**Expected Results:**
- Task key generated in format "T-{epic}-{feature}-{number}"
- Task status set to workflow default (e.g., "todo")
- Priority set to 8
- Epic/Feature IDs populated
- No errors returned

**Success Criteria:**
- [ ] Task created successfully with valid key
- [ ] Default status from workflow service applied
- [ ] Priority and title match input
- [ ] Mock repository.Create() called exactly once
- [ ] No real database interaction

**Test Evidence:** Run `go test ./internal/services/ -run TestTaskService_CreateTask_Happy_Path -v`

---

### Scenario 2: CreateTask - Input Validation
**Tasks covered:** T-E15-F08-001
**Test File:** internal/services/task_service_test.go (multiple validation tests)

**Preconditions:**
- Service instantiated with MockTaskRepository

**Test Cases:**

| Test Case | Input | Expected Error |
|-----------|-------|----------------|
| Empty title | `Title: ""` | "title is required" |
| Empty epic key | `EpicKey: ""` | "epic key is required" |
| Empty feature key | `FeatureKey: ""` | "feature key is required" |
| Invalid priority (too high) | `Priority: 15` | "priority must be between 1 and 10" |
| Invalid priority (too low) | `Priority: 0` | Default priority (5) assigned |

**Steps:**
1. For each test case, call CreateTask with invalid input
2. Verify appropriate error returned
3. Verify no repository calls made for validation failures

**Expected Results:**
- Empty fields rejected with clear error messages
- Out-of-range priority rejected
- Priority=0 treated as "use default" (5)
- Validation happens before repository calls

**Success Criteria:**
- [ ] Empty title rejected
- [ ] Empty epic/feature keys rejected
- [ ] Invalid priority rejected
- [ ] Default priority assigned when priority=0
- [ ] No repository calls for validation failures

**Test Evidence:** Run `go test ./internal/services/ -run TestTaskService_CreateTask -v`

---

### Scenario 3: GetTask - Retrieve Existing Task
**Tasks covered:** T-E15-F08-001
**Test File:** internal/services/task_service_test.go (TestTaskService_GetTask_Found)

**Preconditions:**
- MockTaskRepository configured to return task for key "E07-F01-001"

**Steps:**
1. Call `svc.GetTask(ctx, "E07-F01-001")`
2. Verify mock repository GetByKey method called with correct key
3. Check returned task matches mock data

**Expected Results:**
- Task returned with correct key, title, status
- Mock repository.GetByKey() called once with "E07-F01-001"
- No errors

**Success Criteria:**
- [ ] Task retrieved successfully
- [ ] Correct repository method called
- [ ] Case-insensitive key lookup works
- [ ] Error wrapped with business context if not found

**Test Evidence:** Run `go test ./internal/services/ -run TestTaskService_GetTask -v`

---

### Scenario 4: UpdateTask - Partial Update
**Tasks covered:** T-E15-F08-001
**Test File:** internal/services/task_service_test.go (TestTaskService_UpdateTask_Partial_Update)

**Preconditions:**
- Existing task with key "E07-F01-001", title "Old Title", priority 5

**Steps:**
1. Create TaskUpdates with only title: `updates := TaskUpdates{Title: ptr("New Title")}`
2. Call `svc.UpdateTask(ctx, "E07-F01-001", updates)`
3. Verify only title updated, priority unchanged

**Expected Results:**
- Task title changed to "New Title"
- Priority remains 5
- Other fields unchanged
- Only non-nil fields in TaskUpdates applied

**Success Criteria:**
- [ ] Only title updated (partial update works)
- [ ] Priority unchanged (nil field ignored)
- [ ] Mock repository Update() called with correct task
- [ ] No validation errors for partial updates

**Test Evidence:** Run `go test ./internal/services/ -run TestTaskService_UpdateTask_Partial -v`

---

### Scenario 5: DeleteTask - Prevent Delete with Dependents
**Tasks covered:** T-E15-F08-001
**Test File:** internal/services/task_service_test.go (TestTaskService_DeleteTask_Has_Dependents)

**Preconditions:**
- Task "E07-F01-001" exists
- Task "E07-F01-002" depends on "E07-F01-001"

**Steps:**
1. Call `svc.DeleteTask(ctx, "E07-F01-001")`
2. Verify error returned indicating dependents exist
3. Confirm repository Delete() NOT called

**Expected Results:**
- Error returned: "cannot delete task with dependents"
- Repository.Delete() not called
- Task remains in database (mock)

**Success Criteria:**
- [ ] Delete blocked when dependents exist
- [ ] Clear error message returned
- [ ] Repository Delete() not called
- [ ] Dependency check happens before delete attempt

**Test Evidence:** Run `go test ./internal/services/ -run TestTaskService_DeleteTask_Has_Dependents -v`

---

### Scenario 6: ListTasks - Filtering and Sorting
**Tasks covered:** T-E15-F08-001
**Test File:** internal/services/task_service_test.go (TestTaskService_ListTasks_*)

**Preconditions:**
- Mock repository returns tasks:
  - T-E07-F01-001: status=todo, agent=backend, order=1, priority=8
  - T-E07-F01-002: status=in_progress, agent=frontend, order=1, priority=5
  - T-E07-F01-003: status=completed, agent=backend, order=2, priority=3
  - T-E07-F01-004: status=todo, agent=qa, order=null, priority=9

**Test Cases:**

| Test Case | Filters | Expected Result |
|-----------|---------|-----------------|
| No filters (default) | `{}` | T-E07-F01-001, T-E07-F01-002, T-E07-F01-004 (completed excluded) |
| Show all | `{ShowAll: true}` | All 4 tasks |
| Filter by status | `{Status: "todo"}` | T-E07-F01-001, T-E07-F01-004 |
| Filter by agent | `{AgentType: "backend"}` | T-E07-F01-001, T-E07-F01-003 |
| Combined filters | `{Status: "todo", AgentType: "backend"}` | T-E07-F01-001 only |

**Sorting Verification:**
- Expected order: [order1-prio8, order1-prio5, order2-prio3, null-prio9]
- Tasks with same execution_order sorted by priority descending

**Steps:**
1. For each test case, call ListTasks with filters
2. Verify correct tasks returned
3. Verify sort order: execution_order ascending, then priority descending
4. Confirm completed tasks excluded by default

**Expected Results:**
- Filters applied correctly
- Sorting matches specification (order ASC, priority DESC)
- ShowAll=false excludes completed tasks
- ShowAll=true includes all tasks

**Success Criteria:**
- [ ] Status filter works
- [ ] AgentType filter works
- [ ] Combined filters work (AND logic)
- [ ] Completed tasks excluded by default
- [ ] ShowAll flag includes completed tasks
- [ ] Sorting by execution_order ascending
- [ ] Secondary sort by priority descending

**Test Evidence:** Run `go test ./internal/services/ -run TestTaskService_ListTasks -v`

---

### Scenario 7: Error Wrapping - Business Context
**Tasks covered:** T-E15-F08-001
**Acceptance Criteria:** AC-008

**Preconditions:**
- MockTaskRepository configured to return NotFoundError for key "E07-F01-999"

**Steps:**
1. Call `svc.GetTask(ctx, "E07-F01-999")`
2. Verify error returned
3. Check error message contains task key
4. Verify error is wrapped (can use errors.As to unwrap)

**Expected Error Message Format:**
```
failed to get task E07-F01-999: task not found: E07-F01-999
```

**Expected Results:**
- Error includes business context: "failed to get task {key}"
- Error includes technical context: repository error message
- Error chain preserves original error (errors.As works)
- Task key included in error message for debugging

**Success Criteria:**
- [ ] Error message includes task key
- [ ] Error message includes operation context ("failed to get task")
- [ ] Error wrapping preserves original error
- [ ] All CRUD methods wrap errors consistently

**Test Evidence:** Run `go test ./internal/services/ -run TestTaskService -v` and verify error messages in test output

---

### Scenario 8: Mock-Based Testing - No Real Database
**Tasks covered:** T-E15-F08-001
**Acceptance Criteria:** AC-009, AC-010

**Preconditions:**
- None (this is a meta-test of the test suite itself)

**Steps:**
1. Read test file: `internal/services/task_service_test.go`
2. Verify MockTaskRepository definition exists
3. Verify all tests use MockTaskRepository
4. Confirm no imports of `internal/test` (real DB helper)
5. Run all service tests: `go test ./internal/services/ -v`

**Expected Results:**
- All tests use MockTaskRepository with function fields
- No real database connections in test code
- Tests run fast (<1 second for 52 tests)
- Tests are deterministic (same result every run)

**Success Criteria:**
- [ ] MockTaskRepository defined with function fields
- [ ] All 52 tests use mocks (grep for "MockTaskRepository")
- [ ] No imports of internal/test package
- [ ] No calls to test.GetTestDB()
- [ ] Tests run in <1 second (cached results indicate fast execution)
- [ ] Test coverage >80% (AC-010)

**Test Evidence:**
```bash
go test ./internal/services/ -cover
grep -r "test.GetTestDB" internal/services/*_test.go  # Should return nothing
grep -r "MockTaskRepository" internal/services/*_test.go  # Should show all tests
```

---

### Scenario 9: Quality Gates - Formatting and Linting
**Tasks covered:** T-E15-F08-001
**Acceptance Criteria:** AC-011, AC-012, AC-013

**Preconditions:**
- Clean working directory

**Steps:**
1. Run `make fmt` - should produce no changes
2. Run `make lint` - should produce 0 issues
3. Run `make test` - should pass with all tests green

**Expected Results:**
- `make fmt`: No files changed
- `make lint`: Output shows "0 issues"
- `make test`: All tests pass, "ok" status for all packages

**Success Criteria:**
- [ ] make fmt produces no changes
- [ ] make lint produces 0 violations
- [ ] make test passes (all repository, service, CLI tests green)
- [ ] No warnings in test output

**Test Evidence:**
```bash
make fmt    # Should show no output
make lint   # Should show "0 issues"
make test   # Should show PASS for all packages
```

---

### Scenario 10: Interface-Based Design - Mock and Real Repo Compatibility
**Tasks covered:** T-E15-F08-001
**Acceptance Criteria:** AC-015

**Preconditions:**
- Service interface defined
- Both MockTaskRepository and *repository.TaskRepository exist

**Steps:**
1. Verify MockTaskRepository implements TaskRepository interface
2. Verify *repository.TaskRepository implements TaskRepository interface
3. Confirm service constructor accepts interface: `repo TaskRepository`
4. Test that service works with mock (unit tests)
5. Verify service would work with real repository (no type mismatch)

**Expected Results:**
- Both mock and real repository implement same interface
- Service depends on interface, not concrete type
- No type assertions or type switches in service code
- Tests prove service works with mock
- Architecture allows swapping to real repository

**Success Criteria:**
- [ ] TaskRepository interface defined
- [ ] MockTaskRepository implements interface (all methods present)
- [ ] repository.TaskRepository implements interface
- [ ] Service constructor signature: `NewTaskService(repo TaskRepository, ...)`
- [ ] No type assertions in service code
- [ ] Tests use mock successfully

**Test Evidence:**
```bash
# Verify interface definition
grep "type TaskRepository interface" internal/services/task_service.go

# Verify mock implements interface
go build ./internal/services/  # Should compile without errors

# Verify real repository implements interface
go build ./internal/repository/  # Should compile without errors
```

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-02-17 17:06:15 |
| Result | ✅ PASSED (12/12 scenarios) |
| Results File | [UAT-E15-F08-20260217-170615-results.md](results/UAT-E15-F08-20260217-170615-results.md) |

**Previous Sessions:** 1 session

---

## Notes for UAT Execution

**Show/Tell Pattern for Backend Services:**

Since TaskService is a backend service layer with no UI, UAT validation focuses on:
1. **Spec Intent**: Quote exact PRD/AC language being verified
2. **Implementation Code**: Show service method implementation
3. **Test Code**: Show test function with setup, execution, assertions
4. **Use Case**: How is this method used? (CLI commands will call these methods)
5. **Run Verification**: Execute tests and show results
6. **Compare to Spec**: Map test results to acceptance criteria
7. **User Verdict**: Get PASS/FAIL based on complete evidence

**Test Execution Commands:**
```bash
# Run all service tests
go test ./internal/services/ -v

# Run specific method tests
go test ./internal/services/ -run TestTaskService_CreateTask -v
go test ./internal/services/ -run TestTaskService_GetTask -v
go test ./internal/services/ -run TestTaskService_UpdateTask -v
go test ./internal/services/ -run TestTaskService_DeleteTask -v
go test ./internal/services/ -run TestTaskService_ListTasks -v

# Run quality gates
make fmt
make lint
make test

# Check test coverage
go test ./internal/services/ -cover
```

**Key Validation Points:**
- All 15 acceptance criteria from task spec
- Integration with E15-F01 foundation (interface compliance)
- Mock-based testing (no real database)
- Error wrapping with business context
- Partial updates via TaskUpdates DTOs
- Filtering and sorting logic in ListTasks
- Quality gates (fmt, lint, test)

**UAT Conductor:** Claude (QA Agent)
**Generated:** 2026-02-17
