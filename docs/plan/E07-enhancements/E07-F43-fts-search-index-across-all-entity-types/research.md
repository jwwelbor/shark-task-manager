# E07-F43 Research Report: FTS Search Index Across All Entity Types

**Status**: Research complete
**Feature**: E07-F43 - FTS Search Index Across All Entity Types
**Date**: 2026-07-05

## Summary

E07-F43 should stay a feature and should proceed through research before
specification. The work is cross-cutting: it changes persistence, repository
search semantics, service wiring, CLI output, tag filtering, and note indexing.
The existing code already has two search paths, but neither satisfies the
feature:

- `internal/repository/search/repository.go` has the user-facing cross-entity
  `SearchAll` path, but it is a `LIKE '%query%'` union and currently omits
  note content, ranking, snippets, stemming, and idea rows.
- The same file also has a task-only FTS5 path (`task_search_fts`,
  `RebuildIndex`, `IndexTask`, `Search`, `SearchWithSnippets`,
  `SearchByEpic`, `SearchByFeature`) that is not wired into non-test code and
  only indexes tasks.
- `internal/db/db.go` creates `task_search_fts` opportunistically and only
  warns when FTS5 is unavailable. E07-F43 must turn the selected search backend
  into an explicit capability requirement rather than a silent fallback.

The recommended direction is to replace both legacy paths with one unified
search subsystem and keep `SearchService` as the public service boundary.

## Existing Implementation

### Cross-entity LIKE search

`internal/repository/search/repository.go` defines `EntitySearchResult` and
`SearchAll(ctx, query, entityType)`. `SearchAll` unions epics, features, tasks,
bugs, change-cards, and tech-debt with `LIKE` checks against title/key and some
description columns. It orders by entity type and key, not relevance.

Current gaps:

- CLI allows `--type=idea`, but `SearchAll` does not include ideas.
- Notes are not searched through this path.
- Entity type filtering happens in Go after the full union instead of pushing
  all filters into SQL.
- There is no rank, snippet, stemming, phrase support, or tokenizer strategy.
- `EntitySearchResult.Severity` is overloaded for bug severity and tech-debt
  category. A new result shape should avoid extending that overload.

### Task-only FTS5 stub

`internal/repository/search/repository.go` also contains task-only FTS methods:

- `RebuildIndex`
- `IndexTask`
- `Search`
- `SearchWithSnippets`
- `SearchByEpic`
- `SearchByFeature`

These operate on `task_search_fts` and include examples of FTS5 ranking and
snippet generation, but they are task-only and not connected to current CLI
query search.

`internal/db/db.go` creates `task_search_fts` in `migrateSearchFTS` with
`tokenize='porter unicode61'`, but FTS5 creation failure is logged as a warning
and migrations continue. If E07-F43 chooses FTS5, that behavior must change so
an unsupported runtime fails loudly.

### Service boundary and tag filtering

`internal/services/search_service.go` is the right service boundary to keep. It
wraps the repository and applies `--tag` filtering through `TagQuerier` with AND
semantics. It also preserves B014 behavior by validating tags even when the
search backend returns zero rows.

The new design should push tag filtering into repository SQL for performance,
but preserve the service-level error contract:

- unregistered tags still return `*UnregisteredTagError`
- nil `TagService` with tag filters still returns `*TagFilterUnavailableError`
- zero search hits must not skip tag validation

### CLI integration

`internal/cli/commands/search.go` is already a thin wrapper around
`cli.GetSearchService().SearchAll`. This is the correct command entry point to
preserve. The CLI currently prints only entity type, key, title, status, and bug
severity. E07-F43 should extend output to include rank/snippet when available
and should update help text so `tech_debt` and `idea` are consistently listed.

`internal/cli/service_accessors.go` wires `GetSearchService` to
`repository.NewSearchRepository(db)` and `GetTagService()`. That wiring is the
main CLI composition point for any new search repository/service constructor.

### Notes

Notes are stored in `entity_notes` and accessed through:

- `internal/models/entity_note.go`
- `internal/services/note_service.go`
- `internal/repository/note/repository.go`

There is no `EntityTypeNote`; notes are attached to parent entities. The feature
statement says "notes content folded in", so the cleaner interpretation is to
index notes into the parent entity's search document, not return notes as an
independent entity type. If product wants note rows as first-class results, the
spec must explicitly define a `note` result type and how it maps back to parent
entity context.

## Integration Points

Persistence:

- `internal/db/db.go` - schema version, migrations, and existing
  `task_search_fts` creation
- `internal/repository/search/repository.go` - search repository and result DTOs
- `internal/repository/search/*_test.go` and
  `internal/repository/search_all_test.go` - existing repository coverage

Service layer:

- `internal/services/search_service.go` - query boundary and tag behavior
- `internal/services/tag_service.go` / `internal/services/tag_query.go` - tag
  filter lookup semantics
- `internal/services/note_service.go` - note write path that should trigger
  parent re-indexing
- Entity write services that need indexing hooks: `task_service.go`,
  `feature_service.go`, `epic_service.go`, `bug_service.go`,
  `change_card_service.go`, `tech_debt_service.go`, `idea_service.go`

CLI:

- `internal/cli/commands/search.go` - search command parsing/output
- `internal/cli/service_accessors.go` - service/repository wiring
- `internal/cli/commands/search_query_test.go`,
  `internal/cli/commands/list_tag_test.go` - query/tag CLI contracts

Models:

- `internal/models/entity_note.go` - registered entity types and note model
- entity models for searchable fields and key/title/description conventions

Docs/planning context:

- `docs/plan/E07-enhancements/E07-F43-fts-search-index-across-all-entity-types/feature.md`
- E10-F04 historical FTS origin, E28-F05 tag-based search/list querying, and
  B014 tag validation regression are referenced in existing feature notes.

## Dependency Map

E07-F43 depends on prior search and tagging behavior:

- E10-F04 introduced the task-only FTS5 infrastructure that should be removed
  or replaced.
- E28-F05 added tag-based list/search filtering. This feature must preserve the
  repeatable `--tag` interface and AND semantics.
- B014 fixed tag validation on zero search hits. This must remain covered.
- TD-002 is subsumed: tag filtering should move into search SQL rather than
  post-load Go filtering.

Within E07, this feature also touches ongoing workflow/config infrastructure
only indirectly through docs and entity search discoverability. It should not
change route-based workflow semantics.

## Search Backend Findings

FTS5 is available in the current codebase and documented as a build-tag
capability in `docs/architecture/tech-stack.md`. The existing implementation
uses FTS5 syntax, FTS5 ranking, and `snippet(...)`, but availability is optional.

The Turso article "Beyond FTS5: Building Transactional Full-Text Search in
TursoDB" (2026-01-27, https://turso.tech/blog/beyond-fts5) describes native
Turso FTS as experimental at publication time. It offers:

- `CREATE INDEX ... USING fts`
- tokenizer options including default, raw, simple, whitespace, and ngram
- BM25 scoring through `fts_score`
- highlighting through `fts_highlight`
- transactional index updates
- explicit `OPTIMIZE INDEX` maintenance because background merges are disabled

Recommendation: do not commit to FTS5 in the spec until the architect validates
which backend is supported by this project's actual SQLite/libSQL deployment
targets. If native Turso FTS is chosen, the spec must include a compatibility
strategy for local SQLite tests and cloud/Turso execution.

## Extension vs New Analysis

Extend:

- Keep `SearchService` as the application-facing service.
- Keep `shark search` command parsing and service invocation shape.
- Keep `TagQuerier` validation/error behavior.
- Reuse `EntitySearchResult` only if it is expanded cleanly; otherwise introduce
  a new result DTO and adapt CLI output.
- Reuse repository package location `internal/repository/search`.

Replace:

- Replace `SearchAll` implementation with indexed search plus SQL-level
  entity type/tag filters.
- Replace task-only FTS methods or remove them once the unified path has
  equivalent coverage.
- Replace `task_search_fts` with a unified schema/index. Do not leave both
  active without an explicit migration plan.

Add:

- Search index schema/migration for all searchable entities.
- Backfill/rebuild operation for existing data.
- Service-layer indexing hooks for entity create/update/delete and note create.
- Snippet/ranking fields in repository and CLI output.
- Tests covering idea rows, note content, entity type filtering, tag filtering,
  zero-result tag validation, and backend-unavailable failure behavior.

Avoid:

- SQLite triggers as the primary sync path. The project architecture expects
  business writes through services, and service hooks are easier to test and
  reason about across entity types.
- A second unrelated search API. Consolidate rather than adding another path.
- Treating notes as first-class entity results unless the spec deliberately
  updates the product contract.

## Recommended Implementation Approach

1. Specify backend choice first: FTS5, Turso native FTS, or an adapter that
   supports both. The spec must define the SQL surface, ranking expression,
   highlight/snippet expression, tokenizer/stemming behavior, and unsupported
   backend failure mode.
2. Define a unified search document model with fields for entity type, entity
   ID, key, title, description/body, note text, metadata text, status, tags, and
   parent keys where relevant.
3. Add a migration that drops or supersedes `task_search_fts`, creates the new
   index/table, backfills all current entities, and bumps `CurrentSchemaVersion`.
4. Refactor `SearchRepository.SearchAll` behind the existing service interface.
   Push entity type and tag filters into SQL while retaining service-level tag
   validation semantics.
5. Add a small indexing collaborator used by entity services and `NoteService`
   after successful writes. Keep dependencies optional in tests where search is
   irrelevant.
6. Update `shark search` output to show ranking/snippet fields without breaking
   JSON consumers.
7. Delete or deprecate task-only FTS methods and tests after equivalent unified
   coverage exists.

## Risks and Open Questions

- Backend compatibility: native Turso FTS may not be available in the local
  SQLite driver used by tests. The design needs an explicit strategy.
- Entity type naming: current code uses `change` and `tech_debt`; user-facing
  docs often say change-card and tech-debt. The spec should normalize aliases.
- Notes: parent-folded note search is recommended, but independent note results
  require a product decision.
- Ranking semantics: current `SearchAll` has deterministic key ordering. New
  ranking changes output order and needs tests.
- Re-indexing: service hooks must not leave stale rows on failed transactions.
  If indexing occurs outside the entity write transaction, the spec must define
  retry/rebuild behavior.
- Cloud migrations: `.sharkconfig.json` currently has `skip_migrations: true`
  for configured DB settings. The migration rollout path must account for
  manual/admin migration workflows.

## Exit Gate

- Existing related code identified with paths: complete.
- Extension points documented: complete.
- Backend options and risks identified: complete.
- Actionable recommendations for specification: complete.
