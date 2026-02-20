# Technical Design: E15-F11 Service Layer Completion and CLI Integration

**Feature**: E15-F11 Service Layer Completion and CLI Integration
**Author**: Architect (Claude Sonnet 4.6)
**Date**: 2026-02-19
**Status**: Ready for Test Planning

---

## 1. Overview

This document provides the technical design for completing the service layer migration for
Epic E15. It covers six new service methods that must be added, interface additions required
on existing repository contracts, and the migration approach for each of the 13 remaining
fat-controller CLI files.

The goal is to eliminate all direct repository access from CLI commands so that every command
follows the three-step thin-wrapper pattern: **parse → call service → format output**.

---

## 2. Current State Analysis

### 2.1 Already Complete (Do Not Re-implement)

The following tasks from the research report are already done:

- **T-E15-F11-006**: `CriteriaService` is fully implemented at `internal/services/criteria_service.go`
  (276 lines). `ImportCriteria`, `ListCriteria`, `CheckCriterion`, `FailCriterion`,
  `GetFeatureCriteria` all exist.
- **T-E15-F11-010**: `GetCriteriaService(ctx)` global accessor is registered in
  `internal/cli/services_global.go`.
- **`GetTaskServiceWithDeps()`**: Already wires `sessionRepo` via `workSessionAdapter`,
  `relRepo`, `docRepo`. This is the correct accessor to use for analytics commands.

### 2.2 Existing Stubs (Must Be Implemented)

Two methods in `TaskService` exist but return `fmt.Errorf("not implemented")`:

```go
// internal/services/task_service.go
func (s *TaskService) TransitionStatus(ctx context.Context, key string, ...) error {
    return fmt.Errorf("not implemented")
}

func (s *TaskService) GetNextStatus(ctx context.Context, key string) ([]string, error) {
    return nil, fmt.Errorf("not implemented")
}
```

These are required before `task_next_status.go` can be migrated.

### 2.3 Fat-Controller Files Requiring Migration

| File | Repositories Used | Complexity |
|------|-------------------|------------|
| `analytics.go` | `WorkSessionRepository`, `EpicRepository`, `FeatureRepository` | Medium |
| `task_history.go` | `TaskHistoryRepository`, `TaskRepository` | Low |
| `task_link.go` | `TaskRepository`, `TaskRelationshipRepository` | Medium |
| `task_next_status.go` | `TaskRepository` (workflow-aware variant), `config`, `workflow` | High |
| `notes_search.go` | `EntityNoteRepository`, `TaskRepository`, `EpicRepository`, `FeatureRepository` | Medium |
| `task_resume.go` | `TaskRepository`, `EntityNoteRepository`, `WorkSessionRepository` | High |
| `task_note.go` | `TaskRepository`, `EntityNoteRepository` | Low |
| `task_context.go` | `TaskRepository` | Low |
| `feature_criteria.go` | `FeatureRepository`, `TaskRepository`, `TaskCriteriaRepository` | Medium |
| `task_criteria.go` | `TaskCriteriaRepository`, `TaskRepository`, `taskfile` | Medium |
| `view.go` | `EpicRepository`, `FeatureRepository`, `TaskRepository` | Low |
| `validate.go` | `EpicRepository`, `FeatureRepository`, `TaskRepository` | Low |
| `history.go` (smart dispatcher) | `TaskHistoryRepository` | Low |

---

## 3. New Service Methods Required

### 3.1 TaskService.GetTaskHistory

**Purpose**: Retrieve task status history by task key. Used by `task_history.go`.

**File**: `internal/services/task_service.go`

**New interface addition to `TaskRepository`**:

```go
// Add to TaskRepository interface in task_service.go
type TaskHistoryRepository interface {
    GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error)
}
```

**Service field addition**:

```go
type TaskService struct {
    // ... existing fields ...
    historyRepo TaskHistoryRepository  // optional
}
```

Note: `GetTaskService()` in `services_global.go` already wires `historyRepo` via
`services.NewTaskService`. Confirm this wiring handles the repository.

Actually, looking at the current constructor `NewTaskService(repo, workflowSvc, creatorSvc, noteRepo)`,
`historyRepo` is not a parameter. The design must either:

**Option A** (preferred - avoid constructor change): Add a `SetHistoryRepo` setter method
on TaskService, consistent with the existing `SetDepRepo`, `SetRelQueryRepo` pattern.

**Option B**: Extend `NewTaskServiceWithRelationships` to accept `historyRepo`.

Use Option A to avoid breaking existing callers.

**Go Signature**:

```go
// SetHistoryRepo sets the optional task history repository.
// Called by GetTaskServiceWithHistory() accessor.
func (s *TaskService) SetHistoryRepo(repo TaskHistoryRepository) {
    s.historyRepo = repo
}

// GetTaskHistory returns the status transition history for a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key (e.g., "E07-F01-001")
//
// Returns the history records in chronological order, or an empty
// slice if the task exists but has no history.
// Returns an error if the task is not found.
func (s *TaskService) GetTaskHistory(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
    // Verify task exists first
    _, err := s.repo.GetByKey(ctx, taskKey)
    if err != nil {
        return nil, fmt.Errorf("task %s not found: %w", taskKey, err)
    }

    if s.historyRepo == nil {
        return nil, fmt.Errorf("history repository not configured")
    }

    histories, err := s.historyRepo.GetHistoryByTaskKey(ctx, taskKey)
    if err != nil {
        return nil, fmt.Errorf("failed to get history for task %s: %w", taskKey, err)
    }

    return histories, nil
}
```

**New accessor in `services_global.go`**:

```go
// GetTaskServiceWithHistory returns a TaskService wired with history repository.
func GetTaskServiceWithHistory() *services.TaskService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    svc := GetTaskService()
    historyRepo := repository.NewTaskHistoryRepository(db)
    svc.SetHistoryRepo(historyRepo)
    return svc
}
```

**Required interface change in `task_service.go`**:

Add `TaskHistoryRepository` interface (defined above) to the file.

---

### 3.2 TaskService.GetSessionAnalytics

**Purpose**: Retrieve work session analytics by feature or epic scope. Used by `analytics.go`.

**Existing**: `GetWorkSessions(ctx, taskKey)` only handles single-task scope.
`repository.WorkSessionRepository` already has `GetSessionAnalyticsByFeature` and
`GetSessionAnalyticsByEpic` methods. The existing `WorkSessionRepository` interface in
`task_service.go` does NOT include these — it only has `GetByTaskID` and `GetSessionStatsByTaskID`.

**Extend the `WorkSessionRepository` interface** in `task_service.go`:

```go
// WorkSessionRepository defines the work session data access interface.
type WorkSessionRepository interface {
    GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error)
    GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*WorkSessionStats, error)
    // New methods for analytics:
    GetSessionAnalyticsByFeature(ctx context.Context, featureID int64, agentType *string) (*repository.SessionAnalytics, error)
    GetSessionAnalyticsByEpic(ctx context.Context, epicID int64, agentType *string) (*repository.SessionAnalytics, error)
}
```

**Note**: This requires importing `repository` package in `task_service.go`. Alternatively,
define a local `SessionAnalytics` struct in the services package and map from the repository type.
The cleaner approach is to define a `services.SessionAnalytics` struct in a new
`internal/services/analytics_dto.go` file to avoid the circular dependency risk.

**New DTO** in `internal/services/analytics_dto.go`:

```go
// SessionAnalyticsInput defines filters for session analytics queries.
type SessionAnalyticsInput struct {
    EpicKey    string
    FeatureKey string
    AgentType  string
}

// SessionAnalyticsResult wraps analytics data with scope information.
type SessionAnalyticsResult struct {
    Scope     string                    `json:"scope"`
    Analytics *repository.SessionAnalytics `json:"analytics"`
}
```

**Service methods**:

```go
// GetSessionAnalytics returns work session analytics for a given scope.
//
// Exactly one of input.EpicKey or input.FeatureKey must be set.
// Returns an error if neither or both are set.
func (s *TaskService) GetSessionAnalytics(ctx context.Context, input SessionAnalyticsInput) (*SessionAnalyticsResult, error) {
    if input.EpicKey == "" && input.FeatureKey == "" {
        return nil, fmt.Errorf("either epic key or feature key must be specified")
    }
    if input.EpicKey != "" && input.FeatureKey != "" {
        return nil, fmt.Errorf("specify either epic key or feature key, not both")
    }

    if s.sessionRepo == nil {
        return nil, fmt.Errorf("session repository not configured")
    }

    var agentTypePtr *string
    if input.AgentType != "" {
        agentTypePtr = &input.AgentType
    }

    if input.FeatureKey != "" {
        feature, err := s.featureRepo.GetByKey(ctx, input.FeatureKey)
        if err != nil {
            return nil, fmt.Errorf("feature %s not found: %w", input.FeatureKey, err)
        }
        analytics, err := s.sessionRepo.GetSessionAnalyticsByFeature(ctx, feature.ID, agentTypePtr)
        if err != nil {
            return nil, fmt.Errorf("failed to get analytics for feature %s: %w", input.FeatureKey, err)
        }
        scope := fmt.Sprintf("Feature %s", input.FeatureKey)
        if input.AgentType != "" {
            scope += fmt.Sprintf(" (Agent: %s)", input.AgentType)
        }
        return &SessionAnalyticsResult{Scope: scope, Analytics: analytics}, nil
    }

    // Epic-level analytics
    epic, err := s.epicRepo.GetByKey(ctx, input.EpicKey)
    if err != nil {
        return nil, fmt.Errorf("epic %s not found: %w", input.EpicKey, err)
    }
    analytics, err := s.sessionRepo.GetSessionAnalyticsByEpic(ctx, epic.ID, agentTypePtr)
    if err != nil {
        return nil, fmt.Errorf("failed to get analytics for epic %s: %w", input.EpicKey, err)
    }
    scope := fmt.Sprintf("Epic %s", input.EpicKey)
    if input.AgentType != "" {
        scope += fmt.Sprintf(" (Agent: %s)", input.AgentType)
    }
    return &SessionAnalyticsResult{Scope: scope, Analytics: analytics}, nil
}
```

**Required fields added to TaskService struct**:

```go
epicRepo    TaskEpicRepository    // needed for analytics scope resolution
featureRepo TaskFeatureRepository // needed for analytics scope resolution
```

These are new interfaces to define in `task_service.go`:

```go
// TaskEpicRepository defines epic lookups needed by TaskService.
type TaskEpicRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Epic, error)
}

// TaskFeatureRepository defines feature lookups needed by TaskService.
type TaskFeatureRepository interface {
    GetByKey(ctx context.Context, key string) (*models.Feature, error)
}
```

**Setter methods**:

```go
func (s *TaskService) SetEpicRepo(repo TaskEpicRepository)    { s.epicRepo = repo }
func (s *TaskService) SetFeatureRepo(repo TaskFeatureRepository) { s.featureRepo = repo }
```

**Accessor**: Use `GetTaskServiceWithDeps()` which already creates the service with a
`workSessionAdapter` wrapping `repository.NewWorkSessionRepository(db)`. Extend it to also
call `SetEpicRepo` and `SetFeatureRepo`.

---

### 3.3 TaskService.CreateRelationship

**Purpose**: Create typed task relationships for all 7 relationship types. `AddDependency()`
only handles `depends_on`. Used by `task_link.go`.

**Existing**: `AddDependency(ctx, taskKey, dependsOnKey string)` in `task_service.go`.
`TaskRelationshipRepository` interface is not directly in `task_service.go` — the relRepo field
is of type `TaskRelationshipRepository` which has methods:
`Create`, `GetOutgoing`, `GetIncoming`, `DetectCycle`, `DeleteByType`, `Delete`.

**New DTO**:

```go
// CreateRelationshipInput defines parameters for creating a typed task relationship.
type CreateRelationshipInput struct {
    FromTaskKey string
    ToTaskKey   string
    RelType     models.RelationshipType
}
```

**Service method**:

```go
// CreateRelationship creates a typed relationship between two tasks.
//
// Supported relationship types: depends_on, blocks, related_to, follows,
// spawned_from, duplicates, references.
//
// For depends_on and blocks relationships, cycle detection is performed
// before creation.
//
// Returns the created relationship or an error. Returns a conflict error
// if the relationship already exists.
func (s *TaskService) CreateRelationship(ctx context.Context, input CreateRelationshipInput) (*models.TaskRelationship, error) {
    if s.relRepo == nil {
        return nil, fmt.Errorf("relationship repository not configured")
    }

    fromTask, err := s.repo.GetByKey(ctx, input.FromTaskKey)
    if err != nil {
        return nil, fmt.Errorf("source task %s not found: %w", input.FromTaskKey, err)
    }

    toTask, err := s.repo.GetByKey(ctx, input.ToTaskKey)
    if err != nil {
        return nil, fmt.Errorf("target task %s not found: %w", input.ToTaskKey, err)
    }

    // Cycle detection for dependency-type relationships
    if input.RelType == models.RelationshipTypeDependsOn || input.RelType == models.RelationshipTypeBlocks {
        if err := s.relRepo.DetectCycle(ctx, fromTask.ID, toTask.ID, string(input.RelType)); err != nil {
            return nil, fmt.Errorf("circular dependency detected: %w", err)
        }
    }

    rel := &models.TaskRelationship{
        FromTaskID:       fromTask.ID,
        ToTaskID:         toTask.ID,
        RelationshipType: input.RelType,
    }

    if err := s.relRepo.Create(ctx, rel); err != nil {
        return nil, fmt.Errorf("failed to create relationship: %w", err)
    }

    return rel, nil
}
```

**Accessor**: Use `GetTaskServiceWithDeps()` — already wires `relRepo`.

---

### 3.4 NoteService.SearchNotesWithTimePeriod

**Purpose**: Search notes with optional time period filtering. Used by `notes_search.go`.

**Existing situation**:
- `NoteService.SearchNotes(ctx, query, noteTypes, entityType, epicKey, featureKey)` exists and
  delegates to `noteRepo.Search(...)`.
- `repository.EntityNoteRepository.SearchWithTimePeriod(ctx, query, noteTypes, epicKey,
  featureKey, since, until)` exists at the repository level (line 274 in entity_note_repository.go).
- The `NoteEntityNoteRepository` interface in `note_service.go` does NOT include
  `SearchWithTimePeriod` — only `Search`.

**Add to `NoteEntityNoteRepository` interface** in `note_service.go`:

```go
type NoteEntityNoteRepository interface {
    // ... existing methods ...
    SearchWithTimePeriod(ctx context.Context, query string, noteTypes []string,
        epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error)
}
```

**New DTO**:

```go
// NoteSearchInput defines parameters for searching notes.
type NoteSearchInput struct {
    Query      string
    NoteTypes  []string
    EntityType *models.EntityType
    EpicKey    string
    FeatureKey string
    Since      string // YYYY-MM-DD format, optional
    Until      string // YYYY-MM-DD format, optional
}
```

**Service method**:

```go
// SearchNotes searches for notes matching the query and filters.
//
// When Since or Until is set, time-period filtering is applied.
// The entity context (task title, epic title, feature title) is
// resolved for each matching note.
//
// Returns NoteSearchResult slice ready for output.
func (s *NoteService) SearchNotes(ctx context.Context, input NoteSearchInput) ([]*NoteSearchResult, error) {
    var notes []*models.EntityNote
    var err error

    if input.Since != "" || input.Until != "" {
        notes, err = s.noteRepo.SearchWithTimePeriod(ctx, input.Query, input.NoteTypes,
            input.EpicKey, input.FeatureKey, input.Since, input.Until)
    } else {
        notes, err = s.noteRepo.Search(ctx, input.Query, input.NoteTypes,
            input.EntityType, input.EpicKey, input.FeatureKey)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to search notes: %w", err)
    }

    results := make([]*NoteSearchResult, 0, len(notes))
    for _, note := range notes {
        result, err := s.resolveNoteContext(ctx, note)
        if err != nil {
            continue // Skip notes where entity context can't be resolved
        }
        results = append(results, result)
    }

    return results, nil
}
```

**New result type** in `note_service.go` or `internal/services/note_dto.go`:

```go
// NoteSearchResult wraps a note with its entity context.
type NoteSearchResult struct {
    EntityType string             `json:"entity_type"`
    EntityKey  string             `json:"entity_key"`
    EntityName string             `json:"entity_name"`
    Note       *models.EntityNote `json:"note"`
    // Backward-compatible task fields
    TaskKey   string `json:"task_key,omitempty"`
    TaskTitle string `json:"task_title,omitempty"`
}
```

**Private helper** (resolves entity key/name per note):

```go
func (s *NoteService) resolveNoteContext(ctx context.Context, note *models.EntityNote) (*NoteSearchResult, error) {
    result := &NoteSearchResult{
        EntityType: string(note.EntityType),
        Note:       note,
    }
    switch note.EntityType {
    case models.EntityTypeTask:
        task, err := s.taskRepo.GetByID(ctx, note.EntityID)
        if err != nil {
            return nil, err
        }
        result.EntityKey = task.Key
        result.EntityName = task.Title
        result.TaskKey = task.Key
        result.TaskTitle = task.Title
    case models.EntityTypeEpic:
        epic, err := s.epicRepo.GetByID(ctx, note.EntityID)
        if err != nil {
            return nil, err
        }
        result.EntityKey = epic.Key
        result.EntityName = epic.Title
    case models.EntityTypeFeature:
        feature, err := s.featureRepo.GetByID(ctx, note.EntityID)
        if err != nil {
            return nil, err
        }
        result.EntityKey = feature.Key
        result.EntityName = feature.Title
    }
    return result, nil
}
```

**NoteService struct additions** (repositories needed for entity context resolution):

The existing `NoteService` already has `epicRepo`, `featureRepo`, `taskRepo` fields per
the architecture rules dependency graph. These are already wired in `GetNoteService(ctx)`.
Verify this is the case — if so, no struct changes are needed.

---

### 3.5 NoteService.GetNoteTimeline

**Purpose**: Return a unified chronological timeline of status history and notes for a task.
Used by `task_note.go`'s `timeline` subcommand.

**New interface needed on `NoteEntityNoteRepository`**:

The timeline requires both `task_history` records and `entity_notes` records. Since NoteService
already has `taskRepo` for entity lookup, add a `TaskHistoryLookup` interface to NoteService:

```go
// NoteTaskHistoryRepository defines the history lookup needed by NoteService for timelines.
type NoteTaskHistoryRepository interface {
    GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error)
}
```

**NoteService struct addition**:

```go
type NoteService struct {
    // ... existing fields ...
    taskHistoryRepo NoteTaskHistoryRepository // optional, for timeline support
}

func (s *NoteService) SetTaskHistoryRepo(repo NoteTaskHistoryRepository) {
    s.taskHistoryRepo = repo
}
```

**New type** in `note_service.go`:

```go
// TimelineEvent represents a unified chronological event (status change or note).
type TimelineEvent struct {
    Timestamp      time.Time `json:"timestamp"`
    EventType      string    `json:"event_type"` // "status" or note type
    Content        string    `json:"content"`
    Actor          string    `json:"actor,omitempty"`
    Reason         string    `json:"reason,omitempty"`
    ReasonDocument *string   `json:"reason_document,omitempty"`
}
```

**Service method**:

```go
// GetNoteTimeline returns a unified chronological timeline of status changes and notes
// for the given task key.
func (s *NoteService) GetNoteTimeline(ctx context.Context, taskKey string) ([]*TimelineEvent, error) {
    task, err := s.taskRepo.GetByKey(ctx, taskKey)
    if err != nil {
        return nil, fmt.Errorf("task %s not found: %w", taskKey, err)
    }

    var events []*TimelineEvent

    // Get status history
    if s.taskHistoryRepo != nil {
        histories, err := s.taskHistoryRepo.GetHistoryByTaskKey(ctx, taskKey)
        if err != nil {
            return nil, fmt.Errorf("failed to get history for task %s: %w", taskKey, err)
        }
        for _, h := range histories {
            var content string
            if h.OldStatus != nil {
                content = fmt.Sprintf("%s → %s", *h.OldStatus, h.NewStatus)
            } else {
                content = fmt.Sprintf("created as %s", h.NewStatus)
            }
            agent := ""
            if h.Agent != nil {
                agent = *h.Agent
            }
            reason := ""
            if h.RejectionReason != nil {
                reason = *h.RejectionReason
            }
            events = append(events, &TimelineEvent{
                Timestamp: h.Timestamp,
                EventType: "status",
                Content:   content,
                Actor:     agent,
                Reason:    reason,
            })
        }
    }

    // Get notes
    notes, err := s.noteRepo.GetByEntity(ctx, models.EntityTypeTask, task.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to get notes for task %s: %w", taskKey, err)
    }
    for _, n := range notes {
        actor := ""
        if n.CreatedBy != nil {
            actor = *n.CreatedBy
        }
        events = append(events, &TimelineEvent{
            Timestamp: n.CreatedAt,
            EventType: string(n.NoteType),
            Content:   n.Content,
            Actor:     actor,
        })
    }

    // Sort events by timestamp
    sort.Slice(events, func(i, j int) bool {
        return events[i].Timestamp.Before(events[j].Timestamp)
    })

    return events, nil
}
```

**Accessor update in `services_global.go`**: Add `SetTaskHistoryRepo` call to `GetNoteService`.

---

### 3.6 ResumeService.GetTaskResume

**Purpose**: Return comprehensive resume context for a task. Used by `task_resume.go`.

**Existing**: `ResumeService` has `GetEpicResume(ctx, epicKey)` and
`GetFeatureResume(ctx, featureKey)`. `ResumeTaskRepository` interface only has
`ListByFeature` and `ListByEpic`.

**Required interface extension** in `resume_service.go`:

```go
type ResumeTaskRepository interface {
    ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
    ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
    // New:
    GetByKey(ctx context.Context, key string) (*models.Task, error)
}
```

**Additional repositories needed** (add as optional fields with setters):

```go
// ResumeNoteRepository already exists in ResumeService struct.
// Need: work session and history repos for full task context.

type ResumeWorkSessionRepository interface {
    GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error)
    GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*WorkSessionStats, error)
    GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error)
}

type ResumeHistoryRepository interface {
    GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error)
}
```

Add to `ResumeService` struct with setters:

```go
type ResumeService struct {
    // ... existing fields ...
    taskSessionRepo ResumeWorkSessionRepository // optional
    taskHistoryRepo ResumeHistoryRepository     // optional
}

func (s *ResumeService) SetTaskSessionRepo(r ResumeWorkSessionRepository) { s.taskSessionRepo = r }
func (s *ResumeService) SetTaskHistoryRepo(r ResumeHistoryRepository)     { s.taskHistoryRepo = r }
```

**New result type** in `resume_service.go` (or `resume_dto.go`):

```go
// TaskResumeContext aggregates all context needed to resume work on a task.
type TaskResumeContext struct {
    Task           *models.Task               `json:"task"`
    ContextData    *models.ContextData        `json:"context_data,omitempty"`
    Notes          []*models.EntityNote       `json:"notes,omitempty"`
    WorkSessions   []*models.WorkSession      `json:"work_sessions,omitempty"`
    SessionStats   *WorkSessionStats          `json:"session_stats,omitempty"`
    ActiveSession  *models.WorkSession        `json:"active_session,omitempty"`
    Dependencies   []string                   `json:"dependencies,omitempty"`
    CompletionMeta *models.CompletionMetadata `json:"completion_metadata,omitempty"`
}
```

**Service method**:

```go
// GetTaskResume returns comprehensive context for resuming work on a task.
//
// Aggregates: task details, parsed context data, notes, work sessions,
// session statistics, active session, dependencies, completion metadata.
// Optional fields gracefully degrade if their repositories are not wired.
func (s *ResumeService) GetTaskResume(ctx context.Context, taskKey string) (*TaskResumeContext, error) {
    task, err := s.taskRepo.GetByKey(ctx, taskKey)
    if err != nil {
        return nil, fmt.Errorf("task %s not found: %w", taskKey, err)
    }

    result := &TaskResumeContext{Task: task}

    // Parse context data from JSON column
    if task.ContextData != nil && *task.ContextData != "" && *task.ContextData != "{}" {
        if cd, err := models.FromJSON(*task.ContextData); err == nil {
            result.ContextData = cd
        }
    }

    // Get notes
    if s.noteRepo != nil {
        notes, err := s.noteRepo.GetByEntity(ctx, models.EntityTypeTask, task.ID)
        if err == nil {
            result.Notes = notes
        }
    }

    // Get work sessions
    if s.taskSessionRepo != nil {
        sessions, _ := s.taskSessionRepo.GetByTaskID(ctx, task.ID)
        result.WorkSessions = sessions

        stats, _ := s.taskSessionRepo.GetSessionStatsByTaskID(ctx, task.ID)
        result.SessionStats = stats

        activeSession, _ := s.taskSessionRepo.GetActiveSessionByTaskID(ctx, task.ID)
        result.ActiveSession = activeSession
    }

    // Parse dependencies from string column
    if task.DependsOn != nil && *task.DependsOn != "" {
        result.Dependencies = parseDependenciesJSON(*task.DependsOn)
    }

    // Build completion metadata
    completedStatus := models.TaskStatus("completed")
    reviewStatus := models.TaskStatus("ready_for_review")
    if task.Status == completedStatus || task.Status == reviewStatus {
        result.CompletionMeta = buildCompletionMeta(task)
    }

    return result, nil
}
```

**Accessor update in `services_global.go`**: Extend `GetResumeService` to call
`SetTaskSessionRepo` and `SetTaskHistoryRepo`.

---

### 3.7 TaskService.TransitionStatus and GetNextStatus (Implement Stubs)

**Purpose**: These methods return `"not implemented"` errors currently. They must be
implemented before `task_next_status.go` can be migrated.

**Current stub signatures** (already exist in `task_service.go`):

```go
func (s *TaskService) TransitionStatus(ctx context.Context, ...) error
func (s *TaskService) GetNextStatus(ctx context.Context, key string) ([]string, error)
```

Review the exact existing signatures by reading the current code. The implementation should:

- `GetNextStatus`: Use `s.workflowSvc.GetTransitionInfo(currentStatus)` to return available transitions.
- `TransitionStatus`: Validate transition via workflow service, then call
  `taskRepo.UpdateStatusForcedWithUnblock(...)`, then trigger cascade.

This is a prerequisite for the `task_next_status.go` migration.

---

## 4. Interface Summary

### 4.1 Changes to `task_service.go`

```
NEW interfaces:
  TaskHistoryRepository { GetHistoryByTaskKey }
  TaskEpicRepository    { GetByKey }
  TaskFeatureRepository { GetByKey }

EXTENDED interfaces:
  WorkSessionRepository: add GetSessionAnalyticsByFeature, GetSessionAnalyticsByEpic

NEW fields on TaskService struct:
  historyRepo  TaskHistoryRepository
  epicRepo     TaskEpicRepository
  featureRepo  TaskFeatureRepository

NEW setter methods:
  SetHistoryRepo(repo TaskHistoryRepository)
  SetEpicRepo(repo TaskEpicRepository)
  SetFeatureRepo(repo TaskFeatureRepository)

NEW service methods:
  GetTaskHistory(ctx, taskKey) ([]*models.TaskHistory, error)
  GetSessionAnalytics(ctx, SessionAnalyticsInput) (*SessionAnalyticsResult, error)
  CreateRelationship(ctx, CreateRelationshipInput) (*models.TaskRelationship, error)
  [Implement TransitionStatus stub]
  [Implement GetNextStatus stub]
```

### 4.2 Changes to `note_service.go`

```
EXTENDED interfaces:
  NoteEntityNoteRepository: add SearchWithTimePeriod

NEW interfaces:
  NoteTaskHistoryRepository { GetHistoryByTaskKey }

NEW field on NoteService struct:
  taskHistoryRepo NoteTaskHistoryRepository

NEW setter method:
  SetTaskHistoryRepo(repo NoteTaskHistoryRepository)

NEW service methods:
  SearchNotes(ctx, NoteSearchInput) ([]*NoteSearchResult, error)
  GetNoteTimeline(ctx, taskKey) ([]*TimelineEvent, error)

NEW types:
  NoteSearchInput, NoteSearchResult, TimelineEvent
```

### 4.3 Changes to `resume_service.go`

```
EXTENDED interfaces:
  ResumeTaskRepository: add GetByKey

NEW interfaces:
  ResumeWorkSessionRepository { GetByTaskID, GetSessionStatsByTaskID, GetActiveSessionByTaskID }
  ResumeHistoryRepository     { GetHistoryByTaskKey }

NEW fields on ResumeService struct:
  taskSessionRepo ResumeWorkSessionRepository
  taskHistoryRepo ResumeHistoryRepository

NEW setter methods:
  SetTaskSessionRepo(r ResumeWorkSessionRepository)
  SetTaskHistoryRepo(r ResumeHistoryRepository)

NEW service methods:
  GetTaskResume(ctx, taskKey) (*TaskResumeContext, error)

NEW types:
  TaskResumeContext
```

### 4.4 Changes to `services_global.go`

```
NEW accessor functions:
  GetTaskServiceWithHistory() *services.TaskService

EXTENDED accessor functions:
  GetTaskServiceWithDeps(): add SetEpicRepo, SetFeatureRepo calls
  GetNoteService():         add SetTaskHistoryRepo call
  GetResumeService():       add SetTaskSessionRepo, SetTaskHistoryRepo calls
```

---

## 5. CLI Migration Approach

For each file, the pattern is:

1. Replace all `cli.GetDB()` + repository construction with the appropriate global accessor.
2. Replace business logic with a single service method call.
3. Keep only: flag parsing, output formatting (JSON/table), exit codes.

### 5.1 analytics.go

**Current**: Calls `repository.NewWorkSessionRepository(db)`, `epicRepo.GetByKey()`,
`featureRepo.GetByKey()`, `sessionRepo.GetSessionAnalyticsByFeature/Epic()` directly.

**Target**:

```go
func runAnalytics(cmd *cobra.Command, args []string) error {
    // Parse
    sessionDuration, _ := cmd.Flags().GetBool("session-duration")
    pauseFrequency, _ := cmd.Flags().GetBool("pause-frequency")
    epicKey, _ := cmd.Flags().GetString("epic")
    featureKey, _ := cmd.Flags().GetString("feature")
    agentType, _ := cmd.Flags().GetString("agent-type")

    if !sessionDuration && !pauseFrequency {
        cli.Error("Please specify at least one analysis type: --session-duration or --pause-frequency")
        os.Exit(3)
    }
    if epicKey == "" && featureKey == "" {
        cli.Error("Please specify --epic or --feature for analysis scope")
        os.Exit(3)
    }

    // Call service
    svc := cli.GetTaskServiceWithDeps()
    result, err := svc.GetSessionAnalytics(cmd.Context(), services.SessionAnalyticsInput{
        EpicKey:    epicKey,
        FeatureKey: featureKey,
        AgentType:  agentType,
    })
    if err != nil {
        return err
    }

    // Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "scope": result.Scope, "analytics": result.Analytics,
            "session_duration": sessionDuration, "pause_frequency": pauseFrequency,
        })
    }
    if sessionDuration { printSessionDurationAnalytics(result.Scope, result.Analytics) }
    if pauseFrequency  { printPauseFrequencyAnalytics(result.Scope, result.Analytics)  }
    return nil
}
```

**Remove imports**: `repository`

---

### 5.2 task_history.go

**Current**: Calls `repository.NewTaskHistoryRepository(db)`, `repository.NewTaskRepository(db)`.

**Target**:

```go
func runTaskHistory(cmd *cobra.Command, args []string) error {
    taskKey := args[0]

    svc := cli.GetTaskServiceWithHistory()
    histories, err := svc.GetTaskHistory(cmd.Context(), taskKey)
    if err != nil {
        return err
    }
    // ... format output (existing outputHistoryJSON/Table/CSV preserved) ...
}
```

**Remove imports**: `repository`

---

### 5.3 task_link.go

**Current**: Calls `repository.NewTaskRepository(db)`, `repository.NewTaskRelationshipRepository(db)`,
does cycle detection and relationship creation inline.

**Target**: Loop over each relationship flag, call `svc.CreateRelationship()` per relationship.

```go
func runTaskLink(cmd *cobra.Command, args []string) error {
    taskKey := args[0]
    relationships := map[string]string{ /* ... flag map ... */ }

    hasRelationships := false
    for _, v := range relationships { if v != "" { hasRelationships = true; break } }
    if !hasRelationships {
        return fmt.Errorf("at least one relationship flag required")
    }

    svc := cli.GetTaskServiceWithDeps()
    var created []struct{ Type, TargetKey, TargetTitle string }

    for relType, targetKeysStr := range relationships {
        if targetKeysStr == "" { continue }
        for _, targetKey := range strings.Split(targetKeysStr, ",") {
            targetKey = strings.TrimSpace(targetKey)
            if targetKey == "" { continue }

            rel, err := svc.CreateRelationship(cmd.Context(), services.CreateRelationshipInput{
                FromTaskKey: taskKey,
                ToTaskKey:   targetKey,
                RelType:     models.RelationshipType(relType),
            })
            if err != nil {
                if strings.Contains(err.Error(), "already exists") {
                    cli.Warning(fmt.Sprintf("Relationship already exists: %s %s %s", taskKey, relType, targetKey))
                    continue
                }
                return err
            }
            _ = rel
            created = append(created, struct{ Type, TargetKey, TargetTitle string }{relType, targetKey, ""})
        }
    }
    // ... output ...
}
```

**Note**: `CreateRelationship` returns the relationship, which has `ToTaskID` but not `ToTaskTitle`.
To display the target title in output, `GetTaskServiceWithDeps()` needs to expose a `GetTask` method
(which already exists). Call `svc.GetTask(ctx, targetKey)` separately just for title display in
the human-readable output path.

**Remove imports**: `repository`

---

### 5.4 task_next_status.go

**Prerequisite**: `TransitionStatus()` and `GetNextStatus()` stubs must be implemented first.

**Current**: Very complex — uses `repository.NewTaskRepositoryWithWorkflow(db, workflowConfig)`,
`workflow.NewService()`, reads config, uses `performTransition()` with `UpdateStatusForcedWithUnblock`.

**Target** (after stubs implemented):

```go
func runTaskNextStatus(cmd *cobra.Command, args []string) error {
    taskKey, _ := NormalizeTaskKey(args[0])
    targetStatus, _ := cmd.Flags().GetString("status")
    preview, _ := cmd.Flags().GetBool("preview")
    force, _ := cmd.Flags().GetBool("force")
    reason, _ := cmd.Flags().GetString("reason")
    reasonDoc, _ := cmd.Flags().GetString("reason-doc")

    // Validate reason-doc path (file existence check stays in command - it's a CLI concern)
    var documentPath *string
    if reasonDoc != "" {
        if err := ValidateRejectionReasonDocPath(reasonDoc); err != nil { ... }
        // file existence check ...
        documentPath = &reasonDoc
    }

    svc := cli.GetTaskService()

    if preview {
        transitions, err := svc.GetNextStatus(cmd.Context(), taskKey)
        // ... format and return ...
    }

    if targetStatus != "" {
        result, err := svc.TransitionStatus(cmd.Context(), services.TransitionStatusInput{
            TaskKey:      taskKey,
            TargetStatus: targetStatus,
            Force:        force,
            Reason:       reason,
            DocumentPath: documentPath,
        })
        // ... format result ...
    }

    // Interactive/auto selection logic remains in command (it's a CLI concern)
    transitions, _ := svc.GetNextStatus(cmd.Context(), taskKey)
    // ... interactive selection or auto-select ...
}
```

**Complexity note**: The interactive prompt (`promptForSelection`, `printTransitions`) and
auto-select logic are pure presentation concerns and stay in the CLI command.

**Remove imports**: `repository`, `workflow`, `config` (most of config usage)

---

### 5.5 notes_search.go

**Current**: Calls `repository.NewEntityNoteRepository(db)`, `taskRepo`, `epicRepo`,
`featureRepo` directly. Constructs `NoteSearchResult` structs with entity context lookup.

**Target**:

```go
func runNotesSearch(cmd *cobra.Command, args []string) error {
    query := args[0]
    epicKey, _ := cmd.Flags().GetString("epic")
    featureKey, _ := cmd.Flags().GetString("feature")
    noteTypesStr, _ := cmd.Flags().GetString("type")
    since, _ := cmd.Flags().GetString("since")
    until, _ := cmd.Flags().GetString("until")
    entityTypeStr, _ := cmd.Flags().GetString("entity-type")

    // Parse note types and entity type (validation stays in command)
    noteTypes := parseNoteTypes(noteTypesStr)
    entityType := parseEntityType(entityTypeStr)

    svc := cli.GetNoteService(cmd.Context())
    results, err := svc.SearchNotes(cmd.Context(), services.NoteSearchInput{
        Query: query, NoteTypes: noteTypes, EntityType: entityType,
        EpicKey: epicKey, FeatureKey: featureKey, Since: since, Until: until,
    })
    if err != nil { return err }

    if cli.GlobalConfig.JSON { return cli.OutputJSON(results) }
    // ... human-readable output ...
}
```

**Remove imports**: `repository`

---

### 5.6 task_resume.go

**Current**: Creates 3 repositories inline, constructs `ResumeContext` struct, parses context
JSON, builds completion metadata — all in the handler.

**Target**:

```go
func runTaskResume(cmd *cobra.Command, args []string) error {
    taskKey := args[0]

    svc := cli.GetResumeService(cmd.Context())
    result, err := svc.GetTaskResume(cmd.Context(), taskKey)
    if err != nil {
        cli.Error(fmt.Sprintf("Task %s not found", taskKey))
        os.Exit(1)
    }

    if cli.GlobalConfig.JSON { return cli.OutputJSON(result) }
    printResumeContext(result)
    return nil
}
```

**Note**: `printResumeContext` signature changes from `*ResumeContext` (local type) to
`*services.TaskResumeContext`. Update accordingly.

**Remove imports**: `repository`, `database/sql`

---

### 5.7 task_note.go (note add, notes list, timeline)

**Current**: `runTaskNoteAdd` and `runTaskNotes` call `repository.NewEntityNoteRepository`.
`runTaskTimeline` calls `repository.NewEntityNoteRepository` and `repository.NewTaskHistoryRepository`.

**Target**:

- `runTaskNoteAdd`: Use `cli.GetNoteService()` and call `svc.AddNote(ctx, AddNoteInput{...})`.
  `AddNote` likely already exists — check existing `NoteService` for `Create` or `Add` methods.
- `runTaskNotes`: Use `svc.GetNotesByEntity(ctx, ...)` — check if this exists or needs adding.
- `runTaskTimeline`: Use `svc.GetNoteTimeline(ctx, taskKey)`.

**Note**: Review which methods already exist on `NoteService` before declaring them missing.
The existing `NoteService` may already have `AddTaskNote` or similar from prior E15 work.

**Remove imports**: `repository`

---

### 5.8 task_context.go

**Current**: Three handlers (`set`, `get`, `clear`) each call `repository.NewTaskRepository(db)`.

**Target**: Use `cli.GetTaskService()` and call appropriate context methods. Verify these
exist on `TaskService` — they likely do given the research report finding that task_context.go
was identified but no service methods are missing.

**Remove imports**: `repository`

---

### 5.9 feature_criteria.go

**Current**: Calls `featureRepo`, `taskRepo`, `criteriaRepo` directly and aggregates
criterion counts inline.

**Target**: Use `cli.GetCriteriaService(cmd.Context())` and call
`svc.GetFeatureCriteria(ctx, featureKey, byTask)`.

`GetFeatureCriteria` already exists in `CriteriaService`. Verify the return type matches
what `feature_criteria.go` needs to display.

**Remove imports**: `repository`

---

### 5.10 task_criteria.go

**Current**: Five subcommands (import, list, check, fail, na) each use `criteriaRepo` and
`taskRepo` directly. Uses `taskfile` package for import.

**Target**: Use `cli.GetCriteriaService(cmd.Context())`:
- `criteria import`: `svc.ImportCriteria(ctx, taskKey)` — already exists
- `criteria list`: `svc.ListCriteria(ctx, taskKey)` — already exists
- `criteria check`: `svc.CheckCriterion(ctx, taskKey, criterionID)` — already exists
- `criteria fail`: `svc.FailCriterion(ctx, taskKey, criterionID, reason)` — already exists
- `criteria na`: Check if `MarkNA` method exists; add if missing.

**Remove imports**: `repository`, `taskfile`

---

### 5.11 view.go

**Current**: Creates `epicRepo`, `featureRepo`, `taskRepo` and passes directly to
`view.NewService(epicRepo, featureRepo, taskRepo)`. The `view.Service` is a specialized
service in `internal/view/` — not the main service layer.

**Assessment**: `view.go` is a border case. The `view.NewService()` constructor already
accepts repository interfaces, making it architecturally clean (thin wrapper pattern is
already mostly satisfied). The main anti-pattern is direct `repository.New*` calls.

**Target approach**: Create a `GetViewService(ctx)` function in `services_global.go` (or
simply leave the repository construction in this command since `view.Service` is already
a proper service-layer wrapper). Alternatively, wrap `view.Service` construction in a
helper function to eliminate the `repository` import from the command.

This is **low priority** since the actual business logic is properly separated in `view.Service`.

**Remove imports**: `repository` (move construction to a helper)

---

### 5.12 validate.go

**Current**: Creates `epicRepo`, `featureRepo`, `taskRepo` and passes to `validation.NewRepositoryAdapter(...)`.
`validation.Validator` is an internal service. The actual validation logic is in
`internal/validation/` — architecturally clean.

**Assessment**: Same situation as `view.go` — thin wrapper already, with a proper internal
service. Low priority to refactor since the business logic is not in the command.

**Target approach**: Create `GetValidationService(ctx)` helper that returns
`validation.Validator`, or simply extract the repository construction to a helper in the command.

**Remove imports**: `repository` (move construction to a helper or accessor)

---

### 5.13 history.go (smart dispatcher)

**Current**: The smart dispatcher `history.go` may call `TaskHistoryRepository` directly.

**Target**: Use `cli.GetTaskServiceWithHistory()` and call `svc.GetTaskHistory(ctx, taskKey)`.

---

## 6. workSessionAdapter Compatibility

The `workSessionAdapter` in `services_global.go` currently bridges:
```
repository.WorkSessionRepository → services.WorkSessionRepository
```

When `WorkSessionRepository` interface is extended to add `GetSessionAnalyticsByFeature` and
`GetSessionAnalyticsByEpic`, the adapter must also implement these new methods:

```go
func (a *workSessionAdapter) GetSessionAnalyticsByFeature(ctx context.Context, featureID int64, agentType *string) (*repository.SessionAnalytics, error) {
    return a.repo.GetSessionAnalyticsByFeature(ctx, featureID, agentType)
}

func (a *workSessionAdapter) GetSessionAnalyticsByEpic(ctx context.Context, epicID int64, agentType *string) (*repository.SessionAnalytics, error) {
    return a.repo.GetSessionAnalyticsByEpic(ctx, epicID, agentType)
}
```

---

## 7. Recommended Task Sequence

The following sequence minimizes dependency conflicts:

**Phase 1: Service Method Prerequisites (must complete before CLI migration)**

1. Implement `TransitionStatus()` and `GetNextStatus()` stubs in `task_service.go`
2. Add `TaskHistoryRepository` interface and `GetTaskHistory()` to `task_service.go`
3. Extend `WorkSessionRepository` interface and add `GetSessionAnalytics()` to `task_service.go`
4. Add `CreateRelationship()` to `task_service.go`
5. Extend `NoteEntityNoteRepository` interface and add `SearchNotes()` + `GetNoteTimeline()` to `note_service.go`
6. Extend `ResumeTaskRepository` interface and add `GetTaskResume()` to `resume_service.go`
7. Update `workSessionAdapter` in `services_global.go` for new interface methods
8. Update `GetTaskServiceWithDeps()`, `GetNoteService()`, `GetResumeService()` accessors
9. Add `GetTaskServiceWithHistory()` accessor

**Phase 2: CLI Migration (can proceed once Phase 1 complete)**

10. Migrate `task_history.go` (low complexity, good first test of history accessor)
11. Migrate `task_note.go` (low complexity, tests note service methods)
12. Migrate `task_context.go` (low complexity)
13. Migrate `task_link.go` (medium complexity, tests CreateRelationship)
14. Migrate `analytics.go` (medium complexity, tests session analytics)
15. Migrate `notes_search.go` (medium complexity, tests SearchNotes)
16. Migrate `feature_criteria.go` (medium complexity, tests CriteriaService)
17. Migrate `task_criteria.go` (medium complexity, all 5 subcommands)
18. Migrate `task_resume.go` (high complexity, tests GetTaskResume)
19. Migrate `task_next_status.go` (high complexity, requires stub implementation)
20. Migrate `history.go` (smart dispatcher)
21. Migrate `view.go` and `validate.go` (low priority, architecture already sound)

**Phase 3: Tests**

22. Service unit tests for all new methods (mocked repositories)
23. CLI tests updated to use mocked services

---

## 8. Testing Requirements

### 8.1 Service Unit Tests

All new service methods require unit tests using mocked repositories:

- `TestTaskService_GetTaskHistory` — verify task not found error, empty history, populated history
- `TestTaskService_GetSessionAnalytics` — verify feature scope, epic scope, missing scope error
- `TestTaskService_CreateRelationship` — verify cycle detection, duplicate error, success cases
- `TestNoteService_SearchNotes` — verify time period path, no-time-period path, entity context resolution
- `TestNoteService_GetNoteTimeline` — verify chronological ordering, mixed event types
- `TestResumeService_GetTaskResume` — verify all optional fields gracefully degrade when repos are nil

### 8.2 Accessor Tests

- Verify `GetTaskServiceWithHistory()` returns properly wired service
- Verify `GetTaskServiceWithDeps()` extensions call the new setters

### 8.3 CLI Integration Tests

- Each migrated command should have at least one integration test verifying the service is
  called (not repository) — use `mock_task_repository.go` pattern
- Preserve all existing tests; do not break existing behavior

---

## 9. Risk Notes

1. **SessionAnalytics import cycle**: If `task_service.go` imports `repository.SessionAnalytics`,
   this may create an import cycle. Use a local `services.SessionAnalytics` struct instead and
   map fields from the repository type.

2. **TransitionStatus stub complexity**: The existing `task_next_status.go` uses
   `taskRepo.UpdateStatusForcedWithUnblock()` — a specialized repository method that handles
   auto-unblocking. The `TransitionStatus` service method must replicate this behavior without
   bypassing the auto-unblock logic.

3. **workSessionAdapter interface extension**: Extending `WorkSessionRepository` requires the
   adapter to implement new methods. If tests mock this interface directly, they must also be
   updated.

4. **ResumeTaskRepository.GetByKey**: Adding `GetByKey` to this interface means the concrete
   `*repository.TaskRepository` must satisfy the extended interface. Verify it has a `GetByKey`
   method (it does — this is standard).

5. **task_next_status.go cascade**: The current code calls `triggerStatusCascade(ctx, repoDb, task.FeatureID)`.
   This must remain in the service layer (not command). The `TransitionStatus()` implementation
   should include the cascade call internally.
