# UAT Evidence: E21-F03 Status Transition Unification

**Collected**: 2026-03-20
**Branch**: e21-f01-entity-interface-foundation
**Collector**: Developer Agent (evidence collection only, no assessment)

---

## Structural Metrics

### File Line Counts

```
   267 internal/services/entity_service.go
   957 internal/services/epic_service.go
  1097 internal/services/feature_service.go
  1420 internal/services/task_service.go
  3741 total
```

### Backward Detection Location (grep: IsBackwardTransition|detectBackward|DetectBackward)

```
internal/services/entity_service.go:15:   // DetectBackward enables backward transition detection and reason requirement.
internal/services/entity_service.go:18:   DetectBackward bool
internal/services/entity_service.go:33:      DetectBackward:            true,
internal/services/entity_service.go:42:      DetectBackward:            false,
internal/services/entity_service.go:130:  if features.DetectBackward {
internal/services/entity_service.go:131:     isBackward, err = s.DetectBackward(currentStatus, targetStatus, opts.Force, opts.Reason)
internal/services/entity_service.go:177:// DetectBackward checks if a transition is backward and enforces reason requirements.
internal/services/entity_service.go:179:// If force is true and IsBackwardTransition errors, isBackward is set to false (graceful).
internal/services/entity_service.go:180:func (s *EntityService) DetectBackward(currentStatus, targetStatus string, force bool, reason string) (bool, error) {
internal/services/entity_service.go:181:  isBackward, err := s.workflowSvc.IsBackwardTransition(currentStatus, targetStatus)
internal/services/task_service.go:624:    isBackward, err := s.entitySvc.DetectBackward(fromStatus, targetStatus, opts.Force, opts.Reason)
internal/services/task_service.go:1327:      isBackward, err = s.entitySvc.GetWorkflowService().IsBackwardTransition(fromStatus, targetStatus)
```

**Observation**: `IsBackwardTransition` called directly only in `entity_service.go:181` and `task_service.go:1327` (the latter is in `executeStatusTransition`, a different code path). Epic and Feature services do NOT contain direct backward detection logic.

### resolveAction Count Per Service File

```
internal/services/bug_service.go:1
internal/services/change_card_service.go:1
internal/services/entity_service.go:1       (ResolveActionForStatus)
internal/services/epic_service.go:1         (makeResolveActionFn - closure)
internal/services/feature_service.go:1      (makeResolveActionFn - closure)
internal/services/task_service.go:1         (resolveAction - delegates to entitySvc.ResolveActionForStatus)
```

### workflowSvc Direct Usage in Epic/Feature Services

```
grep "workflowSvc" epic_service.go:    No matches
grep "workflowSvc" feature_service.go: Only 2 matches, both in comments/constructor doc:
  104://   - If workflowSvc is nil (required dependency)
  158:   // workflowSvc is passed unscoped; FeatureProgressService uses ForLevel(LevelTask) internally.
```

**Observation**: Neither EpicService nor FeatureService have a `workflowSvc` field or direct workflow calls. All workflow access goes through `entitySvc`.

### EntityService Test Coverage

```
entity_service.go:31:   DefaultTransitionFeatures     100.0%
entity_service.go:40:   SimpleTransitionFeatures       100.0%
entity_service.go:63:   NewEntityService               100.0%
entity_service.go:73:   ForLevel                       100.0%
entity_service.go:96:   TransitionStatus                95.5%
entity_service.go:167:  ValidateAndNormalize           100.0%
entity_service.go:180:  DetectBackward                  90.9%
entity_service.go:201:  GetWorkflowService               0.0%
entity_service.go:208:  ResolveActionForStatus          71.4%
entity_service.go:228:  GetNextStatus                   93.3%
```

---

## Scenario 1: Epic TransitionStatus Delegates to EntityService

### PRD Spec Quote

> **Scenario 1: Epic TransitionStatus Delegates to EntityService**
> - Given EpicService is constructed with an EntityService and EntityRepository adapter
> - When `EpicService.TransitionStatus(ctx, "E21", "in_progress", opts)` is called
> - Then the shared `EntityService.TransitionStatus` handles validation, backward detection, status update, and rejection note creation
> - And EpicService adds child feature count to the TransitionResult in its post-hook
> - And the result is identical to the pre-refactoring behavior

### Implementation Code

**File: `internal/services/epic_service.go:229-262`**

```go
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
    entityRepo := s.entityRepo
    if entityRepo == nil {
        entityRepo = &epicEntityRepoAdapter{repo: s.repo}
    }

    // Delegate shared logic to EntityService
    result, err := s.entitySvc.TransitionStatus(
        ctx, entityRepo, "epic", epicKey, targetStatus, opts,
        DefaultTransitionFeatures(),
        s.makeResolveActionFn(ctx),
    )
    if err != nil {
        return nil, err
    }

    // Post-hook: rejection note (entity-specific note repo)
    if (result.IsBackward || result.IsForced) && opts.Reason != "" && s.noteRepo != nil {
        _ = s.noteRepo.CreateRejectionNote(ctx, "epic", result.EntityID,
            0, result.FromStatus, result.ToStatus,
            opts.Reason, opts.Agent, opts.DocumentPath)
    }

    // Post-hook: count child features
    if s.featureRepo != nil {
        features, listErr := s.featureRepo.ListByEpic(ctx, result.EntityID)
        if listErr == nil {
            result.ChildCount = len(features)
        }
    }

    return result, nil
}
```

**File: `internal/services/epic_service.go:264-272` (GetNextStatus delegation)**

```go
func (s *EpicService) GetNextStatus(ctx context.Context, epicKey string) (*NextStatusInfo, error) {
    entityRepo := s.entityRepo
    if entityRepo == nil {
        entityRepo = &epicEntityRepoAdapter{repo: s.repo}
    }
    return s.entitySvc.GetNextStatus(ctx, entityRepo, "epic", epicKey,
        s.makeResolveActionFn(ctx))
}
```

**EpicService struct (line 76-88)** shows `entitySvc *EntityService` and `entityRepo EntityRepository` fields.

### Test Code

Tests in `internal/services/epic_service_test.go`:
- `TestEpicService_TransitionStatus_Valid`
- `TestEpicService_TransitionStatus_Invalid`
- `TestEpicService_TransitionStatus_Force`
- `TestEpicService_TransitionStatus_NotFound`
- `TestEpicService_TransitionStatus_RepoError`
- `TestEpicService_TransitionStatus_UpdateError`
- `TestEpicService_TransitionStatus_WithAction`
- `TestEpicService_TransitionStatus_WithoutAction`
- `TestEpicService_TransitionStatus_ActionJSON`

### Test Execution Output

```
=== RUN   TestEpicService_BackwardTransition_RequiresReason
--- PASS: TestEpicService_BackwardTransition_RequiresReason (0.00s)
=== RUN   TestEpicService_BackwardTransition_WithReason
--- PASS: TestEpicService_BackwardTransition_WithReason (0.00s)
=== RUN   TestEpicService_ForceTransition_RequiresReason
--- PASS: TestEpicService_ForceTransition_RequiresReason (0.00s)
=== RUN   TestEpicService_ForwardTransition_NoReasonRequired
--- PASS: TestEpicService_ForwardTransition_NoReasonRequired (0.00s)
=== RUN   TestEpicService_TransitionStatus_Valid
--- PASS: TestEpicService_TransitionStatus_Valid (0.00s)
=== RUN   TestEpicService_TransitionStatus_Invalid
--- PASS: TestEpicService_TransitionStatus_Invalid (0.00s)
=== RUN   TestEpicService_TransitionStatus_Force
--- PASS: TestEpicService_TransitionStatus_Force (0.00s)
=== RUN   TestEpicService_TransitionStatus_NotFound
--- PASS: TestEpicService_TransitionStatus_NotFound (0.00s)
=== RUN   TestEpicService_TransitionStatus_RepoError
--- PASS: TestEpicService_TransitionStatus_RepoError (0.00s)
=== RUN   TestEpicService_TransitionStatus_UpdateError
--- PASS: TestEpicService_TransitionStatus_UpdateError (0.00s)
=== RUN   TestEpicService_TransitionStatus_WithAction
--- PASS: TestEpicService_TransitionStatus_WithAction (0.00s)
=== RUN   TestEpicService_TransitionStatus_WithoutAction
--- PASS: TestEpicService_TransitionStatus_WithoutAction (0.00s)
=== RUN   TestEpicService_TransitionStatus_ActionJSON
--- PASS: TestEpicService_TransitionStatus_ActionJSON (0.00s)
PASS
ok    github.com/jwwelbor/shark-task-manager/internal/services  0.008s
```

---

## Scenario 2: Feature TransitionStatus Delegates to EntityService

### PRD Spec Quote

> **Scenario 2: Feature TransitionStatus Delegates to EntityService**
> - Given FeatureService is constructed with an EntityService and EntityRepository adapter
> - When `FeatureService.TransitionStatus(ctx, "E21-F03", "in_development", opts)` is called
> - Then the shared `EntityService.TransitionStatus` handles validation, backward detection, status update, and rejection note creation
> - And FeatureService adds child task count to the TransitionResult in its post-hook

### Implementation Code

**File: `internal/services/feature_service.go:234-267`**

```go
func (s *FeatureService) TransitionStatus(ctx context.Context, featureKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
    entityRepo := s.entityRepo
    if entityRepo == nil {
        entityRepo = &featureEntityRepoAdapter{repo: s.repo}
    }

    // Delegate shared logic to EntityService
    result, err := s.entitySvc.TransitionStatus(
        ctx, entityRepo, "feature", featureKey, targetStatus, opts,
        DefaultTransitionFeatures(),
        s.makeResolveActionFn(ctx),
    )
    if err != nil {
        return nil, err
    }

    // Post-hook: rejection note (entity-specific note repo)
    if (result.IsBackward || result.IsForced) && opts.Reason != "" && s.noteRepo != nil {
        _ = s.noteRepo.CreateRejectionNote(ctx, "feature", result.EntityID,
            0, result.FromStatus, result.ToStatus,
            opts.Reason, opts.Agent, opts.DocumentPath)
    }

    // Post-hook: count child tasks
    if s.taskRepo != nil {
        tasks, listErr := s.taskRepo.ListByFeature(ctx, result.EntityID)
        if listErr == nil {
            result.ChildCount = len(tasks)
        }
    }

    return result, nil
}
```

**File: `internal/services/feature_service.go:269-277` (GetNextStatus delegation)**

```go
func (s *FeatureService) GetNextStatus(ctx context.Context, featureKey string) (*NextStatusInfo, error) {
    entityRepo := s.entityRepo
    if entityRepo == nil {
        entityRepo = &featureEntityRepoAdapter{repo: s.repo}
    }
    return s.entitySvc.GetNextStatus(ctx, entityRepo, "feature", featureKey,
        s.makeResolveActionFn(ctx))
}
```

**FeatureService struct (line 84-96)** shows `entitySvc *EntityService` and `entityRepo EntityRepository` fields.

### Test Code

Tests in `internal/services/feature_service_test.go`:
- `TestFeatureService_TransitionStatus_Valid`
- `TestFeatureService_TransitionStatus_Invalid`
- `TestFeatureService_TransitionStatus_Force`
- `TestFeatureService_TransitionStatus_NotFound`
- `TestFeatureService_TransitionStatus_RepoError`
- `TestFeatureService_TransitionStatus_UpdateError`
- `TestFeatureService_TransitionStatus_WithAction`
- `TestFeatureService_TransitionStatus_WithoutAction`

### Test Execution Output

```
=== RUN   TestFeatureService_BackwardTransition_RequiresReason
--- PASS: TestFeatureService_BackwardTransition_RequiresReason (0.00s)
=== RUN   TestFeatureService_ForceTransition_RequiresReason
--- PASS: TestFeatureService_ForceTransition_RequiresReason (0.00s)
=== RUN   TestFeatureService_ForwardTransition_NoReasonRequired
--- PASS: TestFeatureService_ForwardTransition_NoReasonRequired (0.00s)
=== RUN   TestFeatureService_TransitionStatus_Valid
--- PASS: TestFeatureService_TransitionStatus_Valid (0.00s)
=== RUN   TestFeatureService_TransitionStatus_Invalid
--- PASS: TestFeatureService_TransitionStatus_Invalid (0.00s)
=== RUN   TestFeatureService_TransitionStatus_Force
--- PASS: TestFeatureService_TransitionStatus_Force (0.00s)
=== RUN   TestFeatureService_TransitionStatus_NotFound
--- PASS: TestFeatureService_TransitionStatus_NotFound (0.00s)
=== RUN   TestFeatureService_TransitionStatus_RepoError
--- PASS: TestFeatureService_TransitionStatus_RepoError (0.00s)
=== RUN   TestFeatureService_TransitionStatus_UpdateError
--- PASS: TestFeatureService_TransitionStatus_UpdateError (0.00s)
=== RUN   TestFeatureService_TransitionStatus_WithAction
--- PASS: TestFeatureService_TransitionStatus_WithAction (0.00s)
=== RUN   TestFeatureService_TransitionStatus_WithoutAction
--- PASS: TestFeatureService_TransitionStatus_WithoutAction (0.00s)
PASS
ok    github.com/jwwelbor/shark-task-manager/internal/services  0.005s
```

---

## Scenario 3: Task TransitionStatus Hybrid Delegation

### PRD Spec Quote

> **Scenario 3: Task TransitionStatus Hybrid Delegation**
> - Given TaskService is constructed with both its typed repository and EntityService
> - When `TaskService.TransitionStatus(ctx, "E21-F03-001", "in_progress", opts)` is called
> - Then shared validation and backward detection are reused from EntityService
> - And the atomic `StatusUpdateRaw` is called via the typed task repository (not through EntityRepository adapter)
> - And auto-unblocked keys are included in the result message
> - And feature progress is recalculated after the transition

### Implementation Code

**File: `internal/services/task_service.go:603-676`**

```go
func (s *TaskService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
    // Step 1: Get task (task-specific typed repo)
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get task %s: %w", key, err)
    }
    fromStatus := string(task.Status)

    // Step 2: Shared validation and normalization (delegated to EntityService)
    targetStatus, err = s.entitySvc.ValidateAndNormalize(fromStatus, targetStatus, opts.Force)
    if err != nil {
        return nil, err
    }

    // Step 3: Shared force-reason enforcement
    if opts.Force && opts.Reason == "" {
        return nil, ErrForceReasonRequired
    }

    // Step 4: Shared backward detection (delegated to EntityService)
    isBackward, err := s.entitySvc.DetectBackward(fromStatus, targetStatus, opts.Force, opts.Reason)
    if err != nil {
        return nil, err
    }

    // Step 5: Task-specific atomic update via StatusUpdateRaw
    // ... (builds params and calls s.executeStatusTransition)

    txResult, err := s.executeStatusTransition(ctx, task, statusTransitionOpts{
        targetStatus:      targetStatus,
        agent:             agentPtr,
        reason:            reasonPtr,
        documentPath:      docPathPtr,
        force:             opts.Force,
        skipBackwardCheck: true, // already done above via EntityService
    })

    // Step 6: Build result with shared action resolution
    result := &TransitionResult{
        // ...
        OrchestratorAction: s.resolveAction(ctx, task, actualTarget),
    }
    return result, nil
}
```

Key observations:
- Line 613: `s.entitySvc.ValidateAndNormalize(...)` -- delegates validation
- Line 624: `s.entitySvc.DetectBackward(...)` -- delegates backward detection
- Line 641: `s.executeStatusTransition(...)` -- uses task-specific `StatusUpdateRaw` (NOT EntityRepository.UpdateStatus)
- Line 647: `skipBackwardCheck: true` -- avoids duplicate backward check
- Line 666: `s.resolveAction(...)` which internally calls `s.entitySvc.ResolveActionForStatus(...)` (line 793)

### Test Code

**File: `internal/services/task_service_test.go:2175-2210`** (`TestTaskService_TransitionStatus_UsesStatusUpdateRaw`)

```go
func TestTaskService_TransitionStatus_UsesStatusUpdateRaw(t *testing.T) {
    var capturedParams models.StatusUpdateParams
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{ID: 6, Key: "T-E07-F01-006", Status: "todo", FeatureID: 10}, nil
        },
        StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
            capturedParams = params
            return nil, nil
        },
    }
    svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil, nil)
    result, err := svc.TransitionStatus(context.Background(), "T-E07-F01-006", "in_progress", TransitionOptions{
        Agent: "my-agent", Reason: "starting work",
    })
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.True(t, result.Transitioned)
    assert.Equal(t, "todo", result.FromStatus)
    assert.Equal(t, "in_progress", result.ToStatus)
    assert.Equal(t, int64(6), capturedParams.TaskID)
    assert.Equal(t, "todo", capturedParams.OldStatus)
    assert.NotNil(t, capturedParams.Agent)
    assert.Equal(t, "my-agent", *capturedParams.Agent)
}
```

### Test Execution Output

```
=== RUN   TestTaskService_TransitionStatus_UsesStatusUpdateRaw
--- PASS: TestTaskService_TransitionStatus_UsesStatusUpdateRaw (0.00s)
=== RUN   TestTaskService_TransitionStatus_Forced_UsesStatusUpdateRaw
--- PASS: TestTaskService_TransitionStatus_Forced_UsesStatusUpdateRaw (0.00s)
=== RUN   TestTaskService_TransitionStatus_BackwardRequiresReason
--- PASS: TestTaskService_TransitionStatus_BackwardRequiresReason (0.00s)
=== RUN   TestTaskService_TransitionStatus_AutoUnblockInResult
--- PASS: TestTaskService_TransitionStatus_AutoUnblockInResult (0.00s)
PASS
ok    github.com/jwwelbor/shark-task-manager/internal/services  0.004s
```

---

## Scenario 4: Backward Transition with Reason

### PRD Spec Quote

> **Scenario 4: Backward Transition with Reason**
> - Given an Epic in status "in_development"
> - When `TransitionStatus(ctx, "E21", "draft", TransitionOptions{Reason: "Requirements changed"})` is called
> - Then backward detection identifies this as a backward transition
> - And the transition succeeds because a reason is provided
> - And a rejection note is created with the reason

### Implementation Code

**File: `internal/services/entity_service.go:180-196`** (DetectBackward)

```go
func (s *EntityService) DetectBackward(currentStatus, targetStatus string, force bool, reason string) (bool, error) {
    isBackward, err := s.workflowSvc.IsBackwardTransition(currentStatus, targetStatus)
    if err != nil {
        if !force {
            return false, fmt.Errorf("could not determine transition direction: %w", err)
        }
        return false, nil
    }
    if isBackward && !force {
        wf := s.workflowSvc.GetWorkflow()
        requireReason := wf == nil || wf.RequireRejectionReason
        if requireReason && reason == "" {
            return true, &BackwardReasonError{FromStatus: currentStatus, ToStatus: targetStatus}
        }
    }
    return isBackward, nil
}
```

### Test Code

Tests:
- `TestEntityService_TransitionStatus_BackwardWithReason` (entity_service_test.go:214)
- `TestEpicService_BackwardTransition_WithReason` (epic_service_test.go)
- `TestEntityService_DetectBackward_Backward_WithReason` (entity_service_test.go:395)

### Test Execution Output

```
=== RUN   TestEpicService_BackwardTransition_WithReason
--- PASS: TestEpicService_BackwardTransition_WithReason (0.00s)
=== RUN   TestEntityService_TransitionStatus_BackwardWithReason
--- PASS: TestEntityService_TransitionStatus_BackwardWithReason (0.00s)
=== RUN   TestEntityService_DetectBackward_Backward_WithReason
--- PASS: TestEntityService_DetectBackward_Backward_WithReason (0.00s)
```

---

## Scenario 5: Backward Transition without Reason (Error)

### PRD Spec Quote

> **Scenario 5: Backward Transition without Reason (Error)**
> - Given a Feature in status "in_development"
> - When `TransitionStatus(ctx, "E21-F03", "draft", TransitionOptions{})` is called (no reason)
> - Then a `BackwardReasonError` is returned with from-status and to-status
> - And the entity status is not changed

### Implementation Code

Same `DetectBackward` method at `entity_service.go:180-196`. When `isBackward && !force && reason == ""`, returns `&BackwardReasonError{FromStatus: currentStatus, ToStatus: targetStatus}`.

**File: `internal/services/transition_types.go:20-32`** (BackwardReasonError)

```go
type BackwardReasonError struct {
    FromStatus string
    ToStatus   string
}

func (e *BackwardReasonError) Error() string {
    return fmt.Sprintf("backward transition from '%s' to '%s' requires --reason flag", e.FromStatus, e.ToStatus)
}
```

### Test Code

Tests:
- `TestEntityService_TransitionStatus_BackwardWithoutReason` (entity_service_test.go:240)
- `TestEntityService_DetectBackward_Backward_WithoutReason` (entity_service_test.go:407)
- `TestEpicService_BackwardTransition_RequiresReason` (epic_service_test.go)
- `TestFeatureService_BackwardTransition_RequiresReason` (feature_service_test.go)
- `TestTaskService_TransitionStatus_BackwardRequiresReason` (task_service_test.go:2242)

### Test Execution Output

```
=== RUN   TestEpicService_BackwardTransition_RequiresReason
--- PASS: TestEpicService_BackwardTransition_RequiresReason (0.00s)
=== RUN   TestFeatureService_BackwardTransition_RequiresReason
--- PASS: TestFeatureService_BackwardTransition_RequiresReason (0.00s)
=== RUN   TestEntityService_TransitionStatus_BackwardWithoutReason
--- PASS: TestEntityService_TransitionStatus_BackwardWithoutReason (0.00s)
=== RUN   TestEntityService_DetectBackward_Backward_WithoutReason
--- PASS: TestEntityService_DetectBackward_Backward_WithoutReason (0.00s)
=== RUN   TestTaskService_TransitionStatus_BackwardRequiresReason
--- PASS: TestTaskService_TransitionStatus_BackwardRequiresReason (0.00s)
```

---

## Scenario 6: Forced Transition

### PRD Spec Quote

> **Scenario 6: Forced Transition**
> - Given any entity in any status
> - When `TransitionStatus(ctx, key, targetStatus, TransitionOptions{Force: true, Reason: "Emergency fix"})` is called
> - Then transition validation is skipped
> - And the transition succeeds with `IsForced=true` in the result

### Implementation Code

**File: `internal/services/entity_service.go:96-162`** (TransitionStatus lines 117-121 and 123-125)

```go
// Steps 3-4: Validate and normalize
targetStatus, err = s.ValidateAndNormalize(currentStatus, targetStatus, opts.Force)
// ...

// Step 5: Enforce reason for forced transitions
if opts.Force && opts.Reason == "" {
    return nil, ErrForceReasonRequired
}
```

**File: `internal/services/entity_service.go:167-175`** (ValidateAndNormalize)

```go
func (s *EntityService) ValidateAndNormalize(currentStatus, targetStatus string, force bool) (string, error) {
    if !force {
        if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
            return "", err
        }
        targetStatus = s.workflowSvc.NormalizeStatus(targetStatus)
    }
    return targetStatus, nil
}
```

### Test Code

Tests:
- `TestEntityService_TransitionStatus_ForcedWithReason` (entity_service_test.go:165)
- `TestEntityService_ValidateAndNormalize_ForcedSkipsValidation` (entity_service_test.go:369)
- `TestEpicService_TransitionStatus_Force` (epic_service_test.go)
- `TestFeatureService_TransitionStatus_Force` (feature_service_test.go)
- `TestTaskService_TransitionStatus_Forced_UsesStatusUpdateRaw` (task_service_test.go:2212)

### Test Execution Output

```
=== RUN   TestEpicService_ForceTransition_RequiresReason
--- PASS: TestEpicService_ForceTransition_RequiresReason (0.00s)
=== RUN   TestFeatureService_ForceTransition_RequiresReason
--- PASS: TestFeatureService_ForceTransition_RequiresReason (0.00s)
=== RUN   TestEntityService_TransitionStatus_ForcedWithReason
--- PASS: TestEntityService_TransitionStatus_ForcedWithReason (0.00s)
=== RUN   TestEntityService_TransitionStatus_ForcedWithoutReason
--- PASS: TestEntityService_TransitionStatus_ForcedWithoutReason (0.00s)
=== RUN   TestEntityService_ValidateAndNormalize_ForcedSkipsValidation
--- PASS: TestEntityService_ValidateAndNormalize_ForcedSkipsValidation (0.00s)
=== RUN   TestEntityService_DetectBackward_ForceGraceful
--- PASS: TestEntityService_DetectBackward_ForceGraceful (0.00s)
=== RUN   TestEpicService_TransitionStatus_Force
--- PASS: TestEpicService_TransitionStatus_Force (0.00s)
=== RUN   TestFeatureService_TransitionStatus_Force
--- PASS: TestFeatureService_TransitionStatus_Force (0.00s)
=== RUN   TestTaskService_TransitionStatus_Forced_UsesStatusUpdateRaw
--- PASS: TestTaskService_TransitionStatus_Forced_UsesStatusUpdateRaw (0.00s)
```

---

## Scenario 7: Forced Transition without Reason (Error)

### PRD Spec Quote

> **Scenario 7: Forced Transition without Reason (Error)**
> - Given any entity in any status
> - When `TransitionStatus(ctx, key, targetStatus, TransitionOptions{Force: true})` is called (no reason)
> - Then `ErrForceReasonRequired` is returned

### Implementation Code

**File: `internal/services/entity_service.go:123-126`**

```go
// Step 5: Enforce reason for forced transitions
if opts.Force && opts.Reason == "" {
    return nil, ErrForceReasonRequired
}
```

**File: `internal/services/transition_types.go:17`**

```go
ErrForceReasonRequired = errors.New("--force requires --reason to document why validation was bypassed")
```

TaskService also checks this at `task_service.go:619-621`:
```go
if opts.Force && opts.Reason == "" {
    return nil, ErrForceReasonRequired
}
```

### Test Code

- `TestEntityService_TransitionStatus_ForcedWithoutReason` (entity_service_test.go:194)
- `TestEpicService_ForceTransition_RequiresReason` (epic_service_test.go)
- `TestFeatureService_ForceTransition_RequiresReason` (feature_service_test.go)

### Test Execution Output

```
=== RUN   TestEntityService_TransitionStatus_ForcedWithoutReason
--- PASS: TestEntityService_TransitionStatus_ForcedWithoutReason (0.00s)
=== RUN   TestEpicService_ForceTransition_RequiresReason
--- PASS: TestEpicService_ForceTransition_RequiresReason (0.00s)
=== RUN   TestFeatureService_ForceTransition_RequiresReason
--- PASS: TestFeatureService_ForceTransition_RequiresReason (0.00s)
```

Note: Test pattern `"ForceReason|Force.*Required"` matched zero tests (the actual test names use different patterns). However, `"Force"` pattern captured all force-related tests including the "without reason" cases.

---

## Scenario 8: resolveAction Unification

### PRD Spec Quote

> **Scenario 8: resolveAction Unification**
> - Given EntityService is constructed with a workflow service that has status metadata with orchestrator actions
> - When `EntityService.ResolveAction(entity, "in_development", extraPlaceholders)` is called
> - Then base placeholders are generated from `EntityPlaceholders(entity)` via the Entity interface
> - And extra placeholders are merged over the base
> - And the orchestrator action instruction template is populated with all placeholders
> - And the resulting `PopulatedAction` is identical to pre-refactoring behavior

### Implementation Code

**File: `internal/services/entity_service.go:208-223`** (ResolveActionForStatus)

```go
func (s *EntityService) ResolveActionForStatus(status string, placeholders map[string]string) *config.PopulatedAction {
    wf := s.workflowSvc.GetWorkflow()
    if wf == nil || wf.StatusMetadata == nil {
        return nil
    }
    meta, exists := wf.StatusMetadata[status]
    if !exists || meta.OrchestratorAction == nil {
        return nil
    }
    return &config.PopulatedAction{
        Action:      meta.OrchestratorAction.Action,
        AgentType:   meta.OrchestratorAction.AgentType,
        Skills:      meta.OrchestratorAction.Skills,
        Instruction: meta.OrchestratorAction.PopulateTemplate(placeholders),
    }
}
```

**Entity-specific delegation examples:**

Epic (`epic_service.go:308`): `return s.entitySvc.ResolveActionForStatus(status, placeholders)`
Feature (`feature_service.go:312`): `return s.entitySvc.ResolveActionForStatus(status, placeholders)`
Task (`task_service.go:793`): `return s.entitySvc.ResolveActionForStatus(status, placeholders)`

**Inline workflow lookup check**: grep for `workflowSvc.GetWorkflow()`, `workflowSvc.ValidateTransition`, `workflowSvc.IsBackwardTransition`, `workflowSvc.NormalizeStatus` in epic_service.go and feature_service.go returned **no matches**. These services do not perform inline workflow lookups.

### Test Code

- `TestEntityService_ResolveActionForStatus_NilWorkflow` (entity_service_test.go:432)
- `TestEntityService_ResolveActionForStatus_ValidStatus` (entity_service_test.go:442)
- `TestEpicService_TransitionStatus_WithAction` (epic_service_test.go)
- `TestFeatureService_TransitionStatus_WithAction` (feature_service_test.go)
- `TestEpicService_resolveAction_StatusWithAction` (epic_service_test.go)
- `TestFeatureService_resolveAction_StatusWithAction` (feature_service_test.go)

### Test Execution Output

```
=== RUN   TestEntityService_TransitionStatus_WithResolveActionFn
--- PASS: TestEntityService_TransitionStatus_WithResolveActionFn (0.00s)
=== RUN   TestEntityService_ResolveActionForStatus_NilWorkflow
--- PASS: TestEntityService_ResolveActionForStatus_NilWorkflow (0.00s)
=== RUN   TestEntityService_ResolveActionForStatus_ValidStatus
--- PASS: TestEntityService_ResolveActionForStatus_ValidStatus (0.00s)
=== RUN   TestEntityService_GetNextStatus_WithResolveActionFn
--- PASS: TestEntityService_GetNextStatus_WithResolveActionFn (0.00s)
=== RUN   TestEpicService_TransitionStatus_WithAction
--- PASS: TestEpicService_TransitionStatus_WithAction (0.00s)
=== RUN   TestEpicService_GetNextStatus_WithActions
--- PASS: TestEpicService_GetNextStatus_WithActions (0.00s)
=== RUN   TestEpicService_resolveAction_StatusWithAction
--- PASS: TestEpicService_resolveAction_StatusWithAction (0.00s)
=== RUN   TestFeatureService_TransitionStatus_WithAction
--- PASS: TestFeatureService_TransitionStatus_WithAction (0.00s)
=== RUN   TestFeatureService_GetNextStatus_WithActions
--- PASS: TestFeatureService_GetNextStatus_WithActions (0.00s)
=== RUN   TestFeatureService_resolveAction_StatusWithAction
--- PASS: TestFeatureService_resolveAction_StatusWithAction (0.00s)
PASS
ok    github.com/jwwelbor/shark-task-manager/internal/services  0.010s
```

---

## Scenario 9: GetNextStatus Unification

### PRD Spec Quote

> **Scenario 9: GetNextStatus Unification**
> - Given an Epic in status "draft"
> - When `EpicService.GetNextStatus(ctx, "E21")` is called
> - Then the shared `EntityService.GetNextStatus` retrieves available transitions from the workflow service
> - And each transition includes an orchestrator action resolved via the entity-specific placeholder function

### Implementation Code

**File: `internal/services/entity_service.go:228-267`** (EntityService.GetNextStatus)

```go
func (s *EntityService) GetNextStatus(ctx context.Context, repo EntityRepository, entityType string, key string, resolveActionFn ResolveActionFn) (*NextStatusInfo, error) {
    entity, err := repo.GetByKey(ctx, key)
    // ...
    currentStatus := entity.GetStatus()
    transitions := s.workflowSvc.GetTransitionInfo(currentStatus)
    currentMeta := s.workflowSvc.GetStatusMetadata(currentStatus)

    wrapped := make([]TransitionInfoWithAction, 0, len(transitions))
    for _, t := range transitions {
        var action *config.PopulatedAction
        if resolveActionFn != nil {
            action = resolveActionFn(entity, t.TargetStatus)
        }
        wrapped = append(wrapped, TransitionInfoWithAction{...})
    }
    return &NextStatusInfo{...}, nil
}
```

**Epic delegation** (`epic_service.go:264-272`):
```go
func (s *EpicService) GetNextStatus(ctx context.Context, epicKey string) (*NextStatusInfo, error) {
    entityRepo := s.entityRepo
    if entityRepo == nil {
        entityRepo = &epicEntityRepoAdapter{repo: s.repo}
    }
    return s.entitySvc.GetNextStatus(ctx, entityRepo, "epic", epicKey, s.makeResolveActionFn(ctx))
}
```

**Feature delegation** (`feature_service.go:269-277`): identical pattern with `"feature"`.

**Task** (`task_service.go:691-717`): TaskService does NOT delegate to `entitySvc.GetNextStatus()`. Instead it uses `s.entitySvc.GetWorkflowService().GetTransitionInfo(currentStatus)` and `s.entitySvc.GetWorkflowService().IsTerminalStatus(currentStatus)` directly, while calling `s.resolveAction(ctx, task, t.TargetStatus)` for action resolution.

### Test Code

- `TestEntityService_GetNextStatus_HappyPath` (entity_service_test.go:459)
- `TestEntityService_GetNextStatus_EntityNotFound` (entity_service_test.go:486)
- `TestEntityService_GetNextStatus_TerminalStatus` (entity_service_test.go:501)
- `TestEntityService_GetNextStatus_WithResolveActionFn` (entity_service_test.go:519)
- `TestEpicService_GetNextStatus` (epic_service_test.go)
- `TestEpicService_GetNextStatus_Terminal` (epic_service_test.go)
- `TestEpicService_GetNextStatus_NotFound` (epic_service_test.go)
- `TestEpicService_GetNextStatus_WithActions` (epic_service_test.go)
- `TestFeatureService_GetNextStatus` (feature_service_test.go)
- `TestFeatureService_GetNextStatus_Terminal` (feature_service_test.go)
- `TestFeatureService_GetNextStatus_NotFound` (feature_service_test.go)
- `TestFeatureService_GetNextStatus_WithActions` (feature_service_test.go)

### Test Execution Output

```
=== RUN   TestEntityService_GetNextStatus_HappyPath
--- PASS: TestEntityService_GetNextStatus_HappyPath (0.00s)
=== RUN   TestEntityService_GetNextStatus_EntityNotFound
--- PASS: TestEntityService_GetNextStatus_EntityNotFound (0.00s)
=== RUN   TestEntityService_GetNextStatus_TerminalStatus
--- PASS: TestEntityService_GetNextStatus_TerminalStatus (0.00s)
=== RUN   TestEntityService_GetNextStatus_WithResolveActionFn
--- PASS: TestEntityService_GetNextStatus_WithResolveActionFn (0.00s)
=== RUN   TestEpicService_GetNextStatus
--- PASS: TestEpicService_GetNextStatus (0.00s)
=== RUN   TestEpicService_GetNextStatus_Terminal
--- PASS: TestEpicService_GetNextStatus_Terminal (0.00s)
=== RUN   TestEpicService_GetNextStatus_NotFound
--- PASS: TestEpicService_GetNextStatus_NotFound (0.00s)
=== RUN   TestEpicService_GetNextStatus_WithActions
--- PASS: TestEpicService_GetNextStatus_WithActions (0.00s)
=== RUN   TestFeatureService_GetNextStatus
--- PASS: TestFeatureService_GetNextStatus (0.00s)
=== RUN   TestFeatureService_GetNextStatus_Terminal
--- PASS: TestFeatureService_GetNextStatus_Terminal (0.00s)
=== RUN   TestFeatureService_GetNextStatus_NotFound
--- PASS: TestFeatureService_GetNextStatus_NotFound (0.00s)
=== RUN   TestFeatureService_GetNextStatus_WithActions
--- PASS: TestFeatureService_GetNextStatus_WithActions (0.00s)
PASS
ok    github.com/jwwelbor/shark-task-manager/internal/services  0.004s
```

---

## Scenario 10: Quality Gate

### PRD Spec Quote

> **Scenario 10: Quality Gate**
> - Given all F03 refactoring is complete
> - When `make fmt && make lint && make test` is run
> - Then there are zero formatting changes, zero lint warnings, and zero test failures

### Execution Output

**`make fmt`**:
```
Formatting code...
```
(No formatting changes needed)

**`make lint`**:
```
0 issues.
```

**`make test`** (all packages):
```
ok    github.com/jwwelbor/shark-task-manager/internal/cli              (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/cli/commands      (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/cli/scope         (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/config            (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/db                (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/dependency        (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/discovery         (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/fileops           (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/filepath          (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/formatters        (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/init              (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/keygen            (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/keys              (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/models            (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/parser            (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/pathresolver      (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/patterns          (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/reporting         (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/repository        0.854s
ok    github.com/jwwelbor/shark-task-manager/internal/services          (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/slug              (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/status            0.202s
ok    github.com/jwwelbor/shark-task-manager/internal/taskcreation      2.067s
ok    github.com/jwwelbor/shark-task-manager/internal/taskfile          (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/template          (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/templates         (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/utils             (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/validation        (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/view              (cached)
ok    github.com/jwwelbor/shark-task-manager/internal/workflow          (cached)
```

**Result**: Zero formatting changes, zero lint issues, zero test failures across all 30 packages. No `FAIL` lines.

---

## Additional Implementation Notes

### PRD Deviation: ResolveActionForStatus Signature

The PRD specified:
> `EntityService.ResolveAction(entity models.Entity, status string, extraPlaceholders map[string]string) *config.PopulatedAction`

Actual implementation signature (`entity_service.go:208`):
```go
func (s *EntityService) ResolveActionForStatus(status string, placeholders map[string]string) *config.PopulatedAction
```

The implementation uses a callback pattern (`ResolveActionFn`) where entity-specific services build their own placeholders and pass the complete map. The entity object is not passed to EntityService directly; instead, each entity service's `makeResolveActionFn` builds entity-specific placeholders externally and passes them in.

### PRD Deviation: TaskService GetNextStatus Delegation

The PRD specified (Story 5 AC):
> Entity-specific services delegate `GetNextStatus` calls to the shared implementation

Task implementation at `task_service.go:691-717` does NOT delegate to `entitySvc.GetNextStatus()`. Instead, TaskService uses `s.entitySvc.GetWorkflowService()` to access workflow methods directly and builds the `NextStatusInfo` inline. Epic and Feature do fully delegate.

### PRD Deviation: Rejection Note Creation

The PRD specified rejection note creation as part of EntityService.TransitionStatus. The actual implementation places rejection note creation in entity-specific post-hooks (epic_service.go:247-251, feature_service.go:252-256) rather than inside EntityService.TransitionStatus, because each entity type has its own note repository interface. EntityService sets `IsBackward` on the result to enable callers to handle this in their post-hooks.
