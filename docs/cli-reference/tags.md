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

## Required-Tag Enforcement

Tags can be made mandatory at creation time for specific entity types via the `tag_required_for` field in `.sharkconfig.json`. When configured, the corresponding `create` commands exit with process code **1** (internal class `tag_required`) if no `--tag` is supplied.

See [Configuration → `tag_required_for`](configuration.md#tag_required_for) for the field schema, allowed values, and a worked example.
