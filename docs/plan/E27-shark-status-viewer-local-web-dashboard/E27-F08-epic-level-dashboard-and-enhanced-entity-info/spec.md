---
feature_key: E27-F08-epic-level-dashboard-and-enhanced-entity-info
epic_key: E27
title: "Spec: Epic-Level Dashboard and Enhanced Entity Info"
type: combined-spec
---

# Spec: Epic-Level Dashboard and Enhanced Entity Info

**Feature Key**: E27-F08
**Parent Epic**: [E27 PRD](../epic.md) · [E27 Architecture](../architecture.md)
**Feature description**: [feature.md](feature.md)

> This spec is **incremental** on top of E27's existing web viewer (single-file SPA at `internal/viewer/assets/viewer.html`). See feature.md for business context and user stories. See epic PRD for overall viewer architecture.

---

## 1. Requirements

All requirements trace to user stories and functional requirements in [feature.md §"Requirements"](feature.md).

### 1.1 Functional Requirements

#### REQ-F-001 — Epic-scoped Dashboard tab

Clicking an epic node in the tree opens the entity view with a new **Dashboard** tab active by default. The tab renders the same four dashboard sections as the project-level dashboard (entity charts, status overview, feature progress, recent activity, stale entities) but scoped to the selected epic.

**Traceability**: feature.md REQ-F-001, Story 1.

**Testable acceptance criteria**:

- **AC-001.1**: When the user clicks an epic node in the tree, the entity view renders a `Dashboard` tab and this tab is active (`tabState === 'dashboard'`).
- **AC-001.2**: The Dashboard tab button appears **before** (to the left of) the `Spec` tab button in the toggle bar.
- **AC-001.3**: The Dashboard tab renders `Entity Charts`, `Status Overview`, `Feature Progress`, `Recent Activity`, and `Stale Entities` sections — same section titles as the project-level dashboard.
- **AC-001.4**: `Entity Charts` and `Status Overview` for an epic reflect only counts for that epic's features + their tasks (not the whole project).
- **AC-001.5**: `Feature Progress` lists only features belonging to the selected epic.
- **AC-001.6**: `Recent Activity` lists only transitions where the entity key equals the epic key, or belongs to a feature/task under that epic.
- **AC-001.7**: `Stale Entities` lists only the epic, its features, and their tasks (no other epics).
- **AC-001.8**: When a feature or task node is clicked, **no** `Dashboard` tab is rendered (the toggle bar contains only `Spec`, `Transitions`, and optionally `Files`/`Edit`).
- **AC-001.9**: Clicking a feature/task key inside the epic dashboard's `Feature Progress`, `Recent Activity`, or `Stale Entities` tables navigates to that entity (reuses existing `data-navigate-key` delegation).

#### REQ-F-002 — Rename "Info" tab to "Spec"

The `Info` toggle-bar button is renamed to `Spec` on every entity view. Internal identifiers (`id="ev-tab-info"`, `entityViewTab === 'info'`) may remain unchanged to minimise diff.

**Traceability**: feature.md REQ-F-002, Story 2.

**Testable acceptance criteria**:

- **AC-002.1**: For every entity type (epic, feature, task, bug, change card) the tab button text reads `Spec` (not `Info`).
- **AC-002.2**: All existing tab behaviour is preserved: clicking `Spec` renders the entity's markdown via `renderMarkdownPane()`, the `✎ Edit` button still appears when a `file_path` / `linked_content_path` exists, and `Transitions` / `Files` tabs behave identically.
- **AC-002.3**: Navigation history (back/forward) and the URL hash state produce the same spec view that the old "Info" tab produced.

#### REQ-F-003 — Enhanced entity properties panel

The properties panel shown above the toggle bar in the entity view is expanded to display all fields returned by `shark get {entity}` (as seen in `./bin/shark get <key> --json`). Orchestrator action details go in a collapsible `<details>` block that is **collapsed by default**.

**Traceability**: feature.md REQ-F-003, Story 3.

**Testable acceptance criteria**:

- **AC-003.1**: For an **epic**, the panel displays (when non-empty): `Key`, `Status`, `Type`, `Created`, `Updated`, `Priority`, `Business Value`, `Progress`, `Phase`, `Approval Backlog Count`, `Feature Status Rollup` (inline count chips), `File Path`. See §1.4 for the epic field set.
- **AC-003.2**: For a **feature**, the panel displays (when non-empty): `Key`, `Status`, `Type`, `Parent (epic)`, `Created`, `Updated`, `Progress`, `Execution Order`, `Phase`, `Phase Description`, `Display Mode`, `Workflow Position` (e.g. `in_specification (7/15)`), `File Path`.
- **AC-003.3**: For a **task**, the panel displays (when non-empty): `Key`, `Status`, `Type`, `Parent (feature)`, `Agent`, `Priority`, `Execution Order`, `Created`, `Updated`, `Completed`, `Rejection Count`, `Verification Status`, `Tests Passed`, `Dependencies` (comma-separated keys), `Blocked By`, `Blocks`, `File Path`.
- **AC-003.4**: For a **bug**, the panel displays (when non-empty): `Key`, `Status`, `Type`, `Severity`, `Priority`, `Assignee`, `Created`, `Updated`, `File Path`, plus any `linked_entity_key` the API surfaces.
- **AC-003.5**: For a **change card**, the panel displays (when non-empty): `Key`, `Status`, `Type`, `Priority`, `Assignee`, `Created`, `Updated`, `File Path`, plus any `linked_entity_key` the API surfaces.
- **AC-003.6**: When `orchestrator_action` is present, it renders as a collapsible `<details>` block with `<summary>Orchestrator Action</summary>` and shows `action`, `agent_type`, `model`, `skills[]`, `instruction` (in a `<pre>` or similar preformatted block). The block is collapsed by default (no `open` attribute).
- **AC-003.7**: Fields that are `null`, `undefined`, or empty strings are omitted (same rule as the current panel). No blank rows are rendered.
- **AC-003.8**: The panel continues to render `Status` using `getStatusColor()` + `getContrastColor()` (existing status-badge styling).
- **AC-003.9**: The copy-to-clipboard button (`copyBtn`) still appears for `File Path` and `Content Path`.
- **AC-003.10**: No existing field that the current panel already displays is removed. (Regression gate.)

### 1.2 Non-functional Requirements

#### REQ-NF-001 — No new backend endpoints

All data for the epic-scoped dashboard comes from existing cached client state (`summaryData`, `treeData`, `hierarchyData`, `recentActivity`). No new backend route is added.

**Acceptance criteria**:

- **AC-NF-001.1**: `grep -n "/api/v1/viewer" internal/viewer/assets/viewer.html` shows the same set of endpoints before and after this feature.
- **AC-NF-001.2**: The epic dashboard renders within the same time envelope as the project dashboard (subjective; no new network requests on tab switch).

#### REQ-NF-002 — Properties panel data already in memory

The enriched properties panel must be satisfied from data already loaded into `treeData` / `hierarchyData` by existing API calls. If a field required by §1.4 is **not** on the cached entity object, it is simply omitted — the feature must not introduce a new per-entity `shark get` fetch on entity-view open.

**Acceptance criteria**:

- **AC-NF-002.1**: Opening an entity view issues **no** new `GET /api/v1/…` requests beyond those already issued by the current implementation (file fetch, history fetch — unchanged).
- **AC-NF-002.2**: If a field (e.g. `orchestrator_action`) is not present on the cached object, the panel omits it silently; it does not render "undefined" or an empty row.

### 1.3 Acceptance criteria (end-to-end scenarios)

Reuse the four scenarios in [feature.md §"Acceptance Criteria"](feature.md). In addition:

**Scenario 5 — Orchestrator action collapsed by default**

- **Given** any entity whose API payload includes `orchestrator_action`
- **When** the entity view is rendered
- **Then** an `Orchestrator Action` disclosure appears in the properties panel
- **And** it is closed (no `open` attribute) on first render
- **And** clicking the summary reveals the `action`, `agent_type`, `model`, `skills`, and `instruction` fields.

**Scenario 6 — Regression: existing tabs behave identically**

- **Given** any entity
- **When** the user switches between `Spec` (formerly Info) and `Transitions` (and `Files` where applicable)
- **Then** the behaviour matches the current production viewer exactly (markdown rendering, edit button enablement, history table).

### 1.4 Entity field mapping (authoritative)

The properties panel is built from fields that the **existing** API responses already attach to each entity. The cached objects on the client are:

| Entity type | Cached source | Container field path |
| --- | --- | --- |
| epic        | `hierarchyData.epics[i]` | attached to the top-level tree (`treeData[i]`) |
| feature     | `hierarchyData.epics[i].features[j]` | `treeData[i].features[j]` |
| task        | `hierarchyData.epics[i].features[j].tasks[k]` | `treeData[i].features[j].tasks[k]` |
| bug         | `hierarchyData.bugs[i]` |
| change_card | `hierarchyData.change_cards[i]` |

The `renderPropertiesPanel` function MUST operate only on whatever fields are present on the cached object. Fields listed below that are NOT on the hierarchy payload are simply not shown — this is acceptable per AC-NF-002.2 and avoids an extra network round-trip.

**Epic fields**: `key`, `status`, `created_at`, `updated_at`, `priority`, `business_value`, `progress_pct` (if exposed on hierarchy) or derived from features, `phase`, `approval_backlog_count`, `feature_status_rollup`, `file_path`, `linked_content_path`, `orchestrator_action`.

**Feature fields**: `key`, `status`, `created_at`, `updated_at`, `progress_pct`, `execution_order`, `phase`, `phase_description`, `display_mode`, `workflow_position.current_index` + `.statuses.length`, `file_path`, `linked_content_path`, `orchestrator_action`, `parent` (epic key, injected by `findEntityByKey`).

**Task fields**: `key`, `status`, `agent_type`, `priority`, `execution_order`, `created_at`, `updated_at`, `completed_at`, `rejection_count`, `verification_status`, `tests_passed`, `dependencies`, `blocked_by`, `blocks`, `file_path`, `linked_content_path`, `orchestrator_action`, `parent` (feature key, injected).

**Bug / change-card fields**: `key`, `status`, `severity` (bug only), `priority`, `assignee`, `created_at`, `updated_at`, `file_path`, `linked_content_path`, `orchestrator_action`.

### 1.5 Out of scope

See feature.md §"Out of Scope". Reiterating explicitly:

1. No Dashboard tab for features or tasks in this feature.
2. No new backend API endpoints. No new per-entity `shark get` fetch on entity-view open.
3. No dashboard-widget configuration or drag-and-drop.
4. The internal `entityViewTab` state variable retains its string values (`'info'`, `'transitions'`, `'files'`) — rename is visual only. Tab-state refactor is out of scope.
5. The existing project-level dashboard is unchanged.

---

## 2. Architecture

All changes are confined to a single file: `internal/viewer/assets/viewer.html`. No Go code, no new API endpoints, no new assets. Template JS is embedded in that file via `internal/viewer/assets/assets.go` (`go:embed`).

> Follows the E27 single-file SPA pattern established in E27-F03 (see epic architecture doc §"Single-file SPA").

### 2.1 Component changes

#### File modified

- **`internal/viewer/assets/viewer.html`** (sole source change)

#### Tests modified / added

- **`internal/viewer/assets/assets_test.go`** — extend the existing "string presence" tests (which assert that known markers appear in the embedded HTML) with new markers for `Spec`, `Dashboard`, and `Orchestrator Action`. Follows the existing pattern in that file.

#### No Go source changes

- `internal/api/viewer/*` — unchanged.
- `internal/viewer/server/*` — unchanged.
- CLI commands — unchanged.

### 2.2 JS design (inside viewer.html)

#### 2.2.1 Tab state refactor (minimal)

Current state variable:

```js
let entityViewTab = 'info';   // 'info' | 'transitions' | 'files'
```

Add one new value:

```js
let entityViewTab = 'info';   // 'info' | 'transitions' | 'files' | 'dashboard'
```

No rename is required — the `'info'` value continues to represent the Spec pane. Only the **button label** changes from `Info` to `Spec` (AC-002.1).

**Default-tab rule in `navigateToEntity(key)`** (currently at line 1900):

```js
// BEFORE:
entityViewTab = 'info';   // reset to Info tab on every navigation

// AFTER:
const navEntity = findEntityByKey(key);
entityViewTab = (navEntity && navEntity.type === 'epic') ? 'dashboard' : 'info';
```

This is the single control point that makes Dashboard the default for epics (AC-001.1). All other code paths that set `entityViewTab = 'info'` (lines 2756, 3178, 3251 — "return from history / transition table back to spec") remain correct.

#### 2.2.2 `renderEntityView()` — toggle bar + dashboard pane

Current function (around line 2638) builds a toggle bar containing `Info`, `Transitions`, (optional) `Files`, (optional) `Edit`.

**Changes**:

1. Compute `isEpic = entity?.type === 'epic'`.
2. If `isEpic`, prepend a `Dashboard` button:
   ```js
   const dashActive = entityViewTab === 'dashboard' ? ' active' : '';
   const dashBtnHtml = isEpic
     ? `<button class="toggle-btn${dashActive}" id="ev-tab-dashboard">Dashboard</button>`
     : '';
   ```
3. Change the `Info` button's text to `Spec` (AC-002.1); keep `id="ev-tab-info"`:
   ```js
   `<button class="toggle-btn${infoActive}" id="ev-tab-info">Spec</button>`
   ```
4. In the rendered `toggle-bar` template, place `dashBtnHtml` before the Spec button:
   ```js
   <div class="toggle-bar">
     ${dashBtnHtml}
     <button class="toggle-btn${infoActive}"  id="ev-tab-info">Spec</button>
     <button class="toggle-btn${transActive}" id="ev-tab-transitions">Transitions</button>
     ${filesBtnHtml}
     ${editBtnHtml}
   </div>
   ```
5. Wire the new button:
   ```js
   const dashTabBtn = document.getElementById('ev-tab-dashboard');
   if (dashTabBtn) {
     dashTabBtn.addEventListener('click', () => {
       if (entityViewTab !== 'dashboard') {
         entityViewTab = 'dashboard';
         pushNavState();
         renderEntityView();
       }
     });
   }
   ```
6. Extend the pane-render switch:
   ```js
   if (entityViewTab === 'info') {
     renderMarkdownPane(selectedKey, paneEl);
   } else if (entityViewTab === 'dashboard' && isEpic) {
     renderEpicDashboardPane(entity.key, paneEl);
   } else if (entityViewTab === 'files') {
     renderFolderFilesPane(filesDirPath, paneEl);
   } else {
     // Transitions (unchanged)
     ...
   }
   ```
7. Guard: if `entityViewTab === 'dashboard'` but the entity is not an epic (e.g. history state restored from a different entity), fall back to `'info'` and re-render.

#### 2.2.3 New function: `renderEpicDashboardPane(epicKey, paneEl)`

This is the only genuinely new JS function. It mirrors the body of `renderDashboard()` (line 2289) but passes an `epicKey` filter into each section renderer.

```js
/**
 * Renders the epic-scoped dashboard inside an entity-view pane.
 * Mirrors renderDashboard() but passes epicKey to each section.
 * @param {string} epicKey
 * @param {HTMLElement} paneEl
 */
function renderEpicDashboardPane(epicKey, paneEl) {
  const chartsHtml = renderEntityCharts(epicKey);
  paneEl.innerHTML = `
    ${chartsHtml ? `<div class="dashboard-section">
      <div class="dashboard-section-title">Entity Charts</div>
      ${chartsHtml}
    </div>` : ''}
    <div class="dashboard-section">
      <div class="dashboard-section-title">Status Overview</div>
      ${renderStatusBreakdown(epicKey)}
    </div>
    <div class="dashboard-section">
      <div class="dashboard-section-title">Feature Progress</div>
      ${renderFeatureProgress(epicKey)}
    </div>
    <div class="dashboard-section">
      <div class="dashboard-section-title">Recent Activity</div>
      ${renderActiveTransitions(epicKey)}
    </div>
    <div class="dashboard-section">
      <div class="dashboard-section-title">Stale Entities</div>
      ${renderStaleEntities(epicKey)}
    </div>
  `;
  // Reuse existing delegation for [data-navigate-key]
  paneEl.querySelectorAll('[data-navigate-key]').forEach(el => {
    el.addEventListener('click', () => {
      const k = el.getAttribute('data-navigate-key');
      if (k) navigateToEntity(k);
    });
  });
}
```

#### 2.2.4 Section-renderer extensions (optional `epicKey` parameter)

Each of the five existing renderers grows an **optional** `epicKey` parameter. When omitted → project-wide behaviour (today). When supplied → filter to that epic. This keeps `renderDashboard()` (project view) unchanged at its call sites.

**a) `renderStatusBreakdown(epicKey)`** — line 1913

Project-level path: use `summaryData.{epics,features,tasks,…}.by_status` (unchanged).

Epic-scoped path: derive counts from `treeData` filtered to `epicKey`.

```js
function renderStatusBreakdown(epicKey) {
  if (epicKey) {
    // Build local counts from treeData
    const epic = (treeData || []).find(e => e.key === epicKey);
    if (!epic) return `<div class="content-placeholder">Epic not found</div>`;
    const featureCounts = {};   // status -> count
    const taskCounts    = {};
    const epicCounts    = { [epic.status || 'unknown']: 1 };
    for (const f of (epic.features || [])) {
      const s = f.status || 'unknown';
      featureCounts[s] = (featureCounts[s] || 0) + 1;
      for (const t of (f.tasks || [])) {
        const ts = t.status || 'unknown';
        taskCounts[ts] = (taskCounts[ts] || 0) + 1;
      }
    }
    // Convert to the same shape as summaryData.*.by_status:
    //   [{status, count, color, phase, progress_weight}, ...]
    const toByStatus = obj => Object.entries(obj).map(([status, count]) => ({
      status, count, color: getStatusColor(status),
    }));
    // Render using the same card/badge template as the project-level path.
    ...
  }
  // else: existing code path, unchanged.
}
```

All card/badge HTML generation remains identical to today — we just swap the data source. The `phase`/`progress_weight` fields used for the "X active" computation come from the client-side helper `isTerminalStatus(status)` (already in use) rather than from summaryData; this is a slight behavioural difference but yields the same visual result for all statuses defined in `.sharkworkflow*.json`.

**b) `renderFeatureProgress(epicKey)`** — line 1972

```js
function renderFeatureProgress(epicKey) {
  const epics = epicKey
    ? (treeData || []).filter(e => e.key === epicKey)
    : (treeData || []);
  // …rest of function uses `epics` instead of `treeData` directly
}
```

**c) `renderActiveTransitions(epicKey)`** — line 2013

```js
function renderActiveTransitions(epicKey) {
  let rows = recentActivity || [];
  if (epicKey) {
    // Build membership set from treeData: epic key + its features + its tasks
    const epic = (treeData || []).find(e => e.key === epicKey);
    const keys = new Set();
    if (epic) {
      keys.add(epic.key);
      for (const f of (epic.features || [])) {
        keys.add(f.key);
        for (const t of (f.tasks || [])) keys.add(t.key);
      }
    }
    rows = rows.filter(r => keys.has(r.key));
  }
  rows = rows.slice(0, 25);
  // …rest unchanged (iterates `rows`)
}
```

**d) `renderStaleEntities(epicKey)`** — line 2075

Restrict the outer loop to the selected epic when `epicKey` is provided:

```js
const epics = epicKey
  ? (treeData || []).filter(e => e.key === epicKey)
  : (treeData || []);
for (const epic of epics) { … }
```

**e) `renderEntityCharts(epicKey)`** — line 2202

Project-level path (unchanged) builds six cards from `summaryData.*`.

Epic-scoped path reuses `toByStatus(...)` computed in (a) for features and tasks, plus the single-item epic card `{ [epic.status]: 1 }`. Bugs/ideas/change-cards cards are **omitted** in epic scope (they are not owned by a specific epic in the current data model).

#### 2.2.5 Properties panel rewrite — `renderPropertiesPanel(entity)`

Rewrite the function (line 2399) as a dispatch-by-type + shared-base pattern.

```js
function renderPropertiesPanel(entity) {
  if (!entity) return '';
  const rows = [];

  // Shared base fields (in this order, when non-null):
  pushRow(rows, 'Key',      monoCell(entity.key));
  pushRow(rows, 'Status',   statusBadgeCell(entity.status));
  pushRow(rows, 'Type',     entity.type ? escapeHtml(entity.type) : null);
  if (entity.parent) pushRow(rows, 'Parent', monoCell(entity.parent));

  // Type-specific fields:
  switch (entity.type) {
    case 'epic':        appendEpicFields(rows, entity);        break;
    case 'feature':     appendFeatureFields(rows, entity);     break;
    case 'task':        appendTaskFields(rows, entity);        break;
    case 'bug':         appendBugFields(rows, entity);         break;
    case 'change_card': appendChangeCardFields(rows, entity);  break;
  }

  // Shared trailing fields (timestamps, paths):
  if (entity.created_at) pushRow(rows, 'Created', escapeHtml(formatDate(entity.created_at)));
  if (entity.updated_at) pushRow(rows, 'Updated', escapeHtml(formatDate(entity.updated_at)));
  if (entity.file_path)  pushRow(rows, 'File Path',
    `<span style="word-break:break-all">${escapeHtml(entity.file_path)}</span>${copyBtn(entity.file_path)}`);
  if (entity.linked_content_path) pushRow(rows, 'Content Path',
    `<span style="word-break:break-all">${escapeHtml(entity.linked_content_path)}</span>${copyBtn(entity.linked_content_path)}`);

  // Trailing orchestrator-action disclosure (collapsible, outside the grid):
  const orchHtml = renderOrchestratorDetails(entity.orchestrator_action);

  const grid = rows.length
    ? `<div class="props-grid">${rows.join('')}</div>`
    : '';
  return grid + orchHtml;
}

function pushRow(rows, label, valueHtml) {
  if (valueHtml === null || valueHtml === undefined || valueHtml === '') return;
  rows.push(`<div class="props-label">${escapeHtml(label)}</div><div class="props-value">${valueHtml}</div>`);
}

function monoCell(v)        { return v ? `<span class="mono">${escapeHtml(v)}</span>` : null; }
function statusBadgeCell(s) {
  if (!s) return null;
  const c = getStatusColor(s);
  return `<span class="status-badge" style="background:${escapeHtml(c)};color:${getContrastColor(c)}">${escapeHtml(s.replace(/_/g,' '))}</span>`;
}
```

Type-specific appenders surface the per-entity fields from §1.4. Example for epic:

```js
function appendEpicFields(rows, e) {
  if (e.priority != null)          pushRow(rows, 'Priority',       String(e.priority));
  if (e.business_value != null)    pushRow(rows, 'Business Value', String(e.business_value));
  if (e.progress_pct != null)      pushRow(rows, 'Progress',       e.progress_pct + '%');
  if (e.phase)                     pushRow(rows, 'Phase',          escapeHtml(e.phase));
  if (e.approval_backlog_count)    pushRow(rows, 'Approval Backlog', String(e.approval_backlog_count));
  if (e.feature_status_rollup) {
    const chips = Object.entries(e.feature_status_rollup).map(([s, n]) => {
      const c = getStatusColor(s);
      return `<span class="status-badge status-badge-sm" style="background:${escapeHtml(c)};color:${getContrastColor(c)}">${escapeHtml(s.replace(/_/g,' '))} ${n}</span>`;
    }).join(' ');
    pushRow(rows, 'Feature Rollup', chips);
  }
}
```

Feature, task, bug, and change-card appenders follow the same guard-and-push pattern using the field lists in §1.4. Dependency arrays render as comma-separated `<span class="mono">`s.

**Orchestrator-action disclosure** (AC-003.6, Scenario 5):

```js
function renderOrchestratorDetails(oa) {
  if (!oa || typeof oa !== 'object') return '';
  const rows = [];
  if (oa.action)     rows.push(rowHtml('Action',  escapeHtml(oa.action)));
  if (oa.agent_type) rows.push(rowHtml('Agent',   escapeHtml(oa.agent_type)));
  if (oa.model)      rows.push(rowHtml('Model',   escapeHtml(oa.model)));
  if (Array.isArray(oa.skills) && oa.skills.length)
    rows.push(rowHtml('Skills', oa.skills.map(escapeHtml).join(', ')));
  const instr = oa.instruction
    ? `<pre class="orch-instruction" style="white-space:pre-wrap;margin:8px 0 0">${escapeHtml(oa.instruction)}</pre>`
    : '';
  return `
    <details class="orch-action">
      <summary>Orchestrator Action</summary>
      <div class="props-grid" style="margin-top:8px">${rows.join('')}</div>
      ${instr}
    </details>`;
}
function rowHtml(l, v) { return `<div class="props-label">${escapeHtml(l)}</div><div class="props-value">${v}</div>`; }
```

No `open` attribute is emitted — `<details>` defaults to closed (AC-003.6).

### 2.3 CSS additions

Add to the existing `<style>` block (located above the script; new rules, no changes to existing selectors):

```css
.orch-action {
  margin-top: 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 6px 10px;
  background: var(--bg-2);
}
.orch-action > summary {
  cursor: pointer;
  color: var(--fg-dim);
  font-size: 12px;
  user-select: none;
  padding: 2px 0;
}
.orch-action[open] > summary { color: var(--fg); }
.orch-action .props-grid   { margin-top: 8px; }
.orch-instruction          { font-family: var(--font-mono,monospace); font-size: 12px; background: var(--bg-3); padding: 8px; border-radius: var(--radius); }
```

No other selectors change. `.toggle-bar`, `.toggle-btn`, `.dashboard-section`, `.breakdown-grid`, `.progress-list`, `.activity-table`, `.props-grid` are reused as-is.

### 2.4 Data-model changes

**None.** No database schema change, no migration, no new columns, no new API DTO fields. All fields displayed in §1.4 are already produced by existing Go handlers at `internal/api/viewer/handler.go` (`Hierarchy`, `Summary`, `RecentActivity`).

### 2.5 API / interface contracts

**No new endpoints.**

Existing endpoints consumed (unchanged):

- `GET /api/v1/viewer/summary` — used for project dashboard, unchanged.
- `GET /api/v1/viewer/hierarchy` — used for tree + entity lookup, unchanged.
- `GET /api/v1/viewer/recent-activity` — used for Recent Activity, unchanged.
- `GET /api/v1/file/{key}` — used for `Spec` pane, unchanged.
- `GET /api/v1/history/{key}` — used for `Transitions` pane, unchanged.
- `PUT /api/v1/edit/file` — unchanged.

If a future task discovers that a field from §1.4 is not actually present on the hierarchy payload and is considered essential, the required extension is:

- Add the field to `services.HierarchyResponse` + the handler's DTO projection in `internal/api/viewer/handler.go` (`Hierarchy`), so that it arrives on the existing payload.
- **Do NOT** introduce a new per-entity GET call at entity-view open time (violates AC-NF-002.1).

Task generation should include an audit step that lists which §1.4 fields are / are not already on the hierarchy payload and decides per-field: include now, silently skip (missing data), or extend hierarchy DTO.

### 2.6 Key technical decisions

| Decision | Rationale |
| --- | --- |
| Keep the `'info'` tab-state value; only change the button label. | Minimal diff, no history/hash invalidation, satisfies AC-002.1 without touching state persistence. |
| Epic dashboard is a pane **inside** `renderEntityView()`, not a separate `appState`. | Matches existing tab-switching UX (Spec/Transitions/Files); preserves back-button, toolbar, properties panel. Avoids a fourth top-level app state. |
| Add an optional `epicKey` parameter to each existing section renderer. | Maximises reuse — one set of HTML/CSS, two call sites (project scope, epic scope). Zero duplication of card/badge templates. Satisfies AC-001.3's "same visual components" requirement. |
| Client-side filtering by epic. | No new backend endpoint (REQ-NF-001). The hierarchy payload already contains every entity under an epic; `recentActivity` already contains `key` per record. Filtering is O(entities-in-epic), trivially fast. |
| Properties panel is driven off cached hierarchy data, not a per-entity `shark get` fetch. | REQ-NF-002. The hierarchy payload is already loaded at app start. A per-entity fetch on every click would double request volume and add latency on every tree click. |
| Orchestrator-action block uses native `<details>` rather than a custom collapsible. | Native HTML5 element, accessible by default, no JS wiring needed, matches the project's minimal-JS aesthetic. |
| Bugs / ideas / change cards are omitted from the epic-scoped `Entity Charts`. | Current data model does not associate them with an epic. Including zero-everywhere cards would be visually noisy. |

### 2.7 Integration with existing code (file + function references)

All line numbers are against the current `main` (viewer.html is 3411 lines).

| Change | Location | Exact element |
| --- | --- | --- |
| Default epic tab → dashboard | `internal/viewer/assets/viewer.html` line 1900 | `entityViewTab = 'info'` inside `navigateToEntity()` |
| Add `renderEpicDashboardPane` | New function, inserted after `renderDashboard()` (after line 2326) | — |
| Add `epicKey` param — status breakdown | line 1913 — `renderStatusBreakdown()` | signature + early filtered branch |
| Add `epicKey` param — feature progress | line 1972 — `renderFeatureProgress()` | wrap `treeData` iteration |
| Add `epicKey` param — recent activity | line 2013 — `renderActiveTransitions()` | filter `recentActivity` |
| Add `epicKey` param — stale entities | line 2075 — `renderStaleEntities()` | outer-loop filter |
| Add `epicKey` param — entity charts | line 2202 — `renderEntityCharts()` | derive counts from `treeData` sub-tree |
| Dashboard tab button + wiring | line 2638 — `renderEntityView()` | add `dashBtnHtml`, wire `ev-tab-dashboard` click handler, extend pane-render switch |
| Info → Spec button label | line 2668 | change inner text only |
| Properties panel rewrite | line 2399 — `renderPropertiesPanel()` | replace body per §2.2.5 |
| Orchestrator CSS | `<style>` block (above script) | append §2.3 rules |
| Asset test markers | `internal/viewer/assets/assets_test.go` | add string-presence assertions for `>Dashboard<`, `>Spec<`, `Orchestrator Action` |

### 2.8 Risks & mitigations

| Risk | Mitigation |
| --- | --- |
| Browser history/hash state for an epic might restore `entityViewTab='dashboard'` on a non-epic entity after navigation. | In `renderEntityView()`, guard: if `entityViewTab === 'dashboard'` and `!isEpic`, coerce to `'info'` before rendering. |
| Large epic (many features+tasks) renders all charts/tables client-side. | Existing renderers already handle the whole project; one epic is strictly a subset. No new performance risk. |
| A hierarchy-level field listed in §1.4 turns out to be missing from the actual DTO. | Task generation includes an explicit audit (§2.5). Panel silently omits missing fields (AC-NF-002.2). |
| Regression on existing `Info` tab behaviour. | AC-002.2 plus Scenario 6 mandate full behavioural parity; single-label change minimises blast radius. |

---

## 3. Exit gate

- ✅ Every requirement has at least one AC, and every AC is testable against either (a) the rendered DOM, (b) the value of `entityViewTab`, or (c) the network panel.
- ✅ Every architecture decision references existing viewer code (line numbers above) or an existing doc (CLAUDE.md, E27 architecture).
- ✅ File paths listed for every change. Only `internal/viewer/assets/viewer.html` and `internal/viewer/assets/assets_test.go` are modified.
- ✅ No TBDs remain.

---

*Last updated*: 2026-04-13
