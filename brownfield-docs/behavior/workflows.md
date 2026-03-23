# Workflows

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
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
=======
> Generated: 2026-03-20
> Phase: 4 — Behavior Analysis

## End-to-End Workflows

### 1. Task Lifecycle (Basic Profile)

**Trigger**: Developer creates a task
**Participants**: CLI → TaskService → Repository → Database

```mermaid
sequenceDiagram
    actor Dev as Developer
    participant CLI
    participant Svc as TaskService
    participant DB

    Dev->>CLI: shark task create E07 F01 "Implement auth"
    CLI->>Svc: CreateTask(input)
    Svc->>DB: INSERT INTO tasks
    Svc-->>Dev: Created T-E07-F01-001

    Dev->>CLI: shark status advance E07-F01-001
    CLI->>Svc: AdvanceStatus → in_progress
    Svc-->>Dev: Status: in_progress

    Note over Dev: Development work...

    Dev->>CLI: shark status advance E07-F01-001
    CLI->>Svc: AdvanceStatus → ready_for_review
    Svc-->>Dev: Status: ready_for_review

    Dev->>CLI: shark task approve E07-F01-001
    CLI->>Svc: Approve → completed
    Svc-->>Dev: Status: completed
```

### 2. Task Lifecycle (Advanced Profile — TDD Workflow)

**Trigger**: Task enters the advanced pipeline
**Participants**: Multiple agent types

```mermaid
sequenceDiagram
    actor BA as Business Analyst
    actor TL as Tech Lead
    actor Dev as Developer
    actor QA as QA Engineer
    actor PO as Product Owner

    Note over BA: Planning Phase
    BA->>BA: draft → ready_for_refinement_ba
    BA->>BA: in_refinement_ba → ready_for_refinement_tech

    Note over TL: Technical Refinement
    TL->>TL: in_refinement_tech → ready_for_development

    Note over Dev: Development Phase
    Dev->>Dev: in_development → ready_for_code_review

    Note over TL: Code Review
    TL->>TL: in_code_review
    alt Approved
        TL->>TL: → ready_for_qa
    else Changes Needed
        TL->>Dev: → changes_requested
        Dev->>Dev: Fix → ready_for_code_review
    end

    Note over QA: QA Phase
    QA->>QA: in_qa
    alt Tests Pass
        QA->>QA: → ready_for_approval
    else Tests Fail
        QA->>Dev: → qa_failed
        Dev->>Dev: Fix → ready_for_code_review
    end

    Note over PO: Approval Phase
    PO->>PO: in_approval → completed
```

### 3. Entity Status Transition (Generic)

**Trigger**: Any `shark status set` or `shark status advance` command
**Participants**: CLI → EntityService → WorkflowService → Repository

1. Parse entity key and target status
2. Look up entity by key (auto-detect type from key format)
3. Validate transition via WorkflowService
4. If backward transition and reason required: create rejection note
5. Update status in repository
6. Check if any blocked entities should be unblocked
7. Return updated entity + list of unblocked keys

### 4. Project Initialization

**Trigger**: `shark init`
**Steps**:
1. Auto-detect or create project root
2. Create `.sharkconfig.json` with default or selected profile
3. Initialize SQLite database (create tables, indexes, triggers)
4. Apply migrations if upgrading
5. Create `docs/plan/` directory structure

### 5. Resume Work Context Assembly

**Trigger**: `shark task resume E07-F01-001`
**Steps**:
1. Look up task by key
2. Look up parent feature
3. Look up parent epic
4. Retrieve task notes (sorted by creation date)
5. Retrieve recent status history
6. Retrieve context fields
7. Retrieve related documents
8. Assemble and format full context display

### 6. Idea Promotion

**Trigger**: `shark idea promote <id> --epic=E07`
**Steps**:
1. Look up idea by ID
2. Validate idea status (must be `new`)
3. Based on promotion target:
   - Epic: Create new epic from idea title
   - Feature: Create feature under specified epic
   - Task: Create task under specified feature
4. Update idea: status → `converted`, record conversion metadata
5. Return created entity

---

See also: [Business Logic](business-logic.md) | [Decision Logic](decision-logic.md) | [Sequence Diagrams](../diagrams/behavioral/sequence-diagrams.md)
>>>>>>> Stashed changes
