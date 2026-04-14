---
feature_key: E27-F08
doc_type: test-plan
status: draft
spec: ./spec.md
epic_uat_plan: ../uat-plan.md
---

# E27-F08 Test Plan: Epic-Level Dashboard and Enhanced Entity Info

## Overview

This plan covers all automated tests required before E27-F08 ships. Every acceptance criterion in `spec.md §1` has at least one named test case. Tests are assigned to `internal/viewer/assets_test.go` — the existing "string presence" pattern used by all prior E27 features — which verifies that required JS functions, CSS markers, and HTML structures are present in the embedded `viewer.html`.

Tests are **not** end-to-end browser tests — those belong to the epic UAT plan (`../uat-plan.md`). This plan covers Go string-presence tests in `assets_test.go` only. There are no new Go backend files introduced by E27-F08 (the spec explicitly calls out zero changes to `internal/api/viewer/`, `internal/viewer/server/`, or CLI commands — only `viewer.html` and `assets_test.go` change).

---

## 1. AC Test Matrix

### REQ-F-001 / AC-001 — Epic-scoped Dashboard tab

#### TC-F08-001: Dashboard tab renders for epics (AC-001.1)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLEpicDashboardTab` |
| File | `internal/viewer/assets_test.go` |
| Assertion | `viewer.html` contains `ev-tab-dashboard` (the dashboard button ID) and `entityViewTab === 'dashboard'` logic wiring |
| Markers to assert | `"ev-tab-dashboard"`, `"entityViewTab"`, `"renderEpicDashboardPane"` |
| AC covered | AC-001.1 — Dashboard tab renders and is the default for epics |

**Edge case:** The guard `if (entityViewTab === 'dashboard' && !isEpic)` must be present to handle history-state restoration for non-epic entities. Assert marker `"entityViewTab === 'dashboard'"`.

#### TC-F08-002: Dashboard tab appears before Spec tab (AC-001.2)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLDashboardTabPosition` |
| File | `internal/viewer/assets_test.go` |
| Approach | In `viewer.html` source, the string `ev-tab-dashboard` must appear **before** `ev-tab-info` (positional check using `strings.Index`). |
| AC covered | AC-001.2 — Dashboard button precedes Spec button in toggle bar |

**Edge case:** Verify the marker order is not reversed by accident during implementation.

#### TC-F08-003: Dashboard sections present (AC-001.3)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLEpicDashboardSections` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"renderEpicDashboardPane"`, `"Entity Charts"`, `"Status Overview"`, `"Feature Progress"`, `"Recent Activity"`, `"Stale Entities"` |
| AC covered | AC-001.3 — All five dashboard sections present in epic pane |

**Edge case:** Each section title is double-quoted as an exact string in JS template literals; verify the title strings match exactly (case-sensitive).

#### TC-F08-004: Epic-scoped data filtering in section renderers (AC-001.4, AC-001.5, AC-001.6, AC-001.7)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLEpicScopedFiltering` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"epicKey"` parameter present in `renderStatusBreakdown`, `renderFeatureProgress`, `renderActiveTransitions`, `renderStaleEntities`, `renderEntityCharts` |
| AC covered | AC-001.4 through AC-001.7 — all renderers accept and apply epicKey filter |

Implementation check: the string `"function renderStatusBreakdown(epicKey)"` or `"renderStatusBreakdown(epicKey"` must appear; same pattern for the four other renderers.

#### TC-F08-005: No Dashboard tab for features and tasks (AC-001.8)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLNoDashboardForNonEpic` |
| File | `internal/viewer/assets_test.go` |
| Approach | Assert `"isEpic"` variable is present and guards the `dashBtnHtml` assignment. Marker: `"isEpic"` and `"dashBtnHtml"`. |
| AC covered | AC-001.8 — Dashboard tab only rendered when entity is an epic |

**Edge case:** If `isEpic` is absent, the tab would render for all entity types — this is a correctness gate.

#### TC-F08-006: data-navigate-key delegation in epic dashboard (AC-001.9)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLEpicDashboardNavigation` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"data-navigate-key"` already verified by prior E27 tests; confirm `renderEpicDashboardPane` is present (delegating reuse of `data-navigate-key` is implied by the function wiring). |
| AC covered | AC-001.9 — Click navigation from epic dashboard to feature/task entities |
| Note | `data-navigate-key` marker already asserted in `TestViewerHTMLActiveTransitionsImplementation`. This test is a belt-and-suspenders check confirming `renderEpicDashboardPane` references the same delegation. |

---

### REQ-F-002 / AC-002 — Info → Spec rename

#### TC-F08-007: Spec tab label in toggle bar (AC-002.1)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLSpecTabLabel` |
| File | `internal/viewer/assets_test.go` |
| Assertion | The string `">Spec<"` (button inner text) is present and the string `">Info<"` is **absent** |
| AC covered | AC-002.1 — Button label reads "Spec" for all entity types |

**Edge case:** The check `!strings.Contains(content, ">Info<")` must confirm the old label is fully removed. The internal ID `ev-tab-info` must still be present (visual-only rename).

#### TC-F08-008: Internal ev-tab-info ID preserved (AC-002.2)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLSpecTabInternalId` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"ev-tab-info"` still present (ID unchanged), `"renderMarkdownPane"` still present (content unchanged) |
| AC covered | AC-002.2 — Existing tab behaviour preserved |
| Note | `ev-tab-info` is already asserted by `TestViewerHTMLEntityViewToggle`. This new test confirms the rename did not remove it. |

#### TC-F08-009: Navigation hash/history state unchanged (AC-002.3)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLSpecTabHistoryState` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"pushNavState"` present (existing function), `"entityViewTab"` present |
| AC covered | AC-002.3 — Navigation history produces same Spec view as old Info tab |
| Note | The `'info'` string value is retained as the tab state; assert `"=== 'info'"` or `"'info'"` still appears in the entityViewTab logic. |

---

### REQ-F-003 / AC-003 — Enhanced entity properties panel

#### TC-F08-010: Epic-specific properties in panel (AC-003.1)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelEpicFields` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"appendEpicFields"`, `"Priority"`, `"Business Value"`, `"Feature Rollup"`, `"Approval Backlog"`, `"approval_backlog_count"` |
| AC covered | AC-003.1 — Epic panel shows priority, business value, progress, phase, approval backlog count, feature rollup |

**Edge case:** `feature_status_rollup` chips must iterate with status badge HTML; assert `"feature_status_rollup"` appears in `viewer.html`.

#### TC-F08-011: Feature-specific properties in panel (AC-003.2)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelFeatureFields` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"appendFeatureFields"`, `"Execution Order"`, `"execution_order"`, `"workflow_position"`, `"Phase Description"`, `"phase_description"`, `"Display Mode"` |
| AC covered | AC-003.2 — Feature panel shows execution order, phase, phase description, display mode, workflow position |

**Edge case:** `workflow_position.current_index` and `.statuses.length` are computed values; assert `"workflow_position"` appears in the properties panel function body.

#### TC-F08-012: Task-specific properties in panel (AC-003.3)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelTaskFields` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"appendTaskFields"`, `"rejection_count"`, `"verification_status"`, `"tests_passed"`, `"completed_at"`, `"blocked_by"`, `"Blocked By"`, `"Blocks"` |
| AC covered | AC-003.3 — Task panel shows agent, priority, execution order, completed, rejection count, verification status, tests passed, dependencies, blocked by, blocks |

**Edge case:** `dependencies` is an array; assert `"dependencies"` appears in the task fields appender and renders as comma-separated `<span class="mono">` elements.

#### TC-F08-013: Bug-specific properties in panel (AC-003.4)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelBugFields` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"appendBugFields"`, `"severity"`, `"Severity"`, `"assignee"`, `"Assignee"` |
| AC covered | AC-003.4 — Bug panel shows severity, priority, assignee |

#### TC-F08-014: Change card properties in panel (AC-003.5)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelChangeCardFields` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"appendChangeCardFields"`, `"change_card"` (entity type dispatch), `"Assignee"` |
| AC covered | AC-003.5 — Change card panel shows priority, assignee |

#### TC-F08-015: Orchestrator action collapsible disclosure (AC-003.6)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLOrchestratorActionDisclosure` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"renderOrchestratorDetails"`, `"<details"`, `"<summary>Orchestrator Action</summary>"`, `"orch-action"`, `"orch-instruction"`, `"orchestrator_action"` |
| AC covered | AC-003.6 — Orchestrator action renders as collapsible details block, collapsed by default |

**Edge case — no `open` attribute:** Assert the string `"<details open"` is absent (the block must default to closed). The presence of `"<details"` without `open` is verified by asserting `!strings.Contains(content, "<details open")` within the `orch-action` context, or by asserting only `"<details class=\"orch-action\""` is used.

#### TC-F08-016: Null/empty field omission (AC-003.7)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelNullGuard` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"pushRow"` function present, `"function pushRow"` defined, early-return guard `"if (valueHtml === null"` or `"valueHtml === undefined"` |
| AC covered | AC-003.7 — Fields that are null/undefined/empty are omitted; no blank rows rendered |

#### TC-F08-017: Status badge styling preserved (AC-003.8)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelStatusBadge` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"statusBadgeCell"`, `"getStatusColor"`, `"getContrastColor"` in properties panel context |
| AC covered | AC-003.8 — Status continues to use getStatusColor + getContrastColor |
| Note | `getStatusColor` already asserted by existing `TestViewerHTMLEntityViewPropertiesStatusBadge`. This test adds `"statusBadgeCell"` and `"getContrastColor"` as new markers. |

#### TC-F08-018: Copy-to-clipboard button preserved (AC-003.9)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelCopyButton` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"copyBtn"` function present, `"navigator.clipboard"` still present |
| AC covered | AC-003.9 — Copy button appears for File Path and Content Path |
| Note | `"copy-btn"` and `"navigator.clipboard"` are already asserted by `TestViewerHTMLPropertiesPanelFields`. This test narrows to `"copyBtn"` as a helper function name (called from the rewritten `renderPropertiesPanel`). |

#### TC-F08-019: No existing properties panel fields removed (AC-003.10)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLPropertiesPanelRegressionGate` |
| File | `internal/viewer/assets_test.go` |
| Markers | All markers from the existing `TestViewerHTMLPropertiesPanelFields` test must still pass: `"File Path"`, `"Content Path"`, `"props-grid"`, `"props-label"`, `"props-value"`, `"copy-btn"`, `"navigator.clipboard"` |
| AC covered | AC-003.10 — No existing displayed field is removed |
| Note | This test re-asserts the pre-existing markers explicitly to serve as a regression gate during the properties panel rewrite. If the rewrite accidentally removes any marker, this test catches it. |

---

### REQ-NF-001, REQ-NF-002 / AC-NF — No new backend endpoints, no new fetches

#### TC-F08-020: No new API endpoint paths introduced (AC-NF-001.1)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLNoNewAPIEndpoints` |
| File | `internal/viewer/assets_test.go` |
| Approach | Assert that the seven existing API function names (`apiGetWorkflowMeta`, `apiGetHierarchy`, `apiGetSummary`, `apiGetRecentActivity`, `apiGetFile`, `apiGetHistory`, `apiGetFeatureTasks`) still match exactly — no new `apiGet*` function added. Count occurrences if needed. |
| AC covered | AC-NF-001.1 — Endpoint set unchanged |

#### TC-F08-021: Epic dashboard uses treeData, not a new fetch (AC-NF-002.1, AC-NF-002.2)

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLEpicDashboardUsesExistingData` |
| File | `internal/viewer/assets_test.go` |
| Markers | `"renderEpicDashboardPane"` references `"treeData"` and/or `"recentActivity"` (already loaded); no new `fetch(` or `apiGet` call inside `renderEpicDashboardPane` body |
| AC covered | AC-NF-002.1 — No new per-entity GET request on entity-view open; AC-NF-002.2 — missing fields silently omitted |
| Note | A full body analysis is not feasible in a string-presence test. The test asserts `"treeData"` appears in `viewer.html` (already present) and that `renderEpicDashboardPane` is a function (its definition marker appears). The absence of `"fetch("` inside a narrowly-scoped substring is an optional stretch assertion. |

---

### Scenario 5 / AC-003.6 — Orchestrator action collapsed by default

Already covered by TC-F08-015. The `open` attribute absence check is the key edge case asserting "collapsed by default".

---

### Scenario 6 — Regression: existing tabs behave identically

#### TC-F08-022: All pre-F08 assets_test markers still pass

| Field | Value |
|---|---|
| Test function | `TestViewerHTMLF08RegressionGate` |
| File | `internal/viewer/assets_test.go` |
| Approach | Assert a curated set of the highest-risk existing markers that the F08 rewrite could accidentally break: `"renderEntityView"`, `"renderMarkdownPane"`, `"renderDashboard"`, `"ev-tab-transitions"`, `"ev-tab-info"`, `"history-table"`, `"toggle-btn"`, `"props-grid"` |
| AC covered | Scenario 6 — Spec, Transitions, Files tabs behave identically after F08 changes |

---

## 2. Additional Edge Cases

### 2a. Dashboard tab guard: non-epic entity with dashboard history state

**Risk:** If a user's browser history stores `entityViewTab='dashboard'` for an epic, then they navigate to a feature via a link, the guard must coerce the tab back to `'info'`.

**Test:** TC-F08-005 already covers this by asserting the `isEpic` variable and the guard logic marker are present.

### 2b. Empty epic (no features, no tasks)

**Risk:** If an epic has zero features, the dashboard sections must render an empty-state message, not throw a JS error.

**Test:** Assert `"No features found"` marker is still present (already in `TestViewerHTMLDashboardNullGuards`). This existing test acts as the empty-epic guard.

### 2c. Feature rollup chips — zero-count statuses

**Risk:** The spec states fields are omitted when null/empty. If `feature_status_rollup` is an empty object `{}`, `Object.entries({})` is empty so `chips` is an empty string — `pushRow` should then skip the row per AC-003.7.

**Test:** TC-F08-016 asserts the `pushRow` null guard handles empty strings. TC-F08-010 verifies the rollup rendering path exists.

### 2d. `orchestrator_action` missing on entity — no disclosure rendered

**Risk:** If `orchestrator_action` is absent or null, `renderOrchestratorDetails(undefined)` must return `''` without throwing.

**Test:** TC-F08-015 asserts the `renderOrchestratorDetails` function exists. The null guard `"if (!oa || typeof oa !== 'object')"` must be present — add this marker to TC-F08-015.

### 2e. `workflow_position` missing on feature (hierarchy payload gap)

**Risk:** Per AC-NF-002.2, if `workflow_position` is not on the cached hierarchy object, the panel must omit it silently.

**Test:** TC-F08-016's `pushRow` null guard covers this. TC-F08-011 verifies the field is attempted to be rendered; the null guard ensures it does not crash.

### 2f. Dashboard tab button order in toggle bar

**Risk:** If the template string puts `ev-tab-dashboard` after `ev-tab-info`, AC-001.2 fails silently in a pure marker test.

**Test:** TC-F08-002 uses `strings.Index` positional comparison to enforce ordering.

---

## 3. Integration Scenarios

### 3a. Epic dashboard pane rendering — cross-function integration

**What interacts:** `renderEntityView()` → `renderEpicDashboardPane(epicKey, paneEl)` → `renderEntityCharts(epicKey)`, `renderStatusBreakdown(epicKey)`, `renderFeatureProgress(epicKey)`, `renderActiveTransitions(epicKey)`, `renderStaleEntities(epicKey)`.

**What to verify:** All five section renderer signatures accept an `epicKey` parameter (TC-F08-004). `renderEpicDashboardPane` is defined (TC-F08-001, TC-F08-003). The pane switch in `renderEntityView` dispatches to `renderEpicDashboardPane` when `entityViewTab === 'dashboard'` (TC-F08-001 marker: `"renderEpicDashboardPane"`).

**Boundary:** The data handoff from `treeData` (hierarchy) to the epic-scoped renderers is client-side filtering — no new HTTP boundary. TC-F08-021 verifies this contract.

### 3b. Properties panel rewrite — dispatch-by-type integration

**What interacts:** `renderPropertiesPanel(entity)` → `appendEpicFields` / `appendFeatureFields` / `appendTaskFields` / `appendBugFields` / `appendChangeCardFields` → `pushRow` → `renderOrchestratorDetails`.

**What to verify:** All five type-specific appenders exist (TC-F08-010 through TC-F08-014). `pushRow` null guard exists (TC-F08-016). `renderOrchestratorDetails` exists and defaults to closed (TC-F08-015). No existing field removed (TC-F08-019).

**Boundary:** `renderPropertiesPanel` is called from `renderEntityView`. Verifying `renderPropertiesPanel` is still referenced from `renderEntityView` is implicit in TC-F08-022 (regression gate asserts `renderPropertiesPanel` is still present in the file).

### 3c. Spec/Info rename — integration with navigation and history

**What interacts:** Toggle bar button label → `entityViewTab` state → `pushNavState()` → URL hash → on-load hash restoration → `renderEntityView`.

**What to verify:** TC-F08-007 (label is `Spec`), TC-F08-008 (ID preserved), TC-F08-009 (history state and `pushNavState` unchanged).

### 3d. Epic UAT scenarios this feature contributes to

| UAT Scenario | How F08 contributes |
|---|---|
| D3 — Properties panel shows rich metadata | TC-F08-010 through TC-F08-019 verify all entity-type fields are rendered |
| C4 — Clicking a node opens entity view | TC-F08-001 verifies Dashboard tab opens by default for epics |
| B3 — Feature progress bars | TC-F08-004 verifies `renderFeatureProgress(epicKey)` is wired |
| B4 — Recent activity feed | TC-F08-004 verifies `renderActiveTransitions(epicKey)` is wired |
| I1 — Full loop: dashboard → entity view | TC-F08-001, TC-F08-005 verify correct tab routing per entity type |
| J2 — No console errors | Null guards TC-F08-015 (`oa` check), TC-F08-016 (`pushRow`) prevent runtime throws |

---

## 4. Test Infrastructure

### 4a. Existing patterns to follow

| Pattern | Location | Usage for F08 |
|---|---|---|
| String-presence assertions in `assets_test.go` | `internal/viewer/assets_test.go` (all existing tests) | All F08 tests use the same `strings.Contains(content, marker)` pattern |
| `viewer.ViewerHTML` embedded via `go:embed` | `internal/viewer/assets.go` | All tests read from `viewer.ViewerHTML` — no filesystem access needed |
| Negative assertions (`!strings.Contains`) | `TestViewerHTMLStaleEntitiesImplementation`, `TestViewerHTMLDocViewImplemented` | TC-F08-007 (`">Info<"` absent), TC-F08-015 (`"<details open"` absent) |
| Positional ordering check with `strings.Index` | Not yet used — new pattern for F08 | TC-F08-002 (Dashboard before Spec in toggle bar) |

### 4b. New test utility needed

**Positional ordering helper** (one-liner, inline in TC-F08-002):

```go
// In TestViewerHTMLDashboardTabPosition:
dashIdx := strings.Index(content, "ev-tab-dashboard")
infoIdx := strings.Index(content, "ev-tab-info")
if dashIdx == -1 {
    t.Fatal("viewer.html missing ev-tab-dashboard button id")
}
if infoIdx == -1 {
    t.Fatal("viewer.html missing ev-tab-info button id")
}
if dashIdx > infoIdx {
    t.Error("ev-tab-dashboard must appear before ev-tab-info in viewer.html (Dashboard tab must precede Spec tab)")
}
```

No external test utility needed. All tests in `assets_test.go` use the same `package viewer_test` and the shared `viewer.ViewerHTML` global.

### 4c. No new test files needed

All F08 tests are added to the existing `internal/viewer/assets_test.go`. This matches the architecture established by E27-F03 through E27-F07: one test file verifies all structural markers in the single-file SPA.

### 4d. Test file location

| Test file | What it tests | Notes |
|---|---|---|
| `internal/viewer/assets_test.go` | All TC-F08-001 through TC-F08-022 | Extends existing file; no new test files. Package `viewer_test`. |

---

## 5. Test Case Summary

| Test Case ID | AC(s) Covered | Test Function | Notes |
|---|---|---|---|
| TC-F08-001 | AC-001.1 | `TestViewerHTMLEpicDashboardTab` | Dashboard tab wiring for epics |
| TC-F08-002 | AC-001.2 | `TestViewerHTMLDashboardTabPosition` | Positional ordering: Dashboard before Spec |
| TC-F08-003 | AC-001.3 | `TestViewerHTMLEpicDashboardSections` | All 5 section titles present in epic pane |
| TC-F08-004 | AC-001.4–AC-001.7 | `TestViewerHTMLEpicScopedFiltering` | epicKey param in all 5 renderers |
| TC-F08-005 | AC-001.8 | `TestViewerHTMLNoDashboardForNonEpic` | isEpic guard prevents tab for features/tasks |
| TC-F08-006 | AC-001.9 | `TestViewerHTMLEpicDashboardNavigation` | data-navigate-key delegation in epic pane |
| TC-F08-007 | AC-002.1 | `TestViewerHTMLSpecTabLabel` | ">Spec<" present, ">Info<" absent |
| TC-F08-008 | AC-002.2 | `TestViewerHTMLSpecTabInternalId` | ev-tab-info ID still present |
| TC-F08-009 | AC-002.3 | `TestViewerHTMLSpecTabHistoryState` | pushNavState + 'info' state value preserved |
| TC-F08-010 | AC-003.1 | `TestViewerHTMLPropertiesPanelEpicFields` | Epic-specific panel fields |
| TC-F08-011 | AC-003.2 | `TestViewerHTMLPropertiesPanelFeatureFields` | Feature-specific panel fields |
| TC-F08-012 | AC-003.3 | `TestViewerHTMLPropertiesPanelTaskFields` | Task-specific panel fields |
| TC-F08-013 | AC-003.4 | `TestViewerHTMLPropertiesPanelBugFields` | Bug-specific panel fields |
| TC-F08-014 | AC-003.5 | `TestViewerHTMLPropertiesPanelChangeCardFields` | Change card-specific panel fields |
| TC-F08-015 | AC-003.6, Scenario 5 | `TestViewerHTMLOrchestratorActionDisclosure` | details/summary, orch-action CSS, closed by default |
| TC-F08-016 | AC-003.7 | `TestViewerHTMLPropertiesPanelNullGuard` | pushRow null guard omits blank rows |
| TC-F08-017 | AC-003.8 | `TestViewerHTMLPropertiesPanelStatusBadge` | statusBadgeCell + getContrastColor |
| TC-F08-018 | AC-003.9 | `TestViewerHTMLPropertiesPanelCopyButton` | copyBtn helper present |
| TC-F08-019 | AC-003.10 | `TestViewerHTMLPropertiesPanelRegressionGate` | No existing panel field removed |
| TC-F08-020 | AC-NF-001.1 | `TestViewerHTMLNoNewAPIEndpoints` | API function set unchanged |
| TC-F08-021 | AC-NF-002.1, AC-NF-002.2 | `TestViewerHTMLEpicDashboardUsesExistingData` | treeData reuse, no new fetches |
| TC-F08-022 | Scenario 6 | `TestViewerHTMLF08RegressionGate` | Full regression: existing markers survive |

**ACs covered:**

| AC | Test Cases |
|---|---|
| AC-001.1 | TC-F08-001 |
| AC-001.2 | TC-F08-002 |
| AC-001.3 | TC-F08-003 |
| AC-001.4, AC-001.5, AC-001.6, AC-001.7 | TC-F08-004 |
| AC-001.8 | TC-F08-005 |
| AC-001.9 | TC-F08-006 |
| AC-002.1 | TC-F08-007 |
| AC-002.2 | TC-F08-008 |
| AC-002.3 | TC-F08-009 |
| AC-003.1 | TC-F08-010 |
| AC-003.2 | TC-F08-011 |
| AC-003.3 | TC-F08-012 |
| AC-003.4 | TC-F08-013 |
| AC-003.5 | TC-F08-014 |
| AC-003.6, Scenario 5 | TC-F08-015 |
| AC-003.7 | TC-F08-016 |
| AC-003.8 | TC-F08-017 |
| AC-003.9 | TC-F08-018 |
| AC-003.10 | TC-F08-019 |
| AC-NF-001.1 | TC-F08-020 |
| AC-NF-001.2 | UAT Area B (no-network-request assertion is a browser DevTools check; not automatable in Go string tests) |
| AC-NF-002.1 | TC-F08-021 |
| AC-NF-002.2 | TC-F08-016 (pushRow null guard), TC-F08-021 |
| Scenario 6 | TC-F08-022, TC-F08-007, TC-F08-008 |

All ACs have at least one automated test. AC-NF-001.2 (no new network requests on tab switch) is a browser/network-panel check covered by UAT scenario D3/I1 and is explicitly not automatable from Go.

---

## 6. Quality Gates

Before marking E27-F08 `ready_for_task_generation`:

- [ ] Every TC listed in Section 5 has a corresponding test function defined in `assets_test.go`.
- [ ] `make test` passes with zero failures (all existing tests continue green; F08 tests are green because the string markers they assert are present in the implemented `viewer.html`).
- [ ] `make fmt && make lint` pass.
- [ ] Negative assertion TC-F08-007 (`">Info<"` absent) confirms the rename is complete throughout the file.
- [ ] Negative assertion TC-F08-015 (`"<details open"` absent inside orch-action context) confirms the disclosure defaults to collapsed.
- [ ] Positional assertion TC-F08-002 (`ev-tab-dashboard` index < `ev-tab-info` index) confirms tab ordering.
- [ ] AC-NF-001.2 (no new network requests on tab switch) is deferred to epic UAT (Areas D3, I1) — documented explicitly as UAT-only.
- [ ] No Go source files changed outside of `internal/viewer/assets_test.go` and `internal/viewer/assets/viewer.html`.
