---
feature_key: E30-F01-entity-mutation-foundation
epic_key: E30
title: Entity Mutation Foundation
spec_version: 1.0
last_updated: 2026-05-07
complexity: COMPLEX
---

# Spec - E30-F01: Entity Mutation Foundation

> Scope guard: this feature is the foundation for the viewer mutation surface described in the epic PRD. It adds explicit edit and status-transition endpoints for epic, feature, and task entities only. Notes, dependency relationships, and sprint actions are reserved for E30-F02 and E30-F03. See epic PRD Section 3 for scope boundaries and Section 4 for constraints.

---

## 1. Requirements

This feature delivers the incremental write surface for editable epic, feature, and task fields plus workflow-driven status transitions. It reuses the existing service-layer write paths described in the research report and the parent epic docs.

### 1.1 Functional Requirements

| Req | Requirement | What this feature delivers |
|---|---|---|
| **REQ-F-001** | Viewer entity field edits | The viewer can update editable epic, feature, and task fields through explicit PATCH requests without leaving the viewer. |
| **REQ-F-002** | Workflow-driven status transitions | The viewer can transition epic, feature, and task status through explicit POST requests that preserve workflow validation, history, and orchestrator-action resolution. |
| **REQ-F-003** | Entity-specific validation | The mutation layer rejects unsupported fields, invalid keys, invalid JSON bodies, and empty transition targets before calling the underlying services. |
| **REQ-F-004** | Read-only viewer boundary preserved | Existing read endpoints remain unchanged; mutation routes are additive and do not turn `ViewerService` into a write path. |
| **REQ-F-005** | Audit trail preserved | Successful status transitions continue to produce `entity_history` rows through the existing transition engine. |
| **REQ-F-006** | No schema changes | The feature uses the current tables and repository layer; no database migration is required. |

### 1.2 Acceptance Criteria

- A maintainer can PATCH an epic, feature, or task from the viewer and receive the updated entity in the response.
- A maintainer can POST a status transition from the viewer and receive the same transition result shape used by the CLI services.
- Direct status edits are not allowed through the field-update endpoints.
- Invalid entity keys, malformed JSON, empty `target_status`, and unsupported fields return a 400-class error with no partial write.
- History continues to be recorded for successful transitions through the existing service-layer flow.
- The viewer read routes, file editor route, and sprint read endpoints continue to work unchanged.

### 1.3 Non-Functional Requirements

| Req | Requirement | Target |
|---|---|---|
| **REQ-NF-001** | Preserve service-layer business rules | Handlers and viewer code must not duplicate workflow validation, history recording, or transition semantics. |
| **REQ-NF-002** | Local-only posture remains in place | New mutation routes must use the same local CORS wrapper as the existing viewer routes and must not introduce a new auth model. |
| **REQ-NF-003** | Single-write latency | A single-entity PATCH or transition should complete in under 250 ms on a normal local SQLite database. |
| **REQ-NF-004** | Backward compatibility | Existing viewer GET endpoints and the file edit endpoint must remain behaviorally unchanged. |

### 1.4 Out of Scope

- Notes create/update/delete.
- Dependency add/remove or any normalized relationship writes.
- Sprint assignment, removal, readiness, capacity, or planning mutations.
- Arbitrary file mutation through the viewer mutation surface.
- Auth, authorization, or remote collaboration changes.
- Bulk edits or batch mutation APIs.

---

## 2. Architecture

### 2.1 Existing Foundation

These code paths already exist and should be reused, not reimplemented:

- `internal/services/entity_service.go`
- `internal/services/task_service.go`
- `internal/services/feature_service.go`
- `internal/services/epic_service.go`
- `internal/api/task_handler.go`
- `internal/api/feature_handler.go`
- `internal/api/epic_handler.go`
- `internal/api/viewer/handler.go`
- `internal/api/viewer/cors.go`
- `internal/viewer/server/wire.go`
- `internal/viewer/server/server.go`
- `internal/viewer/assets/viewer.html`
- `internal/keys/service.go`

The request/response shapes in the new viewer mutation surface should follow the same thin-handler pattern as the existing entity handlers:

- parse request body
- validate key and required fields
- call the service
- serialize the result

### 2.2 Component Changes

#### Files to Create

| File | Purpose |
|---|---|
| `internal/api/viewer/mutation_handler.go` | New HTTP handler for viewer write routes. Owns request parsing, route registration, and response formatting. |
| `internal/api/viewer/mutation_handler_test.go` | Route, validation, and response-shape tests for the viewer mutation handler. |
| `internal/api/viewer/mutation_service.go` | Thin viewer mutation façade that adapts to the existing task, feature, and epic services. |
| `internal/api/viewer/mutation_service_test.go` | Tests for routing the request to the correct entity service and preserving update/transition semantics. |

#### Files to Modify

| File | Change |
|---|---|
| `internal/viewer/server/wire.go` | Construct and inject the mutation façade alongside the read-only viewer service. |
| `internal/viewer/server/server.go` | Mount the mutation routes under `/api/v1/viewer/` and wrap them with `viewer.WithLocalCORS`. |
| `internal/viewer/assets/viewer.html` | Add entity edit controls and transition controls in the entity detail UI, wired to the new viewer mutation endpoints. |

### 2.3 API and Interface Contracts

Use explicit per-entity routes so request validation stays aligned with the existing service DTOs and so disallowed fields never reach the backend:

#### PATCH routes

- `PATCH /api/v1/viewer/epics/{key}`
- `PATCH /api/v1/viewer/features/{key}`
- `PATCH /api/v1/viewer/tasks/{key}`

#### POST transition routes

- `POST /api/v1/viewer/epics/{key}/transition`
- `POST /api/v1/viewer/features/{key}/transition`
- `POST /api/v1/viewer/tasks/{key}/transition`

#### Request body contracts

| Route | Allowed fields |
|---|---|
| Epic patch | `title`, `description`, `priority`, `business_value`, `size`, `clear_size` |
| Feature patch | `title`, `description`, `execution_order`, `size`, `clear_size` |
| Task patch | `title`, `description`, `priority`, `agent_type`, `execution_order`, `size`, `clear_size` |
| Transition body | `target_status`, `force`, `reason`, `agent` |

Notes:

- `status` is intentionally excluded from patch requests. Status changes must go through the transition route.
- `file_path` is intentionally excluded. File writes remain on `PUT /api/v1/edit/file`.
- `tags` are intentionally excluded. Tag attachment belongs to E30-F02.
- `skip_resequence` is intentionally excluded from the viewer surface to keep the first cut predictable.

#### Response contracts

- PATCH responses return the updated `models.Epic`, `models.Feature`, or `models.Task`.
- Transition responses return the existing `services.TransitionResult`.
- Error responses reuse the current viewer error formatting.

### 2.4 Key Technical Decisions

#### 2.4.1 Keep the mutation layer thin

The new viewer mutation code should not contain entity-specific business rules. It should only:

- validate request shape
- map request fields to the existing service DTOs
- call `UpdateEpic`, `UpdateFeature`, `UpdateTask`, or `TransitionStatus`
- format the response

This follows the same handler pattern used in `internal/api/task_handler.go`, `internal/api/feature_handler.go`, and `internal/api/epic_handler.go`.

#### 2.4.2 Use a viewer-facing mutation façade

Add a small façade in the `internal/api/viewer` package so the handler stays simple and tests can stub one dependency instead of three direct service references. The façade should depend on the same service interfaces the existing entity handlers use.

The façade should expose methods equivalent to:

- `UpdateEpic(ctx, key, updates)`
- `UpdateFeature(ctx, key, updates)`
- `UpdateTask(ctx, key, updates)`
- `TransitionEpic(ctx, key, targetStatus, opts)`
- `TransitionFeature(ctx, key, targetStatus, opts)`
- `TransitionTask(ctx, key, targetStatus, opts)`

#### 2.4.3 Preserve workflow semantics by reusing existing transition methods

The viewer must not patch status directly. It should call the existing transition methods so the following remain intact:

- validation and normalization
- forced-transition checks
- backward-transition detection
- rejection-note creation
- history recording
- orchestrator-action resolution

This is the same behavior already used by the CLI paths in `internal/services/task_service.go`, `internal/services/feature_service.go`, and `internal/services/epic_service.go`.

#### 2.4.4 Keep viewer read routes and file editing separate

The mutation layer must not be merged into `ViewerService` and must not reuse `EditService`. The existing viewer read model stays read-only, and the file writer remains a separate endpoint with different safety concerns.

### 2.5 Integration With Existing Code

#### Service wiring

`internal/viewer/server/wire.go` already constructs the underlying services:

- `TaskService`
- `FeatureService`
- `EpicService`
- `ViewerService`
- `EditService`

Extend the container with a viewer mutation dependency built from the existing task, feature, and epic services. No new domain services are required in `internal/services`.

#### HTTP wiring

`internal/viewer/server/server.go` should register the new mutation handler beside the existing viewer read routes and wrap it with `viewer.WithLocalCORS` so the local-only boundary stays consistent.

#### Frontend wiring

`internal/viewer/assets/viewer.html` should:

- render editable fields only for the selected entity type
- submit PATCH requests to the matching viewer mutation endpoint
- submit status changes to the matching transition endpoint
- refresh the current entity detail pane after a successful mutation by refetching the read endpoint

### 2.6 Data Model Changes

No schema changes are required.

The feature reuses:

- `tasks`
- `entity_history`
- existing epic/feature/task tables and repositories

The only new data is request payloads and UI state, both of which are transient.

### 2.7 Validation and Error Handling

- Use `json.Decoder.DisallowUnknownFields()` on mutation requests so unsupported fields fail fast.
- Normalize keys with the same key-validation logic already used by viewer read routes.
- Return 400 for malformed input, missing `target_status`, unsupported fields, and invalid key formats.
- Return the existing service-layer errors from the underlying update or transition methods without swallowing them.

---

## 3. Implementation Notes

### 3.1 Recommended rollout order

1. Add the mutation façade and route handlers.
2. Wire the new handler into the viewer server.
3. Add tests for handler validation and service delegation.
4. Add the first-pass UI controls in `internal/viewer/assets/viewer.html`.
5. Verify that reads, file editing, and transitions still work independently.

### 3.2 Test Coverage Targets

- PATCH rejects invalid JSON, unknown fields, and empty bodies.
- PATCH updates the right entity type and returns the updated model.
- POST transition calls the existing transition method and returns the transition result.
- Non-transition patch requests cannot alter status.
- Viewer mutation routes remain local-only and use the same CORS wrapper as the existing viewer routes.
- Existing viewer GET routes continue to serve the same payloads.

