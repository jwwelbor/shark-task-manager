# Root Cause Analysis: Features Not Discovered During Shark Sync

**Date:** 2025-12-18
**Project:** wormwoodGM
**Issue:** shark sync finds 0 features when scanning docs/plan directory

## Problem Summary

When running `shark sync --dry-run --verbose` in the wormwoodGM project, no features are discovered despite having hundreds of properly named feature folders.

```
Files scanned:      0
New tasks imported: 0
Tasks updated:      0
```

## Root Cause: Hierarchical Folder Structure Mismatch

### Expected Discovery Structure

The Shark discovery code expects a 2-level hierarchy:

```
docs/plan/
├── E01-content-ingestion/        ✓ Matches epic pattern
│   ├── F01-feature-name/         ✓ Matches feature pattern (direct child of epic)
│   ├── F02-feature-name/         ✓ Matches feature pattern
│   └── ...
├── E02-cli-submission/           ✓ Matches epic pattern
│   ├── F01-feature-name/         ✓ Matches feature pattern
│   └── ...
```

### Actual WormwoodGM Structure

The wormwoodGM project has a 3-level hierarchy with intermediate organizational folders:

```
docs/plan/
├── E01-content-ingestion/                                 ✓ Epic found
│   ├── 01-foundation/                                    ✗ NOT a feature pattern
│   │   ├── F01-content-upload-security-implementation/   ✗ Feature hidden here
│   │   ├── F02-content-processing-storage-implementation/
│   │   ├── F03-ip-scanning-risk-management-implementation/
│   │   └── F04-multi-index-storage-system-implementation/
│   ├── 02-core_ingestion/                                ✗ NOT a feature pattern
│   │   ├── F05-rules-extraction-chunking-implementation/
│   │   ├── F06-character-data-extraction/
│   │   ├── F07-scenario-adventure-builder-implementation/
│   │   ├── F08-tables-equipment-indexer/
│   │   ├── F09-lore-narrative-extractor-implementation/
│   │   ├── F10-visual-asset-processor-implementation/
│   │   └── F17-toc-router-sectioning-implementation/
│   ├── 03-integration/
│   ├── 04-platform/
│   └── 05-missed_features/
├── E02-cli-content-submission/
│   ├── (similar 2-level structure)
```

## Discovery Code Analysis

### Key Scanning Logic (`internal/discovery/folder_scanner.go`)

**Line 99-110: Epic Detection**
```go
// Try to match as epic folder first
if epic, matched := s.matchEpicFolder(path, info.Name()); matched {
    epicFolderMap[path] = epic.Key
    return nil // Continue walking into epic folder
}
```
✓ Correctly identifies `E01-content-ingestion`

**Line 114-146: Feature Detection** (within epic)
```go
// Try to match as feature folder (within epic, possibly nested)
if epicKey, foundUnder := findEpicAncestor(path, epicFolderMap); foundUnder {
    result, matched := s.patternMatcher.MatchFeaturePattern(info.Name(), epicKey)
    if matched {
        // Build feature object
```

The `findEpicAncestor` function (line 31-50) walks UP the directory tree to find which epic a folder belongs to. However, the feature pattern matching on line 116 checks if the CURRENT directory name matches feature patterns.

### Feature Pattern Expectations (`internal/discovery/patterns.go`)

From the .sharkconfig.json in wormwoodGM:

```json
"feature": {
  "folder": [
    "^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$",
    "^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"
  ]
}
```

Pattern Analysis:
1. `^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$`
   - Matches: `E01-F01-feature-name` (full epic+feature in one name)
   - Does NOT match: `F01-feature-name` (would need parent epic in name)

2. `^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$`
   - Matches: `F01-feature-name` ✓
   - Does NOT match: `01-foundation` ✗

### Why Features Aren't Discovered

**When scanning `docs/plan/E01-content-ingestion/01-foundation/`:**

1. Scanner finds epic `E01-content-ingestion` → `epicFolderMap["...E01-content-ingestion"] = "E01"`
2. Scanner descends into E01 and encounters `01-foundation/`
3. `findEpicAncestor("...01-foundation", epicFolderMap)` returns `("E01", true)` ✓
4. `MatchFeaturePattern("01-foundation", "E01")` tries to match against patterns:
   - Pattern 1: `^E01-F\\d{2}-...` - Does "01-foundation" start with "E01-F"? NO ✗
   - Pattern 2: `^F\\d{2}-...` - Does "01-foundation" start with "F"? NO ✗
5. No match! `01-foundation` is skipped (returns nil, not filepath.SkipDir)
6. Scanner DOES continue walking into `01-foundation/` (because no error was returned)
7. Scanner finds `F01-content-upload-security-implementation/`
8. `findEpicAncestor("...F01-content-upload-security-implementation", epicFolderMap)` searches up:
   - Checks `.../01-foundation/` - not in epicFolderMap ✗
   - Checks `.../E01-content-ingestion/` - IS in epicFolderMap! ✓ Returns `("E01", true)`
9. `MatchFeaturePattern("F01-content-upload-security-implementation", "E01")` should match!

Wait, let me re-examine the walking logic more carefully...

## The Real Issue: filepath.Walk Behavior

When `filepath.Walk` encounters a directory, it:
1. Calls the callback function with that directory
2. If callback returns `nil`, continues to walk children
3. If callback returns `filepath.SkipDir`, skips the directory

**Current code behavior at line 148:**
```go
return nil  // Always returns nil, never skips directories
```

This means the scanner walks into EVERY directory under the epic, looking for features.

Let me trace the exact execution for `F01-content-upload-security-implementation`:

**Path:** `/home/jwwelbor/projects/wormwoodGM/docs/plan/E01-content-ingestion/01-foundation/F01-content-upload-security-implementation`

1. `findEpicAncestor()` starts at F01 folder path
2. Checks if `...F01-content-upload-security-implementation` is in `epicFolderMap` → NO
3. Walks up to `...01-foundation` → not in epicFolderMap → NO
4. Walks up to `...E01-content-ingestion` → IS in epicFolderMap! ✓ returns `("E01", true)`
5. Now calls `MatchFeaturePattern("F01-content-upload-security-implementation", "E01")`

The folder name is `F01-content-upload-security-implementation`. Let me check if it matches the pattern:
- Pattern: `^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$`
- Input: `F01-content-upload-security-implementation`
- Does it match? YES! `F` + `01` + `-` + `content-upload-security-implementation`

So why isn't it being found? Let me check if there's a PRD file required...

**Looking at findPrdFile() logic (line 218-255):**
The code searches for prd.md or PRD_* files. Let me check what's in the feature folder:

## CONFIRMED: Debug Output Shows 0 Features Found

Running the command:
```bash
shark sync --dry-run --index --verbose
```

Output:
```
DEBUG: FolderScanner.Scan found 14 epics, 0 features
DEBUG: Epic: E00 (launchpad)
DEBUG: Epic: E01 (content-ingestion)
...
```

The FolderScanner successfully finds 14 epics but discovers 0 features despite the presence of hundreds of feature folders with names like:
- `F01-content-upload-security-implementation`
- `F02-content-processing-storage-implementation`
- `F03-ip-scanning-risk-management-implementation`
- etc.

## Root Cause Identification

### Discovery Code Path

1. **Command:** `shark sync --index --dry-run`
   - Enables discovery mode: `opts.EnableDiscovery = true` (sync.go:170)
   - Calls `runDiscovery()` in sync/discovery.go

2. **Discovery Process (sync/discovery.go:54-58)**
   ```go
   scanner := discovery.NewFolderScanner()
   folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
   ```
   - Creates a FolderScanner
   - Calls Scan with docs/plan directory
   - Returns 14 epics, 0 features

3. **Folder Scanning Logic (discovery/folder_scanner.go:76-149)**

   The algorithm walks through all directories and:
   - Line 99: Try to match as epic → SUCCESS for `E01-content-ingestion`
   - Line 109: Return nil to continue walking into epic (CRITICAL!)
   - Line 114-146: For each child, check if under epic and try to match feature pattern

### The Actual Bug

**In folder_scanner.go, line 148:**
```go
return nil
```

This ALWAYS returns nil, which tells filepath.Walk to continue descending into ALL directories.

**Trace for E01-content-ingestion/01-foundation/:**

1. Scanner encounters `01-foundation` folder
2. Checks if it's under epic E01 → YES
3. Calls `MatchFeaturePattern("01-foundation", "E01")`
4. Pattern 1: `^E\d{2}-F\d{2}-...` → NO match
5. Pattern 2: `^F\d{2}-...` → NO match
6. `matched` returns false
7. The condition `if matched {` is false, so the feature is NOT added
8. BUT: The function returns `nil` (line 148), not `filepath.SkipDir`
9. Walk continues into `01-foundation/` and its children

**Trace for 01-foundation/F01-content-upload-security-implementation/:**

At this point, we're inside the intermediate folder `01-foundation`. Let's see what happens:

1. Scanner encounters `F01-content-upload-security-implementation`
2. Calls `findEpicAncestor("...F01-content-upload-security-implementation", epicFolderMap)`
3. Walks up directory tree:
   - Is `.../F01-content-upload-security-implementation` in epicFolderMap? NO
   - Is `.../01-foundation` in epicFolderMap? NO ← PROBLEM!
   - Is `.../E01-content-ingestion` in epicFolderMap? YES! ✓
4. Returns `("E01", true)`
5. Calls `MatchFeaturePattern("F01-content-upload-security-implementation", "E01")`
6. Pattern 1: `^E\d{2}-F\d{2}-...` → Does "F01-..." start with "E\d{2}-F"? NO
7. Pattern 2: `^F\d{2}-...` → Does "F01-..." start with "F\d{2}-"? YES! ✓
8. Should return true...

**BUT WAIT:** Let me check if there's a deeper issue. Let me trace the regex matching more carefully.
