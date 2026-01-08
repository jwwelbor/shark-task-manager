# Migrating from Local to Turso Cloud Database

This guide explains how to migrate your existing local Shark database to Turso cloud database.

## Overview

Shark supports two database backends:
- **Local**: SQLite file (`shark-tasks.db`) stored on your machine
- **Turso**: Cloud-hosted SQLite accessible from multiple machines

This guide covers migrating from local to cloud while preserving all your data.

## Migration Strategies

### Strategy 1: Export/Import via SQL (Recommended)

This is the most reliable method for migrating data.

#### Step 1: Export Local Database

```bash
# Export all data from local database
sqlite3 shark-tasks.db .dump > shark-backup.sql
```

#### Step 2: Set Up Turso Database

Follow the [Turso Quick Start Guide](./TURSO_QUICKSTART.md) to create and configure your Turso database:

```bash
# Create Turso database
turso db create shark-tasks

# Get URL and create token
turso db show shark-tasks --url
turso db tokens create shark-tasks

# Configure Shark
shark cloud init \
  --url="libsql://shark-tasks-yourorg.turso.io" \
  --auth-token="<your-token>" \
  --non-interactive
```

#### Step 3: Import Data to Turso

```bash
# Use Turso CLI to import SQL dump
turso db shell shark-tasks < shark-backup.sql
```

#### Step 4: Verify Migration

```bash
# Check that data migrated successfully
shark task list
shark epic list
shark feature list

# Verify counts match
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM tasks;"
# Compare with cloud
shark task list --json | jq 'length'
```

### Strategy 2: Manual Configuration Switch

If you have minimal data or want to start fresh with Turso:

#### Option A: Start Fresh

```bash
# Backup local database (just in case)
cp shark-tasks.db shark-tasks.db.backup

# Configure Turso
shark cloud init --url="libsql://..." --auth-token="..." --non-interactive

# Initialize fresh schema
shark init --non-interactive

# Manually recreate epics/features/tasks (if needed)
```

#### Option B: Dual Operation

Keep both databases and choose which to use:

```bash
# Local database
shark --db=./shark-tasks.db task list

# Cloud database (via config)
shark task list

# Or create separate config files
shark --config=.sharkconfig-local.json task list
shark --config=.sharkconfig-cloud.json task list
```

## Post-Migration Steps

### 1. Verify Data Integrity

```bash
# Check all entities migrated
shark epic list
shark feature list
shark task list

# Verify relationships
shark task list E01 F01  # Check tasks link to features correctly
shark feature get E01-F01  # Check feature progress calculation
```

### 2. Test Operations

```bash
# Create new task
shark task create E01 F01 "Test task" --agent=backend

# Update task status
shark task start E01-F01-001

# Complete task
shark task complete E01-F01-001

# Verify from another machine (if multi-machine setup)
```

### 3. Clean Up Local Database (Optional)

```bash
# Archive local database
mv shark-tasks.db shark-tasks.db.archived

# Remove from git tracking (if tracked)
echo "shark-tasks.db.archived" >> .gitignore
```

## Multi-Machine Migration

If you use Shark on multiple machines:

### Primary Machine (with data)

```bash
# 1. Export data
sqlite3 shark-tasks.db .dump > shark-backup.sql

# 2. Configure Turso
shark cloud init --url="libsql://..." --auth-token="..." --non-interactive

# 3. Import to Turso
turso db shell shark-tasks < shark-backup.sql

# 4. Verify
shark task list
```

### Secondary Machines (empty or with old data)

```bash
# 1. Backup local database (if exists)
cp shark-tasks.db shark-tasks.db.backup

# 2. Configure Turso with same credentials
shark cloud init --url="libsql://..." --auth-token="..." --non-interactive

# 3. Verify data is accessible
shark task list  # Should show data from primary machine
```

## Configuration File Changes

Before migration (`.sharkconfig.json`):
```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}
```

After migration (`.sharkconfig.json`):
```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-yourorg.turso.io",
    "auth_token_file": "/home/user/.turso/shark-token"
  }
}
```

## Rollback Plan

If you need to revert to local database:

```bash
# 1. Restore local database from backup
cp shark-tasks.db.backup shark-tasks.db

# 2. Update config to use local
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}

# Or delete config to use defaults
rm .sharkconfig.json

# 3. Verify
shark task list
```

## Troubleshooting

### Import Fails with "table already exists"

The Turso database might already have schema initialized:

```bash
# Option 1: Drop and recreate database
turso db destroy shark-tasks
turso db create shark-tasks

# Option 2: Export data only (without schema)
sqlite3 shark-tasks.db << 'EOF' > shark-data-only.sql
.mode insert epics
SELECT * FROM epics;
.mode insert features
SELECT * FROM features;
.mode insert tasks
SELECT * FROM tasks;
.mode insert task_history
SELECT * FROM task_history;
EOF

# Import data only
turso db shell shark-tasks < shark-data-only.sql
```

### Data Missing After Migration

```bash
# Verify export succeeded
wc -l shark-backup.sql  # Should have many lines

# Check for errors in SQL dump
grep -i "error\|failed" shark-backup.sql

# Re-export with verbose output
sqlite3 shark-tasks.db .dump | tee shark-backup.sql
```

### Performance Issues After Migration

```bash
# Turso may need indexes rebuilt
turso db shell shark-tasks << 'EOF'
ANALYZE;
REINDEX;
EOF
```

### Auth Token Expired After Migration

```bash
# Create new token
turso db tokens create shark-tasks

# Update config
shark cloud init --url="libsql://..." --auth-token="<new-token>" --non-interactive
```

## Best Practices

1. **Always backup** before migration: `cp shark-tasks.db shark-tasks.db.backup`
2. **Test on non-production data** first if possible
3. **Verify data integrity** after migration
4. **Keep local backup** for at least a week after successful migration
5. **Use token files** (`--auth-file`) instead of storing tokens directly in config
6. **Document your migration** date and process for team reference

## Advanced: Selective Migration

To migrate only specific epics or features:

```bash
# Export specific epic
sqlite3 shark-tasks.db << 'EOF' > E01-export.sql
.mode insert epics
SELECT * FROM epics WHERE key = 'E01';
.mode insert features
SELECT * FROM features WHERE epic_id = (SELECT id FROM epics WHERE key = 'E01');
.mode insert tasks
SELECT * FROM tasks WHERE epic_id = (SELECT id FROM epics WHERE key = 'E01');
EOF

# Import to Turso
turso db shell shark-tasks < E01-export.sql
```

## Next Steps

- **Test cloud operations** with `shark cloud status`
- **Set up other machines** following [Turso Quick Start](./TURSO_QUICKSTART.md)
- **Read [CLI Reference](./CLI_REFERENCE.md)** for all cloud commands
- **Update team documentation** with your Turso URL and setup instructions

## Resources

- [SQLite Dump Documentation](https://www.sqlite.org/cli.html#dump)
- [Turso DB Shell](https://docs.turso.tech/reference/turso-cli#db-shell)
- [Shark Database Abstraction](../CLAUDE.md#database-abstraction)
