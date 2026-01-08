# Migration Guide: Custom Folder Paths (DEPRECATED)

> **DEPRECATION NOTICE**
> As of version E07-F19, the custom folder path feature (`custom_folder_path` database columns) has been **removed** in favor of a simplified `file_path` architecture.
>
> This document is maintained for historical reference and to assist users upgrading from older versions.

---

## What Changed

### Before (E07-F18 and earlier)

The system used two separate concepts:
- `custom_folder_path`: Base directory for organizing epics/features
- `file_path`: Complete file path including filename

**Database Schema (Old):**
```sql
CREATE TABLE epics (
    id INTEGER PRIMARY KEY,
    key TEXT UNIQUE NOT NULL,
    custom_folder_path TEXT,  -- Removed
    file_path TEXT,
    ...
);

CREATE TABLE features (
    id INTEGER PRIMARY KEY,
    key TEXT UNIQUE NOT NULL,
    custom_folder_path TEXT,  -- Removed
    file_path TEXT,
    ...
);
```

**CLI (Old):**
```bash
# Old: --path meant "custom folder base path"
shark epic create "Q1 Roadmap" --path="docs/roadmap/2025-q1"

# Old: Features inherited custom_folder_path from parent epic
shark feature create --epic=E01 "User Growth"
```

### After (E07-F19 and later)

The system now uses a single, simple concept:
- `file_path`: Complete file path for each entity

**Database Schema (New):**
```sql
CREATE TABLE epics (
    id INTEGER PRIMARY KEY,
    key TEXT UNIQUE NOT NULL,
    file_path TEXT,  -- Now the only path field
    ...
);

CREATE TABLE features (
    id INTEGER PRIMARY KEY,
    key TEXT UNIQUE NOT NULL,
    file_path TEXT,  -- Now the only path field
    ...
);
```

**CLI (New):**
```bash
# New: --file specifies complete file path
shark epic create "Q1 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# New: No inheritance - each entity has explicit file_path
shark feature create --epic=E01 "User Growth" --file="docs/roadmap/features/growth.md"

# Aliases for backward compatibility (hidden)
shark epic create "Q1 Roadmap" --path="docs/roadmap/2025-q1/epic.md"  # --path is alias for --file
```

---

## 🚀 Quick Migration Path

> **📖 For detailed migration instructions with detection tools, see:**
> **[E07-F19 Migration Guide](E07-F19-MIGRATION-GUIDE.md)**

### Detection Tools Available

Two scripts are provided to help detect and fix migration issues:

1. **Pre-Migration Detection** (`scripts/detect-custom-folder-paths.sh`)
   - Run BEFORE upgrading to identify Case 3 instances
   - Automatically generates fix scripts if needed
   - Safe to run multiple times

2. **Post-Migration Validation** (`scripts/validate-file-paths.sh`)
   - Run AFTER upgrading to verify file paths
   - Detects mismatches between database and filesystem
   - Provides fix recommendations

---

## Migration Instructions

### For Users Upgrading from E07-F18 or Earlier

If you have been using the `custom_folder_path` feature, follow these steps to migrate:

#### Step 1: Pre-Migration Check (RECOMMENDED)

**Before upgrading**, detect if you have any Case 3 instances:

```bash
# Run pre-migration detection
./scripts/detect-custom-folder-paths.sh

# If Case 3 instances found, generate fix script
./scripts/detect-custom-folder-paths.sh shark-tasks.db --generate-fix-script > fix-custom-paths.sql

# Review and apply fixes
cat fix-custom-paths.sql
sqlite3 shark-tasks.db < fix-custom-paths.sql
```

#### Step 2: Backup Your Database

Before making any changes, create a backup:

```bash
cp shark-tasks.db shark-tasks.db.backup
cp shark-tasks.db-wal shark-tasks.db-wal.backup
cp shark-tasks.db-shm shark-tasks.db-shm.backup
```

#### Step 3: Update Shark CLI

Update to the latest version:

```bash
# Homebrew
brew upgrade shark

# Scoop
scoop update shark

# Manual installation
# Download latest release from GitHub
```

#### Step 4: Run Database Migration

The database migration runs automatically on first use. Verify the migration:

```bash
# Check database schema
sqlite3 shark-tasks.db ".schema epics"
sqlite3 shark-tasks.db ".schema features"

# Verify custom_folder_path columns are removed (should not appear in output)
```

#### Step 5: Post-Migration Validation (RECOMMENDED)

**After upgrading**, verify all file paths:

```bash
# Run post-migration validation
./scripts/validate-file-paths.sh

# If mismatches found, fix using sync
shark sync --dry-run              # Preview changes
shark sync --strategy=file-wins   # Apply fixes
```

#### Step 6: Update Your Workflow

**Old Workflow:**
```bash
# Epic with custom folder base path
shark epic create "Q1 Roadmap" --path="docs/roadmap/2025-q1"

# Feature inherits epic's custom_folder_path
shark feature create --epic=E01 "User Growth"
```

**New Workflow:**
```bash
# Epic with explicit file path
shark epic create "Q1 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# Feature with explicit file path (no inheritance)
shark feature create --epic=E01 "User Growth" --file="docs/roadmap/2025-q1/features/user-growth.md"
```

#### Step 7: Verify File Paths (Optional)

After migration, verify that all file paths are correct:

```bash
# List all epics and check file_path
shark epic list --json | jq '.[] | {key, file_path}'

# List all features and check file_path
shark feature list --json | jq '.[] | {key, file_path}'

# List all tasks and check file_path
shark task list --json | jq '.[] | {key, file_path}'
```

---

## What Happened to My Data?

### Database Migration Details

The migration (E07-F19) performed the following operations:

1. **Removed columns:**
   - `epics.custom_folder_path`
   - `features.custom_folder_path`

2. **Retained data:**
   - All existing `file_path` values remain unchanged
   - No epics, features, or tasks were deleted
   - All other columns remain intact

3. **Indexes removed:**
   - `idx_epics_custom_folder_path`
   - `idx_features_custom_folder_path`

### File System Changes

**No file system changes were made.** All your markdown files remain in their original locations.

The migration only affected the database schema, not your actual files.

---

## Frequently Asked Questions

### Q: Will this break my existing project?

**A:** No. The migration is backward compatible. All existing file paths are preserved, and the CLI continues to work with your existing project structure.

### Q: What happens to epics/features created with --path in older versions?

**A:** Their `file_path` values are unchanged. The migration only removes the `custom_folder_path` columns, which were used for inheritance logic. Since `file_path` always took precedence, your files remain exactly where they were.

### Q: Can I still use --path flag?

**A:** Yes, but it now works differently:
- **Old behavior:** `--path` set a base folder, filename was auto-generated
- **New behavior:** `--path` is a hidden alias for `--file`, expects complete path including `.md` extension

For clarity, we recommend using `--file` instead of `--path` in new commands.

### Q: How do I organize epics by time period now?

**Old way (removed):**
```bash
shark epic create "Q1 2025" --path="docs/roadmap/2025-q1"
# Auto-created: docs/roadmap/2025-q1/epic.md

shark feature create --epic=E01 "Feature A"
# Inherited path, auto-created: docs/roadmap/2025-q1/E01-F01-feature-a/feature.md
```

**New way (current):**
```bash
# Specify complete file path for epic
shark epic create "Q1 2025" --file="docs/roadmap/2025-q1/epic.md"

# Specify complete file path for feature (no inheritance)
shark feature create --epic=E01 "Feature A" --file="docs/roadmap/2025-q1/features/feature-a.md"
```

### Q: Why was this feature removed?

**A:** The dual-concept architecture (`custom_folder_path` + `file_path`) added complexity without significant benefits:
- Confusing path resolution rules
- Hidden inheritance behavior
- Overlapping flags with different semantics
- Database schema complexity

The simplified `file_path`-only architecture is:
- Easier to understand and maintain
- More explicit (no hidden inheritance)
- Flexible (you can still organize however you want)
- Simpler codebase

---

## Rollback Instructions

If you need to rollback to the previous version:

### Step 1: Restore Database Backup

```bash
cp shark-tasks.db.backup shark-tasks.db
```

### Step 2: Downgrade Shark CLI

**Homebrew:**
```bash
brew uninstall shark
brew install shark@E07-F18  # Install previous version
```

**Manual:**
Download and install E07-F18 release from GitHub.

### Step 3: Verify

```bash
shark --version
# Should show E07-F18 or earlier

sqlite3 shark-tasks.db ".schema epics" | grep custom_folder_path
# Should show custom_folder_path column
```

---

## Support

If you encounter issues during migration:

1. **Check database backup exists:** Ensure you have `shark-tasks.db.backup`
2. **Verify file paths:** Run `shark epic list --json` and check all `file_path` values
3. **Review migration logs:** Check CLI output for any migration errors
4. **Report issues:** Open a GitHub issue with migration details

---

## Related Documentation

- [CLI Reference](CLI_REFERENCE.md) - Complete command reference with new `--file` flag
- [CLAUDE.md](../CLAUDE.md) - Development guidelines and project overview
- [README.md](../README.md) - Project introduction

---

## Historical Context

The `custom_folder_path` feature was introduced in E07-F18 to support flexible project organization patterns. It allowed:

- Setting a base folder for epics: `--path="docs/roadmap/2025-q1"`
- Automatic inheritance by child features
- Mixed organization strategies within a single project

However, user feedback and code analysis revealed:

- **Confusion:** Two path concepts (`custom_folder_path` vs `file_path`) were hard to explain
- **Complexity:** Path resolution logic had 4 priority levels
- **Hidden behavior:** Inheritance wasn't visible in command syntax
- **Maintenance burden:** Extra database columns, indexes, migration logic

E07-F19 simplified to a single `file_path` concept, removing the inheritance complexity while retaining full organizational flexibility.

**Example - Same Organization, Simpler Approach:**

**Old (E07-F18):**
```bash
# Set base folder, inherits to features
shark epic create "Q1" --path="docs/roadmap/2025-q1"
shark feature create --epic=E01 "Feature A"  # Inherits path implicitly
```

**New (E07-F19):**
```bash
# Explicit file paths (no inheritance)
shark epic create "Q1" --file="docs/roadmap/2025-q1/epic.md"
shark feature create --epic=E01 "Feature A" --file="docs/roadmap/2025-q1/features/a.md"
```

Same file structure, but explicit and clear.

---

**Last Updated:** 2026-01-02
**Deprecated In:** E07-F19 (Database Migration and File Path Standardization)
