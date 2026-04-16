---
feature_key: E07-F39
epic_key: E07
title: Remove legacy relationship tables and dual-path query code
type: combined-spec
---

# E07-F39 — Combined Requirements + Architecture Spec

## 1. Context

See `feature.md` for problem statement and architectural intent. This spec
translates that intent into testable requirements and a concrete change
plan. Epic-level business context is in `docs/plan/E07-enhancements/`
(see `E07-ARCHITECTURE-REVIEW.md` and the relationship work in E21-F11).

**Prior work this builds on:**
- E21-F11 introduced `entity_relationships` as the canonical polymorphic
  store and migrated data from the three legacy tables into it.
- `internal/repository/entityrel/repository.go` already exposes the full
  read/write surface and ships an `EntityRelTaskKeyAdapter` to cover the
  one remaining template-helper call shape.
- `EntityRelationshipService` + `EntityRegistry` (used by
  `shark link list` — see `internal/cli/commands/link.go`) is the
  reference pattern for cross-entity key resolution.

The three legacy tables (`task_relationships`, `feature_relationships`,
`epic_relationships`) still have active code paths. Writes can drift to
stale tables, reads miss cross-entity links, and the viewer hierarchy
carries a bespoke bulk-load adapter that bypasses the canonical service
entirely.

---

## 2. Requirements (Incremental over epic)

### 2.1 Functional Requirements

| ID | Requirement |
|---|---|
| **REQ-F-001** | No production Go code outside `internal/db/db.go` may reference the identifiers `task_relationships`, `feature_relationships`, or `epic_relationships`. Only the DROP TABLE migration and any pre-existing data-migration code in `db.go` / `migrate.go` may. |
| **REQ-F-002** | All task-to-task dependency reads (blockers, blocked-by, dependency trees) must query `entity_relationships` via `EntityRelationshipService` or the existing `entityrel` repository / adapter. No dual-path fallbacks to legacy tables. |
| **REQ-F-003** | Template helpers (`related_tasks`, `related_features`, `related_epics` in `internal/config/template/helpers.go`) must source data from `entity_relationships` (using `EntityRelTaskKeyAdapter` or an equivalent adapter for features/epics) with no fallback to legacy tables. |
| **REQ-F-004** | The three legacy tables must be dropped via a new numbered migration in `internal/db/db.go`, and `CurrentSchemaVersion` must be incremented (11 → 12). The migration is idempotent (`DROP TABLE IF EXISTS`). |
| **REQ-F-005** | The legacy models `internal/models/task_relationship.go`, `feature_relationship.go`, `epic_relationship.go` must be deleted once all callers are removed. `models.EntityRelationship` is the sole in-memory relationship type. |
| **REQ-F-006** | The viewer's `Hierarchy()` and `FeatureTasks()` paths must load relationships on demand via `EntityRelationshipService.GetRelationships(ctx, EntityTypeTask, id)` per task — matching the `shark get {id}` behaviour. No project-wide bulk relationship pre-load. |
| **REQ-F-007** | `taskRelAdapter` (in `internal/viewer/server/wire.go`), `ViewerTaskRelationshipRepository` (interface in `internal/services/viewer_service.go`), `ViewerTaskRelationship` (struct, same file), and `ViewerService.taskRelRepo` / `WithTaskRelRepo(...)` must all be deleted. No substitutes reintroducing project-wide pre-load. |
| **REQ-F-008** | `ViewerTask` must expose relationships that preserve the other entity's type **and** key, so task→bug, task→feature, task→epic links render in the viewer UI. The existing `DependsOn` / `BlockedBy` / `Blocks []string` fields must be replaced (or augmented — see REQ-N-003) with a `Relationships []ViewerRelatedEntity` field carrying direction, relationship type, entity type, and key. |
| **REQ-F-009** | `shark link add <task> <bug> --type=related_to` (and the equivalent task→feature, task→epic variants) must round-trip through the viewer: running the CLI command then loading the viewer hierarchy/detail must include the linked entity. |
| **REQ-F-010** | `internal/cli/commands/migrate_relationships.go` must be removed (legacy tables no longer exist to migrate from) OR explicitly gated behind a "no-op on current schema" check that exits cleanly when legacy tables are absent. The preferred resolution is removal. |

### 2.2 Non-Functional Requirements

| ID | Requirement |
|---|---|
| **REQ-N-001** | `make fmt && make lint && make test` passes clean across all packages. No new lint warnings, no skipped/disabled tests. |
| **REQ-N-002** | `grep -r "task_relationships\|feature_relationships\|epic_relationships" internal/ --include="*.go"` returns only matches inside `internal/db/db.go` and `internal/db/migrate.go`. |
| **REQ-N-003** | Backwards compatibility for existing JSON consumers of `ViewerTask`: if `DependsOn` / `BlockedBy` / `Blocks` JSON fields are kept for a transition period, they must be populated correctly from `entity_relationships` (task→task only). The new `Relationships` field is the source of truth for cross-entity links. Decision flagged as open in §6. |
| **REQ-N-004** | Per-task relationship load adds ≤1 `EntityRelationshipService.GetRelationships` call per task in `Hierarchy()` / `FeatureTasks()`. For a project with N tasks, this is O(N) queries, matching the pre-existing per-task detail cost elsewhere in the viewer. Acceptable — see feature.md §"Intended Architecture". |
| **REQ-N-005** | The DROP migration must not fire on databases that pre-date `entity_relationships` having data. Current production databases have already run the E21-F11 data migration; the pre-drop safety check is therefore a version-guard (`CurrentSchemaVersion`), not a data-integrity check. No additional data validation is required. |

### 2.3 Acceptance Criteria (testable)

| AC | Criterion | How Verified |
|---|---|---|
| **AC-1** | Legacy identifiers confined to `db.go` / `migrate.go` | `grep -r` per REQ-N-002, verified in CI |
| **AC-2** | `shark task deps E07-F01-001` works end-to-end | Existing `task deps` CLI test + manual run |
| **AC-3** | `shark task link` / `shark task unlink` round-trip via `entity_relationships` | Existing `task_relationship_commands_test.go` updated to use new path |
| **AC-4** | Three legacy tables absent after migration | Add test: `SELECT name FROM sqlite_master WHERE type='table' AND name IN (...)` returns empty |
| **AC-5** | Viewer hierarchy JSON for a task includes links to bugs / features / epics | New integration test in `internal/services/viewer_service_test.go`: create task→bug link, assert appears in `Hierarchy()` response |
| **AC-6** | `taskRelAdapter`, `ViewerTaskRelationshipRepository`, `ViewerTaskRelationship`, `WithTaskRelRepo` symbols absent from the codebase | `grep -r` assertion + compile-time verification |
| **AC-7** | `make fmt && make lint && make test` clean | CI gate |
| **AC-8** | `CurrentSchemaVersion` incremented to 12; migration idempotent on re-run | Unit test calling `runMigrations` twice against the same DB |

### 2.4 Out of Scope

- Introducing new relationship **types** (e.g., blocks-cross-entity semantics). This feature only relocates the existing `depends_on` / `blocks` / `related_to` semantics to the canonical store.
- Dashboard aggregate queries for project-wide relationship statistics. Dedicated handler work, tracked separately (see feature.md §"Design Decisions").
- Reworking `task_auto_unblock` semantics or the auto-unblock trigger. Those already target `entity_relationships` post-E21-F11; this feature does not touch that logic.
- CLI UX changes to `shark link`. Command surface is stable — only underlying repo paths move.
- Removing `migrate_relationships.go` test fixtures from `internal/repository/` (the table-level relationship tests `relationship_repositories_test.go`, `task_relationship_repository_test.go`) if they exercise the legacy repo interfaces — they will be deleted alongside the repositories.

---

## 3. Architecture

### 3.1 Component Changes

#### 3.1.1 Repository Layer — Delete/Replace

| File | Action | Rationale |
|---|---|---|
| `internal/repository/task/relationship.go` (439 lines) | **Delete** | All callers must switch to `entityrel.EntityRelationshipRepository`. No shim is added because `EntityRelationshipService` is the recommended caller-facing API. |
| `internal/repository/task/relationship_test.go` | **Delete** | Tests the file being deleted. |
| `internal/repository/feature/relationship.go` (200 lines) | **Delete** | Same as above for features. |
| `internal/repository/epic/relationship.go` (200 lines) | **Delete** | Same as above for epics. |
| `internal/repository/task/dependency.go` | **Modify** — remove dual-path fallbacks at lines 311–319 and 467–475 (per grep). Replace with `entityrel` queries using the existing `EntityRelTaskKeyAdapter` pattern, or delegate to `EntityRelationshipService`. | Follows pattern in `internal/repository/entityrel/repository.go` (`EntityRelTaskKeyAdapter.ListRelatedTaskKeys`). |
| `internal/repository/task_relationship_repository_test.go` | **Delete** | Tests legacy repo. |
| `internal/repository/relationship_repositories_test.go` | **Delete or rewrite** against `entityrel` if coverage gap exists after deletion. |

#### 3.1.2 Config/Template Helpers — Modify

`internal/config/template/helpers.go` (752 lines). Three call sites need replacement (lines 316, 425, 468, 503 per grep). The existing `EntityRelTaskKeyAdapter` already serves `related_tasks`. Add analogous adapters for feature/epic relationships:

- **New**: `EntityRelFeatureKeyAdapter.ListRelatedFeatureKeys(ctx, featureID int64) ([]string, error)` in `internal/repository/entityrel/repository.go`
- **New**: `EntityRelEpicKeyAdapter.ListRelatedEpicKeys(ctx, epicID int64) ([]string, error)` in `internal/repository/entityrel/repository.go`

Both mirror the `EntityRelTaskKeyAdapter` shape at `repository.go:247–293`, selecting `DISTINCT key` from `entity_relationships JOIN features` / `JOIN epics` with matching `from_entity_type` / `to_entity_type`.

Wire the new adapters into whatever interface `helpers.go` currently uses for feature/epic relationships (same pattern as the existing `TaskRelationshipRepository` interface consumed by the template package).

#### 3.1.3 Models — Delete

| File | Action |
|---|---|
| `internal/models/task_relationship.go` | Delete (after callers gone) |
| `internal/models/feature_relationship.go` | Delete (after callers gone) |
| `internal/models/epic_relationship.go` | Delete (after callers gone) |

`models.EntityRelationship` (in `internal/models/entity_relationship.go`) is the sole remaining relationship model.

#### 3.1.4 Viewer Service — Restructure

`internal/services/viewer_service.go` (1715 lines):

- **Delete** `ViewerTaskRelationship` struct (lines 308–316)
- **Delete** `ViewerTaskRelationshipRepository` interface (lines 318–323)
- **Delete** `taskRelRepo` field on `ViewerService` (line 410)
- **Delete** `WithTaskRelRepo` setter (lines 486–489 area)
- **Modify** `ViewerTask` struct (lines 297–306): replace `DependsOn`/`BlockedBy`/`Blocks []string` with the new `Relationships []ViewerRelatedEntity` field. See §3.2 for the type.
- **Modify** `Hierarchy()` — remove bulk relationship map building (referenced at lines 837–838); instead, for each task in the final result set, call `entityRelSvc.GetRelationships(ctx, models.EntityTypeTask, t.ID)` and populate `ViewerTask.Relationships`.
- **Modify** `FeatureTasks()` — same pattern, replacing the `ftBlockedByKeys` map-building block (lines 1480–1508).
- **Add** `entityRelSvc *services.EntityRelationshipService` field (required, not optional — no graceful-degrade behaviour).
- **Add** `entityRegistry EntityRegistry` field (required) for resolving `(entity_type, entity_id) → key`.
- **Modify** `ViewerService` constructor signature to accept `entityRelSvc` and `entityRegistry`.

#### 3.1.5 Viewer Wiring — Modify

`internal/viewer/server/wire.go` (373 lines):

- **Delete** `taskRelAdapter` struct and its `ListAll()` method.
- **Delete** the `WithTaskRelRepo(...)` call on `ViewerService`.
- **Add** construction of `EntityRelationshipService` and pass it (plus the existing `EntityRegistry`) into the `ViewerService` constructor.
- Reference: the CLI accessor pattern `cli.GetEntityRelationshipService()` in `internal/cli/commands/link.go:222`.

#### 3.1.6 CLI Migration Command — Remove

`internal/cli/commands/migrate_relationships.go` — **Delete**. After the DROP migration runs, the legacy-table migrator has nothing to do. If any test depends on this command, remove it.

### 3.2 Data Model Changes

#### 3.2.1 Schema Migration

Add a new migration in `internal/db/db.go` (or the dedicated function in `internal/db/migrate.go`):

```go
// migrateDropLegacyRelationshipTables drops the three legacy relationship
// tables after E21-F11 migrated their data into entity_relationships and
// E07-F39 removed all remaining read/write paths.
func migrateDropLegacyRelationshipTables(db *sql.DB) error {
    stmts := []string{
        `DROP TABLE IF EXISTS task_relationships`,
        `DROP TABLE IF EXISTS feature_relationships`,
        `DROP TABLE IF EXISTS epic_relationships`,
    }
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return fmt.Errorf("drop legacy relationship tables: %w", err)
        }
    }
    return nil
}
```

Call from `runMigrations` after the existing E21-F11 data-migration step.

Bump `CurrentSchemaVersion` from 11 → **12** (per `.claude/rules/database/schema.md` migration checklist, and `internal/db/db.go:438`).

**Developer instruction (mandatory per database-critical.md):** After merging, set `skip_migrations: false` in `.sharkconfig.json`, run one shark command to apply the migration, then set it back to `true`.

#### 3.2.2 In-Memory Types

New struct in `internal/services/viewer_service.go`:

```go
// ViewerRelatedEntity represents a single relationship edge from a task's
// perspective, including the other entity's type and key so cross-entity
// links render correctly in the viewer UI.
type ViewerRelatedEntity struct {
    Direction        string                        `json:"direction"` // "outgoing" | "incoming"
    RelationshipType models.EntityRelationshipType `json:"relationship_type"`
    EntityType       models.EntityType             `json:"entity_type"`
    EntityKey        string                        `json:"entity_key"`
}
```

This mirrors the shape of `linksOutputEntry` in `internal/cli/commands/link.go:203–210` for consistency.

### 3.3 API/Interface Contracts

No external HTTP API surface changes beyond the `ViewerTask` JSON shape (internal viewer endpoint). Viewer consumers must treat:
- **Removed**: `depends_on_keys`, `blocked_by_keys`, `blocks_keys` (subject to REQ-N-003 transition decision).
- **Added**: `relationships` — array of `ViewerRelatedEntity`.

All CLI command surfaces (`shark link add/remove/list`, `shark task deps`, `shark task link/unlink`) remain unchanged at the argument/output level.

### 3.4 Key Technical Decisions

| Decision | Rationale |
|---|---|
| **Delete legacy repos outright (no shim layer)** | `EntityRelationshipService` already exposes the full API surface. A thin shim would add maintenance burden without value. All in-tree callers can migrate directly. |
| **Add per-task `GetRelationships` calls in `Hierarchy()` instead of preserving a bulk loader** | Premature optimization per feature.md. The hierarchy endpoint already executes O(N) queries for task detail; one more per task is within the same order. Avoids the bug surface of raw SQL bypassing service-level logic. |
| **Drop tables in this feature rather than deferring** | `skip_migrations: true` is the default on Turso. Leaving the tables with no code path means they are dead weight on every machine and a footgun for any agent grepping for relationship schema. |
| **Require `entityRelSvc` in ViewerService (not optional)** | The existing `taskRelRepo` was optional "for graceful degradation" — but there's no production scenario where we want a viewer running without relationships. Making it required removes a branch and a nil-check path. |
| **Replace `DependsOn`/`BlockedBy`/`Blocks` with a single `Relationships` field rather than keeping three typed slices** | The three-slice layout hardcodes an assumption that relationships are always task→task and always one of two types. The single polymorphic list preserves all information and matches the canonical `entity_relationships` shape. If the frontend needs the filtered views, it can derive them client-side. |
| **Keep `migrate_relationships.go` deletion in-scope rather than deferring** | The command has no purpose after this feature lands; leaving it would require it to gracefully no-op, which is extra code for no user value. |

### 3.5 Integration with Existing Code

Exact call sites (file:line, behaviour):

| Current Call Site | Replacement |
|---|---|
| `internal/repository/task/dependency.go:314` — `SELECT from_task_id FROM task_relationships ...` | Delegate to `entityRelRepo.GetIncoming(ctx, EntityTypeTask, taskID, []EntityRelationshipType{RelTypeDependsOn})` |
| `internal/repository/task/dependency.go:469` — `SELECT t.status FROM task_relationships tr ...` | Delegate to `entityRelRepo.GetOutgoing(...)` then join to task status via `taskRepo.GetByID` per relationship |
| `internal/config/template/helpers.go:316,425,468,503` | Use `EntityRelTaskKeyAdapter` (already exists) for tasks; add `EntityRelFeatureKeyAdapter` and `EntityRelEpicKeyAdapter` for features/epics |
| `internal/services/viewer_service.go:837–838, 1480–1508` | Replace bulk map build with per-task `entityRelSvc.GetRelationships` calls |
| `internal/viewer/server/wire.go` `taskRelAdapter.ListAll` | Delete; inject `EntityRelationshipService` into `ViewerService` |

Follows pattern in `internal/cli/commands/link.go:runLinks` (service + registry for cross-entity key resolution).

Follows service-design rules in `.claude/rules/services/service-design.md`: "fat services, thin controllers, dumb repositories." All relationship business logic stays in `EntityRelationshipService`; viewer calls the service, does not query the DB directly.

### 3.6 Test Plan

| Test File | Change |
|---|---|
| `internal/repository/task/relationship_test.go` | Delete (tests deleted file) |
| `internal/repository/task_relationship_repository_test.go` | Delete |
| `internal/repository/relationship_repositories_test.go` | Delete or rewrite against `entityrel` |
| `internal/repository/task/dependency_test.go` | Update fixtures to seed `entity_relationships` instead of `task_relationships`. Coverage for `depends_on`/`blocks` unchanged. |
| `internal/repository/template_helpers_integration_test.go` | Update fixtures. Assert `related_tasks`/`related_features`/`related_epics` still resolve via `entity_relationships`. |
| `internal/repository/task_auto_unblock_test.go` | Already targets `entity_relationships` post-E21-F11. Re-run to confirm no regression from dependency.go changes. |
| `internal/cli/commands/task_relationship_commands_test.go` | Update to use `EntityRelationshipService` mocks; no-op if already migrated. |
| `internal/cli/commands/task_get_blocking_test.go`, `task_get_blocking_integration_test.go` | Re-run; update fixtures only if they seed legacy tables directly. |
| `internal/services/viewer_service_test.go` | **Add** test: given task linked to a bug via `entity_relationships`, `Hierarchy()` response for that task includes the bug in `Relationships`. **Add** test: given task→task `depends_on`, same. **Delete** tests covering the deleted `taskRelRepo` injection path. |
| **New** integration test | Run `make fmt && make lint && make test`. Part of CI, gated by AC-7. |
| **New** schema test | In `internal/db/db_test.go` (or similar): after migration, the three legacy tables do not exist (AC-4, AC-8). |

---

## 4. Implementation Order

Task 001 (legacy table references) is a prerequisite for Task 002 (viewer adapter) only indirectly: Task 002's viewer changes require the `EntityRelationshipService` to be the sole relationship path, which 001 establishes. Task 001 is also large enough to be done on its own.

Within Task 001, work in this order to keep the tree compilable at each step:

1. Add `EntityRelFeatureKeyAdapter` and `EntityRelEpicKeyAdapter` in `internal/repository/entityrel/repository.go`
2. Update `internal/config/template/helpers.go` to use the new adapters
3. Remove dual-path fallbacks in `internal/repository/task/dependency.go`
4. Delete the three legacy repo files + their tests
5. Delete the three legacy model files
6. Delete `internal/cli/commands/migrate_relationships.go`
7. Add DROP migration + bump `CurrentSchemaVersion`
8. Run `make fmt && make lint && make test`

Task 002 depends on Task 001 being merged (schema + legacy-repo removal).

---

## 5. Risks

| Risk | Mitigation |
|---|---|
| Hidden caller outside `internal/` (e.g., `cmd/` or external tooling) references a deleted repo | Grep the full tree (`cmd/`, `tools/`) during Task 001 before deletion. Not just `internal/`. |
| Template helper regression on feature/epic `related_*` output if new adapters diverge from legacy query shape | Parity test: compare `related_tasks`/`related_features`/`related_epics` output on a seeded DB before and after changes. |
| Viewer performance regression on large projects due to O(N) per-task `GetRelationships` calls | Benchmark `Hierarchy()` before/after on a seeded DB with ~500 tasks. Per feature.md, this is accepted; but measure to confirm no pathological query cost. |
| Breaking viewer frontend consumers that rely on `depends_on_keys` / `blocked_by_keys` / `blocks_keys` JSON fields | Resolve REQ-N-003 with the viewer frontend maintainer before Task 002 merge. Default: keep the old fields populated for one release, add `relationships`, flag the old fields as deprecated in JSON comments. |
| `skip_migrations: true` default on Turso causes the DROP migration to never run | Covered by the version-guard bump (`CurrentSchemaVersion` 11 → 12). Developer instruction to temporarily flip the flag is in §3.2.1. |

---

## 6. Open Questions

| # | Question | Resolution Owner |
|---|---|---|
| Q1 | REQ-N-003: keep `DependsOn`/`BlockedBy`/`Blocks` JSON fields for one release as deprecated, or replace cleanly with `Relationships` in this feature? | Viewer frontend owner, before Task 002 implementation |

No other TBDs. All file paths, function signatures, and test changes are enumerated above.

---

## 7. Exit Gate Checklist

- [x] Every requirement has a test and an acceptance criterion
- [x] Every architecture decision references an existing pattern (entityrel, shark link list, service design rules) or explicitly calls out a deviation with rationale (§3.4)
- [x] File paths listed for all code modifications (§3.1, §3.5, §3.6)
- [x] Non-critical open questions isolated to §6; none block Task 001 start
- [x] Migration version bump and developer flag-flip instruction documented (§3.2.1)
