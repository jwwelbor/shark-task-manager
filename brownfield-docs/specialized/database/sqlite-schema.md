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
    UNIQUE(from_task_id, to_task_id, relationship_type)
);
```

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
```

## Migration History

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
