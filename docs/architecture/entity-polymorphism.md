# Entity Polymorphism Architecture (E21)

How the five entity types share infrastructure through the Entity interface and EntityService.

## Service Layer

```mermaid
graph TB
    subgraph "CLI Commands"
        BugCmd["bug.go<br/><i>bugServicer interface</i>"]
        ChangeCmd["change.go<br/><i>changeCardServicer interface</i>"]
        TaskCmd["task commands"]
        FeatureCmd["feature commands"]
        EpicCmd["epic commands"]
        StatusCmd["status_group.go<br/><i>dispatchTransition / dispatchNextStatus</i>"]
        RunCmd["run.go<br/><i>runner.EntityTransitioner</i>"]
    end

    subgraph "Entity-Specific Services"
        BugSvc["BugService"]
        CCSvc["ChangeCardService"]
        TaskSvc["TaskService"]
        FeatSvc["FeatureService"]
        EpicSvc["EpicService"]
    end

    subgraph "Shared Services (E21)"
        EntitySvc["EntityService<br/>━━━━━━━━━━━━━━━━<br/>TransitionStatus()<br/>GetNextStatus()<br/>GetNextStatusForEntity()<br/>ValidateAndNormalize()<br/>DetectBackward()<br/>ResolveActionForStatus()"]
        Registry["EntityRegistry<br/>━━━━━━━━━━━━━━━━<br/>map[EntityType]EntityRepository"]
    end

    subgraph "Cross-Cutting Services"
        NoteSvc["NoteService"]
        CtxSvc["ContextService"]
        ResumeSvc["ResumeService"]
    end

    BugCmd --> BugSvc
    ChangeCmd --> CCSvc
    TaskCmd --> TaskSvc
    FeatureCmd --> FeatSvc
    EpicCmd --> EpicSvc
    StatusCmd --> BugSvc & CCSvc & TaskSvc & FeatSvc & EpicSvc
    RunCmd -->|"services satisfy<br/>EntityTransitioner directly"| BugSvc & CCSvc & TaskSvc & FeatSvc & EpicSvc

    BugSvc -->|"composes"| EntitySvc
    CCSvc -->|"composes"| EntitySvc
    TaskSvc -->|"composes"| EntitySvc
    FeatSvc -->|"composes"| EntitySvc
    EpicSvc -->|"composes"| EntitySvc

    NoteSvc -->|"polymorphic lookup"| Registry
    CtxSvc -->|"polymorphic lookup"| Registry
    ResumeSvc -->|"polymorphic lookup"| Registry
```

## Model Layer

```mermaid
classDiagram
    class Entity {
        <<interface>>
        +GetID() int64
        +GetKey() string
        +GetTitle() string
        +GetSlug() string
        +GetEntityType() EntityType
        +GetStatus() string
        +SetStatus(string)
        +GetDescription() string
        +GetFilePath() string
        +GetContextData() *string
        +SetContextData(*string)
        +GetCreatedAt() time.Time
        +GetUpdatedAt() time.Time
        +Validate() error
    }

    class BaseEntity {
        +ID int64
        +Key string
        +Title string
        +Slug *string
        +Description *string
        +FilePath *string
        +ContextData *string
        +CreatedAt time.Time
        +UpdatedAt time.Time
    }

    class Epic {
        +Status EpicStatus
        +Priority int
        +BusinessValue int
    }

    class Feature {
        +Status FeatureStatus
        +EpicID int64
        +ProgressPct float64
        +ExecutionOrder int
    }

    class Task {
        +Status TaskStatus
        +FeatureID int64
        +AgentType string
        +Priority int
        +DependsOn string
        +ExecutionOrder int
    }

    class Bug {
        +Status BugStatus
        +Severity string
        +LinkedEntityType *string
        +LinkedEntityKey *string
    }

    class ChangeCard {
        +Status ChangeCardStatus
        +Priority int
        +EpicID *int64
        +FeatureID *int64
        +Justification *string
        +ImpactAnalysis *string
        +RollbackPlan *string
    }

    Entity <|.. Epic : implements
    Entity <|.. Feature : implements
    Entity <|.. Task : implements
    Entity <|.. Bug : implements
    Entity <|.. ChangeCard : implements
    BaseEntity <|-- Epic : embeds
    BaseEntity <|-- Feature : embeds
    BaseEntity <|-- Task : embeds
    BaseEntity <|-- Bug : embeds
    BaseEntity <|-- ChangeCard : embeds
```

## Repository Layer

```mermaid
graph TB
    subgraph "Polymorphic Interface (E21)"
        EntityRepo["EntityRepository<br/>━━━━━━━━━━━━━━━━<br/>GetByKey() → Entity<br/>GetByID() → Entity<br/>UpdateStatus()<br/>GetContextData()<br/>UpdateContextData()"]
    end

    subgraph "Adapters (E21)"
        BugAdapter["BugRepositoryAdapter"]
        CCAdapter["ChangeCardRepositoryAdapter"]
        TaskAdapter["TaskRepositoryAdapter"]
        FeatAdapter["FeatureRepositoryAdapter"]
        EpicAdapter["EpicRepositoryAdapter"]
    end

    subgraph "Typed Repositories"
        BugRepo["BugRepository<br/><i>*models.Bug</i>"]
        CCRepo["ChangeCardRepository<br/><i>*models.ChangeCard</i>"]
        TaskRepo["TaskRepository<br/><i>*models.Task</i>"]
        FeatRepo["FeatureRepository<br/><i>*models.Feature</i>"]
        EpicRepo["EpicRepository<br/><i>*models.Epic</i>"]
    end

    subgraph "Database"
        DB[(SQLite / Turso)]
    end

    BugAdapter -.->|implements| EntityRepo
    CCAdapter -.->|implements| EntityRepo
    TaskAdapter -.->|implements| EntityRepo
    FeatAdapter -.->|implements| EntityRepo
    EpicAdapter -.->|implements| EntityRepo

    BugAdapter -->|wraps| BugRepo
    CCAdapter -->|wraps| CCRepo
    TaskAdapter -->|wraps| TaskRepo
    FeatAdapter -->|wraps| FeatRepo
    EpicAdapter -->|wraps| EpicRepo

    BugRepo --> DB
    CCRepo --> DB
    TaskRepo --> DB
    FeatRepo --> DB
    EpicRepo --> DB
```

## TransitionStatus Flow

```mermaid
sequenceDiagram
    participant CLI as CLI Command
    participant Svc as BugService / ChangeCardService
    participant ES as EntityService
    participant Repo as EntityRepository (adapter)
    participant DB as Database

    CLI->>Svc: TransitionStatus(ctx, key, target, opts)
    Svc->>ES: TransitionStatus(ctx, repo, entityType, key, target, opts, features, resolveActionFn)
    ES->>Repo: GetByKey(ctx, key)
    Repo->>DB: SELECT ... WHERE key = ?
    DB-->>Repo: row
    Repo-->>ES: models.Entity (polymorphic)

    Note over ES: Idempotency check:<br/>currentStatus == targetStatus?<br/>→ return Transitioned: false

    ES->>ES: ValidateAndNormalize(current, target, force)

    alt DefaultTransitionFeatures (Epic/Feature/Task)
        ES->>ES: DetectBackward()
        ES->>ES: CreateRejectionNote() if backward
    end

    ES->>Repo: UpdateStatus(ctx, id, normalizedStatus)
    Repo->>DB: UPDATE ... SET status = ?
    ES->>ES: recordEntityHistory()

    alt ResolveOrchestratorAction (all types)
        ES->>ES: resolveActionFn(entity, targetStatus)
    end

    ES-->>Svc: *TransitionResult
    Svc-->>CLI: *TransitionResult
```

## What's Shared vs Entity-Specific

| Concern | Shared (EntityService) | Entity-Specific |
|---------|----------------------|-----------------|
| **Status transitions** | TransitionStatus, ValidateAndNormalize, DetectBackward | Feature flags: DefaultTransitionFeatures vs SimpleTransitionFeatures |
| **Next status** | GetNextStatus, GetNextStatusForEntity | Service wrappers pass entity type + resolveActionFn |
| **Notes** | NoteService via EntityRegistry | — |
| **Context data** | ContextService via EntityRegistry | — |
| **Resume** | ResumeService via EntityRegistry | — |
| **Documents** | EntityDocumentService (link/unlink/list) | SetWritableDocRepo wiring per service |
| **Orchestrator actions** | ResolveActionForStatus | makeResolveActionFn (entity-specific placeholders) |
| **CRUD** | — | Per-entity service methods (Create, Get, List, Update, Delete) |
| **Markdown generation** | — | Per-entity template building |
| **Triage** | — | BugService.TriageBug only |
| **Progress rollup** | — | FeatureService, EpicService only |
| **Dependencies** | — | TaskService only |

## Remaining Duplication (Post-E21)

~370 lines of structural duplication remain across entity services:

| Pattern | Where | Lines | Why Not Unified |
|---------|-------|-------|-----------------|
| CRUD service methods | All 5 services | ~300 | Type safety — each returns typed model (*Bug vs *ChangeCard) |
| Orchestrator placeholder callbacks | All 5 services | ~50 | Different placeholder maps per entity type |

Spec file generation (`generateMarkdown`) is per-entity by design — each entity type has different sections (bugs have "Steps to Reproduce"; change-cards have "Justification" / "Rollback Plan"). This is domain-specific content, not structural duplication.

The CRUD duplication is an acceptable trade-off: unifying it would require generics or runtime type assertions that sacrifice compile-time safety.
