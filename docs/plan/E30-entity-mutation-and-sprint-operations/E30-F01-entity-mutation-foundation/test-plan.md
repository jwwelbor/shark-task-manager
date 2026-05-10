---
feature_key: E30-F01-entity-mutation-foundation
epic_key: E30
document_type: test-plan
title: Test Plan - Entity Mutation Foundation
created: 2026-05-07
status: DRAFT
---

# E30-F01 Test Plan

Traceability: every test case below maps to at least one requirement in `./spec.md`, and every requirement in `spec.md` has at least one test case here. Epic UAT references come from `../uat-plan.md`.

Testing rules in force:

- Service-layer tests use mocks for the underlying entity services and history recorder.
- HTTP handler tests use the existing `internal/api/viewer/handler_test.go` pattern: in-process mux, mocked service interface, no real DB unless the test is explicitly a server integration test.
- Only server integration and latency tests use a real in-memory SQLite DB.
- There are no repository-level tests for this feature because the spec requires no schema change.

---

## 1. Spec Drift Analysis

### 1.1 Drift Findings

No material drift identified.

Observations:

- The spec deliberately narrows the mutation surface to epic, feature, and task entities only. Notes, relationships, and sprint actions are correctly deferred to E30-F02 and E30-F03.
- The spec uses explicit per-entity routes instead of a generic mutation route. That is an implementation choice to reduce validation ambiguity, not a scope change.
- The spec keeps file mutation on `PUT /api/v1/edit/file` and keeps the viewer read service read-only. That matches the epic constraints.
- No schema changes are specified, which is consistent with the research report and the epic architecture doc.

### 1.2 Traceability Matrix

| Spec Requirement | Test Cases | Covered |
|---|---|---|
| REQ-F-001 Viewer entity field edits | TC-F01-001, TC-F01-007 | Yes |
| REQ-F-002 Workflow-driven status transitions | TC-F01-004, TC-F01-005 | Yes |
| REQ-F-003 Entity-specific validation | TC-F01-003, TC-F01-005 | Yes |
| REQ-F-004 Read-only viewer boundary preserved | TC-F01-006 | Yes |
| REQ-F-005 Audit trail preserved | TC-F01-004, TC-F01-005 | Yes |
| REQ-F-006 No schema changes | TC-F01-008 | Yes |
| REQ-NF-001 Preserve service-layer business rules | TC-F01-004, TC-F01-005 | Yes |
| REQ-NF-002 Local-only posture remains in place | TC-F01-006 | Yes |
| REQ-NF-003 Single-write latency | TC-F01-009 | Yes |
| REQ-NF-004 Backward compatibility | TC-F01-006 | Yes |

---

## 2. AC Test Matrix

### 2.1 Field edits and transition routes

| AC / Req | Test Case | Techniques Applied | Input / Setup | Expected Outcome | Location |
|---|---|---|---|---|---|
| REQ-F-001 | TC-F01-001 `TestMutationHandler_UpdateFields_PerEntity` | Equivalence Partitioning, Contract surface enumeration | PATCH requests for epic, feature, and task with allowed field sets only | Each route returns 200 and the updated model. Fields not relevant to the entity type are rejected or ignored according to the request contract. | `internal/api/viewer/mutation_handler_test.go` |
| REQ-F-002 | TC-F01-004 `TestMutationHandler_TransitionStatus_PerEntity` | State transitions, Decision table | POST transition requests for epic, feature, and task with valid `target_status` values | Each route returns 200 and a `TransitionResult`. The result preserves workflow validation and orchestrator-action behavior from the existing services. | `internal/api/viewer/mutation_handler_test.go` + `internal/services/{task,feature,epic}_service_test.go` |
| REQ-F-003 | TC-F01-003 `TestMutationHandler_RejectsDisallowedFieldsAndBadJSON` | Attack-class enumeration, Equivalence Partitioning | PATCH body contains `status`, `file_path`, `tags`, or malformed JSON; POST body omits `target_status` | Each request fails with 400 and no partial write. `status` cannot be mutated through the patch path. | `internal/api/viewer/mutation_handler_test.go` |
| REQ-F-004 | TC-F01-006 `TestViewerServer_MountsMutationRoutesAndKeepsReadRoutes` | Contract surface enumeration | Start viewer server in-memory, hit existing GET routes and new mutation routes | Existing GET routes still respond as before; mutation routes are mounted and wrapped with local CORS. | `internal/viewer/server/server_test.go` |
| REQ-F-005 | TC-F01-004 / TC-F01-005 | State transitions | Successful status transition through the new viewer route | Existing entity history assertions remain satisfied through the existing transition engine. | `internal/services/{task,feature,epic}_service_test.go` |
| REQ-F-006 | TC-F01-008 `TestSpec_NoSchemaChange_NoDbMigrationNeeded` | Internal-only justification | Run server startup and existing DB init smoke tests against current schema | No new schema or migration file is required, and startup uses the current database shape unchanged. | `internal/db/db_test.go` + code review gate |
| REQ-NF-001 | TC-F01-004 / TC-F01-005 | Decision table | Transition requests on valid and invalid inputs | Handlers do not duplicate workflow logic; invalid transitions are delegated to the existing service errors. | `internal/api/viewer/mutation_handler_test.go` |
| REQ-NF-002 | TC-F01-006 | Security / boundary enumeration | Requests from local and non-local origins | Local CORS wrapper remains in place; non-local origins do not gain write access. | `internal/viewer/server/server_test.go` |
| REQ-NF-003 | TC-F01-009 `TestMutationRoutes_SingleWriteCompletesWithinBudget` | Boundary Value Analysis | Single PATCH or POST transition against in-memory DB; repeat across epic, feature, task | Each request completes under 250 ms on a normal local SQLite DB. | `internal/viewer/server/server_test.go` or `internal/api/viewer/mutation_handler_test.go` |
| REQ-NF-004 | TC-F01-006 | Backward compatibility | Existing viewer GET endpoints and file editor route | Existing behavior remains unchanged. | `internal/viewer/server/server_test.go` |

### 2.2 Edge cases per AC

- Field edits:
  - Empty body
  - Unknown JSON field
  - Attempted status patch
  - Entity-specific fields sent to the wrong entity type
- Transitions:
  - Empty `target_status`
  - Invalid `target_status`
  - Forced transition without a reason
  - Repeated transition to the current status
- Routing and CORS:
  - Existing GET route still returns the same payload
  - Mutation routes are blocked from non-local origins
- Performance:
  - Single request against a warm in-memory DB
  - Single request after one successful mutation in the same process

---

## 3. Caller-Path Contracts

| Test Case | Production Entrypoint | Lowest Allowed Mock Seam | Forbidden Mocks | Counter-Factual |
|---|---|---|---|---|
| TC-F01-001 | `MutationHandler.UpdateEpic`, `UpdateFeature`, `UpdateTask` | `MutationServicer.UpdateEpic` / `UpdateFeature` / `UpdateTask` | Do not mock the JSON decoder or the mux; the handler must parse real request bodies | A buggy handler that forwards `status` through the patch path or routes the request to the wrong entity service |
| TC-F01-003 | `MutationHandler.Update*` and `MutationHandler.Transition*` | `MutationServicer` methods | Do not mock validation after the service boundary; the handler must reject malformed payloads before delegation | A buggy handler that accepts unknown fields or empty transition targets |
| TC-F01-004 | `MutationHandler.TransitionEpic`, `TransitionFeature`, `TransitionTask` | `MutationServicer.Transition*` | Do not mock `EntityService.TransitionStatus`; the underlying transition flow must run | A buggy implementation that bypasses workflow validation or returns the wrong response shape |
| TC-F01-005 | `TaskService.TransitionStatus`, `FeatureService.TransitionStatus`, `EpicService.TransitionStatus` | `historyRecorder` and repository mocks only | Do not mock the transition method itself; the test must exercise the shared transition engine | A buggy implementation that updates status but skips `entity_history` or rejection-note behavior |
| TC-F01-006 | `viewer/server.StartServer` and `mutationHandler.RegisterRoutes` | `http.Handler` boundary only | Do not mock the route table or the CORS wrapper; use the real mux | A buggy server that forgets to mount mutation routes or exposes them without local-only CORS |
| TC-F01-007 | `viewer.ViewerHTML` embedded asset | N/A - static asset smoke test | Do not mock the embedded asset | A buggy viewer build that omits mutation controls or the fetch helpers needed by the UI |
| TC-F01-008 | `db.InitDB(":memory:")` and server startup | Code review / existing DB smoke tests only | Do not mock the DB schema; this is a no-schema-change gate | A buggy implementation that adds a migration or modifies startup schema unexpectedly |
| TC-F01-009 | `viewer/server.StartServer` with real in-memory DB | Real DB and real HTTP handler | Do not mock the DB or the handler; this is a latency check | A buggy implementation that adds extra round trips or expensive client-side work before save |

Internal-only justification:

- TC-F01-008 is validated by code review plus unchanged database initialization tests. There is no runtime production caller for "no schema change" beyond startup compatibility, so this is a static gate.

---

## 4. Integration Scenarios

These scenarios map to epic UAT coverage:

- UAT Scenario 1: Edit an entity field in the viewer
- UAT Scenario 4: Transition status from the viewer
- UAT Scenario 7: Reject invalid mutations cleanly

### 4.1 Viewer edit flow

**Components:** `internal/viewer/assets/viewer.html`, `internal/viewer/server/server.go`, `internal/api/viewer/mutation_handler.go`, existing entity services.

**What to verify:**

- The viewer shows edit controls for the selected epic, feature, or task.
- Saving a field update sends the correct PATCH request.
- The response updates the current view without requiring a CLI or database edit.

### 4.2 Viewer transition flow

**Components:** `internal/viewer/assets/viewer.html`, `internal/api/viewer/mutation_handler.go`, `internal/services/entity_service.go`, `internal/services/{task,feature,epic}_service.go`.

**What to verify:**

- The viewer sends the transition request with `target_status`.
- The transition obeys existing workflow validation.
- The history surface remains intact after the write.

### 4.3 Invalid mutation flow

**Components:** viewer UI, mutation handler, existing service-layer error mapping.

**What to verify:**

- Invalid JSON never reaches the service layer.
- Unsupported fields fail fast.
- No partial write is visible after a rejected request.

---

## 5. Test Infrastructure

### 5.1 Existing patterns to follow

- `internal/api/viewer/handler_test.go`
  - Existing mock-driven request/response tests for viewer HTTP handlers.
  - Best model for the new mutation handler tests.
- `internal/viewer/server/server_test.go`
  - Existing in-memory server startup and HTTP smoke tests.
  - Best model for route-registration and CORS verification.
- `internal/viewer/assets_test.go`
  - Existing embedded asset smoke checks.
  - Best model for checking that the new UI helpers are present in `viewer.html`.
- `internal/services/task_service_test.go`
  - Existing transition-history and service-boundary test patterns.
  - Best model for audit-trail assertions.
- `internal/services/feature_service_test.go` and `internal/services/epic_service_test.go`
  - Existing update/transition service tests that already validate the service contracts F01 will reuse.

### 5.2 New test helpers needed

- `MockViewerMutationServicer`
  - Mirrors the existing `MockViewerServicer` pattern.
  - Needed for `internal/api/viewer/mutation_handler_test.go`.
- `newMutationMux(...)`
  - Small helper to mount the new mutation handler routes in-process.
- `assertMutationJSON(...)`
  - Optional helper to keep request/response assertions consistent across epic, feature, and task cases.

### 5.3 No new repository helpers required

Because the spec requires no schema change, the feature does not need new repository test scaffolding.

---

## 6. Quality Coverage

| Requirement | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| REQ-F-001 | ✅ TC-F01-001 | N/A | N/A | ✅ TC-F01-007 | N/A | N/A | ✅ thin-handler pattern | N/A |
| REQ-F-002 | ✅ TC-F01-004 | N/A | N/A | ✅ TC-F01-005 | ✅ TC-F01-005 | N/A | ✅ reuse existing transition engine | N/A |
| REQ-F-003 | ✅ TC-F01-003 | N/A | N/A | ✅ clear 400s | ✅ no partial writes | ✅ TC-F01-003 | ✅ fail-fast handler validation | N/A |
| REQ-F-004 | ✅ TC-F01-006 | N/A | ✅ TC-F01-006 | N/A | ✅ TC-F01-006 | ✅ local-only CORS | ✅ separate read/mutation boundary | N/A |
| REQ-F-005 | ✅ TC-F01-004/005 | N/A | N/A | N/A | ✅ TC-F01-005 | N/A | ✅ existing history flow | N/A |
| REQ-F-006 | ✅ TC-F01-008 | N/A | N/A | N/A | N/A | N/A | ✅ no DB migration path | N/A |
| REQ-NF-003 | ✅ TC-F01-009 | ✅ TC-F01-009 | N/A | N/A | N/A | N/A | N/A | N/A |
| REQ-NF-004 | ✅ TC-F01-006 | N/A | ✅ TC-F01-006 | N/A | ✅ TC-F01-006 | N/A | ✅ route separation | N/A |

