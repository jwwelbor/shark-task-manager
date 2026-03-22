# Decision Logic

> Part of the Shark Task Manager Brownfield Analysis
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
