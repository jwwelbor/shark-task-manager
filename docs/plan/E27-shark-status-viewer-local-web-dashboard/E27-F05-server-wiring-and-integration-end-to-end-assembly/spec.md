---
feature_key: E27-F05-server-wiring-and-integration-end-to-end-assembly
epic_key: E27
title: Server Wiring and Integration - End-to-End Assembly
phase: specification
---

# Specification: Server Wiring and Integration — End-to-End Assembly

## 1. Overview

### What

E27-F05 is the **final integration feature** of the E27 Status Viewer epic. It
assembles the four component features (F01 dbinit, F02 viewer API, F03 SPA,
F04 `shark web` CLI) into a single working binary. No new business logic is
introduced. The work is pure wiring:

1. Replace the hardcoded `db.InitDB("shark-tasks.db")` in `cmd/server/main.go`
   with the cloud-aware `dbinit.Init(ctx, dbinit.Options{})` produced by F01.
2. Add a `ViewerService` field to `cmd/server/services.ServiceContainer` and
   construct it inside `WireServices()` from the repositories already wired
   there.
3. Register the seven viewer routes from F02 on the existing `net/http.ServeMux`
   using `viewer.NewHandler(svcs.ViewerService).RegisterRoutes(mux)`.
4. Replace the placeholder `GET /` stub at `cmd/server/main.go` lines 67-69
   with the embedded SPA from F03 (`internal/viewer/assets/viewer.html`) and
   mount the vendor JS under `GET /static/`.
5. Add a `withLocalCORS` middleware scoped to `/api/v1/viewer/*` that echoes
   only `http://localhost:*` and `http://127.0.0.1:*` origins (ADR-E27-007).
6. Extract an importable `StartServer(ctx, opts) error` entry point so that
   `shark web` (F04) can start the server in-process — `cmd/server/main.go`
   becomes a thin wrapper around the same entry point that the CLI uses.

### Why

F01-F04 are each individually landable, but none of them alone delivers the
user-visible feature ("run `shark web`, see a dashboard"). This feature is the
merge seam. It touches exactly the two server entry-point files plus two small
new files — no repository, service, handler, or SPA code is modified here.

### Scope

**In scope:**
- `cmd/server/main.go` modifications (DB init, route registration, SPA mount,
  CORS wrapping, extraction of `StartServer`).
- `cmd/server/services.go` — add `ViewerService` to `ServiceContainer` and
  construct it in `WireServices()`.
- NEW `internal/api/viewer/cors.go` — `withLocalCORS` middleware.
- NEW `internal/viewer/server/server.go` — `StartServer(ctx, Options) error`
  entry point shared by `cmd/server/main.go` and `shark web`.
- Integration smoke tests in `cmd/server/main_test.go` (HTTP round-trip over a
  temp DB).

**Out of scope:**
- Any changes to `internal/dbinit` (F01).
- Any changes to `internal/api/viewer/{handler,service,types}.go` (F02).
- Any changes to `internal/viewer/assets/` (F03).
- Any changes to `internal/cli/commands/web.go` (F04) — F05 only exposes the
  function F04 imports.
- New API endpoints, new SPA features, new services.

---

## 2. Requirements

### Functional Requirements

**REQ-F-001: Cloud-aware DB initialization**
`cmd/server/main.go` MUST call `dbinit.Init(ctx, dbinit.Options{})` in place of
the current hardcoded `db.InitDB("shark-tasks.db")` at line 44. The server
MUST honour `.sharkconfig.json`, support both local SQLite and Turso backends,
and respect `skip_migrations`. The integrity-check log lines and the
`Close()` defer MUST be preserved against the returned `*repository.DB`.

**REQ-F-002: ViewerService wired into ServiceContainer**
`cmd/server/services.go::ServiceContainer` MUST gain a new field:
```go
ViewerService *services.ViewerService
```
`WireServices(db, projectRoot)` MUST construct it using the existing
repositories it already builds (`epicRepo`, `featureRepo`, `taskRepo`,
`historyRepo`, `noteRepo`, plus `bugRepoAdapter` and `changeCardRepoAdapter`),
`workflow.Service`, and `status.CalculationService`, then assign it to the
container before returning. All existing container fields MUST remain
unchanged.

**REQ-F-003: Viewer routes registered**
After the existing three `RegisterRoutes` calls (TaskHandler, FeatureHandler,
EpicHandler at lines 84-86 of `main.go`), `main.go` MUST register the viewer
routes:
```go
viewerHandler := viewer.NewHandler(svcs.ViewerService)
viewerHandler.RegisterRoutes(mux)
```
All seven viewer endpoints (`/api/v1/viewer/summary`, `/hierarchy`,
`/history/{key}`, `/file/{key}`, `/features/{key}/tasks`, `/recent-activity`,
`/workflow-meta`) MUST be mounted on the same `mux`.

**REQ-F-004: SPA served at GET /**
`main.go` MUST replace the placeholder `GET /` handler at lines 67-69:
```go
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Shark Task Manager API - Database Ready")
})
```
with a call into the viewer assets package:
```go
viewer_assets.RegisterStaticRoutes(mux)
```
which mounts:
- `GET /` → `internal/viewer/assets/viewer.html` (Content-Type `text/html; charset=utf-8`)
- `GET /static/` → `http.FileServer(http.FS(embeddedFS))` rooted at
  `internal/viewer/assets/` for vendor JS (`marked.min.js`) and any other
  static files F03 embeds.

Per ADR-E27-005/006 these assets are served from the embedded `go:embed` FS
produced by F03, never from the filesystem or a CDN.

**REQ-F-005: CORS middleware for viewer routes**
A new file `internal/api/viewer/cors.go` MUST export:
```go
func WithLocalCORS(next http.Handler) http.Handler
```
Behaviour (per ADR-E27-007):
- Inspect the `Origin` request header.
- Parse it; if `net/url.Parse` succeeds and `u.Hostname()` is exactly
  `"localhost"` or `"127.0.0.1"` (any port, http or https), echo the raw
  `Origin` value in `Access-Control-Allow-Origin` and set
  `Access-Control-Allow-Methods: GET, OPTIONS` and
  `Access-Control-Allow-Headers: Content-Type`.
- For any other origin, DO NOT set the header (browser enforcement blocks).
- Short-circuit `OPTIONS` preflight with `http.StatusNoContent`.
- Always call `next.ServeHTTP(w, r)` for non-OPTIONS requests, regardless of
  origin validity, so curl / same-origin requests are unaffected.

`main.go` MUST apply `WithLocalCORS` ONLY to the viewer routes, not to the
CRUD routes or `/`. This is achieved by wrapping the registration:
```go
viewerMux := http.NewServeMux()
viewerHandler.RegisterRoutes(viewerMux)
mux.Handle("/api/v1/viewer/", viewer.WithLocalCORS(viewerMux))
```
(Exact pattern is an implementation detail — see §3.4. The invariant is "CORS
headers appear on viewer responses and not on `/api/v1/tasks/...`
responses".)

**REQ-F-006: Extractable StartServer entry point**
A new package `internal/viewer/server` MUST expose:
```go
package server

type Options struct {
    // Listener, if non-nil, is used as the HTTP listener (pre-bound by the
    // caller). If nil, the server binds Addr.
    Listener net.Listener

    // Addr is the bind address (e.g. "127.0.0.1:7777"). Ignored if Listener
    // is non-nil. If empty, defaults to ":8080".
    Addr string

    // ProjectRoot overrides the auto-detected project root. Empty is
    // normally correct.
    ProjectRoot string

    // DB is an already-initialized *repository.DB. If nil, StartServer calls
    // dbinit.Init with Options.ProjectRoot.
    DB *repository.DB

    // Ready, if non-nil, is closed after the listener is bound and routes are
    // registered but BEFORE ListenAndServe blocks. Callers (like `shark web`)
    // use this as a signal to print the URL and open the browser.
    Ready chan<- struct{}
}

func StartServer(ctx context.Context, opts Options) error
```

Behaviour:
1. If `opts.DB` is nil, call `dbinit.Init(ctx, dbinit.Options{ProjectRoot: opts.ProjectRoot})`.
2. Call `db.CheckIntegrity` on the underlying `*sql.DB`; return error on failure.
3. Call `WireServices(db, projectRoot)` — this function stays in package
   `main` of `cmd/server` but is accessible because `StartServer` calls back
   into a wiring hook passed via `Options`, OR because `WireServices` is
   moved into `internal/viewer/server` alongside `StartServer` (see §3.2 for
   the decision).
4. Build the `http.ServeMux` per REQ-F-003, REQ-F-004, REQ-F-005.
5. Wrap in `otelhttp.NewHandler(mux, "shark-api")`.
6. Create the `http.Server{Handler: handler}`. If `opts.Listener` is non-nil,
   use it; otherwise bind `opts.Addr` (or `":8080"` default).
7. Close `opts.Ready` (if non-nil) after the listener is bound and before
   `srv.Serve(listener)` is called.
8. Run `srv.Serve` (or `srv.ListenAndServe` if no listener).
9. When `ctx.Done()` fires, call `srv.Shutdown(ctx2)` with a 30-second
   timeout, matching the existing `shutdownTimeout` constant.
10. Return `http.ErrServerClosed` as nil (graceful shutdown is not an error).

`cmd/server/main.go::main` collapses to:
```go
func main() {
    cfg := loadObservabilityConfig()
    shutdown, _ := observability.InitProvider(cfg)
    defer shutdown(context.Background())
    observability.InitLogger(cfg)

    ctx, cancel := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    if err := server.StartServer(ctx, server.Options{Addr: ":8080"}); err != nil {
        slog.Error("server exited with error", "error", err)
        os.Exit(1)
    }
}
```

**REQ-F-007: F04 CLI integration contract**
`internal/viewer/server.StartServer` MUST be importable from
`internal/cli/commands/web.go` without creating a circular import. This is
why it lives in `internal/viewer/server`, NOT in `cmd/server` (which is
`package main` and not importable).

`shark web` calls:
```go
readyCh := make(chan struct{})
go func() {
    _ = server.StartServer(ctx, server.Options{
        Listener: l,       // pre-bound listener from F04's port-picking logic
        DB:       db,      // already obtained via cli.GetDB
        Ready:    readyCh,
    })
}()
<-readyCh
fmt.Printf("Viewer ready at http://%s\n", l.Addr().String())
// ... open browser ...
<-ctx.Done()
```

**REQ-F-008: Backward compatibility**
After F05 lands:
- `go build ./cmd/server/...` MUST succeed.
- `./bin/shark-task-manager` (or equivalent) MUST start, respond 200 on
  `/health`, return the SPA on `/`, and return valid JSON on all seven viewer
  endpoints.
- The existing CRUD endpoints (`/api/v1/tasks/...`, `/api/v1/features/...`,
  `/api/v1/epics/...`) MUST continue to respond byte-for-byte identically.
- `make test` MUST pass.
- `make lint` MUST pass.
- `make fmt` MUST produce no diff.

### Non-Functional Requirements

**REQ-NF-001: Performance**
Server startup time MUST remain under 500ms from invocation to the `Ready`
signal, matching the existing `cmd/server` startup. F05 adds no blocking I/O
beyond what F01 already contributes.

**REQ-NF-002: Security**
- Default bind address is `:8080` when invoked via `./cmd/server`.
- When invoked via `shark web` (F04), the bind address is `127.0.0.1:<port>`,
  never `0.0.0.0`. The `server.Options.Addr` default inherits whatever the
  caller passes.
- CORS is locked to `localhost`/`127.0.0.1` per ADR-E27-007.
- Viewer endpoints are read-only (enforced by F02); F05 adds no mutation
  paths.

**REQ-NF-003: Graceful shutdown**
The 30-second shutdown timeout MUST be preserved. `srv.Shutdown(ctx)` MUST be
called before the function returns, even on error paths.

---

## 3. Architecture

### 3.1 New file layout

```
cmd/server/
├── main.go              [MODIFIED] ~30 lines; delegates to internal/viewer/server
├── services.go          [MODIFIED] ViewerService added to ServiceContainer
└── main_test.go         [NEW] integration smoke test

internal/
├── api/viewer/
│   ├── handler.go       [from F02]
│   ├── service.go       [from F02]
│   ├── types.go         [from F02]
│   ├── cors.go          [NEW in F05 — WithLocalCORS middleware]
│   └── handler_test.go  [from F02]
├── viewer/
│   ├── assets/          [from F03]
│   │   ├── viewer.html
│   │   └── vendor/marked.min.js
│   ├── assets.go        [from F03 — go:embed]
│   └── server/
│       ├── server.go    [NEW in F05 — StartServer entry point]
│       └── server_test.go [NEW — ready-signal + shutdown tests]
└── services/
    └── viewer_service.go [from F02]
```

### 3.2 Wiring location decision

`WireServices()` currently lives in `cmd/server/services.go` (package `main`).
F04 (`shark web`) cannot import `package main`. Two options:

- **Option A — Move `WireServices` into `internal/viewer/server`.** Pro:
  single source of wiring. Con: `cmd/server/services.go` gets deleted, its
  adapter types (`workSessionAdapter`, `taskHistoryAdapter`) also move.
- **Option B — Keep `WireServices` in `cmd/server` and pass a wiring function
  into `server.StartServer` via `Options.WireServices func(*repository.DB, string) *ServiceContainer`.**
  Pro: minimal file movement. Con: `ServiceContainer` also has to move to
  `internal/viewer/server` (or to a third package) so both callers can type
  the return value.

**Decision: Option A.**

`internal/viewer/server/server.go` will own `WireServices`, `ServiceContainer`,
`workSessionAdapter`, and `taskHistoryAdapter`. `cmd/server/services.go` is
DELETED as part of this feature. `cmd/server/main.go` becomes a ~30-line
wrapper.

Rationale:
- Option B requires `ServiceContainer` to be importable from two places, which
  means moving the type somewhere anyway — same net file movement as Option A.
- Option A makes the contract visible: "the server lives in
  `internal/viewer/server` and is reachable from both `cmd/server/main.go` and
  `shark web`".
- Consistent with ADR-E27-009 ("`shark web` runs the server in-process"). The
  architecture document recommends `internal/viewer/runner`; we use the name
  `internal/viewer/server` because the package's primary export is
  `StartServer`.

> **Note to implementation:** When moving `WireServices`, keep the function
> body byte-for-byte identical. The only delta is the package declaration
> (`package main` → `package server`) and import path updates. No logic
> changes. Run `make test` after the move to confirm nothing regressed.

### 3.3 CORS middleware design (`internal/api/viewer/cors.go`)

```go
package viewer

import (
    "net/http"
    "net/url"
)

// WithLocalCORS echoes the request Origin in Access-Control-Allow-Origin
// only when the origin host is localhost or 127.0.0.1 (any port, any scheme).
// Applies to viewer routes only; see REQ-F-005.
func WithLocalCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if origin != "" {
            if u, err := url.Parse(origin); err == nil {
                host := u.Hostname()
                if host == "localhost" || host == "127.0.0.1" {
                    w.Header().Set("Access-Control-Allow-Origin", origin)
                    w.Header().Set("Vary", "Origin")
                    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
                    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
                }
            }
        }

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

Tests (in `cors_test.go`, in the same package):
- Allowed: `http://localhost:5173`, `http://127.0.0.1:7777`, `https://localhost:9000`.
- Blocked (no ACAO header): `http://example.com`, empty origin, malformed.
- OPTIONS preflight returns 204 for allowed origins.
- GET with allowed origin passes through to the wrapped handler.

### 3.4 SPA + CORS integration in main.go

The shape of route registration inside `StartServer` (in
`internal/viewer/server/server.go`):

```go
func StartServer(ctx context.Context, opts Options) error {
    // 1. DB init (REQ-F-001)
    db := opts.DB
    if db == nil {
        var err error
        db, err = dbinit.Init(ctx, dbinit.Options{ProjectRoot: opts.ProjectRoot})
        if err != nil {
            return fmt.Errorf("dbinit: %w", err)
        }
        defer db.Close()
    }

    if err := dbpkg.CheckIntegrity(db.DB); err != nil {
        return fmt.Errorf("integrity check failed: %w", err)
    }
    slog.Info("Database integrity check passed")

    // 2. Service wiring (REQ-F-002)
    svcs := WireServices(db, opts.ProjectRoot)

    // 3. Routes
    mux := http.NewServeMux()

    // Health + existing CRUD handlers
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        if err := db.DB.Ping(); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            fmt.Fprintf(w, "Database unavailable: %v", err)
            return
        }
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "OK")
    })
    api.NewTaskHandler(svcs.TaskService).RegisterRoutes(mux)
    api.NewFeatureHandler(svcs.FeatureService).RegisterRoutes(mux)
    api.NewEpicHandler(svcs.EpicService).RegisterRoutes(mux)

    // Viewer routes behind CORS middleware (REQ-F-003, REQ-F-005)
    viewerMux := http.NewServeMux()
    viewer.NewHandler(svcs.ViewerService).RegisterRoutes(viewerMux)
    mux.Handle("/api/v1/viewer/", viewer.WithLocalCORS(viewerMux))

    // SPA + static assets (REQ-F-004)
    assets.RegisterStaticRoutes(mux) // from internal/viewer/assets package

    // 4. Wrap in otelhttp
    handler := otelhttp.NewHandler(mux, "shark-api")

    // 5. Listener
    listener := opts.Listener
    if listener == nil {
        addr := opts.Addr
        if addr == "" {
            addr = ":8080"
        }
        var err error
        listener, err = net.Listen("tcp", addr)
        if err != nil {
            return fmt.Errorf("listen %s: %w", addr, err)
        }
    }

    srv := &http.Server{Handler: handler}

    // 6. Ready signal (REQ-F-006, REQ-F-007)
    if opts.Ready != nil {
        close(opts.Ready)
    }

    // 7. Serve + graceful shutdown
    errCh := make(chan error, 1)
    go func() {
        slog.Info("Server started", "addr", listener.Addr().String())
        errCh <- srv.Serve(listener)
    }()

    select {
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
        defer cancel()
        if err := srv.Shutdown(shutdownCtx); err != nil {
            return fmt.Errorf("shutdown: %w", err)
        }
        slog.Info("Server stopped gracefully")
        return nil
    case err := <-errCh:
        if errors.Is(err, http.ErrServerClosed) {
            return nil
        }
        return fmt.Errorf("serve: %w", err)
    }
}
```

### 3.5 Exact integration seams (file + line map)

| # | File | Line(s) in current code | Change |
|---|---|---|---|
| 1 | `cmd/server/main.go` | 44 (`database, err := db.InitDB("shark-tasks.db")`) | DELETE; logic moves to `server.StartServer` which calls `dbinit.Init`. |
| 2 | `cmd/server/main.go` | 49 (`defer database.Close()`) | DELETE; `StartServer` owns DB lifetime. |
| 3 | `cmd/server/main.go` | 54-58 (integrity check) | DELETE; moves to `server.StartServer`. |
| 4 | `cmd/server/main.go` | 61-62 (`svcs := WireServices(repoDB, ".")`) | DELETE; `server.StartServer` calls `WireServices`. |
| 5 | `cmd/server/main.go` | 65 (`mux := http.NewServeMux()`) | DELETE; moves to `server.StartServer`. |
| 6 | `cmd/server/main.go` | 67-69 (GET / stub) | DELETE; replaced by `assets.RegisterStaticRoutes(mux)` inside `server.StartServer`. |
| 7 | `cmd/server/main.go` | 71-81 (GET /health) | MOVED into `server.StartServer`. |
| 8 | `cmd/server/main.go` | 84-86 (three `RegisterRoutes` calls) | MOVED into `server.StartServer`. |
| 9 | `cmd/server/main.go` | 86 (after `NewEpicHandler...`) | ADD new lines in `server.StartServer`: create viewerMux, register viewer handler, wrap with `WithLocalCORS`, mount on `/api/v1/viewer/`. |
| 10 | `cmd/server/main.go` | 92 (`otelhttp.NewHandler`) | MOVED into `server.StartServer`. |
| 11 | `cmd/server/main.go` | 94-132 (server lifecycle) | MOVED into `server.StartServer`. `main()` retains only observability init, signal context, and `server.StartServer(ctx, Options{Addr: ":8080"})`. |
| 12 | `cmd/server/services.go` | whole file | **DELETED.** Contents moved to `internal/viewer/server/wire.go`. |
| 13 | `cmd/server/services.go` | `ServiceContainer` struct lines 144-155 | MOVED to `internal/viewer/server/wire.go`; ADD field `ViewerService *services.ViewerService` before returning. |
| 14 | `cmd/server/services.go` | `WireServices` function body | MOVED to `internal/viewer/server/wire.go`; ADD `ViewerService` construction after line 267 using existing repos + `workflowSvc` + `status.NewCalculationService(db, ...)`. |
| 15 | NEW `internal/api/viewer/cors.go` | — | Create `WithLocalCORS` per §3.3. |
| 16 | NEW `internal/viewer/server/server.go` | — | Create `StartServer` + `Options` per §3.4. |
| 17 | NEW `internal/viewer/server/wire.go` | — | Moved `ServiceContainer` + `WireServices` from deleted `cmd/server/services.go`. |
| 18 | NEW `cmd/server/main_test.go` | — | HTTP round-trip smoke test per §5. |

### 3.6 Package dependency check

```
cmd/server/main.go
    → internal/viewer/server         (NEW)
        → internal/dbinit            (F01)
        → internal/api               (existing CRUD handlers)
        → internal/api/viewer        (F02 + F05 cors.go)
        → internal/viewer/assets     (F03)
        → internal/services          (F02 ViewerService)
        → internal/repository        (existing)
        → internal/workflow          (existing)
        → internal/status            (existing)
        → internal/observability     (existing)

internal/cli/commands/web.go         (F04)
    → internal/viewer/server         (NEW)
```

No cycles. `internal/viewer/server` depends downward on `internal/*` packages
only; nothing in `internal/api/viewer`, `internal/services`, or
`internal/repository` imports `internal/viewer/server`.

### 3.7 Embedded SPA registration contract (from F03)

F03 is expected to export from `internal/viewer/assets`:

```go
package assets

//go:embed viewer.html vendor/*
var FS embed.FS

// RegisterStaticRoutes mounts:
//   GET /           → viewer.html (text/html)
//   GET /static/    → http.FileServer(http.FS(FS)) stripped of "/static/"
func RegisterStaticRoutes(mux *http.ServeMux)
```

F05 uses `RegisterStaticRoutes(mux)` verbatim. If F03 landed with a different
function name, F05 will adapt (one-line change) but does not redefine the
embedding or change the asset set.

---

## 4. Implementation Plan

Ordered, each step should compile and test independently where possible.

1. **Move service wiring.**
   - Create `internal/viewer/server/wire.go`.
   - Copy `ServiceContainer`, `WireServices`, `workSessionAdapter`,
     `taskHistoryAdapter`, and the two compile-time interface assertions from
     `cmd/server/services.go` verbatim.
   - Change `package main` → `package server`.
   - Update imports.
   - DELETE `cmd/server/services.go`.
   - `go build ./...` MUST succeed (but will break `cmd/server/main.go`
     temporarily in step 2).

2. **Extract `StartServer`.**
   - Create `internal/viewer/server/server.go` with `Options` struct and
     `StartServer(ctx, opts)` function per §3.4.
   - Do NOT yet add viewer routes or SPA — reach parity with current
     `cmd/server/main.go` first.
   - Update `cmd/server/main.go` to call `server.StartServer(ctx, Options{Addr: ":8080"})`.
   - `go build ./...` + `make test` MUST pass.

3. **Add `ViewerService` to ServiceContainer.**
   - In `internal/viewer/server/wire.go`, add the `ViewerService` field to
     `ServiceContainer`.
   - In `WireServices`, after the existing service constructions, instantiate
     `services.NewViewerService(...)` with all required deps and assign to
     the container.
   - The F02 feature defines the concrete constructor signature. If F02 is
     not merged yet, add a `// TODO(E27-F02)` stub that returns nil and come
     back to this step once F02 is in `main`.
   - `go build ./...` MUST pass.

4. **Add CORS middleware.**
   - Create `internal/api/viewer/cors.go` with `WithLocalCORS` per §3.3.
   - Create `internal/api/viewer/cors_test.go` with the test cases listed in
     §3.3.
   - `go test ./internal/api/viewer/...` MUST pass.

5. **Register viewer routes + SPA inside `StartServer`.**
   - Create the `viewerMux` block per §3.4.
   - Call `assets.RegisterStaticRoutes(mux)`.
   - `go build ./...` MUST pass.

6. **Integration smoke test.**
   - Create `cmd/server/main_test.go` per §5 below.
   - Spin up the server on port `0` (OS-assigned), hit `/health`, `/`,
     `/api/v1/viewer/summary`, and one CRUD endpoint. Assert 200 + basic
     content.
   - `make test` MUST pass.

7. **Final quality gate.**
   - `make fmt && make lint && make test`.
   - Manual smoke: `./bin/shark-task-manager` in a seeded project, `curl`
     each endpoint, open `http://127.0.0.1:8080/` in a browser, confirm SPA
     renders.

---

## 5. Testing Strategy

### 5.1 Unit tests

| Target | File | What it covers |
|---|---|---|
| `WithLocalCORS` | `internal/api/viewer/cors_test.go` | Allowed/blocked origins, OPTIONS preflight, passthrough to inner handler, headers set/not set. |
| `StartServer` wiring | `internal/viewer/server/server_test.go` | Ready channel closed before serve, context-cancel triggers shutdown, shutdown returns nil on `http.ErrServerClosed`, double-close of Ready is not attempted. |

`server_test.go` uses a pre-bound listener on `127.0.0.1:0` and a mocked DB
path (`dbinit.Init` with a tempdir `ProjectRoot`). Tests must not use the
production `shark-tasks.db`.

### 5.2 Integration smoke test (`cmd/server/main_test.go`)

**Scope:** this is the **only** test in the repo that exercises the full
wiring stack (dbinit → services → handlers → mux → otelhttp → SPA). It is the
gate that proves F01-F04 compose correctly.

```go
func TestServerIntegration_SmokeTest(t *testing.T) {
    // 1. Create tempdir with minimal .sharkconfig.json + empty sqlite DB
    tempDir := t.TempDir()
    writeTestConfig(t, tempDir)

    // 2. Pre-bind a listener on 127.0.0.1:0
    lis, err := net.Listen("tcp", "127.0.0.1:0")
    require.NoError(t, err)

    // 3. Start server in goroutine with ready signal
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    readyCh := make(chan struct{})
    errCh := make(chan error, 1)
    go func() {
        errCh <- server.StartServer(ctx, server.Options{
            Listener:    lis,
            ProjectRoot: tempDir,
            Ready:       readyCh,
        })
    }()
    <-readyCh

    base := "http://" + lis.Addr().String()

    // 4. Assert each endpoint category
    t.Run("health", func(t *testing.T) {
        resp, err := http.Get(base + "/health")
        require.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, 200, resp.StatusCode)
    })

    t.Run("spa_root", func(t *testing.T) {
        resp, err := http.Get(base + "/")
        require.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, 200, resp.StatusCode)
        assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
        body, _ := io.ReadAll(resp.Body)
        assert.Contains(t, string(body), "<html")
    })

    t.Run("viewer_summary", func(t *testing.T) {
        resp, err := http.Get(base + "/api/v1/viewer/summary")
        require.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, 200, resp.StatusCode)
        assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
    })

    t.Run("viewer_cors_allowed", func(t *testing.T) {
        req, _ := http.NewRequest("GET", base+"/api/v1/viewer/summary", nil)
        req.Header.Set("Origin", "http://localhost:5173")
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, "http://localhost:5173",
            resp.Header.Get("Access-Control-Allow-Origin"))
    })

    t.Run("viewer_cors_blocked", func(t *testing.T) {
        req, _ := http.NewRequest("GET", base+"/api/v1/viewer/summary", nil)
        req.Header.Set("Origin", "http://evil.example.com")
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
    })

    t.Run("crud_unchanged", func(t *testing.T) {
        // Existing CRUD endpoint still responds
        resp, err := http.Get(base + "/api/v1/tasks")
        require.NoError(t, err)
        defer resp.Body.Close()
        assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404)
        // NOT CORS-wrapped: no ACAO header even for localhost origin
    })

    // 5. Graceful shutdown
    cancel()
    select {
    case err := <-errCh:
        assert.NoError(t, err)
    case <-time.After(5 * time.Second):
        t.Fatal("server did not shut down within 5s")
    }
}
```

This test uses a real SQLite DB (created fresh per `t.TempDir()`) because the
whole point is to prove wiring end-to-end. Per `.claude/rules/testing/`, this
is an integration test and is allowed to use a real DB, but it does NOT
touch the shared project test DB. Every run creates and destroys its own
SQLite file inside `t.TempDir()`.

### 5.3 Lint/format gate

- `make fmt` — no diff.
- `make lint` — no new warnings.
- `make test` — all existing tests plus new `cors_test.go`, `server_test.go`,
  and `main_test.go` pass.

---

## 6. Dependencies

**All four prerequisite features MUST be merged to `main` before F05 can be
fully implemented:**

| Prerequisite | Feature | Provides | Consumed by F05 as |
|---|---|---|---|
| **F01** | DB Init Extraction | `internal/dbinit` package with `Init(ctx, Options) (*repository.DB, error)` | `server.StartServer` calls `dbinit.Init` to replace hardcoded `db.InitDB`. |
| **F02** | Viewer API | `internal/api/viewer.{Handler,NewHandler,RegisterRoutes}`, `services.ViewerService`, `services.NewViewerService(...)` | `WireServices` constructs `ViewerService`; `StartServer` registers handler routes. |
| **F03** | Single-file SPA | `internal/viewer/assets.{FS,RegisterStaticRoutes}` (or equivalent embed export) | `StartServer` calls `assets.RegisterStaticRoutes(mux)` to serve `GET /` and `GET /static/`. |
| **F04** | `shark web` CLI | Nothing F05 imports from F04. However, F05 MUST publish the `internal/viewer/server.StartServer` function before F04 lands, so F04 can import and call it. | F04 is a consumer of F05's public API. |

**Dependency direction note:** F05 and F04 have a mutual lockstep. F05 must
expose `StartServer` before F04 can implement the CLI. In practice:
1. Merge F01.
2. Merge F02.
3. Merge F03.
4. Merge F05 with the `StartServer` entry point in place and
   `cmd/server/main.go` using it.
5. Merge F04, which imports `internal/viewer/server`.

If the PM wants to merge F04 first, F05 must provide a skeleton
`internal/viewer/server/server.go` with the `StartServer` signature stub (even
if its body is just the existing main.go logic, pre-viewer-routes). F04 can
then land against the stub and F05 completes the wiring afterward.

---

## 7. Out of Scope

- New API endpoints or SPA features.
- Repository / service / handler changes (all owned by F01-F04).
- Authentication, authorization, rate limiting, or any new middleware beyond
  `WithLocalCORS`.
- Configuration of port-picking logic — that belongs to F04.
- Browser-launch logic — that belongs to F04.
- WebSockets / SSE / live updates (see ADR-E27-012).
- Multi-project support — one server instance per project root.

---

## 8. Open Questions

1. **Assets package export name.** F03's exact public API is not yet merged.
   If F03 ships `assets.RegisterStaticRoutes(mux)`, F05 uses it verbatim. If
   F03 ships a different name (e.g. `assets.Handler()`), F05 adapts in a
   one-line change during implementation.
2. **`ServiceContainer` field order.** The current struct groups services by
   entity. `ViewerService` is read-only composition across all entities, so
   it fits naturally as the last field. Not a blocker; PR review can adjust.
3. **`dbinit.Options.ProjectRoot` when invoked from `cmd/server/main.go`.**
   We pass empty string to trigger auto-detection (walks up from CWD). When
   invoked from `shark web`, F04 will pass the CLI's already-resolved
   project root. The `Options` struct accepts both paths.

---

*Last Updated*: 2026-04-11
