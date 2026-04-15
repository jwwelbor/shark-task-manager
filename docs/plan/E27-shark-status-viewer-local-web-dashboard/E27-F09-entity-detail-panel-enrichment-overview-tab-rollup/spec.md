---
feature_key: E27-F09-entity-detail-panel-enrichment-overview-tab-rollup
epic_key: E27
title: "Spec: Entity Detail Panel Enrichment — Overview Tab, Rollups, and Clickable Navigation"
type: combined-spec
---

# Spec: Entity Detail Panel Enrichment — Overview Tab, Rollups, and Clickable Navigation

**Feature Key**: E27-F09
**Parent Epic**: [E27 PRD](../epic.md) · [E27 Architecture](../architecture.md)
**Feature description**: [feature.md](feature.md)

> This spec is **incremental** on top of the viewer delivered in E27-F01…F08. See
> [feature.md](feature.md) for business context, user stories, and detailed visual design. This
> document covers (a) the testable requirements and (b) the architecture for implementation.
>
> Conventions follow the E27-F08 spec (see
> [../E27-F08…/spec.md](../E27-F08-epic-level-dashboard-and-enhanced-entity-info/spec.md))
> — single-file SPA at `internal/viewer/assets/viewer.html`, minimal new Go code, new endpoints
> added through the existing viewer handler + service.

---

## 1. Requirements

All requirements trace to user stories and the `Requirements` block in
[feature.md §User Stories / §Requirements](feature.md).

### 1.1 Functional Requirements

#### REQ-F-001 — Overview tab becomes the default tab for every entity type

**Trace**: feature.md REQ-F-001, Story 1/2/3.

A new **Overview** tab is added to `renderEntityView()` for epic, feature, and task entities.
Overview is the new default tab on entity selection. The **Dashboard** tab introduced by E27-F08
for epics is **preserved**; tab order becomes:

| Entity  | Tab order                                                 |
| ------- | --------------------------------------------------------- |
| epic    | `Overview` · `Spec` · `History` · `Dashboard` · `Files`   |
| feature | `Overview` · `Spec` · `History` · `Files`                 |
| task    | `Overview` · `Spec` · `History` · `Files`                 |

**Testable ACs**:

- **AC-001.1**: When any entity node is clicked, `entityViewTab === 'overview'` and the Overview
  button carries the `.active` class.
- **AC-001.2**: For an epic, the five tab buttons render in the order listed above; `Dashboard`
  is present and still behaves exactly as E27-F08 delivered (verified by clicking it and
  asserting `renderEpicDashboardPane` runs — regression gate on AC-001.8 of E27-F08).
- **AC-001.3**: For a feature or task, no `Dashboard` button is rendered.
- **AC-001.4**: Clicking a different entity in the tree resets `entityViewTab` to `'overview'`
  (not to `'info'`).
- **AC-001.5**: When a URL hash / back-nav restores a non-overview tab value that is valid for
  the current entity, that tab is honored; if the restored value is `'dashboard'` on a
  non-epic entity, it is coerced to `'overview'` (guard).

#### REQ-F-002 — Move `Edit` from a tab-bar button into an inline toggle inside the Spec tab

**Trace**: feature.md REQ-F-002, Story 1 (implicit — tab bar shown in design).

The `✎ Edit` button is removed from the toggle bar and re-rendered inside the **Spec** tab as
an inline button (top-right of the markdown pane). Clicking it toggles the existing E27-F07
inline editor without changing `entityViewTab`.

**Testable ACs**:

- **AC-002.1**: No `#ev-tab-edit` element exists in the toggle bar for any entity type.
- **AC-002.2**: When the Spec pane renders for an entity with a `file_path` or
  `linked_content_path`, a single `✎ Edit` / `View` mode-toggle button appears inside the pane.
- **AC-002.3**: Clicking the Edit toggle swaps the pane to the E27-F07 editor and back, and
  `entityViewTab` remains `'info'`.
- **AC-002.4**: Existing Spec/edit save/cancel behaviour from E27-F07 is preserved.

#### REQ-F-003 — Breadcrumb under the entity title

**Trace**: feature.md REQ-F-010, Story 5.

A breadcrumb renders beneath the entity title using the parent chain derived from
`findEntityByKey` + the cached `hierarchyData`:

| Entity type | Breadcrumb render                      |
| ----------- | -------------------------------------- |
| epic        | `E27`                                  |
| feature     | `E27 › E27-F08`                        |
| task        | `E27 › E27-F08 › E27-F08-002`          |

**Testable ACs**:

- **AC-003.1**: For every entity type, the breadcrumb element is rendered (empty-hidden for
  unparented ideas/bugs/change-cards).
- **AC-003.2**: Each segment is a clickable element with `data-navigate-key="<key>"`; clicking
  calls `navigateToEntity(<key>)` (existing delegation).
- **AC-003.3**: The last segment represents the current entity and is **not** a link (rendered
  without `data-navigate-key`, styled `current`).
- **AC-003.4**: Segment keys use the entity's short display key (E##, E##-F##, E##-F##-###).

#### REQ-F-004 — Feature Status Rollup on Epic Overview

**Trace**: feature.md REQ-F-003, Story 1.

Horizontal row of colored pills, one per distinct feature status in the selected epic,
showing `[● status-name count]`.

**Testable ACs**:

- **AC-004.1**: The section renders only on epic Overview.
- **AC-004.2**: Each pill's background/border color matches `getStatusColor(status)`, the same
  palette used by the existing status badges.
- **AC-004.3**: Counts equal the number of features in the selected epic with that status,
  derived from `treeData` (not a new network call).
- **AC-004.4**: Statuses with zero features are omitted.
- **AC-004.5**: Order of pills follows the workflow phase ordering from
  `workflow-meta` (already cached in `workflowMeta`), falling back to alphabetical when phase
  data is unavailable.

#### REQ-F-005 — Task Status Rollup + segmented progress bar

**Trace**: feature.md REQ-F-004, Story 1/2.

Rendered on Epic Overview (aggregated across all features in the epic) and on Feature
Overview (tasks of the feature only). Row of colored pills + a single 100%-wide 8px
horizontal bar, split into segments proportional to task count per status, ordered by
workflow phase. A `NN%` label is right-aligned.

**Testable ACs**:

- **AC-005.1**: Pills and segments share the same color for the same status.
- **AC-005.2**: On an empty entity (no tasks), the rollup renders a single dim `No tasks`
  placeholder instead of a zero-width bar.
- **AC-005.3**: The `NN%` label equals the weighted-progress percentage
  (`Σ progress_weight × count / total_tasks`), matching the computation already used by the
  existing properties-panel `Progress` field (E27-F08 §1.4).
- **AC-005.4**: All counts are derived from `treeData`; opening the Overview tab issues **no
  new network requests** for rollup data.

#### REQ-F-006 — Work Breakdown on Feature Overview

**Trace**: feature.md REQ-F-012, Story 7.

Three labelled rows — `Agent`, `Human`, `QA Team` — each a mini 60% horizontal bar with the
numeric task count on the right.

**Testable ACs**:

- **AC-006.1**: Counts are derived from the `responsibility` field of each task's status in
  the cached `workflowMeta.levels.task.statuses`. Statuses without an explicit
  `responsibility` contribute to `none` and are **not** shown as a row.
- **AC-006.2**: Bar width = `count / max(1, total_for_all_three_rows) * 100%` (proportional
  within the section, not to the feature total).
- **AC-006.3**: When all three counts are zero, the whole section is hidden.
- **AC-006.4**: The section is only rendered for feature Overview (not epic, not task).

#### REQ-F-007 — Action Items on Feature Overview

**Trace**: feature.md REQ-F-007, Story 2.

A list of tasks in statuses whose `blocks_feature` meta flag is `true`, grouped by status
with an accent-colored left border.

**Testable ACs**:

- **AC-007.1**: The filter uses `workflowMeta.levels.task.statuses[*].blocks_feature === true`.
  Hardcoded status names are forbidden (regression trap — asserted by a grep test against
  the new JS block: no `=== 'ready_for_approval'` style literals).
- **AC-007.2**: Each row shows a status badge, the clickable key (mono,
  `data-navigate-key=<key>`), and the truncated title (CSS `text-overflow: ellipsis`).
- **AC-007.3**: Rows are ordered by status phase, then by `execution_order`, then by key.
- **AC-007.4**: When no task is in a blocking status, the section is omitted (not shown as
  empty).

#### REQ-F-008 — Clickable child-entity tables

**Trace**: feature.md REQ-F-005, REQ-F-006, Stories 1/2.

| Parent    | Table columns                                               |
| --------- | ----------------------------------------------------------- |
| epic      | `KEY` · `TITLE` · `STATUS BADGE` · `PROGRESS BAR (80px)`    |
| feature   | `KEY` · `TITLE` · `STATUS BADGE` · `ORDER`                  |

**Testable ACs**:

- **AC-008.1**: `KEY` cells carry `data-navigate-key=<child-key>` and a `.clickable` class;
  clicking navigates via existing `navigateToEntity`.
- **AC-008.2**: `TITLE` cells are truncated with CSS (no JS slicing) and expose the full
  title via the `title` attribute.
- **AC-008.3**: Tables have no outer border; rows use bottom-border dividers only.
  Hover state: `background: var(--bg-3)`.
- **AC-008.4**: When >10 children, the table body scrolls inside a `max-height: 420px`
  container; the header row is sticky (`position: sticky; top: 0`).
- **AC-008.5**: For features, the progress bar width equals the same percentage shown in the
  properties panel (`progress_pct`); for epics, the features table inherits the same
  per-feature `progress_pct` from `treeData`.

#### REQ-F-009 — Dependency block on Task Overview

**Trace**: feature.md REQ-F-008, Story 3.

Three labelled sub-sections: `Depends On`, `Blocked By`, `Blocks`. Each entry: clickable key
+ truncated title + status badge. Empty groups show `(none)` in dim. `Blocked By` has a
2px red (`#c0392b`) left border accent.

**Testable ACs**:

- **AC-009.1**: Renders only on task Overview.
- **AC-009.2**: Data source is the cached task object's `depends_on`, `blocked_by`,
  `blocks` fields (arrays of keys). No new endpoint.
- **AC-009.3**: Each key entry resolves to a cached entity via `findEntityByKey`;
  unresolved keys render as plain mono text with a dim `(unresolved)` suffix (non-clickable).
- **AC-009.4**: Every resolved key is clickable with `data-navigate-key`.
- **AC-009.5**: `Blocked By` is rendered with `border-left: 2px solid #c0392b` even when the
  group is non-empty.

#### REQ-F-010 — Notes section (all entity types)

**Trace**: feature.md REQ-F-009, Story 4.

A `Notes` section showing the 5 most recent entity notes, with a `Show N more` button that
expands the list inline.

**Testable ACs**:

- **AC-010.1**: The viewer fetches notes from the new endpoint `GET /api/v1/viewer/notes/{key}`
  when the Overview tab renders. The response is cached in memory keyed by entity key for the
  duration of the viewer session; repeat views do not refetch.
- **AC-010.2**: Only the 5 most recent notes (by `created_at DESC`) are rendered by default.
- **AC-010.3**: Each row shows: `timestamp` (formatted via existing `formatDate`, monospace
  dim) · type pill · truncated content (full text exposed via `title` attribute).
- **AC-010.4**: Note-type → color mapping (applied to the type pill):
  `decision=#8e44ad` (purple), `blocker=#c0392b` (red), `comment=var(--fg-dim)` (gray),
  `solution=#27ae60` (green), `implementation=#2980b9` (blue), `question=#17a2b8` (cyan),
  `rejection=#e83e8c` (pink), `reference=var(--fg-dim)`, `future=var(--fg-dim)`,
  `requirement=var(--fg-dim)`, `testing=var(--fg-dim)`.
- **AC-010.5**: When `total > 5`, a `Show N more` button appears; clicking it renders all
  remaining rows inline (no scroll jump) and is replaced by `Show less`.
- **AC-010.6**: When the entity has no notes, the entire section is omitted (not shown as
  empty).
- **AC-010.7**: While the fetch is pending, a single-row skeleton loader (existing
  `.skeleton` CSS) is shown; the Overview's other sections render independently and are not
  blocked by the notes request.

#### REQ-F-011 — Related Docs section (feature + task)

**Trace**: feature.md REQ-F-011, Story 6.

A `Related Docs` section on feature and task Overview (epic may show it only if
documents are already attached — see REQ-NF-003). Fetched from the new endpoint
`GET /api/v1/viewer/related-docs/{key}`.

**Testable ACs**:

- **AC-011.1**: Each row shows a document-icon SVG, truncated path (mono), and an `open`
  button.
- **AC-011.2**: Clicking `open` routes to the existing viewer `Doc View` (`state=4`),
  reusing `openDocumentByPath()` or an equivalent function already wired for the sidebar
  Documents list.
- **AC-011.3**: When the response is empty, the whole section is omitted.
- **AC-011.4**: Fetch is cached per session identically to notes (AC-010.1).
- **AC-011.5**: When the fetch fails (5xx or network), the section renders with a dim
  `Failed to load docs` message but does not break the rest of the Overview.

#### REQ-F-012 — Status left-border accent on the properties panel (Could-Have)

**Trace**: feature.md REQ-F-013, Story 8.

The existing properties-panel container gains a 3px left border whose color matches the
entity's current status.

**Testable ACs**:

- **AC-012.1**: The `.props-grid` element carries an inline
  `border-left: 3px solid <getStatusColor(entity.status)>` style.
- **AC-012.2**: When `entity.status` is empty/unknown, no border is applied (no regression
  on the existing flat look).
- **AC-012.3**: This rule does not interact with the existing grid layout (no column shift;
  `padding-left` is bumped by 3px to compensate).

#### REQ-F-020 — `GET /api/v1/viewer/notes/{key}` endpoint

**Trace**: feature.md REQ-F-020.

A new viewer endpoint returning the notes of any entity (epic, feature, task, bug, change
card) by key.

**Contract**:

```
GET /api/v1/viewer/notes/{key}
200 OK
{
  "entity_type": "feature",
  "entity_key":  "E27-F09",
  "notes": [
    {
      "id":         123,
      "note_type":  "decision",
      "content":    "Chose SSE for push updates",
      "created_by": "human",
      "created_at": "2026-04-13T18:10:00Z"
    },
    ...
  ]
}

404 Not Found  — entity not found (reuses viewer NotFound mapping)
400 Bad Request — malformed key
```

**Testable ACs**:

- **AC-020.1**: Response JSON shape matches the contract above; `notes` is an empty array
  (not `null`) when the entity has no notes.
- **AC-020.2**: Ordering is `created_at DESC` (so client can slice the first 5 without extra
  sorting).
- **AC-020.3**: `metadata` column and `updated_at` are **not** exposed in the response.
  Only the six fields listed are serialised (forward-compat guard).
- **AC-020.4**: The handler accepts any of the key shapes the existing
  `validateAndNormalizeAnyKey` helper accepts (short task key, long task key, slugged, case
  insensitive).
- **AC-020.5**: Unauthenticated access is allowed on `localhost` only — identical CORS/ACL
  behaviour to existing viewer endpoints (no new security surface).

#### REQ-F-021 — `GET /api/v1/viewer/related-docs/{key}` endpoint

**Trace**: feature.md REQ-F-021.

**Contract**:

```
GET /api/v1/viewer/related-docs/{key}
200 OK
{
  "entity_type": "feature",
  "entity_key":  "E27-F09",
  "docs": [
    { "id": 17, "title": "E27-F09 spec", "file_path": "docs/plan/.../spec.md" },
    ...
  ]
}

404 Not Found  — entity not found
400 Bad Request — malformed key
```

**Testable ACs**:

- **AC-021.1**: Empty list returns `{"docs": []}`, not `null`.
- **AC-021.2**: Documents are ordered most-recent-link-first (matches
  `EntityDocumentRepository.ListForEntity` semantics).
- **AC-021.3**: Paths are returned as stored (relative to project root); the client is
  responsible for resolving them.
- **AC-021.4**: Key normalisation, CORS, and auth behaviour match REQ-F-020.

### 1.2 Non-functional Requirements

#### REQ-NF-001 — Overview renders ≤ 300ms from entity selection

**Trace**: feature.md REQ-NF-001.

- **AC-NF-001.1**: Sections that can be computed from `treeData` / `workflowMeta` / the
  entity object render synchronously on the first layout pass (no network gate).
- **AC-NF-001.2**: Notes and Related Docs render asynchronously behind skeleton rows; the
  rest of the Overview does not wait for them.

#### REQ-NF-002 — Accessibility / keyboard navigation

**Trace**: feature.md REQ-NF-010.

- **AC-NF-002.1**: Every clickable entity key (breadcrumb segment, child-table `KEY` cell,
  dependency entry, action-item key) is focusable (`tabindex="0"`), carries
  `role="button"`, and responds to `Enter` / `Space` by invoking the same navigation as a
  click.
- **AC-NF-002.2**: Colour contrasts of new pills and the Work-Breakdown bars meet WCAG AA
  against `var(--bg-2)` — verified by existing `getContrastColor()` helper for text, and by
  test fixtures for the red `#c0392b` accent.

#### REQ-NF-003 — Client-side data reuse; no per-entity full-entity fetch

**Trace**: feature.md §API Requirements, E27-F08 REQ-NF-002.

- **AC-NF-003.1**: Selecting an entity does **not** issue any `shark get`-equivalent
  full-object GET. Rollups and child tables use the already-loaded `treeData`. Only the two
  new small endpoints (notes, related-docs) are called, and each is cached for the session.
- **AC-NF-003.2**: The existing endpoint set from E27-F08 (`/summary`, `/hierarchy`,
  `/history/{key}`, `/file/{key}`, `/recent-activity`, `/workflow-meta`,
  `/folder-files/{path}`, `/features/{key}/tasks`, `PUT /edit/file`) is unchanged.

### 1.3 End-to-end acceptance scenarios

**Scenario 1** — Developer clicks an epic
- **Given** the viewer is loaded and `E27` appears in the sidebar
- **When** the developer clicks `E27`
- **Then** the detail panel renders with `entityViewTab === 'overview'`
- **And** the Feature Status Rollup is visible without any network request for rollup data
- **And** the Features table lists every feature with a clickable key column
- **And** the breadcrumb reads `E27`
- **And** the five tab buttons are `Overview · Spec · History · Dashboard · Files`

**Scenario 2** — Developer drills from epic to feature via the features table
- **Given** the `E27` epic overview is showing
- **When** the developer clicks `E27-F08` in the features table
- **Then** the panel navigates to `E27-F08` with Overview active
- **And** the breadcrumb reads `E27 › E27-F08`
- **And** the tab bar is `Overview · Spec · History · Files` (no Dashboard button)

**Scenario 3** — Developer inspects a blocked task's dependencies
- **Given** a task panel is showing with `blocked_by = ["E27-F09-001"]`
- **When** the Overview tab renders
- **Then** the `Blocked By` group has a red left-border accent
- **And** `E27-F09-001` is clickable and navigates to that task when clicked
- **And** the `Depends On` and `Blocks` groups each render `(none)` if empty

**Scenario 4** — Developer views notes
- **Given** a feature has 8 notes in the `entity_notes` table
- **When** the developer clicks that feature
- **Then** a single `GET /api/v1/viewer/notes/E27-F09` request fires
- **And** 5 most-recent rows render with timestamp · type pill · text
- **And** a `Show 3 more` button appears; clicking it reveals the remaining rows

**Scenario 5** — Developer clicks a breadcrumb segment
- **Given** a task panel showing breadcrumb `E27 › E27-F08 › E27-F08-002`
- **When** the developer clicks `E27-F08`
- **Then** the viewer navigates to feature `E27-F08`
- **And** the sidebar selection updates to match

**Scenario 6** — Regression: E27-F08 Dashboard tab still works for epics
- **Given** an epic panel is showing with the new Overview tab active
- **When** the developer clicks `Dashboard`
- **Then** the E27-F08 epic dashboard renders unchanged (Entity Charts, Status Overview,
  Feature Progress, Recent Activity, Stale Entities — all with their existing content)

### 1.4 Out of scope

See [feature.md §Out of Scope](feature.md). Reiterating the hard exclusions:

1. The E27-F08 **Dashboard** tab (epics only) is preserved unchanged — no content migration
   from Dashboard to Overview beyond the intentional overlap documented in feature.md.
2. No WebSocket / SSE — Overview refreshes only on explicit entity selection.
3. Notes/docs are **read-only** in this feature; creation/editing remains in CLI/spec files.
4. No drag-and-drop reordering of child tables.
5. No Bug / Change-Card Overview content — their panels continue to show the properties grid
   + Spec/History/Files tabs. (Notes section *is* added for bugs/change-cards since
   REQ-F-020 covers them, but no child tables, rollups, or dependencies.)
6. No new per-entity full-object GET. Data gaps discovered in §2.5 audit must be closed
   either by extending `hierarchyData` or by silently omitting the field.

---

## 2. Architecture

### 2.1 Scope of changes — file manifest

| File | Change |
| --- | --- |
| `internal/viewer/assets/viewer.html` | Primary: Overview pane renderers, tab-bar changes, breadcrumb, fetch helpers, new CSS rules |
| `internal/viewer/assets/assets_test.go` | Extend string-presence tests for new markers (`>Overview<`, `Work Breakdown`, `Action Items`, `Depends On`, `Blocked By`) |
| `internal/api/viewer/handler.go` | New handlers `Notes` and `RelatedDocs`; register two new routes in `RegisterRoutes` |
| `internal/api/viewer/service.go` | Extend `ViewerServicer` interface with `Notes(ctx, key)` and `RelatedDocs(ctx, key)` methods |
| `internal/api/viewer/types.go` | No change expected; DTOs live in `internal/services/viewer_service.go` per existing pattern |
| `internal/api/viewer/handler_test.go` | Unit tests for the two new handlers (mock service) |
| `internal/services/viewer_service.go` | Implement `Notes` + `RelatedDocs` service methods; add new DTOs `NotesResponse`, `NoteDTO`, `RelatedDocsResponse`, `RelatedDocDTO`; extend constructor/opt setters to accept an `EntityNoteRepository` interface (new optional dependency). The `entityDocRepo` already exists per E27-F08 but currently only feeds `Hierarchy` — we reuse it here. |
| `internal/services/viewer_service_test.go` | New service tests for `Notes` and `RelatedDocs` using mock note + doc repositories |
| `internal/viewer/server/wire.go` | Wire the `EntityNoteRepository` into `ViewerService` via a new `WithNoteRepo(...)` option, analogous to the existing `WithEntityDocRepo`. |

**No database schema change.** `entity_notes` and `entity_documents`/`documents` tables
already exist and are populated.

### 2.2 Backend design

#### 2.2.1 Viewer service — new methods

Add to `internal/services/viewer_service.go`, following the pattern of
`History(ctx, key)` (line 868) which already resolves entity type + ID:

```go
// --- DTOs ---

type NoteDTO struct {
    ID        int64  `json:"id"`
    NoteType  string `json:"note_type"`
    Content   string `json:"content"`
    CreatedBy string `json:"created_by,omitempty"`
    CreatedAt string `json:"created_at"`
}

type NotesResponse struct {
    EntityType models.EntityType `json:"entity_type"`
    EntityKey  string            `json:"entity_key"`
    Notes      []NoteDTO         `json:"notes"`
}

type RelatedDocDTO struct {
    ID       int64  `json:"id"`
    Title    string `json:"title"`
    FilePath string `json:"file_path"`
}

type RelatedDocsResponse struct {
    EntityType models.EntityType `json:"entity_type"`
    EntityKey  string            `json:"entity_key"`
    Docs       []RelatedDocDTO   `json:"docs"`
}

// --- Repository interfaces owned by the viewer service ---

type ViewerEntityNoteRepository interface {
    GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}
```

The service holds a new optional field `noteRepo ViewerEntityNoteRepository` with a
`WithNoteRepo(r)` setter, mirroring `WithEntityDocRepo` (line 384). When `noteRepo` is nil,
`Notes(ctx, key)` returns `NotesResponse{Notes: []NoteDTO{}}` — matching REQ-F-020 empty-array
behaviour and making the feature degrade gracefully if wiring is incomplete in tests.

Notes method (pattern mirrors `History`):

```go
func (s *ViewerService) Notes(ctx context.Context, key string) (*NotesResponse, error) {
    entityType, err := detectEntityType(key)
    if err != nil {
        return nil, fmt.Errorf("viewer notes: %w", err)
    }
    entityID, err := s.resolveEntityID(ctx, entityType, key)
    if err != nil {
        return nil, fmt.Errorf("viewer notes: %w", err)
    }
    out := &NotesResponse{
        EntityType: entityType,
        EntityKey:  strings.ToUpper(strings.TrimSpace(key)),
        Notes:      []NoteDTO{},
    }
    if s.noteRepo == nil {
        return out, nil
    }
    raw, err := s.noteRepo.GetByEntity(ctx, entityType, entityID)
    if err != nil {
        return nil, fmt.Errorf("viewer notes: failed to load notes for %s %s: %w", entityType, key, err)
    }
    // Order DESC for REQ-F-020 AC-020.2 (repo returns ASC)
    for i := len(raw) - 1; i >= 0; i-- {
        n := raw[i]
        out.Notes = append(out.Notes, NoteDTO{
            ID:        n.ID,
            NoteType:  n.NoteType,
            Content:   n.Content,
            CreatedBy: n.CreatedBy,
            CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
        })
    }
    return out, nil
}
```

RelatedDocs method — use a new narrower interface on the viewer service so we don't have to
re-type `BulkEntityDoc`:

```go
type ViewerEntityDocByEntityRepository interface {
    ListForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error)
}
```

This interface is satisfied by `*entitydoc.EntityDocumentRepository` today (see
`ListForEntity` at `internal/repository/entitydoc/repository.go:130`). Add a second optional
field `docByEntityRepo ViewerEntityDocByEntityRepository` + `WithDocByEntityRepo(r)` setter;
register the same concrete repo via both setters in `wire.go` (it already implements both
interfaces). Implementation mirrors `Notes`.

#### 2.2.2 Handler + routes

Add to `internal/api/viewer/handler.go`:

```go
func (h *ViewerHandler) Notes(w http.ResponseWriter, r *http.Request) {
    raw := r.PathValue("key")
    key, err := validateAndNormalizeAnyKey(raw)
    if err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }
    resp, err := h.svc.Notes(r.Context(), key)
    if err != nil {
        if isNotFound(err) {
            respondError(w, http.StatusNotFound, err.Error())
            return
        }
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
    respondJSON(w, http.StatusOK, resp)
}

// RelatedDocs — structurally identical.
```

In `RegisterRoutes` (line 47), add after the existing routes and before the `OPTIONS`
fallthrough:

```go
mux.Handle("GET "+prefix+"/notes/{key}", wrap(http.HandlerFunc(h.Notes)))
mux.Handle("GET "+prefix+"/related-docs/{key}", wrap(http.HandlerFunc(h.RelatedDocs)))
```

Extend `ViewerServicer` in `internal/api/viewer/service.go` with the two new method
signatures; compile-time checks in `wire.go` enforce the implementation contract (same
pattern already used by `History`, `Summary`, etc.).

#### 2.2.3 Wiring

In `internal/viewer/server/wire.go` (around the existing `WithEntityDocRepo` call at line
300), add:

```go
viewerService.WithNoteRepo(noteRepo) // noteRepo already constructed at line 205
viewerService.WithDocByEntityRepo(entitydoc.NewEntityDocumentRepository(db))
```

`noteRepo` (`*repository.EntityNoteRepository`, the exported alias in
`internal/repository/aliases.go`) already satisfies `ViewerEntityNoteRepository` because its
`GetByEntity` method signature matches the interface.

### 2.3 Frontend design (viewer.html)

All frontend changes are confined to the existing embedded template file. No build step,
no bundler, consistent with E27-F03 and E27-F08.

#### 2.3.1 New `entityViewTab` value and default rule

Current values after E27-F08: `'info' | 'transitions' | 'files' | 'dashboard'`.
Add `'overview'`. The epic-dashboard default rule introduced by E27-F08 in
`navigateToEntity()` becomes:

```js
// BEFORE (E27-F08):
entityViewTab = (navEntity && navEntity.type === 'epic') ? 'dashboard' : 'info';

// AFTER (E27-F09):
entityViewTab = 'overview';
```

This single change (plus removing the branching) delivers AC-001.1 and AC-001.4. The
Dashboard tab is no longer the default for epics; the user opens it explicitly.

#### 2.3.2 Toggle bar changes in `renderEntityView()`

Modify the toggle-bar block (E27-F08 spec §2.2.2 located this around line 2638; with F08
landed the relevant block is roughly at the same place — implementation MUST grep for the
`toggle-bar` template, not a hardcoded line number).

1. **Prepend** an Overview button (always present for epic/feature/task; omit for bug /
   change-card — their panels are simpler and keep the current tab set).
2. **Remove** the `editBtnHtml` from the toggle bar; render the Edit toggle inside the Spec
   pane instead (see §2.3.3).
3. Extend the pane-switch in `renderEntityView()`:

```js
if (entityViewTab === 'overview') {
    renderOverviewPane(entity, paneEl);
} else if (entityViewTab === 'info') {
    renderMarkdownPane(selectedKey, paneEl);            // includes inline Edit toggle
} else if (entityViewTab === 'dashboard' && isEpic) {
    renderEpicDashboardPane(entity.key, paneEl);        // E27-F08
} else if (entityViewTab === 'files') {
    renderFolderFilesPane(filesDirPath, paneEl);
} else {
    // Transitions (unchanged)
}
```

Guard: if `entityViewTab === 'overview'` but the entity type is bug/change-card/idea,
coerce to `'info'` before rendering (AC-001.5).

#### 2.3.3 Inline Edit toggle inside the Spec pane

Move the existing `editBtnHtml` and its click handler from the toggle bar into the header
row of `renderMarkdownPane()`. A single button with two visual states:

```js
// inside renderMarkdownPane(key, paneEl):
const canEdit = !!(entity.file_path || entity.linked_content_path);
const editBtnHtml = canEdit
  ? `<button class="inline-edit-btn" id="ev-inline-edit">${inlineEditMode ? 'View' : '✎ Edit'}</button>`
  : '';
paneEl.innerHTML = `
  <div class="spec-pane-header">${editBtnHtml}</div>
  <div id="spec-body"></div>
`;
// ...existing markdown render / editor swap logic reused unchanged.
```

All existing E27-F07 editor save/cancel wiring stays inside `renderMarkdownPane`.

#### 2.3.4 Breadcrumb

New helper `renderBreadcrumb(entity)` returns an HTML string inserted between the entity
title and the properties panel inside `renderEntityView()`:

```js
function renderBreadcrumb(entity) {
  if (!entity) return '';
  const chain = [];          // [{key, isCurrent}]
  switch (entity.type) {
    case 'task':    chain.push({ key: entity.epic_key || entity.parent_epic_key });
                     chain.push({ key: entity.feature_key || entity.parent });
                     chain.push({ key: entity.key, isCurrent: true });
                     break;
    case 'feature': chain.push({ key: entity.epic_key || entity.parent });
                     chain.push({ key: entity.key, isCurrent: true });
                     break;
    case 'epic':    chain.push({ key: entity.key, isCurrent: true });
                     break;
    default:        return '';  // bug / change-card / idea — no breadcrumb
  }
  return `<div class="breadcrumb">${chain.map(seg => seg.isCurrent
    ? `<span class="breadcrumb-seg current">${escapeHtml(seg.key)}</span>`
    : `<span class="breadcrumb-seg" role="button" tabindex="0" data-navigate-key="${escapeHtml(seg.key)}">${escapeHtml(seg.key)}</span>`
  ).join('<span class="breadcrumb-sep">›</span>')}</div>`;
}
```

`data-navigate-key` delegation is already wired at the pane level by E27-F08 (AC-001.9) and
the pre-existing viewer navigation; no new listener is needed.

Keyboard accessibility (AC-NF-002.1) relies on a small global handler added once:

```js
document.addEventListener('keydown', e => {
  if ((e.key === 'Enter' || e.key === ' ') && e.target.matches('[data-navigate-key][role="button"]')) {
    e.preventDefault();
    const k = e.target.getAttribute('data-navigate-key');
    if (k) navigateToEntity(k);
  }
});
```

#### 2.3.5 `renderOverviewPane(entity, paneEl)` — the central dispatcher

```js
function renderOverviewPane(entity, paneEl) {
  const sections = [];
  switch (entity.type) {
    case 'epic':
      sections.push(renderFeatureRollupSection(entity));
      sections.push(renderTaskRollupSection(collectEpicTasks(entity)));
      sections.push(renderFeaturesTableSection(entity));
      sections.push(notesPlaceholder(entity.key));
      sections.push(relatedDocsPlaceholder(entity.key));
      break;
    case 'feature':
      sections.push(renderTaskRollupSection(entity.tasks || []));
      sections.push(renderWorkBreakdownSection(entity.tasks || []));
      sections.push(renderActionItemsSection(entity.tasks || []));
      sections.push(renderTasksTableSection(entity));
      sections.push(notesPlaceholder(entity.key));
      sections.push(relatedDocsPlaceholder(entity.key));
      break;
    case 'task':
      sections.push(renderDependencyBlockSection(entity));
      sections.push(notesPlaceholder(entity.key));
      sections.push(relatedDocsPlaceholder(entity.key));
      break;
    default:
      paneEl.innerHTML = '';
      return;
  }
  paneEl.innerHTML = sections.filter(Boolean).join('');
  // Fire notes + docs fetches (they render into their placeholder divs).
  loadNotesInto(paneEl.querySelector(`[data-notes-for="${entity.key}"]`), entity.key);
  loadRelatedDocsInto(paneEl.querySelector(`[data-docs-for="${entity.key}"]`), entity.key);
}
```

Each `render*Section` returns an HTML string wrapped in an `.ov-section` block with the
section header pattern defined in [feature.md §Section Header Pattern](feature.md).

#### 2.3.6 Rollup / work-breakdown / action-items computation

All derived from cached `treeData` + `workflowMeta`. A shared helper:

```js
function countByStatus(items) {
  const out = {};
  for (const it of items) {
    const s = (it.status || 'unknown');
    out[s] = (out[s] || 0) + 1;
  }
  return out;
}

function collectEpicTasks(epic) {
  const tasks = [];
  for (const f of (epic.features || [])) {
    for (const t of (f.tasks || [])) tasks.push(t);
  }
  return tasks;
}

function statusMeta(status) {
  const levelMeta = (workflowMeta && workflowMeta.levels && workflowMeta.levels.task) || {};
  return (levelMeta.statuses || []).find(s => s.name === status) || {};
}

function weightedProgress(countsByStatus, total) {
  if (!total) return 0;
  let acc = 0;
  for (const [s, n] of Object.entries(countsByStatus)) {
    const w = statusMeta(s).progress_weight || 0;
    acc += (w * n);
  }
  return Math.round(acc / total);
}

function workBreakdown(tasks) {
  const buckets = { agent: 0, human: 0, qa_team: 0 };
  for (const t of tasks) {
    const r = statusMeta(t.status).responsibility || 'none';
    if (buckets[r] != null) buckets[r]++;
  }
  return buckets;
}

function actionItems(tasks) {
  return tasks
    .filter(t => statusMeta(t.status).blocks_feature === true)
    .sort((a, b) => {
      const pa = statusMeta(a.status).phase || '';
      const pb = statusMeta(b.status).phase || '';
      if (pa !== pb) return pa.localeCompare(pb);
      return (a.execution_order || 0) - (b.execution_order || 0);
    });
}
```

AC-007.1 is satisfied because `actionItems` never references literal status names.

#### 2.3.7 Dependency block

```js
function renderDependencyBlockSection(task) {
  const resolve = keys => (keys || []).map(k => ({ key: k, entity: findEntityByKey(k) }));
  const groups = [
    { label: 'Depends On', items: resolve(task.depends_on) },
    { label: 'Blocked By', items: resolve(task.blocked_by), accent: true },
    { label: 'Blocks',     items: resolve(task.blocks) },
  ];
  const groupHtml = g => `
    <div class="dep-group${g.accent ? ' dep-group-accent' : ''}">
      <div class="dep-label">${escapeHtml(g.label)}</div>
      ${g.items.length
        ? g.items.map(renderDepRow).join('')
        : `<div class="dep-empty">(none)</div>`}
    </div>`;
  return sectionShell('Dependencies', groups.map(groupHtml).join(''));
}
```

`renderDepRow` produces `<key>` with `data-navigate-key` when `entity` resolves,
`<key><span class="dim">(unresolved)</span>` otherwise.

#### 2.3.8 Notes and Related Docs fetch pipeline

```js
const notesCache = new Map();        // key → promise of NotesResponse
const relatedDocsCache = new Map();  // key → promise of RelatedDocsResponse

async function fetchNotes(key) {
  if (!notesCache.has(key)) {
    notesCache.set(key, fetch(`/api/v1/viewer/notes/${encodeURIComponent(key)}`)
      .then(r => r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`))));
  }
  return notesCache.get(key);
}
async function fetchRelatedDocs(key) {
  if (!relatedDocsCache.has(key)) {
    relatedDocsCache.set(key, fetch(`/api/v1/viewer/related-docs/${encodeURIComponent(key)}`)
      .then(r => r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`))));
  }
  return relatedDocsCache.get(key);
}
```

The render functions `loadNotesInto(el, key)` and `loadRelatedDocsInto(el, key)` write a
skeleton row first, then `await fetchNotes/...` and replace the content. On rejection, they
write an empty section (notes) or a dim `Failed to load docs` message (docs), satisfying
AC-011.5 / AC-010.6.

**Invalidation**: the only consumer is the Overview pane; cache lives for the session (SPA
reload drops it). This matches REQ-NF-003 and avoids double-fetch on flick between entities.

### 2.4 CSS additions

Appended to the existing `<style>` block. No existing rules are modified.

```css
/* --- Breadcrumb --- */
.breadcrumb { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--fg-dim); margin: 2px 0 10px; }
.breadcrumb-seg { padding: 2px 4px; border-radius: 3px; cursor: pointer; font-family: var(--font-mono, monospace); }
.breadcrumb-seg:hover { background: var(--bg-3); color: var(--fg); }
.breadcrumb-seg.current { color: var(--fg); cursor: default; }
.breadcrumb-sep { color: var(--fg-dim); user-select: none; }

/* --- Overview sections --- */
.ov-section { margin: 18px 0; }
.ov-section-header { display: flex; align-items: center; gap: 10px; font-size: 10px; font-weight: 600; letter-spacing: 0.12em; text-transform: uppercase; color: var(--fg-dim); margin: 0 0 10px; }
.ov-section-header::after { content: ''; flex: 1; height: 1px; background: var(--border); }

/* --- Rollup pills --- */
.rollup-row { display: flex; flex-wrap: wrap; gap: 6px; }
.rollup-pill { display: inline-flex; align-items: center; gap: 4px; border-radius: 12px; padding: 2px 8px; font-size: 11px; }

/* --- Segmented progress bar --- */
.seg-bar { display: flex; height: 8px; border-radius: 4px; overflow: hidden; background: var(--bg-3); }
.seg-bar-segment { height: 100%; }
.seg-bar-row { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
.seg-bar-label { font-size: 11px; color: var(--fg-dim); font-variant-numeric: tabular-nums; }

/* --- Work breakdown --- */
.wb-row { display: flex; align-items: center; gap: 8px; margin: 4px 0; font-size: 12px; }
.wb-label { width: 80px; color: var(--fg-dim); }
.wb-bar { flex: 1; height: 6px; border-radius: 3px; background: var(--bg-3); overflow: hidden; }
.wb-bar-fill { height: 100%; background: var(--accent, #4a9eff); }
.wb-count { width: 40px; text-align: right; font-variant-numeric: tabular-nums; }

/* --- Action items --- */
.action-item-row { display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-radius: 0 4px 4px 0; margin-bottom: 4px; }

/* --- Child entity tables --- */
.ov-table { width: 100%; border-collapse: collapse; }
.ov-table thead th { position: sticky; top: 0; background: var(--bg-2); text-align: left; font-size: 10px; font-weight: 600; letter-spacing: 0.08em; text-transform: uppercase; color: var(--fg-dim); padding: 6px 8px; border-bottom: 1px solid var(--border); }
.ov-table tbody td { padding: 6px 8px; border-bottom: 1px solid var(--border); font-size: 13px; }
.ov-table tbody tr:hover { background: var(--bg-3); }
.ov-table .clickable { cursor: pointer; color: var(--accent); font-family: var(--font-mono, monospace); }
.ov-table-scroll { max-height: 420px; overflow-y: auto; }

/* --- Dependencies --- */
.dep-group { margin-bottom: 12px; padding-left: 8px; }
.dep-group-accent { border-left: 2px solid #c0392b; }
.dep-label { font-size: 11px; font-weight: 600; color: var(--fg-dim); margin-bottom: 4px; }
.dep-empty { font-size: 12px; color: var(--fg-dim); font-style: italic; }

/* --- Notes --- */
.note-row { display: flex; align-items: flex-start; gap: 8px; padding: 4px 0; font-size: 12px; }
.note-ts { font-family: var(--font-mono, monospace); color: var(--fg-dim); white-space: nowrap; }
.note-type-pill { font-size: 10px; padding: 1px 6px; border-radius: 8px; color: #fff; }
.note-text { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.notes-show-more { margin-top: 6px; font-size: 12px; background: none; border: 0; color: var(--accent); cursor: pointer; padding: 2px 0; }

/* --- Related docs --- */
.reldoc-row { display: flex; align-items: center; gap: 8px; padding: 4px 0; font-size: 12px; }
.reldoc-path { flex: 1; font-family: var(--font-mono, monospace); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.reldoc-open { font-size: 11px; padding: 2px 8px; border-radius: 3px; border: 1px solid var(--border); background: var(--bg-2); cursor: pointer; }

/* --- Inline Spec-tab Edit button --- */
.spec-pane-header { display: flex; justify-content: flex-end; padding: 2px 0 6px; }
.inline-edit-btn { font-size: 12px; background: none; border: 1px solid var(--border); border-radius: 3px; padding: 2px 8px; cursor: pointer; color: var(--fg); }
.inline-edit-btn:hover { background: var(--bg-3); }

/* --- Properties-panel status accent (REQ-F-012) --- */
.props-grid[data-status-accent] { padding-left: calc(var(--pad, 10px) + 3px); }
```

### 2.5 Data-availability audit (mandatory pre-task step)

Before task generation, the feature requires that the following fields on
`hierarchyData.treeData` objects be present. Fields currently exposed by the
`ViewerService.Hierarchy` response (per E27-F08 spec §1.4) — this audit must be re-verified
as the first coding task:

| Field | Used by | Currently on hierarchy payload? | Action if missing |
| --- | --- | --- | --- |
| `task.depends_on`, `task.blocked_by`, `task.blocks` | REQ-F-009 | To verify. | Extend `HierarchyResponse` task projection. |
| `task.execution_order` | REQ-F-008 (tasks table) | Verified in E27-F08. | — |
| `feature.progress_pct` | REQ-F-008 (features table), REQ-F-005 | Verified in E27-F08. | — |
| `task.status`, `feature.status`, `epic.status` | all rollups | Verified. | — |
| `epic.features[].tasks[]` | `collectEpicTasks` | Verified (hierarchy nests them). | — |
| `workflowMeta.levels.task.statuses[*].{phase, progress_weight, responsibility, blocks_feature, color}` | REQ-F-005/006/007 | Verified (E27 workflow-meta endpoint). | — |

If the `depends_on`/`blocked_by`/`blocks` audit fails, the first task in the task-generation
pass is a backend extension of `HierarchyResponse`'s task DTO — not a new per-entity GET
(AC-NF-003.1).

### 2.6 Key technical decisions

| Decision | Rationale |
| --- | --- |
| Add **Overview** as the default tab for all entity types; keep the E27-F08 **Dashboard** tab for epics. | Preserves the analytical charts that have no equivalent in Overview (REQ-F-001 + Out-of-Scope item 1). Two-click Dashboard access is acceptable; Overview is the common path. |
| Add two small viewer endpoints (`notes`, `related-docs`) rather than stuffing notes into `hierarchyData`. | `hierarchyData` is already a large payload loaded on every SPA boot; adding 10× notes × entities would inflate it multiplicatively. Lazy, cached per-entity fetches keep boot fast. |
| Client-side session cache (two `Map`s) for notes/docs. | Zero infra cost; invalidates on SPA reload; matches viewer's read-only nature. No need for ETag or stale-while-revalidate. |
| Rollup, breakdown, action-items computed **client-side** from `treeData` + `workflowMeta`. | REQ-NF-003: no new `shark get`-style fetch. `treeData` is canonical. All filters/counts are O(n) over a small set. |
| Move Edit to an inline toggle inside the Spec pane. | Reclaims tab-bar space for Overview, avoids an awkward 5- or 6-button toggle bar, and matches the pattern readers expect ("Edit this page" lives on the page). |
| Breadcrumb rendered via parent-chain lookup, not a dedicated API. | `findEntityByKey` + `hierarchyData` already knows the parent chain; no backend work. |
| `Blocked By` group always has a red accent — even when empty — for visual consistency. | feature.md calls for a "red left border accent to visually signal risk"; applying it always removes layout-shift when items appear/disappear. |
| Service `Notes` orders DESC in Go, even though the repo returns ASC. | Places the business ordering rule in the service layer per service-design guide; repo stays pure-CRUD; REQ-F-020 AC-020.2 is satisfied server-side so even non-JS clients (curl) see DESC. |
| `ViewerEntityNoteRepository` is a narrow interface defined inside `internal/services/viewer_service.go`. | Matches the existing "consumer-defined interface" pattern (§1.3 service-design guide) already used for `ViewerEpicRepository` et al. No change to `repository.EntityNoteRepository` itself. |
| Reuse `*entitydoc.EntityDocumentRepository` for RelatedDocs. | The repo already has `ListForEntity(ctx, entityType, entityID)`; no new query or schema. Single wiring line added in `wire.go`. |

### 2.7 Integration with existing code (function + line references)

Line numbers against current main (`internal/viewer/assets/viewer.html` = 3925 lines).
Implementation tasks MUST resolve the actual target via function-name grep, not the line
number — they shift as E27-F08 iterations land.

| Change | Location | Specific element |
| --- | --- | --- |
| Default tab = `'overview'` | `viewer.html` `navigateToEntity()` | replace assignment introduced by E27-F08 |
| Guard `entityViewTab` when type-mismatched | `viewer.html` `renderEntityView()` | early coercion before pane switch |
| New tab button `Overview` + wiring | `viewer.html` `renderEntityView()` toggle-bar template | prepend button + `addEventListener` |
| Remove Edit from toggle bar | same location | delete `editBtnHtml` concat |
| Inline Edit in Spec | `viewer.html` `renderMarkdownPane()` | new header row + button handler (keep existing E27-F07 logic) |
| Breadcrumb | `viewer.html` `renderEntityView()` | call `renderBreadcrumb(entity)` after title, before properties grid |
| Properties-panel status accent | `viewer.html` `renderPropertiesPanel()` | inline style + `data-status-accent` |
| `renderOverviewPane` dispatcher | `viewer.html` new function, inserted after `renderEntityView()` | — |
| Helper fns (countByStatus, statusMeta, weightedProgress, workBreakdown, actionItems, collectEpicTasks) | `viewer.html` new block, kept near top of JS (after `findEntityByKey`) | — |
| Fetch + cache helpers (`fetchNotes`, `fetchRelatedDocs`, `loadNotesInto`, `loadRelatedDocsInto`) | `viewer.html` new block above `renderOverviewPane` | — |
| New CSS | `<style>` block at top of `viewer.html` | append §2.4 rules |
| `ViewerService.Notes` + `RelatedDocs` | `internal/services/viewer_service.go` | new methods; new DTOs; new optional repos + `With*` setters |
| Compile-time interface check in handler | `internal/api/viewer/service.go` — `ViewerServicer` | add two new methods |
| Route registration | `internal/api/viewer/handler.go` `RegisterRoutes` | add two `mux.Handle("GET …/notes/{key}" …)` / related-docs lines |
| Handler implementations | `internal/api/viewer/handler.go` | `Notes`, `RelatedDocs` funcs |
| Handler unit tests | `internal/api/viewer/handler_test.go` | mock service, assert status codes + JSON shape |
| Service unit tests | `internal/services/viewer_service_test.go` | mock `noteRepo` / `docByEntityRepo`; cover DESC ordering, empty list, missing repo, error propagation |
| Wire-up | `internal/viewer/server/wire.go` | two lines next to existing `WithEntityDocRepo` |
| Asset string-presence test | `internal/viewer/assets/assets_test.go` | add markers: `>Overview<`, `Work Breakdown`, `Action Items`, `Depends On`, `Blocked By`, `breadcrumb-seg` |

### 2.8 Risks & mitigations

| Risk | Mitigation |
| --- | --- |
| Audit reveals `task.depends_on` etc. are not on `treeData`. | First task in Phase 1 is an audit + conditional backend extension (see §2.5). Panel silently omits missing fields meanwhile. |
| E27-F08 Dashboard tab behaviour regresses because `entityViewTab` default changes. | Explicit regression gate (Scenario 6). Asset string-presence test already retains `>Dashboard<` marker; Dashboard tab click handler is unchanged. |
| Notes fetch races with rapid tree clicking (stale response overwrites newer selection). | `loadNotesInto(el, key)` captures the DOM node closure — when the user navigates away, the old `el` is detached and the `.then` writes into an orphaned node. Harmless, but a `if (!el.isConnected) return;` guard is added defensively. |
| Keyboard users can't navigate breadcrumb / key cells. | Global `keydown` handler added once (§2.3.4); covered by AC-NF-002.1. |
| Large epic (100+ tasks) recomputes rollup on every tab switch. | Rollup is O(tasks_in_epic) per render; memoise by `entity.key` if profiling flags it. Not a pre-opt requirement. |
| `fetchNotes` cache never invalidates in long-running session. | Acceptable for read-only viewer. Documented in REQ-F-010 AC-010.1. A follow-up enhancement (not this feature) could add SSE invalidation. |
| Key normalisation mismatch between URL (`encodeURIComponent`) and server (`validateAndNormalizeAnyKey`). | Server already handles upper/lower + short/long; `encodeURIComponent` covers slugged characters. Handler test case added for slugged keys. |

---

## 3. Exit gate

- [x] Every functional and non-functional requirement has ≥1 testable AC.
- [x] Every AC is verifiable against DOM state, `entityViewTab`, `fetch` network panel, a
      JSON response shape, or a unit test.
- [x] Every architecture decision references an existing pattern (E27-F08 spec, viewer
      service layer, entitydoc repo, workflow-meta endpoint) or explains the deviation.
- [x] File paths enumerated for every change (§2.7).
- [x] API contracts fully specified (§1.1 REQ-F-020 / REQ-F-021 incl. empty-array, order,
      error codes).
- [x] Data-availability audit enumerated with a concrete fallback (§2.5).
- [x] Regression gate on E27-F08 Dashboard behaviour explicit (Scenario 6 + AC-001.2).
- [x] No TBDs in critical sections.

---

*Last updated*: 2026-04-13
