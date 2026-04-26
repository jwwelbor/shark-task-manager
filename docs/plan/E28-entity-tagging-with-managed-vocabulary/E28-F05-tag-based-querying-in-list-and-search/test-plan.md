---
feature_key: E28-F05-tag-based-querying-in-list-and-search
epic_key: E28
document_type: test-plan
title: Test Plan — Tag-Based Querying in List and Search (E28-F05)
---

# E28-F05 — Test Plan

Traceability: every test case below maps to at least one Acceptance Criterion
(AC-1..31) in `./spec.md` §1.3, and every AC has at least one test here. AC
traces that feed into epic-level UAT scenarios in `../uat-plan.md` are
cross-referenced in §2.

Testing rules in force (see `.claude/rules/testing/architecture.md`):
- **Repository tests** use the real DB via `test.NewIsolatedTestDB(t)` (fresh
  isolated DB per test — no DELETE boilerplate needed). Pattern is established
  in `internal/repository/tag/entity_tag_repository_test.go`.
- **Service tests** use mocked repositories and a new `MockTagQueryService`
  (extended from F04's `MockTagService`). No real DB.
- **CLI tests** mock entity services and the tag service. No real DB.
  In-process Cobra invocation with stdout/stderr capture follows the pattern
  in `internal/cli/commands/tags_test.go`.

---

## 1. AC Test Matrix

### 1.1 New TagService query methods — `EntityIDsByTags` (AC-1..AC-6)

All tests in `internal/services/tag_service_test.go`, kind: Unit (mock repos).

The `mockEntityTagRepo` in `tag_service_test.go` gains two new function
fields: `filterEntityIDsFn` and `listTagNamesByEntitiesFn`.

| AC | Test Name | Input / Setup | Expected Outcome |
|---|---|---|---|
| AC-1 | `TestEntityIDsByTags_SingleTagMatchesSome` | `entityType=Task`, `names=["voice"]`; mock `getByNameFn` returns `{ID:3, Name:"voice"}`; mock `filterEntityIDsFn` returns `[42,43,44]` | Returns `[42,43,44]` sorted. `FilterEntityIDs` called once with `tagIDs=[3]`. |
| AC-2 | `TestEntityIDsByTags_TwoTagsIntersection` | `names=["voice","auth"]`; mock returns IDs 3 and 7; `filterEntityIDsFn` returns `[10]` (AND intersection) | Returns `[10]`. `FilterEntityIDs` called once with `tagIDs=[3,7]`. |
| AC-2b | `TestEntityIDsByTags_TwoTagsNoIntersection` | Same setup but `filterEntityIDsFn` returns `[]` | Returns `(nil,nil)` — wait, returns `([]*empty*,nil)`. Returns non-nil empty slice per REQ-F-006 semantics. Returns `[]int64{}` with `nil` error. |
| AC-3 | `TestEntityIDsByTags_NilNamesReturnsNilNil` | `names=nil` | Returns `(nil, nil)`. Zero repo calls. |
| AC-3b | `TestEntityIDsByTags_EmptyNamesReturnsNilNil` | `names=[]string{}` | Returns `(nil, nil)`. Zero repo calls. |
| AC-4 | `TestEntityIDsByTags_UnregisteredNameAborts` | `names=["voice","does-not-exist"]`; `getByNameFn` returns `ErrTagNotFound` for `"does-not-exist"` | Returns `(nil, *UnregisteredTagError{Name:"does-not-exist"})`. `filterEntityIDsFn` call count is 0. |
| AC-5 | `TestEntityIDsByTags_NameNormalization` | `names=["Voice "]` (leading/trailing whitespace, mixed case) | `getByNameFn` invoked with `"voice"` (normalized). Behaves identically to AC-1 with one matching tag. |
| AC-6 | `TestEntityIDsByTags_DuplicateNamesDeduped` | `names=["voice","voice"]` | `getByNameFn` called twice (the service normalizes then deduplicates before calling `FilterEntityIDs`). `filterEntityIDsFn` called exactly once with a single-element `tagIDs=[3]` slice. |

Edge cases:

| AC | Test Name | Input / Setup | Expected Outcome |
|---|---|---|---|
| AC-4b | `TestEntityIDsByTags_UnregisteredNameFirstPos` | Unregistered name is first in slice | Same: `*UnregisteredTagError` returned, zero `FilterEntityIDs` calls. |
| AC-4c | `TestEntityIDsByTags_AllNamesInvalid` | `names=["noexist1","noexist2"]` | Returns error on first invalid name (fail-fast). |

### 1.2 Repository layer — `FilterEntityIDs` and `ListTagNamesByEntities` (AC-7..AC-8)

All tests in `internal/repository/tag/entity_tag_repository_test.go`, kind:
Integration (real DB via `test.NewIsolatedTestDB(t)`). New test functions are
added to the existing file alongside the existing `TestEntityTagRepository_*`
tests.

Each test uses `setupEntityTagRepo(t)` (already defined in that file).
Tags and entities are seeded fresh per test via the helpers already in use
(`insertTestEpic`, `tagRepo.Create`, `entityTagRepo.Attach`).

| AC | Test Name | Input / Setup | Expected Outcome |
|---|---|---|---|
| AC-7a | `TestFilterEntityIDs_SingleTagSingleMatch` | Seed 3 tasks; attach tag `id=T1` to task IDs `[10,20]` only. Call `FilterEntityIDs(ctx, EntityTypeTask, []int64{T1.ID})`. | Returns `[10,20]` sorted ascending. |
| AC-7b | `TestFilterEntityIDs_TwoTagsBothRequired` | Seed 4 tasks; task 10 has tags `{T1,T2}`, tasks 20/30 have `{T1}` only, task 40 has `{T2}` only. Call with `tagIDs=[T1.ID,T2.ID]`. | Returns `[10]` only. |
| AC-7c | `TestFilterEntityIDs_NoMatchReturnsEmpty` | Tags exist, no task has BOTH tags. | Returns `[]int64{}` (non-nil empty), no error. |
| AC-7d | `TestFilterEntityIDs_EntityTypeScopedCorrectly` | Same tag attached to a task (ID 10) and an epic (ID 10). Call with `EntityTypeTask`. | Returns only task ID `[10]`; epic ID is not returned. |
| AC-8 | `TestFilterEntityIDs_EmptySliceReturnsErrEmptyTagIDs` | Call `FilterEntityIDs(ctx, EntityTypeTask, []int64{})`. | Returns `ErrEmptyTagIDs` sentinel. Does not panic. |
| AC-8b | `TestFilterEntityIDs_ThreeTagsAnd` | 5 tasks; task 10 has all three tags, tasks 20/30 have two each (no task 20 or 30 has all three). | Returns `[10]`. |

`ListTagNamesByEntities` tests:

| Test Name | Input / Setup | Expected Outcome |
|---|---|---|
| `TestListTagNamesByEntities_SingleEntity` | Task 10 has tags `["auth","voice"]` (IDs resolved from real vocab). Call `ListTagNamesByEntities(ctx, EntityTypeTask, []int64{10})`. | Returns `[{EntityID:10, TagName:"auth"}, {EntityID:10, TagName:"voice"}]` ordered by entity_id ASC then name ASC. |
| `TestListTagNamesByEntities_MultipleEntities` | Tasks 10 and 20 with different tag sets. | Returns rows interleaved by entity_id ASC, name ASC. All input IDs present in output (even those with zero attachments). |
| `TestListTagNamesByEntities_EmptyInputReturnsEmpty` | `entityIDs=[]int64{}`. | Returns `[]EntityIDTagName{}` (non-nil), no DB call. |
| `TestListTagNamesByEntities_EntityWithNoTagsOmitted` | Task 30 exists but has zero attachments. Input `[10,30]`. | Result contains rows for task 10 only; task 30 has no rows (method does NOT guarantee an entry per input ID — only returns actual attachment rows). |

### 1.3 New TagService methods — `ListTagsForEntity` and `AttachedTagNamesByIDs` (AC-9..AC-10)

Tests in `internal/services/tag_service_test.go`, kind: Unit (mock repos).

| AC | Test Name | Input / Setup | Expected Outcome |
|---|---|---|---|
| AC-9 | `TestListTagsForEntity_TwoAttachments` | Mock `ListByEntity` returns `[{TagID:7},{TagID:3}]`; mock `GetByID` returns `"auth"` for 7 and `"voice"` for 3. | Returns `["auth","voice"]` sorted ascending. |
| AC-9b | `TestListTagsForEntity_NoAttachments` | Mock `ListByEntity` returns `[]`. | Returns `[]string{}` (non-nil empty slice), no error. |
| AC-9c | `TestListTagsForEntity_GetByIDError` | Mock `GetByID` returns error on one of the IDs. | Error propagated; returns `(nil, wrappedError)`. |
| AC-10 | `TestAttachedTagNamesByIDs_FullMatrix` | `entityIDs=[10,20,30]`; `listTagNamesByEntitiesFn` returns `[(10,"auth"),(10,"voice"),(20,"voice")]`. | Returns `map[int64][]string{10:["auth","voice"], 20:["voice"], 30:[]}`. Every input ID present in map. |
| AC-10b | `TestAttachedTagNamesByIDs_EmptyInput` | `entityIDs=[]int64{}`. | Returns non-nil empty map `map[int64][]string{}`, no error, no repo call. |
| AC-10c | `TestAttachedTagNamesByIDs_AllSameEntity` | All rows have same `EntityID`. | Map has one key with all tag names sorted. |

### 1.4 Entity service List filter (AC-11..AC-16)

Tests in `internal/services/{task,feature,epic,bug,change_card,idea}_service_test.go`
and new `*_service_tags_test.go` sibling files as appropriate. Kind: Unit
(mock repos + extended `MockTagService`).

The existing `MockTagService` in `internal/services/mock_tag_service_test.go`
gains three new methods and function fields:

```
EntityIDsByTagsFn    func(ctx, entityType, names, op) ([]int64, error)
ListTagsForEntityFn  func(ctx, entityType, entityID) ([]string, error)
AttachedTagNamesByIDsFn func(ctx, entityType, entityIDs) (map[int64][]string, error)
```

These must satisfy a new narrow interface `TagQuerier` (in
`internal/services/tag_service.go` alongside `TagAttacher`) which entity
services use for the new List/Get query paths.

**TaskService** (AC-11..AC-15 are the canonical case; multiply ×7 for AC-16):

| AC | Test Name | Input / Setup | Expected Outcome |
|---|---|---|---|
| AC-11 | `TestListTasks_TagFilterReturnsMatchingSubset` | Mock `EntityIDsByTags` returns `[101,102]`; mock base `List` returns 5 tasks `[{ID:101,...},{ID:102,...},{ID:103,...},{ID:104,...},{ID:105,...}]`. `filters.Tags=["voice"]`. | Service intersects: returns tasks 101 and 102 only. |
| AC-12 | `TestListTasks_TagPlusStatusFilter` | `filters={Tags:["voice"], Status:"in_progress"}`; `EntityIDsByTags` returns `[101,102]`; base List (already filtered by status) returns `[{ID:101,Status:"in_progress"},{ID:103,Status:"in_progress"}]`. | Intersection: only task 101 (in both ID sets). |
| AC-13 | `TestListTasks_TwoTagsAndSemantics` | `filters.Tags=["voice","auth"]`; mock `EntityIDsByTags` returns `[101]` (AND intersection computed in TagService layer). | Returns `[task 101]` only. |
| AC-14 | `TestListTasks_TagFilterZeroMatches` | `EntityIDsByTags` returns `[]int64{}`. | Returns `[]*models.Task{}` (non-nil empty). Base List NOT called (short-circuit per REQ-F-017). |
| AC-15 | `TestListTasks_UnregisteredTagPropagatesError` | `EntityIDsByTags` returns `(nil, *UnregisteredTagError{Name:"does-not-exist"})`. | Service propagates error unchanged. |
| AC-30 | `TestListTasks_NilTagSvcWithTagsFilter` | `tagSvc=nil`, `filters.Tags=["voice"]`. | Returns `*TagFilterUnavailableError` with message `tag filtering is not available (TagService not wired)`. |
| AC-30b | `TestListTasks_NilTagSvcWithNilFilter` | `tagSvc=nil`, `filters.Tags=nil`. | Returns results normally (no tag-filter path invoked). |

**AC-16 (×7 entity types):** Analogous test exists for each of the remaining
six entities (Epic, Feature via `ListFeatures` and `ListFeaturesByEpicKey`,
Bug, ChangeCard, Idea). Test names follow the template
`TestList<Entity>_TagFilter*`. Because the filter path is identical for each,
one complete suite is written for `TaskService` (above) and then one
representative happy-path test (`TestList<Entity>_TagFilterReturnsMatchingSubset`)
is added for each of the other six.

### 1.5 SearchService tag post-filter (AC-17..AC-19)

Tests in `internal/services/search_service_test.go`. The existing
`mockSearchRepository` stays unchanged. `SearchService` gains a `tagSvc
TagQuerier` optional dependency (nil gracefully degraded).

| AC | Test Name | Input / Setup | Expected Outcome |
|---|---|---|---|
| AC-17 | `TestSearchAll_TagFilterReducesResults` | `SearchAll` returns 3 results: `[task:E07-F01-001, task:E07-F01-002, bug:B001]`; `EntityIDsByTags(EntityTypeTask,["voice"])` returns `[<id of E07-F01-001>]`; `EntityIDsByTags(EntityTypeBug,["voice"])` returns `[]`. `tags=["voice"]`. | Returns only `[task:E07-F01-001]`. Two `EntityIDsByTags` calls (one per entity type bucket). |
| AC-17b | `TestSearchAll_TagFilterCalledPerEntityTypeBucket` | Results span 3 entity types. | `EntityIDsByTags` called exactly 3 times (once per type), not once per result. |
| AC-18 | `TestSearchAll_TagFilterAllZeroMatchesReturnsEmpty` | All `EntityIDsByTags` calls return `[]`. | Returns `[]*repository.EntitySearchResult{}` with no error. |
| AC-19 | `TestSearchAll_TagFilterUnregisteredPropagates` | One `EntityIDsByTags` call returns `*UnregisteredTagError`. | Returns `(nil, *UnregisteredTagError)`. |
| AC-19b | `TestSearchAll_NilTagSvcIgnoresTagFilter` | `tagSvc=nil`, `tags=["voice"]`. | No panic; returns all search results unfiltered (graceful degradation). |
| AC-19c | `TestSearchAll_EmptyTagsNoExtraCall` | `tags=nil`. | `EntityIDsByTags` never called; results returned as-is. |

`EntitySearchResult.ID` field (REQ-F-012):

| Test Name | Input / Setup | Expected Outcome |
|---|---|---|
| `TestEntitySearchResult_IDFieldNonZero` | Mock `SearchAll` returns results with `ID` field populated. | Service propagates non-zero IDs; ID field is present and non-zero for every result. |
| `TestEntitySearchResult_IDFieldPresentInStruct` | Compile-time: struct literal assigns `ID` field. | Compilation confirms field exists. |

### 1.6 Entity service Get-with-tags (AC-20)

Tests in each entity service test file (`task_service_test.go` etc.).

For each of the 6 entity types (AC-20 says "each of 6 entities"):

| Test Name Template | Input / Setup | Expected Outcome |
|---|---|---|
| `TestGet<Entity>WithTags_TwoAttachments` | Mock `Get<Entity>` returns entity; mock `ListTagsForEntity` returns `["auth","voice"]`. | Returns `(*Entity, []string{"auth","voice"}, nil)`. |
| `TestGet<Entity>WithTags_ZeroAttachments` | `ListTagsForEntity` returns `[]string{}`. | Returns `(*Entity, []string{}, nil)` — non-nil empty slice. |
| `TestGet<Entity>WithTags_NilTagSvc` | `tagSvc=nil`. | Returns `(*Entity, nil, nil)` — graceful degradation. |
| `TestGet<Entity>WithTags_ListTagsError` | `ListTagsForEntity` returns error. | Returns `(nil, nil, wrappedError)`. |

### 1.7 CLI list commands with `--tag` flag (AC-21..AC-27)

All tests in `internal/cli/commands/` test files. Kind: CLI integration (mocked
services, in-process Cobra, stdout/stderr capture). No real DB. Pattern
follows existing tests in `internal/cli/commands/tags_test.go`.

| AC | Test Name | Setup | Expected Outcome |
|---|---|---|---|
| AC-21 | `TestListCmd_TagFlagFiltersEpics` | `shark list --tag=voice`; mock epic service `ListEpics` returns 2 results when `tags=["voice"]`. | Exit 0. Output contains exactly the 2 matching epics. |
| AC-22 | `TestListCmd_TagFlagFiltersFeatures` | `shark list E07 --tag=voice`; positional arg `E07` dispatches to feature list; mock returns 1 feature. | Exit 0. Output contains the 1 tagged feature. |
| AC-23 | `TestListCmd_TagFlagFiltersTasks` | `shark list E07 F01 --tag=voice`; dispatches to task list. | Exit 0. Only tagged tasks rendered. |
| AC-24 | `TestTaskList_TwoTagFlagsAndSemantics` | `shark task list --tag=voice --tag=auth`; mock `ListTasks` called with `filters.Tags=["voice","auth"]`. | Exit 0. Mock called with correct tags slice. |
| AC-25 | `TestBugList_TagFlagFilters` | `shark bug list --tag=voice`; mock returns 1 bug. | Exit 0. Bug rendered. |
| AC-25b | `TestChangeList_TagFlagFilters` | `shark change list --tag=voice`. | Exit 0. |
| AC-25c | `TestIdeaList_TagFlagFilters` | `shark idea list --tag=voice`. | Exit 0. |
| AC-25d | `TestFeatureList_TagFlagFilters` | `shark feature list E07 --tag=voice`. | Exit 0. |
| AC-25e | `TestEpicList_TagFlagFilters` | `shark epic list --tag=voice`. | Exit 0. |
| AC-26 | `TestSearchCmd_TagFlagFilters` | `shark search "login" --tag=voice`; mock `SearchService.SearchAll` called with `tags=["voice"]` | Exit 0. Filtered results rendered. |
| AC-27 | `TestListCmd_UnregisteredTagExitsWithCode3` | Mock service returns `*UnregisteredTagError{Name:"does-not-exist"}`; mock `ListTags` returns `["voice","auth"]`. | Exit 3. Stderr contains both `voice` and `auth` in vocab snippet. Stderr ends with substring `shark tags add does-not-exist`. |
| AC-27b | `TestListCmd_UnregisteredTagJSON` | Same as AC-27 but `--json` flag. | Exit 3. JSON body contains `"code":"unregistered_tag"`. |

### 1.8 Entity Get display with tags (AC-28)

Tests in each entity CLI test file.

| AC | Test Name | Setup | Expected Outcome |
|---|---|---|---|
| AC-28 | `TestTaskGet_TagsRenderedInRichDisplay` | Mock `GetTaskWithTags` returns `(*task, ["auth","voice"], nil)`. | Output contains `Tags: auth, voice`. |
| AC-28b | `TestTaskGet_TagsRenderedInJSON` | Same, `--json`. | JSON contains `"tags": ["auth","voice"]`. |
| AC-28c | `TestTaskGet_NoTagsRichDisplay` | Mock `GetTaskWithTags` returns `(*task, [], nil)`. | Output contains `Tags: (none)`. |
| AC-28d | `TestTaskGet_NoTagsJSON` | Same, `--json`. | JSON contains `"tags": []`. |
| AC-28e (×5) | `TestFeatureGet_*`, `TestEpicGet_*`, `TestBugGet_*`, `TestChangeGet_*`, `TestIdeaGet_*` | Analogous to AC-28/AC-28b for each entity. | Same tag rendering behavior. |

### 1.9 Performance regression (AC-29)

Test in `internal/services/tag_service_test.go` or `internal/services/search_service_test.go`.
Kind: Unit with OTel span counting.

| AC | Test Name | Setup | Expected Outcome |
|---|---|---|---|
| AC-29 | `TestListTasks_NoTagFilterIssuesZeroExtraSpans` | Use `tracetest.NewInMemoryExporter()` (pattern from existing `TestAttachMany_EmitsSpanWithAttributes`). Call `ListTasks(ctx, TaskFilters{})` with no Tags field. | Span count for `tag_service.*` spans is 0. |

### 1.10 New typed error — `TagFilterUnavailableError` (AC-30)

Test in `internal/services/tag_errors_test.go`.

| Test Name | Input | Expected Outcome |
|---|---|---|
| `TestTagFilterUnavailableError_Message` | `(&TagFilterUnavailableError{}).Error()` | Returns `"tag filtering is not available (TagService not wired)"`. |

### 1.11 UAT-1 end-to-end (AC-31)

AC-31 is a full end-to-end UAT scenario executed manually (not automated in
F05). It maps to `../uat-plan.md` UAT-1. See §2 for the integration scenario
description.

The automated substrate that AC-31 relies on is covered by AC-21..AC-26 and
AC-11/AC-16.

---

## 2. Integration Scenarios

These span multiple components and map to epic-level UAT scenarios in
`../uat-plan.md`. They are required quality gates for epic UAT completion.
F05 contributes the automated substrate; manual UAT scenarios are verified
at epic release.

### 2.1 TagService → EntityTagRepository boundary: `FilterEntityIDs`

**What:** `TagService.EntityIDsByTags` resolves tag names via `TagRepository`,
then calls `EntityTagRepository.FilterEntityIDs` with numeric IDs.

**Verification at boundary:**
- `FilterEntityIDs` is only called after all names resolve to IDs (no partial
  call on validation failure).
- `FilterEntityIDs` receives deduplicated IDs (AC-6: duplicate name input
  produces single-element ID slice).
- Service never calls `FilterEntityIDs` with empty slice (AC-8 precondition).

**Test location:** `internal/services/tag_service_test.go` (unit, mocked repo).

**Feeds into epic UAT:** UAT-1 (cross-entity filtering), UAT-INT-1 (apply +
rename + filter continuity), UAT-INT-2 (enforcement + filter).

### 2.2 Entity service → TagService boundary: List filter path

**What:** Each of the seven entity List services calls `TagService.EntityIDsByTags`
when `filters.Tags` is non-empty, then intersects the result in memory with
the base List output.

**Verification at boundary:**
- `EntityIDsByTags` is called with the correct `models.EntityType` constant
  for each service (table below).
- Short-circuit: when `EntityIDsByTags` returns an empty slice, the base List
  repository method is NOT called (REQ-F-017).
- Error from `EntityIDsByTags` propagates unchanged (no wrapping that loses
  the typed error).
- `nil` tags filter passes through without `EntityIDsByTags` being called.

| Service | Expected EntityType |
|---|---|
| TaskService | `EntityTypeTask` |
| FeatureService (`ListFeatures`, `ListFeaturesByEpicKey`) | `EntityTypeFeature` |
| EpicService | `EntityTypeEpic` |
| BugService | `EntityTypeBug` |
| ChangeCardService | `EntityTypeChange` |
| IdeaService | `EntityTypeIdea` |

**Test location:** Each `*_service_test.go` (unit, mocked tag service + mocked entity repo).

**Feeds into epic UAT:** UAT-1, UAT-INT-1, UAT-INT-2 (step 7: `shark list --tag=undefined`).

### 2.3 SearchService → TagService boundary: post-filter path

**What:** `SearchService.SearchAll` calls `EntityIDsByTags` once per entity
type bucket in the raw result set, then filters results by ID membership.

**Verification at boundary:**
- One `EntityIDsByTags` call per entity type, not per result row (batching
  per REQ-NF-002).
- Results whose `(entity_type, id)` is in the matching set are kept.
- Results whose entity type has no entries in the filter result are dropped.
- Ideas are NOT present in `SearchAll` result set (out of scope for F05 per
  spec §1.4) — no `EntityIDsByTags` call for `EntityTypeIdea` on the search
  path.

**Test location:** `internal/services/search_service_test.go` (unit, mocked search repo + mocked tag service).

**Feeds into epic UAT:** UAT-1 (`shark search "" --tag=voice`).

### 2.4 CLI → Service boundary: `--tag` flag forwarding

**What:** Each of the seven entity list commands and `shark search` reads
`--tag` flags and passes the slice to its service's Filters DTO.

**Verification at boundary:**
- `--tag=voice --tag=auth` (two flag invocations) produces `["voice","auth"]`,
  not `["voice,auth"]` (ADR-F04-5 from F04 test plan §2.2 applies here).
- Absence of `--tag` produces `nil` in the DTO (not `[]string{}`).
- Top-level `shark list` dispatcher forwards the slice to each entity branch.
- `shark list <unrecognized>` dispatch error path does not forward (silent drop
  is acceptable per REQ-F-019 — no new test needed, existing dispatch error
  test covers this).

**Test location:** `internal/cli/commands/` test files (CLI integration, mocked service).

**Feeds into epic UAT:** UAT-1, UAT-2 (unregistered tag rejection on list path).

### 2.5 Error rendering: `handleEntityServiceError` reuse

**What:** `handleEntityServiceError` from F04
(`internal/cli/commands/tags_shared.go:80`) handles `*UnregisteredTagError`
for all six entity list commands and `shark search`. F05 does NOT modify
this helper; it simply calls it on the error path.

**Verification at boundary:**
- `*UnregisteredTagError` surfaced from any list service renders the
  vocabulary snippet and the `shark tags add <name>` remediation line on
  stderr.
- Exit code is 3 per REQ-F-016.
- Both human-readable and JSON paths render correctly (AC-27b).

**Test location:** `internal/cli/commands/` test files (CLI integration tests
AC-27 and AC-27b).

**Feeds into epic UAT:** UAT-2, UAT-INT-2.

---

## 3. Test Infrastructure

### 3.1 Existing patterns to follow

| Pattern | Location | Notes |
|---|---|---|
| Isolated test DB for repository tests | `internal/repository/tag/entity_tag_repository_test.go` — uses `test.NewIsolatedTestDB(t)` | Use this for all `FilterEntityIDs` and `ListTagNamesByEntities` tests. No DELETE cleanup needed. |
| Mock function-field pattern for service tests | `internal/services/tag_service_test.go` — `mockTagRepo.getByNameFn` etc. | Extend for new `filterEntityIDsFn` and `listTagNamesByEntitiesFn` fields. |
| MockTagService (shared service mock) | `internal/services/mock_tag_service_test.go` | Extend with three new method fields per §3.2. |
| OTel span counting in tests | `internal/services/tag_service_test.go` — `TestAttachMany_EmitsSpanWithAttributes` using `tracetest.NewInMemoryExporter` | Pattern for AC-29 regression test. |
| CLI in-process Cobra test | `internal/cli/commands/tags_test.go` | Pattern for all CLI integration tests (AC-21..AC-28). |
| `insertTestEpic` DB helper | `internal/repository/tag/entity_tag_repository_test.go` | Add analogous `insertTestTask`, `insertTestBug` helpers as needed for repository tests scoped to other entity types. |

### 3.2 New helpers to create

**Extended `MockTagService` in `internal/services/mock_tag_service_test.go`:**

Three new method stubs are added to `MockTagService` to satisfy the new
`TagQuerier` interface:

```go
// New function fields:
entityIDsByTagsFn       func(ctx, entityType, names, op) ([]int64, error)
listTagsForEntityFn     func(ctx, entityType, entityID) ([]string, error)
attachedTagNamesByIDsFn func(ctx, entityType, entityIDs) (map[int64][]string, error)

// New call counters:
EntityIDsByTagsCalls       int
ListTagsForEntityCalls     int
AttachedTagNamesByIDsCalls int
```

Methods delegate to the function field or return nil/empty on the happy path,
recording call counts and appending named events to `Events` (same pattern as
existing `AttachMany`/`EnforceRequired`).

**Extended `mockEntityTagRepo` in `internal/services/tag_service_test.go`:**

```go
filterEntityIDsFn          func(ctx, entityType, tagIDs) ([]int64, error)
listTagNamesByEntitiesFn   func(ctx, entityType, entityIDs) ([]EntityIDTagName, error)
```

Delegation follows the existing `countByTagFn` field pattern.

**CLI test helpers (if not already present):**

A `captureOutput(cmd *cobra.Command) (stdout, stderr string, exitCode int)`
helper — if not already present in the CLI test utilities — for in-process
stdout/stderr capture and exit code assertion. Modeled after the pattern in
`tags_test.go`.

### 3.3 No new test database fixtures

F05 adds no schema changes (REQ-F-020). Existing `test.NewIsolatedTestDB(t)`
creates the F01 schema which is sufficient. Repository tests insert their own
tags and entity rows per-test; no shared fixture files are needed.

### 3.4 No new test packages

All F05 tests live in existing packages:
- `internal/repository/tag` (package `tag_test`) — for repository tests
- `internal/services` (package `services`) — for service unit tests
- `internal/cli/commands` (package `commands`) — for CLI integration tests

---

## 4. Coverage Summary

| AC Range | Description | Primary Location | Min Tests |
|---|---|---|---|
| AC-1..AC-6 | `EntityIDsByTags` method | `tag_service_test.go` | 10 (6 primary + 4 edge) |
| AC-7..AC-8 | `FilterEntityIDs` repository | `entity_tag_repository_test.go` | 7 |
| AC-9..AC-10 | `ListTagsForEntity`, `AttachedTagNamesByIDs` | `tag_service_test.go` | 7 |
| AC-11..AC-16 | Entity List tag filter (×7) | `*_service_test.go` files | 7 primary + 1 per entity type = 13 |
| AC-17..AC-19 | SearchService post-filter | `search_service_test.go` | 7 |
| AC-20 | `GetXxxWithTags` (×6) | `*_service_test.go` files | 4 × 6 = 24 |
| AC-21..AC-27 | CLI list/search `--tag` flag | `internal/cli/commands/*_test.go` | 14 |
| AC-28 | Entity Get tag display (×6) | `internal/cli/commands/*_test.go` | 5 × 6 = 30 |
| AC-29 | Performance regression (no extra spans) | `tag_service_test.go` or `search_service_test.go` | 1 |
| AC-30 | `TagFilterUnavailableError` | `tag_errors_test.go` + service tests | 3 |
| AC-31 | UAT-1 end-to-end | Manual UAT at epic release | — |
| `ListTagNamesByEntities` repo | (supports AC-10/AC-13) | `entity_tag_repository_test.go` | 4 |

**Every AC has at least one test. No orphaned tests (all trace to an AC or
an explicit NFR).**

---

## 5. Epic UAT Scenario Contributions

F05 contributes the automated substrate for these epic UAT scenarios. Scenarios
marked "manual" are executed at epic release against a real running instance.

| UAT Scenario | F05 Contribution | Verification Type |
|---|---|---|
| UAT-1 (cross-entity tagging + filter) | AC-11/AC-16 (list filter), AC-21/AC-25 (CLI list), AC-26 (search) | Automated (unit + CLI integration) + Manual (AC-31) |
| UAT-2 (unregistered tag error) | AC-15 (service), AC-27/AC-27b (CLI) | Automated |
| UAT-5 (rename + filter continuity) | Tested in F03 (rename) + F05 AC-11 (filter after rename works because entity_tags rows use tag ID, not name) | Automated in F03; regression confirmed in F05 AC-7d |
| UAT-INT-1 (apply → rename → filter) | AC-7 (FilterEntityIDs returns correct IDs regardless of name), AC-11 | Automated (unit); full flow manual |
| UAT-INT-2 (enforcement + gate + filter) | AC-15 (unregistered propagates), AC-27 (CLI exit 3) | Automated |
| S-4 (non-maintainer read is unrestricted) | AC-3 (empty names no-op), REQ-F-003 (no gate call in EntityIDsByTags) — covered by absence of `gateAuthorize` call in unit tests | Automated (mock gate call count assertion) |
| S-5 (SQL injection defense) | AC-7 (real DB uses `?` placeholders — injection attempt is blocked at validation before reaching SQL) | Automated (repo integration test uses parameterized query; no Sprintf) |
