---
feature_key: E27-F02-viewer-api-endpoints-read-only-dashboard-data-laye
epic_key: E27
doc_type: specification
status: draft
author: architect
---

# E27-F02 — Specification: Viewer API Endpoints (Read-Only Dashboard Data Layer)

**Feature:** E27-F02 — Viewer API Endpoints - Read-Only Dashboard Data Layer
**Epic:** E27 — Shark Status Viewer — Local Web Dashboard
**Dependency:** **E27-F01 (`internal/dbinit` extraction) MUST be merged first.**

This document combines functional/non-functional requirements with the technical architecture for a single shippable feature. It is bound by, and must not re-litigate, the ADRs in [`../../architecture.md`](../../architecture.md) — particularly ADR-E27-003 (read-only), ADR-E27-004 (interface at consumer), ADR-E27-007 (CORS), and ADR-E27-008 (file serving security).

---

## 1. Overview

### 1.1 What this feature adds

This feature introduces two new Go packages and one new interface:

| Package / File | Purpose |
|---|---|
| `internal/api/viewer/` | HTTP handler package (`handler.go`, `service.go`, `types.go`, `cors.go`, `handler_test.go`). Owns the `ViewerServicer` interface, the seven route registrations, request/response DTOs, and the localhost CORS middleware. |
| `internal/services/viewer_service.go` | The concrete `ViewerService` — a read-only service that composes existing Epic/Feature/Task/Bug/ChangeCard repositories, `workflow.Service`, and `status.CalculationService` to produce dashboard-shaped aggregates. |
| `cmd/server/services.go` (modified) | Adds `ViewerService` to `ServiceContainer` and wires its dependencies inside `WireServices`. |

Seven endpoints are mounted under `/api/v1/viewer/`. They are **strictly read-only** — no `POST`, `PATCH`, `PUT`, or `DELETE` routes are added. Any mutation needed by the UI goes through the existing CRUD endpoints (`/api/v1/tasks/...`, etc.), not through viewer endpoints (ADR-E27-003).

### 1.2 Why

The viewer SPA (F3) needs JSON-shaped aggregates that the existing CRUD endpoints cannot efficiently produce (they return one entity at a time, not roll-ups). Reusing CRUD endpoints would force the SPA to make 30+ round-trips just to render a dashboard, driving up latency and complicating client code. A thin read-only composition layer solves this once, on the server, where repositories are already wired.

### 1.3 Scope boundary

**In scope (this feature):**
- 7 JSON endpoints under `/api/v1/viewer/`
- `ViewerService` read-only composition
- `ViewerServicer` interface + `MockViewerServicer` for handler tests
- Localhost-only CORS middleware (ADR-E27-007)
- DB-path-first file serving with project-root containment (ADR-E27-008)
- Validation of `{key}` path params via existing `internal/models/validation.go` regexes
- One new `EntityHistoryRepository.ListRecentAcrossEntities(ctx, opts)` method (minimal, bounded; see §4.4)

**Out of scope (covered by other features):**
- F1 (`internal/dbinit` extraction) — prerequisite, already in progress
- F3 (SPA HTML + `/static/` asset serving) — UI consumes but does not build these endpoints
- F4 (`shark web` CLI command + runner) — does not modify viewer endpoints
- Authentication, write endpoints, WebSocket push, multi-project aggregation

---

## 2. Requirements

### 2.1 Functional requirements

**Category: Summary Endpoint**

1. **REQ-F-001 — Entity-type counts with status breakdowns**
   - **Description:** `GET /api/v1/viewer/summary` MUST return total counts and per-status counts for all five entity types (epics, features, tasks, bugs, change-cards). Each status entry MUST include the workflow-configured `color` and `phase` so the UI does not need a second round-trip.
   - **Priority:** Must-Have
   - **Acceptance criteria:**
     - [ ] Response contains `epics`, `features`, `tasks`, `bugs`, `change_cards` top-level keys with the shapes defined in §3.1.
     - [ ] `tasks.blocked_count` equals the count of tasks whose `blocked = 1` in the DB regardless of status.
     - [ ] `bugs.severity_counts` is a map of severity → count; absent severities are omitted rather than emitted as 0.
     - [ ] `status_counts[].color` and `status_counts[].phase` are sourced from `workflow.Service` for the entity's level; if an entity's status is not present in the workflow (legacy data), `color = "gray"`, `phase = "unknown"`.
     - [ ] `generated_at` is the server-side request timestamp in RFC3339Nano UTC.
     - [ ] Empty projects return all zeros without error (not a 404).

**Category: Hierarchy Endpoint**

2. **REQ-F-002 — Full epic → feature tree for sidebar**
   - **Description:** `GET /api/v1/viewer/hierarchy` MUST return every epic with its features, in execution order, including per-feature task count and blocked count. Tasks themselves are not included in this payload (the SPA fetches them lazily via `/features/{key}/tasks`).
   - **Priority:** Must-Have
   - **Acceptance criteria:**
     - [ ] Epics ordered by `execution_order ASC, created_at ASC`.
     - [ ] Features within an epic ordered by `execution_order ASC, created_at ASC`.
     - [ ] Each feature exposes `task_count` (total tasks, any status) and `blocked_count` (tasks with `blocked = 1`).
     - [ ] Each entity exposes `status` and `status_color` (resolved from workflow config, same fallback rule as REQ-F-001).
     - [ ] A project with 0 epics returns `{"epics": []}` with HTTP 200, not an error.

**Category: History Endpoint**

3. **REQ-F-003 — Unified status history for any entity type**
   - **Description:** `GET /api/v1/viewer/history/{key}` MUST return the status-change audit trail for any supported entity (epic, feature, task, bug, change-card). Entity type is inferred from the key format (no second path segment).
   - **Priority:** Must-Have
   - **Acceptance criteria:**
     - [ ] Entity type detection uses the same regexes as `internal/models/validation.go` — `E\d{2}`, `E\d{2}-F\d{2}`, `T?-?E\d{2}-F\d{2}-\d{3}`, `B\d{3}`, `CC-\d{3}`.
     - [ ] Keys are normalized to uppercase before lookup (case-insensitive, per CLAUDE.md).
     - [ ] Entries returned newest-first (DESC by `changed_at`).
     - [ ] Each entry includes `id`, `from_status`, `to_status`, optional `agent`, optional `notes`, `changed_at` (RFC3339Nano UTC).
     - [ ] Invalid key format returns HTTP 400 with `{"error": "Bad Request", "message": "invalid entity key: <key>"}`.
     - [ ] Unknown but valid-format key returns HTTP 404.

**Category: File Endpoint**

4. **REQ-F-004 — Raw markdown content of an entity's spec file**
   - **Description:** `GET /api/v1/viewer/file/{key}` MUST return the raw markdown of the entity's file on disk. The path MUST be resolved from the DB's `file_path` column (never from user input) and MUST be contained within the project root (ADR-E27-008).
   - **Priority:** Must-Have
   - **Acceptance criteria:**
     - [ ] Service reads `file_path` from the DB row, joins with the project root, canonicalizes with `filepath.Abs` + `filepath.EvalSymlinks`, then verifies the canonical result is a prefix match of `rootCanon + string(os.PathSeparator)`.
     - [ ] Path-traversal attempt (`../` in DB) returns HTTP 403 with `{"error": "Forbidden", "message": "file path escapes project root"}`.
     - [ ] Missing file on disk (DB row exists, file deleted) returns **HTTP 200** with `{"key": "...", "exists": false}`. The UI surfaces this gracefully instead of treating it as 404.
     - [ ] Successful read returns `{"key": "...", "exists": true, "path": "docs/plan/...", "content": "..."}` where `path` is relative to the project root.
     - [ ] File size > 2 MB returns HTTP 413 `{"error": "Payload Too Large", "message": "file exceeds 2 MiB limit"}` to prevent memory exhaustion from rogue DB paths.

**Category: Feature Tasks Endpoint**

5. **REQ-F-005 — Filterable task list for a feature**
   - **Description:** `GET /api/v1/viewer/features/{key}/tasks` MUST return all tasks in a feature, optionally filtered by status, agent type, and blocked flag, with client-paginated limit/offset.
   - **Priority:** Must-Have
   - **Acceptance criteria:**
     - [ ] Feature key validated with feature regex; invalid → 400; unknown but valid → 404.
     - [ ] Tasks ordered by `execution_order ASC, priority DESC, created_at ASC`.
     - [ ] Query params: `status`, `agent`, `blocked` (`true`/`false`), `limit` (default 200, max 500), `offset` (default 0).
     - [ ] Out-of-range `limit` clamps to max instead of erroring (tolerant read API).
     - [ ] Each task includes `status_color` resolved from workflow config.
     - [ ] Response includes `total` (unfiltered count for pagination) and `tasks` (filtered + paginated).

**Category: Recent Activity Endpoint**

6. **REQ-F-006 — Recent status changes across all entity types**
   - **Description:** `GET /api/v1/viewer/recent-activity` MUST return the N most recent status changes across epics, features, tasks, bugs, and change-cards, optionally filtered by entity type and time window.
   - **Priority:** Must-Have
   - **Acceptance criteria:**
     - [ ] Default `limit = 50`, max 200 (clamp, don't error).
     - [ ] Optional query params: `limit`, `entity_type` (one of `epic`, `feature`, `task`, `bug`, `change_card`), `since` (RFC3339).
     - [ ] Invalid `entity_type` value → HTTP 400.
     - [ ] Malformed `since` value → HTTP 400.
     - [ ] Entries include `entity_type`, `key`, `title`, `from_status`, `to_status`, `changed_at` sorted DESC by `changed_at`.
     - [ ] If a history row references an entity that has since been deleted, it is **omitted** (not returned with `title = null`).

**Category: Workflow Metadata Endpoint**

7. **REQ-F-007 — Full workflow definition for the UI color/phase tables**
   - **Description:** `GET /api/v1/viewer/workflow-meta` MUST return the complete workflow definition — statuses per level with color/phase/weight, and valid transitions per level — so the UI can render status badges and transition graphs without hardcoding any values.
   - **Priority:** Must-Have
   - **Acceptance criteria:**
     - [ ] Response includes a `levels` map keyed by `"epic"`, `"feature"`, `"task"`, `"bug"`, `"change_card"`.
     - [ ] Each level contains `statuses[]` (name + color + phase + progress_weight) and `transitions[]` (from + to + direction).
     - [ ] `direction` is one of `forward` | `backward` | `lateral`, computed by comparing the ordinal of each status in the level's status list.
     - [ ] Values come from `workflow.Service`. No hardcoded status/color constants anywhere in this feature.
     - [ ] If the workflow config is missing a level, that level is emitted as an empty object (not omitted) so the UI has a stable shape.

### 2.2 Non-functional requirements

**Performance**

1. **REQ-NF-001 — Summary endpoint latency**
   - **Target:** p95 < 150 ms on a project with 10 epics × 10 features × 10 tasks × 10 history entries each (≈10k rows).
   - **Measurement:** in-process handler test with seeded fixture DB on commodity hardware.
   - **Justification:** SPA loads this on every page refresh; 150 ms is imperceptible.

2. **REQ-NF-002 — Hierarchy endpoint latency**
   - **Target:** p95 < 300 ms on the same 10k-row fixture.
   - **Measurement:** same as REQ-NF-001.
   - **Justification:** sidebar must feel instant on navigation.

3. **REQ-NF-003 — File endpoint memory bound**
   - **Target:** Reject files > 2 MiB with HTTP 413 (REQ-F-004). Read via `io.ReadAll` on a bounded `io.LimitReader`, never via unbounded buffer growth.
   - **Justification:** prevents memory exhaustion if a DB row points to a binary file.

**Security**

4. **REQ-NF-010 — Localhost-only CORS**
   - **Description:** Viewer endpoints MUST reject cross-origin requests from any origin other than `http://localhost:*` or `http://127.0.0.1:*`, per ADR-E27-007.
   - **Implementation:** `withLocalCORS` middleware in `internal/api/viewer/cors.go`. Echoes the `Origin` header only if its host matches; otherwise does not set `Access-Control-Allow-Origin` (browser then blocks the request). Applied only to `/api/v1/viewer/*` routes.
   - **Compliance:** OWASP ASVS V14 "CORS configuration".

5. **REQ-NF-011 — File path containment**
   - **Description:** File endpoint MUST enforce DB-path-first + canonicalize + prefix-check containment per ADR-E27-008. `file_path` is never concatenated with user input.
   - **Threat mitigated:** Path-traversal via maliciously crafted entity rows; symlink escape from project root.

6. **REQ-NF-012 — Read-only invariant**
   - **Description:** `ViewerService` MUST NOT expose any method that mutates state. All repository methods called by `ViewerService` are CRUD reads or existing query methods. No transactions are started by `ViewerService`.
   - **Verification:** Static check — `go vet` custom rule or a documented review item on the feature-review gate. A follow-up test asserts `ViewerService` methods do not return `error` from a write path (there are none).

7. **REQ-NF-013 — Key-format input validation**
   - **Description:** All `{key}` path params MUST pass the corresponding `internal/models/validation.go` regex (`ValidateEpicKey`, `ValidateFeatureKey`, `ValidateTaskKey`, `ValidateBugKey`, `ValidateChangeCardKey`) before any DB lookup. Invalid keys return 400, never reach the repository layer.

**Observability**

8. **REQ-NF-020 — Tracing**
   - **Description:** Viewer routes MUST inherit the existing `otelhttp` wrapping applied to the root mux at `cmd/server/main.go`. No new middleware needs to be added for tracing.
   - **Verification:** Handler test asserts presence of `traceparent` header propagation into the service call.

9. **REQ-NF-021 — Structured logging**
   - **Description:** Internal errors MUST be logged via `slog.Error` with `entity`, `endpoint`, and wrapped `error` keys. User-visible error messages MUST NOT leak repository/SQL internals.

### 2.3 Acceptance criteria — feature-level

**Scenario 1: Summary on empty project**
- **Given** a freshly initialized project with no epics, features, tasks, bugs, or change-cards
- **When** the SPA issues `GET /api/v1/viewer/summary`
- **Then** the response is HTTP 200 with all `total` fields = 0 and all `status_counts` arrays empty

**Scenario 2: Hierarchy with one epic, two features**
- **Given** an epic E01 with features E01-F01 (10 tasks, 2 blocked) and E01-F02 (3 tasks, 0 blocked)
- **When** the SPA issues `GET /api/v1/viewer/hierarchy`
- **Then** the response contains one epic with two features in execution order, `E01-F01.task_count=10`, `E01-F01.blocked_count=2`, `E01-F02.task_count=3`, `E01-F02.blocked_count=0`

**Scenario 3: File endpoint — path traversal rejected**
- **Given** a malicious DB row where `file_path = "../../etc/passwd"` (injected out-of-band)
- **When** the SPA issues `GET /api/v1/viewer/file/E01`
- **Then** the service canonicalizes the path, detects it escapes the project root, and returns HTTP 403 without reading any file

**Scenario 4: File endpoint — missing file on disk**
- **Given** an epic E01 whose `file_path` points to `docs/plan/E01/epic.md`, which has since been deleted
- **When** the SPA issues `GET /api/v1/viewer/file/E01`
- **Then** the response is HTTP 200 with `{"key":"E01", "exists":false}`

**Scenario 5: Recent activity with deleted entity**
- **Given** a history row for task T-E01-F01-001 that was later deleted
- **When** the SPA issues `GET /api/v1/viewer/recent-activity`
- **Then** the orphaned history row is omitted from the response (not returned with a null title)

**Scenario 6: Workflow meta on short workflow**
- **Given** `.sharkconfig.json` points at `shark-templates/.sharkworkflow-short.json`
- **When** the SPA issues `GET /api/v1/viewer/workflow-meta`
- **Then** the response's `levels.task.statuses[].name` matches the set defined in the short workflow file exactly, with no hardcoded extras

---

## 3. API Contract

All endpoints live under `/api/v1/viewer/`. All responses use `Content-Type: application/json; charset=utf-8`. All timestamps are RFC3339Nano UTC.

### 3.1 `GET /api/v1/viewer/summary`

**Request:** no path params, no query params, no body.

**Response 200:**
```json
{
  "generated_at": "2026-04-11T17:45:00.123456789Z",
  "epics": {
    "total": 5,
    "status_counts": [
      { "status": "active",    "count": 2, "color": "yellow", "phase": "development" },
      { "status": "completed", "count": 3, "color": "green",  "phase": "done" }
    ]
  },
  "features": {
    "total": 20,
    "status_counts": [
      { "status": "in_specification", "count": 4, "color": "blue",   "phase": "planning" },
      { "status": "active",           "count": 8, "color": "yellow", "phase": "development" },
      { "status": "completed",        "count": 8, "color": "green",  "phase": "done" }
    ]
  },
  "tasks": {
    "total": 150,
    "status_counts": [
      { "status": "todo",        "count": 40, "color": "gray",   "phase": "planning" },
      { "status": "in_progress", "count": 20, "color": "yellow", "phase": "development" },
      { "status": "completed",   "count": 90, "color": "green",  "phase": "done" }
    ],
    "blocked_count": 3
  },
  "bugs": {
    "total": 12,
    "status_counts": [
      { "status": "open",     "count": 7, "color": "red",   "phase": "triage" },
      { "status": "resolved", "count": 5, "color": "green", "phase": "done" }
    ],
    "severity_counts": { "critical": 1, "high": 3, "medium": 6, "low": 2 }
  },
  "change_cards": {
    "total": 4,
    "status_counts": [
      { "status": "draft",    "count": 1, "color": "gray",  "phase": "planning" },
      { "status": "approved", "count": 3, "color": "green", "phase": "done" }
    ]
  }
}
```

### 3.2 `GET /api/v1/viewer/hierarchy`

**Request:** no params.

**Response 200:**
```json
{
  "epics": [
    {
      "key": "E01",
      "title": "Authentication & Session Management",
      "status": "active",
      "status_color": "yellow",
      "features": [
        {
          "key": "E01-F01",
          "title": "OAuth2 login",
          "status": "active",
          "status_color": "yellow",
          "task_count": 10,
          "blocked_count": 2
        },
        {
          "key": "E01-F02",
          "title": "Session refresh",
          "status": "in_specification",
          "status_color": "blue",
          "task_count": 3,
          "blocked_count": 0
        }
      ]
    }
  ]
}
```

### 3.3 `GET /api/v1/viewer/history/{key}`

**Request:** `{key}` path param. Accepts epic, feature, task, bug, or change-card keys (case-insensitive).

**Response 200:**
```json
{
  "key": "E01-F01-001",
  "entries": [
    {
      "id": 42,
      "from_status": "in_progress",
      "to_status": "ready_for_review",
      "agent": "backend",
      "notes": "Implementation complete; awaiting review.",
      "changed_at": "2026-04-11T14:30:00.000000000Z"
    },
    {
      "id": 41,
      "from_status": "todo",
      "to_status": "in_progress",
      "agent": "backend",
      "changed_at": "2026-04-11T13:10:00.000000000Z"
    }
  ]
}
```

**Error responses:**
- `400` — invalid key format
- `404` — no entity with that key

### 3.4 `GET /api/v1/viewer/file/{key}`

**Request:** `{key}` path param (any supported entity type).

**Response 200 (file exists):**
```json
{
  "key": "E01-F01-001",
  "exists": true,
  "path": "docs/plan/E01/E01-F01/T-E01-F01-001.md",
  "content": "# Task: ...\n\n...raw markdown..."
}
```

**Response 200 (file missing on disk):**
```json
{
  "key": "E01-F01-001",
  "exists": false
}
```

**Error responses:**
- `400` — invalid key format
- `403` — `file_path` escapes project root after canonicalization
- `404` — no entity with that key
- `413` — file > 2 MiB

### 3.5 `GET /api/v1/viewer/features/{key}/tasks`

**Request:**
- `{key}` path param — feature key
- Query: `status`, `agent`, `blocked` (`true|false`), `limit` (default 200, max 500), `offset` (default 0)

**Response 200:**
```json
{
  "feature_key": "E01-F01",
  "total": 10,
  "tasks": [
    {
      "key": "E01-F01-001",
      "title": "Scaffold OAuth client",
      "status": "completed",
      "status_color": "green",
      "agent_type": "backend",
      "priority": 5,
      "execution_order": 1,
      "blocked": false
    }
  ]
}
```

**Error responses:**
- `400` — invalid feature key or malformed query param
- `404` — feature does not exist

### 3.6 `GET /api/v1/viewer/recent-activity`

**Request:** Query: `limit` (default 50, max 200), `entity_type` (optional), `since` (optional RFC3339).

**Response 200:**
```json
{
  "limit": 50,
  "entries": [
    {
      "entity_type": "task",
      "key": "E01-F01-001",
      "title": "Scaffold OAuth client",
      "from_status": "in_progress",
      "to_status": "ready_for_review",
      "changed_at": "2026-04-11T14:30:00.000000000Z"
    },
    {
      "entity_type": "feature",
      "key": "E01-F02",
      "title": "Session refresh",
      "from_status": "draft",
      "to_status": "in_specification",
      "changed_at": "2026-04-11T12:05:00.000000000Z"
    }
  ]
}
```

**Error responses:**
- `400` — invalid `entity_type` or malformed `since`

### 3.7 `GET /api/v1/viewer/workflow-meta`

**Request:** no params.

**Response 200:**
```json
{
  "levels": {
    "epic": {
      "statuses": [
        { "name": "draft",     "color": "gray",   "phase": "planning",    "progress_weight": 0 },
        { "name": "active",    "color": "yellow", "phase": "development", "progress_weight": 50 },
        { "name": "completed", "color": "green",  "phase": "done",        "progress_weight": 100 }
      ],
      "transitions": [
        { "from": "draft",  "to": "active",    "direction": "forward" },
        { "from": "active", "to": "completed", "direction": "forward" },
        { "from": "active", "to": "draft",     "direction": "backward" }
      ]
    },
    "feature": { "statuses": [...], "transitions": [...] },
    "task":    { "statuses": [...], "transitions": [...] },
    "bug":     { "statuses": [...], "transitions": [...] },
    "change_card": { "statuses": [...], "transitions": [...] }
  }
}
```

### 3.8 Error envelope

Every non-2xx response uses the existing `api.ErrorResponse`:
```json
{
  "error":   "Not Found",
  "message": "epic not found: E99",
  "details": []
}
```

### 3.9 HTTP status code mapping

| Condition | Status |
|---|---|
| Success | 200 |
| Invalid key format, malformed query param, malformed body | 400 |
| File path escapes project root | 403 |
| Entity not found | 404 |
| File > 2 MiB | 413 |
| Unexpected server error | 500 |

---

## 4. Architecture

### 4.1 New packages and files

```
internal/
├── api/
│   └── viewer/                    [NEW]
│       ├── handler.go             # ViewerHandler, RegisterRoutes, 7 endpoint funcs
│       ├── service.go             # ViewerServicer interface + filter option types
│       ├── types.go               # Response DTOs (shapes mirror §3)
│       ├── cors.go                # withLocalCORS middleware (ADR-E27-007)
│       └── handler_test.go        # Table-driven handler tests with MockViewerServicer
└── services/
    ├── viewer_service.go          [NEW] ViewerService (concrete)
    └── viewer_service_test.go     [NEW] Unit tests with mocked repositories

internal/repository/entityhistory/
└── repository.go                  [MODIFIED] adds ListRecentAcrossEntities (§4.4)

cmd/server/
└── services.go                    [MODIFIED] adds ViewerService to ServiceContainer
```

### 4.2 `ViewerServicer` interface (at point of use — ADR-E27-004)

Declared in `internal/api/viewer/service.go`:

```go
package viewer

import (
    "context"
    "time"
)

// ViewerServicer is the minimal contract the viewer HTTP handler depends on.
// The concrete implementation lives in internal/services/ViewerService.
// Defined here so handler tests can inject a MockViewerServicer without
// importing internal/services (matches the TaskServicer pattern in task_handler.go).
type ViewerServicer interface {
    Summary(ctx context.Context) (*SummaryResponse, error)
    Hierarchy(ctx context.Context) (*HierarchyResponse, error)
    History(ctx context.Context, key string) (*HistoryResponse, error)
    File(ctx context.Context, key string) (*FileResponse, error)
    FeatureTasks(ctx context.Context, featureKey string, opts FeatureTaskOptions) (*FeatureTasksResponse, error)
    RecentActivity(ctx context.Context, opts RecentActivityOptions) (*RecentActivityResponse, error)
    WorkflowMeta(ctx context.Context) (*WorkflowMetaResponse, error)
}

// FeatureTaskOptions — validated at the service boundary (service-design.md §4).
type FeatureTaskOptions struct {
    Status    string
    AgentType string
    Blocked   *bool
    Limit     int // clamped to [0, 500]; 0 => 200
    Offset    int // negative values clamped to 0
}

// RecentActivityOptions — validated at the service boundary.
type RecentActivityOptions struct {
    Limit      int        // clamped to [0, 200]; 0 => 50
    EntityType string     // "" | "epic" | "feature" | "task" | "bug" | "change_card"
    Since      *time.Time // nil => no lower bound
}
```

### 4.3 `ViewerService` concrete (in `internal/services/viewer_service.go`)

```go
package services

// ViewerService composes existing repositories into dashboard-shaped aggregates.
// Strictly read-only (ADR-E27-003). No transactions, no mutation methods.
type ViewerService struct {
    epicRepo       EpicRepository
    featureRepo    FeatureRepository
    taskRepo       TaskRepository
    bugRepo        BugRepository
    changeCardRepo ChangeCardRepository
    historyRepo    EntityHistoryRepository
    workflowSvc    *workflow.Service
    statusCalc     *status.CalculationService // for Hierarchy blocked_count / progress
    projectRoot    string                     // needed for File() containment check
}

func NewViewerService(
    epicRepo EpicRepository,
    featureRepo FeatureRepository,
    taskRepo TaskRepository,
    bugRepo BugRepository,
    changeCardRepo ChangeCardRepository,
    historyRepo EntityHistoryRepository,
    workflowSvc *workflow.Service,
    statusCalc *status.CalculationService,
    projectRoot string,
) *ViewerService {
    if epicRepo == nil || featureRepo == nil || taskRepo == nil ||
       bugRepo == nil || changeCardRepo == nil || historyRepo == nil ||
       workflowSvc == nil {
        panic("ViewerService: all required dependencies must be non-nil")
    }
    return &ViewerService{
        epicRepo: epicRepo, featureRepo: featureRepo, taskRepo: taskRepo,
        bugRepo: bugRepo, changeCardRepo: changeCardRepo,
        historyRepo: historyRepo, workflowSvc: workflowSvc,
        statusCalc: statusCalc, projectRoot: projectRoot,
    }
}
```

**Repository interfaces** are declared in the same file as tailored read-only interfaces (service-design.md §2). Example:

```go
type EpicRepository interface {
    ListAll(ctx context.Context) ([]*models.Epic, error)
    GetByKey(ctx context.Context, key string) (*models.Epic, error)
    CountByStatus(ctx context.Context) (map[string]int, error)
}
// Same pattern for FeatureRepository, TaskRepository, BugRepository,
// ChangeCardRepository, EntityHistoryRepository.
```

Concrete `*repository.EpicRepository` etc. implement these structurally via existing methods (adapters added only if signatures mismatch).

### 4.4 Repository additions

**`EntityHistoryRepository.ListRecentAcrossEntities`** — the only net-new repository method:

```go
// ListRecentAcrossEntities returns the N most recent entity_history rows across
// all entity types, joined with the entity table to populate title + key.
// Rows whose entity has been deleted are omitted (per REQ-F-006).
//
// Returns rows ordered by changed_at DESC.
func (r *EntityHistoryRepository) ListRecentAcrossEntities(
    ctx context.Context, opts RecentHistoryFilter,
) ([]*models.RecentHistoryEntry, error)

type RecentHistoryFilter struct {
    Limit      int        // required; caller has already clamped
    EntityType string     // optional; "" means any
    Since      *time.Time // optional; nil means any
}
```

Implementation uses a single `UNION ALL` query across `epics`, `features`, `tasks`, `bugs`, `change_cards` joined to `entity_history`, `ORDER BY changed_at DESC LIMIT ?`. Indexes on `entity_history(changed_at)` and `(entity_type, entity_id)` already exist; no schema change.

**No other repository additions.** Existing methods cover Summary (already have `Count*` helpers), Hierarchy (List + Count), History (`ListByEntity`), File (`GetByKey` + `file_path`), and FeatureTasks (existing task query with filter options).

### 4.5 Handler structure

`internal/api/viewer/handler.go`:

```go
package viewer

type Handler struct {
    svc ViewerServicer
}

func NewHandler(svc ViewerServicer) *Handler {
    if svc == nil {
        panic("viewer.Handler: svc is required")
    }
    return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
    // Apply the localhost CORS middleware to every viewer route.
    wrap := func(f http.HandlerFunc) http.HandlerFunc {
        return withLocalCORS(http.HandlerFunc(f)).(http.HandlerFunc)
    }
    mux.HandleFunc("GET /api/v1/viewer/summary",               wrap(h.Summary))
    mux.HandleFunc("GET /api/v1/viewer/hierarchy",             wrap(h.Hierarchy))
    mux.HandleFunc("GET /api/v1/viewer/history/{key}",         wrap(h.History))
    mux.HandleFunc("GET /api/v1/viewer/file/{key}",            wrap(h.File))
    mux.HandleFunc("GET /api/v1/viewer/features/{key}/tasks",  wrap(h.FeatureTasks))
    mux.HandleFunc("GET /api/v1/viewer/recent-activity",       wrap(h.RecentActivity))
    mux.HandleFunc("GET /api/v1/viewer/workflow-meta",         wrap(h.WorkflowMeta))
    // Preflight (OPTIONS) handled by the middleware itself.
}
```

Each endpoint handler is the canonical 3-step thin wrapper:

```go
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
    resp, err := h.svc.Summary(r.Context())
    if err != nil {
        handleViewerError(w, err)
        return
    }
    respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
    key := strings.ToUpper(r.PathValue("key"))
    if err := validateAnyEntityKey(key); err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }
    resp, err := h.svc.History(r.Context(), key)
    if err != nil {
        handleViewerError(w, err)
        return
    }
    respondJSON(w, http.StatusOK, resp)
}
```

`handleViewerError` extends `api.handleServiceError` with the extra mappings required by viewer: `*viewer.SecurityError` → 403, `*viewer.FileTooLargeError` → 413. These sentinel types live in `service.go`:

```go
type SecurityError struct{ Reason string }
func (e *SecurityError) Error() string { return "security: " + e.Reason }

type FileTooLargeError struct{ Size, Limit int64 }
func (e *FileTooLargeError) Error() string { return "file exceeds limit" }
```

### 4.6 CORS middleware (ADR-E27-007)

`internal/api/viewer/cors.go`:

```go
package viewer

import (
    "net/http"
    "net/url"
)

func withLocalCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if origin != "" {
            if u, err := url.Parse(origin); err == nil {
                h := u.Hostname()
                if h == "localhost" || h == "127.0.0.1" {
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

No origin match → header not set → browser blocks cross-origin request. Same-origin requests (from the embedded SPA) work unaffected.

### 4.7 Server wiring (in `cmd/server/services.go`)

`WireServices` gains a `ViewerService` block at the end:

```go
// Step 6: Viewer service (read-only aggregation)
bugRepoForViewer := repository.NewBugRepository(db)              // reuse existing repo
changeCardRepoForViewer := repository.NewChangeCardRepository(db)
viewerStatusCalc := status.NewCalculationService(db, workflowSvc)
viewerService := services.NewViewerService(
    epicRepo, featureRepo, taskRepo,
    bugRepoForViewer, changeCardRepoForViewer,
    entityHistoryRepo,
    workflowSvc, viewerStatusCalc,
    projectRoot,
)

return &ServiceContainer{
    // ... existing fields ...
    ViewerService: viewerService,
}
```

`ServiceContainer` gains a `ViewerService *services.ViewerService` field. `cmd/server/main.go` adds one line after the existing `RegisterRoutes` calls:

```go
viewer.NewHandler(svcs.ViewerService).RegisterRoutes(mux)
```

### 4.8 Layering — checked against CLAUDE.md

| Rule | Compliance |
|---|---|
| Commands → Services → Repositories | Handler → `ViewerServicer` (interface) → `ViewerService` (concrete) → repositories. No skipping. |
| Thin HTTP handlers | Every handler is parse → call → format, ≤ 20 LOC each. |
| Interface at consumer | `ViewerServicer` declared in `internal/api/viewer`, not in `internal/services`. |
| Constructor DI | `NewViewerService(...)` takes all deps explicitly. |
| `context.Context` first param | Every method signature starts with `ctx context.Context`. |
| No hardcoded status lists | All colors/phases/transitions sourced from `workflow.Service`. |
| Parameterized queries | Only new SQL is the recent-activity UNION query, which uses `?` placeholders. |
| Key-format validation | All `{key}` params validated via `internal/models/validation.go` before DB access. |
| File-path security | DB-path-first + `filepath.Abs` + `EvalSymlinks` + prefix check (ADR-E27-008). |

---

## 5. Implementation Plan

Ordered steps. Each step leaves the tree in a compilable, `make test && make lint` clean state.

### Phase A — Response DTOs and interface (no runtime behavior)

1. **Create `internal/api/viewer/types.go`**
   - Add all response DTO structs exactly matching §3: `SummaryResponse`, `EntityRollup`, `StatusCount`, `TaskRollup`, `BugRollup`, `HierarchyResponse`, `HierarchyEpic`, `HierarchyFeature`, `HistoryResponse`, `HistoryEntry`, `FileResponse`, `FeatureTasksResponse`, `TaskView`, `RecentActivityResponse`, `ActivityEntry`, `WorkflowMetaResponse`, `WorkflowLevelMeta`, `StatusMeta`, `TransitionMeta`.
   - No methods, no imports beyond `time`.

2. **Create `internal/api/viewer/service.go`**
   - Declare `ViewerServicer` interface (§4.2).
   - Declare `FeatureTaskOptions`, `RecentActivityOptions`, `SecurityError`, `FileTooLargeError`.

3. **Create `internal/api/viewer/cors.go`** per §4.6. Unit test with a request from `http://evil.example.com` (no header) and `http://localhost:5173` (header echoed).

### Phase B — Repository addition

4. **Add `ListRecentAcrossEntities` to `internal/repository/entityhistory/repository.go`** per §4.4.
   - UNION ALL across `epics`, `features`, `tasks`, `bugs`, `change_cards` joined on `entity_history`, `ORDER BY changed_at DESC LIMIT ?`.
   - Repository test with seeded fixture (real DB, per `testing/architecture.md` repository-test rule).

### Phase C — Viewer service (behind interface, mockable)

5. **Create `internal/services/viewer_service.go`**
   - Declare minimal repository interfaces (`EpicRepository`, `FeatureRepository`, `TaskRepository`, `BugRepository`, `ChangeCardRepository`, `EntityHistoryRepository` — tailored to the read methods this service needs).
   - Implement `NewViewerService` with nil-check panics on required deps.
   - Implement `Summary`: counts from each repo, enrich per status from workflow config, set `generated_at = time.Now().UTC()`.
   - Implement `Hierarchy`: `ListAll` epics → for each, list features → for each, `taskRepo.CountByFeatureID` + `CountBlockedByFeatureID`.
   - Implement `History`: regex-detect entity type, `GetByKey`, `historyRepo.ListByEntity(type, id)`, map to `HistoryEntry` newest-first.
   - Implement `File`: `GetByKey`, read `file_path`, `filepath.Abs(filepath.Join(root, file_path))`, `EvalSymlinks`, prefix-check, `os.Stat` for size, `io.ReadAll(io.LimitReader(f, 2*1024*1024 + 1))`, return 413 sentinel if exceeded.
   - Implement `FeatureTasks`: validate + clamp opts, list tasks in feature, apply filters, paginate.
   - Implement `RecentActivity`: validate + clamp opts, call `historyRepo.ListRecentAcrossEntities`, map to response.
   - Implement `WorkflowMeta`: iterate `workflow.Service.AllLevels()`, for each level dump statuses + transitions, compute direction via ordinal comparison.

6. **Create `internal/services/viewer_service_test.go`** — mock-based unit tests for each method. Per `testing/architecture.md`, mocks are function-field structs; see §6.

### Phase D — Handler

7. **Create `internal/api/viewer/handler.go`** with `Handler`, `NewHandler`, `RegisterRoutes`, and the seven thin-wrapper methods per §4.5.
   - `Summary`, `Hierarchy`, `WorkflowMeta`, `RecentActivity` — no path params.
   - `History`, `File`, `FeatureTasks` — path param with `validateAnyEntityKey` / `ValidateFeatureKey`.
   - `handleViewerError` function maps `*SecurityError` → 403, `*FileTooLargeError` → 413, key-not-found → 404, other → delegate to `api.handleServiceError`.

8. **Create `internal/api/viewer/handler_test.go`** — table-driven tests per endpoint using `MockViewerServicer`. Verify status codes, JSON shape, and error mapping. See §6.

### Phase E — Server wiring

9. **Modify `cmd/server/services.go`**
   - Add `ViewerService *services.ViewerService` field to `ServiceContainer`.
   - Add the wiring block in `WireServices` per §4.7.

10. **Modify `cmd/server/main.go`** — one new line after existing `RegisterRoutes` calls:
    ```go
    viewer.NewHandler(svcs.ViewerService).RegisterRoutes(mux)
    ```

11. **Run `make fmt && make lint && make test`** — zero failures.

### Phase F — Integration smoke test

12. **Add `cmd/server/main_test.go` addition (or new `cmd/server/viewer_integration_test.go`)** — one end-to-end test that:
    - Creates a temp project dir with a seeded SQLite DB (via `test.GetTestDB()`).
    - Starts the server via `httptest.NewServer(mux)`.
    - Issues GET to each of the 7 endpoints; asserts 200 + basic shape.
    - Per `testing/architecture.md`, this is the "one integration test on a temp DB" exception for a full-stack smoke.

### File-path checklist (for the developer)

| File | Action |
|---|---|
| `internal/api/viewer/types.go` | NEW |
| `internal/api/viewer/service.go` | NEW |
| `internal/api/viewer/cors.go` | NEW |
| `internal/api/viewer/handler.go` | NEW |
| `internal/api/viewer/handler_test.go` | NEW |
| `internal/services/viewer_service.go` | NEW |
| `internal/services/viewer_service_test.go` | NEW |
| `internal/repository/entityhistory/repository.go` | MODIFIED — add `ListRecentAcrossEntities` |
| `internal/repository/entityhistory/repository_test.go` | MODIFIED — test new method |
| `cmd/server/services.go` | MODIFIED — add `ViewerService` to container |
| `cmd/server/main.go` | MODIFIED — register viewer routes (1 line) |
| `cmd/server/viewer_integration_test.go` | NEW — end-to-end smoke |

---

## 6. Testing Strategy

### 6.1 Three layers per `testing/architecture.md`

| Test file | Layer | Uses real DB? |
|---|---|---|
| `internal/api/viewer/handler_test.go` | HTTP handler | **No** — mocks `ViewerServicer` |
| `internal/services/viewer_service_test.go` | Service | **No** — mocks all repositories + workflow |
| `internal/repository/entityhistory/repository_test.go` | Repository | **Yes** — real test DB with cleanup |
| `cmd/server/viewer_integration_test.go` | Full stack smoke | **Yes** — single integration test on temp DB |

### 6.2 Mock pattern

Function-field mocks, per `services/testing.md`:

```go
type MockViewerServicer struct {
    SummaryFunc        func(ctx context.Context) (*SummaryResponse, error)
    HierarchyFunc      func(ctx context.Context) (*HierarchyResponse, error)
    HistoryFunc        func(ctx context.Context, key string) (*HistoryResponse, error)
    FileFunc           func(ctx context.Context, key string) (*FileResponse, error)
    FeatureTasksFunc   func(ctx context.Context, key string, opts FeatureTaskOptions) (*FeatureTasksResponse, error)
    RecentActivityFunc func(ctx context.Context, opts RecentActivityOptions) (*RecentActivityResponse, error)
    WorkflowMetaFunc   func(ctx context.Context) (*WorkflowMetaResponse, error)
}

func (m *MockViewerServicer) Summary(ctx context.Context) (*SummaryResponse, error) {
    if m.SummaryFunc != nil { return m.SummaryFunc(ctx) }
    return nil, errors.New("SummaryFunc not set")
}
// ... delegate pattern for the other 6 methods
```

### 6.3 Handler test coverage

Minimum table entries per endpoint:

| Endpoint | Cases |
|---|---|
| `Summary` | happy path, empty project, service error → 500 |
| `Hierarchy` | happy path, empty project |
| `History/{key}` | happy path (each entity type: epic, feature, task, bug, change-card), invalid key → 400, not found → 404 |
| `File/{key}` | happy path, missing file → 200 `exists:false`, security error → 403, too large → 413, invalid key → 400, not found → 404 |
| `Features/{key}/tasks` | happy path, limit clamp, invalid query → 400, invalid key → 400, not found → 404 |
| `RecentActivity` | happy path, `entity_type` filter, `since` filter, invalid `entity_type` → 400, malformed `since` → 400, limit clamp |
| `WorkflowMeta` | happy path on short workflow, happy path on long workflow |
| CORS middleware | echoes `localhost`, echoes `127.0.0.1`, rejects `evil.example.com`, handles OPTIONS |

### 6.4 Service test coverage (mock repositories)

For each service method, verify:

- Happy path returns expected shape with correct color/phase enrichment
- Empty repository returns empty slices / zero counts, not errors
- Repository error propagates wrapped with `fmt.Errorf("...: %w", err)`
- Invalid options (e.g., negative `limit`) are clamped rather than erroring
- `File` method: path-traversal DB row → `*SecurityError`; missing file → `exists:false`; oversized file → `*FileTooLargeError`

Each test uses the function-field mock pattern with inline fixtures.

### 6.5 Repository test coverage

`ListRecentAcrossEntities`:

- 5 entities (one per type), 10 history rows each → top N sorted DESC correctly
- Filter by `EntityType` → only matching rows
- Filter by `Since` → only rows after that time
- Deleted entity (history row orphaned) → omitted from result
- Mixed entity types in result → each row carries correct `entity_type` label

### 6.6 Integration smoke test

One test, `TestViewerIntegration_HappyPath`:

1. `test.GetTestDB()` — real SQLite
2. Seed: 1 epic, 2 features, 5 tasks per feature, 3 history rows per task, 1 bug, 1 change-card
3. `WireServices(db, tempProjectRoot)` — full container
4. `httptest.NewServer(mux)` — real HTTP stack
5. For each of the 7 endpoints, `GET` the URL, assert HTTP 200, assert top-level JSON keys exist
6. Teardown: close server, close DB, delete temp files

This test guards against wiring regressions without duplicating unit coverage.

### 6.7 Coverage targets

| Package | Target |
|---|---|
| `internal/api/viewer` | ≥ 85% line coverage |
| `internal/services/viewer_service.go` | ≥ 85% line coverage |
| `internal/repository/entityhistory` (new method) | 100% branch coverage |

---

## 7. Dependencies

### 7.1 Feature prerequisites

- **E27-F01 — `internal/dbinit` extraction and `cmd/server` migration.**
  - **Why blocking:** Until `cmd/server/main.go` uses `dbinit.Init`, the server hardcodes `db.InitDB("shark-tasks.db")` and ignores `.sharkconfig.json`. Integration testing this feature against a Turso-configured project would fail; local SQLite paths would ignore project-root auto-detection.
  - **Status to wait for:** F1 merged into `main` and `cmd/server` passes its existing tests.

### 7.2 Existing infrastructure this feature consumes

| Component | Usage |
|---|---|
| `internal/repository.EpicRepository` / `FeatureRepository` / `TaskRepository` | Composed via `ViewerService` constructor; existing methods sufficient for Summary/Hierarchy/FeatureTasks/File/History. |
| `internal/repository.BugRepository` (`repository/bug`) | Counts and severity rollup for Summary. |
| `internal/repository.ChangeCardRepository` (`repository/changecard`) | Counts for Summary. |
| `internal/repository.EntityHistoryRepository` (`repository/entityhistory`) | `ListByEntity` for `/history/{key}`; `ListRecentAcrossEntities` (NEW, §4.4) for `/recent-activity`. |
| `internal/workflow.Service` | Status/color/phase/transition metadata for Summary, Hierarchy, WorkflowMeta. |
| `internal/status.CalculationService` | Blocked-count + progress rollups for Hierarchy. Same service already used by CLI `feature get`. |
| `internal/api.ErrorResponse`, `respondJSON`, `respondError`, `handleServiceError`, `pathParam` | Reused as-is; viewer package imports `internal/api`. (If a future refactor prefers, these move to `internal/api/httputil` per architecture §4.2 — not required by this feature.) |
| `internal/models/validation.go` | `ValidateEpicKey`, `ValidateFeatureKey`, `ValidateTaskKey`, `ValidateBugKey`, `ValidateChangeCardKey` — used by handler to reject malformed `{key}` path params before DB lookup. |
| `otelhttp` wrapping at `cmd/server/main.go:92` | Viewer routes inherit tracing + request metrics for free. |

### 7.3 No new third-party dependencies

This feature adds **zero** new Go modules. Everything is composition over existing packages.

### 7.4 Config coupling

- `.sharkconfig.json` `status_metadata` map is consumed **indirectly** via `workflow.Service`. If the config is missing a status for a level, the fallback `{color: "gray", phase: "unknown"}` applies (REQ-F-001 AC). The viewer never reads `.sharkconfig.json` directly.
- `workflow_config` field selects between `.sharkworkflow-short.json` and `.sharkworkflow.json`; both are supported by this feature without code changes (REQ-F-007 AC, Scenario 6).

---

## 8. Open Questions / Notes for Feature Review

1. **Viewer error sentinels:** should `*SecurityError` and `*FileTooLargeError` live in `internal/api/viewer/service.go` or be elevated to `internal/services/viewer_errors.go`? Current spec says the former (handler package) because they are consumed by `handleViewerError`. Flag for review.
2. **`common.go` move:** Architecture §4.2 suggests moving `respondJSON`/`handleServiceError` to `internal/api/httputil` to avoid sub-package importing parent. The spec assumes direct import of `internal/api` and defers that refactor to a follow-up chore. Flag if reviewer prefers to land the refactor in this feature.
3. **Polling cadence for recent-activity:** the SPA's 10s polling interval is documented in the architecture but not enforced by the endpoint itself. Spec intentionally leaves endpoint stateless; any caching/rate-limiting is a follow-up if contention becomes measurable.
4. **Test DB reuse:** the integration test in Phase F creates a temp DB; verify `test.GetTestDB()` is reentrant enough to coexist with existing repository tests running in parallel, or add a test-specific setup helper.

---

*Specification draft — ready for feature review.*
*Author: architect agent.*
*Date: 2026-04-11.*
