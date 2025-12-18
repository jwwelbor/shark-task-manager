# Backend Design: Incremental Sync Engine

**Epic**: E06-intelligent-scanning
**Feature**: E06-F04-incremental-sync-engine
**Date**: 2025-12-17
**Author**: backend-architect

## Purpose

This document provides detailed backend implementation specifications for the incremental sync engine. It defines data structures, function signatures, algorithms, and implementation patterns for developers.

---

## Implementation Overview

### Files to Modify

```
internal/sync/
├── engine.go          # EXTEND: Add incremental sync logic
├── types.go           # EXTEND: Add LastSyncTime to SyncOptions
├── scanner.go         # MINOR: Already returns mtime via FileInfo
└── conflict.go        # EXTEND: Add timestamp-based conflict detection

internal/init/
├── types.go           # EXTEND: Add LastSyncTime to ConfigDefaults
└── config.go          # EXTEND: Add Save/Load methods for last_sync_time

internal/cli/commands/
└── sync.go            # EXTEND: Add --incremental and --force-full-scan flags
```

### New Files to Create

```
internal/sync/
└── timestamps.go      # NEW: Timestamp comparison utilities
```

---

## Data Structures

### 1. Configuration Extensions

#### ConfigDefaults Structure

**File**: `internal/init/types.go`

```go
type ConfigDefaults struct {
    DefaultEpic  *string `json:"default_epic"`
    DefaultAgent *string `json:"default_agent"`
    ColorEnabled bool    `json:"color_enabled"`
    JSONOutput   bool    `json:"json_output"`

    // NEW FIELD (E06-F04)
    // RFC3339 timestamp of last successful sync
    // null = first sync or never synced
    LastSyncTime *time.Time `json:"last_sync_time,omitempty"`
}
```

**Validation Rules**:
- `LastSyncTime` must be valid RFC3339 format if present
- `null` or missing field is valid (treated as first sync)
- Invalid format → log warning, treat as null

**JSON Example**:
```json
{
  "default_epic": null,
  "default_agent": null,
  "color_enabled": true,
  "json_output": false,
  "last_sync_time": "2025-12-17T14:30:45-08:00"
}
```

#### Config Operations

**File**: `internal/init/config.go`

```go
// LoadConfig reads configuration from file
// Returns nil last_sync_time if field missing or invalid
func LoadConfig(configPath string) (*ConfigDefaults, error) {
    // Read file
    data, err := os.ReadFile(configPath)
    if err != nil {
        if os.IsNotExist(err) {
            // Config doesn't exist, return defaults
            return &ConfigDefaults{
                ColorEnabled: true,
                JSONOutput:   false,
                LastSyncTime: nil,  // No previous sync
            }, nil
        }
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    // Unmarshal JSON
    var config ConfigDefaults
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    // Validate last_sync_time if present
    if config.LastSyncTime != nil {
        if err := validateTimestamp(*config.LastSyncTime); err != nil {
            log.Printf("Warning: Invalid last_sync_time in config: %v, treating as null", err)
            config.LastSyncTime = nil
        }
    }

    return &config, nil
}

// SaveConfig writes entire configuration to file atomically
func SaveConfig(configPath string, config *ConfigDefaults) error {
    // Marshal to JSON
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    // Write to temp file
    tmpPath := configPath + ".tmp"
    if err := os.WriteFile(tmpPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write temp config: %w", err)
    }

    // Atomic rename
    if err := os.Rename(tmpPath, configPath); err != nil {
        os.Remove(tmpPath)  // Cleanup temp file
        return fmt.Errorf("failed to rename config: %w", err)
    }

    return nil
}

// UpdateLastSyncTime updates only the last_sync_time field
// Uses read-modify-write pattern for atomicity
func UpdateLastSyncTime(configPath string, syncTime time.Time) error {
    // Load existing config
    config, err := LoadConfig(configPath)
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Update timestamp
    config.LastSyncTime = &syncTime

    // Save updated config
    if err := SaveConfig(configPath, config); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    return nil
}

// validateTimestamp checks if timestamp is reasonable
func validateTimestamp(ts time.Time) error {
    // Sanity check: must be between 2020 and 2100
    year := ts.Year()
    if year < 2020 || year > 2100 {
        return fmt.Errorf("timestamp year out of range: %d", year)
    }

    // Check if timestamp is too far in future (>1 hour)
    now := time.Now()
    if ts.After(now.Add(1 * time.Hour)) {
        return fmt.Errorf("timestamp is too far in future: %v", ts)
    }

    return nil
}
```

### 2. Sync Options Extensions

#### SyncOptions Structure

**File**: `internal/sync/types.go`

```go
type SyncOptions struct {
    FolderPath    string
    DryRun        bool
    Strategy      ConflictStrategy
    CreateMissing bool
    JSONOutput    bool

    // NEW FIELDS (E06-F04)
    Incremental     bool       // Enable incremental sync mode
    ForceFull       bool       // Force full scan (ignore last_sync_time)
    ConfigPath      string     // Path to .sharkconfig.json
    LastSyncTime    *time.Time // Loaded from config (internal)
    CurrentSyncTime time.Time  // Timestamp at sync start (internal)
}

// Validate checks option validity and compatibility
func (o *SyncOptions) Validate() error {
    // Validate folder path
    if o.FolderPath == "" {
        return fmt.Errorf("folder path is required")
    }

    // Validate strategy
    if !o.Strategy.IsValid() {
        return fmt.Errorf("invalid conflict strategy: %s", o.Strategy)
    }

    // Validate flag combinations
    if o.ForceFull && !o.Incremental {
        return fmt.Errorf("--force-full-scan requires --incremental")
    }

    return nil
}
```

### 3. Conflict Structure Extensions

#### Enhanced Conflict Type

**File**: `internal/sync/conflict.go`

```go
type Conflict struct {
    TaskKey       string
    Field         string
    FileValue     string
    DatabaseValue string

    // NEW FIELDS (E06-F04)
    FileMtime   time.Time  // File modification time
    DBUpdatedAt time.Time  // Database updated_at timestamp
}

// FormatConflict creates human-readable conflict description
func (c *Conflict) FormatConflict() string {
    return fmt.Sprintf(
        "Conflict in %s:\n"+
        "  Field: %s\n"+
        "  Database: \"%s\" (updated %s)\n"+
        "  File: \"%s\" (modified %s)\n",
        c.TaskKey,
        c.Field,
        c.DatabaseValue, c.DBUpdatedAt.Format(time.RFC3339),
        c.FileValue, c.FileMtime.Format(time.RFC3339),
    )
}
```

### 4. Sync Report Extensions

#### Enhanced SyncReport

**File**: `internal/sync/types.go`

```go
type SyncReport struct {
    // Existing fields
    FilesScanned      int
    TasksImported     int
    TasksUpdated      int
    ConflictsDetected int
    ConflictsResolved int
    Warnings          []string
    Errors            []string
    Conflicts         []Conflict

    // NEW FIELDS (E06-F04)
    FilesChanged      int     // Files with mtime > last_sync_time
    FilesSkipped      int     // Files skipped due to mtime filter
    ElapsedSeconds    float64 // Total sync duration
    IncrementalMode   bool    // Was incremental mode used?
    LastSyncTime      *time.Time // Previous sync timestamp
    CurrentSyncTime   time.Time  // This sync timestamp
}

// FormatReport creates human-readable sync report
func (r *SyncReport) FormatReport() string {
    var sb strings.Builder

    if r.IncrementalMode {
        sb.WriteString("Incremental sync completed in %.1f seconds:\n", r.ElapsedSeconds)
    } else {
        sb.WriteString("Full sync completed in %.1f seconds:\n", r.ElapsedSeconds)
    }

    sb.WriteString("  Files scanned: %d\n", r.FilesScanned)

    if r.IncrementalMode {
        sb.WriteString("  Files changed: %d (%d created, %d updated)\n",
            r.FilesChanged, r.TasksImported, r.TasksUpdated)
        sb.WriteString("  Files skipped: %d\n", r.FilesSkipped)
    } else {
        sb.WriteString("  Files processed: %d (%d created, %d updated)\n",
            r.FilesChanged, r.TasksImported, r.TasksUpdated)
    }

    if r.ConflictsDetected > 0 {
        sb.WriteString("  Conflicts detected: %d\n", r.ConflictsDetected)
        sb.WriteString("  Conflicts resolved: %d\n", r.ConflictsResolved)
    }

    if len(r.Warnings) > 0 {
        sb.WriteString("  Warnings: %d\n", len(r.Warnings))
    }

    return sb.String()
}
```

---

## Core Algorithms

### 1. Incremental Sync Engine

#### Main Sync Flow

**File**: `internal/sync/engine.go`

```go
// Sync performs incremental or full synchronization
func (e *SyncEngine) Sync(ctx context.Context, opts SyncOptions) (*SyncReport, error) {
    startTime := time.Now()

    // Initialize report
    report := &SyncReport{
        IncrementalMode: opts.Incremental,
        CurrentSyncTime: opts.CurrentSyncTime,
        LastSyncTime:    opts.LastSyncTime,
        Warnings:        []string{},
        Errors:          []string{},
        Conflicts:       []Conflict{},
    }

    // Step 1: Scan all files in folder
    allFiles, err := e.scanner.Scan(opts.FolderPath)
    if err != nil {
        return nil, fmt.Errorf("failed to scan files: %w", err)
    }
    report.FilesScanned = len(allFiles)

    // Step 2: Filter files based on incremental mode
    filesToProcess := allFiles
    if opts.Incremental && !opts.ForceFull {
        filesToProcess = e.filterChangedFiles(allFiles, opts.LastSyncTime, report)
    }
    report.FilesChanged = len(filesToProcess)
    report.FilesSkipped = report.FilesScanned - report.FilesChanged

    // If no files to process, return early
    if len(filesToProcess) == 0 {
        if opts.Incremental {
            report.Warnings = append(report.Warnings, "No files changed since last sync")
        }
        report.ElapsedSeconds = time.Since(startTime).Seconds()
        return report, nil
    }

    // Step 3: Parse files
    taskDataList, parseWarnings := e.parseFiles(filesToProcess)
    report.Warnings = append(report.Warnings, parseWarnings...)

    if len(taskDataList) == 0 {
        report.ElapsedSeconds = time.Since(startTime).Seconds()
        return report, nil
    }

    // Step 4: Query database for existing tasks
    taskKeys := extractTaskKeys(taskDataList)
    dbTasks, err := e.taskRepo.GetByKeys(ctx, taskKeys)
    if err != nil {
        return nil, fmt.Errorf("failed to query database: %w", err)
    }

    // Step 5: Begin transaction (unless dry-run)
    var tx *sql.Tx
    if !opts.DryRun {
        tx, err = e.db.BeginTx(ctx, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to begin transaction: %w", err)
        }
        defer tx.Rollback()  // Safety net
    }

    // Step 6: Process each task (create or update)
    for _, taskData := range taskDataList {
        if err := e.syncTask(ctx, tx, taskData, dbTasks, opts, report); err != nil {
            return report, err  // Transaction will rollback
        }
    }

    // Step 7: Commit transaction (unless dry-run)
    if !opts.DryRun {
        if err := tx.Commit(); err != nil {
            return nil, fmt.Errorf("failed to commit transaction: %w", err)
        }

        // Step 8: Update last_sync_time in config (only after successful commit)
        if opts.Incremental {
            if err := UpdateLastSyncTime(opts.ConfigPath, opts.CurrentSyncTime); err != nil {
                // Log error but don't fail sync (database is already committed)
                log.Printf("Warning: Failed to update last_sync_time in config: %v", err)
                report.Warnings = append(report.Warnings,
                    "Failed to update last_sync_time in config, next sync may reprocess files")
            }
        }
    }

    report.ElapsedSeconds = time.Since(startTime).Seconds()
    return report, nil
}
```

### 2. File Filtering Algorithm

#### Changed Files Detection

**File**: `internal/sync/engine.go`

```go
// filterChangedFiles returns only files modified after lastSyncTime
func (e *SyncEngine) filterChangedFiles(
    allFiles []FileInfo,
    lastSyncTime *time.Time,
    report *SyncReport,
) []FileInfo {
    // If no last sync time, return all files (full scan)
    if lastSyncTime == nil {
        report.Warnings = append(report.Warnings,
            "No last_sync_time found, performing full scan")
        return allFiles
    }

    changedFiles := []FileInfo{}
    now := time.Now()

    for _, file := range allFiles {
        // Convert to UTC for consistent comparison
        fileMtime := file.ModifiedAt.UTC()
        lastSync := lastSyncTime.UTC()

        // Check for clock skew
        if fileMtime.After(now.Add(60 * time.Second)) {
            // File is >60 seconds in future
            skew := fileMtime.Sub(now)
            report.Warnings = append(report.Warnings,
                fmt.Sprintf("File has future mtime (clock skew: %v): %s", skew, file.FilePath))
            // Continue processing (don't skip)
        }

        // Include file if modified after last sync
        if fileMtime.After(lastSync) {
            changedFiles = append(changedFiles, file)
        }
    }

    return changedFiles
}
```

### 3. Enhanced Conflict Detection

#### Timestamp-Based Conflict Detection

**File**: `internal/sync/conflict.go`

```go
// DetectConflicts checks for conflicts between file and database
// NEW (E06-F04): Only reports conflicts if BOTH file AND database modified since last sync
func (d *ConflictDetector) DetectConflicts(
    fileData *TaskMetadata,
    dbTask *models.Task,
    lastSyncTime *time.Time,
) []Conflict {
    conflicts := []Conflict{}

    // NEW: Timestamp-based conflict detection
    // If lastSyncTime is provided, check if both file and DB modified
    isTimestampConflict := false
    if lastSyncTime != nil {
        fileMtime := fileData.ModifiedAt.UTC()
        dbUpdated := dbTask.UpdatedAt.UTC()
        lastSync := lastSyncTime.UTC()

        fileModified := fileMtime.After(lastSync)
        dbModified := dbUpdated.After(lastSync)

        // Conflict exists only if BOTH modified since last sync
        if fileModified && dbModified {
            isTimestampConflict = true
        } else {
            // No conflict:
            // - If only file modified: file-wins (normal update)
            // - If only DB modified: file unchanged, skip update
            // - If neither modified: no changes, skip
            return conflicts  // Empty list
        }
    } else {
        // No last_sync_time → use E04-F07 behavior (always check fields)
        isTimestampConflict = true
    }

    // EXISTING: Field-level conflict detection
    // (Only executed if timestamp conflict exists)

    // 1. Title conflict
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

    // 2. Description conflict
    if fileData.Description != nil && dbTask.Description != nil {
        if *fileData.Description != *dbTask.Description {
            conflicts = append(conflicts, Conflict{
                TaskKey:       dbTask.Key,
                Field:         "description",
                FileValue:     *fileData.Description,
                DatabaseValue: *dbTask.Description,
                FileMtime:     fileData.ModifiedAt,
                DBUpdatedAt:   dbTask.UpdatedAt,
            })
        }
    }

    // 3. Status conflict (if present in file frontmatter)
    if fileData.Status != nil && dbTask.Status != "" {
        fileStatus := string(*fileData.Status)
        dbStatus := string(dbTask.Status)
        if fileStatus != dbStatus {
            conflicts = append(conflicts, Conflict{
                TaskKey:       dbTask.Key,
                Field:         "status",
                FileValue:     fileStatus,
                DatabaseValue: dbStatus,
                FileMtime:     fileData.ModifiedAt,
                DBUpdatedAt:   dbTask.UpdatedAt,
            })
        }
    }

    return conflicts
}
```

### 4. Repository Extensions

#### Get Tasks Updated Since Timestamp

**File**: `internal/repository/task_repository.go`

```go
// GetUpdatedSince returns tasks updated after the given timestamp
// Used for incremental sync conflict detection
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

    tasks := []*models.Task{}
    for rows.Next() {
        task := &models.Task{}
        err := rows.Scan(
            &task.ID, &task.FeatureID, &task.Key, &task.Title, &task.Description,
            &task.Status, &task.AgentType, &task.Priority, &task.DependsOn,
            &task.AssignedAgent, &task.FilePath, &task.BlockedReason,
            &task.CreatedAt, &task.StartedAt, &task.CompletedAt, &task.BlockedAt,
            &task.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan task: %w", err)
        }
        tasks = append(tasks, task)
    }

    return tasks, rows.Err()
}
```

---

## CLI Integration

### Sync Command Extensions

**File**: `internal/cli/commands/sync.go`

```go
var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Synchronize task files with database",
    Long: `Scans feature folders for task markdown files and syncs with database.

Examples:
  # Full sync (all files)
  shark sync

  # Incremental sync (only changed files)
  shark sync --incremental

  # Force full rescan (ignore last_sync_time)
  shark sync --incremental --force-full-scan

  # Dry-run incremental sync
  shark sync --incremental --dry-run
`,
    RunE: runSync,
}

func init() {
    // Existing flags
    syncCmd.Flags().String("folder", "docs/plan", "Folder to scan")
    syncCmd.Flags().Bool("dry-run", false, "Preview changes without modifying database")
    syncCmd.Flags().String("strategy", "file-wins", "Conflict resolution strategy (file-wins, db-wins)")
    syncCmd.Flags().Bool("json", false, "Output JSON report")

    // NEW FLAGS (E06-F04)
    syncCmd.Flags().Bool("incremental", false, "Enable incremental sync (only process changed files)")
    syncCmd.Flags().Bool("force-full-scan", false, "Force full scan even with --incremental")
}

func runSync(cmd *cobra.Command, args []string) error {
    // Parse flags
    folderPath, _ := cmd.Flags().GetString("folder")
    dryRun, _ := cmd.Flags().GetBool("dry-run")
    strategyStr, _ := cmd.Flags().GetString("strategy")
    jsonOutput, _ := cmd.Flags().GetBool("json")

    // NEW: Parse incremental flags
    incremental, _ := cmd.Flags().GetBool("incremental")
    forceFull, _ := cmd.Flags().GetBool("force-full-scan")

    // Validate strategy
    strategy, err := sync.ParseConflictStrategy(strategyStr)
    if err != nil {
        return fmt.Errorf("invalid strategy: %w", err)
    }

    // Load configuration
    configPath := ".sharkconfig.json"
    config, err := init.LoadConfig(configPath)
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Build sync options
    opts := sync.SyncOptions{
        FolderPath:      folderPath,
        DryRun:          dryRun,
        Strategy:        strategy,
        JSONOutput:      jsonOutput,
        Incremental:     incremental,
        ForceFull:       forceFull,
        ConfigPath:      configPath,
        LastSyncTime:    config.LastSyncTime,  // May be nil
        CurrentSyncTime: time.Now(),           // Capture at start
    }

    // Validate options
    if err := opts.Validate(); err != nil {
        return err
    }

    // Create sync engine
    dbPath := "shark-tasks.db"
    engine, err := sync.NewSyncEngine(dbPath)
    if err != nil {
        return fmt.Errorf("failed to create sync engine: %w", err)
    }
    defer engine.Close()

    // Run sync
    ctx := context.Background()
    report, err := engine.Sync(ctx, opts)
    if err != nil {
        return fmt.Errorf("sync failed: %w", err)
    }

    // Display report
    if jsonOutput {
        return outputJSONReport(report)
    }

    displaySyncReport(report)
    return nil
}

func displaySyncReport(report *sync.SyncReport) {
    fmt.Println(report.FormatReport())

    // Show warnings
    if len(report.Warnings) > 0 {
        fmt.Println("\nWarnings:")
        for _, warning := range report.Warnings {
            fmt.Printf("  - %s\n", warning)
        }
    }

    // Show conflicts
    if len(report.Conflicts) > 0 {
        fmt.Println("\nConflicts:")
        for _, conflict := range report.Conflicts {
            fmt.Print(conflict.FormatConflict())
        }
    }
}
```

---

## Implementation Checklist

### Phase 1: Configuration Management

- [ ] Add `LastSyncTime` field to `ConfigDefaults` struct
- [ ] Implement `LoadConfig()` with validation
- [ ] Implement `SaveConfig()` with atomic write
- [ ] Implement `UpdateLastSyncTime()` helper
- [ ] Add unit tests for config operations

### Phase 2: Sync Engine Extensions

- [ ] Add `Incremental` and `ForceFull` flags to `SyncOptions`
- [ ] Implement `filterChangedFiles()` algorithm
- [ ] Add clock skew detection logic
- [ ] Update `Sync()` to call filtering conditionally
- [ ] Update `Sync()` to save `last_sync_time` after commit

### Phase 3: Conflict Detection Enhancement

- [ ] Add `FileMtime` and `DBUpdatedAt` to `Conflict` struct
- [ ] Implement timestamp-based conflict detection
- [ ] Update `DetectConflicts()` with timestamp logic
- [ ] Add `FormatConflict()` method with timestamps

### Phase 4: Repository Extensions

- [ ] Implement `GetUpdatedSince()` method in `TaskRepository`
- [ ] Add unit tests for timestamp queries
- [ ] Benchmark query performance

### Phase 5: CLI Integration

- [ ] Add `--incremental` flag to sync command
- [ ] Add `--force-full-scan` flag to sync command
- [ ] Update help text with examples
- [ ] Update sync report display

### Phase 6: Testing

- [ ] Unit tests: Config load/save with timestamps
- [ ] Unit tests: File filtering algorithm
- [ ] Unit tests: Timestamp conflict detection
- [ ] Integration test: First sync (no last_sync_time)
- [ ] Integration test: Incremental sync (some files changed)
- [ ] Integration test: Conflict resolution with timestamps
- [ ] Integration test: Force full scan
- [ ] Performance test: 500 files, 5 changed (<2s)

---

## Performance Benchmarks

### Target Performance

```go
// File: internal/sync/engine_benchmark_test.go

func BenchmarkIncrementalSync5Files(b *testing.B) {
    // Setup: 287 total files, 5 changed
    db := setupTestDB(b)
    defer db.Close()

    createTestFiles(b, 287)
    touchFiles(b, 5)  // Update mtime on 5 files

    lastSync := time.Now().Add(-1 * time.Hour)
    opts := sync.SyncOptions{
        Incremental:  true,
        LastSyncTime: &lastSync,
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        engine := sync.NewSyncEngine(db)
        report, err := engine.Sync(context.Background(), opts)
        require.NoError(b, err)
        require.Equal(b, 5, report.FilesChanged)
    }
}

// Target: <2 seconds for 5 files
// Baseline (E04-F07): ~8 seconds for all 287 files
```

---

## Error Handling Patterns

### Error Categories

**1. Configuration Errors** (non-fatal):
```go
// Invalid timestamp → log warning, use full scan
if err := validateTimestamp(ts); err != nil {
    log.Printf("Warning: Invalid last_sync_time: %v, using full scan", err)
    lastSyncTime = nil
}
```

**2. Clock Skew** (warning, continue):
```go
// Future mtime → log warning, process file
if fileMtime.After(now.Add(60 * time.Second)) {
    log.Printf("Warning: File has future mtime: %s", filePath)
    // Continue processing
}
```

**3. Config Write Failure** (warning, don't fail sync):
```go
// Failed to update config → log error, but sync succeeded
if err := UpdateLastSyncTime(config, syncTime); err != nil {
    log.Printf("Error: Failed to update last_sync_time: %v", err)
    log.Printf("Note: Next sync may reprocess files")
    // Don't return error (database already committed)
}
```

**4. Database Errors** (fatal, rollback):
```go
// Transaction error → rollback, return error
if err := tx.Commit(); err != nil {
    // tx.Rollback() already deferred
    return nil, fmt.Errorf("failed to commit transaction: %w", err)
}
```

---

## Summary

### Key Implementation Points

1. **Config Schema**: Add `last_sync_time` field (nullable time.Time)
2. **Filtering Logic**: Compare `file.mtime` with `last_sync_time`
3. **Conflict Detection**: Only report conflicts if both file AND database modified
4. **Update Order**: Database commit THEN config update (data integrity)
5. **Backward Compat**: Null `last_sync_time` → automatic full scan

### Code Reuse

- **Existing**: Scanner, parser, conflict resolver, transaction management
- **Extended**: Sync engine, conflict detector, config management
- **New**: Timestamp utilities, filtering logic, config update

### Performance Strategy

- **Fast Path**: Mtime filtering is O(n) with ~1ns per comparison
- **Reduced I/O**: Only parse changed files (5 vs. 287)
- **Same Transaction Model**: No performance regression for full scans

---

**Document Complete**: 2025-12-17
**Ready for Implementation**: Yes (POC-level detail)
