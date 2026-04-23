---
epic_key: E28
document_type: architecture
title: E28 Architecture — Entity Tagging with Managed Vocabulary
---

# E28 Architecture — Entity Tagging with Managed Vocabulary

This document defines the technical design for Epic E28. It is the source of
truth for architectural decisions; feature PRDs and task specs under E28 MUST
reference this document rather than restating its content. PRD is at
`docs/plan/E28-entity-tagging-with-managed-vocabulary/epic.md`.

---

## 1. Component Overview

E28 introduces three new concerns to the codebase: (a) a closed tag vocabulary
with polymorphic associations, (b) a reusable maintainer authorization gate,
and (c) read-only viewer surface-area for tags. Everything else (entity CRUD,
workflow, sync, file I/O) is untouched.

### 1.1 What changes

| Layer | Change | Package / file |
|---|---|---|
| Schema | Two new tables: `tags`, `entity_tags`. Cascade-delete triggers per parent entity table. Schema version bump. | `internal/db/db.go` (migration function, version constant) |
| Models | `Tag` domain type. `EntityTagLink` for join-table rows. Reuse existing `models.EntityType`. | `internal/models/tag.go` (new) |
| Repository | `TagRepository` (vocabulary CRUD) and `EntityTagRepository` (polymorphic links) under `internal/repository/tag/`. Follows the `entitydoc` / `note` package shape. | `internal/repository/tag/` (new) |
| Service | `TagService` owns all tag business logic (vocabulary admin, attach/detach, enforcement of `tag_required_for`, error shaping). | `internal/services/tag_service.go` (new) |
| Service | `MaintainerGate` (reusable, general-purpose) owns password verification and sudo-style cache. `TagService` is the first consumer. | `internal/auth/maintainer/` (new package) |
| CLI | New command group `shark tags` with `list`, `add`, `rm`, `rename`. `--tag` flag added to existing `create`/`update` commands for all six entity types. New subcommand per entity type: `shark <entity> tag add|rm`. `--tag=` filter added to `shark list` and `shark search`. | `internal/cli/commands/tags.go` (new) plus edits to each entity command file |
| Config | New top-level `.sharkconfig.json` fields: `tag_required_for` (array of entity-type strings) and `maintainer` (nested object containing the password or a pointer to it). Exposed via `config.Config`. | `internal/config/config.go` |
| Viewer API | Entity responses gain a `tags` array. List endpoints accept a `tag` query parameter. New read-only endpoint exposes the vocabulary. | `internal/api/viewer/handler.go`, `internal/services/viewer_service.go` |
| Viewer UI | Tag chips on entity detail views; tag filter control on list views. | `internal/viewer/` |
| Docs | New `docs/cli-reference/tags.md`. Updates to `configuration.md`, `create`, `update`, `list`, `search` pages. | `docs/cli-reference/` |

### 1.2 What stays

- Existing entity schemas (`epics`, `features`, `tasks`, `bugs`, `change_cards`, `ideas`) are **not altered**. Tags attach by polymorphic association, not by a new column on each entity table.
- Existing polymorphic patterns (`entity_notes`, `entity_documents`, `entity_history`, `entity_relationships`) are **reused, not modified**. E28 adds a parallel polymorphic table; it does not refactor the existing ones.
- `models.EntityType` enum is reused. If `idea` is not yet a member (it is not, per `internal/models/entity_note.go`), adding it is treated as incidental infrastructure within E28 — a one-line enum addition plus a new `EntityTypeIdea: true` entry in the `validEntityTypes` map. This is the minimum surface required for `entity_tags.entity_type = 'idea'` to be valid.
- Workflow engine (`internal/services/workflow.Service`), status calculation, slug lookup, `fileops` writer, and file sync are untouched.

### 1.3 Service layering (per `.claude/rules/architecture.md`)

```
CLI (commands/tags.go, commands/task.go, ...)
        │
        ▼
TagService ─── uses ──► MaintainerGate (reusable, in internal/auth/maintainer)
        │                      │
        │                      ▼
        │                  CacheStore (file-backed, project-scoped)
        ▼
TagRepository + EntityTagRepository
        │
        ▼
sqlite / turso (entity_tags, tags)
```

- CLI commands are thin wrappers. No vocabulary validation, no password checks, no cache logic inline. They only: parse flags, call `TagService` or `MaintainerGate`, format output.
- `TagService` owns every business rule: normalization, "unregistered tag" error composition, `tag_required_for` enforcement, uniqueness, and the **decision** to invoke the gate. It **does not implement** the gate.
- `MaintainerGate` owns every access-control rule: password matching, cache lookup, cache write on success, expiry. It exposes a small API (`Authorize(ctx) error`) that any future admin command can consume.
- Repositories own SQL only. No enforcement of `tag_required_for`, no sudo cache, no password comparison in SQL.

---

## 2. Key Technical Decisions (ADRs)

Each decision is stated with the problem, alternatives considered, the choice, and the rationale. These are normative — tasks deviating from these decisions require a new ADR.

### ADR-1: Two tables, polymorphic join — not per-entity-type join tables

**Decision.** Introduce exactly two new tables: `tags` (vocabulary) and `entity_tags` (polymorphic `(entity_type, entity_id, tag_id)` join).

**Alternatives considered.**
(a) Six per-entity join tables (`epic_tags`, `feature_tags`, `task_tags`, `bug_tags`, `change_card_tags`, `idea_tags`) with real FKs.
(b) A single JSON `tags` column on each entity table.
(c) The proposed polymorphic pair.

**Rationale.** The epic's constraint #2 and SC-7 make this mandatory, but the decision stands on its own merits: the codebase already uses this pattern four times (`entity_notes`, `entity_documents`, `entity_history`, `entity_relationships`), so the cost of a new polymorphic table is one `CREATE TABLE`, one set of triggers, and one repository — not a new pattern to learn. Alternative (a) multiplies schema surface by six, requires six repos, six migrations, and breaks the SC-7 schema review. Alternative (b) is a dead-end for filtering at SQL level (`WHERE tags LIKE '%voice%'` is unsafe and indexless).

**Cost.** Polymorphic tables cannot use SQL `FOREIGN KEY` to enforce that `entity_id` is valid for the given `entity_type`. This is mitigated by cascade-delete **triggers** per parent table (as `entity_notes` already does), and by the repository writing `(entity_type, entity_id)` only through an API that resolves the parent key first.

### ADR-2: Vocabulary modification via a reusable maintainer gate, not a tag-specific check

**Decision.** Build the gate as a general-purpose package (`internal/auth/maintainer`) with its own `MaintainerGate` type. `TagService` consumes the gate via dependency injection; it does not embed the password comparison, cache reading, or cache writing.

**Alternatives considered.**
(a) Password check inline inside each `TagService.AddTag` / `RenameTag` / `RemoveTag` method.
(b) A private helper inside the tag service package.
(c) A reusable package as chosen.

**Rationale.** SC-9 and UAT-8 require reusability as a hard acceptance criterion, but the independent reason is that the gate is an **orthogonal cross-cutting concern** — it has nothing to do with tags semantically. Keeping it in `internal/auth/maintainer` lets a future `shark admin purge` or `shark admin reset-schema` consume it without a tag package import. The package has no dependency on tags, entities, or any shark-specific model.

**Interface (normative).**
```go
package maintainer

type Gate interface {
    // Authorize verifies either an explicit password (from --pass) or a live
    // cache entry. Returns nil if authorized, or an *UnauthorizedError whose
    // message tells the user how to obtain the password.
    Authorize(ctx context.Context, providedPass string) error

    // RecordSuccess writes a cache entry with the current timestamp so that
    // subsequent gated commands within the window succeed without --pass.
    RecordSuccess(ctx context.Context) error
}
```

The typical service-layer call shape:
```go
if err := s.gate.Authorize(ctx, input.Password); err != nil { return nil, err }
// ... perform the admin operation ...
_ = s.gate.RecordSuccess(ctx) // intentionally non-fatal; gate cache is best-effort
```

### ADR-3: Cache location is `~/.cache/shark/<project-hash>/maintainer.session`

**Decision.** The sudo-style cache is stored in a user-scoped cache directory, namespaced by a stable hash of the absolute project root path.

**Alternatives considered.**
(a) Project-local `.shark/session` file.
(b) `/tmp/shark-<uid>-<project>/`.
(c) `~/.cache/shark/<project-hash>/` as chosen.

**Rationale.** Addresses PRD open question O-3. Alternative (a) leaks a timestamp into the project directory and trips git-ignore discipline; it also has permissions issues on shared repos. Alternative (b) resets on reboot and may leak across users on multi-user systems. Alternative (c) respects XDG conventions (`XDG_CACHE_HOME` if set, falling back to `~/.cache/`), is private by default (mode 0700), is per-project, and survives reboots. The project hash prevents cross-project cache reuse when a user works on multiple shark projects.

**Cache entry format.** Opaque JSON: `{"last_success": RFC3339, "pass_hash": hex}` (see ADR-6). File is rewritten on each `RecordSuccess`, not appended.

**Cache window.** 60 seconds, fixed in v1. `MaintainerGate` accepts a `Window time.Duration` in its constructor so the window is tunable for tests and future configuration without a v1 release.

### ADR-4: Tag names are lowercase-ASCII slugs, validated by allowlist regex

**Decision.** Tag names match `^[a-z0-9][a-z0-9-]{0,63}$` — lowercase ASCII, digits, hyphens only; must start with a letter or digit; max 64 characters. Input is lowercased via `strings.ToLower` before validation.

**Alternatives considered.**
(a) Unicode-permissive (NFC-normalize, accept any letter).
(b) Case-preserving ("Voice" distinct from "voice").
(c) Lowercase-ASCII allowlist as chosen.

**Rationale.** Consistent with `.claude/rules/go/input-sanitization.md` (regex allowlists, anchored, `strings.TrimSpace`, explicit length limit). Case-insensitivity is a hard requirement of the PRD (vocabulary drift is the stated problem); accepting mixed case and then collapsing at query time silently re-introduces drift on the display side. Pure lowercase-ASCII is the cheapest choice that is interoperable with URL slugs (viewer query parameter), shell arguments, and future search tooling. Unicode can be added later as an expansion of the allowlist without breaking existing rows.

**Display name is the canonical name.** No separate `display_name` column in v1 (defers PRD O-5). Per SC-5/UAT-5, `rename` updates only `tags.name`; entity_tag rows are immutable through a rename. This is the "one row only" guarantee.

### ADR-5: Multi-tag filter is AND (conjunction)

**Decision.** `shark list --tag=a --tag=b` returns entities tagged with **both** `a` AND `b`. OR semantics and configurable semantics are deferred.

**Alternatives considered.**
(a) OR: entities with any of the supplied tags.
(b) Configurable via `--tag-op=and|or`.
(c) AND as chosen.

**Rationale.** Addresses PRD open question O-1. AND is the more useful default for cross-cutting views ("voice AND auth" = the intersection, which is the motivating use case in §1 of the PRD). OR is readily expressed by two separate runs today and by UNION in a future follow-up. Defaulting to AND also matches user intuition from file-system find (`find -tag a -tag b` is an AND). `--tag-op=or` can be added later without breaking the existing behavior.

**SQL shape.**
```sql
SELECT e.*
FROM <entity> e
WHERE NOT EXISTS (
    SELECT 1 FROM unnest(:required_tags) rt
    WHERE NOT EXISTS (
        SELECT 1 FROM entity_tags et
        JOIN tags t ON t.id = et.tag_id
        WHERE et.entity_type = :etype AND et.entity_id = e.id AND t.name = rt
    )
)
```
SQLite lacks `unnest`, so the repository generates the query as N `EXISTS` clauses, one per tag, joined with AND. The `entity_tags(entity_type, entity_id)` composite index from the schema section keeps this planar.

### ADR-6: Password stored hashed in `.sharkconfig.json`; cache stores the same hash

**Decision.** The `.sharkconfig.json` `maintainer.password_hash` field stores a SHA-256 hex digest of the password. `MaintainerGate` hashes the provided `--pass` value and compares hex digests. The cache file stores the same digest so a cache-hit does not require reading `.sharkconfig.json` again on a subsequent gated command.

**Alternatives considered.**
(a) Plaintext password in `.sharkconfig.json`.
(b) Hashed + salted with bcrypt/scrypt/Argon2.
(c) SHA-256 hex digest (unsalted) as chosen.

**Rationale.** Addresses PRD open question O-2. The PRD is explicit about the threat model: "prevent accidental / casual-LLM-agent modification, not withstand a targeted adversary with filesystem access." Under this model, a plaintext password (a) is intelligible to any agent or bystander who opens the file, which directly violates the "prevent casual LLM-agent modification" goal — an agent that can read `.sharkconfig.json` has the password. SHA-256 (c) defeats read-and-type attacks while avoiding the dependency and cost of (b), which would be overkill for a threat model that explicitly excludes filesystem adversaries. If a future epic raises the threat level, migrating to Argon2 is a one-field addition (`password_hash_algo: "argon2id"`) — the existing gate interface does not need to change.

**Bootstrap ergonomic.** Because the user types a plaintext password but must store a hash, we ship `shark admin maintainer set-password` which accepts a plaintext password and writes the digest. No user ever types a hash.

### ADR-7: `tag_required_for` is enforced at service layer, not repository or schema

**Decision.** Enforcement lives in the create paths of `TaskService.CreateTask`, `FeatureService.CreateFeature`, `EpicService.CreateEpic`, `BugService.CreateBug`, `ChangeCardService.Create`, and `IdeaService.Create`. Each service calls `TagService.EnforceRequired(ctx, entityType, tags)` before writing. `entity_tags` schema has **no** NOT NULL "at least one tag" constraint.

**Alternatives considered.**
(a) Database CHECK constraint or trigger enforcing `≥1 tag` for listed entity types.
(b) Repository-layer check on `Create` that inspects config.
(c) Service-layer check as chosen.

**Rationale.** Consistent with `.claude/rules/go/patterns.md` §Validation: business rules live in services, not repositories, and never in the schema. The schema cannot know about a config file, and a trigger would require per-type duplication and a rebuild on config change. Repository (b) would require every repo to import `config`, which creates a circular-dependency risk (`repository → config → db.DatabaseConfig → repository` already has tension). Service (c) keeps the check where all other business rules live and lets a single service method be mocked in tests.

### ADR-8: Rename is atomic, IDs immutable, cascade is implicit

**Decision.** `shark tags rename voice audio` updates `tags.name` in place via `UPDATE tags SET name = ? WHERE id = ?`. No rows in `entity_tags` are touched. On collision (a tag named `audio` already exists), the command fails with a typed error; merge is not offered in v1.

**Alternatives considered.**
(a) Rename = delete+reinsert with per-row re-linking.
(b) Rename + automatic merge on collision.
(c) Rename with hard-error on collision as chosen.

**Rationale.** Addresses PRD open questions O-5 and O-7. SC-5/UAT-5 **requires** that entity_tag rows are not rewritten — this is observable ("inspect `entity_tags` confirms the same row IDs exist"). Alternative (a) violates SC-5. Alternative (b) is ambiguous (does the merged tag's metadata win? what about the usage history?) and introduces a destructive operation disguised as a rename. Alternative (c) is the safest default; users who genuinely want to merge can do it in two commands once a future merge command ships.

### ADR-9: Removing a tag in use requires `--force`

**Decision.** `shark tags rm voice` with existing `entity_tags` rows referring to `voice` fails with `ErrTagInUse` listing the usage count. `--force` causes the repository to delete the `entity_tags` rows in the same transaction as the `tags` row.

**Alternatives considered.** (a) Cascade by default. (b) Always block. (c) Block unless `--force` as chosen.

**Rationale.** Addresses PRD O-6. Silent cascade (a) is a footgun (a mistake on a widely-used tag wipes every entity's association). Always-block (b) means the only path to remove a tag is a hand-written `shark list --tag=X` loop followed by `shark <entity> tag rm` per entity, which is tedious and error-prone. The `--force` pattern (c) is consistent with existing shark conventions (`shark delete --force`) and makes the destructive intent explicit.

### ADR-10: Idea gets first-class `EntityType` membership

**Decision.** The existing `models.EntityType` enum, which currently enumerates `epic`, `feature`, `task`, `change`, `bug`, `tech_debt`, gains `idea`. The `validEntityTypes` map is extended accordingly. This is the minimum incidental schema-adjacent change required by E28.

**Alternatives considered.**
(a) Keep `idea` out of the enum; handle it with a second enum parallel to `EntityType`.
(b) Add `idea` to the enum as chosen.

**Rationale.** The PRD requires six entity types with uniform behavior. (a) creates a second category of polymorphism ("entities that can be noted" vs. "entities that can be tagged") which will rot; (b) unifies the model. The CHECK constraint on `entity_notes.entity_type` does **not** need to grow for E28 — entity_tags has its own (looser, for now, or identical, to be decided during decomposition) CHECK constraint. Either way, the model-level enum is the source of truth for allowed `entity_type` values at the service layer.

---

## 3. Data Model Changes

### 3.1 New tables

```sql
-- Vocabulary registry. One row per named tag.
CREATE TABLE IF NOT EXISTS tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
CREATE INDEX IF NOT EXISTS idx_tags_created_at ON tags(created_at);

-- Polymorphic join.
CREATE TABLE IF NOT EXISTS entity_tags (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL CHECK (entity_type IN
        ('epic', 'feature', 'task', 'bug', 'change', 'idea')),
    entity_id   INTEGER NOT NULL,
    tag_id      INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_type, entity_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_entity_tags_entity
    ON entity_tags(entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_entity_tags_tag
    ON entity_tags(tag_id);

CREATE INDEX IF NOT EXISTS idx_entity_tags_tag_entity
    ON entity_tags(tag_id, entity_type);
```

Index rationale:
- `(entity_type, entity_id)` supports "show all tags on this entity" (entity-detail views, viewer API).
- `(tag_id)` supports "list all entities with this tag" (`tags rm --force` usage-count query, `list --tag=` primary path).
- `(tag_id, entity_type)` supports "list all bugs with this tag" (tag-filtered per-entity-type lists).

### 3.2 Cascade-delete triggers

Because polymorphic tables cannot use `FOREIGN KEY` to the parent entity, E28
creates one trigger per parent table. The pattern mirrors the existing
`entity_notes_cascade_delete_*` triggers in `internal/db/db.go`.

```sql
CREATE TRIGGER IF NOT EXISTS entity_tags_cascade_delete_epic
    AFTER DELETE ON epics
    FOR EACH ROW BEGIN
        DELETE FROM entity_tags WHERE entity_type = 'epic' AND entity_id = OLD.id;
    END;
-- ... and one each for features, tasks, bugs, change_cards, ideas.
```

Six triggers total. Same ordering constraint as existing cascade triggers:
the trigger must exist before any delete on the parent table in a migration.

### 3.3 Config schema additions

New top-level fields in `.sharkconfig.json`:

```json
{
  "tag_required_for": ["task"],

  "maintainer": {
    "password_hash": "a1b2c3...",
    "cache_window_seconds": 60
  }
}
```

In `internal/config/config.go` (`Config` struct):

```go
TagRequiredFor []string          `json:"tag_required_for,omitempty"`
Maintainer     *MaintainerConfig `json:"maintainer,omitempty"`
```

`MaintainerConfig` lives in `internal/config/maintainer.go`:

```go
type MaintainerConfig struct {
    PasswordHash       string `json:"password_hash,omitempty"`
    CacheWindowSeconds int    `json:"cache_window_seconds,omitempty"` // default 60
}
```

### 3.4 No changes to existing tables

`epics`, `features`, `tasks`, `bugs`, `change_cards`, `ideas` are unchanged.
No new columns, no altered CHECK constraints, no altered indexes. This is the
critical property that makes E28 a pure additive migration.

### 3.5 Migration strategy

Per `.claude/rules/database-critical.md`:

1. **New migration function** `migrateAddTagsAndEntityTags(db *sql.DB)` in
   `internal/db/db.go`. Checks whether each of `tags`, `entity_tags`, and the
   six cascade triggers already exist; creates only what is missing. Idempotent.
2. **Bump `CurrentSchemaVersion`** from 13 → 14 in the same PR.
3. **Developer callout** in the migration PR description: "This change adds a
   migration. Set `skip_migrations: false` in `.sharkconfig.json` before
   running the next shark command, then set it back to `true`." Added to the
   task spec that introduces the schema change.
4. **Dual-backend verified**: the SQL is `sqlite3`-only (no Turso-specific
   features). `CREATE TRIGGER IF NOT EXISTS` works on both.
5. **No data to migrate.** Tags is a brand-new concept — no legacy rows exist
   anywhere. This is simpler than the `entity_notes` migration, which had to
   copy data out of `task_notes`.

---

## 4. Integration Approach

### 4.1 Service wiring (CLI)

New accessors in `internal/cli/services_global.go`:

```go
func GetMaintainerGate() *maintainer.Gate {
    projectRoot, _ := cli.ProjectRoot()
    cfg := GetConfig()
    return maintainer.NewGate(projectRoot, cfg.Maintainer)
}

func GetTagService() *services.TagService {
    db, _ := GetDB(context.Background())
    tagRepo := tag.NewTagRepository(db)
    entityTagRepo := tag.NewEntityTagRepository(db)
    cfg := GetConfig()
    gate := GetMaintainerGate()
    return services.NewTagService(tagRepo, entityTagRepo, cfg, gate)
}
```

Follows the existing lazy-per-call pattern of `GetTaskService()` etc.

### 4.2 Service wiring (HTTP / viewer)

In `cmd/server/services.go`, the `WireServices` function adds `TagService` and
injects it into the viewer handler alongside existing services. The viewer
handler does **not** take the `MaintainerGate`; v1 vocabulary management is
CLI-only (PRD scope).

### 4.3 Integration with existing entity commands

Three touch points per entity command:

1. **Create.** `--tag` repeated flag → collected into `[]string` → passed to
   the entity service. The entity service calls
   `tagSvc.EnforceRequired(ctx, entityType, tags)` before validating, then
   `tagSvc.AttachMany(ctx, entityType, entityID, tags)` after the row exists.
2. **Update.** `--tag` on `update` **adds** tags (idempotent via the UNIQUE
   constraint). Removal is explicit: `shark <entity> tag rm <key> <name>`.
3. **List.** `--tag` repeated flag → `[]string` → additional WHERE clauses in
   the entity's list query. See ADR-5 SQL shape.

This keeps the `update` semantics safe: no user ever loses tags by forgetting
to repeat them on update. (The alternative — treat `--tag` on update as
"replace the tag set" — is a trap; shark already does not do this for
dependencies, file_path, or agent_type, and consistency matters.)

### 4.4 Integration with `shark search`

`shark search <query> --tag=<name>` intersects full-text matches with the
tagged-entity set. The existing search pipeline is untouched; `TagService`
provides a method `EntityIDsByTags(ctx, entityType, tagNames, op=AND)` that
the search service consumes to filter its output.

### 4.5 Integration with the viewer (E27)

**API.** Existing handlers that return an entity (`Summary`, `Hierarchy`
items, `FeatureTasks`) gain a `tags []string` field on each entity response
DTO. The field is always present (`[]` if empty, not `null`), so clients never
have to null-check. A new endpoint `GET /api/v1/viewer/tags` returns the full
vocabulary as `{"tags": [{"name": "voice"}, {"name": "auth"}]}`. A new
query-param `?tag=voice&tag=auth` on list endpoints applies the AND filter.

**UI.** Tag chips render on each entity detail view. A top-of-list tag filter
control fetches the vocabulary from the new endpoint and sends selected tags
as `?tag=` parameters. **No create / rename / delete control exists** in the
UI (PRD scope).

### 4.6 Integration with the maintainer gate — for future consumers

The `MaintainerGate` package exposes exactly the surface area a future admin
command needs:

```go
gate := maintainer.NewGate(projectRoot, cfg.Maintainer)
if err := gate.Authorize(ctx, providedPass); err != nil {
    return err
}
defer gate.RecordSuccess(ctx) // best-effort; failure logs but does not fail the op
```

No shark-specific types, no tag-specific concepts. A hypothetical
`shark admin purge --pass=...` is a three-line adoption.

---

## 5. Migration & Backward Compatibility

### 5.1 Fresh databases

`ApplySchemaAndMigrations` creates `tags` and `entity_tags` on first run,
identically to how it creates `epics` / `features` / `tasks` today. No
special case.

### 5.2 Existing databases (the common case)

Developer applies the migration exactly once by flipping `skip_migrations` to
`false` for one shark invocation, per the standard project protocol. The
migration runs, version bumps to 14, developer flips `skip_migrations` back to
`true`. Tags tables exist but are empty.

### 5.3 No rollback

If a developer runs the v14 migration and then downgrades shark to a v13
binary, the `tags` and `entity_tags` tables are simply unused — the old
binary never sees them and never queries them. No rollback migration is
shipped; the v13→v14 migration is forward-only.

### 5.4 User-facing migration (`tag_required_for` surprise)

A developer who adopts E28, sets `tag_required_for: ["task"]`, and runs
`shark task create` without `--tag` will see a clear failure. To prevent this
surprise on shipping, the default config does **not** ship with
`tag_required_for` populated. Opting in to enforcement is an explicit
maintainer action.

### 5.5 Viewer compatibility

Tags appear as an additional field on entity responses. Older viewer clients
that don't know about the field simply ignore it. No version negotiation
required.

---

## 6. Cross-cutting Notes

- **Testing.** Per `.claude/rules/testing/architecture.md`: `TagRepository` and
  `EntityTagRepository` tests use the real DB with cleanup. `TagService` and
  `MaintainerGate` tests mock their dependencies. CLI tests mock `TagService`
  and `MaintainerGate`.
- **Observability.** Repositories follow the existing OpenTelemetry pattern
  (`repoutil.NewTracer`, span per method). `MaintainerGate.Authorize` records
  a span with a single attribute: `maintainer.authorized = true|false`.
  **The provided password and stored hash are never written to spans.**
- **Input sanitization.** Tag name validation lives in `models/tag.go` as a
  compiled regex (per `.claude/rules/go/input-sanitization.md`). The service
  layer lowercases before validation. SQL uses parameterized placeholders
  exclusively.
- **Concurrency.** `MaintainerGate` cache writes are best-effort; two
  concurrent CLI invocations racing to write the cache file will both succeed
  (last-write-wins) because the write is a full-file replace, not an append.
  An occasional lost cache entry at most costs a user one extra `--pass` on
  the next command.
- **No new third-party dependencies.** SHA-256 is in the stdlib; XDG cache
  resolution is a ~20-line helper.
