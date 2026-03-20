# F03: Status Transition Unification -- Implementation Architecture

**Feature**: E21-F03
**Author**: Architect
**Date**: 2026-03-20
**Status**: Approved
**Tier**: STANDARD
**Depends on**: F01 (Entity Interface Foundation) -- must be complete and merged

This document is the developer implementation guide for F03. It specifies how
`TransitionStatus`, `resolveAction`, and `GetNextStatus` are unified into a
shared `EntityService`, and how EpicService, FeatureService, and TaskService
delegate to it while retaining entity-specific behavior.

---

## 1. Scope and Constraints

**In scope:**
- Create `EntityService` struct in `internal/services/entity_service.go` with:
  - `TransitionStatus` -- shared transition algorithm controlled by `TransitionFeatures`
  - `ResolveAction` -- shared orchestrator action resolution with extension points
  - `GetNextStatus` -- shared available-transitions query with action resolution callback
  - `ValidateAndNormalize` -- exported shared validation/normalization helper for TaskService hybrid use
  - `DetectBackward` -- exported shared backward detection helper for TaskService hybrid use
- Refactor EpicService to delegate `TransitionStatus`, `GetNextStatus`, and `resolveAction` to EntityService
- Refactor FeatureService to delegate `TransitionStatus`, `GetNextStatus`, and `resolveAction` to EntityService
- Refactor TaskService to use hybrid delegation (shared validation helpers + task-specific `StatusUpdateRaw`)
- Update constructors for all 3 core services to accept `*EntityService`
- Update `services_global.go` to create and wire `EntityService`
- (Should-Have) Refactor Bug/ChangeCard to delegate via `SimpleTransitionFeatures()`
- Update all tests for new constructor signatures

**Out of scope:**
- Cross-cutting service refactoring (F02 -- NoteService, ContextService, ResumeService)
- Document operations unification (F04)
- Template placeholder unification (F05 -- entity-specific enrichment data)
- CLI command changes (method signatures are unchanged)
- Database schema changes
- `StatusUpdateRaw` refactoring (task-specific, retained as-is)

---

## 2. Key Architecture Decisions

### ADR-F03-001: EntityService Owns Shared Transition Logic

**Decision**: Create `EntityService` with a `TransitionStatus` method that implements
the 10-step transition algorithm once. Entity-specific services compose `EntityService`
and delegate to it with `TransitionFeatures` controlling opt-in/opt-out of features.

**Rationale**: Epic and Feature `TransitionStatus` methods are structurally identical
(~80 lines each), differing only in entity type string and child-counting logic.
Having one implementation eliminates the current 3-way duplication (Epic, Feature, Task)
and the 5-way duplication of `resolveAction`.

**Consequence**: Bug fixes to transition validation, backward detection, and rejection
note creation apply once instead of 3-5 times.

### ADR-F03-002: TaskService Uses Hybrid Delegation (Not Full Delegation)

**Decision**: TaskService reuses shared validation/normalization and backward detection
helpers from EntityService but retains its own `executeStatusTransition` method that calls
`StatusUpdateRaw` directly. TaskService does NOT route status updates through
`EntityRepository.UpdateStatus`.

**Rationale**: TaskService's `StatusUpdateRaw` performs an atomic DB operation including
agent tracking, notes recording, timestamp management (`started_at`, `completed_at`,
`blocked_at`), auto-unblocking of dependent tasks, and session tracking. This is
fundamentally different from Epic/Feature's `repo.UpdateStatus(id, status)`. Routing
through the adapter would require replicating `StatusUpdateRaw` behavior in the adapter,
which would be a leaky abstraction.

**Consequence**: TaskService's `TransitionStatus` is longer (~35 lines) than
Epic/Feature's (~15 lines), but still eliminates ~30 lines of duplicated validation,
backward detection, and action resolution logic from the current ~60-line method.

### ADR-F03-003: TransitionFeatures Config Struct (Not Interface Methods)

**Decision**: Use a `TransitionFeatures` struct with boolean fields to control which
optional steps are active, with `DefaultTransitionFeatures()` and
`SimpleTransitionFeatures()` preset functions.

**Rationale**: Explicit boolean configuration is easier to understand and debug than
implicit interface methods like `SupportsBackward() bool`. The struct makes it clear
at the call site which features are active.

### ADR-F03-004: resolveAction Delegates to Entity-Specific Placeholder Functions

**Decision**: The shared `ResolveAction` method on EntityService handles the workflow
lookup, nil checks, and `PopulatedAction` construction. Entity-specific services pass
a `ResolveActionFn` callback that generates the full placeholder map, including
entity-specific enrichment data.

**Rationale**: The current `resolveAction` implementations have an identical structure
(workflow lookup, nil checks, action construction) but differ in how placeholders are
generated. Each entity type has different enrichment data, document repositories, and
relationship repositories. Rather than trying to abstract these differences (which would
require the Entity interface to expose enrichment data), we let each service provide its
own placeholder function. This gives us the structural deduplication without forcing
enrichment data unification (which is F05's concern).

**Alternative Rejected**: Using `EntityPlaceholders(entity)` as the base and merging
`extraPlaceholders`. This was proposed in the epic architecture but would lose the
enrichment data path that all 3 services currently use. The callback approach preserves
full backward compatibility.

### ADR-F03-005: EntityService Depends on workflow.Service Only (Not noteRepo)

**Decision**: `EntityService` receives `*workflow.Service` in its constructor. For
rejection note creation, entity-specific services pass their own `noteRepo` reference
via a post-hook rather than having `EntityService` own a `noteRepo`.

**Rationale**: The current rejection note creation uses different interfaces across
entity types (TaskService uses `TaskNoteRepository` with `*models.EntityNote` return;
EpicService uses `EpicNoteRepository` with string-based `documentPath`). Unifying
the note interface would require changes to the `TaskNoteRepository` interface
signature. Instead, `EntityService.TransitionStatus` returns `isBackward` and
`opts.Reason` in the result, and the calling service handles rejection note creation
with its own typed note repository. This is simpler and avoids interface changes.

**Updated from epic architecture**: The epic architecture proposed `EntityService`
owning a `RejectionNoteCreator` interface. After codebase inspection, the note repo
interfaces differ enough that delegation is cleaner. The calling service already has
the note repo and can create the rejection note in 3 lines of post-hook code.

---

## 3. EntityService Design

### 3.1 File: `internal/services/entity_service.go`

```go
package services

import (
    "context"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TransitionFeatures controls which optional steps of the TransitionStatus
// algorithm are active for a given entity type.
type TransitionFeatures struct {
    // DetectBackward enables backward transition detection and reason requirement.
    // Set to true for Epic, Feature, Task.
    // Set to false for Bug, ChangeCard.
    DetectBackward bool

    // CreateRejectionNotes signals that the caller will handle rejection note
    // creation in its post-hook. EntityService sets IsBackward on the result
    // to enable this. (Note: EntityService does NOT create the note itself.)
    CreateRejectionNotes bool

    // ResolveOrchestratorAction enables orchestrator action resolution.
    // Set to true for all entity types.
    ResolveOrchestratorAction bool
}

// DefaultTransitionFeatures returns the full feature set used by Epic, Feature, and Task.
func DefaultTransitionFeatures() TransitionFeatures {
    return TransitionFeatures{
        DetectBackward:            true,
        CreateRejectionNotes:      true,
        ResolveOrchestratorAction: true,
    }
}

// SimpleTransitionFeatures returns the reduced feature set used by Bug and ChangeCard.
func SimpleTransitionFeatures() TransitionFeatures {
    return TransitionFeatures{
        DetectBackward:            false,
        CreateRejectionNotes:      false,
        ResolveOrchestratorAction: true,
    }
}

// ResolveActionFn is a callback that generates a PopulatedAction for a given
// entity and target status. Entity-specific services provide this to include
// their enrichment data, document repos, and relationship repos in the
// placeholder generation.
type ResolveActionFn func(entity models.Entity, status string) *config.PopulatedAction

// EntityService provides shared status transition logic for all entity types.
// Entity-specific services compose this and delegate shared steps to it.
type EntityService struct {
    workflowSvc *workflow.Service
}

// NewEntityService creates an EntityService with the workflow service dependency.
func NewEntityService(workflowSvc *workflow.Service) *EntityService {
    requireNonNil(workflowSvc, "EntityService requires a non-nil workflow.Service")
    return &EntityService{
        workflowSvc: workflowSvc,
    }
}
```

### 3.2 TransitionStatus Method

This is the shared 10-step algorithm that replaces the duplicated ~80-line methods
in EpicService and FeatureService.

```go
// TransitionStatus performs a status transition on any entity via its
// EntityRepository adapter. The features parameter controls which optional
// steps are active.
//
// Steps performed:
//  1. Get entity by key via repo
//  2. Extract current status
//  3. Validate transition (unless forced)
//  4. Normalize target status (unless forced)
//  5. Enforce reason requirement for forced transitions
//  6. [opt-in] Detect backward transition and require reason
//  7. Update entity status via repo.UpdateStatus
//  8. Resolve orchestrator action (opt-in, via resolveActionFn)
//  9. Build and return TransitionResult
//
// Note: Rejection note creation is NOT performed by EntityService.
// The result includes IsBackward so the caller can create notes in a post-hook.
func (s *EntityService) TransitionStatus(
    ctx context.Context,
    repo EntityRepository,
    entityType string,
    key string,
    targetStatus string,
    opts TransitionOptions,
    features TransitionFeatures,
    resolveActionFn ResolveActionFn,
) (*TransitionResult, error) {
    // Step 1: Get entity
    entity, err := repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get %s: %w", entityType, err)
    }
    if entity == nil {
        return nil, fmt.Errorf("%s not found: %s", entityType, key)
    }

    currentStatus := entity.GetStatus()

    // Steps 3-4: Validate and normalize
    targetStatus, err = s.ValidateAndNormalize(currentStatus, targetStatus, opts.Force)
    if err != nil {
        return nil, err
    }

    // Step 5: Enforce reason for forced transitions
    if opts.Force && opts.Reason == "" {
        return nil, ErrForceReasonRequired
    }

    // Step 6: Backward detection (opt-in)
    var isBackward bool
    if features.DetectBackward {
        isBackward, err = s.DetectBackward(currentStatus, targetStatus, opts.Force, opts.Reason)
        if err != nil {
            return nil, err
        }
    }

    // Step 7: Update status via repo
    if err := repo.UpdateStatus(ctx, entity.GetID(), targetStatus); err != nil {
        return nil, fmt.Errorf("failed to update %s status: %w", entityType, err)
    }

    // Step 8: Resolve orchestrator action (opt-in)
    var action *config.PopulatedAction
    if features.ResolveOrchestratorAction && resolveActionFn != nil {
        action = resolveActionFn(entity, targetStatus)
    }

    // Step 9: Build result
    return &TransitionResult{
        EntityType:         entityType,
        EntityKey:          key,
        FromStatus:         currentStatus,
        ToStatus:           targetStatus,
        Transitioned:       true,
        OrchestratorAction: action,
        IsBackward:         isBackward,
        IsForced:           opts.Force,
        Reason:             opts.Reason,
        // ChildCount is set by the calling entity service in its post-hook
    }, nil
}
```

### 3.3 Exported Helper Methods for TaskService Hybrid Use

TaskService cannot use the full `TransitionStatus` method because it has its own
`StatusUpdateRaw` call path. Instead, it calls these exported helpers to reuse the
shared validation and backward detection logic.

```go
// ValidateAndNormalize validates a transition and normalizes the target status.
// If force is true, validation is skipped and the target is returned unchanged.
// Returns the (possibly normalized) target status and any validation error.
func (s *EntityService) ValidateAndNormalize(currentStatus, targetStatus string, force bool) (string, error) {
    if !force {
        if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
            return "", err
        }
        targetStatus = s.workflowSvc.NormalizeStatus(targetStatus)
    }
    return targetStatus, nil
}

// DetectBackward checks if a transition is backward and enforces reason requirements.
// Returns isBackward flag and any error (BackwardReasonError if reason missing).
// If force is true and IsBackwardTransition errors, isBackward is set to false (graceful).
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

// GetWorkflowService returns the underlying workflow service.
// Used by entity-specific services that need direct access for
// ValidateStatus, IsTerminalStatus, GetTransitionInfo, etc.
func (s *EntityService) GetWorkflowService() *workflow.Service {
    return s.workflowSvc
}
```

### 3.4 ResolveAction Shared Helper

This replaces the structurally identical `resolveAction` methods across 5 services.
The caller provides a `placeholderFn` that includes entity-specific enrichment.

```go
// ResolveActionForStatus resolves the orchestrator action for a target status.
// Uses the provided placeholders map to populate the action instruction template.
// Returns nil gracefully for nil workflow, missing metadata, or nil OrchestratorAction.
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

### 3.5 GetNextStatus Shared Method

```go
// GetNextStatus returns available status transitions for an entity.
// The resolveActionFn callback generates entity-specific orchestrator actions
// per transition target.
func (s *EntityService) GetNextStatus(
    ctx context.Context,
    repo EntityRepository,
    entityType string,
    key string,
    resolveActionFn ResolveActionFn,
) (*NextStatusInfo, error) {
    entity, err := repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get %s: %w", entityType, err)
    }
    if entity == nil {
        return nil, fmt.Errorf("%s not found: %s", entityType, key)
    }

    currentStatus := entity.GetStatus()
    transitions := s.workflowSvc.GetTransitionInfo(currentStatus)
    currentMeta := s.workflowSvc.GetStatusMetadata(currentStatus)

    wrapped := make([]TransitionInfoWithAction, 0, len(transitions))
    for _, t := range transitions {
        var action *config.PopulatedAction
        if resolveActionFn != nil {
            action = resolveActionFn(entity, t.TargetStatus)
        }
        wrapped = append(wrapped, TransitionInfoWithAction{
            TransitionInfo:     t,
            OrchestratorAction: action,
        })
    }

    return &NextStatusInfo{
        EntityType:           entityType,
        EntityKey:            key,
        CurrentStatus:        currentStatus,
        CurrentPhase:         currentMeta.Phase,
        AvailableTransitions: wrapped,
        IsTerminal:           s.workflowSvc.IsTerminalStatus(currentStatus),
    }, nil
}
```

---

## 4. EpicService Refactoring

### 4.1 Constructor Changes

**Before:**
```go
type EpicService struct {
    repo             EpicRepository
    workflowSvc      *workflow.Service      // used for transition + ValidateStatus
    noteRepo         EpicNoteRepository
    featureRepo      EpicFeatureCounter
    taskRepo         EpicTaskLister
    docRepo          config.DocumentRepository
    relRepo          config.EpicRelationshipRepository
    writableDocRepo  EpicWritableDocumentRepository
    analyticsService *EpicAnalyticsService
    enrichRepo       config.TemplateEnrichmentRepository
}

func NewEpicService(repo EpicRepository, workflowSvc *workflow.Service,
    noteRepo EpicNoteRepository, featureRepo EpicFeatureCounter,
    taskRepo EpicTaskLister) *EpicService
```

**After:**
```go
type EpicService struct {
    repo             EpicRepository
    entitySvc        *EntityService          // NEW: shared transition logic
    entityRepo       EntityRepository        // NEW: polymorphic adapter
    noteRepo         EpicNoteRepository      // retained: rejection notes in post-hook
    featureRepo      EpicFeatureCounter
    taskRepo         EpicTaskLister
    docRepo          config.DocumentRepository
    relRepo          config.EpicRelationshipRepository
    writableDocRepo  EpicWritableDocumentRepository
    analyticsService *EpicAnalyticsService
    enrichRepo       config.TemplateEnrichmentRepository
    // workflowSvc REMOVED -- accessed via entitySvc.GetWorkflowService() when needed
}

func NewEpicService(
    repo EpicRepository,
    entitySvc *EntityService,
    entityRepo EntityRepository,
    noteRepo EpicNoteRepository,
    featureRepo EpicFeatureCounter,
    taskRepo EpicTaskLister,
) *EpicService {
    requireNonNil(repo, "EpicService requires a non-nil EpicRepository")
    requireNonNil(entitySvc, "EpicService requires a non-nil EntityService")
    return &EpicService{
        repo:        repo,
        entitySvc:   entitySvc,
        entityRepo:  entityRepo,
        noteRepo:    noteRepo,
        featureRepo: featureRepo,
        taskRepo:    taskRepo,
    }
}
```

**Note on `workflowSvc` removal**: EpicService currently uses `workflowSvc` for
three things: (1) transition logic in `TransitionStatus`, (2) `ValidateStatus`,
and (3) `GetNextStatus`. After refactoring, (1) and (3) are delegated to
`EntityService`. For (2), EpicService calls `s.entitySvc.GetWorkflowService().ValidateStatus()`.
Any other `workflowSvc` usages (e.g., in completion logic) also route through
`entitySvc.GetWorkflowService()`.

### 4.2 TransitionStatus -- After (from ~80 lines to ~20 lines)

```go
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string,
    targetStatus string, opts TransitionOptions) (*TransitionResult, error) {

    // Delegate shared logic to EntityService
    result, err := s.entitySvc.TransitionStatus(
        ctx, s.entityRepo, "epic", epicKey, targetStatus, opts,
        DefaultTransitionFeatures(),
        s.makeResolveActionFn(ctx),
    )
    if err != nil {
        return nil, err
    }

    // Post-hook: rejection note (entity-specific note repo)
    if (result.IsBackward || result.IsForced) && opts.Reason != "" && s.noteRepo != nil {
        _ = s.noteRepo.CreateRejectionNote(ctx, "epic", /* epicID from entity lookup */
            0, result.FromStatus, result.ToStatus,
            opts.Reason, opts.Agent, opts.DocumentPath)
    }

    // Post-hook: count child features
    if s.featureRepo != nil {
        features, listErr := s.featureRepo.ListByEpic(ctx, /* epicID */)
        if listErr == nil {
            result.ChildCount = len(features)
        }
    }

    return result, nil
}
```

**Issue: EntityService.TransitionStatus does not return the entity ID.**

The post-hooks need the entity's database ID for rejection notes and child counting.
Two approaches:

**Option A (Selected)**: Add `EntityID int64` to `TransitionResult`. This is a
backward-compatible addition (new field, zero value is valid for callers that don't
use it). EntityService sets it from `entity.GetID()` after the successful `GetByKey`.

**Option B**: Have the entity-specific service do its own `GetByKey` before delegating.
This adds a redundant DB query.

We select Option A. `TransitionResult` gains:
```go
type TransitionResult struct {
    // ... existing fields ...
    EntityID int64 `json:"entity_id,omitempty"` // NEW: database ID for post-hooks
}
```

This makes the post-hooks clean:
```go
    // Post-hook: rejection note
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
```

### 4.3 resolveAction -- Replaced by Callback

The current `resolveAction` method on EpicService is ~40 lines with enrichment,
document repo, and relationship repo logic. This method is NOT deleted -- it is
converted into a callback factory:

```go
// makeResolveActionFn returns a ResolveActionFn that generates Epic-specific
// placeholders including enrichment data, related documents, and related epics.
func (s *EpicService) makeResolveActionFn(ctx context.Context) ResolveActionFn {
    return func(entity models.Entity, status string) *config.PopulatedAction {
        epic, ok := entity.(*models.Epic)
        if !ok {
            return nil
        }

        // Fetch enrichment data (optional, graceful degradation)
        var enrichment *config.TemplateEnrichmentData
        if s.enrichRepo != nil {
            data, err := s.enrichRepo.GetEpicEnrichment(ctx, epic.ID)
            if err != nil {
                log.Printf("WARNING: Failed to fetch enrichment data for epic %s: %v", epic.Key, err)
            } else {
                enrichment = data
            }
        }

        var placeholders map[string]string
        if s.docRepo != nil && s.relRepo != nil {
            placeholders = config.EpicPlaceholdersWithRelated(epic, s.docRepo, s.relRepo, ctx, enrichment)
        } else {
            placeholders = config.EpicPlaceholders(epic)
            config.ApplyEnrichmentData(enrichment, placeholders)
        }

        return s.entitySvc.ResolveActionForStatus(status, placeholders)
    }
}
```

This eliminates the `resolveAction` method's workflow lookup, nil checks, and
`PopulatedAction` construction (shared in `EntityService.ResolveActionForStatus`),
while preserving the full entity-specific placeholder generation.

### 4.4 GetNextStatus -- After

```go
func (s *EpicService) GetNextStatus(ctx context.Context, epicKey string) (*NextStatusInfo, error) {
    return s.entitySvc.GetNextStatus(ctx, s.entityRepo, "epic", epicKey,
        s.makeResolveActionFn(ctx))
}
```

### 4.5 ValidateStatus -- After

```go
func (s *EpicService) ValidateStatus(status string) error {
    return s.entitySvc.GetWorkflowService().ValidateStatus(status)
}
```

### 4.6 Lines Removed from EpicService

| Item | Lines Removed |
|------|---------------|
| `TransitionStatus` body (80 -> 20) | -60 |
| `resolveAction` converted to callback (structure shared) | -15 |
| `GetNextStatus` body (30 -> 3) | -27 |
| `workflowSvc` field removed | -1 |
| **Total** | **~103 lines removed** |

---

## 5. FeatureService Refactoring

FeatureService follows the exact same pattern as EpicService. The changes are
structurally identical.

### 5.1 Constructor Changes

```go
type FeatureService struct {
    repo              FeatureRepository
    entitySvc         *EntityService       // NEW
    entityRepo        EntityRepository     // NEW
    noteRepo          FeatureNoteRepository
    taskRepo          FeatureTaskCounter
    docRepo           config.DocumentRepository
    relRepo           config.FeatureRelationshipRepository
    writableDocRepo   FeatureWritableDocumentRepository
    enrichRepo        config.TemplateEnrichmentRepository
    // workflowSvc REMOVED
}

func NewFeatureService(
    repo FeatureRepository,
    entitySvc *EntityService,
    entityRepo EntityRepository,
    noteRepo FeatureNoteRepository,
    taskRepo FeatureTaskCounter,
) *FeatureService
```

### 5.2 TransitionStatus -- After (~20 lines)

```go
func (s *FeatureService) TransitionStatus(ctx context.Context, featureKey string,
    targetStatus string, opts TransitionOptions) (*TransitionResult, error) {

    result, err := s.entitySvc.TransitionStatus(
        ctx, s.entityRepo, "feature", featureKey, targetStatus, opts,
        DefaultTransitionFeatures(),
        s.makeResolveActionFn(ctx),
    )
    if err != nil {
        return nil, err
    }

    // Post-hook: rejection note
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

### 5.3 makeResolveActionFn, GetNextStatus, ValidateStatus

Same pattern as EpicService, with Feature-specific placeholders:

```go
func (s *FeatureService) makeResolveActionFn(ctx context.Context) ResolveActionFn {
    return func(entity models.Entity, status string) *config.PopulatedAction {
        feature, ok := entity.(*models.Feature)
        if !ok {
            return nil
        }
        // ... enrichment + placeholder logic (same structure as current resolveAction) ...
        return s.entitySvc.ResolveActionForStatus(status, placeholders)
    }
}

func (s *FeatureService) GetNextStatus(ctx context.Context, featureKey string) (*NextStatusInfo, error) {
    return s.entitySvc.GetNextStatus(ctx, s.entityRepo, "feature", featureKey,
        s.makeResolveActionFn(ctx))
}

func (s *FeatureService) ValidateStatus(status string) error {
    return s.entitySvc.GetWorkflowService().ValidateStatus(status)
}
```

### 5.4 Lines Removed from FeatureService

| Item | Lines Removed |
|------|---------------|
| `TransitionStatus` body (82 -> 20) | -62 |
| `resolveAction` converted to callback | -15 |
| `GetNextStatus` body (30 -> 3) | -27 |
| `workflowSvc` field removed | -1 |
| **Total** | **~105 lines removed** |

---

## 6. TaskService Refactoring (Hybrid Delegation)

TaskService is the most nuanced refactoring because its `executeStatusTransition`
method calls `StatusUpdateRaw`, which has no equivalent in other entity types.

### 6.1 Constructor Changes

```go
type TaskService struct {
    repo            TaskRepository
    entitySvc       *EntityService       // NEW: shared validation helpers
    // workflowSvc REMOVED -- accessed via entitySvc.GetWorkflowService()
    creatorSvc      *taskcreation.Creator
    noteRepo        TaskNoteRepository
    // ... rest unchanged ...
}

func NewTaskService(
    repo TaskRepository,
    entitySvc *EntityService,     // CHANGED: was *workflow.Service
    creatorSvc *taskcreation.Creator,
    noteRepo TaskNoteRepository,
) *TaskService {
    requireNonNil(repo, "TaskService requires a non-nil TaskRepository")
    requireNonNil(entitySvc, "TaskService requires a non-nil EntityService")
    return &TaskService{
        repo:       repo,
        entitySvc:  entitySvc,
        creatorSvc: creatorSvc,
        noteRepo:   noteRepo,
    }
}
```

### 6.2 TransitionStatus -- After (hybrid delegation)

The key change: shared validation and backward detection are delegated to
`EntityService` helpers. The task-specific `executeStatusTransition` is retained
but uses the validated/normalized target status and backward detection result.

```go
func (s *TaskService) TransitionStatus(ctx context.Context, key string,
    targetStatus string, opts TransitionOptions) (*TransitionResult, error) {

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
    var agentPtr, reasonPtr, docPathPtr *string
    if opts.Agent != "" { agentPtr = &opts.Agent }
    if opts.Reason != "" { reasonPtr = &opts.Reason }
    if opts.DocumentPath != "" { docPathPtr = &opts.DocumentPath }

    txResult, err := s.executeStatusTransition(ctx, task, statusTransitionOpts{
        targetStatus:     targetStatus,
        agent:            agentPtr,
        reason:           reasonPtr,
        documentPath:     docPathPtr,
        force:            opts.Force,
        skipBackwardCheck: true, // already done above via EntityService
    })
    if err != nil {
        return nil, fmt.Errorf("failed to transition task %s to %s: %w", key, targetStatus, err)
    }

    // Step 6: Build result with shared action resolution
    actualTarget := string(txResult.task.Status)
    result := &TransitionResult{
        EntityType:         "task",
        EntityKey:          task.Key,
        EntityID:           task.ID,
        FromStatus:         fromStatus,
        ToStatus:           actualTarget,
        Transitioned:       true,
        Message:            fmt.Sprintf("Transitioned: %s -> %s", fromStatus, actualTarget),
        IsForced:           opts.Force,
        IsBackward:         isBackward,
        Reason:             opts.Reason,
        OrchestratorAction: s.resolveAction(ctx, task, actualTarget),
    }

    if len(txResult.unblockedKeys) > 0 {
        result.Message = fmt.Sprintf("Transitioned: %s -> %s (auto-unblocked: %s)",
            fromStatus, actualTarget, strings.Join(txResult.unblockedKeys, ", "))
    }

    return result, nil
}
```

### 6.3 executeStatusTransition -- Simplified

With validation and backward detection moved to `TransitionStatus`, the
`executeStatusTransition` method can be simplified. The validation and backward
check in steps 1-2 are now skipped when `skipBackwardCheck` is true AND `force`
matches. However, `executeStatusTransition` is also called by other TaskService
methods (e.g., `BlockTask`, `StartTask`), so it must retain its validation logic
for those call paths.

**Decision**: Keep `executeStatusTransition` as-is. It already has `skipBackwardCheck`
and `force` flags. The `TransitionStatus` method now passes `skipBackwardCheck: true`
because it has already performed backward detection. Other callers continue to use
`executeStatusTransition` directly with their own flag settings.

This means validation logic is technically still present in `executeStatusTransition`,
but for the `TransitionStatus` call path, it is bypassed via flags. A follow-up
refactoring could extract `executeStatusTransition` to only do the `StatusUpdateRaw`
+ progress recalculation, but that is out of scope for F03.

### 6.4 resolveAction -- Converted to Use Shared Helper

```go
func (s *TaskService) resolveAction(ctx context.Context, task *models.Task, status string) *config.PopulatedAction {
    // Fetch enrichment data (optional)
    var enrichment *config.TemplateEnrichmentData
    if s.enrichRepo != nil {
        data, err := s.enrichRepo.GetTaskEnrichment(ctx, task.ID)
        if err != nil {
            log.Printf("WARNING: Failed to fetch enrichment data for task %s: %v", task.Key, err)
        } else {
            enrichment = data
        }
    }

    var placeholders map[string]string
    if s.docRepo != nil && s.relRepo != nil {
        placeholders = config.TaskPlaceholdersWithRelated(ctx, task, s.docRepo, s.relRepo, enrichment)
    } else {
        placeholders = config.TaskPlaceholders(task)
        config.ApplyEnrichmentData(enrichment, placeholders)
    }

    // Delegate workflow lookup + PopulatedAction construction to shared helper
    return s.entitySvc.ResolveActionForStatus(status, placeholders)
}
```

### 6.5 GetNextStatus -- After

```go
func (s *TaskService) GetNextStatus(ctx context.Context, key string) (*NextStatusInfo, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get task %s: %w", key, err)
    }

    currentStatus := string(task.Status)
    wfSvc := s.entitySvc.GetWorkflowService()
    transitions := wfSvc.GetTransitionInfo(currentStatus)
    currentMeta := wfSvc.GetStatusMetadata(currentStatus)

    wrapped := make([]TransitionInfoWithAction, 0, len(transitions))
    for _, t := range transitions {
        wrapped = append(wrapped, TransitionInfoWithAction{
            TransitionInfo:     t,
            OrchestratorAction: s.resolveAction(ctx, task, t.TargetStatus),
        })
    }

    return &NextStatusInfo{
        EntityType:           "task",
        EntityKey:            key,
        CurrentStatus:        currentStatus,
        CurrentPhase:         currentMeta.Phase,
        AvailableTransitions: wrapped,
        IsTerminal:           wfSvc.IsTerminalStatus(currentStatus),
    }, nil
}
```

**Note**: TaskService's `GetNextStatus` does NOT delegate to `EntityService.GetNextStatus`
because it uses the typed `*models.Task` for its `resolveAction` call (which needs the
task's typed fields for enrichment). Using `EntityService.GetNextStatus` with a
`ResolveActionFn` callback would work but requires a type assertion inside the callback.
Either approach is acceptable; using the typed repo directly is simpler for Task's case.

### 6.6 ValidateStatus and Other workflowSvc Usages

All `s.workflowSvc.XXX` calls in TaskService are replaced with
`s.entitySvc.GetWorkflowService().XXX`:

```go
func (s *TaskService) ValidateStatus(status string) error {
    return s.entitySvc.GetWorkflowService().ValidateStatus(status)
}
```

### 6.7 Lines Removed from TaskService

| Item | Lines Removed |
|------|---------------|
| `TransitionStatus` shared validation extracted | -10 |
| `resolveAction` workflow lookup/nil-checks shared | -10 |
| `workflowSvc` field removed | -1 |
| **Total** | **~21 lines removed** |

TaskService's savings are smaller because the hybrid approach retains
`executeStatusTransition` with its own validation path. The primary value is
consistency (validation logic is the same across all entity types) rather than
line count reduction.

---

## 7. Bug/ChangeCard Delegation (Should-Have)

### 7.1 Approach

Bug and ChangeCard services can optionally delegate their `SetBugStatus` /
`SetChangeCardStatus` methods to `EntityService.TransitionStatus` using
`SimpleTransitionFeatures()`.

```go
func (s *BugService) SetBugStatus(ctx context.Context, key string, status string,
    force bool, reason string) (*models.Bug, error) {

    opts := TransitionOptions{Force: force, Reason: reason}
    if force && reason == "" {
        opts.Reason = "forced status change"
    }

    _, err := s.entitySvc.TransitionStatus(
        ctx, s.entityRepo, "bug", key, status, opts,
        SimpleTransitionFeatures(),
        nil, // no orchestrator action for bugs (or provide callback if needed)
    )
    if err != nil {
        return nil, err
    }

    // Reload typed Bug
    return s.repo.GetByKey(ctx, key)
}
```

### 7.2 Risk Assessment

Bug and ChangeCard are lower-risk because `SimpleTransitionFeatures` skips backward
detection and rejection notes. The main risk is if their current validation logic
differs subtly from the shared implementation. Current Bug/ChangeCard status methods
are ~25 lines each and use direct workflow validation, matching the shared path.

**Recommendation**: Implement this as a separate task after the core 3 services are
refactored and validated. If any behavioral differences are found, keep the current
implementations.

---

## 8. CLI Wiring Changes (services_global.go)

### 8.1 New: GetEntityService()

```go
var (
    globalEntityService *services.EntityService
    entityServiceOnce   sync.Once
)

func GetEntityService() *services.EntityService {
    entityServiceOnce.Do(func() {
        wfSvc := GetWorkflowService()
        globalEntityService = services.NewEntityService(wfSvc)
    })
    return globalEntityService
}
```

### 8.2 Updated: GetEpicService()

**Before:**
```go
func GetEpicService() *services.EpicService {
    db, err := GetDB(context.Background())
    // ...
    epicRepo := repository.NewEpicRepository(db)
    workflowSvc := GetWorkflowService()
    noteRepo := repository.NewEntityNoteRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)
    svc := services.NewEpicService(epicRepo, workflowSvc, noteRepo, featureRepo, taskRepo)
    // ... setters for docRepo, relRepo, enrichRepo ...
    return svc
}
```

**After:**
```go
func GetEpicService() *services.EpicService {
    db, err := GetDB(context.Background())
    // ...
    epicRepo := repository.NewEpicRepository(db)
    entitySvc := GetEntityService()
    entityRepo := GetEntityRegistry().MustGetRepository(models.EntityTypeEpic)
    noteRepo := repository.NewEntityNoteRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)
    svc := services.NewEpicService(epicRepo, entitySvc, entityRepo, noteRepo, featureRepo, taskRepo)
    // ... setters unchanged ...
    return svc
}
```

### 8.3 Updated: GetFeatureService()

Same pattern -- add `entitySvc` and `entityRepo`, remove direct `workflowSvc`.

### 8.4 Updated: GetTaskService()

```go
func GetTaskService() *services.TaskService {
    db, err := GetDB(context.Background())
    // ...
    taskRepo := repository.NewTaskRepository(db)
    entitySvc := GetEntityService()
    // ... rest of existing wiring ...
    svc := services.NewTaskService(taskRepo, entitySvc, creatorSvc, noteRepo)
    // ... setters unchanged ...
    return svc
}
```

### 8.5 Updated: ResetServices()

```go
func ResetServices() {
    // ... existing resets ...
    globalEntityService = nil
    entityServiceOnce = sync.Once{}
}
```

### 8.6 HTTP Server Wiring (cmd/server/services.go)

The `WireServices` function needs the same updates: create `EntityService` once,
pass to each entity service constructor.

```go
func WireServices(db *repository.DB, projectRoot string) *Services {
    wfSvc := workflow.NewService(projectRoot)
    entitySvc := services.NewEntityService(wfSvc)
    registry := services.NewEntityRegistry()
    // ... register adapters ...

    epicSvc := services.NewEpicService(
        epicRepo, entitySvc,
        registry.MustGetRepository(models.EntityTypeEpic),
        noteRepo, featureRepo, taskRepo,
    )
    // ... similarly for Feature, Task ...
}
```

---

## 9. TransitionResult EntityID Addition

As discussed in Section 4.2, `TransitionResult` gains an `EntityID` field:

```go
// In transition_types.go
type TransitionResult struct {
    EntityType         string                  `json:"entity_type"`
    EntityKey          string                  `json:"entity_key"`
    EntityID           int64                   `json:"entity_id,omitempty"`  // NEW
    FromStatus         string                  `json:"from_status"`
    ToStatus           string                  `json:"to_status"`
    Transitioned       bool                    `json:"transitioned"`
    Message            string                  `json:"message,omitempty"`
    OrchestratorAction *config.PopulatedAction `json:"orchestrator_action"`
    IsBackward         bool                    `json:"is_backward,omitempty"`
    IsForced           bool                    `json:"is_forced,omitempty"`
    Reason             string                  `json:"reason,omitempty"`
    ChildCount         int                     `json:"child_count,omitempty"`
}
```

This is backward-compatible (new field with zero value; `omitempty` hides it from
JSON output when not set).

---

## 10. Migration Strategy -- Implementation Order

### Task 1: Create EntityService with TransitionStatus, helpers, and ResolveActionForStatus

**Files created**: `entity_service.go`
**Files modified**: `transition_types.go` (add EntityID to TransitionResult)
**Changes**:
1. Create `EntityService` struct with `workflowSvc` field
2. Implement `NewEntityService` constructor
3. Implement `TransitionStatus` (shared 9-step algorithm)
4. Implement `ValidateAndNormalize` helper
5. Implement `DetectBackward` helper
6. Implement `ResolveActionForStatus` helper
7. Implement `GetNextStatus` method
8. Implement `GetWorkflowService` accessor
9. Define `TransitionFeatures`, `DefaultTransitionFeatures()`, `SimpleTransitionFeatures()`
10. Define `ResolveActionFn` type
11. Add `EntityID` to `TransitionResult`

**Validation**: `make fmt && make lint && make test`

### Task 2: Write EntityService unit tests

**Files created**: `entity_service_test.go`
**Changes**:
1. Create `MockEntityRepository` for testing
2. Create mock workflow service for testing
3. Write parameterized tests for `TransitionStatus`:
   - Happy path (Epic, Feature entity types)
   - Forced transition with reason
   - Forced transition without reason (error)
   - Backward transition with reason
   - Backward transition without reason (error)
   - SimpleTransitionFeatures (no backward detection)
   - Repository error propagation
   - Entity not found
4. Write tests for `ValidateAndNormalize`
5. Write tests for `DetectBackward`
6. Write tests for `ResolveActionForStatus`
7. Write tests for `GetNextStatus`
8. Target: 85%+ coverage

**Validation**: `make fmt && make lint && make test`

### Task 3: Refactor EpicService to delegate to EntityService

**Files modified**: `epic_service.go`, `epic_service_test.go`
**Changes**:
1. Add `entitySvc *EntityService` and `entityRepo EntityRepository` fields
2. Update constructor to accept new dependencies, remove `workflowSvc` parameter
3. Replace `TransitionStatus` body with delegation + post-hooks
4. Convert `resolveAction` to `makeResolveActionFn` callback factory
5. Replace `GetNextStatus` body with delegation
6. Replace `s.workflowSvc.XXX` calls with `s.entitySvc.GetWorkflowService().XXX`
7. Update all tests for new constructor signature

**Validation**: `make fmt && make lint && make test`

### Task 4: Refactor FeatureService to delegate to EntityService

**Files modified**: `feature_service.go`, `feature_service_test.go`
**Changes**: Same as Task 3 but for FeatureService.

**Validation**: `make fmt && make lint && make test`

### Task 5: Refactor TaskService to use hybrid delegation

**Files modified**: `task_service.go`, `task_service_test.go`
**Changes**:
1. Add `entitySvc *EntityService` field, remove `workflowSvc` field
2. Update constructor to accept `entitySvc` instead of `workflowSvc`
3. Refactor `TransitionStatus` to use `ValidateAndNormalize` and `DetectBackward`
4. Convert `resolveAction` to use `ResolveActionForStatus` shared helper
5. Replace `s.workflowSvc.XXX` with `s.entitySvc.GetWorkflowService().XXX`
6. Update all tests for new constructor signature

**Validation**: `make fmt && make lint && make test`

### Task 6: Update CLI wiring (services_global.go) and HTTP wiring

**Files modified**: `internal/cli/services_global.go`, `cmd/server/services.go`
**Changes**:
1. Add `GetEntityService()` function with `sync.Once`
2. Update `GetEpicService()` to pass `entitySvc` and `entityRepo`
3. Update `GetFeatureService()` to pass `entitySvc` and `entityRepo`
4. Update `GetTaskService()` to pass `entitySvc` instead of `workflowSvc`
5. Update `ResetServices()` to reset `EntityService` state
6. Update `WireServices()` in server for HTTP API

**Validation**: `make fmt && make lint && make test`
**Additional**: Run end-to-end CLI tests: `shark status advance` and `shark status set`
for epics, features, and tasks.

### Task 7: (Should-Have) Refactor Bug/ChangeCard to delegate

**Files modified**: `bug_service.go`, `change_card_service.go`, tests
**Changes**: Optionally delegate `SetBugStatus`/`SetChangeCardStatus` to
`EntityService.TransitionStatus` with `SimpleTransitionFeatures()`.

**Validation**: `make fmt && make lint && make test`

### Dependency Graph

```
Task 1 (EntityService) ──> Task 2 (Tests) ──> Task 3 (Epic) ──┐
                                              Task 4 (Feature) ├──> Task 6 (Wiring)
                                              Task 5 (Task)  ──┘         │
                                                                         v
                                                                  Task 7 (Bug/CC)
```

Tasks 3, 4, 5 can proceed in parallel after Task 2 since they modify separate
service files. Task 6 must follow all three because it changes the constructor
call sites. Task 7 is independent and can be done any time after Task 6.

---

## 11. Summary Metrics

### Lines Removed vs Added

| File | Lines Before (est.) | Lines After (est.) | Net Change |
|------|--------------------|--------------------|------------|
| `entity_service.go` (NEW) | 0 | ~150 | +150 |
| `epic_service.go` | ~400 | ~300 | -100 |
| `feature_service.go` | ~400 | ~300 | -100 |
| `task_service.go` | ~1400 | ~1380 | -20 |
| `transition_types.go` | 73 | 74 | +1 |
| `services_global.go` | ~250 | ~260 | +10 |
| **Total** | | | **~-59 net** |

**Note**: The net line reduction is modest because the shared logic (~150 lines in
`entity_service.go`) replaces duplicated logic across 3 files. The primary value
is not line count but **single-location maintenance**: transition validation,
backward detection, reason enforcement, and action resolution are each implemented
once instead of 3-5 times.

### Duplication Eliminated

| Duplicated Pattern | Copies Before | Copies After | Savings |
|--------------------|---------------|--------------|---------|
| Transition validation + normalization | 3 (Epic, Feature, Task) | 1 (EntityService) | 2 eliminated |
| Backward detection + reason enforcement | 3 | 1 | 2 eliminated |
| Force-reason enforcement | 3 | 1 (EntityService) + 1 (TaskService check) | 1 eliminated |
| `resolveAction` workflow lookup + nil checks | 5 | 1 (`ResolveActionForStatus`) | 4 eliminated |
| `GetNextStatus` entity lookup + wrapping | 3 | 1 (EntityService) + 1 (TaskService inline) | 1 eliminated |

### Bug Fix Effort

| Bug Category | Fix Locations Before | Fix Locations After |
|--------------|---------------------|---------------------|
| Transition validation | 3 | 1 |
| Backward detection | 3 | 1 |
| Rejection note logic | 3 | 3 (post-hooks; shared detection in 1) |
| resolveAction | 5 | 1 (shared) + 5 (placeholder callbacks) |
| GetNextStatus | 3 | 1 + 1 |

### New Entity Type Effort

Adding a 6th entity type's status transition:
- **Before**: Copy ~120 lines from an existing service
- **After**: ~20 lines of delegation code + ~15 lines of placeholder callback

---

## 12. Risk Assessment

### Low Risk
- EpicService and FeatureService delegation: These are structurally identical to
  the shared implementation. The refactoring is a direct extraction.
- `ResolveActionForStatus`: Pure function, easy to test.
- `GetNextStatus` for Epic/Feature: Direct delegation, no behavioral change.

### Medium Risk
- TaskService hybrid delegation: The interaction between `ValidateAndNormalize`,
  `DetectBackward`, and `executeStatusTransition` must be carefully tested to ensure
  no double-validation or skipped validation.
- Constructor signature changes: All callers (CLI wiring, HTTP wiring, tests) must
  be updated. This is tedious but mechanical.

### Mitigation
- Run full test suite after each task.
- TaskService refactoring should be its own commit with focused review.
- End-to-end CLI testing for all 3 entity types after wiring changes.

---

*Derived from F01 implementation (entity_repository.go, entity_registry.go,
*_repo_adapter.go), F02 architecture (02-architecture.md), epic
architecture-design.md, and actual codebase inspection at commit d33297e on branch
e21-f01-entity-interface-foundation.*
