# Data Models

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 3 — Code Reference

## Entity Hierarchy

```mermaid
erDiagram
    EPIC ||--o{ FEATURE : "has many"
    FEATURE ||--o{ TASK : "has many"
    TASK ||--o{ TASK_HISTORY : "has many"
    TASK ||--o{ TASK_NOTE : "has many"
    TASK ||--o{ WORK_SESSION : "has many"
    TASK }o--o{ TASK : "depends on"
    EPIC ||--o{ ENTITY_HISTORY : "polymorphic"
    FEATURE ||--o{ ENTITY_HISTORY : "polymorphic"
    TASK ||--o{ ENTITY_HISTORY : "polymorphic"
    BUG ||--o{ ENTITY_HISTORY : "polymorphic"
    CHANGE_CARD ||--o{ ENTITY_HISTORY : "polymorphic"
    DOCUMENT }o--o{ ENTITY_DOCUMENT : "junction"
```

## Core Models

### Epic (`internal/models/epic.go`)

| Field | Type | DB Column | Constraints | Description |
|-------|------|-----------|-------------|-------------|
| ID | int64 | id | PK, AUTO | Database ID |
| Key | string | key | UNIQUE, NOT NULL | Format: `E##` (e.g., E07) |
| Title | string | title | NOT NULL | Epic name |
| Description | string | description | | Detailed description |
| Status | string | status | NOT NULL | Workflow status |
| Priority | string | priority | CHECK(high\|medium\|low) | Epic priority |
| BusinessValue | *int | business_value | | Business value score |
| FilePath | string | file_path | | Path to markdown file |
| Slug | *string | slug | | Human-readable key suffix |
| CreatedAt | time.Time | created_at | DEFAULT NOW | Creation timestamp |
| UpdatedAt | time.Time | updated_at | AUTO-UPDATE | Last modified |

### Feature (`internal/models/feature.go`)

| Field | Type | DB Column | Constraints | Description |
|-------|------|-----------|-------------|-------------|
| ID | int64 | id | PK, AUTO | Database ID |
| EpicID | int64 | epic_id | FK → epics, CASCADE | Parent epic |
| Key | string | key | UNIQUE, NOT NULL | Format: `E##-F##` |
| Title | string | title | NOT NULL | Feature name |
| Description | string | description | | Detailed description |
| Status | string | status | NOT NULL | Workflow status |
| ProgressPct | float64 | progress_pct | 0.0-100.0 | Calculated progress |
| ExecutionOrder | *int | execution_order | | Sequencing |
| FilePath | string | file_path | | Path to markdown file |
| Slug | *string | slug | | Human-readable suffix |
| StatusOverride | bool | status_override | | Manual status control |
| CreatedAt | time.Time | created_at | DEFAULT NOW | Creation timestamp |
| UpdatedAt | time.Time | updated_at | AUTO-UPDATE | Last modified |

### Task (`internal/models/task.go`)

| Field | Type | DB Column | Constraints | Description |
|-------|------|-----------|-------------|-------------|
| ID | int64 | id | PK, AUTO | Database ID |
| FeatureID | int64 | feature_id | FK → features, CASCADE | Parent feature |
| Key | string | key | UNIQUE, NOT NULL | Format: `T-E##-F##-###` |
| Title | string | title | NOT NULL | Task name |
| Description | string | description | | Detailed description |
| Status | TaskStatus | status | NOT NULL | Workflow status |
| AgentType | *string | agent_type | | Assigned agent role |
| Priority | int | priority | CHECK(1-10) | Priority level |
| DependsOn | *string | depends_on | | JSON array of task keys |
| AssignedAgent | *string | assigned_agent | | Specific agent ID |
| BlockedReason | *string | blocked_reason | | Why task is blocked |
| ExecutionOrder | *int | execution_order | | Sequencing within feature |
| FilePath | string | file_path | | Path to markdown file |
| Slug | *string | slug | | Human-readable suffix |
| StartedAt | *time.Time | started_at | | Work start timestamp |
| CompletedAt | *time.Time | completed_at | | Completion timestamp |
| BlockedAt | *time.Time | blocked_at | | Block timestamp |
| CompletedBy | *string | completed_by | | Who completed |
| CompletionNotes | *string | completion_notes | | Completion notes |
| FilesChanged | *string | files_changed | | JSON: changed files |
| TestsPassed | *bool | tests_passed | | Test result |
| VerificationStatus | *string | verification_status | CHECK(pending\|verified\|needs_rework) | QA status |
| CreatedAt | time.Time | created_at | DEFAULT NOW | Creation timestamp |
| UpdatedAt | time.Time | updated_at | AUTO-UPDATE | Last modified |

### Bug (`internal/models/bug.go`)

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Database ID |
| Key | string | Format: `B###` (e.g., B001) |
| Title | string | Bug summary |
| Description | string | Bug details |
| Status | string | Workflow status |
| Priority | int | 1-10 priority |
| Severity | string | Bug severity |
| LinkedEntityType | *string | Optional: linked entity type |
| LinkedEntityKey | *string | Optional: linked entity key |
| ContextData | map[string]interface{} | JSON context |
| FilePath | string | Markdown file path |

### ChangeCard (`internal/models/change_card.go`)

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Database ID |
| Key | string | Format: `CC-###` (e.g., CC-001) |
| Title | string | Change request summary |
| Description | string | Change details |
| Status | string | Workflow status |
| Priority | int | 1-10 priority |
| LinkedEntityType | *string | Optional: linked entity type |
| LinkedEntityKey | *string | Optional: linked entity key |
| ContextData | map[string]interface{} | JSON context |
| FilePath | string | Markdown file path |

## Supporting Models

### TaskHistory

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Record ID |
| TaskID | int64 | FK → tasks |
| OldStatus | string | Previous status |
| NewStatus | string | New status |
| Agent | *string | Who changed |
| Notes | *string | Change notes |
| Forced | bool | Used --force |
| RejectionReason | *string | Why rejected |
| Timestamp | time.Time | When changed |

### EntityNote

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Note ID |
| EntityType | string | epic\|feature\|task\|bug\|change_card |
| EntityID | int64 | FK to entity |
| NoteType | string | comment\|decision\|blocker\|solution\|rejection\|... |
| Content | string | Note text |
| CreatedBy | *string | Author |
| CreatedAt | time.Time | When created |

### Idea

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Idea ID |
| Key | string | Format: `I-YYYY-MM-DD-##` |
| Title | string | Idea summary |
| Description | string | Idea details |
| Status | string | new\|on_hold\|converted\|archived |
| Priority | int | 1-10 |
| ConvertedToType | *string | epic\|feature\|task |
| ConvertedToKey | *string | Key of promoted entity |

### WorkSession

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Session ID |
| TaskID | int64 | FK → tasks |
| Agent | string | Agent performing work |
| StartedAt | time.Time | Session start |
| EndedAt | *time.Time | Session end |
| Duration | *int64 | Duration in seconds |
| Notes | *string | Session notes |

## Service DTOs

### CreateTaskInput

```go
type CreateTaskInput struct {
    EpicKey        string
    FeatureKey     string
    Title          string
    AgentType      string
    Priority       int
    ExecutionOrder int
    DependsOn      []string
    FilePath       string
    CreateFile     bool
    Force          bool
}
```

### TaskFilters

```go
type TaskFilters struct {
    EpicKey    string
    FeatureKey string
    Status     string
    AgentType  string
    ShowAll    bool
}
```

### TransitionOptions

```go
type TransitionOptions struct {
    Force        bool
    Reason       string
    DocumentPath string
    Agent        string
}
```

### TransitionResult

```go
type TransitionResult struct {
    EntityType         models.EntityType
    EntityKey          string
    FromStatus         string
    ToStatus           string
    Transitioned       bool
    IsBackward         bool
    IsForced           bool
    Reason             string
    OrchestratorAction *config.PopulatedAction
    ChildCount         int
}
```

See also: [Program Structure](program-structure.md) | [Interfaces](interfaces.md) | [API Reference](api-reference.md)
