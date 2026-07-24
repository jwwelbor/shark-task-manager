# Configuration Reference

Complete reference for Shark CLI configuration — both the `.sharkconfig.json` file and `shark config` commands.

## Configuration File (`.sharkconfig.json`)

The `.sharkconfig.json` file is automatically created by `shark admin init` and contains database, UI, and workflow settings.

**Location**: Project root directory (auto-detected by walking up from current directory).

### File Structure

```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  },
  "viewer": "glow",
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": false,
  "require_rejection_reason": true,
  "advance_guard": {
    "enabled": false,
    "mode": "session_from_status",
    "allow_repeat_with_force": true
  },
  "shark_data_path": "shark-data",
  "workflow_config": "shark-data/workflow/",
  "console_width": 0,
  "max_parallel_items": 5,
  "web": {
    "port": 7777
  },
  "default_agent": null,
  "default_epic": null,
  "last_sync_time": "2026-01-16T23:22:45-06:00",
  "epic_workflow": { },
  "feature_workflow": { },
  "status_flow": { },
  "status_metadata": { },
  "status_flow_version": "1.0",
  "special_statuses": { },
  "bug_workflow": { },
  "change_workflow": { },
  "tag_required_for": []
}
```

The `bug_workflow` and `change_workflow` keys configure the workflows for the two standalone defect/change entity types. See [Bug Workflow Configuration](#bug-workflow-configuration) and [Change-Card Workflow Configuration](#change-card-workflow-configuration) below.

The optional `tag_required_for` key gates entity creation on the presence of `--tag`; see [`tag_required_for`](#tag_required_for) below.

### Database Configuration

#### Local SQLite (Default)

```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}
```

#### Turso Cloud

```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-yourorg.turso.io",
    "auth_token_file": "/home/user/.turso/shark-token",
    "skip_migrations": true
  }
}
```

See [Turso Quickstart](../TURSO_QUICKSTART.md) for cloud setup.

<a id="database-skip_migrations"></a>
#### `database.skip_migrations`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `database.skip_migrations` | bool | `false` | When `true`, skips the per-command schema/DDL check (recommended for Turso to avoid ~2-second overhead on every shark invocation). Local SQLite users typically leave this `false`. |

> **Schema bumps (v14 → v15):** Recent migrations added the tag vocabulary
> tables (v14) and `size`/`size_label` columns (v15). When `skip_migrations`
> is `true`, `ApplySchemaIfNeeded` still detects any gap between the recorded
> `schema_version` and `CurrentSchemaVersion` and runs pending migrations
> automatically — no manual toggle is needed. Simply upgrade the binary and
> run any `shark` command. Local SQLite users do not need to do anything;
> migrations always run on local databases.
>
> See
> [Initialization → Migrating an existing project to v15](initialization.md#migrating-an-existing-project-to-v15-e07-size-field--e28-tagging)
> for details.

### UI Preferences

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `color_enabled` | bool | `true` | Enable ANSI color output |
| `json_output` | bool | `false` | Default to JSON output for all commands |
| `interactive_mode` | bool | `false` | Enable interactive prompts |
| `require_rejection_reason` | bool | `true` | Require reason when rejecting tasks |
| `advance_guard` | object | disabled | Replay protection for `shark status advance`. See [Advance Guard](#advance-guard) below. |
| `viewer` | string | `"cat"` | External viewer for `shark view` (e.g., `"glow"`, `"nano"`) |
| `template_directory` | string | unset | Optional explicit prompt directory. Leave unset to derive prompts from `shark_data_path`. |
| `shark_data_path` | string | `"shark-data"` | Content-bundle root holding `skills/`, `prompts/`, `file_templates/`, `agents/`, `workflow/`, and `overrides/`. See [Shark Data Path](#shark-data-path) below. |
| `console_width` | int | `0` (auto-detect) | Override the console width used by list-view tables. See [Console Width](#console-width) below. |

<a id="shark-data-path"></a>
#### Shark Data Path

`shark_data_path` selects the **content-bundle root** — the directory holding
`skills/`, `prompts/`, `file_templates/`, `agents/`, `workflow/`, and `overrides/`. It is
**independent of `workflow_config`** (which selects only the active workflow
graph / status routing); `workflow_config` never drives the bundle root.

All bundle-aware commands resolve this one value the same way —
`shark admin install-shark-data`, `shark admin upgrade`, and
`shark admin validate-data` all materialize, refresh, or validate the resolved
root, and prompt + workflow resolution read from it:

- **Default** (`"shark-data"`): resolves to `<project-root>/shark-data`,
  preserving historical behavior.
- **Relative path**: resolved against the project root. A path that escapes the
  project root via `..` is **rejected** — use an absolute path for shared
  bundles.
- **Absolute path** (or a `~/`-prefixed path, expanded to your home directory):
  honored verbatim. This is the shared-bundle case — point multiple projects at
  one bundle on a shared mount, monorepo, or submodule.

```json
{
  "shark_data_path": "shark-data",
  "workflow_config": "shark-data/workflow"
}
```

```json
{
  "shark_data_path": "~/shared/shark-bundles/standard"
}
```

#### `workflow_config`

`workflow_config` selects the active workflow graph and status routing. Leave it
absent or empty to use Shark's embedded default workflows, set it to a directory
of per-entity YAML files, or set it to a YAML master index file.

Do not point `workflow_config` at `.sharkworkflow.json` or another JSON workflow
file. To migrate an older project to embedded defaults, remove the field and
remove or rename any root `.sharkworkflow.json`. Or run
`shark admin install-shark-data` to extract editable YAML and set
`workflow_config` to the installed bundle's `workflow/` directory.

<a id="console-width"></a>
#### Console Width

`console_width` controls the column width used by list-view tables (`shark list`,
`shark task list`, `shark feature list`, `shark recent`, `shark history`, etc.).
Width-sensitive renderers consult this value to decide how wide to draw the
Title / Notes column so the table fills the terminal without overflowing.

**Resolution order** (highest priority first):

1. **`console_width` from `.sharkconfig.json`** when set to a positive integer
   (clamped to a minimum of 40 cols so list views remain legible).
2. **Auto-detected terminal width** via `term.GetSize` on stdout — the default
   when `console_width` is `0` or missing.
3. **Built-in fallback** (120 cols) — used when stdout is not a TTY (e.g., piped
   output, CI logs).

**When to set it explicitly:**

- You want fixed-width output regardless of your terminal size — e.g. for
  screenshots, code reviews, or sharing in chat where wrap behavior matters.
- You're running `shark` in a wrapper / TUI that lies about its size to
  `term.GetSize`.
- You want narrower output than your terminal width to keep tables readable
  side-by-side with other panes.

**Example** — cap list views at 100 cols even on a wide terminal:

```json
{
  "console_width": 100
}
```

**Override-by-flag is intentionally not supported.** Set `console_width` in
config when you want a non-default; otherwise auto-detect handles the common
case. Use `--no-color` to drop ANSI codes if your wrapper has trouble with
escape sequences but renders width correctly.

<a id="advance-guard"></a>
#### Advance Guard

`advance_guard` enables replay protection for parent-loop driven
`shark status advance` calls. When enabled, guarded advances must include both
`--session <sid>` and `--from-status <expected-status>`. Shark rejects the
advance when either condition is true:

- The entity is no longer at `from_status`.
- The same `entity + session + from_status + outcome` was already consumed.

This is designed for orchestrators such as `/shark-rider run`, not for ordinary
manual CLI use, so the default is disabled for backward compatibility.

**Enable it**:

```json
{
  "advance_guard": {
    "enabled": true,
    "mode": "session_from_status",
    "allow_repeat_with_force": true
  }
}
```

With that config enabled, the parent loop should advance like this:

```bash
shark status advance E38-F07 --outcome fail --session "$SID" --from-status code_review
```

If you need an audited override after a replay rejection, use `--force-repeat`
with a reason, but only when `allow_repeat_with_force` is `true`:

```bash
shark status advance E38-F07 --outcome fail \
  --session "$SID" \
  --from-status code_review \
  --force-repeat \
  --reason "operator override after manual inspection"
```

**Turn it off** again:

```json
{
  "advance_guard": {
    "enabled": false
  }
}
```

**Fields**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `advance_guard.enabled` | bool | `false` | Master on/off switch. `false` preserves historical `status advance` behavior. |
| `advance_guard.mode` | string | `"session_from_status"` | Guard strategy. The current implementation supports `session_from_status`. |
| `advance_guard.allow_repeat_with_force` | bool | `false` when absent | Allows `--force-repeat --reason ...` to override a replay rejection. |

<a id="max-parallel-items"></a>
#### `max_parallel_items`

Caps the number of tied candidates `shark plan` returns for an equally-ranked
tier — bare epic selection, one-level hierarchy selection (`shark plan
<epic|feature>`), and standalone-collection selection (`shark plan
bugs|change-cards|tech-debt`). It does not change rank/order semantics, only
how many tied candidates are included in the response.

```json
{
  "max_parallel_items": 5
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_parallel_items` | int | `5` | Maximum tied candidates returned per planning scope. Absent, zero, or negative values fall back to the default. Set to `1` for deterministic singleton selection. |

### Web Server Configuration

The `web` key configures the `shark web` dashboard server.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `web.port` | int | `0` (use 7777) | TCP port for `shark web`; `0` means auto-select from 7777–7790 |

**Port selection priority** (highest to lowest):
1. `--port` CLI flag — exact port, fails if busy
2. `web.port` in `.sharkconfig.json` — exact port, fails if busy
3. Built-in default: try 7777, then 7778–7790

**Example** — always start on port 9000:

```json
{
  "web": {
    "port": 9000
  }
}
```

You can still override it on the command line:

```bash
shark web --port 8888   # uses 8888 regardless of web.port in config
```

### Default Values

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_agent` | string/null | `null` | Default agent type for task creation and filtering |
| `default_epic` | string/null | `null` | Default epic for task/feature creation |

<a id="maintainer"></a>
### Maintainer

The optional `maintainer` object configures the maintainer authorization gate
(`internal/auth/maintainer`) that protects mutating tag-vocabulary commands
(`shark tags add|rm|rename`) and any future destructive admin operations
(e.g., `shark admin purge`). When the object is absent the gate refuses every
guarded command with a setup hint.

#### JSON shape

```json
{
  "maintainer": {
    "password_hash": "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92",
    "cache_window_seconds": 300
  }
}
```

#### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `password_hash` | string | `""` | Lowercase-hex SHA-256 digest of the maintainer password (no salt). Written by `shark admin maintainer set-password`; never edit by hand. An empty string is treated the same as an absent `maintainer` object. |
| `cache_window_seconds` | int | `60` | Lifetime, in seconds, of a successful sudo-style authorization. A value of `0` or any negative number falls back to the 60-second default (see `MaintainerConfig.CacheWindow()` in `internal/config/maintainer.go`). |

> **Hash details (source: `internal/auth/maintainer/gate.go`):** Passwords are
> hashed with `crypto/sha256` (no salt, no PBKDF/Argon stretching — see
> ADR-F02-6) and compared with `crypto/subtle.ConstantTimeCompare` to defeat
> timing oracles. Plaintext is **never** persisted; the
> `shark admin maintainer set-password` command computes the digest in process
> and writes only `password_hash` back to `.sharkconfig.json`, preserving every
> other key in the file.

#### Behavior matrix

The gate's `Authorize(ctx, providedPass)` method (`internal/auth/maintainer/gate.go`)
returns `nil` on success or `*UnauthorizedError` on every failure. The
`Reason` field is a stable string safe for programmatic handling.

| `maintainer` object state | Caller supplied `--pass` | `gate.Authorize` result |
|---------------------------|--------------------------|--------------------------|
| Absent (key not in `.sharkconfig.json`) | (any) | `*UnauthorizedError{Reason: "missing_config"}` — `Error()` includes the literal `shark admin maintainer set-password` setup hint. |
| Present, `password_hash == ""` | (any) | Same as absent: `*UnauthorizedError{Reason: "missing_config"}`. |
| Present with hash | matches | `nil` (authorized; cache entry is refreshed by `RecordSuccess`). |
| Present with hash | does **not** match | `*UnauthorizedError{Reason: "wrong_password"}`. |
| Present with hash | omitted, no live cache entry | `*UnauthorizedError{Reason: "expired_cache"}`. |
| Present with hash | omitted, cache entry's `pass_hash` differs from current `password_hash` | `*UnauthorizedError{Reason: "hash_mismatch_after_rotation"}` — fires automatically the first time a guarded command runs after `set-password` rotates the password. |
| Present with hash | omitted, cache entry within window and matching hash | `nil` (cache hit). |

The CLI surfaces `*UnauthorizedError.Error()` on stderr and, for the
`missing_config` reason only, an additional `UserHint()` line directing the
user to `shark admin maintainer set-password`. The end-to-end CLI rendering
(JSON envelope, exit-code mapping to `unauthorized` / exit 3) is documented
once in [Tags → Authorization and Password Cache](tags.md#authorization-and-password-cache);
do not duplicate it here.

#### Cache file

`Gate.RecordSuccess` writes a JSON session entry to
`$XDG_CACHE_HOME/shark/<project-hash>/maintainer.session` (falling back to
`os.UserCacheDir()` when `XDG_CACHE_HOME` is unset).
`<project-hash>` is the SHA-256 of the absolute project root path so two
checkouts of the same project on different paths never share a cache. The
file is mode `0600` inside a `0700` directory and is written via the
temp-file + `rename` pattern so concurrent readers never observe a partial
file. A missing or malformed cache file is silently treated as a cache miss
(per REQ-NF-003) — it is never reported as an authorization error.

The cache entry shape is internal but stable for the lifetime of E28-F02:

```json
{
  "last_success": "2026-04-25T11:42:13.987654321-05:00",
  "pass_hash":    "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92"
}
```

#### CLI commands

Only one subcommand exists today; there is **no** `verify` and **no** `clear`
command. The cache can be reset out-of-band by deleting the
`maintainer.session` file under the project's cache directory.

```bash
# Pass the plaintext as a flag value
shark admin maintainer set-password --password "hunter2"

# Pipe-friendly (reads one newline-terminated line from stdin)
printf '%s\n' "hunter2" | shark admin maintainer set-password --password-stdin

# Re-running set-password rotates the password — every existing cache entry
# is invalidated on the next gate check via the hash_mismatch_after_rotation
# reason, so subsequent guarded commands must re-authorize with --pass.
shark admin maintainer set-password --password "new-secret"
```

| Flag | Description |
|------|-------------|
| `--password <plaintext>` | Plaintext password supplied as a flag value. Highest priority. Not stored — only its SHA-256 digest is written to `.sharkconfig.json`. |
| `--password-stdin` | Read one newline-terminated line from `os.Stdin`. Used when `--password` is absent. |

When both sources are absent the command fails with
`set-password: password cannot be empty` (an interactive prompt is not yet
implemented). On success the command prints two lines — first
`Maintainer password configured successfully.` (success message), then
`Run 'shark admin maintainer set-password' again to rotate the password.`
(informational follow-up) — without ever echoing the plaintext or the hash.

**Cross-reference:** See [Tags → Authorization and Password Cache](tags.md#authorization-and-password-cache)
for the user-facing semantics of `--pass` on `shark tags add|rm|rename`, the
cache hit/miss flow across a session, and the JSON error envelope shape
(`{"error":"unauthorized","message":"…"}`).

---

<a id="tag_required_for"></a>
### `tag_required_for`

The optional `tag_required_for` field is a JSON array of entity-type strings.
Entity types listed here require **at least one `--tag` flag at creation
time**. The check is enforced inside the entity service layer
(`enforceTagsRequired` in `internal/services/helpers.go`, called from each of
`task_service.go`, `feature_service.go`, `epic_service.go`,
`bug_service.go`, `change_card_service.go`, and `idea_service.go`) before
any key allocation, file write, or database insert — failures produce no
side effects.

#### JSON shape

```json
{
  "tag_required_for": ["task"]
}
```

The internal Go field is `Config.TagRequiredForTypes` (`internal/config/config.go`)
but the JSON key on disk is always `tag_required_for`. The accessor
`Config.TagRequiredFor()` returns a defensive copy.

#### Type and defaults

`[]string` — array of lowercase entity-type names. Absent, `null`, or empty
(`[]`) disables enforcement for all types.

#### Supported entity-type values

Matching is case-sensitive against `models.EntityType.String()` output. The
allowed values map directly to the six entity create commands:

| Value     | Applies to command           | EntityType constant         |
|-----------|------------------------------|-----------------------------|
| `task`    | `shark task create`          | `models.EntityTypeTask`     |
| `feature` | `shark feature create`       | `models.EntityTypeFeature`  |
| `epic`    | `shark epic create`          | `models.EntityTypeEpic`     |
| `bug`     | `shark bug create`           | `models.EntityTypeBug`      |
| `change`  | `shark change create`        | `models.EntityTypeChange`   |
| `idea`    | `shark idea create`          | `models.EntityTypeIdea`     |

> **Important (ADR-F04-4):** Matching is case-sensitive. A misspelled or
> mis-cased entry (e.g., `"Task"`, `"tasks"`, `"changes"`) silently disables
> enforcement for that type — `EnforceRequired` will simply never find a
> match. Use the exact lowercase strings above.

#### Behavior

- **Creation-time only.** `tag_required_for` only affects the
  `shark <entity> create` commands. The `update` commands and the
  retroactive `shark <entity> tag add|rm` subcommands are explicitly
  **not** subject to this check (see `internal/services/tag_service.go`
  → `EnforceRequired`, which is called only from the create paths).
- **Ordering and duplicates** in the slice are not significant —
  `EnforceRequired` does an exact-match scan.
- **The only way to satisfy the requirement** is to pass one or more
  `--tag <name>` flags on the failing `create` invocation. Retroactive
  `shark <entity> tag add` cannot help on a creation call that has already
  failed (no entity was created to attach to).
- **Tag values must already be registered** in the vocabulary — supplying a
  `--tag` value that is not in the vocabulary fails with
  `*UnregisteredTagError` (a different error class, also exit 3) rather than
  the `tag_required` error covered here. See [Tags → Applying Tags During
  Create/Update](tags.md#applying-tags-during-createupdate).
- **No interaction with the maintainer gate.** Tag attachment on `create` is
  not maintainer-gated; only vocabulary mutation
  (`shark tags add|rm|rename`) is. See [Maintainer](#maintainer) above.

#### Exit code and error message

When the check fails the service layer returns
`*services.TagRequiredError{EntityType: "<entity>"}`
(`internal/services/tag_errors.go`). The CLI maps this to:

| Surface            | Value                                                               |
|--------------------|----------------------------------------------------------------------|
| Process exit code  | **3** (mapped by `tagsErrorCode` in `internal/cli/commands/tags.go`) |
| Internal class code| `tag_required`                                                       |
| Stderr message     | `at least one tag is required for <entity>`                          |

The exit code aligns with the rest of the typed-tag error family
(`unregistered_tag`, `tag_required`, `unauthorized`, `conflict`, `in_use`,
`validation` are all exit 3; `not_found` and `db_error` are exit 1 and 2
respectively).

#### Worked example (UAT-6)

With `.sharkconfig.json` containing:

```json
{
  "tag_required_for": ["task"]
}
```

Attempting to create a task without `--tag` fails:

```bash
$ shark task create E07 F01 "x"
at least one tag is required for task
Available tags:
  audio, backend, voice
Error: exit code 3: at least one tag is required for task
$ echo $?
3
```

(Note the `Available tags:` snippet emitted by
`handleVocabularyErrorWithSnippet`. The remediation line `To add it: …` is
**suppressed** for this error because no specific tag name was rejected — the
user must pick from the listed vocabulary.)

The same task creation with `--tag` succeeds:

```bash
$ shark task create E07 F01 "x" --tag=voice
Created task T-E07-F01-001 ...
```

Other entity types are unaffected because only `task` is listed:

```bash
$ shark epic create "New Epic"            # succeeds (no --tag needed)
$ shark bug create "Login crash"          # succeeds
$ shark feature create E07 "New Feature"  # succeeds
```

#### Error JSON shape (`--json`)

When `--json` is set, two JSON documents are emitted on a failed create —
one on stderr (the inner classification envelope produced by
`writeTagsError`) and one on stdout (the outer `COMMAND_ERROR` envelope
produced by `cmd/shark/main.go` whenever a Cobra `RunE` returns a non-nil
error):

```json
// stderr — stable machine-readable classification
{"error":"tag_required","message":"at least one tag is required for task"}
```

```json
// stdout — generic command-error envelope (every failing shark command)
{"error":true,"code":"COMMAND_ERROR","message":"exit code 3: at least one tag is required for task"}
```

Integrators that need to distinguish `tag_required` from other typed-tag
errors should match on the stderr envelope's `error` field (`tag_required`).
Integrators that only need to detect "any failure" can rely on the stdout
`COMMAND_ERROR` envelope, which is shared across every failing command.

#### Enforcing tags on multiple entity types

```json
{
  "tag_required_for": ["task", "bug"]
}
```

This requires `--tag` on both `shark task create` and `shark bug create`;
the other four families (`epic`, `feature`, `change`, `idea`) remain
unaffected.

**Cross-reference:** See [Tags → Applying Tags During Create/Update](tags.md#applying-tags-during-createupdate)
for the `--tag` flag semantics on every create/update path,
[Tags → Tag-Required Errors](tags.md#tag-required-errors) for the user-facing
error rendering, and [Tags → Retroactive Tagging](tags.md#retroactive-tagging-shark-entity-tag-addrm)
for the per-entity `tag add|rm` subcommands (which are not subject to this
check).

---

### Environment Variables

Shark supports environment variable substitution in config values:

```json
{
  "database": {
    "backend": "$SHARK_DB_BACKEND",
    "url": "$SHARK_DB_URL",
    "auth_token_file": "$SHARK_AUTH_TOKEN_FILE"
  }
}
```

| Variable | Description |
|----------|-------------|
| `SHARK_DB_BACKEND` | Database backend (`local` or `turso`) |
| `SHARK_DB_URL` | Database URL or file path |
| `SHARK_AUTH_TOKEN_FILE` | Path to Turso auth token file |
| `SHARK_OUTPUT` | Default output format (set to `json` for JSON) |

---

## Bug Workflow Configuration

The `bug_workflow` key configures the workflow for bug entities (`B###`). It follows the same structure as `epic_workflow` and `feature_workflow`.

### Structure

```json
{
  "bug_workflow": {
    "version": "1.0",
    "status_flow": {
      "reported": ["triaged", "duplicate", "wont_fix"],
      "triaged": ["in_fix", "wont_fix", "duplicate"],
      "in_fix": ["in_verification", "triaged"],
      "in_verification": ["resolved", "in_fix"],
      "resolved": [],
      "wont_fix": [],
      "duplicate": []
    },
    "status_metadata": {
      "reported": {
        "color": "red",
        "description": "Bug reported, awaiting triage",
        "phase": "planning",
        "progress_weight": 0.05,
        "responsibility": "agent",
        "agent_types": ["business-analyst"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "business-analyst",
          "skills": ["research", "shark"],
          "instruction_template": "bug/reported.tmpl"
        }
      },
      "triaged": {
        "color": "yellow",
        "description": "Triaged, ready for fix",
        "phase": "development",
        "progress_weight": 0.2,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "developer",
          "skills": ["debugging", "implementation"],
          "instruction_template": "bug/triaged.tmpl"
        }
      },
      "in_fix": {
        "color": "blue",
        "description": "Fix in progress",
        "phase": "development",
        "progress_weight": 0.5,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "check_or_resume",
          "agent_type": "developer",
          "skills": ["debugging", "implementation"],
          "instruction_template": "bug/in_fix.tmpl"
        }
      },
      "in_verification": {
        "color": "cyan",
        "description": "Fix applied, awaiting verification",
        "phase": "review",
        "progress_weight": 0.8,
        "responsibility": "agent",
        "agent_types": ["qa"],
        "orchestrator_action": {
          "action": "check_or_resume",
          "agent_type": "qa",
          "skills": ["quality"],
          "instruction_template": "bug/in_verification.tmpl"
        }
      },
      "resolved": {
        "color": "green",
        "description": "Bug verified as fixed",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Bug {id} resolved."
        }
      },
      "wont_fix": {
        "color": "gray",
        "description": "Will not be fixed",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Bug {id} closed as wont_fix."
        }
      },
      "duplicate": {
        "color": "gray",
        "description": "Duplicate of another bug",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Bug {id} closed as duplicate."
        }
      }
    },
    "special_statuses": {
      "_start_": ["reported"],
      "_complete_": ["resolved", "wont_fix", "duplicate"]
    }
  }
}
```

### Template Variables for Bugs

Available in `instruction_template` for bug workflow:

| Variable | Description |
|----------|-------------|
| `{id}` | Bug key (e.g., `B001`) |
| `{title}` | Bug title |
| `{severity}` | Bug severity (`critical`, `high`, `medium`, `low`) |
| `{file_path}` | Path to bug markdown file |
| `{linked_entity_type}` | Type of linked entity (`epic`, `feature`, `task`) |
| `{linked_entity_key}` | Key of the linked entity |

---

## Change-Card Workflow Configuration

The `change_workflow` key configures the workflow for change-card entities (`CC-###`).

### Structure

```json
{
  "change_workflow": {
    "version": "1.0",
    "status_flow": {
      "proposed": ["approved", "declined"],
      "approved": ["in_progress", "declined"],
      "in_progress": ["completed", "approved"],
      "completed": [],
      "declined": []
    },
    "status_metadata": {
      "proposed": {
        "color": "yellow",
        "description": "Awaiting scope assessment and approval",
        "phase": "planning",
        "progress_weight": 0.1,
        "responsibility": "agent",
        "agent_types": ["business-analyst"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "business-analyst",
          "skills": ["assessment", "shark"],
          "instruction_template": "change/proposed.tmpl"
        }
      },
      "approved": {
        "color": "cyan",
        "description": "Approved, ready to implement",
        "phase": "development",
        "progress_weight": 0.2,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "developer",
          "skills": ["implementation"],
          "instruction_template": "change/approved.tmpl"
        }
      },
      "in_progress": {
        "color": "blue",
        "description": "Implementation in progress",
        "phase": "development",
        "progress_weight": 0.6,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "check_or_resume",
          "agent_type": "developer",
          "skills": ["implementation"],
          "instruction_template": "change/in_progress.tmpl"
        }
      },
      "completed": {
        "color": "green",
        "description": "Change implemented and verified",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Change-card {id} completed."
        }
      },
      "declined": {
        "color": "red",
        "description": "Change request declined",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Change-card {id} declined."
        }
      }
    },
    "special_statuses": {
      "_start_": ["proposed"],
      "_complete_": ["completed", "declined"]
    }
  }
}
```

### Template Variables for Change-Cards

Available in `instruction_template` for change-card workflow:

| Variable | Description |
|----------|-------------|
| `{id}` | Change-card key (e.g., `CC-001`) |
| `{title}` | Change-card title |
| `{priority}` | Priority level (1–10) |
| `{requested_by}` | Name of requester |
| `{file_path}` | Path to change-card markdown file |

---

## Configuration Commands

### `shark config show`

Display current configuration including file location and all settings.

```bash
shark config show                    # Show full config
shark config show --patterns         # Show only pattern configuration
shark config show --json             # JSON output
```

### `shark config validate`

Check configuration file for errors and validate settings.

```bash
shark config validate
```

### `shark config validate-patterns`

Validate all regex patterns in `.sharkconfig.json`. Reports results grouped by entity type. Exits non-zero if errors found (CI-friendly).

```bash
shark config validate-patterns
shark config validate-patterns --json
```

### `shark config test-pattern`

Test a regex pattern against a test string. Shows captured groups and validates for entity type.

```bash
shark config test-pattern \
  --pattern="E(?P<number>\d{2})-(?P<slug>[a-z-]+)" \
  --test-string="E04-task-mgmt" \
  --type=epic

shark config test-pattern \
  --pattern="T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3})\.md" \
  --test-string="T-E04-F07-003.md" \
  --type=task
```

**Flags:**
- `--pattern` — Regex pattern to test
- `--test-string` — String to test against
- `--type` — Entity type: `epic`, `feature`, `task` (default: `epic`)

### `shark config add-pattern`

Add a pattern preset to configuration. Patterns are appended; duplicates are skipped.

```bash
shark config add-pattern --preset=special-epics
shark config add-pattern --preset=numeric-only
```

### `shark config list-presets`

List all available pattern presets with descriptions.

```bash
shark config list-presets
```

### `shark config show-preset`

Show full pattern structure for a specific preset in JSON format.

```bash
shark config show-preset standard
shark config show-preset special-epics
```

### `shark config get-format`

Get the configured generation format template for an entity type.

```bash
shark config get-format --type=task
shark config get-format --type=epic --json
```

### `shark config get-status-action`

Get the orchestrator action definition for a specific status. Useful for debugging workflow configuration.

```bash
shark config get-status-action ready_for_development
shark config get-status-action ready_for_development --task=T-E01-F03-002
shark config get-status-action blocked --json
```

---

## Example Configurations

### AI Agent

```json
{
  "database": { "backend": "local", "url": "./shark-tasks.db" },
  "color_enabled": false,
  "json_output": true,
  "interactive_mode": false,
  "require_rejection_reason": true,
  "shark_data_path": "shark-data",
  "workflow_config": "shark-data/workflow/"
}
```

### Human Developer

```json
{
  "database": { "backend": "local", "url": "./shark-tasks.db" },
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": true,
  "viewer": "glow",
  "shark_data_path": "shark-data",
  "workflow_config": "shark-data/workflow/"
}
```

### Team with Cloud Database

```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-team.turso.io",
    "auth_token_file": "$HOME/.turso/team-token"
  },
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": true,
  "require_rejection_reason": true,
  "shark_data_path": "shark-data",
  "workflow_config": "shark-data/workflow/",
  "default_epic": "E07",
  "default_agent": "developer"
}
```

---

## Best Practices

1. **Never commit auth tokens** — Use `auth_token_file` or environment variables
2. **Add `.sharkconfig.json` to `.gitignore`** if it contains sensitive data; check in a `.sharkconfig.template.json` instead
3. **AI agents**: Set `json_output: true`, `color_enabled: false`, `interactive_mode: false`
4. **Multi-environment**: Use `--config` flag to switch between dev/staging/prod configs

## Observability

The optional `observability` key in `.sharkconfig.json` configures OpenTelemetry tracing, metrics, and structured logging (including optional file-based log output via `log_file` / `SHARK_LOG_FILE`). The subsystem is disabled by default with zero overhead when the `observability` key is absent.

For the complete field reference, environment variable overrides, and example configurations, see:

- [Observability Developer Guide](../guides/observability.md) — usage examples, troubleshooting, and file-destination setup
- [Observability Configuration Reference](../guides/observability-config-reference.md) — every field, every environment variable, and complete example configurations

## Related Documentation

- [Workflow Configuration](workflow-configuration.md) - Workflow system reference
- [Global Flags](global-flags.md) - CLI-level configuration flags
- [Setup Commands](setup-commands.md) - `shark admin init` and related setup
- [Turso Quickstart](../TURSO_QUICKSTART.md) - Cloud database setup
- [Observability Developer Guide](../guides/observability.md) - OTel tracing, metrics, and structured logging
- [Observability Configuration Reference](../guides/observability-config-reference.md) - Complete `observability.*` field reference
