# Request Flow

> Part of the Shark Task Manager Brownfield Analysis
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
