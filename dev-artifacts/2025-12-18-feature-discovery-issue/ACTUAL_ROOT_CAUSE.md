# ACTUAL ROOT CAUSE: Feature Discovery Fails Only for E01

**Date:** 2025-12-18
**Status:** ROOT CAUSE IDENTIFIED - DIFFERENT ISSUE

## The Real Problem

Feature discovery works for E02-E07 (where features are direct children) but fails for E01 (where features are nested under intermediate organizational folders like `01-foundation`, `02-core_ingestion`, etc.).

**This is NOT a pattern config issue - it's a feature pattern matching issue.**

## Directory Structure Comparison

### E02-cli-content-submission (WORKS)
```
E02-cli-content-submission/
├── F01-ruleset-management-api-implementation/
├── F02-campaign-management-api-implementation/
├── F03-scenario-management-api-implementation/
├── ... (direct feature children)
└── epic.md
```

Scanner path walk:
1. Visits `E02-cli-content-submission/` → matches epic pattern ✓
2. Visits `F01-ruleset-management-api-implementation/` → matches feature pattern ✓
3. Visits `F02-campaign-management-api-implementation/` → matches feature pattern ✓

Result: **Features found!**

### E01-content-ingestion (FAILS)
```
E01-content-ingestion/
├── 01-foundation/
│   ├── F01-content-upload-security-implementation/
│   ├── F02-content-processing-storage-implementation/
│   ├── F03-ip-scanning-risk-management-implementation/
│   └── F04-multi-index-storage-system-implementation/
├── 02-core_ingestion/
│   ├── F05-rules-extraction-chunking-implementation/
│   ├── F06-character-data-extraction/
│   ├── F07-scenario-adventure-builder-implementation/
│   ├── ... (more features)
├── 03-integration/
├── 04-platform/
├── 05-missed_features/
└── epic.md
```

Scanner path walk:
1. Visits `E01-content-ingestion/` → matches epic pattern ✓ (epicKey = "E01")
2. Visits `01-foundation/` → tries to match as feature?
3. Visits `F01-content-upload-security-implementation/` → tries to match as feature?

Result: **Features NOT found!**

## The Bug in folder_scanner.go

Look at **folder_scanner.go:114-146** (the feature matching logic):

```go
// Try to match as feature folder (within epic, possibly nested)
// Check if this directory is under an epic (any ancestor)
if epicKey, foundUnder := findEpicAncestor(path, epicFolderMap); foundUnder {
    // Try to match feature pattern
    result, matched := s.patternMatcher.MatchFeaturePattern(info.Name(), epicKey)
    if matched {
        // Build feature object and add to features list
        feature := FolderFeature{...}
        features = append(features, feature)
    }
}
```

The problem is that it tries to match EVERY folder that's an ancestor of an epic against the feature pattern.

### What Happens with E01

Walking `/home/wormwoodGM/docs/plan/E01-content-ingestion/01-foundation/F01-content-upload-security-implementation`:

1. Scanner visits folder: `01-foundation`
2. Checks if it's under an epic: YES (ancestor is E01) ✓
3. Tries to match `01-foundation` against feature pattern `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`
4. Does NOT match ✓ (correctly)
5. **Continues walking, eventually visits:**
6. Scanner visits folder: `F01-content-upload-security-implementation`
7. Checks if it's under an epic: YES (ancestor is E01) ✓
8. Tries to match `F01-content-upload-security-implementation` against feature pattern
9. Does NOT match ✗ **THIS IS THE BUG**

## Why Does It Not Match?

The default feature pattern is:
```regex
^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$
```

This pattern REQUIRES the folder name to start with `E##-F##-` format.

Test cases:
- `E02-F01-ruleset-management-api-implementation` → MATCHES (has E02-F01 prefix)
- `F01-content-upload-security-implementation` → DOES NOT MATCH (missing E01- prefix)

The pattern is designed for **direct children of epics**, not nested features!

## The Actual Bug

The FolderScanner's feature matching logic assumes:
1. All features are direct children of their epic folder OR
2. Features contain the full epic key in their folder name (E##-F##-)

But E01 violates both assumptions:
1. Features are NOT direct children (they're nested under intermediate folders)
2. Features DON'T contain the epic key (they're just F##-xxx)

## Solution Required

The scanner needs to match features by **name pattern ONLY** (F##-xxx) when they're found under an epic, regardless of nesting depth.

### Current Pattern (INCORRECT for E01)
```regex
^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$
```
- Requires: E##-F##-xxx
- Works for: E02-E07 (direct children with full keys)
- Fails for: E01 (nested with partial keys)

### What Pattern Should Match
For features nested under an epic, we need to match:
1. Full format: `E##-F##-xxx` (works for E02-E07)
2. Partial format: `F##-xxx` (works for E01 nested features)

## Code Analysis

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner.go`
**Lines:** 114-146

The logic:
```go
if epicKey, foundUnder := findEpicAncestor(path, epicFolderMap); foundUnder {
    result, matched := s.patternMatcher.MatchFeaturePattern(info.Name(), epicKey)
    if matched {
        // Found feature
    }
}
```

The `MatchFeaturePattern` function receives:
- `info.Name()` = folder name being tested (e.g., `F01-content-upload-security-implementation`)
- `epicKey` = the epic key found via ancestor (e.g., `E01`)

But the pattern matcher expects the full `E##-F##-xxx` format and ignores the `epicKey` parameter!

## Files That Need Changes

### 1. Primary Fix: `/home/jwwelbor/projects/shark-task-manager/internal/patterns/defaults.go`

Change the feature folder pattern from:
```go
Folder: []string{
    `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
},
```

To (add second pattern for nested features):
```go
Folder: []string{
    // Standard E##-F##-slug format (for direct children)
    `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
    // Nested format: just F##-slug when under an epic (for E01 style)
    `^F(?P<feature_num>\d{2})-(?P<slug>[a-z0-9-]+)$`,
},
```

### 2. Test: `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go`

Add test case for E01-style nested features:
```go
func TestScanNestedFeatures(t *testing.T) {
    // Create test structure:
    // E01-content-ingestion/
    //   epic.md
    //   01-foundation/
    //     F01-feature/
    //     F02-feature/
    // Should find: 1 epic, 2 features
}
```

## Why Other Epics Work

E02-E07 work because their features are direct children AND use the full `E##-F##-xxx` format:
- Feature folder: `E02-F01-ruleset-management-api-implementation`
- Pattern matches: `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$` ✓

## Test Results

### Before Fix
```
shark sync --dry-run --index
DEBUG: FolderScanner.Scan found 14 epics, 0 features
```

### After Fix (Expected)
```
shark sync --dry-run --index
DEBUG: FolderScanner.Scan found 14 epics, 40 features
```
(Or approximately 40-50 features based on E01's nested structure)

## Key Insight

The issue is **NOT** about passing pattern config or discovering nested folders. The scanner ALREADY walks nested folders correctly via `filepath.Walk()`. 

The issue is that the **feature pattern itself is too restrictive** - it only matches features with the full epic key in their name (`E##-F##-xxx`), not the partial format (`F##-xxx`) used by E01's nested features.

When the pattern matcher tries to match `F01-content-upload-security-implementation` against `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-...`, it fails because the folder name doesn't start with `E01`.

The fix is simple: add an additional feature pattern that matches just `F##-xxx` when features are nested under an epic.
