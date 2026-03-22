# F03: Status Transition Unification -- Test Plan

**Feature**: E21-F03
**Author**: QA Agent
**Date**: 2026-03-20
**Status**: Draft
**Tier**: STANDARD

---

## 1. Acceptance Criteria Test Matrix

This matrix maps each PRD acceptance scenario to concrete test cases with expected results.

### Scenario 1: Epic TransitionStatus Delegates to EntityService

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-01 | Epic valid forward transition | Epic "E21" in status "draft" | `EpicService.TransitionStatus(ctx, "E21", "active", opts)` | TransitionResult with FromStatus="draft", ToStatus="active", Transitioned=true; EntityService handled validation; child feature count populated in post-hook | P0 |
| TC-02 | Epic transition identical to pre-refactoring | Epic "E21" in status "draft" | `EpicService.TransitionStatus(ctx, "E21", "active", opts)` | TransitionResult fields (EntityType, EntityKey, FromStatus, ToStatus, OrchestratorAction, ChildCount) match pre-refactoring output exactly | P0 |
| TC-03 | EpicService constructor accepts EntityService | EntityService and EntityRepository adapter created | `NewEpicService(repo, workflowSvc, entitySvc, entityRepo, ...)` | Service constructed without error; entitySvc field is non-nil | P1 |

### Scenario 2: Feature TransitionStatus Delegates to EntityService

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-04 | Feature valid forward transition | Feature "E21-F03" in status "draft" | `FeatureService.TransitionStatus(ctx, "E21-F03", "active", opts)` | TransitionResult with Transitioned=true; child task count populated in post-hook | P0 |
| TC-05 | Feature transition identical to pre-refactoring | Feature in status "active" | `FeatureService.TransitionStatus(ctx, key, "in_development", opts)` | Same TransitionResult as the existing FeatureService.TransitionStatus produces for the same inputs | P0 |

### Scenario 3: Task TransitionStatus Hybrid Delegation

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-06 | Task transition uses StatusUpdateRaw | Task "E21-F03-001" in status "todo" | `TaskService.TransitionStatus(ctx, key, "in_progress", opts)` | `StatusUpdateRaw` called on typed task repository (NOT via EntityRepository adapter); agent, notes, timestamps handled atomically | P0 |
| TC-07 | Task auto-unblock keys in result | Task blocked; dependents exist | `TaskService.TransitionStatus(ctx, key, "in_progress", opts)` | TransitionResult.Message contains auto-unblocked key list | P0 |
| TC-08 | Task feature progress recalculated | Task transitions to "completed" | `TaskService.TransitionStatus(ctx, key, "completed", opts)` | Feature progress recalculation triggered after status update | P1 |

### Scenario 4: Backward Transition with Reason

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-09 | Epic backward transition succeeds with reason | Epic in "in_development" | `TransitionStatus(ctx, "E21", "draft", {Reason: "Requirements changed"})` | Transition succeeds; IsBackward=true; rejection note created with reason text | P0 |
| TC-10 | Feature backward transition succeeds with reason | Feature in "active" | `TransitionStatus(ctx, key, "draft", {Reason: "Design rework"})` | Transition succeeds; IsBackward=true; result contains reason | P0 |
| TC-11 | Task backward transition succeeds with reason | Task in "in_development" | `TransitionStatus(ctx, key, "todo", {Reason: "Blocked"})` | Transition succeeds via hybrid path; backward detection from shared EntityService helper | P0 |

### Scenario 5: Backward Transition without Reason (Error)

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-12 | Epic backward without reason returns error | Epic in "in_development" | `TransitionStatus(ctx, "E21", "draft", {})` | BackwardReasonError returned; entity status unchanged | P0 |
| TC-13 | Feature backward without reason returns error | Feature in "active" | `TransitionStatus(ctx, key, "draft", {})` | BackwardReasonError returned; entity status unchanged | P0 |
| TC-14 | Task backward without reason returns error | Task in "in_progress" | `TransitionStatus(ctx, key, "todo", {})` | Error returned; StatusUpdateRaw NOT called | P0 |

### Scenario 6: Forced Transition

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-15 | Forced transition skips validation | Any entity in any status | `TransitionStatus(ctx, key, target, {Force: true, Reason: "Emergency"})` | Transition succeeds; IsForced=true; validation skipped | P0 |

### Scenario 7: Forced Transition without Reason (Error)

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-16 | Forced without reason returns error | Any entity | `TransitionStatus(ctx, key, target, {Force: true})` | ErrForceReasonRequired returned | P0 |

### Scenario 8: resolveAction Unification

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-17 | EntityService.ResolveAction produces identical output | Workflow config with orchestrator actions | `EntityService.ResolveAction(resolveActionFn, entity, status)` | PopulatedAction identical to pre-refactoring per-entity resolveAction output | P0 |
| TC-18 | ResolveAction nil workflow returns nil | No workflow configured | `EntityService.ResolveAction(...)` | Returns nil, no panic | P1 |
| TC-19 | ResolveAction nil status metadata returns nil | Status not in config | `EntityService.ResolveAction(...)` | Returns nil, no panic | P1 |
| TC-20 | ResolveAction nil OrchestratorAction returns nil | Status metadata has no action | `EntityService.ResolveAction(...)` | Returns nil, no panic | P1 |

### Scenario 9: GetNextStatus Unification

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-21 | Epic GetNextStatus delegates to shared impl | Epic in "draft" | `EpicService.GetNextStatus(ctx, "E21")` | Available transitions with actions resolved via entity-specific placeholder fn; result identical to pre-refactoring | P0 |
| TC-22 | Feature GetNextStatus delegates | Feature in "draft" | `FeatureService.GetNextStatus(ctx, key)` | Identical to pre-refactoring | P0 |
| TC-23 | Task GetNextStatus delegates | Task in "todo" | `TaskService.GetNextStatus(ctx, key)` | Identical to pre-refactoring | P0 |
| TC-24 | GetNextStatus for terminal status | Any entity in "completed" | `GetNextStatus(ctx, key)` | Empty transitions list or error indicating terminal status | P1 |

### Scenario 10: Quality Gate

| TC ID | Test Case | Precondition | Action | Expected Result | Priority |
|-------|-----------|--------------|--------|-----------------|----------|
| TC-25 | Full quality gate passes | All F03 refactoring complete | `make fmt && make lint && make test` | Zero formatting changes, zero lint warnings, zero test failures | P0 |

---

## 2. Component Test Strategy

### 2.1 EntityService Unit Tests (`entity_service_test.go`)

**New file.** This is the primary test file for F03 and should cover the shared logic comprehensively.

**Mock Requirements:**
- `MockEntityRepository` implementing the `EntityRepository` interface from F01 (methods: `GetByKey`, `UpdateStatus`)
- Reuse existing workflow service test helpers (`newTestEpicWorkflowService`, `newTestEpicWorkflowServiceWithActions` patterns)

**Test Structure -- Parameterized by Entity Type:**

```
TestEntityService_TransitionStatus/
    entity_types=[epic, feature, task_mock, bug, change_card]  (via MockEntityRepository)

    happy_path_forward_transition
    forced_transition_with_reason
    forced_transition_without_reason_error
    backward_transition_with_reason (features.DetectBackward=true)
    backward_transition_without_reason_error (features.DetectBackward=true)
    backward_detection_disabled (features.DetectBackward=false, e.g. Bug)
    entity_not_found_error
    repo_update_error_propagation
    repo_get_error_propagation
    transition_validation_error_from_workflow

TestEntityService_TransitionStatus_TransitionFeatures/
    DefaultTransitionFeatures -- all detection active
    SimpleTransitionFeatures -- backward detection off, rejection notes off
    Custom_combination -- only ResolveOrchestratorAction

TestEntityService_ValidateAndNormalize/
    valid_transition
    invalid_transition_returns_error
    forced_skips_validation
    normalization_applied_when_not_forced

TestEntityService_DetectBackward/
    forward_transition_returns_false
    backward_transition_returns_true
    same_status_returns_false
    workflow_service_nil_returns_false_gracefully

TestEntityService_ResolveAction/
    with_valid_workflow_and_action -- callback invoked, placeholders merged
    nil_workflow -- returns_nil
    missing_status_metadata -- returns_nil
    nil_orchestrator_action_in_metadata -- returns_nil
    extra_placeholders_merged

TestEntityService_GetNextStatus/
    entity_with_available_transitions
    entity_in_terminal_status
    entity_not_found
    action_resolution_callback_invoked_per_transition
```

**Coverage Target:** 85%+ for all EntityService methods.

### 2.2 TaskService Hybrid Delegation Tests (Updates to `task_service_test.go`)

**Existing tests that MUST continue to pass unchanged** (behavioral regression guard):

- `TestTaskService_TransitionStatus_UsesStatusUpdateRaw` -- verifies `StatusUpdateRaw` is called
- `TestTaskService_TransitionStatus_Forced_UsesStatusUpdateRaw` -- forced path still calls `StatusUpdateRaw`
- `TestTaskService_TransitionStatus_BackwardRequiresReason` -- backward detection works
- `TestTaskService_TransitionStatus_AutoUnblockInResult` -- auto-unblock keys in result

**Test modifications required:**
- Constructor calls updated to pass `*EntityService` (or nil EntityService for tests that don't exercise shared logic)
- Mock EntityService OR real EntityService with mocked workflow.Service

**New tests for hybrid delegation:**

```
TestTaskService_TransitionStatus_SharedValidationFromEntityService/
    validation_error_from_EntityService_ValidateAndNormalize_propagates
    backward_detection_uses_EntityService_DetectBackward
    rejection_note_created_after_StatusUpdateRaw_success

TestTaskService_TransitionStatus_DoesNotUseEntityRepositoryUpdateStatus/
    verify_EntityRepository.UpdateStatus_never_called
    verify_StatusUpdateRaw_called_instead
```

### 2.3 EpicService Delegation Tests (Updates to `epic_service_test.go`)

**Existing tests that MUST continue to pass unchanged:**

- `TestEpicService_TransitionStatus_Valid` (TC-01 equivalent)
- `TestEpicService_TransitionStatus_Invalid`
- `TestEpicService_TransitionStatus_Force`
- `TestEpicService_TransitionStatus_NotFound`
- `TestEpicService_TransitionStatus_RepoError`
- `TestEpicService_TransitionStatus_UpdateError`
- `TestEpicService_TransitionStatus_WithAction`
- `TestEpicService_TransitionStatus_WithoutAction`
- `TestEpicService_TransitionStatus_ActionJSON`
- `TestEpicService_GetNextStatus` and variants
- `TestEpicService_BackwardTransition_*` (from `backward_transition_test.go`)

**Test modifications required:**
- Constructor updated to pass `*EntityService` and `EntityRepository` adapter
- Test assertions remain identical (same `TransitionResult` fields, same error types)

**New tests:**

```
TestEpicService_TransitionStatus_ChildCountPostHook/
    epic_with_3_features -- ChildCount = 3 in result
    epic_with_0_features -- ChildCount = 0 in result

TestEpicService_Delegation_InlineLogicRemoved/
    verify_no_direct_workflow_validation_call_from_EpicService (audit, not automated)
```

### 2.4 FeatureService Delegation Tests (Updates to `feature_service_test.go`)

**Same pattern as EpicService.** Existing tests must pass. Constructor updated.

**Existing tests that MUST continue to pass unchanged:**

- `TestFeatureService_TransitionStatus_Valid` through `TestFeatureService_TransitionStatus_UpdateError`
- `TestFeatureService_TransitionStatus_WithAction` / `WithoutAction`
- `TestFeatureService_GetNextStatus` and variants
- `TestFeatureService_BackwardTransition_*`

**New tests:**

```
TestFeatureService_TransitionStatus_ChildCountPostHook/
    feature_with_5_tasks -- ChildCount = 5 in result
    feature_with_0_tasks -- ChildCount = 0 in result
```

### 2.5 Bug/ChangeCard Optional Delegation Tests (Should-Have)

If Story 6 is implemented:

```
TestBugService_SetBugStatus_DelegatesToEntityService/
    uses_SimpleTransitionFeatures (DetectBackward=false)
    no_backward_detection
    no_rejection_notes
    typed_bug_reloaded_after_transition

TestChangeCardService_SetChangeCardStatus_DelegatesToEntityService/
    same_pattern_as_bug
```

### 2.6 Transition Types Tests (Updates to existing)

Existing tests in `transition_types_test.go` and `backward_transition_test.go` must pass unchanged. These validate JSON serialization of `TransitionResult` and `NextStatusInfo` -- no behavioral change expected.

---

## 3. Integration Scenarios for CLI Wiring

### 3.1 `services_global.go` Wiring Verification

The CLI wiring must create an `EntityService` and inject it into entity-specific services.

| TC ID | Test Case | Verification Method | Expected |
|-------|-----------|---------------------|----------|
| INT-01 | `GetEpicService()` returns service with EntityService | Code inspection + `make build` | EpicService constructor receives non-nil `*EntityService` |
| INT-02 | `GetFeatureService()` returns service with EntityService | Code inspection + `make build` | FeatureService constructor receives non-nil `*EntityService` |
| INT-03 | `GetTaskService()` returns service with EntityService | Code inspection + `make build` | TaskService constructor receives non-nil `*EntityService` |
| INT-04 | EntityService shares the global workflow.Service | Code inspection | Single `workflow.Service` instance used by EntityService and entity services |
| INT-05 | EntityRepository adapters created per entity type | Code inspection + build | Epic/Feature adapters created; Task does NOT get adapter for transition path |

### 3.2 CLI Smoke Tests

These verify the complete CLI path works after wiring changes. Run manually or via integration test.

| TC ID | Command | Expected |
|-------|---------|----------|
| CLI-01 | `shark status advance E21-F03-001` | Task advances to next status; output matches pre-refactoring |
| CLI-02 | `shark status set E21 active --reason "Starting"` | Epic status set; result includes OrchestratorAction if configured |
| CLI-03 | `shark status advance E21-F03 --json` | Feature advances; JSON output includes TransitionResult fields |
| CLI-04 | `shark status options E21-F03-001` | Lists available next statuses via shared GetNextStatus |
| CLI-05 | `shark status set E21 draft --reason "Reverting" --force` | Forced backward transition succeeds |

---

## 4. Regression Test Strategy

### 4.1 Existing Test Inventory (Must-Pass)

The following test counts represent the regression baseline. ALL must pass after F03:

| Test File | Relevant Test Count | Risk Level |
|-----------|---------------------|------------|
| `epic_service_test.go` | ~15 TransitionStatus/GetNextStatus tests | HIGH -- constructors change |
| `feature_service_test.go` | ~12 TransitionStatus/GetNextStatus tests | HIGH -- constructors change |
| `task_service_test.go` | ~4 TransitionStatus tests | HIGH -- hybrid delegation is most complex change |
| `backward_transition_test.go` | ~9 backward/force transition tests | HIGH -- backward detection moves to shared code |
| `transition_types_test.go` | ~8 JSON serialization tests | LOW -- no logic change, only types consumed |
| `change_card_service_test.go` | ~1 SetChangeCardStatus test | MEDIUM -- optional delegation (Story 6) |
| `bug_service_test.go` | (status transition tests) | MEDIUM -- optional delegation (Story 6) |

### 4.2 Constructor Signature Update Strategy

The primary source of test breakage will be constructor signature changes. Every test that creates an `EpicService`, `FeatureService`, or `TaskService` must be updated.

**Approach:**
1. Add `*EntityService` parameter to constructors
2. In test files, create a test helper `newTestEntityService(t)` that returns an `*EntityService` with a real `workflow.Service` (pointing to temp config, matching the existing `newTestEpicWorkflowService()` pattern)
3. Pass `newTestEntityService(t)` to all service constructors in tests
4. Existing behavioral assertions remain UNCHANGED -- only constructor calls are updated

**Example pattern:**
```go
func newTestEntityService(t *testing.T) *EntityService {
    t.Helper()
    workflowSvc := workflow.NewService("")  // default config
    return NewEntityService(workflowSvc)
}
```

### 4.3 Regression Verification Command

```bash
# Run after every F03 task completion
make fmt && make lint && make test

# Focused regression for transition logic
go test -v ./internal/services/ -run "Transition|NextStatus|Backward|Force|ResolveAction"
```

### 4.4 Line Count Verification

```bash
# Capture baseline BEFORE F03 work begins
wc -l internal/services/epic_service.go internal/services/feature_service.go \
      internal/services/task_service.go internal/services/bug_service.go \
      internal/services/change_card_service.go > /tmp/f03-baseline.txt

# After F03 completion
wc -l internal/services/epic_service.go internal/services/feature_service.go \
      internal/services/task_service.go internal/services/bug_service.go \
      internal/services/change_card_service.go internal/services/entity_service.go \
      > /tmp/f03-after.txt

# Net reduction target: 300+ lines
```

### 4.5 Duplication Elimination Verification

```bash
# After F03: backward detection should appear only in entity_service.go
grep -rn "IsBackwardTransition\|detectBackward\|DetectBackward" internal/services/*_service.go

# Expected: only entity_service.go contains the implementation
# epic_service.go, feature_service.go, task_service.go should only have delegation calls

# After F03: resolveAction should be unified
grep -c "func.*resolveAction\|func.*ResolveAction" internal/services/*_service.go

# Expected: 1 implementation in entity_service.go; 0 private resolveAction methods in others

# After F03: GetNextStatus should be unified
grep -c "func.*GetNextStatus" internal/services/*_service.go

# Expected: 1 shared in entity_service.go; delegation calls in epic/feature/task services
```

---

## 5. Error Path Test Cases

| TC ID | Error Scenario | Component | Expected Behavior |
|-------|---------------|-----------|-------------------|
| ERR-01 | EntityRepository.GetByKey returns not-found | EntityService.TransitionStatus | Error propagated with entity type and key context |
| ERR-02 | EntityRepository.UpdateStatus fails | EntityService.TransitionStatus | Error returned BEFORE rejection note creation (no partial state) |
| ERR-03 | Task StatusUpdateRaw fails after shared validation passes | TaskService.TransitionStatus | Error clearly indicates task-specific update failure, not shared validation failure |
| ERR-04 | Workflow service returns nil for GetWorkflow() | EntityService.ResolveAction | Returns nil gracefully (no panic) |
| ERR-05 | Status not in workflow metadata | EntityService.ResolveAction | Returns nil gracefully |
| ERR-06 | Child count query fails in post-hook | EpicService.TransitionStatus | Transition itself succeeds; child count error logged or returned separately |
| ERR-07 | Invalid status string (not in workflow) | EntityService.ValidateAndNormalize | Validation error returned with descriptive message |

---

## 6. Non-Functional Validation

| Metric | Target | Verification |
|--------|--------|-------------|
| Test pass rate | 100% | `make test` |
| Lint warnings | 0 | `make lint` |
| EntityService coverage | 85%+ | `go test -cover ./internal/services/ -run EntityService` |
| Net line reduction | 300+ lines | `wc -l` comparison (Section 4.4) |
| Performance impact | <1ms additional latency | Interface dispatch is ~2ns; negligible vs DB I/O |
| TransitionStatus implementations | 1 shared (EntityService) | Grep verification (Section 4.5) |
| resolveAction implementations | 1 shared (EntityService) | Grep verification (Section 4.5) |

---

*Last Updated*: 2026-03-20
