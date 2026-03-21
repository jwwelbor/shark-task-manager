---
feature_key: E21-F11-polymorphic-entity-relationships
epic_key: E21
title: Polymorphic Entity Relationships
description: Unify three inconsistent entity linking patterns into a single polymorphic entity_relationships table supporting typed links between any entity types
---

# Polymorphic Entity Relationships

**Feature Key**: E21-F11-polymorphic-entity-relationships

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

The codebase has **three different patterns** for expressing relationships between entities, none of which are universal:

| Pattern | Used By | Backed By | Cardinality | Cross-Entity? |
|---------|---------|-----------|-------------|---------------|
| Typed relationship table | Task ↔ Task only | `task_relationships` | Many-to-Many | No (task-only) |
| Field-based linking | Bug → Epic/Feature/Task | `LinkedEntityType`/`LinkedEntityKey` columns | One-to-One | Partially |
| Explicit foreign keys | ChangeCard → Epic/Feature/Task | `EpicID`/`FeatureID`/`RelatedTaskID` columns | One-to-Many | Partially |

Additionally, a **legacy `depends_on` JSON field** on the tasks table duplicates the `task_relationships` table functionality.

**What's missing:**
- A bug can't "block" a feature (no cross-entity typed relationships)
- A feature can't "depend on" another feature
- An epic can't be "related to" another epic
- ChangeCards can't reference bugs
- The 7 relationship types (depends_on, blocks, related_to, follows, spawned_from, duplicates, references) are only available between tasks
- Three different patterns must be understood and maintained

### Solution

Create a polymorphic `entity_relationships` table that supports typed links between **any two entities of any type**. This replaces:
1. `task_relationships` table (migrate data)
2. Bug `LinkedEntityType`/`LinkedEntityKey` fields (migrate to relationship)
3. ChangeCard `EpicID`/`FeatureID`/`RelatedTaskID` fields (migrate to relationships)
4. Legacy `depends_on` JSON field on tasks (remove after migration)

### Impact

- Any entity can have typed relationships with any other entity
- 7 relationship types available for all entity combinations (Bug blocks Feature, Epic depends_on Epic, etc.)
- Single relationship API/CLI for all entity types
- Eliminate 3 separate linking patterns + 1 legacy field
- Adding a 6th entity type gets relationship support with zero schema changes

---

## Current State Analysis

### System 1: task_relationships Table (Modern, Task-Only)

**Table Schema:**
```sql
CREATE TABLE task_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id INTEGER NOT NULL REFERENCES tasks(id),
    to_task_id INTEGER NOT NULL REFERENCES tasks(id),
    relationship_type TEXT NOT NULL,  -- depends_on, blocks, related_to, follows, spawned_from, duplicates, references
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_task_id, to_task_id, relationship_type)
);
```

**Strengths:**
- Typed relationships (7 types)
- Cycle detection for depends_on/blocks
- Bidirectional querying (incoming/outgoing)
- Rich service layer (TaskDependencyService)

**Weakness:** Locked to Task ↔ Task only. Can't express "Bug B001 blocks Feature E21-F07".

### System 2: Bug Field-Based Linking

**Fields on bugs table:**
- `linked_entity_type TEXT` — "epic", "feature", or "task"
- `linked_entity_key TEXT` — entity key like "E21-F07-001"

**Weakness:** One link only. No relationship type. No bidirectional querying.

### System 3: ChangeCard Foreign Keys

**Fields on change_cards table:**
- `epic_id INTEGER REFERENCES epics(id)`
- `feature_id INTEGER REFERENCES features(id)`
- `related_task_id INTEGER REFERENCES tasks(id)`

**Weakness:** Only links to epics/features/tasks. Can't link to bugs or other change-cards. No relationship type.

### System 4: Legacy depends_on JSON Field

**Field on tasks table:**
- `depends_on TEXT` — JSON array of task keys

**Weakness:** Duplicates `task_relationships`. No referential integrity. Being phased out.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want to express typed relationships between any entity types so that a bug can block a feature or an epic can depend on another epic.

**Acceptance Criteria**:
- [ ] `entity_relationships` table created with `from_entity_type`, `from_entity_id`, `to_entity_type`, `to_entity_id`, `relationship_type`
- [ ] All 7 relationship types available for any entity combination
- [ ] `shark task link E21-F07-001 E21-F07-002 --type=depends_on` still works (backward compatible)
- [ ] `shark bug link B001 E21-F07 --type=blocks` works (new capability)
- [ ] `shark epic link E21 E22 --type=related_to` works (new capability)

**Story 2**: As a developer, I want existing task relationships migrated to the new table so that no data is lost.

**Acceptance Criteria**:
- [ ] `task_relationships` data migrated to `entity_relationships` with from/to entity_type='task'
- [ ] Bug `LinkedEntityType`/`LinkedEntityKey` migrated to relationship records
- [ ] ChangeCard FK relationships migrated to relationship records
- [ ] Legacy `depends_on` JSON data migrated (where not already in task_relationships)
- [ ] Old tables/fields deprecated after verification

**Story 3**: As a developer, I want cycle detection to work across entity types so that circular depends_on/blocks chains are prevented.

**Acceptance Criteria**:
- [ ] `DetectCycle()` works with polymorphic entity references
- [ ] Prevents: Task A depends_on Bug B depends_on Task A
- [ ] Existing task-only cycle detection behavior preserved

**Story 4**: As a developer, I want bidirectional relationship querying for any entity so that I can ask "what does this bug block?" and "what blocks this feature?"

**Acceptance Criteria**:
- [ ] `shark task deps E21-F07-001` shows dependencies (backward compatible)
- [ ] `shark get E21-F07 --field relationships` shows all relationships for any entity type
- [ ] Incoming and outgoing queries work for all entity types

---

### Should-Have Stories

**Story 5**: As a developer, I want the `shark link` command to work at the top level with auto-detection so that I don't need entity-specific link commands.

**Acceptance Criteria**:
- [ ] `shark link E21-F07-001 B001 --type=related_to` auto-detects both entity types
- [ ] Entity-specific commands (`shark task link`) remain as aliases

---

## Requirements

### Functional Requirements

**Category: Schema**

1. **REQ-F-001**: Polymorphic Entity Relationships Table
   - **Priority**: Must-Have
   - **Schema**:
     ```sql
     CREATE TABLE entity_relationships (
         id INTEGER PRIMARY KEY AUTOINCREMENT,
         from_entity_type TEXT NOT NULL,
         from_entity_id INTEGER NOT NULL,
         to_entity_type TEXT NOT NULL,
         to_entity_id INTEGER NOT NULL,
         relationship_type TEXT NOT NULL,  -- depends_on, blocks, related_to, follows, spawned_from, duplicates, references
         created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
         UNIQUE(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
     );
     CREATE INDEX idx_entity_rel_from ON entity_relationships(from_entity_type, from_entity_id);
     CREATE INDEX idx_entity_rel_to ON entity_relationships(to_entity_type, to_entity_id);
     CREATE INDEX idx_entity_rel_type ON entity_relationships(relationship_type);
     ```

2. **REQ-F-002**: Data Migration
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `task_relationships` → `entity_relationships` (type='task' for both ends)
     - [ ] Bug linked entities → `entity_relationships` (type='related_to')
     - [ ] ChangeCard FK refs → `entity_relationships` (type='related_to')
     - [ ] Task `depends_on` JSON → `entity_relationships` (type='depends_on', deduplicated with task_relationships)
     - [ ] Row counts verified

3. **REQ-F-003**: EntityRelationshipRepository
   - **Priority**: Must-Have
   - **Description**: Single repository replacing TaskRelationshipRepository, with polymorphic entity support
   - **Methods**: Create, Delete, GetByEntity(type, id), GetOutgoing(type, id, relType), GetIncoming(type, id, relType), DetectCycle(fromType, fromID, toType, toID, relType)

4. **REQ-F-004**: EntityRelationshipService
   - **Priority**: Must-Have
   - **Description**: Replaces TaskDependencyService with polymorphic relationship operations
   - **Key**: Cycle detection generalizes from task-only to any entity chain

**Category: Legacy Cleanup**

5. **REQ-F-005**: Deprecate Legacy Systems
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `depends_on` JSON field on tasks table deprecated (nullable, not populated for new tasks)
     - [ ] `LinkedEntityType`/`LinkedEntityKey` on bugs deprecated
     - [ ] `EpicID`/`FeatureID`/`RelatedTaskID` on change_cards deprecated for relationship use (may keep for hierarchy)
     - [ ] Old `task_relationships` table dropped after migration verification

---

## Out of Scope

### Explicitly Excluded

1. **Removing ChangeCard hierarchy FKs entirely**
   - **Why**: `EpicID`/`FeatureID` on ChangeCards may serve a hierarchy purpose (which epic/feature does this change affect?) beyond just "relationship"
   - **Future**: Evaluate whether these should be pure relationships or remain as structural hierarchy

2. **Relationship visualization / graph UI**
   - **Why**: CLI-first project; graph visualization is a presentation concern
   - **Future**: Could be added as a separate feature

---

## Design Notes

### Relationship Types

The existing 7 types from `task_relationships` carry over, with semantic definitions that work across entity types:

| Type | Meaning | Example |
|------|---------|---------|
| `depends_on` | Cannot start until target completes | Feature depends_on Feature |
| `blocks` | Prevents target from starting | Bug blocks Feature |
| `related_to` | Informational link | Epic related_to Epic |
| `follows` | Should be done after target | Task follows Task |
| `spawned_from` | Created as a result of target | Bug spawned_from Task |
| `duplicates` | Same as target | Bug duplicates Bug |
| `references` | Mentions or relates to target | ChangeCard references Task |

### Cycle Detection Generalization

Current task-only DFS cycle detection in `TaskDependencyService.ValidateDependencies()` becomes:

```go
func (s *EntityRelationshipService) DetectCycle(ctx context.Context,
    fromType models.EntityType, fromID int64,
    toType models.EntityType, toID int64,
    relType string) (bool, error) {
    // DFS from (toType, toID) following relType edges
    // If we reach (fromType, fromID), cycle exists
    visited := map[string]bool{}
    return s.dfs(ctx, toType, toID, fromType, fromID, relType, visited)
}
```

---

## Dependencies & Integrations

### Dependencies

- **E21-F08** (Polymorphic Data Model): Same migration pattern (polymorphic tables)
- **E21-F09** (Service Delegation): EntityRelationshipService should be accessible via EntityRegistry

### Integration Points

- **TaskDependencyService**: Replaced by EntityRelationshipService
- **BugService**: Remove LinkEntity field logic, use relationship service
- **ChangeCardService**: Remove FK-based linking for relationship use cases
- **CLI `shark task link/deps/blocks/blocked-by`**: Updated to use new service
- **CLI**: New top-level `shark link` command for cross-entity relationships

---

## Success Metrics

1. **Pattern Unification**: 3 linking patterns + 1 legacy field → 1 polymorphic table
2. **Entity Type Coverage**: All 5 entity types can link to all 5 (25 combinations vs current 1)
3. **Relationship Type Coverage**: 7 types available for all combinations (vs current task-only)
4. **Code Reduction**: TaskDependencyService (~200 lines) + Bug linking code + ChangeCard FK code consolidated

---

*Last Updated*: 2026-03-20
