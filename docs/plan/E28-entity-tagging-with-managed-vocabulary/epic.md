---
epic_key: E28
title: Entity Tagging with Managed Vocabulary
description: Add polymorphic tags across all core entities (epic, feature, task, bug, change-card, idea) with a closed, maintainer-gated vocabulary so teams can query work along cross-cutting dimensions.
---

# Entity Tagging with Managed Vocabulary

**Epic Key**: E28

This PRD is the single source of business context for E28. Features and tasks under this epic MUST reference this document rather than restating its content.

---

## 1. Problem Statement and Business Justification

### Problem

Shark's entity model is strictly hierarchical: `epic → feature → task`, with `bug` and `change-card` each linking to a single parent. This shape captures "what work rolls up to what," but it has no native representation for **cross-cutting concerns** — dimensions that span the hierarchy:

- "Everything touching voice I/O" (spans multiple epics)
- "All auth-related work" (spans features, bugs, and change-cards across three epics)
- "Anything affecting the migration path" (spans tasks and change-cards in unrelated epics)

Today, users attempt to answer these questions by running `shark search <term>`, which has two structural problems:

1. **Search is full-text, not categorical.** It returns anything mentioning the word, not anything *classified as* the concept. A task titled "Fix typo in voice prompt" and a task titled "Implement the voice subsystem" are equally weighted.
2. **Search is not designed for set-membership queries.** There is no stable, reproducible "set of all voice-related tasks"; the answer depends on the query string and the author's word choices.

Ad-hoc workarounds (conventional title prefixes, tag-like tokens in notes, structured frontmatter) all suffer from **vocabulary drift**. Within weeks, the same concept appears as `voice`, `Voice`, `voice-io`, `voice_input`, `VoiceIO`, and any filter breaks silently.

### Business Justification

Tagging is a **multiplicative enabler**. It does not unblock a single high-value user journey; instead, it increases the reliability and cost-effectiveness of every cross-cutting query/reporting capability shark grows from here on:

- **Reliable categorical queries**: a closed vocabulary guarantees that `list --tag=voice` returns the same set for every user on every run.
- **Cheap to extend across entities**: the polymorphic pattern already used by `entity_notes`, `entity_documents`, `entity_history`, and `entity_relationships` means a 6th/7th entity type gets tag support for a one-line registration, not a new join table and migration.
- **Establishes a reusable maintainer gate**: the password-gated admin operation pattern introduced here (sudo-style short-lived cache, CLI `--pass` flag, interactive prompt) is general-purpose and unblocks future work that needs to prevent casual (especially LLM-agent-initiated) destructive admin operations.

Without E28, cross-cutting views remain either impossible or unreliable, and shark accumulates inconsistent ad-hoc tagging conventions in entity titles and notes that are impossible to clean up retroactively.

---

## 2. Goals and Success Criteria (Measurable)

### Goals

1. Give all six core entity types (`epic`, `feature`, `task`, `bug`, `change-card`, `idea`) a uniform tagging mechanism.
2. Guarantee vocabulary consistency by closing the tag set — unregistered tags are rejected at write time.
3. Restrict vocabulary modification to a designated maintainer via a reusable, general-purpose password gate with sudo-style caching.
4. Make tags queryable via `shark list --tag=<name>` and `shark search --tag=<name>` in v1.
5. Allow per-entity-type enforcement of "at least one tag required" via `.sharkconfig.json` (`tag_required_for`).
6. Surface tags and tag-based filtering in the E27 web viewer (read-only; vocabulary management stays CLI-only in v1).

### Measurable Success Criteria

Each criterion is a concrete, testable outcome.

| # | Success Criterion | How It Is Verified |
|---|---|---|
| SC-1 | A user can register a tag, apply it to entities of any of the six types, and retrieve those entities with `shark list --tag=<name>`. | End-to-end UAT: register `voice`, tag one epic, one feature, one task, one bug, one change-card, one idea; `shark list --tag=voice` across listings returns all six. |
| SC-2 | Tagging with an unregistered name fails and the error message includes both the current vocabulary and the `shark tags add` command. | Run `shark task update <key> --tag=does-not-exist`; assert exit non-zero and stderr contains both the vocabulary list and the exact `shark tags add` command string. |
| SC-3 | A non-maintainer (no password, no cached session) cannot run `shark tags add|rm|rename`; the error explains how to obtain the password. | Run `shark tags add foo` with no cache and no `--pass`; assert exit non-zero and stderr references `.sharkconfig.json` maintainer password setup. |
| SC-4 | A maintainer runs `shark tags add foo --pass=...` once, then a second gated command within 60 seconds without `--pass` and it succeeds (sudo-style cache). | Timed shell test: first command with `--pass`, second command within 60s without `--pass` succeeds; a third command after the cache window expires fails without `--pass`. |
| SC-5 | `shark tags rename voice audio` updates the display name without per-entity migration, and `shark list --tag=audio` returns every entity previously returned by `shark list --tag=voice`. | UAT: tag N entities with `voice`, rename, verify the same N entities appear under the new name. Inspect schema to confirm `entity_tags` rows were not rewritten (only the `tags.name` row changed). |
| SC-6 | Setting `tag_required_for: ["task"]` in `.sharkconfig.json` causes `shark task create` without `--tag` to fail, while `shark epic create` without `--tag` still succeeds. | Integration test covering both code paths. |
| SC-7 | The schema adds exactly two new tables (`tags`, `entity_tags`) and no per-entity tag join tables. | Schema review at code review time; `sqlite3 .schema | grep -i tag` shows only these two tables. |
| SC-8 | Any entity view in the E27 web viewer displays that entity's tags, and list views support filtering by tag using the same vocabulary as the CLI. | Manual UAT in the viewer plus an automated API test that hits `/api/entities?tag=<name>` and the vocabulary endpoint. |
| SC-9 | The maintainer gate is implemented as a reusable mechanism (not hard-coded inside the tag vocabulary commands) so that a future admin command can adopt it without reimplementing the gate. | Code review verifies the gate lives in a shared package and `shark tags add/rm/rename` consume it as clients. |
| SC-10 | Documentation for `.sharkconfig.json` fields (`tag_required_for`, `maintainer_password` or equivalent), the `shark tags` command group, and `--tag` on existing `create`/`update`/`list`/`search` pages is updated in the same release. | Docs review as part of the epic's QA gate. |

---

## 3. Scope

### In Scope (v1)

- **Entity coverage**: tagging works uniformly on all six entity types — `epic`, `feature`, `task`, `bug`, `change-card`, `idea`.
- **Schema**: two new tables only — `tags` (vocabulary registry: id, name, timestamps) and `entity_tags` (polymorphic join on `(entity_type, entity_id, tag_id)`).
- **Service layer**: a new `TagService` alongside existing services, following the service-design rules in `.claude/rules/services/service-design.md`.
- **CLI — vocabulary management**: `shark tags list`, `shark tags add <name>`, `shark tags rm <name>`, `shark tags rename <old> <new>` — all mutating operations gated by the maintainer password.
- **CLI — applying tags at create/update time**: `--tag` flag on `shark <entity> create` and `shark <entity> update` for all six entity types.
- **CLI — retroactive tagging**: `shark <entity> tag add <key> <tag>` and `shark <entity> tag rm <key> <tag>` subcommands per entity type.
- **CLI — querying by tag**: `shark list --tag=<name>` (at all list scopes — top-level, per-epic, per-feature) and `shark search --tag=<name>`.
- **Enforcement**: `.sharkconfig.json` `tag_required_for: [<entity-type>, ...]` enforces at least one tag at create time for the listed entity types only.
- **Maintainer gate (reusable)**: password stored in `.sharkconfig.json`, `--pass` CLI flag, sudo-style cache (~60s window), interactive prompt path. Built as a shared mechanism, with `shark tags add/rm/rename` as the first consumers.
- **E27 web viewer integration**:
  - API: include `tags` on entity responses from `internal/api/viewer`; add a `tag` query parameter to list endpoints; expose the vocabulary via a new read-only endpoint.
  - UI: render tag chips on entity detail views in `internal/viewer`; add a tag filter control on list views.
- **Documentation**: `.sharkconfig.json` reference, new `shark tags` CLI reference page, updates to existing `create` / `update` / `list` / `search` pages for `--tag`.

### Out of Scope (v1)

The following are explicitly **deferred**. They are not rejected forever — they are rejected for v1 to keep the surface area small and ship a cohesive MVP.

- **Tag-aware dashboards**: `shark progress`, `shark analytics`, and `shark status` do **not** gain tag filtering in v1. (Added when a real use case surfaces.)
- **Vocabulary management from the web viewer**: `add` / `rm` / `rename` stay CLI-only. The viewer is read-only with respect to the vocabulary.
- **Hierarchical tags**: no parent/child tag relationships, no tag trees.
- **Tag aliases / synonyms**: no "voice is an alias of audio."
- **Auto-suggested tags**: no ML/heuristic tag suggestions derived from entity titles or bodies.
- **Fixing `search` itself**: pre-existing `search` issues (ranking/behavior) are acknowledged but tracked separately — v1 adds the `--tag` filter without reworking `search`.
- **Tag-based permission / visibility rules**: no "only users with X can see entities tagged Y."
- **Bulk tagging by pattern**: no `shark tag-many E07-F01-* --tag=foo`. Users tag one entity at a time, or via `--tag` on creation.
- **Tag description / color metadata**: names only in v1 (deferred as an open question — see §4).
- **Historical freeform-tag migration**: no automated cleanup of ad-hoc tag-like strings in legacy notes/titles.

### Future Considerations (explicitly outside v1, flagged for future epics)

- Generalize `tag_required_for` into a pattern for other field-level requirements (e.g., `agent_required_for`).
- Apply the maintainer gate to other destructive operations (bulk delete, schema admin commands).
- Tag-based analytics as a follow-up epic once usage patterns are visible.

---

## 4. Constraints and Assumptions

### Constraints

1. **Architecture layering** (from `.claude/rules/architecture.md`): all business logic (vocabulary validation, enforcement of `tag_required_for`, cache/gate logic) lives in the service layer. CLI commands remain thin wrappers. Repositories are CRUD-only.
2. **Polymorphic pattern reuse**: the `entity_tags` table MUST follow the polymorphic pattern already established by `entity_notes`, `entity_documents`, `entity_history`, and `entity_relationships`. No per-entity join table.
3. **Exactly two new tables**: `tags` and `entity_tags`. No additional schema for tag categories, tag metadata tables, etc. in v1.
4. **Migration discipline** (from `.claude/rules/database-critical.md`): any new schema requires a migration function, bumped `CurrentSchemaVersion`, and an explicit call-out to the developer about toggling `skip_migrations` in `.sharkconfig.json`.
5. **Dual backend support**: the schema and migration must work on both local SQLite and Turso cloud. No features that exist in one backend only.
6. **Case-insensitive key handling is already solved**: tag name normalization must be consistent (likely lowercase), but entity key lookups continue to use the existing case-insensitive resolution — no new lookup paths required.
7. **CLI is stateless per invocation**: there is no long-running CLI process. Sudo-style cache state must be externalized (file-based) and project-scoped — it cannot live in process memory.
8. **Input sanitization** (from `.claude/rules/go/input-sanitization.md`): tag names must pass the same structural/allowlist validation standards as existing entity keys (regex anchor, `strings.TrimSpace`, maximum length, no SQL string interpolation).
9. **Testing discipline** (from `.claude/rules/testing/architecture.md`): repository tests use the real DB with cleanup; service tests mock repositories; CLI tests mock services. Real DB appears only in repository tests.
10. **Quality gate** (from `.claude/rules/development-workflows.md`): `make fmt && make lint && make test` passes before the epic is declared done.
11. **Workflow compatibility**: the epic must work under both shipped workflow profiles (`shark-templates/.sharkworkflow-short.json` and `shark-templates/.sharkworkflow.json`).

### Assumptions

1. The existing polymorphic infrastructure (`entity_notes`, `entity_documents`, `entity_history`, `entity_relationships`) is a suitable model and can be replicated without architectural changes.
2. The E27 web viewer (`internal/api/viewer`, `internal/viewer`) is live or progressing on a schedule compatible with E28's viewer integration work. If E27 slips, viewer-facing work within E28 is permitted to slip with it but the CLI- and service-layer work proceeds independently.
3. A plaintext password in `.sharkconfig.json` is acceptable to the product owner for v1. The security model is "prevent accidental / casual-LLM-agent modification," not "withstand a targeted adversary with filesystem access." (Decisions about hashing live under refinement open question O-2 below.)
4. Users and LLM agents operating as non-maintainers will accept password prompts / errors as normal operation, not friction that justifies regression.
5. The `.sharkconfig.json` file itself is already subject to the user's judgment about git-commit hygiene (it may contain paths and preferences). Adding a maintainer password follows the same trust model, and the documentation will call this out.
6. 60 seconds is a reasonable default cache window for the sudo-style gate. It is tunable via config if needed, but v1 may ship with a fixed value.
7. No existing shark command currently needs a maintainer gate, so introducing the gate via E28 does not create migration work for other commands.

### Open Questions (to be resolved during feature refinement — not blocking PRD)

- **O-1 Multi-tag filter semantics**: `--tag=a --tag=b` — AND (both required), OR (either), or configurable? (Default needs a decision before implementation.)
- **O-2 Password storage**: plaintext vs. hash in `.sharkconfig.json`? (Decision before implementing the gate.)
- **O-3 Session cache location**: `/tmp/<scope>/`? `~/.cache/shark/<project-hash>`? Project-local `.shark/session`? (Affects multi-project developers.)
- **O-4 Interactive mode gate behavior**: prompt for password on each gated command, or require pre-authentication?
- **O-5 Tag description / color metadata**: include fields now as nullable, or strictly name-only for v1?
- **O-6 `shark tags rm` on a tag that's in use**: block with a usage-count error, cascade-remove, or require `--force`?
- **O-7 Rename collision**: renaming `voice` to `audio` when `audio` already exists — merge, hard error, or require an explicit merge command?

---

## 5. Stakeholder Impact

| Stakeholder | Interest | Impact of E28 | Action Needed |
|---|---|---|---|
| **Solo developers / small teams** (primary users) | Need reliable cross-cutting views across epics/features without manual spreadsheets. | Gain first-class tag queries; must register vocabulary upfront; minor friction on first tag use. | Learn `shark tags` command; agree on team vocabulary; optionally enable `tag_required_for`. |
| **LLM / AI agents** (secondary users; major operators of shark) | Create, list, and update entities during automated workflows. | Agents can tag and filter by tag using the same CLI; **cannot** modify vocabulary without the maintainer password, which is by design — this is the primary guardrail for why the gate exists. | Agent prompts/skills are updated to know: apply existing tags freely; do not attempt `tags add/rm/rename` unless explicitly given the password. |
| **Project maintainers** (designated humans) | Own the vocabulary; keep it curated. | Become the gatekeepers for `add` / `rm` / `rename`; receive a new responsibility and a new password to manage. | Set the password in `.sharkconfig.json`; decide which entity types (if any) require tags; curate the initial vocabulary. |
| **Product owner** (project lead) | Wants consistent reporting across the project. | Gets reproducible "everything tagged X" queries; gains a new lever (`tag_required_for`) for quality control on entity creation. | Define initial vocabulary with maintainers; decide the `tag_required_for` policy. |
| **QA / reviewers** | Verify shipped features behave as specified. | Gain new test surface (tagging, gate, cache, enforcement, viewer). | Absorb into existing QA workflows; test matrix expands by ~10 scenarios per the acceptance criteria below. |
| **Shark codebase maintainers** | Avoid schema bloat and pattern inconsistency. | E28 reuses the polymorphic pattern and adds exactly two tables. No per-entity bloat. Establishes the reusable maintainer-gate pattern. | Review that polymorphic pattern is genuinely reused (not re-invented) at architecture review. |
| **E27 web viewer team / users** | Expect viewer parity with CLI for read paths. | Viewer gains tag display and tag filtering. Vocabulary management intentionally stays CLI-only (deferred to future epic). | Implement viewer API changes (`internal/api/viewer`) and UI changes (`internal/viewer`) in coordination with this epic. |
| **Future admin-command authors** | Will want the maintainer gate for other sensitive ops. | Inherit a reusable gate built here; don't need to reinvent. | No direct action; benefit is downstream. |

---

## 6. High-Level Acceptance Criteria (UAT Scenarios)

These scenarios are the user-acceptance gates for the epic. Each maps to one or more measurable success criteria in §2. Features under this epic will decompose these into more granular feature-level acceptance criteria.

### UAT-1: Register a tag and apply it across entity types

**Given** a fresh shark project with the maintainer password configured in `.sharkconfig.json`
**When** the maintainer runs `shark tags add voice --pass=<password>`
**And** creates one entity of each of the six types with `--tag=voice`
**Then** `shark tags list` shows `voice` in the vocabulary
**And** `shark list --tag=voice` returns exactly those six entities across entity-type listings
**And** the schema has exactly one new row in `tags` and six new rows in `entity_tags`.

_Maps to SC-1, SC-7._

### UAT-2: Unregistered tag is rejected with helpful error

**Given** the vocabulary contains only `voice` and `auth`
**When** a user runs `shark task update E07-F01-001 --tag=does-not-exist`
**Then** the command exits with a non-zero status
**And** stderr lists the current vocabulary (`voice`, `auth`)
**And** stderr includes the exact command the user should run (`shark tags add does-not-exist --pass=<password>`)
**And** the task is unchanged in the database (verified by re-reading the task).

_Maps to SC-2._

### UAT-3: Non-maintainer cannot modify the vocabulary

**Given** no active maintainer cache (no `--pass` in the last 60 seconds)
**When** a user runs `shark tags add experimental` without `--pass`
**Then** the command exits with a non-zero status
**And** stderr explains that vocabulary modification requires the maintainer password
**And** stderr points at `.sharkconfig.json` as the place the password is configured
**And** the vocabulary is unchanged.

_Maps to SC-3._

### UAT-4: Sudo-style cache lets a burst of admin commands through

**Given** the maintainer cache is empty
**When** the maintainer runs `shark tags add foo --pass=<password>` (succeeds)
**And** within 60 seconds runs `shark tags add bar` without `--pass`
**Then** the second command succeeds without prompting
**And** a third command run more than 60 seconds after the first `--pass` invocation fails the same way UAT-3 fails.

_Maps to SC-4._

### UAT-5: Rename updates all uses without per-entity migration

**Given** `voice` is registered and N entities (N ≥ 5, spanning at least three different entity types) are tagged `voice`
**When** the maintainer runs `shark tags rename voice audio --pass=<password>`
**Then** `shark list --tag=audio` returns exactly those N entities
**And** `shark list --tag=voice` returns zero entities (and, depending on O-2 open question, either errors as unregistered or returns empty — either is acceptable so long as it is deterministic and documented)
**And** inspecting `entity_tags` confirms the same row IDs exist (only `tags.name` was rewritten, not the join rows).

_Maps to SC-5, SC-7._

### UAT-6: Per-entity-type enforcement works as configured

**Given** `.sharkconfig.json` has `tag_required_for: ["task"]`
**When** a user runs `shark task create E07 F01 "No tag here"` without `--tag`
**Then** the command fails with a clear error naming the missing tag requirement
**And** the task is not created in the database
**When** the same user runs `shark epic create "No tag here either"` without `--tag`
**Then** the epic is created successfully (because `epic` is not in `tag_required_for`).

_Maps to SC-6._

### UAT-7: Web viewer displays tags and supports filtering

**Given** the E27 web viewer is running and several entities have been tagged `voice`
**When** a user opens any of those entities in the viewer
**Then** the viewer renders the entity's tags (tag chips are visible in the UI)
**When** the user applies a tag filter (`voice`) on a list view
**Then** the list is reduced to entities with that tag, using the same set that `shark list --tag=voice` would return
**And** the viewer does not expose any control to add, remove, or rename tags in the vocabulary (vocabulary management stays CLI-only in v1).

_Maps to SC-8._

### UAT-8: Maintainer gate is a reusable mechanism

**Given** a code reviewer inspects the implementation
**When** they look for the gate logic
**Then** the gate lives in a shared, reusable package (e.g., `internal/auth/maintainer` or equivalent)
**And** `shark tags add`, `shark tags rm`, `shark tags rename` each consume the gate via a shared helper — none of them embed the password check, cache-read, or cache-write inline
**And** the package exposes a clear API that a future admin command (e.g., hypothetical `shark admin purge`) could consume in one place.

_Maps to SC-9._

### UAT-9: Documentation is complete

**Given** the epic's QA gate
**When** a reviewer inspects `docs/cli-reference/` and `.sharkconfig.json` docs
**Then** there is a new `shark tags` command reference page documenting `list / add / rm / rename` and the `--pass` flag
**And** the `.sharkconfig.json` reference documents the `tag_required_for` field and the maintainer password field with examples
**And** the `create`, `update`, `list`, and `search` command reference pages document the `--tag` flag
**And** a migration note is present explaining the one-time `skip_migrations: false` requirement (per `.claude/rules/database-critical.md`).

_Maps to SC-10._

---

*Last Updated*: 2026-04-22
