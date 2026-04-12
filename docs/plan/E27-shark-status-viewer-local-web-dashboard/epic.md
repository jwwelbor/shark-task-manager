---
epic_key: E27
title: Shark Status Viewer - Local Web Dashboard
description: A self-contained, dark-themed web dashboard that reads project state from the shark database (local SQLite or Turso cloud) and renders an IDE-style status viewer with sidebar tree, spec content, transition history, and dashboard summary. Launched via `shark web` from any project directory.
---

# E27 — Shark Status Viewer — Local Web Dashboard

**Epic Key**: E27  
**Status**: ready_for_research  
**Priority**: Medium

---

## Goal

### Problem

Developers and AI agents working with shark have no visual way to browse the project hierarchy, read spec documents, and understand transition history without issuing multiple CLI commands. The `shark status`, `shark get`, and `shark list` commands provide data but no unified context — you can't see a task's spec content alongside its history, or quickly scan all features at a glance.

### Solution

Add a `shark web` command that starts a local HTTP server and opens a browser to an IDE-style dark dashboard. The viewer uses the same service layer as the CLI (including Turso cloud support) and serves a single-file SPA with no build step. Three views: dashboard overview, entity detail with spec, and transition history.

### Impact

- Humans can visually navigate and review project state without memorizing CLI commands
- Spec documents readable inline alongside status/history context
- Agent operators can audit transition history for any entity at a glance
- Works from any directory in a shark project (auto-detects project root)

---

## Architecture Decision

**Option B — Extend existing `cmd/server`** was selected after evaluating four alternatives.

Rationale:
- Server binary already exists with Epic, Feature, Task CRUD handlers
- Service layer already wired (WireServices)
- No new binaries, no code duplication
- Single prerequisite: extract DB init into `internal/dbinit` so server gains cloud/Turso support

---

## Implementation Phases

### Phase 1 — DB Init Extraction (prerequisite, ~1 day)
Extract `initDatabase()` from `internal/cli/db_init.go` into `internal/dbinit/init.go`.  
Modify `cmd/server/main.go` to call `dbinit.Init(opts)` instead of hardcoding `shark-tasks.db`.  
Server now reads `.sharkconfig.json` and supports Turso automatically.

### Phase 2 — Viewer API Endpoints (~2 days)
New package: `internal/api/viewer/`
- `handler.go` — ViewerHandler with 7 endpoints
- `service.go` — ViewerServicer interface
- `types.go` — all request/response structs
- `handler_test.go` — tests with mocked service

New service: `internal/services/viewer_service.go` — ViewerService implements ViewerServicer.

Endpoints:
```
GET /api/v1/viewer/summary              → dashboard rollup (epics/features/tasks/bugs counts)
GET /api/v1/viewer/hierarchy            → full project tree (epics → features with task counts)
GET /api/v1/viewer/history/{key}        → status change audit trail for any entity
GET /api/v1/viewer/file/{key}           → raw markdown content from entity file_path
GET /api/v1/viewer/features/{key}/tasks → full task list for one feature with filters
GET /api/v1/viewer/recent-activity      → N most recent status changes across all types
GET /api/v1/viewer/workflow-meta        → loaded workflow definition (statuses, transitions, colors)
```

### Phase 3 — Single-File SPA (~2 days)
`internal/viewer/assets/viewer.html` — dark-themed IDE-style interface, embedded in server binary via `go:embed`.

Served at `GET /` from `cmd/server/`.

Features:
- **Left sidebar**: collapsible tree (Epics → Features → Tasks), status dots with color coding
- **Dashboard view**: summary pill cards, feature progress bars, recent activity feed
- **Entity view**: properties panel + markdown spec content (Marked.js CDN) + History button
- **History view**: reverse-chronological status change table with colored badges
- **Top nav**: entity count tabs (Epics, Features, Tasks, Bugs), Refresh button
- Keyboard: Escape closes panels, arrow keys navigate tree

Status color palette (from user reference):
- gray: draft, todo, cancelled, closed
- yellow: in_progress, in_development, triaged
- red: blocked, rejected
- green: completed, done, accepted, approved, resolved
- blue: proposed, in_review
- magenta: ready_for_review, ready_for_approval

### Phase 4 — `shark web` Command (~0.5 day)
New file: `internal/cli/commands/web.go`
- Registers `shark web [--port PORT]`
- Picks a free port starting at 7777 if not specified
- Starts viewer server (or reuses cmd/server binary)
- Calls `xdg-open` (Linux) / `open` (macOS) to open the browser
- Prints URL to stdout

---

## New Files

```
internal/dbinit/
└── init.go                          NEW — shared DB init (FindProjectRoot, Init, MustInit)

internal/api/viewer/
├── handler.go                       NEW — ViewerHandler, RegisterRoutes, 7 handlers
├── service.go                       NEW — ViewerServicer interface + filter/options types
├── types.go                         NEW — all response structs (see API Contract below)
└── handler_test.go                  NEW — tests with MockViewerServicer

internal/services/
└── viewer_service.go                NEW — ViewerService implements ViewerServicer

internal/viewer/assets/
└── viewer.html                      NEW — single-file SPA, embedded via go:embed

internal/cli/commands/
└── web.go                           NEW — `shark web` command
```

Modified:
```
cmd/server/main.go                   MODIFY — use dbinit.Init, serve GET /, register viewer routes
cmd/server/services.go               MODIFY — wire ViewerService into ServiceContainer
```

---

## API Contract

### `GET /api/v1/viewer/summary`
Returns dashboard rollup counts for all entity types with status breakdown and colors.

**Response 200:**
```json
{
  "generated_at": "2026-04-11T14:32:00Z",
  "epics": {
    "total": 7,
    "status_counts": [{"status": "active", "count": 5, "color": "yellow", "phase": "development"}]
  },
  "features": {"total": 34, "status_counts": [...]},
  "tasks": {"total": 212, "blocked_count": 3, "status_counts": [...]},
  "bugs": {"total": 8, "status_counts": [...], "severity_counts": {"critical": 1, "high": 3}}
}
```

### `GET /api/v1/viewer/hierarchy`
Full project tree — all epics with nested features and task count rollups. Used to build the sidebar.

### `GET /api/v1/viewer/history/{key}`
Status change audit trail for any entity key (`E07`, `E07-F01`, `E07-F01-001`, `B001`, `CC-001`). Entries newest-first.

### `GET /api/v1/viewer/file/{key}`
Raw markdown content of the entity's spec file. Returns `exists: false` if file is missing on disk (entity still exists in DB).

**Security**: path resolved from DB record only, never from user input. Validated to stay within project root.

### `GET /api/v1/viewer/features/{key}/tasks`
Full task list for a feature with optional filters: `?status=`, `?agent=`, `?blocked=true`, `?limit=`, `?offset=`.

### `GET /api/v1/viewer/recent-activity`
N most recent status changes across all entity types. Optional filters: `?limit=`, `?entity_type=`, `?since=` (RFC3339).

### `GET /api/v1/viewer/workflow-meta`
Complete workflow definition serialized for UI consumption — all statuses with colors/phases/weights, all transitions with direction (forward/backward).

---

## Key Constraints

- **No build step** — viewer.html is vanilla JS + Marked.js from CDN
- **No new dependencies** (Go side) — uses existing service layer
- **CORS headers** — server must set `Access-Control-Allow-Origin: http://localhost:*` for local dev
- **File path security** — `GET /viewer/file/{key}` resolves path from DB, validates within project root
- **Turso support** — inherits from dbinit extraction in Phase 1
- **Port** — defaults to 7777, auto-increments if in use

---

## Success Criteria

- `shark web` starts server and opens browser within 2 seconds
- Hierarchy loads in < 500ms for projects with 500 tasks
- Spec documents render as formatted markdown
- Transition history shows all entries with status badges
- Works from any subdirectory in a shark project
- Works with both local SQLite and Turso cloud

---

*Last Updated*: 2026-04-11  
*Architect*: Claude (E27 architecture session)
