---
feature_key: E07-F39
epic_key: E07
title: Remove legacy relationship tables and dual-path query code
---

# Remove Legacy Relationship Tables and Dual-Path Query Code

## Problem

Three legacy relationship tables (`task_relationships`, `feature_relationships`,
`epic_relationships`) still have active read/write paths in the codebase despite
data having been migrated to the canonical `entity_relationships` table in E21-F11.
This means writes can go to stale tables, reads may miss data, and the same
relationship logic is maintained in multiple places.

Compounding this, the viewer's `Hierarchy()` endpoint was built with a bespoke
bulk-load adapter (`taskRelAdapter`) that bypasses `EntityRelationshipService`
entirely. The stated reason was avoiding N+1 queries across the full hierarchy —
but this concern was premature: the hierarchy endpoint already makes many queries,
one-per-task for relationships is not meaningfully more expensive, and the bypass
produced real bugs (task-only visibility, drift to legacy table).

## Intended Architecture

Entity detail loading in the viewer should follow the same pattern as `shark get {id}`:
query entity details on demand as the user opens them. The viewer tree renders
top-level nodes on load; detail (including relationships) is fetched when a node
is expanded or selected, using the same service calls the CLI uses.

The only legitimate case for a dedicated aggregate query is the **dashboard** — project-
wide counts, progress rollups, blocked task totals — where a wide scan is genuinely
needed. That is a bounded, explicit use case and does not justify a bespoke bulk adapter
for the full relationship graph.

## Scope

**Task 001 — Remove legacy table references**
- Delete or replace `internal/repository/task/relationship.go`,
  `internal/repository/feature/relationship.go`,
  `internal/repository/epic/relationship.go` with calls to the entityrel repo
- Remove dual-path fallbacks in `internal/repository/task/dependency.go`
  and `internal/config/template/helpers.go`
- Remove legacy models (`task_relationship.go`, `feature_relationship.go`,
  `epic_relationship.go`) once callers are gone
- Add DROP TABLE migration for the three legacy tables

**Task 002 — Replace viewer adapter layer**
- Delete `taskRelAdapter` and `ViewerTaskRelationshipRepository` interface
- Wire `EntityRelationshipService` into `ViewerService` for per-entity on-demand calls
- Remove `ViewerTaskRelationship` struct; use `models.EntityRelationship` directly
- Update `ViewerTask` relationship fields to carry entity type + key so cross-entity
  links (task→bug, task→feature) are visible in the hierarchy

## Design Decisions

**Do not pre-load the full relationship graph on hierarchy load.** Load relationships
per entity when that entity is requested, matching `shark get {id}` behaviour.

**Dashboard aggregates are the exception.** If the dashboard needs project-wide
relationship counts or dependency chain stats, those warrant a dedicated query — but
scoped to the dashboard handler, not the general hierarchy service.

## Acceptance Criteria

- No production code references `task_relationships`, `feature_relationships`, or
  `epic_relationships` (db.go DROP migration excepted)
- `taskRelAdapter`, `ViewerTaskRelationshipRepository`, and `ViewerTaskRelationship`
  are deleted
- `shark link add <task> <bug>` round-trips correctly and appears in viewer detail
- `make fmt && make lint && make test` passes clean
