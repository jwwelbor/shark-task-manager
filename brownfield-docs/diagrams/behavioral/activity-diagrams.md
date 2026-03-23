# Activity Diagrams

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 5 — Visual Documentation

## Task Lifecycle Activity

```mermaid
flowchart TD
    START([Task Created]) --> TODO[Status: todo]
    TODO --> |"shark status advance"| IP[Status: in_progress]
    IP --> |"Work complete"| RFR[Status: ready_for_review]
    IP --> |"External blocker"| BLOCKED[Status: blocked]
    BLOCKED --> |"Blocker resolved"| IP
    RFR --> |"Approved"| DONE[Status: completed]
    RFR --> |"Changes needed"| IP
    DONE --> END([Task Complete])
```

## Feature Completion Activity

```mermaid
flowchart TD
    START([Complete Feature Request]) --> CHECK{All tasks completed?}
    CHECK --> |Yes| COMPLETE[Set feature completed]
    CHECK --> |No| FORCE{Force flag?}
    FORCE --> |Yes| CASCADE[Cascade: complete all tasks]
    CASCADE --> COMPLETE
    FORCE --> |No| ERROR[Return error: N tasks incomplete]
    COMPLETE --> RECALC[Recalculate epic progress]
    RECALC --> HISTORY[Record history entries]
    HISTORY --> END([Feature Completed])
    ERROR --> FAIL([Operation Failed])
```

## Epic Cascade Activity

```mermaid
flowchart TD
    START([Complete Epic]) --> LIST[List all features]
    LIST --> LOOP{More features?}
    LOOP --> |Yes| TASKS[List feature tasks]
    TASKS --> TLOOP{More tasks?}
    TLOOP --> |Yes| TCOMPLETE[Complete task + record history]
    TCOMPLETE --> TLOOP
    TLOOP --> |No| FCOMPLETE[Complete feature + record history]
    FCOMPLETE --> LOOP
    LOOP --> |No| ECOMPLETE[Complete epic + record history]
    ECOMPLETE --> END([Epic Completed])
```

## Project Dashboard Activity

```mermaid
flowchart TD
    START([shark status]) --> EPICS[List all epics]
    EPICS --> LOOP{More epics?}
    LOOP --> |Yes| FEATURES[Count features by status]
    FEATURES --> TASKS[Count tasks by status]
    TASKS --> PROGRESS[Calculate weighted progress]
    PROGRESS --> HEALTH[Evaluate health indicators]
    HEALTH --> IMPEDIMENTS[Identify blocked items]
    IMPEDIMENTS --> LOOP
    LOOP --> |No| RENDER[Render dashboard table]
    RENDER --> END([Display to User])
```

See also: [Sequence Diagrams](sequence-diagrams.md) | [Workflows](../behavior/workflows.md)
