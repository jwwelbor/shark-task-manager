---
feature_key: E27-F08-epic-level-dashboard-and-enhanced-entity-info
epic_key: E27
title: Epic-Level Dashboard and Enhanced Entity Info
description: Add an epic-scoped dashboard tab to the web viewer, rename "Info" to "Spec", and enrich the entity info panel to match `shark get` output.
---

# Epic-Level Dashboard and Enhanced Entity Info

**Feature Key**: E27-F08

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Goal

### Problem

When a developer clicks an epic in the tree view, they land directly on the spec file. There is no at-a-glance summary of the epic's features, task counts, or overall progress. The project-level dashboard provides this view for the whole project but cannot be scoped to a single epic. Additionally, the "Info" label on the entity view tab is ambiguous — it actually shows the entity's markdown spec file — and the properties panel shows only a minimal subset of the data available from `shark get {entity}`.

### Solution

- Add a **Dashboard** tab for epics (shown by default when clicking an epic) that renders the same dashboard components as the project-level view, filtered to the selected epic.
- Rename the **Info** tab to **Spec** across all entity types to accurately describe its content.
- Expand the entity **properties panel** for all entity types to surface the same rich fields returned by `shark get {entity}`, with orchestrator action details in a collapsible section.

### Impact

- Developers get immediate epic-scoped status visibility without dropping to the CLI
- Clearer "Spec" label reduces onboarding friction
- Richer info panel eliminates most reasons to run `shark get` separately

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want to click an epic in the tree view and see a dashboard filtered to that epic, so I can immediately understand its progress without switching to the CLI.

**Acceptance Criteria**:
- [ ] Clicking an epic opens a "Dashboard" tab by default (not "Spec")
- [ ] Dashboard shows entity charts, status overview, feature progress, and recent activity scoped to the epic
- [ ] A "Spec" tab is also available and shows the epic's markdown file (same as current "Info" tab)

**Story 2**: As a developer, I want the "Info" button renamed to "Spec" on all entity types, so that the label matches what the tab actually shows.

**Acceptance Criteria**:
- [ ] The "Info" toggle button label reads "Spec" for all entity types (epic, feature, task, bug, change card)
- [ ] All existing tab behavior (markdown rendering, edit pencil, transitions, files) is unchanged

**Story 3**: As a developer, I want the entity properties panel to show the same rich data as `shark get {entity}`, so I don't have to context-switch to the CLI for basic entity details.

**Acceptance Criteria**:
- [ ] Properties panel shows all fields returned by `shark get` for the given entity type
- [ ] Orchestrator action details are present but collapsed by default
- [ ] Panel works correctly for epic, feature, task, bug, and change card entity types

---

## Requirements

### Functional Requirements

**Epic Dashboard Tab**

1. **REQ-F-001**: Dashboard tab for epics
   - **Description**: When any epic is clicked in the tree view, the entity view opens with the "Dashboard" tab active by default. The Dashboard tab renders the same visual components as the project-level dashboard (entity charts, status breakdown, feature progress table, recent activity) but filtered to the selected epic's key.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Dashboard tab button appears to the left of the Spec tab button
     - [ ] Dashboard tab is the default active tab when clicking an epic
     - [ ] All dashboard sections respect the epic filter
     - [ ] Navigating to a feature or task does NOT show the Dashboard tab (epic-only)

2. **REQ-F-002**: Spec tab rename
   - **Description**: The "Info" toggle button is renamed to "Spec" on the entity view tab bar for all entity types.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Button label reads "Spec" (was "Info")
     - [ ] Internal IDs and JS variables may remain as-is
     - [ ] All existing tab behavior is unchanged

3. **REQ-F-003**: Enhanced entity properties panel
   - **Description**: The properties panel (shown in the entity view for all types) is updated to display all fields returned by `shark get {entity}`, matching the richness of the CLI output.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] All fields from `shark get` JSON response are displayed
     - [ ] Orchestrator action details render in a collapsible `<details>` element
     - [ ] Panel is correct for epic, feature, task, bug, and change card
     - [ ] No existing fields are removed

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Epic dashboard load time
   - **Description**: Epic dashboard data fetches reuse existing API endpoints (no new backend endpoints required)
   - **Target**: Dashboard renders within the same time as the project-level dashboard

---

## Acceptance Criteria

**Scenario 1: Click epic in tree — dashboard opens by default**
- **Given** the web viewer is open with a loaded project
- **When** the user clicks an epic node in the tree view
- **Then** the entity view opens with the Dashboard tab active
- **And** the dashboard content is filtered to that epic

**Scenario 2: Switch to Spec tab on an epic**
- **Given** an epic's entity view is open showing the Dashboard tab
- **When** the user clicks the "Spec" tab
- **Then** the epic's markdown spec file is rendered (same as current "Info" tab behavior)

**Scenario 3: Non-epic entity view shows only Spec tab (no Dashboard)**
- **Given** the user clicks a feature or task in the tree
- **When** the entity view opens
- **Then** no Dashboard tab is shown; the Spec tab is the default

**Scenario 4: Info → Spec rename visible on all entity types**
- **Given** any entity (epic, feature, task, bug, change card) is open in the entity view
- **When** the user looks at the tab bar
- **Then** the tab label reads "Spec" (not "Info")

---

## Out of Scope

1. **Feature-level or task-level dashboard tabs** — Dashboard tab is epic-only in this feature. Future work could extend to features.
2. **New backend API endpoints** — All data for the epic dashboard comes from existing endpoints with client-side filtering.
3. **Dashboard customization** — The epic dashboard mirrors the project dashboard; no drag-and-drop or widget configuration.

---

## Dependencies & Integrations

- **`/api/v1/viewer/summary`** — Reused for epic-scoped entity counts (with epic key filter or client-side filter)
- **`/api/v1/viewer/hierarchy`** — Provides the epic's feature/task tree
- **`/api/v1/viewer/recent-activity`** — Filtered to epic scope
- **`/api/v1/viewer/features/{key}/tasks`** — Used for feature progress within the epic
- **`internal/viewer/assets/viewer.html`** — Single file containing all frontend code; all changes are in this file

---

*Last Updated*: 2026-04-13
