---
feature_key: E27-F12
title: Test Plan — Mermaid Diagram Support in Markdown Viewer
---

# Test Plan — E27-F12: Mermaid Diagram Support in Markdown Viewer

**Testing approach**: Manual browser smoke tests. No JS unit-test harness exists (single-file SPA, no build step). One Go gate: `make fmt && make lint && make test`.

---

## AC Test Matrix

| AC | Test Case | Setup / Input | Expected Outcome | Edge Cases |
|----|-----------|---------------|------------------|------------|
| AC-01 | Single valid flowchart renders as SVG | Open a markdown doc with one `flowchart TD\n  A --> B` mermaid block in the spec pane | `<svg>` present inside `.markdown-body`; nodes labelled "A" and "B" visible; arrow edge between them | Node labels with special chars (spaces, hyphens) still render |
| AC-02 | Three distinct diagram types render independently | Doc with three blocks: `flowchart`, `sequenceDiagram`, `stateDiagram` | Three separate `<svg>` elements in DOM order; each diagram visually correct; no cross-contamination of content | One block being very wide must not collapse adjacent block |
| AC-03 | Invalid mermaid syntax → fallback + error banner | Doc with one invalid block (`flowchart\n  A -->`) between two valid ones | Invalid block replaced with escaped `<pre>` + `.mermaid-error-banner` containing failure text; both surrounding blocks render normally | Error message itself must be HTML-escaped (no XSS via error string) |
| AC-04 | CDN blocked → graceful degradation | Block `cdn.jsdelivr.net` via browser devtools "Block request URL" or `/etc/hosts` redirect, then reload a mermaid doc | Every mermaid block falls back to `<pre><code class="language-mermaid">` raw source; headers, lists, tables render normally | Blocking only mermaid CDN (not marked CDN) — marked content still renders |
| AC-05 | Edit → Save round-trip preserves mermaid output | Open mermaid doc, click Edit, click Save without changes | Rendered SVG node/edge count identical before and after save; on-disk file byte-identical (check file size or diff) | Cancel instead of Save must also restore identical render |
| AC-06 | Dark theme — diagrams readable on `--bg-2` | Open flowchart doc; inspect SVG visually and via DevTools | Node fill or text is light (`--fg` ≈ `#e8eaf0`); edge strokes visible against `#161920`; no pure-black-on-near-black | Theme mismatch most likely on secondary/tertiary node fills — verify those too |
| AC-07 | Wide diagram scrolls horizontally within container | Open or author a wide diagram (many columns in a flowchart) | Diagram container scrolls horizontally inside its own `.mermaid-render-container`; right panel not pushed offscreen; page body has no horizontal scroll | Test at narrow viewport (≈1200px) to stress layout |
| AC-08 | No mermaid blocks → render unchanged | Open an existing spec.md that has no mermaid blocks (e.g. this file) | Rendered output byte-equal to pre-feature baseline modulo whitespace; no extra DOM nodes injected | Verify no empty `.mermaid-render-container` divs appear |
| AC-09 | Pinned jsDelivr URL present exactly once | `grep -c 'mermaid' internal/viewer/assets/viewer.html` | Exactly one `<script src="https://cdn.jsdelivr.net/npm/mermaid@10...">` tag; no duplicate or floating `@latest` reference | URL must include explicit major version (`@10` minimum); minor pin preferred |
| AC-10 | `securityLevel: 'strict'` blocks JS click handlers | Open a doc with `click A "javascript:alert(1)"` in mermaid block; click node "A" in rendered SVG | No alert fires; no JS executes; click is silently ignored or prevented | Also verify `innerHTML: false` — no HTML labels in diagram bypass sanitizer |
| AC-11 | Go quality gate passes | `make fmt && make lint && make test` on branch with change | All three commands exit 0; no new lint errors; embed tests (if any) pass | Run on clean checkout to catch any stale build artifacts |
| AC-12 | viewer.html size growth < 5 KB | `wc -c internal/viewer/assets/viewer.html` before and after change | Delta < 5120 bytes | Count bytes not lines — CSS/JS additions are small; Mermaid library is CDN-only |

---

## Integration Scenarios

- **Mermaid + Marked co-existence**: Both CDN scripts load; `marked.parse()` runs first and produces `<pre><code class="language-mermaid">` nodes; `renderMermaidIn` post-processes those nodes. Verify order is preserved — mermaid must not run before `mdDiv.innerHTML` is set.
- **SPA navigation between panes**: Navigate from a mermaid doc to a non-mermaid doc and back. Each navigation calls `renderMarkdownFromString` fresh. Verify no stale SVGs from prior pane remain and new mermaid blocks render correctly (tests `startOnLoad: false` / manual drive path).
- **Edit/Save round-trip with mermaid**: `editOriginalContent` holds raw markdown (fenced source, not SVG). After Save, `renderMarkdownFromString` re-runs and `renderMermaidIn` re-renders. Verify SVG is re-generated from source, not carried over from pre-edit state.
- **Mixed doc (mermaid + tables + frontmatter)**: A realistic spec.md with frontmatter, mermaid blocks, and prose tables. Confirm frontmatter table renders first, mermaid blocks render in their paragraph positions, and surrounding markdown is unaffected.
- **Theme integration**: `themeVariables` hex values (`#161920`, `#e8eaf0`, etc.) match the `--bg-2`/`--fg` values in `viewer.html:12-32`. If the CSS palette is updated in a future feature, the mermaid themeVariables must be updated in sync — note this coupling in a task note.

---

## Test Infrastructure

- **Primary mechanism**: Manual browser smoke tests (Firefox/Chrome DevTools).
- **Fixtures needed**:
  - A markdown file with 1 valid flowchart block (AC-01 baseline).
  - A markdown file with 3 block types: flowchart, sequenceDiagram, stateDiagram (AC-02).
  - A markdown file with bad-syntax block sandwiched between two valid blocks (AC-03).
  - An existing non-mermaid spec.md for regression baseline (AC-08). This file (`spec.md`) works.
- **CDN block simulation**: Browser DevTools → Network → Block request URL pattern `*mermaid*` (AC-04).
- **Go gate**: `make fmt && make lint && make test` — no new Go test files required; the change is HTML/JS only. If `internal/viewer` has embed size assertions, verify they don't hard-code a byte ceiling that would fail AC-12.
- **File size check**: `wc -c internal/viewer/assets/viewer.html` before and after (AC-12).
- **No automated JS tests**: Consistent with E27-F03 baseline; adding a JS test harness is out of scope.
