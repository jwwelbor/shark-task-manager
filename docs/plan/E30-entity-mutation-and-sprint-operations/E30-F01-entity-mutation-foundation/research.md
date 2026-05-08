# E30-F01 Research Report: Entity Mutation Foundation

**Date**: 2026-05-07  
**Status**: Complete  
**Recommendation**: GO

## Executive Summary

E30-F01 is primarily an exposure and wiring problem, not a greenfield domain build. The codebase already has the core write primitives needed for epic, feature, and task mutation: field update methods on each entity service, a shared status-transition engine with history recording, and a viewer stack that is already wired to the read-only aggregates. The main gap is a viewer mutation surface that can call those services directly without pushing business logic into HTTP handlers.

The main architectural decision for this feature is boundary placement. `ViewerService` must remain read-only, `EditService` must stay file-only, and the new mutation surface should be a thin adapter layer that delegates to existing entity services. That keeps validation, history, orchestrator actions, and workflow rules in the service layer where they already live.

## 1. Existing Implementations Relevant To This Feature

| Area | Existing code | Evidence | Why it matters |
|---|---|---|---|
| Read-only viewer API | `internal/api/viewer/handler.go` | Viewer routes are all GET/OPTIONS; no mutation handlers are registered (`RegisterRoutes`, summary through sprint report). | F01 needs new mutation routes beside the existing read-only viewer routes. |
| Viewer read model | `internal/services/viewer_service.go` | `ViewerService` is explicitly read-only and provides summary, hierarchy, history, notes, related docs, sprint overview, sprint plan, and sprint report. | The new mutation path should stay separate from the read model. |
| File-edit surface | `internal/api/viewer/edit_handler.go`, `internal/services/edit_service.go` | `PUT /api/v1/edit/file` writes arbitrary files with path validation and atomic rename. | Keep file edits separate from entity mutation. |
| Shared status engine | `internal/services/entity_service.go` | `TransitionStatus` handles validation, normalization, backward detection, rejection notes, history recording, and orchestrator action resolution. | This is the correct foundation for workflow-driven status changes. |
| Task write path | `internal/services/task_service.go` | `UpdateTask` mutates task fields; `TransitionStatus` delegates to `EntityService`; `ValidateDependencies` enforces dependency readiness. | Tasks already have the right write semantics for viewer exposure. |
| Feature write path | `internal/services/feature_service.go` | `UpdateFeature` and `TransitionStatus` already exist. | Feature edits can be exposed as a thin wrapper over service methods. |
| Epic write path | `internal/services/epic_service.go` | `UpdateEpic` and `TransitionStatus` already exist. | Epic edits can follow the same pattern as feature and task. |
| Entity history read model | `internal/services/entity_history_service.go` | Reads from `entity_history`; writes come from `EntityService.TransitionStatus`. | Mutation flows must preserve the existing audit trail. |
| Polymorphic relationships | `internal/services/entity_relationship_service.go`, `internal/models/entity_relationship.go` | Normalized relationship CRUD, cycle detection, and dependency-style edge support already exist. | F02 can reuse this without reintroducing task-local dependency write logic. |
| Notes service | `internal/services/note_service.go` | Supports `AddNote`, `ListNotes`, and `SearchNotes` across entity types. | Note creation and listing already exist; write exposure just needs routing. |
| Sprint operations | `internal/services/sprint_service.go` | `UpdateSprint`, `AddEntityToSprint`, `RemoveEntityFromSprint`, `StartSprint`, `CloseSprintWithCarryover`, `SetSprintCapacity`, `GetSprintCapacity`, `GetSprintReadiness`, and `PlanSprint` are already implemented. | F03 can call service methods instead of inventing sprint business rules. |
| Viewer/service wiring | `internal/viewer/server/wire.go` | Wire-up already constructs `EntityRelationshipService`, `SprintService`, `ViewerService`, and `EditService`. | The wiring root is already close to what F01 needs; it just lacks mutation exposure. |
| CLI precedent for writes | `internal/cli/commands/task.go`, `internal/cli/commands/feature_helpers.go`, `internal/cli/commands/epic_helpers.go`, `internal/cli/commands/note_generic.go`, `internal/cli/commands/sprint.go` | CLI already routes entity updates and status transitions to the same service layer. | The backend write semantics are already established and tested. |

## 2. Patterns And Conventions That Must Be Followed

- Keep handlers thin. The existing handlers parse input, validate shape, then call services.
- Preserve service-owned business logic. Workflow validation, history, rejection notes, and orchestrator actions live in the services.
- Keep file editing separate from entity mutation. `EditService` is a distinct write path with different safety constraints.
- Keep the viewer local-only. The current viewer posture is read-only and local; F01 should not widen that boundary.
- Reuse the existing workflow engine for status transitions. `EntityService` already contains the shared logic for all transition semantics.
- Treat polymorphic entity models consistently. `EntityType`, `EntityNote`, `EntityHistory`, and `EntityRelationship` are already the shared vocabulary.

## 3. Integration Points

### Services

- `EntityService` for workflow-driven status transitions, backward-transition handling, history recording, and orchestrator actions.
- `TaskService`, `FeatureService`, and `EpicService` for direct entity field updates.
- `NoteService` for note creation and retrieval, which becomes the basis for F02.
- `EntityRelationshipService` for normalized dependency and cross-entity link writes, which becomes the basis for F02.
- `SprintService` for sprint assignments, removals, planning, and capacity operations, which becomes the basis for F03.
- `ViewerService` stays read-only and continues to compose dashboard data.

### Repositories

- Task repository methods for task updates, dependency validation, and dependent lookup.
- Entity relationship repository behind `EntityRelationshipService`.
- Sprint repository backing sprint CRUD, assignment, capacity, and backlog queries.
- Entity history repository behind `EntityHistoryService`.
- Note repository behind `NoteService`.

### CLI Commands

- `shark task update`, `shark task set-status`, and task dependency commands already route to service methods.
- `shark feature update` and `shark epic update` already route to service methods.
- Generic note add commands already call `NoteService.AddNote`.
- Sprint planning commands already call `SprintService` methods for planning and assignment flows.

### Database Tables And Model Surfaces

- `tasks` via `models.Task` and `TaskRepository`.
- `entity_notes` via `models.EntityNote` and `NoteService`.
- `entity_history` via `models.EntityHistory` and `EntityHistoryService`.
- `entity_relationships` via `models.EntityRelationship` and `EntityRelationshipService`.
- `sprints`, `sprint_assignments`, and `sprint_capacity` via `SprintService` and sprint repositories.

## 4. What Can Be Extended Vs What Needs New Code

### Can Be Extended

- `EntityService` can remain the shared engine for status transitions, history, rejection notes, and orchestrator actions.
- `TaskService`, `FeatureService`, and `EpicService` can be reused for field updates and status changes.
- `NoteService` can be extended for viewer-facing note workflows, even though add/list/search already exist.
- `EntityRelationshipService` is the right extension point for dependency and cross-entity link writes.
- `SprintService` already contains the assignment, removal, close, readiness, and planning operations that a viewer action layer should call.
- `ViewerService` can stay read-only while a separate mutation façade is added beside it.

### Needs New Code

- Viewer API endpoints for entity mutation, note mutation, relationship mutation, and sprint actions.
- A viewer-side mutation orchestration layer that maps UI intent to the existing services.
- If F02 standardizes dependencies on `entity_relationships`, a compatibility or migration step will be needed so legacy task-local dependency data does not drift.
- If note edit/delete is added later, those service and repository methods do not exist yet.
- Viewer UI controls for inline edit, dependency management, note editing, and sprint actions.

## 5. Inter-Feature Dependency Map Within E30

E30 splits cleanly into one foundational feature and two follow-on mutation features:

| Feature | Role | Dependency notes |
|---|---|---|
| `E30-F01` Entity Mutation Foundation | Foundation | Establishes the viewer mutation surface, backend wiring, and shared status-transition plumbing for epic/feature/task edits. |
| `E30-F02` Notes and Dependency Relationships | Follow-on | Reuses the mutation surface established by F01 and extends it with note management plus normalized relationship writes. It should not duplicate mutation plumbing. |
| `E30-F03` Sprint Mutation Actions | Follow-on | Reuses the same viewer mutation surface and extends it with sprint assignment, removal, and planning actions. It should not duplicate mutation plumbing. |

Practical dependency order:

1. F01 first, because it establishes the mutation transport and entity-edit entry points.
2. F02 and F03 can then proceed in parallel, because each is a domain-specific extension over the same mutation surface.
3. F02 and F03 should both consume the same service-layer patterns, but they do not need each other as a hard prerequisite.

## 6. Extension-vs-New Analysis

| Component | Extend / New | Notes |
|---|---|---|
| Viewer read handlers | Extend | Keep existing GET routes unchanged. |
| Viewer mutation handlers | New | F01 needs the first write-facing viewer endpoints. |
| `ViewerService` | Extend nowhere | It should remain read-only. |
| `EditService` | No change | Keep file writes separate. |
| `EntityService` | Reuse | This remains the canonical status-transition engine. |
| `TaskService.UpdateTask` / `FeatureService.UpdateFeature` / `EpicService.UpdateEpic` | Reuse | These are the canonical field-update paths. |
| `TaskService.TransitionStatus` / `FeatureService.TransitionStatus` / `EpicService.TransitionStatus` | Reuse | These are already layered over `EntityService`. |
| `EntityRelationshipService` | Reuse / extend in F02 | It already supports create/delete/query and cycle detection. |
| `NoteService` | Reuse / extend in F02 | Add write exposure only if the viewer needs note creation from UI. |
| `SprintService` | Reuse / extend in F03 | It already owns the sprint workflow and capacity semantics. |
| Viewer UI controls | New | The first user-visible mutation affordance must be added here. |

## 7. Technical Risks And Feasibility Assessment

### Risks

- Boundary risk: the viewer is currently read-only, and `ViewerService` explicitly advertises no mutation methods.
- Workflow regression risk: status transitions must continue to flow through `EntityService`, or history and rejection-note behavior will be lost.
- Dependency split-brain risk: `tasks.depends_on` JSON still exists alongside `entity_relationships`. If both are writable, dependency state can diverge.
- Sprint assignment risk: `AddEntityToSprint` enforces one active assignment per entity and only allows assignments in `planning` or `active` status.
- Capacity semantics risk: sprint capacity warnings are advisory, not blocking. The UI must not present them as hard validation.

### Feasibility

Feasibility is high. The codebase already contains the business logic needed for entity edits, status transitions, relationship management, sprint planning, and history tracking. F01 should primarily expose and compose those services rather than create new business rules.

## 8. Recommended Implementation Approach

1. Make the viewer mutation surface a thin adapter over existing services. Do not move business logic into handlers.
2. Keep `EditService` separate for arbitrary file writes. Do not mix file edit behavior with entity mutation.
3. Use `EntityService` as the only status-transition engine for epic, feature, and task transitions.
4. Prefer `EntityRelationshipService` as the canonical dependency write path in F02. If legacy task-local dependencies remain, define one source of truth and a compatibility path.
5. Add viewer actions in priority order: status changes first, then notes and relationships in F02, then sprint assignment and planning actions in F03.
6. Add regression tests around cycle detection, assignment conflicts, history recording, and backward-transition handling before wiring the UI.

