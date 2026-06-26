# Initialization

Initialize Shark CLI infrastructure in the current project.

## Command

### `shark admin init`

**Flags:**
- `--non-interactive` — Skip interactive prompts (recommended for automation)
- `--force` — Overwrite existing config file

**Examples:**

```bash
# Interactive mode
shark admin init

# Non-interactive mode (for AI agents / CI)
shark admin init --non-interactive

# Force overwrite existing config
shark admin init --force
```

**Creates:**
- SQLite database (`shark-tasks.db`)
- Folder structure (`docs/plan/`)
- Configuration file (`.sharkconfig.json`) with sensible defaults

Content (workflows, prompts, skills, agents) is served from the embedded
bundle by default — no `shark-data/` directory is required on disk. Run
`shark admin install-shark-data` to extract the bundle to disk for local
customization.

## Default `.sharkconfig.json`

A fresh `shark admin init` writes a minimal config pointing at the
`shark-data/` bundle for workflow and content:

```json
{
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": false,
  "require_rejection_reason": true,
  "shark_data_path": "shark-data",
  "workflow_config": "shark-data/workflow/",
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db",
    "skip_migrations": false
  },
  "observability": {
    "enabled": false,
    "tracing_enabled": false,
    "metrics_enabled": false,
    "log_level": "info",
    "log_format": "json",
    "service_name": "shark-task-manager",
    "log_file": ""
  }
}
```

The `workflow_config` field points at the `shark-data/workflow/` directory,
which is resolved from the embedded bundle when no `shark-data/` tree exists
on disk. To customize workflows, run `shark admin install-shark-data` to
extract the bundle, then edit `shark-data/workflow/*.yaml` directly.

## When to Use

Run `shark admin init` when:
- Starting a new project
- Setting up Shark in an existing project
- Recreating the database after deletion (see recovery procedures)

## Cautions

⚠️ **DO NOT** run `shark admin init` if you already have a database with
data unless you understand that `--force` is required to clobber an
existing `.sharkconfig.json`. The database itself is never destroyed by
init.

See [Database Critical](../../.claude/rules/database-critical.md) for
recovery procedures if you accidentally delete the database.

## Extracting content bundle to disk

To customize workflows, prompts, skills, or agents, extract the embedded
bundle to disk:

```bash
shark admin install-shark-data
```

This writes `shark-data/` to the project root. The `shark-data/overrides/`
subtree is never overwritten by a subsequent `install-shark-data` or
`upgrade` call, so your customizations are preserved.

## Migrating an existing project to v15 (E07 size field + E28 tagging)

Recent schema bumps:

- **v14** (E28 tagging) — added `tags` and `entity_tags` tables for the
  managed tag vocabulary.
- **v15** (E07 size) — added `size` and `size_label` columns across the six
  entity tables (`epics`, `features`, `tasks`, `bugs`, `change_cards`,
  `ideas`).

### Local SQLite users

No action required. Local SQLite databases always run pending migrations on
the next `shark` invocation, so any pending upgrades apply automatically the
first time you run any command after upgrading the binary.

### Turso / cloud users running with `skip_migrations: true`

Turso users typically set `database.skip_migrations: true` in
`.sharkconfig.json` to avoid the ~2-second DDL overhead on every command.
With that flag set, `ApplySchemaIfNeeded` (in `internal/db/db.go`) compares
the recorded `schema_version` against `CurrentSchemaVersion` (currently
**15**) and **automatically runs any pending migrations** when a gap is
detected — no manual toggle required. Simply upgrade the binary and run any
`shark` command; the migration applies once and records the new version.

> See [`.claude/rules/database-critical.md`](../../.claude/rules/database-critical.md)
> for the full rationale behind `skip_migrations` and the
> `CurrentSchemaVersion` bump procedure used by `internal/db/db.go`.

## Switching workflows

Workflows are defined as YAML files inside `shark-data/workflow/` and
selected by the `workflow_config` field in `.sharkconfig.json`. To
customize:

1. Run `shark admin install-shark-data` to extract the bundle.
2. Edit `shark-data/workflow/<entity>.yaml` (task, feature, epic, bug, etc.).
3. Or place overrides under `shark-data/overrides/workflow/` — these layer
   on top and are never overwritten.

See [Workflow Configuration Guide](../guides/workflow-profiles.md) for
details.

## Related Documentation

- [Configuration](configuration.md) — Configure Shark after initialization
- [Workflow Configuration](workflow-configuration.md) — Workflow file structure
- [Turso Quickstart](../TURSO_QUICKSTART.md) — Cloud database setup
