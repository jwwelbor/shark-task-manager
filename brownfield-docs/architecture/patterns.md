# Design Patterns

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 2 — Architecture Analysis

## Pattern Inventory

### 1. Repository Pattern (Structural)

**Where:** `internal/repository/` (all repository files)

**Implementation:**
- Each domain entity has a dedicated repository: `TaskRepository`, `FeatureRepository`, `EpicRepository`, `BugRepository`, `ChangeCardRepository`
- Repositories receive a `*DB` wrapper via constructor injection
- All methods accept `context.Context` as first parameter
- Methods use parameterized SQL queries (no ORM)

**Example:** `internal/repository/task_repository.go`
```go
type TaskRepository struct { db *DB }
func NewTaskRepository(db *DB) *TaskRepository
func (r *TaskRepository) GetByKey(ctx, key) (*models.Task, error)
func (r *TaskRepository) Create(ctx, task) error
func (r *TaskRepository) UpdateStatus(ctx, id, status, agent, notes) error
```

**Why:** Isolates data access from business logic; enables mock injection in tests.

---

### 2. Constructor Dependency Injection (Creational)

**Where:** All services (`internal/services/`), all repositories (`internal/repository/`)

**Implementation:**
- Pure Go constructor functions — no DI framework, no reflection
- Required dependencies as parameters; optional dependencies as nilable pointers
- Compile-time safety: can't construct without required dependencies
- Service constructors accept interfaces; repositories accept concrete `*DB`

**Example:** `internal/services/task_service.go`
```go
func NewTaskService(
    repo TaskRepository,              // Interface (required)
    entitySvc *EntityService,         // Concrete (required)
    creatorSvc *taskcreation.Creator,  // Optional (can be nil)
) *TaskService
```

**Why:** Explicit dependencies, testable via mock injection, no runtime magic.

---

### 3. Interface-at-Consumer Pattern (Structural)

**Where:** `internal/services/` — repository interfaces defined alongside services

**Implementation:**
- Repository interfaces defined in the _service_ package, not the repository package
- Concrete `*repository.XxxRepository` types implicitly satisfy these interfaces
- Each service defines only the methods it needs (Interface Segregation)

**Example:** `internal/services/task_service.go`
```go
// Defined in services package, implemented by repository.TaskRepository
type TaskRepository interface {
    Create(ctx context.Context, task *models.Task) error
    GetByKey(ctx context.Context, key string) (*models.Task, error)
    UpdateStatus(ctx context.Context, id int64, status models.TaskStatus, ...) error
    // ... only methods TaskService actually needs
}
```

**Why:** Decouples service tests from database; enables small, focused mocks.

---

### 4. Lazy Singleton with sync.Once (Creational)

**Where:** `internal/cli/db_global.go`, `internal/cli/workflow_global.go`, `internal/cli/services_global.go`

**Implementation:**
- Global variables initialized exactly once via `sync.Once`
- Thread-safe, fail-fast (panics on DB failure in CLI context)
- Reset functions provided for test isolation

**Example:** `internal/cli/db_global.go`
```go
var (
    globalDB   *repository.DB
    dbInitOnce sync.Once
    dbInitErr  error
)

func GetDB(ctx context.Context) (*repository.DB, error) {
    dbInitOnce.Do(func() {
        globalDB, dbInitErr = initializeDB(ctx)
    })
    return globalDB, dbInitErr
}
```

**Why:** Expensive resources (DB connection, config parsing) created once, reused across commands.

---

### 5. Command Pattern via Cobra (Behavioral)

**Where:** `internal/cli/commands/` — all 68 command files

**Implementation:**
- Each command is a `*cobra.Command` struct with `RunE` handler
- Commands self-register via `init()` functions
- Three-step handler pattern: parse args → call service → format output
- Side-effect imports in `cmd/shark/main.go` trigger registration

**Example:** `internal/cli/commands/task.go`
```go
var taskCreateCmd = &cobra.Command{
    Use:   "create ...",
    Short: "Create a new task",
    RunE:  runTaskCreate,
}

func init() {
    taskCmd.AddCommand(taskCreateCmd)
}
```

**Why:** Declarative CLI structure, auto-generated help, nested subcommands.

---

### 6. Adapter Pattern (Structural)

**Where:** `internal/services/entity_registry.go`

**Implementation:**
- `EntityRegistry` provides polymorphic access to all entity types
- Concrete repositories wrapped in adapters that satisfy a common `EntityRepository` interface
- Adapters: `EpicRepositoryAdapter`, `FeatureRepositoryAdapter`, `TaskRepositoryAdapter`, `BugRepositoryAdapter`, `ChangeCardRepositoryAdapter`

**Example:**
```go
type EntityRepository interface {
    GetByKey(ctx context.Context, key string) (models.Entity, error)
    UpdateStatus(ctx context.Context, ...) error
}

type EpicRepositoryAdapter struct { repo *repository.EpicRepository }
func (a *EpicRepositoryAdapter) GetByKey(ctx, key) (models.Entity, error) {
    return a.repo.GetByKey(ctx, key)  // Epic implements Entity interface
}
```

**Why:** Enables unified status transitions, context retrieval, and note operations across all entity types without type-switching in every command.

---

### 7. Configuration-Driven Behavior (Behavioral)

**Where:** `.sharkconfig.json`, `internal/config/`, `internal/workflow/`

**Implementation:**
- Workflow profiles define status flows, metadata, and agent routing in JSON
- Workflow service reads config at startup, validates transitions at runtime
- No hardcoded status values in business logic — all derived from config
- Multi-level support: separate flows for epic, feature, task, bug, change

**Example:** `.sharkconfig.json`
```json
{
  "task_workflow": {
    "status_flow": {
      "todo": ["in_progress", "blocked"],
      "in_progress": ["ready_for_review", "blocked"]
    },
    "status_metadata": {
      "todo": { "color": "gray", "phase": "planning" }
    }
  }
}
```

**Why:** Different teams can customize workflows without code changes; supports both simple (5-status) and advanced (19-status) profiles.

---

### 8. Embedded Asset Pattern (Structural)

**Where:** `embedded.go` (project root), `shark-templates/` directory

**Implementation:**
- Go `//go:embed` directive embeds the entire `shark-templates/` directory into the binary
- `embed.FS` provides filesystem-like access at runtime
- Internal packages import the root package's `EmbeddedSharkTemplates` variable
- Includes `all:` prefix to capture dotfiles and underscore-prefixed partials

**Example:** `embedded.go`
```go
//go:embed all:shark-templates
var EmbeddedSharkTemplates embed.FS
```

**Why:** Single-binary distribution without external template files; templates version-locked to binary.

---

### 9. Project Root Auto-Detection (Behavioral)

**Where:** `internal/pathresolver/pathresolver.go`

**Implementation:**
- Walks up directory tree from CWD looking for project markers
- Priority order: `.sharkconfig.json` > `shark-tasks.db` > `.git/`
- Enables running `shark` from any subdirectory

**Why:** AI agents and developers can run commands from any project subdirectory.

---

### 10. Dual Key Format (Data Pattern)

**Where:** `internal/keys/`, `internal/slug/`, `internal/repository/` (lookup methods)

**Implementation:**
- Every entity has both a numeric key (`E07-F01-001`) and a slug (`E07-F01-001-implement-auth`)
- Lookups try exact match first, then parse slug suffix
- Case-insensitive matching throughout
- Slugs auto-generated from titles on creation

**Why:** Numeric keys for machine efficiency; slugged keys for human readability in logs and outputs.

---

### 11. Graceful Degradation for Optional Dependencies (Behavioral)

**Where:** Service methods throughout `internal/services/`

**Implementation:**
- Optional dependencies (note repo, creator service) can be nil
- Methods check for nil before using optional dependencies
- Core operation succeeds even if optional feature is unavailable

**Example:**
```go
func (s *TaskService) BlockTask(ctx, key, reason) (*models.Task, error) {
    // ... block task (required operation) ...

    // Optional: create rejection note
    if s.noteRepo != nil && reason != "" {
        _ = s.noteRepo.CreateRejectionNote(ctx, ...)
    }
    return task, nil
}
```

**Why:** Services can be constructed with minimal dependencies for testing or in contexts where not all features are needed.

---

## Pattern Summary

| Pattern | Category | Count of Usages | Primary Location |
|---------|----------|-----------------|------------------|
| Repository | Structural | 9+ repositories | `internal/repository/` |
| Constructor DI | Creational | 12+ services | `internal/services/` |
| Interface-at-Consumer | Structural | 12+ interfaces | `internal/services/` |
| Lazy Singleton | Creational | 3 singletons | `internal/cli/*_global.go` |
| Command (Cobra) | Behavioral | 68 commands | `internal/cli/commands/` |
| Adapter | Structural | 5 adapters | `internal/services/entity_registry.go` |
| Config-Driven | Behavioral | System-wide | `.sharkconfig.json` + `internal/workflow/` |
| Embedded Assets | Structural | 1 embed | `embedded.go` |
| Auto-Detection | Behavioral | 1 resolver | `internal/pathresolver/` |
| Dual Key | Data | System-wide | `internal/keys/`, `internal/slug/` |
| Graceful Degradation | Behavioral | 6+ services | `internal/services/` |

---

See also: [System Overview](system-overview.md) | [Components](components.md) | [Dependencies](dependencies.md)
