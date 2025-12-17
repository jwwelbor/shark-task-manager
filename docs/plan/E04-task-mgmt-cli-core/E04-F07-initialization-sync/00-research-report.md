# Research Report: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-16
**Author**: project-research-agent

## Executive Summary

This feature implements initialization (`pm init`) and synchronization (`pm sync`) for a Go-based CLI task manager with SQLite database. The project has an established Go codebase with Cobra CLI framework, SQLite database schema, task file parsing, and repository patterns. This feature bridges filesystem-based task markdown files with the database, enabling safe project setup and bidirectional sync.

**Key Findings**:
- Existing database schema is production-ready (from E04-F01)
- Task file parser exists but needs extension for sync-specific needs
- Repository pattern is well-established for database operations
- YAML frontmatter parser (gopkg.in/yaml.v3) is already integrated
- Cobra command structure is in place for CLI commands
- Transaction support exists via database/sql package
- No existing sync metadata tracking - requires new table/fields

---

## Project Context

### Current State

**Existing Components Analyzed**:

1. **Database Layer** (`internal/db/db.go`):
   - SQLite initialization with schema creation
   - Foreign key enforcement enabled
   - WAL mode for concurrency
   - Tables: epics, features, tasks, task_history
   - Automatic updated_at triggers

2. **Repository Layer** (`internal/repository/`):
   - TaskRepository with CRUD operations
   - EpicRepository and FeatureRepository
   - Transaction support via BeginTxContext
   - Context-aware operations for cancellation
   - Atomic status updates with history tracking

3. **Task File Parser** (`internal/taskfile/parser.go`):
   - YAML frontmatter parsing
   - Fields: task_key, status, title, description, file_path, feature, etc.
   - Validation of required fields
   - Content parsing (markdown body)

4. **CLI Framework** (`internal/cli/root.go`):
   - Cobra-based command structure
   - Global config management (JSON output, no-color, verbose, db-path)
   - Viper for config file support (.pmconfig.json)
   - pterm for rich terminal output
   - GetDBPath() helper for database location

5. **File Path Utilities** (`internal/filepath/filepath.go`):
   - Existing patterns for file organization (inferred from project structure)

6. **Models** (`internal/models/`):
   - Task, Epic, Feature, TaskHistory structs
   - Validation methods
   - Status and AgentType enums

**Completed Features**:
- E04-F01: Database Schema (complete)
- E04-F02: CLI Framework (complete)
- E04-F05: Folder Management (complete)
- E04-F06: Task Creation (complete)

**Dependencies**:
- This feature depends on E04-F01 (database) and E04-F02 (CLI)
- This feature is a dependency for team-based workflows (Git sync)

### Business Requirements Summary

From the PRD, this feature must provide:

1. **Initialization**: `pm init` command to set up database, folders, config, templates
2. **File Scanning**: Recursive scan of feature folders for task markdown files
3. **Frontmatter Parsing**: Extract task metadata from YAML frontmatter
4. **Database Sync**: Import new tasks, update existing tasks, detect conflicts
5. **Conflict Resolution**: Strategies for file-wins, database-wins, newer-wins
6. **Epic/Feature Inference**: Auto-detect epic/feature from folder structure
7. **Dry-Run Mode**: Preview changes without applying
8. **Transaction Safety**: All-or-nothing sync with rollback on errors
9. **Idempotency**: Safe to run init/sync multiple times

**Critical Architectural Decision** (from PRD):
- Status is stored ONLY in database, NOT in file frontmatter
- Tasks remain in feature folders regardless of status changes
- File frontmatter contains: key (required), title, description, file_path (optional)
- Sync queries database for status using task key

---

## Technology Stack Analysis

### Current Stack

| Component | Technology | Version | Usage in Project |
|-----------|-----------|---------|------------------|
| **Language** | Go | 1.21+ | Primary language |
| **CLI Framework** | Cobra | Latest | Command structure (`internal/cli/`) |
| **Config Management** | Viper | Latest | .pmconfig.json parsing |
| **Database** | SQLite | 3.35+ | Storage (`shark-tasks.db`) |
| **Database Driver** | mattn/go-sqlite3 | Latest | database/sql driver |
| **YAML Parser** | gopkg.in/yaml.v3 | v3 | Frontmatter parsing |
| **Terminal UI** | pterm | Latest | Rich output, colors, tables |
| **Testing** | Go testing | stdlib | Unit and integration tests |
| **File Walking** | path/filepath | stdlib | Recursive directory traversal |

### Go Patterns in Use

**1. Repository Pattern** (from `internal/repository/`):
```go
type TaskRepository struct {
    db *DB
}

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error)
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error
```

**2. Context Propagation**:
- All repository methods accept `context.Context`
- Enables timeout management, cancellation
- Used with database operations

**3. Error Wrapping**:
```go
return fmt.Errorf("failed to create task: %w", err)
```

**4. Transaction Management**:
```go
tx, err := r.db.BeginTxContext(ctx)
defer tx.Rollback()
// ... operations
tx.Commit()
```

**5. Cobra Command Pattern**:
```go
var taskCmd = &cobra.Command{
    Use:   "task",
    Short: "Task management commands",
    RunE: func(cmd *cobra.Command, args []string) error {
        // command logic
    },
}
```

### Sync-Specific Requirements

**New Packages Needed**:
1. `internal/sync/` - Sync engine and orchestration
2. `internal/filescanner/` - Recursive file discovery
3. `internal/init/` - Initialization logic
4. `internal/conflict/` - Conflict detection and resolution

**Extensions to Existing Packages**:
1. `internal/taskfile/parser.go` - Add sync-specific validation
2. `internal/repository/task_repository.go` - Add bulk import methods
3. `internal/db/db.go` - Add sync metadata table (optional)

---

## Naming Conventions

### Existing Conventions (from codebase analysis)

**Go Package Names**: Lowercase, singular or descriptive
- `internal/db`, `internal/cli`, `internal/models`, `internal/repository`
- Pattern: `internal/<domain>/`

**Go File Names**: Lowercase, snake_case
- `task_repository.go`, `task_history_repository.go`, `epic_repository.go`
- Test files: `*_test.go`
- Integration tests: `*_integration_test.go`

**Go Struct Names**: PascalCase
- `Task`, `Epic`, `Feature`, `TaskRepository`, `TaskMetadata`

**Go Function Names**: PascalCase (exported), camelCase (unexported)
- Exported: `Create()`, `GetByKey()`, `ParseTaskFile()`
- Unexported: `queryTasks()`, `configureSQLite()`

**Database Names** (from schema):
- Tables: lowercase, plural - `tasks`, `epics`, `features`, `task_history`
- Columns: snake_case - `task_id`, `created_at`, `feature_id`, `file_path`
- Enums: lowercase - `todo`, `in_progress`, `completed`

**Task Keys** (from validation):
- Pattern: `T-E##-F##-###`
- Example: `T-E04-F07-001`
- Epic keys: `E##` (e.g., `E04`)
- Feature keys: `E##-F##` (e.g., `E04-F07`)

**File Paths**:
- Feature folders: `docs/plan/<epic-folder>/<feature-folder>/T-<key>.md`
- Example: `docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-001.md`
- Legacy folders: `docs/tasks/todo/`, `docs/tasks/active/`, etc.

### New Naming for This Feature

**Commands**:
- `pm init` - Initialization command
- `pm sync` - Synchronization command

**Packages**:
- `internal/sync/` - Sync orchestration
- `internal/filescanner/` - File discovery
- `internal/conflict/` - Conflict resolution

**Structs** (proposed):
- `SyncEngine` - Main sync orchestrator
- `FileScanner` - Recursive file finder
- `ConflictResolver` - Conflict resolution strategies
- `SyncReport` - Sync operation summary
- `InitConfig` - Initialization configuration

---

## File Organization Patterns

### Current Project Structure

```
shark-task-manager/
├── cmd/
│   └── pm/
│       └── main.go                    # CLI entry point
├── internal/
│   ├── cli/
│   │   ├── root.go                    # Root command, config
│   │   └── commands/
│   │       ├── task.go                # Task commands
│   │       ├── epic.go                # Epic commands
│   │       ├── feature.go             # Feature commands
│   │       ├── validate.go            # Validation commands
│   │       └── config.go              # Config management
│   ├── db/
│   │   └── db.go                      # Database initialization
│   ├── models/
│   │   ├── task.go                    # Task model
│   │   ├── epic.go                    # Epic model
│   │   ├── feature.go                 # Feature model
│   │   └── task_history.go           # Task history model
│   ├── repository/
│   │   ├── repository.go              # Base repository (DB wrapper)
│   │   ├── task_repository.go         # Task CRUD
│   │   ├── epic_repository.go         # Epic CRUD
│   │   ├── feature_repository.go      # Feature CRUD
│   │   └── task_history_repository.go # History CRUD
│   ├── taskfile/
│   │   ├── parser.go                  # YAML frontmatter parser
│   │   └── writer.go                  # Markdown file writer
│   ├── taskcreation/
│   │   ├── creator.go                 # Task creation orchestration
│   │   ├── keygen.go                  # Key generation
│   │   └── validator.go               # Task validation
│   ├── templates/
│   │   ├── loader.go                  # Template loading
│   │   └── renderer.go                # Template rendering
│   ├── filepath/
│   │   └── filepath.go                # File path utilities
│   └── formatters/
│       ├── formatter.go               # Output formatting
│       └── json.go                    # JSON formatting
├── docs/
│   ├── plan/                          # Epic and feature documentation
│   │   └── E04-task-mgmt-cli-core/
│   │       ├── epic.md
│   │       └── E04-F07-initialization-sync/
│   │           └── prd.md
│   └── tasks/                         # Legacy task folders (optional)
│       ├── todo/
│       ├── active/
│       └── completed/
├── templates/                         # Task templates (markdown)
├── shark-tasks.db                     # SQLite database
└── .pmconfig.json                     # Config file (created by pm init)
```

### Proposed Structure for This Feature

```
internal/
├── cli/commands/
│   ├── init.go                # NEW: pm init command
│   └── sync.go                # NEW: pm sync command
├── sync/
│   ├── engine.go              # NEW: Sync orchestration
│   ├── scanner.go             # NEW: File scanning
│   ├── importer.go            # NEW: Task import logic
│   ├── conflict.go            # NEW: Conflict detection
│   ├── resolver.go            # NEW: Conflict resolution
│   └── report.go              # NEW: Sync report generation
├── init/
│   ├── initializer.go         # NEW: Init orchestration
│   ├── schema.go              # NEW: Schema creation (delegates to db)
│   ├── folders.go             # NEW: Folder structure creation
│   ├── config.go              # NEW: Config file generation
│   └── templates.go           # NEW: Template copying
└── repository/
    └── task_repository.go     # EXTEND: Add BulkCreate, GetByKeys methods
```

---

## Testing Standards

### Existing Test Patterns (from codebase)

**1. Unit Tests** (example: `internal/taskfile/parser_test.go`):
- Use table-driven tests
- Test file: `<package>_test.go`
- Use `t.Run()` for subtests

**2. Integration Tests** (example: `internal/taskcreation/integration_test.go`):
- Use `*_integration_test.go` suffix
- Use `internal/test/testdb.go` for test database setup
- Clean up test database in `defer`

**3. Test Helpers** (`internal/test/testdb.go`):
```go
func SetupTestDB(t *testing.T) *sql.DB
func TeardownTestDB(t *testing.T, db *sql.DB)
```

### Test Strategy for This Feature

**1. Init Command Tests**:
- Unit: Test folder creation, config generation
- Integration: Test full `pm init` execution
- Idempotency: Test running `pm init` multiple times

**2. Sync Command Tests**:
- Unit: Test file scanning, frontmatter parsing, conflict detection
- Integration: Test full sync with filesystem and database
- Dry-run: Test preview mode doesn't modify data
- Transaction: Test rollback on errors

**3. File Scanner Tests**:
- Test recursive directory traversal
- Test pattern matching (T-*.md files)
- Test feature folder detection
- Test legacy folder support

**4. Conflict Resolution Tests**:
- Test file-wins strategy
- Test database-wins strategy
- Test newer-wins strategy (timestamp comparison)

**5. Performance Tests**:
- Test sync with 100 files (target: <10 seconds from PRD)
- Test file parsing performance (<10ms per file from PRD)

---

## Existing Persistence Patterns

### Database Interaction Patterns

**1. Context-Aware Operations** (from `internal/repository/task_repository.go`):
```go
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
    query := `INSERT INTO tasks (...) VALUES (...)`
    result, err := r.db.ExecContext(ctx, query, args...)
    // ...
}
```

**Pattern**: All database operations accept `context.Context` for cancellation.

**2. Transaction Pattern** (from `UpdateStatus` method):
```go
tx, err := r.db.BeginTxContext(ctx)
if err != nil {
    return fmt.Errorf("failed to begin transaction: %w", err)
}
defer tx.Rollback()

// Multiple operations...

if err := tx.Commit(); err != nil {
    return fmt.Errorf("failed to commit transaction: %w", err)
}
```

**Pattern**: Explicit transaction with deferred rollback (safety net).

**3. Atomic Updates with History** (from `UpdateStatus` method):
```go
// Start transaction
// 1. Query current status
// 2. Update task status and timestamps
// 3. Create history record
// Commit transaction
```

**Pattern**: Multi-step operations are atomic via transactions.

**4. Error Handling Pattern**:
```go
if err == sql.ErrNoRows {
    return nil, fmt.Errorf("task not found with key %s", key)
}
if err != nil {
    return nil, fmt.Errorf("failed to get task: %w", err)
}
```

**Pattern**: Check for specific errors first, then wrap generic errors.

### File Interaction Patterns

**1. YAML Frontmatter Parsing** (from `internal/taskfile/parser.go`):
```go
// Read file line by line
// Find frontmatter delimiters (---)
// Join frontmatter lines
// yaml.Unmarshal into struct
// Read remaining content
```

**Pattern**: Sequential parsing with delimiter detection.

**2. Atomic File Writing** (from `internal/taskfile/writer.go`):
```go
// Write to temporary file
// Sync to disk
// Rename to target (atomic on Unix)
```

**Pattern**: Atomic writes via temp file + rename.

### Sync-Specific Patterns Needed

**1. Bulk Import Pattern** (new requirement):
```go
func (r *TaskRepository) BulkCreate(ctx context.Context, tasks []*models.Task) error {
    tx, err := r.db.BeginTxContext(ctx)
    defer tx.Rollback()

    stmt, err := tx.PrepareContext(ctx, insertQuery)
    defer stmt.Close()

    for _, task := range tasks {
        _, err = stmt.ExecContext(ctx, args...)
        if err != nil {
            return err // triggers rollback
        }
    }

    return tx.Commit()
}
```

**Pattern**: Prepared statement + transaction for bulk inserts.

**2. File Walking Pattern** (new requirement):
```go
func ScanFeatureFolders(rootPath string) ([]string, error) {
    var taskFiles []string

    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() && strings.HasSuffix(path, ".md") {
            // Check if filename matches T-*.md pattern
            if isTaskFile(info.Name()) {
                taskFiles = append(taskFiles, path)
            }
        }

        return nil
    })

    return taskFiles, err
}
```

**Pattern**: filepath.Walk with filter function.

**3. Epic/Feature Inference Pattern** (new requirement):
```go
func InferFeatureFromPath(filePath string) (epicKey, featureKey string, err error) {
    // Parse path: docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-001.md
    // Extract epic from parent-parent directory name
    // Extract feature from parent directory name
    // Validate against task key in frontmatter
}
```

**Pattern**: Path parsing with validation.

---

## Schema Design Patterns

### Existing Schema (from `internal/db/db.go`)

**Tables**:
1. `epics` - Epic records
2. `features` - Feature records (FK to epics)
3. `tasks` - Task records (FK to features)
4. `task_history` - History records (FK to tasks)

**Key Fields in Tasks Table**:
```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feature_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN (...)),
    agent_type TEXT CHECK (agent_type IN (...)),
    priority INTEGER NOT NULL DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    depends_on TEXT,           -- JSON array
    assigned_agent TEXT,
    file_path TEXT,            -- IMPORTANT: Current file location
    blocked_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    blocked_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
);
```

**Critical Field for Sync**: `file_path`
- Stores current file location
- Updated during sync if file moved
- Used to detect file renames/moves

### Sync Metadata Options

**Option 1: Add Columns to Existing Tables** (lightweight):
```sql
ALTER TABLE tasks ADD COLUMN last_synced_at TIMESTAMP;
ALTER TABLE tasks ADD COLUMN sync_hash TEXT;  -- MD5 of content
```

**Option 2: Create Sync Metadata Table** (more flexible):
```sql
CREATE TABLE sync_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_key TEXT NOT NULL UNIQUE,
    last_synced_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    file_modified_at TIMESTAMP,
    db_modified_at TIMESTAMP,
    content_hash TEXT,           -- MD5 of frontmatter + content
    last_conflict_resolution TEXT,  -- 'file-wins', 'database-wins', 'newer-wins'
    FOREIGN KEY (task_key) REFERENCES tasks(key) ON DELETE CASCADE
);
```

**Recommendation**: Start with **Option 1** (simpler) unless PRD explicitly requires full sync history. Can migrate to Option 2 later if needed.

**For MVP**: Use `updated_at` timestamp in tasks table + file modified time for newer-wins strategy. No new table needed.

---

## CLI Command Patterns

### Existing Command Structure (from `internal/cli/commands/`)

**Pattern 1: Cobra Command Definition**:
```go
var taskCmd = &cobra.Command{
    Use:   "task",
    Short: "Task management commands",
    Long:  `...`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Command logic
        return nil
    },
}

func init() {
    RootCmd.AddCommand(taskCmd)
    taskCmd.Flags().StringP("epic", "e", "", "Epic key")
    taskCmd.Flags().StringP("feature", "f", "", "Feature key")
}
```

**Pattern 2: JSON Output Support**:
```go
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(result)
}
// Otherwise, pretty-print
cli.Success("Operation completed")
```

**Pattern 3: Database Connection**:
```go
dbPath, err := cli.GetDBPath()
if err != nil {
    return err
}

db, err := sql.Open("sqlite3", dbPath)
if err != nil {
    return fmt.Errorf("failed to open database: %w", err)
}
defer db.Close()
```

### New Commands for This Feature

**1. Init Command** (`pm init`):
```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize PM CLI infrastructure",
    Long:  `Creates database schema, folder structure, config file, and templates.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Init logic
    },
}

// Flags:
// --non-interactive: Skip prompts
// --force: Overwrite existing config
```

**2. Sync Command** (`pm sync`):
```go
var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Synchronize task files with database",
    Long:  `Scans feature folders for task markdown files and syncs with database.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Sync logic
    },
}

// Flags:
// --folder: Sync specific folder only
// --dry-run: Preview changes
// --strategy: Conflict resolution strategy (file-wins, database-wins, newer-wins)
// --force: Overwrite database with file contents
// --create-missing: Auto-create missing epics/features
// --cleanup: Delete orphaned database records
// --json: JSON output
```

---

## Performance Considerations

### Performance Requirements (from PRD)

1. **Init Performance**: `pm init` must complete in <5 seconds
2. **Sync Performance**: Process 100 files in <10 seconds
3. **YAML Parsing**: <10ms per file
4. **Database Operations**: Use bulk inserts for efficiency

### Optimization Strategies

**1. Bulk Database Operations**:
```go
// Prepare statement once, execute multiple times
stmt, err := tx.PrepareContext(ctx, insertQuery)
defer stmt.Close()

for _, task := range tasks {
    _, err = stmt.ExecContext(ctx, args...)
}
```

**2. Parallel File Parsing** (optional optimization):
```go
// Use goroutines + channels for parallel parsing
type parseResult struct {
    file     string
    metadata *TaskMetadata
    err      error
}

results := make(chan parseResult, len(files))
for _, file := range files {
    go func(f string) {
        metadata, err := ParseTaskFile(f)
        results <- parseResult{f, metadata, err}
    }(file)
}
```

**Caution**: Only parallelize if profiling shows parsing is bottleneck.

**3. Database Indexing** (already exists):
- `tasks.key` has UNIQUE index (automatic)
- `tasks.feature_id` has index
- `tasks.status` has index

**4. Transaction Batching**:
- Use single transaction for entire sync
- Rollback if any operation fails
- Commit only at end (atomic sync)

---

## Security Patterns

### Existing Security (from PRD and codebase)

1. **Foreign Key Enforcement**: Enabled in SQLite
2. **Parameterized Queries**: Used throughout repositories
3. **Database File Permissions**: Should be 600 (read/write owner only)

### Sync-Specific Security Concerns

**1. File Path Injection**:
- Validate that scanned files are within allowed directories
- Sanitize file paths before database storage
- Prevent path traversal attacks

**2. YAML Injection**:
- YAML parsing is read-only (no code execution risk with gopkg.in/yaml.v3)
- Validate frontmatter structure
- Reject files with invalid/malicious YAML

**3. Atomic Operations**:
- All sync operations in transaction (rollback on error)
- File writes are atomic (temp file + rename pattern exists)

**4. Backup Strategy**:
- `pm sync --backup` flag to create database backup before sync
- Store backup with timestamp: `shark-tasks.db.backup.2025-12-16T10-30-00`

---

## Error Handling Patterns

### Existing Error Patterns (from codebase)

**1. Error Wrapping** (Go 1.13+ pattern):
```go
return fmt.Errorf("failed to create task: %w", err)
```

**2. Specific Error Checks**:
```go
if err == sql.ErrNoRows {
    return nil, fmt.Errorf("task not found with key %s", key)
}
```

**3. Exit Codes** (from CLI framework):
- 0: Success
- 1: Validation error, user error
- 2: System error (database, filesystem)

### Sync-Specific Error Handling

**1. Sync Errors** (new error types needed):
```go
type SyncError struct {
    File    string
    Message string
    Err     error
}

func (e *SyncError) Error() string {
    return fmt.Sprintf("sync error in %s: %s", e.File, e.Message)
}
```

**2. Error Recovery Strategy**:
- Invalid YAML: Log warning, skip file, continue
- Missing required field: Log warning, skip file, continue
- Database error: Rollback, halt sync, exit code 2
- Filesystem error: Rollback, halt sync, exit code 2

**3. Conflict Detection** (not an error, but needs reporting):
```go
type Conflict struct {
    TaskKey       string
    Field         string
    FileValue     string
    DatabaseValue string
    Resolution    string  // 'file-wins', 'database-wins', 'newer-wins'
}
```

---

## Related PRD Analysis

### Dependencies from Other Features

**E04-F01: Database Schema**
- Provides tables: epics, features, tasks, task_history
- Provides db.InitDB() for schema creation
- Used by: `pm init` for database setup

**E04-F02: CLI Framework**
- Provides Cobra command structure
- Provides config management (Viper)
- Provides CLI utilities (Success, Error, Warning, OutputJSON)
- Used by: `pm init` and `pm sync` commands

**E04-F05: Folder Management**
- Provides folder structure patterns
- Used by: `pm init` for creating folders

**E04-F06: Task Creation**
- Provides task validation logic
- Provides key generation patterns
- Provides file writing patterns (atomic writes)
- Used by: `pm sync` for validating imported tasks

### Integration Points

**1. Init Command Integration**:
- Uses `db.InitDB()` to create schema
- Uses folder creation patterns from E04-F05
- Creates `.pmconfig.json` with Viper

**2. Sync Command Integration**:
- Uses `taskfile.ParseTaskFile()` to read frontmatter
- Uses `TaskRepository.Create()` to import tasks
- Uses `TaskRepository.Update()` to sync changes
- Uses `TaskRepository.GetByKey()` to check existence
- Uses `EpicRepository` and `FeatureRepository` to infer/create epics and features

**3. Transaction Coordination**:
- Single transaction for entire sync operation
- Rollback if any operation fails
- Commit only after all files processed

---

## Recommendations

### Critical Implementation Decisions

1. **Use existing `taskfile.ParseTaskFile()`** for frontmatter parsing
2. **Extend TaskRepository** with `BulkCreate()` and `GetByKeys()` methods
3. **Create new packages**: `internal/sync/`, `internal/init/`
4. **Use filepath.Walk** for recursive file scanning
5. **Single transaction per sync** for atomicity
6. **Store file_path in database** for conflict detection
7. **Use updated_at timestamps** for newer-wins strategy (no new table needed)
8. **Validate epic/feature existence** before importing tasks
9. **Support --create-missing flag** to auto-create epics/features
10. **JSON output support** for programmatic use

### Open Questions for Design Phase

1. **Sync metadata storage**: Add columns to tasks table vs. new table? → **Recommendation: Use updated_at only (no new table for MVP)**
2. **Parallel file parsing**: Needed for 100 files in 10 seconds? → **Recommendation: Profile first, optimize if needed**
3. **Conflict history**: Store past conflicts in database? → **Recommendation: Log to stdout only (no persistence for MVP)**
4. **Epic/feature auto-creation**: What metadata to use? → **Recommendation: Minimal metadata (key + inferred title from folder name)**
5. **Legacy folder support**: Scan docs/tasks/* folders? → **Recommendation: Yes, support with --folder flag**

### Next Steps

1. **01-interface-contracts.md**: Define interfaces for SyncEngine, FileScanner, ConflictResolver
2. **03-data-design.md**: Decide on sync metadata approach (new columns vs. new table)
3. **02-architecture.md**: Design sync orchestration flow and component interactions
4. **04-backend-design.md**: Detail Go package structure and implementation
5. **05-frontend-design.md**: Design CLI UX (commands, flags, output formats)
6. **06-security-design.md**: Expand file path validation and atomic operation details

---

## References

### Go Ecosystem

- Cobra CLI Framework: https://github.com/spf13/cobra
- Viper Config: https://github.com/spf13/viper
- gopkg.in/yaml.v3: https://github.com/go-yaml/yaml
- pterm Terminal UI: https://github.com/pterm/pterm
- database/sql: https://pkg.go.dev/database/sql
- path/filepath: https://pkg.go.dev/path/filepath

### Project Files Analyzed

- `internal/db/db.go` - Database initialization
- `internal/repository/task_repository.go` - Task CRUD operations
- `internal/taskfile/parser.go` - YAML frontmatter parsing
- `internal/cli/root.go` - CLI framework setup
- `internal/models/task.go` - Task model
- PRD: `docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/prd.md`

### Similar Projects (for pattern reference)

- git-sync: Bidirectional file/database sync
- terraform: State file management and drift detection
- alembic: Database migration orchestration

---

**Research Complete**: 2025-12-16
**Next Document**: 01-interface-contracts.md (coordinator creates)
