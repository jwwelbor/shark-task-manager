---
feature_key: E28-F03-tag-vocabulary-service-and-cli
epic_key: E28
document_type: spec
title: "Spec — Tag Vocabulary Service and CLI"
---

# Spec — Tag Vocabulary Service and CLI

Combined requirements + architecture specification for feature **E28-F03**.

**References (not restated here).** This spec is incremental over the documents
below; consult them for business context and system-level decisions.

- Epic PRD: `docs/plan/E28-entity-tagging-with-managed-vocabulary/epic.md`
  (§2 SC-1, SC-2, SC-3, SC-4, SC-5, SC-9; §3 scope; §4 open questions O-1,
  O-6, O-7; §6 UAT-1, UAT-2, UAT-3, UAT-4, UAT-5, UAT-8)
- Epic architecture: `docs/plan/E28-entity-tagging-with-managed-vocabulary/architecture.md`
  (ADR-2 reusable gate, ADR-4 name normalization, ADR-8 rename semantics,
  ADR-9 rm-in-use policy; §1.3 service layering; §4.1 CLI wiring; §4.6
  future-consumer contract)
- F01 spec (schema): `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F01-tags-schema-and-migration/spec.md`
- F02 spec (gate): `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F02-reusable-maintainer-authorization-gate/spec.md`
- Feature description: `docs/plan/E28-entity-tagging-with-managed-vocabulary/E28-F03-tag-vocabulary-service-and-cli/feature.md`

**Scope of this feature.** Deliver the admin surface for the closed tag
vocabulary: a new `TagService` that owns vocabulary business rules (name
normalization, uniqueness, in-use checks, rename collision, gate invocation)
and the thin CLI command group `shark tags list | add | rm | rename`.
Entity-tag attachment and `--tag` flags on other commands are **out of scope
— deferred to F04**. Tag-filtered listing is out of scope — deferred to F05.
Viewer integration is out of scope — deferred to F06.

**What F03 ships.** New file `internal/services/tag_service.go`, new file
`internal/cli/commands/tags.go`, new `GetTagService()` accessor in
`internal/cli/services_global.go` (or a companion file), and updated CLI
reference documentation for the new commands. F01 supplies the schema and
repositories; F02 supplies the maintainer gate; this feature is the
first consumer of the gate and the anchor of every future tag feature.

---

## 1. Requirements

### 1.1 Functional Requirements

**REQ-F-001 — Package and file locations.**
The vocabulary service MUST live at `internal/services/tag_service.go`. The
CLI command group MUST live at `internal/cli/commands/tags.go`. The CLI
accessor `GetTagService()` MUST live at `internal/cli/tag_global.go` (new
file, parallel to `internal/cli/maintainer_global.go`). The service MUST
depend only on interfaces from `internal/repository/tag` (F01) and
`internal/auth/maintainer` (F02) — never on concrete struct types.
Traces to epic architecture §1.3 (service layering) and F02 REQ-F-010
(future-consumer contract).

**REQ-F-002 — `TagService` public surface.**
`TagService` MUST expose exactly these public methods (listed with their
business contract, not signatures — signatures are in §2.3):

| Method | Gate invoked? | Business contract |
|---|:---:|---|
| `ListTags` | no | Return the full vocabulary, ordered by name ascending. Open to all users. |
| `AddTag` | yes | Validate the normalized name, authorize via the gate, create the tag, call `RecordSuccess`. Return the created `*models.Tag`. |
| `RemoveTag` | yes | Authorize, look up the tag by name, determine usage count, decide per ADR-9 whether `--force` is required, delete, `RecordSuccess`. |
| `RenameTag` | yes | Authorize, validate both names, look up source tag, detect collision (ADR-8: hard error), rename, `RecordSuccess`. Return the renamed `*models.Tag`. |
| `ValidateName` | no | Pure function exposed so F04's `TagService.AttachByNames` (future) can share a single implementation. Accepts raw input, returns the normalized (lowercased-trimmed) form or a typed validation error. |

Traces to epic §2 SC-1, SC-2, SC-5, SC-9; UAT-1, UAT-2, UAT-5, UAT-8.

**REQ-F-003 — Name normalization and validation rules (ADR-4).**
Every user-supplied tag name that enters any mutating service method MUST be
passed through a single canonicalization step:
1. `strings.TrimSpace(input)`
2. `strings.ToLower(result)`
3. Validated against `models.ValidateTagName` (which enforces
   `^[a-z0-9][a-z0-9-]{0,63}$` per the F01 model layer).

Inputs that fail validation MUST return a typed `*ValidationError` (see
REQ-F-007) whose `Error()` string names the field (`"tag name"` or
`"old name"` / `"new name"` for rename) and explains the allowed character
set. The **same** normalization MUST be applied to `AddTag`, `RemoveTag`,
and both arguments of `RenameTag`. Traces to ADR-4 and epic input-sanitization
rules (CLAUDE.md `go/input-sanitization.md`).

**REQ-F-004 — Gate invocation pattern (ADR-2, F02 REQ-F-010).**
Every mutating method (`AddTag`, `RemoveTag`, `RenameTag`) MUST invoke
`s.gate.Authorize(ctx, providedPass)` as its **first** step (before any name
validation, before any DB read), and MUST call `s.gate.RecordSuccess(ctx)` on
a successful outcome. `RecordSuccess` errors MUST be logged but MUST NOT
propagate as a method failure (best-effort per F02 REQ-F-003).

`ListTags` MUST NOT invoke the gate. It is always readable.

The service MUST NOT reach past the `Gate` interface. No type assertion, no
reflection, no back-channel: the service sees `Authorize(ctx, pass) error`
and `RecordSuccess(ctx) error`. Traces to epic ADR-2, §1.3 and F02 REQ-F-010.

**REQ-F-005 — `RemoveTag` in-use policy (ADR-9).**
`RemoveTag(ctx, name, force bool)` MUST:
1. Authorize.
2. Normalize and validate the name.
3. Look up the tag via `TagRepository.GetByName`. If not found, return a
   typed `*NotFoundError`.
4. Query `EntityTagRepository.CountByTag(tag.ID)`.
5. If `count > 0` and `force == false`, return a typed `*TagInUseError`
   carrying the usage count. The error `Error()` string MUST include the
   count and the exact command the user can re-run with `--force` to
   proceed. **The tag MUST NOT be deleted in this branch.**
6. If `count > 0` and `force == true`, call `TagRepository.Delete(id, true)`
   (which internally removes `entity_tags` rows in the same transaction
   per F01).
7. If `count == 0`, call `TagRepository.Delete(id, false)`.
8. On successful delete, call `RecordSuccess`.

Traces to ADR-9, epic UAT (§6 is silent on rm — open question O-6 in the
PRD; ADR-9 is the architectural resolution).

**REQ-F-006 — `RenameTag` collision policy (ADR-8).**
`RenameTag(ctx, oldName, newName string)` MUST:
1. Authorize.
2. Normalize and validate both names. If `oldName == newName` after
   normalization, return a typed `*ValidationError` explaining that the
   new name must differ from the old.
3. Look up the source tag via `TagRepository.GetByName(oldName)`. If not
   found, return a typed `*NotFoundError`.
4. Pre-check collision: attempt `TagRepository.GetByName(newName)`. If it
   returns a tag, return a typed `*ConflictError` stating that `newName`
   already exists. (This pre-check is **not** for correctness — the
   repository's UNIQUE constraint already guarantees it — it is to surface
   the collision as a typed error without relying on the repository's
   wrapped `ErrTagConflict`.)
5. Call `TagRepository.Rename(tag.ID, newName)`. If it returns
   `tag.ErrTagConflict` (race with another rename), translate to
   `*ConflictError`.
6. The rename MUST touch only `tags.name` — `entity_tags` rows are
   immutable (F01 already guarantees this in its repository). The service
   MUST NOT iterate `entity_tags` during rename. Traces to epic SC-5 / UAT-5.
7. On success, call `RecordSuccess`. Return the updated `*models.Tag`.

Traces to ADR-8, epic SC-5, SC-7, UAT-5.

**REQ-F-007 — Typed service errors.**
`TagService` MUST define and export the following typed error structs, each
implementing `error`. Callers (the CLI) use `errors.As` on these types to
map to exit codes. `maintainer.UnauthorizedError` (from F02) is also
surfaced unchanged — the service MUST NOT wrap it in a service-local type.

| Type | Fields | Error() contains | CLI exit code |
|---|---|---|---|
| `*ValidationError` | `Field string; Message string` | `"invalid <field>: <message>"` | 3 |
| `*NotFoundError` | `Name string` | `"tag not found: <name>"` | 1 |
| `*ConflictError` | `Name string` | `"tag already exists: <name>"` | 3 |
| `*TagInUseError` | `Name string; Count int64` | `"tag %q is in use by %d entities; re-run with --force to delete it and its associations"` | 3 |
| `*maintainer.UnauthorizedError` | (from F02) | F02 message text | 3 |

Traces to epic SC-2 (helpful error messages), SC-3 (gate error guidance),
and the error-handling rule set in `.claude/rules/go/error-handling.md`.

**REQ-F-008 — Actionable error messages for unregistered-name errors on `rm`/`rename`.**
`RemoveTag` and `RenameTag` failure with `*NotFoundError` MUST produce a
CLI stderr string (composed in the command layer, REQ-F-010) that includes
(a) the current vocabulary list (first 10 names, with "…and N more" if
longer), and (b) the exact `shark tags add` command the user could run.
This mirrors SC-2 (which is about attaching a tag but applies the same UX
principle to vocabulary admin errors). The service MUST expose the
vocabulary via `ListTags` so the command can assemble the message without
reaching into the repository directly. Traces to epic SC-2 extended.

**REQ-F-009 — CLI command tree (`shark tags`).**
A new root command `shark tags` MUST exist with four subcommands:

| Command | Args | Flags | Gate? |
|---|---|---|:---:|
| `shark tags list` | none | `--json` (global) | no |
| `shark tags add <name>` | `<name>` (exactly 1) | `--pass <value>` | yes |
| `shark tags rm <name>` | `<name>` (exactly 1) | `--pass <value>`, `--force` | yes |
| `shark tags rename <old> <new>` | `<old> <new>` (exactly 2) | `--pass <value>` | yes |

Each mutating command MUST accept `--pass` (string, optional). When empty,
the gate relies on a live cache entry. Interactive password prompting is
**not** in scope for F03 (explicit **out of scope** item in §1.4).

Each command MUST be a thin wrapper per the CLI integration rules
(`.claude/rules/services/cli-integration.md`): parse flags, call
`cli.GetTagService()`, format output, map errors to exit codes. No
vocabulary validation, no gate logic, no DB access inline.

Traces to epic §3 scope ("shark tags list / add / rm / rename"), UAT-1,
UAT-3, UAT-5.

**REQ-F-010 — CLI error-to-exit-code mapping.**
The `tags` commands MUST map service errors to exit codes exactly as in the
REQ-F-007 table. On `*UnauthorizedError` from the gate, stderr MUST include
`Error()` plus (when `UserHint()` is non-empty) the hint text on a second
line. On `*NotFoundError` (rm / rename), stderr MUST include the vocabulary
snippet and the `shark tags add` hint per REQ-F-008.

**REQ-F-011 — JSON output format.**
With `--json`:
- `shark tags list` emits `[{"name":"audio"},{"name":"voice"}]` (a JSON
  array of objects; ID and timestamps are **not** emitted — names are the
  public vocabulary, IDs are internal).
- `shark tags add <name>` emits `{"name":"<name>"}` after success.
- `shark tags rename <old> <new>` emits `{"old":"<old>","new":"<new>"}`.
- `shark tags rm <name>` emits `{"name":"<name>","removed":true}`.

Error paths with `--json` MUST emit a JSON document on stderr of the form
`{"error":"<code>","message":"<text>"}` where `<code>` is one of
`unauthorized`, `not_found`, `conflict`, `in_use`, `validation`, `db_error`.
Traces to `.claude/rules/cli/patterns.md` JSON output rules.

**REQ-F-012 — CLI accessor (`cli.GetTagService`).**
`internal/cli/tag_global.go` (new file) MUST expose:

```go
func GetTagService() *services.TagService
```

The accessor MUST follow the existing lazy-init pattern (one new service
instance per invocation; panic on DB failure to match `GetTaskService`):
resolve `DB` via `cli.GetDB`, resolve the gate via `cli.GetMaintainerGate`,
construct `tag.NewTagRepository` and `tag.NewEntityTagRepository`, then
`services.NewTagService(...)`. Traces to epic architecture §4.1.

**REQ-F-013 — CLI reference documentation.**
A new page `docs/cli-reference/tags.md` MUST document the four subcommands,
the `--pass` flag, the `--force` flag on `rm`, the JSON output shapes from
REQ-F-011, and cross-references to
`docs/cli-reference/configuration.md#maintainer` (which F02 already covers).
Additionally, `docs/cli-reference/README.md` MUST be updated to list the new
`shark tags` command group. Traces to epic SC-10 (docs completeness; this
feature contributes its share).

### 1.2 Non-Functional Requirements

**REQ-NF-001 — No business logic in the command layer.**
CLI command handlers for `tags` MUST pass `make lint` with no new
`internal/repository` or `database/sql` imports. A static check at review
time verifies that no command handler:
- constructs a `TagRepository` or `EntityTagRepository`;
- reaches into `internal/auth/maintainer` directly (beyond consuming
  `*maintainer.UnauthorizedError` via `errors.As`);
- compares passwords or reads cache state;
- validates tag names (that is `TagService.ValidateName`'s job).

Traces to `.claude/rules/architecture.md` anti-patterns and
`.claude/rules/cli/patterns.md`.

**REQ-NF-002 — Observability (tracing).**
Every `TagService` public method MUST emit exactly one OpenTelemetry span
named `tag_service.<method_name>` (e.g. `tag_service.add_tag`). Spans MUST
record, at most, these attributes: `tag.name` (normalized, safe to log —
the vocabulary is public), `tag.force` (for `RemoveTag`), and the standard
outcome attribute set by `repoutil.RecordSpanError`. Spans MUST NOT record
the `--pass` value or anything derived from it. Traces to epic §6
(observability) and F02 REQ-NF-001.

**REQ-NF-003 — Testability via interfaces.**
`TagService` MUST depend on the following interfaces (all already exported
by F01 / F02), never on concrete types:
- `tag.TagRepositoryInterface`
- `tag.EntityTagRepositoryInterface`
- `maintainer.Gate`

Service tests MUST use in-file mock implementations of these three
interfaces. CLI tests MUST mock `TagService` behind a service-interface
(defined locally in the command package). Real DB MUST NOT appear in
service or CLI tests (per `.claude/rules/testing/architecture.md`).

**REQ-NF-004 — Input sanitization consistency.**
The service's `ValidateName` MUST be the **only** path by which user input
reaches `models.ValidateTagName`. F04 (attachment) is expected to call it
from `TagService.AttachByNames`. No duplicate regex, no parallel
normalization. Traces to `.claude/rules/go/input-sanitization.md` and
ADR-4.

**REQ-NF-005 — Backward compatibility.**
Adding `shark tags` MUST NOT break any existing command's argument parsing
or help output. `shark` already disallows top-level command names that
collide; a grep for `AddCommand.*tags` MUST return zero matches in
`internal/cli/commands/*.go` before this feature lands. Traces to CLAUDE.md
"CLI Commands Reference".

**REQ-NF-006 — Dual-backend neutral.**
`TagService` MUST NOT reference SQLite-specific or Turso-specific types.
All DB access flows through the F01 repositories, which are already
backend-neutral. Traces to epic constraint #5.

### 1.3 Acceptance Criteria (testable)

Each criterion is phrased so a test can pass or fail it unambiguously.

| # | Acceptance Criterion | Verification |
|---|---|---|
| AC-1 | `ListTags` returns tags in ascending name order and does NOT invoke `Gate.Authorize`. | Service unit test with a mock gate that records all calls; assert zero calls. Verifies REQ-F-002, REQ-F-004. |
| AC-2 | `AddTag(ctx, "Voice", "pw")` normalizes to `"voice"`, calls `gate.Authorize("pw")` exactly once with success, then `repo.Create("voice")`, then `gate.RecordSuccess` once. | Service unit test with call-order recording mocks. Verifies REQ-F-002, REQ-F-003, REQ-F-004. |
| AC-3 | `AddTag(ctx, "Voice!", "pw")` returns `*ValidationError` and does NOT call `gate.Authorize` **or** wait — **or** it DOES call `Authorize` first and then returns `*ValidationError`. Spec choice: authorize first, then validate (REQ-F-004 says gate is first). Test asserts `Authorize` was called once and `Create` was NOT called. | Service unit test. Verifies REQ-F-004, REQ-F-003. |
| AC-4 | `AddTag(ctx, "voice", "")` with no live cache returns the exact `*UnauthorizedError` produced by the gate, unwrapped — `errors.As(err, &unauthorized)` is true; `Create` was NOT called. | Service unit test with mock gate returning `*UnauthorizedError`. Verifies REQ-F-004, REQ-F-007. |
| AC-5 | `RemoveTag(ctx, "voice", false, "pw")` with `CountByTag = 7` returns `*TagInUseError{Name:"voice", Count:7}`; `Delete` is NOT called. | Service unit test. Verifies REQ-F-005. |
| AC-6 | `RemoveTag(ctx, "voice", true, "pw")` with `CountByTag = 7` calls `repo.Delete(id, true)` and then `gate.RecordSuccess`. | Service unit test. Verifies REQ-F-005. |
| AC-7 | `RemoveTag(ctx, "voice", false, "pw")` with `CountByTag = 0` calls `repo.Delete(id, false)` and then `gate.RecordSuccess`. | Service unit test. Verifies REQ-F-005. |
| AC-8 | `RenameTag(ctx, "voice", "audio", "pw")` where `"audio"` already exists returns `*ConflictError{Name:"audio"}`; `repo.Rename` is NOT called. | Service unit test with `GetByName("audio")` returning a tag. Verifies REQ-F-006. |
| AC-9 | `RenameTag(ctx, "voice", "voice", "pw")` returns `*ValidationError` citing that names must differ; `repo.Rename` is NOT called. | Service unit test. Verifies REQ-F-006. |
| AC-10 | `RenameTag(ctx, "voice", "audio", "pw")` success case calls `repo.Rename(id, "audio")` exactly once and does NOT call any `EntityTagRepository` method. | Service unit test with mock `EntityTagRepository` recording calls. Verifies REQ-F-006, epic SC-5. |
| AC-11 | `gate.RecordSuccess` returning an error does NOT cause any mutating method to return a non-nil error; the error is logged (observable via a test logger). | Service unit test with mock gate returning error from `RecordSuccess`. Verifies REQ-F-004. |
| AC-12 | `cli.GetTagService()` returns a `*services.TagService` whose `ListTags` works against a real `t.TempDir()` project with a migrated DB. | CLI integration test using `cli.GetDB`/`cli.ResetDB`. Verifies REQ-F-012. |
| AC-13 | `shark tags list --json` in an empty project prints `[]\n` and exits 0. | CLI test with mocked service returning empty slice. Verifies REQ-F-011. |
| AC-14 | `shark tags add voice --pass wrong` exits 3 and stderr contains `"incorrect maintainer password"` (from F02's `UnauthorizedError`). | CLI test with mocked service returning `*UnauthorizedError{Reason:"wrong_password"}`. Verifies REQ-F-010. |
| AC-15 | `shark tags add voice` (no `--pass`, no cache) exits 3 and stderr contains `"shark admin maintainer set-password"` — the `UserHint()` line is surfaced. | CLI test with mocked service returning `*UnauthorizedError{Reason:"missing_config"}`. Verifies REQ-F-010, epic SC-3. |
| AC-16 | `shark tags rm voice` with 7 uses exits 3 and stderr contains `"is in use by 7 entities"` and the text `"--force"`. | CLI test with mocked service returning `*TagInUseError{Name:"voice", Count:7}`. Verifies REQ-F-010, REQ-F-005. |
| AC-17 | `shark tags rm nonexistent --pass pw` exits 1 and stderr contains `"tag not found: nonexistent"` and lists the current vocabulary and an example `shark tags add` command. | CLI test with mocked service returning `*NotFoundError`; the command layer pulls vocabulary from its second mock call to `ListTags`. Verifies REQ-F-008, REQ-F-010. |
| AC-18 | `shark tags rename voice audio --pass pw` success path prints `Renamed voice to audio` on stdout and exits 0; with `--json` prints `{"old":"voice","new":"audio"}`. | CLI test. Verifies REQ-F-009, REQ-F-011. |
| AC-19 | `TagService` package imports (visible via `go list -f '{{.Imports}}' ./internal/services/`) include `internal/repository/tag` and `internal/auth/maintainer`, and do NOT include `internal/cli` or `github.com/spf13/cobra`. | Static test. Verifies REQ-F-001, REQ-NF-003. |
| AC-20 | The CLI commands in `internal/cli/commands/tags.go` do NOT import `internal/repository/tag`, `internal/auth/maintainer` (except for `errors.As` on `*UnauthorizedError`), or `database/sql`. | Static test using `go list` or a package-level `TestImports`. Verifies REQ-NF-001. |
| AC-21 | `docs/cli-reference/tags.md` exists and documents the four subcommands, the `--pass` flag, the `--force` flag, and the JSON shapes; `docs/cli-reference/README.md` has a link to it. | Docs review at QA gate. Verifies REQ-F-013. |
| AC-22 | `TagService` method spans, observed via an in-memory OTel recorder in a unit test, contain the tag name attribute but do NOT contain any attribute named `pass`, `password`, `hash`, or `maintainer.*`. | Unit test with `sdktrace/tracetest`. Verifies REQ-NF-002. |

### 1.4 Out of Scope (for this feature)

Explicitly deferred; not rejected, just not F03's responsibility:

- **Entity-tag attach/detach** (`--tag` flag on `shark <entity> create` /
  `update`, `shark <entity> tag add/rm`, `TagService.AttachByNames`,
  `EnforceRequired`) — F04.
- **`tag_required_for` enforcement and the corresponding config field** —
  F04 (the config field does not land in F03).
- **Tag-filtered querying** (`shark list --tag=`, `shark search --tag=`,
  `TagService.EntityIDsByTags`) — F05.
- **Viewer API and UI changes** (tag chips, tag filter, vocabulary endpoint) — F06.
- **Interactive password prompt** for any `shark tags *` command. Users rely
  on `--pass` or the F02 cache. Interactive prompting across terminals is
  its own non-trivial UX surface and is deferred.
- **`tags describe`, `tags info`, `tags usage`** (showing how many and which
  entities carry a tag) — future; not blocking v1.
- **Tag description / color metadata** — explicit out-of-scope in the epic
  (§3 out of scope; PRD O-5 deferred).
- **Hierarchical tags, aliases, merge-on-rename** — explicit out-of-scope
  in the epic (§3; PRD O-7 resolved to "hard error" by ADR-8).

---

## 2. Architecture

This section is a detailed design for F03. It aligns with the epic
architecture document (do not restate it) and specifies new files, function
signatures, and interaction flows at a level sufficient for an implementer
to begin TDD.

### 2.1 Component changes

| Change | Path | New / Modified | Notes |
|---|---|:-:|---|
| `TagService` business logic | `internal/services/tag_service.go` | new | Core deliverable |
| `TagService` typed errors | `internal/services/tag_errors.go` | new | Exported error types per REQ-F-007 |
| `TagService` unit tests | `internal/services/tag_service_test.go` | new | Mocks in-file |
| CLI command group | `internal/cli/commands/tags.go` | new | 4 subcommands |
| CLI command tests | `internal/cli/commands/tags_test.go` | new | Mocks `TagService` via a local interface |
| CLI service accessor | `internal/cli/tag_global.go` | new | `GetTagService()` — parallel to `maintainer_global.go` |
| CLI service accessor test | `internal/cli/tag_global_test.go` | new | Integration smoke test |
| Docs — command reference | `docs/cli-reference/tags.md` | new | Per REQ-F-013 |
| Docs — README index | `docs/cli-reference/README.md` | modified | Add a row to the "Advanced / Entity Commands" table and a link under the group "Vocabulary" |
| Interface for CLI tests | `internal/cli/commands/tags.go` (local `tagServiceIface`) | new (internal) | Allows the command handlers to be constructed with a mock per `admin_maintainer.go` precedent |

No edits to F01's `internal/repository/tag/*` are required. No edits to
F02's `internal/auth/maintainer/*` are required. No new migrations — F01
already shipped the schema.

### 2.2 Data model changes

**None.** F01 already created `tags` and `entity_tags`. F03 does not
introduce or alter any tables, columns, indexes, triggers, or constraints.
No `.sharkconfig.json` schema change (the `tag_required_for` field is an
F04 concern). `CurrentSchemaVersion` remains at whatever F01 bumped it to;
F03 does not bump it.

### 2.3 Interfaces and contracts

Concrete service signatures. Types live where the table in §2.1 indicates.

```go
// internal/services/tag_service.go

package services

import (
    "context"
    "github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
)

// TagService owns the vocabulary business rules. CLI and (future) HTTP
// handlers consume it; it consumes F01 repositories and the F02 gate.
type TagService struct {
    tagRepo       tag.TagRepositoryInterface
    entityTagRepo tag.EntityTagRepositoryInterface
    gate          maintainer.Gate
}

// NewTagService constructs a TagService. All dependencies are required;
// the constructor panics on nil per the architecture's constructor rules.
func NewTagService(
    tagRepo tag.TagRepositoryInterface,
    entityTagRepo tag.EntityTagRepositoryInterface,
    gate maintainer.Gate,
) *TagService

// ValidateName normalizes input (trim + lowercase) and validates the
// result against models.ValidateTagName. Returns the normalized form on
// success, or *ValidationError on failure. This is the single entry point
// for name validation; F04's AttachByNames will consume it.
func (s *TagService) ValidateName(raw string) (string, error)

// ListTags returns the full vocabulary, ordered by name ascending.
// Open to all users — no gate invocation.
func (s *TagService) ListTags(ctx context.Context) ([]*models.Tag, error)

// AddTag normalizes name, authorizes, creates the tag, and records success.
// Returns *ValidationError / *UnauthorizedError / *ConflictError /
// *RepositoryError as applicable.
func (s *TagService) AddTag(ctx context.Context, name, providedPass string) (*models.Tag, error)

// RemoveTag normalizes name, authorizes, looks up the tag, enforces the
// in-use policy (ADR-9), deletes, and records success.
func (s *TagService) RemoveTag(ctx context.Context, name string, force bool, providedPass string) error

// RenameTag normalizes both names, authorizes, pre-checks collision,
// renames, and records success. Returns the updated tag.
func (s *TagService) RenameTag(ctx context.Context, oldName, newName, providedPass string) (*models.Tag, error)
```

Typed errors live in `internal/services/tag_errors.go`:

```go
package services

type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string { /* ... */ }

type NotFoundError struct {
    Name string
}

func (e *NotFoundError) Error() string { /* ... */ }

type ConflictError struct {
    Name string
}

func (e *ConflictError) Error() string { /* ... */ }

type TagInUseError struct {
    Name  string
    Count int64
}

func (e *TagInUseError) Error() string { /* ... */ }
```

Note: these error types are **tag-scoped** (named in the context of the tag
service). If future services need overlapping names, scope them per-service
rather than promoting to a shared package. This aligns with the existing
practice in `internal/services` where per-service errors are common.

### 2.4 CLI command surface

File: `internal/cli/commands/tags.go`. Structure mirrors
`internal/cli/commands/admin_maintainer.go` (the F02 precedent).

```
shark tags
├── list            (no auth)
├── add <name>      --pass <value>
├── rm <name>       --pass <value> --force
└── rename <o> <n>  --pass <value>
```

Each subcommand:
1. Parses arguments (`cobra.ExactArgs(1)` / `cobra.ExactArgs(2)`).
2. Parses flags (`--pass`, `--force`).
3. Obtains the service: `svc := cli.GetTagService()`.
4. Calls the appropriate service method.
5. Maps errors via the table in REQ-F-007 and REQ-F-010.
6. Prints success output (plain or JSON per REQ-F-011).

To enable CLI tests with mocks (REQ-NF-003), each subcommand's handler is
constructed by a helper `newTagsListCmd(svc tagServiceIface) *cobra.Command`
(etc.), where `tagServiceIface` is a package-local interface whose methods
mirror the service's public surface. The production command wires the real
`GetTagService()`; tests pass a mock. This is the same pattern
`admin_maintainer.go` uses with `newAdminMaintainerSetPasswordCmd`.

Exit codes are written via `os.Exit` in a small helper (consistent with
existing command idioms; see the error-handling rule table). Handlers
prefer `return err` and let a common post-run translator map the typed
error to an exit code, minimizing `os.Exit` fanout in handler bodies.

### 2.5 CLI accessor wiring

File: `internal/cli/tag_global.go` (new; parallel to `maintainer_global.go`).

```go
package cli

import (
    "context"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
    "github.com/jwwelbor/shark-task-manager/internal/services"
)

// GetTagService returns a *TagService wired to the current project's DB
// and maintainer gate. Panics on DB failure to match GetTaskService.
func GetTagService() *services.TagService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database for tag service: %v", err))
    }
    tagRepo := tag.NewTagRepository(db)
    entityTagRepo := tag.NewEntityTagRepository(db)
    gate := GetMaintainerGate() // already validated by F02
    return services.NewTagService(tagRepo, entityTagRepo, gate)
}
```

The accessor creates a new service instance per call (matches
`GetTaskService` pattern), shares the global DB, and constructs a fresh
gate (per F02's "new instance per call" contract).

### 2.6 Interaction flow — `shark tags add <name> --pass P`

Sequence (one happy path, one auth failure):

```
User                    CLI handler             TagService               Gate                    TagRepository
 │                          │                       │                       │                       │
 │ add voice --pass P       │                       │                       │                       │
 ├─────────────────────────>│                       │                       │                       │
 │                          │ GetTagService()       │                       │                       │
 │                          │ svc.AddTag(ctx,       │                       │                       │
 │                          │   "voice","P")        │                       │                       │
 │                          ├──────────────────────>│                       │                       │
 │                          │                       │ gate.Authorize("P")   │                       │
 │                          │                       ├──────────────────────>│                       │
 │                          │                       │ nil                   │                       │
 │                          │                       │<──────────────────────┤                       │
 │                          │                       │ ValidateName("voice") │                       │
 │                          │                       │ → "voice", nil        │                       │
 │                          │                       │ repo.Create("voice")  │                       │
 │                          │                       ├───────────────────────────────────────────────>
 │                          │                       │ *models.Tag, nil      │                       │
 │                          │                       │<───────────────────────────────────────────────
 │                          │                       │ gate.RecordSuccess()  │                       │
 │                          │                       ├──────────────────────>│                       │
 │                          │                       │ nil                   │                       │
 │                          │ *models.Tag, nil      │<──────────────────────┤                       │
 │                          │<──────────────────────┤                       │                       │
 │ "Added tag voice"        │                       │                       │                       │
 │<─────────────────────────┤                       │                       │                       │
```

Auth failure early-returns after `Authorize`:
```
svc.AddTag → gate.Authorize("wrong") → *UnauthorizedError{Reason:"wrong_password"}
                                     → service returns err unwrapped
CLI prints "incorrect maintainer password" → exit 3.
```

### 2.7 Integration with existing code

**No edits** to F01 or F02 source files are required by this feature. The
only modifications to existing files are:

- `docs/cli-reference/README.md` — add a link/row for the new `shark tags`
  group (REQ-F-013).
- `internal/cli/services_global.go` — **optional**: add an `init()`-level
  import or a comment pointing at `tag_global.go`. No code change required;
  the new file registers its accessor via package-level declaration.

Potentially affected but **not edited**:

- `internal/cli/commands/admin_maintainer.go` — F02 ships the
  `set-password` command. F03 does not touch it; it only **consumes** the
  gate via `cli.GetMaintainerGate()`.
- Entity-specific services (`TaskService`, `FeatureService`, etc.) — F03
  does not touch them. F04 will.

### 2.8 Key technical decisions (F03-local)

Most design decisions are inherited from the epic ADRs. This section
records decisions that are local to F03 and not addressed at the epic
level.

**D1. Authorize-before-validate order.**
`AddTag` / `RemoveTag` / `RenameTag` call `Authorize` before `ValidateName`.
Rationale: an unauthenticated caller must not be able to probe the name
validator (which would leak "which names are valid" and could support
enumeration under a weaker future threat model). `Authorize` is
constant-time with respect to the password (F02 REQ-NF-002), so this adds
no timing oracle. Cost: a user who types a wrong password AND a wrong name
sees the auth error first; they will see the validation error on the next
attempt. Acceptable. REQ-F-004 encodes this order.

**D2. Typed errors defined per-service, not in a shared package.**
`ValidationError`, `NotFoundError`, `ConflictError`, `TagInUseError` are
all defined in `internal/services/tag_errors.go`, not in a generic
`internal/errors` package. Rationale: existing services follow this
pattern (e.g., `workflow.TransitionError`, `taskcreation.InvalidKeyError`),
and overlap with other services' `NotFoundError` is accepted — callers
always use the service-scoped type via `errors.As`. Consolidating into a
shared package is a refactor best done when >2 services exhibit the same
shape.

**D3. CLI error-to-exit-code mapping uses a single post-run translator.**
Instead of scattering `os.Exit(N)` across handlers, each handler returns
its error to Cobra and a `PersistentPostRunE` (or the existing cli error
wrapper) maps the typed error to the exit code per REQ-F-010. This matches
the cleanup-hook concern raised in `.claude/rules/services/cli-integration.md`
("prefer returning typed errors over calling os.Exit() directly"). If the
existing wrapper does not yet support this shape, the F03 task that
implements the CLI layer adds a small translator shim inside the `tags`
command package (kept local, not promoted to a shared helper until a
second caller wants it).

**D4. Command-package local `tagServiceIface` for test mocking.**
CLI tests mock the service via a local interface (REQ-NF-003, AC-13–AC-18,
F03 §2.4). The interface is unexported and lives in `tags.go`. Rationale:
mirrors F02's `maintainerBootstrapServiceIface` precedent and keeps test
seams at the boundary of the command package without polluting the
services package's public surface.

**D5. `RemoveTag` argument order is `(ctx, name, force, providedPass)` — not `(ctx, name, providedPass, force)`.**
Rationale: `force` is a behavior-changing flag that pairs semantically with
the name; `providedPass` is an auth concern that applies uniformly across
every mutating method. Grouping the business parameters first (name,
force) and the auth parameter last makes the signature easier to read at
the call site. REQ-F-005 uses this order and AC-5/AC-6/AC-7 verify it.

### 2.9 Testing approach

Per `.claude/rules/testing/architecture.md`:

| Suite | Dependencies | What it verifies |
|---|---|---|
| Service unit tests (`tag_service_test.go`) | All mocked: `tag.TagRepositoryInterface`, `tag.EntityTagRepositoryInterface`, `maintainer.Gate` | REQ-F-002..008, AC-1..AC-11, AC-22 |
| CLI command tests (`tags_test.go`) | Mocked `tagServiceIface` | REQ-F-009..011, AC-13..AC-18, AC-20 |
| Accessor integration test (`tag_global_test.go`) | Real DB under `t.TempDir()`, real config | AC-12, REQ-F-012 |
| Static / import tests | `go list` / `go vet` harness | AC-19, AC-20 |

A small amount of shared test scaffolding may be added — e.g., a mock
`Gate` in `internal/services/mocks_test.go` — but NEW mocks MUST use the
function-field style defined in `.claude/rules/services/testing.md` (no
testify mocks, no framework).

No real-DB tests for `TagService`. No real-DB tests for CLI handlers.

---

## 3. Traceability

This spec's requirements trace back to the epic PRD as follows:

| Epic PRD | F03 Requirements | F03 Acceptance Criteria |
|---|---|---|
| SC-1 (register + apply + retrieve) | REQ-F-002 (AddTag); partially — attach/retrieve is F04/F05 | AC-2, AC-13 |
| SC-2 (helpful errors on unregistered) | REQ-F-008 (error UX) | AC-17 |
| SC-3 (non-maintainer blocked) | REQ-F-004, REQ-F-007 | AC-4, AC-15 |
| SC-4 (sudo cache) | REQ-F-004 (consumes F02's cache; test in F02) | AC-4 (gate integration) |
| SC-5 (rename without migration) | REQ-F-006 | AC-10 |
| SC-7 (schema unchanged) | None — F01 owns | (verified by F01) |
| SC-9 (reusable gate) | REQ-F-004 (consumes gate interface only) | AC-19 |
| SC-10 (docs) | REQ-F-013 | AC-21 |
| UAT-1 (register + tag 6 entities) | REQ-F-002 `AddTag` portion only | AC-2, AC-13 |
| UAT-2 (unregistered error text) | REQ-F-008 | AC-17 |
| UAT-3 (no password fails with guidance) | REQ-F-004, REQ-F-010 | AC-15 |
| UAT-4 (cache window works) | REQ-F-004 (consumes F02) | AC-4 (delegated) |
| UAT-5 (rename is atomic) | REQ-F-006 | AC-10 |
| UAT-8 (gate is reusable) | REQ-F-004, REQ-F-010, REQ-F-012 | AC-19, AC-20 |

Decisions inherited without restatement:
- ADR-2 → REQ-F-004
- ADR-4 → REQ-F-003
- ADR-8 → REQ-F-006
- ADR-9 → REQ-F-005

---

## 4. Risks and mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|:-:|:-:|---|
| F02's `Gate` interface surface does not match REQ-F-010 assumptions | low | high | F02 is merged; interface is verified in §2.3. If F02 changes, F03 task spec is revisited. |
| `RecordSuccess` errors are noisy under filesystem pressure | medium | low | REQ-F-004 logs and swallows; F02 REQ-F-003 already specifies best-effort. |
| CLI post-run error translator conflict with existing `cli.Error`-style idioms | medium | low | D3 keeps the translator local to the `tags` command package; no cross-cutting change. |
| Accessor pattern drift from `maintainer_global.go` | low | low | §2.5 shows the exact shape; reviewer check at code review. |
| Naming collision of service-scoped typed errors with other services' error types | low | low | D2 accepts this; callers use `errors.As` with the specific typed pointer. |

---

## 5. Exit gate (for reviewer)

- [x] Every requirement in §1.1 / §1.2 has a unique ID and at least one AC.
- [x] Every AC in §1.3 is testable (observable outcome, not a design note).
- [x] Every architecture decision in §2.8 references an existing pattern
  or explicitly explains deviation.
- [x] File paths are listed for every code change in §2.1.
- [x] No TBDs in sections 1.1, 1.2, 1.3, 2.1, 2.3.
- [x] Scope is incremental: no restatement of epic PRD business context,
  no restatement of epic architecture decisions beyond pointers.
- [x] F01 / F02 dependencies are explicit; F04 / F05 / F06 dependencies
  are explicit as out-of-scope.

---

*Last Updated*: 2026-04-23
