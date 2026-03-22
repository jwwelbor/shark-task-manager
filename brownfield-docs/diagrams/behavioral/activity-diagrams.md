# Activity Diagrams

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 5 — Visual Documentation

## 1. Task Lifecycle (Basic Workflow)

```mermaid
stateDiagram-v2
    [*] --> todo: Create task
    todo --> in_progress: Start work
    todo --> blocked: External dependency
    in_progress --> ready_for_review: Complete work
    in_progress --> blocked: Hit blocker
    blocked --> todo: Unblock
    blocked --> in_progress: Unblock + resume
    ready_for_review --> completed: Approve
    ready_for_review --> in_progress: Request changes
    completed --> [*]
```

## 2. Task Lifecycle (Advanced Workflow — 19 Statuses)

```mermaid
stateDiagram-v2
    state "Planning Phase" as planning {
        [*] --> draft
        draft --> ready_for_refinement_ba
        ready_for_refinement_ba --> in_refinement_ba
        in_refinement_ba --> ready_for_refinement_tech
        ready_for_refinement_tech --> in_refinement_tech
        in_refinement_tech --> ready_for_development
    }

    state "Development Phase" as dev {
        ready_for_development --> in_development
    }

    state "Review Phase" as review {
        in_development --> ready_for_code_review
        ready_for_code_review --> in_code_review
        in_code_review --> changes_requested
        changes_requested --> in_development: Fix issues
        in_code_review --> ready_for_qa: Approved
    }

    state "QA Phase" as qa {
        ready_for_qa --> in_qa
        in_qa --> qa_failed
        qa_failed --> in_development: Fix bugs
        in_qa --> ready_for_approval: Tests pass
    }

    state "Approval Phase" as approval {
        ready_for_approval --> in_approval
        in_approval --> completed: Approved
        in_approval --> changes_requested: Rejected
    }

    completed --> [*]

    state "Special States" as special {
        blocked
        on_hold
        cancelled
    }

    note right of special
        Any status can transition
        to blocked, on_hold, or
        cancelled (with reason)
    end note
```

## 3. Entity Auto-Detection Flow

```mermaid
flowchart TD
    START([User runs: shark get KEY]) --> PARSE[Parse KEY format]

    PARSE --> CHECK_EPIC{Matches E## pattern?}
    CHECK_EPIC -->|Yes| EPIC[Route to Epic handler]
    CHECK_EPIC -->|No| CHECK_FEAT{Matches E##-F## or F## pattern?}

    CHECK_FEAT -->|Yes| FEAT[Route to Feature handler]
    CHECK_FEAT -->|No| CHECK_TASK{Matches E##-F##-### pattern?}

    CHECK_TASK -->|Yes| TASK[Route to Task handler]
    CHECK_TASK -->|No| CHECK_BUG{Matches B### pattern?}

    CHECK_BUG -->|Yes| BUG[Route to Bug handler]
    CHECK_BUG -->|No| CHECK_CC{Matches CC-### pattern?}

    CHECK_CC -->|Yes| CC[Route to ChangeCard handler]
    CHECK_CC -->|No| ERR[Error: unrecognized key format]

    EPIC --> SVC[Call appropriate service]
    FEAT --> SVC
    TASK --> SVC
    BUG --> SVC
    CC --> SVC
    SVC --> OUTPUT{--json flag?}
    OUTPUT -->|Yes| JSON[JSON output]
    OUTPUT -->|No| TABLE[Formatted table output]
```

## 4. Database Initialization Decision Flow

```mermaid
flowchart TD
    START([First DB access]) --> FIND[Find project root]
    FIND --> CONFIG{.sharkconfig.json exists?}

    CONFIG -->|Yes| READ[Read config]
    CONFIG -->|No| DEFAULT[Use default: local SQLite]

    READ --> BACKEND{database.backend?}
    BACKEND -->|"turso"| TURSO[Connect via libsql]
    BACKEND -->|"local" / default| LOCAL[Open shark-tasks.db]

    TURSO --> SCHEMA_CHECK
    LOCAL --> PRAGMA[Apply SQLite PRAGMAs]
    PRAGMA --> SCHEMA_CHECK

    SCHEMA_CHECK{Schema version current?}
    SCHEMA_CHECK -->|Yes| SKIP[Skip DDL — fast path]
    SCHEMA_CHECK -->|No / table missing| APPLY[Apply schema + migrations]
    APPLY --> SET_VER[Set schema_version]
    SET_VER --> READY
    SKIP --> READY([Database ready])
```

## 5. Workflow Profile Application

```mermaid
flowchart TD
    START([shark init update --workflow=advanced]) --> BACKUP[Backup .sharkconfig.json]
    BACKUP --> LOAD[Load current config]
    LOAD --> PROFILE[Load profile: advanced]

    PROFILE --> MERGE{Merge strategy}
    MERGE --> PRESERVE[Preserve: database config,<br/>viewer, project root,<br/>last sync, custom fields]
    MERGE --> REPLACE[Replace: status_flow,<br/>status_metadata,<br/>agent_types]

    PRESERVE --> WRITE[Write merged .sharkconfig.json]
    REPLACE --> WRITE
    WRITE --> VALIDATE[Validate config]
    VALIDATE --> DONE([Profile applied])
```

---

See also: [Sequence Diagrams](sequence-diagrams.md) | [Business Logic](../../behavior/business-logic.md)
