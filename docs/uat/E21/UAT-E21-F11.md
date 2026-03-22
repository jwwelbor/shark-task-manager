# UAT Test Guide - Polymorphic Entity Relationships

**Feature:** E21-F11 - Polymorphic Entity Relationships
**Epic:** E21 - Entity Polymorphism and Duplication Reduction
**Generated:** 2026-03-21
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Introduce entity polymorphism and reduce cross-entity code duplication in Shark Task Manager by establishing shared interfaces, a registry pattern, and unified services that operate on any entity type.

**This Feature's Role:** Unifies three inconsistent entity linking patterns (task_relationships table, bug flat columns, change_card FKs) plus a legacy depends_on JSON field into a single polymorphic `entity_relationships` table. Enables typed relationships between any entity types (25 combinations vs prior task-only).

**Related Features:**
- E21-F01: Entity Interface Foundation (completed) - provides Entity interface used by relationship service
- E21-F02: Cross-Cutting Service Unification (completed) - EntityRegistry pattern
- E21-F03: Status Transition Unification (completed)
- E21-F10: CLI Command Consolidation (completed)

**Integration Points:**
- EntityRelationshipService replaces TaskDependencyService for task dependency operations
- CLI `shark link`/`shark task link`/`shark task unlink` commands use new service
- `shark task deps` reads from entity_relationships table
- Bug and ChangeCard services updated to use relationships instead of flat columns/FKs
- Database migration adds entity_relationships table and migrates legacy data

---

## Design Intent

**From Feature PRD:**
> Create a polymorphic `entity_relationships` table that supports typed links between **any two entities of any type**. This replaces: (1) task_relationships table, (2) Bug LinkedEntityType/LinkedEntityKey fields, (3) ChangeCard EpicID/FeatureID/RelatedTaskID fields, (4) Legacy depends_on JSON field on tasks.

**From Architecture Spec:**
> Schema uses UNIQUE(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) constraint. No foreign-key constraints on entity IDs (SQLite limitation for polymorphic tables). Referential integrity enforced at service/repository layer. 8 relationship types: depends_on, blocks, related_to, follows, spawned_from, duplicates, references, linked_to.

**Key Design Decisions:**
- `linked_to` added as 8th relationship type for bug context links (semantic distinction from structured dependencies)
- Entity type stored as text matching existing `models.EntityType` constants ('epic', 'feature', 'task', 'bug', 'change')
- DFS cycle detection generalized from task-only to any entity chain for depends_on/blocks types
- Migration command (`shark admin migrate relationships`) handles one-time data migration from legacy tables

---

## Cross-Feature Integration Tests

### Integration Scenario 1: Entity Interface + Relationships
**Features:** E21-F01 (Entity Interface) + E21-F11 (Relationships)
**Scenario:** EntityRelationshipService resolves entities using EntityType from the Entity interface

Steps:
1. Create a relationship between a task and a bug using entity types from models.EntityType
2. Query relationships for the task
3. Verify entity type strings match models.EntityType constants

Expected Result: Relationship types are consistent with Entity interface type system

### Integration Scenario 2: CLI Commands + Relationship Service
**Features:** E21-F10 (CLI Consolidation) + E21-F11 (Relationships)
**Scenario:** Link/unlink commands correctly invoke EntityRelationshipService

Steps:
1. Use `shark link` to create a cross-entity relationship
2. Use `shark task link` to create a task-to-task relationship (backward compat)
3. Use `shark task deps` to view dependencies
4. Use `shark task unlink` to remove a relationship

Expected Result: All CLI paths correctly create/remove/query entity_relationships records

---

## Feature Acceptance Validation

| Feature AC | Description | Status |
|------------|-------------|--------|
| REQ-F-001 | entity_relationships table with from/to entity types and 8 relationship types | [ ] |
| REQ-F-002 | Data migration from task_relationships, bug links, change_card FKs, legacy depends_on | [ ] |
| REQ-F-003 | EntityRelationshipRepository with Create, Delete, GetByEntity, GetOutgoing, GetIncoming, DetectCycle | [ ] |
| REQ-F-004 | EntityRelationshipService replacing TaskDependencyService | [ ] |
| REQ-F-005 | Legacy linking code deprecated/removed | [ ] |
| Story 1 | Any entity can have typed relationships with any other entity | [ ] |
| Story 2 | Existing task relationships migrated with no data loss | [ ] |
| Story 3 | Cycle detection works across entity types | [ ] |
| Story 4 | Bidirectional relationship querying for any entity | [ ] |
| Story 5 | Top-level `shark link` command with auto-detection | [ ] |

---

## Test Scenarios

### Scenario 1: Schema and Migration (T-E21-F11-001)
**Tasks covered:** T-E21-F11-001

**Steps:**
1. Verify entity_relationships table exists in database schema
2. Verify UNIQUE constraint on (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
3. Verify CHECK constraints on entity types and relationship types
4. Verify indexes exist (idx_er_from, idx_er_to, idx_er_type)
5. Verify CurrentSchemaVersion was bumped

**Success Criteria:**
- [ ] Table created with correct schema
- [ ] Constraints enforce valid entity types and relationship types
- [ ] Indexes exist for performance

### Scenario 2: Repository CRUD Operations (T-E21-F11-002)
**Tasks covered:** T-E21-F11-002

**Steps:**
1. Verify Create relationship works for all entity type combinations
2. Verify Delete relationship works
3. Verify GetByEntity returns all relationships for a given entity
4. Verify GetOutgoing returns relationships from an entity filtered by type
5. Verify GetIncoming returns relationships to an entity filtered by type
6. Verify duplicate relationship creation returns appropriate error

**Success Criteria:**
- [ ] Repository tests pass for all CRUD operations
- [ ] Polymorphic queries work across entity types
- [ ] Unique constraint violations handled gracefully

### Scenario 3: Cycle Detection (T-E21-F11-003)
**Tasks covered:** T-E21-F11-003

**Steps:**
1. Verify DFS cycle detection prevents A depends_on B depends_on A (same entity type)
2. Verify cycle detection works across entity types (Task depends_on Bug depends_on Task)
3. Verify non-cyclic chains are allowed
4. Verify cycle detection only applies to depends_on and blocks relationship types
5. Verify related_to and other types allow "cycles" (they're informational)

**Success Criteria:**
- [ ] Cycle detection prevents circular depends_on/blocks chains
- [ ] Cross-entity cycle detection works
- [ ] Non-cyclic relationships are allowed
- [ ] Informational relationship types are not cycle-checked

### Scenario 4: Data Migration (T-E21-F11-004)
**Tasks covered:** T-E21-F11-004

**Steps:**
1. Verify migration command exists (`shark admin migrate relationships` or similar)
2. Verify task_relationships data migrates to entity_relationships with entity_type='task'
3. Verify bug linked_entity_type/linked_entity_key migrates to entity_relationships
4. Verify change_card FK relationships migrate to entity_relationships
5. Verify legacy depends_on JSON data is handled (deduplicated with existing relationships)
6. Verify row counts match between old and new tables

**Success Criteria:**
- [ ] Migration command runs without errors
- [ ] All legacy relationships preserved in new table
- [ ] No duplicate records created
- [ ] Migration is idempotent (safe to run multiple times)

### Scenario 5: CLI Commands (T-E21-F11-005)
**Tasks covered:** T-E21-F11-005

**Steps:**
1. Verify `shark link <entity1> <entity2> --type=<type>` works with auto-detection
2. Verify `shark task link <task1> <task2> --type=depends_on` backward compatibility
3. Verify `shark task unlink <task1> <task2>` removes relationship
4. Verify `shark task deps <task>` shows dependency tree from new table
5. Verify link command validates entity keys exist

**Success Criteria:**
- [ ] Top-level `shark link` command works
- [ ] Entity-specific link commands remain functional
- [ ] Dependency display reads from entity_relationships
- [ ] Error handling for invalid entity keys

### Scenario 6: Legacy Code Removal (T-E21-F11-006)
**Tasks covered:** T-E21-F11-006

**Steps:**
1. Verify legacy task_relationships, epic_relationships, feature_relationships repository code removed or deprecated
2. Verify TaskDependencyService replaced by EntityRelationshipService
3. Verify bug LinkedEntityType/LinkedEntityKey handling updated
4. Verify change_card FK relationship logic updated
5. Verify all tests pass after legacy code removal

**Success Criteria:**
- [ ] Legacy relationship repositories removed or deprecated
- [ ] No code references old relationship patterns for new operations
- [ ] All existing tests pass
- [ ] `make fmt && make lint && make test` passes

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | (none yet) |
| Result | - |
| Results File | - |

**Previous Sessions:** None
