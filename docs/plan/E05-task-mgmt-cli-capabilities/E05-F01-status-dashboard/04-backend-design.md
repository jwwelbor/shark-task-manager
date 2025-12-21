# Backend Architecture: Status Dashboard (E05-F01)

**Document Version**: 1.0
**Status**: Complete Architecture Specification
**Last Updated**: 2025-12-19

---

## Table of Contents

1. [System Design Overview](#system-design-overview)
2. [Service Layer Architecture](#service-layer-architecture)
3. [Database Query Design](#database-query-design)
4. [Data Models & Output Contracts](#data-models--output-contracts)
5. [Performance Strategy](#performance-strategy)
6. [CLI Integration](#cli-integration)
7. [Error Handling & Edge Cases](#error-handling--edge-cases)
8. [Implementation Approach](#implementation-approach)

---

## System Design Overview

### 1.1 Architecture Position

The Status Dashboard (`shark status` command) integrates into the existing CLI architecture at the command layer, orchestrating data from multiple repositories to produce a comprehensive project view.

```
┌─────────────────────────────────────────────────────────────────┐
│ CLI Layer: shark status command (cobra.Command)                  │
├─────────────────────────────────────────────────────────────────┤
│ • Parse flags (--epic, --json, --no-color, --recent)             │
│ • Validate input                                                 │
│ • Call StatusService.GetDashboard()                              │
│ • Format output (JSON vs Rich tables)                            │
└─────────────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Service Layer: StatusService                                    │
├─────────────────────────────────────────────────────────────────┤
│ • Aggregate data from repositories                              │
│ • Filter by epic (if specified)                                 │
│ • Calculate progress metrics                                    │
│ • Group tasks by agent type                                     │
│ • Apply time-based filters (recent=24h|7d|etc)                  │
│ • Build StatusDashboard struct                                  │
└─────────────────────────────────────────────────────────────────┘
              ↓ (dependency injection)
┌──────────────┬──────────────┬──────────────┬──────────────────┐
│ Epic         │ Feature      │ Task         │ TaskHistory      │
│ Repository   │ Repository   │ Repository   │ Repository       │
├──────────────┴──────────────┴──────────────┴──────────────────┤
│ Existing repositories with optimized query methods             │
└──────────────────────────────────────────────────────────────┘
              ↓
┌──────────────────────────────────────────────────────────────┐
│ Database Layer: SQLite with WAL mode                          │
└──────────────────────────────────────────────────────────────┘
```

### 1.2 Data Flow

```
User runs: shark status --epic=E01 --json --recent=7d

↓

Command Handler (status.go)
├── Parse flags into StatusRequest
├── Create context with timeout (5 seconds)
├── Initialize StatusService(db, repos)
└── Call service.GetDashboard(ctx, request)

↓

StatusService
├── Query epics (filtered by E01 if specified)
├── Query features for each epic
├── Calculate epic progress with task aggregation
├── Query active tasks (in_progress status)
│   └── Group by agent_type
├── Query blocked tasks with reasons
├── Query recent completions (filtered by --recent=7d)
├── Build StatusDashboard struct
└── Return (dashboard, error)

↓

Output Formatter
├── If --json: Marshal to JSON with indentation
├── If --no-color: Strip Rich formatting, output plain tables
├── Else: Use Rich library for colored tables/progress bars
└── Write to stdout
```

### 1.3 Integration Points

The status service integrates with existing components:

- **Repositories**: Uses `EpicRepository`, `FeatureRepository`, `TaskRepository`, `TaskHistoryRepository`
- **Models**: Works with `Epic`, `Feature`, `Task`, `TaskHistory` models
- **Database**: Executes optimized queries against SQLite with indexes
- **CLI**: Plugs into Cobra command structure with global flags
- **Output Formatting**: Integrates with pterm/Rich for colored output, JSON marshaling for structured data

Key design principle: **Minimal changes to existing code**. The status service is self-contained, using only public APIs of existing repositories.

---

## Service Layer Architecture

### 2.1 StatusService Interface

```go
// StatusService aggregates project data for dashboard display
type StatusService interface {
    // GetDashboard returns complete dashboard data for specified filters
    // Context should have timeout for large projects (recommend 5s)
    GetDashboard(ctx context.Context, request *StatusRequest) (*StatusDashboard, error)

    // GetEpicStats returns metrics for a single epic (used by detailed views)
    GetEpicStats(ctx context.Context, epicKey string) (*EpicStats, error)
}

// StatusRequest describes what data to retrieve
type StatusRequest struct {
    // EpicKey filters dashboard to single epic (empty = all epics)
    EpicKey string

    // RecentWindow filters recent completions (e.g., "24h", "7d", "30d")
    // Default: "24h"
    RecentWindow string

    // IncludeArchived includes archived epics/features (default: false)
    IncludeArchived bool
}
```

**Key Design Decisions**:

1. **Interface-based**: StatusService is defined as an interface, not a concrete type. This allows for:
   - Easy testing with mock implementations
   - Future caching layer as a wrapper
   - Better separation of concerns

2. **Context support**: All methods accept `context.Context` for:
   - Request cancellation (user Ctrl+C)
   - Timeout management (abort long queries)
   - Future distributed tracing support

3. **Single responsibility**: StatusService focuses only on data aggregation, not output formatting. Output formatting is handled by command layer or separate formatters.

### 2.2 Service Implementation Structure

The concrete implementation (`statusService`) will have:

```go
type statusService struct {
    db               *db.DB
    epicRepo         *repository.EpicRepository
    featureRepo      *repository.FeatureRepository
    taskRepo         *repository.TaskRepository
    taskHistoryRepo  *repository.TaskHistoryRepository
}

// NewStatusService creates service with dependency injection
func NewStatusService(
    database *db.DB,
    epics *repository.EpicRepository,
    features *repository.FeatureRepository,
    tasks *repository.TaskRepository,
    history *repository.TaskHistoryRepository,
) StatusService {
    return &statusService{
        db:              database,
        epicRepo:        epics,
        featureRepo:     features,
        taskRepo:        tasks,
        taskHistoryRepo: history,
    }
}
```

**Private Methods** (internal implementation):

```go
// aggregateTaskStats calculates task counts by status for scope
func (s *statusService) aggregateTaskStats(ctx context.Context, scope scope) (*TaskStats, error)

// calculateEpicProgress computes progress percentage from task completion
func (s *statusService) calculateEpicProgress(ctx context.Context, epicID int64) (float64, error)

// groupTasksByAgent organizes tasks into agent_type groups with counts
func (s *statusService) groupTasksByAgent(ctx context.Context, tasks []*models.Task) map[string][]*models.Task

// filterByRecentWindow filters tasks by completion timestamp
func (s *statusService) filterByRecentWindow(tasks []*models.Task, window string) []*models.Task

// determineEpicHealth returns "healthy"/"warning"/"critical" based on progress/blockers
func (s *statusService) determineEpicHealth(progress float64, blockedCount int) string
```

### 2.3 Data Aggregation Logic

#### Epic Progress Calculation

Progress is calculated from task completion (not from feature progress):

```
Epic Progress = (sum of completed tasks in epic) / (total tasks in epic) * 100

Example:
  Epic E01 has features F01, F02, F03
  F01: 10 tasks, 8 completed
  F02: 15 tasks, 5 completed
  F03: 5 tasks, 5 completed

  Epic E01 progress = (8 + 5 + 5) / (10 + 15 + 5) * 100 = 18/30 * 100 = 60%
```

**Implementation approach**:
1. Fetch all features for epic
2. For each feature, fetch task count by status
3. Sum completed tasks across all features
4. Sum total tasks across all features
5. Calculate percentage

**Why not use feature progress_pct?**
- Feature progress_pct is cached/denormalized data
- Task status is the source of truth
- Provides consistent, real-time accuracy
- Handles features with no tasks gracefully (0% progress)

#### Active Task Grouping

Group in-progress tasks by agent_type for workload visibility:

```
Active Tasks = tasks where status = "in_progress"
  → Group by agent_type (frontend, backend, api, testing, devops, general, null)
  → Sort agents alphabetically
  → Within each agent, sort by task key
```

**Implementation approach**:
1. Query all tasks with status = "in_progress" (filtered by epic if specified)
2. Use map[string][]*models.Task to group by agent_type
3. Handle NULL agent_type as "unassigned" group
4. Sort groups consistently for stable output

#### Recent Completions Filtering

Apply time-based filter to completed tasks:

```
Valid timeframes: "24h", "7d", "30d", "90d" (case-insensitive)

Recent completions = tasks where:
  status = "completed" AND
  completed_at >= (now - timeframe)

Sorted by completed_at DESC (most recent first)
```

**Implementation approach**:
1. Parse timeframe into duration (e.g., "7d" → 7*24*time.Hour)
2. Calculate cutoff timestamp: now - duration
3. Query completed tasks since cutoff
4. Sort by completed_at descending
5. Return list with relative time format ("2 hours ago")

### 2.4 Filtering and Sorting

#### Epic Filtering

When `--epic=<key>` is specified:

1. **Validation**: Verify epic exists in database
2. **Scope narrowing**: All subsequent queries filtered to this epic's features and tasks
3. **Error handling**: Return error if epic not found (not silent failure)

#### Task Status Constants

Use model constants for consistency:

```go
models.TaskStatusTodo          // "todo"
models.TaskStatusInProgress    // "in_progress"
models.TaskStatusBlocked       // "blocked"
models.TaskStatusReadyForReview // "ready_for_review"
models.TaskStatusCompleted     // "completed"
models.TaskStatusArchived      // "archived"
```

#### Agent Type Sorting

Canonical agent type order (for consistent output):

```go
var AgentTypesOrder = []string{
    "frontend",
    "backend",
    "api",
    "testing",
    "devops",
    "general",
    "unassigned", // for NULL agent_type
}
```

Rationale: Frontend and backend are primary concerns (shown first), supporting types follow, unassigned last.

### 2.5 Error Handling Strategy

**Three error categories**:

1. **Query Errors** (database connection, timeout, constraint violation)
   - Wrap with context: `fmt.Errorf("failed to fetch epics: %w", err)`
   - Return to CLI layer for user-friendly display
   - Set exit code 2 (system error)

2. **Validation Errors** (invalid epic key, unsupported timeframe)
   - Return `StatusError` with message for user
   - Set exit code 1 (user error)
   - Example: "Invalid epic key: E999 (epic not found)"

3. **Data Inconsistency** (orphaned tasks, missing features)
   - Should not happen with FK constraints, but handle gracefully
   - Log warning, include in dashboard with 0 counts
   - Never crash; always return usable dashboard

**Error wrapping pattern**:

```go
if err != nil {
    return nil, fmt.Errorf("get epics for status dashboard: %w", err)
}
```

---

## Database Query Design

### 3.1 Query Optimization Principles

1. **Minimize round trips**: Use JOINs instead of sequential queries (avoid N+1)
2. **Aggregate in database**: Use COUNT, SUM in SQL, not application code
3. **Index-aware**: Queries leverage existing indexes on frequently queried columns
4. **Single query per section**: Each dashboard section built from one or two optimized queries
5. **Parameterized queries**: All inputs bound as parameters (SQL injection prevention)

### 3.2 Schema Context

**Relevant tables**:

```
epics
├── id (PK)
├── key (UNIQUE)
├── title
├── status (ENUM: draft|active|completed|archived)
├── priority
└── ... (other fields)

features
├── id (PK)
├── epic_id (FK → epics.id)
├── key (UNIQUE)
├── title
├── status
└── ... (other fields)

tasks
├── id (PK)
├── feature_id (FK → features.id)
├── key (UNIQUE)
├── title
├── status (ENUM: todo|in_progress|blocked|ready_for_review|completed|archived)
├── agent_type (VARCHAR)
├── blocked_reason (nullable)
├── completed_at (TIMESTAMP, nullable)
└── ... (other fields)

task_history
├── id (PK)
├── task_id (FK → tasks.id)
├── old_status
├── new_status
├── changed_at (TIMESTAMP)
└── ... (other fields)
```

**Existing indexes to leverage**:

- `idx_features_epic_id` on features(epic_id)
- `idx_tasks_feature_id` on tasks(feature_id)
- `idx_tasks_status` on tasks(status)
- `idx_tasks_agent_type` on tasks(agent_type)
- `idx_tasks_completed_at` on tasks(completed_at)

### 3.3 Query Patterns for Each Dashboard Section

#### Query 1: Project Summary (All Epics)

**Purpose**: Get overall project metrics

**SQL**:
```sql
-- Count of epics by status
SELECT
    COUNT(*) as total_epics,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_epics,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_epics
FROM epics
WHERE archived = false;

-- Count of features by status
SELECT
    COUNT(*) as total_features,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_features,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_features
FROM features
WHERE epic_id IN (
    SELECT id FROM epics WHERE archived = false
);

-- Count of tasks by status
SELECT
    COUNT(*) as total_tasks,
    SUM(CASE WHEN status = 'todo' THEN 1 ELSE 0 END) as todo_count,
    SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) as in_progress_count,
    SUM(CASE WHEN status = 'ready_for_review' THEN 1 ELSE 0 END) as review_count,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_count,
    SUM(CASE WHEN status = 'blocked' THEN 1 ELSE 0 END) as blocked_count
FROM tasks
WHERE feature_id IN (
    SELECT f.id FROM features f
    JOIN epics e ON f.epic_id = e.id
    WHERE e.archived = false
);

-- Overall progress (sum of all completed / sum of all tasks)
SELECT
    CAST(
        SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS FLOAT)
        / COUNT(*) * 100 as overall_progress
FROM tasks
WHERE feature_id IN (
    SELECT f.id FROM features f
    JOIN epics e ON f.epic_id = e.id
    WHERE e.archived = false
);
```

**Optimization notes**:
- Three separate queries (each returns single row with aggregates)
- Could be combined into single query with CTEs for efficiency
- Each uses COUNT/SUM for database-level aggregation (fast)
- Avoid N+1 by using single query with aggregates vs. loading all rows

#### Query 2: Epic Breakdown

**Purpose**: Get one row per epic with progress and task counts

**SQL**:
```sql
SELECT
    e.id,
    e.key,
    e.title,
    e.priority,
    COUNT(DISTINCT t.id) as total_tasks,
    SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END) as completed_tasks,
    SUM(CASE WHEN t.status = 'blocked' THEN 1 ELSE 0 END) as blocked_tasks,
    CAST(
        SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END) AS FLOAT)
        / COUNT(DISTINCT t.id) * 100 as progress_pct
FROM epics e
LEFT JOIN features f ON e.id = f.epic_id
LEFT JOIN tasks t ON f.id = t.feature_id
WHERE e.archived = false
GROUP BY e.id, e.key, e.title, e.priority
ORDER BY e.priority DESC, e.key ASC;
```

**Optimization notes**:
- Uses LEFT JOINs to handle epics with no features/tasks
- Single query produces all rows needed for epic table
- GROUP BY with aggregates in database
- ORDER BY enforces sorting by priority then key
- No N+1: Single query returns all epics with computed progress

#### Query 3: Active Tasks (in_progress)

**Purpose**: Get all in-progress tasks with their details for grouping by agent

**SQL**:
```sql
SELECT
    t.id,
    t.key,
    t.title,
    t.agent_type,
    e.key as epic_key,
    f.key as feature_key
FROM tasks t
JOIN features f ON t.feature_id = f.id
JOIN epics e ON f.epic_id = e.id
WHERE t.status = 'in_progress'
  AND e.archived = false
ORDER BY t.agent_type ASC NULLS LAST, t.key ASC;
```

**Optimization notes**:
- Uses indexes on tasks(status) and relationships
- NULLS LAST puts unassigned tasks at end
- Sorted by agent_type for stable grouping
- No filtering of archived (in_progress tasks shouldn't be archived, but defensive)

#### Query 4: Blocked Tasks with Reasons

**Purpose**: Get blocked tasks with their blocking reasons

**SQL**:
```sql
SELECT
    t.id,
    t.key,
    t.title,
    t.blocked_reason,
    t.priority,
    t.blocked_at,
    e.key as epic_key,
    f.key as feature_key
FROM tasks t
JOIN features f ON t.feature_id = f.id
JOIN epics e ON f.epic_id = e.id
WHERE t.status = 'blocked'
  AND e.archived = false
ORDER BY t.priority DESC, t.blocked_at DESC NULLS LAST;
```

**Optimization notes**:
- Single query for all blocked tasks
- Sorted by priority (high first) then timestamp (most recent first)
- Includes blocked_reason field for display
- Could add index on (status, priority) if blocking is common operation

#### Query 5: Recent Completions

**Purpose**: Get tasks completed in recent timeframe (e.g., last 24 hours)

**SQL**:
```sql
SELECT
    t.id,
    t.key,
    t.title,
    t.completed_at,
    e.key as epic_key,
    f.key as feature_key
FROM tasks t
JOIN features f ON t.feature_id = f.id
JOIN epics e ON f.epic_id = e.id
WHERE t.status = 'completed'
  AND t.completed_at >= datetime('now', ?1)
  AND e.archived = false
ORDER BY t.completed_at DESC
LIMIT 100;
```

**Parameters**:
- `?1`: SQL modifier like `-24 hours`, `-7 days` for SQLite `datetime()` function

**Optimization notes**:
- Uses index on tasks(completed_at)
- Filters by timestamp in database (not application code)
- LIMIT 100 prevents large result sets
- Sorted by most recent first
- NULL completed_at handled by constraint (should never be NULL for completed tasks)

#### Query 6: Epic-Filtered Project Summary

**Purpose**: When `--epic=E01` specified, get summary for that epic only

**SQL**:
```sql
-- Modified to filter to single epic
SELECT
    COUNT(*) as total_features,
    SUM(CASE WHEN f.status = 'active' THEN 1 ELSE 0 END) as active_features,
    SUM(CASE WHEN f.status = 'completed' THEN 1 ELSE 0 END) as completed_features
FROM features f
WHERE f.epic_id = (SELECT id FROM epics WHERE key = ?1);

SELECT
    COUNT(*) as total_tasks,
    SUM(CASE WHEN t.status = 'todo' THEN 1 ELSE 0 END) as todo_count,
    SUM(CASE WHEN t.status = 'in_progress' THEN 1 ELSE 0 END) as in_progress_count,
    SUM(CASE WHEN t.status = 'ready_for_review' THEN 1 ELSE 0 END) as review_count,
    SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END) as completed_count,
    SUM(CASE WHEN t.status = 'blocked' THEN 1 ELSE 0 END) as blocked_count
FROM tasks t
WHERE t.feature_id IN (
    SELECT id FROM features
    WHERE epic_id = (SELECT id FROM epics WHERE key = ?1)
);
```

**Optimization notes**:
- Subquery on epic.key = ?1 is cached/optimized by SQLite
- Results in same table structure as full dashboard

### 3.4 Index Requirements

**Ensure these indexes exist** (document for migration):

```sql
-- Already exist (verify in schema)
CREATE INDEX IF NOT EXISTS idx_features_epic_id ON features(epic_id);
CREATE INDEX IF NOT EXISTS idx_tasks_feature_id ON tasks(feature_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_completed_at ON tasks(completed_at);

-- Verify or add for performance
CREATE INDEX IF NOT EXISTS idx_tasks_agent_type ON tasks(agent_type);
CREATE INDEX IF NOT EXISTS idx_epics_archived ON epics(archived);
CREATE INDEX IF NOT EXISTS idx_features_archived ON features(archived);
```

**Composite indexes for common queries**:

```sql
-- Helps with (status, completed_at) filtering
CREATE INDEX IF NOT EXISTS idx_tasks_status_completed_at
    ON tasks(status, completed_at);

-- Helps with epic progress calculation
CREATE INDEX IF NOT EXISTS idx_tasks_feature_status
    ON tasks(feature_id, status);
```

---

## Data Models & Output Contracts

### 4.1 Intermediate Data Structures

These structs are used internally by StatusService and returned in StatusDashboard:

```go
// StatusDashboard represents complete dashboard data
type StatusDashboard struct {
    Summary           *ProjectSummary        `json:"summary"`
    Epics             []*EpicSummary         `json:"epics"`
    ActiveTasks       map[string][]*TaskInfo `json:"active_tasks"`      // grouped by agent_type
    BlockedTasks      []*BlockedTaskInfo     `json:"blocked_tasks"`
    RecentCompletions []*CompletionInfo      `json:"recent_completions"`
    Filter            *DashboardFilter       `json:"filter,omitempty"`  // populated if filtered
}

// ProjectSummary contains high-level project metrics
type ProjectSummary struct {
    Epics   *CountBreakdown `json:"epics"`    // {total, active, completed}
    Features *CountBreakdown `json:"features"` // {total, active, completed}
    Tasks   *StatusBreakdown `json:"tasks"`    // {total, todo, in_progress, ready_for_review, completed, blocked}
    OverallProgress float64 `json:"overall_progress"` // 0.0 to 100.0
    BlockedCount    int     `json:"blocked_count"`
}

// CountBreakdown shows distribution across status (active/completed)
type CountBreakdown struct {
    Total     int `json:"total"`
    Active    int `json:"active"`
    Completed int `json:"completed"`
}

// StatusBreakdown shows distribution across task statuses
type StatusBreakdown struct {
    Total        int `json:"total"`
    Todo         int `json:"todo"`
    InProgress   int `json:"in_progress"`
    ReadyForReview int `json:"ready_for_review"`
    Completed    int `json:"completed"`
    Blocked      int `json:"blocked"`
}

// EpicSummary represents single epic with progress
type EpicSummary struct {
    Key        string  `json:"key"`
    Title      string  `json:"title"`
    Progress   float64 `json:"progress"`    // 0.0 to 100.0
    Health     string  `json:"health"`      // "healthy"|"warning"|"critical"
    TasksTotal int     `json:"tasks_total"`
    TasksCompleted int `json:"tasks_completed"`
    BlockedCount   int `json:"blocked_count"`
    Priority   string  `json:"priority"`    // "high"|"medium"|"low"
    Status     string  `json:"status"`      // "draft"|"active"|"completed"|"archived"
}

// TaskInfo represents task in active/blocked list
type TaskInfo struct {
    Key       string  `json:"key"`
    Title     string  `json:"title"`
    EpicKey   string  `json:"epic_key"`
    FeatureKey string `json:"feature_key"`
    AgentType *string `json:"agent_type,omitempty"`
}

// BlockedTaskInfo includes blocking reason
type BlockedTaskInfo struct {
    Key          string     `json:"key"`
    Title        string     `json:"title"`
    EpicKey      string     `json:"epic_key"`
    FeatureKey   string     `json:"feature_key"`
    BlockedReason *string   `json:"blocked_reason,omitempty"`
    Priority     int        `json:"priority"`
    BlockedAt    time.Time  `json:"blocked_at"`
}

// CompletionInfo includes relative time
type CompletionInfo struct {
    Key         string    `json:"key"`
    Title       string    `json:"title"`
    EpicKey     string    `json:"epic_key"`
    FeatureKey  string    `json:"feature_key"`
    CompletedAt time.Time `json:"completed_at"`
    CompletedAgo string   `json:"completed_ago"` // e.g., "2 hours ago"
}

// DashboardFilter indicates what filtering was applied
type DashboardFilter struct {
    EpicKey      *string `json:"epic_key,omitempty"`      // if --epic specified
    RecentWindow *string `json:"recent_window,omitempty"` // if --recent specified
}
```

**Design rationale**:

1. **Separate from models.Task**: These DTOs are tailored for dashboard display, not general-purpose task data
2. **JSON tags**: All fields have `json` tags for proper JSON marshaling
3. **Nullable fields**: Use pointers for optional fields (e.g., `*string` for blocked_reason)
4. **Grouped data**: `ActiveTasks` is map[string][]*TaskInfo for agent-based grouping
5. **Health status**: Computed field on EpicSummary (not stored in database)
6. **Relative time**: CompletionInfo includes both timestamp and human-readable "ago" format

### 4.2 JSON Output Schema

**Full example output**:

```json
{
  "summary": {
    "epics": {
      "total": 3,
      "active": 2,
      "completed": 1
    },
    "features": {
      "total": 12,
      "active": 8,
      "completed": 4
    },
    "tasks": {
      "total": 127,
      "todo": 45,
      "in_progress": 12,
      "ready_for_review": 5,
      "completed": 60,
      "blocked": 5
    },
    "overall_progress": 47.2,
    "blocked_count": 5
  },
  "epics": [
    {
      "key": "E01",
      "title": "Identity Platform",
      "progress": 60.0,
      "health": "healthy",
      "tasks_total": 50,
      "tasks_completed": 30,
      "blocked_count": 2,
      "priority": "high",
      "status": "active"
    },
    {
      "key": "E02",
      "title": "Task Management CLI",
      "progress": 40.0,
      "health": "warning",
      "tasks_total": 50,
      "tasks_completed": 20,
      "blocked_count": 1,
      "priority": "high",
      "status": "active"
    },
    {
      "key": "E03",
      "title": "Documentation System",
      "progress": 100.0,
      "health": "healthy",
      "tasks_total": 27,
      "tasks_completed": 27,
      "blocked_count": 0,
      "priority": "medium",
      "status": "completed"
    }
  ],
  "active_tasks": {
    "frontend": [
      {
        "key": "T-E01-F02-005",
        "title": "Build user profile component",
        "epic_key": "E01",
        "feature_key": "F02",
        "agent_type": "frontend"
      },
      {
        "key": "T-E02-F01-003",
        "title": "Create task list UI",
        "epic_key": "E02",
        "feature_key": "F01",
        "agent_type": "frontend"
      }
    ],
    "backend": [
      {
        "key": "T-E01-F01-002",
        "title": "Implement JWT validation",
        "epic_key": "E01",
        "feature_key": "F01",
        "agent_type": "backend"
      }
    ],
    "general": [
      {
        "key": "T-E03-F01-001",
        "title": "Write API documentation",
        "epic_key": "E03",
        "feature_key": "F01",
        "agent_type": "general"
      }
    ]
  },
  "blocked_tasks": [
    {
      "key": "T-E01-F02-003",
      "title": "User authentication flow",
      "epic_key": "E01",
      "feature_key": "F02",
      "blocked_reason": "Waiting for API specification from backend team",
      "priority": 8,
      "blocked_at": "2025-12-17T14:30:00Z"
    },
    {
      "key": "T-E02-F01-007",
      "title": "Task dependency validation",
      "epic_key": "E02",
      "feature_key": "F01",
      "blocked_reason": "Missing dependency graph algorithm implementation",
      "priority": 6,
      "blocked_at": "2025-12-18T09:15:00Z"
    }
  ],
  "recent_completions": [
    {
      "key": "T-E01-F01-003",
      "title": "JWT token generation",
      "epic_key": "E01",
      "feature_key": "F01",
      "completed_at": "2025-12-18T10:30:00Z",
      "completed_ago": "2 hours ago"
    },
    {
      "key": "T-E01-F02-001",
      "title": "Login form component",
      "epic_key": "E01",
      "feature_key": "F02",
      "completed_at": "2025-12-18T05:45:00Z",
      "completed_ago": "7 hours ago"
    }
  ],
  "filter": {
    "epic_key": null,
    "recent_window": "24h"
  }
}
```

**Schema validation**:

- All timestamp fields use RFC3339 format (ISO 8601): "2025-12-18T10:30:00Z"
- Progress values: 0.0 to 100.0 (float)
- All IDs/keys: strings (not numbers, for consistency)
- Status enums: lowercase, snake_case
- Health enum: "healthy", "warning", "critical"
- Active tasks grouped by agent_type (string key), null agent_type handled as separate group or empty string

### 4.3 Health Status Calculation

Epic health determined by:

```go
func (s *statusService) determineEpicHealth(progress float64, blockedCount int) string {
    // Critical: <25% progress OR >3 blocked tasks
    if progress < 25.0 || blockedCount > 3 {
        return "critical"
    }

    // Warning: 25-74% progress with 1-3 blocked tasks
    if progress < 75.0 || blockedCount > 0 {
        return "warning"
    }

    // Healthy: ≥75% progress AND no blocked tasks
    return "healthy"
}
```

Health map for CLI color coding:
- `healthy` → green
- `warning` → yellow
- `critical` → red

---

## Performance Strategy

### 5.1 Performance Goals

- **Dashboard rendering**: <500ms for 100 epics with 1000+ tasks
- **Query execution**: <200ms for all database queries
- **Output formatting**: <100ms for JSON serialization and table rendering
- **Memory usage**: <50MB for typical project data (100 epics, 1000 tasks)

### 5.2 Query Optimization Techniques

#### Technique 1: Query Aggregation

Execute aggregations in database, not application:

```go
// GOOD: Single query with COUNT/SUM
SELECT COUNT(*) as total,
       SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END) as completed
FROM tasks WHERE feature_id = ?

// BAD: Load all tasks, count in Go
tasks := fetchAllTasks()  // N rows × memory
completed := 0
for _, t := range tasks {
    if t.Status == "completed" { completed++ }
}
```

**Expected improvement**: 10-50x faster for large datasets

#### Technique 2: JOIN vs Multiple Queries

```go
// GOOD: Single JOIN query
SELECT t.key, t.title, e.key as epic_key
FROM tasks t
JOIN features f ON t.feature_id = f.id
JOIN epics e ON f.epic_id = e.id
WHERE t.status = 'in_progress'

// BAD: N+1 - fetch tasks, then for each task fetch epic
tasks := fetchTasks()
for _, t := range tasks {
    f := fetchFeature(t.FeatureID)      // N queries
    e := fetchEpic(f.EpicID)            // N queries
}
```

**Expected improvement**: 100-1000x faster due to eliminating N+1

#### Technique 3: Index Leverage

Queries designed to use existing indexes:

```sql
-- Uses idx_tasks_status index
WHERE t.status = 'in_progress'

-- Uses idx_tasks_completed_at index
WHERE t.completed_at >= ?

-- Uses idx_tasks_feature_id for JOIN
JOIN features f ON t.feature_id = f.id
```

Verify with `EXPLAIN QUERY PLAN`:

```bash
sqlite3 shark-tasks.db "EXPLAIN QUERY PLAN SELECT ..."
```

#### Technique 4: Selective Field Selection

```go
// GOOD: Select only needed fields
SELECT t.key, t.title, t.agent_type FROM tasks WHERE ...

// LESS GOOD: Select everything
SELECT * FROM tasks WHERE ...
```

**Expected improvement**: 5-10% with large rows

#### Technique 5: Result Limiting

For sections that can have large results:

```sql
-- Recent completions: limit to 100
WHERE t.completed_at >= ?
LIMIT 100

-- Active tasks: typically <100, but safe limit
WHERE t.status = 'in_progress'
LIMIT 1000
```

### 5.3 Caching Considerations

**Not recommended** for initial implementation:

- Status data is derived from multiple queries (complex cache invalidation)
- Dashboard is typically run once at command time (not repeated)
- Cache would only help if user runs multiple queries in sequence
- Complexity not justified for <500ms target

**Future enhancement**:

If performance becomes issue, wrap StatusService in caching layer:

```go
type cachedStatusService struct {
    underlying StatusService
    cache      map[string]*statusDashboard  // key = epic_key + recent_window
    mu         sync.RWMutex
    ttl        time.Duration  // 5 minutes
}
```

### 5.4 Benchmarking Approach

**Benchmark scenarios** (in tests):

1. **Empty database**: 0 epics, 0 tasks
2. **Small project**: 3 epics, 12 features, 127 tasks
3. **Large project**: 100 epics, 500 features, 2000+ tasks
4. **Slow database**: Simulate with PRAGMA busy_timeout
5. **Filtered query**: --epic=E01 (should be <50ms)

**Benchmark code structure**:

```go
func BenchmarkStatusServiceGetDashboard(b *testing.B) {
    // Setup: create test database with known data
    db := setupTestDB()
    defer db.Close()

    service := NewStatusService(db, repos...)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.GetDashboard(ctx, &StatusRequest{})
    }
    b.StopTimer()

    // Verify performance
    b.ReportAllocs()  // Show memory allocations
    if b.Elapsed().Seconds()/float64(b.N) > 0.5 {
        b.Fatalf("Too slow: %v per iteration", b.Elapsed()/time.Duration(b.N))
    }
}
```

**Run benchmarks**:

```bash
go test -bench=StatusService -benchmem ./internal/service
```

### 5.5 Monitoring & Profiling

**In-process monitoring**:

```go
// Track query times
start := time.Now()
epics, err := s.epicRepo.List(ctx)
duration := time.Since(start)
if duration > 100*time.Millisecond {
    log.Printf("WARNING: GetDashboard epic query took %v", duration)
}
```

**CPU profiling** (if needed):

```bash
go test -cpuprofile=cpu.prof ./internal/service
go tool pprof cpu.prof
```

**Memory profiling**:

```bash
go test -memprofile=mem.prof ./internal/service
go tool pprof mem.prof
```

### 5.6 Achieving <500ms Goal

**Strategy**:

1. **Database queries** (target: <200ms)
   - Use indexes: Verify with EXPLAIN QUERY PLAN
   - Aggregate in SQL: COUNT, SUM, GROUP BY in database
   - JOINs not N+1: Single query per data section

2. **Application logic** (target: <150ms)
   - Minimal Go code: Only grouping and filtering
   - No unnecessary allocations: Use preallocated slices
   - Parallel queries: If using context, can parallelize independent queries

3. **Output formatting** (target: <150ms)
   - JSON marshaling: Standard library json.Marshal (fast)
   - Table rendering: pterm library is optimized
   - Color codes: Applied at format time, not query time

**For typical projects (3-5 epics, 50-100 tasks)**:
- Actual runtime: 50-150ms (well under budget)
- For large projects: Scale linearly with data size

---

## CLI Integration

### 6.1 Command Definition

The `shark status` command integrates into existing CLI structure:

```go
// Location: internal/cli/commands/status.go

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Display project dashboard",
    Long: `Display comprehensive project status dashboard with epic progress,
active tasks, blocked tasks, and recent completions.

The dashboard shows:
  • Project summary (total epics/features/tasks)
  • Epic breakdown with progress percentages
  • Active tasks grouped by agent type
  • Blocked tasks with blocking reasons
  • Recent completions from last 24 hours

Examples:
  shark status                 Show full project dashboard
  shark status --epic=E01      Show status for specific epic only
  shark status --json          Output as JSON for parsing
  shark status --no-color      Output without color codes
  shark status --recent=7d     Show completions from last 7 days
  shark status --epic=E01 --json  Combine multiple options`,

    RunE: runStatus,
}
```

### 6.2 Command Flags

```go
func init() {
    statusCmd.Flags().StringVar(
        &flagEpicKey, "epic", "",
        "Filter dashboard to specific epic (e.g., 'E01')",
    )

    statusCmd.Flags().StringVar(
        &flagRecentWindow, "recent", "24h",
        "Time window for recent completions (e.g., '24h', '7d', '30d')",
    )

    statusCmd.Flags().BoolVar(
        &flagIncludeArchived, "include-archived", false,
        "Include archived epics and features (default: false)",
    )

    // Standard flags from RootCmd are inherited:
    // --json, --no-color, --verbose, --db, --config

    cli.RootCmd.AddCommand(statusCmd)
}
```

### 6.3 Command Handler Implementation

```go
func runStatus(cmd *cobra.Command, args []string) error {
    // 1. Load database
    database, err := initDatabase()
    if err != nil {
        return cli.Error(fmt.Sprintf("Failed to load database: %v", err))
    }
    defer database.Close()

    // 2. Initialize repositories
    epicRepo := repository.NewEpicRepository(database)
    featureRepo := repository.NewFeatureRepository(database)
    taskRepo := repository.NewTaskRepository(database)
    historyRepo := repository.NewTaskHistoryRepository(database)

    // 3. Create status service
    service := status.NewStatusService(database, epicRepo, featureRepo, taskRepo, historyRepo)

    // 4. Build request from flags
    request := &status.StatusRequest{
        EpicKey:         flagEpicKey,
        RecentWindow:    flagRecentWindow,
        IncludeArchived: flagIncludeArchived,
    }

    // 5. Get dashboard data with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    dashboard, err := service.GetDashboard(ctx, request)
    if err != nil {
        return cli.Error(fmt.Sprintf("Error retrieving status: %v", err))
    }

    // 6. Output results
    if cli.GlobalConfig.JSON {
        return outputJSON(dashboard)
    } else {
        return outputRichTable(dashboard)
    }
}
```

### 6.4 Output Formatting

#### JSON Output

```go
func outputJSON(dashboard *status.StatusDashboard) error {
    data, err := json.MarshalIndent(dashboard, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal JSON: %w", err)
    }
    fmt.Println(string(data))
    return nil
}
```

#### Rich Table Output

Uses pterm library for colored tables:

```go
func outputRichTable(dashboard *status.StatusDashboard, noColor bool) error {
    // Project Summary section
    outputProjectSummary(dashboard.Summary)

    // Epic Breakdown section
    outputEpicTable(dashboard.Epics)

    // Active Tasks section
    outputActiveTasks(dashboard.ActiveTasks)

    // Blocked Tasks section
    outputBlockedTasks(dashboard.BlockedTasks)

    // Recent Completions section
    outputRecentCompletions(dashboard.RecentCompletions)

    return nil
}
```

**Color coding implementation**:

```go
func getHealthColor(health string) string {
    switch health {
    case "healthy":
        return "[green]"  // pterm color code
    case "warning":
        return "[yellow]"
    case "critical":
        return "[red]"
    default:
        return "[white]"
    }
}

func renderProgressBar(progress float64, health string) string {
    filled := int(progress / 5)      // 20-char bar
    empty := 20 - filled
    bar := "[" + strings.Repeat("#", filled) + strings.Repeat("-", empty) + "]"

    color := getHealthColor(health)
    return fmt.Sprintf("%s %s[reset] %5.1f%%", color, bar, progress)
}
```

### 6.5 Edge Cases and Error Handling

#### No Data Cases

```go
// No epics found
if len(dashboard.Epics) == 0 {
    fmt.Println("\n[yellow]No epics found. Create epics to get started.[/reset]")
    return nil  // Not an error condition
}

// No active tasks
if len(dashboard.ActiveTasks) == 0 {
    fmt.Println("\n[green]No tasks currently in progress[/reset]")
}

// No blocked tasks
if len(dashboard.BlockedTasks) == 0 {
    fmt.Println("\n[green]No blocked tasks[/reset]")
}

// No recent completions
if len(dashboard.RecentCompletions) == 0 {
    fmt.Printf("\n[gray]No tasks completed in last %s[/reset]\n", recentWindow)
}
```

#### Epic Not Found

When `--epic=E999` specified and epic doesn't exist:

```go
if request.EpicKey != "" {
    epic, err := epicRepo.GetByKey(ctx, request.EpicKey)
    if err != nil {
        return cli.Error(fmt.Sprintf(
            "Epic not found: %s\nUse 'shark epic list' to see available epics",
            request.EpicKey))
    }
}
```

#### Invalid Timeframe

When `--recent=invalid` specified:

```go
validTimeframes := map[string]bool{
    "24h": true, "1d": true,
    "7d": true, "48h": true,
    "30d": true, "90d": true,
}
if !validTimeframes[request.RecentWindow] {
    return cli.Error(fmt.Sprintf(
        "Invalid timeframe: %s\nValid options: 24h, 7d, 30d, 90d",
        request.RecentWindow))
}
```

#### Terminal Width

Handle narrow/wide terminals gracefully:

```go
func getTerminalWidth() int {
    width, _, _ := terminal.GetSize(int(os.Stdout.Fd()))
    if width < 80 {
        width = 80  // Minimum width
    }
    return width
}

// Truncate long titles
func truncateTitle(title string, maxWidth int) string {
    if len(title) <= maxWidth {
        return title
    }
    return title[:maxWidth-3] + "..."
}
```

#### Database Connection Error

```go
database, err := initDatabase()
if err != nil {
    return cli.Error(fmt.Sprintf(
        "Failed to open database: %v\nCheck database path with --db flag",
        err))
    // Exit code 2 (system error)
}
```

#### Timeout on Large Projects

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

dashboard, err := service.GetDashboard(ctx, request)
if err == context.DeadlineExceeded {
    return cli.Error(
        "Dashboard query timed out (>5s)\n" +
        "Try filtering with --epic=<key> or check database performance")
    // Exit code 2 (system error)
}
```

---

## Error Handling & Edge Cases

### 7.1 Error Categories

#### Category 1: Database Errors

| Scenario | Error Message | Exit Code |
|----------|---------------|-----------|
| Connection failed | "Failed to open database: [details]" | 2 |
| Query timeout | "Dashboard query timed out (>5s)" | 2 |
| Query fails | "Error retrieving status: [details]" | 2 |
| Constraint violation | "Data inconsistency detected: [details]" | 2 |

#### Category 2: Validation Errors

| Scenario | Error Message | Exit Code |
|----------|---------------|-----------|
| Invalid epic key | "Epic not found: E999" | 1 |
| Invalid timeframe | "Invalid timeframe: badformat (try: 24h, 7d, 30d)" | 1 |
| Missing required arg | "Error: no arguments provided" | 2 |

#### Category 3: Data Issues

| Scenario | Handling | Impact |
|----------|----------|--------|
| Orphaned tasks (no feature) | Skip in aggregation, log warning | Data missing from dashboard |
| NULL agent_type | Group as "unassigned" | Display normally |
| Incomplete timestamps | Use zero values or skip | May omit from time-based filters |
| Archived epics | Exclude by default, include with --include-archived | Reduces clutter |

### 7.2 Defensive Coding Patterns

#### Pattern 1: Nil Checks

```go
// When reading optional fields
if task.BlockedReason != nil {
    fmt.Println("Reason: " + *task.BlockedReason)
} else {
    fmt.Println("Reason: (none)")
}
```

#### Pattern 2: Graceful Degradation

```go
// If a single epic query fails, don't crash
epic, err := s.epicRepo.GetByKey(ctx, key)
if err != nil {
    log.Printf("Warning: could not load epic %s: %v", key, err)
    // Continue with partial data
    epic = &models.Epic{Key: key, Title: "Unknown"}
}
```

#### Pattern 3: Value Validation

```go
// Validate progress before display
progress := math.Min(100.0, math.Max(0.0, dashboard.Summary.OverallProgress))
fmt.Printf("Progress: %.1f%%\n", progress)
```

### 7.3 Common Edge Cases

#### Edge Case 1: Empty Project

```
Input: Project with 0 epics
Expected: Display message "No epics found"
Exit Code: 0 (success)
```

#### Edge Case 2: Epic with No Tasks

```
Input: Epic E01 with 2 features but 0 tasks
Expected: Progress = 0%, display empty lists
Exit Code: 0
```

#### Edge Case 3: All Tasks Blocked

```
Input: Epic E01 with 10 tasks, all status=blocked
Expected: Progress = 0%, blocked section shows all 10, health=critical
Exit Code: 0
```

#### Edge Case 4: Very Wide Terminal

```
Input: Terminal width > 200 characters
Expected: Use full width for tables, no truncation
Exit Code: 0
```

#### Edge Case 5: Very Narrow Terminal

```
Input: Terminal width < 80 characters
Expected: Truncate titles, wrap if necessary, still readable
Exit Code: 0
```

---

## Implementation Approach

### 8.1 File Structure

**New files to create**:

```
internal/status/
├── status.go              # Service interface & implementation
├── models.go              # Data structures (StatusDashboard, EpicSummary, etc)
├── errors.go              # Error types
└── status_test.go         # Unit tests + benchmarks

internal/cli/commands/
└── status.go              # Command definition & handler (updated)
└── status_test.go         # Command integration tests (new)
```

**Modified files**:

- `internal/cli/commands/commands.go` - Import status command package
- `internal/cli/root.go` - No changes needed (global flags already exist)

### 8.2 Dependency Injection Pattern

**Service creation**:

```go
// In command handler
service := status.NewStatusService(
    database,
    repository.NewEpicRepository(database),
    repository.NewFeatureRepository(database),
    repository.NewTaskRepository(database),
    repository.NewTaskHistoryRepository(database),
)
```

**No changes to existing repositories needed** - use existing public methods only.

**Repositories used**:

- `EpicRepository.List(ctx)` - Get all epics
- `EpicRepository.GetByKey(ctx, key)` - Get single epic
- `FeatureRepository.ListByEpic(ctx, epicID)` - Get features for epic
- `TaskRepository.ListByStatus(ctx, status)` - Get tasks by status
- `TaskRepository.ListByFeature(ctx, featureID)` - Get tasks for feature
- `TaskHistoryRepository.GetRecentCompletions(ctx, since)` - Get recent completions

*Note: If these methods don't exist, use existing methods and implement filtering in service layer*

### 8.3 Testing Strategy

#### Unit Tests

```go
// Test each service method independently

func TestStatusService_GetDashboard_EmptyProject(t *testing.T) {
    // Arrange: Create mock repositories that return empty data
    // Act: Call service.GetDashboard()
    // Assert: Dashboard has zero counts, no epics, no tasks
}

func TestStatusService_GetDashboard_WithBlocked(t *testing.T) {
    // Arrange: Create epic with 5 blocked tasks
    // Act: Call service.GetDashboard()
    // Assert: EpicSummary.Health == "critical"
}

func TestStatusService_FilterByRecentWindow(t *testing.T) {
    // Test each timeframe: 24h, 7d, 30d, 90d
    // Verify tasks older than window are excluded
}

func TestStatusService_GroupTasksByAgent(t *testing.T) {
    // Test grouping with various agent types
    // Test NULL agent_type handling
    // Test sorting order
}
```

#### Integration Tests

```go
// Test with real database (like existing tests)

func TestStatusCommand_FullProject(t *testing.T) {
    // Create test database with known data
    db := setupTestDB()
    defer db.Close()

    // Run command with various flags
    cmd := &cobra.Command{
        RunE: runStatus,
    }
    // Verify output format and correctness
}

func TestStatusCommand_JSONOutput(t *testing.T) {
    // Run command with --json
    // Parse output as JSON
    // Verify structure and values
}

func TestStatusCommand_FilteredByEpic(t *testing.T) {
    // Run command with --epic=E01
    // Verify output contains only E01 data
}
```

#### Benchmark Tests

```go
func BenchmarkStatusService_GetDashboard_SmallProject(b *testing.B) {
    // 3 epics, 12 features, 127 tasks
    // Target: <50ms
}

func BenchmarkStatusService_GetDashboard_LargeProject(b *testing.B) {
    // 100 epics, 500 features, 2000 tasks
    // Target: <500ms
}
```

### 8.4 Implementation Phases

#### Phase 1: Core Service (3-4 hours)

1. Create `internal/status/models.go` - Data structures
2. Create `internal/status/status.go` - Service interface & basic implementation
3. Implement `GetDashboard()` with basic flow
4. Write unit tests for data structures and aggregation logic

**Deliverable**: Service that aggregates data from repositories, no output formatting

#### Phase 2: Database Queries (2-3 hours)

1. Implement efficient queries for each dashboard section
2. Add query methods to StatusService
3. Optimize queries with indexes and JOINs
4. Benchmark query performance

**Deliverable**: Service that queries database efficiently (<200ms for 100 epics)

#### Phase 3: CLI Command (2-3 hours)

1. Create `internal/cli/commands/status.go`
2. Implement command handler and flag parsing
3. Integrate with StatusService
4. Add JSON output formatting

**Deliverable**: Functional `shark status --json` command

#### Phase 4: Rich Output Formatting (2-3 hours)

1. Implement Rich table output for each section
2. Add color coding based on health/status
3. Handle terminal width and wrapping
4. Handle edge cases (empty sections, etc)

**Deliverable**: Fully formatted `shark status` with colors and tables

#### Phase 5: Testing & Optimization (3-4 hours)

1. Write integration tests
2. Write benchmark tests
3. Profile and optimize hot paths
4. Verify <500ms goal on large projects

**Deliverable**: Tested, optimized service meeting performance goals

### 8.5 Code Quality Standards

Follow existing project patterns:

- **Error handling**: Use `fmt.Errorf("context: %w", err)` for wrapping
- **Logging**: Use project's logging approach (pterm for user output)
- **Comments**: Document public interfaces and complex logic
- **Naming**: Consistent with existing codebase
- **Testing**: Test coverage >80% for business logic
- **Benchmarks**: Include benchmarks for performance-critical paths

---

## Summary

This architecture provides:

1. **Clean separation**: Service layer handles data aggregation, CLI handles presentation
2. **Performance**: Database queries optimized with JOINs and indexes, target <500ms for 100 epics
3. **Flexibility**: Support for JSON output, filtering, and custom timeframes
4. **Maintainability**: Clear interfaces, comprehensive error handling, testable design
5. **Extensibility**: Easy to add caching, new output formats, or additional metrics

The implementation leverages existing repositories and database schema without requiring changes to core infrastructure.

---

## Appendix: Quick Reference

### Command Usage

```bash
shark status                       # Full dashboard
shark status --json               # JSON output
shark status --epic=E01           # Single epic
shark status --recent=7d          # 7-day completions
shark status --no-color           # Plain text
shark status --epic=E01 --json    # Combined options
```

### File Locations

- Service: `/internal/status/status.go`
- Models: `/internal/status/models.go`
- Command: `/internal/cli/commands/status.go`
- Tests: `/internal/status/status_test.go`, `/internal/cli/commands/status_test.go`

### Performance Targets

| Operation | Target | Acceptable |
|-----------|--------|------------|
| Query execution | <200ms | <300ms |
| Output formatting | <100ms | <150ms |
| Total dashboard | <500ms | <700ms |

### Exit Codes

- 0: Success (include empty projects)
- 1: User error (invalid input)
- 2: System error (database, timeout)
