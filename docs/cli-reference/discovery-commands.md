# Discovery Commands

Search and discovery commands for finding tasks, notes, and related documents across your project.

## Overview

Shark provides three discovery mechanisms:

- **`shark search`** - Find tasks by file metadata (files changed during completion)
- **`shark notes search`** - Search note content across all entities (epics, features, tasks)
- **`shark related-docs`** - Manage document links attached to epics, features, or tasks

These commands are useful for tracing which tasks touched specific files, finding past decisions or solutions recorded in notes, and maintaining links between entities and their supporting documentation.

---

## `shark search`

Search for tasks using completion metadata like files changed. Supports partial filename matching. Results are ordered by completion date (most recent first).

**Usage:**

```
shark search [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--file <name>` | File name or path to search for (required) |
| `--epic <key>` | Filter by epic key |
| `--feature <key>` | Filter by feature key |
| `--status <status>` | Filter by task status |
| `--json` | Output in JSON format |

**Examples:**

```bash
# Find tasks that touched a specific file
shark search --file="useTheme.ts"

# Search within a specific epic
shark search --file="task_repository" --epic E10

# Search within a specific feature with JSON output
shark search --file="models/task.go" --json
```

---

## `shark notes search`

Search for notes containing the specified query across all entities (epics, features, tasks). The search is case-insensitive and supports filtering by entity type, epic, feature, note type, and time period.

**Usage:**

```
shark notes search <query> [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--entity-type <type>` | | Filter by entity type (`epic`, `feature`, or `task`) |
| `--epic <key>` | `-e` | Filter by epic key (e.g., `E10`) |
| `--feature <key>` | `-f` | Filter by feature key (e.g., `E10-F01`) |
| `--type <types>` | `-t` | Filter by note type (comma-separated for multiple) |
| `--since <date>` | | Filter notes created after date (YYYY-MM-DD format) |
| `--until <date>` | | Filter notes created before date (YYYY-MM-DD format) |
| `--json` | | Output in JSON format |

**Note types** include: `decision`, `solution`, `rejection`, `implementation`, `question`, and others depending on your workflow.

**Examples:**

```bash
# Search all notes for a keyword
shark notes search "singleton pattern"

# Search only epic notes
shark notes search "dark mode" --entity-type epic

# Search decision notes within a specific epic
shark notes search "bug" --type decision,solution --epic E10

# Search with a date range and JSON output
shark notes search "performance" --since 2026-01-01 --until 2026-01-31 --json
```

---

## `shark related-docs`

Manage related documents linked to epics, features, or tasks. Related documents provide traceable links between entities and their supporting files (design docs, specifications, implementation guides, etc.).

**Usage:**

```
shark related-docs [command]
```

**Available Commands:**

| Command | Description |
|---------|-------------|
| `add` | Add a related document |
| `list` | List related documents |
| `delete` | Delete a related document link |

---

### `shark related-docs add`

Add a related document to an epic, feature, or task. The document is created or retrieved if it already exists with the same title and path. The document is then linked to exactly one parent entity.

**Usage:**

```
shark related-docs add <title> <path> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--epic <key>` | Epic key (e.g., `E01`) |
| `--feature <key>` | Feature key (e.g., `E01-F01`) |
| `--task <key>` | Task key (e.g., `T-E01-F01-001`) |

Exactly one of `--epic`, `--feature`, or `--task` is required.

**Examples:**

```bash
# Link a design doc to an epic
shark related-docs add "OAuth Specification" docs/oauth.md --epic=E01

# Link implementation notes to a feature
shark related-docs add "Implementation Notes" docs/notes.md --feature=E01-F01

# Link detailed notes to a specific task
shark related-docs add "Task Details" docs/details.md --task=T-E01-F01-001
```

---

### `shark related-docs list`

List all documents linked to an epic, feature, or task. Requires exactly one of `--epic`, `--feature`, or `--task` flags.

**Usage:**

```
shark related-docs list [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--epic <key>` | Epic key (e.g., `E01`) |
| `--feature <key>` | Feature key (e.g., `E01-F01`) |
| `--task <key>` | Task key (e.g., `T-E01-F01-001`) |
| `--json` | Output in JSON format |

**Examples:**

```bash
# List docs linked to an epic
shark related-docs list --epic=E01

# List docs linked to a feature in JSON format
shark related-docs list --feature=E01-F01 --json

# List docs linked to a task
shark related-docs list --task=T-E01-F01-001
```

---

### `shark related-docs delete`

Remove a document link from an epic, feature, or task. The document itself is not deleted from the database, only the link is removed. Delete is idempotent -- it succeeds even if the document is not linked to the parent.

**Usage:**

```
shark related-docs delete <title> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--epic <key>` | Epic key (e.g., `E01`) |
| `--feature <key>` | Feature key (e.g., `E01-F01`) |
| `--task <key>` | Task key (e.g., `T-E01-F01-001`) |

Exactly one of `--epic`, `--feature`, or `--task` is required.

**Examples:**

```bash
# Remove a doc link from an epic
shark related-docs delete "OAuth Specification" --epic=E01

# Remove a doc link from a feature
shark related-docs delete "Implementation Notes" --feature=E01-F01

# Remove a doc link from a task
shark related-docs delete "Task Details" --task=T-E01-F01-001
```

---

## Related Documentation

- [Task Commands](task-commands.md) - Task lifecycle and management
- [Epic Commands](epic-commands.md) - Epic management
- [Feature Commands](feature-commands.md) - Feature management
- [JSON Output](json-output.md) - JSON response structures
- [Key Formats](key-formats.md) - Case insensitive keys and slug support
