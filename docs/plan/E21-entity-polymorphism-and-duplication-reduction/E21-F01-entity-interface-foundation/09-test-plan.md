# F01: Entity Interface Foundation -- Test Plan

**Feature**: E21-F01
**Tier**: STANDARD
**Author**: QA Agent
**Date**: 2026-03-19

This is the developer TDD guide for F01. Tests should be written before or alongside implementation, following the 4-task dependency chain: ChangeCard normalization -> Entity interface -> Adapters -> Registry.

---

## 1. Acceptance Criteria Test Matrix

### AC1: Entity Interface Defined

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC1-01: Interface has 14 methods | `internal/models/entity.go` exists | Reflection or manual inspection confirms: GetID, GetKey, GetTitle, GetSlug, GetEntityType, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt, Validate | N/A -- compile-time contract |
| TC-AC1-02: Interface file location | N/A | File is in `internal/models/`, not `internal/services/` | N/A |

### AC2: EntityType Enum Complete

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC2-01: All 5 types present | `models.EntityType` defined in `entity_note.go` | Constants exist: EntityTypeEpic, EntityTypeFeature, EntityTypeTask, EntityTypeBug, EntityTypeChange | Already defined -- verify no duplicates introduced in `entity.go` |
| TC-AC2-02: GetEntityType returns correct type | Instantiate each of 5 models | `epic.GetEntityType() == EntityTypeEpic`, etc. for all 5 | N/A |

### AC3: All Models Implement Entity Interface

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC3-01: Compile-time checks | `var _ Entity = (*Epic)(nil)` etc. in `entity.go` | `go build ./...` succeeds with zero errors | N/A -- binary pass/fail |
| TC-AC3-02: Accessor methods return correct values | Populate each model with known data | Each getter returns the populated value | Nil `*string` fields (Slug, FilePath, Description, ContextData) return `""` |
| TC-AC3-03: SetStatus mutates correctly | Call `SetStatus("new_status")` on each model | `GetStatus()` returns `"new_status"` | Empty string status |
| TC-AC3-04: SetContextData mutates correctly | Call `SetContextData(&json)` then `SetContextData(nil)` | Get returns matching value; nil round-trips correctly | Nil -> non-nil -> nil transitions |

### AC4: Backward Compatibility Preserved

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC4-01: Existing tests pass | Full codebase unchanged except F01 additions | `make test` -- 100% pass rate | N/A |
| TC-AC4-02: Direct field access works | Code like `epic.Key`, `task.FeatureID` | `make build` -- zero compile errors | N/A |
| TC-AC4-03: ChangeCard tests updated | ChangeCard tests use `*string` for Slug/FilePath | All ChangeCard tests pass with `strPtr()` helper | N/A |

### AC5: EntityRepository Interface Defined

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC5-01: Interface has 6 methods | `internal/services/entity_repository.go` exists | Methods: GetByKey, GetByID, UpdateStatus, Update, GetContextData, UpdateContextData | N/A -- compile-time contract |
| TC-AC5-02: Methods accept/return models.Entity | Inspect signatures | `GetByKey` returns `(models.Entity, error)`, `Update` accepts `models.Entity` | N/A |

### AC6: Five Adapter Implementations

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC6-01: Each adapter satisfies EntityRepository | Compile-time check: `var _ EntityRepository = (*EpicRepositoryAdapter)(nil)` etc. | `go build ./...` succeeds | N/A |
| TC-AC6-02: GetByKey delegates correctly | Mock typed repo returns known entity | Adapter's GetByKey returns same entity as `models.Entity` | Repo returns error -- adapter propagates it |
| TC-AC6-03: Update rejects wrong type | Pass `*models.Task` to `EpicRepositoryAdapter.Update` | Error message contains "expected *models.Epic, got *models.Task" | Nil entity input |
| TC-AC6-04: UpdateStatus converts typed status | Call adapter.UpdateStatus with string status | Typed repo receives correctly cast status (e.g., `EpicStatus("active")`) | Empty string status |
| TC-AC6-05: Context data via get-set-update | Bug/ChangeCard adapters lacking dedicated context methods | GetContextData retrieves via GetByID; UpdateContextData does get-set-update | Entity not found during context update |

### AC7: EntityRegistry Operational

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC7-01: Register + GetRepository | Register Epic adapter, then GetRepository(EntityTypeEpic) | Returns the registered adapter | N/A |
| TC-AC7-02: Unregistered type lookup | GetRepository for unregistered type | Returns clear error (not panic) | All types unregistered (empty registry) |
| TC-AC7-03: MustGetRepository panics on missing | MustGetRepository for unregistered type | `recover()` catches panic with descriptive message | N/A |
| TC-AC7-04: Duplicate registration panics | Register same type twice | `recover()` catches panic | N/A |
| TC-AC7-05: RegisteredTypes returns all | Register all 5 types | RegisteredTypes returns slice of length 5 containing all types | Empty registry returns empty slice |
| TC-AC7-06: Thread safety | Concurrent Register + GetRepository in goroutines | No data race (run with `-race` flag) | N/A |

### AC8: ChangeCard Type Normalization

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC8-01: Model fields are *string | Inspect `ChangeCard.Slug` and `ChangeCard.FilePath` | Both are `*string` type | N/A |
| TC-AC8-02: Repo scan handles NULL | DB row has NULL slug/filepath | `card.Slug == nil`, `card.FilePath == nil` | N/A -- `database/sql` handles automatically |
| TC-AC8-03: Repo scan handles non-NULL | DB row has "my-slug" for slug | `*card.Slug == "my-slug"` | N/A |
| TC-AC8-04: Service sets Slug correctly | Create ChangeCard with title "My Change" | `*card.Slug == "my-change"` (via `utils.GenerateSlug`) | Empty title |
| TC-AC8-05: Service FilePath nil check | `card.FilePath == nil` with non-empty projectRoot | No panic; file write skipped | `card.FilePath` is `&""` (empty but non-nil) |
| TC-AC8-06: CLI display nil-safe | `card.FilePath == nil` | No "File" row displayed | Non-nil empty string still skips display |

### AC9: Quality Gate Passes

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC9-01: Full quality gate | All F01 code complete | `make fmt && make lint && make test` -- zero issues | N/A -- mandatory gate |

### AC10: Test Coverage

| Test Case | Input/Precondition | Expected Outcome | Edge Cases |
|---|---|---|---|
| TC-AC10-01: Interface method coverage | All 5 entity types tested | Every one of the 14 methods called and asserted for each type (70 assertions total) | N/A |
| TC-AC10-02: Registry operation coverage | All registry methods tested | Register, GetRepository, MustGetRepository, RegisteredTypes -- all exercised | N/A |
| TC-AC10-03: Adapter delegation coverage | All 5 adapters tested | Each adapter's 6 methods tested (30 test points) | N/A |

---

## 2. Component Test Strategy

### 2.1 `internal/models/entity_test.go`

**Purpose**: Compile-time satisfaction + accessor correctness for all 5 models.

**Key tests**:

```
TestEntity_CompileTimeSatisfaction
  - Verified by var _ checks in entity.go (build failure = test failure)

TestEntity_Accessors (table-driven over 5 types)
  - For each model:
    - Populate all fields with known values
    - Assert each of 14 accessor methods returns expected value
    - Assert nil *string fields return ""

TestEntity_SetStatus (table-driven over 5 types)
  - Set status string, verify GetStatus() returns it

TestEntity_SetContextData
  - nil -> non-nil -> nil round-trip for each model

TestEntity_GetEntityType (table-driven)
  - Each model returns its correct EntityType constant
```

**Edge cases to cover**:
- Zero-value struct (all fields default) -- accessors must not panic
- Nil pointer fields (Slug, FilePath, Description, ContextData) return `""`

### 2.2 `internal/services/entity_registry_test.go`

**Purpose**: Registration, lookup, thread safety.

**Key tests**:

```
TestEntityRegistry_RegisterAndGet
  - Register mock adapter, retrieve it

TestEntityRegistry_UnregisteredType
  - GetRepository returns error with type name in message

TestEntityRegistry_MustGetRepository_Panics
  - Unregistered type panics (assert via recover)

TestEntityRegistry_DuplicateRegistration_Panics
  - Same type registered twice panics (assert via recover)

TestEntityRegistry_RegisteredTypes
  - Register 5 types, verify all returned
  - Empty registry returns empty slice

TestEntityRegistry_ConcurrentAccess
  - Goroutines doing Register + GetRepository simultaneously
  - Run with -race flag; no data race
```

### 2.3 `internal/services/entity_adapter_test.go`

**Purpose**: Adapter delegation to typed repos with mock repositories.

**Key tests**:

```
TestEpicRepositoryAdapter_GetByKey
  - Mock EpicRepository.GetByKey returns *Epic
  - Adapter returns it as models.Entity
  - Verify Key via GetKey()

TestEpicRepositoryAdapter_Update_WrongType
  - Pass *models.Task to Epic adapter's Update
  - Assert error contains "expected *models.Epic"

TestEpicRepositoryAdapter_UpdateStatus
  - Verify mock receives EpicStatus("active")

TestFeatureRepositoryAdapter_UpdateStatus_GetSetUpdate
  - Feature adapter calls GetByID, sets status, calls Update
  - Verify all 3 mock calls made in order

TestBugRepositoryAdapter_ContextData_GetSetUpdate
  - Bug adapter: GetContextData calls GetByID, returns ContextData field
  - UpdateContextData calls GetByID, sets field, calls Update

Test<Entity>RepositoryAdapter (parameterized for all 5)
  - Table-driven: each adapter instantiated with mock repo
  - GetByKey, GetByID, Update (correct type), UpdateStatus tested
```

**Mock pattern**: Use function-field mocks per project convention.

### 2.4 ChangeCard normalization tests (updates to existing files)

**Files to update**:
- `internal/services/change_card_service_test.go` -- update string literals to `strPtr("value")`
- `internal/models/change_card_test.go` -- update field assignments

**Add helper** (in test file or shared test utils):
```go
func strPtr(s string) *string { return &s }
```

**Key assertions**:
- ChangeCard creation sets `Slug` as `*string` (non-nil, correct value)
- ChangeCard with nil Slug/FilePath passes Validate()
- Repository round-trip preserves nil vs non-nil for both fields

---

## 3. Integration Scenarios (Downstream Validation)

F01 is purely additive. Its outputs are consumed by F02-F05:

| Downstream Feature | What It Exercises from F01 | Validation Point |
|---|---|---|
| F02: Cross-Cutting Service Unification | EntityRegistry.GetRepository() to replace switch statements in NoteService, ContextService, ResumeService | Registry returns correct adapter for each entity type; adapters delegate correctly |
| F03: Status Transition Unification | EntityRepository.UpdateStatus() and Entity.SetStatus() for polymorphic status transitions | Adapters correctly convert string status to typed status |
| F04: Document Operations Unification | EntityRepository.GetByKey() to resolve entity for document linking | Adapter returns models.Entity; GetKey(), GetEntityType() work |
| F05: CLI Accessor Consolidation | EntityRegistry.RegisteredTypes() for dynamic command generation | All 5 types registered and discoverable |

**F01 completion gate**: F02 can begin IFF all tests in sections 2.1-2.4 pass AND `make fmt && make lint && make test` passes.

---

## 4. Regression Checklist

These existing tests MUST continue passing after each F01 task. Run `make test` after every task.

### Task 1 (ChangeCard Normalization) -- Highest regression risk

- [ ] `internal/services/change_card_service_test.go` -- all tests pass with `*string` updates
- [ ] `internal/models/change_card_test.go` -- Validate() tests pass
- [ ] `internal/repository/change_card_repository_test.go` -- CRUD tests pass (if any exist)
- [ ] `internal/cli/commands/change_card_commands_test.go` -- CLI display tests pass (if any exist)
- [ ] Full suite: `make test` -- zero failures

### Task 2 (Entity Interface + Implementations) -- Zero regression risk

- [ ] All existing model tests pass (no model fields changed, only methods added)
- [ ] Full suite: `make test` -- zero failures
- [ ] New: `internal/models/entity_test.go` -- all accessor tests pass

### Task 3 (EntityRepository + Adapters) -- Zero regression risk

- [ ] All existing service tests pass (no existing service code modified)
- [ ] Full suite: `make test` -- zero failures
- [ ] New: `internal/services/entity_adapter_test.go` -- all adapter tests pass

### Task 4 (EntityRegistry) -- Zero regression risk

- [ ] All existing tests pass (no existing code modified)
- [ ] Full suite: `make test` -- zero failures
- [ ] New: `internal/services/entity_registry_test.go` -- all registry tests pass

### Final Gate

```bash
make fmt && make lint && make test
```

Zero formatting changes. Zero lint warnings. Zero test failures. This is the F01 exit criteria and the F02 entry criteria.

---

## 5. Performance Tests (Informational)

Per REQ-NF-003 and REQ-NF-004, add benchmarks in `internal/models/entity_test.go`:

```
BenchmarkEntityInterface_Dispatch  -- target: <5ns per accessor call
BenchmarkEntityRegistry_Lookup     -- target: <50ns per GetRepository call
```

These are informational and do not block F01 completion, but should be included for baseline measurement.

---

*Traces to: PRD AC1-AC10, Architecture doc Section 8 (Test Strategy), UAT Plan P0/P1*
