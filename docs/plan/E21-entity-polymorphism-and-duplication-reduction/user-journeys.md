# User Journeys

**Epic**: [Entity Polymorphism and Duplication Reduction](./epic.md)

---

## Overview

This document maps the developer workflows that are directly affected by the entity polymorphism refactoring. Since this is an internal architecture epic, all journeys describe developer experiences rather than end-user interactions.

---

## Journey 1: Adding a New Entity Type

**Persona**: New Entity Author

**Goal**: Add a fully functional entity type to Shark with CRUD, status transitions, notes, context, and document linking

**Preconditions**:
- Entity polymorphism (E21 F01-F05) is complete
- EntityRegistry is initialized with existing entity types
- Shared EntityService handles cross-cutting operations

### Current State (Before E21)

1. **Developer defines model struct** in `internal/models/`
   - Create new file with struct, status type alias, and Validate() method
   - Copy shared fields from an existing entity (10 fields)
   - Files touched: 1

2. **Developer creates repository** in `internal/repository/`
   - Implement GetByKey, GetByID, Create, Update, Delete, UpdateStatus, GetContextData, UpdateContextData
   - Copy SQL patterns from an existing repository
   - Files touched: 1

3. **Developer creates service** in `internal/services/`
   - Copy GetEntity, CreateEntity, TransitionStatus, LinkDocument, UnlinkDocument, resolveAction from an existing service
   - Adapt type names and error messages
   - Files touched: 1

4. **Developer updates NoteService**
   - Add new switch branch to resolveEntityID (copy existing branch, change type)
   - Add new switch branch to GetEntityDetails
   - Add SetNewEntityRepo() setter method
   - Files touched: 1

5. **Developer updates ContextService**
   - Add new switch branches to getContextJSON and setContextJSON
   - Add SetNewEntityRepo() setter method
   - Files touched: 1

6. **Developer updates ResumeService**
   - Add new switch branches and setter methods
   - Files touched: 1

7. **Developer creates CLI commands**
   - Create command file with CRUD, note, context subcommands
   - Files touched: 1

8. **Developer updates CLI accessors**
   - Add GetNewEntityService() to services_global.go or services_global_ext.go
   - Wire repository and service dependencies
   - Files touched: 1-2

9. **Developer updates unified commands**
   - Add key detection for new entity prefix to core commands (get, status, search, list)
   - Add dispatch branches to status advance, status set, etc.
   - Files touched: 4-6

10. **Developer updates database schema**
    - Add table, indexes, migration
    - Files touched: 1

11. **Developer adds templates**
    - Create entity template file
    - Register in template system
    - Files touched: 1-2

**Total files touched: 15+**
**Estimated effort: ~2 weeks**
**Lines of duplicated code: ~400+ (service layer alone)**

### Target State (After E21)

1. **Developer defines model struct** implementing the `Entity` interface
   - Create new file with struct and Entity interface methods (GetID, GetKey, GetTitle, etc.)
   - Entity-specific fields only -- shared behavior inherited via interface
   - Files touched: 1

2. **Developer creates repository** implementing both typed and `EntityRepository` interfaces
   - Implement entity-specific queries plus the generic EntityRepository adapter
   - Files touched: 1

3. **Developer registers entity in EntityRegistry**
   - Call `registry.Register(models.EntityTypeNewEntity, repo, service)` at startup
   - Cross-cutting services (notes, context, resume, document linking, status transitions) work automatically
   - Files touched: 1

4. **Developer writes entity-specific service logic** (if any)
   - Only unique business logic (e.g., milestone deadlines, sprint capacity)
   - Compose with EntityService for shared operations
   - Files touched: 0-1

5. **Developer updates database schema**
   - Add table, indexes, migration
   - Files touched: 1

**Total files touched: 3-5**
**Estimated effort: 1-2 days**
**Lines of duplicated code: 0 (shared logic inherited)**

**Success Outcome**: A new entity type achieves full feature parity with existing entities for all cross-cutting operations by implementing the Entity interface and registering with the EntityRegistry. No switch statements, no setter methods, no copy-paste of service logic.

---

## Journey 2: Fixing a Cross-Cutting Bug

**Persona**: Codebase Maintainer

**Goal**: Fix a bug that affects status transitions for all entity types

**Preconditions**:
- Bug is identified in status transition logic (e.g., backward transition does not require a reason when it should)
- Bug affects all 5 entity types

### Current State (Before E21)

1. **Developer identifies the bug** in one entity service (e.g., TaskService.TransitionStatus)
2. **Developer fixes the bug** in TaskService.TransitionStatus (~80 lines of status transition logic)
3. **Developer must check 4 other entity services** to see if they have the same bug
   - EpicService.TransitionStatus
   - FeatureService.TransitionStatus
   - BugService.TransitionStatus
   - ChangeCardService.TransitionStatus
4. **Developer applies the same fix** to each service independently
   - Each implementation has diverged slightly, so the fix may not be identical
   - Risk of missing one entity or introducing a regression in an entity that was working
5. **Developer writes/updates tests** in 5 separate test files

**Files touched: 5 service files + 5 test files = 10 files**
**Risk: High (inconsistent fixes, missed entities, copy-paste errors)**

### Target State (After E21)

1. **Developer identifies the bug** in EntityService.TransitionStatus
2. **Developer fixes the bug** in the single shared implementation
3. **Developer writes/updates test** in EntityService test file
4. **Developer runs `make test`** to confirm all entity types pass

**Files touched: 1 service file + 1 test file = 2 files**
**Risk: Low (single implementation, all entities inherit the fix)**

**Success Outcome**: A cross-cutting bug fix requires editing 1-2 files instead of 10. All entity types are guaranteed to receive the fix because they share the same code path.

---

## Journey 3: Adding a Cross-Cutting Feature

**Persona**: Codebase Maintainer

**Goal**: Add entity archiving -- the ability to archive any entity type

**Preconditions**:
- Archiving is a new cross-cutting feature that sets a status and records a timestamp
- Must work for all entity types

### Current State (Before E21)

1. **Developer adds archive logic to EpicService**
   - Implement ArchiveEpic method (~30 lines)
   - Add archived status to epic workflow
2. **Developer duplicates to FeatureService** (~30 lines, adapted)
3. **Developer duplicates to TaskService** (~30 lines, adapted)
4. **Developer duplicates to BugService** (~30 lines, adapted)
5. **Developer duplicates to ChangeCardService** (~30 lines, adapted)
6. **Developer updates CLI commands** for each entity type (5 command files)
7. **Developer writes tests** for each entity service (5 test files)

**Files touched: 5 services + 5 commands + 5 tests = 15 files**
**New lines of code: ~150 (30 x 5) in services alone**

### Target State (After E21)

1. **Developer adds archive logic to EntityService**
   - Implement `ArchiveEntity(ctx, repo EntityRepository, key string) error` (~30 lines, once)
2. **Developer adds archived status to workflow configuration**
3. **Developer adds single CLI command** that uses registry to dispatch to any entity type
4. **Developer writes test** for EntityService.ArchiveEntity (once)

**Files touched: 1 service + 1 command + 1 test = 3 files**
**New lines of code: ~30 in service**

**Success Outcome**: A new cross-cutting feature requires 1/5th the code and 1/5th the files. Behavioral consistency across entity types is guaranteed by shared implementation.

---

## Journey 4: Onboarding a New Contributor

**Persona**: New Entity Author (variant: first-time contributor)

**Goal**: Understand the entity system well enough to make a contribution

### Current State (Before E21)

1. **Contributor reads CLAUDE.md** and architecture docs
2. **Contributor looks at one entity** (e.g., Task) as reference
3. **Contributor looks at another entity** (e.g., Bug) and notices differences
   - Different error message formatting
   - Slightly different method signatures
   - Different nil-check patterns
4. **Contributor is confused** about which pattern is "correct"
5. **Contributor asks maintainer** or guesses based on the most recent entity (Bug/ChangeCard)

**Result: Slow onboarding, risk of propagating inconsistencies**

### Target State (After E21)

1. **Contributor reads CLAUDE.md** and architecture docs
2. **Contributor sees the Entity interface** -- a single, clear contract for what every entity must implement
3. **Contributor sees EntityService** -- a single reference implementation for cross-cutting operations
4. **Contributor understands** that entity-specific services only contain unique logic

**Result: Clear contract, consistent patterns, fast onboarding**

**Success Outcome**: A new contributor can understand the entity system by reading 2 files (entity.go interface, entity_service.go shared logic) instead of comparing 5 independent implementations.

---

## Journey Impact Summary

| Journey | Files Touched (Before) | Files Touched (After) | Improvement |
|---------|----------------------|---------------------|-------------|
| Add new entity type | 15+ | 3-5 | 3-5x fewer files |
| Fix cross-cutting bug | 10 | 2 | 5x fewer files |
| Add cross-cutting feature | 15 | 3 | 5x fewer files |
| Onboard new contributor | Compare 5 implementations | Read 2 files | Clear contract |

---

*See also*: [Requirements](./requirements.md)
