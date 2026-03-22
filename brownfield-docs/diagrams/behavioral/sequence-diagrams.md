# Sequence Diagrams

> Part of the Shark Task Manager Brownfield Analysis
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
