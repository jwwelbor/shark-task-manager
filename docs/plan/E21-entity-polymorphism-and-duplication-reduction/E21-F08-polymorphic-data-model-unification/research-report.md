# E21-F08 Feature Research Report: Polymorphic Data Model Unification

**Date**: 2026-03-20
**Feature**: E21-F08 - Polymorphic Data Model Unification
**Researcher**: Researcher Agent

---

## Executive Summary

This feature consolidates 5 per-entity document join tables and the task-only history table into 2 polymorphic tables (`entity_documents`, `entity_history`), following the proven `entity_notes` pattern already in the codebase. Codebase analysis confirms the existing polymorphic pattern is mature (645 lines, search, rejection notes, transaction support), the document repository is the primary duplication target (449 lines with 15 per-entity methods), and the `EntityService.TransitionStatus` method is the natural integration point for history recording. The migration is low-risk because it follows an established pattern and the `EntityDocumentService` already abstracts document operations via callbacks.

---

## 1. Codebase Patterns Relevant to This Feature

### 1.1 Proven Polymorphic Pattern: `entity_notes`

The `entity_notes` table is the reference implementation for E21-F08. It uses `entity_type TEXT + entity_id INTEGER` to support all 5 entity types in one table.

**Key characteristics:**
- Column pair: `entity_type` (TEXT NOT NULL) + `entity_id` (INTEGER NOT NULL)
- Validated entity types: `EntityTypeEpic`, `EntityTypeFeature`, `EntityTypeTask`, `EntityTypeChange`, `EntityTypeBug` (defined in `internal/models/entity_note.go`, lines 10-18)
- Composite index: `idx_entity_notes_lookup ON entity_notes(entity_type, entity_id)`
- Repository: `internal/repository/entity_note_repository.go` (645 lines)
- Features: CRUD, search, rejection notes, transaction-aware variants (`CreateRejectionNoteWithTx`)

**What to replicate for `entity_documents` and `entity_history`:**
- Same `entity_type + entity_id` column pattern
- Same `models.EntityType` constants for type discrimination
- Same composite index pattern
- Same repository structure (single file, entity-type-agnostic methods)

### 1.2 Current Document Architecture (Duplication Target)

**5 separate join tables** exist, one per entity type:

| Table | Location (schema) | Location (migration) |
|-------|-------------------|---------------------|
| `epic_documents` | `internal/db/db.go:279-287` | `internal/db/migrate.go:1074-1082` |
| `feature_documents` | `internal/db/db.go:296-304` | `internal/db/migrate.go:990-998` |
| `task_documents` | `internal/db/db.go:313-321` | `internal/db/migrate.go:1158-1166` |
| `bug_documents` | `internal/db/db.go:2288-2296` | E18 migration |
| `change_card_documents` | `internal/db/db.go:2312-2320` | E18 migration |

All 5 tables have identical structure: `{entity}_id INTEGER NOT NULL, document_id INTEGER NOT NULL, created_at TIMESTAMP`, with FK to the entity table and FK to `documents(id)`.

**Repository**: `internal/repository/document_repository.go` (449 lines)
- Contains 15 per-entity methods: `LinkToEpic`, `LinkToFeature`, `LinkToTask`, `LinkToBug`, `LinkToChangeCard`, `UnlinkFromEpic`, `UnlinkFromFeature`, `UnlinkFromTask`, `UnlinkFromBug`, `UnlinkFromChangeCard`, `ListForEpic`, `ListForFeature`, `ListForTask`, `ListForBug`, `ListForChangeCard`
- Core methods (`CreateOrGet`, `GetByID`, `GetByTitle`, `Delete`) are entity-agnostic -- these stay unchanged
- `task_documents` additionally has a `link_type` column (via `LinkToTaskWithType`)

### 1.3 Current History Architecture (Task-Only)

**Single table**: `task_history` (defined at `internal/db/db.go:183-194`)
- Columns: `id, task_id, old_status, new_status, agent, notes, forced, rejection_reason, timestamp`
- FK: `task_id REFERENCES tasks(id) ON DELETE CASCADE`
- Only available for tasks -- epics, features, bugs, change cards have NO history tracking

**Repository**: `internal/repository/task_history_repository.go` (360 lines)
- Methods: `Create`, `ListByTask`, `ListRecent`, `ListWithFilters`, `GetHistoryByTaskKey`, `GetByID`, `GetRejectionHistoryForTask`
- `ListWithFilters` supports joining through tasks -> features -> epics for scope-based filtering
- Model: `internal/models/task_history.go` -- `TaskHistory` struct with `TaskID int64`

**Service**: `internal/services/task_history_service.go` (171 lines)
- `GetTaskHistory`, `ListHistory`, `GetWorkSessions`, `GetSessionAnalytics`
- Task-specific by design; does not handle other entity types

### 1.4 Entity Interface Foundation (E21-F01)

The `models.Entity` interface (`internal/models/entity.go:11-26`) provides:
- `GetID() int64` -- database ID for `entity_id` column
- `GetEntityType() EntityType` -- returns the type string for `entity_type` column
- `GetStatus() string` -- current status for history tracking

All 5 entity types implement this interface. This is the foundation for polymorphic operations.

### 1.5 EntityService.TransitionStatus (History Integration Point)

`internal/services/entity_service.go:110-187` performs status transitions for all entity types. Currently it:
1. Gets entity by key
2. Validates transition
3. Updates status via repo
4. Creates rejection note (optional)
5. Resolves orchestrator action

**Missing step**: Creating a history record. This is the natural injection point for automatic entity_history recording. After step 7 (UpdateStatus), a new step should create a history record in `entity_history`.

---

## 2. Existing Implementations with File Paths

### Schema & Migration

| Component | File Path | Lines |
|-----------|-----------|-------|
| Core schema (task_history, documents, join tables) | `/home/jwwel/projects/shark-task-manager/internal/db/db.go` | 183-321, 2288-2320 |
| Migration functions | `/home/jwwel/projects/shark-task-manager/internal/db/migrate.go` | 256-267, 990-1166 |
| Schema version constant | `/home/jwwel/projects/shark-task-manager/internal/db/db.go:397` | `CurrentSchemaVersion = 6` |
| entity_notes schema (reference pattern) | `/home/jwwel/projects/shark-task-manager/internal/db/db.go` | (in entity_notes CREATE TABLE) |

### Models

| Component | File Path |
|-----------|-----------|
| Entity interface | `/home/jwwel/projects/shark-task-manager/internal/models/entity.go` |
| EntityType constants | `/home/jwwel/projects/shark-task-manager/internal/models/entity_note.go:10-18` |
| TaskHistory model | `/home/jwwel/projects/shark-task-manager/internal/models/task_history.go` |
| EntityNote model | `/home/jwwel/projects/shark-task-manager/internal/models/entity_note.go` |
| Document model | `/home/jwwel/projects/shark-task-manager/internal/models/document.go` (assumed) |

### Repository Layer

| Component | File Path | Lines |
|-----------|-----------|-------|
| Document repository (all 15 entity-specific methods) | `/home/jwwel/projects/shark-task-manager/internal/repository/document_repository.go` | 449 |
| Task history repository | `/home/jwwel/projects/shark-task-manager/internal/repository/task_history_repository.go` | 360 |
| Entity note repository (reference pattern) | `/home/jwwel/projects/shark-task-manager/internal/repository/entity_note_repository.go` | 645 |

### Service Layer

| Component | File Path | Lines |
|-----------|-----------|-------|
| EntityDocumentService (callback-based abstraction) | `/home/jwwel/projects/shark-task-manager/internal/services/entity_document_service.go` | 178 |
| EntityService (TransitionStatus) | `/home/jwwel/projects/shark-task-manager/internal/services/entity_service.go` | ~215 |
| TaskHistoryService | `/home/jwwel/projects/shark-task-manager/internal/services/task_history_service.go` | 171 |
| Task/Feature/Epic/Bug/ChangeCard services | `/home/jwwel/projects/shark-task-manager/internal/services/` | Various |

### CLI Commands

| Component | File Path |
|-----------|-----------|
| Related docs (add/delete/list) | `/home/jwwel/projects/shark-task-manager/internal/cli/commands/related_docs.go` |
| History command (project-wide) | `/home/jwwel/projects/shark-task-manager/internal/cli/commands/history.go` |
| Task history command | `/home/jwwel/projects/shark-task-manager/internal/cli/commands/task_history.go` |
| Status group (status history) | `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status_group.go` |

---

## 3. Integration Points

### 3.1 Database Layer (`internal/db/`)

**Changes needed:**
- New migration function to create `entity_documents` and `entity_history` tables
- Data migration from 5 document tables and `task_history` into new polymorphic tables
- Bump `CurrentSchemaVersion` from 6 to 7
- Drop old tables after verification
- Per `database-critical.md`: developer must set `skip_migrations: false` before running

**Risk**: The `task_documents` table has a `link_type` column not present in other document tables. The `entity_documents` table must include this column or migrate it to a metadata field.

### 3.2 Repository Layer (`internal/repository/`)

**New files needed:**
- `entity_document_repository.go` -- polymorphic document linking (replaces 15 entity-specific methods in `document_repository.go`)
- `entity_history_repository.go` -- polymorphic history for all entity types (replaces `task_history_repository.go`)

**Files to modify:**
- `document_repository.go` -- Remove 15 entity-specific Link/Unlink/List methods; keep `CreateOrGet`, `GetByID`, `GetByTitle`, `Delete`

**Pattern to follow**: `entity_note_repository.go` -- same method signatures parameterized by `EntityType` + `entityID`

### 3.3 Service Layer (`internal/services/`)

**EntityDocumentService** (`entity_document_service.go`):
- Currently uses callback functions (`linkFn`, `unlinkFn`, `listFn`) to abstract per-entity operations
- After migration: simplify to use single `EntityDocumentRepository` directly instead of callbacks
- The service already accepts `models.EntityType` -- the interface is naturally polymorphic

**EntityService** (`entity_service.go`):
- `TransitionStatus` method needs a new step 7.5 between UpdateStatus and rejection note creation
- New optional dependency: `EntityHistoryRepository` (set via `SetHistoryRepo` pattern matching `SetNoteRepo`)
- History record created for ALL transitions (not just backward/forced), capturing `from_status`, `to_status`, `agent`, `notes`

**TaskHistoryService** (`task_history_service.go`):
- Must be generalized to `EntityHistoryService` supporting all entity types
- Or kept as TaskHistoryService but backed by the new `entity_history` table with `entity_type='task'` filter
- Work session analytics remain task-specific (no generalization needed)

### 3.4 CLI Commands

**`related-docs` commands** (`related_docs.go`):
- Currently support `--epic`, `--feature`, `--task` flags
- Must add `--bug` and `--change` flags
- Service layer changes are transparent; CLI just adds new flag parsing

**`status history` / `history` commands** (`history.go`, `status_group.go`):
- Currently task-only (`shark status history E07-F01-001`)
- Must support all entity types: `shark status history E07` (epic), `shark status history E07-F01` (feature), `shark status history B001` (bug)
- Key format auto-detection already exists -- just need to route to entity history service

### 3.5 Tests

**Repository tests to update/create:**
- `entity_document_repository_test.go` -- new, follows `entity_note_repository_test.go` pattern
- `entity_history_repository_test.go` -- new
- `document_repository_test.go` -- reduce to core CRUD tests
- `task_history_repository_test.go` -- may become a thin wrapper or redirect

**Service tests:**
- `entity_service_test.go` -- add tests for history recording in TransitionStatus
- `entity_document_service_test.go` -- simplify after callback removal

---

## 4. Extension vs New Code Analysis

| Component | Approach | Rationale |
|-----------|----------|-----------|
| `entity_documents` table | **New** table + migration | Cannot extend 5 separate tables; must create unified table |
| `entity_history` table | **New** table + migration | Cannot extend task-only table to polymorphic |
| `EntityDocumentRepository` | **New** repository file | Clean break from 15-method monolith |
| `EntityHistoryRepository` | **New** repository file | Generalized from task-only pattern |
| `DocumentRepository` | **Extend** (remove methods) | Keep core CRUD, drop entity-specific methods |
| `EntityDocumentService` | **Extend** (simplify) | Remove callback pattern, use direct repository |
| `EntityService.TransitionStatus` | **Extend** (add history step) | Natural injection point, minimal change |
| `TaskHistoryService` | **Extend** (generalize or wrap) | Backward compatibility for task-specific queries |
| `related_docs.go` CLI | **Extend** (add flags) | Add --bug, --change flags |
| `history.go` / `status_group.go` CLI | **Extend** (route by entity type) | Leverage existing key auto-detection |
| `db.go` / `migrate.go` | **Extend** (add migration) | Standard migration pattern |
| `models/entity_history.go` | **New** model file | New `EntityHistory` struct with `EntityType` + `EntityID` |

**Summary**: 4 new files, 8-10 files modified. No architectural changes -- all follow established patterns.

---

## 5. Inter-Feature Technical Dependency Map

```
E21-F01 (Entity Interface) -----> E21-F08 (This Feature)
  [COMPLETED]                       Uses Entity.GetEntityType() and Entity.GetID()
                                    for polymorphic entity_type + entity_id

E21-F07 (BaseEntity Struct) -----> E21-F08
  [ready_for_task_generation]       BaseEntity.GetEntityType() provides entity_type values
                                    NOT a hard blocker -- E21-F01 already provides this

E21-F08 -----> E21-F09 (Service Delegation)
                 EntityService.TransitionStatus should auto-create history records
                 F08 provides the entity_history table; F09 wires it into transitions

E21-F08 -----> E21-F11 (Polymorphic Entity Relationships)
                 Same polymorphic table pattern reused for entity_relationships
                 F08 proves the migration and repository patterns

E21-F08 -----> E21-F12 (Polymorphic Acceptance Criteria)
                 Same polymorphic pattern; or removal of unused system
                 F08 establishes the precedent
```

**Hard dependency**: E21-F01 must be complete (it IS complete).
**Soft dependency**: E21-F07 (BaseEntity) provides convenience but is not required -- entity types already implement `GetEntityType()`.
**Blocks**: E21-F09, E21-F11, E21-F12 all benefit from this feature being done first.

---

## 6. Implementation Approach Recommendations

### 6.1 Phased Implementation (Recommended)

**Phase 1: Schema + Repository (2-3 tasks)**
1. Create `entity_documents` and `entity_history` tables via migration
2. Create `EntityDocumentRepository` and `EntityHistoryRepository`
3. Write migration functions to copy data from old tables
4. Bump `CurrentSchemaVersion` to 7

**Phase 2: Service Layer Updates (2-3 tasks)**
1. Simplify `EntityDocumentService` to use new polymorphic repository
2. Add `EntityHistoryRepository` dependency to `EntityService`
3. Add history recording step in `TransitionStatus`
4. Generalize or wrap `TaskHistoryService` for all entity types

**Phase 3: CLI + Cleanup (2-3 tasks)**
1. Add `--bug` and `--change` flags to `related-docs` commands
2. Update `status history` to support all entity types via key auto-detection
3. Drop old per-entity document tables
4. Clean up old repository methods

### 6.2 Key Technical Decisions

**Decision 1: `link_type` column for `entity_documents`**
- `task_documents` has a `link_type` column; others do not
- Recommendation: Include `link_type TEXT DEFAULT 'general'` in `entity_documents` for forward compatibility
- Migrate task_documents with their existing link_type values; other entities get 'general'

**Decision 2: History model naming**
- Option A: New `EntityHistory` model with `EntityType` + `EntityID` fields
- Option B: Reuse `TaskHistory` with added entity fields
- Recommendation: **Option A** -- clean separation, no confusion with legacy model

**Decision 3: Old table retention period**
- Feature spec says "drop after verification"
- Recommendation: Keep old tables through one release cycle, then drop in a separate migration
- This provides a rollback path if issues are discovered

**Decision 4: History recording in TransitionStatus vs per-service**
- Option A: Single recording point in `EntityService.TransitionStatus` (DRY)
- Option B: Each entity service records its own history
- Recommendation: **Option A** -- TransitionStatus is already the unified entry point for all transitions; adding history here means zero per-entity code needed

### 6.3 Risk Mitigations

| Risk | Mitigation |
|------|------------|
| Data loss during migration | Create new tables alongside old; verify row counts before dropping old |
| `skip_migrations` flag | Document in PR that developer must set `skip_migrations: false` |
| Performance regression on large datasets | Composite index on `(entity_type, entity_id)` ensures O(log n) lookups |
| Breaking existing `task history` queries | Keep `TaskHistoryService` as a thin wrapper over `EntityHistoryRepository` with `entity_type='task'` filter |
| FK integrity (polymorphic columns lack FK) | Application-level validation in repository; same approach as `entity_notes` |

---

## 7. Quantitative Impact

| Metric | Before | After |
|--------|--------|-------|
| Document join tables | 5 | 1 |
| History tables | 1 (task-only) | 1 (all entities) |
| Total tables affected | 6 | 2 |
| Entity types with document support | 5/5 (via 5 tables) | 5/5 (via 1 table) |
| Entity types with history support | 1/5 (task only) | 5/5 (all) |
| Per-entity document repository methods | 15 | 3 (link, unlink, list -- all polymorphic) |
| Code to add new entity's document support | ~60 lines (new table + 3 methods) | 0 lines (register entity type) |
| `document_repository.go` lines | 449 | ~150 (core CRUD only) |

---

*Last Updated*: 2026-03-20
