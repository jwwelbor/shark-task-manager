# F02: Cross-Cutting Service Unification -- Implementation Architecture

**Feature**: E21-F02
**Author**: Architect
**Date**: 2026-03-19
**Status**: Approved
**Tier**: STANDARD
**Depends on**: F01 (Entity Interface Foundation) -- must be complete and merged

This document is the developer implementation guide for F02. It specifies how NoteService, ContextService, and ResumeService are refactored to use the EntityRegistry created in F01, and how `services_global.go` is updated to wire the registry.

---

## 1. Scope and Constraints

**In scope:**
- Refactor NoteService to use EntityRegistry (eliminate 2 switch statements, 5 per-entity interfaces, 2 setter methods)
- Refactor ContextService to use EntityRegistry (eliminate 2 switch statements, 5 per-entity interfaces, 2 setter methods)
- Refactor ResumeService to use EntityRegistry (eliminate 3 per-entity interfaces, 3 setter methods for bug/change/session repos; partially unify entity lookup)
- Add `GetEntityRegistry()` to `services_global.go` with `sync.Once` initialization
- Update `GetNoteService()`, `GetContextService()`, `GetResumeService()` wiring
- Update all tests to use new constructors

**Out of scope:**
- Status transition unification (F03)
- Document operations unification (F04)
- Template placeholder unification (F05)
- Entity-specific service internals (EpicService, FeatureService, TaskService)
- Database schema changes
- HTTP API handler changes

---

## 2. Key Architecture Decisions

### ADR-F02-001: NoteService and ContextService Fully Unify via Registry

**Decision**: NoteService and ContextService replace all per-entity repository fields with a single `*EntityRegistry` field. All switch-based dispatch is eliminated.

**Rationale**: Both services use only `GetByKey`, `GetByID`, `GetContextData`, and `UpdateContextData` -- all methods that the `EntityRepository` interface provides. The mapping is 1:1 with no entity-specific logic in any branch.

**Consequence**: Adding a 6th entity type requires zero changes to NoteService or ContextService.

### ADR-F02-002: ResumeService Partially Unifies -- Entity Lookup Only

**Decision**: ResumeService replaces its per-entity repository fields for simple entity lookup (Bug, ChangeCard) with the registry, but retains typed repository fields for entity-specific queries (ListByEpic, ListByFeature, ListByFeature, GetContextData with typed repos, session repo).

**Rationale**: ResumeService has entity-specific return types (`EpicResumeContext`, `FeatureResumeContext`, `TaskResumeContext`, `BugResumeContext`, `ChangeResumeContext`) and performs entity-specific aggregation (features list for epic, task rollups for feature, work sessions for task). These are fundamentally different operations that cannot be expressed through the generic `EntityRepository` interface. The `GetBugResume` and `GetChangeResume` methods only need `GetByKey` (which the registry provides) plus notes lookup (which already uses the note repo). The epic, feature, and task resume methods need typed repos for `ListByEpic`, `ListByFeature`, and session stats.

**Consequence**: Adding a 6th entity type still requires adding a new `Get<Entity>Resume` method to ResumeService (because each entity type has a unique resume context shape), but the entity lookup and note retrieval are handled by the registry. The per-entity repository interfaces (`ResumeBugRepository`, `ResumeChangeCardRepository`) are removed. The interfaces for Epic, Feature, and Task repos remain because they expose methods beyond what EntityRepository provides.

### ADR-F02-003: Registry Initialized with sync.Once in services_global.go

**Decision**: A single `GetEntityRegistry()` function in `services_global.go` initializes the registry lazily using `sync.Once`, matching the existing pattern for `GetDB()` and workflow service initialization.

**Rationale**: Consistent with established CLI wiring patterns. The registry is expensive to create (requires DB handle and 5 repository + adapter constructions) and should be created at most once per CLI invocation.

### ADR-F02-004: ResumeService Session Repo Stays as Setter

**Decision**: `ResumeService.SetSessionRepo()` is retained (not replaced by registry) because the work session repository is task-specific and has no EntityRepository adapter.

**Rationale**: Work sessions are associated with tasks only, not with all entity types. They do not implement the Entity interface and have no EntityRepository adapter. Forcing them into the registry would add complexity without benefit.

### ADR-F02-005: Constructor Nil-Check Panics

**Decision**: All three refactored constructors panic if the `registry` parameter is nil.

**Rationale**: A nil registry is a programming/wiring error that should be caught immediately at startup, not at the first note/context/resume operation. This matches the existing pattern where `services_global.go` panics on DB failure.

---

## 3. NoteService Refactoring

### 3.1 Before (Current State -- 227 lines)

```go
// 5 per-entity repository interfaces (lines 19-47, ~29 lines)
type NoteEpicRepository interface { ... }
type NoteFeatureRepository interface { ... }
type NoteTaskRepository interface { ... }
type NoteChangeCardRepository interface { ... }
type NoteBugRepository interface { ... }

// Struct with 5+1 repo fields (lines 101-108)
type NoteService struct {
    noteRepo       NoteEntityNoteRepository
    epicRepo       NoteEpicRepository
    featureRepo    NoteFeatureRepository
    taskRepo       NoteTaskRepository
    changeCardRepo NoteChangeCardRepository
    bugRepo        NoteBugRepository
}

// Constructor taking 4 required repos (line 121-128)
func NewNoteService(noteRepo, epicRepo, featureRepo, taskRepo) *NoteService

// 2 setter methods (lines 111-118)
func (s *NoteService) SetChangeCardRepo(repo NoteChangeCardRepository)
func (s *NoteService) SetBugRepo(repo NoteBugRepository)

// 5-branch switch: resolveEntityID (lines 186-227)
// 5-branch switch: GetEntityDetails (lines 57-98)
```

### 3.2 After (Target State -- ~120 lines estimated)

```go
// NoteEntityNoteRepository stays (unchanged -- this is the note-specific repo, not an entity repo)
type NoteEntityNoteRepository interface {
    Create(ctx context.Context, note *models.EntityNote) error
    GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
    GetByEntityAndType(ctx context.Context, entityType models.EntityType, entityID int64, noteTypes []string) ([]*models.EntityNote, error)
    Search(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error)
    SearchWithTimePeriod(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error)
}

// NoteEntityDetails stays (unchanged)
type NoteEntityDetails struct {
    Key  string
    Name string
}

// NoteService -- 2 fields instead of 6
type NoteService struct {
    noteRepo NoteEntityNoteRepository
    registry *EntityRegistry
}

// Constructor -- 2 params instead of 4+setters
func NewNoteService(noteRepo NoteEntityNoteRepository, registry *EntityRegistry) *NoteService {
    if registry == nil {
        panic("NoteService: EntityRegistry must not be nil")
    }
    return &NoteService{
        noteRepo: noteRepo,
        registry: registry,
    }
}

// resolveEntityID -- 6 lines instead of 42
func (s *NoteService) resolveEntityID(ctx context.Context, entityType models.EntityType, key string) (int64, error) {
    repo, err := s.registry.GetRepository(entityType)
    if err != nil {
        return 0, fmt.Errorf("failed to resolve entity for note operation: %w", err)
    }
    entity, err := repo.GetByKey(ctx, key)
    if err != nil {
        return 0, fmt.Errorf("%s not found: %s: %w", entityType, key, err)
    }
    return entity.GetID(), nil
}

// GetEntityDetails -- 8 lines instead of 42
func (s *NoteService) GetEntityDetails(ctx context.Context, entityType models.EntityType, entityID int64) *NoteEntityDetails {
    repo, err := s.registry.GetRepository(entityType)
    if err != nil {
        return nil
    }
    entity, err := repo.GetByID(ctx, entityID)
    if err != nil {
        return nil
    }
    return &NoteEntityDetails{
        Key:  entity.GetKey(),
        Name: entity.GetTitle(),
    }
}

// AddNote, ListNotes, SearchNotes, SearchNotesWithTimePeriod -- UNCHANGED
// (they call resolveEntityID which is now registry-based)
```

### 3.3 Lines Removed

| Item | Lines Removed |
|------|---------------|
| 5 per-entity repository interfaces | -29 |
| 4 extra struct fields | -4 |
| 2 setter methods | -8 |
| `resolveEntityID` switch branches | -36 (replaced by 6-line registry call) |
| `GetEntityDetails` switch branches | -34 (replaced by 8-line registry call) |
| Constructor simplification | -4 |
| **Total** | **~107 lines removed** (net after adding 14 lines for registry-based implementations) |

---

## 4. ContextService Refactoring

### 4.1 Before (Current State -- 308 lines)

```go
// 5 per-entity repository interfaces (lines 11-41, ~31 lines)
type ContextEpicRepository interface { ... }
type ContextFeatureRepository interface { ... }
type ContextTaskRepository interface { ... }
type ContextBugRepository interface { ... }
type ContextChangeCardRepository interface { ... }

// Struct with 5 repo fields (lines 47-53)
type ContextService struct {
    epicRepo, featureRepo, taskRepo, bugRepo, changeCardRepo
}

// Constructor taking 3 required repos + 2 setters (lines 56-72)
func NewContextService(epicRepo, featureRepo, taskRepo) *ContextService
func (s *ContextService) SetBugRepo(repo)
func (s *ContextService) SetChangeCardRepo(repo)

// 5-branch switch: getContextJSON (lines 137-178)
// 5-branch switch: setContextJSON (lines 181-224)
```

### 4.2 After (Target State -- ~170 lines estimated)

```go
// Bug type alias stays (line 44, unchanged)
type Bug = models.Bug

// ContextService -- 1 field instead of 5
type ContextService struct {
    registry *EntityRegistry
}

// Constructor -- 1 param instead of 3+setters
func NewContextService(registry *EntityRegistry) *ContextService {
    if registry == nil {
        panic("ContextService: EntityRegistry must not be nil")
    }
    return &ContextService{
        registry: registry,
    }
}

// getContextJSON -- 8 lines instead of 42
func (s *ContextService) getContextJSON(ctx context.Context, entityType models.EntityType, entityKey string) (*string, error) {
    repo, err := s.registry.GetRepository(entityType)
    if err != nil {
        return nil, fmt.Errorf("unsupported entity type for context: %w", err)
    }
    entity, err := repo.GetByKey(ctx, entityKey)
    if err != nil {
        return nil, fmt.Errorf("%s not found: %s: %w", entityType, entityKey, err)
    }
    return repo.GetContextData(ctx, entity.GetID())
}

// setContextJSON -- 8 lines instead of 44
func (s *ContextService) setContextJSON(ctx context.Context, entityType models.EntityType, entityKey string, contextJSON *string) error {
    repo, err := s.registry.GetRepository(entityType)
    if err != nil {
        return fmt.Errorf("unsupported entity type for context: %w", err)
    }
    entity, err := repo.GetByKey(ctx, entityKey)
    if err != nil {
        return fmt.Errorf("%s not found: %s: %w", entityType, entityKey, err)
    }
    return repo.UpdateContextData(ctx, entity.GetID(), contextJSON)
}

// GetContext, SetContextField, ClearContext -- UNCHANGED
// isValidContextField, updateContextField -- UNCHANGED
```

### 4.3 Lines Removed

| Item | Lines Removed |
|------|---------------|
| 5 per-entity repository interfaces | -31 |
| 4 extra struct fields | -4 |
| 2 setter methods | -8 |
| `getContextJSON` switch branches | -34 (replaced by 8-line registry call) |
| `setContextJSON` switch branches | -36 (replaced by 8-line registry call) |
| Constructor simplification | -4 |
| **Total** | **~101 lines removed** (net after adding 16 lines for registry-based implementations) |

### 4.4 Critical Behavior Preservation Note

The current ContextService has entity-specific patterns for context data:
- **Epic and Feature**: Use dedicated `GetContextData`/`UpdateContextData` repository methods
- **Task and Bug**: Use the `ContextData` field directly on the entity (get entity, read/write field, update entity)
- **ChangeCard**: Uses `UpdateContextData` for writes, but reads the field directly

After refactoring, all 5 entity types go through `EntityRepository.GetContextData()` and `EntityRepository.UpdateContextData()`. The adapter implementations handle the differences:
- `EpicRepositoryAdapter.GetContextData` -> calls `repo.GetContextData(ctx, id)`
- `TaskRepositoryAdapter.GetContextData` -> calls `repo.GetByID`, returns `task.ContextData`
- `BugRepositoryAdapter.GetContextData` -> calls `repo.GetByID`, returns `bug.ContextData`

This is already implemented in F01's adapter code. The ContextService refactoring simply delegates to these adapters via the registry -- the adapters handle the per-entity differences transparently.

---

## 5. ResumeService Refactoring

### 5.1 Before (Current State -- 403 lines)

```go
// 4 per-entity repository interfaces used for entity lookup:
type ResumeEpicRepository interface { GetByKey; GetContextData }
type ResumeFeatureRepository interface { GetByKey; GetContextData; ListByEpic }
type ResumeTaskRepository interface { GetByKey; ListByFeature; ListByEpic }
type ResumeEntityNoteRepository interface { GetByEntity }
type ResumeWorkSessionRepository interface { GetByTaskID; GetSessionStatsByTaskID; GetActiveSessionByTaskID }
type ResumeBugRepository interface { GetByKey }
type ResumeChangeCardRepository interface { GetByKey }

// Struct with 7 repo fields (lines 130-138)
type ResumeService struct {
    epicRepo, featureRepo, taskRepo, noteRepo, sessionRepo, bugRepo, changeCardRepo
}

// Constructor (4 required) + 3 setters (lines 141-163)
func NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo) *ResumeService
func (s *ResumeService) SetSessionRepo(repo)
func (s *ResumeService) SetBugRepo(repo)
func (s *ResumeService) SetChangeCardRepo(repo)
```

### 5.2 After (Target State -- ~340 lines estimated)

The refactoring is **partial** (per ADR-F02-002). Bug and ChangeCard entity lookup moves to the registry. Epic, Feature, and Task retain typed repos because they use methods beyond the EntityRepository interface.

```go
// REMOVED interfaces:
// - ResumeBugRepository (GetByKey only -- covered by EntityRepository)
// - ResumeChangeCardRepository (GetByKey only -- covered by EntityRepository)
// REMOVED setters:
// - SetBugRepo
// - SetChangeCardRepo

// RETAINED interfaces (need methods beyond EntityRepository):
type ResumeEpicRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Epic, error)
    GetContextData(ctx context.Context, epicID int64) (*string, error)
}

type ResumeFeatureRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Feature, error)
    GetContextData(ctx context.Context, featureID int64) (*string, error)
    ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
}

type ResumeTaskRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Task, error)
    ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
    ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
}

type ResumeEntityNoteRepository interface {
    GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

type ResumeWorkSessionRepository interface {
    GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error)
    GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*ResumeSessionStats, error)
    GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error)
}

// ResumeService -- registry replaces bug/change repos
type ResumeService struct {
    epicRepo    ResumeEpicRepository
    featureRepo ResumeFeatureRepository
    taskRepo    ResumeTaskRepository
    noteRepo    ResumeEntityNoteRepository
    sessionRepo ResumeWorkSessionRepository
    registry    *EntityRegistry
}

// Constructor -- 5 params (adds registry, removes nothing from required)
func NewResumeService(
    epicRepo ResumeEpicRepository,
    featureRepo ResumeFeatureRepository,
    taskRepo ResumeTaskRepository,
    noteRepo ResumeEntityNoteRepository,
    registry *EntityRegistry,
) *ResumeService {
    if registry == nil {
        panic("ResumeService: EntityRegistry must not be nil")
    }
    return &ResumeService{
        epicRepo:    epicRepo,
        featureRepo: featureRepo,
        taskRepo:    taskRepo,
        noteRepo:    noteRepo,
        registry:    registry,
    }
}

// SetSessionRepo stays (per ADR-F02-004)
func (s *ResumeService) SetSessionRepo(repo ResumeWorkSessionRepository) {
    s.sessionRepo = repo
}

// GetBugResume -- uses registry instead of bugRepo
func (s *ResumeService) GetBugResume(ctx context.Context, bugKey string) (*BugResumeContext, error) {
    repo, err := s.registry.GetRepository(models.EntityTypeBug)
    if err != nil {
        return nil, fmt.Errorf("bug support not configured: %w", err)
    }
    entity, err := repo.GetByKey(ctx, bugKey)
    if err != nil {
        return nil, fmt.Errorf("bug not found: %s: %w", bugKey, err)
    }
    bug, ok := entity.(*models.Bug)
    if !ok {
        return nil, fmt.Errorf("unexpected entity type for bug: %T", entity)
    }

    resumeCtx := &BugResumeContext{Bug: bug}
    notes, err := s.noteRepo.GetByEntity(ctx, models.EntityTypeBug, bug.ID)
    if err == nil {
        resumeCtx.Notes = notes
    }
    return resumeCtx, nil
}

// GetChangeResume -- uses registry instead of changeCardRepo
func (s *ResumeService) GetChangeResume(ctx context.Context, changeKey string) (*ChangeResumeContext, error) {
    repo, err := s.registry.GetRepository(models.EntityTypeChange)
    if err != nil {
        return nil, fmt.Errorf("change support not configured: %w", err)
    }
    entity, err := repo.GetByKey(ctx, changeKey)
    if err != nil {
        return nil, fmt.Errorf("change not found: %s: %w", changeKey, err)
    }
    card, ok := entity.(*models.ChangeCard)
    if !ok {
        return nil, fmt.Errorf("unexpected entity type for change: %T", entity)
    }

    resumeCtx := &ChangeResumeContext{ChangeCard: card}
    notes, err := s.noteRepo.GetByEntity(ctx, models.EntityTypeChange, card.ID)
    if err == nil {
        resumeCtx.Notes = notes
    }
    return resumeCtx, nil
}

// GetEpicResume, GetFeatureResume, GetTaskResume -- UNCHANGED
// (they use typed repos for ListByEpic, ListByFeature, sessions, etc.)
```

### 5.3 Type Assertion for Bug/ChangeCard Resume

The `GetBugResume` and `GetChangeResume` methods need the concrete typed entity (not `models.Entity`) because they return it in the typed resume context struct. The `EntityRepository.GetByKey` returns `models.Entity`, so a type assertion is required:

```go
entity, err := repo.GetByKey(ctx, bugKey)
bug, ok := entity.(*models.Bug)
```

This is safe because:
1. The registry maps `EntityTypeBug` to `BugRepositoryAdapter`
2. `BugRepositoryAdapter.GetByKey` returns `*models.Bug` (wrapped as `models.Entity`)
3. The type assertion will always succeed for the correct entity type

### 5.4 Lines Removed

| Item | Lines Removed |
|------|---------------|
| `ResumeBugRepository` interface | -3 |
| `ResumeChangeCardRepository` interface | -3 |
| 2 setter methods (SetBugRepo, SetChangeCardRepo) | -8 |
| bugRepo/changeCardRepo struct fields | -2 |
| `GetBugResume` nil-check simplification | -3 |
| `GetChangeResume` nil-check simplification | -3 |
| **Total** | **~22 lines removed** (net after adding ~8 lines for registry lookups + type assertions) |

---

## 6. CLI Wiring Changes (services_global.go)

### 6.1 New: GetEntityRegistry()

```go
var (
    globalRegistry *services.EntityRegistry
    registryOnce   sync.Once
)

// GetEntityRegistry returns the global EntityRegistry, initializing it if needed.
// Uses sync.Once for thread-safe lazy initialization.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
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

### 6.2 Updated: GetNoteService()

**Before (18 lines):**
```go
func GetNoteService(ctx context.Context) (*services.NoteService, error) {
    noteServiceOnce.Do(func() {
        db, err := GetDB(ctx)
        if err != nil { ... }
        noteRepo := repository.NewEntityNoteRepository(db)
        epicRepo := repository.NewEpicRepository(db)
        featureRepo := repository.NewFeatureRepository(db)
        taskRepo := repository.NewTaskRepository(db)
        svc := services.NewNoteService(noteRepo, epicRepo, featureRepo, taskRepo)
        changeCardRepo := repository.NewChangeCardRepository(db)
        svc.SetChangeCardRepo(changeCardRepo)
        bugRepo := repository.NewBugRepository(db)
        svc.SetBugRepo(bugRepo)
        globalNoteService = svc
    })
    ...
}
```

**After (10 lines):**
```go
func GetNoteService(ctx context.Context) (*services.NoteService, error) {
    noteServiceOnce.Do(func() {
        db, err := GetDB(ctx)
        if err != nil {
            noteServiceErr = fmt.Errorf("failed to get database for NoteService: %w", err)
            return
        }
        noteRepo := repository.NewEntityNoteRepository(db)
        globalNoteService = services.NewNoteService(noteRepo, GetEntityRegistry())
    })
    if noteServiceErr != nil {
        return nil, noteServiceErr
    }
    return globalNoteService, nil
}
```

**Lines saved: ~8 lines** (removed 6 repository constructions + 2 setter calls; added 1 registry call).

### 6.3 Updated: GetContextService()

**Before (16 lines):**
```go
func GetContextService(ctx context.Context) (*services.ContextService, error) {
    contextServiceOnce.Do(func() {
        db, err := GetDB(ctx)
        if err != nil { ... }
        epicRepo := repository.NewEpicRepository(db)
        featureRepo := repository.NewFeatureRepository(db)
        taskRepo := repository.NewTaskRepository(db)
        svc := services.NewContextService(epicRepo, featureRepo, taskRepo)
        bugRepo := repository.NewBugRepository(db)
        svc.SetBugRepo(bugRepo)
        changeCardRepo := repository.NewChangeCardRepository(db)
        svc.SetChangeCardRepo(changeCardRepo)
        globalContextService = svc
    })
    ...
}
```

**After (10 lines):**
```go
func GetContextService(ctx context.Context) (*services.ContextService, error) {
    contextServiceOnce.Do(func() {
        db, err := GetDB(ctx)
        if err != nil {
            contextServiceErr = fmt.Errorf("failed to get database for ContextService: %w", err)
            return
        }
        _ = db // DB accessed via registry; kept for error handling consistency
        globalContextService = services.NewContextService(GetEntityRegistry())
    })
    if contextServiceErr != nil {
        return nil, contextServiceErr
    }
    return globalContextService, nil
}
```

**Note**: ContextService no longer needs the DB handle directly. However, `GetEntityRegistry()` calls `GetDB()` internally, so the DB is initialized. The `GetContextService` function can optionally skip the explicit `GetDB` call. The implementation should just call `GetEntityRegistry()` directly and let it handle DB initialization.

**Simplified After (8 lines):**
```go
func GetContextService(ctx context.Context) (*services.ContextService, error) {
    contextServiceOnce.Do(func() {
        globalContextService = services.NewContextService(GetEntityRegistry())
    })
    return globalContextService, nil
}
```

**Lines saved: ~10 lines** (removed 5 repository constructions + 2 setter calls; replaced with 1 registry call).

**Note on error handling**: Since `GetEntityRegistry()` panics on DB failure (matching the CLI fail-fast pattern), there is no error to capture. The `contextServiceErr` variable is no longer needed. However, if the team prefers to keep the error-returning signature for consistency with the other accessor functions, the panicking behavior in `GetEntityRegistry` provides the safety net.

### 6.4 Updated: GetResumeService()

**Before (16 lines):**
```go
func GetResumeService(ctx context.Context) (*services.ResumeService, error) {
    resumeServiceOnce.Do(func() {
        db, err := GetDB(ctx)
        if err != nil { ... }
        epicRepo := repository.NewEpicRepository(db)
        featureRepo := repository.NewFeatureRepository(db)
        taskRepo := repository.NewTaskRepository(db)
        noteRepo := repository.NewEntityNoteRepository(db)
        sessionRepo := &resumeSessionAdapter{repo: repository.NewWorkSessionRepository(db)}
        svc := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo)
        svc.SetSessionRepo(sessionRepo)
        globalResumeService = svc
    })
    ...
}
```

**After (14 lines):**
```go
func GetResumeService(ctx context.Context) (*services.ResumeService, error) {
    resumeServiceOnce.Do(func() {
        db, err := GetDB(ctx)
        if err != nil {
            resumeServiceErr = fmt.Errorf("failed to get database for ResumeService: %w", err)
            return
        }
        epicRepo := repository.NewEpicRepository(db)
        featureRepo := repository.NewFeatureRepository(db)
        taskRepo := repository.NewTaskRepository(db)
        noteRepo := repository.NewEntityNoteRepository(db)
        sessionRepo := &resumeSessionAdapter{repo: repository.NewWorkSessionRepository(db)}
        svc := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo, GetEntityRegistry())
        svc.SetSessionRepo(sessionRepo)
        globalResumeService = svc
    })
    if resumeServiceErr != nil {
        return nil, resumeServiceErr
    }
    return globalResumeService, nil
}
```

**Lines saved: ~2 lines** (removed `SetBugRepo` + `SetChangeCardRepo` calls and their repo constructions; added registry param). ResumeService retains typed repos for epic/feature/task, so the savings are smaller here.

### 6.5 Updated: ResetServices()

Add registry reset:

```go
func ResetServices() {
    globalNoteService = nil
    noteServiceErr = nil
    noteServiceOnce = sync.Once{}

    globalContextService = nil
    contextServiceErr = nil
    contextServiceOnce = sync.Once{}

    globalResumeService = nil
    resumeServiceErr = nil
    resumeServiceOnce = sync.Once{}

    // Reset registry
    globalRegistry = nil
    registryOnce = sync.Once{}
}
```

### 6.6 Import Changes

Add `models` import to `services_global.go` (for `models.EntityTypeEpic`, etc.):

```go
import (
    // existing imports...
    "github.com/jwwelbor/shark-task-manager/internal/models"
)
```

### 6.7 Total Lines Saved in services_global.go

| Function | Lines Before | Lines After | Saved |
|----------|-------------|-------------|-------|
| GetNoteService | 18 | 10 | 8 |
| GetContextService | 16 | 8 | 8 |
| GetResumeService | 16 | 14 | 2 |
| GetEntityRegistry (new) | 0 | 18 | -18 |
| ResetServices (registry) | 0 | 3 | -3 |
| **Total** | **50** | **53** | **-3** |

The `services_global.go` file grows slightly (~3 lines net) due to the new `GetEntityRegistry()` function, but the actual duplication is eliminated. The net savings are in the service files themselves.

---

## 7. Test Updates

### 7.1 NoteService Tests

**Remove**: Per-entity mock repository definitions (`MockNoteEpicRepository`, `MockNoteFeatureRepository`, etc.)

**Add**: Mock `EntityRegistry` with mock `EntityRepository` entries.

**Pattern**:
```go
func TestNoteService_AddNote(t *testing.T) {
    // Create mock EntityRepository
    mockEntityRepo := &MockEntityRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
            return &models.Epic{ID: 42, Key: "E01", Title: "Test Epic"}, nil
        },
    }

    // Create registry with mock
    registry := services.NewEntityRegistry()
    registry.Register(models.EntityTypeEpic, mockEntityRepo)

    // Create mock note repo
    mockNoteRepo := &MockNoteEntityNoteRepository{
        CreateFunc: func(ctx context.Context, note *models.EntityNote) error {
            assert.Equal(t, int64(42), note.EntityID)
            return nil
        },
    }

    // Create service
    svc := services.NewNoteService(mockNoteRepo, registry)

    // Test
    note, err := svc.AddNote(ctx, models.EntityTypeEpic, "E01", "comment", "test content", "agent")
    assert.NoError(t, err)
    assert.Equal(t, int64(42), note.EntityID)
}
```

**Parameterize**: Test all 5 entity types using table-driven tests with a single mock registry containing all 5 types.

### 7.2 ContextService Tests

Same pattern as NoteService tests. Mock `EntityRepository` entries in registry provide `GetContextData` and `UpdateContextData` responses.

### 7.3 ResumeService Tests

Update constructor calls to include the registry parameter. For `GetBugResume` and `GetChangeResume` tests, replace mock `ResumeBugRepository`/`ResumeChangeCardRepository` with registry-based mocks.

For `GetEpicResume`, `GetFeatureResume`, `GetTaskResume` tests -- update constructor signature only (add registry param); test logic is unchanged since these methods still use typed repos.

### 7.4 Mock EntityRepository

A shared mock can be defined in a test helper or in `mocks_test.go`:

```go
type MockEntityRepository struct {
    GetByKeyFunc        func(ctx context.Context, key string) (models.Entity, error)
    GetByIDFunc         func(ctx context.Context, id int64) (models.Entity, error)
    UpdateStatusFunc    func(ctx context.Context, id int64, status string) error
    UpdateFunc          func(ctx context.Context, entity models.Entity) error
    GetContextDataFunc  func(ctx context.Context, id int64) (*string, error)
    UpdateContextDataFunc func(ctx context.Context, id int64, data *string) error
}

func (m *MockEntityRepository) GetByKey(ctx context.Context, key string) (models.Entity, error) {
    if m.GetByKeyFunc != nil {
        return m.GetByKeyFunc(ctx, key)
    }
    return nil, fmt.Errorf("GetByKey not implemented")
}
// ... (implement remaining methods)
```

---

## 8. Migration Strategy -- Implementation Order

### Task 1: NoteService Refactoring

**Files modified**: `note_service.go`, `note_service_test.go`
**Changes**:
1. Remove 5 per-entity repository interfaces
2. Change struct to use `*EntityRegistry` instead of 5 repo fields
3. Change constructor signature: `NewNoteService(noteRepo, registry)`
4. Add nil-check panic in constructor
5. Replace `resolveEntityID` switch with registry lookup
6. Replace `GetEntityDetails` switch with registry lookup
7. Remove `SetChangeCardRepo` and `SetBugRepo` setter methods
8. Update all tests

**Validation**: `make fmt && make lint && make test`

### Task 2: ContextService Refactoring

**Files modified**: `context_service.go`, `context_service_test.go`
**Changes**:
1. Remove 5 per-entity repository interfaces
2. Change struct to use `*EntityRegistry` instead of 5 repo fields
3. Change constructor signature: `NewContextService(registry)`
4. Add nil-check panic in constructor
5. Replace `getContextJSON` switch with registry lookup
6. Replace `setContextJSON` switch with registry lookup
7. Remove `SetBugRepo` and `SetChangeCardRepo` setter methods
8. Update all tests

**Validation**: `make fmt && make lint && make test`

### Task 3: ResumeService Refactoring

**Files modified**: `resume_service.go`, `resume_service_test.go`
**Changes**:
1. Remove `ResumeBugRepository` and `ResumeChangeCardRepository` interfaces
2. Add `registry *EntityRegistry` field to struct
3. Change constructor signature: add `registry` param
4. Add nil-check panic in constructor
5. Remove `SetBugRepo` and `SetChangeCardRepo` setter methods
6. Refactor `GetBugResume` to use registry + type assertion
7. Refactor `GetChangeResume` to use registry + type assertion
8. Update all tests

**Validation**: `make fmt && make lint && make test`

### Task 4: CLI Wiring (services_global.go)

**Files modified**: `services_global.go`
**Changes**:
1. Add `GetEntityRegistry()` function with `sync.Once`
2. Update `GetNoteService()` to use `NewNoteService(noteRepo, GetEntityRegistry())`
3. Update `GetContextService()` to use `NewContextService(GetEntityRegistry())`
4. Update `GetResumeService()` to pass `GetEntityRegistry()` and remove bug/change setter calls
5. Update `ResetServices()` to reset registry state
6. Add `models` import

**Validation**: `make fmt && make lint && make test`
**Additional**: Run end-to-end CLI tests for note, context, and resume operations across all 5 entity types.

### Task 5: Test Updates and Validation

**Files modified**: All test files for the 3 services
**Changes**:
1. Create shared `MockEntityRepository` if not already available
2. Update NoteService tests to use mock registry
3. Update ContextService tests to use mock registry
4. Update ResumeService tests with new constructor
5. Add parameterized tests across all 5 entity types for NoteService and ContextService
6. Remove old per-entity mock repository definitions

**Validation**: Full `make fmt && make lint && make test` pass. Verify test coverage does not decrease.

### Dependency Graph

```
Task 1 (NoteService) ──┐
Task 2 (ContextService) ├──> Task 4 (CLI Wiring) ──> Task 5 (Test Validation)
Task 3 (ResumeService) ─┘
```

Tasks 1-3 can proceed in parallel since they modify separate files. Task 4 depends on all three because it changes the constructor call sites. Task 5 is the final validation pass.

**Alternative**: Tasks 1-3 can be sequenced if a single developer works on them, committing after each service refactoring. This allows incremental validation. The CLI wiring (Task 4) must be done last because the constructor signatures must change in the service files before the wiring can be updated.

---

## 9. Summary Metrics

### Lines Removed vs Added

| File | Lines Before | Lines After (est.) | Net Change |
|------|-------------|-------------------|------------|
| `note_service.go` | 227 | ~120 | -107 |
| `context_service.go` | 308 | ~170 | -138 |
| `resume_service.go` | 403 | ~380 | -23 |
| `services_global.go` (affected functions) | 50 | 53 | +3 |
| **Total** | **988** | **~723** | **-265** |

### Items Eliminated

| Category | Count |
|----------|-------|
| Per-entity repository interfaces removed | 12 (5+5+2) |
| Setter methods removed | 7 (2+2+3) |
| Switch statements eliminated | 4 (2 in NoteService + 2 in ContextService) |
| Switch branches eliminated | ~20 |

### Zero-Change Guarantee

All existing CLI commands that call `GetNoteService()`, `GetContextService()`, or `GetResumeService()` require zero code changes. Only the service constructors and `services_global.go` wiring change. The service method signatures (`AddNote`, `GetContext`, `SetContextField`, `GetEpicResume`, etc.) are unchanged.

---

*Derived from F01 architecture (02-architecture.md), epic architecture-design.md, and actual codebase inspection at commit d33297e on branch e21-f01-entity-interface-foundation.*
