# Request Flow

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 5 — Visual Documentation

## CLI Request Flow

```mermaid
flowchart LR
    subgraph Input
        USER["User Input<br/>shark get E07-F01-001"]
    end

    subgraph Cobra["Cobra Framework"]
        PARSE["Parse Command<br/>+ Global Flags"]
        PRERUN["PersistentPreRunE<br/>(Init config, detect root)"]
    end

    subgraph Command["Command Handler"]
        ARGS["Parse Args<br/>(key = E07-F01-001)"]
        SVCGET["Get Service<br/>(cli.GetTaskService())"]
    end

    subgraph Service["Service Layer"]
        BIZ["Business Logic<br/>(validate, query, transform)"]
    end

    subgraph Repository["Repository"]
        SQL["SQL Query<br/>(SELECT * FROM tasks)"]
    end

    subgraph Database["SQLite"]
        DATA["Data<br/>(task record)"]
    end

    subgraph Output
        FORMAT["Format<br/>(JSON or Table)"]
        DISPLAY["Display<br/>(stdout)"]
    end

    subgraph Cleanup
        POSTRUN["PersistentPostRunE<br/>(Close DB)"]
    end

    USER --> PARSE --> PRERUN --> ARGS --> SVCGET --> BIZ --> SQL --> DATA
    DATA --> SQL --> BIZ --> FORMAT --> DISPLAY
    DISPLAY --> POSTRUN
```

## Data Flow: Status Transition

```mermaid
flowchart TD
    subgraph Input["User Input"]
        CMD["shark status advance E07-F01-001"]
    end

    subgraph Read["Read Phase"]
        R1["Get task from DB"]
        R2["Get workflow config"]
        R3["Get dependencies"]
    end

    subgraph Validate["Validation Phase"]
        V1["Validate transition"]
        V2["Check dependencies"]
        V3["Check backward rules"]
    end

    subgraph Write["Write Phase"]
        W1["Update task status"]
        W2["Create history record"]
        W3["Check orchestrator actions"]
    end

    subgraph Output["Output"]
        O1["TransitionResult"]
    end

    CMD --> R1 & R2 & R3
    R1 & R2 --> V1
    R3 --> V2
    R1 --> V3
    V1 & V2 & V3 --> W1
    W1 --> W2 --> W3 --> O1
```

## Data Flow: Progress Calculation

```mermaid
flowchart LR
    subgraph Sources["Data Sources"]
        TASKS["Tasks<br/>(status per task)"]
        CONFIG["Config<br/>(status weights)"]
    end

    subgraph Calculation["Calculation"]
        WEIGHT["Apply weights<br/>(status → 0-100)"]
        AGG["Aggregate<br/>(sum / total)"]
    end

    subgraph Results["Results"]
        WP["Weighted Progress %"]
        CP["Completion Progress %"]
        WB["Work Breakdown"]
        AI["Action Items"]
        HI["Health Indicator"]
    end

    TASKS --> WEIGHT
    CONFIG --> WEIGHT
    WEIGHT --> AGG
    AGG --> WP & CP
    TASKS --> WB & AI & HI
    CONFIG --> HI
```

See also: [Sequence Diagrams](../behavioral/sequence-diagrams.md) | [Component Diagram](../structural/component-diagram.md)
=======
> Generated: 2026-03-20
> Phase: 5 — Visual Documentation

## CLI Command Request Flow

### General Flow (All Commands)

```mermaid
sequenceDiagram
    participant User
    participant Main as cmd/shark/main.go
    participant Cobra as cobra.Execute()
    participant Cmd as Command Handler
    participant Global as Global Accessors
    participant Svc as Service
    participant Repo as Repository
    participant DB as SQLite/Turso

    User->>Main: shark status advance E07-F01-001
    Main->>Cobra: RootCmd.Execute()
    Cobra->>Cobra: PersistentPreRunE<br/>(init config, templates)
    Cobra->>Cmd: RunE handler

    Cmd->>Cmd: Step 1: Parse args
    Cmd->>Global: cli.GetTaskService()

    Note over Global,DB: Lazy Init (first call only)
    Global->>Global: GetDB() via sync.Once
    Global->>DB: Open connection
    DB-->>Global: *repository.DB
    Global->>Global: GetWorkflowService()
    Global->>Global: NewTaskService(repo, entitySvc, ...)
    Global-->>Cmd: *TaskService

    Cmd->>Svc: Step 2: svc.AdvanceStatus(ctx, key)
    Svc->>Repo: repo.GetByKey(ctx, "E07-F01-001")
    Repo->>DB: SELECT * FROM tasks WHERE key = ?
    DB-->>Repo: Row data
    Repo-->>Svc: *models.Task

    Svc->>Svc: Validate transition via workflow
    Svc->>Repo: repo.UpdateStatus(ctx, id, newStatus)
    Repo->>DB: UPDATE tasks SET status = ?
    DB-->>Repo: Success
    Repo-->>Svc: nil error
    Svc-->>Cmd: *models.Task (updated)

    Cmd->>Cmd: Step 3: Format output
    alt --json flag
        Cmd->>User: JSON output
    else default
        Cmd->>User: "Advanced task E07-F01-001 to in_progress"
    end

    Cobra->>Cobra: PersistentPostRunE<br/>(close DB)
```

### Status Transition Flow (with Rejection Notes)

```mermaid
sequenceDiagram
    participant Cmd as Command Handler
    participant Entity as EntityService
    participant WF as WorkflowService
    participant Repo as Repository
    participant NoteRepo as NoteRepository

    Cmd->>Entity: SetStatus(ctx, key, newStatus, reason, force)

    Entity->>Repo: GetByKey(ctx, key)
    Repo-->>Entity: entity

    alt not force
        Entity->>WF: ValidateTransition(current, target)
        alt invalid transition
            WF-->>Entity: error
            Entity-->>Cmd: "invalid transition" error
        end
    end

    alt reason provided AND backward transition
        Entity->>NoteRepo: CreateRejectionNote(ctx, ...)
        NoteRepo-->>Entity: ok
    end

    Entity->>Repo: StatusUpdateRaw(ctx, params)
    Repo-->>Entity: unblockedKeys

    alt unblockedKeys not empty
        Entity-->>Cmd: success + unblocked entities list
    else
        Entity-->>Cmd: success
    end
```

### Entity Creation Flow

```mermaid
sequenceDiagram
    participant Cmd as Command Handler
    participant Svc as TaskService
    participant Creator as TaskCreator
    participant EpicRepo as EpicRepository
    participant FeatRepo as FeatureRepository
    participant TaskRepo as TaskRepository
    participant FS as Filesystem

    Cmd->>Svc: CreateTask(ctx, CreateTaskInput)

    Svc->>EpicRepo: GetByKey(ctx, epicKey)
    EpicRepo-->>Svc: *Epic

    Svc->>FeatRepo: GetByKey(ctx, featureKey)
    FeatRepo-->>Svc: *Feature

    Svc->>Creator: GenerateTaskKey(epic, feature)
    Creator-->>Svc: "E07-F01-003"

    Svc->>Svc: Build *models.Task
    Svc->>Svc: task.Validate()

    Svc->>TaskRepo: Create(ctx, task)
    TaskRepo-->>Svc: task.ID set

    Svc->>Creator: CreateTaskFile(task, filePath)
    Creator->>FS: Write docs/plan/.../task.md
    FS-->>Creator: ok
    Creator-->>Svc: file path

    Svc-->>Cmd: *models.Task (with ID and file path)
```

## Database Initialization Flow

```mermaid
sequenceDiagram
    participant CLI as CLI Global
    participant PRR as PathResolver
    participant Config as Config Loader
    participant DB as db.InitDB()
    participant SQLite as SQLite

    CLI->>PRR: FindProjectRoot()
    PRR->>PRR: Walk up dirs<br/>.sharkconfig.json > shark-tasks.db > .git/
    PRR-->>CLI: projectRoot

    CLI->>Config: LoadConfig(projectRoot)
    Config-->>CLI: backend, url, authTokenFile

    alt backend == "turso"
        CLI->>DB: InitTursoDB(url, token)
    else backend == "local" (default)
        CLI->>DB: InitDB(projectRoot + "/shark-tasks.db")
    end

    DB->>SQLite: sql.Open("sqlite3", path + "?_foreign_keys=on")
    DB->>SQLite: PRAGMA journal_mode = WAL
    DB->>SQLite: PRAGMA cache_size = -64000
    DB->>SQLite: PRAGMA mmap_size = 30000000000

    DB->>DB: ApplySchemaIfNeeded()
    alt schema version current
        DB->>DB: Skip DDL (fast path)
    else schema needs migration
        DB->>SQLite: CREATE TABLE IF NOT EXISTS ...
        DB->>SQLite: Run migrations
        DB->>SQLite: Update schema_version
    end

    DB-->>CLI: *sql.DB (wrapped in *repository.DB)
```

---

See also: [Component Diagram](../structural/component-diagram.md) | [Architecture Overview](../../architecture/system-overview.md)
>>>>>>> Stashed changes
