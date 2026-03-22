# E21-F09: Entity Service Delegation Completion -- Test Plan

**Feature**: E21-F09
**Tier**: STANDARD
**Author**: QA Agent
**Date**: 2026-03-21

This feature completes the delegation story by making BugService, ChangeCardService, and TaskService fully delegate status transitions and next-status logic to EntityService. The testing strategy focuses on **delegation correctness** (each service calls EntityService with proper arguments), **behavioral equivalence** (existing tests pass unchanged except constructor setup), and **post-hook preservation** (task auto-unblock still works).

---

## 1. Epic UAT Scenario Decomposition

### Mapping F09 to Epic UAT Scenarios

F09 completes the service delegation layer, directly enabling the "single point of fix" and "consistency" promises validated in epic-level UAT.

| Epic UAT Scenario | F09 Contribution | F09 Validation Method |
|---|---|---|
| **Scenario 2: Fix Cross-Cutting Bug** (Journey 2) | After F09, ALL 5 entity services delegate transition logic to EntityService.TransitionStatus. A bug in transition validation is fixed in 1 file, not 5. | Verify: BugService and ChangeCardService call `entitySvc.TransitionStatus` (not inline workflow validation). TaskService delegates full flow instead of calling helper methods individually. |
| **Scenario 3: Add Cross-Cutting Feature** (Journey 3) | Adding a new transition hook (e.g., audit logging) to EntityService.TransitionStatus applies to all 5 entity types automatically. | Verify: All 5 entity services pass through EntityService.TransitionStatus with appropriate TransitionFeatures. |
| **UAT Metric 4: Test Pass Rate** | F09 must achieve 100% existing test pass rate after constructor signature changes. | `make fmt && make lint && make test` after each implementation step. |
| **P0: Zero Behavioral Regression** | `shark status advance B001`, `shark status advance CC-001`, `shark status advance E01-F01-001` must produce identical results before and after. | All existing service tests pass (with updated constructor setup). |

---

## 2. AC Test Matrix

### AC Group 1: REQ-F-001 -- BugService EntityService Delegation

#### AC-1.1: BugService struct has entitySvc and entityRepo fields

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-001 | BugService constructor accepts EntityService and EntityRepository | Call `NewBugService(repo, entitySvc, entityRepo, epicRepo, featureRepo, taskRepo, projectRoot)` | No panic; service created successfully | `entitySvc` is nil -- should panic or be documented as required |
| TC-F09-002 | Compile-time verification of new constructor signature | Compile test file that uses new constructor | Compiles without error | Old constructor signature (without entitySvc/entityRepo) should fail to compile |

**Test pattern**: Constructor tests in `bug_service_test.go`. Follow existing `mockBugRepo` pattern.

#### AC-1.2: SetBugStatus delegates to entitySvc.TransitionStatus

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-003 | SetBugStatus calls EntityService.TransitionStatus with correct parameters | Bug in status "draft", target "ready_for_development" | `entitySvc.TransitionStatus` called with: `entityRepo`, `EntityTypeBug`, bug key, target status, `SimpleTransitionFeatures()`, and `makeResolveActionFn` callback | |
| TC-F09-004 | SetBugStatus with force=true passes Force option | Bug "draft" -> "completed" with force=true | `TransitionOptions{Force: true}` passed to EntityService | force=true without reason should return `ErrForceReasonRequired` (EntityService handles) |
| TC-F09-005 | SetBugStatus re-fetches bug after transition | Successful transition | Returns `*models.Bug` from `repo.GetByKey` (not from EntityService result) | Repo.GetByKey returns error after transition -- error propagated |
| TC-F09-006 | SetBugStatus propagates EntityService errors | EntityService returns validation error | Error returned to caller unchanged | |
| TC-F09-007 | SetBugStatus handles entity not found | Bug key does not exist in entityRepo | Error from EntityService propagated | |

**Test pattern**: Mock `entitySvc.TransitionStatus` via a mock `EntityService` or by using a mock `EntityRepository` + real EntityService with mock workflow. The existing pattern uses mock repos + real workflow service constructed from temp config files (see `entity_service_test.go`).

#### AC-1.3: AdvanceBugStatus uses entitySvc.GetNextStatus then SetBugStatus

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-008 | AdvanceBugStatus calls GetNextStatus for current status | Bug in "draft" status | `entitySvc.GetNextStatus` called with bug's entity repo, EntityTypeBug, bug key | |
| TC-F09-009 | AdvanceBugStatus uses first available transition | GetNextStatus returns transitions: ["ready_for_development", "cancelled"] | SetBugStatus called with "ready_for_development" | |
| TC-F09-010 | AdvanceBugStatus with no available transitions | Bug in terminal status "completed" | Error: "cannot advance bug ... no valid transitions" | |
| TC-F09-011 | AdvanceBugStatus returns typed Bug model | Successful advance | Returns `*models.Bug` (not generic Entity) | |

#### AC-1.4: resolveAction replaced by makeResolveActionFn

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-012 | makeResolveActionFn returns callback that produces PopulatedAction | Bug entity, target status with configured action | Callback returns `*config.PopulatedAction` with bug placeholders | |
| TC-F09-013 | makeResolveActionFn callback handles non-Bug entity gracefully | Non-bug entity passed to callback | Returns nil (no panic) | |
| TC-F09-014 | makeResolveActionFn callback returns nil for unconfigured status | Status with no action configured | Returns nil | |

**Test pattern**: Follow `epic_service_test.go` `TestEpicService_makeResolveActionFn_*` pattern.

#### AC-1.5: History recording handled by EntityService

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-015 | BugService does not call recordBugHistory directly | SetBugStatus transitions bug | No direct history repo call from BugService; EntityService records history | Verify by absence of `recordBugHistory` method |
| TC-F09-016 | SetEntityHistoryRepo removed from BugService | Attempt to call `SetEntityHistoryRepo` | Compile error (method no longer exists) | |

#### AC-1.6: All existing BugService tests pass

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-017 | Full BugService test suite passes | `go test ./internal/services/ -run TestBug` | All tests pass | Constructor setup updated for new signature |

---

### AC Group 2: REQ-F-002 -- ChangeCardService EntityService Delegation

Follows identical pattern to BugService (REQ-F-001). Test IDs: TC-F09-018 through TC-F09-034.

| Test ID | Description | Mirrors |
|---|---|---|
| TC-F09-018 | ChangeCardService constructor accepts EntityService and EntityRepository | TC-F09-001 |
| TC-F09-019 | Compile-time verification of new constructor | TC-F09-002 |
| TC-F09-020 | SetChangeCardStatus delegates to EntityService.TransitionStatus | TC-F09-003 |
| TC-F09-021 | SetChangeCardStatus with force=true | TC-F09-004 |
| TC-F09-022 | SetChangeCardStatus re-fetches change card after transition | TC-F09-005 |
| TC-F09-023 | SetChangeCardStatus propagates EntityService errors | TC-F09-006 |
| TC-F09-024 | SetChangeCardStatus handles entity not found | TC-F09-007 |
| TC-F09-025 | AdvanceChangeCardStatus calls GetNextStatus | TC-F09-008 |
| TC-F09-026 | AdvanceChangeCardStatus uses first available transition | TC-F09-009 |
| TC-F09-027 | AdvanceChangeCardStatus with no available transitions | TC-F09-010 |
| TC-F09-028 | AdvanceChangeCardStatus returns typed ChangeCard model | TC-F09-011 |
| TC-F09-029 | makeResolveActionFn returns callback with change card placeholders | TC-F09-012 |
| TC-F09-030 | makeResolveActionFn callback handles non-ChangeCard entity | TC-F09-013 |
| TC-F09-031 | makeResolveActionFn callback returns nil for unconfigured status | TC-F09-014 |
| TC-F09-032 | History recording handled by EntityService | TC-F09-015 |
| TC-F09-033 | SetEntityHistoryRepo removed from ChangeCardService | TC-F09-016 |
| TC-F09-034 | Full ChangeCardService test suite passes | TC-F09-017 |

---

### AC Group 3: REQ-F-003 -- TaskService Full TransitionStatus Delegation

#### AC-3.1: TaskService.TransitionStatus calls entitySvc.TransitionStatus

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-035 | TransitionStatus delegates to entitySvc.TransitionStatus | Task "draft" -> "ready_for_refinement_ba" | EntityService.TransitionStatus called with task's EntityRepository adapter, `DefaultTransitionFeatures()`, and `makeResolveActionFn` | |
| TC-F09-036 | Custom taskEntityRepoAdapter routes UpdateStatus through StatusUpdateRaw | Task adapter's UpdateStatus called during transition | `StatusUpdateRaw` called with agent, reason, documentPath from TransitionOptions | Verify StatusUpdateRaw params match opts |
| TC-F09-037 | TransitionStatus propagates EntityService errors | EntityService returns workflow validation error | Error returned to caller | |
| TC-F09-038 | TransitionStatus with force=true and reason | Force transition | EntityService receives `TransitionOptions{Force: true, Reason: "..."}` | Force without reason returns `ErrForceReasonRequired` (handled by EntityService) |

#### AC-3.2: Auto-unblock runs as post-hook

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-039 | Auto-unblock runs after successful transition | Task blocked dependent; blocker transitions to "completed" | Dependent tasks auto-unblocked; result.Message includes unblocked keys | |
| TC-F09-040 | Auto-unblock does NOT run on failed transition | EntityService returns error | No auto-unblock attempted | |
| TC-F09-041 | Auto-unblock with no dependents | Task with no blocked dependents transitions | Empty unblocked list; message unchanged | |
| TC-F09-042 | Auto-unblock result merged into TransitionResult | 2 tasks auto-unblocked | TransitionResult.Message contains both unblocked keys | |

#### AC-3.3: ErrForceReasonRequired handled by EntityService

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-043 | TaskService does NOT check ErrForceReasonRequired inline | Force=true, Reason="" | Error comes from EntityService, not TaskService inline check | Verify by code inspection: no `ErrForceReasonRequired` in TaskService |

#### AC-3.4: History recording handled by EntityService

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-044 | TaskService does not call recordEntityHistory directly | Successful transition | EntityService handles history via its internal historyRepo | Verify by absence of explicit `recordEntityHistory` call |

#### AC-3.5: resolveAction replaced by makeResolveActionFn

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-045 | TaskService.makeResolveActionFn returns callback with task placeholders | Task entity, target status | Callback returns PopulatedAction for task | |
| TC-F09-046 | makeResolveActionFn callback handles non-Task entity | Non-task entity | Returns nil | |

#### AC-3.6: All existing TaskService transition tests pass

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-047 | Full TaskService test suite passes | `go test ./internal/services/ -run TestTask` | All tests pass | Some test setup may need updates for delegation pattern |

---

### AC Group 4: REQ-F-004 -- TaskService GetNextStatus Delegation

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-048 | GetNextStatus delegates to entitySvc.GetNextStatus | Task in "draft" status | `entitySvc.GetNextStatus` called with task EntityRepository, EntityTypeTask, task key, and makeResolveActionFn | |
| TC-F09-049 | GetNextStatus returns NextStatusInfo from EntityService | Task with valid transitions | Returns NextStatusInfo with AvailableTransitions populated | |
| TC-F09-050 | GetNextStatus for terminal status | Task in "completed" | Returns empty AvailableTransitions | |
| TC-F09-051 | GetNextStatus inline implementation removed | Code inspection | No direct `GetWorkflowService().GetTransitionInfo` call in TaskService.GetNextStatus | Verify by code review |
| TC-F09-052 | All existing GetNextStatus tests pass | `go test ./internal/services/ -run TestTaskService.*NextStatus` | All pass | |

---

### AC Group 5: REQ-F-005 -- ResumeService Per-Entity Repo Field Reduction

Per spec Decision 4, ResumeService is left as-is. Per-entity repos are justified by entity-specific query needs.

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-053 | ResumeService constructor unchanged | Existing constructor call | Compiles and works | |
| TC-F09-054 | All existing ResumeService tests pass | `go test ./internal/services/ -run TestResume` | All pass | No changes expected |

---

### AC Group 6: REQ-F-006 -- Service Accessor Wiring

#### AC-6.1: GetBugService wiring

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-055 | GetBugService passes EntityService and EntityRepository to constructor | Inspect `services_global.go` | `GetEntityService()` and `GetEntityRegistry().MustGetRepository(models.EntityTypeBug)` called | |
| TC-F09-056 | GetBugService no longer calls SetEntityHistoryRepo | Inspect `services_global.go` | No `SetEntityHistoryRepo` call | |

#### AC-6.2: GetChangeCardService wiring

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-057 | GetChangeCardService passes EntityService and EntityRepository to constructor | Inspect `services_global.go` | Same pattern as BugService | |
| TC-F09-058 | GetChangeCardService no longer calls SetEntityHistoryRepo | Inspect `services_global.go` | No `SetEntityHistoryRepo` call | |

#### AC-6.3: WireServices (HTTP server)

| Test ID | Description | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-F09-059 | cmd/server/services.go WireServices updated for new constructors | `make build` | Compiles successfully | |

---

## 3. Integration Scenarios

### Integration Scenario 1: Bug Status Transition End-to-End

**Components**: CLI command layer -> BugService -> EntityService -> EntityRepository -> workflow.Service

**Steps**:
1. Create bug via `shark bug create`
2. Advance status via `shark status advance B001`
3. Verify status changed
4. Verify history record created in `entity_history` table

**Validates**: EntityService delegation produces identical CLI output to pre-refactor behavior.

**Epic UAT Contribution**: Scenario 2 (single point of fix) -- BugService now flows through EntityService.

### Integration Scenario 2: ChangeCard Status Transition End-to-End

**Components**: CLI command layer -> ChangeCardService -> EntityService -> EntityRepository -> workflow.Service

**Steps**:
1. Create change card via `shark change create`
2. Advance status via `shark status advance CC-001`
3. Verify status changed
4. Verify history record created

**Validates**: Same as Integration Scenario 1 but for change cards.

### Integration Scenario 3: Task Transition with Auto-Unblock

**Components**: CLI command layer -> TaskService -> EntityService -> TaskRepository (StatusUpdateRaw) -> auto-unblock logic

**Steps**:
1. Create two tasks T1 and T2 where T2 depends on T1
2. Block T2 (set to blocked status)
3. Complete T1 via `shark status advance`
4. Verify T2 is auto-unblocked

**Validates**: Task-specific post-hook (auto-unblock) still works after full EntityService delegation.

**Epic UAT Contribution**: Scenario 2 (single point of fix) -- TaskService's transition validation is centralized but task-specific behavior preserved.

### Integration Scenario 4: Orchestrator Action Resolution

**Components**: BugService/ChangeCardService -> EntityService -> config.ActionService -> orchestrator actions

**Steps**:
1. Configure orchestrator action for bug status "ready_for_development"
2. Transition bug to "ready_for_development"
3. Verify orchestrator action populated in TransitionResult

**Validates**: `makeResolveActionFn` callback correctly generates PopulatedAction with entity-specific placeholders.

### Integration Scenario 5: Service Accessor Wiring Smoke Test

**Components**: CLI global accessors -> service constructors -> EntityService -> EntityRegistry

**Steps**:
1. Build project: `make build`
2. Run `shark bug list` (exercises GetBugService)
3. Run `shark change list` (exercises GetChangeCardService)
4. Run `shark status advance` on a task (exercises GetTaskService)

**Validates**: All service accessor wiring compiles and works at runtime.

---

## 4. Test Infrastructure

### Existing Test Patterns to Follow

| Pattern | File | Description |
|---|---|---|
| Mock BugRepository | `internal/services/bug_service_test.go` | `mockBugRepo` struct with function fields |
| Mock ChangeCardRepository | `internal/services/change_card_service_test.go` | `mockChangeCardRepo` struct with function fields |
| Mock TaskRepository | `internal/services/task_service_test.go` | `MockTaskRepository` struct with function fields |
| Mock EntityRepository | `internal/services/entity_service_test.go` | `mockEntityRepo` struct with function fields |
| EntityService test setup | `internal/services/entity_service_test.go` | Creates temp `.sharkconfig.json`, real `workflow.Service`, mock repos |
| EpicService makeResolveActionFn tests | `internal/services/epic_service_test.go` (lines ~1225-1270) | Tests for `makeResolveActionFn` callback behavior |
| FeatureService delegation tests | `internal/services/feature_service_test.go` | Tests TransitionStatus delegation to EntityService |
| Table-driven transition tests | `internal/services/task_service_test.go` | `TestTaskService_StartTask_Scenarios` pattern |

### New Test Helpers Needed

**No new test helpers are required.** The existing mock infrastructure is sufficient:

1. `mockBugRepo` (exists) -- needs no changes
2. `mockChangeCardRepo` (exists) -- needs no changes
3. `MockTaskRepository` (exists) -- needs no changes
4. `mockEntityRepo` (exists) -- already used by entity_service_test.go
5. `mockEntityHistoryRecorder` (exists in `entity_service_test.go`) -- already tests history recording

### Test Setup Changes Required

The primary test changes are **constructor call updates** in test files:

1. **`bug_service_test.go`**: All `NewBugService(repo, workflowSvc, ...)` calls must add `entitySvc` and `entityRepo` parameters
2. **`change_card_service_test.go`**: Same pattern as bug service
3. **`task_service_test.go`**: TransitionStatus and GetNextStatus tests may need mock EntityService or adjusted entity adapter setup

### Test Execution Commands

```bash
# Full regression
make fmt && make lint && make test

# Targeted service tests
go test -v ./internal/services/ -run TestBug
go test -v ./internal/services/ -run TestChangeCard
go test -v ./internal/services/ -run TestTask
go test -v ./internal/services/ -run TestResume

# Build verification (catches wiring issues)
make build
```

---

## 5. Non-Functional Requirements Validation

### REQ-NF-001: Behavioral Equivalence

**Validation Method**: Full existing test suite passes after implementation. No test logic changes allowed (only constructor setup changes).

| Check | Method |
|---|---|
| Bug transitions identical | Existing `TestBugService_SetBugStatus_*` tests pass |
| ChangeCard transitions identical | Existing `TestChangeCardService_SetStatus_*` tests pass |
| Task transitions identical | Existing `TestTaskService_TransitionStatus_*` tests pass |
| Task auto-unblock identical | Existing auto-unblock tests pass |
| History recording identical | EntityService already handles history for epic/feature; verify same for bug/change/task |
| Orchestrator actions identical | Existing action resolution tests pass |

### REQ-NF-002: No Performance Regression

**Validation Method**: No additional database queries per transition. EntityService.TransitionStatus is already used by EpicService and FeatureService without issues.

| Check | Method |
|---|---|
| No extra DB queries | Code review: BugService/ChangeCardService delegation adds zero additional repo calls beyond what EntityService already does |
| Re-fetch after transition | One additional `repo.GetByKey` call to return typed model (acceptable, already pattern for Advance methods) |

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Constructor signature change breaks callers | HIGH | LOW | Compiler catches all call sites; finite number of callers (tests, accessors, WireServices) |
| Task auto-unblock broken by delegation | MEDIUM | HIGH | TC-F09-039 through TC-F09-042 specifically test post-hook preservation |
| StatusUpdateRaw not called via adapter | MEDIUM | HIGH | TC-F09-036 verifies adapter routes through StatusUpdateRaw |
| Removed methods still referenced | LOW | LOW | Compiler catches; TC-F09-016/TC-F09-033 verify removal |
| ResumeService accidentally broken | LOW | MEDIUM | TC-F09-053/TC-F09-054 verify no regressions |

---

*Last Updated*: 2026-03-21
