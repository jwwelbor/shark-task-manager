# Feature PRD: E10-F05 Work Sessions & Resume Context

**Feature Key**: E10-F05
**Epic**: [E10: Advanced Task Intelligence & Context Management](../epic.md)
**Status**: Draft
**Priority**: Could Have (Phase 3)
**Execution Order**: 5

---

## Goal

### Problem

AI development agents and human developers experience significant productivity loss when pausing and resuming work on tasks. When an agent's conversation ends or a developer switches tasks, critical context is lost:

- **What progress was made** before pausing (completed steps vs. remaining steps)
- **What decisions were made** and why (architectural choices, trade-offs considered)
- **What blockers were encountered** and how they were resolved
- **What questions remain unanswered** (awaiting design decisions, clarifications needed)
- **How much actual time was invested** (for estimation improvement)

Currently, there's no structured way to store this resume context or track multiple work sessions. The existing `started_at` and `completed_at` timestamps only capture the first start and final completion, hiding all intermediate pause/resume cycles.

**Real Example from E13**: Task T-E13-F02-004 (useContentColor composable) was completed in a single session according to timestamps, but actually involved 3 distinct work sessions:
1. Initial implementation (10/21 tests passing, paused to research gradient edge cases)
2. Edge case fixes (18/21 tests passing, paused for team standup)
3. Final polish (21/21 tests passing, documentation complete)

Without tracking these sessions and structured resume context, future agents resuming similar tasks have no visibility into the iterative development process, actual time invested, or intermediate blockers resolved.

### Solution

Create a comprehensive work session tracking and resume context system that:

- **Tracks multiple work sessions** per task with distinct start/end times and outcomes (completed, paused, blocked)
- **Stores structured resume context** as JSON including progress tracking, implementation decisions, open questions, blockers, acceptance criteria status, and related tasks
- **Provides resume command** that aggregates task details, notes (from E10-F01), completion metadata (from E10-F02), context data, and work sessions into a single comprehensive view
- **Enables session analytics** for estimation improvement and pattern discovery (average session duration, pause frequency, interruption analysis)

The system introduces:
1. New table `work_sessions` for multi-session tracking with agent_id, started_at, ended_at, outcome
2. New column `tasks.context_data` (JSON) for structured resume context
3. CLI commands: `shark task context`, `shark task resume`, `shark task sessions`, `shark analytics`

### Impact

**For AI Agents**:
- **90% faster resume**: Single `shark task resume` command provides everything needed (vs. 10+ minutes re-analyzing codebase)
- **Zero context loss**: Complete progress tracking eliminates duplicate work and rediscovered decisions
- **Clear next steps**: `context_data.remaining_steps` tells agent exactly what to do next

**For Human Developers**:
- **Interruption recovery**: Resume work after meetings/context switches without manual note review
- **Handoff efficiency**: Take over paused tasks from other developers with full context

**For Product Managers**:
- **Accurate estimates**: Session analytics reveal actual time patterns for estimation improvement (20% more accurate estimates based on historical session data)
- **Interruption visibility**: Identify tasks with frequent pauses (indicator of blockers or unclear requirements)
- **Velocity metrics**: Track actual work time vs. calendar time for realistic planning

**For Tech Leads**:
- **Quality assurance**: Review acceptance criteria status to verify task completeness before approval
- **Knowledge preservation**: Implementation decisions captured in context_data supplement decision notes from E10-F01

---

## User Personas

### Persona 1: AI Development Agent (Claude Code)

**Profile**:
- **Role/Title**: AI-powered development agent performing backend, frontend, and test implementation
- **Experience Level**: Expert technical knowledge but limited session continuity (conversations end, contexts reset)
- **Key Characteristics**:
  - Works asynchronously with frequent pause/resume cycles
  - Must resume work across conversation boundaries with zero prior memory
  - Benefits from structured context to eliminate re-analysis overhead

**Goals Related to This Feature**:
1. **Pause work with complete context** when conversation ends or blocker encountered
2. **Resume work immediately** with structured guidance on what's done, what's next, and open questions
3. **Track progress systematically** to avoid duplicate work across multiple sessions
4. **Record implementation decisions** within context for future reference

**Pain Points This Feature Addresses**:
- **Resume Overhead**: Currently spends 10+ minutes re-reading files, notes, and history to understand task state
- **Duplicate Work**: Without progress tracking, may redo completed steps or miss remaining steps
- **Lost Questions**: Open questions not captured, leading to guesswork or incorrect assumptions

**Success Looks Like**:
Agent can pause at any point (end of conversation, blocker encountered), resume hours or days later with `shark task resume T-XYZ`, and immediately continue work with complete context in <1 minute.

---

### Persona 2: Human Developer (Technical Lead)

**Profile**:
- **Role/Title**: Senior developer or tech lead managing development workflow and code quality
- **Experience Level**: 5+ years development experience, responsible for task review and approval
- **Key Characteristics**:
  - Reviews task completions for quality and completeness
  - Needs to verify acceptance criteria are met before approving
  - Must resume paused tasks when taking over from other developers

**Goals Related to This Feature**:
1. **Verify task completeness** by checking acceptance criteria status before approval
2. **Resume tasks paused by others** with full context (what's done, what remains, open questions)
3. **Understand implementation decisions** without manual code analysis
4. **Track actual time invested** for estimation improvement

**Pain Points This Feature Addresses**:
- **Incomplete Visibility**: Can't see which acceptance criteria are met vs. pending without manual code review
- **Handoff Friction**: Taking over paused tasks requires extensive context gathering from notes, code, and original assignee
- **Opaque Progress**: No structured view of completed vs. remaining steps

**Success Looks Like**:
Tech lead can review a task ready for approval, run `shark task resume T-XYZ`, see 7/7 acceptance criteria complete with verification notes, approve confidently in <3 minutes.

---

### Persona 3: Product Manager

**Profile**:
- **Role/Title**: Product manager overseeing feature development and delivery
- **Experience Level**: 3+ years product management, focuses on delivery velocity and risk management
- **Key Characteristics**:
  - Tracks feature completion and identifies delivery risks
  - Needs accurate estimates for roadmap planning
  - Reports on delivery status and velocity to stakeholders

**Goals Related to This Feature**:
1. **Improve estimation accuracy** using actual session duration data
2. **Identify interrupted tasks** that may indicate blockers or unclear requirements
3. **Track actual work time** vs. calendar time for realistic velocity metrics
4. **Report on progress** with confidence in completeness via acceptance criteria tracking

**Pain Points This Feature Addresses**:
- **Inaccurate Estimates**: Estimates based on gut feel, not historical data; leads to missed deadlines
- **Hidden Interruptions**: Tasks taking longer than expected with no visibility into why (frequent pauses, blockers)
- **Opaque Velocity**: Can't distinguish between active work time and waiting time

**Success Looks Like**:
PM can run `shark analytics --session-duration --epic E13`, see average session duration is 2.5 hours with 3.2 sessions per task, use this data to improve estimates, resulting in 20% more accurate delivery predictions.

---

## User Stories

### Must-Have Stories

**Story 1**: As an AI agent, I want to store structured resume context when pausing work so that I can resume efficiently in a future session without re-analyzing the codebase.

**Acceptance Criteria**:
- [ ] ALTER TABLE tasks ADD COLUMN context_data TEXT (valid JSON)
- [ ] CLI command `shark task context set <task-key> --field <field> "<value>"` updates specific context field
- [ ] Supported fields: current_step, completed_steps, remaining_steps, implementation_decisions, open_questions
- [ ] Context data is valid JSON (validated before storage)
- [ ] Context fields can be updated independently (partial updates supported)

---

**Story 2**: As an AI agent resuming work, I want one command that provides all context (task details, notes, completion metadata, context data, work sessions) so that I don't waste time gathering information from multiple sources.

**Acceptance Criteria**:
- [ ] CLI command `shark task resume <task-key>` exists
- [ ] Output includes task details (title, description, status, priority, dependencies)
- [ ] Output includes all notes chronologically (from E10-F01 task_notes)
- [ ] Output includes completion metadata if task completed (from E10-F02)
- [ ] Output includes context_data parsed and formatted (progress, decisions, questions, blockers, acceptance criteria status, related tasks)
- [ ] Output includes work sessions with durations and outcomes
- [ ] Output highlights open questions and blockers prominently
- [ ] `--json` flag outputs structured JSON for AI agent parsing

---

**Story 3**: As an AI agent, I want my work sessions automatically tracked when I start and complete tasks so that time data is captured without manual effort.

**Acceptance Criteria**:
- [ ] CREATE TABLE work_sessions with columns: id, task_id, agent_id, started_at, ended_at, outcome
- [ ] `shark task start <task-key>` automatically creates new work session with started_at=now, ended_at=NULL
- [ ] `shark task complete <task-key>` automatically ends current session with ended_at=now, outcome='completed'
- [ ] `shark task block <task-key>` automatically ends current session with outcome='blocked'
- [ ] Only one active session (ended_at=NULL) per task at a time
- [ ] Agent ID captured from system context or CLI flag

---

**Story 4**: As a product manager, I want to view all work sessions for a task so that I can understand how much actual time was invested and identify interruption patterns.

**Acceptance Criteria**:
- [ ] CLI command `shark task sessions <task-key>` shows all sessions with start time, end time, duration, outcome, agent ID
- [ ] Duration calculated as (ended_at - started_at) and formatted human-readable (e.g., "2h 15m")
- [ ] Active sessions (ended_at=NULL) show "In Progress" for duration
- [ ] Output includes session count and total time spent
- [ ] `--json` flag outputs structured JSON

---

### Should-Have Stories

**Story 5**: As a product manager, I want to see session analytics (average session duration, pause frequency) across an epic so that I can improve future estimates.

**Acceptance Criteria**:
- [ ] CLI command `shark analytics --session-duration --epic <epic-key>` calculates average session duration
- [ ] Output includes: total sessions, average duration, median duration, total time
- [ ] CLI command `shark analytics --pause-frequency --epic <epic-key>` shows how often tasks are paused
- [ ] Output includes: tasks with multiple sessions, average sessions per task, pause rate
- [ ] Can filter by agent type: `--agent-type frontend`

---

**Story 6**: As an AI agent, I want to pause a task with a note about why so that the next session knows the reason for interruption.

**Acceptance Criteria**:
- [ ] CLI command `shark task session pause <task-key> --note "<reason>"` exists
- [ ] Command ends current work session with ended_at=now, outcome='paused'
- [ ] Pause note stored in work_sessions table (new column: pause_note TEXT)
- [ ] Note displayed in `shark task sessions` and `shark task resume` output

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when I try to set context on a non-existent task, I want a clear error message.

**Acceptance Criteria**:
- [ ] Error message: "Task T-XYZ not found"
- [ ] Exit code 1 (not found)
- [ ] Suggests checking task key with `shark task list`

**Error Story 2**: As a user, when I try to set context with invalid JSON field, I want to see supported field names.

**Acceptance Criteria**:
- [ ] Error message: "Invalid context field: xyz (must be one of: current_step, completed_steps, remaining_steps, implementation_decisions, open_questions, blockers, acceptance_criteria_status, related_tasks)"
- [ ] Exit code 3 (invalid state)

**Error Story 3**: As a user, when I try to start a task that already has an active session, I want to know the conflict.

**Acceptance Criteria**:
- [ ] Error message: "Task T-XYZ already has an active work session started at 2025-12-26 14:15. Complete or pause current session first."
- [ ] Exit code 3 (invalid state)
- [ ] Suggests using `shark task sessions T-XYZ` to view active session

---

## Requirements

### Functional Requirements

**Category: Context Storage**

1. **REQ-F-008**: Structured Context Storage
   - **Description**: System must allow storing structured JSON context data for tasks with predefined fields
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] ALTER TABLE tasks ADD COLUMN context_data TEXT
     - [ ] `shark task context set <task-key> --field <field> "<value>"` updates specific field
     - [ ] Supported fields: current_step, completed_steps, remaining_steps, implementation_decisions, open_questions, blockers, acceptance_criteria_status, related_tasks
     - [ ] Context data validated as valid JSON before storage
     - [ ] Partial updates supported (update one field without overwriting others)

---

**Category: Resume Command**

2. **REQ-F-009**: Resume Command with Full Context
   - **Description**: System must provide comprehensive context retrieval for resuming tasks in a single command
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] CLI command `shark task resume <task-key>` exists
     - [ ] Output includes task details (title, description, status, priority, dependencies)
     - [ ] Output includes all notes chronologically (from E10-F01)
     - [ ] Output includes completion metadata (from E10-F02)
     - [ ] Output includes context_data parsed with sections: progress, decisions, questions, blockers, acceptance criteria status, related tasks
     - [ ] Output includes work sessions with durations and outcomes
     - [ ] Human-readable formatting with sections for each context type
     - [ ] Highlights open questions and blockers prominently (visual indicators)
     - [ ] `--json` flag outputs structured JSON

---

**Category: Work Session Tracking**

3. **REQ-F-018**: Work Session Tracking
   - **Description**: System must track individual work sessions for tasks with automatic session creation and termination
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] CREATE TABLE work_sessions with columns: id, task_id, agent_id, started_at, ended_at, outcome, pause_note
     - [ ] `shark task start` automatically creates new work session
     - [ ] `shark task complete` ends session with outcome='completed'
     - [ ] `shark task block` ends session with outcome='blocked'
     - [ ] `shark task session pause <task-key> --note "<reason>"` ends session with outcome='paused'
     - [ ] Only one active session per task (constraint: ended_at IS NULL)
     - [ ] Foreign key constraint: work_sessions.task_id → tasks.id

---

4. **REQ-F-019**: Session History and Analytics
   - **Description**: System must provide visibility into work session patterns for individual tasks and aggregated analytics
   - **User Story**: Links to Story 4, Story 5
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task sessions <task-key>` shows all sessions with duration
     - [ ] Duration calculated and formatted human-readable
     - [ ] Output includes total time, session count, average duration
     - [ ] `shark analytics --session-duration --epic <epic>` calculates average duration across epic
     - [ ] `shark analytics --pause-frequency --epic <epic>` shows pause patterns
     - [ ] Optional filter by agent type: `--agent-type frontend`

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-002**: Resume Command Performance
   - **Description**: Resume command must complete in <2 seconds for tasks with up to 100 notes and 10 sessions
   - **Measurement**: Execute `shark task resume <task-key>` and measure execution time
   - **Target**: p95 < 2 seconds
   - **Justification**: Resume command is information-dense; users will tolerate slight delay for comprehensive view

2. **REQ-NF-003**: Context Update Performance
   - **Description**: Context data updates must complete in <500ms
   - **Measurement**: Execute `shark task context set` and measure execution time
   - **Target**: p99 < 500ms
   - **Justification**: Context updates happen frequently during task execution; must be fast

**Data Integrity**

3. **REQ-NF-004**: Foreign Key Enforcement
   - **Description**: All work session records must enforce referential integrity via foreign keys
   - **Implementation**: `work_sessions.task_id` → `tasks.id` with `ON DELETE CASCADE`
   - **Compliance**: SQLite `PRAGMA foreign_keys = ON`
   - **Risk Mitigation**: Prevents orphaned sessions when tasks are deleted

4. **REQ-NF-006**: JSON Validation
   - **Description**: context_data field must contain valid JSON
   - **Implementation**: Validate JSON before insert/update, return error on invalid JSON
   - **Testing**: Attempt to insert malformed JSON, verify error returned
   - **Risk Mitigation**: Prevents data corruption and parsing errors

5. **REQ-NF-013**: Active Session Constraint
   - **Description**: Each task can have at most one active work session (ended_at IS NULL)
   - **Implementation**: Database constraint or application-level validation
   - **Testing**: Attempt to start task with existing active session, verify error
   - **Risk Mitigation**: Prevents duplicate session tracking and data inconsistency

**Usability**

6. **REQ-NF-007**: Human-Readable Output
   - **Description**: All CLI commands must provide human-readable output by default
   - **Implementation**: Format sections clearly, use visual indicators for blockers/questions, colorize timestamps
   - **Testing**: Manual review of `shark task resume` output for readability
   - **Risk Mitigation**: Improves developer experience and adoption

7. **REQ-NF-008**: JSON Output Mode
   - **Description**: All CLI commands must support `--json` flag for machine-readable output
   - **Implementation**: Marshal results to JSON when `--json` flag present
   - **Testing**: Verify all commands produce valid, parseable JSON with `--json`
   - **Risk Mitigation**: Enables AI agent automation and scripting

**Backward Compatibility**

8. **REQ-NF-009**: Database Migration
   - **Description**: All schema changes must be applied via automatic migrations
   - **Implementation**: Use migration system in internal/db/migrations.go
   - **Testing**: Test migration on existing database, verify no data loss
   - **Risk Mitigation**: Existing Shark installations upgrade seamlessly

9. **REQ-NF-010**: CLI Compatibility
   - **Description**: Existing `shark task start` and `shark task complete` commands continue to work unchanged (session tracking is transparent)
   - **Implementation**: Session creation/termination is automatic, no required flags
   - **Testing**: Run existing CLI test suite, verify no regressions
   - **Risk Mitigation**: Prevents breaking user workflows

---

## Database Schema

### New Table: work_sessions

```sql
CREATE TABLE work_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    agent_id TEXT,            -- 'claude', agent execution ID, or username
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,       -- NULL for active sessions
    outcome TEXT CHECK (outcome IN (
        'completed',          -- Task completed in this session
        'paused',             -- Work paused, will resume later
        'blocked'             -- Work blocked by external dependency
    )),
    pause_note TEXT,          -- Reason for pause (optional)
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Index for fast retrieval of all sessions for a task
CREATE INDEX idx_work_sessions_task_id ON work_sessions(task_id);

-- Index for active session queries
CREATE INDEX idx_work_sessions_active ON work_sessions(task_id, ended_at) WHERE ended_at IS NULL;

-- Index for agent-based analytics
CREATE INDEX idx_work_sessions_agent ON work_sessions(agent_id);
```

**Schema Design Rationale**:
- **ended_at NULL for active sessions**: Simplifies query for active sessions (`WHERE ended_at IS NULL`)
- **outcome as CHECK constraint**: Three fixed outcomes unlikely to change; simpler than separate table
- **pause_note optional**: Not all pauses have explicit reasons (conversation timeout, context switch)
- **ON DELETE CASCADE**: Sessions meaningless without parent task; cascade prevents orphaned records
- **Partial index on active sessions**: Optimizes common query pattern for finding active sessions

---

### New Column: tasks.context_data

```sql
ALTER TABLE tasks ADD COLUMN context_data TEXT;  -- JSON blob
```

**JSON Structure**:

```json
{
  "progress": {
    "completed_steps": [
      "Created base ThemeToggle.vue component",
      "Added icon switching logic (Sun/Moon/Monitor)"
    ],
    "current_step": "Implementing dropdown menu for 3 theme options",
    "remaining_steps": [
      "Add keyboard shortcuts (Ctrl+Shift+T)",
      "Add tests (8 test cases planned)",
      "Update documentation"
    ]
  },
  "implementation_decisions": {
    "component_location": "frontend/src/components/ThemeToggle.vue",
    "uses_composable": "useTheme from @/composables/useTheme",
    "icon_library": "lucide-vue-next (Sun, Moon, Monitor icons)",
    "pattern": "Dropdown menu with 3 options, not just toggle"
  },
  "open_questions": [
    "Should theme toggle be in header or settings page?",
    "Do we need keyboard shortcut or just click?"
  ],
  "blockers": [
    {
      "description": "Awaiting design decision on placement",
      "blocker_type": "external_dependency",
      "blocked_since": "2025-12-26 14:30"
    }
  ],
  "acceptance_criteria_status": [
    {"criterion": "Toggle switches between light/dark", "status": "complete"},
    {"criterion": "Dropdown shows 3 options (light/dark/system)", "status": "in_progress"},
    {"criterion": "Icon reflects current theme", "status": "complete"},
    {"criterion": "Keyboard accessible", "status": "pending"},
    {"criterion": "Tests pass", "status": "pending"}
  ],
  "related_tasks": [
    "T-E13-F05-003",  // useTheme composable (completed)
    "T-E13-F05-001"   // Dark mode CSS (completed)
  ]
}
```

**Field Definitions**:
- **progress**: Tracks what's done, what's current, what remains
- **implementation_decisions**: Key architectural and technical choices (supplements E10-F01 decision notes)
- **open_questions**: Unanswered questions awaiting external input
- **blockers**: Array of blocker objects with type and timestamp
- **acceptance_criteria_status**: Per-criterion tracking (alternative to E10-F04 if not implemented)
- **related_tasks**: Task keys for related/dependent tasks (supplements E10-F03 relationships)

**Schema Design Rationale**:
- **JSON vs. normalized tables**: JSON chosen for flexibility and ease of partial updates; not queried frequently (only on resume)
- **acceptance_criteria_status in context**: Redundant with E10-F04 if implemented, but provides fallback for simple AC tracking
- **related_tasks as array**: Simpler than E10-F03 relationships table for basic use cases

---

## CLI Commands Specification

### Command: `shark task context set`

**Purpose**: Set or update a specific field in task context data

**Syntax**:
```bash
shark task context set <task-key> --field <field> "<value>" [--json]
```

**Arguments**:
- `<task-key>`: Required. Task key (e.g., T-E13-F05-004)
- `--field <field>`: Required. Context field to update (see supported fields below)
- `"<value>"`: Required. Field value (quoted string; for arrays/objects, use JSON string)
- `--json`: Optional. Output JSON

**Supported Fields**:
- `current_step`: String describing current work step
- `completed_steps`: JSON array of completed steps
- `remaining_steps`: JSON array of remaining steps
- `implementation_decisions`: JSON object with decision key-value pairs
- `open_questions`: JSON array of question strings
- `blockers`: JSON array of blocker objects
- `acceptance_criteria_status`: JSON array of criterion objects
- `related_tasks`: JSON array of task keys

**Examples**:
```bash
# Set current step
shark task context set T-E13-F05-004 --field current_step "Implementing dropdown menu for 3 theme options"

# Add completed step (append to array)
shark task context set T-E13-F05-004 --field completed_steps '["Created base component", "Added icon switching"]'

# Add open question
shark task context set T-E13-F05-004 --field open_questions '["Should toggle be in header or settings?"]'

# Add blocker
shark task context set T-E13-F05-004 --field blockers '[{"description": "Awaiting design decision", "blocker_type": "external_dependency", "blocked_since": "2025-12-26 14:30"}]'
```

---

### Command: `shark task resume`

**Purpose**: Get comprehensive context for resuming a task (aggregates task details, notes, completion metadata, context data, work sessions)

**Syntax**:
```bash
shark task resume <task-key> [--json]
```

**Example**:
```bash
shark task resume T-E13-F05-004
```

**Output**: Human-readable formatted sections including:
- Task overview (title, epic, status, priority, dependencies)
- Progress overview (completed/current/remaining steps)
- Open questions (highlighted)
- Active blockers (highlighted)
- Implementation decisions
- Acceptance criteria status
- Timeline (recent activity from notes and history)
- Work sessions (durations and outcomes)
- Related tasks
- Next steps (derived from context)

---

### Command: `shark task sessions`

**Purpose**: View all work sessions for a task

**Syntax**:
```bash
shark task sessions <task-key> [--json]
```

**Example**:
```bash
shark task sessions T-E13-F02-004
```

**Output**:
```
Task T-E13-F02-004: Create useContentColor Composable (3 sessions, 3h 30m total)

Session 1: 2025-12-26 10:00 - 11:30 (1h 30m) → paused
  Agent: claude (abcf742)
  Note: Created base composable, 10/21 tests passing

Session 2: 2025-12-26 13:00 - 14:15 (1h 15m) → paused
  Agent: claude (abcf742)
  Note: Added badge/card helpers, 18/21 tests passing

Session 3: 2025-12-26 15:00 - 15:45 (45m) → completed
  Agent: claude (abcf742)
  Note: 21/21 tests passing, documentation complete

Average Session: 1h 10m
```

---

### Command: `shark analytics`

**Purpose**: Analyze work session patterns across epics/features

**Syntax**:
```bash
shark analytics --session-duration [--epic <epic>] [--feature <feature>] [--agent-type <type>] [--json]
shark analytics --pause-frequency [--epic <epic>] [--feature <feature>] [--json]
```

**Examples**:
```bash
# Average session duration for epic
shark analytics --session-duration --epic E13

# Pause frequency analysis
shark analytics --pause-frequency --epic E13
```

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Context Data Storage**
- **Given** a task T-E13-F05-004 exists
- **When** user executes `shark task context set T-E13-F05-004 --field current_step "Implementing dropdown menu"`
- **Then** context_data column updated with JSON: `{"current_step": "Implementing dropdown menu"}`
- **And** existing context fields preserved (partial update)
- **And** success message displayed

**Scenario 2: Resume Command Aggregation**
- **Given** task T-E13-F05-004 has 5 notes, context_data with progress/questions/blockers, and 2 work sessions
- **When** user executes `shark task resume T-E13-F05-004`
- **Then** output includes all task details, all 5 notes chronologically, parsed context data, 2 sessions with durations
- **And** open questions highlighted prominently
- **And** blockers highlighted prominently

**Scenario 3: Automatic Session Creation**
- **Given** task T-E13-F05-004 has no active sessions
- **When** user executes `shark task start T-E13-F05-004 --agent claude`
- **Then** new work session created with started_at=now, ended_at=NULL, agent_id='claude'
- **And** task status updated to in_progress

**Scenario 4: Session History View**
- **Given** task T-E13-F02-004 has 3 completed sessions (1h 30m, 1h 15m, 45m)
- **When** user executes `shark task sessions T-E13-F02-004`
- **Then** all 3 sessions displayed with start/end times, durations, outcomes, agent IDs
- **And** total time displayed: 3h 30m
- **And** average session displayed: 1h 10m

---

## Out of Scope

### Explicitly Excluded

1. **Context Data Versioning / History**
   - **Why**: Adds significant complexity; context_data is meant for current state only, not historical tracking
   - **Future**: Phase 4 enhancement if users request historical context tracking
   - **Workaround**: Historical context visible through timeline notes (E10-F01)

2. **Session Editing After Creation**
   - **Why**: Sessions are immutable audit trail; editing would compromise time tracking accuracy
   - **Future**: Not planned unless compliance requires correction capability
   - **Workaround**: Add corrective note explaining session discrepancy

3. **Real-Time Session Duration Alerts**
   - **Why**: CLI tool doesn't run continuously; no background process for real-time monitoring
   - **Future**: Web UI version could include real-time alerts for long-running sessions
   - **Workaround**: Manual check via `shark task sessions <task-key>`

4. **Automatic Context Extraction from Code**
   - **Why**: Complex AI/NLP feature requiring significant research and implementation effort
   - **Future**: Phase 4+ enhancement if AI-powered analysis proves valuable
   - **Workaround**: Manual context updates by agents during implementation

5. **Cross-Task Context Relationships**
   - **Why**: Overlaps with E10-F03 (Task Relationships); context_data is task-scoped only
   - **Future**: Not planned; use E10-F03 relationships for cross-task dependencies
   - **Workaround**: Store related task keys in context_data.related_tasks array

---

## Success Metrics

### Primary Metrics

1. **Resume Efficiency (Agent Perspective)**
   - **What**: Time from `shark task resume` execution to first productive action
   - **Target**: <1 minute for 90% of resumes (vs. 10+ minutes pre-feature baseline)
   - **Timeline**: 2 weeks after Phase 3 release
   - **Measurement**: AI agent self-report + timing logs

2. **Context Capture Adoption**
   - **What**: Percentage of paused tasks with context_data populated
   - **Target**: 70% of paused/blocked tasks have at least 3 context fields populated (progress, decisions, questions)
   - **Timeline**: 1 month after Phase 3 release
   - **Measurement**: SQL query on tasks with populated context_data

3. **Estimation Accuracy Improvement**
   - **What**: Variance reduction in task time estimates
   - **Target**: 20% reduction in estimate variance for Phase 4 vs. Phase 3 (using Phase 3 session data for Phase 4 estimates)
   - **Timeline**: 3 months after Phase 3 release (measure during Phase 4 planning)
   - **Measurement**: Compare estimated vs. actual session durations

---

### Secondary Metrics

- **Session Count per Task**: Average 2.5 sessions per task (indicates healthy pause/resume patterns)
- **Pause Reason Analysis**: Top 3 pause reasons identified, actionable insights for process improvement
- **Resume Command Usage**: 5+ uses per week per active developer
- **Context Field Coverage**: 80% of context records have all recommended fields populated

---

## Dependencies & Integrations

### Dependencies

**Required Features** (Must be implemented before E10-F05):
- **E10-F01**: Task Activity & Notes System
  - `shark task resume` includes all notes chronologically
  - Timeline view integrates notes with sessions
- **E10-F02**: Task Completion Intelligence
  - `shark task resume` includes completion metadata for completed tasks
  - Completion metadata shows verification status

**Optional Features** (Enhance E10-F05 if available):
- **E10-F03**: Task Relationships & Dependencies
  - `shark task resume` could show relationship graph instead of simple related_tasks array
- **E10-F04**: Acceptance Criteria & Search
  - `shark task resume` could pull AC status from task_criteria table instead of context_data.acceptance_criteria_status

**Existing Infrastructure**:
- `tasks` table (required for context_data column)
- `task_history` table (required for timeline integration)
- Cobra CLI framework
- Repository pattern

---

### Integration Points

**Timeline Integration**:
- `shark task resume` must query `task_history`, `task_notes` (E10-F01), and `work_sessions` tables
- Merge all events chronologically for unified timeline

**Resume Context Aggregation**:
- Combines task details, notes (E10-F01), completion metadata (E10-F02), context data, and work sessions

---

## Open Questions

- **Q1**: Should context_data support schema versioning for future evolution?
  - **Recommendation**: No for Phase 3; treat as unstructured JSON. Add versioning if breaking changes needed in Phase 4.

- **Q2**: Should there be a maximum number of work sessions per task?
  - **Recommendation**: No hard limit, but warn if >10 sessions (may indicate task should be split).

- **Q3**: Should `shark task resume` output be customizable (show/hide sections)?
  - **Recommendation**: Not initially; provide comprehensive output by default. Add `--sections` flag in Phase 4 if users request it.

- **Q4**: Should session analytics support time-based filtering (last 30 days, last quarter)?
  - **Recommendation**: Yes, add `--after <date>` and `--before <date>` flags for time-range filtering.

---

*Last Updated*: 2025-12-26
*Status*: Ready for Review
*Author*: BusinessAnalyst Agent
