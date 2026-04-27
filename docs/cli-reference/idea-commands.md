# Idea Commands

Lightweight idea capture and management for tracking raw ideas before they become structured epics, features, or tasks.

Ideas provide a low-friction way to record thoughts, feature requests, and improvement suggestions. When an idea matures, it can be converted into a structured entity (epic, feature, or task) using the `shark idea convert` command.

## Key Format

Ideas use auto-generated keys in the format `I-YYYY-MM-DD-xx`, where:

- `YYYY-MM-DD` is the creation date
- `xx` is a two-digit sequence number for that day

Examples: `I-2026-02-25-01`, `I-2026-02-25-02`, `I-2026-01-15-01`

## Idea Statuses

| Status | Description |
|--------|-------------|
| `new` | Freshly captured idea (default) |
| `on_hold` | Parked for later consideration |
| `converted` | Promoted to epic, feature, or task |
| `archived` | Soft-deleted / no longer relevant |

---

## `shark idea create`

Create a new idea with an auto-generated key.

**Usage:**

```
shark idea create <title> [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--description` | string | Idea description |
| `--notes` | string | Additional notes |
| `--priority` | int | Priority (1-10) |
| `--status` | string | Initial status: `new`, `on_hold`, `converted`, `archived` (default `new`) |
| `--order` | int | Order for sorting ideas |
| `--depends-on` | strings | Dependent idea keys |
| `--related-docs` | strings | Related document paths |
| `--size` | string | Size estimate: `XS`, `S`, `M`, `L`, `XL`, `XXL` or numeric `1`, `2`, `3`, `5`, `8`, `13`. Optional; absence stores NULL. |
| `--tag` | string | Tag to apply (repeatable). Tag must be registered; see `shark tags list`. |

**Examples:**

```bash
# Create a simple idea
shark idea create "New feature idea"

# Create idea with description and priority
shark idea create "Backend optimization" --description="Improve query performance" --priority=8

# Create idea on hold with notes
shark idea create "UI redesign" --status=on_hold --notes="Waiting for design review"

# Create idea with size estimate
shark idea create "Add webhook support" --size=M
shark idea create "Full platform rewrite" --size=XXL

# With one or more tags (tags must already be registered)
shark idea create "Magic-link login" --tag=auth --tag=voice
```

**JSON Output:**

```bash
shark idea create "API rate limiting" --priority=7 --size=M --json
```

```json
{
  "key": "I-2026-02-25-01",
  "title": "API rate limiting",
  "status": "new",
  "priority": 7,
  "size": 3,
  "created_at": "2026-02-25T14:30:00Z",
  "updated_at": "2026-02-25T14:30:00Z"
}
```

> **Note:** `size` is omitted from the JSON response when `--size` is not provided. `size_label` is only emitted by `get` / `status` commands; use `shark idea get <key> --field size_label` if you need the label after creation.

---

## `shark idea get`

Display detailed information about a specific idea.

**Usage:**

```
shark idea get <idea-key> [flags]
```

**Flags:**

No command-specific flags. Supports global flags (`--json`, `--field`, etc.).

**Examples:**

```bash
# Get idea details (table format)
shark idea get I-2026-02-25-01

# Get idea details as JSON
shark idea get I-2026-02-25-01 --json

# Extract a single field
shark idea get I-2026-02-25-01 --json --field status
```

**JSON Output:**

```bash
shark idea get I-2026-02-25-01 --json
```

```json
{
  "key": "I-2026-02-25-01",
  "title": "API rate limiting",
  "description": "Add rate limiting to public API endpoints",
  "status": "new",
  "priority": 7,
  "notes": "Consider using token bucket algorithm",
  "order": 0,
  "depends_on": [],
  "related_docs": [],
  "created_at": "2026-02-25T14:30:00Z",
  "updated_at": "2026-02-25T15:00:00Z"
}
```

---

## `shark idea list`

List ideas with optional filtering by status and priority.

By default, archived ideas are hidden unless `--status=archived` is specified.

**Usage:**

```
shark idea list [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--status` | string | Filter by status: `new`, `on_hold`, `converted`, `archived` |
| `--priority` | int | Filter by priority (1-10) |

**Examples:**

```bash
# List all non-archived ideas
shark idea list

# List only new ideas
shark idea list --status=new

# List high-priority ideas
shark idea list --priority=8

# List as JSON
shark idea list --json
```

**JSON Output:**

```bash
shark idea list --json
```

```json
[
  {
    "key": "I-2026-02-25-01",
    "title": "API rate limiting",
    "status": "new",
    "priority": 7,
    "created_at": "2026-02-25T14:30:00Z",
    "updated_at": "2026-02-25T15:00:00Z"
  },
  {
    "key": "I-2026-02-24-01",
    "title": "Dark mode support",
    "status": "on_hold",
    "priority": 4,
    "created_at": "2026-02-24T09:15:00Z",
    "updated_at": "2026-02-24T09:15:00Z"
  }
]
```

---

## `shark idea update`

Update properties of an existing idea. All properties can be updated using flags.

**Usage:**

```
shark idea update <idea-key> [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--title` | string | Update title |
| `--description` | string | Update description |
| `--notes` | string | Update notes |
| `--priority` | int | Update priority (1-10) |
| `--status` | string | Update status |
| `--order` | int | Update order |
| `--depends-on` | strings | Update dependencies |
| `--related-docs` | strings | Update related document paths |
| `--size` | string | New size: `XS`\|`S`\|`M`\|`L`\|`XL`\|`XXL` or `1`\|`2`\|`3`\|`5`\|`8`\|`13`. Use `clear` to remove (set to NULL). Flag absent = no change. |
| `--tag` | string | Tag to apply additively (repeatable). Empty = no change; use `shark idea tag rm` to detach. |

**Examples:**

```bash
# Update title
shark idea update I-2026-02-25-01 --title="Updated title"

# Update priority and status
shark idea update I-2026-02-25-01 --priority=9 --status=on_hold

# Add notes
shark idea update I-2026-02-25-01 --notes="Additional context from stakeholder meeting"

# Set or update size
shark idea update I-2026-02-25-01 --size=L
shark idea update I-2026-02-25-01 --size=5

# Clear size (set back to NULL)
shark idea update I-2026-02-25-01 --size=clear

# Additive tagging
shark idea update I-2026-02-25-01 --tag=auth
```

> `--tag` on update is **additive only**. Use `shark idea tag rm I-... <name>` to detach a single tag.

**JSON Output:**

```bash
shark idea update I-2026-02-25-01 --priority=9 --json
```

```json
{
  "key": "I-2026-02-25-01",
  "title": "API rate limiting",
  "status": "new",
  "priority": 9,
  "notes": "",
  "updated_at": "2026-02-25T16:00:00Z"
}
```

---

## `shark idea delete`

Delete an idea. By default performs a soft delete (archives the idea). Use `--hard` for permanent deletion.

Confirmation is required unless `--force` is provided.

**Usage:**

```
shark idea delete <idea-key> [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--hard` | bool | Perform hard delete (permanent removal) |
| `--force` | bool | Skip confirmation prompt |

**Examples:**

```bash
# Soft delete (archives the idea)
shark idea delete I-2026-02-25-01

# Hard delete (permanent, with confirmation prompt)
shark idea delete I-2026-02-25-01 --hard

# Hard delete without confirmation
shark idea delete I-2026-02-25-01 --hard --force
```

---

## `shark idea convert`

Convert a lightweight idea into a structured entity (epic, feature, or task). Once converted, the idea status automatically changes to `converted` and a new entity is created with the idea's title and description.

**Usage:**

```
shark idea convert [command]
```

**Available Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `epic` | Convert idea to a new epic |
| `feature` | Convert idea to a feature (requires `--epic`) |
| `task` | Convert idea to a task (requires `--epic` and `--feature`) |

### `shark idea convert epic`

Convert an idea into a new epic.

```bash
# Convert idea to epic
shark idea convert epic I-2026-02-25-01
```

### `shark idea convert feature`

Convert an idea into a feature within an existing epic.

```bash
# Convert idea to feature in epic E10
shark idea convert feature I-2026-02-25-01 --epic=E10
```

### `shark idea convert task`

Convert an idea into a task within an existing epic and feature.

```bash
# Convert idea to task in E10-F02
shark idea convert task I-2026-02-25-01 --epic=E10 --feature=E10-F02
```

**JSON Output:**

```bash
shark idea convert epic I-2026-02-25-01 --json
```

```json
{
  "idea_key": "I-2026-02-25-01",
  "converted_to": "epic",
  "entity_key": "E18",
  "title": "API rate limiting",
  "status": "converted"
}
```

---

## `shark idea tag add`

Retroactively attach a registered tag to an existing idea (AC-26). Re-running with the same arguments is a no-op (idempotent).

**Usage:**
```bash
shark idea tag add <idea-key> <name> [--json]
```

**Examples:**
```bash
shark idea tag add I-2026-02-25-01 voice
shark idea tag add I-2026-02-25-01 voice --json   # idempotent
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag attached (or already attached) |
| 1 | `not_found` | Idea key not found |
| 2 | `db_error` | Database error |
| 3 | `unregistered_tag` | Tag name not in vocabulary |
| 3 | `validation` | Name validation error |

On an unregistered tag (process exit **3**, internal class `unregistered_tag`), stderr shows the SC-2 vocabulary snippet and the `To add it: shark tags add <name>` remediation line. See [Tags → Unregistered Tag Errors](tags.md#unregistered-tag-errors).

---

## `shark idea tag rm`

Retroactively detach a registered tag from an idea.

**Usage:**
```bash
shark idea tag rm <idea-key> <name> [--json]
```

**Examples:**
```bash
shark idea tag rm I-2026-02-25-01 voice
shark idea tag rm I-2026-02-25-01 voice   # idempotent no-op
```

**Exit codes:**
| Process exit | Internal class | Condition |
|------|----------------|-----------|
| 0 | — | Tag detached (or was not attached) |
| 1 | `not_found` | Idea key not found, or tag name not in vocabulary |
| 2 | `db_error` | Database error |
| 3 | `validation` | Name validation error |

---

## Typical Workflow

1. **Capture** ideas as they come up:
   ```bash
   shark idea create "Add webhook support" --priority=6
   shark idea create "Improve error messages" --notes="Users confused by error codes"
   ```

2. **Review** and prioritize ideas:
   ```bash
   shark idea list
   shark idea update I-2026-02-25-01 --priority=9
   ```

3. **Convert** mature ideas to structured entities:
   ```bash
   shark idea convert epic I-2026-02-25-01
   ```

4. **Clean up** ideas no longer relevant:
   ```bash
   shark idea delete I-2026-02-24-03
   ```

## Related Documentation

- [Epic Commands](epic-commands.md) - Managing epics
- [Feature Commands](feature-commands.md) - Managing features
- [Task Commands](task-commands.md) - Managing tasks
- [Key Formats](key-formats.md) - Entity key format reference
- [JSON Output](json-output.md) - JSON response structures
