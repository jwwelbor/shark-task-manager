# E21-F11: Polymorphic Entity Relationships — Research Report

## Executive Summary

The codebase currently has **four distinct, inconsistent** mechanisms for expressing entity relationships. The task_relationships system is the most mature (typed, cycle-detected, indexed), but it is task-only. The epic_relationships and feature_relationships tables replicate its schema for their own entity types, producing ~1,000 lines of near-identical code. Bugs use flat columns (`linked_entity_type` / `linked_entity_key`) on the `bugs` table itself. Change-cards use direct foreign-key columns (`epic_id`, `feature_id`, `related_task_id`). A legacy `depends_on` JSON text column on `tasks` partially overlaps with `task_relationships`. Unifying these into a single `entity_relationships` table is architecturally sound and will eliminate approximately 800–900 lines of duplicated repository code, at the cost of a careful migration and moderate test churn.

---

## Research Questions

1. What is the exact schema and business logic of each relationship pattern?
2. How many files and lines of code are affected?
3. What is the overlap/deduplication risk between the `depends_on` column and `task_relationships`?
4. What service and CLI code would need to change?
5. What is the migration complexity and risk?

---

## Methodology

- Read all four repository implementations and their model definitions
- Read the task dependency service and the task_dependency helper file
- Grepped for all `depends_on`, `LinkedEntityType`, `LinkedEntityKey`, `EpicID`, `FeatureID`, `RelatedTaskID` usages
- Counted lines in all directly affected files
- Reviewed DB schema in `internal/db/db.go`

---

## Findings

### Finding 1: Three Type-Specific Relationship Tables (Near-Identical Code)

**Summary:** The codebase has three separate SQLite tables and three separate repository structs, all storing directed typed relationships between entities of the same kind.

| Table | Repository | Model | Lines (Repo) | Lines (Model) |
|---|---|---|---|---|
| `task_relationships` | `TaskRelationshipRepository` | `TaskRelationship` | 438 | 57 |
| `feature_relationships` | `FeatureRelationshipRepository` | `FeatureRelationship` | 199 | 31 |
| `epic_relationships` | `EpicRelationshipRepository` | `EpicRelationship` | 199 | 31 |

**Schema (all three are structurally identical):**

```sql
-- task_relationships example
CREATE TABLE task_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id INTEGER NOT NULL REFERENCES tasks(id),
    to_task_id INTEGER NOT NULL REFERENCES tasks(id),
    relationship_type TEXT NOT NULL CHECK(relationship_type IN (
        'depends_on', 'blocks', 'related_to', 'follows',
        'spawned_from', 'duplicates', 'references'
    )),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_task_id, to_task_id, relationship_type)
);
-- Same pattern for feature_relationships and epic_relationships
```

**Difference:** `task_relationships` adds `DetectCycle()` (DFS cycle detection, ~75 lines), `GetOutgoing()`, `GetIncoming()`, and `ListRelatedTaskKeys()`. The feature and epic variants only have `Create`, `GetByID`, `ListRelated*`, `Delete`, `DeleteBy*`, and `GetRelated*Keys`. No cycle detection exists for feature/epic relationships.

**Implications:** All three tables could be replaced by a single `entity_relationships` table with `from_entity_type`, `from_entity_id`, `to_entity_type`, `to_entity_id`, `relationship_type`. Cycle detection would become a cross-entity capability.

---

### Finding 2: Bug Linking — Flat Columns on the bugs Table

**Summary:** Bugs link to any single entity via two optional text columns directly on the `bugs` row.

**Schema:**
```sql
-- bugs table (relevant columns only)
linked_entity_type TEXT,   -- "epic", "feature", "task" (or NULL)
linked_entity_key  TEXT,   -- e.g., "E07", "E07-F01", "E07-F01-001" (or NULL)
```

**Model (internal/models/bug.go):**
```go
LinkedEntityType *string  `json:"linked_entity_type,omitempty" db:"linked_entity_type"`
LinkedEntityKey  *string  `json:"linked_entity_key,omitempty" db:"linked_entity_key"`
```

**Behavior:**
- Single link only (a bug can link to at most one entity at a time)
- No referential integrity (text key, not a foreign key)
- The type tag drives display/lookup logic but has no DB constraint
- 14 files reference `LinkedEntityType` or `LinkedEntityKey`

**Key consumers:**
- `internal/repository/bug_repository.go` (353 lines) — stores/retrieves columns
- `internal/services/bug_service.go` (471 lines) — creates/updates bugs with link
- `internal/cli/commands/bug.go` (480 lines) — parses `--linked-to` flag
- `internal/services/bug_dto.go` — CreateBugInput has `LinkedEntityType` / `LinkedEntityKey`
- `internal/config/template_helpers.go` — template uses linked entity fields

**Implications:** Migration to `entity_relationships` would allow bugs to link to multiple entities. However, the existing single-link pattern is a deliberate design choice and moving it requires updating 14 files. The `bug_aggregate.go` file (192 lines) builds composite bug views that include linked entity data.

---

### Finding 3: Change-Card Linking — Direct Foreign Key Columns

**Summary:** Change-cards link to epics, features, and tasks via three separate nullable integer foreign-key columns on the `change_cards` row.

**Schema:**
```sql
-- change_cards table (relevant columns only)
epic_id         INTEGER REFERENCES epics(id),
feature_id      INTEGER REFERENCES features(id),
related_task_id INTEGER REFERENCES tasks(id),
```

**Model (internal/models/change_card.go):**
```go
EpicID        *int64  `json:"epic_id,omitempty" db:"epic_id"`
FeatureID     *int64  `json:"feature_id,omitempty" db:"feature_id"`
RelatedTaskID *int64  `json:"related_task_id,omitempty" db:"related_task_id"`
```

**Behavior:**
- Can link to at most one of each entity type simultaneously
- Referential integrity enforced via SQLite foreign keys
- `ChangeCardRepository.ListByEpic()` and `ListByFeature()` use these columns for filtering
- The `ChangeCardRepoFilter` struct has `EpicID *int64` and `FeatureID *int64` filter fields

**Key consumers:**
- `internal/repository/change_card_repository.go` (393 lines) — all CRUD includes these cols
- `internal/services/change_card_service.go` (443 lines) — resolves keys to IDs
- `internal/cli/commands/change.go` (479 lines) — parses `--epic`, `--feature`, `--task` flags

**Implications:** This is the most tightly coupled pattern. The `List` query filters on `epic_id` and `feature_id` directly. Migrating to `entity_relationships` would require join-based filtering, breaking these simple WHERE clauses. This is the highest-risk migration component.

---

### Finding 4: Legacy `depends_on` JSON Column — Partially Superseded

**Summary:** The `tasks` table has a `depends_on TEXT` column storing a JSON array of task key strings (e.g., `["E07-F01-001", "E07-F01-002"]`). This predates the `task_relationships` table.

**Evidence of dual-system awareness:**
- `task_dependency.go:getTaskDependentsInTx()` comment: "checks both the legacy depends_on JSON field and the task_relationships table"
- `task_auto_unblock_test.go` has `TestAutoUnblock_MixedDependencies_LegacyAndRelationships` (line 677) — tests both systems together
- `GetTaskDependents()` reads only from `depends_on` column (not from `task_relationships`)
- `task_dependency_service.go:AddDependency()` writes to `task_relationships` (not `depends_on`)

**Current state:** Both systems coexist. New dependencies created via CLI use `task_relationships`. Old tasks may have data only in `depends_on`. The auto-unblock logic merges both sources in transactions.

**Code volume:**
- `task_dependency.go`: 501 lines — handles both legacy and new patterns
- Validation in `models/validation.go`: `ValidateDependsOn()` validates JSON format

**Implications:** This is a deduplication opportunity **within** the existing task system, independent of cross-entity polymorphism. The `depends_on` column should be drained to `task_relationships` before or as part of the E21-F11 migration. Failing to do so means the new `entity_relationships` table would still need to read both sources.

---

### Finding 5: TaskDependencyService — Service-Layer Dependency Management

**Summary:** `internal/services/task_dependency_service.go` (511 lines) is the service-layer owner for task-to-task relationships. It uses two repository interfaces: `TaskDependencyRepository` (writable) and `TaskRelationshipQueryRepository` (readable).

**Key interfaces defined in services package:**
- `TaskDependencyRepository` — Create, Delete, DetectCycle, GetOutgoing
- `TaskRelationshipQueryRepository` — GetByTaskID, GetOutgoing, GetIncoming

**Cycle detection:** Two implementations exist:
1. In `task_relationship_repository.go` (`DetectCycle` method, DFS, ~75 lines)
2. In `task_dependency_service.go` (`detectCircularDependency`, DFS, ~25 lines)

These are redundant implementations of the same algorithm. The repository-level one uses IDs; the service-level one uses key strings via `GetTaskDependents`.

**Implications:** In a polymorphic world, cycle detection would need to operate on the unified `entity_relationships` table using entity type + ID pairs. The service-level cycle detection (key-based) is easier to generalize than the repository-level one (ID-based).

---

## File Inventory

### Files Requiring New Creation

| File | Purpose | Estimated Lines |
|---|---|---|
| `internal/models/entity_relationship.go` | Polymorphic model, typed enum | ~60 |
| `internal/repository/entity_relationship_repository.go` | Unified CRUD + cycle detection | ~400 |
| `internal/services/entity_relationship_service.go` | Business logic for all entity linking | ~250 |
| DB migration in `internal/db/db.go` | `entity_relationships` table + indexes | ~60 |

### Files Requiring Significant Modification

| File | Current Lines | Change Required |
|---|---|---|
| `internal/repository/task_dependency.go` | 501 | Rewrite `GetTaskDependents` to use new table; remove JSON-parsing of `depends_on` |
| `internal/repository/task_repository.go` | 1,829 | Remove `depends_on` from all 15+ SELECT/INSERT/UPDATE queries |
| `internal/models/task.go` | ~70 | Remove `DependsOn *string` field |
| `internal/models/validation.go` | ~220 | Remove `ValidateDependsOn`, `ErrInvalidDependsOn` |
| `internal/services/task_dependency_service.go` | 511 | Update `AddDependency`, `RemoveDependency`, `ValidateDependencies` to use new repo |
| `internal/repository/bug_repository.go` | 353 | Remove `linked_entity_type`/`linked_entity_key` columns from CRUD |
| `internal/repository/bug_aggregate.go` | 192 | Update aggregate queries for new join-based link lookup |
| `internal/services/bug_service.go` | 471 | Update CreateBugInput handling; use EntityRelationshipService |
| `internal/services/bug_dto.go` | ~50 | Remove `LinkedEntityType`/`LinkedEntityKey` from DTO |
| `internal/cli/commands/bug.go` | 480 | Update `--linked-to` flag handling |
| `internal/repository/change_card_repository.go` | 393 | Remove `epic_id`/`feature_id`/`related_task_id` from CRUD; update `ListByEpic`/`ListByFeature` |
| `internal/services/change_card_service.go` | 443 | Update filtering and creation to use EntityRelationshipService |
| `internal/cli/commands/change.go` | 479 | Update `--epic`/`--feature`/`--task` flag handling |
| `internal/config/template_helpers.go` | ~200 | Update bug template rendering for linked entity |
| `internal/db/db.go` | 2,866 | Add migration, update schema sections |

### Files to Be Deprecated/Deleted

| File | Lines | Replacement |
|---|---|---|
| `internal/repository/task_relationship_repository.go` | 438 | `entity_relationship_repository.go` |
| `internal/repository/epic_relationship_repository.go` | 199 | `entity_relationship_repository.go` |
| `internal/repository/feature_relationship_repository.go` | 199 | `entity_relationship_repository.go` |
| `internal/models/task_relationship.go` | 57 | `entity_relationship.go` |
| `internal/models/epic_relationship.go` | 31 | `entity_relationship.go` |
| `internal/models/feature_relationship.go` | 31 | `entity_relationship.go` |

**Deleted code total: ~955 lines**

### Test Files Requiring Updates

| File | Lines |
|---|---|
| `internal/repository/task_relationship_repository_test.go` | 1,126 |
| `internal/repository/relationship_repositories_test.go` | 774 |
| `internal/repository/task_dependency.go`-related tests | ~1,500 (est.) |
| `internal/repository/bug_repository_test.go` | ~200 |
| `internal/repository/bug_aggregate_test.go` | ~200 |
| `internal/services/task_dependency_service` tests | ~300 |
| `internal/services/bug_service_test.go` | ~200 |
| `internal/services/change_card_service_test.go` | ~200 |

---

## Migration Complexity Assessment

### Phase 1: New Table + Task Relationship Migration (Medium)

1. Add `entity_relationships` table (migration in `db.go`)
2. Create `EntityRelationship` model and `EntityRelationshipRepository`
3. Migrate data from `task_relationships` to `entity_relationships` (data migration script)
4. Migrate data from `depends_on` JSON column to `entity_relationships` (parse JSON, insert rows)
5. Update `TaskRelationshipQueryRepository` / `TaskDependencyRepository` interfaces to point at new table
6. Update `task_dependency.go`'s `GetTaskDependents` and `AutoUnblockDependents`

**Risk:** The `depends_on` JSON migration needs careful deduplication — the same dependency may exist in both `task_relationships` and `depends_on`. A `SELECT DISTINCT` plus UNIQUE constraint handles this.

### Phase 2: Bug Linking Migration (Low-Medium)

1. Update `entity_relationships` to store bug links (from_entity_type='bug', to_entity_type=...)
2. Data migration: read `linked_entity_type`/`linked_entity_key` from each bug row, insert into `entity_relationships`
3. Update `BugRepository.Create/Update` to write to `entity_relationships` instead of columns
4. Update `bug_aggregate.go` to JOIN `entity_relationships`
5. Remove columns from `bugs` table (DROP COLUMN migration)

**Risk:** The bug aggregate queries that join the linked entity for display will need rethinking. Currently it is a simple column read; after migration it becomes a join. Some query performance testing is warranted.

### Phase 3: Change-Card FK Migration (High)

1. Data migration: convert `epic_id`, `feature_id`, `related_task_id` to rows in `entity_relationships`
2. Update `ChangeCardRepository.ListByEpic` and `ListByFeature` to use EXISTS subquery or JOIN on `entity_relationships`
3. Update `ChangeCardRepository.Create/Update` to write links after insert
4. Update `ChangeCardRepoFilter` to work with key-based rather than ID-based filtering
5. Remove FK columns from `change_cards` table

**Risk:** The `ListByEpic` / `ListByFeature` filtering is currently a trivial `WHERE epic_id = ?`. After migration, it becomes a JOIN or subquery. The query plan must be verified to use indexes. The FK cascade behavior (if any) is lost and must be handled at the service layer.

### Phase 4: Remove Deprecated Tables (Low, done last)

Drop `task_relationships`, `feature_relationships`, `epic_relationships` tables after all consumers are migrated and verified.

---

## Risk Areas

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| `depends_on` / `task_relationships` deduplication creates duplicate rows | Medium | High | Use UNIQUE constraint + INSERT OR IGNORE in migration |
| Change-card filter performance regression | Medium | Medium | Add composite index on `entity_relationships(from_entity_type, from_entity_id)` and `(to_entity_type, to_entity_id)` |
| Cycle detection breaks across entity types | Low | High | Scope cycle detection to same entity type pair; cross-type cycles are not semantically meaningful |
| Bug aggregate queries become N+1 | Low | Medium | Implement eager join in the new aggregate query |
| `task_dependency.go` auto-block logic uses both sources simultaneously (in-flight) | High | High | Phase the migration: keep `depends_on` reads until all data is in `entity_relationships`, then drop |
| DB schema version bump needed | High | High | Bump `CurrentSchemaVersion` in `db.go` and instruct developer to set `skip_migrations: false` |

---

## Recommendations

### 1. Implement in Phases, Not One PR

The four patterns are loosely coupled. Migrate in order:
- Task relationships first (most used, best tested)
- Bug linking second (single FK, simpler)
- Change-card FKs third (most complex query impact)

### 2. Retain `depends_on` Column Through Transition

Do not DROP the `depends_on` column until `task_dependency.go` is fully migrated. The auto-block/auto-unblock logic reads both sources in the same transaction and must remain consistent.

### 3. Polymorphic Table Design

```sql
CREATE TABLE entity_relationships (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    from_entity_type  TEXT NOT NULL CHECK(from_entity_type IN ('epic','feature','task','bug','change_card')),
    from_entity_id    INTEGER NOT NULL,
    to_entity_type    TEXT NOT NULL CHECK(to_entity_type IN ('epic','feature','task','bug','change_card')),
    to_entity_id      INTEGER NOT NULL,
    relationship_type TEXT NOT NULL CHECK(relationship_type IN (
        'depends_on','blocks','related_to','follows','spawned_from','duplicates','references','linked_to'
    )),
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
);
CREATE INDEX idx_er_from ON entity_relationships(from_entity_type, from_entity_id);
CREATE INDEX idx_er_to ON entity_relationships(to_entity_type, to_entity_id);
CREATE INDEX idx_er_type ON entity_relationships(relationship_type);
```

Note: No FK constraint on `from_entity_id`/`to_entity_id` — SQLite cannot enforce FKs to multiple tables from one column. Referential integrity must be enforced at the service/repository layer.

### 4. Preserve Cycle Detection for Task Dependencies Only

Cycle detection (DFS) is only semantically meaningful for `depends_on`/`blocks` between tasks. Implement it scoped to `entity_type='task'` pairs. Do not generalize to epic/feature/bug cross-type cycles.

### 5. Duplicate `linked_to` Relationship Type

Add `linked_to` as a relationship type for the bug single-link pattern. This preserves the semantic distinction between structured task dependencies and the bug's informal "context" link to any entity.

---

## Recommended Implementation Order (Tasks)

1. New `entity_relationships` table + model + repository (no migration yet)
2. Data migration script: `task_relationships` → `entity_relationships`
3. Data migration script: `depends_on` JSON → `entity_relationships` (with dedup)
4. Update `TaskDependencyRepository` interface to use new table
5. Update `task_dependency.go` auto-block/auto-unblock to use new table
6. Remove `depends_on` column from `tasks` after validation
7. Data migration: bug `linked_entity_*` → `entity_relationships`
8. Update `BugRepository` and `bug_aggregate.go`
9. Data migration: change-card FKs → `entity_relationships`
10. Update `ChangeCardRepository` List queries
11. Remove old tables: `task_relationships`, `feature_relationships`, `epic_relationships`
12. Update all tests

---

## References

- `internal/repository/task_relationship_repository.go` — 438 lines, cycle detection, typed links
- `internal/repository/epic_relationship_repository.go` — 199 lines, near-duplicate of above
- `internal/repository/feature_relationship_repository.go` — 199 lines, near-duplicate of above
- `internal/repository/task_dependency.go` — 501 lines, dual-source (depends_on + task_relationships)
- `internal/services/task_dependency_service.go` — 511 lines, service orchestration
- `internal/repository/bug_repository.go` — 353 lines, flat-column link pattern
- `internal/repository/change_card_repository.go` — 393 lines, FK-column link pattern
- `internal/db/db.go` — 2,866 lines, schema definitions and migrations
- `internal/models/task_relationship.go`, `epic_relationship.go`, `feature_relationship.go` — 119 lines combined
- `internal/repository/relationship_repositories_test.go` — 774 lines
- `internal/repository/task_relationship_repository_test.go` — 1,126 lines
