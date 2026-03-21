# E21-F09: Entity Service Delegation Completion -- Specification

**Feature**: Entity Service Delegation Completion
**Epic**: E21 (Entity Polymorphism and Duplication Reduction)
**Status**: In Specification
**Date**: 2026-03-21

---

## Context

See epic PRD `docs/plan/E21-entity-polymorphism-and-duplication-reduction/epic.md` for business context and motivation. See epic architecture doc `docs/plan/E21-entity-polymorphism-and-duplication-reduction/architecture-design.md` for system-level design decisions.

E21-F01 created `EntityService`, `EntityRegistry`, and `EntityRepository` adapters. E21-F08 unified the polymorphic data model. This feature completes the delegation story by making all entity services use the shared infrastructure.

### Current State Summary

| Component | Status | Detail |
|-----------|--------|--------|
| EpicService.TransitionStatus | DONE | Delegates to EntityService.TransitionStatus |
| FeatureService.TransitionStatus | DONE | Delegates to EntityService.TransitionStatus |
| TaskService.TransitionStatus | PARTIAL | Uses ValidateAndNormalize + DetectBackward but NOT full delegation (custom executeStatusTransition for auto-unblock) |
| BugService.SetBugStatus | NOT DONE | Inline workflow validation, no EntityService |
| ChangeCardService.SetChangeCardStatus | NOT DONE | Inline workflow validation, no EntityService |
| NoteService | DONE | Uses EntityRegistry |
| ContextService | DONE | Uses EntityRegistry |
| ResumeService | PARTIAL | Has EntityRegistry but still uses per-entity repo fields for entity-specific queries |
| Document operations | DUPLICATED | Each service has LinkDocument/UnlinkDocument/ListRelatedDocumentsByKey thin wrappers |
| Service accessor wiring | PARTIAL | EntityRegistry initialized once, but BugService/ChangeCardService wired without EntityService |

---

## Requirements

### Functional Requirements

**REQ-F-001: BugService EntityService Delegation**
- **Description**: BugService composes EntityService and delegates `SetBugStatus` and `AdvanceBugStatus` transition logic to `EntityService.TransitionStatus` using `SimpleTransitionFeatures()`.
- **Priority**: Must-Have
- **Traces to**: Epic PRD Section "Service Layer Duplication"
- **Acceptance Criteria**:
  - BugService struct has an `entitySvc *EntityService` field
  - BugService struct has an `entityRepo EntityRepository` field
  - `SetBugStatus` delegates to `entitySvc.TransitionStatus` instead of inline workflow validation
  - `AdvanceBugStatus` uses `entitySvc.GetNextStatus` to determine next status, then calls `entitySvc.TransitionStatus`
  - `resolveAction` is replaced by a `makeResolveActionFn` callback passed to EntityService
  - History recording is handled by EntityService (remove `recordBugHistory` calls)
  - All existing BugService tests pass
  - BugService constructor accepts `*EntityService` and `EntityRepository` parameters

**REQ-F-002: ChangeCardService EntityService Delegation**
- **Description**: ChangeCardService composes EntityService and delegates `SetChangeCardStatus` and `AdvanceChangeCardStatus` transition logic to `EntityService.TransitionStatus` using `SimpleTransitionFeatures()`.
- **Priority**: Must-Have
- **Traces to**: Epic PRD Section "Service Layer Duplication"
- **Acceptance Criteria**:
  - ChangeCardService struct has an `entitySvc *EntityService` field
  - ChangeCardService struct has an `entityRepo EntityRepository` field
  - `SetChangeCardStatus` delegates to `entitySvc.TransitionStatus` instead of inline workflow validation
  - `AdvanceChangeCardStatus` uses `entitySvc.GetNextStatus` to determine next status, then calls `entitySvc.TransitionStatus`
  - `resolveAction` is replaced by a `makeResolveActionFn` callback
  - History recording is handled by EntityService (remove `recordChangeCardHistory` calls)
  - All existing ChangeCardService tests pass
  - ChangeCardService constructor accepts `*EntityService` and `EntityRepository` parameters

**REQ-F-003: TaskService Full TransitionStatus Delegation**
- **Description**: TaskService delegates the full TransitionStatus flow to EntityService instead of calling individual helper methods (ValidateAndNormalize, DetectBackward). Task-specific logic (auto-unblock, StatusUpdateRaw) is handled via a post-hook or custom EntityRepository adapter.
- **Priority**: Must-Have
- **Traces to**: Epic PRD Section "Service Layer Duplication"
- **Acceptance Criteria**:
  - TaskService.TransitionStatus calls `entitySvc.TransitionStatus` with the task's EntityRepository adapter
  - Task-specific auto-unblock logic runs as a post-hook after `entitySvc.TransitionStatus` returns
  - `ErrForceReasonRequired` check is removed from TaskService (EntityService handles it)
  - Entity history recording is handled by EntityService (remove explicit `recordEntityHistory` call)
  - The `resolveAction` helper is replaced by a `makeResolveActionFn` callback
  - All existing TaskService transition tests pass
  - Auto-unblock behavior is preserved

**REQ-F-004: TaskService GetNextStatus Delegation**
- **Description**: TaskService.GetNextStatus delegates to EntityService.GetNextStatus instead of duplicating the logic.
- **Priority**: Must-Have
- **Traces to**: Epic PRD Section "3x GetNextStatus"
- **Acceptance Criteria**:
  - TaskService.GetNextStatus calls `entitySvc.GetNextStatus` with the task's EntityRepository adapter and a `makeResolveActionFn` callback
  - The inline implementation (GetWorkflowService().GetTransitionInfo, wrapping transitions) is removed
  - All existing GetNextStatus tests pass

**REQ-F-005: ResumeService Per-Entity Repo Field Reduction**
- **Description**: Reduce per-entity repository fields in ResumeService where EntityRegistry can replace them.
- **Priority**: Should-Have
- **Traces to**: Epic PRD Section "Cross-Cutting Service Duplication"
- **Acceptance Criteria**:
  - `GetBugResume` and `GetChangeResume` continue to use EntityRegistry (already done)
  - `GetEpicResume`, `GetFeatureResume`, `GetTaskResume` entity lookup uses EntityRegistry where possible
  - Per-entity repos retained ONLY for entity-specific queries not available on EntityRepository (e.g., `GetContextData`, `ListByEpic`, `ListByFeature`, `ListByEpic` on tasks)
  - Constructor signature simplified where repos can be removed
  - All existing ResumeService tests pass

**REQ-F-006: Service Accessor Wiring for BugService and ChangeCardService**
- **Description**: Update `service_accessors.go` and `services_global.go` to pass EntityService and EntityRepository to BugService and ChangeCardService constructors.
- **Priority**: Must-Have
- **Traces to**: Epic PRD Section "Service Accessor Simplification"
- **Acceptance Criteria**:
  - `GetBugService()` creates EntityService, gets EntityRepository from EntityRegistry, passes both to BugService constructor
  - `GetChangeCardService()` creates EntityService, gets EntityRepository from EntityRegistry, passes both to ChangeCardService constructor
  - `cmd/server/services.go` WireServices updated similarly
  - BugService and ChangeCardService no longer need separate `SetEntityHistoryRepo` calls (EntityService handles history)

### Non-Functional Requirements

**REQ-NF-001: Behavioral Equivalence**
- **Description**: All status transitions, note operations, context operations, history recording, and orchestrator action resolution must produce identical results before and after.
- **Measurement**: Full existing test suite passes without modification (except test setup changes for new constructor signatures)
- **Target**: Zero behavioral changes

**REQ-NF-002: No Performance Regression**
- **Description**: Delegation should not add measurable overhead. EntityService.TransitionStatus is already called by EpicService and FeatureService with no issues.
- **Measurement**: No additional database queries per transition beyond existing pattern

---

## Out of Scope

1. **Merging entity services into one** -- Entity-specific logic (Task auto-unblock, Epic cascade, Bug triage) justifies separate services. See feature.md "Out of Scope".
2. **CLI command consolidation** -- Addressed separately in E21-F10.
3. **Document operation deduplication** -- The thin wrappers (LinkDocument/UnlinkDocument/ListRelatedDocumentsByKey) in each entity service are 3-5 lines each and serve as a typed API boundary. Consolidating them would require a generic interface that adds complexity without meaningful reduction. Excluded from this feature.
4. **AdvanceBugStatus/AdvanceChangeCardStatus return type change** -- These currently return `*models.Bug`/`*models.ChangeCard` not `*TransitionResult`. Changing the return type would be a public API change affecting CLI commands. Keep existing return types and convert internally.

---

## Architecture

### Key Technical Decisions

**Decision 1: BugService/ChangeCardService use EntityService composition (same pattern as EpicService/FeatureService)**

Rationale: EpicService and FeatureService already demonstrate this pattern successfully. BugService and ChangeCardService should follow the same approach for consistency. See `internal/services/epic_service.go` lines 82-95 for the established pattern.

**Decision 2: TaskService custom EntityRepository adapter handles StatusUpdateRaw**

Rationale: TaskService uses `executeStatusTransition` which calls `StatusUpdateRaw` (an atomic update with auto-unblock). The standard `EntityRepository.UpdateStatus` does a simple status field update. Two approaches considered:

- Option A: Create a custom EntityRepository adapter for tasks that calls `StatusUpdateRaw` instead of `UpdateStatus`. This would let EntityService.TransitionStatus handle the full flow, but the adapter's `UpdateStatus` would have side effects (auto-unblock) that are invisible to EntityService.
- **Option B (selected)**: Keep TaskService calling EntityService.TransitionStatus but with a custom EntityRepository adapter whose `UpdateStatus` calls the standard update. Task-specific auto-unblock runs as a post-hook after EntityService returns. This preserves the clean separation where EntityService handles validation/backward-detection/notes and TaskService handles task-specific side effects.

This matches the existing Epic/Feature pattern where post-hooks (child counting) run after EntityService.TransitionStatus returns.

**Decision 3: ResumeService retains per-entity repos for entity-specific queries**

Rationale: EntityRepository interface only exposes `GetByKey`, `GetByID`, `UpdateStatus`, `GetContextData`. ResumeService needs `ListByEpic`, `ListByFeature`, `GetContextData` which require typed repositories. The EntityRegistry lookup replaces the initial entity retrieval, but the aggregate queries require typed repos. Partial simplification is possible for initial entity lookup in GetEpicResume/GetFeatureResume/GetTaskResume.

**Decision 4: BugService/ChangeCardService AdvanceStatus methods convert TransitionResult internally**

Rationale: `AdvanceBugStatus` returns `*models.Bug` and `AdvanceChangeCardStatus` returns `*models.ChangeCard`. These are public APIs consumed by CLI commands. Rather than changing the return type (breaking change), the methods will call EntityService.TransitionStatus internally and then re-fetch the entity to return the typed model.

### Component Changes

#### Files to Modify

| File | Change | Lines (est.) |
|------|--------|-------------|
| `internal/services/bug_service.go` | Add entitySvc/entityRepo fields, refactor SetBugStatus/AdvanceBugStatus to delegate, add makeResolveActionFn, remove recordBugHistory, update constructor | ~80 lines changed |
| `internal/services/change_card_service.go` | Add entitySvc/entityRepo fields, refactor SetChangeCardStatus/AdvanceChangeCardStatus to delegate, add makeResolveActionFn, remove recordChangeCardHistory, update constructor | ~80 lines changed |
| `internal/services/task_service.go` | Refactor TransitionStatus to delegate full flow to EntityService, refactor GetNextStatus to delegate, add makeResolveActionFn, remove inline resolveAction | ~60 lines changed |
| `internal/services/resume_service.go` | Use EntityRegistry for initial entity lookup in GetEpicResume/GetFeatureResume/GetTaskResume, potentially remove redundant repo fields | ~30 lines changed |
| `internal/cli/services_global.go` | Update GetBugService and GetChangeCardService to pass EntityService and EntityRepository | ~20 lines changed |
| `internal/cli/service_accessors.go` | Update GetBugService and GetChangeCardService accessor wiring | ~10 lines changed |
| `cmd/server/services.go` | Update WireServices for BugService and ChangeCardService if applicable | ~10 lines changed |

#### Test Files to Modify

| File | Change |
|------|--------|
| `internal/services/bug_service_test.go` | Update constructor calls to pass EntityService mock, update SetBugStatus/AdvanceBugStatus test expectations |
| `internal/services/change_card_service_test.go` | Update constructor calls to pass EntityService mock, update status transition test expectations |
| `internal/services/task_service_test.go` | Update TransitionStatus and GetNextStatus tests to verify delegation pattern |
| `internal/services/resume_service_test.go` | Update if constructor signature changes |

### Detailed Design

#### BugService Refactoring

**Current constructor** (`internal/services/bug_service.go` line 62):
```go
func NewBugService(
    repo BugRepository,
    workflowSvc *workflow.Service,
    epicRepo LinkValidatorEpicRepo,
    featureRepo LinkValidatorFeatureRepo,
    taskRepo LinkValidatorTaskRepo,
    projectRoot string,
) *BugService
```

**New constructor**:
```go
func NewBugService(
    repo BugRepository,
    entitySvc *EntityService,
    entityRepo EntityRepository,
    epicRepo LinkValidatorEpicRepo,
    featureRepo LinkValidatorFeatureRepo,
    taskRepo LinkValidatorTaskRepo,
    projectRoot string,
) *BugService
```

**SetBugStatus delegation pattern** (replaces inline logic at lines 320-345):
```go
func (s *BugService) SetBugStatus(ctx context.Context, key string, status string, force bool) (*models.Bug, error) {
    opts := TransitionOptions{Force: force}
    _, err := s.entitySvc.TransitionStatus(
        ctx, s.entityRepo, models.EntityTypeBug, key, status, opts,
        SimpleTransitionFeatures(),
        s.makeResolveActionFn(),
    )
    if err != nil {
        return nil, err
    }
    // Re-fetch to return typed model
    return s.repo.GetByKey(ctx, key)
}
```

**AdvanceBugStatus delegation pattern** (replaces inline logic at lines 297-317):
```go
func (s *BugService) AdvanceBugStatus(ctx context.Context, key string) (*models.Bug, error) {
    info, err := s.entitySvc.GetNextStatus(
        ctx, s.entityRepo, models.EntityTypeBug, key,
        s.makeResolveActionFn(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get next status for bug %s: %w", key, err)
    }
    if len(info.AvailableTransitions) == 0 {
        return nil, fmt.Errorf("cannot advance bug %s: no valid transitions from status %q", key, info.CurrentStatus)
    }
    nextStatus := info.AvailableTransitions[0].TargetStatus
    return s.SetBugStatus(ctx, key, nextStatus, false)
}
```

**makeResolveActionFn** (replaces `resolveAction` at lines 379-398):
```go
func (s *BugService) makeResolveActionFn() ResolveActionFn {
    return func(entity models.Entity, status string) *config.PopulatedAction {
        bug, ok := entity.(*models.Bug)
        if !ok {
            return nil
        }
        placeholders := config.BugPlaceholders(bug)
        return s.entitySvc.ResolveActionForStatus(status, placeholders)
    }
}
```

**Removals**:
- `resolveAction` method (replaced by `makeResolveActionFn`)
- `recordBugHistory` method (EntityService handles history via SetHistoryRepo)
- `entityHistoryRepo` field and `SetEntityHistoryRepo` setter (EntityService owns history)
- Direct `workflowSvc` field usage for transition validation (EntityService handles it)

Note: `workflowSvc` field is retained for `GetValidTransitions`, `ValidateStatus` in `TriageBug`, and `CreateBug` default status.

#### ChangeCardService Refactoring

Follows identical pattern to BugService. See BugService section above -- same structural changes apply.

**Current constructor** (`internal/services/change_card_service.go` line 57):
```go
func NewChangeCardService(
    repo ChangeCardRepository,
    workflowSvc *workflow.Service,
    epicRepo ChangeCardEpicRepo,
    featureRepo ChangeCardFeatureRepo,
    projectRoot string,
) *ChangeCardService
```

**New constructor**:
```go
func NewChangeCardService(
    repo ChangeCardRepository,
    entitySvc *EntityService,
    entityRepo EntityRepository,
    epicRepo ChangeCardEpicRepo,
    featureRepo ChangeCardFeatureRepo,
    projectRoot string,
) *ChangeCardService
```

**Removals**: Same as BugService (`resolveAction`, `recordChangeCardHistory`, `entityHistoryRepo`, `SetEntityHistoryRepo`).

Note: `workflowSvc` field is retained for `GetValidTransitions` and `CreateChangeCard` default status.

#### TaskService TransitionStatus Refactoring

**Current flow** (`internal/services/task_service.go` lines 583-662):
1. Get task from typed repo
2. Call `entitySvc.ValidateAndNormalize`
3. Check `ErrForceReasonRequired` inline
4. Call `entitySvc.DetectBackward`
5. Call `executeStatusTransition` (task-specific atomic update with auto-unblock)
6. Call `recordEntityHistory` explicitly
7. Build `TransitionResult` inline
8. Call `resolveAction` inline

**New flow**:
1. Delegate to `entitySvc.TransitionStatus` with task's EntityRepository adapter
2. Post-hook: run auto-unblock logic via `s.executeAutoUnblock` if the transition succeeded
3. Merge auto-unblock results into the TransitionResult.Message

The challenge is that TaskService's EntityRepository adapter currently uses the standard `UpdateStatus` which just sets the status column. The `executeStatusTransition` method does more (StatusUpdateRaw with agent, reason, etc. plus auto-unblock).

**Solution**: Split TaskService's transition into two phases:
1. EntityService handles validation, backward detection, status update, history, rejection notes using the standard EntityRepository adapter (which calls simple `UpdateStatus` on the task)
2. TaskService post-hook handles: (a) storing agent/reason/documentPath on the task record via a supplementary repo call, and (b) auto-unblock processing

This requires adding a method to TaskRepository or extending the entity adapter's UpdateStatus to also set agent/reason fields. The simplest approach is to create a `taskEntityRepoAdapter` whose `UpdateStatus` calls `StatusUpdateRaw` instead.

```go
type taskEntityRepoAdapter struct {
    repo TaskServiceRepository
    opts *statusTransitionOpts // set before each TransitionStatus call
}

func (a *taskEntityRepoAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
    return a.repo.StatusUpdateRaw(ctx, id, models.TaskStatus(status),
        a.opts.agent, a.opts.reason, nil, a.opts.documentPath, a.opts.force)
}
```

Then TaskService.TransitionStatus becomes:
```go
func (s *TaskService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
    // Prepare adapter with transition-specific options
    adapter := s.makeTaskEntityAdapter(opts)

    result, err := s.entitySvc.TransitionStatus(
        ctx, adapter, models.EntityTypeTask, key, targetStatus, opts,
        DefaultTransitionFeatures(),
        s.makeResolveActionFn(ctx),
    )
    if err != nil {
        return nil, err
    }

    // Post-hook: auto-unblock dependents
    unblockedKeys := s.autoUnblockDependents(ctx, key, targetStatus)
    if len(unblockedKeys) > 0 {
        result.Message = fmt.Sprintf("Transitioned: %s -> %s (auto-unblocked: %s)",
            result.FromStatus, result.ToStatus, strings.Join(unblockedKeys, ", "))
    }

    return result, nil
}
```

**GetNextStatus delegation**:
```go
func (s *TaskService) GetNextStatus(ctx context.Context, key string) (*NextStatusInfo, error) {
    return s.entitySvc.GetNextStatus(ctx, s.entityRepo, models.EntityTypeTask, key,
        s.makeResolveActionFn(ctx))
}
```

**Removals**:
- Inline `ValidateAndNormalize` + `DetectBackward` + `ErrForceReasonRequired` check (EntityService handles these)
- Inline `recordEntityHistory` call (EntityService handles it)
- Inline `TransitionResult` construction (EntityService returns it)
- `resolveAction` private method (replaced by `makeResolveActionFn`)
- Duplicated transition info wrapping in `GetNextStatus`

#### ResumeService Simplification

**Current state**: GetBugResume and GetChangeResume already use EntityRegistry. GetEpicResume, GetFeatureResume, GetTaskResume use typed repos.

**Change**: For GetEpicResume/GetFeatureResume/GetTaskResume, the initial entity lookup (GetByKey) can use EntityRegistry. However, the subsequent calls (`GetContextData`, `ListByEpic`, `ListByFeature`) require typed interfaces not available on EntityRepository.

**Practical impact**: Minimal reduction. The typed repos are needed for aggregate queries. The constructor cannot be simplified because those repos are genuinely required. The only change is to add `registry.GetRepository` as an alternative lookup path for the initial GetByKey, but this adds complexity without meaningful benefit since the typed repos already have GetByKey.

**Decision**: Leave ResumeService as-is for this feature. The per-entity repos are justified by entity-specific query needs. The switch-statement duplication was already eliminated (GetBugResume/GetChangeResume use registry).

#### Service Accessor Wiring

**`internal/cli/services_global.go` GetBugService** (lines 372-394):
```go
func GetBugService() *services.BugService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    workflowSvc := GetWorkflowService()
    bugRepo := repository.NewBugRepository(db)
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)

    projectRoot, _ := FindProjectRoot()
    if projectRoot == "" {
        projectRoot = "."
    }

    entitySvc := GetEntityService()
    entityRepo := GetEntityRegistry().MustGetRepository(models.EntityTypeBug)

    svc := services.NewBugService(bugRepo, entitySvc, entityRepo, epicRepo, featureRepo, taskRepo, projectRoot)
    docRepo := repository.NewDocumentRepository(db)
    entityDocRepo := repository.NewEntityDocumentRepository(db)
    svc.SetWritableDocRepo(docRepo, entityDocRepo)
    // No longer need: svc.SetEntityHistoryRepo(...) -- EntityService handles it
    return svc
}
```

Same pattern for `GetChangeCardService`.

**`cmd/server/services.go`**: If WireServices constructs BugService/ChangeCardService, update similarly.

### Integration Points

| Integration Point | Impact |
|-------------------|--------|
| `internal/cli/commands/bug.go` | No changes needed -- calls BugService methods which maintain same signatures |
| `internal/cli/commands/change.go` | No changes needed -- calls ChangeCardService methods which maintain same signatures |
| `internal/cli/commands/status_group.go` | No changes needed -- calls entity service TransitionStatus/GetNextStatus which maintain same signatures |
| `internal/cli/commands/task.go` | No changes needed -- calls TaskService methods which maintain same signatures |
| `cmd/server/services.go` | Update WireServices for new BugService/ChangeCardService constructors |

### Data Model Changes

None. No schema changes. No migrations.

### API/Interface Changes

**Constructor signature changes** (breaking for direct callers):
- `NewBugService`: adds `*EntityService` and `EntityRepository` parameters
- `NewChangeCardService`: adds `*EntityService` and `EntityRepository` parameters

**Method signature changes**: None. All public method signatures remain identical.

**Removed public methods**:
- `BugService.SetEntityHistoryRepo` -- no longer needed
- `ChangeCardService.SetEntityHistoryRepo` -- no longer needed

---

## Test Strategy

### Unit Tests (mocked repos)

1. **BugService transition delegation**: Verify that SetBugStatus calls EntityService.TransitionStatus with correct parameters (SimpleTransitionFeatures, correct entityType, correct key)
2. **BugService advance delegation**: Verify AdvanceBugStatus calls GetNextStatus then SetBugStatus
3. **ChangeCardService transition delegation**: Same as BugService
4. **TaskService full delegation**: Verify TransitionStatus calls EntityService.TransitionStatus and that auto-unblock post-hook still runs
5. **TaskService GetNextStatus delegation**: Verify delegation to EntityService.GetNextStatus

### Integration Tests

1. Run `make test` -- all existing tests must pass
2. Bug/ChangeCard status transitions produce identical results (same statuses, same history records, same orchestrator actions)
3. Task transitions with auto-unblock still work correctly

### Regression Tests

No new regression tests needed -- existing test suite covers all behaviors. Test setup may need updates for new constructor signatures.

---

## Implementation Order

1. **BugService refactoring** (REQ-F-001) -- Simplest case, establishes pattern
2. **ChangeCardService refactoring** (REQ-F-002) -- Same pattern as BugService
3. **Service accessor wiring** (REQ-F-006) -- Wire EntityService into Bug/ChangeCard constructors
4. **TaskService TransitionStatus delegation** (REQ-F-003) -- Most complex, requires custom adapter
5. **TaskService GetNextStatus delegation** (REQ-F-004) -- Simple delegation
6. **ResumeService assessment** (REQ-F-005) -- Evaluate if any simplification is practical

---

*Last Updated*: 2026-03-21
