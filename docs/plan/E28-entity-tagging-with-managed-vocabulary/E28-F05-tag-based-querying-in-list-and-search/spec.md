---
feature_key: E28-F05-tag-based-querying-in-list-and-search
epic_key: E28
document_type: spec
title: E28-F05 Spec — Tag-Based Querying in List and Search
---

# E28-F05 Spec — Tag-Based Querying in List and Search

This specification covers **what** (requirements) and **how** (architecture) in
a single document, because F05 is brownfield wiring work that reuses fully
established patterns from F01–F04. It is INCREMENTAL — it assumes every
decision and facility made by F01 (schema), F02 (maintainer gate), F03
(vocabulary CRUD), and F04 (`AttachMany` / `DetachOne` / `EnforceRequired`,
per-entity `--tag` flag, `<entity> tag add|rm` subcommands, vocabulary-error
rendering) already exists.

**Parent documents (do NOT restate):**

- Epic PRD — `docs/plan/E28-entity-tagging-with-managed-vocabulary/epic.md`
  (§1 problem statement, §2 success criteria, §4 constraints, §6 UAT)
- Epic Architecture — `docs/plan/E28-entity-tagging-with-managed-vocabulary/architecture.md`
  (§2 ADRs, §4.3 list-integration pattern, §4.4 search-integration pattern,
  §4.5 viewer-integration pattern)
- Feature description — `./feature.md` (thin description, integration points)
- F04 Spec (the direct predecessor) — `../E28-F04-entity-tag-attachment-and-enforcement/spec.md`
  (especially §2.6 entity-service integration, §2.7 CLI changes, §2.8 wiring,
  §2.10 ADR-F04-1 "additive update semantics")

**Branch reference:** Line numbers below refer to the state of the
`E28-entity-tagging-with-managed-vocabulary` branch at commit `a267e87`
(F04 completion). Tasks that land between spec approval and implementation
MUST re-verify line numbers.

---

## 1. Requirements

All requirements are INCREMENTAL over the epic and over F01–F04. They trace
to the PRD Success Criteria (SC-n) and UAT scenarios (UAT-n) in the epic PRD.

### 1.1 Functional Requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-F-001 | `TagService` MUST expose `EntityIDsByTags(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)`. The method normalizes and validates every name via `TagService.ValidateName`, then returns the sorted deduplicated list of entity IDs (for the given `entityType`) whose `entity_tags` rows satisfy the AND-intersection of the supplied tag names. Empty `names` returns `(nil, nil)` (no filter — the caller interprets this as "do not apply a tag filter", not "return everything"). Unknown names produce `*UnregisteredTagError` (REQ-F-001 of F04 already defines this type). `TagQueryOp` is a typed string with the single value `TagQueryOpAnd = "and"` in v1; `op == ""` is treated as AND. | SC-1, UAT-1; Epic Architecture ADR-5 |
| REQ-F-002 | `EntityIDsByTags` MUST be backed by a new repository method `EntityTagRepository.FilterEntityIDs(ctx, entityType models.EntityType, tagIDs []int64) ([]int64, error)` that executes a single parameterized SQL statement of the shape in §2.3 (N EXISTS clauses joined with AND, one per tag ID). The statement MUST use `?` placeholders exclusively — no string interpolation of tag IDs or entity type. The method MUST NOT be called with an empty `tagIDs` slice (service-layer pre-check). | Epic Architecture ADR-5, `.claude/rules/go/input-sanitization.md` |
| REQ-F-003 | `EntityIDsByTags` MUST NOT consume the maintainer gate. Reading tagged entities is an open operation, consistent with F04 REQ-F-005 and the "LLM agents as operators" row in epic PRD §5. | Epic PRD §5; F04 REQ-F-005 |
| REQ-F-004 | `TagService` MUST expose `ListTagsForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error)` that returns the **normalized tag names** attached to `(entityType, entityID)`, sorted ascending. Uses `EntityTagRepositoryInterface.ListByEntity` and `TagRepositoryInterface.GetByID`. Returns a non-nil empty slice when no tags are attached. Does NOT consume the gate. | SC-1; Epic Architecture §4.3 "detail views" |
| REQ-F-005 | `TaskService.ListTasks`, `EpicService.ListEpics`, `FeatureService.ListFeatures`, `FeatureService.ListFeaturesByEpicKey`, `BugService.ListBugs`, `ChangeCardService.ListChangeCards`, and `IdeaService.ListIdeas` MUST each grow a new filter field `Tags []string` on their respective `*Filters` DTO (`TaskFilters`, `EpicFilters`, `FeatureFilters`, `BugFilters`, `ChangeCardFilter`, `IdeaFilter` — existing types). When `len(filters.Tags) == 0` the List behaves identically to F04; when `len(filters.Tags) > 0` the service calls `tagSvc.EntityIDsByTags(ctx, <entityType>, filters.Tags, TagQueryOpAnd)` and intersects the result set with the other filters via in-memory set membership over the entity ID. Ordering of existing filters (status, agent, etc.) relative to the tag filter is irrelevant because the tag filter is an additional conjunction (AND). | SC-1, UAT-1; Epic Architecture §4.3 |
| REQ-F-006 | When `tagSvc.EntityIDsByTags` returns a zero-length slice (tags are registered but no entity of that type has the full AND-intersection), the owning List method MUST return an empty non-nil slice `[]*models.<Entity>{}` with no error. The CLI MUST render the existing "no results" branch (`cli.Info("No bugs found")`, etc.) — no new UX is introduced for this case. | SC-1 |
| REQ-F-007 | When `tagSvc.EntityIDsByTags` returns `*UnregisteredTagError`, the owning List method MUST propagate that error unchanged to the caller. The six list commands MUST render the SC-2 vocabulary-snippet error shape via `handleEntityServiceError` (`internal/cli/commands/tags_shared.go:80`) — the helper from F04 already supports `*UnregisteredTagError` and needs no changes for F05. | SC-2, UAT-2 |
| REQ-F-008 | Each of the seven entity List services (per REQ-F-005) MUST accept the tag filter via the existing `*Filters` DTO without changing method signatures. Specifically: no new arguments, no new exported methods, no service struct changes beyond the optional `tagSvc` dependency that F04 already added to five of the six entity services (`TaskService`, `FeatureService`, `EpicService`, `BugService`, `ChangeCardService`, `IdeaService`). `ChangeCardService` MUST gain the same optional `tagSvc TagAttacher` dependency now (F04 already added it; no new wiring is needed if the SetTagService pattern or constructor arg was added there — otherwise F05 adds it using the exact same pattern). | `.claude/rules/services/service-design.md` rule 3 "use DTOs for >3 parameters"; F04 REQ-F-018 |
| REQ-F-009 | `shark list` top-level dispatcher (`internal/cli/commands/list.go`) MUST accept a repeated `--tag <name>` flag. The flag is bound once at the `listCmd` level via `listCmd.Flags().StringSliceVar(&listTags, "tag", nil, "Filter by tag (repeatable; all tags must match, AND semantics).")` and forwarded to each delegated list runner (`runEpicListWithFlags`, `runFeatureListWithFlags`, `runTaskListWithFlags`, `runBugList`, `runChangeList`, `runIdeaList`, `runTdList`). The `tech_debt` branch MUST forward the flag even though tech-debt tag integration is NOT part of E28 scope — the forwarded value is always `nil` for tech-debt (see §1.4 out-of-scope). | SC-1; Epic Architecture §4.3 |
| REQ-F-010 | Each entity-specific `list` sub-command (`shark task list`, `shark feature list`, `shark epic list`, `shark bug list`, `shark change list`, `shark idea list`) MUST accept its own `--tag <name>` repeated flag with the same binding shape as REQ-F-009. When invoked directly the command MUST wire the slice into its respective `*Filters` DTO. When invoked indirectly via the top-level `shark list` dispatcher (REQ-F-009), the dispatcher sets the per-command flag via `cmd.Flags().Set("tag", ...)` identically to how it forwards `--status` and `--all` today. | SC-1 |
| REQ-F-011 | `shark search <query>` MUST accept the same repeated `--tag <name>` flag. The search service MUST be updated so that when `len(tags) > 0`, AFTER the existing `SearchAll` returns its `[]*repository.EntitySearchResult`, the service calls `tagSvc.EntityIDsByTags(ctx, et, tags, TagQueryOpAnd)` **once per entity type present in the results** (batched by type bucket to avoid N queries per result). The search service then filters its result slice, keeping only rows whose `(entity_type, entity_id)` is present in the matching ID set. The existing FTS pipeline (repository layer) is NOT modified. | SC-1, UAT-1; Epic Architecture §4.4 |
| REQ-F-012 | The `SearchRepository.SearchAll` result row shape (`EntitySearchResult`) currently carries `entity_type`, `key`, `title`, `status`, `severity` but NOT the entity's primary-key `id`. F05 MUST add an unexported-or-exported `ID int64` field to `EntitySearchResult` and populate it in every `UNION ALL` branch of the SQL (already selects from `epics`, `features`, `tasks`, `bugs`, `change_cards`; ideas is NOT currently covered by `SearchAll` — adding ideas to search is OUT OF SCOPE, see §1.4). A nil or missing ID is a bug; tests assert the ID is non-zero for every result. | REQ-F-011 |
| REQ-F-013 | `TagService` MUST expose `AttachedTagNamesByIDs(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)` returning a map from entity ID to its sorted list of attached tag names. Empty input returns a non-nil empty map. Used by entity services' "list with tags" and "get with tags" paths (REQ-F-014, REQ-F-015) to avoid N+1 tag lookups. Implementation issues one SQL call via a new repository method `EntityTagRepository.ListTagNamesByEntities(ctx, entityType, entityIDs []int64) ([]EntityIDTagName, error)` (§2.3). | REQ-F-014, REQ-F-015 |
| REQ-F-014 | Each of the six entity Get service methods (`TaskService.GetTask`, `FeatureService.GetFeature`, `EpicService.GetEpic`, `BugService.GetBug`, `ChangeCardService.GetChangeCard`, `IdeaService.GetIdea`) MUST be augmented such that the returned domain model's caller can obtain the tags attached to the entity WITHOUT a second round-trip coded inside each command. Implementation: each entity service gains a sibling method `GetXxxWithTags(ctx, key) (*models.<Entity>, []string, error)` that calls the existing `GetXxx` then `tagSvc.ListTagsForEntity(ctx, entityType, entity.ID)`. The original `GetXxx` signatures are UNCHANGED (so legacy callers and unit tests do not break); `GetXxxWithTags` returns an extra `[]string` alongside the entity. When `tagSvc` is nil, `GetXxxWithTags` returns `(entity, nil, nil)` (graceful degradation consistent with F04 REQ-F-018). | SC-1; feature.md §"Tags field populated on Get responses" |
| REQ-F-015 | The six CLI `shark <entity> get` commands MUST switch to the new `GetXxxWithTags` accessor and include the tag-name list in their display output (both JSON via a `"tags"` array field on the response envelope, and human-readable via a `Tags: voice, auth` line in the rich display). JSON envelope: `{ "<entity_type>": {...existing fields...}, "tags": ["voice","auth"], ... }` — the `tags` field is ALWAYS present (empty array when no tags), per epic architecture §4.5 "clients never have to null-check". | SC-1; Epic Architecture §4.5 |
| REQ-F-016 | When either `--tag` on `shark list`/`shark search` OR the tag-filter path in any list service returns `*UnregisteredTagError` (unregistered tag supplied by the user), the CLI MUST exit with the same non-zero exit code used by F04 AC-21 (`exit code 3`). The vocabulary snippet and `To add it: shark tags add <name>` remediation MUST appear on stderr via `handleEntityServiceError`. | SC-2, UAT-2; F04 REQ-F-015, REQ-F-016 |
| REQ-F-017 | List service performance: when `len(filters.Tags) > 0`, the service MUST first compute the tagged-ID set via one `EntityIDsByTags` call, then short-circuit the base List query if the tagged-ID set is empty (zero IDs → return `[]` without hitting the base List repository). When non-empty, base List runs unchanged and the result is filtered by set membership in memory. Rationale: the tagged-ID set is expected to be small relative to the full entity population for useful cross-cutting queries. | `.claude/rules/services/service-design.md` §orchestration |
| REQ-F-018 | `list --tag` and `search --tag` CLI flag semantics MUST document AND conjunction explicitly in the flag description string and in `docs/cli-reference/tags.md`. The help-text string MUST read (verbatim in v1): `"Filter by tag (repeatable; AND — all tags must match)."`. | SC-1; Epic Architecture ADR-5 |
| REQ-F-019 | When the top-level `shark list` dispatcher resolves a non-recognized first positional argument (not an epic key, not `idea`/`ideas`/`bug`/`bugs`/`change`/`changes`/`tech_debt`), the `--tag` flag MUST NOT be silently dropped. The existing `ParseListArgs` failure path already returns an error; `--tag` is simply never forwarded in that path (no new behavior required). Adding a tag filter to an invalid list target is a user error surfaced by the existing dispatch error. | SC-1 |
| REQ-F-020 | No new migration, no `CurrentSchemaVersion` bump. The F01 schema (`tags`, `entity_tags`, the three `entity_tags` indexes) is sufficient. The `(tag_id, entity_type)` composite index (`idx_entity_tags_tag_entity`) already in F01 is what makes the AND-intersection query planar. | F04 REQ-F-NF; F01 schema |

### 1.2 Non-Functional Requirements

| ID | Requirement | Notes |
|---|---|---|
| REQ-NF-001 | **Performance — list path.** For N supplied `--tag` values and T total entities of the given type, `EntityIDsByTags` issues ONE SQL statement with N `EXISTS` sub-clauses. The `(tag_id, entity_type)` composite index (F01) keeps each `EXISTS` an O(log M) index seek where M is the number of `entity_tags` rows for that tag. Total filter cost: O(N · log M). In-memory set membership over T entities is O(T) with a hash set. Expected N ≤ 3 and M ≤ T. No additional per-entity round-trips. | Epic Architecture ADR-5 |
| REQ-NF-002 | **Performance — search path.** `SearchAll` is unchanged. Tag post-filtering runs one `EntityIDsByTags` call PER entity type present in the result set (at most 5: `epic`/`feature`/`task`/`bug`/`change` — ideas currently not in SearchAll). Worst case: 5 extra SQL statements regardless of result size. | REQ-F-011 |
| REQ-NF-003 | **Observability.** All three new `TagService` methods (`EntityIDsByTags`, `ListTagsForEntity`, `AttachedTagNamesByIDs`) MUST emit an OTel span using the existing `getTracer()` pattern. Span names: `tag_service.entity_ids_by_tags`, `tag_service.list_tags_for_entity`, `tag_service.attached_tag_names_by_ids`. Recorded attributes: `entity.type`, `tag.count` (for the first), `entity.id` (for the second), `entity.count` (for the third), `filter.op` (for the first, always `"and"` in v1). No vocabulary contents or raw inputs are attributed. | Epic Architecture §6; F04 REQ-NF-002 |
| REQ-NF-004 | **Input sanitization.** All tag names coming from `--tag` flags MUST pass through `TagService.ValidateName` before reaching any repository. Entity IDs are always numeric post-lookup IDs (never raw user strings). Repository SQL uses `?` parameterized placeholders exclusively. | `.claude/rules/go/input-sanitization.md` |
| REQ-NF-005 | **Concurrency.** A concurrent `shark tags rm --force <name>` between the `EntityIDsByTags` pre-filter and the base List call can cause the filtered ID set to reference entities whose tag attachment has just been removed. F05 accepts this as a benign race (operational, not user-facing): the resulting list may include entities that no longer have the tag at wall-clock read time. No locking or transaction is introduced. Consistent with F04 REQ-NF-004. | F04 REQ-NF-004 |
| REQ-NF-006 | **Testing.** Repository tests for `FilterEntityIDs` and `ListTagNamesByEntities` use the real DB with cleanup (per `.claude/rules/testing/repository-tests.md`). Service tests for `EntityIDsByTags`, `ListTagsForEntity`, `AttachedTagNamesByIDs`, and the seven entity List methods use the existing mocks (`MockTagService` from F04, `MockTagRepository` / `MockEntityTagRepository` to be added). CLI tests mock the services. No real DB in service or CLI tests. | `.claude/rules/testing/architecture.md` |
| REQ-NF-007 | **Dual backend.** All SQL uses the same feature set as F01 (`EXISTS`, parameterized placeholders, composite indexes). No SQLite-only or Turso-only features. | Epic PRD §4 constraint 5 |
| REQ-NF-008 | **Backward compatibility.** Databases at F01 schema version (14) require NO migration for F05. Commands without `--tag` behave identically to pre-F05. Commands without tag enforcement see no behavioral change. | F04 REQ-NF-007 |
| REQ-NF-009 | **No new CLI short flag.** `--tag` has no `-t` shorthand in v1 (reserves `-t` for future use; matches F04 REQ-F-012). | F04 REQ-F-012 |
| REQ-NF-010 | **No regression in existing list/search tests.** Every pre-F05 test for the seven entity list methods and the `SearchAll` service MUST pass unmodified after F05 changes, except for two allowed modifications: (a) `EntitySearchResult` struct gains an `ID` field (REQ-F-012) — tests that compare the struct by value need the field added; (b) list-service constructor tests that now receive an optional `tagSvc` argument may need to pass `nil`. No other test rewrites are permitted. | Brownfield safety |

### 1.3 Acceptance Criteria

Each criterion is a testable statement. IDs map into the test plan.

| ID | Acceptance Criterion | Test kind |
|---|---|---|
| AC-1 | `TagService.EntityIDsByTags(ctx, EntityTypeTask, []string{"voice"}, TagQueryOpAnd)` with 3 tasks tagged `voice` and 5 untagged returns the 3 matching IDs in sorted ascending order. | Unit (mock repo) |
| AC-2 | `TagService.EntityIDsByTags(ctx, EntityTypeTask, []string{"voice", "auth"}, TagQueryOpAnd)` with 3 tasks tagged `voice`, 4 tagged `auth`, and 1 tagged both returns exactly the 1 intersection ID. | Unit (mock repo) |
| AC-3 | `TagService.EntityIDsByTags(ctx, EntityTypeTask, nil, TagQueryOpAnd)` returns `(nil, nil)`. `EntityIDsByTags(ctx, ..., []string{}, ...)` returns `(nil, nil)`. | Unit |
| AC-4 | `TagService.EntityIDsByTags(ctx, EntityTypeTask, []string{"voice", "does-not-exist"}, TagQueryOpAnd)` returns `(nil, *UnregisteredTagError{Name: "does-not-exist"})` and issues ZERO repository filter calls (fails on name resolution before `FilterEntityIDs`). | Unit |
| AC-5 | `TagService.EntityIDsByTags(ctx, EntityTypeTask, []string{"Voice "}, TagQueryOpAnd)` (untrimmed, mixed case) normalizes to `voice` and behaves identically to AC-1 with one tag. | Unit |
| AC-6 | `TagService.EntityIDsByTags(ctx, EntityTypeTask, []string{"voice", "voice"}, TagQueryOpAnd)` (duplicate names) deduplicates and issues exactly one `FilterEntityIDs` call with a single-element `tagIDs` slice. | Unit |
| AC-7 | `EntityTagRepository.FilterEntityIDs(ctx, EntityTypeTask, []int64{1, 2})` against a seeded DB returns only entity IDs whose `entity_tags` rows include BOTH `tag_id=1` AND `tag_id=2`. | Integration (real DB, repository test) |
| AC-8 | `EntityTagRepository.FilterEntityIDs(ctx, EntityTypeTask, []int64{})` panics or returns `ErrEmptyTagIDs` (explicit repository-layer precondition failure — the service layer MUST never call with empty `tagIDs`). | Unit (documents the invariant) |
| AC-9 | `TagService.ListTagsForEntity(ctx, EntityTypeTask, 42)` with two tags attached (`auth` id=7, `voice` id=3) returns `["auth", "voice"]` (sorted ascending). | Unit |
| AC-10 | `TagService.AttachedTagNamesByIDs(ctx, EntityTypeTask, []int64{10, 20, 30})` with attachments `(10,voice)`, `(10,auth)`, `(20,voice)`, `(30)` returns `{10:["auth","voice"], 20:["voice"], 30:[]}`. The map MUST contain an entry for every input ID (including ones with zero attachments). | Unit |
| AC-11 | `TaskService.ListTasks(ctx, TaskFilters{Tags: []string{"voice"}})` with `voice` registered and attached to tasks `[E07-F01-001, E07-F01-002]` returns exactly those two tasks, filters applied via `EntityIDsByTags` ∩ base-list result. | Unit (mock repos + mock tagSvc) |
| AC-12 | `TaskService.ListTasks(ctx, TaskFilters{Tags: []string{"voice"}, Status: "in_progress"})` intersects both filters: returns tasks that are BOTH tagged `voice` AND in status `in_progress`. | Unit |
| AC-13 | `TaskService.ListTasks(ctx, TaskFilters{Tags: []string{"voice", "auth"}})` applies AND intersection of both tags (relies on `EntityIDsByTags` AC-2). | Unit |
| AC-14 | `TaskService.ListTasks(ctx, TaskFilters{Tags: []string{"voice"}})` when `voice` is registered but NO task is tagged `voice` returns `[]*models.Task{}` (non-nil empty slice) with no error. | Unit |
| AC-15 | `TaskService.ListTasks(ctx, TaskFilters{Tags: []string{"does-not-exist"}})` returns `*UnregisteredTagError` from the service (base List is not called). | Unit |
| AC-16 | For each of the seven entity List services (Task, Epic, Feature via `ListFeatures` and `ListFeaturesByEpicKey`, Bug, ChangeCard, Idea) — an analogue of AC-11 passes. | Unit (each entity) |
| AC-17 | `SearchService.SearchAll(ctx, "login", "")` with three results (`[task:E07-F01-001 login, task:E07-F01-002 fix login, bug:B001 login broken]`) and `TagFilters: []string{"voice"}` attached only to `E07-F01-001` returns exactly `[task:E07-F01-001]`. | Unit (mock search repo + mock tagSvc) |
| AC-18 | `SearchService.SearchAll` with a tag filter that matches zero entities returns `[]` with no error. | Unit |
| AC-19 | `SearchService.SearchAll` with `tags=[does-not-exist]` returns `*UnregisteredTagError`. | Unit |
| AC-20 | `TaskService.GetTaskWithTags(ctx, "E07-F01-001")` with two attachments returns `(*Task, []string{"auth","voice"}, nil)`. With zero attachments returns `(*Task, []string{}, nil)`. With `tagSvc==nil` returns `(*Task, nil, nil)`. | Unit (each of 6 entities) |
| AC-21 | `shark list --tag=voice` (top-level dispatcher, no positional args — lists epics tagged `voice`) exits 0 and prints only epics tagged `voice`. | Integration (real CLI, in-mem DB) |
| AC-22 | `shark list E07 --tag=voice` lists only features of epic `E07` that are tagged `voice`. | Integration |
| AC-23 | `shark list E07 F01 --tag=voice` lists only tasks of feature `E07-F01` that are tagged `voice`. | Integration |
| AC-24 | `shark task list --tag=voice --tag=auth` (direct entity list, two tags, AND) returns only tasks tagged with BOTH. | Integration |
| AC-25 | `shark bug list --tag=voice`, `shark change list --tag=voice`, `shark idea list --tag=voice`, `shark feature list E07 --tag=voice`, `shark epic list --tag=voice` — each returns only entities of the given type tagged `voice`. | Integration (5 cases) |
| AC-26 | `shark search "login" --tag=voice` returns only search results tagged `voice`. | Integration |
| AC-27 | `shark list --tag=does-not-exist` exits with code 3, stderr lists the vocabulary, and stderr ends with `To add it: shark tags add does-not-exist` (reuses F04 helper). | Integration |
| AC-28 | `shark task get E07-F01-001` with tags attached renders `Tags: auth, voice` in the rich display. `--json` output contains `"tags": ["auth","voice"]`. Missing tags render `Tags: (none)` in rich mode and `"tags": []` in JSON. | Integration (each of 6 entities) |
| AC-29 | `shark list` (no `--tag` flag) has zero extra SQL statements compared to pre-F05. Verified by spanning an OTel test recorder and asserting span count equality. | Integration (regression) |
| AC-30 | Constructing any of the seven entity services with a nil `tagSvc` still permits `ListXxx(ctx, <filters with Tags: nil>)` to succeed (graceful degradation per F04 REQ-F-018). When `tagSvc` is nil AND `filters.Tags` is non-empty, the list method MUST return a new typed error `*TagFilterUnavailableError` with message `tag filtering is not available (TagService not wired)`. | Unit (each entity) |
| AC-31 | UAT-1 end-to-end: register `voice`, create one of each of the six entity types with `--tag=voice`, run `shark list --tag=voice` at each appropriate scope, and `shark search "" --tag=voice` — the outputs combined cover all six entities. (Search with empty query returns `[]` per existing behavior; the UAT scenario uses the entity-specific list commands to verify coverage, not search.) | UAT |

### 1.4 Out of Scope for F05

- **No new vocabulary-management operations.** `shark tags add/rm/rename`
  are F03 territory; F05 does not change any gated operation.
- **No `--tag-op=or`** or other multi-tag semantics. AND is the only
  supported operator in v1 (Epic Architecture ADR-5). `TagQueryOp` is a
  typed string so ADR-5's "v2 may add OR" path is cheap, but no OR logic
  is shipped here.
- **No tag-filtering on `shark progress`, `shark status`, `shark analytics`,
  `shark list tech_debt`.** Epic PRD §3 explicitly defers dashboards and
  tech-debt tag integration. The dispatcher MUST forward `--tag` to these
  branches as nil (see REQ-F-009).
- **No FTS rework.** `shark search` continues to use LIKE-based UNION query;
  tag filtering is a post-filter. Any FTS5 rework is a separate feature.
- **No ideas in cross-entity search.** `SearchAll` today does not cover
  ideas; F05 does not add them. The per-entity `shark idea list --tag=`
  still works. Adding ideas to `SearchAll` is out of scope (may be a future
  task).
- **No new viewer API / UI changes.** F06 consumes the new `TagService`
  methods (`EntityIDsByTags`, `ListTagsForEntity`, `AttachedTagNamesByIDs`);
  those consumers are F06's work, not F05's.
- **No tag-based sorting** (e.g. `--sort-by=tag-count`). Tag filtering
  composes with existing `--sort-by`; no new sort keys.
- **No `--no-tag`, `--without-tag`, or negation.** Users who need the
  complement should run `shark list` without `--tag` and post-filter
  externally. This is a documented limitation.
- **No bulk list export with tags column.** The `--json` shape adds a
  `tags` field per entity on `get`, but list responses do NOT grow a `tags`
  array per row in v1 (hot path; inflates payload size). Adding it would
  require `AttachedTagNamesByIDs` on every list call. Deferred pending
  demonstrated need.
- **No new CLI command groups.** Every flag and behavior attaches to
  existing commands.
- **No new migration, no schema version bump.**

---

## 2. Architecture

Every section below follows existing project patterns; deviations are
explicitly justified. File paths are absolute within the repo; line numbers
refer to the state of the `E28-entity-tagging-with-managed-vocabulary`
branch as of commit `a267e87` (F04 completion).

### 2.1 Component Overview

F05 adds **zero new packages**, **zero new database tables**, **zero new
migrations**, and **zero new CLI commands**. It adds:

- 3 new methods to `TagService` (`EntityIDsByTags`, `ListTagsForEntity`,
  `AttachedTagNamesByIDs`) plus 6 new `GetXxxWithTags` wrappers on the six
  entity services.
- 2 new methods to `EntityTagRepository` (`FilterEntityIDs`,
  `ListTagNamesByEntities`).
- 1 new field (`Tags []string`) on each of 7 `*Filters` DTOs.
- 1 new field (`ID int64`) on `EntitySearchResult`.
- 1 new typed error (`TagFilterUnavailableError`) in
  `internal/services/tag_errors.go`.
- 1 new typed string (`TagQueryOp` + constant `TagQueryOpAnd`) in
  `internal/services/tag_dto.go` or new `internal/services/tag_query.go`.
- `--tag` flag added to `shark list`, `shark search`, and the six
  entity-specific `list` commands.
- Edits to each of the seven List service methods to consume the tag filter.
- Edits to each of the six Get command paths to call `GetXxxWithTags` and
  render the tag slice.
- Edits to `SearchService.SearchAll` to post-filter by tags.

```
CLI (commands/list.go, commands/search.go, commands/<entity>.go)
    │                    │
    ▼                    ▼
  --tag flag       --tag flag
    │                    │
    ▼                    ▼
List service     SearchService
    │                    │
    │ calls              │ calls after SearchRepository.SearchAll
    ▼                    ▼
TagService.EntityIDsByTags(ctx, entityType, names, AND)
    │
    ▼
TagRepository.GetByName (per name) → tag IDs
EntityTagRepository.FilterEntityIDs(ctx, entityType, tagIDs)
    │
    ▼
one SQL:  SELECT entity_id FROM entity_tags et ...
          WHERE entity_type = ? AND EXISTS(... tag_id=?) AND EXISTS(... tag_id=?) ...

CLI <entity> get
    │
    ▼
<Entity>Service.Get<Entity>WithTags(ctx, key)
    │ calls
    ▼
TagService.ListTagsForEntity(ctx, entityType, entityID)
    │
    ▼
EntityTagRepository.ListByEntity + TagRepository.GetByID (per attached tag)
```

### 2.2 Data Model Changes

**None to the database schema.** F01 already delivers:

- `entity_tags` with the `(entity_type, entity_id, tag_id)` shape
- `idx_entity_tags_tag_entity` composite `(tag_id, entity_type)` — makes
  the AND-intersection query planar (see §2.3 SQL plan rationale).
- `tags` with `name` unique (NOCASE).

No `CurrentSchemaVersion` bump. No new tables, columns, indexes, or
triggers.

### 2.3 Repository Layer — `internal/repository/tag/entity_tag_repository.go`

Two new methods, both of which follow the existing tracing/span pattern
(`entityTagTracer` + `repoutil.RecordSpanError`).

#### 2.3.1 `FilterEntityIDs`

**Signature (added to `EntityTagRepositoryInterface` in `internal/repository/tag/interfaces.go`):**

```go
// FilterEntityIDs returns the sorted, deduplicated list of entity_ids whose
// entity_tags rows include EVERY tag_id in tagIDs (AND intersection) for
// the given entity_type.
//
// Precondition: len(tagIDs) >= 1. A caller passing an empty slice is a
// programming error; the method returns ErrEmptyTagIDs (sentinel). The
// service layer enforces this precondition.
//
// SQL shape (ADR-5): one outer SELECT against entity_tags with N EXISTS
// sub-clauses joined by AND, one per tagID. Uses the composite index
// (tag_id, entity_type) introduced in F01 so each EXISTS is an index seek.
FilterEntityIDs(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error)
```

**SQL template (generated dynamically for N tag IDs):**

```sql
-- Outer query scoped to one entity type so the composite
-- (tag_id, entity_type) index can be used for each EXISTS subquery.
SELECT DISTINCT et0.entity_id
FROM entity_tags et0
WHERE et0.entity_type = ?
  AND et0.tag_id = ?                                  -- first tag, anchors the row set
  AND EXISTS (SELECT 1 FROM entity_tags et1
              WHERE et1.entity_type = ? AND et1.entity_id = et0.entity_id AND et1.tag_id = ?)
  AND EXISTS (SELECT 1 FROM entity_tags et2
              WHERE et2.entity_type = ? AND et2.entity_id = et0.entity_id AND et2.tag_id = ?)
  -- ... one EXISTS per additional tag ...
ORDER BY et0.entity_id ASC
```

For `N=1` the query reduces to `SELECT et.entity_id FROM entity_tags et
WHERE et.entity_type = ? AND et.tag_id = ? ORDER BY et.entity_id ASC` and
skips the DISTINCT (the UNIQUE constraint `(entity_type, entity_id, tag_id)`
guarantees uniqueness already). The implementation uses a `strings.Builder`
to assemble the query with a fixed-per-tag pattern, appending one `EXISTS`
clause per additional tag and one pair of `?` placeholders per clause. All
values go through placeholders — no string interpolation.

**Parameterized binding order** (for N tag IDs):

1. `entityType` (outer)
2. `tagIDs[0]` (outer anchor)
3. For each `i in [1, N)`:
   1. `entityType` (inner i)
   2. `tagIDs[i]` (inner i)

**Sentinel error:**

```go
// in internal/repository/tag/errors.go (or alongside existing errors)
var ErrEmptyTagIDs = errors.New("entity_tag repository: FilterEntityIDs requires at least one tagID")
```

#### 2.3.2 `ListTagNamesByEntities`

**Signature:**

```go
// EntityIDTagName is a flat result row for ListTagNamesByEntities.
type EntityIDTagName struct {
    EntityID int64
    TagName  string
}

// ListTagNamesByEntities returns (entity_id, tag_name) rows for every
// attachment on the given (entityType, entityIDs). Rows are ordered by
// entity_id ASC then tag_name ASC. Empty entityIDs returns an empty
// (non-nil) slice without touching the DB.
ListTagNamesByEntities(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]EntityIDTagName, error)
```

**SQL:**

```sql
SELECT et.entity_id, t.name
FROM entity_tags et
JOIN tags t ON t.id = et.tag_id
WHERE et.entity_type = ?
  AND et.entity_id IN (?, ?, ?, ...)
ORDER BY et.entity_id ASC, t.name ASC
```

Parameterized placeholder count = `1 + len(entityIDs)`. Implementation
generates the `?, ?, ?` list dynamically — same pattern as many existing
`In-clause` helpers in the codebase.

#### 2.3.3 Interface additions

Both methods are added to `EntityTagRepositoryInterface` in
`internal/repository/tag/interfaces.go`. Concrete implementation lives in
`internal/repository/tag/entity_tag_repository.go` alongside existing
methods. Tracer name and attribute patterns match existing (`db.system`,
`db.operation`, `db.table`, `entity.type`, `tag.count`).

### 2.4 Service Layer — `internal/services/tag_service.go`

Three new methods on `TagService`. One new supporting type.

#### 2.4.1 `TagQueryOp` — new typed string

New file `internal/services/tag_query.go` (or extend `tag_errors.go`):

```go
package services

// TagQueryOp is the multi-tag filter operator for EntityIDsByTags.
// Only "and" is supported in v1 (see Epic Architecture ADR-5).
type TagQueryOp string

const (
    TagQueryOpAnd TagQueryOp = "and"
)

// normalizeOp returns the canonical op. An empty string defaults to AND
// so callers (CLI, HTTP) can pass "" without special-casing.
func (o TagQueryOp) normalize() TagQueryOp {
    if o == "" {
        return TagQueryOpAnd
    }
    return o
}
```

#### 2.4.2 `EntityIDsByTags`

```go
// EntityIDsByTags returns the sorted, deduplicated list of entity IDs for
// entityType that satisfy the multi-tag AND intersection of names.
//
// Empty or nil `names` returns (nil, nil) — callers interpret this as "no
// tag filter", not "all entities". Any unknown name produces
// *UnregisteredTagError and the method issues no EntityTagRepository call.
// Duplicate names in the same call are deduplicated. Names are normalized
// via ValidateName.
func (s *TagService) EntityIDsByTags(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)
```

**Algorithm:**

1. If `len(names) == 0` → return `(nil, nil)`.
2. Start span `tag_service.entity_ids_by_tags` with attributes
   `entity.type`, `tag.count = len(names)`, `filter.op = string(op.normalize())`.
3. Normalize and deduplicate names via `ValidateName` + a `map[string]struct{}`.
4. For each unique normalized name, call `s.tagRepo.GetByName`. On
   `tagrepo.ErrTagNotFound` return `*UnregisteredTagError{Name: name}`.
   Collect tag IDs in input-order (order does not matter for AND but is
   deterministic for logging).
5. Call `s.entityTagRepo.FilterEntityIDs(ctx, entityType, tagIDs)`.
6. Return the result.

#### 2.4.3 `ListTagsForEntity`

```go
// ListTagsForEntity returns the sorted ascending list of normalized tag
// names attached to (entityType, entityID). Returns an empty non-nil
// slice when no tags are attached.
func (s *TagService) ListTagsForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error)
```

**Algorithm:**

1. Span `tag_service.list_tags_for_entity` with `entity.type`,
   `entity.id` attributes.
2. Call `s.entityTagRepo.ListByEntity(ctx, entityType, entityID)`.
3. For each link, call `s.tagRepo.GetByID(link.TagID)`; collect names.
   (This is N round-trips per entity; for the Get path N is typically
   ≤ 5 so no batching is introduced here. The list-with-tags path uses
   `AttachedTagNamesByIDs` instead — see REQ-F-013 and §2.4.4.)
4. Sort names ascending; return.

Alternative considered (rejected for v1): add a single JOIN'd repo method
`EntityTagRepository.ListTagNamesByEntity(ctx, entityType, entityID) ([]string, error)` to
avoid the N round-trips in step 3. Rejected because `AttachedTagNamesByIDs`
(§2.4.4) already provides the batched shape via
`ListTagNamesByEntities` and `ListTagsForEntity` is the Get-path helper
used for single entities — the N round-trips are acceptable for N ≤ 5 and
match the existing `AttachMany` loop's round-trip count.

#### 2.4.4 `AttachedTagNamesByIDs`

```go
// AttachedTagNamesByIDs returns a map from entityID to the sorted list of
// attached tag names. Every input ID appears in the map (even those with
// zero attachments, mapped to an empty non-nil slice). Empty input returns
// a non-nil empty map.
func (s *TagService) AttachedTagNamesByIDs(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)
```

**Algorithm:**

1. If `len(entityIDs) == 0` → return `(map[int64][]string{}, nil)`.
2. Span with `entity.type`, `entity.count = len(entityIDs)`.
3. Call `s.entityTagRepo.ListTagNamesByEntities(ctx, entityType, entityIDs)`.
4. Bucket rows into a `map[int64][]string`, accumulating names in order.
5. For every entityID in the input not already in the map, add an empty
   slice entry.
6. Sort each slice (the repository already returns ordered; defensive sort
   is cheap).
7. Return the map.

### 2.5 Entity Service Integration (seven List methods + six GetXxxWithTags)

#### 2.5.1 Filter DTO extension

Each of the following DTOs gains a `Tags []string` field with
`json:"tags,omitempty"`:

| DTO | File | Line |
|---|---|---|
| `TaskFilters` | `internal/services/task_dto.go` | 51 |
| `EpicFilters` | `internal/services/epic_dto.go` | 84 |
| `FeatureFilters` | `internal/services/epic_dto.go` | 120 |
| `BugFilters` | `internal/services/bug_dto.go` | 42 |
| `ChangeCardFilter` | `internal/services/change_dto.go` (find at `ChangeCardFilter` definition) | — |
| `IdeaFilter` / `repository.IdeaFilter` pass-through | `internal/services/idea_dto.go` | — |

For `ChangeCardFilter` and `IdeaFilter`: if the service layer currently
passes `*repository.XxxRepoFilter` directly, F05 either (a) introduces a
service-layer DTO `ChangeCardFilters` / `IdeaFilters` that wraps the
repo filter and adds `Tags []string`, or (b) adds `Tags []string` to the
existing DTO the service uses. The implementing task MUST pick (a) if the
repo filter is shared with other layers (HTTP); otherwise (b). Either way
the CLI-facing surface gains `Tags []string`.

#### 2.5.2 List method edits (uniform pattern)

Each affected List method (seven methods total) is edited to insert two
blocks:

**Block 1 — pre-filter, before the existing base-list query.** Runs only
when `len(filters.Tags) > 0`:

```go
// internal/services/task_service.go:553 (ListTasks, illustrative shape)
var taggedIDSet map[int64]struct{}
if len(filters.Tags) > 0 {
    if s.tagSvc == nil {
        return nil, &TagFilterUnavailableError{}
    }
    ids, err := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeTask, filters.Tags, TagQueryOpAnd)
    if err != nil {
        return nil, err
    }
    if len(ids) == 0 {
        return []*models.Task{}, nil // REQ-F-017 short-circuit
    }
    taggedIDSet = make(map[int64]struct{}, len(ids))
    for _, id := range ids {
        taggedIDSet[id] = struct{}{}
    }
}
```

**Block 2 — post-filter, after the base-list query returns.** Runs only
when `taggedIDSet != nil`:

```go
if taggedIDSet != nil {
    kept := tasks[:0]
    for _, t := range tasks {
        if _, ok := taggedIDSet[t.ID]; ok {
            kept = append(kept, t)
        }
    }
    tasks = kept
}
```

Both blocks are inserted via a single helper function to keep the seven
List methods uniform:

```go
// internal/services/tag_filter_helper.go (new file)

// filterByTagIDs applies an in-memory set-membership filter over ents by
// extracting each entity's ID via getID. Returns the filtered slice
// in-place (ents is reused). When taggedIDSet is nil, returns ents
// unchanged.
func filterByTagIDs[E any](ents []E, taggedIDSet map[int64]struct{}, getID func(E) int64) []E {
    if taggedIDSet == nil {
        return ents
    }
    kept := ents[:0]
    for _, e := range ents {
        if _, ok := taggedIDSet[getID(e)]; ok {
            kept = append(kept, e)
        }
    }
    return kept
}
```

(Go generics are supported in Go 1.23.4+ per project minimum.)

#### 2.5.3 `GetXxxWithTags` wrappers

Six new service methods, one per entity. Pattern (illustrated for task):

```go
// internal/services/task_service.go (append after GetTask at line 432)

// GetTaskWithTags returns the task and the sorted list of tag names
// attached to it. When tagSvc is nil, the tags slice is nil (graceful
// degradation — consistent with F04 REQ-F-018).
func (s *TaskService) GetTaskWithTags(ctx context.Context, key string) (*models.Task, []string, error) {
    ctx, span := s.getTracer().Start(ctx, "TaskService.GetTaskWithTags",
        trace.WithAttributes(attribute.String("task.key", key)))
    defer span.End()

    task, err := s.GetTask(ctx, key)
    if err != nil {
        return nil, nil, err
    }
    if s.tagSvc == nil {
        return task, nil, nil
    }
    names, err := s.tagSvc.ListTagsForEntity(ctx, models.EntityTypeTask, task.ID)
    if err != nil {
        return task, nil, recordSpanError(span, fmt.Errorf("load tags for task %s: %w", key, err))
    }
    return task, names, nil
}
```

The five other entity services grow the corresponding method
(`GetFeatureWithTags`, `GetEpicWithTags`, `GetBugWithTags`,
`GetChangeCardWithTags`, `GetIdeaWithTags`), each calling the existing
`Get<Entity>` then `ListTagsForEntity` with the appropriate
`models.EntityType<X>`.

**Interface extension:** the `TagAttacher` interface in
`internal/services/bug_service.go:50` gains a fourth method:

```go
type TagAttacher interface {
    EnforceRequired(ctx context.Context, entityType models.EntityType, names []string) error
    AttachMany(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error
    DetachOne(ctx context.Context, entityType models.EntityType, entityID int64, name string) error

    // New in F05 — every entity service needs it.
    ListTagsForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error)
}
```

`*TagService` satisfies this interface automatically (compile-time check
already exists: `var _ TagAttacher = (*TagService)(nil)` at
`bug_service.go:56`). Tests that implement `TagAttacher` via mocks need to
add the new method — `internal/services/mock_tag_service_test.go`
already exists and grows one method.

#### 2.5.4 `TagFilterUnavailableError`

New typed error in `internal/services/tag_errors.go`:

```go
// TagFilterUnavailableError indicates a tag filter was requested but the
// entity service was constructed without a TagService (tagSvc is nil).
// Exit code 3 (invalid state, matches F04 `UnregisteredTagError`).
type TagFilterUnavailableError struct{}

func (e *TagFilterUnavailableError) Error() string {
    return "tag filtering is not available (TagService not wired)"
}
```

Handled by the existing `tagsErrorCode` function in
`internal/cli/commands/tags.go` (add a case that maps to the same exit
code and JSON `code` as `UnregisteredTagError`).

### 2.6 Search Service — `internal/services/search_service.go`

#### 2.6.1 Signature change

```go
// SearchService provides cross-entity full-text search with optional tag
// post-filtering (AND intersection).
type SearchService struct {
    repo    SearchRepository
    tagSvc  TagAttacher // optional; nil disables --tag on search
}

// NewSearchService constructs a SearchService. tagSvc is optional; pass
// nil to disable tag filtering.
func NewSearchService(repo SearchRepository, tagSvc TagAttacher) *SearchService {
    return &SearchService{repo: repo, tagSvc: tagSvc}
}

// SearchAll searches across all entity types. Empty entityType means no
// type filter. When len(tags) > 0, results are post-filtered to only
// rows whose (entity_type, id) is in the AND-intersection of the given
// tag names. Empty tags is a no-op.
func (s *SearchService) SearchAll(ctx context.Context, query string, entityType string, tags []string) ([]*repository.EntitySearchResult, error)
```

`tags` is a new parameter. The existing two-argument call-site at
`internal/cli/commands/search.go:97` is updated to pass the parsed
`--tag` slice (or nil). No callers other than the CLI exist today.

#### 2.6.2 Algorithm

1. Existing: call `s.repo.SearchAll(ctx, query, entityTypePtr)` — returns
   `[]*EntitySearchResult`.
2. If `len(tags) == 0` → return results unchanged.
3. If `s.tagSvc == nil` → return `*TagFilterUnavailableError`.
4. Bucket results by `entity_type`:
   ```go
   byType := map[string][]int64{}
   for _, r := range results {
       byType[r.EntityType] = append(byType[r.EntityType], r.ID)
   }
   ```
5. For each distinct entity type in `byType`, call
   `s.tagSvc.EntityIDsByTags(ctx, models.EntityType(type), tags, TagQueryOpAnd)`
   to get the full tagged-ID set for that type. Build a per-type
   `map[int64]struct{}`. (Note: this is `EntityIDsByTags` over the whole
   type, not intersected with the search-returned subset — correct because
   the final filter is `result.ID ∈ taggedSet`.) On any error, propagate.
6. Walk results; keep rows whose `(EntityType, ID)` is in the corresponding
   `map[int64]struct{}`.
7. Return the filtered slice (non-nil, possibly empty).

#### 2.6.3 `EntitySearchResult.ID`

`internal/repository/search/repository.go:32` — add `ID int64
\`json:"id,omitempty"\`` to the struct, and add `id` to the SELECT column
list in each of the five UNION branches:

```sql
SELECT 'epic'    AS entity_type, id, key, title, status, '' AS severity FROM epics    WHERE ...
SELECT 'feature' AS entity_type, id, key, title, status, '' AS severity FROM features WHERE ...
SELECT 'task'    AS entity_type, id, key, title, status, '' AS severity FROM tasks    WHERE ...
SELECT 'bug'     AS entity_type, id, key, title, CAST(status AS TEXT), CAST(severity AS TEXT) FROM bugs WHERE ...
SELECT 'change'  AS entity_type, id, key, title, CAST(status AS TEXT), '' AS severity FROM change_cards WHERE ...
```

Row scan order becomes `entity_type, id, key, title, status, severity`.
The existing JSON tag `id,omitempty` means clients that do not use ID
pay no JSON-size cost when the value is zero; however F05 guarantees ID
is non-zero in every row so `omitempty` is effectively inert. The tag is
there for defensive compatibility only.

### 2.7 CLI Changes

#### 2.7.1 Top-level `shark list` dispatcher — `internal/cli/commands/list.go`

Add package-level var and flag registration in `init()` (line 32):

```go
var listTags []string

func init() {
    cli.RootCmd.AddCommand(listCmd)
    listCmd.Flags().String("status", "", "Filter by status")
    listCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")
    listCmd.Flags().Bool("show-all", false, "Show all items including completed")
    _ = listCmd.Flags().MarkDeprecated("show-all", "use --all instead")
    listCmd.Flags().Bool("all", false, "Show all items including completed")
    listCmd.Flags().Int("priority", 0, "Filter ideas by priority (1-10)")

    // F05 REQ-F-009: repeatable tag filter (AND semantics, ADR-5).
    listCmd.Flags().StringSliceVar(&listTags, "tag", nil,
        "Filter by tag (repeatable; AND — all tags must match).")
}
```

Forwarding in each dispatched branch (lines 63–101): the per-command
list flag now includes `--tag`, so the dispatcher sets it:

```go
// For each branch, before calling the delegated runner:
for _, t := range listTags {
    _ = <delegatedCmd>.Flags().Set("tag", t)
}
```

`StringSliceVar` accepts multiple `Set` calls and accumulates. Alternative
forwarding: `cmd.Flags().Lookup("tag").Value.Set(strings.Join(listTags, ","))` once. Either
works; the loop is clearer.

For the `tech_debt` branch, `tdListCmd` does NOT bind a `--tag` flag in
F05 (tech-debt not in scope). Forwarding is skipped for that branch.

The `runTaskListWithFlags`, `runFeatureListWithFlags`,
`runEpicListWithFlags` helpers (lines 110–143) gain a fourth argument
`tags []string` and pass it through to the per-command runner via
`cmd.Flags().Set("tag", ...)` as above, or by extending their signatures
if the implementing task prefers.

#### 2.7.2 Per-entity list commands

Each of the following files gains a package-level `<entity>ListTags
[]string` var and a `StringSliceVar` registration in `init()`:

| Command | File | Runner | Flag registration location |
|---|---|---|---|
| `shark task list` | `internal/cli/commands/task.go` | `runTaskList` @ 66 | in task.go's init() |
| `shark feature list` | `internal/cli/commands/feature.go` | `runFeatureList` @ 212 | feature.go init |
| `shark epic list` | `internal/cli/commands/epic.go` | `runEpicList` @ 187 | epic.go init |
| `shark bug list` | `internal/cli/commands/bug.go` | `runBugList` @ 347 | bug.go init |
| `shark change list` | `internal/cli/commands/change.go` | `runChangeList` @ 339 | change.go init |
| `shark idea list` | `internal/cli/commands/idea.go` | `runIdeaList` @ 281 | idea.go init |

Each runner is edited to:

1. Parse `tags, _ := cmd.Flags().GetStringSlice("tag")`.
2. Assign `filters.Tags = tags` on the service filter DTO before calling
   the List service.
3. Wrap the service call in the existing error path; `handleEntityServiceError`
   already renders `*UnregisteredTagError`.

Pattern (task):

```go
// internal/cli/commands/task.go:84
svc := cli.GetTaskService()
tags, _ := cmd.Flags().GetStringSlice("tag")
tasks, err := svc.ListTasks(cmd.Context(), services.TaskFilters{
    EpicKey: epicKey, FeatureKey: featureKey, Status: status,
    AgentType: agentType, ShowAll: showAll, Blocked: blocked,
    MinPriority: minPriority, MaxPriority: maxPriority,
    Tags: tags,
})
if err != nil {
    return handleEntityServiceError(cmd, cli.GetTagService(), err, "task", "")
}
```

#### 2.7.3 `shark search` — `internal/cli/commands/search.go`

Add a package-level `var searchTags []string` and in `init()` (line 189):

```go
searchCmd.Flags().StringSliceVar(&searchTags, "tag", nil,
    "Filter by tag (repeatable; AND — all tags must match).")
```

`runSearchQuery` (line 84) passes `searchTags` through:

```go
results, err := cli.GetSearchService().SearchAll(cmd.Context(), query, entityTypeFlag, searchTags)
if err != nil {
    return handleEntityServiceError(cmd, cli.GetTagService(), err, "search", "")
}
```

The file-search branch (`runSearchFile`) does NOT gain `--tag` — file
search is task-only and the caller can compose with `shark task list --tag`
after. File-search tag integration is out of scope per §1.4 (no bulk / no
advanced composition).

#### 2.7.4 `shark <entity> get` — tag rendering

Each of the six `get` commands switches from `GetXxx` to `GetXxxWithTags`
and adds tag rendering:

- **Task get (`runTaskGet` @ `task.go:106`):** `svc.GetTaskWithTags(ctx, taskKey)`.
  Pass the tag slice into `cli.GetDisplayService().ResolveTaskAction` (or
  a new `DisplayTaskOptions.Tags` field if that service aggregates
  display data). The rich output prints `Tags: voice, auth` below the
  existing status line (or `Tags: (none)` when empty). The JSON envelope
  adds `"tags": [...]`.
- **Feature get, Epic get, Bug get, Change get, Idea get:** same pattern
  against their respective `Get<Entity>WithTags` accessor. Each `get`
  command file updates one or two printing helpers to include the tag
  line. JSON envelope gains `"tags": []string`.

The display additions are small (~3 lines of print per command + JSON
envelope key). Exact formatting matches existing "Notes: N" / "Related docs: N"
lines in each entity's display helpers.

### 2.8 CLI Service Wiring

#### 2.8.1 `cli.GetSearchService()` — `internal/cli/services_global.go`

Updated to pass `GetTagService()` as the second argument:

```go
func GetSearchService() *services.SearchService {
    db, _ := GetDB(context.Background())
    searchRepo := search.NewSearchRepository(db)
    return services.NewSearchService(searchRepo, GetTagService())
}
```

#### 2.8.2 Entity services already wire `tagSvc`

`cli.GetTaskService()`, `GetFeatureService()`, `GetEpicService()`,
`GetBugService()`, `GetIdeaService()` already wire `GetTagService()` per
F04 (see grep hits at services_global.go:271, :451, :485). F05 verifies
the same is true for `ChangeCardService` — if not, F05 adds the wiring
identically.

#### 2.8.3 HTTP service wiring — `cmd/server/services.go`

`WireServices` already constructs `TagService`. F05 updates the
`SearchService` construction to pass `TagService` as the new second
argument. No other change.

### 2.9 Key Technical Decisions (F05-specific ADRs)

These sit beneath the epic-level ADR-1…ADR-10. Epic ADRs are authoritative;
F05 ADRs resolve F05-local questions only.

#### ADR-F05-1: Tag filter is in-memory set intersection, not SQL JOIN

**Decision.** After computing the tagged-ID set via `EntityIDsByTags`, the
list service performs the intersection with the base-list result in Go via
a `map[int64]struct{}` lookup. No SQL JOIN between the tag-filter query
and the base entity table is introduced.

**Alternatives considered.**
(a) Single SQL query per list path with a JOIN to `entity_tags` and a
    `HAVING COUNT(*) = N` clause to express AND-intersection.
(b) Add a `WHERE id IN (SELECT entity_id FROM entity_tags ...)` subquery
    to each of the seven list repository methods.
(c) In-memory set intersection as chosen.

**Rationale.**

- (a) duplicates the AND-intersection logic across seven list paths and
  entangles tag awareness with every entity's list SQL. That turns the
  list repositories into "smart repositories" contrary to
  `.claude/rules/services/service-design.md` §Anti-Patterns.
- (b) similarly modifies every list repository; the JOIN subquery is
  always present even for `--tag`-less calls (or we branch at the
  repository, adding complexity). Either path couples the repositories to
  the tag schema.
- (c) keeps list repositories completely unaware of tags. Tag filtering
  is composed at the service layer, which is where filter composition
  already lives. Memory cost: a map of int64 IDs — for realistic entity
  counts (< 100k tasks in a project) this is well under 1 MB. CPU cost:
  O(T) over the base-list result.

The short-circuit in REQ-F-017 ensures that when the tag filter reduces
the tagged-ID set to zero, we don't run the base list at all — the worst
case where tag filtering adds overhead without value is avoided.

**Cost.** For very large (M ≥ 100k) entity-type tables where the base
list returns ≥ 10k rows and the tag filter matches ≤ 100, fetching 10k
rows just to discard 9900 is wasteful. F05 accepts this — the project
target scale is far below those numbers, and if scale demands change, a
future optimization can move the filter into repository-level JOINs at
the cost of modifying each list repo. ADR-F05-1 is revisitable but not
in scope for v1.

#### ADR-F05-2: Search tag-filter computes full tagged-ID set per type (not intersected with search result IDs)

**Decision.** When `shark search --tag=voice` runs, the search service
calls `EntityIDsByTags(..., entityTypeX, tags, AND)` ONCE per entity type
in the search result. The returned ID set is the full set of
`entityTypeX`-IDs tagged with all of `tags`, not intersected with the
search result first.

**Alternatives considered.**
(a) Compute the `(type, id)` set of search results first, then ask
    `EntityIDsByTags` to restrict to that subset.
(b) Compute the tagged-ID set per type (possibly very large), then
    in-memory filter the search result against it.

**Rationale.** (a) requires a new repository method (`FilterEntityIDs`
variant that takes both `tagIDs` AND a restriction set). Too much new
surface for a first-shipping feature. (b) is the current
`EntityIDsByTags` signature; for realistic search result sizes (< 100
rows) and tagged-ID set sizes (< 1000 rows per tag per type), the
waste is negligible (a few hundred extra IDs moved through memory). If
future scale makes this wasteful, we add the restriction variant then.

#### ADR-F05-3: Domain models remain tag-free; tags live on DTOs and service return tuples

**Decision.** `models.Task`, `models.Feature`, etc. do NOT grow a `Tags
[]string` field. Tag exposure happens via the `GetXxxWithTags` wrapper
that returns `(*Entity, []string, error)` — the tag slice is a sibling
return, not a field on the model.

**Alternatives considered.**
(a) Add `Tags []string` to each of the six entity models.
(b) Return a wrapper struct `EntityWithTags{Entity *models.Task, Tags []string}`.
(c) Return a tuple `(*models.Task, []string, error)` as chosen.

**Rationale.**
- (a) couples the model layer to the tag feature. Every repo, every
  serialization, every test that constructs a `models.Task` has to worry
  about whether `Tags` is populated or not, invented silent defaults
  (nil vs []), and JSON round-trip behavior. Large blast radius for a
  read-only display concern.
- (b) creates a new type per entity (six new types) for no structural
  benefit over the tuple.
- (c) is the minimum-surface option. The domain model stays pure (used by
  creation, update, workflow, sync). The display path — which already
  composes multiple read sources (display data, notes, docs, context) —
  simply adds the tag slice to its composition. Matches the existing
  pattern at `task.go:123-133` where the `get` command already assembles
  `relatedDocs, blockedBy, blocks, deps, notes` as separate values.

The JSON envelope (REQ-F-015) adds `"tags": []string` at the envelope
level, not inside the entity's own JSON. This is identical to how
`notes` is already added at the envelope level in the `get` output
today.

#### ADR-F05-4: `Tags []string` on list-service filter DTOs (not a separate `TagQuery` struct)

**Decision.** Each list filter DTO gains a plain `Tags []string` field.
No wrapping struct like `TagQuery{Names []string, Op TagQueryOp}`.

**Alternatives considered.**
(a) `Tags TagQuery` where `TagQuery` carries `{Names []string, Op TagQueryOp}`
    to future-proof for OR/NOT.
(b) `Tags []string` as chosen.

**Rationale.** AND is the only supported op in v1 (Epic Architecture
ADR-5). Introducing `TagQuery` now exports complexity users cannot
actually use. When OR or NOT lands, the CLI gains a flag (`--tag-op`)
and the list filter gains a second field (`TagOp TagQueryOp`) — both
additive, zero-cost to existing callers. Until then, `[]string` is
the honest shape.

#### ADR-F05-5: `EntitySearchResult` gains `ID int64` (not a parallel lookup)

**Decision.** Add `ID int64` to `EntitySearchResult` and populate it in
the SearchRepository SQL UNION. Do not perform a second lookup per
result row to fetch IDs.

**Alternatives considered.**
(a) After `SearchAll`, for each result row, call the appropriate
    `GetByKey` to obtain the entity ID.
(b) Add `ID` to the row struct and fetch it in the UNION SQL.

**Rationale.** (a) is N extra round-trips per search result — an
obvious no. (b) is a one-column addition to an existing query with
zero cost change on the DB side. Adding the column future-proofs any
downstream consumer that wants to correlate search results with other
tables.

### 2.10 Documentation Changes

- **`docs/cli-reference/README.md`** — update command tables:
  - "Inspect" section: `shark list` row and `shark search` row document
    `--tag` flag.
  - "Advanced" section: each per-entity `list` row notes `--tag` is
    available.
- **`docs/cli-reference/tags.md`** — new section "Filtering by tag":
  - Explain `--tag` usage with examples across the six entity types.
  - State AND semantics explicitly (link to Epic Architecture ADR-5).
  - Document the unregistered-tag error shape (same shape as F04, reuse
    F04's example block).
  - Document the `shark <entity> get` tag-display behavior.
- **`docs/cli-reference/list-commands.md`, search-commands.md, task-commands.md,
  feature-commands.md, epic-commands.md, bug-commands.md, change-commands.md,
  idea-commands.md** — add `--tag` under the command's flag table.

### 2.11 Testing Additions

Per `.claude/rules/testing/architecture.md`:

#### Service tests (mocked repos)

- `internal/services/tag_service_test.go` — 10 new cases for
  `EntityIDsByTags`, `ListTagsForEntity`, `AttachedTagNamesByIDs`
  (happy path, empty input, unregistered name, duplicates, normalization,
  sort order, empty result).
- `internal/services/task_service_test.go` — new test suite
  `TestListTasks_TagFilter*` covering AC-11, AC-12, AC-13, AC-14, AC-15.
  Uses existing `MockTagService` (`mock_tag_service_test.go`).
- Analogous additions in `epic_service_test.go`, `feature_service_test.go`,
  `bug_service_test.go`, `change_card_service_test.go`,
  `idea_service_test.go`, `search_service_test.go`.

#### Repository tests (real DB)

- `internal/repository/tag/entity_tag_repository_test.go` — new tests for
  `FilterEntityIDs` (AC-7, AC-8) and `ListTagNamesByEntities` (AC-10).
  Seeds two entity types (task + bug) and a small matrix of tag
  attachments; asserts sorted-ID output and membership.

#### CLI tests (mocked services)

- `internal/cli/commands/list_test.go` — tag-flag forwarding across
  `list E07 --tag=x`, `list E07 F01 --tag=x`, `list --tag=x`.
- `internal/cli/commands/task_test.go`, `bug_test.go`, etc. — each adds
  a `--tag` on list test against the mocked service.
- `internal/cli/commands/search_query_test.go` — add tag-flag test.

#### Integration (end-to-end CLI with in-memory DB)

- `cmd/server/services_test.go` or a new CLI integration test file
  covers AC-21 through AC-28 end-to-end.
- `AC-29` (zero-regression SQL-call count) uses an OTel in-memory span
  recorder to assert that `shark list` without `--tag` emits the same
  span tree as before F05.

### 2.12 Files Modified / Created Summary

| Type | Path | Change |
|---|---|---|
| Modify | `internal/repository/tag/interfaces.go` | Add `FilterEntityIDs` and `ListTagNamesByEntities` to `EntityTagRepositoryInterface` |
| Modify | `internal/repository/tag/entity_tag_repository.go` | Implement the two new methods with tracing |
| Create | `internal/repository/tag/errors.go` or modify existing errors file | Add `ErrEmptyTagIDs` sentinel |
| Create | `internal/services/tag_query.go` | `TagQueryOp` type + `TagQueryOpAnd` constant |
| Modify | `internal/services/tag_service.go` | Add `EntityIDsByTags`, `ListTagsForEntity`, `AttachedTagNamesByIDs` |
| Modify | `internal/services/tag_errors.go` | Add `TagFilterUnavailableError` |
| Modify | `internal/services/bug_service.go` (interface) | Add `ListTagsForEntity` method to `TagAttacher` |
| Modify | `internal/services/task_service.go` | Add `GetTaskWithTags`; edit `ListTasks` to apply tag filter |
| Modify | `internal/services/feature_service.go` | Add `GetFeatureWithTags`; edit `ListFeatures` + `ListFeaturesByEpicKey` |
| Modify | `internal/services/epic_service.go` | Add `GetEpicWithTags`; edit `ListEpics` |
| Modify | `internal/services/bug_service.go` | Add `GetBugWithTags`; edit `ListBugs` |
| Modify | `internal/services/change_card_service.go` | Add `GetChangeCardWithTags`; edit `ListChangeCards` (verify `tagSvc` wiring first) |
| Modify | `internal/services/idea_service.go` | Add `GetIdeaWithTags`; edit `ListIdeas` |
| Modify | `internal/services/task_dto.go` | `Tags []string` on `TaskFilters` |
| Modify | `internal/services/epic_dto.go` | `Tags []string` on `EpicFilters`, `FeatureFilters` |
| Modify | `internal/services/bug_dto.go` | `Tags []string` on `BugFilters` |
| Modify | `internal/services/change_dto.go` | `Tags []string` on the change-card filter type |
| Modify | `internal/services/idea_dto.go` (or new wrapper) | `Tags []string` on the idea filter |
| Create | `internal/services/tag_filter_helper.go` | Shared `filterByTagIDs` generic helper |
| Modify | `internal/services/search_service.go` | New `tagSvc` field; `SearchAll` signature gains `tags []string` |
| Modify | `internal/repository/search/repository.go` | Add `ID int64` to `EntitySearchResult` and to each UNION branch |
| Modify | `internal/cli/commands/list.go` | `--tag` flag + forwarding to per-entity runners |
| Modify | `internal/cli/commands/search.go` | `--tag` flag + pass to `SearchAll` |
| Modify | `internal/cli/commands/task.go` | `--tag` flag on `taskListCmd`; update `runTaskList`; update `runTaskGet` to call `GetTaskWithTags` |
| Modify | `internal/cli/commands/feature.go` | same pattern |
| Modify | `internal/cli/commands/epic.go` | same pattern |
| Modify | `internal/cli/commands/bug.go` | same pattern |
| Modify | `internal/cli/commands/change.go` | same pattern |
| Modify | `internal/cli/commands/idea.go` | same pattern |
| Modify | `internal/cli/services_global.go` | Update `GetSearchService()` to pass `GetTagService()` |
| Modify | `cmd/server/services.go` | Update `SearchService` construction (2nd arg) |
| Modify | `internal/services/mock_tag_service_test.go` | Add `ListTagsForEntity` mock method |
| Modify | `internal/services/tag_service_test.go` | 10+ new test cases |
| Modify | `internal/services/task_service_test.go` + 5 others | Tag-filter list tests |
| Modify | `internal/services/search_service_test.go` | Tag-filter search tests |
| Modify | `internal/repository/tag/entity_tag_repository_test.go` | Real-DB tests for the 2 new methods |
| Modify | `internal/cli/commands/list_test.go` + 6 others | CLI tag-flag forwarding tests |
| Modify | `docs/cli-reference/README.md` | Flag table updates |
| Modify | `docs/cli-reference/tags.md` | "Filtering by tag" section |
| Modify | 7 per-command docs pages | `--tag` flag notes |

### 2.13 Integration with Existing Patterns (Rationale Cross-References)

- **Generic filter helper** — mirrors how existing list paths sort/filter
  in memory. Follows the stdlib-generics approach used elsewhere in the
  codebase (Go ≥ 1.23.4).
- **Optional `tagSvc` dependency on entity services** — identical to F04
  REQ-F-018. Nil degrades gracefully for tests and partial rollouts.
- **`--tag` CLI flag shape** — identical to F04 REQ-F-012: `StringSliceVar`,
  no `-t` short, nil when omitted.
- **Vocabulary-error rendering** — reuses F04's `handleEntityServiceError`
  and `handleVocabularyErrorWithSnippet` unchanged. F05 adds no new error
  rendering code for the `UnregisteredTagError` path.
- **OTel span naming and attribute set** — follows the existing
  `tagServiceTracerName = "shark/services/tag"` and the
  `repoutil.NewTracer("internal/repository/tag/entity_tag")` patterns.
- **Repository SQL style** — uses parameterized placeholders, dynamic
  IN-clauses, and tracing spans exactly as `EntityTagRepository.Attach`
  and `ListByEntity` do today.
- **Service layering** — list services orchestrate (tag pre-filter → base
  list → in-memory filter); repositories remain pure SQL; models unchanged.
  Matches `.claude/rules/architecture.md`.

---

## 3. Exit-Gate Self-Review

Against the exit gate stated in the ready_for_specification instructions:

- **Every requirement is testable.** Each `REQ-F-*` and `REQ-NF-*` maps
  to at least one `AC-*`; each `AC-*` is either Unit (mock-driven),
  Integration (real DB / real CLI), or UAT.
- **Every architecture decision references existing patterns or explains
  deviation.** ADR-F05-1 through ADR-F05-5 each name the alternatives and
  cite the project rule or epic ADR that governs the decision. §2.13
  explicitly cross-references every reused pattern.
- **File paths listed for all changes.** §2.12 lists the 30+ modified /
  created files with absolute package paths. Line numbers reference
  commit `a267e87`.
- **No TBDs in critical sections.** The only conditional language
  ("if that service aggregates display data", "if the repo filter is
  shared") appears in §2.5.1 and §2.7.4 and describes a binary choice
  the implementing task resolves by inspecting one line of code. Both
  choices are documented.

Open items for tasks (not spec-level TBDs):

- `ChangeCardService` may or may not already have the optional `tagSvc`
  dependency; the task that edits `ListChangeCards` verifies and adds it
  if missing (same code as F04 did for the other five services).
- The `FeatureFilters` type may be a shared service-layer DTO or may be
  per-call in `ListFeatures` / `ListFeaturesByEpicKey`; the task adds
  `Tags []string` wherever the filter lives.

Both are mechanical and do not require additional spec text.

---

*Last Updated*: 2026-04-24
