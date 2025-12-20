# Shark CLI Synchronization Guide

## Overview

The `shark sync` command synchronizes task markdown files with the database, enabling Git-based workflows where task files are edited in text editors, merged via Git, and kept in sync with the Shark CLI database.

## Quick Start

```bash
# Preview what would change (dry-run)
shark sync --dry-run

# Sync all task files
shark sync

# Sync and create missing epics/features
shark sync --create-missing
```

## When to Use Sync

Use `shark sync` when:
- You've pulled Git changes that include new or modified task files
- You've edited task files directly in a text editor
- You're migrating existing task markdown files to Shark CLI
- You want to ensure database reflects current file state

## How Sync Works

### File Organization

Tasks are organized under feature folders:
```
docs/
└── plan/
    └── E04-task-mgmt-cli/
        └── E04-F07-init-sync/
            ├── T-E04-F07-001.md
            ├── T-E04-F07-002.md
            └── T-E04-F07-003.md
```

**Key concept**: Tasks remain in their feature folders regardless of status changes. Status is managed in the database only, not in files or file locations.

### Sync Process

1. **Scan** - Recursively find all task markdown files
2. **Parse** - Extract frontmatter (key, title, description)
3. **Query** - Look up tasks in database by key
4. **Compare** - Detect conflicts between file and database
5. **Resolve** - Apply conflict resolution strategy
6. **Update** - Modify database in a single transaction

## Command Options

### Basic Usage

```bash
shark sync [flags]
```

### Available Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--folder <path>` | Sync only specific folder | All folders |
| `--dry-run` | Preview changes without applying | `false` |
| `--strategy <strategy>` | Conflict resolution strategy | `file-wins` |
| `--create-missing` | Auto-create missing epics/features | `false` |
| `--cleanup` | Delete orphaned database tasks | `false` |
| `--force` | Overwrite database (alias for --strategy=file-wins) | `false` |
| `--json` | Output results in JSON format | `false` |

### Conflict Resolution Strategies

| Strategy | Behavior |
|----------|----------|
| `file-wins` | File metadata overwrites database (default) |
| `database-wins` | Database metadata preserved, file unchanged |
| `newer-wins` | Most recently modified wins (file timestamp vs database updated_at) |

## Examples

### Import Existing Task Files

First-time import of existing markdown files:

```bash
# Preview what will be imported
shark sync --create-missing --dry-run

# Import tasks, creating epics/features as needed
shark sync --create-missing
```

### After Git Pull

Sync database after pulling changes from collaborators:

```bash
# Check what changed
git pull
shark sync --dry-run

# Apply changes
shark sync
```

### Sync Specific Folder

Sync only a specific feature folder:

```bash
shark sync --folder=docs/plan/E04-task-mgmt-cli/E04-F07-init-sync
```

### Force File to Overwrite Database

When file is authoritative source:

```bash
shark sync --strategy=file-wins
# or shorthand:
shark sync --force
```

### Preserve Database, Ignore File Changes

When database is authoritative:

```bash
shark sync --strategy=database-wins
```

### Use Timestamps to Resolve Conflicts

Most recent edit wins:

```bash
shark sync --strategy=newer-wins
```

### Clean Up Orphaned Tasks

Remove database tasks whose files no longer exist:

```bash
# Preview orphaned tasks
shark sync --cleanup --dry-run

# Delete orphaned tasks
shark sync --cleanup
```

## Frontmatter Format

### Required Fields

```yaml
---
key: T-E04-F07-001
---
```

### Optional Fields

```yaml
---
key: T-E04-F07-001
title: Implement sync engine
description: Create synchronization orchestration engine
---
```

### Fields NOT in Frontmatter

These fields are managed **exclusively** in the database:
- `status` - Use `shark task update-status` to change
- `priority` - Use `shark task update` to change
- `agent_type` - Set during task creation
- `depends_on` - Managed via CLI commands

### Complete Example

```markdown
---
key: T-E04-F07-001
title: Implement sync engine
description: Create the main synchronization engine that orchestrates file scanning, conflict detection, and database updates.
---

# Task: Implement sync engine

## Description

Implement the SyncEngine component that coordinates all sync operations...

## Acceptance Criteria

- [ ] Scan feature folders recursively
- [ ] Parse YAML frontmatter
- [ ] Detect conflicts between file and database
- [ ] Apply resolution strategy
```

## Understanding Conflicts

### What Causes Conflicts?

Conflicts occur when file metadata differs from database:

| Field | File | Database | Result |
|-------|------|----------|--------|
| title | "New Title" | "Old Title" | Conflict |
| description | "New desc" | "Old desc" | Conflict |
| file_path | `/new/path` | `/old/path` | Conflict |

### How Conflicts Are Resolved

**File-wins strategy** (default):
```
Database title: "Implement authentication"
File title:     "Add user authentication"
→ Database updated to: "Add user authentication"
```

**Database-wins strategy**:
```
Database title: "Implement authentication"
File title:     "Add user authentication"
→ Database unchanged: "Implement authentication"
```

**Newer-wins strategy**:
```
Database updated: 2025-12-15 10:00
File modified:    2025-12-16 09:00
→ File is newer, use file title
```

### Conflict Report

Sync displays conflicts clearly:

```
Conflict detected in T-E04-F07-001:
  Field: title
  Database: "Implement authentication"
  File: "Add user authentication"
  Resolution: file-wins (title updated to "Add user authentication")

Conflict detected in T-E04-F07-001:
  Field: description
  Database: "Add auth system"
  File: "Implement user authentication system"
  Resolution: file-wins (description updated)
```

## Sync Reports

### Human-Readable Output

```
Sync completed:
  Files scanned: 47
  New tasks imported: 5
  Existing tasks updated: 3
  Conflicts resolved: 2
  Warnings: 1
  Errors: 0

Details:
  - Imported: T-E04-F07-006, T-E04-F07-007 (2 new tasks)
  - Updated: T-E04-F07-001 (title changed)
  - Warning: Invalid frontmatter in docs/legacy/old-task.md
```

### JSON Output

```bash
shark sync --json
```

```json
{
  "files_scanned": 47,
  "tasks_imported": 5,
  "tasks_updated": 3,
  "conflicts_resolved": 2,
  "warnings": [
    "Invalid frontmatter in docs/legacy/old-task.md"
  ],
  "errors": []
}
```

## Common Workflows

### Workflow 1: First-Time Migration

Migrate existing task files to Shark CLI:

```bash
# 1. Initialize Shark CLI
shark init

# 2. Preview import
shark sync --create-missing --dry-run

# 3. Review output, then import
shark sync --create-missing

# 4. Verify
shark task list
```

### Workflow 2: Daily Development

Sync after git pull:

```bash
# 1. Pull changes from team
git pull origin main

# 2. Sync database
shark sync

# 3. Continue working
shark task list --status=todo
```

### Workflow 3: Resolve Conflicts

Handle conflicting changes:

```bash
# 1. Preview conflicts
shark sync --dry-run

# 2. Review conflict report

# 3. Choose resolution strategy
shark sync --strategy=file-wins    # Trust file
# or
shark sync --strategy=database-wins # Trust database
# or
shark sync --strategy=newer-wins    # Use timestamps
```

### Workflow 4: Clean Up Project

Remove orphaned tasks:

```bash
# 1. Preview orphaned tasks
shark sync --cleanup --dry-run

# 2. Review which tasks will be deleted

# 3. Clean up
shark sync --cleanup
```

## Error Handling

### Invalid YAML

**Error**: Invalid frontmatter in file

**Behavior**: File is skipped with warning, sync continues

```
Warning: Invalid frontmatter in docs/plan/E04-cli/E04-F07/broken.md
  Skipping file and continuing...
```

### Missing Required Field

**Error**: Frontmatter missing `key` field

**Behavior**: File is skipped with warning

```
Warning: Missing required field 'key' in docs/plan/E04-cli/E04-F07/invalid.md
  Skipping file and continuing...
```

### Missing Epic/Feature

**Error**: Task references non-existent epic/feature

**Without --create-missing**:
```
Warning: Task T-E99-F01-001 references non-existent feature E99-F01
  Skipping task. Use --create-missing to auto-create.
```

**With --create-missing**:
```
Info: Created epic E99 (inferred from task file)
Info: Created feature E99-F01 (inferred from task file)
Info: Imported task T-E99-F01-001
```

### Key Mismatch

**Error**: Filename doesn't match frontmatter key

```
Warning: Key mismatch in docs/plan/E04-cli/E04-F07/T-E04-F07-001.md
  Filename key: T-E04-F07-001
  Frontmatter key: T-E04-F07-002
  Using frontmatter key (T-E04-F07-002)
```

### Database Constraint Violation

**Error**: Duplicate key or foreign key violation

**Behavior**: Transaction rolled back, no partial updates

```
Error: Duplicate task key T-E04-F07-001
  Transaction rolled back. No changes made.
```

## Transaction Safety

All sync operations use database transactions:

```
BEGIN TRANSACTION
  - Import task 1
  - Import task 2
  - Update task 3
  - Resolve conflict 4
  [Error occurs here]
ROLLBACK  ← All changes undone
```

**Benefits**:
- All-or-nothing updates
- No partial data corruption
- Safe to retry after failures

## Performance

Sync is optimized for large codebases:

- **100 files in < 10 seconds** (PRD requirement)
- Bulk database operations (not one-by-one)
- Efficient file scanning (only task markdown files)
- Minimal memory footprint

### Performance Tips

1. **Use --folder for large repos**:
   ```bash
   shark sync --folder=docs/plan/E04-current-epic
   ```

2. **Batch sync after multiple pulls**:
   ```bash
   git pull --rebase
   shark sync  # Once after all merges
   ```

3. **Skip dry-run in automation**:
   ```bash
   shark sync --json  # No dry-run overhead
   ```

## Security

### Path Traversal Prevention

Sync validates all file paths:
- Rejects paths outside project root
- Rejects symlinks
- Rejects absolute paths in frontmatter

### File Size Limits

Protection against malicious files:
- Maximum file size: 1 MB
- Maximum frontmatter size: 100 KB

### SQL Injection Prevention

All database queries use parameterized statements (no SQL injection risk).

## Best Practices

### 1. Always Dry-Run First

```bash
shark sync --dry-run  # Preview changes
shark sync            # Apply if OK
```

### 2. Commit Before Sync

```bash
git add .
git commit -m "Before sync"
shark sync
```

### 3. Use Consistent Strategies

Pick one strategy for your team:
- **file-wins**: Files are source of truth (recommended)
- **database-wins**: CLI is source of truth
- **newer-wins**: Collaborative editing

### 4. Review Conflict Reports

Read conflict output before accepting changes:
```bash
shark sync --dry-run | grep "Conflict"
```

### 5. Keep Frontmatter Minimal

Only include: key, title, description
Avoid duplicating database-managed fields.

## Troubleshooting

See [Troubleshooting Guide](../troubleshooting.md) for common issues:
- Sync fails with transaction error
- Tasks not imported
- Conflicts not resolving
- Performance issues

## See Also

- [Initialization Guide](initialization.md) - Set up Shark CLI
- [Task Management](../CLI.md) - Create and manage tasks
- [Troubleshooting](../troubleshooting.md) - Common issues
