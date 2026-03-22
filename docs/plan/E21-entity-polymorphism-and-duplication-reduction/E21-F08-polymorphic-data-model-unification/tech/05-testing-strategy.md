# Testing Strategy: E21-F08 Polymorphic Data Model Unification

**Feature**: E21-F08
**Author**: Architect
**Date**: 2026-03-20
**Status**: Draft

---

## 1. Test Categories

### 1.1 Repository Tests (Real Database)

Per project testing architecture: repository tests are the ONLY tests that use a real database.

#### EntityDocumentRepository Tests

**File**: `internal/repository/entity_document_repository_test.go`

**Pattern**: Follow `entity_note_repository_test.go` (established pattern)

| Test | Description | Assertions |
|------|-------------|------------|
| `TestEntityDocumentRepo_Link` | Link a document to each entity type | Row created in entity_documents; re-linking same doc is idempotent (INSERT OR IGNORE) |
| `TestEntityDocumentRepo_Unlink` | Unlink a document from each entity type | Row removed; unlink non-existent is no-op |
| `TestEntityDocumentRepo_ListForEntity` | List documents for each entity type | Returns correct documents; ordered by created_at DESC; empty list for no links |
| `TestEntityDocumentRepo_LinkWithType` | Link with specific link_type | link_type preserved in query results |
| `TestEntityDocumentRepo_InvalidEntityType` | Link with invalid entity_type | Returns validation error |
| `TestEntityDocumentRepo_CascadeDelete` | Delete a document | entity_documents rows cascade-deleted via FK |

**Setup/Teardown pattern**:
```go
func TestEntityDocumentRepo_Link(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := repository.NewDB(database)
    repo := repository.NewEntityDocumentRepository(db)
    docRepo := repository.NewDocumentRepository(db)

    // Clean up before test
    database.ExecContext(ctx, "DELETE FROM entity_documents WHERE entity_type = 'task' AND entity_id = 99999")

    // Create test document
    doc, err := docRepo.CreateOrGet(ctx, "Test Doc", "/tmp/test.md")
    require.NoError(t, err)

    // Test
    err = repo.Link(ctx, models.EntityTypeTask, 99999, doc.ID, "general")
    assert.NoError(t, err)

    // Verify
    docs, err := repo.ListForEntity(ctx, models.EntityTypeTask, 99999)
    assert.NoError(t, err)
    assert.Len(t, docs, 1)

    // Cleanup
    defer database.ExecContext(ctx, "DELETE FROM entity_documents WHERE entity_type = 'task' AND entity_id = 99999")
}
```

#### EntityHistoryRepository Tests

**File**: `internal/repository/entity_history_repository_test.go`

| Test | Description | Assertions |
|------|-------------|------------|
| `TestEntityHistoryRepo_Create` | Create history record for each entity type | Row created; ID assigned; all fields stored correctly |
| `TestEntityHistoryRepo_ListByEntity` | List history for specific entity | Returns records in changed_at DESC order; filtered by entity_type + entity_id |
| `TestEntityHistoryRepo_ListRecent` | List most recent history across all types | Returns N most recent records; respects limit |
| `TestEntityHistoryRepo_ListWithFilters` | Filter by entity_type, changed_by, since, status | Each filter works independently and in combination |
| `TestEntityHistoryRepo_Create_Validation` | Create with empty to_status | Returns validation error |
| `TestEntityHistoryRepo_Create_InvalidType` | Create with invalid entity_type | Returns validation error |

#### Migration Tests

**File**: `internal/db/migrate_test.go` (extend existing)

| Test | Description | Assertions |
|------|-------------|------------|
| `TestMigratePolymorphicTables_FreshDB` | Run on empty database (no old tables) | New tables created; no errors; no data to migrate |
| `TestMigratePolymorphicTables_WithData` | Run with existing data in old tables | Data copied correctly; row counts match; old tables dropped |
| `TestMigratePolymorphicTables_Idempotent` | Run twice | Second run is no-op; no errors; no duplicate data |
| `TestMigratePolymorphicTables_LinkTypePreserved` | task_documents with link_type values | link_type column values preserved in entity_documents |
| `TestMigratePolymorphicTables_HistoryFieldMapping` | task_history field mapping | old_status->from_status, agent->changed_by, timestamp->changed_at all correct |

### 1.2 Service Tests (Mocked Repositories)

Per project testing architecture: service tests use mocked repositories, NEVER real database.

#### EntityService History Recording Tests

**File**: `internal/services/entity_service_test.go` (extend existing)

| Test | Description | Assertions |
|------|-------------|------------|
| `TestEntityService_TransitionStatus_CreatesHistory` | Transition with history repo set | `historyRepo.Create` called with correct EntityType, FromStatus, ToStatus, ChangedBy |
| `TestEntityService_TransitionStatus_NoHistoryRepo` | Transition with nil history repo | No panic; transition succeeds without history record |
| `TestEntityService_TransitionStatus_HistoryError` | History recording fails | Transition still succeeds (history is non-blocking); error logged |
| `TestEntityService_TransitionStatus_ForcedHistory` | Forced transition records forced=true | History record has Forced=true |
| `TestEntityService_TransitionStatus_BackwardHistory` | Backward transition records rejection_reason | History record has RejectionReason set |

**Mock pattern**:
```go
type MockEntityHistoryRecorder struct {
    CreateFunc func(ctx context.Context, history *models.EntityHistory) error
    Created    []*models.EntityHistory // captures all created records
}

func (m *MockEntityHistoryRecorder) Create(ctx context.Context, history *models.EntityHistory) error {
    m.Created = append(m.Created, history)
    if m.CreateFunc != nil {
        return m.CreateFunc(ctx, history)
    }
    return nil
}

func TestEntityService_TransitionStatus_CreatesHistory(t *testing.T) {
    mockHistoryRepo := &MockEntityHistoryRecorder{}
    mockEntityRepo := &MockEntityRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
            return &models.Feature{
                BaseEntity: models.BaseEntity{ID: 42, Key: "E21-F07"},
                Status:     models.FeatureStatus("draft"),
            }, nil
        },
        UpdateStatusFunc: func(ctx context.Context, id int64, status string) error {
            return nil
        },
    }

    svc := NewEntityService(mockWorkflow, mockNoteRepo)
    svc.SetHistoryRepo(mockHistoryRepo)

    result, err := svc.TransitionStatus(ctx, mockEntityRepo, "feature", "E21-F07", "active", opts, features)
    require.NoError(t, err)

    // Verify history was recorded
    require.Len(t, mockHistoryRepo.Created, 1)
    h := mockHistoryRepo.Created[0]
    assert.Equal(t, models.EntityTypeFeature, h.EntityType)
    assert.Equal(t, int64(42), h.EntityID)
    assert.Equal(t, "draft", *h.FromStatus)
    assert.Equal(t, "active", h.ToStatus)
}
```

#### EntityDocumentService Tests (Simplified)

**File**: `internal/services/entity_document_service_test.go` (rewrite)

After refactoring, `EntityDocumentService` uses direct repository calls instead of callbacks. Tests simplify accordingly.

| Test | Description | Assertions |
|------|-------------|------------|
| `TestEntityDocumentService_LinkDocument` | Link document to entity | `docRepo.CreateOrGet` called; `linkRepo.Link` called with correct params |
| `TestEntityDocumentService_UnlinkDocument` | Unlink document from entity | `docRepo.GetByTitle` called; `linkRepo.Unlink` called |
| `TestEntityDocumentService_ListDocuments` | List documents for entity | `linkRepo.ListForEntity` called; results passed through |
| `TestEntityDocumentService_LinkDocument_EntityNotFound` | Link to non-existent entity | Returns error from entity lookup |

#### EntityHistoryService Tests

**File**: `internal/services/entity_history_service_test.go` (new)

| Test | Description | Assertions |
|------|-------------|------------|
| `TestEntityHistoryService_GetHistory` | Get history for any entity type | Calls `historyRepo.ListByEntity` with correct entity_type + entity_id |
| `TestEntityHistoryService_GetHistory_TaskBackwardCompat` | Get task history via entity history | Same results as old `TaskHistoryService.GetTaskHistory` |
| `TestEntityHistoryService_GetRecentHistory` | Get recent history project-wide | Calls `historyRepo.ListRecent` |

### 1.3 CLI Tests (Mocked Services)

Per project testing architecture: CLI tests use mocked services, NEVER real database or real services.

| Test | Description | Assertions |
|------|-------------|------------|
| `TestStatusHistory_FeatureKey` | `shark status history E21-F07` | Parses feature key; calls EntityHistoryService; formats output |
| `TestStatusHistory_BugKey` | `shark status history B001` | Parses bug key; calls EntityHistoryService |
| `TestRelatedDocs_BugFlag` | `shark related-docs add --bug=B001 --path=x.md` | Parses --bug flag; calls EntityDocumentService |
| `TestRelatedDocs_ChangeFlag` | `shark related-docs list --change=CC-001` | Parses --change flag; calls EntityDocumentService |

---

## 2. Model Validation Tests

**File**: `internal/models/entity_history_test.go` (new)

| Test | Description | Assertions |
|------|-------------|------------|
| `TestEntityHistory_Validate_Valid` | Valid EntityHistory | No error |
| `TestEntityHistory_Validate_EmptyEntityType` | Empty entity_type | Returns error |
| `TestEntityHistory_Validate_InvalidEntityType` | entity_type="invalid" | Returns error |
| `TestEntityHistory_Validate_ZeroEntityID` | entity_id=0 | Returns error |
| `TestEntityHistory_Validate_EmptyToStatus` | Empty to_status | Returns error |
| `TestEntityHistory_Validate_NilFromStatus` | from_status=nil (initial) | No error (allowed for initial status) |

---

## 3. Integration Smoke Tests

These are manual verification steps to be performed after the migration:

| Step | Command | Expected Result |
|------|---------|-----------------|
| 1 | `shark related-docs list --epic=E21` | Returns same documents as before migration |
| 2 | `shark related-docs list --feature=E21-F01` | Returns same documents as before migration |
| 3 | `shark related-docs add --bug=B001 --path=docs/test.md` | Document linked to bug (was previously impossible) |
| 4 | `shark related-docs list --bug=B001` | Shows the linked document |
| 5 | `shark status advance E21-F08-001` | Advances status AND creates entity_history record |
| 6 | `shark status history E21-F08-001` | Shows the status transition |
| 7 | `shark status history E21-F07` | Shows feature status changes (was previously impossible) |
| 8 | `shark status history B001` | Shows bug status changes (was previously impossible) |

---

## 4. Test Coverage Targets

| Component | Target | Rationale |
|-----------|--------|-----------|
| `EntityDocumentRepository` | 90%+ | Critical data access; covers all entity types |
| `EntityHistoryRepository` | 90%+ | Critical data access; covers all entity types |
| `EntityHistory.Validate()` | 100% | Simple validation; all branches testable |
| `EntityService.TransitionStatus` (history step) | 100% | Critical business logic; nil-safe and error paths |
| `EntityDocumentService` (simplified) | 80%+ | Simpler after callback removal |
| `EntityHistoryService` | 80%+ | Query-only service |
| Migration function | 80%+ | Complex but idempotent; key paths are migration + verification |

---

## 5. Regression Risk Matrix

| Existing Feature | Risk Level | Test Coverage |
|-----------------|------------|---------------|
| `shark related-docs list --epic=E01` | Medium | Existing tests + new repo tests |
| `shark related-docs add --task=E01-F01-001 --path=x.md` | Medium | Existing tests + new repo tests |
| `shark status history E01-F01-001` (task history) | High | New delegation tests via TaskHistoryService |
| Task status transitions creating history | Low | Existing trigger + new EntityService recording |
| `shark task sessions` (work session analytics) | Medium | Verify TaskHistoryService delegation works |

The highest regression risk is the task history delegation path: `TaskHistoryService` -> `EntityHistoryRepository` with `entity_type='task'` filter. This must produce identical results to the old `TaskHistoryRepository` -> `task_history` table path.

---

*Last Updated*: 2026-03-20
