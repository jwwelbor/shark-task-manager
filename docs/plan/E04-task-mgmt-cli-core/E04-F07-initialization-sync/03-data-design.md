# Data Design: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-16
**Author**: db-admin

## Purpose

This document defines data storage and schema changes for sync metadata tracking. It describes how the system tracks synchronization state, detects conflicts, and maintains consistency between filesystem and database.

---

## Data Architecture Decision

### Approach: Minimal Schema Changes

**Decision**: Use existing database schema without adding sync-specific tables.

**Rationale**:
1. The existing `tasks` table already has `file_path` and `updated_at` fields
2. File modified timestamps can be read from filesystem
3. Conflict detection can be done in-memory during sync
4. Sync history can be tracked via existing `task_history` table
5. Reduces schema complexity and migration overhead

**What We Have**:
- `tasks.file_path` - Stores current file location (used for conflict detection)
- `tasks.updated_at` - Stores last database update time (used for newer-wins strategy)
- `task_history` - Already tracks task changes with timestamps and notes

**What We DON'T Need**:
- No new `sync_metadata` table
- No `last_synced_at` column
- No `content_hash` column (compare values directly)

---

## Existing Schema (Relevant Parts)

### Tasks Table (from E04-F01)

```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feature_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived')),
    agent_type TEXT CHECK (agent_type IN ('frontend', 'backend', 'api', 'testing', 'devops', 'general')),
    priority INTEGER NOT NULL DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    depends_on TEXT,           -- JSON array of task keys
    assigned_agent TEXT,
    file_path TEXT,            -- *** CRITICAL FOR SYNC: Stores actual file location ***
    blocked_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    blocked_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- *** CRITICAL FOR SYNC: Used for newer-wins ***

    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
);

-- Existing indexes
CREATE UNIQUE INDEX idx_tasks_key ON tasks(key);
CREATE INDEX idx_tasks_feature_id ON tasks(feature_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_agent_type ON tasks(agent_type);
CREATE INDEX idx_tasks_status_priority ON tasks(status, priority);
CREATE INDEX idx_tasks_priority ON tasks(priority);

-- Automatic updated_at trigger
CREATE TRIGGER tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### Task History Table (for sync audit trail)

```sql
CREATE TABLE task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    old_status TEXT,
    new_status TEXT NOT NULL,
    agent TEXT,
    notes TEXT,                -- *** SYNC WILL USE THIS: "Imported from file", "Updated from file during sync" ***
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_history_task_id ON task_history(task_id);
CREATE INDEX idx_task_history_timestamp ON task_history(timestamp DESC);
```

### Epics and Features Tables (for --create-missing flag)

```sql
CREATE TABLE epics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived')),
    priority TEXT NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    business_value TEXT CHECK (business_value IN ('high', 'medium', 'low')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived')),
    progress_pct REAL NOT NULL DEFAULT 0.0 CHECK (progress_pct >= 0.0 AND progress_pct <= 100.0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
);
```

---

## Data Flow During Sync

### Sync Operation Data Flow

```
┌──────────────────────────────────────────────────────────────────┐
│ 1. FILE SCANNING                                                  │
│                                                                    │
│  Filesystem                                                        │
│    ├── docs/plan/E04-epic/E04-F07-feature/T-E04-F07-001.md       │
│    └── File Modified Time: 2025-12-16 10:00:00                   │
│                                                                    │
│  Scan Result:                                                      │
│    - File Path: /abs/path/to/T-E04-F07-001.md                    │
│    - Modified At: 2025-12-16 10:00:00                            │
│    - Inferred Epic: E04                                           │
│    - Inferred Feature: E04-F07                                    │
└────────────────────────────┬───────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│ 2. FRONTMATTER PARSING                                            │
│                                                                    │
│  Parse YAML:                                                       │
│    key: T-E04-F07-001                                             │
│    title: "Implement sync engine"                                 │
│    description: "Core sync orchestration logic"                   │
│    file_path: docs/plan/.../T-E04-F07-001.md  (optional)         │
│                                                                    │
│  TaskMetadata:                                                     │
│    - Key: T-E04-F07-001                                           │
│    - Title: "Implement sync engine"                               │
│    - Description: "Core sync orchestration logic"                 │
│    - FilePath: docs/plan/.../T-E04-F07-001.md                    │
│    - ModifiedAt: 2025-12-16 10:00:00                             │
└────────────────────────────┬───────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│ 3. DATABASE QUERY                                                 │
│                                                                    │
│  SELECT * FROM tasks WHERE key = 'T-E04-F07-001';                │
│                                                                    │
│  Result:                                                           │
│    - ID: 42                                                        │
│    - Key: T-E04-F07-001                                           │
│    - Title: "Add sync engine"  *** CONFLICT ***                  │
│    - Description: "Core sync orchestration logic"  (same)         │
│    - FilePath: docs/tasks/created/T-E04-F07-001.md  *** CONFLICT │
│    - UpdatedAt: 2025-12-15 09:00:00                              │
│    - Status: in_progress  (DATABASE ONLY - not in file)          │
└────────────────────────────┬───────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│ 4. CONFLICT DETECTION                                             │
│                                                                    │
│  Compare:                                                          │
│    title:       "Implement sync engine" (file) vs                │
│                 "Add sync engine" (database)  → CONFLICT          │
│    description: (same) → NO CONFLICT                              │
│    file_path:   /abs/.../T-E04-F07-001.md (actual) vs            │
│                 docs/tasks/created/T-E04-F07-001.md (db)          │
│                 → CONFLICT (file moved)                           │
│                                                                    │
│  Conflicts Detected:                                               │
│    1. Field: title, DB: "Add...", File: "Implement..."           │
│    2. Field: file_path, DB: "docs/tasks/...", File: "docs/plan..." │
└────────────────────────────┬───────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│ 5. CONFLICT RESOLUTION (Strategy: file-wins)                     │
│                                                                    │
│  Apply Resolution:                                                 │
│    title: Use file value → "Implement sync engine"               │
│    file_path: Use actual location → /abs/.../T-E04-F07-001.md   │
│                                                                    │
│  Preserve Database-Only Fields:                                   │
│    status: in_progress (unchanged)                                │
│    priority: 5 (unchanged)                                        │
│    agent_type: backend (unchanged)                                │
│    depends_on: [...] (unchanged)                                  │
└────────────────────────────┬───────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│ 6. DATABASE UPDATE (Transaction)                                 │
│                                                                    │
│  BEGIN TRANSACTION;                                                │
│                                                                    │
│    UPDATE tasks                                                    │
│    SET title = 'Implement sync engine',                          │
│        file_path = '/abs/.../T-E04-F07-001.md'                   │
│    WHERE id = 42;                                                 │
│    -- updated_at automatically set by trigger                     │
│                                                                    │
│    INSERT INTO task_history (task_id, old_status, new_status,    │
│                              agent, notes)                         │
│    VALUES (42, 'in_progress', 'in_progress', 'sync',             │
│            'Updated from file during sync: title, file_path');    │
│                                                                    │
│  COMMIT;                                                           │
└──────────────────────────────────────────────────────────────────┘
```

---

## Sync Metadata Tracking Strategy

### Approach 1: Use Existing Fields (CHOSEN)

**Implementation**:
- Use `tasks.updated_at` for database modification time
- Read file modified time from filesystem (os.Stat)
- Compare timestamps for "newer-wins" strategy
- Store sync events in `task_history` table with notes

**Advantages**:
- No schema changes required
- Leverages existing automatic triggers
- Simple and maintainable
- Sufficient for all PRD requirements

**Disadvantages**:
- No explicit "last sync time" field
- Cannot track content hash (rely on direct comparison)

### Approach 2: Add Sync Metadata Table (REJECTED)

**Why Rejected**:
- Adds complexity without clear benefit
- All required information available from existing schema
- Sync is not frequent enough to justify caching
- Would require additional maintenance and queries

---

## Conflict Detection Algorithm

### Data Structures for Conflict Detection

**In-Memory Comparison** (no persistent storage needed):

```go
type ConflictDetectionData struct {
    TaskKey       string
    FileData      TaskFileData
    DatabaseData  TaskDatabaseData
}

type TaskFileData struct {
    Title       string
    Description *string
    FilePath    string      // Actual file path
    ModifiedAt  time.Time   // From os.Stat(file).ModTime()
}

type TaskDatabaseData struct {
    ID          int64
    Title       string
    Description *string
    FilePath    *string     // Stored path in database
    UpdatedAt   time.Time   // From tasks.updated_at
    Status      string      // DATABASE ONLY - preserved during sync
    Priority    int         // DATABASE ONLY - preserved
    AgentType   *string     // DATABASE ONLY - preserved
}
```

### Conflict Detection Logic

```go
func DetectConflicts(fileData TaskFileData, dbData TaskDatabaseData) []Conflict {
    conflicts := []Conflict{}

    // 1. Title conflict
    if fileData.Title != "" && fileData.Title != dbData.Title {
        conflicts = append(conflicts, Conflict{
            Field:         "title",
            FileValue:     fileData.Title,
            DatabaseValue: dbData.Title,
        })
    }

    // 2. Description conflict
    if fileData.Description != nil && *fileData.Description != *dbData.Description {
        conflicts = append(conflicts, Conflict{
            Field:         "description",
            FileValue:     *fileData.Description,
            DatabaseValue: *dbData.Description,
        })
    }

    // 3. File path conflict (always update to actual location)
    if dbData.FilePath == nil || *dbData.FilePath != fileData.FilePath {
        conflicts = append(conflicts, Conflict{
            Field:         "file_path",
            FileValue:     fileData.FilePath,
            DatabaseValue: *dbData.FilePath,
        })
    }

    return conflicts
}
```

### Resolution Strategy Application

**File-Wins Strategy**:
```sql
UPDATE tasks
SET title = ?,              -- Use file value
    description = ?,        -- Use file value
    file_path = ?           -- Use actual file location
    -- updated_at automatically updated by trigger
WHERE key = ?;
```

**Database-Wins Strategy**:
```sql
-- No UPDATE needed
-- Database values are authoritative
-- Optionally update file if --update-files flag set
```

**Newer-Wins Strategy**:
```go
if fileData.ModifiedAt.After(dbData.UpdatedAt) {
    // File is newer → use file-wins logic
} else {
    // Database is newer → use database-wins logic
}
```

---

## Data Validation Rules

### Frontmatter Validation

**Required Fields**:
- `key` (string, pattern: `T-E##-F##-###`)

**Optional Fields**:
- `title` (string)
- `description` (string)
- `file_path` (string)

**Forbidden Fields in Frontmatter** (from PRD architectural decision):
- `status` - MUST NOT be in file (database only)
- `priority` - MUST NOT be in file (database only)
- `agent_type` - MUST NOT be in file (database only)
- `depends_on` - MUST NOT be in file (database only)

**Validation Logic**:
```go
func ValidateFrontmatter(metadata TaskMetadata) error {
    // Required: key
    if metadata.Key == "" {
        return errors.New("task_key is required")
    }

    // Validate key format
    if !regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`).MatchString(metadata.Key) {
        return fmt.Errorf("invalid task key format: %s", metadata.Key)
    }

    // Optional fields - no validation if empty
    // Title and description can be empty (preserve database values)

    return nil
}
```

### Key Consistency Validation

**Rule**: Task key in frontmatter MUST match key in filename.

**Validation**:
```go
// File: T-E04-F07-001.md
// Frontmatter: key: T-E04-F07-001
// → VALID

// File: T-E04-F07-001.md
// Frontmatter: key: T-E04-F07-002
// → INVALID (log warning, skip file)
```

**Implementation**:
```go
func ValidateKeyConsistency(filename, frontmatterKey string) error {
    // Extract key from filename
    expectedKey := strings.TrimSuffix(filename, ".md")

    if expectedKey != frontmatterKey {
        return fmt.Errorf("key mismatch: filename=%s, frontmatter=%s",
            expectedKey, frontmatterKey)
    }

    return nil
}
```

---

## Epic and Feature Management

### Auto-Creation Strategy (--create-missing flag)

**Data Sources for Inference**:
1. **From folder structure**: `docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/`
   - Epic: E04 (from grandparent directory)
   - Feature: E04-F07 (from parent directory)

2. **From task key**: `T-E04-F07-001`
   - Epic: E04 (extract from key)
   - Feature: E04-F07 (extract from key)

**Auto-Creation Data**:
```go
// Minimal epic creation
epic := &models.Epic{
    Key:          "E04",                        // From inference
    Title:        "E04 (Auto-created)",         // Placeholder
    Description:  "Auto-created during sync",   // Note auto-creation
    Status:       "active",                     // Default
    Priority:     "medium",                     // Default
}

// Minimal feature creation
feature := &models.Feature{
    EpicID:       epicID,                       // From epic lookup
    Key:          "E04-F07",                    // From inference
    Title:        "E04-F07 (Auto-created)",     // Placeholder
    Description:  "Auto-created during sync",   // Note auto-creation
    Status:       "active",                     // Default
    ProgressPct:  0.0,                          // Default
}
```

**Database Operations**:
```sql
-- 1. Check if epic exists
SELECT id FROM epics WHERE key = 'E04';

-- 2. If not exists and --create-missing, create epic
INSERT INTO epics (key, title, description, status, priority)
VALUES ('E04', 'E04 (Auto-created)', 'Auto-created during sync', 'active', 'medium');

-- 3. Check if feature exists
SELECT id FROM features WHERE key = 'E04-F07';

-- 4. If not exists and --create-missing, create feature
INSERT INTO features (epic_id, key, title, description, status, progress_pct)
VALUES (?, 'E04-F07', 'E04-F07 (Auto-created)', 'Auto-created during sync', 'active', 0.0);
```

**Without --create-missing Flag**:
```go
// If epic or feature doesn't exist, skip task and log warning
log.Warnf("Task %s references non-existent feature %s. Use --create-missing or create feature first.",
    taskKey, featureKey)
```

---

## Task History Tracking

### Sync Events Recorded

**Event Types**:
1. **Task Import**: New task imported from file
2. **Task Update**: Existing task updated from file
3. **Conflict Resolution**: Conflict detected and resolved

**History Record Format**:
```go
type TaskHistoryRecord struct {
    TaskID    int64
    OldStatus string      // Same as new status (status not changed during sync)
    NewStatus string      // Current status
    Agent     *string     // "sync" (identifies sync operation)
    Notes     *string     // Detailed description of sync action
    Timestamp time.Time   // Automatic
}
```

**Example History Records**:

**1. Task Import**:
```sql
INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
VALUES (42, NULL, 'todo', 'sync', 'Task imported from file: docs/plan/.../T-E04-F07-001.md');
```

**2. Task Update (No Conflicts)**:
```sql
INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
VALUES (42, 'in_progress', 'in_progress', 'sync', 'File path updated to: docs/plan/.../T-E04-F07-001.md');
```

**3. Task Update (With Conflicts)**:
```sql
INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
VALUES (42, 'in_progress', 'in_progress', 'sync',
        'Updated from file during sync (file-wins): title ("Add..." → "Implement..."), file_path');
```

**4. Task Deletion (--cleanup flag)**:
```sql
-- History record automatically deleted via CASCADE
-- But we can add a record before deleting task:
INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
VALUES (42, 'completed', 'archived', 'sync', 'Task file not found, marked for deletion');

DELETE FROM tasks WHERE id = 42;
-- Cascade deletes history
```

---

## Query Patterns for Sync

### Efficient Bulk Queries

**1. Bulk Task Lookup by Keys**:
```sql
-- Fetch all tasks matching scanned file keys
SELECT id, key, title, description, file_path, updated_at, status, priority, agent_type
FROM tasks
WHERE key IN (?, ?, ?, ...);  -- Bind array of keys
```

**2. Epic/Feature Existence Check**:
```sql
-- Check multiple epics/features in one query
SELECT key FROM epics WHERE key IN (?, ?, ?);
SELECT key FROM features WHERE key IN (?, ?, ?);
```

**3. Orphaned Task Detection (--cleanup flag)**:
```sql
-- Find tasks with file_path not in scanned files list
SELECT id, key, file_path
FROM tasks
WHERE file_path NOT IN (?, ?, ?, ...);  -- Bind array of scanned file paths
```

### Transaction Pattern for Sync

```sql
BEGIN TRANSACTION;

-- 1. Create missing epics (if --create-missing)
INSERT INTO epics (...) VALUES (...);

-- 2. Create missing features (if --create-missing)
INSERT INTO features (...) VALUES (...);

-- 3. Bulk insert new tasks (using prepared statement)
INSERT INTO tasks (...) VALUES (?);  -- Execute for each new task

-- 4. Update existing tasks (using prepared statement)
UPDATE tasks SET title = ?, description = ?, file_path = ? WHERE key = ?;

-- 5. Create history records
INSERT INTO task_history (...) VALUES (?);  -- Execute for each task

-- 6. Delete orphaned tasks (if --cleanup)
DELETE FROM tasks WHERE id IN (?);

COMMIT;  -- Or ROLLBACK on any error
```

---

## Data Integrity Constraints

### Enforced Constraints (Existing)

1. **Foreign Keys**:
   - `tasks.feature_id` REFERENCES `features.id` ON DELETE CASCADE
   - `features.epic_id` REFERENCES `epics.id` ON DELETE CASCADE
   - Ensures tasks cannot be orphaned

2. **Unique Keys**:
   - `tasks.key` UNIQUE
   - `features.key` UNIQUE
   - `epics.key` UNIQUE
   - Prevents duplicate keys during bulk import

3. **Check Constraints**:
   - `tasks.status` IN ('todo', 'in_progress', ...)
   - `tasks.priority` BETWEEN 1 AND 10
   - Validates data during import

### Sync-Specific Constraints

**1. Key Format Validation** (application-level):
```go
// Regex: T-E##-F##-###
validTaskKey := regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`)
```

**2. File Path Validation** (application-level):
```go
// Ensure file path is within allowed directories
func ValidateFilePath(path string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }

    allowedRoots := []string{
        "/path/to/docs/plan",
        "/path/to/docs/tasks",
    }

    for _, root := range allowedRoots {
        if strings.HasPrefix(absPath, root) {
            return nil
        }
    }

    return fmt.Errorf("file path outside allowed directories: %s", path)
}
```

---

## Performance Optimization

### Indexing Strategy (Existing)

**Already Optimized**:
- `tasks.key` has UNIQUE index (automatic)
- `tasks.feature_id` has index (for foreign key lookups)
- `task_history.task_id` has index (for history queries)

**No New Indexes Needed**: Sync queries use existing indexes efficiently.

### Bulk Operation Performance

**Prepared Statements for Bulk Insert**:
```go
// Prepare once
stmt, err := tx.PrepareContext(ctx, `
    INSERT INTO tasks (feature_id, key, title, description, status, file_path)
    VALUES (?, ?, ?, ?, ?, ?)
`)
defer stmt.Close()

// Execute multiple times
for _, task := range tasks {
    _, err = stmt.ExecContext(ctx, task.FeatureID, task.Key, ...)
}
```

**Transaction Batching**:
- Use single transaction for entire sync
- Reduces commit overhead (WAL mode benefits)
- Target: 100 inserts in <1 second

---

## Data Migration Considerations

### No Migration Required

**Reason**: This feature uses existing schema without changes.

**Compatibility**:
- Works with existing database created by E04-F01
- No ALTER TABLE statements needed
- No data migration scripts needed

### Future Migration Path (if needed)

**If we later add sync metadata table**:
```sql
-- Migration script (future, if needed)
CREATE TABLE sync_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_key TEXT NOT NULL UNIQUE,
    last_synced_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    content_hash TEXT,
    FOREIGN KEY (task_key) REFERENCES tasks(key) ON DELETE CASCADE
);

CREATE INDEX idx_sync_metadata_task_key ON sync_metadata(task_key);
```

**For MVP**: Not needed. Use existing fields.

---

## Backup and Recovery

### Pre-Sync Backup (--backup flag)

**Backup Strategy**:
```go
// Copy database file before sync
func CreateBackup(dbPath string) (string, error) {
    timestamp := time.Now().Format("2006-01-02T15-04-05")
    backupPath := fmt.Sprintf("%s.backup.%s", dbPath, timestamp)

    // Copy file
    return backupPath, copyFile(dbPath, backupPath)
}
```

**Backup Location**: Same directory as database
**Backup Naming**: `shark-tasks.db.backup.2025-12-16T10-30-00`

### Restore from Backup

**Manual Restore**:
```bash
# If sync goes wrong, restore from backup
cp shark-tasks.db.backup.2025-12-16T10-30-00 shark-tasks.db
```

**Automatic Rollback**: Transaction rollback handles most error cases (no backup restore needed).

---

## Data Consistency Rules

### Consistency Guarantees

**1. Task Uniqueness**:
- Task key is UNIQUE in database
- One task per key (enforced by database)

**2. Epic/Feature Relationships**:
- Every task must have valid feature_id
- Every feature must have valid epic_id
- Enforced by foreign key constraints

**3. File Path Tracking**:
- `tasks.file_path` always reflects actual file location after sync
- Updated during every sync operation

**4. Status Consistency**:
- Status is NEVER read from file frontmatter
- Status is ALWAYS preserved during sync (database only)

**5. Timestamp Consistency**:
- `updated_at` automatically updated by trigger on any UPDATE
- File modified time read from filesystem (os.Stat)
- Timestamps used for newer-wins strategy

---

## Error Recovery Strategies

### Database Errors

**Foreign Key Violation** (missing epic/feature):
```
Error: FOREIGN KEY constraint failed
Action: Rollback transaction, log error
Recovery: User must create epic/feature first or use --create-missing
```

**Unique Key Violation** (duplicate task key):
```
Error: UNIQUE constraint failed: tasks.key
Action: Rollback transaction, log error
Recovery: This indicates data corruption (key in file already in DB but query missed it)
```

### Transaction Rollback

**Automatic Rollback Triggers**:
- Any SQL error (constraint violation, syntax error)
- Context cancellation (timeout, user interrupt)
- File read error mid-sync

**Post-Rollback State**:
- Database unchanged (transaction never committed)
- No partial updates
- File paths not updated
- History records not created

---

## Testing Data Scenarios

### Test Data Sets

**1. Clean Import** (new tasks, no conflicts):
```
Files: 10 new task files
Database: Empty
Expected: 10 tasks imported, no conflicts
```

**2. Conflict Resolution** (file vs database):
```
Files: 5 files with title "New Title"
Database: Same 5 tasks with title "Old Title"
Expected: 5 conflicts detected, resolved per strategy
```

**3. File Moved** (file path conflict):
```
Files: Task at docs/plan/.../T-E04-F07-001.md
Database: Task with file_path docs/tasks/created/T-E04-F07-001.md
Expected: File path updated to new location
```

**4. Missing Epic/Feature**:
```
Files: Task T-E99-F99-001 (E99-F99 doesn't exist)
Database: No epic E99 or feature E99-F99
Expected (without --create-missing): Task skipped, warning logged
Expected (with --create-missing): Epic E99 and feature E99-F99 created, task imported
```

**5. Orphaned Tasks** (--cleanup flag):
```
Files: 10 task files scanned
Database: 15 tasks (5 files deleted)
Expected: 5 orphaned tasks deleted
```

---

## Summary

### Data Storage Approach

- **No new tables**: Use existing schema
- **Use existing fields**: `file_path`, `updated_at`
- **Track sync in history**: Use `task_history` table with agent='sync'
- **Conflict detection**: In-memory comparison during sync
- **Resolution**: Update database based on strategy

### Critical Data Fields

| Field | Table | Purpose | Updated During Sync? |
|-------|-------|---------|---------------------|
| `key` | tasks | Task identifier | No (unique, immutable) |
| `title` | tasks | Task title | Yes (if conflict, per strategy) |
| `description` | tasks | Task description | Yes (if conflict, per strategy) |
| `file_path` | tasks | File location | Yes (always updated to actual) |
| `updated_at` | tasks | Last DB update | Yes (automatic trigger) |
| `status` | tasks | Task status | No (database only, preserved) |
| `priority` | tasks | Task priority | No (database only, preserved) |
| `agent_type` | tasks | Agent type | No (database only, preserved) |

---

**Document Complete**: 2025-12-16
**Next Document**: 02-architecture.md (backend-architect creates)
