---
feature_key: E07-F38
title: Test Plan — Auto-reopen parent entities on child regression
status: draft
created: 2026-04-06
references:
  - spec.md                  # Single source of truth for requirements and ACs
  - uat-plan.md              # UAT scenarios this plan satisfies
  - research.md              # Brownfield analysis and integration points
  - architecture.md          # ADR set (especially ADR-001 through ADR-006)
---

# E07-F38 Test Plan: Auto-reopen Parent Entities on Child Regression

Every test case in this plan traces to a specific acceptance criterion (AC-01 through AC-16) from `spec.md` Section 1.4. No orphaned tests. No untested ACs.

---

## 1. AC Test Matrix

### AC-01 — Task backward transition reopens parent feature to prior non-terminal status

**Requirement:** REQ-F-001, REQ-F-004  
**UAT:** UAT-S1

| Field | Value |
|-------|-------|
| Test name | `TestCascade_TaskBackwardReopensFeature` |
| File | `internal/services/cascade_reopen_test.go` (new) |
| Type | Unit (mocked repositories) |

**Setup:**
- Task `E07-F01-003` is in status `completed` (a terminal status per the test workflow config).
- Parent feature `E07-F01` is in status `completed` (terminal).
- `entity_history` for `E07-F01` contains one prior non-terminal row: `to_status = "in_qa"`.
- All repos mocked via function-field mock structs.

**Test steps:**
1. Call `TaskService.TransitionStatus(ctx, "E07-F01-003", "in_development", opts)`.
2. Allow the post-hook cascade to execute (sync, not goroutine).
3. Assert via captured mock calls.

**Expected outcome:**
- `CascadeFeatureRepo.UpdateStatusTx` called once with `id = featureID`, `status = "in_qa"`.
- `EntityHistoryTxRecorder.CreateTx` called once for feature with `notes` starting with `auto_reopen: triggered by E07-F01-003 regression`.
- Transaction committed once (no rollback).

**Edge cases:**
- Feature's prior non-terminal row is the very first row (only one history row exists — the terminal transition itself). The history query returns empty; the test verifies fallback path fires (see AC-07).
- History contains multiple non-terminal rows — verifies that `ORDER BY changed_at DESC LIMIT 1` returns the most recent.
- `from_status` is NOT terminal (forward transition): cascade MUST NOT fire. Verified by asserting `UpdateStatusTx` is never called.

---

### AC-02 — Task backward transition also reopens grandparent epic

**Requirement:** REQ-F-001  
**UAT:** UAT-S1

| Field | Value |
|-------|-------|
| Test name | `TestCascade_TaskBackwardReopensEpic` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) |

**Setup:**
- Task `E07-F01-003` in `completed`.
- Feature `E07-F01` in `completed`.
- Epic `E07` in `completed`.
- History for feature has prior non-terminal `in_qa`; history for epic has prior non-terminal `in_development`.

**Test steps:**
1. Call `TaskService.TransitionStatus(ctx, "E07-F01-003", "in_development", opts)`.
2. Capture all `UpdateStatusTx` and `CreateTx` calls.

**Expected outcome:**
- `CascadeFeatureRepo.UpdateStatusTx` called with feature ID and `"in_qa"`.
- `CascadeEpicRepo.UpdateStatusTx` called with epic ID and `"in_development"`.
- `CreateTx` called exactly twice: once for feature history row, once for epic history row.
- Both updates happen within a single transaction (same `*sql.Tx` object passed to both).

**Edge cases:**
- Epic history has no prior non-terminal rows; falls back to aggregation status (see AC-07).
- Feature has prior non-terminal history but epic does not; each leg independently runs its fallback.

---

### AC-03 — Feature backward transition reopens parent epic only

**Requirement:** REQ-F-002  
**UAT:** UAT-S1 (feature-leg scenario)

| Field | Value |
|-------|-------|
| Test name | `TestCascade_FeatureBackwardReopensEpic` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) |

**Setup:**
- Feature `E07-F01` transitions from `completed` (terminal) to `in_qa` (non-terminal).
- Epic `E07` in `completed`.
- History for epic has prior non-terminal `in_development`.

**Test steps:**
1. Call `FeatureService.TransitionStatus(ctx, "E07-F01", "in_qa", opts)`.
2. Capture mock calls.

**Expected outcome:**
- `CascadeEpicRepo.UpdateStatusTx` called once with epic's prior non-terminal `"in_development"`.
- `CascadeFeatureRepo.UpdateStatusTx` NEVER called (feature was the trigger, not an ancestor).
- `CreateTx` called once: one epic history row.

**Edge cases:**
- Feature transitions forward (non-terminal → non-terminal): cascade must not fire.
- Feature transitions from non-terminal → terminal (forward): cascade must not fire.
- Feature's `from_status` is `"completed"` but `"completed"` is configured as non-terminal in a custom profile: cascade must NOT fire (the terminal check must use the live workflow config, not a hardcoded list).

---

### AC-04 — Non-terminal feature is skipped; epic still checked

**Requirement:** REQ-F-008  
**UAT:** UAT-S3 (second-regression scenario)

| Field | Value |
|-------|-------|
| Test name | `TestCascade_NonTerminalFeatureContinuesToEpic` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) |

**Setup:**
- Task regresses from `completed` to `in_development`.
- Feature `E07-F01` is already in `in_qa` (non-terminal).
- Epic `E07` is in `completed` (terminal).
- Epic history has prior non-terminal `in_development`.

**Test steps:**
1. Call `TaskService.TransitionStatus(ctx, "E07-F01-003", "in_development", opts)`.

**Expected outcome:**
- `CascadeFeatureRepo.UpdateStatusTx` NEVER called (feature is already non-terminal → skipped).
- `CascadeEpicRepo.UpdateStatusTx` called once with `"in_development"`.
- Exactly one `CreateTx` call: for the epic history row only.

**Edge cases:**
- Feature is non-terminal AND epic is non-terminal: both skipped → full no-op (see AC-05).
- Feature skip log: if the cascade logs the skip at DEBUG level, verify the log message does not appear at WARN or ERROR.

---

### AC-05 — Both ancestors non-terminal: complete no-op

**Requirement:** REQ-F-008  
**UAT:** UAT-S3 (idempotent, all ancestors already open)

| Field | Value |
|-------|-------|
| Test name | `TestCascade_AllAncestorsNonTerminalNoOp` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) |

**Setup:**
- Task regresses from `completed` to `in_development`.
- Feature `E07-F01` is in `in_qa` (non-terminal).
- Epic `E07` is in `in_development` (non-terminal).

**Expected outcome:**
- `CascadeFeatureRepo.UpdateStatusTx`: 0 calls.
- `CascadeEpicRepo.UpdateStatusTx`: 0 calls.
- `EntityHistoryTxRecorder.CreateTx`: 0 calls.
- `BeginTx` on DB: 0 calls (transaction must not be opened for a no-op cascade).

**Edge cases:**
- All ancestors have 50+ history rows and are non-terminal: verifies the terminal-status check short-circuits before any history query is issued.

---

### AC-06 — Each auto-reopen writes a correctly formatted history row

**Requirement:** REQ-F-007  
**UAT:** UAT-S7

| Field | Value |
|-------|-------|
| Test name | `TestCascade_HistoryRowFormat` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) |

**Setup:**
- Task `E07-F01-003` regresses from `completed`.
- Both feature and epic are terminal.
- Both have prior non-terminal history (no fallback needed).

**Expected outcome (captured `*models.EntityHistory` fields):**

Feature history row:
- `EntityType == "feature"`
- `FromStatus == "completed"`
- `ToStatus == "in_qa"`
- `Notes` starts with `"auto_reopen: triggered by E07-F01-003 regression (task)"`
- `ChangedBy == "system"`
- No `[fallback:]` suffix.

Epic history row:
- `EntityType == "epic"`
- `FromStatus == "completed"`
- `ToStatus == "in_development"`
- `Notes` starts with `"auto_reopen: triggered by E07-F01-003 regression (task)"`
- No `[fallback:]` suffix.

**Edge cases:**
- Creation-trigger format: notes must be `"auto_reopen: triggered by <key> creation (task)"` — tested separately in `TestCascade_HistoryRowFormat_CreationTrigger`.
- `ChangedBy` must never be the user's agent ID for cascade rows.
- Notes are 120 chars or fewer (prevent DB truncation if a notes field limit exists).

---

### AC-07 — Fallback to aggregation status when no prior non-terminal history

**Requirement:** REQ-F-004 (step 2)  
**UAT:** UAT-S6 sub-scenario A

| Field | Value |
|-------|-------|
| Test name | `TestResolveReopenTarget_FallbackAggregation` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) |

**Setup:**
- Feature `E07-F01` is `completed`.
- `GetLastNonTerminalStatus` mock returns `("", false, nil)` (empty result).
- Workflow profile defines `_aggregation_` statuses: `["in_development", "in_qa"]`.

**Expected outcome:**
- `resolveReopenTarget` returns `("in_development", "aggregation", nil)`.
- History row `notes` includes `[fallback: aggregation]`.

**Edge cases:**
- `GetLastNonTerminalStatus` returns an error: `resolveReopenTarget` must NOT silently use the aggregation fallback on error — it must return the error. The cascade then logs via `slog.Warn` and returns (per REQ-N-003).
- Aggregation slice has more than one entry: verifies `aggStatuses[0]` is used (the first element).
- Aggregation slice is empty: falls through to step 3 (initial status, see AC-08).

---

### AC-08 — Fallback to workflow initial status when no aggregation statuses

**Requirement:** REQ-F-004 (step 3)  
**UAT:** UAT-S6 sub-scenario B

| Field | Value |
|-------|-------|
| Test name | `TestResolveReopenTarget_FallbackInitial` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) |

**Setup:**
- `GetLastNonTerminalStatus` returns empty.
- Workflow profile defines `_aggregation_: []` (empty).
- `GetInitialStatusString()` returns `"draft"`.

**Expected outcome:**
- `resolveReopenTarget` returns `("draft", "initial", nil)`.
- History row `notes` includes `[fallback: initial]`.

**Edge cases:**
- `GetInitialStatusString()` returns an empty string: verifies a defensive check prevents an empty-string status from being written.
- The fallback suffix must appear in the `notes` column even when the initial fallback is itself a terminal status (configuration error — the cascade should log a WARN and skip the update rather than reopening into a terminal status).

---

### AC-09 — Idempotent on second regression: no duplicate history rows

**Requirement:** REQ-F-008  
**UAT:** UAT-S3

| Field | Value |
|-------|-------|
| Test name | `TestCascade_IdempotentOnSecondRegression` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) with call-count assertions |

**Setup (first regression):**
- Both parents terminal. First task regression fires; parents reopen. Captures first round of `UpdateStatusTx` and `CreateTx` calls.

**Setup (second regression, simulated):**
- After first regression, parents are now non-terminal. Second mock call to `GetByID` on feature and epic now returns non-terminal status.

**Test steps:**
1. Simulate first call: parents terminal → cascade fires → parents transition.
2. Reset mock state so `GetByID` returns the now-non-terminal status.
3. Simulate second call: cascade invoked again.

**Expected outcome after second regression:**
- `UpdateStatusTx` total call count: 2 (one per parent, both from first regression only).
- `CreateTx` total call count: 2.
- After second call: still 2 (no new calls added).

**Edge cases:**
- Rapid concurrent calls (goroutine-based): outside the mock scope, but a note in the test documents that the in-transaction re-fetch provides the idempotency guarantee.

---

### AC-10 — Existing `maybeReopenParentFeature` tests pass after refactor

**Requirement:** REQ-F-003 (SC9 regression check)  
**UAT:** UAT-S2, INT-6

| Field | Value |
|-------|-------|
| Test name | All `TestTaskService_CreateTask_Reopen*` (lines 2173–2617 of `task_service_test.go`) |
| File | `internal/services/task_service_test.go` (existing, NOT modified) |
| Type | Unit (mocked) |

**What to verify:**
- `TestTaskService_CreateTask_ReopensTerminalFeature` — feature reopens when task created under terminal feature.
- `TestTaskService_CreateTask_ReopenRecordsHistory` — history row written.
- `TestTaskService_CreateTask_ReopensArchivedFeature` — archived (terminal) feature reopens.
- `TestTaskService_CreateTask_NoReopenNonTerminalFeature` — non-terminal feature unchanged.
- `TestTaskService_CreateTask_NoReopenNilFeatureService` — nil feature service doesn't crash.
- `TestTaskService_CreateTask_ReopenFailureDoesNotFailCreate` — cascade failure is non-blocking.
- `TestTaskService_CreateTask_ReopensCancelledFeature` — cancelled (terminal) feature reopens.
- `TestTaskService_CreateTask_CreatorSvcPath_ReopensFeature` — creator-service path also reopens.

**Expected outcome:** All 8 tests pass against the refactored implementation with no test modifications. If any test explicitly asserts `aggStatuses[0]` and the fixture now provides history rows, that test is updated as part of the refactor deliverable (not a failure of this test plan).

---

### AC-11 — Bug regression does not trigger cascade

**Requirement:** REQ-F-006  
**UAT:** UAT-S5

| Field | Value |
|-------|-------|
| Test name | `TestCascade_BugDoesNotTriggerCascade` |
| File | `internal/services/cascade_reopen_test.go` (or a new `internal/services/bug_service_test.go` section) |
| Type | Unit (mocked) |

**Setup:**
- Feature `E07-F01` in `completed`. Epic `E07` in `completed`.
- `BugService.TransitionStatus` called with bug `B042` transitioning from `closed` to `open`.

**Expected outcome:**
- `CascadeFeatureRepo.UpdateStatusTx`: 0 calls.
- `CascadeEpicRepo.UpdateStatusTx`: 0 calls.
- `EntityHistoryTxRecorder.CreateTx`: 0 calls.

**Implementation note:** This test validates the structural exclusion (no cascade hook in `BugService`). The test confirms by inspection that `BugService.TransitionStatus` does not call any `cascade*` function.

**Edge cases:**
- Change-card regression: identical verification for `ChangeCardService.TransitionStatus` (`TestCascade_ChangeCardDoesNotTriggerCascade`).
- Bug linked to feature via `shark link` or `related-docs`: link existence must not affect cascade behavior.

---

### AC-12 — Cascade fires correctly on both basic and advanced workflow profiles

**Requirement:** REQ-F-005  
**UAT:** UAT-S4

| Field | Value |
|-------|-------|
| Test name | `TestCascade_BasicProfile` and `TestCascade_AdvancedProfile` (table-driven) |
| File | `internal/services/backward_transition_test.go` (extend existing) |
| Type | Unit (mocked) |

**Test table:**

| Profile | Terminal statuses | Prior non-terminal history | Expected reopen target |
|---------|------------------|---------------------------|----------------------|
| basic | `["completed"]` | `"in_progress"` | `"in_progress"` |
| advanced | `["completed", "cancelled"]` | `"in_qa"` | `"in_qa"` |
| custom | `["shipped"]` | `"building"` | `"building"` |

**Setup per profile:**
- A temp config file written per `newTestEpicWorkflowServiceForBackward` pattern (already established in `backward_transition_test.go`).
- History mock returns the appropriate prior non-terminal for that profile.

**Expected outcome:**
- Each profile iteration: correct reopen target used, correct fallback markers in notes (none expected, history is populated).
- No hardcoded status names appear in `cascade_reopen.go` (verified by code review during implementation; the test enforces profile-agnosticism by using custom profile names not found anywhere in the codebase).

**Edge cases:**
- Custom profile with no `_complete_` entries: `IsTerminalStatus` returns false for all statuses; cascade never fires. Verified with a test variant where the trigger task's `from_status` is the custom profile's terminal but `IsTerminalStatus` is mocked to return false.

---

### AC-13 — Cascade transaction failure is non-blocking

**Requirement:** REQ-N-003  
**UAT:** Implicit (non-regression of child transition)

| Field | Value |
|-------|-------|
| Test name | `TestCascade_TxFailureIsNonBlocking` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Unit (mocked) |

**Setup:**
- Parents are terminal; cascade should fire.
- Mock `db.BeginTx` (or `UpdateStatusTx`) returns a simulated error.

**Sub-test A — BeginTx failure:**
- `cascadeDeps.db.BeginTx` returns `errors.New("simulated BeginTx failure")`.
- Original `TransitionStatus` call returns `(task, nil)` (success).
- `slog.Warn` emitted with structured fields.

**Sub-test B — UpdateStatusTx failure (feature leg):**
- `CascadeFeatureRepo.UpdateStatusTx` returns error.
- Transaction rolls back.
- `CascadeEpicRepo.UpdateStatusTx`: 0 calls (stopped before epic leg).
- `EntityHistoryTxRecorder.CreateTx`: 0 calls.
- Child transition still returns success.

**Sub-test C — Commit failure:**
- `UpdateStatusTx` calls succeed; `tx.Commit()` returns error.
- Both updates appear to roll back (verified by checking captured call counts are non-zero but no commit occurred).
- Child transition still returns success.

**Expected outcome (all sub-tests):** `TransitionStatus` return value is unchanged. No error propagated to caller. Exactly one `slog.Warn` log emitted per cascade failure.

**Edge cases:**
- `slog.Warn` must include `triggerKey`, `ancestor`, and `error` fields (verified by capturing log output using a test slog handler or by asserting the structured log message contains the key strings).

---

### AC-14 — `shark status history` renders auto-reopen rows with distinct label

**Requirement:** REQ-F-009  
**UAT:** UAT-S7

| Field | Value |
|-------|-------|
| Test name | `TestStatusHistoryFormatter_AutoReopenLabel` |
| File | `internal/cli/commands/status_test.go` (new section, or dedicated formatter test) |
| Type | Unit (mocked history list, no real DB) |

**Setup:**
- Mock history list containing three rows:
  1. Manual transition row: `notes = "QA rejected, step 3 failed"`.
  2. Auto-reopen row: `notes = "auto_reopen: triggered by E07-F01-003 regression (task)"`.
  3. Auto-reopen row with fallback: `notes = "auto_reopen: triggered by E07-F02-001 creation (task) [fallback: aggregation]"`.

**Human output assertions:**
- Row 2 rendered with distinct label (e.g., `[auto]` prefix, or distinctly colored column in the table). The exact presentation token is a developer-time decision; the test asserts the label token is present using a string-contains check.
- Row 1 has no such label.
- Row 3 renders the fallback marker visibly.

**JSON output assertions:**
- `--json` output: `notes` field for row 2 starts with `"auto_reopen:"`.
- `--json` output: `notes` field for row 1 does NOT start with `"auto_reopen:"`.
- `jq`-equivalent filter `select(.notes | startswith("auto_reopen:"))` returns exactly 2 rows.

**Edge cases:**
- Empty history list: formatter renders no rows, no panic.
- All rows are auto-reopen: formatter renders all rows with the label.
- `notes` is `null`/empty: no label applied, no crash.

---

### AC-15 — Cascade overhead ≤50ms P95

**Requirement:** REQ-N-001  
**UAT:** PERF-1

| Field | Value |
|-------|-------|
| Test name | `BenchmarkCascade_BothLegs` |
| File | `internal/services/cascade_reopen_test.go` |
| Type | Benchmark (uses test database, not mocks) |

**Setup:**
- Test DB with `~500 tasks`, `~50 history rows per entity` (matches UAT PERF-1 specification).
- Feature and epic both in terminal state.
- Benchmark runs `cascadeParentReopens` 1000 iterations.

**Expected outcome:**
- `go test -bench=BenchmarkCascade_BothLegs -benchtime=1000x` results in P95 ≤50ms per operation.
- A separate sub-benchmark `BenchmarkCascade_IdempotentPath` (both ancestors already non-terminal) shows P95 ≤10ms.

**Edge cases:**
- Benchmark must disable the slog.Warn noise with a test no-op slog handler.
- Benchmark fixtures must not share test DB state with repository tests (use separate test DB prefixed rows).

---

### AC-16 — `make fmt && make lint && make test` exits 0

**Requirement:** REQ-N-007  
**UAT:** CI gate

| Field | Value |
|-------|-------|
| Test name | (CI gate — not a unit test) |
| File | Makefile |
| Type | Integration gate |

**Verification:**
- After all implementation tasks complete, run `make fmt && make lint && make test`.
- Exit code must be 0.
- No new `golangci-lint` warnings introduced compared to the baseline before this feature.
- Test suite includes all new tests listed in this plan.

---

## 2. Integration Scenarios

These cross-component tests verify boundaries between the cascade, existing post-hooks, and other subsystems. They map directly to `uat-plan.md` INT-1 through INT-6.

### INT-1: Cascade coexists with `recalculateFeatureProgress`

**Components:** `TaskService.TransitionStatus`, `recalculateFeatureProgress`, `cascadeParentReopens`  
**File:** `internal/services/task_service_test.go` (new section)

**Verify:**
- After task regresses from `completed`, both post-hooks fire sequentially.
- `recalculateFeatureProgress` uses the task's NEW non-terminal status.
- `cascadeParentReopens` uses the task's OLD terminal `from_status` to decide whether to fire.
- No race: both post-hooks are synchronous in the same goroutine.
- Feature's `progress_weight` rollup after the test shows the regressed task no longer counted at 100%.

**Test setup:**
- Feature has 2 tasks: one stays `completed`, one regresses.
- Expect progress to drop from 100% to 50%.
- Expect feature status to reopen via cascade.

---

### INT-2: Cascade coexists with auto-unblock dependents

**Components:** `TaskService.TransitionStatus`, `adapter.unblockedKeys`, `cascadeParentReopens`  
**File:** `internal/services/task_service_test.go` (new section)

**Verify:**
- A task with dependents regresses from `completed`.
- Auto-unblock logic (which fires when a task transitions to non-blocked) runs as normal.
- Cascade also fires for terminal parents.
- No interference between the two post-hooks.

**Test setup:**
- Task A `completed` with task B blocked-by A. Feature and epic terminal.
- A regresses: cascade fires (feature reopens), auto-unblock checks for dependents of A.
- Assert: no panic, no duplicate history writes, final state is coherent.

---

### INT-3: Orchestrator action reflects reopened state immediately

**Components:** `cascadeParentReopens`, `epic_service.go` orchestrator resolver  
**File:** `internal/services/epic_service_test.go` (new assertion) or CLI integration test

**Verify:**
- After cascade reopens epic `E07` from `completed` to `in_development`, a subsequent `shark epic get E07 --json` returns `orchestrator_action` matching the workflow config for `in_development`.
- The stale `completed` orchestrator action is NOT returned.

**Test setup:**
- Use real test DB (repository-layer integration test).
- Epic is `completed`, trigger cascade via `TaskService.TransitionStatus`.
- Read epic's `orchestrator_action` field after cascade.

---

### INT-4: Cascade fires identically via HTTP API

**Components:** HTTP task status handler, `TaskService.TransitionStatus`, cascade  
**File:** `cmd/server/` integration test or `internal/cli` end-to-end test  
**Scope:** Verify parity, not a new full test suite

**Verify:**
- `PATCH /api/v1/tasks/{key}/status` with backward transition triggers cascade.
- HTTP 200 returned even if cascade emits WARN.
- Subsequent `GET /api/v1/features/{key}` reflects reopened status.

**Note:** This is a smoke-level integration check, not a full regression suite for the HTTP layer.

---

### INT-5: Cascade fires via `shark task next-status` advance command

**Components:** CLI `status advance` command, `TaskService.TransitionStatus`, cascade  
**File:** End-to-end CLI test or documented manual smoke test

**Verify:**
- `shark status advance E07-F01-003` from a terminal status triggers the cascade.
- Applies regardless of CLI verb: `set`, `advance`, `next-status`, `reopen`, `set-status`.

**Test setup:**
- Unit test: mock `TaskService` and verify `TransitionStatus` is called; cascade behavior is verified separately by cascade unit tests.
- Manual smoke test: documented in `test_plans/smoke-tests.md` (to be produced during implementation).

---

### INT-6: Existing CreateTask reopen test suite stays green [SC9]

**Components:** Refactored `maybeReopenParentFeature`, unified `cascadeParentReopens`  
**File:** `internal/services/task_service_test.go` lines 2173–2617 (existing tests)

**Verify:**
- `go test ./internal/services/ -run TestTaskService.*Reopen` is green.
- If any test asserts `aggregation_status[0]` but the unified helper now resolves via history, that test receives updated fixture state (a prior non-terminal history row) as part of Phase 5 deliverables.
- No test file modification of the behavioral assertions (only fixture adjustments are permitted).

---

## 3. Test Infrastructure

### 3.1 Existing patterns to follow

| Pattern | Location | Purpose | Use for |
|---------|----------|---------|---------|
| Function-field mock structs | `internal/services/task_service_test.go`, `backward_transition_test.go` | Inline per-test behavior | All new service-layer mocks |
| `newTestEpicWorkflowServiceForBackward` helper | `internal/services/backward_transition_test.go:19` | Creates a temp workflow config and workflow.Service | Profile-parameterized cascade tests (AC-12) |
| `config.ClearWorkflowCache()` + `t.Cleanup` | `backward_transition_test.go:67–70` | Clears global workflow config cache between tests | Any test that writes a temp config |
| Repository test: clean-before + defer cleanup | `internal/repository/entityhistory/repository_test.go` | DELETE rows before test, defer DELETE after | All new repository tests |
| `test.GetTestDB()` pattern | `internal/test/` | Returns shared test DB connection | `GetLastNonTerminalStatus`, `UpdateStatusTx` repository tests |
| `test.SeedTestData()` pattern | `internal/test/` | Seeds epic/feature for repo tests | Repository tests requiring parent entities |
| Existing `mockEpicRepo`, `mockFeatureRepo` | `internal/services/backward_transition_test.go` | Pre-built mock structs | Extend for `CascadeFeatureRepo`, `CascadeEpicRepo` |

### 3.2 New test helpers needed

| Helper | File | Purpose |
|--------|------|---------|
| `newCascadeTestWorkflowService(t, terminalStatuses, aggStatuses)` | `internal/services/cascade_reopen_test.go` | Creates a temp config with configurable terminal and aggregation statuses for profile-parameterized tests. |
| `mockParentReopenHistoryQuerier` | `internal/services/cascade_reopen_test.go` | Mock implementing `ParentReopenHistoryQuerier` with a `GetLastNonTerminalStatusFn` field. |
| `mockCascadeFeatureRepo` | `internal/services/cascade_reopen_test.go` | Mock implementing `CascadeFeatureRepo` with `GetByIDFn` and `UpdateStatusTxFn` fields. |
| `mockCascadeEpicRepo` | `internal/services/cascade_reopen_test.go` | Mock implementing `CascadeEpicRepo` with `GetByIDFn` and `UpdateStatusTxFn` fields. |
| `mockEntityHistoryTxRecorder` | `internal/services/cascade_reopen_test.go` | Mock implementing `EntityHistoryTxRecorder` with a `CreateTxFn` field and a call log slice. |
| `captureLogHandler` (slog handler) | `internal/services/cascade_reopen_test.go` | An `slog.Handler` that captures `slog.Warn` calls into a slice for assertion in AC-13 and error-path tests. |

### 3.3 Repository tests (real DB)

These tests use the real test database per `.claude/rules/testing/repository-tests.md`.

| Test | File | What it tests |
|------|------|---------------|
| `TestGetLastNonTerminalStatus_HappyPath` | `internal/repository/entityhistory/repository_test.go` | Returns the most recent non-terminal `to_status`. |
| `TestGetLastNonTerminalStatus_EmptyResult` | Same | Returns `("", false, nil)` when all history rows are terminal. |
| `TestGetLastNonTerminalStatus_TerminalSetFiltering` | Same | Rows with `to_status IN (terminalStatuses)` are excluded. |
| `TestGetLastNonTerminalStatus_MultipleRows_ReturnsMostRecent` | Same | `ORDER BY changed_at DESC LIMIT 1` selects newest non-terminal row. |
| `TestEntityHistoryRepo_CreateTx_HappyPath` | Same | Writes row inside an open transaction; commits successfully. |
| `TestEntityHistoryRepo_CreateTx_Rollback` | Same | Row does not persist if caller rolls back the transaction. |
| `TestFeatureRepo_UpdateStatusTx_HappyPath` | `internal/repository/feature_repository_test.go` | Updates feature status inside a transaction. |
| `TestFeatureRepo_UpdateStatusTx_Rollback` | Same | Rollback leaves feature status unchanged. |
| `TestEpicRepo_UpdateStatusTx_HappyPath` | `internal/repository/epic_repository_test.go` | Updates epic status inside a transaction. |
| `TestEpicRepo_UpdateStatusTx_Rollback` | Same | Rollback leaves epic status unchanged. |

**Cleanup pattern for all repository tests:**
```go
// Before test (mandatory)
_, _ = database.ExecContext(ctx, "DELETE FROM entity_history WHERE entity_id = ? AND entity_type = ?", entityID, entityType)

// After test (defer)
defer database.ExecContext(ctx, "DELETE FROM entity_history WHERE entity_id = ? AND entity_type = ?", entityID, entityType)
```

### 3.4 Coverage requirements

Per REQ-N-006:
- `internal/services/cascade_reopen.go`: ≥80% line coverage.
- All error paths (`BeginTx` failure, history-lookup error, `UpdateStatusTx` failure, Commit failure): 100% explicit unit test coverage.
- `internal/repository/entityhistory/repository.go` new methods: covered by repository tests above.
- `internal/repository/feature_repository.go` and `epic_repository.go` `UpdateStatusTx`: covered by repository tests above.

---

## 4. Test File Summary

| File | Status | Tests contributed |
|------|--------|-------------------|
| `internal/services/cascade_reopen_test.go` | New | AC-01..AC-09, AC-12, AC-13, AC-15 |
| `internal/services/backward_transition_test.go` | Extend | AC-12 profile table, INT-1, INT-2 |
| `internal/services/task_service_test.go` | Extend (fixtures only) | AC-10 regression check, INT-1, INT-2 |
| `internal/services/feature_service_test.go` | Extend | AC-03, `maybeReopenParentEpic` refactor coverage |
| `internal/repository/entityhistory/repository_test.go` | Extend | `GetLastNonTerminalStatus` and `CreateTx` (3.3) |
| `internal/repository/feature_repository_test.go` | Extend | `UpdateStatusTx` (3.3) |
| `internal/repository/epic_repository_test.go` | Extend | `UpdateStatusTx` (3.3) |
| `internal/cli/commands/status_test.go` | Extend | AC-14 formatter test |
| (INT-3, INT-4, INT-5) | Manual/integration | Documented as smoke tests |

---

## 5. Quality Gates

Before this feature advances from `ready_for_task_generation`:

- [ ] Every AC-01 through AC-16 has at least one explicit test case in this plan.
- [ ] Every test case traces to a spec.md acceptance criterion.
- [ ] Every new test file follows the repository-vs-mock split: repository tests use real DB; service and CLI tests use mocks.
- [ ] No test hardcodes status names from the basic or advanced profiles (`"completed"`, `"in_qa"`, etc.) except inside the config fixture string — all runtime status checks delegate to `workflow.Service`.
- [ ] `TestCascade_BasicProfile` and `TestCascade_AdvancedProfile` use the same test helper body with only the config fixture differing.
- [ ] Benchmark `BenchmarkCascade_BothLegs` is runnable via `make test` with `-bench` flag.
- [ ] All repository cleanup patterns follow the clean-before + defer-cleanup convention.

---

## 6. Traceability Summary

| Acceptance Criterion | Requirement | UAT Scenario | Test Case(s) |
|---------------------|-------------|--------------|--------------|
| AC-01 | REQ-F-001, REQ-F-004 | UAT-S1 | `TestCascade_TaskBackwardReopensFeature` |
| AC-02 | REQ-F-001 | UAT-S1 | `TestCascade_TaskBackwardReopensEpic` |
| AC-03 | REQ-F-002 | UAT-S1 | `TestCascade_FeatureBackwardReopensEpic` |
| AC-04 | REQ-F-008 | UAT-S3 | `TestCascade_NonTerminalFeatureContinuesToEpic` |
| AC-05 | REQ-F-008 | UAT-S3 | `TestCascade_AllAncestorsNonTerminalNoOp` |
| AC-06 | REQ-F-007 | UAT-S7 | `TestCascade_HistoryRowFormat`, `TestCascade_HistoryRowFormat_CreationTrigger` |
| AC-07 | REQ-F-004 (step 2) | UAT-S6-A | `TestResolveReopenTarget_FallbackAggregation` |
| AC-08 | REQ-F-004 (step 3) | UAT-S6-B | `TestResolveReopenTarget_FallbackInitial` |
| AC-09 | REQ-F-008 | UAT-S3 | `TestCascade_IdempotentOnSecondRegression` |
| AC-10 | REQ-F-003 | UAT-S2, INT-6 | Existing `TestTaskService_CreateTask_Reopen*` suite (regression) |
| AC-11 | REQ-F-006 | UAT-S5 | `TestCascade_BugDoesNotTriggerCascade`, `TestCascade_ChangeCardDoesNotTriggerCascade` |
| AC-12 | REQ-F-005 | UAT-S4 | `TestCascade_BasicProfile`, `TestCascade_AdvancedProfile` |
| AC-13 | REQ-N-003 | (non-blocking failure) | `TestCascade_TxFailureIsNonBlocking` (sub-tests A, B, C) |
| AC-14 | REQ-F-009 | UAT-S7 | `TestStatusHistoryFormatter_AutoReopenLabel` |
| AC-15 | REQ-N-001 | PERF-1 | `BenchmarkCascade_BothLegs`, `BenchmarkCascade_IdempotentPath` |
| AC-16 | REQ-N-007 | CI gate | `make fmt && make lint && make test` |
