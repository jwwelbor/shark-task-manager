---
feature_key: E27-F13-sprint-mode-dashboard-tree-and-planning-surface
epic_key: E27
title: "Spec: Sprint Mode Dashboard, Tree, and Planning Surface"
type: combined-spec
---

# Spec: Sprint Mode Dashboard, Tree, and Planning Surface

**Feature Key**: E27-F13  
**Parent Epic**: [E27 PRD](../epic.md) · [E27 Architecture](../architecture.md)  
**Feature description**: [feature.md](feature.md)  
**Research report**: [RESEARCH-REPORT.md](RESEARCH-REPORT.md)

> This spec is incremental on top of the viewer established by E27-F01 through E27-F12.
> It adds a new first-class Sprint mode to the embedded viewer, with `Overview`, `Plan`,
> and `Report` subviews, sprint-aware navigation, and a read-first planning surface.
> The implementation remains aligned to the existing single-file SPA architecture unless a
> later task proves a narrower extension is impossible.

---

## 1. Requirements

### 1.1 Functional Requirements

#### REQ-F-001 - Sprint mode is a first-class top-level viewer mode

**Trace**: feature.md seed description, recommendation doc, wireframes.

The viewer must expose a top-level `Sprint` mode alongside the existing dashboard/entity/doc
surfaces. Sprint mode is not a nested detail panel. It is a sibling view in the same viewer
shell and can be reached from the header or any equivalent primary mode switch.

Sprint mode has three subviews:

- `Overview`
- `Plan`
- `Report`

`Overview` is the default landing state for Sprint mode.

**Testable ACs**:

- [ ] The header renders an explicit Sprint mode entry that switches the app into Sprint mode.
- [ ] Sprint mode contains `Overview`, `Plan`, and `Report` subview controls.
- [ ] `Overview` is the default Sprint subview on first entry.
- [ ] Switching away from Sprint mode and back preserves the last selected Sprint subview for
      the session unless the user explicitly changes it.

#### REQ-F-002 - Overview is the read-first command board

**Trace**: feature.md seed description, recommendation doc, wireframes.

Sprint `Overview` must prioritize current-sprint observability over planning controls. It must
surface the current sprint identity, live work status, blockers, readiness, capacity pressure,
and recent changes in a single screen.

The overview must include, at minimum:

- sprint name and date range
- readiness score
- capacity utilization
- blocked count
- work-in-progress count
- recent status changes or newly blocked items

**Testable ACs**:

- [ ] The Overview view renders the sprint identity and date range prominently.
- [ ] The Overview view renders readiness and capacity signals without requiring a second
      navigation step.
- [ ] Blockers and recent changes are visible on the initial Overview render.
- [ ] The Overview view remains readable when the sprint has no blockers or no recent changes;
      empty states are shown instead of broken layout.

#### REQ-F-003 - Plan view supports backlog shaping without hiding state

**Trace**: feature.md initial scope, recommendation doc, wireframes.

Sprint `Plan` must provide a planning workspace for backlog shaping. It should support filtering
and selection of sprint-scoped work, and it must keep capacity visible so the user can judge
scope without leaving the view.

Plan view capabilities include:

- filtering by status
- filtering by agent type
- filtering by sprint assignment
- bulk selection
- explicit actions to stage work for the sprint

The first release is read-first. Planning actions are allowed, but they must remain explicit and
do not replace the dashboard as the default user entry point.

**Testable ACs**:

- [ ] Plan view renders filter controls for status, agent type, and sprint assignment.
- [ ] Plan view supports bulk selection of candidate items.
- [ ] Capacity visibility remains present while Plan view is open.
- [ ] No planning action is triggered implicitly by just opening the view.

#### REQ-F-004 - Report view surfaces burndown, velocity, and trend data

**Trace**: feature.md initial scope, recommendation doc, wireframes.

Sprint `Report` must expose historical and trend-oriented reporting for the current sprint and,
where useful, previous completed sprints. The view must cover:

- burndown
- velocity
- trend / forecast movement
- carryover or scope-change summary where available

The report view is lower priority than live work visibility and should never replace the Overview
as the first thing users see in Sprint mode.

**Testable ACs**:

- [ ] Report view renders burndown and velocity sections.
- [ ] Report view can show trend or scope-change summary data when available.
- [ ] Empty reporting data is handled gracefully and does not break the page.

#### REQ-F-005 - Sprint tree is integrated into the existing left rail

**Trace**: feature.md initial scope, recommendation doc, wireframes.

The existing left rail must gain a Sprint tree section that keeps sprint context visible next to
the entity hierarchy. The Sprint tree should surface:

- active sprint
- upcoming sprint
- completed sprint archive
- per-sprint buckets such as ready, in progress, blocked, and done

The sprint tree must coexist with the existing hierarchy tree instead of replacing it.

**Testable ACs**:

- [ ] The left rail contains a dedicated Sprint tree section.
- [ ] Active, upcoming, and archived sprint groups are visible.
- [ ] Sprint status buckets are visible inside the active sprint scope.
- [ ] Entity hierarchy remains available in the same left rail.

#### REQ-F-006 - Detail drawer preserves entity-level drill-in

**Trace**: feature.md seed description, recommendation doc, wireframes.

When the user selects an item in Sprint mode, the right-side detail drawer must show the same
entity context the viewer already provides elsewhere:

- entity metadata
- current status
- status history
- assignment history
- related work
- jump back to the entity dashboard or entity view

Sprint mode must not reduce the ability to inspect a selected entity in depth.

**Testable ACs**:

- [ ] Selecting a sprint item opens the relevant entity detail in the drawer.
- [ ] The drawer shows current status and history information.
- [ ] A clear jump-back path to the entity view remains available.

#### REQ-F-007 - Sprint mode stays read-first and minimizes accidental mutation

**Trace**: feature.md out-of-scope guidance, recommendation doc.

The first release must keep Sprint mode primarily observational. Any write-capable planning
actions must be explicit, isolated, and visually subordinate to the main dashboard. The default
interaction should be inspect, filter, and compare, not mutate.

**Testable ACs**:

- [ ] The default Sprint experience is informational, not mutation-first.
- [ ] Write-capable controls are visually distinct from the main observational panels.
- [ ] The user can inspect the full sprint state without making changes.

### 1.2 Non-Functional Requirements

#### REQ-NF-001 - Preserve the embedded single-file viewer architecture

**Description**: Sprint mode must extend the current embedded `viewer.html` SPA. It must not
introduce a frontend framework, a bundler, or a second dashboard application.

**Target**: No build-step change and no frontend stack migration for this feature.

#### REQ-NF-002 - Keep the UI responsive within the existing viewer model

**Description**: Sprint mode should feel like an extension of the current viewer, not a slow
mode switch.

**Target**: Mode switching and initial Sprint overview render should remain responsive on the same
projects that already load the current viewer acceptably.

#### REQ-NF-003 - Accessibility and keyboard parity

**Description**: Sprint mode navigation must be keyboard accessible and must not depend solely on
color to communicate state.

**Target**: Mode switches, subview switches, and sprint-tree selection should be focusable and
usable by keyboard.

#### REQ-NF-004 - Local-only security posture is unchanged

**Description**: Sprint mode must inherit the existing local-viewer security model. It must not
expand network exposure beyond the current localhost-only viewer assumptions.

**Target**: No new auth surface and no broader CORS exposure.

### 1.3 Acceptance Criteria

**Scenario 1: Entering Sprint mode**

- **Given** the viewer is open
- **When** the user selects `Sprint`
- **Then** the viewer switches to Sprint mode
- **And** `Overview` is selected by default
- **And** the current sprint health is visible without additional navigation

**Scenario 2: Inspecting planning state**

- **Given** Sprint mode is open
- **When** the user selects `Plan`
- **Then** the view shows filters, candidate work, and capacity visibility
- **And** the user can inspect items without leaving the planning surface

**Scenario 3: Reviewing sprint history**

- **Given** Sprint mode is open
- **When** the user selects `Report`
- **Then** burndown and velocity data are visible
- **And** empty or partial data renders safely

**Scenario 4: Tree and drawer coordination**

- **Given** the left rail shows sprint scopes and the entity hierarchy
- **When** the user selects a sprint item or entity
- **Then** the detail drawer updates to the selected item
- **And** the viewer keeps the sprint context visible alongside the hierarchy

---

## 2. Architecture

### 2.1 Scope of changes

| File | Change |
| --- | --- |
| `internal/viewer/assets/viewer.html` | Add Sprint mode state, subview routing, sprint tree rendering, overview/plan/report panels, detail drawer refinements, and supporting CSS/JS. |
| `internal/services/viewer_service.go` | Add Sprint-facing aggregation methods and DTOs that compose existing sprint services. |
| `internal/api/viewer/handler.go` | Add viewer routes for Sprint data and wire them through the existing read-only handler pattern. |
| `internal/api/viewer/types.go` | Re-export any new viewer DTOs needed by the handler tests. |
| `internal/viewer/server/wire.go` | Inject sprint service dependencies into the viewer service wiring. |
| `internal/services/sprint_service.go` | Reuse existing sprint readiness, capacity, planning, and backlog outputs. |
| `internal/services/sprint_analytics_service.go` | Reuse existing burndown, velocity, and summary outputs. |
| `internal/cli/services_global.go` | Reuse the existing service factory patterns as the dependency source for the viewer wiring. |
| `internal/cli/commands/sprint.go` | Reference implementation for the sprint information model and user-facing semantics. |
| `internal/viewer/assets_test.go` | Extend viewer asset tests to cover Sprint mode markers and regressions. |

### 2.2 Backend design

Sprint mode should be implemented by extending the current viewer aggregation layer rather than
creating a separate dashboard service. The current codebase already has the right lower-level
building blocks:

- `SprintService` for readiness, capacity, planning, and backlog shaping
- `SprintAnalyticsService` for burndown, velocity, and summary reporting
- `ViewerService` as the read-only facade consumed by the viewer handlers

The recommendation is to add Sprint-specific viewer methods that compose the existing sprint
services into the exact shapes the SPA needs. That keeps the browser layer thin and preserves the
repo convention that HTTP handlers stay simple.

Proposed backend responsibilities:

- load current and upcoming sprint context
- derive Overview metrics
- derive Plan candidates and filters
- derive Report data for burndown and velocity
- provide sprint tree and entity-drill data in a shape the SPA can consume

The viewer API should remain read-only. Any future write-capable planning action should route to
the existing sprint/service paths, not a new viewer mutation surface.

### 2.3 Frontend design

The Sprint UI should remain inside the existing embedded `viewer.html` SPA.

Recommended UI structure:

- header mode switch with `Dashboard`, `Sprint`, and the current entity/doc surfaces
- left rail with a `Sprint` tree section above the entity hierarchy
- main canvas with `Overview`, `Plan`, and `Report` subviews
- right-side detail drawer for item inspection

The frontend state model should add a Sprint mode without breaking the current dashboard and
entity/doc flows. The simplest safe approach is to extend the existing application state machine
with:

- current top-level mode
- current Sprint subview
- selected sprint key or scope
- selected entity key for the detail drawer

The UI should reuse the existing dark, IDE-style styling language from `viewer.html` instead of
introducing a new visual system.

### 2.4 Data integration

Sprint mode should read from existing sprint data and analytics sources. The project already has
the necessary persistence surfaces:

- `sprints`
- `sprint_assignments`
- `task_history`
- `tasks`
- `bugs`
- `change_cards`
- `tech_debts`

No schema change is indicated by the current research. The feature should prefer service-layer
composition over new SQL patterns in the browser-facing layer.

### 2.5 Security and performance

- Keep the existing localhost-only viewer posture.
- Do not add websocket or SSE push in the first release.
- Keep Sprint mode read-first so the UI does not become dependent on write-side validation to be
  useful.
- Preserve the existing no-build-step embedded asset strategy.

### 2.6 Testing strategy

The feature should be validated at three levels:

- viewer asset tests for Sprint mode markers and state transitions
- handler/service tests for the new Sprint aggregation methods
- integration tests that confirm the sprint surface uses the existing viewer wiring and does not
  regress the current dashboard/entity views

### 2.7 Exit gate

This spec is complete when:

- the requirements above are specific and testable
- the architecture section identifies the actual files and layers to extend
- the spec is grounded in the existing viewer and sprint services
- the next workflow step can break the work into implementation tasks without reopening scope

