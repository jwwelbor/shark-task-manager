# Database Management - CRITICAL WARNINGS

## ⚠️ MIGRATIONS AND skip_migrations FLAG

The project uses Turso cloud database with `"skip_migrations": true` in `.sharkconfig.json` to avoid ~2s DDL overhead on every command. **Bumping `CurrentSchemaVersion` is the only required step to make a new migration run on existing databases.**

### How it works

When `skip_migrations: true`, the entry point calls `ApplySchemaIfNeeded` (in `internal/db/db.go`). That function reads the stored `schema_version` and compares it to `CurrentSchemaVersion`:

- If the stored version is **behind** `CurrentSchemaVersion`, it calls `ApplySchemaAndMigrations` — running all migrations including your new one.
- If the stored version is **equal or ahead**, it returns early (no DDL executed).

Bumping `CurrentSchemaVersion` (e.g., 13 → 14) is therefore **both necessary and sufficient** — the toggle is not a required per-migration step.

### When You Add a Migration (new table, column, view, index, constraint change):

1. Add the migration function in `internal/db/db.go` and call it from `runMigrations()`.
2. **Bump `CurrentSchemaVersion`** in `internal/db/db.go` (e.g., 13 → 14). This is the only step required — `ApplySchemaIfNeeded` will automatically detect the version gap and run your migration on the next shark command, regardless of `skip_migrations`.
3. _(Optional belt-and-braces)_ If you suspect a previous migration shipped without bumping the version and you need to force a full reapply, set `skip_migrations: false` in `.sharkconfig.json` for one run, then reset it to `true`. This forces `ApplySchemaAndMigrations` unconditionally, bypassing the version check entirely.

### Why the toggle exists (but is not required per-migration)

- `skip_migrations: false` bypasses `ApplySchemaIfNeeded` and calls `ApplySchemaAndMigrations` directly — useful as a manual fallback when the version bump was accidentally omitted in a prior release.
- All migrations in this codebase use `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, and `CREATE TRIGGER IF NOT EXISTS`, making them safe to rerun.
- The `$SHARK_DB_BACKEND=turso` environment variable activates this path; local SQLite always runs `ApplySchemaAndMigrations` unconditionally.

### Checklist When Adding Any Migration

- [ ] Added migration function in `internal/db/db.go`
- [ ] Called migration from `runMigrations()`
- [ ] Bumped `CurrentSchemaVersion` constant (e.g., 13 → 14) — **this is the key step**
- [ ] (Optional) Set `skip_migrations: false` for one run if you need to force a reapply, then reset to `true`

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
