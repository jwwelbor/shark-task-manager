# Initialization

Initialize Shark CLI infrastructure in the current project.

## Command

### `shark init`

**Flags:**
- `--non-interactive`: Skip interactive prompts (recommended for automation)

**Examples:**

```bash
# Interactive mode
shark init

# Non-interactive mode (for AI agents)
shark init --non-interactive
```

**Creates:**
- SQLite database (`shark-tasks.db`)
- Folder structure (`docs/plan/`)
- Configuration file (`.sharkconfig.json`)
- Templates directory (`shark-templates/`)

## When to Use

Run `shark init` when:
- Starting a new project
- Setting up Shark in an existing project
- Recreating the database after deletion (see recovery procedures)

## Cautions

⚠️ **DO NOT** run `shark init` if you already have a database with data. This can cause data loss.

See [Database Critical](../../.claude/rules/database-critical.md) for recovery procedures if you accidentally delete the database.

## Related Documentation

- [Configuration](configuration.md) - Configure Shark after initialization
- [Turso Quickstart](../TURSO_QUICKSTART.md) - Cloud database setup
