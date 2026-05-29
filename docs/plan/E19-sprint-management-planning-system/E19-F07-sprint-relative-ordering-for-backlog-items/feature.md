---
feature_key: E19-F07-sprint-relative-ordering-for-backlog-items
epic_key: E19
title: Sprint-relative ordering for backlog items
description: Add a sprint-scoped pull order to sprint_assignments so users can sequence work within a sprint independently of epic/feature execution order.
size: 5
---

# Sprint-relative ordering for backlog items

**Feature Key**: E19-F07-sprint-relative-ordering-for-backlog-items

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Problem

`shark sprint next` sorts candidates by entity-level `ExecutionOrder`, then `priority`, then `assigned_at` (see `internal/services/sprint_service.go:1007-1030`). Those fields encode *project-level* importance — "this should be built before that within the epic/feature" and "this matters overall." They do not encode *sprint-level* intent — "of the things I committed to this sprint, here's the order I want to pull them."

The two concepts collide:

- A high-priority task in an epic may not be the right next pull this sprint (dependencies, agent availability, context).
- Reordering for a sprint by editing `--order` / `--priority` mutates the entity's project-level ordering, which leaks back into all other views and into future sprints.
- `--order` also renumbers siblings by default, so a tactical sprint nudge causes ripple edits across the whole feature.

Observed in wormwoodGM: `sprint next` returned the least-relevant queued task because the better candidates had no `ExecutionOrder` set and one outlier did.

## Solution

Add a sprint-scoped order column to `sprint_assignments` and use it as the primary sort key in `GetNextTask`. Default it to insertion order so existing workflows keep working; let users explicitly reorder when needed.

## Requirements

### Data model

- `sprint_assignments.sprint_order INTEGER` (nullable)
- Migration: backfill `sprint_order` for active/planning sprints in `assigned_at` order; bump `CurrentSchemaVersion`
- Uniqueness: `(sprint_id, sprint_order)` unique when not null

### Defaults

- `sprint add`: auto-assign `sprint_order = max(existing) + 1`
- `sprint close --carryover=next`: carry relative order across, renumber from 1 in the receiving sprint
- `sprint close --carryover=backlog`: clear `sprint_order`

### Sort precedence in `GetNextTask`

1. `sprint_order` (nulls last) — sprint-relative intent wins
2. `ExecutionOrder` — fall back to project-level intent
3. `priority` — then importance
4. `assigned_at` — then FIFO

### Commands

- `shark sprint reorder <sprint> <entity> <position>` — move to position N, shift others
- `shark sprint reorder <sprint> --top <entity>` / `--bottom <entity>` — common-case shortcuts
- `shark sprint add` accepts `--at=<position>` (defaults to end)

### Display

- `shark sprint backlog --view=ordered` shows numbered list reflecting `sprint_order`; consider making this the default for active sprints
- `shark sprint plan` surfaces the column
- JSON output exposes `sprint_order` on each assignment

## Acceptance Criteria

- [ ] `sprint add` assigns `sprint_order` automatically; new items land at the end
- [ ] `sprint next` returns items in `sprint_order` ascending, falling back to existing tiers
- [ ] `sprint reorder` moves an item and renumbers the rest without touching entity-level fields
- [ ] Editing entity `priority` or `ExecutionOrder` does not change `sprint_order`
- [ ] `sprint close --carryover=next` preserves relative order; `--carryover=backlog` clears it
- [ ] Migration backfills existing sprints; no manual intervention required
- [ ] `sprint backlog --view=ordered` shows numbered pull queue

## Open Questions

- **Dense vs. sparse ordering**: renumber on every move (simple, small sprints) or fractional indices (cheap inserts, ugly numbers)? Lean dense given typical sprint size.
- **Carryover semantics**: preserve relative order on `--carryover=next` and renumber to 1..N — confirm this matches user expectation.
- **Interaction with `--parallel`**: `sprint_order` stays strictly unique per sprint even when entity `ExecutionOrder` ties. Verify this doesn't break parallel-execution semantics.
- **Default view**: should `sprint backlog` default to the ordered view for active sprints, or keep the current view and require an opt-in flag?

## Out of Scope

- Reordering across sprints (move item from one sprint to another while preserving its order in the destination)
- Drag-and-drop UI — CLI only
- Auto-reordering based on dependencies or readiness scores

## Migration / Tech-debt origin

Migrated from TD-033 on 2026-05-15. TD-033 was deleted after migration; its history lives here.

---

*Last Updated*: 2026-05-15
