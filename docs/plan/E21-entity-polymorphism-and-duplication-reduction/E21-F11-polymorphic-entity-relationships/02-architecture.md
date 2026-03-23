# E21-F11: Polymorphic Entity Relationships — Architecture Specification

**Feature**: E21-F11-polymorphic-entity-relationships
**Status**: Architecture
**Date**: 2026-03-21
**Author**: Architect Agent

---

## Overview

This document specifies the full architecture for replacing four inconsistent entity-linking mechanisms with a single polymorphic `entity_relationships` table. The design eliminates approximately 955 lines of duplicated repository code while enabling typed relationships between any two entities of any type.

---

## 1. Database Schema

### 1.1 New Table: `entity_relationships`

```sql
CREATE TABLE IF NOT EXISTS entity_relationships (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    from_entity_type  TEXT NOT NULL CHECK(from_entity_type IN (
                        'epic', 'feature', 'task', 'bug', 'change_card'
                      )),
    from_entity_id    INTEGER NOT NULL,
    to_entity_type    TEXT NOT NULL CHECK(to_entity_type IN (
                        'epic', 'feature', 'task', 'bug', 'change_card'
                      )),
    to_entity_id      INTEGER NOT NULL,
    relationship_type TEXT NOT NULL CHECK(relationship_type IN (
                        'depends_on', 'blocks', 'related_to', 'follows',
                        'spawned_from', 'duplicates', 'references', 'linked_to'
                      )),
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
);

CREATE INDEX IF NOT EXISTS idx_er_from
    ON entity_relationships(from_entity_type, from_entity_id);

CREATE INDEX IF NOT EXISTS idx_er_to
    ON entity_relationships(to_entity_type, to_entity_id);

CREATE INDEX IF NOT EXISTS idx_er_type
    ON entity_relationships(relationship_type);
```

**Design notes:**

- No foreign-key constraints on `from_entity_id` / `to_entity_id`. SQLite cannot enforce foreign keys across multiple tables from a single column. Referential integrity is enforced at the service/repository layer.
- `linked_to` is added as an 8th relationship type specifically to represent the informal single-link pattern currently used by bugs. This preserves semantic distinction between structured task dependencies and a bug's "context" link.
- `change_card` uses an underscore to match the existing `EntityTypeChange` = `"change"` — see note below.

**Important:** The `from_entity_type` / `to_entity_type` CHECK constraint uses `'change_card'` but the existing `models.EntityType` constant is `EntityTypeChange = "change"`. The migration must choose one and apply it consistently. **Recommendation**: use `'change'` to match the existing `EntityType` constant and avoid a model layer inconsistency.

Updated CHECK constraints using existing constants:

```sql
CHECK(from_entity_type IN ('epic', 'feature', 'task', 'bug', 'change'))
CHECK(to_entity_type   IN ('epic', 'feature', 'task', 'bug', 'change'))
```

### 1.2 Migration Function in `internal/db/db.go`

The migration is added as a new numbered function in the existing `runMigrations()` chain. `CurrentSchemaVersion` must be incremented.

```go
// migrateAddEntityRelationships creates the entity_relationships table
// and its supporting indexes. This replaces the three type-specific
// relationship tables (task_relationships, epic_relationships,
// feature_relationships) and the bug/change_card flat-column patterns.
//
// Data migration from old tables is handled by a separate one-time
// migration script; this function only creates the schema objects.
func migrateAddEntityRelationships(db *sql.DB) error {
    stmts := []string{
        `CREATE TABLE IF NOT EXISTS entity_relationships (
            id                INTEGER PRIMARY KEY AUTOINCREMENT,
            from_entity_type  TEXT NOT NULL CHECK(from_entity_type IN (
                                'epic','feature','task','bug','change'
                              )),
            from_entity_id    INTEGER NOT NULL,
            to_entity_type    TEXT NOT NULL CHECK(to_entity_type IN (
                                'epic','feature','task','bug','change'
                              )),
            to_entity_id      INTEGER NOT NULL,
            relationship_type TEXT NOT NULL CHECK(relationship_type IN (
                                'depends_on','blocks','related_to','follows',
                                'spawned_from','duplicates','references','linked_to'
                              )),
            created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(from_entity_type, from_entity_id,
                   to_entity_type,   to_entity_id, relationship_type)
        )`,
        `CREATE INDEX IF NOT EXISTS idx_er_from
             ON entity_relationships(from_entity_type, from_entity_id)`,
        `CREATE INDEX IF NOT EXISTS idx_er_to
             ON entity_relationships(to_entity_type, to_entity_id)`,
        `CREATE INDEX IF NOT EXISTS idx_er_type
             ON entity_relationships(relationship_type)`,
    }

    for _, stmt := range stmts {
        if _, err := db.Exec(stmt); err != nil {
            return fmt.Errorf("migrateAddEntityRelationships: %w", err)
        }
    }
    return nil
}
```

**Developer action required**: Set `skip_migrations: false` in `.sharkconfig.json` before running the first shark command after this migration is committed. Reset to `true` after the schema is applied. Also bump `CurrentSchemaVersion` by 1.

### 1.3 Data Migration Queries

These queries are run once, in order, to drain data from the legacy systems. They should be wrapped in a transaction and can be executed via a dedicated `shark admin migrate relationships` command or directly via SQLite.

**Phase 1 — task_relationships → entity_relationships:**

```sql
-- Migrate task-to-task relationships (deduplication via INSERT OR IGNORE)
INSERT OR IGNORE INTO entity_relationships
    (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type, created_at)
SELECT
    'task', from_task_id,
    'task', to_task_id,
    relationship_type,
    created_at
FROM task_relationships;
```

**Phase 2 — depends_on JSON column → entity_relationships:**

```sql
-- Migrate legacy depends_on JSON arrays to entity_relationships.
-- Each element of the JSON array is a task key; we join to tasks to get the ID.
-- The UNIQUE constraint + INSERT OR IGNORE prevents duplicates with Phase 1.
INSERT OR IGNORE INTO entity_relationships
    (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
SELECT
    'task'     AS from_entity_type,
    t.id       AS from_entity_id,
    'task'     AS to_entity_type,
    dep_t.id   AS to_entity_id,
    'depends_on'
FROM tasks t
CROSS JOIN json_each(t.depends_on) AS je
JOIN tasks dep_t ON dep_t.key = je.value
WHERE t.depends_on IS NOT NULL
  AND t.depends_on != '[]'
  AND t.depends_on != 'null';
```

**Phase 3 — bug linked_entity columns → entity_relationships:**

```sql
-- Migrate bug → entity links.
-- linked_entity_type maps to the entity_type column values already in use.
-- Relationship type is 'linked_to' (the bug informal context link).
INSERT OR IGNORE INTO entity_relationships
    (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
SELECT
    'bug' AS from_entity_type,
    b.id  AS from_entity_id,
    b.linked_entity_type  AS to_entity_type,
    CASE b.linked_entity_type
        WHEN 'epic'    THEN e.id
        WHEN 'feature' THEN f.id
        WHEN 'task'    THEN t.id
    END   AS to_entity_id,
    'linked_to'
FROM bugs b
LEFT JOIN epics    e ON b.linked_entity_type = 'epic'    AND e.key = b.linked_entity_key
LEFT JOIN features f ON b.linked_entity_type = 'feature' AND f.key = b.linked_entity_key
LEFT JOIN tasks    t ON b.linked_entity_type = 'task'    AND t.key = b.linked_entity_key
WHERE b.linked_entity_type IS NOT NULL
  AND b.linked_entity_key  IS NOT NULL;
```

**Phase 4 — change_card FK columns → entity_relationships:**

```sql
-- Migrate change_card → epic relationships
INSERT OR IGNORE INTO entity_relationships
    (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
SELECT 'change', cc.id, 'epic', cc.epic_id, 'related_to'
FROM change_cards cc
WHERE cc.epic_id IS NOT NULL;

-- Migrate change_card → feature relationships
INSERT OR IGNORE INTO entity_relationships
    (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
SELECT 'change', cc.id, 'feature', cc.feature_id, 'related_to'
FROM change_cards cc
WHERE cc.feature_id IS NOT NULL;

-- Migrate change_card → task relationships
INSERT OR IGNORE INTO entity_relationships
    (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
SELECT 'change', cc.id, 'task', cc.related_task_id, 'related_to'
FROM change_cards cc
WHERE cc.related_task_id IS NOT NULL;
```

**Phase 5 — epic/feature relationship tables → entity_relationships:**

```sql
-- epic_relationships
INSERT OR IGNORE INTO entity_relationships
    (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type, created_at)
SELECT 'epic', from_epic_id, 'epic', to_epic_id, relationship_type, created_at
FROM epic_relationships;

-- feature_relationships
INSERT OR IGNORE INTO entity_relationships
    (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type, created_at)
SELECT 'feature', from_feature_id, 'feature', to_feature_id, relationship_type, created_at
FROM feature_relationships;
```

**Verification query:**

```sql
SELECT
    (SELECT COUNT(*) FROM entity_relationships WHERE from_entity_type='task' AND to_entity_type='task') AS task_rels,
    (SELECT COUNT(*) FROM task_relationships) AS old_task_rels,
    (SELECT COUNT(*) FROM entity_relationships WHERE from_entity_type='bug') AS bug_rels,
    (SELECT COUNT(*) FROM bugs WHERE linked_entity_type IS NOT NULL) AS old_bug_links,
    (SELECT COUNT(*) FROM entity_relationships WHERE from_entity_type='change') AS change_rels;
```

---

## 2. Model Layer

### 2.1 New Model: `internal/models/entity_relationship.go`

```go
package models

import (
    "fmt"
    "time"
)

// EntityRelationshipType represents the type of a polymorphic entity relationship.
// These types carry over from task_relationships and gain new semantics
// applicable across entity type combinations.
type EntityRelationshipType string

const (
    EntityRelDependsOn   EntityRelationshipType = "depends_on"   // Cannot start until target completes
    EntityRelBlocks      EntityRelationshipType = "blocks"       // Prevents target from starting
    EntityRelRelatedTo   EntityRelationshipType = "related_to"   // Informational link
    EntityRelFollows     EntityRelationshipType = "follows"      // Should be done after target
    EntityRelSpawnedFrom EntityRelationshipType = "spawned_from" // Created as result of target
    EntityRelDuplicates  EntityRelationshipType = "duplicates"   // Same work as target
    EntityRelReferences  EntityRelationshipType = "references"   // Mentions or uses target output
    EntityRelLinkedTo    EntityRelationshipType = "linked_to"    // Informal context link (bug pattern)
)

// ValidEntityRelationshipTypes is the set of all valid relationship types.
var ValidEntityRelationshipTypes = map[EntityRelationshipType]bool{
    EntityRelDependsOn:   true,
    EntityRelBlocks:      true,
    EntityRelRelatedTo:   true,
    EntityRelFollows:     true,
    EntityRelSpawnedFrom: true,
    EntityRelDuplicates:  true,
    EntityRelReferences:  true,
    EntityRelLinkedTo:    true,
}

// CyclicRelationshipTypes are the relationship types for which circular
// chains must be prevented. Only applies within the same entity type pair
// (e.g., task depends_on task). Cross-entity cycles are not enforced.
var CyclicRelationshipTypes = map[EntityRelationshipType]bool{
    EntityRelDependsOn: true,
    EntityRelBlocks:    true,
}

// EntityRelationship is the polymorphic relationship between any two entities.
type EntityRelationship struct {
    ID               int64                  `json:"id"                db:"id"`
    FromEntityType   EntityType             `json:"from_entity_type"  db:"from_entity_type"`
    FromEntityID     int64                  `json:"from_entity_id"    db:"from_entity_id"`
    ToEntityType     EntityType             `json:"to_entity_type"    db:"to_entity_type"`
    ToEntityID       int64                  `json:"to_entity_id"      db:"to_entity_id"`
    RelationshipType EntityRelationshipType `json:"relationship_type" db:"relationship_type"`
    CreatedAt        time.Time              `json:"created_at"        db:"created_at"`
}

// Validate performs structural validation on the EntityRelationship.
// Business rules (cycle detection, entity existence) are enforced at the service layer.
func (er *EntityRelationship) Validate() error {
    if er.FromEntityID == 0 {
        return fmt.Errorf("from_entity_id must not be zero")
    }
    if er.ToEntityID == 0 {
        return fmt.Errorf("to_entity_id must not be zero")
    }
    if !ValidEntityTypes[er.FromEntityType] {
        return fmt.Errorf("invalid from_entity_type: %s", er.FromEntityType)
    }
    if !ValidEntityTypes[er.ToEntityType] {
        return fmt.Errorf("invalid to_entity_type: %s", er.ToEntityType)
    }
    if !ValidEntityRelationshipTypes[er.RelationshipType] {
        return fmt.Errorf("invalid relationship_type: %s", er.RelationshipType)
    }
    if er.FromEntityType == er.ToEntityType && er.FromEntityID == er.ToEntityID {
        return fmt.Errorf("entity cannot have a relationship with itself")
    }
    return nil
}

// IsCyclic reports whether this relationship type requires cycle detection
// when both entities are of the same type.
func (er *EntityRelationship) IsCyclic() bool {
    return CyclicRelationshipTypes[er.RelationshipType] &&
        er.FromEntityType == er.ToEntityType
}
```

---

## 3. Repository Layer

### 3.1 `EntityRelationshipRepository` Interface

The interface is defined in the services package at the point of use, following the project's "accept interfaces, return structs" principle.

```go
// In internal/services/entity_relationship_service.go

// EntityRelationshipRepository defines the data access contract for
// EntityRelationshipService. The concrete implementation is
// *repository.EntityRelationshipRepository.
type EntityRelationshipRepository interface {
    // Create inserts a new polymorphic relationship.
    // Returns an error if the UNIQUE constraint is violated.
    Create(ctx context.Context, rel *models.EntityRelationship) error

    // Delete removes a relationship by its primary key.
    Delete(ctx context.Context, id int64) error

    // DeleteByEntitiesAndType removes the specific directed relationship
    // between two entities of the given type.
    DeleteByEntitiesAndType(
        ctx context.Context,
        fromType models.EntityType, fromID int64,
        toType models.EntityType, toID int64,
        relType models.EntityRelationshipType,
    ) error

    // GetByEntity returns all relationships where the entity appears
    // on either the from or to side (bidirectional).
    GetByEntity(
        ctx context.Context,
        entityType models.EntityType,
        entityID int64,
    ) ([]*models.EntityRelationship, error)

    // GetOutgoing returns relationships where this entity is the source,
    // optionally filtered by one or more relationship types.
    GetOutgoing(
        ctx context.Context,
        entityType models.EntityType,
        entityID int64,
        relTypes []models.EntityRelationshipType,
    ) ([]*models.EntityRelationship, error)

    // GetIncoming returns relationships where this entity is the target,
    // optionally filtered by one or more relationship types.
    GetIncoming(
        ctx context.Context,
        entityType models.EntityType,
        entityID int64,
        relTypes []models.EntityRelationshipType,
    ) ([]*models.EntityRelationship, error)
}
```

### 3.2 Concrete Implementation: `internal/repository/entity_relationship_repository.go`

```go
package repository

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityRelationshipRepository provides data access for the entity_relationships table.
type EntityRelationshipRepository struct {
    db *DB
}

// NewEntityRelationshipRepository creates a new EntityRelationshipRepository.
func NewEntityRelationshipRepository(db *DB) *EntityRelationshipRepository {
    return &EntityRelationshipRepository{db: db}
}

// Create inserts a new polymorphic relationship record.
func (r *EntityRelationshipRepository) Create(
    ctx context.Context,
    rel *models.EntityRelationship,
) error {
    if err := rel.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    query := `
        INSERT INTO entity_relationships
            (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
        VALUES (?, ?, ?, ?, ?)
    `
    result, err := r.db.ExecContext(ctx, query,
        rel.FromEntityType, rel.FromEntityID,
        rel.ToEntityType, rel.ToEntityID,
        rel.RelationshipType,
    )
    if err != nil {
        if strings.Contains(err.Error(), "UNIQUE constraint failed") {
            return fmt.Errorf("relationship already exists: %s(%d) -[%s]-> %s(%d)",
                rel.FromEntityType, rel.FromEntityID,
                rel.RelationshipType,
                rel.ToEntityType, rel.ToEntityID)
        }
        return fmt.Errorf("failed to create entity relationship: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get last insert id: %w", err)
    }
    rel.ID = id
    return nil
}

// Delete removes a relationship by primary key.
func (r *EntityRelationshipRepository) Delete(ctx context.Context, id int64) error {
    result, err := r.db.ExecContext(ctx,
        `DELETE FROM entity_relationships WHERE id = ?`, id)
    if err != nil {
        return fmt.Errorf("failed to delete entity relationship: %w", err)
    }
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("entity relationship not found with id %d", id)
    }
    return nil
}

// DeleteByEntitiesAndType removes a specific directed relationship.
func (r *EntityRelationshipRepository) DeleteByEntitiesAndType(
    ctx context.Context,
    fromType models.EntityType, fromID int64,
    toType models.EntityType, toID int64,
    relType models.EntityRelationshipType,
) error {
    result, err := r.db.ExecContext(ctx,
        `DELETE FROM entity_relationships
         WHERE from_entity_type = ? AND from_entity_id = ?
           AND to_entity_type   = ? AND to_entity_id   = ?
           AND relationship_type = ?`,
        fromType, fromID, toType, toID, relType,
    )
    if err != nil {
        return fmt.Errorf("failed to delete entity relationship: %w", err)
    }
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("relationship not found: %s(%d) -[%s]-> %s(%d)",
            fromType, fromID, relType, toType, toID)
    }
    return nil
}

// GetByEntity returns all relationships (incoming and outgoing) for an entity.
func (r *EntityRelationshipRepository) GetByEntity(
    ctx context.Context,
    entityType models.EntityType,
    entityID int64,
) ([]*models.EntityRelationship, error) {
    query := `
        SELECT id, from_entity_type, from_entity_id,
               to_entity_type, to_entity_id, relationship_type, created_at
        FROM entity_relationships
        WHERE (from_entity_type = ? AND from_entity_id = ?)
           OR (to_entity_type   = ? AND to_entity_id   = ?)
        ORDER BY created_at ASC
    `
    rows, err := r.db.QueryContext(ctx, query,
        entityType, entityID, entityType, entityID)
    if err != nil {
        return nil, fmt.Errorf("failed to query entity relationships: %w", err)
    }
    defer rows.Close()
    return r.scanRelationships(rows)
}

// GetOutgoing returns outgoing relationships for an entity, optionally filtered by type.
func (r *EntityRelationshipRepository) GetOutgoing(
    ctx context.Context,
    entityType models.EntityType,
    entityID int64,
    relTypes []models.EntityRelationshipType,
) ([]*models.EntityRelationship, error) {
    if len(relTypes) == 0 {
        rows, err := r.db.QueryContext(ctx,
            `SELECT id, from_entity_type, from_entity_id,
                    to_entity_type, to_entity_id, relationship_type, created_at
             FROM entity_relationships
             WHERE from_entity_type = ? AND from_entity_id = ?
             ORDER BY created_at ASC`,
            entityType, entityID)
        if err != nil {
            return nil, fmt.Errorf("failed to query outgoing relationships: %w", err)
        }
        defer rows.Close()
        return r.scanRelationships(rows)
    }

    placeholders := make([]string, len(relTypes))
    args := make([]interface{}, 0, 2+len(relTypes))
    args = append(args, entityType, entityID)
    for i, rt := range relTypes {
        placeholders[i] = "?"
        args = append(args, rt)
    }

    query := fmt.Sprintf(`
        SELECT id, from_entity_type, from_entity_id,
               to_entity_type, to_entity_id, relationship_type, created_at
        FROM entity_relationships
        WHERE from_entity_type = ? AND from_entity_id = ?
          AND relationship_type IN (%s)
        ORDER BY created_at ASC
    `, strings.Join(placeholders, ","))

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query outgoing relationships by type: %w", err)
    }
    defer rows.Close()
    return r.scanRelationships(rows)
}

// GetIncoming returns incoming relationships for an entity, optionally filtered by type.
func (r *EntityRelationshipRepository) GetIncoming(
    ctx context.Context,
    entityType models.EntityType,
    entityID int64,
    relTypes []models.EntityRelationshipType,
) ([]*models.EntityRelationship, error) {
    if len(relTypes) == 0 {
        rows, err := r.db.QueryContext(ctx,
            `SELECT id, from_entity_type, from_entity_id,
                    to_entity_type, to_entity_id, relationship_type, created_at
             FROM entity_relationships
             WHERE to_entity_type = ? AND to_entity_id = ?
             ORDER BY created_at ASC`,
            entityType, entityID)
        if err != nil {
            return nil, fmt.Errorf("failed to query incoming relationships: %w", err)
        }
        defer rows.Close()
        return r.scanRelationships(rows)
    }

    placeholders := make([]string, len(relTypes))
    args := make([]interface{}, 0, 2+len(relTypes))
    args = append(args, entityType, entityID)
    for i, rt := range relTypes {
        placeholders[i] = "?"
        args = append(args, rt)
    }

    query := fmt.Sprintf(`
        SELECT id, from_entity_type, from_entity_id,
               to_entity_type, to_entity_id, relationship_type, created_at
        FROM entity_relationships
        WHERE to_entity_type = ? AND to_entity_id = ?
          AND relationship_type IN (%s)
        ORDER BY created_at ASC
    `, strings.Join(placeholders, ","))

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query incoming relationships by type: %w", err)
    }
    defer rows.Close()
    return r.scanRelationships(rows)
}

// scanRelationships is a helper that scans query rows into EntityRelationship slices.
func (r *EntityRelationshipRepository) scanRelationships(
    rows *sql.Rows,
) ([]*models.EntityRelationship, error) {
    var rels []*models.EntityRelationship
    for rows.Next() {
        rel := &models.EntityRelationship{}
        err := rows.Scan(
            &rel.ID,
            &rel.FromEntityType, &rel.FromEntityID,
            &rel.ToEntityType, &rel.ToEntityID,
            &rel.RelationshipType,
            &rel.CreatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan entity relationship: %w", err)
        }
        rels = append(rels, rel)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating entity relationships: %w", err)
    }
    return rels, nil
}
```

### 3.3 Query Patterns for Polymorphic Lookups

The polymorphic design requires that callers resolve entity keys to IDs before calling the repository. The service layer owns this resolution logic.

**Pattern — key-to-ID resolution in the service:**

```go
// EntityResolver is a minimal interface for resolving entity keys to IDs.
// Each entity repository implements this by satisfying GetByKey.
type EntityResolver interface {
    GetByKey(ctx context.Context, key string) (models.Entity, error)
}
```

The service holds a map of `EntityType → EntityResolver` to support cross-entity lookups without type-switching in the repository.

---

## 4. Service Layer

### 4.1 `EntityRelationshipService` — Struct and Constructor

File: `internal/services/entity_relationship_service.go`

```go
package services

import (
    "context"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityLookupFunc resolves an entity key to an (EntityType, int64 ID) pair.
// The lookup function is provided by the caller and encapsulates the
// per-entity-type repository call.
type EntityLookupFunc func(ctx context.Context, key string) (models.EntityType, int64, error)

// RelationshipWithEntity is the service-layer DTO returned by relationship queries.
// It enriches raw EntityRelationship records with entity metadata for display.
type RelationshipWithEntity struct {
    RelationshipType string `json:"relationship_type"`
    Direction        string `json:"direction"` // "outgoing" | "incoming"
    EntityType       string `json:"entity_type"`
    EntityKey        string `json:"entity_key"`
    EntityTitle      string `json:"entity_title"`
    EntityStatus     string `json:"entity_status"`
}

// EntityRelationshipService manages polymorphic relationships between entities.
// It replaces TaskDependencyService for relationship operations and adds
// cross-entity capability.
type EntityRelationshipService struct {
    repo   EntityRelationshipRepository
    lookup EntityLookupFunc // resolves "E21-F07-001" → (EntityTypeTask, 42)
}

// NewEntityRelationshipService constructs an EntityRelationshipService.
//
// Parameters:
//   - repo: data access for entity_relationships table (required)
//   - lookup: function that resolves an entity key to its type+id (required)
func NewEntityRelationshipService(
    repo EntityRelationshipRepository,
    lookup EntityLookupFunc,
) *EntityRelationshipService {
    requireNonNil(repo, "EntityRelationshipService requires a non-nil repo")
    requireNonNil(lookup, "EntityRelationshipService requires a non-nil lookup function")
    return &EntityRelationshipService{
        repo:   repo,
        lookup: lookup,
    }
}
```

### 4.2 Core Service Methods

```go
// CreateRelationship creates a typed relationship between two entities identified by key.
// For depends_on and blocks within the same entity type, cycle detection is performed.
func (s *EntityRelationshipService) CreateRelationship(
    ctx context.Context,
    fromKey, toKey string,
    relType models.EntityRelationshipType,
) (*models.EntityRelationship, error) {
    fromEntityType, fromID, err := s.lookup(ctx, fromKey)
    if err != nil {
        return nil, fmt.Errorf("entity not found: %s: %w", fromKey, err)
    }

    toEntityType, toID, err := s.lookup(ctx, toKey)
    if err != nil {
        return nil, fmt.Errorf("entity not found: %s: %w", toKey, err)
    }

    rel := &models.EntityRelationship{
        FromEntityType:   fromEntityType,
        FromEntityID:     fromID,
        ToEntityType:     toEntityType,
        ToEntityID:       toID,
        RelationshipType: relType,
    }

    if err := rel.Validate(); err != nil {
        return nil, fmt.Errorf("invalid relationship: %w", err)
    }

    // Cycle detection: only for depends_on/blocks within the same entity type
    if rel.IsCyclic() {
        hasCycle, err := s.DetectCycle(ctx, fromEntityType, fromID, toEntityType, toID, relType)
        if err != nil {
            return nil, fmt.Errorf("cycle detection failed: %w", err)
        }
        if hasCycle {
            return nil, fmt.Errorf("circular dependency detected: %s → %s would create a cycle",
                fromKey, toKey)
        }
    }

    if err := s.repo.Create(ctx, rel); err != nil {
        return nil, fmt.Errorf("failed to create relationship %s -[%s]-> %s: %w",
            fromKey, relType, toKey, err)
    }

    return rel, nil
}

// RemoveRelationship removes a specific typed relationship between two entities.
func (s *EntityRelationshipService) RemoveRelationship(
    ctx context.Context,
    fromKey, toKey string,
    relType models.EntityRelationshipType,
) error {
    fromEntityType, fromID, err := s.lookup(ctx, fromKey)
    if err != nil {
        return fmt.Errorf("entity not found: %s: %w", fromKey, err)
    }

    toEntityType, toID, err := s.lookup(ctx, toKey)
    if err != nil {
        return fmt.Errorf("entity not found: %s: %w", toKey, err)
    }

    if err := s.repo.DeleteByEntitiesAndType(ctx,
        fromEntityType, fromID, toEntityType, toID, relType); err != nil {
        return fmt.Errorf("failed to remove relationship %s -[%s]-> %s: %w",
            fromKey, relType, toKey, err)
    }

    return nil
}

// GetRelationships retrieves all relationships for an entity, optionally filtered by type.
// Returns enriched DTOs with entity metadata. Metadata enrichment is done by the service.
func (s *EntityRelationshipService) GetRelationships(
    ctx context.Context,
    entityKey string,
    typeFilter []models.EntityRelationshipType,
) ([]RelationshipWithEntity, error) {
    entityType, entityID, err := s.lookup(ctx, entityKey)
    if err != nil {
        return nil, fmt.Errorf("entity not found: %s: %w", entityKey, err)
    }

    allRels, err := s.repo.GetByEntity(ctx, entityType, entityID)
    if err != nil {
        return nil, fmt.Errorf("failed to get relationships for %s: %w", entityKey, err)
    }

    var result []RelationshipWithEntity
    for _, rel := range allRels {
        // Apply type filter
        if len(typeFilter) > 0 {
            matched := false
            for _, ft := range typeFilter {
                if rel.RelationshipType == ft {
                    matched = true
                    break
                }
            }
            if !matched {
                continue
            }
        }

        direction := "outgoing"
        relatedType := rel.ToEntityType
        relatedID := rel.ToEntityID
        if rel.FromEntityType != entityType || rel.FromEntityID != entityID {
            direction = "incoming"
            relatedType = rel.FromEntityType
            relatedID = rel.FromEntityID
        }

        // Resolve the related entity's key and title
        relatedEntity, err := s.resolveEntityByTypeAndID(ctx, relatedType, relatedID)
        if err != nil {
            continue // Skip unresolvable entities; log in production
        }

        result = append(result, RelationshipWithEntity{
            RelationshipType: string(rel.RelationshipType),
            Direction:        direction,
            EntityType:       string(relatedType),
            EntityKey:        relatedEntity.GetKey(),
            EntityTitle:      relatedEntity.GetTitle(),
            EntityStatus:     relatedEntity.GetStatus(),
        })
    }

    return result, nil
}
```

### 4.3 Cycle Detection Algorithm

The generalized DFS cycle detection operates on the `entity_relationships` table. It is scoped to same-entity-type pairs only (per the research findings: cross-type cycles are not semantically meaningful).

```go
// DetectCycle reports whether creating a relationship from (fromType, fromID)
// to (toType, toID) would introduce a cycle in the dependency graph.
//
// Cycle detection is only performed when fromType == toType. Cross-entity
// cycles (e.g., Task depends_on Bug depends_on Task) are not detected by
// this method — they require a different policy decision and are out of scope.
//
// Returns (true, nil) if a cycle would be created, (false, nil) if safe,
// or (false, err) on lookup failure.
func (s *EntityRelationshipService) DetectCycle(
    ctx context.Context,
    fromType models.EntityType, fromID int64,
    toType models.EntityType, toID int64,
    relType models.EntityRelationshipType,
) (bool, error) {
    // Only check cycles within the same entity type
    if fromType != toType {
        return false, nil
    }

    // Direct self-link
    if fromID == toID {
        return true, nil
    }

    visited := make(map[int64]bool)
    // DFS from toID following relType edges; if we reach fromID, cycle exists
    return s.dfs(ctx, toType, toID, toType, fromID, relType, visited)
}

// dfs performs a depth-first search from (currentType, currentID) looking
// for (targetID). Returns true if targetID is reachable.
func (s *EntityRelationshipService) dfs(
    ctx context.Context,
    currentType models.EntityType, currentID int64,
    targetType models.EntityType, targetID int64,
    relType models.EntityRelationshipType,
    visited map[int64]bool,
) (bool, error) {
    if visited[currentID] {
        return false, nil
    }
    visited[currentID] = true

    outgoing, err := s.repo.GetOutgoing(ctx, currentType, currentID,
        []models.EntityRelationshipType{relType})
    if err != nil {
        return false, fmt.Errorf("dfs lookup failed for %s(%d): %w",
            currentType, currentID, err)
    }

    for _, rel := range outgoing {
        // Only follow edges to the same entity type
        if rel.ToEntityType != targetType {
            continue
        }
        if rel.ToEntityID == targetID {
            return true, nil // Cycle found
        }
        found, err := s.dfs(ctx, rel.ToEntityType, rel.ToEntityID,
            targetType, targetID, relType, visited)
        if err != nil {
            return false, err
        }
        if found {
            return true, nil
        }
    }

    return false, nil
}
```

**Complexity**: O(V + E) where V is entities and E is relationships reachable from the start. Bounded by the `entity_relationships` table size for the given entity type. Matches the existing `TaskRelationshipRepository.DetectCycle` semantics but operates on the new table.

### 4.4 Integration with EntityRegistry

If the `EntityRegistry` (E21-F09) provides a centralized entity lookup, the `EntityLookupFunc` is constructed from it:

```go
// BuildLookupFromRegistry creates an EntityLookupFunc backed by the EntityRegistry.
// This is the standard wiring for production use.
func BuildLookupFromRegistry(registry EntityRegistry) EntityLookupFunc {
    return func(ctx context.Context, key string) (models.EntityType, int64, error) {
        entity, err := registry.GetByKey(ctx, key)
        if err != nil {
            return "", 0, err
        }
        return entity.GetEntityType(), entity.GetID(), nil
    }
}
```

If the EntityRegistry is not yet available (phased delivery), the lookup function can be implemented inline using key pattern detection (E## → epic, E##-F## → feature, etc.) and direct repository lookups.

### 4.5 TaskDependencyService Backward Compatibility

`TaskDependencyService` is **not deleted** in the initial delivery. Instead, its methods that manage `task_relationships` are rerouted to `EntityRelationshipService`:

```go
// AddDependency delegates to EntityRelationshipService using the task entity type.
func (s *TaskDependencyService) AddDependency(ctx context.Context, taskKey, depKey string) error {
    _, err := s.relSvc.CreateRelationship(ctx, taskKey, depKey,
        models.EntityRelDependsOn)
    return err
}

// CreateTypedRelationship delegates to EntityRelationshipService.
func (s *TaskDependencyService) CreateTypedRelationship(
    ctx context.Context, taskKey, targetKey, relType string,
) (*models.Task, error) {
    _, err := s.relSvc.CreateRelationship(ctx, taskKey, targetKey,
        models.EntityRelationshipType(relType))
    if err != nil {
        return nil, err
    }
    return s.repo.GetByKey(ctx, targetKey)
}
```

This approach provides backward compatibility for all existing callers without requiring a flag day change across all CLI commands.

---

## 5. CLI Integration

### 5.1 Updated `shark task link` Command

The existing `task_link.go` requires no flag changes. The implementation is updated to delegate to `EntityRelationshipService` instead of `TaskDependencyService.CreateTypedRelationship`. The CLI surface stays identical.

```go
// runTaskLink — updated body (Step 2 only; Steps 1 and 3 unchanged)
relSvc := cli.GetEntityRelationshipService()
_, err := relSvc.CreateRelationship(ctx, taskKey, targetKey,
    models.EntityRelationshipType(relType))
```

### 5.2 New Top-Level `shark link` Command

File: `internal/cli/commands/link.go`

```go
// linkCmd creates typed relationships between any two entities.
// Entity types are auto-detected from key format.
var linkCmd = &cobra.Command{
    Use:   "link <from-key> <to-key> --type=<relationship-type>",
    Short: "Create a typed relationship between any two entities",
    Long: `Create typed relationships between any two entities (epics, features, tasks, bugs, or change-cards).
Entity type is auto-detected from key format.

Relationship Types:
  depends_on    Task/Feature/Epic depends on another completing
  blocks        Entity blocks another from proceeding
  related_to    Informational link between entities
  follows       Should be done after target
  spawned_from  Created as a result of another entity
  duplicates    Duplicate work
  references    Mentions or uses output of another
  linked_to     Informal context link (bugs)

Examples:
  shark link E21-F07-001 B001 --type=related_to
  shark link B001 E21-F07 --type=blocks
  shark link E21 E22 --type=related_to
  shark link E21-F07-001 E21-F07-002 --type=depends_on`,
    Args: cobra.ExactArgs(2),
    RunE: runLink,
}

func init() {
    linkCmd.Flags().String("type", "", "Relationship type (required)")
    _ = linkCmd.MarkFlagRequired("type")
    cli.RootCmd.AddCommand(linkCmd)
}

func runLink(cmd *cobra.Command, args []string) error {
    fromKey := args[0]
    toKey := args[1]
    relType, _ := cmd.Flags().GetString("type")

    svc := cli.GetEntityRelationshipService()
    rel, err := svc.CreateRelationship(
        cmd.Context(), fromKey, toKey,
        models.EntityRelationshipType(relType),
    )
    if err != nil {
        return fmt.Errorf("failed to create relationship: %w", err)
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(rel)
    }

    cli.Success(fmt.Sprintf("Created: %s -[%s]-> %s",
        fromKey, relType, toKey))
    return nil
}
```

### 5.3 Updated `shark task deps` Command

`task_deps.go` currently uses `svc.GetTaskRelationships()` from `TaskDependencyService`. After the migration, this is rerouted to `EntityRelationshipService.GetRelationships()`. The output format for `RelationshipWithEntity` replaces the existing `RelationshipWithTask` DTO. Fields are compatible; only the field name changes (`TaskKey` → `EntityKey`, `TaskTitle` → `EntityTitle`, `TaskStatus` → `EntityStatus`).

The existing `RelationshipRepositoryInterface` used in `buildDependencyTree` must also be updated to call `GetOutgoing` with the new signature:

```go
// Updated interface in task_deps.go
type RelationshipRepositoryInterface interface {
    GetOutgoing(ctx context.Context,
        entityType models.EntityType, entityID int64,
        relTypes []models.EntityRelationshipType,
    ) ([]*models.EntityRelationship, error)

    GetIncoming(ctx context.Context,
        entityType models.EntityType, entityID int64,
        relTypes []models.EntityRelationshipType,
    ) ([]*models.EntityRelationship, error)
}
```

### 5.4 New Top-Level `shark deps` Command

File: `internal/cli/commands/deps.go`

```go
// depsCmd shows all relationships for any entity (auto-detects type from key).
var depsCmd = &cobra.Command{
    Use:   "deps <entity-key>",
    Short: "Show all relationships for an entity",
    Args:  cobra.ExactArgs(1),
    RunE:  runDeps,
}

func init() {
    depsCmd.Flags().String("type", "", "Filter by relationship types (comma-separated)")
    cli.RootCmd.AddCommand(depsCmd)
}

func runDeps(cmd *cobra.Command, args []string) error {
    entityKey := args[0]
    typeFilterStr, _ := cmd.Flags().GetString("type")
    // ... parse typeFilterStr into []models.EntityRelationshipType ...

    svc := cli.GetEntityRelationshipService()
    rels, err := svc.GetRelationships(cmd.Context(), entityKey, typeFilter)
    if err != nil {
        return fmt.Errorf("failed to get relationships for %s: %w", entityKey, err)
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "entity_key":    entityKey,
            "relationships": rels,
        })
    }

    // Human-readable output
    // ... format similar to printTaskDeps but using EntityKey/EntityTitle ...
    return nil
}
```

### 5.5 Backward Compatibility for Entity-Specific Link Commands

All existing commands remain registered:
- `shark task link` — delegates to `EntityRelationshipService`
- `shark task deps` — delegates to `EntityRelationshipService`
- `shark task blocked-by` — delegates to `EntityRelationshipService`
- `shark task blocks` — delegates to `EntityRelationshipService`
- `shark task unlink` — delegates to `EntityRelationshipService`
- `shark feature next-status`, `shark epic next-status` — unchanged (no relationship queries)

The entity-specific commands can be documented as aliases for the top-level commands in a future docs pass.

### 5.6 Global Service Accessor

Added to `internal/cli/services_global.go`:

```go
// GetEntityRelationshipService returns an EntityRelationshipService wired
// with the global database and an auto-detecting entity lookup function.
func GetEntityRelationshipService() *services.EntityRelationshipService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    relRepo := repository.NewEntityRelationshipRepository(db)
    lookup := buildAutoDetectLookup(db)
    return services.NewEntityRelationshipService(relRepo, lookup)
}

// buildAutoDetectLookup creates an EntityLookupFunc that resolves entity keys
// using key-format detection (E## → epic, E##-F## → feature, etc.).
func buildAutoDetectLookup(db *repository.DB) services.EntityLookupFunc {
    epicRepo    := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo    := repository.NewTaskRepository(db)
    bugRepo     := repository.NewBugRepository(db)
    changeRepo  := repository.NewChangeCardRepository(db)

    return func(ctx context.Context, key string) (models.EntityType, int64, error) {
        // Key format auto-detection (same logic as existing core commands)
        switch detectEntityType(key) {
        case models.EntityTypeEpic:
            e, err := epicRepo.GetByKey(ctx, key)
            if err != nil { return "", 0, err }
            return models.EntityTypeEpic, e.ID, nil
        case models.EntityTypeFeature:
            f, err := featureRepo.GetByKey(ctx, key)
            if err != nil { return "", 0, err }
            return models.EntityTypeFeature, f.ID, nil
        case models.EntityTypeTask:
            t, err := taskRepo.GetByKey(ctx, key)
            if err != nil { return "", 0, err }
            return models.EntityTypeTask, t.ID, nil
        case models.EntityTypeBug:
            b, err := bugRepo.GetByKey(ctx, key)
            if err != nil { return "", 0, err }
            return models.EntityTypeBug, b.ID, nil
        case models.EntityTypeChange:
            c, err := changeRepo.GetByKey(ctx, key)
            if err != nil { return "", 0, err }
            return models.EntityTypeChange, c.ID, nil
        default:
            return "", 0, fmt.Errorf("cannot determine entity type from key: %s", key)
        }
    }
}
```

---

## 6. Migration Strategy

### 6.1 Phase 1: Schema + Task Relationships (Deliver First)

**Goal**: New table live, task relationship data migrated, task CLI unchanged.

1. Add `entity_relationships` table migration to `internal/db/db.go`
2. Bump `CurrentSchemaVersion`
3. Create `internal/models/entity_relationship.go`
4. Create `internal/repository/entity_relationship_repository.go`
5. Create `internal/services/entity_relationship_service.go`
6. Wire `EntityRelationshipService` into `TaskDependencyService` (delegation pattern)
7. Run data migration Phase 1 (task_relationships) and Phase 2 (depends_on JSON)
8. Update `task_dependency.go`'s `GetTaskDependents` to read from `entity_relationships`
9. Add `GetEntityRelationshipService()` to `services_global.go`
10. Verify: existing `shark task link`, `shark task deps` commands continue to work

**Risk**: Low. New table is additive. Old table still exists. Both systems work during transition.

### 6.2 Phase 2: Bug Linking Migration

**Goal**: Bug links moved to `entity_relationships`. `LinkedEntityType`/`LinkedEntityKey` columns deprecated.

1. Run data migration Phase 3 (bug linked_entity columns)
2. Update `BugRepository.Create/Update` to write to `entity_relationships` instead of flat columns
3. Update `bug_aggregate.go` to JOIN `entity_relationships` for linked entity display
4. Remove `LinkedEntityType`/`LinkedEntityKey` from `CreateBugInput` DTO
5. Update `bug.go` CLI to use `--linked-to <key>` flag still, but routed through `EntityRelationshipService`
6. Mark `linked_entity_type` / `linked_entity_key` columns as deprecated (do not drop yet)

**Risk**: Medium. `bug_aggregate.go` join query must be performance-tested. Add composite index on `entity_relationships(from_entity_type, from_entity_id)` before testing (already in schema).

### 6.3 Phase 3: ChangeCard FK Migration

**Goal**: ChangeCard FK links moved to `entity_relationships`. `ListByEpic`/`ListByFeature` use EXISTS subquery.

1. Run data migration Phase 4 (change_card FK columns)
2. Update `ChangeCardRepository.ListByEpic` to use EXISTS subquery on `entity_relationships`
3. Update `ChangeCardRepository.ListByFeature` similarly
4. Update `ChangeCardRepository.Create/Update` to write links via `EntityRelationshipService` after insert
5. Update `ChangeCardRepoFilter` to support key-based filtering
6. Keep `epic_id`/`feature_id`/`related_task_id` columns (they may serve hierarchy purposes per the feature PRD out-of-scope note); nullify them for new records after migration

**Revised ListByEpic query pattern:**

```sql
-- After migration: uses entity_relationships JOIN
SELECT cc.*
FROM change_cards cc
WHERE EXISTS (
    SELECT 1 FROM entity_relationships er
    WHERE er.from_entity_type = 'change'
      AND er.from_entity_id   = cc.id
      AND er.to_entity_type   = 'epic'
      AND er.to_entity_id     = ?   -- epic.id
)
```

**Risk**: High. Query plan must be verified. The composite index `idx_er_from` on `(from_entity_type, from_entity_id)` supports the EXISTS lookup efficiently.

### 6.4 Phase 4: Drop Legacy Tables

**Goal**: Remove `task_relationships`, `feature_relationships`, `epic_relationships`.

Prerequisites:
- All phases 1–3 verified in production
- No code references old tables
- Old data confirmed migrated (row count verification)

```sql
DROP TABLE IF EXISTS task_relationships;
DROP TABLE IF EXISTS feature_relationships;
DROP TABLE IF EXISTS epic_relationships;
```

Also remove the `depends_on` column from `tasks` after verifying no consumer reads it:

```sql
-- SQLite does not support DROP COLUMN before version 3.35.0
-- Use the table-rebuild pattern if needed:
ALTER TABLE tasks DROP COLUMN depends_on;
```

### 6.5 Rollback Plan

All phases use additive changes (new table + INSERT OR IGNORE). The old tables/columns are not dropped until Phase 4. If any phase fails:

1. **Phase 1 rollback**: Drop `entity_relationships` table. Revert `TaskDependencyService` to use `task_relationships` directly. Old table is intact.
2. **Phase 2 rollback**: Revert `BugRepository` to read flat columns. Old columns still present.
3. **Phase 3 rollback**: Revert `ChangeCardRepository` to use FK columns. Old FK columns still present.
4. **Phase 4 rollback**: Not applicable (table drops are non-reversible; restore from backup if needed).

**Backup recommendation**: Run `sqlite3 shark-tasks.db .dump > backup-pre-f11-migration.sql` before each phase.

---

## 7. Files to Create, Modify, and Deprecate

### 7.1 New Files to Create

| File | Purpose | Est. Lines |
|------|---------|------------|
| `internal/models/entity_relationship.go` | Polymorphic model, type enum, Validate() | ~70 |
| `internal/repository/entity_relationship_repository.go` | Unified CRUD + GetOutgoing/GetIncoming | ~250 |
| `internal/services/entity_relationship_service.go` | Service + cycle detection + interface defs | ~300 |
| `internal/cli/commands/link.go` | Top-level `shark link` command | ~80 |
| `internal/cli/commands/deps.go` | Top-level `shark deps` command | ~80 |
| `internal/repository/entity_relationship_repository_test.go` | Repository integration tests | ~400 |
| `internal/services/entity_relationship_service_test.go` | Service unit tests with mocks | ~300 |

### 7.2 Files to Modify Significantly

| File | Change Required | Risk |
|------|----------------|------|
| `internal/db/db.go` | Add migration function, bump `CurrentSchemaVersion` | Low |
| `internal/cli/services_global.go` | Add `GetEntityRelationshipService()` and `buildAutoDetectLookup()` | Low |
| `internal/services/task_dependency_service.go` | Route `CreateTypedRelationship`, `AddDependency`, `RemoveDependency` to `EntityRelationshipService` | Low |
| `internal/repository/task_dependency.go` | Update `GetTaskDependents` to query `entity_relationships` instead of `depends_on` JSON + `task_relationships` | Medium |
| `internal/cli/commands/task_deps.go` | Update `RelationshipRepositoryInterface` signature for new method signatures | Medium |
| `internal/cli/commands/task_link.go` | Route to `GetEntityRelationshipService()` | Low |
| `internal/cli/commands/task_unlink.go` | Route to `GetEntityRelationshipService()` | Low |
| `internal/repository/bug_repository.go` | Write links to `entity_relationships`, deprecate flat columns | Medium |
| `internal/repository/bug_aggregate.go` | JOIN `entity_relationships` for linked entity data | Medium |
| `internal/services/bug_service.go` | Use `EntityRelationshipService` for linking | Medium |
| `internal/services/bug_dto.go` | Remove `LinkedEntityType`/`LinkedEntityKey` (Phase 2) | Low |
| `internal/cli/commands/bug.go` | Route `--linked-to` to `EntityRelationshipService` | Low |
| `internal/repository/change_card_repository.go` | Update ListByEpic/ListByFeature to EXISTS subquery; write links post-insert | High |
| `internal/services/change_card_service.go` | Use `EntityRelationshipService` for link management | Medium |
| `internal/cli/commands/change.go` | Route `--epic`/`--feature`/`--task` flags to `EntityRelationshipService` | Low |
| `internal/config/template_helpers.go` | Update bug template rendering for linked entity from new DTO | Low |

### 7.3 Files to Deprecate and Delete (Phase 4)

| File | Lines | Replacement |
|------|-------|-------------|
| `internal/repository/task_relationship_repository.go` | 438 | `entity_relationship_repository.go` |
| `internal/repository/epic_relationship_repository.go` | 199 | `entity_relationship_repository.go` |
| `internal/repository/feature_relationship_repository.go` | 199 | `entity_relationship_repository.go` |
| `internal/models/task_relationship.go` | 57 | `entity_relationship.go` |
| `internal/models/epic_relationship.go` | 31 | `entity_relationship.go` |
| `internal/models/feature_relationship.go` | 31 | `entity_relationship.go` |

**Total code deleted**: ~955 lines

### 7.4 Test Files to Update

| File | Required Change |
|------|----------------|
| `internal/repository/task_relationship_repository_test.go` | Migrate tests to `entity_relationship_repository_test.go` |
| `internal/repository/relationship_repositories_test.go` | Migrate epic/feature tests to new repository test |
| `internal/services/task_dependency_service_test.go` (est.) | Update mocks to use `EntityRelationshipRepository` interface |
| `internal/repository/bug_repository_test.go` | Update for removed flat columns |
| `internal/repository/bug_aggregate_test.go` | Update for JOIN-based link query |
| `internal/services/bug_service_test.go` | Update mock expectations |
| `internal/services/change_card_service_test.go` | Update mock expectations |

---

## 8. Key Design Decisions

### Decision 1: Cycle Detection Scoped to Same Entity Type

**Context**: The feature PRD suggests generalizing cycle detection to cross-entity chains (Task → Bug → Task). The research report recommends keeping it scoped to same-type pairs.

**Decision**: Scope `DetectCycle` to same-entity-type pairs. The `rel.IsCyclic()` predicate returns `false` when `FromEntityType != ToEntityType`.

**Rationale**: Cross-entity cycles (e.g., Bug blocks Feature, Feature depends_on Task, Task spawned_from Bug) are semantically ambiguous and edge cases in practice. The complexity of cross-type cycle detection far exceeds the benefit at this stage. Same-type cycles (Task A depends_on Task B depends_on Task A) are the primary risk and are fully addressed.

### Decision 2: `linked_to` as a Distinct Relationship Type

**Context**: Bug's current "linked entity" is a single informal reference, not a structured dependency.

**Decision**: Introduce `linked_to` as an 8th relationship type, distinct from `related_to`.

**Rationale**: `linked_to` preserves the semantic distinction between a bug's informal "this bug is about entity X" link and a formal `related_to` relationship. It also makes queries unambiguous: `WHERE relationship_type = 'linked_to'` returns only bug-style context links.

### Decision 3: No Foreign Key Constraints on Polymorphic IDs

**Context**: SQLite cannot enforce a foreign key from `from_entity_id` to multiple parent tables.

**Decision**: Referential integrity for `from_entity_id` and `to_entity_id` is enforced at the service layer by looking up both entities before calling `repo.Create()`.

**Rationale**: This is the standard pattern for polymorphic associations in SQLite. The lookup-before-create approach is already used by `TaskDependencyService`.

### Decision 4: Phased Delivery, Not Flag Day

**Context**: 14+ files are affected. A single-PR migration carries high merge conflict and regression risk.

**Decision**: Deliver in 4 phases. Old tables/columns coexist with new table during transition.

**Rationale**: Each phase has a clear rollback strategy. Bug reporting and change-card linking can continue unchanged while Phase 1 is being validated.

### Decision 5: `TaskDependencyService` as Delegation Facade

**Context**: Many CLI commands and tests depend on `TaskDependencyService`.

**Decision**: Keep `TaskDependencyService` but have it delegate to `EntityRelationshipService` for relationship operations. Do not require callers to be updated immediately.

**Rationale**: This is the proven strangler fig pattern. `TaskDependencyService` becomes a thin facade that can be removed in a future cleanup once all callers have been updated.

---

## 9. Quality Checklist

Before marking Phase 1 complete:

- [ ] `entity_relationships` table created and indexed
- [ ] `CurrentSchemaVersion` bumped in `internal/db/db.go`
- [ ] `EntityRelationship` model validates correctly (unit tests)
- [ ] `EntityRelationshipRepository` all methods covered by integration tests with real DB
- [ ] `EntityRelationshipService` all methods covered by service unit tests with mocked repo
- [ ] Cycle detection tested: self-link, direct A→B→A, indirect A→B→C→A
- [ ] `shark task link` backward compatibility tested manually
- [ ] `shark task deps` backward compatibility tested manually
- [ ] `shark link` top-level command works for task→task and cross-entity
- [ ] `make fmt && make lint && make test` passes

Before marking Phase 4 (legacy cleanup) complete:

- [ ] All Phase 1–3 behaviors verified in production database
- [ ] Row counts confirmed via verification query
- [ ] Old relationship tables dropped without breaking any test
- [ ] `depends_on` column removed from `tasks` table
- [ ] `linked_entity_type`/`linked_entity_key` columns removed from `bugs` table
- [ ] `epic_id`/`feature_id`/`related_task_id` retained on `change_cards` (hierarchy use case per PRD out-of-scope note)
- [ ] All deprecated model/repository files deleted
- [ ] Full test suite passes: `make test`

---

*Architecture specification complete. Reviewed against feature PRD (E21-F11) and research report.*
