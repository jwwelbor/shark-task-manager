# Data Models: E21-F08 Polymorphic Data Model Unification

**Feature**: E21-F08
**Author**: Architect
**Date**: 2026-03-20
**Status**: Draft

---

## 1. New Tables

### 1.1 `entity_documents` (Replaces 5 per-entity document join tables)

**Description**: Polymorphic join table linking any entity type to the `documents` table. Follows the `entity_notes` pattern.

**Replaces**: `epic_documents`, `feature_documents`, `task_documents`, `bug_documents`, `change_card_documents`

```sql
CREATE TABLE IF NOT EXISTS entity_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    link_type TEXT DEFAULT 'general',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_type, entity_id, document_id)
);

CREATE INDEX IF NOT EXISTS idx_entity_documents_lookup
    ON entity_documents(entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_entity_documents_document
    ON entity_documents(document_id);
```

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT | Unique identifier |
| entity_type | TEXT | NOT NULL | Entity type: 'epic', 'feature', 'task', 'bug', 'change' |
| entity_id | INTEGER | NOT NULL | ID of the entity in its respective table |
| document_id | INTEGER | NOT NULL, FK documents(id) ON DELETE CASCADE | ID of the linked document |
| link_type | TEXT | DEFAULT 'general' | Type of document link (e.g., 'general', 'specification', 'design') |
| created_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | When the link was created |

**Indexes**:
- `idx_entity_documents_lookup` on (entity_type, entity_id) -- primary lookup path
- `idx_entity_documents_document` on (document_id) -- for cascade delete and reverse lookups

**Unique Constraint**: (entity_type, entity_id, document_id) -- prevents duplicate links

**Validation Rules**:
- entity_type: Must be one of the values in `models.ValidEntityTypes`
- entity_id: Must be > 0
- document_id: Must reference an existing document (enforced by FK)

### 1.2 `entity_history` (Replaces `task_history`, extends to all entity types)

**Description**: Polymorphic status change audit trail for all entity types. Generalizes `task_history` to support Epic, Feature, Task, Bug, and ChangeCard.

**Replaces**: `task_history` (task-only)

```sql
CREATE TABLE IF NOT EXISTS entity_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    from_status TEXT,
    to_status TEXT NOT NULL,
    changed_by TEXT,
    notes TEXT,
    forced INTEGER NOT NULL DEFAULT 0,
    rejection_reason TEXT,
    changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_entity_history_lookup
    ON entity_history(entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_entity_history_time
    ON entity_history(changed_at);

CREATE INDEX IF NOT EXISTS idx_entity_history_entity_time
    ON entity_history(entity_type, entity_id, changed_at);
```

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT | Unique identifier |
| entity_type | TEXT | NOT NULL | Entity type: 'epic', 'feature', 'task', 'bug', 'change' |
| entity_id | INTEGER | NOT NULL | ID of the entity in its respective table |
| from_status | TEXT | (nullable) | Previous status (NULL for initial creation) |
| to_status | TEXT | NOT NULL | New status after transition |
| changed_by | TEXT | (nullable) | Agent ID or "user" for manual changes |
| notes | TEXT | (nullable) | Optional notes for the transition |
| forced | INTEGER | NOT NULL, DEFAULT 0 | 1 if transition was forced, 0 otherwise |
| rejection_reason | TEXT | (nullable) | Reason for backward/forced transition |
| changed_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | When the transition occurred |

**Indexes**:
- `idx_entity_history_lookup` on (entity_type, entity_id) -- primary lookup path
- `idx_entity_history_time` on (changed_at) -- for time-range queries
- `idx_entity_history_entity_time` on (entity_type, entity_id, changed_at) -- for entity-specific time-ordered queries

**Validation Rules**:
- entity_type: Must be one of the values in `models.ValidEntityTypes`
- entity_id: Must be > 0
- to_status: Must be non-empty
- from_status: Can be NULL (for initial status), otherwise non-empty

**Business Rules**:
- Every status transition creates exactly one record
- Records are append-only (never updated or deleted by the application)
- `forced=1` indicates the transition bypassed normal workflow validation
- `rejection_reason` is populated for backward or forced transitions when a reason is provided

---

## 2. Go Model Structs

### 2.1 `EntityHistory` Model

**Location**: `internal/models/entity_history.go`

```go
package models

import (
    "errors"
    "fmt"
    "time"
)

// EntityHistory represents a status change audit trail entry for any entity type.
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

// Validate validates the EntityHistory fields (structural validation only).
func (h *EntityHistory) Validate() error {
    if h.EntityType == "" {
        return errors.New("entity_type cannot be empty")
    }
    if !ValidEntityTypes[h.EntityType] {
        return fmt.Errorf("invalid entity_type: %s", h.EntityType)
    }
    if h.EntityID <= 0 {
        return errors.New("entity_id must be positive")
    }
    if h.ToStatus == "" {
        return errors.New("to_status cannot be empty")
    }
    return nil
}
```

**Key differences from `TaskHistory`**:

| Field | TaskHistory | EntityHistory |
|-------|-------------|---------------|
| Entity reference | `TaskID int64` | `EntityType EntityType` + `EntityID int64` |
| Status validation | `ValidateTaskStatus()` (task statuses only) | No status value validation (service layer validates via workflow) |
| Forced field | Not present in Go struct (DB has `forced BOOLEAN DEFAULT FALSE`) | `Forced bool` (explicit, non-nullable, aligns with DB `INTEGER NOT NULL DEFAULT 0`) |

### 2.2 `EntityDocument` Link Model (Optional)

The `entity_documents` table does not require a dedicated Go model struct because:
- The join table rows are created/queried by the repository
- The CLI/service layer works with `*models.Document` objects, not join records
- The `entity_type` and `entity_id` are method parameters, not struct fields

However, for migration verification and internal use, a lightweight struct may be useful:

```go
// EntityDocumentLink represents a row in the entity_documents join table.
// Used internally for migration verification and batch operations.
type EntityDocumentLink struct {
    ID         int64      `json:"id" db:"id"`
    EntityType EntityType `json:"entity_type" db:"entity_type"`
    EntityID   int64      `json:"entity_id" db:"entity_id"`
    DocumentID int64      `json:"document_id" db:"document_id"`
    LinkType   string     `json:"link_type" db:"link_type"`
    CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}
```

---

## 3. Repository Interfaces

### 3.1 EntityDocumentRepository Interface

**Location**: Defined at point of use in `internal/services/entity_document_service.go`

```go
// EntityDocumentLinkRepository provides polymorphic document linking operations.
// It replaces the 15 entity-specific Link/Unlink/List methods in DocumentRepository.
type EntityDocumentLinkRepository interface {
    // Link creates an association between an entity and a document.
    // Uses INSERT OR IGNORE to handle duplicate links gracefully.
    Link(ctx context.Context, entityType models.EntityType, entityID, documentID int64, linkType string) error

    // Unlink removes the association between an entity and a document.
    Unlink(ctx context.Context, entityType models.EntityType, entityID, documentID int64) error

    // ListForEntity retrieves all documents linked to a specific entity.
    // Returns documents ordered by created_at DESC.
    ListForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error)
}
```

### 3.2 EntityHistoryRepository Interface

**Location**: Defined at point of use in `internal/services/entity_service.go` (for recording) and `internal/services/entity_history_service.go` (for querying)

```go
// EntityHistoryRecorder is the minimal interface used by EntityService
// to record status change history during transitions.
type EntityHistoryRecorder interface {
    Create(ctx context.Context, history *models.EntityHistory) error
}

// EntityHistoryQuerier is the full query interface used by EntityHistoryService.
type EntityHistoryQuerier interface {
    EntityHistoryRecorder
    ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error)
    ListRecent(ctx context.Context, limit int) ([]*models.EntityHistory, error)
    ListWithFilters(ctx context.Context, filters EntityHistoryFilters) ([]*models.EntityHistory, error)
}

// EntityHistoryFilters defines filters for querying entity history.
type EntityHistoryFilters struct {
    EntityType *models.EntityType // Filter by entity type
    EntityID   *int64             // Filter by specific entity
    ChangedBy  *string            // Filter by agent/user
    Since      *time.Time         // Filter by timestamp (>= since)
    FromStatus *string            // Filter by previous status
    ToStatus   *string            // Filter by new status
    Limit      int                // Max records (default 50)
    Offset     int                // Pagination offset
}
```

---

## 4. Tables Being Dropped

After migration verification, the following tables are dropped:

| Table | Row Count Source | Migrated To |
|-------|-----------------|-------------|
| `epic_documents` | `SELECT COUNT(*) FROM epic_documents` | `entity_documents` with entity_type='epic' |
| `feature_documents` | `SELECT COUNT(*) FROM feature_documents` | `entity_documents` with entity_type='feature' |
| `task_documents` | `SELECT COUNT(*) FROM task_documents` | `entity_documents` with entity_type='task' |
| `bug_documents` | `SELECT COUNT(*) FROM bug_documents` | `entity_documents` with entity_type='bug' |
| `change_card_documents` | `SELECT COUNT(*) FROM change_card_documents` | `entity_documents` with entity_type='change' |
| `task_history` | `SELECT COUNT(*) FROM task_history` | `entity_history` with entity_type='task' |

---

## 5. Relationship Diagram

```
                    ┌──────────────┐
                    │   documents  │
                    │              │
                    │ id (PK)      │
                    │ title        │
                    │ file_path    │
                    │ created_at   │
                    └──────┬───────┘
                           │
                    FK: document_id
                           │
                ┌──────────┴───────────┐
                │  entity_documents    │
                │                      │
                │ id (PK)              │
                │ entity_type (TEXT)    │──── 'epic' | 'feature' | 'task' | 'bug' | 'change'
                │ entity_id (INTEGER)  │──── References epics.id / features.id / tasks.id / etc.
                │ document_id (FK)     │
                │ link_type (TEXT)     │
                │ created_at           │
                │ UNIQUE(type,id,doc)  │
                └──────────────────────┘


                ┌──────────────────────┐
                │  entity_history      │
                │                      │
                │ id (PK)              │
                │ entity_type (TEXT)    │──── 'epic' | 'feature' | 'task' | 'bug' | 'change'
                │ entity_id (INTEGER)  │──── References epics.id / features.id / tasks.id / etc.
                │ from_status (TEXT)    │
                │ to_status (TEXT)      │
                │ changed_by (TEXT)     │
                │ notes (TEXT)          │
                │ forced (INTEGER)      │
                │ rejection_reason      │
                │ changed_at            │
                └──────────────────────┘
```

**Note on FK integrity**: Like `entity_notes`, these polymorphic tables cannot have SQL-level foreign keys to multiple parent tables via `entity_type + entity_id`. Integrity is enforced at the application layer (repository validates entity existence before linking). This is the same approach used by `entity_notes` and is proven safe in this codebase.

---

*Last Updated*: 2026-03-20
