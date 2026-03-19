# E19 Sprint Management & Planning System -- Research Report

**Epic Key**: E19
**Date**: 2026-03-18
**Status**: Complete
**Researcher**: AI Researcher Agent

---

## Executive Summary

E19 proposes adding sprints as a first-class entity to Shark Task Manager with lifecycle management, task assignment, capacity tracking, and analytics (velocity, burndown, summary). This research finds the epic is **technically feasible** with no blocking issues. The codebase has well-established patterns for adding new entity types (recently demonstrated by E18 with bugs and change-cards), and the proposed schema is straightforward additive work. The primary risk is the E13 dependency for session-based analytics, which is currently in `draft` status and has not started implementation. A phased delivery strategy is recommended to decouple core sprint management from session-dependent analytics.

---

## (1) Market / Competitive Landscape

### How Comparable Tools Handle Sprints

**Jira / Linear / Shortcut**: Full sprint management with web UI boards, drag-and-drop planning, velocity charts, and burndown. These are GUI-first tools with API access. Shark differentiates by being CLI-first and AI-agent-friendly.

**GitHub Projects**: Iteration fields on project boards. No velocity tracking, no capacity model. Lightweight but limited analytics.

**Taskwarrior / Todo.txt**: No sprint concept at all. Pure task tracking without iteration planning. This is Shark's current state.

**Plane.so (open source)**: Sprints with cycles, estimates, and analytics. Web-based. No CLI interface.

### Key Observations

1. **Sprint management is table-stakes for project planning tools.** Every serious project management tool has some form of iteration/sprint support. Shark's lack of this is the largest functional gap identified in the AI PM Journey Map analysis.

2. **CLI-first sprint management is a differentiator.** No major tool offers sprint lifecycle management entirely through CLI. This positions Shark uniquely for AI orchestrator workflows where programmatic sprint planning is essential.

3. **Capacity-based planning with agent types is novel.** Most tools track capacity per person. Shark's agent-type capacity model (backend, frontend, qa) is architecturally distinct and well-suited to AI-augmented teams where agents are role-based rather than individual-based.

4. **Text-based burndown is adequate for CLI.** Tools like `spark` (shell sparklines) and taskwarrior's charting show that ASCII/Unicode terminal charts are usable for trend data. Full graphical burndown can be deferred to a future web dashboard epic.

### Relevance to E19 Requirements

The PRD's scope boundaries are well-aligned with market positioning. Excluding web UI, story point estimation ceremonies, and cross-team coordination keeps E19 focused on the CLI-first, AI-agent-friendly niche. The "Could Have" items (auto-creation, dashboard integration) match common patterns in competing tools and are sensible stretch goals.

---

## (2) Feasibility Assessment by Requirement Area

### Sprint Lifecycle Management (REQ-F-001 through REQ-F-003) -- FEASIBLE, LOW RISK

**Assessment**: Straightforward. The codebase has proven patterns for entity lifecycle management.

**Evidence**:
- E18 added bugs (`B###` key format) and change-cards (`CC-###`) with full CRUD, status transitions, and CLI commands. The `S###` key format follows the same standalone entity pattern.
- The `internal/keys/service.go` already handles `EntityTypeBug` and `EntityTypeChange` with regex-based key parsing. Adding `EntityTypeSprint` requires ~20 lines of pattern matching.
- The service layer pattern is mature: `BugService`, `ChangeCardService` in `internal/services/` provide templates. A `SprintService` follows the same constructor injection pattern.
- The `internal/db/db.go` migration system supports idempotent schema additions. Current schema version is 6. Adding sprint tables bumps to 7.

**Complexity**: Medium (M). Multiple commands but each follows established patterns.

### Task-to-Sprint Assignment (REQ-F-004 through REQ-F-006) -- FEASIBLE, MEDIUM RISK

**Assessment**: Feasible with careful constraint design. The many-to-one assignment (task belongs to at most one active/planning sprint) requires a partial unique index.

**Evidence**:
- SQLite supports partial indexes: `CREATE UNIQUE INDEX idx_active_assignment ON sprint_assignments(task_id) WHERE removed_at IS NULL;` -- this enforces the single-sprint constraint.
- The `sprint_assignments` table with soft-delete (`removed_at` column) preserves historical data for velocity calculations while allowing reassignment.
- Existing `task_relationships` table shows the join-table pattern is well-understood in the codebase.

**Risk**: The carryover logic (REQ-F-006) is the most complex requirement. Closing a sprint with `--carryover=next` must atomically: (1) mark current sprint as closing, (2) create or identify the next sprint, (3) reassign incomplete tasks, (4) generate completion record, (5) transition to completed. This requires a multi-step transaction managed by the service layer, which is the established pattern.

**Complexity**: Large (L). The carryover logic and constraint enforcement need careful implementation and testing.

### Sprint Analytics (REQ-F-007 through REQ-F-009) -- FEASIBLE, DEPENDENCY RISK

**Assessment**: Core velocity and completion metrics are feasible without E13. Detailed cycle-time-by-phase analytics (REQ-F-009 `--detailed`) depend on the `work_sessions` table from E13.

**Evidence**:
- **Velocity** (REQ-F-007): Requires counting completed tasks per sprint from `sprint_assignments` joined with `tasks` where `status = 'completed'`. This is a simple aggregation query -- no dependency on E13.
- **Burndown** (REQ-F-008): Requires daily snapshots of remaining tasks. Two approaches: (a) query `task_history` for status changes during the sprint period, or (b) store daily snapshots in a new table. Approach (a) uses existing infrastructure. The `task_history` table already records status transitions with timestamps, which is sufficient for reconstructing daily remaining-task counts.
- **Sprint Summary** (REQ-F-009): Basic summary (planned vs. completed, velocity, carryover) is feasible. The `--detailed` flag's "average cycle time by phase" requires per-phase duration data that comes from `work_sessions` (E13's `task_sessions` / `work_sessions` table). The `work_sessions` table schema already exists in the database (migration added in version 5), but the commands to populate it are part of E13 which is in `draft` status.

**Risk**: If E13 is not completed, the `--detailed` cycle-time-by-phase analytics will have no data. Recommendation: implement the `--detailed` flag but show "insufficient data" when work_sessions are empty, matching the existing pattern for velocity with <3 sprints.

**Complexity**: Large (L). Analytics queries, burndown reconstruction, and text-based chart rendering.

### Database Schema (REQ-F-010) -- FEASIBLE, LOW RISK

**Assessment**: Additive schema changes with no impact on existing tables.

**Evidence**:
- The three proposed tables (`sprints`, `sprint_assignments`, `sprint_capacity`) follow established patterns.
- The `sprints` table mirrors the structure of `bugs` and `change_cards` (id, key, name, status, slug, file_path, timestamps).
- Foreign key constraints on `sprint_assignments` reference `sprints(id)` and `tasks(id)` -- both existing tables.
- The migration pattern in `internal/db/db.go` is well-established with 22+ existing migrations. Adding 3 tables and ~10 indexes is routine.

**Complexity**: Small (S). Schema definition only; straightforward migration.

### Sprint Planning View (REQ-F-011 through REQ-F-013) -- FEASIBLE, MEDIUM RISK

**Assessment**: The planning view aggregates data from multiple sources (backlog tasks, capacity, readiness score). This requires a composite query but no new infrastructure.

**Evidence**:
- Backlog query: `SELECT * FROM tasks WHERE status IN (...) AND id NOT IN (SELECT task_id FROM sprint_assignments WHERE removed_at IS NULL)` -- standard SQL, uses existing indexes.
- Capacity utilization: aggregate assigned task priorities grouped by agent_type against sprint_capacity targets.
- Readiness score: computed in the service layer from capacity, dependency, and assignment data. The `internal/status/` package already computes similar composite scores (health indicators, progress weights).

**Complexity**: Medium (M). Multiple queries composed into a single view.

### Capacity Management (REQ-F-014 through REQ-F-015) -- FEASIBLE, LOW RISK

**Assessment**: CRUD operations on `sprint_capacity` table plus configuration defaults in `.sharkconfig.json`.

**Evidence**:
- The `.sharkconfig.json` already supports custom sections (status_metadata, workflow profiles, sprint_defaults would follow the same pattern).
- The `internal/config/` package handles JSON configuration loading and validation.

**Complexity**: Small (S). Standard CRUD with configuration integration.

### Non-Functional Requirements -- FEASIBLE, LOW RISK

**Assessment**: All NFRs are achievable within existing patterns.

- **Performance (REQ-NF-001, REQ-NF-002)**: SQLite with proper indexes handles the proposed query patterns efficiently. The 50-sprint / 1000-task target is well within SQLite's capabilities.
- **Backward Compatibility (REQ-NF-004)**: Schema changes are additive (new tables only). No existing table modifications. All existing commands and tests are unaffected.
- **JSON Output (REQ-NF-005)**: All entity commands already support `--json` and `--field`. Sprint commands follow the same pattern via `cli.OutputJSON()`.
- **Security (REQ-NF-006)**: No new authentication mechanisms. Sprint data uses existing database access patterns.

---

## (3) System-Wide Impact -- Interactions with Other Epics

### Direct Dependencies

| Epic | Relationship | Impact | Mitigation |
|------|-------------|--------|------------|
| **E13** (Workflow-Aware Task Command System) | E19 depends on E13 for `work_sessions` data used in sprint summary `--detailed` analytics | E13 is in `draft` status with 0% progress. If E13 is not completed before E19, cycle-time-by-phase analytics will have no data | Implement `--detailed` with graceful degradation: show "insufficient data" when work_sessions are empty. Core sprint features (velocity, burndown from task_history) work independently |
| **E15** (Service Layer Architecture Refactoring) | E19 should follow E15's service layer patterns | E15 is `active` at 79% progress. The service layer pattern is mature enough to use. Sprint commands should use the service layer exclusively (no fat controllers) | Follow established patterns from `BugService`, `ChangeCardService`. No blocker |
| **E16** (Multi-Level Workflow) | E19's task-to-sprint assignment interacts with multi-level workflow status transitions | E16 is `active` at 96%. Sprint backlog views must respect the active workflow profile's status groupings | Use `workflow.Service` to determine status phase groupings for backlog display. No conflict |

### Indirect Interactions

| Epic | Interaction | Notes |
|------|-------------|-------|
| **E04** (CLI Core) | Sprint commands register with the existing CLI framework | Standard `init()` registration. No conflict |
| **E07** (Enhancements) | E07 template variable enrichment (in progress on this branch) could provide sprint context to templates | Sprint data could be exposed as template variables in a future enhancement. No current conflict |
| **E08** (Idea Capture) | Ideas can be promoted to tasks, which can then be assigned to sprints | No direct interaction; ideas are upstream of sprint planning |
| **E10** (Context Management) | Sprint context (current sprint, assignment status) could enrich task context | Sprint assignment status should be included in `shark task get` and `shark task resume` output. Additive enhancement |
| **E17** (CLI Simplification) | Sprint commands should follow E17's simplified command patterns | E17 is completed. Sprint commands should use the `shark sprint` command group pattern established by `shark bug` and `shark change` |
| **E18** (Bug and Change-Card Management) | Bugs and change-cards can be assigned to sprints (mentioned in user journey Alt Path A for mid-sprint scope changes) | The PRD shows `shark sprint add S024 B042` -- sprint assignment should support bug and change-card keys, not just task keys. This needs to be considered in schema design (sprint_assignments should reference a polymorphic entity or have separate join tables) |

### Key Design Decision: Sprint Assignment Polymorphism

The PRD's user journey shows assigning a bug (`B042`) to a sprint. The current `sprint_assignments` schema only references `task_id`. This needs resolution:

**Option A**: Keep `sprint_assignments.task_id` as-is. Only tasks can be formally assigned to sprints. Bugs and change-cards are tracked informally.
**Option B**: Change to polymorphic references: `entity_type` + `entity_id` columns (matching the `entity_notes` pattern already in the codebase).
**Option C**: Separate join tables: `sprint_task_assignments`, `sprint_bug_assignments`, `sprint_change_assignments`.

**Recommendation**: Option B (polymorphic). The `entity_notes` table already uses this pattern (`entity_type ENUM('epic','feature','task','bug','change_card')` + `entity_id`). This is consistent with the codebase and supports the PRD's user journey without requiring schema changes if new entity types are added later.

---

## (4) Existing Capability Overlap with Defined Scope

### What Already Exists

| Capability | Current State | E19 Enhancement |
|------------|--------------|-----------------|
| **Task status tracking** | Full lifecycle with `task_history` audit trail | Sprint adds temporal grouping on top of existing status tracking. No modification to existing status flow |
| **Progress calculation** | `internal/status/` calculates weighted and completion progress for features/epics | Sprint progress is a new dimension (sprint-scoped task completion). Can reuse `status.CalculationService` patterns but needs sprint-scoped queries |
| **Agent type tracking** | Tasks have `agent_type` field. `status_metadata` tracks responsibility per status | Sprint capacity builds on agent_type. Allocation calculation groups assigned tasks by agent_type |
| **Analytics** | `shark analytics` provides project/epic-level analytics. `shark progress` shows progress breakdown | Sprint analytics (velocity, burndown, summary) are additive. They do not replace existing analytics but provide sprint-scoped views |
| **Work sessions** | `work_sessions` table and `WorkSession` model exist. Database schema is created (migration v5) | Sprint summary `--detailed` can query work_sessions for cycle-time-by-phase. Requires E13 commands to populate the data |
| **Key parsing** | `internal/keys/service.go` handles E##, E##-F##, E##-F##-###, B###, CC-### formats | Needs `S###` format added. ~20 lines following existing regex patterns |
| **Entity CRUD pattern** | Bug and change-card entities added in E18 with model/repository/service/CLI command stack | Sprint follows identical pattern. E18 provides the most recent and relevant template |
| **Configuration** | `.sharkconfig.json` with status_metadata, workflow profiles, viewer settings | Sprint defaults section is additive. Follows existing config loading patterns |
| **Template system** | Entity file templates for epics, features, tasks, bugs, change-cards in `shark-templates/` | Sprint may optionally have file templates for sprint notes/goals. Not required for MVP |
| **Dashboard** | `shark status` shows project overview with epic/feature rollups | REQ-F-017 (Could Have) adds active sprint info to dashboard. Additive change |

### What Does NOT Exist

1. **Sprint entity model** -- `internal/models/sprint.go` does not exist. Needs new model, validation, and JSON tags.
2. **Sprint repository** -- No `sprint_repository.go`. Needs CRUD, assignment, capacity, and analytics queries.
3. **Sprint service** -- No `sprint_service.go`. Needs lifecycle management, planning logic, readiness scoring, and analytics computation.
4. **Sprint CLI commands** -- No `sprint.go` in commands. Needs ~15 subcommands registered under `shark sprint`.
5. **Sprint database tables** -- `sprints`, `sprint_assignments`, `sprint_capacity` do not exist.
6. **Burndown chart rendering** -- No text-based chart rendering exists in the codebase. Needs ASCII/Unicode chart utility.

### Reuse Opportunities

- **Entity pattern**: Copy E18's bug implementation pattern (model -> repository -> service -> CLI commands).
- **Key generation**: Extend `internal/keys/service.go` and `internal/keygen/generator.go` for `S###` keys.
- **Status calculation**: Adapt `internal/status/status.go` patterns for sprint progress.
- **Configuration loading**: Extend `internal/config/` for sprint defaults.
- **Service accessors**: Add `GetSprintService()` to `internal/cli/services_global.go`.

---

## (5) Risk Assessment

| # | Risk | Probability | Impact | Mitigation | Related Requirements |
|---|------|-------------|--------|------------|---------------------|
| R1 | **E13 dependency delays sprint analytics** -- E13 is in `draft` status. If it is not implemented before or alongside E19, the `--detailed` sprint summary (cycle-time-by-phase) will have no data | High | Medium | Implement `--detailed` with graceful degradation. Core velocity and burndown work from `task_history` without E13. Document E13 as a recommended prerequisite, not a hard blocker | REQ-F-009 |
| R2 | **Polymorphic sprint assignment complexity** -- If bugs and change-cards need sprint assignment (per user journey), the schema and repository logic become more complex than a simple task_id foreign key | Medium | Medium | Adopt the `entity_type + entity_id` pattern from `entity_notes`. Plan this from the start rather than retrofitting | REQ-F-004, REQ-F-005 |
| R3 | **Burndown chart rendering quality** -- Text-based charts in terminal have limited resolution and may not be useful for sprints with many tasks or long durations | Medium | Low | Start with a simple day-by-day table format. Add ASCII chart as a "nice to have". JSON output enables external visualization tools | REQ-F-008 |
| R4 | **Scope creep into estimation workflow** -- Users may expect story point estimation integrated with sprint capacity, which is explicitly out of scope | Medium | Low | Use task priority (1-10) as the capacity unit, clearly documented. The `capacity_points` column name is generic enough to accommodate future story points without schema changes | REQ-F-014 |
| R5 | **Sprint carryover edge cases** -- Auto-carryover when closing sprints involves complex state transitions (reassigning tasks, creating next sprint, handling no-next-sprint scenario) | Medium | Medium | Implement as a transaction in the service layer. Extensive test coverage for edge cases (no next sprint, all tasks completed, empty sprint). Follow the established transaction pattern | REQ-F-006 |
| R6 | **Performance of velocity queries with many sprints** -- Aggregation queries across sprint_assignments, tasks, and sprints could be slow at scale | Low | Low | The 50-sprint / 1000-task target is small for SQLite. Proper indexes on sprint_assignments(sprint_id, task_id) and sprints(status) are sufficient. No concern at projected scale | REQ-NF-001, REQ-NF-002 |
| R7 | **Single active sprint constraint** -- The "only one active sprint" rule may be too restrictive for teams running overlapping sprints during transitions | Low | Low | The PRD acknowledges this and allows warning-only for overlapping planning sprints. The constraint applies only to `active` status, not `planning` | REQ-F-002 |

### No Feasibility Blockers Identified

All identified risks have viable mitigations. No risk is rated as a showstopper. The highest-impact risk (R1, E13 dependency) has a clear graceful degradation path.

---

## (6) Recommendations

### 1. Phased Delivery Strategy

Deliver E19 in three phases to manage complexity and decouple from E13:

**Phase 1: Sprint Core** (Must Have -- REQ-F-001 through REQ-F-006, REQ-F-010)
- Sprint model, repository, service, and CLI commands
- Sprint lifecycle (create, start, close, archive)
- Task-to-sprint assignment with polymorphic entity support
- Sprint backlog view
- Sprint close with carryover
- Database schema with all three tables

**Phase 2: Sprint Analytics** (Must Have -- REQ-F-007 through REQ-F-009)
- Velocity calculation from completed sprints
- Burndown using task_history data
- Sprint summary (basic)
- Sprint summary `--detailed` with graceful degradation for missing work_sessions

**Phase 3: Sprint Planning** (Should Have -- REQ-F-011 through REQ-F-015)
- Planning view with backlog and capacity
- Bulk task assignment
- Readiness scoring
- Capacity management with configuration defaults

### 2. Use Polymorphic Sprint Assignment from Day One

Design `sprint_assignments` with `entity_type` + `entity_id` columns instead of `task_id` only. This supports the PRD's user journey of assigning bugs to sprints and avoids a schema migration later.

### 3. Follow E18 Implementation Pattern

The E18 bug/change-card implementation is the most recent and relevant template. Follow the same model -> repository -> service -> CLI command stack for consistency and to leverage team familiarity.

### 4. Decouple from E13 with Graceful Degradation

Do not make E13 a hard prerequisite. Implement analytics that work from `task_history` (which is already populated). Add "insufficient data" messages for features that require `work_sessions`. This allows E19 to ship independently.

### 5. Start with Table-Based Burndown, Not Charts

For REQ-F-008, start with a simple day-by-day table showing ideal vs. actual remaining tasks. ASCII chart rendering can be added as an enhancement. The `--json` output is more important for AI orchestrator consumption.

### 6. Add Sprint Context to Existing Commands

After core sprint functionality is in place, enhance existing commands to show sprint context:
- `shark get E07-F01-001` should show sprint assignment if the task is in a sprint
- `shark status` dashboard should show active sprint summary (REQ-F-017)
- `shark task resume` should include sprint context

---

## References

- E19 PRD: `docs/plan/E19-sprint-management-planning-system/` (epic.md, personas.md, user-journeys.md, requirements.md, success-metrics.md, scope.md)
- E18 Bug/Change-Card Implementation: `internal/services/bug_service.go`, `internal/services/change_card_service.go`, `internal/repository/bug_repository.go`, `internal/repository/change_card_repository.go`
- E13 Epic (dependency): `docs/plan/E13-workflow-aware-task-command-system/epic.md` -- status: `draft`, 0% progress
- Key Parsing Service: `internal/keys/service.go`
- Entity Notes Polymorphic Pattern: `internal/repository/entity_note_repository.go` (entity_type + entity_id pattern)
- Work Sessions Model: `internal/models/work_session.go`
- Database Schema: `internal/db/db.go` (CurrentSchemaVersion = 6)
- Status Calculation Service: `internal/status/status.go`
- AI PM Journey Map Analysis: `dev-artifacts/2026-01-10-task-command-ux-analysis/AI-PM-Journey-Map.md`

---

*Research complete. All 6 sections addressed. No unresolved feasibility blockers. Overlap with E13 explicitly addressed with graceful degradation strategy.*
