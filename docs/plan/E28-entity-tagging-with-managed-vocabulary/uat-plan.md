---
epic_key: E28
document_type: uat-plan
title: E28 UAT Plan — Entity Tagging with Managed Vocabulary
---

# E28 UAT Plan — Entity Tagging with Managed Vocabulary

This document defines the epic-level user acceptance plan for E28. Each
scenario is a black-box acceptance gate: it describes **what to verify**, not
**how to implement the check**. Each scenario maps back to at least one
Success Criterion (SC-*) from `epic.md` §2. A scenario is only accepted when
the described observable outcome matches exactly.

Scope of this UAT plan:
- Vocabulary management (`shark tags list|add|rm|rename`)
- Tag application (`--tag` on create/update; `<entity> tag add|rm`)
- Tag-based querying (`shark list --tag=`, `shark search --tag=`)
- Per-entity-type enforcement (`tag_required_for`)
- Reusable maintainer gate
- E27 viewer integration (read-only)
- Documentation completeness
- Cross-feature integration scenarios
- Performance and security considerations

Reference abbreviations used throughout: **M** = maintainer actor (knows the
password), **A** = agent / non-maintainer actor (no password, no valid cache).

---

## 1. UAT Scenarios (Mapped to Epic Success Criteria)

Each numbered scenario extends or matches one of the §6 scenarios in
`epic.md`, with concrete verification steps and expanded edge cases.

### UAT-1: End-to-end cross-entity tag application

**Maps to:** SC-1, SC-7

**Preconditions.**
- Fresh shark project.
- Maintainer password configured (hash present in `.sharkconfig.json`).
- One pre-existing entity of each type (epic, feature, task, bug,
  change-card, idea) exists for update-based tagging, plus space to create
  new ones for create-based tagging.

**Actions.**
1. M runs `shark tags add voice --pass=<p>`.
2. M creates (or updates, depending on subcommand support) one entity of each
   of the six types with `--tag=voice`.
3. M runs `shark tags list`.
4. M runs `shark list --tag=voice` (and any per-entity-type equivalents).

**Verification.**
- Step 3 returns `voice` in the vocabulary.
- Step 4 enumerates exactly the six tagged entities across entity-type
  listings.
- Direct schema inspection (`SELECT COUNT(*) FROM tags`, `FROM entity_tags`)
  shows 1 tag row and 6 entity_tags rows.
- No tagged entity is of a type not covered (no orphan entity types).

**Fail conditions.**
- Any tagged entity is missing from the listing.
- The vocabulary table has more than one row.
- entity_tags has fewer than six or more than six rows.

---

### UAT-2: Unregistered tag is rejected with actionable error

**Maps to:** SC-2

**Preconditions.** Vocabulary contains exactly `voice` and `auth`. At least
one task exists.

**Actions.**
1. A runs `shark task update <key> --tag=does-not-exist`.

**Verification.**
- Exit code is non-zero.
- stderr lists the current vocabulary (both `voice` and `auth` visible).
- stderr includes the literal substring `shark tags add does-not-exist` so a
  user can copy-paste the fix.
- The task, re-read via `shark task get <key>`, is unchanged (its tag set
  has not grown).

**Fail conditions.**
- Exit code zero.
- Error text lacks either the current vocabulary or the `shark tags add`
  command.
- Task state changes as a side effect of the rejected call.

---

### UAT-3: Non-maintainer cannot modify the vocabulary

**Maps to:** SC-3

**Preconditions.** No active maintainer cache. No `--pass` provided.

**Actions.**
1. A runs `shark tags add experimental`.
2. A runs `shark tags rm voice`.
3. A runs `shark tags rename voice audio`.

**Verification.** For each of the three actions:
- Exit code is non-zero.
- stderr explains the maintainer password is required.
- stderr references `.sharkconfig.json` as the place the password is
  configured.
- The vocabulary is unchanged after all three attempts.

**Fail conditions.**
- Any of the three commands succeeds.
- Error text does not mention where to configure the password.

---

### UAT-4: Sudo-style cache admits a burst of admin commands

**Maps to:** SC-4

**Preconditions.** Cache empty. `cache_window_seconds` is the configured
value (default 60).

**Actions.**
1. M runs `shark tags add foo --pass=<p>`.
2. Within the cache window, M runs `shark tags add bar` **without** `--pass`.
3. Wait until the cache window has elapsed (`cache_window_seconds` + 5s
   margin).
4. M runs `shark tags add baz` **without** `--pass`.

**Verification.**
- Step 1 succeeds.
- Step 2 succeeds without prompting and without `--pass`.
- Step 4 fails in the same manner as UAT-3.

**Fail conditions.**
- Step 2 fails.
- Step 4 succeeds.
- Cache window is shorter or longer than configured by more than 5 seconds.

---

### UAT-5: Rename updates uses without per-entity migration

**Maps to:** SC-5, SC-7

**Preconditions.** `voice` is registered. N ≥ 5 entities spanning at least
three distinct entity types are tagged `voice`. A snapshot of the
`entity_tags` row IDs for those associations has been captured.

**Actions.**
1. M runs `shark tags rename voice audio --pass=<p>`.
2. A runs `shark list --tag=audio`.
3. A runs `shark list --tag=voice`.
4. Direct schema inspection: `SELECT id, entity_type, entity_id, tag_id FROM
   entity_tags` for the originally-tagged entities.

**Verification.**
- Step 2 returns exactly the same N entities captured in the precondition.
- Step 3 either errors as "unregistered tag" or returns zero rows, and the
  behavior is documented. (Either is acceptable per PRD UAT-5 as long as it
  is deterministic.)
- Step 4 shows the same entity_tags row IDs as the precondition snapshot —
  proving no rows were rewritten.
- The `tags` table still has exactly one row covering the old + new name
  (same `id`, new `name`).

**Fail conditions.**
- Any entity_tags row ID differs from the snapshot.
- Any entity previously tagged `voice` is missing under `audio`.
- Step 3 is non-deterministic across repeated invocations.

---

### UAT-6: Per-entity-type enforcement works as configured

**Maps to:** SC-6

**Preconditions.** `.sharkconfig.json` has `tag_required_for: ["task"]`.
Vocabulary contains at least one registered tag.

**Actions.**
1. M runs `shark task create E07 F01 "No tag"` without `--tag`.
2. M runs `shark task create E07 F01 "With tag" --tag=<existing>`.
3. M runs `shark epic create "No tag epic"` without `--tag`.

**Verification.**
- Step 1 fails with exit non-zero; stderr names the missing `--tag`
  requirement; task is not created.
- Step 2 succeeds; task is created with the tag attached.
- Step 3 succeeds; epic is created (because `epic` is not in
  `tag_required_for`).

**Fail conditions.**
- Step 1 succeeds or the task row appears in the database.
- Step 3 fails because enforcement leaked across entity types.

---

### UAT-7: Web viewer displays tags and supports filtering

**Maps to:** SC-8

**Preconditions.** E27 web viewer is running. Several entities have been
tagged `voice`.

**Actions.**
1. A opens an entity detail view in the viewer for one of the `voice`-tagged
   entities.
2. A applies the tag filter `voice` on a list view.
3. A inspects the viewer for any control that can add, rename, or delete
   tags in the vocabulary.
4. A calls the new `GET /api/v1/viewer/tags` endpoint (or equivalent).
5. A calls a list endpoint with `?tag=voice`.

**Verification.**
- Step 1 shows visible tag chips for `voice`.
- Step 2 reduces the visible list to the same set that
  `shark list --tag=voice` returns.
- Step 3 finds **no** such control anywhere (vocabulary management is
  CLI-only).
- Step 4 returns the full vocabulary.
- Step 5 returns entities filtered to those tagged `voice` — and the set
  matches step 2 and CLI output exactly.

**Fail conditions.**
- A tag chip is missing from the detail view of a tagged entity.
- The list view set differs from the CLI output.
- A vocabulary-modification control exists in the viewer.

---

### UAT-8: Maintainer gate is reusable (code review)

**Maps to:** SC-9

**Preconditions.** Implementation exists in the PR.

**Actions (code-review verification, not runtime).**
1. Inspect `internal/auth/maintainer` (or equivalent path chosen in
   decomposition). It must be a standalone package.
2. Inspect `shark tags add`, `shark tags rm`, `shark tags rename`
   implementations. Each must obtain the gate via dependency injection, not
   construct the gate inline with hard-coded details.
3. Grep for password comparison (`password_hash`, `sha256`, `subtle.ConstantTimeCompare`, etc.) — all of it must live inside the maintainer package, not in the tag service or tag command files.
4. Inspect the maintainer package's public API: is there at least one
   method (`Authorize`, `RecordSuccess`) that is entity-agnostic and
   shark-domain-agnostic?

**Verification.**
- Steps 1 and 2 show the gate is genuinely shared.
- Step 3 finds password-handling logic only inside the maintainer package.
- Step 4 confirms the API has no tag-specific types.

**Fail conditions.**
- Password comparison code exists inside `internal/services/tag_service.go`
  or `internal/cli/commands/tags.go`.
- The gate API references `Tag` or tag-specific types.
- The cache-write logic is duplicated per gated command.

---

### UAT-9: Documentation is complete

**Maps to:** SC-10

**Preconditions.** Epic QA gate in effect; all implementation complete.

**Actions.**
1. Review `docs/cli-reference/` for a new `tags.md` page documenting `list`,
   `add`, `rm`, `rename`, and the `--pass` flag.
2. Review `.sharkconfig.json` reference for the `tag_required_for` field and
   the `maintainer` block, each with at least one example.
3. Review `create.md`, `update.md`, `list.md`, `search.md` for `--tag`
   documentation.
4. Review whether there is a migration note explaining the one-time
   `skip_migrations: false` requirement.
5. Review the web viewer docs for the tag chip and filter behavior (or an
   explicit "no viewer docs for this epic because UI is self-describing"
   rationale).

**Verification.**
- All above items are present and accurate.
- Cross-links between pages exist (e.g. `list --tag` links to `tags.md`).

**Fail conditions.**
- Any CLI surface from §1 of this UAT plan is undocumented in the release.
- `.sharkconfig.json` reference lacks `tag_required_for` or `maintainer`.

---

## 2. Cross-Feature Integration Scenarios

These scenarios verify correct behavior where E28 surfaces meet one another,
not just in isolation. They are **required** gates for epic completion.

### UAT-INT-1: Apply, rename, and verify filter continuity

**What is verified.** Tagging, renaming, and filtering work together without
data loss.

**Flow.**
1. M registers `voice`.
2. M creates one task and one bug with `--tag=voice`.
3. A runs `shark list --tag=voice` — both appear.
4. M renames `voice` to `audio`.
5. A runs `shark list --tag=audio` — both still appear.
6. M removes tag `audio` from the task via `shark task tag rm`.
7. A runs `shark list --tag=audio` — only the bug appears.

**Pass:** Each step's expected set matches exactly.

### UAT-INT-2: Enforcement + gate + filter under one workflow

**What is verified.** The three cross-cutting features compose correctly.

**Flow.**
1. Maintainer sets `tag_required_for: ["task"]`.
2. A runs `shark task create ... --tag=undefined` (tag unregistered) —
   fails per UAT-2.
3. A runs `shark task create ...` without `--tag` — fails per UAT-6.
4. A tries `shark tags add undefined` — fails per UAT-3.
5. M runs `shark tags add undefined --pass=<p>` — succeeds.
6. A runs `shark task create ... --tag=undefined` — succeeds.
7. A runs `shark list --tag=undefined` — the new task appears.

**Pass:** The failure modes are distinct and correctly attributed (missing
tag vs. unregistered tag vs. unauthorized), and the end-to-end happy path
succeeds.

### UAT-INT-3: Viewer matches CLI under concurrent updates

**What is verified.** Viewer read-path reflects CLI writes without needing
restart/refresh (subject to viewer's normal cache/refresh behavior).

**Flow.**
1. Viewer running. User has list view open filtered by `voice`.
2. M runs `shark task create ... --tag=voice` via CLI.
3. User refreshes the list view.
4. User opens the new task's detail view.

**Pass:** Step 3 shows the new task; step 4 shows the `voice` chip.

### UAT-INT-4: Remove-in-use safety net

**What is verified.** `shark tags rm` on a tag with associations fails unless
`--force`, and `--force` cleans up correctly.

**Flow.**
1. M registers `voice`; M tags three entities with it.
2. M runs `shark tags rm voice --pass=<p>` — fails; error names the usage
   count (3); vocabulary unchanged.
3. M runs `shark tags rm voice --force --pass=<p>` — succeeds; both
   `tags.voice` and the three `entity_tags` rows are gone.
4. A runs `shark list --tag=voice` — errors as unregistered (consistent
   with UAT-2).

**Pass:** Step 2 preserves data; step 3 removes everything atomically
(transactional — no half-state where the tag is gone but entity_tags rows
remain).

### UAT-INT-5: Polymorphic cascade on entity delete

**What is verified.** Deleting a tagged entity removes its entity_tags rows
via the new triggers.

**Flow.**
1. M registers `voice`; M tags a task with it.
2. Snapshot `entity_tags` row count.
3. Delete the task via `shark task delete`.
4. Re-query `entity_tags`.

**Pass:** The row referring to the deleted task is gone. Row count dropped
by exactly 1.

### UAT-INT-6: Turso-backend parity

**What is verified.** Every v1 feature behaves identically on Turso cloud.

**Flow.**
1. Configure a second test project with Turso backend.
2. Run UAT-1, UAT-2, UAT-4, UAT-6 against the Turso-backed project.

**Pass:** Observable results identical to local SQLite. Any divergence is a
release blocker.

---

## 3. Performance Considerations

E28 is not expected to be performance-critical at realistic project scale,
but the following properties are verified.

### P-1: `shark list --tag=` with a single tag is index-supported

**What is verified.** The common case (one tag filter) uses the
`idx_entity_tags_tag_entity` index and does not scan.

**Flow.**
1. Seed 10,000 tasks, of which 500 are tagged `voice`.
2. Run `EXPLAIN QUERY PLAN SELECT ... FROM tasks JOIN entity_tags ... WHERE
   entity_tags.tag_id = ? AND entity_tags.entity_type = 'task'`.
3. Verify the plan uses `idx_entity_tags_tag_entity` (or an equivalent
   covering index), not a full scan.
4. Time the CLI `shark task list --tag=voice` — target < 500ms end-to-end on
   commodity hardware.

**Pass:** EXPLAIN confirms index usage; CLI runtime is within the target.

### P-2: AND-filter on N tags is tractable

**What is verified.** AND-filter on up to 5 tags stays within reason.

**Flow.** Seed 10,000 tasks with varied tag combinations; run `shark list
--tag=a --tag=b --tag=c` for N=1..5 and measure runtime.

**Pass:** N=5 completes in < 1.5s. Sub-linear-in-N ideally, super-linear is
acceptable up to the threshold.

### P-3: Viewer list endpoint latency does not regress

**What is verified.** Adding the `tags` field to viewer entity responses
does not materially increase list-endpoint latency.

**Flow.** Benchmark the viewer list endpoint with and without tags loaded.

**Pass:** P95 latency increase is ≤ 15% on a 1000-entity workload. If
higher, either a second query strategy (batch-load tags per list page) or
an index is required.

### P-4: Vocabulary size ceiling for UI

**What is verified.** The viewer's tag filter control remains usable with a
few hundred tags in the vocabulary.

**Flow.** Seed 500 tags; open the viewer's tag filter.

**Pass:** Filter loads in < 500ms; scrolling/searching within the filter is
responsive.

---

## 4. Security Considerations

The PRD defines the threat model as "prevent accidental / casual LLM-agent
modification" and explicitly **excludes** targeted filesystem adversaries.
These UAT scenarios match that model.

### S-1: Password never appears in process output or logs

**What is verified.** Neither `--pass=<p>` nor the stored hash is echoed to
stdout, stderr, verbose logs, or OpenTelemetry spans.

**Flow.**
1. M runs `shark tags add foo --pass=secret --verbose`.
2. Grep stdout and stderr for the literal `secret`.
3. Grep the stored hash from `.sharkconfig.json` in stdout and stderr.
4. Inspect any captured OpenTelemetry span attributes for either value.

**Pass:** Neither the plaintext password nor the hash appears anywhere in
captured output.

### S-2: Cache file is private

**What is verified.** The maintainer cache file is created with mode 0600 on
POSIX platforms.

**Flow.** Trigger cache creation; stat the file.

**Pass:** `ls -l` shows `-rw-------` (or equivalent on non-POSIX).

### S-3: Cache respects the project scope

**What is verified.** A cache entry written for Project A does not grant
access to Project B.

**Flow.**
1. M runs `shark tags add foo --pass=<p>` in Project A (populates A's
   cache).
2. Change working directory to Project B (different absolute path, different
   maintainer password).
3. M runs `shark tags add bar` in Project B without `--pass`.

**Pass:** Step 3 fails per UAT-3.

### S-4: Non-maintainer read access is unrestricted

**What is verified.** Reading tags (listing, filtering, viewer display) is
open to any caller regardless of maintainer status.

**Flow.** A (no password, no cache) runs `shark tags list`, `shark list
--tag=voice`, and opens the viewer.

**Pass:** All three succeed. The gate applies only to `add`, `rm`, `rename`.

### S-5: SQL injection defense

**What is verified.** Tag names, entity keys, and filter inputs that contain
SQL metacharacters do not compromise the database.

**Flow.**
1. A runs `shark list --tag="'; DROP TABLE tags; --"` (should fail on
   validation regex well before reaching SQL, but the regex is the last
   line of defense, not the only one).
2. Inspect the repository code: every query uses `?` placeholders; no
   `fmt.Sprintf` into a query string.
3. After the attempted injection, verify `tags` table still exists.

**Pass:** Validation rejects the malformed tag name; no SQL is altered;
table is intact.

---

## 5. Mapping to Success Criteria

Completeness check. Each SC must have at least one UAT scenario. No scenario
is "orphaned" (not mapping back to at least one SC).

| SC | UAT coverage |
|---|---|
| SC-1 | UAT-1, UAT-INT-1 |
| SC-2 | UAT-2, UAT-INT-2 |
| SC-3 | UAT-3, UAT-INT-2, S-3, S-4 |
| SC-4 | UAT-4, S-2, S-3 |
| SC-5 | UAT-5, UAT-INT-1 |
| SC-6 | UAT-6, UAT-INT-2 |
| SC-7 | UAT-1, UAT-5 (schema row-count assertion) |
| SC-8 | UAT-7, UAT-INT-3 |
| SC-9 | UAT-8 |
| SC-10 | UAT-9 |

Additional coverage not tied to a single SC but required by the PRD:
- Cascade-on-delete trigger: UAT-INT-5
- Remove-in-use safety: UAT-INT-4
- Turso parity: UAT-INT-6
- Performance: P-1 through P-4
- Security: S-1 through S-5

---

## 6. Exit Criteria for Epic

The epic is considered UAT-complete when:

1. Every scenario UAT-1 through UAT-9 passes.
2. Every cross-feature scenario UAT-INT-1 through UAT-INT-6 passes.
3. Every performance scenario P-1 through P-4 is within its stated pass
   threshold.
4. Every security scenario S-1 through S-5 passes.
5. `make fmt && make lint && make test` is green (from
   `.claude/rules/development-workflows.md`).
6. The code reviewer confirms UAT-8 (reusability of the gate) on the actual
   implementation, not on intent.
7. Schema version bumped to 14 in `internal/db/db.go`; migration PR includes
   the developer callout about `skip_migrations`.
8. Documentation pass UAT-9 complete.
