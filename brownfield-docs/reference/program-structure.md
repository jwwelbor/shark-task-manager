# Program Structure

> Part of the Shark Task Manager Brownfield Analysis
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
