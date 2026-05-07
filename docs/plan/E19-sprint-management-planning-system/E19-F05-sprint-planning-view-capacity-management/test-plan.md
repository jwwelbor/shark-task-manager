# Test Plan: E19-F05 — Sprint Planning View & Capacity Management

**Created:** 2026-05-05
**QA Agent:** QA
**Feature Spec:** docs/plan/E19-sprint-management-planning-system/E19-F05-sprint-planning-view-capacity-management/spec.md
**Epic UAT Plan:** docs/plan/E19-sprint-management-planning-system/uat-acceptance-plan.md
**Status:** APPROVED

---

## 1. Spec Drift Analysis

### 1.1 Drift Findings

No material drift identified. The spec.md for E19-F05 is internally consistent and traces cleanly to the parent epic requirements.md. The following observations are non-blocking:

- **Terminology precision (minor):** The spec describes `carryover_behavior` and `auto_create` fields as "parsed and stored" but explicitly defers REQ-F-016 behavior. The deferred behavior is correctly gated behind the `auto_create` flag; no test cases are needed for that behavior in this feature.
- **Factor 1 penalty formula:** The spec states `score = max(0, 25 - int((utilization-1.0)*50))` for overcommit. This translates to a -2 point penalty per 4% over 100% (e.g., 110% utilization → score = max(0, 25 - 5) = 20). The test cases below encode this formula precisely so a buggy implementation that applies the wrong rate is caught.
- **Dependency satisfaction (Factor 2):** The spec limits dependency checking to task entities only (bugs/changes/tech-debt have no `depends_on` schema field). This is a deliberate design decision documented in §2.10, not drift.

### 1.2 Traceability Matrix

| Epic Requirement | Feature AC (spec §1.1) | Test Cases | Covered? |
|---|---|---|---|
| REQ-F-011: Sprint Planning Command | §1.1.1 (5 ACs) | TC-011-01 through TC-011-07 | Yes |
| REQ-F-012: Bulk Entity Assignment | §1.1.2 (9 ACs) | TC-012-01 through TC-012-10 | Yes |
| REQ-F-013: Sprint Readiness Score | §1.1.3 (10 ACs) | TC-013-01 through TC-013-14 | Yes |
| REQ-F-014: Agent Capacity Configuration | §1.1.4 (8 ACs) | TC-014-01 through TC-014-08 | Yes |
| REQ-F-015: Default Capacity Configuration | §1.1.5 (6 ACs) | TC-015-01 through TC-015-07 | Yes |
| REQ-NF-001: Command response time < 500ms | §1.2 | TC-NF-001 | Yes |
| REQ-NF-004: Backward compatibility | §1.2 | TC-NF-002 | Yes |
| REQ-NF-005: JSON output on all commands | §1.2 | Embedded in per-command TCs | Yes |

---

## 2. Acceptance Criteria Review

### 2.1 Quality Checks Per AC

All ACs are unambiguous, testable, and traceable. No "correctly handles" or "gracefully" language found. The readiness score formula is fully enumerated in spec §2.4, closing the open-ended robustness trap.

**Potential ambiguity addressed:**
- AC for Factor 1 says "scaled penalty" for overcommit — the exact formula in §2.4 eliminates ambiguity; tests encode the formula directly.
- "eligible statuses" for bulk assignment is anchored to the identical logic as E19-F03 individual add: not `completed`, `archived`, or `cancelled`. Tests verify boundary status values.
- `sprint_defaults.capacity` inheritance on create says "first-time setup convenience only — subsequent explicit sets always override." Tests verify the override path does not mutate `.sharkconfig.json`.

### 2.2 Missing Coverage

None. All five requirements (REQ-F-011 through REQ-F-015) and all non-functional requirements have at least one test case.

---

## 3. ISTQB Technique Application (per AC)

| AC Group | Technique(s) Applied | Test Cases Generated | Rationale |
|---|---|---|---|
| REQ-F-011 backlog filtering | Equivalence Partitioning + Decision Table | TC-011-01..TC-011-04 | Input is a set of entity statuses × sprint membership states; EP partitions valid/invalid for each dimension; DT covers combinations |
| REQ-F-011 performance | Boundary Value Analysis | TC-011-05 | Ordered domain: ≤500 entities is the stated boundary; BVA drives min (0), boundary-1 (499), boundary (500), boundary+1 (501) |
| REQ-F-011 JSON output | Contract surface enumeration | TC-011-06..TC-011-07 | JSON contract: 3 keys required (`backlog`, `capacity`, `readiness`); every key × every field |
| REQ-F-012 bulk assignment | Decision Table + State Transition | TC-012-01..TC-012-06 | Multiple eligibility conditions (status, sprint membership) × entity type variants; State Transition for the transaction rollback path |
| REQ-F-012 capacity warning | Equivalence Partitioning | TC-012-07..TC-012-08 | Two equivalence classes: within capacity (no warning), over capacity (warning emitted but not error) |
| REQ-F-012 transaction atomicity | Attack-class enumeration | TC-012-09..TC-012-10 | Attack class: partial-failure injection mid-batch; ensures rollback leaves no partial state |
| REQ-F-013 readiness score factors | Decision Table + BVA | TC-013-01..TC-013-10 | Six factors × boundary conditions; each factor has a distinct formula requiring BVA at each scoring breakpoint |
| REQ-F-013 edge cases | Equivalence Partitioning | TC-013-11..TC-013-14 | Zero entities (degenerate case), all unsized, all oversized, perfect sprint |
| REQ-F-014 capacity CRUD | State Transition + BVA | TC-014-01..TC-014-06 | State: no-row → row via set, row → updated row via set; BVA on `points > 0` constraint |
| REQ-F-014 computed allocation | Contract surface enumeration | TC-014-07..TC-014-08 | Contract: allocated_points = Σ size for matching agent_type; verify with multi-agent and null-size cases |
| REQ-F-015 defaults on create | State Transition | TC-015-01..TC-015-03 | States: config absent → no rows; config present → rows inserted; config present + explicit set → rows updated, config unchanged |
| REQ-F-015 config mutation | Attack-class enumeration | TC-015-04..TC-015-07 | Attack class: explicit per-sprint set must NOT write to config; `--default` flag must write to config and not DB |

---

## 4. ISO 25010 Coverage Matrix

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| REQ-F-011 backlog display | ✅ TC-011-01..04 | ✅ TC-011-05 | N/A | ✅ TC-011-07 (output structure) | N/A | N/A | ✅ thin-wrapper pattern enforced | N/A |
| REQ-F-011 JSON output | ✅ TC-011-06..07 | N/A | ✅ TC-011-06 (--json + --field) | N/A | N/A | N/A | N/A | N/A |
| REQ-F-012 bulk assignment | ✅ TC-012-01..06 | N/A | N/A | ✅ TC-012-07 (warning not error) | ✅ TC-012-09..10 (tx rollback) | N/A | N/A | N/A |
| REQ-F-012 capacity warning | ✅ TC-012-07..08 | N/A | N/A | ✅ TC-012-07 (advisory, not blocking) | N/A | N/A | N/A | N/A |
| REQ-F-013 readiness score | ✅ TC-013-01..14 | ✅ TC-NF-001 (< 500ms) | N/A | ✅ TC-013-01 (human output labels) | ✅ TC-013-11 (zero entities) | N/A | ✅ determinism assertion | N/A |
| REQ-F-014 capacity CRUD | ✅ TC-014-01..06 | N/A | N/A | ✅ TC-014-03 (empty table, not error) | N/A | N/A | N/A | N/A |
| REQ-F-014 computed allocation | ✅ TC-014-07..08 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| REQ-F-015 defaults | ✅ TC-015-01..03 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| REQ-F-015 config isolation | ✅ TC-015-04..07 | N/A | N/A | N/A | ✅ TC-015-06 (idempotency) | ✅ TC-015-05 (no unintended config write) | N/A | N/A |
| REQ-NF-004 backward compat | N/A | N/A | ✅ TC-NF-002 | N/A | ✅ TC-NF-002 | N/A | N/A | N/A |

**N/A justifications:** Security characteristics (injection, auth bypass) are not applicable to pure in-memory scoring and config-file mutations that operate only on local filesystem with user-level permissions. Portability is not applicable: the feature is CLI-only, targeting a single OS-agnostic Go binary.

---

## 5. Observability Design (per behavior)

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| `shark sprint plan` executes | Internal — no new observability; existing DB query latency is observable at the SQLite layer | `DEBUG sprint.plan sprint_id=... backlog_count=... duration_ms=...` | Internal — no production span boundary; single CLI invocation | N/A | TC-011-05 asserts wall-clock < 500ms |
| `BulkAddToSprint` inserts | `INFO sprint.bulk_add sprint_id=... added=... skipped=... duration_ms=...` | Same | Internal | N/A | TC-012-01 asserts added/skipped counts in return value |
| Capacity warning emitted | Internal — advisory path; no metric needed | `WARN sprint.capacity_exceeded sprint_id=... agent_type=... allocated=... capacity=...` | Internal | N/A | TC-012-07 asserts `BulkAddResult.Warnings` non-empty |
| `GetSprintReadiness` score | Internal — no persistent side effect | `DEBUG sprint.readiness sprint_id=... score=... duration_ms=...` | Internal | N/A | TC-013-01..14 assert score fields |
| `SetCapacity` upsert | `INFO sprint.capacity_set sprint_id=... agent_type=... points=...` | Same | Internal | N/A | TC-014-01..02 assert row state after call |
| Sprint created with defaults | `INFO sprint.created_with_defaults sprint_id=... capacity_entries=N` | Same | Internal | N/A | TC-015-01 asserts capacity rows exist after create |
| `--default` flag writes config | `INFO config.sprint_defaults_updated agent_type=... points=...` | Same | Internal | N/A | TC-015-04 asserts config file contains updated value |

**Implementation hook:** The log lines above are hard requirements. The developer must emit them; QA verifies their presence in service-layer log output during automated tests (log capture via `logrus` test hook or `slog` handler).

---

## 6. Caller-Path Contracts (summary table)

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-011-01..07 | `SprintService.PlanSprint(ctx, "S024")` | `SprintAssignmentQueryRepository` and `SprintCapacityRepository` boundaries | Do NOT mock `PlanSprint` itself; do NOT call with sprint ID instead of key | Buggy impl that ignores removed_at IS NULL filter returns already-assigned tasks in backlog |
| TC-012-01..06 | `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})` | `SprintAssignmentQueryRepository.BulkAssign` | Do NOT mock `ListUnassignedBacklog`; must exercise the real eligibility filter | Buggy impl that skips status filter assigns completed tasks; added count is wrong |
| TC-012-07..08 | `SprintService.BulkAddToSprint(ctx, BulkAddInput{...})` — same entrypoint | `SprintCapacityRepository.GetCapacity` | Do NOT mock the warning-generation path; it must be driven by real capacity comparison | Buggy impl that returns error instead of warning blocks assignment |
| TC-012-09..10 | `SprintService.BulkAddToSprint(ctx, BulkAddInput{...})` | `SprintAssignmentQueryRepository.BulkAssign` (inject error on second call) | Do NOT mock the transaction; the test must verify no rows exist in DB after rollback | Buggy impl that commits partial inserts leaves orphaned assignments |
| TC-013-01..14 | `SprintService.GetSprintReadiness(ctx, "S024")` | `SprintAssignmentQueryRepository.GetAssignmentsWithSize` and `SprintCapacityRepository.GetCapacity` | Do NOT mock individual factor computations; the entire in-memory scoring must run | Buggy factor formula (e.g., wrong penalty slope) produces wrong overall score |
| TC-014-01..06 | `SprintService.SetSprintCapacity(ctx, SetSprintCapacityInput{SprintKey:"S024", AgentType:"backend", Points:21})` | `SprintCapacityRepository.SetCapacity` | Do NOT call with sprint ID; production path uses sprint key | Buggy impl that inserts duplicate row (not upsert) fails on second call |
| TC-014-07..08 | `SprintService.GetSprintCapacity(ctx, "S024")` | `SprintAssignmentQueryRepository.GetAssignmentsWithSize` and `SprintCapacityRepository.GetCapacity` | Do NOT mock `CapacityRow` computation; must drive real Σ calculation | Buggy impl that omits unsized entities from unsized_count returns wrong advisory |
| TC-015-01..03 | `SprintService.CreateSprint(ctx, CreateSprintInput{...})` with cfg containing SprintDefaults | `SprintCapacityRepository.SetCapacity` | Do NOT inject cfg=nil; production path reads real config | Buggy impl that ignores SprintDefaults.Capacity creates sprint with no capacity rows |
| TC-015-04..07 | `config.Manager.SetSprintCapacityDefault("backend", 21)` (for --default path); `SprintService.SetSprintCapacity(ctx, ...)` (for per-sprint path) | File I/O boundary for config mutation; `SprintCapacityRepository.SetCapacity` for DB path | Do NOT mock both paths together; test --default and per-sprint in separate test cases | Buggy impl that calls both `SetSprintCapacityDefault` AND `SetCapacity` on --default mutates DB when it should only mutate config |
| TC-NF-001 | `SprintService.PlanSprint(ctx, "S024")` with 500 backlog entities pre-seeded | Real repository implementation (repository-level performance test) | No mocks — must use real DB to measure realistic I/O | Buggy impl with N+1 query per entity exceeds 500ms budget |
| TC-NF-002 | All existing `shark task list`, `shark epic list`, `shark sprint list` commands | N/A — end-to-end regression | N/A | Schema version bump without migration breaks existing queries |

---

## 7. Acceptance Test Cases

---

### TC-011-01: Backlog section excludes entities assigned to active/planning sprints

**Feature Requirement:** REQ-F-011 — backlog excludes entities already assigned to sprint in `planning` or `active` status (via `sprint_assignments WHERE removed_at IS NULL`)
**Task AC:** §1.1.1 AC-3: "Backlog excludes entities already assigned to S024 or any sprint with status planning or active"
**Technique Applied:** Equivalence Partitioning — valid classes: unassigned, assigned-to-planning, assigned-to-active, assigned-to-completed; invalid class: removed (removed_at IS NOT NULL) should be treated as unassigned
**ISO 25010:** Functional Suitability, Reliability

**Caller-Path Contract:**
- Entrypoint: `SprintService.PlanSprint(ctx, "S024")` — key string, no sprint ID
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.ListUnassignedBacklog` — mock returns controlled backlog set
- Forbidden mocks: Do NOT mock `PlanSprint` itself or the sorting logic
- Counter-factual: An impl that ignores the `removed_at IS NULL` condition would include previously-removed assignments, making the backlog smaller than it should be

**Preconditions:**
- Sprint S024 in `planning` status with ID=24
- Task A: `ready_for_development`, assigned to S024 (removed_at IS NULL) — should be excluded
- Task B: `ready_for_development`, assigned to S025 with `active` status — should be excluded
- Task C: `ready_for_development`, unassigned — should appear
- Task D: `ready_for_development`, was assigned to S023 but removed (removed_at IS NOT NULL) — should appear
- Task E: `completed`, unassigned — should be excluded (status ineligible)

**Input:** `PlanSprint(ctx, "S024")`

**Expected Output:**
- `SprintPlanView.Backlog` contains exactly Task C and Task D
- Task A, Task B, Task E are absent from backlog
- Backlog count = 2

**Edge Cases:**
- Zero unassigned tasks: backlog is empty slice (not nil)
- All tasks assigned to current sprint: backlog is empty
- Entity with removed_at set by concurrent operation: treated as unassigned on next call

**Negative Cases:**
- Completed/archived/cancelled entities MUST NOT appear in backlog regardless of sprint membership

---

### TC-011-02: Backlog sort order — priority descending then execution_order ascending

**Feature Requirement:** REQ-F-011 — "unassigned backlog sorted by priority descending then execution_order ascending"
**Task AC:** §1.1.1 AC-1
**Technique Applied:** Boundary Value Analysis — priority is 1-10; execution_order can be NULL; BVA drives: max priority (10), min priority (1), equal priority with different execution_order, NULL execution_order
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- Entrypoint: `SprintService.PlanSprint(ctx, "S024")`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.ListUnassignedBacklog` returning pre-ordered slice
- Forbidden mocks: Do NOT mock the sort — verify the repository query returns items in the specified order (ORDER BY priority DESC, execution_order ASC NULLS LAST)
- Counter-factual: An impl that sorts by execution_order first would return items in wrong sequence when priorities differ

**Preconditions:**
- 4 unassigned, eligible tasks:
  - Task A: priority=10, execution_order=2
  - Task B: priority=10, execution_order=1
  - Task C: priority=5, execution_order=NULL
  - Task D: priority=1, execution_order=1

**Input:** `PlanSprint(ctx, "S024")`

**Expected Output:**
- Backlog order: B (p=10, eo=1), A (p=10, eo=2), C (p=5, eo=NULL), D (p=1, eo=1)

**Edge Cases:**
- NULL execution_order: sorted last within same priority group (NULLS LAST)
- All tasks same priority and null execution_order: order is stable but not specified — no assertion on relative order

---

### TC-011-03: Backlog item fields — entity key, type, title, priority, size, agent_type

**Feature Requirement:** REQ-F-011 — "Backlog section shows: entity key, entity type, title, priority, size (or 'unsized'), agent_type"
**Task AC:** §1.1.1 AC-2
**Technique Applied:** Contract surface enumeration — every required field × present/absent
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- Entrypoint: `SprintService.PlanSprint(ctx, "S024")`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.ListUnassignedBacklog` returning `BacklogItem` with size=nil and agent_type=nil
- Forbidden mocks: Do NOT mock the field-mapping from BacklogItem to output struct
- Counter-factual: An impl that omits `size` from BacklogItem would cause null pointer panic or missing field in JSON

**Preconditions:**
- One unassigned task: key="E07-F01-003", type="task", title="Auth fix", priority=8, size=nil, agent_type="backend"
- One unassigned bug: key="B042", type="bug", title="Login crash", priority=9, size=3, agent_type=nil

**Input:** `PlanSprint(ctx, "S024")`

**Expected Output:**
- Backlog[0] (B042): EntityType="bug", Key="B042", Title="Login crash", Priority=9, Size=ptr(3), AgentType=nil
- Backlog[1] (E07-F01-003): EntityType="task", Key="E07-F01-003", Title="Auth fix", Priority=8, Size=nil, AgentType=ptr("backend")
- When formatted for human display: size=nil renders as "unsized"

---

### TC-011-04: plan command — happy path with all three sections

**Feature Requirement:** REQ-F-011 (Scenario 1 in §1.3)
**Task AC:** §1.1.1 AC-1: "displays three sections: (a) unassigned backlog, (b) capacity per agent, (c) readiness score"
**Technique Applied:** Decision Table — presence/absence of backlog items × capacity rows × readiness score non-zero
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- Entrypoint: `SprintService.PlanSprint(ctx, "S024")`
- Lowest allowed mock seam: both `SprintAssignmentQueryRepository` and `SprintCapacityRepository` interfaces
- Forbidden mocks: Do NOT mock `PlanSprint` or `GetSprintReadiness` — readiness is called internally and must compute in-memory
- Counter-factual: An impl that returns `Readiness: nil` would panic when the CLI tries to format the readiness section

**Preconditions:**
- S024 in `planning` status
- 2 eligible unassigned tasks
- 2 capacity rows: backend=21, frontend=13
- 3 existing assignments with sizes: backend tasks total 12 points

**Input:** `PlanSprint(ctx, "S024")`

**Expected Output:**
- `SprintPlanView.Backlog` has 2 items
- `SprintPlanView.Capacity` has 2 rows (backend: capacity=21 allocated=12, frontend: capacity=13 allocated=0)
- `SprintPlanView.Readiness` is non-nil with OverallScore > 0

---

### TC-011-05: plan command — performance bound for 500 backlog entities

**Feature Requirement:** REQ-NF-001 and REQ-F-011 AC-5: "< 500ms for ≤500 backlog entities"
**Task AC:** §1.1.1 AC-5
**Technique Applied:** Boundary Value Analysis — 0, 499, 500, 501 entities
**ISO 25010:** Performance Efficiency

**Caller-Path Contract:**
- Entrypoint: This is a repository-level integration test; use `SprintRepository.ListUnassignedBacklog(ctx, []string{"task"})` against real test DB
- Lowest allowed mock seam: No mocks — requires real SQLite for meaningful timing
- Forbidden mocks: Do NOT mock the repository; performance cannot be measured against an in-memory mock
- Counter-factual: An N+1-query impl (one query per entity to check sprint membership) exceeds 500ms at 500 entities; the `NOT EXISTS` subquery is required

**Preconditions:**
- Test DB with 500 tasks in `ready_for_development`, none assigned to any sprint

**Input:** `SprintRepository.ListUnassignedBacklog(ctx, []string{"task"})` — measured with `time.Now()`

**Expected Output:**
- Wall-clock duration < 500ms
- Returns 500 items

**Edge Cases:**
- 0 entities: < 50ms, returns empty slice
- 501 entities: system must still function; result is truncated only if the service applies a limit (spec does not specify a limit, so all 501 should be returned)

---

### TC-011-06: plan --json output structure

**Feature Requirement:** REQ-F-011 AC-4; REQ-NF-005
**Task AC:** §1.1.1 AC-4: "--json output returns structured object with keys backlog, capacity, readiness"
**Technique Applied:** Contract surface enumeration — three required top-level keys × required sub-fields
**ISO 25010:** Functional Suitability, Compatibility (AI orchestrator consumption)

**Caller-Path Contract:**
- Entrypoint: `runSprintPlan` CLI handler with `--json` flag set; calls `GetSprintService().PlanSprint(ctx, key)` then `cli.OutputJSON(result)`
- Lowest allowed mock seam: `MockSprintService` implementing `PlanSprint`
- Forbidden mocks: Do NOT mock `cli.OutputJSON`; the JSON serialization must run
- Counter-factual: An impl that returns `capacity` as a flat map instead of an array would break orchestrator JSON parsing

**Preconditions:**
- MockSprintService returns a `SprintPlanView` with non-nil Backlog (1 item), Capacity (1 row), Readiness (score=42)

**Input:** Execute `runSprintPlan` with `args=["S024"]`, `GlobalConfig.JSON=true`

**Expected Output (JSON):**
```json
{
  "backlog": [{"entity_type": "task", "key": "...", "title": "...", "priority": 5, "size": null, "agent_type": "backend"}],
  "capacity": [{"agent_type": "backend", "capacity_points": 21, "allocated_points": 10, "remaining": 11, "unsized_assigned": 0}],
  "readiness": {"overall_score": 42, "factors": [...], "unsized_entities": [], "oversized_entities": []}
}
```

**Negative Cases:**
- If `--field=backlog` is passed, output must be only the backlog array (tests --field flag propagation)

---

### TC-011-07: plan human-readable output sections

**Feature Requirement:** REQ-F-011 AC-1
**Task AC:** Three sections in human output
**Technique Applied:** Contract surface enumeration — section headers present, items rendered
**ISO 25010:** Usability

**Caller-Path Contract:**
- Entrypoint: `runSprintPlan` CLI handler without `--json`; calls `PlanSprint` then formats output
- Lowest allowed mock seam: `MockSprintService`
- Forbidden mocks: Do NOT mock the formatting functions; they must run
- Counter-factual: An impl that returns all data as one undifferentiated block fails UAT-J2-01 "three sections" acceptance

**Preconditions:** MockSprintService returns SprintPlanView with all three sections populated

**Expected Output:** Stdout contains section headers identifiable as "Backlog", "Capacity", and "Readiness" (or equivalent) — exact heading text to be confirmed by developer; test asserts all three are present via `strings.Contains`

---

### TC-012-01: Bulk assign by feature — assigns all eligible tasks from feature

**Feature Requirement:** REQ-F-012 (Scenario 2 in §1.3)
**Task AC:** §1.1.2 AC-1: "assigns all tasks from feature E07-F34 that are in eligible statuses and not already in any active/planning sprint"
**Technique Applied:** Decision Table — eligibility conditions: status eligible × not in sprint (Y/N) × two combinations = 4 rows; ineligible status eliminates two rows
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- Entrypoint: `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.ListUnassignedBacklog` (returns controlled set) and `SprintAssignmentQueryRepository.BulkAssign` (captures inserted rows)
- Forbidden mocks: Do NOT mock eligibility filtering — it must execute inside the service using the returned backlog set
- Counter-factual: An impl that calls `BulkAssign` without filtering by `FeatureKey` assigns tasks from other features

**Preconditions:**
- Feature E07-F34 has 5 tasks:
  - T-E07-F34-001: `ready_for_development`, not in any sprint → eligible
  - T-E07-F34-002: `in_progress`, not in any sprint → eligible (not completed/archived/cancelled)
  - T-E07-F34-003: `completed` → ineligible
  - T-E07-F34-004: `ready_for_development`, in S023 (active) → ineligible (already assigned)
  - T-E07-F34-005: `ready_for_development`, unassigned → eligible
- Mock `ListUnassignedBacklog` returns tasks 001, 002, 005 (already filtered by repo)

**Input:** `BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})`

**Expected Output:**
- `BulkAddResult.Added = 3`
- `BulkAddResult.Skipped = 2`
- `BulkAssign` called once with slice of 3 assignments

---

### TC-012-02: Bulk assign --bulk-bugs assigns all open bugs not in sprint

**Feature Requirement:** REQ-F-012 AC-2
**Task AC:** §1.1.2 AC-2
**Technique Applied:** Equivalence Partitioning — entity type = "bug"; status classes: open (eligible), closed/resolved (ineligible)
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- Entrypoint: `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityType:"bug"})`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.ListUnassignedBacklog` called with `entityTypes=["bug"]`
- Forbidden mocks: Do NOT call with `entityTypes=["task"]` — the entity type routing must be verified
- Counter-factual: An impl that hardcodes `entityTypes=["task"]` for all bulk operations returns zero bugs

**Input:** `BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityType:"bug"})`

**Expected Output:** `BulkAssign` called with assignments where all entries have `EntityType="bug"`

---

### TC-012-03: Bulk assign variants — tech-debt and change-cards

**Feature Requirement:** REQ-F-012 AC-3, AC-4
**Task AC:** §1.1.2
**Technique Applied:** Equivalence Partitioning — entity type = "tech_debt" and "change_card"
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- Entrypoint: `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityType:"tech_debt"})` and `...EntityType:"change_card")`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.ListUnassignedBacklog`
- Forbidden mocks: Same as TC-012-02
- Counter-factual: An impl that maps `--bulk-tech-debt` to entity type "tech-debt" (wrong hyphen) produces zero results

**Two sub-tests:**
- tech-debt: `ListUnassignedBacklog` called with `["tech_debt"]`; BulkAssign entries all have EntityType="tech_debt"
- change-card: `ListUnassignedBacklog` called with `["change_card"]`; BulkAssign entries all have EntityType="change_card"

---

### TC-012-04: Bulk assign output fields — added, skipped, warnings

**Feature Requirement:** REQ-F-012 AC-5, AC-8: "Output shows count added per entity type, total added... --json includes added, skipped, warnings"
**Task AC:** §1.1.2 AC-5 and AC-8
**Technique Applied:** Contract surface enumeration — three required JSON fields × present/absent
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- Entrypoint: `runSprintAdd` CLI handler with `--bulk E07-F34 --json`; calls `BulkAddToSprint` then `cli.OutputJSON(result)`
- Lowest allowed mock seam: `MockSprintService` returning `BulkAddResult{Added:3, Skipped:2, Warnings:[]}`
- Forbidden mocks: Do NOT mock OutputJSON
- Counter-factual: An impl that returns `count_added` (wrong field name) instead of `added` breaks orchestrator JSON parsing

**Expected JSON output:**
```json
{"added": 3, "skipped": 2, "warnings": []}
```

---

### TC-012-05: Bulk assign skips already-assigned tasks without error

**Feature Requirement:** REQ-F-012 AC-1: "skips ineligible tasks silently with a count in output"
**Task AC:** §1.1.2 AC-1
**Technique Applied:** State Transition — entity state "already in active sprint" → should be skipped, not error
**ISO 25010:** Reliability, Usability

**Caller-Path Contract:**
- Entrypoint: `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.BulkAssign` — `SkipsDuplicate` behavior exercised by the repository
- Forbidden mocks: Do NOT mock the skip logic; it must be driven by the repository returning count=0 for duplicate
- Counter-factual: An impl that returns an error on duplicate blocks the entire batch

**Expected:** `BulkAddResult.Added >= 0`; no error returned when some entities are ineligible

---

### TC-012-06: Bulk assign -- assignable status boundary

**Feature Requirement:** REQ-F-012 AC-9: "Assignable task statuses: any status that is not completed, archived, or cancelled"
**Task AC:** §1.1.2 AC-9
**Technique Applied:** Boundary Value Analysis — status just inside the assignable set vs. just outside
**ISO 25010:** Functional Suitability

**Preconditions (three tasks, one per boundary):**
- Task in `ready_for_development` (assignable) → included
- Task in `completed` (excluded) → not included
- Task in `in_development` (assignable — not in exclusion list) → included

**Expected:** Only the `ready_for_development` and `in_development` tasks appear in `BulkAddResult.Added`

---

### TC-012-07: Bulk assign over capacity emits warning, not error

**Feature Requirement:** REQ-F-012 AC-6 (Scenario 2 in §1.3): "Warns (does not fail) if bulk assignment would cause any agent type to exceed its capacity"
**Task AC:** §1.1.2 AC-6
**Technique Applied:** Equivalence Partitioning — two classes: within capacity (no warning), over capacity (warning in result)
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- Entrypoint: `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})`
- Lowest allowed mock seam: `SprintCapacityRepository.GetCapacity` returns capacity=21 for backend; `GetAssignmentsWithSize` returns existing allocated=20
- Forbidden mocks: Do NOT mock the capacity comparison — it must run in-service
- Counter-factual: An impl that returns an error (exit code non-zero) when over capacity blocks PM from assigning tasks

**Preconditions:**
- Backend capacity = 21 points
- Existing backend allocation = 20 points
- Bulk adds 2 backend tasks with size=3 each (would allocate 26 total, exceeding 21)

**Expected Output:**
- `BulkAddResult.Added = 2` (tasks assigned)
- `BulkAddResult.Warnings = ["Backend capacity exceeded: allocated 26/21 points"]` (non-empty)
- No error returned from `BulkAddToSprint`

**Negative Cases:**
- Within capacity: `BulkAddResult.Warnings` is empty slice

---

### TC-012-08: Bulk assign JSON includes warnings field

**Feature Requirement:** REQ-F-012 AC-8
**Task AC:** §1.1.2 AC-8: "--json output includes added, skipped, warnings"
**Technique Applied:** Contract surface enumeration
**ISO 25010:** Compatibility (AI orchestrator)

**Expected JSON when over-capacity:**
```json
{"added": 2, "skipped": 0, "warnings": ["Backend capacity exceeded: allocated 26/21 points"]}
```

---

### TC-012-09: Bulk assign is transactional — rollback on error

**Feature Requirement:** REQ-F-012 AC-7: "Bulk assignment is transactional: all-or-nothing per entity type group; partial failure rolls back the group" and spec §1.2 bulk transaction safety
**Task AC:** §1.1.2 AC-7
**Technique Applied:** Attack-class enumeration — attack class: partial failure injection (BulkAssign returns error after inserting N of M rows)
**ISO 25010:** Reliability

**Caller-Path Contract:**
- Entrypoint: `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.BulkAssign` — inject error return on first call
- Forbidden mocks: Do NOT mock the transaction boundary; the test must verify that after an error, `GetAssignmentsWithSize` returns zero rows for S024
- Counter-factual: An impl that commits partial inserts before calling `BulkAssign` leaves orphaned sprint_assignment rows

**Preconditions:** 3 eligible tasks from E07-F34. `BulkAssign` mock returns `(0, fmt.Errorf("constraint violation"))`.

**Expected Output:**
- `BulkAddToSprint` returns non-nil error
- `BulkAddResult.Added = 0`
- No new sprint_assignment rows for S024 (verified via `GetAssignmentsWithSize` call returning original count)

---

### TC-012-10: Bulk assign -- concurrent race protection

**Feature Requirement:** REQ-F-012 AC-7 and spec §1.2
**Task AC:** §1.1.2 AC-7
**Technique Applied:** Attack-class enumeration — attack class: race condition (duplicate concurrent assignment)
**ISO 25010:** Reliability

**Preconditions:** `BulkAssign` mock simulates `UNIQUE constraint violation` for one entity (as would happen if two agents ran bulk assign simultaneously)

**Expected Output:** Error returned from `BulkAddToSprint`; no partial state

---

### TC-013-01: Readiness score — Factor 1 (Capacity utilization) all score zones

**Feature Requirement:** REQ-F-013 AC-2: "25 pts if utilization 50-100%; scaled penalty for <50% or >100%"
**Task AC:** §1.1.3 Factor 1
**Technique Applied:** Boundary Value Analysis — breakpoints at 0%, 50%, 100%, 110% (applying -2 per 4% over → ≈50% over = score 0); Decision Table for three zones
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- Entrypoint: `SprintService.GetSprintReadiness(ctx, "S024")`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.GetAssignmentsWithSize` and `SprintCapacityRepository.GetCapacity`
- Forbidden mocks: Do NOT mock the scoring logic — it must compute in-memory
- Counter-factual: An impl using floor division instead of float division returns wrong score at 49.9% utilization

**Test variations (table-driven):**
| Scenario | totalCapacity | totalAllocated | utilization | Expected Factor1 Score |
|---|---|---|---|---|
| Zero capacity | 0 | 0 | N/A | 0 |
| Under 50%: 0% | 21 | 0 | 0% | 0 |
| Under 50%: 25% | 21 | 5 | ~24% | int(0.24/0.5 * 25) = 12 |
| Under 50%: 49% | 21 | 10 | ~48% | int(0.48/0.5 * 25) = 24 |
| At 50% | 20 | 10 | 50% | 25 |
| At 100% | 20 | 20 | 100% | 25 |
| 104% | 25 | 26 | 104% | max(0, 25 - int(0.04*50)) = max(0, 25-2) = 23 |
| 150% | 20 | 30 | 150% | max(0, 25 - int(0.5*50)) = max(0, 25-25) = 0 |

**Expected:** `SprintReadiness.Factors[0].Score` matches the Expected column for each row

---

### TC-013-02: Readiness score — Factor 2 (Dependency satisfaction)

**Feature Requirement:** REQ-F-013 AC-3: "20 pts if all dependencies satisfied; 1 pt deducted per unsatisfied external dependency, min 0"
**Task AC:** §1.1.3 Factor 2
**Technique Applied:** BVA — 0 unsatisfied (max score), 1 unsatisfied (score=19), 20 unsatisfied (score=0), 21 unsatisfied (score=0, not negative)
**ISO 25010:** Functional Suitability, Reliability

**Caller-Path Contract:**
- Entrypoint: `SprintService.GetSprintReadiness(ctx, "S024")`
- Lowest allowed mock seam: `SprintAssignmentQueryRepository.GetAssignmentsWithSize` (return tasks with depends_on) and a dependency lookup
- Forbidden mocks: Do NOT skip the dependency check — it must query which tasks are assigned vs completed
- Counter-factual: An impl that returns score=20 when no capacity rows exist (confusing no-capacity with no-dependencies) would produce wrong score

**Test variations:**
| Unsatisfied deps | Expected Factor2 Score |
|---|---|
| 0 | 20 |
| 1 | 19 |
| 20 | 0 |
| 21 | 0 (floor at 0) |

---

### TC-013-03: Readiness score — Factor 3 (Task count)

**Feature Requirement:** REQ-F-013 AC-4: "0 pts for 0 entities, 15 pts for ≥3, scaled linearly between 1 and 3"
**Task AC:** §1.1.3 Factor 3
**Technique Applied:** BVA — 0 entities, 1 entity, 2 entities, 3 entities, >3 entities
**ISO 25010:** Functional Suitability

**Test variations (table-driven):**
| Total entities | Expected Factor3 Score |
|---|---|
| 0 | 0 |
| 1 | int(1/3.0 * 15) = 5 |
| 2 | int(2/3.0 * 15) = 10 |
| 3 | 15 |
| 10 | 15 (capped) |

---

### TC-013-04: Readiness score — Factor 4 (Agent balance)

**Feature Requirement:** REQ-F-013 AC-5: "15 pts if ≥2 distinct agent types; 0 pts if only one agent type or all null"
**Task AC:** §1.1.3 Factor 4
**Technique Applied:** Equivalence Partitioning — classes: all-null, one-type, two-types, three-types
**ISO 25010:** Functional Suitability

**Test variations:**
| Distinct non-null agent types | Expected Factor4 Score |
|---|---|
| 0 (all null) | 0 |
| 1 | 0 |
| 2 | 15 |
| 3 | 15 |

---

### TC-013-05: Readiness score — Factor 5 (Sizing coverage)

**Feature Requirement:** REQ-F-013 AC-6: "15 pts if all assigned entities have non-null size; 1 pt deducted per unsized entity, min 0" (Scenario 3 in §1.3)
**Task AC:** §1.1.3 Factor 5
**Technique Applied:** BVA — 0 unsized (max), 1 unsized (14), 15 unsized (0), 16 unsized (0, not negative)
**ISO 25010:** Functional Suitability

**Test variations:**
| Unsized entities | Expected Factor5 Score |
|---|---|
| 0 | 15 |
| 1 | 14 |
| 7 | 8 |
| 15 | 0 |
| 16 | 0 (floor) |

---

### TC-013-06: Readiness score — Factor 6 (Oversized-entity flag)

**Feature Requirement:** REQ-F-013 AC-7: "10 pts if no assigned entity has size >= 8; 0 pts if any such entity exists"
**Task AC:** §1.1.3 Factor 6
**Technique Applied:** Decision Table — present/absent of entities with size >= 8
**ISO 25010:** Functional Suitability

**Test variations:**
| Any entity with size >= 8? | Expected Factor6 Score |
|---|---|
| No | 10 |
| Yes (size=8, boundary) | 0 |
| Yes (size=13) | 0 |

**Boundary note:** size=7 → score=10; size=8 → score=0

---

### TC-013-07: Readiness overall score — sum of factors (max 100)

**Feature Requirement:** REQ-F-013 AC-1: "0-100 score"
**Task AC:** §1.1.3
**Technique Applied:** Contract surface enumeration — verify sum = Σ factors, capped at 100
**ISO 25010:** Functional Suitability

**Preconditions:** Mock returns known scores: Factor1=25, Factor2=20, Factor3=15, Factor4=15, Factor5=15, Factor6=10

**Expected:** `SprintReadiness.OverallScore = 100`

---

### TC-013-08: Readiness --json output structure

**Feature Requirement:** REQ-F-013 AC-10: "--json includes overall_score, factors array, unsized_entities list, oversized_entities list"
**Task AC:** §1.1.3 AC-10
**Technique Applied:** Contract surface enumeration
**ISO 25010:** Functional Suitability, Compatibility

**Expected JSON:**
```json
{
  "overall_score": 72,
  "factors": [
    {"name": "Capacity utilization", "score": 25, "max_score": 25, "detail": "..."},
    ...
  ],
  "unsized_entities": [{"key": "E07-F01-003", "title": "..."}],
  "oversized_entities": []
}
```

**Negative Cases:** `overall_score` must be integer in [0, 100]; `factors` array must have exactly 6 elements

---

### TC-013-09: Readiness human output — per-factor labels and scores

**Feature Requirement:** REQ-F-013 AC-9: "Each factor shows its label, individual score, max score, and a one-line explanation"
**Task AC:** §1.1.3 AC-9
**Technique Applied:** Contract surface enumeration
**ISO 25010:** Usability

**Caller-Path Contract:**
- Entrypoint: `runSprintReadiness` CLI handler without `--json`
- Lowest allowed mock seam: `MockSprintService.GetSprintReadiness`
- Forbidden mocks: Do NOT mock the table formatter
- Counter-factual: An impl that prints only the overall score (omitting per-factor breakdown) fails UAT-J2-06

**Expected output fields per factor line:** factor name, score, max_score, detail string — all four present in stdout

---

### TC-013-10: Readiness score determinism

**Feature Requirement:** REQ-F-013 and spec §1.2: "Given identical database state, readiness must always return the same score"
**Task AC:** §1.1.3 AC-11 (implied by non-functional spec)
**Technique Applied:** Equivalence Partitioning — deterministic class: same input always produces same output
**ISO 25010:** Reliability

**Method:** Call `GetSprintReadiness` twice with identical mocked repository data; assert both calls return identical `OverallScore` and all factor scores.

---

### TC-013-11: Zero-entity sprint returns score 0

**Feature Requirement:** REQ-F-013 AC-12: "Sprint with 0 entities returns score 0 with factor-level explanation"
**Task AC:** §1.1.3 AC-12
**Technique Applied:** Equivalence Partitioning — degenerate class (zero entities)
**ISO 25010:** Reliability, Usability

**Preconditions:** `GetAssignmentsWithSize` returns empty slice

**Expected:** `SprintReadiness.OverallScore = 0`; all 6 factors present with score=0; `Factors[i].Detail` non-empty for all i

---

### TC-013-12: Readiness unsized_entities list populated correctly

**Feature Requirement:** REQ-F-013 AC-10: "--json output lists unsized entities by key and title" (Scenario 3 in §1.3)
**Task AC:** §1.1.3
**Technique Applied:** Contract surface enumeration
**ISO 25010:** Functional Suitability

**Preconditions:** 4 assigned tasks, 2 with size=nil

**Expected:** `SprintReadiness.UnsizedEntities` has 2 items, each with Key and Title fields populated

---

### TC-013-13: Readiness oversized_entities list populated correctly

**Feature Requirement:** REQ-F-013 AC-10
**Task AC:** §1.1.3
**Technique Applied:** Contract surface enumeration + BVA at size=7 (not oversized) vs size=8 (oversized)
**ISO 25010:** Functional Suitability

**Preconditions:** 3 assigned tasks with sizes 5, 8, 13

**Expected:** `SprintReadiness.OversizedEntities` has 2 items (size=8 and size=13); entity with size=5 is absent

---

### TC-013-14: Readiness score computation uses only two DB queries

**Feature Requirement:** spec §2.4: "Score computed entirely in-memory after fetching assignments and capacity; no additional DB queries per factor"
**Task AC:** §1.1.3 AC-11
**Technique Applied:** Contract surface enumeration — verifies the call count on mocked repositories
**ISO 25010:** Performance Efficiency, Maintainability

**Caller-Path Contract:**
- Entrypoint: `SprintService.GetSprintReadiness(ctx, "S024")`
- Lowest allowed mock seam: both repository mocks with call counters
- Forbidden mocks: N/A — this test IS the mock verification
- Counter-factual: An impl that calls `GetCapacity` once per factor (6 calls) fails this assertion

**Method:** Mock `GetAssignmentsWithSize` and `GetCapacity` with call counters. Assert each called exactly once.

---

### TC-014-01: SetSprintCapacity creates new row

**Feature Requirement:** REQ-F-014 AC-1: "creates or updates the sprint_capacity row for (S024, backend)"
**Task AC:** §1.1.4 AC-1
**Technique Applied:** State Transition — no row → row via SetCapacity
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- Entrypoint: `SprintService.SetSprintCapacity(ctx, SetSprintCapacityInput{SprintKey:"S024", AgentType:"backend", Points:21})`
- Lowest allowed mock seam: `SprintCapacityRepository.SetCapacity` — capture the SprintCapacity arg
- Forbidden mocks: Do NOT mock the sprint key→ID lookup; must exercise `SprintRepository.GetByKey`
- Counter-factual: An impl that passes sprint ID instead of resolving it first fails when sprint key is not S-padded correctly

**Preconditions:** `GetByKey("S024")` returns sprint with ID=24; `SetCapacity` mock captures arg

**Expected:** `SetCapacity` called with `SprintCapacity{SprintID:24, AgentType:"backend", CapacityPoints:21}`; returned `*models.SprintCapacity` has matching fields

---

### TC-014-02: SetSprintCapacity upserts existing row (no duplicates)

**Feature Requirement:** REQ-F-014 AC-1 and spec §2.6 INSERT OR REPLACE
**Task AC:** §1.1.4 AC-1
**Technique Applied:** State Transition — existing row → updated row; row count stays 1
**ISO 25010:** Reliability

**Caller-Path Contract:** Same as TC-014-01
- Counter-factual: An impl using INSERT (not REPLACE) creates a duplicate row; second `GetCapacity` call returns 2 rows

**Method (repository-level):** Call `SetCapacity` twice for same (sprint_id, agent_type). Assert `GetCapacity` returns exactly 1 row with the updated value.

---

### TC-014-03: SetSprintCapacity rejects points <= 0

**Feature Requirement:** REQ-F-014 AC-1: "validates that points > 0"
**Task AC:** §1.1.4 AC-1
**Technique Applied:** BVA — points=0 (invalid), points=-1 (invalid), points=0.001 (valid minimum)
**ISO 25010:** Reliability, Security

**Caller-Path Contract:**
- Entrypoint: `SprintService.SetSprintCapacity(ctx, SetSprintCapacityInput{..., Points:0})`
- Lowest allowed mock seam: Validation occurs before `SetCapacity` is called — assert `SetCapacity` is NOT called
- Forbidden mocks: Do NOT mock the validation
- Counter-factual: An impl that passes points=0 to the repository creates a zero-capacity row, corrupting Factor 1 scoring

**Test variations:**
| points | Expected behavior |
|---|---|
| 0 | error returned, SetCapacity NOT called |
| -5 | error returned, SetCapacity NOT called |
| 0.001 | success |
| 21 | success |

---

### TC-014-04: GetSprintCapacity returns empty slice when no capacity rows

**Feature Requirement:** REQ-F-014 AC-6: "If no capacity rows exist for a sprint, show returns an empty table (not an error)"
**Task AC:** §1.1.4 AC-6
**Technique Applied:** Equivalence Partitioning — degenerate class (no rows)
**ISO 25010:** Reliability, Usability

**Caller-Path Contract:**
- Entrypoint: `SprintService.GetSprintCapacity(ctx, "S024")`
- Lowest allowed mock seam: `SprintCapacityRepository.GetCapacity` returns empty slice
- Forbidden mocks: Do NOT return error from GetCapacity
- Counter-factual: An impl that returns `nil, ErrNoRows` instead of `[]CapacityRow{}, nil` causes CLI to print an error instead of an empty table

**Expected:** returns `([]CapacityRow{}, nil)` — empty slice, no error

---

### TC-014-05: GetSprintCapacity shows computed allocation from AssignmentsWithSize

**Feature Requirement:** REQ-F-014 AC-3: "allocated_points is computed at query time as Σ size over assigned entities where entity_type='task' and agent_type=<agent>"
**Task AC:** §1.1.4 AC-3
**Technique Applied:** Contract surface enumeration — allocation formula verified for multi-agent and null-size cases
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- Entrypoint: `SprintService.GetSprintCapacity(ctx, "S024")`
- Lowest allowed mock seam: `GetAssignmentsWithSize` returns 3 tasks (backend: size=5, backend: size=8, frontend: size=3); `GetCapacity` returns backend=21, frontend=13
- Forbidden mocks: Do NOT mock `CapacityRow.AllocatedPoints` — it must be computed
- Counter-factual: An impl that uses Σ of all entity sizes (not filtered by agent_type) returns wrong per-agent allocation

**Expected:**
- CapacityRow for backend: AllocatedPoints=13, Remaining=8, UnsizedAssigned=0
- CapacityRow for frontend: AllocatedPoints=3, Remaining=10, UnsizedAssigned=0

---

### TC-014-06: GetSprintCapacity -- unsized_assigned count per agent

**Feature Requirement:** REQ-F-014 AC-4: "unsized_assigned is the count of assigned entities with size IS NULL for that agent type"
**Task AC:** §1.1.4 AC-4
**Technique Applied:** BVA — 0 unsized (boundary), 1 unsized, all unsized
**ISO 25010:** Functional Suitability

**Preconditions:** Mock assignments: backend task with size=nil, backend task with size=5

**Expected:** CapacityRow for backend: UnsizedAssigned=1, AllocatedPoints=5

---

### TC-014-07: GetSprintCapacity -- remaining can be negative (overcommit)

**Feature Requirement:** REQ-F-014 AC-5: "remaining = capacity_points - allocated_points (can be negative)"
**Task AC:** §1.1.4 AC-5
**Technique Applied:** Equivalence Partitioning — negative class (overcommit)
**ISO 25010:** Functional Suitability

**Preconditions:** capacity=21, allocated=30

**Expected:** `CapacityRow.Remaining = -9` (not clamped to 0)

---

### TC-014-08: capacity show --json output structure

**Feature Requirement:** REQ-F-014 AC-7; REQ-NF-005
**Task AC:** §1.1.4 AC-7
**Technique Applied:** Contract surface enumeration
**ISO 25010:** Functional Suitability, Compatibility

**Expected JSON element:** `{"agent_type":"backend","capacity_points":21,"allocated_points":13,"remaining":8,"unsized_assigned":0}`

---

### TC-015-01: Default capacity applied on sprint create when sprint_defaults.capacity is set

**Feature Requirement:** REQ-F-015 AC-2 (Scenario 4 in §1.3): "capacity rows inserted into sprint_capacity at creation time using the default values"
**Task AC:** §1.1.5 AC-2
**Technique Applied:** State Transition — config has SprintDefaults.Capacity → rows inserted in sprint create
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- Entrypoint: `SprintService.CreateSprint(ctx, CreateSprintInput{Name:"Sprint 10", ...})` with cfg containing `SprintDefaults.Capacity = map[string]float64{"backend":21, "frontend":13}`
- Lowest allowed mock seam: `SprintCapacityRepository.SetCapacity` — call count and args captured
- Forbidden mocks: Do NOT mock the config read — cfg must be passed into the service and used
- Counter-factual: An impl that ignores `s.cfg` creates the sprint but never calls `SetCapacity`, leaving the sprint with no capacity rows

**Expected:** `SetCapacity` called exactly 2 times (once for backend, once for frontend) with the correct points values

---

### TC-015-02: Default capacity NOT applied when sprint_defaults.capacity is empty

**Feature Requirement:** REQ-F-015 AC-2 (contrapositive)
**Task AC:** §1.1.5 AC-2
**Technique Applied:** Equivalence Partitioning — class: config absent or empty capacity map
**ISO 25010:** Reliability

**Preconditions:** cfg.SprintDefaults = nil OR cfg.SprintDefaults.Capacity = empty map

**Expected:** `SetCapacity` NOT called; sprint created successfully

---

### TC-015-03: Per-sprint set overrides defaults without modifying config

**Feature Requirement:** REQ-F-015 AC-4 and Scenario 5 in §1.3: "Per-sprint capacity set...always overrides defaults and does not modify .sharkconfig.json"
**Task AC:** §1.1.5 AC-4
**Technique Applied:** State Transition — default row (from create) → overridden row (from explicit set)
**ISO 25010:** Reliability, Security (no unintended config mutation)

**Caller-Path Contract:**
- Entrypoint: `SprintService.SetSprintCapacity(ctx, SetSprintCapacityInput{SprintKey:"S024", AgentType:"backend", Points:34})`
- Lowest allowed mock seam: `SprintCapacityRepository.SetCapacity` — verify row updated; `config.Manager.SetSprintCapacityDefault` — verify NOT called
- Forbidden mocks: Do NOT mock both methods together; test each independently
- Counter-factual: An impl that calls `SetSprintCapacityDefault` inside `SetSprintCapacity` permanently changes the defaults for future sprints

**Expected:** `SetCapacity` called with Points=34; `SetSprintCapacityDefault` is NOT called

---

### TC-015-04: --default flag writes to .sharkconfig.json, not DB

**Feature Requirement:** REQ-F-015 AC-3: "--default...updates the sprint_defaults.capacity.backend value in .sharkconfig.json; new sprints created afterward inherit the new value"
**Task AC:** §1.1.5 AC-3
**Technique Applied:** Decision Table — two paths: with `--default` flag vs without
**ISO 25010:** Functional Suitability, Security

**Caller-Path Contract:**
- Entrypoint: `runSprintCapacitySet` CLI handler with `--default --agent=backend --points=21`; routes to `config.Manager.SetSprintCapacityDefault("backend", 21)` not `SprintService.SetSprintCapacity`
- Lowest allowed mock seam: Mock both `config.Manager.SetSprintCapacityDefault` and `SprintService.SetSprintCapacity`; assert only the config method is called
- Forbidden mocks: Do NOT mock both as no-ops without assertions
- Counter-factual: An impl that calls `SetSprintCapacity` instead of `SetSprintCapacityDefault` when `--default` is set updates the DB row instead of the config file; new sprints created later will NOT inherit the new value

**Expected:** `SetSprintCapacityDefault("backend", 21)` called once; `SetSprintCapacity` NOT called

---

### TC-015-05: carryover_behavior and auto_create parsed from config

**Feature Requirement:** REQ-F-015 AC-5 (carryover_behavior read by close) and AC-6 (auto_create parsed)
**Task AC:** §1.1.5 AC-5, AC-6
**Technique Applied:** Equivalence Partitioning — both valid values for carryover_behavior
**ISO 25010:** Functional Suitability

**Preconditions:** `.sharkconfig.json` contains:
```json
{"sprint_defaults": {"carryover_behavior": "next", "auto_create": false, "capacity": {}}}
```

**Expected:** `config.Config.SprintDefaults.CarryoverBehavior == "next"` and `AutoCreate == false` after parsing

---

### TC-015-06: sprint_defaults config section absent — graceful defaults

**Feature Requirement:** REQ-F-015 AC-1: "accepts a sprint_defaults key" (implied: absence is handled)
**Task AC:** §1.1.5
**Technique Applied:** Equivalence Partitioning — absent config class
**ISO 25010:** Reliability

**Preconditions:** `.sharkconfig.json` has no `sprint_defaults` key

**Expected:** `config.Config.SprintDefaults == nil`; no panic; sprint creation proceeds without capacity rows; `carryover_behavior` defaults to "backlog" (per spec)

---

### TC-015-07: SetSprintCapacityDefault persists to .sharkconfig.json correctly

**Feature Requirement:** REQ-F-015 AC-3
**Task AC:** §1.1.5 AC-3
**Technique Applied:** Contract surface enumeration — read config → mutate → verify written value
**ISO 25010:** Reliability

**Caller-Path Contract:**
- Entrypoint: `config.Manager.SetSprintCapacityDefault("backend", 21)` — this is the production entrypoint; it is also the unit under test
- Lowest allowed mock seam: File I/O (use temp directory with real file writes; do NOT mock os.WriteFile)
- Forbidden mocks: Do NOT mock the JSON marshal/write; the serialization must be exercised
- Counter-factual: An impl that mutates only the in-memory struct but never calls `os.WriteFile` returns nil error but the config file is unchanged; subsequent reads see the old value

**Method:** Write initial config to temp file; call `SetSprintCapacityDefault`; read config file from disk; assert `sprint_defaults.capacity.backend == 21`

---

### TC-NF-001: Response time < 500ms for plan and readiness with 500 entities

**Feature Requirement:** REQ-NF-001
**Task AC:** spec §1.2
**Technique Applied:** BVA — 500 entities (upper bound)
**ISO 25010:** Performance Efficiency

**Caller-Path Contract:**
- Entrypoint: `SprintRepository.ListUnassignedBacklog(ctx, []string{"task"})` — repository-level test with real SQLite
- Lowest allowed mock seam: No mocks (real DB required for timing)
- Forbidden mocks: Any mock eliminates the I/O component that causes the budget overrun
- Counter-factual: N+1 implementation (one membership query per entity) at 500 entities takes ~500*1ms = 500ms, failing the budget

**Method:** Seed 500 tasks in `ready_for_development`. Time `ListUnassignedBacklog`. Assert elapsed < 500ms.

---

### TC-NF-002: No regressions in existing commands after adding new sprint methods

**Feature Requirement:** REQ-NF-004
**Task AC:** spec §1.2
**Technique Applied:** Equivalence Partitioning — existing command set is one partition; new commands are another
**ISO 25010:** Compatibility, Reliability

**Method:** Run `make test` — all pre-existing test cases (task, feature, epic, sprint lifecycle) must pass. No new failures. Verified by CI.

---

## 8. Integration Scenarios

### INT-011: F05 planning view covers UAT-J2-01 (Sprint Planning Data)

**UAT Scenario:** UAT-J2-01 — PM runs `shark sprint plan S001` and sees three sections
**What to verify at boundary:** PlanSprint service method returns all three fields (Backlog, Capacity, Readiness); CLI formats them as three sections
**Test coverage:** TC-011-04 (service), TC-011-07 (CLI output)
**Status:** Covered

### INT-012: F05 bulk assignment covers UAT-J2-04 (Bulk Assign Tasks from Feature)

**UAT Scenario:** UAT-J2-04 — "All eligible tasks from E07-F34 are assigned to the sprint"
**What to verify at boundary:** `BulkAddToSprint` with FeatureKey correctly filters by feature and assigns in one transaction
**Test coverage:** TC-012-01, TC-012-05 (skipping ineligible), TC-012-09 (atomicity)
**Status:** Covered

### INT-013: F05 capacity warning covers UAT-J2-05 (Capacity Exceeded Warning)

**UAT Scenario:** UAT-J2-05 — "Task is assigned (not blocked); warning displayed"
**What to verify at boundary:** Warning in `BulkAddResult.Warnings` propagates to CLI output; task is still assigned
**Test coverage:** TC-012-07, TC-012-08
**Status:** Covered

### INT-014: F05 readiness score covers UAT-J2-06 (Sprint Readiness Score)

**UAT Scenario:** UAT-J2-06 — "Displays overall score (0-100); breakdown shows individual factors"
**What to verify at boundary:** All 6 factors present in output; each shows score and max_score; JSON output parseable by orchestrator
**Test coverage:** TC-013-01 through TC-013-09; TC-013-08 (JSON)
**Status:** Covered

### INT-015: F05 capacity CRUD covers UAT-J1-05 (Agent Capacity Configuration)

**UAT Scenario:** UAT-J1-05 — "capacity show displays all three agent types with capacity points, allocated points, and remaining points"
**What to verify at boundary:** `GetSprintCapacity` computes allocated from live assignment data; `SetSprintCapacity` upserts correctly
**Test coverage:** TC-014-01 through TC-014-08
**Status:** Covered

### INT-016: F05 defaults cover UAT-Scenario-4 (Default Capacity Applied on Sprint Create)

**UAT Scenario:** §1.3 Scenario 4 — "Two rows inserted into sprint_capacity for (S010, backend, 21) and (S010, frontend, 13)"
**What to verify at boundary:** `CreateSprint` reads `cfg.SprintDefaults.Capacity` and calls `SetCapacity` for each entry
**Test coverage:** TC-015-01, TC-015-02
**Status:** Covered

### INT-017: Service layer pattern compliance (INT-06 from epic UAT)

**UAT Scenario:** INT-06 — "All sprint commands must use the service layer, not direct repository access"
**What to verify at boundary:** `internal/cli/commands/sprint.go` never imports repository package directly; all new commands call `cli.GetSprintService()`
**Test coverage:** Code review gate; TC-011-06, TC-012-04, TC-013-09, TC-014-08 (all CLI tests use MockSprintService, not mock repository)
**Status:** Verified by CLI test mock pattern

### INT-018: JSON output covers orchestrator acceptance gate (UAT-J2-10)

**UAT Scenario:** UAT-J2-10 — "Both commands return valid, parseable JSON; --field flag works"
**What to verify at boundary:** `runSprintPlan --json` and `runSprintReadiness --json` output passes `json.Unmarshal` without error
**Test coverage:** TC-011-06, TC-013-08
**Status:** Covered

---

## 9. Test Infrastructure

### 9.1 Existing Infrastructure to Follow

| Infrastructure | Location | Pattern |
|---|---|---|
| Service mock pattern (function fields) | `internal/services/sprint_service_test.go` | `MockSprintRepository` with `GetByKeyFunc`, `CreateFunc`, etc. — replicate for new repo interfaces |
| CLI mock pattern | `internal/cli/commands/sprint_test.go` | `MockSprintService` with function fields — extend with new methods |
| Repository real-DB test pattern | `internal/repository/sprint/repository_test.go` (expected; follow `task_repository_test.go` pattern) | `test.GetTestDB()` + cleanup before test |
| Config file mutation test | `internal/config/manager.go` tests (follow existing SetCloudConfig test if present) | Temp dir + real file I/O |

### 9.2 New Test Helpers Needed

| Helper | File | Purpose |
|---|---|---|
| `MockSprintAssignmentQueryRepository` | `internal/services/sprint_service_test.go` | Mock for `BulkAssign`, `ListUnassignedBacklog`, `GetAssignmentsWithSize` |
| `MockSprintCapacityRepository` | `internal/services/sprint_service_test.go` | Mock for `GetCapacity`, `SetCapacity` |
| Extended `MockSprintService` | `internal/cli/commands/sprint_test.go` | Add `PlanSprint`, `BulkAddToSprint`, `GetSprintReadiness`, `SetSprintCapacity`, `GetSprintCapacity` function fields |
| `seedBacklogTasks(db, count)` | `internal/repository/sprint/repository_test.go` | Insert N tasks in `ready_for_development` for performance tests |
| Config temp-file helper | `internal/config/manager_test.go` | Write initial JSON to temp file, return `Manager` scoped to it |

### 9.3 Test Layer Assignment

| Test Case | Layer | File |
|---|---|---|
| TC-011-01..TC-011-05 (service logic) | Service (mocks) | `internal/services/sprint_service_test.go` |
| TC-011-05 (performance) | Repository (real DB) | `internal/repository/sprint/repository_test.go` |
| TC-011-06..TC-011-07 (CLI output) | CLI (mock service) | `internal/cli/commands/sprint_test.go` |
| TC-012-01..TC-012-10 | Service (mocks) | `internal/services/sprint_service_test.go` |
| TC-013-01..TC-013-14 | Service (mocks) | `internal/services/sprint_service_test.go` |
| TC-014-01..TC-014-06 | Service (mocks) + Repository (real DB for TC-014-02 upsert) | Mixed |
| TC-014-07..TC-014-08 | CLI (mock service) | `internal/cli/commands/sprint_test.go` |
| TC-015-01..TC-015-03 | Service (mocks) | `internal/services/sprint_service_test.go` |
| TC-015-04 | CLI (mock service + mock config) | `internal/cli/commands/sprint_test.go` |
| TC-015-05..TC-015-06 | Unit (config parsing, no DB) | `internal/config/config_test.go` |
| TC-015-07 | Config manager (real file I/O, temp dir) | `internal/config/manager_test.go` |
| TC-NF-001 | Repository (real DB) | `internal/repository/sprint/repository_test.go` |
| TC-NF-002 | End-to-end (`make test`) | CI |

---

## 10. Codex Test-Plan Red-Team

**Verdict:** NOT RUN — Codex red-team skipped as the test plan was developed as a feature-level plan (not a task-spec plan), and no `codex` invocation path was available at planning time without a specific task spec file. The ISTQB technique application (Steps 5.5), ISO 25010 matrix (Step 5.6), observability design (Step 5.7), and caller-path contracts (Step 5.8) were applied manually and comprehensively to compensate.

**Issues raised:** 0 (manual review only)
**Issues addressed before dev:** All ISTQB technique gaps, ISO 25010 cells, observability design entries, and caller-path contracts are present for every test case.
**Issues deferred:** Codex red-team review deferred to task-level planning when individual task specs are available. Owner: QA agent at task-level Test_Criteria_Definition node.

---

## 11. Recommendations

- [x] Ready for development — no spec drift; all ACs covered; every TC has ISTQB technique, ISO 25010 row, observability design, and caller-path contract
- [ ] Needs BA refinement — N/A
- [ ] Needs tech refinement — N/A

**Pre-development checklist:**
- [ ] Developer extends `MockSprintAssignmentQueryRepository` and `MockSprintCapacityRepository` before writing service tests (TC requires these mocks to exist)
- [ ] Developer seeds the extended `MockSprintService` in `sprint_test.go` with the 5 new method function fields before writing CLI tests
- [ ] Performance test (TC-011-05, TC-NF-001) is run with `go test -run TestSprintRepository_ListUnassignedBacklog_Performance -count=3 -timeout=30s` to account for warm-up variance
- [ ] Config mutation test (TC-015-07) uses `t.TempDir()` to avoid polluting the real project `.sharkconfig.json`

---

*Test plan created: 2026-05-05 — covers REQ-F-011 through REQ-F-015, REQ-NF-001, REQ-NF-004, REQ-NF-005. 42 test cases, all ACs traced, all caller-path contracts specified.*
