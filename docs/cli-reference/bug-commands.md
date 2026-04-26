# Bug Commands

Complete reference for `shark bug` subcommands. Bugs are standalone entities for tracking defects with their own workflow and key format.

---

## Key Format

Bugs use the `B###` key format (e.g., `B001`, `B042`). Keys are auto-generated sequentially on creation.

---

## `shark bug create`

Create a new bug report.

```bash
shark bug create <title> [flags]
```

**Arguments:**
- `title` — Bug title (required)

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--severity` | string | `medium` | Bug severity: `critical`, `high`, `medium`, `low` |
| `--link` | string | — | Link to existing entity (epic key, feature key, or task key) |
| `--file` | string | — | Custom file path for the bug markdown file |
| `--force` | bool | `false` | Overwrite existing file if it exists |
| `--size` | string | — | Size estimate: `XS`, `S`, `M`, `L`, `XL`, `XXL` or numeric `1`, `2`, `3`, `5`, `8`, `13`. Optional; absence stores NULL. |
| `--tag` | string | — | Tag to apply (repeatable). Tag must be registered; see `shark tags list`. |

**Examples:**

```bash
# Basic bug report
shark bug create "Login page crashes on submit"

# Critical bug linked to a feature
shark bug create "Login page crashes on submit" --severity=critical --link=E07-F01

# High severity bug linked to a specific task
shark bug create "JWT tokens not expiring" --severity=high --link=E07-F01-003

# With custom file path
shark bug create "Race condition in async handler" --file=docs/bugs/B001-race-condition.md

# With size estimate
shark bug create "Auth service memory leak" --severity=high --size=L

# With one or more tags (tags must already be in the vocabulary)
shark bug create "Login crashes" --severity=high --tag=auth
shark bug create "Race in login flow" --tag=auth --tag=voice
```

**Output (default):**
```
Created bug B001: Login page crashes on submit
  File: docs/plan/bugs/B001.md
  Severity: critical
  Status: reported
```

**Output (`--json`, with `--size` flag set):**
```json
{
  "key": "B001",
  "title": "Auth service memory leak",
  "status": "reported",
  "severity": "high",
  "size": 5,
  "file_path": "docs/plan/bugs/B001.md",
  "created_at": "2026-03-05T10:00:00Z"
}
```

> **Note:** `size` is omitted from the JSON response when `--size` is not provided. `size_label` is only emitted by `get` / `status` commands; use `shark bug get <key> --field size_label` if you need the label after creation.


---

## `shark bug get`

Get details of a specific bug.

```bash
shark bug get <key> [flags]
```

**Arguments:**
- `key` — Bug key (e.g., `B001`)

**Flags:**
| Flag | Description |
|------|-------------|
| `--json` | Machine-readable JSON output |
| `--field <name>` | Extract single field (implies `--json`) |

**Examples:**

```bash
shark bug get B001
shark bug get B001 --json
shark bug get B001 --field status
shark bug get B001 --field severity
```

**Output (default):**
```
Bug B001: Login page crashes on submit
  Status:   reported
  Severity: critical
  Linked:   feature E07-F01
  Created:  2026-03-05
```

---

## `shark bug list`

List bugs with optional filtering.

```bash
shark bug list [flags]
```

**Flags:**
| Flag | Type | Description |
|------|------|-------------|
| `--status` | string | Filter by status (e.g., `reported`, `in_fix`) |
| `--severity` | string | Filter by severity: `critical`, `high`, `medium`, `low` |
| `--link` | string | Filter by linked entity key |
| `--json` | bool | Machine-readable JSON output |

**Examples:**

```bash
# All open bugs
shark bug list

# Filter by severity
shark bug list --severity=critical
shark bug list --severity=high

# Filter by status
shark bug list --status=in_fix
shark bug list --status=reported

# Bugs linked to a specific feature
shark bug list --link=E07-F01

# Combined filters
shark bug list --severity=critical --status=reported
```

---

## `shark bug update`

Update bug fields.

```bash
shark bug update <key> [flags]
```

**Arguments:**
- `key` — Bug key (e.g., `B001`)

**Flags:**
| Flag | Type | Description |
|------|------|-------------|
| `--title` | string | New bug title |
| `--severity` | string | New severity: `critical`, `high`, `medium`, `low` |
| `--size` | string | New size: `XS`\|`S`\|`M`\|`L`\|`XL`\|`XXL` or `1`\|`2`\|`3`\|`5`\|`8`\|`13`. Use `clear` to remove (set to NULL). Flag absent = no change. |
| `--tag` | string | Tag to apply additively (repeatable). Empty = no change; use `shark bug tag rm` to detach. |
| `--json` | bool | JSON output |

**Examples:**

```bash
shark bug update B001 --severity=high
shark bug update B001 --title="Login page crashes on mobile submit"
shark bug update B001 --severity=critical --json

# Set or update size
shark bug update B001 --size=M

# Clear size (set back to NULL)
shark bug update B001 --size=clear

# Additive tagging (does not detach existing tags)
shark bug update B001 --tag=auth
shark bug update B001 --tag=auth --tag=voice
```

> `--tag` on update is **additive only**. Repeating a tag already attached is a no-op. Use `shark bug tag rm B001 <name>` to detach a single tag.

---

## `shark bug delete`

Delete a bug.

```bash
shark bug delete <key> [flags]
```

**Arguments:**
- `key` — Bug key (e.g., `B001`)

**Flags:**
| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

**Examples:**

```bash
shark bug delete B001
shark bug delete B001 --force
```

---

## `shark bug triage`

Triage a bug: set severity and advance status from `reported` to `triaged`.

```bash
shark bug triage <key> --severity=<SEVERITY> [flags]
```

**Arguments:**
- `key` — Bug key (e.g., `B001`)

**Flags:**
| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--severity` | string | Yes | Confirmed severity: `critical`, `high`, `medium`, `low` |
| `--json` | bool | No | JSON output |

**Examples:**

```bash
shark bug triage B001 --severity=high
shark bug triage B001 --severity=critical --json
```

**Note:** Triage is a combined operation — it sets the confirmed severity and transitions the bug from `reported` → `triaged`. Use `shark status advance` to continue through subsequent workflow stages.

---

## `shark bug note add`

Add a note to a bug.

```bash
shark bug note add <key> --type=<TYPE> [--content=<CONTENT>] [flags]
```

**Arguments:**
- `key` — Bug key (e.g., `B001`)

**Flags:**
| Flag | Type | Description |
|------|------|-------------|
| `--type` | string | Note type: `comment`, `decision`, `blocker`, `context` |
| `--content` | string | Note content (if omitted, opens editor) |
| `--json` | bool | JSON output |

**Examples:**

```bash
shark bug note add B001 --type=comment --content="Reproduced on Safari 17.2 and Chrome 120"
shark bug note add B001 --type=decision --content="Root cause: race condition in async handler"
shark bug note add B001 --type=blocker --content="Waiting for auth service to expose session endpoint"
```

---

## `shark bug notes`

List all notes for a bug.

```bash
shark bug notes <key> [flags]
```

**Arguments:**
- `key` — Bug key (e.g., `B001`)

**Flags:**
| Flag | Description |
|------|-------------|
| `--json` | JSON output |

**Examples:**

```bash
shark bug notes B001
shark bug notes B001 --json
```

---

## `shark bug tag add`

Retroactively attach a registered tag to an existing bug. Re-running with the same arguments is a no-op (idempotent).

```bash
shark bug tag add <key> <name> [--json]
```

**Arguments:**
- `key` — Bug key (e.g., `B001`)
- `name` — Tag name (must already be registered in the vocabulary)

**Examples:**

```bash
# Attach 'auth' to bug B001
shark bug tag add B001 auth

# Attach is idempotent (no duplicate row)
shark bug tag add B001 auth   # exit 0, no-op

# JSON output
shark bug tag add B001 auth --json
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag attached (or already attached) |
| 1 | `not_found` | Bug key not found |
| 2 | `db_error` | Database error |
| 3 | `unregistered_tag` | Tag name not in vocabulary |
| 3 | `validation` | Name validation error |

**Unregistered-tag error (process exit **3**, internal class `unregistered_tag`):** stderr shows the service error line, an `Available tags:` header, the current vocabulary (two-space-indented), the `To add it: shark tags add <name>` remediation line, and a trailing `Error: exit code 3: ...` line. See [Tags → Unregistered Tag Errors](tags.md#unregistered-tag-errors) for worked output.

---

## `shark bug tag rm`

Retroactively detach a registered tag from a bug. Removing a tag that is not currently attached is a no-op (exit 0) **as long as the tag name is in the vocabulary**.

```bash
shark bug tag rm <key> <name> [--json]
```

**Arguments:**
- `key` — Bug key (e.g., `B001`)
- `name` — Tag name (must be in the vocabulary)

**Examples:**

```bash
# Detach 'auth' from B001
shark bug tag rm B001 auth

# Re-running is idempotent
shark bug tag rm B001 auth   # exit 0, no-op

# Vocabulary-miss (tag name does not exist): process exit 1 with snippet
shark bug tag rm B001 does-not-exist
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag detached (or was not attached) |
| 1 | `not_found` | Bug key not found, or tag name not in vocabulary |
| 2 | `db_error` | Database error |
| 3 | `validation` | Name validation error |

---

## `shark bug context set`

Set a context field on a bug.

```bash
shark bug context set <key> --field=<FIELD> --value=<VALUE> [flags]
```

**Examples:**

```bash
shark bug context set B001 --field=environment --value=production
shark bug context set B001 --field=browser --value="Safari 17.2"
shark bug context set B001 --field=reproduction_steps --value="1. Navigate to /login 2. Submit form"
```

---

## `shark bug context get`

Get all context fields for a bug.

```bash
shark bug context get <key> [--json]
```

---

## `shark bug context clear`

Clear one or all context fields.

```bash
shark bug context clear <key> [--field=<FIELD>] [--json]
```

---

## Workflow Commands (Generic)

Use `shark status` commands to manage bug lifecycle:

```bash
# Advance to next status
shark status advance B001

# Set status directly
shark status set B001 in_fix
shark status set B001 wont_fix

# View status history
shark status history B001

# View valid next statuses
shark status options B001
```

---

## Bug Workflow

Bugs have a dedicated workflow distinct from task workflows.

### Status Flow

```
reported → triaged → in_fix → in_verification → resolved
                  ↘ wont_fix (terminal)
         ↘ duplicate (terminal)
```

### Status Reference

| Status | Phase | Color | Responsibility | Description |
|--------|-------|-------|----------------|-------------|
| `reported` | planning | red | business-analyst | Bug reported, awaiting triage |
| `triaged` | development | yellow | developer | Triaged, severity confirmed, ready for fix |
| `in_fix` | development | blue | developer | Fix actively in progress |
| `in_verification` | review | cyan | qa | Fix applied, awaiting QA verification |
| `resolved` | done | green | — | Bug verified as fixed |
| `wont_fix` | done | gray | — | Acknowledged but will not be fixed |
| `duplicate` | done | gray | — | Duplicate of another bug |

### Orchestrator Routing

| Status | Action | Agent |
|--------|--------|-------|
| `reported` | `spawn_agent` | business-analyst |
| `triaged` | `spawn_agent` | developer |
| `in_fix` | `check_or_resume` | developer |
| `in_verification` | `check_or_resume` | qa |
| `resolved` | `archive` | — |
| `wont_fix` | `archive` | — |
| `duplicate` | `archive` | — |

---

## Severity Reference

| Severity | Description |
|----------|-------------|
| `critical` | System down, data loss, security breach — immediate action required |
| `high` | Major functionality broken, no workaround — fix in current sprint |
| `medium` | Functionality impaired, workaround exists — fix soon |
| `low` | Minor issue, cosmetic, or edge case — fix when convenient |

---

## Linking Bugs to Entities

Bugs can optionally link to an existing entity for context. The link does not create a dependency — it is informational.

```bash
# Link to an epic
shark bug create "Auth regression" --link=E07

# Link to a feature
shark bug create "Login fails on mobile" --link=E07-F01

# Link to a specific task
shark bug create "Race condition" --link=E07-F01-003
```

To list bugs by link:
```bash
shark bug list --link=E07-F01
```

---

## Related Documentation

- [Change Commands](change-commands.md) - Change-card management
- [Status Commands](status-commands.md) - Workflow management
- [Key Formats](key-formats.md) - Bug key format (B###)
- [Workflow Configuration](workflow-configuration.md) - Custom bug workflow
- [Configuration](configuration.md) - `bug_workflow` config reference
