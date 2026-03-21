---
feature_key: E21-F08-polymorphic-data-model-unification
epic_key: E21
title: Polymorphic Data Model Unification
description: Unify document associations and history tracking into polymorphic tables available to all entity types
---

# Polymorphic Data Model Unification

**Feature Key**: E21-F08-polymorphic-data-model-unification

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

The database has **inconsistent polymorphism** across cross-cutting concerns:

| Concern | Current State | Available To |
|---------|--------------|-------------|
| Notes | `entity_notes` (polymorphic, entity_type + entity_id) | All 5 entity types |
| Documents | `epic_documents`, `feature_documents`, `task_documents`, `bug_documents`, `change_card_documents` (5 separate tables) | All 5 types, but via 5 duplicate tables |
| History | `task_history` (task-only) | Only Task |

Bugs and ChangeCards cannot have related documents. Only Tasks have status change history. This contradicts the core design intent that **all entities should have access to all additional features**.

Adding document support for bugs requires creating a 4th table (`bug_documents`) and a 4th set of repository methods — rather than simply registering the entity type in an existing polymorphic table.

### Solution

1. Create a polymorphic `entity_documents` table following the same pattern as `entity_notes` (entity_type + entity_id)
2. Create a polymorphic `entity_history` table for status change tracking across all entity types
3. Migrate existing data from the 3 separate document tables and `task_history` into the new tables
4. Remove the old per-entity tables after migration

### Impact

- All 5 entity types (and future entity types) automatically get document and history support
- Adding a 6th entity type requires **zero schema changes** for documents and history
- Eliminate 3 duplicate document tables and their associated repository code
- Enable audit trail for Epic, Feature, Bug, and ChangeCard status changes (currently task-only)

---

## User Personas

### Persona 1: Developer Agent

**Goals Related to This Feature**:
1. Link design documents to bugs and change-cards (currently impossible)
2. View status change history for features and epics (currently impossible)
3. Use the same API/commands for document management regardless of entity type

**Pain Points This Feature Addresses**:
- `shark related-docs add --bug=B001 --path=docs/analysis.md` doesn't work — bugs have no document table
- `shark status history E21-F07` returns nothing — only tasks have history
- Adding document support for a new entity type requires a new table + new repository + new service methods

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want to link documents to any entity type so that bugs and change-cards can reference design docs and analysis.

**Acceptance Criteria**:
- [ ] `entity_documents` table created with `entity_type`, `entity_id`, `document_id` columns
- [ ] Document linking works for all 5 entity types via same CLI commands
- [ ] Existing epic/feature/task document links preserved after migration

**Story 2**: As a developer, I want status change history for all entity types so that I can audit when and why any entity's status changed.

**Acceptance Criteria**:
- [ ] `entity_history` table created with `entity_type`, `entity_id`, `from_status`, `to_status`, `changed_at`, `changed_by`, `notes` columns
- [ ] All status transitions (via TransitionStatus, SetStatus, AdvanceStatus) create history records for all entity types
- [ ] Existing task_history data preserved after migration
- [ ] `shark status history E21-F07` returns feature status changes
- [ ] `shark status history B001` returns bug status changes

**Story 3**: As a developer, I want the old per-entity document tables removed so that there's one canonical way to store document associations.

**Acceptance Criteria**:
- [ ] Data migrated from `epic_documents`, `feature_documents`, `task_documents`, `bug_documents`, `change_card_documents` to `entity_documents`
- [ ] Old tables dropped after successful migration verification
- [ ] Repository code updated to use single `entity_documents` table
- [ ] CLI commands produce identical output post-migration

---

### Should-Have Stories

**Story 4**: As a developer, I want history records to capture the agent/user who made the change so that I can trace who changed what.

**Acceptance Criteria**:
- [ ] `changed_by` column stores agent ID or "user" for manual changes
- [ ] Transition options propagate agent identity to history records

---

## Requirements

### Functional Requirements

**Category: Schema Changes**

1. **REQ-F-001**: Polymorphic Entity Documents Table
   - **Description**: Create `entity_documents` table with polymorphic entity reference
   - **Priority**: Must-Have
   - **Schema**:
     ```sql
     CREATE TABLE entity_documents (
         id INTEGER PRIMARY KEY AUTOINCREMENT,
         entity_type TEXT NOT NULL,  -- 'epic', 'feature', 'task', 'bug', 'change_card'
         entity_id INTEGER NOT NULL,
         document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
         created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
         UNIQUE(entity_type, entity_id, document_id)
     );
     CREATE INDEX idx_entity_documents_lookup ON entity_documents(entity_type, entity_id);
     ```

2. **REQ-F-002**: Polymorphic Entity History Table
   - **Description**: Create `entity_history` table for status change audit trail
   - **Priority**: Must-Have
   - **Schema**:
     ```sql
     CREATE TABLE entity_history (
         id INTEGER PRIMARY KEY AUTOINCREMENT,
         entity_type TEXT NOT NULL,
         entity_id INTEGER NOT NULL,
         from_status TEXT NOT NULL,
         to_status TEXT NOT NULL,
         changed_by TEXT,
         notes TEXT,
         changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
     );
     CREATE INDEX idx_entity_history_lookup ON entity_history(entity_type, entity_id);
     CREATE INDEX idx_entity_history_time ON entity_history(changed_at);
     ```

3. **REQ-F-003**: Data Migration
   - **Description**: Migrate existing document associations and task history to new polymorphic tables
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `epic_documents` rows migrated to `entity_documents` with entity_type='epic'
     - [ ] `feature_documents` rows migrated with entity_type='feature'
     - [ ] `task_documents` rows migrated with entity_type='task'
     - [ ] `task_history` rows migrated to `entity_history` with entity_type='task'
     - [ ] Row counts verified before/after migration
     - [ ] Old tables dropped after verification

4. **REQ-F-004**: Repository Consolidation
   - **Description**: Replace per-entity document repositories with single polymorphic repository
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Single `EntityDocumentRepository` replaces `EpicDocumentRepository`, `FeatureDocumentRepository`, `TaskDocumentRepository`
     - [ ] Single `EntityHistoryRepository` replaces `TaskHistoryRepository`
     - [ ] All methods accept `entityType` and `entityID` parameters
     - [ ] Existing service interfaces updated to use new repositories

**Category: Service Layer Updates**

5. **REQ-F-005**: History Recording in TransitionStatus
   - **Description**: EntityService.TransitionStatus automatically creates entity_history records
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Every status transition creates a history record with from_status, to_status, changed_by, notes
     - [ ] Works for all 5 entity types without per-entity code

### Non-Functional Requirements

**Data Integrity**

1. **REQ-NF-001**: Migration Safety
   - **Description**: Migration must be reversible if data validation fails
   - **Measurement**: Old tables retained until new table data verified
   - **Target**: Zero data loss during migration

**Performance**

2. **REQ-NF-002**: Query Performance
   - **Description**: Polymorphic table queries must not degrade vs per-entity tables
   - **Measurement**: `entity_type + entity_id` composite index ensures O(log n) lookup
   - **Target**: No measurable performance difference for typical dataset sizes (<100k rows)

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Document Linking for Bugs**
- **Given** a bug B001 exists
- **When** `shark related-docs add --bug=B001 --path=docs/analysis.md`
- **Then** document is linked to the bug via entity_documents table
- **And** `shark related-docs list --bug=B001` shows the linked document

**Scenario 2: History for Features**
- **Given** a feature E21-F07 exists with status "draft"
- **When** status is advanced to "active"
- **Then** `shark status history E21-F07` shows the transition record with timestamp

**Scenario 3: Migration Preservation**
- **Given** epic E01 has 3 linked documents in `epic_documents`
- **When** migration runs
- **Then** `shark related-docs list --epic=E01` returns the same 3 documents from `entity_documents`

---

## Out of Scope

### Explicitly Excluded

1. **Entity Table Unification** (single `entities` table for all types)
   - **Why**: Too large a change with entity-specific columns varying significantly (Task has 20+ fields)
   - **Future**: Could be revisited if entity-specific fields converge further

2. **Cascading Delete Across Polymorphic Tables**
   - **Why**: SQLite foreign keys can't reference polymorphic entity_type + entity_id pairs
   - **Workaround**: Application-level cascade in delete operations, or triggers per entity type

---

## Design Notes

### Migration Strategy

The migration adds new tables alongside old ones, migrates data, then drops old tables. This is safe because:

1. New tables created with `IF NOT EXISTS`
2. Data copied with `INSERT INTO entity_documents SELECT ... FROM epic_documents`
3. Row counts verified
4. Old tables dropped only after verification
5. `CurrentSchemaVersion` bumped to trigger migration

### Impact on `skip_migrations` Flag

Per database-critical rules: this change adds a migration. The developer must temporarily set `skip_migrations: false` in `.sharkconfig.json` before running the next shark command, then set it back to `true`.

### Polymorphic Pattern (Matching entity_notes)

The `entity_notes` table already proves this pattern works in the codebase:

```sql
-- Already exists and works
CREATE TABLE entity_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    note_type TEXT NOT NULL DEFAULT 'comment',
    content TEXT NOT NULL,
    ...
);
```

`entity_documents` and `entity_history` follow the exact same pattern.

---

## Dependencies & Integrations

### Dependencies

- **E21-F07** (BaseEntity): If completed first, entity_type values come from `BaseEntity.GetEntityType()`
- **E21-F09** (Service Delegation): EntityService.TransitionStatus should create history records automatically

### Related Features

- **E21-F11** (Polymorphic Entity Relationships): Same polymorphic table pattern for entity_relationships
- **E21-F12** (Polymorphic Acceptance Criteria): Same polymorphic table pattern for entity_criteria

### Integration Points

- **EntityDocumentService**: Refactored to use single `entity_documents` table
- **CLI Commands**: `related-docs` commands updated to work with all entity types
- **Status commands**: `status history` command updated to query `entity_history` for any entity type

---

## Success Metrics

### Primary Metrics

1. **Entity Type Coverage**
   - **Target**: All 5 entity types have document and history support (currently 3/5 for docs, 1/5 for history)
   - **Measurement**: `shark related-docs add` and `shark status history` work for all entity types

2. **Table Count Reduction**
   - **Target**: 6 tables (epic_documents, feature_documents, task_documents, bug_documents, change_card_documents, task_history) replaced by 2 (entity_documents, entity_history)
   - **Measurement**: `.schema` output in SQLite

3. **Repository Code Reduction**
   - **Target**: 5 document join tables and their repository methods replaced by 1 polymorphic repository; 1 history repository generalized
   - **Measurement**: Lines of repository code

---

*Last Updated*: 2026-03-20
