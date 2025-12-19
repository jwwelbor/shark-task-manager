# Directory Structure Comparison

## Expected Structure (Tests Show This Works)

```
docs/plan/
├── E04-task-mgmt-cli-core/           ✓ Epic
│   ├── E04-F01-database-schema/      ✓ Feature (DIRECT CHILD of epic)
│   ├── E04-F07-initialization-sync/  ✓ Feature (DIRECT CHILD of epic)
│   ├── prd.md                        (Optional, in feature folder)
│   └── epic.md
```

**Discovery Result:** 1 epic, 2 features ✓

---

## WormwoodGM Actual Structure

```
docs/plan/
├── E01-content-ingestion/                                 ✓ Epic found
│   ├── 01-foundation/                                    ✗ Intermediate folder
│   │   ├── F01-content-upload-security-implementation/  ✗ NOT FOUND (hidden)
│   │   ├── F02-content-processing-storage-implementation/
│   │   ├── F03-ip-scanning-risk-management-implementation/
│   │   ├── F04-multi-index-storage-system-implementation/
│   │   ├── PRD_F01-content-upload-security.md           (At parent level!)
│   │   ├── PRD_F02-content-processing-storage.md
│   │   ├── PRD_F03-ip-scanning-risk-management.md
│   │   └── PRD_F04-multi-index-storage-system.md
│   ├── 02-core_ingestion/                               ✗ Intermediate folder
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
```

**Discovery Result:** 1 epic, 0 features ✗

---

## Why Features Are Not Discovered

### Discovery Algorithm (folder_scanner.go, line 76-149)

1. Walk through all directories in docs/plan/
2. For each directory:
   a. Check if it matches epic pattern → YES for `E01-content-ingestion`
   b. Check if it's under an epic ancestor → Will be checked for child dirs
   c. Check if it matches feature pattern → Compare folder name against patterns

### Feature Pattern Matching (patterns.go, line 131-186)

Patterns from .sharkconfig.json:
```json
"feature": {
  "folder": [
    "^E(?P<epic_num>\\d{2})-F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$",
    "^F(?P<number>\\d{2})-(?P<slug>[a-z0-9-]+)$"
  ]
}
```

**Trace for `01-foundation` folder:**
1. findEpicAncestor() finds `E01` ✓
2. MatchFeaturePattern("01-foundation", "E01"):
   - Pattern 1: `^E\d{2}-F\d{2}-...` → "01-foundation" starts with "01-", not "E\d{2}-F" → NO
   - Pattern 2: `^F\d{2}-...` → "01-foundation" starts with "0", not "F" → NO
   - Result: NO MATCH ✗

**Consequence:** `01-foundation` is skipped but walking continues (returns nil, not SkipDir)

**Trace for `F01-content-upload-security-implementation` folder (nested under 01-foundation):**
1. findEpicAncestor() walks up:
   - Checks `...F01-content-upload-security-implementation` → not in epicFolderMap
   - Checks `...01-foundation` → not in epicFolderMap
   - Checks `...E01-content-ingestion` → IS in epicFolderMap! ✓ Returns ("E01", true)
2. MatchFeaturePattern("F01-content-upload-security-implementation", "E01"):
   - Pattern 1: `^E01-F\d{2}-...` → "F01-..." does NOT start with "E01-F" → NO
   - Pattern 2: `^F\d{2}-...` → "F01-..." starts with "F01-" → YES! ✓

**Expected Result:** Should match and be discovered!

---

## The Real Problem: Debug Output

Looking at folder_scanner.go line 155-162, there's debug output:

```go
// DEBUG: Print what was found
fmt.Printf("DEBUG: FolderScanner.Scan found %d epics, %d features\n", len(epics), len(features))
```

This debug output should show what's being found. When running:
```
shark sync --dry-run --verbose
```

The output shows:
```
Files scanned:      0
New tasks imported: 0
```

This suggests either:
1. The discovery code is not being called by the sync engine
2. Or the scanner is finding the features but they're being filtered out somewhere else

---

## Investigation Needed

1. Verify if FolderScanner.Scan is actually being called
2. Check if the debug output is being produced
3. Trace the full path from sync command → discovery → feature matching
4. Look for filtering that might exclude features after discovery

