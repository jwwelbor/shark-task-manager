---
feature_key: E27-F13
doc_type: test-plan
status: draft
---

# Test Plan - E27-F13: Sprint Mode Dashboard, Tree, and Planning Surface

**Spec source:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F13-sprint-mode-dashboard-tree-and-planning-surface/spec.md`  
**Feature PRD:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F13-sprint-mode-dashboard-tree-and-planning-surface/feature.md`  
**Research report:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F13-sprint-mode-dashboard-tree-and-planning-surface/RESEARCH-REPORT.md`  
**Epic UAT plan:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/uat-plan.md`

This test plan is feature-level. There are no implementation tasks yet, so the coverage is traced
directly to `spec.md` rather than to task specs.

---

## Spec Drift Analysis

### Drift findings

- No drift found between `feature.md`, `RESEARCH-REPORT.md`, and `spec.md`.
- The spec keeps the feature read-first, keeps the existing embedded viewer architecture, and
  reuses the sprint service and analytics stack identified in research.
- Assumption carried into test design: Sprint mode v1 remains primarily observational. Any write
  actions remain explicit and visually subordinate.

### Traceability matrix

| Feature spec requirement | Test case(s) | Covered? | Notes |
| --- | --- | --- | --- |
| REQ-F-001 Sprint mode is a first-class top-level viewer mode | TC-001 | Yes | Mode switch + default Overview |
| REQ-F-002 Overview is the read-first command board | TC-002 | Yes | Health, blockers, readiness, capacity, changes |
| REQ-F-003 Plan view supports backlog shaping without hiding state | TC-003 | Yes | Filters, selection, capacity visibility |
| REQ-F-004 Report view surfaces burndown, velocity, and trend data | TC-004 | Yes | Report rendering and empty states |
| REQ-F-005 Sprint tree is integrated into the existing left rail | TC-005 | Yes | Sprint tree plus hierarchy coexist |
| REQ-F-006 Detail drawer preserves entity-level drill-in | TC-006 | Yes | Metadata, history, assignment history, related work |
| REQ-F-007 Sprint mode stays read-first and minimizes accidental mutation | TC-007 | Yes | No implicit mutation on load or navigation |
| REQ-NF-001 Preserve embedded single-file viewer architecture | TC-008 | Yes | No framework/build step regression |
| REQ-NF-002 Keep the UI responsive within the existing viewer model | TC-011 | Yes | Mode switch/render responsiveness |
| REQ-NF-003 Accessibility and keyboard parity | TC-009 | Yes | Keyboard navigation and focus behavior |
| REQ-NF-004 Local-only security posture is unchanged | TC-010 | Yes | Local origins accepted, external origins rejected |

---

## Acceptance Criteria Review

### Ambiguity findings

- None requiring spec changes.
- The only deliberate scope decision is that Sprint mode v1 is read-first. That is explicit in
  the spec and is treated as the default assumption in the test plan.

### Missing coverage

- None. Every spec requirement has at least one dedicated test case.

---

## ISTQB Technique Application

| AC | Technique(s) applied | Test cases generated | Rationale |
| --- | --- | --- | --- |
| REQ-F-001 | State transition | TC-001 | Top-level mode switch has a clear lifecycle |
| REQ-F-002 | Equivalence partitioning + boundary value analysis | TC-002 | Overview data has present/absent/empty partitions |
| REQ-F-003 | Decision table | TC-003 | Filters combine status, agent, and sprint assignment |
| REQ-F-004 | Equivalence partitioning + boundary value analysis | TC-004 | Report data has empty / partial / populated partitions |
| REQ-F-005 | State transition + equivalence partitioning | TC-005 | Tree groups and buckets vary by sprint state |
| REQ-F-006 | Contract surface enumeration | TC-006 | Drawer consumes multiple related data surfaces |
| REQ-F-007 | Attack-class enumeration | TC-007 | Prevent implicit mutation and accidental write paths |
| REQ-NF-001 | Contract surface enumeration | TC-008 | Embedded asset contract and no-build boundary |
| REQ-NF-002 | Boundary value analysis | TC-011 | Render timing and redundant fetch count boundaries |
| REQ-NF-003 | State transition | TC-009 | Keyboard-driven navigation and focus changes |
| REQ-NF-004 | Attack-class enumeration | TC-010 | Local-only, origin-restricted access model |

---

## ISO 25010 Coverage Matrix

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| REQ-F-001 | Yes | N/A | Yes | Yes | N/A | N/A | Yes | N/A |
| REQ-F-002 | Yes | N/A | Yes | Yes | Yes | N/A | N/A | N/A |
| REQ-F-003 | Yes | N/A | Yes | Yes | Yes | N/A | Yes | N/A |
| REQ-F-004 | Yes | Yes | Yes | Yes | Yes | N/A | N/A | N/A |
| REQ-F-005 | Yes | N/A | Yes | Yes | N/A | N/A | Yes | N/A |
| REQ-F-006 | Yes | N/A | Yes | Yes | Yes | N/A | N/A | N/A |
| REQ-F-007 | Yes | N/A | N/A | Yes | Yes | Yes | Yes | N/A |
| REQ-NF-001 | N/A | N/A | Yes | N/A | N/A | N/A | Yes | Yes |
| REQ-NF-002 | N/A | Yes | Yes | Yes | Yes | N/A | N/A | N/A |
| REQ-NF-003 | Yes | N/A | Yes | Yes | Yes | N/A | N/A | N/A |
| REQ-NF-004 | N/A | N/A | Yes | N/A | Yes | Yes | N/A | N/A |

Notes:

- `N/A` is intentional where a characteristic does not materially apply to the AC.
- Performance is primarily relevant for overview switching and report loading, not for static
  rendering helpers.
- Security is most relevant for local-origin enforcement and mutation avoidance.

---

## Observability Design

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
| --- | --- | --- | --- | --- | --- |
| Sprint overview data assembly | Existing service spans from `SprintService.GetSprintReadiness` and `GetSprintCapacity` | Existing service logging conventions | Existing OTel spans in sprint service methods | N/A | TC-002 verifies the overview payload renders from the composed data bundle |
| Sprint report data assembly | Existing service spans from `SprintAnalyticsService.GetBurndown` and `GetVelocity` | Existing service logging conventions | Existing OTel spans in analytics service methods | N/A | TC-004 verifies burndown and velocity output is present |
| Top-level mode switching | Internal-only - no dedicated runtime observability planned for v1 | N/A | N/A | N/A | TC-001 verifies the DOM/state transition directly |
| Plan filtering and selection | Internal-only - no dedicated runtime observability planned for v1 | N/A | N/A | N/A | TC-003 verifies filtered candidate lists and explicit staging |
| Local-origin rejection | HTTP response header / status inspection | Existing viewer handler logs | Handler/request trace if already wired | N/A | TC-010 verifies external origins are rejected |

---

## Caller-Path Contracts

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
| --- | --- | --- | --- | --- |
| TC-001 | `switchTopLevelMode('sprint')` from the existing header click handler | `fetchSprintOverview` / viewer API client | Do not mock the mode router or the Sprint overview renderer | Catches a click handler that flips the mode label but never loads Sprint data |
| TC-002 | `renderSprintOverview(payload)` called from the Sprint mode render path | Fetch boundary to the viewer Sprint overview payload | Do not mock the overview sections individually | Catches a renderer that drops blockers, readiness, or capacity cards |
| TC-003 | `renderSprintPlan(payload, filters)` and `applySprintPlanFilters(filters)` | Viewer API payload / filtered candidate data | Do not mock the selection model or the filter router | Catches a Plan view that ignores one of the filter dimensions or mutates implicitly |
| TC-004 | `renderSprintReport(payload)` from the Sprint Report route | Fetch boundary to analytics-backed report data | Do not mock the chart and summary renderers separately | Catches a report view that omits burndown, velocity, or trend summary data |
| TC-005 | `renderSprintTree(treeData)` and `toggleSprintTreeNode(key)` | Existing hierarchy fetch / sprint-scope payload | Do not mock the left-rail tree layout or the hierarchy data loader | Catches a tree that renders only one sprint group or hides the entity hierarchy |
| TC-006 | `selectSprintItem(key)` feeding the existing detail-drawer render path | Viewer detail payload / history fetch boundary | Do not mock the drawer render helpers or the jump-back navigation | Catches drawer updates that fail to keep entity history and related work visible |
| TC-007 | `renderSprintPlan` event wiring and explicit action handlers | Browser event layer only | Do not mock away the action buttons or form controls | Catches an implementation that performs write-side actions on load or on passive selection |
| TC-008 | Embedded viewer asset load path and server route registration | File embed / HTTP route boundary | Do not mock the asset embed or server startup path | Catches a regression that adds a second frontend build step or breaks the embedded SPA contract |
| TC-009 | Keyboard handler path for mode, subview, and tree selection | DOM event boundary | Do not mock the focusable elements themselves | Catches a UI that works with clicks but not with Enter/Space or focus traversal |
| TC-010 | Viewer handler CORS and localhost origin path | HTTP handler boundary | Do not mock the CORS middleware or origin check | Catches a server that accepts remote origins or broadens the security surface |
| TC-011 | `renderSprintOverview` and `renderSprintReport` load path under realistic data volume | Real service boundary with mocked repositories only | Do not mock the service methods being validated | Catches a mode switch that triggers redundant calls or stalls on oversized payloads |

---

## Acceptance Test Cases

### TC-001 - Sprint mode entry and default Overview

**Feature Requirement:** REQ-F-001  
**Technique Applied:** State transition  
**ISO 25010 Characteristic(s):** Functional Suitability, Usability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** `switchTopLevelMode('sprint')` from the existing header click handler
- **Lowest allowed mock seam:** `fetchSprintOverview` / viewer API client
- **Forbidden mocks:** do not mock the mode router or the Sprint overview renderer
- **Counter-factual:** catches a click handler that flips the mode label but never loads Sprint data

**Preconditions:**
- Viewer is open on the existing dashboard or entity surface.

**Input:**
- User selects `Sprint` from the top-level mode switch.

**Expected Output:**
- The viewer enters Sprint mode.
- `Overview` is the active Sprint subview.
- The previously selected Sprint subview is preserved when the user returns to Sprint mode later in the session.

**Edge Cases:**
- Returning from Entity view to Sprint mode restores the last Sprint subview.
- An invalid restored subview is coerced to `Overview`.

**Negative Cases:**
- Sprint mode entry must not trigger a full-page reload.
- Sprint mode entry must not mutate project state.

### TC-002 - Overview renders sprint health signals

**Feature Requirement:** REQ-F-002  
**Technique Applied:** Equivalence partitioning + boundary value analysis  
**ISO 25010 Characteristic(s):** Functional Suitability, Usability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `renderSprintOverview(payload)` called from the Sprint mode render path
- **Lowest allowed mock seam:** fetch boundary to the viewer Sprint overview payload
- **Forbidden mocks:** do not mock the overview sections individually
- **Counter-factual:** catches a renderer that drops blockers, readiness, or capacity cards

**Preconditions:**
- Viewer has a valid current sprint selected.

**Input:**
- Overview payload with sprint name, date range, readiness score, capacity utilization,
  blocked count, WIP count, and recent changes.

**Expected Output:**
- The current sprint identity is displayed prominently.
- Readiness, capacity, blocked count, and WIP count are visible on first render.
- Recent changes or an empty-state message render without breaking layout.

**Edge Cases:**
- Zero blocked items.
- Zero recent changes.
- No capacity pressure / all capacity rows empty.

**Negative Cases:**
- Overview must not hide the sprint identity behind a second navigation step.
- Overview must not fail when one of the summary signals is empty.

### TC-003 - Plan filters and bulk staging

**Feature Requirement:** REQ-F-003  
**Technique Applied:** Decision table  
**ISO 25010 Characteristic(s):** Functional Suitability, Usability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `renderSprintPlan(payload, filters)` and `applySprintPlanFilters(filters)`
- **Lowest allowed mock seam:** viewer API payload / filtered candidate data
- **Forbidden mocks:** do not mock the selection model or the filter router
- **Counter-factual:** catches a Plan view that ignores one of the filter dimensions or mutates implicitly

**Preconditions:**
- Viewer has candidate sprint items and capacity data available.

**Input:**
- A payload with candidate work plus filter combinations for status, agent type, and sprint
  assignment.

**Expected Output:**
- The Plan view shows the filtered candidate list that matches the active filter combination.
- Bulk selection works on the displayed items.
- Capacity remains visible while planning.

**Edge Cases:**
- No filters selected.
- One filter selected.
- Conflicting filters that yield an empty result.

**Negative Cases:**
- Opening the Plan view must not add or remove sprint assignments automatically.

### TC-004 - Report view renders burndown and velocity

**Feature Requirement:** REQ-F-004  
**Technique Applied:** Equivalence partitioning + boundary value analysis  
**ISO 25010 Characteristic(s):** Functional Suitability, Performance Efficiency, Usability

**Caller-Path Contract:**
- **Entrypoint:** `renderSprintReport(payload)` from the Sprint Report route
- **Lowest allowed mock seam:** fetch boundary to analytics-backed report data
- **Forbidden mocks:** do not mock the chart and summary renderers separately
- **Counter-factual:** catches a report view that omits burndown, velocity, or trend summary data

**Preconditions:**
- Current sprint exists; analytics payload is available or intentionally empty.

**Input:**
- Report payload with burndown points, velocity history, and trend summary data.

**Expected Output:**
- Burndown and velocity sections render.
- Trend or scope-change summary renders when present.
- Empty or partial data renders safely without a broken screen.

**Edge Cases:**
- No historical sprint data.
- Planning sprint with no burndown series.
- One-point history.

**Negative Cases:**
- Report view must not block the Overview or Plan subviews.

### TC-005 - Sprint tree coexists with the entity hierarchy

**Feature Requirement:** REQ-F-005  
**Technique Applied:** State transition + equivalence partitioning  
**ISO 25010 Characteristic(s):** Functional Suitability, Usability, Maintainability

**Caller-Path Contract:**
- **Entrypoint:** `renderSprintTree(treeData)` and `toggleSprintTreeNode(key)`
- **Lowest allowed mock seam:** existing hierarchy fetch / sprint-scope payload
- **Forbidden mocks:** do not mock the left-rail tree layout or the hierarchy data loader
- **Counter-factual:** catches a tree that renders only one sprint group or hides the entity hierarchy

**Preconditions:**
- Viewer has sprint tree data plus the existing hierarchy tree available.

**Input:**
- Active sprint, upcoming sprint, completed archive, and per-sprint bucket data.

**Expected Output:**
- The left rail shows a Sprint tree section.
- Active, upcoming, archive, and bucketed sprint scopes are visible.
- The entity hierarchy remains available in the same rail.

**Edge Cases:**
- No upcoming sprint.
- No archive entries.
- Empty bucket counts.

**Negative Cases:**
- Sprint tree rendering must not remove the existing hierarchy tree.

### TC-006 - Detail drawer preserves entity drill-in

**Feature Requirement:** REQ-F-006  
**Technique Applied:** Contract surface enumeration  
**ISO 25010 Characteristic(s):** Functional Suitability, Usability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `selectSprintItem(key)` feeding the existing detail-drawer render path
- **Lowest allowed mock seam:** viewer detail payload / history fetch boundary
- **Forbidden mocks:** do not mock the drawer render helpers or the jump-back navigation
- **Counter-factual:** catches drawer updates that fail to keep entity history and related work visible

**Preconditions:**
- A sprint item or entity is visible in Sprint mode.

**Input:**
- User selects a sprint item from the tree or Plan view.

**Expected Output:**
- The detail drawer shows entity metadata, current status, history, assignment history, and
  related work.
- A clear jump-back path to the entity view remains available.

**Edge Cases:**
- Missing related work.
- Missing assignment history.
- Entity not found.

**Negative Cases:**
- Selecting an item must not navigate away from Sprint mode.

### TC-007 - Sprint mode remains read-first

**Feature Requirement:** REQ-F-007  
**Technique Applied:** Attack-class enumeration  
**ISO 25010 Characteristic(s):** Functional Suitability, Security, Maintainability

**Caller-Path Contract:**
- **Entrypoint:** `renderSprintPlan` event wiring and explicit action handlers
- **Lowest allowed mock seam:** browser event layer only
- **Forbidden mocks:** do not mock away the action buttons or form controls
- **Counter-factual:** catches an implementation that performs write-side actions on load or on
  passive selection

**Preconditions:**
- Sprint mode is open.

**Input:**
- User opens Sprint mode, inspects items, switches between subviews, and does not activate any
  explicit action button.

**Expected Output:**
- No write request is issued implicitly.
- Read-only inspection remains available without mutation.
- Write-capable controls, if present, are explicit and visually subordinate.

**Edge Cases:**
- Repeated refreshes.
- Switching between subviews without selecting items.

**Negative Cases:**
- No implicit add/remove/update action may fire on load or on selection alone.

### TC-008 - Embedded single-file viewer architecture is preserved

**Feature Requirement:** REQ-NF-001  
**Technique Applied:** Contract surface enumeration  
**ISO 25010 Characteristic(s):** Maintainability, Portability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** embedded viewer asset load path and server route registration
- **Lowest allowed mock seam:** file embed / HTTP route boundary
- **Forbidden mocks:** do not mock the asset embed or server startup path
- **Counter-factual:** catches a regression that adds a second frontend build step or breaks the
  embedded SPA contract

**Preconditions:**
- Repository state before and after the change can be compared.

**Input:**
- Build and run the viewer after the Sprint-mode change.

**Expected Output:**
- The viewer still loads from the embedded `viewer.html` SPA.
- No frontend bundler or framework dependency is introduced.

**Edge Cases:**
- Clean checkout.
- Offline environment.

**Negative Cases:**
- No extra build artifact is allowed to become required for launch.

### TC-009 - Keyboard parity and accessibility

**Feature Requirement:** REQ-NF-003  
**Technique Applied:** State transition  
**ISO 25010 Characteristic(s):** Usability, Accessibility, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** keyboard handler path for mode, subview, and tree selection
- **Lowest allowed mock seam:** DOM event boundary
- **Forbidden mocks:** do not mock the focusable elements themselves
- **Counter-factual:** catches a UI that works with clicks but not with Enter/Space or focus
  traversal

**Preconditions:**
- Viewer is open in a browser.

**Input:**
- Keyboard navigation across mode buttons, Sprint subviews, and tree items.

**Expected Output:**
- Focus moves to the interactive controls.
- Enter/Space activates the same behavior as mouse clicks.
- State changes are reachable without a mouse.

**Edge Cases:**
- Rapid focus moves between mode and tree controls.
- Returning focus after switching subviews.

**Negative Cases:**
- Sprint mode controls must not be keyboard-inert.

### TC-010 - Local-only security posture is unchanged

**Feature Requirement:** REQ-NF-004  
**Technique Applied:** Attack-class enumeration  
**ISO 25010 Characteristic(s):** Security, Reliability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** viewer HTTP handlers and CORS middleware
- **Lowest allowed mock seam:** handler boundary
- **Forbidden mocks:** do not mock the CORS middleware or origin check
- **Counter-factual:** catches a server that accepts remote origins or broadens the security
  surface

**Preconditions:**
- Viewer server is running locally.

**Input:**
- Requests from localhost origins and from a non-local origin.

**Expected Output:**
- Localhost origins continue to work.
- External origins are rejected or denied access.
- No new auth surface is introduced.

**Edge Cases:**
- `http://localhost:<any-port>` origins.
- `http://127.0.0.1:<any-port>` origins.

**Negative Cases:**
- A remote origin must not receive a permissive CORS response.

### TC-011 - Overview/report responsiveness stays within the existing viewer model

**Feature Requirement:** REQ-NF-002  
**Technique Applied:** Boundary value analysis  
**ISO 25010 Characteristic(s):** Performance Efficiency, Usability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `renderSprintOverview` and `renderSprintReport` load path under realistic data volume
- **Lowest allowed mock seam:** real service boundary with mocked repositories only
- **Forbidden mocks:** do not mock the service methods being validated
- **Counter-factual:** catches a mode switch that triggers redundant calls or stalls on oversized
  payloads

**Preconditions:**
- Representative sprint data exists.

**Input:**
- Small, medium, and larger sprint payloads.

**Expected Output:**
- Sprint view switches remain responsive.
- The viewer does not spam duplicate fetches on a single navigation action.

**Edge Cases:**
- Empty sprint.
- Sprint with many items.

**Negative Cases:**
- A mode switch must not degrade into a full-page reload.

---

## Integration Scenarios

- Sprint mode and the existing viewer shell: ensure the new top-level mode fits into the same
  header, left rail, and detail-drawer layout used by the current dashboard.
- Sprint services and viewer aggregation: verify the Sprint overview and report payloads are
  composed from `SprintService` and `SprintAnalyticsService`, not from ad hoc browser logic.
- Sprint tree and entity hierarchy: verify both can coexist in the left rail without the tree
  becoming mutually exclusive.
- Detail drawer and history/navigation: verify sprint item selection still leads to inspectable
  entity detail, history, and a jump-back path.

Reference UAT areas from `uat-plan.md` that this feature contributes to:

- Area B - Dashboard Overview
- Area C - Sidebar Navigation
- Area D - Entity View
- Area F - Database Backend Support
- Area H - Security Boundaries

Sprint mode also introduces a new future UAT slice that is not yet named in the epic plan.

---

## Test Infrastructure

### Existing patterns to follow

- `internal/viewer/assets_test.go` - string-presence and embedded-asset regression tests for the
  single-file SPA.
- `internal/api/viewer/handler_test.go` - handler tests using the mock-service pattern.
- `internal/services/viewer_service_test.go` - service tests with function-field mock repositories.
- `internal/services/sprint_service_test.go` - sprint planning and readiness service tests.
- `internal/services/sprint_analytics_service_test.go` - sprint reporting service tests.
- `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F03-single-file-spa-ide-style-dark-dashboard-interface/RESEARCH-REPORT.md`
  - Prior viewer research for the same embedded SPA architecture.

### New helpers likely needed

- A mock Sprint viewer service for handler tests.
- A small set of Sprint-mode viewer payload builders for service tests.
- A browser smoke fixture for a project with one active sprint, one upcoming sprint, and one
  completed sprint.

### Test approach notes

- Go service and handler tests should mock at the service boundary or the repository boundary,
  never the browser layer.
- Frontend behavior should be covered by viewer asset tests and manual browser smoke tests.
- Sprint data and reporting should reuse the existing sprint service and analytics tests as the
  source of truth for expected values.

