---
feature_key: E27-F05-server-wiring-and-integration-end-to-end-assembly
epic_key: E27
title: Test Plan — Server Wiring and Integration (End-to-End Assembly)
phase: test_planning
---

# Test Plan: E27-F05 — Server Wiring and Integration

## 1. Scope

This plan covers quality assurance for the final assembly feature of E27. F05
is a pure wiring feature — no new business logic — so the test strategy
prioritises:

1. **Integration smoke tests** that prove F01-F04 compose correctly through a
   real HTTP round-trip.
2. **CORS unit tests** for the new `WithLocalCORS` middleware.
3. **`StartServer` unit tests** for the new importable entry point.
4. **Regression validation** that existing tests still pass and the CLI is
   unaffected.

Tests that are expected to already exist in prerequisite features (F01-F03
unit tests, F02 handler tests) are referenced but not duplicated here.

---

## 2. Acceptance Criteria Matrix

| AC ID | Acceptance Criterion (from spec) | Test Case(s) | Priority |
|---|---|---|---|
| REQ-F-001 | `cmd/server/main.go` calls `dbinit.Init` — no hardcoded `db.InitDB("shark-tasks.db")` | TC-001, TC-002 | High |
| REQ-F-002 | `ServiceContainer` gains `ViewerService` field; `WireServices()` constructs it | TC-003 | High |
| REQ-F-003 | Seven viewer routes registered on the mux and returning HTTP 200 | TC-004 | High |
| REQ-F-004 | `GET /` serves viewer.html with `Content-Type: text/html` | TC-005 | High |
| REQ-F-005 | `WithLocalCORS` echoes localhost/127.0.0.1 origins; blocks external | TC-006 through TC-011 | High |
| REQ-F-006 | `StartServer(ctx, opts)` closes `opts.Ready` before blocking `Serve` | TC-012 | High |
| REQ-F-006 | `StartServer` performs graceful shutdown on context cancellation | TC-013 | High |
| REQ-F-007 | `internal/viewer/server` is importable from CLI without circular import | TC-014 | High |
| REQ-F-008 | Existing CRUD endpoints unaffected | TC-015 | High |
| REQ-F-008 | `make test` passes (all existing tests continue to pass) | TC-016 | High |
| REQ-F-008 | `make lint` passes | TC-017 | Medium |
| REQ-F-008 | `make fmt` produces no diff | TC-017 | Medium |
| REQ-NF-001 | Server ready signal within 500ms of invocation | TC-018 | Medium |
| REQ-NF-002 | CORS locked to localhost/127.0.0.1 | TC-006 through TC-011 | High |
| REQ-NF-003 | 30-second graceful shutdown timeout preserved | TC-013 | Medium |

---

## 3. Test Cases

### 3.1 Integration Smoke Tests (`cmd/server/main_test.go`)

These tests are added to the existing `cmd/server/main_test.go` alongside the
current graceful-shutdown and otelhttp tests. They require a real temporary
SQLite DB (per spec §5.2 — allowed for integration tests that don't touch the
shared project DB).

---

**TC-001: Server starts with dbinit (no hardcoded path)**

**Story / AC:** REQ-F-001

**Priority:** High

**Preconditions:**
- `internal/dbinit` package from F01 is merged.
- `cmd/server/main.go` has been updated to call `server.StartServer`.

**Test Steps:**
1. Create a `t.TempDir()` directory and write a minimal `.sharkconfig.json`
   pointing to a local SQLite backend.
2. Construct `server.Options{Listener: <port-0 listener>, ProjectRoot: tempDir, Ready: readyCh}`.
3. Start `server.StartServer(ctx, opts)` in a goroutine.
4. Wait on `readyCh` (or timeout after 5s).

**Expected Results:**
- `readyCh` is closed without error — server started successfully.
- No `db.InitDB("shark-tasks.db")` call is made from `main.go` (verified by
  code inspection and by the fact that CWD does not have a `shark-tasks.db`
  after the test).

**Test Data:** `tempDir` from `t.TempDir()`

---

**TC-002: Server falls back correctly when no config present**

**Story / AC:** REQ-F-001 (auto-detection fallback)

**Priority:** Medium

**Preconditions:** Same as TC-001 but `Options.ProjectRoot` is explicitly set
to a directory with no `.sharkconfig.json` — triggers auto-detection fallback.

**Test Steps:**
1. Create a bare `t.TempDir()` (no config file).
2. Start server with `ProjectRoot: tempDir`.
3. Wait on `readyCh`.

**Expected Results:**
- Server starts (dbinit creates a fresh SQLite DB in the temp dir).
- `/health` returns 200.

---

**TC-003: All 7 viewer API endpoints return HTTP 200**

**Story / AC:** REQ-F-002, REQ-F-003

**Priority:** High

**Preconditions:**
- F01-F03 merged. Server started as in TC-001.

**Test Steps:**
After server is ready, issue `http.Get` to each viewer endpoint:

| Path | Expected Status |
|---|---|
| `GET /api/v1/viewer/summary` | 200 |
| `GET /api/v1/viewer/hierarchy` | 200 |
| `GET /api/v1/viewer/history/E01` | 200 |
| `GET /api/v1/viewer/file/E01-F01-001` | 200 |
| `GET /api/v1/viewer/features/E01-F01/tasks` | 200 |
| `GET /api/v1/viewer/recent-activity` | 200 |
| `GET /api/v1/viewer/workflow-meta` | 200 |

**Expected Results:**
- Each response status code is 200.
- Each response `Content-Type` header contains `application/json`.
- Response bodies are valid JSON (parseable without error).

**Notes:** An empty DB (no epics/features) is acceptable — responses will be
empty arrays/objects. The goal is route registration, not data correctness.

---

**TC-004: `GET /` serves viewer.html with HTML content**

**Story / AC:** REQ-F-004

**Priority:** High

**Preconditions:** F03 `internal/viewer/assets` merged. Server started as TC-001.

**Test Steps:**
1. `http.Get(base + "/")`

**Expected Results:**
- Status 200.
- `Content-Type` header contains `text/html`.
- Response body contains `<html` (case-insensitive).
- Response body does NOT contain `"Shark Task Manager API - Database Ready"`
  (the old placeholder is gone).

---

**TC-005: `GET /` returns non-empty HTML (SPA embedded correctly)**

**Story / AC:** REQ-F-004, REQ-NF (embedded asset, not CDN)

**Priority:** High

**Preconditions:** Same as TC-004.

**Test Steps:**
1. Read full body of `GET /`.
2. Verify no network request is needed to render (content is self-contained in
   response).

**Expected Results:**
- Body length > 0.
- Body contains a `<html` opening tag.
- No `src` or `href` pointing to external hosts (CDN).

---

### 3.2 CORS Tests (`internal/api/viewer/cors_test.go`)

These are unit tests for the `WithLocalCORS` middleware, using `httptest`.
They do NOT require a running server or database.

---

**TC-006: `localhost` origin echoed in ACAO header**

**Story / AC:** REQ-F-005

**Priority:** High

**Test Steps:**
1. Create a `httptest.NewRecorder()`.
2. Build a `GET` request with `Origin: http://localhost:5173`.
3. Pass through `WithLocalCORS(innerHandler)`.

**Expected Results:**
- `Access-Control-Allow-Origin` == `http://localhost:5173`.
- `Access-Control-Allow-Methods` contains `GET`.
- `Access-Control-Allow-Headers` contains `Content-Type`.
- `Vary` header contains `Origin`.
- Inner handler is called (response body present).

---

**TC-007: `127.0.0.1` origin echoed in ACAO header**

**Story / AC:** REQ-F-005

**Priority:** High

**Test Steps:**
1. Same as TC-006 but `Origin: http://127.0.0.1:7777`.

**Expected Results:**
- `Access-Control-Allow-Origin` == `http://127.0.0.1:7777`.
- CORS headers set as in TC-006.
- Inner handler called.

---

**TC-008: HTTPS localhost origin echoed**

**Story / AC:** REQ-F-005 (any scheme allowed)

**Priority:** Medium

**Test Steps:**
1. `Origin: https://localhost:9000`.

**Expected Results:**
- `Access-Control-Allow-Origin` == `https://localhost:9000`.

---

**TC-009: External origin blocked — no ACAO header**

**Story / AC:** REQ-F-005, REQ-NF-002

**Priority:** High

**Test Steps:**
1. `Origin: http://evil.example.com`.

**Expected Results:**
- `Access-Control-Allow-Origin` header is absent (empty string).
- Inner handler is still called (non-OPTIONS pass-through).
- Response status is whatever the inner handler sets (not blocked at
  middleware level).

---

**TC-010: Empty origin — no ACAO header**

**Story / AC:** REQ-F-005

**Priority:** Medium

**Test Steps:**
1. Request with no `Origin` header.

**Expected Results:**
- `Access-Control-Allow-Origin` header is absent.
- Inner handler is called.

---

**TC-011: OPTIONS preflight — 204 with correct headers for localhost origin**

**Story / AC:** REQ-F-005

**Priority:** High

**Test Steps:**
1. `OPTIONS` request with `Origin: http://localhost:5173`.

**Expected Results:**
- Status 204 (`http.StatusNoContent`).
- `Access-Control-Allow-Origin` == `http://localhost:5173`.
- `Access-Control-Allow-Methods` contains `GET, OPTIONS`.
- Inner handler is NOT called (middleware short-circuits).

---

**TC-011b: OPTIONS preflight for external origin — no ACAO, 204**

**Story / AC:** REQ-F-005

**Priority:** Medium

**Test Steps:**
1. `OPTIONS` request with `Origin: http://evil.example.com`.

**Expected Results:**
- Status 204 (OPTIONS always short-circuits).
- `Access-Control-Allow-Origin` header is absent.

---

**TC-011c: CRUD routes NOT wrapped by CORS middleware**

**Story / AC:** REQ-F-005 (scoped only to `/api/v1/viewer/`)

**Priority:** High

**Test Steps:**
1. Start server as in TC-001.
2. Issue `GET /api/v1/tasks` with `Origin: http://localhost:5173`.

**Expected Results:**
- `Access-Control-Allow-Origin` header is absent.
- Response status is 200 (list endpoint works).

---

### 3.3 `StartServer` Unit Tests (`internal/viewer/server/server_test.go`)

These tests verify the server entry-point lifecycle without touching the
viewer SPA or real project data.

---

**TC-012: `opts.Ready` closed before `Serve` blocks**

**Story / AC:** REQ-F-006

**Priority:** High

**Test Steps:**
1. Pre-bind `net.Listen("tcp", "127.0.0.1:0")`.
2. Create a `readyCh := make(chan struct{})`.
3. Create a cancelable context.
4. Start `server.StartServer(ctx, server.Options{Listener: lis, ProjectRoot: tempDir, Ready: readyCh})` in a goroutine.
5. Wait on `readyCh` with a 3-second timeout.

**Expected Results:**
- `readyCh` is closed within 3 seconds — server bound and serving before
  `readyCh` was closed.
- `http.Get` to the bound address succeeds immediately after `readyCh` is closed.

---

**TC-013: Context cancellation triggers graceful shutdown**

**Story / AC:** REQ-F-006, REQ-NF-003

**Priority:** High

**Test Steps:**
1. Start server as in TC-012.
2. Wait for `readyCh`.
3. Call `cancel()`.
4. Wait on the goroutine error channel with a 10-second timeout.

**Expected Results:**
- `StartServer` returns `nil` (graceful shutdown is not treated as an error).
- Return happens within 10 seconds.

---

**TC-014: Import path does not create circular dependency**

**Story / AC:** REQ-F-007

**Priority:** High

**Test Steps:**
1. Run `go build ./internal/cli/commands/...` (which imports `internal/viewer/server`).
2. Run `go vet ./...`.

**Expected Results:**
- Build succeeds with zero errors.
- No import cycle errors.

This is a compile-time test, not a runtime test. Failure would be a build
error, not a test failure.

---

**TC-014b: `StartServer` callable with pre-bound DB (F04 contract)**

**Story / AC:** REQ-F-006, REQ-F-007

**Priority:** High

**Preconditions:** A real `*repository.DB` from `dbinit.Init` on a tempdir.

**Test Steps:**
1. Call `dbinit.Init(ctx, dbinit.Options{ProjectRoot: tempDir})` to get `db`.
2. Pass `db` via `server.Options{DB: db, ...}`.
3. Start server, wait on `readyCh`.
4. Issue `GET /health`.

**Expected Results:**
- Server starts without re-initialising DB.
- `/health` returns 200.
- `db.Close()` called by the test (not by `StartServer`) after cancel —
  verifying `StartServer` does NOT close a caller-provided DB.

---

### 3.4 Regression Tests

---

**TC-015: Existing CRUD endpoints unchanged**

**Story / AC:** REQ-F-008

**Priority:** High

**Test Steps:**
1. Start server as in TC-001.
2. Issue the following requests:

| Path | Expected Status |
|---|---|
| `GET /health` | 200, body "OK" |
| `GET /api/v1/tasks` | 200 or 404 (empty DB) |
| `GET /api/v1/features` | 200 or 404 |
| `GET /api/v1/epics` | 200 or 404 |

**Expected Results:**
- All endpoints respond (not 500).
- CRUD routes do NOT have `Access-Control-Allow-Origin` even when
  `Origin: http://localhost:5173` is sent (CORS is viewer-only).

---

**TC-016: Full test suite passes (`make test`)**

**Story / AC:** REQ-F-008

**Priority:** High

**Test Steps:**
1. From project root: `make test`

**Expected Results:**
- Exit code 0.
- All pre-existing tests pass (no regressions from:
  - `cmd/server/main_test.go` graceful-shutdown tests
  - `internal/api/*_test.go`
  - `internal/repository/*_test.go`
  - `internal/services/*_test.go`
  - All other packages)
- New tests in `cors_test.go`, `server_test.go`, and `main_test.go` pass.

---

**TC-017: Lint and format gates**

**Story / AC:** REQ-F-008

**Priority:** Medium

**Test Steps:**
1. `make fmt` — verify no file diff produced.
2. `make lint` — verify exit code 0, no new warnings.

**Expected Results:**
- Both commands exit 0.
- No new golangci-lint findings in the new files.

---

### 3.5 Performance / Timing Test

---

**TC-018: Ready signal within 500ms**

**Story / AC:** REQ-NF-001

**Priority:** Medium

**Test Steps:**
1. Record `time.Now()` before starting `StartServer`.
2. Wait on `readyCh`.
3. Record elapsed duration.

**Expected Results:**
- Elapsed time < 500ms in a fresh tempdir (no migrations to run).

**Notes:** This is asserted in `server_test.go` alongside TC-012. It guards
against accidental slow-path regressions in `dbinit.Init`.

---

## 4. Test Files and Locations

| Test File | Test Cases | Notes |
|---|---|---|
| `cmd/server/main_test.go` | TC-001, TC-002, TC-003, TC-004, TC-005, TC-011c, TC-015, TC-016 | Extends existing file; integration tests using `t.TempDir()` DB |
| `internal/api/viewer/cors_test.go` | TC-006 through TC-011b | New file; `httptest` only, no DB |
| `internal/viewer/server/server_test.go` | TC-012, TC-013, TC-014b, TC-018 | New file; uses `net.Listen(":0")` + tempdir DB |
| Build/CI | TC-014, TC-016, TC-017 | `go build`, `make test`, `make lint` |

---

## 5. Out-of-Scope Tests

The following are explicitly NOT tested in this plan (owned by prerequisite features):

- `internal/api/viewer/handler_test.go` — F02 owns handler unit tests.
- `internal/services/viewer_service_test.go` — F02 owns service tests.
- `internal/viewer/assets` embed correctness — F03 owns asset tests.
- `internal/dbinit` correctness — F01 owns dbinit tests.
- Browser-launch behavior — F04 owns `shark web` command tests.

---

## 6. Test Infrastructure Notes

- **Real DB for integration tests:** `cmd/server/main_test.go` may use a real
  SQLite DB created in `t.TempDir()`. This is permitted per
  `.claude/rules/testing/architecture.md` for integration tests that do not
  touch the shared test DB.
- **No mocks in smoke tests:** The whole point of TC-001 through TC-015 is to
  prove F01-F04 compose end-to-end. Mocking any layer defeats the purpose.
- **`t.TempDir()` isolation:** Each test creates its own directory; tests are
  parallel-safe with respect to the database file.
- **Port 0:** All tests bind on `127.0.0.1:0` (OS-assigned port) to avoid
  port conflicts during `make test`.
- **Timeout discipline:** All `readyCh` waits must have a `time.After` guard
  to prevent indefinite hangs in CI.

---

## 7. Pass/Fail Criteria

| Gate | Requirement |
|---|---|
| All High-priority TCs | MUST pass before F05 is approved |
| All Medium-priority TCs | MUST pass before merge to main |
| No regressions | `make test` from repo root exits 0 |
| Lint clean | `make lint` exits 0 |
| Format clean | `make fmt` produces no diff |
| Smoke test completeness | TC-003 covers all 7 viewer endpoints |

---

*Generated: 2026-04-11 — QA Agent for E27-F05*
