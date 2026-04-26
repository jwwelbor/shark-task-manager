---
feature_key: E28-F06-web-viewer-tag-integration
epic_key: E28
document_type: uat-evidence
title: UAT Evidence Package — E28-F06
session_date: 2026-04-25
collected_by: Claude (evidence-only; no analysis)
---

# UAT Evidence Package — E28-F06

This file collects raw evidence for Codex red-team review. **No Claude
assessment.** Just facts: spec quotes, code references, test output.

## Source Artifacts

- **Spec:** `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F06-web-viewer-tag-integration/spec.md`
- **Test Plan:** `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F06-web-viewer-tag-integration/test-plan.md`
- **Feature:** `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F06-web-viewer-tag-integration/feature.md`
- **Implementation files:**
  - `internal/services/viewer_service.go` (DTOs, TagReader, Tags/Hierarchy/FeatureTasks methods)
  - `internal/api/viewer/handler.go` (route registration, parseTagsQuery, respondUnregisteredTagError)
  - `internal/api/viewer/service.go` (ViewerServicer interface)
  - `internal/viewer/server/wire.go` (WithTagService injection; no MaintainerGate)
  - `internal/viewer/assets/viewer.html` (tag chips, filter control)
- **Test files:**
  - `internal/services/viewer_service_test.go` (40+ tag tests)
  - `internal/api/viewer/handler_test.go` (mock scaffolding only)
- **QA reports already filed:**
  - `qa_reports/20260424-182654-T-E28-F06-001-qa-results.md` (PASS)
  - `qa_reports/20260424-185528-T-E28-F06-002-qa-results.md` (PASS)
  - `qa_reports/20260424-230311-T-E28-F06-007-qa.md` (PASS)
- **Task review:** `task_reviews/E28-F06-task-review.md` (PASS, requirements coverage matrix)

---

## Per-AC Evidence

### AC-01 — Empty vocabulary returns `{"tags":[]}`

**Spec quote (spec.md L94):**
> AC-01 — `curl -s http://localhost:<port>/api/v1/viewer/tags` with an empty
> vocabulary returns `200 OK` with body `{"tags":[]}`.

**Implementation:** `internal/services/viewer_service.go:549-572`
```go
func (s *ViewerService) Tags(ctx context.Context) (*TagsResponse, error) {
    ...
    if s.tagSvc == nil {
        return &TagsResponse{Tags: []TagDTO{}}, nil
    }
    tags, err := s.tagSvc.ListTags(ctx)
    ...
    out := make([]TagDTO, len(tags))
    ...
    return &TagsResponse{Tags: out}, nil
}
```
Note: `make([]TagDTO, len(tags))` returns `[]TagDTO{}` (not nil) when len==0,
which JSON-marshals as `[]`.

**Test output (passing):**
- `TestViewerService_Tags_EmptyVocabulary` — PASS
- `TestViewerService_Tags_NilSliceMarshalGuard` — PASS

**Handler implementation:** `internal/api/viewer/handler.go:476-484`
```go
func (h *ViewerHandler) Tags(w http.ResponseWriter, r *http.Request) {
    result, err := h.svc.Tags(r.Context())
    if err != nil {
        slog.Error("viewer tags failed", "endpoint", "tags", "error", err)
        respondError(w, http.StatusInternalServerError, "failed to load tags")
        return
    }
    respondJSON(w, http.StatusOK, result)
}
```

**Handler test:** No automated test invokes the `GET /api/v1/viewer/tags`
route. `TagsFunc` field exists on `MockViewerServicer` (handler_test.go:40)
but is never used.

---

### AC-02 — Populated vocabulary returns names alphabetically

**Spec quote (spec.md L95):** Returns `{"tags":[{"name":"auth"},{"name":"voice"}]}` (alphabetical).

**Implementation:** Tag service `ListTags` already returns alphabetical order
(F03 contract). Viewer service projects 1:1.

**Test output:**
- `TestViewerService_Tags_AlphabeticalOrdering` — PASS
- `TestViewerService_Tags_DelegatesToListTagsOnce` — PASS
- `TestViewerService_Tags_ProjectsTagToDTO` — PASS

**Handler test:** None (see AC-01 note).

---

### AC-03 — Non-GET methods return 404 or 405

**Spec quote (spec.md L96):**
> `curl -X POST http://localhost:<port>/api/v1/viewer/tags -d '{"name":"foo"}'`
> returns 404 or 405 — no tag is created in the DB.

**Implementation:** `internal/api/viewer/handler.go:69` registers only:
```go
mux.Handle("GET "+prefix+"/tags", wrap(http.HandlerFunc(h.Tags)))
```
Method-specific pattern (Go 1.22 `mux.Handle("GET ...")`) means non-GET methods
hit the default 404/405 path.

**Test output:** **NO automated handler test for AC-03.** Searched
`internal/api/viewer/*_test.go` — no test issues a non-GET request to
`/api/v1/viewer/tags`. Test plan TC-AC03-1..5 are unimplemented.

---

### AC-04 — Hierarchy DTOs always carry non-null `tags: []`

**Spec quote (spec.md L97):** Every `tags` field is `[]`, never `null`.

**Implementation:** `internal/services/viewer_service.go:1148+` (`applyTagsToHierarchy`)
walks the response and assigns `[]string{}` when no tags. Type definitions at
lines around 230-280 declare `Tags []string \`json:"tags"\``.

**Test output:**
- `TestViewerService_Hierarchy_NilTagSvc_TagsAlwaysNonNil` — PASS
- `TestViewerService_Hierarchy_TagSvcWired_UntaggedEntities` — PASS
- `TestViewerService_FeatureTasks_TagDecoration_EmptyTagsNonNil` — PASS

---

### AC-05 — Tagged entities show correct names

**Implementation:** `applyTagsToHierarchy` looks up `tagsByEntity[entityType][id]`
per DTO and assigns. `fetchTagsForHierarchy` (line 1087) batches calls.

**Test output:**
- `TestViewerService_Hierarchy_TagDecoration_SingleEpicTagged` — PASS
- `TestViewerService_Hierarchy_TagDecoration_BatchCallCount` — PASS
- `TestViewerService_Hierarchy_TagDecoration_SkipsEmptyEntityTypes` — PASS

---

### AC-06, AC-07 — Hierarchy filter pruning + AND semantics

**Implementation:**
- `computeHierarchyTagIDSets` (line 1058) calls `EntityIDsByTags(ctx, et, tags, TagQueryOpAnd)` per type.
- `pruneHierarchy` (line 1219) walks tree and removes non-matching subtrees.

**Test output (all PASS):**
- `TestViewerService_Hierarchy_TagFilter_PrunesUnmatchedEpics`
- `TestViewerService_Hierarchy_TagFilter_EpicDirectTaggedNoMatchingFeatures`
- `TestViewerService_Hierarchy_TagFilter_FlatEntitiesIndependentlyFiltered`
- `TestViewerService_Hierarchy_TagFilter_NothingTagged_EmptyResult`
- `TestViewerService_Hierarchy_TagFilter_AndSemanticsOneCallPerType`

---

### AC-08 — Unregistered tag → 400 with `unregistered_tags` array

**Spec quote (spec.md L101):**
> Returns `400 Bad Request` with body `{"error":"Bad Request","message":"unregistered tag: does-not-exist","unregistered_tags":["does-not-exist"]}`.

**Implementation:** `internal/api/viewer/handler.go:96-103`
```go
result, err := h.svc.Hierarchy(r.Context(), opts)
if err != nil {
    var unregErr *services.UnregisteredTagError
    if errors.As(err, &unregErr) {
        respondUnregisteredTagError(w, unregErr)
        return
    }
    ...
}
```
And `respondUnregisteredTagError` at line 510-520:
```go
resp := tagErrorResponse{
    Error:            "Bad Request",
    Message:          err.Error(),
    UnregisteredTags: []string{err.Name},
}
```

**Service-level test (PASS):**
- `TestViewerService_Hierarchy_TagFilter_UnregisteredTagErrorPropagates`

**Handler test:** **NO automated handler test for AC-08.** Test plan TC-AC08-1..4
are unimplemented at the handler layer.

---

### AC-09 — FeatureTasks tag filter pre-pagination, Total reflects pre-page count

**Implementation:** `internal/services/viewer_service.go:1785+` — tag filter at
line 1829, decoration at line 1905.

**Test output (all PASS):**
- `TestViewerService_FeatureTasks_TagFilter_BeforePagination`
- `TestViewerService_FeatureTasks_NoTagFilter_UnchangedBehavior`
- `TestViewerService_FeatureTasks_TagDecoration_PageScopedIDs`

---

### AC-10 — FeatureTasks: 400 (unregistered) vs 404 (missing feature)

**Implementation:** `internal/api/viewer/handler.go:235-245`
```go
opts := services.FeatureTaskOptions{
    ...
    Tags:    parseTagsQuery(r),
}
result, err := h.svc.FeatureTasks(r.Context(), key, opts)
if err != nil {
    var unregErr *services.UnregisteredTagError
    if errors.As(err, &unregErr) {
        respondUnregisteredTagError(w, unregErr)
        return
    }
    if isNotFound(err) { ... 404 ... }
    ...
}
```
Feature-not-found returns from service before tag validation runs (the service
loads the feature first).

**Handler test:** **NO automated handler test for AC-10.** Test plan
TC-AC10-1..4 are unimplemented.

---

### AC-11, AC-12, AC-13 — UI chips, filter control, no mutation controls

**Marked as manual UAT in spec.md §2.6 item 5.**

**Code-level evidence:**
- `viewer.html:1391` — `.tag-chip` CSS class
- `viewer.html:1413` — `.tag-filter-chips` container
- `viewer.html:2019` — `loadVocabulary()` async function
- `viewer.html:2030` — `renderTagChipsHtml(entity)` (returns null on empty)
- `viewer.html:2044` — `applyTagFilter(selectedTags)` builds repeated `?tag=` params
- `viewer.html:2061+` — `renderTagFilterControl()` with empty-state for no vocabulary
- `viewer.html:4331` — call site: `renderTagChipsHtml(entity) || ''`

**T-E28-F06-007 QA report verdict: PASS** for all UI-side ACs (AC-T1..AC-T5,
including AC-13 mutation-control-absence by grep).

---

### AC-14 — Graceful degradation when TagReader is nil

**Implementation:**
- `Tags()` line 553-556: `if s.tagSvc == nil { return &TagsResponse{Tags: []TagDTO{}}, nil }`
- `Hierarchy()` line 820+: `fetchTagsForHierarchy` early-returns when nil; `applyTagsToHierarchy` always assigns `[]string{}` to nil entries.
- `FeatureTasks()` similar.

**Test output (all PASS):**
- `TestViewerService_Tags_NilTagSvc_GracefulDegradation`
- `TestViewerService_Tags_NilTagSvc_NoPanic`
- `TestViewerService_Hierarchy_NilTagSvc_TagsAlwaysNonNil`
- `TestViewerService_Hierarchy_NilTagSvc_WithTagFilter_Graceful`
- `TestViewerService_Hierarchy_NilTagSvc_NoTagFilter_EmptyTags`
- `TestViewerService_FeatureTasks_NilTagSvc_TagFilter_Graceful`

**Handler test:** None. (Handler is a thin pass-through.)

---

### AC-15 — `wire.go` does NOT pass MaintainerGate to ViewerService

**Spec quote (spec.md L108):** "Server-startup-time grep of
`internal/viewer/server/wire.go` shows that `MaintainerGate` is NOT passed into
`NewViewerService`."

**Code inspection:**
```
$ grep -n "MaintainerGate\|maintainer.Gate" internal/services/viewer_service.go
(zero matches)

$ grep -n "MaintainerGate\|maintainer.Gate\|maintainer.New" internal/viewer/server/wire.go
8:	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
38:// loadMaintainerGate builds a maintainer.Gate from the project's
42:func loadMaintainerGate(projectRoot string) maintainer.Gate {
50:	return maintainer.NewFileGate(projectRoot, mc, mc.CacheWindow())
275:	tagGate := loadMaintainerGate(projectRoot)
371:	// REQ-F-014: MaintainerGate is NOT passed to ViewerService (defense-in-depth).
```

**Constructor signature** (`viewer_service.go:461-475`):
```go
func NewViewerService(
    epicRepo ViewerEpicRepository,
    featureRepo ViewerFeatureRepository,
    taskRepo ViewerTaskRepository,
    bugRepo ViewerBugRepository,
    changeCardRepo ViewerChangeCardRepository,
    historyRepo ViewerEntityHistoryRepository,
    workflowSvc *workflow.Service,
    statusCalc *status.CalculationService,
    projectRoot string,
    entityRelSvc *EntityRelationshipService,
    entityRegistry *EntityRegistry,
) *ViewerService {
```
No `*maintainer.Gate` parameter. Wire site at line 371 explicitly comments the
omission. The `maintainer` import in wire.go is for the **TagService** wiring
(line 275: `tagGate := loadMaintainerGate(projectRoot)`), not the viewer.

---

### AC-16 — Hierarchy issues ≤ 6 extra SQL statements

**Implementation:** `fetchTagsForHierarchy` (line 1087+) iterates entity types
present and calls `AttachedTagNamesByIDs` once per type. The mock counter in
tests asserts the exact call count.

**Test output (all PASS):**
- `TestViewerService_Hierarchy_TagDecoration_BatchCallCount` — exactly 6 calls when all 6 types present
- `TestViewerService_Hierarchy_TagDecoration_SkipsEmptyEntityTypes` — fewer when types absent
- `TestViewerService_Hierarchy_TagDecoration_EmptyHierarchy_ZeroCalls` — 0 calls
- `TestViewerService_Hierarchy_TagFilter_AndSemanticsOneCallPerType` — confirms per-type batching for filter side

---

### AC-17 — `make fmt && make lint && make test` passes

**Output:**
```
$ make fmt
Formatting code...

$ make lint
0 issues.

$ make test
(no FAIL or --- FAIL lines)
ok  github.com/jwwelbor/shark-task-manager/internal/services
ok  github.com/jwwelbor/shark-task-manager/internal/api/viewer
ok  github.com/jwwelbor/shark-task-manager/internal/viewer
ok  github.com/jwwelbor/shark-task-manager/internal/viewer/server
... (all packages pass)
```

---

## Summary of Test Coverage Gaps

The following ACs have NO automated handler-layer test:

| AC | Description | Service test? | Handler test? |
|----|-------------|---------------|---------------|
| AC-01 | GET /tags happy path | YES | NO |
| AC-02 | GET /tags alphabetical | YES | NO |
| AC-03 | POST/PUT/PATCH/DELETE → 404/405 | N/A | **NO** |
| AC-08 | Hierarchy 400 with unregistered_tags shape | YES (propagation) | **NO** |
| AC-10 | FeatureTasks 400 vs 404 ordering | N/A | **NO** |
| AC-14 (handler) | Graceful response when service degraded | YES (service) | NO |

The mock scaffolding (`MockViewerServicer.TagsFunc`) was added but never used.
Search confirms zero test cases reference `/api/v1/viewer/tags`,
`UnregisteredTagError`, or `tagErrorResponse` in
`internal/api/viewer/*_test.go`.

REQ-NF-012 ("An automated test MUST assert that POST/PUT/PATCH/DELETE to
`/api/v1/viewer/tags` ... produce 404 or 405 — no write endpoint exists. This
test is a regression tripwire for REQ-F-013.") is **not satisfied**.

The QA reports for T-E28-F06-001/002/007 are PASS. There are NO QA reports for
T-E28-F06-003, T-E28-F06-004, T-E28-F06-005, or T-E28-F06-006.

---

## Process Observations (facts only)

- 5 of 7 tasks were force-advanced to `ready_for_approval` together at
  `2026-04-25T03:51` per `updated_at` timestamps in `shark list`.
- Only 3 of 7 tasks have QA reports filed in `qa_reports/`.
- Task review document (PASS) lists T-004 and T-005 as owners of AC-03, AC-08,
  AC-10 handler tests — those tests are not present in the codebase.
