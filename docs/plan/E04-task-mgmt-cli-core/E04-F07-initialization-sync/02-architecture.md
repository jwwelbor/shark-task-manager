# Architecture: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-16
**Author**: backend-architect

## Purpose

This document defines the system architecture for initialization and synchronization features. It describes component interactions, data flows, and architectural patterns for `pm init` and `pm sync` commands.

---

## System Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLI Layer (Cobra)                               │
│                                                                               │
│  ┌──────────────────────┐                  ┌─────────────────────────────┐  │
│  │   pm init            │                  │   pm sync                    │  │
│  │   (init.go)          │                  │   (sync.go)                  │  │
│  │                      │                  │                              │  │
│  │  Flags:              │                  │  Flags:                      │  │
│  │   --non-interactive  │                  │   --folder                   │  │
│  │   --force            │                  │   --dry-run                  │  │
│  │   --db               │                  │   --strategy                 │  │
│  │   --config           │                  │   --create-missing           │  │
│  └──────────┬───────────┘                  └──────────┬──────────────────┘  │
│             │                                          │                      │
└─────────────┼──────────────────────────────────────────┼──────────────────────┘
              │                                          │
              ▼                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Application Layer (Business Logic)                   │
│                                                                               │
│  ┌──────────────────────────────────────────┐                                │
│  │  Initializer (internal/init/)             │                                │
│  │                                           │                                │
│  │  + Initialize(opts InitOptions)          │                                │
│  │    - Create database                     │                                │
│  │    - Create folders                      │                                │
│  │    - Create config                       │                                │
│  │    - Copy templates                      │                                │
│  └──────────────────────────────────────────┘                                │
│                                                                               │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │  SyncEngine (internal/sync/engine.go)                                   │  │
│  │                                                                          │  │
│  │  + Sync(opts SyncOptions) SyncReport                                    │  │
│  │    1. Scan files (FileScanner)                                          │  │
│  │    2. Parse frontmatter (TaskFileParser)                                │  │
│  │    3. Query database (Repositories)                                     │  │
│  │    4. Detect conflicts (ConflictDetector)                               │  │
│  │    5. Resolve conflicts (ConflictResolver)                              │  │
│  │    6. Update database (Transaction)                                     │  │
│  │    7. Generate report                                                   │  │
│  └──────────────────────────────────────────┬───────────────────────────────┘  │
│                                              │                                │
│  ┌──────────────────────┐  ┌───────────────┴───────────┐  ┌──────────────┐  │
│  │  FileScanner         │  │  ConflictDetector          │  │  Conflict    │  │
│  │  (scanner.go)        │  │  (conflict.go)             │  │  Resolver    │  │
│  │                      │  │                            │  │  (resolver.go│  │
│  │  + Scan(root) []file│  │  + Detect(file, db)        │  │              │  │
│  │  + InferEpicFeature  │  │    []Conflict              │  │  + Resolve() │  │
│  └──────────────────────┘  └────────────────────────────┘  └──────────────┘  │
└───────────────────────────────────────────┬─────────────────────────────────┘
                                            │
                                            ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Data Access Layer (Repository Pattern)                    │
│                                                                               │
│  ┌──────────────────────┐  ┌──────────────────────┐  ┌───────────────────┐  │
│  │  TaskRepository      │  │  EpicRepository      │  │  FeatureRepository│  │
│  │  (extended)          │  │  (extended)          │  │  (extended)       │  │
│  │                      │  │                      │  │                   │  │
│  │  + BulkCreate()      │  │  + CreateIfNotExists()│  │  + CreateIfNotExists()│  │
│  │  + GetByKeys()       │  │  + GetByKey()        │  │  + GetByKey()     │  │
│  │  + UpdateMetadata()  │  │                      │  │                   │  │
│  └──────────────────────┘  └──────────────────────┘  └───────────────────┘  │
│                                                                               │
└───────────────────────────────────────────┬─────────────────────────────────┘
                                            │
                                            ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Database Layer (SQLite)                             │
│                                                                               │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐  ┌───────────┐  │
│  │  epics         │  │  features      │  │  tasks         │  │  task_    │  │
│  │                │  │                │  │                │  │  history  │  │
│  └────────────────┘  └────────────────┘  └────────────────┘  └───────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         Filesystem Layer (Task Files)                         │
│                                                                               │
│  ┌──────────────────────┐  ┌──────────────────────┐                          │
│  │  TaskFileParser      │  │  TaskFileWriter      │                          │
│  │  (parser.go)         │  │  (writer.go)         │                          │
│  │                      │  │                      │                          │
│  │  + ParseTaskFile()   │  │  + WriteTaskFile()   │                          │
│  └──────────────────────┘  └──────────────────────┘                          │
│                                                                               │
│  Filesystem Structure:                                                        │
│    docs/plan/E04-epic/E04-F07-feature/T-E04-F07-001.md                       │
│    docs/tasks/todo/*.md (legacy)                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Component Architecture

### 1. CLI Layer Components

#### 1.1 Init Command (`internal/cli/commands/init.go`)

**Responsibilities**:
- Parse command-line flags
- Validate user input
- Create Initializer instance
- Invoke initialization
- Display results (human or JSON)

**Interface**:
```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize Shark CLI infrastructure",
    Long:  `Creates database schema, folder structure, config file, and templates.`,
    RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
    // 1. Parse flags
    opts := parseInitOptions(cmd)

    // 2. Create initializer
    initializer := init.NewInitializer()

    // 3. Run initialization
    ctx := context.Background()
    result, err := initializer.Initialize(ctx, opts)
    if err != nil {
        return err
    }

    // 4. Display results
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }

    displayInitSuccess(result)
    return nil
}
```

**Dependencies**:
- `internal/init.Initializer` - Core initialization logic
- `internal/cli` - CLI utilities (OutputJSON, Success, Error)

#### 1.2 Sync Command (`internal/cli/commands/sync.go`)

**Responsibilities**:
- Parse command-line flags (folder, dry-run, strategy, etc.)
- Validate conflict resolution strategy
- Create SyncEngine instance
- Invoke synchronization
- Display sync report (human or JSON)

**Interface**:
```go
var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Synchronize task files with database",
    Long:  `Scans feature folders for task markdown files and syncs with database.`,
    RunE:  runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
    // 1. Parse flags
    opts := parseSyncOptions(cmd)

    // 2. Validate options
    if err := opts.Validate(); err != nil {
        return err
    }

    // 3. Create sync engine
    engine, err := sync.NewSyncEngine(opts.DBPath)
    if err != nil {
        return err
    }
    defer engine.Close()

    // 4. Run sync
    ctx := context.Background()
    report, err := engine.Sync(ctx, opts)
    if err != nil {
        return err
    }

    // 5. Display report
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(report)
    }

    displaySyncReport(report)
    return nil
}
```

**Dependencies**:
- `internal/sync.SyncEngine` - Core sync logic
- `internal/cli` - CLI utilities

---

### 2. Application Layer Components

#### 2.1 Initializer (`internal/init/initializer.go`)

**Responsibilities**:
- Orchestrate initialization steps
- Create database schema
- Create folder structure
- Create config file
- Copy task templates
- Handle idempotency (skip existing resources)

**Architecture**:
```go
type Initializer struct {
    // No persistent state needed
}

func NewInitializer() *Initializer {
    return &Initializer{}
}

func (init *Initializer) Initialize(ctx context.Context, opts InitOptions) (*InitResult, error) {
    result := &InitResult{}

    // Step 1: Create database
    if err := init.createDatabase(opts.DBPath); err != nil {
        return nil, &InitError{Step: "database", Err: err}
    }
    result.DatabaseCreated = true
    result.DatabasePath = opts.DBPath

    // Step 2: Create folders
    folders, err := init.createFolders()
    if err != nil {
        return nil, &InitError{Step: "folders", Err: err}
    }
    result.FoldersCreated = folders

    // Step 3: Create config
    if err := init.createConfig(opts); err != nil {
        return nil, &InitError{Step: "config", Err: err}
    }
    result.ConfigCreated = true

    // Step 4: Copy templates
    count, err := init.copyTemplates()
    if err != nil {
        return nil, &InitError{Step: "templates", Err: err}
    }
    result.TemplatesCopied = count

    return result, nil
}
```

**Sub-Components**:
- `createDatabase()` - Delegates to `db.InitDB()`
- `createFolders()` - Creates `docs/plan`, `templates` directories
- `createConfig()` - Writes `.pmconfig.json` with defaults
- `copyTemplates()` - Copies embedded templates to `templates/` folder

**Error Handling**:
- Each step wraps errors with context
- Returns InitError with step name for clarity
- Idempotency: Check existence before creating

#### 2.2 SyncEngine (`internal/sync/engine.go`)

**Responsibilities**:
- Orchestrate sync process
- Manage database transaction
- Coordinate scanner, parser, conflict detector, resolver
- Generate sync report
- Handle dry-run mode

**Architecture**:
```go
type SyncEngine struct {
    db            *sql.DB
    taskRepo      *repository.TaskRepository
    epicRepo      *repository.EpicRepository
    featureRepo   *repository.FeatureRepository
    scanner       *FileScanner
    detector      *ConflictDetector
    resolver      *ConflictResolver
}

func NewSyncEngine(dbPath string) (*SyncEngine, error) {
    // Open database
    db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
    if err != nil {
        return nil, err
    }

    // Create repositories
    repoDb := repository.NewDB(db)
    taskRepo := repository.NewTaskRepository(repoDb)
    epicRepo := repository.NewEpicRepository(repoDb)
    featureRepo := repository.NewFeatureRepository(repoDb)

    // Create sync components
    scanner := NewFileScanner()
    detector := NewConflictDetector()
    resolver := NewConflictResolver()

    return &SyncEngine{
        db:          db,
        taskRepo:    taskRepo,
        epicRepo:    epicRepo,
        featureRepo: featureRepo,
        scanner:     scanner,
        detector:    detector,
        resolver:    resolver,
    }, nil
}

func (e *SyncEngine) Sync(ctx context.Context, opts SyncOptions) (*SyncReport, error) {
    report := &SyncReport{}

    // Step 1: Scan files
    files, err := e.scanFiles(opts.FolderPath)
    if err != nil {
        return nil, err
    }
    report.FilesScanned = len(files)

    // Step 2: Parse files and collect task data
    tasks, err := e.parseFiles(files)
    if err != nil {
        return nil, err
    }

    // Step 3: Begin transaction (unless dry-run)
    var tx *sql.Tx
    if !opts.DryRun {
        tx, err = e.db.BeginTx(ctx, nil)
        if err != nil {
            return nil, err
        }
        defer tx.Rollback()
    }

    // Step 4: Sync each task
    for _, taskData := range tasks {
        if err := e.syncTask(ctx, tx, taskData, opts, report); err != nil {
            return report, err  // Transaction will rollback
        }
    }

    // Step 5: Commit transaction (unless dry-run)
    if !opts.DryRun {
        if err := tx.Commit(); err != nil {
            return nil, err
        }
    }

    return report, nil
}
```

**Key Design Decisions**:
1. **Single Transaction**: All database operations in one transaction
2. **Early Transaction**: Begin transaction early, defer rollback
3. **Dry-Run Support**: Skip transaction begin/commit in dry-run mode
4. **Error Propagation**: Any error triggers rollback
5. **Report Accumulation**: Build report as sync progresses

#### 2.3 FileScanner (`internal/sync/scanner.go`)

**Responsibilities**:
- Recursively scan directories for task markdown files
- Filter files matching pattern `T-*.md`
- Infer epic and feature keys from directory structure
- Return file metadata for each task file

**Architecture**:
```go
type FileScanner struct {
    // No state needed
}

func NewFileScanner() *FileScanner {
    return &FileScanner{}
}

func (s *FileScanner) Scan(rootPath string) ([]TaskFileInfo, error) {
    var files []TaskFileInfo

    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Skip directories
        if info.IsDir() {
            return nil
        }

        // Check if filename matches task pattern
        if !s.isTaskFile(info.Name()) {
            return nil
        }

        // Infer epic and feature from path
        epicKey, featureKey, err := s.inferEpicFeature(path)
        if err != nil {
            // Log warning, continue
            log.Warnf("Could not infer epic/feature for %s: %v", path, err)
        }

        // Add to results
        files = append(files, TaskFileInfo{
            FilePath:   path,
            FileName:   info.Name(),
            EpicKey:    epicKey,
            FeatureKey: featureKey,
            ModifiedAt: info.ModTime(),
        })

        return nil
    })

    return files, err
}

func (s *FileScanner) isTaskFile(filename string) bool {
    // Match pattern: T-E##-F##-###.md
    return regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}\.md$`).MatchString(filename)
}

func (s *FileScanner) inferEpicFeature(filePath string) (epicKey, featureKey string, err error) {
    // Parse directory structure
    // Example: docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-001.md
    //          parent: E04-F07-initialization-sync → feature key
    //          grandparent: E04-task-mgmt-cli-core → epic key

    dir := filepath.Dir(filePath)
    featureDir := filepath.Base(dir)

    // Extract feature key from directory name
    featureKey = extractKeyFromDirName(featureDir)  // E04-F07

    // Extract epic key from feature key or grandparent directory
    epicKey = extractEpicFromFeatureKey(featureKey)  // E04

    return epicKey, featureKey, nil
}
```

**Inference Strategy**:
1. Try to extract feature key from parent directory name (pattern: `E##-F##-*`)
2. Extract epic key from feature key (pattern: `E##`)
3. If inference fails, try parsing task key from filename
4. If all fails, return empty keys (sync will require manual specification)

#### 2.4 ConflictDetector (`internal/sync/conflict.go`)

**Responsibilities**:
- Compare file metadata with database record
- Detect conflicts in: title, description, file_path
- Return list of conflicts with details

**Architecture**:
```go
type ConflictDetector struct {
    // No state needed
}

func NewConflictDetector() *ConflictDetector {
    return &ConflictDetector{}
}

func (d *ConflictDetector) DetectConflicts(fileData *TaskMetadata, dbTask *models.Task) []Conflict {
    conflicts := []Conflict{}

    // 1. Title conflict
    if fileData.Title != "" && fileData.Title != dbTask.Title {
        conflicts = append(conflicts, Conflict{
            TaskKey:       dbTask.Key,
            Field:         "title",
            FileValue:     fileData.Title,
            DatabaseValue: dbTask.Title,
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
            })
        }
    }

    // 3. File path conflict (always update to actual location)
    actualPath := fileData.FilePath
    if dbTask.FilePath == nil || *dbTask.FilePath != actualPath {
        dbValue := ""
        if dbTask.FilePath != nil {
            dbValue = *dbTask.FilePath
        }
        conflicts = append(conflicts, Conflict{
            TaskKey:       dbTask.Key,
            Field:         "file_path",
            FileValue:     actualPath,
            DatabaseValue: dbValue,
        })
    }

    return conflicts
}
```

**Conflict Detection Rules**:
- **Title**: Conflict if file has title AND differs from database
- **Description**: Conflict if both exist AND differ
- **File Path**: Always conflict if database path != actual path (file moved)
- **No conflict for**: status, priority, agent_type (database-only fields)

#### 2.5 ConflictResolver (`internal/sync/resolver.go`)

**Responsibilities**:
- Apply conflict resolution strategy
- Return resolved task model ready for database update
- Preserve database-only fields

**Architecture**:
```go
type ConflictResolver struct {
    // No state needed
}

func NewConflictResolver() *ConflictResolver {
    return &ConflictResolver{}
}

func (r *ConflictResolver) Resolve(conflicts []Conflict, fileData *TaskMetadata, dbTask *models.Task, strategy ConflictStrategy) (*models.Task, error) {
    // Create copy of database task (preserve all fields)
    resolved := *dbTask

    for _, conflict := range conflicts {
        switch strategy {
        case ConflictStrategyFileWins:
            r.applyFileValue(&resolved, conflict)
        case ConflictStrategyDatabaseWins:
            // Keep database value (no change)
        case ConflictStrategyNewerWins:
            r.applyNewerValue(&resolved, conflict, fileData, dbTask)
        }
    }

    // Always update file_path to actual location (regardless of strategy)
    actualPath := fileData.FilePath
    resolved.FilePath = &actualPath

    return &resolved, nil
}

func (r *ConflictResolver) applyFileValue(task *models.Task, conflict Conflict) {
    switch conflict.Field {
    case "title":
        task.Title = conflict.FileValue
    case "description":
        desc := conflict.FileValue
        task.Description = &desc
    case "file_path":
        task.FilePath = &conflict.FileValue
    }
}

func (r *ConflictResolver) applyNewerValue(task *models.Task, conflict Conflict, fileData *TaskMetadata, dbTask *models.Task) {
    // Compare timestamps
    if fileData.ModifiedAt.After(dbTask.UpdatedAt) {
        // File is newer → use file value
        r.applyFileValue(task, conflict)
    }
    // Otherwise, keep database value (no change)
}
```

**Strategy Implementation**:
- **file-wins**: Always use file value
- **database-wins**: Keep database value (no update)
- **newer-wins**: Compare timestamps, use newer source

---

### 3. Data Access Layer Extensions

#### 3.1 TaskRepository Extensions

**New Methods**:

**BulkCreate**:
```go
func (r *TaskRepository) BulkCreate(ctx context.Context, tasks []*models.Task) (int, error) {
    if len(tasks) == 0 {
        return 0, nil
    }

    // Prepare statement
    query := `
        INSERT INTO tasks (feature_id, key, title, description, status, agent_type,
                          priority, depends_on, assigned_agent, file_path, blocked_reason)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

    stmt, err := r.db.PrepareContext(ctx, query)
    if err != nil {
        return 0, fmt.Errorf("failed to prepare statement: %w", err)
    }
    defer stmt.Close()

    // Execute for each task
    count := 0
    for _, task := range tasks {
        if err := task.Validate(); err != nil {
            return count, fmt.Errorf("validation failed for task %s: %w", task.Key, err)
        }

        result, err := stmt.ExecContext(ctx,
            task.FeatureID, task.Key, task.Title, task.Description, task.Status,
            task.AgentType, task.Priority, task.DependsOn, task.AssignedAgent,
            task.FilePath, task.BlockedReason,
        )
        if err != nil {
            return count, fmt.Errorf("failed to insert task %s: %w", task.Key, err)
        }

        id, _ := result.LastInsertId()
        task.ID = id
        count++
    }

    return count, nil
}
```

**GetByKeys**:
```go
func (r *TaskRepository) GetByKeys(ctx context.Context, keys []string) (map[string]*models.Task, error) {
    if len(keys) == 0 {
        return map[string]*models.Task{}, nil
    }

    // Build IN clause
    placeholders := strings.Repeat("?,", len(keys))
    placeholders = placeholders[:len(placeholders)-1]  // Remove trailing comma

    query := fmt.Sprintf(`
        SELECT id, feature_id, key, title, description, status, agent_type, priority,
               depends_on, assigned_agent, file_path, blocked_reason,
               created_at, started_at, completed_at, blocked_at, updated_at
        FROM tasks
        WHERE key IN (%s)
    `, placeholders)

    // Convert keys to []interface{} for query
    args := make([]interface{}, len(keys))
    for i, key := range keys {
        args[i] = key
    }

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query tasks: %w", err)
    }
    defer rows.Close()

    // Build map
    result := make(map[string]*models.Task)
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
        result[task.Key] = task
    }

    return result, rows.Err()
}
```

**UpdateMetadata**:
```go
func (r *TaskRepository) UpdateMetadata(ctx context.Context, task *models.Task) error {
    // Update only metadata fields (not status, priority, agent_type)
    query := `
        UPDATE tasks
        SET title = ?, description = ?, file_path = ?
        WHERE id = ?
    `

    result, err := r.db.ExecContext(ctx, query,
        task.Title, task.Description, task.FilePath, task.ID,
    )
    if err != nil {
        return fmt.Errorf("failed to update task metadata: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("task not found with id %d", task.ID)
    }

    return nil
}
```

#### 3.2 EpicRepository Extensions

**CreateIfNotExists**:
```go
func (r *EpicRepository) CreateIfNotExists(ctx context.Context, epic *models.Epic) (*models.Epic, bool, error) {
    // Check if exists
    existing, err := r.GetByKey(ctx, epic.Key)
    if err == nil {
        // Already exists
        return existing, false, nil
    }

    // Create new epic
    if err := r.Create(ctx, epic); err != nil {
        return nil, false, fmt.Errorf("failed to create epic: %w", err)
    }

    return epic, true, nil
}

func (r *EpicRepository) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
    query := `
        SELECT id, key, title, description, status, priority, business_value,
               created_at, updated_at
        FROM epics
        WHERE key = ?
    `

    epic := &models.Epic{}
    err := r.db.QueryRowContext(ctx, query, key).Scan(
        &epic.ID, &epic.Key, &epic.Title, &epic.Description, &epic.Status,
        &epic.Priority, &epic.BusinessValue, &epic.CreatedAt, &epic.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("epic not found with key %s", key)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get epic: %w", err)
    }

    return epic, nil
}
```

#### 3.3 FeatureRepository Extensions

Similar to Epic repository, add `CreateIfNotExists` and `GetByKey` methods.

---

## Data Flow Diagrams

### Init Command Flow

```
User
 │
 ├─> pm init --non-interactive
 │
 ▼
InitCommand (CLI)
 ├─> Parse flags
 ├─> Create InitOptions
 │
 ▼
Initializer
 ├─> Step 1: Create Database
 │   ├─> db.InitDB(dbPath)
 │   ├─> Create schema (tables, indexes, triggers)
 │   └─> Set file permissions (600 on Unix)
 │
 ├─> Step 2: Create Folders
 │   ├─> os.MkdirAll("docs/plan", 0755)
 │   ├─> os.MkdirAll("templates", 0755)
 │   └─> Skip if already exists (idempotent)
 │
 ├─> Step 3: Create Config
 │   ├─> Check if .pmconfig.json exists
 │   ├─> Prompt user (unless --force or --non-interactive)
 │   ├─> Write JSON to temp file
 │   └─> Rename to .pmconfig.json (atomic)
 │
 ├─> Step 4: Copy Templates
 │   ├─> Read embedded templates
 │   ├─> Write to templates/ folder
 │   └─> Return count of templates copied
 │
 └─> Return InitResult

InitCommand
 └─> Display success message with next steps
```

### Sync Command Flow

```
User
 │
 ├─> pm sync --strategy=file-wins --dry-run
 │
 ▼
SyncCommand (CLI)
 ├─> Parse flags
 ├─> Create SyncOptions
 ├─> Validate options
 │
 ▼
SyncEngine
 ├─> Phase 1: File Scanning
 │   ├─> FileScanner.Scan(rootPath)
 │   ├─> Recursively walk directories
 │   ├─> Filter T-*.md files
 │   ├─> Infer epic/feature from path
 │   └─> Return []TaskFileInfo
 │
 ├─> Phase 2: File Parsing
 │   ├─> For each file:
 │   │   ├─> TaskFileParser.ParseTaskFile(path)
 │   │   ├─> Extract frontmatter (key, title, description)
 │   │   ├─> Validate frontmatter
 │   │   └─> Build TaskMetadata
 │   └─> Return []TaskMetadata
 │
 ├─> Phase 3: Database Query
 │   ├─> Extract all task keys from files
 │   ├─> TaskRepository.GetByKeys(keys)
 │   ├─> Single query with IN clause
 │   └─> Return map[key]*Task
 │
 ├─> Phase 4: Begin Transaction (unless --dry-run)
 │   ├─> db.BeginTx(ctx, nil)
 │   └─> defer tx.Rollback() (safety net)
 │
 ├─> Phase 5: Process Each Task
 │   │
 │   ├─> Case 1: New Task (not in database)
 │   │   ├─> Validate epic/feature exists
 │   │   │   ├─> EpicRepository.GetByKey(epicKey)
 │   │   │   ├─> FeatureRepository.GetByKey(featureKey)
 │   │   │   └─> If not exists:
 │   │   │       ├─> If --create-missing: Create epic/feature
 │   │   │       └─> Else: Log warning, skip task
 │   │   ├─> Create Task model (status=todo)
 │   │   ├─> TaskRepository.Create(task)
 │   │   ├─> Create history record (agent=sync, notes="Imported from file")
 │   │   └─> Increment report.TasksImported
 │   │
 │   ├─> Case 2: Existing Task (in database)
 │   │   ├─> ConflictDetector.DetectConflicts(fileData, dbTask)
 │   │   ├─> For each conflict:
 │   │   │   ├─> ConflictResolver.Resolve(conflicts, strategy)
 │   │   │   └─> Build resolved task model
 │   │   ├─> TaskRepository.UpdateMetadata(resolvedTask)
 │   │   ├─> Create history record (agent=sync, notes="Updated from file: title, file_path")
 │   │   ├─> Increment report.TasksUpdated
 │   │   └─> Append conflicts to report.Conflicts
 │   │
 │   └─> Handle Errors
 │       ├─> Invalid YAML: Log warning, skip file
 │       ├─> Missing key: Log warning, skip file
 │       ├─> Database error: Return error (triggers rollback)
 │       └─> Foreign key violation: Return error (triggers rollback)
 │
 ├─> Phase 6: Optional Cleanup (if --cleanup flag)
 │   ├─> Find orphaned tasks (file_path not in scanned files)
 │   ├─> TaskRepository.Delete(orphanedTasks)
 │   └─> Increment report.TasksDeleted
 │
 ├─> Phase 7: Commit Transaction (unless --dry-run)
 │   ├─> tx.Commit()
 │   └─> If error: Return error (rollback already deferred)
 │
 └─> Return SyncReport

SyncCommand
 └─> Display sync report (human or JSON)
```

---

## Transaction Management

### Transaction Boundaries

**Init Command**:
- **Database operations**: Use internal transaction (handled by db.InitDB)
- **Filesystem operations**: No transaction (idempotent by design)

**Sync Command**:
- **Single transaction** encompasses ALL database operations:
  - Epic creation (if --create-missing)
  - Feature creation (if --create-missing)
  - Task inserts (new tasks)
  - Task updates (existing tasks)
  - History record inserts
  - Task deletes (if --cleanup)

**Transaction Pattern**:
```go
// Begin transaction
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()  // Safety net (no-op if committed)

// ... perform all database operations ...

// Commit transaction
if err := tx.Commit(); err != nil {
    return err
}
```

**Rollback Triggers**:
- SQL error (constraint violation, syntax error)
- Foreign key violation (missing epic/feature)
- Unique key violation (duplicate task key)
- Context cancellation (timeout, Ctrl+C)
- Any `return err` before commit

**Commit Conditions**:
- All files processed successfully
- All database operations completed
- No errors encountered
- Not in dry-run mode

---

## Error Handling Strategy

### Error Classification

**1. Non-Fatal Errors** (log warning, skip, continue):
```go
// Invalid YAML
log.Warnf("Invalid frontmatter in %s, skipping", file)
report.Warnings = append(report.Warnings, fmt.Sprintf("Invalid frontmatter in %s", file))
continue  // Skip this file, process others

// Missing required field
log.Warnf("Task %s missing required field 'key', skipping", file)
report.Warnings = append(report.Warnings, fmt.Sprintf("Missing key in %s", file))
continue

// Key mismatch
log.Warnf("Key mismatch in %s: filename=%s, frontmatter=%s", file, expectedKey, actualKey)
report.Warnings = append(report.Warnings, fmt.Sprintf("Key mismatch in %s", file))
continue
```

**2. Fatal Errors** (rollback, halt, return error):
```go
// Database connection error
if err := db.Ping(); err != nil {
    return nil, fmt.Errorf("database connection failed: %w", err)
}

// Transaction begin error
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return nil, fmt.Errorf("failed to begin transaction: %w", err)
}

// Foreign key violation (missing epic/feature without --create-missing)
if err := repo.Create(task); err != nil {
    if isForeignKeyError(err) {
        return fmt.Errorf("task %s references non-existent feature %s (use --create-missing or create feature first)", task.Key, featureKey)
    }
    return fmt.Errorf("failed to create task: %w", err)
}

// Transaction commit error
if err := tx.Commit(); err != nil {
    return nil, fmt.Errorf("failed to commit transaction: %w", err)
}
```

### Error Wrapping

**Use `fmt.Errorf` with `%w` for error chains**:
```go
if err != nil {
    return fmt.Errorf("failed to sync task %s: %w", taskKey, err)
}
```

**Custom Error Types**:
```go
type SyncError struct {
    Operation string
    File      string
    TaskKey   string
    Err       error
}

func (e *SyncError) Error() string {
    return fmt.Sprintf("sync error in %s (%s): %v", e.File, e.Operation, e.Err)
}

func (e *SyncError) Unwrap() error {
    return e.Err
}
```

---

## Concurrency and Context Handling

### Context Propagation

**All database operations accept context**:
```go
func (e *SyncEngine) Sync(ctx context.Context, opts SyncOptions) (*SyncReport, error) {
    // Context passed to all operations
    tx, err := e.db.BeginTxContext(ctx, nil)

    _, err = e.taskRepo.Create(ctx, task)

    _, err = e.taskRepo.GetByKeys(ctx, keys)
}
```

**Timeout Management**:
```go
// CLI command sets timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

engine := sync.NewSyncEngine(dbPath)
report, err := engine.Sync(ctx, opts)
```

**Cancellation Handling**:
- User presses Ctrl+C → context cancelled → transaction rolled back
- Timeout expires → context cancelled → transaction rolled back

### Concurrency Considerations

**Sequential Processing** (no parallelism needed for MVP):
- Files are processed one by one in a single transaction
- Simplifies error handling and transaction management
- Performance is sufficient for PRD requirements (<10s for 100 files)

**Future Optimization** (if needed):
- Parallel file parsing (goroutines + channels)
- But keep single transaction for database operations

---

## Performance Optimization

### Query Optimization

**1. Bulk Lookups**:
```go
// GOOD: Single query with IN clause
tasks, err := repo.GetByKeys(ctx, []string{"T-E04-F07-001", "T-E04-F07-002", ...})

// BAD: N queries (avoid)
for _, key := range keys {
    task, err := repo.GetByKey(ctx, key)
}
```

**2. Prepared Statements for Bulk Inserts**:
```go
stmt, err := tx.PrepareContext(ctx, insertQuery)
defer stmt.Close()

for _, task := range tasks {
    _, err = stmt.ExecContext(ctx, ...)
}
```

**3. Single Transaction**:
- Reduces commit overhead
- WAL mode benefits (readers don't block)

### Filesystem Optimization

**1. Efficient File Walking**:
```go
// filepath.Walk is efficient for <10,000 files
err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
    // ...
})
```

**2. File Stat Caching**:
```go
// os.FileInfo already provides modified time
// No need for separate os.Stat call
modTime := info.ModTime()
```

### Memory Optimization

**Stream Processing** (for large file sets):
```go
// Process files one at a time (don't load all into memory)
for _, file := range files {
    metadata, err := parseTaskFile(file)
    if err != nil {
        continue
    }

    // Process immediately
    syncTask(ctx, tx, metadata, opts)
}
```

---

## Security Considerations

### File Path Validation

**Prevent Path Traversal**:
```go
func validateFilePath(path string) error {
    // Convert to absolute path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }

    // Define allowed roots
    allowedRoots := []string{
        "/abs/path/to/docs/plan",
        "/abs/path/to/docs/tasks",
    }

    // Check if path is within allowed roots
    for _, root := range allowedRoots {
        if strings.HasPrefix(absPath, root) {
            return nil
        }
    }

    return fmt.Errorf("file path outside allowed directories: %s", path)
}
```

### Database File Permissions

**Set Restrictive Permissions** (Unix):
```go
func setDatabasePermissions(dbPath string) error {
    if runtime.GOOS == "windows" {
        return nil  // Not applicable on Windows
    }

    // Set permissions to 600 (read/write owner only)
    return os.Chmod(dbPath, 0600)
}
```

### YAML Injection Prevention

**Read-Only Parsing** (no code execution risk):
- gopkg.in/yaml.v3 is safe for read-only parsing
- Validate struct fields after unmarshaling
- Reject unexpected fields (strict unmarshal)

---

## Testing Strategy

### Unit Tests

**1. Initializer Tests**:
- Test database creation (idempotent)
- Test folder creation (idempotent)
- Test config creation (prompt behavior)
- Test template copying

**2. SyncEngine Tests** (with mock repositories):
- Test file scanning logic
- Test conflict detection
- Test conflict resolution strategies
- Test dry-run mode (no database changes)

**3. FileScanner Tests**:
- Test recursive directory traversal
- Test pattern matching (T-*.md)
- Test epic/feature inference

**4. ConflictDetector Tests**:
- Test title conflict detection
- Test description conflict detection
- Test file_path conflict detection

**5. ConflictResolver Tests**:
- Test file-wins strategy
- Test database-wins strategy
- Test newer-wins strategy (timestamp comparison)

### Integration Tests

**1. Full Init Test**:
```go
func TestInitCommand(t *testing.T) {
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")

    initializer := init.NewInitializer()
    result, err := initializer.Initialize(context.Background(), InitOptions{
        DBPath:         dbPath,
        NonInteractive: true,
    })

    assert.NoError(t, err)
    assert.True(t, result.DatabaseCreated)
    assert.FileExists(t, dbPath)
    assert.FileExists(t, ".pmconfig.json")
}
```

**2. Full Sync Test**:
```go
func TestSyncCommand(t *testing.T) {
    // Setup: Create database and task files
    db := setupTestDB(t)
    defer db.Close()

    createTestTaskFiles(t, []string{"T-E04-F07-001.md", "T-E04-F07-002.md"})

    // Execute sync
    engine := sync.NewSyncEngine(db)
    report, err := engine.Sync(context.Background(), SyncOptions{
        FolderPath: "testdata",
        DryRun:     false,
        Strategy:   ConflictStrategyFileWins,
    })

    assert.NoError(t, err)
    assert.Equal(t, 2, report.FilesScanned)
    assert.Equal(t, 2, report.TasksImported)
}
```

**3. Conflict Resolution Test**:
```go
func TestSyncWithConflicts(t *testing.T) {
    // Setup: Task in database with title "Old Title"
    db := setupTestDB(t)
    createTaskInDB(t, db, "T-E04-F07-001", "Old Title")

    // Create file with title "New Title"
    createTestFile(t, "T-E04-F07-001.md", "New Title")

    // Execute sync with file-wins strategy
    engine := sync.NewSyncEngine(db)
    report, err := engine.Sync(context.Background(), SyncOptions{
        Strategy: ConflictStrategyFileWins,
    })

    assert.NoError(t, err)
    assert.Equal(t, 1, report.ConflictsResolved)

    // Verify database updated
    task := getTaskFromDB(t, db, "T-E04-F07-001")
    assert.Equal(t, "New Title", task.Title)
}
```

### Performance Tests

**1. Init Performance** (target: <5 seconds):
```go
func BenchmarkInit(b *testing.B) {
    for i := 0; i < b.N; i++ {
        tmpDir := b.TempDir()
        initializer := init.NewInitializer()
        _, err := initializer.Initialize(context.Background(), InitOptions{
            DBPath:         filepath.Join(tmpDir, "test.db"),
            NonInteractive: true,
        })
        assert.NoError(b, err)
    }
}
```

**2. Sync Performance** (target: 100 files in <10 seconds):
```go
func BenchmarkSync100Files(b *testing.B) {
    // Setup: Create 100 task files
    createTestTaskFiles(b, 100)

    for i := 0; i < b.N; i++ {
        engine := sync.NewSyncEngine(db)
        _, err := engine.Sync(context.Background(), SyncOptions{})
        assert.NoError(b, err)
    }
}
```

---

## Deployment Considerations

### Database Initialization

**First Run**:
```bash
# User runs init command
pm init

# Database created with schema
# Folders created: docs/plan/, templates/
# Config created: .pmconfig.json
```

**Subsequent Runs**:
```bash
# Init is idempotent (safe to re-run)
pm init

# Skips existing database (no error)
# Skips existing folders (no error)
# Prompts before overwriting config (unless --force)
```

### Migration Path

**From Legacy System** (with status-based folders):
```bash
# Sync legacy folders first
pm sync --folder=docs/tasks/todo
pm sync --folder=docs/tasks/active
pm sync --folder=docs/tasks/completed

# Then sync feature folders (new structure)
pm sync --folder=docs/plan
```

**After Git Pull** (new/modified tasks from collaborators):
```bash
# Sync to update database with file changes
pm sync
```

---

## Extensibility Points

### Future Enhancements

**1. Bidirectional Sync** (update files from database):
```go
// New flag: --update-files
// With database-wins strategy:
//   - Read task from database
//   - Update file frontmatter
//   - Atomic file write (temp + rename)
```

**2. Watch Mode** (continuous sync):
```go
// New command: pm sync --watch
// Use fsnotify to watch for file changes
// Auto-sync when files modified
```

**3. Sync History Table** (track past sync operations):
```sql
CREATE TABLE sync_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_started_at TIMESTAMP,
    sync_completed_at TIMESTAMP,
    files_scanned INTEGER,
    tasks_imported INTEGER,
    tasks_updated INTEGER,
    strategy TEXT
);
```

**4. Selective Field Sync** (sync only certain fields):
```go
// New flag: --fields=title,description
// Only sync specified fields
```

---

## Summary

### Component Responsibilities

| Component | Responsibility | Key Methods |
|-----------|---------------|-------------|
| InitCommand | CLI interface for init | runInit() |
| SyncCommand | CLI interface for sync | runSync() |
| Initializer | Orchestrate init steps | Initialize() |
| SyncEngine | Orchestrate sync process | Sync() |
| FileScanner | Discover task files | Scan(), inferEpicFeature() |
| ConflictDetector | Compare file vs database | DetectConflicts() |
| ConflictResolver | Apply resolution strategy | Resolve() |
| TaskRepository | Database operations | BulkCreate(), GetByKeys(), UpdateMetadata() |

### Data Flow Summary

**Init**: CLI → Initializer → DB/Filesystem
**Sync**: CLI → SyncEngine → Scanner → Parser → Detector → Resolver → Repository → DB

### Transaction Strategy

- **Init**: No transaction needed (idempotent operations)
- **Sync**: Single transaction for ALL database operations, rollback on any error

---

**Document Complete**: 2025-12-16
**Next Document**: 04-backend-design.md (backend-architect creates)
