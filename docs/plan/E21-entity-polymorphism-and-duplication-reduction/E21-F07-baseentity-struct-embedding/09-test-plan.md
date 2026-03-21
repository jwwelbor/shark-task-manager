# F07: BaseEntity Struct Embedding -- Test Plan

**Feature**: E21-F07
**Tier**: COMPLEX
**Author**: QA Agent
**Date**: 2026-03-20

This is the comprehensive test plan for the BaseEntity struct embedding refactoring. Because this feature is a pure internal refactoring with zero behavioral change, the testing strategy is dominated by **regression validation** (existing tests must pass without modification) and **structural verification** (embedding mechanics work correctly). New test code is limited to validating BaseEntity itself.

---

## 1. Epic UAT Scenario Decomposition

### Mapping F07 to Epic UAT Scenarios

F07 contributes to epic-level acceptance in specific, measurable ways. The table below maps each epic UAT scenario to F07's scope.

| Epic UAT Scenario | F07 Contribution | F07 Validation Method |
|---|---|---|
| **Scenario 1: Add New Entity Type** (Journey 1) | BaseEntity embed reduces new entity boilerplate from ~25 lines (10 fields + 14 methods) to ~1 line (embed) + 4 methods | After F07 completion, verify that a hypothetical entity struct with BaseEntity embed compiles and satisfies Entity interface with only GetEntityType, GetStatus, SetStatus, Validate |
| **Scenario 4: Onboarding and Code Navigability** (Journey 4) | BaseEntity centralizes shared field documentation. Reading `entity.go` shows all shared fields + methods in one place instead of 5 | After F07, verify entity.go contains both the Entity interface (14 methods) and BaseEntity struct (9 fields + 10 methods) as a self-documenting single-file reference |
| **UAT Metric 4: Test Pass Rate** | F07 must achieve 100% existing test pass rate at every per-entity migration boundary | `make fmt && make lint && make test` after each of 7 tasks |
| **UAT Metric 6: Interface Satisfaction** | F07 preserves all 5 compile-time checks: `var _ Entity = (*Epic)(nil)` etc. | Verify 5 checks present in entity.go after all migrations |
| **P0: Zero Behavioral Regression** | Core F07 requirement. All existing tests pass without modification. No JSON output changes. No repository behavior changes. | Full quality gate after every migration task |

### F07-Specific UAT Verification Scenarios

**UAT-F07-01: New Shared Field Addition (Validates Scenario 1 benefit)**

- **Given**: All 5 entity types embed BaseEntity
- **When**: A developer adds `DeletedAt *time.Time` to BaseEntity
- **Then**: All 5 entity types automatically have the field, `epic.DeletedAt` compiles, and no other files require modification
- **Validation**: This is a manual spot-check after F07 completion. Not automated.

**UAT-F07-02: Cost of New Entity Type (Validates Scenario 1)**

- **Given**: BaseEntity exists with 9 fields + 10 methods
- **When**: Creating a new entity type struct
- **Then**: The struct needs only: `BaseEntity` embed line + typed Status field + GetEntityType + GetStatus + SetStatus + Validate = 4 methods, not 14
- **Validation**: Count methods required after F07 vs. before. Must be 4 per-entity, not 14.

---

## 2. Component Test Strategy

### 2.1 BaseEntity Struct and Methods (`internal/models/entity.go`)

**Purpose**: Validate the new BaseEntity struct in isolation before it is embedded in any entity type.

**New tests to write** (append to `internal/models/entity_test.go`):

| Test Name | Purpose | Assertions |
|---|---|---|
| `TestBaseEntity_FieldAccess` | All 10 getter methods return correct values when fields are populated | GetID returns int64, GetKey returns string, GetTitle returns string, GetSlug returns dereferenced `*string`, GetDescription returns dereferenced `*string`, GetFilePath returns dereferenced `*string`, GetContextData returns `*string` pointer, GetCreatedAt returns non-zero time, GetUpdatedAt returns non-zero time |
| `TestBaseEntity_NilFields` | Nil-safe behavior for optional pointer fields | GetSlug returns `""` when Slug is nil, GetDescription returns `""` when Description is nil, GetFilePath returns `""` when FilePath is nil, GetContextData returns `nil` when ContextData is nil |
| `TestBaseEntity_SetContextData` | Mutator method correctness including nil transitions | nil -> non-nil: GetContextData returns correct pointer; non-nil -> nil: GetContextData returns nil |
| `TestBaseEntity_ZeroValue` | Zero-value struct does not panic on any accessor call | `BaseEntity{}` -- all getters return zero values without panic |

**Estimated new test lines**: ~50

### 2.2 Embedding Promotion Verification (`internal/models/entity_test.go`)

**Purpose**: Verify that Go struct embedding correctly promotes fields and methods from BaseEntity to each entity type.

| Test Name | Purpose | Assertions |
|---|---|---|
| `TestBaseEntity_EmbeddingPromotion` | Field promotion works for Epic (simplest entity) | `epic.ID`, `epic.Key`, `epic.Title` resolve correctly; `Entity` interface methods work via promotion; entity-specific `GetEntityType()` and `GetStatus()` take precedence over promoted methods |
| `TestBaseEntity_JSONSerialization` | JSON output is flat (no nested `base_entity` or `BaseEntity` wrapper) | Marshal an Epic with BaseEntity; unmarshal to `map[string]interface{}`; verify `id`, `key`, `title`, `status`, `priority` are top-level keys; verify `BaseEntity` and `base_entity` keys do NOT exist |

**Estimated new test lines**: ~40

### 2.3 Existing Test Suites (No Modification Required)

These test suites validate backward compatibility. They must all pass without ANY modification after each migration task.

| Test Suite | File(s) | # Tests (est.) | What It Validates for F07 |
|---|---|---|---|
| Entity interface accessors | `internal/models/entity_test.go` (431 lines) | ~50 | All 14 Entity interface methods work correctly for all 5 types. Field assignment via `epic.Key = "E01"` compiles and works. |
| Repository CRUD | `internal/repository/*_test.go` | ~30 | Database `Scan()` calls work with promoted fields (`&epic.Key` resolves to `&epic.BaseEntity.Key`). Insert/update round-trips preserve all field values. |
| Service layer | `internal/services/*_test.go` | ~40 | Entity interface method calls in polymorphic contexts. Status transition validation. Note, context, resume operations. |
| CLI commands | `internal/cli/commands/*_test.go` | ~20 | Direct field access and JSON serialization in commands. |
| **Total existing** | | **~140** | |

**Why no modification is required**: Go struct embedding with anonymous fields promotes both fields and methods transparently. `epic.Key` after embedding resolves to `epic.BaseEntity.Key` at compile time. All test assertions that use field access, interface method calls, or JSON serialization continue to work identically.

---

## 3. API Contract Test Cases

F07 has **no external API changes** (no HTTP API changes, no CLI output format changes, no database schema changes). The "API contract" for F07 is the **internal Go type system contract**: struct field access, interface satisfaction, and JSON serialization format.

### 3.1 Go Type System Contract

| Contract | Test Method | Pass Criteria |
|---|---|---|
| **Entity interface satisfaction** | Compile-time checks: `var _ Entity = (*Epic)(nil)` etc. (5 checks) | `go build ./...` succeeds with zero errors |
| **Field promotion** | `epic.Key`, `task.Title`, `feature.Description` still compile | `go build ./...` succeeds; existing tests that assign/read promoted fields pass |
| **Method resolution order** | When outer type defines `GetStatus()`, it takes precedence over BaseEntity (which has no GetStatus) | `entity.GetStatus()` calls the per-entity method, not a BaseEntity method (BaseEntity has no GetStatus) |
| **No ambiguous field reference** | Task.ContextData removed (promoted from BaseEntity instead) | `task.ContextData` resolves unambiguously to `task.BaseEntity.ContextData`; `go build ./...` succeeds |

### 3.2 JSON Serialization Contract

| Entity | Contract | Test Method |
|---|---|---|
| Epic | JSON output identical before and after embedding. No `base_entity` wrapper. | `TestBaseEntity_JSONSerialization` (new test) + existing CLI tests that validate `shark epic get --json` output |
| Feature | Same | Existing repository and CLI tests |
| Task | Same | Existing repository and CLI tests |
| Bug | Same | Existing repository and CLI tests |
| ChangeCard | Same | Existing repository and CLI tests |

**Specific JSON assertions for new test**:
- `json.Marshal(epic)` produces a map with top-level keys: `id`, `key`, `title`, `slug`, `status`, `priority`, `created_at`, `updated_at`
- No key named `BaseEntity` or `base_entity` exists in the output
- The `status` field value comes from `Epic.Status` (typed `EpicStatus`), not from any BaseEntity field

### 3.3 Database Scan Contract

| Repository | Scan Pattern | Why Safe |
|---|---|---|
| EpicRepository | `&epic.ID, &epic.Key, &epic.Title, ...` | Field promotion: `&epic.Key` resolves to `&epic.BaseEntity.Key`. Name-based, not order-based. |
| FeatureRepository | Same pattern | Same reason |
| TaskRepository (~14 scan sites) | Same pattern | Same reason. Special attention: `&task.ContextData` must resolve unambiguously after removing standalone ContextData field from Task. |
| BugRepository (centralized `scanBug()` helper) | Same pattern via helper | Same reason |
| ChangeCardRepository | Same pattern | Same reason |

**No repository code changes are required.** This is the most important claim of the architecture doc, and existing repository tests validate it automatically.

---

## 4. Integration Scenarios

### 4.1 Per-Entity Migration Integration (7-task sequential verification)

Each task must pass the full quality gate before the next begins. This is the primary integration test strategy.

| Task | Change | Integration Verification | Risk Level |
|---|---|---|---|
| Task 1: Define BaseEntity | Add struct + 10 methods to entity.go | `make fmt && make lint && make test` -- all existing tests pass, no code uses BaseEntity yet | Very Low (additive only) |
| Task 2: Migrate Epic | Embed BaseEntity, remove 9 fields + 10 methods from epic.go | Full quality gate. Specifically watch: epic_repository_test.go, entity_test.go Epic tests, epic_service_test.go | Medium |
| Task 3: Migrate Feature | Same pattern on feature.go | Full quality gate. Watch: feature_repository_test.go, feature_service_test.go | Medium |
| Task 4: Migrate Bug | Same pattern on bug.go | Full quality gate. Watch: bug_repository_test.go, bug service tests | Medium |
| Task 5: Migrate ChangeCard | Same pattern on change_card.go | Full quality gate. Watch: change_card_repository_test.go, change_card_service_test.go | Medium |
| Task 6: Migrate Task | Embed BaseEntity, remove 9 fields + 10 methods. **Critical**: Remove standalone ContextData field to avoid ambiguity. | Full quality gate. This is the highest-risk task: Task has 19 specific fields, ~14 scan sites, and the ContextData field removal. Watch ALL task tests carefully. | High |
| Task 7: Cleanup + verification | Remove dead code, spot-check JSON output | Full quality gate + manual JSON spot-checks via CLI | Low |

### 4.2 Cross-Feature Integration with E21

| Sibling Feature | Integration Point | Verification |
|---|---|---|
| **E21-F01** (completed) | F07 changes the _implementation_ of the Entity interface without changing the interface itself. F01's compile-time checks remain in entity.go. | All 5 `var _ Entity = (*)` checks still present and compile. |
| **E21-F02** (completed) | Cross-cutting services (NoteService, ContextService, ResumeService) use Entity interface, not struct fields directly. Unaffected by BaseEntity embedding. | F02 integration tests pass without modification after F07. |
| **E21-F03** (completed) | EntityService.TransitionStatus calls Entity.GetStatus()/SetStatus(). These remain per-entity methods. Unaffected by BaseEntity. | F03 status transition tests pass without modification. |
| **E21-F08** (draft) | Polymorphic Data Model Unification may build on BaseEntity for shared table schema patterns. F07 provides the foundation. | No validation needed now; F08 consumes F07's output. |

### 4.3 Cross-Epic Integration

| Epic | Overlap | Verification |
|---|---|---|
| **E15** (Service Layer Architecture) | E15 refactors CLI commands to use services. F07 changes model layer only. No overlap with service layer method signatures. | `make test` covers all E15-migrated commands. |
| **E19** (Sprint Management) | E19 would add a new entity type. F07 reduces the cost of that addition. | Validated by UAT-F07-02 scenario. |

---

## 5. Acceptance Criteria Test Matrix

### Story 1: Shared fields defined in one place

| AC | Test Case ID | Test Method | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|---|
| BaseEntity contains 9 fields (ID, Key, Title, Slug, Description, FilePath, ContextData, CreatedAt, UpdatedAt) | TC-S1-01 | `TestBaseEntity_FieldAccess` | Populate all 9 fields | All getters return correct values | N/A |
| Status excluded from BaseEntity (Option B) | TC-S1-02 | Code inspection + compile check | BaseEntity struct definition | No `Status` field in BaseEntity; each entity has typed Status on outer struct | N/A |
| All 5 entity types embed BaseEntity | TC-S1-03 | `go build ./...` | All 5 model files updated | Compilation succeeds; entity_test.go passes | N/A |
| Entity-specific typed status still works | TC-S1-04 | Existing `TestEntity_SetStatus` (entity_test.go) | Set status on each entity type | GetStatus returns correct typed status as string | Empty string status |
| All existing tests pass without modification | TC-S1-05 | `make test` | Full test suite | 100% pass rate, zero test file modifications | N/A |

### Story 2: Entity interface methods on BaseEntity

| AC | Test Case ID | Test Method | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|---|
| BaseEntity implements 10 methods | TC-S2-01 | `TestBaseEntity_FieldAccess` + `TestBaseEntity_NilFields` | Populated and zero-value BaseEntity | All 10 methods callable and return correct values | Nil pointer fields return `""` |
| Each entity overrides only GetEntityType and Validate | TC-S2-02 | Code inspection + entity_test.go | Each entity type | Only 4 methods defined per-entity: GetEntityType, GetStatus, SetStatus, Validate | N/A |
| Compile-time interface satisfaction | TC-S2-03 | `go build ./...` | 5 compile-time checks in entity.go | Build succeeds | N/A |

### Story 3: Direct field access still works

| AC | Test Case ID | Test Method | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|---|
| `epic.Key`, `task.Title`, `feature.Status` compile | TC-S3-01 | `go build ./...` + existing tests | All codebase files using direct field access | Compilation succeeds | N/A |
| JSON serialization identical | TC-S3-02 | `TestBaseEntity_JSONSerialization` (new) + existing tests | Marshal each entity type | Flat JSON, no `base_entity` wrapper, all fields present | Nil optional fields omitted with `omitempty` |
| Database scan operations work | TC-S3-03 | Existing repository tests | CRUD operations for all 5 entity types | All repository tests pass | N/A |

### Requirement-Level Matrix

| Requirement | ACs Covered | Test Cases | Priority |
|---|---|---|---|
| REQ-F-001: BaseEntity Definition | TC-S1-01, TC-S1-02, TC-S2-01 | TestBaseEntity_FieldAccess, TestBaseEntity_NilFields, TestBaseEntity_ZeroValue | Must-Have |
| REQ-F-002: Entity Type Refactoring | TC-S1-03, TC-S2-02, TC-S2-03, TC-S3-01 | go build, entity_test.go, code inspection | Must-Have |
| REQ-F-003: Status Type Compatibility | TC-S1-02, TC-S1-04 | TestEntity_SetStatus (existing), code inspection | Must-Have |
| REQ-F-004: Repository Scan Compatibility | TC-S3-03 | All existing repository tests | Must-Have |
| REQ-NF-001: Zero API Surface Change | TC-S1-05, TC-S3-01, TC-S3-02, TC-S3-03 | make test, go build, JSON test | Must-Have |

---

## 6. Performance, Security, and Non-Functional Approach

### 6.1 Performance

**Assertion**: Go struct embedding is a compile-time mechanism with zero runtime cost. Method dispatch for promoted methods is identical to direct method calls (no virtual dispatch overhead). Memory layout may differ by a few bytes of padding but is negligible.

**Verification approach**: Informational benchmarks, not blocking.

| Benchmark | Location | Target | Notes |
|---|---|---|---|
| `BenchmarkBaseEntity_GetKey` | `internal/models/entity_test.go` | < 5ns per call | Verify method promotion has no dispatch overhead compared to direct method calls |
| `BenchmarkEntity_InterfaceDispatch` | `internal/models/entity_test.go` | < 5ns per call | Compare interface dispatch before and after embedding (should be identical) |

These benchmarks are informational. They do not block F07 completion.

### 6.2 Security

F07 has **no security surface**. It is a pure internal code refactoring that:
- Does not change any external API
- Does not change any database schema
- Does not change any input validation logic
- Does not change any authentication or authorization behavior
- Does not introduce any new dependencies

No security testing is required.

### 6.3 Backward Compatibility (REQ-NF-001)

This is the most critical non-functional requirement. Verification is embedded throughout the test plan:

| Check | When | How | Failure Action |
|---|---|---|---|
| All existing tests pass | After each of 7 tasks | `make fmt && make lint && make test` | STOP. Fix before proceeding. |
| JSON output unchanged | Task 7 (cleanup) | Manual spot-check: `shark epic get E21 --json`, `shark feature get E21-F07 --json`, `shark task list E21 F01 --json` | STOP. Investigate JSON tag conflicts. |
| No compile errors | After each entity migration | `go build ./...` | STOP. Field promotion not working as expected. |
| Compile-time interface checks present | Task 7 | `grep 'var _ Entity' internal/models/entity.go` -- must return 5 lines | Re-add missing checks. |

### 6.4 Quantitative Verification

| Metric | Before | After | How to Measure |
|---|---|---|---|
| Shared field declarations | 50 (10 x 5) | 9 (in BaseEntity) | Count shared field declarations across 5 model files. After: only in entity.go. |
| Shared method implementations | ~70 (14 x 5) | 10 (in BaseEntity) + 20 (4 x 5 per-entity) | Count Entity interface method implementations. |
| Total lines in model files | ~485 | ~355 (est. -130) | `wc -l internal/models/{epic,feature,task,bug,change_card,entity}.go` |
| Cost to add new shared field | 5 files | 1 file | Manual assessment after completion. |

---

## Regression Checklist (Per-Task)

Use this checklist after each migration task:

```
[ ] make fmt -- zero formatting changes
[ ] make lint -- zero lint warnings
[ ] make test -- 100% pass rate
[ ] go build ./... -- zero compile errors
[ ] Compile-time interface checks: grep 'var _ Entity' internal/models/entity.go returns 5 lines
```

### High-Risk Task: Task 6 (Migrate Task)

Task migration requires extra attention:
- [ ] `task.ContextData` standalone field removed (promoted from BaseEntity instead)
- [ ] No ambiguous field reference compile error
- [ ] All ~14 task repository scan sites work (validated by task repository tests)
- [ ] Task.Validate() unchanged (most complex validation method)
- [ ] All task-specific fields (19) correctly declared on outer struct
- [ ] `make test` -- all task tests pass

---

## Exit Gate

- [x] Full 6-section plan written (UAT decomposition, component strategy, API contracts, integration scenarios, AC matrix, performance/security)
- [x] Epic UAT traceability documented (Section 1 maps F07 to UAT Scenarios 1, 4, Metrics 4, 6, P0)
- [x] All acceptance criteria have test cases with IDs
- [x] Per-task regression checklist defined
- [x] Cross-feature and cross-epic integration verified
- [x] Performance approach documented (informational benchmarks)
- [x] Security approach documented (no surface)

---

*Traces to: Feature PRD (feature.md) REQ-F-001 through REQ-F-004, REQ-NF-001; Architecture (02-architecture.md) Sections 3-7; Testing Strategy (05-testing-strategy.md); Epic UAT Acceptance Plan (uat-acceptance-plan.md) Scenarios 1, 4, Metrics 4, 6, P0*
