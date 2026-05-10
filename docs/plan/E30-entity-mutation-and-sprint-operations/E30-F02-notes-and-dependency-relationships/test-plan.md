---
feature_key: E30-F02-notes-and-dependency-relationships
epic_key: E30
document_type: test-plan
title: Test Plan - Notes and Dependency Relationships
created: 2026-05-07
status: DRAFT
---

# E30-F02 Test Plan

Traceability: every test case below maps to at least one requirement in `./spec.md`, and every requirement in `spec.md` has at least one test case here. Epic UAT references come from `../uat-plan.md`.

Testing rules in force:

- HTTP handler tests use the existing `internal/api/viewer/mutation_handler_test.go` pattern: in-process mux, mocked service façade, no real DB unless the test is explicitly a server integration test.
- Service tests use mocks for the underlying repositories and entity registry.
- Only server integration and latency tests use a real in-memory SQLite DB.
- There are no repository-level tests for this feature because the spec requires no schema change.
- The current `WithLocalCORS` helper is a relevant infrastructure constraint: note and relationship mutation routes need POST and DELETE coverage, so the test plan asserts the new routes are still local-only and that preflight stays correct for the new methods.

---

## 1. Spec Drift Analysis

### 1.1 Drift Findings

No material scope drift identified.

Observations:

- The spec correctly keeps this feature focused on notes and normalized relationship writes. Status edits remain in F02's parent feature and Sprint operations remain in F03.
- The spec uses the canonical `entity_relationships` store for dependency-style writes, which matches the architecture note in the E30 research report.
- The spec says no schema change is required, which is consistent with the feature scope and with the current service/repository layout.
- One infrastructure gap is worth calling out: the current viewer CORS helper in `internal/api/viewer/cors.go` advertises `GET, PUT, OPTIONS`. F02 adds browser-facing `POST` and `DELETE` mutation routes, so the implementation must either extend the helper or add equivalent coverage in the viewer server wiring. The test plan treats this as a route-boundary requirement, not as new product scope.

### 1.2 Traceability Matrix

| Requirement | Test Cases | Covered |
|---|---|---|
| REQ-F02-001 Add notes from the viewer | TC-F02-001 | Yes |
| REQ-F02-002 Keep notes visible immediately | TC-F02-002 | Yes |
| REQ-F02-003 Add dependency-style relationships | TC-F02-003 | Yes |
| REQ-F02-004 Remove dependency-style relationships | TC-F02-004 | Yes |
| REQ-F02-005 Validate note and relationship payloads | TC-F02-005, TC-F02-005b | Yes |
| REQ-F02-006 Preserve canonical relationship writes | TC-F02-003, TC-F02-004, TC-F02-006 | Yes |
| REQ-F02-007 Preserve the viewer boundary | TC-F02-007, TC-F02-008 | Yes |
| REQ-F02-008 No schema change | TC-F02-009 | Yes |
| REQ-NF02-001 No silent writes | TC-F02-002, TC-F02-008 | Yes |
| REQ-NF02-002 Local-only mutation surface | TC-F02-007 | Yes |
| REQ-NF02-003 Fast single-item writes | TC-F02-010 | Yes |
| REQ-NF02-004 Deterministic validation failures | TC-F02-005, TC-F02-005b | Yes |

---

## 2. AC Test Matrix

### 2.1 Functional coverage

| AC / Req | Test Case | Techniques Applied | Caller Path Focus | Input / Setup | Expected Outcome | Location |
|---|---|---|---|---|---|---|
| REQ-F02-001 | TC-F02-001 `TestMutationHandler_AddNote_PerEntity` | Equivalence Partitioning, Contract surface enumeration | `MutationHandler` note-create route -> `MutationService.AddNote` -> `NoteService.AddNote` | POST note payloads for epic, feature, and task; valid `note_type`, content, and optional `created_by` | Each supported entity type returns 201 with an `EntityNote` response and the note is persisted against the source entity key. | `internal/api/viewer/mutation_handler_test.go` + `internal/services/note_service_test.go` |
| REQ-F02-002 | TC-F02-002 `TestViewerNotes_RefreshesAfterCreateAndStaysInContext` | State transitions, Contract surface enumeration | note-create route -> `ViewerHandler.Notes` refresh path -> embedded viewer note composer hook | Create a note, then fetch notes for the same entity in the same server session | The new note is returned by the notes view immediately after create; the user remains on the same entity context. | `internal/api/viewer/mutation_handler_test.go` + `internal/api/viewer/handler_test.go` + `internal/viewer/assets_test.go` |
| REQ-F02-003 | TC-F02-003 `TestMutationHandler_CreateRelationship_PerEntity` | Decision table, Contract surface enumeration | `MutationHandler` relationship-create route -> `MutationService.CreateRelationship` -> `EntityRelationshipService.CreateRelationship` | POST relationship payloads from epic, feature, and task source keys to supported target keys | Each supported entity type returns 201 with an `EntityRelationship` response and the stored edge is sourced from the current entity key. | `internal/api/viewer/mutation_handler_test.go` + `internal/services/entity_relationship_service_test.go` |
| REQ-F02-004 | TC-F02-004 `TestMutationHandler_DeleteRelationship_PerEntity` | State transitions, Contract surface enumeration | `MutationHandler` relationship-delete route -> `MutationService.DeleteRelationship` or `UnlinkEntities` -> `EntityRelationshipService.UnlinkEntities` | DELETE relationship routes for epic, feature, and task source keys with a valid relationship type and target key | Each delete returns 204 and the directed edge is removed from the normalized relationship store. | `internal/api/viewer/mutation_handler_test.go` + `internal/services/entity_relationship_service_test.go` |
| REQ-F02-005 | TC-F02-005 `TestMutationHandler_RejectsMalformedShapeAndUnsupportedTypes` | Attack-class enumeration, Equivalence Partitioning | note/relationship handler boundary | Malformed JSON, malformed source key, invalid note type, empty note content, invalid relationship type | Each request fails with 400 before persistence and no service call is made for the malformed payload. | `internal/api/viewer/mutation_handler_test.go` |
| REQ-F02-005 | TC-F02-005b `TestMutationService_RejectsDuplicateSelfLinkAndCycles` | State transitions, Attack-class enumeration | `MutationService.CreateRelationship` -> `EntityRelationshipService.CreateRelationship` | Duplicate edge, self-link, and cycle-creating dependency link | Each invalid relationship fails deterministically with 409 and no partial write is left behind. | `internal/services/entity_relationship_service_test.go` + `internal/api/viewer/mutation_service_test.go` |
| REQ-F02-006 | TC-F02-006 `TestRelationshipWrites_UseEntityRelationshipsOnly` | Contract surface enumeration | relationship write path under viewer mutation surface | Real in-memory DB seeded with a task row that already has a `depends_on` value | Relationship create/delete writes only the `entity_relationships` model; the legacy `tasks.depends_on` value remains unchanged. | `internal/viewer/server/server_test.go` + `internal/services/entity_relationship_service_test.go` |
| REQ-F02-007 | TC-F02-007 `TestViewerServer_MountsNotesAndRelationshipRoutesWithLocalCORS` | Contract surface enumeration, Boundary enumeration | `viewer/server.StartServer` -> `WireServices` -> `MutationHandler.RegisterRoutes` -> `ViewerHandler.RegisterRoutes` | Start the real viewer server with an in-memory DB and hit existing GET routes plus the new note/relationship routes | Existing GET viewer routes and the file edit route continue to work; the new mutation routes are mounted beside them and remain localhost-only. | `internal/viewer/server/server_test.go` |
| REQ-F02-008 | TC-F02-009 `TestSpec_NoSchemaChange_NoMigrationArtifacts` | Static gate, Internal-only justification | startup path and repo artifact check | Existing schema initialization and project tree inspection | No migration file or schema change is required for this feature; the server still boots against the existing SQLite schema. | `internal/viewer/server/server_test.go` + code review gate |

### 2.2 Validation edge cases per AC

- Note create:
  - Empty body
  - Unknown JSON field
  - Missing `note_type`
  - Empty or whitespace-only content
  - Invalid source key type
  - Missing source entity
- Relationship create:
  - Missing `relationship_type`
  - Unsupported relationship type
  - Invalid source key
  - Invalid target key
  - Missing source entity
  - Missing target entity
  - Self-link
  - Duplicate edge
  - Cycle-causing dependency edge
- Relationship delete:
  - Missing relationship type or target key
  - Relationship not found
  - Invalid source or target key

### 2.3 Non-functional coverage

| NFR / Req | Test Case | Notes |
|---|---|---|
| REQ-NF02-001 | TC-F02-002, TC-F02-008 | Viewer reads, route entry, and fetches must remain side-effect free. |
| REQ-NF02-002 | TC-F02-007 | New note and relationship routes must use the same localhost-only boundary as the rest of the viewer surface. |
| REQ-NF02-003 | TC-F02-010 `TestMutationRoutes_SingleWriteCompletesWithinBudget` | Single note create, relationship create, and relationship delete each complete within 250 ms on a normal local SQLite DB. |
| REQ-NF02-004 | TC-F02-005, TC-F02-005b | Handler-level shape/type failures return 400; missing entity lookups return 404; duplicate/self/cycle writes return 409. |

---

## 3. Caller-Path Contracts

| Test Case | Production Entrypoint | Lowest Allowed Mock Seam | Forbidden Mocks | Counter-Factual |
|---|---|---|---|---|
| TC-F02-001 | `viewer/server.StartServer` -> `MutationHandler.RegisterRoutes` -> note-create handler -> `MutationService.AddNote` -> `NoteService.AddNote` | `NoteEntityNoteRepository` and `EntityRegistry` | Do not mock JSON decoding or the mux; the handler must parse a real request and route by key type | A buggy implementation that creates the note on the wrong entity type or skips note-type validation. |
| TC-F02-002 | `viewer/server.StartServer` -> note-create handler -> `ViewerHandler.Notes` refresh path -> embedded viewer refresh hook | `NoteEntityNoteRepository` plus the viewer read service | Do not mock the refresh sequence away; the test must prove the same entity context shows the new note after save | A buggy implementation that writes the note but leaves the viewer stale until a full reload or a different entity. |
| TC-F02-003 | `viewer/server.StartServer` -> relationship-create handler -> `MutationService.CreateRelationship` -> `EntityRelationshipService.CreateRelationship` | `EntityRelationshipRepository` and entity resolvers | Do not mock cycle detection inside the service; the real validation path must run | A buggy implementation that inserts a link without checking self-link or cycle rules. |
| TC-F02-004 | `viewer/server.StartServer` -> relationship-delete handler -> `MutationService.DeleteRelationship` / `UnlinkEntities` -> `EntityRelationshipService.UnlinkEntities` | `EntityRelationshipRepository` | Do not mock the delete path above the service boundary; the test must exercise the actual directed-edge removal contract | A buggy implementation that returns 204 but leaves the link in the normalized store. |
| TC-F02-005 | `MutationHandler.AddNote` / `CreateRelationship` request parsing boundary | `MutationServicer` methods only for the happy-path cases; failure cases should stop before delegation | Do not mock validation after the handler boundary; malformed payloads must be rejected before the service sees them | A buggy implementation that accepts malformed JSON, invalid note types, empty content, or unsupported relationship types. |
| TC-F02-005b | `MutationService.CreateRelationship` and `EntityRelationshipService.CreateRelationship` | `EntityRelationshipRepository` and the resolver used for key lookup | Do not mock the service-side validation; duplicate/self/cycle protection must come from the production service logic | A buggy implementation that returns 201 for duplicate or cyclic edges. |
| TC-F02-006 | `viewer/server.StartServer` -> relationship write route -> repository state check | Real in-memory DB and the viewer mutation façade | Do not mock the DB or the relationship repository; the test must prove the canonical write target is `entity_relationships` only | A buggy implementation that mutates `tasks.depends_on` or performs dual writes. |
| TC-F02-007 | `viewer/server.StartServer` and `MutationHandler.RegisterRoutes` / `ViewerHandler.RegisterRoutes` | `http.Handler` boundary only | Do not mock the route table or the CORS wrapper; use the real mux and actual Origin headers | A buggy server that forgets to mount the note/relationship routes or exposes them without local-only CORS. |
| TC-F02-008 | `viewer/server.StartServer`, `db.InitDB(":memory:")`, and existing viewer/file-edit routes | Code review / startup smoke only | Do not mock schema initialization; this is a no-schema-change gate | A buggy implementation that adds a migration file or widens the database shape unexpectedly. |
| TC-F02-009 | Repo artifact scan + `viewer/server.StartServer` | Static check plus startup smoke | Do not mock the artifact scan; it must inspect the feature branch/file tree | A buggy implementation that lands a migration or schema file as a side effect of the feature. |
| TC-F02-010 | `viewer/server.StartServer` with real in-memory DB | Real DB and real HTTP handler | Do not mock the DB or the handler; this is a latency check | A buggy implementation that adds unnecessary round trips or expensive client-side processing before save. |

Internal-only justification:

- TC-F02-008 and TC-F02-009 are validated by code review plus unchanged database initialization tests. There is no runtime production caller for "no schema change" beyond startup compatibility and repo hygiene.

---

## 4. Integration Scenarios

These scenarios map to epic UAT coverage:

- UAT Scenario 2: Add a note to an entity
- UAT Scenario 3: Add and remove a dependency or relationship
- UAT Scenario 7: Reject invalid mutations cleanly

### 4.1 Viewer note flow

**Components:** `internal/viewer/assets/viewer.html`, `internal/viewer/server/server.go`, `internal/api/viewer/mutation_handler.go`, `internal/services/note_service.go`, `internal/api/viewer/handler.go`.

**What to verify:**

- The viewer shows an inline note composer for epic, feature, and task detail panes.
- Saving a note sends the correct POST request from the current entity context.
- The new note appears in the notes section after the request completes.
- The viewer does not leave the current entity context after the save.

### 4.2 Viewer relationship flow

**Components:** `internal/viewer/assets/viewer.html`, `internal/viewer/server/server.go`, `internal/api/viewer/mutation_handler.go`, `internal/services/entity_relationship_service.go`.

**What to verify:**

- The viewer exposes add/remove controls for directed relationships on epic, feature, and task detail panes.
- Creating a relationship uses the current entity as the source key.
- Deleting the relationship removes the directed edge from the normalized store and the current entity view.
- Duplicate links, self-links, missing entities, and cycle-creating dependency edges are rejected with deterministic error codes.

### 4.3 Boundary and regression flow

**Components:** viewer UI, viewer read routes, edit route, mutation handler, service-layer error mapping.

**What to verify:**

- Opening the viewer, fetching notes, or loading related docs does not mutate state.
- The existing read-only viewer routes continue to respond as before.
- The existing file edit route still behaves independently of the new entity mutation routes.
- The mutation routes stay local-only and preserve the viewer boundary.

---

## 5. Test Infrastructure

### 5.1 Existing patterns to follow

- `internal/api/viewer/mutation_handler_test.go`
  - Existing in-process mux and mock façade pattern for viewer HTTP handlers.
  - Best model for the new note and relationship handler cases.
- `internal/viewer/server/server_test.go`
  - Existing in-memory server startup and HTTP smoke tests.
  - Best model for route registration, local CORS, and latency verification.
- `internal/api/viewer/handler_test.go`
  - Existing read-path verification for notes and related viewer routes.
  - Best model for the post-create refresh assertion.
- `internal/viewer/assets_test.go`
  - Existing embedded asset smoke checks.
  - Best model for checking that the viewer HTML renders the new note and relationship controls.
- `internal/services/note_service_test.go`
  - Existing add/list validation pattern for notes across entity types.
  - Best model for service-layer note creation coverage.
- `internal/services/entity_relationship_service_test.go`
  - Existing create/delete/cycle-detection coverage for normalized relationships.
  - Best model for the duplicate/self-link/cycle behavior.

### 5.2 New test helpers needed only if the current patterns are insufficient

- `mockNoteRelationshipMutationServicer`
  - Extends the current viewer mutation mock with note and relationship methods.
  - Needed for `internal/api/viewer/mutation_handler_test.go` once the new routes are added.
- `newNoteRelationshipMux(...)`
  - Small helper to mount the expanded mutation handler routes in-process.
- `assertNoTaskDependencyWrite(...)`
  - Optional helper for the canonical-write test if the implementation needs a raw database snapshot comparison.
- `newViewerMutationServer(...)`
  - Optional helper for the in-memory server integration test if the route matrix gets large.

### 5.3 No new repository helpers expected

Because the spec requires no schema change, the feature does not need new repository test scaffolding.

---

## 6. Quality Coverage

| Requirement | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| REQ-F02-001 | ✅ TC-F02-001 | N/A | N/A | ✅ note composer present | N/A | N/A | ✅ thin-handler pattern | N/A |
| REQ-F02-002 | ✅ TC-F02-002 | N/A | ✅ same-session refresh | ✅ visible note immediately | ✅ no stale view after write | N/A | ✅ reuse viewer read path | N/A |
| REQ-F02-003 | ✅ TC-F02-003 | N/A | ✅ cross-entity source/target coverage | N/A | ✅ cycle-aware create | N/A | ✅ normalized store reuse | N/A |
| REQ-F02-004 | ✅ TC-F02-004 | N/A | ✅ same route family as create | N/A | ✅ delete correctness | N/A | ✅ single delete path | N/A |
| REQ-F02-005 | ✅ TC-F02-005/005b | N/A | N/A | ✅ clear 4xx responses | ✅ no partial writes | ✅ validation gates | ✅ service-owned rules | N/A |
| REQ-F02-006 | ✅ TC-F02-006 | N/A | ✅ no dual-write dependency model | N/A | ✅ single canonical write target | ✅ no `tasks.depends_on` mutation path | ✅ one source of truth | N/A |
| REQ-F02-007 | ✅ TC-F02-007/008 | N/A | ✅ viewer and edit routes coexist | ✅ unchanged viewer UX | ✅ no boundary regression | ✅ local-only CORS | ✅ separate read/write layers | N/A |
| REQ-F02-008 | ✅ TC-F02-008/009 | N/A | N/A | N/A | ✅ startup unchanged | N/A | ✅ no migration drift | N/A |
| REQ-NF02-003 | ✅ TC-F02-010 | ✅ TC-F02-010 | N/A | N/A | N/A | N/A | N/A | N/A |

---

## 7. Ready Check

This plan is ready for shark advancement once the implementation adds the F02 routes, the local-only CORS coverage for POST and DELETE, and the viewer refresh hooks required by the note and relationship flows.
