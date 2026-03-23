# Interfaces

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 3 — Code Reference

## Domain Interface

### Entity Interface (`internal/models/entity.go`)

All entities implement this common interface:

```go
type Entity interface {
    GetID() int64
    GetKey() string
    GetTitle() string
    GetSlug() *string
    GetDescription() string
    GetStatus() string
    SetStatus(status string)
    GetEntityType() EntityType
    GetFilePath() string
    GetContextData() map[string]interface{}
    SetContextData(data map[string]interface{})
    GetCreatedAt() time.Time
    GetUpdatedAt() time.Time
    Validate() error
}
```

**Implementors**: Epic, Feature, Task, Bug, ChangeCard

## Repository Interfaces (defined in services)

### TaskRepository

=======
> Generated: 2026-03-20
> Phase: 3 — Code Reference

## Additional Repository Types

The agents discovered several additional repositories not covered in the initial scan:

| Repository | File | Purpose |
|------------|------|---------|
| `SearchRepository` | `search_repository.go` | Cross-entity LIKE search + FTS5 full-text search |
| `TaskCriteriaRepository` | `task_criteria_repository.go` | Acceptance criteria CRUD + status tracking |
| `WorkSessionRepository` | `work_session_repository.go` | Work session tracking + analytics |

### SearchRepository Methods
- `SearchAll(ctx, query, entityType)` — Cross-entity LIKE-based search
- `Search(ctx, query, limit)` — FTS5 full-text search
- `SearchWithSnippets(ctx, query, limit)` — FTS5 with highlighted snippets
- `SearchByEpic/Feature(ctx, key, query, limit)` — Scoped search
- `RebuildIndex(ctx)` / `IndexTask(ctx, taskID)` — FTS5 index management

### WorkSessionRepository Methods
- `Create/GetByID/GetByTaskID/GetActiveSessionByTaskID` — CRUD
- `EndSession(ctx, sessionID, outcome, notes)` — End active session
- `GetSessionStatsByTaskID(ctx, taskID)` — Return `SessionStats`
- `GetSessionAnalyticsByEpic/Feature(ctx, entityID, agentType)` — Return `SessionAnalytics`

---

## Repository Interfaces (Defined in Service Layer)

### TaskRepository (`internal/services/task_service.go`)
>>>>>>> Stashed changes
```go
type TaskRepository interface {
    Create(ctx context.Context, task *models.Task) error
    GetByKey(ctx context.Context, key string) (*models.Task, error)
    GetByID(ctx context.Context, id int64) (*models.Task, error)
    Update(ctx context.Context, task *models.Task) error
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context) ([]*models.Task, error)
    ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
    ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
<<<<<<< Updated upstream
    UpdateStatus(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error
    UpdateStatusForced(ctx context.Context, id int64, status models.TaskStatus, agent, notes, reason, docPath *string, force bool) error
    GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error)
    SearchByFile(ctx context.Context, filePath string) ([]*models.Task, error)
}
```

### FeatureRepository

```go
type FeatureRepository interface {
    Create(ctx context.Context, feature *models.Feature) error
    GetByKey(ctx context.Context, key string) (*models.Feature, error)
    GetByID(ctx context.Context, id int64) (*models.Feature, error)
    Update(ctx context.Context, feature *models.Feature) error
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context) ([]*models.Feature, error)
    ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
    CalculateProgress(ctx context.Context, featureID int64) (float64, error)
}
```

### EpicRepository

```go
type EpicRepository interface {
    Create(ctx context.Context, epic *models.Epic) error
    GetByKey(ctx context.Context, key string) (*models.Epic, error)
    GetByID(ctx context.Context, id int64) (*models.Epic, error)
    Update(ctx context.Context, epic *models.Epic) error
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context) ([]*models.Epic, error)
}
```

### EntityNoteRepository

```go
type EntityNoteRepository interface {
    CreateNote(ctx context.Context, note *models.EntityNote) error
    GetByEntity(ctx context.Context, entityType string, entityID int64) ([]*models.EntityNote, error)
    GetByType(ctx context.Context, entityType string, entityID int64, noteType string) ([]*models.EntityNote, error)
}
```

### EntityHistoryRepository

```go
type EntityHistoryRepository interface {
    Create(ctx context.Context, entry *models.EntityHistory) error
    GetByEntity(ctx context.Context, entityType string, entityID int64) ([]*models.EntityHistory, error)
    GetByDateRange(ctx context.Context, entityType string, entityID int64, from, to time.Time) ([]*models.EntityHistory, error)
}
```

## Service Interfaces

### workflow.Service

```go
type Service interface {
    ForLevel(level Level) *Service
    ValidateTransition(fromStatus, toStatus string) error
    IsValidTransition(from, to string) bool
    GetNextStatus(currentStatus string) (string, error)
    GetValidTransitions(currentStatus string) []string
    GetStatusMetadata(status string) (*StatusMetadata, error)
    ValidateStatus(status string) error
    GetDefaultStatus() string
    IsTerminalStatus(status string) bool
}
```

### config.ActionService

```go
type ActionService interface {
    GetActions(status string) (*PopulatedAction, error)
    ValidateAction(action string) error
}
```

## CLI Interfaces

### Global Config

```go
type Config struct {
    JSON       bool   // --json flag
    NoColor    bool   // --no-color flag
    Verbose    bool   // --verbose flag
    ConfigFile string // --config flag
    DBPath     string // --db flag
    Field      string // --field flag
}
```

### Output Functions

```go
func OutputJSON(data interface{}) error
func OutputTable(headers []string, rows [][]string) error
func Success(message string)
func Error(message string)
func Warning(message string)
func Info(message string)
```

## HTTP API Interfaces

### ServiceContainer (`cmd/server/services.go`)

```go
type ServiceContainer struct {
    TaskService      *services.TaskService
    FeatureService   *services.FeatureService
    EpicService      *services.EpicService
    BugService       *services.BugService
    ChangeCardService *services.ChangeCardService
    NoteService      *services.NoteService
    ContextService   *services.ContextService
    ResumeService    *services.ResumeService
}

func WireServices(db *repository.DB, projectRoot string) *ServiceContainer
```

See also: [Program Structure](program-structure.md) | [Data Models](data-models.md) | [API Reference](api-reference.md)
=======
    GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error)
    UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error
    UpdateStatusForced(...) error
    UpdateStatusForcedWithUnblock(...) ([]string, error)
    StatusUpdateRaw(ctx context.Context, params models.StatusUpdateParams) ([]string, error)
    FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error)
    ListByKeyPrefix(ctx context.Context, prefix string) ([]*models.Task, error)
}
```

### EntityRepository (`internal/services/entity_registry.go`)
```go
type EntityRepository interface {
    GetByKey(ctx context.Context, key string) (models.Entity, error)
    StatusUpdateRaw(ctx context.Context, params models.StatusUpdateParams) ([]string, error)
}
```

### TaskDependencyRepository (`internal/services/task_dependency_service.go`)
```go
type TaskDependencyRepository interface {
    // Relationship CRUD and queries
}
```

## Domain Interfaces

### Entity (`internal/models/entity.go`)
```go
type Entity interface {
    GetKey() string
    GetTitle() string
    GetStatus() string
    GetEntityType() EntityType
    GetID() int64
}
```

Implemented by: `Epic`, `Feature`, `Task`, `Bug`, `ChangeCard`

### EntityType (`internal/models/entity.go`)
```go
type EntityType string

const (
    EntityTypeEpic       EntityType = "epic"
    EntityTypeFeature    EntityType = "feature"
    EntityTypeTask       EntityType = "task"
    EntityTypeBug        EntityType = "bug"
    EntityTypeChangeCard EntityType = "change"
)
```

## CLI API (Core Commands)

### Auto-Detected Commands

| Command | Key Pattern | Entity |
|---------|------------|--------|
| `shark get <key>` | `E##` | Epic |
| `shark get <key>` | `E##-F##` | Feature |
| `shark get <key>` | `E##-F##-###` | Task |
| `shark get <key>` | `B###` | Bug |
| `shark get <key>` | `CC-###` | ChangeCard |

### Global Flags (All Commands)

| Flag | Type | Default | Purpose |
|------|------|---------|---------|
| `--json` | bool | false | Machine-readable JSON output |
| `--field <name>` | string | "" | Extract single field (implies --json) |
| `--no-color` | bool | false | Disable colored output |
| `--verbose` / `-v` | bool | false | Debug logging |
| `--db <path>` | string | auto | Override database path |
| `--config <path>` | string | auto | Override config path |

## Service Accessors (`internal/cli/services_global.go`)

| Accessor | Returns | Thread-Safe |
|----------|---------|------------|
| `GetDB(ctx)` | `(*repository.DB, error)` | Yes (sync.Once) |
| `GetWorkflowService()` | `*workflow.Service` | Yes (sync.Once) |
| `GetTaskService()` | `*services.TaskService` | Yes (new per call) |
| `GetFeatureService()` | `*services.FeatureService` | Yes (new per call) |
| `GetEpicService()` | `*services.EpicService` | Yes (new per call) |
| `GetNoteService()` | `*services.NoteService` | Yes (new per call) |
| `GetContextService()` | `*services.ContextService` | Yes (new per call) |
| `GetResumeService()` | `*services.ResumeService` | Yes (new per call) |
| `GetEntityRegistry()` | `*services.EntityRegistry` | Yes (sync.Once) |
| `GetEntityService()` | `*services.EntityService` | Yes (sync.Once) |

---

See also: [Program Structure](program-structure.md) | [Data Models](data-models.md) | [Patterns](../architecture/patterns.md)
>>>>>>> Stashed changes
