# Sprint Planning Web UI Recommendation

This document executes [sprint-planning-web-ui-prompt.md](./sprint-planning-web-ui-prompt.md) and turns it into a concrete design recommendation for the Shark web UI.

## Recommendation

Use **both** a sprint dashboard and a sprint tree, with the dashboard as the primary surface and the tree as a persistent navigation layer.

Make the experience **interactive, but read-first**.

- The dashboard should be the thing users land on.
- The sprint tree should expose sprint scope, active work, blockers, and status movement without forcing users into the planning view.
- Planning actions should exist, but the default value of the screen should be visibility into status, work in progress, and change over time.

This is the right fit for Shark because the product is not just about planning sprints. It is about understanding what is happening, what changed, what is blocked, and how work is flowing. Planning matters, but status and live work visibility matter at least as much.

## UX Strategy

Design the surface as an **operational command board**, not a generic sprint planner.

- **Primary lens:** current sprint health and live work state.
- **Secondary lens:** backlog readiness and planning actions.
- **Tertiary lens:** historical reporting such as velocity and burndown.

The UI should answer, in this order:

1. What sprint am I looking at?
2. What is happening right now?
3. What is blocked or at risk?
4. What is ready to be pulled in?
5. What should I change next?

This order keeps the interface aligned with the user’s emphasis that viewing sprints, status, and current work is as important as planning itself.

## Information Architecture

### Global Navigation

- Keep the existing entity dashboard intact.
- Add a top-level **Sprint** entry alongside the current project/entity views.
- Treat sprint views as a sibling mode, not a nested detail panel.

### Sprint Mode Layout

#### Left Sidebar

- Project tree remains available.
- Add a sprint tree section above or adjacent to the entity hierarchy.
- The sprint tree should show:
  - active sprint
  - upcoming sprint
  - completed sprint archive
  - per-sprint work buckets such as ready, in progress, blocked, and done

#### Main Content

- Sprint summary header
- Readiness score
- Capacity by agent type
- Live status / work-happening strip
- Backlog inspection panel
- Burndown and velocity reporting
- Bulk planning actions

#### Secondary Panel

- Entity detail drill-in
- Status history
- Quick jump to entity dashboard
- Planning notes or rationale

## Screen / Component Structure

### 1. Sprint Overview Header

The header should make the current sprint identity obvious and expose the fastest health signals.

Include:

- sprint name and date range
- readiness score
- capacity utilization
- blocked count
- work-in-progress count
- delta since last refresh

### 2. Live Work Strip

This is the most important addition relative to a planning-only design.

It should surface:

- items currently moving
- newly blocked items
- items recently completed
- items whose status changed since the last refresh

This strip should be visually prominent, because it gives the user immediate situational awareness.

### 3. Capacity and Readiness Panel

Show capacity by agent type with clear over/under allocation indicators.

The readiness score should be explainable, not just numeric:

- what is ready
- what is missing
- what is over-committed
- what is stalled

### 4. Backlog Inspection Panel

This is the planning workspace, but it should not dominate the page.

It should support:

- filtering by status
- filtering by agent type
- filtering by sprint assignment
- adding or removing items from the sprint
- bulk selection

### 5. Reporting Panel

Keep burndown and velocity visible, but lower than live status in visual priority.

The reporting panel should answer:

- are we on track
- how is this sprint trending
- what changed versus forecast

### 6. Detail Drawer

When the user selects an item, show:

- entity metadata
- current status
- status history
- assignment history
- related work
- direct link back to the entity dashboard

## Interaction Model

### Minimum Usable Interactions

If this ships as a read-first surface, the minimum valuable interactions are:

- switch sprint
- expand and collapse sprint scopes
- filter by status / agent / work state
- inspect an entity in detail
- pin or unpin items in the current working set
- jump between sprint view and entity view

### Planning Interactions

If the surface becomes write-capable later, the first actions to add should be:

- bulk add to sprint
- bulk remove from sprint
- capacity adjustment by agent type
- mark item as ready or blocked within the sprint planning flow

Those actions should not displace the dashboard. They should sit behind explicit controls so the interface stays readable for casual users.

## Default Visibility

Show these by default:

- current sprint
- current work in progress
- blockers
- capacity pressure
- readiness score
- recent changes

Hide or de-emphasize these until requested:

- historical sprint archive
- full backlog beyond the current planning slice
- low-value metadata
- raw assignment history

## Candidate Approaches

### 1. Separate Sprint Tab Only

**Pros**

- simple to discover
- easy to implement
- clean mental model

**Cons**

- too narrow
- status and live work become secondary
- users lose the ability to navigate sprint context and entity context together

**Verdict:** insufficient on its own.

### 2. Sprint Tree Only

**Pros**

- excellent for navigation
- makes sprint membership and grouping visible

**Cons**

- lacks a strong planning / reporting center
- feels like an extension of the existing hierarchy instead of a true sprint surface

**Verdict:** useful, but incomplete.

### 3. Dashboard + Sprint Tree

**Pros**

- balances overview and drill-down
- keeps status visibility front and center
- supports both PM/Scrum Master workflows and power-user workflows
- scales better as sprint analytics grow

**Cons**

- more design complexity
- requires stronger hierarchy and information density management

**Verdict:** best option.

## Interactive Or Read-Only

The right answer is **interactive, but not centered on mutation**.

The initial UX should be:

- read-first
- filterable
- navigable
- explainable

That makes the dashboard immediately useful for:

- PMs and Scrum Masters scanning the sprint
- orchestrator-style users checking live work state
- casual users who just want to know what is happening

Write actions can come later without changing the core layout.

## Why This Is the Right Surface

This feature should not be framed as “a planning page.”

It should be framed as:

- a sprint observability surface
- a work-in-motion surface
- a planning surface only as a consequence of the above

That aligns with the repo’s existing dark IDE-style viewer and with how Shark is already used: to understand status, flow, and change, not just to prepare work.

If the user opens the sprint view, they should immediately understand:

- what is happening
- what is blocked
- what is ready
- what is at risk
- what should happen next

That is the strongest possible fit for the web UI.

## Final Answer

- **Does sprint planning belong in the web UI?** Yes.
- **Should the surface be interactive?** Yes, but primarily for inspection, filtering, and navigation in v1.
- **Should it use a tab, a tree, or both?** Both.
- **How should it stay cohesive with the existing viewer?** Keep the same dark, IDE-style visual language, preserve the entity dashboard, and treat sprint mode as a sibling view rather than a separate product.

*Last Updated*: 2026-05-06
