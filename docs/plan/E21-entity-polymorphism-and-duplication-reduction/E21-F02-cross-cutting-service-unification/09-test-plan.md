# E21-F02: Cross-Cutting Service Unification -- Test Plan

**Feature**: E21-F02
**Author**: QA Agent
**Date**: 2026-03-19
**Status**: Draft
**Tier**: STANDARD

---

## 1. Acceptance Criteria Test Matrix

Each acceptance criterion from the PRD is mapped to concrete, actionable test cases. Tests are organized by the user story they validate and the persona they serve.

### Story 1: NoteService Registry-Based Dispatch

**Persona**: Codebase Maintainer -- wants bug fixes in one place, not five.

| AC ID | Acceptance Criterion | Test Case | Type | TDD Priority |
|-------|---------------------|-----------|------|--------------|
| S1-AC1 | `resolveEntityID` uses registry, no switch | TC-01: Parameterized test calling `resolveEntityID` for all 5 entity types via mock registry | Unit | P0 |
| S1-AC1 | No switch on entity type in `resolveEntityID` | TC-02: Static analysis -- `grep -c "case models.EntityType" note_service.go` returns 0 | Metric | P0 |
| S1-AC2 | `GetEntityDetails` uses registry, no switch | TC-03: Parameterized test calling `GetEntityDetails` for all 5 entity types via mock registry | Unit | P0 |
| S1-AC3 | All 5 `SetXxxRepo()` removed from NoteService | TC-04: Compile check -- NoteService has no `Set*Repo` methods | Compile | P0 |
| S1-AC4 | Constructor accepts `*EntityRegistry` | TC-05: `NewNoteService(noteRepo, registry)` compiles and returns non-nil | Unit | P0 |
| S1-AC5 | All note operations identical for all 5 types | TC-06: `AddNote`, `ListNotes` produce same structure for epic, feature, task, bug, change | Unit | P1 |

### Story 2: ContextService Registry-Based Dispatch

**Persona**: Codebase Maintainer -- wants context ops to extend automatically.

| AC ID | Acceptance Criterion | Test Case | Type | TDD Priority |
|-------|---------------------|-----------|------|--------------|
| S2-AC1 | `getContextJSON` uses registry, no switch | TC-07: Parameterized test for `GetContext` across all 5 entity types | Unit | P0 |
| S2-AC2 | `setContextJSON` uses registry, no switch | TC-08: Parameterized test for `SetContextField` across all 5 entity types | Unit | P0 |
| S2-AC3 | All `SetXxxRepo()` removed | TC-09: Compile check -- ContextService has no `Set*Repo` methods | Compile | P0 |
| S2-AC4 | Constructor accepts `*EntityRegistry` | TC-10: `NewContextService(registry)` compiles and returns non-nil | Unit | P0 |
| S2-AC5 | All context operations identical | TC-11: `GetContext`, `SetContextField`, `ClearContext` for all 5 entity types | Unit | P1 |

### Story 3: ResumeService Registry-Based Entity Lookup

**Persona**: Codebase Maintainer -- wants resume to extend to new entities.

| AC ID | Acceptance Criterion | Test Case | Type | TDD Priority |
|-------|---------------------|-----------|------|--------------|
| S3-AC1 | Bug/Change lookup uses registry | TC-12: `GetBugResume` resolves bug via `registry.GetRepository(EntityTypeBug)` | Unit | P0 |
| S3-AC2 | `SetBugRepo` and `SetChangeCardRepo` removed | TC-13: Compile check -- ResumeService has no `SetBugRepo`/`SetChangeCardRepo` | Compile | P0 |
| S3-AC3 | Constructor accepts `*EntityRegistry` | TC-14: `NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo, registry)` compiles | Unit | P0 |
| S3-AC4 | Resume operations identical | TC-15: `GetBugResume` and `GetChangeResume` return correct typed context with notes | Unit | P1 |

### Story 4: CLI Wiring Simplification

**Persona**: Service Layer Consumer -- wants simpler init-wire pattern.

| AC ID | Acceptance Criterion | Test Case | Type | TDD Priority |
|-------|---------------------|-----------|------|--------------|
| S4-AC1 | `GetEntityRegistry()` exists with `sync.Once` | TC-16: Verify `GetEntityRegistry()` returns same instance on repeated calls | Integration | P0 |
| S4-AC2 | Registry registers all 5 entity types | TC-17: `GetEntityRegistry().RegisteredTypes()` returns 5 types | Integration | P0 |
| S4-AC3 | `GetNoteService` uses registry | TC-18: Call `GetNoteService()` and verify it returns a functional service | Integration | P1 |
| S4-AC4 | All setter calls removed from `services_global.go` | TC-19: `grep -c "Set.*Repo" services_global.go` returns 0 (excluding SetSessionRepo) | Metric | P0 |
| S4-AC5 | CLI commands produce identical behavior | TC-20: End-to-end smoke tests (see Section 4) | E2E | P1 |

### Error Stories

| AC ID | Acceptance Criterion | Test Case | Type | TDD Priority |
|-------|---------------------|-----------|------|--------------|
| E1-AC1 | Unregistered type returns clear error | TC-21: `registry.GetRepository("sprint")` returns error containing `no repository registered for entity type "sprint"` | Unit | P0 |
| E1-AC2 | Error propagates with context | TC-22: NoteService `AddNote` with unregistered type returns error containing service context | Unit | P0 |
| E1-AC3 | No panic on unregistered type | TC-23: Verify no panic -- error returned cleanly | Unit | P0 |
| E2-AC1 | Nil registry panics at construction | TC-24: `NewNoteService(noteRepo, nil)` panics with message "NoteService: EntityRegistry must not be nil" | Unit | P0 |
| E2-AC2 | All 3 constructors panic on nil | TC-25: Same for `NewContextService(nil)` and `NewResumeService(..., nil)` | Unit | P0 |

---

## 2. Component Test Strategy

### 2.1 NoteService Tests (note_service_test.go)

**What changes**: The existing test file uses per-entity mock repos (`mockNoteEpicRepo`, `mockNoteFeatureRepo`, `mockNoteTaskRepo`). These will be replaced with a single `MockEntityRepository` registered in a mock `EntityRegistry`.

**Test pattern after refactoring**:

```go
func TestNoteService_AddNote_AllEntityTypes(t *testing.T) {
    entityTypes := []struct {
        entityType models.EntityType
        key        string
        entityID   int64
    }{
        {models.EntityTypeEpic, "E01", 1},
        {models.EntityTypeFeature, "E01-F01", 2},
        {models.EntityTypeTask, "E01-F01-001", 3},
        {models.EntityTypeBug, "B001", 4},
        {models.EntityTypeChange, "CC-001", 5},
    }

    for _, et := range entityTypes {
        t.Run(string(et.entityType), func(t *testing.T) {
            mockEntityRepo := &MockEntityRepository{
                GetByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
                    // Return appropriate entity with correct ID
                },
            }
            registry := NewEntityRegistry()
            registry.Register(et.entityType, mockEntityRepo)

            noteRepo := &mockNoteEntityNoteRepo{...}
            svc := NewNoteService(noteRepo, registry)

            note, err := svc.AddNote(ctx, et.entityType, et.key, "comment", "content", "agent")
            assert.NoError(t, err)
            assert.Equal(t, et.entityID, note.EntityID)
        })
    }
}
```

**Existing tests to preserve** (update constructor only):
- `TestNoteService_AddNote_Epic` -- adapt to use registry
- `TestNoteService_AddNote_Feature` -- adapt to use registry
- `TestNoteService_AddNote_Task` -- adapt to use registry
- `TestNoteService_AddNote_InvalidKey` -- adapt to use registry
- `TestNoteService_AddNote_InvalidNoteType` -- no change in logic
- `TestNoteService_ListNotes` -- adapt constructor
- `TestNoteService_ListNotes_WithTypeFilter` -- adapt constructor
- `TestNoteService_SearchNotes` -- adapt constructor
- `TestNoteService_ResolveEntityID_UnsupportedType` -- now tests unregistered entity type via registry

**New tests to add**:
- `TestNoteService_AddNote_AllEntityTypes` -- parameterized across 5 types
- `TestNoteService_GetEntityDetails_AllEntityTypes` -- parameterized across 5 types
- `TestNoteService_NilRegistry_Panics` -- constructor nil check

**Mock cleanup**: Remove `mockNoteEpicRepo`, `mockNoteFeatureRepo`, `mockNoteTaskRepo`. Use shared `MockEntityRepository` from `entity_registry_test.go` (already exists, may need `GetByKeyFunc`/`GetByIDFunc` function fields added).

### 2.2 ContextService Tests (context_service_test.go)

**What changes**: Replace per-entity mock repos (`mockContextEpicRepo`, `mockContextFeatureRepo`, `mockContextTaskRepo`, `mockContextChangeCardRepo`) with mock `EntityRepository` entries. The critical behavior preservation is that adapter implementations handle the per-entity differences (epic uses `GetContextData` repo method; task reads `ContextData` field).

**Existing tests to preserve** (adapt constructor):
- `TestContextService_GetContext_Epic_NoContext`
- `TestContextService_GetContext_Epic_WithContext`
- `TestContextService_GetContext_Task` -- important: verifies task-specific context read path
- `TestContextService_GetContext_InvalidKey`
- `TestContextService_SetContextField_Epic`
- `TestContextService_SetContextField_MergeSemantics`
- `TestContextService_SetContextField_Feature`
- `TestContextService_SetContextField_Task` -- important: verifies task-specific context write path
- `TestContextService_ClearContext_Epic`
- `TestContextService_ClearContext_Task`
- `TestContextService_GetContext_EmptyJSON`
- `TestContextService_UnsupportedEntityType` -- now tests unregistered type
- All ChangeCard context tests -- previously used setter, now use registry

**New tests to add**:
- `TestContextService_GetContext_AllEntityTypes` -- parameterized
- `TestContextService_SetContextField_AllEntityTypes` -- parameterized
- `TestContextService_ClearContext_AllEntityTypes` -- parameterized
- `TestContextService_NilRegistry_Panics`

**Critical behavioral test**: Verify that after refactoring, the ContextService `getContextJSON` for a task entity returns the same data as before. The adapter handles the `task.ContextData` field read, but the test must confirm this path works end-to-end through the registry.

**Mock cleanup**: Remove `mockContextEpicRepo`, `mockContextFeatureRepo`, `mockContextTaskRepo`, `mockContextChangeCardRepo`. The `MockEntityRepository` mock needs `GetContextDataFunc` and `UpdateContextDataFunc` function fields.

### 2.3 ResumeService Tests (resume_service_bug_change_test.go)

**What changes**: Replace `mockResumeBugRepo` and `mockResumeChangeCardRepo` with registry-based mocks. Remove `SetBugRepo`/`SetChangeCardRepo` calls.

**Existing tests to preserve** (adapt constructor and entity lookup):
- `TestResumeService_GetBugResume_ReturnsContext` -- replace `svc.SetBugRepo(mockBugRepo)` with registry registration
- `TestResumeService_GetBugResume_NotFound` -- adapt to registry error path
- `TestResumeService_GetBugResume_NilBugRepo` -- becomes "unregistered bug type" test
- `TestResumeService_GetBugResume_IncludesNotes` -- adapt constructor
- `TestResumeService_GetChangeResume_ReturnsContext` -- replace `svc.SetChangeCardRepo()`
- `TestResumeService_GetChangeResume_NotFound`
- `TestResumeService_GetChangeResume_NilChangeRepo` -- becomes "unregistered change type" test
- `TestResumeService_GetChangeResume_IncludesNotes`

**New tests to add**:
- `TestResumeService_NilRegistry_Panics`
- `TestResumeService_GetBugResume_TypeAssertion` -- verify `entity.(*models.Bug)` succeeds
- `TestResumeService_GetChangeResume_TypeAssertion` -- verify `entity.(*models.ChangeCard)` succeeds

**Mock cleanup**: Remove `mockResumeBugRepo`, `mockResumeChangeCardRepo`.

### 2.4 Shared MockEntityRepository

The existing `mockEntityRepository` in `entity_registry_test.go` is minimal (no-op methods). For F02 service tests, it needs function fields for flexible test behavior. Define a richer mock:

```go
// In mocks_test.go or entity_registry_test.go
type MockEntityRepository struct {
    GetByKeyFunc          func(ctx context.Context, key string) (models.Entity, error)
    GetByIDFunc           func(ctx context.Context, id int64) (models.Entity, error)
    UpdateStatusFunc      func(ctx context.Context, id int64, status string) error
    UpdateFunc            func(ctx context.Context, entity models.Entity) error
    GetContextDataFunc    func(ctx context.Context, id int64) (*string, error)
    UpdateContextDataFunc func(ctx context.Context, id int64, data *string) error
}
```

This mock is shared across NoteService, ContextService, and ResumeService tests.

---

## 3. Non-Functional Verification

### 3.1 Zero Behavioral Regression (REQ-NF02-001)

**Gate**: `make fmt && make lint && make test` must pass after each task.

**Procedure**:
1. Run before starting any F02 work (baseline)
2. Run after each of the 5 implementation tasks
3. Any failure blocks further work

### 3.2 Switch Statement Elimination (UAT Metric 3)

**Procedure**:
```bash
grep -c 'case models.EntityType\|case "epic"\|case "feature"\|case "task"\|case "bug"\|case "change' \
  internal/services/note_service.go \
  internal/services/context_service.go \
  internal/services/resume_service.go
```

**Target**: 0 matches across all 3 files.

### 3.3 Setter Method Elimination (UAT Metric 5)

**Procedure**:
```bash
grep -c 'func (s \*.*Service) Set.*Repo' \
  internal/services/note_service.go \
  internal/services/context_service.go \
  internal/services/resume_service.go
```

**Target**: 0 matches. Exception: `ResumeService.SetSessionRepo` is retained per ADR-F02-004 and is excluded from this metric (it is task-specific, not entity-type-specific).

### 3.4 Net Line Reduction (REQ-NF02-004)

**Procedure**:
```bash
# Before refactoring
wc -l internal/services/note_service.go internal/services/context_service.go \
      internal/services/resume_service.go internal/cli/services_global.go

# After refactoring
wc -l internal/services/note_service.go internal/services/context_service.go \
      internal/services/resume_service.go internal/cli/services_global.go
```

**Target**: Net reduction of at least 150 lines.

### 3.5 Registry Lookup Performance (REQ-NF02-002)

**Procedure**: Benchmark test (optional, low priority).
```go
func BenchmarkEntityRegistry_GetRepository(b *testing.B) {
    reg := NewEntityRegistry()
    reg.Register(models.EntityTypeTask, &mockEntityRepository{})
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        reg.GetRepository(models.EntityTypeTask)
    }
}
```

**Target**: < 50ns per operation (O(1) map lookup).

---

## 4. Integration & End-to-End Scenarios

These scenarios verify that CLI commands produce identical output after the refactoring. They exercise the full path: CLI command -> service accessor -> service -> registry -> adapter -> typed repository.

### 4.1 Note Operations (all 5 entity types)

| Scenario | Command | Expected |
|----------|---------|----------|
| Add note to epic | `shark epic note add E21 --content="test" --type=comment` | Note created, no error |
| Add note to feature | `shark feature note add E21-F02 --content="test" --type=comment` | Note created, no error |
| Add note to task | `shark task note add E21-F02-001 --content="test" --type=comment` | Note created, no error |
| List notes | `shark task notes E21-F02-001` | Lists notes correctly |

### 4.2 Context Operations (all 5 entity types)

| Scenario | Command | Expected |
|----------|---------|----------|
| Set context on epic | `shark epic context set E21 --field current_step --value "testing"` | Context updated |
| Get context from epic | `shark epic context get E21` | Returns context JSON |
| Set context on feature | `shark feature context set E21-F02 --field current_step --value "testing"` | Context updated |
| Clear context | `shark epic context clear E21` | Context cleared |

### 4.3 Resume Operations

| Scenario | Command | Expected |
|----------|---------|----------|
| Resume task | `shark task resume E21-F02-001` | Full task resume context |
| Resume epic | `shark epic resume E21` | Full epic resume context |

### 4.4 Regression Smoke

After all F02 tasks complete, run the full test suite:
```bash
make fmt && make lint && make test
```

This is the ultimate quality gate. If any test fails, F02 is not complete.

---

## 5. Test Execution Order (TDD Sequence)

The test plan maps to the implementation task order from the architecture document:

| Task | Tests Written First (TDD) | Implementation | Validation |
|------|--------------------------|----------------|------------|
| Task 1: NoteService | TC-01 through TC-06, TC-21-TC-24 | Refactor `note_service.go` | `make test` |
| Task 2: ContextService | TC-07 through TC-11, TC-25 (ContextService part) | Refactor `context_service.go` | `make test` |
| Task 3: ResumeService | TC-12 through TC-15, TC-25 (ResumeService part) | Refactor `resume_service.go` | `make test` |
| Task 4: CLI Wiring | TC-16 through TC-19 | Update `services_global.go` | `make test` |
| Task 5: Validation | TC-20 (E2E), Metrics 3.2-3.4 | Full regression | `make fmt && make lint && make test` |

**Dependency note**: Tasks 1-3 modify separate service files and can be developed in parallel. Task 4 depends on all three because it updates constructor call sites. Task 5 is the final validation pass.

---

## 6. Risk-Based Test Priorities

| Priority | What | Why | Failure Impact |
|----------|------|-----|----------------|
| **P0** | Zero behavioral regression (`make test` passes) | This is a pure refactoring -- any behavior change is a bug | Blocks all work |
| **P0** | Switch statement elimination | Core value prop of F02 | Feature not delivering its promise |
| **P0** | Constructor nil-check panics | Catches wiring bugs at startup | Silent failures in production |
| **P1** | Parameterized tests across all 5 entity types | Validates extensibility claim | New entity types may not work |
| **P1** | CLI E2E smoke tests | Validates user-facing behavior | Users encounter regressions |
| **P2** | Line reduction metrics | Quantitative validation | Does not block functionality |
| **P2** | Benchmark performance | Registry lookup is O(1) map access | Negligible real-world impact |

---

## 7. Exit Criteria

F02 test plan is actionable for TDD when:

- [ ] Every acceptance criterion in the PRD has at least one test case mapped
- [ ] Component test strategy specifies mock patterns for all 3 services
- [ ] Integration points (CLI wiring, `services_global.go`) have test coverage
- [ ] Non-functional metrics have measurement procedures
- [ ] Test execution order aligns with implementation task dependencies
- [ ] Existing tests that must be adapted are enumerated

---

*Traces to: [PRD](prd.md) Stories 1-6, Error Stories 1-2 | [Architecture](02-architecture.md) Sections 3-7 | [UAT Plan](../uat-acceptance-plan.md) Scenario 1 (P2), Metrics 3, 5*
