---
feature_key: E28-F04-entity-tag-attachment-and-enforcement
epic_key: E28
document_type: spec
title: Entity Tag Attachment and Enforcement — Combined Spec
---

# E28-F04 — Entity Tag Attachment and Enforcement

This is the single combined **Requirements + Architecture** specification for
feature E28-F04. Business context and epic-level architectural decisions are
NOT restated here; this document references the PRD and the epic architecture
doc.

**Upstream context:**

- Epic PRD: `docs/plan/E28-entity-tagging-with-managed-vocabulary/epic.md`
  (see §1 Problem Statement, §2 Goals/Success Criteria, §3 Scope, §6 UAT).
- Epic Architecture: `docs/plan/E28-entity-tagging-with-managed-vocabulary/architecture.md`
  (binding ADRs: ADR-4, ADR-5, ADR-7, ADR-10; design §4.3 entity-command
  integration).
- Feature research: `./research.md` (authoritative map of existing code,
  line-level integration points, extension-vs-new analysis).
- Feature thin description: `./feature.md`.

**Upstream feature dependencies (all complete on this branch):**

- F01 — schema (`tags`, `entity_tags`, cascade triggers, indexes).
- F02 — `maintainer.Gate` package and `shark admin maintainer set-password`.
- F03 — `TagService` skeleton with `ValidateName`, `ListTags`, `AddTag`,
  `RemoveTag`, `RenameTag`; `shark tags list|add|rm|rename` CLI; typed
  errors `ValidationError`, `NotFoundError`, `ConflictError`,
  `TagInUseError`.

**Downstream features blocked by F04:**

- F05 — tag-based querying in list/search (requires `entity_tags` rows to
  query against).
- F06 — viewer tag integration (requires attach semantics to mirror).

---

## 1. Requirements

All requirements are INCREMENTAL over the epic and over F01–F03. They trace
to the PRD Success Criteria (SC-n) and UAT scenarios (UAT-n) in the epic.

### 1.1 Functional Requirements

| ID | Requirement | Traces to |
|---|---|---|
| REQ-F-001 | `TagService` MUST expose `AttachMany(ctx, entityType models.EntityType, entityID int64, names []string) error`. The method normalizes and validates every name via `TagService.ValidateName`, resolves each name to a `*models.Tag` via `TagRepositoryInterface.GetByName`, and attaches via `EntityTagRepositoryInterface.Attach`. Unknown names abort the call before any `Attach` runs. Empty `names` is a no-op returning `nil`. | SC-1, UAT-1, UAT-2 |
| REQ-F-002 | `TagService` MUST expose `DetachOne(ctx, entityType models.EntityType, entityID int64, name string) error`. The method normalizes and validates the name, resolves it to a `*models.Tag`, and calls `EntityTagRepositoryInterface.Detach`. Detaching a tag that is not attached is NOT an error (idempotent; matches the repository's no-op semantics). Detaching a name that is not in the vocabulary returns `*NotFoundError`. | SC-1, UAT-1 |
| REQ-F-003 | `TagService` MUST expose `EnforceRequired(ctx, entityType models.EntityType, names []string) error`. The method reads `Config.TagRequiredFor` (see REQ-F-011). If `entityType.String()` is present in `TagRequiredFor` AND `len(names) == 0`, the method returns a new typed error `*TagRequiredError{EntityType string}`. Otherwise the method returns `nil`. `EnforceRequired` does NOT call the repository. | SC-6, UAT-6 |
| REQ-F-004 | `AttachMany` MUST return a new typed error `*UnregisteredTagError{Name string}` (distinct from F03's `NotFoundError` which is used for vocabulary-management errors) when a provided name passes structural validation but is not present in the vocabulary. The name carried by the error is the normalized form. | SC-2, UAT-2 |
| REQ-F-005 | Both `AttachMany` and `DetachOne` MUST NOT invoke the maintainer gate. Attaching/detaching a registered tag is an open operation available to any user. Only `AddTag`/`RemoveTag`/`RenameTag` (F03) consume the gate. | Research §4; epic PRD §5 Stakeholder Impact (LLM/AI agents) |
| REQ-F-006 | `TagService`'s constructor `NewTagService` MUST grow a fourth required dependency: a `TagEnforcementConfig` interface (defined in `internal/services/tag_service.go`) that exposes a single method `TagRequiredFor() []string`. `*config.Config` will satisfy this interface via a method `(c *Config) TagRequiredFor() []string`. Passing a `nil` config into the constructor panics (matches the existing `requireNonNil` pattern for the other three dependencies, AC-T1). | SC-6 |
| REQ-F-007 | `internal/config/config.go`'s `Config` struct MUST gain a new field `TagRequiredFor []string \`json:"tag_required_for,omitempty"\`` and a method `func (c *Config) TagRequiredFor() []string { if c == nil { return nil }; return c.TagRequiredForSlice /* or field */ }`. Ordering and duplicates in the configured slice are not significant; membership is tested with case-sensitive exact match against `entityType.String()` (the tag CHECK constraint already pins allowed values to lowercase). | SC-6, UAT-6 |
| REQ-F-008 | Each of the six entity Create service methods MUST call `tagSvc.EnforceRequired(ctx, <entityType>, input.Tags)` **before** any key allocation, file write, validation of other fields, or database insert. If `EnforceRequired` returns non-nil the method returns that error unchanged. If the injected `tagSvc` is nil (tags disabled in construction), the call is skipped. Affected methods: `TaskService.CreateTask` (`internal/services/task_service.go:270`), `FeatureService.CreateFeature` (`feature_service.go:757`), `EpicService.CreateEpic` (`epic_service.go:404`), `BugService.CreateBug` (`bug_service.go:92`), `ChangeCardService.CreateChangeCard` (`change_card_service.go:77`), `IdeaService.CreateIdea` (`idea_service.go:65`). | SC-6, UAT-6 |
| REQ-F-009 | After a successful entity insert (row exists with an `ID`), each of the six Create methods MUST call `tagSvc.AttachMany(ctx, <entityType>, created.ID, input.Tags)`. A non-nil return from `AttachMany` propagates unchanged to the caller, leaving the entity persisted (see ADR-F04-2 "no transactions in F04" decision below). If `tagSvc` is nil the call is skipped. | SC-1, UAT-1 |
| REQ-F-010 | Each of the six entity Update service methods MUST call `tagSvc.AttachMany(ctx, <entityType>, existing.ID, updates.Tags)` when `len(updates.Tags) > 0`. **Update semantics are additive only.** Empty `updates.Tags` (nil or `[]string{}`) performs no attach/detach work. Removal on update is explicitly NOT supported (see REQ-F-014 for the explicit removal surface). Affected methods: `UpdateTask`, `UpdateFeature`, `UpdateEpic`, `UpdateBug`, `UpdateChangeCard`, `UpdateIdea`. | Epic Architecture §4.3 |
| REQ-F-011 | Each of the six entity Create DTOs MUST gain a field `Tags []string \`json:"tags,omitempty"\``. Each of the six Update DTOs MUST gain the same field (as `[]string`, NOT `*[]string`, because empty-means-no-change; see REQ-F-010). DTOs touched: `CreateTaskInput`, `TaskUpdates`, `CreateFeatureInput`, `FeatureUpdates`, `CreateEpicInput`, `EpicUpdates`, `CreateBugInput`, `BugUpdates`, `CreateChangeCardInput`, `ChangeCardUpdates`, `CreateIdeaInput`, `UpdateIdeaInput`. | SC-1 |
| REQ-F-012 | The six `shark <entity> create` and six `shark <entity> update` CLI commands MUST accept a repeated `--tag <name>` flag, bound via `cmd.Flags().StringSliceVar(&flag, "tag", nil, "Tag to apply (repeatable).")`. `--tag` is never required by Cobra (enforcement is at the service layer); when omitted, nil is passed to the service. There is no short form (`-t`) in v1 (reserves `-t` for future use; no existing flag claims `-t` across these commands). | SC-1, SC-6 |
| REQ-F-013 | Six new `shark <entity> tag` subcommands MUST exist, each with two sub-subcommands `add` and `rm`, giving the twelve commands `shark task tag add`, `shark task tag rm`, `shark feature tag add`, `shark feature tag rm`, `shark epic tag add`, `shark epic tag rm`, `shark bug tag add`, `shark bug tag rm`, `shark change tag add`, `shark change tag rm`, `shark idea tag add`, `shark idea tag rm`. Each takes exactly two positional arguments: the entity key and the tag name. Invocation shape: `shark <entity> tag add <key> <name>`. | Epic Architecture §4.3 |
| REQ-F-014 | `shark <entity> tag rm <key> <name>` MUST call `TagService.DetachOne` for the resolved entity's ID. This is the only user-facing path for removing a single tag attachment from an entity in F04. | Epic Architecture §4.3 |
| REQ-F-015 | When `AttachMany` fails with `*UnregisteredTagError`, any CLI surface that invoked it (either `--tag` on create/update, or `<entity> tag add`) MUST render the SC-2 error shape on stderr: a single-line error message, followed by the current vocabulary (first 10 tag names comma-separated, with `…and N more` if truncated), followed by the exact remediation command `To add it: shark tags add <name>`. This logic is shared with F03's `handleTagsRmRenameError` (file `internal/cli/commands/tags.go:278`); F04 reuses that helper via an exported wrapper or relocation (see §2.7). | SC-2, UAT-2 |
| REQ-F-016 | CLI exit codes MUST follow the project convention `.claude/rules/go/error-handling.md` and match F03's mapping in `tagsErrorCode` (`tags.go:325`): 0 = success; 1 = `NotFoundError` (vocabulary lookup, entity key not found); 2 = database error (any un-typed repository error); 3 = `ValidationError`, `ConflictError`, `UnregisteredTagError`, `TagRequiredError`. `--json` error output uses the existing `writeTagsError` helper with `code` values `"not_found"`, `"validation"`, `"conflict"`, `"unregistered_tag"`, `"tag_required"`, `"db_error"`. | Research §4; project standard |
| REQ-F-017 | The `cli.GetTagService()` accessor (`internal/cli/tag_global.go`) MUST be updated to pass `cli.GetConfig()` as the fourth argument to `services.NewTagService`. The HTTP service wiring in `cmd/server/services.go` MUST perform the same pass-through when constructing `TagService`. Existing tests that construct `TagService` directly (e.g., `internal/services/tag_service_test.go`) MUST update their constructor calls to pass a config or a small test stub implementing `TagEnforcementConfig`. | Research §2.3 |
| REQ-F-018 | Each of the six entity services (`TaskService`, `FeatureService`, `EpicService`, `BugService`, `ChangeCardService`, `IdeaService`) MUST grow an **optional** `tagSvc *TagService` dependency. When `tagSvc` is nil (production paths where wiring was not updated during a partial rollout; tests that don't care about tags), both `EnforceRequired` and `AttachMany` calls are skipped with no error. Optional status matches the existing pattern for `noteRepo`/`creatorSvc`. | Research §2.5 |

### 1.2 Non-Functional Requirements

| ID | Requirement | Notes |
|---|---|---|
| REQ-NF-001 | **Performance.** `AttachMany` with N registered names is O(N) round-trips: one `GetByName` + one `Attach` per name. This is acceptable given expected N ≤ 5 in realistic use; no batch `GetByName`/`Attach` is introduced in F04. | Research §2.1 (Attach is idempotent `INSERT OR IGNORE`). |
| REQ-NF-002 | **Observability.** All three new `TagService` methods MUST emit an OTel span using the existing `getTracer()` pattern. Span names: `tag_service.attach_many`, `tag_service.detach_one`, `tag_service.enforce_required`. Recorded attributes: `entity.type`, `entity.id`, `tag.count` (for AttachMany only), `tag.name` (for DetachOne only). No attributes carrying vocabulary contents or raw input. | Epic Architecture §6 |
| REQ-NF-003 | **Input sanitization.** Every tag name MUST pass through `TagService.ValidateName` before a repository call. Entity IDs MUST be the post-lookup numeric IDs, never raw user strings. SQL uses only parameterized placeholders (repository layer already compliant). | `.claude/rules/go/input-sanitization.md` |
| REQ-NF-004 | **Concurrency.** `AttachMany` is not atomic across (a) the `GetByName` existence check and (b) the `Attach` insert. A concurrent `shark tags rm --force <name>` between those calls can cause the `Attach` to fail with a foreign-key error on `tag_id`. F04 acknowledges this race as in-scope out-of-concern (operational, not user-facing): the failure surfaces as a generic DB error (exit code 2) and the user retries. Wrapping the entire `AttachMany` in a transaction is explicitly deferred (see ADR-F04-2). | ADR-F04-2 |
| REQ-NF-005 | **Testing.** `TagService` methods tested with mocked repositories (`internal/services/tag_service_test.go`). Entity services tested with a new shared `MockTagService` (`internal/services/mock_tag_service_test.go`) to avoid 6× duplication. CLI commands tested with a mocked service implementing the same `tagServiceIface` used by F03. No real DB in service or CLI tests. | `.claude/rules/testing/architecture.md` |
| REQ-NF-006 | **Dual backend.** All SQL is unchanged from F01; F04 introduces no backend-specific features. SQLite + Turso both supported without branches. | Epic PRD §4 constraint 5 |
| REQ-NF-007 | **Backward compatibility.** Databases that already exist under F01 schema version (14) require NO migration for F04. Configs without `tag_required_for` and without `--tag` usage behave identically to pre-F04. | Epic Architecture §5 |

### 1.3 Acceptance Criteria

Each criterion is a testable statement; the implementing task MUST supply an
automated test or an explicit UAT step. IDs map into the test plan.

| ID | Acceptance Criterion | Test kind |
|---|---|---|
| AC-1 | `TagService.AttachMany(ctx, EntityTypeTask, 42, []string{"voice", "auth"})` with both tags registered issues exactly two `Attach` calls in order and returns nil. | Unit (mock repos) |
| AC-2 | `TagService.AttachMany(ctx, EntityTypeTask, 42, []string{"voice", "does-not-exist"})` with only `voice` registered issues exactly ZERO `Attach` calls and returns `*UnregisteredTagError{Name: "does-not-exist"}`. | Unit |
| AC-3 | `TagService.AttachMany(ctx, EntityTypeTask, 42, nil)` issues zero repository calls and returns nil. | Unit |
| AC-4 | `TagService.AttachMany(ctx, EntityTypeTask, 42, []string{"Voice "})` (untrimmed, mixed case) normalizes to `voice` and behaves identically to AC-1 with one tag. | Unit |
| AC-5 | `TagService.AttachMany(ctx, EntityTypeTask, 42, []string{"voice", "voice"})` (duplicate in same call) issues two `Attach` calls; the second is a no-op per the existing `INSERT OR IGNORE` semantics. | Unit |
| AC-6 | `TagService.DetachOne(ctx, EntityTypeTask, 42, "voice")` when `voice` is registered issues one `GetByName` and one `Detach`, returns nil. | Unit |
| AC-7 | `TagService.DetachOne(ctx, EntityTypeTask, 42, "voice")` when `voice` is NOT registered returns `*NotFoundError{Name: "voice"}` and issues zero `Detach` calls. | Unit |
| AC-8 | `TagService.DetachOne(ctx, EntityTypeTask, 42, "voice")` when `voice` is registered but NOT attached returns nil (Detach's existing no-op semantics are surfaced unchanged). | Unit |
| AC-9 | `TagService.EnforceRequired(ctx, EntityTypeTask, nil)` with config `TagRequiredFor: []string{"task"}` returns `*TagRequiredError{EntityType: "task"}`. | Unit |
| AC-10 | `TagService.EnforceRequired(ctx, EntityTypeTask, []string{"voice"})` with config `TagRequiredFor: []string{"task"}` returns nil. | Unit |
| AC-11 | `TagService.EnforceRequired(ctx, EntityTypeEpic, nil)` with config `TagRequiredFor: []string{"task"}` returns nil. | Unit |
| AC-12 | `TagService.EnforceRequired(ctx, EntityTypeTask, nil)` with config where `TagRequiredFor` is nil or empty returns nil. | Unit |
| AC-13 | Constructing `TagService` with a nil `TagEnforcementConfig` panics with a message matching `requires a non-nil` substring. | Unit |
| AC-14 | `AttachMany` MUST NOT call `gate.Authorize`. Verified by constructing with a `MockGate` whose `Authorize` always returns a rejection error; `AttachMany` still succeeds. | Unit |
| AC-15 | For each of the six entity services, `CreateXxx` with `input.Tags = nil` and `Config.TagRequiredFor` empty calls neither `EnforceRequired` nor `AttachMany` beyond the initial nil-check, and the entity is created successfully. | Unit (each entity) |
| AC-16 | For each of the six entity services, `CreateXxx` with `Config.TagRequiredFor` containing the entity's type and `input.Tags = nil` returns `*TagRequiredError` BEFORE any row is written (verified by mock repo not receiving `Create`). | Unit (each entity) |
| AC-17 | For each of the six entity services, `CreateXxx` with `input.Tags = []string{"voice"}` and `voice` registered calls `AttachMany` exactly once AFTER the entity was persisted (mock `Create` invoked before mock `Attach`). | Unit (each entity) |
| AC-18 | For each of the six entity services, `UpdateXxx` with `updates.Tags = []string{"voice"}` calls `AttachMany` exactly once. `updates.Tags = nil` and `updates.Tags = []string{}` both make zero attach-related repository calls. | Unit (each entity) |
| AC-19 | `shark task create E07 F01 "x" --tag=voice --tag=auth` (both registered) creates the task and attaches both tags. Verified by the integration test spawning the actual CLI against an in-memory DB. | Integration |
| AC-20 | `shark task update E07-F01-001 --tag=voice --tag=voice` is idempotent (two requests produce one attachment; entity_tags row count grows by 1 only). | Integration |
| AC-21 | `shark task update E07-F01-001 --tag=does-not-exist` exits with code 3, stderr lists the vocabulary, and stderr contains the exact substring `To add it: shark tags add does-not-exist`. | Integration |
| AC-22 | With `.sharkconfig.json` set to `{"tag_required_for": ["task"]}`, `shark task create E07 F01 "x"` (no `--tag`) exits non-zero with a clear stderr naming `task` as the missing requirement. `shark epic create "x"` (no `--tag`) succeeds. | Integration (matches UAT-6) |
| AC-23 | `shark bug tag add B001 voice` attaches `voice` to bug B001 once. Re-running the exact same command is a no-op (exit 0, no duplicate row). | Integration |
| AC-24 | `shark bug tag rm B001 voice` removes the attachment. Re-running is a no-op (exit 0, idempotent). | Integration |
| AC-25 | `shark bug tag rm B001 does-not-exist` (vocab does not contain `does-not-exist`) exits with code 1 and stderr includes the vocabulary-snippet plus `To add it: shark tags add does-not-exist`. | Integration |
| AC-26 | `shark idea tag add <id> voice` functions identically to `shark task tag add` for idea entity type. The presence of `idea` in `models.ValidEntityTypes` (from F01) is sufficient; no enum change required. | Integration |
| AC-27 | `.sharkconfig.json` JSON round-trip (marshal → unmarshal) preserves `TagRequiredFor` exactly. Empty/missing field unmarshals to `nil` slice. | Unit (`internal/config/config_test.go`) |
| AC-28 | `UnregisteredTagError.Error()` returns the literal text `tag is not registered: <name>` where `<name>` is the normalized name. `TagRequiredError.Error()` returns the literal text `at least one tag is required for <entityType>`. | Unit |

### 1.4 Out of Scope for F04

- **No changes to `shark list` or `shark search`**: tag-based querying is F05.
- **No viewer integration**: tag rendering and tag-filter UI are F06.
- **No bulk-tagging command** (`shark tag-many <pattern>`): PRD §3 explicitly
  out-of-scope.
- **No tag replacement semantics on update**: `--tag` on update is additive
  only. Replacement, if ever wanted, would be a separate future feature.
- **No transactional guarantees across entity-insert + tag-attach**: see
  ADR-F04-2 below; accepted partial-write semantics are documented.
- **No new schema or migration**: F01 finished the DDL; F04 does not bump
  `CurrentSchemaVersion`.
- **No changes to `MaintainerGate`**: F04 does not consume the gate.
- **No `-t` short flag** for `--tag`.
- **No web API changes**: the HTTP server wiring is updated only to pass the
  config through; no new handlers, no new DTO fields on web responses in F04
  (that's F06).

---

## 2. Architecture

Every section below follows existing project patterns; deviations are
explicitly justified. File paths are absolute within the repo; line numbers
refer to the state of the `E28-entity-tagging-with-managed-vocabulary`
branch as of commit `f7f1f9d` (F03 completion).

### 2.1 Component Overview

F04 adds **zero new packages**, **zero new repository methods**, and
**zero new database tables or columns**. It adds three methods to an
existing service, two typed error values, one config field, and a
set of per-entity wiring edits that follow a uniform pattern.

```
CLI (commands/{task,feature,epic,bug,change,idea}.go) ── new flag --tag
CLI (commands/<entity>_tag.go factory)                 ── new subcommand
    │
    ▼
Entity Service (CreateXxx, UpdateXxx)
    │   (new calls to tagSvc if non-nil)
    ▼
TagService ── new methods: AttachMany, DetachOne, EnforceRequired
    │                                    │
    │                                    └─ reads Config.TagRequiredFor
    ▼
Repository (tag)  ── unchanged (F01 interfaces already sufficient)
```

### 2.2 Data Model Changes

**None.** F01 schema (`tags`, `entity_tags`, six cascade triggers, three
indexes) is sufficient. No migration, no `CurrentSchemaVersion` bump.

Config schema changes (`.sharkconfig.json`):

```json
{
  "tag_required_for": ["task"]
}
```

Additive JSON; absent field ≡ `nil` slice ≡ no enforcement. The existing
`maintainer` block is untouched (F02 territory). No `.sharkconfig.json`
schema file to edit — JSON is schemaless in this project.

### 2.3 Config Changes — `internal/config/config.go`

**New field** on `Config`:

```go
// TagRequiredFor lists entity types that MUST carry at least one tag at
// creation time. Consumed by services.TagService.EnforceRequired (E28-F04).
// Values are entity-type strings as returned by models.EntityType.String()
// ("task", "feature", "epic", "bug", "change", "idea"). Absent or empty =
// no enforcement.
TagRequiredFor []string `json:"tag_required_for,omitempty"`
```

**New method** on `*Config` that also satisfies the new
`services.TagEnforcementConfig` interface:

```go
// TagRequiredFor returns the configured list of entity types that require
// at least one tag on create. Returns nil when the config is nil or the
// field is absent.
func (c *Config) TagRequiredFor() []string {
    if c == nil {
        return nil
    }
    return c.tagRequiredForCopy() // defensive copy so callers can't mutate
}
```

> **Naming collision note.** Both the field and the method want the name
> `TagRequiredFor`. We use an exported method with a different-cased
> backing field `tagRequiredForField []string` stored on the struct, or
> use a field name like `TagRequiredForRaw` with json tag
> `"tag_required_for"`. **Decision: rename the field to
> `TagRequiredForTypes []string` with json tag `"tag_required_for"`.** The
> exported method `TagRequiredFor() []string` returns it. This matches
> Go conventions (exported method on interface, backing storage not
> colliding) and keeps the JSON field name intact.

Updated test: extend `internal/config/config_test.go` with a round-trip
case modeled on `TestConfig_Maintainer_RoundTrip` (line ~1020) —
see AC-27.

### 2.4 Error Types — `internal/services/tag_errors.go`

Two new typed errors are appended:

```go
// UnregisteredTagError is returned by TagService.AttachMany when a name
// passes structural validation but is not present in the vocabulary.
// Distinct from NotFoundError (which is used by vocabulary-management
// paths in F03) so CLI command layers can produce the SC-2 error shape
// without overloading NotFoundError's meaning.
//
// CLI exit code: 3.
// Error() format: "tag is not registered: <Name>"
type UnregisteredTagError struct {
    Name string // normalized (lowercased + trimmed)
}

func (e *UnregisteredTagError) Error() string {
    return fmt.Sprintf("tag is not registered: %s", e.Name)
}

// TagRequiredError is returned by TagService.EnforceRequired when the
// given entity type is listed in Config.TagRequiredFor but the provided
// name slice is empty.
//
// CLI exit code: 3.
// Error() format: "at least one tag is required for <EntityType>"
type TagRequiredError struct {
    EntityType string // the string form of the entity type
}

func (e *TagRequiredError) Error() string {
    return fmt.Sprintf("at least one tag is required for %s", e.EntityType)
}
```

These follow the pattern already established in `tag_errors.go` for
`ValidationError`, `NotFoundError`, `ConflictError`, `TagInUseError`.

### 2.5 TagService — `internal/services/tag_service.go`

**Constructor change:**

```go
// TagEnforcementConfig is the narrow contract TagService needs from config.
// *config.Config satisfies it (see §2.3).
type TagEnforcementConfig interface {
    TagRequiredFor() []string
}

type TagService struct {
    tagRepo       tagrepo.TagRepositoryInterface
    entityTagRepo tagrepo.EntityTagRepositoryInterface
    gate          maintainer.Gate
    cfg           TagEnforcementConfig // NEW — REQ-F-006
    tracer        trace.Tracer
}

func NewTagService(
    tagRepo tagrepo.TagRepositoryInterface,
    entityTagRepo tagrepo.EntityTagRepositoryInterface,
    gate maintainer.Gate,
    cfg TagEnforcementConfig, // NEW
) *TagService {
    requireNonNil(tagRepo, "TagService requires a non-nil TagRepositoryInterface")
    requireNonNil(entityTagRepo, "TagService requires a non-nil EntityTagRepositoryInterface")
    requireNonNil(gate, "TagService requires a non-nil Gate")
    requireNonNil(cfg, "TagService requires a non-nil TagEnforcementConfig")
    return &TagService{
        tagRepo:       tagRepo,
        entityTagRepo: entityTagRepo,
        gate:          gate,
        cfg:           cfg,
    }
}
```

**`AttachMany`:**

```go
// AttachMany attaches the named tags to (entityType, entityID). All names
// must be registered in the vocabulary; encountering an unregistered name
// aborts before any Attach call.
//
// REQ-F-001, REQ-F-004. Does not invoke gate (REQ-F-005).
func (s *TagService) AttachMany(
    ctx context.Context,
    entityType models.EntityType,
    entityID int64,
    names []string,
) error {
    ctx, span := s.getTracer().Start(ctx, "tag_service.attach_many",
        trace.WithAttributes(
            attribute.String("entity.type", string(entityType)),
            attribute.Int64("entity.id", entityID),
            attribute.Int("tag.count", len(names)),
        ),
    )
    defer span.End()

    if len(names) == 0 {
        return nil
    }

    // Phase 1: resolve all names (all-or-nothing).
    resolved := make([]*models.Tag, 0, len(names))
    for _, raw := range names {
        normalized, err := s.ValidateName(raw)
        if err != nil {
            return recordSpanError(span, err) // *ValidationError
        }
        t, err := s.tagRepo.GetByName(ctx, normalized)
        if err != nil {
            if errors.Is(err, tagrepo.ErrTagNotFound) {
                return recordSpanError(span, &UnregisteredTagError{Name: normalized})
            }
            return recordSpanError(span, fmt.Errorf("tag service: attach many: lookup %q: %w", normalized, err))
        }
        resolved = append(resolved, t)
    }

    // Phase 2: attach each (Attach is idempotent via INSERT OR IGNORE).
    for _, t := range resolved {
        if err := s.entityTagRepo.Attach(ctx, entityType, entityID, t.ID); err != nil {
            return recordSpanError(span, fmt.Errorf("tag service: attach many: attach %q: %w", t.Name, err))
        }
    }
    return nil
}
```

**`DetachOne`:**

```go
// DetachOne detaches a single tag from (entityType, entityID). The tag
// must exist in the vocabulary; if the attachment does not exist, detach
// is a no-op (repository contract).
//
// REQ-F-002. Does not invoke gate (REQ-F-005).
func (s *TagService) DetachOne(
    ctx context.Context,
    entityType models.EntityType,
    entityID int64,
    name string,
) error {
    ctx, span := s.getTracer().Start(ctx, "tag_service.detach_one")
    defer span.End()

    normalized, err := s.ValidateName(name)
    if err != nil {
        return recordSpanError(span, err)
    }
    span.SetAttributes(
        attribute.String("entity.type", string(entityType)),
        attribute.Int64("entity.id", entityID),
        attribute.String("tag.name", normalized),
    )

    t, err := s.tagRepo.GetByName(ctx, normalized)
    if err != nil {
        if errors.Is(err, tagrepo.ErrTagNotFound) {
            return recordSpanError(span, &NotFoundError{Name: normalized})
        }
        return recordSpanError(span, fmt.Errorf("tag service: detach one: lookup %q: %w", normalized, err))
    }

    if err := s.entityTagRepo.Detach(ctx, entityType, entityID, t.ID); err != nil {
        return recordSpanError(span, fmt.Errorf("tag service: detach one %q: %w", normalized, err))
    }
    return nil
}
```

**`EnforceRequired`:**

```go
// EnforceRequired returns *TagRequiredError when Config.TagRequiredFor
// contains entityType.String() and names is empty.
//
// REQ-F-003. Does not invoke gate and does not call the repository.
func (s *TagService) EnforceRequired(
    ctx context.Context,
    entityType models.EntityType,
    names []string,
) error {
    _, span := s.getTracer().Start(ctx, "tag_service.enforce_required",
        trace.WithAttributes(
            attribute.String("entity.type", string(entityType)),
            attribute.Int("tag.count", len(names)),
        ),
    )
    defer span.End()

    if len(names) > 0 {
        return nil // fast path
    }
    required := s.cfg.TagRequiredFor()
    et := string(entityType)
    for _, r := range required {
        if r == et {
            return recordSpanError(span, &TagRequiredError{EntityType: et})
        }
    }
    return nil
}
```

### 2.6 Entity Service Integration (six services)

Each service gains an optional `tagSvc *TagService`. The hook shape is
IDENTICAL across all six; the only variation is the `models.EntityType`
constant and the ID read after insert.

**Pattern per `CreateXxx`** (illustrated for `TaskService.CreateTask`
at `internal/services/task_service.go:270`):

```go
func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, error) {
    // ... existing required-field validation ...

    // NEW: enforce tag_required_for before any persistence.
    // REQ-F-008.
    if s.tagSvc != nil {
        if err := s.tagSvc.EnforceRequired(ctx, models.EntityTypeTask, input.Tags); err != nil {
            return nil, err
        }
    }

    // ... existing epic/feature lookup, key generation, model build,
    //     model.Validate, fileops write, repo.Create ...

    // NEW: attach tags after the task row has an ID.
    // REQ-F-009.
    if s.tagSvc != nil && len(input.Tags) > 0 {
        if err := s.tagSvc.AttachMany(ctx, models.EntityTypeTask, task.ID, input.Tags); err != nil {
            return nil, err // NOTE: entity is persisted; see ADR-F04-2.
        }
    }

    return task, nil
}
```

**Pattern per `UpdateXxx`** (illustrated for `TaskService.UpdateTask`
at `internal/services/task_service.go:434`):

```go
func (s *TaskService) UpdateTask(ctx context.Context, key string, updates TaskUpdates) (*models.Task, error) {
    // ... existing lookup + field updates ...

    // NEW: additive tag attach.
    // REQ-F-010.
    if s.tagSvc != nil && len(updates.Tags) > 0 {
        if err := s.tagSvc.AttachMany(ctx, models.EntityTypeTask, task.ID, updates.Tags); err != nil {
            return nil, err
        }
    }

    return task, nil
}
```

**Per-service constructor change** (uniform across all six):

```go
func NewTaskService(
    repo TaskRepository,
    workflowSvc *workflow.Service,
    creatorSvc *taskcreation.Creator,
    noteRepo TaskNoteRepository,
    tagSvc *TagService, // NEW — optional; nil disables tag behaviour.
) *TaskService {
    return &TaskService{
        repo:        repo,
        workflowSvc: workflowSvc.ForLevel(workflow.LevelTask),
        creatorSvc:  creatorSvc,
        noteRepo:    noteRepo,
        tagSvc:      tagSvc,
    }
}
```

Mapping of service ↔ EntityType constant (all constants already exist in
`internal/models/entity_note.go`):

| Service | EntityType constant |
|---|---|
| TaskService | `models.EntityTypeTask` |
| FeatureService | `models.EntityTypeFeature` |
| EpicService | `models.EntityTypeEpic` |
| BugService | `models.EntityTypeBug` |
| ChangeCardService | `models.EntityTypeChange` |
| IdeaService | `models.EntityTypeIdea` |

Create/Update line numbers per the research report (§2.5) — any
task touching these files must re-confirm line numbers at implementation
time since other work on this branch may shift them.

### 2.7 CLI Changes

**Flag wiring.** Every existing `*Create*` and `*Update*` parse helper
(enumerated in research §2.7) gains a `--tag` flag read via
`StringSliceVar`, which collects repeated occurrences into a `[]string`.
The helper passes the slice through to the service-input DTO's new
`Tags` field.

Example edit at `internal/cli/commands/task_helpers.go:207` (parseCreateTaskInput):

```go
var tags []string
cmd.Flags().StringSliceVar(&tags, "tag", nil,
    "Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
// ... after parsing ...
input.Tags = tags
```

Identical edit at each of the 12 sites listed in research §2.7.

**New per-entity `tag` subcommand.** A single shared factory at new file
`internal/cli/commands/entity_tag_cmd.go` produces the `add`/`rm`
subcommands:

```go
// makeEntityTagCmd builds `shark <entity> tag add|rm <key> <name>` for a
// given entity type. Each entity command (task, feature, ...) calls
// this factory in its init() and registers the returned *cobra.Command.
//
// The factory is deliberately entity-agnostic: it receives the entity
// type, a function for resolving the user-provided key to an int64 ID,
// and uses the shared GetTagService accessor.
func makeEntityTagCmd(
    entityType models.EntityType,
    resolveKey func(ctx context.Context, key string) (int64, error),
) *cobra.Command { ... }
```

Each of the six entity command files (`task.go`, `feature.go`, `epic.go`,
`bug.go`, `change.go`, `idea.go`) registers one line:

```go
taskCmd.AddCommand(makeEntityTagCmd(models.EntityTypeTask, resolveTaskID))
```

where `resolveTaskID` is a tiny helper that calls the existing entity
service's `GetByKey` equivalent and returns `entity.ID`. The factory
calls `TagService.AttachMany` (for `add`) or `TagService.DetachOne`
(for `rm`).

**Error handling reuse.** The existing helper
`handleTagsRmRenameError` (`internal/cli/commands/tags.go:278`) is
lightly refactored:

- Rename to `handleVocabularyErrorWithSnippet(cmd, svcLister, name, err)`
  to make its scope clear.
- Keep it in `tags.go` with an exported wrapper OR move to a new
  `internal/cli/commands/tags_shared.go` file.
- The factory in `entity_tag_cmd.go` calls the same helper when the
  service returns `*UnregisteredTagError`, `*NotFoundError`, or
  `*ValidationError`. New code path: add an `errors.As(err,
  &unregistered)` branch that renders the same snippet shape.

Error-code mapping (REQ-F-016) extends `tagsErrorCode` in `tags.go`:

```go
var unregistered *services.UnregisteredTagError
if errors.As(err, &unregistered) {
    return "unregistered_tag", 3
}
var required *services.TagRequiredError
if errors.As(err, &required) {
    return "tag_required", 3
}
```

**Exit codes per REQ-F-016:**

| Typed error | code | exit |
|---|---|---|
| `*services.NotFoundError` | `"not_found"` | 1 |
| `*services.ValidationError` | `"validation"` | 3 |
| `*services.ConflictError` | `"conflict"` | 3 |
| `*services.TagInUseError` | `"in_use"` | 3 |
| `*services.UnregisteredTagError` (NEW) | `"unregistered_tag"` | 3 |
| `*services.TagRequiredError` (NEW) | `"tag_required"` | 3 |
| `*maintainer.UnauthorizedError` | `"unauthorized"` | 3 |
| anything else | `"db_error"` | 2 |

### 2.8 CLI Service Wiring — `internal/cli/tag_global.go` and per-entity accessors

`GetTagService()` updated:

```go
func GetTagService() *services.TagService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database for tag service: %v", err))
    }
    tagRepo := tagrepo.NewTagRepository(db)
    entityTagRepo := tagrepo.NewEntityTagRepository(db)
    gate := GetMaintainerGate()
    cfg := GetConfig() // NEW — 4th arg.
    return services.NewTagService(tagRepo, entityTagRepo, gate, cfg)
}
```

Each entity service accessor in `internal/cli/services_global.go` and
related files (e.g., `GetTaskService`, `GetFeatureService`, ...) gets a
trailing argument `GetTagService()` passed into the entity service
constructor (see §2.6 constructor signatures).

### 2.9 HTTP Service Wiring — `cmd/server/services.go`

`WireServices` updated identically to the CLI path: construct one
`TagService` with the four dependencies and pass it into each of the six
entity service constructors alongside existing dependencies. This file
is not heavily tested; it changes uniformly.

### 2.10 Key Technical Decisions (F04-specific ADRs)

#### ADR-F04-1: Inter-service dependency direction (entity service → TagService)

**Decision.** Entity services (`TaskService`, `FeatureService`, ...) hold an
optional `*TagService` reference. This is the first same-level inter-service
dependency in the codebase. It is one-way (`TagService` never imports entity
services) and optional.

**Alternatives considered.**
(a) Orchestrator layer above services that composes tag + entity operations
(violates `.claude/rules/services/service-design.md` "No circular
dependencies" rule and adds a layer for three line-items of work).
(b) Entity services call `EntityTagRepository` directly (violates the
"services, not repos, from services" layering).
(c) Keep `TagService` as the only caller and have `TaskService` expose a hook
that tags are attached after create (inverts the dependency but complicates
the CLI).
(d) Optional dependency on `*TagService`, as chosen.

**Rationale.** (d) is the smallest delta consistent with
`.claude/rules/architecture.md` (services can call other services; circular
dependencies are forbidden, one-way is fine) and with the existing optional-
dependency pattern (`noteRepo` for TaskService, `creatorSvc` for TaskService).
Because the dependency is optional, test construction of entity services does
not need to change in suites that don't exercise tagging.

#### ADR-F04-2: No transactions around entity-insert + tag-attach

**Decision.** The sequence (a) insert entity row, (b) call
`AttachMany` to attach tags is NOT wrapped in a transaction in F04. If
(b) fails, the entity remains persisted with zero tags. The user sees an
error; subsequent retries of `update --tag` are idempotent.

**Alternatives considered.**
(a) Wrap (a) + (b) in a `sql.Tx` owned by the entity service, pushing
`AttachMany(tx, ...)` overloads through `TagService` and the repositories.
(b) Compensate by deleting the inserted row on tag-attach failure.
(c) Accept partial-write semantics, as chosen.

**Rationale.** Option (a) requires a `tx`-aware variant of every repository
method on the attach path, duplicating six-to-eight public methods and
widening the `TagService` API. Option (b) is fragile (a failing
`Delete` leaves the user in a worse state than a partial attach).
Option (c) matches the existing behavior of related hooks (e.g., note
creation in update paths is also not transactional with the entity write)
and relies on `AttachMany` failures being vanishingly rare in practice —
the only failure modes are a concurrent `tags rm --force` (REQ-NF-004) or
a DB connectivity issue. Both are better handled by retry-on-idempotent
rather than rollback. A future epic can add transactional semantics via
a tx-aware `AttachManyTx` method if real-world data shows a need.

#### ADR-F04-3: `UnregisteredTagError` is distinct from `NotFoundError`

**Decision.** Introduce a new typed error `UnregisteredTagError` used
exclusively on the attach path. F03's `NotFoundError` remains the
vocabulary-management "tag does not exist" error.

**Alternatives considered.**
(a) Reuse `NotFoundError` everywhere; disambiguate by context in the CLI
handler.
(b) Introduce `UnregisteredTagError`, as chosen.

**Rationale.** The two paths have different error-rendering needs at the
CLI: `shark tags rm does-not-exist` and `shark task update --tag=does-not-
exist` both say "the vocabulary does not contain this name," but the
remediation differs (the former is a no-op — the user may already be done;
the latter requires registering the tag first). Distinct types let the
CLI choose its remediation text from the type, not from which command
invoked it. Distinct types also make future behavior divergence cheap
(e.g., if an UnregisteredTagError ever needs to ship a suggested-match
list via fuzzy matching — out of F04 scope).

#### ADR-F04-4: Case-sensitive match of `tag_required_for` values

**Decision.** `EnforceRequired` compares `entityType.String()` to entries in
`Config.TagRequiredFor` with a case-sensitive `==`. Allowed values are the
six lowercase strings `task`, `feature`, `epic`, `bug`, `change`, `idea`.

**Alternatives considered.**
(a) Case-insensitive match.
(b) Validate each entry at config-load time and reject unknown values with
a clear error.
(c) Case-sensitive match with no load-time validation, as chosen.

**Rationale.** `entityType.String()` always produces the lowercase form (per
F01 `entity_tags.entity_type` CHECK constraint), so a misspelled or
mis-cased config entry silently disables enforcement for that type. (a) is
permissive but masks typos. (b) is strictly better but adds ~20 LoC to
config validation and is orthogonal to F04's goals. (c) ships the smallest
surface; (b) can be added later without breaking data. The docs update
(see §2.11) explicitly lists the six allowed values so maintainers do not
have to guess.

#### ADR-F04-5: `--tag` repeats rather than comma-separation

**Decision.** `--tag` is a repeated string flag (`--tag=voice --tag=auth`),
never a comma-separated list (`--tag=voice,auth`).

**Alternatives considered.**
(a) Comma-separated via `StringSliceVar` which already treats commas as
separators by default in Cobra.
(b) Repeated flag only, explicitly setting `StringSliceVar` to disable
comma-split.
(c) Support both.

**Rationale.** A valid tag name may contain hyphens but not commas (per
ADR-4 allowlist `^[a-z0-9][a-z0-9-]{0,63}$`). Comma support adds nothing
and creates subtle bugs for values like `--tag="foo,bar"` that should be
user-visible error (two invocations) rather than silently split. We set
the slice via `StringSliceVar` and rely on users reaching for repetition,
which matches how `--depends-on` and similar multi-value flags work
elsewhere in shark. The help text explicitly says "(repeatable)".

### 2.11 Documentation Changes

- `docs/cli-reference/tags.md` — new sections at the bottom: "Applying
  tags during create/update" and "Retroactive tagging: `shark <entity>
  tag add|rm`". Add AC-19/AC-20/AC-21/AC-23/AC-24/AC-25 as examples.
- `docs/cli-reference/task-commands.md`, `feature-commands.md`,
  `epic-commands.md`, `bug-commands.md`, `change-commands.md`, idea
  section of `README.md` — document the new `--tag` flag on create and
  update, and the new `tag add|rm` subcommand.
- `docs/cli-reference/configuration.md` — new subsection documenting
  `tag_required_for`, listing the six allowed values, and including a
  worked example that matches UAT-6.
- `docs/cli-reference/README.md` — one-line mention under **Advanced**
  or **Vocabulary** section that retroactive tagging is now available.

### 2.12 Testing Additions

| Test file | Additions |
|---|---|
| `internal/services/tag_service_test.go` | Table-driven tests for `AttachMany`, `DetachOne`, `EnforceRequired` covering AC-1 through AC-14, AC-28. Mock `TagEnforcementConfig` via a tiny `type stubCfg struct{ values []string }` with the single method. |
| `internal/services/mock_tag_service_test.go` (new file) | Shared `MockTagService` implementing just `AttachMany`, `DetachOne`, `EnforceRequired`. Used by the six entity-service test files. |
| `internal/services/task_service_test.go` | Add tests per AC-15, AC-16, AC-17, AC-18 for the task branch. |
| `internal/services/feature_service_test.go` | Same, for feature. |
| `internal/services/epic_service_test.go` | Same, for epic. |
| `internal/services/bug_service_test.go` | Same, for bug. |
| `internal/services/change_card_service_test.go` | Same, for change. |
| `internal/services/idea_service_test.go` | Same, for idea. AC-26 also belongs here as an integration-adjacent unit test. |
| `internal/cli/commands/task_test.go` | CLI-level test for `--tag` on create and update. Mock `tagServiceIface`. |
| `internal/cli/commands/<entity>_test.go` (5 files) | Same for feature/epic/bug/change/idea. |
| `internal/cli/commands/entity_tag_cmd_test.go` (new file) | Table-driven test over the six entity types × (add/rm) × (happy, unregistered, not-found, db-error). |
| `internal/config/config_test.go` | Round-trip test for `TagRequiredFor` (AC-27). |

### 2.13 Files Modified / Created Summary

**New files (2):**
1. `internal/cli/commands/entity_tag_cmd.go` — shared factory for `shark <entity> tag add|rm`.
2. `internal/services/mock_tag_service_test.go` — shared mock used by six entity-service tests.

**Modified files (services layer, 8):**
- `internal/services/tag_service.go` — three new methods, constructor grows `cfg` arg.
- `internal/services/tag_errors.go` — two new typed errors.
- `internal/services/task_service.go` — constructor, `CreateTask` hook, `UpdateTask` hook, new optional field.
- `internal/services/feature_service.go` — same shape.
- `internal/services/epic_service.go` — same shape.
- `internal/services/bug_service.go` — same shape.
- `internal/services/change_card_service.go` — same shape.
- `internal/services/idea_service.go` — same shape.

**Modified files (DTOs, 5):**
- `internal/services/task_dto.go` — `Tags []string` on `CreateTaskInput` + `TaskUpdates`.
- `internal/services/epic_dto.go` — `Tags []string` on `CreateEpicInput` + `EpicUpdates` + `CreateFeatureInput` + `FeatureUpdates`.
- `internal/services/bug_dto.go` — `Tags []string` on `CreateBugInput` + `BugUpdates`.
- `internal/services/change_dto.go` — `Tags []string` on `CreateChangeCardInput` + `ChangeCardUpdates`.
- `internal/services/idea_dto.go` — `Tags []string` on `CreateIdeaInput` + `UpdateIdeaInput`.

**Modified files (CLI, 12+):**
- `internal/cli/commands/task.go`, `task_helpers.go` — `--tag` on create + update; register tag subcommand.
- `internal/cli/commands/feature.go`, `feature_helpers.go` — same shape.
- `internal/cli/commands/epic.go`, `epic_helpers.go` — same shape.
- `internal/cli/commands/bug.go` — same shape.
- `internal/cli/commands/change.go` — same shape.
- `internal/cli/commands/idea.go` — same shape.
- `internal/cli/commands/tags.go` — extend `tagsErrorCode` with two new branches; rename `handleTagsRmRenameError` → `handleVocabularyErrorWithSnippet` (optional refactor).
- `internal/cli/tag_global.go` — pass config to `NewTagService`.
- `internal/cli/services_global.go` and each per-entity accessor — pass `GetTagService()` to each entity service constructor.

**Modified files (config, 2):**
- `internal/config/config.go` — new field + new method.
- `internal/config/config_test.go` — round-trip test.

**Modified files (HTTP wiring, 1):**
- `cmd/server/services.go` — pass `TagService` to entity service constructors.

**Modified files (docs, 6):**
- `docs/cli-reference/tags.md`, `configuration.md`, `task-commands.md`,
  `feature-commands.md`, `epic-commands.md`, `bug-commands.md`,
  `change-commands.md`, `README.md` (1 line).

**Unchanged:**
- Every repository file (F01 interfaces and implementations are sufficient).
- Every model file (F01 enum already has `EntityTypeIdea`).
- `internal/db/db.go` (no schema change; `CurrentSchemaVersion` stays).
- `internal/auth/maintainer/` package (F04 does not consume the gate).
- Viewer (`internal/api/viewer`, `internal/viewer`) — F06 territory.
- `shark list`, `shark search` paths — F05 territory.

### 2.14 Integration with Existing Patterns (Rationale Cross-References)

| Decision | Pattern referenced |
|---|---|
| Optional `tagSvc` field on entity services | Existing optional fields: `noteRepo` on `TaskService`, `creatorSvc` on `TaskService`, `EpicNoteRepository` on `EpicService`. Pattern: "nil disables the feature, no error." See `.claude/rules/services/service-design.md` §5. |
| Error types in `tag_errors.go` | Follows `.claude/rules/go/error-handling.md` § Custom Error Types. |
| Typed service errors, CLI maps to exit codes | `.claude/rules/go/error-handling.md` § Exit Codes. |
| OTel span per service method | Existing `TagService` methods in F03 (`ListTags`, `AddTag`, `RemoveTag`, `RenameTag`) all emit spans; F04 follows the same shape. |
| No repository changes | Research §2.1 confirms all needed methods exist. |
| Idempotent attach | F01's `entity_tags` uses `INSERT OR IGNORE` (research §2.1); AC-5 and AC-20 rely on this. |
| Cascade delete on entity delete | F01 installed six cascade triggers (epic PRD §4 constraint; architecture §3.2). No test required in F04 because F01 tested it. |
| Case-insensitive name resolution | F01's `tags.name` uses `COLLATE NOCASE`; `TagRepository.GetByName` relies on it. F04's `ValidateName` normalizes before the lookup, so NOCASE is belt-and-suspenders. |

---

## 3. Exit-Gate Self-Review

- [x] Every requirement (REQ-F-001..018, REQ-NF-001..007) is testable and has at least one AC (AC-1..28) or a referenced existing test.
- [x] Every architecture decision either references an existing project pattern or explains the deviation (five F04-specific ADRs above).
- [x] File paths are listed in §2.13 with modification scope.
- [x] No TBDs in critical sections. One minor naming micro-decision (`TagRequiredForTypes` backing field name in §2.3) is locked.
- [x] Requirements trace to Epic PRD Success Criteria (SC-1..10) and UAT scenarios (UAT-1..9); trace table embedded per row in §1.1.
- [x] Spec does NOT restate epic business context; it references epic.md sections.
- [x] Spec does NOT restate existing architecture; it references architecture.md ADRs and existing code at specific paths/lines.

---

*Last Updated*: 2026-04-23
