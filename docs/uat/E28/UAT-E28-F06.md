---
feature_key: E28-F06-web-viewer-tag-integration
epic_key: E28
document_type: uat-guide
title: UAT Test Guide — Web Viewer Tag Integration
generated: 2026-04-24
status: Ready for UAT
---

# UAT Test Guide — Web Viewer Tag Integration

**Feature:** E28-F06 — Web Viewer Tag Integration
**Epic:** E28 — Entity Tagging with Managed Vocabulary
**Generated:** 2026-04-24

---

## Epic Context

**Epic Goal:** Add a managed (closed) tag vocabulary to shark, with retroactive
tagging support across all six entity types (epic, feature, task, bug, change,
idea), tag-based querying in `list`/`search` (AND semantics), and read-only
surfacing in the E27 web viewer.

**This Feature's Role:** F06 closes the loop on the epic by surfacing tags in
the web viewer. The viewer becomes a read-only consumer of the tag system —
showing tag chips on entity detail views and a multi-select tag filter on list
views. Vocabulary management intentionally stays CLI-only.

**Sibling Features:**
- E28-F01 — Tags Schema and Migration ✅ completed
- E28-F02 — Reusable Maintainer Authorization Gate ✅ completed
- E28-F03 — Tag Vocabulary Service and CLI ✅ completed
- E28-F04 — Entity Tag Attachment and Enforcement ✅ completed
- E28-F05 — Tag-Based Querying in List and Search ✅ completed
- **E28-F06 — Web Viewer Tag Integration ⬅️ THIS FEATURE**
- E28-F07 — Tagging Documentation (draft, parallel)

**Integration Points:**
- F06 consumes `TagService.ListTags` (F03), `EntityIDsByTags` (F05), and
  `AttachedTagNamesByIDs` (F05) via a narrow `TagReader` interface.
- F06 adds NO new SQL, NO new repository methods, NO migration. Pure additive
  wiring on existing API and viewer surface.
- The `MaintainerGate` (F02) is explicitly NOT injected — defense-in-depth.

---

## Design Intent

**From Epic PRD (SC-8):** "Any entity view in the web viewer displays the
entity's tags; vocabulary management stays CLI-only."

**From Feature Description:** "Surface tags in the E27 web viewer (read-only),
add `tags[]` to every entity response DTO, accept `?tag=` query params on list
endpoints (AND semantics), expose `GET /api/v1/viewer/tags` for the vocabulary,
and render tag chips + a tag filter control in the UI."

**Key Design Decisions (ADRs from spec §2.4):**
- **ADR-F06-1:** Decorate DTOs in-memory via batched `AttachedTagNamesByIDs`
  calls (one per entity type) — not N+1.
- **ADR-F06-2:** `tags` field is always non-nil — `[]` not `null`.
- **ADR-F06-3:** Viewer never imports `MaintainerGate` (defense-in-depth).
- **ADR-F06-4:** Hierarchy filter prunes the tree; UI does not see hidden
  nodes.
- **ADR-F06-5:** `Hierarchy` grows `HierarchyOptions`; `FeatureTasks` extends
  existing `FeatureTaskOptions`.
- **ADR-F06-6:** UI is pure CSS + vanilla JS additions; no new build step.

---

## Cross-Feature Integration Tests

### Integration Scenario 1: F03 Vocabulary ↔ F06 Viewer Endpoint

**Features:** E28-F03 (`shark tags add`) + E28-F06 (`GET /api/v1/viewer/tags`)

Steps:
1. Run `shark tags add voice --pass=<p>` and `shark tags add auth --pass=<p>`
2. Boot the viewer server.
3. Curl `GET /api/v1/viewer/tags`.

Expected: Returns `{"tags":[{"name":"auth"},{"name":"voice"}]}` — alphabetical.

### Integration Scenario 2: F04 Tag Attachment ↔ F06 Hierarchy Decoration

**Features:** F04 (`shark <entity> tag add`) + F06 (`GET /api/v1/viewer/hierarchy`)

Steps:
1. Tag one of each entity type (epic, feature, task, bug, change, idea) with `voice`.
2. Curl `GET /api/v1/viewer/hierarchy`.

Expected: Tagged entities show `"tags":["voice"]`; untagged show `"tags":[]`
(never `null`).

### Integration Scenario 3: F05 AND-Semantics ↔ F06 Filter

**Features:** F05 (`EntityIDsByTags` AND op) + F06 (`?tag=` query)

Steps:
1. Tag one entity with both `voice` and `auth`; another with only `voice`.
2. Curl `GET /api/v1/viewer/hierarchy?tag=voice&tag=auth`.

Expected: Only the entity with BOTH tags appears.

### Integration Scenario 4: F02 Gate ↔ F06 (NOT injected)

**Features:** F02 MaintainerGate + F06 ViewerService

Verify: `grep "maintainer\.Gate\|MaintainerGate" internal/services/viewer_service.go`
returns zero matches; `NewViewerService` constructor signature does not contain
a gate parameter.

---

## Epic Acceptance Validation

| Epic SC | Description | F06 Contribution | Status |
|---------|-------------|------------------|--------|
| SC-1 | Closed managed vocabulary across six entity types | F06 surfaces vocabulary in viewer (read-only) | [ ] |
| SC-8 | Any entity view in viewer displays tags; vocabulary mgmt CLI-only | All F06 ACs trace to this | [ ] |
| SC-9 | Defense-in-depth: viewer cannot mutate vocabulary | AC-15 (no gate), REQ-F-013 (no mutation routes) | [ ] |

---

## Feature Acceptance Validation

The 17 ACs from spec §1.3 below.

| AC | Description | Test Source | Status |
|----|-------------|-------------|--------|
| AC-01 | Empty vocab → `{"tags":[]}` | service test ✅ | [ ] |
| AC-02 | Populated alphabetical | service test ✅ | [ ] |
| AC-03 | POST/PUT/PATCH/DELETE → 404/405 | **No handler test ⚠️** | [ ] |
| AC-04 | All entity DTOs `tags: []` not null | service test ✅ | [ ] |
| AC-05 | Tagged entities carry correct names | service test ✅ | [ ] |
| AC-06 | `?tag=voice` prunes subtrees | service test ✅ | [ ] |
| AC-07 | `?tag=a&tag=b` AND semantics | service test ✅ | [ ] |
| AC-08 | Unregistered tag → 400 with `unregistered_tags` | **No handler test ⚠️** | [ ] |
| AC-09 | FeatureTasks tag filter pre-pagination | service test ✅ | [ ] |
| AC-10 | FeatureTasks 400 (tag) vs 404 (feature) | **No handler test ⚠️** | [ ] |
| AC-11 | UI chips on tagged entities | manual UAT | [ ] |
| AC-12 | UI filter control + refetch | manual UAT | [ ] |
| AC-13 | No UI mutation controls | manual UAT | [ ] |
| AC-14 | Graceful nil-tagSvc | service test ✅; **handler 200/{tags:[]} test ⚠️** | [ ] |
| AC-15 | No MaintainerGate in wire.go | code grep ✅ | [ ] |
| AC-16 | ≤6 SQL stmts via mock counter | service test ✅ | [ ] |
| AC-17 | `make fmt && lint && test` passes | quality gate | [ ] |

---

## Test Scenarios

### Scenario 1: Vocabulary Endpoint (AC-01, AC-02, AC-03)
**Tasks covered:** T-E28-F06-001 (service), T-E28-F06-004 (handler+route)

Service-side coverage in `internal/services/viewer_service_test.go`:
- `TestViewerService_Tags_EmptyVocabulary` — empty case
- `TestViewerService_Tags_AlphabeticalOrdering` — alphabetical
- `TestViewerService_Tags_DelegatesToListTagsOnce` — single delegation
- `TestViewerService_Tags_ProjectsTagToDTO` — projection

Handler-side coverage in `internal/api/viewer/handler_test.go`:
- `MockViewerServicer.TagsFunc` field added (line 40), but **no test cases
  invoke the new route**. AC-03 (POST/PUT/PATCH/DELETE method tests) is not
  covered by any automated test.

### Scenario 2: Hierarchy Decoration (AC-04, AC-05, AC-16)
**Tasks covered:** T-E28-F06-002

Service-side: 6 tests covering decoration, batching, empty hierarchy, large
trees (TestViewerService_Hierarchy_TagDecoration_*).

### Scenario 3: Hierarchy Filter (AC-06, AC-07, AC-08)
**Tasks covered:** T-E28-F06-002 (filter logic), T-E28-F06-005 (handler wiring)

Service-side: 6 tests covering pruning, AND semantics, error propagation.
Handler-side: AC-08 (400 with `unregistered_tags`) is **not** tested at the
handler layer.

### Scenario 4: FeatureTasks Filter and Decoration (AC-09, AC-10)
**Tasks covered:** T-E28-F06-003 (service), T-E28-F06-005 (handler)

Service-side: 4 tests covering pre-pagination filter, page-scoped decoration,
graceful nil-tagSvc.
Handler-side: AC-10 (400 vs 404 ordering) is **not** tested at the handler
layer.

### Scenario 5: UI Chips and Filter (AC-11, AC-12, AC-13)
**Tasks covered:** T-E28-F06-007

Asset is embedded; `internal/viewer/assets_test.go` (103 tests) confirms it
serves. UI rendering is manual UAT only.

Verified by inspection:
- `.tag-chip` CSS class at `viewer.html:1391`
- `loadVocabulary()` JS at `viewer.html:2019`
- `renderTagChipsHtml()` at `viewer.html:2030`
- `applyTagFilter()` at `viewer.html:2044`
- Mutation control absence verified by QA report (T-E28-F06-007 PASS)

### Scenario 6: Graceful Degradation (AC-14)
**Tasks covered:** T-E28-F06-001/002/003 (service); T-E28-F06-004 (handler)

Service-side: nil-tagSvc tests for Tags, Hierarchy, FeatureTasks all pass.
Handler-side: handler always delegates to service; no separate test.

### Scenario 7: Defense-in-Depth (AC-15)
**Tasks covered:** T-E28-F06-006

Verified by code inspection:
- `internal/services/viewer_service.go` — `NewViewerService` signature has 11
  parameters, none of type `*maintainer.Gate`. No `maintainer` import.
- `internal/viewer/server/wire.go:371` — explicit comment: `// REQ-F-014:
  MaintainerGate is NOT passed to ViewerService (defense-in-depth).`

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-04-25 |
| Result | REJECT (Codex red-team verdict; human accepted) |
| Results File | `docs/uat/E28/results/UAT-E28-F06-20260425-000605-results.md` |
| Tasks routed back | T-E28-F06-004 (AC-03), T-E28-F06-005 (AC-08/AC-10), T-E28-F06-007 (AC-12) |
| Tasks remaining at in_approval | T-E28-F06-003, T-E28-F06-006 |
| Tasks already completed | T-E28-F06-001, T-E28-F06-002 |

**Previous Sessions:** None prior to 2026-04-25.
