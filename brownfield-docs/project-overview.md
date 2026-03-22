# Project Overview — Shark Task Manager

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 1 — Discovery & Initialization

## Project Identity

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
