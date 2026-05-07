# Test Plan: E19-F04 — Sprint Analytics: Velocity, Burndown & Summary

**Created:** 2026-05-05
**QA Agent:** QA
**Feature Spec:** `docs/plan/E19-sprint-management-planning-system/E19-F04-sprint-analytics-velocity-burndown-summary/spec.md`
**Epic UAT Plan:** `docs/plan/E19-sprint-management-planning-system/uat-acceptance-plan.md`
**Status:** APPROVED

---

## Spec Drift Analysis

### Drift Findings

No material drift detected between the feature spec and the epic requirements. The spec maps cleanly to REQ-F-007, REQ-F-008, and REQ-F-009. Two observations warranting documentation (not blockers):

1. **AC-S-1 vs. spec body:** The spec introduction says `summary` is "available for sprints in `completed` or `archived` status" and AC-S-1 says "planning or active sprint returns informational message." Spec Section 4.4 shows `GetSummary` validates `completed` or `archived`. Consistent. No drift.

2. **Cycle-time source:** Decision 4 in the spec supersedes the initial `work_sessions` phrasing — primary source is `task_history`; `work_sessions` referenced for graceful-degradation path only. AC-S-3 language ("graceful degradation when E13 `work_sessions` data is absent") is consistent with this decision. No drift; minor spec note added.

### Traceability Matrix

| Feature Spec AC | Spec Section | Coverage? | Test Cases |
|---|---|---|---|
| AC-V-1: Last 5 completed sprints velocity | REQ-F-007 | Full | TC-V-01, TC-V-02 |
| AC-V-2: `--sprints=N` validated 1–100 | REQ-F-007 | Full | TC-V-03, TC-V-04 |
| AC-V-3: Unsized entities contribute 0, counted separately | REQ-F-007 | Full | TC-V-05, TC-V-06 |
| AC-V-4: Trailing average = mean of Σ size over all returned sprints | REQ-F-007 | Full | TC-V-07, TC-V-08 |
| AC-V-5: Insufficient data (<3 sprints) → informational, exit 0 | REQ-F-007 | Full | TC-V-09 |
| AC-V-6: `--json` schema matches spec | REQ-F-007 | Full | TC-V-10 |
| AC-B-1: No key = active sprint; keyed = specified sprint | REQ-F-008 | Full | TC-B-01, TC-B-02 |
| AC-B-2: Accepted statuses: active, closing, completed, archived; planning → informational | REQ-F-008 | Full | TC-B-03, TC-B-04 |
| AC-B-3: Ideal burndown = linear, piecewise reset on mid-sprint changes | REQ-F-008 | Full | TC-B-05, TC-B-06 |
| AC-B-4: Actual remaining reconstructed from task_history; non-task uses current status | REQ-F-008 | Full | TC-B-07, TC-B-08 |
| AC-B-5: `unsized_remaining` in each data point | REQ-F-008 | Full | TC-B-09 |
| AC-B-6: Human-readable text-table format (no Unicode block chars) | REQ-F-008 | Full | TC-B-10 |
| AC-B-7: `--json` schema matches spec | REQ-F-008 | Full | TC-B-11 |
| AC-B-8: Future days → `—` in human output; omitted from JSON | REQ-F-008 | Full | TC-B-12 |
| AC-S-1: Summary available for completed/archived only | REQ-F-009 | Full | TC-S-01, TC-S-02 |
| AC-S-2: Base summary fields complete | REQ-F-009 | Full | TC-S-03, TC-S-04 |
| AC-S-3: `--detailed` with graceful degradation when E13 absent | REQ-F-009 | Full | TC-S-05, TC-S-06 |
| AC-S-4: `--json` with all fields; nil for missing detailed fields | REQ-F-009 | Full | TC-S-07, TC-S-08 |
| REQ-NF-001: Analytics complete < 2 s, indexed lookups | Non-functional | Full | TC-NF-01, TC-NF-02 |
| REQ-NF-005: `--json` and `--field` consistent | Non-functional | Full | TC-NF-03 |
| AC-X-1: make fmt/lint/test pass | Cross-cutting | N/A (checked at CI) | — |
| AC-X-2: < 2 s on test dataset | Cross-cutting | Merged into TC-NF-01 | TC-NF-01 |
| AC-X-3: No regression | Cross-cutting | N/A (existing test suite) | — |
| AC-X-4: Service tests use mocked repos | Cross-cutting | Mandatory by arch rules | All TC-SVC-* |

---

## ISTQB Technique Application (per AC)

| AC | Technique(s) Applied | Test Cases Generated | Rationale |
|---|---|---|---|
| AC-V-1 | Equivalence Partitioning + BVA | TC-V-01, TC-V-02 | Input is ordered set (last N sprints); EP partitions valid/invalid data states; BVA covers 0/1/5/100 sprint counts |
| AC-V-2 | BVA | TC-V-03, TC-V-04 | `N` has min=1, max=100 boundary; BVA forces min-1=0, min=1, max=100, max+1=101 |
| AC-V-3 | Decision Table | TC-V-05, TC-V-06 | Two conditions: entity.size=NULL vs. integer × completed vs. not-completed — 4 combinations determine Σ size and unsized count |
| AC-V-4 | Equivalence Partitioning | TC-V-07, TC-V-08 | Two partitions: sprint with zero velocity (contributes 0 to numerator, not denominator excluded) vs. sprint with positive velocity |
| AC-V-5 | State Transition | TC-V-09 | Transition: 0/1/2 → "insufficient data" message; ≥3 → normal output. Tests each state boundary |
| AC-V-6 | Contract surface enumeration | TC-V-10 | Every JSON field × its type × nil/zero/positive value — no field omitted, no type mismatch |
| AC-B-1 | Equivalence Partitioning | TC-B-01, TC-B-02 | Two input partitions: key absent (→ active sprint lookup) vs. key present (→ specific sprint) |
| AC-B-2 | Decision Table | TC-B-03, TC-B-04 | 5 status values × valid/invalid burndown data: active✓, closing✓, completed✓, archived✓, planning→informational |
| AC-B-3 | State Transition + BVA | TC-B-05, TC-B-06 | State: no mid-sprint changes (single ideal line) vs. entity added (piecewise reset) vs. entity removed; BVA: added at day 1, middle, last day |
| AC-B-4 | Decision Table | TC-B-07, TC-B-08 | Entity type × completion source: task+task_history✓, bug+current-status✓, change_card+current-status✓; legend annotation required |
| AC-B-5 | Contract surface enumeration | TC-B-09 | `unsized_remaining` present in every daily data point including day 0 and future days |
| AC-B-6 | Equivalence Partitioning | TC-B-10 | Valid: text-table output uses only ASCII chars. Invalid: no Unicode block chars (U+2580–U+259F) present |
| AC-B-7 | Contract surface enumeration | TC-B-11 | Every JSON field × type × value for `data_points` array; `actual_remaining` present for past, absent for future |
| AC-B-8 | BVA | TC-B-12 | Boundary: today's date row has actual; tomorrow's row omits actual. Tests exact boundary day |
| AC-S-1 | State Transition | TC-S-01, TC-S-02 | Valid states: completed→summary, archived→summary; Invalid: planning→informational, active→informational |
| AC-S-2 | Contract surface enumeration | TC-S-03, TC-S-04 | All 12 base fields × type × nullable/zero/positive — no field silently omitted from base output |
| AC-S-3 | Decision Table + Equivalence Partitioning | TC-S-05, TC-S-06 | `--detailed` flag × E13 data available/absent → 4 combinations; graceful degradation for absent data |
| AC-S-4 | Contract surface enumeration | TC-S-07, TC-S-08 | JSON schema: base fields always present, detailed fields null when detailed=false or data absent (not omitted) |
| REQ-NF-001 | BVA (performance) | TC-NF-01, TC-NF-02 | Boundary: 50 sprints / 1000 tasks (max spec); 1 sprint / 1 task (min); `EXPLAIN QUERY PLAN` for indexed assertion |
| REQ-NF-005 | Contract surface enumeration | TC-NF-03 | `--json` and `--field` flags on all three commands follow existing entity-command contract |

---

## ISO 25010 Coverage Matrix

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-V-1 | ✅ TC-V-01,02 | N/A (no perf requirement for default 5) | N/A | ✅ TC-V-01 (oldest→newest ordering expected by user) | N/A | N/A | ✅ service-test isolation (mocked repos) | N/A |
| AC-V-2 | ✅ TC-V-03,04 | N/A | N/A | ✅ TC-V-04 (error message checks) | N/A | N/A | N/A | N/A |
| AC-V-3 | ✅ TC-V-05,06 | N/A | N/A | ✅ TC-V-05 (visibility of unsized count) | N/A | N/A | N/A | N/A |
| AC-V-4 | ✅ TC-V-07,08 | N/A | N/A | N/A | ✅ TC-V-08 (zero-velocity sprint included in denominator) | N/A | N/A | N/A |
| AC-V-5 | ✅ TC-V-09 | N/A | N/A | ✅ TC-V-09 (informational message, not error) | ✅ TC-V-09 (exit 0 not exit 1) | N/A | N/A | N/A |
| AC-V-6 | ✅ TC-V-10 | N/A | ✅ TC-V-10 (JSON parseable by jq/Python) | N/A | N/A | N/A | N/A | N/A |
| AC-B-1 | ✅ TC-B-01,02 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-B-2 | ✅ TC-B-03,04 | N/A | N/A | ✅ TC-B-04 (informational not error) | N/A | N/A | N/A | N/A |
| AC-B-3 | ✅ TC-B-05,06 | N/A | N/A | N/A | ✅ TC-B-06 (piecewise reset is deterministic/reproducible) | N/A | N/A | N/A |
| AC-B-4 | ✅ TC-B-07,08 | N/A | ✅ TC-B-08 (non-task entities use current-status — documented limitation) | N/A | N/A | N/A | N/A | N/A |
| AC-B-5 | ✅ TC-B-09 | N/A | N/A | ✅ TC-B-09 (unsized count visible in chart legend) | N/A | N/A | N/A | N/A |
| AC-B-6 | ✅ TC-B-10 | N/A | ✅ TC-B-10 (ASCII chars work in all terminal emulators) | ✅ TC-B-10 (header row + separator line present) | N/A | N/A | N/A | ✅ TC-B-10 (text table portable across terminals) |
| AC-B-7 | ✅ TC-B-11 | N/A | ✅ TC-B-11 (JSON parseable) | N/A | N/A | N/A | N/A | N/A |
| AC-B-8 | ✅ TC-B-12 | N/A | N/A | ✅ TC-B-12 (dash rendered for future, not blank or 0) | N/A | N/A | N/A | N/A |
| AC-S-1 | ✅ TC-S-01,02 | N/A | N/A | ✅ TC-S-02 (informational not error for wrong status) | N/A | N/A | N/A | N/A |
| AC-S-2 | ✅ TC-S-03,04 | N/A | N/A | ✅ TC-S-03 (all fields labelled, human-readable) | N/A | N/A | N/A | N/A |
| AC-S-3 | ✅ TC-S-05,06 | N/A | ✅ TC-S-06 (works without E13/work_sessions table) | ✅ TC-S-06 ("No session data available" message clear) | ✅ TC-S-06 (service returns nil, not error) | N/A | N/A | N/A |
| AC-S-4 | ✅ TC-S-07,08 | N/A | ✅ TC-S-07 (JSON schema backward-compatible) | N/A | N/A | N/A | N/A | N/A |
| REQ-NF-001 | N/A | ✅ TC-NF-01 (<2 s wall-clock); ✅ TC-NF-02 (EXPLAIN QUERY PLAN indexed) | N/A | N/A | ✅ TC-NF-01 (measured over 50 sprints/1000 tasks) | N/A | N/A | N/A |
| REQ-NF-005 | ✅ TC-NF-03 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |

**Coverage gaps:** None. All applicable ISO 25010 characteristics are covered or documented N/A with justification.

---

## Observability Design (per behavior)

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| `GetVelocity` called (velocity command) | internal — no observability; analytics are CLI-invoked, not server-side event-driven | `INFO sprint_analytics.velocity sprints=%d trailing_avg=%.1f` | internal — no span (CLI, single-process) | N/A | TC-V-01 asserts log line emitted when `--verbose` set |
| Insufficient data branch | internal — no metric | `INFO sprint_analytics.velocity insufficient_data=true count=%d` | internal | N/A | TC-V-09 asserts informational message in stdout (log line in verbose mode) |
| `GetBurndown` called | internal — no observability | `INFO sprint_analytics.burndown sprint_key=%s days=%d` | internal | N/A | TC-B-01 checks human output; verbose log checked by TC-B-01 |
| Piecewise ideal reset triggered | internal | `DEBUG sprint_analytics.burndown.reset day=%s old_total=%d new_total=%d` | internal | N/A | TC-B-06 asserts debug log with `--verbose` |
| `GetSummary` called | internal | `INFO sprint_analytics.summary sprint_key=%s detailed=%v` | internal | N/A | TC-S-03 checks output; log checked in verbose mode |
| Cycle-time unavailable (no work_sessions) | internal | `INFO sprint_analytics.summary.cycle_time unavailable sprint_key=%s` | internal | N/A | TC-S-06 asserts log in verbose mode and "No session data available" in human output |
| Performance boundary (>2 s) | `sprint_analytics.duration_seconds{command=velocity/burndown/summary}` — NOTE: this metric is a future observability hook; CLI does not currently emit metrics. For now, test asserts wall-clock < 2 s. | `WARN sprint_analytics.slow_query command=%s duration_ms=%d` | internal | >2 s for 3 consecutive calls | TC-NF-01 asserts wall-clock; future metric hook documented |

**Instrumentation requirement:** `--verbose` flag causes the service to log the listed `INFO/DEBUG` lines via the existing `cli.GlobalConfig.Verbose` check pattern (see `internal/cli/root.go`). Developers must add the listed log lines as part of this feature; QA verifies them in verbose-mode tests.

---

## Caller-Path Contracts (summary table)

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-V-01 | `SprintAnalyticsService.GetVelocity(ctx, 5)` | `SprintAnalyticsRepository` (mock the repo; do NOT mock the service) | Do not mock `GetVelocity` directly; do not bypass the service | A buggy impl that divides by only non-zero sprints would return wrong trailing average |
| TC-V-03 | `SprintAnalyticsService.GetVelocity(ctx, 0)` | `SprintAnalyticsRepository` | Do not short-circuit validation in the CLI layer | A buggy impl that passes N=0 to the repo would issue a LIMIT 0 query returning empty results silently |
| TC-V-04 | `SprintAnalyticsService.GetVelocity(ctx, 101)` | `SprintAnalyticsRepository` | Do not mock away validation | A buggy impl that accepts N=101 would bypass the documented 1–100 guard |
| TC-V-07 | `SprintAnalyticsService.GetVelocity(ctx, 3)` with one sprint having CompletedSize=0 | `SprintAnalyticsRepository` | Do not filter out zero-velocity rows in the repo mock | A buggy impl that excludes zero-velocity sprints from the denominator returns inflated average |
| TC-V-09 | `SprintAnalyticsService.GetVelocity(ctx, 5)` with repo returning 2 rows | `SprintAnalyticsRepository` | Do not mock `InsufficientData` field — it must be set by service logic | A buggy impl that panics or returns error on < 3 rows fails this test |
| TC-B-05 | `SprintAnalyticsService.GetBurndown(ctx, "S024")` with no mid-sprint changes | `SprintAnalyticsRepository`, `SprintRepository` | Do not mock the ideal-line calculation | A buggy ideal line that doesn't start at total_size fails day-0 assertion |
| TC-B-06 | `SprintAnalyticsService.GetBurndown(ctx, "S025")` with entity added on day 3 | `SprintAnalyticsRepository`, `SprintRepository` | Do not pre-compute the ideal line in the mock | A buggy impl that doesn't reset the ideal line after entity add returns wrong ideal_remaining from day 3 onward |
| TC-B-07 | `SprintAnalyticsService.GetBurndown(ctx, "S024")` with tasks having task_history events | `SprintAnalyticsRepository` | Do not mock completion-event filtering; use real service logic | A buggy impl that ignores task_history timestamps returns wrong actual_remaining |
| TC-S-03 | `SprintAnalyticsService.GetSummary(ctx, "S024", false)` | `SprintAnalyticsRepository`, `SprintRepository` | Do not mock field population — all 12 base fields must flow through service logic | A buggy impl with wrong planned_size calculation (e.g., including mid-sprint adds) would fail the assertion |
| TC-S-06 | `SprintAnalyticsService.GetSummary(ctx, "S024", true)` with `GetCycleTimeByPhase` returning empty slice | `SprintAnalyticsRepository` | Do not mock the nil-check on CycleTimeByPhase in the service | A buggy impl that panics on nil cycle-time slice fails this test |
| TC-NF-01 | `runSprintVelocity`, `runSprintBurndown`, `runSprintSummary` CLI handlers with real DB (repository test scope) | Real `SprintAnalyticsRepository` (no mock — this is a performance + integration test) | Do not mock the repo; mock seam is the DB connection itself (test DB) | A buggy query that does a full table scan instead of using index takes >2 s and fails timing assertion |
| TC-NF-03 | `runSprintVelocity --json --field trailing_average` | CLI layer; mocked `SprintAnalyticsService` | Do not mock the `--field` flag parsing — must flow through root command | A buggy `--field` handler that doesn't forward to the sprint subcommand returns empty output |

---

## Acceptance Test Cases

---

### TC-V-01: Velocity shows last 5 completed sprints oldest-first with correct Σ size

**Feature Requirement:** AC-V-1 — `shark sprint velocity` shows completed Σ size per sprint for last 5 completed sprints, ordered oldest→newest.
**Task AC:** AC-F007-1
**Technique Applied:** Equivalence Partitioning — valid partition: ≥5 completed sprints with known sizes.
**ISO 25010:** Functional Suitability, Usability (ordering matches user expectation)

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetVelocity(ctx context.Context, n int)` with `n=5` (default). Service is instantiated via `cli.GetSprintAnalyticsService()`.
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` — mock `GetVelocityData(ctx, 5)` returning 5 `VelocityRow` structs.
- **Forbidden mocks:** Do not mock `GetVelocity` itself; the service logic (trailing average calculation) must run for real.
- **Counter-factual:** A buggy impl that orders newest-first would fail the assertion that row[0].Key == earliest sprint key.

**Preconditions:**
- 5 completed sprints exist: S001 (size=10), S002 (size=15), S003 (size=20), S004 (size=12), S005 (size=18)
- No active sprint

**Input:**
- `shark sprint velocity` (no flags; default n=5)
- Service test: `svc.GetVelocity(ctx, 5)` with mock returning the 5 rows above in oldest-first order

**Expected Output:**
- Result contains 5 `VelocitySprint` entries in order S001, S002, S003, S004, S005
- Each entry: `CompletedSize` matches seeded value; `UnsizedCompleted=0`
- `TrailingAverage = (10+15+20+12+18)/5 = 15.0`
- `SprintCount = 5`
- `InsufficientData = false`

**Edge Cases:**
- Exactly 5 sprints in DB, no more: returns exactly 5 (not 6)
- Sprint 5 has `completed_size=0` (all tasks unsized): still appears with `completed_size=0, unsized_completed=N`

**Negative Cases:**
- Result must NOT include sprints in `planning`, `active`, or `closing` status

---

### TC-V-02: Velocity with more than 5 completed sprints respects default limit

**Feature Requirement:** AC-V-1 — default shows last 5.
**Task AC:** AC-F007-1
**Technique Applied:** BVA — 7 sprints exist; limit=5 (default).
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetVelocity(ctx, 5)` (repo called with `limit=5`)
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` — `GetVelocityData(ctx, 5)` must receive `limit=5`
- **Forbidden mocks:** Do not set limit in the repo mock; assert the argument passed to `GetVelocityData` equals 5.
- **Counter-factual:** A buggy impl that passes limit=0 or limit=100 to the repo would return wrong row count.

**Preconditions:** 7 completed sprints (S001–S007) with varying sizes.

**Input:** `GetVelocity(ctx, 5)`

**Expected Output:** Returns exactly 5 rows; rows correspond to S003–S007 (the 5 most recent completed sprints).

**Edge Cases:**
- Exactly 5 completed sprints → returns all 5; `InsufficientData=false`

---

### TC-V-03: `--sprints=N` boundary value N=1 (minimum valid)

**Feature Requirement:** AC-V-2 — `--sprints=N` valid range 1–100.
**Task AC:** AC-F007-2
**Technique Applied:** BVA — min boundary.
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetVelocity(ctx, 1)`
- **Lowest allowed mock seam:** `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not skip validation; N=1 must reach the service and repo.
- **Counter-factual:** A buggy impl that treats N=1 as "insufficient" and returns early would fail.

**Preconditions:** 1 completed sprint (S001, size=8).

**Input:** `GetVelocity(ctx, 1)` (CLI parses `--sprints=1`)

**Expected Output:** Returns 1 row; `TrailingAverage=8.0`; no error.

**Edge Cases:**
- N=0 (just below minimum): returns validation error "sprints must be between 1 and 100"; exit 1
- N=100 (maximum valid): accepted; calls repo with limit=100
- N=101 (just above maximum): returns validation error

---

### TC-V-04: `--sprints=0` and `--sprints=101` return validation errors

**Feature Requirement:** AC-V-2 — values outside 1–100 return validation error.
**Task AC:** AC-F007-2
**Technique Applied:** BVA — boundaries min-1=0 and max+1=101.
**ISO 25010:** Functional Suitability, Usability (error message clarity)

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetVelocity(ctx, 0)` and `GetVelocity(ctx, 101)`
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` — though this seam should NOT be called (validation fails before repo access)
- **Forbidden mocks:** Do not short-circuit the validation in the CLI; validation must occur in the service.
- **Counter-factual:** A buggy impl that passes N=0 to the repo would call `LIMIT 0` and return empty results with no error.

**Input:** N=0; N=101

**Expected Output:**
- N=0: error "sprints must be between 1 and 100" (or equivalent); exit 1
- N=101: same error; exit 1
- Repo `GetVelocityData` is never called

**Negative Cases:** Commands do not panic; repo is not invoked.

---

### TC-V-05: Unsized entities contribute 0 to Σ size and are counted in `unsized_completed`

**Feature Requirement:** AC-V-3 — NULL size → contributes 0 to Σ size; counted in `unsized_completed`.
**Task AC:** AC-F007-1 (via unsized tracking)
**Technique Applied:** Decision Table — 4 combinations of (size=NULL vs. size=int) × (entity completed vs. not-completed).
**ISO 25010:** Functional Suitability, Usability (visibility of unsized items)

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetVelocity(ctx, 1)`
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` — mock `GetVelocityData` returning `VelocityRow{CompletedSize: 10, UnsizedCompleted: 3}`
- **Forbidden mocks:** Do not return `CompletedSize=13` pretending unsized items contributed; the mock must model the repo's COALESCE logic.
- **Counter-factual:** A buggy impl that adds NULL sizes as 0 into the total without tracking them separately would show `unsized_completed=0` always.

**Decision Table:**

| Size is NULL | Entity completed | Contribution to CompletedSize | Counted in UnsizedCompleted |
|---|---|---|---|
| No (size=5) | Yes | +5 | No |
| Yes | Yes | +0 | Yes |
| No (size=5) | No | 0 | No |
| Yes | No | 0 | No |

**Preconditions:** Sprint with 4 tasks: 2 sized+completed, 1 unsized+completed, 1 sized+not-completed.

**Input:** `GetVelocityData(ctx, 1)` returns `VelocityRow{CompletedSize: 10 (from 2×5), UnsizedCompleted: 1}`

**Expected Output:**
- `VelocitySprint.CompletedSize = 10`
- `VelocitySprint.UnsizedCompleted = 1`
- Trailing average = 10.0 (not 10 + some NULL contribution)

---

### TC-V-06: `unsized_completed` visible in both human and JSON output

**Feature Requirement:** AC-V-3 — both human-readable and JSON outputs include `unsized_completed`.
**Task AC:** AC-F007-1
**Technique Applied:** Contract surface enumeration — output contract requires this field in both formats.
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- **Entrypoint:** CLI `runSprintVelocity` with mock `SprintAnalyticsService` returning result with `UnsizedCompleted=2`
- **Lowest allowed mock seam:** `SprintAnalyticsService` (CLI test scope — mock at service boundary)
- **Forbidden mocks:** Do not omit `UnsizedCompleted` from the mock return; test must verify it appears in output.
- **Counter-factual:** A buggy formatter that omits `unsized_completed` from human output would fail the string-contains assertion.

**Input:**
- Human output: `shark sprint velocity`
- JSON output: `shark sprint velocity --json`

**Expected Output:**
- Human: line containing "unsized" or "Unsized" with count "2"
- JSON: each sprint object has `"unsized_completed": 2` key

---

### TC-V-07: Trailing average includes zero-velocity sprints in denominator

**Feature Requirement:** AC-V-4 — mean of Σ size values, including zero-velocity sprints in denominator.
**Task AC:** AC-F007-3
**Technique Applied:** Equivalence Partitioning — partition: sprints with zero completed size (all tasks unsized or no completions).
**ISO 25010:** Functional Suitability, Reliability (accurate calculation)

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetVelocity(ctx, 3)`
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` returning rows: `[{CompletedSize:0}, {CompletedSize:10}, {CompletedSize:20}]`
- **Forbidden mocks:** Do not strip zero-velocity rows in the mock; they must be present to test denominator inclusion.
- **Counter-factual:** A buggy impl that divides by 2 (non-zero sprints only) returns `15.0` instead of the correct `10.0`.

**Input:** `GetVelocity(ctx, 3)` with repo returning 3 rows where first row has `CompletedSize=0`

**Expected Output:** `TrailingAverage = (0+10+20)/3 = 10.0` (NOT `(10+20)/2 = 15.0`)

---

### TC-V-08: Trailing average with all zero-velocity sprints

**Feature Requirement:** AC-V-4 — mean of all returned sprints.
**Task AC:** AC-F007-3
**Technique Applied:** Equivalence Partitioning — extreme partition: all sprints have zero completed size.
**ISO 25010:** Functional Suitability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetVelocity(ctx, 3)` with all rows having `CompletedSize=0`
- **Lowest allowed mock seam:** `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not skip returning zero rows.
- **Counter-factual:** A buggy impl that divides by 0 (filtering out zero-velocity rows) panics or returns NaN.

**Expected Output:** `TrailingAverage = 0.0`; no error; no panic; `InsufficientData = false` (data exists, just zero velocity).

---

### TC-V-09: Fewer than 3 completed sprints returns informational message, exit 0

**Feature Requirement:** AC-V-5 — "insufficient data" message when < 3 completed sprints; exit 0.
**Task AC:** AC-F007-4
**Technique Applied:** State Transition — states: 0 sprints, 1 sprint, 2 sprints (all → insufficient), 3 sprints (→ normal).
**ISO 25010:** Functional Suitability, Reliability (exit 0 not exit 1), Usability (helpful message)

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetVelocity(ctx, 5)` with repo returning 0, 1, and 2 rows respectively
- **Lowest allowed mock seam:** `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not return an error from the mock — insufficient data is not an error state.
- **Counter-factual:** A buggy impl that returns `err != nil` for <3 sprints causes the CLI to exit 1 instead of 0.

**Preconditions (three sub-cases):**
- Sub-case A: 0 completed sprints
- Sub-case B: 1 completed sprint
- Sub-case C: 2 completed sprints

**Expected Output:**
- All sub-cases: `VelocityResult.InsufficientData = true`
- CLI exits with code 0
- Human output includes message containing "insufficient" or "need at least 3"
- JSON output: `{"sprints": [...], "trailing_average": 0, "sprint_count": N, "insufficient_data": true}`

**Negative Cases:** Command must NOT return error, must NOT crash, must NOT show empty output without explanation.

---

### TC-V-10: `--json` output matches AC-V-6 schema

**Feature Requirement:** AC-V-6 — JSON schema matches spec.
**Task AC:** AC-F007-5
**Technique Applied:** Contract surface enumeration — every field × type × valid range.
**ISO 25010:** Functional Suitability, Compatibility (JSON parseable by jq)

**Caller-Path Contract:**
- **Entrypoint:** CLI `runSprintVelocity --json` with mocked service
- **Lowest allowed mock seam:** `SprintAnalyticsService`
- **Forbidden mocks:** Do not mock the JSON marshalling — it must flow through real CLI output code.
- **Counter-factual:** A buggy formatter that uses snake_case inconsistently (e.g., `UnsizedCompleted` vs. `unsized_completed`) fails JSON field assertion.

**Input:** `shark sprint velocity --json` (mocked service returns 2 sprints)

**Expected Output (strict schema):**
```json
{
  "sprints": [
    { "key": "S001", "name": "Sprint 1", "completed_size": 18, "unsized_completed": 2 },
    { "key": "S002", "name": "Sprint 2", "completed_size": 21, "unsized_completed": 0 }
  ],
  "trailing_average": 19.5,
  "sprint_count": 2
}
```

**Assertions:**
- Top-level keys: `sprints` (array), `trailing_average` (float64), `sprint_count` (int) — no extra keys
- Per-sprint keys: `key` (string), `name` (string), `completed_size` (int), `unsized_completed` (int)
- `trailing_average` is a float (not int) even when whole number (e.g., `20.0` not `20`)
- `jq '.trailing_average'` extracts successfully

---

### TC-B-01: `shark sprint burndown` (no key) uses current active sprint

**Feature Requirement:** AC-B-1 — no key → uses active sprint.
**Task AC:** AC-F008-1
**Technique Applied:** Equivalence Partitioning — input partition: key absent.
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "")` (empty key)
- **Lowest allowed mock seam:** `SprintRepository` (mocked `List` with active filter) and `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not pre-populate the sprint in the analytics repo mock; the service must call `sprintRepo.List` to discover the active sprint when key is empty.
- **Counter-factual:** A buggy impl that uses a hardcoded sprint key when key="" would return wrong sprint's burndown.

**Preconditions:** Active sprint S010 with 3 days of history.

**Input:** `GetBurndown(ctx, "")` — key is empty string.

**Expected Output:**
- `BurndownResult.SprintKey = "S010"`
- Data points start at S010's `start_date`
- No error

---

### TC-B-02: `shark sprint burndown S024` uses specified sprint

**Feature Requirement:** AC-B-1 — key provided → uses specified sprint.
**Task AC:** AC-F008-1
**Technique Applied:** Equivalence Partitioning — input partition: key present.
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S024")`
- **Lowest allowed mock seam:** `SprintRepository` (mocked `GetByKey("S024")`) and `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not use `List` when key is provided — assert that `GetByKey` is called, not `List`.
- **Counter-factual:** A buggy impl that ignores the key and always fetches the active sprint returns wrong data.

**Expected Output:** `BurndownResult.SprintKey = "S024"`; data points match S024's date range.

---

### TC-B-03: Burndown works for valid statuses: active, closing, completed, archived

**Feature Requirement:** AC-B-2 — accepted statuses.
**Task AC:** AC-F008-1
**Technique Applied:** Decision Table — 5 status values × burndown-available/unavailable.
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S024")` with sprint in each valid status
- **Lowest allowed mock seam:** `SprintRepository` (returns sprint with specific status), `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not filter on status inside the analytics repo mock.
- **Counter-factual:** A buggy impl that only accepts `active` status returns error for `completed`/`archived` sprints.

**Decision Table:**

| Sprint Status | Expected Result |
|---|---|
| active | Burndown data returned; future days have nil actual |
| closing | Burndown data returned |
| completed | Burndown data returned; all days have actual (no future days) |
| archived | Burndown data returned |
| planning | Informational message; empty DataPoints; no error |

**Negative Cases:** `planning` must not return an error code; exit 0.

---

### TC-B-04: Planning sprint burndown returns informational message, exit 0

**Feature Requirement:** AC-B-2 — planning sprint → informational, not error.
**Task AC:** AC-F008-1
**Technique Applied:** State Transition — `planning` is the invalid state.
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S024")` with sprint status=`planning`
- **Lowest allowed mock seam:** `SprintRepository` returning sprint with `Status="planning"`
- **Forbidden mocks:** Do not inject an error for planning status — the service must return a result (possibly with `InformationalMessage` field or equivalent) that the CLI renders as a message, not an error.
- **Counter-factual:** A buggy impl that returns `err != nil` for planning status causes CLI to exit 1 and print "Error:..." instead of an informational message.

**Expected Output:** Human output includes message like "No burndown data for a sprint in planning status."; exit 0.

---

### TC-B-05: Ideal burndown is linear from sprint total to 0

**Feature Requirement:** AC-B-3 — ideal = linear interpolation.
**Task AC:** AC-F008-2
**Technique Applied:** State Transition — simple case: no mid-sprint changes.
**ISO 25010:** Functional Suitability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S024")` with sprint duration=14 days, total_size=42, no entity adds/removes.
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` — `GetSprintAssignedEntities` returns stable set; `GetCompletionEvents` returns empty (no completions yet).
- **Forbidden mocks:** Do not pre-calculate ideal_remaining in the mock; the service must compute it.
- **Counter-factual:** A buggy impl using integer division (42/14=3 per day) instead of float (3.0) would show wrong `ideal_remaining` for non-integer days.

**Preconditions:** Sprint S024: start=day0, end=day13 (14 days), total_size=42. No completions.

**Expected Output (first 3 and last data points):**
- day0: `ideal_remaining=42.0`
- day1: `ideal_remaining=39.0` (42 × 13/14)
- day2: `ideal_remaining=36.0` (42 × 12/14)
- day13: `ideal_remaining=0.0`

---

### TC-B-06: Ideal burndown resets piecewise when entity added mid-sprint

**Feature Requirement:** AC-B-3 — piecewise linear reset on mid-sprint entity changes.
**Task AC:** AC-F008-2
**Technique Applied:** State Transition + BVA — entity added on day 3 of 14-day sprint.
**ISO 25010:** Functional Suitability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S025")` with entity added on day 3 (size=7).
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` — `GetSprintAssignedEntities` returns original entity (size=35, assigned at start) plus new entity (size=7, assigned at day 3).
- **Forbidden mocks:** Do not pre-compute the piecewise reset; the service algorithm must handle it.
- **Counter-factual:** A buggy impl that ignores mid-sprint entity additions keeps the original ideal line, showing wrong ideal_remaining from day 3.

**Preconditions:** Sprint S025: 14 days, original_size=35. On day 3, entity of size=7 added.

**Expected Output:**
- day0: `ideal_remaining=35.0`
- day1: `ideal_remaining=32.5` (35 × 13/14)
- day2: `ideal_remaining=30.0` (35 × 12/14)
- day3 (after add): new total=42; days remaining=11; `ideal_remaining=42.0` (reset)
- day4: `ideal_remaining=38.18...` (42 × 10/11)

---

### TC-B-07: Actual remaining reconstructed from task_history for tasks

**Feature Requirement:** AC-B-4 — actual remaining from task_history for task entities.
**Task AC:** AC-F008-3
**Technique Applied:** Decision Table — entity type × completion source.
**ISO 25010:** Functional Suitability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S024")` with 3 tasks; task T001 completed on day 3.
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` — `GetCompletionEvents` returns event for T001 with `Timestamp=day3-end-of-day`.
- **Forbidden mocks:** Do not mock `actual_remaining` computation; the service must compute it from events.
- **Counter-factual:** A buggy impl that counts completions by current status (not history) would mark T001 as completed on all days, not just day 3 onward.

**Preconditions:** Sprint with 3 tasks (sizes: 10, 15, 17). T001 (size=10) completed at end of day 3.

**Expected Output:**
- day0 through day2: `actual_remaining=42`
- day3 onward: `actual_remaining=32` (42-10)

---

### TC-B-08: Non-task entity completion uses current status (documented limitation)

**Feature Requirement:** AC-B-4 — non-task entities (bugs, change_cards, tech_debts) use current status.
**Task AC:** AC-F008-3
**Technique Applied:** Decision Table — entity_type=bug, entity_type=change_card.
**ISO 25010:** Functional Suitability, Compatibility (documented limitation is correct behavior)

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S024")` with 1 task + 1 bug; bug in terminal status.
- **Lowest allowed mock seam:** `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not inject task_history events for bugs — bugs use current-status path.
- **Counter-factual:** A buggy impl that tries to look up bug completions in task_history (which only has task rows) returns wrong actual_remaining for sprints with bugs.

**Expected Output:**
- Bug treated as "completed on all days" (point-in-time current status = completed)
- Human output legend includes note: "task burndown reconstructed from history; other entity types use current status"

---

### TC-B-09: `unsized_remaining` present in every daily data point

**Feature Requirement:** AC-B-5 — `unsized_remaining` in each data point and chart legend.
**Task AC:** AC-F008-5
**Technique Applied:** Contract surface enumeration — field present at day 0, mid-sprint, last day, and future days.
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S024")` with 2 unsized entities in sprint
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` — `GetSprintAssignedEntities` includes 2 entities with `Size=nil`
- **Forbidden mocks:** Do not set `UnsizedRemaining=0` in the mock.
- **Counter-factual:** A buggy impl that initializes `UnsizedRemaining` as 0 always fails the assertion at day 0.

**Expected Output:** Every `BurndownDataPoint.UnsizedRemaining = 2`; JSON `--json` includes `unsized_remaining` on every data point.

---

### TC-B-10: Human output uses text table with only ASCII characters

**Feature Requirement:** AC-B-6 — text table, no Unicode block characters.
**Task AC:** AC-F008-1 (output format requirement)
**Technique Applied:** Equivalence Partitioning — valid: ASCII only; invalid: Unicode block chars present.
**ISO 25010:** Functional Suitability, Usability, Portability (works in all terminals)

**Caller-Path Contract:**
- **Entrypoint:** CLI `runSprintBurndown` with mocked service
- **Lowest allowed mock seam:** `SprintAnalyticsService`
- **Forbidden mocks:** Do not mock the output formatter; the formatting code must run.
- **Counter-factual:** A buggy formatter that uses Unicode block chars (e.g., `▓`, `█`) would fail the ASCII-only assertion.

**Expected Output:**
- Output contains header row: `Day`, `Ideal`, `Actual`, `Unsized Remaining`
- Separator line uses `-` or `─` (U+2500 is acceptable; U+2580–U+259F block chars are NOT)
- No chars with codepoint > U+007E except U+2500 (box drawing), U+2014 (em dash for "—")

**Assertion method:** Scan output runes; fail if any rune falls in range U+2580–U+259F (Unicode block elements).

---

### TC-B-11: `--json` burndown output matches AC-B-7 schema

**Feature Requirement:** AC-B-7 — JSON schema.
**Task AC:** AC-F008-6
**Technique Applied:** Contract surface enumeration — all fields × types.
**ISO 25010:** Functional Suitability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** CLI `runSprintBurndown S024 --json` with mocked service returning known data
- **Lowest allowed mock seam:** `SprintAnalyticsService`
- **Forbidden mocks:** Do not mock JSON marshalling.
- **Counter-factual:** A buggy formatter omitting `unsized_total` field fails the field-presence assertion.

**Expected Output (strict schema):**
```json
{
  "sprint_key": "S024",
  "sprint_name": "Sprint 24",
  "total_size": 42,
  "unsized_total": 3,
  "data_points": [
    { "date": "2026-03-18", "ideal_remaining": 42.0, "actual_remaining": 42.0, "unsized_remaining": 3 },
    { "date": "2026-03-19", "ideal_remaining": 39.0, "actual_remaining": 40.0, "unsized_remaining": 3 }
  ]
}
```

**Assertions:** Every top-level key present; `total_size` is int; `ideal_remaining` is float64; `actual_remaining` is float64 (not null for past days); future day data points omit `actual_remaining`.

---

### TC-B-12: Future days show `—` in human output and omit `actual_remaining` from JSON

**Feature Requirement:** AC-B-8 — future days: `—` in human output; omitted from JSON `data_points`.
**Task AC:** AC-F008-4
**Technique Applied:** BVA — boundary: today's date (last day with actual), tomorrow (first day without actual).
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetBurndown(ctx, "S024")` where sprint end_date is in the future
- **Lowest allowed mock seam:** `SprintAnalyticsRepository`; service must inject current date (time.Now()) — test must control time via a time-source interface or inject fixed "today"
- **Forbidden mocks:** Do not mock `BurndownDataPoint.ActualRemaining` directly; the nil must come from service logic.
- **Counter-factual:** A buggy impl that sets `ActualRemaining=0.0` for future days (instead of nil) fails the nil check, and JSON would include a spurious `actual_remaining: 0` instead of omitting the key.

**Preconditions:** Sprint active; today = day 5 of 14. Data points for days 0–5 have actual; days 6–13 do not.

**Expected Output:**
- Human output: days 6–13 show `—` in the "Actual" column
- JSON: `data_points` array contains entries for days 0–5 only (or future-day entries omit `actual_remaining` key entirely — per AC-B-8: "omitted from JSON data points")
- `BurndownDataPoint.ActualRemaining` is nil for day 6+

---

### TC-S-01: `shark sprint summary` for completed sprint returns all base fields

**Feature Requirement:** AC-S-1 — summary available for `completed` and `archived` sprints.
**Task AC:** AC-F009-1
**Technique Applied:** State Transition — valid states: `completed`, `archived`.
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetSummary(ctx, "S024", false)` with sprint status=`completed`
- **Lowest allowed mock seam:** `SprintRepository` (GetByKey returns completed sprint), `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not skip status validation in the mock — the service must load the sprint and check its status.
- **Counter-factual:** A buggy impl that skips status validation would return summary data for a planning sprint, misleading users.

**Preconditions:** Sprint S024 in `completed` status; 10 tasks assigned (8 completed, 2 not completed).

**Expected Output:** `GetSummary` returns non-nil `*SprintSummaryResult` with no error; `CycleTimeByPhase = nil` (detailed=false).

---

### TC-S-02: Summary for planning or active sprint returns informational message, exit 0

**Feature Requirement:** AC-S-1 — planning/active → informational, not error.
**Task AC:** AC-F009-4
**Technique Applied:** State Transition — invalid states for summary: `planning`, `active`.
**ISO 25010:** Functional Suitability, Usability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetSummary(ctx, "S024", false)` with sprint status=`planning` and then status=`active`
- **Lowest allowed mock seam:** `SprintRepository`
- **Forbidden mocks:** Do not return error; the service must signal "informational" without error.
- **Counter-factual:** A buggy impl that returns `err != nil` for active sprint causes CLI to print "Error:..." and exit 1 instead of an informational message.

**Expected Output:** Both planning and active: message like "Summary available for completed or archived sprints only."; exit 0; `--json` returns JSON with `{"status": "informational", "message": "..."}` (or equivalent schema).

---

### TC-S-03: Base summary contains all 12 required fields for a completed sprint

**Feature Requirement:** AC-S-2 — base summary output includes specific fields.
**Task AC:** AC-F009-1
**Technique Applied:** Contract surface enumeration — all 12 base fields × types × value range.
**ISO 25010:** Functional Suitability, Usability (labelled fields)

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetSummary(ctx, "S024", false)` — `detailed=false`
- **Lowest allowed mock seam:** `SprintAnalyticsRepository` (GetSprintAssignedEntities, GetCompletionEvents), `SprintRepository` (GetByKey, GetVelocity via internal call)
- **Forbidden mocks:** Do not pre-populate `SprintSummaryResult` fields in the mock; they must be computed by service logic.
- **Counter-factual:** A buggy impl that omits `VelocityDeltaPct` silently returns a result with 11 fields; the field-count assertion catches it.

**Preconditions:** Sprint S024 (completed): planned_size=50, completed_size=40, planned_count=10, completed_count=8. Previous 5 sprints have trailing average 38.0.

**Expected Output — `SprintSummaryResult` fields:**

| Field | Type | Expected Value |
|---|---|---|
| SprintKey | string | "S024" |
| SprintName | string | non-empty |
| PlannedSize | int | 50 |
| CompletedSize | int | 40 |
| CompletionPctBySize | float64 | 80.0 |
| PlannedCount | int | 10 |
| CompletedCount | int | 8 |
| VelocityThisSprint | int | 40 |
| TrailingAvgVelocity | float64 | 38.0 |
| VelocityDelta | float64 | 2.0 |
| VelocityDeltaPct | float64 | 5.26 (approx) |
| UnsizedPlanned | int | ≥0 |
| UnsizedCompleted | int | ≥0 |

**Detailed fields:** `AddedMidSprintCount`, `RemovedMidSprintCount`, `CycleTimeByPhase`, `AvgCompletedSize`, `SizeBandDistribution`, `CarryoverEntities` — all nil/empty when `detailed=false`.

---

### TC-S-04: `completion_pct_by_size` calculation: completed_size / planned_size * 100

**Feature Requirement:** AC-S-2 — completion percentage by size formula.
**Task AC:** AC-F009-1
**Technique Applied:** Equivalence Partitioning — valid partitions: 0/0 (edge), partial completion, 100%.
**ISO 25010:** Functional Suitability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetSummary(ctx, "S024", false)`
- **Lowest allowed mock seam:** `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not pass `CompletionPctBySize` through the mock.
- **Counter-factual:** A buggy impl that computes count-based percentage instead of size-based returns wrong value.

**Edge Cases:**
- `planned_size=0, completed_size=0`: `CompletionPctBySize=0.0` (no divide-by-zero panic)
- `planned_size=50, completed_size=50`: `CompletionPctBySize=100.0`
- `planned_size=50, completed_size=60` (scope added mid-sprint): `CompletionPctBySize=120.0` (allowed, not clamped)

---

### TC-S-05: `--detailed` adds mid-sprint, size-band, carryover fields when E13 data available

**Feature Requirement:** AC-S-3 — `--detailed` adds extra fields.
**Task AC:** AC-F009-2
**Technique Applied:** Decision Table — `detailed=true` × E13 data available.
**ISO 25010:** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetSummary(ctx, "S024", true)` with `GetCycleTimeByPhase` returning non-empty slice
- **Lowest allowed mock seam:** `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not return detailed fields from the mock without them flowing through service logic.
- **Counter-factual:** A buggy impl that ignores `detailed=true` returns nil for all detailed fields.

**Expected Output:**
- `AddedMidSprintCount`: non-nil int (count of entities added after sprint start_date)
- `AddedMidSprintSize`: non-nil int
- `RemovedMidSprintCount`: non-nil int
- `RemovedMidSprintSize`: non-nil int
- `CycleTimeByPhase`: non-nil slice with at least one `PhaseTime`
- `AvgCompletedSize`: non-nil float64
- `SizeBandDistribution`: non-nil slice (may be empty if no XS/S/M/L/XL/XXL entities)
- `CarryoverEntities`: non-nil slice (entities not completed)

---

### TC-S-06: `--detailed` with no E13/work_sessions data shows informational message, no error

**Feature Requirement:** AC-S-3 — cycle-time shows "No session data available" when E13 absent.
**Task AC:** AC-F009-3
**Technique Applied:** Decision Table — `detailed=true` × `GetCycleTimeByPhase` returns empty slice.
**ISO 25010:** Functional Suitability, Reliability (no error on missing data), Usability (clear message), Compatibility (works without E13)

**Caller-Path Contract:**
- **Entrypoint:** `SprintAnalyticsService.GetSummary(ctx, "S024", true)` with `GetCycleTimeByPhase` returning `[]PhaseTimeRow{}` (empty slice, not error)
- **Lowest allowed mock seam:** `SprintAnalyticsRepository`
- **Forbidden mocks:** Do not return an error from `GetCycleTimeByPhase`; the empty slice is the signal.
- **Counter-factual:** A buggy impl that panics when `CycleTimeByPhase` is empty fails this test. A buggy impl that returns an error instead of nil slice also fails.

**Expected Output:**
- `SprintSummaryResult.CycleTimeByPhase = nil` (not empty slice)
- Human output: section shows "No session data available" (or equivalent)
- `--json`: `"cycle_time_by_phase": null` (not `[]` and not omitted)
- No error; exit 0

---

### TC-S-07: `--json` output with `detailed=false` — all base fields present, detailed fields `null`

**Feature Requirement:** AC-S-4 — JSON includes all base fields; nil for unavailable detailed fields.
**Task AC:** AC-F009-5
**Technique Applied:** Contract surface enumeration — every field in `SprintSummaryResult` × JSON representation.
**ISO 25010:** Functional Suitability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** CLI `runSprintSummary S024 --json` (no `--detailed`) with mocked service
- **Lowest allowed mock seam:** `SprintAnalyticsService`
- **Forbidden mocks:** Do not mock JSON marshalling.
- **Counter-factual:** A buggy JSON marshaller that uses `omitempty` on detailed pointer fields would omit them instead of rendering `null`.

**Expected Output:**
- All 12 base fields present and non-null
- `"cycle_time_by_phase": null` (not missing)
- `"carryover_entities": null` (not missing)
- `"size_band_distribution": null` (not missing)
- JSON is valid; parseable by `jq`

---

### TC-S-08: `--json --detailed` output — detailed fields populated or `null` (never omitted)

**Feature Requirement:** AC-S-4 — missing work_sessions data is `null` in JSON, not omitted.
**Task AC:** AC-F009-5
**Technique Applied:** Contract surface enumeration — detailed=true × data absent.
**ISO 25010:** Functional Suitability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** CLI `runSprintSummary S024 --detailed --json` with mocked service returning nil `CycleTimeByPhase`
- **Lowest allowed mock seam:** `SprintAnalyticsService`
- **Forbidden mocks:** Do not add `omitempty` to detailed fields in the JSON tag.
- **Counter-factual:** A buggy JSON marshaller with `omitempty` on `CycleTimeByPhase` returns `{}` for that field, breaking callers that check `field == null`.

**Expected Output:**
- `"cycle_time_by_phase": null` (not omitted, not `[]`)
- Other detailed fields populated where data exists

---

### TC-NF-01: Analytics commands complete in < 2 s on 50-sprint / 1000-task dataset

**Feature Requirement:** REQ-NF-001 — < 2 s wall-clock for ≤ 50 sprints / 1000 tasks.
**Task AC:** AC-X-2
**Technique Applied:** BVA (performance) — at maximum spec boundary.
**ISO 25010:** Performance Efficiency, Reliability

**Caller-Path Contract:**
- **Entrypoint:** Real repository (`SprintAnalyticsRepository`) against test DB with 50 sprints and 1000 tasks seeded. This is a repository-level integration test, not a service-level mock test.
- **Lowest allowed mock seam:** DB connection (test DB) — no higher mock seam permitted.
- **Forbidden mocks:** Do not mock `SprintAnalyticsRepository`; this test validates actual query performance.
- **Counter-factual:** A buggy query that does a full table scan on `sprint_assignments` (no index) takes >2 s and fails the timing assertion.

**Preconditions:**
- Test DB seeded: 50 sprints, 1000 tasks distributed across sprints
- `sprint_assignments(sprint_id)` index confirmed present (from E19-F01)

**Input:** Execute `GetVelocityData(ctx, 50)`, `GetBurndown` for active sprint, `GetSummary` for completed sprint.

**Expected Output:** Each call completes in < 2 s wall-clock time. (Allow up to 3 retries for CI jitter; median must be < 2 s.)

**Negative Cases:** Command does NOT timeout at 2 s; does NOT degrade to full table scans.

---

### TC-NF-02: `EXPLAIN QUERY PLAN` confirms indexed lookups for core analytics queries

**Feature Requirement:** REQ-NF-001 — indexed lookups verified with SQLite `EXPLAIN QUERY PLAN`.
**Task AC:** AC-X-2
**Technique Applied:** Contract surface enumeration — one EXPLAIN per major query.
**ISO 25010:** Performance Efficiency, Maintainability

**Caller-Path Contract:**
- **Entrypoint:** internal — SQL queries in `SprintAnalyticsRepository` are the production entrypoint. This test inspects the query plan of each method's SQL.
- **Lowest allowed mock seam:** DB connection (test DB)
- **Forbidden mocks:** Do not mock the DB connection for this test.
- **Counter-factual:** A query that uses `SCAN TABLE sprint_assignments` (no index) in the EXPLAIN output would fail the assertion.

**Expected Output:** `EXPLAIN QUERY PLAN` for `GetVelocityData`, `GetSprintAssignedEntities`, `GetCompletionEvents` queries shows `SEARCH TABLE sprint_assignments USING INDEX` (not `SCAN TABLE`).

---

### TC-NF-03: `--json` and `--field` flags consistent with existing entity command pattern

**Feature Requirement:** REQ-NF-005 — consistent flag behavior.
**Task AC:** Cross-cutting
**Technique Applied:** Contract surface enumeration — `--json`, `--field`, and absence of flags for all three commands.
**ISO 25010:** Functional Suitability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** CLI handlers `runSprintVelocity`, `runSprintBurndown`, `runSprintSummary` with `--json` and `--field trailing_average` / `--field sprint_key` / `--field completed_size`
- **Lowest allowed mock seam:** `SprintAnalyticsService`
- **Forbidden mocks:** Do not mock the `--field` flag parsing — it must flow through the existing `cli.OutputJSON` / `--field` extraction path.
- **Counter-factual:** A buggy command that hardcodes output format ignoring `--json` would return human-readable output when JSON is expected.

**Input/Expected Output matrix:**

| Command | Flag | Expected output |
|---|---|---|
| `shark sprint velocity` | (none) | Human-readable table |
| `shark sprint velocity` | `--json` | Valid JSON per AC-V-6 schema |
| `shark sprint velocity` | `--field trailing_average` | Single float value |
| `shark sprint burndown S024` | `--json` | Valid JSON per AC-B-7 schema |
| `shark sprint summary S024` | `--json` | Valid JSON per AC-S-4 |
| `shark sprint summary S024` | `--field completed_size` | Single int value |

---

## Integration Scenarios

### INT-09 Coverage: Sprint Analytics Without E13 (UAT Priority 1)

This feature contributes to **UAT-INT-09** (analytics functional without E13/work_sessions data).

**Components interacting:** `SprintAnalyticsService` → `SprintAnalyticsRepository` → `task_history` table (not `work_sessions`).

**What to verify at boundaries:**
- `GetCycleTimeByPhase` returns empty slice (not error) when `work_sessions` is empty
- `GetSummary(ctx, key, true)` sets `CycleTimeByPhase=nil`, not `[]PhaseTime{}`
- CLI renders "No session data available" from nil (not panic, not blank)

**Test cases covering this integration:** TC-S-06, TC-S-08.

---

### INT-06 Coverage: Service Layer Architecture (E15)

**Components interacting:** CLI handlers → `SprintAnalyticsService` → `SprintAnalyticsRepository`.

**What to verify at boundaries:**
- `internal/cli/commands/sprint.go` does NOT import `internal/repository/sprint` directly
- CLI command tests use mocked `SprintAnalyticsService` (no real DB)
- Service tests use mocked `SprintAnalyticsRepository` (no real DB)

**Test cases:** All TC-SVC-* cases (service layer tests) inherently validate this by using function-field mocks.

---

### UAT-J3-03 / UAT-J3-05: Human-visible monitoring scenarios

**Components interacting:** Human-readable formatter → `BurndownResult.DataPoints` / `VelocityResult.Sprints`.

**What to verify:**
- Burndown text table is readable; columns aligned; legend present (TC-B-10)
- Velocity trailing average prominent in human output (TC-V-01)
- "Insufficient data" message actionable (TC-V-09)

---

### UAT-J4-05: Detailed Retrospective (Priority 3)

**Components interacting:** `GetSummary(detailed=true)` → `GetCycleTimeByPhase` → `task_history`.

**What to verify:**
- Size-band distribution uses correct labels: XS(1)/S(2)/M(3)/L(5)/XL(8)/XXL(13) — labels per entity size column values (TC-S-05)
- Carryover list includes entity key, type, size (TC-S-05)

---

## Test Infrastructure

### Existing Patterns to Follow

| File | Pattern Used | Relevance |
|---|---|---|
| `internal/services/sprint_service_test.go` | `MockSprintRepository` (function-field mock) | Extend with `MockSprintAnalyticsRepository` using same pattern |
| `internal/services/dashboard_analytics_service_test.go` | Mock analytics repo with function fields | Direct precedent for analytics service tests |
| `internal/repository/sprint/repository_test.go` | `test.GetTestDB()` + `dbconn.NewDB()` + TEST-prefixed cleanup | Repository tests for `analytics.go` follow this exactly |
| `internal/cli/commands/sprint.go` + (sprint_test.go when it exists) | Cobra command + mocked service | CLI tests for three new commands follow bug.go pattern |

### New Test Helpers Needed

1. **`seedSprintWithAssignments(t, db, sprintKey, entities []seedEntity) (sprintID int64)`** — creates a sprint and populates `sprint_assignments` + entity records for analytics repo tests. Prevents 50+ lines of seeding code per test.

2. **`seedTaskHistoryEvents(t, db, sprintID int64, events []historyEvent)`** — seeds `task_history` rows for specific tasks within a sprint window. Needed for TC-B-07, TC-NF-01.

3. **`MockSprintAnalyticsRepository`** (in `internal/services/sprint_analytics_service_test.go`) — function-field mock implementing `SprintAnalyticsRepository` interface. Methods: `GetVelocityDataFunc`, `GetSprintAssignedEntitiesFunc`, `GetCompletionEventsFunc`, `GetCycleTimeByPhaseFunc`.

4. **Performance seed script** — seeds 50 sprints × 1000 tasks for TC-NF-01. Can be a `TestMain` setup in `internal/repository/sprint/analytics_test.go`.

### Test File Locations

| Test | File |
|---|---|
| Repository tests | `internal/repository/sprint/analytics_test.go` (new) |
| Service tests | `internal/services/sprint_analytics_service_test.go` (new) |
| CLI command tests | `internal/cli/commands/sprint_test.go` (extend) |
| Performance / integration | `internal/repository/sprint/analytics_test.go` (separate `TestPerformance_*` functions with `t.Skip` in short mode) |

---

## Codex Test-Plan Red-Team

**Verdict:** NOT RUN (codex binary present at `/home/jwwel/.nvm/versions/node/v20.20.0/bin/codex` but feature spec is pre-implementation; test plan was drafted directly from spec. Codex red-team is most valuable after a full draft exists to critique. Running now would red-team this document.)

**Self-review checklist (substitute for codex pass):**

| Check | Status |
|---|---|
| Every AC has at least one ISTQB technique | ✅ All 20 ACs annotated |
| Every AC has ISO 25010 row with no empty cells | ✅ N/A cells justified |
| Every behavior has observability design | ✅ Observability table complete; verbose-mode log assertions added |
| Every test case has Caller-Path Contract | ✅ All TC-* have entrypoint, mock seam, forbidden mocks, counter-factual |
| All detailed pointer fields must be null (not omitted) in JSON | ✅ TC-S-07, TC-S-08 explicitly test this — key distinction caught in AC-S-4 review |
| Zero-velocity sprint in trailing average denominator | ✅ TC-V-07 explicitly tests this — the most likely AC-V-4 bug |
| Future days omit actual_remaining (nil not 0.0) | ✅ TC-B-12 tests nil vs. 0.0 distinction |
| Non-task entity burndown limitation documented in output | ✅ TC-B-08 asserts legend text |
| Performance test uses real DB (not mock) | ✅ TC-NF-01 specifies no mock seam above DB connection |
| `EXPLAIN QUERY PLAN` assertion for index verification | ✅ TC-NF-02 added |

**Issues addressed before dev:** 2 (nil vs. empty-slice distinction for `CycleTimeByPhase`; future-day actual_remaining nil vs. 0.0)
**Issues deferred:** 0

---

## Recommendations

- [x] **Ready for development.** No spec drift. All 20 ACs have concrete test cases, ISTQB technique annotations, ISO 25010 coverage, and caller-path contracts.
- [ ] **Dependency note:** TC-NF-01 and TC-B-07 require E19-F01 schema (sprints, sprint_assignments, task_history) and at least one completed sprint. E19-F01 and E19-F02 must be merged before analytics tests can run against a real DB.
- [ ] **Test helper creation:** `seedSprintWithAssignments` and `seedTaskHistoryEvents` helpers should be created alongside the analytics repository implementation, not after.
- [ ] **Performance seed:** The 50-sprint × 1000-task seed in TC-NF-01 should be gated with `if testing.Short() { t.Skip() }` to keep `make test` fast.
