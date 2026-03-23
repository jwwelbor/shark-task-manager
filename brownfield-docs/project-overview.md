<<<<<<< Updated upstream
# Project Overview

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22T12:00:00Z
=======
# Project Overview — Shark Task Manager

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
>>>>>>> Stashed changes
> Phase: 1 — Discovery & Initialization

## Project Identity

<<<<<<< Updated upstream
| Field | Value |
|-------|-------|
| **Name** | Shark Task Manager |
| **Repository** | `github.com/jwwelbor/shark-task-manager` |
| **Language** | Go 1.23.4+ |
| **Type** | CLI Tool + HTTP API |
| **Domain** | Project & Task Management with AI-driven workflows |
| **License** | Not specified in build files |
| **Database** | SQLite (local) / Turso libSQL (cloud) |

## Purpose

Shark Task Manager is a Go-based CLI tool and HTTP API server for managing hierarchical project work items — **epics**, **features**, and **tasks** — with support for AI-driven development workflows. It provides two workflow profiles (basic 5-status and advanced 19-status) with agent routing, status flow enforcement, and config-driven behavior. The system uses SQLite for local persistence with optional Turso cloud database for multi-machine synchronization.

## Technology Stack

| Category | Technology | Version | Purpose |
|----------|-----------|---------|---------|
| Language | Go | 1.23.4+ | Statically typed, compiled |
| Database (local) | SQLite3 | via go-sqlite3 v1.14.32 | WAL mode, FTS5 enabled |
| Database (cloud) | Turso/libSQL | v0.0.0 (dev commit) | Multi-machine sync |
| CLI Framework | Cobra | v1.10.2 | Command hierarchy, flags |
| Configuration | Viper | v1.21.0 | JSON/YAML config, env vars |
| Terminal UI | pterm | v0.12.82 | Colored output, tables |
| Testing | testify | v1.11.1 | Assertions, mocks |
| Build | Make | - | Build, test, lint, fmt |
| CI/CD | GitHub Actions | 3 workflows | CI, Release, Release-test |
| Release | GoReleaser | v2.x | Cross-platform builds |
| Hot Reload | Air | latest | Dev server auto-rebuild |
| Linting | golangci-lint | v2.9.0 | Static analysis |

## Key Statistics

| Metric | Value |
|--------|-------|
| Total Go files | 1,197 |
| Production Go files | 592 |
| Test Go files | 605 |
| Production LOC | ~135,000 |
| Test LOC | ~257,000 |
| Test-to-code ratio | 1.9:1 |
| Internal packages | 31 |
| CLI entry points | 11 (4 primary, 7 utility) |
| Documentation files | ~1,955 .md |
| CI/CD workflows | 3 |
| Entity templates | 80+ |
| External dependencies | 9 direct, ~100+ transitive |

## Module/Package Inventory

### Primary Entry Points (`cmd/`)

| Binary | Path | Purpose | Built by default |
|--------|------|---------|-----------------|
| shark | `cmd/shark/` | Main CLI tool | Yes |
| shark-task-manager | `cmd/server/` | HTTP API server | Yes |
| demo | `cmd/demo/` | Interactive demo | Yes |
| test-db | `cmd/test-db/` | DB integration tests | Yes |
| backfill-slugs | `cmd/backfill-slugs/` | Slug migration | No |
| migrate | `cmd/migrate/` | DB migration runner | No |
| cleanup | `cmd/cleanup/` | Epic cascade deletion | No |
| create-epic | `cmd/create-epic/` | Direct epic creation | No |
| migrate-exec-order | `cmd/migrate-exec-order/` | Schema migration | No |
| test-backfill | `cmd/test-backfill/` | Slug test | No |
| testmig | `cmd/testmig/` | Migration test | No |

### Core Packages (`internal/`)

| Package | Files | LOC | Category | Purpose |
|---------|-------|-----|----------|---------|
| repository | 77 | 30,133 | Data Access | Epic, Feature, Task, History CRUD |
| services | 62 | 28,560 | Business Logic | Task, Feature, Epic lifecycle |
| config | 27 | 14,408 | Configuration | .sharkconfig.json management |
| db | 27 | 11,225 | Infrastructure | SQLite setup, schema, migrations |
| status | 20 | 7,373 | Business Logic | Progress, health, action items |
| init | 19 | 5,242 | Infrastructure | Project initialization, profiles |
| cli | 19 | 4,918 | Presentation | CLI framework, global config |
| cli/commands | - | - | Presentation | CLI command handlers |
| runner | 12 | 4,603 | Orchestration | Task execution workflows |
| discovery | 15 | 4,418 | Infrastructure | Filesystem scanning |
| patterns | 16 | 4,379 | Shared | File/entity key patterns |
| models | 28 | 3,212 | Domain | Epic, Feature, Task models |
| taskcreation | 7 | 3,062 | Business Logic | Task key generation |
| templates | 9 | 2,902 | Infrastructure | Entity template rendering |
| workflow | 6 | 2,251 | Business Logic | Status transitions, agent routing |
| validation | 8 | 2,189 | Shared | Input validation |
| reporting | 6 | 1,765 | Presentation | Report generation |
| keygen | 7 | 1,719 | Shared | ID/slug generation |
| parser | 6 | 1,688 | Infrastructure | Markdown parsing |
| keys | 4 | 1,658 | Shared | Entity key handling |
| formatters | 7 | 1,627 | Presentation | JSON/table formatting |
| fileops | ~4 | ~500 | Infrastructure | Atomic file operations |
| pathresolver | ~3 | ~300 | Infrastructure | Project root detection |
| slug | ~3 | ~300 | Shared | Slug generation |
| progress | ~2 | ~200 | Business Logic | Progress calculation |
| taskfile | ~3 | ~300 | Infrastructure | Task file read/write |
| test | ~3 | ~300 | Testing | Test utilities |

### Supporting Directories

| Directory | Files | Purpose |
|-----------|-------|---------|
| docs/ | ~1,955 | CLI reference, guides, plans |
| docs/plan/ | many | Epic/feature/task markdown files |
| shark-templates/ | 133 | Status-specific entity templates |
| test-fixtures/ | 15 | Test data and fixtures |
| scripts/ | 10 | Utility shell scripts |
| dev-artifacts/ | 114 | Development workspace artifacts |
| .claude/ | many | AI agent rules and prompts |
| .github/workflows/ | 3 | CI/CD pipeline configs |

## High-Level Architecture

Shark follows a **clean layered architecture** with four primary layers:

```
CLI Commands / HTTP Handlers  (Presentation)
         ↓
    Service Layer             (Business Logic)
         ↓
   Repository Layer           (Data Access)
         ↓
  SQLite / Turso DB           (Persistence)
```

**Key architectural characteristics:**
- **Clean Architecture**: Strict layer separation with dependency injection
- **Dual Entry Points**: CLI (Cobra) and HTTP API (net/http)
- **Service Layer**: All business logic centralized in services
- **Repository Pattern**: Pure CRUD data access, no business logic
- **Config-Driven Workflows**: Status flows, agent routing, and progress weights defined in JSON
- **Dual Database Backend**: Local SQLite with optional Turso cloud sync
- **Entity Hierarchy**: Epics → Features → Tasks (plus Bugs and Change-Cards)
- **Dual Key Format**: Numeric keys (E07-F01-001) and slugged keys (E07-F01-001-task-name)
- **Template System**: 80+ status-specific Go templates for entity files

## Project Classification

- **Type**: Monolith (single deployable binary with optional HTTP server)
- **Maturity**: Active development, pre-1.0
- **Team Size**: Designed for solo to small team use
- **Deployment**: Single binary distribution (no containers)
- **Testing**: Comprehensive (1.9:1 test-to-code ratio)

See also: [Architecture Analysis](architecture/system-overview.md)
=======
| Attribute | Value |
|-----------|-------|
| **Name** | Shark Task Manager |
| **Repository** | `github.com/jwwelbor/shark-task-manager` |
| **Language** | Go 1.23.4 |
| **Type** | Monolith (CLI + HTTP API server) |
| **Domain** | Project/Task Management with AI-driven development workflows |
| **License** | MIT |
| **Primary Binary** | `shark` (CLI tool) |

## Purpose

Shark Task Manager is a Go-based CLI tool and HTTP API for managing hierarchical project entities (Epics, Features, Tasks, Bugs, Change Cards, Ideas) with configurable workflow profiles. It uses SQLite for local persistence (with optional Turso cloud database) and is designed to support AI-driven development workflows where AI agents advance tasks through multi-stage pipelines.

## Technology Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.23.4 | Primary language |
| SQLite (mattn/go-sqlite3) | v1.14.32 | Local database (CGO-based) |
| Turso (libsql-client-go) | v0.0.0-20251219 | Cloud database backend |
| Cobra | v1.10.2 | CLI framework |
| Viper | v1.21.0 | Configuration management |
| pterm | v0.12.82 | Terminal output formatting (tables, colors) |
| testify | v1.11.1 | Test assertions |
| golang.org/x/term | v0.32.0 | Terminal interaction |
| golang.org/x/text | v0.28.0 | Unicode text processing |
| yaml.v3 | v3.0.1 | YAML parsing |
| GoReleaser | v2 | Multi-platform release builds |
| golangci-lint | v2.9.0 | Static analysis |
| Air | latest | Hot-reload development |

## Key Statistics

| Metric | Count |
|--------|-------|
| **Total Go files** | 1,679 |
| **Production Go files (internal/)** | 266 |
| **Test Go files (internal/)** | 288 |
| **Command entry points (cmd/)** | 12 |
| **Internal packages** | 30 |
| **CLI command files** | 68 |
| **Model files** | 20 |
| **Repository files** | 22 |
| **Service files** | 38 |
| **Production LOC (internal/)** | ~190,694 |
| **Test LOC (internal/)** | ~351,740 |
| **Markdown documentation files** | 5,636 |
| **GitHub Actions workflows** | 3 (CI, release, release-test) |

## Application Entry Points

| Binary | Source | Purpose |
|--------|--------|---------|
| `shark` | `cmd/shark/main.go` | Primary CLI tool — all user-facing commands |
| `shark-task-manager` | `cmd/server/main.go` | HTTP API server |
| `demo` | `cmd/demo/main.go` | Interactive demo with sample data |
| `test-db` | `cmd/test-db/main.go` | Database integration test runner |

Additional utility commands in `cmd/`: `backfill-slugs`, `cleanup`, `create-epic`, `migrate`, `migrate-exec-order`, `test-backfill`, `testmig`.

## Module/Package Inventory

### Core Application Packages

| Package | Path | Prod Files | Test Files | Purpose |
|---------|------|-----------|-----------|---------|
| **cli** | `internal/cli/` | 82 | 100 | CLI framework, commands, output formatting |
| **services** | `internal/services/` | 38 | 22 | Business logic layer (fat services pattern) |
| **repository** | `internal/repository/` | 22 | 50 | Data access layer (pure CRUD) |
| **models** | `internal/models/` | 20 | 6 | Domain entities (Epic, Feature, Task, Bug, etc.) |
| **db** | `internal/db/` | 9 | 17 | Database initialization, schema, migrations |
| **config** | `internal/config/` | 13 | 14 | Configuration management |
| **workflow** | `internal/workflow/` | 3 | 3 | Workflow profiles and status transition validation |
| **status** | `internal/status/` | 11 | 9 | Config-driven status calculations and displays |

### Supporting Packages

| Package | Path | Prod Files | Test Files | Purpose |
|---------|------|-----------|-----------|---------|
| **init** | `internal/init/` | 11 | 8 | Project initialization, workflow profiles |
| **discovery** | `internal/discovery/` | 8 | 7 | Filesystem epic/feature/task discovery |
| **patterns** | `internal/patterns/` | 8 | 8 | File pattern matching and validation |
| **templates** | `internal/templates/` | 3 | 6 | Template rendering for entity files |
| **validation** | `internal/validation/` | 5 | 3 | Entity validation rules |
| **formatters** | `internal/formatters/` | 4 | 3 | Output formatting (JSON, table) |
| **taskcreation** | `internal/taskcreation/` | 3 | 4 | Task key generation and creation |
| **taskfile** | `internal/taskfile/` | 3 | 4 | Markdown task file parsing/writing |
| **reporting** | `internal/reporting/` | 3 | 3 | Report generation |
| **keygen** | `internal/keygen/` | 3 | 4 | Key generation utilities |
| **utils** | `internal/utils/` | 3 | 3 | Shared utilities |
| **keys** | `internal/keys/` | 2 | 2 | Key format parsing and normalization |
| **fileops** | `internal/fileops/` | 2 | 1 | Atomic file operations |
| **parser** | `internal/parser/` | 2 | 4 | Frontmatter/markdown parsing |
| **slug** | `internal/slug/` | 1 | 1 | Slug generation from titles |
| **pathresolver** | `internal/pathresolver/` | 1 | 2 | Project root auto-detection |
| **filepath** | `internal/filepath/` | 1 | 1 | File path utilities |
| **progress** | `internal/progress/` | 1 | 0 | Progress calculation |
| **dependency** | `internal/dependency/` | 1 | 1 | Dependency graph utilities |
| **template** | `internal/template/` | 1 | 1 | Template engine |
| **view** | `internal/view/` | 1 | 1 | Entity file viewer |
| **test** | `internal/test/` | 1 | 0 | Test database utilities |

## High-Level Architecture

The system follows a **layered architecture** with clean separation:

```
┌─────────────────────────────────────────────────────┐
│                  Entry Points                        │
│   CLI (Cobra)          HTTP API Server               │
│   cmd/shark/           cmd/server/                   │
└──────────┬──────────────────┬────────────────────────┘
           │                  │
┌──────────▼──────────────────▼────────────────────────┐
│              Service Layer (internal/services/)       │
│   TaskService  FeatureService  EpicService           │
│   NoteService  ContextService  ResumeService         │
│   BugService   ChangeCardService                     │
└──────────────────────┬───────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────┐
│          Repository Layer (internal/repository/)      │
│   TaskRepo  FeatureRepo  EpicRepo  NoteRepo          │
│   BugRepo   ChangeCardRepo  IdeaRepo  DocumentRepo   │
└──────────────────────┬───────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────┐
│           Database Layer (internal/db/)               │
│   SQLite (local)  │  Turso (cloud)                   │
│   WAL mode, FK, 64MB cache                           │
└──────────────────────────────────────────────────────┘
```

### Data Flow

**CLI Command → Service Layer → Repository → Database**

1. **Command Layer**: Parse arguments/flags, call service, format output (JSON/table)
2. **Service Layer**: All business logic, workflow validation, orchestration
3. **Repository Layer**: Pure CRUD data access
4. **Database Layer**: SQLite with schema migrations

### Entity Hierarchy

```
Epic (E##)
  └── Feature (E##-F##)
        └── Task (E##-F##-###)

Bug (B###)           — standalone, optionally linked
Change-Card (CC-###) — standalone, optionally linked
Idea                 — can be promoted to task/feature
```

### Workflow Profiles

Two configurable profiles:
- **Basic** (5 statuses): `todo → in_progress → ready_for_review → completed` (+ `blocked`)
- **Advanced** (19 statuses): Multi-stage TDD workflow with agent routing (BA, developer, tech_lead, QA, product_owner)

## Build & CI/CD

- **Build**: `make build` (Go with CGO for SQLite FTS5), `make shark` (CLI only)
- **Test**: `make test` (sequential, `-p=1` for database isolation)
- **Lint**: `make lint` (golangci-lint v2.9.0)
- **CI**: GitHub Actions — 3 workflows (CI on push, release, release-test)
- **Release**: GoReleaser v2 — multi-platform (linux/darwin/windows, amd64/arm64)
- **Cross-platform**: CGO enabled for Linux; CGO disabled for macOS/Windows (no Apple SDK)

## Configuration

- **Primary config**: `.sharkconfig.json` (workflow profiles, database backend, status metadata)
- **Project root detection**: Auto-walks up directories looking for `.sharkconfig.json`, `shark-tasks.db`, or `.git/`
- **Templates**: Embedded Go templates via `//go:embed` (`shark-templates/` directory)
- **Database backend**: Environment-variable driven (`$SHARK_DB_BACKEND`, `$SHARK_DB_URL`)

## Notable Architectural Features

1. **Dual key format**: Both numeric (`E07-F01-001`) and slugged (`E07-F01-001-task-name`) keys, case-insensitive
2. **Polymorphic entity system** (E21): Entity interface foundation for cross-cutting operations
3. **Config-driven workflows**: Status flows, agent routing, and metadata all defined in `.sharkconfig.json`
4. **File-database sync**: Markdown task files in `docs/plan/` synced with SQLite
5. **Embedded templates**: Go `embed.FS` for portable template distribution

## Active Development

The project is under active development with extensive planning documentation in `docs/plan/`. Current epics include:
- **E21**: Entity polymorphism and duplication reduction
- **E22**: External orchestration runner (shark run subcommand)

Development uses shark itself for task tracking (dogfooding), with `dev-artifacts/` for debugging sessions and prototypes.

---

See also: [Architecture](architecture/system-overview.md) | [Code Reference](reference/program-structure.md) | [Technical Debt](technical-debt/summary.md)
>>>>>>> Stashed changes
