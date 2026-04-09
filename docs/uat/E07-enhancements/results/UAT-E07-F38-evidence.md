# UAT Evidence File — E07-F38 Auto-reopen parent entities on child regression

**Date:** 2026-04-07  
**Collector:** Claude Code (evidence only, no assessment)

---

## Spec Quotes (from feature.md Acceptance Criteria)

- **AC1:** When a task in a closed feature transitions backward into a non-terminal status, the parent feature is automatically reopened in the same transaction.
- **AC2:** When a feature in a closed epic transitions backward, the parent epic is automatically reopened.
- **AC3:** The reopen target status is the most recent non-terminal status from `entity_history`, with sensible fallback for entities lacking history.
- **AC4:** An `entity_history` row is written for each auto-reopen with a `notes` field that distinguishes it from manual transitions.
- **AC5:** `shark status history <key>` visually distinguishes auto-reopen rows.
- **AC6:** Existing creation-trigger reopens (`CreateTask`/`CreateFeature`) use the same cascade helper — no parallel implementations remain.
- **AC7:** Bugs and change-cards never trigger cascade reopens.
- **AC8:** Both basic and advanced workflow profiles work; the implementation does not hardcode status names.
- **AC9:** Cascade overhead is ≤50ms per transition (P95).
- **AC10:** All `make fmt && make lint && make test` checks pass.

---

## Task Spec Quotes (T-E07-F38-003, AC-T4)

> AC-T4: `maybeReopenParentEpic` (feature_service.go ~line 788) is refactored to call `cascadeParentReopens`; its old inline logic is removed.

---

## Implementation Code References

### cascade_reopen.go — entry point

**File:** `internal/services/cascade_reopen.go`

Key functions:
- `cascadeParentReopens(ctx, deps, trigger)` — main entry point, walks feature→epic, atomic tx
- `resolveReopenTarget(ctx, historyQuerier, entityType, entityID, levelWf)` — three-step fallback (history → aggregation → initial)
- `buildAutoReopenNotes(trigger, fallbackKind)` — formats `auto_reopen:` prefix per REQ-F-007

**No hardcoded status names:** `grep '"completed"\|"in_progress"\|"in_development"\|"in_qa"\|"todo"' cascade_reopen.go` → **no output** ✅

**Idempotency:** Phase 3 re-fetches ancestors inside the transaction via `GetByIDTx` before updating. If already non-terminal at re-fetch, skips update (no write, walk continues).

**Best-effort contract:** All errors inside `cascadeParentReopens` are logged via `slog.Warn` and swallowed. The triggering transition is never failed.

### TaskService wiring (internal/services/task_service.go:630)

```go
if s.cascadeEnabled() && taskErr == nil {
    go cascadeParentReopens(ctx, s.cascadeDepsBundle(), cascadeTrigger{...})
```
Wait — evidence note: this is `go cascadeParentReopens` — called in a goroutine. This means it is **async**, not within the same transaction as the transition.

### FeatureService wiring (internal/services/feature_service.go:279)

```go
if s.cascadeEnabled() {
    cascadeParentReopens(ctx, s.cascadeDepsBundle(), cascadeTrigger{...})
```
Feature service calls it **synchronously** (no `go` keyword).

### maybeReopenParentEpic in FeatureService (feature_service.go:847-869)

```go
func (s *FeatureService) maybeReopenParentEpic(...) {
    if !s.cascadeEnabled() {
        s.legacyMaybeReopenParentEpic(ctx, epic, featureKey)
        return
    }
    // Comment explains the design decision...
    s.legacyMaybeReopenParentEpic(ctx, epic, featureKey)
}
```
**Critical observation:** Even when `cascadeEnabled()` returns `true`, the code path falls through to `legacyMaybeReopenParentEpic`. The cascade helper is NOT called from the `CreateFeature` path.

### maybeReopenParentFeature in TaskService (task_service.go:1107)

```go
func (s *TaskService) maybeReopenParentFeature(...) {
    if !s.cascadeEnabled() {
        s.legacyMaybeReopenParentFeature(ctx, featureKey, taskKey)
        return
    }
    feature, err := s.featureService.GetFeature(ctx, featureKey)
    ...
    cascadeParentReopens(ctx, s.cascadeDepsBundle(), cascadeTrigger{...})
}
```
`CreateTask` path correctly calls `cascadeParentReopens` when deps wired. ✅

### CLI wiring (internal/cli/services_global.go)

```go
svc.SetCascadeDeps(d.db, d.featureRepo, epicRepo, entityHistoryRepo, entityHistoryRepo)
```
Present in `GetTaskService()`, `GetTaskServiceWithHistory()`, `GetTaskServiceWithDocs()`. ✅

### HTTP wiring (cmd/server/services.go:223-225)

```go
taskService.SetCascadeDeps(db, featureRepo, epicRepo, entityHistoryRepo, entityHistoryRepo)
featureService.SetCascadeDeps(db, epicRepo, entityHistoryRepo, entityHistoryRepo)
```
Both services wired. ✅

### Status history formatter (internal/cli/commands/status_group.go:529-533)

```go
// If the notes start with the "auto_reopen:" prefix, a bracketed "[auto-reopen]" label
if strings.HasPrefix(notes, "auto_reopen:") {
```
Uses `strings.HasPrefix` — matches notes format produced by `buildAutoReopenNotes`. ✅

---

## Test Code References

### TestCascade_TaskBackwardReopensFeature (cascade_reopen_test.go)
- Feature `featureID=101` in status `"completed"` (terminal)
- Epic `epicID=201` in status `"in_development"` (non-terminal, not affected)
- History querier returns `"in_qa"` as prior non-terminal for feature
- Asserts: `featureRepo.UpdateStatusTx` called once with `status="in_qa"`
- Asserts: `histTx.CreateTx` called once for feature with notes `auto_reopen:`

### TestCascade_TaskBackwardReopensEpic (cascade_reopen_test.go)
- Both feature and epic in terminal status
- Both should be reopened
- History querier returns `"in_qa"` for feature, `"in_development"` for epic
- Asserts: `featureRepo.updateStatusTxCalls == 1`, `epicRepo.updateStatusTxCalls == 1`
- Asserts: `histTx.calls == 2`

### TestCascade_IdempotentOnSecondRegression
- First cascade: feature is terminal → reopened
- Second cascade (same parents): feature now non-terminal (already reopened) → skipped
- Asserts: `featureRepo.updateStatusTxCalls == 1` total (not 2)

### TestCascade_ConcurrentCascadeOnSameAncestorWritesExactlyOneHistoryRow
- 10 goroutines fire cascade simultaneously
- Feature mock: first `GetByIDTx` returns terminal, subsequent return non-terminal
- Asserts: `histTx.calls == 1` (exactly one history row written)

### TestCascade_BugDoesNotTriggerCascade
- Verifies that `BugService` does not call `cascadeParentReopens`
- Structural: no cascade deps in BugService; cascade is only in TaskService/FeatureService

### TestCascade_TxFailureIsNonBlocking (8 sub-scenarios)
- `BeginTx failure` → cascade exits, no panic, triggering call unaffected
- `UpdateStatusTx feature failure` → cascade exits, no panic
- `Feature history write failure` → cascade exits, no panic
- `Feature refetch failure inside tx` → cascade exits, no panic
- `Epic update failure` → cascade exits, no panic
- `Epic history write failure` → cascade exits, no panic
- `Epic refetch failure inside tx` → cascade exits, no panic
- `Commit failure` → cascade exits, no panic

### status_group_test.go — auto-reopen label tests
- `"AC-T1: auto_reopen prefix gets distinct label in human output"` — ✅
- `"AC-T3: detection is purely strings.HasPrefix on auto_reopen:"` — ✅
- `"auto_reopen label visible in table row"` — row[4] contains `[auto-reopen]` ✅

---

## Test Execution Output

### Full test suite
```
make test → ALL PASS (no FAIL lines)
```

### Cascade unit tests (internal/services, TestCascade_*)
```
TestCascade_TaskBackwardReopensFeature       PASS
TestCascade_TaskBackwardReopensEpic          PASS
TestCascade_FeatureBackwardReopensEpic       PASS
TestCascade_NonTerminalFeatureContinuesToEpic PASS
TestCascade_AllAncestorsNonTerminalNoOp      PASS
TestCascade_HistoryRowFormat                 PASS
TestCascade_HistoryRowFormat_CreationTrigger PASS
TestCascade_IdempotentOnSecondRegression     PASS
TestCascade_TxFailureIsNonBlocking (8 sub)  PASS
TestCascade_OuterLookupFailureIsNonBlocking  PASS
TestCascade_HistoryRowHasChangedAt           PASS
TestCascade_EmptyTerminalSetGuard            PASS
TestCascade_ConcurrentCascadeOnSameAncestorWritesExactlyOneHistoryRow PASS
TestCascade_BugDoesNotTriggerCascade         PASS
```

### EntityHistory repository tests
```
TestEntityHistoryRepo_Create (5 sub)             PASS
TestEntityHistoryRepo_GetLastNonTerminalStatus (5 sub) PASS
TestEntityHistoryRepo_CreateTx (3 sub)           PASS
```

### Service wiring tests
```
TestTaskService_TransitionStatus_* (9 tests)     ALL PASS
TestFeatureService_TransitionStatus_* (8 tests)  ALL PASS
```

---

## Critical Observations for Assessor

### FINDING 1 — AC1 async concern
`TaskService.TransitionStatus` fires `cascadeParentReopens` in a **goroutine** (`go cascadeParentReopens(...)`). This means the cascade happens **outside** the transition transaction. AC1 says "in the same transaction" — the cascade uses its own separate transaction. Depending on interpretation:
- **Narrow reading:** The cascade transaction is separate from the triggering transition. Not literally "same transaction."
- **Broad reading:** The transition completes first, then the cascade atomically reopens parents. Atomicity of the cascade transaction itself is preserved; the two-phase approach is documented in the code as best-effort.

The feature PRD says: "Add a post-hook to `TaskService.TransitionStatus` and `FeatureService.TransitionStatus` that detects backward transitions out of terminal status and reopens the parent chain in a **single transaction**."

The task-level cascade (`TaskService`) uses a goroutine (async). The feature-level cascade (`FeatureService`) is synchronous.

### FINDING 2 — AC6 / AC-T4 gap: CreateFeature still uses legacy path
`FeatureService.maybeReopenParentEpic` calls `legacyMaybeReopenParentEpic` **even when cascade deps are wired**. The code has a comment explaining the design: they don't have a feature ID to pass to the cascade at `CreateFeature` call time (only an `*models.Epic`). Task spec AC-T4 requires this to be removed. It is not removed — the legacy path and its `legacyMaybeReopenParentEpic` function still exist and are called unconditionally.

**Impact:** When a new feature is created under a completed epic, the epic reopens to its aggregation status (legacy behavior), not its prior non-terminal status from history (cascade behavior). This violates AC6 and AC3 for the CreateFeature path.

**Scope note:** The UAT scenarios S1 (UAT rejection cycle) and S7 (failure non-blocking) are unaffected by this gap — those test the `TransitionStatus` post-hook path which correctly uses the cascade. Only the `CreateFeature` creation trigger is affected.

### FINDING 3 — AC1 atomicity: "same transaction" interpretation
The feature PRD description says "single transaction" for the cascade. In the implementation, `TaskService.TransitionStatus` fires the cascade in a goroutine, so it's a separate (but atomic within itself) transaction. `FeatureService.TransitionStatus` fires synchronously. Whether the goroutine approach satisfies AC1's "same transaction" language depends on whether the AC was intended to mean "one DB transaction encompassing both the task status change and the cascade" vs "the cascade itself is atomic."
