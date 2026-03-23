# Component Diagram

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 5 — Visual Documentation

## System-Level Component Relationships

```mermaid
graph TB
    subgraph EntryPoints["Entry Points"]
        SHARK["shark CLI<br/>(cmd/shark)"]
        SERVER["HTTP Server<br/>(cmd/server)"]
    end

    subgraph CLILayer["CLI Layer (internal/cli)"]
        ROOT["Root Command<br/>Global Config, Lifecycle"]
        CMDS["Commands<br/>(30+ thin wrappers)"]
        DBGLOB["DB Global<br/>(Lazy Singleton)"]
        SVCGLOB["Service Accessors<br/>(GetTaskService, etc.)"]
        OUTPUT["Output Helpers<br/>(JSON, Table, Success)"]
    end

    subgraph ServiceLayer["Service Layer (internal/services)"]
        TSVC["TaskService"]
        FSVC["FeatureService"]
        ESVC["EpicService"]
        ENTSVC["EntityService<br/>(Polymorphic)"]
        FPSVC["FeatureProgressService"]
        ERSVC["EntityRelationshipService"]
        BSVC["BugService"]
        CSVC["ChangeCardService"]
        NSVC["NoteService"]
    end

    subgraph SupportSvc["Support Services"]
        WFSVC["workflow.Service<br/>(Status Transitions)"]
        CFGSVC["config.ActionService<br/>(Orchestrator Actions)"]
        CREATOR["taskcreation.Creator<br/>(Key Gen, File Create)"]
        STATCALC["status.CalculationService<br/>(Progress, Health)"]
    end

    subgraph RepoLayer["Repository Layer (internal/repository)"]
        TREPO["TaskRepository"]
        FREPO["FeatureRepository"]
        EREPO["EpicRepository"]
        BREPO["BugRepository"]
        CREPO["ChangeCardRepository"]
        NREPO["EntityNoteRepository"]
        HREPO["EntityHistoryRepository"]
    end

    subgraph DataLayer["Data Layer"]
        DB["SQLite / Turso<br/>(16 tables, v10 schema)"]
        FS["Filesystem<br/>(docs/plan/ markdown)"]
        CFG[".sharkconfig.json<br/>(Workflow Config)"]
        TMPL["shark-templates/<br/>(80+ templates)"]
    end

    SHARK --> ROOT
    ROOT --> CMDS
    ROOT --> DBGLOB
    ROOT --> SVCGLOB
    CMDS --> OUTPUT

    SERVER --> TSVC & FSVC & ESVC

    SVCGLOB --> TSVC & FSVC & ESVC & ENTSVC

    TSVC --> TREPO & WFSVC & CREATOR & ENTSVC
    FSVC --> FREPO & WFSVC & TREPO
    ESVC --> EREPO & WFSVC & FREPO & TREPO
    ENTSVC --> WFSVC & HREPO & NREPO
    FPSVC --> STATCALC & FREPO & TREPO
    BSVC --> BREPO & ENTSVC
    CSVC --> CREPO & ENTSVC

    CREATOR --> TMPL & FS
    WFSVC --> CFG
    STATCALC --> CFG
    DBGLOB --> DB

    TREPO & FREPO & EREPO & BREPO & CREPO & NREPO & HREPO --> DB
```

See also: [Package Dependencies](package-dependencies.md) | [System Overview](../architecture/system-overview.md)
