# F01-feature-context.md — E26 Codebase Research

## Executive Summary

Auto-reopen-on-child-regression is **partially implemented** for new entity creation but entirely absent for backward status transitions. Two `maybeReopen*` helpers exist in the service layer — both targeting one level only (task reopens feature; feature creation reopens epic) — and both use the workflow's `_aggregation_` status rather than the entity's own status history. The primary deliverable of E26 is:

1. Extend the existing `maybeReopen*` pattern to fire on **status backward transitions**, not just on creation.
2. Walk the **full parent chain** (task → feature → epic) in both triggers.
3. Replace the hardcoded "jump to aggregation status" with a **history lookup** for the most recent non-terminal status per ancestor.
4. Record auto-reopen transitions in `entity_history` with a distinguishable reason.

No new tables are required. The `entity_history.notes` column is sufficient to carry the `auto_reopen` reason. No new external packages are needed.

---

## Research Questions Addressed

1. What partial implementations already exist?
2. What patterns and conventions must be followed?
3. What are the exact integration points?
4. What can be extended vs what needs new code?
5. What are the technical risks?
6. Recommended implementation approach?

---

## Existing Implementations (With File Paths)

### 1. `maybeReopenParentFeature` — TaskService (new-task trigger only)

**File:** `internal/services/task_service.go`, lines 1046–1092

Called from:
- `CreateTask` (line 320 and 382) when a new task is created under a terminal feature.

**Behavior:**
- Gets the parent feature by `featureKey` via `featureService.GetFeature`.
- Calls `workflow.Service.ForLevel(LevelFeature).IsTerminalStatus(feature.Status)`.
- Gets `_aggregation_` statuses via `featureWf.GetAggregationStatuses()` and moves feature to `aggStatuses[0]` — **not a history lookup**.
- Calls `featureService.UpdateFeature(ctx, feature.Key, FeatureUpdates{Status: &targetStatus})`.
- Records history via `recordEntityHistory(...)` with `notes = "auto-reopened: new task ... created under terminal feature"`.
- Does **NOT** cascade further to the parent epic.

**Limitation:** Only fires on `CreateTask`, not on `TransitionStatus`. Only reopens feature, not epic.

---

### 2. `maybeReopenParentEpic` — FeatureService (new-feature trigger only)

**File:** `internal/services/feature_service.go`, lines 780–814

Called from:
- `CreateFeature` (line 776) when a new feature is created under a terminal epic.

**Behavior:**
- Receives the pre-fetched `*models.Epic` directly (already available during feature creation).
- Calls `epicWf.IsTerminalStatus(epic.Status)`.
- Gets `aggStatuses[0]` — **not a history lookup**.
- Calls `epicLookupRepo.Update(ctx, epic)` (bypasses service, writes directly to repo).
- Records history via `recordEntityHistory(...)` with `notes = "auto-reopened: new feature ... created under terminal epic"`.

**Limitation:** Only fires on `CreateFeature`, not on `TransitionStatus`. Epic is moved to aggregation status, not its own prior non-terminal status.

---

### 3. `TaskService.TransitionStatus` — No parent reopen

**File:** `internal/services/task_service.go`, lines 584–622

Post-hooks currently:
- Auto-unblock dependents (via `adapter.unblockedKeys`).
- Recalculate feature progress via `recalculateFeatureProgress(ctx, task.FeatureID)`.

**Missing:** No parent feature or epic reopen when task transitions backward from a terminal status.

---

### 4. `FeatureService.TransitionStatus` — No parent reopen

**File:** `internal/services/feature_service.go`, lines 209–237

Post-hooks currently:
- Count child tasks (via `taskRepo.ListByFeature`).

**Missing:** No parent epic reopen when feature transitions backward from a terminal status.

---

### 5. `EpicService.TransitionStatus` — No parent reopen

**File:** `internal/services/epic_service.go`, lines 202–227

No parent-level hooks (epic is the top of the hierarchy).

---

### 6. `EntityHistoryRepository` — Existing, minimal

**File:** `internal/repository/entityhistory/repository.go`

Methods:
- `Create(ctx, *models.EntityHistory) error` — inserts a row.
- `ListByEntity(ctx, EntityType, entityID) ([]*models.EntityHistory, error)` — returns all history for one entity, `changed_at DESC`.

**Missing method:** No `GetLastNonTerminalStatus(ctx, EntityType, entityID) (string, error)` — this needs to be added.

---

### 7. `EntityHistoryRecorder` interface — Already in EntityService

**File:** `internal/services/entity_service.go`, lines 15–20

```go
type EntityHistoryRecorder interface {
    Create(ctx context.Context, history *models.EntityHistory) error
}
```

`recordEntityHistory(...)` helper at lines 31–57 is the standard pattern for writing history. Used by TaskService, FeatureService, and EpicService.

---

### 8. `workflow.Service.IsTerminalStatus` and `GetAggregationStatuses`

**File:** `internal/workflow/service.go`

- `IsTerminalStatus(status string) bool` — line 146. Reads from `_complete_` in `special_statuses`.
- `GetAggregationStatuses() []string` — line 137. Reads from `_aggregation_`.
- `GetInitialStatusString() string` — line 84. Fallback for when no prior non-terminal history exists.
- `ForLevel(level string) *Service` — line 66. Produces a level-scoped service (LevelTask, LevelFeature, LevelEpic).

These are **already available** and must be used (do not hardcode status names).

---

### 9. `EntityService.TransitionStatus` — Extension point

**File:** `internal/services/entity_service.go`, lines ~160–270

`EntityService.TransitionStatus` is called by all three entity services. It:
1. Validates transition.
2. Normalizes target status.
3. Updates entity status via `repo.UpdateStatus`.
4. Records history via `recordEntityHistory`.
5. Creates rejection notes if backward.
6. Resolves orchestrator action.

The cascade reopen hook fits naturally **after step 7 returns** — as a post-hook in the calling entity service (same pattern as `recalculateFeatureProgress` in TaskService).

---

### 10. `EntityHistoryQuerier` — Separate read interface

**File:** `internal/services/entity_history_service.go`, lines 10–14

```go
type EntityHistoryQuerier interface {
    ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error)
}
```

Used by `EntityHistoryService` for displaying history. The same interface (or a smaller one with a new `GetLastNonTerminalStatus`) needs to be made available to the cascade logic.

---

### 11. `entity_history` table schema — No new columns needed

**File:** `internal/db/db.go`, lines 217–228

```sql
CREATE TABLE IF NOT EXISTS entity_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    from_status TEXT,
    to_status TEXT NOT NULL,
    changed_by TEXT,
    notes TEXT,        -- <-- sufficient for "auto_reopen: triggered by E07-F01-003 regression"
    forced INTEGER NOT NULL DEFAULT 0,
    rejection_reason TEXT,
    changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

The `notes` column is sufficient to encode the auto-reopen reason and triggering child key. **No schema migration required.**

---

### 12. Existing test coverage

- `internal/services/backward_transition_test.go` — tests backward transition detection for feature and epic services.
- `internal/services/task_service_test.go` lines 2173–2617 — tests `maybeReopenParentFeature` via `CreateTask`.
- `internal/repository/status_cascade_test.go` — tests status breakdown queries.

---

## Patterns and Conventions That Must Be Followed

| Convention | Location | Requirement |
|------------|----------|-------------|
| Terminal status detection | `workflow.Service.IsTerminalStatus()` | Never hardcode status names |
| Workflow level scoping | `workflow.Service.ForLevel(level)` | Use LevelTask/LevelFeature/LevelEpic |
| History recording | `recordEntityHistory(...)` in entity_service.go | Standard for all auto-transitions |
| Post-hook pattern | Entity service `TransitionStatus` post-hooks | Add cascade as a non-blocking post-hook |
| Best-effort non-blocking | `slog.Warn(...)` on error | Cascade failure must not fail the original transition |
| Constructor injection | All services | Dependencies injected via constructors, nil-safe |
| Transaction ownership | Services own transactions | Multi-repo operations wrap in a service-owned `BeginTx` |
| Interface at point of use | `EntityHistoryQuerier`, `TaskRepository`, etc. | Narrow interfaces defined in consuming package |

---

## Integration Points

### Where cascade hooks install

| Trigger | Location | New hook |
|---------|----------|----------|
| Task `CreateTask` (already present) | `task_service.go:320,382` → `maybeReopenParentFeature` | Extend to also cascade to epic |
| Task `TransitionStatus` backward | `task_service.go:617–621` (post-hook block) | Add `maybeCascadeParentReopens(ctx, task)` |
| Feature `CreateFeature` (already present) | `feature_service.go:776` → `maybeReopenParentEpic` | Already walks one level; now use history lookup |
| Feature `TransitionStatus` backward | `feature_service.go:228–236` (post-hook block) | Add `maybeCascadeParentEpicReopen(ctx, result)` |

### Repository changes needed

| Change | Location | Reason |
|--------|----------|--------|
| Add `GetLastNonTerminalStatus(ctx, EntityType, entityID, terminalStatuses []string) (string, bool, error)` | `internal/repository/entityhistory/repository.go` | History lookup for reopen target status |
| Add `GetByID(ctx, epicID) (*models.Epic, error)` to `FeatureEpicLookup` interface | `internal/services/feature_service.go:45` | Feature needs epic by ID to cascade from TransitionStatus |
| Add `GetByID(ctx, featureID) (*models.Feature, error)` to `AnalyticsFeatureRepository` | `internal/services/task_service.go:141` | Task needs feature by ID for cascade |

### History interface change needed

The `EntityHistoryQuerier` interface (currently only `ListByEntity`) needs a new method OR a new narrow interface:

```go
// ParentReopenHistoryQuerier is the interface needed by the cascade reopen logic.
type ParentReopenHistoryQuerier interface {
    GetLastNonTerminalStatus(ctx context.Context, entityType models.EntityType, entityID int64, terminalStatuses []string) (string, bool, error)
}
```

This method runs a simple query:
```sql
SELECT to_status FROM entity_history
WHERE entity_type = ? AND entity_id = ? AND to_status NOT IN (...)
ORDER BY changed_at DESC LIMIT 1
```

---

## Extension vs New Code Analysis

| Component | Extend or New | Rationale |
|-----------|---------------|-----------|
| `maybeReopenParentFeature` in TaskService | **Extend** — rename to `maybeCascadeParentReopens`, add epic leg and history-lookup | Pattern exists, only missing scope and trigger |
| `maybeReopenParentEpic` in FeatureService | **Extend** — add history-lookup instead of aggregation status | Pattern exists, only missing history lookup |
| `EntityHistoryRepository` | **Extend** — add `GetLastNonTerminalStatus` | Already has `ListByEntity`; new method is a targeted SQL query |
| `TaskService.TransitionStatus` post-hook | **Extend** — add cascade call in existing post-hook block | Same pattern as `recalculateFeatureProgress` |
| `FeatureService.TransitionStatus` post-hook | **Extend** — add cascade call in existing post-hook block | Same pattern as child task counter |
| `FeatureEpicLookup` interface | **Extend** — add `GetByID` | Currently only `GetByKey`; ID-based lookup needed for TransitionStatus path |
| `AnalyticsFeatureRepository` interface | **Extend** — add `GetByID` | Currently only `GetByKey` |
| `entity_history` schema | **No change** — `notes` column is sufficient | Existing column accommodates auto-reopen reason string |
| New `ParentReopenHistoryQuerier` interface | **New interface, extend concrete repo** | Small interface at point of use; concrete class already has the data |

---

## What Can Be Reused Without Modification

- `workflow.Service.IsTerminalStatus()` — used as-is.
- `workflow.Service.ForLevel()` — used as-is.
- `workflow.Service.GetInitialStatusString()` — used as-is for fallback.
- `workflow.Service.GetAggregationStatuses()` — used as fallback only (if no history found).
- `recordEntityHistory(...)` helper — used as-is.
- `EntityService.TransitionStatus` — called as-is; cascade runs in entity-specific post-hooks.
- `entity_history` table — used as-is; `notes` carries the auto-reopen label.
- Existing test mocks (`mockEntityHistoryRecorder`, `mockFeatureRepo`, etc.) — extend for new tests.

---

## Technical Risks and Feasibility Assessment

### Risk 1: History lookup returns stale data
**Probability:** Medium | **Impact:** Medium

If `entity_history` was not populated for all past transitions (e.g., entity was created before history recording was wired), `GetLastNonTerminalStatus` returns no result. Mitigation: fall back to `workflow.Service.GetInitialStatusString()` or `GetAggregationStatuses()[0]`.

### Risk 2: Transaction boundary for cascade
**Probability:** High | **Impact:** High

The epic constraint says all transitions must be in a single transaction. Currently `EntityService.TransitionStatus` uses `repo.UpdateStatus` (no Tx parameter). The cascade post-hooks call `featureService.UpdateFeature` and `epicLookupRepo.Update` — both separate DB calls.

**Options:**
1. Accept non-atomicity for now (cascade is best-effort, matches existing pattern of `maybeReopenParentFeature`) — breaks the SC1 atomicity requirement.
2. Add `StatusUpdateRawWithTx` equivalent to feature and epic repos, and own the transaction boundary in the cascade. This is more work but correct per the epic constraint.

**Recommendation:** Given the epic's explicit constraint ("single transaction"), Option 2 is required. The cascade function must own a transaction: `task update + feature update + epic update` in one `BeginTx/Commit`.

This is the single most significant implementation challenge.

### Risk 3: Idempotency (SC8)
**Probability:** Low | **Impact:** Medium

If two tasks regress simultaneously, both will try to reopen the feature. The second transition must be a no-op. This is achieved by checking `IsTerminalStatus(current)` before updating — if the feature was already reopened by the first, the second sees non-terminal and skips. Standard idempotency pattern, no special handling needed beyond the existing terminal check.

### Risk 4: Bugs/change-cards leaking into cascade
**Probability:** Low | **Impact:** Low

The cascade only fires when task `TransitionStatus` is called from `TaskService`. Bugs use `BugService.TransitionStatus` which has no cascade hook. The entity type check at the cascade entry point (`models.EntityTypeTask`) provides safety.

### Risk 5: Test coverage gap for cascade on TransitionStatus
**Probability:** High | **Impact:** Medium

Existing tests only cover `CreateTask` path. New tests needed for:
- Task backward transition triggers feature reopen.
- Feature reopen triggers epic reopen (full chain).
- Idempotency (second trigger is no-op).
- History lookup returns prior non-terminal status.
- Fallback when no history exists.

---

## Recommended Implementation Approach

### Phase 1: Add history lookup to EntityHistoryRepository (pure extension)

Add `GetLastNonTerminalStatus(ctx, entityType, entityID, terminalStatuses []string) (string, bool, error)` to `internal/repository/entityhistory/repository.go`.

Add narrow interface at point of use:

```go
// internal/services/task_service.go or a new file
type ParentReopenHistoryQuerier interface {
    GetLastNonTerminalStatus(ctx context.Context, entityType models.EntityType,
        entityID int64, terminalStatuses []string) (string, bool, error)
}
```

### Phase 2: Unified cascade function (new helper)

Create `cascadeParentReopens(ctx, childKey, featureID, historyQuerier)` in the service layer. This function:
1. Looks up feature by ID.
2. If feature is terminal: looks up its last non-terminal status in history (fallback: initial status).
3. Updates feature status via `featureRepo.UpdateStatus(ctx, tx, ...)`.
4. Records history for feature.
5. Looks up epic by feature.EpicID.
6. If epic is terminal: same history lookup + update.
7. Records history for epic.
8. Wraps steps 3–7 in a single transaction (with the original child transition if possible).

### Phase 3: Wire into task TransitionStatus

In `task_service.go:TransitionStatus` post-hook block (lines 614–620), after `recalculateFeatureProgress`:

```go
// Post-hook: cascade parent reopens if task was previously terminal
if task != nil && taskErr == nil {
    priorStatus := result.FromStatus // the status BEFORE this transition
    taskWf := s.entitySvc.GetWorkflowService()
    if taskWf.IsTerminalStatus(priorStatus) && !taskWf.IsTerminalStatus(result.ToStatus) {
        s.cascadeParentReopens(ctx, key, task.FeatureID)
    }
}
```

### Phase 4: Wire into feature TransitionStatus

In `feature_service.go:TransitionStatus` post-hook block (lines 228–235):

```go
// Post-hook: cascade epic reopen if feature was previously terminal
featureWf := s.entitySvc.GetWorkflowService()
if result.Transitioned && featureWf.IsTerminalStatus(result.FromStatus) &&
    !featureWf.IsTerminalStatus(result.ToStatus) {
    s.maybeCascadeEpicReopen(ctx, result.EntityID, result.EntityKey)
}
```

### Phase 5: Unify `maybeReopenParentFeature` (existing CreateTask path)

Replace current `aggStatuses[0]` logic with the new history-lookup approach. This unifies both triggers and satisfies the "no two parallel implementations" constraint in the epic.

### Phase 6: Tests

- Unit: `TestCascadeParentReopens_*` in `task_service_test.go` and `feature_service_test.go`.
- Repository: `TestGetLastNonTerminalStatus_*` in `internal/repository/entityhistory/`.
- Integration: profile-parameterized cascade tests (basic + advanced).

---

## Feasibility: Confirmed

The implementation is feasible within existing architecture:

- No new tables or external dependencies.
- No schema changes (notes column sufficient).
- No CLI command changes (behavior is transparent).
- Transaction challenge is solvable by adding `UpdateStatus` with Tx to feature and epic repos (both already have `BeginTx`).
- All terminal-status logic delegates to `workflow.Service` — profile-agnostic by construction.

The main complexity is the transaction boundary for the full cascade. Estimated additional implementation effort: medium (3–5 service-layer tasks, 2 repository tasks, comprehensive tests).

---

## References

- `internal/services/task_service.go` — lines 320, 382, 584–622, 1046–1092
- `internal/services/feature_service.go` — lines 43–51, 209–237, 780–814
- `internal/services/epic_service.go` — lines 20–45, 202–227
- `internal/services/entity_service.go` — lines 15–57, ~160–270
- `internal/services/backward_transition_test.go` — full file
- `internal/services/task_service_test.go` — lines 2173–2617
- `internal/repository/entityhistory/repository.go` — full file
- `internal/workflow/service.go` — lines 66, 84, 124–153, 137–143
- `internal/db/db.go` — lines 215–233, 436–438 (CurrentSchemaVersion = 11)
- `docs/plan/E26-auto-reopen-parent-entities-on-child-regression/epic.md`
