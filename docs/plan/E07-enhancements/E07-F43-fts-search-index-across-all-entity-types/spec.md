# E07-F43 Specification: FTS Search Index Across All Entity Types

**Status**: Ready for test planning
**Feature**: E07-F43 - FTS Search Index Across All Entity Types
**Research**: [research.md](research.md)
**Parent Epic**: [E07 Enhancements](../epic.md)

## Scope

This feature replaces Shark's current split search implementation with one
unified indexed search subsystem for project entities. It is incremental to the
E07 enhancement epic and does not restate epic-level goals.

In scope:

- Replace `SearchRepository.SearchAll` with an indexed search implementation.
- Replace or remove the dead task-only FTS5 path and `task_search_fts`.
- Index searchable content for epics, features, tasks, bugs, change-cards,
  tech-debt, ideas, and notes folded into parent entity documents.
- Preserve `shark search` query mode, `--type`, `--tag`, and JSON output.
- Add ranking, snippets/highlights, stemming/tokenization behavior, and SQL-level
  filtering for entity type and tags.
- Use SQLite FTS5 as the primary backend for this feature and fail loudly when
  FTS5 is unavailable.

Out of scope:

- Semantic/vector search.
- Graph queries such as "what depends on X" beyond indexing dependency keys as
  searchable metadata.
- A new interactive search UI.
- Treating notes as first-class result rows unless a later product decision
  changes the contract.
- Changing workflow routing, claim leases, or status semantics.

## User Stories

1. As a Shark user, I can run `shark search "term"` and find matching work items
   across all tracked entity types, including ideas and tech-debt.
2. As a Shark user, I can find entities by content stored in their notes without
   separately searching notes.
3. As a Shark user, I can filter search results by entity type and registered
   tags without losing existing validation behavior.
4. As a Shark maintainer, I can rebuild or backfill the search index after a
   migration or data repair.
5. As an implementer, I can rely on one search path instead of choosing between
   a LIKE union and an unused task-only FTS path.

## Functional Requirements

| ID | Requirement |
| --- | --- |
| REQ-F-001 | `shark search "query"` MUST use a unified indexed search path rather than `LIKE '%query%'` scans. |
| REQ-F-002 | Search MUST cover epics, features, tasks, bugs, change-cards, tech-debt, and ideas as result entity types. |
| REQ-F-003 | Notes MUST be folded into their parent entity search document so note content can make the parent entity match. |
| REQ-F-004 | Search results MUST include entity type, database ID, key, title, status, rank, and snippet fields in JSON mode. |
| REQ-F-005 | Human-readable output MUST include snippets when available and preserve readable entity/key/title/status output. |
| REQ-F-006 | `--type` MUST filter in the search backend for all accepted values: epic, feature, task, bug, change, idea, tech_debt. |
| REQ-F-007 | `--tag` MUST preserve repeatable AND semantics and MUST push eligible filtering into SQL/index queries. |
| REQ-F-008 | Unregistered tags MUST still return the existing `UnregisteredTagError` behavior even when the search query has zero matches. |
| REQ-F-009 | If tag filtering is requested and `TagService` is unavailable, search MUST return the existing `TagFilterUnavailableError`. |
| REQ-F-010 | Entity create/update/delete operations MUST update or remove the corresponding search document. |
| REQ-F-011 | Note creation MUST re-index the parent entity document. |
| REQ-F-012 | A full rebuild/backfill operation MUST rebuild the index from all source tables and notes. |
| REQ-F-013 | The FTS5 backend MUST provide relevance ranking, snippets, and `porter unicode61` tokenizer behavior unless the implementation documents a strictly compatible replacement. |
| REQ-F-014 | Task-only FTS methods and `task_search_fts` MUST be removed or explicitly superseded once the unified path is live. |
| REQ-F-015 | The implementation MUST close or explicitly supersede TD-002 when SQL-level tag filtering is delivered. |

## Non-Functional Requirements

| ID | Requirement |
| --- | --- |
| REQ-NF-001 | FTS5 availability MUST be verified during schema setup or search initialization; silent fallback to degraded LIKE search is not allowed. |
| REQ-NF-002 | Search must remain deterministic for equal-rank rows by applying stable secondary ordering by entity type and key. |
| REQ-NF-003 | Index writes MUST be transactionally consistent with entity writes or have a documented rebuild/retry recovery path. |
| REQ-NF-004 | Tests MUST run in the local Go test environment using SQLite FTS5 and MUST NOT require a cloud-only Turso feature. |
| REQ-NF-005 | The new search path MUST avoid N+1 tag lookups; tag filters are validated once and applied in set-oriented SQL. |
| REQ-NF-006 | Query input MUST be escaped or parameterized for the selected backend's match syntax. |

## Acceptance Criteria

| ID | Scenario | Expected Result |
| --- | --- | --- |
| AC-001 | Search for a term in an epic title. | The epic appears with entity type `epic`, rank, and snippet. |
| AC-002 | Search for a term in a feature description. | The feature appears without requiring a title/key match. |
| AC-003 | Search for a term in a task note. | The parent task appears because note content is folded into the task document. |
| AC-004 | Search for a term in a bug description with `--type=bug`. | Only matching bugs appear. |
| AC-005 | Search with `--type=idea`. | Matching ideas appear; no unsupported-type error is returned. |
| AC-006 | Search with `--type=tech_debt`. | Matching tech-debt rows appear with correct key/title/status. |
| AC-007 | Search with two `--tag` flags. | Results include only entities carrying all requested tags. |
| AC-008 | Search with an unregistered tag and zero text matches. | The command still returns the existing unregistered-tag error. |
| AC-009 | Search when backend support is unavailable. | Setup or search returns a clear error rather than silently degrading. |
| AC-010 | Update an entity title. | A subsequent search finds the new title and no stale old-title-only row. |
| AC-011 | Delete or cancel an entity where search rows should be removed. | The search index no longer returns stale rows for removed content. |
| AC-012 | Run full index rebuild/backfill. | All supported entity types and parent-folded notes are searchable afterward. |
| AC-013 | JSON output is requested. | Results include stable fields needed by downstream tooling, including rank and snippet. |
| AC-014 | Legacy task-only FTS tests are replaced. | No production-only dead FTS path remains untested or unused. |

## Architecture Overview

Use the existing CLI -> service -> repository layering:

- CLI parsing remains in `internal/cli/commands/search.go`.
- Business behavior remains in `internal/services/search_service.go`.
- Data access and backend-specific query syntax remain in
  `internal/repository/search/repository.go`.
- Schema and migrations remain in `internal/db/db.go`.

The design introduces a unified search document abstraction. Each indexed row
represents one searchable parent entity. Notes contribute text to their parent
entity row instead of becoming independent result rows.

## Component Changes

| Path | Change |
| --- | --- |
| `internal/db/db.go` | Add migration for unified search schema/index, bump `CurrentSchemaVersion`, and replace silent FTS5 skip with selected-backend capability validation. |
| `internal/repository/search/repository.go` | Replace LIKE union and task-only FTS methods with unified index CRUD, rebuild, and query methods. |
| `internal/services/search_service.go` | Preserve public `SearchAll` behavior while coordinating tag validation and SQL-level tag filtering inputs. |
| `internal/services/note_service.go` | Re-index parent entity after successful note creation. |
| `internal/services/task_service.go` | Add search indexing hook for task create/update/delete paths. |
| `internal/services/feature_service.go` | Add search indexing hook for feature create/update/delete paths. |
| `internal/services/epic_service.go` | Add search indexing hook for epic create/update/delete paths. |
| `internal/services/bug_service.go` | Add search indexing hook for bug create/update/delete paths. |
| `internal/services/change_card_service.go` | Add search indexing hook for change-card create/update/delete paths. |
| `internal/services/tech_debt_service.go` | Add search indexing hook for tech-debt create/update/delete paths. |
| `internal/services/idea_service.go` | Add search indexing hook for idea create/update/delete/promote paths. |
| `internal/cli/service_accessors.go` | Wire the search repository/service with any new dependencies. |
| `internal/cli/commands/search.go` | Update help text and output formatting for rank/snippet fields. |
| `internal/repository/search_all_test.go` | Replace LIKE-focused coverage with unified search coverage. |
| `internal/repository/search_repository_test.go` | Replace task-only FTS tests with backend-specific unified index tests. |
| `internal/services/search_service_test.go` | Preserve and extend tag filtering behavior tests. |
| `internal/cli/commands/search_query_test.go` | Cover accepted types and output behavior. |
| `internal/cli/commands/list_tag_test.go` | Preserve repeatable `--tag` parsing/search behavior. |

## Data Model

The exact DDL depends on backend choice, but the logical search document must
store or derive these fields:

| Field | Purpose |
| --- | --- |
| entity_type | One of epic, feature, task, bug, change, tech_debt, idea. |
| entity_id | Source table primary key for tag joins and parent lookups. |
| key | Shark key for display and key search. |
| title | Primary display/search field. |
| body | Description or body text from the source entity. |
| note_text | Aggregated note content for the parent entity. |
| metadata_text | Status, parent keys, tags, dependency keys, and other searchable metadata. |
| status | Display field returned in result DTO. |
| updated_at | Optional staleness/debug aid for rebuild checks. |

E07-F43 uses SQLite FTS5 as the primary backend. Replace `task_search_fts` with
a unified FTS virtual table plus companion metadata needed for entity
ID/type/tag filtering. The repository boundary may keep the implementation
swappable for future Turso native FTS support, but this feature does not require
or depend on cloud-only native Turso FTS.

## Interface Contracts

Repository contract:

- Query accepts context, search query, optional entity type, zero or more tag
  names or resolved tag IDs, and limit/options.
- Query returns ordered `EntitySearchResult` values with rank and snippet.
- Rebuild recreates all search documents from current entity and note tables.
- Index/remove operations accept entity type and entity ID or key.

Service contract:

- `SearchService.SearchAll` keeps the existing command-facing semantics.
- Tag errors remain typed and compatible with existing CLI error handling.
- Empty query still returns an empty result set.

CLI contract:

- `shark search "query"` remains the primary interface.
- `--file` mode remains task file search and is not part of this index.
- `--json` returns result objects with the existing base fields plus rank and
  snippet.

## Key Decisions

| Decision | Rationale |
| --- | --- |
| Keep `SearchService` and `shark search` as boundaries. | This matches existing layering and limits CLI churn. |
| Fold notes into parent search rows. | `EntityTypeNote` does not exist; parent-folding matches the feature text and avoids inventing a new result type. |
| Push filters into the repository query. | TD-002 exists because post-load filtering is inefficient and semantically awkward. |
| Use SQLite FTS5 as the primary backend. | The project already documents/builds with FTS5, local Go tests can exercise it, and Turso native FTS was experimental in the cited January 2026 article. |
| Keep a repository boundary that can support a later Turso-native adapter. | Future native FTS may offer better BM25/highlight ergonomics, but it should not block this feature or make tests cloud-dependent. |

## Cross-Feature Interactions

No E07 interaction map was found for I-## contracts. This feature consumes
historical behavior from E10-F04 and E28-F05 by reference, but no formal I-##
contract is declared in this epic directory.

## Cross-Epic Integrations

No E07 cross-epic map or global X-## row was found for this feature. The feature
does validate cross-entity behavior across bug, change-card, tech-debt, idea,
task, feature, and epic domains, but that is internal to Shark's entity model,
not a product-level X-## integration.

## Testing Requirements

Testing must cover:

- Repository-level unified search for each entity type.
- Note content folded into parent rows.
- Entity type filter pushed into the query.
- Tag filter happy path and unregistered-tag error path.
- B014 zero-hit tag validation.
- Backend unavailable failure.
- Rebuild/backfill behavior.
- CLI text and JSON output.
- Migration from existing `task_search_fts`.

Repository tests may use a real test database. Service and CLI tests must use
mocks per `.claude/rules/testing/architecture.md`.

## Risks

- FTS5 support may be absent if the binary is built without the required build
  tag; this feature must turn that into a loud setup/runtime error.
- Future Turso native FTS behavior may differ from FTS5 for ranking, tokenizers,
  and snippets/highlights.
- Service hooks may create stale index rows if not transactionally coordinated.
- Existing JSON consumers may rely on current result shape and order.
- Alias drift between `change`, `change_card`, `tech_debt`, and user-facing
  spellings can create type filter regressions.

## Open Questions Resolved for This Spec

- Notes are treated as parent-folded content, not independent result rows.
- SQLite FTS5 is the chosen backend for this feature; future Turso native FTS
  can be evaluated behind the repository boundary later.
- No interaction-map IDs are invented because no E07 interaction map exists.
