# Database Management - CRITICAL WARNINGS

## ⚠️ MIGRATIONS AND skip_migrations FLAG

The project uses Turso cloud database with `"skip_migrations": true` in `.sharkconfig.json` to avoid ~2s DDL overhead on every command. **When you add a new migration, you MUST call it out explicitly** so the developer can temporarily enable migrations.

### When You Add a Migration (new table, column, view, index, constraint change):

1. **Tell the developer:** _"This change adds a migration. Set `skip_migrations: false` in `.sharkconfig.json` before running the next shark command, then set it back to `true`."_
2. **Also bump `CurrentSchemaVersion`** in `internal/db/db.go` — this ensures `ApplySchemaIfNeeded` re-runs on databases that already recorded the old version.
3. **After the migration runs once**, the developer should set `skip_migrations` back to `true`.

### Why This Matters

- `skip_migrations: true` causes `ApplySchemaIfNeeded` to check `CurrentSchemaVersion` and skip all DDL if the DB is up to date
- If you add a migration without bumping the version, the migration **never runs** on existing databases
- The `$SHARK_DB_BACKEND=turso` environment variable activates this path; local SQLite always runs migrations

### Checklist When Adding Any Migration

- [ ] Added migration function in `internal/db/db.go`
- [ ] Called migration from `runMigrations()`
- [ ] Bumped `CurrentSchemaVersion` constant (e.g., 2 → 3)
- [ ] Told the developer to set `skip_migrations: false` in `.sharkconfig.json` and run one shark command
- [ ] Developer resets `skip_migrations: true` after migration applies

---

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

4. **Database is now empty** - create new epics/features/tasks as needed

## Database Recovery

If you accidentally deleted the database, there is no automatic sync from files.

Options:

1. **Restore from backup:**
   ```bash
   cp /path/to/backup/shark-tasks.db .
   ```

2. **If specific tasks are duplicated**, manually remove duplicate file or reset task in database

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

2. **If no backup exists**, you must recreate tasks manually:
   ```bash
   # Reinitialize database
   ./bin/shark init --non-interactive

   # Recreate epics, features, and tasks from task files
   # (Database is source of truth - task files are output only)
   ```

   **Note**: There is no automatic filesystem-to-database sync. You must recreate entities manually.

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
