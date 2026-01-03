# E07-F19 Migration Guide: File Path Flag Standardization

## Overview

E07-F19 standardizes file path handling by removing the `custom_folder_path` columns from the `epics` and `features` tables. This guide helps you detect and fix any issues before or after upgrading.

## What Changed

**Removed:**
- `epics.custom_folder_path` column
- `features.custom_folder_path` column
- `--path` flag support (replaced by `--file`)

**Preserved:**
- All `file_path` values remain unchanged
- All epic, feature, and task data remains intact
- `--file` / `--filename` / `--filepath` flags continue to work

## Impact by Use Case

### Case 1: Default Paths Only (No Impact)

**Before:**
```bash
shark epic create "My Epic"
# File: docs/plan/E07-my-epic/epic.md
# Database: file_path = NULL
```

**After Migration:**
✅ No change - works exactly the same

---

### Case 2: Explicit File Paths (No Impact)

**Before:**
```bash
shark epic create "Q1" --filename="docs/roadmap/q1.md"
# File: docs/roadmap/q1.md
# Database: file_path = "docs/roadmap/q1.md"
```

**After Migration:**
✅ No change - works exactly the same

---

### Case 3: Custom Folder Base Paths (⚠️ Requires Action)

**Before:**
```bash
shark epic create "Q1" --path="docs/roadmap/2025-q1"
# File: docs/roadmap/2025-q1/epic.md
# Database: custom_folder_path = "docs/roadmap/2025-q1", file_path = NULL
```

**After Migration:**
⚠️ **ACTION REQUIRED**
- Migration removes `custom_folder_path` column
- `file_path` remains `NULL`
- System expects file at default location: `docs/plan/E07-q1/epic.md`
- But file is actually at: `docs/roadmap/2025-q1/epic.md`

## Detection Tools

### Pre-Migration Detection (Before Upgrading)

**Purpose:** Identify epics/features using `custom_folder_path` before upgrading to E07-F19.

**Usage:**
```bash
# Basic check
./scripts/detect-custom-folder-paths.sh

# Specify database path
./scripts/detect-custom-folder-paths.sh path/to/shark-tasks.db

# Generate fix script
./scripts/detect-custom-folder-paths.sh shark-tasks.db --generate-fix-script > fix-custom-paths.sql
```

**Output:**
```
=========================================
E07-F19 Pre-Migration Detection
=========================================
Checking for custom_folder_path usage...

⚠️  Found epics using custom_folder_path (Case 3):
key   title           file_path  custom_folder_path
----  --------------  ---------  ---------------------------
E01   Q1 2025 Roadmap NULL       docs/roadmap/2025-q1

=========================================
ACTION REQUIRED BEFORE UPGRADE
=========================================
```

**What to Do:**
1. Run the detection script
2. If Case 3 instances found:
   - Option A: Update `file_path` in database (see "Fixing Case 3" below)
   - Option B: Wait and use `shark sync` after upgrade
3. If no Case 3 instances: Safe to upgrade

---

### Post-Migration Validation (After Upgrading)

**Purpose:** Detect path mismatches between database and filesystem after migration.

**Usage:**
```bash
# Basic validation
./scripts/validate-file-paths.sh

# Specify database and docs root
./scripts/validate-file-paths.sh shark-tasks.db docs/plan
```

**Output:**
```
=========================================
E07-F19 Post-Migration Validation
=========================================
Checking for file path mismatches...

⚠️  Epic E01: file_path is NULL, expected default at docs/plan/E01-q1-2025-roadmap/epic.md (MISSING)

=========================================
⚠️  VALIDATION FAILED
=========================================
Found 1 path mismatch(es).
```

**What to Do:**
1. Run validation script
2. If mismatches found, see "Fixing Case 3" below
3. Re-run validation after fixes

---

## Fixing Case 3 Issues

### Option A: Update Database to Point to Actual Files

Update the `file_path` column to point to where files actually exist:

```bash
# Update epic
shark epic update E01 --file="docs/roadmap/2025-q1/epic.md"

# Update feature
shark feature update E01-F01 --file="docs/roadmap/2025-q1/features/feature.md"
```

**SQL Equivalent:**
```sql
UPDATE epics SET file_path = 'docs/roadmap/2025-q1/epic.md' WHERE key = 'E01';
UPDATE features SET file_path = 'docs/roadmap/2025-q1/features/feature.md' WHERE key = 'E01-F01';
```

**When to Use:**
- You want to keep files in their current custom locations
- Files are well-organized in custom structure
- You don't want to reorganize project structure

---

### Option B: Move Files to Expected Default Locations

Move files to where the system expects them:

```bash
# Create default directories
mkdir -p docs/plan/E01-q1-2025-roadmap

# Move epic file
mv docs/roadmap/2025-q1/epic.md docs/plan/E01-q1-2025-roadmap/epic.md

# Move feature file
mkdir -p docs/plan/E01-F01-feature-name
mv docs/roadmap/2025-q1/features/feature.md docs/plan/E01-F01-feature-name/feature.md
```

**When to Use:**
- You prefer the standard folder structure
- You want consistency with default paths
- You don't have many files to move

---

### Option C: Use Sync to Auto-Detect and Fix

Let `shark sync` detect file locations and update the database:

```bash
# Preview what sync will do
shark sync --dry-run

# Use filesystem as source of truth
shark sync --strategy=file-wins

# Use database as source of truth (moves files)
shark sync --strategy=database-wins
```

**When to Use:**
- You have many files to fix
- You want automated detection
- You trust sync logic to resolve conflicts

---

## Migration Workflow

### Before Upgrading to E07-F19

1. **Backup your database:**
   ```bash
   cp shark-tasks.db shark-tasks.db.backup-before-e07-f19
   cp shark-tasks.db-wal shark-tasks.db-wal.backup-before-e07-f19
   cp shark-tasks.db-shm shark-tasks.db-shm.backup-before-e07-f19
   ```

2. **Run pre-migration detection:**
   ```bash
   ./scripts/detect-custom-folder-paths.sh
   ```

3. **Fix Case 3 issues (if any):**
   - Choose Option A, B, or C from "Fixing Case 3 Issues" above

4. **Re-run detection to verify:**
   ```bash
   ./scripts/detect-custom-folder-paths.sh
   ```

   Expected output:
   ```
   ✅ SAFE TO UPGRADE
   No custom_folder_path usage detected.
   ```

5. **Upgrade to E07-F19:**
   ```bash
   git pull
   make build
   ```

---

### After Upgrading to E07-F19

1. **Run post-migration validation:**
   ```bash
   ./scripts/validate-file-paths.sh
   ```

2. **Fix any path mismatches (if found):**
   - Use Option A, B, or C from "Fixing Case 3 Issues"

3. **Re-run validation to verify:**
   ```bash
   ./scripts/validate-file-paths.sh
   ```

   Expected output:
   ```
   ✅ VALIDATION PASSED
   All file paths are valid.
   ```

4. **Test normal operations:**
   ```bash
   shark epic list
   shark feature list
   shark task list
   ```

---

## Rollback Procedure

If you encounter issues after upgrading:

1. **Restore database backup:**
   ```bash
   cp shark-tasks.db.backup-before-e07-f19 shark-tasks.db
   cp shark-tasks.db-wal.backup-before-e07-f19 shark-tasks.db-wal
   cp shark-tasks.db-shm.backup-before-e07-f19 shark-tasks.db-shm
   ```

2. **Downgrade to E07-F18:**
   ```bash
   git checkout E07-F18
   make build
   ```

3. **Verify rollback:**
   ```bash
   shark epic list
   shark feature list
   ```

---

## Frequently Asked Questions

### Q: Will the migration delete my files?

**A:** No. The migration only removes database columns. No files are touched or moved.

### Q: Will I lose any data in the database?

**A:** No. The migration only removes the `custom_folder_path` columns. All other data (titles, descriptions, `file_path` values, etc.) is preserved.

### Q: What happens if I used `--path` but also set `--filename`?

**A:** If you set both, `--filename` takes precedence. Your `file_path` is already correct, so no action needed (Case 2).

### Q: Can I still use custom file paths after E07-F19?

**A:** Yes! Use the `--file` flag:
```bash
shark epic create "My Epic" --file="docs/custom/location/epic.md"
```

### Q: How do I know which case applies to me?

**A:** Run the pre-migration detection script:
```bash
./scripts/detect-custom-folder-paths.sh
```

It will tell you if you have any Case 3 instances.

### Q: What if I have hundreds of epics/features to fix?

**A:** Use the sync approach (Option C):
```bash
shark sync --strategy=file-wins
```

This will auto-detect all file locations and update the database.

---

## Support

If you encounter issues during migration:

1. Check this guide for solutions
2. Run the detection/validation scripts
3. Create a GitHub issue with:
   - Output from detection/validation scripts
   - Database schema: `sqlite3 shark-tasks.db ".schema epics"`
   - Error messages

---

## Summary

| Use Case | Impact | Action Required |
|----------|--------|-----------------|
| **Case 1**: Default paths only | ✅ None | None - upgrade safely |
| **Case 2**: Used `--filename` | ✅ None | None - upgrade safely |
| **Case 3**: Used `--path` | ⚠️ Yes | Update `file_path` or move files |

**Key Tools:**
- **Pre-migration:** `./scripts/detect-custom-folder-paths.sh`
- **Post-migration:** `./scripts/validate-file-paths.sh`
- **Auto-fix:** `shark sync --strategy=file-wins`

**Bottom Line:**
Most users will have zero impact. If you used `--path`, run the detection script before upgrading to identify what needs to be fixed.
