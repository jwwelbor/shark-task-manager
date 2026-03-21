# Scope Boundaries

**Epic**: [Entity Polymorphism and Duplication Reduction](./epic.md)

---

## Overview

This document explicitly defines what is included and excluded from this epic. Clear scope boundaries prevent over-abstraction and keep the refactoring focused on measurable duplication reduction.

---

## In Scope

### Included Layers

**1. Models Layer (`internal/models/`)**
- Define the Entity interface in `internal/models/entity.go`
- Add accessor methods to all 5 model structs (Epic, Feature, Task, Bug, ChangeCard)
- Add compile-time interface satisfaction checks
- Define EntityType enum with all entity types

**2. Service Layer (`internal/services/`)**
- Create EntityService with shared cross-cutting logic (status transitions, document linking, orchestrator action resolution)
- Refactor NoteService to use EntityRepository map / registry
- Refactor ContextService to use EntityRepository map / registry
- Refactor ResumeService to use EntityRepository map / registry
- Create EntityRegistry for entity type registration and lookup
- Refactor entity-specific services (EpicService, FeatureService, TaskService, BugService, ChangeCardService) to compose with EntityService for shared logic

**3. Repository Layer (`internal/repository/`) -- Adapters Only**
- Create EntityRepository interface
- Create adapter implementations wrapping each existing typed repository
- Existing typed repositories are NOT rewritten -- adapters provide the bridge

**4. Template Layer (`internal/templates/`)**
- Create shared EntityPlaceholders function for template placeholder generation
- Refactor entity-specific placeholder functions to extend the base

**5. CLI Accessor Layer (`internal/cli/services_global*.go`)**
- Update service wiring to use EntityRegistry
- Consolidate per-entity setter calls into registry registration

### Included Entity Types

All 5 existing entity types are in scope for interface implementation and adapter creation:
- Epic
- Feature
- Task
- Bug
- ChangeCard

### Included Features (from epic.md)

| Feature | Scope | Effort |
|---------|-------|--------|
| F01: Entity Interface Foundation | Define interface, implement on all models, create EntityRepository with adapters | M |
| F02: Cross-Cutting Service Unification | Refactor NoteService, ContextService, ResumeService to use registry | L |
| F03: Status Transition Unification | Extract shared TransitionStatus into EntityService | L |
| F04: Document Operations Unification | Move document linking to EntityService | S |
| F05: Template Placeholder Unification | Create shared placeholder generation | S |

---

## Out of Scope

### Explicitly Excluded Features

**1. Generic CLI Commands (F06 from epic.md)**
- **Why It's Out of Scope**: F06 is rated XL (10+ tasks, incremental per entity). The CLI command layer has ~24,000 lines across 76 files. Refactoring it alongside the service layer creates too large a blast radius for a single epic.
- **Future Consideration**: A follow-on epic can create a generic EntityCommand builder that generates CRUD, note, and context commands for any registered entity type.
- **Workaround**: Entity-specific CLI commands continue to work as-is. The service-layer unification (F01-F05) provides the foundation that makes CLI unification straightforward in the future.

**2. Repository Layer Rewrite**
- **Why It's Out of Scope**: Rewriting the 5 typed repositories into a single generic repository would require changing SQL queries, column mappings, and scan logic for all entity types. This is high risk and low value compared to the adapter approach.
- **Future Consideration**: If Go generics mature to support database scanning patterns cleanly, a generic repository could replace the typed ones.
- **Workaround**: EntityRepository adapters wrap existing typed repositories, providing polymorphic access without rewriting any SQL.

**3. Database Schema Changes**
- **Why It's Out of Scope**: The refactoring is purely at the Go application layer. No database tables, columns, indexes, or constraints need to change.
- **Justification**: The Entity interface maps to fields that already exist in all entity tables. No schema evolution is required.

**4. HTTP API Handler Refactoring**
- **Why It's Out of Scope**: HTTP handlers in `cmd/server/` are a separate entry point. Refactoring them to use the EntityRegistry is valuable but orthogonal to the service-layer unification.
- **Future Consideration**: HTTP handlers can adopt the EntityRegistry after the service layer is unified, using the same pattern as CLI accessors.
- **Workaround**: HTTP handlers continue to use entity-specific service accessors.

**5. CLI Command File Consolidation**
- **Why It's Out of Scope**: Merging the ~76 CLI command files into fewer files or using code generation is a separate organizational concern from the service-layer polymorphism.
- **Justification**: Command files are thin wrappers. Their duplication is a readability concern, not a correctness concern. Service-layer unification is the higher-value target.

**6. New Entity Type Addition**
- **Why It's Out of Scope**: This epic builds the infrastructure for easy entity addition but does not itself add any new entity types (no Milestone, Sprint, etc.).
- **Justification**: Adding a new entity type would mix refactoring with feature development, complicating validation of the refactoring's success.

---

### Edge Cases and Scenarios Not Covered

**1. Entity Types with No Status (Hypothetical)**
- **Impact**: Low -- all current entity types have status fields
- **Rationale**: The Entity interface includes GetStatus/SetStatus. If a future entity has no status concept, it can return a constant (e.g., "active") or the interface can be split into a StatusEntity sub-interface in a future refactoring.
- **Mitigation**: Document the assumption that all entities have status fields.

**2. Entities with Non-String Status Types**
- **Impact**: None currently -- all status types are string aliases (EpicStatus, TaskStatus, etc.)
- **Rationale**: The Entity interface standardizes on `GetStatus() string` and `SetStatus(string)`. Type-safe status values are preserved in the typed model structs for entity-specific code.
- **Mitigation**: Entity-specific services continue to use typed status values internally; the shared EntityService works with string representation.

**3. Multi-Parent Entities**
- **Impact**: None currently -- each entity has at most one parent
- **Rationale**: The Entity interface does not include parent accessor methods (GetParentID, GetParentType) because parent relationships vary by entity type (Feature has EpicID, Task has FeatureID, Bug has optional LinkedEntityKey). Parent logic remains in entity-specific services.
- **Mitigation**: If multi-parent entities are needed in the future, a separate ParentEntity interface can be defined.

**4. EntityRepository Adapter Type Assertion Failures**
- **Impact**: Low -- occurs only if an Entity is passed to the wrong repository
- **Rationale**: Adapters perform type assertions (e.g., `entity.(*models.Epic)`) which panic or return errors if the wrong entity type is provided. The EntityRegistry prevents this by routing entity types to their correct repository.
- **Mitigation**: Adapter type assertions return descriptive errors; registry guarantees correct routing.

---

## Alternative Approaches Considered But Rejected

**Alternative 1: Go Generics Instead of Interface**
- **Description**: Use Go generics (`EntityService[T models.Entity]`) instead of an interface-based approach.
- **Pros**: Stronger type safety; no type assertions in adapters; compiler catches more errors
- **Cons**: Go generics do not support method sets on type parameters (no `T.GetKey()` without interface constraints anyway); generics add complexity to constructor signatures and dependency injection; the Go community consensus is that interfaces are preferred for this pattern
- **Decision Rationale**: Rejected because Go generics still require an interface constraint, making them equivalent to the interface approach but with more complex syntax. The interface approach is idiomatic Go and well-understood by the contributor base.

**Alternative 2: Code Generation**
- **Description**: Use `go generate` to create per-entity service methods from a template, eliminating runtime polymorphism entirely.
- **Pros**: Zero runtime overhead; generated code is fully typed; no interface dispatch
- **Cons**: Generated code is harder to debug; template changes require regeneration; build toolchain becomes more complex; generated code still needs to be reviewed and tested
- **Decision Rationale**: Rejected because the runtime overhead of interface dispatch is negligible (~1-2ns) and the maintenance cost of code generation outweighs its benefits for this use case. Interface-based polymorphism is simpler to understand, debug, and maintain.

**Alternative 3: Embedded Struct for Shared Fields**
- **Description**: Create a `BaseEntity` struct with the 10 shared fields and embed it in all entity structs.
- **Pros**: Eliminates accessor method boilerplate; shared fields accessed directly via `entity.BaseEntity.Key`
- **Cons**: Go embedding is not true inheritance; method promotion conflicts when entity types override behavior; breaks existing field access patterns (`epic.Key` vs. `epic.BaseEntity.Key`); JSON serialization changes; requires significant model refactoring
- **Decision Rationale**: Rejected because it would break existing code that accesses fields directly (violating REQ-NF-002) and introduces Go-specific pitfalls around method promotion and JSON tags. The interface approach is additive and non-breaking.

---

## Dependency Map

```
F01: Entity Interface Foundation
 |
 +---> F02: Cross-Cutting Service Unification (depends on F01)
 |
 +---> F03: Status Transition Unification (depends on F01)
 |
 +---> F04: Document Operations Unification (depends on F01)
 |
 +---> F05: Template Placeholder Unification (depends on F01)
```

F01 is the critical path. F02-F05 can proceed in parallel after F01 is complete, though F02 and F03 are the highest-value features and should be prioritized.

---

## Future Epic Candidates

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| Generic CLI Command Builder (F06) | Medium | Depends on E21 F01-F03 |
| Generic HTTP API Handlers | Low | Depends on E21 F01-F03 + HTTP API stability |
| Repository Generics Migration | Low | Depends on Go generics maturity |
| New Entity Type: Sprint | Medium | Depends on E21 (entity extension infrastructure) |
| New Entity Type: Milestone | Low | Depends on E21 (entity extension infrastructure) |

---

*See also*: [Requirements](./requirements.md)
