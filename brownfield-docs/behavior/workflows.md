# Workflows

> Part of the Shark Task Manager Brownfield Analysis
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
