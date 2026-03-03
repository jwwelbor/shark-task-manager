# Architecture Overview

**Project**: Shark Task Manager
**Generated**: 2026-03-02

## System Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                     Entry Points                               │
│                                                                │
│   cmd/shark/main.go          cmd/server/main.go                │
│   (CLI Binary)               (HTTP API Server)                 │
└────────────┬─────────────────────────┬─────────────────────────┘
             │                         │
┌────────────▼─────────────────────────▼─────────────────────────┐
│                  Presentation Layer                            │
│                                                                │
│   internal/cli/commands/     cmd/server/handlers/              │
│   - Parse args/flags         - Parse HTTP requests             │
│   - Call service methods     - Call service methods            │
│   - Format output            - Format JSON responses           │
│     (JSON/table)               (status codes)                  │
└────────────────────────┬───────────────────────────────────────┘
                         │
┌────────────────────────▼───────────────────────────────────────┐
│                  Service Layer                                 │
│                  internal/services/                            │
│                                                                │
│   TaskService          FeatureService       EpicService        │
│   - Status transitions - Progress rollups   - Feature rollups  │
│   - Dependency checks  - Task summaries     - Impediments      │
│   - Blocking/unblock   - Health indicators  - Analytics        │
│                                                                │
│   NoteService          ContextService       ResumeService      │
│   - Entity notes       - Context fields     - Resume context   │
│                                                                │
│   ┌───────── ────┐  ┌──────────────────┐  ┌─────────────────┐  │
│   │ workflow.Svc │  │ taskcreation.Svc │  │ status.CalcSvc  │  │
│   │ (transitions)│  │ (key generation) │  │ (progress calc) │  │
│   └────────── ───┘  └──────────────────┘  └─────────────────┘  │
└────────────────────────┬───────────────────────────────────────┘
                         │
┌────────────────────────▼───────────────────────────────────────┐
│                  Repository Layer                              │
│                  internal/repository/                          │
│                                                                │
│   TaskRepository       FeatureRepository    EpicRepository     │
│   TaskHistoryRepo      EntityNoteRepo       DocumentRepo       │
│   TaskRelationshipRepo IdeaRepository                          │
│                                                                │
│   - Pure CRUD operations                                       │
│   - Parameterized SQL queries                                  │
│   - Transaction participation via *sql.Tx                      │
│   - NO business logic                                          │
└────────────────────────┬───────────────────────────────────────┘
                         │
┌────────────────────────▼───────────────────────────────────────┐
│                  Database Layer                                │
│                  internal/db/db.go                             │
│                                                                │
│   ┌──────────────────────┐  ┌────────────────────────────┐     │
│   │ SQLite (Local)       │  │ Turso (Cloud)              │     │
│   │ shark-tasks.db       │  │ libsql://...turso.io       │     │
│   │ WAL mode, FTS5       │  │ WebSocket, token auth      │     │
│   └──────────────────────┘  └────────────────────────────┘     │
└────────────────────────────────────────────────────────────────┘
```

## Layering Rules

| Layer | Owns | Must NOT |
|-------|------|----------|
| **CLI Commands** | Arg parsing, output formatting, exit codes | Contain business logic, call repos, manage transactions |
| **Services** | Business rules, validation, orchestration, transactions | Format output, know about CLI/HTTP |
| **Repositories** | SQL queries, prepared statements, CRUD | Contain business rules, calculate progress, derive status |
| **Database** | Schema, constraints, triggers, migrations | Expose implementation details |

## Entity Hierarchy

```
Epic (E07)
  └── Feature (E07-F01)
        └── Task (E07-F01-001)
```

- **Keys are case-insensitive** and support dual format (numeric + slug)
- **Auto-detection**: Key pattern determines entity type
  - `E##` → Epic
  - `E##-F##` → Feature
  - `E##-F##-###` → Task

## Data Flow

### Command Execution

```
User → CLI Arg Parse → Service Method → Repository Query → Database
                                                              │
Database → Domain Model → Service (apply business rules) → Format Output → User
```

### Status Transition

```
Command: shark status advance E07-F01-001
  │
  ├─ CLI: Parse key, call TaskService.AdvanceStatus()
  │
  ├─ Service: Get task → Validate transition via workflow.Service
  │           → Check dependencies → Update status via repo
  │
  ├─ Repository: UPDATE tasks SET status = ? WHERE id = ?
  │
  └─ Trigger: INSERT INTO task_history (auto via DB trigger)
```

## Dependency Injection

```
CLI Entry Point
  │
  ├─ cli.GetDB() ──→ repository.DB (singleton, thread-safe)
  │
  ├─ cli.GetWorkflowService() ──→ workflow.Service (reads .sharkconfig.json)
  │
  └─ cli.GetTaskService()
       ├─ repository.NewTaskRepository(db)
       ├─ repository.NewEntityNoteRepository(db)
       ├─ workflow.Service
       └─ services.NewTaskService(repo, workflow, creator, noteRepo)
```

- **No DI framework** — pure constructor injection
- **Interface-based** — services depend on repository interfaces, not concrete types
- **Compile-time safe** — constructor signatures enforce contracts
- **CLI**: Global accessors with lazy init (`internal/cli/services_global.go`)
- **HTTP**: Explicit wiring at server startup (`cmd/server/services.go`)

## Workflow Engine

- **Configuration-driven**: Status definitions, transitions, and metadata in `.sharkconfig.json`
- **Two profiles**: Basic (5 statuses) and Advanced (19 statuses)
- **Agent routing**: Advanced profile assigns statuses to agent types (ba, developer, tech_lead, qa, product_owner)
- **Status metadata**: Color, phase, progress weight, responsibility, blocks_feature flag
- **No hardcoded statuses**: All behavior derived from configuration

## File System Integration

```
docs/plan/
  └── {epic-key}/
        ├── epic.md
        └── {feature-key}/
              ├── feature.md
              └── tasks/
                    └── {task-key}.md  (YAML frontmatter + Markdown)
```

- **Sync direction**: Filesystem → Database (unidirectional)
- **Status**: Database-only (never synced from files)
- **Templates**: Rendered via `internal/templates/` using Go `text/template`
- **Atomic writes**: `internal/fileops/` uses `O_EXCL` flag for race protection

## Configuration System

- **File**: `.sharkconfig.json` (project root)
- **Loader**: `internal/config/manager.go`
- **Sections**: database, workflow, status_metadata, templates, viewer
- **Profile Service**: `internal/init/profile_service.go` — applies Basic/Advanced profiles
- **Preservation**: Database config, viewer settings, and custom fields survive profile updates

## Key Architectural Decisions

1. **SQLite as primary store** — Zero-config, embedded, excellent Go CGo driver
2. **Service layer for all business logic** — Reusable across CLI and HTTP API
3. **Constructor DI without framework** — Compile-time safety, explicit dependencies
4. **Configuration-driven workflows** — Status metadata in JSON, not hardcoded
5. **Dual key format** — Human-readable slugs alongside numeric keys
6. **Database as source of truth** — Files are output artifacts, not authoritative
7. **WAL mode** — Concurrent reads during writes for CLI responsiveness
