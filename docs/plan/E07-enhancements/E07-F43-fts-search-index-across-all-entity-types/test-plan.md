# E07-F43 Test Plan: FTS Search Index Across All Entity Types

**Status**: Ready for task generation
**Feature Spec**: [spec.md](spec.md)
**Research**: [research.md](research.md)

## Strategy

Use production-shaped tests at the correct layer:

- Repository tests may use a real test database and should verify FTS5 schema,
  backfill, query ranking/snippets, and entity coverage.
- Service tests must mock repositories and tag services to verify business
  semantics, error propagation, and B014 zero-hit tag validation.
- CLI tests must use mocked services or command-level seams; they must not use a
  real database.

The implementation must first write failing tests for the relevant task slice,
then implement. Full Go quality gate still applies after Go changes:
`make fmt && make lint && make test`.

## AC Test Matrix

| AC | Technique | Primary Test Case | Edge Cases |
| --- | --- | --- | --- |
| AC-001 | Equivalence partitioning | TC-001 repository search returns epic title match with rank/snippet | mixed case query; no match |
| AC-002 | Equivalence partitioning | TC-002 repository search returns feature description/body match | empty description; key-only match |
| AC-003 | Contract surface enumeration | TC-003 note creation re-indexes parent task and search finds note text | multiple notes; note update via rebuild |
| AC-004 | Decision table | TC-004 `--type=bug` returns only bugs | bug severity present; non-bug same text excluded |
| AC-005 | Equivalence partitioning | TC-005 `--type=idea` finds ideas | idea status variants; invalid type still rejected |
| AC-006 | Equivalence partitioning | TC-006 `--type=tech_debt` finds tech-debt rows | category/severity metadata present |
| AC-007 | Decision table | TC-007 two tags apply AND semantics in SQL/search path | one tag only; no entity has both |
| AC-008 | Attack-class enumeration | TC-008 unregistered tag with zero text hits returns typed tag error | registered tag with zero hits returns empty results |
| AC-009 | Failure injection | TC-009 backend unavailable fails loudly | migration path; search initialization path |
| AC-010 | State transition | TC-010 update entity title refreshes index row | old title no longer matches alone |
| AC-011 | State transition | TC-011 delete/remove entity removes index row | deleted note text no longer matches parent |
| AC-012 | State transition | TC-012 rebuild backfills all entity types and notes | empty database; partial stale index |
| AC-013 | Contract surface enumeration | TC-013 CLI JSON includes rank and snippet fields | empty result JSON; text output remains readable |
| AC-014 | Contract surface enumeration | TC-014 dead task-only FTS path removed/superseded | no production caller to deleted methods |

## Caller-Path Contracts

### TC-001: Epic Title Match

**Requirement**: AC-001, REQ-F-001, REQ-F-002, REQ-F-004
**Test Location**: `internal/repository/search_all_test.go`
**Entrypoint**: `SearchRepository.SearchAll(ctx, "enhancements", nil)` from `internal/repository/search/repository.go`
**Lowest Allowed Mock Seam**: real repository test database
**Forbidden Mocks**: do not mock `SearchRepository` or call private query helpers
**Counter-factual**: a LIKE-only or task-only implementation would not return a ranked/snippet-bearing epic result.

### TC-002: Feature Description Match

**Requirement**: AC-002, REQ-F-001, REQ-F-002
**Test Location**: `internal/repository/search_all_test.go`
**Entrypoint**: `SearchRepository.SearchAll(ctx, "unified index", &featureType)`
**Lowest Allowed Mock Seam**: real repository test database
**Forbidden Mocks**: do not seed the search index directly without source feature data plus backfill/index operation
**Counter-factual**: an implementation that indexes only title/key would miss description-only matches.

### TC-003: Note Text Re-indexes Parent

**Requirement**: AC-003, REQ-F-003, REQ-F-011
**Test Location**: `internal/services/note_service_test.go` and repository integration coverage in `internal/repository/search_all_test.go`
**Entrypoint**: `NoteService.AddNote(ctx, entityType, entityKey, noteType, content, createdBy)` from `internal/services/note_service.go`
**Lowest Allowed Mock Seam**: in service tests, mock the search index collaborator; in repository tests, use real DB
**Forbidden Mocks**: do not bypass `NoteService.AddNote` by inserting directly into `entity_notes` for the service hook test
**Counter-factual**: an implementation that creates notes but never re-indexes parent entities would fail to find the parent by note text.

### TC-004: Bug Type Filter

**Requirement**: AC-004, REQ-F-006
**Test Location**: `internal/repository/search_all_test.go`
**Entrypoint**: `SearchRepository.SearchAll(ctx, "login", &bugType)`
**Lowest Allowed Mock Seam**: real repository test database
**Forbidden Mocks**: do not post-filter results in the test before asserting
**Counter-factual**: an implementation that applies type filtering after returning all rows would expose non-bug rows or hide SQL-filter regressions.

### TC-005: Idea Type Filter

**Requirement**: AC-005, REQ-F-002, REQ-F-006
**Test Location**: `internal/repository/search_all_test.go` and `internal/cli/commands/search_query_test.go`
**Entrypoint**: repository `SearchAll` plus CLI `validateSearchType("idea")`
**Lowest Allowed Mock Seam**: repository test DB; no DB in CLI unit test
**Forbidden Mocks**: do not assert only CLI type validation; repository must prove idea rows are indexed
**Counter-factual**: the current implementation accepts `idea` in CLI but does not return ideas from repository search.

### TC-006: Tech-Debt Type Filter

**Requirement**: AC-006, REQ-F-002, REQ-F-006
**Test Location**: `internal/repository/search_all_test.go`
**Entrypoint**: `SearchRepository.SearchAll(ctx, "refactor", &techDebtType)`
**Lowest Allowed Mock Seam**: real repository test database
**Forbidden Mocks**: do not reuse task rows as fake tech-debt rows
**Counter-factual**: an implementation that mishandles `tech_debt` naming or filtering would return zero rows or wrong entity type.

### TC-007: SQL-Level Tag AND Filter

**Requirement**: AC-007, REQ-F-007, REQ-NF-005
**Test Location**: `internal/services/search_service_test.go` plus repository integration coverage
**Entrypoint**: `SearchService.SearchAll(ctx, "query", "", []string{"voice", "auth"})`
**Lowest Allowed Mock Seam**: service test may mock repository and `TagQuerier`; repository integration must use real DB
**Forbidden Mocks**: do not simulate tag filtering solely by manually slicing result arrays in the test
**Counter-factual**: a post-load or OR-semantics implementation would keep entities that have only one requested tag.

### TC-008: Zero-Hit Unregistered Tag Validation

**Requirement**: AC-008, REQ-F-008
**Test Location**: `internal/services/search_service_test.go`
**Entrypoint**: `SearchService.SearchAll(ctx, "no-match", "", []string{"missing-tag"})`
**Lowest Allowed Mock Seam**: mock repository returning empty slice and mock `TagQuerier`
**Forbidden Mocks**: do not skip `TagQuerier.EntityIDsByTags` because there are no result buckets
**Counter-factual**: the B014 regression would return an empty result without validating the missing tag.

### TC-009: FTS5 Unavailable Fails Loudly

**Requirement**: AC-009, REQ-NF-001, REQ-NF-004
**Test Location**: `internal/db/db_test.go` or focused migration test near existing DB migration tests
**Entrypoint**: schema/migration function that creates unified FTS5 index
**Lowest Allowed Mock Seam**: controlled test DB or build/environment seam that simulates FTS5 create failure
**Forbidden Mocks**: do not assert on printed warnings only
**Counter-factual**: the current `migrateSearchFTS` behavior would swallow FTS5 creation errors and continue.

### TC-010: Entity Update Refreshes Index

**Requirement**: AC-010, REQ-F-010
**Test Location**: service tests for one representative service plus repository search integration
**Entrypoint**: production update method on representative entity service, such as `FeatureService.UpdateFeature`
**Lowest Allowed Mock Seam**: service test may mock search index collaborator; repository test uses real DB
**Forbidden Mocks**: do not call the indexer directly without exercising the service update path
**Counter-factual**: an implementation that updates the source row but not the index would continue returning stale title text.

### TC-011: Entity Removal Removes Index Row

**Requirement**: AC-011, REQ-F-010
**Test Location**: service tests for representative delete/cancel path plus repository integration
**Entrypoint**: production delete/cancel method used by CLI for the selected representative entity
**Lowest Allowed Mock Seam**: search index collaborator in service tests
**Forbidden Mocks**: do not delete from index directly in service behavior tests
**Counter-factual**: stale index rows would keep deleted content searchable.

### TC-012: Full Rebuild Backfills Everything

**Requirement**: AC-012, REQ-F-012
**Test Location**: `internal/repository/search_all_test.go`
**Entrypoint**: unified rebuild/backfill method on `SearchRepository`
**Lowest Allowed Mock Seam**: real repository test database
**Forbidden Mocks**: do not seed search table directly as the only setup
**Counter-factual**: an implementation that only backfills tasks would miss epics, features, bugs, change-cards, tech-debt, ideas, or notes.

### TC-013: CLI JSON and Text Output Contract

**Requirement**: AC-013, REQ-F-004, REQ-F-005
**Test Location**: `internal/cli/commands/search_query_test.go`
**Entrypoint**: `runSearchQuery(cmd, args)` with production flag parsing shape
**Lowest Allowed Mock Seam**: mocked search service at CLI service boundary
**Forbidden Mocks**: do not test only `printEntitySearchResults`; drive command parsing and output mode
**Counter-factual**: an implementation that forgets rank/snippet JSON fields or breaks text output would fail the output assertions.

### TC-014: Legacy FTS Path Removed or Superseded

**Requirement**: AC-014, REQ-F-014
**Test Location**: `internal/repository/search/repository_test.go` or compile-time coverage in affected tests
**Entrypoint**: production search path through `SearchService.SearchAll`
**Lowest Allowed Mock Seam**: repository interface in service tests
**Forbidden Mocks**: do not leave passing tests that exercise only deleted task-only helpers
**Counter-factual**: dead `Search`, `SearchWithSnippets`, or `IndexTask` methods could remain uncalled in production while tests falsely imply search is covered.

## ISO 25010 Coverage

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| AC-001 | TC-001 | TC-001 rank/order sanity | N/A | snippet readability in TC-001 | N/A | parameterized query in TC-001 | unified path asserted | FTS5 local |
| AC-002 | TC-002 | N/A | N/A | N/A | N/A | parameterized query | unified path asserted | FTS5 local |
| AC-003 | TC-003 | N/A | service/repo boundary | N/A | re-index behavior | N/A | hook seam covered | FTS5 local |
| AC-004 | TC-004 | SQL filter | type compatibility | N/A | N/A | N/A | avoids Go post-filter | FTS5 local |
| AC-005 | TC-005 | N/A | idea entity support | CLI type clarity | N/A | N/A | entity coverage | FTS5 local |
| AC-006 | TC-006 | N/A | tech_debt naming | CLI type clarity | N/A | N/A | entity coverage | FTS5 local |
| AC-007 | TC-007 | set-oriented tags | tag service contract | N/A | zero/empty sets | tag validation | no N+1 | FTS5 local |
| AC-008 | TC-008 | N/A | B014 behavior | error clarity | regression guard | tag validation | explicit sentinel path | N/A |
| AC-009 | TC-009 | N/A | environment support | error clarity | loud failure | N/A | no silent fallback | local build tag |
| AC-010 | TC-010 | N/A | service hooks | N/A | stale-row guard | N/A | production caller | FTS5 local |
| AC-011 | TC-011 | N/A | service hooks | N/A | stale-row guard | N/A | production caller | FTS5 local |
| AC-012 | TC-012 | rebuild sanity | all entities | N/A | recovery path | N/A | backfill coverage | FTS5 local |
| AC-013 | TC-013 | N/A | JSON consumers | text output | N/A | output escaping | CLI boundary | N/A |
| AC-014 | TC-014 | N/A | production caller | N/A | dead-code guard | N/A | removes duplicate path | N/A |

## Integration Scenarios

1. CLI to service to repository:
   - `shark search "query" --type=idea --tag=voice --json`
   - Verify parsed flags reach `SearchService.SearchAll` with production shape.
   - Verify result formatting includes rank and snippet.

2. Entity service write to search index:
   - Update a representative entity title through its production service.
   - Verify the index collaborator is called after successful persistence.
   - Verify failed persistence does not update the index.

3. Note service to parent index:
   - Add a note through `NoteService.AddNote`.
   - Verify parent entity is re-indexed and the note text is searchable.

4. Migration and rebuild:
   - Start from source rows and no unified search rows.
   - Run migration/backfill/rebuild.
   - Verify each supported entity type is searchable.

## Test Infrastructure

Existing patterns to follow:

- Repository DB tests: `internal/repository/search_all_test.go`,
  `internal/repository/search_repository_test.go`,
  `internal/repository/test-shark-tasks.db` helpers.
- Service mocks: `internal/services/search_service_test.go`,
  `internal/services/mock_tag_service_test.go`.
- CLI command tests: `internal/cli/commands/search_query_test.go`,
  `internal/cli/commands/list_tag_test.go`.
- DB migration tests: `internal/db/db_test.go`,
  `internal/db/*migration*_test.go`.

New helpers likely needed:

- Unified search fixture seeding all supported entity types.
- FTS5 capability assertion helper shared by repository tests.
- Search index collaborator mock for entity service and note service tests.
- Result assertion helper for rank/snippet fields.

## Observability Requirements

| Behavior | Evidence |
| --- | --- |
| Rebuild/backfill runs | structured log with entity counts by type and total indexed rows |
| Search query executes | trace span or debug log including entity type filter, tag count, result count, and duration |
| Index update fails | error log with entity type/key and operation |
| FTS5 unavailable | clear error returned from setup/init path; no warning-only behavior |

## Cross-Feature Contract Tests

No I-## interactions were declared in `spec.md`; no cross-feature contract test
is required.

## Cross-Epic Integration Tests

No X-## integrations were declared in `spec.md`; no cross-epic integration test
is required.

## Exit Gate

- Every acceptance criterion has at least one test case: complete.
- Every test case has a caller-path contract: complete.
- Edge cases identified for each AC: complete.
- Integration scenarios cover CLI, service, repository, migration, and note
  boundaries: complete.
- Existing test infrastructure referenced: complete.
