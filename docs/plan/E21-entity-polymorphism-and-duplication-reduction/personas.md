# User Personas

**Epic**: [Entity Polymorphism and Duplication Reduction](./epic.md)

---

## Overview

This is an internal architecture refactoring epic. The "users" are developers and contributors who maintain, extend, and consume the Shark codebase. There are no end-user-facing behavioral changes. The personas below describe the developer roles whose daily work is directly affected by the entity system's architecture.

---

## Primary Personas

### Persona 1: Codebase Maintainer

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Core contributor responsible for fixing bugs, updating cross-cutting features, and keeping the codebase healthy
- **Experience Level**: Deep familiarity with Shark's architecture; works across models, services, repositories, and CLI layers daily
- **Key Characteristics**:
  - Fixes bugs that span multiple entity types (e.g., a status transition edge case that affects all 5 entities)
  - Implements cross-cutting features (notes, context, document linking) that must work identically for all entities
  - Reviews pull requests and enforces architectural consistency
  - Spends significant time on repetitive changes when a fix must be applied to 5 entity services independently

**Goals Related to This Epic**:
1. Fix a cross-cutting bug in one place and have it apply to all entity types automatically
2. Implement a new cross-cutting feature (e.g., entity archiving) by writing it once against the Entity interface
3. Maintain confidence that all entity types behave consistently for shared operations

**Pain Points This Epic Addresses**:
- A status transition bug fix requires editing 5 service files with nearly identical changes, risking copy-paste errors
- Adding document linking to a new entity type requires duplicating ~30 lines of boilerplate from an existing entity service
- NoteService and ContextService contain 5-branch switch statements that must be extended every time an entity type is added
- Behavioral inconsistencies between entities (e.g., EpicService.GetEpic returns nil-check error differently than BugService.GetBug) because each implementation drifted independently

**Success Looks Like**:
A maintainer discovers a bug in status transition logic, fixes it in `EntityService.TransitionStatus`, runs `make test`, and all 5 entity types inherit the fix. The entire change touches 1 file instead of 5.

---

### Persona 2: New Entity Author

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Developer tasked with adding a new entity type to Shark (e.g., a "Milestone" or "Sprint" entity)
- **Experience Level**: Familiar with Go; may not know Shark internals deeply. Uses existing entities as reference for how to build a new one.
- **Key Characteristics**:
  - Studies existing entity implementations to understand the pattern
  - Needs clear guidance on what is required vs. what is inherited
  - Wants to avoid touching 15+ files just to get a basic entity working
  - Cares about getting CRUD, status transitions, notes, context, and CLI commands working with minimal effort

**Goals Related to This Epic**:
1. Add a new entity type by implementing the Entity interface, creating an EntityRepository adapter, and registering it -- without duplicating cross-cutting service logic
2. Get notes, context, document linking, and status transitions for free from the shared EntityService
3. Focus implementation time on entity-specific business logic (e.g., sprint capacity, milestone deadlines) rather than plumbing

**Pain Points This Epic Addresses**:
- Today, adding a new entity requires copying and modifying code in 15+ files across models, repositories, services, CLI commands, and CLI accessors
- Each cross-cutting service (NoteService, ContextService, ResumeService) needs a new switch branch, new setter method, and new accessor function
- No documentation or interface contract defines what "implementing an entity" means -- the pattern must be reverse-engineered from existing code
- Estimated effort for a new entity: ~2 weeks. Target after this epic: ~2 days.

**Success Looks Like**:
A developer adds a new "Milestone" entity by: (1) defining the model struct with Entity interface methods, (2) creating a repository that also implements EntityRepository, (3) registering it in the EntityRegistry, and (4) writing only milestone-specific service logic. Cross-cutting features (notes, context, status transitions, document linking) work automatically. Total files modified: 3-5. Total time: 1-2 days.

---

### Persona 3: Service Layer Consumer (CLI/API Developer)

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Developer who writes CLI commands or HTTP API handlers that consume entity services
- **Experience Level**: Understands the three-layer architecture (command -> service -> repository); works primarily in `internal/cli/commands/` or `cmd/server/`
- **Key Characteristics**:
  - Writes thin command wrappers that call service methods and format output
  - Needs predictable, consistent service method signatures across entity types
  - Works with the service accessor pattern (`cli.GetTaskService()`, `cli.GetEpicService()`)
  - Benefits from generic commands that work across entity types (e.g., a single `status advance` command that routes to the correct entity service)

**Goals Related to This Epic**:
1. Write generic CLI commands that operate on any entity type without per-entity switch statements
2. Use consistent service method signatures so that command logic is predictable across entity types
3. Access entity services through the registry pattern instead of per-entity accessor functions

**Pain Points This Epic Addresses**:
- The `status advance` command contains a switch statement that dispatches to 5 different service methods with slightly different signatures
- Adding a new entity type to unified commands requires editing every dispatch point
- CLI accessor files (`services_global.go`, `services_global_ext.go`) grow linearly with entity count
- Service method signatures are inconsistent across entity types (e.g., some return `*TransitionResult`, others return the entity directly)

**Success Looks Like**:
The `status advance` command calls `registry.GetService(entityType).TransitionStatus(ctx, key, target, opts)` -- one line that works for all entity types. Adding a new entity type requires zero changes to existing commands.

---

## Secondary Personas

- **Test Author**: Writes tests for cross-cutting services. Benefits from testing shared logic once instead of verifying the same behavior 5 times across 5 entity test suites.
- **Code Reviewer**: Reviews PRs that add entity features. Benefits from a clear interface contract that makes it easy to verify completeness and consistency.

---

## Persona Validation Notes

- These personas are based on the actual development experience documented in the epic's Current State Analysis, which quantifies the duplication (1,255+ lines, 15+ files per entity addition).
- Confidence is high for Persona 1 (Maintainer) and Persona 3 (Consumer) as they represent current active development roles in the Shark project.
- Persona 2 (New Entity Author) is validated by the recent E18 experience: adding Bug and ChangeCard entities required touching 15+ files and took approximately 2 weeks of development effort.
- All personas are internal (developers), not external users. This epic has no end-user-facing behavioral changes.

---

*See also*: [User Journeys](./user-journeys.md)
