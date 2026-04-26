# Change Commands

Complete reference for `shark change` subcommands. Change-cards are standalone entities for tracking lightweight changes and enhancements with their own workflow and key format.

---

## Key Format

Change-cards use the `CC-###` key format (e.g., `CC-001`, `CC-042`). Keys are auto-generated sequentially on creation.

---

## `shark change create`

Create a new change-card.

```bash
shark change create <title> [flags]
```

**Arguments:**
- `title` — Change-card title (required)

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--description` | string | — | Detailed description |
| `--link` | string | — | Link to epic or feature key |
| `--requested-by` | string | — | Name or ID of the requester |
| `--priority` | int | `5` | Priority level (1–10, where 10 = highest) |
| `--justification` | string | — | Business justification for the change |
| `--tag` | string | — | Tag to apply (repeatable). Tag must be registered; see `shark tags list`. |
| `--json` | bool | `false` | JSON output |

**Examples:**

```bash
# Basic change-card
shark change create "Add dark mode toggle"

# With metadata
shark change create "Add dark mode toggle" \
  --link=E07 \
  --priority=8 \
  --requested-by=alice \
  --description="Users have requested dark mode for accessibility reasons"

# Linked to a feature
shark change create "Increase session timeout to 8 hours" \
  --link=E07-F01 \
  --priority=6 \
  --requested-by=security-team

# With one or more tags (tags must already be registered)
shark change create "Increase session timeout" --tag=auth
shark change create "Redesign settings panel" --tag=frontend --tag=ux
```

**Output (default):**
```
Created change-card CC-001: Add dark mode toggle
  File: docs/plan/changes/CC-001.md
  Priority: 8
  Status: proposed
```

**Output (`--json`):**
```json
{
  "key": "CC-001",
  "title": "Add dark mode toggle",
  "status": "proposed",
  "priority": 8,
  "requested_by": "alice",
  "epic_id": 7,
  "file_path": "docs/plan/changes/CC-001.md",
  "created_at": "2026-03-05T10:00:00Z"
}
```

---

## `shark change get`

Get details of a specific change-card.

```bash
shark change get <key> [flags]
```

**Arguments:**
- `key` — Change-card key (e.g., `CC-001`)

**Flags:**
| Flag | Description |
|------|-------------|
| `--json` | Machine-readable JSON output |
| `--field <name>` | Extract single field (implies `--json`) |

**Examples:**

```bash
shark change get CC-001
shark change get CC-001 --json
shark change get CC-001 --field status
shark change get CC-001 --field priority
```

**Output (default):**
```
Change-Card CC-001: Add dark mode toggle
  Status:       proposed
  Priority:     8
  Requested by: alice
  Linked:       epic E07
  Created:      2026-03-05
```

---

## `shark change list`

List change-cards with optional filtering.

```bash
shark change list [flags]
```

**Flags:**
| Flag | Type | Description |
|------|------|-------------|
| `--status` | string | Filter by status (e.g., `proposed`, `in_progress`) |
| `--link` | string | Filter by linked entity key (epic or feature) |
| `--show-all` | bool | Include terminal statuses (`completed`, `declined`) |
| `--json` | bool | Machine-readable JSON output |

**Examples:**

```bash
# Active change-cards (hides completed/declined by default)
shark change list

# Including terminal statuses
shark change list --show-all

# Filter by status
shark change list --status=proposed
shark change list --status=approved

# Linked to an epic
shark change list --link=E07

# Linked to a feature
shark change list --link=E07-F01
```

---

## `shark change update`

Update change-card fields.

```bash
shark change update <key> [flags]
```

**Arguments:**
- `key` — Change-card key (e.g., `CC-001`)

**Flags:**
| Flag | Type | Description |
|------|------|-------------|
| `--title` | string | New title |
| `--priority` | int | New priority (1–10) |
| `--assigned-to` | string | Assign to a person/agent |
| `--requested-by` | string | Update requester |
| `--justification` | string | Update business justification |
| `--impact-analysis` | string | Update impact analysis |
| `--rollback-plan` | string | Update rollback plan |
| `--tag` | string | Tag to apply additively (repeatable). Empty = no change; use `shark change tag rm` to detach. |
| `--json` | bool | JSON output |

**Examples:**

```bash
shark change update CC-001 --priority=9
shark change update CC-001 --assigned-to=bob
shark change update CC-001 --title="Add dark mode with user preference persistence"
shark change update CC-001 --impact-analysis="Affects UI layer only, no DB changes required"
shark change update CC-001 --rollback-plan="Revert feature flag, redeploy previous build"

# Additive tagging
shark change update CC-001 --tag=auth
```

> `--tag` on update is **additive only**. Use `shark change tag rm CC-001 <name>` to detach a single tag.

---

## `shark change delete`

Delete a change-card.

```bash
shark change delete <key> [flags]
```

**Arguments:**
- `key` — Change-card key (e.g., `CC-001`)

**Flags:**
| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

**Examples:**

```bash
shark change delete CC-001
shark change delete CC-001 --force
```

---

## `shark change approve`

Approve a change-card, transitioning it from `proposed` → `approved`.

```bash
shark change approve <key> [flags]
```

**Arguments:**
- `key` — Change-card key (e.g., `CC-001`)

**Flags:**
| Flag | Description |
|------|-------------|
| `--json` | JSON output |

**Examples:**

```bash
shark change approve CC-001
shark change approve CC-001 --json
```

**Note:** Use `shark status set CC-001 declined` to decline a change-card instead.

---

## `shark change note add`

Add a note to a change-card.

```bash
shark change note add <key> --type=<TYPE> [--content=<CONTENT>] [flags]
```

**Arguments:**
- `key` — Change-card key (e.g., `CC-001`)

**Flags:**
| Flag | Type | Description |
|------|------|-------------|
| `--type` | string | Note type: `comment`, `decision`, `blocker`, `context` |
| `--content` | string | Note content |
| `--json` | bool | JSON output |

**Examples:**

```bash
shark change note add CC-001 --type=comment --content="Stakeholders approved in design review"
shark change note add CC-001 --type=decision --content="Implemented as feature flag for gradual rollout"
shark change note add CC-001 --type=blocker --content="Waiting for design mockups from UX team"
```

---

## `shark change notes`

List all notes for a change-card.

```bash
shark change notes <key> [--json]
```

**Examples:**

```bash
shark change notes CC-001
shark change notes CC-001 --json
```

---

## `shark change tag add`

Retroactively attach a registered tag to an existing change-card. Re-running with the same arguments is a no-op (idempotent).

```bash
shark change tag add <key> <name> [--json]
```

**Arguments:**
- `key` — Change-card key (e.g., `CC-001`)
- `name` — Tag name (must already be registered in the vocabulary)

**Examples:**

```bash
shark change tag add CC-001 auth
shark change tag add CC-001 auth --json   # idempotent
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag attached (or already attached) |
| 1 | `not_found` | Change-card key not found |
| 2 | `db_error` | Database error |
| 3 | `unregistered_tag` | Tag name not in vocabulary |
| 3 | `validation` | Name validation error |

On an unregistered tag (process exit **3**, internal class `unregistered_tag`), stderr shows the SC-2 vocabulary snippet and the `To add it: shark tags add <name>` remediation line. See [Tags → Unregistered Tag Errors](tags.md#unregistered-tag-errors).

---

## `shark change tag rm`

Retroactively detach a registered tag from a change-card.

```bash
shark change tag rm <key> <name> [--json]
```

**Examples:**

```bash
shark change tag rm CC-001 auth
shark change tag rm CC-001 auth   # idempotent no-op
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag detached (or was not attached) |
| 1 | `not_found` | Change-card key not found, or tag name not in vocabulary |
| 2 | `db_error` | Database error |
| 3 | `validation` | Name validation error |

---

## `shark change context set`

Set a context field on a change-card.

```bash
shark change context set <key> --field=<FIELD> --value=<VALUE> [flags]
```

**Examples:**

```bash
shark change context set CC-001 --field=deployment_date --value=2026-03-15
shark change context set CC-001 --field=ticket_id --value=JIRA-4521
shark change context set CC-001 --field=estimated_effort --value="2 days"
```

---

## `shark change context get`

Get all context fields for a change-card.

```bash
shark change context get <key> [--json]
```

---

## `shark change context clear`

Clear one or all context fields.

```bash
shark change context clear <key> [--field=<FIELD>] [--json]
```

---

## Workflow Commands (Generic)

Use `shark status` commands to manage change-card lifecycle:

```bash
# Advance to next status
shark status advance CC-001

# Set status directly
shark status set CC-001 approved
shark status set CC-001 declined

# View status history
shark status history CC-001

# View valid next statuses
shark status options CC-001
```

---

## Change-Card Workflow

Change-cards have a dedicated workflow for lightweight change management.

### Status Flow

```
proposed → approved → in_progress → completed
         ↘ declined (terminal)
```

### Status Reference

| Status | Phase | Color | Responsibility | Description |
|--------|-------|-------|----------------|-------------|
| `proposed` | planning | yellow | business-analyst | Awaiting scope assessment and approval |
| `approved` | development | cyan | developer | Approved, ready to implement |
| `in_progress` | development | blue | developer | Implementation in progress |
| `completed` | done | green | — | Change implemented and verified |
| `declined` | done | red | — | Change request declined |

### Orchestrator Routing

| Status | Action | Agent |
|--------|--------|-------|
| `proposed` | `spawn_agent` | business-analyst |
| `approved` | `spawn_agent` | developer |
| `in_progress` | `check_or_resume` | developer |
| `completed` | `archive` | — |
| `declined` | `archive` | — |

---

## Differences from Tasks

| Aspect | Tasks | Change-Cards |
|--------|-------|--------------|
| Key format | `E07-F01-001` | `CC-001` |
| Hierarchy | Nested under epic → feature | Standalone (optional link) |
| Workflow | Advanced TDD (19 statuses) or Basic (5) | Fixed 5-status change workflow |
| Severity | — | — |
| Priority | — | 1–10 |
| Approval mechanism | Product owner in workflow | `shark change approve` command |
| Target entity | Feature delivery | Ad-hoc changes to any entity |

---

## Linking Change-Cards to Entities

Change-cards can optionally link to an existing epic or feature for context.

```bash
# Link to an epic
shark change create "Add rate limiting" --link=E07

# Link to a feature
shark change create "Increase session timeout" --link=E07-F01
```

To list change-cards by link:
```bash
shark change list --link=E07
shark change list --link=E07-F01
```

---

## Related Documentation

- [Bug Commands](bug-commands.md) - Bug management
- [Status Commands](status-commands.md) - Workflow management
- [Key Formats](key-formats.md) - Change-card key format (CC-###)
- [Workflow Configuration](workflow-configuration.md) - Custom change workflow
- [Configuration](configuration.md) - `change_workflow` config reference
