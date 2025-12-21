# Verification and Fix Details

## How to Verify the Bug

### Current Behavior (Broken)

```bash
cd /home/jwwelbor/projects/wormwoodGM
/home/jwwelbor/projects/shark-task-manager/bin/shark sync --dry-run --index --verbose
```

Output:
```
DEBUG: FolderScanner.Scan found 14 epics, 0 features
```

The FolderScanner finds all 14 epics but ZERO features.

### Expected Behavior (Fixed)

After the fix, the same command should output something like:
```
DEBUG: FolderScanner.Scan found 14 epics, 48 features
DEBUG: Epic: E01 (content-ingestion)
DEBUG: Feature: E01-F01 (epic=E01, slug=content-upload-security-implementation)
DEBUG: Feature: E01-F02 (epic=E01, slug=content-processing-storage-implementation)
... (etc)
```

Where ~48 features represents the actual count of F##-* folders in the wormwood project.

---

## Why the Current Code Fails

### Code Path Analysis

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go`

**Lines 54-55:**
```go
scanner := discovery.NewFolderScanner()
folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
                                                                                                    ↑
                                                                        PASSING nil INSTEAD OF CONFIG
```

### Pattern Initialization Chain

1. **sync/discovery.go:54** Creates FolderScanner
2. **discovery/folder_scanner.go:18-22** NewFolderScanner() initializes with:
   ```go
   func NewFolderScanner() *FolderScanner {
       return &FolderScanner{
           patternMatcher: NewPatternMatcher(patterns.GetDefaultPatterns()),
       }
   }
   ```
3. **patterns/patterns.go** GetDefaultPatterns() returns hard-coded default patterns
4. These defaults might not match wormwood's config patterns!

### What Should Happen

1. Load config from `.sharkconfig.json` ✓ (sync/discovery.go:152-158 in parent code)
2. Create sync engine with config ✓
3. Pass config to FolderScanner ✗ (MISSING!)
4. FolderScanner uses project-specific patterns ✗ (DOESN'T HAPPEN)

---

## The Fix

### Step 1: Add Pattern Config to SyncEngine

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/sync/engine.go`

The SyncEngine needs to store the pattern config. In the New functions or as a field:

```go
type SyncEngine struct {
    // ... existing fields ...
    patternConfig *patterns.PatternConfig  // ← ADD THIS
}
```

### Step 2: Pass Config to FolderScanner.Scan()

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go` (Line 55)

Change from:
```go
folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
```

To:
```go
folderEpics, folderFeatures, _, err := scanner.Scan(
    filepath.Join(e.docsRoot, opts.FolderPath),
    e.patternConfig,  // ← PASS THE CONFIG
)
```

### Step 3: Load Pattern Config in Discovery

If pattern config isn't already loaded, load it in `runDiscovery()`:

```go
func (e *SyncEngine) runDiscovery(ctx context.Context, opts SyncOptions) (*DiscoveryReport, error) {
    // ... existing code ...

    // Load pattern config if not already loaded
    if e.patternConfig == nil {
        configManager := config.NewManager(findConfigPath())
        cfg, err := configManager.Load()
        if err == nil && cfg != nil && cfg.Patterns != nil {
            e.patternConfig = cfg.Patterns
        }
    }

    // ... rest of function ...
}
```

---

## Testing the Fix

### Unit Test to Add

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go`

Add a test case for nested feature organization:

```go
{
    name: "scan nested features under organizational folders",
    setupFunc: func(t *testing.T) string {
        tmpDir := t.TempDir()
        epicDir := filepath.Join(tmpDir, "E01-epic")

        // Create intermediate organizational folder
        intermediateDir := filepath.Join(epicDir, "01-foundation")

        // Create feature folders under intermediate
        require.NoError(t, os.MkdirAll(filepath.Join(intermediateDir, "F01-feature-one"), 0755))
        require.NoError(t, os.MkdirAll(filepath.Join(intermediateDir, "F02-feature-two"), 0755))

        // Create PRD files in intermediate folder (wormwood style)
        require.NoError(t, os.WriteFile(
            filepath.Join(intermediateDir, "PRD_F01-feature-one.md"),
            []byte("# PRD"), 0644))

        return tmpDir
    },
    expectedEpics:    1,
    expectedFeatures: 2,  // Should find both F01 and F02
    validateFeatures: func(t *testing.T, features []FolderFeature) {
        keys := make(map[string]bool)
        for _, feature := range features {
            keys[feature.Key] = true
        }
        assert.True(t, keys["E01-F01"], "Should find F01 even under 01-foundation")
        assert.True(t, keys["E01-F02"], "Should find F02 even under 01-foundation")
    },
}
```

### Integration Test with wormwoodGM

```bash
cd /home/jwwelbor/projects/wormwoodGM

# Before fix: shows 0 features
/home/jwwelbor/projects/shark-task-manager/bin/shark sync --dry-run --index 2>&1 | grep "features"

# After fix: shows actual feature count
/home/jwwelbor/projects/shark-task-manager/bin/shark sync --dry-run --index 2>&1 | grep "features"
```

---

## Expected Feature Count in wormwoodGM

Based on directory structure:
- E01: F01-F04 under 01-foundation, F05-F10, F17 under 02-core_ingestion = ~14 features
- E02: Multiple features
- E03-E10: Various features
- Total: Approximately 40-50 features

After fix, the scan should report this actual count instead of 0.

---

## Potential Side Effects

1. **Performance:** Loading and passing pattern config adds minimal overhead
2. **Backward Compatibility:** Passing config doesn't break existing functionality
3. **Pattern Matching:** Features should now be found in nested structures
4. **Related Docs:** PRD files at parent level (like wormwood) may need path adjustment

---

## Related Code Sections

### Pattern Config Loading (sync/engine.go or discovery.go)
Need to ensure pattern config is available when needed.

### Default Patterns (patterns/patterns.go)
Check if default patterns are sufficient or need updating.

### Feature File Matching (discovery/folder_scanner.go:218-255)
The `findPrdFile()` function looks for files in the feature folder. In wormwood, PRD files are at the intermediate level. This might need separate handling.

---

## Recommended Implementation Order

1. ✓ Identify root cause (DONE - pattern config not passed)
2. Add pattern config field to SyncEngine
3. Load pattern config in runDiscovery()
4. Pass pattern config to FolderScanner.Scan()
5. Add unit test for nested features
6. Test with wormwoodGM project
7. Verify feature discovery works
8. Check that related docs are properly indexed

