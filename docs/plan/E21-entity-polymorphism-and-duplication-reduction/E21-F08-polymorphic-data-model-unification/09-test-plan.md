# F08: Polymorphic Data Model Unification -- Test Plan

**Feature**: E21-F08
**Tier**: COMPLEX
**Author**: QA Agent
**Date**: 2026-03-20

This is the comprehensive test plan for unifying document associations and history tracking into polymorphic tables. F08 is a data migration feature with destructive table drops, cross-cutting service hooks, and new CLI capabilities. The testing strategy is dominated by **migration correctness** (zero data loss), **backward compatibility** (existing commands produce identical output), and **new capability verification** (bugs/change-cards get document linking; all entity types get history).

---

## 1. Epic UAT Scenario Decomposition

### Mapping F08 to Epic UAT Scenarios

F08 contributes to epic-level acceptance by extending polymorphic patterns to the data layer (documents and history), directly enabling the extensibility promise validated in UAT Scenarios 1 and 3.

| Epic UAT Scenario | F08 Contribution | F08 Validation Method |
|---|---|---|
| **Scenario 1: Add New Entity Type** (Journey 1) | After F08, adding a 6th entity type requires zero schema changes for documents and history -- they automatically work via the polymorphic `entity_documents` and `entity_history` tables | Verify: a new entity_type value in `entity_documents` and `entity_history` requires zero DDL. Existing `EntityDocumentRepository.Link()` and `EntityHistoryRepository.Create()` accept the new type without code changes. |
| **Scenario 2: Fix Cross-Cutting Bug** (Journey 2) | History recording is centralized in `EntityService.TransitionStatus`. A bug in history recording is fixed in 1 file, affecting all 5 entity types. | Verify: `EntityService.TransitionStatus` creates `entity_history` records for all 5 entity types via a single code path. Mock tests confirm single-point-of-fix. |
| **Scenario 3: Add Cross-Cutting Feature** (Journey 3) | Document linking and history querying are polymorphic. Adding a new document link type or history filter applies to all entity types simultaneously. | Verify: `EntityDocumentRepository` and `EntityHistoryRepository` accept any valid `EntityType` without per-entity branching. |
| **UAT Metric 4: Test Pass Rate** | F08 must achieve 100% existing test pass rate at every phase boundary (Schema, Service, CLI). | `make fmt && make lint && make test` after each phase. |
| **P0: Zero Behavioral Regression** | Existing `shark related-docs list --epic=E01` and `shark status history E01-F01-001` must produce identical output after migration. | Smoke tests comparing pre/post migration output for all entity types with existing data. |

### F08-Specific UAT Verification Scenarios

**UAT-F08-01: Document Linking for Bugs (New Capability)**

- **Given**: Bug B001 exists, `entity_documents` table exists
- **When**: `shark related-docs add --bug=B001 --path=docs/analysis.md`
- **Then**: Document is linked; `shark related-docs list --bug=B001` shows the document
- **Validation**: CLI integration smoke test

**UAT-F08-02: Status History for Features (New Capability)**

- **Given**: Feature E21-F07 exists with at least one status transition recorded in `entity_history`
- **When**: `shark status history E21-F07`
- **Then**: Shows feature status change records with timestamp, from_status, to_status
- **Validation**: CLI integration smoke test

**UAT-F08-03: Migration Data Preservation**

- **Given**: Existing database with data in `epic_documents`, `feature_documents`, `task_documents`, `task_history`
- **When**: Migration runs (schema version 6 -> 7)
- **Then**: All document links preserved in `entity_documents`; all task history preserved in `entity_history`; old tables dropped; row counts verified
- **Validation**: Migration repository tests with row count assertions

**UAT-F08-04: Task History Backward Compatibility**

- **Given**: `task_history` data migrated to `entity_history` with entity_type='task'
- **When**: `shark status history E01-F01-001` (task key)
- **Then**: Output identical to pre-migration output (same records, same ordering)
- **Validation**: TaskHistoryService delegation test comparing old and new paths

---

## 2. Component Test Strategy

### 2.1 EntityHistory Model (`internal/models/entity_history.go`)

**Purpose**: Validate the new EntityHistory struct's structural validation in isolation.

**New tests** (`internal/models/entity_history_test.go`):

| Test Name | Purpose | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| `TestEntityHistory_Validate_Valid` | Valid record passes validation | All fields populated, valid entity_type | No error | |
| `TestEntityHistory_Validate_AllEntityTypes` | Each of 5 entity types passes | EntityType = epic/feature/task/bug/change | No error for each | |
| `TestEntityHistory_Validate_EmptyEntityType` | Empty entity_type rejected | EntityType = "" | Error: "entity_type cannot be empty" | |
| `TestEntityHistory_Validate_InvalidEntityType` | Unknown entity_type rejected | EntityType = "milestone" | Error: "invalid entity_type" | |
| `TestEntityHistory_Validate_ZeroEntityID` | Zero entity_id rejected | EntityID = 0 | Error: "entity_id must be positive" | Negative ID (-1) also rejected |
| `TestEntityHistory_Validate_EmptyToStatus` | Empty to_status rejected | ToStatus = "" | Error: "to_status cannot be empty" | |
| `TestEntityHistory_Validate_NilFromStatus` | Nil from_status accepted (initial) | FromStatus = nil | No error | Represents initial status assignment |

**Estimated new test lines**: ~80

### 2.2 EntityDocumentRepository (`internal/repository/entity_document_repository.go`)

**Purpose**: Validate polymorphic document linking CRUD against real database. Follows `entity_note_repository_test.go` pattern.

**New tests** (`internal/repository/entity_document_repository_test.go`):

| Test Name | Purpose | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| `TestEntityDocumentRepo_Link` | Link document to each entity type | entity_type=task, entity_id=99999, document_id=valid | Row created in entity_documents | |
| `TestEntityDocumentRepo_LinkIdempotent` | Re-linking same document is no-op | Same link params twice | INSERT OR IGNORE; no error; 1 row | Tests UNIQUE constraint |
| `TestEntityDocumentRepo_Unlink` | Remove document link | Previously linked doc | Row removed; ListForEntity returns empty | |
| `TestEntityDocumentRepo_UnlinkNonExistent` | Unlink non-existent link is no-op | entity_id with no links | No error; 0 rows affected | |
| `TestEntityDocumentRepo_ListForEntity` | List documents for entity | 3 documents linked to task | Returns 3 documents, ordered by created_at DESC | Empty list for entity with no links |
| `TestEntityDocumentRepo_ListForEntity_MultipleTypes` | Different entity types don't leak | Doc linked to task and epic | ListForEntity(task) returns only task's docs | Cross-type isolation |
| `TestEntityDocumentRepo_LinkWithType` | Link with specific link_type | link_type="specification" | link_type preserved and queryable | Default "general" when empty |
| `TestEntityDocumentRepo_CascadeDelete` | FK cascade on document delete | Delete parent document | entity_documents rows cascade-deleted | |
| `TestEntityDocumentRepo_AllEntityTypes` | Parameterized: link works for each of 5 entity types | entity_type = epic/feature/task/bug/change | Link + List round-trip succeeds for each | |

**Estimated new test lines**: ~200

### 2.3 EntityHistoryRepository (`internal/repository/entity_history_repository.go`)

**Purpose**: Validate polymorphic history CRUD and queries against real database.

**New tests** (`internal/repository/entity_history_repository_test.go`):

| Test Name | Purpose | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| `TestEntityHistoryRepo_Create` | Create history record | Valid EntityHistory for feature | Row created; ID assigned; all fields stored | |
| `TestEntityHistoryRepo_Create_AllEntityTypes` | Parameterized creation for each type | entity_type = epic/feature/task/bug/change | ID assigned for each | |
| `TestEntityHistoryRepo_Create_Validation` | Empty to_status rejected | ToStatus = "" | Returns validation error before DB insert | |
| `TestEntityHistoryRepo_Create_InvalidType` | Invalid entity_type rejected | EntityType = "invalid" | Returns validation error | |
| `TestEntityHistoryRepo_ListByEntity` | List history for specific entity | 5 records for feature, 3 for task | ListByEntity(feature) returns 5, ordered by changed_at DESC | Empty list for entity with no history |
| `TestEntityHistoryRepo_ListRecent` | List N most recent across all types | 10 records across 3 types, limit=5 | Returns 5 most recent regardless of type | limit=0 returns default |
| `TestEntityHistoryRepo_ListWithFilters_EntityType` | Filter by entity_type | records for task and feature; filter by task | Returns only task records | |
| `TestEntityHistoryRepo_ListWithFilters_ChangedBy` | Filter by agent | Records with agent="dev" and "qa" | Filter by "dev" returns only dev records | |
| `TestEntityHistoryRepo_ListWithFilters_Since` | Filter by timestamp | Records before and after cutoff | Returns only records after cutoff | |
| `TestEntityHistoryRepo_ListWithFilters_Combined` | Multiple filters together | entity_type + changed_by + since | Intersection of all filters | |
| `TestEntityHistoryRepo_ForcedAndRejection` | Forced flag and rejection_reason stored | forced=true, rejection_reason="Emergency" | Both fields round-trip correctly | forced=false default |

**Estimated new test lines**: ~250

### 2.4 Migration Function (`internal/db/migrate.go`)

**Purpose**: Validate data migration correctness, idempotency, and row count verification.

**New tests** (`internal/db/migrate_test.go` -- extend existing):

| Test Name | Purpose | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| `TestMigratePolymorphicTables_FreshDB` | Empty database (no old tables) | New database with no data | New tables created; no errors; no data to migrate | |
| `TestMigratePolymorphicTables_WithData` | Full migration with existing data | 3 epic_documents, 2 feature_documents, 5 task_documents, 10 task_history rows | Row counts match: epic=3, feature=2, task=5, history=10; old tables dropped | |
| `TestMigratePolymorphicTables_Idempotent` | Run migration twice | Pre-populated old tables, run twice | Second run is no-op; no duplicate data; no errors | |
| `TestMigratePolymorphicTables_LinkTypePreserved` | task_documents.link_type values preserved | task_documents with link_type="specification" | entity_documents row has link_type="specification" | NULL link_type becomes "general" |
| `TestMigratePolymorphicTables_HistoryFieldMapping` | task_history field mapping correct | task_history with old_status, new_status, agent, timestamp, forced, rejection_reason | entity_history: from_status=old_status, to_status=new_status, changed_by=agent, changed_at=timestamp, forced=COALESCE(forced,0), rejection_reason preserved | NULL forced becomes 0 |
| `TestMigratePolymorphicTables_PartialRun` | Migration interrupted after table creation but before drop | entity_documents exists AND epic_documents exists | Re-run copies remaining data (INSERT OR IGNORE prevents duplicates), verifies, drops old | |
| `TestMigratePolymorphicTables_BugAndChangeDocuments` | Bug and change_card document migration | bug_documents with 2 rows, change_card_documents with 1 row | entity_documents: bug=2, change=1 | These may be empty in test DB |
| `TestMigratePolymorphicTables_VerificationFailure` | Row count mismatch aborts drop | Simulate: old table has 5 rows but new table has 3 | Error returned; old tables NOT dropped | Tests safety mechanism |

**Estimated new test lines**: ~300

### 2.5 Existing Test Suites (Must Pass Without Modification)

| Test Suite | Files | What It Validates for F08 |
|---|---|---|
| Document repository CRUD | `document_repository_test.go` | `CreateOrGet`, `GetByID`, `GetByTitle`, `Delete` still work after entity-specific methods removed |
| Task history service | `task_history_service_test.go` | `GetTaskHistory`, `ListHistory`, `GetWorkSessions` produce same results via delegation |
| Entity document service | `entity_document_service_test.go` | Document operations for epic/feature/task continue working after callback removal |
| Related-docs CLI | Commands tests | `shark related-docs list --epic=E01` still works |
| Status history CLI | Commands tests | `shark status history E01-F01-001` still works for tasks |
| All entity services | `*_service_test.go` | TransitionStatus for all entity types continues working |

---

## 3. API Contract Test Cases

F08 introduces no new HTTP API endpoints. The "API contracts" are:

1. **CLI Command Contracts** -- existing commands produce identical output; new flags work
2. **Repository Interface Contracts** -- new polymorphic repositories satisfy their interfaces
3. **Service Interface Contracts** -- EntityService gains optional EntityHistoryRecorder dependency

### 3.1 CLI Command Contracts

| Command | Contract | Before F08 | After F08 | Validation |
|---|---|---|---|---|
| `shark related-docs list --epic=E01` | Returns same documents | Queries `epic_documents` | Queries `entity_documents WHERE entity_type='epic'` | Integration smoke test |
| `shark related-docs list --feature=E21-F01` | Returns same documents | Queries `feature_documents` | Queries `entity_documents WHERE entity_type='feature'` | Integration smoke test |
| `shark related-docs list --task=E01-F01-001` | Returns same documents | Queries `task_documents` | Queries `entity_documents WHERE entity_type='task'` | Integration smoke test |
| `shark related-docs add --bug=B001 --path=x.md` | **New**: links doc to bug | Not supported (error) | Links via `entity_documents` with entity_type='bug' | New CLI test |
| `shark related-docs list --bug=B001` | **New**: lists bug documents | Not supported (error) | Queries `entity_documents WHERE entity_type='bug'` | New CLI test |
| `shark related-docs add --change=CC-001 --path=x.md` | **New**: links doc to change-card | Not supported (error) | Links via `entity_documents` with entity_type='change' | New CLI test |
| `shark status history E01-F01-001` | Returns same task history | Queries `task_history` | Queries `entity_history WHERE entity_type='task'` | Integration smoke test |
| `shark status history E21-F07` | **New**: returns feature history | Not supported (empty/error) | Queries `entity_history WHERE entity_type='feature'` | New CLI test |
| `shark status history E21` | **New**: returns epic history | Not supported (empty/error) | Queries `entity_history WHERE entity_type='epic'` | New CLI test |
| `shark status history B001` | **New**: returns bug history | Not supported (empty/error) | Queries `entity_history WHERE entity_type='bug'` | New CLI test |

### 3.2 Repository Interface Contracts

| Interface | Method | Input | Output | Error Cases |
|---|---|---|---|---|
| `EntityDocumentLinkRepository` | `Link(ctx, entityType, entityID, documentID, linkType)` | Valid entity_type, positive IDs | nil error; row in entity_documents | Invalid entity_type: validation error; nonexistent document_id: FK error |
| `EntityDocumentLinkRepository` | `Unlink(ctx, entityType, entityID, documentID)` | Valid entity_type, positive IDs | nil error; row removed | Non-existent link: no-op (no error) |
| `EntityDocumentLinkRepository` | `ListForEntity(ctx, entityType, entityID)` | Valid entity_type, positive ID | `[]*models.Document` ordered by created_at DESC | No links: empty slice (not nil) |
| `EntityHistoryRecorder` | `Create(ctx, *EntityHistory)` | Valid EntityHistory | nil error; ID assigned | Validation failure: error before DB write |
| `EntityHistoryQuerier` | `ListByEntity(ctx, entityType, entityID)` | Valid params | `[]*EntityHistory` ordered by changed_at DESC | No history: empty slice |
| `EntityHistoryQuerier` | `ListRecent(ctx, limit)` | limit > 0 | N most recent records across all types | limit=0: default to 50 |
| `EntityHistoryQuerier` | `ListWithFilters(ctx, filters)` | Any combination of filter fields | Filtered results; AND semantics for combined filters | All filters nil: returns all (with limit) |

### 3.3 EntityService History Recording Contract

| Aspect | Contract | Validation |
|---|---|---|
| **Injection** | `EntityHistoryRecorder` is an optional dependency on `EntityService` (nil-safe) | `TestEntityService_TransitionStatus_NoHistoryRepo` |
| **Recording point** | After `UpdateStatus` succeeds and before rejection note creation | Service test verifying call order |
| **Fields captured** | EntityType from entity, EntityID from entity, FromStatus from entity.GetStatus(), ToStatus from target, ChangedBy from opts.Agent, Notes from opts.Reason, Forced from opts.Force | `TestEntityService_TransitionStatus_CreatesHistory` |
| **Non-blocking** | If history recording fails, the transition still succeeds | `TestEntityService_TransitionStatus_HistoryError` |

---

## 4. Integration Scenarios

### 4.1 Phase-Boundary Integration (3-phase sequential verification)

Each phase must pass the full quality gate before the next begins.

| Phase | Changes | Integration Verification | Risk Level |
|---|---|---|---|
| **Phase 1: Schema + Repos** | New tables, migration function, EntityDocumentRepository, EntityHistoryRepository | `make fmt && make lint && make test`; manual `sqlite3 shark-tasks.db .schema` to verify new tables | High (destructive migration) |
| **Phase 2: Services** | Simplified EntityDocumentService, history recording in EntityService, EntityHistoryService | `make fmt && make lint && make test`; verify `shark status advance` creates entity_history record | Medium |
| **Phase 3: CLI + Cleanup** | New --bug/--change flags, entity history for all types, remove old repo methods | `make fmt && make lint && make test`; full smoke test of all new and existing CLI commands | Medium |

### 4.2 Cross-Feature Integration within E21

| Sibling Feature | Integration Point | F08 Verification |
|---|---|---|
| **E21-F01** (Entity Interface, COMPLETED) | F08 uses `Entity.GetEntityType()` and `Entity.GetID()` for polymorphic columns. F01 provides these via the Entity interface. | Compile-time: `entity.GetEntityType()` returns valid `EntityType` used in repository calls. Covered by existing entity_test.go. |
| **E21-F03** (Status Transition Unification, COMPLETED) | F08 adds history recording step into `EntityService.TransitionStatus` (the unified method that F03 created). F03's delegation pattern means history recording automatically applies to all 5 entity types. | Service test: `TestEntityService_TransitionStatus_CreatesHistory` verifies history is created for any entity type passed through TransitionStatus. |
| **E21-F07** (BaseEntity, ready_for_task_generation) | F07 may change entity model internals (struct embedding). F08 uses Entity interface methods only, not struct fields. No conflict. | If F07 merges before F08: run full test suite. If F08 merges before F07: no impact (F08 only adds tables and service hooks). |
| **E21-F09** (Service Delegation, not started) | F09 wires entity-specific services to delegate to EntityService. F08's history recording in EntityService means F09 automatically gets history. F08 should merge before F09. | After F09 merges: verify that entity-specific `TransitionStatus` calls still produce history records (inherited from EntityService). |
| **E21-F11** (Polymorphic Relationships, not started) | F11 follows the same polymorphic table pattern (`entity_type + entity_id`). F08 proves the migration approach. | F08 completion validates the pattern. F11 can reuse migration template. |
| **E21-F12** (Polymorphic Acceptance Criteria, not started) | Same pattern as F11. F08 establishes precedent. | No verification needed now; F12 consumes F08's patterns. |

### 4.3 Cross-Epic Integration

| Epic | Overlap | Verification |
|---|---|---|
| **E15** (Service Layer Architecture) | E15 migrates CLI commands to use services. F08 changes service internals (EntityDocumentService simplification, EntityService history hook). E15-migrated commands must still work. | `make test` covers all E15-migrated commands. Specific attention to `related-docs` and `status history` commands if they were migrated by E15. |
| **E19** (Sprint Management) | E19 would add a Sprint entity type. After F08, Sprint automatically gets document and history support via polymorphic tables (zero schema work). | Validated by UAT-F08-01 concept: any new entity_type value works in entity_documents and entity_history without DDL. |

### 4.4 Data Migration Integration

This is the highest-risk integration scenario in F08.

| Scenario | Precondition | Action | Expected Outcome | Risk |
|---|---|---|---|---|
| **Production-like migration** | Database with real epic/feature/task documents and task_history records | Run migration (schema v6->v7) | All data preserved; old tables dropped; `shark related-docs list` produces identical output for all entities | HIGH |
| **Empty database migration** | Fresh database, no old tables | Run migration | New tables created; no errors; no data to migrate | LOW |
| **Database with no documents** | Existing database but empty document tables | Run migration | Tables created and dropped; zero rows migrated; no errors | LOW |
| **Database with link_type data** | task_documents rows with non-null link_type | Run migration | link_type values preserved in entity_documents | MEDIUM |
| **Database with NULL forced in task_history** | task_history rows where forced is NULL | Run migration | COALESCE(forced, 0) normalizes to 0 in entity_history | MEDIUM |

---

## 5. Acceptance Criteria Test Matrix

### Story 1: Document Linking for Any Entity Type

| AC | TC ID | Test Method | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|---|
| `entity_documents` table created with entity_type, entity_id, document_id columns | TC-S1-01 | Migration test: `TestMigratePolymorphicTables_FreshDB` | Fresh database | Table exists with all required columns and indexes | |
| Document linking works for all 5 entity types via same CLI commands | TC-S1-02 | Repository test: `TestEntityDocumentRepo_AllEntityTypes` | link for epic/feature/task/bug/change | Link + ListForEntity round-trip succeeds for each type | |
| Existing epic/feature/task document links preserved after migration | TC-S1-03 | Migration test: `TestMigratePolymorphicTables_WithData` | Pre-populated old document tables | Row counts match; data accessible via `ListForEntity` | Empty old tables (0 rows) |
| `shark related-docs add --bug=B001 --path=docs/analysis.md` works | TC-S1-04 | CLI smoke test | Bug key, document path | Document linked; `related-docs list --bug=B001` shows it | Bug does not exist: error |
| `shark related-docs add --change=CC-001 --path=x.md` works | TC-S1-05 | CLI smoke test | Change-card key, document path | Document linked; `related-docs list --change=CC-001` shows it | |

### Story 2: Status Change History for All Entity Types

| AC | TC ID | Test Method | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|---|
| `entity_history` table created with required columns | TC-S2-01 | Migration test: `TestMigratePolymorphicTables_FreshDB` | Fresh database | Table exists with: entity_type, entity_id, from_status, to_status, changed_by, notes, forced, rejection_reason, changed_at | |
| All status transitions create history records for all entity types | TC-S2-02 | Service test: `TestEntityService_TransitionStatus_CreatesHistory` | TransitionStatus called for each entity type | EntityHistory record created with correct EntityType, FromStatus, ToStatus | Initial status (nil from_status) |
| Existing task_history data preserved after migration | TC-S2-03 | Migration test: `TestMigratePolymorphicTables_HistoryFieldMapping` | Pre-populated task_history | All fields mapped correctly: old_status->from_status, agent->changed_by, timestamp->changed_at | NULL forced -> 0 |
| `shark status history E21-F07` returns feature status changes | TC-S2-04 | CLI smoke test + EntityHistoryService test | Feature key with history records | Returns history ordered by changed_at DESC | Feature with no history: empty output |
| `shark status history B001` returns bug status changes | TC-S2-05 | CLI smoke test | Bug key with history records | Returns history records | |
| Forced transitions recorded with forced=true | TC-S2-06 | Service test: `TestEntityService_TransitionStatus_ForcedHistory` | TransitionStatus with opts.Force=true | EntityHistory.Forced=true, RejectionReason populated | |
| Backward transitions record rejection_reason | TC-S2-07 | Service test: `TestEntityService_TransitionStatus_BackwardHistory` | Backward transition with reason | EntityHistory.RejectionReason populated | |

### Story 3: Old Per-Entity Tables Removed

| AC | TC ID | Test Method | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|---|
| Data migrated from 5 document tables to entity_documents | TC-S3-01 | Migration test: `TestMigratePolymorphicTables_WithData` | Data in all 5 old tables | Row counts: epic_old=epic_new, feature_old=feature_new, etc. | Tables with 0 rows |
| Old tables dropped after migration verification | TC-S3-02 | Migration test: `TestMigratePolymorphicTables_WithData` | Successful migration | `tableExists("epic_documents")` returns false | |
| Repository code updated to single entity_documents table | TC-S3-03 | Code inspection + compile test | `go build ./...` | No references to `epic_documents`, `feature_documents`, `task_documents` in repository code | |
| CLI commands produce identical output post-migration | TC-S3-04 | Integration smoke tests | `shark related-docs list --epic=E01` pre and post migration | Identical document list output | |
| Verification failure aborts table drop | TC-S3-05 | Migration test: `TestMigratePolymorphicTables_VerificationFailure` | Simulated row count mismatch | Error returned; old tables NOT dropped | |

### Story 4: Changed_by Captures Agent Identity

| AC | TC ID | Test Method | Input | Expected Outcome | Edge Cases |
|---|---|---|---|---|---|
| `changed_by` stores agent ID for agent-initiated changes | TC-S4-01 | Service test | TransitionStatus with opts.Agent="dev-agent" | EntityHistory.ChangedBy = "dev-agent" | |
| `changed_by` stores "user" for manual changes | TC-S4-02 | Service test | TransitionStatus with opts.Agent="" or nil | EntityHistory.ChangedBy = nil (nullable) | Default handling |
| Transition options propagate agent identity | TC-S4-03 | Service test | Full TransitionStatus flow with agent | ChangedBy field populated from opts through to EntityHistory | |

### Requirement-Level Traceability

| Requirement | ACs Covered | Test Cases | Priority |
|---|---|---|---|
| REQ-F-001: Polymorphic Entity Documents Table | TC-S1-01, TC-S1-02, TC-S1-04, TC-S1-05 | EntityDocumentRepo tests, CLI smoke tests | Must-Have |
| REQ-F-002: Polymorphic Entity History Table | TC-S2-01, TC-S2-02, TC-S2-04, TC-S2-05 | EntityHistoryRepo tests, service tests, CLI smoke tests | Must-Have |
| REQ-F-003: Data Migration | TC-S1-03, TC-S2-03, TC-S3-01, TC-S3-02, TC-S3-04, TC-S3-05 | Migration tests, smoke tests | Must-Have |
| REQ-F-004: Repository Consolidation | TC-S3-03 | Compile test, code inspection | Must-Have |
| REQ-F-005: History Recording in TransitionStatus | TC-S2-02, TC-S2-06, TC-S2-07 | EntityService tests | Must-Have |
| REQ-NF-001: Migration Safety | TC-S3-05 | Verification failure test | Must-Have |
| REQ-NF-002: Query Performance | Informational benchmark (Section 6) | Index verification | Should-Have |

---

## 6. Performance, Security, and Non-Functional Approach

### 6.1 Performance

**Primary concern**: Polymorphic table queries should not degrade compared to per-entity tables.

| Benchmark | Location | Target | Approach |
|---|---|---|---|
| Document lookup by entity_type + entity_id | EntityDocumentRepository | < 5ms for typical dataset (<100k rows) | Composite index `(entity_type, entity_id)` ensures O(log n); same performance as per-entity tables with single-column index |
| History lookup by entity_type + entity_id | EntityHistoryRepository | < 5ms for typical dataset | Three indexes: lookup, time, entity_time |
| Migration execution time | Migration function | < 10s for databases with <10k total rows | Single-pass INSERT INTO ... SELECT for each table |

**Verification approach**: Informational -- not blocking. If performance is measurably worse on typical datasets, investigate index coverage.

**Index verification test** (add to migration tests):

```sql
-- Verify indexes exist after migration
SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='entity_documents';
-- Expected: idx_entity_documents_lookup, idx_entity_documents_document
SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='entity_history';
-- Expected: idx_entity_history_lookup, idx_entity_history_time, idx_entity_history_entity_time
```

### 6.2 Security

F08 has **minimal security surface**. Specific considerations:

| Concern | Risk | Mitigation |
|---|---|---|
| SQL injection via entity_type | Low | entity_type validated against `models.ValidEntityTypes` before any SQL execution |
| Data integrity (polymorphic FK) | Low | Application-level validation in repository (same approach as `entity_notes`); no new attack surface |
| Migration data exposure | None | Migration operates on local database; no network operations |

No dedicated security testing required beyond the input validation tests in repository test suites.

### 6.3 Data Integrity

This is the most critical non-functional concern for F08.

| Check | When | How | Failure Action |
|---|---|---|---|
| Row count verification | During migration | `verifyMigration()` compares old table counts vs new table counts per entity_type | Abort migration; do NOT drop old tables |
| link_type preservation | During migration | Migration test `TestMigratePolymorphicTables_LinkTypePreserved` | Fix migration SQL |
| History field mapping | During migration | Migration test `TestMigratePolymorphicTables_HistoryFieldMapping` | Fix column mapping in INSERT SELECT |
| UNIQUE constraint on entity_documents | Ongoing | `(entity_type, entity_id, document_id)` prevents duplicate links | INSERT OR IGNORE handles gracefully |
| Append-only history | Ongoing | EntityHistory records are never updated or deleted by the application | Code review: no UPDATE/DELETE on entity_history |

### 6.4 Backward Compatibility

| Aspect | Before F08 | After F08 | Validation |
|---|---|---|---|
| `shark related-docs list --epic=E01` | Queries `epic_documents` | Queries `entity_documents WHERE entity_type='epic'` | Smoke test: identical output |
| `shark related-docs list --task=E01-F01-001` | Queries `task_documents` | Queries `entity_documents WHERE entity_type='task'` | Smoke test: identical output |
| `shark status history E01-F01-001` | Queries `task_history` | Queries `entity_history WHERE entity_type='task'` | Smoke test: identical output |
| `TaskHistoryService.GetTaskHistory()` | Direct `task_history` query | Delegates to EntityHistoryRepository with filter | Service delegation test |
| `TaskHistoryService.GetWorkSessions()` | Task-specific work session analytics | Remains task-specific (unchanged logic, new backing table) | Existing service tests must pass |
| `shark task sessions` | Work session analytics command | Same behavior, different backing table | Existing CLI tests must pass |

### 6.5 Quantitative Verification

| Metric | Before | After | How to Measure |
|---|---|---|---|
| Document join tables | 5 | 1 | `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE '%_documents'` |
| History tables | 1 (task-only) | 1 (all entities) | Schema inspection |
| Entity types with document support | 5/5 (via 5 tables) | 5/5 (via 1 table) | `shark related-docs add` for each type |
| Entity types with history support | 1/5 (task only) | 5/5 (all) | `shark status history` for each type |
| Per-entity document repo methods | 15 | 3 | `grep -c 'func.*EntityDocumentRepository' entity_document_repository.go` |
| document_repository.go lines | 449 | ~150 (core CRUD) | `wc -l` |
| Code to add new entity's document support | ~60 lines | 0 lines | Manual assessment |

---

## Integration Smoke Test Checklist

Run after each phase boundary and at feature completion:

```
Phase 1 (Schema + Repos):
[ ] make fmt -- zero formatting changes
[ ] make lint -- zero lint warnings
[ ] make test -- 100% pass rate
[ ] sqlite3 shark-tasks.db ".tables" -- entity_documents and entity_history exist
[ ] sqlite3 shark-tasks.db ".tables" -- epic_documents, feature_documents, task_documents, task_history do NOT exist
[ ] sqlite3 shark-tasks.db "SELECT COUNT(*) FROM entity_documents" -- matches pre-migration total

Phase 2 (Services):
[ ] make fmt && make lint && make test
[ ] shark status advance <task-key> -- verify entity_history record created
[ ] sqlite3 shark-tasks.db "SELECT * FROM entity_history ORDER BY changed_at DESC LIMIT 5" -- shows recent transitions

Phase 3 (CLI + Cleanup):
[ ] make fmt && make lint && make test
[ ] shark related-docs list --epic=E21 -- returns documents (existing)
[ ] shark related-docs list --feature=E21-F01 -- returns documents (existing)
[ ] shark related-docs add --bug=<key> --path=docs/test.md -- succeeds (new capability)
[ ] shark related-docs list --bug=<key> -- shows linked document (new capability)
[ ] shark status history <task-key> -- shows task history (backward compat)
[ ] shark status history <feature-key> -- shows feature history (new capability)
[ ] shark status history <bug-key> -- shows bug history (new capability)
[ ] go build ./... -- no references to dropped tables in compiled code
```

---

## High-Risk Items Requiring Extra Attention

1. **Migration verification logic** -- If the `verifyMigration` function has a bug that incorrectly reports success, old tables get dropped with incomplete data. Test this function independently with simulated mismatches.

2. **TaskHistoryService delegation** -- The task history backward compatibility path (`TaskHistoryService -> EntityHistoryRepository` with `entity_type='task'` filter) must produce results identical to the old `TaskHistoryRepository -> task_history` path. This includes field ordering, NULL handling, and work session analytics.

3. **History recording non-blocking behavior** -- If `EntityHistoryRecorder.Create()` fails, the status transition MUST still succeed. A test must verify that history recording errors are swallowed (logged, not propagated) and do not roll back the status update.

4. **INSERT OR IGNORE for documents vs INSERT for history** -- Document migration uses INSERT OR IGNORE (due to UNIQUE constraint handling re-runs). History migration uses plain INSERT (no UNIQUE constraint; re-runs could create duplicates). The migration must check `tableExists(old_table)` before each INSERT to prevent duplicate history on re-runs.

---

## Exit Gate

- [x] Full 6-section plan written (UAT decomposition, component strategy, API contracts, integration scenarios, AC matrix, performance/security)
- [x] Epic UAT traceability documented (Section 1 maps F08 to UAT Scenarios 1, 2, 3, Metric 4, P0)
- [x] All acceptance criteria have test cases with IDs (TC-S1-01 through TC-S4-03)
- [x] Migration safety explicitly tested (verification failure, idempotency, field mapping)
- [x] Cross-feature and cross-epic integration verified (F01, F03, F07, F09, F11, F12, E15, E19)
- [x] Performance approach documented (index verification, informational benchmarks)
- [x] Security approach documented (input validation, same pattern as entity_notes)
- [x] Data integrity approach documented (row count verification, append-only history, UNIQUE constraints)
- [x] Backward compatibility verification plan documented (smoke tests for all existing commands)
- [x] High-risk items explicitly called out with specific test strategies

---

*Traces to: Feature PRD (feature.md) REQ-F-001 through REQ-F-005, REQ-NF-001, REQ-NF-002; Architecture (02-architecture.md) Sections 2-6; Data Models (03-data-models.md); Migration Strategy (04-migration-strategy.md); Testing Strategy (05-testing-strategy.md); Epic UAT Acceptance Plan (uat-acceptance-plan.md) Scenarios 1, 2, 3, Metric 4, P0*
