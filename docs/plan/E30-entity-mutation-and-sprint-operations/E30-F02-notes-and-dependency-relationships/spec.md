---
feature_key: E30-F02-notes-and-dependency-relationships
epic_key: E30
title: Notes and Dependency Relationships
spec_version: 1.0
last_updated: 2026-05-07
complexity: STANDARD
---

# Spec - E30-F02: Notes and Dependency Relationships

> Scope guard: this feature is the follow-on increment to the viewer mutation surface established by E30-F01. It implements the note and relationship portions of the epic, while keeping status edits and Sprint actions out of scope. See epic PRD Section 3 for scope boundaries and Section 4 for constraints.

---

## 1. Requirements

This feature adds write actions for notes and normalized entity relationships on the same epic, feature, and task surfaces already used by the viewer mutation layer. It does not introduce new workflow semantics, new tables, or a new read model.

### 1.1 Functional Requirements

| Req | Requirement | Testable expectation |
|---|---|---|
| **REQ-F02-001** | Add notes from the viewer | A user can create a note on an epic, feature, or task from the viewer without leaving the current entity context. |
| **REQ-F02-002** | Keep notes visible immediately | After a successful note write, the current entity's notes section refreshes and shows the new note in the same viewer session. |
| **REQ-F02-003** | Add dependency-style relationships | A user can create a directed relationship between two supported entities using the normalized relationship store, with the current entity acting as the source. |
| **REQ-F02-004** | Remove dependency-style relationships | A user can remove an existing directed relationship between two supported entities from the same viewer surface. |
| **REQ-F02-005** | Validate note and relationship payloads | The viewer rejects invalid keys, invalid note types, empty note content, unsupported relationship types, duplicate links, self-links, and cycle-creating dependency links before persistence. |
| **REQ-F02-006** | Preserve canonical relationship writes | All new dependency writes go through `entity_relationships`; the viewer must not write to `tasks.depends_on`. |
| **REQ-F02-007** | Preserve the viewer boundary | The read-only viewer routes, file edit route, and local-only CORS posture remain intact while the new mutation routes are added beside them. |
| **REQ-F02-008** | No schema change | The feature reuses the existing note and relationship tables; no database migration is required. |

### 1.2 Acceptance Criteria

- A maintainer can add a note to an epic, feature, or task from the viewer and see it in the notes section after refresh.
- A maintainer can add a dependency-style relationship from an epic, feature, or task to another supported entity and see the relationship reflected after refresh.
- A maintainer can remove an existing relationship and the current entity view no longer shows that link after refresh.
- A request with a malformed key, invalid note type, empty note content, duplicate relationship, self-link, or cycle returns a 4xx response and does not partially write.
- Existing viewer GET routes and the file edit route continue to behave as before.
- No new tables or migrations are introduced for this feature.

### 1.3 Non-Functional Requirements

| Req | Requirement | Target |
|---|---|---|
| **REQ-NF02-001** | No silent writes | Notes and relationships only change in response to explicit user actions. Opening the viewer must not mutate state. |
| **REQ-NF02-002** | Local-only mutation surface | The new write routes use the same localhost-only CORS wrapper as the existing viewer routes. |
| **REQ-NF02-003** | Fast single-item writes | A single note create or relationship create/delete should complete within 250 ms on a normal local SQLite database. |
| **REQ-NF02-004** | Deterministic validation failures | Invalid input should fail fast with 400 for shape/type problems, 404 for missing entities, and 409 for duplicate or cycle-causing dependency writes. |

### 1.4 Out Of Scope

- Editing or deleting notes.
- Status changes.
- Sprint planning actions.
- Bulk note or bulk relationship operations.
- Arbitrary schema changes or migration work.
- Replacing the viewer read model or file edit path.
- Expanding relationship writes into a new generalized graph editor.

---

## 2. Architecture

### 2.1 Existing Foundation

The feature reuses the note, relationship, and viewer plumbing that already exists:

| Area | Existing code | Why it matters |
|---|---|---|
| Viewer read routes | `internal/api/viewer/handler.go` | Already exposes note reads, related-doc reads, and the entity-focused viewer routes that F02 extends. |
| Viewer mutation routes | `internal/api/viewer/mutation_handler.go` | Already owns the edit and transition surface; F02 extends the same handler with note and relationship endpoints. |
| Viewer mutation façade | `internal/api/viewer/mutation_service.go` | Already isolates viewer writes from the service layer; F02 adds note and relationship methods here rather than pushing logic into HTTP handlers. |
| Note service | `internal/services/note_service.go` | Already provides `AddNote`, `ListNotes`, and `SearchNotes` with key resolution and entity registry lookup. |
| Relationship service | `internal/services/entity_relationship_service.go` | Already provides normalized create/delete operations, cycle detection, and validation against the relationship model. |
| Viewer wire-up | `internal/viewer/server/wire.go` | Already constructs the viewer stack and is the right place to inject the additional note and relationship collaborators. |
| Viewer UI | `internal/viewer/assets/viewer.html` | Already renders notes and dependency sections for the detail panes; F02 adds inline mutation controls and refresh behavior. |
| Local-only CORS | `internal/api/viewer/cors.go` | The write routes must reuse the same localhost-only boundary as the existing viewer routes. |

### 2.2 Component Changes

| File | Change |
|---|---|
| `internal/api/viewer/mutation_handler.go` | Add note-create and relationship-create/delete routes, request parsing, key validation, and error mapping. |
| `internal/api/viewer/mutation_handler_test.go` | Add handler tests for note creation, relationship creation/deletion, validation failures, and route registration. |
| `internal/api/viewer/mutation_service.go` | Add note and relationship methods and the small key-resolution adapter needed to resolve source and target entities. |
| `internal/api/viewer/mutation_service_test.go` | Add tests for note delegation, relationship delegation, duplicate/cycle handling, and entity lookup wiring. |
| `internal/viewer/server/wire.go` | Inject `NoteService`, `EntityRelationshipService`, and the entity lookup adapter into the viewer mutation façade. |
| `internal/viewer/assets/viewer.html` | Add note composer controls and relationship add/remove controls to the epic, feature, and task detail panes. |
| `internal/viewer/assets_test.go` | Update embedded-asset checks so the new viewer actions and fetch helpers are covered. |
| `internal/viewer/server/server_test.go` | Verify that the new routes are mounted, remain local-only, and do not disturb the existing GET surfaces. |

### 2.3 API And Interface Contracts

F02 extends the existing viewer mutation surface with explicit note and relationship endpoints. The current entity key remains the source context for all routes.

#### Note create routes

- `POST /api/v1/viewer/epics/{key}/notes`
- `POST /api/v1/viewer/features/{key}/notes`
- `POST /api/v1/viewer/tasks/{key}/notes`

#### Relationship create routes

- `POST /api/v1/viewer/epics/{key}/relationships`
- `POST /api/v1/viewer/features/{key}/relationships`
- `POST /api/v1/viewer/tasks/{key}/relationships`

#### Relationship delete routes

- `DELETE /api/v1/viewer/epics/{key}/relationships/{relationship_type}/{to_key}`
- `DELETE /api/v1/viewer/features/{key}/relationships/{relationship_type}/{to_key}`
- `DELETE /api/v1/viewer/tasks/{key}/relationships/{relationship_type}/{to_key}`

#### Note request body

| Field | Required | Notes |
|---|---|---|
| `note_type` | Yes | Must satisfy the existing note-type validator. |
| `content` | Yes | Empty or whitespace-only content is rejected. |
| `created_by` | No | Preserved when present; omitted otherwise. |

#### Relationship create request body

| Field | Required | Notes |
|---|---|---|
| `relationship_type` | Yes | Must satisfy the existing relationship validator. |
| `to_key` | Yes | Target entity key; the target type is inferred from the key format. |

#### Response contracts

- Note create returns `models.EntityNote` with HTTP 201 Created.
- Relationship create returns `models.EntityRelationship` with HTTP 201 Created.
- Relationship delete returns HTTP 204 with no body.
- Validation failures return the existing viewer error shape.

### 2.4 Technical Decisions And Rationale

#### 2.4.1 Keep business logic in services

The viewer handlers should only parse input, validate shape, and call the service façade. The note and relationship rules already live in `NoteService` and `EntityRelationshipService`; duplicating them in the HTTP layer would make validation drift likely.

#### 2.4.2 Keep `entity_relationships` canonical for writes

The viewer must write dependency-style links only through `entity_relationships`. The legacy `tasks.depends_on` JSON remains a read-compatible compatibility path, not a second write target.

Rationale:

- The normalized model already supports directed links across supported entity types.
- The relationship repository already performs duplicate detection and cycle-aware deletes/creates.
- A dual-write model would create split-brain dependency state.

#### 2.4.3 Keep note writes add-only in this feature

The existing note service already supports note creation and retrieval, but not edit/delete. F02 should expose only creation from the viewer and rely on the existing notes read surface for visibility.

Rationale:

- It keeps the feature aligned with the current service API.
- It avoids inventing delete semantics before the repository and UI both support them.
- It keeps the increment small enough to ship independently.

#### 2.4.4 Resolve entity keys before calling the relationship service

`EntityRelationshipService` works with entity types and IDs, not raw keys. The viewer mutation façade should therefore resolve the source and target keys using the same entity-key rules already used by the viewer package before calling the relationship service.

Rationale:

- The UI only knows keys, not database IDs.
- The key format rules already exist and are shared across the viewer.
- This keeps the handler thin and keeps ID lookup out of the browser.

#### 2.4.5 Preserve the viewer boundary

The new routes should be mounted beside the existing viewer routes and wrapped with the same local-only CORS middleware. `ViewerService` remains read-only, and `EditService` remains file-only.

### 2.5 Data Model Changes

No new tables are required.

This feature reuses:

- `entity_notes` for note persistence.
- `entity_relationships` for directed relationships.
- `tasks.depends_on` only as a compatibility read path, not as a write target.
- `entity_history` only through the existing service-layer behavior already owned by other features.

No migration is needed for this increment.

### 2.6 UI And Interaction Changes

The viewer should add explicit mutation controls to the existing detail panes rather than introducing a new screen.

Planned UI behavior:

- The notes section gets an inline composer with note type, content, and optional creator.
- The dependency/relationship section gets add and remove controls tied to the current entity key.
- After a successful write, the UI invalidates the relevant cached data and rerenders the current entity panel.
- Validation errors render inline near the control that triggered the request.

### 2.7 Validation And Error Mapping

The handler should map errors to consistent HTTP responses:

| Error class | Response |
|---|---|
| Malformed JSON, missing required fields, invalid keys, invalid note types, invalid relationship types | 400 Bad Request |
| Missing source entity, missing target entity, missing relationship on delete | 404 Not Found |
| Duplicate relationship, self-link, cycle-causing dependency relationship | 409 Conflict |
| Unclassified service failure | 500 Internal Server Error |

### 2.8 Traceability

| Requirement | Primary validation focus |
|---|---|
| REQ-F02-001 | POST note create succeeds on epic, feature, and task routes. |
| REQ-F02-002 | Notes section refreshes after create. |
| REQ-F02-003 | Relationship create succeeds on the same entity routes. |
| REQ-F02-004 | Relationship delete succeeds and removes the link from the current view. |
| REQ-F02-005 | Invalid requests fail with deterministic 4xx responses. |
| REQ-F02-006 | No writes occur through `tasks.depends_on`. |
| REQ-F02-007 | Existing viewer GET routes and edit route remain intact. |
| REQ-F02-008 | No schema or migration files are added. |

---

## 3. Ready Check

This spec is ready for shark advancement.
