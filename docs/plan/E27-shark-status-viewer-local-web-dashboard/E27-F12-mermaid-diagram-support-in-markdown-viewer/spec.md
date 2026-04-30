---
feature_key: E27-F12
title: Mermaid Diagram Support in Markdown Viewer
spec_version: 1.0
status: in_specification
---

# Spec — E27-F12: Mermaid Diagram Support in Markdown Viewer

> **Business context**: see epic PRD `docs/plan/E27-shark-status-viewer-local-web-dashboard/epic.md` ("Goal" → Solution; "Phase 3 — Single-File SPA").
> **System architecture**: see epic PRD ("Architecture Decision", "Key Constraints").
> **Feature description**: see `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F12-mermaid-diagram-support-in-markdown-viewer/feature.md`.
> **Upstream feature**: E27-F03 (single-file SPA) shipped the markdown rendering pipeline; E27-F07 (inline editor) added the round-trip path through `renderMarkdownFromString`. This feature plugs Mermaid into the same pipeline as a presentational, additive enhancement.

---

## 1. Requirements

### 1.1 Functional Requirements

> Notation: requirements are **incremental over the epic**. The epic already specifies that the SPA renders markdown via Marked.js (epic Phase 3, "Entity view"). This feature adds rendering for ` ```mermaid ` fenced code blocks specifically; everything else in the markdown pipeline is unchanged.

| ID | Requirement | Trace |
|---|---|---|
| **REQ-F-001** | When a markdown document rendered in the spec pane contains a fenced code block with the language identifier `mermaid` (case-insensitive), the SPA MUST render the block's contents as an inline SVG diagram in place of the code block, instead of emitting it as a `<pre><code>` text node. | feature.md "Observation"; epic Phase 3 ("markdown spec content") |
| **REQ-F-002** | Multiple ` ```mermaid ` blocks in a single document MUST each render independently. A render failure on one block MUST NOT prevent later blocks from attempting to render, and MUST NOT corrupt non-mermaid content. | Defensive layout; mirrors the existing per-pane fallback behaviour at `viewer.html:3722-3727` |
| **REQ-F-003** | If a ` ```mermaid ` block contains a syntax error, the SPA MUST display the **raw mermaid source** (escaped, in a `<pre>`) plus a small inline error banner identifying it as a mermaid render failure. The rest of the document MUST render normally. | feature.md "Why It Matters" (docs must remain readable); CDN-blocked fallback pattern at `viewer.html:3722-3727` |
| **REQ-F-004** | If the Mermaid library is unavailable (CDN blocked, network failure, script error), every ` ```mermaid ` block MUST fall back to the existing rendered-text behaviour: a syntax-highlighted-style `<pre><code>` block that is at least readable as source. The rest of the markdown document MUST render normally via Marked.js. | feature.md "Why It Matters"; epic Key Constraints ("No build step" — CDN reliance is real) |
| **REQ-F-005** | The mermaid render MUST occur AFTER `marked.parse()` populates the DOM, by post-processing the freshly inserted `markdown-body` subtree. Rendering MUST NOT mutate `editOriginalContent` or any state read by the Edit flow (E27-F07). The raw mermaid source on disk is what round-trips through Edit/Save unchanged. | feature.md scope; preserve `editOriginalContent` semantics at `viewer.html:3632, 3849` |
| **REQ-F-006** | Mermaid rendering MUST be invoked from **both** call sites that currently call `marked.parse()`: the frontmatter+body branch and the body-only branch in `renderMarkdownFromString` (`viewer.html:3716, 3718`). Edit-mode → Save round-trips re-render via the same path (`viewer.html:3833, 3849`) and MUST therefore also render mermaid blocks without an additional code path. | `viewer.html:3703-3741` (single rendering function); E27-F07 round-trip |
| **REQ-F-007** | The visual theme of rendered diagrams MUST be `dark` (or the closest Mermaid-supported equivalent). Background, edges, and text MUST be readable against the existing `--bg-2` (`#161920`) markdown surface. The default Mermaid `default` (light) theme is NOT acceptable. | Epic Phase 3 ("dark-themed IDE-style interface"); existing `:root` palette in `viewer.html:12-32` |
| **REQ-F-008** | The Mermaid library MUST be loaded with `securityLevel: 'strict'` (or the strictest setting available in the chosen version). The viewer MUST NOT enable click-handlers, JS execution, or HTML embedding inside diagrams. | Defense-in-depth; viewer is a local app reading repo-controlled markdown but the boundary is non-negotiable |
| **REQ-F-009** | The CDN-loaded Mermaid script MUST be pinned to a specific major+minor version (e.g. `mermaid@10.9.x`), not floating to `latest`. The reference MUST be on jsDelivr to match the existing `marked` reference (`viewer.html:7`). | Epic Key Constraints ("No new dependencies" Go-side; vendor parity for JS-side) |
| **REQ-F-010** | The rendered diagram container MUST be horizontally scrollable when the diagram is wider than the spec-body content area. Diagrams MUST NOT visually overflow into adjacent panels or break the IDE-style 3-pane layout. | Epic Phase 3 (3-panel layout); existing `.markdown-body pre` `overflow-x: auto` precedent at `viewer.html:1075` |

### 1.2 Non-Functional Requirements

| ID | Requirement | Trace |
|---|---|---|
| **REQ-NF-001** | No new Go-side dependencies. No changes to any Go file in `internal/`, `cmd/`, or `pkg/`. The change is strictly inside `internal/viewer/assets/viewer.html`. | Epic Key Constraints; feature.md scope |
| **REQ-NF-002** | One new client-side JS dependency: Mermaid via jsDelivr CDN. No npm install, no build step, no bundler. Same loading model as the existing `marked` script tag at `viewer.html:7`. | Epic Key Constraints ("No build step", existing CDN pattern) |
| **REQ-NF-003** | First-paint of a markdown document with N mermaid blocks MUST keep the existing single-document-pane render budget. Rendering N blocks SHOULD execute concurrently via `mermaid.run({ querySelector: ... })` rather than serially. Target: a typical architecture doc with 3-5 small flowcharts renders to interactive within 500 ms on a warm cache, comparable to the current Marked-only render. | Epic Success Criteria ("Spec documents render as formatted markdown"); contemporary Mermaid 10.x parallel render API |
| **REQ-NF-004** | The viewer MUST continue to function (with degraded mermaid behaviour per REQ-F-004) when offline / CDN-blocked. No new hard CDN dependencies for non-mermaid content. | Existing offline-degraded pattern; no regression |
| **REQ-NF-005** | The viewer MUST NOT introduce a Content Security Policy header server-side, since the existing viewer relies on inline scripts. The new dependency reuses the existing `<script src="https://cdn.jsdelivr.net/...">` allowance. | `cmd/server/main.go` (no CSP today; out-of-scope to add one in this feature) |
| **REQ-NF-006** | Rendered SVGs MUST inherit the IDE-style dark theme via Mermaid's `themeVariables` overrides keyed off the existing CSS custom properties (`--bg-2`, `--fg`, `--accent`, `--border`). Hard-coded hex colors are acceptable for the initial implementation if extracting CSS vars at runtime is awkward, but the chosen palette MUST match the existing `--bg-2` / `--fg` pair. | Epic visual consistency; existing palette at `viewer.html:12-32` |

### 1.3 Acceptance Criteria

Each criterion is a single, testable observation. Tests are E2E / smoke unless marked otherwise; viewer.html has no unit-test harness today.

| ID | Criterion |
|---|---|
| **AC-01** | Loading a markdown document containing a single ` ```mermaid\nflowchart TD\n  A --> B\n``` ` block in the spec pane renders an `<svg>` element inside the `markdown-body` subtree, with at least two visible nodes labelled "A" and "B" and an arrow edge between them. |
| **AC-02** | Loading a markdown document containing three valid ` ```mermaid ` blocks (e.g. flowchart, sequenceDiagram, stateDiagram) renders three independent `<svg>` elements in document order, each visually correct. |
| **AC-03** | Loading a markdown document containing a ` ```mermaid ` block with invalid syntax (e.g. `flowchart\n  A -->`) renders the raw block source as an escaped `<pre>` plus a small inline error banner with text identifying it as a mermaid render failure. The rest of the document below the bad block (including a subsequent valid mermaid block) renders normally. |
| **AC-04** | Loading any markdown document with the Mermaid CDN blocked (simulated by adding an entry that resolves `cdn.jsdelivr.net` to `127.0.0.1`, or by bytecode-blocking the script load) renders every ` ```mermaid ` block as a `<pre><code class="language-mermaid">` source listing. Marked.js content (headers, lists, tables) renders normally. |
| **AC-05** | The same document rendered in view mode and after Edit → Save produces visually identical output for mermaid blocks (same SVG node count, same edge labels). The on-disk source MUST be byte-identical before and after the round-trip when no edits are made. |
| **AC-06** | The rendered SVG diagram is visually readable against `--bg-2` (`#161920`): node fill is darker than the page background OR node text is `--fg` light gray; edge strokes are visible (not pure black-on-near-black). Verified by manual inspection on the canonical test docs. |
| **AC-07** | A diagram wider than the spec-body container scrolls horizontally inside its own container; it does NOT push the right-hand history panel offscreen and does NOT cause the body to scroll horizontally. |
| **AC-08** | A document with no ` ```mermaid ` blocks at all renders identically to the pre-feature behaviour (regression baseline: visit any existing spec.md without mermaid before/after the change; rendered HTML byte-equal modulo whitespace). |
| **AC-09** | The Mermaid script is loaded from a pinned jsDelivr URL (e.g. `https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js` or a more specific minor pin). The URL MUST appear exactly once in `viewer.html`. |
| **AC-10** | Mermaid is initialized with `securityLevel: 'strict'` (or the strictest equivalent for the pinned version) and a dark theme. A diagram containing `click A "javascript:alert(1)" "x"` MUST NOT execute any script when clicked (manual smoke). |
| **AC-11** | `make fmt && make lint && make test` passes on a clean checkout including the change. (The change is HTML/JS-only, but the embed assertion in `internal/viewer` package tests, if present, must still pass.) |
| **AC-12** | The `internal/viewer/assets/viewer.html` file size growth attributable to this feature is < 5 KB of in-file content (the Mermaid library itself is loaded from CDN, not embedded). |

### 1.4 Out of Scope

Restated from `feature.md` "Out of Scope" (which was deferred to refinement); finalized here. Any of these requires a separate feature:

- **Embedding the Mermaid library bundle in the binary** (offline-first). Today's viewer already relies on `marked` from jsDelivr; adopting offline-first is an epic-level change.
- **Mermaid theme switching at runtime** (light/dark toggle). The viewer is dark-themed only; no theme switcher exists.
- **Editing diagrams visually**. The Edit flow remains a textarea on raw markdown source. There is no graphical editor in scope.
- **Diagram export** (download as SVG/PNG). The browser's native "Inspect → Save SVG" suffices for now.
- **Server-side mermaid pre-rendering** (e.g. a Go mermaid renderer). Rendering is purely client-side.
- **Custom mermaid plugins or non-default diagram types** beyond what core Mermaid 10.x supports out of the box. Diagrams in this repo (per `feature.md`: sequence/flow/state) are all stock types.
- **Auto-refresh** of a rendered diagram while editing. Save-and-render is sufficient.
- **Mermaid in the entity-detail Info tab or anywhere outside the markdown spec pane.** Mermaid only renders inside the spec pane's `markdown-body` subtree, never in DTO-driven UI.

---

## 2. Architecture

### 2.1 Component Changes

> Pattern alignment: follows the existing single-file SPA pattern (E27-F03). No new files, no new packages, no Go code. The change is a presentational enhancement layered on top of the existing `renderMarkdownFromString` function in `viewer.html`.

#### Files Modified

| File | Change |
|---|---|
| `internal/viewer/assets/viewer.html` | (a) Add a second `<script>` tag for the pinned Mermaid CDN, immediately after the existing `marked.min.js` reference at line 7. (b) Add a one-time initialization helper `initMermaid()` that calls `mermaid.initialize({ startOnLoad: false, theme: 'dark' or custom dark themeVariables, securityLevel: 'strict' })`. (c) Add a `renderMermaidIn(rootEl)` helper that finds `code.language-mermaid` blocks inside `rootEl`, transforms each parent `<pre>` into a `<div class="mermaid">` wrapper, calls `mermaid.run({ nodes })` (or equivalent for the pinned version), and on per-block error catches the exception and replaces the block with a fallback raw-source `<pre>` plus error banner. (d) Insert a single `renderMermaidIn(mdDiv)` call near `viewer.html:3720`, after `mdDiv` is populated and before `paneEl.appendChild(contentWrapper)`. (e) Add a `.mermaid-error-banner` and `.mermaid-render-container` CSS rule in the existing `<style>` block, near `.markdown-body pre` (`viewer.html:1070`). |

#### Files Created

None. This is a strictly additive change to one existing file.

### 2.2 Data Model Changes

**No data model changes.** No database migrations, no schema changes, no new Go structs, no API changes.

The on-disk markdown source is unchanged: ` ```mermaid ` fenced blocks are already valid CommonMark and are already round-tripped correctly by the Edit flow today (they just render as text). This feature only changes the **presentation** of those blocks in the SPA.

### 2.3 API / Interface Contracts

**No new endpoints.** No existing endpoint shape changes.

The existing `GET /api/v1/viewer/file/{key}` endpoint already returns `content` as the raw markdown string. The change is entirely client-side: the SPA consumes that same string and post-processes mermaid fences after `marked.parse()`.

#### Client-side API surface (informational)

Two new helper functions are added, both in the existing `<script>` block of `viewer.html`. Neither is exported globally beyond the existing module-level scope used by other helpers (`escapeHtml`, `parseFrontmatter`, `renderFrontmatterTable`).

```js
// One-time global init. Idempotent. Called from renderMarkdownFromString
// before the first render. Safe to call repeatedly.
function initMermaid() { /* mermaid.initialize({...}) */ }

// Find every <pre><code class="language-mermaid"> inside rootEl,
// convert to <div class="mermaid"> wrappers, and render via mermaid.run().
// Per-block try/catch swaps in a fallback <pre> + error banner on failure.
async function renderMermaidIn(rootEl) { /* ... */ }
```

Function placement: immediately above `renderMarkdownFromString` (`viewer.html:3703`), since it is the only caller.

### 2.4 Key Technical Decisions

| Decision | Rationale | Alternative considered |
|---|---|---|
| **D1.** Use the **CDN** load model (jsDelivr, pinned tag) for Mermaid, mirroring the existing `marked` reference. | Aligns with epic Key Constraint "No build step". The viewer is already CDN-dependent for `marked`; adding one more script tag is a net-zero increase in failure modes. Embedding mermaid (~2 MB minified) inside `viewer.html` would inflate the embedded asset by ~2× and is out of scope per feature.md. | Vendor the bundle into `internal/viewer/assets/`. Rejected: adds 2 MB to the binary, no precedent for vendored JS in this repo, contradicts epic Key Constraint. |
| **D2.** Render mermaid in a **post-`marked.parse()` pass** by transforming the DOM, rather than via a Marked.js custom renderer / extension. | The post-pass approach is one extra `querySelectorAll` call and zero coupling to Marked's renderer API surface (which differs between Marked v4 and v9+). It is the pattern Mermaid documents officially under "Use with markdown". | Marked custom renderer for `code(code, infostring)`. Rejected: more code, deeper Marked-version coupling, harder error-isolation per block. |
| **D3.** Initialize Mermaid with `startOnLoad: false` and explicitly drive rendering via `mermaid.run({ nodes })` per pane. | The viewer is a SPA: pane content is replaced on every entity navigation. `startOnLoad: true` would only render the first document and miss every subsequent one. Manual `mermaid.run` is the supported "SPA mode" recipe. | `startOnLoad: true` and re-init on pane swap. Rejected: documented anti-pattern, brittle, per-pane re-init has its own race conditions. |
| **D4.** Use `securityLevel: 'strict'` and **disable** clickable nodes, click-bind callbacks, and HTML labels. | The viewer renders developer-authored markdown from the local filesystem, so the trust boundary is the same as `cat`. But the feature description does not mandate clickable diagrams, and the viewer is a long-running local process where one bad markdown file should not be able to execute JS. Defense-in-depth at zero feature cost. | `securityLevel: 'loose'` to permit click-bindings. Rejected: no requirement for it, real risk for zero benefit. |
| **D5.** Use Mermaid's **`'dark'` built-in theme** as the starting baseline, with optional `themeVariables` overrides for `background`, `primaryColor`, `lineColor`, `textColor` to match `--bg-2` / `--fg` / `--accent` / `--border` exactly. | Mermaid 10.x ships a curated `dark` theme that is already reasonably close to the IDE palette. Overriding 4–6 themeVariables is the well-trodden path; reinventing a theme is not. | Custom Mermaid theme via `theme: 'base'` + full themeVariables. Rejected: 30+ knobs to tune for marginal visual gain; defer to a future polish pass. |
| **D6.** **Per-block error isolation** in `renderMermaidIn` via try/catch around each `mermaid.run` call (or per-node call), with the fallback being escaped raw source + an inline error banner. | Mirrors the existing `marked` CDN-blocked fallback pattern at `viewer.html:3722-3727`. A single bad mermaid block should never blank the document. | Single global try/catch around the whole pass. Rejected: one bad block would hide the rest, contradicting REQ-F-002. |
| **D7.** Pin Mermaid to a major+minor version (`mermaid@10` initially; the team may move to a specific minor pin like `@10.9.0` if drift is observed). | Floating `latest` is a known supply-chain risk and a known cause of "it worked yesterday" regressions. Pinning is the same discipline applied to the existing `marked` reference. | Float to `@latest`. Rejected: explicitly a security/stability anti-pattern. |
| **D8.** Keep the **edit round-trip** untouched. The raw markdown source carries through `editOriginalContent` (`viewer.html:3632`), Cancel restores the same string (`viewer.html:3849`), Save re-renders via `renderMarkdownFromString` which re-runs `renderMermaidIn`. | Preserves the E27-F07 contract: source-of-truth is the on-disk markdown, the viewer is a presentation. No state about rendered SVGs needs to persist. | Cache parsed mermaid AST per pane. Rejected: premature optimization; render is fast enough at the documented diagram counts. |
| **D9.** Inject mermaid styles via the existing `<style>` block (one new `.mermaid-error-banner` rule and one `.mermaid-render-container { overflow-x: auto; }` wrapper rule). | Consistent with how every other viewer style is authored. No new stylesheet. The Mermaid library injects its own SVG-internal styles. | External `mermaid.css`. Rejected: viewer is a single embedded file by design (`go:embed`). |
| **D10.** Concurrency: render all mermaid blocks in a single `mermaid.run({ nodes: [...] })` call when supported, otherwise fall back to a sequential `for` loop with per-block try/catch. | Mermaid 10.x supports batch render; this is the documented fast path. Sequential is the safe portable fallback. | One `await mermaid.run` per block, sequentially, always. Rejected: fine for 1-3 blocks, observably slow for 10+ on a complex architecture doc. |

### 2.5 Integration with Existing Code

#### Front-end SPA — single file: `internal/viewer/assets/viewer.html`

Five small additive edits. None modifies existing logic; each is a localized insert.

##### Edit 1 — Script tag (header, ~line 7)

Insert immediately AFTER the existing line:

```html
<script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
```

Add:

```html
<script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
```

(Final pin TBD by implementer based on `mermaid@10` minor at implementation time; record the chosen tag in the task notes.)

##### Edit 2 — CSS (around `viewer.html:1070`, near `.markdown-body pre`)

```css
.mermaid-render-container {
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 12px;
  margin-bottom: 14px;
  overflow-x: auto;          /* REQ-F-010 */
  text-align: center;        /* center small diagrams */
}

.mermaid-render-container svg {
  max-width: 100%;
  height: auto;
}

.mermaid-error-banner {
  background: rgba(220, 60, 60, 0.08);
  border: 1px solid rgba(220, 60, 60, 0.4);
  border-radius: var(--radius-sm);
  color: #ff8b8b;
  font-size: 12px;
  padding: 6px 10px;
  margin-bottom: 6px;
}
```

##### Edit 3 — Helper functions (insert above `renderMarkdownFromString`, ~line 3700)

```js
let __mermaidInitialized = false;

function initMermaid() {
  if (__mermaidInitialized) return;
  if (typeof mermaid === 'undefined') return; // CDN unavailable; REQ-F-004
  mermaid.initialize({
    startOnLoad: false,
    theme: 'dark',
    securityLevel: 'strict',
    themeVariables: {
      // Align with --bg-2 / --fg / --accent / --border (REQ-NF-006)
      background: '#161920',
      primaryColor: '#1e222b',
      primaryTextColor: '#e8eaf0',
      primaryBorderColor: '#2a2f3d',
      lineColor: '#8a8f9e',
      secondaryColor: '#252a35',
      tertiaryColor: '#0d0f12'
    }
  });
  __mermaidInitialized = true;
}

async function renderMermaidIn(rootEl) {
  if (!rootEl) return;
  if (typeof mermaid === 'undefined') return; // REQ-F-004: leave <pre><code> as-is
  initMermaid();

  const codeBlocks = rootEl.querySelectorAll('pre > code.language-mermaid');
  for (const codeEl of codeBlocks) {
    const preEl = codeEl.parentElement;
    const source = codeEl.textContent || '';
    const wrapper = document.createElement('div');
    wrapper.className = 'mermaid-render-container';
    preEl.parentElement.replaceChild(wrapper, preEl);

    try {
      const id = 'mmd-' + Math.random().toString(36).slice(2, 10);
      const { svg } = await mermaid.render(id, source);
      wrapper.innerHTML = svg;
    } catch (err) {
      // REQ-F-003 — per-block fallback
      wrapper.innerHTML =
        `<div class="mermaid-error-banner">Mermaid render failed: ${escapeHtml(String(err && err.message || err))}</div>` +
        `<pre style="white-space:pre-wrap">${escapeHtml(source)}</pre>`;
    }
  }
}
```

Notes:
- `mermaid.render(id, source)` is the v10 single-block API and returns `{ svg, bindFunctions }`. Per-block calls (vs. batched `mermaid.run`) keep the per-block try/catch trivial; D10's batch optimization is a future-only follow-up if perf becomes an issue.
- `escapeHtml` already exists at `viewer.html:2084` — reuse, do not duplicate.

##### Edit 4 — Wire into `renderMarkdownFromString` (~line 3720)

Replace the inner block:

```js
if (typeof marked !== 'undefined') {
  const mdDiv = document.createElement('div');
  mdDiv.className = 'markdown-body';
  if (frontmatter) {
    mdDiv.innerHTML = renderFrontmatterTable(frontmatter) + marked.parse(body);
  } else {
    mdDiv.innerHTML = marked.parse(rawContent);
  }
  contentWrapper.appendChild(mdDiv);
}
```

with:

```js
if (typeof marked !== 'undefined') {
  const mdDiv = document.createElement('div');
  mdDiv.className = 'markdown-body';
  if (frontmatter) {
    mdDiv.innerHTML = renderFrontmatterTable(frontmatter) + marked.parse(body);
  } else {
    mdDiv.innerHTML = marked.parse(rawContent);
  }
  contentWrapper.appendChild(mdDiv);
  // E27-F12: post-process mermaid fences after marked has produced
  // <pre><code class="language-mermaid"> blocks. Async, fire-and-forget;
  // failures are isolated per block (REQ-F-002, REQ-F-003).
  renderMermaidIn(mdDiv);
}
```

The fallback branch (`marked` undefined) remains unchanged — REQ-F-004 already covers that the CDN-blocked branch shows raw source for everything, mermaid included.

##### Edit 5 — No change to `enterEditMode` / `cancelEdit` / `saveEdit`

The Save path (`viewer.html:3833`) calls `renderMarkdownFromString(content, editPaneEl)`, which is now mermaid-aware automatically. The Cancel path (`viewer.html:3849`) calls the same function with `editOriginalContent`. No edit-mode code is touched (REQ-F-005, D8).

#### Go side

**No changes.** Specifically:
- `internal/viewer/assets.go` (the embed wrapper, if present) does not change — it embeds the same path.
- `internal/api/viewer/` handlers are untouched — the `/file/{key}` endpoint already returns the raw markdown string.
- `cmd/server/main.go` is untouched — no new routes, no new headers, no CSP.
- No new tests in `internal/services/viewer_service_test.go`. The change has no service-layer behavior.

#### Test approach

The viewer SPA has no JS unit-test harness today (consistent with E27-F03's "single-file SPA, no build step" constraint). Verification is therefore manual / smoke-driven:

1. **Smoke fixtures**: pick three existing docs in the repo that already contain ` ```mermaid ` blocks (e.g. epic / feature architecture docs under `docs/plan/E27-*/architecture.md`) and confirm AC-01, AC-02, AC-06.
2. **Bad-syntax fixture**: temporarily author a test markdown file with one valid block, one syntactically broken block, and one valid block; confirm AC-03.
3. **CDN-block fixture**: use browser devtools "Network → Block request URL" on the mermaid CDN URL and reload; confirm AC-04.
4. **Round-trip fixture**: open a doc with mermaid blocks, click Edit, click Save without changing the textarea; confirm AC-05 (file mtime may change but content byte-equal; rendered SVG identical).
5. **Regression baseline**: before merging, capture the rendered HTML of an existing non-mermaid spec (e.g. this very file, post-merge) and diff against the post-merge render of the same file. Whitespace-modulo equality required (AC-08).
6. **Lint/format gate**: `make fmt && make lint && make test` (AC-11).

A small per-feature "manual test plan" document in the test-planning phase will codify these into a checklist; this spec deliberately stops at the "what to verify" level since the exit gate for in_specification doesn't include test design.

#### Risk register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Mermaid CDN outage | Low | Low (per REQ-F-004 fallback) | Fall back to raw source; consider vendoring in a future epic if CDN reliance proves painful. |
| Mermaid 10.x → 11.x breaking changes | Low | Medium | Pin to `@10` (D7); revisit when 11 stabilizes. |
| Diagram CSS pollution into `markdown-body` | Low | Low | Mermaid scopes its styles inside the SVG; container is isolated by class name. |
| Performance regression on docs with many large diagrams | Low | Medium | D10 (batch render) is the documented escape hatch if it triggers; unlikely at current diagram counts. |
| `themeVariables` palette drifts from CSS custom properties | Medium | Low | Hex values mirror `viewer.html:12-32`; if the palette is rebrand-changed, this block updates with it. Acceptable manual coupling for a single-file SPA. |

---

## 3. Traceability Matrix

| Requirement | Acceptance Criteria | Files |
|---|---|---|
| REQ-F-001 | AC-01, AC-02 | `viewer.html` Edit 3, Edit 4 |
| REQ-F-002 | AC-02, AC-03 | `viewer.html` Edit 3 (per-block try/catch) |
| REQ-F-003 | AC-03 | `viewer.html` Edit 3 (catch branch + `.mermaid-error-banner`) |
| REQ-F-004 | AC-04 | `viewer.html` Edit 3 (typeof guard) + existing fallback at line 3722 |
| REQ-F-005 | AC-05 | `viewer.html` Edit 4 (mdDiv post-process); no edit-mode change |
| REQ-F-006 | AC-01, AC-05 | `viewer.html` Edit 4 (single call site covers both branches via shared `mdDiv`) |
| REQ-F-007 | AC-06 | `viewer.html` Edit 3 (`themeVariables`) |
| REQ-F-008 | AC-10 | `viewer.html` Edit 3 (`securityLevel: 'strict'`) |
| REQ-F-009 | AC-09 | `viewer.html` Edit 1 (pinned URL) |
| REQ-F-010 | AC-07 | `viewer.html` Edit 2 (`overflow-x: auto`) |
| REQ-NF-001 | AC-11 | (no Go file modified) |
| REQ-NF-002 | AC-09, AC-12 | `viewer.html` Edit 1 |
| REQ-NF-003 | (manual perf smoke) | `viewer.html` Edit 3 |
| REQ-NF-004 | AC-04 | `viewer.html` Edit 3 (typeof guard) |
| REQ-NF-005 | (no server change) | `cmd/server/main.go` (untouched) |
| REQ-NF-006 | AC-06 | `viewer.html` Edit 3 (`themeVariables`) |

Every requirement traces to at least one acceptance criterion or to an explicit "no change" artifact. Every architecture decision references an existing pattern in the codebase or explains the deviation.

---

*Last Updated*: 2026-04-30  
*Architect*: Claude (E27-F12 specification session)
