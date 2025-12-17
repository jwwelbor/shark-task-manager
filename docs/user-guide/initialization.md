# PM CLI Initialization Guide

## Overview

The `pm init` command sets up the PM CLI infrastructure for your project, creating the necessary database, folder structure, configuration file, and task templates in a single operation.

## Quick Start

```bash
pm init
```

This command will:
1. Create the SQLite database (`shark-tasks.db`) with the complete schema
2. Set up the folder structure under `docs/plan/`
3. Create a default configuration file (`.pmconfig.json`)
4. Copy task templates to the `templates/` directory

## Command Options

### Basic Usage

```bash
pm init [flags]
```

### Available Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--non-interactive` | Skip all prompts, use defaults | `false` |
| `--force` | Overwrite existing config and templates | `false` |
| `--db-path <path>` | Custom database file path | `shark-tasks.db` |
| `--config-path <path>` | Custom config file path | `.pmconfig.json` |
| `--json` | Output results in JSON format | `false` |

### Examples

#### Initialize with defaults
```bash
pm init
```

#### Initialize without prompts (for CI/CD)
```bash
pm init --non-interactive
```

#### Force overwrite existing config
```bash
pm init --force
```

#### Use custom paths
```bash
pm init --db-path=./data/tasks.db --config-path=./config/pm.json
```

#### JSON output for automation
```bash
pm init --json --non-interactive
```

## What Gets Created

### 1. Database File

Location: `shark-tasks.db` (or custom path)

The database includes:
- `epics` table - Top-level project organization
- `features` table - Mid-level feature tracking
- `tasks` table - Individual work items
- `task_history` table - Audit trail of changes
- Indexes for performance
- Foreign key constraints
- Triggers for automatic timestamp updates

**File permissions**: 600 (read/write for owner only) on Unix systems

### 2. Folder Structure

```
docs/
└── plan/
    ├── E01-epic-name/
    │   └── E01-F01-feature-name/
    │       └── T-E01-F01-001.md
    └── ...
templates/
└── task-template.md
```

**Folder permissions**: 755 (standard directory permissions)

### 3. Configuration File

Location: `.pmconfig.json`

```json
{
  "default_epic": null,
  "default_agent": null,
  "color_enabled": true,
  "json_output": false
}
```

**Customization**:
- `default_epic`: Set your most-used epic key (e.g., "E04")
- `default_agent`: Set default agent type (e.g., "backend", "frontend")
- `color_enabled`: Enable/disable colored output
- `json_output`: Default output format

### 4. Task Templates

Templates are copied to the `templates/` directory for easy task creation:
- `task-template.md` - Standard task template with frontmatter

## Idempotency

The `pm init` command is idempotent - you can run it multiple times safely:

- **Existing database**: Skipped (not overwritten)
- **Existing folders**: Skipped (not recreated)
- **Existing config**: Prompts for confirmation (unless `--non-interactive` or `--force`)
- **Existing templates**: Skipped unless `--force` is used

### Re-running Init

```bash
# Safe to run again - skips existing files
pm init

# Force overwrite config and templates
pm init --force

# No prompts, skip existing files
pm init --non-interactive
```

## Performance

`pm init` completes in **< 5 seconds** on typical systems, including:
- Database schema creation
- Folder creation
- Config file generation
- Template copying

## Next Steps

After initialization, you can:

1. **Configure defaults** - Edit `.pmconfig.json`:
   ```bash
   nano .pmconfig.json
   ```

2. **Create your first epic**:
   ```bash
   pm epic create --key=E01 --title="My First Epic"
   ```

3. **Create a feature**:
   ```bash
   pm feature create --epic=E01 --key=E01-F01 --title="My First Feature"
   ```

4. **Create tasks**:
   ```bash
   pm task create --feature=E01-F01 --title="My First Task" --agent=backend
   ```

5. **Import existing tasks** (if you have markdown files):
   ```bash
   pm sync --create-missing
   ```

## Troubleshooting

### Database Already Exists

**Symptom**: Message "Database already exists, skipping"

**Solution**: This is normal if you've initialized before. The existing database is preserved.

If you want to start fresh:
```bash
rm shark-tasks.db
pm init
```

### Permission Denied

**Symptom**: Error creating database or folders

**Solution**: Ensure you have write permissions in the current directory:
```bash
ls -la | grep shark-tasks.db
chmod 755 .  # For directory
```

### Config File Prompt in CI/CD

**Symptom**: Init hangs waiting for input in automated scripts

**Solution**: Use `--non-interactive` flag:
```bash
pm init --non-interactive
```

### Templates Not Copied

**Symptom**: `templates/` directory is empty

**Solution**: Re-run with `--force` to copy templates:
```bash
pm init --force
```

## Integration with Existing Projects

If you have an existing project with task files:

1. **Initialize PM CLI**:
   ```bash
   pm init
   ```

2. **Sync existing task files**:
   ```bash
   pm sync --create-missing --dry-run  # Preview changes
   pm sync --create-missing             # Import tasks
   ```

See the [Synchronization Guide](synchronization.md) for details on importing existing tasks.

## Configuration Examples

### Team Configuration

For team projects, share `.pmconfig.json` in version control:

```json
{
  "default_epic": "E04",
  "default_agent": "backend",
  "color_enabled": true,
  "json_output": false
}
```

### CI/CD Configuration

For automated environments, disable colors and use JSON:

```json
{
  "default_epic": null,
  "default_agent": null,
  "color_enabled": false,
  "json_output": true
}
```

## Security Considerations

1. **Database Permissions**: The database file is created with 600 permissions on Unix systems (owner read/write only).

2. **Config File**: Contains no sensitive data by default. If you add API keys or secrets, ensure proper permissions:
   ```bash
   chmod 600 .pmconfig.json
   ```

3. **Version Control**: You can safely commit `.pmconfig.json` to Git (no secrets). Add database to `.gitignore`:
   ```bash
   echo "shark-tasks.db" >> .gitignore
   echo "shark-tasks.db-shm" >> .gitignore
   echo "shark-tasks.db-wal" >> .gitignore
   ```

## Advanced Usage

### Custom Database Location

Store database in a dedicated data directory:

```bash
mkdir -p data
pm init --db-path=data/project-tasks.db
```

### Multiple Projects

Use different databases for different projects:

```bash
# Project A
cd ~/projects/project-a
pm init --db-path=tasks-a.db

# Project B
cd ~/projects/project-b
pm init --db-path=tasks-b.db
```

### Programmatic Access

Use JSON output for programmatic access:

```bash
result=$(pm init --json --non-interactive)
echo $result | jq '.database_created'
```

## See Also

- [Synchronization Guide](synchronization.md) - Import and sync task files
- [Task Management](../CLI.md) - Create and manage tasks
- [Troubleshooting](../troubleshooting.md) - Common issues and solutions
