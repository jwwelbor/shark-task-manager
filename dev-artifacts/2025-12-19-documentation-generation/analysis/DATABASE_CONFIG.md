# Shark Task Manager - SQLite Configuration (PRAGMAs)

## Overview
This document details the SQLite configuration pragmas used in the Shark Task Manager. These settings optimize for data integrity, performance, and concurrency while maintaining compatibility with SQLite's design principles.

**Configuration Type**: Production-ready optimizations applied at database initialization

---

## PRAGMA Configuration Summary

| PRAGMA | Value | Purpose | Impact |
|--------|-------|---------|--------|
| `foreign_keys` | ON | Enable foreign key constraints | Data integrity, referential consistency |
| `journal_mode` | WAL | Write-Ahead Logging for concurrency | Better concurrency, slight I/O overhead |
| `busy_timeout` | 5000 | Wait 5 seconds for locked database | Graceful retry vs. immediate failure |
| `synchronous` | NORMAL | Balance safety and performance | Faster writes, minimal crash risk |
| `cache_size` | -64000 | 64 MB in-memory cache | Faster queries, reduced disk I/O |
| `temp_store` | MEMORY | Store temp tables in memory | Faster temp operations, reduced disk I/O |
| `mmap_size` | 30000000000 | 30 GB memory-mapped I/O | Faster random access, larger virtual address space |

---

## PRAGMA Configurations Explained

### 1. foreign_keys = ON

**Location**: `internal/db/db.go` line 38

```sql
PRAGMA foreign_keys = ON;
```

**Purpose**:
- Enable SQLite's foreign key constraint checking
- Enforce referential integrity at the database level
- Prevent orphaned records and data inconsistencies

**Default State**:
- OFF in SQLite (for backward compatibility with older schemas)
- Application explicitly enables it at initialization

**Data Integrity Impact**:

**Without foreign_keys (OFF)**:
```sql
-- This would be allowed, creating orphaned feature
INSERT INTO features (epic_id, key, title, status) VALUES (999, 'E04-F01', 'Title', 'draft');
-- Results in feature with non-existent epic_id=999
```

**With foreign_keys (ON)** - Actual behavior:
```sql
-- This is REJECTED
INSERT INTO features (epic_id, key, title, status) VALUES (999, 'E04-F01', 'Title', 'draft');
-- Error: FOREIGN KEY constraint failed
-- epic_id=999 doesn't exist in epics table
```

**Foreign Key Relationships Protected**:
```
features.epic_id → epics.id
tasks.feature_id → features.id
task_history.task_id → tasks.id
```

**Verification Command**:
```sql
PRAGMA foreign_keys;
-- Returns: 1 (ON) or 0 (OFF)
-- Must return 1 for integrity checks to work
```

**Code Implementation**:
```go
// internal/db/db.go line 54-60
var fkEnabled int
if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fkEnabled); err != nil {
    return fmt.Errorf("failed to verify foreign_keys: %w", err)
}
if fkEnabled != 1 {
    return fmt.Errorf("foreign_keys not enabled")
}
```

**Constraint Checking Overhead**:
- ON: ~5-10% slowdown for INSERT/UPDATE/DELETE (acceptable trade-off)
- Scales logarithmically with table size (indexed FK columns)
- Worth the cost for data integrity guarantee

**Cascading Delete Behavior**:
```sql
DELETE FROM epics WHERE id = 5;

-- With foreign_keys ON:
-- 1. Find all features with epic_id = 5
-- 2. Find all tasks in those features
-- 3. Find all history for those tasks
-- 4. Delete in reverse dependency order
-- 5. All-or-nothing transaction

-- Result:
-- - Epic deleted
-- - All child features deleted (ON DELETE CASCADE)
-- - All child tasks deleted (ON DELETE CASCADE)
-- - All history records deleted (ON DELETE CASCADE)
```

**Importance**:
```
Critical for schema integrity. This is NOT optional.
The initialization code fails if foreign_keys != ON.
```

---

### 2. journal_mode = WAL

**Location**: `internal/db/db.go` line 39

```sql
PRAGMA journal_mode = WAL;
```

**Purpose**:
- Enable Write-Ahead Logging (WAL) mode
- Improve concurrency: readers don't block writers
- Provide better crash recovery

**Default Mode**: DELETE journal mode (not WAL)

**How WAL Works**:

**Traditional (DELETE) Mode**:
```
Write transaction:
  1. Lock database
  2. Write changes to main file
  3. Sync to disk
  4. Unlock database
  ↓
Read blocked during steps 1-4
```

**WAL Mode**:
```
Write transaction:
  1. Write changes to WAL file (append-only)
  2. Sync WAL file to disk
  ↓
Read can proceed (reads committed data from main file)

Later (checkpoint):
  1. Merge WAL into main file
  2. Readers pick up new data
```

**Concurrency Benefit**:
```
Scenario: Multiple agents running tasks concurrently

DELETE mode:
  Agent A updates task 1 → Database locked
  Agent B tries to read task 2 → BLOCKED (waits up to 5 seconds)
  Result: Contention, potential timeouts

WAL mode:
  Agent A writes task 1 update to WAL → Main file accessible
  Agent B reads task 2 from main file → NOT BLOCKED
  Result: Smooth concurrency, no contention
```

**Disk Files Created**:
```
shark-tasks.db              (main database)
shark-tasks.db-wal          (write-ahead log)
shark-tasks.db-shm          (shared memory)
```

**File Sizes**:
- Main file: Grows as tasks/features/epics added
- WAL file: ~0 KB (after checkpoint), up to several MB during heavy writes
- Shared memory: ~32 KB (small, shared across connections)

**Checkpoint Behavior**:
```sql
-- WAL checkpoints automatically when:
-- 1. Database is closed
-- 2. WAL file reaches ~1 GB
-- 3. Explicit PRAGMA checkpoint_command

-- Manual checkpoint (rarely needed):
PRAGMA wal_autocheckpoint = 1000;  -- Checkpoint every 1000 pages
PRAGMA wal_checkpoint = TRUNCATE;   -- Truncate WAL after checkpoint
```

**Trade-offs**:

**Pros**:
- Better concurrency (writers don't block readers)
- Faster writes (append to WAL is faster)
- Better crash recovery
- Real-world benefit for multi-user systems

**Cons**:
- Uses slightly more disk I/O during checkpoint
- Requires two files (main + WAL) - may complicate backups
- Slightly slower single-writer scenario
- Not supported on network filesystems

**For Shark Task Manager**:
- CLI is single-user (one agent at a time)
- API is multi-user (multiple agents)
- WAL is beneficial for API server with concurrent connections
- Single CLI invocations don't see concurrency benefit

---

### 3. busy_timeout = 5000

**Location**: `internal/db/db.go` line 40

```sql
PRAGMA busy_timeout = 5000;
```

**Purpose**:
- Set timeout for handling database locks
- Retry for 5 seconds instead of immediate failure
- Graceful handling of contention

**Default**: 0 ms (immediate failure)

**Behavior**:

**Without timeout (0 ms)**:
```
Agent A: UPDATE task 1 → Database locked
Agent B: SELECT task 2 → SQLITE_BUSY error immediately
Result: Agent B fails, must retry manually
```

**With 5000 ms timeout**:
```
Agent A: UPDATE task 1 → Database locked
Agent B: SELECT task 2 → SQLITE_BUSY error
        → Retry after 1 ms
        → Retry after 1 ms
        → ... (repeat for up to 5 seconds)
        → If lock released: Succeed
        → If lock held 5 seconds: SQLITE_BUSY error
Result: Agent B waits for lock, succeeds if released in 5 seconds
```

**Lock Scenarios**:

**Scenario 1: Brief Lock (< 5 seconds)**
```
Time 0 ms: Agent A: Begin transaction
Time 100 ms: Agent B: Try to read → BUSY
Time 150 ms: Agent A: Commit
Time 151 ms: Agent B: Retry succeeds → Proceed
Total wait: 51 ms (transparent to agent)
```

**Scenario 2: Extended Lock (> 5 seconds)**
```
Time 0 ms: Agent A: Begin expensive operation
Time 100 ms: Agent B: Try to read → BUSY
Time 5100 ms: Timeout reached, still locked
Result: SQLITE_BUSY error returned to agent
Agent B must handle error (retry, fail, etc.)
```

**Timeout Value Justification**:
```
5 seconds = reasonable timeout for:
  - Single task update: ~10-100 ms (plenty of buffer)
  - Feature progress recalc: ~200-500 ms (plenty of buffer)
  - Bulk operations: ~1000-3000 ms (within timeout)
  - Deadlock detection: 5 sec > typical operation time

Too short (100 ms):
  - Risk timeout on legitimate slow operations
  - CLI tool might fail unnecessarily

Too long (30 seconds):
  - User waits a long time for error
  - Poor user experience

5 seconds = sweet spot
```

**Application Impact**:
```go
// Application doesn't need explicit retry logic
// SQLite handles retries internally

err := db.Exec("UPDATE tasks SET status = ? WHERE id = ?", "in_progress", 42)
// If database is locked:
//   1. SQLite waits up to 5 seconds
//   2. If lock released: Succeeds transparently
//   3. If lock held 5 seconds: Returns error

// Application decides how to handle:
if err != nil {
    log.Errorf("Database operation failed: %v", err)
    // Retry, fail, or queue for later
}
```

**Concurrency Implication**:
- CLI: Single operations (no multi-agent contention)
- API: May see temporary contention if multiple agents run simultaneously
- Timeout ensures graceful degradation vs. immediate failure

---

### 4. synchronous = NORMAL

**Location**: `internal/db/db.go` line 41

```sql
PRAGMA synchronous = NORMAL;
```

**Purpose**:
- Balance data safety and write performance
- Reduce fsync() calls to disk
- Recover quickly from crashes

**Synchronous Levels**:

| Level | fsync() calls | Safety | Speed | Best for |
|-------|---------------|--------|-------|----------|
| OFF (0) | 0 | Poorest (data loss on crash) | Fastest | Dev/test only |
| NORMAL (1) | 1 per transaction | Good (with WAL) | Faster | **Production** |
| FULL (2) | 2+ per transaction | Best | Slowest | Critical systems |
| EXTRA (3) | 3+ per transaction | Maximum | Very slow | Ultra-critical |

**How It Works**:

**NORMAL Mode (Shark Task Manager)**:
```
Transaction:
  1. Write to WAL file → fsync (1 call)
  2. Write to main file
  3. Write to WAL index → fsync (0 calls in NORMAL mode)

Result: 1 fsync per transaction
Cost: ~10-50 ms per transaction (typical SSD)
Risk: Very low with WAL mode enabled
```

**FULL Mode (Alternative)**:
```
Transaction:
  1. Write to main file → fsync (1 call)
  2. Write to WAL → fsync (1 call)
  3. Write index → fsync (1 call)

Result: 3 fsyncs per transaction
Cost: ~30-150 ms per transaction (3x slower)
Risk: Nearly zero
Trade: Speed for safety
```

**Why NORMAL for Shark Task Manager**:

**Safety Analysis with WAL**:
```
Crash scenario: Power loss during task update

NORMAL + WAL:
  1. Change written to WAL and synced
  2. Main file changes in cache (not synced yet)
  3. Power loss!
  4. Recovery: Replay WAL changes to main file
  5. Result: Task update recovered ✓

FULL + WAL:
  1. Change written to WAL and synced
  2. Change written to main file and synced
  3. Power loss after step 2: Already safe ✓

Result: Both NORMAL and FULL are safe with WAL.
NORMAL is sufficient and faster.
```

**Write Performance Comparison**:
```
Task update (single UPDATE statement):

NORMAL: ~10-20 ms (1 fsync)
FULL: ~30-60 ms (multiple fsyncs)

100 concurrent task updates:
NORMAL: ~1-2 seconds total (batched fsyncs)
FULL: ~3-6 seconds total (more fsyncs)

Improvement: 50-66% faster with NORMAL
```

**Crash Recovery**:
```
With NORMAL + WAL:
  Uncommitted transactions: Lost (expected)
  Committed transactions: Recovered from WAL (safe)
  Database integrity: Guaranteed by fsync + WAL
```

---

### 5. cache_size = -64000

**Location**: `internal/db/db.go` line 42

```sql
PRAGMA cache_size = -64000;
```

**Purpose**:
- Set in-memory cache to 64 MB (negative = megabytes)
- Reduce disk I/O by caching frequently accessed pages
- Improve query performance

**Default**: -2000 (approximately 2 MB)

**How Cache Works**:

**Cache Hit**:
```
Query: SELECT * FROM tasks WHERE key = 'T-E04-F06-001'

1. Check cache for task pages → FOUND
2. Return from RAM (~1 µs)
3. Total: ~1-5 ms query time
```

**Cache Miss** (hits disk):
```
Query: SELECT * FROM tasks WHERE key = 'T-E04-F06-001'

1. Check cache for task pages → NOT FOUND
2. Read from disk → ~5-10 ms (SSD)
3. Cache the pages
4. Return result
5. Total: ~10-20 ms query time
```

**Memory Trade-offs**:

| Cache Size | Memory Used | Hit Rate | Query Speed | System Impact |
|------------|------------|----------|-------------|---------------|
| 0 (no cache) | 0 MB | 0% | Slowest | None |
| 2 MB | 2 MB | 20-40% | Slow | Minimal |
| **64 MB** | **64 MB** | **70-90%** | **Fast** | **Good** |
| 256 MB | 256 MB | 95%+ | Very fast | High memory use |

**64 MB Justification**:

**For Shark Task Manager**:
```
Typical database size: 5-50 MB
- 200 tasks × 5 KB per task = 1 MB
- 50 features × 2 KB = 100 KB
- 10 epics × 1 KB = 10 KB
- Indexes: ~100 KB
- Total: ~1-2 MB

64 MB cache = 32-64x database size
Result: Entire database fits in cache after first access
Hit rate: 90%+ (very high)
```

**Query Performance Impact**:
```
Without cache (2 MB):
  - Task lookup: ~15-20 ms (cache miss, disk I/O)
  - Feature progress: ~50-100 ms (multiple disk reads)
  - List tasks: ~30-50 ms (multiple pages)

With 64 MB cache:
  - Task lookup: ~2-5 ms (cache hit, RAM only)
  - Feature progress: ~5-15 ms (all pages cached)
  - List tasks: ~5-10 ms (all pages cached)

Improvement: 3-10x faster queries
```

**Memory Usage Context**:
```
64 MB cache:
  - Server with 512 MB RAM: 12.5% of RAM (acceptable)
  - Server with 4 GB RAM: 1.6% of RAM (negligible)
  - CLI process: 64 MB temporary (collected on exit)

Trade: Small memory cost for significant speed improvement
```

**Page Size**:
```sql
PRAGMA page_size;
-- Returns: 4096 (4 KB per page, typical)

64 MB cache = 16,384 pages
Typical query: 1-10 pages per result (usually RAM)
```

---

### 6. temp_store = MEMORY

**Location**: `internal/db/db.go` line 43

```sql
PRAGMA temp_store = MEMORY;
```

**Purpose**:
- Store temporary tables and indices in memory
- Avoid temporary disk I/O
- Improve performance for complex queries

**Default**: DEFAULT (decides based on availability)

**When Temp Storage Is Used**:

**Scenario 1: DISTINCT on Large Result**
```sql
SELECT DISTINCT status FROM tasks;

Without temp_store = MEMORY:
  1. Create temp table on disk
  2. Sort results to disk (~5-10 MB)
  3. Remove duplicates
  4. Return results
  5. Time: 100-500 ms

With temp_store = MEMORY:
  1. Create temp table in RAM
  2. Sort results in RAM (if < available memory)
  3. Remove duplicates
  4. Return results
  5. Time: 5-20 ms
```

**Scenario 2: Complex JOIN with ORDER BY**
```sql
SELECT t.*, f.title, e.title
FROM tasks t
JOIN features f ON t.feature_id = f.id
JOIN epics e ON f.epic_id = e.id
ORDER BY t.priority DESC
LIMIT 50;

With temp_store = MEMORY:
  - Temporary join results in RAM
  - Sorting in RAM
  - Much faster than disk sorting
```

**Scenario 3: Aggregate GROUP BY**
```sql
SELECT status, COUNT(*) as count
FROM tasks
GROUP BY status;

With temp_store = MEMORY:
  - Temp aggregation table in RAM
  - Fast grouping
  - No disk I/O
```

**Trade-offs**:

**Pros**:
- Faster complex queries (2-50x improvement)
- Reduced disk I/O
- Temporary data doesn't persist (memory only)

**Cons**:
- Uses RAM (acceptable, temp data cleared after query)
- If insufficient RAM: Falls back to disk (but slower)
- Very large result sets might exhaust RAM

**For Shark Task Manager**:
```
Typical queries:
  - Task lookups: No temp storage needed (simple queries)
  - Feature progress: Grouping/aggregation (benefits from temp_store)
  - List operations: Simple filtering (minimal temp storage)

Overall: Beneficial for dashboard/reporting queries
```

**Memory Available**:
```
64 MB cache (PRAGMA cache_size)
+ Temp query buffers (usually < 10 MB)
= Total RAM use: 64-75 MB reasonable

System check: Any modern system has > 256 MB available
```

---

### 7. mmap_size = 30000000000

**Location**: `internal/db/db.go` line 44

```sql
PRAGMA mmap_size = 30000000000;
```

**Purpose**:
- Enable memory-mapped I/O for large databases
- 30 GB mapping size (large virtual address space)
- Faster random access to database pages

**What Is Memory-Mapped I/O**:

**Traditional I/O**:
```
Read database page:
  1. Application calls read()
  2. System call to kernel
  3. Kernel reads from disk to buffer
  4. Kernel copies to application memory
  5. Application reads value

Overhead: 2 copies, 1 system call per page
Time: 5-10 µs per page
```

**Memory-Mapped I/O (mmap)**:
```
Setup: Database mapped to virtual address space
  1. Database pages appear as regular RAM

Read database page:
  1. Application reads virtual address
  2. If in RAM: Direct access (~10 ns)
  3. If on disk: Page fault (OS handles transparently)
  4. OS reads from disk, returns

Overhead: 0 copies (after initial setup), 1 page fault if needed
Time: 10 ns (cache hit) or 5-10 µs (page fault)
```

**mmap_size Value Meaning**:
```
mmap_size = 30000000000 = 30 GB

Meaning: Reserve 30 GB virtual address space for database mapping
Actual usage: Only maps actual database file size (unused space is virtual)

For 5 MB database:
  - Virtual reservation: 30 GB
  - Actual memory use: 5 MB (or less if cached)
  - Benefit: Entire database can be mapped into address space
```

**Performance Impact**:

**Without mmap** (traditional I/O):
```
Query: Find task by key (SELECT * FROM tasks WHERE key = ?)
  1. Read index page from disk: 5-10 µs
  2. Find key in index: 1-2 µs
  3. Read data page from disk: 5-10 µs
  4. Return data: 1-2 µs
  Total: 12-24 µs

Cache hit (in buffer cache):
  1. Read from buffer: 100-500 ns
  2. Find key: 1-2 µs
  3. Read data: 100-500 ns
  4. Return: 1-2 µs
  Total: 2-6 µs
```

**With mmap** (memory-mapped I/O):
```
Query: Same task lookup
  1. Access virtual address (index): 10-20 ns (page fault triggers read)
  2. Find key: 1-2 µs
  3. Access virtual address (data): 10-20 ns
  4. Return data: 1-2 µs
  Total: 2-5 µs (similar, but simpler)

Cache hit (in RAM):
  1. Access from RAM: 10 ns
  2. Find key: 1-2 µs
  3. Access data: 10 ns
  4. Return: 1-2 µs
  Total: 2-3 µs (slightly faster)

Primary benefit: More efficient paging than read() calls
```

**30 GB Limit**:
```
mmap_size = 0: Disable mmap (use traditional I/O)
mmap_size = 1000000: Map 1 MB only
mmap_size = 30000000000: Map 30 GB (current setting)

For Shark Task Manager:
  - Database: 5-50 MB typically
  - 30 GB limit: Future-proof, doesn't constrain
  - Virtual address space: 64-bit systems have plenty
```

**Compatibility**:
```
mmap supported on:
  - Linux ✓
  - macOS ✓
  - Windows (7+) ✓
  - Mobile platforms ✓

Not supported on:
  - Some network filesystems
  - Some embedded systems

For Shark Task Manager: Universal support
```

**Trade-offs**:

**Pros**:
- More efficient paging
- Faster random access
- Works well with OS page cache

**Cons**:
- Uses virtual address space (not a concern on 64-bit)
- Not supported on all filesystems
- Slightly different behavior on crash (rare)

**For Shark Task Manager**:
```
Benefit: Marginal for small databases (5-50 MB)
Reason: Database already fits in buffer cache
Trade: No downside, optional optimization
Status: Good to have, not critical
```

---

## Configuration Impact Analysis

### Combined Performance Effect

**Scenario: 500 tasks, multiple agents querying**

**Without Optimized PRAGMAs**:
```
Initial load (cold cache):
  - Task lookup: ~20 ms (disk I/O)
  - Feature progress: ~150 ms (multiple disk reads)
  - List 50 tasks: ~100 ms

Subsequent queries (warm cache):
  - Task lookup: ~5 ms
  - Feature progress: ~20 ms
  - List 50 tasks: ~15 ms

Contentious writes:
  - Update task: Immediate SQLITE_BUSY (no retry)
  - Bulk update: May timeout
```

**With Optimized PRAGMAs**:
```
Initial load (cold cache):
  - Task lookup: ~5 ms (cache hit after first)
  - Feature progress: ~15 ms
  - List 50 tasks: ~10 ms

Subsequent queries (warm cache):
  - Task lookup: ~2 ms
  - Feature progress: ~5 ms
  - List 50 tasks: ~3 ms

Contentious writes:
  - Update task: Retry for up to 5 seconds (graceful)
  - Bulk update: Succeeds or times out gracefully
```

**Improvement**: 2-3x faster queries, better concurrency handling

### Single-User vs. Multi-User

**Single CLI User**:
```
Impact of each PRAGMA:
  - foreign_keys: Required for integrity (not performance)
  - journal_mode: Minimal impact (no concurrency)
  - busy_timeout: No contention (not triggered)
  - synchronous: Small benefit
  - cache_size: Large benefit (warm cache)
  - temp_store: Benefit only for complex queries
  - mmap_size: Minimal benefit

Overall: Better query performance, same write speed
```

**Multi-User API Server**:
```
Impact of each PRAGMA:
  - foreign_keys: Required for integrity
  - journal_mode: LARGE benefit (readers don't block writers)
  - busy_timeout: LARGE benefit (graceful retry)
  - synchronous: Medium benefit
  - cache_size: Large benefit (shared across users)
  - temp_store: Medium benefit
  - mmap_size: Medium benefit (faster paging)

Overall: Excellent concurrency, good query speed
```

---

## Data Integrity Guarantees

### PRAGMA Combinations for Safety

**This Configuration Provides**:

1. **Foreign Key Integrity**:
   - No orphaned records
   - Referential consistency guaranteed
   - Cascading deletes work correctly

2. **Crash Recovery**:
   - WAL mode + NORMAL synchronous
   - Committed changes recovered
   - Database never left in inconsistent state

3. **Concurrent Access**:
   - WAL allows reader/writer concurrency
   - busy_timeout prevents race conditions
   - Multiple agents can access simultaneously

4. **Query Consistency**:
   - Large cache reduces stale data reads
   - Memory-mapped I/O provides consistent view
   - Proper isolation level maintained

**Verification**:
```sql
-- Check all critical PRAGMAs
PRAGMA foreign_keys;        -- Should be: 1
PRAGMA journal_mode;        -- Should be: wal
PRAGMA busy_timeout;        -- Should be: 5000
PRAGMA synchronous;         -- Should be: 1 (NORMAL)
PRAGMA cache_size;          -- Should be: -64000
PRAGMA temp_store;          -- Should be: 1 (MEMORY)
PRAGMA mmap_size;           -- Should be: 30000000000
```

---

## Configuration Best Practices

### Applied in This Project

✓ **Explicit configuration**: Not relying on defaults
✓ **Safety first**: Foreign keys enabled
✓ **Concurrency ready**: WAL mode for multi-user
✓ **Graceful degradation**: Busy timeout prevents hard failures
✓ **Performance optimized**: Cache and temp storage
✓ **Future-proof**: 30 GB mmap for growth

### Could Be Added

? **PRAGMA optimize**: Auto-analyze statistics
  - Benefit: Query planner updates statistics
  - Trigger: Periodically (e.g., weekly)

? **PRAGMA incremental_vacuum**: Reclaim disk space
  - Benefit: Database file smaller after deletes
  - Trade: More I/O

? **PRAGMA query_only**: Prevent writes (read-only mode)
  - Benefit: Safety for backup processes
  - Use: Snapshot queries

---

## Troubleshooting PRAGMA Settings

### Check Current Configuration

```bash
sqlite3 shark-tasks.db "PRAGMA foreign_keys; PRAGMA journal_mode; PRAGMA busy_timeout;"
```

### Verify WAL Files

```bash
ls -la shark-tasks.db*

# Expected output:
# -rw-r--r-- 1 user group 10485760 Dec 19 15:30 shark-tasks.db
# -rw-r--r-- 1 user group    65536 Dec 19 15:30 shark-tasks.db-shm
# -rw-r--r-- 1 user group   327680 Dec 19 15:30 shark-tasks.db-wal

# If WAL files missing: Check journal_mode
```

### Disable WAL (Troubleshooting Only)

```sql
PRAGMA journal_mode = DELETE;
-- WAL files will be merged and deleted
-- Not recommended for production
```

### Clear Cache (if memory bloated)

```sql
PRAGMA shrink_memory;
-- Releases unused cache memory
-- Not recommended during active use
```

---

## Performance Tuning Further

### If Database Is Slow

1. **Check index usage**:
   ```bash
   sqlite3 shark-tasks.db "EXPLAIN QUERY PLAN SELECT * FROM tasks WHERE status = 'todo';"
   ```

2. **Increase cache if available**:
   ```sql
   PRAGMA cache_size = -256000;  -- 256 MB (if system has RAM)
   ```

3. **Run ANALYZE** (if needed):
   ```sql
   ANALYZE;
   PRAGMA optimize;  -- SQLite 3.20+
   ```

4. **Check for missing indexes**:
   - Review query plans
   - Add indexes for frequent WHERE clauses

### If Database Is Disk-Heavy

1. **Reduce WAL checkpoint interval**:
   ```sql
   PRAGMA wal_autocheckpoint = 10000;  -- Checkpoint more often
   ```

2. **Compact database**:
   ```sql
   VACUUM;  -- Rebuild database, reclaim space
   ```

### If Writes Are Slow

1. **Use PRAGMA synchronous = FULL** (if crash risk acceptable):
   ```sql
   PRAGMA synchronous = FULL;  -- Safer, slower
   ```

2. **Batch operations in transactions**:
   ```sql
   BEGIN;
   UPDATE task SET ... WHERE id = 1;
   UPDATE task SET ... WHERE id = 2;
   ...
   COMMIT;
   -- Faster than individual transactions
   ```

---

## Conclusion

The SQLite configuration in Shark Task Manager is production-ready, balancing:

- **Safety**: Foreign keys, WAL, proper synchronous level
- **Performance**: 64 MB cache, memory-mapped I/O, memory temp storage
- **Concurrency**: WAL mode, busy timeout for multi-user scenarios
- **Maintainability**: Explicit settings, well-documented

These settings provide a solid foundation for both CLI (single-user) and API (multi-user) operation.

