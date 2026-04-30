---
feature_key: E27-F11
doc_type: test-plan
status: draft
---

# Test Plan — E27-F11: Surface entity size in the web viewer (JSON + UI)

> **Spec source:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F11-surface-entity-size-in-the-web-viewer-json-ui/spec.md`
> **Epic UAT plan:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/uat-plan.md`
> **Test infrastructure:** `internal/services/viewer_service_test.go` (4230 lines, function-field mock pattern)

---

## 1. AC Test Matrix

Every acceptance criterion from spec.md §1.3 maps to at least one test case below.

---

### AC-01: Bug with non-nil Size appears in hierarchy JSON

**Criterion:** `GET /api/v1/viewer/hierarchy` for a project containing one bug with `Size = 5` includes `"size": 5` on that bug's entry under `bugs[]`.

| Test Case | ID | Layer | File |
|---|---|---|---|
| Bug size populated in hierarchy response | TC-AC01-1 | Service (Go unit) | `internal/services/viewer_service_test.go` |

**TC-AC01-1 — Bug size populated**

- Setup: `buildViewerService(t, ...)` with empty epic/feature/task repos. Wire `svc.WithBugListRepo(&mockViewerBugListRepo{ListAllFunc: ...})` returning one `models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001"}, Size: ptrInt(5), Status: "triaged"}`.
- Input: `svc.Hierarchy(context.Background(), HierarchyOptions{})`.
- Expected: `resp.Bugs` has length 1; `resp.Bugs[0].Size != nil`; `*resp.Bugs[0].Size == 5`.
- Pattern: identical to `TestViewerService_Hierarchy_NilTagSvc_TagsAlwaysNonNil` at line 3173 — same builder + `WithBugListRepo` + `WithChangeCardListRepo` wiring.

Edge cases:
- Size value 1 (XS), 13 (XXL): verify the int is passed through unchanged (service does no label translation).
- Change-card and idea variants covered by AC-03.

---

### AC-02: Bug with nil Size has NO "size" key in hierarchy JSON

**Criterion:** `GET /api/v1/viewer/hierarchy` for a project containing one bug with `Size = nil` produces a bug entry with NO `size` key.

| Test Case | ID | Layer | File |
|---|---|---|---|
| Bug size nil — struct field nil | TC-AC02-1 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| Bug size nil — JSON wire omitempty | TC-AC02-2 | Service (Go unit) | `internal/services/viewer_service_test.go` |

**TC-AC02-1 — Bug size nil (struct level)**

- Setup: same as TC-AC01-1 but `models.Bug.Size = nil`.
- Expected: `resp.Bugs[0].Size == nil`.

**TC-AC02-2 — Bug size nil (wire level, omitempty)**

- Setup: same nil-size bug.
- Action: `data, _ := json.Marshal(resp.Bugs[0])`.
- Expected: `!bytes.Contains(data, []byte(`"size"`))`.
- Rationale: Guards REQ-F-001 / REQ-NF-003 at the serialization boundary. The `omitempty` tag on `*int` only omits nil, not zero, so this also proves the field type is `*int` not `int`.

Edge cases:
- A `FlatEntity` whose `Size` is a non-nil pointer to `0` would serialize as `"size":0`. This is a data-integrity concern upstream (the DB `CHECK` constraint on canonical values prevents 0 from being stored). No test needed for `0` here — the model-layer `ValidateSize` rejects it before storage.

---

### AC-03: Size populated and omitted for change-card and idea entities

**Criterion:** AC-01 and AC-02 hold equivalently for `change_cards[]` and `ideas[]`.

| Test Case | ID | Layer | File |
|---|---|---|---|
| ChangeCard size populated | TC-AC03-1 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| ChangeCard size nil | TC-AC03-2 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| ChangeCard size nil — wire omitempty | TC-AC03-3 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| Idea size populated | TC-AC03-4 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| Idea size nil | TC-AC03-5 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| Idea size nil — wire omitempty | TC-AC03-6 | Service (Go unit) | `internal/services/viewer_service_test.go` |

**TC-AC03-1 — ChangeCard size populated**

- Setup: `svc.WithChangeCardListRepo(&mockViewerChangeCardListRepo{ListAllFunc: ...})` returning `models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001"}, Size: ptrInt(3)}`.
- Expected: `resp.ChangeCards[0].Size != nil && *resp.ChangeCards[0].Size == 3`.

**TC-AC03-2 — ChangeCard size nil**

- Setup: `models.ChangeCard.Size = nil`.
- Expected: `resp.ChangeCards[0].Size == nil`.

**TC-AC03-3 — ChangeCard size nil wire**

- Action: `data, _ := json.Marshal(resp.ChangeCards[0])`.
- Expected: `!bytes.Contains(data, []byte(`"size"`))`.

**TC-AC03-4 — Idea size populated**

- Setup: `svc.ideaRepo` wired via the builder with `mockViewerIdeaListRepo.ListAllFunc` (note: `ListAll(ctx context.Context)` — no `includeTerminal` arg, per line 3156 in test file). Returns `models.Idea{Key: "IDEA-01", Size: ptrInt(8)}`.
- Expected: `resp.Ideas[0].Size != nil && *resp.Ideas[0].Size == 8`.
- Note: `Idea.Size` is at `models.Idea.Size` (not `BaseEntity.Size`) — test must source from the direct field, per spec §2.5.

**TC-AC03-5 — Idea size nil**

- Expected: `resp.Ideas[0].Size == nil`.

**TC-AC03-6 — Idea size nil wire**

- Expected: `!bytes.Contains(data, []byte(`"size"`))`.

Edge cases:
- All three entity types in a single hierarchy call (multi-entity test): wire all three repos, assert each of `resp.Bugs[0].Size`, `resp.ChangeCards[0].Size`, `resp.Ideas[0].Size` independently — verifies the three construction sites don't interfere.

---

### AC-04: Epic, feature, task size exposure unchanged (no regression)

**Criterion:** Existing hierarchy entries for epic / feature / task continue to expose `size` exactly as before.

| Test Case | ID | Layer | File |
|---|---|---|---|
| Epic/feature/task size not regressed | TC-AC04-1 | Service (Go unit) | existing tests in `viewer_service_test.go` |

**TC-AC04-1 — Shape regression**

- This is satisfied by the existing test suite (`TestViewerService_Hierarchy_EmbedsTasks`, etc.) which must continue to pass without modification.
- Explicit action: run `go test ./internal/services/ -run TestViewerService_Hierarchy` before and after the change; the passing set must be a superset.
- No new test function is required for this AC — it is verified by not-breaking existing tests.

Edge cases:
- An epic with `Size = nil` — already covered by existing tests; `FlatEntity` is only used for bugs/change-cards/ideas, not epics/features/tasks which have their own response structs. Verify epics and features use their own response type (not `FlatEntity`) so the struct change does not accidentally affect them.

---

### AC-05: Size row appears in entity detail panel Info tab for size = 5

**Criterion:** Navigating to an entity with `size = 5` shows `L` (or `L (5)`) in the Info tab as a "Size" row.

| Test Case | ID | Layer | File |
|---|---|---|---|
| Size row present when size non-nil | TC-AC05-1 | Front-end smoke (manual) | `internal/viewer/assets/viewer.html` |

**TC-AC05-1 — Info tab Size row (manual)**

- Precondition: Local project DB contains at least one bug with `size = 5`.
- Action: Run `shark web`; navigate to that bug in the viewer; open the "Info" tab.
- Expected:
  - A row labelled "Size" is present.
  - The value cell shows `L (5)` (per spec §2.5 detail-panel format: `formatSizeLabel(e.size) + ' (' + e.size + ')'`).
- Applicable entity types: bug, change-card, idea (plus epic, feature, task which already carry size). Test all six in one manual pass.
- Console: DevTools Console must show zero JS errors during navigation.

Edge cases:
- `size = 1` → expects `XS (1)`.
- `size = 13` → expects `XXL (13)`.
- All six entity types: at least one of each type with a non-nil size must be tested.

---

### AC-06: No "Size" row when entity has nil size

**Criterion:** Navigating to an entity with `size = nil` shows NO "Size" row in the Info tab.

| Test Case | ID | Layer | File |
|---|---|---|---|
| Size row absent when size nil | TC-AC06-1 | Front-end smoke (manual) | `internal/viewer/assets/viewer.html` |

**TC-AC06-1 — Info tab no Size row (manual)**

- Precondition: Project contains entities (bug, change-card, idea) with no size set.
- Action: Navigate to each; open "Info" tab.
- Expected: No "Size" row exists; no empty row with an empty value cell.
- Also covers: idea entities — which currently have no case in the switch at line 3472. After the change, `case 'idea':` must be present and calling `appendIdeaFields`. If the idea has nil size, the row must still be absent.

Edge cases:
- An idea entity without the `appendIdeaFields` function (regression check): if the case is missing from the switch, the Size row will never appear for ideas. Confirm `appendIdeaFields` is both defined and called from the dispatcher.

---

### AC-07: Size badge in sidebar tree for entities with non-nil size

**Criterion:** Tree nodes whose payload has a non-nil size show a small `XS`/`S`/`M`/`L`/`XL`/`XXL` badge. Nodes without a size show no badge.

| Test Case | ID | Layer | File |
|---|---|---|---|
| Size badge present for all six entity types | TC-AC07-1 | Front-end smoke (manual) | `internal/viewer/assets/viewer.html` |
| No size badge when size nil | TC-AC07-2 | Front-end smoke (manual) | `internal/viewer/assets/viewer.html` |

**TC-AC07-1 — Badge present (manual)**

- Precondition: At least one entity of each type (epic, feature, task, bug, change-card, idea) has a non-nil size.
- Action: Open viewer; examine sidebar tree nodes for those entities.
- Expected: A small badge with the t-shirt label (e.g., `L`) appears to the right of the entity title span. CSS class `.size-badge` is present on the element.
- Note: Badge appears in `buildEpicNodeHtml`, `buildFeatureNodeHtml`, `buildTaskNodeHtml`, and `buildFlatSectionHtml` — all four builders must be verified.

**TC-AC07-2 — No badge (manual)**

- Precondition: Entities with `size = null` (missing `size` key in hierarchy JSON).
- Expected: No `<span class="size-badge">` element in those tree nodes. No empty badge space or placeholder.

Edge cases:
- An entity where `size` is missing entirely from the JSON payload (not `null`, just absent) — the JS defensive check `if (size == null)` in `buildSizeBadgeHtml` covers both `null` and `undefined`, so this case is equivalent to nil. Verify this in DevTools by inspecting the raw hierarchy response.

---

### AC-08: Non-canonical size (e.g., size = 7) renders as `7` without crashing

**Criterion:** Receiving `size = 7` (non-canonical) renders the badge text `7` and does NOT throw a JS error.

| Test Case | ID | Layer | File |
|---|---|---|---|
| Defensive fallback for non-canonical size | TC-AC08-1 | Front-end smoke (manual/dev-tools) | `internal/viewer/assets/viewer.html` |

**TC-AC08-1 — Fallback rendering (manual)**

- Setup: Temporarily modify a test DB record to set `size = 7` (non-canonical), or mock the hierarchy API response in DevTools to return `"size": 7`.
- Action: Load viewer; navigate to that entity.
- Expected:
  - Sidebar badge shows `7` (the raw integer, per `SIZE_LABELS[n] || String(n)`).
  - Info tab Size row shows `7 (7)` (raw integer in both positions, per defensive branch).
  - DevTools Console shows zero JS errors.
- This mirrors the defensive branch in `internal/cli/commands/helpers.go:912` (`formatSize` fallback).

Edge cases:
- `size = 0`: same defensive path; renders `0`.
- `size = -1`: same; renders `-1`.

---

### AC-09: `make fmt && make lint && make test` passes

**Criterion:** The full build-quality gate passes.

| Test Case | ID | Layer | File |
|---|---|---|---|
| Full quality gate | TC-AC09-1 | Automated CI | Makefile |

**TC-AC09-1 — Quality gate (automated)**

- Action: `make fmt && make lint && make test` on the implementation branch.
- Expected: Exit code 0 on all three targets.
- Precondition: No formatting drift (`make fmt` produces no diff), no lint warnings, all unit tests pass including the new ones added in AC-10.

---

### AC-10: New service-layer unit tests in viewer_service_test.go

**Criterion:** Tests cover: (a) flat-entity size populated from model, (b) flat-entity size omitted when model size is nil, (c) all three entity families (bug, change-card, idea).

| Test Case | ID | Layer | File |
|---|---|---|---|
| Bug size populated (struct + wire) | TC-AC10-1 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| Bug size nil | TC-AC10-2 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| ChangeCard size populated | TC-AC10-3 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| ChangeCard size nil | TC-AC10-4 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| Idea size populated | TC-AC10-5 | Service (Go unit) | `internal/services/viewer_service_test.go` |
| Idea size nil | TC-AC10-6 | Service (Go unit) | `internal/services/viewer_service_test.go` |

These are the same test cases as AC-01 through AC-03 — listed here as the AC-10 checklist confirms they exist as actual test functions. AC-10 is the meta-criterion requiring the tests to be written; AC-01/02/03 define their content.

---

## 2. Integration Scenarios

### 2.1 Service-to-JSON boundary (Go side)

**Components:** `ViewerService.Hierarchy()` → `FlatEntity` → `encoding/json`

**What to verify at the boundary:**

The `FlatEntity` struct change (`Size *int json:"size,omitempty"`) must be consistent with the serialization contract:

| Scenario | Input | Expected wire output |
|---|---|---|
| Size = 5 | `FlatEntity{Size: ptrInt(5)}` | `{"key":"...","size":5,...}` |
| Size = nil | `FlatEntity{Size: nil}` | `{"key":"...",...}` (no `size` key) |
| Size = 0 (invalid, never stored) | `FlatEntity{Size: ptrInt(0)}` | `{"key":"...","size":0,...}` — `0` is NOT omitted because `omitempty` on `*int` only omits nil pointers |

The third row is a known gotcha: `*int` with value `0` serializes as `"size":0`, not omitted. This is correct behavior because the model layer's `ValidateSize` rejects 0 before any record reaches the DB. No test needed for the invalid-0 case, but the implementer must use `*int`, not `int`, to preserve the nil-omit contract.

**Epic UAT coverage:** Scenario B1 ("Summary cards reflect the database") and the general "data parity with CLI" guarantee both depend on this boundary being correct. Specifically, UAT scenario I2 ("Edit with CLI, refresh in browser") exercises this path live.

### 2.2 Hierarchy API → SPA rendering

**Components:** `GET /api/v1/viewer/hierarchy` → `buildFlatSectionHtml` / `buildEpicNodeHtml` / `buildFeatureNodeHtml` / `buildTaskNodeHtml` → DOM badge

**What to verify:**

- The SPA reads `item.size` from the hierarchy JSON. If `size` is absent (nil), `item.size` evaluates to `undefined`. The `buildSizeBadgeHtml(size)` helper must guard `if (size == null)` — which covers both `null` and `undefined` (JS abstract equality).
- The `SIZE_LABELS` lookup map `{1:'XS',2:'S',3:'M',5:'L',8:'XL',13:'XXL'}` must match the canonical set in `internal/models/size.go`. If the Go model adds a new canonical value, the JS map must be updated in sync. This is a future-maintenance risk noted in spec §2.4 (D4).

**Epic UAT coverage:** UAT scenario C1 ("Tree renders all epics and features") requires sidebar nodes to be well-formed; a crashing JS badge helper would break C1. UAT scenario J2 ("No console errors") provides the catch-all guard.

### 2.3 Entity detail panel — `appendIdeaFields` dispatcher gap

**Components:** viewer.html line 3472 switch → `appendIdeaFields` (new function) → `pushRow`

**What to verify:**

Before this feature, `ideas` had no `case` in the switch at line 3472 (only `epic`, `feature`, `task`, `bug`, `change_card`). The implementation must add `case 'idea': appendIdeaFields(rows, entity); break;`. If this case is missing:

- The `Size` row will never appear for idea entities in the Info tab.
- No JS error occurs (the switch silently falls through with shared base fields).
- The gap would be invisible to the automated test suite (which has no JS unit tests).

**Verification:** Manual smoke test TC-AC06-1 must explicitly exercise an idea entity to catch this.

**Epic UAT coverage:** UAT scenario D3 ("Properties panel") requires entity-type-specific fields to appear. An idea with `size = 5` not showing a Size row would violate D3 after this feature is claimed complete.

### 2.4 Cross-entity regression: existing epic/feature/task size exposure

**Components:** `*models.Epic`, `*models.Feature`, `*models.Task` (which all embed `BaseEntity.Size`) vs `FlatEntity.Size` (new field)

**What to verify:**

Epics, features, and tasks are NOT serialized as `FlatEntity`. They use their own response structs (e.g., `EpicNode`, `FeatureNode`, `TaskInView`). The `FlatEntity.Size` field addition must not cause any field shadowing or struct collision in those types. Verified implicitly by the existing test suite passing unchanged (TC-AC04-1).

**Epic UAT coverage:** UAT scenario B1 and B3 ("Feature progress bars") and C1 depend on the epic/feature/task response shapes being stable.

---

## 3. Test Infrastructure

### 3.1 Existing patterns to follow

| Pattern | Location | How it applies to E27-F11 |
|---|---|---|
| `buildViewerService(t, ...)` builder | `internal/services/viewer_service_test.go:258` | All new Go tests use this builder; do not construct `ViewerService` directly. |
| `svc.WithBugListRepo(...)` | Used in `TestViewerService_Hierarchy_NilTagSvc_TagsAlwaysNonNil` (line 3201) | Wire bug/change-card/idea repos after service construction for size tests. |
| `mockViewerBugListRepo` | Defined at line 3113 | Reuse as-is; `ListAllFunc` returns `[]*models.Bug`. |
| `mockViewerChangeCardListRepo` | Defined at line 3132 | Reuse as-is; `ListAllFunc` returns `[]*models.ChangeCard`. |
| `mockViewerIdeaListRepo` | Defined at line 3151 | Reuse as-is; note `ListAll(ctx context.Context)` signature (no `includeTerminal`). |
| `bytes.Contains(data, []byte("\"size\""))` | Pattern described in spec §2.5 | Use for wire-level omitempty tests (TC-AC02-2, TC-AC03-3, TC-AC03-6). |

### 3.2 Helper function needed

No new mock types are required. One small helper function should be added at the top of the new test block to reduce repetition:

```go
// ptrInt returns a pointer to n. Used to set optional int fields in test fixtures.
func ptrInt(n int) *int { return &n }
```

Check whether `ptrInt` or equivalent already exists elsewhere in `viewer_service_test.go` before adding it. Search for `func ptrInt` or `func intPtr` — if found, reuse it.

### 3.3 No new infrastructure needed

- No new mock types.
- No new test database (all service tests are mock-based, per `.claude/rules/testing/architecture.md`).
- No JS unit-test harness (per epic constraint: no new dependencies, REQ-NF-004). Front-end coverage is manual smoke.
- No new test helper files.

### 3.4 Placement of new tests

Append new test functions immediately after `TestViewerService_HierarchyOptions_HasTagsField` (line 3101) and before the mock helper definitions (line 3111), or after the last existing hierarchy tag test in the file. Group them under a clear comment block:

```go
// ----- E27-F11: FlatEntity.Size field population and wire-format tests -----
```

---

## 4. Test Execution Guide

### 4.1 Automated (Go unit tests)

```bash
# Run only the new E27-F11 service tests
go test -v ./internal/services/ -run "TestViewerService_Hierarchy_FlatEntity"

# Run the full viewer service test suite (regression check)
go test -v ./internal/services/ -run "TestViewerService"

# Full quality gate
make fmt && make lint && make test
```

### 4.2 Manual front-end smoke

Prerequisite: project with seeded data (per UAT plan §1 — at least one entity of each type with a non-nil size, and at least one of each type without a size).

1. `make build && ./bin/shark web`
2. Open `http://127.0.0.1:7777` in Chrome or Edge.
3. Open DevTools → Console tab; leave it visible throughout.
4. Sidebar badge checks (AC-07):
   a. For each of epic, feature, task, bug, change-card, idea with non-nil size: confirm badge with t-shirt label appears in the tree node.
   b. For entities with no size: confirm no badge and no empty placeholder.
5. Info tab Size row checks (AC-05, AC-06):
   a. Click a bug with `size = 5`; open Info tab; confirm "Size" row shows `L (5)`.
   b. Click a bug with no size; open Info tab; confirm no "Size" row.
   c. Click an idea with `size = 8`; open Info tab; confirm "Size" row shows `XL (8)`.
   d. Click an idea with no size; open Info tab; confirm no "Size" row.
6. Non-canonical size (AC-08):
   a. In DevTools Network, intercept the hierarchy response; modify a bug entry to have `"size": 7`; reload.
   b. Confirm sidebar badge shows `7` (not a crash, not blank).
   c. Confirm Info tab shows `7 (7)` (not a crash).
   d. Confirm Console shows zero JS errors.
7. Wire format check:
   a. In DevTools Network; select `GET /api/v1/viewer/hierarchy`; inspect the response JSON.
   b. For an entity without size: confirm `size` key is absent from its JSON object.
   c. For an entity with size: confirm `"size":<integer>` is present.
8. Console clean check (J2 UAT): zero JS errors and zero failed network requests after navigating through 5+ entities.

---

## 5. UAT Scenario Coverage Mapping

This feature contributes to the following E27 UAT scenarios:

| UAT Scenario | How E27-F11 contributes |
|---|---|
| **B1** — Summary cards reflect the database | The hierarchy endpoint is the same data source; adding `size` to flat entities does not change summary counts but must not corrupt them. |
| **C1** — Tree renders all epics and features | Sidebar badge additions to tree nodes must not break tree rendering for entities without a size. |
| **D3** — Properties panel shows entity-specific fields | Size row in Info tab is an entity-specific field for all six entity families. AC-05 / AC-06 validate this directly. |
| **I1** — Full loop against a live project | Clicking through entity types will exercise the badge and Info tab row. A JS crash from the badge builder would break I1. |
| **J2** — No console errors | AC-08 defensive fallback and general badge/row logic must be error-free. J2 catches any unhandled JS exception. |

---

## 6. Exit Gate Checklist

- [x] Every AC in spec.md §1.3 (AC-01 through AC-10) has at least one test case defined above.
- [x] Edge cases identified for each AC (nil vs non-nil size, all six entity families, non-canonical value, wire-format omitempty).
- [x] Integration scenarios cover all cross-component boundaries: service-to-JSON, JSON-to-SPA, detail-panel dispatcher gap, epic/feature/task regression.
- [x] Test patterns reference existing infrastructure: `buildViewerService`, `mockViewerBugListRepo`, `mockViewerChangeCardListRepo`, `mockViewerIdeaListRepo`, `WithBugListRepo` / `WithChangeCardListRepo` wiring — all from `viewer_service_test.go` lines 3113–3168, 3201.
- [x] No orphaned tests — every test case traces to at least one AC.
- [x] Manual smoke scope defined for front-end tests (no JS unit-test harness added per REQ-NF-004).
- [x] UAT scenario coverage mapped (B1, C1, D3, I1, J2).
