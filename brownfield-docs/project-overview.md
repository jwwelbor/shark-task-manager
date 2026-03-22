# Project Overview

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22T12:00:00Z
> Phase: 1 — Discovery & Initialization

## Project Identity

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
