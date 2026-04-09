# Setup and Maintenance Commands

Commands for initializing projects, managing database migrations, configuring cloud backends, and inspecting workflow configuration.

## Overview

Shark provides several command groups for project setup and ongoing maintenance:

- **Initialization** (`shark init`) -- Create and update project infrastructure (database, config, templates)
- **Validation** (`shark validate`) -- Check database integrity and detect orphaned records
- **Migration** (`shark migrate`) -- Run one-time database schema upgrades and data transformations
- **Cloud** (`shark cloud`) -- Configure Turso cloud database for multi-machine access
- **Workflow** (`shark workflow`) -- Inspect and validate the status workflow defined in `.sharkconfig.json`

All commands listed here support the standard [global flags](global-flags.md) (`--json`, `--verbose`, `--config`, `--db`, `--no-color`, `--field`).

---

## Initialization

### shark admin init

Initialize Shark CLI infrastructure by creating the database, the
`docs/plan/` and `shark-templates/` folder structure, a default
`.sharkconfig.json`, and copying the embedded `shark-templates/` tree.

The `shark-templates/` tree is **re-synced from the embedded version on
every run**, so template/workflow updates shipped with a new `shark` binary
flow through automatically. Files you have modified locally are NOT
overwritten — they are reported as differing from the shipped version, and
you can re-run with `--force` to accept the upstream version.

This command is idempotent and safe to run multiple times.

```
Usage:
  shark admin init [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--non-interactive` | Skip all prompts (use defaults) |
| `--force` | Overwrite existing config and locally-modified templates |

**Examples:**

```bash
# Initialize with default settings (interactive)
shark admin init

# Initialize without prompts (for automation / CI)
shark admin init --non-interactive

# Pick up template updates after upgrading the shark binary; locally-modified
# files are reported but not touched
shark admin init

# Force overwrite existing config and locally-modified templates with the
# shipped versions
shark admin init --force
```

**Default config:** see [Initialization](initialization.md#default-sharkconfigjson)
for the exact JSON shape written by a fresh init. Workflow definitions live
in `shark-templates/.sharkworkflow-short.json` (default) or
`shark-templates/.sharkworkflow.json` — referenced via the `workflow_config`
field in `.sharkconfig.json` rather than embedded inline.

> **Note:** The basic/advanced workflow "profiles" subsystem
> (`shark init update --workflow=...`, `shark init merge ...`) was removed.
> To switch workflows, edit `workflow_config` in `.sharkconfig.json`. See
> [Switching workflows](initialization.md#switching-workflows).

---

## Validation

### shark validate

Validate database integrity by checking file paths and relationships.

Checks performed:
- Task file paths exist on the filesystem
- Features reference existing epics (relationship integrity)
- Tasks reference existing features (relationship integrity)
- Detects orphaned records (records with missing parents)

```
Usage:
  shark validate [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--json` | Output results as JSON |
| `--verbose` | Show detailed validation information |

**Examples:**

```bash
# Validate database integrity
shark validate

# Output results as JSON (useful for CI pipelines)
shark validate --json

# Show detailed information about each check
shark validate --verbose
```

---

## Migration

### shark migrate

Parent command for database migrations and data transformations. These are typically one-time operations that evolve the database as new features are added.

```
Usage:
  shark migrate [command]
```

**Available Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `backfill-slugs` | Backfill slug columns from existing file paths |

---

### shark migrate backfill-slugs

Extract slugs from existing `file_path` values and populate the `slug` columns for epics, features, and tasks. This enables the dual key format (numeric and human-readable slugged keys).

The migration uses a three-phase approach for maximum coverage:

1. **Phase 1** -- Extract epic/feature slugs from task paths (highest coverage)
2. **Phase 2** -- Extract epic slugs from feature paths (fill gaps)
3. **Phase 3** -- Extract slugs from the entity's own `file_path`

The migration is idempotent -- it can be run multiple times safely. Only records with `NULL` or empty slugs will be updated.

```
Usage:
  shark migrate backfill-slugs [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview changes without applying them |
| `-v, --verbose` | Show detailed output |

**Examples:**

```bash
# Preview changes without applying them
shark migrate backfill-slugs --dry-run

# Apply migration
shark migrate backfill-slugs

# Apply with verbose output
shark migrate backfill-slugs --verbose

# Get JSON output for automation
shark migrate backfill-slugs --json
```

---

## Cloud

### shark cloud

Parent command for configuring and managing Turso cloud database integration.

```
Usage:
  shark cloud [command]
```

**Available Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `init` | Initialize Turso cloud database configuration |
| `status` | Show cloud database configuration status |

---

### shark cloud init

Initialize Turso cloud database by configuring connection URL and authentication. Updates `.sharkconfig.json` to use Turso as the database backend instead of local SQLite.

```
Usage:
  shark cloud init [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--url <url>` | Turso database URL (`libsql://...`) |
| `--auth-token <token>` | Turso auth token (JWT) |
| `--auth-file <path>` | Path to file containing auth token |
| `--non-interactive` | Skip prompts, fail if required info missing |

**Examples:**

```bash
# Interactive setup with prompts
shark cloud init

# Non-interactive with flags
shark cloud init --url="libsql://mydb.turso.io" --auth-token="eyJ..."

# Use token from file (more secure)
shark cloud init --url="libsql://mydb.turso.io" --auth-file="~/.turso/token"

# Use environment variable TURSO_AUTH_TOKEN (token auto-detected)
shark cloud init --url="libsql://mydb.turso.io"
```

**Authentication priority order:**

1. `--auth-token` flag (direct value)
2. `--auth-file` flag (read from file)
3. `TURSO_AUTH_TOKEN` environment variable

See [Turso Quick Start](../TURSO_QUICKSTART.md) for a full setup walkthrough.

---

### shark cloud status

Display the current cloud database configuration including backend type, connection URL, and authentication status. Reads from `.sharkconfig.json`.

```
Usage:
  shark cloud status [flags]
```

**Examples:**

```bash
# Show current cloud status
shark cloud status

# Show status in JSON format
shark cloud status --json
```

**Sample output:**

```
Cloud Database Status
--------------------
Cloud database is CONFIGURED
Backend: turso
URL: libsql://shark-tasks-yourorg.turso.io
Auth token file: /home/user/.turso/shark-token
```

---

## Workflow

### shark workflow

Parent command for workflow configuration operations including listing, validation, and action inspection. The workflow system allows customizing task status transitions via `.sharkconfig.json`.

```
Usage:
  shark workflow [command]
```

**Available Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `list` | Display configured workflow |
| `validate` | Validate workflow configuration |
| `show-actions` | Display workflow orchestrator actions |
| `validate-actions` | Validate workflow orchestrator actions |

---

### shark workflow list

Display the configured status workflow from `.sharkconfig.json`. Shows all statuses and their valid transitions, highlighting special statuses (`_start_` and `_complete_`). If no custom workflow is configured, displays the default workflow.

```
Usage:
  shark workflow list [flags]
```

**Examples:**

```bash
# Display workflow (human-readable table)
shark workflow list

# Display workflow (JSON format)
shark workflow list --json
```

---

### shark workflow validate

Validate the workflow configuration in `.sharkconfig.json` for correctness.

Checks performed:
- Required special statuses (`_start_`, `_complete_`) are defined
- All status references in transitions are defined
- All statuses are reachable from `_start_` statuses
- All statuses have a path to `_complete_` statuses
- No circular references with no terminal path

**Exit codes:**
- `0` -- Configuration is valid
- `2` -- Configuration is invalid (specific errors displayed)

```
Usage:
  shark workflow validate [flags]
```

**Examples:**

```bash
# Validate configuration
shark workflow validate

# Validate with JSON output
shark workflow validate --json
```

---

### shark workflow show-actions

Display all orchestrator actions defined in the workflow configuration. Shows actions grouped by workflow phase with agent types and skills. Displays all three entity levels (epic, feature, task) by default.

**Exit codes:**
- `0` -- Success
- `1` -- Status or action type not found
- `2` -- Configuration error

```
Usage:
  shark workflow show-actions [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--status <status>` | Show action for a specific status only |
| `--action-type <type>` | Filter by action type (`spawn_agent`, `pause`, `wait_for_triage`, `archive`) |
| `--level <level>` | Filter by entity level (`epic`, `feature`, `task`) |

**Examples:**

```bash
# Show all levels
shark workflow show-actions

# Show only epic actions
shark workflow show-actions --level=epic

# Task actions in JSON
shark workflow show-actions --level=task --json

# Show action for a specific status
shark workflow show-actions --status=ready_for_development

# Filter by action type
shark workflow show-actions --action-type=spawn_agent --json
```

---

### shark workflow validate-actions

Validate that all orchestrator actions in the workflow configuration are properly defined.

Checks performed:
- Action schema correctness (valid action types, required fields)
- Completeness (actionable statuses have actions defined)
- `spawn_agent` actions have required `agent_type` and `skills`
- `instruction_templates` are non-empty and syntactically valid

**Exit codes:**
- `0` -- Validation passed (or passed with warnings in non-strict mode)
- `1` -- Validation failed (errors found, or warnings in `--strict` mode)

```
Usage:
  shark workflow validate-actions [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--level <level>` | Filter by entity level (`epic`, `feature`, `task`) |
| `--strict` | Fail with exit code 1 if any status lacks an orchestrator action |

**Examples:**

```bash
# Validate all levels
shark workflow validate-actions

# Validate only task workflow
shark workflow validate-actions --level=task

# Validate only epic workflow
shark workflow validate-actions --level=epic

# Strict mode -- fail on any warnings
shark workflow validate-actions --strict

# JSON output
shark workflow validate-actions --json
```

---

## Related Documentation

- [Global Flags](global-flags.md) -- Flags available to all commands
- [Workflow Configuration](workflow-config.md) -- Customize status flows, colors, phases
- [Workflow Profiles Guide](../guides/workflow-profiles.md) -- Apply predefined workflow profiles
- [Turso Quick Start](../TURSO_QUICKSTART.md) -- Cloud database setup guide
- [Turso Migration Guide](../TURSO_MIGRATION.md) -- Migrate local data to cloud
- [Configuration Commands](configuration.md) -- View and manage config settings
