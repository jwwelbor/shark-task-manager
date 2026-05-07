---
feature_key: E19-F01-sprint-database-schema-core-entity-foundation
epic_key: E19
title: Sprint Database Schema & Core Entity Foundation
spec_version: 1.0
last_updated: 2026-05-05
complexity: STANDARD
---

# Spec — E19-F01: Sprint Database Schema & Core Entity Foundation

> **Scope guard.** This feature delivers the *foundational data layer* only: three new tables, one schema migration, S### key parsing, and the `Sprint` model with structural validation. **Out of scope here:** sprint CLI commands, sprint services, lifecycle transitions, capacity allocation queries, analytics. Those live in E19-F02 through E19-F05.

---

## 1. Requirements (Mapped)

This feature satisfies two requirements from [E19 requirements.md](../requirements.md):

| Req | Title | What this feature delivers |
|---|---|---|
| **REQ-F-001** | Sprint Creation | The DB-level pre-conditions: a `sprints` table with auto-incrementing `S###` keys, columns for name/dates/goal/status/slug/file_path, and the model+validators that the future `shark sprint create` command (E19-F02) will write through. The CLI command itself is E19-F02. |
| **REQ-F-010** | Sprint Database Tables | The complete schema: `sprints`, `sprint_assignments` (polymorphic on entity_type), `sprint_capacity`, all required indexes, the partial unique index for one-active-sprint-per-entity, the `CurrentSchemaVersion` bump from 17 → 18, and idempotent migration. |

REQ-F-002 through REQ-F-009 and REQ-F-011 onward depend on this feature but are implemented downstream.

---

## 2. Architectural Context

### 2.1 Current state (verified at spec time)

- **`CurrentSchemaVersion` is 17** (`internal/db/db.go:444`), bumped most recently by **B018** which **dropped** `entity_type` `CHECK` constraints from polymorphic-association tables (`entity_notes`, `entity_relationships`, `entity_tags`). See `internal/db/db.go:436-444` history block.
- The polymorphic pattern to mirror is **`entity_notes`** (`internal/db/db.go:1698-1811`), specifically the `migrateEntityNotes` function.
- Key parsing is centralised in **`internal/keys/service.go`** — the `KeyService.Parse()` method currently recognises Epic/Feature/Task/Bug/Change. Sprint must be added to this parser (and matching `EntityType` constant added).
- Lower-level boolean-form helpers live in **`internal/keys/validation.go`** (`IsEpicKey`, `IsTaskKey`, `IsTechDebtKey`, etc.). A matching `IsSprintKey` should be added there for symmetry, even though most callers route through `KeyService.Parse()`.

### 2.2 Critical user-feedback constraints

- **`feedback_entity_type_check_constraints`** — When introducing a new entity_type value, *either* extend every entity-type allowlist (DB CHECKs in `db.go` + Go validators in `models/validation.go`) *or* drop the CHECKs in favour of app-layer validation. **Post-B018 the polymorphic tables no longer carry CHECKs**, so for `sprint_assignments.entity_type` we follow the post-B018 convention: **no DB CHECK constraint**, validation lives in the app layer (model `Validate()` + service layer). This avoids re-introducing a constraint that B018 just removed.
- **`feedback_concise_docs_small_features`** — STANDARD-tier feature, so this spec is intentionally short.
- **`feedback_new_commands_all_entities`** — Not directly applicable here (no CLI commands in this feature), but the polymorphic design *enables* the downstream sprint commands in E19-F03 to support all four entity types out of the box.

---

## 3. Schema Design

### 3.1 New migration function

Location: `internal/db/db.go`, registered in `runMigrations()` between the existing tech-debt and tags migrations (post-`migrateTechDebtTable`). Function name: **`migrateSprintTables`**. Idempotent — checks `sqlite_master` for existence before creating.

### 3.2 Schema version bump

```go
// internal/db/db.go (history comment + constant)
//
//   16 — E07-F42 (size columns)
//   17 — B018  (drop entity_type CHECKs from polymorphic-association tables)
//   18 — E19-F01 (sprints, sprint_assignments, sprint_capacity)
//
const CurrentSchemaVersion = 18
```

### 3.3 DDL

```sql
-- ─── Table 1: sprints ────────────────────────────────────────────────────────
CREATE TABLE sprints (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    key         TEXT NOT NULL UNIQUE,                  -- 'S001', 'S024', …
    name        TEXT NOT NULL,
    goal        TEXT,
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    status      TEXT NOT NULL DEFAULT 'planning',      -- workflow-validated at service layer
    slug        TEXT,                                   -- auto-generated from name
    file_path   TEXT,                                   -- optional sprint markdown file
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (start_date < end_date)
);

CREATE UNIQUE INDEX idx_sprints_key    ON sprints(key);
CREATE INDEX        idx_sprints_status ON sprints(status);
CREATE INDEX        idx_sprints_slug   ON sprints(slug);

CREATE TRIGGER sprints_updated_at
AFTER UPDATE ON sprints
FOR EACH ROW
BEGIN
    UPDATE sprints SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ─── Table 2: sprint_assignments (polymorphic) ───────────────────────────────
-- entity_type is one of: 'task', 'bug', 'change_card', 'tech_debt'.
-- Per the post-B018 convention (see B018 commit / db.go:436-444), we do NOT
-- add a CHECK constraint on entity_type — validation happens at the app layer.
CREATE TABLE sprint_assignments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    sprint_id    INTEGER NOT NULL,
    entity_type  TEXT    NOT NULL,                     -- 'task' | 'bug' | 'change_card' | 'tech_debt'
    entity_id    INTEGER NOT NULL,
    assigned_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    removed_at   TIMESTAMP,                            -- soft delete; NULL = active
    FOREIGN KEY (sprint_id) REFERENCES sprints(id) ON DELETE CASCADE
);

CREATE INDEX idx_sprint_assignments_sprint ON sprint_assignments(sprint_id);
CREATE INDEX idx_sprint_assignments_entity ON sprint_assignments(entity_type, entity_id);

-- One-active-sprint-per-entity is enforced by a partial unique index
-- (this is the core integrity guarantee for REQ-F-004 acceptance criterion #5):
CREATE UNIQUE INDEX idx_sprint_assignments_active_one
    ON sprint_assignments(entity_type, entity_id)
    WHERE removed_at IS NULL;

-- Cascade-delete triggers — mirror entity_notes pattern (db.go:1750-1773).
-- One trigger per parent table; deleting an entity nulls out its sprint assignment.
CREATE TRIGGER sprint_assignments_cascade_delete_task
AFTER DELETE ON tasks
FOR EACH ROW
BEGIN
    DELETE FROM sprint_assignments WHERE entity_type = 'task' AND entity_id = OLD.id;
END;

CREATE TRIGGER sprint_assignments_cascade_delete_bug
AFTER DELETE ON bugs
FOR EACH ROW
BEGIN
    DELETE FROM sprint_assignments WHERE entity_type = 'bug' AND entity_id = OLD.id;
END;

CREATE TRIGGER sprint_assignments_cascade_delete_change_card
AFTER DELETE ON change_cards
FOR EACH ROW
BEGIN
    DELETE FROM sprint_assignments WHERE entity_type = 'change_card' AND entity_id = OLD.id;
END;

CREATE TRIGGER sprint_assignments_cascade_delete_tech_debt
AFTER DELETE ON tech_debts
FOR EACH ROW
BEGIN
    DELETE FROM sprint_assignments WHERE entity_type = 'tech_debt' AND entity_id = OLD.id;
END;

-- ─── Table 3: sprint_capacity ────────────────────────────────────────────────
CREATE TABLE sprint_capacity (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    sprint_id        INTEGER NOT NULL,
    agent_type       TEXT NOT NULL,
    capacity_points  REAL NOT NULL DEFAULT 0,           -- planned capacity (set by PM)
    -- allocated_points is computed at query time as Σ(size) over assigned entities,
    -- per REQ-F-014 acceptance criterion #3. We persist a column for future
    -- caching/snapshot use but do not maintain it via triggers in this feature.
    allocated_points REAL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sprint_id) REFERENCES sprints(id) ON DELETE CASCADE,
    UNIQUE (sprint_id, agent_type)
);

CREATE INDEX  idx_sprint_capacity_sprint ON sprint_capacity(sprint_id);
CREATE TRIGGER sprint_capacity_updated_at
AFTER UPDATE ON sprint_capacity
FOR EACH ROW
BEGIN
    UPDATE sprint_capacity SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### 3.4 Why no DB CHECK on `sprint_assignments.entity_type`

Per the B018 history note (`internal/db/db.go:436-444`) and the user-feedback memory `feedback_entity_type_check_constraints`, **the polymorphic-association tables (`entity_notes`, `entity_relationships`, `entity_tags`) had their entity_type CHECK constraints dropped** because every new entity type required coordinated migrations of every CHECK and every Go validator. Re-introducing a CHECK on a *new* polymorphic table here would re-create exactly the problem B018 solved. Validation therefore happens **only** at the app layer (§4.2).

---

## 4. Code Changes

### 4.1 Key parsing — `internal/keys/service.go`

Add S### support to the canonical parser:

```go
const (
    EntityTypeEpic     EntityType = "epic"
    EntityTypeFeature  EntityType = "feature"
    EntityTypeTask     EntityType = "task"
    EntityTypeBug      EntityType = "bug"
    EntityTypeChange   EntityType = "change"
    EntityTypeSprint   EntityType = "sprint"      // NEW
    EntityTypeUnknown  EntityType = "unknown"
)

// Add to ParsedKey:
type ParsedKey struct {
    // ...existing fields...
    SprintNum string  // NEW — numeric part of S### (e.g., "024" from S024)
}

// Add the regex (mirrors bugVariablePattern):
sprintKeyPattern = regexp.MustCompile(`^S(\d{3})$`)   // strict 3-digit zero-padded

// Add to Parse(), before the bug case (S has its own letter prefix, no ambiguity):
if m := sprintKeyPattern.FindStringSubmatch(upper); m != nil {
    result.EntityType = EntityTypeSprint
    result.SprintNum  = m[1]
    result.Normalized = fmt.Sprintf("S%s", m[1])
    return result
}

// Add to Format():
case EntityTypeSprint:
    return fmt.Sprintf("S%s", parsed.SprintNum)
```

### 4.2 Lower-level helpers — `internal/keys/validation.go`

Add a parallel `IsSprintKey` for symmetry with `IsEpicKey`/`IsTechDebtKey`:

```go
sprintKeyPattern = regexp.MustCompile(`^S\d{3}$`)

func IsSprintKey(s string) bool {
    return sprintKeyPattern.MatchString(Normalize(s))
}
```

### 4.3 Sprint model — `internal/models/sprint.go` (new file)

```go
package models

import "time"

// SprintStatus is a typed alias; valid values are workflow-defined and
// validated at the service layer (mirrors TaskStatus pattern, NOT a hardcoded enum).
type SprintStatus string

type Sprint struct {
    ID        int64
    Key       string         // 'S001', 'S024', …
    Name      string
    Goal      string
    StartDate time.Time
    EndDate   time.Time
    Status    SprintStatus
    Slug      string
    FilePath  string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type SprintAssignment struct {
    ID         int64
    SprintID   int64
    EntityType string  // 'task' | 'bug' | 'change_card' | 'tech_debt'
    EntityID   int64
    AssignedAt time.Time
    RemovedAt  *time.Time
}

type SprintCapacity struct {
    ID              int64
    SprintID        int64
    AgentType       string
    CapacityPoints  float64
    AllocatedPoints *float64
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

### 4.4 Validators — `internal/models/validation.go`

```go
// Add sprint key pattern alongside the existing epic/feature/task patterns:
sprintKeyPattern = regexp.MustCompile(`^S\d{3}$`)

// Sentinel error (mirrors ErrInvalidEpicKey):
var ErrInvalidSprintKey = errors.New(`invalid sprint key format: must match ^S\d{3}$`)

func ValidateSprintKey(key string) error {
    if !sprintKeyPattern.MatchString(key) {
        return fmt.Errorf("%w: got %q", ErrInvalidSprintKey, key)
    }
    return nil
}

// Add a validator for sprint_assignment.entity_type (this is the app-layer
// allowlist that replaces the absent DB CHECK constraint, per §3.4):
var ErrInvalidSprintAssignmentEntityType = errors.New(
    "invalid sprint assignment entity_type: must be task, bug, change_card, or tech_debt")

func ValidateSprintAssignmentEntityType(entityType string) error {
    valid := map[string]bool{
        "task":        true,
        "bug":         true,
        "change_card": true,
        "tech_debt":   true,
    }
    if !valid[entityType] {
        return fmt.Errorf("%w: got %q", ErrInvalidSprintAssignmentEntityType, entityType)
    }
    return nil
}

// Sprint.Validate() — structural only (status validity is workflow-driven, see service layer).
func (s *Sprint) Validate() error {
    if err := ValidateSprintKey(s.Key); err != nil {
        return err
    }
    if strings.TrimSpace(s.Name) == "" {
        return ErrEmptyTitle  // reuse existing sentinel — semantically "name cannot be empty"
    }
    if !s.EndDate.After(s.StartDate) {
        return errors.New("sprint end_date must be after start_date")
    }
    if strings.TrimSpace(string(s.Status)) == "" {
        return errors.New("sprint status cannot be empty")
    }
    return nil
}

// SprintAssignment.Validate() — entity_type allowlist + non-zero IDs.
func (sa *SprintAssignment) Validate() error {
    if sa.SprintID <= 0 {
        return errors.New("sprint_id must be greater than 0")
    }
    if sa.EntityID <= 0 {
        return errors.New("entity_id must be greater than 0")
    }
    return ValidateSprintAssignmentEntityType(sa.EntityType)
}
```

### 4.5 Slug generation

The `sprints.slug` column is auto-populated from `name` using the same lowercasing/hyphenation algorithm used by epics/features (`internal/repository/key_lookup.go` slug helpers). This feature does **not** add slug-aware lookup to the new repository; that is part of E19-F02. We only ensure the column exists and is indexed.

---

## 5. Layering & Out-of-Scope (this feature)

| Layer | This feature delivers | Deferred to later E19 features |
|---|---|---|
| **DB / migration** | All three tables, indexes, triggers, partial unique index, schema version bump | — |
| **Models** | `Sprint`, `SprintAssignment`, `SprintCapacity`, validators | — |
| **`keys/` package** | S### parsing in `Parse()` + `IsSprintKey` helper | — |
| **Repository** | **Not in this feature** — no `SprintRepository` yet | E19-F02 |
| **Service** | **Not in this feature** | E19-F02 (lifecycle), E19-F03 (assignment), E19-F04 (analytics), E19-F05 (planning) |
| **CLI** | **Not in this feature** | E19-F02 onward |
| **Workflow config** | **Not in this feature** — sprint-level workflow lives in `.sharkworkflow*.json`, configured in E19-F02 | E19-F02 |

---

## 6. Test Plan

All tests use the existing patterns referenced in `.claude/rules/testing/*`. Repository-style tests use the real test DB (`test.GetTestDB()`); model and key tests are pure unit tests.

### 6.1 Migration tests — `internal/db/sprint_tables_migration_test.go` (new)

Mirror `internal/db/entity_notes_migration_test.go` and `internal/db/change_cards_table_test.go`.

| Test | Asserts |
|---|---|
| `TestMigrateSprintTables_CreatesAllThreeTables` | After running migration, `sqlite_master` contains `sprints`, `sprint_assignments`, `sprint_capacity`. |
| `TestMigrateSprintTables_Idempotent` | Running migration twice does not error and produces identical schema. |
| `TestMigrateSprintTables_CreatesAllIndexes` | All seven indexes from §3.3 exist (`idx_sprints_key`, `idx_sprints_status`, `idx_sprints_slug`, `idx_sprint_assignments_sprint`, `idx_sprint_assignments_entity`, `idx_sprint_assignments_active_one`, `idx_sprint_capacity_sprint`). |
| `TestMigrateSprintTables_PartialUniqueIndex` | Inserting two `sprint_assignments` rows with the same `(entity_type, entity_id)` and both `removed_at IS NULL` raises `UNIQUE constraint failed`. The same insertion with one row's `removed_at` set succeeds. |
| `TestMigrateSprintTables_StartEndDateCheck` | `INSERT INTO sprints (..., start_date='2026-04-01', end_date='2026-03-18', ...)` raises `CHECK constraint failed`. |
| `TestMigrateSprintTables_CascadeDeleteFromTask` | Inserting a sprint_assignment for a task, then deleting the task, leaves zero rows in `sprint_assignments`. |
| `TestMigrateSprintTables_CascadeDeleteFromBug` | Same as above, bug variant. |
| `TestMigrateSprintTables_CascadeDeleteFromChangeCard` | Same as above, change_card variant. |
| `TestMigrateSprintTables_CascadeDeleteFromTechDebt` | Same as above, tech_debt variant. |
| `TestMigrateSprintTables_NoEntityTypeCheckConstraint` | Inserting `entity_type='whatever'` succeeds at the DB layer (asserts the post-B018 convention is preserved — entity_type validation is app-layer-only). |
| `TestSchemaVersionBumpedTo18` | After `ApplySchemaAndMigrations`, `getSchemaVersion(db) == 18`. |

### 6.2 Key parsing tests — extend `internal/keys/service_test.go`

Add cases to `TestKeyService_DetectEntityType`, `TestKeyService_Parse`, `TestKeyService_Normalize`, `TestKeyService_Format`:

| Input | Expected |
|---|---|
| `"S001"` | `EntityTypeSprint`, normalized `"S001"`, `SprintNum="001"` |
| `"s024"` (case-insensitive) | `EntityTypeSprint`, normalized `"S024"` |
| `"S0"`, `"S1"`, `"S0001"`, `"SPRINT-1"` | `EntityTypeUnknown` (strict 3-digit only) |
| `Format(ParsedKey{EntityType: EntityTypeSprint, SprintNum: "024"})` | `"S024"` |

### 6.3 Validation tests — `internal/models/sprint_test.go` (new)

| Test | Asserts |
|---|---|
| `TestValidateSprintKey_Valid` | `"S001"`, `"S024"`, `"S999"` all return nil. |
| `TestValidateSprintKey_Invalid` | `""`, `"S0"`, `"S1"`, `"s024"` (lowercase — caller is responsible for `Normalize` upstream), `"E07"`, `"S0001"` all return `ErrInvalidSprintKey`. |
| `TestValidateSprintAssignmentEntityType_Valid` | All four allowed values (`task`, `bug`, `change_card`, `tech_debt`) return nil. |
| `TestValidateSprintAssignmentEntityType_Invalid` | `""`, `"epic"`, `"feature"`, `"idea"` (intentionally not allowlisted), `"TASK"`, `" task "` all return `ErrInvalidSprintAssignmentEntityType`. |
| `TestSprint_Validate_*` | Empty name, end_date<=start_date, empty status, valid case all behave correctly. |
| `TestSprintAssignment_Validate_*` | sprint_id<=0, entity_id<=0, invalid entity_type all rejected; happy path passes. |

### 6.4 Lower-level helper test — `internal/keys/validation_test.go`

Add a `TestIsSprintKey` mirroring `TestIsTechDebtKey`.

---

## 7. Acceptance Criteria

Mapped 1:1 to REQ-F-001 and REQ-F-010 acceptance criteria from `requirements.md`:

### From REQ-F-010 (Sprint Database Tables)

- [ ] **AC-1** — `sprints` table exists with columns `id, key, name, goal, start_date, end_date, status, slug, file_path, created_at, updated_at`.
- [ ] **AC-2** — `sprint_assignments` table exists with columns `id, sprint_id (FK), entity_type, entity_id, assigned_at, removed_at`.
- [ ] **AC-3** — `sprint_assignments` has **no** `task_id` FK column (polymorphic, mirroring `entity_notes`).
- [ ] **AC-4** — Partial unique index `idx_sprint_assignments_active_one` exists on `(entity_type, entity_id) WHERE removed_at IS NULL` and enforces one-active-sprint-per-entity.
- [ ] **AC-5** — `sprint_capacity` table exists with columns `id, sprint_id (FK), agent_type, capacity_points, allocated_points` and `UNIQUE(sprint_id, agent_type)`.
- [ ] **AC-6** — Foreign key `sprint_assignments.sprint_id REFERENCES sprints(id) ON DELETE CASCADE` is present; `entity_type` + `entity_id` are app-validated (no DB FK to entity tables, no DB CHECK on entity_type — consistent with post-B018 convention).
- [ ] **AC-7** — Indexes exist on `sprints.status`, `sprints.slug`, `sprint_assignments(entity_type, entity_id)`, `sprint_assignments(sprint_id)`, and `sprint_capacity(sprint_id)`.
- [ ] **AC-8** — Migration is idempotent (running it twice does not error, asserted by `TestMigrateSprintTables_Idempotent`).
- [ ] **AC-9** — `CurrentSchemaVersion` is bumped to `18`; `internal/db/db.go` history comment is updated.
- [ ] **AC-10** — Cascade-delete triggers exist for all four parent entity types (`tasks`, `bugs`, `change_cards`, `tech_debts`), so deleting a parent entity removes its sprint assignments.

### From REQ-F-001 (Sprint Creation — *foundational subset only*)

- [ ] **AC-11** — Sprint key format `S###` (zero-padded 3-digit) is recognised by `keys.KeyService.Parse()` and returns `EntityTypeSprint`.
- [ ] **AC-12** — `keys.IsSprintKey()` exists and validates the format case-insensitively.
- [ ] **AC-13** — `models.Sprint` struct exists with `Validate()` method that enforces non-empty name, `end_date > start_date`, non-empty status, and valid `S###` key.
- [ ] **AC-14** — `models.SprintAssignment.Validate()` enforces `entity_type ∈ {task, bug, change_card, tech_debt}` via `ValidateSprintAssignmentEntityType`.

> **Note.** The behavioural REQ-F-001 criteria — auto-incrementing key allocation, `--start`/`--end`/`--goal` flag handling, status defaulting to `planning`, `--json` output — depend on a `SprintRepository` and `SprintService` that are **deferred to E19-F02**. This feature ships *only* the schema, model, and key-parsing prerequisites for those.

### Cross-cutting

- [ ] **AC-15** — `make fmt && make lint && make test` all pass.
- [ ] **AC-16** — All tests listed in §6 are present and green.
- [ ] **AC-17** — No existing tests regress (verified by full `make test`).

---

## 8. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| `S###` key conflicts with the existing `B###`/`C###` parser branches | Low | `S` is a unique letter prefix; the regex `^S\d{3}$` is anchored and non-overlapping with bug (`^B\d+$`) and change (`^C\d+$`) patterns. Keys-package tests cover ambiguity directly. |
| Re-introducing an `entity_type` CHECK by accident (against B018 convention) | Medium | Explicitly tested (`TestMigrateSprintTables_NoEntityTypeCheckConstraint`); spec §3.4 documents the rationale; reviewer checklist below. |
| Forgetting to bump `CurrentSchemaVersion` (most-common migration bug per `database-critical.md`) | High | `TestSchemaVersionBumpedTo18` is a hard fail-gate; spec lists the bump explicitly in §3.2. |
| Cascade-delete triggers missed for one of the four parent tables | Medium | One test per parent table in §6.1 (four tests). |
| Slug collisions across sprints | Low | Slugs are not required to be unique (per `database/schema.md` slug architecture); lookup will require both `key` and `slug` to match in E19-F02. |

---

## 9. Reviewer Checklist (for the upcoming Technical Review gate)

- [ ] DDL exactly matches §3.3, including the `CHECK (start_date < end_date)` clause and the partial unique index.
- [ ] No `entity_type IN (...)` `CHECK` clause exists on `sprint_assignments` (verifies B018 convention is honoured).
- [ ] `CurrentSchemaVersion` is `18` and the history comment is updated.
- [ ] `EntityTypeSprint` constant added to `internal/keys/service.go`; `Parse()`, `Format()`, and `Normalize()` all handle it.
- [ ] `ValidateSprintAssignmentEntityType` exists in `internal/models/validation.go` and lists all four entity types — adding a fifth entity type later requires updating only this one Go function (no DB migration).
- [ ] Test count matches §6 (≥11 migration tests, ≥4 key-parsing additions, ≥6 model/validation tests, 1 helper test).

---

*End of spec.*
