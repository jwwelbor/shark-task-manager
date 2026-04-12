---
feature_key: E27-F01-db-init-extraction-shared-database-initialization
epic_key: E27
title: DB Init Extraction - Shared Database Initialization Package
phase: specification
---

# Specification: DB Init Extraction — Shared Database Initialization Package

## 1. Overview

### What
Extract the cloud-aware database initialization logic currently embedded in
`internal/cli/db_init.go` into a new shared package `internal/dbinit`. The new
package exposes a small, entry-point-agnostic API (`Init` / `MustInit`) that
both `cmd/shark` (CLI) and `cmd/server` (HTTP viewer) can call to obtain a
`*repository.DB` that honours `.sharkconfig.json` settings (local SQLite vs
Turso cloud, auth-token loading, schema migration toggles, etc.).

### Why
`cmd/server/main.go` currently hardcodes `db.InitDB("shark-tasks.db")` and has
no awareness of:
- `.sharkconfig.json` discovery via project-root walk
- The `database.backend` field (local vs turso)
- Turso auth-token loading from `auth_token_file` or `TURSO_AUTH_TOKEN`
- The `skip_migrations` fast-path for cloud databases
- Env-var expansion in config fields

All of this logic already lives in `internal/cli/db_init.go` and
`internal/cli/db_helper.go`, but it lives in package `cli` — importing it from
`cmd/server` would pull in the entire Cobra command tree and create an
undesirable dependency direction (`cmd/server` → `internal/cli`).

E27 introduces a local web dashboard (`cmd/server`) that **must** work against
Turso-backed projects. This feature is the foundational refactor that unlocks
that capability. It is a pure refactor — no user-visible behaviour changes in
the CLI, and the server gains functionality it was previously missing.

### Scope
- **In scope:** create `internal/dbinit` package, move logic, update
  `internal/cli/db_init.go` to delegate, update `cmd/server/main.go` to call
  `dbinit`, preserve all existing CLI behaviour, preserve all tests.
- **Out of scope:** changes to the `internal/db` package itself, changes to
  `repository.DB` semantics, new CLI flags, new config fields, any
  E27-F02+ viewer functionality.

---

## 2. Requirements

### Functional Requirements

**REQ-F-001: Shared Init API**
The new package `internal/dbinit` MUST expose at least:
```go
func Init(ctx context.Context, opts Options) (*repository.DB, error)
func MustInit(ctx context.Context, opts Options) *repository.DB
```
- `Init` returns errors (used by HTTP server for graceful startup failure)
- `MustInit` panics on error (used by CLI for fail-fast entry points)

**REQ-F-002: Options struct**
`dbinit.Options` MUST support:
- `ProjectRoot string` — explicit project root; if empty, `Init` walks up from
  the current working directory using the same algorithm as
  `cli.FindProjectRoot` (looking for `.sharkconfig.json`, then
  `shark-tasks.db`, then `.git/`).
- `ConfigPath string` — optional override for the config file path; if empty,
  defaults to `filepath.Join(projectRoot, ".sharkconfig.json")`.

**REQ-F-003: Backend selection**
`Init` MUST read `.sharkconfig.json` and:
- For backend `sqlite`, `local`, or empty string → call `db.InitDB(dbPath)`
  and wrap in `repository.NewDB`. Defaults `dbPath` to
  `<projectRoot>/shark-tasks.db` when config URL is empty.
- For backend `turso` → call into the driver registry
  (`db.InitDatabase` via `internal/cli/db_helper.go` logic), extract the
  underlying `*sql.DB` from `*db.TursoDriver`, apply schema/migrations
  (honouring `skip_migrations`), and wrap in `repository.NewDB`.
- For any other backend → return `fmt.Errorf("unsupported database backend: %q", backend)`.

**REQ-F-004: Turso auth-token loading**
When backend is `turso`, `Init` MUST:
1. If `auth_token_file` is set in config, read the token via
   `db.LoadAuthToken` and build the connection string via
   `db.BuildTursoConnectionString`.
2. Otherwise, if `TURSO_AUTH_TOKEN` env var is set, use it.
3. Otherwise, proceed with the URL as-is (letting the driver fail with a
   meaningful error).

**REQ-F-005: Migration fast-path**
When `skip_migrations: true` is set in the database config, `Init` MUST call
`db.ApplySchemaIfNeeded` (which short-circuits on matching schema version)
instead of the unconditional `db.ApplySchemaAndMigrations`. This is required
to preserve the ~2s startup cost savings for Turso-backed CLIs.

**REQ-F-006: CLI behaviour preservation**
After this refactor, `internal/cli/db_init.go` MUST:
- Continue to expose `initDatabase(ctx context.Context) (*repository.DB, error)`
  with the same signature so that `internal/cli/db_global.go`'s `GetDB` call
  continues to compile without change.
- Be a thin delegate to `dbinit.Init` that wires in
  `cli.FindProjectRoot` as the project-root provider.
- Produce identical behaviour for all current CLI tests
  (`db_helper_test.go`, `db_global_test.go`).

**REQ-F-007: Server behaviour change**
`cmd/server/main.go` MUST:
- Replace `db.InitDB("shark-tasks.db")` + `repository.NewDB(database)` with a
  single call to `dbinit.Init(ctx, dbinit.Options{})`.
- Use the returned `*repository.DB` directly for `WireServices`.
- Preserve the integrity-check log lines and `defer database.Close()` semantics
  (on the `*repository.DB`, which exposes `Close` via its embedded `*sql.DB`).

**REQ-F-008: No circular imports**
The new `internal/dbinit` package MUST NOT import:
- `internal/cli` (otherwise `cmd/server` → `internal/dbinit` → `internal/cli`
  pulls the Cobra tree back in).
- `internal/services`, `internal/repository/*` beyond `repository.NewDB` and
  the `*repository.DB` type.

It MAY import:
- `internal/db` (all of it — drivers, config, schema, migrations)
- `internal/config` (for `Manager.Load`)
- `internal/repository` (for `NewDB` and the `DB` type only)

### Non-Functional Requirements

**REQ-NF-001: Zero regression in startup time**
Local SQLite init time MUST remain within 5% of pre-refactor measurements. The
`skip_migrations` fast-path for Turso MUST continue to skip DDL when the
schema version matches.

**REQ-NF-002: Test coverage**
The new `internal/dbinit` package MUST reach ≥80% line coverage via unit
tests. All existing tests in `internal/cli` and `cmd/server` MUST pass
unchanged (no test deletions permitted — only additions or moves).

**REQ-NF-003: Logging parity**
`cmd/server/main.go` MUST continue to emit the existing
`"Database initialized successfully"` and
`"Database integrity check passed"` `slog.Info` lines, so deployment log
scrapers are not broken.

### Acceptance Criteria

- **AC-1:** `internal/dbinit/init.go` exists and exports `Init`, `MustInit`,
  and `Options` with the signatures defined above.
- **AC-2:** `grep -rn "db.InitDB" cmd/ internal/cli/` returns only occurrences
  inside `dbinit` and `internal/db` itself (no direct hardcoded calls from
  entry points or CLI commands).
- **AC-3:** Running `go build ./...` and `make test` both pass on a clean
  checkout with no source changes beyond this feature.
- **AC-4:** `make fmt && make lint && make test` is green.
- **AC-5:** Running `./bin/server` against a Turso-configured
  `.sharkconfig.json` connects to Turso (not to a local `shark-tasks.db`),
  verified by the `/health` endpoint and by the absence of a new local
  `shark-tasks.db` file in `cwd`.
- **AC-6:** Running `./bin/shark list` behaves identically before and after
  the refactor against both a local SQLite project and a Turso project.
- **AC-7:** `internal/cli/db_init.go` is ≤15 lines of non-comment code and
  contains only a delegate call.
- **AC-8:** `internal/dbinit` has no import of `internal/cli`.

---

## 3. Architecture

### Package Layout
```
internal/
├── dbinit/                    # NEW
│   ├── init.go                # Init, MustInit, Options, resolveConfig
│   ├── project_root.go        # FindProjectRoot (moved or duplicated from cli/root.go)
│   └── init_test.go           # Unit tests with table-driven scenarios
├── cli/
│   ├── db_init.go             # REDUCED to delegate
│   ├── db_global.go           # unchanged
│   ├── db_helper.go           # GetDatabasePathForBackup stays; Turso helpers moved
│   └── root.go                # FindProjectRoot either stays and is called by
│                              # dbinit via a function option, or is moved
│                              # to dbinit and re-exported by cli.
cmd/
├── server/
│   └── main.go                # calls dbinit.Init instead of db.InitDB
└── shark/
    └── main.go                # no change (CLI uses cli.GetDB → initDatabase → dbinit.Init)
```

### Public API of `internal/dbinit`

```go
package dbinit

import (
    "context"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

// Options configures how Init locates and opens the database.
type Options struct {
    // ProjectRoot is the project root directory. If empty, Init walks up
    // from the current working directory looking for .sharkconfig.json,
    // shark-tasks.db, or .git.
    ProjectRoot string

    // ConfigPath is the path to the shark config file. If empty, defaults
    // to <ProjectRoot>/.sharkconfig.json.
    ConfigPath string
}

// Init opens the database according to .sharkconfig.json and returns a
// fully initialized *repository.DB. Works for local SQLite and Turso cloud
// backends, honouring auth_token_file, TURSO_AUTH_TOKEN, and skip_migrations.
func Init(ctx context.Context, opts Options) (*repository.DB, error)

// MustInit is a fail-fast variant that panics on error. Intended for CLI
// entry points where startup failure is unrecoverable.
func MustInit(ctx context.Context, opts Options) *repository.DB
```

### Internal Helpers (unexported)

```go
// resolveProjectRoot returns opts.ProjectRoot if set, otherwise walks up
// from cwd looking for .sharkconfig.json, shark-tasks.db, or .git/.
func resolveProjectRoot(opts Options) (string, error)

// resolveConfigPath returns opts.ConfigPath if set, otherwise
// filepath.Join(projectRoot, ".sharkconfig.json").
func resolveConfigPath(opts Options, projectRoot string) string

// loadDatabaseConfig reads .sharkconfig.json and returns the parsed
// db.DatabaseConfig with env-var expansion applied.
// (This is the body of cli.GetDatabaseConfig, moved here.)
func loadDatabaseConfig(configPath string) (db.DatabaseConfig, error)

// initLocal opens a local SQLite database.
func initLocal(cfg db.DatabaseConfig, projectRoot string) (*repository.DB, error)

// initTurso opens a Turso database, loads auth token, and applies schema.
func initTurso(ctx context.Context, cfg db.DatabaseConfig) (*repository.DB, error)
```

### Delegate in `internal/cli/db_init.go` (after refactor)

```go
package cli

import (
    "context"
    "github.com/jwwelbor/shark-task-manager/internal/dbinit"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

// initDatabase is retained for backwards-compat with db_global.go's GetDB.
// It is now a thin delegate to dbinit.Init using cli-specific project-root
// discovery.
func initDatabase(ctx context.Context) (*repository.DB, error) {
    return dbinit.Init(ctx, dbinit.Options{})
}
```

### Updated `cmd/server/main.go` (key fragment)

```go
repoDB, err := dbinit.Init(context.Background(), dbinit.Options{})
if err != nil {
    slog.Error("Failed to initialize database", "error", err)
    os.Exit(1)
}
defer repoDB.Close()

slog.Info("Database initialized successfully")

// Integrity check on the underlying *sql.DB
if err := db.CheckIntegrity(repoDB.DB()); err != nil {
    slog.Error("Database integrity check failed", "error", err)
    os.Exit(1)
}
slog.Info("Database integrity check passed")

svcs := WireServices(repoDB, ".")
```

> **Note:** If `*repository.DB` does not currently expose an underlying
> `*sql.DB` accessor, we add one (`DB() *sql.DB`) as part of this feature.
> This is additive and does not affect existing callers.

### Integration Points

| Component | Change |
|---|---|
| `internal/dbinit/init.go` | NEW — owns the logic |
| `internal/cli/db_init.go` | reduced to ~10 lines, delegates to `dbinit.Init` |
| `internal/cli/db_helper.go` | `GetDatabaseConfig` / `InitializeDatabaseFromConfig` / `GetDatabasePathForBackup` either stay (as thin wrappers) or are re-exported from `dbinit` for backup command use |
| `internal/cli/db_global.go` | unchanged — `GetDB` still calls `initDatabase` |
| `cmd/server/main.go` | swap `db.InitDB` → `dbinit.Init`, use `repoDB` directly |
| `cmd/server/services.go` | no change — still receives `*repository.DB` |
| `internal/cli/root.go` | `FindProjectRoot` either stays and is called via a `dbinit` option, or is moved/duplicated into `dbinit`. Decision: **duplicate the small function into `internal/dbinit`** to keep `dbinit` free of `internal/cli` imports. `internal/cli/root.go` keeps its own copy for CLI's own uses (path normalization, config file discovery, etc.). The function is ~20 lines and duplication is cheaper than another shared package. |
| `internal/cli/db_helper_test.go` | tests that still pertain to `GetDatabasePathForBackup` stay; tests covering `GetDatabaseConfig` / `InitializeDatabaseFromConfig` either stay (wrappers preserved) or are moved to `internal/dbinit/init_test.go` |

### Error Flow

```
dbinit.Init()
├── resolveProjectRoot()                        → wraps: "dbinit: project root: %w"
├── loadDatabaseConfig()                         → wraps: "dbinit: load config: %w"
├── (local) db.InitDB()                          → wraps: "dbinit: init local db: %w"
│   └── repository.NewDB()
└── (turso) db.InitDatabase() → TursoDriver
    ├── LoadAuthToken()                          → wraps: "dbinit: load auth token: %w"
    ├── BuildTursoConnectionString()
    ├── tursoDriver.GetSQLDB()                   → wraps: "dbinit: get sql.DB: %w"
    ├── ApplySchemaIfNeeded or ApplySchemaAndMigrations → wraps: "dbinit: apply schema: %w"
    └── repository.NewDB()
```

All error messages prefixed with `"dbinit: "` for easy grep in logs.

### Concurrency & Lifecycle

- `Init` is safe to call concurrently, but each call produces a fresh
  `*repository.DB`. Callers that want a singleton (like the CLI's
  `db_global.go`) must handle memoization themselves — this preserves the
  current `sync.Once` pattern.
- `MustInit` simply calls `Init` and panics on error; it is NOT a singleton.
- The server creates exactly one `*repository.DB` at startup and reuses it
  for all HTTP requests (via `WireServices`).

---

## 4. Implementation Plan

Ordered so each step leaves `make test` green.

1. **Create package skeleton**
   - Create `internal/dbinit/init.go` with empty `Init`, `MustInit`,
     `Options` declarations and a package doc comment.
   - Create `internal/dbinit/project_root.go` with a private `findProjectRoot`
     that duplicates `cli.FindProjectRoot` logic.
   - Run `go build ./...` to confirm compile.

2. **Move config loading**
   - Copy the body of `cli.GetDatabaseConfig` into an unexported
     `loadDatabaseConfig` in `internal/dbinit/init.go` (keeping env-var
     expansion, default fallbacks, all branches).
   - Add table-driven test `TestLoadDatabaseConfig` covering:
     missing config file, missing `database` section, local backend, turso
     backend, env-var expansion.

3. **Implement local init path**
   - Implement `initLocal(cfg, projectRoot) (*repository.DB, error)` that
     mirrors the `if backend == sqlite|local|""` branch of current
     `cli.initDatabase`.
   - Add `TestInit_LocalSQLite` using a temp dir with a minimal
     `.sharkconfig.json` + empty `shark-tasks.db`.

4. **Implement Turso init path**
   - Implement `initTurso(ctx, cfg) (*repository.DB, error)` mirroring the
     turso branch: `LoadAuthToken`, `BuildTursoConnectionString`,
     `db.InitDatabase`, `TursoDriver.GetSQLDB`, `ApplySchemaIfNeeded` vs
     `ApplySchemaAndMigrations`, `repository.NewDB`.
   - Add `TestInit_Turso_AuthTokenFile` and `TestInit_Turso_EnvVar` using
     mock Turso endpoints OR flagging these as build-tag-gated integration
     tests that require `TURSO_AUTH_TOKEN`.

5. **Wire up `Init` and `MustInit`**
   - `Init` resolves project root + config path, calls `loadDatabaseConfig`,
     dispatches to `initLocal` or `initTurso`, wraps errors with
     `"dbinit: "` prefix.
   - `MustInit` wraps `Init` with panic-on-error.

6. **Add `*sql.DB` accessor to `repository.DB` (if missing)**
   - Grep for an existing accessor. If none, add
     `func (d *DB) DB() *sql.DB { return d.DB }` or equivalent — name
     chosen to avoid collision with the embedded field.
   - Update `cmd/server/main.go` to use it for `db.CheckIntegrity`.

7. **Collapse `internal/cli/db_init.go`**
   - Replace the entire function body with the delegate:
     `return dbinit.Init(ctx, dbinit.Options{})`.
   - Remove now-unused imports from `internal/cli/db_init.go`.
   - Run `go test ./internal/cli/...` — all tests should still pass.

8. **Update `cmd/server/main.go`**
   - Replace `db.InitDB("shark-tasks.db")` + `repository.NewDB(database)`
     with `dbinit.Init(context.Background(), dbinit.Options{})`.
   - Update `db.CheckIntegrity` call to use `repoDB.DB()`.
   - Update `defer` to close via `repoDB`.
   - Preserve all `slog` lines verbatim.

9. **Decide on `cli.GetDatabaseConfig` / `cli.InitializeDatabaseFromConfig`**
   - Option A (preferred): keep them as **thin wrappers** that call into
     `dbinit` internals (exported as `dbinit.LoadDatabaseConfig` if needed
     by `GetDatabasePathForBackup`). This preserves
     `cloud.go`'s `shark cloud status` command untouched.
   - Option B: delete the wrappers and update all call sites.
   - **Recommendation:** Option A, to minimize blast radius.

10. **Run the quality gate**
    - `make fmt && make lint && make test`.
    - Manually smoke-test `./bin/shark list` (local) and `./bin/server`
      against a Turso config (or a local config if Turso unavailable).

11. **Update feature docs and open questions**
    - Add a note to `feature.md` linking to this spec.
    - Record any deferred decisions (e.g. whether to move
      `FindProjectRoot` entirely into `dbinit` in a follow-up) as shark
      feature notes.

### Files Touched

| Path | Action |
|---|---|
| `internal/dbinit/init.go` | CREATE |
| `internal/dbinit/project_root.go` | CREATE |
| `internal/dbinit/init_test.go` | CREATE |
| `internal/dbinit/project_root_test.go` | CREATE |
| `internal/cli/db_init.go` | MODIFY (reduce to delegate) |
| `internal/cli/db_helper.go` | MODIFY (thin wrappers that forward to `dbinit`) |
| `internal/cli/db_helper_test.go` | UNCHANGED (tests wrappers which still work) |
| `internal/repository/db.go` (or equivalent) | MODIFY if no `*sql.DB` accessor exists |
| `cmd/server/main.go` | MODIFY (use `dbinit.Init`) |
| `cmd/server/main_test.go` | MODIFY only if it mocks `db.InitDB` directly |

---

## 5. Testing Strategy

### Unit Tests (new, in `internal/dbinit`)

| Test | Scenario | Assertion |
|---|---|---|
| `TestLoadDatabaseConfig_MissingFile` | Config path doesn't exist | Returns default local config with `shark-tasks.db` in project root |
| `TestLoadDatabaseConfig_NoDatabaseSection` | Valid JSON but no `database` key | Returns default local config |
| `TestLoadDatabaseConfig_LocalBackend` | `{"database":{"backend":"local","url":"./foo.db"}}` | Returns local config with expanded path |
| `TestLoadDatabaseConfig_TursoBackend` | `{"database":{"backend":"turso","url":"libsql://...","auth_token_file":"..."}}` | Returns turso config with all fields parsed |
| `TestLoadDatabaseConfig_EnvVarExpansion` | URL contains `${HOME}` | Returns expanded URL |
| `TestResolveProjectRoot_FromOptions` | `opts.ProjectRoot` set | Returns it unchanged |
| `TestResolveProjectRoot_WalkUp_SharkConfig` | Nested dir, parent has `.sharkconfig.json` | Returns parent dir |
| `TestResolveProjectRoot_WalkUp_DBFile` | Nested dir, parent has `shark-tasks.db` only | Returns parent dir |
| `TestResolveProjectRoot_WalkUp_GitFallback` | Nested dir, only `.git/` exists | Returns git root |
| `TestResolveProjectRoot_NotFound` | No markers anywhere | Returns error |
| `TestInit_LocalSQLite_HappyPath` | Temp dir with valid local config | Returns non-nil `*repository.DB`, can query schema |
| `TestInit_LocalSQLite_DefaultURL` | Config with no URL field | Falls back to `<projectRoot>/shark-tasks.db` |
| `TestInit_UnsupportedBackend` | `{"database":{"backend":"postgres"}}` | Returns error containing `"unsupported database backend"` |
| `TestMustInit_PanicsOnError` | Invalid config | Calling `MustInit` panics |

### Integration Tests

- **`TestInit_Turso_AuthTokenFile`** (build tag `integration` + requires
  `TURSO_TEST_URL` and a test token file) — verifies full Turso path
  end-to-end.
- **`TestInit_Turso_SkipMigrationsFastPath`** — asserts schema is NOT
  re-applied when `skip_migrations: true` and version matches.
- Reuse existing `internal/db/init_db_integration_test.go` — no changes
  required.

### Regression Tests (must keep passing unchanged)

- `internal/cli/db_helper_test.go` — covers `GetDatabaseConfig`,
  `InitializeDatabaseFromConfig`, `GetDatabasePathForBackup`.
- `internal/cli/db_global_test.go` — covers `GetDB`, `CloseDB`, `ResetDB`.
- `cmd/server/main_test.go` — covers server startup.

### Manual Smoke Tests

1. **Local CLI:** `./bin/shark list` from project root and from
   `docs/plan/` subdir — both should work.
2. **Local server:** `./bin/server &` then `curl localhost:8080/health` —
   should return `OK`.
3. **Turso CLI:** point `.sharkconfig.json` at a Turso project, run
   `./bin/shark list` — should hit Turso (verify via Turso dashboard or
   network trace).
4. **Turso server:** same config, run `./bin/server`, hit `/health` —
   should return `OK` and NOT create a local `shark-tasks.db`.

### Coverage Target
- `internal/dbinit`: ≥80% line coverage.
- No regression in overall project coverage (`make test-coverage`).

---

## 6. Dependencies

### Blocks (what depends on this)
- **E27-F02+** — every subsequent E27 feature (HTTP viewer endpoints,
  real-time updates, observability wiring, etc.) requires the server to
  read the same `.sharkconfig.json` as the CLI, so they all sit on top of
  this refactor.

### Blocked By
- Nothing. This is the foundation feature of E27. It has no hard external
  dependencies.

### Code Dependencies (read-only)
- `internal/db` — driver registry, `InitDB`, `InitDatabase`,
  `ApplySchemaAndMigrations`, `ApplySchemaIfNeeded`, `LoadAuthToken`,
  `BuildTursoConnectionString`, `TursoDriver`, `CheckIntegrity`,
  `DatabaseConfig`.
- `internal/config` — `Manager.Load` for parsing `.sharkconfig.json`.
- `internal/repository` — `DB` type and `NewDB` constructor (plus new
  `DB()` accessor if added).

### Risks
- **R-1: `*repository.DB` internals** — if `repository.DB` embeds `*sql.DB`
  as an unnamed field, adding a method named `DB()` collides with the
  embedded promoted field access. Mitigation: inspect the struct first; if
  collision exists, name the accessor `SQL()` or `Underlying()`.
- **R-2: Hidden behaviour in `cli.initDatabase`** — any subtle branch we
  miss causes a CLI regression. Mitigation: side-by-side diff of old vs
  new code paths + running full `make test` after step 7.
- **R-3: `FindProjectRoot` duplication drift** — if the CLI version and
  dbinit version diverge over time, behaviour becomes inconsistent.
  Mitigation: add a comment in both files pointing to the other; consider
  a follow-up task to fully consolidate in a shared package after E27
  ships.
- **R-4: Turso driver extraction** — the current code casts
  `database.(*db.TursoDriver)` and calls `GetSQLDB`. If the driver
  registry is later refactored to return a different type, this cast
  fails. Mitigation: add an explicit type-assertion check with a clear
  error message (already present in the existing code); cover with a unit
  test.

---

## Open Questions

1. Should `FindProjectRoot` be moved into `internal/dbinit` and re-exported
   by `internal/cli`, or duplicated? Current recommendation: **duplicate**
   for simplicity; revisit if a third consumer appears.
2. Should the CLI wrappers in `internal/cli/db_helper.go` be kept or
   deleted? Current recommendation: **keep as thin forwarders** to avoid
   touching `cloud.go` and other callers in this feature.
3. Should we add an `Options.Backend` override for tests that want to force
   a backend regardless of config? Current recommendation: **no** — keep
   the API minimal; tests can write temp config files instead.

---

*Last Updated*: 2026-04-11
