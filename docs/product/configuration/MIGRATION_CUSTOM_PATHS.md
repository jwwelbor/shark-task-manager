# Migration Guide: Custom Folder Base Paths Feature

## Overview

This guide helps you understand and implement the custom folder base paths feature for epics and features, which allows flexible organization of project documentation outside the default `docs/plan/` directory structure.

## What Changed

### Database Schema Updates

Two new columns have been added to support custom folder paths:

```sql
ALTER TABLE epics ADD COLUMN custom_folder_path TEXT;
ALTER TABLE features ADD COLUMN custom_folder_path TEXT;
```

These columns store relative paths to custom folder locations for organizing epics and features.

### New Indexes

For performance optimization:

```sql
CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);
CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);
```

### Command-Line Interface

New `--path` flag added to `epic create` and `feature create` commands:

```bash
shark epic create "Q1 2025 Roadmap" --path="docs/roadmap/2025-q1"
shark feature create --epic=E01 "User Growth" --path="docs/roadmap/2025-q1/user-growth"
```

## Backward Compatibility

**Good news:** This feature is 100% backward compatible!

- Existing databases will automatically apply migrations when running commands
- Existing epics and features without custom paths continue to work unchanged
- Default behavior (`docs/plan/{epic-key}/`) remains unchanged
- The `custom_folder_path` column defaults to NULL for existing records
- No action required for existing projects

## Migration for Existing Installations

If you're using Shark Task Manager and want to apply the database schema updates, follow these steps:

### Option 1: Automatic Migration (Recommended)

The database schema is automatically updated when you run any Shark command:

```bash
# Simply run any command - migrations apply automatically
shark epic list
# or
shark task list
```

The migrations check if columns already exist before adding them, so it's safe to run multiple times.

### Option 2: Manual Migration

If you want to manually apply the migrations to your database:

```sql
-- Add custom_folder_path column to epics table
ALTER TABLE epics ADD COLUMN custom_folder_path TEXT;
CREATE INDEX IF NOT EXISTS idx_epics_custom_folder_path ON epics(custom_folder_path);

-- Add custom_folder_path column to features table
ALTER TABLE features ADD COLUMN custom_folder_path TEXT;
CREATE INDEX IF NOT EXISTS idx_features_custom_folder_path ON features(custom_folder_path);
```

Run these commands in your SQLite client:

```bash
# Open SQLite CLI
sqlite3 shark-tasks.db

# Paste the SQL commands above
# Press Ctrl+D to exit
```

### Verification

Verify the migration was successful:

```bash
# Query the database to check columns exist
sqlite3 shark-tasks.db ".schema epics" | grep custom_folder_path
sqlite3 shark-tasks.db ".schema features" | grep custom_folder_path

# Should output:
# custom_folder_path TEXT,
```

## Using Custom Folder Paths

### Basic Usage

Create an epic with a custom folder path:

```bash
shark epic create "Q1 2025 Roadmap" --path="docs/roadmap/2025-q1"
```

This organizes the epic in a custom location while maintaining backward compatibility.

### Organizational Patterns

**By Time Period:**

```bash
shark epic create "Q1 2025" --path="docs/roadmap/2025-q1"
shark epic create "Q2 2025" --path="docs/roadmap/2025-q2"
```

**By Business Domain:**

```bash
shark epic create "Core Product" --path="docs/product/core"
shark epic create "Platform Services" --path="docs/platform/services"
```

**By Team:**

```bash
shark epic create "Mobile Team OKRs" --path="docs/teams/mobile"
shark epic create "Backend Team OKRs" --path="docs/teams/backend"
```

**By Project Type:**

```bash
shark epic create "Customer-Facing Features" --path="docs/features/customer"
shark epic create "Internal Infrastructure" --path="docs/infrastructure/internal"
```

### Feature Path Inheritance

Features created under an epic inherit the epic's custom folder path:

```bash
# Create epic with custom path
shark epic create "Mobile Initiative" --path="docs/mobile/2025"

# Create features - they inherit the epic's path
shark feature create --epic=E01 "iOS App"        # Stored in docs/mobile/2025/
shark feature create --epic=E01 "Android App"   # Stored in docs/mobile/2025/

# Feature can override the inherited path
shark feature create --epic=E01 "Web App" --path="docs/web/2025"  # Stored in docs/web/2025/
```

### Path Resolution Priority

When creating features or tasks, paths are resolved in this order:

1. **Explicit `--filename`** (highest priority)
   ```bash
   shark feature create --epic=E01 "Auth" --filename="docs/specs/auth.md"
   # Always stored at docs/specs/auth.md (overrides everything)
   ```

2. **Explicit `--path`**
   ```bash
   shark feature create --epic=E01 "API" --path="docs/api"
   # Stored in custom path
   ```

3. **Inherited from parent**
   ```bash
   # Epic has --path="docs/roadmap"
   shark feature create --epic=E01 "Feature 1"
   # Inherits docs/roadmap/ from epic
   ```

4. **Default location**
   ```bash
   shark feature create --epic=E01 "Feature 2"
   # No custom paths - uses docs/plan/E01-feature-2/
   ```

## File System Synchronization

### Sync with Custom Paths

The `shark sync` command properly handles custom folder paths:

```bash
# Sync discovers epics and features in custom locations
shark sync

# Dry-run shows what will be synced
shark sync --dry-run

# File vs DB conflicts resolved with strategy
shark sync --strategy=file-wins
```

### Discovery Process

Sync automatically discovers:
- Epics with `custom_folder_path` set in database
- Features with inherited or explicit custom paths
- Tasks in custom folder locations
- Metadata in YAML frontmatter

### Handling Custom Paths During Sync

The sync engine:

1. **Reads database records** with custom_folder_path values
2. **Discovers files** in custom locations
3. **Matches records** to files by key
4. **Updates database** with file paths
5. **Handles conflicts** using your chosen strategy

**Example:**

```bash
# Database has epic E01 with custom_folder_path="docs/roadmap"
# Epic file exists at docs/roadmap/E01-initiative/epic.md

shark sync
# Discovers epic.md in docs/roadmap/
# Matches to E01 record
# Updates file_path in database
# Syncs feature and task files
```

## Migration Checklist

- [ ] Verify database file location (default: `shark-tasks.db`)
- [ ] Back up your database: `cp shark-tasks.db shark-tasks.db.backup`
- [ ] Run any Shark command to apply migrations automatically
- [ ] Verify columns were added: `sqlite3 shark-tasks.db ".schema epics"`
- [ ] Test creating an epic with `--path` flag
- [ ] Test sync with custom paths: `shark sync --dry-run`
- [ ] Verify file organization in docs directory

## Rollback (If Needed)

If you need to rollback the feature:

### Remove Custom Paths from Database

```bash
sqlite3 shark-tasks.db << 'EOF'
UPDATE epics SET custom_folder_path = NULL;
UPDATE features SET custom_folder_path = NULL;
EOF
```

### Restore from Backup

```bash
# If migration caused issues
cp shark-tasks.db.backup shark-tasks.db
```

## Troubleshooting

### "no such column: custom_folder_path" Error

**Cause:** Migration didn't apply

**Solution:**
```bash
# Trigger migration by running any command
shark epic list

# Or manually run migrations:
sqlite3 shark-tasks.db < migration_commands.sql
```

### Custom Paths Not Working

**Check database:**
```bash
sqlite3 shark-tasks.db "SELECT key, custom_folder_path FROM epics LIMIT 5;"
```

**Check command:**
```bash
# Use --path flag correctly
shark epic create "Test" --path="docs/test"

# Verify it was stored
shark epic get E01 --json | grep custom_folder_path
```

### Sync Can't Find Custom Path Files

**Ensure sync strategy is correct:**
```bash
# Try file-wins strategy
shark sync --strategy=file-wins --dry-run

# Check what files sync finds
shark sync --dry-run --verbose
```

## Performance Considerations

- Indexes on `custom_folder_path` optimize queries
- NULL values (default) don't impact performance
- Sync performance depends on file count, not path count
- No performance regression for projects not using custom paths

## FAQ

**Q: Do I need to migrate existing projects?**
A: No. The feature is backward compatible. Existing projects work unchanged.

**Q: Can I change a custom folder path later?**
A: Yes, update the epic/feature with a new `--path` value. Files aren't automatically moved.

**Q: Do tasks inherit custom paths?**
A: Yes. Tasks inherit the feature's path, which inherits from the epic.

**Q: Can I mix default and custom paths?**
A: Yes! You can have some epics with custom paths and others in `docs/plan/`.

**Q: What if I have path conflicts?**
A: Use `shark sync --strategy=file-wins` or `--strategy=database-wins` to resolve.

**Q: How are paths stored?**
A: Relative to project root, no leading slashes, no trailing slashes.

**Q: Can I use absolute paths?**
A: No. Paths must be relative to project root for security.

## Resources

- [CLI Reference](./CLI_REFERENCE.md) - Full command documentation
- [Synchronization Guide](./user-guide/synchronization.md) - Sync behavior details
- [Initialization Guide](./user-guide/initialization.md) - Setup instructions

## Support

For issues or questions:

1. Check the [troubleshooting](#troubleshooting) section above
2. Review [CLI_REFERENCE.md](./CLI_REFERENCE.md) for command syntax
3. Run `shark --help` for command help
4. Report bugs on GitHub with your database backup (remove sensitive data)
