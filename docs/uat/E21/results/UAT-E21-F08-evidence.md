# UAT Evidence: E21-F08 Polymorphic Data Model Unification

**Collected**: 2026-03-21
**Feature**: E21-F08 — Polymorphic Data Model Unification
**Purpose**: Raw evidence collection for User Acceptance Testing

---

## Scenario 1: Schema Migration and Data Preservation (T-E21-F08-001)

### Spec Quotes

From `feature.md`:

> **REQ-F-001**: Polymorphic Entity Documents Table
> - Create `entity_documents` table with polymorphic entity reference
> - Schema: `entity_type TEXT NOT NULL`, `entity_id INTEGER NOT NULL`, `document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE`, `UNIQUE(entity_type, entity_id, document_id)`

> **REQ-F-002**: Polymorphic Entity History Table
> - Create `entity_history` table for status change audit trail
> - Schema: `entity_type TEXT NOT NULL`, `entity_id INTEGER NOT NULL`, `from_status TEXT NOT NULL`, `to_status TEXT NOT NULL`, `changed_by TEXT`, `notes TEXT`, `changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`

> **REQ-F-003**: Data Migration
> - `epic_documents` rows migrated to `entity_documents` with entity_type='epic'
> - `feature_documents` rows migrated with entity_type='feature'
> - `task_documents` rows migrated with entity_type='task'
> - `task_history` rows migrated to `entity_history` with entity_type='task'
> - Row counts verified before/after migration
> - Old tables dropped after verification

> **REQ-NF-001**: Migration Safety — Old tables retained until new table data verified. Target: Zero data loss during migration.

### Implementation Code

**Schema DDL** (`internal/db/db.go`):

```
Line 183: CREATE TABLE IF NOT EXISTS entity_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    from_status TEXT,
    to_status TEXT NOT NULL,
    changed_by TEXT,
    notes TEXT,
    forced BOOLEAN DEFAULT 0,
    rejection_reason TEXT,
    changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
Lines 197-199: CREATE INDEX IF NOT EXISTS idx_entity_history_lookup ON entity_history(entity_type, entity_id);
                CREATE INDEX IF NOT EXISTS idx_entity_history_time ON entity_history(changed_at);
                CREATE INDEX IF NOT EXISTS idx_entity_history_entity_time ON entity_history(entity_type, entity_id, changed_at);
```

```
Line 302: CREATE TABLE IF NOT EXISTS entity_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    link_type TEXT NOT NULL DEFAULT 'general',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_type, entity_id, document_id)
);
Lines 313-314: CREATE INDEX IF NOT EXISTS idx_entity_documents_lookup ON entity_documents(entity_type, entity_id);
                CREATE INDEX IF NOT EXISTS idx_entity_documents_document ON entity_documents(document_id);
```

**Migration function** (`internal/db/db.go`, lines 2478-2531): `migrateToPolymorphicTables(db)` -- orchestrates:
1. `createPolymorphicTables(db)` (line 2535) -- creates entity_documents + entity_history with indexes
2. `migrateDocumentDataToPolymorphic(db)` (line 2592) -- copies from 5 old doc tables using INSERT OR IGNORE
3. `migrateHistoryDataToPolymorphic(db)` (line 2641) -- copies task_history to entity_history with field mapping
4. `verifyPolymorphicMigration(db)` (line 2696) -- row count verification
5. `dropOldDocumentAndHistoryTables(db)` (line 2743) -- drops old tables, creates compatibility views

**Key migration queries**:
- Document: `INSERT OR IGNORE INTO entity_documents ... SELECT 'epic', epic_id, document_id, 'general', created_at FROM epic_documents` (line 2626)
- Task docs with link_type: `INSERT OR IGNORE INTO entity_documents ... SELECT 'task', task_id, document_id, COALESCE(link_type, 'general'), created_at FROM task_documents` (line 2620)
- History: `INSERT INTO entity_history ... SELECT 'task', task_id, old_status, new_status, agent, notes, COALESCE(forced, 0), rejection_reason, timestamp FROM task_history` (line 2664)

### Test Code

File: `internal/db/migration_polymorphic_test.go`

| Test Function | Lines | Key Assertions |
|---|---|---|
| `TestMigratePolymorphicTables_FreshDB` | 228-281 | entity_documents has 6 columns, entity_history has 10 columns, indexes >= 2 and == 3, no old TABLEs on fresh DB |
| `TestMigratePolymorphicTables_WithData` | 285-331 | 2 epic, 1 feature, 2 task, 1 bug, 1 change docs migrated; 3 task_history rows migrated; old tables dropped, views created |
| `TestMigratePolymorphicTables_Idempotent` | 335-362 | Second run produces no duplicates |
| `TestMigratePolymorphicTables_LinkTypePreserved` | 366-414 | 'specification' and 'design' link_types preserved, NULL -> 'general' |
| `TestMigratePolymorphicTables_HistoryFieldMapping` | 418-474 | entity_type='task', from_status, to_status, changed_by, notes, forced, rejection_reason, changed_at all mapped correctly; NULL forced -> 0 |
| `TestMigratePolymorphicTables_MissingOldTables` | 478-525 | Migration succeeds when only some old tables exist |
| `TestMigratePolymorphicTables_EmptyOldTables` | 529-554 | Migration succeeds with empty old tables |
| `TestMigratePolymorphicTables_VerificationFailure` | 558-587 | When counts don't match, error contains "verification failed", old tables not dropped |
| `TestMigratePolymorphicTables_PartialRun` | 591-617 | Handles interrupted migration (new tables + old tables both exist) |
| `TestMigratePolymorphicTables_IndexesCreated` | 620-660 | Verifies idx_entity_documents_lookup, idx_entity_documents_document, idx_entity_history_lookup, idx_entity_history_time, idx_entity_history_entity_time |
| `TestMigratePolymorphicTables_UniqueConstraint` | 664-698 | INSERT OR IGNORE on duplicate does not create second row |
| `TestInitDB_FreshDatabase_NoOldTables` | 702-727 | entity_documents + entity_history exist; old doc TABLEs do not exist; compatibility VIEWs exist; task_history TABLE exists |

### Test Output

```
=== RUN   TestMigratePolymorphicTables_FreshDB
--- PASS: TestMigratePolymorphicTables_FreshDB (0.00s)
=== RUN   TestMigratePolymorphicTables_WithData
--- PASS: TestMigratePolymorphicTables_WithData (0.00s)
=== RUN   TestMigratePolymorphicTables_Idempotent
--- PASS: TestMigratePolymorphicTables_Idempotent (0.00s)
=== RUN   TestMigratePolymorphicTables_LinkTypePreserved
--- PASS: TestMigratePolymorphicTables_LinkTypePreserved (0.00s)
=== RUN   TestMigratePolymorphicTables_HistoryFieldMapping
--- PASS: TestMigratePolymorphicTables_HistoryFieldMapping (0.00s)
=== RUN   TestMigratePolymorphicTables_MissingOldTables
--- PASS: TestMigratePolymorphicTables_MissingOldTables (0.00s)
=== RUN   TestMigratePolymorphicTables_EmptyOldTables
--- PASS: TestMigratePolymorphicTables_EmptyOldTables (0.00s)
=== RUN   TestMigratePolymorphicTables_VerificationFailure
--- PASS: TestMigratePolymorphicTables_VerificationFailure (0.00s)
=== RUN   TestMigratePolymorphicTables_PartialRun
--- PASS: TestMigratePolymorphicTables_PartialRun (0.00s)
=== RUN   TestMigratePolymorphicTables_IndexesCreated
--- PASS: TestMigratePolymorphicTables_IndexesCreated (0.00s)
=== RUN   TestMigratePolymorphicTables_UniqueConstraint
--- PASS: TestMigratePolymorphicTables_UniqueConstraint (0.00s)
PASS
ok  	github.com/jwwelbor/shark-task-manager/internal/db	0.024s
```

**InitDB fresh DB test:**
```
=== RUN   TestInitDB_FreshDatabase_NoOldTables
--- PASS: TestInitDB_FreshDatabase_NoOldTables (0.04s)
PASS
```

### Factual Notes

- The feature spec lists `from_status TEXT NOT NULL` in REQ-F-002, but the actual DDL uses `from_status TEXT` (nullable) -- this matches the EntityHistory model where `FromStatus` is `*string`. The nullability is needed because initial status creation has no `from_status`.
- The entity_history table has 10 columns (including `forced` and `rejection_reason` not listed in REQ-F-002 spec), which were carried over from the legacy `task_history` table for full field preservation.
- The `entity_documents` table has a `link_type` column not in REQ-F-001 spec, preserved from `task_documents.link_type`.

---

## Scenario 2: EntityHistory Model (T-E21-F08-002)

### Spec Quotes

From `feature.md`:
> `entity_history` table created with `entity_type`, `entity_id`, `from_status`, `to_status`, `changed_at`, `changed_by`, `notes` columns

### Implementation Code

File: `internal/models/entity_history.go`

```go
// Lines 15-26
type EntityHistory struct {
    ID              int64      `json:"id" db:"id"`
    EntityType      EntityType `json:"entity_type" db:"entity_type"`
    EntityID        int64      `json:"entity_id" db:"entity_id"`
    FromStatus      *string    `json:"from_status,omitempty" db:"from_status"`
    ToStatus        string     `json:"to_status" db:"to_status"`
    ChangedBy       *string    `json:"changed_by,omitempty" db:"changed_by"`
    Notes           *string    `json:"notes,omitempty" db:"notes"`
    Forced          bool       `json:"forced" db:"forced"`
    RejectionReason *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`
    ChangedAt       time.Time  `json:"changed_at" db:"changed_at"`
}
```

Validate() (lines 32-46): checks EntityType not empty, valid entity type, EntityID > 0, ToStatus not empty. Does NOT validate status values (per architecture rules).

`EntityDocumentLink` struct (lines 50-57): represents entity_documents join table row.

### Test Code

File: `internal/models/entity_history_test.go` -- 14 test functions:

| Test | Validates |
|---|---|
| `TestEntityHistory_Validate_Valid` | Valid EntityHistory with FromStatus=nil passes |
| `TestEntityHistory_Validate_EmptyEntityType` | Empty EntityType returns error containing "entity_type" |
| `TestEntityHistory_Validate_InvalidEntityType` | Invalid EntityType returns error containing "invalid entity_type" |
| `TestEntityHistory_Validate_ZeroEntityID` | Zero EntityID returns error containing "entity_id" |
| `TestEntityHistory_Validate_NegativeEntityID` | Negative EntityID returns error |
| `TestEntityHistory_Validate_EmptyToStatus` | Empty ToStatus returns error containing "to_status" |
| `TestEntityHistory_Validate_NilFromStatusAccepted` | Nil FromStatus accepted |
| `TestEntityHistory_Validate_AllEntityTypes` | All 5 entity types (epic/feature/task/bug/change) pass |
| `TestEntityHistory_Validate_ArbitraryStatusAccepted` | Arbitrary status values pass (no status validation in model) |
| `TestEntityHistory_ForcedField_IsBool` | Forced defaults to false, settable to true |
| `TestEntityHistory_JSONSerialization` | JSON tags correct, required fields present, omitempty works |
| `TestEntityHistory_JSONSerialization_OmitemptyNil` | Nil pointer fields omitted from JSON |
| `TestEntityHistory_FieldCount` | All 10 fields accessible |
| `TestEntityHistory_Validate_AllNullableFieldsNil` | All nullable fields nil simultaneously is valid |

### Test Output

```
=== RUN   TestEntityHistory_Validate_Valid
--- PASS: TestEntityHistory_Validate_Valid (0.00s)
=== RUN   TestEntityHistory_Validate_EmptyEntityType
--- PASS: TestEntityHistory_Validate_EmptyEntityType (0.00s)
=== RUN   TestEntityHistory_Validate_InvalidEntityType
--- PASS: TestEntityHistory_Validate_InvalidEntityType (0.00s)
=== RUN   TestEntityHistory_Validate_ZeroEntityID
--- PASS: TestEntityHistory_Validate_ZeroEntityID (0.00s)
=== RUN   TestEntityHistory_Validate_NegativeEntityID
--- PASS: TestEntityHistory_Validate_NegativeEntityID (0.00s)
=== RUN   TestEntityHistory_Validate_EmptyToStatus
--- PASS: TestEntityHistory_Validate_EmptyToStatus (0.00s)
=== RUN   TestEntityHistory_Validate_NilFromStatusAccepted
--- PASS: TestEntityHistory_Validate_NilFromStatusAccepted (0.00s)
=== RUN   TestEntityHistory_Validate_AllEntityTypes
    --- PASS: epic/feature/task/bug/change (0.00s each)
--- PASS: TestEntityHistory_Validate_AllEntityTypes (0.00s)
=== RUN   TestEntityHistory_Validate_ArbitraryStatusAccepted
--- PASS: TestEntityHistory_Validate_ArbitraryStatusAccepted (0.00s)
=== RUN   TestEntityHistory_ForcedField_IsBool
--- PASS: TestEntityHistory_ForcedField_IsBool (0.00s)
=== RUN   TestEntityHistory_JSONSerialization
--- PASS: TestEntityHistory_JSONSerialization (0.00s)
=== RUN   TestEntityHistory_JSONSerialization_OmitemptyNil
--- PASS: TestEntityHistory_JSONSerialization_OmitemptyNil (0.00s)
=== RUN   TestEntityHistory_FieldCount
--- PASS: TestEntityHistory_FieldCount (0.00s)
=== RUN   TestEntityHistory_Validate_AllNullableFieldsNil
--- PASS: TestEntityHistory_Validate_AllNullableFieldsNil (0.00s)
PASS
ok  	github.com/jwwelbor/shark-task-manager/internal/models	0.003s
```

All 14 tests pass.

---

## Scenario 3: Polymorphic Document Linking (T-E21-F08-003, T-E21-F08-005)

### Spec Quotes

> **REQ-F-004**: Single `EntityDocumentRepository` replaces `EpicDocumentRepository`, `FeatureDocumentRepository`, `TaskDocumentRepository`. All methods accept `entityType` and `entityID` parameters. Existing service interfaces updated to use new repositories.

> **Story 1 AC**: Document linking works for all 5 entity types via same CLI commands.

### Implementation Code

**Repository** (`internal/repository/entity_document_repository.go`):
- `EntityDocumentRepository` struct with `Link`, `Unlink`, `ListForEntity` methods (lines 12-125)
- `Link` uses `INSERT OR IGNORE INTO entity_documents` with `COALESCE(NULLIF(?, ''), 'general')` for default link_type (line 49)
- `Unlink` uses `DELETE FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?` (line 73)
- `ListForEntity` joins `documents d INNER JOIN entity_documents ed` with `ORDER BY ed.created_at DESC` (line 96-101)
- Input validation via `validateEntityType()` and `validateEntityID()` (lines 22-35)

**Adapter** (`internal/repository/polymorphic_doc_adapter.go`):
- `PolymorphicDocRepoAdapter` wraps `EntityDocumentRepository` with `ListForEpic`, `ListForFeature`, `ListForTask` methods delegating to `ListForEntity` with appropriate entity type.

**Service** (`internal/services/entity_document_service.go`):
- `EntityDocumentService` with `LinkDocumentByKey`, `UnlinkDocumentByKey`, `ListDocumentsByKey` (lines 30-122)
- Uses `entityLookupFn` callback to resolve entity key to (entityID, entityType) pair
- Depends on `EntityDocumentRepository` (writable) and `EntityDocumentLinkRepository` (polymorphic) interfaces

### Test Code

**Repository tests** (`internal/repository/entity_document_repository_test.go`):

| Test | Key Assertions |
|---|---|
| `TestEntityDocumentRepo_Link` | Links docs for all 5 entity types, verifies rows created |
| `TestEntityDocumentRepo_LinkIdempotent` | Duplicate link produces exactly 1 row |
| `TestEntityDocumentRepo_LinkWithType` | Empty link_type -> "general"; custom "specification" preserved |
| `TestEntityDocumentRepo_Unlink` | Row removed, document itself still exists, unlink non-existent is no-op |
| `TestEntityDocumentRepo_ListForEntity` | Returns 3 docs, DESC ordering, fields populated, cross-type isolation, empty result is non-nil slice |
| `TestEntityDocumentRepo_InvalidEntityType` | All 4 invalid types rejected for Link/Unlink/ListForEntity |
| `TestEntityDocumentRepo_CascadeDelete` | Deleting document cascades to entity_documents |
| `TestEntityDocumentRepo_InvalidEntityID` | entity_id=0 and -1 rejected |

**Service tests** (`internal/services/entity_document_service_test.go`):

| Test | Key Assertions |
|---|---|
| `TestEntityDocumentService_LinkDocumentByKey_HappyPath` | Correct entityType/entityID/docID/linkType captured |
| `TestEntityDocumentService_LinkDocumentByKey_EntityNotFound` | Error: "entity not found: not found" |
| `TestEntityDocumentService_LinkDocumentByKey_CreateOrGetFails` | Error propagated, Link not called |
| `TestEntityDocumentService_LinkDocumentByKey_LinkFails` | Error: "failed to link document to epic E07: FK violation" |
| `TestEntityDocumentService_UnlinkDocumentByKey_HappyPath` | Correct params captured |
| `TestEntityDocumentService_UnlinkDocumentByKey_DocumentNotFound_Idempotent` | No error for missing doc |
| `TestEntityDocumentService_UnlinkDocumentByKey_EntityNotFound` | Error propagated |
| `TestEntityDocumentService_ListDocumentsByKey_HappyPath` | Returns 2 docs, correct entity params |
| `TestEntityDocumentService_ListDocumentsByKey_NilResult` | nil -> non-nil empty slice |
| `TestEntityDocumentService_LinkDocumentByKey_AllEntityTypes` | All 5 types (epic/feature/task/bug/change) work |

### Test Output

**Repository:**
```
=== RUN   TestEntityDocumentRepo_Link
--- PASS: TestEntityDocumentRepo_Link (0.00s)
=== RUN   TestEntityDocumentRepo_LinkIdempotent
--- PASS: TestEntityDocumentRepo_LinkIdempotent (0.00s)
=== RUN   TestEntityDocumentRepo_LinkWithType
--- PASS: TestEntityDocumentRepo_LinkWithType (0.00s)
=== RUN   TestEntityDocumentRepo_Unlink
--- PASS: TestEntityDocumentRepo_Unlink (0.00s)
=== RUN   TestEntityDocumentRepo_ListForEntity
--- PASS: TestEntityDocumentRepo_ListForEntity (0.00s)
=== RUN   TestEntityDocumentRepo_InvalidEntityType (12 subtests)
--- PASS: TestEntityDocumentRepo_InvalidEntityType (0.00s)
=== RUN   TestEntityDocumentRepo_CascadeDelete
--- PASS: TestEntityDocumentRepo_CascadeDelete (0.00s)
=== RUN   TestEntityDocumentRepo_InvalidEntityID
--- PASS: TestEntityDocumentRepo_InvalidEntityID (0.00s)
PASS ok 0.009s
```

**Service:**
```
=== RUN   TestEntityDocumentService_LinkDocumentByKey_HappyPath
--- PASS (0.00s)
... (all 13 tests PASS)
=== RUN   TestEntityDocumentService_LinkDocumentByKey_AllEntityTypes
    --- PASS: epic/feature/task/bug/change (0.00s each)
PASS ok 0.004s
```

---

## Scenario 4: Polymorphic History CRUD (T-E21-F08-004)

### Spec Quotes

> **REQ-F-004**: Single `EntityHistoryRepository` replaces `TaskHistoryRepository`. All methods accept `entityType` and `entityID` parameters.

### Implementation Code

File: `internal/repository/entity_history_repository.go`

- `EntityHistoryRepository` struct with `Create`, `ListByEntity`, `ListRecent`, `ListWithFilters` methods (lines 28-242)
- `Create` calls `history.Validate()` before INSERT, sets `history.ID` after (lines 40-75)
- `ListByEntity` queries by entity_type + entity_id, ORDER BY changed_at DESC (lines 79-104)
- `ListRecent` queries all types with LIMIT (lines 108-133)
- `ListWithFilters` with dynamic WHERE clause building for EntityType, EntityID, ChangedBy, Since, FromStatus, ToStatus + pagination (lines 138-207)
- `EntityHistoryFilters` struct with all-optional filter fields (lines 14-23)
- Shared `scanEntityHistories` helper scans all 10 columns (lines 211-242)

### Test Code

File: `internal/repository/entity_history_repository_test.go`

| Test | Key Assertions |
|---|---|
| `TestEntityHistoryRepo_Create` | Creates for all 5 entity types, ID set, DB verified, 5 distinct types counted |
| `TestEntityHistoryRepo_Create_Validation` | Empty to_status rejected with "validation failed", no DB row created |
| `TestEntityHistoryRepo_Create_InvalidType` | Invalid entity type rejected |
| `TestEntityHistoryRepo_ListByEntity` | Returns correct 3 records for target, not other entity's records; DESC ordering; empty result returns non-nil slice |
| `TestEntityHistoryRepo_ListRecent` | Returns exactly limit records, DESC ordered; limit exceeding total returns all |
| `TestEntityHistoryRepo_ListWithFilters` | 10 subtests: filter_by_entity_type, filter_by_changed_by, filter_by_to_status, filter_by_from_status, filter_by_since, filter_combined, filter_entity_type_and_entity_id, pagination, default_limit_50, desc_ordering |
| `TestEntityHistoryRepo_NullHandling` | NULL FromStatus/ChangedBy/Notes/RejectionReason round-trip as nil; Forced=true persisted correctly |

### Test Output

```
=== RUN   TestEntityHistoryRepo_Create
    --- PASS: create_epic/create_feature/create_task/create_bug/create_change
--- PASS: TestEntityHistoryRepo_Create (0.00s)
=== RUN   TestEntityHistoryRepo_Create_Validation
    --- PASS: empty_to_status
--- PASS: TestEntityHistoryRepo_Create_Validation (0.00s)
=== RUN   TestEntityHistoryRepo_Create_InvalidType
--- PASS: TestEntityHistoryRepo_Create_InvalidType (0.00s)
=== RUN   TestEntityHistoryRepo_ListByEntity
    --- PASS: returns_correct_records / no_records_returns_empty_slice
--- PASS: TestEntityHistoryRepo_ListByEntity (0.01s)
=== RUN   TestEntityHistoryRepo_ListRecent
    --- PASS: returns_limited_records / limit_exceeds_total
--- PASS: TestEntityHistoryRepo_ListRecent (0.00s)
=== RUN   TestEntityHistoryRepo_ListWithFilters (10 subtests all PASS)
--- PASS: TestEntityHistoryRepo_ListWithFilters (0.00s)
=== RUN   TestEntityHistoryRepo_NullHandling
--- PASS: TestEntityHistoryRepo_NullHandling (0.00s)
PASS ok 0.022s
```

---

## Scenario 5: History Recording in EntityService.TransitionStatus (T-E21-F08-006)

### Spec Quotes

> **REQ-F-005**: EntityService.TransitionStatus automatically creates entity_history records. Every status transition creates a history record with from_status, to_status, changed_by, notes. Works for all 5 entity types without per-entity code.

### Implementation Code

File: `internal/services/entity_service.go`

```go
// Line 72: EntityService struct
type EntityService struct {
    workflowSvc *workflow.Service
    noteRepo    RejectionNoteCreator
    historyRepo EntityHistoryRecorder // optional, for history recording during transitions
}
```

```go
// Lines 107-108: SetHistoryRepo sets the entity history recorder.
func (s *EntityService) SetHistoryRepo(repo EntityHistoryRecorder) {
    s.historyRepo = repo
}
```

**TransitionStatus Step 7.5** (lines 173-202):
```go
if s.historyRepo != nil {
    history := &models.EntityHistory{
        EntityType: entityType,
        EntityID:   entity.GetID(),
        ToStatus:   targetStatus,
        Forced:     opts.Force,
        ChangedAt:  time.Now(),
    }
    if fromStatus := entity.GetStatus(); fromStatus != "" {
        history.FromStatus = &fromStatus
    }
    if opts.Agent != "" {
        history.ChangedBy = &opts.Agent
    }
    if opts.Reason != "" {
        // ... sets Notes
    }
    if err := s.historyRepo.Create(ctx, history); err != nil {
        log.Printf("warning: failed to record entity history for %s %s: %v", entityType, key, err)
    }
}
```

Key: History recording is **non-blocking** -- errors are logged but do not fail the transition.

**ForLevel propagation** (lines 91-97): `historyRepo` is copied when creating level-scoped instances.

### Test Code

File: `internal/services/entity_service_test.go`

| Test | Lines | Key Assertions |
|---|---|---|
| `TestEntityService_TransitionStatus_CreatesHistory` | 660-701 | 1 history record created; entity_type=feature, entity_id=42, from_status="draft", to_status="active", changed_by="agent-ba", forced=false |
| `TestEntityService_TransitionStatus_NoHistoryRepo` | 714-739 | Transition succeeds with nil historyRepo (graceful degradation) |
| `TestEntityService_TransitionStatus_HistoryError` | 741-772 | History error does NOT block transition; logs warning "failed to record entity history" |
| `TestEntityService_TransitionStatus_ForcedHistory` | 774-806 | forced=true, notes="emergency rollback", changed_by="admin" |
| `TestEntityService_TransitionStatus_BackwardHistory` | 813-844 | Backward transition history recorded with reason |
| `TestEntityService_ForLevel_PropagatesHistoryRepo` | (separate test) | historyRepo preserved through ForLevel() |

Compile-time assertion (line 18):
```go
var _ EntityHistoryRecorder = (*repository.EntityHistoryRepository)(nil)
```

### Test Output

```
=== RUN   TestEntityService_TransitionStatus_CreatesHistory
--- PASS: TestEntityService_TransitionStatus_CreatesHistory (0.00s)
=== RUN   TestEntityService_TransitionStatus_NoHistoryRepo
--- PASS: TestEntityService_TransitionStatus_NoHistoryRepo (0.00s)
=== RUN   TestEntityService_TransitionStatus_HistoryError
2026/03/21 07:55:14 warning: failed to record entity history for epic E01: database connection lost
--- PASS: TestEntityService_TransitionStatus_HistoryError (0.00s)
=== RUN   TestEntityService_TransitionStatus_ForcedHistory
--- PASS: TestEntityService_TransitionStatus_ForcedHistory (0.00s)
=== RUN   TestEntityService_TransitionStatus_BackwardHistory
--- PASS: TestEntityService_TransitionStatus_BackwardHistory (0.00s)
=== RUN   TestEntityService_ForLevel_PropagatesHistoryRepo
--- PASS: TestEntityService_ForLevel_PropagatesHistoryRepo (0.00s)
PASS ok 0.012s
```

---

## Scenario 6: EntityHistoryService (T-E21-F08-007)

### Spec Quotes

> **Story 2 AC**: `shark status history E21-F07` returns feature status changes. `shark status history B001` returns bug status changes.

### Implementation Code

File: `internal/services/entity_history_service.go`

- `EntityHistoryService` with `GetHistory` and `GetRecentHistory` methods (lines 23-64)
- `GetHistory` resolves entity key via `EntityRegistry`, then calls `historyRepo.ListByEntity` (lines 40-55)
- `GetRecentHistory` defaults to limit=50, calls `historyRepo.ListRecent` (lines 59-64)
- Depends on `EntityHistoryQuerier` interface (ListByEntity + ListRecent) and `EntityRegistry` for key resolution

### Test Code

File: `internal/services/entity_history_service_test.go`

| Test | Key Assertions |
|---|---|
| `TestEntityHistoryService_GetHistory` | Returns 2 entries for feature E21-F07 (entity_id=42) |
| `TestEntityHistoryService_GetHistory_EntityNotFound` | Error contains "E99-F99-999" and "failed to get history" |
| `TestEntityHistoryService_GetHistory_UnregisteredType` | Error contains "no repository registered for entity type" |
| `TestEntityHistoryService_GetHistory_EmptyResult` | Non-nil empty slice |
| `TestEntityHistoryService_GetHistory_AllEntityTypes` | All 5 types (epic/feature/task/bug/change) work correctly |
| `TestEntityHistoryService_GetRecentHistory` | Returns 3 entries with limit=10 |
| `TestEntityHistoryService_GetRecentHistory_DefaultLimit` | 0->50, -1->50, 25->25 |
| `TestEntityHistoryService_GetRecentHistory_EmptyResult` | Non-nil empty slice |
| `TestEntityHistoryService_GetHistory_RepoError` | Error propagated: "database connection failed" |

### Test Output

```
=== RUN   TestEntityHistoryService_GetHistory
--- PASS (0.00s)
=== RUN   TestEntityHistoryService_GetHistory_EntityNotFound
--- PASS (0.00s)
=== RUN   TestEntityHistoryService_GetHistory_UnregisteredType
--- PASS (0.00s)
=== RUN   TestEntityHistoryService_GetHistory_EmptyResult
--- PASS (0.00s)
=== RUN   TestEntityHistoryService_GetHistory_AllEntityTypes
    --- PASS: epic/feature/task/bug/change (0.00s each)
--- PASS (0.00s)
=== RUN   TestEntityHistoryService_GetRecentHistory
--- PASS (0.00s)
=== RUN   TestEntityHistoryService_GetRecentHistory_DefaultLimit (3 subtests all PASS)
--- PASS (0.00s)
=== RUN   TestEntityHistoryService_GetRecentHistory_EmptyResult
--- PASS (0.00s)
=== RUN   TestEntityHistoryService_GetHistory_RepoError
--- PASS (0.00s)
PASS ok 0.004s
```

---

## Scenario 7: CLI Updates (T-E21-F08-008)

### Spec Quotes

> **Story 2 AC**: `shark status history E21-F07` returns feature status changes. `shark status history B001` returns bug status changes.

### Implementation Code

File: `internal/cli/commands/status_group.go`

`runStatusHistory` function (lines 553-613):
1. Parses entity type and key via `ParseGetArgs(args)`
2. Normalizes "change_card" to "change" (ADR-1)
3. Calls `cli.GetEntityHistoryService().GetHistory(ctx, entityType, key)`
4. Applies --limit flag truncation
5. Maps `EntityHistory` to `StatusHistoryEntry` structs
6. Outputs `StatusHistoryResult` as JSON or human-readable table

**CLI wiring** (`internal/cli/services_global.go`, line 408-430):
```go
func GetEntityHistoryService() *services.EntityHistoryService {
    db := GetDB(...)
    historyRepo := repository.NewEntityHistoryRepository(db)
    registry := buildEntityRegistry(db) // registers all 5 entity types
    return services.NewEntityHistoryService(historyRepo, registry)
}
```

**JSON output types**:
- `StatusHistoryResult{EntityType, EntityKey, History []StatusHistoryEntry, Total}` (lines 35-40)
- `StatusHistoryEntry{Timestamp, OldStatus, NewStatus, Agent, Notes}` (lines 43-49)

### Test Code

File: `internal/cli/commands/status_group_test.go`

| Test | Key Assertions |
|---|---|
| `TestStatusHistoryResult_JSON` (3 subtests) | With entries: roundtrip serialization correct; empty history: `[]`; nil history: serializes as null |
| `TestStatusHistoryEntry_JSON` (3 subtests) | All fields populated correct; optional fields omitted when empty; roundtrip deserialization |
| `TestStatusHistory_FeatureKey` | EntityHistory->StatusHistoryEntry mapping for features; from_status/changed_by/notes mapped from pointers |
| `TestStatusHistory_BugKey` | entity_type="bug" in JSON output |
| `TestStatusHistory_ChangeCardNormalization` | "change_card" normalized to "change" in output |
| `TestStatusHistory_EpicKey` | Works for epic entity type |
| `TestStatusHistory_LimitTruncation` | --limit=3 keeps last 3 of 5 entries |
| `TestStatusHistory_EmptyResult` | Empty history returns correct JSON structure with total=0 |

### Test Output

```
=== RUN   TestStatusHistoryResult_JSON
    --- PASS: with_entries / with_empty_history / nil_history_serializes_as_null
--- PASS (0.00s)
=== RUN   TestStatusHistoryEntry_JSON
    --- PASS: all_fields_populated / optional_fields_omitted_when_empty / roundtrip_deserialization
--- PASS (0.00s)
=== RUN   TestStatusHistory_FeatureKey
--- PASS (0.00s)
=== RUN   TestStatusHistory_BugKey
--- PASS (0.00s)
=== RUN   TestStatusHistory_ChangeCardNormalization
--- PASS (0.00s)
=== RUN   TestStatusHistory_EpicKey
--- PASS (0.00s)
=== RUN   TestStatusHistory_LimitTruncation
--- PASS (0.00s)
=== RUN   TestStatusHistory_EmptyResult
--- PASS (0.00s)
PASS ok 0.010s
```

---

## Scenario 8: Legacy Code Removal (T-E21-F08-009)

### Spec Quotes

> **Story 3 AC**: Data migrated from `epic_documents`, `feature_documents`, `task_documents`, `bug_documents`, `change_card_documents` to `entity_documents`. Old tables dropped after successful migration verification. Repository code updated to use single `entity_documents` table.

> **REQ-F-004**: Single `EntityDocumentRepository` replaces `EpicDocumentRepository`, `FeatureDocumentRepository`, `TaskDocumentRepository`.

### Evidence

**No entity-specific document table references in document_repository.go:**
Grep for `epic_documents|feature_documents|task_documents` in `internal/repository/document_repository.go` returned: **No matches found**.

**Clean compilation:**
```
$ go build ./...
(no output -- clean build)
```

**Compatibility approach:**
Old per-entity document tables are replaced by VIEWs (not TABLEs) that map to `entity_documents`. Verified in migration test `TestMigratePolymorphicTables_WithData` (lines 319-330):
```go
for _, table := range []string{"epic_documents", "feature_documents", "task_documents", "bug_documents", "change_card_documents"} {
    var tableCount int
    db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&tableCount)
    assert.Equal(t, 0, tableCount, "%s TABLE should not exist after migration", table)
    var viewCount int
    db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name=?`, table).Scan(&viewCount)
    assert.Equal(t, 1, viewCount, "%s VIEW should exist after migration", table)
}
```

**PolymorphicDocRepoAdapter** (`internal/repository/polymorphic_doc_adapter.go`):
Wraps `EntityDocumentRepository` with `ListForEpic/ListForFeature/ListForTask` methods that delegate to `ListForEntity` -- provides backward compatibility for callers that haven't been updated to use polymorphic API directly.

**Display views updated:** The `db.go` display views (epic_display_data, feature_display_data, task_display_data) now JOIN `entity_documents` instead of per-entity tables (lines 1844, 1901, 1972).

**task_history TABLE preserved** for backward compatibility (per `TestMigratePolymorphicTables_EmptyOldTables` line 546):
```go
assert.True(t, testTableExists(t, db, "task_history"), "task_history TABLE should still exist")
```

---

## Full Suite Results

### make fmt
```
Formatting code...
```
No changes (clean).

### make lint
```
0 issues.
```

### make test
All tests pass (output truncated to last 30 lines shows final packages passing):
```
PASS
ok  github.com/jwwelbor/shark-task-manager/internal/workflow  (cached)
```

### go build ./...
Clean compilation with no output (success).

---

## Summary of Factual Notes

1. **entity_history schema discrepancy**: The feature spec REQ-F-002 lists `from_status TEXT NOT NULL` but implementation uses nullable `from_status TEXT` (and model uses `*string`). This is deliberate -- initial status changes have no from_status. The entity_history table also has 10 columns (adding `forced`, `rejection_reason`) vs the 7 listed in the spec, carrying over task_history fields.

2. **entity_documents has link_type column**: Not mentioned in REQ-F-001 spec but present in implementation, preserving task_documents.link_type during migration.

3. **task_history TABLE retained**: The old `task_history` table is kept alongside `entity_history` for backward compatibility. Data is duplicated in both tables after migration. The 5 per-entity document TABLEs are dropped and replaced with VIEWs.

4. **History recording is non-blocking**: EntityService.TransitionStatus logs warnings on history creation failure but does not fail the transition itself. This is a design decision tested in `TestEntityService_TransitionStatus_HistoryError`.

5. **All test commands produce PASS**: 11 migration tests, 14 model tests, 8 repo doc tests, 7 repo history tests, 13 service doc tests, 9 service history tests, 6 entity service history tests, 8+ CLI tests -- all passing.
