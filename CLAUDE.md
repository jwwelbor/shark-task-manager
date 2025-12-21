# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Shark Task Manager** is a Go-based CLI tool and HTTP API for managing project tasks, features, and epics with AI-driven development workflows. It uses SQLite for persistence and follows clean architecture principles.

### Key Technologies
- **Go**: 1.23.4+ (statically typed, compiled)
- **SQLite**: Local database with WAL mode for concurrency
- **Cobra**: CLI framework for structured command hierarchy
- **Viper**: Configuration management

---

## Quick Build & Development Commands

### Build Commands
```bash
make build              # Build all binaries (shark-task-manager, shark CLI, demo, test-db)
make shark             # Build only the Shark CLI tool
make install-shark     # Install Shark CLI to ~/go/bin
```

### Testing
```bash
make test              # Run all tests with verbose output
make test-coverage     # Run tests with HTML coverage report (coverage.html)
make test-db           # Run specific database integration tests
```

### Code Quality
```bash
make fmt               # Format code with gofmt
make vet               # Run go vet for static analysis
make lint              # Run golangci-lint (auto-installs if needed)
```

### Cleanup
```bash
make clean             # Remove binaries and SQLite files
```

---

## ⚠️ DATABASE MANAGEMENT - CRITICAL

### DO NOT DELETE OR RECREATE THE DATABASE

The database file (`shark-tasks.db`) is the single source of truth for all project data. **Deleting it will cause data loss and sync errors.**

#### What To Do If Database Is Corrupted

If you need to reset the database:

1. **Backup first** (save the .db file elsewhere)
2. **Delete ONLY the database file and WAL files:**
   ```bash
   rm shark-tasks.db shark-tasks.db-shm shark-tasks.db-wal
   ```
3. **Reinitialize:**
   ```bash
   ./bin/shark init --non-interactive
   ```
4. **Resync filesystem to database:**
   ```bash
   ./bin/shark sync --dry-run              # Preview changes
   ./bin/shark sync --strategy=file-wins   # Apply changes
   ```

#### What NOT To Do

❌ **DO NOT** run `make clean` during development (it deletes the database)
❌ **DO NOT** use `rm shark*` or glob patterns that match the database
❌ **DO NOT** delete the database to fix sync errors (fix the sync instead)
❌ **DO NOT** modify task files while running sync operations

#### If Sync Fails with "UNIQUE constraint failed: tasks.key"

This means you're trying to create tasks that already exist. Options:

1. **Check if database exists:**
   ```bash
   ls -lh shark-tasks.db
   ```

2. **If database was deleted, restore from backup:**
   ```bash
   cp /path/to/backup/shark-tasks.db .
   ```

3. **If files are out of sync with database:**
   ```bash
   ./bin/shark sync --dry-run --strategy=database-wins  # Use DB as source of truth
   ```

4. **If specific tasks are duplicated**, manually remove duplicate file or reset task in database

---

## Database Migrations & Custom Folder Paths

### Auto-Migration System

The database uses automatic migrations for backward compatibility:

- Migrations run automatically when `InitDB()` is called
- Each migration checks if columns already exist before adding them
- Safe to run multiple times - idempotent operations
- No manual migration scripts required for end users

### Custom Folder Path Feature

The custom folder base path feature adds new optional columns:

```sql
ALTER TABLE epics ADD COLUMN custom_folder_path TEXT;
ALTER TABLE features ADD COLUMN custom_folder_path TEXT;
```

**Backward Compatible:**
- Existing databases work unchanged
- New columns default to NULL
- Default behavior (`docs/plan/{epic-key}/`) unchanged
- Automatic migration applies on first run

**Indexes for Performance:**
```sql
CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);
CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);
```

### Data Layout Flexibility

With custom folder paths, projects can organize in multiple ways:

**Traditional (default):**
```
docs/plan/
├── E01-epic/
│   ├── epic.md
│   ├── E01-F01-feature/
│   │   └── feature.md
│   └── E01-F02-feature/
│       └── feature.md
└── E02-epic/
    └── epic.md
```

**Organized by Time Period (custom --path):**
```
docs/roadmap/
├── 2025-q1/          # Epic with --path="docs/roadmap/2025-q1"
│   ├── epic.md
│   ├── user-growth/  # Feature inherits path
│   │   └── feature.md
│   └── retention/
│       └── feature.md
└── 2025-q2/          # Epic with --path="docs/roadmap/2025-q2"
    └── epic.md
```

**Mixed Organization:**
```
docs/
├── roadmap/2025/     # Custom path epics
│   ├── epic.md
│   └── features/
├── plan/             # Default path epics
│   ├── E03-epic/
│   │   └── epic.md
└── legacy/           # Legacy features with custom path
    └── feature.md
```

### Migration Guide

For detailed migration instructions, including how to update existing projects, see `docs/MIGRATION_CUSTOM_PATHS.md`:

```bash
# Automatic migration (recommended)
shark epic list  # Any command triggers migration

# Manual verification
sqlite3 shark-tasks.db ".schema epics" | grep custom_folder_path
sqlite3 shark-tasks.db ".schema features" | grep custom_folder_path
```

---

## Project Architecture

### Directory Structure
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
│   ├── reporting/                # Report generation
│   └── test/                     # Test utilities
│
├── Makefile                      # Build and development targets
├── README.md                     # User-facing documentation
└── go.mod                        # Go module definition
```

### Data Flow

**CLI Command → Command Handler → Repository → Database**

1. **Command Layer** (`internal/cli/commands/`): Parse arguments, call repositories
2. **Repository Layer** (`internal/repository/`): CRUD operations, transactions, validation
3. **Database Layer** (`internal/db/`): SQLite schema, constraints, triggers
4. **Models** (`internal/models/`): Strongly-typed data structures with validation

### Key Design Patterns

#### 1. **Dependency Injection via Constructors**
- Repositories created with injected DB: `NewTaskRepository(db *DB)`
- No DI framework; constructor injection is explicit and compile-safe
- Manual wiring in command handlers

#### 2. **Repository Pattern for Data Access**
Each entity (Epic, Feature, Task) has a repository with:
- CRUD methods (Create, Read, Update, Delete)
- Query methods (GetByID, GetByStatus, List, Filter)
- Atomic operations (especially task status transitions)
- Progress calculation for parents (Epic/Feature progress from Task completion)

#### 3. **Cobra Command Structure**
- `RootCmd` in `internal/cli/root.go` with global flags (`--json`, `--no-color`, `--verbose`)
- Subcommands registered via `init()` functions in each command file
- Commands automatically register themselves when imported

#### 4. **File-Database Sync**
- `internal/sync/`: Synchronizes markdown task files with SQLite database
- Handles conflicts (file vs. database wins strategies)
- Discovery scans filesystem for epic/feature/task structure
- Status is managed exclusively in database (not synced from files)

---

## Database Schema & Key Concepts

### Core Tables
- **epics**: Top-level organizational units (E04, E07, etc.)
  - `custom_folder_path`: Optional folder base path for flexible organization (inherited by features)
- **features**: Features within epics (E04-F01, E04-F02, etc.)
  - `custom_folder_path`: Optional folder base path (overrides inherited epic path)
- **tasks**: Atomic work items (T-E04-F06-001, etc.)
  - `file_path`: File location within project
- **task_history**: Audit trail of task status changes

### SQLite Configuration
- **Foreign Keys**: Enabled (`PRAGMA foreign_keys = ON`)
- **WAL Mode**: Write-Ahead Logging for better concurrency
- **Indexes**: 10+ indexes for query performance
- **Triggers**: Auto-update timestamps, cascade deletes
- **Constraints**: NOT NULL, UNIQUE, CHECK, FOREIGN KEY

### Task Lifecycle States
```
todo → in_progress → ready_for_review → completed
                  ↘ blocked ↗ (can return to todo)
```

Commands for state transitions:
- `start`: todo → in_progress
- `complete`: in_progress → ready_for_review
- `approve`: ready_for_review → completed
- `reopen`: ready_for_review → in_progress
- `block/unblock`: Any status ↔ blocked

### Progress Calculation
- Feature progress = (completed tasks / total tasks) × 100%
- Epic progress = sum of all feature progresses / number of features
- Calculated in repository layer, not stored (derived data)

---

## CLI Command Structure

### Root Command: `shark`
Global flags available to all commands:
- `--json`: Machine-readable JSON output (required for AI agents)
- `--no-color`: Disable colored output
- `--verbose` / `-v`: Enable debug logging
- `--db`: Override database path (default: `shark-tasks.db`)
- `--config`: Override config file path (default: `.sharkconfig.json`)

### Command Categories

#### Initialization
- `shark init --non-interactive`: Setup project infrastructure (folders, database, config)

#### Epic Management
- `shark epic create --title="..." [--path=<folder>] [--filename=<path>] [--force] [--priority=...] [--business-value=...] [--json]`
  - `--path`: Custom folder base path for organizing epic. Relative to root. Example: `docs/roadmap/2025-q1`
  - `--filename`: Custom file path (relative to root, must include .md). Takes precedence over `--path`
  - `--force`: Reassign file if already claimed by another epic or feature
- `shark epic list [--json]`
- `shark epic get <epic-key> [--json]`

#### Feature Management
- `shark feature create --epic=<epic-key> --title="..." [--path=<folder>] [--filename=<path>] [--force] [--execution-order=...] [--json]`
  - `--path`: Custom folder path for feature. Inherits epic's path if not specified. Example: `docs/features/auth`
  - `--filename`: Custom file path (relative to root, must include .md). Takes precedence over `--path`
  - `--force`: Reassign file if already claimed by another feature or epic
- `shark feature list [EPIC] [--json]` - List features, optionally filter by epic key
  - Examples: `shark feature list`, `shark feature list E04`, `shark feature list E04 --json`
  - Flag syntax still works: `shark feature list --epic=E04`
- `shark feature get <feature-key> [--json]`

**Custom Folder Path Organization:**

Epic and feature creation now support custom folder base paths (via `--path`) for flexible project organization:

```bash
# Organize by quarter
shark epic create "Q1 2025 Roadmap" --path="docs/roadmap/2025-q1"

# Features inherit epic's custom path
shark feature create --epic=E01 "User Growth"  # Stored in docs/roadmap/2025-q1/

# Feature overrides epic's custom path
shark feature create --epic=E01 "Legacy API" --path="docs/legacy"  # Stored in docs/legacy/
```

**Path Resolution Order (highest to lowest priority):**
1. `--filename` - Explicit file path
2. `--path` - Custom folder base path
3. Inherited from parent (feature inherits from epic)
4. Default: `docs/plan/{epic-key}/` or `docs/plan/{epic-key}/{feature-key}/`

Refer to `docs/CLI_REFERENCE.md` for detailed examples, `docs/MIGRATION_CUSTOM_PATHS.md` for database updates, and database schema changes below.

#### Task Management (Primary AI Interface)
- `shark task next [--agent=<type>] [--epic=<epic>] [--json]`: Get next available task
- `shark task list [EPIC] [FEATURE] [--status=<status>] [--agent=<type>] [--json]` - List tasks with flexible positional filtering
  - Examples: `shark task list`, `shark task list E04`, `shark task list E04 F01`, `shark task list E04-F01`
  - Flag syntax still works: `shark task list --epic=E04 --feature=F01`
- `shark task get <task-key> [--json]`
- `shark task create --epic=E04 --feature=F06 --title="..." [--agent=<type>] [--priority=<1-10>] [--depends-on=...] [--filename=<path>] [--force]`
  - `--filename`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another task
- `shark task start <task-key> [--agent=<agent-id>] [--json]`
- `shark task complete <task-key> [--notes="..."] [--json]` (ready for review)
- `shark task approve <task-key> [--notes="..."] [--json]` (mark completed)
- `shark task reopen <task-key> [--notes="..."] [--json]` (back to in_progress)
- `shark task block <task-key> --reason="..." [--json]`
- `shark task unblock <task-key> [--json]`

#### Synchronization
- `shark sync [--dry-run] [--strategy=<strategy>] [--create-missing] [--cleanup] [--pattern=<type>] [--json]`

#### Configuration
- `shark config set <key> <value>`
- `shark config get <key>`

---

## Task & Feature Creation Standards

### Creating Tasks for Development Work

**All development tasks MUST be created through shark** following this workflow:

1. **Create Feature** (if new feature area):
   ```bash
   ./bin/shark feature create --epic=E07 "Feature Title" --execution-order=1
   ```

2. **Create Tasks** in the feature:
   ```bash
   ./bin/shark task create --epic=E07 --feature=F01 "Task Title" --priority=5
   ```

3. **Update task file** at `docs/plan/{epic}/{feature}/tasks/{task-key}.md`:
   - Add implementation details to task frontmatter
   - Include specification, acceptance criteria, test plan
   - Link related documents using `related-docs:` frontmatter field
   - Example:
     ```yaml
     ---
     task_key: T-E07-F06-001
     status: todo
     feature: /path/to/feature
     priority: 5
     dependencies: []
     related-docs:
       - path/to/design-doc.md
       - path/to/specification.md
     ---
     ```

4. **Generate related documentation** separately:
   - Design documents go in `docs/plan/{epic}/{feature}/`
   - Implementation guides go in `docs/plan/{epic}/{feature}/implementation/`
   - Link these in task `related-docs:` field

5. **DO NOT** create standalone documentation files unless they're referenced in shark tasks

### Task Status & Lifecycle

Tasks flow through these states:
- **todo**: Created but not started
- **in_progress**: Work has begun
- **ready_for_review**: Implementation complete, awaiting approval
- **completed**: Approved and merged
- **blocked**: Waiting on external dependency

Update status with:
```bash
./bin/shark task start <task-key>
./bin/shark task complete <task-key>
./bin/shark task approve <task-key>
./bin/shark task block <task-key> --reason="..."
```

---

## Common Development Tasks

### Adding a New CLI Command
1. Create file in `internal/cli/commands/` (e.g., `my_command.go`)
2. Implement command handler with Cobra's `&cobra.Command`
3. Register in an `init()` function:
   ```go
   func init() {
       cli.RootCmd.AddCommand(myCmd)
   }
   ```
4. Handle `cli.GlobalConfig` for JSON/verbose output
5. Call appropriate repository methods for data operations

### Adding a Repository Method
1. Open the relevant repository file (`task_repository.go`, `epic_repository.go`, etc.)
2. Add method that:
   - Takes `*sql.Tx` for transaction support OR works with `r.db`
   - Returns error as second value (`(T, error)`)
   - Uses prepared statements or parameterized queries
   - Includes proper error wrapping: `fmt.Errorf("operation failed: %w", err)`

### Running a Single Test
```bash
go test -v ./internal/repository -run TestTaskStatusUpdate
```

### Database Debugging
```bash
sqlite3 shark-tasks.db          # Open SQLite CLI
.tables                          # List tables
.schema tasks                    # View task table schema
SELECT * FROM tasks LIMIT 5;    # Query data
```

### Hot-Reload Development
```bash
make dev  # Starts air which watches for file changes and rebuilds
```

---

## Important Patterns & Constraints

### Error Handling
- Always return errors explicitly; use `fmt.Errorf("context: %w", err)` for wrapping
- Exit codes: 0 (success), 1 (not found), 2 (DB error), 3 (invalid state)
- Never ignore errors with `_`; if unused, return them

### Database Transactions
- Use `tx.Rollback()` with defer for all multi-statement operations
- Atomic status updates wrap multiple queries in a single transaction
- `task_history` records created automatically with triggers for status changes

### CLI Output
- Use `cli.GlobalConfig.JSON` to check if JSON output is needed
- Always output JSON with indentation when requested: use `cli.OutputJSON(data)`
- Table output via `cli.OutputTable(headers, rows)` for human readability
- Use provided functions: `cli.Success()`, `cli.Error()`, `cli.Warning()`, `cli.Info()`

### Validation
- Models have `Validate()` methods in `internal/models/validation.go`
- Validate at model layer BEFORE database operations
- Database constraints (CHECK, FOREIGN KEY) provide additional safety

### File System Sync
- Task files are markdown at `docs/plan/<epic>/<feature>/<task-key>.md`
- File sync is unidirectional: filesystem → database (with conflict resolution)
- Status is NEVER synced from files; it's database-only for audit trail
- Discovery scans filesystem for epic/feature hierarchy

---

## Testing Notes

### Test Organization
- Tests live alongside implementation: `foo.go` + `foo_test.go`
- Test database: `internal/repository/test-shark-tasks.db` (auto-cleaned before test run)
- Use `make test` to clean and run all tests

### Integration Tests
- `internal/repository/*_test.go`: Database-backed tests with real SQLite
- Use `internal/test/testdb.go` to create fresh test database
- Transaction rollback between tests ensures isolation

### Mock Objects
- `mock_task_repository.go`: Used for command testing without database

### Running Tests
```bash
make test              # Full suite
make test-coverage     # With coverage HTML report
go test -v ./...       # Manual verbose run
go test -run TestName  # Specific test
```

---

## Performance & Optimization Notes

### SQLite Configuration
- **WAL Mode**: Better concurrency (writes don't block reads)
- **Busy Timeout**: 5-second timeout prevents immediate failures on contention
- **Memory Mapped I/O**: `mmap_size=30GB` for large databases
- **Cache Size**: `-64000` = ~64MB in-memory cache

### Indexes
- Frequently queried columns (key, status, epic_id, feature_id) are indexed
- Composite indexes on (epic_id, status) for filtered queries
- Index usage improves `task next`, filtering operations

### Progress Calculation
- NOT cached at database level (kept as derived data)
- Feature progress calculated from task count in repository
- Calculated on-demand only when explicitly requested (e.g., `feature get`, `epic get`)

---

## Development Workspace & Patterns

### Dev Workspace Structure

When working on development tasks, use the following workspace pattern:

```
dev-artifacts/{YYYY-MM-DD}-{task-name}/
├── analysis/        # Investigation and documentation
├── scripts/         # Verification and test scripts
├── verification/    # Test results and validation
└── shared/          # Reusable development utilities
```

**Date Formatting**: Extract the current date from system context at conversation start (format: YYYY-MM-DD). Use this date consistently for workspace naming.

**Example**: For a task starting on 2025-12-18 to fix a database bug:
```
dev-artifacts/2025-12-18-fix-database-bug/
├── analysis/DebugInfo-{timestamp}-bug-description.md
├── scripts/verify-fix.sh
├── verification/test-results.txt
└── shared/helper-functions.go
```

### Development Patterns

#### Specifications or Planning
- **DO NOT** include development time estimates or estimated hours/weeks
- **DO include** task complexity sizing using t-shirt sizes (XS, S, M, L, XL, XXL) or story points (1, 2, 3, 5, 8, 13)
- Tasks rated L, XL, XXL, 5, 8, or 13 must be broken down into smaller chunks
- Use `/docs/PRP_WORKFLOW.md` as reference for development workflows

#### Development Artifacts
- Store artifacts in workspace: `dev-artifacts/{YYYY-MM-DD}-{task-name}/`
- Script types:
  - **verification**: Quick tests to validate assumptions
  - **analysis**: Code inspection and pattern discovery
  - **debugging**: Troubleshooting and investigation tools
  - **prototyping**: Experimental implementations
- **Commit guidelines**: Commit only useful artifacts; delete experimental ones
- **Cleanup**: Remove task folders after completion unless valuable for reference

#### Debugging & Troubleshooting
- Document debugging sessions with filename: `DebugInfo-{timestamp}-{5-word-bug-description}.md`
- File must include:
  - Identified problem description
  - Relevant file paths
  - Proposed solution

#### Migration or Refactoring
- Update code to follow project guidelines
- **DO NOT** create migration scripts/artifacts unless explicitly requested
- **DO NOT** leave deprecated methods around unless requested
- Adjust all tests to work with refactored code

#### Testing
- Use the standard `testing` package for creating tests.
- Prefer table-driven tests for covering multiple cases.
- Use interfaces for mocking dependencies, a common Go practice.
- Focus on testing business logic and public APIs.
- Organize related checks within a single test function using `t.Run` for sub-tests.

---

## Version & Release Notes

Check recent commits and `README.md` for release history. The project uses semantic versioning and is actively maintained with frequent enhancements (E01-E07 feature sets completed).

