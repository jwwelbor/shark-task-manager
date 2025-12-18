# Architecture: Incremental Sync Engine

**Epic**: E06-intelligent-scanning
**Feature**: E06-F04-incremental-sync-engine
**Date**: 2025-12-17
**Author**: architect

## Purpose

This document defines the architecture for the incremental sync engine that tracks file modification times and only processes changed files since the last sync. It extends the existing E04-F07 synchronization system with intelligent change detection, conflict resolution, and performance optimization.

---

## System Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLI Layer (Cobra)                               │
│                                                                               │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │   shark sync                                                            │  │
│  │   (sync.go)                                                             │  │
│  │                                                                          │  │
│  │  Existing Flags:            New Flags (E06-F04):                       │  │
│  │   --folder                   --incremental (enable incremental mode)   │  │
│  │   --dry-run                  --force-full-scan (override last_sync)    │  │
│  │   --strategy                                                            │  │
│  │   --json                                                                │  │
│  └──────────────────────────────────┬─────────────────────────────────────┘  │
│                                     │                                         │
└─────────────────────────────────────┼─────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Application Layer (Business Logic)                   │
│                                                                               │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │  SyncEngine (internal/sync/engine.go) - EXTENDED                       │  │
│  │                                                                          │  │
│  │  Existing:                        New (E06-F04):                        │  │
│  │   + Sync(opts)                     + LoadLastSyncTime(config)          │  │
│  │     1. Scan files                  + FilterChangedFiles(files, time)   │  │
│  │     2. Parse frontmatter           + DetectTimestampConflicts()        │  │
│  │     3. Query database              + SaveLastSyncTime(config, time)    │  │
│  │     4. Detect conflicts                                                 │  │
│  │     5. Resolve conflicts                                                │  │
│  │     6. Update database                                                  │  │
│  │     7. Generate report                                                  │  │
│  └──────────────────────────────────────┬───────────────────────────────────┘  │
│                                          │                                    │
│  ┌──────────────────────┐  ┌────────────┴───────────┐  ┌──────────────────┐  │
│  │  FileScanner         │  │  ConflictDetector      │  │  ConflictResolver│  │
│  │  (scanner.go)        │  │  (conflict.go)         │  │  (resolver.go)   │  │
│  │  - EXTENDED          │  │  - EXTENDED            │  │                  │  │
│  │                      │  │                        │  │                  │  │
│  │  + FilterByMtime()   │  │  + DetectTimestamp     │  │  + Resolve()     │  │
│  │  + GetMtime(file)    │  │    Conflicts()         │  │                  │  │
│  └──────────────────────┘  └────────────────────────┘  └──────────────────┘  │
└───────────────────────────────────────┬─────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Configuration Layer (Filesystem)                          │
│                                                                               │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │  .sharkconfig.json                                                      │  │
│  │                                                                          │  │
│  │  Existing Fields:                 New Field (E06-F04):                 │  │
│  │   {                                {                                    │  │
│  │     "default_epic": null,            "last_sync_time":                 │  │
│  │     "default_agent": null,             "2025-12-17T14:30:45-08:00"    │  │
│  │     "color_enabled": true,         }                                    │  │
│  │     "json_output": false                                                │  │
│  │   }                                                                      │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Data Access Layer (Repository Pattern)                    │
│                                                                               │
│  ┌──────────────────────┐  ┌──────────────────────┐  ┌───────────────────┐  │
│  │  TaskRepository      │  │  EpicRepository      │  │  FeatureRepository│  │
│  │  - EXTENDED          │  │                      │  │                   │  │
│  │                      │  │                      │  │                   │  │
│  │  + GetUpdatedSince() │  │  + GetByKey()        │  │  + GetByKey()     │  │
│  │    (for conflict     │  │                      │  │                   │  │
│  │     detection)       │  │                      │  │                   │  │
│  └──────────────────────┘  └──────────────────────┘  └───────────────────┘  │
└───────────────────────────────────────┬─────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Database Layer (SQLite)                             │
│                                                                               │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐  ┌───────────┐  │
│  │  tasks         │  │  features      │  │  epics         │  │  task_    │  │
│  │                │  │                │  │                │  │  history  │  │
│  │  updated_at ◄──┼──┼────────────────┼──┼────────────────┼──┼───────────┼──┐
│  │  (timestamp    │  │  updated_at    │  │  updated_at    │  │           │  │
│  │   comparison)  │  │                │  │                │  │           │  │
│  └────────────────┘  └────────────────┘  └────────────────┘  └───────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Architecture Principles

For this POC-level feature, we follow these principles:

1. **Extend, Don't Rewrite**: Build on top of E04-F07's proven sync engine
2. **Simple Timestamp Comparison**: Use filesystem mtime vs. last_sync_time (no Git integration yet)
3. **Config-Based State**: Store last_sync_time in .sharkconfig.json (no separate state file)
4. **Fail-Safe Defaults**: Missing last_sync_time → automatic full scan (backward compatible)
5. **Transaction Safety**: Maintain E04-F07's atomic transaction model

---

## Component Architecture

### 1. Enhanced Sync Engine

#### Incremental Sync Flow

```
Sync(ctx, opts):
  ├─> Step 0: Load Configuration
  │     ├─> Read .sharkconfig.json
  │     ├─> Extract last_sync_time (nullable)
  │     └─> Capture current_sync_time = NOW()
  │
  ├─> Step 1: Scan Files (EXISTING)
  │     ├─> FileScanner.Scan(rootPath)
  │     └─> Return []FileInfo with mtime
  │
  ├─> Step 2: Filter Changed Files (NEW)
  │     ├─> If --force-full-scan: Skip filtering (process all)
  │     ├─> If last_sync_time == nil:
  │     │     ├─> Log warning: "No last_sync_time, performing full scan"
  │     │     └─> Skip filtering (process all)
  │     ├─> If --incremental:
  │     │     ├─> For each file:
  │     │     │     ├─> Compare file.mtime > last_sync_time
  │     │     │     ├─> If true: Include in changed_files
  │     │     │     └─> If false: Skip file
  │     │     └─> Return filtered list
  │     └─> Else: Process all files (backward compatible)
  │
  ├─> Step 3: Parse Files (EXISTING, but only changed files)
  │     ├─> For each changed file:
  │     │     ├─> Parse frontmatter
  │     │     └─> Build TaskMetadata
  │     └─> Return []TaskMetadata
  │
  ├─> Step 4: Query Database (EXISTING)
  │     ├─> Extract task keys
  │     ├─> TaskRepository.GetByKeys(keys)
  │     └─> Return map[key]*Task
  │
  ├─> Step 5: Begin Transaction (EXISTING)
  │     ├─> db.BeginTx(ctx, nil)
  │     └─> defer tx.Rollback()
  │
  ├─> Step 6: Detect Conflicts (ENHANCED)
  │     ├─> For each task:
  │     │     ├─> If task not in DB: No conflict (new task)
  │     │     ├─> If file.mtime <= last_sync_time: No conflict (file unchanged)
  │     │     ├─> If dbTask.updated_at <= last_sync_time: No conflict (db unchanged)
  │     │     ├─> If BOTH file.mtime AND dbTask.updated_at > last_sync_time:
  │     │     │     ├─> CONFLICT DETECTED
  │     │     │     ├─> Compare metadata fields (title, description, status)
  │     │     │     └─> Add to conflicts list
  │     │     └─> Return []Conflict
  │
  ├─> Step 7: Resolve Conflicts (EXISTING strategy, NEW detection logic)
  │     ├─> Apply --strategy (file-wins, db-wins, manual)
  │     ├─> Build resolved task models
  │     └─> Log conflict resolutions
  │
  ├─> Step 8: Update Database (EXISTING)
  │     ├─> Insert new tasks
  │     ├─> Update existing tasks
  │     └─> Create history records
  │
  ├─> Step 9: Commit Transaction (EXISTING)
  │     └─> tx.Commit()
  │
  ├─> Step 10: Update Last Sync Time (NEW)
  │     ├─> If transaction successful:
  │     │     ├─> Write last_sync_time = current_sync_time to config
  │     │     └─> Atomic file write (temp + rename)
  │     └─> If transaction failed: Skip (preserve old timestamp)
  │
  └─> Return SyncReport (with performance metrics)
```

### 2. Configuration Management

#### Config Schema Extension

```go
// Existing config structure (internal/init/types.go)
type ConfigDefaults struct {
    DefaultEpic  *string `json:"default_epic"`
    DefaultAgent *string `json:"default_agent"`
    ColorEnabled bool    `json:"color_enabled"`
    JSONOutput   bool    `json:"json_output"`

    // NEW FIELD (E06-F04)
    LastSyncTime *time.Time `json:"last_sync_time,omitempty"`
}
```

#### Config Operations

**Load Config**:
```go
func LoadConfig(configPath string) (*ConfigDefaults, error) {
    // Read .sharkconfig.json
    // Unmarshal JSON
    // Validate last_sync_time format (RFC3339)
    // Return config struct
}
```

**Save Last Sync Time**:
```go
func SaveLastSyncTime(configPath string, syncTime time.Time) error {
    // Read existing config
    // Update last_sync_time field
    // Marshal to JSON
    // Write to temp file
    // Atomic rename (temp → .sharkconfig.json)
}
```

**Validation**:
- `last_sync_time` must be RFC3339 format with timezone
- Invalid format → log warning, proceed with full scan
- Missing field → null (treated as first sync)

### 3. File Change Detection

#### Modification Time Comparison

```go
type FileInfo struct {
    FilePath    string
    FileName    string
    EpicKey     string
    FeatureKey  string
    ModifiedAt  time.Time  // EXISTING field from os.FileInfo
}

func (e *SyncEngine) FilterChangedFiles(files []FileInfo, lastSyncTime *time.Time) []FileInfo {
    // If lastSyncTime is nil, return all files (full scan)
    if lastSyncTime == nil {
        return files
    }

    changedFiles := []FileInfo{}
    for _, file := range files {
        // Compare file mtime with last sync time
        if file.ModifiedAt.After(*lastSyncTime) {
            changedFiles = append(changedFiles, file)
        }
    }

    return changedFiles
}
```

#### Clock Skew Handling

```go
func (e *SyncEngine) CheckClockSkew(fileTime time.Time, now time.Time) bool {
    // Check if file is in the future
    if fileTime.After(now) {
        skew := fileTime.Sub(now)

        // Allow small skew (±60 seconds)
        if skew <= 60*time.Second {
            return false  // No warning
        }

        // Large skew (>60 seconds)
        e.logWarning("File has future mtime (clock skew: %v): %s", skew, file)
        return true
    }

    return false
}
```

### 4. Enhanced Conflict Detection

#### Timestamp-Based Conflict Detection

```go
type ConflictDetector struct {
    // Existing fields
}

func (d *ConflictDetector) DetectConflicts(
    fileData *TaskMetadata,
    dbTask *models.Task,
    lastSyncTime *time.Time,
) []Conflict {
    conflicts := []Conflict{}

    // NEW: Check if this is a timestamp conflict
    // (both file AND database modified since last sync)
    isTimestampConflict := false
    if lastSyncTime != nil {
        fileMtime := fileData.ModifiedAt
        dbUpdated := dbTask.UpdatedAt

        if fileMtime.After(*lastSyncTime) && dbUpdated.After(*lastSyncTime) {
            isTimestampConflict = true
        }
    }

    // If no timestamp conflict, there's no conflict at all
    // (either only file changed OR only database changed)
    if !isTimestampConflict {
        return conflicts  // Empty list
    }

    // EXISTING: Check metadata conflicts (title, description, etc.)
    // But ONLY if timestamp conflict exists
    if fileData.Title != "" && fileData.Title != dbTask.Title {
        conflicts = append(conflicts, Conflict{
            TaskKey:       dbTask.Key,
            Field:         "title",
            FileValue:     fileData.Title,
            DatabaseValue: dbTask.Title,
            FileMtime:     fileData.ModifiedAt,
            DBUpdatedAt:   dbTask.UpdatedAt,
        })
    }

    // ... check other fields ...

    return conflicts
}
```

#### Conflict Structure Extension

```go
type Conflict struct {
    TaskKey       string
    Field         string
    FileValue     string
    DatabaseValue string

    // NEW FIELDS (E06-F04)
    FileMtime     time.Time  // File modification time
    DBUpdatedAt   time.Time  // Database updated_at timestamp
}
```

### 5. Repository Extensions

#### Query Tasks Updated Since Timestamp

```go
// TaskRepository extension (internal/repository/task_repository.go)
func (r *TaskRepository) GetUpdatedSince(ctx context.Context, since time.Time) ([]*models.Task, error) {
    query := `
        SELECT id, feature_id, key, title, description, status, agent_type, priority,
               depends_on, assigned_agent, file_path, blocked_reason,
               created_at, started_at, completed_at, blocked_at, updated_at
        FROM tasks
        WHERE updated_at > ?
        ORDER BY updated_at DESC
    `

    rows, err := r.db.QueryContext(ctx, query, since)
    if err != nil {
        return nil, fmt.Errorf("failed to query updated tasks: %w", err)
    }
    defer rows.Close()

    // Scan rows into task models
    tasks := []*models.Task{}
    for rows.Next() {
        task := &models.Task{}
        err := rows.Scan(/* ... fields ... */)
        if err != nil {
            return nil, fmt.Errorf("failed to scan task: %w", err)
        }
        tasks = append(tasks, task)
    }

    return tasks, rows.Err()
}
```

---

## Data Flow Diagrams

### Incremental Sync Flow

```
User
 │
 ├─> shark sync --incremental
 │
 ▼
SyncCommand (CLI)
 ├─> Parse flags (--incremental, --force-full-scan)
 ├─> Create SyncOptions with incremental=true
 │
 ▼
SyncEngine
 │
 ├─> Step 0: Load Configuration
 │     ├─> Read .sharkconfig.json
 │     ├─> last_sync_time = "2025-12-17T12:00:00Z" (or null)
 │     └─> current_sync_time = NOW() = "2025-12-17T14:30:00Z"
 │
 ├─> Step 1: Scan Files
 │     ├─> Walk docs/plan recursively
 │     ├─> Find 287 files (all task files)
 │     ├─> Extract mtime for each file
 │     └─> Return []FileInfo (287 files)
 │
 ├─> Step 2: Filter Changed Files (NEW)
 │     ├─> For each of 287 files:
 │     │     ├─> File A: mtime = 12:10:00 → SKIP (< last_sync)
 │     │     ├─> File B: mtime = 13:45:00 → INCLUDE (> last_sync)
 │     │     ├─> File C: mtime = 14:20:00 → INCLUDE (> last_sync)
 │     │     └─> ... (285 files skipped, 5 files included)
 │     └─> Return 5 changed files
 │
 ├─> Step 3: Parse Files (only 5 files, not 287)
 │     ├─> Parse frontmatter for 5 changed files
 │     └─> Return 5 TaskMetadata objects
 │
 ├─> Step 4: Query Database
 │     ├─> Extract keys: [T-E04-F06-003, T-E04-F06-007, ...]
 │     ├─> GetByKeys([...5 keys...])
 │     ├─> Find: 2 existing tasks, 3 new tasks
 │     └─> Return map[key]*Task (2 entries)
 │
 ├─> Step 5: Detect Conflicts (for 2 existing tasks)
 │     ├─> Task T-E04-F06-003:
 │     │     ├─> file.mtime = 14:20:00 (> last_sync)
 │     │     ├─> db.updated_at = 13:30:00 (> last_sync)
 │     │     ├─> BOTH modified since last sync → CONFLICT
 │     │     ├─> Compare metadata:
 │     │     │     ├─> title: DB="Old Title", File="New Title" → CONFLICT
 │     │     │     └─> status: DB="completed", File="completed" → NO CONFLICT
 │     │     └─> Add 1 conflict (field: title)
 │     │
 │     └─> Task T-E04-F06-007:
 │           ├─> file.mtime = 13:45:00 (> last_sync)
 │           ├─> db.updated_at = 11:00:00 (< last_sync)
 │           ├─> Only file modified → NO CONFLICT
 │           └─> Update database with file values
 │
 ├─> Step 6: Resolve Conflicts (1 conflict)
 │     ├─> Strategy = file-wins (default)
 │     ├─> Apply file value: title = "New Title"
 │     └─> Log: "Conflict in T-E04-F06-003: title: Old Title → New Title (file-wins)"
 │
 ├─> Step 7: Update Database (transaction)
 │     ├─> Insert 3 new tasks
 │     ├─> Update 2 existing tasks
 │     └─> Create 5 history records
 │
 ├─> Step 8: Commit Transaction
 │     └─> All operations successful
 │
 ├─> Step 9: Update Last Sync Time (NEW)
 │     ├─> Write last_sync_time = "2025-12-17T14:30:00Z" to config
 │     └─> Atomic file write
 │
 └─> Return SyncReport:
       ├─> files_scanned: 287
       ├─> files_changed: 5
       ├─> files_created: 3
       ├─> files_updated: 2
       ├─> conflicts_detected: 1
       ├─> conflicts_resolved: 1 (file-wins)
       └─> elapsed_time: 1.8 seconds
```

### Force Full Scan Flow

```
User
 │
 ├─> shark sync --incremental --force-full-scan
 │
 ▼
SyncEngine
 │
 ├─> Load last_sync_time (ignored due to --force-full-scan)
 ├─> Scan all 287 files
 ├─> Skip filtering (process all files)
 ├─> Parse all 287 files
 ├─> Query database for all 287 tasks
 ├─> Detect conflicts (same logic)
 ├─> Update database
 └─> Save new last_sync_time
```

---

## Performance Optimization

### Incremental Sync Performance Targets

| Scenario | Files Scanned | Files Processed | Target Time | Strategy |
|----------|--------------|-----------------|-------------|----------|
| **Typical session** | 287 | 5-7 | <2 seconds | Mtime filtering |
| **After Git pull** | 287 | 10-20 | <3 seconds | Mtime filtering |
| **Large changes** | 500 | 100 | <5 seconds | Mtime filtering + bulk ops |
| **Full scan** | 500 | 500 | <30 seconds | No filtering (E04-F07 performance) |

### Optimization Strategies

**1. Fast Mtime Filtering**:
```go
// O(n) time complexity where n = total files
// But each mtime comparison is ~1ns (timestamp comparison)
// For 500 files: ~500ns = 0.0005ms overhead
```

**2. Filesystem Stat Efficiency**:
```go
// os.FileInfo.ModTime() is already fetched during Walk
// No additional filesystem syscalls needed
func (s *FileScanner) Scan(rootPath string) ([]FileInfo, error) {
    files := []FileInfo{}

    filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        // info.ModTime() is FREE (already fetched by Walk)
        files = append(files, FileInfo{
            FilePath:   path,
            ModifiedAt: info.ModTime(),  // No extra syscall
        })
    })
}
```

**3. Reduced Parsing Overhead**:
- Only parse changed files (5 instead of 287)
- YAML parsing is expensive (~5-10ms per file)
- Savings: 282 files × 8ms = 2.26 seconds saved

**4. Reduced Database Queries**:
- Only query tasks for changed files (5 keys instead of 287)
- Single IN clause query: O(log n) with index
- Savings: Minimal (queries are already bulk), but cleaner

---

## Error Handling

### Error Scenarios

**1. Invalid last_sync_time in Config**:
```go
// Parse error → log warning, fallback to full scan
if err := time.Parse(time.RFC3339, lastSyncStr); err != nil {
    e.logWarning("Invalid last_sync_time in config: %v, performing full scan", err)
    lastSyncTime = nil  // Treat as first sync
}
```

**2. Clock Skew (Future mtime)**:
```go
// File mtime is 5 minutes in future → log warning, still process
if file.ModTime().After(now.Add(60 * time.Second)) {
    e.logWarning("File has future mtime (possible clock skew): %s", file)
}
// Continue processing (don't skip file)
```

**3. Config Write Failure**:
```go
// Failed to update last_sync_time → log error, but sync still successful
if err := SaveLastSyncTime(config, syncTime); err != nil {
    e.logError("Failed to update last_sync_time in config: %v", err)
    e.logError("Next sync will reprocess same files")
}
// Don't fail entire sync operation
```

**4. Zero Changed Files**:
```go
// No files changed → skip transaction, preserve last_sync_time
if len(changedFiles) == 0 {
    return &SyncReport{
        FilesScanned: len(allFiles),
        FilesChanged: 0,
        Message:      "No files changed since last sync",
    }, nil
}
// Don't update last_sync_time (preserve original)
```

---

## Transaction Management

### Transaction Scope (Unchanged from E04-F07)

**Single Transaction** encompasses:
1. Epic/Feature creation (if --create-missing)
2. Task inserts (new tasks)
3. Task updates (existing tasks)
4. History record inserts

**Outside Transaction**:
1. File scanning (read-only)
2. File parsing (read-only)
3. Config read/write (separate atomic operation)

### Rollback Behavior

**If transaction rolls back**:
- Database unchanged (atomic)
- `last_sync_time` NOT updated (preserves old value)
- Next sync will retry same files

**Success Condition**:
- Transaction commits successfully
- THEN update `last_sync_time`
- Order matters: database first, then config

---

## Security Considerations

### Timestamp Validation

**1. Timezone Handling**:
```go
// Always use UTC for comparisons
fileTime := info.ModTime().UTC()
lastSync := lastSyncTime.UTC()

if fileTime.After(lastSync) {
    // ...
}
```

**2. Timestamp Injection Prevention**:
```go
// Validate config timestamp on load
func ValidateTimestamp(ts string) (*time.Time, error) {
    parsed, err := time.Parse(time.RFC3339, ts)
    if err != nil {
        return nil, fmt.Errorf("invalid timestamp format: %w", err)
    }

    // Sanity check: reject absurd timestamps
    if parsed.Year() < 2020 || parsed.Year() > 2100 {
        return nil, fmt.Errorf("timestamp out of valid range: %v", parsed)
    }

    return &parsed, nil
}
```

### File Path Validation (Existing from E04-F07)

No changes needed. Existing path validation prevents traversal attacks.

---

## Testing Strategy

### Unit Tests

**1. Config Management**:
- Load config with valid last_sync_time
- Load config with null last_sync_time
- Load config with invalid last_sync_time (error handling)
- Save last_sync_time (atomic write)

**2. File Filtering**:
- Filter with last_sync_time (some files changed)
- Filter with null last_sync_time (all files included)
- Filter with all files unchanged (empty result)
- Filter with clock skew (future mtime)

**3. Conflict Detection**:
- Both file and DB modified → conflict detected
- Only file modified → no conflict
- Only DB modified → no conflict (file skipped)
- Neither modified → no conflict (file skipped)

### Integration Tests

**1. First Sync (No last_sync_time)**:
- Config has no last_sync_time
- Sync all files
- Config updated with timestamp

**2. Incremental Sync (Some Files Changed)**:
- Setup: last_sync_time set
- Modify 5 files (touch to update mtime)
- Sync processes only 5 files
- Config updated with new timestamp

**3. Conflict Scenario**:
- Setup: last_sync_time set
- Modify task in database (via CLI)
- Modify same task file
- Sync detects conflict
- Strategy resolves conflict

**4. Force Full Scan**:
- Setup: last_sync_time set
- Run sync with --force-full-scan
- All files processed
- Config updated

---

## Migration Path

### Backward Compatibility

**Existing Projects (No last_sync_time)**:
- First sync with --incremental → automatic full scan
- Config updated with timestamp
- Subsequent syncs are incremental

**Existing Projects (Regular sync)**:
- Regular sync (no --incremental flag) → full scan (E04-F07 behavior)
- Still updates last_sync_time (for future incremental syncs)
- Gradual adoption: can mix full and incremental syncs

---

## Summary

### Key Architectural Decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| **Store last_sync_time in .sharkconfig.json** | Simplicity, no separate state file | Config file grows (minimal) |
| **Use mtime comparison** | Fast, no Git dependency | Misses content changes with old mtime (rare) |
| **Null last_sync_time → full scan** | Backward compatible, fail-safe | First incremental sync is full scan |
| **Update timestamp after commit** | Data integrity (db and config in sync) | Failed config write → next sync duplicates work |
| **Extend E04-F07 engine** | Code reuse, proven transaction model | Adds complexity to existing engine |

### Component Extensions

| Component | Change Type | New Functionality |
|-----------|-------------|-------------------|
| **SyncEngine** | Extended | Load/save last_sync_time, filter changed files |
| **FileScanner** | Extended | Expose mtime in FileInfo (already present) |
| **ConflictDetector** | Extended | Timestamp-based conflict detection |
| **ConfigDefaults** | Extended | Add last_sync_time field |
| **TaskRepository** | New Method | GetUpdatedSince(timestamp) |

### Performance Impact

- **Best Case** (5 files changed): 5-10s → 1-2s (5x improvement)
- **Worst Case** (all files changed): Same as E04-F07 (no regression)
- **Overhead** (filtering): <100ms for 500 files (negligible)

---

**Document Complete**: 2025-12-17
**Next Document**: 04-backend-design.md (detailed implementation specs)
