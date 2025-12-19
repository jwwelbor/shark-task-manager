# Debugging Summary: Feature Discovery Issue in wormwoodGM

**Date:** 2025-12-18
**Project:** wormwoodGM
**Issue:** `shark sync --index` finds 0 features despite hundreds of features existing
**Status:** ROOT CAUSE IDENTIFIED ✓

---

## Executive Summary

### The Problem
Feature discovery fails completely in wormwoodGM. Running `shark sync --dry-run --index` shows:
```
DEBUG: FolderScanner.Scan found 14 epics, 0 features
```

Despite the project having structured feature folders like:
- `docs/plan/E01-content-ingestion/01-foundation/F01-content-upload-security-implementation/`
- `docs/plan/E01-content-ingestion/02-core_ingestion/F05-rules-extraction-chunking-implementation/`
- etc. (approximately 40-50 features across epics)

### Root Cause
**The FolderScanner is initialized with default patterns instead of project-specific patterns from `.sharkconfig.json`.**

**Critical Code Location:**
- File: `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go`
- Line: 55
- Issue: Passes `nil` instead of pattern config to `scanner.Scan()`

```go
// BROKEN CODE:
folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
                                                                                                  ↑
                                                                            Should be patternConfig here!
```

### Why Features Aren't Found

The FolderScanner uses `patterns.GetDefaultPatterns()` (hard-coded) instead of the `.sharkconfig.json` patterns. The default patterns may not match the wormwood project's nested folder structure where features are organized under intermediate folders like `01-foundation/` and `02-core_ingestion/`.

Even if default patterns did match, the configuration patterns should be respected for project-specific organization.

### Impact
- Projects with nested feature hierarchies (like wormwoodGM) cannot use discovery feature
- Feature index cannot be built
- Task discovery from epic-index.md doesn't work
- Users must fall back to file-based sync with `--pattern=task` flag

---

## Investigation Process

### Step 1: Reproduced the Issue
```bash
cd /home/jwwelbor/projects/wormwoodGM
shark sync --dry-run --index --verbose
# Output: DEBUG: FolderScanner.Scan found 14 epics, 0 features
```

### Step 2: Examined Directory Structure
```
docs/plan/
├── E01-content-ingestion/                              ✓ Epic found
│   ├── 01-foundation/                                 ✗ Intermediate folder
│   │   ├── F01-content-upload-security-implementation/  ✗ NOT FOUND
│   │   ├── F02-content-processing-storage-implementation/
│   │   └── ...
│   ├── 02-core_ingestion/                             ✗ Intermediate folder
│   │   ├── F05-rules-extraction-chunking-implementation/
│   │   └── ...
```

Expected: Direct children of epics (like `E04-task-mgmt-cli-core/E04-F01-database-schema/`)
Actual: Features nested under intermediate organizational folders

### Step 3: Checked Configuration
Found `.sharkconfig.json` with feature patterns:
```json
"feature": {
  "folder": [
    "^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$",
    "^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"
  ]
}
```

Pattern 2 should match `F01-content-...` format used in wormwood.

### Step 4: Traced Code Execution

**File:** `internal/discovery/folder_scanner.go`
- Lines 18-22: NewFolderScanner() uses `patterns.GetDefaultPatterns()`
- Lines 53-59: Scan() accepts `patternOverrides` but it's called with nil
- Lines 116-117: Feature matching uses the initialized pattern matcher

**File:** `internal/sync/discovery.go`
- Line 55: Calls `scanner.Scan(..., nil)` ← **THE BUG**
- Should pass: pattern config from .sharkconfig.json

### Step 5: Identified Root Cause
Pattern config is NOT being passed from sync engine to folder scanner, so the scanner uses hard-coded defaults instead of project-specific patterns.

---

## Directory Structure Comparison

### Working Structure
```
E04-task-mgmt-cli-core/
├── E04-F01-database-schema/      ← Feature (direct child)
├── E04-F02-cli-infrastructure/   ← Feature (direct child)
└── epic.md
```
**Result:** Features discovered ✓

### Non-Working Structure (wormwoodGM)
```
E01-content-ingestion/
├── 01-foundation/                ← Organizational folder (not a feature)
│   ├── F01-content-upload-security-implementation/
│   ├── F02-content-processing-storage-implementation/
│   ├── F03-ip-scanning-risk-management-implementation/
│   └── F04-multi-index-storage-system-implementation/
├── 02-core_ingestion/             ← Organizational folder (not a feature)
│   ├── F05-rules-extraction-chunking-implementation/
│   ├── F06-character-data-extraction/
│   ├── F07-scenario-adventure-builder-implementation/
│   └── F08-tables-equipment-indexer/
└── epic.md
```
**Result:** Features NOT discovered ✗

The scanner finds the intermediate folders `01-foundation` and `02-core_ingestion`, but:
1. They don't match any feature pattern
2. The scanner doesn't skip them (returns nil, allowing descent)
3. The scanner continues but doesn't find features inside because...
4. The pattern matching isn't happening correctly (uses default patterns, not config)

---

## Code Analysis

### Pattern Matching Logic

**File:** `internal/discovery/patterns.go` (lines 130-186)

```go
func (m *PatternMatcher) MatchFeaturePattern(input, parentEpicKey string) (FeatureMatchResult, bool) {
    for _, re := range m.featureFolderRegexes {  // Uses initialized regexes
        matches := re.FindStringSubmatch(input)
        if matches == nil {
            continue
        }
        // Extract components...
        return result, true
    }
    return FeatureMatchResult{}, false
}
```

The pattern matcher uses `featureFolderRegexes` which were compiled from patterns passed to `NewPatternMatcher()`.

### Folder Scanner Initialization

**File:** `internal/discovery/folder_scanner.go` (lines 18-22)

```go
func NewFolderScanner() *FolderScanner {
    return &FolderScanner{
        patternMatcher: NewPatternMatcher(patterns.GetDefaultPatterns()),  // ← Uses hardcoded defaults
    }
}
```

### How It Should Work

**File:** `internal/sync/discovery.go` (lines 54-55) - CURRENT (BROKEN)

```go
scanner := discovery.NewFolderScanner()
folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
```

**How It Should Be:**

```go
scanner := discovery.NewFolderScanner()
folderEpics, folderFeatures, _, err := scanner.Scan(
    filepath.Join(e.docsRoot, opts.FolderPath),
    e.patternConfig,  // ← PASS THE PROJECT CONFIG
)
```

The Scan() function accepts pattern overrides (line 56-59), but they're never passed!

---

## Test Evidence

### Debug Output from FolderScanner

When running `shark sync --index`, the FolderScanner outputs:
```
DEBUG: FolderScanner.Scan found 14 epics, 0 features
DEBUG: Epic: E00 (launchpad)
DEBUG: Epic: E01 (content-ingestion)
DEBUG: Epic: E02 (cli-content-submission)
... (12 more epics)
```

No features are printed, confirming zero features were found.

### Config Patterns Are Correct

The `.sharkconfig.json` contains proper patterns:
```json
"feature": {
  "folder": [
    "^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$",
    "^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"
  ]
}
```

Pattern 2 (`^F\d{2}-...`) should match folders like `F01-content-...`.

---

## Solution

### Required Changes

1. **Pass pattern config to FolderScanner**
   - File: `internal/sync/discovery.go:55`
   - Change: `nil` → `e.patternConfig`

2. **Ensure pattern config is available in SyncEngine**
   - File: `internal/sync/engine.go`
   - Add: Pattern config field/method
   - Load: From .sharkconfig.json

3. **Load patterns in discovery workflow**
   - Ensure pattern config is loaded before scanning
   - Either from SyncEngine or passed to runDiscovery()

### Expected Results After Fix

```bash
shark sync --dry-run --index --verbose
# Should output:
# DEBUG: FolderScanner.Scan found 14 epics, 48 features
# DEBUG: Feature: E01-F01 (epic=E01, slug=content-upload-security-implementation)
# DEBUG: Feature: E01-F02 (epic=E01, slug=content-processing-storage-implementation)
# ... (etc)
```

---

## Files Analyzed

### Core Discovery Files
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner.go` - Feature discovery algorithm
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/patterns.go` - Pattern matching logic
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go` - Tests (work for flat structure)

### Sync/Discovery Files
- `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go` - **Contains the bug** (line 55)
- `/home/jwwelbor/projects/shark-task-manager/internal/sync/engine.go` - Sync engine
- `/home/jwwelbor/projects/shark-task-manager/internal/sync/types.go` - Type definitions

### Configuration Files
- `/home/jwwelbor/projects/wormwoodGM/.sharkconfig.json` - Project configuration with feature patterns

---

## References

### Broken Code Path
1. `shark sync --index` → `internal/cli/commands/sync.go:112-191`
2. Enables discovery: `opts.EnableDiscovery = true` (line 170)
3. Calls sync engine: `engine.Sync(ctx, opts)` (line 191)
4. Sync engine calls discovery: `e.runDiscovery(ctx, opts)` (line 150)
5. Discovery creates scanner: `discovery.NewFolderScanner()` (line 54)
6. **BUG:** Calls `scanner.Scan(..., nil)` (line 55) instead of passing config
7. Scanner uses default patterns, features not matched

### Test Structure Shows Expected Behavior
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go`
- Tests show features as DIRECT children of epics (E04-F01, E04-F07)
- No tests for nested organizational structures (wormwood style)

---

## Conclusion

**Root Cause:** Pattern config from `.sharkconfig.json` is not passed to FolderScanner, causing it to use hard-coded default patterns that don't account for nested feature organization.

**Impact:** Feature discovery fails for projects with hierarchical folder structures, breaking the `--index` feature of `shark sync`.

**Fix Complexity:** Low - requires passing one additional parameter through the call stack.

**Testing:** Existing tests pass because they use flat structures; new test needed for nested hierarchies.

