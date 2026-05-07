---
feature_key: E07-F43-fts-search-index-across-all-entity-types
epic_key: E07
title: FTS Search Index Across All Entity Types
description: Replace the LIKE-based SearchAll and dead task-only FTS5 stub with a unified full-text search index covering all 8 entity types (epic/feature/task/bug/change/tech-debt/idea/note). Notes content folded in; synced via service-layer hooks. Adds proper ranking, snippets, and stemming across the whole project. Implementation may not be FTS5 — see https://turso.tech/blog/beyond-fts5 for libSQL alternatives that should be evaluated before committing to FTS5.
size:
---

# FTS Search Index Across All Entity Types

**Feature Key**: E07-F43-fts-search-index-across-all-entity-types

---

## Triage Notes

### Observed problem

`shark search` (full-text query mode) does substring `LIKE '%query%'` matches against `title`, `key`, and (for some types) `description` only. It misses notes content, has no relevance ranking, no snippets, no stemming. A separate FTS5 infrastructure exists in the codebase but is unused, tasks-only, and has no sync triggers — see `internal/repository/search/repository.go:147` (`RebuildIndex`/`IndexTask` are never called from non-test code).

### Proposed scope

- Single unified search index covering all 8 entity types: epic, feature, task, bug, change-card, tech-debt, idea, note
- Notes content folded into the parent entity's row (re-indexed on note write)
- Synced via service-layer hooks rather than SQLite triggers (CLAUDE.md guarantees all writes go through services)
- Replaces both: the LIKE-based `SearchAll` and the dead task-only FTS5 stub

### Implementation approach (TBD)

FTS5 is the obvious starting point but **may not be the right answer**. Review https://turso.tech/blog/beyond-fts5 before committing — libSQL may now offer better-than-FTS5 options (vector/BM25/etc.) that change the design. The "FTS" in the title is generic, not FTS5-specific.

### Decomposition (5 tasks, sized L overall)

1. Schema migration: drop `task_search_fts`, add unified search table, backfill from all 8 source tables — bump `CurrentSchemaVersion`
2. Unified `SearchService` API with MATCH + snippets + entity-type/tag filters
3. Wire `Index/Remove` hooks into all 8 entity services + `NoteService` (re-index parent on note write)
4. Rewire `runSearchQuery` (CLI) to use new service; preserve `--type` and `--tag` flags
5. Delete legacy: `Search`, `SearchWithSnippets`, `SearchByEpic`, `SearchByFeature`, `RebuildIndex`, `IndexTask`, `criterion_text` references, LIKE-based `SearchAll`

### Risks to resolve during refinement

- **Turso/libSQL FTS5 support** — must be a hard requirement, not a silent fallback. Currently `internal/db/db.go:1125` swallows FTS5 unavailability with a warning. If the chosen approach is FTS5, that fallback must become an error.
- **Tags and dependency keys in metadata column** — decide upfront whether they're indexed in the metadata column. Adding later means a re-backfill migration.
- **Dependency search semantics** — graph queries like "what depends on X" stay structured. Folding dependency keys into the indexed metadata gives `search "E07-F01-001"` enough surface to find blockers, but FTS isn't a graph engine.

### Subsumes / supersedes

- **TD-002** — Push --tag filter into search FTS UNION SQL instead of post-load Go filter. The unified index pushes tag filtering into SQL natively. Close TD-002 when this ships.

### Historical context

- E10-F04 (completed) — Acceptance Criteria & Search. Origin of the dead FTS5 stub.
- E28-F05 (completed) — Tag-Based Querying in List and Search. Most recent search work; introduced the `--tag` flag this feature must preserve.
- B014 (completed) — past search bug, reference only.

### Reference

- https://turso.tech/blog/beyond-fts5 — libSQL search options to evaluate before committing to FTS5

---

*Last Updated*: 2026-05-05
*Status*: Triage stub — refinement pending.
