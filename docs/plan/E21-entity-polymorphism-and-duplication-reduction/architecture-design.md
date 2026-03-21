# Architecture Design: E21 Entity Polymorphism and Duplication Reduction

**Author**: Architect
**Date**: 2026-03-19
**Status**: Approved
**Epic**: E21

---

## 1. Entity Interface Definition

### Location: `internal/models/entity.go`

```go
package models

import "time"

// Entity is the polymorphic interface implemented by all domain entity types.
// It provides accessor methods for the 10 shared fields common to all entities,
// plus mutation methods for status and context data, and a validation method.
//
// The interface is additive -- existing direct field access (e.g., epic.Key)
// continues to work. The interface is used only by cross-cutting services
// (NoteService, ContextService, ResumeService, EntityService) that need to
// operate on entities polymorphically.
type Entity interface {
    GetID() int64
    GetKey() string
    GetTitle() string
    GetSlug() string
    GetEntityType() EntityType
    GetStatus() string
    SetStatus(status string)
    GetDescription() string
    GetFilePath() string
    GetContextData() *string
    SetContextData(data *string)
    GetCreatedAt() time.Time
    GetUpdatedAt() time.Time
    Validate() error
}
```

### Implementation Pattern (per model)

Each model gets a set of accessor methods. Example for Epic:

```go
// internal/models/epic.go (additions only -- existing code unchanged)

func (e *Epic) GetID() int64                 { return e.ID }
func (e *Epic) GetKey() string               { return e.Key }
func (e *Epic) GetTitle() string             { return e.Title }
func (e *Epic) GetSlug() string {
    if e.Slug != nil { return *e.Slug }
    return ""
}
func (e *Epic) GetEntityType() EntityType    { return EntityTypeEpic }
func (e *Epic) GetStatus() string            { return string(e.Status) }
func (e *Epic) SetStatus(status string)      { e.Status = EpicStatus(status) }
func (e *Epic) GetDescription() string {
    if e.Description != nil { return *e.Description }
    return ""
}
func (e *Epic) GetFilePath() string {
    if e.FilePath != nil { return *e.FilePath }
    return ""
}
func (e *Epic) GetContextData() *string      { return e.ContextData }
func (e *Epic) SetContextData(data *string)  { e.ContextData = data }
func (e *Epic) GetCreatedAt() time.Time      { return e.CreatedAt }
func (e *Epic) GetUpdatedAt() time.Time      { return e.UpdatedAt }
// Validate() already exists on Epic.
```

### Compile-Time Interface Satisfaction Checks

```go
// internal/models/entity.go
var (
    _ Entity = (*Epic)(nil)
    _ Entity = (*Feature)(nil)
    _ Entity = (*Task)(nil)
    _ Entity = (*Bug)(nil)
    _ Entity = (*ChangeCard)(nil)
)
```

### ChangeCard Type Normalization (Prerequisite)

Before implementing the Entity interface, ChangeCard must be normalized:

```go
// internal/models/change_card.go -- BEFORE
type ChangeCard struct {
    // ...
    Slug     string  `json:"slug,omitempty" db:"slug"`      // string
    FilePath string  `json:"file_path,omitempty" db:"file_path"` // string
    // ...
}

// internal/models/change_card.go -- AFTER
type ChangeCard struct {
    // ...
    Slug     *string `json:"slug,omitempty" db:"slug"`      // *string (matches other entities)
    FilePath *string `json:"file_path,omitempty" db:"file_path"` // *string (matches other entities)
    // ...
}
```

Affected files:
- `internal/models/change_card.go` -- field type change
- `internal/repository/change_card_repository.go` -- scan logic update (Scan into `*string`)
- `internal/services/change_card_service.go` -- code that sets Slug/FilePath
- `internal/services/change_card_service_test.go` -- test updates
- `internal/cli/commands/change*.go` -- any direct field access

---

## 2. EntityRepository Adapter Pattern

### Location: `internal/services/entity_repository.go`

The EntityRepository interface is defined at the consumer side (services layer), not in the repository layer. This follows the project's established "define interfaces at point of use" pattern.

```go
package services

import (
    "context"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityRepository provides polymorphic data access for any entity type.
// It wraps typed repositories to support cross-cutting operations.
//
// Implementations are thin adapters that delegate to typed repositories
// (EpicRepository, FeatureRepository, etc.) and perform type assertions
// where needed.
type EntityRepository interface {
    // GetByKey retrieves an entity by its key. Returns the entity as a
    // models.Entity interface value.
    GetByKey(ctx context.Context, key string) (models.Entity, error)

    // GetByID retrieves an entity by its database ID.
    GetByID(ctx context.Context, id int64) (models.Entity, error)

    // UpdateStatus updates the status field of an entity.
    UpdateStatus(ctx context.Context, id int64, status string) error

    // Update persists all fields of the entity to the database.
    // The entity parameter must be the correct concrete type for this adapter.
    Update(ctx context.Context, entity models.Entity) error

    // GetContextData retrieves the context_data JSON string for an entity.
    GetContextData(ctx context.Context, id int64) (*string, error)

    // UpdateContextData updates the context_data JSON string for an entity.
    UpdateContextData(ctx context.Context, id int64, data *string) error
}
```

### Adapter Implementation Pattern

Each adapter wraps an existing typed repository. Example for Epic:

```go
// internal/services/epic_repo_adapter.go

package services

import (
    "context"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicRepositoryAdapter wraps EpicRepository to satisfy EntityRepository.
type EpicRepositoryAdapter struct {
    repo EpicRepository // existing typed interface
}

func NewEpicRepositoryAdapter(repo EpicRepository) *EpicRepositoryAdapter {
    return &EpicRepositoryAdapter{repo: repo}
}

func (a *EpicRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
    return a.repo.GetByKey(ctx, key)
}

func (a *EpicRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
    return a.repo.GetByID(ctx, id)
}

func (a *EpicRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
    epic, err := a.repo.GetByID(ctx, id)
    if err != nil {
        return fmt.Errorf("failed to get epic for status update: %w", err)
    }
    epic.Status = models.EpicStatus(status)
    return a.repo.Update(ctx, epic)
}

func (a *EpicRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
    epic, ok := entity.(*models.Epic)
    if !ok {
        return fmt.Errorf("EpicRepositoryAdapter.Update: expected *models.Epic, got %T", entity)
    }
    return a.repo.Update(ctx, epic)
}

func (a *EpicRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
    return a.repo.GetContextData(ctx, id)
}

func (a *EpicRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
    return a.repo.UpdateContextData(ctx, id, data)
}
```

**Note on Task and Bug adapters**: These entities lack dedicated `GetContextData`/`UpdateContextData` repository methods. Their adapters implement these by calling `GetByID`, accessing the `ContextData` field, and calling `Update`:

```go
func (a *TaskRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
    task, err := a.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    return task.ContextData, nil
}

func (a *TaskRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
    task, err := a.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }
    task.ContextData = data
    return a.repo.Update(ctx, task)
}
```

---

## 3. EntityRegistry Design

### Location: `internal/services/entity_registry.go`

```go
package services

import (
    "fmt"
    "sync"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityRegistry maps EntityType to its EntityRepository adapter.
// Cross-cutting services use the registry to perform polymorphic operations
// without maintaining per-entity repository fields or switch statements.
//
// The registry is initialized once at application startup (or lazily on
// first access for CLI commands) and shared across all cross-cutting services.
type EntityRegistry struct {
    mu    sync.RWMutex
    repos map[models.EntityType]EntityRepository
}

// NewEntityRegistry creates an empty registry.
func NewEntityRegistry() *EntityRegistry {
    return &EntityRegistry{
        repos: make(map[models.EntityType]EntityRepository),
    }
}

// Register associates an EntityRepository adapter with an EntityType.
// Panics if the same EntityType is registered twice (programming error).
func (r *EntityRegistry) Register(entityType models.EntityType, repo EntityRepository) {
    r.mu.Lock()
    defer r.mu.Unlock()
    if _, exists := r.repos[entityType]; exists {
        panic(fmt.Sprintf("EntityRegistry: duplicate registration for %s", entityType))
    }
    r.repos[entityType] = repo
}

// GetRepository returns the EntityRepository for the given EntityType.
// Returns an error if the EntityType is not registered.
func (r *EntityRegistry) GetRepository(entityType models.EntityType) (EntityRepository, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    repo, ok := r.repos[entityType]
    if !ok {
        return nil, fmt.Errorf("no repository registered for entity type %q", entityType)
    }
    return repo, nil
}

// MustGetRepository returns the EntityRepository or panics.
// Use only in initialization code where missing registration is a programming error.
func (r *EntityRegistry) MustGetRepository(entityType models.EntityType) EntityRepository {
    repo, err := r.GetRepository(entityType)
    if err != nil {
        panic(err)
    }
    return repo
}

// RegisteredTypes returns all registered entity types.
func (r *EntityRegistry) RegisteredTypes() []models.EntityType {
    r.mu.RLock()
    defer r.mu.RUnlock()
    types := make([]models.EntityType, 0, len(r.repos))
    for t := range r.repos {
        types = append(types, t)
    }
    return types
}
```

### Registry Initialization (CLI)

The registry is initialized lazily in `services_global.go`, replacing the current per-entity setter pattern:

```go
// internal/cli/services_global.go -- AFTER refactoring

var (
    globalRegistry     *services.EntityRegistry
    registryOnce       sync.Once
)

func GetEntityRegistry() *services.EntityRegistry {
    registryOnce.Do(func() {
        db, err := GetDB(context.Background())
        if err != nil {
            panic(fmt.Sprintf("failed to get database for EntityRegistry: %v", err))
        }

        globalRegistry = services.NewEntityRegistry()
        globalRegistry.Register(models.EntityTypeEpic,
            services.NewEpicRepositoryAdapter(repository.NewEpicRepository(db)))
        globalRegistry.Register(models.EntityTypeFeature,
            services.NewFeatureRepositoryAdapter(repository.NewFeatureRepository(db)))
        globalRegistry.Register(models.EntityTypeTask,
            services.NewTaskRepositoryAdapter(repository.NewTaskRepository(db)))
        globalRegistry.Register(models.EntityTypeBug,
            services.NewBugRepositoryAdapter(repository.NewBugRepository(db)))
        globalRegistry.Register(models.EntityTypeChange,
            services.NewChangeCardRepositoryAdapter(repository.NewChangeCardRepository(db)))
    })
    return globalRegistry
}
```

---

## 4. EntityService Composition Pattern

### Location: `internal/services/entity_service.go`

The EntityService contains shared cross-cutting logic that was previously duplicated across 5 entity services.

```go
package services

import (
    "context"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TransitionFeatures controls which aspects of the full TransitionStatus
// logic are active for a given entity type. This allows Bug and ChangeCard
// to opt out of features they do not use (backward detection, rejection
// notes, child counting) while sharing the core transition validation.
type TransitionFeatures struct {
    // DetectBackward enables backward transition detection and reason
    // requirement. Set to true for Epic, Feature, Task.
    // Set to false for Bug, ChangeCard (simpler workflow).
    DetectBackward bool

    // CreateRejectionNotes enables creation of rejection notes for
    // backward or forced transitions. Requires DetectBackward.
    CreateRejectionNotes bool

    // CountChildren enables child entity counting in the TransitionResult.
    // Set to true for Epic (counts features), Feature (counts tasks).
    // Set to false for Task, Bug, ChangeCard (leaf/standalone entities).
    CountChildren bool

    // ResolveOrchestratorAction enables orchestrator action resolution
    // from workflow metadata. Set to true for all entity types.
    ResolveOrchestratorAction bool
}

// DefaultTransitionFeatures returns the full feature set used by
// Epic, Feature, and Task.
func DefaultTransitionFeatures() TransitionFeatures {
    return TransitionFeatures{
        DetectBackward:            true,
        CreateRejectionNotes:      true,
        CountChildren:             false, // set per entity type
        ResolveOrchestratorAction: true,
    }
}

// SimpleTransitionFeatures returns the reduced feature set used by
// Bug and ChangeCard -- validation and status update only.
func SimpleTransitionFeatures() TransitionFeatures {
    return TransitionFeatures{
        DetectBackward:            false,
        CreateRejectionNotes:      false,
        CountChildren:             false,
        ResolveOrchestratorAction: true,
    }
}

// EntityService provides shared cross-cutting operations for all entity types.
// Entity-specific services compose EntityService and delegate shared logic
// to it while retaining entity-specific pre/post hooks.
type EntityService struct {
    workflowSvc *workflow.Service
    noteRepo    RejectionNoteCreator // interface for creating rejection notes
}

// RejectionNoteCreator is the minimal interface needed by EntityService
// for creating rejection notes during backward transitions.
type RejectionNoteCreator interface {
    CreateRejectionNote(ctx context.Context, entityType string, entityID int64,
        fromID int64, fromStatus, toStatus, reason, agent string, docPath interface{}) error
}

// NewEntityService creates an EntityService with shared dependencies.
func NewEntityService(workflowSvc *workflow.Service, noteRepo RejectionNoteCreator) *EntityService {
    return &EntityService{
        workflowSvc: workflowSvc,
        noteRepo:    noteRepo,
    }
}

// TransitionStatus performs a status transition on any entity via its
// EntityRepository adapter. The features parameter controls which aspects
// of the transition logic are active.
//
// Steps performed:
//  1. Get entity by key via repo
//  2. Extract current status
//  3. Validate transition (unless forced)
//  4. Normalize target status (unless forced)
//  5. Enforce reason requirement for forced transitions
//  6. [opt-in] Detect backward transition and require reason
//  7. Update entity status via repo
//  8. [opt-in] Create rejection note for backward/forced transitions
//  9. Resolve orchestrator action
//  10. Build and return TransitionResult
func (s *EntityService) TransitionStatus(
    ctx context.Context,
    repo EntityRepository,
    entityType string,
    key string,
    targetStatus string,
    opts TransitionOptions,
    features TransitionFeatures,
) (*TransitionResult, error) {
    entity, err := repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get %s: %w", entityType, err)
    }
    if entity == nil {
        return nil, fmt.Errorf("%s not found: %s", entityType, key)
    }

    currentStatus := entity.GetStatus()

    // Validate transition (unless forced)
    if !opts.Force {
        if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
            return nil, err
        }
        targetStatus = s.workflowSvc.NormalizeStatus(targetStatus)
    }

    // Enforce reason for forced transitions
    if opts.Force && opts.Reason == "" {
        return nil, ErrForceReasonRequired
    }

    // Backward detection (opt-in)
    var isBackward bool
    if features.DetectBackward {
        isBackward, err = s.workflowSvc.IsBackwardTransition(currentStatus, targetStatus)
        if err != nil {
            if !opts.Force {
                return nil, fmt.Errorf("could not determine transition direction: %w", err)
            }
            isBackward = false
        }
        if isBackward && !opts.Force {
            wf := s.workflowSvc.GetWorkflow()
            requireReason := wf == nil || wf.RequireRejectionReason
            if requireReason && opts.Reason == "" {
                return nil, &BackwardReasonError{FromStatus: currentStatus, ToStatus: targetStatus}
            }
        }
    }

    // Update status
    if err := repo.UpdateStatus(ctx, entity.GetID(), targetStatus); err != nil {
        return nil, fmt.Errorf("failed to update %s status: %w", entityType, err)
    }

    // Rejection note (opt-in)
    if features.CreateRejectionNotes && (isBackward || opts.Force) && opts.Reason != "" && s.noteRepo != nil {
        _ = s.noteRepo.CreateRejectionNote(ctx, entityType, entity.GetID(),
            0, currentStatus, targetStatus,
            opts.Reason, opts.Agent, opts.DocumentPath)
    }

    // Resolve orchestrator action (opt-in)
    var action *config.PopulatedAction
    if features.ResolveOrchestratorAction {
        action = s.resolveActionForEntity(entity, targetStatus)
    }

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
        // ChildCount is set by the calling entity service if features.CountChildren is true
    }, nil
}

// resolveActionForEntity resolves the orchestrator action for an entity's
// target status. This is the shared implementation that replaces 5 identical
// resolveAction methods across entity services.
func (s *EntityService) resolveActionForEntity(entity models.Entity, status string) *config.PopulatedAction {
    wf := s.workflowSvc.GetWorkflow()
    if wf == nil || wf.StatusMetadata == nil {
        return nil
    }
    meta, exists := wf.StatusMetadata[status]
    if !exists || meta.OrchestratorAction == nil {
        return nil
    }

    // Build placeholders from Entity interface
    placeholders := EntityPlaceholders(entity)

    return &config.PopulatedAction{
        Action:      meta.OrchestratorAction.Action,
        AgentType:   meta.OrchestratorAction.AgentType,
        Skills:      meta.OrchestratorAction.Skills,
        Instruction: meta.OrchestratorAction.PopulateTemplate(placeholders),
    }
}

// EntityPlaceholders generates template placeholders from the Entity interface.
// Entity-specific placeholder functions extend this base set.
func EntityPlaceholders(entity models.Entity) map[string]string {
    return map[string]string{
        "entity_type":  string(entity.GetEntityType()),
        "entity_key":   entity.GetKey(),
        "entity_title": entity.GetTitle(),
        "entity_slug":  entity.GetSlug(),
        "status":       entity.GetStatus(),
    }
}
```

---

## 5. Migration Strategy: Entity-Specific Services Compose Shared Service

Entity-specific services retain their typed repositories for entity-specific operations. They add the shared `EntityService` as a composed dependency and delegate cross-cutting operations to it.

### Before (EpicService)

```go
type EpicService struct {
    repo        EpicRepository
    workflowSvc *workflow.Service
    noteRepo    EpicNoteRepository
    featureRepo EpicFeatureCounter
    // ...
}

func (s *EpicService) TransitionStatus(ctx, key, target string, opts TransitionOptions) (*TransitionResult, error) {
    // 80 lines of transition logic duplicated with Feature and Task
}

func (s *EpicService) resolveAction(ctx, epic, status) *config.PopulatedAction {
    // 18 lines duplicated across all 5 services
}
```

### After (EpicService)

```go
type EpicService struct {
    repo        EpicRepository      // typed repo for epic-specific queries
    entitySvc   *EntityService      // shared cross-cutting logic
    entityRepo  EntityRepository    // adapter for polymorphic operations
    featureRepo EpicFeatureCounter  // for child counting
    // workflowSvc removed -- EntityService owns it
    // noteRepo removed -- EntityService owns it
}

func NewEpicService(
    repo EpicRepository,
    entitySvc *EntityService,
    entityRepo EntityRepository,
    featureRepo EpicFeatureCounter,
) *EpicService {
    return &EpicService{
        repo:        repo,
        entitySvc:   entitySvc,
        entityRepo:  entityRepo,
        featureRepo: featureRepo,
    }
}

func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
    features := DefaultTransitionFeatures()
    features.CountChildren = true // Epic counts features

    result, err := s.entitySvc.TransitionStatus(ctx, s.entityRepo, "epic", epicKey, targetStatus, opts, features)
    if err != nil {
        return nil, err
    }

    // Entity-specific post-hook: count child features
    if s.featureRepo != nil {
        features, listErr := s.featureRepo.ListByEpic(ctx, result.EntityKey)
        if listErr == nil {
            result.ChildCount = len(features)
        }
    }

    return result, nil
}
```

### Bug/ChangeCard Migration (Simple Pattern)

Bug and ChangeCard use `SimpleTransitionFeatures()` to opt out of backward detection, rejection notes, and child counting:

```go
func (s *BugService) SetBugStatus(ctx context.Context, key string, status string, force bool) (*models.Bug, error) {
    // Validate status is valid
    if err := s.entitySvc.workflowSvc.ValidateStatus(status); err != nil {
        return nil, fmt.Errorf("invalid bug status %q: %w", status, err)
    }

    opts := TransitionOptions{Force: force}
    if force {
        opts.Reason = "forced status change" // Bug does not require user-provided reason
    }

    result, err := s.entitySvc.TransitionStatus(ctx, s.entityRepo, "bug", key, status, opts, SimpleTransitionFeatures())
    if err != nil {
        return nil, err
    }

    // Reload to return typed Bug
    bug, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, err
    }
    _ = result // result available for orchestrator action if needed
    return bug, nil
}
```

**Alternative**: Bug and ChangeCard can also keep their current simpler methods and only adopt the shared pattern for operations that are truly duplicated (like `resolveAction`). This is the lower-risk approach and is recommended for the initial implementation. Full TransitionStatus unification for Bug/ChangeCard can be deferred to a follow-up if the value is clear.

---

## 6. NoteService/ContextService Refactoring Pattern

### Before (NoteService)

```go
type NoteService struct {
    noteRepo       NoteEntityNoteRepository
    epicRepo       NoteEpicRepository       // 1 of 5
    featureRepo    NoteFeatureRepository    // 2 of 5
    taskRepo       NoteTaskRepository       // 3 of 5
    changeCardRepo NoteChangeCardRepository // 4 of 5
    bugRepo        NoteBugRepository        // 5 of 5
}

func (s *NoteService) resolveEntityID(ctx, entityType, key) (int64, error) {
    switch entityType {
    case models.EntityTypeEpic:
        epic, err := s.epicRepo.GetByKey(ctx, key)
        // ...
        return epic.ID, nil
    case models.EntityTypeFeature:
        feature, err := s.featureRepo.GetByKey(ctx, key)
        // ...
        return feature.ID, nil
    // ... 3 more branches
    }
}
```

### After (NoteService)

```go
type NoteService struct {
    noteRepo NoteEntityNoteRepository
    registry *EntityRegistry
}

func NewNoteService(noteRepo NoteEntityNoteRepository, registry *EntityRegistry) *NoteService {
    return &NoteService{noteRepo: noteRepo, registry: registry}
}

func (s *NoteService) resolveEntityID(ctx context.Context, entityType models.EntityType, key string) (int64, error) {
    repo, err := s.registry.GetRepository(entityType)
    if err != nil {
        return 0, fmt.Errorf("unsupported entity type %q: %w", entityType, err)
    }
    entity, err := repo.GetByKey(ctx, key)
    if err != nil {
        return 0, fmt.Errorf("failed to get %s %s: %w", entityType, key, err)
    }
    return entity.GetID(), nil
}

func (s *NoteService) GetEntityDetails(ctx context.Context, entityType models.EntityType, key string) (*NoteEntityDetails, error) {
    repo, err := s.registry.GetRepository(entityType)
    if err != nil {
        return nil, fmt.Errorf("unsupported entity type %q: %w", entityType, err)
    }
    entity, err := repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get %s %s: %w", entityType, key, err)
    }
    return &NoteEntityDetails{
        Key:   entity.GetKey(),
        Title: entity.GetTitle(),
    }, nil
}
```

**Lines saved**: ~50 lines (5 repo interfaces removed, 2 switch statements of 5 branches each replaced by 2 registry lookups, 5 setter methods removed).

The same pattern applies to **ContextService** and **ResumeService**.

---

## 7. File Organization

### New Files

| File | Purpose | Size Estimate |
|------|---------|---------------|
| `internal/models/entity.go` | Entity interface, compile-time checks | ~80 lines |
| `internal/services/entity_repository.go` | EntityRepository interface | ~30 lines |
| `internal/services/entity_registry.go` | EntityRegistry struct | ~60 lines |
| `internal/services/entity_service.go` | Shared EntityService with TransitionStatus, resolveAction, EntityPlaceholders | ~200 lines |
| `internal/services/epic_repo_adapter.go` | EpicRepositoryAdapter | ~50 lines |
| `internal/services/feature_repo_adapter.go` | FeatureRepositoryAdapter | ~50 lines |
| `internal/services/task_repo_adapter.go` | TaskRepositoryAdapter | ~60 lines |
| `internal/services/bug_repo_adapter.go` | BugRepositoryAdapter | ~60 lines |
| `internal/services/change_repo_adapter.go` | ChangeCardRepositoryAdapter | ~60 lines |

### Modified Files

| File | Changes |
|------|---------|
| `internal/models/epic.go` | Add ~14 accessor methods |
| `internal/models/feature.go` | Add ~14 accessor methods |
| `internal/models/task.go` | Add ~14 accessor methods |
| `internal/models/bug.go` | Add ~14 accessor methods |
| `internal/models/change_card.go` | Normalize Slug/FilePath to `*string`, add ~14 accessor methods |
| `internal/services/note_service.go` | Remove 5 repo interfaces, 5 setter methods, 2 switch statements; accept EntityRegistry |
| `internal/services/context_service.go` | Remove 5 repo interfaces, 5 setter methods, 2 switch statements; accept EntityRegistry |
| `internal/services/resume_service.go` | Remove setter methods; accept EntityRegistry |
| `internal/services/epic_service.go` | Delegate TransitionStatus and resolveAction to EntityService |
| `internal/services/feature_service.go` | Delegate TransitionStatus and resolveAction to EntityService |
| `internal/services/task_service.go` | Delegate TransitionStatus and resolveAction to EntityService |
| `internal/services/bug_service.go` | Delegate resolveAction to EntityService (TransitionStatus delegation optional) |
| `internal/services/change_card_service.go` | Delegate resolveAction to EntityService (TransitionStatus delegation optional) |
| `internal/cli/services_global.go` | Replace setter-based wiring with EntityRegistry initialization |

### New Test Files

| File | Purpose |
|------|---------|
| `internal/models/entity_test.go` | Interface satisfaction tests, accessor tests |
| `internal/services/entity_repository_test.go` | Adapter tests with mock repos |
| `internal/services/entity_registry_test.go` | Registry registration and lookup tests |
| `internal/services/entity_service_test.go` | TransitionStatus tests parameterized by entity type |

---

## 8. Feature Execution Order and Dependencies

```
F01: Entity Interface Foundation
 |   [Prereq: ChangeCard type normalization]
 |   [Define Entity interface, implement on all 5 models]
 |   [Create EntityRepository interface + 5 adapters]
 |   [Create EntityRegistry]
 |
 +---> F02: Cross-Cutting Service Unification
 |         [Refactor NoteService -> use EntityRegistry]
 |         [Refactor ContextService -> use EntityRegistry]
 |         [Refactor ResumeService -> use EntityRegistry]
 |         [Update CLI accessors in services_global.go]
 |
 +---> F03: Status Transition Unification (after E15 confirmation)
 |         [Create EntityService with TransitionStatus]
 |         [Refactor EpicService -> delegate to EntityService]
 |         [Refactor FeatureService -> delegate to EntityService]
 |         [Refactor TaskService -> delegate to EntityService]
 |         [Optionally: Bug/ChangeCard delegation]
 |
 +---> F04: Document Operations Unification
 |         [Move LinkDocument/UnlinkDocument to EntityService]
 |         [Refactor 5 entity services to delegate]
 |
 +---> F05: Template Placeholder Unification
           [Create EntityPlaceholders base function]
           [Refactor per-entity placeholder functions]
```

F01 is the critical path. F02-F05 can proceed in parallel after F01 completes, though F02 and F03 should be prioritized for maximum value delivery.

---

## 9. Design Decisions Summary

| Decision | Rationale | Alternative Considered |
|----------|-----------|------------------------|
| Interface-based polymorphism (not generics) | Idiomatic Go, incremental, proven in large codebases | Go generics: equivalent but more complex syntax |
| Accessor methods (not struct embedding) | Non-breaking, preserves direct field access | BaseEntity embedding: breaks `epic.Key` access pattern |
| Registry as a map (not service locator) | Simple, O(1) lookup, compile-time type-safe keys | DI framework: over-engineering for 5 entity types |
| TransitionFeatures config (not interface methods) | Explicit opt-in, no behavioral surprise | Interface method `SupportsBackward() bool`: implicit, harder to discover |
| EntityRepository at services layer | Consumer-side interface definition per project convention | Repository layer: would couple repos to service concerns |
| Adapters wrap typed repos (not replace them) | Zero risk to existing code, incremental adoption | Generic repository: high risk, requires SQL rewrite |

---

*References: All code examples derived from actual codebase patterns at commit 7299cc1 on the file-path-updates branch.*
