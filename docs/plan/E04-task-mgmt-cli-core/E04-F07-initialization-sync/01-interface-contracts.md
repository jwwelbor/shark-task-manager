# Interface Contracts: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-16
**Author**: feature-architect (coordinator)

## Purpose

This document defines the contracts between system components for initialization and synchronization operations. All architects MUST align their designs with these contracts.

---

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                                │
│  ┌──────────────────┐              ┌──────────────────┐         │
│  │  pm init         │              │  pm sync         │         │
│  │  (init command)  │              │  (sync command)  │         │
│  └────────┬─────────┘              └────────┬─────────┘         │
│           │                                  │                    │
└───────────┼──────────────────────────────────┼────────────────────┘
            │                                  │
            ▼                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Application Layer                            │
│  ┌──────────────────┐              ┌──────────────────┐         │
│  │  Initializer     │              │  SyncEngine      │         │
│  │  (orchestrator)  │              │  (orchestrator)  │         │
│  └────────┬─────────┘              └────────┬─────────┘         │
│           │                                  │                    │
│           │                        ┌─────────┴─────────┐         │
│           │                        │                   │         │
│           │                        ▼                   ▼         │
│           │              ┌──────────────────┐ ┌──────────────┐  │
│           │              │  FileScanner     │ │  Conflict    │  │
│           │              │  (file discovery)│ │  Resolver    │  │
│           │              └──────────────────┘ └──────────────┘  │
└───────────┼──────────────────────┬──────────────────────────────┘
            │                      │
            ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Data Layer                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐  │
│  │  TaskRepository  │  │  EpicRepository  │  │  Feature     │  │
│  │                  │  │                  │  │  Repository  │  │
│  └────────┬─────────┘  └────────┬─────────┘  └──────┬───────┘  │
│           │                     │                    │           │
└───────────┼─────────────────────┼────────────────────┼───────────┘
            │                     │                    │
            └─────────────────────┴────────────────────┘
                                  │
                                  ▼
                          ┌──────────────────┐
                          │  SQLite Database │
                          └──────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      Filesystem Layer                            │
│  ┌──────────────────┐  ┌──────────────────┐                     │
│  │  TaskFileParser  │  │  TaskFileWriter  │                     │
│  │  (YAML parser)   │  │  (atomic write)  │                     │
│  └────────┬─────────┘  └────────┬─────────┘                     │
│           │                     │                                │
└───────────┼─────────────────────┼────────────────────────────────┘
            │                     │
            ▼                     ▼
      ┌──────────────────────────────────┐
      │  Markdown Files (task frontmatter)│
      └──────────────────────────────────┘
```

---

## 1. CLI Command Contracts

### 1.1 Init Command

**Command**: `pm init`

**Flags**:
```go
--non-interactive   // Skip all prompts (for automation)
--force            // Overwrite existing config and templates
--db string        // Database path (default: shark-tasks.db)
--config string    // Config file path (default: .pmconfig.json)
```

**Exit Codes**:
- `0` - Success (database, folders, config created)
- `1` - User error (invalid flags)
- `2` - System error (permission denied, disk full, database error)

**Output (Human-Readable)**:
```
Shark CLI initialized successfully!

✓ Database created: shark-tasks.db
✓ Folder structure created: docs/plan/, templates/
✓ Config file created: .pmconfig.json
✓ Templates copied: 2 files

Next steps:
1. Edit .pmconfig.json to set default epic and agent
2. Create tasks with: pm task create --epic=E01 --feature=F01 --title="Task title" --agent=backend
3. Import existing tasks with: pm sync
```

**Output (JSON)**:
```json
{
  "status": "success",
  "database_created": true,
  "database_path": "shark-tasks.db",
  "folders_created": ["docs/plan", "templates"],
  "config_created": true,
  "config_path": ".pmconfig.json",
  "templates_copied": 2
}
```

### 1.2 Sync Command

**Command**: `pm sync`

**Flags**:
```go
--folder string       // Sync specific folder only (e.g., docs/plan/E04-epic/E04-F06-feature)
--dry-run            // Preview changes without applying
--strategy string    // Conflict resolution: file-wins, database-wins, newer-wins (default: file-wins)
--force              // Alias for --strategy=file-wins
--create-missing     // Auto-create missing epics/features
--cleanup            // Delete orphaned database records (tasks without files)
--json               // JSON output
--db string          // Database path
```

**Exit Codes**:
- `0` - Success (sync completed, conflicts resolved)
- `1` - User error (invalid flags, missing epic/feature without --create-missing)
- `2` - System error (database error, filesystem error, transaction rollback)

**Output (Human-Readable)**:
```
Sync completed:
  Files scanned: 47
  New tasks imported: 5
  Existing tasks updated: 3
  Conflicts resolved: 2
  Warnings: 1
  Errors: 0

Conflicts:
  T-E01-F02-003:
    Field: title
    Database: "Implement user authentication"
    File: "Add user authentication feature"
    Resolution: file-wins (title updated to "Add user authentication feature")

Warnings:
  - Invalid frontmatter in docs/tasks/legacy/invalid.md, skipping
```

**Output (JSON)**:
```json
{
  "status": "success",
  "summary": {
    "files_scanned": 47,
    "tasks_imported": 5,
    "tasks_updated": 3,
    "conflicts_resolved": 2,
    "warnings": 1,
    "errors": 0
  },
  "conflicts": [
    {
      "task_key": "T-E01-F02-003",
      "field": "title",
      "database_value": "Implement user authentication",
      "file_value": "Add user authentication feature",
      "resolution": "file-wins"
    }
  ],
  "warnings": [
    "Invalid frontmatter in docs/tasks/legacy/invalid.md, skipping"
  ]
}
```

---

## 2. Initializer Interface

**Package**: `internal/init`

**Interface**:
```go
type Initializer interface {
    // Initialize sets up Shark CLI infrastructure
    // Returns error if initialization fails
    Initialize(ctx context.Context, opts InitOptions) (*InitResult, error)
}

type InitOptions struct {
    DBPath          string  // Database file path
    ConfigPath      string  // Config file path (default: .pmconfig.json)
    NonInteractive  bool    // Skip prompts
    Force           bool    // Overwrite existing config/templates
}

type InitResult struct {
    DatabaseCreated bool    // True if database was created (false if already existed)
    DatabasePath    string  // Absolute path to database
    FoldersCreated  []string // List of folders created
    ConfigCreated   bool    // True if config was created
    ConfigPath      string  // Absolute path to config
    TemplatesCopied int     // Number of templates copied
}
```

**Contract**:
1. MUST be idempotent (safe to run multiple times)
2. MUST create database schema if not exists (via `db.InitDB()`)
3. MUST create folder structure: `docs/plan/`, `templates/`
4. MUST create config file `.pmconfig.json` with defaults (if not exists or --force)
5. MUST copy task templates to `templates/` folder
6. MUST NOT overwrite existing config unless --force flag set
7. MUST prompt user before overwriting config (unless --non-interactive)
8. MUST set database file permissions to 600 on Unix systems
9. MUST return error if database creation fails
10. MUST complete in <5 seconds (from PRD performance requirement)

**Default Config Content**:
```json
{
  "default_epic": null,
  "default_agent": null,
  "color_enabled": true,
  "json_output": false
}
```

---

## 3. SyncEngine Interface

**Package**: `internal/sync`

**Interface**:
```go
type SyncEngine interface {
    // Sync synchronizes filesystem with database
    // Returns sync report and error if sync fails
    Sync(ctx context.Context, opts SyncOptions) (*SyncReport, error)
}

type SyncOptions struct {
    DBPath           string              // Database file path
    FolderPath       string              // Specific folder to sync (empty = all)
    DryRun           bool                // Preview changes only
    Strategy         ConflictStrategy    // Conflict resolution strategy
    CreateMissing    bool                // Auto-create missing epics/features
    Cleanup          bool                // Delete orphaned database records
}

type ConflictStrategy string

const (
    ConflictStrategyFileWins     ConflictStrategy = "file-wins"
    ConflictStrategyDatabaseWins ConflictStrategy = "database-wins"
    ConflictStrategyNewerWins    ConflictStrategy = "newer-wins"
)

type SyncReport struct {
    FilesScanned      int         // Total files scanned
    TasksImported     int         // New tasks imported
    TasksUpdated      int         // Existing tasks updated
    TasksDeleted      int         // Orphaned tasks deleted (if --cleanup)
    ConflictsResolved int         // Number of conflicts detected and resolved
    Warnings          []string    // Non-fatal warnings (invalid YAML, etc.)
    Errors            []string    // Fatal errors (should be empty if no error returned)
    Conflicts         []Conflict  // Detailed conflict information
}

type Conflict struct {
    TaskKey       string  // Task key (e.g., T-E01-F02-003)
    Field         string  // Conflicting field (title, description, file_path)
    DatabaseValue string  // Value in database
    FileValue     string  // Value in file
    Resolution    string  // Resolution strategy applied
}
```

**Contract**:
1. MUST scan files recursively under specified folder (or `docs/plan/` if none)
2. MUST parse YAML frontmatter using existing `taskfile.ParseTaskFile()`
3. MUST extract task key from frontmatter (required field)
4. MUST query database for task using key
5. MUST detect conflicts between file and database (title, description, file_path)
6. MUST apply conflict resolution strategy
7. MUST infer epic/feature from folder structure or task key pattern
8. MUST validate epic/feature exists in database (or create if --create-missing)
9. MUST use single transaction for all database operations
10. MUST rollback transaction on any error
11. MUST NOT modify database if --dry-run flag set
12. MUST log warnings for invalid YAML (skip file, continue)
13. MUST create task_history records for imported/updated tasks
14. MUST update file_path in database to match actual file location
15. MUST process 100 files in <10 seconds (from PRD performance requirement)

**Status Handling Contract** (from PRD architectural decision):
- Status is NOT read from file frontmatter
- Status is ONLY stored in database
- When importing new task, set status to "todo" (default)
- When syncing existing task, DO NOT modify status (preserve database value)
- File frontmatter MUST NOT contain status field

---

## 4. FileScanner Interface

**Package**: `internal/sync`

**Interface**:
```go
type FileScanner interface {
    // Scan recursively scans directory for task markdown files
    // Returns list of absolute file paths matching pattern T-*.md
    Scan(rootPath string) ([]string, error)
}

type TaskFileInfo struct {
    FilePath    string     // Absolute path to file
    FileName    string     // Filename (e.g., T-E04-F07-001.md)
    EpicKey     string     // Inferred epic key (e.g., E04)
    FeatureKey  string     // Inferred feature key (e.g., E04-F07)
    ModifiedAt  time.Time  // File modified timestamp
}
```

**Contract**:
1. MUST recursively walk directory tree starting at rootPath
2. MUST match files with pattern `T-*.md` (case-sensitive)
3. MUST return absolute file paths
4. MUST infer epic and feature keys from parent directory names
5. MUST handle both feature folder structure (`docs/plan/E##-epic/E##-F##-feature/T-*.md`) and legacy folders (`docs/tasks/todo/T-*.md`)
6. MUST return error if rootPath does not exist
7. MUST skip files without read permissions (log warning)
8. MUST follow symlinks (default filepath.Walk behavior)

**Epic/Feature Inference Rules**:
1. Parse parent directory name: `E##-F##-*` → feature key
2. Parse grandparent directory name: `E##-*` → epic key
3. If inference fails, parse task key from filename: `T-E##-F##-###` → extract E##-F##
4. If all inference fails, return empty epic/feature keys (sync will require --epic --feature flags or skip task)

---

## 5. ConflictResolver Interface

**Package**: `internal/sync`

**Interface**:
```go
type ConflictResolver interface {
    // DetectConflicts compares file metadata with database record
    // Returns list of detected conflicts
    DetectConflicts(fileTask *TaskMetadata, dbTask *models.Task) []Conflict

    // ResolveConflicts applies resolution strategy and returns resolved task
    // Returns updated task model ready for database update
    ResolveConflicts(conflicts []Conflict, fileTask *TaskMetadata, dbTask *models.Task, strategy ConflictStrategy) (*models.Task, error)
}

type TaskMetadata struct {
    Key          string    // Task key (required)
    Title        string    // Task title (optional in file, conflicts with DB if present)
    Description  *string   // Task description (optional)
    FilePath     string    // Actual file path (used for conflict detection)
    ModifiedAt   time.Time // File modified timestamp (for newer-wins strategy)
}
```

**Contract**:
1. MUST detect conflicts for fields: title, description, file_path
2. MUST NOT detect conflicts for: status, priority, agent_type, depends_on (database-only fields)
3. MUST compare file metadata with database record field-by-field
4. MUST treat missing fields in file as "no conflict" (database value preserved)
5. MUST apply resolution strategy:
   - **file-wins**: Use file value, update database
   - **database-wins**: Keep database value, ignore file value
   - **newer-wins**: Compare file.ModifiedAt with db.UpdatedAt, use newer source
6. MUST preserve database-only fields (status, priority, agent_type) during resolution
7. MUST return updated task model with resolved values

**Field Comparison Rules**:
- **title**: Conflict if file has title AND file.title != db.title
- **description**: Conflict if file has description AND file.description != db.description
- **file_path**: Conflict if actual file path != db.file_path (always update to actual location)

---

## 6. Data Transfer Objects (DTOs)

### 6.1 TaskFile (from filesystem)

**Source**: Parsed from YAML frontmatter

**Structure**:
```yaml
---
key: T-E04-F07-001                  # Required
title: "Implement sync engine"      # Optional (conflicts with DB if present)
description: "Core sync logic"      # Optional (conflicts with DB if present)
file_path: docs/plan/.../T-E04-F07-001.md  # Optional (stored reference, may differ from actual)
---
# Markdown content here...
```

**Go Representation** (extends existing `taskfile.TaskMetadata`):
```go
type TaskMetadata struct {
    TaskKey      string   `yaml:"task_key"`  // Rename to Key in new parsing
    Title        string   `yaml:"title"`
    Description  string   `yaml:"description,omitempty"`
    FilePath     string   `yaml:"file_path,omitempty"`
    // Status is INTENTIONALLY OMITTED (not in frontmatter)
    // Priority, AgentType, DependsOn also omitted (database-only)
}
```

### 6.2 Task (database model)

**Source**: Database table `tasks`

**Structure** (existing `models.Task`):
```go
type Task struct {
    ID            int64
    FeatureID     int64
    Key           string          // Unique task key
    Title         string          // Can conflict with file
    Description   *string         // Can conflict with file
    Status        TaskStatus      // DATABASE ONLY (not in file)
    AgentType     *AgentType      // DATABASE ONLY
    Priority      int             // DATABASE ONLY
    DependsOn     *string         // DATABASE ONLY (JSON array)
    AssignedAgent *string         // DATABASE ONLY
    FilePath      *string         // Can conflict with actual file location
    BlockedReason *string
    CreatedAt     time.Time
    StartedAt     sql.NullTime
    CompletedAt   sql.NullTime
    BlockedAt     sql.NullTime
    UpdatedAt     time.Time       // Used for newer-wins strategy
}
```

### 6.3 Conflict

**Purpose**: Represents a detected conflict between file and database

**Structure**:
```go
type Conflict struct {
    TaskKey       string    // e.g., T-E04-F07-001
    Field         string    // "title", "description", "file_path"
    DatabaseValue string    // Current value in database
    FileValue     string    // Value found in file
    Resolution    string    // "file-wins", "database-wins", "newer-wins"
}
```

---

## 7. Repository Extension Contracts

### 7.1 TaskRepository Extensions

**Package**: `internal/repository`

**New Methods**:
```go
// BulkCreate creates multiple tasks in a single transaction
// Returns number of tasks created and error
func (r *TaskRepository) BulkCreate(ctx context.Context, tasks []*models.Task) (int, error)

// GetByKeys retrieves multiple tasks by their keys
// Returns map of key -> task, missing keys are omitted
func (r *TaskRepository) GetByKeys(ctx context.Context, keys []string) (map[string]*models.Task, error)

// UpdateMetadata updates only metadata fields (title, description, file_path)
// Does NOT update status, priority, agent_type (database-only fields)
func (r *TaskRepository) UpdateMetadata(ctx context.Context, task *models.Task) error
```

**Contract**:
1. **BulkCreate**:
   - MUST use prepared statement for performance
   - MUST use single transaction for all inserts
   - MUST rollback if any insert fails
   - MUST validate all tasks before inserting
   - MUST return count of successfully created tasks
   - SHOULD complete 100 inserts in <1 second

2. **GetByKeys**:
   - MUST use IN clause for efficiency (`WHERE key IN (?, ?, ...)`)
   - MUST return map for O(1) lookup
   - MUST NOT error if some keys not found (return partial map)
   - SHOULD complete 100 lookups in <100ms

3. **UpdateMetadata**:
   - MUST update only: title, description, file_path
   - MUST NOT update: status, priority, agent_type, depends_on
   - MUST update updated_at timestamp automatically (trigger)
   - MUST validate task before updating

### 7.2 EpicRepository Extensions

**New Methods**:
```go
// CreateIfNotExists creates epic only if it doesn't exist
// Returns epic (existing or newly created) and whether it was created
func (r *EpicRepository) CreateIfNotExists(ctx context.Context, epic *models.Epic) (*models.Epic, bool, error)

// GetByKey retrieves epic by key
func (r *EpicRepository) GetByKey(ctx context.Context, key string) (*models.Epic, error)
```

**Contract**:
1. MUST check existence before inserting
2. MUST return existing epic if already exists (idempotent)
3. MUST use transaction to prevent race conditions

### 7.3 FeatureRepository Extensions

**New Methods**:
```go
// CreateIfNotExists creates feature only if it doesn't exist
// Returns feature (existing or newly created) and whether it was created
func (r *FeatureRepository) CreateIfNotExists(ctx context.Context, feature *models.Feature) (*models.Feature, bool, error)

// GetByKey retrieves feature by key
func (r *FeatureRepository) GetByKey(ctx context.Context, key string) (*models.Feature, error)
```

**Contract**:
1. MUST check existence before inserting
2. MUST return existing feature if already exists (idempotent)
3. MUST validate epic_id foreign key

---

## 8. Error Contracts

### 8.1 Error Types

```go
// SyncError represents errors during sync operation
type SyncError struct {
    Operation string    // "scan", "parse", "import", "update"
    File      string    // File path where error occurred
    TaskKey   string    // Task key (if known)
    Err       error     // Underlying error
}

// ValidationError represents frontmatter validation failures
type ValidationError struct {
    File      string    // File path
    Field     string    // Missing/invalid field
    Message   string    // Human-readable error
}

// InitError represents initialization failures
type InitError struct {
    Step    string      // "database", "folders", "config", "templates"
    Message string      // Human-readable error
    Err     error       // Underlying error
}
```

### 8.2 Error Handling Strategy

**Non-Fatal Errors** (log warning, skip, continue):
- Invalid YAML frontmatter
- Missing required field (key)
- File read permission denied
- Key mismatch (filename vs. frontmatter)

**Fatal Errors** (rollback, halt, return error):
- Database connection failure
- Transaction begin/commit failure
- Foreign key constraint violation (missing epic/feature without --create-missing)
- Filesystem write permission denied (for --update-files flag)

**Error Code Mapping**:
```go
// CLI exit codes
const (
    ExitSuccess       = 0  // Operation completed successfully
    ExitUserError     = 1  // Invalid flags, validation error
    ExitSystemError   = 2  // Database error, filesystem error
)
```

---

## 9. Transaction Boundary Contracts

### 9.1 Init Transaction Boundary

**Scope**: Database operations only (schema creation)
- Database schema creation uses transaction (if supported by Alembic/migrations)
- Folder creation is NOT transactional (filesystem operations)
- Config file creation is atomic (write to temp, rename)

### 9.2 Sync Transaction Boundary

**Scope**: All database operations in single transaction

**Transaction Includes**:
1. Epic creation/lookup (if --create-missing)
2. Feature creation/lookup (if --create-missing)
3. Task import (INSERT)
4. Task update (UPDATE)
5. Task history creation (INSERT)
6. Task deletion (DELETE, if --cleanup)

**Transaction Excludes**:
- File scanning (read-only, no transaction needed)
- File parsing (read-only)
- File writing (if --update-files, separate atomic operation)

**Rollback Triggers**:
- Any database constraint violation
- Foreign key violation
- Unique key violation (task key already exists)
- Database connection error

**Commit Conditions**:
- All files processed successfully
- All database operations completed
- No fatal errors encountered

---

## 10. Naming Standards

### 10.1 Package Names

- `internal/init` - Initialization logic
- `internal/sync` - Synchronization logic
- Existing: `internal/repository`, `internal/taskfile`, `internal/models`

### 10.2 File Names

- `init.go` - Init command
- `sync.go` - Sync command
- `engine.go` - Sync engine orchestration
- `scanner.go` - File scanning
- `conflict.go` - Conflict detection and resolution
- `report.go` - Sync report generation

### 10.3 Function Names

- Exported: `Initialize()`, `Sync()`, `Scan()`, `DetectConflicts()`, `ResolveConflicts()`
- Unexported: `inferEpicFromPath()`, `parseTaskKey()`, `validateFrontmatter()`

### 10.4 Constants

```go
const (
    DefaultDBPath      = "shark-tasks.db"
    DefaultConfigPath  = ".pmconfig.json"
    FeatureFolderRoot  = "docs/plan"
    LegacyTaskFolder   = "docs/tasks"
    TemplateFolder     = "templates"
)
```

---

## 11. Testing Contracts

### 11.1 Unit Test Requirements

**Init Tests**:
- Test database creation (idempotent)
- Test folder creation (idempotent)
- Test config creation (prompt behavior, --force flag)
- Test template copying

**Sync Tests**:
- Test file scanning (recursive, pattern matching)
- Test frontmatter parsing (valid, invalid YAML)
- Test conflict detection (title, description, file_path)
- Test conflict resolution (file-wins, database-wins, newer-wins)
- Test epic/feature inference (from path, from key)
- Test dry-run mode (no database changes)

**Repository Extension Tests**:
- Test BulkCreate (single transaction, rollback on error)
- Test GetByKeys (partial results, missing keys)
- Test UpdateMetadata (preserves database-only fields)

### 11.2 Integration Test Requirements

- Test full `pm init` command (database + folders + config)
- Test full `pm sync` command (filesystem + database)
- Test sync with conflicts (apply strategies)
- Test transaction rollback (error during sync)
- Test --create-missing flag (auto-create epics/features)
- Test --cleanup flag (delete orphaned tasks)

### 11.3 Performance Test Requirements

- Test `pm init` completes in <5 seconds
- Test `pm sync` with 100 files completes in <10 seconds
- Test YAML parsing <10ms per file
- Test bulk insert 100 tasks in <1 second

---

## Critical Reminders

1. **Status is database-only**: File frontmatter MUST NOT contain status field
2. **Conflict resolution**: Apply strategy consistently across all fields
3. **Transaction safety**: Single transaction for entire sync, rollback on any error
4. **Idempotency**: Both init and sync MUST be safe to run multiple times
5. **Error handling**: Non-fatal errors (invalid YAML) skip file, fatal errors (database) rollback
6. **Epic/feature inference**: Validate existence or create with --create-missing
7. **File path tracking**: Always update database file_path to actual location
8. **Atomic operations**: Use temp file + rename for config writes
9. **Performance**: Bulk operations for efficiency (prepared statements)
10. **Context propagation**: All database operations accept context.Context

---

**Document Complete**: 2025-12-16
**Next Document**: 03-data-design.md (db-admin creates)
