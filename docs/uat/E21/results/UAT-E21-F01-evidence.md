# UAT Evidence - E21-F01 Entity Interface Foundation

## Scenario 1: Entity Interface Completeness

### Spec Quotes

From 02-architecture.md Section 2:
> "Entity is the polymorphic interface implemented by all domain entity types. It provides accessor methods for the shared fields common to all entities."

From 09-test-plan.md AC1:
> "Interface has 14 methods: GetID, GetKey, GetTitle, GetSlug, GetEntityType, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt, Validate"

From 09-test-plan.md AC3:
> "Compile-time checks: `var _ Entity = (*Epic)(nil)` etc. in entity.go — go build succeeds with zero errors"

From 09-test-plan.md AC8:
> "ChangeCard Slug and FilePath are *string type"

### Implementation Code

**Entity interface** (`internal/models/entity.go:11-26`):
- 14 methods defined: GetID, GetKey, GetTitle, GetSlug, GetEntityType, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt, Validate

**Compile-time checks** (`internal/models/entity.go:29-35`):
```go
var (
    _ Entity = (*Epic)(nil)
    _ Entity = (*Feature)(nil)
    _ Entity = (*Task)(nil)
    _ Entity = (*Bug)(nil)
    _ Entity = (*ChangeCard)(nil)
)
```

**ChangeCard normalization** (`internal/models/change_card.go:47-48`):
```go
Slug     *string `json:"slug,omitempty" db:"slug"`
FilePath *string `json:"file_path,omitempty" db:"file_path"`
```

### Test Code

**File:** `internal/models/entity_test.go` (356 lines)
- `TestEntity_Accessors` — table-driven over all 5 types, validates all 12 getters
- `TestEntity_GetEntityType` — verifies correct EntityType constant for each model
- `TestEntity_SetStatus` — verifies mutation for all 5 types + empty string
- `TestEntity_SetContextData` — verifies nil→non-nil→nil round-trip for all 5 types
- `TestEntity_NilPointerFields` — verifies zero-value structs return "" for *string fields
- `TestEntity_ZeroValueAccessors` — verifies no panics on zero-value structs

### Test Output

All entity tests PASS (7 test functions, 30+ sub-tests):
```
--- PASS: TestEntity_Accessors (5 subtests)
--- PASS: TestEntity_GetEntityType (5 subtests)
--- PASS: TestEntity_SetStatus (5 subtests)
--- PASS: TestEntity_SetContextData (5 subtests)
--- PASS: TestEntity_NilPointerFields (5 subtests)
--- PASS: TestEntity_ZeroValueAccessors (5 subtests)
```

---

## Scenario 2: EntityRepository Interface and Adapters

### Spec Quotes

From 02-architecture.md Section 5:
> "EntityRepository provides polymorphic data access for any entity type. It wraps typed repositories to support cross-cutting operations."
> "Methods: GetByKey, GetByID, UpdateStatus, Update, GetContextData, UpdateContextData"

From 02-architecture.md Section 6.2:
> "Feature adapter: calls GetByID, sets status, calls Update (get-set-update pattern)"
> "Task adapter: Task repository's UpdateStatus has a different signature. The adapter normalizes this by doing get-set-update like Feature."

From 09-test-plan.md AC6:
> "Update rejects wrong type: Pass *models.Task to EpicRepositoryAdapter.Update → Error message contains 'expected *models.Epic, got *models.Task'"

### Implementation Code

**EntityRepository interface** (`internal/services/entity_repository.go:14-33`):
- 6 methods: GetByKey, GetByID, UpdateStatus, Update, GetContextData, UpdateContextData

**Adapter files** (5 files in `internal/services/`):
- `epic_repo_adapter.go` — direct delegation (repo has all methods)
- `feature_repo_adapter.go` — UpdateStatus uses get-set-update
- `task_repo_adapter.go` — UpdateStatus/GetContextData/UpdateContextData use get-set-update
- `bug_repo_adapter.go` — GetContextData/UpdateContextData use get-set-update
- `change_repo_adapter.go` — direct delegation for most, GetContextData uses get-set-update

### Test Code

**File:** `internal/services/entity_adapter_test.go` (871 lines)
- 5 mock typed repositories defined (mockEpicAdapterRepo, mockFeatureAdapterRepo, etc.)
- 38 test functions covering all 5 adapters
- Tests cover: GetByKey, GetByID, UpdateStatus, Update (correct type), Update (wrong type), GetContextData, GetContextData (nil), UpdateContextData, error propagation
- `TestAllAdapters_SatisfyEntityRepository` — compile-time check for all 5

### Test Output

All 38 adapter tests PASS:
```
--- PASS: TestEpicAdapter_GetByKey
--- PASS: TestEpicAdapter_Update_WrongType
--- PASS: TestFeatureAdapter_UpdateStatus_GetSetUpdate
--- PASS: TestTaskAdapter_UpdateStatus_GetSetUpdate
--- PASS: TestBugAdapter_GetContextData
--- PASS: TestChangeCardAdapter_UpdateStatus
--- PASS: TestAllAdapters_SatisfyEntityRepository
[... all 38 tests pass]
```

---

## Scenario 3: EntityRegistry Operations

### Spec Quotes

From 02-architecture.md Section 7:
> "The registry is thread-safe with sync.RWMutex and provides Register, GetRepository, MustGetRepository, and RegisteredTypes."
> "Registration is expected at startup; lookups happen at request time."

From 09-test-plan.md AC7:
> "Duplicate Register panics"
> "Thread safety: concurrent Register + GetRepository in goroutines — no data race (run with -race flag)"

### Implementation Code

**EntityRegistry** (`internal/services/entity_registry.go:16-79`):
- `sync.RWMutex` for thread safety
- `Register` — panics on duplicate
- `GetRepository` — returns error on missing
- `MustGetRepository` — panics on missing
- `RegisteredTypes` — returns sorted slice

### Test Code

**File:** `internal/services/entity_registry_test.go` (172 lines)
- `TestEntityRegistry_RegisterAndGet` — register 2 types, verify correct lookup
- `TestEntityRegistry_UnregisteredType` — returns error with type name
- `TestEntityRegistry_MustGetRepository_Panics` — assert.Panics
- `TestEntityRegistry_MustGetRepository_Success` — happy path
- `TestEntityRegistry_DuplicateRegistration_Panics` — assert.Panics
- `TestEntityRegistry_RegisteredTypes` — empty + all 5 types + sorted
- `TestEntityRegistry_ConcurrentAccess` — 50 goroutines with -race flag

### Test Output

All 7 registry tests PASS (including race-safe test):
```
--- PASS: TestEntityRegistry_RegisterAndGet
--- PASS: TestEntityRegistry_UnregisteredType
--- PASS: TestEntityRegistry_MustGetRepository_Panics
--- PASS: TestEntityRegistry_MustGetRepository_Success
--- PASS: TestEntityRegistry_DuplicateRegistration_Panics
--- PASS: TestEntityRegistry_RegisteredTypes (2 subtests)
--- PASS: TestEntityRegistry_ConcurrentAccess

Race detector: ok (no races detected)
```

---

## Scenario 4: Backward Compatibility

### Quality Gate Output

```
make fmt: zero formatting changes
make lint: 0 issues
make test: all 30 packages pass, zero failures
```

Package list (all ok):
- internal/cli, internal/cli/commands, internal/cli/scope
- internal/config, internal/db, internal/dependency
- internal/discovery, internal/fileops, internal/filepath
- internal/formatters, internal/init, internal/keygen
- internal/keys, internal/models, internal/parser
- internal/pathresolver, internal/patterns, internal/reporting
- internal/repository, internal/services, internal/slug
- internal/status, internal/taskcreation, internal/taskfile
- internal/template, internal/templates, internal/utils
- internal/validation, internal/view, internal/workflow

### Build Verification

`go build ./...` succeeds — compile-time interface satisfaction checks pass for all 5 entity types.
