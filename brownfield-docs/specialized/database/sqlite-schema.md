<<<<<<< Updated upstream
# SQLite Schema Documentation

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 9 — Specialized Documentation

## Schema Overview

- **Engine**: SQLite 3 (via `github.com/mattn/go-sqlite3`)
- **Schema Version**: 10 (`CurrentSchemaVersion` in `internal/db/db.go`)
- **Tables**: 16
- **Indexes**: 40+
- **Triggers**: 3+ (auto-timestamps)
- **WAL Mode**: Enabled for concurrent read/write

## SQLite Configuration (PRAGMAs)

```sql
PRAGMA foreign_keys = ON;          -- Enforce referential integrity
PRAGMA journal_mode = WAL;         -- Write-Ahead Logging
PRAGMA busy_timeout = 5000;        -- 5s lock wait timeout
PRAGMA synchronous = NORMAL;       -- Balance speed/durability
PRAGMA cache_size = -64000;        -- 64MB page cache
PRAGMA temp_store = MEMORY;        -- In-memory temp tables
PRAGMA mmap_size = 30000000000;    -- 30GB memory-mapped I/O
```

## Core Entity Tables

### `epics`
```sql
CREATE TABLE IF NOT EXISTS epics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    priority TEXT DEFAULT 'medium' CHECK(priority IN ('high', 'medium', 'low')),
    business_value INTEGER,
    file_path TEXT DEFAULT '',
    slug TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `features`
```sql
CREATE TABLE IF NOT EXISTS features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL REFERENCES epics(id) ON DELETE CASCADE,
    key TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    progress_pct REAL DEFAULT 0.0,
    execution_order INTEGER,
    file_path TEXT DEFAULT '',
    slug TEXT,
    status_override BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `tasks`
```sql
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feature_id INTEGER NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    key TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'todo',
    agent_type TEXT,
    priority INTEGER DEFAULT 5 CHECK(priority >= 1 AND priority <= 10),
    depends_on TEXT,
    assigned_agent TEXT,
    file_path TEXT DEFAULT '',
    slug TEXT,
    blocked_reason TEXT,
    execution_order INTEGER,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    blocked_at TIMESTAMP,
    completed_by TEXT,
    completion_notes TEXT,
    files_changed TEXT,
    tests_passed BOOLEAN,
    verification_status TEXT CHECK(verification_status IN ('pending', 'verified', 'needs_rework')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `bugs`
```sql
CREATE TABLE IF NOT EXISTS bugs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    priority INTEGER DEFAULT 5 CHECK(priority >= 1 AND priority <= 10),
    severity TEXT,
    linked_entity_type TEXT,
    linked_entity_key TEXT,
    context_data TEXT DEFAULT '{}',
    file_path TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `change_cards`
```sql
CREATE TABLE IF NOT EXISTS change_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    priority INTEGER DEFAULT 5 CHECK(priority >= 1 AND priority <= 10),
    linked_entity_type TEXT,
    linked_entity_key TEXT,
    context_data TEXT DEFAULT '{}',
    file_path TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## History & Audit Tables

### `entity_history` (polymorphic)
```sql
CREATE TABLE IF NOT EXISTS entity_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    from_status TEXT,
    to_status TEXT,
    changed_by TEXT,
    notes TEXT,
    forced INTEGER DEFAULT 0,
    rejection_reason TEXT,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `task_history` (legacy, retained for compatibility)
```sql
CREATE TABLE IF NOT EXISTS task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    old_status TEXT,
    new_status TEXT,
    agent TEXT,
    notes TEXT,
    forced BOOLEAN DEFAULT 0,
    rejection_reason TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Relationship Tables

### `entity_relationships`
```sql
CREATE TABLE IF NOT EXISTS entity_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_entity_type TEXT NOT NULL,
    from_entity_id INTEGER NOT NULL,
    to_entity_type TEXT NOT NULL,
    to_entity_id INTEGER NOT NULL,
    relationship_type TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
);
```

### `task_relationships` (legacy)
```sql
CREATE TABLE IF NOT EXISTS task_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    to_task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    relationship_type TEXT NOT NULL CHECK(relationship_type IN (
        'depends_on', 'blocks', 'related_to', 'follows',
        'spawned_from', 'duplicates', 'references'
    )),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
=======
# SQLite Database Schema

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 9 — Specialized Documentation

## Overview

| Property | Value |
|----------|-------|
| **Engine** | SQLite (mattn/go-sqlite3 v1.14.32) |
| **Cloud Option** | Turso (libsql-client-go) |
| **File** | `shark-tasks.db` |
| **Mode** | WAL (Write-Ahead Logging) |
| **Schema Version** | 6 |
| **Foreign Keys** | Enabled |
| **Cache** | 64MB in-memory |
| **mmap** | 30GB |

## SQLite PRAGMA Configuration

```sql
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;     -- 64MB
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 30000000000; -- 30GB
```

Source: `internal/db/db.go:43-52`

## Table Schema

### Core Entity Tables

#### `epics`
```sql
CREATE TABLE epics (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    key             TEXT NOT NULL UNIQUE,        -- E.g., "E07"
    title           TEXT NOT NULL,
    description     TEXT,
    status          TEXT NOT NULL,
    priority        TEXT NOT NULL CHECK (IN ('high', 'medium', 'low')),
    business_value  TEXT CHECK (IN ('high', 'medium', 'low')),
    file_path       TEXT,
    slug            TEXT,                        -- Added via migration
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- Trigger: auto-update updated_at on UPDATE
```

#### `features`
```sql
CREATE TABLE features (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id         INTEGER NOT NULL REFERENCES epics(id) ON DELETE CASCADE,
    key             TEXT NOT NULL UNIQUE,        -- E.g., "E07-F01"
    title           TEXT NOT NULL,
    description     TEXT,
    status          TEXT NOT NULL,
    progress_pct    REAL NOT NULL DEFAULT 0.0 CHECK (>= 0.0 AND <= 100.0),
    execution_order INTEGER NULL,
    file_path       TEXT,
    slug            TEXT,                        -- Added via migration
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- Trigger: auto-update updated_at on UPDATE
```

#### `tasks`
```sql
CREATE TABLE tasks (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    feature_id      INTEGER NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    key             TEXT NOT NULL UNIQUE,        -- E.g., "T-E07-F01-001"
    title           TEXT NOT NULL,
    description     TEXT,
    status          TEXT NOT NULL,
    agent_type      TEXT,
    priority        INTEGER NOT NULL DEFAULT 5 CHECK (>= 1 AND <= 10),
    depends_on      TEXT,                       -- Deprecated: use task_relationships
    assigned_agent  TEXT,
    file_path       TEXT,
    blocked_reason  TEXT,
    execution_order INTEGER NULL,
    slug            TEXT,                        -- Added via migration
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    blocked_at      TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- Trigger: auto-update updated_at on UPDATE
```

### Audit & History Tables

#### `task_history`
```sql
CREATE TABLE task_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id     INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    old_status  TEXT,
    new_status  TEXT NOT NULL,
    agent       TEXT,
    notes       TEXT,
    forced      BOOLEAN DEFAULT FALSE,
    timestamp   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### `task_notes`
```sql
CREATE TABLE task_notes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id     INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    note_type   TEXT NOT NULL CHECK (IN (
        'comment', 'decision', 'blocker', 'solution',
        'reference', 'implementation', 'testing',
        'future', 'question', 'rejection'
    )),
    content     TEXT NOT NULL,
    created_by  TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Relationship Tables

#### `task_relationships`
```sql
CREATE TABLE task_relationships (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id        INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    to_task_id          INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    relationship_type   TEXT NOT NULL CHECK (IN (
        'depends_on', 'blocks', 'related_to', 'follows',
        'spawned_from', 'duplicates', 'references'
    )),
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
>>>>>>> Stashed changes
    UNIQUE(from_task_id, to_task_id, relationship_type)
);
```

<<<<<<< Updated upstream
## Supporting Tables

### `entity_notes`
```sql
CREATE TABLE IF NOT EXISTS entity_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    note_type TEXT NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `task_notes` (legacy)
```sql
CREATE TABLE IF NOT EXISTS task_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    note_type TEXT CHECK(note_type IN (
        'comment', 'decision', 'blocker', 'solution', 'reference',
        'implementation', 'testing', 'future', 'question', 'rejection'
    )),
    content TEXT NOT NULL,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `documents` + `entity_documents`
```sql
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, file_path)
);

CREATE TABLE IF NOT EXISTS entity_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    link_type TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_type, entity_id, document_id)
);
```

### `ideas`
```sql
CREATE TABLE IF NOT EXISTS ideas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_date TEXT NOT NULL,
    priority INTEGER DEFAULT 5 CHECK(priority >= 1 AND priority <= 10),
    display_order INTEGER DEFAULT 0,
    notes TEXT DEFAULT '',
    related_docs TEXT DEFAULT '[]',
    dependencies TEXT DEFAULT '[]',
    status TEXT DEFAULT 'new' CHECK(status IN ('new', 'on_hold', 'converted', 'archived')),
    converted_to_type TEXT CHECK(converted_to_type IN ('epic', 'feature', 'task')),
    converted_to_key TEXT,
    converted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `work_sessions`
```sql
CREATE TABLE IF NOT EXISTS work_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    agent TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    duration INTEGER,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `schema_version`
```sql
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Triggers

```sql
-- Auto-update timestamps
CREATE TRIGGER IF NOT EXISTS epics_updated_at AFTER UPDATE ON epics
    FOR EACH ROW BEGIN UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER IF NOT EXISTS features_updated_at AFTER UPDATE ON features
    FOR EACH ROW BEGIN UPDATE features SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER IF NOT EXISTS tasks_updated_at AFTER UPDATE ON tasks
    FOR EACH ROW BEGIN UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER IF NOT EXISTS ideas_updated_at AFTER UPDATE ON ideas
    FOR EACH ROW BEGIN UPDATE ideas SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
=======
### Document Tables

#### `documents`
```sql
CREATE TABLE documents (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    file_path   TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, file_path)
);
```

#### Junction Tables: `epic_documents`, `feature_documents`, `task_documents`
```sql
-- Same pattern for all three:
CREATE TABLE {entity}_documents (
    {entity}_id     INTEGER NOT NULL REFERENCES {entities}(id) ON DELETE CASCADE,
    document_id     INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ({entity}_id, document_id)
);
```

### Ideas Table

#### `ideas`
```sql
CREATE TABLE ideas (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    key                 TEXT NOT NULL UNIQUE,    -- Format: I-YYYY-MM-DD-xx
    title               TEXT NOT NULL,
    description         TEXT,
    created_date        TIMESTAMP NOT NULL,
    priority            INTEGER CHECK (>= 1 AND <= 10),
    display_order       INTEGER,
    notes               TEXT,
    related_docs        TEXT,                   -- JSON array
    dependencies        TEXT,                   -- JSON array
    status              TEXT NOT NULL DEFAULT 'new' CHECK (IN ('new','on_hold','converted','archived')),
    converted_to_type   TEXT CHECK (IN ('epic','feature','task')),
    converted_to_key    TEXT,
    converted_at        TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- Trigger: auto-update updated_at on UPDATE
```

### Additional Tables (Added via Migrations)

- `entity_notes` — Polymorphic notes for any entity type
- `bugs` — Bug tracking entity
- `change_cards` — Change management entity
- `work_sessions` — Work session tracking
- `task_criteria` — Acceptance criteria
- `schema_version` — Migration version tracking

## Index Inventory

| Table | Index | Columns | Type |
|-------|-------|---------|------|
| epics | idx_epics_key | key | UNIQUE |
| epics | idx_epics_status | status | |
| epics | idx_epics_slug | slug | |
| features | idx_features_key | key | UNIQUE |
| features | idx_features_epic_id | epic_id | |
| features | idx_features_status | status | |
| features | idx_features_slug | slug | |
| tasks | idx_tasks_key | key | UNIQUE |
| tasks | idx_tasks_feature_id | feature_id | |
| tasks | idx_tasks_status | status | |
| tasks | idx_tasks_agent_type | agent_type | |
| tasks | idx_tasks_status_priority | status, priority | Composite |
| tasks | idx_tasks_priority | priority | |
| tasks | idx_tasks_file_path | file_path | |
| tasks | idx_tasks_slug | slug | |
| task_history | idx_task_history_task_id | task_id | |
| task_history | idx_task_history_timestamp | timestamp DESC | |
| task_notes | idx_task_notes_task_id | task_id | |
| task_notes | idx_task_notes_type | note_type | |
| task_notes | idx_task_notes_created_at | created_at | |
| task_notes | idx_task_notes_task_type | task_id, note_type | Composite |
| task_relationships | idx_...from | from_task_id | |
| task_relationships | idx_...to | to_task_id | |
| task_relationships | idx_...type | relationship_type | |
| task_relationships | idx_...from_type | from_task_id, relationship_type | Composite |
| task_relationships | idx_...to_type | to_task_id, relationship_type | Composite |

**Total indexes**: 25+ (including migration-added indexes)

## Triggers

| Trigger | Table | Event | Action |
|---------|-------|-------|--------|
| epics_updated_at | epics | AFTER UPDATE | Set updated_at = CURRENT_TIMESTAMP |
| features_updated_at | features | AFTER UPDATE | Set updated_at = CURRENT_TIMESTAMP |
| tasks_updated_at | tasks | AFTER UPDATE | Set updated_at = CURRENT_TIMESTAMP |
| ideas_updated_at | ideas | AFTER UPDATE | Set updated_at = CURRENT_TIMESTAMP |

## Entity-Relationship Diagram

```mermaid
erDiagram
    EPICS ||--o{ FEATURES : contains
    FEATURES ||--o{ TASKS : contains
    TASKS ||--o{ TASK_HISTORY : "has history"
    TASKS ||--o{ TASK_NOTES : "has notes"
    TASKS }o--o{ TASKS : "task_relationships"
    EPICS }o--o{ DOCUMENTS : "epic_documents"
    FEATURES }o--o{ DOCUMENTS : "feature_documents"
    TASKS }o--o{ DOCUMENTS : "task_documents"

    EPICS {
        int id PK
        text key UK
        text title
        text status
        text slug
    }

    FEATURES {
        int id PK
        int epic_id FK
        text key UK
        text title
        text status
        text slug
    }

    TASKS {
        int id PK
        int feature_id FK
        text key UK
        text title
        text status
        text agent_type
        int priority
        text slug
    }

    TASK_HISTORY {
        int id PK
        int task_id FK
        text old_status
        text new_status
        text agent
    }

    TASK_NOTES {
        int id PK
        int task_id FK
        text note_type
        text content
    }

    DOCUMENTS {
        int id PK
        text title
        text file_path
    }

    IDEAS {
        int id PK
        text key UK
        text title
        text status
        text converted_to_type
    }
>>>>>>> Stashed changes
```

## Migration History

<<<<<<< Updated upstream
| Version | Changes |
|---------|---------|
| 1 | Initial schema (epics, features, tasks, task_history) |
| 2 | Add slug columns |
| 3 | Add task_notes, task_relationships |
| 4 | Add documents, entity_documents |
| 5 | Add ideas table |
| 6 | Add execution_order to features and tasks |
| 7 | Add entity_history (polymorphic) |
| 8 | Add bugs, change_cards tables |
| 9 | Add entity_relationships, entity_notes |
| 10 | Add work_sessions, verification fields |

## Data Access Patterns

| Pattern | SQL | Used By |
|---------|-----|---------|
| Get by key | `SELECT * FROM tasks WHERE UPPER(key) = UPPER(?)` | All get commands |
| Get by slug | `SELECT * FROM tasks WHERE UPPER(key) = UPPER(?) AND slug = ?` | Slug lookups |
| List by parent | `SELECT * FROM tasks WHERE feature_id = ? ORDER BY execution_order` | List commands |
| Status update | `UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?` | Status commands |
| Progress calc | `SELECT status, COUNT(*) FROM tasks WHERE feature_id = ? GROUP BY status` | Progress |
| History query | `SELECT * FROM entity_history WHERE entity_type = ? AND entity_id = ? ORDER BY changed_at DESC` | History |

See also: [Data Models](../reference/data-models.md) | [Dependencies](../architecture/dependencies.md)
=======
Schema version 6 includes the following migration phases:
1. **Base schema** — Core tables (epics, features, tasks, task_history, task_notes, task_relationships, documents, ideas)
2. **Slug support** — Added `slug` column to epics, features, tasks
3. **Entity notes** — Added polymorphic `entity_notes` table
4. **Bugs & Change Cards** — Added `bugs` and `change_cards` tables
5. **Work sessions & Criteria** — Added `work_sessions` and `task_criteria` tables
6. **File path & execution order** — Added columns to various tables

Migrations are idempotent — safe to re-run. Version checking via `schema_version` table skips DDL when already current.

Source: `internal/db/db.go`, `internal/db/migrate.go`, `internal/db/migrate_slug_backfill.go`

---

See also: [Architecture Dependencies](../../architecture/dependencies.md) | [Data Models](../../reference/data-models.md)
>>>>>>> Stashed changes
