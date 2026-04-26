# Feature Research Report: E28-F04 — Entity Tag Attachment and Enforcement

## 1. Strategic Context (from epic + architecture)

F04 is the **integration feature** in E28: it takes the vocabulary primitives built by F01 (schema), F02 (maintainer gate), and F03 (TagService + `shark tags` CLI) and wires them onto every create/update path for all six core entity types. F04 also introduces the third config field (`tag_required_for`) which the service layer must consult pre-insert. Once F04 lands, F05 (tag-based querying) becomes implementable, and F06 (viewer) can display the attached tags.

Binding ADRs for F04:
- **ADR-5** — `entity_tags` shape and multi-tag filter is AND (query side is F05, but the attach side must be compatible with the join queries F05 will issue).
- **ADR-7** — `tag_required_for` enforcement lives at the **service layer** (one enforce call per CreateXxx method), not in repository or schema.
- **ADR-10** — `EntityTypeIdea` is a first-class `models.EntityType`.
- **ADR-4** (inherited) — tag names are lowercase ASCII slugs; TagService owns normalization.
- Update semantics are **additive, never replacing** (§4.3 of architecture) — removal is explicit via `shark <entity> tag rm`.

---

## 2. Existing Implementations — What Already Exists and Can Be Extended

Everything below is already committed on this branch. F04 extends, it does not rebuild.

### 2.1 Repository layer — complete and reusable
- `internal/repository/tag/interfaces.go` — `TagRepositoryInterface` and `EntityTagRepositoryInterface` fully defined.
- `internal/repository/tag/entity_tag_repository.go` — `Attach`, `Detach`, `ListByEntity`, `ListByTag`, `ListByEntityType`, `CountByTag` all implemented with OTel tracing. `Attach` uses `INSERT OR IGNORE` so re-attaching is idempotent (the exact property F04's `AttachMany` needs).
- `internal/repository/tag/tag_repository.go` — vocabulary CRUD including `GetByName` (needed by `AttachByNames`) and `List` (needed by the SC-2 stderr vocabulary listing).
- Schema (F01) already created `tags`, `entity_tags`, the six cascade-delete triggers, and the `(entity_type, entity_id)` / `(tag_id)` / `(tag_id, entity_type)` composite indexes in `internal/db/db.go`. No new migration is required by F04.

**F04 adds zero new repository methods. Pure service+CLI+DTO work.**

### 2.2 Service layer (TagService) — partial; F04 completes it
File: `internal/services/tag_service.go`

**Exists today (from F03):** constructor, `SetTracer`, `ValidateName`, `ListTags`, `AddTag`, `RemoveTag`, `RenameTag`, plus the typed errors `ValidationError`, `NotFoundError`, `ConflictError`, `TagInUseError` (in `tag_errors.go`).

**F04 must add exactly three methods to `TagService`**:
1. `AttachMany(ctx, entityType models.EntityType, entityID int64, names []string) error`
2. `DetachOne(ctx, entityType models.EntityType, entityID int64, name string) error`
3. `EnforceRequired(ctx, entityType models.EntityType, names []string) error`

Internal contract:
- `AttachMany` calls `ValidateName` (already exists) for each name, resolves via `tagRepo.GetByName`, and loops `entityTagRepo.Attach`. Unregistered names must return an error shape suitable for the SC-2 stderr contract — new typed error or enriched `NotFoundError` carrying vocabulary snapshot for CLI rendering (mirrors `handleTagsRmRenameError` in `tags.go` lines 278–325).
- `DetachOne` resolves the name and calls `entityTagRepo.Detach` (already a no-op when row doesn't exist).
- `EnforceRequired` reads `TagRequiredFor` from Config and fails with a typed `TagRequiredError{EntityType string}` when the entity type is listed AND `len(names) == 0`.

**Important**: `TagService` currently does **not** hold a `*config.Config`. Its constructor is `NewTagService(tagRepo, entityTagRepo, gate)`. F04 must add a fourth dependency (the config, or a narrower `TagEnforcementConfig` interface) and update `cli.GetTagService()` at `internal/cli/tag_global.go`.

### 2.3 Config layer — `Maintainer` exists, `TagRequiredFor` does NOT yet exist
File: `internal/config/config.go`

`TagRequiredFor` is absent from Config. F04 must:
1. Add `TagRequiredFor []string \`json:"tag_required_for,omitempty"\`` to `Config`.
2. Extend JSON round-trip test (peer: `config_test.go:1020 TestConfig_Maintainer_RoundTrip`).
3. Wire `cli.GetTagService()` to read from `cli.GetConfig()` (pattern established by `GetMaintainerGate()` in `internal/cli/maintainer_global.go`).

No `.sharkconfig.json` schema migration required (JSON is additive).

### 2.4 CLI service accessor — complete
- `internal/cli/tag_global.go` — `GetTagService()` constructs a fresh TagService per call. F04 edits it to pass the config.

### 2.5 Entity services — six Create methods, six Update methods to edit

| Service file | Create method | Update method |
|---|---|---|
| `internal/services/task_service.go` | `CreateTask` line 270 | `UpdateTask` line 434 |
| `internal/services/feature_service.go` | `CreateFeature` line 757 | `UpdateFeature` line 872 |
| `internal/services/epic_service.go` | `CreateEpic` line 404 | `UpdateEpic` line 492 |
| `internal/services/bug_service.go` | `CreateBug` line 92 | `UpdateBug` line 214 |
| `internal/services/change_card_service.go` | `CreateChangeCard` line 77 | `UpdateChangeCard` line 227 |
| `internal/services/idea_service.go` | `CreateIdea` line 65 | `UpdateIdea` line 167 |

Per architecture §4.3 the hook shape in each Create method:
```go
// 1. Before any persistence or key generation:
if err := tagSvc.EnforceRequired(ctx, models.EntityTypeTask, input.Tags); err != nil {
    return nil, err
}
// 2. After the entity row exists and has an ID:
if err := tagSvc.AttachMany(ctx, models.EntityTypeTask, created.ID, input.Tags); err != nil {
    return nil, err
}
```
In each Update method, only `AttachMany` (idempotent/additive).

**Dependency injection concern.** Entity services currently do NOT hold a `*TagService` reference. Constructors must grow a new parameter. Follow the existing pattern: add `tagSvc *TagService` as an **optional** constructor dependency (matches `noteRepo` optional pattern).

### 2.6 Entity DTOs — six Create DTOs to extend, six Update DTOs to extend

| DTO file | Create DTO | Update DTO |
|---|---|---|
| `internal/services/task_dto.go` | `CreateTaskInput` line 8 | `TaskUpdates` line 28 |
| `internal/services/epic_dto.go` | `CreateEpicInput` line 10, `CreateFeatureInput` line 36 | `EpicUpdates` line 26, `FeatureUpdates` line 51 |
| `internal/services/bug_dto.go` | `CreateBugInput` line 6 | `BugUpdates` line 20 |
| `internal/services/change_dto.go` | `CreateChangeCardInput` line 4 | `ChangeCardUpdates` line 23 |
| `internal/services/idea_dto.go` | `CreateIdeaInput` line 7 | `UpdateIdeaInput` line 39 |

Each gets `Tags []string \`json:"tags,omitempty"\``. For `UpdateXxx`, the field is `Tags []string` (not `*[]string`) — additive semantics, empty slice means "no change."

### 2.7 CLI commands — six create sites, six update sites, plus new `<entity> tag` subcommand per type

**Create/update flag wiring** (existing call sites to edit):
- Task: `parseCreateTaskInput` at `internal/cli/commands/task_helpers.go:207`; update via `parseTaskUpdates` at `task_helpers.go:266`.
- Feature: `parseCreateFeatureInput` at `feature_helpers.go:773`; update at `feature_helpers.go:1016`.
- Epic: `epic_helpers.go:603`; update at `epic_helpers.go:923` and `:966`.
- Bug: `runBugCreate` at `bug.go:203`; update at `bug.go:344`.
- Change-card: `buildCreateChangeCardInput` at `change.go:411`; update at `change.go:330`.
- Idea: create at `idea.go:282`; update at `idea.go:304`.

**Flag to add at every create/update site**:
```go
cmd.Flags().StringSliceVar(&flagTags, "tag", nil, "Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
```

**New subcommand per entity type: `shark <entity> tag add/rm <key> <name>`.** Follow pattern of `makeNoteCmd`/`makeNotesCmd` in `internal/cli/commands/bug.go:168`. F04 should add a symmetric `makeTagAttachCmd(entityType string)` factory helper, then register it via `bugCmd.AddCommand(makeTagAttachCmd("bug"))` etc.

**Error shaping** — `handleTagsRmRenameError` at `internal/cli/commands/tags.go:278–325` already constructs the vocabulary-snippet stderr message. F04 reuses this function for unregistered-tag error shape on attach.

### 2.8 Tests — existing patterns to replicate
- `internal/services/tag_service_test.go` — mocks `TagRepositoryInterface`, `EntityTagRepositoryInterface`, and `maintainer.Gate`. F04 extends this file with tests for the three new methods.
- `internal/cli/commands/tags_test.go` — CLI tests that inject mock via `newTagsAddCmd(mockSvc)`. Replicate for entity tag subcommands.
- Each entity service test file (e.g. `task_service_test.go`) needs `MockTagService` — **create shared mock in new `internal/services/mock_tag_service_test.go`** to avoid 6× duplication.

### 2.9 Models — no changes
`models.EntityType` already includes `EntityTypeIdea` (per F01, ADR-10). F04 uses the existing constants.

---

## 3. Integration Points (summary)

| Touch point | File(s) | Type of change |
|---|---|---|
| `Config.TagRequiredFor` | `internal/config/config.go` | Add field + JSON tag |
| Add 3 methods to `TagService` | `internal/services/tag_service.go` | Add `AttachMany`, `DetachOne`, `EnforceRequired` |
| Add `TagRequiredError` (and possibly `UnregisteredTagError`) | `internal/services/tag_errors.go` | New typed errors |
| Wire config into TagService constructor | `internal/services/tag_service.go`, `internal/cli/tag_global.go` | Constructor + accessor |
| Inject `TagService` into 6 entity services | `internal/services/{task,feature,epic,bug,change_card,idea}_service.go` | Constructor param + struct field |
| Hook `EnforceRequired` + `AttachMany` into 6 CreateXxx methods | same six files | In-body calls pre/post persistence |
| Hook `AttachMany` into 6 UpdateXxx methods | same six files | In-body call |
| Add `Tags []string` to 6 Create DTOs + 6 Update DTOs | `internal/services/*_dto.go` | Field additions |
| `--tag` repeated flag on 6 create and 6 update CLI commands | `internal/cli/commands/{task,feature,epic,bug,change,idea}.go` + helpers | Flag wiring |
| New `shark <entity> tag add/rm` subcommand (6 entities) | same six files + new `tag_attach_helpers.go` | ~80 LOC shared helper, 6 registrations |
| Extract/reuse `handleTagsRmRenameError` vocabulary-snippet logic | `internal/cli/commands/tags.go` | Export or relocate to `tags_shared.go` |
| Update `cli.GetTagService()` to pass config | `internal/cli/tag_global.go` | 1-line change |
| Update HTTP server service wiring | `cmd/server/services.go` | Inject TagService into entity services |
| Tests: TagService 3 new methods (table-driven) | `internal/services/tag_service_test.go` | Extend |
| Tests: shared `MockTagService` | `internal/services/mock_tag_service_test.go` | New file |
| Tests: entity service Create/Update with tag hooks | 6× `*_service_test.go` | Extend |
| Tests: CLI `--tag` + `<entity> tag add/rm` | 6× `internal/cli/commands/*_test.go` | New cases |
| Tests: Config.TagRequiredFor round-trip | `internal/config/config_test.go` | New case |

---

## 4. Inter-Feature Dependency Map Within E28

```
F01 (schema) ─┬──► F03 (TagService: List/Add/Rm/Rename + CLI `shark tags ...`)
              │       │
              │       └─► F04 (TagService: AttachMany/DetachOne/EnforceRequired;
              │                             --tag on create/update;
              │                             shark <entity> tag add/rm;
              │                             adds Config.TagRequiredFor)
              │                   │
              │                   └─► F05 (SELECT … WHERE tag filters in list/search)
              │                   │
              │                   └─► F06 (viewer display + filter)
              │
F02 (maintainer gate) ──► F03 (consumed by Add/Rm/Rename)
                              ► (NOT F04 — attach/detach are open to all users)
```

**Critical read**: F04 does **not** consume the maintainer gate. Attaching a registered tag to an entity is an open operation; only registering/removing/renaming the tag name itself is gated.

F04 blocks F05 and F06.

---

## 5. Extension-vs-New Analysis

| Component | Extends / New | Notes |
|---|---|---|
| `tags` / `entity_tags` schema | Extend (nothing) | Complete in F01. |
| `EntityTagRepository` methods | Extend (nothing) | Idempotent Attach already in place. |
| `TagService.ValidateName` | Reuse | Comment already anticipates F04. |
| `TagService` errors | Extend + add 1-2 | New `TagRequiredError`; enrich `NotFoundError` or add `UnregisteredTagError`. |
| `AttachMany/DetachOne/EnforceRequired` | **New** | 3 methods, ~100 LOC total. |
| `TagService` constructor | Extend | Add config dependency. |
| `cli.GetTagService()` | Extend | Pass config. |
| `Config.TagRequiredFor` | **New field** | Single line + round-trip test. |
| 6 entity services | Extend | Optional `tagSvc` param + 2 hook lines per Create, 1 per Update. |
| 6 Create DTOs + 6 Update DTOs | Extend | Add `Tags []string`. |
| 6 CLI create + 6 CLI update commands | Extend | `--tag` StringSlice flag. |
| `shark <entity> tag add/rm` | **New via shared factory** | `makeTagAttachCmd(entityType)` + 6 registrations. ~120 LOC. |
| Vocabulary-snippet error rendering | Extend / Relocate | Reuse `handleTagsRmRenameError` logic. |
| HTTP service wiring | Extend | Inject TagService into entity services in `cmd/server/services.go`. |
| Viewer | Extend (nothing) | F06 handles viewer integration. |
| Tests | Extend + 1 new file | `mock_tag_service_test.go`. |

**Net new files: ≤ 2.** Everything else is extension.

---

## 6. Recommended Implementation Approach

### 6.1 Suggested task decomposition (order matters)
1. **Config + error types** (atomic).
   - Add `Config.TagRequiredFor` and round-trip test.
   - Add `TagRequiredError` to `tag_errors.go`.
2. **Extend TagService** (independent of entity services).
   - Add `AttachMany`, `DetachOne`, `EnforceRequired` with table-driven tests.
   - Update `NewTagService` signature to accept config; update `cli.GetTagService()`.
3. **Shared CLI helper** for unregistered-tag stderr shape. Extract vocabulary-snippet logic from `handleTagsRmRenameError`.
4. **Wire TagService into one entity service end-to-end (proof of concept)** — recommend starting with **bug** (simplest Create/Update signatures). Verify the full path:
   - DTO extension (`CreateBugInput.Tags`, `BugUpdates.Tags`).
   - `CreateBug` calls `tagSvc.EnforceRequired` pre-insert and `tagSvc.AttachMany` post-insert.
   - `UpdateBug` calls `tagSvc.AttachMany` additively.
   - New `bugCmd.AddCommand(makeTagAttachCmd("bug"))` with `add`/`rm` subcommands.
   - `--tag` flag on `bugCreateCmd` and `bugUpdateCmd`.
   - Full tests with mocked `TagService`.
5. **Replicate pattern to 5 remaining entities** in parallel tasks (task, feature, epic, change-card, idea).
6. **Documentation updates** — extend `docs/cli-reference/tags.md`, update create/update reference pages, update `configuration.md`.

### 6.2 Key design decisions for the spec
- **Service-to-service dependency direction.** Each entity service imports `services.TagService`. This is the first inter-service dependency. Architecture §1.3 allows it. Make it optional (nil means "tags disabled"), matching the `noteRepo` pattern.
- **Transaction boundary for create.** If `AttachMany` fails partway after entity insert, entity exists with partial tags. **Recommendation: don't introduce transactions in F04 scope** — `Attach` is idempotent via `INSERT OR IGNORE`, retries are safe. Document and accept.
- **Unregistered tag error vs NotFoundError.** Reuse `NotFoundError` for "tag X doesn't exist in vocabulary"; CLI differentiator is context.
- **Tag name case and whitespace** — handled by `TagService.ValidateName`; `AttachMany` delegates fully.
- **Empty `input.Tags`.** `AttachMany` with `[]string{}` is no-op. `EnforceRequired` with `[]string{}` fails only when entity type is in `tag_required_for`.
- **CLI flag type.** Use `StringSliceVar`. Check existing flags before claiming `-t` short form.

### 6.3 Non-goals for F04 (explicit)
- No changes to `shark list` or `shark search` query paths — that's F05.
- No changes to the viewer — that's F06.
- No new schema or migration — F01 finished the DDL.
- No changes to `MaintainerGate` API — F02 finished it and F04 doesn't consume it.
- No bulk tagging command — PRD §3 Out of Scope.

---

## 7. Exit Gate Checklist

- [x] All existing related code identified with file paths.
- [x] Extension points documented.
- [x] Inter-feature dependency map explicit.
- [x] Extension-vs-new analysis shows F04 is ~85% extension; net new files ≤ 2.
- [x] Actionable for architect in specify step (proof-of-concept entity: bug).
