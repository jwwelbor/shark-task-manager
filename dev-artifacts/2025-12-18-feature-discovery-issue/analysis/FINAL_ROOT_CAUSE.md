# Final Root Cause Analysis: Zero Features Discovered in wormwoodGM

**Date:** 2025-12-18
**Status:** ROOT CAUSE IDENTIFIED ✓
**Severity:** Critical - Breaks feature discovery for nested organizational structures

---

## Problem Statement

Running `shark sync --index --dry-run` in the wormwoodGM project produces:
```
DEBUG: FolderScanner.Scan found 14 epics, 0 features
```

Despite the project having hundreds of properly-named features in folders like:
- `F01-content-upload-security-implementation`
- `F02-content-processing-storage-implementation`
- `F03-ip-scanning-risk-management-implementation`

---

## Directory Structure Analysis

### Expected Structure (What Works)
```
docs/plan/
├── E04-task-mgmt-cli-core/          ← Epic (matches pattern)
│   ├── E04-F01-database-schema/     ← Feature (direct child, matches pattern)
│   └── E04-F07-initialization-sync/ ← Feature (direct child, matches pattern)
```

**Result:** Epics: 1, Features: 2 ✓

### Actual wormwoodGM Structure (What Breaks)
```
docs/plan/
├── E01-content-ingestion/                              ← Epic found ✓
│   ├── 01-foundation/                                 ← INTERMEDIATE (no pattern match) ✗
│   │   ├── F01-content-upload-security-implementation/ ← Feature (HIDDEN)
│   │   ├── F02-content-processing-storage-implementation/
│   │   ├── F03-ip-scanning-risk-management-implementation/
│   │   └── F04-multi-index-storage-system-implementation/
│   ├── 02-core_ingestion/                             ← INTERMEDIATE (no pattern match) ✗
│   │   ├── F05-rules-extraction-chunking-implementation/
│   │   ├── F06-character-data-extraction/
│   │   ├── F07-scenario-adventure-builder-implementation/
│   │   └── ... (more features)
│   ├── 03-integration/
│   ├── 04-platform/
│   └── 05-missed_features/
```

**Result:** Epics: 14 (found), Features: 0 (NOT found) ✗

---

## Feature Pattern Configuration

From `.sharkconfig.json`:
```json
"feature": {
  "folder": [
    "^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$",
    "^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"
  ]
}
```

**Pattern 1:** `E##-F##-slug` (e.g., `E01-F01-feature-name`)
**Pattern 2:** `F##-slug` (e.g., `F01-feature-name`) - Should match features in wormwood!

---

## Code Trace: Why Features Aren't Found

### File: `internal/discovery/folder_scanner.go` (Scan function, lines 76-149)

**Algorithm:**
```
for each directory in docs/plan:
  1. Check if matches epic pattern
     - YES: Add to epics, track in epicFolderMap, return nil (continue walking)
     - NO: Continue to step 2

  2. Check if under an epic ancestor
     - YES: Try to match feature pattern
       - YES: Add to features
       - NO: Do nothing
     - NO: Do nothing

  3. Return nil (always allows walking into subdirectories)
```

### Execution Trace for E01-content-ingestion/01-foundation/F01-content-upload-security-implementation

**Step 1: Walking encounters `E01-content-ingestion/`**
- Line 99: `matchEpicFolder("E01-content-ingestion")` → MATCH ✓
- Line 108: `epicFolderMap[path] = "E01"` ← Records epic location
- Line 109: `return nil` ← Continues walking into E01
- **Result:** Epic E01 found ✓

**Step 2: Walking encounters `E01-content-ingestion/01-foundation/`**
- Line 96: `stats.FoldersScanned++`
- Line 99: `matchEpicFolder("01-foundation")` → NO MATCH ✗
  - Pattern 1: `^E\d{2}-F\d{2}-...` NO (starts with "0")
  - Pattern 2: (no second pattern for epics)
- Line 114: `findEpicAncestor("...01-foundation", epicFolderMap)`
  - Walks up: "...01-foundation" not in map
  - Walks up: "...E01-content-ingestion" IS in map!
  - Returns `("E01", true)` ✓
- Line 116: `MatchFeaturePattern("01-foundation", "E01")`
  - Pattern 1: `^E\d{2}-F\d{2}-...` → "01-foundation" starts with "0", not "E\d{2}-F" NO ✗
  - Pattern 2: `^F\d{2}-...` → "01-foundation" starts with "0", not "F" NO ✗
  - Returns `false`
- Line 117: `if matched` → FALSE, feature not added
- Line 148: `return nil` ← **CRITICAL: Always returns nil, allows descent**
- **Result:** Intermediate folder skipped but walk continues

**Step 3: Walking encounters `E01-content-ingestion/01-foundation/F01-content-upload-security-implementation/`**
- Line 96: `stats.FoldersScanned++`
- Line 99: `matchEpicFolder("F01-content-upload-security-implementation")` → NO ✗
- Line 114: `findEpicAncestor("...F01-content-upload-security-implementation", epicFolderMap)`
  - Walks up: "...F01-content-upload-security-implementation" not in map
  - Walks up: "...01-foundation" not in map
  - Walks up: "...E01-content-ingestion" IS in map!
  - Returns `("E01", true)` ✓
- Line 116: `MatchFeaturePattern("F01-content-upload-security-implementation", "E01")`
  - Pattern 1: `^E\d{2}-F\d{2}-...`
    - String: "F01-content-upload-security-implementation"
    - Must start with "E" + digits + "-F" + digits + "-"
    - "F01-..." starts with "F", NOT "E\d{2}-F"
    - NO MATCH ✗
  - Pattern 2: `^F\d{2}-...`
    - String: "F01-content-upload-security-implementation"
    - Must start with "F" + digits + "-"
    - "F01-..." starts with "F01-"
    - **SHOULD MATCH** ✓

**Expected:** Feature should be found here!

---

## CRITICAL DISCOVERY: The REAL Bug

Looking at line 116-117 in folder_scanner.go:

```go
result, matched := s.patternMatcher.MatchFeaturePattern(info.Name(), epicKey)
if matched {
    // Add feature...
}
```

The pattern matcher should return `matched=true` for `F01-content-upload-security-implementation` against pattern `^F\d{2}-...`.

**But it's not!**

### Hypothesis 1: The patterns in config aren't being loaded

Check if pattern config is being used. In discovery.go:54-55:
```go
scanner := discovery.NewFolderScanner()
folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
```

**The third parameter is `nil`** - this means pattern overrides are nil!

### In folder_scanner.go, lines 53-59:

```go
func (s *FolderScanner) Scan(docsRoot string, patternOverrides *patterns.PatternConfig) (
    []FolderEpic, []FolderFeature, ScanStats, error) {

    // Override patterns if provided
    if patternOverrides != nil {
        s.patternMatcher = NewPatternMatcher(patternOverrides)
    }
```

Since `patternOverrides` is nil, the pattern matcher uses whatever patterns were initialized in `NewFolderScanner()`:

### In folder_scanner.go, lines 18-22:

```go
func NewFolderScanner() *FolderScanner {
    return &FolderScanner{
        patternMatcher: NewPatternMatcher(patterns.GetDefaultPatterns()),
    }
}
```

**This uses `patterns.GetDefaultPatterns()`!**

### The Bug: Default Patterns vs Config Patterns

The sync/discovery.go code loads pattern config from .sharkconfig.json but doesn't pass it to the FolderScanner!

**In sync/discovery.go, line 55:**
```go
folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
                                                                                                   ↑
                                                                          PASSING nil INSTEAD OF CONFIG!
```

The FolderScanner gets initialized with `patterns.GetDefaultPatterns()` instead of the project-specific patterns from .sharkconfig.json.

---

## Proof of Hypothesis

Check what `GetDefaultPatterns()` returns vs what's in .sharkconfig.json.

**Current code path:**
1. Sync command loads `.sharkconfig.json` ✓ (line 153)
2. Creates sync engine ✓
3. Calls runDiscovery() ✓
4. Creates FolderScanner with default patterns (BUG! Should use config patterns)
5. FolderScanner uses default patterns, not project patterns
6. Default patterns may not match wormwood's nested structure

---

## Root Cause Summary

**Primary Issue:** FolderScanner.Scan() is called with `patternOverrides=nil`, causing it to use default patterns instead of project-specific patterns from `.sharkconfig.json`.

**Secondary Issue:** Even with default patterns, the intermediate folders like "01-foundation" don't match any feature pattern, and the scanner doesn't skip them, continuing descent into nested folders. The feature pattern matching for deeply nested features (`E01 -> 01-foundation -> F01`) should still work with proper patterns, but doesn't because:
1. The config patterns aren't being passed to the scanner
2. Default patterns may not include the flexible nested structure

---

## Impact

- Features organized under intermediate organizational folders are not discovered
- Discovery only works for features that are direct children of epics (the expected structure)
- Projects using multi-level hierarchies (like wormwoodGM) cannot use the discovery feature
- Workaround: Use `--pattern=task` for file-based sync instead of discovery

---

## Solution

Pass the pattern configuration from sync/discovery.go to the FolderScanner:

```go
// In sync/discovery.go, around line 54
scanner := discovery.NewFolderScanner()
folderEpics, folderFeatures, _, err := scanner.Scan(
    filepath.Join(e.docsRoot, opts.FolderPath),
    e.patternConfig,  // ← PASS THE CONFIG INSTEAD OF nil
)
```

This requires:
1. Storing the pattern config in the SyncEngine
2. Passing it to Scan() instead of nil
3. Ensuring the pattern config is loaded from .sharkconfig.json

---

## Files Affected

**Root cause location:**
- `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go` (line 55)

**Related files:**
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner.go`
- `/home/jwwelbor/projects/shark-task-manager/internal/sync/engine.go`
- `/home/jwwelbor/projects/shark-task-manager/internal/sync/types.go`

