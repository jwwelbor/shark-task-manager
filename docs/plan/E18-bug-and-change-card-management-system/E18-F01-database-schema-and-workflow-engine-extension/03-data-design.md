# E18-F01 Data Design: Database Schema and Migrations

**Feature**: E18-F01
**Date**: 2026-03-03
**Status**: Draft

---

## 1. New Tables

### 1.1 Entity: bugs

**Description**: Stores bug reports as standalone entities with optional linking to epics, features, or tasks.

**Table Name**: `bugs`

**Pattern Source**: `ideas` table in `internal/db/db.go` (lines 337-370)

#### Schema

```sql
CREATE TABLE IF NOT EXISTS bugs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'reported',
    severity TEXT NOT NULL DEFAULT 'medium'
        CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    slug TEXT,
    linked_entity_type TEXT,
    linked_entity_key TEXT,
    context_data TEXT,
    file_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### Column Details

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT | Internal database identifier |
| key | TEXT | NOT NULL, UNIQUE | Bug key in B### format (e.g., B001, B002) |
| title | TEXT | NOT NULL | Human-readable bug title |
| status | TEXT | NOT NULL, DEFAULT 'reported' | Current workflow status |
| severity | TEXT | NOT NULL, DEFAULT 'medium', CHECK | Bug severity: critical, high, medium, low |
| slug | TEXT | - | Auto-generated slug from title for human-readable URLs |
| linked_entity_type | TEXT | - | Type of linked entity: 'epic', 'feature', 'task', or NULL |
| linked_entity_key | TEXT | - | Key of linked entity (e.g., E07, E07-F01, E07-F01-001), or NULL |
| context_data | TEXT | - | JSON blob for arbitrary metadata (environment, component, reproduction steps) |
| file_path | TEXT | - | Path to bug markdown file relative to project root |
| created_at | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Last modification timestamp |

#### Indexes

```sql
CREATE INDEX IF NOT EXISTS idx_bugs_key ON bugs(key);
CREATE INDEX IF NOT EXISTS idx_bugs_status ON bugs(status);
CREATE INDEX IF NOT EXISTS idx_bugs_severity ON bugs(severity);
CREATE INDEX IF NOT EXISTS idx_bugs_linked_entity_key ON bugs(linked_entity_key);
CREATE INDEX IF NOT EXISTS idx_bugs_slug ON bugs(slug);
```

#### Trigger

```sql
CREATE TRIGGER IF NOT EXISTS bugs_updated_at
    AFTER UPDATE ON bugs
    FOR EACH ROW
    BEGIN
        UPDATE bugs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;
```

#### Validation Rules

- `key`: Required, unique, format B### (e.g., B001). Validated at service layer.
- `title`: Required, minimum 1 character. Validated at model layer.
- `status`: Required, must be a valid bug workflow status. Validated at service layer via `workflow.Service.ForLevel("bug")`.
- `severity`: Required, must be one of: critical, high, medium, low. Enforced by CHECK constraint and service layer.
- `linked_entity_type`: Optional. If provided, must be a valid entity type. Validated at service layer (F02).
- `linked_entity_key`: Optional. If provided with `linked_entity_type`, the referenced entity must exist. Validated at service layer (F02).

#### Business Rules (Enforced in F02 Service Layer)

- Key generation: Auto-increment from max existing bug number + 1
- Soft delete not applicable (bugs use terminal statuses: resolved, wont_fix, duplicate)
- Link validation: Service layer verifies linked entity exists before saving
- context_data is free-form JSON; no schema enforcement at database level

---

### 1.2 Entity: change_cards

**Description**: Stores change requests/proposals as standalone entities with optional linking to epics, features, or tasks.

**Table Name**: `change_cards`

**Pattern Source**: Same as `bugs` table, but without `severity` column.

#### Schema

```sql
CREATE TABLE IF NOT EXISTS change_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'proposed',
    slug TEXT,
    linked_entity_type TEXT,
    linked_entity_key TEXT,
    context_data TEXT,
    file_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### Column Details

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT | Internal database identifier |
| key | TEXT | NOT NULL, UNIQUE | Change-card key in C### format (e.g., C001, C002) |
| title | TEXT | NOT NULL | Human-readable change-card title |
| status | TEXT | NOT NULL, DEFAULT 'proposed' | Current workflow status |
| slug | TEXT | - | Auto-generated slug from title |
| linked_entity_type | TEXT | - | Type of linked entity, or NULL |
| linked_entity_key | TEXT | - | Key of linked entity, or NULL |
| context_data | TEXT | - | JSON blob for arbitrary metadata (justification, impact description) |
| file_path | TEXT | - | Path to change-card markdown file relative to project root |
| created_at | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Last modification timestamp |

#### Indexes

```sql
CREATE INDEX IF NOT EXISTS idx_change_cards_key ON change_cards(key);
CREATE INDEX IF NOT EXISTS idx_change_cards_status ON change_cards(status);
CREATE INDEX IF NOT EXISTS idx_change_cards_linked_entity_key ON change_cards(linked_entity_key);
CREATE INDEX IF NOT EXISTS idx_change_cards_slug ON change_cards(slug);
```

#### Trigger

```sql
CREATE TRIGGER IF NOT EXISTS change_cards_updated_at
    AFTER UPDATE ON change_cards
    FOR EACH ROW
    BEGIN
        UPDATE change_cards SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;
```

#### Validation Rules

Same as bugs, except:
- `key`: Format C### (e.g., C001)
- `status`: Must be a valid change-card workflow status. Validated via `workflow.Service.ForLevel("change")`.
- No `severity` column

---

## 2. entity_notes Table Migration

### 2.1 Current Schema

```sql
CREATE TABLE entity_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('epic', 'feature', 'task')),
    entity_id INTEGER NOT NULL,
    note_type TEXT CHECK (note_type IN (
        'comment', 'decision', 'blocker', 'solution', 'reference',
        'implementation', 'testing', 'future', 'question', 'rejection'
    )) NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT
);
```

**Problem**: The CHECK constraint `entity_type IN ('epic', 'feature', 'task')` prevents inserting notes for bugs and change-cards.

### 2.2 Target Schema

```sql
CREATE TABLE entity_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,  -- CHECK constraint REMOVED
    entity_id INTEGER NOT NULL,
    note_type TEXT CHECK (note_type IN (
        'comment', 'decision', 'blocker', 'solution', 'reference',
        'implementation', 'testing', 'future', 'question', 'rejection'
    )) NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT
);
```

**Change**: Only the `entity_type` CHECK constraint is removed. All other constraints, columns, and types remain identical.

### 2.3 Migration Strategy

SQLite does not support `ALTER TABLE ... DROP CONSTRAINT`. The migration uses the standard table-recreation pattern.

```
Migration: migrateEntityNotesRemoveCheckConstraint

Precondition check:
  Query sqlite_master for the CREATE TABLE statement of entity_notes.
  If the statement does NOT contain "CHECK (entity_type IN", migration is already done.
  Return early (idempotent).

If CHECK constraint exists:
  1. BEGIN TRANSACTION
  2. CREATE TABLE entity_notes_new (
       -- same schema, but entity_type has no CHECK constraint
     )
  3. INSERT INTO entity_notes_new SELECT * FROM entity_notes
  4. DROP TABLE entity_notes
     -- This also drops all indexes and triggers on entity_notes
  5. ALTER TABLE entity_notes_new RENAME TO entity_notes
  6. Recreate indexes:
     - idx_entity_notes_type
     - idx_entity_notes_created_at
     - idx_entity_notes_entity_type
     - idx_entity_notes_type_entity
  7. Recreate cascade delete triggers:
     - entity_notes_cascade_delete_task
     - entity_notes_cascade_delete_feature
     - entity_notes_cascade_delete_epic
  8. COMMIT

On any error: ROLLBACK (deferred rollback via defer tx.Rollback())
```

### 2.4 Indexes to Recreate After Migration

```sql
CREATE INDEX IF NOT EXISTS idx_entity_notes_type ON entity_notes(note_type);
CREATE INDEX IF NOT EXISTS idx_entity_notes_created_at ON entity_notes(created_at);
CREATE INDEX IF NOT EXISTS idx_entity_notes_entity_type ON entity_notes(entity_type, entity_id, note_type);
CREATE INDEX IF NOT EXISTS idx_entity_notes_type_entity ON entity_notes(note_type, entity_type, entity_id);
```

### 2.5 Triggers to Recreate After Migration

```sql
CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_task
    AFTER DELETE ON tasks
    FOR EACH ROW
    BEGIN
        DELETE FROM entity_notes WHERE entity_type = 'task' AND entity_id = OLD.id;
    END;

CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_feature
    AFTER DELETE ON features
    FOR EACH ROW
    BEGIN
        DELETE FROM entity_notes WHERE entity_type = 'feature' AND entity_id = OLD.id;
    END;

CREATE TRIGGER IF NOT EXISTS entity_notes_cascade_delete_epic
    AFTER DELETE ON epics
    FOR EACH ROW
    BEGIN
        DELETE FROM entity_notes WHERE entity_type = 'epic' AND entity_id = OLD.id;
    END;
```

**Note**: Cascade delete triggers for bugs and change-cards are NOT added in F01. They will be added in F02/F03 when the bug/change-card repository delete operations are implemented.

### 2.6 Migration Safety

- **Transaction-protected**: All steps run within a single transaction
- **Rollback on failure**: `defer tx.Rollback()` ensures cleanup on any error
- **Idempotent**: Checks if CHECK constraint still exists before running
- **Data preservation**: `INSERT INTO ... SELECT * FROM` preserves all rows and values
- **Zero downtime**: Migration runs during `InitDB()` which is called on every shark command startup

---

## 3. Relationship to Existing Tables

### 3.1 No Foreign Keys to bugs/change_cards

The `bugs` and `change_cards` tables are **standalone entities** (like `ideas`). They do not have foreign key relationships to `epics`, `features`, or `tasks`.

The `linked_entity_type` + `linked_entity_key` columns provide **soft links** -- polymorphic references validated at the service layer, not enforced by database foreign keys. This is the same pattern used by the `ideas` table's `converted_to_type`/`converted_to_key` columns.

**Rationale**: SQLite foreign keys cannot reference polymorphic targets (a bug can link to an epic, feature, or task). Service-layer validation is the established pattern for this.

### 3.2 entity_notes Relationship

After the F01 migration, `entity_notes.entity_type` accepts any string value (validated at service layer). Bugs and change-cards use:
- `entity_type = 'bug'` with `entity_id` referencing `bugs.id`
- `entity_type = 'change'` with `entity_id` referencing `change_cards.id`

Cascade delete triggers for bugs and change-cards will be added in F02/F03:
```sql
-- Added in F02:
CREATE TRIGGER entity_notes_cascade_delete_bug
    AFTER DELETE ON bugs FOR EACH ROW BEGIN
        DELETE FROM entity_notes WHERE entity_type = 'bug' AND entity_id = OLD.id;
    END;

-- Added in F03:
CREATE TRIGGER entity_notes_cascade_delete_change_card
    AFTER DELETE ON change_cards FOR EACH ROW BEGIN
        DELETE FROM entity_notes WHERE entity_type = 'change' AND entity_id = OLD.id;
    END;
```

---

## 4. Schema Version

The current `SchemaVersion` in `internal/db/db.go` will be incremented to reflect the E18-F01 changes. The new tables and migration are idempotent, so the version serves as documentation, not as a migration gate.

---

## 5. Turso (Cloud Database) Compatibility

All SQL statements used in F01 are standard SQLite syntax compatible with Turso/libSQL:
- `CREATE TABLE IF NOT EXISTS`
- `CREATE INDEX IF NOT EXISTS`
- `CREATE TRIGGER IF NOT EXISTS`
- `ALTER TABLE ... RENAME TO`
- `CHECK` constraints
- `AUTOINCREMENT`

No Turso-specific changes are needed.

---

*Last Updated*: 2026-03-03
