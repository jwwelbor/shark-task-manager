# F02: Cross-Cutting Service Unification

**Feature Key**: E21-F02
**Epic**: E21 - Entity Polymorphism and Duplication Reduction
**Complexity Tier**: STANDARD (score 11/27)
**Execution Order**: 2 (depends on F01 completion)
**Effort Estimate**: L (5-8 tasks)
**Risk**: Medium (breadth of CLI commands exercising these services creates regression surface)

---

## Epic

- **Epic PRD**: [Entity Polymorphism and Duplication Reduction](../epic.md)
- **Epic Architecture**: [Architecture Design](../architecture-design.md)

---

## Goal

### Problem

Three cross-cutting services -- NoteService, ContextService, and ResumeService -- each maintain per-entity repository fields and multi-branch switch statements to dispatch operations by entity type. NoteService has 5 repository interface definitions, 5 setter methods, and 2 five-branch switch statements (`resolveEntityID`, `GetEntityDetails`). ContextService has the same pattern with `getContextJSON` and `setContextJSON`. ResumeService uses setter methods (`SetSessionRepo`, `SetBugRepo`, `SetChangeCardRepo`) for entity-specific wiring. The CLI accessor layer in `services_global.go` follows an identical init-repos-wire-setters pattern for each service.

When a 6th entity type is added (e.g., Sprint for E19), every one of these services must be modified: add a repository interface, add a setter method, add a switch branch, update the CLI accessor. This is the exact duplication the epic targets.

### Solution

Refactor NoteService, ContextService, and ResumeService to accept an `EntityRegistry` (created in F01) instead of per-entity repository fields. Replace switch-based dispatch with a single `registry.GetRepository(entityType)` call. Update CLI accessor wiring in `services_global.go` to initialize the registry once and pass it to all cross-cutting services. Remove all per-entity setter methods from these three services.

### Impact

- Eliminate 14+ switch branches (5 branches x 2 switches in NoteService + 5 branches x 2 switches in ContextService, reduced to 0 switches)
- Remove ~15 per-entity repository interface definitions from cross-cutting services
- Remove ~6 setter methods and their corresponding CLI wiring calls
- Net reduction of ~230 lines across cross-cutting services
- Adding a 6th entity type requires zero modifications to NoteService, ContextService, or ResumeService (they automatically support any registered entity type)

---

## Key Personas

From the [epic personas](../personas.md):

- **Codebase Maintainer**: Directly benefits from eliminating switch statements and setter methods. Bug fixes to entity resolution logic apply once via the registry pattern instead of being replicated per entity type.
- **New Entity Author**: Adding a new entity type no longer requires modifying NoteService, ContextService, or ResumeService. A single `EntityRegistry.Register()` call in `services_global.go` makes the new entity visible to all cross-cutting services.
- **Service Layer Consumer**: CLI accessors in `services_global.go` become simpler. Instead of initializing 5 repositories and calling 5 setter methods per cross-cutting service, a single registry is created and shared.

---

## User Stories

### Must-Have Stories

**Story 1**: As a codebase maintainer, I want NoteService to resolve entity IDs and entity details via the EntityRegistry, so that adding a new entity type does not require modifying NoteService.

**Acceptance Criteria**:
- [ ] `resolveEntityID` uses `registry.GetRepository(entityType).GetByKey()` instead of a 5-branch switch
- [ ] `GetEntityDetails` uses registry-based dispatch instead of a 5-branch switch
- [ ] All 5 existing `SetXxxRepo()` setter methods are removed from NoteService
- [ ] NoteService constructor accepts `*EntityRegistry` as a dependency
- [ ] All existing note operations (add, list, get) produce identical results for all 5 entity types

**Story 2**: As a codebase maintainer, I want ContextService to read and write context data via the EntityRegistry, so that context operations extend to new entities automatically.

**Acceptance Criteria**:
- [ ] `getContextJSON` uses `registry.GetRepository(entityType).GetContextData()` instead of a 5-branch switch
- [ ] `setContextJSON` uses `registry.GetRepository(entityType).UpdateContextData()` instead of a 5-branch switch
- [ ] All `SetXxxRepo()` setter methods are removed from ContextService
- [ ] ContextService constructor accepts `*EntityRegistry` as a dependency
- [ ] All existing context operations (get, set, clear) produce identical results for all 5 entity types

**Story 3**: As a codebase maintainer, I want ResumeService to use the EntityRegistry for entity lookup, so that resume operations extend to new entities automatically.

**Acceptance Criteria**:
- [ ] Entity lookup uses `registry.GetRepository(entityType)` instead of per-entity repository fields
- [ ] All `SetXxxRepo()` setter methods are removed from ResumeService
- [ ] ResumeService constructor accepts `*EntityRegistry` as a dependency
- [ ] All existing resume operations produce identical results for all 5 entity types

**Story 4**: As a codebase maintainer, I want `services_global.go` to create a single `EntityRegistry` and pass it to all cross-cutting services, so that the init-repos-wire-setters pattern is replaced by a single registry initialization.

**Acceptance Criteria**:
- [ ] A `GetEntityRegistry()` function exists in `services_global.go` (lazy-initialized via `sync.Once`)
- [ ] `GetEntityRegistry()` registers all 5 entity types with their `EntityRepository` adapters (from F01)
- [ ] `GetNoteService()`, `GetContextService()`, `GetResumeService()` use the shared registry instead of manual repo wiring
- [ ] All per-entity setter calls in `services_global.go` for these 3 services are removed
- [ ] All CLI commands that use notes, context, or resume produce identical behavior

---

### Should-Have Stories

**Story 5**: As a test author, I want to test NoteService and ContextService with a mock EntityRegistry, so that I test cross-cutting dispatch logic once instead of writing per-entity mock setups.

**Acceptance Criteria**:
- [ ] NoteService tests use a mock `EntityRegistry` with mock `EntityRepository` entries
- [ ] ContextService tests use a mock `EntityRegistry`
- [ ] Tests verify behavior for all registered entity types via parameterized test cases
- [ ] Per-entity mock repository interfaces previously defined in NoteService and ContextService test files are removed

**Story 6**: As a codebase maintainer, I want the per-entity repository interface definitions removed from NoteService, ContextService, and ResumeService, so that these services have no entity-type-specific code.

**Acceptance Criteria**:
- [ ] `NoteEpicRepository`, `NoteFeatureRepository`, `NoteTaskRepository`, `NoteChangeCardRepository`, `NoteBugRepository` interfaces are removed from `note_service.go`
- [ ] Equivalent per-entity interfaces are removed from `context_service.go` and `resume_service.go`
- [ ] No compilation errors after removal (all usage replaced by `EntityRegistry` dispatch)
- [ ] Net line reduction across these 3 service files is at least 150 lines

---

### Edge Case & Error Stories

**Error Story 1**: As a codebase maintainer, when a CLI command references an entity type that is not registered in the EntityRegistry, I want a clear error message stating which entity type is unsupported, so that I can diagnose the registration gap.

**Acceptance Criteria**:
- [ ] `registry.GetRepository(entityType)` returns an error with the message `no repository registered for entity type "<type>"`
- [ ] NoteService, ContextService, and ResumeService propagate this error to the caller with added context (e.g., "failed to resolve entity for note operation: no repository registered for entity type "sprint"")
- [ ] The error does not panic or crash the CLI

**Error Story 2**: As a codebase maintainer, when a cross-cutting service receives a nil EntityRegistry, I want the service to fail at construction time with a clear panic message, so that the wiring bug is caught immediately during development.

**Acceptance Criteria**:
- [ ] NoteService, ContextService, and ResumeService constructors panic if the `registry` parameter is nil
- [ ] The panic message identifies which service received a nil registry (e.g., "NoteService: EntityRegistry must not be nil")

---

## Requirements

### Functional Requirements

**Category: NoteService Refactoring**

1. **REQ-F02-001**: NoteService Registry-Based Entity Resolution
   - **Description**: NoteService's `resolveEntityID` method must use the EntityRegistry to look up entities by type and key, replacing the current 5-branch switch statement.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-005
   - **Acceptance Criteria**:
     - [ ] `resolveEntityID(ctx, entityType, key)` calls `registry.GetRepository(entityType)` then `repo.GetByKey(ctx, key)` then returns `entity.GetID()`
     - [ ] No switch statement on entity type remains in `resolveEntityID`

2. **REQ-F02-002**: NoteService Registry-Based Entity Details
   - **Description**: NoteService's `GetEntityDetails` method must use the EntityRegistry for entity lookup, replacing the current 5-branch switch statement.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-005
   - **Acceptance Criteria**:
     - [ ] `GetEntityDetails(ctx, entityType, key)` calls `registry.GetRepository(entityType)` then `repo.GetByKey(ctx, key)` then returns `NoteEntityDetails{Key: entity.GetKey(), Title: entity.GetTitle()}`
     - [ ] No switch statement on entity type remains in `GetEntityDetails`

3. **REQ-F02-003**: NoteService Constructor Change
   - **Description**: NoteService constructor must accept `*EntityRegistry` instead of individual per-entity repository parameters.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-005
   - **Acceptance Criteria**:
     - [ ] `NewNoteService(noteRepo NoteEntityNoteRepository, registry *EntityRegistry)` is the constructor signature
     - [ ] Constructor panics if `registry` is nil
     - [ ] All 5 `SetXxxRepo()` setter methods are removed

**Category: ContextService Refactoring**

4. **REQ-F02-004**: ContextService Registry-Based Context Get
   - **Description**: ContextService's `getContextJSON` method must use the EntityRegistry for context data retrieval, replacing the current 5-branch switch statement.
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-006
   - **Acceptance Criteria**:
     - [ ] `getContextJSON(ctx, entityType, key)` calls `registry.GetRepository(entityType)` then `repo.GetByKey(ctx, key)` for entity ID, then `repo.GetContextData(ctx, id)`
     - [ ] No switch statement on entity type remains in `getContextJSON`

5. **REQ-F02-005**: ContextService Registry-Based Context Set
   - **Description**: ContextService's `setContextJSON` method must use the EntityRegistry for context data updates, replacing the current 5-branch switch statement.
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-006
   - **Acceptance Criteria**:
     - [ ] `setContextJSON(ctx, entityType, key, data)` calls `registry.GetRepository(entityType)` then `repo.GetByKey(ctx, key)` for entity ID, then `repo.UpdateContextData(ctx, id, data)`
     - [ ] No switch statement on entity type remains in `setContextJSON`

6. **REQ-F02-006**: ContextService Constructor Change
   - **Description**: ContextService constructor must accept `*EntityRegistry` instead of individual per-entity repository parameters.
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-006
   - **Acceptance Criteria**:
     - [ ] `NewContextService(registry *EntityRegistry)` is the constructor signature
     - [ ] Constructor panics if `registry` is nil
     - [ ] All `SetXxxRepo()` setter methods are removed

**Category: ResumeService Refactoring**

7. **REQ-F02-007**: ResumeService Registry-Based Entity Lookup
   - **Description**: ResumeService's entity lookup logic must use the EntityRegistry for entity retrieval, replacing per-entity repository fields and any type-based dispatch.
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-007
   - **Acceptance Criteria**:
     - [ ] Entity lookup uses `registry.GetRepository(entityType).GetByKey(ctx, key)`
     - [ ] All `SetXxxRepo()` setter methods are removed from ResumeService
     - [ ] ResumeService constructor accepts `*EntityRegistry`

**Category: CLI Accessor Wiring**

8. **REQ-F02-008**: CLI EntityRegistry Initialization
   - **Description**: `services_global.go` must create and cache a shared `EntityRegistry` instance with all 5 entity types registered, using `sync.Once` for thread-safe lazy initialization.
   - **User Story**: Story 4
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-012 (partial), Epic REQ-F-013
   - **Acceptance Criteria**:
     - [ ] `GetEntityRegistry() *EntityRegistry` function exists in `services_global.go`
     - [ ] Uses `sync.Once` for initialization (consistent with existing `GetDB()` pattern)
     - [ ] Registers all 5 entity types: Epic, Feature, Task, Bug, ChangeCard
     - [ ] Each registration uses the `EntityRepository` adapter from F01 wrapping the typed repository

9. **REQ-F02-009**: CLI Cross-Cutting Service Wiring Update
   - **Description**: `GetNoteService()`, `GetContextService()`, and `GetResumeService()` in `services_global.go` must be updated to pass the shared `EntityRegistry` instead of creating per-entity repositories and calling setter methods.
   - **User Story**: Story 4
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-013
   - **Acceptance Criteria**:
     - [ ] `GetNoteService()` calls `NewNoteService(noteRepo, GetEntityRegistry())`
     - [ ] `GetContextService()` calls `NewContextService(GetEntityRegistry())`
     - [ ] `GetResumeService()` calls `NewResumeService(..., GetEntityRegistry())`
     - [ ] All per-entity `SetXxxRepo()` calls are removed from these 3 accessor functions
     - [ ] Net line reduction in `services_global.go` for these 3 functions is at least 30 lines

**Category: Per-Entity Interface Removal**

10. **REQ-F02-010**: Remove Per-Entity Repository Interfaces from Cross-Cutting Services
    - **Description**: The per-entity repository interfaces defined within NoteService, ContextService, and ResumeService source files must be removed, since entity access is now via the EntityRegistry.
    - **User Story**: Story 6
    - **Priority**: Should-Have
    - **Traces to**: Epic REQ-F-005, REQ-F-006, REQ-F-007
    - **Acceptance Criteria**:
      - [ ] `NoteEpicRepository`, `NoteFeatureRepository`, `NoteTaskRepository`, `NoteBugRepository`, `NoteChangeCardRepository` interfaces removed from `note_service.go`
      - [ ] Equivalent per-entity interfaces removed from `context_service.go`
      - [ ] Equivalent per-entity interfaces removed from `resume_service.go`
      - [ ] No compilation errors after removal

---

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF02-001**: Zero Behavioral Changes
   - **Description**: All existing CLI commands that exercise note, context, and resume operations must produce identical output after the refactoring.
   - **Measurement**: All existing tests pass without modification (except test infrastructure updates for new constructor signatures)
   - **Target**: 100% test pass rate with zero behavioral deltas
   - **Traces to**: Epic REQ-NF-001

**Performance**

2. **REQ-NF02-002**: Registry Lookup Overhead
   - **Description**: The registry lookup replacing switch statements must not introduce meaningful latency.
   - **Measurement**: Benchmark `registry.GetRepository(entityType)` vs. direct switch dispatch
   - **Target**: Less than 50ns per lookup (O(1) map access)
   - **Traces to**: Epic REQ-NF-004
   - **Justification**: Registry lookups happen once per CLI command invocation. The target is orders of magnitude below database I/O cost (~1-10ms).

**Test Coverage**

3. **REQ-NF02-003**: Cross-Cutting Service Test Coverage
   - **Description**: Refactored NoteService, ContextService, and ResumeService must maintain or improve test coverage.
   - **Measurement**: `go test -cover` on `internal/services/`
   - **Target**: Coverage for refactored methods must not decrease from pre-refactoring levels
   - **Traces to**: Epic REQ-NF-006

**Maintainability**

4. **REQ-NF02-004**: Code Reduction
   - **Description**: The refactoring must achieve a net line reduction in the affected files.
   - **Measurement**: `wc -l` on `note_service.go`, `context_service.go`, `resume_service.go`, `services_global.go` before and after
   - **Target**: Net reduction of at least 150 lines across these 4 files (conservative; expected ~230 based on architecture analysis)
   - **Traces to**: Epic REQ-NF-007

**Build Quality Gate**

5. **REQ-NF02-005**: Quality Gate
   - **Description**: `make fmt && make lint && make test` must pass after every task in this feature.
   - **Measurement**: CI/CD pipeline or manual execution
   - **Target**: Zero lint warnings, zero test failures
   - **Traces to**: Epic REQ-NF-009

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: NoteService Uses Registry**
- **Given** NoteService is constructed with an EntityRegistry containing all 5 entity types
- **When** `resolveEntityID` is called with `entityType=EntityTypeEpic` and a valid epic key
- **Then** the entity ID is resolved via `registry.GetRepository(EntityTypeEpic).GetByKey()` without any switch statement
- **And** the result is identical to the pre-refactoring behavior

**Scenario 2: ContextService Uses Registry**
- **Given** ContextService is constructed with an EntityRegistry containing all 5 entity types
- **When** `setContextJSON` is called for a task entity
- **Then** context data is updated via `registry.GetRepository(EntityTypeTask).UpdateContextData()` without any switch statement
- **And** the stored context data matches the input exactly

**Scenario 3: CLI Wiring Simplified**
- **Given** `services_global.go` has been updated with `GetEntityRegistry()`
- **When** `GetNoteService()` is called
- **Then** it returns a NoteService constructed with the shared EntityRegistry
- **And** no per-entity setter methods are called

**Scenario 4: Unregistered Entity Type**
- **Given** an EntityRegistry with only 3 of 5 entity types registered
- **When** NoteService attempts to resolve an entity of an unregistered type
- **Then** a clear error is returned: `no repository registered for entity type "<type>"`
- **And** the CLI does not panic or crash

**Scenario 5: End-to-End Note Operations**
- **Given** the full application with refactored NoteService
- **When** `shark task note add E21-F02-001 --type comment "test"` is executed
- **Then** the note is created successfully
- **And** `shark task notes E21-F02-001` displays the note

**Scenario 6: End-to-End Context Operations**
- **Given** the full application with refactored ContextService
- **When** `shark feature context set E21-F02 --field current_step --value "testing"` is executed
- **Then** the context field is updated
- **And** `shark feature context get E21-F02` returns the updated value

**Scenario 7: Quality Gate**
- **Given** all F02 refactoring is complete
- **When** `make fmt && make lint && make test` is run
- **Then** there are zero formatting changes, zero lint warnings, and zero test failures

---

## Out of Scope

### Explicitly Excluded

1. **Status Transition Unification (EntityService.TransitionStatus)**
   - **Why**: Status transition logic is a separate concern with higher complexity and risk. It requires the EntityService composition pattern and TransitionFeatures config, which are distinct from registry-based dispatch.
   - **Future**: Addressed in F03 (Status Transition Unification), which also depends on F01.
   - **Workaround**: Entity-specific services continue to own their TransitionStatus methods until F03.

2. **Document Operations Unification (LinkDocument, UnlinkDocument)**
   - **Why**: Document operations are already partially abstracted via `document_helpers.go`. Their unification is a separate, lower-priority effort.
   - **Future**: Addressed in F04 (Document Operations Unification).

3. **Template Placeholder Unification (EntityPlaceholders)**
   - **Why**: Template placeholders are orthogonal to cross-cutting service dispatch. They do not use switch statements in the same way.
   - **Future**: Addressed in F05 (Template Placeholder Unification).

4. **Entity-Specific Service Refactoring (EpicService, FeatureService, TaskService internals)**
   - **Why**: F02 only refactors the 3 cross-cutting services (NoteService, ContextService, ResumeService) and their CLI wiring. Entity-specific services are not modified beyond removing setter calls.

5. **Database Schema Changes**
   - **Why**: F02 is a pure service-layer refactoring. No database tables, columns, or migrations are involved.

6. **HTTP API Handler Changes**
   - **Why**: HTTP handlers are not modified. They continue to call entity-specific services. Cross-cutting HTTP endpoints (if any) would be a separate effort.

---

## Success Metrics

### Primary Metrics

1. **Switch Statement Elimination**
   - **What**: Count of entity-type switch branches in NoteService, ContextService, ResumeService
   - **Target**: 0 switch branches (reduced from 14+)
   - **Timeline**: After all F02 tasks complete
   - **Measurement**: `grep -c "case models.EntityType" note_service.go context_service.go resume_service.go`

2. **Setter Method Elimination**
   - **What**: Count of `SetXxxRepo()` methods on cross-cutting services
   - **Target**: 0 setter methods (reduced from ~6)
   - **Timeline**: After all F02 tasks complete
   - **Measurement**: `grep -c "func (s \*.*Service) Set.*Repo" note_service.go context_service.go resume_service.go`

3. **Net Line Reduction**
   - **What**: Total lines in `note_service.go` + `context_service.go` + `resume_service.go` + `services_global.go`
   - **Target**: Reduction of at least 150 lines from pre-refactoring baseline (current total: ~1,339 lines)
   - **Timeline**: After all F02 tasks complete
   - **Measurement**: `wc -l` on the 4 files

### Secondary Metrics

- **New Entity Extension Effort**: Adding a 6th entity type requires 0 modifications to NoteService, ContextService, or ResumeService (only 1 `Register()` call in `services_global.go`)
- **Test Pass Rate**: 100% of existing tests pass after refactoring

---

## Dependencies & Integrations

### Dependencies

- **F01 (Entity Interface Foundation)**: F02 requires the `EntityRepository` interface, the 5 adapter implementations, and the `EntityRegistry` created in F01. F01 must be complete and merged before F02 work begins.
- **Existing typed repositories**: F02 wraps existing repositories via F01 adapters. No changes to the typed repositories themselves.

### Integration Points

- **`services_global.go`**: The CLI wiring layer. F02 replaces the per-entity init-wire-set pattern with a single `GetEntityRegistry()` function.
- **Existing CLI commands**: All commands that call `GetNoteService()`, `GetContextService()`, or `GetResumeService()` are implicitly affected. No command code changes are required -- only the service constructor signatures change.
- **Existing tests**: Test files for NoteService, ContextService, and ResumeService must be updated to use the new constructor signatures (pass `EntityRegistry` instead of per-entity repos + setters).

---

## Estimated Tasks

1. Refactor NoteService to accept EntityRegistry and replace switch statements
2. Refactor ContextService to accept EntityRegistry and replace switch statements
3. Refactor ResumeService to accept EntityRegistry and replace setter methods
4. Create GetEntityRegistry() in services_global.go and update cross-cutting service wiring
5. Update NoteService tests to use mock EntityRegistry
6. Update ContextService and ResumeService tests to use mock EntityRegistry
7. End-to-end validation: run full test suite and verify all note/context/resume CLI operations

---

## Requirement Traceability

| Feature Requirement | Epic Requirement | Coverage |
|---|---|---|
| REQ-F02-001, REQ-F02-002, REQ-F02-003 (NoteService) | REQ-F-005 | Full |
| REQ-F02-004, REQ-F02-005, REQ-F02-006 (ContextService) | REQ-F-006 | Full |
| REQ-F02-007 (ResumeService) | REQ-F-007 | Full |
| REQ-F02-008 (EntityRegistry in CLI) | REQ-F-012 | Partial (registry creation; service mapping deferred to F03) |
| REQ-F02-009 (CLI wiring update) | REQ-F-013 | Partial (cross-cutting accessors; entity-specific accessors remain) |
| REQ-F02-010 (interface removal) | REQ-F-005, REQ-F-006, REQ-F-007 | Full |
| REQ-NF02-001 (zero behavioral changes) | REQ-NF-001 | Full |
| REQ-NF02-002 (registry performance) | REQ-NF-004 | Full |
| REQ-NF02-003 (test coverage) | REQ-NF-006 | Partial (cross-cutting services; EntityService coverage is F03) |
| REQ-NF02-004 (code reduction) | REQ-NF-007 | Partial (~150-230 lines of epic's 800+ target) |
| REQ-NF02-005 (quality gate) | REQ-NF-009 | Full |

---

*Last Updated*: 2026-03-19
