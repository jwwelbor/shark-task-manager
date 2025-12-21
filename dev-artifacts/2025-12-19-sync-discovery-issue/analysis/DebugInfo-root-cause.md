# Debug Info: Sync Discovery Issue for E07 Tasks

**Issue**: `shark sync --index --create-missing` is not discovering and creating tasks in:
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/tasks/`
- `docs/plan/E07-enhancements/E07-F09-custom-folder-base-paths/tasks/`

## Root Cause

There is a **regex pattern mismatch** between the file scanner and the pattern registry:

### 1. Scanner's Strict Pattern (scanner.go:36)
```go
keyPattern: regexp.MustCompile(`^T-(E\d{2})-(F\d{2})-\d{3}\.md$`)
```
This pattern **ONLY** matches task filenames with format: `T-E##-F##-###.md`
- ✓ Matches: `T-E07-F08-001.md`
- ✗ **FAILS**: `T-E07-F08-001-database-schema-migration.md`
- ✗ **FAILS**: `T-E07-F08-002-repository-methods.md`

### 2. Pattern Registry's Flexible Pattern (defaults.go:48)
```go
`^T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3}).*\.md$`
```
This pattern matches task filenames with **optional descriptive suffix**: `T-E##-F##-###<anything>.md`
- ✓ Matches all variations above

## Impact on Sync Logic

**File Flow:**
1. **FileScanner.Scan()** walks the filesystem
2. Calls `scanner.extractKeyFromFilename()` → uses **strict keyPattern**
3. When strict pattern fails, **epicKey and featureKey become empty**
4. File still gets added to scan results (lines 106-113)
5. In **SyncEngine.parseFiles()**, the pattern registry uses **flexible pattern** ✓
6. Task file IS discovered and matched... BUT

**The Problem:**
- When `scanner.extractKeyFromFilename()` fails (line 182), it returns empty strings
- This makes it hard to infer the feature/epic relationship early
- The sync engine CAN eventually extract the task key from the filename via the parser
- BUT if there's an issue with the path-based feature inference (line 97), the sync might fail

## File Status Check

**E07-F08 (with descriptive names - FAILING):**
```
-rw------- T-E07-F08-001-database-schema-migration.md  ← Restrictive perms + descriptive name
-rw------- T-E07-F08-002-repository-methods.md
```

**E07-F09 (simple names - WORKS):**
```
-rw-r--r-- T-E07-F09-001.md
-rw-r--r-- T-E07-F09-002.md
```

**Working example E04-F01:**
```
-rw-r--r-- T-E04-F01-001.md
-rw-r--r-- T-E04-F01-002.md
```

## Issues Found (3)

### Issue #1: Strict Regex Pattern (FIXED ✓)
**File**: `internal/sync/scanner.go:36`
**Problem**: The `keyPattern` regex was too strict, rejecting filenames with descriptive suffixes
**Root Cause**: Pattern used `\d{3}\.md$` which required `.md` immediately after task number
**Solution**: Changed to `\d{3}.*\.md$` to allow optional descriptive text before `.md`

### Issue #2: Directory Permissions (FIXED ✓)
**Path**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/`
**Problem**: Directory had restrictive permissions `700` (drwx------)
**Impact**: FileScanner couldn't read the directory, silently skipped it during filesystem walk
**Solution**: Changed permissions to `755` (drwxr-xr-x)

### Issue #3: Pattern Registry Not Updated (FIXED ✓)
**File**: `internal/sync/patterns.go:37`
**Problem**: The PatternRegistry used for filtering which files to scan still had the OLD strict regex pattern
**Why it mattered**: Even though scanner.go and defaults.go were fixed, the PatternRegistry (used by FileScanner.getMatchingPattern) was still rejecting files with descriptive names
**Impact**: Files matching T-E##-F##-###-description.md were FILTERED OUT by the scanner, so they never reached the parsing stage
**Root cause**: Three separate pattern definitions that weren't kept in sync:
  1. `scanner.go:36` - keyPattern (for extracting keys from filenames)
  2. `patterns.go:37` - PatternRegistry Task pattern (for filtering which files to scan) ← **THE BOTTLENECK**
  3. `defaults.go:48` - Default pattern (reference pattern in config)
**Solution**: Updated PatternRegistry regex to `^T-E\d{2}-F\d{2}-\d{3}.*\.md$`

## Changes Made

1. **Scanner keyPattern regex updated** (internal/sync/scanner.go:36):
   ```go
   - keyPattern: regexp.MustCompile(`^T-(E\d{2})-(F\d{2})-\d{3}\.md$`)
   + keyPattern: regexp.MustCompile(`^T-(E\d{2})-(F\d{2})-\d{3}.*\.md$`)
   ```

2. **Pattern Registry regex updated** (internal/sync/patterns.go:37):
   ```go
   - Regex:   regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}\.md$`),
   + Regex:   regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}.*\.md$`),
   ```

3. **Directory permissions fixed**:
   ```bash
   chmod -R 755 docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features
   ```

## Verification

After applying all 3 fixes:

**Scanner improvements:**
- Before: 68 total files scanned, E07-F08 files skipped
- After: 97 total files scanned, all E07-F08 files found

**Database imports:**
- Before: 68 tasks imported (no E07-F08 tasks)
- After: 74 tasks imported (includes 6 E07-F08 tasks)

**E07-F08 tasks now discoverable:**
- ✓ T-E07-F08-001 - Add file_path columns to epics and features tables
- ✓ T-E07-F08-002 - Implement GetByFilePath and UpdateFilePath
- ✓ T-E07-F08-003 - Make ValidateCustomFilename accessible
- ✓ T-E07-F08-004 - Add --filename and --force flags to epic create
- ✓ T-E07-F08-005 - Add --filename and --force flags to feature create
- ✓ T-E07-F08-006 - Update CLI reference and user documentation

**Test command:** `shark task get T-E07-F08-001` now works correctly
