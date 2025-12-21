# Code References and File Locations

## Root Cause Location

### Primary Bug
**File:** `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go`
**Lines:** 54-58
**Issue:** Pattern config not passed to FolderScanner

```go
// CURRENT (BROKEN):
scanner := discovery.NewFolderScanner()
folderEpics, folderFeatures, _, err := scanner.Scan(filepath.Join(e.docsRoot, opts.FolderPath), nil)
                                                                                                  ↑
                                          SHOULD BE: e.patternConfig (not nil)
```

**Should be:**
```go
scanner := discovery.NewFolderScanner()
folderEpics, folderFeatures, _, err := scanner.Scan(
    filepath.Join(e.docsRoot, opts.FolderPath),
    e.patternConfig,  // ← Pass project-specific patterns
)
```

---

## Related Code Sections

### 1. Pattern Initialization Chain

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner.go`
**Lines:** 18-22

```go
// NewFolderScanner creates a new folder scanner with default patterns
func NewFolderScanner() *FolderScanner {
    return &FolderScanner{
        patternMatcher: NewPatternMatcher(patterns.GetDefaultPatterns()),
    }
}
```

This uses hard-coded defaults. The Scan() function accepts pattern overrides but they're not passed from discovery.go.

---

### 2. Pattern Override Logic

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner.go`
**Lines:** 53-59

```go
// Scan walks directory tree and discovers epics/features
func (s *FolderScanner) Scan(docsRoot string, patternOverrides *patterns.PatternConfig) (
    []FolderEpic, []FolderFeature, ScanStats, error) {

    // Override patterns if provided
    if patternOverrides != nil {
        s.patternMatcher = NewPatternMatcher(patternOverrides)
    }
    // ... rest of function ...
}
```

The condition at line 57 would work IF patterns were passed, but they aren't.

---

### 3. Feature Discovery Logic

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner.go`
**Lines:** 114-146

```go
// Try to match as feature folder (within epic, possibly nested)
// Check if this directory is under an epic (any ancestor)
if epicKey, foundUnder := findEpicAncestor(path, epicFolderMap); foundUnder {
    // Try to match feature pattern
    result, matched := s.patternMatcher.MatchFeaturePattern(info.Name(), epicKey)
    if matched {
        // Build feature object
        featureKey := result.EpicID + "-" + result.FeatureID
        if result.EpicID == "" {
            // If pattern didn't extract epic ID, use the ancestor epic key
            featureKey = epicKey + "-" + result.FeatureID
        }

        feature := FolderFeature{
            Key:     featureKey,
            EpicKey: epicKey,
            Slug:    result.FeatureSlug,
            Path:    path,
        }

        // Find PRD file
        prdPath, prdFilename := s.findPrdFile(path)
        if prdPath != nil {
            feature.PrdPath = prdPath
            stats.FilesAnalyzed++
        }

        // Catalog related documents (exclude PRD file)
        relatedDocs := s.catalogRelatedDocs(path, prdFilename)
        feature.RelatedDocs = relatedDocs
        stats.FilesAnalyzed += len(relatedDocs)

        features = append(features, feature)
    }
}
```

This code uses `s.patternMatcher` which was initialized with default patterns (not project patterns).

---

### 4. Pattern Matching

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/discovery/patterns.go`
**Lines:** 130-186

```go
// MatchFeaturePattern tries to match input against feature patterns (first match wins)
func (m *PatternMatcher) MatchFeaturePattern(input, parentEpicKey string) (FeatureMatchResult, bool) {
    for _, re := range m.featureFolderRegexes {
        matches := re.FindStringSubmatch(input)
        if matches == nil {
            continue
        }

        // Extract named capture groups
        result := FeatureMatchResult{}
        names := re.SubexpNames()

        for i, name := range names {
            if i == 0 || name == "" {
                continue
            }
            if i < len(matches) {
                switch name {
                case "epic_id":
                    result.EpicID = matches[i]
                case "feature_id":
                    result.FeatureID = matches[i]
                case "feature_slug", "slug":
                    result.FeatureSlug = matches[i]
                case "epic_num":
                    result.EpicNum = matches[i]
                case "feature_num", "number":
                    result.FeatureNum = matches[i]
                }
            }
        }

        // Build EpicID from epic_num if not set
        if result.EpicID == "" && result.EpicNum != "" {
            result.EpicID = "E" + result.EpicNum
        }

        // Build FeatureID from number if not set
        if result.FeatureID == "" && result.FeatureNum != "" {
            result.FeatureID = "F" + result.FeatureNum
        }

        // If epic_id still not captured, use parent epic key
        if result.EpicID == "" {
            result.EpicID = parentEpicKey
        }

        // Validate required fields
        if result.EpicID == "" || result.FeatureID == "" {
            continue
        }

        return result, true
    }

    return FeatureMatchResult{}, false
}
```

This uses `m.featureFolderRegexes` which are compiled from patterns at initialization.

---

### 5. Discovery Process Entry Point

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go`
**Lines:** 14-102

```go
// runDiscovery orchestrates the discovery workflow
func (e *SyncEngine) runDiscovery(ctx context.Context, opts SyncOptions) (*DiscoveryReport, error) {
    report := &DiscoveryReport{
        Warnings: []string{},
    }

    // Build discovery options
    discoveryOpts := discovery.DiscoveryOptions{
        DocsRoot:        e.docsRoot,
        IndexPath:       filepath.Join(e.docsRoot, opts.FolderPath, "epic-index.md"),
        Strategy:        mapDiscoveryStrategy(opts.DiscoveryStrategy),
        DryRun:          opts.DryRun,
        ValidationLevel: mapValidationLevel(opts.ValidationLevel),
    }

    // ... index parsing code ...

    // Step 2: Scan folder structure
    scanner := discovery.NewFolderScanner()  // ← Line 54: Creates with default patterns
    folderEpics, folderFeatures, _, err := scanner.Scan(
        filepath.Join(e.docsRoot, opts.FolderPath),
        nil,  // ← Line 55: BUG - should pass e.patternConfig
    )
    if err != nil {
        return nil, fmt.Errorf("failed to scan folders: %w", err)
    }

    // ... rest of function ...
}
```

---

## Test Files

### Existing Tests (Show Expected Behavior)

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go`
**Key Test:** `TestFolderScanner_Scan` (line 14)

Shows tests with flat structure:
- E04-F01-database-schema (direct child of epic)
- E04-F07-initialization-sync (direct child of epic)

Tests pass because structure matches expected pattern.

**Lines 105-128** - Example that works:
```go
{
    name: "scan epic with features (E##-F##-feature-slug)",
    setupFunc: func(t *testing.T) string {
        tmpDir := t.TempDir()
        epicDir := filepath.Join(tmpDir, "E04-task-mgmt-cli-core")
        require.NoError(t, os.MkdirAll(epicDir, 0755))
        require.NoError(t, os.MkdirAll(filepath.Join(epicDir, "E04-F01-database-schema"), 0755))
        require.NoError(t, os.MkdirAll(filepath.Join(epicDir, "E04-F07-initialization-sync"), 0755))
        return tmpDir
    },
    expectedEpics:    1,
    expectedFeatures: 2,  // ✓ Features found
    // ...
}
```

---

## Configuration File

**File:** `/home/jwwelbor/projects/wormwoodGM/.sharkconfig.json`
**Relevant Section:** Lines 20-30

```json
"feature": {
  "file": [
    "^prd\\.md$",
    "^PRD_F(?P<number>\\d{2})-(?P<slug>.+)\\.md$",
    "^(?P<slug>[a-z0-9-]+)\\.md$"
  ],
  "folder": [
    "^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$",
    "^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"
  ]
}
```

Pattern 2 in `folder` array: `^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$`
- Should match: `F01-content-upload-security-implementation` ✓
- Should match: `F05-rules-extraction-chunking-implementation` ✓

---

## Project Structure

### wormwoodGM Feature Organization
```
/home/jwwelbor/projects/wormwoodGM/
├── .sharkconfig.json                                       ← Config file
└── docs/plan/
    ├── E01-content-ingestion/                             ✓ Epic found
    │   ├── 01-foundation/                                 ✗ Not matched
    │   │   ├── F01-content-upload-security-implementation/  ✗ NOT FOUND
    │   │   ├── F02-content-processing-storage-implementation/
    │   │   ├── F03-ip-scanning-risk-management-implementation/
    │   │   ├── F04-multi-index-storage-system-implementation/
    │   │   ├── PRD_F01-content-upload-security.md
    │   │   ├── PRD_F02-content-processing-storage.md
    │   │   ├── PRD_F03-ip-scanning-risk-management.md
    │   │   └── PRD_F04-multi-index-storage-system.md
    │   ├── 02-core_ingestion/                             ✗ Not matched
    │   │   ├── F05-rules-extraction-chunking-implementation/
    │   │   ├── F06-character-data-extraction/
    │   │   ├── F07-scenario-adventure-builder-implementation/
    │   │   ├── F08-tables-equipment-indexer/
    │   │   ├── F09-lore-narrative-extractor-implementation/
    │   │   ├── F10-visual-asset-processor-implementation/
    │   │   └── F17-toc-router-sectioning-implementation/
    │   ├── 03-integration/
    │   ├── 04-platform/
    │   ├── 05-missed_features/
    │   └── epic.md
    ├── E02-cli-content-submission/
    ├── E03-voice-integration/
    ├── ... (more epics)
```

---

## Sync Engine Integration

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/sync/engine.go`

SyncEngine needs pattern config field to store and pass to scanner.

### Current Structure (Simplified)
```go
type SyncEngine struct {
    db           *sql.DB
    docsRoot     string
    epicRepo     *repository.EpicRepository
    featureRepo  *repository.FeatureRepository
    scanner      *FileScanner
    // ... other fields ...
    // MISSING: patternConfig *patterns.PatternConfig
}
```

### Constructor (NewSyncEngine)
Needs to load or accept pattern config to store it.

---

## Command Entry Point

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/sync.go`
**Lines:** 112-191 (runSync function)

```go
func runSync(cmd *cobra.Command, args []string) error {
    // ... flag parsing ...

    // Line 170: Set EnableDiscovery
    opts := sync.SyncOptions{
        // ...
        EnableDiscovery:   syncIndex,  // ← true when --index flag used
        // ...
    }

    // Line 181: Create engine
    engine, err := sync.NewSyncEngineWithPatterns(dbPath, patterns)

    // Line 191: Run sync (calls discovery if EnableDiscovery is true)
    syncReport, err := engine.Sync(ctx, opts)
}
```

---

## Summary of Changes Needed

### 1. SyncEngine (engine.go)
- Add `patternConfig *patterns.PatternConfig` field
- Load patterns in constructor or discovery method

### 2. Discovery (discovery.go)
- Line 55: Change `nil` to `e.patternConfig`
- Ensure patterns are loaded before scanning

### 3. Tests (folder_scanner_test.go)
- Add test case for nested features
- Verify pattern config passing works

---

## Verification Steps

### Before Fix
```bash
cd /home/jwwelbor/projects/wormwoodGM
/home/jwwelbor/projects/shark-task-manager/bin/shark sync --dry-run --index --verbose
# Output: DEBUG: FolderScanner.Scan found 14 epics, 0 features
```

### After Fix
```bash
cd /home/jwwelbor/projects/wormwoodGM
/home/jwwelbor/projects/shark-task-manager/bin/shark sync --dry-run --index --verbose
# Output: DEBUG: FolderScanner.Scan found 14 epics, 48 features (actual count varies)
```

