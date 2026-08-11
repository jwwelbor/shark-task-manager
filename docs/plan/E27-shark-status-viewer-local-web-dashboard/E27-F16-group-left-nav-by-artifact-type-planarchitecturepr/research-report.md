---
research_schema: 2
entity_key: E27-F16
entity_type: feature
recipe: universal
rigor: standard
categories: [frontend, backend]
related_work: true
---

# Research report — Group Left Nav by Artifact Type (Plan/Architecture/Product) + Configurable Browsable Folders

## Scope

E27-F16 regroups the `shark web` left nav — currently one flat stack of
collapsible sidebar sections (`Tags`, `Docs`, `Epics`, `Bugs`, `Change
Cards`, `Tech Debt`, `Ideas`, `Questions`) — into three top-level,
independently collapsible groups: **Plan** (the existing tracked-entity
sections: epic→feature→task hierarchy, bugs, change-cards, tech-debt,
ideas, questions, sprints), **Architecture** (a browsable view of
`docs/architecture/*`), and **Product** (a browsable view of `docs/product/*`).
It also adds a `.sharkconfig.json` section letting a user register
additional browsable folders (relative to the project root, `../`
traversal rejected, reusing the existing path-safety check rather than
introducing a new one).

Vocabulary: a *sidebar section* is the existing collapsible unit rendered
by `renderSidebarSection(sectionId, title, bodyHtml, options)`
(`internal/viewer/assets/viewer.html:8257`) with expand/collapse state
persisted per-section in `localStorage` under `SIDEBAR_SECTION_STORAGE_KEY`
(`viewer.html:3434-3445`). A *folder browser entry* is the existing
`tree-node-folder` pattern keyed `FOLDER_KEY_PREFIX + <relPath>`
(`viewer.html:2454`, `8252-8255`) that, on click, calls the backend
`FolderFiles` endpoint to list a directory's immediate children
(`viewer.html:8462-8463`, `internal/services/viewer_service.go:2572`).
A *top-level group* (Plan/Architecture/Product) is new: today there is no
nesting above individual sidebar sections — `renderSidebar()` concatenates
section HTML directly (`viewer.html:8285-8341`).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F16-group-left-nav-by-artifact-type-planarchitecturepr/feature.md`
  (Problem/Solution sections define the three groups and the config
  requirement, though the rest of the file — User Stories, Requirements,
  Acceptance Criteria, Success Metrics — is still unfilled template
  placeholder text); `internal/viewer/assets/viewer.html:8285-8341`
  (`renderSidebar()` — the current flat section list this feature
  regroups); `.claude/rules/go/input-sanitization.md` (the path-safety
  pattern language the feature.md explicitly points at).
- [x] `affected_implementation_or_contract` — Evidence: `internal/viewer/assets/viewer.html:8257-8272` (`renderSidebarSection` —
  the section-rendering primitive to be nested under a new group wrapper),
  `internal/viewer/assets/viewer.html:8252-8255` (`buildDocsBrowserBodyHtml`
  — the existing single `Docs` folder-browser entry pointed at `docs`,
  the direct precedent for new Architecture/Product entries),
  `internal/services/viewer_service.go:2572-2638` (`ViewerService.FolderFiles`
  — generic, already-secured directory listing keyed by project-root-relative
  path; not hardcoded to `docs/`), `internal/config/config.go:60-117,296-300`
  (`Config` struct and `WebConfig{Port int}` — the only existing web-specific
  config section; it has no field for browsable folders today, so a new
  field/type is required), `.sharkconfig.json` (live project config — no
  `web.browsable_folders`-shaped key present).
- [x] `related_work` — Evidence: epic `docs/plan/E27-shark-status-viewer-local-web-dashboard/epic.md`
  (Option B decision — extend the existing server/viewer stack, do not
  build a second app) and `architecture.md` (component map: SPA + viewer
  API + viewer service, all additive to existing layers); sibling
  `E27-F13-sprint-mode-dashboard-tree-and-planning-surface/RESEARCH-REPORT.md`
  (established precedent: "reuse the current sidebar tree ... instead of
  building a second navigation system" — same reuse posture this feature
  should take for grouping); sibling `E27-F10-tree-view-enhancements/spec.md`
  (documents the existing `.entity-completed`/`.entity-cancelled` styling,
  the "Show all items" toggle, and "Collapse all" — all sidebar-tree
  behaviors that a new group wrapper must not regress); sibling
  `E27-F07-inline-markdown-editor-web-viewer-file-editing/feature.md` and
  commit `86b65785` ("resolve PR review comments — security status codes,
  Escape key, **folder view**, parentDir") — the folder-browser click
  affordance and `FOLDER_KEY_PREFIX` were introduced/refined here, making
  F07 the folder-browsing sibling of record; commit `1f5fc28d`
  ("[codex] viewer web updates and node cleanup (#86)") introduced
  `renderSidebarSection`/`buildDocsBrowserBodyHtml` and the current `Docs`
  section (git log `-S"renderSidebarSection"` / `-S"buildDocsBrowserBodyHtml"`).
- [x] `pattern_contract` — Evidence: `internal/services/viewer_service.go:2576-2638`
  establishes the exact reusable path-containment pattern (`filepath.Abs` →
  `filepath.EvalSymlinks` on both project root and target → `isContained`
  check → `SecurityError` on escape) already applied identically at three
  call sites (`File`, `FileByPath`, `FolderFiles` — lines 2030-2085,
  2159, 2601); this is precisely the "reuse shark's existing path-safety
  check" the feature.md instructs rather than introducing a new one.
  `internal/viewer/assets/viewer.html:8257-8272` establishes the sidebar
  section-rendering contract (collapse state via `data-sidebar-section`
  and `isSidebarSectionExpanded`) that a new "group" wrapper must compose
  with, not replace, to keep the existing per-section
  expand/collapse-all wiring (`viewer.html:8391-8405`) working.
- [x] `dependency_impact` — Evidence: `renderSidebar()`
  (`viewer.html:8285-8341`) is the single call site that must change to
  emit group wrappers around the existing `renderSidebarSection(...)`
  calls; `internal/cli/commands/web.go` and `cmd/server/services.go`
  (cited in `E27-F13-.../RESEARCH-REPORT.md` §Integration Points) are the
  CLI/server wiring points that would need to thread a new
  `web.browsable_folders`-shaped config value from `.sharkconfig.json`
  down to the viewer/summary or hierarchy API response so the frontend
  knows which extra folders to render — no such plumbing exists yet
  (`internal/api/viewer/types.go` has no folder-list field); `shark config
  show` / `shark config validate` (`internal/config/config.go`) are the
  consumers that would surface a misconfigured browsable-folder path, and
  any new config field needs `RawData` preservation compatibility per the
  existing `Config` struct comment ("Store raw config data to preserve
  unknown fields").

## Capability map

| Capability | Source | Decision | Application to E27-F16 |
| --- | --- | --- | --- |
| Collapsible sidebar section primitive (`renderSidebarSection`, per-section localStorage expand state, expand-all/collapse-all wiring) | `internal/viewer/assets/viewer.html:8257-8272, 8391-8405`; introduced by commit `1f5fc28d` | **REUSE** | Keep every existing section (`Epics`, `Bugs`, `Change Cards`, `Tech Debt`, `Ideas`, `Questions`, `Sprint` header at line 6806) exactly as-is; nest them inside a new "Plan" group wrapper rather than rewriting the section renderer. |
| Folder-browser entry + generic `FolderFiles` backend (`buildDocsBrowserBodyHtml`, `FOLDER_KEY_PREFIX`, `ViewerService.FolderFiles`) | `internal/viewer/assets/viewer.html:8252-8255, 2454`; `internal/services/viewer_service.go:2572-2638`; folder-view affordance refined in E27-F07 (commit `86b65785`) | **EXTEND** | `FolderFiles` already accepts any project-root-relative path and is not hardcoded to `docs/` — add two more folder-browser entries (`docs/architecture`, `docs/product`) using the identical pattern as the existing single `Docs` entry, no backend changes needed for the fixed Architecture/Product roots. |
| Path-containment security check (`isContained` + `EvalSymlinks` pattern) | `internal/services/viewer_service.go:2030-2085, 2159, 2601` (three existing call sites) | **REUSE** | This is the exact check the feature.md requires reuse of for validating user-registered browsable-folder paths — do not add a second traversal check; either route registered folders through `FolderFiles` (which already rejects escapes) or extract/call the same `isContained` helper for config-time validation. |
| `WebConfig` (`.sharkconfig.json` `web.*` section) | `internal/config/config.go:296-300` — currently `{Port int}` only | **EXTEND** | Add a new field (e.g. `BrowsableFolders []BrowsableFolderConfig` or similar) to `WebConfig` rather than inventing a new top-level config section — `web.*` is already the viewer-specific home and `RawData` preserves unknown fields for forward/back compatibility. |
| Top-level nav grouping (Plan/Architecture/Product wrapper above sections) | None found — `renderSidebar()` currently concatenates section HTML with no grouping layer (`viewer.html:8285-8341`) | **NEW** | No existing capability to extend; this is genuinely new UI structure (a group header/collapse wrapping multiple existing sections) that must be added without disturbing the section-level state model already in place. |
| Viewer API surface for exposing configured browsable folders to the frontend (`internal/api/viewer/types.go`, hierarchy/summary responses) | `internal/services/viewer_service.go:260-305` (`HierarchyResponse`, `SummaryResponse`) — no folder-list field today | **NEW** | The frontend cannot discover user-registered folders from the config file directly (browser has no filesystem access); a new response field or endpoint is needed to surface `web.browsable_folders` from server to client. |
| Sprint mode's "reuse the tree/detail mechanics, don't build a second nav system" precedent | `E27-F13-sprint-mode-dashboard-tree-and-planning-surface/RESEARCH-REPORT.md` §4-5 | **REUSE** (precedent, not code) | Same posture applies here: F16 should compose the existing section/folder primitives into groups, not introduce a parallel tree component. |
| F10's completed/cancelled dimming and "Show all items"/"Collapse all" sidebar behaviors | `E27-F10-tree-view-enhancements/spec.md` REQ-F-001–004 | **REUSE** (must not regress) | Grouping must not break the existing `showCompleted`/`showAllFiles` filters or the collapse-all button, which currently operate directly on `expandedEpics`/`expandedFeatures` and per-section state — any new group-level collapse control is additive, not a replacement. |

## Findings

1. **The three target groups map cleanly onto existing sidebar sections with
   no restructuring of section internals.** "Plan" is simply all the
   currently-existing entity sections (`Epics`, `Bugs`, `Change Cards`,
   `Tech Debt`, `Ideas`, `Questions`, and the `Sprint` mode header) wrapped
   together; "Architecture" and "Product" are each a single folder-browser
   entry identical in shape to the existing `Docs` entry, just pointed at
   `docs/architecture` and `docs/product` instead of `docs`. The only truly
   new UI concept is the group wrapper itself — `renderSidebar()`
   (`viewer.html:8285-8341`) has no grouping layer today.

2. **The generic `FolderFiles` backend already does everything needed for
   Architecture/Product browsing with zero backend changes** — it is keyed
   by an arbitrary project-root-relative path, not hardcoded to `docs/`
   (`internal/services/viewer_service.go:2572-2638`), and both
   `docs/architecture` and `docs/product` already exist in this repo and
   are readable through it today (verified: `docs/architecture/` has 10+
   files including `adr/`, `SYSTEM_DESIGN.md`; `docs/product/` has
   `cross-epic-integration-map.md`, `progress.md`).

3. **The path-safety check the feature.md asks to reuse is a single,
   already-triplicated pattern** (`isContained` after `EvalSymlinks` on
   both root and target), used identically for `File`, `FileByPath`, and
   `FolderFiles`. For user-registered folders, the lowest-risk design is to
   validate registered paths at config-load/`config validate` time using
   this same helper (or by routing them straight through `FolderFiles`,
   which already enforces it at request time regardless of what the config
   contains) — either way, no new traversal-checking code should be
   written.

4. **There is no existing plumbing to get a configured list of folders from
   `.sharkconfig.json` into the browser.** `WebConfig` currently has only
   `Port`; `HierarchyResponse`/`SummaryResponse` have no folder-list field;
   the frontend's only current "extra folder" is the single hardcoded
   `'docs'` string literal in `buildDocsBrowserBodyHtml()`
   (`viewer.html:8253`). This is real, additive work — a new config field,
   a way to surface it to the viewer API response, and frontend code that
   builds one folder-browser entry per configured folder (Architecture,
   Product, plus zero or more user-registered ones) instead of the single
   hardcoded `Docs` entry.

5. **The feature.md content itself is only partially filled in.** The
   Problem/Solution/Impact narrative is complete and specific, but User
   Stories, Requirements, Acceptance Criteria, and Success Metrics are all
   still template placeholders (`[Describe...]`, `[Requirement Title]`,
   etc.). A specification pass will need to fill these in from the
   Solution narrative and this research's findings — this research does
   not invent acceptance criteria that the feature owner has not stated.

## Decisions

- **Reuse `renderSidebarSection` and `FolderFiles` unchanged**; add a new,
  thin group-wrapper rendering function that nests existing section HTML
  (Plan) or new folder-browser-entry HTML (Architecture, Product) inside a
  collapsible group header, following the same `data-*`/localStorage
  persistence idiom already used at the section level so expand/collapse
  behavior stays consistent between groups and sections.
- **Extend `WebConfig` with a new field for user-registered browsable
  folders** rather than adding a separate top-level config section — this
  keeps all viewer-specific configuration under the existing `web.*` key.
- **Validate registered folder paths using the existing `isContained`/
  `EvalSymlinks` pattern**, either by calling it directly at config-load
  time or by relying on `FolderFiles`'s existing enforcement at request
  time (or both, for early feedback via `shark config validate` plus
  defense-in-depth at request time) — do not write a second path-safety
  check.
- **Add a viewer API surface (new response field or endpoint) that
  publishes the resolved Architecture/Product/user-registered folder list**
  so the frontend can build the corresponding folder-browser entries
  without needing filesystem or config access of its own.
- **Do not touch F10's completed/cancelled/show-all/collapse-all behaviors**
  — the new grouping is purely a rendering wrapper around existing section
  output; those behaviors continue to operate at the section/tree-node
  level.

## Sources

- `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F16-group-left-nav-by-artifact-type-planarchitecturepr/feature.md`
- `docs/plan/E27-shark-status-viewer-local-web-dashboard/epic.md`
- `docs/plan/E27-shark-status-viewer-local-web-dashboard/architecture.md`
- `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F13-sprint-mode-dashboard-tree-and-planning-surface/RESEARCH-REPORT.md`
- `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F10-tree-view-enhancements/spec.md`
- `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F07-inline-markdown-editor-web-viewer-file-editing/feature.md`
- `internal/viewer/assets/viewer.html` (lines 2454, 3401-3445, 8172-8405, 8462-8463, 6806)
- `internal/services/viewer_service.go` (lines 260-305, 2030-2085, 2159, 2558-2638)
- `internal/config/config.go` (lines 60-117, 296-300)
- `.sharkconfig.json`
- `.claude/rules/go/input-sanitization.md`
- `docs/architecture/` and `docs/product/` (directory listings, confirming both exist and are non-empty)
- Git evidence: `git log --oneline -S"renderSidebarSection" -- internal/viewer/assets/viewer.html` (introduced by `1f5fc28d`), `git log --oneline -S"buildDocsBrowserBodyHtml"` (same commit), `git log --oneline -S"FOLDER_KEY_PREFIX"` (folder view refined by `86b65785`, E27-F07)

RECOMMENDED OUTCOME: standard
