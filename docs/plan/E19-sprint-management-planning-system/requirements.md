# Requirements

**Epic**: [Sprint Management & Planning System](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for the Sprint Management & Planning System. Requirements are organized by functional area and prioritized using MoSCoW.

**Requirement Traceability**: Each requirement maps to specific [user journeys](./user-journeys.md) and [personas](./personas.md).

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for launch; epic fails without these
- **Should Have**: Important but workarounds exist; target for initial release
- **Could Have**: Valuable but deferrable; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

### Must Have Requirements

#### Sprint Lifecycle Management

**REQ-F-001**: Sprint Creation
- **Description**: Users can create a named sprint with start date, end date, and optional goal text via `shark sprint create`
- **User Story**: As a PM, I want to create a time-boxed sprint so that I can group work into a planned iteration
- **Acceptance Criteria**:
  - [ ] `shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01` creates a sprint with key `S024` in `planning` status
  - [ ] `shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01 --goal="Complete template enrichment"` creates sprint with goal
  - [ ] Sprint key is auto-assigned as `S###` with zero-padded incrementing number
  - [ ] Start date must be before end date; system rejects invalid date ranges
  - [ ] `--json` flag outputs sprint data in machine-readable JSON format
- **Related Journey**: [Sprint Creation & Configuration](./user-journeys.md#journey-1-sprint-creation--configuration), Steps 1-2

**REQ-F-002**: Sprint Status Transitions
- **Description**: Sprints transition through lifecycle states: `planning` -> `active` -> `closing` -> `completed`, with `cancelled` as an alternative terminal state
- **User Story**: As a PM, I want to move a sprint through its lifecycle so that the system tracks where each sprint is
- **Acceptance Criteria**:
  - [ ] `shark sprint start S024` transitions from `planning` to `active`
  - [ ] `shark sprint close S024` transitions from `active` to `closing`, then to `completed` after carryover processing
  - [ ] `shark sprint archive S024` transitions from `completed` to `archived`
  - [ ] Only one sprint can be in `active` status at a time; `shark sprint start` fails if another sprint is active
  - [ ] Status transitions are validated -- invalid transitions return descriptive error messages
  - [ ] All transitions are recorded in sprint history for audit
- **Related Journey**: [Sprint Creation](./user-journeys.md#journey-1-sprint-creation--configuration), Step 6; [Sprint Close](./user-journeys.md#journey-4-sprint-close--retrospective), Step 2

**REQ-F-003**: Sprint CRUD Operations
- **Description**: Full CRUD operations for sprint entities: create, read, update, delete
- **User Story**: As a PM, I want to view, update, and manage sprint details so that sprint information stays current
- **Acceptance Criteria**:
  - [ ] `shark sprint get S024` displays sprint details (name, dates, goal, status, task count, capacity summary)
  - [ ] `shark sprint get S024 --json` outputs machine-readable sprint data
  - [ ] `shark sprint list` shows all sprints with status filter support (`--status=active`, `--status=planning`)
  - [ ] `shark sprint update S024 --goal="Updated goal" --end=2026-04-08` updates sprint attributes
  - [ ] `shark sprint delete S024` removes a sprint in `planning` status (cannot delete active or completed sprints)
  - [ ] `shark get S024` auto-detects sprint entity type from the `S###` key format
- **Related Journey**: [Sprint Creation](./user-journeys.md#journey-1-sprint-creation--configuration), Steps 1-4

#### Task-to-Sprint Assignment

**REQ-F-004**: Individual Task Assignment
- **Description**: Tasks can be assigned to and removed from sprints
- **User Story**: As a PM, I want to assign specific tasks to a sprint so that the sprint scope is defined
- **Acceptance Criteria**:
  - [ ] `shark sprint add S024 E07-F01-001` assigns task to sprint
  - [ ] `shark sprint remove S024 E07-F01-001` removes task from sprint
  - [ ] A task can belong to at most one sprint in `planning` or `active` status
  - [ ] Attempting to assign a task already in another active/planning sprint returns an error with the conflicting sprint key
  - [ ] Assigning a task that would exceed agent capacity displays a warning but proceeds (advisory)
  - [ ] `--json` output for both operations
- **Related Journey**: [Sprint Planning](./user-journeys.md#journey-2-sprint-planning), Steps 2, 5

**REQ-F-005**: Sprint Backlog View
- **Description**: View all tasks assigned to a sprint, grouped by status
- **User Story**: As a PM, I want to see all tasks in a sprint grouped by their status so that I can track progress
- **Acceptance Criteria**:
  - [ ] `shark sprint backlog S024` displays tasks grouped by status category (todo/planning, in-progress, review, qa, completed, blocked)
  - [ ] Shows completion percentage (completed tasks / total tasks)
  - [ ] `--blocked` filter shows only blocked tasks with blocking reason and days blocked
  - [ ] `--json` output for programmatic consumption
  - [ ] Output includes task key, title, status, agent type, and priority for each task
- **Related Journey**: [Sprint Monitoring](./user-journeys.md#journey-3-sprint-monitoring), Steps 1, 3

**REQ-F-006**: Sprint Close with Carryover
- **Description**: Closing a sprint handles incomplete tasks by moving them to the next sprint or back to the backlog
- **User Story**: As a PM, I want incomplete tasks handled automatically when I close a sprint so that nothing is lost
- **Acceptance Criteria**:
  - [ ] `shark sprint close S024 --carryover=next` moves incomplete tasks to the next sprint in `planning` status
  - [ ] `shark sprint close S024 --carryover=backlog` unassigns incomplete tasks (they return to the general backlog)
  - [ ] If `--carryover=next` is used and no next sprint exists, the system creates a new sprint in `planning` status with dates following the closed sprint's pattern
  - [ ] Completed tasks remain associated with the closed sprint for velocity calculation
  - [ ] Sprint close generates a completion record with summary statistics
  - [ ] Default carryover behavior is configurable in `.sharkconfig.json`
- **Related Journey**: [Sprint Close & Retrospective](./user-journeys.md#journey-4-sprint-close--retrospective), Steps 1-2

#### Sprint Analytics

**REQ-F-007**: Velocity Calculation
- **Description**: Calculate and display team velocity from historical sprint data
- **User Story**: As a PM, I want to see historical velocity so that I can plan future sprints with realistic scope
- **Acceptance Criteria**:
  - [ ] `shark sprint velocity` shows completed task count (or points) per sprint for the last 5 sprints
  - [ ] Displays trailing average velocity
  - [ ] `--sprints=N` flag overrides the number of historical sprints shown
  - [ ] `--json` output includes per-sprint data and averages
  - [ ] Shows "insufficient data" message when fewer than 3 completed sprints exist
- **Related Journey**: [Sprint Monitoring](./user-journeys.md#journey-3-sprint-monitoring), Step 4

**REQ-F-008**: Sprint Burndown
- **Description**: Display sprint progress as a burndown of remaining work over time
- **User Story**: As a PM, I want to see a burndown chart so that I can tell if the sprint is on track
- **Acceptance Criteria**:
  - [ ] `shark sprint burndown` shows burndown for the active sprint (or `shark sprint burndown S024` for a specific sprint)
  - [ ] Displays ideal burndown line (linear from total to zero over sprint duration)
  - [ ] Displays actual remaining tasks/points per day
  - [ ] Text-based chart suitable for terminal display (using ASCII art or Unicode block characters)
  - [ ] `--json` output includes daily data points for both ideal and actual lines
  - [ ] Burndown accounts for tasks added or removed mid-sprint
- **Related Journey**: [Sprint Monitoring](./user-journeys.md#journey-3-sprint-monitoring), Step 2

**REQ-F-009**: Sprint Summary Report
- **Description**: Comprehensive sprint summary for retrospective discussions
- **User Story**: As a PM, I want a detailed sprint summary so that I can run data-driven retrospectives
- **Acceptance Criteria**:
  - [ ] `shark sprint summary S024` displays: planned task count, completed task count, completion percentage, velocity (this sprint), velocity comparison to trailing average
  - [ ] `shark sprint summary S024 --detailed` adds: tasks added mid-sprint, tasks removed mid-sprint, average cycle time by phase, agent utilization vs. capacity, blocked time analysis, carryover task list
  - [ ] `--json` output for programmatic processing
  - [ ] Summary is available for any sprint in `completed` or `archived` status
- **Related Journey**: [Sprint Close & Retrospective](./user-journeys.md#journey-4-sprint-close--retrospective), Steps 1, 3

#### Database Schema

**REQ-F-010**: Sprint Database Tables
- **Description**: Database schema to store sprint data, task assignments, and capacity
- **User Story**: As the system, I need persistent storage for sprint entities and relationships so that sprint data survives across sessions
- **Acceptance Criteria**:
  - [ ] `sprints` table with columns: id, key, name, goal, start_date, end_date, status, slug, file_path, created_at, updated_at
  - [ ] `sprint_assignments` table with columns: id, sprint_id (FK), task_id (FK), assigned_at, removed_at (nullable for soft-delete)
  - [ ] `sprint_capacity` table with columns: id, sprint_id (FK), agent_type, capacity_points, allocated_points
  - [ ] Foreign key constraints enforce referential integrity
  - [ ] Indexes on sprint status, task_id, and sprint_id for query performance
  - [ ] Migration is idempotent and follows existing migration patterns in `internal/db/db.go`
  - [ ] `CurrentSchemaVersion` is bumped when migration is added

---

### Should Have Requirements

#### Sprint Planning View

**REQ-F-011**: Sprint Planning Command
- **Description**: Interactive planning view that shows backlog, capacity, and readiness in a single view
- **User Story**: As a PM, I want a single planning view so that I can see everything needed to scope a sprint
- **Acceptance Criteria**:
  - [ ] `shark sprint plan S024` displays: (a) unassigned backlog tasks sorted by priority and dependency order, (b) current sprint task count and capacity utilization per agent type, (c) sprint readiness score
  - [ ] Backlog shows only tasks in statuses eligible for sprint assignment (`ready_for_development` or earlier, not already in another sprint)
  - [ ] `--json` output for orchestrator consumption
- **Related Journey**: [Sprint Planning](./user-journeys.md#journey-2-sprint-planning), Step 1

**REQ-F-012**: Bulk Task Assignment
- **Description**: Assign all eligible tasks from a feature to a sprint in one command
- **User Story**: As a PM, I want to bulk-assign tasks from a feature so that sprint planning is faster
- **Acceptance Criteria**:
  - [ ] `shark sprint add S024 --bulk E07-F34` assigns all tasks from E07-F34 that are in assignable statuses and not already in a sprint
  - [ ] Displays count of tasks added and updated capacity utilization
  - [ ] Warns if bulk assignment exceeds any agent type's capacity
- **Related Journey**: [Sprint Planning](./user-journeys.md#journey-2-sprint-planning), Step 3

**REQ-F-013**: Sprint Readiness Score
- **Description**: Calculate a readiness score (0-100) for a sprint based on planning quality indicators
- **User Story**: As a PM, I want a readiness score so that I know if the sprint is well-planned before starting
- **Acceptance Criteria**:
  - [ ] `shark sprint readiness S024` displays score with breakdown
  - [ ] Score factors: capacity utilization (penalizes >100% or <50%), dependency satisfaction (penalizes external blocked dependencies), task count (penalizes empty sprint), agent balance (penalizes single-agent-type sprints)
  - [ ] Each factor shows its individual score and contribution to overall readiness
  - [ ] `--json` output with full breakdown
- **Related Journey**: [Sprint Planning](./user-journeys.md#journey-2-sprint-planning), Step 4

#### Capacity Management

**REQ-F-014**: Agent Capacity Configuration
- **Description**: Set and view capacity targets per agent type per sprint
- **User Story**: As a PM, I want to set capacity per agent type so that I can balance workload
- **Acceptance Criteria**:
  - [ ] `shark sprint capacity set S024 --agent=backend --points=21` sets capacity for an agent type
  - [ ] `shark sprint capacity show S024` displays all agent types with capacity, allocated, and remaining points
  - [ ] Allocated points are automatically calculated from assigned task priorities or story point estimates
  - [ ] `--json` output for programmatic consumption
- **Related Journey**: [Sprint Creation](./user-journeys.md#journey-1-sprint-creation--configuration), Step 3

**REQ-F-015**: Default Capacity Configuration
- **Description**: Configure default capacity per agent type in `.sharkconfig.json` so it does not need to be set per sprint
- **User Story**: As a PM, I want default capacity values so that sprint setup is faster for stable teams
- **Acceptance Criteria**:
  - [ ] `.sharkconfig.json` supports a `sprint_defaults.capacity` section with per-agent-type defaults
  - [ ] New sprints inherit default capacity when no explicit capacity is set
  - [ ] Per-sprint capacity overrides defaults
  - [ ] `shark sprint capacity set --default --agent=backend --points=21` updates the default configuration

---

### Could Have Requirements

**REQ-F-016**: Sprint Auto-Creation
- **Description**: Automatically create the next sprint when the current one closes
- **User Story**: As a PM, I want sprints created automatically so that the cadence is maintained without manual intervention
- **Acceptance Criteria**:
  - [ ] Configurable in `.sharkconfig.json`: `sprint_defaults.auto_create: true`
  - [ ] New sprint duration matches the closed sprint's duration
  - [ ] New sprint start date is the day after the closed sprint's end date
  - [ ] Auto-created sprint is in `planning` status

**REQ-F-017**: Sprint Integration with `shark status` Dashboard
- **Description**: The project dashboard (`shark status`) shows active sprint information
- **User Story**: As a PM, I want sprint info on the dashboard so that sprint status is always visible
- **Acceptance Criteria**:
  - [ ] `shark status` shows active sprint name, dates, progress percentage, and days remaining
  - [ ] `shark status` shows sprint velocity if available

**REQ-F-018**: Sprint History for Entities
- **Description**: View which sprints a task was assigned to historically
- **User Story**: As a PM, I want to see sprint history for a task so that I can track carryover patterns
- **Acceptance Criteria**:
  - [ ] `shark sprint history E07-F01-001` shows all sprints the task was assigned to with dates
  - [ ] Includes current assignment and historical (removed/carried-over) assignments

---

## Non-Functional Requirements

### Performance

**REQ-NF-001**: Sprint Command Response Time
- **Description**: All sprint commands must complete within acceptable response times
- **Measurement**: Wall-clock time from command invocation to output display
- **Target**: Sprint CRUD operations < 500ms; analytics commands (velocity, burndown, summary) < 2 seconds for projects with up to 50 sprints and 1000 tasks
- **Justification**: Consistent with existing shark command performance expectations; analytics commands have higher tolerance due to aggregation queries

**REQ-NF-002**: Database Query Efficiency
- **Description**: Sprint-related queries must use indexed lookups and avoid full table scans
- **Measurement**: SQLite EXPLAIN QUERY PLAN output shows index usage
- **Target**: All frequently-used queries (sprint get, backlog view, assignment check) use index scans
- **Justification**: Sprint commands will be called frequently during sprint execution; poor query performance would degrade daily workflow

### Data Integrity

**REQ-NF-003**: Sprint Assignment Consistency
- **Description**: Task-to-sprint assignments must maintain referential integrity and prevent invalid states
- **Measurement**: Database constraints prevent orphaned assignments and double-active-sprint scenarios
- **Target**: Foreign key constraints on sprint_assignments; unique constraint on (task_id) WHERE removed_at IS NULL to prevent duplicate active assignments; CHECK constraint on sprint status values
- **Justification**: Corrupted sprint data would produce incorrect velocity calculations and misleading analytics

### Compatibility

**REQ-NF-004**: Backward Compatibility
- **Description**: Adding sprint support must not break existing commands or data
- **Measurement**: All existing tests pass after sprint schema migration
- **Target**: Zero breaking changes to existing CLI commands, database schema (additive only), or workflow behavior
- **Justification**: Sprint management is an additive feature; existing users must not experience regressions

**REQ-NF-005**: JSON Output Consistency
- **Description**: All sprint commands support `--json` output following existing JSON output patterns
- **Measurement**: JSON output validates against expected schema; `--field` flag works for extracting individual fields
- **Target**: All sprint commands support `--json` and `--field` flags consistent with existing entity commands
- **Justification**: AI orchestrator agent depends on machine-readable output for automated sprint planning

### Security

**REQ-NF-006**: Sprint Data Access
- **Description**: Sprint data follows the same access patterns as other entities (no additional authentication required for local database; Turso auth for cloud)
- **Measurement**: Sprint commands work identically to task/epic/feature commands for both local and Turso backends
- **Target**: No new authentication mechanisms; sprint data inherits existing database security model
- **Justification**: Sprint management does not introduce new trust boundaries

---

*See also*: [Success Metrics](./success-metrics.md), [Scope](./scope.md)
