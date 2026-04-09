# UAT Test Guide - Auto-reopen parent entities on child regression

**Feature:** E07-F38 - Auto-reopen parent entities on child regression
**Epic:** E07 - Enhancements
**Generated:** 2026-04-07
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Ongoing enhancements to the Shark Task Manager, adding capabilities that improve developer workflow and AI agent orchestration.

**This Feature's Role:** E07-F38 solves a critical dashboard integrity problem — when a task regresses (UAT rejection, QA failure), the parent feature and epic remain "completed" on the dashboard while active child work is ongoing. This feature auto-reopens the parent chain in a single atomic transaction, restoring correct visibility for both human developers and AI orchestrators.

**Related Features:**
- E07-F30: Entity history tracking (foundational — E07-F38 reads from entity_history)
- E07-F31: Status history display (E07-F38 extends it with `[auto-reopen]` labels)

**Integration Points:**
- `entity_history` table: read by `GetLastNonTerminalStatus`, written by `EntityHistoryTxRecorder.CreateTx`
- `TaskService.TransitionStatus`: post-hook fires `cascadeParentReopens` on backward transitions
- `FeatureService.TransitionStatus`: post-hook fires `cascadeParentReopens` on backward transitions
- `shark status history <key>`: display layer shows `[auto-reopen]` label on auto-reopen rows

---

## Design Intent

**From Feature PRD (AC1-AC10):**
> AC1: When a task in a closed feature transitions backward into a non-terminal status, the parent feature is automatically reopened in the same transaction.
> AC2: When a feature in a closed epic transitions backward, the parent epic is automatically reopened.
> AC3: The reopen target status is the most recent non-terminal status from `entity_history`, with sensible fallback.
> AC4: An `entity_history` row is written for each auto-reopen with `notes` field starting `auto_reopen:`.
> AC5: `shark status history <key>` visually distinguishes auto-reopen rows.
> AC6: Existing creation-trigger reopens (CreateTask/CreateFeature) use the same cascade helper — no parallel implementations remain.
> AC7: Bugs and change-cards never trigger cascade reopens.
> AC8: Both workflow profiles work; no hardcoded status names.
> AC9: Cascade overhead ≤50ms per transition (P95).
> AC10: All `make fmt && make lint && make test` pass.

---

## Test Scenarios

### Scenario S1: UAT rejection cycle — full cascade

**Tasks covered:** T-E07-F38-001, T-E07-F38-002, T-E07-F38-003, T-E07-F38-004

**Covered by tests:**
- `TestCascade_TaskBackwardReopensFeature` — AC1
- `TestCascade_TaskBackwardReopensEpic` — AC1+AC2 full chain
- `TestCascade_FeatureBackwardReopensEpic` — AC2
- `TestFeatureService_TransitionStatus_*` — wiring verification

**Success Criteria:**
- [ ] Task backward transition triggers cascadeParentReopens
- [ ] Both feature and epic are reopened to history-resolved targets in single transaction
- [ ] History rows written with `auto_reopen:` prefix

---

### Scenario S2: New task under completed feature reopens chain (creation trigger)

**Tasks covered:** T-E07-F38-003, T-E07-F38-006

**Covered by tests:**
- `TestCascade_HistoryRowFormat_CreationTrigger` — notes format for creation
- `TestTaskService_CreateTask_*` — existing reopen tests

**Success Criteria:**
- [ ] CreateTask fires cascade (when cascade deps wired)
- [ ] CreateFeature fires cascade for epic reopen

---

### Scenario S3: Idempotency — concurrent regressions

**Tasks covered:** T-E07-F38-002, T-E07-F38-006

**Covered by tests:**
- `TestCascade_IdempotentOnSecondRegression`
- `TestCascade_ConcurrentCascadeOnSameAncestorWritesExactlyOneHistoryRow`

**Success Criteria:**
- [ ] Second regression observes parents already non-terminal, skips updates
- [ ] Exactly one `auto_reopen:` history row per ancestor per cascade sequence

---

### Scenario S4: Workflow profile compatibility

**Tasks covered:** T-E07-F38-002, T-E07-F38-006

**Covered by:**
- `cascade_reopen.go`: no hardcoded status names (verified by grep)
- Both short and long workflow configs tested via `newTestTaskWorkflowService`

**Success Criteria:**
- [ ] No hardcoded status strings in `cascade_reopen.go`
- [ ] Terminal classification via `workflow.Service.IsTerminalStatus`

---

### Scenario S5: Bugs/change-cards never cascade

**Tasks covered:** T-E07-F38-002, T-E07-F38-006

**Covered by tests:**
- `TestCascade_BugDoesNotTriggerCascade`

**Success Criteria:**
- [ ] No cascade hook in BugService or ChangeCardService
- [ ] Bug/change-card status transitions leave feature/epic unchanged

---

### Scenario S6: Fallback when no prior history

**Tasks covered:** T-E07-F38-002, T-E07-F38-006

**Covered by tests:**
- `TestCascade_EmptyTerminalSetGuard` — empty terminal set guard
- `resolveReopenTarget` three-step fallback logic

**Success Criteria:**
- [ ] History hit → returns prior non-terminal status
- [ ] No history → falls back to aggregation status
- [ ] No aggregation → falls back to initial status
- [ ] Fallback marker visible in history notes

---

### Scenario S7: Failure is non-blocking (best-effort)

**Tasks covered:** T-E07-F38-002

**Covered by tests:**
- `TestCascade_TxFailureIsNonBlocking` (8 sub-scenarios)
- `TestCascade_OuterLookupFailureIsNonBlocking` (2 sub-scenarios)

**Success Criteria:**
- [ ] DB errors during cascade do NOT fail the triggering transition
- [ ] Errors logged via `slog.Warn` with structured fields

---

### Scenario S8: Status history display label

**Tasks covered:** T-E07-F38-005

**Covered by tests:**
- `status_group_test.go` — `auto_reopen prefix gets distinct label`
- `auto_reopen label visible in table row`

**Success Criteria:**
- [ ] History rows with `notes` starting `auto_reopen:` show `[auto-reopen]` label
- [ ] Non-auto-reopen rows unchanged

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-04-07 |
| Result | Pending |
| Results File | - |

**Previous Sessions:** None (this is the first UAT run)
