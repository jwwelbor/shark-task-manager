# Tags Commands

Complete reference for the `shark tags` command group — the managed vocabulary surface for entity tagging.

## Overview

The `shark tags` commands manage the closed tag vocabulary: a registry of lowercase-normalized names that can be applied to epics, features, tasks, bugs, change-cards, and ideas (all six entity families). All mutating operations (`add`, `rm`, `rename`) require maintainer authorization via the `--pass` flag or a live password cache.

> **Process-level exit codes:** The `shark` CLI wrapper emits process exit code **1** for every typed error (see `cmd/shark/main.go`). The "exit code 3:" and "exit code 1:" strings that appear inside error messages on stderr are **internal classification labels** produced by `tagsErrorCode` (in `internal/cli/commands/tags.go`) for JSON error envelopes and log correlation — they are **not** the OS process exit status. Callers that need to distinguish error classes should parse the JSON envelope's `error` field (`unregistered_tag`, `tag_required`, `not_found`, `conflict`, `in_use`, `validation`, `unauthorized`, `db_error`) rather than rely on `$?`.

**Authorization:** Mutating commands require a valid maintainer password. See [Maintainer Configuration](configuration.md#maintainer) for setup instructions, including how to set the password with `shark admin maintainer set-password`.

**Name rules:** Tag names are lowercased and trimmed automatically. Valid names match `^[a-z0-9][a-z0-9-]{0,63}$` — lowercase letters, digits, and hyphens only; must start with a letter or digit; maximum 64 characters.

## Quick Reference

| Command | Description | Auth Required |
|---------|-------------|:---:|
| `shark tags list` | List the full tag vocabulary | no |
| `shark tags add <name>` | Register a new tag name | yes |
| `shark tags rm <name>` | Remove a tag from the vocabulary | yes |
| `shark tags rename <old> <new>` | Rename a tag across all entities | yes |

---

## `shark tags list`

List all registered tag names, ordered alphabetically.

**Usage:**
```bash
shark tags list [--json]
```

**Flags:**
- `--json` - Output in JSON format (global flag)

**Examples:**
```bash
# Human-readable list
shark tags list

# JSON output for scripting
shark tags list --json
```

**Output (plain):**
```
audio
backend
frontend
voice
```

**Output (`--json`):**
```json
[{"name":"audio"},{"name":"backend"},{"name":"frontend"},{"name":"voice"}]
```

The JSON array contains one object per tag. Only the `name` field is emitted — IDs and timestamps are internal details that are not part of the public vocabulary surface.

**Error JSON (empty vocabulary):**
```json
[]
```

An empty vocabulary returns an empty JSON array and exits 0.

---

## `shark tags add`

Register a new tag name in the vocabulary.

**Usage:**
```bash
shark tags add <name> [--pass <value>] [--json]
```

**Arguments:**
- `<name>` — The tag name to register. Automatically lowercased and trimmed. Must match `^[a-z0-9][a-z0-9-]{0,63}$`.

**Flags:**
- `--pass <value>` — Maintainer password. When omitted, the command uses the live password cache (populated by a previous successful mutating command within the cache window). If the cache is empty, the command exits 1 (internal class `unauthorized`) with guidance.
- `--json` — Output in JSON format (global flag)

**Examples:**
```bash
# Add a tag (using the password cache from a recent session)
shark tags add voice

# Add a tag with an explicit password
shark tags add voice --pass mypassword

# Add a tag and get JSON confirmation
shark tags add voice --pass mypassword --json
```

**Output (plain):**
```
Added tag voice
```

**Output (`--json`):**
```json
{"name":"voice"}
```

**Exit codes:**
| Code | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag successfully registered |
| 1 | `unauthorized`, `validation`, `conflict` | Incorrect password, missing password (no cache), name validation error, or tag already exists |
| 1 | `db_error` | Database error (the `--json` envelope carries `"code":"db_error"` for this subclass) |

> **Note:** Process exit code is always **1** for non-zero results from `shark tags add`. The JSON `error` field distinguishes subclasses (`unauthorized`, `validation`, `conflict`, `db_error`). The human-readable error line on stderr includes an internal `Error: exit code 3: ...` prefix for classification errors (3 = invalid state in the internal taxonomy) and `Error: exit code 2: ...` for database errors, but the process exit status is 1 in both cases.

**Error handling:**
- If `--pass` is omitted and no cache entry exists, stderr includes the text `shark admin maintainer set-password` as a hint. See [Maintainer Configuration](configuration.md#maintainer).
- If the tag name already exists, a conflict error is returned (process exit 1, internal class `conflict`).
- If the password is wrong, an authorization error is returned (process exit 1, internal class `unauthorized`).

**Example error output (conflict):**

```
$ shark tags add voice --pass secret
tag already exists: voice
Error: exit code 3: tag already exists: voice
$ echo $?
1
```

**Error JSON shape (stderr):**
```json
{"error":"unauthorized","message":"incorrect maintainer password"}
```

```json
{"error":"conflict","message":"tag already exists: voice"}
```

```json
{"error":"validation","message":"invalid tag name: name must match ^[a-z0-9][a-z0-9-]{0,63}$"}
```

---

## `shark tags rm`

Remove a tag from the vocabulary. If the tag is currently applied to any entities, the command requires `--force` to confirm deletion (which also removes all entity-tag associations).

**Usage:**
```bash
shark tags rm <name> [--pass <value>] [--force] [--json]
```

**Arguments:**
- `<name>` — The tag name to remove.

**Flags:**
- `--pass <value>` — Maintainer password. Same cache behavior as `shark tags add`.
- `--force` — Required when the tag is in use by one or more entities. Without this flag, the command exits 1 (internal class `in_use`) and shows the usage count. When specified, the tag and all its entity associations are removed atomically.
- `--json` — Output in JSON format (global flag)

**Examples:**
```bash
# Remove an unused tag
shark tags rm deprecated --pass mypassword

# Remove a tag that is in use (requires --force)
shark tags rm deprecated --pass mypassword --force

# JSON output
shark tags rm deprecated --pass mypassword --json
```

**Output (plain):**
```
Removed tag deprecated
```

**Output (`--json`):**
```json
{"name":"deprecated","removed":true}
```

**Exit codes:**
| Code | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag successfully removed |
| 1 | `not_found` | Tag not found |
| 1 | `in_use`, `unauthorized`, `validation` | Tag is in use (and `--force` was not provided), incorrect/missing password, or validation error |
| 1 | `db_error` | Database error |

> **Note:** Process exit code is **1** for all non-zero outcomes. The JSON envelope's `error` field identifies the subclass.

**In-use protection:**

When a tag is currently applied to entities:
```
Error: tag "voice" is in use by 7 entities; re-run with --force to delete it and its associations
```

Re-run the same command with `--force` to proceed. The deletion and removal of all entity associations happen atomically.

**Not-found error:**

When the tag name does not exist in the vocabulary, stderr includes:
1. `tag not found: <name>`
2. The current vocabulary (first 10 names, with "…and N more" if longer)
3. The `shark tags add <name>` command to register the name

**Error JSON shape (stderr):**
```json
{"error":"not_found","message":"tag not found: deprecated"}
```

```json
{"error":"in_use","message":"tag \"voice\" is in use by 7 entities; re-run with --force to delete it and its associations"}
```

---

## `shark tags rename`

Rename a tag across all entities that carry it. The rename is atomic — all entity associations are updated transparently via the schema (no per-entity update needed).

**Usage:**
```bash
shark tags rename <old> <new> [--pass <value>] [--json]
```

**Arguments:**
- `<old>` — Current tag name.
- `<new>` — New tag name. Must be a valid, unique name that does not already exist in the vocabulary.

**Flags:**
- `--pass <value>` — Maintainer password. Same cache behavior as `shark tags add`.
- `--json` — Output in JSON format (global flag)

**Examples:**
```bash
# Rename a tag
shark tags rename voice audio --pass mypassword

# Rename with JSON output
shark tags rename voice audio --pass mypassword --json
```

**Output (plain):**
```
Renamed voice to audio
```

**Output (`--json`):**
```json
{"old":"voice","new":"audio"}
```

**Exit codes:**
| Code | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag successfully renamed |
| 1 | `not_found` | Source tag (`<old>`) not found |
| 1 | `conflict`, `validation`, `unauthorized` | Target name already exists (collision), names are identical after normalization, incorrect/missing password, or validation error |
| 1 | `db_error` | Database error |

> **Note:** Process exit code is **1** for all non-zero outcomes. The JSON envelope's `error` field identifies the subclass.

**Error handling:**
- If `<new>` already exists in the vocabulary, the rename fails with a conflict error (process exit 1, internal class `conflict`). Use `shark tags rm` to remove the existing tag first if needed.
- If `<old>` and `<new>` normalize to the same name, a validation error is returned (process exit 1, internal class `validation`).
- If `<old>` is not found, stderr includes the current vocabulary list and an `shark tags add` hint.

**Error JSON shape (stderr):**
```json
{"error":"conflict","message":"tag already exists: audio"}
```

```json
{"error":"not_found","message":"tag not found: voice"}
```

```json
{"error":"validation","message":"invalid tag name: new name must differ from old name"}
```

---

## JSON Output Shapes

All four subcommands support `--json`. This section summarizes all JSON shapes for quick reference.

### Success shapes

| Command | JSON output |
|---------|-------------|
| `shark tags list` | `[{"name":"audio"},{"name":"voice"}]` |
| `shark tags add <name>` | `{"name":"<name>"}` |
| `shark tags rm <name>` | `{"name":"<name>","removed":true}` |
| `shark tags rename <old> <new>` | `{"old":"<old>","new":"<new>"}` |

An empty vocabulary from `list` emits `[]`.

### Error shape (stderr)

All failing `shark` commands emit a top-level `COMMAND_ERROR` JSON envelope on stderr (produced by `cmd/shark/main.go`). The `shark tags` and `shark <entity> tag` commands additionally emit a compact `writeTagsError` envelope with the short internal classification code on stderr **before** the outer envelope, so two JSON documents appear back-to-back on stderr when `--json` is set.

**Inner classification envelope (emitted only by tag-related commands):**

```json
{"error":"<code>","message":"<text>"}
```

**Outer envelope (emitted by every failing `shark` command):**

```json
{
  "error": true,
  "code": "COMMAND_ERROR",
  "message": "<text or 'exit code N: <inner message>'>"
}
```

For the `--tag=<name>` flag path on `create`/`update`, only the outer envelope is emitted (the inner classification envelope is produced by `handleVocabularyErrorWithSnippet`, which is not on that code path today). See [Unregistered Tag Errors](#unregistered-tag-errors) for worked examples.

| `error` code (inner) | Condition |
|--------------|-----------|
| `unauthorized` | Wrong password or missing password/cache |
| `not_found` | Tag name not in vocabulary |
| `conflict` | Tag name already exists (on add or rename) |
| `in_use` | Tag is in use; `--force` required to remove |
| `validation` | Name failed validation rules |
| `db_error` | Database error |

---

## Authorization and Password Cache

All mutating commands (`add`, `rm`, `rename`) invoke the maintainer authorization gate before performing any operation. The gate supports a short-lived password cache so repeated operations within a session do not require re-entering the password.

**First use in a session:**
```bash
shark tags add voice --pass mypassword
# Authorization succeeds; password is cached for the cache window
```

**Subsequent operations (within the cache window):**
```bash
shark tags add audio
# Uses the cached authorization; --pass not required
```

**Cache miss (no --pass, no cache):**
```
Error: no maintainer password configured
Hint: run `shark admin maintainer set-password` to configure a password
```

For setup instructions and cache window configuration, see [Maintainer Configuration](configuration.md#maintainer).

---

## Exit Code Summary

The `shark` CLI process emits exit code **0** on success and **1** for every typed error; callers that need to distinguish error classes should read the `error` field in the JSON envelope (or the internal `Error: exit code <N>:` prefix on the stderr human-readable line).

| Process exit | Internal class | Meaning | Common cause |
|------|----------------|---------|-------------|
| 0 | — | Success | Operation completed |
| 1 | `not_found` | Not found | Tag name does not exist in vocabulary |
| 1 | `db_error` | Database error | SQLite or Turso error |
| 1 | `unauthorized` / `validation` / `conflict` / `in_use` | Auth / validation / conflict / in-use | Wrong password, invalid name, name collision, tag in use without `--force` |
| 1 | `unregistered_tag` | Attach path: tag not in vocabulary | `--tag=<name>` on create/update, or `shark <entity> tag add <key> <name>`, when `<name>` is not registered |
| 1 | `tag_required` | Create-path: no `--tag` supplied when `tag_required_for` lists this entity type | See [Configuration → `tag_required_for`](configuration.md#tag_required_for) |

---

## Applying Tags During Create/Update

The six entity families — `task`, `feature`, `epic`, `bug`, `change`, `idea` — accept a repeatable `--tag <name>` flag on both their `create` and `update` subcommands. Tag names must already be registered in the vocabulary (`shark tags add <name>`); attempting to apply an unregistered name exits 1 (internal class `unregistered_tag`). See [Unregistered Tag Errors](#unregistered-tag-errors) below for the exact error shape — and note that as of the current build the SC-2 vocabulary snippet and "To add it:" remediation line render **only** on the `shark <entity> tag add|rm` subcommand path, not on the `--tag` path.

**Key semantics (`--tag` flag on create/update, and `tag add|rm` subcommands):**

- `--tag` is **repeatable**: pass it once per name (`--tag=voice --tag=auth`). Per ADR-F04-5 (spec §2.10), the recommended, forward-compatible invocation is **one `--tag=<name>` per value**. Cobra's `StringSliceVar` will also split a single `--tag=voice,auth` on commas, but a valid tag name may not contain commas — so explicit repetition is safer and surfaces invalid characters as clear validation errors instead of silently splitting the input.
- `--tag` on `create` attaches all provided tags immediately after the entity is persisted.
- `--tag` on `update` is **additive only** — it attaches new tags without detaching existing ones. Empty or omitted `--tag` performs no change. To detach a tag, use `shark <entity> tag rm`.
- Attach is idempotent: repeating `--tag=voice` on the same entity does not create duplicate attachments (AC-20). `shark <entity> tag add` is likewise idempotent.
- If the tag name is **not in the vocabulary**, the command fails before any mutation (see [Unregistered Tag Errors](#unregistered-tag-errors)).
- If the **entity key** is not found (for `tag add|rm`), the command exits 1 with a `not_found` error.
- Tag attachments are NOT maintainer-gated — any user can apply a registered tag. Only vocabulary mutation (`shark tags add|rm|rename`) requires the maintainer password.

**Examples (AC-19):**

```bash
# Create a task with two tags (both must be registered)
shark task create E07 F01 "Implement JWT validation" --tag=voice --tag=auth

# Create an epic with one tag
shark epic create "Voice Auth Epic" --tag=voice

# Create a bug with a tag
shark bug create "Login crashes" --severity=high --tag=auth

# Create a change-card with a tag
shark change create "Increase session timeout" --tag=auth

# Create an idea with a tag
shark idea create "Magic-link login" --tag=voice

# Additive tag on update
shark task update E07-F01-001 --tag=frontend
```

**Idempotency (AC-20):**

```bash
# Re-running the same --tag is a no-op: entity_tags row count does not grow.
shark task update E07-F01-001 --tag=voice --tag=voice
shark task update E07-F01-001 --tag=voice   # same net effect
```

### Unregistered Tag Errors

When an unregistered tag name reaches any attach-path command — whether via `shark <entity> tag add` or via `--tag` on `shark <entity> create|update` — the SC-2 error shape is rendered on stderr in plain-text mode and the process exits **3**:

1. The service-level error line (`tag is not registered: <name>`).
2. An `Available tags:` header on its own line (omitted when the vocabulary is empty).
3. The current vocabulary (first 10 tag names, two-space-indented, comma-separated; `…and N more` on the next line when truncated).
4. The exact remediation line: `To add it: shark tags add <name>`.
5. A trailing `Error: exit code 3: <original-message>` line emitted by Cobra after `RunE` returns the wrapped error.

This unified rendering applies to all six entity families (task, feature, epic, bug, change, idea) on all create and update paths.

**Example — `tag add` on an unregistered name:**

```bash
$ shark task tag add T-E01-F01-001 does-not-exist
tag is not registered: does-not-exist
Available tags:
  audio, backend, voice
To add it: shark tags add does-not-exist
Error: exit code 3: tag is not registered: does-not-exist

$ echo $?
3
```

**Example — `--tag` on create with an unregistered name:**

```bash
$ shark task create E01 F01 "x" --tag=does-not-exist
tag is not registered: does-not-exist
Available tags:
  audio, backend, voice
To add it: shark tags add does-not-exist
Error: exit code 3: tag is not registered: does-not-exist

$ echo $?
3
```

**Example — `tag rm` on a name not in the vocabulary:**

```bash
$ shark task tag rm T-E01-F01-001 does-not-exist
tag not found: does-not-exist
Available tags:
  audio, backend, voice
To add it: shark tags add does-not-exist
Error: exit code 1: tag not found: does-not-exist

$ echo $?
1
```

**Error JSON shape (`--json`) — unregistered tag:**

```json
{"error":"unregistered_tag","message":"tag is not registered: does-not-exist"}
{
  "error": true,
  "code": "COMMAND_ERROR",
  "message": "exit code 3: tag is not registered: does-not-exist"
}
```

(Two JSON documents are emitted to stderr back-to-back: the inner `unregistered_tag` envelope produced by `writeTagsError` — which carries the short classification code — and the outer `COMMAND_ERROR` envelope produced by `cmd/shark/main.go` when Cobra's `Execute()` returns a non-nil error. Integrators should read whichever shape matches the stable contract they want to depend on; the outer `COMMAND_ERROR` envelope is emitted by every failing `shark` command, so it is the safest general parser target.)

### Tag-Required Errors

When `tag_required_for` lists an entity type and the user creates an entity without supplying any `--tag` flag, the command exits **3** with:

1. The error line: `at least one tag is required for <entity>`.
2. An `Available tags:` header and snippet (when the vocabulary is non-empty).

The remediation line (`To add it:`) is **not** emitted for tag-required errors because no specific tag name was rejected; the user should choose from the listed vocabulary.

**Example:**

```bash
$ shark bug create "Login crash"   # assuming bug is in tag_required_for
at least one tag is required for bug
Available tags:
  audio, backend, voice
Error: exit code 3: at least one tag is required for bug

$ echo $?
3
```

---

## Retroactive Tagging: `shark <entity> tag add|rm`

Each of the six entity command groups exposes a `tag` subcommand with `add` and `rm` sub-subcommands for retroactively attaching or detaching a single registered tag on an existing entity.

**Invocation shape:**
```bash
shark <entity> tag add <key> <name>
shark <entity> tag rm  <key> <name>
```

Where `<entity>` is one of `task`, `feature`, `epic`, `bug`, `change`, `idea`. Each command takes exactly two positional arguments: the entity key and the tag name.

See [Applying Tags During Create/Update → Key semantics](#applying-tags-during-createupdate) for the shared semantics that apply to both the `--tag` flag and the `tag add|rm` subcommands (repeatability, idempotency, no maintainer gate, etc.).

**Additional notes specific to the `tag add|rm` subcommand path:**

- `tag add` / `tag rm` render the SC-2 vocabulary snippet + `To add it: shark tags add <name>` remediation line on stderr when the tag name is not in the vocabulary — this is the only CLI surface that renders the snippet today (see [Unregistered Tag Errors](#unregistered-tag-errors)).
- Process exit code is always **0** on success or **1** on error; the internal classification is `unregistered_tag` for `tag add` on an unknown name and `not_found` for `tag rm` on an unknown name.

### `shark <entity> tag add`

Attach a registered tag to an existing entity.

**Examples (AC-23):**

```bash
# Attach 'voice' to bug B001
shark bug tag add B001 voice

# Re-running is idempotent (no duplicate row):
shark bug tag add B001 voice   # exit 0, no-op

# Attach a tag to other entity types
shark task tag add E07-F01-001 auth
shark feature tag add E07-F01 voice
shark epic tag add E07 voice
shark change tag add CC-001 auth
shark idea tag add I-2026-02-25-01 voice   # AC-26
```

**Output (default):**
```
Attached tag "voice" to bug B001
```

**Output (`--json`):**
```json
{"entity_type":"bug","entity_key":"B001","tag":"voice","attached":true}
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag attached (or already attached — idempotent no-op) |
| 1 | `not_found` | Entity key not found |
| 1 | `unregistered_tag` | Tag name not in vocabulary |
| 1 | `validation` | Name validation error |
| 1 | `db_error` | Database error |

### `shark <entity> tag rm`

Detach a registered tag from an existing entity.

**Examples (AC-24, AC-25):**

```bash
# Detach 'voice' from bug B001
shark bug tag rm B001 voice

# Re-running is idempotent (attachment is already gone):
shark bug tag rm B001 voice   # exit 0, no-op

# Attempting to detach a tag name that is not in the vocabulary (AC-25)
shark bug tag rm B001 does-not-exist
# Process exit 1, internal class 'not_found'. Actual stderr:
#   tag not found: does-not-exist
#   Available tags:
#     audio, backend, voice
#   To add it: shark tags add does-not-exist
#   Error: exit code 1: tag not found: does-not-exist
```

**Output (default):**
```
Detached tag "voice" from bug B001
```

**Output (`--json`):**
```json
{"entity_type":"bug","entity_key":"B001","tag":"voice","detached":true}
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag detached (or was not attached — idempotent no-op) |
| 1 | `not_found` | Entity key not found, or tag name not in vocabulary |
| 1 | `validation` | Name validation error |
| 1 | `db_error` | Database error |

---

## Filtering by Tag (`--tag` on `list` and `search`)

Every entity list command and the top-level `shark list` dispatcher accept a repeatable `--tag <name>` flag that restricts results to entities carrying **all** of the supplied tags (AND semantics). Tag names must be registered in the vocabulary; supplying an unregistered name exits **3** with the SC-2 vocabulary-snippet error (see [Unregistered Tag Errors](#unregistered-tag-errors)).

### Flag binding (REQ-F-018)

The verbatim help text for every `--tag` flag on list/search commands is:

```
Filter by tag (repeatable; AND — all tags must match).
```

### Commands that accept `--tag` on the list/search path

| Command | Scope |
|---------|-------|
| `shark list [--tag=<name>]` | Top-level dispatcher; forwarded to the correct entity branch |
| `shark list E## [--tag=<name>]` | Features in epic; forwarded to feature list |
| `shark list E## F## [--tag=<name>]` | Tasks in feature; forwarded to task list |
| `shark task list [--tag=<name>]` | Tasks |
| `shark feature list [--tag=<name>]` | Features |
| `shark epic list [--tag=<name>]` | Epics |
| `shark bug list [--tag=<name>]` | Bugs |
| `shark change list [--tag=<name>]` | Change-cards |
| `shark idea list [--tag=<name>]` | Ideas |
| `shark search <query> [--tag=<name>]` | Cross-entity full-text search (tasks, features, epics, bugs, change-cards) |

### AND semantics

When multiple `--tag` flags are supplied, the command returns only entities that have **all** of the supplied tags. Use **one `--tag=<name>` per tag**; do not comma-join names in a single flag value (e.g. `--tag=voice,auth` is parsed as two names by Cobra's `StringSliceVar`, but this behaviour is an implementation detail — prefer the explicit form):

```bash
# Correct: two distinct --tag flags
shark task list --tag=voice --tag=auth

# Also accepted by Cobra (comma-split), but explicit form is preferred:
shark task list --tag=voice,auth
```

There is no `--tag-op=or` or negation in v1. OR semantics and `--without-tag` are out of scope.

### Examples

```bash
# List all epics tagged 'voice' (AC-21)
shark list --tag=voice

# List features in epic E07 tagged 'voice' (AC-22)
shark list E07 --tag=voice

# List tasks in E07-F01 tagged 'voice' (AC-23)
shark list E07 F01 --tag=voice

# List tasks tagged with BOTH 'voice' AND 'auth' (AND semantics, AC-24)
shark task list --tag=voice --tag=auth

# List bugs tagged 'voice' (AC-25)
shark bug list --tag=voice

# List change-cards tagged 'voice' (AC-25b)
shark change list --tag=voice

# List ideas tagged 'voice' (AC-25c)
shark idea list --tag=voice

# List features in epic E07 tagged 'voice' (AC-25d)
shark feature list E07 --tag=voice

# List epics tagged 'voice' (AC-25e)
shark epic list --tag=voice

# Search for 'login' and restrict to results tagged 'voice' (AC-26)
shark search "login" --tag=voice
```

### Unregistered tag on list/search path (AC-27)

When a `--tag` value is not in the vocabulary, the service returns `*UnregisteredTagError` and the CLI renders the SC-2 vocabulary snippet (via the existing `handleEntityServiceError` helper from F04):

```
tag is not registered: does-not-exist
Available tags:
  audio, backend, voice
To add it: shark tags add does-not-exist
Error: exit code 3: tag is not registered: does-not-exist

$ echo $?
3
```

With `--json` the inner error envelope (stderr) carries:

```json
{"error":"unregistered_tag","message":"tag is not registered: does-not-exist"}
```

### Zero-match behaviour

When the tag filter is satisfied but no entities of the requested type carry all of the tags, the command exits **0** and renders the same "no results" message as a filter-free list with no rows:

```
No tasks found
```

This is not an error; the exit code is 0.

### Performance

For list commands: one `EntityIDsByTags` SQL call is issued against the `entity_tags` table (using `EXISTS` sub-clauses, one per tag), followed by an in-memory intersection with the base-list results. No per-entity round-trips.

For `shark search`: one `EntityIDsByTags` call is issued **per entity type present in the raw FTS result set** (at most 5: `epic`, `feature`, `task`, `bug`, `change`). Worst-case 5 additional SQL statements regardless of result-set size.

---

## Required-Tag Enforcement

Tags can be made mandatory at creation time for specific entity types via the `tag_required_for` field in `.sharkconfig.json`. When configured, the corresponding `create` commands exit with process code **1** (internal class `tag_required`) if no `--tag` is supplied.

See [Configuration → `tag_required_for`](configuration.md#tag_required_for) for the field schema, allowed values, and a worked example.

---

## Web Viewer Tag Integration

The shark web viewer (E27) surfaces the same vocabulary that the CLI manages. Tag display and filtering are **read-only** in the viewer — the UI never offers add/rename/remove controls (those remain CLI-only and maintainer-gated, see [Authorization and Password Cache](#authorization-and-password-cache)).

### What the viewer renders

- Each entity card (epic, feature, task, bug, change-card, idea) renders a horizontal row of `.tag-chip` elements one per attached tag (E28-F06 AC-11). Untagged entities render no chips and no `Tags:` label.
- A tag-filter control populates itself on first page load via `GET /api/v1/viewer/tags` (E28-F06 AC-12). Selecting one or more chips in that control re-fetches the currently visible list with `?tag=<name>` query params appended.
- Multiple selected tags use **AND semantics** (E28-F06 AC-07) — identical to the CLI's `--tag` flag on `list`/`search`.
- The viewer never imports the maintainer gate (E28-F06 AC-15); no UI surface anywhere offers add/rename/remove (E28-F06 AC-13).

### Viewer HTTP API summary

| Endpoint | Behaviour |
|---|---|
| `GET /api/v1/viewer/tags` | Returns the current vocabulary as `{"tags":[{"name":"audio"},...]}` (alphabetical). Empty vocabulary returns `{"tags":[]}` (E28-F06 AC-01, AC-02). |
| `POST /api/v1/viewer/tags` | Returns 404/405 — vocabulary mutation is not exposed via the viewer API (E28-F06 AC-03). |
| `GET /api/v1/viewer/hierarchy[?tag=<name>...]` | Decorates every entity DTO with a `tags` array (always non-null; `[]` when none — E28-F06 AC-04, AC-05). With one or more `?tag=` params the entire tree is pruned to entities tagged with **all** supplied names (AND semantics, E28-F06 AC-06, AC-07). |
| `GET /api/v1/viewer/features/<feature-key>/tasks[?tag=<name>...]` | Returns only the tasks of `<feature-key>` matching the tag filter; pagination `total` reflects the post-filter count (E28-F06 AC-09). |

### Unregistered tag on the viewer API

When any viewer endpoint receives a `?tag=<name>` value not in the vocabulary, it returns `400 Bad Request` with the JSON envelope (E28-F06 AC-08, AC-10):

```json
{
  "error": "Bad Request",
  "message": "unregistered tag: does-not-exist",
  "unregistered_tags": ["does-not-exist"]
}
```

For a feature-scoped task list with both an unregistered tag **and** a missing feature, the feature lookup runs first — so a missing feature returns `404`, not `400` (E28-F06 AC-10).

### Graceful degradation

If the viewer process boots without `TagReader` wired (e.g. tag service failed to initialize), `GET /api/v1/viewer/tags` returns `{"tags":[]}`, every entity DTO `tags` field is `[]`, and no `500` is emitted (E28-F06 AC-14).

For the visual layout of the chip row and filter control, see the [Status Viewer UI Reference](../status-viewer-ui.md). For the architectural decisions that govern this integration (decoration in-memory, never-null arrays, hierarchy filter prunes the tree, viewer never imports `MaintainerGate`), see the [E28-F06 specification](../plan/E28-entity-tagging-with-managed-vocabulary/E28-F06-web-viewer-tag-integration/spec.md).

---

## Schema and Migration (v13 → v14)

E28 introduces two new tables — `tags` (the vocabulary) and `entity_tags` (the polymorphic join) — plus six per-parent cascade-delete triggers and three supporting indexes. The schema version moves from **13 → 14**. There are no per-entity tag columns or per-entity tag join tables (epic SC-7).

**Migration is automatic** for local SQLite databases and for Turso/cloud databases that have `skip_migrations: false`. For Turso/cloud users running with `skip_migrations: true` (the default for performance), a one-time toggle is required:

1. Set `"skip_migrations": false` in `.sharkconfig.json`
2. Run any `shark` command once (the migration applies and bumps the recorded schema version)
3. Set `"skip_migrations": true` again

See [Initialization → Migrating an existing project to v14 (E28 tagging)](initialization.md#migrating-an-existing-project-to-v14-e28-tagging) for the full procedure and [Configuration → `skip_migrations`](configuration.md#database-skip_migrations) for the field reference.

The migration is purely additive — no existing rows or columns are altered, and the migration is idempotent on both backends.

---

## Coverage Matrix

This matrix maps every E28 acceptance criterion (AC) and success criterion (SC) referenced by features F01–F06 to the section or page that documents it. Use this when reviewing whether the user-facing documentation closes the epic's SC-10 / UAT-9 documentation gate.

### Epic Success Criteria (SC-1 .. SC-10)

| ID | Criterion (summary) | Documented in |
|---|---|---|
| SC-1 | End-to-end: register a tag, apply across six entity types, retrieve via `shark list --tag=` | tags.md → [Filtering by Tag](#filtering-by-tag---tag-on-list-and-search) + [Applying Tags During Create/Update](#applying-tags-during-createupdate) |
| SC-2 | Unregistered-tag error includes vocabulary list and `shark tags add` command string | tags.md → [Unregistered Tag Errors](#unregistered-tag-errors) |
| SC-3 | Non-maintainer cannot run `shark tags add\|rm\|rename`; error explains how to obtain the password | tags.md → [Authorization and Password Cache](#authorization-and-password-cache); [configuration.md → maintainer](configuration.md#maintainer) |
| SC-4 | Sudo-style password cache (60s window) | tags.md → [Authorization and Password Cache](#authorization-and-password-cache); [configuration.md → maintainer](configuration.md#maintainer) (`cache_window_seconds`) |
| SC-5 | `shark tags rename` updates display name without per-entity migration | tags.md → [`shark tags rename`](#shark-tags-rename) |
| SC-6 | `tag_required_for: ["task"]` blocks `shark task create` without `--tag`; other types still succeed | tags.md → [Required-Tag Enforcement](#required-tag-enforcement) → [configuration.md → tag_required_for](configuration.md#tag_required_for) |
| SC-7 | Schema adds exactly two new tables (`tags`, `entity_tags`); no per-entity join tables | tags.md → [Schema and Migration (v13 → v14)](#schema-and-migration-v13--v14) |
| SC-8 | Web viewer displays tags and supports filtering using the same vocabulary | tags.md → [Web Viewer Tag Integration](#web-viewer-tag-integration) → [status-viewer-ui.md](../status-viewer-ui.md) |
| SC-9 | Maintainer gate is a reusable mechanism, not hard-coded inside tag commands | tags.md → [Authorization and Password Cache](#authorization-and-password-cache) (mechanism shared with future admin commands); spec: [E28-F02](../plan/E28-entity-tagging-with-managed-vocabulary/E28-F02-reusable-maintainer-authorization-gate/spec.md) |
| SC-10 | Documentation for `.sharkconfig.json` fields, `shark tags`, and `--tag` on every existing reference page is updated in the same release | tags.md (this page) + [configuration.md](configuration.md) + [core-commands.md](core-commands.md) + [discovery-commands.md](discovery-commands.md) + per-entity command pages ([task](task-commands.md), [feature](feature-commands.md), [epic](epic-commands.md), [bug](bug-commands.md), [change](change-commands.md), [idea](idea-commands.md)) |

### F01 — Schema and Migration

F01 has no numbered ACs; the user-visible surface is the v13→v14 migration callout.

| Surface | Documented in |
|---|---|
| Two new tables, indexes, six cascade triggers, schema version bump | tags.md → [Schema and Migration (v13 → v14)](#schema-and-migration-v13--v14) |
| `idea` member added to `models.EntityType` | tags.md → [Applying Tags During Create/Update](#applying-tags-during-createupdate) (idea is a first-class taggable entity); [Filtering by Tag](#filtering-by-tag---tag-on-list-and-search) (`shark idea list --tag=`) |
| One-time `skip_migrations: false` toggle for Turso/cloud users | tags.md → [Schema and Migration (v13 → v14)](#schema-and-migration-v13--v14) → [initialization.md → Migrating an existing project to v14](initialization.md#migrating-an-existing-project-to-v14-e28-tagging) |

### F02 — Reusable Maintainer Authorization Gate

F02's ACs are mostly internal (gate behavior, cache file mode, span attributes). The user-facing surface is the password setup command and the cache window.

| ID | Criterion (summary) | Documented in |
|---|---|---|
| F02 AC-1..AC-3 | Authorize accepts correct password / rejects wrong / hints `shark admin maintainer set-password` when no password configured | tags.md → [Authorization and Password Cache](#authorization-and-password-cache); [configuration.md → maintainer](configuration.md#maintainer) |
| F02 AC-4..AC-6 | Sudo-style cache (success populates, expires after window, invalidated when `password_hash` changes) | tags.md → [Authorization and Password Cache](#authorization-and-password-cache); [configuration.md → maintainer](configuration.md#maintainer) (`cache_window_seconds`) |
| F02 AC-7..AC-10, AC-13..AC-14 | Cache file modes, constant-time compare, malformed-cache safety, OTel attributes, accessor wiring, package import boundaries | Internal — out of scope for user-facing CLI reference |
| F02 AC-11..AC-12 | `shark admin maintainer set-password --password` writes hash and never echoes secrets | [configuration.md → maintainer](configuration.md#maintainer) |

### F03 — Tag Vocabulary Service and CLI

| ID | Criterion (summary) | Documented in |
|---|---|---|
| F03 AC-1 | `ListTags` returns sorted; no auth required | tags.md → [`shark tags list`](#shark-tags-list) |
| F03 AC-2..AC-4 | `AddTag` normalizes/authorizes/creates/records; invalid name → `*ValidationError`; missing cache + no `--pass` → `*UnauthorizedError` | tags.md → [`shark tags add`](#shark-tags-add) + [Authorization and Password Cache](#authorization-and-password-cache) |
| F03 AC-5..AC-7 | `RemoveTag` blocks when in use unless `--force`; `--force` deletes with associations; unused name deletes without `--force` | tags.md → [`shark tags rm`](#shark-tags-rm) (In-use protection) |
| F03 AC-8..AC-10 | `RenameTag`: collision → `*ConflictError`; same-name → `*ValidationError`; success updates only `tags.name` (closes SC-5) | tags.md → [`shark tags rename`](#shark-tags-rename) |
| F03 AC-11..AC-12, AC-19..AC-20, AC-22 | `RecordSuccess` non-fatal; accessor wiring; package import boundaries; OTel attributes | Internal — not user-facing |
| F03 AC-13 | `shark tags list --json` on empty vocab prints `[]` exit 0 | tags.md → [`shark tags list`](#shark-tags-list) (empty vocabulary) |
| F03 AC-14..AC-15 | `--pass wrong` → `incorrect maintainer password`; missing cache → `shark admin maintainer set-password` hint | tags.md → [`shark tags add`](#shark-tags-add) Error handling |
| F03 AC-16 | `shark tags rm voice` with N uses surfaces count and `--force` text | tags.md → [`shark tags rm`](#shark-tags-rm) (In-use protection) |
| F03 AC-17 | `shark tags rm nonexistent` exits with vocabulary listing and `shark tags add` example | tags.md → [`shark tags rm`](#shark-tags-rm) Not-found error |
| F03 AC-18 | `shark tags rename voice audio` plain + JSON shapes | tags.md → [`shark tags rename`](#shark-tags-rename) |
| F03 AC-21 | `tags.md` documents four subcommands, `--pass`, `--force`, JSON shapes; linked from README | This document (tags.md) + [README.md](README.md#vocabulary) |

### F04 — Entity Tag Attachment and Enforcement

| ID | Criterion (summary) | Documented in |
|---|---|---|
| F04 AC-1..AC-8 | Service-layer attach/detach behaviour: name resolution, normalization, dedup, idempotency, no-op on missing | tags.md → [Applying Tags During Create/Update](#applying-tags-during-createupdate) + [Retroactive Tagging](#retroactive-tagging-shark-entity-tag-addrm) |
| F04 AC-9..AC-12 | `EnforceRequired` semantics across entity types and config shapes | tags.md → [Required-Tag Enforcement](#required-tag-enforcement) → [configuration.md → tag_required_for](configuration.md#tag_required_for) |
| F04 AC-13..AC-14 | Constructor invariants; `AttachMany` is not maintainer-gated | tags.md → [Applying Tags During Create/Update](#applying-tags-during-createupdate) ("Tag attachments are NOT maintainer-gated") |
| F04 AC-15..AC-18 | Per-entity-service create/update wiring (six entities) | tags.md → [Applying Tags During Create/Update](#applying-tags-during-createupdate) (covers all six entity families on create + update) |
| F04 AC-19 | `shark task create … --tag=voice --tag=auth` attaches both | tags.md → [Applying Tags During Create/Update → Examples](#applying-tags-during-createupdate) |
| F04 AC-20 | Idempotency on update: duplicate `--tag` values produce one row | tags.md → [Applying Tags During Create/Update → Idempotency](#applying-tags-during-createupdate) |
| F04 AC-21 | `--tag=does-not-exist` exits with vocabulary list + `To add it: shark tags add does-not-exist` | tags.md → [Unregistered Tag Errors](#unregistered-tag-errors) |
| F04 AC-22 | `tag_required_for: ["task"]` blocks task create without `--tag`; epic create unaffected | tags.md → [Tag-Required Errors](#tag-required-errors) + [Required-Tag Enforcement](#required-tag-enforcement) → [configuration.md → tag_required_for](configuration.md#tag_required_for) |
| F04 AC-23 | `shark bug tag add B001 voice` attaches once; re-running is idempotent | tags.md → [Retroactive Tagging → `shark <entity> tag add`](#shark-entity-tag-add) |
| F04 AC-24 | `shark bug tag rm B001 voice` detaches; re-running is idempotent | tags.md → [Retroactive Tagging → `shark <entity> tag rm`](#shark-entity-tag-rm) |
| F04 AC-25 | `shark bug tag rm B001 does-not-exist` (vocab does not contain) exits 1 with vocabulary snippet + remediation | tags.md → [Retroactive Tagging → `shark <entity> tag rm`](#shark-entity-tag-rm) |
| F04 AC-26 | `shark idea tag add` works identically to `shark task tag add` | tags.md → [Retroactive Tagging → `shark <entity> tag add`](#shark-entity-tag-add) (examples include `shark idea tag add`) |
| F04 AC-27 | `tag_required_for` JSON round-trip preserves slice; missing field unmarshals to nil | [configuration.md → tag_required_for](configuration.md#tag_required_for) |
| F04 AC-28 | Error message strings (`tag is not registered: <name>`, `at least one tag is required for <entityType>`) | tags.md → [Unregistered Tag Errors](#unregistered-tag-errors) + [Tag-Required Errors](#tag-required-errors) |

### F05 — Tag-Based Querying in List and Search

| ID | Criterion (summary) | Documented in |
|---|---|---|
| F05 AC-1..AC-8 | `EntityIDsByTags` + `FilterEntityIDs` semantics (sorted, AND intersection, nil/empty handling, normalization, dedup, repository invariants) | tags.md → [Filtering by Tag](#filtering-by-tag---tag-on-list-and-search) (AND semantics + Performance) |
| F05 AC-9..AC-10 | `ListTagsForEntity` / `AttachedTagNamesByIDs` (used by viewer for batch decoration) | tags.md → [Web Viewer Tag Integration](#web-viewer-tag-integration) |
| F05 AC-11..AC-16 | `<Entity>Service.ListXxx(...Tags: …)` filters across all seven list services | tags.md → [Filtering by Tag → Commands that accept --tag](#filtering-by-tag---tag-on-list-and-search) |
| F05 AC-17..AC-19 | `SearchService.SearchAll(... TagFilters …)` semantics including unregistered-tag error | tags.md → [Filtering by Tag](#filtering-by-tag---tag-on-list-and-search); [discovery-commands.md](discovery-commands.md) |
| F05 AC-20 | `GetXxxWithTags`: returns `[]` for none, nil for missing tagSvc | tags.md → [Filtering by Tag](#filtering-by-tag---tag-on-list-and-search) (rendering); [task-commands.md](task-commands.md) (`shark task get` `Tags:` row) |
| F05 AC-21..AC-23 | `shark list [--tag=]`, `shark list E## [--tag=]`, `shark list E## F## [--tag=]` | tags.md → [Filtering by Tag → Examples](#filtering-by-tag---tag-on-list-and-search); also [core-commands.md](core-commands.md) |
| F05 AC-24 | `shark task list --tag=voice --tag=auth` AND semantics | tags.md → [Filtering by Tag → AND semantics](#filtering-by-tag---tag-on-list-and-search) |
| F05 AC-25 | Per-entity `list --tag=` for bug, change, idea, feature (in epic), epic | tags.md → [Filtering by Tag → Examples](#filtering-by-tag---tag-on-list-and-search) |
| F05 AC-26 | `shark search "..." --tag=voice` | tags.md → [Filtering by Tag → Examples](#filtering-by-tag---tag-on-list-and-search); [discovery-commands.md](discovery-commands.md) |
| F05 AC-27 | `shark list --tag=does-not-exist` exits 3 with vocabulary snippet + `To add it:` line | tags.md → [Filtering by Tag → Unregistered tag on list/search path](#unregistered-tag-on-listsearch-path-ac-27) |
| F05 AC-28 | `shark task get` renders `Tags: a, b` (or `Tags: (none)`); `--json` carries `"tags": [...]` | tags.md → [Filtering by Tag](#filtering-by-tag---tag-on-list-and-search) (rendering note); [task-commands.md](task-commands.md) (`shark task get` output) |
| F05 AC-29 | No regression: `shark list` with no `--tag` issues zero extra SQL | tags.md → [Filtering by Tag → Performance](#filtering-by-tag---tag-on-list-and-search) |
| F05 AC-30 | Nil `tagSvc` permits tag-free lists; `Tags: non-empty` returns `*TagFilterUnavailableError` | Internal — graceful-degradation fallback |
| F05 AC-31 | UAT-1 end-to-end across six entity types | Integration UAT — covered by SC-1 row above |

### F06 — Web Viewer Tag Integration

| ID | Criterion (summary) | Documented in |
|---|---|---|
| F06 AC-01..AC-03 | `GET/POST /api/v1/viewer/tags` semantics (sorted vocabulary, no mutation surface) | tags.md → [Web Viewer Tag Integration → HTTP API summary](#web-viewer-tag-integration) |
| F06 AC-04..AC-07 | Hierarchy DTOs carry non-null `tags`; `?tag=` filter prunes with AND semantics | tags.md → [Web Viewer Tag Integration → HTTP API summary](#web-viewer-tag-integration) |
| F06 AC-08..AC-10 | Unregistered tag returns 400 envelope; feature-not-found takes precedence over bad tag | tags.md → [Web Viewer Tag Integration → Unregistered tag on the viewer API](#unregistered-tag-on-the-viewer-api) |
| F06 AC-11..AC-13 | UI renders `.tag-chip` row; tag filter populated from `/api/v1/viewer/tags`; no add/rename/remove controls | tags.md → [Web Viewer Tag Integration → What the viewer renders](#web-viewer-tag-integration); [status-viewer-ui.md](../status-viewer-ui.md) |
| F06 AC-14 | TagReader-not-wired graceful degradation: `{"tags":[]}`, all DTO `tags` `[]`, no 500 | tags.md → [Web Viewer Tag Integration → Graceful degradation](#graceful-degradation) |
| F06 AC-15..AC-16 | Viewer never imports `MaintainerGate`; hierarchy issues at most 6 extra SQL statements | tags.md → [Web Viewer Tag Integration](#web-viewer-tag-integration); architectural invariants enforced by code review |
| F06 AC-17 | Quality gate (`make fmt && make lint && make test`) passes with F06 merged | Internal — quality gate, not user-facing reference |
