---
feature_key: E28-F06-web-viewer-tag-integration
epic_key: E28
document_type: test-plan
title: E28-F06 Test Plan — Web Viewer Tag Integration
---

# E28-F06 Test Plan — Web Viewer Tag Integration

This document covers the complete test strategy for F06: wiring the E28 tag
system into the E27 web viewer. It is organized as:

1. AC Test Matrix — one row per acceptance criterion in spec.md §1.3
2. Integration Scenarios — cross-component boundary verification
3. Test Infrastructure — existing patterns and new test helpers needed

All test code must follow the project testing architecture:

- **Service tests** (`internal/services/viewer_service_test.go`) — mocked
  `TagReader` and repo dependencies; NO real database.
- **Handler tests** (`internal/api/viewer/handler_test.go`) — mocked
  `ViewerServicer`; NO real database.
- **Repository tests** — F06 adds zero SQL; the F05 repository tests already
  cover `FilterEntityIDs` and `ListTagNamesByEntities`.
- **CLI tests** — F06 touches zero CLI files; no CLI tests required.
- **UI tests** — manual UAT only (see UAT-7 in `uat-plan.md`).

Reference files:
- `internal/api/viewer/handler_test.go` — existing `MockViewerServicer` pattern
- `internal/services/viewer_service_test.go` — existing mock repo pattern
- `internal/services/tag_errors.go` — `UnregisteredTagError{Name string}`,
  `TagFilterUnavailableError`

---

## 1. AC Test Matrix

Each row maps one acceptance criterion to its test cases, expected outcomes,
and edge cases. AC IDs come from spec.md §1.3.

---

### AC-01 — Empty vocabulary returns `{"tags":[]}`

**Requirement trace:** REQ-F-001

**Test type:** Service unit test + handler unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC01-1: Service `Tags()` with empty vocabulary | `MockTagReader.ListTagsFunc` returns `[]*models.Tag{}` | Returns `&TagsResponse{Tags: []TagDTO{}}` — non-nil slice, length 0 |
| TC-AC01-2: Handler GET happy path, empty vocabulary | Mock service `Tags()` returns `TagsResponse{Tags: []TagDTO{}}` | HTTP 200, body `{"tags":[]}` — `tags` key present and is JSON array, NOT null |
| TC-AC01-3: JSON null guard | The `[]TagDTO{}` zero-value (not nil) check | `json.Marshal(TagsResponse{Tags: nil})` gives `{"tags":null}` — test MUST assign `[]TagDTO{}`, not `nil` |

**Edge cases:**
- `tags` field must serialize as `[]` not `null` when vocabulary is empty (ADR-F06-2).
- When `tagSvc` is nil (graceful degradation), `Tags()` must also return `{"tags":[]}` with no error — covered in AC-14.

---

### AC-02 — Populated vocabulary returns names alphabetically

**Requirement trace:** REQ-F-001

**Test type:** Service unit test + handler unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC02-1: Service `Tags()` alphabetical ordering | `MockTagReader.ListTagsFunc` returns tags `[voice, auth]` (unordered) | Returns `TagsResponse.Tags` with `auth` before `voice` (sort ascending by name) |
| TC-AC02-2: Handler GET populated vocabulary | Mock service returns `TagsResponse{Tags: [{Name:"auth"},{Name:"voice"}]}` | HTTP 200, body `{"tags":[{"name":"auth"},{"name":"voice"}]}` |
| TC-AC02-3: Delegation | Service calls `tagSvc.ListTags(ctx)` exactly once | Mock counter asserts one `ListTags` call; no other tag repo methods called |

**Edge cases:**
- Tags with identical names — impossible (vocabulary uniqueness constraint), but mock should still return deterministic results.
- Single-tag vocabulary: `[{"name":"auth"}]` — no sorting bug.

---

### AC-03 — Non-GET methods to `/tags` return 404 or 405

**Requirement trace:** REQ-F-013, REQ-NF-012

**Test type:** Handler unit test (HTTP method assertion — security regression tripwire)

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC03-1: POST to `/api/v1/viewer/tags` | `httptest.NewRequest("POST", "/api/v1/viewer/tags", body)` | HTTP 404 or 405 (either is valid per ServeMux behavior); body does NOT parse as a tag creation response |
| TC-AC03-2: PUT to `/api/v1/viewer/tags` | `httptest.NewRequest("PUT", "/api/v1/viewer/tags", body)` | HTTP 404 or 405 |
| TC-AC03-3: PATCH to `/api/v1/viewer/tags` | `httptest.NewRequest("PATCH", "/api/v1/viewer/tags", body)` | HTTP 404 or 405 |
| TC-AC03-4: DELETE to `/api/v1/viewer/tags` | `httptest.NewRequest("DELETE", "/api/v1/viewer/tags", nil)` | HTTP 404 or 405 |
| TC-AC03-5: No-mutation side-effect guard | Mock `ViewerServicer.Tags()` NOT called for non-GET methods | `TagsFunc` call counter stays at 0 after POST/PUT/PATCH/DELETE |

**Edge cases:**
- The test must assert that the actual service `TagsFunc` is NOT invoked on mutation attempts.
- Test both `/api/v1/viewer/tags` and `/api/v1/viewer/tags/somename` paths.

---

### AC-04 — Hierarchy response always carries non-null `tags: []` on all entity DTOs

**Requirement trace:** REQ-F-003

**Test type:** Service unit test + handler unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC04-1: Hierarchy with no tags wired (service nil tagSvc) | Service constructed WITHOUT `WithTagService`; mock repos return epics/features/tasks/bugs/changes/ideas | Every DTO's `Tags` field is `[]string{}` (not nil); JSON encodes as `[]` not `null` |
| TC-AC04-2: Hierarchy with tagSvc wired but entities untagged | `MockTagReader.AttachedTagNamesByIDsFunc` returns empty `map[int64][]string{}` | Every DTO `Tags` is `[]string{}` |
| TC-AC04-3: Handler JSON field presence check | Mock service `HierarchyFunc` returns response with all DTOs having `Tags: []string{}` | HTTP 200; JSON parsed response has `epics[0].tags == []`, `epics[0].features[0].tags == []`, `epics[0].features[0].tasks[0].tags == []`, `bugs[0].tags == []`, `change_cards[0].tags == []`, `ideas[0].tags == []` |
| TC-AC04-4: Go nil vs empty slice marshal guard | Test that `var s []string; json.Marshal(s)` gives `null` while `s := []string{}; json.Marshal(s)` gives `[]` | Documents why the service MUST assign `[]string{}` not leave nil |

**Edge cases:**
- `ActivityRecord` DTOs must NOT have a `Tags` field added (REQ-F-007: excluded from requirement).
- `SummaryEntityCounts` DTOs must NOT have a `Tags` field (REQ-F-006: counts-shaped, not entity-shaped).

---

### AC-05 — Tagged entities show correct tag names in hierarchy response

**Requirement trace:** REQ-F-003, UAT-7

**Test type:** Service unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC05-1: Single entity type tagged | One epic ID=1 tagged `voice`; mock `AttachedTagNamesByIDs` returns `{1: ["voice"]}` | `HierarchyEpic.Tags == ["voice"]`; all other epics have `Tags == []` |
| TC-AC05-2: All six entity types tagged | Mock epics/features/tasks/bugs/changes/ideas each have one entity with `voice` tag | Each DTO type carries `["voice"]`; untagged entities carry `[]` |
| TC-AC05-3: Multiple tags on one entity | Mock returns `{epicID: ["auth", "voice"]}` for one epic | `HierarchyEpic.Tags == ["auth", "voice"]` (order matches what `AttachedTagNamesByIDs` returns) |
| TC-AC05-4: Batching correctness | Hierarchy has 3 epics, 5 features, 10 tasks, 2 bugs, 2 changes, 1 idea | Service calls `AttachedTagNamesByIDs` exactly 6 times (once per entity type) — mock call counter asserts |

**Edge cases:**
- `AttachedTagNamesByIDs` returns nil for an ID → service must substitute `[]string{}`.
- One entity type has zero entities → no `AttachedTagNamesByIDs` call for that type (call count ≤ 6, not exactly 6).

---

### AC-06 — `?tag=voice` hierarchy prunes unmatched subtrees

**Requirement trace:** REQ-F-010, UAT-7

**Test type:** Service unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC06-1: Basic tree pruning | Epic E1 has feature F1(tagged `voice`) and F2(not tagged). Epic E2 has no tagged features. `EntityIDsByTags` returns IDs for F1 and F1's tasks. | Result contains E1 with only F1; E2 is pruned; F2 is pruned |
| TC-AC06-2: Feature-tagged but task-untagged | Feature is tagged `voice`, its tasks are not. | Feature remains in result (it matches directly); tasks under it follow same pruning (only tasks also tagged survive, but feature itself stays) |
| TC-AC06-3: Epic with matching direct tag, zero matching features | Epic is tagged `voice`, none of its features/tasks are. | Epic appears in result (it matched directly) with empty features slice |
| TC-AC06-4: Flat entities independently filtered | Hierarchy has bugs B1(tagged), B2(untagged). Feature tree has no matches. | `bugs` list contains only B1; `change_cards` and `ideas` similarly filtered independently |
| TC-AC06-5: Empty result when no matches | No entities tagged `voice`. | Returns `HierarchyResponse` with empty `epics`, `bugs`, `change_cards`, `ideas` slices |

**Edge cases:**
- Pruning must not mutate the pre-filter result in-place if the data is shared (use copies or return-by-value approach).
- Pruned epics: epic whose ALL features are pruned AND epic itself lacks direct tag → epic removed.

---

### AC-07 — `?tag=voice&tag=auth` applies AND semantics

**Requirement trace:** REQ-F-010, Epic Architecture ADR-5

**Test type:** Service unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC07-1: AND intersection | Entity has `voice` only. `EntityIDsByTags("voice AND auth")` returns empty set. | Entity absent from result |
| TC-AC07-2: Both tags present | Entity has both `voice` AND `auth`. Mock returns entity ID in both sets. | Entity present in result with `Tags: ["auth", "voice"]` |
| TC-AC07-3: Service calls EntityIDsByTags once per filter-tag-set | Two-tag filter `[voice, auth]` | `EntityIDsByTags` called with the full `[]string{"voice","auth"}` slice in ONE call; mock records the exact args |

**Edge cases:**
- `EntityIDsByTags` is called with `TagQueryOpAnd`; test asserts the op value on the mock call.
- Single-tag filter is still AND logic (degenerate case, same code path).

---

### AC-08 — Unregistered tag in hierarchy filter returns 400

**Requirement trace:** REQ-F-011

**Test type:** Handler unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC08-1: Handler catches UnregisteredTagError | `HierarchyFunc` in mock returns `&services.UnregisteredTagError{Name: "does-not-exist"}` | HTTP 400; body has `error`, `message`, and `unregistered_tags` fields |
| TC-AC08-2: `unregistered_tags` array field | Same as above | `unregistered_tags: ["does-not-exist"]` in JSON response body |
| TC-AC08-3: 500 not returned for unregistered tag | Same as above | HTTP status code is 400, NOT 500 |
| TC-AC08-4: Unknown service error still returns 500 | `HierarchyFunc` returns `fmt.Errorf("db error")` (not an `*UnregisteredTagError`) | HTTP 500 (existing error path unchanged) |

**Edge cases:**
- `errors.As(err, &unregErr)` type assertion — test both wrapped and unwrapped `UnregisteredTagError`.
- `unregistered_tags` must be a JSON array, never null.

---

### AC-09 — `/features/{key}/tasks?tag=voice` filters and Total reflects pre-pagination count

**Requirement trace:** REQ-F-005, REQ-F-008

**Test type:** Service unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC09-1: Tag filter applied before pagination | Feature has 10 tasks; 3 are tagged `voice`. `opts.Limit=2, opts.Tags=["voice"]`. `EntityIDsByTags` returns 3 IDs. | Response `Total == 3` (not 10); `Tasks` has 2 items (page 1 of 3-item set) |
| TC-AC09-2: No tag filter — unchanged behavior | `opts.Tags` empty; same 10-task feature | Response `Total == 10`; behavior identical to pre-F06 |
| TC-AC09-3: Post-pagination decoration | With 3-task result page (after filter+paginate) | `AttachedTagNamesByIDs` called with IDs of the 2-item page, NOT the 10-item full set OR the 3-item pre-pagination set |
| TC-AC09-4: ID list for decoration is page-scoped | `opts.Limit=2, opts.Offset=1` (page 2) on filtered result of 3 | Decoration called with IDs of 2 tasks on page 2 (not all 3 filtered IDs) |

**Edge cases:**
- Empty page after filter: `Total == 0`, `Tasks == []`, no `AttachedTagNamesByIDs` call needed.
- Tag filter + status filter together: tag filter runs first on the full feature task list, then status/agent/blocked filters apply to the tag-filtered subset.

---

### AC-10 — Feature task endpoint returns 400 for unregistered tag, 404 for missing feature

**Requirement trace:** REQ-F-012

**Test type:** Handler unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC10-1: Feature exists, tag unregistered → 400 | `FeatureTasksFunc` returns `*services.UnregisteredTagError{Name:"ghost"}` | HTTP 400 with `unregistered_tags: ["ghost"]` |
| TC-AC10-2: Feature missing → 404 | `FeatureTasksFunc` returns a "not found" error (feature lookup fails) | HTTP 404 (existing behavior unchanged) |
| TC-AC10-3: Feature missing takes precedence over tag check | Feature not found (isNotFound returns true) | HTTP 404, not 400; feature lookup runs BEFORE tag validation |
| TC-AC10-4: 400 NOT 404 when feature exists | Feature found but tag unregistered | HTTP 400 confirmed (not 404) |

**Edge cases:**
- Feature 404 and tag 400 are mutually exclusive — feature lookup is first.
- `isNotFound` helper (existing in handler.go) must NOT be called for `UnregisteredTagError`.

---

### AC-11 — UI renders tag chips on entity detail views, no chips when untagged

**Requirement trace:** REQ-F-016

**Test type:** Manual UAT (no automated test for UI in v1)

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC11-1 (manual): Tagged entity detail | Entity with `["voice", "auth"]` tags | UI shows two `.tag-chip` elements with text `voice` and `auth` in the entity header area |
| TC-AC11-2 (manual): Untagged entity detail | Entity with `tags: []` | NO `.tag-chip` elements; NO "Tags:" label rendered |
| TC-AC11-3 (manual): Six entity types covered | Tag-chip rendering on epic detail, feature detail, task detail, bug detail, change-card detail, idea detail | All six entity types show chips when tagged |

**Note:** Automated test coverage via `internal/viewer/assets_test.go` verifies that `viewer.html` is correctly served. F06 does not change that invariant. UI rendering assertions are manual only per spec.md §2.6.

---

### AC-12 — UI tag filter control populates from `GET /viewer/tags` and triggers refetch

**Requirement trace:** REQ-F-017

**Test type:** Manual UAT

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC12-1 (manual): Filter control populated on load | Viewer opened with two registered tags (`auth`, `voice`) | Filter control shows two chips: `auth` and `voice` |
| TC-AC12-2 (manual): Chip selection triggers refetch | User clicks `voice` chip in the filter | Hierarchy list refetches with `?tag=voice`; list narrows |
| TC-AC12-3 (manual): Multiple chips selected | User clicks both `auth` and `voice` chips | List refetches with `?tag=auth&tag=voice` (AND semantics) |
| TC-AC12-4 (manual): Empty vocabulary state | `GET /viewer/tags` returns `{"tags":[]}` | Filter control renders empty-state text "No tags registered yet" (disabled state) |
| TC-AC12-5 (manual): Feature-tasks list refetched on chip toggle (UAT regression — T-E28-F06-007) | (a) Open a feature whose Overview Tasks table contains tasks with mixed tags. (b) Toggle a chip (e.g., `voice`). (c) Toggle the chip off again. | (b) Network panel shows a request to `GET /api/v1/viewer/features/{key}/tasks?tag=voice` (in addition to the hierarchy refetch with `?tag=voice`); the Tasks table updates to show only tasks tagged `voice`. (c) Network panel shows `GET /api/v1/viewer/features/{key}/tasks` with NO `?tag=` params; the Tasks table reverts to the unfiltered list. Repeated `?tag=a&tag=b` semantics apply when more than one chip is selected (AC-T3). |

---

### AC-13 — No mutation controls in the UI

**Requirement trace:** REQ-F-018, UAT-7

**Test type:** Manual UAT (visual/DOM scan)

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC13-1 (manual): No add-tag control | Full UI inspection | No "+" button, no text input for adding tags, no form with tag name field |
| TC-AC13-2 (manual): No rename/remove controls | Full UI inspection | No rename, edit, or delete affordances on vocabulary chips |
| TC-AC13-3 (manual): CLI hint if present | Any affordance pointing to CLI | If any hint exists, it shows the CLI command string, NOT a form |

---

### AC-14 — Graceful degradation when TagReader not wired

**Requirement trace:** REQ-F-015, REQ-NF-011

**Test type:** Service unit test + handler unit test

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC14-1: `Tags()` with nil tagSvc | `ViewerService` constructed WITHOUT `WithTagService(...)` | Returns `&TagsResponse{Tags: []TagDTO{}}`, no error |
| TC-AC14-2: `Hierarchy()` with nil tagSvc | Same service; opts.Tags empty | All DTOs carry `Tags: []string{}`; zero `AttachedTagNamesByIDs` calls |
| TC-AC14-3: `Hierarchy()` with opts.Tags set, nil tagSvc | `opts.Tags = ["voice"]` | Tag filter silently ignored (no error, no panic); DTOs carry `Tags: []string{}` |
| TC-AC14-4: `FeatureTasks()` with nil tagSvc, opts.Tags set | `opts.Tags = ["voice"]` | Tag filter silently ignored; tasks carry `Tags: []string{}`; no panic |
| TC-AC14-5: No per-request error logging | nil-tagSvc requests | Mock logger (or log output capture) shows NO error-level logs; at most one debug log at construction |
| TC-AC14-6: Handler GET tags — empty response | Mock service `Tags()` returns `TagsResponse{Tags: []TagDTO{}}` | HTTP 200 with body `{"tags":[]}` (not 500) |

**Edge cases:**
- Service MUST NOT panic when `tagSvc` is nil and any viewer endpoint is called.
- Per REQ-NF-011, NO errors are logged per-request when degraded.

---

### AC-15 — `wire.go` does not pass MaintainerGate to ViewerService

**Requirement trace:** REQ-F-014

**Test type:** Code inspection test (automated grep / compile-time assertion)

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC15-1: wire.go grep | `grep -n "MaintainerGate\|maintainer.Gate\|maintainer.New" internal/viewer/server/wire.go` | Zero matches — no gate reference in the viewer wiring |
| TC-AC15-2: ViewerService constructor signature | Inspect `NewViewerService` signature and all setter methods | No parameter of type `*maintainer.Gate` or any maintainer interface |
| TC-AC15-3: Compile-time — no maintainer import in viewer service | `grep "maintainer" internal/services/viewer_service.go` | Zero matches |

**Note:** This is verifiable at code-review time on the PR. The spec marks this as a defense-in-depth invariant (ADR-F06-3). Include a test assertion in `wire_test.go` that the viewer service wiring compiles without a gate parameter.

---

### AC-16 — Hierarchy issues at most 6 extra SQL statements (via mock call counting)

**Requirement trace:** REQ-NF-001

**Test type:** Service unit test (mock call counter)

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC16-1: All six entity types present | Hierarchy tree has epics, features, tasks, bugs, changes, ideas | `AttachedTagNamesByIDs` call counter == 6 (exactly one per type) |
| TC-AC16-2: Only three entity types present | Hierarchy has epics and features (no tasks/bugs/changes/ideas) | `AttachedTagNamesByIDs` call counter == 2 (only called for types that have IDs) |
| TC-AC16-3: Empty hierarchy | No epics, no flat entities | `AttachedTagNamesByIDs` call counter == 0 |
| TC-AC16-4: Large tree does NOT increase call count | Mock 100 epics, 500 features, 5000 tasks | Call counter == 3 (one per type) — verifies O(1) not O(n) SQL |

**Edge cases:**
- Mock counter must be reset between subtests.
- `EntityIDsByTags` calls are separate (for filter, not decoration) — count them independently.

---

### AC-17 — `make fmt && make lint && make test` passes cleanly

**Requirement trace:** Quality Gate (`.claude/rules/development-workflows.md`)

**Test type:** CI gate (not a unit test)

| Test Case | Input / Setup | Expected Outcome |
|---|---|---|
| TC-AC17-1: No compilation errors | F06 implementation merged | `go build ./...` exits 0 |
| TC-AC17-2: No lint failures | F06 changes | `make lint` exits 0 |
| TC-AC17-3: No test regressions | All existing tests | `make test` exits 0; zero previously-passing tests now fail |
| TC-AC17-4: Viewer-package tests pass | `go test ./internal/api/viewer/...` | All tests pass |
| TC-AC17-5: Service-package tests pass | `go test ./internal/services/...` | All tests pass |

---

## 2. Integration Scenarios

These scenarios verify correct behavior at component boundaries. Each maps to
one or more UAT scenarios from `uat-plan.md`.

---

### IS-1: Service ↔ TagReader interface boundary

**Components:** `ViewerService` ↔ `TagReader` (which `*services.TagService` satisfies)

**What to verify:**
- `ViewerService.Tags()` calls `TagReader.ListTags(ctx)` and correctly projects `*models.Tag` → `TagDTO{Name}`.
- `ViewerService.Hierarchy()` calls `TagReader.EntityIDsByTags(ctx, entityType, names, TagQueryOpAnd)` with the AND operator constant (not OR).
- `ViewerService.Hierarchy()` calls `TagReader.AttachedTagNamesByIDs(ctx, entityType, ids)` with the correct entity type constant per entity DTO type (`EntityTypeEpic`, `EntityTypeFeature`, `EntityTypeTask`, `EntityTypeBug`, `EntityTypeChange`, `EntityTypeIdea`).
- `ViewerService.FeatureTasks()` calls `EntityIDsByTags` with `EntityTypeTask` only.

**Tests referencing this boundary:** TC-AC02-3, TC-AC05-4, TC-AC07-3, TC-AC09-3

**Epic UAT scenarios satisfied:** UAT-7 (step 5: set of entities matches CLI output)

---

### IS-2: Handler ↔ Service interface boundary (ViewerServicer)

**Components:** `ViewerHandler` ↔ `ViewerServicer` interface

**What to verify:**
- `ViewerHandler.Tags()` invokes `h.svc.Tags(r.Context())` — no extra parameters.
- `ViewerHandler.Hierarchy()` passes `services.HierarchyOptions{Tags: parseTagsQuery(r)}` correctly — interface signature change from `Hierarchy(ctx)` to `Hierarchy(ctx, opts HierarchyOptions)` is tested by the mock.
- `ViewerHandler.FeatureTasks()` passes `opts.Tags` field in the `FeatureTaskOptions` struct.
- `respondUnregisteredTagError()` helper is invoked (not generic 500) when error is `*services.UnregisteredTagError`.

**Tests referencing this boundary:** TC-AC08-1, TC-AC08-4, TC-AC10-1, TC-AC10-2

**Epic UAT scenarios satisfied:** UAT-7, UAT-INT-3

---

### IS-3: Handler query-parameter parsing boundary

**Components:** `parseTagsQuery()` helper ↔ HTTP request

**What to verify:**
- `?tag=voice` → `["voice"]`
- `?tag=voice&tag=auth` → `["voice", "auth"]`
- `?tag=&tag=voice&tag=%20%20` (blank + whitespace-only) → `["voice"]` (blanks dropped)
- `?tag=  Voice  ` (leading/trailing whitespace) → `["Voice"]` (trimmed but NOT lowercased — normalization is service-side)

**Tests:** Standalone unit tests for `parseTagsQuery` function OR inline assertions in handler tests.

**Reference to spec:** REQ-F-009, REQ-NF-005

---

### IS-4: Wiring ↔ ViewerService (wire.go integration)

**Components:** `internal/viewer/server/wire.go` ↔ `ViewerService.WithTagService()`

**What to verify:**
- After `WireServices()`, `viewerService.tagSvc` is non-nil (the real `*services.TagService` is injected).
- `WireServices()` does NOT pass a `MaintainerGate` to `NewViewerService`.
- `wire_test.go` (existing) continues to compile and pass with F06 changes.

**Tests referencing this boundary:** TC-AC15-1, TC-AC15-2, TC-AC15-3

**Epic UAT scenarios satisfied:** UAT-8 (reusability audit)

---

### IS-5: Service ↔ `UnregisteredTagError` propagation chain

**Components:** `TagReader.EntityIDsByTags` → `ViewerService.Hierarchy/FeatureTasks` → `ViewerHandler` → HTTP 400

**What to verify:**
- `*services.UnregisteredTagError` returned from `tagSvc.EntityIDsByTags` propagates unchanged through the service to the handler.
- Handler uses `errors.As(err, &unregErr)` (not `errors.Is`or type assertion).
- The `tagErrorResponse.UnregisteredTags` field is populated from `unregErr.Name` (single string — field is `Name`, not `Names`, per `tag_errors.go` line 81).
- Handler responds 400 (not 500) for this error type.

**Important implementation note:** `UnregisteredTagError.Name` is a single `string` field. The `unregistered_tags` response array will therefore contain one element when the error originates from a single-name validation. If the spec references `err.Names`, the implementation must adapt — either wrap the single `Name` in a slice `[]string{unregErr.Name}`, or (if the vocabulary tag repo changes to return a multi-name error) update accordingly. Tests must cover this mapping explicitly.

**Tests referencing this boundary:** TC-AC08-1, TC-AC08-2, TC-AC10-1

**Epic UAT scenarios satisfied:** UAT-2 (unregistered tag → actionable error), UAT-INT-2

---

### IS-6: UI ↔ `/api/v1/viewer/tags` API boundary

**Components:** `viewer.html` JS ↔ `GET /api/v1/viewer/tags`

**What to verify (manual UAT):**
- `loadVocabulary()` JS function calls the correct endpoint path on viewer boot.
- The `{"tags":[{"name":"auth"},...]}` response is correctly parsed and rendered as chips in `renderTagFilterControl()`.
- `renderTagChips(entity)` receives `entity.tags` (a `[]string` array from the hierarchy/tasks endpoint) and creates DOM elements.
- Chip click appends `?tag=<name>` to the refetch URL.

**This boundary has no automated test in v1** (per spec.md §2.6 item 5). Covered by UAT-7.

---

### IS-7: Viewer ↔ F05 tag repository methods (inherited performance)

**Components:** `ViewerService` ↔ `TagReader` ↔ `entity_tag_repository.go` (F05 methods)

**What to verify:**
- `AttachedTagNamesByIDs` uses the `ListTagNamesByEntities` repository method (one `IN (...)` statement per call) — not N+1 per entity.
- `EntityIDsByTags` uses the `FilterEntityIDs` repository method (one statement per call) — not per-entity lookups.
- F06 does NOT add any new repository methods; it consumes the F05 ones.

**Tests referencing this boundary:** TC-AC16-1 through TC-AC16-4 (service-level mock counting); actual SQL coverage is in F05 repository tests.

**Epic UAT scenarios satisfied:** UAT-7, P-3 (viewer list endpoint latency does not regress)

---

## 3. Test Infrastructure

### 3.1 Existing test patterns to follow

| File | Pattern | Applicability to F06 |
|---|---|---|
| `internal/api/viewer/handler_test.go` | `MockViewerServicer` with `Func` fields; `httptest.NewRecorder` + `httptest.NewRequest`; route-level tests via registered `mux` | Extend with `TagsFunc` field and updated `HierarchyFunc` signature; add `Tags`, `Hierarchy-tag-filter`, and `FeatureTasks-tag-filter` test groups |
| `internal/services/viewer_service_test.go` | `mockViewerEpicRepo`, `mockViewerTaskRepo` etc. with `Func` fields; `newTestViewerService()` helper to wire up service | Add `mockTagReader` struct alongside existing mocks; extend `newTestViewerService()` to accept optional `TagReader` |
| `internal/services/tag_service_test.go` | `MockTagRepository` and `MockMaintainerGate` patterns | Reuse patterns but define `mockTagReader` as a simpler consumer interface mock (only 3 methods needed) |
| `internal/services/task_service_tags_test.go` | Demonstrates `UnregisteredTagError` assertion with `errors.As` | Copy the assertion pattern for handler tests (TC-AC08-1, TC-AC10-1) |
| `internal/viewer/server/wire_test.go` | Existing `WireServices` compile/integration test | Ensure F06 wiring change (`viewerService.WithTagService(tagSvc)`) does not break existing test; add assertion that viewer service has non-nil tagSvc after wiring |

### 3.2 New test helpers needed

#### A. `mockTagReader` (in `internal/services/viewer_service_test.go`)

No existing `MockTagReader` in viewer service tests. Must be created:

```go
type mockTagReader struct {
    ListTagsFunc              func(ctx context.Context) ([]*models.Tag, error)
    EntityIDsByTagsFunc       func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)
    AttachedTagNamesByIDsFunc func(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)
    // Call counters for AC-16 query-count assertions
    AttachedTagNamesByIDsCallCount int
}

// Methods delegate to Func fields if non-nil; return zero values otherwise.
```

This mirrors the function-field mock pattern already used throughout the codebase (`MockViewerServicer`, `mockViewerEpicRepo`, etc.).

#### B. `TagsFunc` field on `MockViewerServicer` (in `internal/api/viewer/handler_test.go`)

```go
// Add to existing MockViewerServicer struct:
TagsFunc func(ctx context.Context) (*services.TagsResponse, error)

// Add method:
func (m *MockViewerServicer) Tags(ctx context.Context) (*services.TagsResponse, error) {
    if m.TagsFunc != nil {
        return m.TagsFunc(ctx)
    }
    return nil, errors.New("TagsFunc not set in mock")
}
```

#### C. Updated `HierarchyFunc` signature in `MockViewerServicer`

The `Hierarchy` method signature changes from `Hierarchy(ctx context.Context)` to
`Hierarchy(ctx context.Context, opts services.HierarchyOptions)`. The mock must be updated:

```go
// Change:
HierarchyFunc func(ctx context.Context) (*services.HierarchyResponse, error)
// To:
HierarchyFunc func(ctx context.Context, opts services.HierarchyOptions) (*services.HierarchyResponse, error)
```

This is a compile-time breaking change in the mock; all existing tests using `HierarchyFunc` must be updated to match the new signature even if they ignore `opts`.

#### D. `tagErrorResponse` struct assertion helpers (in `internal/api/viewer/handler_test.go`)

Helper to parse and assert the 400 error response shape:

```go
type tagErrorResponse struct {
    Error            string   `json:"error"`
    Message          string   `json:"message"`
    UnregisteredTags []string `json:"unregistered_tags"`
}

func assertTagErrorResponse(t *testing.T, body []byte, expectedNames []string) {
    t.Helper()
    var resp tagErrorResponse
    if err := json.Unmarshal(body, &resp); err != nil {
        t.Fatalf("failed to parse tag error response: %v", err)
    }
    if resp.Error == "" { t.Error("error field missing") }
    if resp.Message == "" { t.Error("message field missing") }
    if len(resp.UnregisteredTags) != len(expectedNames) {
        t.Errorf("unregistered_tags length: got %d want %d", len(resp.UnregisteredTags), len(expectedNames))
    }
}
```

### 3.3 No repository tests needed

F06 adds zero SQL and zero repository methods. The existing F05 repository tests
(`internal/repository/tag/entity_tag_repository_test.go`) already cover:

- `FilterEntityIDs` (used by `EntityIDsByTags`)
- `ListTagNamesByEntities` (used by `AttachedTagNamesByIDs`)

### 3.4 No CLI tests needed

F06 touches zero CLI files (`internal/cli/commands/`). Confirmed by spec.md §1.3 AC-17
and §2.6 item 4.

### 3.5 No new database migration

Per REQ-F-020 / spec.md §2.7: `CurrentSchemaVersion` stays at 14. No `skip_migrations: false`
toggle required for F06.

---

## 4. Test Execution

```bash
# Run all affected test packages
go test ./internal/api/viewer/... -v
go test ./internal/services/... -v -run TestViewerService

# Run the full quality gate (mandatory before declaring F06 complete)
make fmt && make lint && make test
```

---

## 5. Exit Gate Verification

| Exit-Gate Criterion | Status |
|---|---|
| Every AC in spec.md §1.3 has at least one test case | AC-01 through AC-17 each have explicit test cases above |
| Edge cases identified for each AC | Documented in each AC row |
| Integration scenarios cover cross-component boundaries | IS-1 through IS-7 cover all interacting component pairs |
| Test patterns reference existing infrastructure | §3.1 maps each pattern to its source file |

---

*Last Updated: 2026-04-24*
