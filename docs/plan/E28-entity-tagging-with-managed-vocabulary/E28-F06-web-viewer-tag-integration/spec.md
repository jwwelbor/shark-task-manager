---
feature_key: E28-F06-web-viewer-tag-integration
epic_key: E28
document_type: spec
title: E28-F06 Spec — Web Viewer Tag Integration
---

# E28-F06 Spec — Web Viewer Tag Integration

This specification covers **what** (requirements) and **how** (architecture) in
a single document, because F06 is brownfield wiring work that reuses fully
established patterns from F01–F05. It is INCREMENTAL — it assumes every
decision and facility made by F01 (schema), F02 (maintainer gate), F03
(vocabulary CRUD), F04 (`AttachMany` / `DetachOne` / `EnforceRequired`,
per-entity `--tag` flag, `<entity> tag add|rm` subcommands,
vocabulary-error rendering), and F05 (`EntityIDsByTags`,
`ListTagsForEntity`, `AttachedTagNamesByIDs`, `GetXxxWithTags`,
`ListXxx` tag filter via `*Filters.Tags`) already exists.

**Parent documents (do NOT restate):**

- Epic PRD — `docs/plan/E28-entity-tagging-with-managed-vocabulary/epic.md`
  (§2 success criteria SC-1, SC-8; §3 scope "E27 web viewer integration";
  §6 UAT-7)
- Epic Architecture —
  `docs/plan/E28-entity-tagging-with-managed-vocabulary/architecture.md`
  (§4.2 viewer service wiring, §4.5 viewer API+UI integration, §5.5
  backward compatibility, ADR-5 AND semantics)
- Feature description — `./feature.md`
- F05 Spec (direct predecessor) —
  `../E28-F05-tag-based-querying-in-list-and-search/spec.md` (§1 for
  REQ-F-001 `EntityIDsByTags`, REQ-F-004 `ListTagsForEntity`, REQ-F-013
  `AttachedTagNamesByIDs`)

**Branch reference:** Line numbers below refer to the state of the
`E28-entity-tagging-with-managed-vocabulary` branch at commit `f985a7c`
(F05 completion). Tasks that land between spec approval and
implementation MUST re-verify line numbers.

---

## 1. Requirements

All requirements are INCREMENTAL over the epic and over F01–F05. They
trace to the PRD Success Criteria (SC-n) and UAT scenarios (UAT-n) in
the epic PRD.

### 1.1 Functional Requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-F-001 | The viewer service MUST expose a new method `Tags(ctx context.Context) (*TagsResponse, error)` in `internal/services/viewer_service.go` that returns `{"tags": [{"name": "<normalized-name>"}, ...]}` ordered by `name` ascending. The method delegates to `TagService.ListTags` (already implemented — F03) and reshapes each `*models.Tag` into a narrow DTO. No gate is consumed (reading vocabulary is an open operation — consistent with F03 and SC-8). When no tags are registered, `tags` MUST be a non-nil empty slice `[]`, never `null`. | SC-8, UAT-7; Epic Architecture §4.5 |
| REQ-F-002 | `ViewerHandler` MUST register a new route `GET /api/v1/viewer/tags` wrapped with `WithLocalCORS`, mounted alongside the existing seven routes in `ViewerHandler.RegisterRoutes` (`internal/api/viewer/handler.go:47`). The handler method `Tags(w http.ResponseWriter, r *http.Request)` follows the existing three-step pattern: no parameters to parse → call `h.svc.Tags(r.Context())` → respond 200 with JSON (`respondJSON`). On service error log with `slog.Error("viewer tags failed", "endpoint", "tags", "error", err)` and respond `500` with `"failed to load tags"`. | SC-8, UAT-7; `.claude/rules/services/http-integration.md` |
| REQ-F-003 | Every hierarchical entity response DTO returned by the viewer handlers (`Summary`, `Hierarchy`, `FeatureTasks`) MUST include a `tags []string` field that is ALWAYS present — `[]` when empty, never `null`. Specifically: `HierarchyEpic`, `HierarchyFeature`, `ViewerTask`, `FlatEntity` (used for bugs, change cards, ideas in `HierarchyResponse`). The field is populated by the viewer service using `tagSvc.AttachedTagNamesByIDs` (F05 REQ-F-013) batched per entity type to avoid N+1 queries. `ActivityRecord` is EXCLUDED from this requirement — recent-activity records are history events, not entity snapshots, and their shape does not change in F06. | SC-1, SC-8, UAT-7; Epic Architecture §4.5 |
| REQ-F-004 | `ViewerService.Hierarchy` MUST, after loading the full epic→feature→task tree plus the flat `Bugs`/`ChangeCards`/`Ideas` slices, issue EXACTLY FIVE `tagSvc.AttachedTagNamesByIDs` calls — one per entity type actually present (`epic`, `feature`, `task`, `bug`, `change`, `idea`; the `idea` call is issued only if `ideas` are included). Each call passes the full list of entity IDs for that type. The returned `map[int64][]string` is used to populate the `Tags` field on each DTO via O(1) lookup. No other approach (N+1 per-entity lookups, single query per slice) is permitted. | SC-8; REQ-NF-001; Epic Architecture §4.5 |
| REQ-F-005 | `ViewerService.FeatureTasks` MUST, after the existing pagination-and-filter pipeline produces the final `[]*ViewerTask`, issue ONE `tagSvc.AttachedTagNamesByIDs(ctx, EntityTypeTask, ids)` call for the IDs in the returned page only (not the pre-filter set). Tasks excluded by status/agent/blocked filters or by the pagination window do NOT contribute to the ID list. | SC-8; REQ-NF-001 |
| REQ-F-006 | `ViewerService.Summary` aggregate response does NOT carry per-entity `tags` fields (its DTO is counts-shaped, not entity-shaped). REQ-F-003's "every entity DTO" scope therefore excludes `SummaryEntityCounts`, `SummaryTaskCounts`, `SummaryBugCounts`. The `Summary` endpoint is unchanged in F06. | SC-8 (no contradiction: SC-8 says "any entity view displays tags"; Summary is a counts view, not an entity view) |
| REQ-F-007 | All viewer list-style handlers that are NOT a direct list-of-entities (`History`, `Notes`, `RelatedDocs`, `File`, `FolderFiles`, `RecentActivity`, `WorkflowMeta`) are UNCHANGED in F06. None of them return entity summaries; they return audit/notes/docs/filesystem/metadata shapes that do not carry tags in v1. | SC-8; feature.md §Thin Description |
| REQ-F-008 | `ViewerService.FeatureTasks` MUST accept a `Tags []string` field on its existing `FeatureTaskOptions` DTO (`internal/services/viewer_service.go:295`). When `len(opts.Tags) == 0`, behavior is identical to F05. When `len(opts.Tags) > 0`, the service calls `tagSvc.EntityIDsByTags(ctx, EntityTypeTask, opts.Tags, TagQueryOpAnd)` BEFORE the existing filters/pagination, and intersects the task-ID set with the feature-scoped task list via in-memory set membership. The tag filter is applied strictly before pagination so `Total` reflects the tag-filtered-but-pre-pagination count. | SC-1, SC-8, UAT-7; Epic Architecture ADR-5, §4.5 |
| REQ-F-009 | `ViewerHandler.FeatureTasks` (`handler.go:170`) MUST accept a repeatable `?tag=<name>` query parameter. Parsing: `r.URL.Query()["tag"]` returns `[]string`. Each element is trimmed; empty-after-trim elements are dropped; case-lowering happens inside the service (not the handler — consistent with F04/F05 where the service owns normalization). The slice is written to `opts.Tags` in the existing `FeatureTaskOptions` construction. No new handler-level validation of tag-name format (the service returns `*UnregisteredTagError` for unknown names; see REQ-F-011). | REQ-F-008; Epic Architecture §4.5; `.claude/rules/services/http-integration.md` |
| REQ-F-010 | `ViewerService.Hierarchy` MUST accept an optional `HierarchyOptions` DTO with a `Tags []string` field. When `len(opts.Tags) > 0`, the method post-filters the hierarchy tree so that ONLY epics, features, and tasks (hierarchical) with the AND-intersection tag set remain visible. Flat entities (`bugs`, `change_cards`, `ideas`) are independently filtered by the SAME tag set (one `EntityIDsByTags` call per flat-entity type — consistent with F05 REQ-F-005 semantics across entity types). An epic with zero matching features and zero matching direct tags is pruned from the result. A feature with zero matching tasks and zero matching direct tags is pruned. This "show only tagged subtree" behavior is needed for UAT-7 ("apply a tag filter on a list view"). | SC-1, SC-8, UAT-7; Epic Architecture §4.5 |
| REQ-F-011 | `ViewerHandler.Hierarchy` (`handler.go:87`) MUST accept a repeatable `?tag=<name>` query parameter, parsed identically to REQ-F-009, and forward it via `opts.Tags` to `h.svc.Hierarchy(ctx, opts)`. When the service returns `*services.UnregisteredTagError` (propagated from F05's `EntityIDsByTags`), the handler MUST respond `400 Bad Request` with the error's message and an `unregistered_tags` string-array field listing which names were unregistered — so the UI can surface "tag 'voice' is not in the vocabulary" without a second round-trip. Other errors produce `500 Internal Server Error` via the existing pattern. | SC-2, SC-8; F05 REQ-F-007, REQ-F-016 |
| REQ-F-012 | `ViewerHandler.FeatureTasks` MUST return `400 Bad Request` (with the same `unregistered_tags` shape as REQ-F-011) when `?tag=` contains unregistered names, via the same `*UnregisteredTagError` detection path. `404 Not Found` continues to apply to the feature-not-found case (`isNotFound(err)` check — unchanged). The two error cases are mutually exclusive: the feature lookup happens first; if the feature exists but tags are unregistered, 400 applies. | SC-2, SC-8 |
| REQ-F-013 | The viewer MUST NOT expose ANY mutation endpoint for the vocabulary. No POST, PUT, PATCH, or DELETE handler is registered for the `tags` resource. This is enforced by the absence of such registrations in `RegisterRoutes`; verified by an HTTP-level test that asserts non-GET methods return `405 Method Not Allowed` (or 404, depending on the ServeMux behavior — see §2.2). | SC-8 "vocabulary management stays CLI-only"; UAT-7 "no control to add, remove, or rename" |
| REQ-F-014 | The `ViewerService` is NOT injected with `MaintainerGate`. `WireServices` in `internal/viewer/server/wire.go` MUST NOT pass the gate to `NewViewerService`. Even the constructor signature MUST NOT accept one. This is a defense-in-depth invariant: the viewer has no path that needs the gate, and giving it one would make a future mistake (e.g., adding a write handler) tempting. | SC-9; feature.md §Thin Description; Epic Architecture §4.2 |
| REQ-F-015 | `ViewerService` MUST accept a `TagReader` interface (defined in the viewer service file) with the three methods it consumes: `ListTags(ctx) ([]*models.Tag, error)`, `EntityIDsByTags(ctx, EntityType, []string, TagQueryOp) ([]int64, error)`, `AttachedTagNamesByIDs(ctx, EntityType, []int64) (map[int64][]string, error)`. The constructor gains an optional setter `WithTagService(TagReader)` (matching the existing `WithEntityDocRepo`, `WithIdeaRepo`, etc. pattern). When not set, the service MUST degrade gracefully: `Tags()` returns `{"tags": []}`; all entity DTOs carry `tags: []`; the `Tags` filter field is silently ignored (logged at debug). This graceful-degradation invariant matches F04 REQ-F-018 and guarantees the viewer binary still boots if the config has no tag setup. | `.claude/rules/services/service-design.md` §5 "optional dependencies degrade gracefully"; F04 REQ-F-018 |
| REQ-F-016 | The UI (`internal/viewer/assets/viewer.html`) MUST render tag chips on every entity detail view that currently shows a header area. Specifically: epic detail, feature detail, task detail, bug detail, change-card detail, idea detail. Chip styling: a `.tag-chip` class with the same visual language as existing pills (see the existing `#header-pills` element pattern). When `tags` is an empty array, NO chips and NO label are rendered (not an empty "Tags:" label). The chip text is the normalized tag name (lowercase ASCII). | SC-8, UAT-7 |
| REQ-F-017 | The UI MUST add a tag filter control on every list view (hierarchy sidebar and any list-like surface that reads from `GET /api/v1/viewer/hierarchy` or `GET /api/v1/viewer/features/{key}/tasks`). Control shape: a multi-select chip list populated by `GET /api/v1/viewer/tags` on viewer boot (cached for the session). User-selected chips are appended to subsequent list fetches as `?tag=<name>` repeated params. The filter control is stateless w.r.t. URL (F06 does NOT introduce deep-linkable filter URLs — that's a future enhancement). | SC-8, UAT-7 |
| REQ-F-018 | The UI MUST NOT render any "add tag", "rename tag", or "remove tag" control. The `GET /api/v1/viewer/tags` response is read-only. If the UI needs to offer affordances to the user ("add a new tag"), the affordance MUST be a link/tooltip that points at the CLI command string (e.g., `shark tags add <name> --pass=...`), NOT a form. | SC-8 "vocabulary management stays CLI-only"; UAT-7 |
| REQ-F-019 | The UI MUST gracefully handle the "viewer not wired with TagService" case (REQ-F-015 graceful degradation). When `GET /api/v1/viewer/tags` returns `{"tags": []}`, the filter control renders the disabled empty state ("No tags registered yet"); chips on entity views simply don't render. No JavaScript errors are thrown. | REQ-F-015; UAT-7 |
| REQ-F-020 | No new migration, no `CurrentSchemaVersion` bump. The F01 schema (`tags`, `entity_tags`, indexes) and the F05 service APIs (`EntityIDsByTags`, `ListTagsForEntity`, `AttachedTagNamesByIDs`) are sufficient. The viewer service calls them as a read-only consumer. | F05 REQ-F-020 |

### 1.2 Non-Functional Requirements

| ID | Requirement | Notes |
|---|---|---|
| REQ-NF-001 | **Performance — Hierarchy endpoint.** `Hierarchy` MUST issue no more than 6 extra SQL statements regardless of tree size: one `AttachedTagNamesByIDs` call per entity type present (≤ 6 types). Each call uses the F05 `ListTagNamesByEntities` repository method (one `IN (...)` statement per call). The total extra cost for a 100-epic / 1000-feature / 10000-task tree is therefore bounded at 6 SQL statements, not 11100. | Epic Architecture §4.5 |
| REQ-NF-002 | **Performance — FeatureTasks endpoint.** `FeatureTasks` MUST issue at most 2 extra SQL statements: (a) zero or one `EntityIDsByTags` when `?tag=` is supplied, (b) one `AttachedTagNamesByIDs` to decorate the final returned page. The second call uses only the IDs that survived filtering+pagination. | REQ-F-005, REQ-F-008 |
| REQ-NF-003 | **Performance — Tags endpoint.** `GET /api/v1/viewer/tags` issues exactly ONE SQL statement (`TagService.ListTags` → `TagRepository.List`). Response size is proportional to vocabulary size. No pagination in v1 — vocabulary is expected to be O(100) names at most; a page-then-infinite-scroll is a future enhancement. | SC-8 |
| REQ-NF-004 | **Observability.** The new `ViewerService.Tags` method MUST emit an OTel span using the existing pattern (see `ViewerService.Summary` for the reference shape). Span name: `viewer_service.tags`. Attributes: `tag.count` (the number of rows returned). No tag names are attributed. The decoration work inside `Hierarchy` and `FeatureTasks` already runs under those methods' existing spans; no new spans are added for the decoration step. | F05 REQ-NF-003; Epic Architecture §6 |
| REQ-NF-005 | **Input sanitization.** The handler does NO validation of the `?tag=` value format beyond trimming whitespace and dropping empties (REQ-F-009). The service layer (`TagService.ValidateName` inside `EntityIDsByTags`) owns normalization and allowlist validation, consistent with F05 REQ-NF-004. Handler never string-interpolates query values into anything. | `.claude/rules/go/input-sanitization.md`; F05 REQ-NF-004 |
| REQ-NF-006 | **CORS.** The new `GET /api/v1/viewer/tags` route MUST be wrapped with `WithLocalCORS` (existing middleware at `internal/api/viewer/cors.go`). No bespoke CORS handling. | Existing viewer contract |
| REQ-NF-007 | **Dual backend.** All viewer-layer changes are additive DTOs and additive service method calls; no SQL is written at the viewer layer. F05's SQL (used by `EntityIDsByTags`, `AttachedTagNamesByIDs`) is already dual-backend; F06 inherits that property. | Epic PRD §4 constraint 5 |
| REQ-NF-008 | **Backward compatibility.** Viewer clients on the old HTML that don't know about `tags` fields MUST continue to work — the new field is additive. The UI change in `viewer.html` SHIPS in the same commit so that a mismatched client is not a common case. When the UI is running and the API still carries `tags`, existing rendering paths are NOT affected because the new elements are DOM-additive. | Epic Architecture §5.5; feature.md §Thin Description |
| REQ-NF-009 | **Testing.** Viewer service tests use the existing repository-mock pattern with `MockTagService` (create if missing, mirroring F05). `ViewerHandler` tests use `MockViewerServicer` (already exists at `internal/api/viewer/handler_test.go`). CLI is NOT touched in F06 — no CLI tests. No real DB outside repository tests. | `.claude/rules/testing/architecture.md` |
| REQ-NF-010 | **No new asset dependencies.** `viewer.html` is a single-file HTML + inline CSS + inline JS. F06 MUST NOT introduce any new external `<script>` or `<link>` dependencies. All chip rendering, filter control, and vocabulary fetch are written in vanilla JS added to the existing inline script. | Existing viewer contract (single self-contained HTML) |
| REQ-NF-011 | **Graceful degradation.** When `TagReader` is not wired (REQ-F-015), the server MUST NOT panic on any viewer endpoint, MUST NOT log errors per request, and MUST return the same response shape as the happy path with empty `tags` fields and an empty vocabulary. A debug-level log is emitted once at `ViewerService` construction time noting that the tag decoration is disabled. | REQ-F-015 |
| REQ-NF-012 | **Security — read-only invariant.** An automated test MUST assert that `POST`, `PUT`, `PATCH`, `DELETE` to `/api/v1/viewer/tags` and `/api/v1/viewer/tags/<name>` all produce `404 Not Found` or `405 Method Not Allowed` — no write endpoint exists. This test is a regression tripwire for REQ-F-013. | SC-8, UAT-7 |

### 1.3 Acceptance Criteria

| ID | Criterion | Trace |
|---|---|---|
| AC-01 | `curl -s http://localhost:<port>/api/v1/viewer/tags` with an empty vocabulary returns `200 OK` with body `{"tags":[]}`. | REQ-F-001 |
| AC-02 | After `shark tags add voice --pass=<p>` and `shark tags add auth --pass=<p>`, the same curl returns `{"tags":[{"name":"auth"},{"name":"voice"}]}` (alphabetical). | REQ-F-001 |
| AC-03 | `curl -X POST http://localhost:<port>/api/v1/viewer/tags -d '{"name":"foo"}'` returns 404 or 405 — no tag is created in the DB (`shark tags list` is unchanged). | REQ-F-013, REQ-NF-012 |
| AC-04 | `GET /api/v1/viewer/hierarchy` on a fresh project returns JSON where every `epics[*].tags`, `epics[*].features[*].tags`, `epics[*].features[*].tasks[*].tags`, `bugs[*].tags`, `change_cards[*].tags`, `ideas[*].tags` is an empty array `[]`, never `null`. | REQ-F-003 |
| AC-05 | After tagging one epic, one feature, one task, one bug, one change-card, and one idea with `voice`, the same hierarchy endpoint shows `"tags":["voice"]` on each of those six entities and `[]` on all others. | REQ-F-003, UAT-7 |
| AC-06 | `GET /api/v1/viewer/hierarchy?tag=voice` returns only the epics/features/tasks with `voice` in their tag set AND the flat bugs/change-cards/ideas with `voice`. Epics whose entire subtree is untagged are pruned. | REQ-F-010, UAT-7 |
| AC-07 | `GET /api/v1/viewer/hierarchy?tag=voice&tag=auth` returns only entities tagged with BOTH `voice` AND `auth` (AND semantics — ADR-5 of epic architecture). | REQ-F-010; Epic Architecture ADR-5 |
| AC-08 | `GET /api/v1/viewer/hierarchy?tag=does-not-exist` returns `400 Bad Request` with body `{"error":"Bad Request","message":"unregistered tag: does-not-exist","unregistered_tags":["does-not-exist"]}`. | REQ-F-011 |
| AC-09 | `GET /api/v1/viewer/features/E07-F01/tasks?tag=voice` returns only the tasks within E07-F01 that have `voice`. `Total` reflects the tag-filtered count before pagination. | REQ-F-005, REQ-F-008 |
| AC-10 | `GET /api/v1/viewer/features/E07-F01/tasks?tag=does-not-exist` returns 400 (NOT 404 when the feature exists). `GET /api/v1/viewer/features/E99-F99/tasks?tag=voice` returns 404 (feature lookup comes first). | REQ-F-012 |
| AC-11 | In the UI, opening an entity that has tags shows a horizontal row of `.tag-chip` elements containing the tag names; opening an untagged entity shows NO chips and NO "Tags:" label. | REQ-F-016 |
| AC-12 | In the UI, the tag filter control populates itself on first page load by calling `GET /api/v1/viewer/tags`; selecting a chip in the filter triggers a refetch of the currently visible list with `?tag=<name>` appended. | REQ-F-017 |
| AC-13 | No UI control anywhere offers "add tag", "rename tag", or "remove tag" (manual scan). The filter-control UI does not have a "+" button or a text input for arbitrary tag names. | REQ-F-018, UAT-7 |
| AC-14 | When the server boots with `TagReader` NOT wired (e.g. config load failure), `GET /api/v1/viewer/tags` returns `{"tags":[]}`, every entity DTO `tags` field is `[]`, and no 500 is emitted. | REQ-F-015, REQ-NF-011 |
| AC-15 | Server-startup-time grep of `internal/viewer/server/wire.go` shows that `MaintainerGate` is NOT passed into `NewViewerService`. | REQ-F-014 |
| AC-16 | A unit test asserts `GET /api/v1/viewer/hierarchy` issues at most 6 extra SQL statements per request (measured via a counting sqlmock or by span attributes). | REQ-NF-001 |
| AC-17 | Running `make fmt && make lint && make test` passes with F06 merged. Zero regressions in `internal/api/viewer/...`, `internal/services/viewer_service_test.go`, `internal/viewer/...`. | `.claude/rules/development-workflows.md` Quality Gate |

### 1.4 Out of Scope for F06

The following are explicitly DEFERRED (either to F07 documentation,
future epics, or dropped from v1):

- **Vocabulary management in the UI**: no add / rename / remove buttons
  or forms. Only CLI.
- **Deep-linkable filter URLs**: selecting tags does not update
  `window.location.search`. Bookmark/share of a tag-filtered view is
  future work.
- **Viewer support for tag descriptions or colors**: names only.
  Description/color fields do not exist in the DB (PRD O-5 deferred).
- **Recent activity tag filter**: `GET /api/v1/viewer/recent-activity`
  does NOT accept `?tag=` in F06. RecentActivity rows are history
  events, not entity snapshots.
- **Summary endpoint tag enrichment**: `GET /api/v1/viewer/summary`
  counts do not grow a `by_tag` breakdown in v1.
- **Tech-debt tag integration**: `tech_debt` is not in the six-entity
  set; its viewer surface (if any today) is untouched by F06.
- **Tag-aware viewer dashboards**: progress/analytics/status
  dashboards in the viewer do NOT gain tag filtering (epic PRD §3
  "out of scope").
- **Search endpoint tag parity in the viewer**: the viewer does not
  yet surface `shark search`; F06 does not add a search surface.
  Search-tag parity for the CLI is F05.
- **Websocket push of vocabulary changes**: clients refresh the
  vocabulary on page load only. A rename via CLI while the viewer is
  open shows the old name until reload (acceptable for v1).
- **History entries showing tag changes**: `entity_tags` writes are
  not recorded in `entity_history` in F04; this stays out of scope
  for F06.

---

## 2. Architecture

### 2.1 Component Changes

#### 2.1.1 Files created

None. F06 is pure additive wiring on existing files.

#### 2.1.2 Files modified

| File | Changes |
|---|---|
| `internal/services/viewer_service.go` | Add `TagReader` interface; add `TagsResponse`, `TagDTO` types; add `Tags []string` field to `HierarchyEpic`, `HierarchyFeature`, `ViewerTask`, `FlatEntity`; add `HierarchyOptions` DTO; add `Tags []string` field to `FeatureTaskOptions`; add `WithTagService(TagReader)` setter and `tagSvc TagReader` private field; new method `Tags(ctx) (*TagsResponse, error)`; change `Hierarchy(ctx)` signature to `Hierarchy(ctx, opts HierarchyOptions)` (all existing callers migrate — there is only one: the handler); add tag-decoration step inside `Hierarchy` and `FeatureTasks`; add tag-filter step inside `Hierarchy` (when `opts.Tags` non-empty) and `FeatureTasks` (when `opts.Tags` non-empty). |
| `internal/api/viewer/service.go` | Extend `ViewerServicer` interface with `Tags(ctx) (*services.TagsResponse, error)` and update the `Hierarchy` signature to accept `services.HierarchyOptions`. The compile-time assertion `var _ ViewerServicer = (*services.ViewerService)(nil)` forces the concrete type to match. |
| `internal/api/viewer/handler.go` | Add `Tags(w, r)` method; register `GET <prefix>/tags` in `RegisterRoutes`; extend `Hierarchy(w, r)` to parse `?tag=` and pass `HierarchyOptions`; extend `FeatureTasks(w, r)` to parse `?tag=` into `opts.Tags`; add helper `parseTagsQuery(r)` that returns `[]string`; add helper `respondUnregisteredTagError(w, err)` that emits the new shape with `unregistered_tags`. |
| `internal/api/viewer/handler_test.go` | Extend `MockViewerServicer` with `TagsFunc` and update `HierarchyFunc` signature; add tests for the new route (GET/POST/PUT/etc.), for `?tag=` parsing on `Hierarchy` and `FeatureTasks`, for 400/unregistered-tag handling, and for graceful degradation when `Tags()` returns the empty-vocabulary shape. |
| `internal/services/viewer_service_test.go` | Add tests for: `Tags` method (empty, populated, delegates to `TagReader.ListTags`); `Hierarchy` decoration (asserts exactly 6 max `AttachedTagNamesByIDs` calls via mock counter); `Hierarchy` filter (AND-intersect pruning); `FeatureTasks` decoration (ID list is the post-pagination page); `FeatureTasks` filter (applied before pagination); graceful degradation when `tagSvc` is nil (all DTOs carry `[]`, `Tags()` returns `{tags: []}`). |
| `internal/viewer/server/wire.go` | In `WireServices`, after the `tagSvc` is constructed (already done at line ~274), add `viewerService.WithTagService(tagSvc)`. Do NOT pass `MaintainerGate` to `NewViewerService` (REQ-F-014). |
| `internal/viewer/assets/viewer.html` | Inline CSS additions: `.tag-chip` class (color, padding, pill shape), `.tag-filter-chips` container, `.tag-filter-chips .selected` state. Inline JS additions: `loadVocabulary()` on boot; `renderTagChips(entity)` helper invoked from each entity detail-render path; `renderTagFilterControl()` mounted into the hierarchy sidebar header; `applyTagFilter(tags)` rebuilds query strings. Chip/filter markup uses existing theming tokens (`--accent`, `--border`, etc.). |

No new files. No deletions. No renames.

### 2.1.3 Packages unchanged

- `internal/services/tag_service.go` — all three consumed methods
  (`ListTags`, `EntityIDsByTags`, `AttachedTagNamesByIDs`) already
  exist (F03, F05). F06 consumes, does not modify.
- `internal/repository/tag/...` — no repository changes.
- `internal/auth/maintainer/...` — REQ-F-014 forbids the viewer from
  importing this package.
- `internal/db/db.go` — no schema change.
- `internal/config/config.go` — no config-field change.

### 2.2 Data Model Changes

None. Zero schema changes, zero migrations, zero
`CurrentSchemaVersion` bump. The F01 schema and the F05 service APIs
are sufficient.

The only data-shape changes are in the JSON wire format of the viewer
HTTP responses (additive `tags` fields plus a new response type for
`GET /api/v1/viewer/tags`). See §2.4.

### 2.3 API / Interface Contracts

#### 2.3.1 New service-layer types

```go
// In internal/services/viewer_service.go (near the other *Response types).

// TagDTO is the narrow projection of models.Tag that the viewer exposes.
// Names are normalized lowercase-ASCII (enforced at F03 add-time).
type TagDTO struct {
    Name string `json:"name"`
}

// TagsResponse is the response type for ViewerService.Tags.
// Tags is always a non-nil slice (may be empty) to satisfy AC-01.
type TagsResponse struct {
    Tags []TagDTO `json:"tags"`
}

// HierarchyOptions carries filter options for ViewerService.Hierarchy.
// Nil/zero values mean "no filter" — identical to pre-F06 behavior.
type HierarchyOptions struct {
    Tags []string // empty → no filter (AC-04 still applies)
}

// TagReader is the narrow consumer contract that ViewerService needs from
// TagService. *services.TagService satisfies it. Defined here so the
// viewer service can be tested with an in-memory mock without importing
// the full tag package chain.
type TagReader interface {
    ListTags(ctx context.Context) ([]*models.Tag, error)
    EntityIDsByTags(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)
    AttachedTagNamesByIDs(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)
}
```

`TagQueryOp` and `UnregisteredTagError` already exist (F04, F05).

#### 2.3.2 Extended existing types

```go
// Additive: every entity DTO surfaced to the viewer carries Tags.
type HierarchyEpic struct {
    *models.Epic
    Features    []*HierarchyFeature `json:"features"`
    StatusColor string              `json:"status_color"`
    StatusPhase string              `json:"status_phase"`
    Docs        []*HierarchyDoc     `json:"docs"`
    Tags        []string            `json:"tags"` // NEW (REQ-F-003); always non-nil
}

type HierarchyFeature struct {
    *models.Feature
    TaskCount    int             `json:"task_count"`
    BlockedCount int             `json:"blocked_count"`
    StatusColor  string          `json:"status_color"`
    StatusPhase  string          `json:"status_phase"`
    Tasks        []*ViewerTask   `json:"tasks"`
    Docs         []*HierarchyDoc `json:"docs"`
    Tags         []string        `json:"tags"` // NEW (REQ-F-003); always non-nil
}

type ViewerTask struct {
    *models.Task
    StatusColor   string                `json:"status_color"`
    StatusPhase   string                `json:"status_phase"`
    Relationships []ViewerRelatedEntity `json:"relationships"`
    Tags          []string              `json:"tags"` // NEW (REQ-F-003); always non-nil
}

type FlatEntity struct {
    // existing fields (Key, Title, Status, ...)
    Tags []string `json:"tags"` // NEW (REQ-F-003); always non-nil
}

// FeatureTaskOptions grows a Tags slice.
type FeatureTaskOptions struct {
    Status  string
    Agent   string
    Blocked *bool
    Limit   int
    Offset  int
    Tags    []string // NEW (REQ-F-008); empty → no tag filter
}
```

JSON encoding note: the `Tags` field is declared as `[]string`
(non-pointer slice). To guarantee `[]` instead of `null` in the wire
format, the service MUST always assign `[]string{}` when no tags are
attached, rather than leaving it nil. See §2.5 for the population
logic.

#### 2.3.3 New handler interface additions

```go
// In internal/api/viewer/service.go — ViewerServicer interface.
type ViewerServicer interface {
    // ... existing methods ...
    Tags(ctx context.Context) (*services.TagsResponse, error)                             // NEW
    Hierarchy(ctx context.Context, opts services.HierarchyOptions) (*services.HierarchyResponse, error) // MODIFIED signature
    FeatureTasks(ctx context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error) // UNCHANGED signature (opts.Tags is additive)
}
```

The `Hierarchy` signature change breaks only one caller: the handler
in the same package. Handler-test mocks are updated in the same PR.

#### 2.3.4 New HTTP endpoint

**Route:** `GET /api/v1/viewer/tags`

**Query parameters:** none.

**Response 200 OK:**

```json
{
  "tags": [
    {"name": "auth"},
    {"name": "voice"}
  ]
}
```

**Response 500 Internal Server Error:** when `TagService.ListTags`
fails for a reason other than "not wired" (e.g. DB connection error).
Shape: existing `api.ErrorResponse`.

**Response 200 OK, empty vocabulary:** `{"tags":[]}` — never `null`.

**Methods other than GET:** 404 or 405 (ServeMux default; REQ-NF-012
asserts either).

#### 2.3.5 Extended HTTP endpoints — query params

**`GET /api/v1/viewer/hierarchy?tag=<name>[&tag=<name2>...]`**

Parsing: `r.URL.Query()["tag"]` → trim each → drop empties →
`HierarchyOptions.Tags`. See REQ-F-009 and REQ-F-011 for error
handling.

**`GET /api/v1/viewer/features/{key}/tasks?tag=<name>[&tag=<name2>...]`**

Parsing: identical to above → `FeatureTaskOptions.Tags`.

#### 2.3.6 New error response shape for unregistered tags

When the handler catches `*services.UnregisteredTagError`:

```json
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Bad Request",
  "message": "unregistered tag: does-not-exist",
  "unregistered_tags": ["does-not-exist"]
}
```

Implementation: extend `api.ErrorResponse` shape locally in the
viewer package, OR (preferred) define a viewer-specific
`tagErrorResponse` struct so `api.ErrorResponse` stays narrow. The
latter avoids leaking a tag-specific field into other handlers.

```go
// In internal/api/viewer/handler.go.
type tagErrorResponse struct {
    Error            string   `json:"error"`
    Message          string   `json:"message"`
    UnregisteredTags []string `json:"unregistered_tags,omitempty"`
}

func respondUnregisteredTagError(w http.ResponseWriter, err *services.UnregisteredTagError) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusBadRequest)
    _ = json.NewEncoder(w).Encode(tagErrorResponse{
        Error:            "Bad Request",
        Message:          err.Error(),
        UnregisteredTags: err.Names, // F04 defines this exported field
    })
}
```

(If `UnregisteredTagError` exposes names via a different field name,
use that — verified at implementation time.)

### 2.4 Key Technical Decisions

Each decision below is normative for F06. Deviations require a new
ADR.

#### ADR-F06-1: Viewer decorates DTOs in-memory with batched tag lookups

**Decision.** The viewer service populates `Tags` fields by calling
`TagReader.AttachedTagNamesByIDs` once per entity type present in the
response. It does NOT issue a separate query per entity, and it does
NOT change any existing repository method signature to carry tags
inline.

**Alternatives considered.**
(a) Per-entity lookup in `GetByKey` paths (N+1 problem).
(b) SQL join in the base repository queries to materialize tags as a
JSON-array column.
(c) Post-query batched lookup (chosen).

**Rationale.** (a) is rejected by REQ-NF-001. (b) would require
touching six entity repositories and changing their DTO shape, which
violates F05 REQ-NF-010 ("no regression in existing list tests") and
creates per-entity duplication of the tag-join SQL. (c) reuses the
F05 `AttachedTagNamesByIDs` method exactly as designed (F05 REQ-F-013
says: "used by entity services' list-with-tags and get-with-tags
paths to avoid N+1"). F06 is a new consumer, not a new pattern.

**Cost.** The `Hierarchy` method temporarily holds a
`map[EntityType]map[int64][]string` in memory while it walks the tree
to assign `Tags` fields. Peak memory is proportional to the
tagged-entity count, not the total entity count, and is bounded by
the number of `entity_tags` rows in the DB. For a 10000-task project
with 100 tags in use, that's O(10000) map entries — a few hundred KB.

#### ADR-F06-2: `Tags` field is always non-nil slice, never `null`

**Decision.** Every `Tags []string` field in the wire format is
always a non-nil slice. The service populates `[]string{}` when no
tags are attached, never leaves it nil.

**Alternatives considered.**
(a) Omit the field via `omitempty` when empty.
(b) Allow `null` and document clients must null-check.
(c) Always present, `[]` when empty (chosen).

**Rationale.** Epic Architecture §4.5 calls this out explicitly:
"clients never have to null-check." The Go zero value for `[]string`
is `nil`, which marshals as `null`; we must assign an empty slice to
override. This is a well-known Go JSON-encoding pattern; several
existing DTOs in the file already follow it (e.g., `NotesResponse.Notes`
is always non-nil per AC-020.1). `omitempty` (a) breaks the
client-simplification goal because the client still has to check
"field present?". (b) defeats the whole point.

**Cost.** A one-line `if decorated == nil { decorated = []string{} }`
assignment per entity in the decoration step. Negligible.

#### ADR-F06-3: Viewer never imports `MaintainerGate`

**Decision.** `ViewerService`'s constructor, setters, and any exported
surface MUST NOT accept `maintainer.Gate`. `WireServices` MUST NOT
pass one.

**Alternatives considered.**
(a) Inject the gate "for future use" but never call it.
(b) Inject only when a write endpoint exists.
(c) Never inject (chosen).

**Rationale.** Defense in depth. The gate is an access-control
primitive; its presence signals "this service might do privileged
ops." The viewer is read-only by scope (SC-8: "vocabulary management
stays CLI-only in v1"). Injecting a gate makes a future PR that adds
a write endpoint easy and tempting — without the PR author having to
add a new dependency. Not injecting means the PR author would have
to add a new dependency (the gate) to `NewViewerService`, which is
a loud signal in code review. This is a smell-control ADR, not a
runtime invariant.

**Cost.** If a future epic needs to allow maintainer writes from the
viewer, `WireServices` and `NewViewerService` grow one parameter.
This is a well-signposted future change.

#### ADR-F06-4: Hierarchy filter prunes the tree, does not mark

**Decision.** When `?tag=` is supplied on `GET /api/v1/viewer/hierarchy`,
the response is pruned: epics/features with no matching descendant
AND no direct tag match are omitted from the response. The UI does
not need to hide nodes.

**Alternatives considered.**
(a) Return the full tree with a boolean `matches_filter` flag on each
node; UI hides non-matches.
(b) Return a flat list of matching entities plus a separate
breadcrumb map.
(c) Return a pruned tree (chosen).

**Rationale.** (a) bloats the payload for large trees (10000-task
projects sending 10000 booleans) and pushes filter semantics into
the UI. (b) loses the hierarchy, making it hard to display "the
feature this task belongs to." (c) matches user intent — "show me
only the tagged work" — and minimizes response size. The server
already has the tree and the tag-matched ID set; pruning is O(n)
over the walked tree.

**Cost.** UI can no longer easily render "X of Y epics match" hints.
This is acceptable for v1; adding such a hint is future work.

#### ADR-F06-5: `Hierarchy` signature changes; `FeatureTasks` does not

**Decision.** `ViewerService.Hierarchy` grows an options argument
(`HierarchyOptions`). `ViewerService.FeatureTasks` keeps its current
signature and extends `FeatureTaskOptions` with a `Tags` field.

**Alternatives considered.**
(a) Both methods grow new options.
(b) Both methods take query params unstructured (`map[string]string`).
(c) `Hierarchy` grows `HierarchyOptions`; `FeatureTasks` extends
`FeatureTaskOptions` (chosen).

**Rationale.** `FeatureTasks` already has an options struct; extending
it is the idiomatic path and doesn't break any callers (all call
sites pass a struct literal). `Hierarchy` has no options struct; the
idiomatic upgrade is to add one, because future filter work (entity
type, depth limit) will want more fields. Adding `HierarchyOptions`
is a one-line breaking change (only the handler calls it) and sets
the pattern for future filters. (b) is too loose — types catch
typos.

**Cost.** The `ViewerServicer` interface changes, which updates
tests. All updates are in the same PR.

#### ADR-F06-6: UI chip component is a pure CSS + vanilla JS addition

**Decision.** The `.tag-chip` class is defined in the same inline
`<style>` block already present in `viewer.html`. Rendering is vanilla
`document.createElement` JS in the same inline `<script>` block. No
new build step, no new asset.

**Alternatives considered.**
(a) Introduce a small templating library.
(b) Introduce a Web Components custom element.
(c) Vanilla JS + CSS in-file (chosen).

**Rationale.** The viewer is a deliberately self-contained HTML file
(REQ-NF-010); adding a library or build step re-opens the question
of whether to adopt a frontend framework, which is far outside F06
scope. Vanilla JS is consistent with the rest of the file.

**Cost.** Future viewer features that want more sophisticated
components will eventually have to grapple with the same question.
F06 doesn't solve that; it defers.

### 2.5 Integration with Existing Code

#### 2.5.1 Service layer — `ViewerService.Hierarchy` decoration and filter

**Current shape** (simplified): builds epics with features & tasks,
then the flat slices, returns `HierarchyResponse`.

**After F06:**

```go
func (s *ViewerService) Hierarchy(ctx context.Context, opts HierarchyOptions) (*HierarchyResponse, error) {
    ctx, span := s.tracer().Start(ctx, "viewer_service.hierarchy")
    defer span.End()

    // 1. Build the tree and flat lists exactly as pre-F06.
    resp, err := s.buildHierarchyUnfiltered(ctx) // extracted from current body
    if err != nil { return nil, err }

    // 2. If tag filter requested, compute ID sets per entity type.
    //    EntityIDsByTags returns *UnregisteredTagError which we propagate.
    var idSets map[models.EntityType]map[int64]struct{}
    if len(opts.Tags) > 0 && s.tagSvc != nil {
        idSets, err = s.computeTagIDSets(ctx, opts.Tags)
        if err != nil { return nil, err } // UnregisteredTagError or repo error
    }

    // 3. Collect all entity IDs per type for decoration.
    ids := collectIDs(resp) // map[EntityType][]int64

    // 4. Fetch tag names per type (REQ-F-004: 6 calls max).
    tagsByEntity := map[models.EntityType]map[int64][]string{}
    if s.tagSvc != nil {
        for et, idList := range ids {
            if len(idList) == 0 { continue }
            m, err := s.tagSvc.AttachedTagNamesByIDs(ctx, et, idList)
            if err != nil { return nil, err }
            tagsByEntity[et] = m
        }
    }

    // 5. Walk tree; assign Tags fields (always non-nil — ADR-F06-2).
    applyTagsToTree(resp, tagsByEntity)

    // 6. If filter requested, prune (ADR-F06-4).
    if idSets != nil {
        pruneTree(resp, idSets)
    }

    return resp, nil
}
```

Helpers (`buildHierarchyUnfiltered`, `collectIDs`, `applyTagsToTree`,
`pruneTree`, `computeTagIDSets`) are private to the service. Tests
drive the public method; unit-testing the helpers in isolation is
optional.

**Error handling.** `*UnregisteredTagError` from `EntityIDsByTags`
propagates unchanged. `AttachedTagNamesByIDs` errors propagate. When
`s.tagSvc == nil` the filter branch is a no-op AND the decoration
branch assigns `[]string{}` to every DTO — graceful degradation per
REQ-F-015.

#### 2.5.2 Service layer — `ViewerService.FeatureTasks` decoration and filter

**After F06:**

```go
func (s *ViewerService) FeatureTasks(ctx context.Context, featureKey string, opts FeatureTaskOptions) (*FeatureTasksResponse, error) {
    // ... existing feature lookup + base task list build ...

    // 1. Tag filter (pre-pagination, per REQ-F-008).
    if len(opts.Tags) > 0 && s.tagSvc != nil {
        taggedIDs, err := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeTask, opts.Tags, TagQueryOpAnd)
        if err != nil { return nil, err }
        tasks = intersectByID(tasks, taggedIDs) // in-memory set intersection
    }

    // 2. Existing status/agent/blocked filters + pagination.
    // (unchanged — runs after tag filter per REQ-F-008)

    // 3. Post-pagination decoration.
    resultPage := /* existing return value */
    if s.tagSvc != nil && len(resultPage.Tasks) > 0 {
        ids := make([]int64, 0, len(resultPage.Tasks))
        for _, t := range resultPage.Tasks { ids = append(ids, t.ID) }
        tagMap, err := s.tagSvc.AttachedTagNamesByIDs(ctx, models.EntityTypeTask, ids)
        if err != nil { return nil, err }
        for _, t := range resultPage.Tasks {
            t.Tags = tagMap[t.ID]
            if t.Tags == nil { t.Tags = []string{} }
        }
    } else {
        for _, t := range resultPage.Tasks { t.Tags = []string{} }
    }

    return resultPage, nil
}
```

#### 2.5.3 Service layer — new `Tags` method

```go
func (s *ViewerService) Tags(ctx context.Context) (*TagsResponse, error) {
    ctx, span := s.tracer().Start(ctx, "viewer_service.tags")
    defer span.End()

    if s.tagSvc == nil {
        span.SetAttributes(attribute.Int("tag.count", 0))
        return &TagsResponse{Tags: []TagDTO{}}, nil
    }

    tags, err := s.tagSvc.ListTags(ctx)
    if err != nil { return nil, fmt.Errorf("viewer service: list tags: %w", err) }

    out := make([]TagDTO, len(tags))
    for i, t := range tags { out[i] = TagDTO{Name: t.Name} }
    span.SetAttributes(attribute.Int("tag.count", len(out)))
    return &TagsResponse{Tags: out}, nil
}
```

#### 2.5.4 Service layer — `WithTagService` setter

```go
func (s *ViewerService) WithTagService(r TagReader) *ViewerService {
    s.tagSvc = r
    return s
}
```

Matches the existing `WithEntityDocRepo`, `WithIdeaRepo`, etc. pattern
in the same file.

#### 2.5.5 Handler layer — new `Tags` method + query parse helper

```go
// Tags returns the full tag vocabulary for the viewer filter UI.
// GET /api/v1/viewer/tags
func (h *ViewerHandler) Tags(w http.ResponseWriter, r *http.Request) {
    result, err := h.svc.Tags(r.Context())
    if err != nil {
        slog.Error("viewer tags failed", "endpoint", "tags", "error", err)
        respondError(w, http.StatusInternalServerError, "failed to load tags")
        return
    }
    respondJSON(w, http.StatusOK, result)
}

// parseTagsQuery extracts, trims, and de-empties the repeated ?tag= values.
func parseTagsQuery(r *http.Request) []string {
    raw := r.URL.Query()["tag"]
    out := make([]string, 0, len(raw))
    for _, v := range raw {
        v = strings.TrimSpace(v)
        if v != "" { out = append(out, v) }
    }
    return out
}
```

Registered in `RegisterRoutes`:

```go
mux.Handle("GET "+prefix+"/tags", wrap(http.HandlerFunc(h.Tags)))
```

Hierarchy and FeatureTasks handlers grow the same parse step:

```go
opts := services.HierarchyOptions{ Tags: parseTagsQuery(r) }
result, err := h.svc.Hierarchy(r.Context(), opts)
if err != nil {
    var unregErr *services.UnregisteredTagError
    if errors.As(err, &unregErr) {
        respondUnregisteredTagError(w, unregErr)
        return
    }
    // existing error handling
}
```

#### 2.5.6 Wiring — `internal/viewer/server/wire.go`

The existing `WireServices` function builds `tagSvc` at approximately
line 270–281 and injects it into the entity services. F06 adds exactly
one line after `viewerService` construction (around line 364):

```go
viewerService.WithTagService(tagSvc)
```

Placement: after the existing `viewerService.WithEntityDocRepo(...)`
chain and before `WithIdeaRepo(...)` or at the end of the chain —
ordering does not matter, but idiomatic placement is immediately
after the setters that consume the same DB.

REQ-F-014 is enforced by the absence of any `maintainer.Gate`
parameter on `NewViewerService` or any of its setters.

#### 2.5.7 UI — `internal/viewer/assets/viewer.html`

The existing viewer is a single HTML file (~5000 lines) with inline
CSS + JS. F06 additions:

**CSS additions (one block, ~30 lines):**

```css
.tag-chip {
    display: inline-block;
    padding: 2px 8px;
    margin-right: 4px;
    background: var(--accent-dim);
    color: var(--fg);
    border-radius: var(--radius-sm);
    font-size: 0.85em;
    font-weight: 500;
}
.tag-filter-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    padding: 8px;
    border-bottom: 1px solid var(--border);
}
.tag-filter-chips .chip {
    cursor: pointer;
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    background: var(--bg-3);
    color: var(--fg-muted);
    user-select: none;
}
.tag-filter-chips .chip.selected {
    background: var(--accent);
    color: var(--fg);
}
.tag-filter-chips .empty {
    color: var(--fg-dim);
    font-style: italic;
}
```

**JS additions (three helpers, ~80 lines total):**

1. `loadVocabulary()`: fetches `/api/v1/viewer/tags` on viewer boot,
   caches in a module-scoped `vocabulary` array. Called once from
   the existing init function. Handles the `{tags: []}` empty case
   by rendering the "No tags registered yet" empty state. No retry
   in v1 — a failed fetch means the filter control stays in its
   empty state; chips still render on entity detail because the
   `tags` arrays on entities come from a different endpoint.

2. `renderTagChips(entity)`: given an entity object with a `tags`
   array, returns a DOM `<span>` containing one `.tag-chip` per tag.
   Returns `null` when the array is empty (caller skips appending).
   Called from each of the six entity detail-render paths (epic,
   feature, task, bug, change, idea).

3. `renderTagFilterControl(container)`: mounts the multi-select chip
   list into the provided container (sidebar header or list-view
   header). Chip click toggles selection and triggers a refetch of
   the currently visible list with `?tag=<selected>[&tag=<selected2>]`
   appended. Refetch uses the same entry points as the existing
   sidebar refresh.

The exact insertion points for the three helpers in the 5000-line file
are identified during task decomposition, not in the spec.

### 2.6 Testing Approach

Per `.claude/rules/testing/architecture.md`:

1. **Service tests** (`internal/services/viewer_service_test.go`) use
   mocks for `TagReader` (new `MockTagReader` in the test file).
   Tests cover:
   - `Tags()` — empty and populated vocabulary; delegation.
   - `Tags()` with nil `tagSvc` — returns `{tags: []}`, no error.
   - `Hierarchy(opts)` without `opts.Tags` — all DTOs carry `[]`;
     exactly 0 extra `EntityIDsByTags` calls; ≤ 6
     `AttachedTagNamesByIDs` calls.
   - `Hierarchy(opts{Tags: ["voice"]})` — prunes tree; flat slices
     filtered independently.
   - `Hierarchy(opts{Tags: ["voice", "auth"]})` — AND semantics
     verified (an entity with only one tag is pruned).
   - `Hierarchy(opts{Tags: ["does-not-exist"]})` — `*UnregisteredTagError`
     propagates.
   - `Hierarchy(opts)` with nil `tagSvc` — `opts.Tags` is silently
     ignored; every DTO carries `[]`.
   - `FeatureTasks(opts)` — tag filter applied BEFORE pagination;
     `Total` reflects post-tag-filter count.
   - `FeatureTasks` — decoration page-scoped (not pre-filter).

2. **Handler tests** (`internal/api/viewer/handler_test.go`) use the
   existing `MockViewerServicer` pattern:
   - `GET /api/v1/viewer/tags` — happy path, 500 on service error.
   - `POST /api/v1/viewer/tags` — 404 or 405 (REQ-NF-012).
   - `GET /api/v1/viewer/hierarchy?tag=voice` — forwards to service.
   - `GET /api/v1/viewer/hierarchy?tag=&tag=voice&tag=%20%20` —
     empties dropped.
   - `GET /api/v1/viewer/hierarchy?tag=does-not-exist` — 400 with
     `unregistered_tags` field.
   - `GET /api/v1/viewer/features/{key}/tasks?tag=voice` — same.
   - `GET /api/v1/viewer/features/{key}/tasks?tag=does-not-exist` —
     400 (not 404) when feature exists.

3. **No repository tests.** F06 adds no SQL. Repository tests for
   F05's `FilterEntityIDs` and `ListTagNamesByEntities` already
   cover the underlying SQL.

4. **No CLI tests.** F06 touches zero CLI files.

5. **UI test.** No automated test for `viewer.html` in v1. The existing
   asset-serving test (`internal/viewer/assets_test.go`) verifies
   the file is served; F06 does not change that invariant. Manual
   UAT per UAT-7 is the gate.

### 2.7 Migration Notes

Per `.claude/rules/database-critical.md`:

- **No migration.** `CurrentSchemaVersion` stays at the F01-bumped
  value (14).
- **No `skip_migrations: false` toggle required.** Developers merging
  F06 do not have to flip the config.
- **No data backfill.** `entity_tags` rows are untouched.

### 2.8 Backward Compatibility

Per Epic Architecture §5.5 and feature.md §Thin Description:

- Clients on a pre-F06 viewer HTML still work with the F06 API — the
  new `tags` fields on response DTOs are additive.
- A pre-F06 API server (hypothetically, if rolled back) combined
  with F06-UI HTML would result in empty tag chip rows (the UI
  reads `entity.tags` which would be `undefined`, degrading to no
  chips — REQ-F-019). No error.
- The F06 UI HTML ships in the same commit as the API changes, so
  the mismatch window in practice is zero.

---

## 3. Exit Gate Mapping

| Exit-Gate Criterion | Where Satisfied |
|---|---|
| Every requirement is testable | All REQ-F / REQ-NF rows map to AC-## rows in §1.3 or to the test list in §2.6. |
| Every architecture decision references existing patterns or explains deviation | ADR-F06-1 cites F05 REQ-F-013; ADR-F06-2 cites Epic Architecture §4.5; ADR-F06-3 cites the "defense-in-depth" smell-control rationale; ADR-F06-4 explains prune-vs-mark deviation; ADR-F06-5 cites idiomatic Go options pattern; ADR-F06-6 cites REQ-NF-010. |
| File paths listed for all changes | §2.1.2 table lists every file modified with exact paths. No files created (§2.1.1). |
| No TBDs in critical sections | None. |

---

*Last Updated*: 2026-04-24
