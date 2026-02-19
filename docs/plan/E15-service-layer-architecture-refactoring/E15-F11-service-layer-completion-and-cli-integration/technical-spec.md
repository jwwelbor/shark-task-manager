# Technical Specification: E15-F11 Service Layer Completion and CLI Integration

**Feature Key**: E15-F11-service-layer-completion-and-cli-integration
**Date**: 2026-02-19
**Status**: Ready for Development

---

## 1. Overview

This specification defines the technical approach to eliminate the remaining 13 fat-controller CLI command files by migrating their business logic into the service layer. The goal is zero `repository.New*` or `cli.GetDB` calls in command handler functions (excluding infrastructure commands).

### Architecture Reminder

```
CLI Command (thin wrapper)
    → parse args/flags
    → call service method
    → format output
    ↓
Service Layer (all business logic)
    → orchestration, validation, transactions
    ↓
Repository Layer (pure data access)
    → CRUD queries only
```

---

## 2. Scope Summary

### Files to Migrate (13 total)

| File | Lines | Complexity | New Service Needed | Service to Call |
|------|-------|------------|-------------------|-----------------|
| `task_criteria.go` | 460 | High | `CriteriaService` (new) | `GetCriteriaService()` |
| `task_note.go` | 442 | High | None (extend `NoteService`) | `GetNoteService()` |
| `task_resume.go` | 401 | Medium | None (extend `ResumeService`) | `GetResumeService()` |
| `task_context.go` | 397 | Low | None (`ContextService` exists) | `GetContextService()` |
| `task_next_status.go` | 361 | High | None (`TaskService` methods exist) | `GetTaskService()` |
| `task_history.go` | 247 | Medium | None (new method on `TaskService`) | `GetTaskService()` |
| `history.go` | 277 | Medium | None (new method on `TaskService`) | `GetTaskService()` |
| `analytics.go` | 238 | Medium | None (new method on `TaskService`) | `GetTaskServiceWithDeps()` |
| `notes_search.go` | 230 | Low | None (`NoteService` exists) | `GetNoteService()` |
| `task_link.go` | 198 | Low | None (`TaskService` methods exist) | `GetTaskServiceWithDeps()` |
| `feature_criteria.go` | 195 | Medium | `CriteriaService` (new, shared) | `GetCriteriaService()` |
| `view.go` | 104 | Low | None (entity services have file paths) | Existing services |
| `validate.go` | 98 | Low | Accepted exception (see §8) | N/A |

### Infrastructure Exceptions (no change needed)

- `cloud.go` - cloud connectivity, not business logic
- `init.go` - project initialization, not business logic
- `migrate_backfill_slugs.go` - one-time migration utility

---

## 3. New Service: CriteriaService

The most significant new service required. Used by both `task_criteria.go` and `feature_criteria.go`.

### 3.1 File Location

`internal/services/criteria_service.go`

### 3.2 Repository Interface

```go
// CriteriaRepository defines the data access interface for task acceptance criteria.
// Defined in criteria_service.go (consumer side).
type CriteriaRepository interface {
    Create(ctx context.Context, criteria *models.TaskCriteria) error
    GetByID(ctx context.Context, id int64) (*models.TaskCriteria, error)
    GetByTaskID(ctx context.Context, taskID int64) ([]*models.TaskCriteria, error)
    Update(ctx context.Context, criteria *models.TaskCriteria) error
    UpdateStatus(ctx context.Context, id int64, status models.CriteriaStatus, notes *string) error
    Delete(ctx context.Context, id int64) error
    DeleteByTaskID(ctx context.Context, taskID int64) error
    GetSummaryByTaskID(ctx context.Context, taskID int64) (*repository.CriteriaSummary, error)
}

// CriteriaTaskRepository defines the task repository interface needed by CriteriaService.
type CriteriaTaskRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Task, error)
}

// CriteriaFeatureRepository defines the feature repository interface needed by CriteriaService.
type CriteriaFeatureRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Feature, error)
    ListTasks(ctx context.Context, featureID int64) ([]*models.Task, error)
}
```

Note: `ListTasks` on `FeatureRepository` should be added if not present; fall back to `TaskRepository.ListByFeature` if needed (see §3.3 alternative).

### 3.3 Service Struct and Constructor

```go
// CriteriaService provides business logic for task acceptance criteria.
type CriteriaService struct {
    criteriaRepo CriteriaRepository
    taskRepo     CriteriaTaskRepository
    featureRepo  CriteriaFeatureRepository
    tasklistRepo CriteriaTaskListRepository // for feature-level aggregation
}

// NewCriteriaService creates a CriteriaService with injected dependencies.
func NewCriteriaService(
    criteriaRepo CriteriaRepository,
    taskRepo CriteriaTaskRepository,
    featureRepo CriteriaFeatureRepository,
    tasklistRepo CriteriaTaskListRepository,
) *CriteriaService
```

The concrete `*repository.TaskCriteriaRepository` implements `CriteriaRepository`.
The concrete `*repository.TaskRepository` implements `CriteriaTaskRepository`.
The concrete `*repository.FeatureRepository` implements `CriteriaFeatureRepository`.

### 3.4 DTOs

```go
// CriteriaImportInput contains parameters for importing criteria from a file.
type CriteriaImportInput struct {
    TaskKey string
}

// CriteriaImportResult is returned after an import operation.
type CriteriaImportResult struct {
    TaskKey  string `json:"task_key"`
    Imported int    `json:"imported"`
    Message  string `json:"message"`
}

// CriteriaListResult contains criteria and summary for a task.
type CriteriaListResult struct {
    TaskKey  string                        `json:"task_key"`
    Criteria []*models.TaskCriteria        `json:"criteria"`
    Summary  *repository.CriteriaSummary   `json:"summary"`
}

// FeatureCriteriaResult contains aggregated criteria across all tasks in a feature.
type FeatureCriteriaResult struct {
    FeatureKey      string                `json:"feature_key"`
    FeatureTitle    string                `json:"feature_title"`
    TaskCount       int                   `json:"task_count"`
    TotalCount      int                   `json:"total_count"`
    PendingCount    int                   `json:"pending_count"`
    InProgressCount int                   `json:"in_progress_count"`
    CompleteCount   int                   `json:"complete_count"`
    FailedCount     int                   `json:"failed_count"`
    NACount         int                   `json:"na_count"`
    CompletionPct   float64               `json:"completion_pct"`
    TaskSummaries   []TaskCriteriaSummary `json:"task_summaries,omitempty"`
}

// TaskCriteriaSummary is per-task criteria aggregation within a feature result.
type TaskCriteriaSummary struct {
    TaskKey         string  `json:"task_key"`
    TaskTitle       string  `json:"task_title"`
    TotalCount      int     `json:"total_count"`
    PendingCount    int     `json:"pending_count"`
    InProgressCount int     `json:"in_progress_count"`
    CompleteCount   int     `json:"complete_count"`
    FailedCount     int     `json:"failed_count"`
    NACount         int     `json:"na_count"`
    CompletionPct   float64 `json:"completion_pct"`
}
```

### 3.5 Service Methods

```go
// ImportCriteria imports acceptance criteria from the task's markdown file into the database.
// Parses checklist format: "- [ ] pending" and "- [x] complete".
// Returns the count of criteria imported.
func (s *CriteriaService) ImportCriteria(
    ctx context.Context,
    taskKey string,
) (*CriteriaImportResult, error)

// ListCriteria retrieves all acceptance criteria for a task with summary counts.
func (s *CriteriaService) ListCriteria(
    ctx context.Context,
    taskKey string,
) (*CriteriaListResult, error)

// CheckCriterion marks a criterion as complete (status="complete", sets verified_at).
// The note parameter adds optional verification notes.
func (s *CriteriaService) CheckCriterion(
    ctx context.Context,
    taskKey string,
    criterionID int64,
    note string,
) (*models.TaskCriteria, error)

// FailCriterion marks a criterion as failed (status="failed").
// The note parameter is required to explain the failure reason.
func (s *CriteriaService) FailCriterion(
    ctx context.Context,
    taskKey string,
    criterionID int64,
    note string,
) (*models.TaskCriteria, error)

// GetFeatureCriteria aggregates acceptance criteria across all tasks in a feature.
// When byTask is true, includes per-task breakdowns in the result.
func (s *CriteriaService) GetFeatureCriteria(
    ctx context.Context,
    featureKey string,
    byTask bool,
) (*FeatureCriteriaResult, error)
```

**Internal business logic in service:**
- `ImportCriteria`: Validates task exists, checks task has a file path, calls `taskfile.ParseCriteriaFromFile`, loops creating criteria records. No loops allowed in CLI command.
- `CheckCriterion` / `FailCriterion`: Validates task exists, validates criterion belongs to task (via GetByID, verify TaskID matches), calls `UpdateStatus` on repo.
- `GetFeatureCriteria`: Validates feature exists, loads all tasks via `tasklistRepo.ListByFeature`, loads criteria for each task, aggregates counts.

---

## 4. New Methods on Existing Services

### 4.1 TaskService: GetTaskHistory

Used by both `task_history.go` (per-task view) and `history.go` (project-wide view).

#### 4.1.1 New Repository Interface Extension

Add to `TaskHistoryRepository` interface in `task_service.go`:

```go
// TaskHistoryRepository defines the interface for task history data access.
// Add this interface to task_service.go (consumer side).
type TaskHistoryRepository interface {
    GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error)
    ListWithFilters(ctx context.Context, filters repository.HistoryFilters) ([]*models.TaskHistory, error)
}
```

The concrete `*repository.TaskHistoryRepository` implements this interface.

#### 4.1.2 TaskService Field and Constructor Update

Add optional `historyRepo` field to `TaskService`. Wire via `SetHistoryRepo()` setter (matching existing `SetDepRepo`, `SetRelQueryRepo` patterns):

```go
// SetHistoryRepo sets the history repository for history retrieval methods.
// Optional dependency - history methods return empty results if not set.
func (s *TaskService) SetHistoryRepo(historyRepo TaskHistoryRepository)
```

#### 4.1.3 New TaskService Methods

```go
// GetTaskHistory retrieves the complete status change history for a task.
// Returns history records in chronological order (oldest first).
func (s *TaskService) GetTaskHistory(
    ctx context.Context,
    taskKey string,
) ([]*models.TaskHistory, error)

// HistoryFilters defines parameters for querying project-wide history.
type HistoryFilters struct {
    AgentID    string
    Since      string // ISO 8601 timestamp string
    EpicKey    string
    FeatureKey string
    OldStatus  string
    NewStatus  string
    Limit      int
    Offset     int
    Format     string // "json", "csv", "table"
}

// ListHistory retrieves task history entries with optional filters for project-wide history.
// Handles epic/feature key resolution (converting string keys to IDs for query).
func (s *TaskService) ListHistory(
    ctx context.Context,
    filters HistoryFilters,
) ([]*models.TaskHistory, error)
```

Note: `HistoryFilters` maps to `repository.HistoryFilters` after key resolution. The service is responsible for resolving string epic/feature keys to IDs.

### 4.2 TaskService: GetAnalytics

Used by `analytics.go`.

#### 4.2.1 DTOs

```go
// AnalyticsInput defines the parameters for analytics queries.
type AnalyticsInput struct {
    EpicKey         string
    FeatureKey      string
    AgentType       string
    SessionDuration bool
    PauseFrequency  bool
}

// AnalyticsResult wraps session analytics results.
// Wraps repository.SessionAnalytics to decouple from repository type.
type AnalyticsResult struct {
    Scope           string               `json:"scope"` // "epic" or "feature"
    ScopeKey        string               `json:"scope_key"`
    AgentType       string               `json:"agent_type,omitempty"`
    SessionAnalysis *SessionDurationData `json:"session_duration,omitempty"`
    PauseAnalysis   *PauseFrequencyData  `json:"pause_frequency,omitempty"`
}

// SessionDurationData contains session duration metrics.
type SessionDurationData struct {
    TotalSessions   int     `json:"total_sessions"`
    AverageDuration float64 `json:"average_duration_minutes"`
    MedianDuration  float64 `json:"median_duration_minutes"`
    TotalTime       float64 `json:"total_time_minutes"`
}

// PauseFrequencyData contains pause frequency metrics.
type PauseFrequencyData struct {
    TotalSessions    int     `json:"total_sessions"`
    SessionsWithPause int   `json:"sessions_with_pause"`
    AveragePauses    float64 `json:"average_pauses_per_session"`
    PauseFrequencyPct float64 `json:"pause_frequency_pct"`
}
```

#### 4.2.2 New TaskService Method

```go
// GetAnalytics retrieves work session analytics for a given scope (epic or feature).
// Handles epic/feature key resolution and returns structured analytics data.
// Requires the work session repository to be wired (available in GetTaskServiceWithDeps).
func (s *TaskService) GetAnalytics(
    ctx context.Context,
    input AnalyticsInput,
) (*AnalyticsResult, error)
```

**Internal logic:**
- Validate that at least one of `EpicKey` or `FeatureKey` is provided.
- Validate that at least one analysis type is selected.
- Resolve key to ID via `epicRepo` or `featureRepo` (need these as optional fields on TaskService, or pass them via the service accessor).
- Call `workSessionRepo.GetSessionAnalyticsByEpic` or `GetSessionAnalyticsByFeature`.
- Map `repository.SessionAnalytics` to `AnalyticsResult`.

**Note on dependencies**: `GetTaskServiceWithDeps()` already wires `workSessionRepo`. Add optional `epicRepo` and `featureRepo` fields to `TaskService` via setter methods, or pass them through the analytics input. The recommended approach is to add `SetEpicRepo` and `SetFeatureRepo` setters matching the existing `SetDepRepo` pattern, and wire them in `GetTaskServiceWithDeps()`.

### 4.3 NoteService: GetNoteTimeline

Used by `task_note.go` (the `task timeline` subcommand).

#### 4.3.1 New NoteService Repository Interface Extension

```go
// NoteTimelineHistoryRepository defines the history repo interface needed for timeline.
// Add to note_service.go (consumer side).
type NoteTimelineHistoryRepository interface {
    GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error)
    GetRejectionHistoryForTask(ctx context.Context, taskID int64) ([]*models.TaskHistory, error)
}
```

Add optional field to `NoteService` via setter:
```go
// SetHistoryRepo sets the history repository for timeline methods.
func (s *NoteService) SetHistoryRepo(historyRepo NoteTimelineHistoryRepository)
```

#### 4.3.2 DTOs

```go
// TimelineEvent represents a unified timeline event (status change or note).
type TimelineEvent struct {
    Timestamp time.Time `json:"timestamp"`
    EventType string    `json:"event_type"` // "status", note type string, "rejection"
    Content   string    `json:"content"`
    Actor     string    `json:"actor,omitempty"`
}

// NoteTimeline aggregates status history and notes in chronological order.
type NoteTimeline struct {
    TaskKey string          `json:"task_key"`
    Events  []TimelineEvent `json:"events"`
}
```

#### 4.3.3 New NoteService Method

```go
// GetTaskTimeline aggregates task status history and notes into a unified chronological timeline.
// Returns events sorted by timestamp (oldest first).
// Requires the historyRepo to be set via SetHistoryRepo; returns empty events list if not set.
func (s *NoteService) GetTaskTimeline(
    ctx context.Context,
    taskKey string,
) (*NoteTimeline, error)
```

**Internal logic:**
- Get task via `s.taskRepo.GetByKey` (NoteService already has task repo).
- Add task creation event.
- Get history via `s.historyRepo.GetHistoryByTaskKey` (convert to TimelineEvent).
- Get rejection history via `s.historyRepo.GetRejectionHistoryForTask` (convert to TimelineEvent).
- Get notes via `s.noteRepo.GetByEntity` (convert to TimelineEvent).
- Merge and sort all events by timestamp.
- Return `NoteTimeline`.

### 4.4 ResumeService: GetTaskResume

Used by `task_resume.go`.

#### 4.4.1 New Repository Interfaces for ResumeService

```go
// ResumeWorkSessionRepository defines the work session repo interface needed by ResumeService.
type ResumeWorkSessionRepository interface {
    GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error)
    GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*services.WorkSessionStats, error)
    GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error)
}
```

The concrete `*workSessionAdapter` (already in `service_accessors.go`) implements this interface.

Add optional field to `ResumeService` via setter:
```go
// SetWorkSessionRepo sets the work session repository for task resume context.
func (s *ResumeService) SetWorkSessionRepo(repo ResumeWorkSessionRepository)
```

#### 4.4.2 DTOs

```go
// TaskResumeContext aggregates all context needed to resume a task.
type TaskResumeContext struct {
    Task          *models.Task               `json:"task"`
    ContextData   *models.ContextData        `json:"context_data,omitempty"`
    Notes         []*models.EntityNote       `json:"notes,omitempty"`
    WorkSessions  []*models.WorkSession      `json:"work_sessions,omitempty"`
    SessionStats  *WorkSessionStats          `json:"session_stats,omitempty"`
    ActiveSession *models.WorkSession        `json:"active_session,omitempty"`
    Dependencies  []string                   `json:"dependencies,omitempty"`
}
```

#### 4.4.3 New ResumeService Method

Extend the `ResumeTaskRepository` interface to include `GetByKey`:

```go
// ResumeTaskRepository extends the existing interface.
type ResumeTaskRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Task, error)        // ADD
    GetContextData(ctx context.Context, taskID int64) (*string, error)     // ADD
    ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
    ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
}
```

```go
// GetTaskResume aggregates all context needed to resume work on a task.
// Includes: task details, context data, notes, work sessions, active session, dependencies.
func (s *ResumeService) GetTaskResume(
    ctx context.Context,
    taskKey string,
) (*TaskResumeContext, error)
```

**Internal logic:**
- Get task by key via `s.taskRepo.GetByKey`.
- Get context data via `s.taskRepo.GetContextData`.
- Get notes via `s.noteRepo.GetByEntity`.
- Get work sessions via `s.workSessionRepo.GetByTaskID` (if set).
- Get session stats via `s.workSessionRepo.GetSessionStatsByTaskID` (if set).
- Get active session via `s.workSessionRepo.GetActiveSessionByTaskID` (if set).
- Get dependencies from task (they are stored as a JSON field on `models.Task`, or via `dependsOn` list from `task.DependsOn`).
- Return `TaskResumeContext`.

---

## 5. Global Service Accessors

New and updated accessors needed in `internal/cli/services_global.go` and `internal/cli/service_accessors.go`.

### 5.1 GetCriteriaService (new)

Add to `internal/cli/service_accessors.go`:

```go
// GetCriteriaService returns a CriteriaService instance.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing pattern for CLI entry points).
func GetCriteriaService() *services.CriteriaService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    criteriaRepo := repository.NewTaskCriteriaRepository(db)
    taskRepo := repository.NewTaskRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    return services.NewCriteriaService(criteriaRepo, taskRepo, featureRepo, taskRepo)
}
```

Note: `taskRepo` implements both `CriteriaTaskRepository` and `CriteriaTaskListRepository` since `TaskRepository` already has `ListByFeature`.

### 5.2 GetTaskService - Update for History

Update `GetTaskService()` and `GetTaskServiceWithDeps()` in `services_global.go` to wire the history repo:

```go
// In GetTaskService():
historyRepo := repository.NewTaskHistoryRepository(db)
svc := services.NewTaskService(taskRepo, workflowSvc, creatorSvc, noteRepo)
svc.SetHistoryRepo(historyRepo)
return svc

// In GetTaskServiceWithDeps():
historyRepo := repository.NewTaskHistoryRepository(db)
svc := services.NewTaskServiceWithRelationships(...)
svc.SetHistoryRepo(historyRepo)
svc.SetEpicRepo(epicRepo)     // for analytics
svc.SetFeatureRepo(featureRepo) // for analytics
return svc
```

### 5.3 GetNoteService - Update for Timeline

Update `GetNoteService()` in `services_global.go` to wire the history repo:

```go
// In GetNoteService():
historyRepo := repository.NewTaskHistoryRepository(db)
svc := services.NewNoteService(noteRepo, epicRepo, featureRepo, taskRepo)
svc.SetHistoryRepo(historyRepo)
return svc
```

### 5.4 GetResumeService - Update for Task Resume

Update `GetResumeService()` in `services_global.go` to wire work sessions:

```go
// In GetResumeService():
sessionRepo := &workSessionAdapter{repo: repository.NewWorkSessionRepository(db)}
svc := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo)
svc.SetWorkSessionRepo(sessionRepo)
return svc
```

---

## 6. Migration Approach by Group

Migrations should proceed in this order within each task to manage risk.

### Group A: Low-Effort Migrations (existing services, new method calls)

**Files**: `task_context.go`, `notes_search.go`, `task_link.go`

**task_context.go** (397 lines → ~40 lines after):
- `runTaskContextSet` → call `contextSvc.SetContextField(ctx, models.EntityTypeTask, key, field, value)`
- `runTaskContextGet` → call `contextSvc.GetContext(ctx, models.EntityTypeTask, key)`
- `runTaskContextClear` → call `contextSvc.ClearContext(ctx, models.EntityTypeTask, key)`
- All three service methods already exist in `ContextService`.
- Add `GetContextService` to accessor pattern (already in `services_global.go`).

**notes_search.go** (230 lines → ~35 lines after):
- `runNotesSearch` → call `noteSvc.SearchNotes(ctx, query, noteTypes, entityType, epicKey, featureKey)`
- `SearchNotes` already exists in `NoteService`.
- The `NoteSearchResult` struct mapping (adding entity context) is presentation logic; keep in command.

**task_link.go** (198 lines → ~30 lines after):
- `runTaskLink` → call `taskSvc.LinkRelationships(ctx, taskKey, relMap)` (or call `AddDependency` per type)
- Check if `TaskService.LinkRelationships` exists or needs a new method that accepts a map of `relType -> []targetKeys`.
- Current `TaskService` has `AddDependency` for `depends_on`. A new `CreateRelationships` method may be needed to batch-create relationships of multiple types.

**New method needed on TaskService** (if not present):
```go
// CreateRelationships creates typed relationships from a source task to target tasks.
// The relationships map keys are relationship type strings; values are target task keys.
func (s *TaskService) CreateRelationships(
    ctx context.Context,
    sourceTaskKey string,
    relationships map[string][]string,
) ([]RelationshipWithTask, error)
```

### Group B: Moderate-Effort Migrations (new service methods required)

**Files**: `task_history.go`, `history.go`, `task_resume.go`, `analytics.go`

**task_history.go** (247 lines → ~30 lines after):
- `runTaskHistory` → call `taskSvc.GetTaskHistory(ctx, taskKey)`
- New method `GetTaskHistory` on `TaskService` (§4.1.3).
- The `HistoryOutput` / `HistoryEntry` output structs are presentation DTOs; keep in command file.

**history.go** (277 lines → ~40 lines after):
- `runHistory` → call `taskSvc.ListHistory(ctx, historyFilters)`
- New method `ListHistory` on `TaskService` (§4.1.3).
- The `historyFilters` parsing (epic/feature key positional args) stays in command.
- CSV formatting stays in command.

**task_resume.go** (401 lines → ~35 lines after):
- `runTaskResume` → call `resumeSvc.GetTaskResume(ctx, taskKey)`
- New method `GetTaskResume` on `ResumeService` (§4.4.3).
- The existing `ResumeContext` output struct is presentation; keep in command or move to service as `TaskResumeContext`.
- The `repository.SessionStats` in the current command's `ResumeContext` should become `services.WorkSessionStats`.

**analytics.go** (238 lines → ~35 lines after):
- `runAnalytics` → call `taskSvc.GetAnalytics(ctx, analyticsInput)`
- New method `GetAnalytics` on `TaskService` (§4.2.2).
- Flag parsing/validation logic moves to service (validate at least one analysis type, validate scope).
- Actually: validation that flags are present is acceptable in command layer (argument parsing); the actual analytics computation moves to service.

### Group C: High-Effort Migrations (complex subcommands, new CriteriaService)

**Files**: `task_criteria.go`, `task_note.go`, `task_next_status.go`, `feature_criteria.go`

**task_criteria.go** (460 lines → ~80 lines after):
- All `runTaskCriteria*` functions → call `criteriaSvc.*` methods (§3.5).
- Multiple subcommands: import, list, check, fail, update, na, reset (verify full list in file).
- Need a `GetCriteriaService()` accessor.
- The summary display formatting logic stays in command.

**task_note.go** (442 lines → ~60 lines after):
- `runTaskNoteAdd` → call `noteSvc.AddNote(ctx, models.EntityTypeTask, taskKey, noteType, content, createdBy)`
- `runTaskNotes` → call `noteSvc.ListNotes(ctx, models.EntityTypeTask, taskKey, noteTypes)`
- `runTaskTimeline` → call `noteSvc.GetTaskTimeline(ctx, taskKey)` (new method, §4.3).
- The timeline display sorting and truncation logic stays in command.

**task_next_status.go** (361 lines → ~80 lines after):
- `runTaskNextStatus` → call `taskSvc.GetNextStatus(ctx, taskKey)` for preview, `taskSvc.TransitionStatus(ctx, taskKey, targetStatus, opts)` for transition.
- Both methods already exist on `TaskService`.
- **Critical**: The interactive prompt (reading from stdin when no `--status` flag) stays in the CLI command layer. This is input/output concern, not business logic.
- **Critical**: The `triggerStatusCascade` function currently in this file must move into `TaskService.TransitionStatus` if not already there. Verify `TransitionStatus` already handles cascade (it should based on PRD).
- The `NextStatusResult` output struct stays in command.
- The `TransitionChoice` display struct stays in command.

**feature_criteria.go** (195 lines → ~30 lines after):
- `runFeatureCriteria` → call `criteriaSvc.GetFeatureCriteria(ctx, featureKey, byTask)` (§3.5).
- The `FeatureCriteriaSummary` / `TaskCriteriaSummary` output structs currently in command become `services.FeatureCriteriaResult` / `services.TaskCriteriaSummary`.

### Group D: View Command (path resolution only)

**view.go** (104 lines → ~60 lines after):
- The business logic is: resolve entity key → get file path.
- Use existing service methods: `epicSvc.GetEpic(ctx, key)`, `featureSvc.GetFeature(ctx, key)`, `taskSvc.GetTask(ctx, key)` to get the entity, then access `entity.FilePath`.
- The `os.Exec` / viewer invocation stays in CLI layer.
- The `scope.NewInterpreter().ParseScope(args)` stays in CLI layer (it's argument parsing).
- Reduce repository calls: replace `repository.NewEpicRepository(repoDb)` → `cli.GetEpicService().GetEpic(...)`.

---

## 7. Test Requirements

### 7.1 New Service Tests

For each new service and new method, create tests in the service package using mocked repositories.

**CriteriaService tests** (`internal/services/criteria_service_test.go`):
- `TestCriteriaService_ImportCriteria_Success`
- `TestCriteriaService_ImportCriteria_TaskNotFound`
- `TestCriteriaService_ImportCriteria_NoFilePath`
- `TestCriteriaService_ListCriteria_Success`
- `TestCriteriaService_CheckCriterion_Success`
- `TestCriteriaService_FailCriterion_RequiresNote`
- `TestCriteriaService_GetFeatureCriteria_AggregatesCorrectly`

**TaskService new method tests** (add to `internal/services/task_service_test.go`):
- `TestTaskService_GetTaskHistory_Success`
- `TestTaskService_GetTaskHistory_TaskNotFound`
- `TestTaskService_ListHistory_WithFilters`
- `TestTaskService_GetAnalytics_ByEpic`
- `TestTaskService_GetAnalytics_ByFeature`
- `TestTaskService_GetAnalytics_ValidationErrors`
- `TestTaskService_CreateRelationships_MultipleTpes`

**NoteService new method tests** (add to `internal/services/note_service_test.go`):
- `TestNoteService_GetTaskTimeline_Success`
- `TestNoteService_GetTaskTimeline_NoHistoryRepo`
- `TestNoteService_GetTaskTimeline_MergesAndSorts`

**ResumeService new method tests** (add to `internal/services/resume_service_test.go` - needs creation):
- `TestResumeService_GetTaskResume_Success`
- `TestResumeService_GetTaskResume_TaskNotFound`
- `TestResumeService_GetTaskResume_WithWorkSessions`

### 7.2 CLI Command Tests

For each migrated command file, the corresponding test files must not use `test.GetTestDB()` for command-level tests. Mock the service instead.

**Implementation pattern for CLI tests**:

The current CLI test pattern uses `cli.GetDB` under the hood. For commands that previously used direct repo access, the tests should be rewritten to test the command parsing and output formatting with a mocked service:

```go
// Example pattern - override the global service accessor for testing
func TestRunTaskHistory(t *testing.T) {
    // Set up mock service via dependency injection in service accessor
    // OR test the command by verifying service method called with correct args
    // Use the existing mock_task_repository.go as a model
}
```

Files requiring CLI test updates (priority order):
1. `task_criteria_test.go` - already exists, update to use mock service
2. `task_history_test.go` - already exists, update to use mock service
3. `notes_search_filter_test.go` / `notes_search_integration_test.go` - update integration tests
4. `task_note_timeline_test.go` - update to use mock service
5. New tests for `analytics.go`, `history.go`, `task_resume.go`, `task_link.go`, `view.go`

---

## 8. Architecture Decisions

### ADR-001: validate.go - Accepted Infrastructure Exception

**Context**: `validate.go` uses `validation.NewValidator(epicRepo, featureRepo, taskRepo)` which is an adapter pattern already encapsulating repository logic.

**Decision**: Accept `validate.go` as an infrastructure exception. The `validation` package already provides proper abstraction. Creating a `ValidationService` that wraps the existing `validation` package provides no architectural value.

**Criteria for exception**: The command calls a package-level constructor (`validation.NewValidator()`), not `repository.New*` or `cli.GetDB` for business logic. The validation package is the boundary.

**Action**: Document this as an accepted pattern. Remove from the "fat controller" count. The grep check in AC-001 will still flag it - update the acceptance criteria grep to also exclude `validate.go`.

### ADR-002: CriteriaService as New Service vs Extension of TaskService

**Context**: Task and feature criteria could either be a new `CriteriaService` or methods added to `TaskService` / `FeatureService`.

**Decision**: Create a new `CriteriaService`. Reasoning: criteria span both task and feature domains, cannot cleanly belong to one. The aggregate is `Criteria`, not `Task` or `Feature`. Adding criteria methods to `TaskService` would make it a god service. A dedicated service follows the single-responsibility principle.

**Consequence**: One new file, one new global accessor.

### ADR-003: Interactive Prompt Stays in CLI for task_next_status.go

**Context**: `task_next_status.go` reads from stdin when no `--status` flag is provided. This is interactive selection logic.

**Decision**: The interactive prompt (reading user input, displaying numbered choices) stays in the CLI command layer. This is an I/O concern, not a business rule. The service provides the list of available transitions via `GetNextStatus()`; the CLI displays them and reads the selection.

**Consequence**: `runTaskNextStatus` remains non-trivial (~80 lines) but contains only: flag parsing, interactive prompt logic, service call, output formatting.

### ADR-004: History Repository as Optional Dependency via Setter

**Context**: Adding `historyRepo` to `TaskService` and `NoteService` requires changing constructor signatures, which would break existing callers.

**Decision**: Use the setter pattern (`SetHistoryRepo`) established by `SetDepRepo`, `SetRelQueryRepo`, `SetWritableDocRepo`. This avoids breaking changes to constructors.

**Consequence**: History methods return empty results gracefully if `historyRepo` not set. Service accessors wire it in.

### ADR-005: Analytics Depends on EpicRepo and FeatureRepo in TaskService

**Context**: `GetAnalytics` needs to resolve epic/feature keys to IDs. `TaskService` doesn't currently have epic/feature repo references.

**Decision**: Add `SetEpicRepo` and `SetFeatureRepo` setters to `TaskService`, following the established setter pattern. Wire them only in `GetTaskServiceWithDeps()` since analytics is a higher-capability operation. `GetTaskService()` does not need them.

**Consequence**: `GetTaskServiceWithDeps()` is the correct accessor for `analytics.go`. Already used by `task_link.go` equivalent operations.

---

## 9. Implementation Order

This order minimizes risk and enables parallel development across tasks.

### Phase 1: Service Method Additions (prerequisite for all migrations)

Implement in this order to unblock migrations:

1. `TaskService.GetTaskHistory` + `TaskService.ListHistory` + `SetHistoryRepo`
2. `NoteService.GetTaskTimeline` + `NoteService.SetHistoryRepo`
3. `ResumeService.GetTaskResume` + `ResumeService.SetWorkSessionRepo`
4. `TaskService.GetAnalytics` + `TaskService.SetEpicRepo` + `TaskService.SetFeatureRepo`
5. `TaskService.CreateRelationships` (check if it already exists)
6. `CriteriaService` (new file) with all 5 methods
7. Update all global service accessors to wire new dependencies

### Phase 2: Group A Migrations (low-effort, high confidence)

8. `task_context.go` - call `ContextService` methods
9. `notes_search.go` - call `NoteService.SearchNotes`
10. `task_link.go` - call `TaskService.CreateRelationships`

### Phase 3: Group B Migrations

11. `task_history.go` - call `TaskService.GetTaskHistory`
12. `history.go` - call `TaskService.ListHistory`
13. `task_resume.go` - call `ResumeService.GetTaskResume`
14. `analytics.go` - call `TaskService.GetAnalytics`

### Phase 4: Group C Migrations (high-effort)

15. `task_note.go` - call `NoteService` methods + `GetTaskTimeline`
16. `feature_criteria.go` - call `CriteriaService.GetFeatureCriteria`
17. `task_criteria.go` - call all `CriteriaService` methods
18. `task_next_status.go` - call `TaskService.GetNextStatus` + `TransitionStatus`

### Phase 5: Group D + Test Compliance

19. `view.go` - call entity service methods for path resolution
20. Update CLI tests to use mock services
21. Run `make fmt && make lint && make test` - verify zero failures

---

## 10. Verification Criteria

After implementation is complete:

```bash
# AC-001: Zero fat controllers (excluding infrastructure and validate.go)
grep -rn "repository\.New\|cli\.GetDB\|repoDb" internal/cli/commands/ \
  --include="*.go" \
  | grep -v "_test\.go" \
  | grep -v "mock_" \
  | grep -v "cloud\.go" \
  | grep -v "init\.go" \
  | grep -v "migrate_backfill_slugs\.go" \
  | grep -v "validate\.go"
# Expected: empty output

# AC-002: All tests pass
make fmt && make lint && make test
# Expected: 0 failures, 0 lint errors

# AC-003: Build succeeds
make build
# Expected: successful build
```

---

## 11. Key Files and Locations

| Artifact | Location |
|----------|----------|
| New `CriteriaService` | `internal/services/criteria_service.go` |
| New `CriteriaService` tests | `internal/services/criteria_service_test.go` |
| Updated `TaskService` (new methods) | `internal/services/task_service.go` |
| Updated `NoteService` (timeline) | `internal/services/note_service.go` |
| Updated `ResumeService` (task resume) | `internal/services/resume_service.go` |
| New global accessor `GetCriteriaService` | `internal/cli/service_accessors.go` |
| Updated service accessors (wire new deps) | `internal/cli/services_global.go` |
| 13 migrated CLI command files | `internal/cli/commands/*.go` |

---

*Specification created*: 2026-02-19
*Architect*: Architect Agent (E15-F11 technical review)
