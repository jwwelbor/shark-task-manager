---
epic_key: E26
title: Architecture — Auto-reopen parent entities on child regression
last_updated: 2026-04-06
status: design
---

# E26 Architecture: Auto-reopen Parent Entities on Child Regression

This document defines the technical design for the auto-reopen cascade. It builds directly on the research findings in `F01-feature-context.md` and adheres to the existing service-layer / repository-layer separation defined in `.claude/rules/architecture.md` and `.claude/rules/services/service-design.md`.

---

## 1. Component Overview

### 1.1 What changes

| Layer | Component | Change |
|-------|-----------|--------|
| Repository | `internal/repository/entityhistory/repository.go` | Add `GetLastNonTerminalStatus(ctx, entityType, entityID, terminalStatuses)` query method. |
| Repository | `internal/repository/feature_repository.go` | Add `UpdateStatusTx(ctx, tx, id, status, agent, notes)` for in-transaction updates. |
| Repository | `internal/repository/epic_repository.go` | Add `UpdateStatusTx(ctx, tx, id, status, agent, notes)` for in-transaction updates. |
| Repository | `internal/repository/task_repository.go` | Confirm `StatusUpdateRawTx` (or equivalent in-tx variant) exists; add if missing. |
| Service | `internal/services/cascade_reopen.go` (new) | Hosts the unified `cascadeParentReopens` helper, the `ParentReopenHistoryQuerier` interface, and the `ReopenTarget` resolver. |
| Service | `internal/services/task_service.go` | (a) Add post-hook in `TransitionStatus` calling `cascadeParentReopens` when prior status was terminal and new status is non-terminal. (b) Refactor `maybeReopenParentFeature` to call the unified helper. |
| Service | `internal/services/feature_service.go` | (a) Add post-hook in `TransitionStatus` calling cascade for the parent epic only. (b) Refactor `maybeReopenParentEpic` to call the unified helper. |
| Service | `internal/services/task_service.go`, `feature_service.go` | Inject the new `ParentReopenHistoryQuerier` and a `*repository.DB` reference (for owning the cascade transaction). Both are optional; cascade degrades gracefully when nil. |
| CLI wiring | `internal/cli/services_global.go` | Wire the new dependencies into `GetTaskService()` and `GetFeatureService()`. |
| HTTP wiring | `cmd/server/services.go` | Wire the same dependencies into `WireServices()`. |

### 1.2 What stays unchanged

- `EntityService.TransitionStatus` and the underlying transition flow (validation, normalization, history recording, rejection notes, action resolution). The cascade is layered **on top** of an already-completed child transition as a post-hook, not embedded in `EntityService`.
- `workflow.Service` API. We use the existing `IsTerminalStatus`, `ForLevel`, `GetInitialStatusString`, and `GetAggregationStatuses` without modification.
- `entity_history` table schema. The existing `notes` column carries the auto-reopen reason string. **No migration, no `CurrentSchemaVersion` bump.**
- `recordEntityHistory` helper. Used as-is for writing the auto-reopen rows.
- All existing CLI commands, JSON output schemas, and exit codes. The cascade is observable only through additional `entity_history` rows and updated parent statuses.
- Bug and change-card services. They have no cascade hook, satisfying the "bugs/change-cards never cascade" requirement by construction.
- `EpicService` (no parent above epic).

### 1.3 New code surface (estimated)

| File | Purpose | Approx. LOC |
|------|---------|-------------|
| `internal/services/cascade_reopen.go` | Unified cascade helper, interface, target resolver | 180 |
| `internal/services/cascade_reopen_test.go` | Unit tests with mocked dependencies | 350 |
| `internal/repository/entityhistory/repository.go` (additions) | `GetLastNonTerminalStatus` | 35 |
| `internal/repository/entityhistory/repository_test.go` (additions) | Repository test for new method | 80 |
| `internal/repository/feature_repository.go` (additions) | `UpdateStatusTx` | 25 |
| `internal/repository/epic_repository.go` (additions) | `UpdateStatusTx` | 25 |
| `internal/services/task_service.go` (modifications) | Post-hook + refactor `maybeReopenParentFeature` | net +30 |
| `internal/services/feature_service.go` (modifications) | Post-hook + refactor `maybeReopenParentEpic` | net +25 |

Total estimated net new code: ~750 LOC including tests.

---

## 2. Architectural Decision Records

### ADR-001: Cascade lives in a new helper file, not in `EntityService`

**Context.** Three places need to trigger the cascade: `TaskService.TransitionStatus` (post-hook), `FeatureService.TransitionStatus` (post-hook), and the existing `CreateTask`/`CreateFeature` paths (which already have ad-hoc reopen logic). `EntityService` is shared by Bug, ChangeCard, Task, Feature, and Epic — adding the cascade there would either pollute Bug/ChangeCard transitions with parent-walking logic or require an entity-type-aware branch inside generic code.

**Decision.** Place the cascade in a new file `internal/services/cascade_reopen.go` inside the `services` package. Expose a single function `cascadeParentReopens(ctx, deps, trigger)` that the parent-aware services (Task and Feature) call from their post-hook blocks. `EntityService` is untouched.

**Rationale.**
- Keeps `EntityService` generic and entity-type-agnostic.
- Bugs and change-cards are excluded by construction — there is no cascade hook in `BugService.TransitionStatus` or `ChangeCardService.TransitionStatus`. SC6 is satisfied without defensive entity-type checks at the cascade entry point.
- Single source of truth for cascade behavior. The existing `maybeReopenParentFeature` and `maybeReopenParentEpic` helpers are refactored to call this function, satisfying SC9 ("no two parallel implementations").

**Alternatives considered.**
- (A) Embed cascade logic inside `EntityService.TransitionStatus`. Rejected: would force entity-type branching into shared code and risk Bug/ChangeCard regressions.
- (B) Place cascade inside each service file (`task_service.go`, `feature_service.go`). Rejected: would duplicate the history-lookup logic and risk drift between the two implementations.

**Consequences.** New file `cascade_reopen.go` becomes the canonical home for parent-reopen logic. Future cascade-related work (e.g., the deferred `auto_reopen_enabled` opt-out flag) lives in one place.

---

### ADR-002: Reopen target is "last non-terminal status from history", with workflow fallback

**Context.** SC3 requires each ancestor to return to "its own previous non-terminal status from `status_history`". The existing `maybeReopen*` helpers cheat by jumping to `_aggregation_[0]`, which is a hardcoded entry point regardless of where the entity actually was before completion. A feature that was in `in_qa` when prematurely completed must come back to `in_qa`, not to whatever `_aggregation_` points at.

**Decision.** Resolve the reopen target using a three-step lookup, encapsulated in `cascade_reopen.go::resolveReopenTarget`:

1. **Primary**: Query `entity_history` for the most recent row where `to_status` is **not** in the workflow's set of terminal statuses for that entity type. Return that `to_status`.
2. **Fallback A**: If no such row exists, return `workflowSvc.GetAggregationStatuses()[0]` if available.
3. **Fallback B**: If aggregation statuses are empty, return `workflowSvc.GetInitialStatusString()`.

The terminal-status set is computed once per cascade call by enumerating all statuses in the active workflow profile and filtering by `IsTerminalStatus`. This keeps the SQL `NOT IN (...)` clause workflow-driven rather than hardcoded.

**Rationale.**
- Honors SC3 in the common case (history exists, last non-terminal status is meaningful).
- Honors SC5 (workflow-agnostic) — the terminal set is derived from `workflow.Service`, never hardcoded.
- Honors UAT-6 (fallback when no history exists) by gracefully degrading to aggregation status, then to initial status.
- Encapsulates the resolution in one function — easy to unit-test, easy to extend with the deferred `reopen_fallback` config override.

**Alternatives considered.**
- (A) Use `_aggregation_[0]` always (current behavior). Rejected: violates SC3 and the primary UAT scenario.
- (B) Use `workflowSvc.GetInitialStatusString()` always. Rejected: re-runs the entire workflow from the beginning, losing all progress context.
- (C) Add a new column `previous_non_terminal_status` to entity tables. Rejected: introduces denormalization, requires migration, and is fragile under schema evolution. The history table already has the data.

**Consequences.** A single SQL query per ancestor per cascade. With features and epics having ≤1 ancestor each, the cascade adds at most 2 history queries on top of the existing transition. Performance impact: well within the 50ms budget.

---

### ADR-003: Cascade owns its own transaction; child transition commits first

**Context.** The epic's hard constraint says all transitions in a cascade must be atomic ("a partial cascade is unacceptable"). The current `EntityService.TransitionStatus` commits the child transition before the post-hook block runs. Wrapping the child transition + cascade in one outer transaction would require restructuring `EntityService` and thread `*sql.Tx` through every repository method — a much bigger surgery than this epic permits.

**Decision.** Adopt a **two-phase commit pattern** with the cascade owning its own transaction:

1. **Phase 1 (existing):** `EntityService.TransitionStatus` runs the child transition end-to-end and commits. This phase is unchanged.
2. **Phase 2 (new):** The post-hook in `TaskService.TransitionStatus` (or `FeatureService.TransitionStatus`) detects "prior status was terminal, new status is non-terminal" and calls `cascadeParentReopens`. This function opens a **new transaction**, performs the feature update, feature history write, epic update, and epic history write, then commits or rolls back **as a unit**.

The cascade transaction guarantees that **all parent updates are atomic relative to each other**. The child transition is already committed by the time the cascade starts, but this is acceptable because:

- The cascade only fires when the child transition has succeeded. There is no scenario where the child stays terminal but parents reopen (the trigger condition would be false).
- If the cascade transaction fails, the parents stay terminal — the child is correctly non-terminal but the dashboard shows a stale parent. This is the **same failure mode the system has today** (the existing `maybeReopen*` helpers are best-effort), so we are not regressing safety; we are improving it from "no cascade at all" to "atomic cascade across all parents".
- A `slog.Warn` is emitted on cascade failure with the triggering child key, the failing ancestor, and the SQL error. The orchestrator can treat this as a high-signal monitoring event.

**Rationale.**
- Satisfies the spirit of SC1 (atomicity across the parent chain) without an `EntityService` rewrite.
- Matches the existing best-effort post-hook idiom (`recalculateFeatureProgress`, `maybeReopenParentFeature`) — cascade failure does not fail the original child transition.
- Bounded blast radius: the only change to existing code paths is the addition of a post-hook call. The transaction-owning logic lives in the new helper file.
- Recoverable: if the cascade fails, a re-trigger (e.g., another regression on the same feature) will succeed and self-heal the parent chain.

**Alternatives considered.**
- (A) Single outer transaction wrapping child + cascade. Rejected: requires threading `*sql.Tx` through `EntityService.TransitionStatus`, every entity repository's `UpdateStatus`, and the history recorder. Out of scope for this epic; would be a multi-epic refactor.
- (B) No transaction at all (cascade as a sequence of independent updates). Rejected: violates SC1 outright. A failure between feature update and epic update would leave the dashboard in a worse state than today.
- (C) Synchronous retry loop on cascade failure. Rejected: hides root-cause errors and adds latency. The `slog.Warn` + idempotent re-trigger semantics are sufficient.

**Consequences.**
- The "atomic" guarantee in SC1 is **scoped to the parent chain** (feature + epic update together) rather than the full child + parents chain. This is documented explicitly in the test plan and the inline comments on `cascadeParentReopens`.
- Future epic could promote the cascade into the outer transaction once `EntityService` adopts a Tx-aware repository interface. The current design does not block that future work.

---

### ADR-004: Idempotency via terminal-status precondition check

**Context.** SC8 requires reopening an already non-terminal ancestor to be a no-op (no history row, no status change). UAT-3 covers concurrent regressions where two tasks regress in rapid succession and the second must observe the parents already reopened.

**Decision.** Inside `cascadeParentReopens`, before updating each ancestor, **re-fetch the ancestor's current status within the cascade transaction** and check `workflowSvc.ForLevel(level).IsTerminalStatus(currentStatus)`. If the ancestor is already non-terminal, skip the update **and** skip the history write. Continue walking up the chain (a non-terminal feature does not preclude a terminal epic — though in practice this is unusual).

**Rationale.**
- Correctness: re-fetching inside the transaction ensures we observe a consistent snapshot. No race window.
- Simplicity: no separate locking, no uniqueness constraint on history rows, no advisory locks.
- Matches the natural cascade walking logic (we already fetch the ancestor by ID; checking its status is free).

**Alternatives considered.**
- (A) Database-level `UPDATE ... WHERE status IN (terminal_set)`. Rejected: returns 0 rows on idempotent path, but still emits a transaction commit and we cannot easily distinguish "already reopened" from "update failed". The Go-side check is clearer.
- (B) Unique constraint on `entity_history(entity_id, notes)` to prevent duplicate reason rows. Rejected: makes legitimate repeated reopens (e.g., reopen → re-complete → reopen again) impossible.

**Consequences.** UAT-3 passes as a free side effect of the standard cascade flow. Idempotency does not require any post-cascade reconciliation.

---

### ADR-005: Auto-reopen audit trail uses `notes` column with structured prefix

**Context.** SC7 requires each auto-reopen row to be distinguishable from a manual transition and to reference the triggering child entity key. UAT-7 requires the row to be visually distinct in `shark status history` output. We do not want a schema migration if the existing column is sufficient.

**Decision.** Write auto-reopen reasons into `entity_history.notes` using a structured prefix:

```
auto_reopen: triggered by E07-F01-003 regression (task)
auto_reopen: triggered by E07-F01-003 regression (task) [fallback: initial status]
auto_reopen: triggered by E07-F01 reopen (feature)
```

The `auto_reopen:` prefix is the discriminator. The triggering child key follows. The trigger type ("regression" vs "creation") is appended. If the resolver fell back to aggregation or initial status, a `[fallback: ...]` suffix is added.

**No schema change. No `CurrentSchemaVersion` bump. No `skip_migrations` flip required.**

The CLI `shark status history` command (and JSON output) renders rows with `notes` starting with `auto_reopen:` using a distinct color/prefix label. This is a small presentation-layer addition in `internal/cli/commands/status.go` and adjacent formatters.

**Rationale.**
- Minimal database surface change (zero).
- Easy to grep, easy to filter in queries (`WHERE notes LIKE 'auto_reopen:%'`).
- Self-documenting: the reason field contains the trigger key and the cascade level.
- Consistent with the existing `maybeReopen*` helpers' use of the `notes` column.

**Alternatives considered.**
- (A) Add `triggered_by_entity_key TEXT` and `transition_kind TEXT` columns to `entity_history`. Rejected: requires migration, `CurrentSchemaVersion` bump, and `skip_migrations` developer flip. Not justified for an audit-trail label.
- (B) Add a separate `auto_reopen_events` table. Rejected: denormalization, harder to query, breaks the `shark status history` command's single-source-of-truth assumption.

**Consequences.** Future queries that need structured filtering (e.g., "show me all cascades triggered by tasks in epic E07") rely on `LIKE` patterns rather than indexed columns. This is acceptable given the small expected volume of auto-reopen rows.

---

### ADR-006: Two refinement-time decisions are deferred to BA/Tech refinement of features, not pre-resolved here

**Context.** The epic explicitly lists three "refinement-time decisions" that should be locked during BA/tech refinement. They are **not** architecture decisions — they are config-shape and behavior-toggle choices. This architecture document does not pre-resolve them.

**Decision.** This architecture is deliberately built so that **all three decisions are additive and non-blocking**:

| Decision | How architecture accommodates it without pre-resolving |
|----------|--------------------------------------------------------|
| Add `is_terminal: true` to `status_metadata`? | The terminal-status set is computed via `workflow.Service.IsTerminalStatus`. Whether that method reads from `_complete_` (current) or a new `is_terminal` field is internal to `workflow.Service` — the cascade does not care. |
| Add per-profile `auto_reopen_enabled` flag? | The cascade is gated on a single function call. Adding a config check (`if !cfg.AutoReopenEnabled() { return }`) at the top of `cascadeParentReopens` is a one-line change. |
| Add per-workflow `reopen_fallback` status override? | `resolveReopenTarget` already has a three-step fallback chain. Inserting a config-driven override between steps 1 and 2 is local to the resolver. |

**Rationale.** Avoids architecture-by-committee. The structural design is correct regardless of which way each refinement decision lands. Feature-level work tickets will pick up these decisions and slot the changes into the architecture defined here.

**Consequences.** Feature decomposition (next phase) creates separate features/tasks for each refinement decision once locked, but none of them will require restructuring the cascade.

---

## 3. Data Model Changes

**None.**

The `entity_history` table is reused as-is. The `notes` column carries the auto-reopen reason string per ADR-005. No new tables, no new columns, no migration. `CurrentSchemaVersion` does **not** change. `skip_migrations` does **not** need to be flipped.

This satisfies the epic's "prefer to avoid new tables" constraint without compromise.

---

## 4. Integration Approach

### 4.1 Trigger detection

The cascade fires from exactly two trigger points, each guarded by a precondition check. There are no other entry points.

#### Trigger A: Backward transition out of a terminal status

Installed in `TaskService.TransitionStatus` and `FeatureService.TransitionStatus` as post-hooks, after the existing `recalculateFeatureProgress` / child task counter calls.

```go
// Inside TaskService.TransitionStatus, after existing post-hooks
priorStatus := result.FromStatus
taskWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelTask)
if taskWf.IsTerminalStatus(priorStatus) && !taskWf.IsTerminalStatus(result.ToStatus) {
    // Fetch the task to get FeatureID (already cached in adapter.lastTask in most paths)
    if task, taskErr := s.repo.GetByKey(ctx, key); taskErr == nil && task != nil {
        s.cascadeFromTask(ctx, task, key)
    }
}
```

The same shape is used in `FeatureService.TransitionStatus`, calling `s.cascadeFromFeature(ctx, feature, featureKey)`, which only walks the epic leg.

#### Trigger B: New child created under a terminal parent

Already implemented for `CreateTask` and `CreateFeature`. The existing `maybeReopenParentFeature` and `maybeReopenParentEpic` helpers are **refactored** to call the unified `cascadeParentReopens` function instead of using their current ad-hoc aggregation-status logic. This satisfies SC9 (no parallel implementations) and unifies behavior.

```go
// In TaskService.CreateTask, replacing existing maybeReopenParentFeature
func (s *TaskService) maybeReopenParentFeature(ctx context.Context, featureKey, taskKey string) {
    // Refactored: delegate to unified cascade
    s.cascadeForNewChild(ctx, featureKey, taskKey, "task creation")
}
```

### 4.2 Cascade flow

The unified helper signature:

```go
// cascade_reopen.go
func cascadeParentReopens(ctx context.Context, deps cascadeDeps, trigger cascadeTrigger) {
    // 1. Open transaction
    tx, err := deps.db.BeginTx(ctx, nil)
    if err != nil { slog.Warn("cascade tx begin failed", ...); return }
    defer tx.Rollback()

    // 2. Walk feature leg
    feature := deps.featureRepo.GetByID(ctx, trigger.featureID)
    if deps.featureWf.IsTerminalStatus(string(feature.Status)) {
        target := resolveReopenTarget(ctx, deps.historyQuerier, models.EntityTypeFeature,
                                      feature.ID, deps.featureWf)
        deps.featureRepo.UpdateStatusTx(ctx, tx, feature.ID, target, ...)
        recordEntityHistoryTx(ctx, tx, deps.historyRepo, models.EntityTypeFeature,
                              feature.ID, oldStatus, target, false,
                              EntityHistoryOpts{Agent: "system", Reason: trigger.reason()})
    }

    // 3. Walk epic leg
    epic := deps.epicRepo.GetByID(ctx, feature.EpicID)
    if deps.epicWf.IsTerminalStatus(string(epic.Status)) {
        target := resolveReopenTarget(ctx, deps.historyQuerier, models.EntityTypeEpic,
                                      epic.ID, deps.epicWf)
        deps.epicRepo.UpdateStatusTx(ctx, tx, epic.ID, target, ...)
        recordEntityHistoryTx(ctx, tx, deps.historyRepo, models.EntityTypeEpic,
                              epic.ID, oldStatus, target, false,
                              EntityHistoryOpts{Agent: "system", Reason: trigger.reason()})
    }

    // 4. Commit
    if err := tx.Commit(); err != nil { slog.Warn("cascade tx commit failed", ...); return }
}
```

`cascadeFromFeature` is the same shape but skips step 2 (it starts at the epic leg directly).

### 4.3 Connections to existing code

- **Reads `workflow.Service`** via the level-scoped `ForLevel(workflow.LevelTask|LevelFeature|LevelEpic)`. No changes to `workflow.Service` API.
- **Writes `entity_history`** via the existing `EntityHistoryRecorder` interface. We add a Tx-aware variant `recordEntityHistoryTx` that mirrors `recordEntityHistory` but accepts `*sql.Tx` and routes through a new repo method `CreateTx`. This is a small additive change to the entity history repository.
- **Reads/writes feature and epic status** via new `UpdateStatusTx` methods. These are pure additions; existing `UpdateStatus` (non-Tx) remains for non-cascade callers.
- **Bug and ChangeCard services**: untouched. They never call `cascadeParentReopens`. SC6 is satisfied structurally.

### 4.4 Service wiring

`TaskService` and `FeatureService` constructors gain optional dependencies:

```go
func NewTaskService(
    repo TaskRepository,
    workflowSvc *workflow.Service,
    creatorSvc *taskcreation.Creator,
    noteRepo TaskNoteRepository,
    // NEW (all optional, nil-safe)
    db *repository.DB,
    featureRepo CascadeFeatureRepo,
    epicRepo CascadeEpicRepo,
    historyQuerier ParentReopenHistoryQuerier,
    historyTxRecorder EntityHistoryTxRecorder,
) *TaskService
```

When any of the new dependencies is nil, the cascade is silently disabled — preserving backward compatibility for tests and any embedding of `TaskService` that does not need cascade behavior. CLI and HTTP wiring inject all of them.

### 4.5 Where the existing code paths plug in

| Existing call site | Today | After E26 |
|--------------------|-------|-----------|
| `TaskService.CreateTask` (line 320, 382) | Calls `maybeReopenParentFeature` (feature only, aggregation status) | Calls refactored `maybeReopenParentFeature` which delegates to `cascadeParentReopens` (feature + epic, history-resolved target) |
| `FeatureService.CreateFeature` (line 776) | Calls `maybeReopenParentEpic` (epic only, aggregation status) | Calls refactored `maybeReopenParentEpic` which delegates to `cascadeFromFeature` (epic only via cascade helper, history-resolved target) |
| `TaskService.TransitionStatus` (post-hook block, line 614–620) | Calls `recalculateFeatureProgress` only | Adds cascade post-hook after `recalculateFeatureProgress` |
| `FeatureService.TransitionStatus` (post-hook block, line 228–235) | Counts child tasks only | Adds cascade post-hook |
| `EpicService.TransitionStatus` | No parent hook | No change (epic is top of hierarchy) |
| `BugService.TransitionStatus`, `ChangeCardService.TransitionStatus` | No parent hook | No change (excluded by construction) |

---

## 5. Migration Strategy

**No data migration required.** This is a behavioral change that activates on transitions occurring **after** the feature ships. The epic explicitly excludes retroactive cleanup of pre-existing stale parents.

**No schema migration required.** `CurrentSchemaVersion` does not change. `skip_migrations` does not need to be flipped on Turso.

**Rollout sequence:**

1. Land repository additions (`GetLastNonTerminalStatus`, `UpdateStatusTx`, `CreateTx` on entity history repo). These are pure additions and ship safely.
2. Land `cascade_reopen.go` with `cascadeParentReopens`, `resolveReopenTarget`, and the new interfaces. Comprehensive unit tests with mocked dependencies. No production wiring yet.
3. Land service-layer wiring updates to `TaskService` and `FeatureService` constructors (with nil-safe optional dependencies).
4. Update CLI and HTTP wiring to inject the new dependencies. Cascade is now live in production paths but only fires on the new triggers.
5. Refactor `maybeReopenParentFeature` and `maybeReopenParentEpic` to delegate to the unified helper. Existing tests for the `CreateTask` reopen path stay green (SC9).
6. Add CLI presentation update to `shark status history` to highlight `auto_reopen:` rows (UAT-7).

Each phase is independently shippable and reversible by reverting the most recent commit. The cascade is silent (slog.Warn-only on failure) until the helpers in phase 5 are refactored, so if anything misbehaves in phases 3–4, the only observable effect is duplicate cascade attempts on `CreateTask` (one from the legacy helper, one from the new path) — both of which are idempotent per ADR-004.

**Rollback plan.** If a critical issue is discovered post-rollout, the cascade can be disabled in one place by adding an early `return` at the top of `cascadeParentReopens`. This restores legacy behavior immediately without a code revert.

---

## 6. Performance Analysis

Per the epic constraint: cascade overhead must be ≤50ms per transition on a typical project.

**Per-cascade cost (worst case, both legs fire):**

| Operation | Estimated cost |
|-----------|----------------|
| `BeginTx` | 1ms |
| `featureRepo.GetByID` | 1ms (indexed) |
| `historyQuerier.GetLastNonTerminalStatus` (feature) | 2ms (indexed by entity_type + entity_id, ORDER BY changed_at DESC LIMIT 1) |
| `featureRepo.UpdateStatusTx` | 2ms |
| `historyRepo.CreateTx` (feature) | 2ms |
| `epicRepo.GetByID` | 1ms |
| `historyQuerier.GetLastNonTerminalStatus` (epic) | 2ms |
| `epicRepo.UpdateStatusTx` | 2ms |
| `historyRepo.CreateTx` (epic) | 2ms |
| `Commit` | 2ms |
| **Total worst case** | **~17ms** |

Idempotent skip path (parent already non-terminal): ~5ms (BeginTx + GetByID + Commit).

**Index requirements for `entity_history`:**

The existing schema does not specify an index on `(entity_type, entity_id, changed_at)`. The history-lookup query is:

```sql
SELECT to_status FROM entity_history
WHERE entity_type = ? AND entity_id = ? AND to_status NOT IN (?, ?, ?, ...)
ORDER BY changed_at DESC LIMIT 1
```

We rely on the existing `entity_history` index (created in `internal/db/db.go`). If the existing schema does not already cover `(entity_type, entity_id, changed_at DESC)`, this is the **one** thing that may need a schema change. This is flagged as a feature-decomposition checkpoint: the BA/tech refinement of F01 must verify the index exists, and if it does not, **adding the index is a migration** and follows the standard `CurrentSchemaVersion` bump protocol.

**Conclusion.** Comfortably under the 50ms budget with significant headroom.

---

## 7. Risks and Mitigations

| Risk | Source | Probability | Impact | Mitigation |
|------|--------|-------------|--------|------------|
| Cascade transaction commits but child transition row indicates stale parent | ADR-003 two-phase commit | Low | Medium | `slog.Warn` on cascade failure includes triggering child key. Re-trigger is idempotent and self-heals. Documented in inline comments and the Phase 5 release note. |
| `entity_history` index missing for `(entity_type, entity_id, changed_at)` | Existing schema | Medium | Medium | F01 refinement verifies. If missing, add via standard migration with `CurrentSchemaVersion` bump. Flagged in section 6. |
| Concurrent regressions race on parent status check | ADR-004 idempotency | Low | Low | Re-fetch happens inside the cascade transaction. SQLite with WAL mode serializes writers. Race window is closed by the transaction. |
| Refactoring `maybeReopen*` helpers breaks existing CreateTask reopen tests | Phase 5 | Low | Medium | SC9 explicitly requires existing tests to stay green. Run the existing `task_service_test.go` reopen suite (lines 2173–2617) as a regression check after Phase 5. |
| Bug or change-card transitions accidentally trigger cascade | Cross-cutting | Very low | Medium | Cascade is gated by the post-hook only existing in `TaskService` and `FeatureService`. `BugService` and `ChangeCardService` have no hook. Verified by negative test (UAT-5). |
| The deferred refinement decisions (section 2, ADR-006) constrain the architecture in ways we did not anticipate | Refinement | Low | Low | All three decisions are additive and slot in at well-defined extension points (config check at top of `cascadeParentReopens`, fallback chain in `resolveReopenTarget`, `IsTerminalStatus` internal). |

---

## 8. Out-of-Scope Confirmation (No Architecture Surface)

The following are **explicitly not** addressed by this architecture, matching the epic's "Out of Scope" section:

- Bugs and change-cards: never trigger cascade (no hook in their services).
- Forward-direction cascades (auto-completing parents): handled by existing rollup logic, untouched.
- Cross-epic reopens: cascade walks task → its own feature → its own epic only. No sibling traversal.
- Notifications/alerting: only `slog.Warn` on cascade failure. No webhooks, no email.
- UI/dashboard work: no HTTP API or web UI changes. Downstream consumers benefit automatically once data is correct.
- Retroactive backfill: not applicable. Cascade activates on new transitions only.
- Disabling cascade for orchestrator-driven transitions: orchestrator transitions go through the same `TaskService.TransitionStatus` and cascade exactly like manual ones. This is desired behavior.

---

## 9. Summary

| Aspect | Decision |
|--------|----------|
| New file | `internal/services/cascade_reopen.go` |
| New repo methods | `GetLastNonTerminalStatus`, `UpdateStatusTx` (feature), `UpdateStatusTx` (epic), `CreateTx` (entity history) |
| Schema changes | None (notes column reused) |
| Migration | None (no `CurrentSchemaVersion` bump, no `skip_migrations` flip) |
| Trigger sites | `TaskService.TransitionStatus`, `FeatureService.TransitionStatus`, `TaskService.CreateTask` (refactored), `FeatureService.CreateFeature` (refactored) |
| Reopen target | History lookup → aggregation fallback → initial-status fallback |
| Atomicity | Cascade transaction owns feature + epic update atomically. Child commit happens before. |
| Idempotency | Re-fetch ancestor status inside cascade transaction; skip if non-terminal. |
| Audit trail | `entity_history.notes` with `auto_reopen:` prefix |
| Profile-agnostic | Yes — terminal classification via `workflow.Service.IsTerminalStatus`. No hardcoded status names. |
| Bug/change-card exclusion | Structural — no cascade hook in their services |
| Backward compat | Existing CLI commands, JSON schemas, exit codes unchanged |
| Performance | ~17ms worst case, well under 50ms budget |

This design satisfies all 10 success criteria and 8 UAT scenarios in the epic PRD. Refinement-time decisions (terminal flag, opt-out toggle, fallback override) slot into extension points without restructuring.
