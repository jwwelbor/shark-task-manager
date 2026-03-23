# Program Structure

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 3 — Code Reference

## Entry Points (`cmd/`)

### Primary Binaries

| Binary | Entry Point | Purpose |
|--------|------------|---------|
| **shark** | `cmd/shark/main.go` | Main CLI tool — imports all commands via side-effect, sets build version, executes root Cobra command |
| **shark-task-manager** | `cmd/server/main.go` | HTTP API server — minimal implementation with health endpoints |
| **demo** | `cmd/demo/main.go` | Interactive demo — creates sample data, demonstrates lifecycle |
| **test-db** | `cmd/test-db/main.go` | Integration tests — validates CRUD, status updates, cascades |

### Utility Binaries (not built by default)

| Binary | Purpose |
|--------|---------|
| `cmd/backfill-slugs/` | One-time slug migration |
| `cmd/migrate/` | Database migration runner |
| `cmd/migrate-exec-order/` | Add execution_order column |
| `cmd/cleanup/` | Delete epic with cascade |
| `cmd/create-epic/` | Direct epic creation |
| `cmd/test-backfill/` | Test slug backfill |
| `cmd/testmig/` | Test schema migrations |

## Core Packages (`internal/`)

### CLI Framework (`internal/cli/`)

| File | Purpose | Complexity |
|------|---------|-----------|
| `root.go` | Root Cobra command, global config struct, lifecycle hooks, project root detection | High |
| `db_global.go` | Thread-safe lazy DB singleton (`sync.Once`), auto-cleanup via PostRun | Medium |
| `services_global.go` | Global service accessor functions (`GetTaskService()`, `GetEntityService()`) | Medium |
| `output.go` | Output helpers: `OutputJSON()`, `OutputTable()`, `Success()`, `Error()` | Low |
| `field_extractor.go` | `--field` flag implementation for extracting single JSON fields | Low |

### CLI Commands (`internal/cli/commands/`)

| File | Commands | Description |
|------|----------|-------------|
| `get.go` | `shark get <key>` | Auto-detect entity type and display details |
| `list.go` | `shark list [epic] [feature]` | Smart entity listing with filtering |
| `create.go` | `shark create <type>` | Create epic/feature/task |
| `update.go` | `shark update <key>` | Update entity fields |
| `delete.go` | `shark delete <key>` | Delete entity |
| `view.go` | `shark view <key>` | View entity markdown file |
| `status.go` | `shark status [key]` | Dashboard, set, advance, options, history |
| `search.go` | `shark search <query>` | Cross-entity search |
| `task.go` | `shark task <subcommand>` | 19 task subcommands |
| `feature.go` | `shark feature <subcommand>` | 13 feature subcommands |
| `epic.go` | `shark epic <subcommand>` | 14 epic subcommands |
| `idea.go` | `shark idea <subcommand>` | 6 idea subcommands |
| `bug.go` | `shark bug <subcommand>` | 10 bug subcommands |
| `change.go` | `shark change <subcommand>` | 10 change-card subcommands |
| `analytics.go` | `shark analytics [key]` | Project analytics |
| `progress.go` | `shark progress <key>` | Progress breakdown |
| `notes.go` | `shark notes <key>` | Entity notes |
| `context.go` | `shark context <subcommand>` | Entity context management |
| `history.go` | `shark history <key>` | Entity change history |
| `related_docs.go` | `shark related-docs` | Document management |
| `admin.go` | `shark admin <subcommand>` | Init, config, cloud, migrate |

### Service Layer (`internal/services/`)

| File | Service | Key Responsibility |
|------|---------|-------------------|
| `task_service.go` | TaskService | Task CRUD, status transitions, dependency validation |
| `task_dto.go` | - | CreateTaskInput, TaskUpdates, TaskFilters DTOs |
| `feature_service.go` | FeatureService | Feature CRUD, completion logic, cascading |
| `feature_dto.go` | - | CreateFeatureInput, FeatureFilters DTOs |
| `feature_progress_service.go` | FeatureProgressService | Progress, health, work breakdown, action items |
| `epic_service.go` | EpicService | Epic CRUD, rollups, impediments, cascading |
| `epic_dto.go` | - | CreateEpicInput, EpicFilters DTOs |
| `entity_service.go` | EntityService | Polymorphic status transitions, history recording |
| `entity_relationship_service.go` | EntityRelationshipService | Dependency management, cycle detection |
| `entity_history_service.go` | EntityHistoryService | History queries, change tracking |
| `bug_service.go` | BugService | Bug CRUD, triage |
| `change_card_service.go` | ChangeCardService | Change card CRUD, approval |
| `note_service.go` | NoteService | Entity notes management |
| `context_service.go` | ContextService | Entity context retrieval |
| `resume_service.go` | ResumeService | Resume work with full context |
| `transition_types.go` | - | TransitionOptions, TransitionResult types |

### Repository Layer (`internal/repository/`)

| File | Repository | Tables |
|------|-----------|--------|
| `task_repository.go` | TaskRepository | tasks |
| `feature_repository.go` | FeatureRepository | features |
| `epic_repository.go` | EpicRepository | epics |
| `bug_repository.go` | BugRepository | bugs |
| `change_card_repository.go` | ChangeCardRepository | change_cards |
| `entity_note_repository.go` | EntityNoteRepository | entity_notes, task_notes |
| `entity_history_repository.go` | EntityHistoryRepository | entity_history |
| `task_history_repository.go` | TaskHistoryRepository | task_history |
| `work_session_repository.go` | WorkSessionRepository | work_sessions |
| `entity_relationship_repository.go` | EntityRelationshipRepository | entity_relationships, task_relationships |
| `entity_document_repository.go` | EntityDocumentRepository | entity_documents, documents |
| `idea_repository.go` | IdeaRepository | ideas |

### Domain Models (`internal/models/`)

| File | Types | Description |
|------|-------|-------------|
| `entity.go` | Entity interface, EntityType | Common entity interface |
| `epic.go` | Epic | Top-level work unit |
| `feature.go` | Feature | Feature within epic |
| `task.go` | Task, TaskStatus | Atomic work item |
| `bug.go` | Bug | Bug tracking entity |
| `change_card.go` | ChangeCard | Change request entity |
| `task_history.go` | TaskHistory | Status change record |
| `entity_note.go` | EntityNote | Entity annotation |
| `idea.go` | Idea | Pre-promotion capture |
| `entity_relationship.go` | EntityRelationship | Cross-entity links |
| `validation.go` | Validation functions | Structural validation (no workflow) |

### Support Packages

| Package | Files | Purpose |
|---------|-------|---------|
| `config/` | 27 | Configuration loading, workflow profiles, action routing |
| `db/` | 27 | Schema creation, migrations (v10), SQLite PRAGMAs, Turso support |
| `status/` | 20 | Progress calculation, health indicators, work breakdown |
| `workflow/` | 6 | Status transition validation, multi-level profiles |
| `discovery/` | 15 | Filesystem scanning for entity hierarchy |
| `patterns/` | 16 | Entity key regex patterns, file path patterns |
| `taskcreation/` | 7 | Task key generation, file creation |
| `templates/` | 9 | Go template rendering |
| `validation/` | 8 | Input validation utilities |
| `reporting/` | 6 | Report generation |
| `keygen/` | 7 | ID and slug generation |
| `parser/` | 6 | Markdown frontmatter parsing |
| `keys/` | 4 | Entity key parsing and normalization |
| `formatters/` | 7 | JSON, table, text formatting |
| `fileops/` | ~4 | Atomic file operations (O_EXCL) |
| `pathresolver/` | ~3 | Project root auto-detection |
| `slug/` | ~3 | Slug generation from titles |
| `runner/` | 12 | Task execution/orchestration |
| `sync/` | ~5 | File-database synchronization |
| `init/` | 19 | Project initialization, profile setup |

See also: [Interfaces](interfaces.md) | [Data Models](data-models.md) | [API Reference](api-reference.md)
=======
> Generated: 2026-03-20
> Phase: 3 — Code Reference

## Entry Points

### `cmd/shark/main.go` (60 LOC)
Primary CLI binary. Sets version info, executes Cobra root command, handles top-level error routing with exit codes.

### `cmd/server/main.go`
HTTP API server. Minimal implementation — health checks and schema validation.

### `cmd/demo/main.go`
Interactive demo with sample data creation.

## Core Packages

### `internal/cli/` — CLI Framework (82 prod files)

| File | Purpose |
|------|---------|
| `root.go` | Root command, global flags, PersistentPreRunE/PostRunE lifecycle |
| `services_global.go` | Service accessor functions (15+ accessors with sync.Once) |
| `service_accessors.go` | Additional service accessor utilities |
| `db_global.go` | Database singleton management |
| `workflow_global.go` | Workflow service singleton |
| `output.go` | JSON/table/text output formatting |
| `config.go` | Global config struct |

### `internal/cli/commands/` — Command Handlers (68 files)

**Core commands** (auto-detect entity type):
`get.go`, `list.go`, `create.go`, `delete_dispatch.go`, `update_dispatch.go`, `view.go`

**Status & workflow**:
`status.go`, `status_group.go`, `status_priority.go`, `history.go`, `workflow.go`, `workflow_show_actions.go`, `workflow_validate_actions.go`

**Task commands**:
`task.go`, `task_context.go`, `task_criteria.go`, `task_deps.go`, `task_helpers.go`, `task_history.go`, `task_link.go`, `task_next_status.go`, `task_note.go`, `task_resume.go`, `task_sessions.go`, `task_unlink.go`

**Feature commands**:
`feature.go`, `feature_context.go`, `feature_criteria.go`, `feature_helpers.go`, `feature_next_status.go`, `feature_note.go`, `feature_resume.go`, `feature_set_status.go`

**Epic commands**:
`epic.go`, `epic_context.go`, `epic_helpers.go`, `epic_next_status.go`, `epic_note.go`, `epic_resume.go`, `epic_set_status.go`

**Other entity commands**:
`bug.go`, `change.go`, `change_card_commands.go`, `idea.go`

**Discovery & search**:
`search.go`, `notes_search.go`, `related_docs.go`

**Admin & setup**:
`admin.go`, `init.go`, `config.go`, `cloud.go`, `migrate.go`, `migrate_backfill_slugs.go`, `validate.go`

**Shared**:
`commands.go`, `helpers.go`, `errors.go`, `shared_flags.go`, `validators.go`, `validators_reason_doc.go`, `render_common.go`, `orchestrator_display.go`, `file_assignment.go`

**Mocks** (for testing):
`mock_document_repository.go`, `mock_idea_repository.go`, `mock_task_repository.go`

**Registration**:
`zzz_manage_register.go` — Ensures manage commands are registered last

### `internal/services/` — Business Logic (38 prod files)

| File | Purpose |
|------|---------|
| `entity_service.go` | Shared status transition logic |
| `entity_registry.go` | Polymorphic entity access |
| `entity_registry_test.go` | Registry tests |
| `task_service.go` | Task lifecycle operations |
| `feature_service.go` | Feature lifecycle operations |
| `epic_service.go` | Epic lifecycle operations |
| `bug_service.go` | Bug tracking operations |
| `change_card_service.go` | Change card operations |
| `note_service.go` | Entity note management |
| `context_service.go` | Context aggregation |
| `resume_service.go` | Resume context assembly |
| `entity_document_service.go` | Document management |
| `task_dependency_service.go` | Dependency graph operations |
| `display_service.go` | Display data preparation |
| `epic_analytics_service.go` | Epic analytics calculation |
| `transition_types.go` | Status transition types |
| `backward_transition_test.go` | Backward transition tests |

### `internal/repository/` — Data Access (22 prod files)

| File | Purpose |
|------|---------|
| `db.go` | DB connection wrapper |
| `task_repository.go` | Task CRUD + queries (1,806 LOC) |
| `feature_repository.go` | Feature CRUD + queries |
| `epic_repository.go` | Epic CRUD + queries |
| `bug_repository.go` | Bug CRUD |
| `change_card_repository.go` | Change card CRUD |
| `entity_note_repository.go` | Polymorphic notes |
| `task_history_repository.go` | Status change history |
| `document_repository.go` | Related documents |
| `idea_repository.go` | Idea management |
| `task_relationship_repository.go` | Task relationships |

### `internal/models/` — Domain Types (20 prod files)

| File | Entity/Type |
|------|-------------|
| `entity.go` | Entity interface (polymorphic) |
| `epic.go` | Epic struct + methods |
| `feature.go` | Feature struct + methods |
| `task.go` | Task struct + methods |
| `bug.go` | Bug struct |
| `change_card.go` | ChangeCard struct |
| `idea.go` | Idea struct |
| `entity_note.go` | EntityNote struct |
| `task_note.go` | TaskNote struct |
| `task_history.go` | TaskHistory struct |
| `task_criteria.go` | TaskCriteria struct |
| `document.go` | Document struct |
| `work_session.go` | WorkSession struct |
| `completion_metadata.go` | CompletionMetadata struct |
| `context_data.go` | ContextData struct |
| `status_update.go` | StatusUpdate struct |
| `epic_relationship.go` | EpicRelationship struct |
| `feature_relationship.go` | FeatureRelationship struct |
| `task_relationship.go` | TaskRelationship struct |
| `validation.go` | Validation rules |

### `internal/db/` — Database (9 prod files)

| File | Purpose |
|------|---------|
| `db.go` | Schema creation, SQLite config, versioning (2,335 LOC) |
| `migrate.go` | Migration functions (1,560 LOC) |
| `migrate_slug_backfill.go` | Slug backfill migration (628 LOC) |

### `internal/workflow/` — Workflow Engine (3 prod files)

| File | Purpose |
|------|---------|
| `service.go` | Workflow validation service |
| `levels.go` | Entity level constants |
| `types.go` | Status and transition types |

---

See also: [Interfaces](interfaces.md) | [Data Models](data-models.md) | [Components](../architecture/components.md)
>>>>>>> Stashed changes
