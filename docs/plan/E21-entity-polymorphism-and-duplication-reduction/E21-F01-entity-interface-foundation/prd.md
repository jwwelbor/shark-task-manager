# F01: Entity Interface Foundation

**Feature Key**: E21-F01
**Epic**: E21 - Entity Polymorphism and Duplication Reduction
**Complexity Tier**: STANDARD (score 13/27)
**Execution Order**: 1 (critical path -- all other features depend on this)
**Effort Estimate**: M (3-5 tasks)
**Risk**: Zero (purely additive, no existing code changes beyond adding methods)

---

## Goal

Establish the foundational polymorphic abstractions (Entity interface, EntityRepository interface, EntityRegistry) that enable all subsequent features (F02-F05) to unify cross-cutting logic.

**Traces to epic goal**: The epic aims to eliminate ~1,255+ lines of cross-entity duplication and reduce the effort to add a new entity type from ~2 weeks / 15+ files to ~2 days / 3-5 files. F01 creates the interface contracts and adapter layer that make this unification possible. Without F01, none of the service unification (F02-F05) can proceed.

---

## Key Personas

From the [epic personas](../personas.md):

- **Codebase Maintainer**: Needs a single interface contract that all entities satisfy, so cross-cutting bug fixes and features apply once. F01 creates that contract.
- **New Entity Author**: Needs a clear, documented interface to implement when adding a new entity type. F01 defines that interface and the adapter/registry pattern for plugging in.
- **Service Layer Consumer**: Needs consistent entity access patterns. F01 provides the EntityRepository and EntityRegistry that will eventually enable generic CLI/API commands.

---

## Requirements (MoSCoW)

### Must Have

**REQ-F-001: Entity Interface Definition**
- *As a codebase maintainer*, I want a single `models.Entity` interface with accessor methods for the 10 shared fields, so that cross-cutting services can operate on any entity polymorphically.
- Acceptance: Interface defined in `internal/models/entity.go` with 14 methods (GetID, GetKey, GetTitle, GetSlug, GetEntityType, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt, Validate). `EntityType` enum includes all 5 types.

**REQ-F-002: Entity Interface Implementation on All Models**
- *As a codebase maintainer*, I want all 5 existing model structs to satisfy the Entity interface, so that I can pass any entity to cross-cutting services.
- Acceptance: Compile-time interface satisfaction checks (`var _ Entity = (*Epic)(nil)` etc.) pass for all 5 models. Existing direct field access (`epic.Key`, `task.FeatureID`) continues to work unchanged. All existing tests pass without modification.

**REQ-F-003: EntityRepository Interface Definition**
- *As a new entity author*, I want a standard polymorphic repository interface, so that my entity automatically works with shared services.
- Acceptance: `EntityRepository` interface defined in `internal/services/` with GetByKey, GetByID, UpdateStatus, Update, GetContextData, UpdateContextData methods.

**REQ-F-004: EntityRepository Adapters for Existing Repositories**
- *As a codebase maintainer*, I want adapter implementations wrapping each existing typed repository, so that existing repositories work with the new interface without rewriting them.
- Acceptance: 5 adapters (Epic, Feature, Task, Bug, ChangeCard) implement EntityRepository. Type assertions include clear error messages. Existing typed repository methods remain available.

**REQ-F-012: EntityRegistry Implementation**
- *As a new entity author*, I want to register my entity in one place, so that all cross-cutting services recognize it automatically.
- Acceptance: Thread-safe `EntityRegistry` supports Register, GetRepository, MustGetRepository, RegisteredTypes. Duplicate registration panics (programming error). Unregistered type lookup returns a clear error.

### Must Have (Prerequisite)

**ChangeCard Type Normalization**
- *As a codebase maintainer*, I want ChangeCard.Slug and ChangeCard.FilePath normalized from `string` to `*string`, so that all entities have consistent field types and the Entity interface can use uniform accessor signatures.
- Acceptance: ChangeCard model fields are `*string`. Repository scan logic, service code, CLI commands, and tests updated. All existing ChangeCard tests pass with updated type assertions.

### Non-Functional (Must Have)

| Requirement | Target | Measurement |
|---|---|---|
| REQ-NF-001: Zero Behavioral Changes | 100% existing test pass rate | `make test` |
| REQ-NF-002: Typed Access Preservation | Zero compile errors in existing code | `make build` |
| REQ-NF-003: Interface Dispatch Overhead | Less than 5ns per call | Benchmark test (informational) |
| REQ-NF-004: Registry Lookup Performance | O(1) map access, less than 50ns | Benchmark test (informational) |
| REQ-NF-005: Entity Interface Test Coverage | 100% of interface methods for all 5 types | `go test -cover` |
| REQ-NF-009: Phase Boundary Quality Gate | Zero lint warnings, zero test failures | `make fmt && make lint && make test` |

---

## Scope

### In Scope

1. **Entity interface definition** (`internal/models/entity.go`)
   - 14 accessor/mutator methods for shared fields
   - `EntityType` enum with all 5 entity types
   - Compile-time interface satisfaction checks for all 5 models

2. **Entity interface implementation on all 5 models**
   - Add ~14 accessor methods to each model struct
   - Handle type differences (e.g., `*string` vs `string` for Slug/FilePath)

3. **ChangeCard type normalization** (prerequisite)
   - Normalize `ChangeCard.Slug` from `string` to `*string`
   - Normalize `ChangeCard.FilePath` from `string` to `*string`
   - Update repository scan logic, service code, CLI commands, and tests

4. **EntityRepository interface** (`internal/services/entity_repository.go`)
   - Polymorphic CRUD and cross-cutting data access methods
   - Defined at the services layer (consumer-side, per project convention)

5. **EntityRepository adapters** (one per entity type)
   - `EpicRepositoryAdapter`, `FeatureRepositoryAdapter`, `TaskRepositoryAdapter`, `BugRepositoryAdapter`, `ChangeCardRepositoryAdapter`
   - Each wraps the existing typed repository with type assertions

6. **EntityRegistry** (`internal/services/entity_registry.go`)
   - Thread-safe map of `EntityType -> EntityRepository`
   - Register, GetRepository, MustGetRepository, RegisteredTypes methods

7. **Tests**
   - Compile-time interface satisfaction checks
   - Accessor method tests for all 5 entity types
   - Adapter tests with mock repositories
   - Registry registration and lookup tests

### Out of Scope

- Refactoring any existing service to use the Entity interface (that is F02-F05)
- CLI command changes
- Database schema changes
- HTTP API handler changes
- EntityService with TransitionStatus (that is F03)
- CLI accessor consolidation to use EntityRegistry (that is F02)

---

## Acceptance Criteria

### AC1: Entity Interface Defined
Given the `internal/models/entity.go` file exists,
When a developer inspects the interface,
Then `models.Entity` has exactly 14 methods: GetID, GetKey, GetTitle, GetSlug, GetEntityType, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt, Validate.

### AC2: EntityType Enum Complete
Given the `models.EntityType` type is defined,
When all entity types are listed,
Then the enum includes: Epic, Feature, Task, Bug, ChangeCard (5 values).

### AC3: All Models Implement Entity Interface
Given compile-time interface satisfaction checks are present,
When `make build` is run,
Then all 5 model structs (Epic, Feature, Task, Bug, ChangeCard) compile without errors as Entity interface implementations.

### AC4: Backward Compatibility Preserved
Given existing code that accesses fields directly (e.g., `epic.Key`, `task.FeatureID`),
When `make build && make test` is run,
Then all existing code compiles and all existing tests pass without modification (except ChangeCard-related tests updated for the `*string` type change).

### AC5: EntityRepository Interface Defined
Given the `internal/services/entity_repository.go` file exists,
When a developer inspects the interface,
Then `EntityRepository` has methods: GetByKey, GetByID, UpdateStatus, Update, GetContextData, UpdateContextData -- all operating on `models.Entity`.

### AC6: Five Adapter Implementations
Given one adapter per entity type,
When each adapter is instantiated with a typed repository,
Then it satisfies the `EntityRepository` interface and delegates to the typed repository with proper type assertions and clear error messages on type mismatch.

### AC7: EntityRegistry Operational
Given an `EntityRegistry` instance,
When all 5 entity types are registered,
Then GetRepository returns the correct adapter for each type, MustGetRepository panics for unregistered types, RegisteredTypes returns all 5 types, and duplicate registration panics.

### AC8: ChangeCard Type Normalization
Given the ChangeCard model,
When Slug and FilePath fields are inspected,
Then both are `*string` (matching Epic, Feature, Task, Bug patterns). Repository scan logic, service code, and tests are updated accordingly.

### AC9: Quality Gate Passes
Given all F01 code is complete,
When `make fmt && make lint && make test` is run,
Then there are zero formatting changes, zero lint warnings, and zero test failures.

### AC10: Test Coverage
Given new test files exist for the Entity interface, adapters, and registry,
When `go test -cover` is run on relevant packages,
Then 100% of Entity interface methods are tested for all 5 types, and registry operations have full coverage.

---

## Dependencies

- **External**: None -- F01 is the first feature and has no dependencies on other E21 features.
- **Internal prerequisite**: ChangeCard type normalization must be completed before Entity interface implementation on ChangeCard.
- **Downstream**: F02, F03, F04, F05 all depend on F01 completion.

---

## Technical Design Reference

See [architecture-design.md](../architecture-design.md) sections:
- Section 1: Entity Interface Definition (interface methods, implementation pattern, compile-time checks)
- Section 2: EntityRepository Adapter Pattern (interface definition, adapter implementation, Task/Bug context data handling)
- Section 3: EntityRegistry Design (registry struct, thread-safe operations, initialization pattern)

---

## Estimated Tasks

1. ChangeCard Slug/FilePath type normalization (`*string`)
2. Define Entity interface and EntityType enum
3. Implement Entity interface on all 5 model structs
4. Define EntityRepository interface and create 5 adapter implementations
5. Create EntityRegistry with registration and lookup

---

## Requirement Traceability

| Feature Requirement | Epic Requirement | Coverage |
|---|---|---|
| Entity interface (14 methods) | REQ-F-001 | Full |
| Implementation on 5 models | REQ-F-002 | Full |
| EntityRepository interface | REQ-F-003 | Full |
| 5 repository adapters | REQ-F-004 | Full |
| EntityRegistry | REQ-F-012 | Full |
| Zero behavioral changes | REQ-NF-001 | Full |
| Typed access preserved | REQ-NF-002 | Full |
| Interface dispatch overhead | REQ-NF-003 | Full |
| Registry lookup performance | REQ-NF-004 | Full |
| Test coverage 100% | REQ-NF-005 | Full |
| Quality gate passes | REQ-NF-009 | Full |

---

*Last Updated*: 2026-03-19
