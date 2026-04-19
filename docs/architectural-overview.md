# Shark Task Manager — Architectural Overview

How Shark facilitates AI-driven development lifecycle (DLC) workflows through workflow state machines, prompt templates, the web viewer, and integration with external AI skills and agents.

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Entity Hierarchy & Key Formats](#entity-hierarchy--key-formats)
3. [Workflow State Machines](#workflow-state-machines)
4. [Orchestrator Action System](#orchestrator-action-system)
5. [Prompt Template System](#prompt-template-system)
6. [Context & Resume System](#context--resume-system)
7. [Agent Handoff Flow](#agent-handoff-flow)
8. [Web Viewer & HTTP API](#web-viewer--http-api)
9. [External Skills & Agent Integration](#external-skills--agent-integration)
10. [Internal Service Architecture](#internal-service-architecture)
11. [End-to-End DLC Flow](#end-to-end-dlc-flow)

---

## System Overview

Shark is a **workflow orchestration layer** that sits between a human or AI orchestrator and a codebase. It manages three types of entities (epics, features, tasks), tracks which AI agent is responsible at each workflow step, generates populated prompt instructions from templates, and exposes context through a CLI and HTTP API.

```mermaid
graph TB
    subgraph "External AI Host (e.g., Claude Code)"
        ORCH[Orchestrator / PM Agent]
        SKILLS[Skills Library<br/>specification-writing, research,<br/>architecture, implementation,<br/>test-driven-development, quality]
        AGENTS[Specialized Agents<br/>business-analyst, researcher,<br/>architect, developer, qa, uat-agent]
    end

    subgraph "Shark Task Manager"
        CLI[CLI Interface<br/>shark status advance<br/>shark task resume]
        API[HTTP API<br/>/api/v1/tasks/:key]
        WF[Workflow Engine<br/>.sharkworkflow.json]
        TMPL[Template Engine<br/>shark-templates/]
        CTX[Context Store<br/>ContextData]
        DB[(SQLite / Turso)]
    end

    subgraph "Project Artifacts"
        MD[Markdown Files<br/>docs/plan/**/*.md]
        CODE[Codebase<br/>Go source, tests]
    end

    ORCH -->|advance status / get action| CLI
    ORCH -->|advance status / get action| API
    CLI --> WF
    API --> WF
    WF -->|resolve template| TMPL
    WF -->|read/write context| CTX
    CTX --> DB
    TMPL -->|populated instruction| ORCH
    ORCH -->|spawn with skills| AGENTS
    AGENTS -->|write spec, code, tests| MD
    AGENTS -->|write spec, code, tests| CODE
    AGENTS -->|set status| CLI
```

---

## Entity Hierarchy & Key Formats

Shark organizes work in three levels. All entity types share the same workflow state machine pattern.

```mermaid
graph TD
    E["Epic  E07<br/>E07-user-management"]
    F1["Feature  E07-F01<br/>E07-F01-authentication"]
    F2["Feature  E07-F02<br/>E07-F02-authorization"]
    T1["Task  E07-F01-001<br/>T-E07-F01-001-jwt-validation"]
    T2["Task  E07-F01-002"]
    T3["Task  E07-F02-001"]

    E --> F1
    E --> F2
    F1 --> T1
    F1 --> T2
    F2 --> T3
```

**Key auto-detection** — entity type is derived from the key pattern:

| Pattern | Entity | Example |
|---------|--------|---------|
| `E##` | Epic | `E07` |
| `E##-F##` | Feature | `E07-F01` |
| `E##-F##-###` | Task | `E07-F01-001` |

All keys are case-insensitive. Slugged variants (`E07-user-management`) work everywhere.

---

## Workflow State Machines

Each entity level has its own state machine defined in `.sharkworkflow-short.json`. The state machine drives which agent is invoked, with which skills, using which prompt template.

### Epic Workflow

```mermaid
stateDiagram-v2
    [*] --> draft
    draft --> ready_for_refinement

    ready_for_refinement --> in_refinement : business-analyst\n[opus + specification-writing, epic]
    in_refinement --> ready_for_research

    ready_for_research --> in_research : researcher\n[sonnet + research, brownfield-analysis]
    in_research --> ready_for_design

    ready_for_design --> in_design : architect\n[opus + architecture]
    in_design --> ready_for_decomposition

    ready_for_decomposition --> in_decomposition : product-manager\n[sonnet + task, feature]
    in_decomposition --> ready_for_feature_review

    ready_for_feature_review --> in_feature_review : tech-lead / client\n[sonnet + quality, assessment]
    in_feature_review --> active

    active --> completed

    ready_for_refinement --> blocked
    ready_for_refinement --> on_hold
    blocked --> ready_for_refinement
    on_hold --> active
```

### Feature Workflow

```mermaid
stateDiagram-v2
    [*] --> draft
    draft --> ready_for_assessment

    ready_for_assessment --> in_assessment : researcher\n[sonnet + assessment, brownfield-analysis]
    in_assessment --> ready_for_research
    in_assessment --> ready_for_specification

    ready_for_research --> in_research : researcher\n[sonnet + research]
    in_research --> ready_for_specification

    ready_for_specification --> in_specification : architect\n[opus + specification-writing, architecture]
    in_specification --> ready_for_test_planning

    ready_for_test_planning --> in_test_planning : qa\n[sonnet + quality]
    in_test_planning --> ready_for_task_generation

    ready_for_task_generation --> in_task_generation : product-manager\n[sonnet + task]
    in_task_generation --> ready_for_task_review

    ready_for_task_review --> in_task_review : tech-lead\n[sonnet + quality, assessment]
    in_task_review --> active

    active --> completed
```

### Task Workflow

```mermaid
stateDiagram-v2
    [*] --> draft
    draft --> ready_for_development

    ready_for_development --> in_development : developer\n[sonnet + implementation, tdd]
    in_development --> ready_for_code_review

    ready_for_code_review --> in_code_review : tech-lead\n[sonnet + quality]
    in_code_review --> ready_for_qa

    ready_for_qa --> in_qa : qa\n[sonnet + quality]
    in_qa --> ready_for_approval

    ready_for_approval --> in_approval : uat-agent\n[codex + quality]
    in_approval --> completed

    in_development --> blocked
    blocked --> ready_for_development
    in_development --> on_hold
    on_hold --> ready_for_development
```

---

## Orchestrator Action System

Every status in the workflow has an `orchestrator_action` defined in the workflow JSON. When an entity enters a `ready_for_*` status, the orchestrator reads the action and routes work accordingly.

### Action Types

| Action | Behavior |
|--------|----------|
| `spawn_agent` | Create a new agent session with the specified type, model, skills, and populated instruction |
| `check_or_resume` | Resume an existing in-progress agent session; spawn new if none found |
| `cascade` | Propagate status change to all child entities (e.g., epic → features) |
| `advance_status` | Auto-advance without agent intervention (draft states) |
| `pause` | Hold for manual human intervention |
| `archive` | Finalize and close the entity |
| `wait_for_triage` | Queue for human review before routing |

### Action Resolution Flow

```mermaid
sequenceDiagram
    participant Orch as Orchestrator
    participant Shark as Shark CLI/API
    participant DB as SQLite/Turso
    participant Tmpl as Template Engine
    participant Agent as AI Agent

    Orch->>Shark: shark status advance E07-F01-001
    Shark->>DB: fetch task + current status
    DB-->>Shark: task{status: "ready_for_development"}
    Shark->>DB: lookup status_metadata["ready_for_development"]
    DB-->>Shark: orchestrator_action{agent: "developer", model: "sonnet",<br/>skills: [...], template: "task_short/ready_for_development.tmpl"}
    Shark->>DB: fetch context_data, related_docs, related_tasks
    DB-->>Shark: context + relationships
    Shark->>Tmpl: populate template with placeholders
    Tmpl-->>Shark: populated instruction string
    Shark-->>Orch: PopulatedAction{action, agent_type, model, skills, instruction}
    Orch->>Agent: spawn developer agent with skills + instruction
    Agent->>Agent: execute TDD workflow
    Agent->>Shark: shark status set E07-F01-001 in_development
```

### PopulatedAction Structure

```json
{
  "action": "spawn_agent",
  "agent_type": "developer",
  "model": "claude-sonnet-4-20250514",
  "skills": ["implementation", "test-driven-development"],
  "instruction": "Develop task E07-F01-001: \"Implement JWT token validation\".\n\nCheck for existing implementation...\n\nTDD IMPLEMENTATION\n..."
}
```

---

## Prompt Template System

Shark uses a Go `text/template`-based system with Jinja2-style syntax. Templates live in `shark-templates/` organized by entity type and status.

### Template Directory Structure

```
shark-templates/
├── partials/                    ← Shared template fragments
│   ├── _resume_preamble.tmpl    ← "Previous session was interrupted, check for existing work"
│   ├── _tdd_process.tmpl        ← Standard TDD workflow steps
│   ├── _code_review_process.tmpl← Code review checklist
│   ├── _qa_process.tmpl         ← QA execution steps
│   ├── _exit_gate.tmpl          ← Completion criteria validation
│   ├── _read_section.tmpl       ← Standard document reading order
│   ├── _commands.tmpl           ← Shark CLI command hints
│   └── _client.tmpl             ← Client/stakeholder review instructions
│
├── epic_short/                  ← Epic prompts (short workflow)
│   ├── ready_for_refinement.tmpl    → BA: write epic PRD
│   ├── ready_for_research.tmpl      → Researcher: brownfield analysis
│   ├── ready_for_design.tmpl        → Architect: tech design
│   ├── ready_for_decomposition.tmpl → PM: decompose into features
│   └── ready_for_feature_review.tmpl→ Tech lead: review features
│
├── feature_short/               ← Feature prompts
│   ├── ready_for_assessment.tmpl    → Researcher: assess complexity
│   ├── ready_for_specification.tmpl → Architect: write feature spec
│   ├── ready_for_test_planning.tmpl → QA: write test plan
│   └── ready_for_task_generation.tmpl→ PM: generate tasks
│
└── task_short/                  ← Task prompts
    ├── ready_for_development.tmpl   → Developer: TDD implementation
    ├── ready_for_code_review.tmpl   → Tech lead: review code
    ├── ready_for_qa.tmpl            → QA: execute test plan
    └── ready_for_approval.tmpl      → UAT agent: acceptance criteria
```

### Template Placeholder Variables

Templates are auto-populated with entity context before being sent to an agent:

| Category | Placeholders |
|----------|-------------|
| **Entity** | `{{.id}}`, `{{.key}}`, `{{.title}}`, `{{.status}}`, `{{.file_path}}`, `{{.created_at}}`, `{{.updated_at}}` |
| **Task-specific** | `{{.task_key}}`, `{{.priority}}`, `{{.agent_type}}`, `{{.execution_order}}` |
| **Relationships** | `{{.related_docs}}` (CSV paths), `{{.related_tasks}}` (CSV keys) |
| **Feature-specific** | `{{.feature_key}}`, `{{.epic_key}}`, `{{.related_features}}` |
| **Epic-specific** | `{{.epic_key}}`, `{{.business_value}}`, `{{.related_epics}}` |
| **Resume context** | `{{.is_resume}}`, `{{.current_step}}`, `{{.completed_steps}}`, `{{.remaining_steps}}` |
| **Decisions/Blockers** | `{{.implementation_decisions}}`, `{{.open_questions}}`, `{{.blockers}}` |

### Template Rendering Flow

```mermaid
graph LR
    WF["Workflow Config\n(.sharkworkflow-short.json)"]
    META["Status Metadata\norchestrator_action.instruction_template"]
    TMPL["Template File\ntask_short/ready_for_development.tmpl"]
    PARTIALS["Partial Templates\n_resume_preamble.tmpl\n_tdd_process.tmpl"]
    PH["Placeholder Data\nTaskPlaceholdersWithRelated()"]
    OUT["Populated Instruction\n(sent to agent)"]

    WF -->|status: ready_for_development| META
    META -->|template path| TMPL
    TMPL -->|include partials| PARTIALS
    PH -->|entity + context + docs| TMPL
    TMPL --> OUT
```

---

## Context & Resume System

Shark maintains structured context data per entity so agents can resume interrupted work without losing state.

### ContextData Model

```mermaid
classDiagram
    class ContextData {
        +Progress ProgressContext
        +ImplementationDecisions map[string]string
        +OpenQuestions []string
        +Blockers []BlockerContext
        +Metadata map[string]interface{}
    }

    class ProgressContext {
        +CurrentStep string
        +CompletedSteps []string
        +RemainingSteps []string
    }

    class BlockerContext {
        +Description string
        +CreatedAt time.Time
        +ResolvedAt *time.Time
    }

    ContextData --> ProgressContext
    ContextData --> BlockerContext
```

### Resume Context Aggregation

The `ResumeService` assembles a complete picture for agents resuming work:

```mermaid
graph TB
    CMD["shark task resume E07-F01-001"]
    RS["ResumeService"]
    TR["TaskRepository"]
    NR["NoteRepository"]
    CTX["ContextData"]
    WS["WorkSessions"]

    CMD --> RS
    RS --> TR
    RS --> NR
    RS --> CTX
    RS --> WS

    TR -->|task metadata| RS
    NR -->|rejection notes, comments| RS
    CTX -->|progress, decisions, blockers| RS
    WS -->|session history, outcomes| RS

    RS -->|TaskResumeContext| CMD
```

**CLI commands for context management:**

```bash
# Read current context
shark task resume E07-F01-001

# Update progress after partial work
shark context set E07-F01-001 --field current_step --value "Step 2: writing tests"
shark context set E07-F01-001 --field completed_steps --value '["Step 1: read specs"]'

# Clear context on fresh start
shark context clear E07-F01-001
```

---

## Agent Handoff Flow

This is the core loop: each `ready_for_*` status triggers an agent, each agent completes work and sets the next status, which triggers the next agent.

```mermaid
sequenceDiagram
    participant H as Human / Tech Director
    participant PM as Product Manager Agent
    participant BA as Business Analyst Agent
    participant RE as Researcher Agent
    participant AR as Architect Agent
    participant DEV as Developer Agent
    participant QA as QA Agent
    participant UAT as UAT Agent
    participant S as Shark (CLI/API)

    H->>S: shark epic create "User Management"
    S-->>H: E07 created (status: draft)
    H->>S: shark status advance E07
    S-->>PM: spawn_agent(business-analyst, opus)<br/>instruction: "Write epic PRD for E07"

    PM->>S: shark status set E07 in_refinement
    PM->>S: [writes PRD to docs/plan/E07/E07.md]
    PM->>S: shark status set E07 ready_for_research
    S-->>RE: spawn_agent(researcher, sonnet)<br/>instruction: "Analyze codebase for E07"

    RE->>S: shark status set E07 in_research
    RE->>S: [writes research report]
    RE->>S: shark status set E07 ready_for_design
    S-->>AR: spawn_agent(architect, opus)<br/>instruction: "Write tech design for E07"

    AR->>S: [writes feature specs, architecture docs]
    AR->>S: shark status set E07 ready_for_decomposition
    S-->>PM: spawn_agent(product-manager, sonnet)<br/>instruction: "Decompose E07 into features + tasks"

    PM->>S: [creates features F01..F05, creates tasks]
    PM->>S: shark status set E07 active

    loop For each task
        S-->>DEV: spawn_agent(developer, sonnet)<br/>instruction: "TDD implementation of E07-F01-001"
        DEV->>S: shark status set E07-F01-001 in_development
        DEV->>S: [implements code + tests]
        DEV->>S: shark status set E07-F01-001 ready_for_code_review
        S-->>QA: spawn_agent(tech-lead, sonnet)<br/>instruction: "Review code for E07-F01-001"
        QA->>S: shark status set E07-F01-001 ready_for_qa
        S-->>UAT: spawn_agent(uat-agent, codex)<br/>instruction: "UAT for E07-F01-001"
        UAT->>S: shark status set E07-F01-001 completed
    end

    H->>S: shark status E07
    S-->>H: Epic E07: 12/12 tasks completed (100%)
```

---

## Web Viewer & HTTP API

The `cmd/server` package exposes an HTTP API that the web viewer and orchestrators use to inspect entity state, relationships, and resolved orchestrator actions.

### Viewer Architecture

```mermaid
graph TB
    subgraph "Web Viewer (Browser)"
        UI[Entity Dashboard<br/>status, progress, notes]
        TREE[Relationship Tree<br/>epics → features → tasks]
        ACTION[Next Action Panel<br/>agent_type, skills, instruction]
    end

    subgraph "HTTP Server (cmd/server)"
        ROUTER[Chi Router<br/>/api/v1/]
        TH[TaskHandler]
        FH[FeatureHandler]
        EH[EpicHandler]
    end

    subgraph "Viewer Services"
        VS[ViewerService<br/>entity resolution, file reading]
        DS[DisplayService<br/>resolve orchestrator action]
        RS[ResumeService<br/>aggregate context]
    end

    subgraph "Data Layer"
        TR[ViewerTaskRepository]
        FR[ViewerFeatureRepository]
        ER[ViewerEpicRepository]
        DB[(SQLite / Turso<br/>+ viewer_task_relationships view)]
    end

    UI --> ROUTER
    TREE --> ROUTER
    ACTION --> ROUTER
    ROUTER --> TH
    ROUTER --> FH
    ROUTER --> EH
    TH --> VS
    FH --> VS
    EH --> VS
    VS --> DS
    VS --> RS
    VS --> TR
    VS --> FR
    VS --> ER
    TR --> DB
    FR --> DB
    ER --> DB
```

### Key API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/epics/:key` | Epic with feature rollups, orchestrator action |
| `GET` | `/api/v1/features/:key` | Feature with task summaries, spec file content |
| `GET` | `/api/v1/tasks/:key` | Task with context, related docs, next action |
| `PATCH` | `/api/v1/tasks/:key/advance` | Advance to next status |
| `PATCH` | `/api/v1/tasks/:key/status` | Set status directly |
| `GET` | `/api/v1/tasks/:key/resume` | Full resume context for agents |

### Viewer Response for a Task

```json
{
  "key": "E07-F01-001",
  "title": "Implement JWT token validation",
  "status": "ready_for_development",
  "file_path": "docs/plan/E07/E07-F01/T-E07-F01-001.md",
  "valid_transitions": ["in_development", "blocked", "on_hold"],
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "model": "claude-sonnet-4-20250514",
    "skills": ["implementation", "test-driven-development"],
    "instruction": "Develop task E07-F01-001: \"Implement JWT token validation\".\n..."
  },
  "context_data": {
    "current_step": null,
    "completed_steps": [],
    "implementation_decisions": {},
    "blockers": []
  },
  "notes": [],
  "related_docs": ["docs/plan/E07/E07-F01/spec.md", "docs/plan/E07/E07-F01/test-plan.md"]
}
```

---

## External Skills & Agent Integration

Shark does not implement the agents — it delegates to the **host AI CLI** (e.g., Claude Code). The `orchestrator_action` tells the host CLI what agent type, model, and skills to load.

### Skills Referenced in Workflow

| Skill | Used By | Purpose |
|-------|---------|---------|
| `specification-writing` | BA, Architect | Write PRDs, feature specs, task specs |
| `epic` | BA | Epic creation and PRD workflow |
| `research` | Researcher | Codebase analysis, brownfield review |
| `brownfield-analysis` | Researcher | Legacy code assessment |
| `architecture` | Architect | System design, tech decisions |
| `implementation` | Developer | Code implementation patterns |
| `test-driven-development` | Developer | TDD workflow (red-green-refactor) |
| `quality` | Tech Lead, QA, UAT | Code review, validation, acceptance |
| `assessment` | Researcher, Tech Lead | Scope and complexity estimation |
| `task` | PM | Task decomposition and sequencing |
| `feature` | PM | Feature definition and breakdown |

### Agent Type Mapping

```mermaid
graph LR
    subgraph "Shark Workflow Statuses"
        R1[ready_for_refinement]
        R2[ready_for_research]
        R3[ready_for_design]
        R4[ready_for_decomposition]
        R5[ready_for_development]
        R6[ready_for_code_review]
        R7[ready_for_qa]
        R8[ready_for_approval]
    end

    subgraph "Host AI CLI Agent Types"
        BA[business-analyst<br/>model: opus]
        RE[researcher<br/>model: sonnet]
        AR[architect<br/>model: opus]
        PM[product-manager<br/>model: sonnet]
        DEV[developer<br/>model: sonnet]
        TL[tech-lead<br/>model: sonnet]
        QA[qa<br/>model: sonnet]
        UAT[uat-agent<br/>model: codex]
    end

    R1 --> BA
    R2 --> RE
    R3 --> AR
    R4 --> PM
    R5 --> DEV
    R6 --> TL
    R7 --> QA
    R8 --> UAT
```

### Host CLI Integration Points

Shark assumes the host AI CLI provides:

1. **Agent spawning** — ability to create agents by type with specified skills
2. **Skill loading** — named skills that configure agent behavior (e.g., `implementation` skill loads TDD patterns)
3. **Model selection** — route to `opus`, `sonnet`, `haiku`, or external models (`codex`, `o3`)
4. **Session management** — resume interrupted agent sessions (used by `check_or_resume` action)

```mermaid
graph TB
    subgraph "Shark Provides"
        WF[Workflow state machine]
        ACT[Orchestrator action resolution]
        TMPL[Populated prompt instruction]
        CTX[Entity context data]
    end

    subgraph "Host AI CLI Provides"
        SPAWN[Agent spawning infrastructure]
        SKILL[Skills library]
        MODEL[Model routing<br/>opus / sonnet / codex]
        SESSION[Session continuity<br/>check_or_resume]
    end

    WF --> ACT
    ACT --> TMPL
    CTX --> TMPL
    TMPL -->|instruction string| SPAWN
    ACT -->|agent_type| SPAWN
    ACT -->|skills list| SKILL
    ACT -->|model hint| MODEL
    ACT -->|action: check_or_resume| SESSION
```

---

## Internal Service Architecture

```mermaid
graph TB
    subgraph "Entry Points"
        CLI_CMD[CLI Commands<br/>internal/cli/commands/]
        HTTP_HDLR[HTTP Handlers<br/>cmd/server/]
    end

    subgraph "Service Layer (internal/services/)"
        TS[TaskService<br/>lifecycle, transitions]
        FS[FeatureService<br/>progress, summaries]
        ES[EpicService<br/>rollups, impediments]
        DS[DisplayService<br/>format, resolve actions]
        RS_SVC[ResumeService<br/>aggregate context]
        CS[ContextService<br/>read/write ContextData]
        WF_SVC[WorkflowService<br/>config-driven status validation]
    end

    subgraph "Repository Layer (internal/repository/)"
        TR[TaskRepository]
        FR[FeatureRepository]
        ER[EpicRepository]
        NR[EntityNoteRepository]
        CTX_R[ContextRepository]
    end

    subgraph "Supporting Packages"
        TMPL_ENG[Template Engine<br/>internal/templates/]
        ACTION[Action Package<br/>internal/config/action/]
        WF_CFG[Workflow Config<br/>.sharkworkflow-short.json]
        DB[(SQLite / Turso)]
    end

    CLI_CMD --> TS
    CLI_CMD --> FS
    CLI_CMD --> ES
    CLI_CMD --> DS
    HTTP_HDLR --> TS
    HTTP_HDLR --> FS
    HTTP_HDLR --> ES
    HTTP_HDLR --> DS

    TS --> TR
    TS --> WF_SVC
    FS --> FR
    ES --> ER
    DS --> ACTION
    DS --> TMPL_ENG
    RS_SVC --> TR
    RS_SVC --> FR
    RS_SVC --> ER
    RS_SVC --> NR
    CS --> CTX_R

    ACTION --> WF_CFG
    TMPL_ENG --> WF_CFG
    TR --> DB
    FR --> DB
    ER --> DB
    NR --> DB
    CTX_R --> DB
```

**Layering rules (enforced):**
- CLI commands are thin wrappers: parse → call service → format output. No business logic.
- Services own all business logic, workflow validation, transactions.
- Repositories contain only data access (CRUD). No progress calculation, no workflow logic.

---

## End-to-End DLC Flow

The complete development lifecycle from idea to shipped feature:

```mermaid
flowchart TD
    START([Human: shark epic create]) --> EPIC_DRAFT[Epic: draft]

    EPIC_DRAFT --> |shark status advance| BA_SPAWN[BA Agent spawned\nopus + specification-writing, epic]
    BA_SPAWN --> PRD[PRD written to E07.md\ngoals, scope, criteria, constraints]
    PRD --> RESEARCH_Q[Epic: ready_for_research]

    RESEARCH_Q --> |shark status advance| RE_SPAWN[Researcher spawned\nsonnet + research, brownfield-analysis]
    RE_SPAWN --> ANALYSIS[Codebase analysis written\ntech constraints, dependencies]
    ANALYSIS --> DESIGN_Q[Epic: ready_for_design]

    DESIGN_Q --> |shark status advance| AR_SPAWN[Architect spawned\nopus + architecture]
    AR_SPAWN --> ARCH[Architecture docs written\nfeature specs scaffolded]
    ARCH --> DECOMP_Q[Epic: ready_for_decomposition]

    DECOMP_Q --> |shark status advance| PM_SPAWN[PM Agent spawned\nsonnet + task, feature]
    PM_SPAWN --> TASKS[Features + Tasks created\nE07-F01 through E07-F05\nT-E07-F01-001 through ...N]
    TASKS --> EPIC_ACTIVE[Epic: active\nFeatures: active]

    EPIC_ACTIVE --> FEATURE_LOOP

    subgraph FEATURE_LOOP[Feature Lifecycle - repeated per feature]
        FA[Feature: ready_for_assessment] --> |researcher| FASSESS[Complexity assessed]
        FASSESS --> FS_Q[Feature: ready_for_specification]
        FS_Q --> |architect| FSPEC[Feature spec.md written\nArchitecture, patterns, scope]
        FSPEC --> FT_Q[Feature: ready_for_test_planning]
        FT_Q --> |qa| FTEST[test-plan.md written\nTest cases, edge cases, fixtures]
        FTEST --> FTG_Q[Feature: ready_for_task_generation]
        FTG_Q --> |product-manager| FTASKS[Tasks generated\nsequenced, scoped, ordered]
        FTASKS --> FACTIVE[Feature: active]
    end

    FACTIVE --> TASK_LOOP

    subgraph TASK_LOOP[Task Lifecycle - repeated per task]
        TD[Task: ready_for_development] --> |developer\nsonnet + implementation, tdd| TIMPL[TDD implementation\nfailing tests → code → refactor\nmake fmt && make lint && make test]
        TIMPL --> TCR_Q[Task: ready_for_code_review]
        TCR_Q --> |tech-lead\nsonnet + quality| TCR[Code review report\npatterns, security, quality]
        TCR --> TQA_Q[Task: ready_for_qa]
        TQA_Q --> |qa\nsonnet + quality| TQA[QA execution\ntest plan scenarios executed]
        TQA --> TUAT_Q[Task: ready_for_approval]
        TUAT_Q --> |uat-agent\ncodex + quality| TUAT[UAT red-team review\nacceptance criteria checked]
        TUAT --> TDONE[Task: completed ✓]
    end

    TDONE --> CHECK{All tasks\ncompleted?}
    CHECK -->|No| TASK_LOOP
    CHECK -->|Yes| FDONE[Feature: completed ✓]
    FDONE --> ECHECK{All features\ncompleted?}
    ECHECK -->|No| FEATURE_LOOP
    ECHECK -->|Yes| EPIC_DONE([Epic: completed ✓])
```

---

## Configuration Reference

The workflow and agent routing are fully configurable via `.sharkworkflow-short.json`:

```json
{
  "task_workflow": {
    "status_flow": {
      "ready_for_development": ["in_development", "blocked", "on_hold"],
      "in_development": ["ready_for_code_review", "blocked", "on_hold"]
    },
    "status_metadata": {
      "ready_for_development": {
        "description": "Ready for developer to implement",
        "phase": "development",
        "color": "yellow",
        "progress_weight": 0.10,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "developer",
          "model": "claude-sonnet-4-20250514",
          "skills": ["implementation", "test-driven-development"],
          "instruction_template": "task_short/ready_for_development.tmpl"
        }
      }
    }
  }
}
```

**To customize the workflow:**
1. Copy `shark-templates/.sharkworkflow-short.json` to a custom path
2. Edit statuses, transitions, agent types, skills, or templates
3. Update `workflow_config` in `.sharkconfig.json` to point at your file
4. `shark admin init` leaves custom files outside `shark-templates/` untouched

---

## DLC Flow — Slide Version

Three diagrams sized to sit side by side on one slide. Each covers one phase of the lifecycle.

### Phase 1 of 3 — Epic Planning

```mermaid
flowchart TD
    A([Epic created]) --> B[draft]
    B --> C[ready_for_refinement]
    C --> |BA · opus| D[in_refinement\nWrite PRD]
    D --> E[ready_for_research]
    E --> |Researcher · sonnet| F[in_research\nBrownfield analysis]
    F --> G[ready_for_design]
    G --> |Architect · opus| H[in_design\nTech design + feature scaffolds]
    H --> I[ready_for_decomposition]
    I --> |PM · sonnet| J[in_decomposition\nCreate features + tasks]
    J --> K[ready_for_feature_review]
    K --> |Tech Lead · sonnet| L[in_feature_review\nReview scope + sequencing]
    L --> M([Epic: active\nFeatures queued])
```

### Phase 2 of 3 — Feature Specification

```mermaid
flowchart TD
    A([Feature created]) --> B[ready_for_assessment]
    B --> |Researcher · sonnet| C[in_assessment\nComplexity + scope check]
    C --> D[ready_for_specification]
    D --> |Architect · opus| E[in_specification\nWrite spec.md\nArchitecture · patterns · scope]
    E --> F[ready_for_test_planning]
    F --> |QA · sonnet| G[in_test_planning\nWrite test-plan.md\nScenarios · edge cases]
    G --> H[ready_for_task_generation]
    H --> |PM · sonnet| I[in_task_generation\nGenerate + sequence tasks]
    I --> J[ready_for_task_review]
    J --> |Tech Lead · sonnet| K[in_task_review\nValidate tasks + order]
    K --> L([Feature: active\nTasks queued])
```

### Phase 3 of 3 — Task Execution

```mermaid
flowchart TD
    A([Task queued]) --> B[ready_for_development]
    B --> |Developer · sonnet\nimplementation · tdd| C[in_development\nRed → Green → Refactor\nmake fmt · lint · test]
    C --> D[ready_for_code_review]
    D --> |Tech Lead · sonnet\nquality| E[in_code_review\nReview report\npatterns · security · quality]
    E --> F[ready_for_qa]
    F --> |QA · sonnet\nquality| G[in_qa\nExecute test-plan.md\nscenarios · edge cases]
    G --> H[ready_for_approval]
    H --> |UAT Agent · codex\nquality| I[in_approval\nRed-team review\nacceptance criteria]
    I --> J([Task: completed ✓])
```

---

## Summary

Shark's DLC facilitation rests on four pillars:

| Pillar | Mechanism | Files |
|--------|-----------|-------|
| **Workflow** | JSON state machine with per-status metadata | `.sharkworkflow-short.json` |
| **Prompts** | Template system auto-populated with entity context | `shark-templates/**/*.tmpl` |
| **Context** | Structured ContextData with progress, decisions, blockers | `internal/models/context_data.go` |
| **Integration** | `orchestrator_action` tells host CLI which agent/model/skills to use | `internal/config/action/`, `internal/services/display_service.go` |

The host AI CLI (Claude Code or equivalent) provides the agent spawning infrastructure, skills library, and model routing. Shark provides the workflow definition, state persistence, context aggregation, and populated prompt instructions.
