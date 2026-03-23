# Sequence Diagrams

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
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
=======
> Generated: 2026-03-20
> Phase: 5 — Visual Documentation

## 1. Task Status Advance (Happy Path)

```mermaid
sequenceDiagram
    participant User
    participant Cmd as status advance cmd
    participant Svc as TaskService
    participant Entity as EntityService
    participant WF as WorkflowService
    participant Repo as TaskRepository
    participant DB as SQLite

    User->>Cmd: shark status advance E07-F01-001
    Cmd->>Svc: AdvanceTaskStatus(ctx, "E07-F01-001")
    Svc->>Repo: GetByKey(ctx, "E07-F01-001")
    Repo->>DB: SELECT * FROM tasks WHERE key = ?
    DB-->>Repo: task row
    Repo-->>Svc: *Task{status: "todo"}

    Svc->>Entity: AdvanceStatus(ctx, task)
    Entity->>WF: GetValidTransitions("todo")
    WF-->>Entity: ["in_progress", "blocked"]
    Entity->>WF: GetNextStatus("todo")
    WF-->>Entity: "in_progress"

    Entity->>Repo: StatusUpdateRaw(ctx, params)
    Repo->>DB: UPDATE tasks SET status='in_progress'
    DB-->>Repo: ok
    Repo-->>Entity: unblockedKeys=[]

    Entity-->>Svc: *Task{status: "in_progress"}
    Svc-->>Cmd: *Task
    Cmd-->>User: "Advanced E07-F01-001 to in_progress"
```

## 2. Task Creation Flow

```mermaid
sequenceDiagram
    participant User
    participant Cmd as task create cmd
    participant Svc as TaskService
    participant ERepo as EpicRepository
    participant FRepo as FeatureRepository
    participant TRepo as TaskRepository
    participant Creator as TaskCreator
    participant FS as Filesystem

    User->>Cmd: shark task create E07 F01 "Implement auth"
    Cmd->>Cmd: Parse args → CreateTaskInput
    Cmd->>Svc: CreateTask(ctx, input)

    Svc->>ERepo: GetByKey(ctx, "E07")
    ERepo-->>Svc: *Epic{id: 1}

    Svc->>FRepo: GetByKey(ctx, "E07-F01")
    FRepo-->>Svc: *Feature{id: 5}

    Svc->>Creator: GenerateKey(epic, feature)
    Creator-->>Svc: "T-E07-F01-003"

    Svc->>TRepo: Create(ctx, task)
    TRepo-->>Svc: task.ID = 42

    Svc->>Creator: CreateFile(task)
    Creator->>FS: Write docs/plan/E07/E07-F01/T-E07-F01-003.md
    FS-->>Creator: ok

    Svc-->>Cmd: *Task{key, id, filePath}
    Cmd-->>User: "Created task T-E07-F01-003"
```

## 3. Entity Get with Field Extraction

```mermaid
sequenceDiagram
    participant User
    participant Main as main.go
    participant Root as RootCmd
    participant Cmd as get cmd
    participant Svc as Service
    participant Repo as Repository

    User->>Main: shark get E07-F01-001 --field status
    Main->>Root: Execute()
    Root->>Root: PersistentPreRunE (init config)
    Root->>Cmd: get RunE

    Cmd->>Cmd: Detect entity type from key<br/>"E07-F01-001" → task
    Cmd->>Svc: GetTask(ctx, key)
    Svc->>Repo: GetByKey(ctx, key)
    Repo-->>Svc: *Task
    Svc-->>Cmd: *Task

    Cmd->>Cmd: --field "status"<br/>OutputField(task, "status")
    Cmd-->>User: "in_progress"
```

## 4. Backward Transition with Rejection Note

```mermaid
sequenceDiagram
    participant User
    participant Cmd as status set cmd
    participant Entity as EntityService
    participant WF as WorkflowService
    participant Repo as Repository
    participant NoteRepo as NoteRepository

    User->>Cmd: shark status set E07-F01-001 changes_requested --reason "Missing tests"
    Cmd->>Entity: SetStatus(ctx, key, "changes_requested", "Missing tests", false)

    Entity->>Repo: GetByKey(ctx, key)
    Repo-->>Entity: *Task{status: "in_code_review"}

    Entity->>WF: ValidateTransition("in_code_review", "changes_requested")
    WF-->>Entity: valid

    Note over Entity: Backward transition detected
    Entity->>NoteRepo: CreateRejectionNote(ctx, entityType, id,<br/>"in_code_review", "changes_requested",<br/>"Missing tests")
    NoteRepo-->>Entity: ok

    Entity->>Repo: StatusUpdateRaw(ctx, params)
    Repo-->>Entity: ok

    Entity-->>Cmd: *Task{status: "changes_requested"}
    Cmd-->>User: "Set E07-F01-001 to changes_requested"
```

## 5. Resume Task with Full Context

```mermaid
sequenceDiagram
    participant User
    participant Cmd as task resume cmd
    participant Resume as ResumeService
    participant TaskRepo as TaskRepository
    participant FeatRepo as FeatureRepository
    participant EpicRepo as EpicRepository
    participant NoteRepo as NoteRepository

    User->>Cmd: shark task resume E07-F01-001
    Cmd->>Resume: ResumeTask(ctx, "E07-F01-001")

    Resume->>TaskRepo: GetByKey(ctx, key)
    TaskRepo-->>Resume: *Task

    Resume->>FeatRepo: GetByID(ctx, task.FeatureID)
    FeatRepo-->>Resume: *Feature

    Resume->>EpicRepo: GetByID(ctx, feature.EpicID)
    EpicRepo-->>Resume: *Epic

    Resume->>NoteRepo: GetByEntityKey(ctx, key)
    NoteRepo-->>Resume: [notes...]

    Resume->>Resume: Aggregate context:<br/>- Task details + status<br/>- Feature context<br/>- Epic context<br/>- Recent notes<br/>- Work history

    Resume-->>Cmd: ResumeContext{task, feature, epic, notes}
    Cmd-->>User: Full context display
```

---

See also: [Request Flow](../data-flow/request-flow.md) | [Activity Diagrams](activity-diagrams.md)
>>>>>>> Stashed changes
