# Architecture Review: Entity Polymorphism and Duplication Reduction

**Date:** 2026-03-18
**Status:** Proposed
**Author:** Architect Agent

---

## 1. Executive Summary

The Shark Task Manager codebase has five entity types -- Epic, Feature, Task, Bug, and ChangeCard -- that share substantial structural and behavioral overlap. Every new cross-cutting feature (notes, context, documents, status transitions, templates, display) must be independently implemented for each entity type. This review quantifies the duplication, identifies what truly varies per entity, and proposes an incremental path toward polymorphism via a shared Entity interface.

---

## 2. Current State Analysis

### 2.1 Scale of the Problem

| Layer | Files | Total Lines | Entity Types |
|-------|-------|-------------|--------------|
| Models | 5 | ~401 | Epic, Feature, Task, Bug, ChangeCard |
| Repositories | 5 | ~4,227 | Epic, Feature, Task, Bug, ChangeCard |
| Services | 5 (core) + 3 (cross-cutting) | ~5,806 | All |
| CLI Commands | ~76 | ~24,041 | All |
| CLI Accessors | 2 | ~647 | All |
| **Total** | **~96** | **~35,122** | |

### 2.2 Shared Fields Across All Entities

Every entity has these fields (or near-equivalents):

| Field | Epic | Feature | Task | Bug | ChangeCard |
|-------|------|---------|------|-----|------------|
| `ID` (int64) | Y | Y | Y | Y | Y |
| `Key` (string) | Y | Y | Y | Y | Y |
| `Title` (string) | Y | Y | Y | Y | Y |
| `Slug` (*string or string) | Y | Y | Y | Y | Y |
| `Description` (*string) | Y | Y | Y | Y | Y |
| `Status` (typed string) | Y | Y | Y | Y | Y |
| `FilePath` (*string or string) | Y | Y | Y | Y | Y |
| `ContextData` (*string) | Y | Y | Y | Y | Y |
| `CreatedAt` (time.Time) | Y | Y | Y | Y | Y |
| `UpdatedAt` (time.Time) | Y | Y | Y | Y | Y |

That is 10 shared fields across all 5 entity types. Only the `Status` type wrapper differs (EpicStatus, FeatureStatus, TaskStatus, BugStatus, ChangeCardStatus), but they are all `string` underneath.

### 2.3 Entity-Specific Fields

**Epic only:** Priority, BusinessValue
**Feature only:** EpicID (parent), ProgressPct, StatusOverride, ExecutionOrder
**Task only:** FeatureID (parent), AgentType, Priority (int), DependsOn, AssignedAgent, BlockedReason, ExecutionOrder, completion metadata (CompletedBy, CompletionNotes, FilesChanged, TestsPassed, etc.), RejectionCount
**Bug only:** Severity, LinkedEntityType, LinkedEntityKey
**ChangeCard only:** Priority (int), RequestedBy, AssignedTo, EpicID, FeatureID, RelatedTaskID, Justification, ImpactAnalysis, RollbackPlan

### 2.4 Duplicated Service Patterns

The following method patterns are copy-pasted (with entity name substitution) across services:

#### A. CRUD Operations (duplicated 5x)

```
GetEntity(ctx, key) -> (*Entity, error)         -- identical pattern across all 5
ListEntities(ctx, filters) -> ([]*Entity, error) -- identical pattern across all 5
CreateEntity(ctx, input) -> (*Entity, error)     -- identical pattern across all 5
UpdateEntity(ctx, key, updates) -> (*Entity, error) -- identical pattern across all 5
DeleteEntity(ctx, key) -> error                  -- identical pattern across all 5
```

**Example of identical Get patterns:**

```go
// EpicService.GetEpic
func (s *EpicService) GetEpic(ctx context.Context, key string) (*models.Epic, error) {
    epic, err := s.repo.GetByKey(ctx, key)
    if err != nil { return nil, fmt.Errorf("failed to get epic %s: %w", key, err) }
    if epic == nil { return nil, fmt.Errorf("epic not found: %s", key) }
    return epic, nil
}

// BugService.GetBug
func (s *BugService) GetBug(ctx context.Context, key string) (*models.Bug, error) {
    bug, err := s.repo.GetByKey(ctx, key)
    if err != nil { return nil, fmt.Errorf("failed to get bug %s: %w", key, err) }
    return bug, nil
}

// ChangeCardService.GetChangeCard
func (s *ChangeCardService) GetChangeCard(ctx context.Context, key string) (*models.ChangeCard, error) {
    card, err := s.repo.GetByKey(ctx, key)
    if err != nil { return nil, fmt.Errorf("failed to get change-card %s: %w", key, err) }
    return card, nil
}
```

These are structurally identical. Only the type name and error message vary.

#### B. Status Transition Operations (duplicated 5x)

```
TransitionStatus(ctx, key, target, opts)  -- identical logic per entity
AdvanceStatus(ctx, key)                   -- identical logic per entity
SetStatus(ctx, key, status, force)        -- identical logic per entity
GetNextStatus(ctx, key)                   -- identical logic per entity
ValidateStatus(status)                    -- identical logic per entity
GetValidTransitions(status)               -- identical logic per entity
```

**The TransitionStatus method is nearly identical across Epic, Feature, and Task services:**
1. Get entity by key
2. Extract current status string
3. Validate transition (unless forced)
4. Normalize status
5. Check backward transition
6. Require reason for backward
7. Update entity status
8. Create rejection note if backward
9. Count children for warning
10. Resolve orchestrator action
11. Return TransitionResult

The only variation is the model type cast and which child entities to count.

#### C. Document Operations (duplicated 5x)

```
LinkDocument(ctx, key, title, path)       -- identical pattern per entity
UnlinkDocument(ctx, key, title)           -- identical pattern per entity
ListRelatedDocumentsByKey(ctx, key)       -- identical pattern per entity
SetWritableDocRepo(repo)                  -- identical pattern per entity
```

These already use shared helpers (`linkDocumentToEntity`, `unlinkDocumentFromEntity`) but each entity service still has its own wrapper that does:
1. Check writableDocRepo is not nil
2. Look up entity by key
3. Call shared helper with entity ID

#### D. Orchestrator Action Resolution (duplicated 5x)

```
resolveAction(ctx, entity, status)        -- identical pattern per entity
GetOrchestratorAction(entity)             -- identical pattern per entity
```

Pattern in each:
1. Get workflow from workflowSvc
2. Check StatusMetadata for status
3. Get OrchestratorAction from metadata
4. Build placeholders from entity fields
5. Return PopulatedAction

The only variation is which placeholder function to call.

### 2.5 Duplicated Cross-Cutting Services

**NoteService** -- Contains a 5-branch `switch` on EntityType in both `resolveEntityID` and `GetEntityDetails`. Each branch does the same thing: call `repo.GetByKey(ctx, key)` and extract `.ID` or `.Key`/`.Title`.

**ContextService** -- Contains two 5-branch `switch` statements (`getContextJSON` and `setContextJSON`). Each branch does the same thing: call repo.GetByKey, then get/set ContextData.

**DisplayService** -- Has separate `GetEpicDisplayInfo`, `GetFeatureDisplayInfo` methods with distinct but structurally similar `populatePlanningInfo`/`populateAggregationInfo` methods.

### 2.6 Duplicated Repository Interface Definitions

Each service defines its own repository interface. The NoteService alone defines 5 separate "note repository" interfaces -- one per entity type -- each with identical method signatures (`GetByKey`, `GetByID`):

```go
type NoteEpicRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Epic, error)
    GetByID(ctx context.Context, id int64) (*models.Epic, error)
}
type NoteFeatureRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Feature, error)
    GetByID(ctx context.Context, id int64) (*models.Feature, error)
}
type NoteTaskRepository interface { ... }
type NoteChangeCardRepository interface { ... }
type NoteBugRepository interface { ... }
```

These differ only in return type. Similarly, ContextService defines 5 separate repository interfaces.

### 2.7 Duplicated CLI Accessor Functions

`services_global.go` has separate `GetEpicService()`, `GetFeatureService()`, `GetTaskService()`, `GetBugService()`, `GetChangeCardService()` functions that follow the same pattern:
1. Get DB
2. Get workflow service
3. Create entity-specific repository
4. Create entity-specific service
5. Wire optional dependencies
6. Return service

### 2.8 Estimated Duplication

Based on the patterns above, I estimate:

| Duplicated Pattern | Instances | Lines Each | Total Duplicate Lines |
|---|---|---|---|
| CRUD service methods | 5x | ~40 | ~200 |
| Status transition methods | 5x | ~80 | ~400 |
| Document linking methods | 5x | ~30 | ~150 |
| Action resolution methods | 5x | ~25 | ~125 |
| Repository interface defs | ~15 | ~5 | ~75 |
| CLI accessor functions | 5x | ~25 | ~125 |
| NoteService switch branches | 2x5 | ~8 | ~80 |
| ContextService switch branches | 2x5 | ~10 | ~100 |
| **Estimated total** | | | **~1,255 lines** |

This is a conservative estimate. When you factor in the CLI command implementations (76 files, ~24,000 lines) where each entity type has parallel `create`, `get`, `list`, `update`, `delete`, `note`, `context`, `status` commands, the effective duplication is much higher.

---

## 3. What Truly Varies Per Entity

### 3.1 Organizational Tier

| Entity | Level | Has Parent | Has Children | Key Pattern |
|--------|-------|-----------|--------------|-------------|
| Epic | Top | No | Features | `E##` |
| Feature | Mid | Epic | Tasks | `E##-F##` |
| Task | Leaf | Feature | None | `E##-F##-###` |
| Bug | Standalone | No (linked optionally) | None | `B###` |
| ChangeCard | Standalone | No (linked optionally) | None | `CC-###` |

### 3.2 Entity-Specific Business Logic

**Epic:** Feature rollup, progress calculation from features, cascade status to children, impediment tracking

**Feature:** Task rollup, progress calculation from tasks, status override/auto-calculation, execution ordering

**Task:** Dependency graph, agent assignment, completion metadata, acceptance criteria, work sessions, history tracking, TDD workflow phases

**Bug:** Severity levels, linked entity validation, triage workflow

**ChangeCard:** Approval workflow, impact analysis, rollback plan, requested-by/assigned-to

### 3.3 Summary

The common surface area (CRUD, status transitions, notes, context, documents, templates, display) is roughly 60-70% of each entity's code. The entity-specific logic (hierarchy management, progress rollups, dependencies, severity, approval) is 30-40%.

---

## 4. Proposed Architecture

### 4.1 Design Principles

1. **Appropriate**: Introduce polymorphism where it eliminates real duplication, not for theoretical purity
2. **Proven**: Use Go interfaces and composition -- no reflection, no code generation, no generics abuse
3. **Simple**: Incremental adoption -- existing code continues to work while new patterns are introduced

### 4.2 Core Entity Interface

```go
// internal/models/entity.go

// Entity is the common interface implemented by all entity types.
// It provides access to the universal fields that every entity shares.
type Entity interface {
    // Identity
    GetID() int64
    GetKey() string
    GetTitle() string
    GetSlug() string
    GetEntityType() EntityType

    // Status
    GetStatus() string
    SetStatus(status string)

    // Metadata
    GetDescription() string
    GetFilePath() string
    GetContextData() *string
    SetContextData(data *string)
    GetCreatedAt() time.Time
    GetUpdatedAt() time.Time

    // Validation
    Validate() error
}
```

Each existing model struct implements this interface via simple accessor methods:

```go
// On models.Epic:
func (e *Epic) GetID() int64           { return e.ID }
func (e *Epic) GetKey() string         { return e.Key }
func (e *Epic) GetTitle() string       { return e.Title }
func (e *Epic) GetEntityType() EntityType { return EntityTypeEpic }
func (e *Epic) GetStatus() string      { return string(e.Status) }
func (e *Epic) SetStatus(s string)     { e.Status = EpicStatus(s) }
// ... etc for all fields
```

This is non-invasive -- existing code that uses `epic.Key` continues to work. The interface only adds a polymorphic capability on top.

### 4.3 Generic Entity Repository Interface

```go
// internal/repository/entity_repository.go

// EntityRepository defines the common data access operations shared by all entities.
// Entity-specific repositories embed this and add their own methods.
type EntityRepository interface {
    GetByKey(ctx context.Context, key string) (models.Entity, error)
    GetByID(ctx context.Context, id int64) (models.Entity, error)
    Create(ctx context.Context, entity models.Entity) error
    Update(ctx context.Context, entity models.Entity) error
    Delete(ctx context.Context, id int64) error
    UpdateStatus(ctx context.Context, id int64, status string) error
    GetContextData(ctx context.Context, id int64) (*string, error)
    UpdateContextData(ctx context.Context, id int64, data *string) error
}
```

Each concrete repository already satisfies most of these methods. The adaptation is:
- Return `models.Entity` instead of `*models.Epic` (concrete methods still return typed pointers for callers that need them)
- Add a thin `AsEntity()` adapter or have the repository methods return both

**Important**: We do NOT replace the existing typed repository interfaces. Services that need `*models.Task` specifically still use `TaskRepository`. The `EntityRepository` is an additional interface for cross-cutting services.

### 4.4 Generic Entity Service for Cross-Cutting Operations

```go
// internal/services/entity_service.go

// EntityService provides operations that are identical across all entity types:
// status transitions, document linking, context management, notes.
type EntityService struct {
    workflowSvc     *workflow.Service
    noteRepo        EntityNoteRepository
    writableDocRepo WritableDocumentRepository
}

// TransitionStatus is the SINGLE implementation for all entity types.
func (s *EntityService) TransitionStatus(
    ctx context.Context,
    repo EntityRepository,
    key string,
    targetStatus string,
    opts TransitionOptions,
) (*TransitionResult, error) {
    entity, err := repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get entity: %w", err)
    }

    currentStatus := entity.GetStatus()
    entityType := string(entity.GetEntityType())

    if !opts.Force {
        if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
            return nil, err
        }
    }

    // ... identical logic that currently exists in 5 places ...

    if err := repo.UpdateStatus(ctx, entity.GetID(), targetStatus); err != nil {
        return nil, fmt.Errorf("failed to update %s status: %w", entityType, err)
    }

    // ... rejection notes, child counting, action resolution ...

    return &TransitionResult{
        EntityType: entityType,
        EntityKey:  key,
        FromStatus: currentStatus,
        ToStatus:   targetStatus,
        // ...
    }, nil
}
```

**Entity-specific services compose the generic service:**

```go
type EpicService struct {
    repo          EpicRepository       // typed repo for epic-specific queries
    entitySvc     *EntityService       // shared logic
    featureRepo   EpicFeatureCounter   // epic-specific: child counting
    // ...
}

func (s *EpicService) TransitionStatus(ctx, key, target, opts) (*TransitionResult, error) {
    // Delegate to shared implementation
    result, err := s.entitySvc.TransitionStatus(ctx, s.repo, key, target, opts)
    if err != nil {
        return nil, err
    }
    // Epic-specific: count child features
    if s.featureRepo != nil {
        features, _ := s.featureRepo.ListByEpic(ctx, result.EntityID)
        result.ChildCount = len(features)
    }
    return result, nil
}
```

### 4.5 Unified Cross-Cutting Services

**NoteService** currently has 5-branch switches. With the Entity interface:

```go
// Before: 5 separate repository interfaces + 5-branch switch
func (s *NoteService) resolveEntityID(ctx, entityType, key) (int64, error) {
    switch entityType {
    case "epic":    epic, err := s.epicRepo.GetByKey(ctx, key); return epic.ID, err
    case "feature": feature, err := s.featureRepo.GetByKey(ctx, key); return feature.ID, err
    // ... 3 more branches, all identical pattern
    }
}

// After: single EntityRepository lookup
func (s *NoteService) resolveEntityID(ctx, entityType, key) (int64, error) {
    repo, ok := s.repos[entityType]
    if !ok { return 0, fmt.Errorf("unsupported entity type: %s", entityType) }
    entity, err := repo.GetByKey(ctx, key)
    if err != nil { return 0, fmt.Errorf("%s not found: %s: %w", entityType, key, err) }
    return entity.GetID(), nil
}
```

**ContextService** becomes drastically simpler:

```go
// Before: 5-branch switch in getContextJSON AND setContextJSON (10 branches total)
// After:
func (s *ContextService) getContextJSON(ctx, entityType, key) (*string, error) {
    repo, ok := s.repos[entityType]
    if !ok { return nil, fmt.Errorf("unsupported: %s", entityType) }
    return repo.GetContextData(ctx, key)
}
```

### 4.6 Placeholder Resolution via Entity Interface

```go
// Before: 5 separate functions
config.EpicPlaceholders(epic)
config.FeaturePlaceholders(feature)
config.TaskPlaceholders(task)
config.BugPlaceholders(bug)
config.ChangeCardPlaceholders(card)

// After: single function using Entity interface
func EntityPlaceholders(entity models.Entity) map[string]string {
    return map[string]string{
        "id":          entity.GetKey(),
        "key":         entity.GetKey(),
        "title":       entity.GetTitle(),
        "status":      entity.GetStatus(),
        "entity_type": string(entity.GetEntityType()),
        "description": entity.GetDescription(),
        "file_path":   entity.GetFilePath(),
        // ... all common fields
    }
}

// Entity-specific placeholders can extend:
func TaskPlaceholders(task *models.Task) map[string]string {
    m := EntityPlaceholders(task)
    m["agent_type"] = safeDeref(task.AgentType)
    m["feature_key"] = parseFeatureKeyFromTaskKey(task.Key)
    // ... task-specific fields
    return m
}
```

### 4.7 Registry Pattern for Entity Resolution

A central registry maps entity types to their repositories and services:

```go
// internal/services/entity_registry.go

type EntityRegistry struct {
    repos    map[models.EntityType]EntityRepository
    services map[models.EntityType]EntityTypeService
}

func NewEntityRegistry() *EntityRegistry {
    return &EntityRegistry{
        repos:    make(map[models.EntityType]EntityRepository),
        services: make(map[models.EntityType]EntityTypeService),
    }
}

func (r *EntityRegistry) Register(entityType models.EntityType, repo EntityRepository, svc EntityTypeService) {
    r.repos[entityType] = repo
    r.services[entityType] = svc
}

func (r *EntityRegistry) GetRepo(entityType models.EntityType) (EntityRepository, error) {
    repo, ok := r.repos[entityType]
    if !ok {
        return nil, fmt.Errorf("unknown entity type: %s", entityType)
    }
    return repo, nil
}
```

This replaces all the `SetBugRepo()`, `SetChangeCardRepo()` setter methods with a single registration mechanism.

---

## 5. Migration Strategy

### Phase 1: Foundation (Low Risk, High Value)
**Estimated effort: M (3-5 tasks)**

1. Define `Entity` interface in `internal/models/entity.go`
2. Implement interface methods on all 5 entity structs (mechanical, backward-compatible)
3. Add `EntityRepository` interface in `internal/repository/`
4. Create adapters so existing typed repositories satisfy `EntityRepository`
5. No existing code changes -- purely additive

**Value:** Enables all subsequent phases. Zero risk to existing functionality.

### Phase 2: Unify Cross-Cutting Services (Medium Risk, High Value)
**Estimated effort: L (5-8 tasks)**

1. Refactor `NoteService` to use `EntityRepository` map instead of 5 separate repos + switch
2. Refactor `ContextService` to use `EntityRepository` map
3. Refactor `ResumeService` similarly
4. Replace all `SetBugRepo()` / `SetChangeCardRepo()` patterns with registry
5. Update CLI accessors to use registry pattern

**Value:** Eliminates ~300 lines of switch-statement duplication. Adding a 6th entity type becomes trivial (one registry call vs 3+ service modifications).

### Phase 3: Unify Status Transitions (Medium Risk, Very High Value)
**Estimated effort: L (5-8 tasks)**

1. Extract `EntityService.TransitionStatus()` as shared implementation
2. Refactor `EpicService.TransitionStatus()` to delegate + add epic-specific logic
3. Refactor `FeatureService.TransitionStatus()` similarly
4. Refactor `TaskService.TransitionStatus()` similarly
5. Do the same for `AdvanceStatus`, `GetNextStatus`, `ValidateStatus`, `resolveAction`

**Value:** Eliminates ~400 lines of the most complex duplicated logic. Bug fixes to transition logic apply once, not 5 times.

### Phase 4: Unify Document Operations (Low Risk, Medium Value)
**Estimated effort: S (2-3 tasks)**

1. Move `LinkDocument`/`UnlinkDocument`/`ListRelatedDocumentsByKey` to `EntityService`
2. Entity-specific services delegate to `EntityService` with their typed repo

**Value:** Eliminates ~150 lines. Shared helpers already exist; this completes the unification.

### Phase 5: Unify Template Placeholders (Low Risk, Medium Value)
**Estimated effort: S (2-3 tasks)**

1. Create `EntityPlaceholders(entity models.Entity)` base function
2. Have entity-specific placeholder functions extend from base
3. Update template helpers to use Entity interface where possible

**Value:** Adding template variables for all entities becomes a single change.

### Phase 6: Unify CLI Commands (High Effort, Very High Value)
**Estimated effort: XL (10+ tasks, can be done incrementally)**

1. Create generic `EntityCommand` builder that generates `get`, `list`, `create`, `update`, `delete`, `note`, `context` commands for any entity type
2. Entity-specific commands (e.g., `task deps`, `bug triage`, `change approve`) remain as extensions
3. This is the largest phase and can be done entity-by-entity

**Value:** Adding a new entity type requires near-zero CLI work. Currently it requires creating ~15 command files.

---

## 6. Trade-offs and Risks

### 6.1 Benefits

| Benefit | Impact |
|---------|--------|
| Adding a new entity type goes from ~2 weeks to ~2 days | Very High |
| Bug fixes to cross-cutting logic apply once | High |
| Consistent behavior guaranteed across entities | High |
| Reduced cognitive load (one pattern to understand, not 5) | High |
| Smaller codebase (~1,200+ lines eliminated) | Medium |
| Easier testing (test shared logic once) | Medium |

### 6.2 Risks

| Risk | Mitigation |
|------|------------|
| **Over-abstraction**: Making entities too generic could lose type safety | Keep typed repositories and services for entity-specific operations. Only use Entity interface for truly shared logic. |
| **Performance**: Extra interface indirection | Negligible in Go. Interface dispatch is 1-2 nanoseconds. |
| **Migration disruption**: Refactoring many files at once | Incremental phases. Each phase is self-contained. Existing typed code continues to work alongside new generic code. |
| **Loss of explicitness**: Harder to find entity-specific code | Clear naming convention: `EntityService` for shared, `EpicService` for epic-specific. IDE navigation still works. |
| **Testing complexity**: Generic tests harder to write | Table-driven tests with entity type as a parameter. Mock the Entity interface. |
| **Go's type system limitations**: No generic methods on interfaces (Go 1.18+ generics help with containers but not method signatures) | Use type assertions where needed. The Entity interface returns `interface{}` for entity-specific fields, but the typed repositories remain available for type-safe access. |

### 6.3 What NOT to Unify

Some things should remain entity-specific:

- **Task dependency graph**: Only tasks have dependencies; this is fundamental to task identity
- **Feature/Epic progress rollup**: Hierarchical aggregation is specific to the parent-child entities
- **Bug severity and triage**: Bug-specific workflow
- **ChangeCard approval and impact analysis**: ChangeCard-specific workflow
- **Task completion metadata**: Tests passed, files changed, etc.
- **Task work sessions and history**: Task-specific tracking

The Entity interface should not try to abstract these. The principle is: if only one or two entity types need it, keep it entity-specific.

---

## 7. Relationship to Existing Architecture

This proposal is compatible with and extends the existing architecture:

- **Service Layer Pattern** (E15): Entity services continue to follow the existing service design principles. The generic EntityService is an additional shared service, not a replacement.
- **Repository Pattern**: Typed repositories remain. EntityRepository is an additional interface for cross-cutting access.
- **Workflow Service**: The existing `workflow.Service.ForLevel()` pattern maps cleanly to entity types.
- **CLI Thin Wrapper Pattern** (E17): Generic CLI commands would be even thinner than current entity-specific ones.

---

## 8. Concrete Next Steps

If this proposal is approved:

1. Create a new Epic (e.g., E21) for "Entity Polymorphism"
2. Start with Phase 1 (Foundation) -- define Entity interface, implement on all models
3. Validate the approach with Phase 2 (NoteService refactor) -- this is the smallest cross-cutting service
4. If successful, proceed with Phases 3-6

Phase 1 can be done in a single development session and provides the foundation for all subsequent work with zero risk to existing functionality.

---

## 9. Appendix: File Inventory

### Files That Would Be Modified

**Phase 1 (Foundation):**
- `internal/models/entity.go` (new)
- `internal/models/epic.go` (add interface methods)
- `internal/models/feature.go` (add interface methods)
- `internal/models/task.go` (add interface methods)
- `internal/models/bug.go` (add interface methods)
- `internal/models/change_card.go` (add interface methods)

**Phase 2 (Cross-Cutting):**
- `internal/services/note_service.go`
- `internal/services/context_service.go`
- `internal/services/resume_service.go`
- `internal/cli/services_global.go`

**Phase 3 (Status Transitions):**
- `internal/services/entity_service.go` (new)
- `internal/services/epic_service.go`
- `internal/services/feature_service.go`
- `internal/services/task_service.go`
- `internal/services/bug_service.go`
- `internal/services/change_card_service.go`

### Files That Would Be Deleted/Consolidated (Phases 4-6)

No files are deleted in Phases 1-5. Phase 6 (CLI) could consolidate many command files but this is optional and entity-by-entity.
