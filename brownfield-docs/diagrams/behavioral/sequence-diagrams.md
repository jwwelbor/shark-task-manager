# Sequence Diagrams

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 5 — Visual Documentation

## 1. CLI Command Execution Flow

```mermaid
sequenceDiagram
    participant U as User
    participant C as Cobra Root
    participant CMD as Command Handler
    participant SVC as Service
    participant REPO as Repository
    participant DB as SQLite

    U->>C: shark <command> [args] [flags]
    C->>C: PersistentPreRunE (init config, set flags)
    C->>CMD: RunE(cmd, args)
    CMD->>CMD: Parse args/flags
    CMD->>SVC: cli.GetTaskService()
    Note over SVC: Lazy init: GetDB() + NewRepo + NewService
    CMD->>SVC: service.Method(ctx, args)
    SVC->>REPO: repo.Query(ctx, params)
    REPO->>DB: SQL query
    DB-->>REPO: Result rows
    REPO-->>SVC: Domain models
    SVC-->>CMD: Result or error
    CMD->>CMD: Format output (JSON/table)
    CMD-->>U: Formatted output
    C->>C: PersistentPostRunE (close DB)
```

## 2. Status Advance with Dependency Check

```mermaid
sequenceDiagram
    participant CMD as Command
    participant SVC as TaskService
    participant WF as workflow.Service
    participant REPO as TaskRepository
    participant HIST as HistoryRepo

    CMD->>SVC: TransitionStatus(ctx, "E07-F01-001", opts)
    SVC->>REPO: GetByKey(ctx, "E07-F01-001")
    REPO-->>SVC: Task{status: "todo"}
    SVC->>WF: GetNextStatus("todo")
    WF-->>SVC: "in_progress"
    SVC->>WF: ValidateTransition("todo", "in_progress")
    WF-->>SVC: nil (valid)
    SVC->>SVC: ValidateDependencies(ctx, key, "in_progress")
    SVC->>REPO: GetTaskDependents(ctx, key)
    REPO-->>SVC: [dep1{completed}, dep2{completed}]
    Note over SVC: All deps met
    SVC->>REPO: UpdateStatus(ctx, id, "in_progress", agent, notes)
    REPO->>REPO: SQL UPDATE + trigger creates history
    SVC->>HIST: Create(entity_history entry)
    SVC-->>CMD: TransitionResult{from: "todo", to: "in_progress"}
```

## 3. Feature Progress Calculation

```mermaid
sequenceDiagram
    participant CMD as Command
    participant FPS as FeatureProgressService
    participant CFG as Config
    participant REPO as TaskRepository

    CMD->>FPS: GetProgress(ctx, "E07-F01")
    FPS->>REPO: ListByFeature(ctx, featureID)
    REPO-->>FPS: [task1, task2, task3, task4]
    FPS->>CFG: GetStatusMetadata(each status)
    CFG-->>FPS: {progress_weight: 0|50|100, ...}

    Note over FPS: task1: completed (weight=100)<br/>task2: in_progress (weight=50)<br/>task3: todo (weight=0)<br/>task4: todo (weight=0)

    FPS->>FPS: weighted = (100+50+0+0)/4 = 37.5%
    FPS->>FPS: completion = 1/4 = 25%
    FPS-->>CMD: {weighted: 37.5, completion: 25, total: 4}
```

## 4. Database Initialization

```mermaid
sequenceDiagram
    participant CLI as GetDB()
    participant ONCE as sync.Once
    participant CFG as Config
    participant DB as db.InitDB
    participant SQL as SQLite

    CLI->>ONCE: Do(initFunc)
    Note over ONCE: Executes only once
    ONCE->>CFG: Read .sharkconfig.json
    CFG-->>ONCE: {backend: "local", url: "shark-tasks.db"}

    alt Local SQLite
        ONCE->>DB: InitDB("shark-tasks.db")
        DB->>SQL: sql.Open("sqlite3", path)
        DB->>SQL: PRAGMA foreign_keys = ON
        DB->>SQL: PRAGMA journal_mode = WAL
        DB->>SQL: PRAGMA cache_size = -64000
        DB->>DB: ApplySchemaIfNeeded()
        DB->>SQL: CREATE TABLE IF NOT EXISTS ...
    else Turso Cloud
        ONCE->>DB: InitDB(tursoURL, authToken)
        DB->>SQL: libsql.Open(url, token)
        DB->>DB: Check schema_version
    end

    DB-->>CLI: *repository.DB
```

## 5. Task Creation with File

```mermaid
sequenceDiagram
    participant CMD as Command
    participant SVC as TaskService
    participant EPIC as EpicRepo
    participant FEAT as FeatureRepo
    participant CREATOR as Creator
    participant TMPL as Templates
    participant FOPS as FileOps
    participant REPO as TaskRepo

    CMD->>SVC: CreateTask(ctx, CreateTaskInput)
    SVC->>EPIC: GetByKey(ctx, "E07")
    EPIC-->>SVC: Epic{ID: 1}
    SVC->>FEAT: GetByKey(ctx, "E07-F01")
    FEAT-->>SVC: Feature{ID: 5}
    SVC->>CREATOR: GenerateTaskKey("E07", "F01")
    CREATOR->>REPO: ListByFeature(ctx, 5)
    REPO-->>CREATOR: [existing tasks...]
    CREATOR-->>SVC: "T-E07-F01-003"
    SVC->>CREATOR: CreateTaskFile(task, path)
    CREATOR->>TMPL: Render("task_todo.tmpl", task)
    TMPL-->>CREATOR: Markdown content
    CREATOR->>FOPS: WriteEntityFile(opts)
    Note over FOPS: O_EXCL atomic write
    FOPS-->>CREATOR: WriteResult{path}
    SVC->>REPO: Create(ctx, task)
    REPO-->>SVC: task.ID = 42
    SVC-->>CMD: *models.Task
```

See also: [Activity Diagrams](activity-diagrams.md) | [Workflows](../behavior/workflows.md)
