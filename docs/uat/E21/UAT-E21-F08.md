# UAT Test Guide - Polymorphic Data Model Unification

**Feature:** E21-F08 - Polymorphic Data Model Unification
**Epic:** E21 - Entity Polymorphism and Duplication Reduction
**Generated:** 2026-03-21
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Reduce code duplication and inconsistency across entity types by applying polymorphic patterns to the data model, services, and CLI layers.

**This Feature's Role:** F08 unifies document associations and history tracking into polymorphic tables (`entity_documents`, `entity_history`) available to all entity types, replacing 6 entity-specific tables with 2 polymorphic ones.

**Related Features:**
- E21-F01 (Entity Interface Foundation) - COMPLETED - Provides `Entity.GetEntityType()` used by F08
- E21-F03 (Status Transition Unification) - COMPLETED - Provides `EntityService.TransitionStatus` where F08 hooks history recording
- E21-F06 (Enhancements and Maintenance) - ACTIVE - Ongoing maintenance
- E21-F07 (BaseEntity Struct Embedding) - ready_for_task_generation - Model layer changes, minimal overlap with F08
- E21-F09 (Service Delegation) - draft - Will inherit F08's history recording automatically

**Integration Points:**
- F08 hooks into `EntityService.TransitionStatus` (from F03) to record history for all entity types
- F08 uses `Entity.GetEntityType()` (from F01) for polymorphic column values
- Future features (F11, F12) will follow F08's polymorphic table pattern

---

## Design Intent

**From Feature PRD:**
> The database has inconsistent polymorphism across cross-cutting concerns. Notes use a single polymorphic table, but documents use 5 separate tables, and history is task-only. Adding document support for bugs requires creating a new table and new repository methods — rather than simply registering the entity type.

**From Architecture Doc:**
> Two new tables replace six. Three repository methods replace fifteen. Zero new architectural concepts. Follows the proven `entity_notes` pattern already in the codebase.

**Key Design Decisions:**
- ADR-1: Include `link_type` in `entity_documents` (preserves task document link types)
- ADR-2: New `EntityHistory` model (not extending `TaskHistory` — clean separation)
- ADR-3: History recording centralized in `EntityService.TransitionStatus`
- ADR-4: Old tables dropped in same migration after row count verification

---

## Cross-Feature Integration Tests

### Integration Scenario 1: History Recording via TransitionStatus
**Features:** E21-F03 + E21-F08
**Scenario:** Status transitions for any entity type automatically create history records via the centralized `EntityService.TransitionStatus` hook.

Steps:
1. Advance a task status via `shark status advance`
2. Query `entity_history` for the task
3. Verify history record exists with correct from_status, to_status, entity_type

Expected Result: History record created for every status transition, regardless of entity type.

### Integration Scenario 2: Entity Interface Polymorphic Columns
**Features:** E21-F01 + E21-F08
**Scenario:** `Entity.GetEntityType()` provides the `entity_type` value used in polymorphic table columns.

Steps:
1. Verify each entity type (epic, feature, task, bug, change) has a valid `GetEntityType()` return
2. Verify these values match the `entity_type` values in `entity_documents` and `entity_history`

Expected Result: Entity interface and polymorphic tables use consistent entity_type values.

---

## Epic Acceptance Validation

| Epic AC | Description | Feature Contribution | Status |
|---------|-------------|---------------------|--------|
| Extensibility | Adding a new entity type requires zero schema changes for cross-cutting concerns | F08 ensures new entity types get document and history support automatically via polymorphic tables | [ ] |
| Single-Point-of-Fix | Fixing a cross-cutting bug requires changing 1 file, not 5 | History recording is centralized in EntityService.TransitionStatus; document linking uses single repository | [ ] |
| Consistency | All entity types have access to all features | All 5 types now have document linking and history (was 3/5 for docs, 1/5 for history) | [ ] |

---

## Feature Acceptance Validation

| Feature AC | Description | Status |
|------------|-------------|--------|
| AC-S1-1 | `entity_documents` table created with entity_type, entity_id, document_id, link_type columns | [ ] |
| AC-S1-2 | Document linking works for all 5 entity types via same CLI commands | [ ] |
| AC-S1-3 | Existing epic/feature/task document links preserved after migration | [ ] |
| AC-S2-1 | `entity_history` table created with required columns | [ ] |
| AC-S2-2 | All status transitions create history records for all entity types | [ ] |
| AC-S2-3 | Existing task_history data preserved after migration | [ ] |
| AC-S2-4 | `shark status history <feature-key>` returns feature status changes | [ ] |
| AC-S2-5 | `shark status history <bug-key>` returns bug status changes | [ ] |
| AC-S3-1 | Data migrated from 5 document tables + task_history to polymorphic tables | [ ] |
| AC-S3-2 | Old tables dropped after verification | [ ] |
| AC-S3-3 | Repository code uses single entity_documents table | [ ] |
| AC-S3-4 | CLI commands produce identical output post-migration | [ ] |
| AC-S4-1 | `changed_by` column stores agent ID or nil for manual changes | [ ] |

---

## Test Scenarios

### Scenario 1: Schema Migration and Data Preservation
**Tasks covered:** T-E21-F08-001

**Steps:**
1. Verify `entity_documents` table exists with correct schema (entity_type, entity_id, document_id, link_type, created_at)
2. Verify `entity_history` table exists with correct schema (entity_type, entity_id, from_status, to_status, changed_by, notes, forced, rejection_reason, changed_at)
3. Verify indexes exist on both tables
4. Verify old tables (`epic_documents`, `feature_documents`, `task_documents`, `bug_documents`, `change_card_documents`, `task_history`) are dropped
5. Verify migration is idempotent (runs clean on both fresh and existing databases)

**Success Criteria:**
- [ ] New polymorphic tables created with all required columns
- [ ] Composite indexes on (entity_type, entity_id) exist
- [ ] Migration verification catches row count mismatches
- [ ] Old tables successfully dropped after verification

### Scenario 2: EntityHistory Model and Validation
**Tasks covered:** T-E21-F08-002

**Steps:**
1. Verify EntityHistory struct has all required fields
2. Verify validation rejects empty entity_type, invalid entity_type, zero entity_id, empty to_status
3. Verify validation accepts nil from_status (initial status)
4. Verify all 5 entity types pass validation

**Success Criteria:**
- [ ] Model validation correctly enforces all structural constraints
- [ ] All 5 entity types are accepted

### Scenario 3: Polymorphic Document Linking
**Tasks covered:** T-E21-F08-003, T-E21-F08-005

**Steps:**
1. Verify EntityDocumentRepository.Link() works for all 5 entity types
2. Verify duplicate links are handled gracefully (INSERT OR IGNORE)
3. Verify Unlink() removes associations
4. Verify ListForEntity() returns documents ordered by created_at DESC
5. Verify cross-entity-type isolation (task docs don't leak to epic queries)
6. Verify EntityDocumentService simplified to use single polymorphic repository

**Success Criteria:**
- [ ] Link/Unlink/ListForEntity round-trip works for all entity types
- [ ] UNIQUE constraint prevents duplicate links
- [ ] Entity type isolation maintained
- [ ] Service simplified from callback pattern to direct repository calls

### Scenario 4: Polymorphic History CRUD
**Tasks covered:** T-E21-F08-004

**Steps:**
1. Verify EntityHistoryRepository.Create() stores records for all entity types
2. Verify ListByEntity() returns history ordered by changed_at DESC
3. Verify ListRecent() returns N most recent records across all types
4. Verify ListWithFilters() supports entity_type, changed_by, since, from_status, to_status filters
5. Verify forced flag and rejection_reason are stored correctly

**Success Criteria:**
- [ ] History CRUD works for all 5 entity types
- [ ] Filtering and ordering correct
- [ ] Forced transitions and rejection reasons preserved

### Scenario 5: History Recording in EntityService.TransitionStatus
**Tasks covered:** T-E21-F08-006

**Steps:**
1. Verify TransitionStatus creates entity_history record for each transition
2. Verify history records capture: entity_type, entity_id, from_status, to_status, changed_by, notes, forced
3. Verify history recording is non-blocking (failure doesn't roll back status update)
4. Verify optional dependency (nil historyRepo degrades gracefully)

**Success Criteria:**
- [ ] Every status transition creates a history record
- [ ] History recording errors do not block status transitions
- [ ] Works with nil historyRepo (backward compat)

### Scenario 6: EntityHistoryService Query Operations
**Tasks covered:** T-E21-F08-007

**Steps:**
1. Verify GetHistory() returns history for any entity type by key
2. Verify entity key auto-detection (E## = epic, E##-F## = feature, etc.)
3. Verify GetRecentHistory() returns recent history across all types
4. Verify filtering by entity_type, changed_by, time range

**Success Criteria:**
- [ ] History queries work for all entity types
- [ ] Key auto-detection routes to correct entity type

### Scenario 7: CLI Updates — Status History and Related-Docs
**Tasks covered:** T-E21-F08-008

**Steps:**
1. Verify `shark status history <feature-key>` returns feature history
2. Verify `shark status history <epic-key>` returns epic history
3. Verify `shark status history <task-key>` backward compatible
4. Verify existing related-docs commands work unchanged

**Success Criteria:**
- [ ] Status history works for all entity types via CLI
- [ ] Related-docs commands backward compatible

### Scenario 8: Legacy Code Removal
**Tasks covered:** T-E21-F08-009

**Steps:**
1. Verify old entity-specific document methods removed from document_repository.go
2. Verify no dead code or unused imports remain
3. Verify build succeeds with clean compilation
4. Verify all tests pass after cleanup

**Success Criteria:**
- [ ] Dead code removed
- [ ] Clean build with no warnings
- [ ] All tests pass

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-03-21 08:00:00 |
| Result | REJECT (blockers: history not wired for task/bug/change; dead APIs) |
| Results File | docs/uat/E21/results/UAT-E21-F08-20260321-080000-results.md |

**Previous Sessions:**
- 2026-03-21: REJECT — Codex red-team found 2 blockers (history wiring gap, dead APIs) + 3 non-blocking issues
