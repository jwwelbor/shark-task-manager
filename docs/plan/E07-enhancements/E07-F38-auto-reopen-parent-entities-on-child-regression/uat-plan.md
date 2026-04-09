---
epic_key: E26
title: UAT Plan — Auto-reopen parent entities on child regression
last_updated: 2026-04-06
status: design
---

# E26 UAT Plan

This document defines the User Acceptance Test plan for E26. It maps directly to the success criteria (SC1–SC10) and high-level UAT scenarios (UAT-1 through UAT-8) in `epic.md`, and to the architecture decisions in `architecture.md`.

UAT for this epic is **mostly developer-facing** (the "users" are developers and AI agents who consume the shark workflow). The acceptance test descriptions below describe **what to verify**, not how to implement the test code — implementation guidance is left to the feature decomposition phase.

---

## 1. UAT Scenarios

Each scenario below is independently runnable, has explicit Given/When/Then steps, and maps to one or more success criteria.

### UAT-S1: UAT rejection cycle (the primary scenario) [SC1, SC3, SC4, SC7]

**Description.** A QA tester rejects a completed feature. The developer regresses the failing task. All terminal ancestors must reopen automatically to their correct historical statuses, and the cascade must be visible in `shark status history`.

**Given:**
- Project initialized with the advanced workflow profile.
- Epic `E07` exists in status `completed`. `shark status history E07` shows it was previously in `in_development` before being marked complete.
- Feature `E07-F01` exists in status `completed`. `shark status history E07-F01` shows it was previously in `in_qa` before being marked complete.
- Task `E07-F01-003` exists in status `completed`.

**When:**
- A developer runs `shark status set E07-F01-003 in_development --reason="QA rejected step 3"`.

**Then:**
- `shark get E07-F01-003 --field status` returns `in_development`.
- `shark get E07-F01 --field status` returns `in_qa` (the feature's prior non-terminal status).
- `shark get E07 --field status` returns `in_development` (the epic's prior non-terminal status).
- `shark status history E07-F01 --json` contains a new row with `notes` starting with `auto_reopen:` and referencing `E07-F01-003`.
- `shark status history E07 --json` contains a new row with `notes` starting with `auto_reopen:` and referencing `E07-F01-003`.
- The two new history rows appear in the database **either both or neither** (atomicity check via SQL: count rows added in the cascade transaction window).

**Acceptance:** All five "Then" assertions pass. No manual `shark status set` is needed for the parents.

---

### UAT-S2: New task under a completed feature reopens the chain [SC2, SC4, SC9]

**Description.** Adding a follow-up task to a completed feature must reopen both the feature and the epic, using the same unified cascade path (not the legacy aggregation-status logic).

**Given:**
- Epic `E07` is `completed`, history has prior non-terminal entries.
- Feature `E07-F01` is `completed`, history has prior non-terminal entries.

**When:**
- A developer runs `shark task create E07 F01 "Follow-up fix"`.

**Then:**
- The new task is created in the workflow's default initial status.
- `shark get E07-F01 --field status` returns the feature's prior non-terminal status from history.
- `shark get E07 --field status` returns the epic's prior non-terminal status from history.
- The existing test suite for "new task reopens feature" (in `task_service_test.go` lines 2173–2617) is still green against the unified implementation.
- No two history rows exist for the same cascade (the legacy `maybeReopen*` is not running in parallel with the new cascade).

**Acceptance:** Both ancestors reopen, both reach their history-resolved targets (not aggregation-status), and the existing CreateTask reopen test suite still passes. SC9 is the critical regression check.

---

### UAT-S3: Idempotency under concurrent regressions [SC8]

**Description.** Two tasks regress in rapid succession under the same parent. The cascade must fire exactly once for each ancestor, not twice. The second regression must observe the parents already reopened and write nothing.

**Given:**
- Feature `E07-F01` is `completed`. Epic `E07` is `completed`.
- Tasks `E07-F01-001` and `E07-F01-002` are both `completed`.

**When:**
- The developer regresses `E07-F01-001` first: `shark status set E07-F01-001 in_development --reason="..."`.
- Immediately after (in the next CLI invocation), the developer regresses `E07-F01-002`: `shark status set E07-F01-002 in_development --reason="..."`.

**Then:**
- After the first regression: `E07-F01` and `E07` are both reopened, each with one new `auto_reopen:` history row.
- After the second regression: `E07-F01` and `E07` remain in their reopened states. **No additional history rows** are written for the parents by the second regression. The second regression's child task is still updated normally.
- A query `SELECT COUNT(*) FROM entity_history WHERE entity_id = E07-F01.id AND notes LIKE 'auto_reopen:%'` returns exactly `1`.
- A query `SELECT COUNT(*) FROM entity_history WHERE entity_id = E07.id AND notes LIKE 'auto_reopen:%'` returns exactly `1`.

**Acceptance:** Cascade is idempotent. No duplicate history rows. No status churn (e.g., the parent does not flicker between two non-terminal statuses).

---

### UAT-S4: Workflow profile compatibility [SC5]

**Description.** The cascade must work with no code changes across the basic workflow profile, the advanced workflow profile, and a custom profile with non-standard status names.

**Sub-scenario A — Basic profile:**

**Given:**
- Project initialized with `shark init --workflow=basic`.
- Feature `E07-F01` is `completed`. History shows it was previously in `in_progress`.

**When:**
- A child task regresses out of `completed`.

**Then:**
- Feature reopens to `in_progress` (the basic profile's prior non-terminal status from history).

**Sub-scenario B — Advanced profile:**

**Given:**
- Project initialized with `shark init --workflow=advanced`.
- Feature `E07-F01` is `completed`. History shows it was previously in `in_qa`.

**When:**
- A child task regresses out of `completed`.

**Then:**
- Feature reopens to `in_qa`.

**Sub-scenario C — Custom profile:**

**Given:**
- Project uses a custom workflow profile with statuses `kickoff`, `building`, `validating`, `shipped` (where `shipped` is configured as the `_complete_` terminal). `building` is the prior non-terminal status in history.

**When:**
- A child regresses out of `shipped`.

**Then:**
- Feature reopens to `building`.
- No code path references the strings `completed`, `in_qa`, `in_development`, etc. (verified by code review during implementation).

**Acceptance:** All three sub-scenarios pass. No status names hardcoded anywhere in the cascade implementation.

---

### UAT-S5: Bugs and change-cards never trigger cascade [SC6]

**Description.** Bugs and change-cards are standalone entities. Even when linked to a feature or epic, regressing them must not reopen any parent.

**Sub-scenario A — Bug regression:**

**Given:**
- Feature `E07-F01` is `completed`. Epic `E07` is `completed`.
- Bug `B042` is linked to `E07-F01`. Bug `B042` is in status `closed`.

**When:**
- A developer runs `shark status set B042 open`.

**Then:**
- Bug `B042` is now `open`.
- Feature `E07-F01` is **still** `completed`.
- Epic `E07` is **still** `completed`.
- No `auto_reopen:` history rows were written for the feature or epic.

**Sub-scenario B — Change-card regression:**

**Given:**
- Feature `E07-F01` is `completed`. Epic `E07` is `completed`.
- Change-card `CC-001` is linked to `E07-F01`.

**When:**
- A developer regresses `CC-001` from `approved` to `proposed`.

**Then:**
- Change-card `CC-001` is now `proposed`.
- Feature and epic are unchanged.
- No `auto_reopen:` history rows for the parents.

**Acceptance:** Both sub-scenarios show zero side effects on parents. SC6 is satisfied structurally (no cascade hook in `BugService` or `ChangeCardService`).

---

### UAT-S6: Fallback when no prior non-terminal history exists [Architecture ADR-002, UAT-6]

**Description.** When an ancestor has no usable prior non-terminal history (e.g., it was bulk-imported or its history table is empty for that entity), the cascade must fall back gracefully. Architecture defines a three-step fallback chain: history → aggregation → initial.

**Sub-scenario A — Aggregation fallback:**

**Given:**
- Feature `E07-F01` is `completed`. Its `entity_history` contains exactly one row: the transition into `completed`. No prior non-terminal entries.
- The active workflow profile defines `_aggregation_` statuses.

**When:**
- A child task regresses.

**Then:**
- Feature reopens to the first aggregation status (`_aggregation_[0]`).
- The history row's `notes` includes a `[fallback: aggregation]` marker (per ADR-005).

**Sub-scenario B — Initial-status fallback:**

**Given:**
- Same as above, but the workflow profile defines no aggregation statuses.

**When:**
- A child task regresses.

**Then:**
- Feature reopens to `workflowSvc.GetInitialStatusString()`.
- The history row's `notes` includes a `[fallback: initial]` marker.

**Acceptance:** Both sub-scenarios reach a valid non-terminal status without an error. The fallback path is observable in the audit trail.

---

### UAT-S7: Audit trail is distinguishable from manual transitions [SC7]

**Description.** When a developer or AI agent inspects history, auto-reopens must be visually and structurally distinct from manual transitions.

**Given:**
- Any cascade has just fired (from any UAT scenario above).

**When:**
- A developer runs `shark status history E07-F01` (table format).
- And separately: `shark status history E07-F01 --json`.

**Then (table format):**
- The auto-reopen row is rendered with a distinct visual marker — for example, an `[auto]` prefix or a distinct color column. The exact rendering is a presentation-layer decision but it must be **immediately obvious** that this row is not from a manual `shark status set`.
- The triggering child key is visible in the row (either inline in the reason column or in a separate "triggered by" column).

**Then (JSON output):**
- The row's `notes` field starts with the literal string `auto_reopen:`.
- The row's `notes` field contains the triggering child key (e.g., `E07-F01-003`).
- The row's `changed_by` field is `system` (not a user agent ID).
- A consumer can filter for auto-reopens using `jq '.[] | select(.notes | startswith("auto_reopen:"))'`.

**Acceptance:** Both table and JSON output meet the distinguishability requirements. A second test using `grep auto_reopen` against `shark status history --json` returns the expected count of cascade rows.

---

### UAT-S8: Dashboard rollups reflect reopened state immediately [SC1, SC4]

**Description.** After a cascade, the project dashboard and epic status views must reflect the reopened state. This is the user-visible payoff of the entire epic — the dashboard becomes trustworthy again.

**Given:**
- Any cascade has just fired (use the UAT-S1 setup as the canonical case).

**When:**
- A developer runs `shark status` (project dashboard).
- And: `shark status E07` (epic-level rollup).
- And: `shark feature get E07-F01` (feature detail with progress breakdown).

**Then:**
- The project dashboard shows `E07` in its reopened (non-terminal) status, not `completed`.
- The epic status rollup shows `E07-F01` in its reopened (non-terminal) status, not `completed`.
- The feature progress breakdown reflects the regressed child task as in-flight (not in `completed` count).
- Weighted progress percentage drops accordingly (the regressed task no longer contributes 100% weight).

**Acceptance:** Three CLI commands all show consistent, reopened state. No staleness, no caching artifacts.

---

## 2. Cross-Feature Integration Scenarios

These scenarios stress the interaction between the cascade and other shark subsystems. They are integration tests that span multiple services.

### INT-1: Cascade interacts correctly with `recalculateFeatureProgress`

**Description.** When a task regresses, two post-hooks fire in `TaskService.TransitionStatus`: `recalculateFeatureProgress` and the new cascade. They must not conflict.

**Verify:**
- After cascade fires, the feature progress recalculation reflects both the task's new status **and** the feature's reopened status.
- The feature's `progress_weight` rollup is consistent with the new task statuses (the regressed task no longer counts toward `completed`).
- No race condition between the two post-hooks (they serialize naturally because `TransitionStatus` is single-threaded per call).

---

### INT-2: Cascade interacts correctly with auto-unblock dependents

**Description.** `TaskService.TransitionStatus` already auto-unblocks dependent tasks. When a task regresses out of `completed`, dependents that were unblocked by its completion should NOT be re-blocked by the cascade — that is a separate concern. But if dependents existed and were `blocked`, the regression of the parent task may reblock them via existing logic. The cascade must coexist cleanly.

**Verify:**
- A regression that triggers both auto-unblock side effects and a cascade produces a coherent final state.
- No `dev-artifacts` race conditions or database deadlocks.

---

### INT-3: Cascade interacts correctly with orchestrator action resolution

**Description.** After a cascade reopens an ancestor, the next time an orchestrator queries that ancestor (via `shark epic get E07 --json` or similar), the `orchestrator_action` field must reflect the reopened status — not the stale terminal status.

**Verify:**
- `shark epic get E07 --json` after a cascade returns `orchestrator_action` matching the reopened status's configured action.
- An AI agent (e.g., DevAgent) querying for next work routes correctly based on the new status.

---

### INT-4: Cascade fires correctly through the HTTP API

**Description.** The cascade is a service-layer feature, so it must work identically when triggered via the HTTP API as when triggered via the CLI.

**Verify:**
- A `PATCH /api/v1/tasks/E07-F01-003/status` with `{"status": "in_development"}` produces the same parent-reopen behavior as the CLI command.
- The HTTP response for the child status update returns 200 even if the cascade emits a `slog.Warn` (cascade failure must not surface as an HTTP 500).
- Subsequent HTTP `GET /api/v1/features/E07-F01` reflects the reopened state.

---

### INT-5: Cascade fires correctly under the `shark task next-status` advance command

**Description.** `shark task next-status` is the orchestrator-friendly transition command. If it advances backward (e.g., from `completed` back to `in_qa` via a workflow-defined regression edge), the cascade must fire.

**Verify:**
- `shark task next-status E07-F01-003` from `completed` triggers the cascade if the next workflow edge is non-terminal.
- The cascade fires regardless of which CLI verb (`set`, `advance`, `next-status`, `reopen`, `set-status`) initiated the regression.

---

### INT-6: Existing CreateTask reopen test suite stays green [SC9]

**Description.** This is the strongest regression check. The pre-E26 test suite for "new task reopens feature" (in `task_service_test.go` lines 2173–2617) was written against the legacy `maybeReopenParentFeature` helper. After Phase 5 of the rollout (which refactors that helper to use the unified cascade), all those tests must still pass without modification.

**Verify:**
- `go test ./internal/services/ -run TestTaskService.*Reopen` is green.
- No test had to be modified to accommodate the new history-resolved target. (Tests that explicitly check for `aggregation_status[0]` may need updating if the test fixture now produces a different history state — those updates are part of Phase 5's deliverables and must be reviewed for correctness.)

---

## 3. Performance Considerations

### PERF-1: Cascade overhead within budget

**Description.** Per epic constraint, cascade overhead must be ≤50ms per transition on a typical project.

**Verify:**
- Benchmark `TaskService.TransitionStatus` with cascade enabled vs. disabled (toggle via the optional dependency injection).
- On a database with ~500 tasks and ~50 history rows per entity, cascade adds <30ms in the 95th percentile.
- Worst case (cold cache, full feature + epic walk, history lookup on both ancestors) stays under 50ms.

### PERF-2: History lookup query uses index

**Description.** The new `GetLastNonTerminalStatus` query must use an index on `entity_history(entity_type, entity_id, changed_at DESC)`.

**Verify:**
- `EXPLAIN QUERY PLAN` on the lookup query shows an index seek, not a full table scan.
- If the index does not exist (per architecture section 6, this is the one possible schema change), it is added via the standard migration protocol with `CurrentSchemaVersion` bump.

### PERF-3: Idempotent skip path is fast

**Description.** When the parent is already non-terminal (idempotent case), the cascade must be cheap.

**Verify:**
- Cascade in idempotent mode (parent already non-terminal) completes in <10ms.
- No unnecessary history queries — the terminal-status check should short-circuit before `GetLastNonTerminalStatus` is called.

---

## 4. Security Considerations

This epic is internal behavior — no new attack surface is introduced. Standard checks apply:

### SEC-1: No new SQL injection surface

**Verify:**
- The new `GetLastNonTerminalStatus` query uses parameterized placeholders for `entity_type`, `entity_id`, and the terminal-status set (the `NOT IN (...)` clause is built with `?` placeholders, not string interpolation).
- No user-supplied input reaches the query directly.

### SEC-2: No privilege escalation via auto-reopen

**Verify:**
- Auto-reopen transitions are recorded as `changed_by = "system"`, not as the user who triggered the regression. This is a deliberate audit-trail choice — the cascade is system behavior, not user behavior.
- No CLI flag allows a user to forge the `changed_by` field on cascade rows.

### SEC-3: Bug/change-card linkage cannot leak into cascade

**Verify:**
- A bug linked to a feature via `shark related-docs` or any other linkage mechanism does not cause the cascade to fire when the bug regresses. Confirmed by UAT-S5.
- A malicious user cannot construct a scenario where regressing a bug indirectly reopens an unrelated epic.

---

## 5. Mapping: Success Criteria → UAT Scenarios

| Success Criterion | Covered By |
|---|---|
| SC1 — Backward transition reopens terminal ancestors | UAT-S1, UAT-S8 |
| SC2 — New child under terminal parent reopens ancestors | UAT-S2 |
| SC3 — Each ancestor returns to its own previous non-terminal status | UAT-S1, UAT-S2, UAT-S4 |
| SC4 — Cascade walks full chain in one operation | UAT-S1, UAT-S8 |
| SC5 — Works on basic, advanced, and custom profiles | UAT-S4 |
| SC6 — Bugs/change-cards never trigger cascade | UAT-S5 |
| SC7 — Auto-reopen recorded with distinguishable reason | UAT-S1, UAT-S7 |
| SC8 — Reopening already non-terminal is no-op | UAT-S3 |
| SC9 — Existing reopen tests still pass | UAT-S2, INT-6 |
| SC10 — Manual parent-cleanup interventions drop to zero | Dogfood metric, tracked post-rollout (no synthetic UAT) |

Every success criterion except SC10 has a synthetic acceptance test. SC10 is a usage metric tracked over time and is observed during the post-release dogfood window.

---

## 6. Mapping: UAT Scenarios → Epic UAT Scenarios

The epic PRD listed 8 high-level UAT scenarios (UAT-1 through UAT-8). This plan refines and renames them as UAT-S1 through UAT-S8, with the following mapping:

| Epic UAT (PRD section 6) | This Plan |
|---|---|
| UAT-1 (UAT rejection cycle) | UAT-S1 |
| UAT-2 (New task under completed feature) | UAT-S2 |
| UAT-3 (Idempotency under concurrent regressions) | UAT-S3 |
| UAT-4 (Workflow profile compatibility) | UAT-S4 |
| UAT-5 (Bugs and change-cards do not cascade) | UAT-S5 |
| UAT-6 (Fallback when no prior non-terminal history) | UAT-S6 |
| UAT-7 (Audit trail is distinguishable) | UAT-S7 |
| UAT-8 (Dashboard rollups reflect reopened state) | UAT-S8 |

There are no orphaned PRD UAT scenarios. All 8 are covered.

---

## 7. UAT Execution Notes

**Execution context.** This plan is executed by the QA-equivalent role at the **end** of the epic (after all features are decomposed, implemented, and tech-reviewed). For E26, this means after the cascade implementation feature ships and before the epic is approved for `completed`.

**Manual vs automated.** UAT-S1 through UAT-S8 are amenable to **scripted** execution — a single test fixture can set up the project state, run the trigger CLI command, and assert the post-conditions via `shark get --json` and `shark status history --json`. INT-1 through INT-6 are integration tests that run as part of `make test`. PERF-1 through PERF-3 require benchmark setup and are run manually before release.

**Pass/fail definition.** The epic exits the in_uat phase only when:
1. All UAT-S scenarios pass without manual intervention (no developer touches a parent status during the test).
2. All INT scenarios pass.
3. PERF-1 measurement is within budget.
4. SC10 baseline is captured (count of manual `shark status set` commands targeting features/epics in the prior 30 days, used as a comparison point post-release).

**Failure escalation.** Any UAT-S failure blocks the epic's progression. A PERF or INT failure is escalated to architecture review and may require revising ADR-003 (the two-phase commit decision) or adding the missing index per ADR-002 risk.
