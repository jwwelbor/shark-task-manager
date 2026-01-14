# Work Session Tracking Design

**Epic**: [E13 Workflow-Aware Task Command System](../epic.md)

**Last Updated**: 2026-01-11

---

## Overview

Work session tracking records when agents claim and finish tasks, enabling time-in-phase analytics, stale work detection, and orchestrator health monitoring.

**Design Principle**: Simple schema with automatic session management, minimal overhead on commands.

---

## Database Schema

### task_sessions Table

```sql
CREATE TABLE task_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    agent_id TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    outcome TEXT CHECK(outcome IN ('completed', 'rejected', 'blocked', 'abandoned')),
    notes TEXT,
    duration_minutes INTEGER GENERATED ALWAYS AS (
        CASE
            WHEN ended_at IS NOT NULL THEN
                CAST((julianday(ended_at) - julianday(started_at)) * 1440 AS INTEGER)
            ELSE NULL
        END
    ) STORED,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_sessions_task_id ON task_sessions(task_id);
CREATE INDEX idx_task_sessions_agent_id ON task_sessions(agent_id);
CREATE INDEX idx_task_sessions_started_at ON task_sessions(started_at);
CREATE INDEX idx_task_sessions_active ON task_sessions(task_id, ended_at)
    WHERE ended_at IS NULL;
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER | Primary key |
| task_id | INTEGER | Foreign key to tasks.id |
| agent_id | TEXT | Agent identifier (backend, ai-coder, frontend, etc.) |
| started_at | TIMESTAMP | When work began (claim command) |
| ended_at | TIMESTAMP | When work ended (finish/reject/block) or NULL if active |
| outcome | TEXT | How session ended: completed, rejected, blocked, abandoned |
| notes | TEXT | Optional notes from finish/reject |
| duration_minutes | INTEGER | Computed: minutes from start to end (NULL if active) |
| created_at | TIMESTAMP | Record creation time |
| updated_at | TIMESTAMP | Record last update time |

### Session Outcomes

| Outcome | Meaning | Triggered By |
|---------|---------|-------------|
| completed | Phase finished successfully | `shark task finish` |
| rejected | Work rejected, sent back | `shark task reject` |
| blocked | Work stopped due to blocker | `shark task block` |
| abandoned | Session never finished | Manual cleanup / timeout |

---

## WorkSession Repository

### Location

`internal/repository/work_session_repository.go` (existing, enhance)

### API Methods

```go
type WorkSessionRepository struct {
    db *DB
}

func NewWorkSessionRepository(db *DB) *WorkSessionRepository

// Create starts a new work session
func (r *WorkSessionRepository) Create(ctx context.Context, session *WorkSession) error

// GetByID retrieves session by ID
func (r *WorkSessionRepository) GetByID(ctx context.Context, id int64) (*WorkSession, error)

// GetActiveSessionByTaskID finds active (ended_at IS NULL) session for task
// Returns nil if no active session
func (r *WorkSessionRepository) GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*WorkSession, error)

// EndSession marks session as ended with outcome
func (r *WorkSessionRepository) EndSession(ctx context.Context, sessionID int64, outcome SessionOutcome, notes *string) error

// ListForTask returns all sessions for a task (ordered by started_at DESC)
func (r *WorkSessionRepository) ListForTask(ctx context.Context, taskID int64) ([]*WorkSession, error)

// ListForAgent returns all sessions for an agent
func (r *WorkSessionRepository) ListForAgent(ctx context.Context, agentID string, limit int) ([]*WorkSession, error)

// GetActiveSessions returns all active sessions (ended_at IS NULL)
func (r *WorkSessionRepository) GetActiveSessions(ctx context.Context) ([]*WorkSession, error)

// GetStaleActiveSessions returns active sessions older than threshold
func (r *WorkSessionRepository) GetStaleActiveSessions(ctx context.Context, threshold time.Duration) ([]*WorkSession, error)

// GetSessionStats returns aggregate statistics
func (r *WorkSessionRepository) GetSessionStats(ctx context.Context, filters SessionStatsFilters) (*SessionStats, error)
```

### Models

```go
// WorkSession represents a work session
type WorkSession struct {
    ID              int64
    TaskID          int64
    AgentID         string
    StartedAt       time.Time
    EndedAt         *time.Time
    Outcome         *SessionOutcome
    Notes           *string
    DurationMinutes *int
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// SessionOutcome represents how a session ended
type SessionOutcome string

const (
    SessionOutcomeCompleted  SessionOutcome = "completed"
    SessionOutcomeRejected   SessionOutcome = "rejected"
    SessionOutcomeBlocked    SessionOutcome = "blocked"
    SessionOutcomeAbandoned  SessionOutcome = "abandoned"
)

// SessionStats represents aggregate session statistics
type SessionStats struct {
    TotalSessions      int
    AverageDuration    float64  // minutes
    MedianDuration     float64  // minutes
    CompletedCount     int
    RejectedCount      int
    BlockedCount       int
    AbandonedCount     int
    ByAgentType        map[string]*AgentStats
    ByPhase            map[string]*PhaseStats
}

type AgentStats struct {
    SessionCount       int
    AverageDuration    float64
    CompletionRate     float64  // completed / (completed + rejected + blocked)
}

type PhaseStats struct {
    SessionCount       int
    AverageDuration    float64
    RejectionRate      float64  // rejected / total
}
```

---

## Command Integration

### Claim Command (Session Start)

```go
func runTaskClaim(cmd *cobra.Command, args []string) error {
    // ... validate and get task ...

    // Begin transaction
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 1. Update task status
    err = taskRepo.UpdateStatus(ctx, tx, task.ID, targetStatus, agentID)
    if err != nil {
        return err
    }

    // 2. Create work session
    sessionRepo := NewWorkSessionRepository(db)
    session := &WorkSession{
        TaskID:    task.ID,
        AgentID:   agentID,
        StartedAt: time.Now(),
    }
    err = sessionRepo.Create(ctx, session)
    if err != nil {
        return err
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return err
    }

    // Output includes session info
    cli.Success(fmt.Sprintf("Task claimed. Session started at %s", session.StartedAt))
    return nil
}
```

### Finish Command (Session End with Completion)

```go
func runTaskFinish(cmd *cobra.Command, args []string) error {
    // ... validate and get task ...

    notes, _ := cmd.Flags().GetString("notes")

    // Begin transaction
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 1. Update task status
    err = taskRepo.UpdateStatus(ctx, tx, task.ID, targetStatus, nil)
    if err != nil {
        return err
    }

    // 2. End active work session
    sessionRepo := NewWorkSessionRepository(db)
    activeSession, err := sessionRepo.GetActiveSessionByTaskID(ctx, task.ID)
    if err != nil {
        return err
    }

    if activeSession != nil {
        var notesPtr *string
        if notes != "" {
            notesPtr = &notes
        }
        err = sessionRepo.EndSession(ctx, activeSession.ID, SessionOutcomeCompleted, notesPtr)
        if err != nil {
            return err
        }

        // Calculate duration for output
        duration := time.Since(activeSession.StartedAt)
        cli.Info(fmt.Sprintf("Work session: %s", formatDuration(duration)))
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return err
    }

    return nil
}

func formatDuration(d time.Duration) string {
    hours := int(d.Hours())
    minutes := int(d.Minutes()) % 60
    if hours > 0 {
        return fmt.Sprintf("%dh %dm", hours, minutes)
    }
    return fmt.Sprintf("%dm", minutes)
}
```

### Reject Command (Session End with Rejection)

```go
func runTaskReject(cmd *cobra.Command, args []string) error {
    // ... validate and get task ...

    reason, _ := cmd.Flags().GetString("reason")

    // Begin transaction
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 1. Update task status
    err = taskRepo.UpdateStatus(ctx, tx, task.ID, targetStatus, nil)
    if err != nil {
        return err
    }

    // 2. End active work session with rejection outcome
    sessionRepo := NewWorkSessionRepository(db)
    activeSession, err := sessionRepo.GetActiveSessionByTaskID(ctx, task.ID)
    if err != nil {
        return err
    }

    if activeSession != nil {
        err = sessionRepo.EndSession(ctx, activeSession.ID, SessionOutcomeRejected, &reason)
        if err != nil {
            return err
        }

        duration := time.Since(activeSession.StartedAt)
        cli.Info(fmt.Sprintf("Work session: %s (rejected)", formatDuration(duration)))
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return err
    }

    return nil
}
```

---

## Session Queries

### shark task sessions (New Command)

```bash
# Show all sessions for a task
shark task sessions T-E07-F20-001

# Show with JSON output
shark task sessions T-E07-F20-001 --json
```

**Output (Human-Readable)**:
```
Task T-E07-F20-001 Work Sessions:

Session 1:
  Agent: backend
  Started: 2026-01-11 10:00:00
  Ended: 2026-01-11 12:30:00
  Duration: 2h 30m
  Outcome: completed
  Notes: API implementation complete

Session 2:
  Agent: backend
  Started: 2026-01-11 09:00:00
  Ended: 2026-01-11 09:45:00
  Duration: 45m
  Outcome: rejected
  Notes: Acceptance criteria incomplete

Total Time: 3h 15m
Completion Rate: 50% (1/2 sessions completed)
```

**Output (JSON)**:
```json
{
  "task_key": "T-E07-F20-001",
  "sessions": [
    {
      "id": 2,
      "agent_id": "backend",
      "started_at": "2026-01-11T10:00:00Z",
      "ended_at": "2026-01-11T12:30:00Z",
      "duration_minutes": 150,
      "outcome": "completed",
      "notes": "API implementation complete"
    },
    {
      "id": 1,
      "agent_id": "backend",
      "started_at": "2026-01-11T09:00:00Z",
      "ended_at": "2026-01-11T09:45:00Z",
      "duration_minutes": 45,
      "outcome": "rejected",
      "notes": "Acceptance criteria incomplete"
    }
  ],
  "total_duration_minutes": 195,
  "completion_rate": 0.5
}
```

### Active Session Detection

```go
// Check if task has active session (for claim validation)
func hasActiveSession(ctx context.Context, taskID int64) (bool, *WorkSession, error) {
    sessionRepo := NewWorkSessionRepository(db)
    session, err := sessionRepo.GetActiveSessionByTaskID(ctx, taskID)
    if err != nil {
        return false, nil, err
    }
    return session != nil, session, nil
}
```

### Stale Session Detection (Orchestrator)

```bash
# Find sessions active for > 4 hours (potential issues)
shark task sessions --active --stale=4h --json
```

```go
func detectStaleSessions() {
    sessionRepo := NewWorkSessionRepository(db)
    threshold := 4 * time.Hour
    staleSessions, err := sessionRepo.GetStaleActiveSessions(ctx, threshold)
    if err != nil {
        log.Error("Failed to detect stale sessions: %v", err)
        return
    }

    for _, session := range staleSessions {
        log.Warn("Stale session detected: task=%d, agent=%s, started=%s",
            session.TaskID, session.AgentID, session.StartedAt)

        // Orchestrator action: mark as abandoned or reassign
        err = sessionRepo.EndSession(ctx, session.ID, SessionOutcomeAbandoned, nil)
        if err != nil {
            log.Error("Failed to abandon stale session: %v", err)
        }
    }
}
```

---

## Analytics Queries

### Phase Duration Analytics

```sql
-- Average time spent in each phase
SELECT
    sm.phase,
    COUNT(s.id) AS session_count,
    ROUND(AVG(s.duration_minutes), 1) AS avg_duration_minutes,
    ROUND(AVG(CASE WHEN s.outcome = 'completed' THEN s.duration_minutes END), 1) AS avg_completed_duration,
    ROUND(SUM(CASE WHEN s.outcome = 'rejected' THEN 1 ELSE 0 END) * 100.0 / COUNT(s.id), 1) AS rejection_rate_pct
FROM task_sessions s
JOIN tasks t ON s.task_id = t.id
JOIN (
    -- Derive phase from task status at session start
    SELECT 'in_development' AS status, 'development' AS phase
    UNION SELECT 'in_code_review', 'review'
    UNION SELECT 'in_qa', 'qa'
    -- ... etc
) sm ON t.status = sm.status
WHERE s.ended_at IS NOT NULL
GROUP BY sm.phase
ORDER BY sm.phase;
```

**Result**:
```
phase       | session_count | avg_duration | avg_completed | rejection_rate
------------|---------------|--------------|---------------|---------------
development | 45            | 180.5        | 195.2         | 8.9%
review      | 42            | 45.3         | 40.1          | 11.9%
qa          | 40            | 60.2         | 55.8          | 12.5%
approval    | 38            | 15.0         | 15.0          | 0.0%
```

### Agent Performance

```sql
-- Agent session statistics
SELECT
    s.agent_id,
    COUNT(s.id) AS total_sessions,
    ROUND(AVG(s.duration_minutes), 1) AS avg_duration,
    SUM(CASE WHEN s.outcome = 'completed' THEN 1 ELSE 0 END) AS completed_count,
    SUM(CASE WHEN s.outcome = 'rejected' THEN 1 ELSE 0 END) AS rejected_count,
    ROUND(SUM(CASE WHEN s.outcome = 'completed' THEN 1 ELSE 0 END) * 100.0 / COUNT(s.id), 1) AS completion_rate_pct
FROM task_sessions s
WHERE s.ended_at IS NOT NULL
GROUP BY s.agent_id
ORDER BY total_sessions DESC;
```

---

## Migration

### Add task_sessions Table

```sql
-- Migration: Add task_sessions table
-- This is a new table, safe to add without data migration

CREATE TABLE IF NOT EXISTS task_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    agent_id TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    outcome TEXT CHECK(outcome IN ('completed', 'rejected', 'blocked', 'abandoned')),
    notes TEXT,
    duration_minutes INTEGER GENERATED ALWAYS AS (
        CASE
            WHEN ended_at IS NOT NULL THEN
                CAST((julianday(ended_at) - julianday(started_at)) * 1440 AS INTEGER)
            ELSE NULL
        END
    ) STORED,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_sessions_task_id ON task_sessions(task_id);
CREATE INDEX idx_task_sessions_agent_id ON task_sessions(agent_id);
CREATE INDEX idx_task_sessions_started_at ON task_sessions(started_at);
CREATE INDEX idx_task_sessions_active ON task_sessions(task_id, ended_at) WHERE ended_at IS NULL;

-- Add trigger for updated_at
CREATE TRIGGER update_task_sessions_updated_at
AFTER UPDATE ON task_sessions
FOR EACH ROW
BEGIN
    UPDATE task_sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### Backward Compatibility

**Existing databases** (no task_sessions table):
- Commands work without sessions (sessions are optional)
- Session creation fails silently (log warning)
- Session queries return empty results

**After migration**:
- All new claims create sessions automatically
- Existing tasks retroactively get sessions on next claim

---

## Testing

### Unit Tests

```go
func TestWorkSessionRepository_Create(t *testing.T) {
    db := test.GetTestDB()
    defer db.Close()

    repo := NewWorkSessionRepository(db)
    session := &WorkSession{
        TaskID:    1,
        AgentID:   "backend",
        StartedAt: time.Now(),
    }

    err := repo.Create(context.Background(), session)
    assert.NoError(t, err)
    assert.NotZero(t, session.ID)
}

func TestWorkSessionRepository_EndSession(t *testing.T) {
    // ... create session ...

    notes := "Work completed"
    err := repo.EndSession(ctx, session.ID, SessionOutcomeCompleted, &notes)
    assert.NoError(t, err)

    // Verify ended_at and outcome set
    retrieved, _ := repo.GetByID(ctx, session.ID)
    assert.NotNil(t, retrieved.EndedAt)
    assert.Equal(t, SessionOutcomeCompleted, *retrieved.Outcome)
    assert.NotNil(t, retrieved.DurationMinutes)
}
```

### Integration Tests

```go
func TestFullSessionCycle(t *testing.T) {
    // 1. Create task
    task := createTestTask(t)

    // 2. Claim task (starts session)
    claimTask(t, task.Key, "backend")
    sessions, _ := sessionRepo.ListForTask(ctx, task.ID)
    assert.Len(t, sessions, 1)
    assert.Nil(t, sessions[0].EndedAt)

    // 3. Finish task (ends session)
    finishTask(t, task.Key, "Work done")
    sessions, _ = sessionRepo.ListForTask(ctx, task.ID)
    assert.Len(t, sessions, 1)
    assert.NotNil(t, sessions[0].EndedAt)
    assert.Equal(t, SessionOutcomeCompleted, *sessions[0].Outcome)
}
```

---

## Performance

### Query Optimization

**Indexes**:
- `idx_task_sessions_task_id` - Fast lookup by task
- `idx_task_sessions_agent_id` - Agent performance queries
- `idx_task_sessions_active` - Quick active session detection

**Generated Column**:
- `duration_minutes` computed on INSERT/UPDATE
- No runtime calculation needed for queries

**Expected Performance**:
- Create session: < 5ms
- End session: < 10ms
- Get active session: < 2ms (indexed query)
- List sessions for task: < 5ms

---

## References

- [System Architecture](./system-architecture.md)
- [Command Specifications](./command-specifications.md)
- [Epic Requirements](../requirements.md) - REQ-F-013
