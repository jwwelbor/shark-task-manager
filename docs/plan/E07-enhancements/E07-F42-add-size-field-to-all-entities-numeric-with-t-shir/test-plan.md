---
feature_key: E07-F42
title: "Test Plan: Add Size field to all entities (numeric with t-shirt mapping)"
status: draft
last_updated: 2026-04-25
spec_ref: spec.md
---

# E07-F42 Test Plan: Size Field on All Entities

**Feature Key**: E07-F42
**Spec**: [spec.md](spec.md)
**Test Architect**: QA Agent

---

## 1. AC Test Matrix

Every acceptance criterion from `spec.md §2.1` and every acceptance scenario from `§2.3` is mapped to at least one test case below.

---

### REQ-F-001 — `Size` field on `BaseEntity`

**AC**: A `BaseEntity` with `Size = nil` round-trips through JSON marshal/unmarshal and DB round-trip without error; `GetSize()` returns nil; `Size = ptr(5)` round-trips and `GetSize()` returns the same value.

#### TC-F001-A: nil Size round-trips through JSON
- **Test type**: Unit (model layer, no DB)
- **File**: `internal/models/entity_test.go` (extend existing)
- **Setup**: Construct a `BaseEntity{}` with `Size` unset (nil)
- **Steps**: `json.Marshal`, then `json.Unmarshal` into a new struct
- **Expected**: No error; unmarshalled `Size` is nil; `GetSize()` returns nil
- **Edge cases**: Confirm `"size"` key is absent from JSON output when nil (omitempty)

#### TC-F001-B: ptr(5) Size round-trips through JSON
- **Test type**: Unit (model layer)
- **File**: `internal/models/entity_test.go`
- **Setup**: `BaseEntity{Size: ptr(5)}`
- **Steps**: Marshal to JSON, unmarshal back
- **Expected**: `GetSize()` returns pointer to `5`; JSON contains `"size":5`

#### TC-F001-C: `SetSize` mutates the field
- **Test type**: Unit
- **File**: `internal/models/entity_test.go`
- **Setup**: `BaseEntity{}`; call `SetSize(ptr(3))`
- **Expected**: `GetSize()` now returns pointer to `3`; call `SetSize(nil)` → `GetSize()` returns nil

---

### REQ-F-002 — Canonical numeric values

**AC**: `ValidateSize(int)` accepts `{1, 2, 3, 5, 8, 13}` and rejects everything else.

#### TC-F002-A: All valid canonical values accepted (table-driven)
- **Test type**: Unit
- **File**: `internal/models/size_test.go` (new file)
- **Setup**: None
- **Steps**: Call `ValidateSize(n)` for each of `{1, 2, 3, 5, 8, 13}`
- **Expected**: All return `nil`

#### TC-F002-B: Invalid numeric values rejected (table-driven)
- **Test type**: Unit
- **File**: `internal/models/size_test.go`
- **Inputs**: `{0, 4, 6, 7, 9, 10, 11, 12, 14, -1, -8, 100, 21}`
- **Expected**: Each returns `ErrInvalidSize` (via `errors.Is`)

#### TC-F002-C: Error wraps `ErrInvalidSize` sentinel
- **Test type**: Unit
- **File**: `internal/models/size_test.go`
- **Steps**: Call `ValidateSize(4)`, assert `errors.Is(err, models.ErrInvalidSize)`
- **Expected**: `true`

---

### REQ-F-003 — T-shirt label mapping (bidirectional)

**AC**: `ParseSize` handles all valid label forms (case-insensitive, whitespace-trimmed), all valid numeric strings, and rejects unknown values. `SizeLabel` maps numeric to label.

#### TC-F003-A: ParseSize — all valid label forms (table-driven)
- **Test type**: Unit
- **File**: `internal/models/size_test.go`
- **Inputs and expected**:

| Input | Expected value |
|-------|----------------|
| `"XS"` | `1` |
| `"xs"` | `1` |
| `"Xs"` | `1` |
| `" XS "` | `1` (whitespace stripped) |
| `"S"` | `2` |
| `"M"` | `3` |
| `"L"` | `5` |
| `"XL"` | `8` |
| `"xl"` | `8` |
| `"XXL"` | `13` |
| `"xxl"` | `13` |

- **Expected**: Each returns `(canonical_int, nil)`

#### TC-F003-B: ParseSize — all valid numeric strings (table-driven)
- **Test type**: Unit
- **File**: `internal/models/size_test.go`
- **Inputs**: `"1"`, `"2"`, `"3"`, `"5"`, `"8"`, `"13"`, `" 5 "` (leading/trailing space)
- **Expected**: Each returns `(n, nil)`

#### TC-F003-C: ParseSize — invalid inputs rejected (table-driven)
- **Test type**: Unit
- **File**: `internal/models/size_test.go`
- **Inputs**: `"XXXL"`, `"4"`, `"medium"`, `""`, `"0"`, `"14"`, `"-1"`, `"abc"`, `" "`, `"1.5"`, `"'; DROP TABLE epics; --"`, string of length 21 (oversized)
- **Expected**: Each returns non-nil error wrapping `ErrInvalidSize`

#### TC-F003-D: SizeLabel — all valid numerics map to labels (table-driven)
- **Test type**: Unit
- **File**: `internal/models/size_test.go`
- **Inputs and expected**:

| Input | Expected label |
|-------|----------------|
| `1` | `"XS"` |
| `2` | `"S"` |
| `3` | `"M"` |
| `5` | `"L"` |
| `8` | `"XL"` |
| `13` | `"XXL"` |

- **Expected**: Each returns `(label, nil)`

#### TC-F003-E: SizeLabel — invalid numeric returns error
- **Test type**: Unit
- **File**: `internal/models/size_test.go`
- **Inputs**: `0`, `4`, `6`, `14`, `-1`
- **Expected**: `("", error)` with `errors.Is(err, ErrInvalidSize)`

#### TC-F003-F: ParseSize/SizeLabel round-trip (table-driven)
- **Test type**: Unit
- **File**: `internal/models/size_test.go`
- **Steps**: For each valid label, `ParseSize(label)` → `SizeLabel(result)` → compare
- **Expected**: Original label (upper-cased canonical form) is recovered

---

### REQ-F-004 — `--size` flag on all 6 `create` commands

**AC**: Each create command exits 0 with `--size 5`, `--size L`, and no flag; exits non-zero with `--size 4` or `--size XXXL`. Persisted row has canonical numeric value.

#### TC-F004-A: `--size` flag registered on all 6 create commands (table-driven)
- **Test type**: Unit (CLI flag inspection, no DB)
- **File**: `internal/cli/commands/*_test.go` (one subtest per entity in a table)
- **Setup**: Inspect `epicCreateCmd`, `featureCreateCmd`, `taskCreateCmd`, `bugCreateCmd`, `changeCreateCmd`, `ideaCreateCmd`
- **Steps**: `cmd.Flags().Lookup("size")`
- **Expected**: Flag is non-nil for all 6; default value is `""`

#### TC-F004-B: Valid label form accepted by all 6 create commands
- **Test type**: CLI (mock service)
- **File**: `internal/cli/commands/*_test.go`
- **Setup**: Mock service that captures `CreateXInput.Size`
- **Steps**: Parse `--size L` flag, call handler
- **Expected**: Mock receives `Size = ptr(5)`; command returns nil error

#### TC-F004-C: Valid numeric form accepted by all 6 create commands
- **Test type**: CLI (mock service)
- **Steps**: Parse `--size 8`, call handler
- **Expected**: Mock receives `Size = ptr(8)`

#### TC-F004-D: Flag absent → Size nil on all 6 create commands
- **Test type**: CLI (mock service)
- **Steps**: Call handler with no `--size` flag
- **Expected**: Mock receives `Size = nil`

#### TC-F004-E: Invalid value causes create to fail (non-zero exit)
- **Test type**: CLI (mock service)
- **Inputs**: `--size 4`, `--size XXXL`, `--size medium`
- **Expected**: Handler returns non-nil error before calling service; error message mentions allowed values

---

### REQ-F-005 — `--size` flag on all 6 `update` commands

**AC**: Update with `--size L` persists 5; update with `--size clear` sets to NULL; flag absent makes no change.

#### TC-F005-A: `--size` flag registered on all 6 update commands
- **Test type**: Unit (CLI flag inspection)
- **File**: `internal/cli/commands/*_test.go`
- **Expected**: Flag is non-nil, default `""`

#### TC-F005-B: `--size <valid>` on update passes correct value to service
- **Test type**: CLI (mock service)
- **Steps**: Parse `--size XL`, call update handler
- **Expected**: Service `UpdateXInput.Size = ptr(8)`, `ClearSize = false`

#### TC-F005-C: `--size clear` on update sets `ClearSize = true`
- **Test type**: CLI (mock service)
- **Steps**: Parse `--size clear`, call update handler
- **Expected**: Service `UpdateXInput.ClearSize = true`, `Size = nil`

#### TC-F005-D: Flag absent → no size mutation (sentinel behavior)
- **Test type**: CLI (mock service)
- **Steps**: Call update handler with no `--size` flag
- **Expected**: `UpdateXInput.Size = nil`, `ClearSize = false`

#### TC-F005-E: `--size clear` case-insensitivity (if applicable per spec)
- **Test type**: CLI
- **Note**: The spec defines `clear` as the exact literal. Test that only the exact lowercase string `"clear"` is treated as the clear sentinel; `"Clear"` and `"CLEAR"` should produce a `ParseSize` parse error (they are not valid size values).
- **Expected**: `"Clear"` → error; `"clear"` → `ClearSize = true`

---

### REQ-F-006 — Output renders both forms

**AC**: JSON output contains `"size": <int> | null`; human output shows `<label> (<num>)` or `—`.

#### TC-F006-A: JSON output contains numeric size field
- **Test type**: Unit
- **File**: `internal/models/entity_test.go` or integration
- **Setup**: Entity with `Size = ptr(3)`
- **Steps**: `json.Marshal(entity)`
- **Expected**: JSON contains `"size":3`

#### TC-F006-B: JSON output omits size when nil (omitempty)
- **Setup**: Entity with `Size = nil`
- **Expected**: JSON does NOT contain `"size"` key (omitempty behavior)

#### TC-F006-C: `formatSize` helper — non-nil returns `"<label> (<num>)"`
- **Test type**: Unit
- **File**: `internal/cli/commands/` or `internal/formatters/` (wherever `formatSize` lives)
- **Inputs**: `ptr(1)` → `"XS (1)"`, `ptr(5)` → `"L (5)"`, `ptr(13)` → `"XXL (13)"`
- **Expected**: Exact format string

#### TC-F006-D: `formatSize` helper — nil returns `"—"`
- **Input**: `nil`
- **Expected**: `"—"`

---

### REQ-F-007 — `--field size` extraction

**AC**: `shark get <key> --field size` returns the numeric form; `--field size_label` returns the label; both exit code 4 when Size is NULL.

#### TC-F007-A: `--field size` returns numeric for sized entity
- **Test type**: Integration (CLI + real DB)
- **Note**: This is an E2E test exercised via the existing `--field` extraction path; verify the JSON contract in unit tests first
- **Unit coverage**: JSON marshal of `BaseEntity{Size: ptr(5)}` produces `"size":5`
- **Expected behavior**: `--field size` extracts `5`

#### TC-F007-B: `--field size` on entity with nil Size exits code 4
- **Test type**: Integration (CLI behavior)
- **Expected**: Exit code 4 (matches existing `--field` behavior for null/missing fields)

#### TC-F007-C: `--field size_label` returns label string for sized entity
- **Expected behavior**: `--field size_label` extracts `"L"` when size is 5

---

### REQ-F-008 — `size` column on all six entity tables

**AC**: `PRAGMA table_info` lists `size INTEGER` for each of the six tables; existing rows have NULL after migration.

#### TC-F008-A: Migration adds `size` column to all six tables (table-driven)
- **Test type**: Repository/DB integration (real DB via `test.GetTestDB()`)
- **File**: `internal/db/db_test.go` (extend)
- **Setup**: `InitDB` on a fresh temp DB
- **Steps**: For each of `{epics, features, tasks, bugs, change_cards, ideas}`:
  - `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'size'`
- **Expected**: Count is 1 for each table

#### TC-F008-B: Existing rows have NULL size after migration
- **Test type**: Repository/DB integration
- **Setup**: Seed one row in each of the six tables without providing a size value; apply migration
- **Steps**: `SELECT size FROM <table> WHERE id = <seeded_id>`
- **Expected**: Result is NULL for each

---

### REQ-F-009 — Schema version bump + idempotent migration

**AC**: Fresh DB applies migration once; already-migrated DB is a no-op; migration completes without error.

#### TC-F009-A: Migration idempotency — applying twice produces no error
- **Test type**: DB integration
- **File**: `internal/db/db_test.go` (extend, mirroring `TestMigration_RejectionReason`)
- **Setup**: `InitDB` on a temp DB (applies migration once)
- **Steps**: Call `ApplySchemaAndMigrations` (or `migrateAddSizeColumns` directly) a second time
- **Expected**: No error; `pragma_table_info` still returns exactly one `size` column per table

#### TC-F009-B: `CurrentSchemaVersion` is 15 after migration
- **Test type**: DB integration
- **File**: `internal/db/db_test.go`
- **Steps**: `InitDB`; query `SELECT schema_version FROM schema_version` (or equivalent version table)
- **Expected**: Version is ≥ 15

#### TC-F009-C: Migration completes within 1 second on 10,000-row DB (REQ-NF-001)
- **Test type**: DB performance integration
- **File**: `internal/db/db_test.go`
- **Setup**: Seed 10,000 rows spread across the six affected tables (e.g., 2,000 tasks + proportional others)
- **Steps**: Time the migration; record duration in test output via `t.Logf`
- **Expected**: Duration < 1 second; test fails with logged time if exceeded
- **Note**: This is a benchmark guard, not a strict benchmark. Use `testing.T` not `testing.B`.

---

### REQ-F-010 — `size` round-trips through every CRUD path

**AC**: For each of the 6 entity types, create with size, read back, assert 5; update to 1, read back, assert 1; set to nil, read back, assert nil.

#### TC-F010-{A..F}: `TestXRepository_SizeRoundTrip` for each entity (6 tests)
- **Test type**: Repository integration (real DB)
- **Files**: One test per entity, in the entity's `repository_test.go`:
  - `internal/repository/epic/repository_test.go`
  - `internal/repository/feature/repository_test.go`
  - `internal/repository/task/repository_test.go`
  - `internal/repository/bug/repository_test.go`
  - `internal/repository/changecard/repository_test.go` (add if missing)
  - `internal/repository/idea/repository_test.go`
- **Setup**: `test.GetTestDB()`, create parent entities as needed, clean `DELETE` before test
- **Steps**:
  1. Create entity with `Size = ptr(5)`; assert read-back `Size == ptr(5)`
  2. Update entity with `Size = ptr(1)`; assert read-back `Size == ptr(1)`
  3. Update entity with `Size = nil`; assert read-back `Size == nil`
- **Cleanup**: `defer DELETE WHERE id = <id>`
- **Edge cases**: Verify no scan error when `Size` is NULL in database (Go `*int` scanning nil)

---

### REQ-F-011 — `{{ .size }}` and `{{ .size_label }}` template variables

**AC**: With `size = 5`, `{{ .size }}` = `"5"` and `{{ .size_label }}` = `"L"`; with nil, both render as `""`.

#### TC-F011-A: Template placeholders populated correctly for each entity type (table-driven)
- **Test type**: Unit
- **File**: `internal/config/template/helpers_test.go` (extend, following `TestTaskPlaceholders_AllFields`)
- **Setup**: For each entity type, construct a model instance with `Size = ptr(5)`
- **Steps**: Call `XPlaceholders(entity)`; inspect `m["size"]` and `m["size_label"]`
- **Expected**: `m["size"] == "5"`, `m["size_label"] == "L"`

#### TC-F011-B: Template placeholders are empty string when size is nil
- **Test type**: Unit
- **File**: `internal/config/template/helpers_test.go`
- **Setup**: Entity with `Size = nil`
- **Expected**: `m["size"] == ""`, `m["size_label"] == ""`

#### TC-F011-C: `applySizePlaceholders` shared helper tested in isolation
- **Test type**: Unit
- **File**: `internal/config/template/helpers_test.go`
- **Setup**: Call `applySizePlaceholders(ptr(8), m)` and `applySizePlaceholders(nil, m)`
- **Expected**: With ptr(8): `m["size"] == "8"`, `m["size_label"] == "XL"`; with nil: both `""`

---

### REQ-F-012 — `complexity_tier` fallback preserved

**AC**: Entity with `complexity_tier="L"` in Metadata and `Size=nil` still renders `complexity_tier="L"`; same entity with `Size=ptr(8)` renders both surfaces populated independently.

#### TC-F012-A: `complexity_tier` not overwritten by size field
- **Test type**: Unit
- **File**: `internal/config/template/helpers_test.go`
- **Setup**: Task with `Metadata["complexity_tier"] = "L"`, `Size = nil`
- **Steps**: Call `TaskPlaceholders(task)`
- **Expected**: `m["complexity_tier"] == "L"`; `m["size"] == ""` (both correct independently)

#### TC-F012-B: Both surfaces populated when both are set
- **Setup**: Task with `Metadata["complexity_tier"] = "L"`, `Size = ptr(8)`
- **Expected**: `m["complexity_tier"] == "L"`, `m["size"] == "8"`, `m["size_label"] == "XL"`

---

### Acceptance Scenario Coverage

#### AC-Scenario-1: Create with t-shirt label → persists as numeric
- **Covered by**: TC-F004-B (CLI flag parsing) + TC-F010-A (repo round-trip)
- **Service test**: TC-SVC-A below confirms service passes `Size = ptr(5)` to repo when `ParseSize("L")` is called

#### AC-Scenario-2: Create with numeric value → human display shows label
- **Covered by**: TC-F004-C (CLI) + TC-F006-C (`formatSize`) + TC-F010-A (repo)

#### AC-Scenario-3: Reject invalid size
- **Covered by**: TC-F004-E (CLI exits non-zero) + TC-F002-B (ValidateSize) + TC-F003-C (ParseSize)
- **Additional**: Assert no DB row created — verified by TC-F004-E (handler returns error before calling service)

#### AC-Scenario-4: Update size, then clear it
- **Covered by**: TC-F005-B (service receives update value) + TC-F005-C (service receives ClearSize=true) + TC-F010-A steps 2–3 (repo round-trip for both)

#### AC-Scenario-5: Migration is idempotent
- **Covered by**: TC-F009-A

#### AC-Scenario-6: Backward compatibility — NULL size reads cleanly
- **Covered by**: TC-F008-B (existing rows have NULL after migration) + TC-F006-B (JSON omits field) + TC-F006-D (human renders `—`) + REQ-NF-002 test (make test passes)

#### AC-Scenario-7: Template variables
- **Covered by**: TC-F011-A + TC-F011-B

---

## 2. Integration Scenarios

### Integration Scenario IS-1: CLI → Service → Repository for `create --size`

**Components interacting**: CLI command handler → `CreateXInput` DTO → Service `CreateX` → Repository `Create`

**What to verify at boundaries**:
1. CLI parses `--size L` → `ParseSize("L")` → `ptr(5)` → stored in `CreateXInput.Size`
2. Service reads `input.Size`, sets `entity.Size = input.Size`, calls `repo.Create`
3. Repository binds `entity.Size` as `?` parameter in `INSERT ... size = ?`
4. Read-back via `repo.GetByKey` scans `size` column into `entity.Size`

**Test file**: Service test (`internal/services/*_service_test.go`) with mock repo capturing `entity.Size`

**Epic UAT contribution**: This feature has no explicit epic-level UAT plan (E07 is a placeholder epic). The integration scenario substitutes — it verifies the end-to-end data flow for the primary happy path across the full stack.

---

### Integration Scenario IS-2: Update flow including `clear` sentinel

**Components**: CLI update handler → `UpdateXInput{Size: nil, ClearSize: true}` → Service `UpdateX` → Repository `Update`

**What to verify**:
1. `--size clear` literal → sets `ClearSize = true`, `Size = nil` in DTO
2. Service detects `ClearSize == true` → sets `model.Size = nil` before calling `repo.Update`
3. Repository executes `UPDATE ... size = NULL` (SQLite accepts nil `*int` as NULL)
4. Read-back returns `Size == nil`

**Test file**: Service test + TC-F010-A step 3 (repo integration)

---

### Integration Scenario IS-3: Schema migration on existing populated database

**Components**: `internal/db/db.go` migration machinery → existing rows in six tables

**What to verify**:
1. Migration detects missing `size` column via `pragma_table_info`
2. Executes `ALTER TABLE ... ADD COLUMN size INTEGER NULL`
3. Existing rows are unaffected (no data loss, `size` = NULL)
4. Subsequent `SELECT` queries that include `size` scan NULL correctly into `*int` fields

**Test file**: `internal/db/db_test.go` — TC-F008-B, TC-F009-A

---

### Integration Scenario IS-4: Template rendering with size data

**Components**: `internal/config/template/helpers.go` → `applySizePlaceholders` → template substitution

**What to verify**:
1. Feature/task/epic/bug/change/idea placeholders all populate `"size"` and `"size_label"`
2. `complexity_tier` is unaffected (independent map entry)
3. Nil size produces empty-string placeholders (templates render cleanly with no `<nil>`)

**Test file**: `internal/config/template/helpers_test.go` — TC-F011-A through TC-F012-B

---

### Integration Scenario IS-5: Backward compatibility — no regressions

**Components**: Entire existing test suite

**What to verify**:
1. All existing tests pass after `make test` with no modifications other than (a) tests asserting create/update behavior that now get `size: null` in JSON, and (b) snapshot tests that include the new `"size": null` field
2. All existing `shark task get`, `shark feature get`, etc. commands work without `--size` flag

**Test execution**: `make fmt && make lint && make test` — must be green

---

## 3. Service Layer Tests

These supplement the integration scenarios with focused mock-based coverage per service (no real DB).

### TC-SVC-A: Service `CreateX` propagates `Size` to repository (table-driven, 6 entity services)

- **Test type**: Service unit (mock repo)
- **File**: `internal/services/*_service_test.go` (extend existing or add `size` sub-tests)
- **Setup**: Mock repo with `createFn` that captures the entity passed to it
- **Input**: `CreateXInput{..., Size: ptr(5)}`
- **Expected**: Captured `entity.Size == ptr(5)`

### TC-SVC-B: Service `UpdateX` — `ClearSize = true` sets model.Size = nil

- **Setup**: Mock repo with `updateFn` capturing entity
- **Input**: `UpdateXInput{..., ClearSize: true}`
- **Expected**: Captured `entity.Size == nil`

### TC-SVC-C: Service `UpdateX` — `Size = ptr(8)` updates field

- **Input**: `UpdateXInput{..., Size: ptr(8), ClearSize: false}`
- **Expected**: Captured `entity.Size == ptr(8)`

### TC-SVC-D: Service `UpdateX` — neither flag set → no change to size

- **Setup**: Mock repo returns existing entity with `Size = ptr(3)`; `updateFn` captures entity
- **Input**: `UpdateXInput{}` (Size nil, ClearSize false)
- **Expected**: Captured `entity.Size == ptr(3)` (unchanged)

### TC-SVC-E: Service `CreateX` passes `Size = nil` when no flag provided

- **Input**: `CreateXInput{..., Size: nil}`
- **Expected**: Captured `entity.Size == nil`

---

## 4. Input Sanitization Tests (REQ-NF-003)

### TC-SAN-A: ParseSize strips leading/trailing whitespace
- **Inputs**: `" XS "`, `" 5 "`, `"\t13\t"`
- **Expected**: Parse succeeds with correct numeric value

### TC-SAN-B: ParseSize is case-insensitive for labels
- **Inputs**: `"xs"`, `"XS"`, `"xS"`, `"Xs"`
- **Expected**: All return `(1, nil)`

### TC-SAN-C: ParseSize rejects oversized input (> 20 characters)
- **Input**: `strings.Repeat("A", 21)`
- **Expected**: Returns error wrapping `ErrInvalidSize`

### TC-SAN-D: ParseSize rejects control characters and injection payloads
- **Inputs**: `"'; DROP TABLE epics; --"`, `"1\x00"`, `"1\n"`, `"1 OR 1=1"`
- **Expected**: Each returns non-nil error (not a valid label or Fibonacci number)

### TC-SAN-E: Error messages quote user input with `%q`
- **Steps**: Trigger `ParseSize("4")`; inspect `err.Error()`
- **Expected**: Error string contains `"4"` quoted as `"4"` (not unquoted, matching the `%q` format)

---

## 5. Non-Functional Test Cases

### TC-NF-001: REQ-NF-001 — Migration runtime < 1 second
- **Test type**: DB performance integration
- **File**: `internal/db/db_test.go`
- **Covered by**: TC-F009-C

### TC-NF-002: REQ-NF-002 — Backward compatibility (`make test` passes)
- **Test type**: Full regression
- **Execution**: `make fmt && make lint && make test`
- **Expected**: Exit code 0; only test-file changes are those explicitly listed in REQ-NF-002

---

## 6. Test Infrastructure

### Existing infrastructure to leverage

| Infrastructure | Location | Used for |
|---|---|---|
| `test.GetTestDB()` | `internal/test/testdb.go` | Repository integration tests (TC-F008, TC-F009, TC-F010) |
| Table-driven test pattern | `internal/models/validation_sanitization_test.go` | Model unit tests (TC-F002, TC-F003, TC-SAN) |
| Mock repo function-field pattern | `internal/services/bug_service_test.go:16-60` | Service unit tests (TC-SVC) |
| `testify/assert` and `testify/require` | Already imported in all test files | Assertion helpers |
| `initDB` + `t.TempDir()` pattern | `internal/db/db_test.go:14-28` | Migration idempotency test (TC-F009) |
| `XPlaceholders(entity)` helper pattern | `internal/config/template/helpers_test.go:14-65` | Template placeholder tests (TC-F011) |
| `errors.Is(err, ErrInvalidX)` pattern | `internal/models/validation_sanitization_test.go:53-57` | Sentinel error assertions |

### New test utilities needed

| Utility | Location | Why needed |
|---|---|---|
| `ptr(n int) *int` helper function | Each new `*_test.go` file (or a shared `internal/test/helpers.go`) | Concise pointer creation for `Size *int` fields; avoids `func() *int { n := 5; return &n }()` inline noise |
| Performance seed helper (for TC-F009-C) | `internal/db/db_test.go` | Seed 10,000 rows across six tables; can be a local `seedLargeDB(t, db)` helper |

### New test files to create

| File | Type | Content |
|---|---|---|
| `internal/models/size_test.go` | Pure unit | All `ParseSize`, `SizeLabel`, `ValidateSize` tests (TC-F002, TC-F003, TC-SAN) |

### Existing test files to extend

| File | Extension |
|---|---|
| `internal/models/entity_test.go` | Add TC-F001-A, TC-F001-B, TC-F001-C |
| `internal/models/entity_test.go` (or per-entity test files) | Per-entity `Validate()` rejects `Size = ptr(4)`, accepts `nil` and `ptr(5)` |
| `internal/db/db_test.go` | Add TC-F008-A, TC-F008-B, TC-F009-A, TC-F009-B, TC-F009-C |
| `internal/repository/epic/repository_test.go` | Add TC-F010-A (`TestEpicRepository_SizeRoundTrip`) |
| `internal/repository/feature/repository_test.go` | Add TC-F010-B (`TestFeatureRepository_SizeRoundTrip`) |
| `internal/repository/task/repository_test.go` | Add TC-F010-C (`TestTaskRepository_SizeRoundTrip`) |
| `internal/repository/bug/repository_test.go` | Add TC-F010-D (`TestBugRepository_SizeRoundTrip`) |
| `internal/repository/changecard/aggregate_test.go` or new file | Add TC-F010-E (`TestChangeCardRepository_SizeRoundTrip`) |
| `internal/repository/idea/repository_test.go` | Add TC-F010-F (`TestIdeaRepository_SizeRoundTrip`) |
| `internal/services/*_service_test.go` (6 files) | Add TC-SVC-A through TC-SVC-E per service |
| `internal/config/template/helpers_test.go` | Add TC-F011-A, TC-F011-B, TC-F011-C, TC-F012-A, TC-F012-B |
| `internal/cli/commands/epic_create_test.go` | Add TC-F004-A through TC-F004-E (epic) |
| `internal/cli/commands/*_test.go` (5 more entity test files) | Add TC-F004-A through TC-F004-E, TC-F005-A through TC-F005-E (per entity) |

---

## 7. Exit Gate Checklist

- [x] Every AC in spec.md §2.1 (REQ-F-001 through REQ-F-012, REQ-NF-001 through REQ-NF-003) has at least one test case
- [x] Every acceptance scenario in spec.md §2.3 (Scenario 1–7) is mapped to test cases
- [x] Edge cases identified for each AC (nil vs ptr, whitespace, casing, invalid numerics, oversized input, injection payloads)
- [x] Integration scenarios cover all cross-component boundaries: CLI→Service, Service→Repository, DB migration, Template rendering, and Backward compatibility
- [x] Test patterns reference existing infrastructure (test.GetTestDB, mock function-field pattern, testify, t.TempDir)
- [x] New test utilities identified (`ptr` helper, performance seed helper)
- [x] Compliance with testing architecture rules: repo tests use real DB; service and CLI tests use mocks; model tests are pure unit
- [x] Idempotency of migration explicitly tested
- [x] Input sanitization (REQ-NF-003) explicitly tested with injection payloads
