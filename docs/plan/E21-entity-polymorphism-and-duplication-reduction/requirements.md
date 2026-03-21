# Requirements

**Epic**: [Entity Polymorphism and Duplication Reduction](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for introducing entity polymorphism and reducing cross-entity duplication in Shark Task Manager.

**Requirement Traceability**: Each requirement maps to specific [user journeys](./user-journeys.md) and [personas](./personas.md).

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for the epic's goals; the refactoring fails without these
- **Should Have**: Important for full value delivery; workarounds exist temporarily
- **Could Have**: Valuable but deferrable; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

### Must Have Requirements

#### Entity Interface Foundation (F01)

**REQ-F-001**: Entity Interface Definition
- **Description**: Define a `models.Entity` interface with accessor methods for the 10 shared fields (ID, Key, Title, Slug, Description, Status, FilePath, ContextData, CreatedAt, UpdatedAt) plus EntityType, SetStatus, SetContextData, and Validate.
- **User Story**: As a codebase maintainer, I want a single interface that all entity types implement so that cross-cutting services can operate on any entity polymorphically.
- **Acceptance Criteria**:
  - [ ] `models.Entity` interface is defined in `internal/models/entity.go`
  - [ ] Interface includes: GetID, GetKey, GetTitle, GetSlug, GetEntityType, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt, Validate
  - [ ] Each accessor returns the appropriate Go type (int64, string, *string, time.Time)
  - [ ] `models.EntityType` enum includes: Epic, Feature, Task, Bug, ChangeCard
- **Related Journey**: Journey 1, Journey 4

**REQ-F-002**: Entity Interface Implementation on All Models
- **Description**: All 5 existing model structs (Epic, Feature, Task, Bug, ChangeCard) must implement the Entity interface via accessor methods added to each struct.
- **User Story**: As a codebase maintainer, I want all existing entities to satisfy the Entity interface so that I can pass any entity to cross-cutting services.
- **Acceptance Criteria**:
  - [ ] `models.Epic` implements `models.Entity` (compile-time verified via interface assignment)
  - [ ] `models.Feature` implements `models.Entity`
  - [ ] `models.Task` implements `models.Entity`
  - [ ] `models.Bug` implements `models.Entity`
  - [ ] `models.ChangeCard` implements `models.Entity`
  - [ ] Existing code that accesses fields directly (e.g., `epic.Key`) continues to work unchanged
  - [ ] All existing tests pass without modification
- **Related Journey**: Journey 1 Step 1, Journey 2

**REQ-F-003**: EntityRepository Interface Definition
- **Description**: Define an `EntityRepository` interface with generic CRUD and cross-cutting data access methods that work with the Entity interface.
- **User Story**: As a new entity author, I want a standard repository interface so that my entity automatically works with shared services.
- **Acceptance Criteria**:
  - [ ] `EntityRepository` interface defined with: GetByKey, GetByID, Create, Update, Delete, UpdateStatus, GetContextData, UpdateContextData
  - [ ] Methods accept/return `models.Entity` for polymorphic use
  - [ ] Interface is defined in `internal/services/` or `internal/repository/` (consumer-side)
- **Related Journey**: Journey 1

**REQ-F-004**: EntityRepository Adapters for Existing Repositories
- **Description**: Create adapter implementations that wrap each existing typed repository (EpicRepository, FeatureRepository, etc.) to satisfy the EntityRepository interface.
- **User Story**: As a codebase maintainer, I want existing repositories to work with the new EntityRepository interface without rewriting them.
- **Acceptance Criteria**:
  - [ ] Adapter for EpicRepository implements EntityRepository
  - [ ] Adapter for FeatureRepository implements EntityRepository
  - [ ] Adapter for TaskRepository implements EntityRepository
  - [ ] Adapter for BugRepository implements EntityRepository
  - [ ] Adapter for ChangeCardRepository implements EntityRepository
  - [ ] Adapters perform type assertions where needed (Entity -> concrete type) with clear error messages
  - [ ] Existing typed repository methods remain available for entity-specific operations
- **Related Journey**: Journey 1

#### Cross-Cutting Service Unification (F02)

**REQ-F-005**: NoteService Registry-Based Dispatch
- **Description**: Refactor NoteService to use an EntityRepository map (keyed by EntityType) instead of per-entity repository fields and switch statements.
- **User Story**: As a codebase maintainer, I want NoteService to resolve entity IDs via the registry so that adding a new entity type does not require modifying NoteService.
- **Acceptance Criteria**:
  - [ ] `resolveEntityID` method uses `registry[entityType].GetByKey()` instead of a 5-branch switch
  - [ ] `GetEntityDetails` method uses registry-based dispatch
  - [ ] All `SetXxxRepo()` setter methods are removed
  - [ ] NoteService constructor accepts an EntityRepository map or EntityRegistry
  - [ ] All existing note operations produce identical results (behavioral parity verified by tests)
- **Related Journey**: Journey 2, Journey 3

**REQ-F-006**: ContextService Registry-Based Dispatch
- **Description**: Refactor ContextService to use EntityRepository map instead of per-entity repository fields and switch statements.
- **User Story**: As a codebase maintainer, I want ContextService to use the registry so that context operations extend to new entities automatically.
- **Acceptance Criteria**:
  - [ ] `getContextJSON` and `setContextJSON` use `registry[entityType].GetContextData()` / `UpdateContextData()` instead of 5-branch switches
  - [ ] All `SetXxxRepo()` setter methods are removed
  - [ ] ContextService constructor accepts an EntityRepository map or EntityRegistry
  - [ ] All existing context operations produce identical results
- **Related Journey**: Journey 2, Journey 3

**REQ-F-007**: ResumeService Registry-Based Dispatch
- **Description**: Refactor ResumeService to use EntityRepository map instead of per-entity repository fields and switch statements.
- **User Story**: As a codebase maintainer, I want ResumeService to use the registry so that resume operations extend to new entities automatically.
- **Acceptance Criteria**:
  - [ ] Entity lookup uses registry-based dispatch
  - [ ] All `SetXxxRepo()` setter methods are removed
  - [ ] All existing resume operations produce identical results
- **Related Journey**: Journey 2

#### Status Transition Unification (F03)

**REQ-F-008**: Shared TransitionStatus Implementation
- **Description**: Extract the status transition logic (currently duplicated in 5 entity services, ~80 lines each) into a single `EntityService.TransitionStatus` method.
- **User Story**: As a codebase maintainer, I want status transition logic in one place so that bug fixes apply to all entity types simultaneously.
- **Acceptance Criteria**:
  - [ ] `EntityService.TransitionStatus(ctx, repo EntityRepository, key, targetStatus string, opts TransitionOptions) (*TransitionResult, error)` implemented
  - [ ] Handles all 10 steps: get entity, extract status, validate transition, normalize, check backward, require reason, update status, create rejection note, count children, resolve action
  - [ ] Entity-specific services delegate to EntityService.TransitionStatus and add only entity-specific pre/post logic
  - [ ] All 5 entity types produce identical transition behavior for shared logic
  - [ ] Status history recording works for all entity types
- **Related Journey**: Journey 2

**REQ-F-009**: Entity-Specific Transition Extensions
- **Description**: Each entity service must retain the ability to add entity-specific pre-transition and post-transition logic while delegating shared logic to EntityService.
- **User Story**: As a codebase maintainer, I want entity services to compose with EntityService so that unique behaviors (e.g., task dependency checks, feature progress cascade) are preserved.
- **Acceptance Criteria**:
  - [ ] TaskService can check dependencies before allowing transition
  - [ ] FeatureService can cascade status to parent epic after transition
  - [ ] EpicService can update feature rollup after transition
  - [ ] BugService can enforce triage-specific rules
  - [ ] ChangeCardService can enforce approval-specific rules
  - [ ] Extension points are clearly defined (pre-transition hook, post-transition hook, or composition pattern)
- **Related Journey**: Journey 2

#### Document Operations Unification (F04)

**REQ-F-010**: Shared Document Linking
- **Description**: Move LinkDocument, UnlinkDocument, and ListRelatedDocumentsByKey from individual entity services into EntityService.
- **User Story**: As a codebase maintainer, I want document linking in one place so that all entities get consistent behavior.
- **Acceptance Criteria**:
  - [ ] `EntityService.LinkDocument(ctx, repo EntityRepository, key, docPath, description string) error` implemented
  - [ ] `EntityService.UnlinkDocument(ctx, repo EntityRepository, key, docPath string) error` implemented
  - [ ] `EntityService.ListRelatedDocuments(ctx, repo EntityRepository, key string) ([]Document, error)` implemented
  - [ ] All 5 entity services delegate document operations to EntityService
  - [ ] Existing document linking behavior is preserved (verified by tests)
- **Related Journey**: Journey 3

#### Template Placeholder Unification (F05)

**REQ-F-011**: Base Entity Placeholders
- **Description**: Create a shared function that generates template placeholders from any Entity interface, with entity-specific placeholder functions extending the base.
- **User Story**: As a codebase maintainer, I want template placeholder generation to use the Entity interface so that shared fields produce consistent placeholders.
- **Acceptance Criteria**:
  - [ ] `EntityPlaceholders(entity models.Entity) map[string]string` generates placeholders for all shared fields
  - [ ] Entity-specific placeholder functions call EntityPlaceholders and add their unique fields
  - [ ] Existing template rendering produces identical output
- **Related Journey**: Journey 3

#### Entity Registry (F02 dependency)

**REQ-F-012**: EntityRegistry Implementation
- **Description**: Create an EntityRegistry that maps EntityType to EntityRepository and EntityTypeService, replacing per-entity setter methods in cross-cutting services.
- **User Story**: As a new entity author, I want to register my entity in one place so that all cross-cutting services recognize it automatically.
- **Acceptance Criteria**:
  - [ ] `EntityRegistry` struct defined with `Register(entityType, repo, service)` method
  - [ ] `GetRepository(entityType) (EntityRepository, error)` returns the registered repository
  - [ ] `GetService(entityType) (EntityTypeService, error)` returns the registered service
  - [ ] All 5 existing entity types are registered at application startup
  - [ ] Cross-cutting services receive the registry via constructor injection
  - [ ] Adding a 6th entity type requires only a `Register()` call, no modifications to existing cross-cutting services
- **Related Journey**: Journey 1

---

### Should Have Requirements

**REQ-F-013**: CLI Accessor Consolidation
- **Description**: Consolidate the per-entity CLI accessor functions (GetTaskService, GetEpicService, GetBugService, etc.) to use the EntityRegistry for cross-cutting operations.
- **User Story**: As a service layer consumer, I want unified CLI accessors so that adding a new entity does not require new accessor functions for cross-cutting operations.
- **Acceptance Criteria**:
  - [ ] `cli.GetEntityService(entityType)` accessor function exists alongside entity-specific accessors
  - [ ] Unified commands (status advance, status set, get, search) use registry-based dispatch
  - [ ] Entity-specific accessors remain for entity-specific commands (e.g., `shark bug triage`)
- **Related Journey**: Journey 1 Step 3

**REQ-F-014**: Cross-Cutting Service Test Consolidation
- **Description**: Replace duplicated entity-specific tests for cross-cutting operations with shared test suites that verify behavior via the Entity interface.
- **User Story**: As a test author, I want to test cross-cutting logic once instead of 5 times.
- **Acceptance Criteria**:
  - [ ] EntityService.TransitionStatus has a shared test suite with parameterized entity types
  - [ ] NoteService tests use mock EntityRepository instead of per-entity mocks
  - [ ] ContextService tests use mock EntityRepository
  - [ ] Entity-specific tests only cover entity-specific logic (dependencies, progress, triage)
- **Related Journey**: Journey 2

---

### Could Have Requirements

**REQ-F-015**: Generic CLI Command Builder
- **Description**: Create a generic command builder that generates get, list, create, update, delete, note, and context commands for any registered entity type.
- **User Story**: As a service layer consumer, I want to generate standard CLI commands for a new entity type without writing boilerplate command code.
- **Acceptance Criteria**:
  - [ ] `NewEntityCommand(entityType, service)` generates a Cobra command tree with standard subcommands
  - [ ] Entity-specific commands can extend the generated tree with custom subcommands
  - [ ] Existing CLI behavior is preserved
- **Related Journey**: Journey 1
- **Note**: This is F06 from the epic, rated XL. Deferrable to a follow-on epic if scope pressure requires it.

---

## Non-Functional Requirements

### Backward Compatibility

**REQ-NF-001**: Zero Behavioral Changes
- **Description**: All existing CLI commands, API endpoints, and service methods must produce identical behavior after the refactoring. This is a pure internal restructuring.
- **Measurement**: All existing tests pass without modification (excluding test infrastructure updates)
- **Target**: 100% test pass rate with zero behavioral deltas
- **Justification**: This is a refactoring epic. Users and AI agents must not notice any difference in Shark's external behavior.

**REQ-NF-002**: Typed Access Preservation
- **Description**: Existing code that accesses entity fields directly (e.g., `task.FeatureID`, `bug.Severity`) must continue to work. The Entity interface is additive, not a replacement.
- **Measurement**: Compile-time verification; existing code compiles without changes
- **Target**: Zero compile errors in code outside the refactored services
- **Justification**: Gradual adoption is critical. Entity-specific code must not be forced to use the generic interface.

### Performance

**REQ-NF-003**: Interface Dispatch Overhead
- **Description**: The performance overhead of interface dispatch (calling methods via Entity interface vs. direct field access) must be negligible.
- **Measurement**: Benchmark comparison of direct field access vs. interface method call
- **Target**: Less than 5ns overhead per interface method call (Go's typical vtable dispatch is ~1-2ns)
- **Justification**: Interface dispatch is used in service-layer orchestration, not in hot loops. The overhead is orders of magnitude below the database I/O cost.

**REQ-NF-004**: Registry Lookup Performance
- **Description**: EntityRegistry lookups must be O(1) via map access.
- **Measurement**: Benchmark of registry.GetRepository(entityType)
- **Target**: Less than 50ns per lookup
- **Justification**: Registry lookups happen once per CLI command invocation. Sub-microsecond is acceptable.

### Test Coverage

**REQ-NF-005**: Entity Interface Test Coverage
- **Description**: All Entity interface implementations must have compile-time interface satisfaction checks and runtime accessor tests.
- **Measurement**: Test coverage report for entity interface methods
- **Target**: 100% coverage of interface methods for all 5 entity types
- **Justification**: The Entity interface is the foundation of the shared service layer. Any implementation bug would cascade to all cross-cutting operations.

**REQ-NF-006**: Shared Service Test Coverage
- **Description**: EntityService must have comprehensive test coverage for all shared operations (status transition, document linking, etc.).
- **Measurement**: `go test -cover ./internal/services/`
- **Target**: 80%+ coverage for EntityService
- **Justification**: EntityService replaces logic previously spread across 5 services. Its correctness is multiplied by the number of entity types.

### Maintainability

**REQ-NF-007**: Code Reduction Target
- **Description**: The refactoring must measurably reduce total lines of code in the service layer.
- **Measurement**: `wc -l` on service files before and after
- **Target**: Net reduction of 800+ lines across service layer (conservative target from the 1,255-line duplication estimate, accounting for new shared code)
- **Justification**: The primary motivation is duplication reduction. If the shared code is larger than the eliminated duplicates, the refactoring has failed.

**REQ-NF-008**: New Entity Effort Reduction
- **Description**: Adding a new entity type must require significantly fewer files and less code than the current approach.
- **Measurement**: Count of files that must be created or modified for a new entity type
- **Target**: 5 or fewer files (vs. current 15+)
- **Justification**: This is the key developer experience improvement. The reduction must be concrete and verifiable.

### Build Quality Gate

**REQ-NF-009**: Phase Boundary Quality Gate
- **Description**: `make fmt && make lint && make test` must pass at each feature boundary (after F01, after F02, etc.). No feature may be merged with lint warnings or test failures.
- **Measurement**: CI/CD pipeline or manual execution at feature boundaries
- **Target**: Zero lint warnings, zero test failures at each boundary
- **Justification**: Incremental refactoring demands that the codebase is stable at every intermediate state.

---

*See also*: [Success Metrics](./success-metrics.md), [Scope](./scope.md)
