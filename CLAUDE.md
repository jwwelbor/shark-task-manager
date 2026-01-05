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

### Examples

```bash
# All these commands work the same, regardless of your current directory:

# From project root
./bin/shark task list

# From docs/plan subdirectory
cd docs/plan
../../bin/shark task list  # Still finds shark-tasks.db in root

# From deeply nested directory
cd docs/plan/E04-task-mgmt-cli-core/features
../../../../bin/shark task list  # Still works!
```

### Override Behavior

You can still explicitly specify paths when needed:

```bash
# Use specific database file
shark --db=/path/to/other-project/shark-tasks.db task list

# Use specific config file
shark --config=/path/to/custom-config.json task list
```

### Benefits for AI Agents

This feature is particularly useful when AI agents are working in subdirectories:
- No need to track or compute the path to project root
- No risk of creating duplicate databases in subdirectories
- Consistent behavior across all project directories

---

## Database Migrations

### Auto-Migration System

The database uses automatic migrations for backward compatibility:

- Migrations run automatically when `InitDB()` is called
- Each migration checks if columns already exist before adding them
- Safe to run multiple times - idempotent operations
- No manual migration scripts required for end users

---

## Slug Architecture

Shark supports **dual key format** for epics, features, and tasks, providing both machine-readable numeric keys and human-readable slugged keys.

### Key Formats

**Epics:**
- Numeric: `E04`
- Slugged: `E04-epic-name`

**Features:**
- Numeric: `E04-F02` or `F02`
- Slugged: `E04-F02-feature-name` or `F02-feature-name`

**Tasks:**
- Numeric: `T-E04-F02-001`
- Slugged: `T-E04-F02-001-task-name`

### How It Works

**Automatic Slug Generation:**
- Slugs are automatically generated from titles when entities are created
- Generated by lowercasing title, replacing spaces/underscores with hyphens, removing special characters
- Stored in the `slug` column alongside the numeric `key` column
- Examples:
  - "User Authentication" → `user-authentication`
  - "API_Design & Testing" → `api-design-testing`
  - "Deploy to Production!!!" → `deploy-to-production`

**Dual Key Lookup:**
- All CLI commands accept BOTH numeric and slugged keys
- Repository layer automatically detects format and performs appropriate lookup
- Numeric lookup is tried first for performance (exact match on `key` column)
- Slugged lookup parses the key and matches both numeric key + slug

**Database Schema:**
```sql
ALTER TABLE epics ADD COLUMN slug TEXT;
ALTER TABLE features ADD COLUMN slug TEXT;
ALTER TABLE tasks ADD COLUMN slug TEXT;

CREATE INDEX idx_epics_slug ON epics(slug);
CREATE INDEX idx_features_slug ON features(slug);
CREATE INDEX idx_tasks_slug ON tasks(slug);
```

### Usage Examples

**Epics:**
```bash
# Create epic (slug auto-generated)
shark epic create "User Management System"
# Output: Epic E07 created with slug "user-management-system"

# Both formats work for retrieval
shark epic get E07
shark epic get E07-user-management-system

# List works with both formats
shark epic list
```

**Features:**
```bash
# Create feature
shark feature create --epic=E07 "Authentication & Authorization"
# Output: Feature E07-F01 created with slug "authentication-authorization"

# All these work
shark feature get E07-F01
shark feature get F01
shark feature get E07-F01-authentication-authorization
shark feature get F01-authentication-authorization
```

**Tasks:**
```bash
# Create task
shark task create "Implement JWT token validation" --epic=E07 --feature=F01 --agent=backend
# Output: Task T-E07-F01-001 created with slug "implement-jwt-token-validation"

# All these work
shark task start T-E07-F01-001
shark task start T-E07-F01-001-implement-jwt-token-validation

shark task get T-E07-F01-001
shark task get T-E07-F01-001-implement-jwt-token-validation
```

### Benefits

**For Humans:**
- Self-documenting keys that reveal what the task/feature/epic is about
- Easier to remember and communicate
- Better readability in logs, commits, and documentation

**For AI Agents:**
- Both formats work transparently
- No need to remember which format to use
- Backward compatible with existing numeric keys

**For Systems:**
- Numeric keys remain the source of truth (stored in `key` column)
- Slugs are supplementary for convenience
- No risk of slug conflicts (lookup requires matching both numeric key and slug)

### Implementation Details

**Lookup Strategy (Task Example):**
1. Try exact match on `key` column (handles legacy numeric keys)
2. If not found and key contains slug suffix:
   - Parse numeric key: `T-E07-F01-001` from `T-E07-F01-001-implement-jwt-token-validation`
   - Parse slug: `implement-jwt-token-validation`
   - Query: `WHERE key = ? AND slug = ?`
3. If slug doesn't match, return "not found" (prevents false matches)

**Slug Uniqueness:**
- Slugs are NOT required to be globally unique
- Lookup requires BOTH numeric key AND slug to match
- This prevents collisions between tasks with similar titles
- Example: Two tasks titled "Update README" will have different numeric keys but same slug

**Backward Compatibility:**
- Existing databases work without slugs (slug column is NULL)
- Migration command available to backfill slugs: `shark migrate slugs`
- All commands work with numeric keys even if slugs don't exist

### Slug Migration

For existing databases without slugs:

```bash
# Backfill slugs for all epics, features, and tasks
shark migrate slugs

# Verify slugs were generated
shark task list --json | jq '.[].slug'
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
  - `file_path`: File location within project
- **features**: Features within epics (E04-F01, E04-F02, etc.)
  - `file_path`: File location within project
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

### Key Format Flexibility

**All entity keys are case insensitive:**
- Epic keys: `E07`, `e07`, `E07-user-management`, `e07-user-management`
- Feature keys: `E07-F01`, `e07-f01`, `F01`, `f01`
- Task keys: `E07-F20-001`, `e07-f20-001` (short format), `T-E07-F20-001`, `t-e07-f20-001` (traditional)

**Short task key format (recommended):**
- Use `E07-F20-001` instead of `T-E07-F20-001`
- The `T-` prefix is optional and automatically normalized
- Both formats work identically in all commands

**Positional argument syntax:**
- Feature create: `shark feature create E07 "Feature Title"`
- Task create: `shark task create E07 F20 "Task Title"` or `shark task create E07-F20 "Task Title"`
- Legacy flag syntax still fully supported

### Command Categories

#### Initialization
- `shark init --non-interactive`: Setup project infrastructure (folders, database, config)

#### Epic Management
- `shark epic create --title="..." [--file=<path>] [--force] [--priority=...] [--business-value=...] [--json]`
  - `--file`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another epic or feature
- `shark epic list [--json]`
- `shark epic get <epic-key> [--json]`
  - Case insensitive: `shark epic get E07`, `shark epic get e07`

#### Feature Management
- **Positional syntax (recommended):** `shark feature create <epic-key> "<title>" [--file=<path>] [--force] [--execution-order=...] [--json]`
- **Flag syntax (legacy):** `shark feature create --epic=<epic-key> --title="..." [--file=<path>] [--force] [--execution-order=...] [--json]`
  - `--file`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another feature or epic
  - Case insensitive: `shark feature create E07 "Title"`, `shark feature create e07 "Title"`
- `shark feature list [EPIC] [--json]` - List features, optionally filter by epic key
  - Examples: `shark feature list`, `shark feature list E04`, `shark feature list e04`, `shark feature list E04 --json`
  - Flag syntax still works: `shark feature list --epic=E04`
- `shark feature get <feature-key> [--json]`
  - Case insensitive: `shark feature get E07-F01`, `shark feature get e07-f01`, `shark feature get F01`, `shark feature get f01`

**File Path Organization:**

Epics and features support custom file paths for flexible project organization:

```bash
# Create epic with custom file path
shark epic create "Q1 2025 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# Create feature with custom file path
shark feature create --epic=E01 "User Growth" --file="docs/roadmap/2025-q1/features/user-growth.md"

# Default behavior (no --file flag)
shark epic create "User Management"  # Creates docs/plan/E07-user-management/epic.md
shark feature create E07 "Authentication"  # Positional syntax (recommended)
shark feature create --epic=E07 --title="Authentication"  # Flag syntax (legacy)
# Creates: docs/plan/E07-user-management/E07-F01-authentication/feature.md
```

Refer to `docs/CLI_REFERENCE.md` for detailed examples and usage patterns.

#### Task Management (Primary AI Interface)
- `shark task next [--agent=<type>] [--epic=<epic>] [--json]`: Get next available task
- `shark task list [EPIC] [FEATURE] [--status=<status>] [--agent=<type>] [--json]` - List tasks with flexible positional filtering
  - Examples: `shark task list`, `shark task list E04`, `shark task list e04`, `shark task list E04 F01`, `shark task list E04-F01`
  - Flag syntax still works: `shark task list --epic=E04 --feature=F01`
- `shark task get <task-key> [--json]`
  - Short format (recommended): `shark task get E07-F20-001`, `shark task get e07-f20-001`
  - Traditional format: `shark task get T-E07-F20-001`, `shark task get t-e07-f20-001`
- **Positional syntax (recommended):** `shark task create <epic> <feature> "<title>" [--agent=<type>] [--priority=<1-10>] [--depends-on=...] [--file=<path>] [--force]`
  - 3-arg format: `shark task create E07 F20 "Task Title"`
  - 2-arg format: `shark task create E07-F20 "Task Title"`
  - Case insensitive: `shark task create e07 f20 "Task Title"`
- **Flag syntax (legacy):** `shark task create --epic=E04 --feature=F06 --title="..." [--agent=<type>] [--priority=<1-10>] [--depends-on=...] [--file=<path>] [--force]`
  - `--file`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another task
- `shark task start <task-key> [--agent=<agent-id>] [--json]`
  - Short format: `shark task start E07-F20-001`, `shark task start e07-f20-001`
- `shark task complete <task-key> [--notes="..."] [--json]` (ready for review)
  - Short format: `shark task complete E07-F20-001`, `shark task complete e07-f20-001`
- `shark task approve <task-key> [--notes="..."] [--json]` (mark completed)
  - Short format: `shark task approve E07-F20-001`, `shark task approve e07-f20-001`
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
   # Positional syntax (recommended)
   ./bin/shark feature create E07 "Feature Title" --execution-order=1

   # Flag syntax (legacy, still supported)
   ./bin/shark feature create --epic=E07 --title="Feature Title" --execution-order=1
   ```

2. **Create Tasks** in the feature:
   ```bash
   # Positional syntax (recommended)
   ./bin/shark task create E07 F01 "Task Title" --priority=5
   # OR combined format
   ./bin/shark task create E07-F01 "Task Title" --priority=5

   # Flag syntax (legacy, still supported)
   ./bin/shark task create --epic=E07 --feature=F01 --title="Task Title" --priority=5
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
# Short format (recommended)
./bin/shark task start E07-F20-001
./bin/shark task complete E07-F20-001
./bin/shark task approve E07-F20-001
./bin/shark task block E07-F20-001 --reason="..."

# Traditional format (still supported)
./bin/shark task start T-E07-F20-001
./bin/shark task complete T-E07-F20-001

# Case insensitive
./bin/shark task start e07-f20-001
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
6. **CRITICAL**: Write tests using MOCKED repositories (never use real database in CLI tests)
   - Create `my_command_test.go` with mock repository
   - Test command logic, argument parsing, output formatting
   - See "Testing Architecture" section below for patterns

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

## Testing Architecture - CRITICAL PATTERNS

### ⚠️ TESTING GOLDEN RULE ⚠️

**ONLY repository tests should use the real database. Everything else MUST use mocked repositories.**

This rule is critical for:
- Test isolation (no data pollution between tests)
- Test speed (in-memory mocks are 100x faster)
- Test reliability (no flaky tests from database state)
- Parallel test execution (no database contention)

### Test Categories

#### 1. Repository Tests (`internal/repository/*_test.go`)
**✅ USE REAL DATABASE - MUST CLEAN UP**

These tests verify database operations (CRUD, transactions, queries).

**Requirements:**
- Create test database using `test.GetTestDB()`
- Clean up test data BEFORE each test (DELETE existing records)
- Use `test.SeedTestData()` for consistent fixtures
- Never rely on data from previous tests
- Verify database constraints, triggers, indexes

**Example:**
```go
func TestTaskRepository_Create(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewTaskRepository(db)

    // CRITICAL: Clean up existing data first
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

    // Seed fresh test data
    epicID, featureID := test.SeedTestData()

    // Run test
    task := &models.Task{...}
    err := repo.Create(ctx, task)
    assert.NoError(t, err)

    // Cleanup at end (optional, but good practice)
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
}
```

#### 2. CLI Command Tests (`internal/cli/commands/*_test.go`)
**❌ NEVER USE REAL DATABASE - USE MOCKS**

These tests verify command logic, argument parsing, and output formatting.

**Requirements:**
- Create mock repository interfaces
- Inject mocks into command handlers
- Test command behavior without database
- Verify JSON/table output formatting
- Test error handling

**Example:**
```go
type MockTaskRepository struct {
    CreateFunc func(ctx context.Context, task *models.Task) error
    GetFunc    func(ctx context.Context, id int64) (*models.Task, error)
}

func TestTaskCreateCommand(t *testing.T) {
    // Create mock
    mockRepo := &MockTaskRepository{
        CreateFunc: func(ctx context.Context, task *models.Task) error {
            task.ID = 123
            return nil
        },
    }

    // Inject mock into command handler
    // Test command execution
    // Verify mock was called correctly
    // Verify output format
}
```

#### 3. Service Layer Tests (`internal/sync/*_test.go`, `internal/status/*_test.go`)
**❌ NEVER USE REAL DATABASE - USE MOCKS**

These tests verify business logic and service orchestration.

**Requirements:**
- Mock all repository dependencies
- Test service logic in isolation
- Verify correct repository method calls
- Test error propagation and handling

#### 4. Unit Tests (models, utils, parsers)
**❌ NO DATABASE - PURE LOGIC**

These test pure functions with no dependencies.

**Requirements:**
- No database, no file system, no network
- Test data transformations, validations, parsing
- Fast, deterministic, parallel-safe

### Test Organization

```
internal/
├── repository/
│   ├── task_repository.go
│   └── task_repository_test.go       # ✅ Uses real DB + cleanup
├── cli/commands/
│   ├── task.go
│   ├── task_test.go                  # ❌ Uses mocks only
│   └── mock_task_repository.go       # Mock interface
├── sync/
│   ├── sync.go
│   └── sync_test.go                  # ❌ Uses mocks only
└── models/
    ├── task.go
    └── task_test.go                  # ❌ Pure logic only
```

### Common Testing Mistakes

❌ **WRONG: CLI test using real database**
```go
func TestTaskCommand(t *testing.T) {
    database := test.GetTestDB()  // DON'T DO THIS
    // This causes test pollution and flaky tests
}
```

✅ **CORRECT: CLI test using mock**
```go
func TestTaskCommand(t *testing.T) {
    mockRepo := &MockTaskRepository{...}
    // Test command logic in isolation
}
```

❌ **WRONG: Repository test without cleanup**
```go
func TestCreate(t *testing.T) {
    database := test.GetTestDB()
    repo := NewTaskRepository(db)
    repo.Create(ctx, task)  // Leaves data in DB
    // Next test will see this data!
}
```

✅ **CORRECT: Repository test with cleanup**
```go
func TestCreate(t *testing.T) {
    database := test.GetTestDB()
    repo := NewTaskRepository(db)

    // Clean first
    database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", task.Key)

    repo.Create(ctx, task)

    // Verify and cleanup
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
}
```

### Running Tests

```bash
make test              # Full suite
make test-coverage     # With coverage HTML report
go test -v ./...       # Manual verbose run
go test -run TestName  # Specific test

# Run only repository tests (with DB)
go test -v ./internal/repository

# Run only CLI tests (no DB, fast)
go test -v ./internal/cli/commands
```

### Test Database

- Location: `internal/repository/test-shark-tasks.db`
- Created by: `internal/test/testdb.go`
- Shared across repository tests (fast, avoids recreation)
- MUST be cleaned before each test to avoid pollution

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
- **CRITICAL**: Only repository tests use real database; everything else uses mocks (see "Testing Architecture" section)
- Use interfaces for mocking dependencies, a common Go practice.
- Focus on testing business logic and public APIs.
- Organize related checks within a single test function using `t.Run` for sub-tests.
- Repository tests MUST clean up data before each test to ensure isolation

---

## Version & Release Notes

Check recent commits and `README.md` for release history. The project uses semantic versioning and is actively maintained with frequent enhancements (E01-E07 feature sets completed).

