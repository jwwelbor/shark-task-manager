---
feature_key: E27-F11
title: Surface entity size in the web viewer (JSON + UI)
spec_version: 1.0
status: in_specification
---

# Spec — E27-F11: Surface entity size in the web viewer (JSON + UI)

> **Business context**: see epic PRD `docs/plan/E27-shark-status-viewer-local-web-dashboard/epic.md` ("Goal" section).
> **System architecture**: see epic PRD ("Architecture Decision", "Implementation Phases", "API Contract").
> **Feature description**: see `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F11-surface-entity-size-in-the-web-viewer-json-ui/feature.md`.
> **Upstream feature**: E07-F42 added the canonical `Size *int` field with t-shirt labels. CLI surface complete; this feature closes the viewer-side gap.

---

## 1. Requirements

### 1.1 Functional Requirements

> Notation: requirements are **incremental over the epic** (which already defines the hierarchy JSON shape and the SPA structure). They add the size field to flat entities and the size badge to the UI.

| ID | Requirement | Trace |
|---|---|---|
| **REQ-F-001** | The hierarchy JSON response (`GET /api/v1/viewer/hierarchy`) MUST expose a numeric `size` field for **bug**, **change-card**, and **idea** entries when the underlying record has a non-nil `Size`. The field MUST be omitted (not `null`, not `0`) when `Size` is nil, mirroring the existing `*models.Epic` / `*models.Feature` / `*models.Task` behaviour. | Epic API Contract `/api/v1/viewer/hierarchy`; feature.md scope item 1 |
| **REQ-F-002** | The `size` field on flat-entity hierarchy entries MUST carry the same canonical Fibonacci integer (1, 2, 3, 5, 8, 13) that the model layer enforces via `models.ValidateSize`. The viewer MUST NOT translate, alias, or recompute the value. | E07-F42 size canonicalization; `internal/models/size.go` |
| **REQ-F-003** | The viewer SPA (`viewer.html`) MUST render a size indicator for any hierarchy entity whose payload contains a non-empty `size` field. This applies to all six entity families now exposing size: epic, feature, task, bug, change-card, idea. | feature.md scope item 2 |
| **REQ-F-004** | The size indicator MUST display the t-shirt label form (XS/S/M/L/XL/XXL), matching the format produced by `formatSize` in `internal/cli/commands/helpers.go`. The numeric value MAY be shown in the badge tooltip but MUST NOT be the primary display token. | feature.md scope item 2; CLI parity (`formatSize`) |
| **REQ-F-005** | The size indicator MUST appear in the **entity detail panel "Info" tab** as a labelled row ("Size") for every entity type for which size is exposed in the hierarchy or detail JSON. | feature.md scope item 2 ("'Size' row in the entity detail panel") |
| **REQ-F-006** | The size indicator MUST appear in the **sidebar tree** as a small badge next to the entity title for entities that have a non-nil size. Entities with nil size MUST render with no badge (no placeholder, no "—"). | feature.md scope item 2 ("small badge on hierarchy rows/cards") |
| **REQ-F-007** | If the SPA receives an unrecognized numeric size (i.e. not in {1, 2, 3, 5, 8, 13}), it MUST silently fall back to displaying the raw integer rather than crashing or rendering an empty badge. This mirrors the defensive branch in `formatSize`. | `internal/cli/commands/helpers.go:907` (defensive `formatSize` fallback) |
| **REQ-F-008** | The viewer service MUST populate the `Size` field on `FlatEntity` from the source model (`models.Bug.Size`, `models.ChangeCard.Size`, `models.Idea.Size`) at the three existing `&FlatEntity{...}` construction sites in `internal/services/viewer_service.go`. | feature.md scope item 1 (lines 988, 1006, 1027) |

### 1.2 Non-Functional Requirements

| ID | Requirement | Trace |
|---|---|---|
| **REQ-NF-001** | No new database queries. Size is already part of the row read by the bug / change-card / idea `ListAll` repository calls (it lives in `BaseEntity`/`Idea.Size`). The change is purely a JSON projection update. | `models.BaseEntity.Size`, `models.Idea.Size` |
| **REQ-NF-002** | Hierarchy endpoint latency MUST remain under the epic's stated budget (< 500 ms for 500 tasks). Adding one int pointer per flat entity is bounded O(N) work over already-loaded slices and adds well below 1 ms in practice. | Epic Success Criteria |
| **REQ-NF-003** | Wire format MUST be backwards compatible: existing clients that ignore unknown JSON fields MUST continue to function. The `omitempty` tag guarantees nil sizes do not appear at all (no breaking shape change for entities without a size). | Default Go JSON-encoding rules; existing shape stability |
| **REQ-NF-004** | The change MUST NOT introduce new dependencies (Go side or JS side). Rendering uses existing helpers and CSS variables in `viewer.html`. | Epic Key Constraints ("No new dependencies") |
| **REQ-NF-005** | Test additions MUST follow the existing service-test mock pattern (`internal/services/viewer_service_test.go` uses function-field mocks via `mockViewerBugListRepo`, `mockViewerChangeCardListRepo`, `mockViewerIdeaListRepo`). No real database. | `.claude/rules/services/testing.md` |

### 1.3 Acceptance Criteria

Each criterion is a single, testable observation.

| ID | Criterion |
|---|---|
| **AC-01** | `GET /api/v1/viewer/hierarchy` for a project containing one bug with `Size = 5` includes `"size": 5` on that bug's entry under `bugs[]`. |
| **AC-02** | `GET /api/v1/viewer/hierarchy` for a project containing one bug with `Size = nil` produces a bug entry with NO `size` key (verified via raw JSON byte inspection or `jq 'has("size")'`). |
| **AC-03** | AC-01 and AC-02 hold equivalently for `change_cards[]` (with a ChangeCard) and `ideas[]` (with an Idea). |
| **AC-04** | Existing hierarchy entries for epic / feature / task continue to expose `size` exactly as before (no regression). Verified by an existing-shape snapshot test that does not change. |
| **AC-05** | In `viewer.html`, navigating to an entity with `size = 5` shows `L` (or `L (5)`) in the Info tab as a "Size" row. |
| **AC-06** | In `viewer.html`, navigating to an entity with `size = nil` shows NO "Size" row in the Info tab. |
| **AC-07** | In the sidebar tree, an epic / feature / task / bug / change-card / idea node whose payload has a non-nil size shows a small `XS`/`S`/`M`/`L`/`XL`/`XXL` badge. Nodes without a size show no badge. |
| **AC-08** | Receiving a hierarchy payload with an unexpected `size = 7` (non-canonical) renders the badge text `7` and does NOT throw a JS error (verifiable via console-clean smoke). |
| **AC-09** | `make fmt && make lint && make test` passes on a clean checkout including the changes. |
| **AC-10** | New service-layer unit tests in `internal/services/viewer_service_test.go` cover: (a) flat-entity size populated from model, (b) flat-entity size omitted when model size is nil, (c) all three entity families (bug, change-card, idea). |

### 1.4 Out of Scope

Restated from `feature.md` for completeness; do not expand without re-spec:

- **Tech-debt in the web viewer.** Tech-debt is not in `HierarchyResponse`; no `tech_debt` repo is wired into `ViewerService`. File a separate feature when prioritized.
- **Size-based filtering or sorting** in the viewer. This feature is read-only display.
- **Editing size from the viewer UI.** Size mutation remains a CLI-only operation.
- **Changes to the size value model, parser, or label set.** That work shipped in E07-F42 and the `feat/update-size-flag-and-td-size` branch.
- **Server-side translation of numeric size to label.** Labels are computed in the SPA; the JSON wire format is the canonical numeric value (matches CLI `--field size` and `--field size_label` distinction).

---

## 2. Architecture

### 2.1 Component Changes

> Pattern alignment: follows the existing `viewer_service.go` projection pattern. No new services, repositories, or wiring required.

#### Files Modified

| File | Change |
|---|---|
| `internal/services/viewer_service.go` | Add `Size *int \`json:"size,omitempty"\`` to the `FlatEntity` struct (around line 195–202). Populate `Size: b.Size`, `Size: cc.Size`, `Size: idea.Size` at the three `&FlatEntity{...}` construction sites (around lines 988, 1006, 1027). |
| `internal/services/viewer_service_test.go` | Add tests for the populated/nil-size paths on bug, change-card, and idea (see §2.5 Test Plan). |
| `internal/viewer/assets/viewer.html` | (a) Add a `formatSizeLabel(n)` helper near the existing `getStatusColor` / `statusBadgeCell` block. (b) Add a `buildSizeBadgeHtml(size)` helper that returns `''` for nil/missing size. (c) Insert a "Size" row appender in the entity-detail Info tab dispatch (line 3472 switch + a new `appendIdeaFields` for ideas). (d) Insert size-badge HTML into the existing `buildFlatSectionHtml` and `buildEpicNodeHtml` / `buildFeatureNodeHtml` / `buildTaskNodeHtml` template strings, immediately after the `tree-node-title` span. (e) Add a small `.size-badge` CSS rule in the existing `<style>` block. |

#### Files Created

None. This is a strictly additive change to existing files.

### 2.2 Data Model Changes

**No database migrations.** Size already exists on every relevant table:

- `bugs.size`, `change_cards.size`, `tasks.size`, `features.size`, `epics.size`, `tech_debt.size` — all shipped in E07-F42 and the size-flag/TD branch.
- `ideas.size` — added in the same wave (`internal/models/idea.go:40`, `Size *int \`json:"size,omitempty" db:"size"\``).

**Single Go struct change** (additive, optional field):

```go
// internal/services/viewer_service.go (existing struct, MODIFIED)
type FlatEntity struct {
    Key         string   `json:"key"`
    Title       string   `json:"title"`
    Status      string   `json:"status"`
    StatusColor string   `json:"status_color"`
    Tags        []string `json:"tags"`
    Size        *int     `json:"size,omitempty"` // NEW (REQ-F-001, REQ-F-002, REQ-F-008)
    dbID        int64    // unexported; unchanged
}
```

`*int` (not `int`) is mandatory: it preserves `omitempty` semantics for the "no size set" case. This matches the existing `BaseEntity.Size *int` shape.

### 2.3 API / Interface Contracts

#### Wire-format example: hierarchy bugs entry (size present)

```json
{
  "key": "B007",
  "title": "Login button stuck",
  "status": "triaged",
  "status_color": "#e7b30e",
  "tags": ["auth"],
  "size": 5
}
```

#### Wire-format example: hierarchy idea entry (size absent)

```json
{
  "key": "IDEA-12",
  "title": "Bulk reassign",
  "status": "proposed",
  "status_color": "#60a5fa",
  "tags": []
}
```

`size` is silently omitted, not `null`. Existing clients that don't know about `size` see exactly the byte sequence they saw before this change for any entity without a size.

#### No new endpoints

All four endpoints that touch hierarchy or entity-detail JSON keep their existing paths and parameter semantics:

- `GET /api/v1/viewer/hierarchy` — gains optional `size` on flat entries.
- `GET /api/v1/viewer/features/{key}/tasks` — already exposes size via the embedded `*models.Task` (verify; no change expected).
- `GET /api/v1/viewer/recent-activity` — out of scope; no entity payload here.
- `GET /api/v1/viewer/file/{key}` — content endpoint, unaffected.

### 2.4 Key Technical Decisions

| Decision | Rationale | Alternative considered |
|---|---|---|
| **D1.** Add `Size *int` directly to `FlatEntity` rather than introducing a discriminated `BaseFlat` super-struct. | The other 5 fields on `FlatEntity` already follow this pattern. Introducing a new abstraction for one extra field would be over-engineering and inconsistent with the rest of the file. Aligns with `.claude/rules/services/service-design.md` ("Anemic services: services should add value, not abstractions"). | A shared `BaseFlat { Size *int }` embedded across response types — rejected as premature generalization. |
| **D2.** Serialize size as the numeric Fibonacci int, not the t-shirt label. | The existing CLI `--field size` returns the int and `--field size_label` returns the label. The SPA can compute the label client-side via a small helper, while the JSON stays normalized and machine-friendly (matches REQ-F-002). Server-side translation would require pulling `models.SizeLabel` into the projection layer for no upside. | Wire-side label string — rejected as redundant projection. |
| **D3.** Render size **both** in the sidebar (badge) **and** the detail panel (row), not just one. | The feature description (line 27) explicitly says "small badge on hierarchy rows/cards **and/or** a 'Size' row in the entity detail panel". Both surfaces are cheap and answer different user questions: the badge gives at-a-glance scanning, the row gives unambiguous identification. | Detail-panel row only — rejected as it forces extra clicks for at-a-glance views. |
| **D4.** Compute the t-shirt label client-side via a tiny in-page lookup map (`{1:'XS',2:'S',3:'M',5:'L',8:'XL',13:'XXL'}`). | Mirrors the canonical map in `internal/models/size.go:25`. Avoids any new fetch or calculation cost. The map is a 6-entry literal — not a meaningful duplication burden. | Use a dynamic call to `/api/v1/viewer/workflow-meta` — rejected as the workflow endpoint has no business publishing the size enum. |
| **D5.** Add `appendIdeaFields(rows, e)` to the entity-detail dispatcher (line 3472 switch) so the `Size` row works for ideas. | Currently `viewer.html` line 3472 has no `case 'idea':` branch; ideas fall through with only the shared base fields. Adding one is required for AC-05 on idea detail panels and is a natural symmetry fix. | Extend only `appendBugFields` / `appendChangeCardFields` and let ideas remain feature-poor — rejected as it leaves the Size row missing for ideas. |
| **D6.** No change to the `models` package. | `models.SizeLabel` already exists; nothing the model layer needs to do here. Models must stay free of viewer concerns per the layered architecture (`.claude/rules/architecture.md`). | Add a `models.FormatSize()` helper — rejected as the SPA cannot import Go anyway. |
| **D7.** Defensive client-side fallback for unknown numeric sizes. | Mirrors the defensive branch in `formatSize` (`internal/cli/commands/helpers.go:912`). A non-canonical size in the DB is a defect upstream of the viewer, but the viewer must not crash on data it didn't author (REQ-F-007). | Throw / show "?" — rejected as confusing and inconsistent with CLI behaviour. |

### 2.5 Integration with Existing Code

#### Go service layer

**File**: `internal/services/viewer_service.go`

1. **Struct edit** (around line 195–202):
   - Insert `Size *int \`json:"size,omitempty"\`` between `Tags` (line 200) and `dbID` (line 201).
2. **Construction-site edits** at the three flat-entity append blocks:
   - Line ~988 (bug): add `Size: b.Size,` after `Tags`. Field exists at `models.Bug.BaseEntity.Size` (`internal/models/bug.go:34` embeds `BaseEntity`; size is at `internal/models/entity.go:55`).
   - Line ~1006 (change-card): add `Size: cc.Size,`. Same `BaseEntity` embedding (`internal/models/change_card.go:32`).
   - Line ~1027 (idea): add `Size: idea.Size,`. `Idea` does NOT embed `BaseEntity` but defines `Size *int` directly (`internal/models/idea.go:40`).

No other functions in `viewer_service.go` need changes — the field flows through `result.Bugs`, `result.ChangeCards`, `result.Ideas` automatically.

**File**: `internal/services/viewer_service_test.go`

Add new tests adjacent to the existing flat-entity tests (search for `mockViewerBugListRepo`, `mockViewerChangeCardListRepo`, `mockViewerIdeaListRepo` blocks starting at line 3113):

- `TestViewerService_Hierarchy_FlatEntity_BugSize_Populated` — feed a `models.Bug` with a non-nil `Size` via `mockViewerBugListRepo.ListAllFunc`; assert `resp.Bugs[0].Size != nil && *resp.Bugs[0].Size == 5`.
- `TestViewerService_Hierarchy_FlatEntity_BugSize_Nil` — same but with `Size: nil`; assert `resp.Bugs[0].Size == nil`.
- `TestViewerService_Hierarchy_FlatEntity_ChangeCardSize_Populated` and `_Nil` — same shape for change-cards.
- `TestViewerService_Hierarchy_FlatEntity_IdeaSize_Populated` and `_Nil` — same shape for ideas (note `mockViewerIdeaListRepo.ListAllFunc` does not take an `includeTerminal` arg — check the signature at line 3156).
- `TestViewerService_Hierarchy_FlatEntity_BugSize_OmitemptyJSON` — marshal `resp.Bugs[0]` to JSON via `encoding/json` and assert (a) `bytes.Contains(data, []byte("\"size\":5"))` when populated, and (b) `!bytes.Contains(data, []byte("\"size\""))` when nil. This guards REQ-F-001's omit-on-nil contract at the wire level.

Mock pattern is identical to the existing tag-decoration tests — no new mock types needed.

#### Front-end SPA

**File**: `internal/viewer/assets/viewer.html`

Five additive edits, all small. None touches existing logic; each is an insert.

1. **CSS** (around the existing `.status-badge` rule, ~line 311):

   ```css
   .size-badge {
     display: inline-block;
     padding: 1px 6px;
     margin-left: 6px;
     font-size: 10px;
     font-weight: 600;
     line-height: 1.4;
     color: #cbd5e1;
     background: #334155;
     border-radius: 8px;
     vertical-align: middle;
   }
   .props-row-size .props-cell-value { font-weight: 500; }
   ```

2. **JS helpers** (insert near `getStatusColor` / `statusBadgeCell`, ~line 3327):

   ```js
   const SIZE_LABELS = { 1: 'XS', 2: 'S', 3: 'M', 5: 'L', 8: 'XL', 13: 'XXL' };

   function formatSizeLabel(n) {
     if (n == null) return '';
     return SIZE_LABELS[n] || String(n); // REQ-F-007 defensive fallback
   }

   function buildSizeBadgeHtml(size) {
     if (size == null) return '';
     const label = formatSizeLabel(size);
     return `<span class="size-badge" title="size ${escapeHtml(String(size))}">${escapeHtml(label)}</span>`;
   }
   ```

3. **Detail-panel "Size" row** in each `appendXFields(rows, e)` function (lines 3338, 3358, 3377, 3405, 3417). Add at the bottom of each:

   ```js
   if (e.size != null) pushRow(rows, 'Size', escapeHtml(formatSizeLabel(e.size)) + ' (' + escapeHtml(String(e.size)) + ')');
   ```

   And add a new function (between `appendChangeCardFields` and the dispatcher):

   ```js
   function appendIdeaFields(rows, e) {
     if (e.priority != null) pushRow(rows, 'Priority', String(e.priority));
     if (e.size != null) pushRow(rows, 'Size', escapeHtml(formatSizeLabel(e.size)) + ' (' + escapeHtml(String(e.size)) + ')');
   }
   ```

   Then add to the switch at line 3472:

   ```js
   case 'idea': appendIdeaFields(rows, entity); break;
   ```

4. **Sidebar badge** in tree node builders (`buildEpicNodeHtml`, `buildFeatureNodeHtml`, `buildTaskNodeHtml`, `buildFlatSectionHtml`). Inside each, immediately after `<span class="tree-node-title">${escapeHtml(item.title || '')}</span>`, append:

   ```js
   ${buildSizeBadgeHtml(item.size)}
   ```

   The exact insert point in `buildFlatSectionHtml` is line 4884; equivalent insertion points in the epic/feature/task builders are above lines 4754–4804.

5. **No HTML markup changes outside the JS template strings** — there is no separate HTML template region to update.

#### CLI / repository / DB

No changes. Size is already populated by the ListAll repository methods that the viewer service calls (size column is read by the existing `SELECT * FROM bugs/change_cards/ideas` queries).

---

## 3. Test Plan Summary (Detailed plan in next phase)

### 3.1 Service-layer tests (Go)

Six new tests in `internal/services/viewer_service_test.go`, organized as a single table-driven suite where natural:

- Bug: size populated, size nil, JSON omitempty boundary.
- Change-card: size populated, size nil.
- Idea: size populated, size nil.

All tests use the existing `mockViewerBugListRepo` / `mockViewerChangeCardListRepo` / `mockViewerIdeaListRepo` patterns (lines 3113–3168).

### 3.2 Front-end smoke

Manual smoke list (the viewer has no JS unit-test harness today; per epic constraints, no new build dependency is added):

1. With local sample DB, run `shark web` and confirm a bug/change-card/idea with size shows the badge in the sidebar and the Size row in the Info tab.
2. Confirm an entity without a size shows neither badge nor row.
3. Confirm DevTools network panel shows no `null` size keys in `/api/v1/viewer/hierarchy` for entries without size.
4. Confirm DevTools console is error-free.

### 3.3 No regressions

- Existing snapshot/shape tests on `HierarchyResponse` should keep passing.
- `make fmt && make lint && make test` MUST be green.

---

## 4. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Idea's `Size` is at `models.Idea.Size` (no `BaseEntity` embedding), easy to assume same pointer-receiver path as Bug/ChangeCard. | Low | Low (compile-time error) | Spec note in §2.5; integrators read directly from `idea.Size`. |
| HTML escape oversight in size badge → unlikely XSS. | Very Low | Low | Inputs are integer; we still wrap them in `escapeHtml(String(n))` (defensive). |
| Some existing snapshot test asserts EXACT byte sequence of hierarchy JSON for a flat entity with a populated size and was generated before this change. | Low | Medium | The CI run will surface this; update the affected golden file. The change is a strict superset for nil-size cases (REQ-NF-003). |

---

## 5. Exit Gate

- [x] Every requirement is testable (REQ-F-001..008 each map to ≥ 1 AC).
- [x] Every architecture decision references existing patterns (service layer, FlatEntity projection, CLI `formatSize`, model `SizeLabel`) or explicitly justifies deviation (D5 explicitly creates `appendIdeaFields` because no existing pattern exists for ideas).
- [x] File paths listed for every change: `internal/services/viewer_service.go`, `internal/services/viewer_service_test.go`, `internal/viewer/assets/viewer.html`.
- [x] No TBDs in critical sections.
