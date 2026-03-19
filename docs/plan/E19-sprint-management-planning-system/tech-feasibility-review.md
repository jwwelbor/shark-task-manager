# E19 Sprint Management & Planning System -- Technical Feasibility Review

**Epic Key**: E19
**Review Date**: 2026-03-18
**Reviewer**: Architect Agent
**Overall Assessment**: **APPROVED**

---

## 1. Technical Feasibility by Requirement Area

### 1.1 Sprint Lifecycle Management (REQ-F-001 through REQ-F-003) -- FEASIBLE

**Assessment**: Directly implementable using proven patterns.

**Architectural Approach**:

The sprint entity follows the same model -> repository -> service -> CLI command stack established by E18 (bugs and change-cards). Specific implementation mapping:

| Component | Pattern Source | New Files |
|-----------|---------------|-----------|
| Model | `internal/models/bug.go` | `internal/models/sprint.go` |
| Repository | `internal/repository/bug_repository.go` | `internal/repository/sprint_repository.go` |
| Service | `internal/services/bug_service.go` | `internal/services/sprint_service.go` |
| CLI Commands | `internal/cli/commands/bug.go` | `internal/cli/commands/sprint.go` |
| Key Parsing | `internal/keys/service.go` (EntityTypeBug) | Add `EntityTypeSprint` with `S###` regex |
| Service Accessor | `internal/cli/services_global.go` (GetBugService) | Add `GetSprintService()` |

**Key Format**: `S###` (e.g., `S001`, `S024`). The key parsing service in `internal/keys/service.go` already handles `B###` (bugs) and `CC-###` (change-cards) as standalone entity types. Adding `S###` requires approximately 20 lines: a new `EntityTypeSprint` constant, a regex pattern `^S(\d{3})(?:-([A-Z](?:[A-Z0-9-]*[A-Z0-9])?))?$`, and a case in the `Parse()` switch.

**Sprint Status Lifecycle**: The sprint has its own lifecycle independent of the task workflow:

```
planning -> active -> closing -> completed
                   -> cancelled
completed -> archived
```

This lifecycle is simpler than the task workflow (5 states vs 19 in advanced profile). It does NOT need to integrate with the existing `workflow.Service` which is designed for epic/feature/task status flows. Instead, the `SprintService` will manage its own status transitions internally with a simple state machine, similar to how `BugService` handles bug triage states.

**Single Active Sprint Constraint**: Enforced at the service layer. `SprintService.StartSprint()` will query for any sprint in `active` status before allowing the transition from `planning` to `active`. This is a business rule (service layer), not a database constraint, because the constraint is conditional (only one `active` at a time, but multiple `planning` sprints are allowed).

**Risk**: LOW. All patterns are proven and well-understood in the codebase.

---

### 1.2 Task-to-Sprint Assignment (REQ-F-004 through REQ-F-006) -- FEASIBLE

**Assessment**: Feasible with the polymorphic assignment pattern recommended by research.

**Architectural Approach**:

The `sprint_assignments` table uses the polymorphic `entity_type + entity_id` pattern already established by `entity_notes`:

```sql
CREATE TABLE IF NOT EXISTS sprint_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sprint_id INTEGER NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('task', 'bug', 'change_card')),
    entity_id INTEGER NOT NULL,
    assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    removed_at TIMESTAMP  -- NULL means active assignment; non-NULL means historical
);

-- Enforce: each entity can be in at most one active/planning sprint
CREATE UNIQUE INDEX IF NOT EXISTS idx_sprint_active_assignment
    ON sprint_assignments(entity_type, entity_id) WHERE removed_at IS NULL;

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_sprint_assignments_sprint ON sprint_assignments(sprint_id);
CREATE INDEX IF NOT EXISTS idx_sprint_assignments_entity ON sprint_assignments(entity_type, entity_id);
```

**Key Design Decisions**:

1. **Polymorphic over task-only**: Aligns with the PRD user journey (Journey 3, Alt Path A: `shark sprint add S024 B042`). The `entity_notes` table already uses this pattern successfully. This avoids a breaking schema migration later.

2. **Soft-delete via `removed_at`**: When a task is removed from a sprint (manual removal or sprint close with carryover), `removed_at` is set rather than deleting the row. This preserves the assignment history needed for velocity calculations and sprint summary reports.

3. **Partial unique index**: SQLite supports `WHERE removed_at IS NULL` in unique indexes, which enforces the "one active sprint per entity" constraint at the database level. This is the correct approach -- it prevents double-assignment even under concurrent access.

**Carryover Logic** (REQ-F-006): This is the most complex requirement. The `SprintService.CloseSprint()` method must:

1. Begin a transaction
2. Set sprint status to `closing`
3. Query incomplete assigned entities (status not in terminal states)
4. Based on `--carryover` flag:
   - `next`: Find or create the next `planning` sprint, then soft-delete old assignments and create new ones on the next sprint
   - `backlog`: Simply soft-delete assignments (entities return to unassigned backlog)
5. Record sprint completion statistics (completed count, carryover count, velocity)
6. Set sprint status to `completed`
7. Commit transaction

This follows the established transaction pattern from the service design rules. The transaction is owned by the service layer, not the repository.

**Risk**: MEDIUM. Carryover logic has multiple edge cases (no next sprint, all tasks completed, empty sprint, polymorphic entity handling). Requires comprehensive test coverage. The complexity is manageable because it follows the established transaction pattern.

---

### 1.3 Sprint Analytics (REQ-F-007 through REQ-F-009) -- FEASIBLE

**Assessment**: Core analytics are feasible using existing `task_history` data. Detailed cycle-time analytics depend on E13 with graceful degradation.

**Architectural Approach**:

**Velocity Calculation** (REQ-F-007):

```sql
-- Velocity query: count completed entities per sprint
SELECT s.key, s.name, COUNT(*) as completed_count
FROM sprint_assignments sa
JOIN sprints s ON sa.sprint_id = s.id
JOIN tasks t ON sa.entity_type = 'task' AND sa.entity_id = t.id
WHERE s.status IN ('completed', 'archived')
  AND t.status = 'completed'
GROUP BY s.id
ORDER BY s.end_date DESC
LIMIT ?;
```

This query joins `sprint_assignments` with `tasks` to count completed tasks per sprint. For polymorphic entities (bugs, change-cards), similar joins are needed. The `SprintRepository` will expose a `GetVelocityData(ctx, limit int)` method that returns per-sprint completion counts. The `SprintService` computes the trailing average in Go code (not SQL) for clarity and testability.

**Burndown** (REQ-F-008):

The burndown reconstructs daily remaining-task counts from `task_history`:

1. Get all task IDs assigned to the sprint (from `sprint_assignments`)
2. Get sprint date range (start_date to end_date or today if active)
3. For each day in the range, count tasks NOT in a terminal status as of that day by querying `task_history` for the latest status transition before that day
4. Compute ideal burndown as a linear decrease from total to zero

This approach uses existing `task_history` data without requiring a new snapshot table. Performance is acceptable because:
- A 2-week sprint has ~14 data points
- Each data point requires one query against indexed `task_history` (indexed on task_id, created_at)
- For a sprint with 30 tasks and 14 days, this is approximately 14 queries with small result sets

For text-based rendering, a simple day-by-day table is the initial approach:

```
Day  | Ideal | Actual | Status
-----|-------|--------|-------
D01  |    30 |     30 | On track
D02  |    28 |     29 | Slightly behind
...
D14  |     0 |      3 | 3 remaining
```

ASCII chart rendering (sparklines or bar chart) can be added as an enhancement. The `--json` output provides raw data points for external visualization.

**Sprint Summary** (REQ-F-009):

Basic summary data (planned count, completed count, velocity, carryover list) comes from `sprint_assignments` and sprint metadata. No dependency on E13.

The `--detailed` flag adds "average cycle time by phase" which requires `work_sessions` data from E13. Implementation approach:

```go
func (s *SprintService) GetSprintSummary(ctx context.Context, key string, detailed bool) (*SprintSummary, error) {
    // ... basic summary always works ...

    if detailed {
        cycleTimeData, err := s.getPhraseCycleTimes(ctx, sprintID)
        if err != nil || len(cycleTimeData) == 0 {
            summary.CycleTimeNote = "Cycle time data unavailable. Enable work session tracking (E13) for phase-level analysis."
        } else {
            summary.CycleTimeByPhase = cycleTimeData
        }
    }
    return summary, nil
}
```

This graceful degradation shows an informational message when work_sessions data is absent, consistent with the velocity "insufficient data" pattern.

**Risk**: MEDIUM. The burndown reconstruction from `task_history` is the novel part. If `task_history` does not have sufficient granularity (e.g., status changes are infrequent), the burndown may show flat lines between transitions. This is acceptable for CLI output and can be improved later with snapshot-based tracking if needed. The analytics queries are aggregations over indexed columns and will perform well within the 50-sprint / 1000-task target.

---

### 1.4 Database Schema (REQ-F-010) -- FEASIBLE

**Assessment**: Additive schema changes following established migration patterns.

**Schema Design**:

```sql
-- Table: sprints
CREATE TABLE IF NOT EXISTS sprints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    goal TEXT,
    start_date TEXT NOT NULL,  -- ISO 8601 date (YYYY-MM-DD)
    end_date TEXT NOT NULL,    -- ISO 8601 date (YYYY-MM-DD)
    status TEXT NOT NULL DEFAULT 'planning'
        CHECK (status IN ('planning', 'active', 'closing', 'completed', 'cancelled', 'archived')),
    slug TEXT,
    file_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sprints_key ON sprints(key);
CREATE INDEX IF NOT EXISTS idx_sprints_status ON sprints(status);
CREATE INDEX IF NOT EXISTS idx_sprints_slug ON sprints(slug);

-- Trigger: auto-update updated_at
CREATE TRIGGER IF NOT EXISTS sprints_updated_at
AFTER UPDATE ON sprints
FOR EACH ROW
BEGIN
    UPDATE sprints SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Table: sprint_assignments (polymorphic)
CREATE TABLE IF NOT EXISTS sprint_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sprint_id INTEGER NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('task', 'bug', 'change_card')),
    entity_id INTEGER NOT NULL,
    assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    removed_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sprint_active_assignment
    ON sprint_assignments(entity_type, entity_id) WHERE removed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sprint_assignments_sprint ON sprint_assignments(sprint_id);
CREATE INDEX IF NOT EXISTS idx_sprint_assignments_entity ON sprint_assignments(entity_type, entity_id);

-- Table: sprint_capacity
CREATE TABLE IF NOT EXISTS sprint_capacity (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sprint_id INTEGER NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    agent_type TEXT NOT NULL,
    capacity_points INTEGER NOT NULL DEFAULT 0,
    allocated_points INTEGER NOT NULL DEFAULT 0,
    UNIQUE(sprint_id, agent_type)
);

CREATE INDEX IF NOT EXISTS idx_sprint_capacity_sprint ON sprint_capacity(sprint_id);
```

**Migration Integration**:

- Current `CurrentSchemaVersion` is 6. Sprint migration bumps to 7.
- Migration function `migrateSprints(db *sql.DB)` will be added to `runMigrations()` in `internal/db/db.go`.
- Migration checks `IF NOT EXISTS` for idempotency, following the established pattern.
- All three tables are created in a single migration function.

**Date Storage**: Sprint dates use `TEXT` type with ISO 8601 format (`YYYY-MM-DD`). SQLite does not have a native date type; TEXT with ISO format allows date comparison via string comparison (which works correctly for ISO 8601). This matches how `created_at` and `updated_at` timestamps are stored.

**Foreign Key Note**: `sprint_assignments.entity_id` cannot have a foreign key constraint because it references different tables based on `entity_type`. This is the same trade-off made by `entity_notes`. Referential integrity is enforced at the service layer: the `SprintService.AssignEntity()` method validates that the entity exists before creating the assignment.

**Risk**: LOW. The schema is straightforward and additive. No existing tables are modified.

---

### 1.5 Sprint Planning View (REQ-F-011 through REQ-F-013) -- FEASIBLE

**Assessment**: Composite query across existing data with new sprint context.

**Architectural Approach**:

The planning view (`shark sprint plan S024`) aggregates three data sources:

1. **Available Backlog**: Tasks not assigned to any active/planning sprint, in eligible statuses
2. **Current Sprint Allocation**: Tasks already assigned, grouped by agent type
3. **Readiness Score**: Composite metric computed in the service layer

**Backlog Query**:
```sql
SELECT t.* FROM tasks t
WHERE t.status IN (?, ?, ?)  -- Eligible statuses from workflow config
  AND t.id NOT IN (
    SELECT sa.entity_id FROM sprint_assignments sa
    WHERE sa.entity_type = 'task' AND sa.removed_at IS NULL
  )
ORDER BY t.priority DESC, t.execution_order ASC;
```

**Readiness Score Computation** (service layer):
```
score = (capacity_factor * 0.30) + (dependency_factor * 0.30) + (task_count_factor * 0.20) + (balance_factor * 0.20)
```

Where:
- `capacity_factor`: 100 if 70-100% utilized; decreases linearly below 50% or above 110%
- `dependency_factor`: 100 if no blocked external dependencies; decreases per blocked dependency
- `task_count_factor`: 100 if at least 5 tasks; decreases for empty/near-empty sprints
- `balance_factor`: 100 if all agent types have assignments; decreases with single-type concentration

This is computed entirely in Go code (no SQL), following the pattern from `internal/status/status.go` which computes health indicators and progress weights.

**Risk**: LOW. The backlog query uses standard SQL with existing indexes. The readiness score is service-layer computation with no external dependencies.

---

### 1.6 Capacity Management (REQ-F-014 through REQ-F-015) -- FEASIBLE

**Assessment**: CRUD operations on `sprint_capacity` plus configuration defaults.

**Configuration Extension**:

`.sharkconfig.json` gains a new `sprint_defaults` section:

```json
{
  "sprint_defaults": {
    "duration_days": 14,
    "carryover": "next",
    "auto_create": false,
    "capacity": {
      "backend": 21,
      "frontend": 13,
      "qa": 8,
      "ba": 5
    }
  }
}
```

The `internal/config/` package already handles JSON configuration loading. Adding a new section follows the same pattern as `status_metadata` and `workflow_profiles`. The config loader reads the section; the `SprintService` uses the defaults when creating a sprint if no explicit capacity is set.

**Allocated Points Calculation**: When a task is assigned to a sprint, its priority value (1-10) is used as the "points" cost against the agent type's capacity. This is computed by the `SprintService` during assignment, not stored redundantly. The `sprint_capacity.allocated_points` column is updated as a cached aggregate during assignment operations to avoid recomputation on every `capacity show` query.

**Risk**: LOW. Standard CRUD with configuration integration.

---

### 1.7 Non-Functional Requirements -- FEASIBLE

| NFR | Assessment |
|-----|-----------|
| **REQ-NF-001** (Response Time) | Sprint CRUD < 500ms is trivially achievable with indexed SQLite queries. Analytics < 2s is achievable given the query patterns described above. The burndown's per-day query pattern (14 queries for a 2-week sprint) completes in under 100ms total on SQLite with indexes. |
| **REQ-NF-002** (Query Efficiency) | All frequently-used queries have corresponding indexes defined in the schema. The partial unique index on `sprint_assignments` serves both constraint enforcement and query acceleration. |
| **REQ-NF-003** (Assignment Consistency) | Foreign keys on `sprint_assignments.sprint_id`, partial unique index on `(entity_type, entity_id) WHERE removed_at IS NULL`, and CHECK constraint on sprint status values provide database-level integrity. Service-layer validation adds business rules. |
| **REQ-NF-004** (Backward Compatibility) | All schema changes are additive (3 new tables, no modifications to existing tables). Existing commands, tests, and workflows are completely unaffected. |
| **REQ-NF-005** (JSON Output) | All sprint commands will support `--json` and `--field` flags using the existing `cli.OutputJSON()` and `cli.GlobalConfig` patterns. |
| **REQ-NF-006** (Security) | Sprint data uses the same database access patterns as all other entities. No new authentication mechanisms are needed. Turso cloud support is automatic since sprint queries go through the same `*sql.DB` interface. |

**Risk**: LOW across all non-functional requirements.

---

## 2. Architectural Concerns

### 2.1 Sprint Workflow Independence

**Concern**: Should sprints use the existing `workflow.Service` for status transitions?

**Decision**: NO. The sprint lifecycle (`planning -> active -> closing -> completed`) is fundamentally different from the task/feature/epic workflow. The `workflow.Service` is designed for configurable, multi-phase workflows with agent routing and status metadata. Sprints have a simple, fixed lifecycle that does not need configuration, agent routing, or workflow profiles.

**Implementation**: `SprintService` manages its own status transitions with a simple Go map of valid transitions:

```go
var sprintTransitions = map[string][]string{
    "planning":  {"active", "cancelled"},
    "active":    {"closing", "cancelled"},
    "closing":   {"completed"},
    "completed": {"archived"},
    "cancelled": {},  // terminal
    "archived":  {},  // terminal
}
```

This is simpler, more readable, and avoids coupling sprint lifecycle to the task workflow configuration.

### 2.2 Integration with Existing Entity Display

**Concern**: Should `shark get E07-F01-001` show sprint assignment information?

**Decision**: YES, as a follow-up enhancement. The `DisplayService` (or `TaskService.GetTask`) can optionally query `sprint_assignments` to include sprint context in task display. This is additive and should be implemented after core sprint functionality is stable.

**Implementation**: Add an optional `SprintInfo` field to the task display response:

```go
type TaskDisplayInfo struct {
    // ... existing fields ...
    Sprint *SprintInfo `json:"sprint,omitempty"`
}

type SprintInfo struct {
    Key       string `json:"key"`
    Name      string `json:"name"`
    Status    string `json:"status"`
    StartDate string `json:"start_date"`
    EndDate   string `json:"end_date"`
}
```

### 2.3 Sprint File System Representation

**Concern**: Should sprints have markdown files like epics, features, and tasks?

**Decision**: OPTIONAL, not required for MVP. Sprints are primarily database-managed entities (similar to bugs and change-cards). File-based representation can be added later if there is demand for sprint notes or documentation in the file system.

**Implementation**: The `sprints` table includes `file_path` and `slug` columns for forward compatibility. If files are not created, these columns remain NULL.

### 2.4 Performance of Burndown Reconstruction

**Concern**: Reconstructing burndown from `task_history` requires multiple queries.

**Decision**: The task_history-based approach is acceptable for the target scale (30 tasks, 14-day sprint = 14 queries). If performance becomes an issue at larger scale, a daily snapshot approach can be added without changing the API contract.

**Verification**: The `task_history` table has an index on `task_id` (implicit from foreign key) and `created_at`. The per-day query scans at most N rows per task (where N is the number of status transitions during the sprint), which is typically 2-4. For 30 tasks, each day query processes approximately 60-120 rows -- negligible for SQLite.

---

## 3. Dependency and Integration Risk Assessment

### 3.1 E13 (Workflow-Aware Task Command System) -- Soft Dependency

**Status**: `draft`, 0% progress
**Impact on E19**: Only the `shark sprint summary --detailed` flag (cycle-time-by-phase analytics) depends on E13's `work_sessions` data.
**Mitigation**: Graceful degradation. The `--detailed` flag will show an informational message when work_sessions data is unavailable. Core sprint functionality (lifecycle, assignment, velocity, burndown, basic summary) has zero dependency on E13.
**Risk Level**: ACCEPTABLE. No architectural concern.

### 3.2 E15 (Service Layer Architecture Refactoring) -- Enabler

**Status**: `active`, 79% progress
**Impact on E19**: E19 must follow the service layer pattern. The pattern is mature and well-documented.
**Risk Level**: NONE. E15's progress is an enabler. The `BugService` and `ChangeCardService` from E18 provide proven templates.

### 3.3 E16 (Multi-Level Workflow) -- Aligned

**Status**: `active`, 96% progress
**Impact on E19**: Sprint backlog grouping must respect the active workflow profile's status phase definitions (planning, development, review, qa, done). The `workflow.Service` provides phase information via `GetStatusMetadata()`.
**Risk Level**: NONE. The integration point is a read-only query of workflow configuration.

### 3.4 E18 (Bug and Change-Card Management) -- Template Provider

**Status**: `active`, 100% (essentially complete)
**Impact on E19**: E18's implementation pattern is the primary template for E19. The polymorphic assignment pattern for sprint-assigned bugs is enabled by E18's entity model.
**Risk Level**: NONE.

### 3.5 Cross-Epic Technical Conflicts

No technical conflicts identified. E19 introduces new database tables, a new entity type, and new CLI commands -- all additive. No existing code paths are modified by the sprint feature.

---

## 4. Technical Debt Assessment

### 4.1 New Technical Debt Created

| Item | Severity | Description | Mitigation |
|------|----------|-------------|------------|
| Polymorphic FK without DB enforcement | Low | `sprint_assignments.entity_id` has no foreign key constraint because it references multiple tables based on `entity_type`. Referential integrity is enforced at the service layer | This is the same pattern used by `entity_notes`. Service-layer enforcement is consistent and tested. Orphaned assignments are cleaned up during sprint close |
| Burndown from task_history | Low | Burndown reconstruction queries `task_history` per day. If task_history grows very large, these queries could slow down | Add snapshot-based burndown as an optimization if needed. The current approach works at the target scale (50 sprints, 1000 tasks) |
| Sprint capacity as cached aggregate | Low | `sprint_capacity.allocated_points` is a cached sum that could become stale if assignment operations fail mid-transaction | The service layer updates capacity within the same transaction as assignment changes. The cache can be recalculated from assignments if needed |

### 4.2 Existing Technical Debt Impact

E19 does not exacerbate existing technical debt:

- **Fat controllers**: E19 sprint commands will use the service layer exclusively (no legacy direct-repository-access pattern). This aligns with the E15 target architecture.
- **Test coverage**: E19 follows the testing architecture (mocked repositories for service tests, real database only for repository tests).
- **Configuration management**: Sprint defaults extend the existing `.sharkconfig.json` pattern without introducing a new configuration mechanism.

### 4.3 Technical Debt Reduced

E19 provides a new reference implementation of the full entity lifecycle (model -> repository -> service -> CLI) using current best practices. Future entity types can use E19 as a template alongside E18.

---

## 5. Phased Delivery Recommendation

Based on complexity analysis and dependency assessment, the recommended delivery order:

### Phase 1: Sprint Core (Must Have)

**Scope**: REQ-F-001 through REQ-F-006, REQ-F-010
**Complexity**: Large (L)
**Dependencies**: None

Deliverables:
- Sprint model, repository, service, CLI commands
- Sprint lifecycle (create, get, list, update, delete, start, close, archive)
- Key parsing for `S###` format
- Task/bug/change-card to sprint assignment (polymorphic)
- Sprint backlog view
- Sprint close with carryover (next or backlog)
- Database schema (3 tables + indexes + triggers)
- Service accessor in `services_global.go`

### Phase 2: Sprint Analytics (Must Have)

**Scope**: REQ-F-007 through REQ-F-009
**Complexity**: Large (L)
**Dependencies**: Phase 1

Deliverables:
- Velocity calculation and display
- Burndown reconstruction from task_history
- Sprint summary (basic and --detailed with graceful degradation)
- Text-based chart rendering utility (day-by-day table format)

### Phase 3: Sprint Planning (Should Have)

**Scope**: REQ-F-011 through REQ-F-015
**Complexity**: Medium (M)
**Dependencies**: Phase 1

Deliverables:
- Sprint planning view with backlog and capacity
- Bulk task assignment
- Readiness scoring algorithm
- Capacity management CRUD
- Default capacity configuration in .sharkconfig.json

---

## 6. Recommended Actions for Technical Refinement

1. **Adopt polymorphic sprint assignment from day one**. The `entity_type + entity_id` pattern is proven by `entity_notes` and supports the PRD's user journey for assigning bugs to sprints.

2. **Do not integrate with `workflow.Service` for sprint lifecycle**. Sprint status transitions are simple and fixed. Use a service-internal state machine.

3. **Start burndown with table format**. ASCII chart rendering can be added as a follow-up. The `--json` output is the higher-priority output mode for AI orchestrator consumption.

4. **Implement carryover in a single service-layer transaction**. This is the highest-complexity operation and requires comprehensive test coverage for edge cases.

5. **Add sprint context to `shark get` as a follow-up**. After core sprint functionality is stable, enhance task/bug/change-card display to show sprint assignment context.

6. **Plan for `CurrentSchemaVersion` bump to 7**. The developer must set `skip_migrations: false` in `.sharkconfig.json` before running the first shark command after the migration is added, then set it back to `true`.

---

## 7. Overall Assessment

| Evaluation Area | Finding | Rating |
|----------------|---------|--------|
| Technical Feasibility | All requirement areas are implementable using proven codebase patterns. No novel technology or external dependencies needed | PASS |
| Architectural Alignment | Sprint entity follows established model/repo/service/CLI pattern. No architectural shortcuts or anti-patterns introduced | PASS |
| Dependency Risk | E13 soft dependency handled via graceful degradation. E15/E16/E18 are enablers, not blockers | PASS |
| Performance | All queries are indexed. Target scale (50 sprints, 1000 tasks) is well within SQLite capabilities. Analytics under 2s | PASS |
| Data Integrity | Foreign keys, partial unique indexes, CHECK constraints, and service-layer validation provide defense in depth | PASS |
| Backward Compatibility | All changes are additive. Zero modifications to existing tables, commands, or tests | PASS |
| Technical Debt | Minimal new debt (polymorphic FK, cached aggregates). No exacerbation of existing debt. New reference implementation | PASS |

**OVERALL: APPROVED**

The epic is technically feasible with no blocking issues. The codebase has mature patterns for every aspect of the implementation. The recommended phased delivery (Core -> Analytics -> Planning) manages complexity while delivering incremental value. The E13 dependency is fully mitigated by graceful degradation.

The epic is ready to advance to the next workflow phase.

---

*Technical Feasibility Review Complete -- 2026-03-18*
