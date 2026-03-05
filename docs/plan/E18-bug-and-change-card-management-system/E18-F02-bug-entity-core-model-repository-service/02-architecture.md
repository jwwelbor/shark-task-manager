# E18-F02: Bug Entity Core -- Architecture

**Feature**: Bug Entity Core (Model, Repository, Service)
**Complexity**: STANDARD
**Date**: 2026-03-03

---

## 1. Architecture Overview

This feature adds a new **Bug** entity to the Shark domain model, following the same three-layer architecture used by all existing entities (Epic, Feature, Task, Idea):

```
CLI Command (F04 scope) --> BugService --> BugRepository --> SQLite (bugs table from F01)
                                |
                                +--> workflow.Service.ForLevel("bug")
                                +--> fileops.WriteEntityFile (markdown)
                                +--> Link validation via Epic/Feature/Task repos
```

The Bug entity is a **standalone entity** (like Idea) -- it is not part of the Epic > Feature > Task hierarchy. It has its own key format (`B###`), its own workflow definition, and its own file storage location (`docs/bugs/`).

---

## 2. Key Architecture Decisions

### ADR-1: Bug as Standalone Entity (Not Hierarchical)

**Decision**: Bugs are independent entities with optional links to epics, features, or tasks via `linked_entity_key`/`linked_entity_type` fields. They are NOT children of any entity.

**Rationale**: Bugs can affect any part of the system and are not scoped to a single feature. A bug may relate to multiple features or to infrastructure that spans epics. A link is an optional reference, not a parent-child relationship.

**Consequence**: No `epic_id` or `feature_id` foreign keys. Link validation is done at the service layer, not via database constraints. This keeps the schema simple and avoids cascading deletes.

### ADR-2: Bug Key Format B### (3 Digits)

**Decision**: Bug keys follow the format `B###` (e.g., B001, B042, B999). Keys are never reused after deletion.

**Rationale**: Simple, monotonically increasing keys. The `B` prefix makes bugs visually distinct from other entities (`E`, `F`, `T`, `I` prefixes). Three digits support up to 999 bugs, which is sufficient for the initial release.

**Consequence**: Key generation requires a `GetNextKey` repository method that queries `MAX(CAST(SUBSTR(key, 2) AS INTEGER))` from the bugs table.

### ADR-3: Workflow Integration via ForLevel("bug")

**Decision**: Bug status transitions use the existing `workflow.Service.ForLevel("bug")` mechanism. F01 adds `LevelBug = "bug"` to `internal/workflow/levels.go` and a default bug workflow to `.sharkconfig.json`.

**Rationale**: Reuses the proven multi-level workflow system (E16) instead of creating bug-specific status management. Workflow profiles already support configuring status flows per entity level.

**Consequence**: The `workflow` package needs `LevelBug` constant (F01 scope). The BugService calls `workflowSvc.ForLevel(workflow.LevelBug)` in its constructor. Status transitions, validation, and next-status queries all delegate to the workflow service.

### ADR-4: Link Validation at Service Layer

**Decision**: When `LinkedEntityKey` is provided during bug creation, the BugService detects the entity type from the key format and calls the appropriate repository (`EpicRepository`, `FeatureRepository`, or `TaskRepository`) to verify existence.

**Rationale**: This follows the existing key auto-detection pattern used throughout the CLI. Database foreign keys would require complex polymorphic relationships and would prevent linking to entities that use different key patterns.

**Consequence**: BugService constructor needs access to Epic, Feature, and Task repositories for link validation. These are injected as interfaces (`LinkValidatorEpicRepo`, `LinkValidatorFeatureRepo`, `LinkValidatorTaskRepo`) with minimal method requirements (just `GetByKey`).

### ADR-5: Atomic Create with File Rollback

**Decision**: `BugService.CreateBug` wraps database insert and file creation in a transaction. If file creation fails, the database insert is rolled back.

**Rationale**: Prevents orphaned database records without corresponding markdown files. This matches the existing pattern used by task creation in `taskcreation.Creator`.

**Consequence**: BugService needs `*repository.DB` injected for transaction management (same pattern as TaskService).

### ADR-6: assigned_to as Context Field, Not Column

**Decision**: The `assigned_to` field set during triage is stored in the `context_data` JSON column, not as a dedicated database column.

**Rationale**: `assigned_to` is only set during triage and is optional metadata. Adding a dedicated column for a field used in one operation adds unnecessary schema complexity. The context_data JSON column already exists (F01 schema) and supports arbitrary key-value metadata.

**Consequence**: Triage sets `assigned_to` by reading context_data JSON, adding/updating the key, and writing it back. This is a standard JSON merge operation.

---

## 3. Component Dependency Graph

```
BugService
|-- BugRepository (interface, defined in services package)
|   +-- *repository.BugRepository (concrete implementation)
|       +-- *repository.DB
|
|-- *workflow.Service (ForLevel("bug"))
|   +-- .sharkconfig.json (bug workflow config from F01)
|
|-- *repository.DB (for transaction management in CreateBug)
|
|-- LinkValidatorEpicRepo (interface: GetByKey only)
|   +-- *repository.EpicRepository
|       +-- *repository.DB
|
|-- LinkValidatorFeatureRepo (interface: GetByKey only)
|   +-- *repository.FeatureRepository
|       +-- *repository.DB
|
+-- LinkValidatorTaskRepo (interface: GetByKey only)
    +-- *repository.TaskRepository
        +-- *repository.DB
```

---

## 4. Integration Points

### 4.1 Workflow Package (F01 Prerequisite)

F01 must deliver:
- `workflow.LevelBug = "bug"` constant in `internal/workflow/levels.go`
- Default bug workflow in `.sharkconfig.json` with statuses: `reported`, `triaged`, `in_fix`, `in_verification`, `resolved`, `wont_fix`, `duplicate`
- `ForLevel("bug")` returns the bug-specific workflow config

F02 consumes:
- `workflowSvc.ForLevel(workflow.LevelBug)` in BugService constructor
- `workflowSvc.GetDefaultStatus()` for initial bug status
- `workflowSvc.GetNextStatus(current)` for AdvanceBugStatus
- `workflowSvc.ValidateTransition(from, to)` for SetBugStatus
- `workflowSvc.IsTerminalStatus(status)` for terminal status checks

### 4.2 Models Package (F02 Delivers)

New files:
- `internal/models/bug.go`: Bug struct, BugStatus type, BugSeverity type and constants, Validate(), ValidateBugKey()

Modifications:
- `internal/models/entity_note.go`: Add `EntityTypeBug EntityType = "bug"` and update `ValidEntityTypes` map

### 4.3 Context and Note Services (F02 Modifies)

- `internal/services/context_service.go`: Add `case models.EntityTypeBug:` to `getContextJSON` and `setContextJSON` switch statements. Requires a `BugContextRepo` interface with `GetByKey` and `UpdateContextData` methods.
- `internal/services/note_service.go`: Add `case models.EntityTypeBug:` for entity existence validation. Requires a `BugEntityValidator` interface with `GetByKey` method.

### 4.4 File Operations (Existing, No Changes)

- `fileops.WriteEntityFile` with `EntityType: "bug"` and `UseAtomicWrite: true`
- File path: `docs/bugs/B###.md` (relative to project root)
- Directory creation: automatic by fileops

### 4.5 CLI Services Global (F02 Modifies)

- `internal/cli/services_global.go`: Add `GetBugService()` accessor function following the existing pattern (see `GetTaskService()`, `GetNoteService()`).

### 4.6 Templates Package (F02 Delivers)

- New template directory: `templates/bug/`
- Bug markdown template with frontmatter and structured sections

---

## 5. File Inventory

### New Files

| File | Layer | Purpose |
|------|-------|---------|
| `internal/models/bug.go` | Model | Bug struct, BugStatus, BugSeverity, Validate(), ValidateBugKey() |
| `internal/repository/bug_repository.go` | Repository | BugRepository concrete implementation (CRUD + filtering + key gen) |
| `internal/services/bug_service.go` | Service | BugService with all business logic |
| `internal/services/bug_dto.go` | Service | CreateBugInput, BugFilters, TriageBugInput, BugUpdates DTOs |
| `templates/bug/bug.md.tmpl` | Template | Bug markdown template |

### Modified Files

| File | Change |
|------|--------|
| `internal/models/entity_note.go` | Add `EntityTypeBug` constant and update `ValidEntityTypes` |
| `internal/workflow/levels.go` | **F01 scope**: Add `LevelBug = "bug"` |
| `internal/services/context_service.go` | Add bug case to entity type switch (+ BugContextRepo interface) |
| `internal/services/note_service.go` | Add bug case to entity validation (+ BugEntityValidator interface) |
| `internal/cli/services_global.go` | Add `GetBugService()` accessor |

### Test Files

| File | Purpose |
|------|---------|
| `internal/models/bug_test.go` | Model validation, key validation |
| `internal/repository/bug_repository_test.go` | CRUD + filtering with real DB |
| `internal/services/bug_service_test.go` | Business logic with mocked repos |

---

## 6. Patterns to Follow

### Pattern Source: IdeaService (Standalone Entity)

The Idea entity is the closest architectural analog to Bug:
- Standalone (not hierarchical)
- Simple key format
- Auto-key generation
- Repository interface defined in services package
- Service with constructor injection

### Pattern Source: TaskService (Service Layer Architecture)

The Task entity provides patterns for:
- Workflow-integrated status transitions via `workflowSvc.ForLevel()`
- Transaction management for atomic operations
- File creation during entity creation
- Global service accessor wiring in `services_global.go`

### Pattern Source: ContextService (Entity Type Registration)

Context and Note services use entity type switch statements. Bug needs to be added as a new case following the existing pattern.

---

## 7. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| F01 not delivering LevelBug workflow | BugService cannot validate transitions | F01 is execution_order=1 dependency. Verify LevelBug constant and workflow config exist before starting F02. |
| Concurrent key generation race condition | Two bugs could get same B### key | GetNextKey uses `MAX()` SQL query. If concurrent access is needed, use INSERT with UNIQUE constraint and retry on conflict. For local SQLite with single-writer WAL mode, this is not a practical concern. |
| Link validation entity type detection fails | Invalid linked entity accepted or valid one rejected | Reuse existing key pattern regex from `models/validation.go`. Test with all entity key formats including slugged keys. |
| Template rendering failure during CreateBug | Bug created in DB without markdown file | Wrap in transaction; rollback DB insert if file creation fails. |

---

*Last Updated: 2026-03-03*
