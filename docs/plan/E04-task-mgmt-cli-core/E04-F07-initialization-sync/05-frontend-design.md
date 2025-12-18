# Frontend Design (CLI UX): Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-16
**Author**: frontend-architect

## Purpose

This document defines the CLI user experience, command-line interfaces, output formatting, and interaction patterns for `pm init` and `pm sync` commands.

---

## Command: `pm init`

### Basic Usage

```bash
# Initialize with defaults
pm init

# Non-interactive mode (for automation)
pm init --non-interactive

# Force overwrite existing config
pm init --force

# Specify custom database path
pm init --db=custom.db

# Specify custom config path
pm init --config=.custom-config.json
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--non-interactive` | | bool | false | Skip all prompts, use defaults |
| `--force` | | bool | false | Overwrite existing config and templates |
| `--db` | | string | shark-tasks.db | Database file path |
| `--config` | | string | .pmconfig.json | Config file path |
| `--json` | | bool | false | JSON output (from global flags) |
| `--no-color` | | bool | false | Disable colored output (from global flags) |

### Human-Readable Output

**Success**:
```
Shark CLI initialized successfully!

✓ Database created: shark-tasks.db
✓ Folder structure created: docs/plan/, templates/
✓ Config file created: .pmconfig.json
✓ Templates copied: 2 files

Next steps:
1. Edit .pmconfig.json to set default epic and agent
2. Create tasks with: pm task create --epic=E01 --feature=F01 --title="Task title" --agent=backend
3. Import existing tasks with: pm sync
```

**Idempotent Run** (already initialized):
```
Shark CLI initialized successfully!

✓ Database exists: shark-tasks.db
✓ Folder structure exists: docs/plan/, templates/
✓ Config file exists: .pmconfig.json

Next steps:
1. Edit .pmconfig.json to set default epic and agent
2. Create tasks with: pm task create --epic=E01 --feature=F01 --title="Task title" --agent=backend
3. Import existing tasks with: pm sync
```

**Interactive Prompt** (config exists, no --force):
```
Config file already exists at .pmconfig.json. Overwrite? (y/N): n
Skipping config file creation.

Shark CLI initialized successfully!

✓ Database exists: shark-tasks.db
✓ Folder structure exists: docs/plan/, templates/
✓ Config file exists: .pmconfig.json (not modified)
```

### JSON Output

```json
{
  "database_created": true,
  "database_path": "/abs/path/to/shark-tasks.db",
  "folders_created": ["/abs/path/to/docs/plan", "/abs/path/to/templates"],
  "config_created": true,
  "config_path": "/abs/path/to/.pmconfig.json",
  "templates_copied": 2
}
```

### Error Output

**Human-Readable**:
```
✗ Error: Failed to create database: permission denied
```

**JSON**:
```json
{
  "status": "error",
  "error": "initialization failed at step 'database': Failed to create database: permission denied"
}
```

---

## Command: `pm sync`

### Basic Usage

```bash
# Sync all feature folders
pm sync

# Sync specific folder
pm sync --folder=docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation

# Sync legacy task folders
pm sync --folder=docs/tasks/todo

# Preview changes (dry-run)
pm sync --dry-run

# Use database-wins strategy
pm sync --strategy=database-wins

# Auto-create missing epics/features
pm sync --create-missing

# Delete orphaned database tasks
pm sync --cleanup

# JSON output for automation
pm sync --json

# Combined: dry-run + file-wins + create-missing
pm sync --dry-run --strategy=file-wins --create-missing
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--folder` | | string | docs/plan | Sync specific folder only |
| `--dry-run` | | bool | false | Preview changes without applying |
| `--strategy` | | string | file-wins | Conflict resolution: file-wins, database-wins, newer-wins |
| `--create-missing` | | bool | false | Auto-create missing epics/features |
| `--cleanup` | | bool | false | Delete orphaned database tasks (files deleted) |
| `--json` | | bool | false | JSON output (from global flags) |
| `--no-color` | | bool | false | Disable colored output (from global flags) |

### Human-Readable Output

**Successful Sync** (no conflicts):
```
✓ Sync completed:
  Files scanned: 47
  New tasks imported: 5
  Existing tasks updated: 3
  Conflicts resolved: 0
  Warnings: 0
  Errors: 0
```

**Sync with Conflicts**:
```
✓ Sync completed:
  Files scanned: 47
  New tasks imported: 5
  Existing tasks updated: 3
  Conflicts resolved: 2
  Warnings: 1
  Errors: 0

Conflicts:
  T-E01-F02-003:
    Field: title
    Database: "Implement user authentication"
    File: "Add user authentication feature"
    Resolution: file-wins (title updated to "Add user authentication feature")

  T-E04-F07-001:
    Field: file_path
    Database: "docs/tasks/created/T-E04-F07-001.md"
    File: "docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-001.md"
    Resolution: file-wins (file_path updated to actual location)

Warnings:
  - Invalid frontmatter in docs/tasks/legacy/invalid.md, skipping
```

**Dry-Run Mode**:
```
⚠ Dry-run mode: No changes will be made

✓ Sync completed:
  Files scanned: 47
  New tasks imported: 5 (preview)
  Existing tasks updated: 3 (preview)
  Conflicts resolved: 2 (preview)
  Warnings: 0
  Errors: 0

Conflicts:
  [... conflicts listed ...]
```

**Sync with Cleanup**:
```
✓ Sync completed:
  Files scanned: 47
  New tasks imported: 5
  Existing tasks updated: 3
  Tasks deleted (orphaned): 2
  Conflicts resolved: 0
  Warnings: 0
  Errors: 0
```

**No Changes**:
```
✓ Sync completed:
  Files scanned: 47
  New tasks imported: 0
  Existing tasks updated: 0
  Conflicts resolved: 0
  Warnings: 0
  Errors: 0

No changes needed. Database is up to date.
```

### JSON Output

```json
{
  "files_scanned": 47,
  "tasks_imported": 5,
  "tasks_updated": 3,
  "tasks_deleted": 0,
  "conflicts_resolved": 2,
  "warnings": [
    "Invalid frontmatter in docs/tasks/legacy/invalid.md, skipping"
  ],
  "errors": [],
  "conflicts": [
    {
      "task_key": "T-E01-F02-003",
      "field": "title",
      "database_value": "Implement user authentication",
      "file_value": "Add user authentication feature",
      "resolution": "file-wins"
    },
    {
      "task_key": "T-E04-F07-001",
      "field": "file_path",
      "database_value": "docs/tasks/created/T-E04-F07-001.md",
      "file_value": "docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-001.md",
      "resolution": "file-wins"
    }
  ]
}
```

### Error Scenarios

**Missing Epic/Feature** (without --create-missing):
```
✗ Error: Sync failed: task T-E99-F99-001 references non-existent feature E99-F99
Suggestion: Use --create-missing to auto-create missing epics/features, or create feature E99-F99 first with:
  pm feature create --epic=E99 --key=E99-F99 --title="Feature Title"
```

**Database Connection Error**:
```
✗ Error: Failed to create sync engine: failed to open database: no such file or directory
Suggestion: Run 'pm init' first to initialize the database.
```

**Invalid Strategy**:
```
✗ Error: Invalid strategy: unknown-strategy (valid: file-wins, database-wins, newer-wins)
```

---

## Interactive Workflows

### First-Time Setup

```bash
# Step 1: Clone project
git clone https://github.com/user/project.git
cd project

# Step 2: Initialize Shark CLI
pm init

Shark CLI initialized successfully!

✓ Database created: shark-tasks.db
✓ Folder structure created: docs/plan/, templates/
✓ Config file created: .pmconfig.json
✓ Templates copied: 2 files

Next steps:
1. Edit .pmconfig.json to set default epic and agent
2. Create tasks with: pm task create --epic=E01 --feature=F01 --title="Task title" --agent=backend
3. Import existing tasks with: pm sync

# Step 3: Import existing task files
pm sync --create-missing

✓ Sync completed:
  Files scanned: 120
  New tasks imported: 120
  Existing tasks updated: 0
  Conflicts resolved: 0
  Warnings: 0
  Errors: 0

# Done!
```

### Git Pull Workflow

```bash
# Step 1: Pull changes from collaborators
git pull origin main

remote: Counting objects: 15, done.
remote: Compressing objects: 100% (10/10), done.
remote: Total 15 (delta 5), reused 15 (delta 5)
Unpacking objects: 100% (15/15), done.
From github.com:user/project
   abc1234..def5678  main       -> origin/main
 M docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-001.md
 A docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-005.md
Updating abc1234..def5678
Fast-forward
 2 files changed, 50 insertions(+), 10 deletions(-)

# Step 2: Sync database with file changes
pm sync

✓ Sync completed:
  Files scanned: 121
  New tasks imported: 1
  Existing tasks updated: 1
  Conflicts resolved: 1
  Warnings: 0
  Errors: 0

Conflicts:
  T-E04-F07-001:
    Field: title
    Database: "Implement sync engine"
    File: "Add sync engine implementation"
    Resolution: file-wins (title updated to "Add sync engine implementation")

# Done! Database is now in sync with filesystem
```

### Preview Changes Before Syncing

```bash
# Step 1: Run dry-run to preview
pm sync --dry-run

⚠ Dry-run mode: No changes will be made

✓ Sync completed:
  Files scanned: 47
  New tasks imported: 5 (preview)
  Existing tasks updated: 3 (preview)
  Conflicts resolved: 2 (preview)
  Warnings: 0
  Errors: 0

Conflicts:
  T-E01-F02-003:
    Field: title
    Database: "Implement user authentication"
    File: "Add user authentication feature"
    Resolution: file-wins (title updated to "Add user authentication feature")

# Step 2: Review changes, then apply if acceptable
pm sync

✓ Sync completed:
  Files scanned: 47
  New tasks imported: 5
  Existing tasks updated: 3
  Conflicts resolved: 2
  Warnings: 0
  Errors: 0

# Changes applied!
```

---

## Color and Formatting

### Color Scheme

| Element | Color | Symbol |
|---------|-------|--------|
| Success | Green | ✓ |
| Warning | Yellow | ⚠ |
| Error | Red | ✗ |
| Info | Blue | ℹ |
| Section Header | Cyan (bold) | |

### No-Color Mode

When `--no-color` flag is set or output is not a TTY:

```
[SUCCESS] Sync completed:
  Files scanned: 47
  New tasks imported: 5
  Existing tasks updated: 3
  Conflicts resolved: 2
  Warnings: 0
  Errors: 0
```

---

## Progress Indicators

### File Scanning Progress (future enhancement)

```
Scanning files... 47 files found
Parsing frontmatter... 45/47 files parsed
Syncing with database... 40/45 tasks processed
✓ Sync completed
```

### Spinner During Long Operations

```
⠋ Syncing 100 files with database...
```

---

## Help Text

### pm init --help

```
Initialize Shark CLI infrastructure

Usage:
  pm init [flags]

Description:
  Initialize Shark CLI infrastructure by creating database schema,
  folder structure, configuration file, and task templates.

  This command is idempotent and safe to run multiple times.

Examples:
  # Initialize with default settings
  pm init

  # Initialize without prompts (for automation)
  pm init --non-interactive

  # Force overwrite existing config
  pm init --force

Flags:
      --non-interactive    Skip all prompts (use defaults)
      --force              Overwrite existing config and templates
      --db string          Database file path (default: shark-tasks.db)
      --config string      Config file path (default: .pmconfig.json)

Global Flags:
      --json               Output in JSON format
      --no-color           Disable colored output
  -v, --verbose            Enable verbose/debug output
```

### pm sync --help

```
Synchronize task files with database

Usage:
  pm sync [flags]

Description:
  Synchronize task markdown files with the database by scanning feature folders,
  parsing frontmatter, detecting conflicts, and applying resolution strategies.

  Status is managed exclusively in the database and is NOT synced from files.

Examples:
  # Sync all feature folders
  pm sync

  # Sync specific folder
  pm sync --folder=docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation

  # Preview changes without applying (dry-run)
  pm sync --dry-run

  # Use database-wins strategy for conflicts
  pm sync --strategy=database-wins

  # Auto-create missing epics/features
  pm sync --create-missing

  # Delete orphaned database tasks (files deleted)
  pm sync --cleanup

Flags:
      --folder string      Sync specific folder only (default: docs/plan)
      --dry-run            Preview changes without applying them
      --strategy string    Conflict resolution strategy: file-wins, database-wins, newer-wins (default: file-wins)
      --create-missing     Auto-create missing epics/features
      --cleanup            Delete orphaned database tasks (files deleted)

Global Flags:
      --json               Output in JSON format
      --no-color           Disable colored output
  -v, --verbose            Enable verbose/debug output
      --db string          Database file path (default: shark-tasks.db)
```

---

## Accessibility Considerations

### Screen Reader Support

- Use semantic labels: [SUCCESS], [WARNING], [ERROR], [INFO]
- Provide complete sentences in messages
- Avoid relying solely on color or symbols

### Unicode Fallback

If terminal doesn't support Unicode symbols:

```
[SUCCESS] Sync completed:
  Files scanned: 47
  New tasks imported: 5
  ...
```

---

## Exit Codes

| Exit Code | Meaning | Example |
|-----------|---------|---------|
| 0 | Success | Sync completed, init completed |
| 1 | User error | Invalid flags, missing required args |
| 2 | System error | Database error, filesystem error, transaction rollback |
| 130 | User interrupt (Ctrl+C) | User cancelled operation |

---

## Verbose Mode

When `-v` or `--verbose` flag is set:

```
pm sync -v

[DEBUG] Using database: /abs/path/to/shark-tasks.db
[DEBUG] Scanning folder: docs/plan
[DEBUG] Found 47 task files
[DEBUG] Parsing T-E04-F07-001.md
[DEBUG] Parsing T-E04-F07-002.md
...
[DEBUG] Querying database for 47 task keys
[DEBUG] Found 42 tasks in database, 5 new tasks
[DEBUG] Beginning transaction
[DEBUG] Importing T-E04-F07-003
[DEBUG] Updating T-E04-F07-001 (conflict: title)
...
[DEBUG] Committing transaction
✓ Sync completed:
  Files scanned: 47
  New tasks imported: 5
  Existing tasks updated: 3
  Conflicts resolved: 2
  Warnings: 0
  Errors: 0
```

---

## Automation Support

### JSON Output for CI/CD

```bash
# Use JSON output in scripts
pm sync --json > sync-report.json

# Parse with jq
FILES_SCANNED=$(jq '.files_scanned' sync-report.json)
CONFLICTS=$(jq '.conflicts_resolved' sync-report.json)

if [ "$CONFLICTS" -gt 0 ]; then
  echo "Warning: $CONFLICTS conflicts detected"
  jq '.conflicts' sync-report.json
fi
```

### Non-Interactive Mode

```bash
# Initialize in CI/CD
pm init --non-interactive --json

# Sync with auto-create missing
pm sync --create-missing --json
```

---

## User Feedback Mechanisms

### Warnings vs Errors

**Warnings** (non-fatal, logged but sync continues):
- Invalid YAML frontmatter
- Missing required field (key)
- Key mismatch (filename vs frontmatter)
- File read permission denied

**Errors** (fatal, sync halts and rolls back):
- Database connection failure
- Transaction begin/commit failure
- Foreign key constraint violation (missing epic/feature without --create-missing)

### Suggestions in Error Messages

Always provide actionable suggestions:

```
✗ Error: task T-E99-F99-001 references non-existent feature E99-F99
Suggestion: Use --create-missing to auto-create missing epics/features, or create feature E99-F99 first with:
  pm feature create --epic=E99 --key=E99-F99 --title="Feature Title"
```

---

**Document Complete**: 2025-12-16
**Next Document**: 06-security-design.md (security-architect creates)
