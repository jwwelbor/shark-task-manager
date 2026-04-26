---
feature_key: E07-F42
title: Add Size field to all entities (numeric with t-shirt mapping)
status: in_specification
spec_type: combined-requirements-architecture
last_updated: 2026-04-25
---

# E07-F42 Specification: Size Field on All Entities

**Feature Key**: E07-F42
**Parent Epic**: [E07 — Enhancements](../epic.md) (placeholder epic for cross-cutting CLI enhancements)
**Complexity**: COMPLEX (18/27 — see decision note on the feature)
**Reference Precedent**: E28 — Entity Tagging with Managed Vocabulary (mirror its breadth pattern)

---

## 1. Context (Reference Only)

This feature lives in E07 (a placeholder enhancements epic — there is no goal-narrative epic PRD to restate). The "why" comes from `.claude/rules/development-workflows.md`, which already prescribes t-shirt sizes (XS–XXL) or Fibonacci story points (1, 2, 3, 5, 8, 13) for sizing tasks but provides no enforceable representation of that guidance.

The feature description on `feature.md` (lines 21–49) is the authoritative scope sketch for this work. **This spec elaborates that sketch into testable requirements and an executable architecture.** It does not restate the rationale.

Key facts inherited from triage:

- The shared `BaseEntity` (`internal/models/entity.go:36`) is the right place for the new field because all six typed entities embed it.
- `complexity_tier` lives only in the loose `Metadata` map and is surfaced through `internal/config/template/helpers.go:376-388`. It has no schema, no validation, no flag, no rollups.
- The change is **breadth, not depth**: 6 entity tables, ~10 files, additive/nullable. Risk is consistency across surfaces, not algorithmic difficulty.

---

## 2. Requirements (Incremental)

### 2.1 Functional Requirements

#### Core data model

- **REQ-F-001 — `Size` field on `BaseEntity`**
  Add `Size *int` to `BaseEntity` in `internal/models/entity.go`. Field is nullable (`*int`) so that adoption is progressive — existing rows and entities created without `--size` remain valid.
  *Testable*: a `BaseEntity` with `Size = nil` round-trips through JSON marshal/unmarshal and DB round-trip without error; `GetSize() *int` accessor returns nil; `Size = ptr(5)` round-trips and accessor returns the same pointer value.

- **REQ-F-002 — Canonical numeric values (Fibonacci)**
  The only valid stored values are `{1, 2, 3, 5, 8, 13}`. Any other integer must be rejected at the model validation layer.
  *Testable*: `ValidateSize(int)` accepts each of `{1, 2, 3, 5, 8, 13}` and rejects `0`, `4`, `6`, `7`, `9`, `12`, `14`, negative numbers, and `100`.

- **REQ-F-003 — T-shirt label mapping (bidirectional)**
  Define a single canonical mapping: `XS=1, S=2, M=3, L=5, XL=8, XXL=13`. Both forms accepted on input; both forms presentable on output.
  *Testable*:
  - `ParseSize("XS")`, `ParseSize("xs")`, `ParseSize("Xs")`, `ParseSize(" XS ")` all return `(1, nil)`.
  - `ParseSize("1")`, `ParseSize(" 1 ")` return `(1, nil)`.
  - `ParseSize("XXL")` returns `(13, nil)`; `ParseSize("13")` returns `(13, nil)`.
  - `ParseSize("XXXL")`, `ParseSize("4")`, `ParseSize("medium")`, `ParseSize("")` return non-nil error.
  - `SizeLabel(1)` returns `"XS"`; `SizeLabel(13)` returns `"XXL"`; `SizeLabel(0)` returns `""` and an error or empty-by-contract sentinel.

#### CLI surface

- **REQ-F-004 — `--size` flag on all 6 `create` commands**
  Add `--size <value>` to: `shark epic create`, `shark feature create`, `shark task create`, `shark bug create`, `shark change create`, `shark idea create`. Value accepts either form (e.g., `--size 5` or `--size L`). Flag is optional; absence stores NULL.
  *Testable*: each create command exits 0 with `--size 5`, with `--size L`, and with the flag omitted; each exits non-zero with `--size 4` or `--size XXXL`. The persisted row reflects the canonical numeric value.

- **REQ-F-005 — `--size` flag on all 6 `update` commands**
  Add `--size <value>` to the corresponding `update` command for each entity type. Behavior:
  - Flag absent → no change to size (sentinel-respecting, like the existing `--execution-order=-1` pattern but using `--size=""` since the canonical type is string-input).
  - `--size <valid>` → update to that value.
  - `--size clear` (literal) → set the column back to NULL.
  *Testable*: `shark task update <key> --size L` then `shark get <key> --field size` returns `5` (or `L` per output format); `shark task update <key> --size clear` then `shark get <key> --field size` returns `null`/empty.

- **REQ-F-006 — Output renders both forms**
  In JSON output, the entity object includes `size` as a numeric (`null | 1 | 2 | 3 | 5 | 8 | 13`). In human/table output, render as `<label> (<num>)` (e.g., `M (3)`) or `—` when nil. The CLI must not rely on the database's persistence form for display — it always goes through the mapper.
  *Testable*: `shark task get <key> --json` contains `"size": 5`; non-JSON `shark get <key>` includes `Size: L (5)`.

- **REQ-F-007 — `--field size` extraction**
  `shark get <key> --field size` returns the **numeric** form (matches the JSON contract from REQ-F-006). A separate `--field size_label` returns the t-shirt label. When size is NULL, both fields exit code 4 (field-not-found / null sentinel — match existing `--field` behavior for NULL fields).
  *Testable*: `shark get <key> --field size` on a sized entity prints `5`; on an unsized entity exits code 4 (matches existing convention for missing/null fields per `docs/cli-reference/global-flags.md`).

#### Persistence

- **REQ-F-008 — `size` column on all six entity tables**
  Add `size INTEGER NULL` to: `epics`, `features`, `tasks`, `bugs`, `change_cards`, `ideas`. (TechDebt is excluded from CLI surfaces in this feature — see Out of Scope item OOS-3.)
  *Testable*: `PRAGMA table_info(<table>)` lists a `size` column of type `INTEGER` for each of the six tables; existing rows have `NULL` after migration.

- **REQ-F-009 — Schema version bump**
  Bump `CurrentSchemaVersion` from `14` to `15` in `internal/db/db.go`. The new migration `migrateAddSizeColumns(db *sql.DB) error` is idempotent (`ALTER TABLE … ADD COLUMN size INTEGER` guarded by `pragma_table_info` check, mirroring the existing `file_path`/`execution_order` migration pattern at `internal/db/db.go:507-590`).
  *Testable*: opening a fresh DB applies the migration once; opening an already-migrated DB is a no-op (column-existence check returns >0); the migration completes without error on the populated test database used by repository integration tests.

#### Repository integration

- **REQ-F-010 — `size` round-trips through every CRUD path**
  All `INSERT`, `SELECT`, and `UPDATE` statements that touch the six affected tables include the `size` column. The field flows through Repository → Service → CLI for each entity type without information loss.
  *Testable*: for each of the six entity types, an end-to-end test creates an entity with `--size L`, retrieves it, asserts the persisted value is `5`, updates to `--size XS`, and asserts the persisted value is `1`.

#### Template surface

- **REQ-F-011 — `{{ .size }}` and `{{ .size_label }}` template variables**
  Add both placeholders to the entity-template renderer (`internal/config/template/helpers.go`) for every entity type. When size is NULL, both placeholders render as the empty string. The new variables are independent from `complexity_tier`.
  *Testable*: rendering an entity template with `size=5` produces `5` for `{{ .size }}` and `L` for `{{ .size_label }}`; with `size=nil` both render as `""`.

- **REQ-F-012 — `complexity_tier` fallback preserved (one release)**
  The existing `complexity_tier` extraction (`helpers.go:376-388`, `helpers.go:480-488`) remains in place. Behavior: when `Size` is non-nil, the new placeholders take their value from `Size`; `complexity_tier` is unchanged in its current form. Deprecation is announced in this release's CHANGELOG and removal is deferred (OOS-1).
  *Testable*: an entity with `complexity_tier="L"` in `Metadata` and `Size=nil` still renders `complexity_tier="L"` in templates; the same entity with `Size=ptr(8)` renders `size=8` and `size_label="XL"` and `complexity_tier="L"` (both surfaces populated, neither overrides the other).

### 2.2 Non-Functional Requirements

- **REQ-NF-001 — Migration runtime**
  The schema-15 migration adds one nullable column to six tables and creates no indexes. On a database with 10,000 rows total across all six tables, migration must complete in under 1 second.
  *Measurement*: timed run of `ApplySchemaAndMigrations` on a seeded fixture in repository integration tests; record actual duration in test output.

- **REQ-NF-002 — Backward compatibility**
  All existing CLI invocations that do not use `--size` must continue to work unchanged. No existing test (unit, integration, or CLI) may need to be modified except (a) tests that explicitly exercise create/update on the affected entities and want to additionally assert the new field's default, or (b) snapshot tests of JSON output that now contain a `"size": null` field.
  *Measurement*: `make test` passes; the diff to existing test files is limited to (a) and (b).

- **REQ-NF-003 — Input sanitization**
  `ParseSize` and `ValidateSize` follow `.claude/rules/go/input-sanitization.md`:
  - `strings.TrimSpace` on input before parsing.
  - Case-insensitive label match via `strings.ToUpper`.
  - Errors quote user input with `%q` to prevent log injection.
  - Allowlist enforcement (no string interpolation into SQL — `size` is bound as an `int` parameter).
  *Measurement*: code review against `internal/models/validation.go` patterns; a sanitization test covers whitespace, casing, control characters, oversized strings (>20 chars rejected without ambiguity), and `'; DROP TABLE epics; --` style payloads.

### 2.3 Acceptance Criteria (Feature-Level)

**Scenario 1: Create with t-shirt label**
- **Given** an existing epic E07 and feature E07-F01
- **When** the user runs `shark task create E07 F01 "Some Task" --size L`
- **Then** the new task is persisted with `size = 5`
- **And** `shark get <key> --field size` returns `5`
- **And** `shark get <key> --json` contains `"size": 5`

**Scenario 2: Create with numeric value**
- **Given** the same setup as Scenario 1
- **When** the user runs `shark task create E07 F01 "Other Task" --size 8`
- **Then** the new task is persisted with `size = 8`
- **And** the human-readable display shows `XL (8)`

**Scenario 3: Reject invalid size**
- **Given** any entity creation context
- **When** the user runs `shark task create E07 F01 "Bad Task" --size 4`
- **Then** the command exits non-zero
- **And** stderr contains an error mentioning the allowed values `{1, 2, 3, 5, 8, 13}` or labels `{XS, S, M, L, XL, XXL}`
- **And** no database row is created

**Scenario 4: Update size, then clear it**
- **Given** an entity with `size = 5`
- **When** the user runs `shark task update <key> --size XS`
- **Then** the entity persists with `size = 1`
- **When** the user then runs `shark task update <key> --size clear`
- **Then** the entity persists with `size = NULL`

**Scenario 5: Migration is idempotent**
- **Given** a database that already has the schema-15 migration applied
- **When** shark runs again and `ApplySchemaIfNeeded` checks the version
- **Then** the version check returns "no work needed" and no DDL executes
- **And** existing `size` values are preserved

**Scenario 6: Backward compatibility**
- **Given** a database with rows that have `size = NULL` (rows created pre-feature)
- **When** the user runs any read command (`shark get`, `shark list`, `shark task get`, etc.)
- **Then** every command succeeds
- **And** size renders as `—` in human output and `null` in JSON

**Scenario 7: Template variables**
- **Given** an entity template that references `{{ .size }}` and `{{ .size_label }}`
- **When** the entity is rendered with `size = 5`
- **Then** `{{ .size }}` renders `5` and `{{ .size_label }}` renders `L`
- **Given** the same template with `size = nil`
- **Then** both placeholders render as the empty string

### 2.4 Out of Scope (This Feature)

- **OOS-1 — `complexity_tier` removal**: kept as a fallback for one release per the feature description. Deprecation announcement only; removal is a follow-up.
- **OOS-2 — Workflow gate enforcement**: e.g., "block transition to `ready_for_development` if size > M". Configurable in `.sharkconfig.json` later. No code paths gated on size in this feature.
- **OOS-3 — Tech-debt entity coverage**: TechDebt has no `create`/`update` CLI surface today (no `shark techdebt` command exists in user-facing CLI). The model field and DB column are excluded from this feature. If/when a tech-debt CLI is added, follow this feature's pattern.
- **OOS-4 — Rollups and size-based analytics**: e.g., feature-level size sum, epic-level size distribution, "remaining size points" on dashboards. Deferred — requires UX design.
- **OOS-5 — Required-on-create per entity type**: similar to E28's `tag_required_for`. Out of scope; sizes are always optional in this feature.
- **OOS-6 — Migration of existing `complexity_tier` Metadata into `size`**: sometimes `complexity_tier` is `"L"` and could mechanically become `size=5`. This feature leaves them independent. A separate one-shot migration command (e.g., `shark admin migrate complexity-to-size`) is a valid follow-up but out of scope here.
- **OOS-7 — HTTP API surface**: the HTTP API in `cmd/server/` is not extended in this feature. The DB column is added so the API will see it on `SELECT`, but no DTO/handler changes are made. Follow-up work can wire `--size` parity into the HTTP layer.

---

## 3. Architecture

### 3.1 Affected Surfaces (File-Level Inventory)

| Surface | File | Change |
|---|---|---|
| Model — shared field | `internal/models/entity.go` | Add `Size *int` to `BaseEntity`; add `GetSize() *int` and `SetSize(*int)` accessor methods (interface additions optional but recommended) |
| Model — size primitives | `internal/models/size.go` *(new)* | Canonical values `{1,2,3,5,8,13}`, label map, `ParseSize(string) (int, error)`, `SizeLabel(int) (string, error)`, `ValidateSize(int) error`, sentinel error `ErrInvalidSize` |
| Model — validation | `internal/models/validation.go` | Add `ErrInvalidSize` (or import from `size.go`); add `ValidateSize` reference comment to align with the existing pattern (`ValidateNoteType`, etc.) |
| Model — interface | `internal/models/entity.go` | Optionally add `GetSize() *int` and `SetSize(*int)` to the `Entity` interface — recommended for symmetry with `GetContextData`/`SetContextData` |
| DB — schema bump | `internal/db/db.go` | `CurrentSchemaVersion = 15`; new `migrateAddSizeColumns(db *sql.DB) error` called from `runMigrations()` |
| Repo — Epic | `internal/repository/epic/repository.go` | Add `size` to `INSERT`, `SELECT` (all variants), `UPDATE` |
| Repo — Feature | `internal/repository/feature/repository.go` | Same |
| Repo — Task | `internal/repository/<task package>/repository.go` | Same |
| Repo — Bug | `internal/repository/bug/repository.go` | Same |
| Repo — Change | `internal/repository/changecard/repository.go` | Same |
| Repo — Idea | `internal/repository/idea/repository.go` | Same |
| Service — DTO | `internal/services/*_dto.go` | Add `Size *int` to `CreateXInput` and `UpdateXInput` for each of the six entity services that have DTOs; pass through to repository |
| Service — body | `internal/services/<entity>_service.go` (×6) | `CreateX`/`UpdateX` methods set the new field on the model before persistence; no business logic beyond passthrough |
| CLI — create commands | `internal/cli/commands/{epic,feature,task,bug,change,idea}.go` | Add `--size` flag; parse with `models.ParseSize`; pass to service |
| CLI — update commands | Same files | Add `--size` flag with `clear` literal handling; parse and pass to service |
| Template — placeholders | `internal/config/template/helpers.go` | Populate `placeholders["size"]` and `placeholders["size_label"]` for every entity-aware helper (currently feature/task; extend pattern to epic/bug/change/idea helpers if they exist, or add a shared `applySizePlaceholders(entity, placeholders)` helper) |
| Templates — content | `shark-templates/**/*.tmpl` | Update entity templates (`feature_short/`, `task/`, etc.) to display `{{ .size_label }} ({{ .size }})` in their frontmatter or body where appropriate |

### 3.2 Data Model

#### New column

```sql
ALTER TABLE epics         ADD COLUMN size INTEGER NULL;
ALTER TABLE features      ADD COLUMN size INTEGER NULL;
ALTER TABLE tasks         ADD COLUMN size INTEGER NULL;
ALTER TABLE bugs          ADD COLUMN size INTEGER NULL;
ALTER TABLE change_cards  ADD COLUMN size INTEGER NULL;
ALTER TABLE ideas         ADD COLUMN size INTEGER NULL;
```

**No CHECK constraint on the column.** The decision rationale: SQLite `CHECK (size IN (1,2,3,5,8,13))` would be brittle if the canonical set ever changes (e.g., adding `21`). Validation lives in `models.ValidateSize` per the existing pattern (`.claude/rules/go/patterns.md` — "Two levels of validation"). Database constraints are reserved for `NOT NULL`, `UNIQUE`, and `FOREIGN KEY` (see `.claude/rules/database/schema.md`).

**No new index.** The triage notes do not call for query patterns over size (no "list features with size > M"). Adding an index would be premature; revisit when rollups (OOS-4) are designed.

#### Migration function (sketch)

```go
// migrateAddSizeColumns adds the nullable `size` column to each of the six
// entity tables. Idempotent: each ALTER is guarded by a pragma_table_info
// existence check, mirroring migrateFilePath/migrateExecutionOrder upstream.
//
// DEVELOPER NOTE: This function adds schema version 15. Bump
// CurrentSchemaVersion to 15 (if not already done) so ApplySchemaIfNeeded
// detects the version gap. See .claude/rules/database-critical.md.
//
// Part of Epic E07 — Enhancements (E07-F42).
func migrateAddSizeColumns(db *sql.DB) error {
    tables := []string{"epics", "features", "tasks", "bugs", "change_cards", "ideas"}
    for _, table := range tables {
        var exists int
        err := db.QueryRow(
            `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'size'`,
            table,
        ).Scan(&exists)
        if err != nil {
            return fmt.Errorf("migrateAddSizeColumns: check %s: %w", table, err)
        }
        if exists == 0 {
            //nolint:gosec // table is from a hardcoded allowlist above; cannot inject
            stmt := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN size INTEGER NULL`, table)
            if _, err := db.Exec(stmt); err != nil {
                return fmt.Errorf("migrateAddSizeColumns: alter %s: %w", table, err)
            }
        }
    }
    return nil
}
```

Pattern reference: matches `migrateAddTagsAndEntityTags` (`internal/db/db.go:3290`) and the `file_path`/`execution_order` blocks (`internal/db/db.go:507-590`). Bump `CurrentSchemaVersion` from `14` to `15` (`internal/db/db.go:438`).

### 3.3 Model Layer

#### `internal/models/size.go` (new file)

```go
package models

import (
    "errors"
    "fmt"
    "strconv"
    "strings"
)

// ErrInvalidSize is returned when a size value is not in the canonical
// Fibonacci set {1, 2, 3, 5, 8, 13} or its label form {XS, S, M, L, XL, XXL}.
var ErrInvalidSize = errors.New(
    "invalid size: must be one of 1, 2, 3, 5, 8, 13 (or XS, S, M, L, XL, XXL)")

// canonicalSizes is the allowed numeric set, in ascending order.
var canonicalSizes = []int{1, 2, 3, 5, 8, 13}

// sizeLabels maps numeric values to their canonical t-shirt labels.
var sizeLabels = map[int]string{
    1: "XS", 2: "S", 3: "M", 5: "L", 8: "XL", 13: "XXL",
}

// labelToSize is the inverse of sizeLabels, normalized to uppercase keys.
var labelToSize = map[string]int{
    "XS": 1, "S": 2, "M": 3, "L": 5, "XL": 8, "XXL": 13,
}

// validSize is an O(1) allowlist for canonical numeric values.
var validSize = map[int]bool{1: true, 2: true, 3: true, 5: true, 8: true, 13: true}

// ParseSize accepts either a t-shirt label ("XS"–"XXL", case-insensitive)
// or a numeric string ("1", "2", "3", "5", "8", "13") and returns the
// canonical numeric value. Whitespace is trimmed. Empty input is an error.
func ParseSize(input string) (int, error) {
    trimmed := strings.TrimSpace(input)
    if trimmed == "" {
        return 0, fmt.Errorf("%w: input cannot be empty", ErrInvalidSize)
    }
    upper := strings.ToUpper(trimmed)
    if v, ok := labelToSize[upper]; ok {
        return v, nil
    }
    n, err := strconv.Atoi(trimmed)
    if err != nil {
        return 0, fmt.Errorf("%w: got %q", ErrInvalidSize, input)
    }
    if !validSize[n] {
        return 0, fmt.Errorf("%w: got %q", ErrInvalidSize, input)
    }
    return n, nil
}

// SizeLabel returns the canonical t-shirt label for a numeric size.
// Returns ("", error) if the input is not in the canonical set.
func SizeLabel(n int) (string, error) {
    if label, ok := sizeLabels[n]; ok {
        return label, nil
    }
    return "", fmt.Errorf("%w: got %d", ErrInvalidSize, n)
}

// ValidateSize returns nil if n is in the canonical Fibonacci set.
// Use this in entity .Validate() methods for structural checks.
func ValidateSize(n int) error {
    if !validSize[n] {
        return fmt.Errorf("%w: got %d", ErrInvalidSize, n)
    }
    return nil
}
```

**Rationale**:
- Allowlist via `map[int]bool` matches `ValidateNoteType` style at `internal/models/validation.go:171-189`.
- Error wrapping with `%q` quotes user input to prevent log injection per `.claude/rules/go/input-sanitization.md`.
- Sentinel error `ErrInvalidSize` enables `errors.Is(err, models.ErrInvalidSize)` checks at the service/CLI layer.
- No external dependencies; pure stdlib.

#### `internal/models/entity.go` change

```go
type BaseEntity struct {
    // ... existing fields ...
    Size *int `json:"size,omitempty" db:"size"`
}

func (b *BaseEntity) GetSize() *int      { return b.Size }
func (b *BaseEntity) SetSize(s *int)     { b.Size = s }
```

Add `GetSize` and `SetSize` to the `Entity` interface (lines 11–26) so cross-cutting code (e.g., template rendering, future analytics) can read sizes polymorphically:

```go
type Entity interface {
    // ... existing methods ...
    GetSize() *int
    SetSize(s *int)
}
```

**Backward-compat verification**: `BaseEntity` is embedded by `Epic`, `Feature`, `Task`, `Bug`, `ChangeCard`, `TechDebt`. The compile-time interface checks at the bottom of `entity.go` (`var _ Entity = (*Epic)(nil)`, etc.) will catch any entity that fails to inherit the new accessors via embedding — none should fail because `BaseEntity` provides them.

`Idea` does **not** embed `BaseEntity` (it pre-dates the polymorphic refactor — see `internal/models/idea.go`). For Idea, add `Size *int` directly to the struct and define `GetSize`/`SetSize` methods on `*Idea`. Since `Idea` is not in the `Entity` interface check list, no interface conformance work is needed beyond the struct field.

#### Per-entity `.Validate()` updates

For each of the six affected entity types, extend `Validate()` to call `ValidateSize` when `Size != nil`. Follow the existing pattern from `Task.Validate` (`internal/models/task.go:49-73`):

```go
if t.Size != nil {
    if err := ValidateSize(*t.Size); err != nil {
        return err
    }
}
```

### 3.4 Repository Layer

For each of the six repositories listed in §3.1, the change is mechanical:

1. **`INSERT` statements**: add `size` to the column list and bind `entity.Size` (`*int` — `database/sql` handles nil → NULL automatically).
2. **`SELECT` statements (all variants)**: add `size` to the projected columns; add `&entity.Size` to the `Scan` call.
3. **`UPDATE` statements**: add `size = ?` to the `SET` clause and bind `entity.Size`.

Reference pattern: see `internal/repository/feature/repository.go:50-77` (Create) and `:92-115` (GetByID). The `execution_order` field already follows this exact `*int` pattern, so the diff for `size` is one-line additions in each statement.

**Test verification**: every existing repository test that creates and reads back an entity will exercise the new column (Size will be nil and the existing assertions will pass unchanged). Add one focused test per repository: `TestXRepository_SizeRoundTrip` that creates with `Size = ptr(5)`, reads back, asserts equality, updates to `ptr(8)`, reads back, asserts; sets to nil, reads back, asserts.

### 3.5 Service Layer

Each of the six entity services (`task_service`, `feature_service`, `epic_service`, `bug_service`, `change_card_service`, `idea_service`) needs:

1. **DTO addition**: `CreateXInput.Size *int` and `UpdateXInput.Size *int` (the `*int` type is intentional — distinguishes "not provided" from "set to NULL"). For `Update`, an additional `ClearSize bool` field is needed to disambiguate "no change" from "clear to NULL" (mirrors how the existing `--execution-order=-1` sentinel works for execution order).
2. **Service body**: in `CreateX`, copy `input.Size` onto the model before calling `repo.Create`. In `UpdateX`, when `input.ClearSize` is true, set `model.Size = nil`; else when `input.Size != nil`, set `model.Size = input.Size`; else leave unchanged. Then call `repo.Update`.
3. **Validation**: model `.Validate()` already covers size (per §3.3). Services do not need additional business validation — there is no workflow gate (OOS-2).

Pattern reference: `TaskService.CreateTask` and `TaskService.UpdateTask` in `internal/services/task_service.go` (the existing input DTO pattern at `internal/services/task_dto.go`).

### 3.6 CLI Layer

#### Flag definition (one per command)

Add to each of the 12 commands (6 create + 6 update):

```go
cmd.Flags().StringVar(&sizeFlag, "size", "",
    "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove on update)")
```

**Why `StringVar` not `IntVar`**: the flag must accept either form (`5` or `L`). Parsing happens in the command handler via `models.ParseSize`. This matches the spec's "both forms accepted on input" decision (REQ-F-003).

#### Command handler logic

Create command:

```go
var sizePtr *int
if sizeFlag != "" {
    n, err := models.ParseSize(sizeFlag)
    if err != nil {
        return fmt.Errorf("invalid --size value: %w", err)
    }
    sizePtr = &n
}
input := services.CreateTaskInput{
    // ... existing fields ...
    Size: sizePtr,
}
```

Update command:

```go
var sizePtr *int
clearSize := false
if sizeFlag == "clear" {
    clearSize = true
} else if sizeFlag != "" {
    n, err := models.ParseSize(sizeFlag)
    if err != nil {
        return fmt.Errorf("invalid --size value: %w", err)
    }
    sizePtr = &n
}
input := services.UpdateTaskInput{
    // ... existing fields ...
    Size:      sizePtr,
    ClearSize: clearSize,
}
```

**Note on the `clear` literal**: this design avoids the awkwardness of using a sentinel like `-1` for an int-valued flag. `clear` is documented in the flag help text and is unambiguous because no canonical size value collides with the string `"clear"`.

#### Output rendering

JSON output is automatic — `BaseEntity.Size` already has the `json:"size,omitempty"` tag. The `omitempty` causes `null`/missing in JSON when `Size` is nil; the `--field size` extractor (REQ-F-007) handles that by exiting code 4, matching existing behavior.

Human/table output uses a small helper (in `internal/cli/commands/` or `internal/formatters/`):

```go
func formatSize(s *int) string {
    if s == nil {
        return "—"
    }
    label, err := models.SizeLabel(*s)
    if err != nil {
        return fmt.Sprintf("%d", *s) // defensive: should never trigger
    }
    return fmt.Sprintf("%s (%d)", label, *s)
}
```

Add a `Size` column to entity-list tables and a `Size:` line to entity-detail views. (See `internal/formatters/` for existing patterns.)

### 3.7 Template Layer

`internal/config/template/helpers.go` currently populates per-entity placeholders in helpers like `featurePlaceholders` (around line 370) and `taskPlaceholders` (around line 480). Each of these reads `entity.Metadata["complexity_tier"]` for backward compat (REQ-F-012).

The size placeholders are added alongside, **independently** of `complexity_tier`:

```go
// Populate size and size_label from the typed Size field.
if feature.Size != nil {
    placeholders["size"] = strconv.Itoa(*feature.Size)
    if label, err := models.SizeLabel(*feature.Size); err == nil {
        placeholders["size_label"] = label
    } else {
        placeholders["size_label"] = ""
    }
} else {
    placeholders["size"] = ""
    placeholders["size_label"] = ""
}
```

A shared helper `applySizePlaceholders(size *int, placeholders map[string]string)` extracted into the same file removes duplication across the per-entity helpers. Add it once and call from each entity helper.

**Template content updates** (in `shark-templates/`): optional in this feature — the placeholders exist whether templates use them or not. If we want them visible by default, update `shark-templates/feature_short/*.tmpl` to include `Size: {{ .size_label }} ({{ .size }})` in the frontmatter or display block. This is a copy-paste change mirroring how `complexity_tier` is currently surfaced.

### 3.8 Key Technical Decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | **Store as numeric, accept both forms on input** | Verbatim from the user decision in `feature.md` lines 30–32. Canonical numeric storage avoids casing/whitespace ambiguity and supports future arithmetic (rollups, OOS-4). The label layer is a UX convention, not a storage concern. |
| D2 | **Nullable column, no `NOT NULL` constraint** | Progressive adoption per the feature description. Existing rows must continue to validate. Required-on-create gates are deferred to OOS-5 and configurable per entity type when added. |
| D3 | **No `CHECK` constraint in SQL; validation in `models.ValidateSize`** | Aligns with `.claude/rules/go/patterns.md` two-level validation. Lets the canonical set evolve without a schema migration. Preserves the existing pattern (e.g., status validity is not a CHECK either). |
| D4 | **`StringVar` for the `--size` flag** | The flag accepts both numeric and label forms. Cobra's `IntVar` would reject `"L"`. Parsing is done in the handler via `models.ParseSize`. |
| D5 | **`clear` literal for "set to NULL" on update** | Avoids reusing a numeric sentinel (e.g., `0` or `-1`) that could collide with a future canonical size value. Documented in flag help text. Matches the spirit of `--execution-order=-1` but cleaner since size has no meaningful negative values. |
| D6 | **`complexity_tier` and `size` coexist for one release** | Verbatim from the feature description. Templates that already use `{{ .complexity_tier }}` continue to work. Deprecation announcement in CHANGELOG; removal is a follow-up (OOS-1). |
| D7 | **Add `GetSize`/`SetSize` to the `Entity` interface** | Symmetry with `GetContextData`/`SetContextData`. Enables the shared template helper (§3.7) to be entity-agnostic. Compile-time checks at `entity.go:79-86` enforce conformance. |
| D8 | **`Idea` gets the field on its own struct, not via `BaseEntity`** | `Idea` does not embed `BaseEntity` (predates the refactor — `internal/models/idea.go:21-39`). Touching that embedding is out of scope for this feature; we add the field directly to `Idea` and define accessors on `*Idea`. |
| D9 | **TechDebt excluded** | No CLI surface exists for tech-debt today (OOS-3). Adding the model field and column without a usable surface adds dead code. Defer until tech-debt CLI lands. |
| D10 | **No new index on `size`** | No query pattern in scope reads or filters by size. Adding an index would be premature optimization. Revisit with rollups (OOS-4). |
| D11 | **Size is a feature-level attribute everywhere; no special casing per entity type** | Consistency. Mirrors E28's approach to tags. The triage scoring noted "consistency across files" as the dominant risk; uniform behavior reduces the surface area for inconsistency bugs. |

### 3.9 Integration Points (Concrete File Paths)

For implementers:

- **Models (Go structs and validation)**:
  - `internal/models/entity.go:36` — add `Size *int` to `BaseEntity`
  - `internal/models/entity.go:11-26` — add `GetSize`/`SetSize` to `Entity` interface
  - `internal/models/size.go` — **new file** with `ParseSize`, `SizeLabel`, `ValidateSize`, `ErrInvalidSize`
  - `internal/models/{epic,feature,task,bug,change_card}.go` — extend `Validate()` to call `ValidateSize` when `Size != nil`
  - `internal/models/idea.go:21-39` — add `Size *int` directly to struct; add accessors; extend `Validate()`

- **Database / migration**:
  - `internal/db/db.go:438` — bump `CurrentSchemaVersion` to `15`
  - `internal/db/db.go` (new function near `migrateAddTagsAndEntityTags` at line `3290`) — `migrateAddSizeColumns`
  - `internal/db/db.go:508` (`runMigrations`) — call `migrateAddSizeColumns` (after the existing tag migration call)

- **Repositories** (one-line additions to INSERT/SELECT/UPDATE):
  - `internal/repository/epic/repository.go`
  - `internal/repository/feature/repository.go`
  - `internal/repository/<task>/repository.go` (locate the task repo package — search `INSERT INTO tasks`)
  - `internal/repository/bug/repository.go`
  - `internal/repository/changecard/repository.go`
  - `internal/repository/idea/repository.go`

- **Services**:
  - `internal/services/task_service.go` and `internal/services/task_dto.go`
  - `internal/services/feature_service.go` and `internal/services/feature_dto.go`
  - `internal/services/epic_service.go` and `internal/services/epic_dto.go`
  - `internal/services/bug_service.go` and `internal/services/bug_dto.go`
  - `internal/services/change_service.go` and `internal/services/change_dto.go`
  - `internal/services/idea_service.go` and `internal/services/idea_dto.go`
  - DTO additions: `Size *int`; on Update DTOs also `ClearSize bool`

- **CLI commands** (parse `--size`, call service):
  - `internal/cli/commands/epic.go`
  - `internal/cli/commands/feature.go`
  - `internal/cli/commands/task.go`
  - `internal/cli/commands/bug.go`
  - `internal/cli/commands/change.go`
  - `internal/cli/commands/idea.go`

- **Templates / placeholders**:
  - `internal/config/template/helpers.go` — add shared `applySizePlaceholders` helper; call from each per-entity placeholder builder (lines `~370` for feature, `~480` for task; add equivalent for epic/bug/change/idea if those helpers exist, else add new helpers)
  - `shark-templates/feature_short/*.tmpl` and similar — optional content updates to surface `{{ .size_label }} ({{ .size }})`

- **Documentation**:
  - `docs/cli-reference/feature-commands.md`, `task-commands.md`, `epic-commands.md`, `bug-commands.md`, `change-commands.md`, `idea-commands.md` — add `--size` flag documentation
  - `docs/cli-reference/global-flags.md` — no change (flag is per-entity-command, not global)
  - `CHANGELOG.md` — announce `--size` flag and `complexity_tier` deprecation

### 3.10 Testing Strategy (Outline — full plan in test plan node)

Per `.claude/rules/testing/architecture.md`:

- **Model tests** (`internal/models/size_test.go`, **new**): table-driven coverage of `ParseSize` (all valid forms, all invalid forms, whitespace, casing, empty, oversized, special chars), `SizeLabel` (all six values + invalid), `ValidateSize` (canonical set + boundaries).
- **Per-entity model tests** (extend existing `*_test.go`): assert `Validate()` rejects `Size = ptr(4)` and accepts `Size = nil` and `Size = ptr(5)`.
- **Repository tests** (`internal/repository/<entity>/repository_test.go`, extend each): one new test per entity, `TestXRepository_SizeRoundTrip`, with cleanup per `.claude/rules/testing/repository-tests.md`.
- **Service tests** (`internal/services/*_service_test.go`, extend with mocks): verify that `CreateX` passes `Size` to repo.Create; `UpdateX` with `ClearSize=true` calls repo.Update with model.Size = nil.
- **CLI tests** (`internal/cli/commands/*_test.go`, extend with mock services): verify `--size L` parses to `5`, `--size 4` returns error, `--size clear` on update sets `ClearSize=true`.
- **Migration test** (`internal/db/db_test.go`, extend or new): apply migration twice on the same DB; assert idempotency. Use seeded fixture and time the migration to satisfy REQ-NF-001.
- **Template test** (`internal/config/template/helpers_test.go`, extend): given an entity with `Size = ptr(5)`, assert `placeholders["size"] == "5"` and `placeholders["size_label"] == "L"`; with `Size = nil`, assert both are `""`. Verify `complexity_tier` still works independently (REQ-F-012).

### 3.11 Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Inconsistent column adoption — one of the 6 repositories' SELECTs forgets `size` and the field returns nil silently | Medium | Medium | The `TestXRepository_SizeRoundTrip` per entity catches this. Add a `make lint`-time grep guard or a meta-test that introspects every entity-table SELECT and asserts `size` is in the projection (out-of-scope nice-to-have). |
| Idea diverges from the BaseEntity pattern, future-proofing concerns | Low | Low | Acknowledged via D8. Idea's separate accessor implementation is documented in code comments; future BaseEntity-adoption refactor of Idea is a follow-up unrelated to this feature. |
| User confusion between `--size` (this feature) and `--priority` / `--execution-order` (existing) | Medium | Low | Flag help text makes the distinction explicit. Documentation update lists all sizing/ordering attributes side-by-side. |
| `complexity_tier` users expect `size` to mirror their values automatically | Medium | Low | OOS-6 explicitly defers automatic migration. CHANGELOG note advises users to backfill `--size` deliberately. The fallback (D6) means existing `complexity_tier` continues to work in templates. |
| Migration runs against a Turso database with `skip_migrations: true` and the schema-version bump is missed | Low | High (data inconsistency between deployments) | The schema-version bump (REQ-F-009) is the single required step per `.claude/rules/database-critical.md`. The migration checklist there is followed. The optional belt-and-braces toggle is documented for one-off recovery. |

---

## 4. Exit Gate Checklist

- [x] Every requirement is testable (each REQ has an explicit "Testable" clause or appears in §2.3 acceptance scenarios)
- [x] Every architecture decision references existing patterns or explains deviation (see §3.8 D1–D11; pattern references throughout §3.4–§3.7)
- [x] File paths listed for all changes (§3.1 inventory and §3.9 concrete paths)
- [x] No TBDs in critical sections (Requirements, Architecture, Data Model, Migration, Decisions)
- [x] Out-of-scope items explicit (§2.4 OOS-1 through OOS-7) and traced to feature description where applicable
- [x] Backward compatibility addressed (REQ-NF-002, REQ-F-012, D2, D6, Scenario 6)
- [x] Migration strategy follows `.claude/rules/database-critical.md` (schema version bump + idempotent migration; §3.2 and Risks)
- [x] Aligns with two-level validation rule (§3.3 model layer; D3)
- [x] Aligns with input sanitization rules (REQ-NF-003; §3.3 helper design)
- [x] Aligns with service layer pattern (§3.5; new `--size` flow goes CLI → service → repo, no fat controller)

---

*Specification complete — ready for test planning.*
