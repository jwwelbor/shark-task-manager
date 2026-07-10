# Design Patterns Catalog

**Project**: Shark Task Manager
**Generated**: 2026-03-02

## Pattern Summary

| # | Pattern | Primary Location | Purpose |
|---|---------|-----------------|---------|
| 1 | Repository | `internal/repository/` | Data access encapsulation |
| 2 | Service Layer | `internal/services/` | Business logic orchestration |
| 3 | Constructor DI | Service/Repository constructors | Explicit dependencies |
| 4 | Global Service Accessor | `internal/cli/services_global.go` | Lazy service wiring for CLI |
| 5 | Cobra Command Structure | `internal/cli/root.go` | CLI organization and lifecycle |
| 6 | Database Singleton | `internal/cli/db_global.go` | Single connection, cloud-aware |
| 7 | Atomic File Writer | `internal/fileops/writer.go` | Race-safe entity file creation |
| 8 | Configuration Manager | `internal/config/manager.go` | Load/parse/merge config |
| 9 | Workflow State Machine | `internal/workflow/` | Status transition validation |
| 10 | Template Renderer | `internal/templates/renderer.go` | Entity markdown generation |
| 11 | Discovery Scanner | `internal/discovery/` | Filesystem entity discovery |
| 12 | Error Handling | All layers | Contextual error wrapping |
| 13 | Two-Level Validation | `internal/models/validation.go` | Structural vs business rules |
| 14 | Initialization Orchestrator | `internal/init/initializer.go` | Project setup sequence |

---

## 1. Repository Pattern

**Purpose**: Encapsulate data access, provide clean interface for CRUD operations.

**Files**: `internal/repository/task_repository.go`, `epic_repository.go`, `feature_repository.go`

```go
type TaskRepository struct {
    db *DB
}

func NewTaskRepository(db *DB) *TaskRepository {
    return &TaskRepository{db: db}
}

func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    query := `SELECT id, key, title, status FROM tasks WHERE key = ?`
    // ... parameterized query, handles sql.ErrNoRows
}
```

**Rules**:
- Pure CRUD — no business logic, progress calculation, or workflow checks
- Accepts `*sql.Tx` for service-managed transactions
- Returns `(*Model, error)` — never formats output
- Uses parameterized queries exclusively

---

## 2. Service Layer Pattern

**Purpose**: All business logic, validation, orchestration, and transaction management.

**Files**: `internal/services/task_service.go`, `feature_service.go`, `epic_service.go`, `note_service.go`

```go
type TaskService struct {
    repo        TaskRepository        // Interface, not concrete
    workflowSvc *workflow.Service
    creatorSvc  *taskcreation.Creator // Optional
    noteRepo    TaskNoteRepository    // Optional
}

func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to start task %s: %w", key, err)
    }
    if err := s.workflowSvc.ValidateTransition(string(task.Status), "in_progress"); err != nil {
        return nil, fmt.Errorf("cannot start task: %w", err)
    }
    // ... update status, return updated task
}
```

**Rules**:
- Accepts `context.Context` as first parameter
- Uses business-level inputs (key, not ID)
- Returns domain models, never formatted output
- Wraps errors with business context
- Owns transactions when multiple repo calls needed

---

## 3. Constructor Dependency Injection

**Purpose**: Compile-time safe, explicit dependencies without DI framework.

**Files**: All service and repository constructors

```go
func NewTaskService(
    repo TaskRepository,              // Interface
    workflowSvc *workflow.Service,    // Required
    creatorSvc *taskcreation.Creator,  // Optional (can be nil)
    noteRepo TaskNoteRepository,      // Optional (can be nil)
) *TaskService { ... }
```

**Rules**:
- No DI framework — pure Go constructors
- Services depend on interfaces, not concrete types
- Required dependencies panic on nil (fail-fast)
- Optional dependencies degrade gracefully

---

## 4. Global Service Accessor Pattern

**Purpose**: Simple service access for CLI commands with lazy initialization.

**Files**: `internal/cli/services_global.go:102-308`

```go
func GetTaskService() *services.TaskService {
    d := buildTaskServiceDeps()  // Lazy: gets DB, builds repos
    svc := services.NewTaskService(d.taskRepo, d.workflowSvc, d.creatorSvc, d.noteRepo)
    return svc
}

// In CLI command:
func runTaskStart(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), args[0], agentID)
    // ...
}
```

**Rules**:
- New service instance per call (lightweight, no shared state)
- Reuses global DB and workflow service (expensive to recreate)
- Panics on DB failure (fail-fast for CLI)

---

## 5. Cobra Command Structure

**Purpose**: CLI organization with global flags, auto-registration, and lifecycle hooks.

**Files**: `internal/cli/root.go`, `internal/cli/commands/*.go`

```go
var RootCmd = &cobra.Command{
    Use:   "shark",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return initConfig()  // Initialize before any subcommand
    },
    PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
        return CloseDB()  // Cleanup after command completes
    },
}

// Subcommands self-register via init()
func init() {
    cli.RootCmd.AddCommand(myCmd)
}
```

**Global Flags**: `--json`, `--field`, `--no-color`, `--verbose`, `--db`, `--config`

---

## 6. Database Singleton

**Purpose**: Single connection instance, thread-safe lazy init, cloud-aware backend selection.

**Files**: `internal/cli/db_global.go:10-75`

```go
var (
    globalDB   *repository.DB
    dbInitOnce sync.Once
)

func GetDB(ctx context.Context) (*repository.DB, error) {
    dbInitOnce.Do(func() {
        globalDB, dbInitErr = initDatabase(ctx)  // Reads .sharkconfig.json
    })
    return globalDB, dbInitErr
}
```

**Rules**:
- `sync.Once` ensures single initialization
- Cloud-aware: detects SQLite vs Turso from config
- Cleanup via Cobra `PersistentPostRunE` hook
- `ResetDB()` for test isolation

---

## 7. Atomic File Writer

**Purpose**: Race-safe entity file creation with consistent error handling.

**Files**: `internal/fileops/writer.go`

```go
writer := fileops.NewEntityFileWriter()
result, err := writer.WriteEntityFile(fileops.WriteOptions{
    Content:        content,
    ProjectRoot:    projectRoot,
    FilePath:       filePath,
    EntityType:     "task",
    UseAtomicWrite: true,   // O_EXCL flag prevents race conditions
    Force:          false,  // Don't overwrite existing
})
```

**Features**: Atomic writes via `O_EXCL`, auto directory creation, existing file linking, 87.1% test coverage.

---

## 8. Configuration Manager

**Purpose**: Load, parse, and merge `.sharkconfig.json` with field preservation on updates.

**Files**: `internal/config/manager.go`, `internal/init/profile_service.go`

- Parses JSON into raw map (preserves unknown fields)
- Profile application merges workflow fields while preserving database, viewer, project_root
- Automatic backup before updates

---

## 9. Workflow State Machine

**Purpose**: Configuration-driven status transition validation and metadata.

**Files**: `internal/workflow/`

- Reads status definitions from `.sharkconfig.json`
- `ValidateTransition(from, to)` — checks allowed transitions
- `GetNextStatus(current)` — determines next valid status
- `ForLevel(level)` — scopes to epic/feature/task level
- No hardcoded statuses anywhere in code

### Status ordering and selection contract

Never derive meaning from alphabetical or map-iteration order. The workflow
layer guarantees exactly two ordering/selection contracts:

- **`StatusFlow` slices are pass-first.** Every `StatusFlow` slice for a
  route-based workflow is produced by a single function,
  `uniqueSortedOutcomeTargets` (`internal/config/workflow/steps.go`), which
  orders targets by the semantic priority of their outcome key (pass, fail,
  blocked, extras). `AvailableTransitions[0]` is therefore always the
  happy-path route — a contract, not an accident. Sites that rely on it carry
  a `//shark:ordered` annotation.
- **Semantic selection goes through named selectors.** When code needs "the"
  status of a kind — the aggregation/reopen status, the status of a phase,
  the archive terminal, the done-but-not-archived sprint status — it calls a
  named selector in `internal/config/workflow/selectors.go`
  (`PrimaryAggregationStatus`, `StatusForPhase`, `ArchiveTerminalStatus`,
  `CompletedSprintStatus`, `DefaultTransition`).

Selectors apply the **designation rule**: one candidate wins trivially;
several candidates require exactly one step tagged `primary: true` in the
workflow YAML. Zero or multiple tags is a config error (`shark admin workflow
validate` rejects it for the selections it can see), and at runtime the
selectors return an `AmbiguousSelectionError` naming the candidates and the
fix — an arbitrary pick never happens. "No candidate" is a distinct error
(`NoCandidateError`) so callers can fall back or skip.

`make lint` enforces the boundary: a positional `[0]` / `[len-1]` pick on an
identifier matching `Statuses|Transitions|Targets` outside the selector file
fails CI unless the line is annotated `//shark:ordered <reason>`.

---

## 10. Template Renderer

**Purpose**: Generate entity markdown files from Go templates.

**Files**: `internal/templates/renderer.go`, `internal/templates/loader.go`

```go
renderer := templates.NewRenderer(loader)
content, err := renderer.Render(agentType, templates.TemplateData{
    Key: taskKey, Title: title, Priority: priority,
})
```

Custom functions: `join`, `quote`, `isEmpty`, `formatTime`, `formatDate`

---

## 11. Discovery Scanner

**Purpose**: Walk filesystem to find epics/features, detect conflicts with index.

**Files**: `internal/discovery/folder_scanner.go`, `types.go`, `conflict_resolver.go`

- Scans `docs/plan/` directory tree
- Matches folder names to epic/feature patterns
- Conflict strategies: index-precedence, folder-precedence, merge

---

## 12. Error Handling (Multi-Layer)

**Purpose**: Consistent error wrapping with context at each architectural layer.

```
Repository: "failed to get task: sql: no rows"
Service:    "failed to start task E07-F01-001: failed to get task: sql: no rows"
Command:    "Task not found: E07-F01-001" → exit code 1
```

- Custom types: `NotFoundError`, `ValidationError`, `ConflictError`
- Sentinel errors: `ErrTaskNotFound`, `ErrInvalidStatus`
- Exit codes: 0=success, 1=not found, 2=DB error, 3=invalid state

---

## 13. Two-Level Validation

**Purpose**: Separate structural validation (models) from business validation (services) to avoid circular imports.

**Files**: `internal/models/validation.go`

```go
// Level 1 (models): Structural only — no workflow imports
func (t *Task) Validate() error {
    if t.Title == "" { return errors.New("title cannot be empty") }
    if t.Priority < 1 || t.Priority > 10 { return errors.New("priority: 1-10") }
    return nil
}

// Level 2 (services): Business rules via workflow.Service
if err := s.workflowSvc.ValidateStatus(string(task.Status)); err != nil { ... }
```

---

## 14. Initialization Orchestrator

**Purpose**: Sequence project setup steps with structured error handling.

**Files**: `internal/init/initializer.go`, `profile_service.go`

Steps: Create database → Create folders → Create config → Copy templates

Each step returns structured `InitError` with step name and wrapped cause.

---

## Pattern Relationships

```
CLI Command ──calls──→ Global Service Accessor ──creates──→ Service
                                                              │
Service ──uses──→ Repository (via interface)                  │
Service ──uses──→ Workflow State Machine                      │
Service ──uses──→ Template Renderer → Atomic File Writer      │
                                                              │
Repository ──uses──→ Database Singleton                       │
                                                              │
All layers ──follow──→ Error Handling + Two-Level Validation
```
