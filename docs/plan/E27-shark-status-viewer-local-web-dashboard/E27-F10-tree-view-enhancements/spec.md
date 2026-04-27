---
feature_key: E27-F10-tree-view-enhancements
epic_key: E27
title: "Spec: Tree View Enhancements — Accessibility, Filter Consistency, Persistence"
type: combined-spec
---

# Spec: Tree View Enhancements

**Feature Key**: E27-F10
**Parent Epic**: [E27 PRD](../epic.md) · [E27 Architecture](../architecture.md)
**Feature description**: [feature.md](feature.md)
**Sibling spec convention reference**: [E27-F09 spec](../E27-F09-entity-detail-panel-enrichment-overview-tab-rollup/spec.md)

> This spec is **incremental** on top of the viewer delivered in E27-F01…F09. It covers five
> small, focused improvements to the sidebar tree in `internal/viewer/assets/viewer.html`. All
> changes are contained in that single file (CSS + JS). No Go code, no new endpoints, no DB
> changes, no new dependencies.
>
> See epic PRD §"Architecture Decision" and §"Single-File SPA" for the constraint that the
> viewer is a vanilla-JS, single-file SPA with no build step.

---

## 1. Requirements

All requirements trace to the five tasks in this feature
(`tasks/T-E27-F10-001…005.md`) and the user-reported observations captured there.

### 1.1 Functional Requirements

#### REQ-F-001 — Completed entities are visually de-emphasised without relying on color

**Trace**: T-E27-F10-001 (colorblind accessibility).

Today, "completed" status is signalled almost exclusively by a small green status dot
(`viewer.html:1584` palette). For colorblind users this is unreliable. The tree must add a
non-color affordance — dim the title and key text — so the status is readable without
distinguishing green from grey.

A `.entity-completed` CSS class (analogous to the existing `.entity-cancelled` at
`viewer.html:310-322`) is applied to any tree node, sidebar flat-section row, and
content-area table row whose status is `completed` (per the entity-specific terminal
mapping in REQ-F-005). The class dims `.tree-node-key` and `.tree-node-title` opacity to
~0.55 (lighter than the 0.4 used for cancelled, so they remain distinguishable from each
other) and **does not apply line-through** (line-through is reserved for cancelled, per
the open question in T-E27-F10-001).

**Testable ACs**:

- **AC-001.1**: A node with `status === 'completed'` (or any other entity-specific
  "completed-equivalent" status from REQ-F-005) renders with class
  `entity-completed` on the outer `<div class="tree-node …">`.
- **AC-001.2**: The CSS rule for `.entity-completed .tree-node-key,
  .entity-completed .tree-node-title` sets `opacity: 0.55` and `text-decoration: none`.
- **AC-001.3**: A cancelled node (status in `CANCELLED_STATUSES`) continues to render
  with class `entity-cancelled` and `text-decoration: line-through` (regression gate
  on the existing rule at `viewer.html:310-322`).
- **AC-001.4**: An entity that is BOTH completed and cancelled (impossible in practice
  but defensively): cancelled wins (`isCancelledStatus()` is checked first in the
  class assembly).
- **AC-001.5**: The `.entity-completed` style is also applied to `<tr>` rows in
  the content-area tables that today receive `.entity-cancelled` (i.e. the
  feature-list and task-list tables; see `viewer.html:4132, 4246`).

#### REQ-F-002 — Single "Show all items" toggle controls both completed and cancelled visibility

**Trace**: T-E27-F10-002.

The sidebar checkbox previously labelled **"Show completed"** at `viewer.html:4823`
governs only the `status === 'completed'` predicate (`viewer.html:4720, 4762, 4790,
4837`). Cancelled items are currently always shown. This is inconsistent — the user
expects a single "show everything terminal" toggle.

The label is renamed to **"Show all items"** and the predicate is broadened so that when
the checkbox is **off**, both completed and cancelled (and all other terminal statuses
defined in REQ-F-005) are hidden. When **on**, all entities are shown.

**Testable ACs**:

- **AC-002.1**: The `<label>` text is "Show all items" (not "Show completed").
- **AC-002.2**: With checkbox **off**, no node/row whose status matches the
  entity-specific terminal-status list (REQ-F-005) is rendered in any of: epic
  hierarchy children, feature hierarchy children, task hierarchy children, or any
  flat section (Bugs, Change Cards, Ideas).
- **AC-002.3**: With checkbox **on**, every entity is rendered regardless of status
  (subject only to the existing tag filter, which is independent).
- **AC-002.4**: The JS variable name MAY remain `showCompleted` internally to minimise
  diff churn, but a code comment documents that it now means "show all (incl.
  terminal)". Renaming to `showAll` is acceptable but not required.
- **AC-002.5**: No regression to the second checkbox "Show all files" at
  `viewer.html:4827` — it remains a separate, independent control.

#### REQ-F-003 — "Collapse all" button collapses every expanded node

**Trace**: T-E27-F10-003.

A new **"Collapse all"** button is added to the sidebar filter bar
(`viewer.html:4820-4829`). Clicking it clears `expandedEpics` and `expandedFeatures`
(`viewer.html:2242-2243`) and re-renders the sidebar, returning the tree to a
fully-collapsed state. "Expand all" is **out of scope** for this feature.

**Testable ACs**:

- **AC-003.1**: A `<button id="collapse-all-btn">Collapse all</button>` is rendered
  inside the `.sidebar-filter-bar`.
- **AC-003.2**: Clicking the button calls a handler that executes
  `expandedEpics.clear(); expandedFeatures.clear();` and then `renderSidebar()`.
- **AC-003.3**: After the click, every previously-expanded epic and feature is
  collapsed (no `▼` arrows visible, no child nodes rendered).
- **AC-003.4**: The button does not change the `showCompleted` / `showAllFiles` state.
- **AC-003.5**: Clicking the button when nothing is expanded is a no-op (no error).

#### REQ-F-004 — "Show all items" state persists across reloads via `localStorage`

**Trace**: T-E27-F10-004.

The sidebar filter checkbox state must survive page reloads. We use `localStorage`
(not cookies) because (a) no other shark viewer state currently uses cookies — there is
no existing precedent to be consistent with, (b) `localStorage` is simpler (no
serialization/expiry), and (c) it stays scoped per-origin which matches the
`localhost:7777` origin model of `shark web`.

The persisted key is `shark.viewer.showAllItems` and the persisted value is the string
`"true"` or `"false"`. On page load, the initial value of the JS variable
(`showCompleted`) is read from `localStorage`; if the key is unset, the default remains
`false` (matching today's behavior).

**Testable ACs**:

- **AC-004.1**: When the checkbox is toggled, `localStorage.setItem('shark.viewer.showAllItems', String(checked))`
  is called.
- **AC-004.2**: On page load (before `renderSidebar()` runs), `showCompleted` is
  initialised from `localStorage.getItem('shark.viewer.showAllItems') === 'true'`.
- **AC-004.3**: With no value stored, `showCompleted` defaults to `false` (current
  behavior).
- **AC-004.4**: A stored value of `"true"` causes the checkbox to render checked AND
  causes terminal-status entities to be rendered on first paint.
- **AC-004.5**: A malformed/legacy stored value (anything other than `"true"`) is
  treated as `false` — no error is thrown, no exception bubbles.
- **AC-004.6**: Reads/writes are wrapped in `try/catch` so that a `SecurityError` from
  a privacy-locked browser does not break sidebar rendering. On read failure, fall back
  to `false`. On write failure, silently swallow (the in-memory state still works for
  the current session).

#### REQ-F-005 — Per-entity-type terminal-status filter (Ideas fix)

**Trace**: T-E27-F10-005 (ideas show regardless of show-all flag).

Today the filter predicate at `viewer.html:4790` is hardcoded to
`i.status !== 'completed'`. Ideas use `converted` and `archived` as their terminal
statuses (see `internal/models/idea.go:13-16`), so no idea ever matches
`status === 'completed'` and **all ideas pass through regardless of the toggle**. The
same hardcoded check exists at `viewer.html:4720, 4762, 4837` (for tasks, features,
epics) — those happen to be correct for those entity types but only by coincidence.

We introduce a single helper:

```js
// Returns true if the status is a "completed-equivalent" terminal status for that
// entity type. Used by the show-all-items filter.
function isHiddenTerminalStatus(status, entityType) { … }
```

The mapping is:

| Entity type   | Statuses hidden when "Show all items" is OFF                           |
| ------------- | ---------------------------------------------------------------------- |
| `epic`        | `completed`, `cancelled`, `closed`, `wont_fix`, `rejected`             |
| `feature`     | `completed`, `cancelled`, `closed`, `wont_fix`, `rejected`             |
| `task`        | `completed`, `done`, `cancelled`, `closed`, `wont_fix`, `rejected`     |
| `bug`         | `completed`, `done`, `verified`, `closed`, `wont_fix`, `rejected`      |
| `change_card` | `completed`, `done`, `approved`, `rejected`, `closed`                  |
| `idea`        | `converted`, `archived`                                                |

This mapping is the union of (a) the existing `TERMINAL_STATUSES` set
(`viewer.html:1659-1662`), (b) the existing `CANCELLED_STATUSES` set
(`viewer.html:1668-1670`), and (c) idea-specific statuses from
`internal/models/idea.go`. The helper falls back to `TERMINAL_STATUSES ∪
CANCELLED_STATUSES` if the entity type is unknown — defensive default that hides
terminal items rather than leaking them.

**Testable ACs**:

- **AC-005.1**: With "Show all items" **off**, ideas with `status === 'converted'` are
  hidden from the Ideas flat section (regression test for the reported bug).
- **AC-005.2**: With "Show all items" **off**, ideas with `status === 'archived'` are
  hidden.
- **AC-005.3**: With "Show all items" **off**, ideas with `status === 'new'` or
  `'on_hold'` remain visible.
- **AC-005.4**: With "Show all items" **on**, all ideas are visible regardless of
  status.
- **AC-005.5**: For tasks, with "Show all items" **off**, tasks with `status === 'done'`
  are now hidden (previously they were visible — minor scope expansion, consistent with
  REQ-F-002 intent).
- **AC-005.6**: For bugs, with "Show all items" **off**, bugs with `status ===
  'resolved'`/`'verified'`/`'closed'`/`'wont_fix'`/`'rejected'` are hidden.
- **AC-005.7**: For each call site that currently uses `status !== 'completed'`
  (`viewer.html:4720, 4762, 4790, 4837`), the predicate is replaced with a call to
  `isHiddenTerminalStatus(status, entityType)`.

### 1.2 Non-Functional Requirements

#### REQ-NF-001 — No measurable performance regression on tree render

- **Description**: Adding the helper, CSS class, button, and `localStorage` read must
  not measurably slow tree rendering.
- **Target**: `renderSidebar()` execution time on a 200-entity hierarchy stays within
  ±5% of pre-change measurement (manual profiling via DevTools Performance tab).
- **Justification**: Epic E27 success criteria require hierarchy load < 500 ms for
  500-task projects.

#### REQ-NF-002 — Accessibility: dimming as a non-color signal

- **Description**: REQ-F-001 introduces opacity-based dimming as the primary
  "completed" signal so the tree is usable by colorblind users.
- **Standard**: WCAG 2.1 Success Criterion 1.4.1 ("Use of Color") — color is not the
  only means of conveying information.
- **Testing**: Manual check by the colorblind user who reported the original issue
  (per T-E27-F10-001 acceptance: "Verified by user (colorblind) on actual project
  data"). Automated check: assert that `.entity-completed` rule includes a non-color
  property (`opacity`).

#### REQ-NF-003 — Resilience to `localStorage` unavailability

- **Description**: Per REQ-F-004 AC-004.6, all `localStorage` access is wrapped in
  `try/catch`. A `SecurityError` (privacy-mode, third-party context, or storage quota)
  must not break the viewer.
- **Measurement**: Manual verification with browser privacy mode enabled — the page
  still renders and the in-memory toggle still works for the current session.

### 1.3 Acceptance Criteria (Feature-Level Given/When/Then)

**Scenario 1 — Colorblind affordance**

- **Given** an epic with `status === 'completed'` and a feature with `status === 'completed'`
- **When** the user opens the viewer
- **Then** both nodes render with class `entity-completed`
- **And** the node text appears at ~55% opacity (visibly dimmer than active siblings)
- **And** the node text is **not** struck through (only cancelled nodes are struck)

**Scenario 2 — Single show-all toggle**

- **Given** the project contains: 1 active epic, 1 completed epic, 1 cancelled epic
- **And** the "Show all items" checkbox is **off**
- **When** the sidebar renders
- **Then** only the active epic appears
- **When** the user clicks the checkbox
- **Then** all three epics appear

**Scenario 3 — Collapse all**

- **Given** the user has expanded 3 epics and 5 features
- **When** the user clicks "Collapse all"
- **Then** every epic and feature collapses
- **And** the page does not scroll-jump (the click handler does not change scroll
  position)

**Scenario 4 — Persistence across reload**

- **Given** the user toggles "Show all items" **on**
- **When** the user reloads the page
- **Then** the checkbox renders checked
- **And** completed/cancelled entities appear without the user re-toggling

**Scenario 5 — Ideas filter fix**

- **Given** an idea with `status === 'converted'`
- **And** the "Show all items" checkbox is **off**
- **When** the sidebar renders
- **Then** the converted idea is hidden
- **When** the user toggles the checkbox **on**
- **Then** the converted idea appears

### 1.4 Out of Scope

- **"Expand all" button** — explicitly deferred per T-E27-F10-003 ("Out of scope unless
  trivial"). Users can click individual epic arrows.
- **Granular filtering by status family** (e.g. "show cancelled but not completed") —
  the user model is binary: hide-terminal or show-everything.
- **Per-entity-type "show all" toggles** — one global toggle is enough for the user
  story in this feature.
- **Cookie-based persistence** — `localStorage` is used instead, per REQ-F-004
  rationale.
- **Server-side filtering** — `GET /api/v1/viewer/hierarchy` continues to return all
  entities; filtering remains client-side. Adding `?status=` to that endpoint is a
  future refactor, not part of this feature.
- **Migration of any existing `localStorage` keys** — there are no existing keys to
  migrate.
- **Backend / Go code changes** — none. All changes are in `viewer.html`.

---

## 2. Architecture

### 2.1 Component Changes

| File                                            | Change           | Scope                                                                                                     |
| ----------------------------------------------- | ---------------- | --------------------------------------------------------------------------------------------------------- |
| `internal/viewer/assets/viewer.html` (CSS)      | MODIFY           | Add `.entity-completed` rule next to existing `.entity-cancelled` block at lines 310-322. Add a `.sidebar-filter-bar button` rule for the new "Collapse all" button. |
| `internal/viewer/assets/viewer.html` (JS state) | MODIFY           | Add comment on `let showCompleted` (line 2239) clarifying new "show all items" semantics. No rename required. |
| `internal/viewer/assets/viewer.html` (JS init)  | MODIFY           | Read `localStorage` for `shark.viewer.showAllItems` immediately before `treeData`/state declarations (around line 2240) so the value is set before first `renderSidebar()`. |
| `internal/viewer/assets/viewer.html` (JS helper)| ADD              | New function `isHiddenTerminalStatus(status, entityType)` near the existing `isTerminalStatus` / `isCancelledStatus` helpers (around lines 1659-1674). |
| `internal/viewer/assets/viewer.html` (JS render)| MODIFY           | Replace four `status !== 'completed'` predicates (lines 4720, 4762, 4790, 4837) with `!isHiddenTerminalStatus(status, entityType)`. Pass entity type literally at each call site. |
| `internal/viewer/assets/viewer.html` (sidebar)  | MODIFY           | In `renderSidebar()` (line 4813): rename label text "Show completed" → "Show all items"; add `<button id="collapse-all-btn">`. |
| `internal/viewer/assets/viewer.html` (wiring)   | MODIFY           | In the post-render wiring block (line 4860+): add change handler that writes `localStorage`; add click handler for `#collapse-all-btn` that clears both `Set`s and re-renders. |
| `internal/viewer/assets/viewer.html` (CSS for completed in tables) | MODIFY | Extend the `tr.entity-cancelled td` rule (line 319) with a sibling `tr.entity-completed td` rule (no line-through, opacity 0.55). |
| `internal/viewer/assets/viewer.html` (table rows) | MODIFY         | At lines 4132, 4246, where `entity-cancelled` is conditionally added to `<tr>`, also conditionally add `entity-completed` for completed rows. |

**No new files. No deleted files.** Total expected diff: ~80–110 lines added/modified
in a single file.

### 2.2 Data Model Changes

**None.** No DB schema changes, no new migrations, no API response shape changes. All
filtering remains client-side on the existing `/api/v1/viewer/hierarchy` response.

### 2.3 API / Interface Contracts

**No new endpoints. No changes to existing endpoints.** The hierarchy response shape
stays as defined in E27 epic API Contract.

**Internal JS function contract (new):**

```js
/**
 * Returns true if the given status should be hidden from the tree when the
 * "Show all items" checkbox is OFF, for the given entity type.
 * Falls back to the union of TERMINAL_STATUSES and CANCELLED_STATUSES if
 * entityType is unknown.
 *
 * @param {string} status      e.g. 'completed', 'converted', 'cancelled'
 * @param {string} entityType  one of 'epic'|'feature'|'task'|'bug'|'change_card'|'idea'
 * @returns {boolean}
 */
function isHiddenTerminalStatus(status, entityType) { ... }
```

**Internal `localStorage` contract (new):**

| Key                            | Value (string)                  | Default  | Read on  | Written on                    |
| ------------------------------ | ------------------------------- | -------- | -------- | ----------------------------- |
| `shark.viewer.showAllItems`    | `"true"` or `"false"`           | unset    | page load | every checkbox change         |

### 2.4 Key Technical Decisions and Rationale

#### Decision 1 — Use `localStorage` over cookies

**Rationale**:

- No existing shark viewer state uses cookies (`grep localStorage|cookie viewer.html`
  returned only CSS `border-collapse` matches), so there is no consistency obligation.
- `localStorage` is the natural choice for purely-client UI state; no server-side
  read, no `Set-Cookie` round-trip.
- Origin-scoped (per `localhost:7777`), which matches `shark web`'s deployment model.
- Simpler API: `getItem`/`setItem` vs. `document.cookie` parsing.

**Alternatives rejected**:

- *Cookies*: would require a parser and `Path=/` discipline; provide nothing useful
  here.
- *`sessionStorage`*: state would not survive across browser sessions, defeating the
  user's "I want to set this once" goal in T-E27-F10-004.

#### Decision 2 — Add a CSS class instead of inline `style="opacity:…"`

**Rationale**: matches the existing pattern at `viewer.html:310-322` for
`.entity-cancelled`. Single source of truth for the dimming value; easier to tweak.

#### Decision 3 — One helper `isHiddenTerminalStatus(status, entityType)` instead of inlining the maps

**Rationale**:

- Eliminates the four-call-site bug class that produced T-E27-F10-005 in the first
  place — adding a new entity type or status now requires editing one map, not four
  predicates.
- Co-locates the entity-type-specific terminal lists with the existing
  `TERMINAL_STATUSES` / `CANCELLED_STATUSES` constants for discoverability.
- Keeps the call sites short and readable: `if (!showCompleted &&
  isHiddenTerminalStatus(item.status, 'idea')) continue;`

**Alternative rejected**: inlining per-call-site maps. Would re-introduce the same drift
risk T-E27-F10-005 surfaced.

#### Decision 4 — Keep `showCompleted` variable name; only rename the user-facing label

**Rationale**: Renaming the global to `showAll` would touch ~6 sites for cosmetic
benefit and produce a noisy diff. A single comment on the declaration documents the
new semantics. The user-facing label is the only thing the user sees, so that is what
gets renamed.

#### Decision 5 — Do NOT apply `line-through` to completed (only to cancelled)

**Rationale** (resolves the open question in T-E27-F10-001): the user explicitly
asked whether cancelled should remain distinguishable from completed. Reserving
line-through for cancelled keeps the two visually distinct: completed = dimmed,
cancelled = dimmed + struck. This also gives the colorblind user two non-color
dimensions (opacity and text-decoration) for the cancelled state.

#### Decision 6 — Wrap all `localStorage` access in `try/catch`

**Rationale**: privacy-mode browsers (Safari ITP, Firefox strict mode) throw
`SecurityError` on `localStorage` access in some configurations. The viewer must keep
working for the current session even if persistence is unavailable.

### 2.5 Integration with Existing Code

**Specific integration points** (file paths and line ranges from current `viewer.html`):

1. **CSS — completed class** (insert after line 322, before line 324):

   ```css
   .entity-completed .tree-node-key,
   .entity-completed .tree-node-title {
     opacity: 0.55;
   }
   tr.entity-completed td {
     opacity: 0.55;
   }
   ```

2. **CSS — collapse-all button** (insert in the `.sidebar-filter-bar` block near
   line 232):

   ```css
   .sidebar-filter-bar button {
     /* small, secondary style — match existing toggle visual weight */
   }
   ```

3. **JS — `isHiddenTerminalStatus` helper** (insert after `isCancelledStatus`,
   line 1674):

   ```js
   const HIDDEN_TERMINAL_BY_TYPE = {
     epic:        new Set(['completed', 'cancelled', 'closed', 'wont_fix', 'rejected']),
     feature:     new Set(['completed', 'cancelled', 'closed', 'wont_fix', 'rejected']),
     task:        new Set(['completed', 'done', 'cancelled', 'closed', 'wont_fix', 'rejected']),
     bug:         new Set(['completed', 'done', 'verified', 'closed', 'wont_fix', 'rejected']),
     change_card: new Set(['completed', 'done', 'approved', 'rejected', 'closed']),
     idea:        new Set(['converted', 'archived']),
   };
   function isHiddenTerminalStatus(status, entityType) {
     const s = (status || '').toLowerCase();
     const set = HIDDEN_TERMINAL_BY_TYPE[entityType];
     if (set) return set.has(s);
     // Defensive default: hide if it's terminal OR cancelled by either set
     return TERMINAL_STATUSES.has(s) || CANCELLED_STATUSES.has(s);
   }
   ```

4. **JS — `localStorage` init** (insert at line 2239, replacing the existing
   `let showCompleted = false;`):

   ```js
   // showCompleted now means "show all items including terminal states".
   // Persisted in localStorage as 'shark.viewer.showAllItems'.
   let showCompleted = (function() {
     try { return localStorage.getItem('shark.viewer.showAllItems') === 'true'; }
     catch (e) { return false; }
   })();
   ```

5. **JS — filter call sites** (modify lines 4720, 4762, 4790, 4837):

   - line 4720 (`buildFeatureNodeHtml` task loop): pass `'task'`
   - line 4762 (`buildEpicNodeHtml` feature loop): pass `'feature'`
   - line 4790 (`buildFlatSectionHtml`): the section title is one of `'Bugs'`,
     `'Change Cards'`, `'Ideas'` — map to `'bug'` / `'change_card'` / `'idea'`
     respectively. Pass the entity type as a second parameter to
     `buildFlatSectionHtml(title, items, entityType)`.
   - line 4837 (`renderSidebar` epic loop): pass `'epic'`

6. **JS — sidebar rendering** (modify lines 4820-4829):

   - Change `Show completed` → `Show all items`
   - Add `<button id="collapse-all-btn" type="button">Collapse all</button>` after
     the `.sidebar-filter-label` block.

7. **JS — wiring** (modify lines 4860-4880, where the checkbox change handler lives):

   - In `showCompletedCb.onchange`: after setting `showCompleted = this.checked;`,
     wrap a `try { localStorage.setItem('shark.viewer.showAllItems', String(this.checked)); } catch (e) {}`.
   - Add a click handler:

     ```js
     const collapseAllBtn = document.getElementById('collapse-all-btn');
     if (collapseAllBtn) {
       collapseAllBtn.onclick = function() {
         expandedEpics.clear();
         expandedFeatures.clear();
         renderSidebar();
       };
     }
     ```

8. **JS — table rows** (modify lines 4132, 4246):

   - Today: `const rowClass = isCancelledStatus(f.status) ? ' class="entity-cancelled"' : '';`
   - New:
     ```js
     let rowClass = '';
     if (isCancelledStatus(f.status))             rowClass = ' class="entity-cancelled"';
     else if ((f.status || '') === 'completed')   rowClass = ' class="entity-completed"';
     ```

### 2.6 Testing Strategy

The viewer is a single static asset embedded via `go:embed`. There is no JS unit-test
harness in the repo today (`internal/viewer/assets_test.go` only checks that the
asset is non-empty and parsable). Verification is therefore manual but scripted:

1. **Build**: `make build` (verifies `go:embed` still picks up the asset).
2. **Run**: `./bin/shark web` against a project with mixed entity statuses
   (this repo qualifies — it has completed epics, cancelled features, and ideas in
   various statuses).
3. **AC walk-through**: each AC in §1.1/§1.2 is exercised by hand against the
   running dashboard. The implementation task (downstream of this spec) MUST attach a
   short verification log enumerating each AC and its observed behavior.
4. **Regression**: confirm the existing `assets_test.go` and `wire_test.go` continue
   to pass (`go test ./internal/viewer/...`). They do not assert content beyond
   non-emptiness, so they will pass unchanged.
5. **Quality gate**: `make fmt && make lint && make test` per project policy. (No Go
   files change in this feature, but the gate still runs.)

### 2.7 Risks and Mitigations

| Risk                                                               | Likelihood | Mitigation                                                                                                  |
| ------------------------------------------------------------------ | ---------- | ----------------------------------------------------------------------------------------------------------- |
| Renaming the label confuses users mid-session                       | Low        | Cosmetic; documented in commit message.                                                                     |
| `localStorage` write throws in privacy mode                          | Medium     | REQ-F-004 AC-004.6: all access wrapped in `try/catch`, in-memory state always works.                        |
| `isHiddenTerminalStatus` map drifts from `internal/models/idea.go`  | Low        | The map is co-located with the existing JS constants and called out in the helper's docstring; idea statuses are stable (`new`/`on_hold`/`converted`/`archived`). Adding a new idea status would require updating both Go and the map. |
| Adding the button changes filter-bar height and breaks layout       | Low        | Filter bar already wraps via flex; visual check during AC walk-through.                                     |
| Per-task `done` status now hidden when toggle is OFF (was visible)  | Low        | Intentional — REQ-F-002/REQ-F-005 align task hiding with epic/feature behavior. Documented in AC-005.5.     |

---

## 3. Traceability Matrix

| Task            | Requirement(s)         | Primary AC(s)                |
| --------------- | ---------------------- | ---------------------------- |
| T-E27-F10-001   | REQ-F-001, REQ-NF-002  | AC-001.1…001.5, NF-002       |
| T-E27-F10-002   | REQ-F-002              | AC-002.1…002.5               |
| T-E27-F10-003   | REQ-F-003              | AC-003.1…003.5               |
| T-E27-F10-004   | REQ-F-004, REQ-NF-003  | AC-004.1…004.6, NF-003       |
| T-E27-F10-005   | REQ-F-005              | AC-005.1…005.7               |

---

*Last Updated*: 2026-04-26
*Architect*: Claude (Opus) — E27-F10 specification session
