---
feature_key: E13-F04-session-tracking-and-analytics
epic_key: E13
title: Session Tracking and Analytics
description: Add task_sessions database table. Integrate session tracking with claim/finish/reject commands. Provide phase duration queries for AI orchestrator analytics.
---

# Session Tracking and Analytics

**Feature Key**: E13-F04-session-tracking-and-analytics

---

## Epic

- **Epic PRD**: [E13 Workflow-Aware Task Command System](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md) - REQ-F-013
- **Epic Architecture**: [Session Tracking Architecture](../../architecture/session-tracking.md)

---

## Goal

### Problem

AI orchestrators and project managers lack visibility into how long tasks spend in each workflow phase. When tasks are claimed by agents, there is no record of when work started, when it finished, or how long it took. This makes it impossible to:
- Detect stale work sessions (agents that claimed but never finished)
- Calculate realistic time estimates for sprint planning
- Identify bottlenecks in the workflow (which phases take longest)
- Monitor agent performance and completion rates
- Provide meaningful analytics for continuous improvement

### Solution

Implement automatic work session tracking that records timestamps when tasks are claimed and finished. The `task_sessions` table tracks each work session with start/end times, agent ID, and outcome (completed, rejected, blocked, abandoned). Session tracking is integrated seamlessly into existing workflow commands (`claim`, `finish`, `reject`) without requiring additional manual steps. Duration calculations are performed automatically using SQLite generated columns.

### Impact

**Expected Outcomes**:
- Enable AI orchestrators to detect and reassign stale work (sessions active > 4 hours)
- Provide project managers with accurate phase duration data for sprint planning
- Identify workflow bottlenecks through average time-in-phase analytics
- Track agent performance with completion rates and rejection rates
- Support data-driven workflow optimization decisions

**Measurable Metrics**:
- Session data captured for 100% of task transitions after deployment
- Stale session detection reduces abandoned work by 80%
- Phase duration analytics enable 20% more accurate sprint planning
- Agent performance data available for all active agents

---

## User Personas

### Persona 1: Atlas (AI Orchestrator Agent)

**Profile**:
- **Role/Title**: AI-powered task orchestrator that assigns work to specialized agents
- **Experience Level**: Autonomous system, requires no human intervention
- **Key Characteristics**:
  - Polls for available tasks every 30 seconds
  - Claims tasks for specific agent types (backend, frontend, qa, etc.)
  - Needs to detect when agents fail to complete work
  - Operates continuously across multiple features/epics

**Goals Related to This Feature**:
1. Detect stale work sessions where agents claimed but never finished
2. Monitor agent availability and workload distribution
3. Track task progress through workflow phases automatically
4. Identify when to reassign work or escalate issues

**Pain Points This Feature Addresses**:
- Cannot detect when an agent has abandoned work (claimed but never finished)
- No visibility into how long tasks have been "in progress"
- Unable to calculate realistic time estimates for remaining work
- Cannot identify which agents are productive vs. stalled

**Success Looks Like**:
Atlas can query active sessions, identify any that are stale (>4 hours), automatically mark them as abandoned, and reassign the work to another available agent without human intervention.

### Persona 2: Sarah (Project Manager)

**Profile**:
- **Role/Title**: Technical project manager at software development company
- **Experience Level**: 5+ years managing development teams, comfortable with CLI tools
- **Key Characteristics**:
  - Responsible for sprint planning and velocity tracking
  - Needs data to estimate task duration and team capacity
  - Reviews workflow efficiency and identifies bottlenecks
  - Reports on team performance to stakeholders

**Goals Related to This Feature**:
1. Understand how long tasks actually take in each workflow phase
2. Use historical data for more accurate sprint planning
3. Identify workflow phases that consistently take longer than expected
4. Monitor team productivity and completion rates

**Pain Points This Feature Addresses**:
- Estimates are guesses without historical time-in-phase data
- Cannot identify bottlenecks (which phase causes delays)
- No way to measure if workflow changes improved efficiency
- Difficult to explain velocity variations to stakeholders

**Success Looks Like**:
Sarah runs `shark task sessions --stats` and sees that code review averages 45 minutes but QA averages 60 minutes with 12% rejection rate. She uses this data to propose adding another QA agent and tracks improvement over next sprint.

---

## User Stories

### Must-Have Stories

**Story 1**: As Atlas (AI orchestrator), I want work sessions created automatically when I claim tasks so that I can track when work started without additional commands.

**Acceptance Criteria**:
- [ ] `shark task claim T-E07-F20-001 --agent=backend` creates a work session record
- [ ] Session records task_id, agent_id, and started_at timestamp
- [ ] Session is created in same database transaction as status update (atomic)
- [ ] Session creation failure rolls back the entire claim operation
- [ ] Session ID is returned in JSON output for reference

**Story 2**: As Dev (developer agent), when I finish a task, the work session is automatically closed so duration is calculated accurately.

**Acceptance Criteria**:
- [ ] `shark task finish T-E07-F20-001` closes the active work session
- [ ] Session ended_at is set to current timestamp
- [ ] Session outcome is set to "completed"
- [ ] Optional notes from `--notes` flag are stored in session
- [ ] Duration is calculated automatically using generated column
- [ ] Session closure happens in same transaction as status update

**Story 3**: As Atlas, when I reject a task, the work session is closed as rejected so I can track rejection rates by agent.

**Acceptance Criteria**:
- [ ] `shark task reject T-E07-F20-001 --reason="..."` closes active session
- [ ] Session outcome is set to "rejected"
- [ ] Rejection reason is stored in session notes
- [ ] Session closure is atomic with status update
- [ ] Rejection rate can be calculated from session outcome data

**Story 4**: As Sarah (PM), I want to view all work sessions for a task so I can see how many times it was rejected or reworked.

**Acceptance Criteria**:
- [ ] `shark task sessions T-E07-F20-001` shows all sessions for task
- [ ] Output includes agent, start time, end time, duration, outcome
- [ ] Sessions are ordered by started_at (most recent first)
- [ ] Total time across all sessions is calculated
- [ ] Completion rate (completed / total sessions) is displayed
- [ ] JSON output available with `--json` flag

**Story 5**: As Atlas, I want to query for stale active sessions so I can detect abandoned work and reassign tasks.

**Acceptance Criteria**:
- [ ] `shark task sessions --active --stale=4h` shows sessions active > 4 hours
- [ ] Query uses indexed lookup on ended_at IS NULL and started_at
- [ ] Results include task key, agent ID, and time since start
- [ ] Can be filtered by agent type: `--agent=backend`
- [ ] JSON output enables programmatic processing
- [ ] Query executes in < 1 second for database with 10,000 tasks

---

### Should-Have Stories

**Story 6**: As Sarah, I want phase duration analytics so I can identify workflow bottlenecks.

**Acceptance Criteria**:
- [ ] `shark analytics phase-duration` shows average time per phase
- [ ] Output includes session count, avg duration, rejection rate per phase
- [ ] Can filter by date range: `--since="2026-01-01"`
- [ ] Can filter by epic: `--epic=E07`
- [ ] JSON output includes median and percentiles (p50, p90, p99)

**Story 7**: As Sarah, I want agent performance analytics so I can identify high-performing and struggling agents.

**Acceptance Criteria**:
- [ ] `shark analytics agent-performance` shows stats by agent
- [ ] Output includes total sessions, avg duration, completion rate
- [ ] Completion rate calculated as completed / (completed + rejected + blocked)
- [ ] Can filter by agent type or specific agent ID
- [ ] Results sorted by total sessions (most active first)

---

### Could-Have Stories

**Story 8**: As Sarah, I want to export session data to CSV so I can analyze in Excel or BI tools.

**Acceptance Criteria**:
- [ ] `shark task sessions --export=csv` generates CSV file
- [ ] CSV includes all session fields (task_key, agent_id, timestamps, outcome)
- [ ] Can filter before export (e.g., by date range or agent)
- [ ] File name includes timestamp: `sessions-2026-01-11T10-30-00.csv`

---

### Edge Case & Error Stories

**Error Story 1**: As Atlas, when a task has an active session and I try to claim it again, I want a clear error so I know the task is already assigned.

**Acceptance Criteria**:
- [ ] `shark task claim T-E07-F20-001` on already-claimed task fails
- [ ] Error message: "Task already claimed by agent 'backend' since [timestamp]"
- [ ] Suggests: "Use 'shark task finish' or 'shark task reject' to complete work"
- [ ] JSON error includes session details (agent_id, started_at)

**Error Story 2**: As a developer, when a session exists but task is in wrong status, the system handles gracefully.

**Acceptance Criteria**:
- [ ] If active session exists but task status is `ready_for_*`, close session as abandoned
- [ ] If `finish` called but no active session exists, command still succeeds (backward compat)
- [ ] Warning logged: "No active session found, skipping session closure"
- [ ] Status update still happens normally

---

## Requirements

### Functional Requirements

**Category: Session Lifecycle Management**

1. **REQ-F-013-001**: Session Creation on Claim
   - **Description**: Automatically create work session record when task is claimed
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Session created in same transaction as status update
     - [ ] Session includes task_id, agent_id, started_at timestamp
     - [ ] Failure to create session rolls back entire claim operation
     - [ ] Only one active session allowed per task at a time

2. **REQ-F-013-002**: Session Closure on Finish
   - **Description**: Automatically close session when task phase is finished
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Sets ended_at timestamp to current time
     - [ ] Sets outcome to "completed"
     - [ ] Stores optional notes from `--notes` flag
     - [ ] Duration calculated automatically via generated column
     - [ ] Closure happens atomically with status update

3. **REQ-F-013-003**: Session Closure on Reject
   - **Description**: Close session with rejected outcome when work is rejected
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Sets outcome to "rejected"
     - [ ] Stores rejection reason in notes field
     - [ ] Ended_at timestamp set to current time
     - [ ] Atomic with status transition

4. **REQ-F-013-004**: Session Pause on Block
   - **Description**: Close session with blocked outcome when task is blocked
   - **User Story**: Links to Story 3 (variant)
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Sets outcome to "blocked"
     - [ ] Stores block reason in notes
     - [ ] Future: Resume session on unblock (not in initial version)

**Category: Session Queries**

5. **REQ-F-013-005**: List Task Sessions
   - **Description**: View all work sessions for a specific task
   - **User Story**: Links to Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task sessions <task-key>` command implemented
     - [ ] Shows agent, timestamps, duration, outcome per session
     - [ ] Ordered by started_at DESC (most recent first)
     - [ ] Calculates total time and completion rate
     - [ ] Supports JSON output with `--json` flag

6. **REQ-F-013-006**: Detect Stale Sessions
   - **Description**: Query for active sessions exceeding time threshold
   - **User Story**: Links to Story 5
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task sessions --active --stale=<duration>` command
     - [ ] Returns sessions where ended_at IS NULL and age > threshold
     - [ ] Includes task key, agent ID, time since start
     - [ ] Can filter by agent type or epic
     - [ ] Query executes in < 1 second for 10K tasks

7. **REQ-F-013-007**: Phase Duration Analytics
   - **Description**: Aggregate statistics on time spent in each workflow phase
   - **User Story**: Links to Story 6
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Shows session count, avg duration, rejection rate per phase
     - [ ] Supports date range filtering
     - [ ] Includes median and percentile data (p50, p90, p99)
     - [ ] JSON output for programmatic consumption

8. **REQ-F-013-008**: Agent Performance Analytics
   - **Description**: Statistics on agent productivity and completion rates
   - **User Story**: Links to Story 7
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Shows total sessions, avg duration, completion rate by agent
     - [ ] Completion rate = completed / (completed + rejected + blocked)
     - [ ] Sorted by activity level (most active first)
     - [ ] Can filter by agent type or specific agent ID

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-013-001**: Session Creation Overhead
   - **Description**: Session creation must not significantly slow claim command
   - **Measurement**: Measure claim command execution time before/after
   - **Target**: Session creation adds < 10ms to claim operation (< 5% overhead)
   - **Justification**: AI orchestrator polls every 30s; slow claims delay task assignment

2. **REQ-NF-013-002**: Stale Session Query Performance
   - **Description**: Stale session detection must scale to large task databases
   - **Measurement**: Query execution time with varying task counts
   - **Target**: < 1 second for 10,000 tasks, < 5 seconds for 100,000 tasks
   - **Justification**: Orchestrator runs stale detection every 5 minutes

3. **REQ-NF-013-003**: Analytics Query Performance
   - **Description**: Phase and agent analytics must return in reasonable time
   - **Measurement**: Query execution time for aggregate statistics
   - **Target**: < 3 seconds for database with 50,000 sessions
   - **Justification**: PM reviews analytics during planning sessions

**Data Integrity**

4. **REQ-NF-013-004**: Atomic Session Operations
   - **Description**: Session changes must be atomic with status updates
   - **Implementation**: Wrap session + status update in database transaction
   - **Testing**: Simulate failures during operation, verify rollback
   - **Risk Mitigation**: Prevents orphaned sessions or status mismatches

5. **REQ-NF-013-005**: Duration Accuracy
   - **Description**: Duration calculations must be accurate to the minute
   - **Implementation**: Use SQLite julianday() function for timestamp math
   - **Validation**: Generate column ensures consistent calculation
   - **Risk Mitigation**: Avoids rounding errors in application code

**Backward Compatibility**

6. **REQ-NF-013-006**: Graceful Degradation
   - **Description**: Commands work on databases without task_sessions table
   - **Implementation**: Check if table exists before session operations
   - **Behavior**: Log warning and skip session tracking if table missing
   - **Justification**: Supports gradual migration without forcing schema upgrade

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Complete Task Session Lifecycle**
- **Given** a task T-E07-F20-001 in status "ready_for_development"
- **When** Atlas runs `shark task claim T-E07-F20-001 --agent=backend`
- **Then** task status changes to "in_development"
- **And** a work session is created with agent_id="backend" and started_at=now
- **When** developer runs `shark task finish T-E07-F20-001 --notes="API complete"`
- **Then** task status changes to "ready_for_code_review"
- **And** session is closed with outcome="completed", notes="API complete", ended_at=now
- **And** duration_minutes is calculated automatically

**Scenario 2: Rejected Task with Session Tracking**
- **Given** a task T-E07-F20-002 claimed by agent "backend"
- **When** code reviewer runs `shark task reject T-E07-F20-002 --reason="Tests missing"`
- **Then** task status changes to appropriate rejection status
- **And** session is closed with outcome="rejected", notes="Tests missing"
- **And** duration_minutes reflects actual work time
- **When** another agent later claims and completes the task
- **Then** a second session is created and closed
- **And** `shark task sessions T-E07-F20-002` shows both sessions

**Scenario 3: Stale Session Detection**
- **Given** 5 tasks claimed by various agents
- **And** 2 tasks were claimed > 4 hours ago and never finished
- **When** orchestrator runs `shark task sessions --active --stale=4h --json`
- **Then** exactly 2 sessions are returned
- **And** each includes task_key, agent_id, and time_since_start
- **When** orchestrator marks sessions as abandoned
- **Then** ended_at is set and outcome="abandoned"

**Scenario 4: Phase Duration Analytics**
- **Given** 100 completed tasks across various workflow phases
- **When** PM runs `shark analytics phase-duration --since="2026-01-01" --json`
- **Then** output shows statistics for each phase (development, review, qa, approval)
- **And** includes session_count, avg_duration_minutes, rejection_rate_pct
- **And** data is accurate to within 1 minute

**Scenario 5: Error Handling - Double Claim**
- **Given** task T-E07-F20-003 already claimed by agent "backend"
- **When** orchestrator tries `shark task claim T-E07-F20-003 --agent=frontend`
- **Then** command fails with exit code 1
- **And** error message: "Task already claimed by agent 'backend' since 2026-01-11 10:00:00"
- **And** suggests: "Use 'shark task finish' or 'shark task reject' to complete work first"

---

## Out of Scope

### Explicitly Excluded

1. **Session Resume on Unblock**
   - **Why**: Adds complexity to unblock command; unclear if same session or new session
   - **Future**: Will be addressed in E14 (Advanced Workflow Features)
   - **Workaround**: Blocked sessions are closed; unblock creates new session on next claim

2. **Session Editing or Deletion**
   - **Why**: Sessions are audit records and should be immutable
   - **Future**: Admin tools may allow marking sessions as invalid (but not deleting)
   - **Workaround**: Incorrect sessions remain but can be filtered out of analytics

3. **Manual Session Creation**
   - **Why**: Sessions are automatic to ensure consistency and prevent user error
   - **Future**: Not planned; sessions are system-managed only
   - **Workaround**: N/A - sessions tied to workflow commands only

4. **Session Pause/Resume**
   - **Why**: Difficult to implement accurately; unclear when to pause (break? context switch?)
   - **Future**: Could be added if agents support explicit pause command
   - **Workaround**: Close session when work stops; new session on restart

5. **Real-Time Session Monitoring Dashboard**
   - **Why**: Out of scope for CLI tool; requires separate UI/web interface
   - **Future**: Could be built on top of session data via API
   - **Workaround**: Poll `shark task sessions --active` for monitoring

---

## Design

### Database Schema

See [Session Tracking Architecture](../../architecture/session-tracking.md) for complete database design.

**Key Schema Elements**:
- `task_sessions` table with task_id, agent_id, timestamps, outcome
- Generated column `duration_minutes` for automatic duration calculation
- Indexes on task_id, agent_id, started_at, and active sessions (ended_at IS NULL)
- Foreign key to tasks table with CASCADE delete
- CHECK constraint on outcome enum values

### Repository Layer

**WorkSessionRepository** (`internal/repository/work_session_repository.go`):
- `Create(ctx, session)` - Start new session
- `GetActiveSessionByTaskID(ctx, taskID)` - Find active session for task
- `EndSession(ctx, sessionID, outcome, notes)` - Close session
- `ListForTask(ctx, taskID)` - All sessions for task
- `GetStaleActiveSessions(ctx, threshold)` - Detect abandoned work
- `GetSessionStats(ctx, filters)` - Aggregate analytics

### Command Integration

**Claim Command**:
1. Validate task status and transition
2. Begin database transaction
3. Update task status
4. Create work session
5. Commit transaction
6. Return JSON with session info

**Finish Command**:
1. Validate task is in `in_*` status
2. Begin transaction
3. Update task status to next `ready_for_*`
4. Find and close active session with outcome="completed"
5. Commit transaction
6. Display session duration in output

**Reject Command**:
1. Validate rejection reason provided
2. Begin transaction
3. Update task status to rejection target
4. Close active session with outcome="rejected"
5. Store reason in session notes
6. Commit transaction

**Sessions Query Command**:
- New command: `shark task sessions <task-key>`
- Retrieves all sessions ordered by started_at DESC
- Calculates total time and completion rate
- Supports `--json` flag for programmatic access

---

## Test Plan

### Unit Tests

**Repository Tests** (`internal/repository/work_session_repository_test.go`):
- [ ] `TestWorkSessionRepository_Create` - Session creation with all fields
- [ ] `TestWorkSessionRepository_GetActiveSession` - Find active session by task ID
- [ ] `TestWorkSessionRepository_EndSession` - Close session with outcome and notes
- [ ] `TestWorkSessionRepository_ListForTask` - Retrieve all sessions for task
- [ ] `TestWorkSessionRepository_GetStaleActiveSessions` - Threshold filtering
- [ ] `TestWorkSessionRepository_GetSessionStats` - Aggregate calculations

**Model Tests** (`internal/models/work_session_test.go`):
- [ ] `TestWorkSession_Validate` - Field validation rules
- [ ] `TestSessionOutcome_Valid` - Enum value validation
- [ ] Duration calculation accuracy (via generated column)

### Integration Tests

**Command Integration** (`internal/cli/commands/task_claim_test.go`, etc.):
- [ ] `TestClaimCommand_CreatesSession` - Claim creates session atomically
- [ ] `TestFinishCommand_ClosesSession` - Finish closes with completed outcome
- [ ] `TestRejectCommand_ClosesSession` - Reject closes with rejected outcome
- [ ] `TestBlockCommand_ClosesSession` - Block closes with blocked outcome
- [ ] `TestClaimCommand_FailsOnActiveSession` - Error when double-claiming
- [ ] `TestFinishCommand_NoActiveSession` - Graceful handling when no session

**Full Lifecycle Tests** (`internal/cli/integration_test.go`):
- [ ] `TestFullSessionLifecycle` - Claim, finish, verify session data
- [ ] `TestRejectionLifecycle` - Claim, reject, re-claim, finish (2 sessions)
- [ ] `TestStaleSessionDetection` - Create old session, query with threshold
- [ ] `TestMultipleAgentSessions` - Different agents work on same task

### Analytics Tests

**Query Tests** (`internal/repository/work_session_analytics_test.go`):
- [ ] `TestSessionStats_PhaseDuration` - Phase analytics accuracy
- [ ] `TestSessionStats_AgentPerformance` - Agent statistics accuracy
- [ ] `TestSessionStats_CompletionRate` - Completion rate calculation
- [ ] `TestSessionStats_DateFiltering` - Filter by date range

### Performance Tests

**Benchmark Tests** (`internal/repository/work_session_benchmark_test.go`):
- [ ] `BenchmarkCreateSession` - Target: < 5ms
- [ ] `BenchmarkGetActiveSession` - Target: < 2ms (indexed query)
- [ ] `BenchmarkStaleSessionQuery` - Target: < 1s for 10K tasks
- [ ] `BenchmarkSessionStats` - Target: < 3s for 50K sessions

### Schema Migration Tests

**Migration Tests** (`internal/db/migrations_test.go`):
- [ ] `TestMigration_AddTaskSessionsTable` - Schema creation idempotent
- [ ] `TestMigration_Indexes` - All indexes created correctly
- [ ] `TestMigration_GeneratedColumn` - Duration calculation works
- [ ] `TestMigration_Trigger` - updated_at trigger functions

### Manual Testing Scenarios

**Scenario 1: Basic Session Tracking**
1. Create test task in ready_for_development
2. Claim task: `shark task claim T-E07-F99-001 --agent=backend`
3. Verify session created: `shark task sessions T-E07-F99-001`
4. Wait 5 minutes
5. Finish task: `shark task finish T-E07-F99-001 --notes="Test complete"`
6. Verify session closed with ~5 minute duration
7. Check JSON output: `shark task sessions T-E07-F99-001 --json`

**Scenario 2: Stale Session Detection**
1. Create 3 tasks and claim them
2. Update one session's started_at to 5 hours ago (SQL UPDATE)
3. Query stale sessions: `shark task sessions --active --stale=4h`
4. Verify only the old session appears

**Scenario 3: Multiple Sessions Per Task**
1. Claim task as backend agent
2. Reject task after 10 minutes
3. Claim same task as different agent
4. Finish task after 15 minutes
5. View sessions: `shark task sessions <task>`
6. Verify 2 sessions with correct agents and outcomes

**Scenario 4: Analytics Accuracy**
1. Create 20 tasks across different workflow phases
2. Simulate realistic work sessions (vary duration 10-60 min)
3. Include some rejections (20% rate)
4. Run analytics: `shark analytics phase-duration`
5. Manually verify calculations match raw session data

---

## Success Metrics

### Primary Metrics

1. **Session Capture Rate**
   - **What**: Percentage of task transitions that create/close sessions
   - **Target**: 100% of claims and finishes create session records
   - **Timeline**: Immediate after deployment
   - **Measurement**: Count sessions vs. status history records

2. **Stale Session Detection**
   - **What**: Number of abandoned work sessions detected and reassigned
   - **Target**: Reduce abandoned work (no session closure) by 80%
   - **Timeline**: Within 2 weeks of deployment
   - **Measurement**: Weekly report of stale session count

3. **Analytics Adoption**
   - **What**: Project managers actively using session analytics for planning
   - **Target**: 60% of PMs run analytics queries at least weekly
   - **Timeline**: Within 1 month of deployment
   - **Measurement**: Track command usage via telemetry

---

### Secondary Metrics

- **Sprint Planning Accuracy**: Increase estimate accuracy to ±20% (from current ±50%)
- **Workflow Optimization**: Identify and reduce longest-phase duration by 30% within 3 months
- **Agent Performance Visibility**: 100% of active agents have completion rate data
- **Query Performance**: All analytics queries complete in < 3 seconds

---

## Dependencies & Integrations

### Dependencies

- **E13-F01 (Claim Command)**: Session creation integrated into claim workflow
- **E13-F02 (Finish Command)**: Session closure integrated into finish workflow
- **E13-F03 (Reject Command)**: Session closure integrated into reject workflow
- **SQLite Database**: Requires SQLite 3.31+ for generated columns support
- **Existing task_history table**: Session outcomes complement status history

### Integration Requirements

- **Workflow Config Reader**: Session tracking uses workflow config to map phases
- **Task Repository**: Session operations wrapped in same transactions as status updates
- **CLI Global DB**: Sessions use centralized database initialization pattern

---

## Compliance & Security Considerations

**Data Retention**:
- Sessions are audit records and should be retained indefinitely
- No PII stored in sessions (agent IDs are system identifiers, not personal names)
- Notes may contain sensitive information - access control via database permissions

**Audit Trail**:
- Sessions provide complete audit trail of who worked on task and when
- Immutable records (no UPDATE, only INSERT) prevent tampering
- Timestamps use UTC to avoid timezone confusion

**Performance Impact**:
- Session creation adds minimal overhead to task commands (< 10ms)
- Indexed queries ensure analytics don't impact command performance
- Generated columns avoid runtime calculation overhead

---

*Last Updated*: 2026-01-11
