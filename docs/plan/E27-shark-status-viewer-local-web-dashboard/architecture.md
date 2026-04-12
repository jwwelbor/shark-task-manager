---
epic_key: E27
doc_type: architecture
status: draft
---

# E27 — Architecture: Shark Status Viewer (Local Web Dashboard)

**Epic:** E27 — Shark Status Viewer — Local Web Dashboard
**Architect:** Claude (architect agent, in_design phase)
**Date:** 2026-04-11

This document is the system-level design for E27. It refines the architectural direction already recorded in `epic.md` (Option B: extend `cmd/server`), maps it onto the actual state of the codebase, and captures the decisions a decomposer/developer needs in order to produce feature-level specs.

---

## 1. Component Overview

### 1.1 What changes

| Layer | Change | Files |
|---|---|---|
| DB initialization | **Extract** the cloud-aware init currently trapped inside `internal/cli/db_init.go` into a shared `internal/dbinit` package callable from both `cmd/shark` and `cmd/server`. | NEW `internal/dbinit/init.go`; MODIFIED `internal/cli/db_init.go` (delegate), `cmd/server/main.go` (call `dbinit.Init`). |
| HTTP handlers | **Add** a new `internal/api/viewer` package with one handler struct and 7 read-only endpoints, all registered behind the existing `net/http.ServeMux` using Go 1.22 pattern syntax (same style as `internal/api/task_handler.go`). | NEW `internal/api/viewer/{handler,service,types,handler_test}.go`. |
| Service layer | **Add** `internal/services/viewer_service.go` — a *read-only* service that composes existing repositories to produce dashboard-shaped rollups. | NEW `internal/services/viewer_service.go` (+ `viewer_service_test.go`). |
| Static assets | **Add** `internal/viewer/assets/viewer.html` and embed via `go:embed` into the server binary. Serve at `GET /`. | NEW `internal/viewer/assets.go`, NEW `internal/viewer/assets/viewer.html`. |
| CLI surface | **Add** `shark web [--port N] [--no-open]` — a thin Cobra command that starts the server (in-process, not a shellout) and opens the browser. | NEW `internal/cli/commands/web.go`. |
| Server wiring | `cmd/server/main.go` calls `dbinit.Init`, mounts viewer routes, replaces the placeholder `GET /` handler with the embedded SPA. `cmd/server/services.go` wires `ViewerService` into `ServiceContainer`. | MODIFIED `cmd/server/main.go`, `cmd/server/services.go`. |

### 1.2 What stays (explicitly)

- **`internal/api/{epic,feature,task}_handler.go`** — all existing CRUD handlers are untouched. Viewer endpoints are strictly additive, live under `/api/v1/viewer/`, and never mutate state.
- **`cmd/server/main.go`** keeps `net/http.ServeMux` + `otelhttp` wrapping. We do **not** introduce chi, gorilla/mux, echo, or any other router. The Go 1.22+ pattern syntax `GET /api/v1/viewer/history/{key}` is already in use in `task_handler.go` and is sufficient.
- **Repositories, models, workflow service, status calculation service** — all unchanged. The viewer layer is pure composition.
- **Existing `internal/cli/db_init.go`** file remains; its body collapses to `return dbinit.Init(ctx, dbinit.Options{ProjectRoot: root})` so other CLI callers (`GetDB`) keep working with no call-site changes.

### 1.3 New component diagram

```
shark web (Cobra cmd)
      │
      ▼
┌──────────────────────────────────────────────────────────┐
│  cmd/server (in-process; reused by `shark web`)          │
│  ┌────────────────────────────────────────────────────┐  │
│  │ net/http.ServeMux (otelhttp wrapped)               │  │
│  │                                                    │  │
│  │ GET /                 → embedded viewer.html       │  │
│  │ GET /health           → existing                   │  │
│  │ /api/v1/tasks/...     → existing api.TaskHandler   │  │
│  │ /api/v1/features/...  → existing api.FeatureHandl. │  │
│  │ /api/v1/epics/...     → existing api.EpicHandler   │  │
│  │ /api/v1/viewer/...    → NEW api/viewer.Handler ────┼──┐
│  └────────────────────────────────────────────────────┘  │  │
└──────────────────────────────────────────────────────────┘  │
                                                              ▼
                              ┌────────────────────────────────────┐
                              │ services.ViewerService (NEW)       │
                              │   Summary()                        │
                              │   Hierarchy()                      │
                              │   History(key)                     │
                              │   File(key)                        │
                              │   FeatureTasks(key, filters)       │
                              │   RecentActivity(opts)             │
                              │   WorkflowMeta()                   │
                              └──────────┬─────────────────────────┘
                                         │ (composes, read-only)
                         ┌───────────────┼─────────────────┐
                         ▼               ▼                 ▼
                  EpicRepository  FeatureRepository  TaskRepository
                  BugRepository   ChangeCardRepo     TaskHistoryRepository
                  EntityNoteRepo        │                 │
                                        ▼                 ▼
                                  repository.DB  ← dbinit.Init (NEW shared pkg)
                                        │
                                        ▼
                                 local SQLite / Turso
```

The entire viewer stack sits *on top of* the existing layering. Nothing below `services/` needs to know the viewer exists.

---

## 2. Key Technical Decisions (ADRs)

Each decision includes context, the option chosen, and the rejected alternatives. These are the decisions a decomposer/developer must not re-litigate.

### ADR-E27-001: Extend `cmd/server` rather than build a new binary

- **Status:** Accepted (carried forward from epic PRD).
- **Context:** We need an HTTP endpoint to serve a viewer SPA and JSON rollups for a shark project. Four options were considered in the research phase: (A) new `cmd/viewer` binary, (B) extend existing `cmd/server`, (C) run viewer in the CLI process itself, (D) ship as a separate repo / external tool.
- **Decision:** **Option B — extend `cmd/server`.**
- **Rationale:**
  - `cmd/server/services.go` already wires Epic/Feature/Task services via `WireServices`. A new binary would duplicate that wiring or force extraction of a `ServiceContainer` package — scope creep.
  - Existing `internal/api/` handlers already use the same patterns viewer endpoints need (`net/http` + Go 1.22 patterns, `respondJSON`, `handleServiceError`). Reuse is one import away.
  - Only one real gap: `cmd/server/main.go` line 44 hardcodes `db.InitDB("shark-tasks.db")` and ignores `.sharkconfig.json`. That gap must be closed regardless (ADR-002) and benefits every server consumer, not just the viewer.
- **Rejected alternatives:**
  - **A (new binary):** duplicates observability init, service wiring, graceful shutdown, otelhttp. No benefit — the server is already minimal (~130 LOC).
  - **C (in-CLI only):** blocks on the CLI process; awkward for `shark web` UX (user has to Ctrl-C to close the browser-less process); can't be reused by any future team dashboard.
  - **D (separate repo):** contradicts CLAUDE.md's "single source of truth" design. Viewer must read the same DB the CLI writes; cross-repo coupling is worse than an internal package.

### ADR-E27-002: Shared `internal/dbinit` package

- **Status:** Accepted.
- **Context:** `internal/cli/db_init.go::initDatabase` is the only place that correctly reads `.sharkconfig.json`, picks local vs Turso, walks the project root, and applies schema/migrations with `skip_migrations` awareness. `cmd/server/main.go` bypasses all of that with a hardcoded `db.InitDB("shark-tasks.db")`.
- **Decision:** Extract the body of `initDatabase` into a new package `internal/dbinit` with this surface:
  ```go
  package dbinit

  type Options struct {
      ProjectRoot    string // optional; auto-detected if empty
      ConfigPath     string // optional; defaults to <root>/.sharkconfig.json
      SkipMigrations *bool  // optional; overrides config value
  }

  func FindProjectRoot(startDir string) (string, error)
  func Init(ctx context.Context, opts Options) (*repository.DB, error)
  func MustInit(ctx context.Context, opts Options) *repository.DB // panics on error, for main()
  ```
  - `internal/cli/db_init.go::initDatabase` becomes a one-line wrapper around `dbinit.Init`.
  - `internal/cli/root.go::FindProjectRoot` becomes a one-line wrapper around `dbinit.FindProjectRoot` (or the two functions are exposed side-by-side; see §3.2).
  - `cmd/server/main.go` replaces `db.InitDB("shark-tasks.db")` with `dbinit.MustInit(ctx, dbinit.Options{})`.
- **Rationale:**
  - **DRY:** there is exactly one cloud-aware init path in the codebase after this change. Cloud/Turso support "just works" for every entry point.
  - **Testability:** `Options` makes it trivial to inject a test project root.
  - **CLAUDE.md compliance:** "Project root auto-detection" is a documented invariant of the codebase; extracting it into a shared package makes the invariant enforceable instead of accidental.
- **Rejected alternatives:**
  - **Duplicate the logic in `cmd/server/main.go`:** violates DRY and guarantees drift.
  - **Move the logic into `internal/db`:** `internal/db` is supposed to be the low-level SQLite/Turso driver layer; adding project-root detection and config loading there creates an unwanted dependency from `internal/db` on `internal/config`.
  - **Put it in `internal/config`:** wrong direction; config should not import `internal/db`.

### ADR-E27-003: Read-only viewer service — no new mutation paths

- **Status:** Accepted.
- **Context:** The viewer reads summaries, hierarchy, history, files, and recent activity. It does not create, update, advance, or delete anything.
- **Decision:** `services.ViewerService` exposes **read-only** methods. It does **not** wrap `TaskService.StartTask`, `FeatureService.Complete`, etc. If the viewer UI ever needs a write, it calls the existing CRUD endpoints (`PATCH /api/v1/tasks/{key}/...`) directly, not new viewer endpoints.
- **Rationale:**
  - Preserves a clean security perimeter: any future auth layer only needs to guard mutation endpoints, and the viewer stack stays inside a read-only envelope.
  - Keeps `ViewerService` as a thin composition layer with no transaction management and no workflow validation — both of which belong to the entity services.
  - Prevents architectural drift where "just one more helper" turns `ViewerService` into a god service.
- **Rejected alternatives:**
  - **One unified service with read+write dashboard methods:** violates the "one service per aggregate" rule in `service-design.md`.
  - **Let handlers call multiple entity services directly:** violates the "thin handlers" rule — aggregation logic would leak into HTTP handlers.

### ADR-E27-004: `ViewerServicer` interface lives in `internal/api/viewer`

- **Status:** Accepted.
- **Context:** Handlers need to be testable with mocks (per `services/testing.md` — "CLI/HTTP tests use mocked services, never real database"). The codebase convention (e.g. `TaskRepository` interface inside `internal/services/task_service.go`) is: **define interfaces at the point of use.**
- **Decision:** Declare a `ViewerServicer` interface in `internal/api/viewer/service.go`. The concrete `services.ViewerService` implements it structurally. Handler tests in `handler_test.go` use a `MockViewerServicer` with function fields per `services/testing.md`.
- **Rationale:** Matches existing interfaces-at-consumer pattern; zero coupling between the handler package and the concrete service implementation; full mockability.
- **Rejected:** declaring the interface in `internal/services/` would couple the service package to its HTTP consumer and contradict the established pattern.

### ADR-E27-005: Single-file SPA with `go:embed`, no build step

- **Status:** Accepted (carried forward from epic PRD).
- **Context:** `shark` is a single Go binary. Introducing npm/webpack/vite would add a build step to a tool whose entire appeal is "go install and run".
- **Decision:**
  - One file: `internal/viewer/assets/viewer.html`.
  - Vanilla JS (no framework). Marked.js for markdown rendering, loaded **from the same-origin `/static/` path, not from a CDN** (see ADR-006).
  - Embedded into the binary via `//go:embed assets/viewer.html` in `internal/viewer/assets.go`.
  - Served at `GET /` and vendor JS at `GET /static/*` via `http.FileServer(http.FS(embeddedFS))`.
- **Rationale:**
  - No build step = no bundler drift, no node_modules in CI, no version churn. The viewer ships with the binary byte-for-byte reproducibly.
  - Vanilla JS is sufficient for the scope (tree view, markdown panel, history table, dashboard cards). A framework would be cargo-culted complexity.
- **Rejected alternatives:**
  - **React/Vue SPA with build step:** fails the "no new dependencies" epic constraint.
  - **Server-side templating (`html/template`):** makes the dashboard reload-per-click instead of SPA-fluid; history view would lose state on every click; recent activity would need polling-via-page-reload.

### ADR-E27-006: Marked.js loaded from same-origin `/static/`, not a CDN

- **Status:** Accepted (amends epic PRD which mentioned CDN loading).
- **Context:** The epic PRD says "Marked.js CDN". CDNs require network access, which contradicts the "run locally, works offline" story of a developer tool. They also expose the user to CSP/CORS issues and supply-chain risk.
- **Decision:** Vendor `marked.min.js` into `internal/viewer/assets/vendor/marked.min.js`, embed it alongside `viewer.html`, and serve it at `GET /static/vendor/marked.min.js`. The SPA loads it with a relative script tag: `<script src="/static/vendor/marked.min.js"></script>`.
- **Rationale:**
  - Works offline (CI, airplane, locked-down corporate networks).
  - Eliminates a third-party runtime dependency from the security perimeter.
  - Keeps bundle fully reproducible (the embedded binary is byte-for-byte the same as the source file).
- **Rejected:** CDN (`unpkg`/`cdnjs`) — incompatible with offline use and introduces a silent external dependency.

### ADR-E27-007: CORS — echo Origin only for `http://127.0.0.1:*` and `http://localhost:*`

- **Status:** Accepted.
- **Context:** The epic PRD mentions `Access-Control-Allow-Origin: http://localhost:*`. A literal wildcard `*` header is unsafe (allows any website on the internet to read a developer's local shark DB); a static origin breaks when the port is not 7777.
- **Decision:** Add a small middleware `internal/api/viewer/cors.go`:
  ```go
  func withLocalCORS(next http.Handler) http.Handler { ... }
  ```
  It inspects the `Origin` request header; if the URL's host matches `localhost` or `127.0.0.1` (any port), echo it back in `Access-Control-Allow-Origin`. For any other origin, do not set the header (browser then blocks the request). Applies only to `/api/v1/viewer/*` routes.
- **Rationale:**
  - Safe by default: no cross-origin website can read a user's project state.
  - Robust to port selection: auto-incrementing from 7777 works without code changes.
  - Allows a developer running their own frontend on e.g. `http://localhost:5173` to hit the viewer API during SPA development.
- **Rejected alternatives:**
  - `Access-Control-Allow-Origin: *` — trivially exploitable by any open browser tab.
  - Static `http://localhost:7777` — breaks port-increment UX.
  - No CORS at all — forbids any future local cross-origin dev harness.

### ADR-E27-008: File serving is DB-path-first with project-root containment

- **Status:** Accepted.
- **Context:** `GET /api/v1/viewer/file/{key}` must return markdown content from disk. A naive implementation would concatenate `{project_root}/{user_input}` which is a textbook path-traversal vulnerability.
- **Decision:**
  1. The handler parses the key and calls `ViewerService.File(ctx, key)`.
  2. The service looks up the entity by key in the DB and reads its `file_path` column — **never** user input.
  3. It joins `file_path` to the project root using `filepath.Join` then canonicalizes with `filepath.Abs` and `filepath.EvalSymlinks`.
  4. It verifies the canonical result has the project root as a prefix (`strings.HasPrefix(canon, rootCanon+string(os.PathSeparator))`). If not, return `&SecurityError{}` → HTTP 403.
  5. If the file does not exist on disk (entity is in the DB but its markdown file has been deleted), return `{"exists": false, "key": "..."}` with HTTP 200 — the UI surfaces this gracefully.
- **Rationale:**
  - Path resolved entirely from trusted DB state; user input is only a lookup key, validated against `internal/models/validation.go` regexes.
  - Two-layer defense: (a) DB-only path, (b) symlink-resolved containment check.
  - Matches `internal/fileops` atomic-write conventions and CLAUDE.md's "never trust file paths from user input" principle.
- **Rejected:** reading `file_path` directly without the containment check — if a malicious epic entry somehow had a relative path with `../`, we'd escape the project root.

### ADR-E27-009: `shark web` runs the server in-process (not shellout)

- **Status:** Accepted.
- **Context:** Two ways to expose `shark web`: (1) call `cmd/server/main` via `exec.Command`, or (2) factor `cmd/server` startup into a `serverRunner.Run(ctx, opts)` function callable from both `cmd/server/main.go` and `internal/cli/commands/web.go`.
- **Decision:** Option 2 — in-process.
  - Create `internal/viewer/runner/runner.go` with `func Run(ctx context.Context, opts Options) error`. `opts` includes `Port int`, `OpenBrowser bool`, `Addr string`.
  - `cmd/server/main.go` becomes a thin `runner.Run(context.Background(), defaultOpts())` wrapper that also installs the observability provider.
  - `shark web` calls `runner.Run(cmd.Context(), opts)` directly — same binary, no subprocess.
- **Rationale:**
  - No binary discovery problem (shellout would have to find `shark-task-manager` on PATH or next to `shark`).
  - Graceful Ctrl-C just works — a single signal handler, single process tree.
  - Unit-testable: `runner.Run` takes a context and an `Options`, so tests can start the server on a random port and cancel via `ctx`.
- **Rejected:** shellout — fragile path resolution, double process tree, awkward log forwarding, no way for tests to verify the viewer end-to-end without a second binary.

### ADR-E27-010: Port selection — try 7777, then 7778, up to 7790; fail cleanly

- **Status:** Accepted.
- **Context:** The epic PRD says "defaults to 7777, auto-increments if in use". Unbounded auto-increment is a DoS foot-gun (if 65k ports are busy you spin forever).
- **Decision:**
  - Default port 7777.
  - If `--port` is given, honor it strictly — do not auto-increment; fail with exit code 2 and a clear message if taken.
  - If `--port` is omitted, try 7777 through 7790 (14 attempts) using `net.Listen("tcp", ...)` then close. On first success, pass the open listener into the HTTP server. If all 14 fail, exit 2 with a message including the last error.
- **Rationale:** Bounded, fast, predictable; user can always force a specific port. Passing the already-open listener avoids the TOCTOU race where `Listen` succeeds in the probe and then fails when the real server binds the same port.
- **Rejected:** unbounded incrementing; random port with no default; OS-assigned ephemeral port (hard to remember / reconnect).

### ADR-E27-011: Browser launch is best-effort and behind `--no-open`

- **Status:** Accepted.
- **Context:** `xdg-open` / `open` / `start` behave differently on Linux, macOS, Windows, and don't exist in headless CI or over SSH.
- **Decision:**
  - After the server is listening, print the URL to stdout unconditionally: `Viewer ready at http://127.0.0.1:7777`.
  - Attempt `open`/`xdg-open`/`start` (platform-dispatched) **only if** stdin is a TTY **and** `--no-open` is not set.
  - Browser-launch failure is logged at warn level, never fatal.
- **Rationale:** graceful degradation over SSH, in CI, in containers. The URL is always the authoritative output; the browser-open is a nice-to-have.
- **Rejected:** always launching a browser (breaks SSH/CI); only launching with an explicit `--open` flag (worse interactive UX).

### ADR-E27-012: No persistent caching / websocket push — polling only

- **Status:** Accepted.
- **Context:** The viewer could use WebSockets / SSE to push status updates as soon as the CLI changes them. That is materially more complex (fan-out, reconnect logic, schema versioning) than the scope justifies.
- **Decision:** Read-only HTTP + a manual "Refresh" button in the SPA. Optional background polling every 10 seconds for the recent-activity feed only, implemented client-side via `setInterval`.
- **Rationale:** Scope discipline. Local dev tool used for a few minutes at a time; polling is fine. If a future epic needs push updates, it slots in without breaking ADR-003/004.
- **Rejected:** WebSockets, SSE, long-poll — all out of scope for an MVP read-only dashboard.

---

## 3. Data Model Changes

### 3.1 Database schema

**None.** E27 is purely additive at the presentation/composition layer. No new tables, no new columns, no new indexes, no migrations. `CurrentSchemaVersion` in `internal/db/db.go` is **not** bumped.

This is the main reason the epic PRD estimated this as ~5.5 days: there is no database work.

### 3.2 Response DTOs (new, in `internal/api/viewer/types.go`)

New Go structs for HTTP wire format only. None of these are persisted.

```go
// SummaryResponse — GET /api/v1/viewer/summary
type SummaryResponse struct {
    GeneratedAt time.Time        `json:"generated_at"`
    Epics       EntityRollup     `json:"epics"`
    Features    EntityRollup     `json:"features"`
    Tasks       TaskRollup       `json:"tasks"`
    Bugs        BugRollup        `json:"bugs"`
    ChangeCards EntityRollup     `json:"change_cards"`
}

type EntityRollup struct {
    Total        int            `json:"total"`
    StatusCounts []StatusCount  `json:"status_counts"`
}

type StatusCount struct {
    Status string `json:"status"`
    Count  int    `json:"count"`
    Color  string `json:"color"`
    Phase  string `json:"phase"`
}

type TaskRollup struct {
    EntityRollup
    BlockedCount int `json:"blocked_count"`
}

type BugRollup struct {
    EntityRollup
    SeverityCounts map[string]int `json:"severity_counts"`
}

// HierarchyResponse — GET /api/v1/viewer/hierarchy
type HierarchyResponse struct {
    Epics []HierarchyEpic `json:"epics"`
}

type HierarchyEpic struct {
    Key         string            `json:"key"`
    Title       string            `json:"title"`
    Status      string            `json:"status"`
    StatusColor string            `json:"status_color"`
    Features    []HierarchyFeature `json:"features"`
}

type HierarchyFeature struct {
    Key          string `json:"key"`
    Title        string `json:"title"`
    Status       string `json:"status"`
    StatusColor  string `json:"status_color"`
    TaskCount    int    `json:"task_count"`
    BlockedCount int    `json:"blocked_count"`
}

// HistoryResponse — GET /api/v1/viewer/history/{key}
type HistoryResponse struct {
    Key     string          `json:"key"`
    Entries []HistoryEntry  `json:"entries"` // newest first
}

type HistoryEntry struct {
    ID         int64     `json:"id"`
    FromStatus string    `json:"from_status"`
    ToStatus   string    `json:"to_status"`
    Agent      *string   `json:"agent,omitempty"`
    Notes      *string   `json:"notes,omitempty"`
    ChangedAt  time.Time `json:"changed_at"`
}

// FileResponse — GET /api/v1/viewer/file/{key}
type FileResponse struct {
    Key      string `json:"key"`
    Exists   bool   `json:"exists"`
    Path     string `json:"path,omitempty"`     // relative to project root
    Content  string `json:"content,omitempty"`  // raw markdown
}

// FeatureTasksResponse — GET /api/v1/viewer/features/{key}/tasks
type FeatureTasksResponse struct {
    FeatureKey string     `json:"feature_key"`
    Total      int        `json:"total"`
    Tasks      []TaskView `json:"tasks"`
}

type TaskView struct {
    Key            string  `json:"key"`
    Title          string  `json:"title"`
    Status         string  `json:"status"`
    StatusColor    string  `json:"status_color"`
    AgentType      *string `json:"agent_type,omitempty"`
    Priority       int     `json:"priority"`
    ExecutionOrder int     `json:"execution_order"`
    Blocked        bool    `json:"blocked"`
}

// RecentActivityResponse — GET /api/v1/viewer/recent-activity
type RecentActivityResponse struct {
    Limit   int               `json:"limit"`
    Entries []ActivityEntry   `json:"entries"`
}

type ActivityEntry struct {
    EntityType string    `json:"entity_type"` // "epic" | "feature" | "task" | "bug" | "change_card"
    Key        string    `json:"key"`
    Title      string    `json:"title"`
    FromStatus string    `json:"from_status"`
    ToStatus   string    `json:"to_status"`
    ChangedAt  time.Time `json:"changed_at"`
}

// WorkflowMetaResponse — GET /api/v1/viewer/workflow-meta
type WorkflowMetaResponse struct {
    Levels map[string]WorkflowLevelMeta `json:"levels"` // "epic", "feature", "task", "bug", "change_card"
}

type WorkflowLevelMeta struct {
    Statuses    []StatusMeta     `json:"statuses"`
    Transitions []TransitionMeta `json:"transitions"`
}

type StatusMeta struct {
    Name           string `json:"name"`
    Color          string `json:"color"`
    Phase          string `json:"phase"`
    ProgressWeight int    `json:"progress_weight"`
}

type TransitionMeta struct {
    From      string `json:"from"`
    To        string `json:"to"`
    Direction string `json:"direction"` // "forward" | "backward" | "lateral"
}
```

**Internal invariant:** no response struct exposes `ID int64` or foreign keys. Every reference uses the business `key`, per `http-integration.md` §"Exposing Internal Details".

### 3.3 Filter option types (in `internal/api/viewer/service.go`)

```go
type FeatureTaskOptions struct {
    Status    string
    AgentType string
    Blocked   *bool
    Limit     int
    Offset    int
}

type RecentActivityOptions struct {
    Limit      int        // default 50, max 200
    EntityType string     // optional filter
    Since      *time.Time // optional; RFC3339 in query string
}
```

Validation of these options (bounds, enum membership) happens at the **service boundary**, not in the handler, consistent with `services/service-design.md`.

---

## 4. Integration Approach

### 4.1 How new code plugs into existing code

Integration happens at exactly four seams. Each is a one-line change.

1. **`cmd/server/main.go:44`**
   - Before: `database, err := db.InitDB("shark-tasks.db")`
   - After:  `repoDB, err := dbinit.Init(context.Background(), dbinit.Options{})`
   - Remove the `database.Close()` defer; `repository.DB` exposes a `Close()` method that the runner defers.

2. **`cmd/server/services.go::WireServices`**
   - Add a line that constructs `ViewerService` with the same repositories already being constructed for Epic/Feature/Task services. No new interfaces leak into `ServiceContainer` unless we decide to expose it — viewer wiring can stay local to a helper `wireViewer(repoDB, workflowSvc) *services.ViewerService` called from `main.go`.

3. **`cmd/server/main.go::main` (route registration)**
   - After the existing three `RegisterRoutes` calls, add:
     ```go
     viewer.NewHandler(viewerSvc).RegisterRoutes(mux)
     viewer.RegisterStaticAssets(mux) // mounts GET / and GET /static/
     ```
   - The `GET /` stub handler currently at line 67 is removed (or replaced by `RegisterStaticAssets`, which mounts the same path).

4. **`internal/cli/commands/web.go` (NEW)**
   - ~30 LOC. Registers `shark web` with Cobra. Calls `runner.Run(cmd.Context(), runner.Options{Port: port, OpenBrowser: !noOpen})`. Delegates 100% of runtime behavior to `internal/viewer/runner`.

### 4.2 Reuse of existing infrastructure

| Existing piece | How viewer uses it |
|---|---|
| `internal/repository.EpicRepository` / `FeatureRepository` / `TaskRepository` / `TaskHistoryRepository` / bug + change-card repos | Composed via `ViewerService` constructor. No new repo methods needed for Phase 1; if any are needed (e.g. `ListRecentHistory(limit, since)`), they are added to the repository, not to the service. |
| `workflow.Service` | Used by `ViewerService.WorkflowMeta()` and by the color/phase enrichment of every status on every response. Loads from `.sharkconfig.json` exactly as CLI does. |
| `status.CalculationService` | Reused by `ViewerService.Hierarchy()` to compute `BlockedCount` / weighted progress per feature. No duplication of the calculation formula. |
| `internal/api/common.go` (`respondJSON`, `respondError`, `handleServiceError`, `pathParam`) | Re-exported by importing `internal/api` from `internal/api/viewer`. Or moved to `internal/api/httputil` if we want to avoid a sub-package importing its parent — decomposer's call. (Recommend: move to `internal/api/httputil` as a tiny mechanical refactor in Phase 2.) |
| `internal/observability` (otelhttp middleware) | Inherited automatically — the whole mux is wrapped at `cmd/server/main.go:92`, so viewer routes get tracing and metrics for free. |

### 4.3 Security perimeter

- **Bind address:** always `127.0.0.1`, never `0.0.0.0`. The server is only reachable from the local machine. Exposing it to a LAN is a separate epic and must not happen implicitly.
- **CORS:** per ADR-007, locked to `localhost`/`127.0.0.1` origins.
- **File serving:** per ADR-008, DB-path-first with project-root containment.
- **No auth:** local tool for a single-user machine. If multi-user arrives, auth is a separate epic — explicitly out of scope.
- **No writes:** per ADR-003, viewer endpoints are read-only. The CRUD endpoints (`/api/v1/tasks/...`) are already there and already unauthenticated, so this is a property of the viewer scope, not a new vulnerability.

---

## 5. Migration Strategy

### 5.1 Rollout order

The epic PRD already describes four phases. The architecture confirms they are independent and can be delivered/reviewed as four separate features:

1. **F1 — `internal/dbinit` extraction + `cmd/server` migration.** Pure refactor. After this, `cmd/server` supports Turso automatically. Acceptance: existing `cmd/server` tests still pass; new test covers local+turso+failing-config branches.
2. **F2 — Viewer API.** `internal/api/viewer` + `services.ViewerService` + tests. Acceptance: 7 endpoints return correct JSON against a seeded test DB via handler tests with mocked service, plus one integration test covering the full stack on a temp SQLite DB.
3. **F3 — SPA.** `internal/viewer/assets/viewer.html` + `internal/viewer/assets.go` with `go:embed`. Acceptance: curl `/` returns HTML; `/static/vendor/marked.min.js` returns JS; manual smoke test against seeded DB.
4. **F4 — `shark web` CLI.** `internal/viewer/runner` + `internal/cli/commands/web.go`. Acceptance: `./bin/shark web --no-open` starts, prints URL, responds on `/health`, shuts down cleanly on SIGINT.

F1 is the only prerequisite for all others. F2/F3/F4 can parallelize after F1 merges.

### 5.2 Backward compatibility

- **Existing CRUD endpoints:** unchanged. Any existing client (tests, curl scripts, future API consumers) keeps working byte-for-byte.
- **`cmd/server/main.go` entrypoint:** same binary name, same graceful-shutdown behavior, same observability hook. Only the DB init line changes.
- **`internal/cli` callers of `initDatabase`:** unchanged — `initDatabase` becomes a one-line delegator. No CLI commands need to be touched.
- **Databases:** no schema change. Existing local SQLite and Turso databases work with the new code immediately.

### 5.3 Rollback

Each phase is a separate merge. F1 is revertible by restoring the single-line `db.InitDB("shark-tasks.db")`. F2–F4 are strictly additive; reverting them is a file deletion. No data to migrate back.

### 5.4 Observability

- All viewer endpoints are wrapped by `otelhttp` automatically (existing wiring).
- Add `slog.Info` for: server start, port binding, `shark web` URL, graceful shutdown. Match existing server log style.
- No new metrics in Phase 1. If the recent-activity polling proves chatty, add a request-count metric in a follow-up.

---

## 6. Alignment With CLAUDE.md Conventions

This architecture is verified against every rule the auto-loaded CLAUDE.md snippets specify:

| Rule | Adherence |
|---|---|
| **Commands → Services → Repositories** (`architecture.md`) | `shark web` → `runner.Run` → composes `ViewerService`. `api/viewer.Handler` → `ViewerServicer` → repositories. No layer skipping. |
| **Thin HTTP handlers, fat services** (`services/http-integration.md`) | Every endpoint handler is the 3-step pattern: parse → call → format. Aggregation lives in `ViewerService`. |
| **Repositories are dumb** (`architecture.md` anti-patterns) | `ViewerService` composes; repositories only add new `SELECT` methods if needed (e.g., `TaskHistoryRepository.ListRecent(limit, since, entityType)`), never business logic. |
| **Interface at point of use** (`go/patterns.md` §Interface Design) | `ViewerServicer` declared in `api/viewer`, not in `services`. |
| **Constructor DI, no framework** (`architecture.md` §2) | `services.NewViewerService(epicRepo, featureRepo, taskRepo, historyRepo, bugRepo, changeRepo, workflowSvc, statusCalc)`. |
| **`context.Context` first param** (`go/patterns.md`) | Every `ViewerService` method takes `ctx context.Context` first. |
| **Error wrapping with `%w`** (`go/error-handling.md`) | All service errors wrap repository errors with business context. Handler maps typed errors to status codes via `handleServiceError`. |
| **No hardcoded status lists** (`go/patterns.md` §Validation) | Workflow meta endpoint reads from `workflow.Service`. Status color enrichment also reads from the workflow config, never from hardcoded maps. The "reference palette" in the epic PRD is the UI *default* — overridden by `status_metadata` in `.sharkconfig.json` whenever present. |
| **Parameterized queries only** (`go/input-sanitization.md`) | Viewer adds no new SQL; reuses repository methods that already use `?` placeholders. |
| **Key format regex validation** (`go/input-sanitization.md`) | Handlers validate `{key}` path params via existing `internal/models/validation.go` regexes before calling the service. |
| **File path security** (`go/input-sanitization.md`) | ADR-008 enforces DB-path-first + containment check. |
| **Test repositories with real DB, everything else with mocks** (`testing/architecture.md`) | `handler_test.go` mocks `ViewerServicer`. `viewer_service_test.go` mocks all repository interfaces. One integration test uses `test.GetTestDB` per the repository-test rule. |
| **Service layer is entry-point agnostic** (`services/service-design.md`) | `ViewerService` has no Cobra, no HTTP, no `net/http` imports. Same service would serve a future TUI or gRPC transport. |
| **Never delete database / no migrations without version bump** (`database-critical.md`) | No migrations, no schema version change. Rule trivially satisfied. |
| **CLI commands are thin wrappers** (`cli/patterns.md`) | `shark web` parses `--port`/`--no-open`, calls `runner.Run`, formats stdout. ~30 LOC. |

---

## 7. Out of Scope (explicit non-goals)

Calling these out so they don't creep into decomposition:

- **Authentication / authorization** — local single-user tool. Binds to `127.0.0.1`, done.
- **Write operations from the UI** — no "approve task" / "advance status" buttons in Phase 1. The viewer is a reader. Mutation is via CLI.
- **Historical diff visualization** — no blame view, no "what changed between yesterday and today" panel.
- **Multi-project aggregation** — one shark project per server instance.
- **WebSocket / SSE push** — see ADR-012.
- **Theme customization** — dark only. Light theme is a future concern.
- **Export (PDF / CSV)** — not in Phase 1.
- **Mobile / responsive layout beyond "does not crash on narrow widths"** — developer workstation target.

---

## 8. Decomposition Hints (for PM / BA)

Suggested feature split (PM has final say):

1. **F1 — Shared DB init package** (1 feature, ~1 day). Blocker for all others.
2. **F2 — Viewer API & service** (1 feature, ~2 days). Contains the 7 endpoints, `ViewerService`, mocks, handler tests.
3. **F3 — Viewer SPA & static asset serving** (1 feature, ~2 days). The HTML + embedded JS + `GET /` + `GET /static/` routes.
4. **F4 — `shark web` command & runner** (1 feature, ~0.5 day). The CLI command, the runner, the browser-open helper.

Total ~5.5 days matches the epic PRD estimate. Each feature has its own PRD, acceptance tests, and can ship/merge independently after F1 lands.

---

*Architecture sign-off pending feature review gate.*
