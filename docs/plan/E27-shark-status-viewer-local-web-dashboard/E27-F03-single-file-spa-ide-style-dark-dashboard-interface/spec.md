---
feature_key: E27-F03-single-file-spa-ide-style-dark-dashboard-interface
epic_key: E27
title: "Specification — Single-File SPA: IDE-Style Dark Dashboard Interface"
status: in_specification
---

# Specification: E27-F03 — Single-File SPA (IDE-Style Dark Dashboard)

**Last Updated**: 2026-04-11
**Related**:
- [Feature Doc](feature.md)
- [Research Report](RESEARCH-REPORT.md)
- [UI Spec](../../../../docs/status-viewer-ui.md)
- [Epic PRD](../../epic.md)
- [Epic Architecture](../../architecture.md)

---

## 1. Overview

This feature delivers `internal/viewer/assets/viewer.html` — a self-contained, vanilla JS + HTML + CSS Single-Page Application that renders the dark IDE-style status viewer dashboard described in `docs/status-viewer-ui.md`. The SPA is:

- **Single-file**: one HTML document with embedded `<style>` and `<script>` blocks, ~2300 LOC total
- **Zero build step**: no bundler, no TypeScript, no npm; vanilla JS only
- **One external dependency**: Marked.js loaded from CDN (`https://cdn.jsdelivr.net/npm/marked/marked.min.js`) for markdown rendering
- **Embedded in the Go server binary** via `go:embed` in `internal/viewer/assets.go`
- **Served at `GET /`** by the existing `cmd/server/main.go` HTTP server (E27-F05 wires this)
- **Read-only**: consumes the 7 JSON endpoints defined in E27-F02 under `/api/v1/viewer/`, never mutates state

The SPA implements a 4-state IDE-style interface: Pick Folder screen → Dashboard → Entity View → Doc View, with a 320px collapsible sidebar tree navigator, a dashboard with 4 content sections (Status Breakdown, Feature Progress, Active Transitions, Stale Entities), an entity view with properties panel + markdown content + transition history drawer, and cross-view key-click navigation.

### Goals

1. Provide a local browser-based read-only dashboard for inspecting shark project state
2. Match the exact visual design in `docs/status-viewer-ui.md` and the reference screenshots
3. Ship as a single embedded file — no frontend toolchain required

### Non-Goals

- No editing / writing (read-only)
- No authentication / multi-user support (localhost only)
- No build system, no framework (React/Vue/Svelte)
- No service worker / offline support
- No responsive / mobile layout (desktop only)

---

## 2. Visual Requirements

All visual requirements derived from `docs/status-viewer-ui.md` and the reference screenshots at `/home/jwwel/Pictures/status/tracker/`.

### 2.1 Theme

- **Dark slate palette**: deep near-black base, muted gray titles, white primary text
- **Accent color**: blue (`#4d77ff` range) for entity keys, buttons, selected borders
- **Typography**:
  - Monospace for keys (e.g., `E07-F01-001`) in accent blue
  - Sans-serif for titles in secondary (muted) text color
  - Uppercase small-caps for section headers and entity type labels
- **Status badges and dots**: always rendered using the exact hex palette below

### 2.2 Full Status → Color Mapping Table

The SPA embeds this table as a `const STATUS_COLORS` JavaScript object. `getStatusColor(status)` returns the hex or `#666666` as a fallback for unknown statuses.

| Status | Hex | Notes |
|---|---|---|
| `draft` | `#8c8c8c` | gray |
| `todo` | `#4d4dbb` | blue |
| `in_progress` | `#e7b30e` | yellow |
| `open` | `#e7b30e` | yellow (same as in_progress) |
| `done` | `#4d4269` | green |
| `blocked` | `#8c5700` | dark orange |
| `cancelled` | `#4b4b36` | dark gray |
| `triaged` | `#e7b500` | yellow |
| `verified` | `#4d4269` | green (same as done) |
| `closed` | `#4b4b26` | dark gray |
| `wont_fix` | `#4b4b26` | dark gray |
| `identified` | `#8c8c8c` | gray (same as draft) |
| `confirmed` | `#4d4dbb` | blue (same as todo) |
| `proposed` | `#4d4dbb` | blue |
| `evaluating` | `#e7b500` | yellow |
| `accepted` | `#4d4269` | green |
| `promoted` | `#8c4dbb` | purple |
| `rejected` | `#8c4d6c` | dark pink |
| `reopened` | `#e7b500` | yellow |
| `approved` | `#4d4269` | green |
| _fallback_ | `#666666` | unknown statuses |

**Terminal statuses** (excluded from Stale Entity detection):
`done`, `cancelled`, `closed`, `wont_fix`, `verified`, `rejected`, `promoted`, `approved`.

**Stale threshold**: non-terminal entities with `updated_at` more than 7 days ago.

### 2.3 Application States (4)

#### State 1: Pick Folder Screen
- Shown on initial load and when hierarchy load fails
- Centered vertical layout:
  - Large faded folder icon (SVG, top)
  - `Status Tracker` heading (large, primary text)
  - Instruction text ("Select a shark project folder to browse.")
  - "Pick Folder" button (accent blue, prominent)
  - Small note text: "Read-only access. Does not modify project state."
- No sidebar, no header

#### State 2: Dashboard View (default after load)
- 3-panel layout: Header (top), Sidebar (left 320px), Content area (remaining)
- Content area renders 4 dashboard sections (see §2.5)

#### State 3: Entity View (click tree node or dashboard key)
- 3-panel layout (same as Dashboard)
- Content area shows entity metadata + markdown content with Info/Transitions toggle

#### State 4: Doc View (click a design doc node in sidebar)
- 3-panel layout (same as Dashboard)
- Content area shows plain markdown rendering with no history toggle

### 2.4 Header Bar (~40–50px fixed height)

- Left:
  - "Dashboard" title (clickable → returns to Dashboard state)
  - Entity count pills: `2 Epics`, `5 Features`, `3 Tasks`, `1 Bugs`, `1 Ideas` (from `/summary`)
- Right:
  - "Refresh" button (reloads all cached data)
  - "Pick Folder" button (returns to Pick Folder state)

### 2.5 Dashboard Sections (Content Area)

#### 2.5.1 Status Breakdown
- Grid of cards (one per entity type that has items: Epics, Features, Tasks, Bugs, Ideas, Change Cards)
- Each card:
  - Uppercase type label (e.g., `EPICS`)
  - Large total count
  - Row of colored status badges with counts below the total
  - Badge: background = `getStatusColor(status)`, label = status name + count
- Data source: `GET /api/v1/viewer/summary` response `{epics, features, tasks, bugs}.status_counts[]`

#### 2.5.2 Feature Progress
- List of features with horizontal progress bars
- Per-row layout: `[key — accent blue, clickable] [title — secondary] [progress bar — blue fill] [percentage]`
- Data source: derived from `GET /api/v1/viewer/hierarchy` (per-feature task counts) or `GET /api/v1/viewer/features/{key}/tasks`
- Click feature key → navigate to entity view for that feature

#### 2.5.3 Active Transitions
- Title: "Active Transitions"
- Table of last 25 transitions (newest first)
- Per-row layout: `[date] [key — clickable] [from-badge] → [to-badge] [note text] [Open button]`
- "Open" button shown only if the target entity has linked content
- Data source: `GET /api/v1/viewer/recent-activity?limit=25`
- Click key or Open → navigate to entity view

#### 2.5.4 Stale Entities
- Title: "Stale Entities"
- Client-side filter: non-terminal entities with `updated_at > 7 days ago`
- Sorted by staleness descending (oldest `updated_at` first)
- Per-row layout: `[key — clickable] [status-badge] [title] [last update age — "12 days ago"]`
- Data source: derived from cached `treeData` from `/hierarchy` (each node carries `status` and `updated_at`)
- Click key → navigate to entity view

### 2.6 Sidebar Tree Navigator (320px fixed width)

#### 2.6.1 Sections

- **"Hierarchy"** section (sticky uppercase header): tree of epics → features → (docs + tasks)
- **Flat sections** (only rendered if items exist):
  - "Tags"
  - "Tech Debt"
  - "Ideas"
  - "Change Cards"

Each flat section has a sticky uppercase header and a flat list (no nesting) of nodes.

#### 2.6.2 Tree Indentation

| Node type | Left padding | Typography |
|---|---|---|
| Epic | 16px | bold |
| Feature | 40px | normal |
| Doc at epic level | 16px | italic, muted |
| Doc at feature level | 40px | italic, muted |
| Task | 56px | normal |
| Flat section node | 18px | normal |

#### 2.6.3 Node Format

Each entity node renders left-to-right:

```
[▶/▼ arrow (if expandable)] [status dot — colored circle] [key — mono accent blue] [title — muted, truncated with ellipsis]
```

- **Status dot**: 8–10px filled circle, background = `getStatusColor(node.status)`
- **Key**: monospace font, accent blue color
- **Title**: sans-serif, muted gray, `text-overflow: ellipsis` with `overflow: hidden`

#### 2.6.4 Expand / Collapse

- **Epics**: start **expanded** (`expandedEpics` seeded with all epic keys on initial hierarchy load)
- **Features**: start **collapsed**
- Clicking the arrow (`▶`/`▼`) toggles expansion without selecting the node
- Clicking the node body (not the arrow) selects the node
- Expansion state persists across data refreshes (stored in `expandedEpics`/`expandedFeatures` Sets)

#### 2.6.5 Selected State

- Darker background tint (black with slight transparency overlay)
- Accent-colored left border (3px)
- Title text color becomes primary (white) instead of muted

### 2.7 Entity View Layout

Three zones within the content area:

#### 2.7.1 Properties Panel
- Key-value grid rendered from entity metadata
- Fields shown: `path`, `key`, `status` (as colored badge), `type`, `parent`, `agent`, `created_at`, `updated_at`, `file_path`, `linked_content_path`
- `file_path` and `linked_content_path` each have a copy-to-clipboard button
- Status rendered as a colored badge using `getStatusColor(status)`

#### 2.7.2 Content Toolbar
- Entity name displayed prominently
- Toggle buttons:
  - **Info** button (default active): shows markdown content pane
  - **Transitions** button: shows transition history drawer
- "Sync" and "Nav" buttons (from reference screenshot) — optional affordances

#### 2.7.3 Markdown Content Pane (Info view)
- Data source: `GET /api/v1/viewer/file/{key}`
- If `exists: true`: render `content` via `marked.parse()`, wrap in styled `.markdown-body` container
- If `exists: false`: show "No content available for this entity." placeholder
- Markdown styling: headings, paragraphs, code blocks, tables, lists — all in dark theme

#### 2.7.4 History Drawer (Transitions view)
- Data source: `GET /api/v1/viewer/history/{key}`
- Reverse-chronological table
- Per-row layout: `[date] [from-badge] → [to-badge] [note text]`
- Badges use `getStatusColor()` for background
- "Open" button toggles back to Info view (only rendered if content exists)
- If history is empty: "No transition history." placeholder

### 2.8 Doc View Layout

- Content area shows plain markdown rendering of the selected doc file
- No Info/Transitions toggle
- Properties panel shows whatever metadata is associated with the doc (path, title, parent entity)

---

## 3. API Integration

The SPA consumes exactly 7 read-only endpoints from E27-F02 under `/api/v1/viewer/`. All fetches go through a single `api.js`-style module defined inline in the `<script>` block with consistent error handling.

| # | Endpoint | Used By | SPA Action |
|---|---|---|---|
| 1 | `GET /api/v1/viewer/workflow-meta` | Initial load | Supplements `STATUS_COLORS` with dynamic phase/transition data from the workflow config. Fallback if any status is missing from the hardcoded table. |
| 2 | `GET /api/v1/viewer/hierarchy` | Initial load → Sidebar tree | Caches full project tree (epics → features → tasks + docs). Drives sidebar tree rendering, Feature Progress calculation, and Stale Entities filter. |
| 3 | `GET /api/v1/viewer/summary` | Initial load + Refresh | Populates header count pills and Dashboard → Status Breakdown cards. |
| 4 | `GET /api/v1/viewer/recent-activity?limit=25` | Initial load + Refresh | Populates Dashboard → Active Transitions section. |
| 5 | `GET /api/v1/viewer/file/{key}` | Entity View + Doc View | Loads raw markdown content for the selected entity/doc. Response: `{exists, content}`. Rendered via Marked.js. |
| 6 | `GET /api/v1/viewer/history/{key}` | Entity View → History drawer | Loads transition history for the selected entity. Response: array of `{date, from_status, to_status, note}`. |
| 7 | `GET /api/v1/viewer/features/{key}/tasks` | Feature Progress drill-down | Loads tasks for a specific feature when the user clicks a feature key (used by Feature Progress section and as an alternative data source if hierarchy is truncated). |

### Error Handling Strategy

- **Hierarchy load fails** → remain in Pick Folder state, show error toast
- **Summary/recent-activity fails** → show inline error in affected dashboard section, other sections still render
- **File load returns `exists: false`** → show "No content available" placeholder, not an error
- **History load returns empty** → show "No transition history." placeholder
- **Network error on any call** → log to console, show retry affordance

### API Contract Robustness

The SPA tolerates optional fields, empty arrays, and missing data:
- Missing `docs[]` on a feature → no doc nodes rendered
- Empty `tasks[]` on a feature → feature still rendered, no task children
- Missing `severity_counts` on bugs → omit from card
- Missing `phase` on a status → ignore

---

## 4. Component Architecture

All components live in a single `viewer.html` file, organized as labeled sections via `// ============` comment banners in the `<script>` block. The 12 components from the research report:

| # | Component | Responsibility | ~LOC |
|---|---|---|---|
| 1 | **App Shell** | HTML skeleton (`<header>`, `<aside id="sidebar">`, `<main id="content">`), base grid layout | 250 |
| 2 | **CSS Theme** | Dark palette CSS custom properties, status badge classes, tree indentation rules, animations, typography | 350 |
| 3 | **State Machine** | Global state object (`appState`, `selectedKey`, caches), `render()` dispatcher, state transition functions | 150 |
| 4 | **API Client** | 7 `async` fetch wrappers (`apiGetHierarchy()`, `apiGetSummary()`, etc.), shared error handler, loading indicators | 120 |
| 5 | **Sidebar Tree** | `renderSidebar()`, epic/feature/task/doc node builders, expand/collapse handlers, scroll-to-node, selected state | 280 |
| 6 | **Dashboard** | `renderDashboard()` orchestrator, Status Breakdown cards, Feature Progress bars, click handlers | 250 |
| 7 | **Dashboard Activity/Stale** | Active Transitions table + Stale Entities filter and table | 100 |
| 8 | **Entity View** | `renderEntityView()`, properties panel, Info/Transitions toggle, markdown container, Marked.js integration | 250 |
| 9 | **History Drawer** | Transition table renderer, reverse-chronological sort, colored badge pairs | 120 |
| 10 | **Navigation Logic** | `navigateToEntity(key)`: find tree node → expand ancestors → scroll into view → select → transition state | 80 |
| 11 | **Pick Folder Screen** | Initial state UI (folder icon SVG, heading, button, instruction text) | 50 |
| 12 | **Status Color Mapping + Utilities** | `STATUS_COLORS` const, `getStatusColor()`, `formatDate()`, `daysSince()`, `isTerminalStatus()`, DOM helpers | 150 |

**Total: ~2150 LOC** (within the ~2300 LOC complexity assessment)

### File Structure

```
internal/viewer/
├── assets.go           # go:embed declaration
└── assets/
    └── viewer.html     # the complete SPA
```

### Embedding (`internal/viewer/assets.go`)

```go
package viewer

import _ "embed"

//go:embed assets/viewer.html
var ViewerHTML []byte
```

### Serving (delegated to E27-F05)

`cmd/server/main.go` adds a `GET /` handler that writes `viewer.ViewerHTML` with `Content-Type: text/html; charset=utf-8`. CORS headers permit browser access from localhost.

---

## 5. State Machine

### 5.1 States (4)

| State | Description |
|---|---|
| `pick_folder` | Initial state or after failed hierarchy load. Shows Pick Folder screen. |
| `dashboard` | Default state after successful load. Shows dashboard sections. |
| `entity_view` | An entity (epic/feature/task/bug/idea/change-card) is selected. Shows properties + markdown or history. |
| `doc_view` | A design doc node is selected. Shows plain markdown. |

### 5.2 Transitions (7)

| # | From | To | Trigger | Side Effects |
|---|---|---|---|---|
| 1 | `pick_folder` | `dashboard` | Hierarchy load succeeds | Fetch `/summary`, `/recent-activity`; seed `expandedEpics` with all epic keys |
| 2 | `dashboard` | `entity_view` | Click tree entity node OR click entity key anywhere on dashboard | Fetch `/file/{key}` + `/history/{key}`; set `selectedKey` |
| 3 | `dashboard` | `doc_view` | Click doc node in sidebar tree | Fetch `/file/{key}`; set `selectedKey` |
| 4 | `entity_view` | `dashboard` | Click "Dashboard" in header OR press `Escape` | Clear `selectedKey`; re-render dashboard |
| 5 | `doc_view` | `dashboard` | Click "Dashboard" in header OR press `Escape` | Clear `selectedKey`; re-render dashboard |
| 6 | `entity_view` (Info) | `entity_view` (History) | Click "Transitions" toggle | Swap content pane to history drawer (no state reload) |
| 7 | `entity_view` (History) | `entity_view` (Info) | Click "Info" toggle OR "Open" button in history | Swap content pane back to markdown |

### 5.3 State Variables (7)

```javascript
let appState       = 'pick_folder';  // current app state
let selectedKey    = null;           // currently selected entity/doc key
let treeData       = [];             // cached /hierarchy response
let summaryData    = null;           // cached /summary response
let recentActivity = [];             // cached /recent-activity response
let workflowMeta   = null;           // cached /workflow-meta response
let expandedEpics    = new Set();    // epic keys currently expanded in sidebar
let expandedFeatures = new Set();    // feature keys currently expanded in sidebar
```

(8 variables including the two expand sets, but logically 7 pieces of state.)

The `render()` function is a dispatcher that reads `appState` and calls the appropriate render function (`renderPickFolder()`, `renderDashboard()`, `renderEntityView()`, `renderDocView()`).

### 5.4 Keyboard

- `Escape` → if in `entity_view` or `doc_view`, returns to `dashboard`; if a drawer is open, closes the drawer first

---

## 6. Implementation Plan

7 ordered tasks sequenced for incremental review and parallelism where possible.

### Task 1: App Shell, CSS Theme, Status Color Mapping, Pick Folder Screen (~600 LOC)

**Scope**:
- HTML skeleton with header/sidebar/content regions
- CSS custom properties for dark theme (`--bg`, `--fg`, `--accent`, `--sidebar-width`, etc.)
- Typography (monospace for keys, sans-serif for titles)
- Status badge classes + dot styles
- Tree indentation CSS rules (16/40/56/18px)
- `STATUS_COLORS` const + `getStatusColor()` helper
- `isTerminalStatus()` helper
- `formatDate()`, `daysSince()` utilities
- Pick Folder screen HTML + CSS
- State machine skeleton: state variables, `render()` dispatcher stub, event bindings

**Deliverable**: Loadable HTML file that renders the Pick Folder screen with the correct dark theme. No API integration yet.

**Complexity**: M

---

### Task 2: Pick Folder Screen + API Client (~250 LOC)

**Scope**:
- 7 fetch wrapper functions (`apiGetHierarchy`, `apiGetSummary`, `apiGetRecentActivity`, `apiGetWorkflowMeta`, `apiGetFile`, `apiGetHistory`, `apiGetFeatureTasks`)
- Shared error handler with toast/banner UI
- Loading state indicators
- Pick Folder button wired to trigger full initial load sequence: workflow-meta → hierarchy → (summary, recent-activity)
- Transition `pick_folder` → `dashboard` on successful hierarchy load
- Handle load failures: stay on Pick Folder, show error

**Deliverable**: Pick Folder → API call → (empty) Dashboard state transition works end-to-end against a running E27-F02 server.

**Complexity**: S

---

### Task 3: Sidebar Tree Navigator (~350 LOC)

**Scope**:
- `renderSidebar()` orchestrator
- "Hierarchy" section: epics (expanded by default), features (collapsed), tasks, docs
- Flat sections: Tags, Tech Debt, Ideas, Change Cards (only if items exist)
- Node renderer: arrow + dot + key + title with correct indentation per type
- Expand/collapse arrow handlers (toggle `expandedEpics`/`expandedFeatures` sets)
- Click node handler → `selectedKey = key; appState = 'entity_view'; render()`
- Selected state styling
- Scroll-into-view helper

**Deliverable**: Sidebar fully renders from hierarchy data, expand/collapse works, clicking a node highlights it and sets `selectedKey` (but entity view is still a stub).

**Complexity**: L (most complex single component)

---

### Task 4: Dashboard — Status Breakdown + Feature Progress (~300 LOC)

**Scope**:
- `renderDashboard()` orchestrator
- Status Breakdown cards grid: per-entity-type cards with totals and colored status badges
- Data source: `summaryData.{epics,features,tasks,bugs}.status_counts`
- Feature Progress list: key, title, progress bar, percentage
- Data source: derived from `treeData` (feature task counts by status)
- Progress calculation: `(completed_task_count / total_task_count) * 100`
- Click feature key → `navigateToEntity(featureKey)`

**Deliverable**: Dashboard renders Status Breakdown and Feature Progress sections from cached data.

**Complexity**: M

---

### Task 5: Dashboard — Active Transitions + Stale Entities (~250 LOC)

**Scope**:
- Active Transitions section: render `recentActivity` array as a table
- Per-row: date | key (clickable) | from-badge → to-badge | note | Open button
- Stale Entities section: walk `treeData`, filter non-terminal + `daysSince(updated_at) > 7`, sort descending
- Per-row: key | status badge | title | "N days ago"
- Click handlers → `navigateToEntity(key)`

**Deliverable**: Complete Dashboard with all 4 sections rendered from real data.

**Complexity**: M

---

### Task 6: Entity View (Properties + Markdown) (~350 LOC)

**Scope**:
- `renderEntityView()` orchestrator
- Properties panel: key-value grid from entity metadata (looked up from `treeData`)
- Status field as colored badge
- File path copy-to-clipboard button
- Info/Transitions toggle toolbar
- Markdown content pane: load `/file/{key}`, render via `marked.parse()`
- Dark-themed markdown CSS (headings, code blocks, tables, lists)
- Empty state: "No content available" when `exists: false`
- Marked.js CDN `<script>` tag added to HTML head

**Deliverable**: Clicking a tree node loads and renders entity properties + markdown content. Info toggle active.

**Complexity**: M

---

### Task 7: History Drawer + Cross-View Navigation (~300 LOC)

**Scope**:
- History drawer renderer: reverse-chronological table of transitions
- Per-row: date | from-badge → to-badge | note
- Transitions toggle button wires to drawer
- "Open" button in drawer returns to Info view
- `navigateToEntity(key)`: parse key → find node in tree → expand ancestors → scroll into view → select → transition to entity view
- Keyboard: `Escape` closes entity/doc view back to dashboard
- End-to-end flow testing: dashboard key click → entity view → history → back to dashboard
- Doc View: plain markdown rendering for doc nodes (reuse Entity View markdown rendering, hide Info/Transitions toggle)

**Deliverable**: Complete SPA with all 4 states, cross-view navigation, and history drawer.

**Complexity**: M

### Sequencing

```
Task 1 (Shell + CSS + Colors)
   └── Task 2 (API Client + Pick Folder Load)
         ├── Task 3 (Sidebar Tree)
         │     ├── Task 4 (Dashboard: Breakdown + Progress)
         │     │     └── Task 5 (Dashboard: Activity + Stale)
         │     └── Task 6 (Entity View)
         │           └── Task 7 (History + Navigation + Doc View)
```

Tasks 4–5 can proceed in parallel with Task 6 once Task 3 is complete.

---

## 7. Testing Strategy

The SPA is a single HTML file with embedded JS — traditional unit testing frameworks would require introducing a test toolchain. Testing is therefore a hybrid of manual/visual verification and targeted automation.

### 7.1 Visual Verification (Manual)

For each task, verify the output against `docs/status-viewer-ui.md` and the reference screenshots at `/home/jwwel/Pictures/status/tracker/`:

- **Task 1**: Pick Folder screen matches spec layout, dark theme matches reference
- **Task 3**: Sidebar tree matches screenshot indentation, expand/collapse arrows, status dots
- **Task 4**: Status Breakdown cards match screenshot 3 card layout
- **Task 5**: Active Transitions and Stale Entities rows match screenshot format
- **Task 6**: Entity view matches screenshots 1 and 2 (properties panel + content pane)
- **Task 7**: History drawer matches screenshot 2 transition table layout

### 7.2 Functional Verification (Manual End-to-End)

Run the server with E27-F02 API endpoints serving real project data. Manually exercise:

1. Pick Folder → Dashboard transition
2. Refresh button reloads all cached data
3. Click each entity type (epic, feature, task, bug, idea, change-card) from sidebar → entity view renders
4. Click doc node → doc view renders plain markdown
5. Click entity key in Status Breakdown (if applicable), Feature Progress, Active Transitions, Stale Entities → navigates to entity view, tree node highlighted, ancestors expanded, scrolled into view
6. Info ↔ Transitions toggle in entity view
7. Escape key returns to dashboard
8. Dashboard title click returns to dashboard
9. Pick Folder button returns to Pick Folder state

### 7.3 Automated Smoke Tests (Go side)

In `internal/viewer/assets_test.go`:
- `TestViewerHTMLEmbedded` — asserts `ViewerHTML` is non-empty and contains required markers (`<!DOCTYPE html>`, `<script>`, `STATUS_COLORS`)
- `TestViewerHTMLContainsRequiredClasses` — greps for required CSS class names and HTML ids

### 7.4 Browser Compatibility

- Target: latest Chrome and Safari (primary)
- Features used: `fetch`, ES2020 syntax, `Set`, CSS custom properties — all widely supported
- Explicitly not targeted: Internet Explorer, mobile browsers, legacy Edge

### 7.5 Marked.js Fallback

If the CDN-hosted Marked.js fails to load (offline/blocked), markdown content is rendered as preformatted text with a warning banner. Task 6 implements this fallback check.

### 7.6 API Contract Drift Guard

Response type shapes are documented as JSDoc comment blocks at the top of the `<script>` block, making contract drift visible during review. Any field access guarded with `?.` optional chaining.

---

## 8. Dependencies

### Blocking Dependencies

| Dependency | Status | Impact |
|---|---|---|
| **E27-F02 — Viewer API Endpoints** | `in_specification` | SPA cannot function without these 7 endpoints. Tasks 2–7 depend on a live API. |
| **E27-F01 — DB Init Extraction** | required transitively | E27-F02 requires DB init; SPA depends on E27-F02. |

### Non-Blocking Dependencies

| Dependency | Status | Impact |
|---|---|---|
| **E27-F05 — Server Wiring** | separate feature | Handles `GET /` registration and `go:embed` wiring. SPA can be developed against a standalone dev server (e.g., `python -m http.server`) with CORS disabled in the browser, or against a dev-only `GET /` route. |
| **E27-F04 — `shark web` CLI Command** | separate feature | Entry point for launching the viewer. Not required for SPA development. |

### Technical Constraints

- **No build step**: single HTML file, all JS/CSS inline
- **No npm dependencies**: Marked.js loaded from CDN only
- **`go:embed` constraint**: `viewer.html` must live at `internal/viewer/assets/viewer.html` to be embedded by `internal/viewer/assets.go`
- **CORS**: server must allow localhost origin for SPA → API calls
- **Read-only**: SPA makes no write/mutation calls

### External Dependencies

- **Marked.js** (CDN): `https://cdn.jsdelivr.net/npm/marked/marked.min.js` — pinned to a major version for stability

---

## 9. Acceptance Criteria (Feature-Level)

The feature is complete when all of the following hold:

- [ ] `internal/viewer/assets/viewer.html` exists and is a single self-contained HTML file
- [ ] `internal/viewer/assets.go` embeds `viewer.html` via `go:embed`
- [ ] SPA renders the Pick Folder screen on initial load in dark slate theme
- [ ] Clicking Pick Folder triggers hierarchy + summary + recent-activity + workflow-meta loads
- [ ] On successful load, SPA transitions to Dashboard view
- [ ] Dashboard renders all 4 sections: Status Breakdown, Feature Progress, Active Transitions, Stale Entities
- [ ] Sidebar renders the full Hierarchy tree with correct indentation (16/40/56px), expand/collapse, and selected state
- [ ] Sidebar renders flat sections (Ideas, Change Cards, Tags, Tech Debt) when present
- [ ] All status badges and dots use the exact hex colors from the mapping table
- [ ] Clicking a tree entity node transitions to Entity View
- [ ] Entity View shows properties panel, Info/Transitions toggle, and markdown content rendered via Marked.js
- [ ] Clicking "Transitions" shows the history drawer with reverse-chronological transitions
- [ ] Clicking a doc node transitions to Doc View with plain markdown
- [ ] Clicking an entity key anywhere in the dashboard navigates to that entity (expand ancestors, scroll, select)
- [ ] `Escape` key returns to Dashboard from Entity or Doc View
- [ ] Header "Dashboard" click returns to Dashboard
- [ ] Header "Refresh" button reloads all cached data
- [ ] Header "Pick Folder" button returns to Pick Folder state
- [ ] Stale Entities filter excludes terminal statuses and only shows items with `updated_at > 7 days ago`
- [ ] SPA tolerates missing optional fields without crashing
- [ ] Marked.js CDN failure falls back to preformatted text with warning
- [ ] Visual output matches reference screenshots at `/home/jwwel/Pictures/status/tracker/`
- [ ] `go test ./internal/viewer/...` passes (embedded file smoke tests)

---

## 10. Open Questions

1. **Feature Progress data source**: Does `/hierarchy` return enough data to compute progress percentages, or must the SPA call `/features/{key}/tasks` for each feature? To be confirmed against final E27-F02 spec.
2. **Workflow-meta shape**: Exact field list of `/workflow-meta` response — used to supplement `STATUS_COLORS` table. To be confirmed.
3. **Entity count pills source**: Direct from `/summary` totals, or derived from hierarchy walk? Default: use `/summary.{epics,features,tasks,bugs}.total`.

These questions do not block specification advancement — they will be resolved during implementation in collaboration with E27-F02.

---

*End of Specification*
