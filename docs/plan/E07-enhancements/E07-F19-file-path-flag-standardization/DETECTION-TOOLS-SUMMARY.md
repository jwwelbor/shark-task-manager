# E07-F19 Detection Tools - Summary

## What Was Created

Three comprehensive tools to help users detect and fix Case 3 migration issues:

### 1. Pre-Migration Detection Script
**Location:** `scripts/detect-custom-folder-paths.sh`

**Purpose:** Identify epics and features using `custom_folder_path` BEFORE upgrading to E07-F19

**Features:**
- Detects all Case 3 instances (entities with `custom_folder_path` set)
- Shows affected epics and features in table format
- Generates SQL fix scripts automatically
- Safe to run multiple times
- Zero impact on database (read-only)

**Usage:**
```bash
# Basic detection
./scripts/detect-custom-folder-paths.sh

# With specific database path
./scripts/detect-custom-folder-paths.sh path/to/shark-tasks.db

# Generate fix script
./scripts/detect-custom-folder-paths.sh shark-tasks.db --generate-fix-script > fix.sql
```

**Example Output:**
```
=========================================
E07-F19 Pre-Migration Detection
=========================================
⚠️  Found epics using custom_folder_path (Case 3):
key   title           file_path  custom_folder_path
----  --------------  ---------  ---------------------------
E01   Q1 2025 Roadmap NULL       docs/roadmap/2025-q1

ACTION REQUIRED BEFORE UPGRADE
```

---

### 2. Post-Migration Validation Script
**Location:** `scripts/validate-file-paths.sh`

**Purpose:** Detect path mismatches between database and filesystem AFTER upgrading to E07-F19

**Features:**
- Checks if files exist at expected locations
- Validates epics, features, and tasks
- Handles both NULL file_path (default paths) and explicit paths
- Provides actionable fix recommendations
- Safe to run multiple times

**Usage:**
```bash
# Basic validation
./scripts/validate-file-paths.sh

# With specific paths
./scripts/validate-file-paths.sh shark-tasks.db docs/plan
```

**Example Output:**
```
=========================================
E07-F19 Post-Migration Validation
=========================================
⚠️  Epic E01: file_path is NULL, expected default at docs/plan/E01-q1/epic.md (MISSING)

Found 1 path mismatch(es).

RECOMMENDED FIXES:
- Run: shark sync --strategy=file-wins
- Or: shark epic update E01 --file="actual-path.md"
```

---

### 3. Comprehensive Migration Guide
**Location:** `docs/E07-F19-MIGRATION-GUIDE.md`

**Purpose:** Complete reference for E07-F19 migration with detection tools

**Contents:**
- Overview of changes (what was removed/preserved)
- Impact by use case (Case 1, 2, 3 analysis)
- Detection tool usage instructions
- Three fix options (update database, move files, use sync)
- Complete migration workflow (before and after)
- Rollback procedure
- FAQ section
- Troubleshooting guide

**Key Sections:**
1. What Changed
2. Impact by Use Case
3. Detection Tools (both scripts)
4. Fixing Case 3 Issues (3 options)
5. Migration Workflow
6. Rollback Procedure
7. FAQ

---

## How Users Should Use These Tools

### Scenario 1: User Upgrading from E07-F18 (Has custom_folder_path)

**Before Upgrade:**
1. Run: `./scripts/detect-custom-folder-paths.sh`
2. If Case 3 found, choose fix option:
   - **Option A:** Generate and apply SQL fix
   - **Option B:** Wait for upgrade, use sync after
3. Backup database
4. Upgrade to E07-F19

**After Upgrade:**
1. Run: `./scripts/validate-file-paths.sh`
2. If mismatches found, apply fixes (sync or manual)
3. Verify with: `shark epic list`, `shark feature list`

---

### Scenario 2: User Already on E07-F19 (Migration Already Happened)

**Current State:**
- `custom_folder_path` columns already removed
- May have path mismatches

**Action:**
1. Run: `./scripts/validate-file-paths.sh`
2. Fix any mismatches found
3. Done!

---

### Scenario 3: New User (Never Used custom_folder_path)

**Current State:**
- Never used `--path` flag with custom folder base paths
- Only used default paths or `--file`/`--filename`

**Action:**
- No action needed
- Both scripts will report "safe" status
- Can upgrade without concern

---

## Integration with Existing Documentation

### Updated Files

1. **`docs/MIGRATION_CUSTOM_PATHS.md`**
   - Added prominent link to E07-F19-MIGRATION-GUIDE.md
   - Added detection tools section
   - Updated migration steps to include pre/post checks

2. **`README.md`** (already references MIGRATION_CUSTOM_PATHS.md)
   - Line 662: Migration guide link exists
   - No changes needed - existing reference sufficient

---

## Testing the Tools

### Test Pre-Migration Script

```bash
# Should work on any database
./scripts/detect-custom-folder-paths.sh

# If database doesn't have custom_folder_path columns:
# ✅ Safe to upgrade (columns already removed)

# If database has columns but no values:
# ✅ Safe to upgrade (no Case 3 instances)

# If database has columns with values:
# ⚠️ Action required (Case 3 instances found)
```

### Test Post-Migration Script

```bash
# Should work on any database
./scripts/validate-file-paths.sh

# Reports mismatches between database file_path and actual filesystem
```

---

## Key Benefits

### For Users
- **Confidence:** Know exactly what will happen before upgrading
- **Safety:** Detect issues before they become problems
- **Automation:** Auto-generate fix scripts instead of manual work
- **Flexibility:** Three fix options to match different workflows

### For Maintainers
- **Support Reduction:** Users can self-diagnose and fix
- **Documentation:** Comprehensive guide reduces confusion
- **Transparency:** Scripts show exactly what's happening
- **Rollback Path:** Clear rollback instructions if needed

---

## Files Created

```
scripts/
├── detect-custom-folder-paths.sh     # Pre-migration detection
└── validate-file-paths.sh            # Post-migration validation

docs/
├── E07-F19-MIGRATION-GUIDE.md        # Comprehensive guide
├── MIGRATION_CUSTOM_PATHS.md         # Updated with detection tools
└── DETECTION-TOOLS-SUMMARY.md        # This file
```

---

## Next Steps

### Recommended Actions

1. **Test the scripts** on your current database:
   ```bash
   ./scripts/detect-custom-folder-paths.sh
   ./scripts/validate-file-paths.sh
   ```

2. **Review the migration guide:**
   ```bash
   cat docs/E07-F19-MIGRATION-GUIDE.md
   ```

3. **Update README** (optional) to add detection tools section

4. **Announce** these tools to users in:
   - Release notes for E07-F19
   - GitHub README
   - Migration guide

---

## Summary

You now have:
- ✅ Pre-migration detection (before upgrade)
- ✅ Post-migration validation (after upgrade)
- ✅ Comprehensive migration guide
- ✅ Auto-generated fix scripts
- ✅ Three fix strategies
- ✅ Clear rollback path
- ✅ FAQ and troubleshooting

**Bottom Line:** Users can now detect and fix Case 3 issues both before and after upgrading to E07-F19, with clear guidance and automated tools.
