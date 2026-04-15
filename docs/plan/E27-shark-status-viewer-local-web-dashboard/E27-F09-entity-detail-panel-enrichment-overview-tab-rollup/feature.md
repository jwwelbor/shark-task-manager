---
feature_key: E27-F09-entity-detail-panel-enrichment-overview-tab-rollup
epic_key: E27
title: Entity Detail Panel Enrichment - Overview Tab, Rollups, and Clickable Navigation
description: Add a rich Overview tab as the default landing tab for all entity types, containing status rollups, child entity tables, notes, related docs, dependencies, and breadcrumb navigation — bringing the web viewer to parity with `shark get` output while preserving the existing Dashboard tab (charts, recent activity, stale entities).
status: draft
execution_order: 9
---

# E27-F09 — Entity Detail Panel Enrichment

**Feature Key**: E27-F09  
**Epic**: [E27 — Shark Status Viewer - Local Web Dashboard](../../epic.md)

---

## Goal

### Problem

When a developer or AI agent clicks an entity in the `shark web` viewer, the detail panel shows only basic property fields — status badge, timestamps, file path. It does not surface the contextually rich information that `shark get {entity}` shows in the terminal: feature/task rollup counts, child entity tables, notes and decisions, related documents, or dependency chains.

Users who want to understand the actual state of an epic, feature, or task must either switch to the terminal or hunt across multiple tab clicks to assemble the picture. The panel currently requires three or four separate interactions to answer: "What is the status of everything inside this thing, and what needs my attention?"

### Solution

Add a universal **Overview tab** as the default landing tab for all entity types. For epics, Overview becomes the new default, and the existing Dashboard tab is **preserved** (it contains unique content — CSS donut charts, cross-entity Recent Activity, and Stale Entities — that has no equivalent elsewhere). For features and tasks, Overview is simply added as a new default tab. Content per type:

- **Epics**: Feature Status Rollup pills, Task Rollup with segmented progress bar, clickable Features table, Notes list, Related Docs
- **Features**: Task Status Rollup pills, Work Breakdown mini-bars (agent/human/qa), Action Items (blocking tasks), clickable Tasks table, Notes, Related Docs
- **Tasks**: Dependency block (depends-on / blocked-by / blocks — all keys clickable), Notes, Related Docs

Additionally, every entity key rendered anywhere in the detail panel becomes a clickable navigation link, and a breadcrumb is added beneath the entity title for hierarchical orientation.

### Impact

- Users can answer "what is the health of this entity?" in one glance, without switching to the terminal
- Navigation between related entities is a single click instead of a sidebar scroll-and-expand sequence
- Notes and decisions surface on first view, eliminating the hunt through spec markdown
- Retains all existing functionality: copy-filepath button, Spec/Edit tab, History tab, Dashboard tab (epics), Files tab

---

## User Personas

### Developer / AI Agent Working in `shark web`

**Profile**:
- Uses `shark web` to monitor project state, review entity specs, and navigate the hierarchy
- Comfortable with both the CLI and the web viewer
- Frequently compares what they see in the viewer to what `shark get` would show

**Goals Related to This Feature**:
1. Understand the rollup state of an epic or feature without running CLI commands
2. Navigate from an epic to a specific blocked feature with one click
3. See notes and decisions without opening the spec markdown

**Pain Points This Feature Addresses**:
- Current panel is sparse — "properties only" — with no child entity visibility
- Keys appear as inert text, requiring sidebar scrolling to navigate to a related entity
- Notes are buried inside spec markdown files, invisible in the panel view

**Success Looks Like**:
Clicking an epic in the sidebar immediately shows a structured, scannable overview with the features table, rollup badges, and recent notes. No terminal tab-switching required.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer viewing an epic, I want to see a Feature Status Rollup and a clickable features table on the Overview tab so that I can immediately assess progress and navigate to any feature in one click.

**Acceptance Criteria**:
- [ ] Overview tab is the default tab when an entity is selected
- [ ] Feature Status Rollup shows colored pills with status name and count
- [ ] Features table displays: key (clickable), title (truncated), status badge, progress bar
- [ ] Clicking a feature key in the table navigates the viewer to that feature's detail panel
- [ ] Table shows all features; a scrollable max-height applies if >10 features

**Story 2**: As a developer viewing a feature, I want to see a Task Status Rollup, an Action Items section (blocking tasks), and a clickable tasks table so that I know what needs attention.

**Acceptance Criteria**:
- [ ] Task Status Rollup pills are shown with count per status
- [ ] Action Items section shows only tasks in `blocks_feature: true` statuses
- [ ] Each action item row shows status badge, clickable key, and truncated title
- [ ] Tasks table shows: key (clickable), title, status badge, execution order
- [ ] Clicking a task key navigates to that task's detail panel

**Story 3**: As a developer viewing a task, I want to see its dependency block (depends-on, blocked-by, blocks) with all keys clickable so that I can navigate the dependency graph without touching the sidebar.

**Acceptance Criteria**:
- [ ] Overview tab shows three sub-sections: Depends On, Blocked By, Blocks
- [ ] Each entry shows: clickable key, title (truncated), status badge
- [ ] Empty groups show "(none)" in dim color
- [ ] "Blocked By" group has a red left border accent to visually signal risk
- [ ] Clicking a dependency key navigates to that entity's detail panel

**Story 4**: As a developer, I want to see entity notes on the Overview tab (capped at 5, with a "show more" option) so I can review decisions without opening the spec file.

**Acceptance Criteria**:
- [ ] Notes section appears for all entity types on the Overview tab
- [ ] Each note row shows: timestamp, type pill (decision/blocker/comment/etc.), truncated text
- [ ] Only most recent 5 notes are shown by default
- [ ] A "Show N more" button reveals remaining notes inline
- [ ] Hovering a note row shows the full text via `title` attribute
- [ ] Notes are fetched from a `/api/v1/viewer/notes/{key}` endpoint

**Story 5**: As a developer, I want a breadcrumb beneath the entity title showing my position in the hierarchy (e.g., `E27 > E27-F08 > E27-F08-002`) with each segment clickable, so I can navigate upward without the sidebar.

**Acceptance Criteria**:
- [ ] Breadcrumb renders for all entity types
- [ ] Epic entities: shows the epic key only (top of hierarchy)
- [ ] Feature entities: `E27 > E27-F08`
- [ ] Task entities: `E27 > E27-F08 > E27-F08-002`
- [ ] Each breadcrumb segment is a clickable link that navigates the viewer
- [ ] Breadcrumb keys use the entity's display key (short numeric format)

### Should-Have Stories

**Story 6**: As a developer, I want to see Related Docs on the Overview tab (feature and task entities) so I can open linked design documents with one click.

**Acceptance Criteria**:
- [ ] Related Docs section appears when an entity has related documents
- [ ] Each doc row shows: file icon, truncated path, [open] button
- [ ] Clicking [open] opens the doc in the viewer's Doc View (existing State 4)
- [ ] Section is hidden when no related docs exist

**Story 7**: As a developer, I want a Work Breakdown section on the Feature Overview (showing task counts for agent / human / qa_team responsibility) so I can see resource distribution at a glance.

**Acceptance Criteria**:
- [ ] Work Breakdown shows three rows: Agent, Human, QA Team
- [ ] Each row has a label, mini horizontal progress bar (proportional), and task count
- [ ] Values derived from task status `responsibility` field in workflow config
- [ ] Section is hidden or shows all zeros if no tasks exist

### Could-Have Stories

**Story 8**: As a developer, I want the properties grid to have a subtle left-border accent colored to the entity's current status so I can visually orient to health at a glance.

**Acceptance Criteria**:
- [ ] Properties grid card has a 3px left border using the entity's status color
- [ ] Color matches the existing status badge color palette

---

## Design Specification

### Panel Structure (After Enhancement)

```
┌──────────────────────────────────────────────────┐
│ ← Back  │  Title                        KEY      │
│          E27 > E27-F08 > E27-F08-002  ← breadcrumb│
├──────────────────────────────────────────────────┤
│ Properties Grid (status left-border accent)      │
│ ▸ Orchestrator Action (collapsible, unchanged)   │
├──────────────────────────────────────────────────┤
│ Epics:    [Overview●] [Spec] [History] [Dashboard] [Files] │
│ Features: [Overview●] [Spec] [History] [Files]             │
│ Tasks:    [Overview●] [Spec] [History] [Files]             │
│       (Edit button lives inside Spec tab)                  │
├──────────────────────────────────────────────────┤
│  Overview Tab Content (scrollable):              │
│  ── ROLLUP / WORK BREAKDOWN ──────────────────── │
│  ── ACTION ITEMS (feature only) ───────────────  │
│  ── CHILD ENTITY TABLE ────────────────────────  │
│  ── DEPENDENCIES (task only) ──────────────────  │
│  ── NOTES ─────────────────────────────────────  │
│  ── RELATED DOCS ──────────────────────────────  │
└──────────────────────────────────────────────────┘
```

> **Dashboard tab (epics only) is preserved** with its existing content intact:
> Entity Charts (donut charts), Status Overview (status count table), Feature Progress (per-feature bars),
> Recent Activity (cross-entity transition log, up to 25 entries), and Stale Entities (items not updated in >7 days).
> Overview overlaps intentionally with Status Overview and Feature Progress — the difference is that
> Overview is the quick-scan default, while Dashboard provides the deeper analytical/historical view.

### Overview Tab — Epic

| Section | Description |
|---------|-------------|
| Feature Status Rollup | Horizontal row of colored pills: `[● status count]` |
| Task Status Rollup | Colored pills + segmented progress bar (8px, status colors) |
| Features Table | `KEY (clickable) | TITLE | STATUS BADGE | PROGRESS BAR` |
| Notes | Most recent 5 notes, timestamp · type pill · text; "Show N more" |
| Related Docs | List of linked docs with [open] button |

### Overview Tab — Feature

| Section | Description |
|---------|-------------|
| Task Status Rollup | Horizontal row of colored pills + segmented progress bar |
| Work Breakdown | 3 rows (Agent / Human / QA Team) with mini bars and counts |
| Action Items | Tasks in `blocks_feature: true` statuses; badge + clickable key + title |
| Tasks Table | `KEY (clickable) | TITLE | STATUS BADGE | ORDER` |
| Notes | Most recent 5 notes |
| Related Docs | Linked docs with [open] button |

### Overview Tab — Task

| Section | Description |
|---------|-------------|
| Dependencies | "Depends On / Blocked By / Blocks" — each key clickable, status badge shown; red accent on Blocked By |
| Notes | Most recent 5 notes |
| Related Docs | Linked docs with [open] button |

### Section Header Pattern

All sections use the `SECTION NAME ─────────────────` divider:

```css
.ov-section-header {
  display: flex; align-items: center; gap: 10px;
  font-size: 10px; font-weight: 600; letter-spacing: 0.12em;
  text-transform: uppercase; color: var(--fg-dim);
  margin: 20px 0 10px;
}
.ov-section-header::after {
  content: ''; flex: 1; height: 1px; background: var(--border);
}
```

### Status Rollup Pills

```
[● todo 2]  [● in_progress 3]  [● completed 5]  [● blocked 1]
```

CSS: `rollup-pill` — inline-flex, items-center, gap 4px, `background: rgba(statusColor, 0.15)`, `border: 1px solid rgba(statusColor, 0.4)`, border-radius 12px, padding 2px 8px, font-size 11px.

### Segmented Progress Bar

Single 100%-wide bar (8px height) divided into segments proportional to task count per status, ordered by workflow phase. Percentage label right-aligned.

### Action Items Row

```css
.action-item-row {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 8px; border-radius: 0 4px 4px 0;
  border-left: 2px solid <statusColor>;
  background: rgba(statusColor, 0.08);
  margin-bottom: 4px;
}
```

### Child Entity Tables

No outer border. Rows use only bottom-border dividers. Hover: `background: var(--bg-3)`. Key column: monospace, accent-colored, cursor pointer. Progress bar: 80px inline bar.

### Note Rows

```
timestamp (monospace dim) · type pill · truncated text (title= for full)
```

Note type → color mapping: decision=purple, blocker=red, comment=gray, solution=green, implementation=blue, question=cyan, rejection=pink.

### Dependencies (Task)

Three labeled groups (Depends On / Blocked By / Blocks). Each entry: clickable key + title + status badge. "Blocked By" group has `border-left: 2px solid #c0392b`. Empty group shows "(none)".

---

## API Requirements

### New Endpoint: Notes

```
GET /api/v1/viewer/notes/{key}
Response: { notes: [{ type, content, created_at, agent }] }
```

Notes are already stored in the `entity_notes` table. This endpoint needs to be added to the viewer service and handler.

### New Endpoint: Related Docs

```
GET /api/v1/viewer/related-docs/{key}
Response: { docs: [{ path, description }] }
```

Related docs are stored in the `related_docs` table. This endpoint needs to be added.

### Existing Data (No New Endpoint Needed)

- **Rollup data**: Derived client-side from `treeData` flat list already loaded
- **Child entities**: Available via hierarchy data already fetched
- **Dependencies**: Available on the `entity` object (`depends_on`, `blocked_by`, `blocks` fields)
- **Breadcrumb**: Derived from `entity.parent` field + `findEntityByKey()`

---

## Requirements

### Functional Requirements

**Category: Overview Tab**

1. **REQ-F-001**: Overview tab added as the default tab for all entity types (epic, feature, task)
   - Priority: Must-Have
   - For epics: tab order becomes `[Overview] [Spec] [History] [Dashboard] [Files]`; Overview is the new default; Dashboard tab is **retained unchanged**
   - For features/tasks: tab order becomes `[Overview] [Spec] [History] [Files]`; Overview is the new default

2. **REQ-F-002**: Edit button moved from tab-level to inside Spec tab as an inline mode-toggle button
   - Priority: Must-Have

3. **REQ-F-003**: Feature Status Rollup (pills) rendered on Epic Overview
   - Priority: Must-Have

4. **REQ-F-004**: Task Status Rollup with segmented progress bar rendered on Epic Overview and Feature Overview
   - Priority: Must-Have

5. **REQ-F-005**: Clickable Features table on Epic Overview (key, title, status badge, progress bar)
   - Priority: Must-Have

6. **REQ-F-006**: Clickable Tasks table on Feature Overview (key, title, status badge, execution order)
   - Priority: Must-Have

7. **REQ-F-007**: Action Items section on Feature Overview showing tasks in `blocks_feature: true` statuses
   - Priority: Must-Have

8. **REQ-F-008**: Dependency block on Task Overview with three groups (depends-on, blocked-by, blocks), all keys clickable
   - Priority: Must-Have

9. **REQ-F-009**: Notes section on all entity types (capped 5, expandable), fetched from new `/notes/{key}` endpoint
   - Priority: Must-Have

10. **REQ-F-010**: Breadcrumb navigation beneath entity title with clickable segments
    - Priority: Must-Have

11. **REQ-F-011**: Related Docs section on Feature and Task Overview, fetched from new `/related-docs/{key}` endpoint
    - Priority: Should-Have

12. **REQ-F-012**: Work Breakdown mini-bars on Feature Overview (agent/human/qa_team)
    - Priority: Should-Have

13. **REQ-F-013**: Properties grid left-border accent in entity status color
    - Priority: Could-Have

**Category: Backend API**

14. **REQ-F-020**: `GET /api/v1/viewer/notes/{key}` endpoint returning notes for any entity key
    - Priority: Must-Have (required by REQ-F-009)

15. **REQ-F-021**: `GET /api/v1/viewer/related-docs/{key}` endpoint returning related docs for any entity key
    - Priority: Should-Have (required by REQ-F-011)

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Overview tab renders within 300ms of entity selection (notes/related-docs loaded asynchronously, UI shows immediately with skeleton loaders)

**Accessibility**

1. **REQ-NF-010**: All clickable entity keys have appropriate `role="button"` or anchor semantics and are keyboard-focusable

---

## Acceptance Criteria

**Scenario 1: Developer clicks an Epic**
- **Given** the viewer has loaded and an epic is selected in the sidebar
- **When** the entity detail panel renders
- **Then** the Overview tab is active by default
- **And** Feature Status Rollup pills are visible immediately
- **And** the Features table shows all features with clickable keys
- **And** Notes section shows up to 5 most recent notes

**Scenario 2: Developer navigates via child entity table**
- **Given** an epic detail panel is showing with a Features table
- **When** the developer clicks a feature key in the table
- **Then** the viewer navigates to that feature's detail panel
- **And** the sidebar scrolls to reflect the newly selected feature
- **And** the breadcrumb updates to show `E27 > E27-F08`

**Scenario 3: Developer views a task with blocked dependencies**
- **Given** a task detail panel is showing
- **When** the Overview tab renders
- **Then** the Blocked By group has a red left-border accent
- **And** each key in Blocked By is clickable and navigates to that dependency
- **And** the Depends On group shows "(none)" if empty

**Scenario 4: Developer clicks breadcrumb segment**
- **Given** a task detail panel is showing with breadcrumb `E27 > E27-F08 > E27-F08-002`
- **When** the developer clicks `E27-F08` in the breadcrumb
- **Then** the viewer navigates to feature E27-F08's detail panel

---

## Out of Scope

1. **Removing or modifying the existing Dashboard tab** — the Dashboard tab (Entity Charts, Recent Activity, Stale Entities, Status Overview, Feature Progress) is fully preserved; this feature only adds Overview as the new default tab ahead of it
2. **Real-time updates / WebSocket push** — Overview refreshes only on explicit entity selection, not automatically
3. **Note creation/editing in the panel** — notes are read-only in this feature; editing remains in spec files
4. **Drag-and-drop reordering** of child entities in the table
5. **Entity creation from the panel** — no "Add Task" button in the Overview
6. **Bug / Change Card entity types** in Overview — rollup tables limited to Epic/Feature/Task for now

---

## Dependencies

- E27-F08 (completed): Epic dashboard rendering patterns provide reference for rollup visualization
- E27-F07 (completed): Inline markdown editor establishes the Spec/Edit tab pattern to preserve
- E27-F03 (completed): SPA architecture and CSS variables used throughout
- Existing `entity_notes` table and `related_docs` table in the database (schema exists)

---

*Last Updated*: 2026-04-13
