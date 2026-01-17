---
paths: "{internal,cmd}/**/*"
---

# Project Architecture

This rule is loaded when working with files in `internal/` or `cmd/` directories.

## Directory Structure

```
.
├── cmd/                          # Application entry points
│   ├── shark/                    # Main CLI binary
│   ├── server/                   # HTTP API server
│   ├── demo/                     # Interactive demo program
│   ├── test-db/                  # Database integration tests
│   └── ... (other utilities)
│
├── internal/                     # Private application code
│   ├── cli/                      # CLI framework and commands
│   │   ├── commands/             # Command implementations (init, task, epic, feature, sync, etc.)
│   │   └── root.go               # Root command with global config
│   ├── models/                   # Data types (Epic, Feature, Task, TaskHistory)
│   ├── repository/               # Data access layer
│   │   ├── epic_repository.go    # Epic CRUD + progress calculation
│   │   ├── feature_repository.go # Feature CRUD + progress calculation
│   │   ├── task_repository.go    # Task CRUD + atomic status updates
│   │   └── task_history_repository.go
│   ├── db/                       # Database initialization and schema
│   │   └── db.go                 # SQLite setup, PRAGMA configuration, schema creation
│   ├── fileops/                  # Unified file operations
│   │   ├── writer.go             # EntityFileWriter for atomic file creation
│   │   └── writer_test.go        # Comprehensive test suite (87.1% coverage)
│   ├── init/                     # Project initialization (folders, config, templates)
│   ├── sync/                     # File system sync with database
│   ├── discovery/                # Epic/feature/task discovery from filesystem
│   ├── taskfile/                 # Markdown task file parsing and writing
│   ├── taskcreation/             # Task key generation and validation
│   ├── templates/                # Template rendering
│   ├── formatters/               # Output formatting (JSON, table)
│   ├── config/                   # Configuration management
│   ├── patterns/                 # File pattern matching and validation
│   ├── validation/               # Task/epic/feature validation
│   ├── status/                   # Config-driven status calculations
│   ├── reporting/                # Report generation
│   └── test/                     # Test utilities
```

## Data Flow

**CLI Command → Command Handler → Repository → Database**

1. **Command Layer** (`internal/cli/commands/`): Parse arguments, call repositories
2. **Repository Layer** (`internal/repository/`): CRUD operations, transactions, validation
3. **Database Layer** (`internal/db/`): SQLite schema, constraints, triggers
4. **Models** (`internal/models/`): Strongly-typed data structures with validation

## Key Design Patterns

### 1. Dependency Injection via Constructors
- Repositories created with injected DB: `NewTaskRepository(db *DB)`
- No DI framework; constructor injection is explicit and compile-safe
- Manual wiring in command handlers

### 2. Repository Pattern for Data Access
Each entity (Epic, Feature, Task) has a repository with:
- CRUD methods (Create, Read, Update, Delete)
- Query methods (GetByID, GetByStatus, List, Filter)
- Atomic operations (especially task status transitions)
- Progress calculation for parents (Epic/Feature progress from Task completion)

### 3. Cobra Command Structure
- `RootCmd` in `internal/cli/root.go` with global flags (`--json`, `--no-color`, `--verbose`)
- Subcommands registered via `init()` functions in each command file
- Commands automatically register themselves when imported

### 4. Unified File Operations (fileops Package)
The `internal/fileops` package provides centralized file writing for all entities (epics, features, tasks):

**Key Features:**
- **Atomic Write Protection**: Uses `O_EXCL` flag to prevent race conditions
- **File Existence Handling**: Links to existing files instead of overwriting (unless Force=true)
- **Path Resolution**: Handles both absolute and relative paths
- **Directory Creation**: Automatically creates parent directories
- **Verbose Logging**: Optional logger function for debugging
- **Entity-Specific Behavior**: Task-specific `CreateIfMissing` validation

**Usage Pattern:**
```go
writer := fileops.NewEntityFileWriter()
result, err := writer.WriteEntityFile(fileops.WriteOptions{
    Content:         content,
    ProjectRoot:     projectRoot,
    FilePath:        filePath,
    Verbose:         verbose,
    EntityType:      "task", // or "epic", "feature"
    UseAtomicWrite:  true,   // Recommended for all entities
    CreateIfMissing: true,   // Task-specific flag
    Logger:          logFunc,
})
```

**Benefits:**
- Eliminates ~50+ lines of duplicate code across epic/feature/task creation
- Single point of maintenance for file operations
- Consistent error handling and behavior
- 87.1% test coverage with comprehensive positive and negative tests

**Used By:**
- `internal/cli/commands/epic.go` - Epic file creation
- `internal/cli/commands/feature.go` - Feature file creation
- `internal/taskcreation/creator.go` - Task file creation

### 5. File-Database Sync
- `internal/sync/`: Synchronizes markdown task files with SQLite database
- Handles conflicts (file vs. database wins strategies)
- Discovery scans filesystem for epic/feature/task structure
- Status is managed exclusively in database (not synced from files)

## Database Access Pattern

All CLI commands use a centralized database initialization system for consistency and cloud support.

### Implementation Pattern

**Global Database Instance:**
- Location: `internal/cli/db_global.go`
- Thread-safe singleton with lazy initialization
- Automatic cleanup via Cobra lifecycle hooks
- Cloud-aware (reads `.sharkconfig.json` for backend selection)

**Usage in Commands:**

```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    // Get database (initialized lazily on first call)
    repoDb, err := cli.GetDB(cmd.Context())
    if err != nil {
        return fmt.Errorf("failed to get database: %w", err)
    }

    // Use database
    repo := repository.NewTaskRepository(repoDb)
    // ... business logic ...

    // Note: Connection closed automatically by PersistentPostRunE hook
    return nil
}
```

**Key Features:**
- **Lazy initialization**: Database only created when needed
- **Single instance**: All commands share same connection
- **Automatic cleanup**: PersistentPostRunE hook closes connection after command completes
- **Cloud-aware**: Automatically detects SQLite vs Turso from config
- **Thread-safe**: `sync.Once` ensures initialization happens exactly once

**For Testing:**

```go
func TestMyCommand(t *testing.T) {
    defer cli.ResetDB()  // Clean up global state after test

    // Test code here - command will use cli.GetDB() internally
}
```

**Database Backends:**
- **Local SQLite**: Default, file-based (shark-tasks.db)
- **Turso Cloud**: Cloud-hosted SQLite for multi-machine access
- Backend selection is automatic based on `.sharkconfig.json`

**Architecture Benefits:**
- ✅ 370 lines of duplicate code eliminated
- ✅ All 74 commands get cloud support automatically
- ✅ Single point of maintenance
- ✅ Consistent error handling
- ✅ Easy to add future enhancements (pooling, metrics)

## Project Root Auto-Detection

Shark automatically finds the project root by walking up the directory tree, so you can run commands from any subdirectory within your project without specifying `--db`.

### How It Works

When you run any `shark` command, it automatically searches upward from your current directory looking for:

1. `.sharkconfig.json` (primary marker)
2. `shark-tasks.db` (secondary marker)
3. `.git/` directory (fallback for Git projects)

Once found, shark uses that directory as the project root for:
- Database location (`shark-tasks.db`)
- Configuration file (`.sharkconfig.json`)
- All relative file paths

### Benefits for AI Agents

This feature is particularly useful when AI agents are working in subdirectories:
- No need to track or compute the path to project root
- No risk of creating duplicate databases in subdirectories
- Consistent behavior across all project directories

## Enhanced Status Package (internal/status/)

Provides configuration-driven status calculations and displays for features and epics.

### Overview

The status package enables sophisticated status tracking by reading configuration from `.sharkconfig.json` and applying business logic for:
- **Progress Calculation**: Weighted and completion progress metrics
- **Work Breakdown**: Categorization by responsibility (agent, human, qa_team)
- **Health Indicators**: Feature health status (healthy, warning, critical)
- **Action Items**: Tasks requiring attention (ready_for_* statuses)
- **Impediments**: Blocked tasks with age tracking

### Core Functions

**Progress Calculation:**
```go
CalculateProgress(ctx, feature/epic) -> (weighted%, completion%, total)
```
- Weighted progress based on `progress_weight` in status config
- Completion progress raw percentage of completed tasks
- Total task count

**Work Breakdown:**
```go
CalculateWorkRemaining(ctx, feature) -> WorkSummary
```
- Counts tasks by responsibility (agent, human, qa_team, none, blocked)
- Enables resource allocation and capacity planning
- Shows distribution of work across team

**Status Context:**
```go
GetStatusContext(ctx, feature/epic) -> StatusContext
```
- Color information for display
- Phase information (planning, development, review, qa, done)
- Progress weight configuration
- Responsibility assignment
- Feature blocking configuration

**Action Items:**
```go
GetActionItems(ctx, feature/epic) -> map[status][]tasks
```
- Tasks grouped by actionable status
- Filters to statuses with `blocks_feature: true`
- Enables quick identification of what needs attention

### Configuration Structure

Status metadata is defined in `.sharkconfig.json`:

```json
{
  "status_metadata": {
    "todo": {
      "color": "gray",
      "phase": "planning",
      "progress_weight": 0,
      "responsibility": "none",
      "blocks_feature": false
    },
    "in_progress": {
      "color": "yellow",
      "phase": "development",
      "progress_weight": 50,
      "responsibility": "agent",
      "blocks_feature": false
    },
    "ready_for_approval": {
      "color": "magenta",
      "phase": "review",
      "progress_weight": 75,
      "responsibility": "human",
      "blocks_feature": true
    },
    "completed": {
      "color": "green",
      "phase": "done",
      "progress_weight": 100,
      "responsibility": "none",
      "blocks_feature": false
    }
  }
}
```

**Configuration Fields:**
- `color`: ANSI color for terminal display (red, green, yellow, blue, cyan, magenta, gray, white, orange, purple)
- `phase`: Workflow phase for ordering (planning, development, review, qa, approval, done, any)
- `progress_weight`: Contribution to weighted progress (0-100)
- `responsibility`: Who owns tasks in this status (agent, human, qa_team, none)
- `blocks_feature`: Whether status blocks feature completion (true/false)

### Display Integration

The status package is used by:

**Feature Get Command:**
- Progress breakdown (weighted %, completion %, total)
- Work summary (by responsibility)
- Action items (grouped by status)

**Feature List Command:**
- Health indicators (healthy/warning/critical)
- Dual progress display (weighted | completion)
- Action items count

**Epic Get Command:**
- Feature status rollup (counts by status)
- Task status rollup (aggregated across features)
- Impediments list (blocked tasks with age)

### Calculation Formulas

**Weighted Progress:**
```
(Σ(weight × count_in_status) / total_tasks) × 100%
```

**Completion Progress:**
```
(completed_tasks / total_tasks) × 100%
```

**Health Status:**
```
health = {
  "healthy" if no_blockers AND all_approvals_old_< 3_days,
  "warning" if approvals_old_> 3_days OR minor_blockers,
  "critical" if multiple_blockers OR high_priority_blocked
}
```

### Testing

Status calculations are tested with:
- Mock configurations with various weight distributions
- Multiple feature/epic structures
- Edge cases (0 tasks, all tasks same status, mixed responsibilities)
- Configuration validation
