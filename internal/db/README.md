# Database Package

This package provides the complete database foundation for Shark Task Manager, including schema creation, SQLite configuration, and integrity checks.

## Overview

The database uses **SQLite 3** with the following structure:

```
epics (top-level projects)
  ↓ 1:N CASCADE DELETE
features (mid-level components)
  ↓ 1:N CASCADE DELETE
tasks (atomic work units)
  ↓ 1:N CASCADE DELETE
task_history (audit trail)
```

## Files

- **db.go** - Main database initialization and configuration

## Functions

### InitDB(filepath string) (*sql.DB, error)

Initializes the SQLite database with complete schema.

**Parameters:**
- `filepath` - Path to database file (e.g., "shark-tasks.db")

**Returns:**
- `*sql.DB` - Database connection
- `error` - Any initialization error

**What it does:**
1. Opens database connection with foreign keys enabled
2. Configures SQLite PRAGMAs for optimal performance
3. Creates all tables, indexes, and triggers
4. Verifies foreign key enforcement

**Example:**
```go
db, err := db.InitDB("shark-tasks.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### configureSQLite(db *sql.DB) error

Configures SQLite with optimal settings.

**PRAGMAs configured:**
- `foreign_keys = ON` - Enable referential integrity
- `journal_mode = WAL` - Write-Ahead Logging for better concurrency
- `busy_timeout = 5000` - 5 second timeout for locks
- `synchronous = NORMAL` - Balance safety and performance
- `cache_size = -64000` - 64MB cache
- `temp_store = MEMORY` - Store temp tables in memory
- `mmap_size = 30000000000` - Use memory-mapped I/O

### createSchema(db *sql.DB) error

Creates all database tables, indexes, and triggers.

**Tables created:**
1. **epics** - Top-level project organization
2. **features** - Mid-level components within epics
3. **tasks** - Atomic work units within features
4. **task_history** - Audit trail of task status changes

**Indexes created (10 total):**
- `idx_epics_key` - UNIQUE index on epic key
- `idx_epics_status` - Filter by epic status
- `idx_features_key` - UNIQUE index on feature key
- `idx_features_epic_id` - List features by epic
- `idx_features_status` - Filter by feature status
- `idx_tasks_key` - UNIQUE index on task key
- `idx_tasks_feature_id` - List tasks by feature
- `idx_tasks_status` - Filter by task status
- `idx_tasks_agent_type` - Filter by agent type
- `idx_tasks_status_priority` - Composite index for common queries
- `idx_tasks_priority` - Sort by priority
- `idx_task_history_task_id` - Task history lookup
- `idx_task_history_timestamp` - Recent history queries

**Triggers created (3 total):**
- `epics_updated_at` - Auto-update epic.updated_at on change
- `features_updated_at` - Auto-update feature.updated_at on change
- `tasks_updated_at` - Auto-update task.updated_at on change

### CheckIntegrity(db *sql.DB) error

Runs SQLite integrity check to verify database health.

**Example:**
```go
if err := db.CheckIntegrity(db); err != nil {
    log.Fatal("Database corrupted:", err)
}
```

## Schema Details

### Table: epics

Top-level project organization units.

**Columns:**
- `id` INTEGER PRIMARY KEY - Auto-increment ID
- `key` TEXT UNIQUE - Epic key (e.g., "E04")
- `title` TEXT NOT NULL - Epic title
- `description` TEXT - Markdown description
- `status` TEXT NOT NULL - draft|active|completed|archived
- `priority` TEXT NOT NULL - high|medium|low
- `business_value` TEXT - high|medium|low (optional)
- `created_at` TIMESTAMP - UTC creation time
- `updated_at` TIMESTAMP - Auto-updated modification time

**Constraints:**
- UNIQUE key
- CHECK status enum
- CHECK priority enum

### Table: features

Mid-level units within epics.

**Columns:**
- `id` INTEGER PRIMARY KEY - Auto-increment ID
- `epic_id` INTEGER NOT NULL - Foreign key to epics
- `key` TEXT UNIQUE - Feature key (e.g., "E04-F01")
- `title` TEXT NOT NULL - Feature title
- `description` TEXT - Markdown description
- `status` TEXT NOT NULL - draft|active|completed|archived
- `progress_pct` REAL - 0.0-100.0 (cached calculation)
- `created_at` TIMESTAMP - UTC creation time
- `updated_at` TIMESTAMP - Auto-updated modification time

**Constraints:**
- UNIQUE key
- FOREIGN KEY epic_id → epics(id) CASCADE DELETE
- CHECK status enum
- CHECK progress_pct range

### Table: tasks

Atomic work units within features.

**Columns:**
- `id` INTEGER PRIMARY KEY - Auto-increment ID
- `feature_id` INTEGER NOT NULL - Foreign key to features
- `key` TEXT UNIQUE - Task key (e.g., "T-E04-F01-001")
- `title` TEXT NOT NULL - Task title
- `description` TEXT - Markdown description
- `status` TEXT NOT NULL - todo|in_progress|blocked|ready_for_review|completed|archived
- `agent_type` TEXT - frontend|backend|api|testing|devops|general
- `priority` INTEGER - 1-10 (1=highest)
- `depends_on` TEXT - JSON array of task keys
- `assigned_agent` TEXT - Free text agent identifier
- `file_path` TEXT - Path to task markdown file
- `blocked_reason` TEXT - Reason for blocked status
- `created_at` TIMESTAMP - UTC creation time
- `started_at` TIMESTAMP - When status → in_progress
- `completed_at` TIMESTAMP - When status → completed
- `blocked_at` TIMESTAMP - When status → blocked
- `updated_at` TIMESTAMP - Auto-updated modification time

**Constraints:**
- UNIQUE key
- FOREIGN KEY feature_id → features(id) CASCADE DELETE
- CHECK status enum
- CHECK agent_type enum
- CHECK priority range 1-10

### Table: task_history

Audit trail of task status changes.

**Columns:**
- `id` INTEGER PRIMARY KEY - Auto-increment ID
- `task_id` INTEGER NOT NULL - Foreign key to tasks
- `old_status` TEXT - Previous status (NULL for creation)
- `new_status` TEXT NOT NULL - New status
- `agent` TEXT - Who made the change
- `notes` TEXT - Optional change notes
- `timestamp` TIMESTAMP - When change occurred

**Constraints:**
- FOREIGN KEY task_id → tasks(id) CASCADE DELETE

## Usage Examples

### Initialize Database

```go
import "github.com/jwwelbor/shark-task-manager/internal/db"

// Initialize with default settings
database, err := db.InitDB("shark-tasks.db")
if err != nil {
    log.Fatal("Failed to init database:", err)
}
defer database.Close()

// Verify integrity
if err := db.CheckIntegrity(database); err != nil {
    log.Fatal("Database corrupted:", err)
}
```

### In-Memory Database (Testing)

```go
// Use :memory: for fast in-memory testing
database, err := db.InitDB(":memory:")
```

### Custom Path

```go
// Specify custom path
database, err := db.InitDB("/var/data/tasks.db")
```

## Performance Characteristics

With the configured settings:

| Operation | Expected Time | Notes |
|-----------|---------------|-------|
| Database init | < 500ms | Includes schema creation |
| Single INSERT | < 50ms | With indexes |
| Query by key | < 10ms | Using unique index |
| Filtered query | < 100ms | With 10K tasks |
| Cascade DELETE | < 500ms | 1000 tasks |

## Error Handling

The package returns descriptive errors:

```go
db, err := db.InitDB("shark-tasks.db")
if err != nil {
    // Possible errors:
    // - "failed to open database: ..."
    // - "failed to ping database: ..."
    // - "failed to configure SQLite: ..."
    // - "failed to create schema: ..."
    // - "foreign_keys not enabled"
}
```

## Thread Safety

SQLite connections are thread-safe with these settings:
- WAL mode allows concurrent reads
- Single writer at a time (SQLite limitation)
- Connection pooling via `database/sql` package
- Busy timeout prevents immediate lock failures

## Maintenance

### Verify Schema

```sql
-- Check tables exist
SELECT name FROM sqlite_master WHERE type='table';

-- Check foreign keys enabled
PRAGMA foreign_keys;  -- Should return 1

-- Check WAL mode
PRAGMA journal_mode;  -- Should return 'wal'

-- Verify indexes
SELECT name FROM sqlite_master WHERE type='index';
```

### Database Integrity

```sql
-- Run integrity check
PRAGMA integrity_check;  -- Should return 'ok'

-- Check foreign key violations
PRAGMA foreign_key_check;  -- Should be empty
```

### Optimize (Periodic)

```sql
-- Reclaim unused space
VACUUM;

-- Rebuild indexes
REINDEX;

-- Analyze for query optimization
ANALYZE;
```

## Next Steps

After database initialization, use the repository layer:

1. **EpicRepository** - CRUD operations for epics
2. **FeatureRepository** - CRUD operations for features
3. **TaskRepository** - CRUD operations for tasks
4. **TaskHistoryRepository** - Query audit trail

See `internal/repository/` for details.
