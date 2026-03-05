# E18-F03: Change-Card Entity Core -- Architecture

**Feature**: E18-F03 Change-Card Entity Core (Model, Repository, Service)
**Date**: 2026-03-03
**Status**: Proposed
**Complexity**: STANDARD (S-M)

---

## 1. Architecture Overview

This feature adds a ChangeCard entity to Shark following the exact same three-layer pattern used by existing entities (Epic, Feature, Task, Idea) and the parallel Bug entity (E18-F02):

```
CLI Command (F05)  -->  ChangeCardService  -->  ChangeCardRepository  -->  change_cards table (F01)
                             |
                        workflow.Service.ForLevel("change")
```

The ChangeCard is a **standalone entity** -- unlike tasks which are nested under features and epics, change-cards exist at the project level with optional links to epics or features.

### Key Decisions

| # | Decision | Rationale | Alternatives Considered |
|---|----------|-----------|------------------------|
| 1 | Flat `docs/changes/C###.md` file path | Change-cards are standalone, not nested in epic/feature hierarchy | Nested under linked entity (rejected: link is optional, file would need to move on re-link) |
| 2 | `C###` key format (3-digit zero-padded) | Consistent with B### for bugs; simple auto-increment | UUID (rejected: not human-friendly), date-based like ideas (rejected: change-cards don't cluster by date) |
| 3 | `workflow.LevelChange` for workflow scoping | Keeps change-card workflow independent of task/epic/feature workflows | Shared workflow level (rejected: change-cards have different statuses) |
| 4 | Link validation at service layer only | Keeps repository as pure CRUD; follows E15 architecture | Validation in repository (rejected: business logic in wrong layer) |
| 5 | No severity/priority fields | Change-cards are lightweight; if granularity needed, promote to feature | Add priority (rejected: PRD explicitly excludes it) |
| 6 | Entity type `"change"` for notes | Extends existing EntityNoteRepository without code changes | New note table (rejected: unnecessary duplication) |
| 7 | Pattern parity with Bug entity (F02) | Consistent architecture across both new entity types in E18 | Different patterns (rejected: increases cognitive load) |

### Dependencies

- **E18-F01**: Provides `change_cards` table schema and `"change"` workflow level in the workflow engine
- **E15 Service Layer**: Provides established patterns for service/repository/model architecture
- **Existing repositories**: `EpicRepository.GetByKey()` and `FeatureRepository.GetByKey()` for link validation

---

## 2. Component Diagram

```
internal/
  models/
    change_card.go            # ChangeCard struct, ChangeCardStatus type, Validate()
    entity_note.go            # ADD: EntityTypeChange = "change" constant + ValidEntityTypes entry

  repository/
    change_card_repository.go # ChangeCardRepository concrete implementation (pure CRUD)

  services/
    change_card_service.go    # ChangeCardService (all business logic)
    change_dto.go             # CreateChangeCardInput, ChangeCardFilters, ChangeCardUpdates DTOs

  workflow/
    levels.go                 # ADD: LevelChange = "change" constant

  cli/
    services_global.go        # ADD: GetChangeCardService() accessor function

templates/
  change_card.md.tmpl         # Markdown template for change-card files (or inline in service)
```

---

## 3. Workflow Integration

The change-card workflow is defined in F01 and consumed here via `workflowSvc.ForLevel("change")`:

```
proposed  -->  approved  -->  in_progress  -->  completed
    |
    +--->  declined (terminal)
```

- **Default initial status**: Read from `workflowSvc.GetDefaultStatus()` (expected: `proposed`)
- **Terminal statuses**: `completed`, `declined`
- **Convenience method**: `ApproveChangeCard()` wraps the `proposed -> approved` transition
- **All transitions validated** via `workflowSvc.ValidateTransition()` -- no hardcoded status checks

---

## 4. Cross-Cutting Concerns

### Entity Notes
The existing `EntityNoteRepository` already accepts arbitrary `entity_type` strings in SQL. The only code change needed is adding `EntityTypeChange = "change"` to `models.entity_note.go` and updating the `ValidEntityTypes` map. No repository or service changes are needed for notes to work with change-cards.

### Atomic File+DB Operations
`CreateChangeCard` must ensure atomicity: if the markdown file write fails after DB insert, the DB record must be rolled back. Pattern: begin transaction, insert DB record, write file via `fileops.NewEntityFileWriter()`, commit transaction on success or rollback on file failure.

### Key Auto-Generation Concurrency
Key generation uses `SELECT MAX(CAST(SUBSTR(key, 2) AS INTEGER)) FROM change_cards` within a transaction. The `UNIQUE` constraint on `key` column (from F01 schema) provides a safety net against race conditions. If a UNIQUE violation occurs, the service should retry with the next key or return a clear error.

### Dual Key Lookup
`GetByKey` supports both `C001` (numeric) and `C001-add-dark-mode` (slugged) formats. The lookup strategy mirrors tasks: try exact match on `key` first, then parse slug suffix and match `key + slug` pair.

---

## 5. Security Considerations

- **Input Validation**: Title is sanitized (trimmed) at model layer; status validated via workflow service at service layer
- **SQL Injection**: All repository queries use parameterized statements (existing pattern)
- **Link Validation**: Service validates linked entity existence before creating the change-card, preventing dangling references
- **No authentication changes**: Change-cards follow the same access model as other entities

---

## 6. Testing Strategy

| Layer | Test Type | Database | Description |
|-------|-----------|----------|-------------|
| Model | Unit | None | `Validate()` structural checks: empty title, empty status |
| Repository | Integration | Real SQLite | CRUD, key auto-gen, dual-key lookup, linked entity filter |
| Service | Unit (mocked) | None | Create (happy + invalid link), Approve (happy + wrong status), Advance, SetStatus, List filters, Delete |

Service tests use the function-field mock pattern matching existing test conventions.

---

## 7. Out of Scope

- CLI commands (F05)
- Unified CLI key auto-detection (F06)
- Dashboard/analytics integration (F07)
- Change-card promotion to feature (deferred)
- Task-level linking (per epic REQ-F-008)
