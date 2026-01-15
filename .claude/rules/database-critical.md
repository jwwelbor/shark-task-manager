# Database Management - CRITICAL WARNINGS

## ⚠️ DO NOT DELETE OR RECREATE THE DATABASE

The database file (`shark-tasks.db`) is the single source of truth for all project data. **Deleting it will cause data loss and sync errors.**

## What NOT To Do

❌ **DO NOT** run `make clean` during development (it deletes the database)
❌ **DO NOT** use `rm shark*` or glob patterns that match the database
❌ **DO NOT** delete the database to fix sync errors (fix the sync instead)
❌ **DO NOT** modify task files while running sync operations

## What To Do If Database Is Corrupted

If you need to reset the database:

1. **Backup first** (save the .db file elsewhere)
   ```bash
   cp shark-tasks.db shark-tasks.db.backup
   ```

2. **Delete ONLY the database file and WAL files:**
   ```bash
   rm shark-tasks.db shark-tasks.db-shm shark-tasks.db-wal
   ```

3. **Reinitialize:**
   ```bash
   ./bin/shark init --non-interactive
   ```

4. **Resync filesystem to database:**
   ```bash
   ./bin/shark sync --dry-run              # Preview changes
   ./bin/shark sync --strategy=file-wins   # Apply changes
   ```

## If Sync Fails with "UNIQUE constraint failed: tasks.key"

This means you're trying to create tasks that already exist. Options:

1. **Check if database exists:**
   ```bash
   ls -lh shark-tasks.db
   ```

2. **If database was deleted, restore from backup:**
   ```bash
   cp /path/to/backup/shark-tasks.db .
   ```

3. **If files are out of sync with database:**
   ```bash
   ./bin/shark sync --dry-run --strategy=database-wins  # Use DB as source of truth
   ```

4. **If specific tasks are duplicated**, manually remove duplicate file or reset task in database

## Database Files

The following files are database-related and should not be manually edited or deleted:

- `shark-tasks.db` - Main database file
- `shark-tasks.db-shm` - Shared memory file (WAL mode)
- `shark-tasks.db-wal` - Write-Ahead Log (WAL mode)

These files work together for SQLite's Write-Ahead Logging mode which enables better concurrency.

## Recovery from Accidental Deletion

If you accidentally deleted the database:

1. **Check for backups** in the project root:
   ```bash
   ls -la *.db.backup *.db.bak
   ```

2. **If no backup exists**, you can rebuild from filesystem:
   ```bash
   # Reinitialize database
   ./bin/shark init --non-interactive

   # Sync from filesystem
   ./bin/shark sync --strategy=file-wins --create-missing
   ```

   **Note**: This recovers task structure but loses task history and status transitions.

3. **Verify recovery:**
   ```bash
   ./bin/shark task list
   ./bin/shark epic list
   ```

## Prevention

- Add `shark-tasks.db` to important files for backups
- Consider using Turso cloud database for multi-machine sync and automatic backups
- Never use `make clean` during active development
- Be careful with shell globbing patterns like `rm shark*`
