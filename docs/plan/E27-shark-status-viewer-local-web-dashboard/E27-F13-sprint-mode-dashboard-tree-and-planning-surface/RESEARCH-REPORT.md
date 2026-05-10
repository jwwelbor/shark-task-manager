# E27-F13 Research Report: Sprint Mode Dashboard, Tree, and Planning Surface

**Date**: 2026-05-07  
**Status**: Research Complete  
**Feature**: E27-F13 - Sprint Mode Dashboard, Tree, and Planning Surface

## Executive Summary

`E27-F13` is a real feature, not a task. It adds a new top-level Sprint mode to the existing Shark viewer with three subviews, `Overview`, `Plan`, and `Report`, plus sprint-aware navigation and a read-first planning surface. The important finding is that this does not require a new frontend stack or a new sprint data layer. The repo already contains the pieces needed to extend the existing viewer: a single embedded HTML SPA, read-only viewer APIs, a sprint service layer, sprint analytics repositories, and a CLI sprint command surface.

The cleanest implementation path is to extend the existing viewer architecture rather than introduce a separate dashboard system. Sprint-specific UI should sit on top of the existing embedded `viewer.html` state machine, while sprint data should come from the existing `SprintService` and `SprintAnalyticsService` APIs rather than direct repository calls from the browser layer.

## 1. Existing Implementations

### Viewer shell and SPA

- [`internal/viewer/assets/viewer.html`](/home/jwwel/projects/shark-task-manager/internal/viewer/assets/viewer.html)
  - Existing single-file SPA with the three-panel layout, header, sidebar tree, dashboard sections, entity detail view, and doc view.
  - Already contains the UI state machine that Sprint mode can extend.
- [`internal/viewer/assets.go`](/home/jwwel/projects/shark-task-manager/internal/viewer/assets.go)
  - Embeds the viewer asset into the binary via `go:embed`.
- [`internal/viewer/server/server.go`](/home/jwwel/projects/shark-task-manager/internal/viewer/server/server.go)
  - Serves the embedded viewer asset and the API surface.
- [`internal/viewer/server/wire.go`](/home/jwwel/projects/shark-task-manager/internal/viewer/server/wire.go)
  - Wires the viewer server dependencies.
- [`cmd/server/main.go`](/home/jwwel/projects/shark-task-manager/cmd/server/main.go)
  - Hosts the viewer server entrypoint.
- [`internal/cli/commands/web.go`](/home/jwwel/projects/shark-task-manager/internal/cli/commands/web.go)
  - User-facing `shark web` command that starts the viewer in-process.

### Viewer API and service layer

- [`internal/api/viewer/handler.go`](/home/jwwel/projects/shark-task-manager/internal/api/viewer/handler.go)
  - Read-only viewer HTTP handlers already exist and are organized as thin parse/validate/format wrappers.
- [`internal/api/viewer/types.go`](/home/jwwel/projects/shark-task-manager/internal/api/viewer/types.go)
  - Re-exports the viewer service DTOs for the handler layer.
- [`internal/services/viewer_service.go`](/home/jwwel/projects/shark-task-manager/internal/services/viewer_service.go)
  - Aggregation layer for the existing dashboard, hierarchy, history, file, notes, related docs, and tag workflows.
  - This is the natural place to add Sprint-view aggregation methods.

### Sprint domain services already available for reuse

- [`internal/services/sprint_service.go`](/home/jwwel/projects/shark-task-manager/internal/services/sprint_service.go)
  - Already provides sprint lifecycle, planning, backlog, readiness, and capacity operations.
  - Exported outputs worth reusing:
    - `BacklogOptions`
    - `CapacityRow`
    - `SprintReadiness`
    - `SprintPlanView`
- [`internal/services/sprint_analytics_service.go`](/home/jwwel/projects/shark-task-manager/internal/services/sprint_analytics_service.go)
  - Already provides sprint reporting primitives such as velocity, burndown, and sprint summary.
- [`internal/repository/sprint/repository.go`](/home/jwwel/projects/shark-task-manager/internal/repository/sprint/repository.go)
  - Sprint CRUD and status updates at the repository layer.
- [`internal/repository/sprint/analytics.go`](/home/jwwel/projects/shark-task-manager/internal/repository/sprint/analytics.go)
  - Read-only aggregate queries for velocity, assignment sets, completion events, and phase timing.
- [`internal/repository/sprint/analytics_types.go`](/home/jwwel/projects/shark-task-manager/internal/repository/sprint/analytics_types.go)
  - DTOs used by the analytics repository.
- [`internal/cli/commands/sprint.go`](/home/jwwel/projects/shark-task-manager/internal/cli/commands/sprint.go)
  - CLI already exposes sprint planning/readiness/capacity/reporting flows, which gives a good precedent for the viewer UX.
- [`internal/cli/services_global.go`](/home/jwwel/projects/shark-task-manager/internal/cli/services_global.go)
  - Service factory already wires `SprintService` and `SprintAnalyticsService`.

### Relevant epic documentation and prior art

- [`docs/plan/E27-shark-status-viewer-local-web-dashboard/architecture.md`](/home/jwwel/projects/shark-task-manager/docs/plan/E27-shark-status-viewer-local-web-dashboard/architecture.md)
  - The epic architecture already says the viewer should extend the existing server and viewer stack.
- [`docs/plan/E27-shark-status-viewer-local-web-dashboard/sprint-planning-web-ui-recommendation.md`](/home/jwwel/projects/shark-task-manager/docs/plan/E27-shark-status-viewer-local-web-dashboard/sprint-planning-web-ui-recommendation.md)
  - Recommendation explicitly prefers dashboard + sprint tree, interactive but read-first.
- [`docs/plan/E27-shark-status-viewer-local-web-dashboard/sprint-planning-web-ui-wireframes.md`](/home/jwwel/projects/shark-task-manager/docs/plan/E27-shark-status-viewer-local-web-dashboard/sprint-planning-web-ui-wireframes.md)
  - Wireframes define the `Overview`, `Plan`, and `Report` model and the layout priorities.
- [`docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F03-single-file-spa-ide-style-dark-dashboard-interface/RESEARCH-REPORT.md`](/home/jwwel/projects/shark-task-manager/docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F03-single-file-spa-ide-style-dark-dashboard-interface/RESEARCH-REPORT.md)
  - Prior art for the viewer shell, SPA state machine, and embedded asset pattern.

## 2. Integration Points

### Services

- `internal/services/viewer_service.go` is the best integration point for Sprint-specific viewer aggregation.
- The viewer layer should call `SprintService` and `SprintAnalyticsService` rather than talking to sprint repositories directly.
- A Sprint mode will likely need one or more new viewer-service methods that compose:
  - sprint identity and status
  - readiness and capacity
  - backlog or assigned-scope data
  - burndown and velocity outputs
  - recent changes or blockers

### Repositories

- `internal/repository/sprint/repository.go` already supports sprint lookup and lifecycle state.
- `internal/repository/sprint/analytics.go` already supports the reporting queries the Report tab will need.
- `internal/repository/sprint/analytics_types.go` already defines the analytics DTOs, so the viewer should not invent parallel low-level DTOs.

### CLI commands

- `internal/cli/commands/sprint.go` already exposes the same conceptual data the viewer wants to display.
- If Sprint mode later needs write-capable planning controls, that CLI/service surface gives the correct backend precedent for mutation paths.

### Tables and data sources

The current Sprint feature set appears to be built on top of existing tables rather than requiring a new schema:

- `sprints`
- `sprint_assignments`
- `task_history`
- `tasks`
- `bugs`
- `change_cards`
- `tech_debts`

For viewer aggregation, the likely read-only extension point is a new viewer-facing composition layer, not a new persistence model.

## 3. Inter-Feature Dependency Map

### Direct dependencies in this epic

- `E27-F01` - DB init extraction and cloud-aware startup
  - Needed so the server can initialize against the same project root rules as the CLI.
- `E27-F02` - Viewer API endpoints
  - Needed for the viewer to fetch dashboard/hierarchy/history data.
- `E27-F03` - Single-file SPA
  - Sprint mode extends this existing viewer shell instead of replacing it.
- `E27-F05` - Server wiring
  - Needed for serving the viewer asset and registering API routes.
- `E27-F09` - Entity detail panel enrichment
  - Sprint mode can reuse the existing detail-drawer pattern.
- `E27-F10` - Tree view enhancements
  - Sprint mode needs the existing tree/navigation patterns to stay consistent.
- `E27-F11` - Surface entity size
  - Relevant for readiness and planning visibility because sprint planning depends on sizing.
- `E27-F12` - Mermaid diagram support
  - Useful for the reporting and planning docs that may be embedded in the viewer.

### Most relevant sibling context

- `E27-F09` and `E27-F10` are the strongest UI siblings because Sprint mode reuses the same tree/detail mechanics.
- `E27-F03` is the architectural sibling because it established the SPA boundary and state model.
- `E27-F02` is the backend sibling because it established the viewer read API pattern.

## 4. Extension-vs-New Analysis

### Viewer UI

- **Extend**, do not rewrite.
- `internal/viewer/assets/viewer.html` already has the layout, sidebar, dashboard, and detail-panel patterns that Sprint mode needs.
- Sprint mode should add a top-level mode switch and `Overview` / `Plan` / `Report` subviews inside the same SPA rather than introducing a second app.

### Viewer API

- **Extend**, do not replace.
- The existing `internal/api/viewer` package should gain Sprint-focused endpoints or viewer-service methods as needed.
- The Sprint UI should keep using the read-only viewer API layer, not the CLI commands directly.

### Sprint data and reporting

- **Reuse existing services and repositories**.
- `SprintService` already computes planning and readiness data.
- `SprintAnalyticsService` already computes report-oriented data.
- New viewer code should compose these outputs instead of reimplementing the algorithms.

### Data model

- **No new schema is indicated by the current evidence**.
- The sprint service and analytics repository already read the tables needed for planning and reporting.
- If a later specification discovers a missing aggregation, that should be a targeted extension to the service layer rather than a schema redesign.

### Frontend framework choice

- **Keep the existing vanilla JS / embedded HTML approach**.
- The current repo has already committed to the single-file embedded viewer model.
- Adding a frontend framework would increase build complexity without solving a demonstrated problem in this feature.

## 5. Recommended Implementation Approach

1. Extend `internal/services/viewer_service.go` with Sprint-mode aggregation methods that compose `SprintService` and `SprintAnalyticsService`.
2. Extend `internal/api/viewer/handler.go` and `internal/api/viewer/types.go` with Sprint-specific viewer endpoints or response DTOs.
3. Update `internal/viewer/assets/viewer.html` to add a top-level `Sprint` mode and the `Overview`, `Plan`, and `Report` subviews.
4. Reuse the current sidebar tree and detail-drawer patterns instead of building a second navigation system.
5. Keep the first pass read-first:
   - overview health
   - readiness and capacity
   - assigned scope / planning candidates
   - burndown and velocity reporting
6. Avoid new tables, new build tooling, or a new frontend framework until a later phase proves they are necessary.

### Practical sequencing

- First lock the data contract for Sprint overview, plan, and report outputs.
- Then extend the viewer service and API.
- Then wire the new UI states into the embedded SPA.
- Finally add tests for the new viewer-service aggregation and the new rendering states in `viewer.html`.

## Conclusion

`E27-F13` should move forward as an extension of the existing Shark viewer stack, not as a parallel application. The repository already contains the sprint service layer, analytics queries, viewer API shape, and embedded SPA architecture required to support this feature. The main work is integration and UI composition, not foundational data modeling.

