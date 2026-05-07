# Feature Decomposition Review — E27

**Epic:** E27 — Shark Status Viewer — Local Web Dashboard  
**Review Date:** 2026-04-11  
**Reviewer:** Feature Decomposition Review Agent  
**Verdict:** PASS

---

## Summary

The 5-feature decomposition for E27 correctly implements the architectural plan from the epic PRD (Option B: extend cmd/server). All 7 epic success criteria map to specific features, the dependency chain is sound, and the features form a coherent delivery sequence. E27-F03 references the full visual spec and prototype screenshots directly in its description and provides sufficient implementation guidance for the SPA developer. One minor structural observation is noted (boilerplate template remnants in some feature files) but does not affect deliverability.

---

## Requirements Coverage Matrix

| Epic Requirement | Feature(s) Responsible | Coverage |
|---|---|---|
| `shark web` command starts server and opens browser within 2 seconds | E27-F04, E27-F05 | Full |
| Hierarchy loads in < 500ms for projects with 500 tasks | E27-F02 (hierarchy endpoint), E27-F05 (wiring) | Full |
| Spec documents render as formatted markdown | E27-F03 (Marked.js rendering), E27-F02 (`/file/{key}` endpoint) | Full |
| Transition history shows all entries with status badges | E27-F02 (`/history/{key}` endpoint), E27-F03 (History drawer) | Full |
| Works from any subdirectory in a shark project | E27-F01 (dbinit with project root auto-detection), E27-F04 (CLI inherits root detection) | Full |
| Works with both local SQLite and Turso cloud | E27-F01 (extracts cloud-aware init) | Full |
| Left sidebar with collapsible tree, status dots, color coding | E27-F03 | Full |
| Dashboard view with summary cards, progress bars, recent activity, stale entities | E27-F03 | Full |
| Entity view with properties panel + markdown + history | E27-F03 | Full |
| 7 read-only viewer API endpoints | E27-F02 | Full |
| No build step (vanilla JS + CDN) | E27-F03 | Full |
| No new Go dependencies | E27-F04, E27-F05 | Full |
| CORS headers for local development | E27-F05 | Full |
| File path security (`/file/{key}` path resolved from DB, validated within project root) | E27-F02 | Full |
| Port defaults to 7777, auto-increments if in use | E27-F04 | Full |
| `--no-open` flag to skip browser launch | E27-F04 | Full |

---

## Feature Quality Assessment

### E27-F01 — DB Init Extraction (Execution Order 1)

**Quality: Acceptable.** Description is precise about the scope of the refactor: extract `initDatabase()` from `internal/cli/db_init.go` into `internal/dbinit/init.go`, expose `Init(ctx, Options)` and `MustInit(ctx, Options)`, reduce the existing file to a delegate, and update `cmd/server/main.go`. The feature explicitly calls out that it is a pure refactor with no user-facing behavior change and identifies all integration points. No overlap with other features. Suitable for independent task generation.

### E27-F02 — Viewer API Endpoints (Execution Order 2)

**Quality: Acceptable.** Description enumerates all 7 endpoints with their URL patterns, responsibilities, and security constraint for `/file/{key}`. Integration points call out all repositories to be composed. The dependency on E27-F01 is correctly stated. The feature file contains template boilerplate after the substantive content (lines 40-234) but the functional spec itself (lines 1-39) is complete and actionable.

### E27-F03 — Single-File SPA (Execution Order 3)

**Quality: Good — visual spec adequately captured.** The description references:
- `docs/status-viewer-ui.md` by path (the full UI spec with status colors, layout panels, tree structure, dashboard sections, entity/doc views)
- `/home/jwwel/Pictures/status/tracker/` reference screenshots explicitly
- The exact 3-panel layout (fixed header, 320px sidebar, main content area)
- The 4 application states (Pick Folder, Dashboard, Entity, Doc)
- Sidebar tree indentation values (16px/40px/56px), status dot format, and expand/collapse behavior
- All 4 dashboard sections (Status Breakdown, Feature Progress, Active Transitions, Stale Entities)
- Properties panel + Info/Transitions toggle + History drawer layout
- Exact hex color values from the UI spec (e.g., `in_progress` #e7b30e, `blocked` #8c5700, `done` #4d4269)
- Keyboard behavior (Escape to close)
- Click navigation from dashboard widgets to tree nodes

Cross-checking against `docs/status-viewer-ui.md` and the prototype screenshots: all major visual elements are present in the feature description. The screenshots confirm the dark slate 3-panel layout, sidebar tree with colored dots, properties panel, transition history drawer, and dashboard cards — all of which are referenced or implied in the description.

One minor gap: the description does not explicitly mention the flat sidebar sections ("Tags", "Tech Debt", "Ideas", "Change Cards") from the UI spec. These are lower-priority sections and their omission may be intentional scope reduction, but a developer following only the feature file might miss them. This is flagged as a non-blocking observation.

### E27-F04 — shark web CLI Command (Execution Order 4)

**Quality: Acceptable.** Description precisely specifies the Cobra command signature (`shark web [--port N] [--no-open]`), in-process server startup (not shellout), port selection behavior, browser opening (`xdg-open`/`open`), and the dependency on E27-F05's exported `startServer` function. Correctly notes no new Go dependencies. Template boilerplate follows the substantive content but does not affect clarity.

**Note on dependency ordering:** E27-F04 lists only E27-F05 as a dependency ("startServer must exist before CLI calls it"), but E27-F05 in turn depends on E27-F01/F02/F03. This is correctly captured in E27-F05's dependency list. The execution order 4 for F04 and 5 for F05 means F05 must be complete before F04 is integrated — which matches F04's dependency declaration. This is sound.

### E27-F05 — Server Wiring and Integration (Execution Order 5)

**Quality: Acceptable.** Description identifies all files being modified (`cmd/server/main.go`, `cmd/server/services.go`) and the new file (`internal/viewer/assets.go`), specifies the `go:embed` approach, names CORS header requirement, calls out the `startServer(addr, db)` interface contract, and requires integration/smoke tests. Dependencies on F01-F04 are all listed. Template boilerplate follows but does not affect the spec.

---

## Gaps Identified

1. **Flat sidebar sections (non-blocking):** E27-F03 description does not explicitly mention the flat sidebar sections ("Tags", "Tech Debt", "Ideas", "Change Cards") that appear in `docs/status-viewer-ui.md`. The developer will need to read the full UI spec (which is referenced) to discover these. Since the spec is explicitly linked, this is addressable via the reference — not a decomposition gap.

2. **Template boilerplate in E27-F02, F03, F04, F05 (non-blocking):** Four of the five feature files contain blank template scaffolding (User Personas, User Stories, Requirements, Acceptance Criteria with placeholder text). This is cosmetic noise, not a spec gap — the substantive content in the frontmatter + description section is complete. Task generation agents reading these files should use only the frontmatter and the description section above the `## Goal` boilerplate line.

3. **viewer_service_test.go not mentioned in F02 file (non-blocking):** The architecture doc calls for `viewer_service_test.go` in `internal/services/`. E27-F02 does not mention this explicitly in its integration points, though `handler_test.go` is mentioned. Task generators should include the service test file based on architecture doc conventions.

---

## Overlaps Identified

None. The scope boundaries are clean:
- F01 owns DB init extraction only
- F02 owns API handler + service layer only
- F03 owns the HTML file only
- F04 owns the CLI command only
- F05 owns wiring existing pieces together

---

## Ordering Issues

None. The execution order (1→2→3→4→5) correctly reflects the dependency DAG:
- F01 has no E27 dependencies (builds first)
- F02 depends on F01 (DB must work before endpoints are tested)
- F03 depends on F02 (SPA must know the API contract it calls)
- F04 depends on F05 (needs startServer); F05 depends on F01/F02/F03 (needs all pieces)
- F05 depends on F01+F02+F03+F04 (integration feature runs last)

The only subtlety is that F04 and F05 are sequenced as 4→5, but F04 depends on F05. This means F04 task generation should be done after F05 task generation, and F04 implementation should be integrated only after F05's `startServer` export is stable. This is consistent with F04's dependency declaration and is appropriate for task-level sequencing.

---

## E27-F03 Visual Spec Adequacy (Detailed Check)

Per instructions, the following cross-check was performed against `docs/status-viewer-ui.md` and the three prototype screenshots at `/home/jwwel/Pictures/status/tracker/`:

| Visual Requirement | In F03 Description | Status |
|---|---|---|
| Dark slate theme | Implied by "dark-slate IDE-style" | Present |
| 3-panel layout (header/sidebar/content) | Explicitly stated | Present |
| Fixed header with entity count pills + Refresh button | Stated | Present |
| 320px fixed sidebar | Stated | Present |
| Collapsible hierarchy tree | Stated | Present |
| Status dots (colored circles) | Stated | Present |
| Monospace accent-blue key + truncating title | Stated | Present |
| Epic: 16px indent, expanded by default | Stated | Present |
| Feature: 40px indent, collapsed by default | Stated | Present |
| Task: 56px indent | Stated | Present |
| Dashboard: Status Breakdown cards | Stated | Present |
| Dashboard: Feature Progress bars (clickable keys) | Stated | Present |
| Dashboard: Active Transitions (last 25, from/to badges) | Stated | Present |
| Dashboard: Stale Entities (7-day threshold, non-terminal) | Stated | Present |
| Entity view: properties key-value panel | Stated | Present |
| Entity view: Info/Transitions toolbar toggle | Stated | Present |
| Entity view: markdown rendering via Marked.js CDN | Stated | Present |
| History drawer: reverse-chronological from/to badge rows | Stated | Present |
| Doc view: plain markdown rendering | Stated | Present |
| Exact hex color palette (e.g., #e7b30e, #8c5700, #4d4269) | Stated | Present |
| Click-navigation from dashboard widgets to tree node | Stated | Present |
| Escape key closes panels | Stated | Present |
| Pick Folder screen on initial load | Stated | Present |
| Flat sidebar sections (Tags, Tech Debt, Ideas, Change Cards) | Not mentioned | Minor gap (non-blocking — referenced via UI spec) |
| go:embed delivery mechanism | Stated | Present |

**Conclusion:** E27-F03 adequately captures the visual requirements. A developer implementing this feature from the description alone will produce the correct interface. The single missing element (flat sidebar sections) is discoverable via the explicitly linked UI spec.

---

## Recommendations

1. Before task generation for E27-F03: No changes required. Developer must read `docs/status-viewer-ui.md` in full, which the feature description directs them to do.

2. Before task generation for E27-F02, F04, F05: The template boilerplate sections after the substantive content can be ignored by task generators. If spec completeness is desired for consistency, the boilerplate could be trimmed, but it does not block task generation.

3. Task generation order should follow execution order: F01 → F02 → F03 → F04 → F05. The F04/F05 circular-looking dependency resolves at the implementation level: F05 is implemented first (providing `startServer`), then F04 completes its integration.

---

*Review completed: 2026-04-11*
