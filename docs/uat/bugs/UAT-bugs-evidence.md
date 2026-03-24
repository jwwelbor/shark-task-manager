# UAT Evidence — Bug Verification Batch

**Date:** 2026-03-23
**Bugs:** B002, B003, B004, B005, B006

---

## B002: Unstable Turso client dependency (v0.0.0 dev commit)

### Spec

**Bug description:** The `github.com/tursodatabase/libsql-client-go` package is pinned to a pseudo-version (`v0.0.0-20251219100830-236aa1ff8acc`) because the upstream repository has never published a stable release tag.

**Expected behavior:** The Turso Cloud dependency should use a published stable semver tag.

**Fix type:** Investigation/Documentation — no code fix possible due to upstream limitations.

### Implementation

**Commit:** `3bea989` — docs(B002): document unstable Turso client dependency investigation

**File:** `docs/plan/bugs/B002.md` — 100+ lines of investigation findings added

**go.mod line 11:**
```
github.com/tursodatabase/libsql-client-go v0.0.0-20251219100830-236aa1ff8acc
```

**turso_driver.go (line 8):**
```go
_ "github.com/tursodatabase/libsql-client-go/libsql"
```

### Investigation Findings

- `libsql-client-go` is **deprecated** — the commit we use IS the deprecation commit
- Never had stable release tags and never will
- Replacement options evaluated:
  - `turso-go` v0.2.2: Has stable tags but does NOT support Turso Cloud URLs
  - `go-libsql`: Supports Cloud but requires CGO, no stable tags
  - Current dep: Already at latest commit
- **Status: BLOCKED** — requires architectural decision

### Test Output

N/A — documentation-only change, no code modified.

---

## B003: Duplicate history tables (task_history + entity_history)

### Spec

**Bug description:** The test file `internal/repository/task_history_repository_test.go` was a near-identical copy of `internal/repository/task/history_test.go`. Both contained the same 8 test functions, causing redundant test execution.

**Expected behavior:** Test coverage for `TaskHistoryRepository` should not be duplicated across two files.

**Fix scope:** Removed duplicate test file only. The `task_history` table backward-compatibility shim is intentionally kept pending T-E21-F08-004.

### Implementation

**Commit:** `585e0fc` — fix(B003): remove duplicate task_history repository test file

**Deleted file:** `internal/repository/task_history_repository_test.go` (642 lines)

**Authoritative file preserved:** `internal/repository/task/history_test.go` (643 lines)

### Verification

8 test functions confirmed in authoritative file:
1. `TestTaskHistoryRepository_ListWithFilters`
2. `TestTaskHistoryRepository_ListWithFilters_EmptyResults`
3. `TestGetHistoryByTaskKey`
4. `TestGetHistoryByTaskKeyNotFound`
5. `TestGetHistoryByTaskKeyEmptyHistory`
6. `TestTaskHistoryRepository_CreateWithRejectionReason`
7. `TestTaskHistoryRepository_CreateWithoutRejectionReason`
8. `TestTaskHistoryRepository_GetRejectionHistoryForTask`

No duplicate test functions remain (grep verified).

### Test Output

```
PASS: github.com/jwwelbor/shark-task-manager/internal/repository/task (0.087s)
```

All 8 history tests pass. Full repository test suite passes with no regressions.

---

## B004: No graceful shutdown in HTTP server

### Spec

**Bug description:** The HTTP server used `http.ListenAndServe()` directly, which provides no mechanism for graceful shutdown. On SIGINT/SIGTERM, in-flight requests would be abruptly terminated.

**Expected behavior:** Server should handle SIGINT/SIGTERM signals, stop accepting new connections, allow in-flight requests to complete (with timeout), then exit cleanly.

### Implementation

**Commit:** `97e1fa9` — fix(server): add graceful shutdown on SIGINT/SIGTERM

**File:** `cmd/server/main.go`

**Key changes:**
- Lines 21-23: Added `shutdownTimeout = 30 * time.Second` constant
- Lines 81-95: Replaced `http.ListenAndServe()` with `http.Server` struct + goroutine
- Lines 98-107: Signal registration for `SIGINT` and `SIGTERM` via `signal.Notify()`
- Lines 110-118: Graceful shutdown with `srv.Shutdown(ctx)` and 30s timeout

**Before:**
```go
if err := http.ListenAndServe(":"+port, handler); err != nil {
```

**After:**
```go
srv := &http.Server{Addr: ":" + port, Handler: handler}
srvErr := make(chan error, 1)
go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        srvErr <- err
    }
}()
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
select {
case err := <-srvErr: slog.Error("Server failed to start", "error", err); os.Exit(1)
case sig := <-quit: slog.Info("Shutdown signal received", "signal", sig.String())
}
ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
    slog.Error("Server shutdown error", "error", err); os.Exit(1)
}
slog.Info("Server stopped gracefully")
```

### Tests

**File:** `cmd/server/main_test.go` (lines 83-222)

**Test 1:** `TestGracefulShutdown_StopsAcceptingNewConnections` — Verifies `srv.Shutdown()` stops listener and returns `http.ErrServerClosed`

**Test 2:** `TestGracefulShutdown_InFlightRequestsComplete` — Verifies slow in-flight request completes before shutdown returns (100ms simulated delay)

### Test Output

```
=== RUN   TestGracefulShutdown_StopsAcceptingNewConnections
--- PASS (0.00s)
=== RUN   TestGracefulShutdown_InFlightRequestsComplete
--- PASS (0.14s)
PASS ok github.com/jwwelbor/shark-task-manager/cmd/server 0.147s
```

---

## B005: Silently discarded error in feature progress recalculation

### Spec

**Bug description:** In `TaskService.recalculateFeatureProgress()`, the error from `RecalculateAndSetProgress()` was silently discarded with `_ = err`. Database failures during progress recalculation were completely invisible.

**Expected behavior:** Errors should be logged (as warnings, since progress recalculation is non-fatal) so operators can diagnose data integrity issues.

### Implementation

**File:** `internal/services/task_service.go` (lines 1023-1032)

**Before:**
```go
_ = s.featureService.RecalculateAndSetProgress(ctx, featureID)
```

**After:**
```go
if err := s.featureService.RecalculateAndSetProgress(ctx, featureID); err != nil {
    slog.Warn("feature progress recalculation failed after task status change",
        "feature_id", featureID, "error", err)
}
```

**Doc comment updated from:** "silently ignores errors" → "logs a warning on error"

### Tests

**File:** `internal/services/task_service_test.go` (lines 2629-2696)

**Test:** `TestTaskService_recalculateFeatureProgress_LogsErrorNotSilentlyDiscarded`
- Installs a custom `capturingSlogHandler` to capture log records
- Simulates progress recalculation failure via mock repo
- Verifies `slog.Warn` is called with `"error"` attribute
- Confirms `TransitionStatus` still succeeds (non-fatal)
- Assertion message explicitly references B005

### Test Output

Test passes. TransitionStatus succeeds despite progress recalculation failure, and the error is captured in slog output.

---

## B006: Silently discarded error in rejection note creation

### Spec

**Bug description:** In `EntityService.TransitionStatus()`, the error from `CreateRejectionNote()` was silently discarded with `_, _ = ...`. Database failures during rejection note creation were invisible.

**Expected behavior:** Errors should be logged (non-blocking, matching the `recordEntityHistory` pattern) so operators can diagnose issues.

### Implementation

**Commit:** `53879a2` — fix(B006): log rejection note creation errors instead of silently discarding

**File:** `internal/services/entity_service.go` (lines 221-232)

**Before:**
```go
_, _ = s.noteRepo.CreateRejectionNote(ctx, entityType, entity.GetID(),
    0, currentStatus, targetStatus, opts.Reason, agent, docPath)
```

**After:**
```go
if _, err := s.noteRepo.CreateRejectionNote(ctx, entityType, entity.GetID(),
    0, currentStatus, targetStatus, opts.Reason, agent, docPath); err != nil {
    slog.Warn("failed to create rejection note", "entity_type", entityType, "entity_id", entity.GetID(), "error", err)
}
```

**Comment added:** "Non-blocking: errors are logged but not propagated (matches recordEntityHistory pattern)."

### Tests

**File:** `internal/services/entity_service_test.go` (lines 932-1050)

**Test 1:** `TestEntityService_RejectionNote_ErrorIsNotSilentlyDiscarded` (regression test for B006)
- Uses `capturingSlogHandler` to capture log records
- Simulates rejection note creation failure
- Verifies `slog.LevelWarn` log with `"error"` attribute
- Confirms transition still succeeds

**Test 2:** `TestEntityService_RejectionNote_SuccessDoesNotAffectTransition` (happy path)
- Verifies successful note creation doesn't affect transition
- Confirms note creator is called once

### Test Output

```
TestEntityService_RejectionNote_ErrorIsNotSilentlyDiscarded — PASS
TestEntityService_RejectionNote_SuccessDoesNotAffectTransition — PASS
All 183 service tests pass.
```
