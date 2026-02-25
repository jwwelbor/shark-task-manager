# Architecture: E15-F10 CLI Command Refactoring and Enhancements

**Feature**: E15-F10-cli-command-refactoring-and-enhancements
**Date**: 2026-02-18
**Author**: Architect Agent

---

## 1. Overview

This document captures the current state of the CLI command layer, identifies remaining violations of the thin-wrapper pattern, and prescribes the concrete implementation work required for feature E15-F10.

The goal of E15 overall is to migrate from a fat-controller pattern (CLI commands calling repositories directly) to a clean three-layer architecture:

```
CLI Command (thin wrapper)
    → Service Layer (all business logic)
        → Repository Layer (pure data access)
            → SQLite/Turso Database
```

Feature E15-F10 is the final phase of this migration, addressing the remaining CLI commands that still violate the thin-wrapper pattern.

---

## 2. Current State Assessment

### 2.1 Service Layer — Complete

All three primary domain services are fully implemented and ready for CLI integration.

#### EpicService (`internal/services/epic_service.go` — 1069 lines)

Status: **Fully implemented**. No additional service methods required.

Key methods available:
- `GetEpic(ctx, key) (*Epic, error)`
- `ListEpics(ctx, EpicFilters) ([]*Epic, error)`
- `CreateEpic(ctx, CreateEpicInput) (*Epic, error)`
- `UpdateEpic(ctx, key, UpdateEpicInput) (*Epic, error)`
- `DeleteEpic(ctx, key, force) error`
- `TransitionStatus(ctx, key, toStatus, opts) (*Epic, error)`
- `GetNextStatus(ctx, key) (string, error)`
- `CalculateProgress(ctx, key) (weighted, completion float64, total int, error)`
- `GetProgress(ctx, key) (*EpicProgress, error)`
- `GetFeatureRollup(ctx, key) ([]*FeatureRollup, error)`
- `GetTaskStatusRollup(ctx, key) (*TaskStatusRollup, error)`
- `GetImpediments(ctx, key) ([]*Impediment, error)`
- `GetBlockedTasks(ctx, key) ([]*Task, error)`
- `GetRelatedDocuments(ctx, key) ([]*Document, error)`
- `GetHealth(ctx, key) (string, error)`
- `CompleteEpic(ctx, key, force) error`
- `GetFeatures(ctx, key) ([]*Feature, error)`
- `RecalculateStatus(ctx, key) error`
- `CascadeStatusToFeaturesAndTasks(ctx, key, status) error`
- `RenameKey(ctx, oldKey, newKey) error`
- `ResolveEpicPath(ctx, key) (string, error)`

#### FeatureService (`internal/services/feature_service.go` — 1123 lines)

Status: **Fully implemented**. No additional service methods required.

Key methods available:
- `GetFeature(ctx, key) (*Feature, error)`
- `GetFeatureByID(ctx, id) (*Feature, error)`
- `ListFeatures(ctx, FeatureFilters) ([]*Feature, error)`
- `ListFeaturesByEpicKey(ctx, epicKey) ([]*Feature, error)`
- `CreateFeature(ctx, CreateFeatureInput) (*Feature, error)`
- `UpdateFeature(ctx, key, UpdateFeatureInput) (*Feature, error)`
- `DeleteFeature(ctx, key, force) error`
- `TransitionStatus(ctx, key, toStatus, opts) (*Feature, error)`
- `GetNextStatus(ctx, key) (string, error)`
- `CompleteFeature(ctx, key, force) error`
- `GetProgress(ctx, key) (*FeatureProgress, error)`
- `RecalculateAndSetProgress(ctx, featureID) error`
- `RecalculateAndSetProgressByKey(ctx, key) error`
- `GetHealth(ctx, key) (string, error)`
- `GetWorkBreakdown(ctx, key) (*WorkSummary, error)`
- `GetActionItems(ctx, key) (map[string][]*Task, error)`
- `GetEnrichedTaskStatusBreakdown(ctx, key) (*EnrichedTaskStatusBreakdown, error)`
- `GetTaskStatusBreakdown(ctx, key) (*TaskStatusBreakdown, error)`
- `GetTaskCount(ctx, key) (int, error)`
- `UpdateFeatureKey(ctx, oldKey, newKey) error`
- `SetFeatureStatusOverride(ctx, key, status) error`
- `CascadeFeatureStatusToTasks(ctx, key, status) error`
- `ResolveFeaturePath(ctx, key) (string, error)`
- `UpdateFeatureFilePath(ctx, key, newPath) error`
- `ListTasksForFeature(ctx, key) ([]*Task, error)`
- `ListRelatedDocuments(ctx, key) ([]*Document, error)`

#### TaskService (`internal/services/task_service.go`)

Status: **Fully implemented**. Already used throughout task.go commands.

### 2.2 CLI Service Accessors — Complete

The service accessor layer is fully implemented across two files:

**`internal/cli/services_global.go`** — Contains:
- `GetNoteService()`
- `GetContextService()`
- `GetResumeService()`
- `GetTaskService()`
- `ResetServices()`

**`internal/cli/service_accessors.go`** — Contains:
- `GetEpicService()` — Fully wired with epicRepo, featureRepo, taskRepo, docRepo, noteRepo, workflowSvc
- `GetFeatureService()` — Fully wired with featureRepo, epicRepo, taskRepo, noteRepo, docRepo, workflowSvc
- `GetDisplayService()` — Wired with db and workflowSvc
- `epicNoteAdapter` — Adapts EntityNoteRepository to EpicNoteRepository interface
- `featureNoteAdapter` — Adapts EntityNoteRepository to FeatureNoteRepository interface

**Assessment**: The accessor layer is complete. T-E15-F10-004 (originally "Update CLI accessors") is already done; the task needs to be reconsidered.

### 2.3 CLI Command Files — Current Conformance Status

#### Already Thin Wrappers (Compliant)

| File | Lines | Notes |
|------|-------|-------|
| `epic.go` | 280 | All run* functions delegate to perform* helpers in epic_helpers.go; uses `cli.GetEpicService()` |
| `task.go` | 399 | Fully converted — all 13+ run* functions use `cli.GetTaskService()` |
| `feature.go` | 339 | Main run* functions use `cli.GetFeatureService()` |

#### Partially Compliant (Need Completion)

| File | Lines | Remaining Issues |
|------|-------|-----------------|
| `epic_helpers.go` | 1170 | Core operations use GetEpicService/GetFeatureService, but legacy helper functions like `getNextEpicKey` accept `*repository.EpicRepository` directly |
| `feature_helpers.go` | 1307 | Core operations use GetFeatureService, but some helper functions may accept concrete repository types |
| `helpers.go` | 766 | Shared helper utilities — requires review for remaining repo access |
| `task_helpers.go` | 283 | Newly created — likely thin wrappers, but unreviewed |

#### Fat Controllers (Fully Non-Compliant — Direct Repository Access)

| File | Lines | Anti-Pattern Description |
|------|-------|--------------------------|
| `idea.go` | 1002 | Entire file uses direct `repository.NewIdeaRepository(db)` pattern. No IdeaService exists. |
| `task_deps.go` | 780 | Dependency management commands (`task dep add`, `task dep remove`, `task dep list`) call `repository.NewTaskRepository(db)` directly with 3 handler functions |
| `related_docs.go` | 462 | Related document commands call `repository.NewDocumentRepository(db)` directly with 3 handler functions |
| `search.go` | 176 | Search command uses `repository.NewTaskRepository(db)` directly |
| `task_sessions.go` | 153 | Session commands use direct repository access |
| `task_unlink.go` | 178 | File unlink command uses `cli.GetDB()` and repository directly |
| `status.go` | 141 | Status dispatch command uses `cli.GetDB()` directly |

**Total non-compliant code**: ~2892 lines of fat-controller CLI code.

---

## 3. Architecture Decision: Scope of E15-F10

### 3.1 Revised Scope Based on Current State

Given that T-E15-F10-002 (EpicService) and T-E15-F10-003 (FeatureService) are **already complete**, and T-E15-F10-004 (service accessors) is **already complete**, the actual implementation work for E15-F10 is:

1. **T-E15-F10-001**: Refactor the seven fat-controller command files to use services
2. **T-E15-F10-002**: Reclassify as "Verify EpicService integration and add missing service tests" (service exists, integration verification needed)
3. **T-E15-F10-003**: Reclassify as "Verify FeatureService integration and add missing service tests" (service exists, integration verification needed)
4. **T-E15-F10-004**: Reclassify as "Create IdeaService for idea.go refactoring" (the one missing service)

### 3.2 The IdeaService Gap

`idea.go` is a 1002-line fat controller that manages idea capture with keys like `I-2026-01-01-01`. There is currently no `IdeaService` in `internal/services/`. This is the only domain entity without a service layer implementation.

**Decision**: Create `IdeaService` as part of T-E15-F10-004 (repurposed), which enables idea.go refactoring in T-E15-F10-001.

### 3.3 Secondary Domain Commands

The remaining fat controllers (`task_deps.go`, `related_docs.go`, `search.go`, `task_sessions.go`, `task_unlink.go`, `status.go`) implement operations that touch existing domain models (Task, Feature, Epic) but provide secondary functionality (dependency management, document linking, search, sessions, status dispatch).

**Decision**: These should be migrated to call existing services where possible, and new narrow service methods added where the service layer lacks required operations.

---

## 4. Migration Strategy per File

### 4.1 `idea.go` (1002 lines) — Requires New IdeaService

**Current pattern**:
```go
func runIdeaCreate(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    db, err := cli.GetDB(ctx)
    // ...
    repo := repository.NewIdeaRepository(db)
    idea, err := repo.Create(ctx, &models.Idea{...})
    // ...
}
```

**Target pattern**:
```go
func runIdeaCreate(cmd *cobra.Command, args []string) error {
    input := parseIdeaCreateInput(cmd, args)
    svc := cli.GetIdeaService()
    idea, err := svc.CreateIdea(cmd.Context(), input)
    if err != nil {
        return err
    }
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(idea)
    }
    cli.Success(fmt.Sprintf("Created idea %s", idea.Key))
    return nil
}
```

**Required new code**:
- `internal/services/idea_service.go` — IdeaService with CRUD and key generation
- `internal/services/idea_dto.go` — CreateIdeaInput, UpdateIdeaInput, IdeaFilters
- `GetIdeaService()` accessor in `internal/cli/service_accessors.go`

### 4.2 `task_deps.go` (780 lines) — Extend TaskService

**Operations**: `task dep add`, `task dep remove`, `task dep list`, `task dep validate`

**Current pattern**: Direct `repository.NewTaskRepository(db)` access.

**Target**: Add dependency management methods to TaskService:
- `TaskService.AddDependency(ctx, taskKey, depKey) error`
- `TaskService.RemoveDependency(ctx, taskKey, depKey) error`
- `TaskService.ListDependencies(ctx, taskKey) ([]*Task, error)`
- `TaskService.ValidateDependencies(ctx, taskKey) error`

Most of these operations can delegate directly to the TaskRepository, with the service providing business rule enforcement (e.g., circular dependency detection).

### 4.3 `related_docs.go` (462 lines) — Use Existing Service Methods

**Operations**: Link/unlink documents to epics, features, tasks.

**Assessment**: EpicService has `GetRelatedDocuments(ctx, key)`. FeatureService has `ListRelatedDocuments(ctx, key)`. These services could be extended with `LinkDocument` and `UnlinkDocument` methods.

**Target**: Add to EpicService and FeatureService:
- `LinkDocument(ctx, entityKey, docPath) error`
- `UnlinkDocument(ctx, entityKey, docPath) error`

For tasks, add similar methods to TaskService.

### 4.4 `search.go` (176 lines) — Add SearchService or Extend Services

**Operations**: Full-text search across tasks, features, epics.

**Decision**: Create a narrow `SearchService` or add search methods to existing services. Given the cross-entity nature of search, a dedicated service is appropriate.

### 4.5 `task_sessions.go` (153 lines) — Extend TaskService

**Operations**: Session tracking for tasks (start/end work sessions, time tracking).

**Target**: Add session methods to TaskService or create a dedicated SessionService.

### 4.6 `task_unlink.go` (178 lines) — Extend TaskService

**Operations**: Unlink a file from a task record.

**Target**: Add `UnlinkFile(ctx, taskKey) error` to TaskService.

### 4.7 `status.go` (141 lines) — Use Existing Services

**Operations**: Status dispatch/smart status command.

**Target**: Refactor to use `cli.GetTaskService()`, `cli.GetFeatureService()`, or `cli.GetEpicService()` based on key format detection.

---

## 5. Service Architecture — New Additions

### 5.1 IdeaService (New)

```
IdeaService
├── IdeaRepository (interface)
│   └── *repository.IdeaRepository (implementation)
│       └── *repository.DB
└── *workflow.Service (optional — if ideas have workflow)
```

**Constructor**:
```go
func NewIdeaService(repo IdeaRepository) *IdeaService
```

**Interface definition** (`internal/services/idea_service.go`):
```go
type IdeaRepository interface {
    Create(ctx context.Context, idea *models.Idea) error
    GetByID(ctx context.Context, id int64) (*models.Idea, error)
    GetByKey(ctx context.Context, key string) (*models.Idea, error)
    List(ctx context.Context, filter *IdeaFilter) ([]*models.Idea, error)
    Update(ctx context.Context, idea *models.Idea) error
    Delete(ctx context.Context, id int64) error
    MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error
    GetNextSequenceForDate(ctx context.Context, dateStr string) (int, error)
}
```

**Note**: The `IdeaRepository` interface already exists in `idea.go` — it should be moved to the service layer.

**Key generation**: The idea key format (`I-YYYY-MM-DD-xx`) includes business logic (date derivation, sequence generation). This logic must move from `idea.go` into `IdeaService`.

### 5.2 TaskService Extensions

New methods to add:
```go
func (s *TaskService) AddDependency(ctx context.Context, taskKey, depKey string) error
func (s *TaskService) RemoveDependency(ctx context.Context, taskKey, depKey string) error
func (s *TaskService) ListDependencies(ctx context.Context, taskKey string) ([]*models.Task, error)
func (s *TaskService) UnlinkFile(ctx context.Context, taskKey string) error
func (s *TaskService) StartSession(ctx context.Context, taskKey string) (*models.Session, error)
func (s *TaskService) EndSession(ctx context.Context, taskKey string, notes string) error
```

### 5.3 EpicService and FeatureService Extensions

New methods to add:
```go
// EpicService
func (s *EpicService) LinkDocument(ctx context.Context, epicKey, docPath string) error
func (s *EpicService) UnlinkDocument(ctx context.Context, epicKey, docPath string) error

// FeatureService
func (s *FeatureService) LinkDocument(ctx context.Context, featureKey, docPath string) error
func (s *FeatureService) UnlinkDocument(ctx context.Context, featureKey, docPath string) error
```

---

## 6. Accessor Layer Updates

`internal/cli/service_accessors.go` needs one addition:

```go
// GetIdeaService returns an IdeaService instance.
func GetIdeaService() *services.IdeaService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    ideaRepo := repository.NewIdeaRepository(db)
    return services.NewIdeaService(ideaRepo)
}
```

---

## 7. File Line Count Targets

After refactoring, each CLI command file should meet the thin-wrapper standard:

| File | Current Lines | Target Lines | Reduction |
|------|--------------|-------------|-----------|
| `idea.go` | 1002 | ~150 | ~85% |
| `task_deps.go` | 780 | ~100 | ~87% |
| `related_docs.go` | 462 | ~80 | ~83% |
| `search.go` | 176 | ~50 | ~72% |
| `task_sessions.go` | 153 | ~50 | ~67% |
| `task_unlink.go` | 178 | ~40 | ~78% |
| `status.go` | 141 | ~60 | ~57% |

The extracted business logic will move into:
- `internal/services/idea_service.go` (~300 lines)
- `internal/services/idea_dto.go` (~60 lines)
- Extensions to `task_service.go` (~150 lines)
- Extensions to `epic_service.go` (~80 lines)
- Extensions to `feature_service.go` (~80 lines)

---

## 8. Testing Architecture

### 8.1 Service Tests (New/Extended)

All new service code requires unit tests with mocked repositories:

- `internal/services/idea_service_test.go` — Full coverage of IdeaService
- Extended tests in `internal/services/task_service_test.go` — Dependency and session methods
- Extended tests in `internal/services/epic_service_test.go` — Document linking
- Extended tests in `internal/services/feature_service_test.go` — Document linking

### 8.2 CLI Command Tests

Refactored CLI commands require tests with mocked services:

- `internal/cli/commands/idea_test.go` — Tests for each idea command
- `internal/cli/commands/task_deps_test.go` — Tests for dependency commands
- `internal/cli/commands/related_docs_test.go` — Tests for document linking commands
- `internal/cli/commands/search_test.go` — Tests for search command

### 8.3 Test Infrastructure

The mock infrastructure for services already exists in the test files. The pattern to follow is established in `task_service_test.go` and related files.

---

## 9. Implementation Sequence

The recommended implementation order respects dependencies:

1. **T-E15-F10-004** (Create IdeaService): Must happen first as `idea.go` refactoring depends on it
2. **T-E15-F10-001** (Refactor fat controllers): After IdeaService exists, refactor all seven files
3. **T-E15-F10-002** (Verify EpicService integration): Validate epic_helpers.go is fully clean
4. **T-E15-F10-003** (Verify FeatureService integration): Validate feature_helpers.go is fully clean

---

## 10. Quality Gates

Before any task can be marked complete:

1. `make fmt && make lint && make test` must pass with zero failures
2. The refactored command file must contain no `repository.New*` or `cli.GetDB()` calls
3. All business logic extracted to service layer must have unit tests with mocked repositories
4. Command file line counts must be within target ranges (see Section 7)
5. No new direct repository imports in CLI command packages

---

## 11. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Idea key generation logic is complex | Medium | Medium | Move key generation logic entirely into IdeaService; test thoroughly |
| task_deps.go has circular dependency detection logic | Medium | High | Implement in TaskService with full test coverage |
| helpers.go may have hidden direct repo access | Low | Medium | Full grep scan of helpers.go before declaring F10 complete |
| Existing integration tests depend on fat-controller behavior | Medium | Medium | Run full test suite after each file refactoring |
| idea.go IdeaRepository interface conflicts with service layer | Low | Low | Move interface definition to service layer, remove from idea.go |

---

## 12. Out of Scope

The following are explicitly out of scope for E15-F10:

- Refactoring `epic_helpers.go` and `feature_helpers.go` further beyond current state (covered by F07)
- Adding new CLI commands or features
- HTTP API handler refactoring (separate concern)
- Changing the observable CLI behavior or output format
- Performance optimization of service methods
