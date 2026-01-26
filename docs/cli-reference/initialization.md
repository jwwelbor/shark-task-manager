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

### `shark init update`

**Update Shark configuration with workflow profiles or add missing fields.**

**Flags:**
- `--workflow=<profile>` - Apply workflow profile (basic or advanced)
- `--force` - Overwrite existing status configurations
- `--dry-run` - Preview changes without applying
- `--json` - Output results as JSON
- `--verbose` - Show detailed merge information

**Examples:**

```bash
# Add missing configuration fields
shark init update

# Apply basic workflow (5 statuses)
shark init update --workflow=basic

# Apply advanced workflow (19 statuses)
shark init update --workflow=advanced

# Preview changes before applying
shark init update --workflow=advanced --dry-run

# Force overwrite existing configurations
shark init update --workflow=basic --force

# Get JSON output for automation
shark init update --workflow=basic --json
```

**Workflow Profiles:**

**basic** (5 statuses):
- Simple workflow for solo developers
- Statuses: todo, in_progress, ready_for_review, completed, blocked
- No status flow constraints
- Single agent type

**advanced** (19 statuses):
- Comprehensive TDD workflow for teams
- Statuses cover full SDLC: planning, development, review, QA, approval
- Status flow enforcement
- Multiple agent types (ba, tech_lead, developer, qa, product_owner)
- Special status groups

**Behavior:**

**Without `--workflow` flag:**
- Adds missing configuration fields
- Preserves all existing values
- Safe for existing configs

**With `--workflow` flag:**
- Applies specified profile
- Overwrites status_metadata, status_flow, special_statuses
- Preserves database, viewer, project_root
- Creates timestamped backup

**Dry-run mode:**
- Shows preview of changes
- Does not modify files
- Useful for validation

**Protected Fields:**

These fields are NEVER overwritten (without --force):
- `database` - Database configuration
- `project_root` - Project root path
- `viewer` - File viewer configuration
- `last_sync_time` - Last sync timestamp

**Backup:**

Before any write operation, a timestamped backup is created:
```
.sharkconfig.json.backup.YYYYMMDD-HHMMSS
```

Backups can be restored manually if needed.

## Related Documentation

- [Configuration](configuration.md) - Configure Shark after initialization
- [Workflow Profiles Guide](../guides/workflow-profiles.md) - Comprehensive workflow profile guide
- [Turso Quickstart](../TURSO_QUICKSTART.md) - Cloud database setup
