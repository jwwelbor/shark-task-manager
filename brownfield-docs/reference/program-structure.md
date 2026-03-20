# Program Structure

> Part of the Shark Task Manager Brownfield Analysis
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
