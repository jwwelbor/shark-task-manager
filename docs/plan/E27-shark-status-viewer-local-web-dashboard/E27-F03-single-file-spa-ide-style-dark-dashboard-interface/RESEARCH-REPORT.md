# E27-F03 Research Report: Single-File SPA — IDE-Style Dark Dashboard Interface

**Date**: 2026-04-11  
**Status**: Research Complete  
**Feature**: E27-F03 — Single-File SPA - IDE-Style Dark Dashboard Interface

---

## Executive Summary

The SPA is a ~2300-line vanilla HTML/CSS/JS file embedded in the server binary via `go:embed`. It consumes 7 read-only endpoints from E27-F02 (`/api/v1/viewer/`), renders a dark IDE-style 3-panel layout with sidebar tree navigation, dashboard overview, entity detail with markdown rendering, and transition history. No pre-existing HTML implementation exists in the codebase. The `go:embed` pattern is well-established via `embedded.go` at the project root and the `internal/init/templates.go` package. The recommended implementation path is 7 tasks decomposed by application state and major subsystem.

---

## 1. Existing Implementations

### HTML/JS in the Codebase

No existing HTML, SPA, or viewer interface exists:

- `/home/jwwel/projects/shark-task-manager/internal/templates/` — markdown `.tmpl` files only (task execution templates, not HTML UI)
- `/home/jwwel/projects/shark-task-manager/cmd/server/main.go` — the `GET /` route currently returns plain text: `"Shark Task Manager API - Database Ready"`
- No `internal/viewer/`, no `*.html`, no frontend JS files exist anywhere in the project

The SPA (`viewer.html`) will be a net-new file at `internal/viewer/assets/viewer.html`.

### go:embed Pattern (Established)

The project has a working `go:embed` pattern at the root level:

**File**: `/home/jwwel/projects/shark-task-manager/embedded.go`
```go
//go:embed all:shark-templates
var EmbeddedSharkTemplates embed.FS
```

The `internal/init/templates.go` package accesses this via the import alias `stm "github.com/jwwelbor/shark-task-manager"`.

For the viewer, a new embed declaration will live in `internal/viewer/assets.go`:
```go
//go:embed viewer.html
var ViewerHTML embed.FS
```

The server `GET /` handler will serve `viewer.html` content directly from the embedded FS.

### Existing HTTP API Handler Pattern

The handler pattern in `internal/api/` is established:

- `internal/api/common.go` — `respondJSON`, `respondError`, `handleServiceError`, `pathParam` helpers
- Handler struct injects a service interface (e.g., `EpicServicer`)
- Routes registered via `RegisterRoutes(mux *http.ServeMux)`
- Uses Go 1.22+ `net/http` path matching (`{key}` syntax) — no chi router

The `ViewerHandler` in `internal/api/viewer/handler.go` will follow this same pattern.

---

## 2. API Contract (from E27-F02)

The SPA consumes these 7 endpoints (all `GET`, all under `/api/v1/viewer/`):

### Endpoint Summary Table

| Endpoint | SPA Usage | Key Response Fields |
|---|---|---|
| `GET /api/v1/viewer/summary` | Dashboard — Status Breakdown cards | `epics.total`, `epics.status_counts[].{status,count,color}`, same for features/tasks/bugs |
| `GET /api/v1/viewer/hierarchy` | Sidebar tree navigator | Array of epics, each with `key`, `title`, `status`, `slug`, `features[]` → each with `key`, `title`, `status`, `task_count`, optional `docs[]`, `tasks[]` |
| `GET /api/v1/viewer/history/{key}` | Entity View — Transitions tab, History drawer | Array of `{date, from_status, to_status, note}` entries, newest-first |
| `GET /api/v1/viewer/file/{key}` | Entity View — markdown content pane | `{exists: bool, content: string}` — raw markdown text |
| `GET /api/v1/viewer/features/{key}/tasks` | Feature drill-down (feature progress click) | Array of tasks with status, supports `?status=`, `?agent=`, `?blocked=true`, `?limit=`, `?offset=` |
| `GET /api/v1/viewer/recent-activity` | Dashboard — Active Transitions section | Array of `{key, from_status, to_status, note, updated_at}`, supports `?limit=25` |
| `GET /api/v1/viewer/workflow-meta` | Status color/phase lookup table on load | `{statuses: [{name, color, phase, ...}], transitions: [...]}` — full workflow definition |

### Response Shape Details (from Epic PRD)

**`/summary`**:
```json
{
  "generated_at": "2026-04-11T14:32:00Z",
  "epics":    {"total": 7, "status_counts": [{"status": "active", "count": 5, "color": "yellow", "phase": "development"}]},
  "features": {"total": 34, "status_counts": [...]},
  "tasks":    {"total": 212, "blocked_count": 3, "status_counts": [...]},
  "bugs":     {"total": 8, "status_counts": [...], "severity_counts": {"critical": 1, "high": 3}}
}
```

**`/hierarchy`** (inferred from Epic PRD sidebar spec):
```json
[{
  "key": "E01", "title": "Notification System", "status": "active", "slug": "...",
  "features": [{
    "key": "E01-F01", "title": "Email Notifications", "status": "in_progress",
    "task_count": 5,
    "docs": [{"title": "design", "path": "docs/plan/..."}],
    "tasks": [{"key": "E01-F01-001", "title": "Set up email...", "status": "todo"}]
  }]
}]
```

**`/file/{key}`** — Security note: path resolved from DB record only, validated within project root. Returns `{"exists": false}` if file missing.

**`/recent-activity`** — Default `?limit=25`. Optional `?entity_type=`, `?since=` (RFC3339).

**`/workflow-meta`** — SPA loads on initialization to build the `STATUS_COLORS` map client-side as a fallback/supplement to the hardcoded table (see Section 6).

### API Risk Assessment

**Risk**: E27-F02 is currently `in_specification` — API contract may be refined before implementation. The SPA must be tolerant of:
- Missing optional fields (graceful degradation)
- Empty arrays from endpoints (show empty state, not error)
- `exists: false` from `/file/{key}` (show "no content" message)

**Mitigation**: Define the response types in a local `api.js` module within `viewer.html` as documented interfaces, making contract drift visible.

---

## 3. Visual Requirements Analysis

### Source Documents

1. **UI Spec**: `/home/jwwel/projects/shark-task-manager/docs/status-viewer-ui.md`
2. **Reference Screenshots**: `/home/jwwel/Pictures/status/tracker/` (3 images)

### Screenshot Analysis

**Screenshot 1** (`20260411_112944.jpg`) — Entity View:
- Three-panel layout confirmed: narrow sidebar (~320px) on left, properties panel below sidebar, main content area on right
- Header bar at top with "Status Tracker" label, entity count pills ("2 Epics", "5 Features", "3 Tasks", "1 Bugs"), Refresh + Pick Folder buttons top-right
- Sidebar: tree navigator with colored status dots, expand/collapse arrows (▶/▼)
- Epics bold and expanded by default; features indented; tasks further indented
- Selected node: visible highlight (darker background, left accent border)
- Content area: renders markdown with heading "SMS Notifications", sections Origin/Goal/Requirements
- Properties panel (below sidebar or inline): shows key-value pairs for path, key, status, type, etc.
- Dark background throughout — deep slate/near-black background, white/gray text

**Screenshot 2** (`20260411_113020.jpg`) — Transition History View:
- Transition History drawer/panel visible in content area
- Table format: date column | from-status badge | arrow | to-status badge | note text
- Status badges: colored pill shapes — yellow for `in_progress`/`in_review`, green for `completed`, gray for initial
- "Spec" button visible (toggle between content and history)
- Properties panel shows entity metadata (path, key, status, type, parent, agent, created, updated)
- Sidebar shows same tree structure with status dots
- "Sync" and "Nav" buttons visible in top-right of content area

**Screenshot 3** (`20260411_113043.jpg`) — Dashboard View:
- Header: "Status Tracker" | entity count tabs: Epics 2, Features 5, Tasks 3, Bugs 1, Ideas 1 | Refresh + Pick Folder
- Content area shows "Status Overview" section with cards grid:
  - EPICS card: "2" large number, IN_PROGRESS=1 (yellow), BLOCKED=1 (orange)
  - FEATURES card: "5" large number, IN_PROGRESS=1, TODO=1, CANCELLED=1 (multiple colored badges)
  - TASKS card: "3" large number, IN_PROGRESS=1, CANCELLED=2
  - BUGS card: "1" large number, ANALYSED=1 (blue)
  - IDEAS card: "1" large number, INPROGRESS=1
- "Feature Progress" section below: list of features with horizontal progress bars
  - Format: feature key (blue, clickable) | feature title | progress bar (blue fill) | percentage %
  - Features: E01-F01, E01-F02, E01-F03, E01-F01, E02-F02 visible
- "Recent Activity" section: shows last transitions
  - Format: date | key (blue) | from-badge → to-badge | note text
- Sidebar remains visible with same tree structure

### Layout Dimensions (from UI Spec + Screenshots)

- **Header**: fixed height (appears ~40-50px from screenshots)
- **Sidebar**: 320px fixed width
- **Content**: remaining width (fills)
- **Epic indent**: 16px left padding, bold text
- **Feature indent**: 40px left padding
- **Task indent**: 56px left padding
- **Flat section items**: 18px left padding

### Typography (observed from screenshots)

- Monospace font for keys (e.g., `E01-F01`, `T-E01-F01-001`) — accent blue
- Sans-serif for titles — secondary muted text color
- Entity type tabs: caps or small text, dim when inactive
- Status badges: small pill, uppercase or lowercase label, colored background

---

## 4. Technical Approach

### Architecture Decision

**Single HTML file with vanilla JS + Marked.js CDN** — no build step, no npm, no module bundling.

- File location: `internal/viewer/assets/viewer.html`
- Embedded via `go:embed` in `internal/viewer/assets.go`
- Served at `GET /` from `cmd/server/main.go`
- Marked.js loaded from CDN: `https://cdn.jsdelivr.net/npm/marked/marked.min.js`
- No other external dependencies

### go:embed Integration

New file `internal/viewer/assets.go`:
```go
package viewer

import "embed"

//go:embed viewer.html
var ViewerHTML []byte
```

`cmd/server/main.go` change:
```go
import "github.com/jwwelbor/shark-task-manager/internal/viewer"

mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write(viewer.ViewerHTML)
})
```

### File Structure

```
internal/viewer/
├── assets.go        — go:embed declaration
└── assets/
    └── viewer.html  — complete SPA (~2300 LOC)
```

### CORS Consideration

The server must add CORS headers to permit the SPA to call API endpoints when loaded from a browser:
```
Access-Control-Allow-Origin: *  (or specific localhost:port)
```
This is a `cmd/server/main.go` concern, handled via middleware or per-handler headers.

### JavaScript Architecture

All JS inside a single `<script>` tag at bottom of `<body>`:

```
State: {
  appState: 'pick_folder' | 'dashboard' | 'entity_view' | 'doc_view'
  selectedKey: string | null
  treeData: HierarchyNode[]
  summaryData: SummaryResponse | null
  recentActivity: ActivityEntry[]
  workflowMeta: WorkflowMeta | null
  expandedEpics: Set<string>
  expandedFeatures: Set<string>
}

Init flow:
  1. Load workflow-meta (build status color table)
  2. Try load hierarchy (build sidebar tree)
  3. If success → transition to 'dashboard', load summary + recent-activity
  4. If fail → stay on 'pick_folder'

Navigation flow:
  click tree node → load file/{key} + history/{key} → transition to 'entity_view'
  click dashboard key → find node in tree, expand ancestors, scroll to view, select
  click dashboard feature progress bar → navigate to feature entity view
  Escape → return to dashboard or close drawer
```

---

## 5. Component Breakdown

| Component | Description | Complexity | ~LOC |
|---|---|---|---|
| **App Shell** | HTML structure, CSS variables, base layout (header/sidebar/content) | Medium | 250 |
| **CSS Theme** | Dark palette, status badge styles, tree indentation, animations | High | 350 |
| **State Machine** | App state variables, state transition functions, render dispatch | Medium | 150 |
| **API Client** | fetch wrappers for all 7 endpoints, error handling, loading states | Low | 120 |
| **Sidebar Tree** | Hierarchy rendering, expand/collapse, selection, scroll-to-node | High | 280 |
| **Dashboard** | Status Breakdown cards, Feature Progress bars, Active Transitions, Stale Entities | High | 350 |
| **Entity View** | Properties panel, markdown render via Marked.js, Info/Transitions toggle | Medium | 250 |
| **History Drawer** | Transition table with colored badges, reverse-chronological | Low | 120 |
| **Navigation Logic** | Cross-view key-click → find-and-select tree node | Medium | 80 |
| **Pick Folder Screen** | Initial state UI (folder icon, button, instructions) | Low | 50 |
| **Status Color Mapping** | `STATUS_COLORS` const + `getStatusColor(status)` helper | Low | 80 |
| **Utility Helpers** | Date formatting, key parsing, debounce, DOM helpers | Low | 70 |

**Total estimate: ~2150 LOC** (within the ~2300 LOC assessment)

---

## 6. Status Color Mapping

Full status→hex table from `docs/status-viewer-ui.md`:

```javascript
const STATUS_COLORS = {
  // Core entity statuses
  draft:        '#8c8c8c',   // gray
  todo:         '#4d4dbb',   // blue
  in_progress:  '#e7b30e',   // yellow
  open:         '#e7b30e',   // yellow (same as in_progress)
  done:         '#4d4269',   // purple-green
  blocked:      '#8c5700',   // dark orange
  cancelled:    '#4b4b36',   // dark gray
  // Bug statuses
  triaged:      '#e7b500',   // yellow
  verified:     '#4d4269',   // same as done
  closed:       '#4b4b26',   // dark gray
  wont_fix:     '#4b4b26',   // dark gray
  identified:   '#8c8c8c',   // gray (same as draft)
  confirmed:    '#4d4dbb',   // blue (same as todo)
  // Idea / Change Card statuses
  proposed:     '#4d4dbb',   // blue
  evaluating:   '#e7b500',   // yellow
  accepted:     '#4d4269',   // green
  promoted:     '#8c4dbb',   // purple
  rejected:     '#8c4d6c',   // dark pink
  reopened:     '#e7b500',   // yellow
  approved:     '#4d4269',   // green
};

// Fallback for any status not in the table
function getStatusColor(status) {
  return STATUS_COLORS[status] ?? '#666666';
}
```

**Terminal statuses** (excluded from stale entity detection):
`done`, `cancelled`, `closed`, `wont_fix`, `verified`, `rejected`, `promoted`, `approved`

**Stale threshold**: entities with `updated` > 7 days ago AND NOT in a terminal status.

---

## 7. State Machine

### 4 Application States

```
                         load success
[Pick Folder] ─────────────────────────────────► [Dashboard]
                                                    │    ▲
                                        click node  │    │ Escape / Dashboard nav
                                        or entity   │    │
                                                    ▼    │
                                              [Entity View]
                                                    │
                                        click doc   │
                                        in tree     │
                                                    ▼
                                              [Doc View]
```

### State Transition Details

| Transition | Trigger | Action |
|---|---|---|
| Pick Folder → Dashboard | Hierarchy loads successfully | Load summary, recent-activity; render dashboard |
| Dashboard → Entity View | Click tree node OR click entity key in dashboard | Load file/{key} + history/{key}; render entity view |
| Dashboard → Doc View | Click doc node in sidebar tree | Load file content only; render markdown |
| Entity View → Dashboard | Escape key OR click "Dashboard" in header | Clear selected entity; re-render dashboard |
| Doc View → Dashboard | Escape key OR click "Dashboard" in header | Same |
| Entity View (Info) → Entity View (History) | Click "Transitions" toggle button | Show history drawer; hide markdown |
| Entity View (History) → Entity View (Info) | Click "Info" toggle button OR "Open" button | Show markdown; hide history drawer |

### State Variables

```javascript
let appState      = 'pick_folder';   // current application state
let selectedKey   = null;            // currently selected entity key
let treeData      = [];              // cached hierarchy response
let summaryData   = null;            // cached summary response
let recentActivity = [];             // cached recent-activity response
let workflowMeta  = null;            // loaded workflow definition (optional)
let expandedEpics    = new Set();    // set of expanded epic keys
let expandedFeatures = new Set();    // set of expanded feature keys
```

---

## 8. Task Decomposition Recommendation

Recommended 7 tasks for implementing E27-F03:

### Task 1: App Shell, CSS Theme, and Pick Folder Screen
- HTML skeleton: header, sidebar, content area (`<div id="app">`)
- CSS custom properties for dark theme (background, text, accent, sidebar width)
- Status badge CSS classes
- Tree node CSS (indentation levels, hover, selected state)
- Pick Folder Screen HTML/CSS
- State machine skeleton (state variables, `render()` dispatcher stub)
- **Complexity**: M

### Task 2: API Client and Workflow Meta Loading
- `api.js` module (within `<script>`) — 7 fetch functions with error handling
- `STATUS_COLORS` const + `getStatusColor()` helper
- On-load workflow-meta fetch to supplement status color table
- Loading and error state UI patterns
- **Complexity**: S

### Task 3: Sidebar Tree — Hierarchy Rendering and Navigation
- `renderSidebar()` — builds the full tree from hierarchy data
- Expand/collapse arrows and toggle logic
- Node selection (click → `selectedKey`, highlight)
- Flat sections: Bugs, Ideas, Change Cards
- **Complexity**: L (most complex single component)

### Task 4: Dashboard — Status Breakdown Cards and Feature Progress
- `renderDashboard()` entry point
- Status Breakdown cards grid: per entity type, status badge counts
- Feature Progress bars: key, title, progress bar, percentage
- Click handler: feature key → navigate to entity view
- **Complexity**: M

### Task 5: Dashboard — Active Transitions and Stale Entities
- Active Transitions section: last 25 from `/recent-activity`
- Render: date | from-badge | to-badge | note | Open button
- Stale Entities section: filter non-terminal, sort by staleness
- Stale render: key | status badge | title | "X days ago"
- Click handler: key → navigate to entity (find tree node, expand, select)
- **Complexity**: M

### Task 6: Entity View — Properties Panel and Markdown Content
- `renderEntityView()` — 3-zone layout
- Properties panel: key-value grid from entity data, status as colored badge, file path with copy button
- Info/Transitions toggle buttons
- Markdown rendering: load `/file/{key}`, render with Marked.js
- Empty state: "No content available" if `exists: false`
- **Complexity**: M

### Task 7: History Drawer and Cross-View Navigation
- History drawer: reverse-chronological table, colored from/to badges, note text
- Toggle between Info and History views
- `navigateToEntity(key)`: find node in tree, expand ancestors, scroll into view, select, transition to entity view
- Keyboard: Escape closes view / returns to dashboard
- Integration testing: end-to-end flow from dashboard → entity → history → back
- **Complexity**: M

### Task Sequencing

```
Task 1 (Shell + CSS + Pick Folder)
    └── Task 2 (API Client)
          ├── Task 3 (Sidebar Tree)
          │     └── Task 4 (Dashboard: Status Breakdown + Feature Progress)
          │           └── Task 5 (Dashboard: Active Transitions + Stale)
          └── Task 6 (Entity View)
                └── Task 7 (History Drawer + Navigation)
```

---

## 9. Dependencies and Constraints

### Blocking Dependencies

- **E27-F02** (Viewer API Endpoints) — MUST be complete before Task 2+ can be tested against real data
- **E27-F01** (DB Init Extraction) — Server must support Turso before E27-F02 can be wired; transitively required

### Non-Blocking

- **E27-F05** (Server Wiring) — Handles registering `GET /` route; SPA can be developed with a simple static file server

### Technical Constraints

- **No build step**: all JS must be in a single `<script>` tag or inline `<style>` — no TypeScript, no bundler
- **No npm dependencies**: only Marked.js from CDN for markdown rendering
- **go:embed requires file in same package or subdirectory**: `viewer.html` must be under `internal/viewer/assets/`
- **CORS**: server must allow `localhost:*` origin for API requests from SPA
- **Browser File System API**: Pick Folder uses `showDirectoryPicker()` — Chrome/Edge only (Safari limited). For this implementation, the folder is the server's project root — no actual File System API needed; the "Pick Folder" state is a UX placeholder that resolves when the server loads hierarchy successfully.

### Risk Mitigation

| Risk | Mitigation |
|---|---|
| Single file becomes unmaintainable | Section comments, clear function naming, state machine pattern |
| Marked.js CDN unavailable | Bundle fallback: include minified Marked.js inline as a comment-escaped string |
| API contract drift (E27-F02 changes) | Define response type interfaces in comments at top of script; graceful degradation for missing fields |
| Browser compatibility | Target Chrome/Safari latest; avoid cutting-edge APIs except `fetch` |

---

## 10. Architecture Standards Reference

### go:embed Pattern to Follow

```go
// internal/viewer/assets.go
package viewer

import _ "embed"

//go:embed assets/viewer.html
var ViewerHTML []byte
```

Note: Use `[]byte` (not `embed.FS`) for single-file embed — simpler to serve.

### Handler Registration Pattern (from `internal/api/`)

- Use `net/http` ServeMux with method prefix: `"GET /"` syntax (Go 1.22+)
- `respondJSON`, `respondError` helpers from `internal/api/common.go` available but viewer handler may need its own for HTML serving
- Handler file: `internal/api/viewer/handler.go`
- Test file: `internal/api/viewer/handler_test.go` (mock ViewerServicer)

### File Path Security (for `/file/{key}` calls)

- Path resolved exclusively from DB `file_path` column — never from URL query params
- Server-side validation: resolved path must be within project root (`filepath.Rel`)
- SPA never constructs file paths itself — always calls `/api/v1/viewer/file/{key}`

---

## References

1. **UI Specification**: `/home/jwwel/projects/shark-task-manager/docs/status-viewer-ui.md`
2. **Epic PRD (API Contract)**: `/home/jwwel/projects/shark-task-manager/docs/plan/E27-shark-status-viewer-local-web-dashboard/epic.md`
3. **E27-F02 Feature Doc**: `/home/jwwel/projects/shark-task-manager/docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F02-viewer-api-endpoints-read-only-dashboard-data-laye/feature.md`
4. **Reference Screenshots**: `/home/jwwel/Pictures/status/tracker/` (3 images)
5. **go:embed Pattern**: `/home/jwwel/projects/shark-task-manager/embedded.go`
6. **Server Entry Point**: `/home/jwwel/projects/shark-task-manager/cmd/server/main.go`
7. **Service Wiring**: `/home/jwwel/projects/shark-task-manager/cmd/server/services.go`
8. **API Handler Pattern**: `/home/jwwel/projects/shark-task-manager/internal/api/common.go`
9. **Complexity Assessment**: `/home/jwwel/projects/shark-task-manager/docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F03-single-file-spa-ide-style-dark-dashboard-interface/ASSESSMENT-SUMMARY.md`
