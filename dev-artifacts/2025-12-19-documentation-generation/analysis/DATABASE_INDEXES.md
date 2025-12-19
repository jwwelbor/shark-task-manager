# Shark Task Manager - Database Indexes

## Overview
This document details all database indexes in the Shark Task Manager schema. Indexes are critical for query performance, enabling the CLI and API to efficiently retrieve and filter tasks, features, and epics.

**Total Indexes**: 13 (10 application indexes + 3 implicit primary key indexes)

---

## Index Inventory

### EPICS Table Indexes

#### 1. idx_epics_key (UNIQUE)
**Location**: `internal/db/db.go` line 84

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_epics_key ON epics(key);
```

**Type**: UNIQUE, Single-column
**Column**: `epics.key` (TEXT)
**Cardinality**: High (unique)

**Purpose**:
- Enforce uniqueness of epic identifiers (E04, E07, etc.)
- Enable fast lookup by key: `SELECT * FROM epics WHERE key = 'E04'`

**Query Patterns Optimized**:
```sql
-- O(log n) lookup by key
SELECT * FROM epics WHERE key = 'E04';

-- Quick check for key existence
SELECT id FROM epics WHERE key = ?;

-- Prevent duplicate key insertion
INSERT INTO epics (key, ...) VALUES (?, ...);  -- Automatically rejected if duplicate
```

**Performance Impact**: Critical. Epic key lookup is primary access pattern.

**Index Statistics** (Expected):
- Lookup time: ~2-3 disk I/Os
- Inserts/updates: Minimal overhead (unique constraint must be checked)
- Space: ~8-16 bytes per epic

---

#### 2. idx_epics_status (Non-unique)
**Location**: `internal/db/db.go` line 85

```sql
CREATE INDEX IF NOT EXISTS idx_epics_status ON epics(status);
```

**Type**: Non-unique, Single-column
**Column**: `epics.status` (TEXT enum)
**Cardinality**: Low (4 possible values: draft|active|completed|archived)

**Purpose**:
- Filter epics by status
- Enable reports showing active vs. archived epics

**Query Patterns Optimized**:
```sql
-- Find all active epics
SELECT * FROM epics WHERE status = 'active';

-- List draft epics for planning
SELECT * FROM epics WHERE status = 'draft';

-- Count completed epics
SELECT COUNT(*) FROM epics WHERE status = 'completed';

-- Bulk status transitions
UPDATE epics SET status = 'archived' WHERE status = 'completed';
```

**Performance Impact**: Moderate. Status filtering is secondary but valuable for dashboards.

**Index Statistics** (Expected):
- Cardinality: 4 values means ~25% of epics per status bucket
- Lookup time: ~5-10 disk I/Os (branch nodes)
- Space: ~8 bytes per epic

**Selectivity Calculation**:
- If 100 total epics, ~25 per status
- Index reduces full table scan from 100 rows to ~25 rows
- Improvement factor: 4x

---

### FEATURES Table Indexes

#### 3. idx_features_key (UNIQUE)
**Location**: `internal/db/db.go` line 114

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_features_key ON features(key);
```

**Type**: UNIQUE, Single-column
**Column**: `features.key` (TEXT)
**Cardinality**: High (unique)

**Purpose**:
- Enforce uniqueness of feature identifiers (E04-F01, E04-F02, etc.)
- Enable fast lookup by key

**Query Patterns Optimized**:
```sql
-- Primary lookup pattern
SELECT * FROM features WHERE key = 'E04-F01';

-- Fast feature validation
SELECT id FROM features WHERE key = ?;

-- Prevent duplicate keys
INSERT INTO features (key, ...) VALUES (?, ...);
```

**Performance Impact**: Critical. Feature key lookup is primary access pattern.

**Index Statistics** (Expected):
- Lookup time: ~2-3 disk I/Os
- Uniqueness guarantee: Automatic
- Space: ~8-16 bytes per feature

---

#### 4. idx_features_epic_id (Non-unique)
**Location**: `internal/db/db.go` line 115

```sql
CREATE INDEX IF NOT EXISTS idx_features_epic_id ON features(epic_id);
```

**Type**: Non-unique, Single-column (Foreign Key)
**Column**: `features.epic_id` (INTEGER)
**Cardinality**: Medium (depends on feature distribution across epics)

**Purpose**:
- Foreign key index for referential integrity
- Enable efficient child lookup: find all features for a given epic

**Query Patterns Optimized**:
```sql
-- Find all features in epic
SELECT * FROM features WHERE epic_id = 5;

-- Get feature count per epic
SELECT epic_id, COUNT(*) FROM features GROUP BY epic_id;

-- Epic progress calculation (sum of feature progresses)
SELECT AVG(progress_pct) FROM features WHERE epic_id = 5;

-- Cascade delete preparation
DELETE FROM features WHERE epic_id = 5;
```

**Performance Impact**: Essential. Critical for feature queries by epic.

**Index Statistics** (Expected):
- Typical distribution: 5-10 features per epic
- Lookup time: ~3-4 disk I/Os
- Space: ~8-12 bytes per feature

**Foreign Key Benefit**:
- SQLite requires FK index for efficient constraint checking
- Without this index, insert/update validation becomes O(n) scan
- With index, FK validation is O(log n)

---

#### 5. idx_features_status (Non-unique)
**Location**: `internal/db/db.go` line 116

```sql
CREATE INDEX IF NOT EXISTS idx_features_status ON features(status);
```

**Type**: Non-unique, Single-column
**Column**: `features.status` (TEXT enum)
**Cardinality**: Low (4 possible values: draft|active|completed|archived)

**Purpose**:
- Filter features by status
- Support feature status reports and filtering

**Query Patterns Optimized**:
```sql
-- Find all active features
SELECT * FROM features WHERE status = 'active';

-- List completed features
SELECT * FROM features WHERE status = 'completed';

-- Count features by status
SELECT status, COUNT(*) FROM features GROUP BY status;

-- Update features by status
UPDATE features SET status = 'archived' WHERE status = 'completed';
```

**Performance Impact**: Moderate. Status filtering is common in dashboards.

**Index Statistics** (Expected):
- Cardinality: 4 values
- Selectivity: ~25% of features per status
- Lookup time: ~5-10 disk I/Os
- Space: ~8 bytes per feature

---

### TASKS Table Indexes

#### 6. idx_tasks_key (UNIQUE)
**Location**: `internal/db/db.go` line 153

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_key ON tasks(key);
```

**Type**: UNIQUE, Single-column
**Column**: `tasks.key` (TEXT)
**Cardinality**: Very high (unique, formatted as T-E##-F##-###)

**Purpose**:
- Enforce uniqueness of task identifiers (T-E04-F06-001, etc.)
- Enable primary lookup by key

**Query Patterns Optimized**:
```sql
-- Get specific task
SELECT * FROM tasks WHERE key = 'T-E04-F06-001';

-- Check task existence
SELECT id FROM tasks WHERE key = ?;

-- Validate dependencies (parse depends_on field)
SELECT * FROM tasks WHERE key IN ('T-E04-F06-001', 'T-E04-F06-002');

-- Prevent duplicate task keys
INSERT INTO tasks (key, ...) VALUES (?, ...);
```

**Performance Impact**: Critical. Single most important index - task lookups by key are primary operation.

**Index Statistics** (Expected):
- Lookup time: ~2-3 disk I/Os (best case)
- Uniqueness: Enforced by database
- Space: ~12-20 bytes per task

**Expected Scale**:
- Typical project: 200-500 tasks
- Large project: 1000+ tasks
- Index handles scale well with logarithmic complexity

---

#### 7. idx_tasks_feature_id (Non-unique)
**Location**: `internal/db/db.go` line 154

```sql
CREATE INDEX IF NOT EXISTS idx_tasks_feature_id ON tasks(feature_id);
```

**Type**: Non-unique, Single-column (Foreign Key)
**Column**: `tasks.feature_id` (INTEGER)
**Cardinality**: Medium (depends on task distribution)

**Purpose**:
- Foreign key index for referential integrity
- Enable efficient child lookup: find all tasks for a feature

**Query Patterns Optimized**:
```sql
-- Get all tasks in feature
SELECT * FROM tasks WHERE feature_id = 23;

-- Calculate feature progress (count complete tasks)
SELECT COUNT(*) FROM tasks WHERE feature_id = 23 AND status = 'completed';

-- Feature task statistics
SELECT status, COUNT(*) FROM tasks
WHERE feature_id = 23
GROUP BY status;

-- Cascade delete tasks for feature
DELETE FROM tasks WHERE feature_id = 23;
```

**Performance Impact**: Essential. Required for feature→task queries and progress calculation.

**Index Statistics** (Expected):
- Typical distribution: 10-30 tasks per feature
- Lookup time: ~3-4 disk I/Os
- Space: ~8-12 bytes per task

**Foreign Key Enforcement**:
- Ensures efficient constraint checking on insert/update
- Prevents orphaned tasks (no feature_id without corresponding feature)

---

#### 8. idx_tasks_status (Non-unique)
**Location**: `internal/db/db.go` line 155

```sql
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
```

**Type**: Non-unique, Single-column
**Column**: `tasks.status` (TEXT enum)
**Cardinality**: Medium (6 possible values: todo|in_progress|blocked|ready_for_review|completed|archived)

**Purpose**:
- Filter tasks by status for task queries and CLI commands
- Support `task next`, `task list` commands with status filtering

**Query Patterns Optimized**:
```sql
-- Find all todo tasks
SELECT * FROM tasks WHERE status = 'todo' ORDER BY priority DESC;

-- Get tasks by status for agent
SELECT * FROM tasks WHERE status = 'in_progress' AND agent_type = 'developer';

-- Count tasks by status
SELECT status, COUNT(*) FROM tasks GROUP BY status;

-- Bulk status transitions
UPDATE tasks SET status = 'archived' WHERE status = 'completed';

-- Task statistics
SELECT
    status,
    COUNT(*) as count
FROM tasks
GROUP BY status;
```

**Performance Impact**: High. Status filtering is very common in task management commands.

**Index Statistics** (Expected):
- Cardinality: 6 values (~16-17% per status)
- Typical distribution:
  - todo: 30%
  - in_progress: 20%
  - blocked: 5%
  - ready_for_review: 10%
  - completed: 35%
  - archived: 0-5%
- Lookup time: ~4-8 disk I/Os
- Space: ~8 bytes per task

---

#### 9. idx_tasks_agent_type (Non-unique)
**Location**: `internal/db/db.go` line 156

```sql
CREATE INDEX IF NOT EXISTS idx_tasks_agent_type ON tasks(agent_type);
```

**Type**: Non-unique, Single-column
**Column**: `tasks.agent_type` (TEXT)
**Cardinality**: Low (5-10 agent types: developer, architect, designer, etc.)

**Purpose**:
- Filter tasks by agent type
- Support agent-specific task queries

**Query Patterns Optimized**:
```sql
-- Get tasks for specific agent type
SELECT * FROM tasks WHERE agent_type = 'developer' AND status = 'todo';

-- Get next developer task
SELECT * FROM tasks
WHERE agent_type = 'developer' AND status IN ('todo', 'in_progress')
ORDER BY priority DESC
LIMIT 1;

-- Find all architect tasks
SELECT * FROM tasks WHERE agent_type = 'architect';

-- Count tasks by agent type
SELECT agent_type, COUNT(*) FROM tasks GROUP BY agent_type;
```

**Performance Impact**: Medium. Useful for agent-specific queries but not as critical as status.

**Index Statistics** (Expected):
- Cardinality: 5-10 agent types
- Typical distribution: 20-50 tasks per agent type
- Selectivity: ~15-20%
- Lookup time: ~5-8 disk I/Os
- Space: ~8-12 bytes per task

---

#### 10. idx_tasks_status_priority (COMPOSITE - Non-unique)
**Location**: `internal/db/db.go` line 157

```sql
CREATE INDEX IF NOT EXISTS idx_tasks_status_priority ON tasks(status, priority);
```

**Type**: Non-unique, COMPOSITE (2 columns)
**Columns**: `tasks.status` (TEXT), `tasks.priority` (INTEGER)
**Cardinality**: High (6 status values × 10 priority levels = 60 combinations)

**Purpose**:
- Optimize "get next task" query which filters by status and sorts by priority
- Critical for task assignment workflow

**Query Patterns Optimized**:
```sql
-- CRITICAL: Get next task for agent - MOST IMPORTANT QUERY
SELECT * FROM tasks
WHERE status = 'todo' AND agent_type = 'developer'
ORDER BY priority DESC, created_at ASC
LIMIT 1;
-- Uses: idx_tasks_status_priority (finds candidate rows)
--       then idx_tasks_agent_type (optional, but status+priority is primary filter)

-- Ordered task list by priority
SELECT * FROM tasks
WHERE status IN ('todo', 'in_progress')
ORDER BY priority DESC;

-- Priority-aware filtering
SELECT * FROM tasks
WHERE status = 'ready_for_review'
ORDER BY priority DESC;

-- High-priority todo tasks
SELECT * FROM tasks
WHERE status = 'todo' AND priority >= 8
ORDER BY created_at ASC;
```

**Performance Impact**: VERY HIGH. This is the most critical optimization for task management.

**Composite Index Behavior**:
- Covers both filter columns in single index
- No need for separate idx_tasks_status lookup
- Enabled ordering by priority without additional sort
- Reduces query time from O(n log n) to O(log n) for candidate selection

**Index Statistics** (Expected):
- Leaf nodes: 60 possible (status, priority) combinations
- Distribution: Not uniform; depends on task assignment strategy
- Typical range: 5-20 tasks per (status, priority) combo
- Lookup time: ~3-5 disk I/Os (better than single index)
- Space: ~16-24 bytes per task (2 column overhead)

**Query Execution Impact**:
```
WITHOUT index (idx_tasks_status_priority):
  Full table scan (500 tasks) → Filter by status (100 tasks)
  → Sort by priority (25 comparisons) → LIMIT 1
  Time: O(n log n) = 500 * log(500) ≈ 4000 comparisons

WITH index (idx_tasks_status_priority):
  Index range scan → Get first row
  Time: O(log n) = log(500) ≈ 9 disk I/Os
  Speed improvement: ~400x faster
```

**Alternative Consideration**:
- Single index on `status` could handle most cases
- Composite index adds minimal space overhead
- Saves one sort operation in application logic

---

#### 11. idx_tasks_priority (Non-unique)
**Location**: `internal/db/db.go` line 158

```sql
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
```

**Type**: Non-unique, Single-column
**Column**: `tasks.priority` (INTEGER, 1-10)
**Cardinality**: Low (10 possible values: 1 through 10)

**Purpose**:
- Filter tasks by priority level
- Support priority-based queries

**Query Patterns Optimized**:
```sql
-- Get high-priority tasks
SELECT * FROM tasks WHERE priority >= 8 AND status = 'todo';

-- Find urgent tasks
SELECT * FROM tasks WHERE priority = 10 ORDER BY created_at ASC;

-- Priority distribution
SELECT priority, COUNT(*) FROM tasks GROUP BY priority;

-- Update priority range
UPDATE tasks SET priority = 7 WHERE priority <= 3;
```

**Performance Impact**: Low-Medium. Less useful than status_priority composite.

**Index Statistics** (Expected):
- Cardinality: 10 values (1-10)
- Selectivity: ~10% per priority level
- Lookup time: ~6-10 disk I/Os
- Space: ~8-12 bytes per task

**Interaction with Composite Index**:
- `idx_tasks_status_priority` covers priority filtering for status queries
- This single-column index provides alternative optimization for priority-only queries
- Slight redundancy but acceptable for flexibility

---

#### 12. idx_tasks_file_path (Non-unique)
**Location**: `internal/db/db.go` line 159

```sql
CREATE INDEX IF NOT EXISTS idx_tasks_file_path ON tasks(file_path);
```

**Type**: Non-unique, Single-column
**Column**: `tasks.file_path` (TEXT)
**Cardinality**: High (unique or near-unique)

**Purpose**:
- Map markdown files to tasks during file-system sync operations
- Enable fast lookup by file path

**Query Patterns Optimized**:
```sql
-- Find task by file path (sync operation)
SELECT * FROM tasks WHERE file_path = 'docs/plan/E04/F06/T-E04-F06-001.md';

-- Check if file is already mapped
SELECT id FROM tasks WHERE file_path = ?;

-- Find unmapped tasks
SELECT * FROM tasks WHERE file_path IS NULL;

-- Update file path for task
UPDATE tasks SET file_path = ? WHERE key = ?;

-- Bulk update file paths during discovery
UPDATE tasks SET file_path = ?
WHERE key IN (SELECT key FROM tasks WHERE file_path IS NULL);
```

**Performance Impact**: Medium. Critical for sync operations but not daily CLI use.

**Index Statistics** (Expected):
- Cardinality: Near-unique (most tasks have unique file path)
- Lookup time: ~2-4 disk I/Os
- Space: ~16-32 bytes per task (longer strings)

**NULL Handling**:
- SQLite includes NULL values in indexes
- NULL lookups: `WHERE file_path IS NULL` uses full table scan (not index)
- Alternative: Create filtered index for NULL values if needed

**Use Case**:
- Primarily used in `internal/sync/` package
- When syncing files: `SELECT * FROM tasks WHERE file_path = <new_path>`
- Detects duplicate file assignments across tasks

---

### TASK_HISTORY Table Indexes

#### 13. idx_task_history_task_id (Non-unique)
**Location**: `internal/db/db.go` line 186

```sql
CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id);
```

**Type**: Non-unique, Single-column (Foreign Key)
**Column**: `task_history.task_id` (INTEGER)
**Cardinality**: Medium (depends on history depth per task)

**Purpose**:
- Foreign key index for referential integrity
- Enable efficient history lookup by task

**Query Patterns Optimized**:
```sql
-- Get all history for task
SELECT * FROM task_history WHERE task_id = 42 ORDER BY timestamp DESC;

-- Get recent changes for task
SELECT * FROM task_history
WHERE task_id = 42 AND timestamp > datetime('-7 days')
ORDER BY timestamp DESC;

-- Count status changes
SELECT COUNT(*) FROM task_history WHERE task_id = 42;

-- Audit trail for task
SELECT new_status, agent, notes, timestamp
FROM task_history
WHERE task_id = 42
ORDER BY timestamp DESC;
```

**Performance Impact**: Medium-High. Essential for history queries.

**Index Statistics** (Expected):
- Cardinality: Depends on task age
- Typical: 3-10 history records per task
- Lookup time: ~3-4 disk I/Os
- Space: ~8-12 bytes per history record

**Foreign Key Enforcement**:
- Ensures referential integrity on insert
- Prevents history records for non-existent tasks

---

#### 14. idx_task_history_timestamp (Non-unique, DESC)
**Location**: `internal/db/db.go` line 187

```sql
CREATE INDEX IF NOT EXISTS idx_task_history_timestamp ON task_history(timestamp DESC);
```

**Type**: Non-unique, Single-column with DESC ordering
**Column**: `task_history.timestamp` (TIMESTAMP)
**Cardinality**: Very high (unique timestamps)

**Purpose**:
- Enable efficient chronological queries on history
- Support timeline/audit reporting

**Query Patterns Optimized**:
```sql
-- Recent changes across all tasks
SELECT * FROM task_history
ORDER BY timestamp DESC
LIMIT 10;

-- Changes in date range
SELECT * FROM task_history
WHERE timestamp BETWEEN datetime('-30 days') AND datetime('now')
ORDER BY timestamp DESC;

-- Today's changes
SELECT * FROM task_history
WHERE DATE(timestamp) = DATE('now')
ORDER BY timestamp DESC;

-- Status changes by date
SELECT DATE(timestamp) as date, COUNT(*)
FROM task_history
GROUP BY DATE(timestamp)
ORDER BY date DESC;
```

**Performance Impact**: Medium. Useful for auditing and reporting.

**DESC Ordering Benefit**:
- Most recent history accessed first (common query pattern)
- DESC index enables reverse iteration without sort
- SQLite optimizer can use DESC index for ORDER BY timestamp DESC
- Without DESC hint, index can still be used but may require reverse scan

**Index Statistics** (Expected):
- Cardinality: Very high (timestamps are unique or near-unique)
- Typical distribution: ~1-5 records per hour during active work
- Lookup time: ~4-6 disk I/Os for range queries
- Space: ~8-16 bytes per history record

**Combined Query Pattern**:
```sql
-- Combines both history indexes
SELECT * FROM task_history
WHERE task_id = 42 AND timestamp > datetime('-7 days')
ORDER BY timestamp DESC
LIMIT 5;
-- Uses: idx_task_history_task_id (initial filter)
--       Sorted in memory by timestamp (only 5-7 rows)
```

---

## Index Usage Summary Table

| Index Name | Table | Columns | Type | Key Access Pattern | Estimated Lookup Time |
|------------|-------|---------|------|-------------------|----------------------|
| idx_epics_key | epics | key | UNIQUE | Epic by key | O(log n) ~2-3 ms |
| idx_epics_status | epics | status | Non-unique | Epics by status | O(log n) ~5-10 ms |
| idx_features_key | features | key | UNIQUE | Feature by key | O(log n) ~2-3 ms |
| idx_features_epic_id | features | epic_id | Non-unique | Features by epic | O(log n) ~3-4 ms |
| idx_features_status | features | status | Non-unique | Features by status | O(log n) ~5-10 ms |
| idx_tasks_key | tasks | key | UNIQUE | Task by key | O(log n) ~2-3 ms |
| idx_tasks_feature_id | tasks | feature_id | Non-unique | Tasks by feature | O(log n) ~3-4 ms |
| idx_tasks_status | tasks | status | Non-unique | Tasks by status | O(log n) ~4-8 ms |
| idx_tasks_agent_type | tasks | agent_type | Non-unique | Tasks by agent | O(log n) ~5-8 ms |
| idx_tasks_status_priority | tasks | status, priority | Composite | Next task query | O(log n) ~3-5 ms |
| idx_tasks_priority | tasks | priority | Non-unique | Tasks by priority | O(log n) ~6-10 ms |
| idx_tasks_file_path | tasks | file_path | Non-unique | Task by file path | O(log n) ~2-4 ms |
| idx_task_history_task_id | task_history | task_id | Non-unique | History by task | O(log n) ~3-4 ms |
| idx_task_history_timestamp | task_history | timestamp DESC | Non-unique | History timeline | O(log n) ~4-6 ms |

---

## Performance Optimization Strategies

### High-Impact Index Usage
1. **idx_tasks_key**: Used for nearly every task operation
2. **idx_tasks_status_priority**: Critical for "task next" operations
3. **idx_features_epic_id**: Essential for progress calculation
4. **idx_tasks_feature_id**: Used for feature queries

### Medium-Impact Usage
5. **idx_tasks_status**: Status filtering without agent type
6. **idx_task_history_task_id**: Audit trail lookup
7. **idx_features_key**: Feature lookups

### Low-Impact but Useful
8. **idx_tasks_agent_type**: Agent-specific queries
9. **idx_task_history_timestamp**: Timeline queries
10. **idx_epics_status**: Epic status filtering

### Rarely Used
11. **idx_tasks_priority**: Priority-only queries (handled by composite)
12. **idx_tasks_file_path**: Sync operations
13. **idx_epics_key**: Epic key lookup

---

## Index Maintenance

### Index Statistics
SQLite maintains index statistics automatically:
- B-tree structure balanced on insertion
- No manual ANALYZE needed for correctness
- `ANALYZE` command updates query planner statistics if needed

### Index Fragmentation
- SQLite indexes don't fragment like traditional systems
- Long-lived indexes remain efficient
- REINDEX only needed after corruption (rare)

### Index Space Usage
**Estimated Total Index Space**:
```
Average database with 200-500 tasks:
- Task index space: ~50-100 KB (composite + single column indexes)
- History index space: ~20-40 KB
- Feature/Epic indexes: ~5-10 KB
- Total: ~75-150 KB for indexes (~1-2% of typical database size)
```

### Performance Monitoring
Without built-in profiling, observe:
- `EXPLAIN QUERY PLAN` shows if indexes are used
- `PRAGMA optimize;` auto-updates statistics
- Application-level timing helps identify slow queries

---

## Index Best Practices Applied

✓ **Foreign key indexes**: All FK columns indexed for constraint checking
✓ **Unique constraints**: Implemented as UNIQUE indexes for fast lookups
✓ **Composite indexes**: Used for correlated filters (status + priority)
✓ **DESC ordering**: Applied to timestamp index for reverse chronology
✓ **No over-indexing**: Only essential indexes created (selective approach)
✓ **Query-driven design**: Indexes match actual query patterns
✓ **NULL handling**: Considered in file_path index design

---

## Query Optimization Examples

### Example 1: Get Next Task
```sql
-- Query
SELECT * FROM tasks
WHERE status = 'todo'
  AND agent_type = 'developer'
ORDER BY priority DESC, created_at ASC
LIMIT 1;

-- Index used: idx_tasks_status_priority
-- Explanation:
--   1. Range scan on idx_tasks_status_priority
--      finds all (status='todo', any priority)
--   2. Filter by agent_type in memory
--   3. Order by priority DESC (already ordered by index)
--   4. LIMIT 1 returns first row
-- Time complexity: O(log n) + O(k) where k = rows with status='todo'
```

### Example 2: Calculate Feature Progress
```sql
-- Query
SELECT
    COUNT(*) as total_tasks,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_tasks
FROM tasks
WHERE feature_id = 23;

-- Index used: idx_tasks_feature_id
-- Explanation:
--   1. Index range scan finds all tasks with feature_id=23
--   2. Stream through matching rows (no sort needed)
--   3. Aggregate in memory
-- Time complexity: O(log n) + O(k) where k = tasks in feature
-- Typically: 10-30 rows scanned (very fast)
```

### Example 3: Get All Features in Epic
```sql
-- Query
SELECT * FROM features
WHERE epic_id = 5
ORDER BY execution_order;

-- Index used: idx_features_epic_id
-- Explanation:
--   1. Index range scan finds all features in epic
--   2. Sort by execution_order in memory
--   3. Return to caller
-- Time complexity: O(log n) + O(k log k) where k = features in epic
-- Typically: 5-10 features (order step negligible)
```

