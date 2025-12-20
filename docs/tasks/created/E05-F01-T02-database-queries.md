# Task: E05-F01-T02 - Implement Optimized Database Queries for Dashboard

**Feature**: E05-F01 Status Dashboard & Reporting
**Epic**: E05 Task Management CLI Capabilities
**Task Key**: E05-F01-T02

## Description

Implement efficient database queries that aggregate project data for the status dashboard. This task builds the query layer that fetches data in minimal round trips, using JOINs, aggregations, and indexes to achieve <200ms performance target for 100 epics.

This task implements the `GetDashboard` method and 5 supporting query methods:
- `getProjectSummary()` - Overall project metrics
- `getEpics()` - Epic breakdown with progress
- `getActiveTasks()` - In-progress tasks grouped by agent
- `getBlockedTasks()` - Blocked tasks with reasons
- `getRecentCompletions()` - Recently completed tasks with timeframe filtering

**Why This Matters**: Query optimization is critical for performance. Well-designed queries using indexes can be 10-100x faster than naive implementations. This task ensures the dashboard completes in <500ms even for large projects.

## What You'll Build

Complete implementation of StatusService methods in `internal/status/status.go`:

```
GetDashboard(ctx context.Context, request *StatusRequest) (*StatusDashboard, error)
├── getProjectSummary(ctx, epicKey) (*ProjectSummary, error)
├── getEpics(ctx, epicKey) ([]*EpicSummary, error)
├── getActiveTasks(ctx, epicKey) (map[string][]*TaskInfo, error)
├── getBlockedTasks(ctx, epicKey) ([]*BlockedTaskInfo, error)
└── getRecentCompletions(ctx, epicKey, window) ([]*CompletionInfo, error)

Plus helper methods:
├── validateEpicKey(ctx, key) error
├── determineEpicHealth(progress, blockedCount) string
├── groupTasksByAgent(tasks) map[string][]*TaskInfo
├── timeframeToSQLiteModifier(timeframe) string
└── calculateRelativeTime(t time.Time) string
```

## Success Criteria

- [x] `GetDashboard()` returns correct StatusDashboard with all 5 sections populated
- [x] `getProjectSummary()` uses single query (or CTE) to aggregate all counts
- [x] `getEpics()` returns one row per epic with calculated progress via LEFT JOINs
- [x] `getActiveTasks()` returns tasks grouped by agent_type in canonical order
- [x] `getBlockedTasks()` returns blocked tasks sorted by priority then timestamp
- [x] `getRecentCompletions()` filters by timeframe and returns up to 100 tasks
- [x] Epic filtering works: when epicKey specified, all queries filter to that epic's data
- [x] Health status calculation: healthy (≥75% and no blockers), warning, critical
- [x] Relative time formatting: "2 hours ago", "5 minutes ago", "1 day ago"
- [x] Queries achieve <200ms performance target for 100 epics
- [x] No N+1 query problems (use JOINs, not sequential queries)
- [x] All queries use parameterized statements (SQL injection safe)
- [x] Handles edge cases: empty epics, no tasks, NULL agent_type
- [x] All tests pass: `go test ./internal/status -v`

## Implementation Notes

### Critical Query Patterns

#### 1. Project Summary Query

Use aggregations in database, not application code:

```sql
SELECT
    COUNT(*) as total_epics,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_epics
FROM epics
WHERE archived = false;
```

**Why**: COUNT/SUM in database is 10-100x faster than loading all rows and counting in Go.

#### 2. Epic Progress Calculation

Use LEFT JOINs to include epics with no features/tasks:

```sql
SELECT
    e.id, e.key, e.title, e.priority,
    COUNT(DISTINCT t.id) as total_tasks,
    SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END) as completed_tasks
FROM epics e
LEFT JOIN features f ON e.id = f.epic_id
LEFT JOIN tasks t ON f.id = t.feature_id
WHERE e.archived = false
GROUP BY e.id, e.key, e.title, e.priority
ORDER BY e.priority DESC, e.key ASC;
```

**Why**: Single query returns all epics with computed progress. No N+1.

#### 3. Task Grouping by Agent

Query with consistent ordering, let application group by agent_type:

```sql
SELECT t.key, t.title, t.agent_type, e.key as epic_key, f.key as feature_key
FROM tasks t
JOIN features f ON t.feature_id = f.id
JOIN epics e ON f.epic_id = e.id
WHERE t.status = 'in_progress' AND e.archived = false
ORDER BY t.agent_type ASC NULLS LAST, t.key ASC;
```

**Implementation Note**: After scanning results, group into map[string][]*TaskInfo by agent_type. Handle NULL as "unassigned".

#### 4. Timeframe Filtering

Use SQLite's `datetime()` function with modifiers:

```sql
SELECT t.key, t.title, t.completed_at
FROM tasks t
WHERE t.status = 'completed'
  AND t.completed_at >= datetime('now', ?)
ORDER BY t.completed_at DESC
LIMIT 100;
```

**Implementation Note**: Parse timeframe string to SQLite modifier:
- "24h" → "-24 hours"
- "7d" → "-7 days"
- "30d" → "-30 days"

### Performance Optimization Checklist

- [ ] Verify indexes exist on filtered/joined columns:
  ```sql
  CREATE INDEX IF NOT EXISTS idx_features_epic_id ON features(epic_id);
  CREATE INDEX IF NOT EXISTS idx_tasks_feature_id ON tasks(feature_id);
  CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
  CREATE INDEX IF NOT EXISTS idx_tasks_completed_at ON tasks(completed_at);
  CREATE INDEX IF NOT EXISTS idx_tasks_agent_type ON tasks(agent_type);
  ```

- [ ] Use `EXPLAIN QUERY PLAN` to verify index usage:
  ```bash
  sqlite3 shark-tasks.db "EXPLAIN QUERY PLAN SELECT ..."
  # Should show SEARCH TABLE with index, not SCAN TABLE
  ```

- [ ] Limit result sets to prevent memory bloat:
  - Recent completions: LIMIT 100
  - Active tasks: LIMIT 1000

- [ ] Use parameterized queries to avoid parsing overhead

### Error Handling

All query methods should:
1. Wrap errors with context: `fmt.Errorf("get epics for dashboard: %w", err)`
2. Handle context cancellation: `if ctx.Err() != nil { return nil, ctx.Err() }`
3. Return zero values (not nil) for empty sections
4. Handle NULL/missing values gracefully

Example:
```go
func (s *statusService) getActiveTasks(ctx context.Context, epicKey string) (map[string][]*TaskInfo, error) {
    if ctx.Err() != nil {
        return nil, ctx.Err()
    }

    rows, err := s.db.QueryContext(ctx, queryString, args...)
    if err != nil {
        return nil, fmt.Errorf("query active tasks: %w", err)
    }
    defer rows.Close()

    // Group tasks by agent_type
    groups := make(map[string][]*TaskInfo)
    for rows.Next() {
        var task TaskInfo
        var agentType sql.NullString

        if err := rows.Scan(&task.Key, &task.Title, &agentType, ...); err != nil {
            return nil, fmt.Errorf("scan task row: %w", err)
        }

        agent := "unassigned"
        if agentType.Valid {
            agent = agentType.String
        }

        groups[agent] = append(groups[agent], &task)
    }

    return groups, rows.Err()
}
```

### Health Status Logic

```go
func (s *statusService) determineEpicHealth(progress float64, blockedCount int) string {
    // Critical: <25% progress OR >3 blocked tasks
    if progress < 25.0 || blockedCount > 3 {
        return "critical"
    }

    // Warning: 25-74% progress OR 1-3 blocked tasks
    if progress < 75.0 || blockedCount > 0 {
        return "warning"
    }

    // Healthy: ≥75% progress AND no blocked tasks
    return "healthy"
}
```

### Relative Time Formatting

```go
func (s *statusService) calculateRelativeTime(t time.Time) string {
    now := time.Now()
    diff := now.Sub(t)

    switch {
    case diff < time.Minute:
        return fmt.Sprintf("%d seconds ago", int(diff.Seconds()))
    case diff < time.Hour:
        minutes := int(diff.Minutes())
        return fmt.Sprintf("%d minutes ago", minutes)
    case diff < 24*time.Hour:
        hours := int(diff.Hours())
        if hours == 1 {
            return "1 hour ago"
        }
        return fmt.Sprintf("%d hours ago", hours)
    case diff < 7*24*time.Hour:
        days := int(diff.Hours() / 24)
        if days == 1 {
            return "1 day ago"
        }
        return fmt.Sprintf("%d days ago", days)
    default:
        return t.Format("2006-01-02")
    }
}
```

## Dependencies

- Database: `*db.DB` (injected)
- Repositories: EpicRepository, FeatureRepository, TaskRepository (dependency injection pattern, but may query DB directly)
- Models: Epic, Feature, Task from `internal/models`
- Go standard library: context, database/sql, fmt, time

## Related Tasks

- **E05-F01-T01**: Service Data Structures - Defines output types
- **E05-F01-T03**: CLI Command - Calls GetDashboard and handles output
- **E05-F01-T04**: Output Formatting - Receives populated StatusDashboard

## Acceptance Criteria

**Functional**:
- [ ] GetDashboard returns properly populated StatusDashboard
- [ ] Project summary correctly counts epics/features/tasks
- [ ] Epic progress calculated as: (completed tasks / total tasks) * 100
- [ ] Health status correctly identifies critical/warning/healthy epics
- [ ] Active tasks grouped by agent_type in canonical order
- [ ] Blocked tasks sorted by priority DESC, then blocked_at DESC
- [ ] Recent completions filtered by timeframe window
- [ ] Epic filtering: all queries respect --epic flag
- [ ] Edge cases handled: empty epics, NULL agent_type, zero progress
- [ ] Relative time formats correctly: "2 hours ago", "1 day ago"

**Performance**:
- [ ] All queries use JOINs, not N+1 sequential queries
- [ ] Queries complete in <200ms for 100 epics
- [ ] Memory usage <50MB for large datasets
- [ ] No full table scans (verify with EXPLAIN QUERY PLAN)
- [ ] Index usage confirmed for filtered columns

**Code Quality**:
- [ ] All queries parameterized (no string concatenation)
- [ ] Proper error wrapping with context
- [ ] Context cancellation handling
- [ ] NULL value handling with sql.NullString, sql.NullTime
- [ ] Consistent error messages

**Testing**:
- [ ] Unit test: GetDashboard with empty database
- [ ] Unit test: GetDashboard with 3 epics, 127 tasks
- [ ] Unit test: Epic filtering works
- [ ] Unit test: Health status calculation
- [ ] Unit test: Relative time formatting
- [ ] Benchmark: <200ms for query execution (100 epics)
- [ ] All tests pass: `go test ./internal/status -v`

## Verification Steps

```bash
# Test query correctness
go test ./internal/status -run TestGetDashboard -v

# Benchmark query performance
go test -bench=GetDashboard -benchmem ./internal/status
# Should show <200ms for large project

# Verify no N+1 with query counting
# (Can add query counter to db wrapper if available)

# Check index usage
sqlite3 shark-tasks.db
sqlite> EXPLAIN QUERY PLAN SELECT e.key, COUNT(DISTINCT t.id) FROM epics e LEFT JOIN features f ON e.id = f.epic_id LEFT JOIN tasks t ON f.id = t.feature_id WHERE e.archived = false GROUP BY e.id;
# Should show SEARCH with indexes, not SCAN
```

## Implementation Checklist

See Phase 2 in implementation-checklist.md:
- [ ] Task 2.1: Project Summary Query
- [ ] Task 2.2: Epic Breakdown Query with health calculation
- [ ] Task 2.3: Active Tasks Query with grouping
- [ ] Task 2.4: Blocked Tasks Query
- [ ] Task 2.5: Recent Completions Query
- [ ] Task 2.6: Epic Filtering
- [ ] Task 2.7: Query Performance Tests & Benchmarks
