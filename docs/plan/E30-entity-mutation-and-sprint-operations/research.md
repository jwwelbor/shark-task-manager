# E30 Research Report: Entity Mutation and Sprint Operations

**Epic**: [Entity Mutation and Sprint Operations](./epic.md)  
**Date**: 2026-05-07

## Executive Summary

E30 is mostly an exposure and orchestration problem, not a greenfield backend effort. The codebase already has entity update methods, a shared status-transition engine with history recording, note creation, polymorphic entity relationships, and Sprint CRUD/planning services. The missing piece is a viewer/API mutation surface that reuses those primitives without breaking the current read-only dashboard boundary.

The main technical decision is how to handle dependencies. The codebase still carries task-local `depends_on` JSON on `tasks`, but it also has a normalized `entity_relationships` model and service. Any E30 implementation should choose a canonical write path and avoid split-brain dependency writes.

## 1. Existing Implementations Relevant To This Epic

| Area | Existing code | Evidence | Why it matters |
|---|---|---|---|
| Read-only viewer API | `internal/api/viewer/handler.go` | Routes are read-only viewer endpoints plus sprint overview/plan/report; no entity mutation routes are registered (`34-76`). | E30 will need new mutation endpoints, not just a viewer tweak. |
| File editing surface | `internal/api/viewer/edit_handler.go`, `internal/services/edit_service.go` | `PUT /api/v1/edit/file` writes arbitrary project files with path validation and atomic rename (`27-88`, `10-133`). | This is file I/O only, not entity mutation. It should stay separate. |
| Shared status transition engine | `internal/services/entity_service.go` | `TransitionStatus` performs validation, status update, optional rejection notes, history recording, and orchestrator-action resolution (`153-266`). | This is the correct foundation for editable status changes. |
| Entity history read model | `internal/services/entity_history_service.go` | Reads status history from `entity_history`; history is recorded by `EntityService.TransitionStatus` (`16-54`). | Entity mutation flows must preserve this audit trail. |
| Task mutation path | `internal/services/task_service.go` | `UpdateTask` updates task fields; `TransitionStatus` delegates to `EntityService`; `ValidateDependencies` checks dependency completion (`525-588`, `744-870`). | Task editing already exists; viewer wiring can reuse it. |
| Feature mutation path | `internal/services/feature_service.go` | `UpdateFeature` and `TransitionStatus` already exist (`983-1052`, `274-320`). | Feature edit UI can be thin wrappers over service methods. |
| Epic mutation path | `internal/services/epic_service.go` | `UpdateEpic` and `TransitionStatus` already exist (`563-627`, `223-266`). | Epic edit UI can follow the same service-first pattern. |
| Notes service | `internal/services/note_service.go` | Supports `AddNote`, `ListNotes`, and `SearchNotes` across entity types (`25-112`). | Notes can already be created and read; note editing/deletion would be new. |
| Polymorphic relationships | `internal/models/entity_relationship.go`, `internal/services/entity_relationship_service.go` | The model supports `depends_on`, `blocks`, `related_to`, `follows`, `spawned_from`, `duplicates`, `references`, and `linked_to`; the service creates, deletes, queries, and cycle-checks relationships (`8-105`, `63-355`). | This is the normalized dependency/link layer E30 can extend. |
| Viewer relationship rendering | `internal/services/viewer_service.go` and `internal/viewer/server/wire.go` | Viewer DTOs already include `ViewerTask.Relationships`; the viewer wire already constructs `EntityRelationshipService` for read-only composition (`353-360`, `437-460`). | The viewer can already display relationship data; it just cannot mutate it. |
| Sprint lifecycle and planning | `internal/services/sprint_service.go` | Sprint CRUD, status close, entity assignment/removal, and planning view exist (`150-230`, `473-775`, `1805-1865`). | E30 can reuse sprint behavior instead of inventing new sprint semantics. |
| Sprint viewer payloads | `internal/services/viewer_service.go`, `internal/api/viewer/handler.go` | Read-only `SprintOverviewResponse` and `SprintReportResponse` are already exposed by the viewer (`438-466`, `98-165`). | E30 needs sprint mutations, not new read-side aggregates. |
| CLI mutation precedent | `internal/cli/commands/task.go`, `internal/cli/commands/feature_helpers.go`, `internal/cli/commands/epic_helpers.go`, `internal/cli/commands/note_generic.go` | CLI already calls `UpdateTask`, `UpdateFeature`, `UpdateEpic`, `TransitionStatus`, and `AddNote` (`356-409`, `1164-1200`, `1000-1067`, `104-146`). | The backend write semantics already exist; E30 is mainly about surfacing them in the viewer/API. |

## 2. Patterns And Conventions That Must Be Followed

- Keep handlers thin: parse and validate at the boundary, then call a service. `ViewerHandler` and `EditHandler` follow that pattern already (`internal/api/viewer/handler.go:19-21`, `internal/api/viewer/edit_handler.go:27-29`).
- Preserve service-owned business logic. Status validation, history recording, and rejection-note behavior live in `EntityService`, not in handlers or repositories (`internal/services/entity_service.go:153-266`).
- Reuse the existing workflow engine for status transitions. `EntityService.ValidateAndNormalize`, `DetectBackward`, and `ResolveActionForStatus` are the shared path (`internal/services/entity_service.go:269-323`).
- Keep writes explicit and user-triggered. `EditService.WriteFile` is only invoked after a `PUT` request with a validated body (`internal/services/edit_service.go:28-37`, `internal/api/viewer/edit_handler.go:47-88`).
- Maintain the local-only viewer posture. The viewer server wires the API and embedded dashboard without introducing a separate auth or remote-edit surface (`internal/viewer/server/server.go:57-136`).
- Treat polymorphic entities consistently. `EntityType`, `EntityNote`, `EntityHistory`, and `EntityRelationship` are already the shared model vocabulary (`internal/models/entity_note.go:9-43`, `internal/models/entity_history.go:9-26`, `internal/models/entity_relationship.go:8-105`).
- Do not bypass the canonical update paths. `TaskService`, `FeatureService`, `EpicService`, and `SprintService` already own field-level updates and lifecycle transitions (`internal/services/task_service.go:525-588`, `internal/services/feature_service.go:992-1052`, `internal/services/epic_service.go:572-627`, `internal/services/sprint_service.go:150-230`).

## 3. Integration Points

### Services

- `EntityService` for workflow transitions, backward-transition handling, history recording, and orchestrator actions.
- `TaskService`, `FeatureService`, and `EpicService` for direct entity field updates.
- `NoteService` for note creation and retrieval.
- `EntityRelationshipService` for normalized dependency and cross-entity link management.
- `SprintService` for sprint creation, update, close, assignment, removal, and planning views.
- `ViewerService` for read-only composition of dashboard data.

### Repositories

- Task repository methods for task updates, dependency validation, and dependent lookup (`internal/repository/task/repository.go:844-1185`, `internal/repository/task/dependency.go:22-149`).
- Entity relationship repository via `EntityRelationshipService` and its wire-up in `internal/viewer/server/wire.go:437-460`.
- Sprint repository backing sprint CRUD, assignment, capacity, and backlog queries through `SprintService`.
- Entity history repository behind `EntityHistoryService`.
- Note repository behind `NoteService`.

### CLI Commands

- `shark task update`, `shark task set-status`, and dependency commands already route to service methods (`internal/cli/commands/task.go:356-409`, `internal/cli/commands/task_link.go:92-120`, `internal/cli/commands/task_unlink.go:92-120`).
- `shark feature update` and `shark epic update` already route to entity update services (`internal/cli/commands/feature_helpers.go:1164-1200`, `internal/cli/commands/epic_helpers.go:1011-1067`).
- Generic note add commands already call `NoteService.AddNote` (`internal/cli/commands/note_generic.go:104-146`).
- Sprint planning data is already exposed through viewer-facing DTOs, so new UI actions can be layered on top of existing read models.

### Database Tables And Model Surfaces

- `tasks` via `models.Task` and `TaskRepository`.
- `entity_notes` via `models.EntityNote` and `NoteService`.
- `entity_history` via `models.EntityHistory` and `EntityHistoryService`.
- `entity_relationships` via `models.EntityRelationship` and `EntityRelationshipService`.
- `sprints`, `sprint_assignments`, and `sprint_capacity` via `SprintService` and sprint repositories.

## 4. What Can Be Extended Vs What Needs New Code

### Can Be Extended

- `EntityService` can remain the shared engine for status transitions, history, rejection notes, and orchestrator actions.
- `TaskService`, `FeatureService`, and `EpicService` can be reused for field updates and status changes instead of duplicating write logic.
- `NoteService` can be extended if note edit/delete behavior is required, because add/list/search already exist.
- `EntityRelationshipService` is the right extension point for dependency and cross-entity link writes.
- `SprintService` already contains the assignment, removal, close, and planning operations that a viewer action layer should call.
- `ViewerService` can stay read-only while a separate mutation service or handler layer is added beside it.

### Needs New Code

- Viewer API endpoints for entity mutation, note mutation, relationship mutation, and sprint actions.
- A viewer-side mutation orchestration layer that maps UI intent to the existing services.
- If E30 standardizes dependencies on `entity_relationships`, a migration or adapter layer will be needed so task-local `depends_on` data does not drift from the normalized model.
- If E30 needs note edit/delete, those service methods and repository operations do not exist yet.
- Viewer UI controls for inline edit, dependency management, note editing, and sprint actions.

## 5. Technical Risks And Feasibility Assessment

### Risks

- Dual dependency model risk: `tasks.depends_on` JSON still exists alongside `entity_relationships`. If both are writable, dependency state can diverge.
- Boundary risk: the viewer is currently read-only, and `ViewerService` explicitly advertises no mutation methods (`internal/services/viewer_service.go:484-488`). New writes must not leak into that service directly.
- Workflow regression risk: status transitions must continue to flow through `EntityService`, or history and rejection-note behavior will be lost.
- Sprint assignment risk: `AddEntityToSprint` enforces one active assignment per entity and only allows assignments in `planning` or `active` status (`internal/services/sprint_service.go:624-683`). The UI must reflect that rule clearly.
- Capacity semantics risk: sprint capacity warnings are advisory, not blocking (`internal/services/sprint_service.go:677-732`). The UI should not imply hard enforcement.

### Feasibility

Feasibility is high. The codebase already contains almost all business logic needed for entity edits, relationship management, sprint planning, and history tracking. E30 should primarily expose and compose these existing services rather than create new business rules.

## 6. Recommended Implementation Approach

1. Make the viewer mutation surface a thin adapter over existing services. Do not move business logic into handlers.
2. Keep `EditService` separate for arbitrary file writes. Do not mix file edit behavior with entity mutation behavior.
3. Use `EntityService` as the only status-transition engine for all entity types.
4. Prefer `EntityRelationshipService` as the canonical dependency write path if E30 is meant to unify cross-entity links; if task-local `depends_on` remains, define one source of truth and a compatibility path.
5. Add viewer actions in priority order: status changes, note add/edit/delete if needed, relationship add/remove, then Sprint assignment and planning actions.
6. Add regression tests around cycle detection, assignment conflicts, history recording, and backward-transition handling before wiring the UI.

