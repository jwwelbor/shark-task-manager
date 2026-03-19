# E19 Sprint Management & Planning System -- UAT Acceptance Plan

**Epic Key**: E19
**Date**: 2026-03-18
**Author**: QA Agent
**Status**: Active
**Version**: 1.0

---

## Table of Contents

1. [Acceptance Scenarios by User Journey](#1-acceptance-scenarios-by-user-journey)
2. [Success Metrics Validation Plan](#2-success-metrics-validation-plan)
3. [Cross-Epic Integration Test Scenarios](#3-cross-epic-integration-test-scenarios)
4. [Risk-Based Test Priorities](#4-risk-based-test-priorities)
5. [Persona-Based Acceptance Matrix](#5-persona-based-acceptance-matrix)

---

## 1. Acceptance Scenarios by User Journey

### Journey 1: Sprint Creation & Configuration

#### UAT-J1-01: Basic Sprint Creation

**Preconditions:**
- Shark project initialized with advanced workflow profile
- At least one epic with features and tasks exists
- No sprint with key `S001` exists

**Steps:**
1. Run `shark sprint create "Sprint 24" --start=2026-04-01 --end=2026-04-15`
2. Observe output

**Expected Outcomes:**
- Sprint is created with auto-assigned key `S001` (or next available `S###`)
- Sprint status is `planning`
- Confirmation message displays sprint key, name, dates, and status
- Sprint is retrievable via `shark sprint get S001`

**Pass Criteria:** Sprint exists in database with correct name, dates, and `planning` status.

---

#### UAT-J1-02: Sprint Creation with Goal

**Preconditions:** No conflicting sprint exists.

**Steps:**
1. Run `shark sprint create "Sprint 25" --start=2026-04-15 --end=2026-04-29 --goal="Complete E07-F34 template enrichment"`
2. Run `shark sprint get <key>`

**Expected Outcomes:**
- Sprint created with goal text stored
- `shark sprint get` output includes goal text

**Pass Criteria:** Goal text is persisted and displayed.

---

#### UAT-J1-03: Sprint Creation with Invalid Date Range

**Preconditions:** None.

**Steps:**
1. Run `shark sprint create "Bad Sprint" --start=2026-04-15 --end=2026-04-01`

**Expected Outcomes:**
- Command fails with descriptive error: start date must be before end date
- No sprint record is created

**Pass Criteria:** Validation rejects invalid date range with clear error message.

---

#### UAT-J1-04: Sprint Creation with JSON Output

**Preconditions:** None.

**Steps:**
1. Run `shark sprint create "Sprint JSON" --start=2026-05-01 --end=2026-05-15 --json`

**Expected Outcomes:**
- Output is valid JSON
- JSON contains: key, name, start_date, end_date, status, goal (null or empty), created_at
- JSON is parseable by standard tools (`jq`, Python `json.loads`)

**Pass Criteria:** AI orchestrator can consume the output programmatically.

---

#### UAT-J1-05: Agent Capacity Configuration

**Preconditions:** Sprint exists in `planning` status.

**Steps:**
1. Run `shark sprint capacity set S001 --agent=backend --points=21`
2. Run `shark sprint capacity set S001 --agent=frontend --points=13`
3. Run `shark sprint capacity set S001 --agent=qa --points=8`
4. Run `shark sprint capacity show S001`

**Expected Outcomes:**
- Each capacity set command confirms the update
- `capacity show` displays all three agent types with capacity points, allocated points (0 initially), and remaining points

**Pass Criteria:** Capacity targets are stored per agent type per sprint and displayed correctly.

---

#### UAT-J1-06: Sprint Verification via Get

**Preconditions:** Sprint with capacity configured (UAT-J1-05 completed).

**Steps:**
1. Run `shark sprint get S001`

**Expected Outcomes:**
- Displays: sprint name, dates, goal, status (`planning`), task count (0), capacity summary

**Pass Criteria:** All sprint parameters visible in a single command.

---

#### UAT-J1-07: Sprint Update

**Preconditions:** Sprint exists in `planning` status.

**Steps:**
1. Run `shark sprint update S001 --goal="Updated goal text" --end=2026-04-22`
2. Run `shark sprint get S001`

**Expected Outcomes:**
- Goal and end date are updated
- Updated values visible in `sprint get` output

**Pass Criteria:** Sprint attributes are mutable during `planning` status.

---

### Journey 2: Sprint Planning

#### UAT-J2-01: View Sprint Planning Data

**Preconditions:**
- Sprint in `planning` status with capacity configured
- Backlog contains at least 5 tasks in `ready_for_development` status, unassigned to any sprint

**Steps:**
1. Run `shark sprint plan S001`

**Expected Outcomes:**
- Displays three sections: (a) available backlog tasks sorted by priority, (b) current sprint allocation (0 tasks), (c) sprint readiness score
- Backlog only shows tasks in eligible statuses and not assigned to another sprint

**Pass Criteria:** Planning view surfaces all data needed for scoping decisions.

---

#### UAT-J2-02: Assign Individual Task to Sprint

**Preconditions:**
- Sprint in `planning` status
- Task `E07-F01-001` exists and is not assigned to any sprint

**Steps:**
1. Run `shark sprint add S001 E07-F01-001`
2. Run `shark sprint backlog S001`

**Expected Outcomes:**
- Task is assigned; confirmation shows task key and sprint key
- Task appears in sprint backlog view
- If task's agent type has capacity configured, capacity utilization updates

**Pass Criteria:** Task is associated with sprint; visible in backlog.

---

#### UAT-J2-03: Assign Task Already in Another Sprint (Error)

**Preconditions:**
- Task is already assigned to sprint S001 (planning or active status)

**Steps:**
1. Run `shark sprint add S002 <same-task-key>`

**Expected Outcomes:**
- Error message naming the conflicting sprint: "Task X is already assigned to sprint S001"
- Task remains in original sprint

**Pass Criteria:** System prevents double-assignment with informative error.

---

#### UAT-J2-04: Bulk Assign Tasks from Feature

**Preconditions:**
- Sprint in `planning` status
- Feature E07-F34 has at least 3 tasks in `ready_for_development` status, unassigned

**Steps:**
1. Run `shark sprint add S001 --bulk E07-F34`

**Expected Outcomes:**
- All eligible tasks from E07-F34 are assigned to the sprint
- Output shows count of tasks added
- Capacity utilization updates per agent type
- Tasks already in another sprint are skipped (not errored)

**Pass Criteria:** Bulk assignment works; ineligible tasks are skipped gracefully.

---

#### UAT-J2-05: Capacity Exceeded Warning (Advisory, Not Blocking)

**Preconditions:**
- Sprint has capacity for `backend` set to 5 points
- Sprint already has 4 points allocated to `backend`
- A priority-3 task with `agent=backend` exists

**Steps:**
1. Run `shark sprint add S001 <task-key>`

**Expected Outcomes:**
- Task is assigned (not blocked)
- Warning displayed: capacity exceeded (7/5 points for backend)
- Readiness score decreases to reflect overallocation

**Pass Criteria:** Capacity is advisory; task is still assigned with warning.

---

#### UAT-J2-06: Sprint Readiness Score

**Preconditions:**
- Sprint with assigned tasks, capacity configured

**Steps:**
1. Run `shark sprint readiness S001`

**Expected Outcomes:**
- Displays overall score (0-100)
- Breakdown shows individual factors: capacity utilization, dependency satisfaction, task count, agent balance
- Each factor shows its score and contribution

**Pass Criteria:** Score provides actionable quality signal for sprint planning.

---

#### UAT-J2-07: Remove Task from Sprint

**Preconditions:**
- Task is assigned to the sprint

**Steps:**
1. Run `shark sprint remove S001 E07-F01-001`
2. Run `shark sprint backlog S001`

**Expected Outcomes:**
- Task removed from sprint
- Task no longer appears in backlog
- Capacity utilization decreases

**Pass Criteria:** Task is disassociated from sprint cleanly.

---

#### UAT-J2-08: Start Sprint

**Preconditions:**
- Sprint in `planning` status with at least 1 task assigned
- No other sprint is currently `active`

**Steps:**
1. Run `shark sprint start S001`

**Expected Outcomes:**
- Sprint status transitions to `active`
- Confirmation displayed with sprint name and start date
- `shark sprint list --status=active` shows this sprint

**Pass Criteria:** Sprint is activated; constraint on single active sprint is respected.

---

#### UAT-J2-09: Start Sprint When Another is Active (Error)

**Preconditions:**
- Sprint S001 is in `active` status

**Steps:**
1. Run `shark sprint start S002`

**Expected Outcomes:**
- Error: "Cannot start S002; sprint S001 is currently active"
- S002 remains in `planning` status

**Pass Criteria:** Single active sprint constraint enforced with clear error.

---

#### UAT-J2-10: Sprint Planning via JSON (AI Orchestrator)

**Preconditions:**
- Sprint in `planning` status, backlog with tasks

**Steps:**
1. Run `shark sprint plan S001 --json`
2. Run `shark sprint readiness S001 --json`

**Expected Outcomes:**
- Both commands return valid, parseable JSON
- JSON includes structured arrays for backlog tasks, capacity data, and readiness breakdown
- `--field` flag works to extract individual fields

**Pass Criteria:** AI orchestrator can consume all planning data programmatically.

---

### Journey 3: Sprint Monitoring

#### UAT-J3-01: View Sprint Backlog During Execution

**Preconditions:**
- Sprint is `active`
- Tasks are in various statuses (some in development, some completed, some blocked)

**Steps:**
1. Run `shark sprint backlog S001`

**Expected Outcomes:**
- Tasks grouped by status category (planning, in-progress, review, qa, completed, blocked)
- Completion percentage shown (completed / total)
- Each task shows key, title, status, agent type, priority

**Pass Criteria:** Backlog provides actionable snapshot of sprint state.

---

#### UAT-J3-02: View Blocked Items

**Preconditions:**
- Sprint is `active`; at least 1 task is in `blocked` status

**Steps:**
1. Run `shark sprint backlog S001 --blocked`

**Expected Outcomes:**
- Only blocked tasks displayed
- Shows blocking reason and days blocked for each

**Pass Criteria:** PM can quickly identify and prioritize unblocking actions.

---

#### UAT-J3-03: Sprint Burndown

**Preconditions:**
- Sprint is `active` and has been running for at least 2 days
- Some tasks have been completed

**Steps:**
1. Run `shark sprint burndown S001`

**Expected Outcomes:**
- Day-by-day table showing ideal vs. actual remaining tasks
- Ideal burndown decreases linearly from total to zero
- Actual remaining reflects completed tasks
- Shows days remaining

**Pass Criteria:** PM can visually assess if sprint is on track.

---

#### UAT-J3-04: Burndown with JSON Output

**Preconditions:** Sprint active with some progress.

**Steps:**
1. Run `shark sprint burndown S001 --json`

**Expected Outcomes:**
- JSON array with daily data points: date, ideal_remaining, actual_remaining
- Data points cover sprint start_date through today (or end_date)

**Pass Criteria:** Data is consumable for external visualization.

---

#### UAT-J3-05: Velocity Trend

**Preconditions:**
- At least 3 completed sprints with historical data

**Steps:**
1. Run `shark sprint velocity`

**Expected Outcomes:**
- Shows completed tasks/points per sprint for the last 5 sprints
- Displays trailing average velocity
- Format is human-readable with sprint names and completion counts

**Pass Criteria:** PM has velocity trend context for sprint health assessment.

---

#### UAT-J3-06: Velocity with Insufficient Data

**Preconditions:**
- Fewer than 3 completed sprints exist

**Steps:**
1. Run `shark sprint velocity`

**Expected Outcomes:**
- Shows available data (1-2 sprints)
- Displays informational message: "Insufficient data for trend analysis. Need at least 3 completed sprints."
- No error or crash

**Pass Criteria:** Graceful degradation with helpful message.

---

#### UAT-J3-07: Mid-Sprint Scope Change

**Preconditions:**
- Sprint is `active`
- An urgent bug B042 exists, not assigned to any sprint

**Steps:**
1. Run `shark sprint add S001 B042`
2. Run `shark sprint backlog S001`

**Expected Outcomes:**
- Bug is assigned to the active sprint
- Warning about scope change and capacity impact displayed
- Bug appears in sprint backlog

**Pass Criteria:** Mid-sprint additions are supported with scope-change warnings.

---

### Journey 4: Sprint Close & Retrospective

#### UAT-J4-01: Pre-Close Sprint Summary

**Preconditions:**
- Sprint is `active` with mix of completed and incomplete tasks

**Steps:**
1. Run `shark sprint summary S001`

**Expected Outcomes:**
- Displays: completed task count, incomplete task count, completion percentage, velocity (this sprint), velocity comparison to trailing average (if available)

**Pass Criteria:** PM understands sprint outcomes before closing.

---

#### UAT-J4-02: Close Sprint with Carryover to Next Sprint

**Preconditions:**
- Sprint S001 is `active` with 2 incomplete tasks
- Sprint S002 exists in `planning` status

**Steps:**
1. Run `shark sprint close S001 --carryover=next`
2. Run `shark sprint get S001`
3. Run `shark sprint backlog S002`

**Expected Outcomes:**
- S001 transitions to `completed` status
- Completed tasks remain associated with S001 for velocity history
- 2 incomplete tasks appear in S002's backlog
- Final summary displayed

**Pass Criteria:** Carryover moves incomplete work to next sprint; completed work stays for metrics.

---

#### UAT-J4-03: Close Sprint with Carryover to Backlog

**Preconditions:**
- Sprint S001 is `active` with incomplete tasks

**Steps:**
1. Run `shark sprint close S001 --carryover=backlog`

**Expected Outcomes:**
- S001 transitions to `completed`
- Incomplete tasks are unassigned from any sprint (returned to general backlog)
- `shark sprint backlog S001` shows only completed tasks

**Pass Criteria:** Incomplete tasks cleanly returned to backlog.

---

#### UAT-J4-04: Close Sprint -- No Next Sprint Exists (Auto-Create)

**Preconditions:**
- Sprint S001 is `active`
- No other sprint exists in `planning` status
- `--carryover=next` is used

**Steps:**
1. Run `shark sprint close S001 --carryover=next`

**Expected Outcomes:**
- System creates a new sprint in `planning` status with dates following S001's pattern (start = S001.end_date + 1 day, same duration)
- Incomplete tasks are assigned to the newly created sprint
- User is informed of the auto-created sprint key and dates

**Pass Criteria:** Carryover is not blocked by absence of a next sprint.

---

#### UAT-J4-05: Detailed Sprint Retrospective Summary

**Preconditions:**
- Sprint S001 is in `completed` or `archived` status
- Sprint had tasks added and removed mid-sprint

**Steps:**
1. Run `shark sprint summary S001 --detailed`

**Expected Outcomes:**
- All basic summary data (planned vs completed, velocity)
- Additional data: tasks added mid-sprint, tasks removed mid-sprint, agent utilization vs. capacity, carryover task list
- Cycle-time-by-phase data if `work_sessions` available, otherwise informational message about E13 dependency

**Pass Criteria:** Retrospective has actionable data for process improvement discussions.

---

#### UAT-J4-06: Archive Sprint

**Preconditions:**
- Sprint in `completed` status

**Steps:**
1. Run `shark sprint archive S001`
2. Run `shark sprint list`
3. Run `shark sprint list --status=archived`

**Expected Outcomes:**
- Sprint transitions to `archived` status
- Archived sprint excluded from default `sprint list` output
- Archived sprint visible with `--status=archived` filter
- Sprint data preserved for velocity calculations

**Pass Criteria:** Archival cleans active views without losing historical data.

---

#### UAT-J4-07: Sprint Summary with JSON Output

**Preconditions:** Completed sprint.

**Steps:**
1. Run `shark sprint summary S001 --json`
2. Run `shark sprint summary S001 --detailed --json`

**Expected Outcomes:**
- Valid JSON for both basic and detailed summaries
- JSON includes: planned_count, completed_count, completion_percentage, velocity, carryover list

**Pass Criteria:** AI orchestrator and external tools can process retrospective data.

---

#### UAT-J4-08: Sprint Delete Restrictions

**Preconditions:**
- Sprint S001 in `completed` status
- Sprint S002 in `planning` status

**Steps:**
1. Run `shark sprint delete S001` (completed)
2. Run `shark sprint delete S002` (planning)

**Expected Outcomes:**
- Deleting S001 fails with error: "Cannot delete completed sprint. Use archive instead."
- Deleting S002 succeeds (planning sprints can be deleted)

**Pass Criteria:** Completed sprint data is protected from accidental deletion.

---

### Additional Edge Case Scenarios

#### UAT-EDGE-01: Empty Sprint Close

**Preconditions:** Sprint in `active` status with zero tasks assigned.

**Steps:**
1. Run `shark sprint close S001 --carryover=backlog`

**Expected Outcomes:**
- Sprint closes successfully
- Summary shows 0 planned, 0 completed, 0 carryover
- No errors

**Pass Criteria:** Empty sprint does not crash or produce invalid data.

---

#### UAT-EDGE-02: All Tasks Completed Before Close

**Preconditions:** Sprint `active`, all assigned tasks in `completed` status.

**Steps:**
1. Run `shark sprint close S001 --carryover=next`

**Expected Outcomes:**
- Sprint closes with 100% completion
- No carryover items
- Velocity = total completed tasks

**Pass Criteria:** Full completion is the happy case; no carryover needed.

---

#### UAT-EDGE-03: Sprint with Very Long Duration (>30 days)

**Preconditions:** None.

**Steps:**
1. Run `shark sprint create "Long Sprint" --start=2026-04-01 --end=2026-06-01`

**Expected Outcomes:**
- Sprint is created (not blocked)
- Warning displayed about duration exceeding 30 days
- Burndown chart would have 60+ data points but still functions

**Pass Criteria:** System warns but does not block unusual durations.

---

#### UAT-EDGE-04: Sprint List and Filtering

**Preconditions:** Multiple sprints in various statuses (planning, active, completed, archived).

**Steps:**
1. Run `shark sprint list`
2. Run `shark sprint list --status=planning`
3. Run `shark sprint list --status=active`
4. Run `shark sprint list --json`

**Expected Outcomes:**
- Default list excludes archived sprints
- Status filter shows only matching sprints
- JSON output includes all sprint metadata

**Pass Criteria:** Sprint listing is filterable and consistent.

---

#### UAT-EDGE-05: Auto-Detect Sprint Key in Core Commands

**Preconditions:** Sprint S001 exists.

**Steps:**
1. Run `shark get S001`

**Expected Outcomes:**
- `shark get` auto-detects `S###` as a sprint key
- Displays sprint details identical to `shark sprint get S001`

**Pass Criteria:** Core command auto-detection works for sprint keys.

---

#### UAT-EDGE-06: Invalid Sprint Status Transition

**Preconditions:** Sprint in `completed` status.

**Steps:**
1. Run `shark sprint start S001`

**Expected Outcomes:**
- Error: "Invalid status transition from 'completed' to 'active'"
- Sprint status unchanged

**Pass Criteria:** Invalid transitions rejected with descriptive error.

---

## 2. Success Metrics Validation Plan

### Metric 1: Sprint Feature Adoption

| Attribute | Value |
|-----------|-------|
| **Measurement Method** | Count distinct sprint lifecycle events (create, start, close) per iteration cycle |
| **Data Source** | Sprint database records: count sprints that reached `completed` status within each 2-week window |
| **Validation Query** | `SELECT COUNT(*) FROM sprints WHERE status IN ('completed','archived') AND updated_at > date('now','-30 days')` |
| **Pass Threshold** | At least 1 complete lifecycle (create -> start -> close) per iteration cycle within 2 sprints of launch |
| **Fail Threshold** | Zero completed sprints after 2 full iteration cycles post-launch |
| **When to Measure** | 2 sprints post-launch (approximately 4-8 weeks) |
| **UAT Validation** | During UAT, execute a complete sprint lifecycle (UAT-J1 through UAT-J4) to prove the workflow is functional end-to-end. Adoption is a post-launch metric; UAT validates the capability exists. |

---

### Metric 2: Sprint Planning Efficiency

| Attribute | Value |
|-----------|-------|
| **Measurement Method** | Time delta from sprint creation (`created_at`) to sprint start (`status` changes to `active`) |
| **Data Source** | `sprints.created_at` vs. timestamp when status becomes `active` (from sprint history or `updated_at` at `active` transition) |
| **Validation Query** | Compare timestamps for sprints that transitioned from `planning` to `active`. Can be reconstructed from `sprint_history` if implemented, or from `updated_at` if status transitions are the only updates. |
| **Pass Threshold** | Planning-to-active transition under 30 minutes wall-clock time (for pre-prioritized backlog) |
| **Fail Threshold** | Planning takes longer than the informal process it replaces (estimated at 60 minutes) |
| **When to Measure** | 3 sprints post-launch |
| **UAT Validation** | Execute UAT-J1 and UAT-J2 sequences end-to-end. Measure total time to create sprint, configure capacity, assign tasks, and start sprint. Target: under 10 minutes for a sprint with 15-20 pre-prioritized tasks. |

---

### Metric 3: Velocity Prediction Accuracy

| Attribute | Value |
|-----------|-------|
| **Measurement Method** | `|predicted_velocity - actual_velocity| / predicted_velocity * 100` |
| **Data Source** | `shark sprint velocity --json` (trailing average as prediction) and `shark sprint summary --json` (actual completed) |
| **Validation Script** | After 5+ completed sprints: extract trailing-3-sprint average at planning time, compare to actual completed count at close. Compute variance percentage. |
| **Pass Threshold** | Prediction variance under 15% after 5 completed sprints |
| **Minimum Viable** | Variance under 25% after 5 sprints |
| **Fail Threshold** | Variance exceeds 30% after 5 completed sprints, indicating velocity calculation is unreliable |
| **When to Measure** | 5 sprints post-launch (approximately 10-20 weeks) |
| **UAT Validation** | Cannot fully validate in UAT (requires historical data). Validate that: (1) velocity command returns correct data for completed sprints (UAT-J3-05), (2) velocity correctly handles insufficient data (UAT-J3-06), (3) summary shows velocity comparison (UAT-J4-01). |

---

### Metric 4: Sprint Completion Rate

| Attribute | Value |
|-----------|-------|
| **Measurement Method** | `completed_tasks / (completed_tasks + carryover_tasks) * 100` at sprint close |
| **Data Source** | `shark sprint summary --json` output at each sprint close |
| **Validation Query** | From sprint summary JSON: `completion_percentage` field, averaged over trailing 3 sprints |
| **Pass Threshold** | Average completion rate above 80% across trailing 3 sprints |
| **Minimum Viable** | Average above 60% |
| **Fail Threshold** | Average below 50% after 5 sprints (indicates systematic overcommitment) |
| **When to Measure** | 5 sprints post-launch |
| **UAT Validation** | Validate that sprint close correctly calculates completion percentage (UAT-J4-01, UAT-J4-02). Validate with known data: close sprint with 8/10 tasks completed, verify 80% completion rate in summary. |

---

### Metric 5: Agent Capacity Utilization

| Attribute | Value |
|-----------|-------|
| **Measurement Method** | `allocated_points / capacity_points * 100` per agent type at sprint start |
| **Data Source** | `shark sprint capacity show --json` |
| **Pass Threshold** | No agent type exceeds 110% capacity at sprint start; standard deviation across agent types under 20 percentage points |
| **When to Measure** | Per sprint cycle |
| **UAT Validation** | Configure capacity for 3 agent types (UAT-J1-05), assign tasks until one exceeds capacity (UAT-J2-05), verify warning is displayed and capacity show reflects overallocation. |

---

### Metric 6: AI Orchestrator Sprint Automation

| Attribute | Value |
|-----------|-------|
| **Measurement Method** | Count of successful programmatic sprint planning sequences via `--json` commands |
| **Data Source** | Orchestrator logs; automated test scripts executing the full plan sequence |
| **Pass Threshold** | Orchestrator can auto-plan a sprint without error in 90% of attempts |
| **Minimum Viable** | All `--json` commands return expected data shapes |
| **When to Measure** | Immediately post-launch (capability test) |
| **UAT Validation** | Execute full orchestrator workflow using `--json` flags: create sprint (UAT-J1-04), get planning data (UAT-J2-10), add tasks, check readiness (UAT-J2-06 with `--json`), report results. Validate every command produces valid JSON. |

---

### Backward Compatibility (Implicit Metric)

| Attribute | Value |
|-----------|-------|
| **Measurement Method** | Full existing test suite passes after sprint schema migration |
| **Pass Threshold** | Zero test failures; zero regressions in existing commands |
| **Validation** | Run `make test` after migration is applied. Verify `shark task list`, `shark epic list`, `shark status`, `shark analytics` all produce identical output to pre-migration baseline. |

---

## 3. Cross-Epic Integration Test Scenarios

### INT-01: Sprint Assignment with Bugs (E18)

**Interaction**: Bugs (`B###`) created by E18 should be assignable to sprints.

**Preconditions:**
- Bug B001 exists
- Sprint S001 in `planning` status

**Steps:**
1. Run `shark sprint add S001 B001`
2. Run `shark sprint backlog S001`
3. Close sprint; verify bug is included in velocity if completed

**Expected Outcomes:**
- Bug is assigned via polymorphic assignment (entity_type='bug')
- Bug appears in sprint backlog alongside tasks
- Completed bugs count toward sprint velocity

**Pass Criteria:** Polymorphic assignment works for bugs.

---

### INT-02: Sprint Assignment with Change-Cards (E18)

**Interaction**: Change-cards (`CC-###`) created by E18 should be assignable to sprints.

**Preconditions:** Change-card CC-001 exists; sprint in planning.

**Steps:**
1. Run `shark sprint add S001 CC-001`
2. Verify in backlog

**Expected Outcomes:** Change-card is assigned and visible in sprint backlog.

**Pass Criteria:** Polymorphic assignment works for change-cards.

---

### INT-03: Sprint Backlog Respects Workflow Status Phases (E16)

**Interaction**: Sprint backlog grouping must use E16's workflow profile status phase definitions.

**Preconditions:**
- Advanced workflow profile is active
- Sprint has tasks in various statuses (draft, in_development, ready_for_code_review, in_qa, completed, blocked)

**Steps:**
1. Run `shark sprint backlog S001`

**Expected Outcomes:**
- Tasks are grouped by phase (planning, development, review, qa, completed, blocked)
- Phase groupings match the workflow profile's `status_metadata` phase definitions
- Tasks are not miscategorized

**Pass Criteria:** Backlog display aligns with configured workflow phases.

---

### INT-04: Sprint Task Display in Task Get (E10 Context)

**Interaction**: `shark get <task-key>` should show sprint assignment context when a task is assigned to a sprint.

**Preconditions:** Task is assigned to sprint S001.

**Steps:**
1. Run `shark get E07-F01-001`
2. Run `shark get E07-F01-001 --json`

**Expected Outcomes:**
- Task display includes sprint info (sprint key, sprint name, sprint status)
- JSON output includes `sprint` field (or `sprint_assignment` field)

**Pass Criteria:** Sprint context is visible in existing task views.

---

### INT-05: Sprint with Task Resume Context (E10)

**Interaction**: `shark task resume` should include sprint context when the task is assigned to a sprint.

**Preconditions:** Task is assigned to active sprint S001.

**Steps:**
1. Run `shark task resume E07-F01-001`

**Expected Outcomes:**
- Resume output includes sprint assignment info: which sprint, sprint status, sprint progress, days remaining

**Pass Criteria:** Agent resuming work has sprint context.

---

### INT-06: Sprint Commands Follow Service Layer Pattern (E15)

**Interaction**: All sprint commands must use the service layer, not direct repository access.

**Validation Method:**
- Code review: verify `internal/cli/commands/sprint.go` calls `cli.GetSprintService()` and never imports `repository` package directly
- Service tests: verify `internal/services/sprint_service_test.go` uses mocked repositories (no real database)

**Pass Criteria:** Sprint implementation follows E15 architecture; no fat controllers.

---

### INT-07: Sprint Key Auto-Detection in Core Commands (E17)

**Interaction**: `shark get S001`, `shark status S001`, `shark status advance S001` should auto-detect the sprint entity type.

**Steps:**
1. Run `shark get S001`
2. Run `shark status S001`

**Expected Outcomes:**
- Key format `S###` is recognized as sprint
- Appropriate sprint data is returned

**Pass Criteria:** Key auto-detection extends to sprints.

---

### INT-08: Sprint Schema Migration Backward Compatibility

**Interaction**: Adding sprint tables (schema version 7) must not break existing data or commands.

**Steps:**
1. Record output of `shark task list`, `shark epic list`, `shark feature list`, `shark analytics` before migration
2. Apply sprint migration (bump schema version, run shark command)
3. Repeat the same commands

**Expected Outcomes:**
- Identical output for all existing commands
- All existing tests pass (`make test`)
- No data loss or corruption

**Pass Criteria:** Zero regressions from schema migration.

---

### INT-09: Sprint Analytics Use task_history (Not E13 work_sessions)

**Interaction**: Core analytics (velocity, burndown) must work without E13's work_sessions data.

**Preconditions:**
- E13 is NOT implemented (work_sessions table is empty)
- 3+ completed sprints exist

**Steps:**
1. Run `shark sprint velocity`
2. Run `shark sprint burndown S001`
3. Run `shark sprint summary S001 --detailed`

**Expected Outcomes:**
- Velocity correctly counts completed tasks per sprint from sprint_assignments
- Burndown reconstructs from task_history status transitions
- `--detailed` shows informational message about unavailable cycle-time data instead of error

**Pass Criteria:** Sprint analytics are fully functional without E13.

---

### INT-10: Existing `shark status` Dashboard with Sprint Info (E04/E17)

**Interaction**: If REQ-F-017 (Could Have) is implemented, `shark status` dashboard shows active sprint info.

**Steps:**
1. Start a sprint (make it active)
2. Run `shark status`

**Expected Outcomes:**
- Dashboard includes active sprint name, dates, progress percentage, days remaining

**Pass Criteria:** Sprint info integrated into existing dashboard view.

---

## 4. Risk-Based Test Priorities

Based on the research report's risk assessment and feasibility review findings, scenarios are prioritized by risk severity:

### Priority 1 -- CRITICAL (Must Pass Before Any Release)

These scenarios address the highest risks identified in the research and feasibility reviews.

| Scenario | Risk Addressed | Source |
|----------|---------------|--------|
| UAT-J4-02: Close Sprint with Carryover to Next | R5: Sprint carryover edge cases (Medium probability, Medium impact) | Research Risk R5 |
| UAT-J4-03: Close Sprint with Carryover to Backlog | R5: Carryover logic complexity | Research Risk R5 |
| UAT-J4-04: Close Sprint -- No Next Sprint (Auto-Create) | R5: Edge case -- no target sprint for carryover | Research Risk R5 |
| UAT-EDGE-01: Empty Sprint Close | R5: Edge case -- zero tasks | Research Risk R5 |
| UAT-EDGE-02: All Tasks Completed Before Close | R5: Edge case -- 100% completion | Research Risk R5 |
| INT-08: Schema Migration Backward Compatibility | REQ-NF-004 (backward compat is non-negotiable) | Tech Feasibility NFR |
| UAT-J2-03: Assign Task Already in Another Sprint | R2: Polymorphic assignment constraint enforcement | Research Risk R2 |
| INT-09: Analytics Without E13 | R1: E13 dependency -- graceful degradation | Research Risk R1 (Highest rated risk) |

**Rationale:** Carryover is the most complex operation (identified by both research and tech review as highest-complexity requirement). Schema migration must not break existing functionality. Polymorphic assignment constraints prevent data corruption. E13 graceful degradation prevents the most likely failure mode.

### Priority 2 -- HIGH (Must Pass Before Feature Release)

| Scenario | Risk Addressed | Source |
|----------|---------------|--------|
| UAT-J1-01: Basic Sprint Creation | Core lifecycle -- foundation for everything | REQ-F-001 |
| UAT-J1-03: Invalid Date Range | Data validation -- prevents invalid state | REQ-F-001 AC4 |
| UAT-J2-08: Start Sprint | Core lifecycle -- single active sprint constraint | REQ-F-002 |
| UAT-J2-09: Start Sprint When Another Active | R7: Single active sprint constraint | Research Risk R7 |
| UAT-J3-01: View Sprint Backlog | Core monitoring capability | REQ-F-005 |
| UAT-J3-05: Velocity Trend | Core analytics | REQ-F-007 |
| UAT-J3-06: Velocity with Insufficient Data | Graceful degradation pattern | REQ-F-007 AC5 |
| UAT-J4-01: Pre-Close Sprint Summary | Data accuracy for planning decisions | REQ-F-009 |
| INT-01: Sprint Assignment with Bugs | R2: Polymorphic assignment design | Research Risk R2 |
| INT-03: Sprint Backlog Respects Workflow Phases | E16 integration correctness | Tech Review 1.5 |
| UAT-EDGE-06: Invalid Status Transition | Status lifecycle integrity | REQ-F-002 AC5 |

### Priority 3 -- MEDIUM (Should Pass Before Release)

| Scenario | Risk Addressed | Source |
|----------|---------------|--------|
| UAT-J1-04: Sprint Creation with JSON | AI orchestrator compatibility | REQ-NF-005 |
| UAT-J1-05: Agent Capacity Configuration | Capacity model correctness | REQ-F-014 |
| UAT-J2-01: View Sprint Planning Data | Planning view completeness | REQ-F-011 |
| UAT-J2-04: Bulk Assign Tasks | Planning efficiency | REQ-F-012 |
| UAT-J2-05: Capacity Exceeded Warning | R4: Scope creep risk -- capacity is advisory | Research Risk R4 |
| UAT-J2-06: Sprint Readiness Score | Planning quality signal | REQ-F-013 |
| UAT-J2-10: Sprint Planning via JSON | AI orchestrator planning flow | Metric 6 |
| UAT-J3-03: Sprint Burndown | R3: Burndown rendering quality | Research Risk R3 |
| UAT-J4-05: Detailed Retrospective Summary | E13 graceful degradation (detailed path) | Research Risk R1 |
| INT-02: Sprint Assignment with Change-Cards | Polymorphic completeness | E18 interaction |
| INT-04: Sprint Task Display in Task Get | Context enrichment | Tech Review 2.2 |

### Priority 4 -- LOW (Nice to Have Before Release)

| Scenario | Risk Addressed | Source |
|----------|---------------|--------|
| UAT-J1-02: Sprint Creation with Goal | Goal text persistence | REQ-F-001 |
| UAT-J1-06: Sprint Verification via Get | Display completeness | REQ-F-003 |
| UAT-J1-07: Sprint Update | Attribute mutation | REQ-F-003 |
| UAT-J2-07: Remove Task from Sprint | Removal workflow | REQ-F-004 |
| UAT-J3-02: View Blocked Items | Filtered backlog | REQ-F-005 |
| UAT-J3-04: Burndown with JSON | JSON output for burndown | REQ-F-008 |
| UAT-J3-07: Mid-Sprint Scope Change | Scope change tracking | Journey 3 Alt Path A |
| UAT-J4-06: Archive Sprint | Archive lifecycle | REQ-F-003 |
| UAT-J4-07: Sprint Summary JSON | JSON retrospective output | REQ-F-009 |
| UAT-J4-08: Sprint Delete Restrictions | Data protection | Scope edge case |
| UAT-EDGE-03: Long Sprint Duration Warning | Edge case warning | Scope edge case |
| UAT-EDGE-04: Sprint List and Filtering | List completeness | REQ-F-003 |
| UAT-EDGE-05: Auto-Detect Sprint Key | E17 integration | INT-07 |
| INT-05: Sprint with Task Resume Context | Context enrichment | E10 interaction |
| INT-06: Service Layer Code Review | Architecture compliance | E15 |
| INT-07: Key Auto-Detection | CLI simplification | E17 |
| INT-10: Dashboard Sprint Info | Dashboard integration | REQ-F-017 (Could Have) |

---

## 5. Persona-Based Acceptance Matrix

### PM / Scrum Master (Human)

The PM persona needs to run an entire sprint lifecycle using only `shark sprint` commands. Sprint planning should take under 10 minutes. Retrospectives should be data-driven.

| Need | Scenarios That Validate | Journey | Priority |
|------|------------------------|---------|----------|
| Create and configure sprints quickly | UAT-J1-01, UAT-J1-02, UAT-J1-05, UAT-J1-06, UAT-J1-07 | J1 | High |
| Scope sprints with capacity awareness | UAT-J2-01, UAT-J2-02, UAT-J2-04, UAT-J2-05, UAT-J2-06 | J2 | Medium-High |
| Start sprints with guardrails | UAT-J2-08, UAT-J2-09 | J2 | High |
| Monitor sprint progress daily | UAT-J3-01, UAT-J3-02, UAT-J3-03 | J3 | High |
| Assess sprint health via velocity trends | UAT-J3-05, UAT-J3-06 | J3 | High |
| Handle mid-sprint scope changes | UAT-J3-07, UAT-J2-07 | J3 | Medium |
| Close sprints with carryover handling | UAT-J4-02, UAT-J4-03, UAT-J4-04 | J4 | Critical |
| Run data-driven retrospectives | UAT-J4-01, UAT-J4-05 | J4 | High |
| Archive completed sprints | UAT-J4-06 | J4 | Low |
| Forecast using historical velocity | UAT-J3-05 (with 3+ sprints) | J3 | High |

**PM Acceptance Gate:** The PM can execute a complete sprint lifecycle (create -> configure capacity -> assign tasks -> start -> monitor backlog and burndown -> close with carryover -> review summary) using only `shark sprint` commands in a single terminal session, in under 15 minutes.

---

### AI Orchestrator Agent

The AI orchestrator needs deterministic, machine-readable JSON output for all sprint operations. It must be able to auto-plan sprints within capacity constraints.

| Need | Scenarios That Validate | Journey | Priority |
|------|------------------------|---------|----------|
| Create sprints programmatically | UAT-J1-04 | J1 | High |
| Query backlog and capacity via JSON | UAT-J2-10 | J2 | High |
| Assign tasks within capacity constraints | UAT-J2-02, UAT-J2-04, UAT-J2-05 | J2 | High |
| Check readiness score programmatically | UAT-J2-06 (with `--json`) | J2 | Medium |
| Get sprint status and progress via JSON | UAT-J3-04, UAT-J4-07 | J3, J4 | Medium |
| Detect overallocation | UAT-J2-05 | J2 | High |
| All commands support `--json` and `--field` | UAT-J1-04, UAT-J2-10, UAT-J3-04, UAT-J4-07 | All | High |
| Error messages are parseable | UAT-J2-03, UAT-J2-09, UAT-EDGE-06 | J2 | Medium |

**Orchestrator Acceptance Gate:** A scripted sequence of `shark sprint` commands with `--json` flags can: create a sprint, query the backlog, assign the top N tasks by priority up to capacity limits, check readiness score, and produce a structured planning report -- all without human intervention and without any command returning non-JSON output or a non-zero exit code for valid operations.

---

### Developer Agent (Secondary)

| Need | Scenarios That Validate | Priority |
|------|------------------------|----------|
| See assigned tasks scoped to current sprint | UAT-J3-01 (sprint backlog view) | Medium |
| Understand sprint context when resuming work | INT-05 (task resume with sprint context) | Low |
| Know sprint deadline | UAT-J1-06 (sprint get shows dates) | Low |

---

### QA Agent (Secondary)

| Need | Scenarios That Validate | Priority |
|------|------------------------|----------|
| See QA-phase tasks in sprint backlog | UAT-J3-01 (grouped by status phase) | Medium |
| Sprint summary includes QA metrics | UAT-J4-05 (detailed summary) | Low |
| Verify sprint does not break existing QA workflows | INT-08 (backward compatibility) | Critical |

---

## Exit Gate Checklist

- [x] Every user journey (J1-J4) has at least one acceptance scenario -- **28 scenarios across 4 journeys plus 6 edge cases**
- [x] Every success metric (Metrics 1-6 plus backward compatibility) has a validation method with pass/fail thresholds
- [x] Risk areas from research (R1-R7) and feasibility reviews have targeted scenarios -- **all 7 risks mapped to specific test scenarios in Priority section**
- [x] Plan is actionable for feature-level decomposition -- **scenarios are grouped by journey and requirement, enabling direct mapping to feature test strategies**
- [x] Cross-epic integration scenarios cover E13, E15, E16, E17, E18 interactions -- **10 integration scenarios**
- [x] Both primary personas (PM and AI Orchestrator) have acceptance gates defined
- [x] Secondary personas (Developer Agent, QA Agent) have relevant scenarios mapped

---

*UAT Acceptance Plan Complete -- 2026-03-18*
