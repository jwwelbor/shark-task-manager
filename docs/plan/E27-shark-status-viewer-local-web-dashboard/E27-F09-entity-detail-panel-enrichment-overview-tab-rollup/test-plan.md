---
feature_key: E27-F09
doc_type: test-plan
status: draft
author: qa-agent
date: 2026-04-13
---

# Test Plan: E27-F09 — Entity Detail Panel Enrichment

**Feature**: Entity Detail Panel Enrichment — Overview Tab, Rollups, and Clickable Navigation
**Parent Epic**: E27 — Shark Status Viewer — Local Web Dashboard
**Spec**: [spec.md](spec.md)
**Epic UAT Plan**: [uat-plan.md](../uat-plan.md)

---

## 1. Overview

E27-F09 adds an Overview tab (new default) to the entity detail panel for epics, features, and tasks.
It delivers feature/task status rollups, a segmented progress bar, clickable child tables, a
breadcrumb, a dependency block, a notes section, a related-docs section, a work-breakdown section,
and two new backend endpoints (`GET /api/v1/viewer/notes/{key}` and
`GET /api/v1/viewer/related-docs/{key}`). It also relocates the Edit button from the tab bar into
the Spec pane, and adds a status left-border accent to the properties panel.

Changes are incremental on top of E27-F01…F08. All existing functionality must be preserved as
regression gates.

**Files under test** (per spec §2.1 manifest):

| File | Test concern |
| --- | --- |
| `internal/viewer/assets/viewer.html` | HTML/JS structure, string-presence markers |
| `internal/viewer/assets_test.go` | Extend TC-SMOKE-* marker tests |
| `internal/api/viewer/handler.go` | New `Notes` and `RelatedDocs` handlers |
| `internal/api/viewer/handler_test.go` | Handler unit tests (mock service) |
| `internal/services/viewer_service.go` | `Notes` + `RelatedDocs` service methods, DTOs |
| `internal/services/viewer_service_test.go` | Service unit tests (mock repos) |
| `internal/viewer/server/wire.go` | Wiring of `EntityNoteRepository` |

---

## 2. AC Test Matrix

Every acceptance criterion in spec.md §1.1 and §1.2 is covered by at least one named test case.

---

### REQ-F-001 — Overview tab is the default for every entity type

#### TC-F001-1: Overview is the initial active tab on entity selection
- **Type**: JavaScript unit / integration (browser or JSDOM)
- **AC**: AC-001.1
- **Setup**: Any entity node is selected (epic, feature, task).
- **Steps**: Call `selectEntity(key)` or equivalent; inspect the returned DOM / state object.
- **Expected**: `entityViewTab === 'overview'`; the `Overview` button carries `.active`; no
  other tab button carries `.active`.
- **Edge cases**:
  - Entity selection via URL hash (AC-001.5) — `overview` honored; `dashboard` on a non-epic
    coerced to `overview`.
  - Re-selecting the same entity leaves `entityViewTab` unchanged at `'overview'`.

#### TC-F001-2: Epic has exactly five tabs in specified order
- **Type**: HTML marker / string-presence (Go test in `assets_test.go`)
- **AC**: AC-001.2
- **Setup**: `viewer.html` embedded content; epic entity type.
- **Steps**: Assert `strings.Contains(content, "Overview")`, `"Spec"`, `"History"`,
  `"Dashboard"`, `"Files"` exist as tab labels; assert order by finding all five substrings in
  document order.
- **Expected**: All five present; `Dashboard` follows `History`; `Files` is last.
- **Edge cases**: Dashboard tab behaves identically to E27-F08 delivery (regression — see
  TC-REGR-F08).

#### TC-F001-3: Feature and task render no Dashboard tab
- **Type**: DOM assertion (browser / JSDOM)
- **AC**: AC-001.3
- **Setup**: Feature entity selected; task entity selected.
- **Steps**: Assert no element `#ev-tab-dashboard` (or equivalent selector) exists in the
  rendered toggle bar.
- **Expected**: Zero matches for the Dashboard button selector.

#### TC-F001-4: Selecting a new entity resets tab to overview
- **Type**: JavaScript unit
- **AC**: AC-001.4
- **Setup**: User has navigated to the History tab on entity A.
- **Steps**: Click / call `navigateToEntity(entityB)`.
- **Expected**: `entityViewTab === 'overview'` after the navigation.

#### TC-F001-5: Dashboard coercion for non-epic entities (URL restore)
- **Type**: JavaScript unit
- **AC**: AC-001.5
- **Setup**: URL hash / session storage contains `tab=dashboard`; current entity is a feature.
- **Steps**: Restore state.
- **Expected**: `entityViewTab` is coerced to `'overview'`, not left as `'dashboard'`.

---

### REQ-F-002 — Edit button moves from tab bar to inside the Spec pane

#### TC-F002-1: No edit button in the tab bar
- **Type**: HTML marker (Go test in `assets_test.go`)
- **AC**: AC-002.1
- **Setup**: `viewer.html` embedded content.
- **Steps**: `strings.Contains(content, "ev-tab-edit")` must be **false**.
- **Expected**: No `#ev-tab-edit` element exists anywhere in the markup.
- **Edge cases**: Any previously shipped "Edit" tab class names must also be absent.

#### TC-F002-2: Edit toggle renders inside the Spec pane
- **Type**: DOM assertion
- **AC**: AC-002.2
- **Setup**: Entity with a `file_path` is selected; Spec tab is active.
- **Steps**: Assert that a single `✎ Edit` / `View` button exists inside the Spec-pane
  container.
- **Expected**: Exactly one toggle button present inside the Spec pane; zero outside.
- **Edge cases**: Entity without `file_path` — no edit toggle rendered (spec not editable).

#### TC-F002-3: Edit toggle does not change entityViewTab
- **Type**: JavaScript unit
- **AC**: AC-002.3
- **Setup**: Spec tab active (`entityViewTab === 'info'`); entity has a `file_path`.
- **Steps**: Click the edit toggle; then click the View toggle.
- **Expected**: `entityViewTab` remains `'info'` throughout; editor pane appears / disappears.

#### TC-F002-4: Spec save/cancel preserved (E27-F07 regression)
- **Type**: Integration (browser)
- **AC**: AC-002.4
- **Setup**: Entity with an editable spec file.
- **Steps**: Click Edit, modify content, click Save; then repeat with Cancel.
- **Expected**: Save writes the file change; Cancel discards it. No regressions in the
  E27-F07 save/cancel flow.

---

### REQ-F-003 — Breadcrumb under the entity title

#### TC-F003-1: Breadcrumb renders for all entity types
- **Type**: DOM assertion
- **AC**: AC-003.1
- **Setup**: Epic, feature, and task entities loaded.
- **Steps**: For each entity type, assert the breadcrumb element is present.
- **Expected**: Breadcrumb element exists; it is hidden/empty for entities with no parent chain
  (e.g., top-level epic shows only its own key as the non-linked current segment).
- **Edge cases**: Epic breadcrumb is a single non-linked segment.

#### TC-F003-2: Breadcrumb segments are clickable and navigate correctly
- **Type**: DOM assertion + event
- **AC**: AC-003.2
- **Setup**: Task entity at `E27 › E27-F08 › E27-F08-002`.
- **Steps**: Assert each parent segment (`E27`, `E27-F08`) has `data-navigate-key`; simulate
  click on `E27-F08`; assert `navigateToEntity("E27-F08")` was called.
- **Expected**: Click on a parent segment navigates to that entity.

#### TC-F003-3: Current segment is not a link
- **Type**: DOM assertion
- **AC**: AC-003.3
- **Setup**: Any entity.
- **Steps**: Assert the last segment element lacks `data-navigate-key`; assert it carries
  `.current` style class.
- **Expected**: Last breadcrumb segment is unclickable / unstyled as a link.

#### TC-F003-4: Segment keys use short display format
- **Type**: DOM assertion
- **AC**: AC-003.4
- **Setup**: Task `E27-F09-003`.
- **Steps**: Inspect text content of each breadcrumb segment.
- **Expected**: Texts read `E27`, `E27-F09`, `E27-F09-003` (no slug suffixes in display).

---

### REQ-F-004 — Feature Status Rollup on Epic Overview

#### TC-F004-1: Feature rollup only on epic Overview
- **Type**: DOM assertion
- **AC**: AC-004.1
- **Setup**: Epic selected (Overview active); then feature selected.
- **Steps**: Assert rollup section is present for epic; assert it is absent for feature and task.
- **Expected**: Rollup renders exclusively for epic Overview.

#### TC-F004-2: Pill colors match getStatusColor palette
- **Type**: DOM assertion / CSS inspection
- **AC**: AC-004.2
- **Setup**: Epic with features across several statuses.
- **Steps**: For each pill, assert `backgroundColor` (or inline style) matches the return of
  `getStatusColor(status)`.
- **Expected**: No pill uses a hardcoded color not in the workflow palette.

#### TC-F004-3: Pill counts match treeData feature counts
- **Type**: JavaScript unit
- **AC**: AC-004.3
- **Setup**: Mock `treeData` with 3 features: 2 `in_development`, 1 `completed`; no extra
  network call occurs.
- **Steps**: Render Overview; count pills by status.
- **Expected**: `in_development` pill shows `2`; `completed` pill shows `1`; no XHR fired.

#### TC-F004-4: Zero-count statuses omitted from pills
- **Type**: JavaScript unit
- **AC**: AC-004.4
- **Setup**: treeData with features covering only 2 of 8 possible statuses.
- **Expected**: Only the 2 occupied statuses appear as pills.

#### TC-F004-5: Pill order follows workflow phase ordering
- **Type**: JavaScript unit
- **AC**: AC-004.5
- **Setup**: `workflowMeta` loaded with defined phase ordering.
- **Expected**: Pills appear in workflow phase order; fallback to alphabetical when phase data
  is missing.

---

### REQ-F-005 — Task Status Rollup + segmented progress bar

#### TC-F005-1: Task rollup appears on both epic and feature Overview
- **Type**: DOM assertion
- **AC**: AC-005.1 (partial — color consistency with pills)
- **Setup**: Epic and feature both with tasks.
- **Expected**: Rollup section rendered for both; pill and segment colors identical for the
  same status.

#### TC-F005-2: Empty entity shows No-tasks placeholder
- **Type**: DOM assertion
- **AC**: AC-005.2
- **Setup**: Feature / epic with zero tasks in treeData.
- **Expected**: A dim `No tasks` placeholder replaces the progress bar; zero-width bar is
  not rendered.

#### TC-F005-3: Progress percentage matches weighted formula
- **Type**: JavaScript unit
- **AC**: AC-005.3
- **Setup**: Mock `workflowMeta` with known `progress_weight` per status; mock treeData with
  known task counts per status.
  - Example: 2 tasks `completed` (weight 100), 1 task `in_progress` (weight 50), 1 task
    `todo` (weight 0) → weighted = (200 + 50 + 0) / 4 = 62.5 → `63%` (rounded).
- **Steps**: Render Overview; read the `NN%` label.
- **Expected**: Label matches the formula `Σ(weight × count) / total_tasks * 100`.
- **Edge cases**: All tasks completed → `100%`; zero tasks → placeholder (AC-005.2).

#### TC-F005-4: No new network requests for rollup data
- **Type**: JavaScript unit / network spy
- **AC**: AC-005.4
- **Setup**: XHR/fetch spy installed; entity selected.
- **Expected**: Opening Overview issues no requests for rollup data; only the two new
  endpoint calls (notes, related-docs) may fire asynchronously.

---

### REQ-F-006 — Work Breakdown on Feature Overview

#### TC-F006-1: Three responsibility rows computed from workflowMeta
- **Type**: JavaScript unit
- **AC**: AC-006.1
- **Setup**: Mock `workflowMeta` with known `responsibility` fields per task status; mock
  feature tasks in treeData.
- **Steps**: Render feature Overview; inspect Agent, Human, QA Team row counts.
- **Expected**: Counts match tally of tasks by `responsibility`; statuses with no explicit
  `responsibility` (or `none`) are excluded from the three rows.

#### TC-F006-2: Bar widths proportional to max row count
- **Type**: JavaScript unit
- **AC**: AC-006.2
- **Setup**: Agent=4, Human=2, QA Team=1 (max=4).
- **Expected**: Agent bar = 100%, Human bar = 50%, QA Team bar = 25%.

#### TC-F006-3: Section hidden when all counts zero
- **Type**: DOM assertion
- **AC**: AC-006.3
- **Setup**: Feature with tasks all in `none`-responsibility statuses.
- **Expected**: Work Breakdown section is not present in the DOM.

#### TC-F006-4: Section only on feature Overview
- **Type**: DOM assertion
- **AC**: AC-006.4
- **Setup**: Epic selected; task selected.
- **Expected**: Work Breakdown section absent for epic and task.

---

### REQ-F-007 — Action Items on Feature Overview

#### TC-F007-1: No hardcoded status names in blocking filter
- **Type**: Static analysis (grep test in `assets_test.go`)
- **AC**: AC-007.1
- **Steps**: Assert the JS block responsible for Action Items does not contain any string
  literal like `=== 'ready_for_approval'` or `=== 'ready_for_review'` or similar.
- **Expected**: Zero matches; only `blocks_feature === true` comparison used.

#### TC-F007-2: Action Item rows display correctly
- **Type**: DOM assertion
- **AC**: AC-007.2
- **Setup**: Feature with tasks in `blocks_feature=true` statuses.
- **Expected**: Each row has a status badge, clickable key (`data-navigate-key`), and a
  truncated title (CSS `text-overflow: ellipsis`).

#### TC-F007-3: Rows ordered by phase then execution_order then key
- **Type**: JavaScript unit
- **AC**: AC-007.3
- **Setup**: Three action-item tasks: different phases, different `execution_order` values.
- **Expected**: Rows appear in the specified order.

#### TC-F007-4: Section omitted when no blocking tasks
- **Type**: DOM assertion
- **AC**: AC-007.4
- **Setup**: Feature with no tasks in `blocks_feature=true` statuses.
- **Expected**: Action Items section not present in the DOM.

---

### REQ-F-008 — Clickable child-entity tables

#### TC-F008-1: KEY cells are clickable and navigate
- **Type**: DOM assertion + event
- **AC**: AC-008.1
- **Setup**: Epic overview with features in child table; feature overview with tasks.
- **Steps**: Assert each KEY cell has `data-navigate-key` and `.clickable`; simulate click;
  assert `navigateToEntity` is called with the correct key.

#### TC-F008-2: TITLE cells truncate via CSS, not JS slicing
- **Type**: DOM assertion + static check
- **AC**: AC-008.2
- **Steps**: Assert TITLE `<td>` has CSS `text-overflow` style (not a JS `.substring()`
  call); assert `title` attribute holds the full title.

#### TC-F008-3: Table visual rules (no outer border, hover state)
- **Type**: CSS inspection
- **AC**: AC-008.3
- **Expected**: Table container has no outer border; row `tr` uses only bottom-border
  dividers; hover applies `background: var(--bg-3)`.

#### TC-F008-4: Overflow scroll at >10 children with sticky header
- **Type**: DOM assertion
- **AC**: AC-008.4
- **Setup**: Render an epic with 15 features.
- **Expected**: Table body container has `max-height: 420px` and `overflow-y: auto` (or
  `scroll`); header row has `position: sticky; top: 0`.

#### TC-F008-5: Progress bar matches properties panel value
- **Type**: JavaScript unit
- **AC**: AC-008.5
- **Setup**: Feature treeData entry with `progress_pct: 75`.
- **Expected**: Progress bar width style equals `75%`; matches the properties-panel value.

---

### REQ-F-009 — Dependency block on Task Overview

#### TC-F009-1: Dependency block only on task Overview
- **Type**: DOM assertion
- **AC**: AC-009.1
- **Setup**: Epic and feature selected.
- **Expected**: Dependency block not present for epic or feature.

#### TC-F009-2: Data sourced from cached task object, no new endpoint
- **Type**: JavaScript unit + network spy
- **AC**: AC-009.2
- **Setup**: Task with `depends_on=["E27-F09-001"]`, `blocked_by=[]`, `blocks=["E27-F09-003"]`.
- **Expected**: No XHR for dependency data; values read directly from the task's cached fields.

#### TC-F009-3: Resolved keys are clickable; unresolved show dim suffix
- **Type**: DOM assertion
- **AC**: AC-009.3 + AC-009.4
- **Setup**: `depends_on` contains a key that resolves via `findEntityByKey`, and one that
  does not.
- **Expected**: Resolved key has `data-navigate-key`; unresolved renders as plain mono text
  with `(unresolved)` suffix.

#### TC-F009-4: Blocked By section has red left-border accent
- **Type**: CSS inspection
- **AC**: AC-009.5
- **Setup**: Task with non-empty `blocked_by` list.
- **Expected**: The `Blocked By` sub-section has `border-left: 2px solid #c0392b`.

#### TC-F009-5: Empty groups show (none) in dim
- **Type**: DOM assertion
- **AC**: AC-009.3 (implicit empty group behavior)
- **Setup**: Task with no `depends_on` entries.
- **Expected**: `Depends On` group renders `(none)` in a dim style; group is present but
  not hidden.

---

### REQ-F-010 — Notes section (all entity types)

#### TC-F010-1: Notes fetched from new endpoint and cached
- **Type**: JavaScript unit + network spy
- **AC**: AC-010.1
- **Setup**: Entity key `E27-F09`; mock response returning 3 notes.
- **Steps**: Navigate to entity; verify one GET to `/api/v1/viewer/notes/E27-F09`; navigate
  away; navigate back.
- **Expected**: Exactly one fetch on first view; no second fetch on revisit.

#### TC-F010-2: Only 5 most recent notes shown by default
- **Type**: DOM assertion
- **AC**: AC-010.2
- **Setup**: Mock response with 8 notes (ordered newest-first per AC-020.2).
- **Expected**: 5 rows visible; `Show 3 more` button present.

#### TC-F010-3: Note row fields and formatting
- **Type**: DOM assertion
- **AC**: AC-010.3
- **Setup**: Mock note with known `created_at`, `note_type`, and `content`.
- **Expected**: Timestamp formatted by `formatDate`; type pill colored per AC-010.4 mapping;
  content truncated with full text in `title` attribute.

#### TC-F010-4: Type-pill color mapping
- **Type**: DOM assertion / style check
- **AC**: AC-010.4
- **Setup**: One note per type: `decision`, `blocker`, `comment`, `solution`, `implementation`,
  `question`, `rejection`, `reference`, `future`, `requirement`, `testing`.
- **Expected**: Each type pill matches the exact hex or CSS var specified in AC-010.4.

#### TC-F010-5: Show-more / Show-less inline expansion
- **Type**: DOM assertion + event
- **AC**: AC-010.5
- **Setup**: 8 notes in response.
- **Steps**: Click `Show 3 more`; assert all 8 rows visible; assert `Show less` button present;
  click `Show less`; assert back to 5 rows.
- **Expected**: No scroll jump; no DOM reconstruction (rows appended/hidden in-place).

#### TC-F010-6: Section omitted when entity has no notes
- **Type**: DOM assertion
- **AC**: AC-010.6
- **Setup**: Mock response with `notes: []`.
- **Expected**: Notes section element not present in the DOM.

#### TC-F010-7: Skeleton loader while fetch is pending
- **Type**: DOM assertion (async timing)
- **AC**: AC-010.7
- **Setup**: Delay the mock fetch by 100ms.
- **Expected**: `.skeleton` row visible before response; other Overview sections rendered
  independently without waiting for notes.

---

### REQ-F-011 — Related Docs section (feature + task)

#### TC-F011-1: Each row shows doc-icon, truncated path, and open button
- **Type**: DOM assertion
- **AC**: AC-011.1
- **Setup**: Mock related-docs response with 2 documents.
- **Expected**: SVG icon present; path rendered as mono text truncated by CSS; `open` button
  present per row.

#### TC-F011-2: Open button routes to Doc View
- **Type**: Event assertion
- **AC**: AC-011.2
- **Steps**: Click `open` button; assert `openDocumentByPath()` (or equivalent) is called
  with the document's file path; assert viewer state transitions to `state=4`.

#### TC-F011-3: Section omitted when no related docs
- **Type**: DOM assertion
- **AC**: AC-011.3
- **Setup**: Mock response with `docs: []`.
- **Expected**: Related Docs section not rendered.

#### TC-F011-4: Response cached per session
- **Type**: Network spy
- **AC**: AC-011.4
- **Steps**: Navigate away and back; assert only one fetch per session.

#### TC-F011-5: Graceful failure rendering
- **Type**: DOM assertion
- **AC**: AC-011.5
- **Setup**: Mock fetch returns 500.
- **Expected**: Dim `Failed to load docs` message shown; no JS exception; rest of Overview
  unaffected.

---

### REQ-F-012 — Status left-border accent on properties panel

#### TC-F012-1: Props grid has inline border matching status color
- **Type**: DOM / CSS assertion
- **AC**: AC-012.1
- **Setup**: Entity with `status: "in_development"` (known color from `getStatusColor`).
- **Expected**: `.props-grid` has inline `border-left: 3px solid <color>`.

#### TC-F012-2: No border when status is unknown
- **Type**: DOM assertion
- **AC**: AC-012.2
- **Setup**: Entity with empty or unrecognized `status`.
- **Expected**: No `border-left` style applied; layout unchanged.

#### TC-F012-3: Padding compensation does not shift grid
- **Type**: CSS regression
- **AC**: AC-012.3
- **Steps**: Compare `.props-grid` column alignment with and without a status border.
- **Expected**: Adding the 3px left border does not shift column content; `padding-left`
  compensates.

---

### REQ-F-020 — `GET /api/v1/viewer/notes/{key}` endpoint

#### TC-F020-1: Response JSON shape matches contract
- **Type**: Go unit test (`handler_test.go`)
- **AC**: AC-020.1
- **Setup**: Mock service returning `NotesResponse` with 2 notes; call handler via
  `httptest.NewRecorder`.
- **Steps**: `GET /api/v1/viewer/notes/E27-F09`; decode response.
- **Expected**: Fields `entity_type`, `entity_key`, `notes` present; `notes` is `[]` not
  `null` when empty.

#### TC-F020-2: Notes ordered by created_at DESC
- **Type**: Go unit test (`viewer_service_test.go`)
- **AC**: AC-020.2
- **Setup**: Mock `ViewerEntityNoteRepository.GetByEntity` returning 3 notes in ASC order.
- **Expected**: `Notes` slice in response is in reverse (DESC) order.

#### TC-F020-3: Only the six specified fields are serialised
- **Type**: Go unit test
- **AC**: AC-020.3
- **Steps**: JSON-encode a `NoteDTO`; assert no `metadata` or `updated_at` keys present.
- **Expected**: Only `id`, `note_type`, `content`, `created_by`, `created_at` exposed.

#### TC-F020-4: Handler accepts all key shapes
- **Type**: Go unit test (table-driven in `handler_test.go`)
- **AC**: AC-020.4
- **Cases**:
  - Short task key `E27-F09-003`
  - Long task key `T-E27-F09-003`
  - Slugged key `E27-F09-003-some-slug`
  - Uppercase `E27-F09`
  - Lowercase `e27-f09`
- **Expected**: All are normalised and resolved without error; 404 only when entity not found.

#### TC-F020-5: 404 returned for unknown entity
- **Type**: Go unit test
- **AC**: AC-020.1 (404 contract)
- **Setup**: Mock service returns `NotFoundError`.
- **Expected**: HTTP 404; error body does not contain stack traces.

#### TC-F020-6: 400 returned for malformed key
- **Type**: Go unit test
- **Setup**: Key `../../etc/passwd`.
- **Expected**: HTTP 400; `validateAndNormalizeAnyKey` rejects it; no file I/O.

#### TC-F020-7: CORS and auth behaviour identical to existing endpoints
- **Type**: Go unit test (reuse patterns from `cors_test.go`)
- **AC**: AC-020.5
- **Steps**: Send request with `Origin: https://evil.example.com`; send with
  `Origin: http://localhost:5173`.
- **Expected**: Evil origin blocked; localhost origin echoed. No new security surface.

#### TC-F020-8: nil noteRepo degrades gracefully to empty notes
- **Type**: Go unit test (`viewer_service_test.go`)
- **AC**: AC-020.1 (empty-array guarantee when wiring incomplete)
- **Setup**: `ViewerService` constructed without calling `WithNoteRepo`.
- **Expected**: `Notes(ctx, key)` returns `&NotesResponse{Notes: []NoteDTO{}}`, no panic.

---

### REQ-F-021 — `GET /api/v1/viewer/related-docs/{key}` endpoint

#### TC-F021-1: Empty list returns `{"docs": []}` not null
- **Type**: Go unit test
- **AC**: AC-021.1
- **Setup**: Mock service returns `RelatedDocsResponse` with empty docs.
- **Expected**: `"docs":[]` in JSON body, not `null`.

#### TC-F021-2: Documents ordered most-recent-link-first
- **Type**: Go unit test
- **AC**: AC-021.2
- **Setup**: Mock `ListForEntity` returns 3 docs in a specific order.
- **Expected**: Response preserves the repository ordering (most-recent-link-first per
  `EntityDocumentRepository.ListForEntity` semantics).

#### TC-F021-3: Paths returned as stored (relative to project root)
- **Type**: Go unit test
- **AC**: AC-021.3
- **Setup**: Document with `file_path: "docs/plan/E27-F09/spec.md"`.
- **Expected**: Path returned verbatim; no client-side resolution by the service.

#### TC-F021-4: Key normalisation, CORS, auth match REQ-F-020
- **Type**: Go unit test
- **AC**: AC-021.4
- **Steps**: Same table-driven key-shape test as TC-F020-4; same CORS probe as TC-F020-7.
- **Expected**: Consistent behaviour across both new endpoints.

---

### REQ-NF-001 — Overview renders ≤ 300ms from entity selection

#### TC-NF001-1: Synchronous sections rendered without network gate
- **Type**: JavaScript unit (timing assertion)
- **AC**: AC-NF-001.1
- **Setup**: Stub notes/related-docs fetch to delay 500ms.
- **Steps**: Time from `selectEntity()` call to first paint of Overview sections (breadcrumb,
  rollup, child tables, dependency block).
- **Expected**: Synchronous sections present in DOM before 300ms; notes skeleton shows;
  async sections fill in later.

#### TC-NF001-2: Notes and Related Docs render asynchronously
- **Type**: DOM ordering assertion
- **AC**: AC-NF-001.2
- **Expected**: After Overview renders, notes section shows `.skeleton`; other sections
  are complete; skeleton replaced when fetch resolves.

---

### REQ-NF-002 — Accessibility / keyboard navigation

#### TC-NF002-1: Navigable keys are keyboard-focusable
- **Type**: DOM assertion
- **AC**: AC-NF-002.1
- **Setup**: Overview rendered with clickable keys (breadcrumb segments, child-table KEY
  cells, dependency entries, action-item keys).
- **Expected**: Each has `tabindex="0"` and `role="button"`; `Enter` / `Space` events invoke
  the same navigation handler as `click`.

#### TC-NF002-2: WCAG AA contrast for new pills and Work-Breakdown bars
- **Type**: Color-contrast assertion (using `getContrastColor()` helper output)
- **AC**: AC-NF-002.2
- **Setup**: Status colors from workflow config (including the red `#c0392b` accent).
- **Expected**: Text color returned by `getContrastColor(bgColor)` achieves ≥ 4.5:1 ratio
  against `var(--bg-2)` (verified with computed values).

---

### REQ-NF-003 — Client-side data reuse; no per-entity full-entity fetch

#### TC-NF003-1: No full-entity GET on entity selection
- **Type**: Network spy
- **AC**: AC-NF-003.1
- **Setup**: XHR/fetch spy; entity clicked in tree.
- **Expected**: No request to `/api/v1/viewer/entity/{key}` or equivalent full-object
  endpoint; only `/notes/{key}` and `/related-docs/{key}` may fire.

#### TC-NF003-2: Existing endpoint set unchanged (E27-F08 regression)
- **Type**: Static / integration
- **AC**: AC-NF-003.2
- **Steps**: Assert that all E27-F08 endpoints (`/summary`, `/hierarchy`, `/history/{key}`,
  `/file/{key}`, `/recent-activity`, `/workflow-meta`, `/folder-files/{path}`,
  `/features/{key}/tasks`, `PUT /edit/file`) still exist and respond correctly.
- **Expected**: No breaking changes to the existing endpoint set.

---

### E2E Acceptance Scenarios (spec §1.3)

#### TC-E2E-1: Developer clicks an epic
- **Trace**: spec §1.3 Scenario 1
- **Steps**: Load viewer; click `E27` in sidebar.
- **Expected**: Overview active (`entityViewTab='overview'`); Feature Status Rollup
  visible without XHR; Features table with clickable KEY column; breadcrumb reads `E27`;
  five tab buttons in order: `Overview · Spec · History · Dashboard · Files`.

#### TC-E2E-2: Drill from epic to feature via features table
- **Trace**: spec §1.3 Scenario 2
- **Steps**: Click `E27-F08` in the Features table.
- **Expected**: Panel navigates to `E27-F08`; Overview active; breadcrumb reads
  `E27 › E27-F08`; tab bar is `Overview · Spec · History · Files` (no Dashboard).

#### TC-E2E-3: Blocked task dependency inspection
- **Trace**: spec §1.3 Scenario 3
- **Steps**: Navigate to a task with `blocked_by: ["E27-F09-001"]`.
- **Expected**: `Blocked By` group has red left-border accent; `E27-F09-001` is clickable
  and navigates; `Depends On` and `Blocks` each show `(none)` if empty.

#### TC-E2E-4: Notes expand flow
- **Trace**: spec §1.3 Scenario 4
- **Steps**: Click feature with 8 notes; verify one GET to notes endpoint; 5 rows visible;
  `Show 3 more` button appears; clicking it reveals remaining rows.

#### TC-E2E-5: Breadcrumb navigation
- **Trace**: spec §1.3 Scenario 5
- **Steps**: Click `E27-F08` in the breadcrumb of a task panel.
- **Expected**: Viewer navigates to `E27-F08`; sidebar selection updates.

#### TC-E2E-6: E27-F08 Dashboard regression
- **Trace**: spec §1.3 Scenario 6 (AC-001.2 regression gate)
- **Steps**: With epic panel open, click `Dashboard` tab.
- **Expected**: E27-F08 dashboard renders unchanged: Entity Charts, Status Overview,
  Feature Progress, Recent Activity, Stale Entities all present.

---

### HTML Smoke / Marker Tests (`assets_test.go` extensions)

These extend the existing `TestViewerHTMLEmbedded` pattern and run as fast Go string-presence
tests against the embedded `viewer.html`.

| Test ID | Marker / assertion | AC |
| --- | --- | --- |
| TC-SMOKE-F09-01 | `>Overview<` tab label present | AC-001.1 |
| TC-SMOKE-F09-02 | `Work Breakdown` string present | AC-006.1 |
| TC-SMOKE-F09-03 | `Action Items` string present | AC-007.1 |
| TC-SMOKE-F09-04 | `Depends On` string present | AC-009.1 |
| TC-SMOKE-F09-05 | `Blocked By` string present | AC-009.1 |
| TC-SMOKE-F09-06 | `Blocks` string (task dependency) present | AC-009.1 |
| TC-SMOKE-F09-07 | `ev-tab-edit` **absent** | AC-002.1 |
| TC-SMOKE-F09-08 | `api/v1/viewer/notes/` string present | AC-020.1 |
| TC-SMOKE-F09-09 | `api/v1/viewer/related-docs/` string present | AC-021.1 |
| TC-SMOKE-F09-10 | No `=== 'ready_for_approval'` literal in JS | AC-007.1 |
| TC-SMOKE-F09-11 | No `=== 'ready_for_review'` literal in JS | AC-007.1 |
| TC-SMOKE-F09-12 | `blocks_feature` property reference present in JS | AC-007.1 |

---

## 3. Integration Scenarios

These test cross-component boundaries and map to the parent epic UAT plan.

### INT-1: Handler → Service → Mock Repo (notes endpoint)

**Components**: `handler.go` + `viewer_service.go` + mock `ViewerEntityNoteRepository`

**What to verify**:
- Handler parses and normalises the `{key}` path parameter, calls `svc.Notes(ctx, key)`.
- Service resolves entity type from key, calls `noteRepo.GetByEntity`, reverses order, maps
  to DTOs.
- Handler serialises response; status codes map correctly (200, 400, 404).

**Epic UAT reference**: Area D (Entity View) — D3 Properties panel; Area I (Cross-Feature) — I1
Full loop.

### INT-2: Handler → Service → Mock Repo (related-docs endpoint)

**Components**: `handler.go` + `viewer_service.go` + mock `ViewerEntityDocByEntityRepository`

**What to verify**: Analogous to INT-1 but for related-docs. Confirms `ListForEntity` result
mapped to `RelatedDocDTO`; empty list marshals as `[]`.

**Epic UAT reference**: Area D — D1 Spec markdown renders.

### INT-3: Wiring — EntityNoteRepository injected via wire.go

**Components**: `wire.go` + concrete `EntityNoteRepository` + `ViewerService.WithNoteRepo`

**What to verify**:
- `wire.go` calls `svc.WithNoteRepo(noteRepo)` with a non-nil concrete repo.
- A full live test (against a real test DB) fetches notes for a known entity and gets the
  expected count and order.
- Omitting the wiring call (nil noteRepo) does not panic — degrades to empty notes
  (TC-F020-8).

**Epic UAT reference**: Area F — F1 Local SQLite, F2 Turso cloud.

### INT-4: JavaScript Overview → Existing API (treeData / workflowMeta)

**Components**: `renderEntityView` JS + cached `treeData` + cached `workflowMeta`

**What to verify**:
- Rollup counts (feature status, task status, work breakdown, action items) all derived
  solely from already-loaded data structures.
- No additional XHR fired when switching between Overview tabs for different entities.
- Data parity: rollup counts for feature tasks match those returned by
  `GET /api/v1/viewer/hierarchy` and `GET /api/v1/viewer/features/{key}/tasks`.

**Epic UAT reference**: Area B — B1 Summary cards reflect database; I2 Edit with CLI,
refresh in browser.

### INT-5: E27-F07 + E27-F08 Regression (Edit button relocation + Dashboard tab)

**Components**: F07 inline editor + F08 Dashboard pane + F09 Spec pane

**What to verify**:
- Edit toggle works inside the Spec pane for all entity types that had it before.
- Dashboard tab for epics renders all E27-F08 content unchanged.
- `entityViewTab` state machine transitions (`overview` ↔ `info` ↔ `history` ↔ `dashboard` ↔
  `files`) are all valid.

**Epic UAT reference**: Area D — D4 History button; spec §1.3 Scenario 6.

### INT-6: Security — New endpoints respect existing CORS / path-traversal rules

**Components**: `handler.go` CORS middleware + `validateAndNormalizeAnyKey`

**What to verify**:
- `notes/{key}` and `related-docs/{key}` reject non-localhost origins (H2).
- Malformed / path-traversal keys return 400 without leaking filesystem paths (H4).
- No POST/PUT/PATCH/DELETE accepted on the new routes (H6).

**Epic UAT reference**: Area H — H2 CORS rejects non-local; H4 Path-traversal probe; H6 No
write endpoints.

---

## 4. Test Infrastructure

### Existing patterns to follow

| Pattern | Location | Use for |
| --- | --- | --- |
| HTML string-presence tests | `internal/viewer/assets_test.go` | TC-SMOKE-F09-* marker tests |
| Mock service with func-field pattern | `internal/api/viewer/handler_test.go:MockViewerServicer` | Notes / RelatedDocs handler tests |
| Mock repo with func-field pattern | `internal/services/viewer_service_test.go:mockViewerEpicRepo` etc. | Notes / RelatedDocs service tests |
| `httptest.NewRecorder` + `httptest.NewRequest` | `internal/api/viewer/handler_test.go` | All HTTP handler tests |
| CORS probe helper | `internal/api/viewer/cors_test.go` | TC-F020-7, TC-F021-4 |

### New test helpers needed

1. **`mockViewerNoteRepo`** — to be added to `internal/services/viewer_service_test.go`.
   Struct with `GetByEntityFunc func(ctx, entityType, entityID) ([]*models.EntityNote, error)`.

2. **`mockViewerDocByEntityRepo`** — to be added to `internal/services/viewer_service_test.go`.
   Struct with `ListForEntityFunc func(ctx, entityType, entityID) ([]*models.Document, error)`.

3. **`MockViewerServicer` extension** — add `NotesFunc` and `RelatedDocsFunc` fields to the
   existing mock in `internal/api/viewer/handler_test.go`.

4. **`assertNoHardcodedStatusLiteral(t, content, literal string)`** — helper in
   `internal/viewer/assets_test.go` that calls `strings.Contains(content, literal)` and fails
   if found. Used for TC-SMOKE-F09-10 and TC-SMOKE-F09-11.

### Test file placement

| Test case group | File |
| --- | --- |
| TC-SMOKE-F09-* | `internal/viewer/assets_test.go` (extend existing file) |
| TC-F020-*, TC-F021-* (handler) | `internal/api/viewer/handler_test.go` (extend existing file) |
| TC-F020-* (service), TC-F021-* (service) | `internal/services/viewer_service_test.go` (extend existing file) |
| TC-F001-* through TC-NF003-* (JS) | Not Go tests — validated via manual browser testing or a future JSDOM harness |
| TC-E2E-* | Manual browser testing (UAT); automated E2E is out of scope for Phase 1 |

### Test isolation rules

Per project rules (`CLAUDE.md` and `.claude/rules/testing/architecture.md`):
- **Service tests** (`viewer_service_test.go`): always mock repos; never use real DB.
- **Handler tests** (`handler_test.go`): always mock `ViewerServicer` interface; never call
  service implementation.
- **Assets tests** (`assets_test.go`): pure string operations on the embedded HTML; no I/O.
- **Repository tests** (if added): use `test.GetTestDB()` with explicit cleanup.

---

## 5. Out of Scope

Per spec §1.4:

1. E27-F08 Dashboard content is not re-tested in full — only the regression gate TC-REGR-F08
   / TC-E2E-6 is required.
2. WebSocket / SSE live-refresh is not in scope.
3. Notes / related-docs creation or editing (read-only in this feature).
4. Bug / Change-Card Overview panels beyond the Notes section.
5. Drag-and-drop reordering of child tables.
6. Per-entity full-object GET endpoint.
7. Automated JSDOM / Playwright E2E tests — deferred to a future E27-F10 automation epic.

---

## 6. AC Coverage Summary

| AC | Test Case(s) | Covered |
| --- | --- | --- |
| AC-001.1 | TC-F001-1, TC-SMOKE-F09-01 | Yes |
| AC-001.2 | TC-F001-2, TC-E2E-1 | Yes |
| AC-001.3 | TC-F001-3, TC-E2E-2 | Yes |
| AC-001.4 | TC-F001-4 | Yes |
| AC-001.5 | TC-F001-5 | Yes |
| AC-002.1 | TC-F002-1, TC-SMOKE-F09-07 | Yes |
| AC-002.2 | TC-F002-2 | Yes |
| AC-002.3 | TC-F002-3 | Yes |
| AC-002.4 | TC-F002-4 | Yes |
| AC-003.1 | TC-F003-1 | Yes |
| AC-003.2 | TC-F003-2, TC-E2E-5 | Yes |
| AC-003.3 | TC-F003-3 | Yes |
| AC-003.4 | TC-F003-4 | Yes |
| AC-004.1 | TC-F004-1 | Yes |
| AC-004.2 | TC-F004-2 | Yes |
| AC-004.3 | TC-F004-3 | Yes |
| AC-004.4 | TC-F004-4 | Yes |
| AC-004.5 | TC-F004-5 | Yes |
| AC-005.1 | TC-F005-1 | Yes |
| AC-005.2 | TC-F005-2 | Yes |
| AC-005.3 | TC-F005-3 | Yes |
| AC-005.4 | TC-F005-4, TC-NF003-1 | Yes |
| AC-006.1 | TC-F006-1 | Yes |
| AC-006.2 | TC-F006-2 | Yes |
| AC-006.3 | TC-F006-3 | Yes |
| AC-006.4 | TC-F006-4 | Yes |
| AC-007.1 | TC-F007-1, TC-SMOKE-F09-10, TC-SMOKE-F09-11, TC-SMOKE-F09-12 | Yes |
| AC-007.2 | TC-F007-2 | Yes |
| AC-007.3 | TC-F007-3 | Yes |
| AC-007.4 | TC-F007-4 | Yes |
| AC-008.1 | TC-F008-1 | Yes |
| AC-008.2 | TC-F008-2 | Yes |
| AC-008.3 | TC-F008-3 | Yes |
| AC-008.4 | TC-F008-4 | Yes |
| AC-008.5 | TC-F008-5 | Yes |
| AC-009.1 | TC-F009-1 | Yes |
| AC-009.2 | TC-F009-2 | Yes |
| AC-009.3 | TC-F009-3, TC-F009-5 | Yes |
| AC-009.4 | TC-F009-3 | Yes |
| AC-009.5 | TC-F009-4, TC-E2E-3 | Yes |
| AC-010.1 | TC-F010-1, TC-E2E-4 | Yes |
| AC-010.2 | TC-F010-2, TC-E2E-4 | Yes |
| AC-010.3 | TC-F010-3 | Yes |
| AC-010.4 | TC-F010-4 | Yes |
| AC-010.5 | TC-F010-5, TC-E2E-4 | Yes |
| AC-010.6 | TC-F010-6 | Yes |
| AC-010.7 | TC-F010-7 | Yes |
| AC-011.1 | TC-F011-1 | Yes |
| AC-011.2 | TC-F011-2 | Yes |
| AC-011.3 | TC-F011-3 | Yes |
| AC-011.4 | TC-F011-4 | Yes |
| AC-011.5 | TC-F011-5 | Yes |
| AC-012.1 | TC-F012-1 | Yes |
| AC-012.2 | TC-F012-2 | Yes |
| AC-012.3 | TC-F012-3 | Yes |
| AC-020.1 | TC-F020-1, TC-F020-5, TC-F020-6, TC-F020-8 | Yes |
| AC-020.2 | TC-F020-2 | Yes |
| AC-020.3 | TC-F020-3 | Yes |
| AC-020.4 | TC-F020-4 | Yes |
| AC-020.5 | TC-F020-7 | Yes |
| AC-021.1 | TC-F021-1 | Yes |
| AC-021.2 | TC-F021-2 | Yes |
| AC-021.3 | TC-F021-3 | Yes |
| AC-021.4 | TC-F021-4 | Yes |
| AC-NF-001.1 | TC-NF001-1 | Yes |
| AC-NF-001.2 | TC-NF001-2 | Yes |
| AC-NF-002.1 | TC-NF002-1 | Yes |
| AC-NF-002.2 | TC-NF002-2 | Yes |
| AC-NF-003.1 | TC-NF003-1, TC-F005-4 | Yes |
| AC-NF-003.2 | INT-5, TC-E2E-6 | Yes |

**Total ACs from spec**: 57
**All covered**: Yes — every AC has at least one named test case.
