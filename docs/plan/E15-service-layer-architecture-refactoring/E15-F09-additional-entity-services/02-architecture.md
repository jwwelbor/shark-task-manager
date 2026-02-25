# E15-F09 Architecture: Additional Entity Services

**Feature:** E15-F09 - Additional Entity Services
**Date:** 2026-02-18
**Complexity Tier:** STANDARD
**Status:** Implementation already complete - this document records architectural decisions

---

## Executive Summary

All three tasks in E15-F09 are already implemented. EpicService and FeatureService are fully built services with complete CRUD, progress tracking, workflow transitions, and business logic. The CLI accessor functions (`GetEpicService`, `GetFeatureService`) are wired in `internal/cli/service_accessors.go` and already in active use by CLI commands.

This document records the architectural decisions made during implementation and serves as the authoritative reference for the E15-F09 service layer design.

---

## Current State Assessment

### Task Status

| Task | Title | Status |
|------|-------|--------|
| T-E15-F09-001 | Implement EpicService with CRUD and rollup operations | Done - `internal/services/epic_service.go` (~1069 lines) |
| T-E15-F09-002 | Implement FeatureService with CRUD and progress tracking | Done - `internal/services/feature_service.go` (~1123 lines) |
| T-E15-F09-003 | Update CLI accessors for new services | Done - `internal/cli/service_accessors.go` |

### Research Finding

The E15-EPIC-RESEARCH-REPORT.md (written at epic start, 2026-02-16) documented EpicService at 232 lines and FeatureService at 239 lines, each with only status transitions. Subsequent features in E15 (E15-F05 and related) expanded both services to their full implementations before E15-F09 formally reached the development phase.

---

## Architectural Decisions

### Decision 1: Two Constructor Variants per Service

Both EpicService and FeatureService use two constructors following a consistent pattern.

**Standard constructor** - for operations not requiring document/relationship lookup:

```go
// epic_service.go
func NewEpicService(
    repo EpicRepository,
    workflowSvc *workflow.Service,
    noteRepo EpicNoteRepository,
    featureRepo EpicFeatureCounter,
    taskRepo EpicTaskLister,
) *EpicService

// feature_service.go
func NewFeatureService(
    repo FeatureRepository,
    workflowSvc *workflow.Service,
    noteRepo FeatureNoteRepository,
    taskRepo FeatureTaskCounter,
    epicLookupRepo FeatureEpicLookup,
) *FeatureService
```

**With-relationships constructor** - for orchestrator action population (PopulatedAction) which requires linked documents and related entities:

```go
// epic_service.go
func NewEpicServiceWithRelationships(
    repo EpicRepository,
    workflowSvc *workflow.Service,
    noteRepo EpicNoteRepository,
    featureRepo EpicFeatureCounter,
    docRepo config.DocumentRepository,
    relRepo config.EpicRelationshipRepository,
) *EpicService

// feature_service.go
func NewFeatureServiceWithRelationships(
    repo FeatureRepository,
    workflowSvc *workflow.Service,
    noteRepo FeatureNoteRepository,
    taskRepo FeatureTaskCounter,
    docRepo DocumentRepository,
    relRepo FeatureRelationshipRepository,
    epicLookupRepo FeatureEpicLookup,
) *FeatureService
```

**Rationale:** Most CLI operations do not need relationship data. Keeping it optional prevents unnecessary coupling and avoids loading documents that are not needed. The `SetDocRepo` escape hatch on EpicService allows post-construction injection for edge cases.

### Decision 2: Workflow Service Scoped to Entity Level

Both services call `workflowSvc.ForLevel(workflow.LevelEpic)` and `workflowSvc.ForLevel(workflow.LevelFeature)` in their constructors. This scopes workflow validation to the correct entity-level status list and transitions.

```go
// In NewEpicService constructor
workflowSvc: workflowSvc.ForLevel(workflow.LevelEpic),

// In NewFeatureService constructor
workflowSvc: workflowSvc.ForLevel(workflow.LevelFeature),
```

This matches the TaskService pattern (`workflow.LevelTask`) and ensures that epic/feature-specific workflow config is applied rather than the global config.

### Decision 3: Note Repository Adapter Pattern

The concrete `*repository.EntityNoteRepository` uses `models.EntityType` and `*string` for entity type and document path respectively. The `EpicNoteRepository` and `FeatureNoteRepository` service interfaces use plain `string` for both fields, which is simpler for service-layer consumers.

An adapter struct bridges this impedance mismatch in `internal/cli/service_accessors.go`:

```go
// epicNoteAdapter adapts *repository.EntityNoteRepository to services.EpicNoteRepository
type epicNoteAdapter struct {
    repo *repository.EntityNoteRepository
}

func (a *epicNoteAdapter) CreateRejectionNote(
    ctx context.Context,
    entityType string,
    entityID int64,
    historyID int64,
    fromStatus, toStatus, reason, rejectedBy, documentPath string,
) error {
    var dp *string
    if documentPath != "" {
        dp = &documentPath
    }
    _, err := a.repo.CreateRejectionNote(ctx, models.EntityType(entityType), entityID, historyID,
        fromStatus, toStatus, reason, rejectedBy, dp)
    return err
}
```

A `featureNoteAdapter` with identical logic exists for the feature note interface.

**Rationale:** This keeps the service interfaces clean and string-based (avoiding `models.EntityType` imports in service layer), while the concrete repository continues using its own types. The adapter lives in the CLI wiring layer, which is the correct location for such bridging.

### Decision 4: CLI Accessor Design - Panic-on-Failure Pattern

`GetEpicService()` and `GetFeatureService()` follow the same fail-fast pattern as `GetTaskService()`:

```go
func GetEpicService() *services.EpicService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)
    docRepo := repository.NewDocumentRepository(db)
    noteRepo := &epicNoteAdapter{repo: repository.NewEntityNoteRepository(db)}
    workflowSvc := GetWorkflowService()
    svc := services.NewEpicService(epicRepo, workflowSvc, noteRepo, featureRepo, taskRepo)
    svc.SetDocRepo(docRepo)
    return svc
}

func GetFeatureService() *services.FeatureService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)
    noteRepo := &featureNoteAdapter{repo: repository.NewEntityNoteRepository(db)}
    docRepo := repository.NewDocumentRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewFeatureServiceWithRelationships(
        featureRepo, workflowSvc, noteRepo, taskRepo, docRepo, nil, epicRepo,
    )
}
```

Key design choices:
- **Panic on DB failure**: Consistent with `GetTaskService()`. A CLI command that can't reach its database should not silently degrade.
- **New instance per call**: Services are lightweight; the expensive resources (DB connection, workflow service) are global singletons reused across calls.
- **All optional deps wired**: Unlike `GetTaskService()` which passes `nil` for `creatorSvc`, the epic and feature accessors wire all known optional dependencies (noteRepo, featureRepo, taskRepo, docRepo) to provide full functionality.
- **`GetFeatureService` uses `WithRelationships`**: Feature commands regularly need document access for context display; using the relationships constructor avoids a later `SetDocRepo` call.

### Decision 5: EpicService Uses `NewEpicService` + `SetDocRepo`

`GetEpicService()` uses `NewEpicService` (not `NewEpicServiceWithRelationships`) and then calls `svc.SetDocRepo(docRepo)`. This is because `NewEpicService` accepts `EpicTaskLister` (for blocked task queries) as a fifth parameter, but `NewEpicServiceWithRelationships` replaces `taskRepo` with `docRepo/relRepo`.

The `SetDocRepo` escape hatch allows callers to add document support after construction without requiring `EpicRelationshipRepository` (which is nil in CLI context since epic relationships are not fully implemented).

---

## Dependency Graph

### EpicService Dependencies (CLI wiring)

```
GetEpicService()
├── *repository.EpicRepository      (satisfies EpicRepository interface)
├── *repository.FeatureRepository   (satisfies EpicFeatureCounter interface)
├── *repository.TaskRepository      (satisfies EpicTaskLister interface)
├── *repository.DocumentRepository  (via SetDocRepo, satisfies config.DocumentRepository)
├── *epicNoteAdapter                (satisfies EpicNoteRepository interface)
│   └── *repository.EntityNoteRepository
└── *workflow.Service               (scoped to workflow.LevelEpic)
```

### FeatureService Dependencies (CLI wiring)

```
GetFeatureService()
├── *repository.FeatureRepository   (satisfies FeatureRepository interface)
├── *repository.EpicRepository      (satisfies FeatureEpicLookup interface)
├── *repository.TaskRepository      (satisfies FeatureTaskCounter interface)
├── *repository.DocumentRepository  (satisfies DocumentRepository interface)
├── nil                             (FeatureRelationshipRepository - not used in CLI)
├── *featureNoteAdapter             (satisfies FeatureNoteRepository interface)
│   └── *repository.EntityNoteRepository
└── *workflow.Service               (scoped to workflow.LevelFeature)
```

---

## Service Capabilities Reference

### EpicService (`internal/services/epic_service.go`)

**CRUD:**
- `CreateEpic(ctx, input CreateEpicInput) (*models.Epic, error)`
- `GetEpic(ctx, key string) (*models.Epic, error)`
- `UpdateEpic(ctx, key string, updates EpicUpdates) (*models.Epic, error)`
- `DeleteEpic(ctx, key string) error`
- `ListEpics(ctx, filters EpicFilters) ([]*models.Epic, error)`

**Rollup and Progress:**
- `GetFeatureRollup(ctx, epicKey string) (map[models.FeatureStatus]int, error)`
- `GetTaskStatusRollup(ctx, epicKey string) (map[string]int, error)`
- `GetProgress(ctx, epicKey string) (float64, error)`
- `CalculateProgress(ctx, epicID int64) (float64, error)`

**Health and Impediments:**
- `GetHealth(ctx, epicKey string) (string, error)`
- `GetImpediments(ctx, epicKey string) ([]*models.Task, error)`
- `GetBlockedTasks(ctx, epicKey string) ([]*models.Task, error)`

**Status Lifecycle:**
- `TransitionStatus(ctx, epicKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error)`
- `GetNextStatus(ctx, epicKey string) (string, error)`
- `CompleteEpic(ctx, epicKey string) error`
- `CascadeStatusToFeaturesAndTasks(ctx, epicID int64, featureStatus models.FeatureStatus, taskStatus models.TaskStatus) error`

**File and Key Management:**
- `ResolveEpicPath(ctx, epicKey string, filePath string) (string, error)`
- `RenameKey(ctx, oldKey string, newKey string) error`

### FeatureService (`internal/services/feature_service.go`)

**CRUD:**
- `CreateFeature(ctx, input CreateFeatureInput) (*models.Feature, error)`
- `GetFeature(ctx, key string) (*models.Feature, error)`
- `GetFeatureByID(ctx, id int64) (*models.Feature, error)`
- `UpdateFeature(ctx, key string, updates FeatureUpdates) (*models.Feature, error)`
- `DeleteFeature(ctx, key string) error`
- `ListFeatures(ctx) ([]*models.Feature, error)`
- `ListFeaturesByEpicKey(ctx, epicKey string) ([]*models.Feature, error)`

**Progress and Health:**
- `GetProgress(ctx, featureKey string) (float64, float64, int, error)`
- `RecalculateAndSetProgress(ctx, featureID int64) error`
- `RecalculateAndSetProgressByKey(ctx, featureKey string) error`
- `GetHealth(ctx, featureKey string) (string, error)`
- `GetWorkBreakdown(ctx, featureKey string) (*WorkBreakdown, error)`

**Status Breakdown and Action Items:**
- `GetTaskStatusBreakdown(ctx, featureKey string) (map[models.TaskStatus]int, error)`
- `GetEnrichedTaskStatusBreakdown(ctx, featureKey string) (map[string]interface{}, error)`
- `GetActionItems(ctx, featureKey string) (map[string][]*models.Task, error)`
- `ListTasksForFeature(ctx, featureKey string) ([]*models.Task, error)`

**Status Lifecycle:**
- `TransitionStatus(ctx, featureKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error)`
- `GetNextStatus(ctx, featureKey string) (string, error)`
- `CompleteFeature(ctx, featureKey string) error`

**File and Key Management:**
- `ResolveFeaturePath(ctx, featureKey string, filePath string) (string, error)`
- `UpdateFeatureFilePath(ctx, featureKey string, newPath *string) error`
- `UpdateFeatureKey(ctx, oldKey string, newKey string) error`

---

## Interface Definitions

### EpicNoteRepository Interface

```go
type EpicNoteRepository interface {
    CreateRejectionNote(ctx context.Context, entityType string, entityID int64,
        historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error
}
```

Satisfied by `*epicNoteAdapter` (not directly by `*repository.EntityNoteRepository`).

### FeatureNoteRepository Interface

```go
type FeatureNoteRepository interface {
    CreateRejectionNote(ctx context.Context, entityType string, entityID int64,
        historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error
}
```

Satisfied by `*featureNoteAdapter` (not directly by `*repository.EntityNoteRepository`).

---

## Integration Points

### CLI Commands Using EpicService

All epic CLI commands now call `cli.GetEpicService()` rather than creating repositories directly:

- `internal/cli/commands/epic.go` - List and Get operations
- `internal/cli/commands/epic_helpers.go` - Rollup, progress, transition display
- `internal/cli/commands/epic_next_status.go` - Next status transitions
- `internal/cli/commands/epic_set_status.go` - Forced status setting

### CLI Commands Using FeatureService

All feature CLI commands now call `cli.GetFeatureService()` rather than creating repositories directly:

- `internal/cli/commands/feature.go` - Create and Get operations
- `internal/cli/commands/feature_helpers.go` - Progress, health, action items display
- `internal/cli/commands/feature_next_status.go` - Next status transitions
- `internal/cli/commands/feature_set_status.go` - Forced status setting

### HTTP Server Wiring

The HTTP server (`cmd/server/services.go`) wires services explicitly at startup rather than using the global accessor pattern. It constructs `EpicService` and `FeatureService` in the same way as the CLI accessors but uses long-lived instances shared across all request handlers.

---

## Security Considerations

No new security surface introduced. This feature is a pure internal refactoring:
- No new API endpoints
- No new database tables or columns
- No new external integrations
- No change to authentication or authorization behavior
- CLI panic-on-failure is appropriate for single-user CLI context (not HTTP handlers)

---

## Testing Approach

Services are tested with mocked repositories following the established pattern:

**Service unit tests** (`internal/services/epic_service_test.go`, `feature_service_test.go`):
- Mock `EpicRepository`, `EpicNoteRepository`, `EpicFeatureCounter`, `EpicTaskLister`
- Mock `FeatureRepository`, `FeatureNoteRepository`, `FeatureTaskCounter`, `FeatureEpicLookup`
- Verify workflow validation is called with correct transition parameters
- Verify business rules (progress calculation, health thresholds, cascade logic)

**CLI command integration tests** use the service via `cli.GetEpicService()` / `cli.GetFeatureService()` against a test database. These are not unit tests of the service; they verify end-to-end command wiring.

---

## Decisions Not Made (Out of Scope)

- **EpicRelationshipRepository in CLI**: `GetFeatureService` passes `nil` for `relRepo` because epic-feature relationship queries are not yet surfaced in the CLI. This can be wired when needed without changing the service.
- **Service-level caching**: No caching layer. Each call to `GetEpicService()` creates a new instance; repository queries hit the database each time. SQLite is fast enough for CLI use.
- **Singleton pattern for epic/feature services**: Unlike `GetNoteService`, `GetContextService`, and `GetResumeService` (which use `sync.Once`), `GetEpicService` and `GetFeatureService` create a new instance per call (same as `GetTaskService`). This is intentional: consistency with `GetTaskService` and simplicity over micro-optimization.
