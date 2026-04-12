---
feature_key: E27-F03-single-file-spa-ide-style-dark-dashboard-interface
epic_key: E27
title: "Test Plan — E27-F03: Single-File SPA (IDE-Style Dark Dashboard Interface)"
created: 2026-04-11
status: ready
---

# Test Plan: E27-F03 — Single-File SPA (IDE-Style Dark Dashboard Interface)

**Feature**: `E27-F03 — Single-File SPA - IDE-Style Dark Dashboard Interface`
**Spec**: [spec.md](spec.md)
**UI Spec**: [status-viewer-ui.md](../../../../docs/status-viewer-ui.md)
**Research**: [RESEARCH-REPORT.md](RESEARCH-REPORT.md)

---

## 1. Scope and Approach

The SPA is a ~2300-LOC single HTML file with embedded CSS and JS. Because it has no build toolchain, testing is a hybrid strategy:

| Layer | Type | Who |
|---|---|---|
| Go embed smoke tests | Automated (Go test) | CI |
| Visual / structural verification | Manual per-task checkpoint | Developer/QA |
| Functional end-to-end scenarios | Manual, live server | Developer/QA |
| API integration contract checks | Manual, network tab / mock | Developer |
| Exploratory / edge cases | Manual | QA |

**Pass criterion**: all 22 Acceptance Criteria satisfied AND all test cases in this plan marked PASS.

---

## 2. Test Environment

| Item | Requirement |
|---|---|
| Browser | Chrome (latest) — primary. Safari (latest) — secondary. |
| Server | E27-F02 API endpoints running locally (`localhost:<port>`) |
| Project data | Real shark project data with epics, features, tasks, bugs, ideas, change cards |
| Stale data | At least one entity with `updated_at` > 7 days ago in a non-terminal status (for stale entity tests) |
| Terminal data | At least one entity each in: `done`, `cancelled`, `closed`, `verified`, `rejected`, `promoted`, `approved` |
| Markdown content | At least one entity with an associated file that has `exists: true` |
| Marked.js | CDN available (happy-path) AND CDN blocked (fallback test) |

---

## 3. AC Test Matrix

Maps each of the 22 Acceptance Criteria from `spec.md §9` to one or more test case IDs.

| # | Acceptance Criterion | Test Case(s) | Priority |
|---|---|---|---|
| AC-01 | `internal/viewer/assets/viewer.html` exists as a single self-contained HTML file | TC-SMOKE-01 | Critical |
| AC-02 | `internal/viewer/assets.go` embeds `viewer.html` via `go:embed` | TC-SMOKE-01, TC-SMOKE-02 | Critical |
| AC-03 | SPA renders Pick Folder screen on initial load in dark slate theme | TC-VIS-01 | Critical |
| AC-04 | Clicking Pick Folder triggers hierarchy + summary + recent-activity + workflow-meta loads | TC-API-01, TC-NAV-01 | Critical |
| AC-05 | On successful load, SPA transitions to Dashboard view | TC-NAV-01 | Critical |
| AC-06 | Dashboard renders all 4 sections: Status Breakdown, Feature Progress, Active Transitions, Stale Entities | TC-VIS-04, TC-VIS-05, TC-VIS-06, TC-VIS-07 | Critical |
| AC-07 | Sidebar renders full Hierarchy tree with correct indentation (16/40/56px), expand/collapse, selected state | TC-TREE-01, TC-TREE-02, TC-TREE-03, TC-TREE-04 | Critical |
| AC-08 | Sidebar renders flat sections (Ideas, Change Cards, Tags, Tech Debt) when present | TC-TREE-05 | High |
| AC-09 | All status badges and dots use the exact hex colors from the mapping table | TC-COLOR-01 through TC-COLOR-20 | Critical |
| AC-10 | Clicking a tree entity node transitions to Entity View | TC-NAV-02 | Critical |
| AC-11 | Entity View shows properties panel, Info/Transitions toggle, and markdown content via Marked.js | TC-VIS-08, TC-VIS-09 | Critical |
| AC-12 | Clicking "Transitions" shows history drawer with reverse-chronological transitions | TC-VIS-10, TC-NAV-05 | High |
| AC-13 | Clicking a doc node transitions to Doc View with plain markdown | TC-NAV-04 | High |
| AC-14 | Clicking an entity key in dashboard navigates to that entity (expand ancestors, scroll, select) | TC-NAV-03 | Critical |
| AC-15 | Escape key returns to Dashboard from Entity or Doc View | TC-KB-01 | High |
| AC-16 | Header "Dashboard" click returns to Dashboard | TC-NAV-06 | High |
| AC-17 | Header "Refresh" button reloads all cached data | TC-NAV-07 | High |
| AC-18 | Header "Pick Folder" button returns to Pick Folder state | TC-NAV-08 | Medium |
| AC-19 | Stale Entities filter excludes terminal statuses and only shows items with `updated_at > 7 days ago` | TC-STALE-01, TC-STALE-02, TC-STALE-03 | High |
| AC-20 | SPA tolerates missing optional fields without crashing | TC-API-05, TC-API-06, TC-API-07 | High |
| AC-21 | Marked.js CDN failure falls back to preformatted text with warning | TC-API-08 | Medium |
| AC-22 | `go test ./internal/viewer/...` passes (embedded file smoke tests) | TC-SMOKE-01, TC-SMOKE-02, TC-SMOKE-03 | Critical |

---

## 4. Visual Verification Tests

These tests verify each application state and visual component matches `docs/status-viewer-ui.md` and the reference screenshots at `/home/jwwel/Pictures/status/tracker/`.

---

### TC-VIS-01 — Pick Folder Screen Layout

**Story Link**: AC-03  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- Server running with no cached state
- Browser opened to `http://localhost:<port>/` (fresh session or hard-refresh with cache cleared)

**Test Steps**
1. Navigate to `http://localhost:<port>/`
2. Observe the initial rendered state

**Expected Results**
- No sidebar visible
- No header bar visible
- Content is centered vertically and horizontally
- Large faded SVG folder icon displayed at top
- "Status Tracker" heading rendered in large primary text (white/near-white)
- Instruction text: "Select a shark project folder to browse." (or equivalent per spec)
- "Pick Folder" button rendered in accent blue, prominently sized
- Small note text: "Read-only access. Does not modify project state."
- Background is deep dark (near-black) slate color — no light theme bleed
- Compare against reference screenshot `20260411_113043.jpg` sidebar context

**Status**: Not Run

---

### TC-VIS-02 — Dark Theme Baseline

**Story Link**: AC-03  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- Any loaded state (Pick Folder or Dashboard)

**Test Steps**
1. Using browser DevTools, inspect CSS custom properties on `:root` or `html`
2. Verify the following properties exist:
   - `--bg` (or equivalent): deep dark value (e.g., `#1a1a2e` or similar near-black)
   - `--fg` (or equivalent): near-white primary text
   - `--accent`: blue value in the `#4d77ff` range
   - `--sidebar-width`: `320px`
3. Verify no default browser light-mode styles override the dark theme

**Expected Results**
- CSS custom properties present and using dark palette
- `--sidebar-width` resolves to `320px`
- Accent color is blue
- Page background matches the reference screenshots (deep slate / near-black)

**Status**: Not Run

---

### TC-VIS-03 — Header Bar Layout

**Story Link**: AC-06 (dashboard presence), AC-16, AC-17, AC-18  
**Priority**: High  
**Type**: Visual (Manual)

**Preconditions**
- SPA in Dashboard state (hierarchy loaded successfully)

**Test Steps**
1. Inspect header bar
2. Verify left side elements
3. Verify right side elements
4. Verify header height

**Expected Results**
- Header fixed at top of viewport (~40–50px height)
- Left side: "Dashboard" text (clickable link/button)
- Left side: Entity count pills visible — labels like "2 Epics", "5 Features", "3 Tasks", "1 Bugs", "1 Ideas"
- Entity count pills match actual data from `/api/v1/viewer/summary`
- Right side: "Refresh" button
- Right side: "Pick Folder" button
- Both buttons styled consistently (accent or muted color)

**Status**: Not Run

---

### TC-VIS-04 — Dashboard Status Breakdown Cards

**Story Link**: AC-06  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- SPA in Dashboard state
- Project has at least: 1 epic, 1 feature, 1 task, 1 bug

**Test Steps**
1. Navigate to Dashboard view
2. Observe Status Breakdown section
3. Verify one card exists per entity type that has items
4. For each card, inspect its contents

**Expected Results**
- Cards arranged in a grid layout
- Each card shows: uppercase type label (e.g., `EPICS`), large total count number
- Each card shows status badges below the total count
- Status badges have colored backgrounds matching `STATUS_COLORS` table
- Badge labels show status name + count
- Example: EPICS card with "2" total, IN_PROGRESS badge (yellow `#e7b30e`), BLOCKED badge (dark orange `#8c5700`)
- Visual matches reference screenshot `20260411_113043.jpg` Status Overview section

**Status**: Not Run

---

### TC-VIS-05 — Dashboard Feature Progress Bars

**Story Link**: AC-06  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- SPA in Dashboard state
- At least 2 features exist with tasks at varying completion

**Test Steps**
1. Observe Feature Progress section in Dashboard
2. Verify layout of each feature row
3. Click a feature key

**Expected Results**
- Each row: feature key (monospace, accent blue) | feature title (muted) | horizontal progress bar (blue fill) | percentage %
- Progress bar fill percentage matches `(completed_tasks / total_tasks) * 100`
- Feature key is clickable — clicking navigates to Entity View for that feature
- Visual matches reference screenshot `20260411_113043.jpg` Feature Progress section

**Status**: Not Run

---

### TC-VIS-06 — Dashboard Active Transitions

**Story Link**: AC-06  
**Priority**: High  
**Type**: Visual (Manual)

**Preconditions**
- SPA in Dashboard state
- At least 5 task status transitions exist in project history

**Test Steps**
1. Observe Active Transitions section in Dashboard
2. Verify section title "Active Transitions"
3. Inspect row layout for 3 sample rows
4. Verify maximum rows shown

**Expected Results**
- Section shows last 25 transitions, newest first
- Each row: date | key (accent blue, clickable) | from-badge | "→" | to-badge | note text | "Open" button (if entity has content)
- From-badge and to-badge use `getStatusColor()` — colored pill backgrounds
- "Open" button only appears when the transition's entity has linked content
- Clicking key or Open button navigates to entity view
- Visual matches reference screenshot `20260411_113043.jpg` Recent Activity section

**Status**: Not Run

---

### TC-VIS-07 — Dashboard Stale Entities

**Story Link**: AC-06, AC-19  
**Priority**: High  
**Type**: Visual (Manual)

**Preconditions**
- At least one entity with `updated_at` > 7 days ago in a non-terminal status
- At least one entity in a terminal status with old `updated_at` (to verify exclusion)

**Test Steps**
1. Observe Stale Entities section in Dashboard
2. Verify section title "Stale Entities"
3. Verify row contents
4. Verify sorting order

**Expected Results**
- Section title "Stale Entities" visible
- Each row: key (clickable) | status badge | title | last update age ("12 days ago")
- Rows sorted by staleness descending (oldest `updated_at` first)
- Terminal-status entities DO NOT appear (see TC-STALE-02 for detail)
- Entities updated within 7 days DO NOT appear
- Clicking key navigates to entity view

**Status**: Not Run

---

### TC-VIS-08 — Entity View Properties Panel

**Story Link**: AC-11  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- SPA in Entity View state (clicked a task node in sidebar)

**Test Steps**
1. Click any task node in sidebar tree
2. Inspect properties panel in content area
3. Verify all required fields are displayed

**Expected Results**
- Properties panel shows key-value grid with fields: `path`, `key`, `status`, `type`, `parent`, `agent`, `created_at`, `updated_at`, `file_path`, `linked_content_path`
- `status` field rendered as a colored badge using `getStatusColor(status)`
- `file_path` field has a copy-to-clipboard button
- `linked_content_path` field has a copy-to-clipboard button (if present)
- Visual matches reference screenshot `20260411_112944.jpg` properties panel

**Status**: Not Run

---

### TC-VIS-09 — Entity View Markdown Content

**Story Link**: AC-11  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- SPA in Entity View with Info toggle active
- Selected entity has a file that `exists: true`

**Test Steps**
1. Click an entity node that has associated markdown content
2. Verify Info toggle is active by default
3. Inspect rendered markdown in content pane

**Expected Results**
- "Info" toggle button shown as active (highlighted)
- Markdown rendered via Marked.js — headings, paragraphs, code blocks, lists, tables all styled
- Dark-themed markdown: headings in light color, code blocks with muted background, no bright white content areas
- Entity name displayed prominently in content toolbar
- Visual matches reference screenshot `20260411_112944.jpg` content area

**Status**: Not Run

---

### TC-VIS-09b — Entity View No Content Placeholder

**Story Link**: AC-11  
**Priority**: Medium  
**Type**: Functional (Manual)

**Preconditions**
- An entity exists with no associated file (file API returns `exists: false`)

**Test Steps**
1. Navigate to an entity that has no file content
2. Observe Info view

**Expected Results**
- Placeholder text "No content available for this entity." displayed
- No JavaScript error in browser console
- Toggle buttons still visible

**Status**: Not Run

---

### TC-VIS-10 — Entity View History Drawer

**Story Link**: AC-12  
**Priority**: High  
**Type**: Visual (Manual)

**Preconditions**
- SPA in Entity View state
- Selected entity has at least 3 status transitions in history

**Test Steps**
1. Click entity with known transition history
2. Click "Transitions" toggle button
3. Inspect history drawer

**Expected Results**
- Content pane switches to history drawer
- "Transitions" toggle button shown as active
- Table of transitions in reverse-chronological order (newest first)
- Each row: date | from-badge | "→" | to-badge | note text
- Badge backgrounds colored using `getStatusColor()`
- "Open" button visible if entity has linked content — clicking returns to Info view
- If no history: "No transition history." placeholder shown
- Visual matches reference screenshot `20260411_113020.jpg`

**Status**: Not Run

---

### TC-VIS-11 — Doc View Plain Markdown

**Story Link**: AC-13  
**Priority**: High  
**Type**: Visual (Manual)

**Preconditions**
- At least one doc node in sidebar tree (epic or feature has associated design doc)

**Test Steps**
1. Expand a feature in sidebar
2. Click a doc node (italic, muted)
3. Observe Doc View content area

**Expected Results**
- Content area renders plain markdown from the doc file
- No Info/Transitions toggle buttons visible
- Properties panel shows doc metadata (path, title, parent entity)
- Markdown rendering is dark-themed (same styling as Entity View markdown)
- No history drawer accessible

**Status**: Not Run

---

### TC-VIS-12 — Sidebar Tree Indentation

**Story Link**: AC-07  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- SPA in Dashboard or Entity View state with expanded hierarchy

**Test Steps**
1. Expand at least one epic with a feature that has tasks
2. Using DevTools, measure left-padding on each node type

**Expected Results**
- Epic node: `16px` left padding, **bold** font weight
- Feature node: `40px` left padding, normal font weight
- Task node: `56px` left padding, normal font weight
- Doc at epic level: `16px` left padding, italic and muted color
- Doc at feature level: `40px` left padding, italic and muted color
- Flat section nodes (Ideas, Change Cards, etc.): `18px` left padding
- Visual matches reference screenshot `20260411_112944.jpg` sidebar layout

**Status**: Not Run

---

### TC-VIS-13 — Sidebar Node Anatomy

**Story Link**: AC-07  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- Sidebar visible with at least one epic expanded showing a feature and task

**Test Steps**
1. Inspect a task node in the sidebar
2. Verify each element of the node's left-to-right composition

**Expected Results**
- No arrow (▶/▼) on leaf nodes (tasks, docs)
- Arrow (▶/▼) present on expandable nodes (epics, features with children)
- Status dot: 8–10px filled circle, background color = `getStatusColor(node.status)`
- Key: monospace font, accent blue color
- Title: sans-serif, muted gray, `text-overflow: ellipsis` when truncated

**Status**: Not Run

---

## 5. Status Color Tests

Each test verifies the rendered background color (CSS `background-color` or `background`) of a status badge or dot against the exact hex value from the `STATUS_COLORS` table.

**Test method**: In browser DevTools, inspect the element, check computed `background-color`. Convert `rgb()` values to hex for comparison.

**Common setup**: SPA in Dashboard state with test data covering a wide range of statuses.

---

### TC-COLOR-01 — draft = #8c8c8c

**Steps**: Find any entity or badge with status `draft`. Inspect badge/dot background.  
**Expected**: `background-color: rgb(140, 140, 140)` (≡ `#8c8c8c`)  
**Status**: Not Run

---

### TC-COLOR-02 — todo = #4d4dbb

**Steps**: Find entity/badge with status `todo`. Inspect.  
**Expected**: `background-color: rgb(77, 77, 187)` (≡ `#4d4dbb`)  
**Status**: Not Run

---

### TC-COLOR-03 — in_progress = #e7b30e

**Steps**: Find entity/badge with status `in_progress`. Inspect.  
**Expected**: `background-color: rgb(231, 179, 14)` (≡ `#e7b30e`)  
**Status**: Not Run

---

### TC-COLOR-04 — open = #e7b30e (same as in_progress)

**Steps**: Find entity/badge with status `open`. Inspect.  
**Expected**: `background-color: rgb(231, 179, 14)` (≡ `#e7b30e`)  
**Status**: Not Run

---

### TC-COLOR-05 — done = #4d4269

**Steps**: Find entity/badge with status `done`. Inspect.  
**Expected**: `background-color: rgb(77, 66, 105)` (≡ `#4d4269`)  
**Status**: Not Run

---

### TC-COLOR-06 — blocked = #8c5700

**Steps**: Find entity/badge with status `blocked`. Inspect.  
**Expected**: `background-color: rgb(140, 87, 0)` (≡ `#8c5700`)  
**Status**: Not Run

---

### TC-COLOR-07 — cancelled = #4b4b36

**Steps**: Find entity/badge with status `cancelled`. Inspect.  
**Expected**: `background-color: rgb(75, 75, 54)` (≡ `#4b4b36`)  
**Status**: Not Run

---

### TC-COLOR-08 — triaged = #e7b500

**Steps**: Find bug with status `triaged`. Inspect badge.  
**Expected**: `background-color: rgb(231, 181, 0)` (≡ `#e7b500`)  
**Status**: Not Run

---

### TC-COLOR-09 — verified = #4d4269

**Steps**: Find bug with status `verified`. Inspect badge.  
**Expected**: `background-color: rgb(77, 66, 105)` (≡ `#4d4269`)  
**Status**: Not Run

---

### TC-COLOR-10 — closed = #4b4b26

**Steps**: Find entity with status `closed`. Inspect badge.  
**Expected**: `background-color: rgb(75, 75, 38)` (≡ `#4b4b26`)  
**Status**: Not Run

---

### TC-COLOR-11 — wont_fix = #4b4b26

**Steps**: Find bug with status `wont_fix`. Inspect badge.  
**Expected**: `background-color: rgb(75, 75, 38)` (≡ `#4b4b26`)  
**Status**: Not Run

---

### TC-COLOR-12 — identified = #8c8c8c

**Steps**: Find entity with status `identified`. Inspect badge.  
**Expected**: `background-color: rgb(140, 140, 140)` (≡ `#8c8c8c`)  
**Status**: Not Run

---

### TC-COLOR-13 — confirmed = #4d4dbb

**Steps**: Find entity with status `confirmed`. Inspect badge.  
**Expected**: `background-color: rgb(77, 77, 187)` (≡ `#4d4dbb`)  
**Status**: Not Run

---

### TC-COLOR-14 — proposed = #4d4dbb

**Steps**: Find change card with status `proposed`. Inspect badge.  
**Expected**: `background-color: rgb(77, 77, 187)` (≡ `#4d4dbb`)  
**Status**: Not Run

---

### TC-COLOR-15 — evaluating = #e7b500

**Steps**: Find entity with status `evaluating`. Inspect badge.  
**Expected**: `background-color: rgb(231, 181, 0)` (≡ `#e7b500`)  
**Status**: Not Run

---

### TC-COLOR-16 — accepted = #4d4269

**Steps**: Find entity with status `accepted`. Inspect badge.  
**Expected**: `background-color: rgb(77, 66, 105)` (≡ `#4d4269`)  
**Status**: Not Run

---

### TC-COLOR-17 — promoted = #8c4dbb

**Steps**: Find idea with status `promoted`. Inspect badge.  
**Expected**: `background-color: rgb(140, 77, 187)` (≡ `#8c4dbb`)  
**Status**: Not Run

---

### TC-COLOR-18 — rejected = #8c4d6c

**Steps**: Find entity with status `rejected`. Inspect badge.  
**Expected**: `background-color: rgb(140, 77, 108)` (≡ `#8c4d6c`)  
**Status**: Not Run

---

### TC-COLOR-19 — reopened = #e7b500

**Steps**: Find entity with status `reopened`. Inspect badge.  
**Expected**: `background-color: rgb(231, 181, 0)` (≡ `#e7b500`)  
**Status**: Not Run

---

### TC-COLOR-20 — approved = #4d4269

**Steps**: Find entity with status `approved`. Inspect badge.  
**Expected**: `background-color: rgb(77, 66, 105)` (≡ `#4d4269`)  
**Status**: Not Run

---

### TC-COLOR-21 — Unknown status fallback = #666666

**Steps**: Directly invoke `getStatusColor('__unknown_xyz__')` in browser console.  
**Expected**: Returns `'#666666'`  
**Status**: Not Run

---

## 6. Sidebar Tree Tests

### TC-TREE-01 — Epics Start Expanded

**Story Link**: AC-07  
**Priority**: Critical  
**Type**: Functional (Manual)

**Preconditions**
- SPA freshly loaded to Dashboard state (no prior interaction)

**Test Steps**
1. Observe sidebar immediately after hierarchy loads
2. Check all epic nodes

**Expected Results**
- All epic nodes show `▼` (expanded) arrow by default
- Feature children are visible under each epic (though features themselves start collapsed)
- `expandedEpics` Set was seeded with all epic keys on initial load
- No user interaction required to see epics expanded

**Status**: Not Run

---

### TC-TREE-02 — Features Start Collapsed

**Story Link**: AC-07  
**Priority**: Critical  
**Type**: Functional (Manual)

**Preconditions**
- SPA freshly loaded to Dashboard state

**Test Steps**
1. Observe feature nodes in the sidebar on initial load

**Expected Results**
- All feature nodes show `▶` (collapsed) arrow
- Task children are NOT visible without explicit user expand
- Feature docs are NOT visible without explicit user expand

**Status**: Not Run

---

### TC-TREE-03 — Expand/Collapse Arrow Toggle

**Story Link**: AC-07  
**Priority**: Critical  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Dashboard state, at least one feature with tasks visible

**Test Steps**
1. Click the `▶` arrow on a feature node
2. Verify children appear, arrow changes to `▼`
3. Click the `▼` arrow again
4. Verify children disappear, arrow returns to `▶`
5. Verify clicking arrow does NOT select the node (no entity view opens)
6. Verify clicking the node body (key/title area) DOES select the node

**Expected Results**
- Step 2: Feature expands, tasks become visible, arrow = `▼`
- Step 4: Feature collapses, tasks hidden, arrow = `▶`
- Step 5: State remains Dashboard (or unchanged) — no navigation occurs from arrow click
- Step 6: Entity View opens for the feature

**Status**: Not Run

---

### TC-TREE-04 — Selected Node Styling

**Story Link**: AC-07  
**Priority**: Critical  
**Type**: Visual (Manual)

**Preconditions**
- SPA with sidebar visible

**Test Steps**
1. Click a task node in the sidebar
2. Inspect the selected node's CSS

**Expected Results**
- Selected node has darker background tint (black with slight transparency)
- Selected node has 3px left border in accent blue
- Selected node title text color changes to primary (white) instead of muted gray
- Previously unselected nodes remain unstyled
- Only one node appears selected at a time

**Status**: Not Run

---

### TC-TREE-05 — Flat Sections Render When Present

**Story Link**: AC-08  
**Priority**: High  
**Type**: Functional (Manual)

**Preconditions**
- Project has at least: 1 idea AND 1 change card

**Test Steps**
1. Scroll sidebar below the Hierarchy section
2. Check for flat sections

**Expected Results**
- "Ideas" section visible as sticky uppercase header with flat list of idea nodes below
- "Change Cards" section visible with flat list
- Each flat node: status dot | key (accent blue, mono) | title (muted) — same format as hierarchy nodes
- Flat nodes use 18px left padding (not the 16/40/56px hierarchy indentation)
- "Tags" and "Tech Debt" sections ONLY appear if project has data for them

**Status**: Not Run

---

### TC-TREE-06 — Expansion Persists Across Refresh

**Story Link**: AC-07 (expand state persistence spec §2.6.4)  
**Priority**: Medium  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Dashboard, some features manually expanded by user

**Test Steps**
1. Manually expand 2 feature nodes
2. Click the "Refresh" button in the header
3. Observe sidebar after refresh completes

**Expected Results**
- Previously expanded features remain expanded after refresh
- `expandedEpics` and `expandedFeatures` Sets are not reset by refresh
- Epics remain expanded (they are always expanded by default)

**Status**: Not Run

---

## 7. Navigation Tests

### TC-NAV-01 — Pick Folder → Dashboard Transition

**Story Link**: AC-04, AC-05  
**Priority**: Critical  
**Type**: Functional (Manual)

**Preconditions**
- SPA on Pick Folder screen
- E27-F02 API server running with valid project data

**Test Steps**
1. Open browser DevTools Network tab
2. Click "Pick Folder" button
3. Observe network requests made
4. Observe UI transition

**Expected Results**
- 4 network requests fired (in order or parallel):
  - `GET /api/v1/viewer/workflow-meta`
  - `GET /api/v1/viewer/hierarchy`
  - `GET /api/v1/viewer/summary`
  - `GET /api/v1/viewer/recent-activity?limit=25`
- All 4 requests return 200 OK
- SPA transitions to Dashboard state
- Pick Folder screen disappears
- Header, sidebar, and dashboard content sections appear
- No JavaScript errors in console

**Status**: Not Run

---

### TC-NAV-02 — Sidebar Node Click → Entity View

**Story Link**: AC-10  
**Priority**: Critical  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Dashboard state
- Sidebar visible with at least one task node visible (feature expanded)

**Test Steps**
1. Click a task node body (not the arrow) in the sidebar
2. Observe the content area and network activity

**Expected Results**
- `GET /api/v1/viewer/file/{key}` request fired
- `GET /api/v1/viewer/history/{key}` request fired
- Content area transitions to Entity View
- Properties panel rendered with entity metadata
- Info toggle active by default
- Markdown content rendered (or "No content" placeholder if `exists: false`)
- Clicked node shows selected state styling (darker background, left accent border)

**Status**: Not Run

---

### TC-NAV-03 — Dashboard Key Click → Entity Navigation

**Story Link**: AC-14  
**Priority**: Critical  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Dashboard state
- Feature Progress section or Active Transitions section has clickable keys

**Test Steps**
1. In the Feature Progress section, click a feature key link
2. Observe the Entity View content and sidebar behavior
3. Return to Dashboard
4. In the Active Transitions section, click a transition's entity key
5. Observe the same behaviors

**Expected Results**
- Entity View opens for the clicked key
- Sidebar tree scrolls to show the entity's node
- If the entity's feature was collapsed, it is now expanded to reveal the node
- The entity's node is selected (shows selected state styling)
- Both Feature Progress keys and Active Transitions keys correctly navigate

**Status**: Not Run

---

### TC-NAV-04 — Doc Node Click → Doc View

**Story Link**: AC-13  
**Priority**: High  
**Type**: Functional (Manual)

**Preconditions**
- At least one doc node visible in sidebar tree

**Test Steps**
1. Expand a feature that has a doc associated
2. Click the doc node (should appear italic/muted in sidebar)
3. Observe content area

**Expected Results**
- Content area shows Doc View: plain markdown rendering
- `GET /api/v1/viewer/file/{key}` request fired for the doc
- No Info/Transitions toggle buttons visible
- Properties panel shows doc metadata
- Doc node selected in sidebar

**Status**: Not Run

---

### TC-NAV-05 — Entity View Info ↔ Transitions Toggle

**Story Link**: AC-12  
**Priority**: High  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Entity View state
- Entity has both file content and history

**Test Steps**
1. Verify "Info" toggle is active by default
2. Click "Transitions" toggle
3. Verify History Drawer appears
4. Click "Info" toggle
5. Verify Info view returns
6. In History drawer, click "Open" button (if present)
7. Verify Info view returns

**Expected Results**
- Step 2: History drawer renders, "Transitions" toggle highlighted
- Step 3: No additional API calls (history already loaded)
- Step 4: Markdown content view returns, "Info" toggle highlighted
- Step 6: Same result as step 4 — returns to Info view
- No state reload occurs on toggle switches (client-side only)

**Status**: Not Run

---

### TC-NAV-06 — Header "Dashboard" Click → Return to Dashboard

**Story Link**: AC-16  
**Priority**: High  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Entity View state

**Test Steps**
1. Navigate to Entity View (click any sidebar node)
2. Click "Dashboard" text/link in the header
3. Observe result

**Expected Results**
- Content area returns to Dashboard view with all 4 sections
- `selectedKey` cleared
- No entity node selected in sidebar
- No additional API calls unless dashboard data is stale (no forced reload)

**Status**: Not Run

---

### TC-NAV-07 — Refresh Button Reloads Data

**Story Link**: AC-17  
**Priority**: High  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Dashboard state

**Test Steps**
1. Open DevTools Network tab
2. Click "Refresh" button in header
3. Observe network requests

**Expected Results**
- All cached data refetched:
  - `GET /api/v1/viewer/hierarchy`
  - `GET /api/v1/viewer/summary`
  - `GET /api/v1/viewer/recent-activity?limit=25`
  - `GET /api/v1/viewer/workflow-meta`
- Dashboard re-renders with refreshed data
- Previously expanded epics remain expanded (expansion state preserved)
- No page reload (in-place refresh)

**Status**: Not Run

---

### TC-NAV-08 — Header "Pick Folder" Button Returns to Pick Folder State

**Story Link**: AC-18  
**Priority**: Medium  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Dashboard state

**Test Steps**
1. Click "Pick Folder" button in header
2. Observe UI state

**Expected Results**
- Pick Folder screen displayed
- Sidebar and header hidden
- Centered layout with folder icon, heading, button visible
- All cached state cleared (or clearly reset for next load)

**Status**: Not Run

---

## 8. Keyboard Tests

### TC-KB-01 — Escape Key Returns to Dashboard

**Story Link**: AC-15  
**Priority**: High  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Entity View state

**Test Steps**
1. Navigate to Entity View (click any tree node)
2. Press `Escape` key
3. Verify return to Dashboard
4. Navigate to Doc View
5. Press `Escape` key
6. Verify return to Dashboard

**Expected Results**
- Step 3: Dashboard view rendered, Entity View removed
- Step 6: Dashboard view rendered, Doc View removed
- No JavaScript errors
- Selected node styling cleared

**Status**: Not Run

---

### TC-KB-02 — Escape from History Drawer Closes Drawer First

**Story Link**: AC-15 + spec §5.4 behavior  
**Priority**: Medium  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Entity View with History Drawer open (Transitions toggle active)

**Test Steps**
1. Navigate to Entity View
2. Click "Transitions" toggle to open history drawer
3. Press `Escape` once
4. Observe result
5. If drawer closed but still in Entity View, press `Escape` again

**Expected Results**
- Step 3 (first Escape): drawer closes, returns to Info view within Entity View — does NOT jump all the way to Dashboard
- Step 5 (second Escape): returns to Dashboard
- This matches spec §5.4: "if a drawer is open, closes the drawer first"

**Status**: Not Run

---

## 9. API Integration Tests

### TC-API-01 — All 7 Endpoints Called on Initial Load

**Story Link**: AC-04  
**Priority**: Critical  
**Type**: Functional (Manual)

**Preconditions**
- DevTools Network tab open
- SPA on Pick Folder screen

**Test Steps**
1. Click "Pick Folder" button
2. Monitor all requests to `/api/v1/viewer/`

**Expected Results**
- 4 calls on initial load: `workflow-meta`, `hierarchy`, `summary`, `recent-activity?limit=25`
- The remaining 3 endpoints (`file/{key}`, `history/{key}`, `features/{key}/tasks`) are NOT called until user navigates to an entity
- All 4 initial requests return 200 OK with valid JSON
- No duplicate calls on a single load

**Status**: Not Run

---

### TC-API-02 — file/{key} Called on Entity Navigation

**Story Link**: Spec §3  
**Priority**: High  
**Type**: Functional (Manual)

**Preconditions**
- SPA in Dashboard state

**Test Steps**
1. Monitor Network tab
2. Click a task node in sidebar

**Expected Results**
- Exactly 1 call to `GET /api/v1/viewer/file/{key}` with the task's key
- Exactly 1 call to `GET /api/v1/viewer/history/{key}` with the task's key
- Both calls return valid JSON

**Status**: Not Run

---

### TC-API-03 — features/{key}/tasks Called on Feature Progress Click

**Story Link**: Spec §3, endpoint #7  
**Priority**: Medium  
**Type**: Functional (Manual)

**Preconditions**
- Feature Progress section visible in Dashboard

**Test Steps**
1. Monitor Network tab
2. Click a feature key in Feature Progress section

**Expected Results**
- `GET /api/v1/viewer/features/{key}/tasks` request fired (if SPA uses this endpoint for drill-down)
- OR: SPA uses `treeData` from cached hierarchy (no additional call needed) — either behavior is acceptable per spec §3 open question
- Feature entity view opens correctly

**Status**: Not Run

---

### TC-API-04 — Hierarchy Load Failure Stays on Pick Folder

**Story Link**: Spec §3 Error Handling  
**Priority**: High  
**Type**: Functional (Manual)

**Preconditions**
- SPA on Pick Folder screen
- Can simulate server error (stop the API server or intercept with a proxy)

**Test Steps**
1. Stop/break the API server so `/api/v1/viewer/hierarchy` returns 500 or network error
2. Click "Pick Folder"
3. Observe UI behavior

**Expected Results**
- SPA remains on Pick Folder screen (does NOT transition to Dashboard)
- Error toast or inline error message displayed
- No JavaScript crash (no uncaught exceptions in console)
- "Pick Folder" button remains clickable for retry

**Status**: Not Run

---

### TC-API-05 — Missing Optional Fields Tolerated

**Story Link**: AC-20, Spec §3 API Contract Robustness  
**Priority**: High  
**Type**: Functional (Manual)

**Test Data**: Manipulate API response or use test fixture with:
- Feature without `docs[]` field
- Feature without `tasks[]` field  
- Bug response without `severity_counts`
- Status without `phase`

**Test Steps**
1. Load SPA with data missing optional fields (use browser DevTools to intercept response, or mock server)
2. Observe Dashboard and sidebar

**Expected Results**
- Feature without `docs[]`: no doc nodes rendered for that feature — no crash
- Feature without `tasks[]`: feature still renders in sidebar and progress section — no crash
- Bug without `severity_counts`: bug card still renders with count — no crash
- Status without `phase`: status badge still renders with correct color — no crash
- No JavaScript errors in console

**Status**: Not Run

---

### TC-API-06 — Empty Hierarchy Response

**Story Link**: AC-20  
**Priority**: Medium  
**Type**: Functional (Manual)

**Test Data**: API returns `[]` (empty array) from `/hierarchy`

**Test Steps**
1. Intercept or mock `/hierarchy` to return `[]`
2. Click "Pick Folder"

**Expected Results**
- SPA still transitions to Dashboard (empty array is success, not failure)
- Sidebar shows empty Hierarchy section (or "No entities found" message)
- Dashboard sections render empty states gracefully
- No crash

**Status**: Not Run

---

### TC-API-07 — Summary/Recent-Activity Failure Shows Inline Error

**Story Link**: Spec §3 Error Handling  
**Priority**: Medium  
**Type**: Functional (Manual)

**Test Data**: API returns 500 from `/summary` only

**Test Steps**
1. Mock `/summary` to return 500
2. Load SPA (hierarchy succeeds)
3. Observe dashboard

**Expected Results**
- Dashboard loads (hierarchy succeeded)
- Status Breakdown section shows inline error (e.g., "Failed to load status summary")
- Other dashboard sections (Feature Progress from cached hierarchy, Stale Entities) still render
- No full page crash

**Status**: Not Run

---

### TC-API-08 — Marked.js CDN Failure Fallback

**Story Link**: AC-21  
**Priority**: Medium  
**Type**: Functional (Manual)

**Preconditions**
- Can simulate CDN failure (block `cdn.jsdelivr.net` in browser network settings or DevTools)

**Test Steps**
1. Block `cdn.jsdelivr.net` in DevTools (Settings > Network conditions > Block patterns)
2. Load SPA
3. Navigate to an entity with markdown content

**Expected Results**
- Markdown content rendered as preformatted text (`<pre>` block or similar) instead of HTML
- Warning banner displayed (e.g., "Markdown rendering unavailable — CDN blocked")
- No JavaScript crash
- Content is still readable (raw markdown text visible)

**Status**: Not Run

---

## 10. Terminal Status / Stale Entity Filter Tests

### TC-STALE-01 — Non-Terminal Entity > 7 Days = Stale

**Story Link**: AC-19  
**Priority**: High  
**Type**: Functional (Manual)

**Test Data**: Entity with `status: in_progress` and `updated_at: 8+ days ago`

**Test Steps**
1. Ensure test data entity exists (or mock)
2. Load Dashboard
3. Observe Stale Entities section

**Expected Results**
- Entity appears in Stale Entities list
- Row shows: key (clickable) | `in_progress` badge (yellow) | title | "8 days ago" (or appropriate age text)

**Status**: Not Run

---

### TC-STALE-02 — Terminal Status Entities Excluded from Stale

**Story Link**: AC-19  
**Priority**: High  
**Type**: Functional (Manual)

**Test Data**: Entities in EACH terminal status with `updated_at: 30+ days ago`:
`done`, `cancelled`, `closed`, `wont_fix`, `verified`, `rejected`, `promoted`, `approved`

**Test Steps**
1. Ensure entities exist in each terminal status with old updated_at dates
2. Load Dashboard
3. Check Stale Entities section for each

**Expected Results**
- NONE of the terminal-status entities appear in Stale Entities list, regardless of how old their `updated_at` is
- `isTerminalStatus()` function returns `true` for all 8 listed statuses
- Verify in browser console: `isTerminalStatus('done')` → `true`, etc.

**Status**: Not Run

---

### TC-STALE-03 — Entity Updated Within 7 Days Not Stale

**Story Link**: AC-19  
**Priority**: Medium  
**Type**: Functional (Manual)

**Test Data**: Entity with non-terminal status and `updated_at: 3 days ago`

**Test Steps**
1. Verify test entity exists
2. Load Dashboard
3. Check Stale Entities section

**Expected Results**
- Entity does NOT appear in Stale Entities list (3 days < 7-day threshold)
- Threshold is strictly > 7 days, not >= 7 days

**Status**: Not Run

---

### TC-STALE-04 — Stale Entities Sorted Oldest First

**Story Link**: AC-19  
**Priority**: Low  
**Type**: Functional (Manual)

**Test Data**: At least 3 non-terminal entities with different old `updated_at` dates

**Test Steps**
1. Load Dashboard with multiple stale entities
2. Observe order of rows in Stale Entities section

**Expected Results**
- Rows sorted by staleness descending (oldest `updated_at` first)
- Entity with `updated_at: 30 days ago` appears before entity with `updated_at: 10 days ago`

**Status**: Not Run

---

## 11. Automated Smoke Tests (Go)

### TC-SMOKE-01 — Go Test: ViewerHTML Non-Empty and Contains Required Markers

**Story Link**: AC-01, AC-02, AC-22  
**Priority**: Critical  
**Type**: Automated (Go)

**Test Location**: `internal/viewer/assets_test.go` — `TestViewerHTMLEmbedded`

**Test Code** (reference implementation):
```go
func TestViewerHTMLEmbedded(t *testing.T) {
    if len(viewer.ViewerHTML) == 0 {
        t.Fatal("ViewerHTML is empty — go:embed failed")
    }
    content := string(viewer.ViewerHTML)
    required := []string{
        "<!DOCTYPE html>",
        "<script>",
        "STATUS_COLORS",
        "getStatusColor",
        "renderDashboard",
        "renderSidebar",
        "renderEntityView",
        "renderPickFolder",
        "api/v1/viewer",
    }
    for _, marker := range required {
        if !strings.Contains(content, marker) {
            t.Errorf("viewer.html missing required marker: %q", marker)
        }
    }
}
```

**Expected Results**: All assertions pass, no `t.Fatal` or `t.Error` calls fire.

**Run Command**: `go test ./internal/viewer/...`

**Status**: Not Run

---

### TC-SMOKE-02 — Go Test: ViewerHTML Contains Required CSS Classes and IDs

**Story Link**: AC-02, AC-22  
**Priority**: Critical  
**Type**: Automated (Go)

**Test Location**: `internal/viewer/assets_test.go` — `TestViewerHTMLContainsRequiredClasses`

**Test Code** (reference implementation):
```go
func TestViewerHTMLContainsRequiredClasses(t *testing.T) {
    content := string(viewer.ViewerHTML)
    required := []string{
        `id="sidebar"`,
        `id="content"`,
        `id="header"`,
        "pick-folder",           // Pick Folder screen section
        "status-dot",            // Status indicator dots
        "status-badge",          // Status badge pills
        "tree-node",             // Sidebar tree nodes
        "markdown-body",         // Markdown content container
        "history-drawer",        // History/transitions panel
    }
    for _, marker := range required {
        if !strings.Contains(content, marker) {
            t.Errorf("viewer.html missing required CSS class or ID: %q", marker)
        }
    }
}
```

**Expected Results**: All assertions pass.

**Status**: Not Run

---

### TC-SMOKE-03 — Go Test: ViewerHTML Is Valid UTF-8 and Not Truncated

**Story Link**: AC-01, AC-22  
**Priority**: High  
**Type**: Automated (Go)

**Test Location**: `internal/viewer/assets_test.go` — `TestViewerHTMLIsComplete`

**Test Code** (reference implementation):
```go
func TestViewerHTMLIsComplete(t *testing.T) {
    content := string(viewer.ViewerHTML)
    // Must be valid UTF-8
    if !utf8.ValidString(content) {
        t.Fatal("viewer.html is not valid UTF-8")
    }
    // Must have closing tags — not truncated
    if !strings.Contains(content, "</html>") {
        t.Fatal("viewer.html is missing </html> closing tag — file may be truncated")
    }
    if !strings.Contains(content, "</script>") {
        t.Fatal("viewer.html is missing </script> closing tag — file may be truncated")
    }
    // Minimum size check (should be ~2300 LOC ≈ 50KB+ unminified)
    if len(content) < 30000 {
        t.Errorf("viewer.html is suspiciously small (%d bytes) — expected at least 30000 bytes", len(content))
    }
}
```

**Expected Results**: File is valid UTF-8, has closing tags, and exceeds 30KB.

**Status**: Not Run

---

## 12. Exploratory Testing Charters

These sessions are timeboxed (45–60 minutes each) and follow real-user workflows.

### Charter 1: Cross-View Navigation Consistency
**Explore**: Click every entity key found in dashboard sections (Status Breakdown, Feature Progress, Active Transitions, Stale Entities) to discover navigation inconsistencies or missed key-click handlers.

**Risk to discover**: Keys that are NOT clickable despite spec requiring them to be; navigation that doesn't scroll tree into view; broken ancestor expansion.

---

### Charter 2: Sidebar with Deep Hierarchy
**Explore**: Use a project with 5+ epics, each with 5+ features, each with 10+ tasks. Expand all features and scroll the sidebar.

**Risk to discover**: Performance degradation with large trees; scroll position lost after selection; expand/collapse state corruption; layout overflow.

---

### Charter 3: Status Color Completeness
**Explore**: Use project data that exercises every status in the `STATUS_COLORS` table. Look for uncolored (fallback #666666) badges in the UI.

**Risk to discover**: Status names from real project data that don't match the hardcoded table (e.g., `in_decomposition`, `in_development`, `ready_for_development` — shark workflow statuses NOT in the color table).

---

### Charter 4: Error Resilience
**Explore**: Using DevTools, intercept API responses and return 500 or malformed JSON for each of the 7 endpoints one at a time. Observe what breaks.

**Risk to discover**: Uncaught promise rejections; partial render failures that blank the entire UI; missing error boundaries.

---

### Charter 5: Rapid Navigation
**Explore**: Click entity keys rapidly in dashboard, then immediately click different nodes in sidebar, then press Escape multiple times. Look for race conditions.

**Risk to discover**: Stale API responses rendering for wrong entity; double-loading indicators; `selectedKey` set to wrong value; overlapping render states.

---

## 13. Test Execution Order

Recommended execution order per implementation task:

| Phase | Tasks Complete | Run Tests |
|---|---|---|
| Task 1 done | App Shell + CSS + Colors | TC-VIS-01, TC-VIS-02, TC-SMOKE-02 |
| Task 2 done | API Client + Pick Folder Load | TC-NAV-01, TC-API-01, TC-API-04 |
| Task 3 done | Sidebar Tree | TC-VIS-12, TC-VIS-13, TC-TREE-01..06 |
| Task 4 done | Dashboard Breakdown + Progress | TC-VIS-03, TC-VIS-04, TC-VIS-05, TC-COLOR-01..21 |
| Task 5 done | Dashboard Activity + Stale | TC-VIS-06, TC-VIS-07, TC-STALE-01..04, TC-NAV-03 |
| Task 6 done | Entity View + Markdown | TC-VIS-08, TC-VIS-09, TC-VIS-09b, TC-NAV-02, TC-API-02 |
| Task 7 done (full SPA) | History Drawer + Navigation | TC-VIS-10, TC-VIS-11, TC-NAV-04..08, TC-KB-01..02, TC-API-03..08, TC-SMOKE-01..03 |
| All tasks done | Full test run | ALL tests, Exploratory Charters 1–5 |

---

## 14. Quality Gates

The feature MUST NOT advance to `ready_for_task_generation` unless:

- [ ] All Critical-priority test cases: PASS
- [ ] All High-priority test cases: PASS
- [ ] Zero browser console JavaScript errors on any happy-path flow
- [ ] `go test ./internal/viewer/...` passes (TC-SMOKE-01, TC-SMOKE-02, TC-SMOKE-03)
- [ ] All 22 Acceptance Criteria verified as satisfied
- [ ] Status colors verified for at minimum the 6 most common statuses: `draft`, `todo`, `in_progress`, `done`, `blocked`, `cancelled` (TC-COLOR-01 through TC-COLOR-07)
- [ ] Exploratory Charter 3 (status color completeness) executed — any discovered unlabeled statuses documented and addressed

Medium-priority test cases that fail are tracked as defects but do not block advancement if a documented workaround exists.

---

*End of Test Plan*
