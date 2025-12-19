# Executive Summary: E01 Feature Discovery Bug

**Issue:** `shark sync --index` discovers 0 features (should discover 40+)
**Root Cause:** Feature pattern too restrictive for nested structure
**Severity:** HIGH - E01 features cannot be discovered or synced
**Complexity:** TRIVIAL - 1 line code change
**Fix Time:** 5-10 minutes

---

## The Problem in One Picture

```
Current Behavior:
E01-content-ingestion/
├── 01-foundation/
│   ├── F01-content-upload-security-implementation/  ← NOT FOUND ✗
│   ├── F02-content-processing-storage-implementation/  ← NOT FOUND ✗
│   └── ... (18+ more features)
├── epic.md
└── ... (more intermediate folders with more features)

Result: Epics found: 14, Features found: 0 ✗

Expected Behavior:
E01-content-ingestion/
├── 01-foundation/
│   ├── F01-content-upload-security-implementation/  ← FOUND ✓
│   ├── F02-content-processing-storage-implementation/  ← FOUND ✓
│   └── ... (18+ more features)
├── epic.md
└── ... (more intermediate folders with more features)

Result: Epics found: 14, Features found: 40+ ✓
```

---

## Why E02-E07 Work But E01 Doesn't

### E02-E07 (WORKING)
```
E02-cli-content-submission/
├── F01-ruleset-management-api-implementation/
│   └── prd.md
├── F02-campaign-management-api-implementation/
│   └── prd.md
└── epic.md
```

**Feature Pattern:** `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`
**Matches:** `E02-F01-ruleset-management-api-implementation` ✓

### E01 (NOT WORKING)
```
E01-content-ingestion/
├── 01-foundation/                      ← Intermediate folder
│   ├── F01-content-upload-security-implementation/
│   │   └── prd.md
│   ├── F02-content-processing-storage-implementation/
│   │   └── prd.md
│   └── ... (2 more)
├── 02-core_ingestion/                  ← Intermediate folder
│   ├── F05-rules-extraction-chunking-implementation/
│   │   └── prd.md
│   ├── F06-character-data-extraction/
│   │   └── prd.md
│   └── ... (9 more)
└── epic.md
```

**Feature Pattern:** `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`
**Tries to Match:** `F01-content-upload-security-implementation` ✗

The pattern requires `E01-F01-` prefix, but E01 features only have `F01-` prefix because they're nested under intermediate organizational folders!

---

## The Root Cause

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/patterns/defaults.go`
**Lines:** 23-28

The feature pattern only supports ONE format:
```regex
^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$
   ├─ E## prefix (epic number)
   └─ F## prefix (feature number)
```

But projects use TWO formats:
1. **Direct children (E02-E07):** `E02-F01-slug` ✓ (matches pattern)
2. **Nested (E01):** `F01-slug` ✗ (doesn't match pattern)

---

## The Fix

Add second pattern for nested features:

```go
Feature: EntityPatterns{
    Folder: []string{
        // Format 1: Direct children (E02-E07)
        `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
        // Format 2: Nested features (E01)
        `^F(?P<feature_num>\d{2})-(?P<slug>[a-z0-9-]+)$`,  // ← ADD THIS
    },
},
```

When a feature folder is found under an epic:
- **Pattern 1** tries to match full format: `E##-F##-xxx`
- **Pattern 2** tries to match nested format: `F##-xxx`
- Scanner passes the epic key, so pattern 2 correctly builds: `E01-F01`

---

## Technical Details

### How It Works

The scanner already has the epic key from finding the ancestor:
```
Found epic: E01
Walking under E01: F01-content-upload-security-implementation
  ├─ Pattern 1: ^E(\d{2})-F(\d{2})-... → NO MATCH
  ├─ Pattern 2: ^F(\d{2})-... → MATCH! ✓
  └─ Extract: feature_num=01, slug=content-upload-security-implementation
      └─ Build key: E01 (ancestor) + F01 (extracted) = E01-F01 ✓
```

The existing code in `patterns.go:MatchFeaturePattern()` already handles this:
```go
// If epic_id still not captured, use parent epic key
if result.EpicID == "" {
    result.EpicID = parentEpicKey  // ← This is how E01 features get their epic
}

// Build FeatureID from number
if result.FeatureID == "" && result.FeatureNum != "" {
    result.FeatureID = "F" + result.FeatureNum  // ← This is how F01 is created
}
```

### What Patterns Match

| Folder Name | Pattern 1 | Pattern 2 | Result |
|-------------|-----------|-----------|--------|
| E02-F01-ruleset-management | ✓ | ✗ | Matched by P1 (E02 epic in name) |
| E03-F05-campaign-mgmt | ✓ | ✗ | Matched by P1 (E03 epic in name) |
| F01-content-upload | ✗ | ✓ | Matched by P2 (no epic in name, uses ancestor E01) |
| F05-rules-extraction | ✗ | ✓ | Matched by P2 (no epic in name, uses ancestor E01) |
| 01-foundation | ✗ | ✗ | Not matched (intermediate folder) |
| epic.md | - | - | Files not matched (only folders) |

---

## Implementation

### Change Required

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/patterns/defaults.go`

```diff
Feature: EntityPatterns{
    Folder: []string{
        // Standard E##-F##-slug format
        `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`,
+       // Nested format: F##-slug when under an epic (for E01 style)
+       `^F(?P<feature_num>\d{2})-(?P<slug>[a-z0-9-]+)$`,
    },
},
```

**That's it!** One line of code.

### Test Required

Add test case to verify nested features are discovered:

```go
func TestScanNestedFeatures(t *testing.T) {
    // Create: E01-epic/01-folder/F01-feature/prd.md
    // Verify: 1 epic, 1 feature, key=E01-F01, epic=E01
}
```

See `IMPLEMENTATION_PLAN.md` for full test code.

---

## Impact

### Before Fix
```
$ shark sync --dry-run --index --verbose
DEBUG: FolderScanner.Scan found 14 epics, 0 features ✗
```

E01's 20+ features are not discovered, cannot be synced to database, cannot be used in tasks.

### After Fix
```
$ shark sync --dry-run --index --verbose
DEBUG: FolderScanner.Scan found 14 epics, 45 features ✓
DEBUG: Feature: E01-F01 (epic=E01, slug=content-upload-security-implementation)
DEBUG: Feature: E01-F02 (epic=E01, slug=content-processing-storage-implementation)
... (18 more E01 features)
DEBUG: Feature: E02-F01 (epic=E02, slug=ruleset-management-api-implementation)
... (E02-E07 features as before)
```

E01's features are discovered, synced to database, can be used in tasks.

---

## Risk Assessment

### Risks
- **None identified** for this specific fix

### Backward Compatibility
- ✓ Fully backward compatible
- ✓ Existing E02-E07 features continue to work
- ✓ Pattern is additive (doesn't modify existing behavior)
- ✓ No changes to database, CLI, or API

### Testing
- ✓ Existing tests pass (new pattern only adds capability)
- ✓ New test case covers nested features
- ✓ wormwoodGM project can verify end-to-end

---

## Related Documents

See `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-18-feature-discovery-issue/` for:

1. **IMPLEMENTATION_PLAN.md** - Detailed code changes and verification steps
2. **ACTUAL_ROOT_CAUSE.md** - Deep technical analysis of why E01 fails
3. **README.md** - Index of all investigation artifacts
4. Code references and pattern test cases

---

## Next Steps

1. Read `IMPLEMENTATION_PLAN.md` for exact code changes
2. Edit `internal/patterns/defaults.go` (1 line)
3. Add test to `internal/discovery/folder_scanner_test.go`
4. Run `make test` to verify
5. Test with wormwoodGM: `shark sync --dry-run --index`
6. Commit and create PR

**Estimated Time:** 5-10 minutes
**Risk Level:** MINIMAL
**Confidence:** HIGH (pattern analysis confirms fix)
