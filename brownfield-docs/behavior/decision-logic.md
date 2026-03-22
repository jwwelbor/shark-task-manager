# Decision Logic

> Part of the Shark Task Manager Brownfield Analysis
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
