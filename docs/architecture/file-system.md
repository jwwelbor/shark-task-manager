# Shark Task Manager - Filesystem Reference

**Generated:** 2026-02-08
**Project Root:** `/home/jwwelbor/projects/shark-task-manager`
**Repository:** `github.com/jwwelbor/shark-task-manager`
**Go Version:** 1.23.4

---

## Top-Level Structure

```
shark-task-manager/
├── bin/                          # Built binaries
│   ├── shark                     # Main CLI tool (24MB)
│   ├── shark-task-manager        # HTTP server (15MB)
│   ├── demo                      # Interactive demo (15MB)
│   └── test-db                   # Database integration tests (15MB)
├── cmd/                          # Application entry points
├── internal/                     # Private application code (31 packages)
├── docs/                         # Documentation (1,047 MD files)
├── .claude/                      # Claude AI rules and context
├── test/                         # E2E and integration tests
├── scripts/                      # Build, migration, verification scripts
├── shark-data/                   # Optional editable content bundle
├── dev-artifacts/                # Development workspace folders
├── .github/workflows/            # CI/CD pipelines
├── shark-tasks.db                # SQLite database (single source of truth)
├── .sharkconfig.json             # Project configuration (14KB)
├── go.mod / go.sum               # Go dependencies
├── Makefile                      # Build automation
├── CLAUDE.md                     # AI agent instructions
├── README.md                     # Project introduction
└── LICENSE                       # MIT License
```

---

## Command Entry Points (`cmd/`)

Each subdirectory contains a `main.go` serving as an application entry point:

```
cmd/
├── shark/main.go                 # Main CLI tool → bin/shark
├── server/main.go                # HTTP API server → bin/shark-task-manager
├── demo/main.go                  # Interactive demo → bin/demo
├── test-db/main.go               # Database integration tests → bin/test-db
├── backfill-slugs/main.go        # Slug migration utility
├── cleanup/main.go               # Database cleanup utility
├── create-epic/main.go           # Epic creation utility
├── migrate/main.go               # Database migration tool
├── migrate-exec-order/main.go    # Execution order migration
└── test-backfill/main.go         # Test slug backfill
```

**Build targets:** `make build` (all), `make shark` (CLI only), `make install-shark` (~`/go/bin`)

---

## Internal Packages (`internal/`)

### Overview (31 packages, ~414 Go source files, ~223 test files)

```
internal/
├── cli/                          # CLI framework
│   ├── commands/                 # Command implementations (117 files)
│   ├── scope/                    # Command scope interpretation
│   ├── root.go                   # Root command, global flags
│   ├── db_global.go              # Global DB singleton
│   └── output.go                 # JSON/table output helpers
│
├── models/                       # Data types (17 files)
├── repository/                   # Data access layer (72 files)
├── db/                           # Database init, schema, drivers (18 files)
├── config/                       # Configuration management (29 files)
├── sync/                         # File-database synchronization (26 files)
├── status/                       # Status calculations (22 files)
├── fileops/                      # Unified file operations (3 files)
├── taskcreation/                 # Task creation logic (6 files)
├── taskfile/                     # Markdown task file parsing (7 files)
├── discovery/                    # Filesystem scanning (12 files)
├── formatters/                   # Output formatting (9 files)
├── validation/                   # Entity/workflow validation (8 files)
├── workflow/                     # Workflow service (3 files)
├── init/                         # Project initialization (17 files)
├── patterns/                     # File pattern matching (16 files)
├── slug/                         # Slug generation (2 files)
├── keys/                         # Key validation (2 files)
├── keygen/                       # Key generation (6 files)
├── pathresolver/                 # Project root detection (3 files)
├── parser/                       # Generic parsing (7 files)
├── templates/                    # Template rendering (3 files)
├── template/                     # Legacy template renderer (2 files)
├── reporting/                    # Report generation (6 files)
├── progress/                     # Progress calculation (1 file)
├── dependency/                   # Dependency detection (2 files)
├── view/                         # View service (2 files)
├── utils/                        # Utilities (6 files)
├── filepath/                     # File path utilities (2 files)
└── test/                         # Test utilities (1 file)
```

---

### CLI Commands (`internal/cli/commands/`)

117 Go files implementing all CLI commands. Key files:

| File | Purpose |
|------|---------|
| `task.go` | Task lifecycle (start, complete, approve, block, unblock) |
| `task_create.go` | Task creation with positional args |
| `task_list.go` | Task listing with filters |
| `task_next.go` | Get next available task for agent |
| `epic.go` | Epic CRUD commands |
| `feature.go` | Feature CRUD commands |
| `get.go` | Smart entity get dispatcher (auto-detects type from key) |
| `list.go` | Smart list dispatcher (auto-detects scope from args) |
| `status.go` | Status and progress display |
| `history.go` | Change history display |
| `sync.go` | File-database synchronization |
| `cloud.go` | Turso cloud commands |
| `workflow.go` | Workflow configuration |
| `init.go` | Project initialization |
| `config.go` | Configuration management |
| `create.go` | Unified create dispatcher |
| `delete.go` | Unified delete dispatcher |

---

### Models (`internal/models/`)

Data structures used throughout the application:

| File | Entity |
|------|--------|
| `task.go` | Task entity |
| `epic.go` | Epic entity |
| `feature.go` | Feature entity |
| `task_history.go` | Status change audit trail |
| `task_note.go` | Task notes / rejection reasons |
| `task_relationship.go` | Dependencies, blockers |
| `idea.go` | Idea capture system |
| `validation.go` | Entity validation rules |

---

### Repository Layer (`internal/repository/`)

72 files (~24.6K lines) implementing all database operations:

| File | Purpose |
|------|---------|
| `task_repository.go` | Task CRUD + atomic status updates |
| `epic_repository.go` | Epic CRUD + progress calculation |
| `feature_repository.go` | Feature CRUD + progress calculation |
| `task_history_repository.go` | Status change history |
| `task_note_repository.go` | Task notes |
| `document_repository.go` | Document management |
| `idea_repository.go` | Idea storage |
| `search_repository.go` | Cross-entity search |
| `work_session_repository.go` | Work session tracking |
| `task_dependency.go` | Dependency management |

---

### Database Layer (`internal/db/`)

Database initialization, schema, and driver abstraction:

| File | Purpose |
|------|---------|
| `db.go` | SQLite/Turso abstraction |
| `database.go` | Database interface |
| `sqlite_driver.go` | Local SQLite implementation |
| `turso_driver.go` | Turso cloud implementation |
| `migrate.go` | Auto-migrations |
| `auth.go` | Token authentication |

---

### Config (`internal/config/`)

Configuration management and workflow profiles:

| File | Purpose |
|------|---------|
| `config.go` | Config structure |
| `manager.go` | Config CRUD operations |
| `workflow_schema.go` | Status metadata schema |
| `workflow_validator.go` | Workflow validation |
| `orchestrator_action.go` | AI agent routing |

---

### Sync Engine (`internal/sync/`)

File-database synchronization:

| File | Purpose |
|------|---------|
| `engine.go` | Sync orchestration |
| `scanner.go` | Filesystem discovery |
| `resolver.go` | Conflict resolution |
| `conflict.go` | Conflict detection |

---

### Status Package (`internal/status/`)

Configuration-driven status calculations:

| File | Purpose |
|------|---------|
| `progress.go` | Weighted + completion progress |
| `work_breakdown.go` | Work categorization by responsibility |
| `action_items.go` | Actionable tasks identification |
| `context.go` | Status context (colors, phases) |

---

## Documentation (`docs/`)

### Structure (1,047 markdown files)

```
docs/
├── CLI_REFERENCE.md              # CLI documentation entry point
├── TURSO_QUICKSTART.md           # Cloud database setup
├── TURSO_MIGRATION.md            # Migration from local to cloud
├── WORKFLOW_GUIDE.md             # Workflow configuration (35KB)
├── troubleshooting.md            # Common issues
│
├── cli-reference/                # Detailed CLI docs (21 files)
│   ├── task-commands.md          # Task lifecycle commands
│   ├── epic-commands.md          # Epic management
│   ├── feature-commands.md       # Feature management
│   ├── global-flags.md           # --json, --no-color, etc.
│   ├── key-formats.md            # Dual key format support
│   ├── json-output.md            # JSON response structures
│   ├── orchestrator-actions.md   # AI agent routing
│   ├── rejection-reasons.md      # Rejection reason tracking
│   ├── workflow-config.md        # Status flow config
│   └── best-practices.md         # AI agent best practices
│
├── guides/                       # How-to guides
│   └── workflow-profiles.md      # Basic vs Advanced workflows
│
├── plan/                         # Epic/feature/task planning docs
│   ├── E04-task-mgmt-cli-core/
│   ├── E05-task-mgmt-cli-capabilities/
│   ├── E06-intelligent-scanning/
│   ├── E07-enhancements/
│   ├── E08-idea-capture-and-conversion-system/
│   ├── E10-advanced-task-intelligence-context-management/
│   ├── E11-configurable-status-workflow-system/
│   ├── E12-bug-tracker-system/
│   ├── E13-workflow-aware-task-command-system/
│   ├── E14-add-cloud-db-support/
│   └── tech-debt/
│
├── adr/                          # Architecture Decision Records
├── api/                          # API documentation
├── architecture/                 # Architecture docs (this file)
├── design/                       # Design documents
├── development/                  # Development guides
├── examples/                     # Example configurations
├── specs/                        # Technical specifications
├── uat/                          # User acceptance testing
└── workflow/                     # Workflow documentation
```

---

## Claude AI Rules (`.claude/`)

Context-aware documentation loaded automatically based on active files:

```
.claude/
├── settings.local.json           # Local Claude settings
│
├── rules/                        # Modular rules
│   ├── quickref.md               # Quick reference (always loaded)
│   ├── database-critical.md      # DB safety warnings (always loaded)
│   ├── development-workflows.md  # Workflows (always loaded)
│   ├── architecture.md           # Architecture (loaded for internal/**)
│   │
│   ├── cli/                      # Loaded for internal/cli/**
│   │   ├── patterns.md
│   │   └── commands.md
│   │
│   ├── database/                 # Loaded for internal/db/**
│   │   ├── schema.md
│   │   └── cloud-turso.md
│   │
│   ├── go/                       # Loaded for *.go files
│   │   ├── patterns.md
│   │   └── error-handling.md
│   │
│   └── testing/                  # Loaded for *_test.go files
│       ├── architecture.md
│       ├── repository-tests.md
│       └── cli-tests.md
│
├── commands/                     # Custom Claude commands
├── hooks/                        # Git hooks
└── skills/                       # Custom skills
```

---

## Shark Data (`shark-data/`)

Optional editable copy of the embedded content bundle:

```
shark-data/
├── workflow/                     # Per-entity workflow YAML
├── prompts/                      # Agent instruction prompts
├── file_templates/               # Default markdown skeletons for created files
├── skills/                       # Skill instructions and references
├── agents/                       # Agent role definitions
└── overrides/                    # Local replacements, preserved by upgrade
    └── file_templates/           # Upgrade-safe custom created-file skeletons
```

---

## Testing Infrastructure

### Test Organization

```
# Repository tests (real database, must clean up)
internal/repository/*_test.go         # 50+ test files

# CLI tests (mocks only, never real database)
internal/cli/commands/*_test.go       # 67 test files

# Unit tests (pure logic)
internal/*_test.go                    # 106+ test files across packages

# E2E/Integration tests
test/
├── e2e/test_enhanced_status.sh
└── integration/
```

### Test Database
- Location: `internal/repository/test-shark-tasks.db`
- Shared across repository tests
- Must be cleaned before each test

### Test Commands
```bash
make test              # All tests
make test-coverage     # HTML coverage report
go test -v ./internal/repository     # Repository tests only
go test -v ./internal/cli/commands   # CLI tests only (fast, no DB)
```

---

## Scripts (`scripts/`)

```
scripts/
├── migrate-to-global-db.sh       # Database migration
├── complete-migration.py         # Python migration helper
├── fix-migration-errors.py       # Fix migration errors
├── test-homebrew.sh              # Homebrew package test
├── test-manual.sh                # Manual install test
├── test-scoop.ps1                # Windows Scoop test
├── verify-release.sh             # Release verification (Linux/macOS)
├── verify-release.ps1            # Release verification (Windows)
└── update_task_repo.py           # Task repository update
```

---

## CI/CD (`.github/workflows/`)

```
.github/workflows/
├── ci.yml                        # Continuous integration
├── release.yml                   # Release automation
└── release-test.yml              # Release testing
```

---

## Configuration Files

| File | Purpose |
|------|---------|
| `.sharkconfig.json` | Project config (status flow, metadata, DB config) |
| `go.mod` / `go.sum` | Go dependencies |
| `Makefile` | Build automation |
| `.air.toml` | Hot reload configuration (dev mode) |
| `.golangci.yml` | Linter configuration |
| `.goreleaser.yml` | Cross-platform release automation |
| `.gitignore` | Git ignore rules |

---

## Database Files

| File | Purpose |
|------|---------|
| `shark-tasks.db` | Main SQLite database (DO NOT DELETE) |
| `shark-tasks.db-shm` | Shared memory file (WAL mode) |
| `shark-tasks.db-wal` | Write-Ahead Log (WAL mode) |
| `shark-tasks.backup` | Database backup |

---

## Key Architectural Patterns

1. **Clean Architecture:** Commands → Repositories → Database
2. **Global DB Singleton:** `internal/cli/db_global.go` (lazy init, auto-cleanup, cloud-aware)
3. **Dependency Injection:** Constructor-based, no DI framework
4. **Dual Key Architecture:** Numeric (`E07-F01-001`) + slugged (`E07-F01-001-implement-auth`), case-insensitive
5. **Configuration-Driven Workflow:** Status metadata, flow, and agent routing in `.sharkconfig.json`
6. **Test Isolation:** Repository tests use real DB; CLI/unit tests use mocks

---

## Where to Add New Features

| What | Where |
|------|-------|
| New CLI command | `internal/cli/commands/` (register via `init()`) |
| New entity/model | `internal/models/` |
| New DB operation | `internal/repository/` |
| New DB migration | `internal/db/migrate.go` |
| New config field | `internal/config/` |
| New output format | `internal/formatters/` |
| New validation | `internal/validation/` |
| Documentation | `docs/` (appropriate subdirectory) |
| Architecture doc | `docs/architecture/` |
| Planning doc | `docs/plan/{epic}/{feature}/` |

---

## File Count Summary

| Category | Count |
|----------|-------|
| Go source files | ~414 |
| Go test files | ~223 |
| Markdown docs | ~1,047 |
| CLI command files | 117 |
| Repository files | 72 |
| Built binaries | 4 |
| Scripts | 10 |
| Templates | 4 |
