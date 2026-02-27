# Context Commands

Commands for managing structured resume context data on epics, features, and tasks.

## Overview

Entity context data provides structured fields for tracking work progress, decisions, blockers, and open questions on any entity. This is particularly useful for AI agents resuming work across sessions, as context data persists in the database and can be retrieved with a single command.

Context data supports the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `current_step` | String | Describes the current work step |
| `completed_steps` | JSON array | List of completed step strings |
| `remaining_steps` | JSON array | List of remaining step strings |
| `implementation_decisions` | JSON object | Key-value pairs of decisions made |
| `open_questions` | JSON array | List of question strings |
| `blockers` | JSON array | List of blocker objects |
| `acceptance_criteria_status` | JSON array | List of criterion objects |

## Auto-Detection from Key Format

The `shark context` command automatically detects the entity type from the key format. There is no need to specify whether the target is an epic, feature, or task:

| Key Format | Entity Type | Example |
|------------|-------------|---------|
| `E##` | Epic | `E07`, `e07` |
| `E##-F##` | Feature | `E07-F01`, `e07-f01` |
| `E##-F##-###` | Task | `E07-F01-001`, `e07-f01-001` |

All keys are case insensitive. Slugged key formats also work (e.g., `E07-F01-001-implement-auth`).

---

## `shark context`

Get or manage structured resume context data for any entity.

When called with just a key (no subcommand), displays the context data. This is equivalent to `shark context get <key>`.

**Usage:**
```bash
shark context <key> [flags]
shark context [command]
```

**Subcommands:**
- `get` - Get context data (default when no subcommand)
- `set` - Set or update a context field
- `clear` - Clear all context data

**Flags:**
- `-h, --help` - Help for context

**Examples:**

```bash
# Get context (implicit get)
shark context E07                  # Epic context
shark context E07-F01              # Feature context
shark context E07-F01-001          # Task context

# With JSON output
shark context E07-F01-001 --json
```

---

## `shark context get`

Display the current context data for an entity. Entity type is auto-detected from key format.

**Usage:**
```bash
shark context get <key> [flags]
```

**Flags:**
- `-h, --help` - Help for get

**Examples:**

```bash
# Get context for different entity types
shark context get E07              # Epic context
shark context get E07-F01          # Feature context
shark context get E07-F01-001      # Task context

# JSON output
shark context get E07-F01-001 --json
```

**JSON Output Example:**

```json
{
  "entity_type": "task",
  "entity_key": "E07-F01-001",
  "context": {
    "current_step": "Implementing API endpoint",
    "completed_steps": [
      "Set up test framework",
      "Wrote unit tests for auth"
    ],
    "remaining_steps": [
      "Implement token refresh",
      "Add integration tests"
    ],
    "implementation_decisions": {
      "auth_method": "JWT with refresh tokens",
      "password_hashing": "bcrypt with cost 12"
    },
    "open_questions": [
      "Should we support OAuth providers?"
    ],
    "blockers": [],
    "acceptance_criteria_status": []
  }
}
```

---

## `shark context set`

Set or update a specific field in entity context data. Entity type is auto-detected from key format.

**Usage:**
```bash
shark context set <key> [flags]
```

**Flags:**
- `--field string` - Context field to update (required)
- `--value string` - Field value (required)
- `-h, --help` - Help for set

**Supported Fields:**

| Field | Value Format | Example |
|-------|-------------|---------|
| `current_step` | Plain string | `"Implementing API endpoint"` |
| `completed_steps` | JSON array of strings | `'["Step 1","Step 2"]'` |
| `remaining_steps` | JSON array of strings | `'["Step 3","Step 4"]'` |
| `implementation_decisions` | JSON object | `'{"key":"value"}'` |
| `open_questions` | JSON array of strings | `'["Question 1?"]'` |
| `blockers` | JSON array of objects | `'[{"description":"Blocked on API"}]'` |
| `acceptance_criteria_status` | JSON array of objects | `'[{"criterion":"Login works","met":true}]'` |

**Examples:**

```bash
# Set current step (plain string)
shark context set E07-F01-001 --field current_step --value "Implementing API endpoint"

# Set completed steps (JSON array)
shark context set E07-F01 --field completed_steps --value '["Step 1","Step 2"]'

# Set open questions (JSON array)
shark context set E07 --field open_questions --value '["What API version?"]'

# Set implementation decisions (JSON object)
shark context set E07-F01-001 --field implementation_decisions --value '{"auth":"JWT","hash":"bcrypt"}'
```

---

## `shark context clear`

Remove all context data from an entity. Entity type is auto-detected from key format.

**Usage:**
```bash
shark context clear <key> [flags]
```

**Flags:**
- `-h, --help` - Help for clear

**Examples:**

```bash
# Clear context for different entity types
shark context clear E07-F01-001    # Clear task context
shark context clear E07-F01        # Clear feature context
shark context clear E07            # Clear epic context
```

---

## Entity-Specific Context Commands

Context management is also available under each entity's command group. These are equivalent to the smart dispatcher:

```bash
# These pairs are equivalent:
shark context E07              # Smart dispatcher
shark epic context get E07     # Entity-specific

shark context E07-F01          # Smart dispatcher
shark feature context get E07-F01  # Entity-specific

shark context E07-F01-001      # Smart dispatcher
shark task context get E07-F01-001  # Entity-specific
```

The smart dispatcher (`shark context`) is recommended for most use cases.

## Global Flags

All context commands support the standard global flags:

- `--json` - Output in JSON format (machine-readable)
- `--field string` - Extract a single field from JSON output
- `--config string` - Config file path (default: `.sharkconfig.json`)
- `--db string` - Database file path (default: `shark-tasks.db`)
- `--no-color` - Disable colored output
- `-v, --verbose` - Enable verbose/debug output

## Related Documentation

- [Task Commands](task-commands.md) - Task lifecycle management
- [Feature Commands](feature-commands.md) - Feature management
- [Epic Commands](epic-commands.md) - Epic management
- [Key Formats](key-formats.md) - Key format reference
- [JSON Output](json-output.md) - JSON response structures
