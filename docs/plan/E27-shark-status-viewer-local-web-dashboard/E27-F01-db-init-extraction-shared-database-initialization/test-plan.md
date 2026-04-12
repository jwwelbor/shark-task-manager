---
feature_key: E27-F01
epic_key: E27
title: DB Init Extraction — Shared Database Initialization Package
phase: test_planning
---

# Test Plan: E27-F01 — DB Init Extraction

## 1. Summary

This plan covers the test strategy for extracting `internal/cli/db_init.go` logic
into the new shared package `internal/dbinit`. The refactor is a pure internal move
with no user-visible behaviour changes for CLI users; the HTTP server gains the
cloud-awareness it was previously missing. All existing tests must continue to pass
without modification. New tests validate the `dbinit` package in isolation.

**Risk level:** STANDARD — high regression risk due to foundational DB wiring touched
by virtually every CLI command, but scope is well-bounded (pure move of existing code).

---

## 2. Acceptance Criteria Test Matrix

| AC | Requirement | Test Cases | Validation Method |
|----|-------------|------------|-------------------|
| **AC-1** | `internal/dbinit/init.go` exports `Init`, `MustInit`, `Options` | TC-U-001 (compile), build check | `go build ./internal/dbinit/...` |
| **AC-2** | No direct `db.InitDB` calls from `cmd/` or `internal/cli/` | TC-REG-001 | `grep -rn "db.InitDB" cmd/ internal/cli/` returns zero hits outside `internal/db` itself |
| **AC-3** | `go build ./...` and `make test` pass on clean checkout | TC-REG-002 | `make build && make test` |
| **AC-4** | `make fmt && make lint && make test` is green | TC-REG-003 | CI quality gate |
| **AC-5** | `./bin/server` with Turso config hits Turso, not a local file | TC-INT-005 | Manual smoke test + absence of `shark-tasks.db` in cwd |
| **AC-6** | `./bin/shark list` identical before/after against local and Turso | TC-SMOK-001, TC-SMOK-003 | Manual smoke test output diff |
| **AC-7** | `internal/cli/db_init.go` ≤15 non-comment lines | TC-REG-004 | `grep -v '^\s*//' internal/cli/db_init.go \| wc -l` |
| **AC-8** | `internal/dbinit` does not import `internal/cli` | TC-U-002 | `go list -f '{{.Imports}}' ./internal/dbinit/...` |

---

## 3. Unit Tests — `internal/dbinit` Package

File: `internal/dbinit/init_test.go` and `internal/dbinit/project_root_test.go`

All unit tests use temp directories; no real DB credentials required.

### 3.1 `loadDatabaseConfig` — config parsing

| Test ID | Scenario | Setup | Expected Result |
|---------|----------|-------|-----------------|
| **TC-U-003** | Missing config file | Config path does not exist | Returns default local config: `Backend="sqlite"`, URL contains `shark-tasks.db` in project root |
| **TC-U-004** | Valid JSON, no `database` key | `{"workflow_config":"..."}` | Returns default local config (same as missing file) |
| **TC-U-005** | Local backend explicit | `{"database":{"backend":"local","url":"./mydb.db"}}` | Returns `Backend="local"`, URL matches provided value |
| **TC-U-006** | SQLite backend alias | `{"database":{"backend":"sqlite","url":"./mydb.db"}}` | Returns `Backend="sqlite"` |
| **TC-U-007** | Turso backend | `{"database":{"backend":"turso","url":"libsql://x.turso.io","auth_token_file":"./tok"}}` | Returns turso config with all fields populated |
| **TC-U-008** | Env-var expansion in URL | Config URL is `${HOME}/.shark/db` | Returned URL has `$HOME` expanded to actual value |
| **TC-U-009** | Env-var expansion in auth_token_file | `auth_token_file` is `${HOME}/.turso/token` | Returned AuthTokenFile is expanded |
| **TC-U-010** | Empty backend defaults to sqlite | `{"database":{"url":"./mydb.db"}}` (no backend key) | `Backend` defaults to `"sqlite"` |
| **TC-U-011** | Empty URL defaults to projectRoot/shark-tasks.db | `{"database":{"backend":"local"}}` (no url key) | URL becomes `<projectRoot>/shark-tasks.db` |

### 3.2 `resolveProjectRoot` — directory walk

File: `internal/dbinit/project_root_test.go`

| Test ID | Scenario | Setup | Expected Result |
|---------|----------|-------|-----------------|
| **TC-U-012** | `opts.ProjectRoot` set explicitly | `Options{ProjectRoot: "/tmp/proj"}` | Returns `/tmp/proj` without walking |
| **TC-U-013** | Walk finds `.sharkconfig.json` in parent | Nested temp dir; parent has `.sharkconfig.json` | Returns parent dir path |
| **TC-U-014** | Walk finds `shark-tasks.db` only (no config) | Nested temp dir; parent has `shark-tasks.db`, no config | Returns parent dir path |
| **TC-U-015** | Walk falls back to `.git/` | Nested temp dir; grandparent has `.git/` only | Returns grandparent dir path |
| **TC-U-016** | No markers found anywhere | Isolated temp dir with no markers | Returns non-nil error |
| **TC-U-017** | `.sharkconfig.json` wins over `.git/` (closer ancestor) | Parent has `.sharkconfig.json`; grandparent has `.git/` | Returns parent (sharkconfig takes priority) |

### 3.3 `Init` — local SQLite path

| Test ID | Scenario | Setup | Expected Result |
|---------|----------|-------|-----------------|
| **TC-U-018** | Happy path — explicit project root, local config | Temp dir with valid local config and `shark-tasks.db` | Returns non-nil `*repository.DB`, `db.Ping()` succeeds |
| **TC-U-019** | Happy path — no URL in config | Temp dir with config missing `url` field | Falls back to `<projectRoot>/shark-tasks.db`; DB opens successfully |
| **TC-U-020** | `sqlite` backend alias | Config has `backend: "sqlite"` | Opens successfully (same as `local`) |
| **TC-U-021** | Empty backend string | Config has no `backend` key | Treated as local; opens successfully |
| **TC-U-022** | DB path is relative | Config has `url: "./mydir/test.db"` | Path resolved relative to project root; opens successfully |
| **TC-U-023** | Unsupported backend | `{"database":{"backend":"postgres","url":"postgres://..."}}` | Returns error containing `"unsupported database backend"` |

### 3.4 `MustInit` behaviour

| Test ID | Scenario | Setup | Expected Result |
|---------|----------|-------|-----------------|
| **TC-U-024** | Panics on error | Options pointing to non-existent directory | `MustInit` call panics (use `require.Panics`) |
| **TC-U-025** | Returns DB on success | Valid local config in temp dir | Returns non-nil `*repository.DB` without panic |

### 3.5 No circular import

| Test ID | Scenario | Validation |
|---------|----------|------------|
| **TC-U-002** | `internal/dbinit` import list | `go list -f '{{.Imports}}' ./internal/dbinit/...` does NOT contain `internal/cli` |

---

## 4. Integration Tests — Turso Cloud Path

These tests require live Turso credentials and are gated behind a build tag
(`//go:build integration`) so they do not run in standard `make test`.

File: `internal/dbinit/init_integration_test.go`

```
Run command: go test -tags=integration ./internal/dbinit/... \
  -v -run TestInit_Turso
Environment: TEST_TURSO_URL, TEST_TURSO_TOKEN (or TEST_TURSO_TOKEN_FILE)
```

| Test ID | Scenario | Assertion |
|---------|----------|-----------|
| **TC-INT-001** | Turso with auth_token_file | Write token to temp file; set `auth_token_file` in config; call `Init` | DB opens, `Ping()` succeeds, schema is applied |
| **TC-INT-002** | Turso with `TURSO_AUTH_TOKEN` env var | Set env var; config has no `auth_token_file` | DB opens, `Ping()` succeeds |
| **TC-INT-003** | Turso with `skip_migrations: true` | Config has `skip_migrations: true`; schema already at current version | `Init` completes; `ApplySchemaIfNeeded` called (not `ApplySchemaAndMigrations`); log indicates fast-path taken |
| **TC-INT-004** | Turso — missing auth | No `auth_token_file`, no `TURSO_AUTH_TOKEN` env var | Init returns an error (driver rejects unauthenticated request) |
| **TC-INT-005** | Server uses Turso config, no local file created | `./bin/server` running with Turso config | `curl /health` returns `200 OK`; no `shark-tasks.db` created in CWD |

---

## 5. CLI Integration Tests — Verify Delegation

These tests verify that `internal/cli` correctly delegates to `dbinit` and that
the CLI wiring is preserved after the refactor.

### 5.1 `initDatabase` delegates correctly

**File:** `internal/cli/db_init_test.go` (new, or added to `db_global_test.go`)

| Test ID | Scenario | Assertion |
|---------|----------|-----------|
| **TC-CLI-001** | `initDatabase` returns valid DB via `dbinit.Init` | Set CWD to temp dir with local config; call `initDatabase(ctx)` | Returns `*repository.DB`, no error; `Ping()` succeeds |
| **TC-CLI-002** | `initDatabase` propagates `dbinit` errors | CWD with no project markers | Returns error; error message contains recognizable context |

### 5.2 `GetDB` singleton still works after refactor

These are the **existing** `db_global_test.go` tests — they must continue to pass
unchanged.

| Test ID | Existing Test | Must Pass Unchanged |
|---------|--------------|---------------------|
| **TC-CLI-003** | `TestGetDB_InitializesOnce` | Yes |
| **TC-CLI-004** | `TestGetDB_ReturnsCache` | Yes |
| **TC-CLI-005** | `TestResetDB_ClearsState` | Yes |

### 5.3 `db_helper.go` wrappers still work

These are the **existing** `db_helper_test.go` tests — they must continue to pass
unchanged.

| Test ID | Existing Test | Must Pass Unchanged |
|---------|--------------|---------------------|
| **TC-CLI-006** | `TestInitializeDatabase_LocalBackend` | Yes |
| **TC-CLI-007** | `TestInitializeDatabase_TursoBackend` | Yes (skipped if no creds) |
| **TC-CLI-008** | `GetDatabasePathForBackup` tests | Yes |

---

## 6. Server Integration Tests

File: `cmd/server/main_test.go` — existing tests must pass unchanged.

| Test ID | Scenario | Assertion |
|---------|----------|-----------|
| **TC-SRV-001** | Existing `main_test.go` tests | All existing server tests pass after `main.go` is updated to call `dbinit.Init` |
| **TC-SRV-002** | Server starts with local config (smoke) | Start server in test; call `/health` endpoint | HTTP 200, body is `"OK"` |

---

## 7. Regression Tests — Full Test Suite

These run as part of the standard quality gate (`make test`).

| Test ID | Scope | Command | Pass Criterion |
|---------|-------|---------|----------------|
| **TC-REG-001** | No stray `db.InitDB` calls outside `internal/db` | `grep -rn "db.InitDB" cmd/ internal/cli/` | Zero results |
| **TC-REG-002** | Clean build | `go build ./...` | Exit code 0 |
| **TC-REG-003** | Full quality gate | `make fmt && make lint && make test` | All green |
| **TC-REG-004** | `db_init.go` line count | `grep -v '^\s*//' internal/cli/db_init.go \| grep -v '^$' \| wc -l` | ≤15 |
| **TC-REG-005** | All repository tests pass | `go test ./internal/repository/...` | All green (uses test DB, no CLI dependency) |
| **TC-REG-006** | All CLI command tests pass | `go test ./internal/cli/...` | All green (mock-based, no real DB required) |
| **TC-REG-007** | All service tests pass | `go test ./internal/services/...` | All green |
| **TC-REG-008** | Coverage target met | `go test -cover ./internal/dbinit/...` | ≥80% line coverage |
| **TC-REG-009** | No import cycle introduced | `go build ./...` with `-gcflags='-e'` | No import cycle errors |

---

## 8. Manual Smoke Tests

Run after implementation is complete, before advancing to `ready_for_task_generation`.

| Test ID | Steps | Expected |
|---------|-------|----------|
| **TC-SMOK-001** | `./bin/shark list` from project root (local config) | Same output as before refactor; no new errors |
| **TC-SMOK-002** | `./bin/shark list` from `docs/plan/` subdirectory | Same output (project root auto-detection still works) |
| **TC-SMOK-003** | `./bin/shark list` with Turso-configured `.sharkconfig.json` | Connects to Turso; output matches local (or env not available: skip) |
| **TC-SMOK-004** | `./bin/server` with local config; `curl localhost:8080/health` | HTTP 200 `OK`; `slog` lines `"Database initialized successfully"` and `"Database integrity check passed"` present |
| **TC-SMOK-005** | `./bin/server` with Turso config; `curl localhost:8080/health` | HTTP 200 `OK`; no `shark-tasks.db` created in CWD |

---

## 9. Test Data & Fixtures

- **Local SQLite tests:** Use `t.TempDir()` to create isolated directories.
  Write a minimal `.sharkconfig.json` per test. No shared state between tests.
- **Turso tests:** Use `TEST_TURSO_URL` / `TEST_TURSO_TOKEN` env vars.
  If unset, tests are skipped via `t.Skip(...)`.
- **Project root walk tests:** Create temp directory trees using `os.MkdirAll`
  and `os.WriteFile`; no external dependencies.

---

## 10. Quality Gates

The feature is ready for task generation when:

- [ ] This test plan document exists at `docs/plan/.../test-plan.md`
- [ ] All test IDs TC-U-003 through TC-U-025 have corresponding test cases
      defined in `internal/dbinit/init_test.go` and `project_root_test.go`
- [ ] `make fmt && make lint && make test` passes (TC-REG-003)
- [ ] `internal/dbinit` coverage ≥80% (TC-REG-008)
- [ ] No existing test deleted (only additions or moves permitted)
- [ ] All regression tests TC-REG-001 through TC-REG-009 pass
- [ ] Manual smoke tests TC-SMOK-001 and TC-SMOK-004 completed

---

## 11. Risks & Mitigations

| Risk | ID | Impact | Mitigation |
|------|-----|--------|------------|
| Hidden behaviour in `cli.initDatabase` branches missed during move | R-2 | High | Side-by-side diff of old vs new; TC-CLI-001 and TC-CLI-002 exercise the delegate |
| `*repository.DB` accessor name collision with embedded `*sql.DB` | R-1 | Medium | Inspect struct before adding accessor; use `SQL()` or `Underlying()` if `DB()` collides |
| `FindProjectRoot` duplication drifts between `cli` and `dbinit` | R-3 | Low | Comment in both files pointing to the other; TC-U-013 through TC-U-017 guard `dbinit`'s copy |
| Turso driver type cast fails if driver registry changes | R-4 | Medium | Explicit type-assertion check with clear error; TC-INT-001 exercises the cast |
| `skip_migrations` fast-path not preserved | R-5 | Medium | TC-INT-003 asserts `ApplySchemaIfNeeded` is called (not `ApplySchemaAndMigrations`) |

---

*Last Updated: 2026-04-11*
