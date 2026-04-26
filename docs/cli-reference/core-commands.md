# Core Commands (Smart Dispatchers)

Shark provides a set of core commands that automatically detect the entity type (epic, feature, or task) based on the key format you provide. These smart dispatchers eliminate the need to specify `epic`, `feature`, or `task` subcommands for common operations.

## Key Format Detection

All core commands determine the entity type from the key format:

| Key Format | Entity Type | Examples |
|------------|-------------|---------|
| `E##` | Epic | `E07`, `E10`, `e04` |
| `E##-F##` or `F##` | Feature | `E07-F01`, `F01`, `e10-f05` |
| `E##-F##-###` or `T-E##-F##-###` | Task | `E07-F01-001`, `T-E10-F05-003`, `e04-f01-1` |

All keys are **case insensitive**. `E07`, `e07`, `E07-F01`, `e07-f01` all work identically.

Slugged keys are also supported: `E07-user-management`, `E07-F01-authentication`, `E07-F01-001-implement-login`.

---

## get

Retrieve details for any entity. Dispatches to the appropriate handler based on the key format.

### Usage

```
shark get <KEY> [flags]
```

The command also supports space-separated positional arguments for specifying entities:

```
shark get EPIC                    # Get epic details
shark get EPIC FEATURE            # Get feature details
shark get EPIC FEATURE TASKNUM    # Get task details
shark get FULL_TASK_KEY           # Get task details (full key)
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (machine-readable) |
| `--field <name>` | Extract a single field from JSON output (e.g., `--field status`) |
| `-h, --help` | Help for get |

### Examples

```bash
# Get epic details
shark get E10

# Get feature details (two equivalent formats)
shark get E10 F01
shark get E10-F01

# Get task details (multiple equivalent formats)
shark get E10 F01 001
shark get E10 F01 1                # Short task number form
shark get T-E10-F01-001            # Full task key
shark get E10-F01-001              # Short task key

# JSON output
shark get E10 --json

# Extract a single field
shark get E10-F01-001 --json --field status
```

---

## list

List entities at any level. With no arguments, lists epics. With an epic key, lists features. With an epic and feature key, lists tasks.

### Usage

```
shark list [EPIC] [FEATURE] [flags]
```

Positional arguments determine what is listed:

| Arguments | Behavior |
|-----------|----------|
| *(none)* | List all epics |
| `EPIC` | List features in that epic |
| `EPIC FEATURE` | List tasks in that feature |

The combined key format also works: `shark list E10-F01` lists tasks in feature E10-F01.

### Flags

| Flag | Description |
|------|-------------|
| `--all` | Show all items including completed (completed items are hidden by default) |
| `--status <status>` | Filter by status |
| `--sort-by <field>` | Sort by: `key`, `progress`, `status` (default: `key`) |
| `--tag <name>` | Filter by tag (repeatable; AND — all tags must match). Tag must be registered; see [`tags.md`](tags.md) and `shark tags list`. |
| `--json` | Output in JSON format (machine-readable) |
| `--field <name>` | Extract a single field from JSON output |
| `-h, --help` | Help for list |

### Examples

```bash
# List all epics
shark list

# List features in epic E10
shark list E10

# List tasks in feature E10-F01 (two equivalent formats)
shark list E10 F01
shark list E10-F01

# List with filters
shark list E10 F01 --status=in_progress
shark list --all                   # Include completed items

# Sort features by progress
shark list E10 --sort-by=progress

# Filter by a registered tag (AC-21 / AC-22 / AC-23)
shark list --tag=voice                  # Epics tagged 'voice'
shark list E10 --tag=voice              # Features in E10 tagged 'voice'
shark list E10 F01 --tag=voice          # Tasks in E10-F01 tagged 'voice'

# Multi-tag filtering uses AND semantics — only entities carrying ALL tags match
shark list E10 --tag=auth --tag=voice

# JSON output
shark list --json
shark list E10 F01 --json
```

> `--tag` is **repeatable** and uses **AND semantics** across multiple values. Tag names must already be registered in the vocabulary via `shark tags add`; supplying an unregistered name exits 3. See [Tags Commands](tags.md) for the full vocabulary model and entity-level filtering reference.

---

## create

Create a new entity. Dispatches to the appropriate create handler based on the entity type subcommand.

### Usage

```
shark create epic <title> [flags]
shark create feature [EPIC] <title> [flags]
shark create task [EPIC] [FEATURE] <title> [flags]
```

### Subcommands

#### create epic

Create a new epic.

```
shark create epic <title> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--key <key>` | Custom key for the epic (e.g., `E00`, `bugs`). Auto-generates next `E##` if not provided |
| `--description <text>` | Epic description |
| `--priority <level>` | Priority: `low`, `medium`, `high` (default: `medium`) |
| `--status <status>` | Status: `draft`, `active`, `completed`, `archived` (default: `draft`) |
| `--business-value <level>` | Business value: `low`, `medium`, `high` |
| `--file <path>` | Custom file path (e.g., `docs/custom/epic.md`) |
| `--force` | Force reassignment if file already claimed by another entity |

#### create feature

Create a new feature within an epic.

```
shark create feature [EPIC] <title> [flags]
shark create feature <title> --epic=EPIC [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--epic <key>` | Epic key (can also be the first positional argument) |
| `--key <key>` | Custom key for the feature (e.g., `auth`, `F00`). Auto-generates next `F##` if not provided |
| `--description <text>` | Feature description |
| `--status <status>` | Status: `draft`, `active`, `completed`, `archived` (default: `draft`) |
| `--order <int>` | Execution order (lower runs first) |
| `--file <path>` | Custom file path (e.g., `docs/custom/feature.md`) |
| `--force` | Force reassignment if file already claimed by another entity |

#### create task

Create a new task within a feature.

```
shark create task [EPIC] [FEATURE] <title> [flags]
shark create task [EPIC-FEATURE] <title> [flags]
shark create task <title> --epic=EPIC --feature=FEATURE [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--epic <key>` | `-e` | Epic key (can also be the first positional argument) |
| `--feature <key>` | `-f` | Feature key (can also be the second positional argument) |
| `--agent <type>` | `-a` | Agent type (e.g., `backend`, `frontend`, `qa`) |
| `--priority <int>` | `-p` | Priority, 1=highest, 10=lowest (default: `5`) |
| `--description <text>` | `-d` | Detailed description |
| `--order <int>` | | Execution order (lower runs first) |
| `--depends-on <keys>` | | Comma-separated dependency task keys |
| `--key <key>` | | Custom key for the task |
| `--file <path>` | | Custom file path (e.g., `docs/custom/task.md`) |
| `--create` | | Create file if it does not exist |
| `--force` | | Force reassignment if file already claimed by another task |

### Examples

```bash
# Create an epic
shark create epic "Q1 2025 Roadmap"
shark create epic "Bug Fixes" --key=bugs --priority=high

# Create a feature (positional and flag syntax)
shark create feature E07 "User Authentication"
shark create feature "User Authentication" --epic=E07
shark create feature E07 "Custom Feature" --file="docs/custom/feature.md"

# Create a task (3-arg, 2-arg, and flag syntax)
shark create task E07 F01 "Implement login" --agent=backend --priority=5
shark create task E07-F01 "Implement login"
shark create task "Implement login" --epic=E07 --feature=F01

# Create a task with dependencies and execution order
shark create task E07 F01 "Add tests" --order=2 --depends-on="E07-F01-001"
```

---

## delete

Delete an entity by key. The entity type is auto-detected from the key format.

### Usage

```
shark delete <KEY> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--force` | Force deletion with cascade delete of children |
| `--json` | Output in JSON format (machine-readable) |
| `--field <name>` | Extract a single field from JSON output |
| `-h, --help` | Help for delete |

### Examples

```bash
# Delete an epic
shark delete E07

# Delete a feature
shark delete E07-F01

# Delete a task
shark delete E07-F01-001

# Force delete an epic (cascades to features and tasks)
shark delete E07 --force

# Force delete a feature (cascades to tasks)
shark delete E07-F01 --force
```

### Cascade Behavior

- Deleting an **epic** without `--force` will fail if it has features.
- Deleting a **feature** without `--force` will fail if it has tasks.
- Using `--force` cascades the deletion to all child entities.

---

## view

Open an entity's specification file in an external viewer. The viewer is configurable and defaults to `cat`.

### Usage

```
shark view <KEY> [flags]
```

Like `get`, the command supports space-separated positional arguments:

```
shark view EPIC                    # View epic spec
shark view EPIC FEATURE            # View feature spec
shark view EPIC FEATURE TASKNUM    # View task spec
shark view FULL_TASK_KEY           # View task spec (full key)
```

### Flags

| Flag | Description |
|------|-------------|
| `-h, --help` | Help for view |

### Viewer Configuration

The viewer can be set in `.sharkconfig.json`:

```json
{
  "viewer": "glow"
}
```

Common viewer options: `glow` (Markdown renderer), `nano`, `less`, `cat` (default).

### Examples

```bash
# View epic spec
shark view E10

# View feature spec (two equivalent formats)
shark view E10 F01
shark view E10-F01

# View task spec (multiple equivalent formats)
shark view E10 F01 001
shark view T-E10-F01-001

# Case insensitive
shark view e10
```

---

## Global Flags

All core commands support the following global flags:

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (machine-readable) |
| `--field <name>` | Extract a single field from JSON output (e.g., `--field status`) |
| `--config <path>` | Config file path (default: `.sharkconfig.json`) |
| `--db <path>` | Database file path (default: `shark-tasks.db`) |
| `--no-color` | Disable colored output |
| `-v, --verbose` | Enable verbose/debug output |

## Related Documentation

- [Key Formats](key-formats.md) - Detailed key format reference and dual key support
- [Global Flags](global-flags.md) - Complete global flags reference
- [JSON Output](json-output.md) - JSON response structures
- [Task Commands](task-commands.md) - Task lifecycle commands (start, complete, approve, block)
- [Epic Commands](epic-commands.md) - Noun-first epic commands
- [Feature Commands](feature-commands.md) - Noun-first feature commands
- [Tags Commands](tags.md) - Closed tag vocabulary and `--tag` filtering on list/search
