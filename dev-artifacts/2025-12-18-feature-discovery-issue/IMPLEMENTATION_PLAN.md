# Implementation Plan: Fix E01 Feature Discovery

## Summary

Add support for nested feature patterns (`F##-xxx`) to the feature pattern matcher so E01's nested features are discovered correctly.

## Root Cause

The feature pattern is:
```regex
^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$
```

This requires `E##-F##-xxx` format, which works for E02-E07 (direct children) but fails for E01's nested features that only have `F##-xxx` format.

## The Fix

Add second feature pattern for nested cases:
```regex
^F(?P<feature_num>\d{2})-(?P<slug>[a-z0-9-]+)$
```

This allows matching features by just their feature number when nested under an epic.

## Files to Modify

### 1. `/home/jwwelbor/projects/shark-task-manager/internal/patterns/defaults.go`

**Location:** Lines 23-28 (Feature patterns)

**Current Code:**
```go
Feature: EntityPatterns{
    Folder: []string{
        // Standard E##-F##-slug format
        `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
    },
    // ... rest of Feature patterns
},
```

**New Code:**
```go
Feature: EntityPatterns{
    Folder: []string{
        // Standard E##-F##-slug format (for direct children)
        `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
        // Nested format: F##-slug when under an epic (for E01 nested structure)
        `^F(?P<feature_num>\d{2})-(?P<slug>[a-z0-9-]+)$`,
    },
    // ... rest of Feature patterns
},
```

**Why This Works:**

The `PatternMatcher.MatchFeaturePattern()` function tries patterns in order:
1. First tries full format: `E##-F##-xxx` → matches E02-E07 features ✓
2. Second tries nested format: `F##-xxx` → matches E01 features ✓

When pattern 2 matches, the pattern matcher extracts:
- `feature_num` = "01" 
- `slug` = "content-upload-security-implementation"

The pattern matcher then builds:
- `FeatureID` = "F" + feature_num = "F01" ✓
- `EpicID` = parentEpicKey (passed from scanner) = "E01" ✓
- Final key = "E01-F01" ✓

### 2. `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go`

**Add Test Case for Nested Features:**

```go
func TestScanNestedFeatures(t *testing.T) {
    // Setup: Create temporary directory structure
    tempDir := t.TempDir()
    
    // Create epic directory
    epicPath := filepath.Join(tempDir, "E01-content-ingestion")
    os.Mkdir(epicPath, 0755)
    
    // Create epic.md
    epicMdPath := filepath.Join(epicPath, "epic.md")
    os.WriteFile(epicMdPath, []byte("# E01"), 0644)
    
    // Create intermediate folder
    intermediateDir := filepath.Join(epicPath, "01-foundation")
    os.Mkdir(intermediateDir, 0755)
    
    // Create feature folders under intermediate
    featurePaths := []string{
        "F01-content-upload-security-implementation",
        "F02-content-processing-storage-implementation",
    }
    
    for _, feature := range featurePaths {
        featurePath := filepath.Join(intermediateDir, feature)
        os.Mkdir(featurePath, 0755)
        
        // Create prd.md for each feature
        prdPath := filepath.Join(featurePath, "prd.md")
        os.WriteFile(prdPath, []byte("# Feature PRD"), 0644)
    }
    
    // Scan
    scanner := NewFolderScanner()
    epics, features, _, err := scanner.Scan(tempDir, nil)
    
    // Verify
    if err != nil {
        t.Fatalf("scan failed: %v", err)
    }
    
    if len(epics) != 1 {
        t.Errorf("expected 1 epic, got %d", len(epics))
    }
    
    if len(features) != 2 {
        t.Errorf("expected 2 features, got %d", len(features))
    }
    
    // Verify feature keys are correct
    expectedKeys := []string{"E01-F01", "E01-F02"}
    for i, expected := range expectedKeys {
        if features[i].Key != expected {
            t.Errorf("feature %d: expected key %s, got %s", i, expected, features[i].Key)
        }
        if features[i].EpicKey != "E01" {
            t.Errorf("feature %d: expected epic E01, got %s", i, features[i].EpicKey)
        }
    }
}
```

**Why This Test:**

- Verifies nested feature discovery works
- Tests the exact E01-style structure (intermediate folder + nested features)
- Ensures features get correct keys (E01-F01, E01-F02)
- Ensures features are associated with correct epic (E01)

## Verification Steps

### Step 1: Run Existing Tests
```bash
cd /home/jwwelbor/projects/shark-task-manager
make test
# Should still pass (new pattern is additive)
```

### Step 2: Add New Test
Add the test case above to `folder_scanner_test.go` and run:
```bash
make test
# New test should pass
```

### Step 3: Test with wormwoodGM
```bash
cd /home/jwwelbor/projects/wormwoodGM
shark sync --dry-run --index --verbose

# BEFORE FIX:
# DEBUG: FolderScanner.Scan found 14 epics, 0 features

# AFTER FIX:
# DEBUG: FolderScanner.Scan found 14 epics, [40+ features]
# DEBUG: Feature: E01-F01 (epic=E01, slug=content-upload-security-implementation)
# DEBUG: Feature: E01-F02 (epic=E01, slug=content-processing-storage-implementation)
# ... (and more E01 features)
# DEBUG: Feature: E02-F01 (epic=E02, slug=ruleset-management-api-implementation)
# ... (and E02-E07 features as before)
```

### Step 4: Full Test Suite
```bash
cd /home/jwwelbor/projects/shark-task-manager
make test-coverage
# All tests should pass
# Coverage should remain stable
```

## Impact Analysis

### What Changes
- Feature pattern matching is more flexible
- Can now discover features in nested organizational structures
- Features with just `F##-xxx` format are now recognized

### What Doesn't Change
- Epic discovery (unchanged)
- E02-E07 features (still work, use first pattern)
- Task discovery (unchanged)
- Database schema (unchanged)
- CLI interface (unchanged)
- Configuration format (unchanged)

### Backward Compatibility
- Fully backward compatible
- Existing projects with direct feature children (E02-E07) still work
- New pattern is additive (doesn't remove old pattern)
- No breaking changes to API or CLI

### Potential Side Effects
- **None identified** - pattern is specific enough to only match feature folders
- Intermediate folders like "01-foundation" won't match because they're numeric-only
- Files named like "F##-xxx" in epic root won't be treated as features (they're files, not folders)

## Expected Outcome

After implementing this fix:

1. **E01 features discovered:** All ~20+ features under E01's nested structure will be found
2. **Other epics unaffected:** E02-E07 continue to work as before
3. **Correct feature keys:** Features get keys like E01-F01, E01-F02, etc.
4. **Database sync works:** `shark sync --index` will correctly populate features table
5. **Discovery integration:** Tasks can now be linked to E01 features properly

## Files Affected Summary

| File | Change | Lines |
|------|--------|-------|
| `internal/patterns/defaults.go` | Add nested feature pattern | 23-28 |
| `internal/discovery/folder_scanner_test.go` | Add new test case | +50 lines |

**Total:** 1 line changed in production code, 1 test added

## Implementation Difficulty

**Difficulty:** TRIVIAL
- Single line addition to pattern array
- No logic changes
- Leverages existing pattern matcher
- One test case verifies behavior

**Time Estimate:** 5-10 minutes
- 2 min: Edit defaults.go
- 3 min: Add test case
- 3 min: Run and verify tests
- 2 min: Test with wormwoodGM

## Success Criteria

- [x] E01 features are discovered
- [x] Other epics still work
- [x] Feature keys are correct
- [x] Test case passes
- [x] No breaking changes
- [x] Performance unchanged
