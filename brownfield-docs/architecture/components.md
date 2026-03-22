# Components

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 2 — Architecture Analysis

## Component Inventory

### 1. CLI Framework (`internal/cli/`)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Command-line interface framework and entry point orchestration |
| **Files** | 82 prod, 100 test |
| **Technology** | Go + Cobra + pterm |
| **Key Entry Points** | `root.go`, `services_global.go`, `db_global.go` |

**Responsibilities:**
- Root command setup with global flags (`--json`, `--field`, `--no-color`, `--verbose`)
- 68 command files organized by entity type and function
- Global service accessor functions (lazy initialization via `sync.Once`)
- Database singleton management with automatic cleanup
- Output formatting (JSON, tables, success/error/warning messages)
- Command grouping: Inspect, Manage, Advanced

**Key Interfaces Exposed:**
- `GetTaskService()`, `GetFeatureService()`, `GetEpicService()` — Service accessors
- `GetDB(ctx)` — Database connection
- `GetWorkflowService()` — Workflow validation
- `OutputJSON()`, `OutputTable()`, `Success()`, `Error()` — Output helpers

**Dependencies:**
- `internal/services/` — All service types
- `internal/repository/` — DB type for wiring
- `internal/workflow/` — Workflow service
- `internal/config/` — Configuration loading
- `github.com/spf13/cobra` — CLI framework
- `github.com/pterm/pterm` — Terminal output

---

### 2. Service Layer (`internal/services/`)

| Attribute | Value |
|-----------|-------|
| **Purpose** | All business logic, validation, orchestration |
| **Files** | 38 prod, 22 test |
| **Technology** | Go (pure business logic, no framework dependencies) |
| **Key Entry Points** | Service constructors (`NewTaskService`, etc.) |

**Services:**

| Service | File | Purpose |
|---------|------|---------|
| `EntityService` | `entity_service.go` | Shared status transition logic for all entity types |
| `TaskService` | `task_service.go` | Task CRUD, status transitions, dependency validation |
| `FeatureService` | `feature_service.go` | Feature lifecycle, progress rollup |
| `EpicService` | `epic_service.go` | Epic lifecycle, feature rollups |
| `BugService` | `bug_service.go` | Bug tracking and triage |
| `ChangeCardService` | `change_card_service.go` | Change management |
| `NoteService` | `note_service.go` | Entity notes and rejection notes |
| `ContextService` | `context_service.go` | Entity context aggregation |
| `ResumeService` | `resume_service.go` | Resume work with full context |
| `EntityDocumentService` | `entity_document_service.go` | Related document management |
| `EntityRegistry` | `entity_registry.go` | Polymorphic entity access via adapters |
| `TaskDependencyService` | `task_dependency_service.go` | Dependency graph operations |
| `DisplayService` | `display_service.go` | Display mode (planning vs. aggregation) and data preparation |
| `EpicAnalyticsService` | `epic_analytics_service.go` | Epic-level analytics calculations |

**Dependencies:**
- Repository interfaces (defined in service package, implemented by `internal/repository/`)
- `internal/workflow/` — Workflow validation
- `internal/models/` — Domain types
- `internal/taskcreation/` — Task key generation
- `internal/config/` — Configuration types

---

### 3. Repository Layer (`internal/repository/`)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Pure data access (CRUD operations) |
| **Files** | 22 prod, 50 test |
| **Technology** | Go + `database/sql` |
| **Key Entry Points** | Repository constructors |

**Repositories:**

| Repository | File | Entity |
|------------|------|--------|
| `TaskRepository` | `task_repository.go` | Tasks |
| `FeatureRepository` | `feature_repository.go` | Features |
| `EpicRepository` | `epic_repository.go` | Epics |
| `BugRepository` | `bug_repository.go` | Bugs |
| `ChangeCardRepository` | `change_card_repository.go` | Change cards |
| `EntityNoteRepository` | `entity_note_repository.go` | Entity notes |
| `TaskHistoryRepository` | `task_history_repository.go` | Status history |
| `DocumentRepository` | `document_repository.go` | Related documents |
| `IdeaRepository` | `idea_repository.go` | Ideas |
| `DB` | `db.go` | Connection wrapper |

**Dependencies:**
- `internal/models/` — Domain types
- `internal/db/` — Schema initialization
- `database/sql` — Standard library

---

### 4. Database Layer (`internal/db/`)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Database initialization, schema management, migrations |
| **Files** | 9 prod, 17 test |
| **Technology** | SQLite (mattn/go-sqlite3) + Turso (libsql-client-go) |

**Responsibilities:**
- Schema creation (tables, indexes, triggers, constraints)
- Version-checked migrations (`ApplySchemaIfNeeded`)
- SQLite PRAGMA configuration (WAL, FK, cache, mmap)
- Cloud database support (Turso via libsql)

**Dependencies:**
- `github.com/mattn/go-sqlite3` — SQLite driver
- `github.com/tursodatabase/libsql-client-go` — Turso cloud driver

---

### 5. Models (`internal/models/`)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Domain entity types and structural validation |
| **Files** | 20 prod, 6 test |
| **Technology** | Go structs |

**Entities:** Epic, Feature, Task, Bug, ChangeCard, Idea, EntityNote, TaskHistory, TaskCriteria, Document, WorkSession, CompletionMetadata, ContextData, StatusUpdate

**Key Design:** Entity interface for polymorphic operations (E21 feature).

---

### 6. Workflow Engine (`internal/workflow/`)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Config-driven status transition validation |
| **Files** | 3 prod, 3 test |

**Responsibilities:**
- Status flow validation per entity level (epic/feature/task/bug/change)
- Valid transition enumeration
- Terminal status detection
- Status metadata (color, phase, description, agent types)
- Initial status resolution

---

### 7. Status Calculator (`internal/status/`)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Progress calculation, health indicators, work breakdowns |
| **Files** | 11 prod, 9 test |

**Responsibilities:**
- Weighted progress calculation
- Completion progress
- Work remaining by responsibility (agent, human, QA)
- Action items (tasks in actionable statuses)
- Health indicators (healthy/warning/critical)
- Feature/epic impediment tracking

---

### 8. Configuration (`internal/config/`)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Configuration file parsing and template helpers |
| **Files** | 13 prod, 14 test |

**Responsibilities:**
- `.sharkconfig.json` parsing
- Workflow configuration types
- Multi-level workflow support (epic/feature/task)
- Template helper functions
- Status metadata resolution

---

### 9. Supporting Packages

| Package | Files | Purpose |
|---------|-------|---------|
| `init/` | 11 + 8 | Project initialization, workflow profile application |
| `discovery/` | 8 + 7 | Filesystem scanning for entities |
| `patterns/` | 8 + 8 | File pattern matching and validation |
| `templates/` | 3 + 6 | Go template rendering for entity files |
| `validation/` | 5 + 3 | Entity validation rules |
| `formatters/` | 4 + 3 | Output formatting utilities |
| `taskcreation/` | 3 + 4 | Task key generation and file creation |
| `taskfile/` | 3 + 4 | Markdown frontmatter parsing |
| `keygen/` | 3 + 4 | Key generation utilities |
| `keys/` | 2 + 2 | Key format parsing and normalization |
| `fileops/` | 2 + 1 | Atomic file write operations |
| `slug/` | 1 + 1 | Title-to-slug conversion |
| `pathresolver/` | 1 + 2 | Project root auto-detection |

---

See also: [System Overview](system-overview.md) | [Dependencies](dependencies.md) | [Patterns](patterns.md)
