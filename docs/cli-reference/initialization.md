# Initialization

Initialize Shark CLI infrastructure in the current project.

## Command

### `shark admin init`

**Flags:**
- `--non-interactive` — Skip interactive prompts (recommended for automation)
- `--force` — Overwrite existing config and locally-modified template files

**Examples:**

```bash
# Interactive mode
shark admin init

# Non-interactive mode (for AI agents / CI)
shark admin init --non-interactive

# Force overwrite existing config and locally-modified templates
shark admin init --force
```

**Creates:**
- SQLite database (`shark-tasks.db`)
- Folder structure (`docs/plan/`, `shark-templates/`)
- Configuration file (`.sharkconfig.json`) with sensible defaults
- The full embedded `shark-templates/` tree (entity templates, orchestrator templates, partials, and the bundled workflow files)

## Default `.sharkconfig.json`

A fresh `shark admin init` writes a config that points at a workflow file
inside `shark-templates/` rather than embedding status definitions inline:

```json
{
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": false,
  "require_rejection_reason": true,
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db",
    "skip_migrations": false
  },
  "workflow_config": "shark-templates/.sharkworkflow-short.json"
}
```

To switch to the long-form workflow, edit `workflow_config` to point at
`shark-templates/.sharkworkflow.json`. To use a custom workflow, point
`workflow_config` at your own JSON file (anywhere in the project).

## Templates re-sync behavior

`shark admin init` is **idempotent and re-syncs templates on every run** so
that template/workflow updates shipped with a new `shark` binary
automatically flow through to your project. Per file:

| Local state | Default behavior | With `--force` |
|---|---|---|
| File missing | Copy fresh | Copy fresh |
| File matches shipped version | Skip silently | Skip silently |
| File **differs** from shipped version | Skip + warn (your edits are preserved) | **Overwrite** |

When local files differ from the shipped versions, init prints a warning
listing each path so you can review the upstream changes and decide whether
to accept them with `--force`.

## When to Use

Run `shark admin init` when:
- Starting a new project
- Setting up Shark in an existing project
- Picking up template/workflow updates after upgrading the `shark` binary
- Recreating the database after deletion (see recovery procedures)

## Cautions

⚠️ **DO NOT** run `shark admin init` if you already have a database with
data unless you understand that `--force` is required to clobber an
existing `.sharkconfig.json`. The database itself is never destroyed by
init.

⚠️ `--force` will overwrite locally-modified template files with the
shipped versions. Review the warnings printed by a normal init run first.

See [Database Critical](../../.claude/rules/database-critical.md) for
recovery procedures if you accidentally delete the database.

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

The basic/advanced workflow "profiles" subsystem (`shark init update
--workflow=...` and `shark init merge ...`) was removed. Workflows now live
as files inside `shark-templates/` and are selected by setting
`workflow_config` in `.sharkconfig.json`:

| Old approach | New approach |
|---|---|
| `shark init update --workflow=basic` | Set `"workflow_config": "shark-templates/.sharkworkflow-short.json"` |
| `shark init update --workflow=advanced` | Set `"workflow_config": "shark-templates/.sharkworkflow.json"` |
| `shark init merge --workflow=advanced --force` | Same as above — edit `workflow_config` directly |

To customize a workflow, copy one of the shipped files to your own path,
edit it, and point `workflow_config` at your copy. Init will leave your
custom file alone (it only re-syncs files inside `shark-templates/`).

## Related Documentation

- [Configuration](configuration.md) — Configure Shark after initialization
- [Workflow Configuration](workflow-configuration.md) — Workflow file structure
- [Turso Quickstart](../TURSO_QUICKSTART.md) — Cloud database setup
