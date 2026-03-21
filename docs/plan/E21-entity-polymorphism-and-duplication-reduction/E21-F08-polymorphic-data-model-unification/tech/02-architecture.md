# Technical Architecture: E21-F08 Polymorphic Data Model Unification

**Feature**: E21-F08
**Complexity Tier**: COMPLEX
**Author**: Architect
**Date**: 2026-03-20
**Status**: Draft

---

## 1. Overview

### Problem

The database has inconsistent polymorphism across cross-cutting concerns:

| Concern | Current State | Tables | Available To |
|---------|--------------|--------|--------------|
| Notes | `entity_notes` (polymorphic) | 1 | All 5 entity types |
| Documents | 5 separate `{entity}_documents` join tables | 5 | All 5 types, but duplicated |
| History | `task_history` (task-only) | 1 | Only Task |

This means:
- Adding document support for a new entity requires a new table, new repository methods, and new service wiring
- Only Tasks have status change audit trails -- Epics, Features, Bugs, and ChangeCards have none
- `document_repository.go` has 15 per-entity methods (449 lines) that differ only by table name and FK column

### Solution

1. Create polymorphic `entity_documents` table following the `entity_notes` pattern
2. Create polymorphic `entity_history` table for all entity types
3. Migrate data from 5 document tables and `task_history` into the new tables
4. Replace 15 per-entity repository methods with 3 polymorphic methods
5. Wire history recording into `EntityService.TransitionStatus`
6. Drop old tables after migration verification

### Design Principles Applied

- **Appropriate**: Follows the proven `entity_notes` pattern already in the codebase (645 lines, battle-tested)
- **Proven**: Polymorphic entity_type + entity_id pattern is established in this project and in Go ecosystem (Terraform, K8s)
- **Simple**: Two new tables replace six. Three repository methods replace fifteen. Zero new architectural concepts.

---

## 2. Architecture

### 2.1 Component Diagram

```
BEFORE:
  epic_documents ──┐
  feature_documents┤
  task_documents  ──┤──> documents table
  bug_documents   ──┤
  change_card_documents┘
  task_history (standalone)

  DocumentRepository (449 lines, 15 entity-specific methods)
  TaskHistoryRepository (360 lines, task-only)

AFTER:
  entity_documents ──> documents table
  entity_history (standalone)

  EntityDocumentRepository (~120 lines, 3 polymorphic methods)
  EntityHistoryRepository (~180 lines, polymorphic)
```

### 2.2 File Impact Map

#### New Files (4)

| File | Purpose | Size Est. |
|------|---------|-----------|
| `internal/models/entity_history.go` | EntityHistory model struct + validation | ~50 lines |
| `internal/repository/entity_document_repository.go` | Polymorphic document linking (Link, Unlink, ListForEntity) | ~120 lines |
| `internal/repository/entity_history_repository.go` | Polymorphic history CRUD + filters | ~180 lines |
| `internal/services/entity_history_service.go` | History query service for all entity types | ~150 lines |

#### Modified Files (10-12)

| File | Changes |
|------|---------|
| `internal/db/db.go` | Add `entity_documents` and `entity_history` table DDL; bump `CurrentSchemaVersion` 6 -> 7 |
| `internal/db/migrate.go` | Add migration function: create new tables, migrate data, drop old tables |
| `internal/repository/document_repository.go` | Remove 15 entity-specific Link/Unlink/List methods (~300 lines removed); keep core CRUD |
| `internal/repository/task_history_repository.go` | Refactor to delegate to EntityHistoryRepository with `entity_type='task'` filter, or deprecate |
| `internal/services/entity_document_service.go` | Simplify: replace callback pattern with direct EntityDocumentRepository calls |
| `internal/services/entity_service.go` | Add optional `EntityHistoryRepository` dependency; add history recording step in `TransitionStatus` |
| `internal/services/task_history_service.go` | Generalize to support all entity types or wrap EntityHistoryRepository |
| `internal/cli/commands/related_docs.go` | Add `--bug` and `--change` flags (if not already present) |
| `internal/cli/commands/status_group.go` | Update `status history` to query EntityHistoryRepository for any entity type |
| `internal/cli/services_global.go` | Wire new repositories into service constructors |

### 2.3 Data Flow

#### Document Linking (After)

```
CLI: shark related-docs add --bug=B001 --path=docs/analysis.md
  |
  v
EntityDocumentService.LinkDocument(ctx, entityType="bug", entityKey="B001", docPath="docs/analysis.md")
  |
  +-- EntityDocumentRepository.CreateOrGet(ctx, title, filePath) --> documents table
  +-- EntityDocumentRepository.Link(ctx, entityType="bug", entityID=42, documentID=7) --> entity_documents table
```

#### History Recording (After)

```
EntityService.TransitionStatus(ctx, repo, "feature", "E21-F07", "active", opts, features)
  |
  +-- Step 7: repo.UpdateStatus(ctx, id, "active")
  +-- Step 7.5 (NEW): historyRepo.Create(ctx, &EntityHistory{
  |       EntityType: "feature",
  |       EntityID:   id,
  |       FromStatus: "draft",
  |       ToStatus:   "active",
  |       ChangedBy:  opts.Agent,
  |       Notes:      opts.Reason,
  |   })
  +-- Step 8: (optional) rejection note creation
```

---

## 3. Key Technical Decisions

### ADR-1: Include `link_type` in `entity_documents`

**Decision**: The `entity_documents` table includes a `link_type TEXT DEFAULT 'general'` column.

**Rationale**: The existing `task_documents` table has a `link_type` column (added in E07-F22) not present in other entity document tables. Including it in the unified table:
- Preserves existing task document link types during migration
- Provides forward compatibility for typed document associations on any entity
- Costs zero overhead when unused (NULL or 'general' default)

**Alternative rejected**: Omit `link_type` and lose existing task document type data.

### ADR-2: New `EntityHistory` model (not extending `TaskHistory`)

**Decision**: Create a new `models.EntityHistory` struct with `EntityType` + `EntityID` fields. The existing `TaskHistory` model is not modified.

**Rationale**:
- `TaskHistory` has task-specific fields (`TaskID int64`) and task-specific validation (`ValidateTaskStatus`)
- `EntityHistory` uses `EntityType` + `EntityID` for polymorphic storage
- Clean separation avoids confusion and breaking changes
- `TaskHistoryService` can internally delegate to `EntityHistoryRepository` with `entity_type='task'` filter for backward compatibility

**Alternative rejected**: Add `EntityType` and `EntityID` fields to `TaskHistory` -- would break existing callers and conflate two concepts.

### ADR-3: History recording in `EntityService.TransitionStatus` (centralized)

**Decision**: All status change history is recorded in a single point: `EntityService.TransitionStatus`, after the status update succeeds.

**Rationale**:
- `EntityService.TransitionStatus` is already the unified entry point for all entity status transitions (Epic, Feature, Task, Bug, ChangeCard)
- Adding history here means zero per-entity code needed for history recording
- History records all transitions: forward, backward, and forced
- The `historyRepo` dependency is optional (nil-safe) for backward compatibility during rollout

**Alternative rejected**: Per-entity history recording in each service's `TransitionStatus` wrapper -- would add 5 identical recording blocks.

### ADR-4: Old tables dropped in same migration after row count verification

**Decision**: Old per-entity document tables and `task_history` are dropped within the migration function, after programmatic verification that all rows were copied.

**Rationale**:
- The migration runs in a single transaction (or with verification steps)
- Row count verification ensures zero data loss
- Keeping old tables creates confusion about which table is canonical
- The `skip_migrations` flag in `.sharkconfig.json` provides a safety mechanism -- developer must explicitly enable migrations

**Risk mitigation**: The migration logs row counts before and after. If counts do not match, the migration aborts without dropping old tables.

---

## 4. Dependencies

### Hard Dependencies

| Dependency | Status | Impact on F08 |
|-----------|--------|---------------|
| E21-F01 (Entity Interface) | **COMPLETED** | `Entity.GetEntityType()` provides entity_type values for polymorphic columns |

### Soft Dependencies

| Dependency | Status | Impact on F08 |
|-----------|--------|---------------|
| E21-F07 (BaseEntity) | ready_for_task_generation | `BaseEntity.GetEntityType()` is convenient but not required -- F01 already provides this |
| E21-F09 (Service Delegation) | not started | F09 would wire TransitionStatus into entity services. F08 adds the history recording step that F09 then inherits. Not a blocker -- F08 can add history to the existing EntityService.TransitionStatus independently. |

### Features This Unblocks

| Feature | How F08 Helps |
|---------|--------------|
| E21-F11 (Polymorphic Entity Relationships) | Same `entity_type + entity_id` table pattern; F08 proves migration approach |
| E21-F12 (Polymorphic Acceptance Criteria) | Same pattern; F08 establishes precedent |

---

## 5. Implementation Phases

### Phase 1: Schema + Models + Repositories (3-4 tasks)

**Goal**: New tables exist, new repositories are functional, migration copies data.

1. Create `entity_documents` and `entity_history` tables via migration in `internal/db/migrate.go`
2. Create `models.EntityHistory` struct in `internal/models/entity_history.go`
3. Create `EntityDocumentRepository` in `internal/repository/entity_document_repository.go`
4. Create `EntityHistoryRepository` in `internal/repository/entity_history_repository.go`
5. Write migration function: copy data from 5 document tables + `task_history` into new tables
6. Verify row counts, then drop old tables
7. Bump `CurrentSchemaVersion` from 6 to 7

**Quality gate**: `make fmt && make lint && make test` passes. Manual verification: `shark related-docs list --epic=E01` returns same results from new table.

### Phase 2: Service Layer Updates (2-3 tasks)

**Goal**: Services use new polymorphic repositories; history recording is active.

1. Simplify `EntityDocumentService`: replace callback pattern (`linkFn`, `unlinkFn`, `listFn`) with direct `EntityDocumentRepository` calls
2. Add `EntityHistoryRepository` as optional dependency to `EntityService`
3. Add history recording step (7.5) in `EntityService.TransitionStatus`
4. Create `EntityHistoryService` (or generalize `TaskHistoryService`) for query operations

**Quality gate**: `make fmt && make lint && make test` passes. Status transitions for all entity types create history records.

### Phase 3: CLI + Cleanup (2-3 tasks)

**Goal**: CLI commands work with all entity types; old code removed.

1. Update `related-docs` commands to support `--bug` and `--change` flags
2. Update `status history` command to query `entity_history` for any entity type (using key auto-detection)
3. Remove 15 entity-specific methods from `document_repository.go`
4. Update `TaskHistoryService` to delegate to `EntityHistoryRepository` with `entity_type='task'` filter
5. Clean up unused imports and dead code

**Quality gate**: `make fmt && make lint && make test` passes. `shark status history E21-F07` returns feature history. `shark related-docs add --bug=B001 --path=docs/x.md` works.

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Data loss during migration | Low | Critical | Row count verification before dropping; transaction wrapping; `skip_migrations` safety flag |
| `task_documents.link_type` data lost | Low | Medium | `entity_documents` includes `link_type` column; migration copies it |
| Performance regression on polymorphic queries | Very Low | Low | Composite index `(entity_type, entity_id)` ensures O(log n); dataset sizes <100k |
| Breaking existing `task history` CLI commands | Low | Medium | `TaskHistoryService` continues to work via delegation to `EntityHistoryRepository` with filter |
| FK integrity (polymorphic columns lack SQL FK) | Low | Low | Application-level validation in repository; same proven approach as `entity_notes` |
| Conflict with E21-F07 BaseEntity changes | Low | Low | F08 touches repository and service layers; F07 touches model layer. Minimal overlap. |

---

## 7. Architecture Compliance Checklist

- [x] Follows existing polymorphic pattern (`entity_notes`)
- [x] New repositories are pure data access (no business logic)
- [x] History recording is in service layer (`EntityService.TransitionStatus`)
- [x] No direct repository calls from CLI commands
- [x] All new code uses `context.Context` as first parameter
- [x] Error wrapping at each layer with business context
- [x] Optional dependencies degrade gracefully (nil-safe `historyRepo`)
- [x] Migration is idempotent (uses `IF NOT EXISTS`, checks column presence)
- [x] `CurrentSchemaVersion` bumped
- [x] `skip_migrations` flag impact documented

---

*Last Updated*: 2026-03-20
