# Decision Logic

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 4 — Behavior Analysis

## Status Transition Decision Tree

```mermaid
flowchart TD
    A[TransitionStatus request] --> B{Entity exists?}
    B -->|No| B1[Return NotFoundError]
    B -->|Yes| C{Target status valid?}
    C -->|No| C1[Return ValidationError]
    C -->|Yes| D{Forward or backward?}
    D -->|Forward| E{Dependencies met?}
    D -->|Backward| F{Reason provided?}
    F -->|No| F1[Return BackwardReasonError]
    F -->|Yes| G{Force flag set?}
    E -->|No| E1[Return DependencyError]
    E -->|Yes| H[Apply transition]
    G -->|No| I{Valid transition in workflow?}
    G -->|Yes| J{Force reason provided?}
    I -->|Yes| H
    I -->|No| I1[Return TransitionError]
    J -->|No| J1[Return ForceReasonError]
    J -->|Yes| H
    H --> K[Record in entity_history]
    K --> L{Orchestrator action defined?}
    L -->|Yes| M[Return TransitionResult with action]
    L -->|No| N[Return TransitionResult]
```

## Entity Key Auto-Detection

```mermaid
flowchart TD
    A[Input Key] --> B{Matches E## pattern?}
    B -->|Yes| C[Route to Epic handler]
    B -->|No| D{Matches E##-F## or F## pattern?}
    D -->|Yes| E[Route to Feature handler]
    D -->|No| F{Matches E##-F##-### or T-E##-F##-### pattern?}
    F -->|Yes| G[Route to Task handler]
    F -->|No| H{Matches B### pattern?}
    H -->|Yes| I[Route to Bug handler]
    H -->|No| J{Matches CC-### pattern?}
    J -->|Yes| K[Route to ChangeCard handler]
    J -->|No| L[Return InvalidKeyError]
```

## Dual Key Lookup Strategy

```mermaid
flowchart TD
    A[Input: E07-F01-001-implement-jwt] --> B[Parse numeric key: E07-F01-001]
    B --> C[Parse slug: implement-jwt]
    C --> D{Try exact match on key column}
    D -->|Found| E[Return entity]
    D -->|Not found| F{Key has slug suffix?}
    F -->|No| G[Return NotFoundError]
    F -->|Yes| H{Query: key=numeric AND slug=suffix}
    H -->|Found| E
    H -->|Not found| G
```

## Progress Calculation Logic

```mermaid
flowchart TD
    A[Calculate Feature Progress] --> B[Get all tasks for feature]
    B --> C[Load status_metadata from config]
    C --> D[For each task: get status weight]
    D --> E["Weighted = Σ(weight × count) / total × 100"]
    D --> F["Completion = completed_count / total × 100"]
    E --> G[Return weighted_progress, completion_progress]
    F --> G
```

## Health Status Logic

```mermaid
flowchart TD
    A[Evaluate Feature Health] --> B{Any blocked tasks?}
    B -->|Yes| C{Multiple blocked OR high-priority blocked?}
    C -->|Yes| D["CRITICAL"]
    C -->|No| E{Blocked > 3 days?}
    E -->|Yes| D
    E -->|No| F["WARNING"]
    B -->|No| G{Pending approvals > 3 days?}
    G -->|Yes| F
    G -->|No| H["HEALTHY"]
```

## Database Backend Selection

```mermaid
flowchart TD
    A[InitDB called] --> B{Read .sharkconfig.json}
    B --> C{database.backend value?}
    C -->|"turso"| D[Read auth_token_file or env var]
    D --> E[Connect via libsql-client-go]
    E --> F{skip_migrations?}
    F -->|Yes| G{Schema version current?}
    G -->|Yes| H[Skip DDL]
    G -->|No| I[Run migrations]
    F -->|No| I
    C -->|"local" or default| J[Open SQLite file]
    J --> K[Apply PRAGMAs]
    K --> I
    I --> L[DB ready]
    H --> L
```

## Workflow Profile Selection

```mermaid
flowchart TD
    A[Load Workflow Config] --> B{workflow_config path set?}
    B -->|Yes| C[Read external workflow JSON]
    B -->|No| D{status_metadata in .sharkconfig.json?}
    D -->|Yes| E[Use inline config]
    D -->|No| F[Use default basic profile]
    C --> G[Parse multi-level workflow]
    E --> G
    F --> G
    G --> H[Create workflow.Service]
    H --> I{ForLevel called?}
    I -->|Epic| J[Return epic-level service]
    I -->|Feature| K[Return feature-level service]
    I -->|Task| L[Return task-level service]
```

## Project Root Detection

```mermaid
flowchart TD
    A[Find Project Root] --> B[Start at CWD]
    B --> C{.sharkconfig.json exists?}
    C -->|Yes| D[Return this directory]
    C -->|No| E{shark-tasks.db exists?}
    E -->|Yes| D
    E -->|No| F{.git/ exists?}
    F -->|Yes| D
    F -->|No| G{At filesystem root?}
    G -->|Yes| H[Return CWD as fallback]
    G -->|No| I[Go up one directory]
    I --> C
```

See also: [Business Logic](business-logic.md) | [Workflows](workflows.md) | [Error Handling](error-handling.md)
=======
> Generated: 2026-03-20
> Phase: 4 — Behavior Analysis

## Key Decision Points

### 1. Entity Type Detection from Key

```mermaid
flowchart TD
    KEY[Input Key] --> NORM[Normalize: uppercase, trim]
    NORM --> RE_EPIC{Matches ^E\d+}
    RE_EPIC -->|Yes| EPIC[Epic]
    RE_EPIC -->|No| RE_FEAT{Matches E\d+-F\d+ or ^F\d+}
    RE_FEAT -->|Yes| FEAT[Feature]
    RE_FEAT -->|No| RE_TASK{Matches E\d+-F\d+-\d+ or ^T-}
    RE_TASK -->|Yes| TASK[Task]
    RE_TASK -->|No| RE_BUG{Matches ^B\d+}
    RE_BUG -->|Yes| BUG[Bug]
    RE_BUG -->|No| RE_CC{Matches ^CC-\d+}
    RE_CC -->|Yes| CC[ChangeCard]
    RE_CC -->|No| ERR[Error: unknown format]
```

Source: `internal/keys/`, `internal/cli/commands/get.go`

### 2. Status Transition Validation

```mermaid
flowchart TD
    REQ[Transition Request] --> FORCE{Force flag?}
    FORCE -->|Yes| SKIP_VALID[Skip validation]
    FORCE -->|No| LOOKUP[Look up status_flow config]
    LOOKUP --> VALID{Target in allowed list?}
    VALID -->|Yes| BACKWARD{Is backward transition?}
    VALID -->|No| REJECT[Error: invalid transition]
    BACKWARD -->|Yes| REASON{Reason required?}
    BACKWARD -->|No| APPLY[Apply transition]
    REASON -->|Yes, reason given| NOTE[Create rejection note]
    REASON -->|Yes, no reason| REJECT2[Error: reason required]
    REASON -->|No| APPLY
    NOTE --> APPLY
    SKIP_VALID --> APPLY
    APPLY --> UNBLOCK{Any entities to unblock?}
    UNBLOCK -->|Yes| DO_UNBLOCK[Unblock dependents]
    UNBLOCK -->|No| DONE[Return updated entity]
    DO_UNBLOCK --> DONE
```

Source: `internal/services/entity_service.go`, `internal/workflow/service.go`

### 3. Database Backend Selection

```mermaid
flowchart TD
    START[GetDB called] --> CONFIG{.sharkconfig.json exists?}
    CONFIG -->|No| LOCAL[Use local: shark-tasks.db]
    CONFIG -->|Yes| READ[Read database.backend]
    READ --> ENV{Uses env var?}
    ENV -->|Yes| RESOLVE[Resolve $SHARK_DB_BACKEND]
    ENV -->|No| VALUE[Use literal value]
    RESOLVE --> CHECK
    VALUE --> CHECK
    CHECK{backend value?}
    CHECK -->|"turso"| TURSO[Connect to Turso URL]
    CHECK -->|"local" / empty| LOCAL
    TURSO --> AUTH[Read auth token]
    AUTH --> CONNECT[Establish libsql connection]
    LOCAL --> OPEN[Open SQLite file]
```

Source: `internal/cli/db_global.go`, `internal/config/`

### 4. Workflow Profile Resolution

```mermaid
flowchart TD
    SVC[WorkflowService.ForLevel] --> LEVEL{Entity level?}
    LEVEL -->|task| TASK_WF[Load task_workflow from config]
    LEVEL -->|feature| FEAT_WF[Load feature_workflow from config]
    LEVEL -->|epic| EPIC_WF[Load epic_workflow from config]
    LEVEL -->|bug| BUG_WF[Load bug_workflow from config]
    LEVEL -->|change| CC_WF[Load change_workflow from config]

    TASK_WF --> FOUND{Config section exists?}
    FEAT_WF --> FOUND
    EPIC_WF --> FOUND
    BUG_WF --> FOUND
    CC_WF --> FOUND

    FOUND -->|Yes| USE[Use level-specific config]
    FOUND -->|No| FALLBACK[Fall back to default task workflow]
```

Source: `internal/workflow/service.go`

### 5. Feature Health Assessment

```mermaid
flowchart TD
    CALC[Calculate Health] --> BLOCKED{Any blocked tasks?}
    BLOCKED -->|Multiple| CRITICAL[Health: CRITICAL]
    BLOCKED -->|One| WARN_BLOCK[Check: priority]
    BLOCKED -->|None| APPROVALS{Approvals aging > 3 days?}
    WARN_BLOCK -->|High priority| CRITICAL
    WARN_BLOCK -->|Low priority| WARNING[Health: WARNING]
    APPROVALS -->|Yes| WARNING
    APPROVALS -->|No| HEALTHY[Health: HEALTHY]
```

Source: `internal/status/`

### 6. Configuration-Driven Behavior

The system uses `.sharkconfig.json` to drive multiple behavioral decisions:

| Config Key | Drives | Decision |
|-----------|--------|----------|
| `status_flow` | Valid transitions | Which statuses can follow which |
| `status_metadata.phase` | Progress weighting | How much a status contributes to progress |
| `status_metadata.responsibility` | Work breakdown | Who owns tasks in this status |
| `status_metadata.blocks_feature` | Action items | Whether status requires attention |
| `status_metadata.color` | Display | Terminal output coloring |
| `require_rejection_reason` | Validation | Whether backward transitions need reasons |
| `interactive_mode` | UX | Whether to prompt for confirmation |

---

See also: [Business Logic](business-logic.md) | [Workflows](workflows.md) | [Activity Diagrams](../diagrams/behavioral/activity-diagrams.md)
>>>>>>> Stashed changes
