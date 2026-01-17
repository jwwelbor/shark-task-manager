# Sync Commands

Synchronize markdown files with SQLite database.

## `shark sync`

**Flags:**
- `--dry-run`: Preview changes without applying them
- `--strategy <strategy>`: Conflict resolution strategy
  - `file-wins`: File system is source of truth
  - `database-wins`: Database is source of truth
  - `newer-wins`: Most recently modified wins
- `--create-missing`: Create missing epics/features from files
- `--cleanup`: Delete orphaned database records (files deleted)
- `--pattern <type>`: Sync specific pattern (`task`, `prp`)
- `--folder <path>`: Sync specific folder only
- `--json`: Output in JSON format

**Examples:**

```bash
# Preview sync changes
shark sync --dry-run --json

# Sync with file system as source of truth
shark sync --strategy=file-wins

# Sync with database as source of truth
shark sync --strategy=database-wins

# Sync with newest modification wins
shark sync --strategy=newer-wins

# Create missing epics/features
shark sync --create-missing

# Delete orphaned records
shark sync --cleanup

# Sync specific folder
shark sync --folder=docs/plan/E07-user-management-system

# Sync only task files
shark sync --pattern=task

# Sync only PRP files
shark sync --pattern=prp

# Sync both task and PRP files
shark sync --pattern=task --pattern=prp
```

## Important Notes

⚠️ **Status is managed exclusively in the database and is NOT synced from files**

This ensures:
- Atomic status transitions
- Complete audit trails via task_history
- Consistent workflow enforcement

## When to Use Sync

Run `shark sync` after:
- Git pull operations
- Branch switches
- Manual file edits
- File system reorganization

## Related Documentation

- [Best Practices](best-practices.md) - Database sync best practices
- [File Paths](file-paths.md) - File organization structure
