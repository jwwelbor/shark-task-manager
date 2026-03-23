# Workflows

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 4 — Behavior Analysis

## Task Lifecycle (Basic Profile)

```mermaid
stateDiagram-v2
    [*] --> todo: Create task
    todo --> in_progress: Start
    in_progress --> ready_for_review: Submit
    in_progress --> blocked: Block
    ready_for_review --> completed: Approve
    ready_for_review --> in_progress: Request changes
    blocked --> in_progress: Unblock
    completed --> [*]
```

## Task Lifecycle (Advanced Profile)

```mermaid
stateDiagram-v2
    [*] --> draft: Create
    draft --> ready_for_refinement_ba: Submit for BA
    ready_for_refinement_ba --> in_refinement_ba: BA starts
    in_refinement_ba --> ready_for_refinement_tech: BA complete
    ready_for_refinement_tech --> in_refinement_tech: Tech starts
    in_refinement_tech --> ready_for_development: Tech complete
    ready_for_development --> in_development: Dev starts
    in_development --> ready_for_code_review: Dev complete
    ready_for_code_review --> in_code_review: Review starts
    in_code_review --> changes_requested: Changes needed
    changes_requested --> in_development: Rework
    in_code_review --> ready_for_qa: Review passed
    ready_for_qa --> in_qa: QA starts
    in_qa --> qa_failed: Tests failed
    qa_failed --> in_development: Fix bugs
    in_qa --> ready_for_approval: QA passed
    ready_for_approval --> in_approval: PO reviews
    in_approval --> completed: Approved

    in_development --> blocked: Blocked
    blocked --> in_development: Unblock
    draft --> cancelled: Cancel
    draft --> on_hold: Hold
    on_hold --> draft: Resume
```

## Entity Creation Workflow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Service
    participant Creator
    participant Repo
    participant DB
    participant FS

    User->>CLI: shark task create E07 F01 "Title"
    CLI->>CLI: Parse args into CreateTaskInput
    CLI->>Service: CreateTask(ctx, input)
    Service->>Repo: GetByKey(ctx, epicKey)
    Repo->>DB: SELECT * FROM epics WHERE key=?
    DB-->>Repo: Epic
    Service->>Repo: GetByKey(ctx, featureKey)
    Repo->>DB: SELECT * FROM features WHERE key=?
    DB-->>Repo: Feature
    Service->>Creator: GenerateTaskKey(epic, feature)
    Creator-->>Service: T-E07-F01-003
    Service->>Creator: CreateTaskFile(task, filePath)
    Creator->>FS: Write markdown (atomic)
    FS-->>Creator: OK
    Service->>Repo: Create(ctx, task)
    Repo->>DB: INSERT INTO tasks
    DB-->>Repo: ID=42
    Service-->>CLI: *models.Task
    CLI->>User: Created task T-E07-F01-003
```

## Status Transition Workflow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Service
    participant Workflow
    participant Repo
    participant DB

    User->>CLI: shark status advance E07-F01-001
    CLI->>Service: TransitionStatus(ctx, key, opts)
    Service->>Repo: GetByKey(ctx, key)
    Repo->>DB: SELECT * FROM tasks
    DB-->>Repo: Task{status: "in_progress"}
    Service->>Workflow: GetNextStatus("in_progress")
    Workflow-->>Service: "ready_for_review"
    Service->>Workflow: ValidateTransition("in_progress", "ready_for_review")
    Workflow-->>Service: OK
    Service->>Service: ValidateDependencies(ctx, key, "ready_for_review")
    Service->>Repo: UpdateStatus(ctx, id, "ready_for_review")
    Repo->>DB: UPDATE tasks SET status=?
    DB-->>Repo: OK
    Note over DB: Trigger: INSERT INTO entity_history
    Service-->>CLI: TransitionResult{from: "in_progress", to: "ready_for_review"}
    CLI->>User: Advanced to ready_for_review
```

## Feature Completion Workflow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant FeatureSvc
    participant TaskRepo
    participant FeatureRepo
    participant EpicSvc

    User->>CLI: shark feature complete E07-F01
    CLI->>FeatureSvc: CompleteFeature(ctx, key, force)
    FeatureSvc->>TaskRepo: ListByFeature(ctx, featureID)
    TaskRepo-->>FeatureSvc: [Task1(completed), Task2(in_progress)]

    alt All tasks completed
        FeatureSvc->>FeatureRepo: UpdateStatus(ctx, id, "completed")
    else Tasks not completed AND force=true
        FeatureSvc->>TaskRepo: UpdateStatus(each, "completed")
        FeatureSvc->>FeatureRepo: UpdateStatus(ctx, id, "completed")
    else Tasks not completed AND force=false
        FeatureSvc-->>CLI: Error: 1 task not completed
    end

    FeatureSvc->>EpicSvc: RecalculateEpicProgress(ctx, epicID)
    FeatureSvc-->>CLI: FeatureCompleteResult
```

## Epic Cascade Workflow

```mermaid
sequenceDiagram
    participant User
    participant EpicSvc
    participant FeatureRepo
    participant TaskRepo
    participant HistoryRepo

    User->>EpicSvc: CompleteEpic(ctx, "E07", force)
    EpicSvc->>FeatureRepo: ListByEpic(ctx, epicID)
    FeatureRepo-->>EpicSvc: [F01, F02, F03]

    loop Each Feature
        EpicSvc->>TaskRepo: ListByFeature(ctx, featureID)
        TaskRepo-->>EpicSvc: [T001, T002]
        loop Each Task
            EpicSvc->>TaskRepo: UpdateStatus(taskID, "completed")
            EpicSvc->>HistoryRepo: Create(history entry)
        end
        EpicSvc->>FeatureRepo: UpdateStatus(featureID, "completed")
        EpicSvc->>HistoryRepo: Create(history entry)
    end

    EpicSvc->>EpicSvc: UpdateEpicStatus("completed")
```

## File Discovery & Sync Workflow

```mermaid
sequenceDiagram
    participant Scanner
    participant FS
    participant Parser
    participant Repo
    participant DB

    Scanner->>FS: Walk docs/plan/
    FS-->>Scanner: File list

    loop Each entity file
        Scanner->>Parser: Parse markdown frontmatter
        Parser-->>Scanner: {key, title, status, ...}
        Scanner->>Repo: GetByKey(ctx, key)

        alt Entity exists in DB
            Scanner->>Scanner: Compare file vs DB
            Note over Scanner: DB wins for status
            Scanner->>Repo: Update(ctx, entity) if content changed
        else Entity NOT in DB
            Scanner->>Repo: Create(ctx, entity)
        end
    end
```

See also: [Business Logic](business-logic.md) | [Decision Logic](decision-logic.md) | [Error Handling](error-handling.md)
