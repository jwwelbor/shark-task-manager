# Investigation Index: E01 Feature Discovery Issue

**Investigation Date:** 2025-12-18
**Project:** shark-task-manager / wormwoodGM
**Status:** ROOT CAUSE IDENTIFIED, FIX DOCUMENTED, READY FOR IMPLEMENTATION

---

## Quick Links

**Start Here:**
1. [EXECUTIVE_SUMMARY.md](EXECUTIVE_SUMMARY.md) - Visual explanation of the issue and fix (5 min read)

**For Implementation:**
2. [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) - Exact code changes and test cases (10 min read)

**For Deep Technical Understanding:**
3. [ACTUAL_ROOT_CAUSE.md](ACTUAL_ROOT_CAUSE.md) - Detailed analysis of why E01 fails (15 min read)

---

## Document Descriptions

### EXECUTIVE_SUMMARY.md
**Length:** 3 pages
**Audience:** Managers, developers, reviewers
**Purpose:** Quick overview of the issue, cause, fix, and impact

Contains:
- Problem visualization
- Why E02-E07 work but E01 doesn't
- The one-line fix
- Impact before/after
- Risk assessment

**Read this if you:** Need a quick understanding of what's wrong and how to fix it

---

### IMPLEMENTATION_PLAN.md
**Length:** 8 pages
**Audience:** Developers implementing the fix
**Purpose:** Exact specifications for code changes and testing

Contains:
- Root cause explanation
- Files to modify (with exact locations)
- Code changes (before/after)
- Test case code (copy-paste ready)
- Verification steps
- Impact analysis
- Success criteria

**Read this if you:** Are implementing the fix or reviewing the code changes

---

### ACTUAL_ROOT_CAUSE.md
**Length:** 6 pages
**Audience:** Developers, architects, QA engineers
**Purpose:** Deep technical analysis of the issue

Contains:
- Directory structure comparison (E02 vs E01)
- How the scanner walks directories
- The bug in folder_scanner.go (lines 114-146)
- Why the pattern doesn't match E01 features
- Pattern analysis with test cases
- Code trace showing exactly where it fails
- Files that need changes

**Read this if you:** Want to understand the technical details or verify the root cause

---

## The Issue (One Sentence)

The feature pattern `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-...` only matches features with full `E##-F##-xxx` format (works for E02-E07) but fails for E01's nested features which only have `F##-xxx` format.

## The Fix (One Sentence)

Add a second pattern `^F(?P<feature_num>\d{2})-(?P<slug>[a-z0-9-]+)$` to match features nested under an epic without the epic number in their folder name.

## Files to Modify

| File | Change | Difficulty |
|------|--------|-----------|
| `/home/jwwelbor/projects/shark-task-manager/internal/patterns/defaults.go` | Add 1 line pattern | TRIVIAL |
| `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go` | Add 50-line test | TRIVIAL |

---

## Evidence

### Before Fix
```bash
$ cd /home/jwwelbor/projects/wormwoodGM
$ shark sync --dry-run --index --verbose
DEBUG: FolderScanner.Scan found 14 epics, 0 features
```

### Expected After Fix
```bash
$ cd /home/jwwelbor/projects/wormwoodGM
$ shark sync --dry-run --index --verbose
DEBUG: FolderScanner.Scan found 14 epics, 45 features
DEBUG: Feature: E01-F01 (epic=E01, slug=content-upload-security-implementation)
DEBUG: Feature: E01-F02 (epic=E01, slug=content-processing-storage-implementation)
... (more E01 features)
DEBUG: Feature: E02-F01 (epic=E02, slug=ruleset-management-api-implementation)
... (E02-E07 features as before)
```

---

## Investigation Timeline

### What We Tested
1. Examined E01 directory structure vs E02-E07 ✓
2. Identified E01 has nested intermediate folders (01-foundation, 02-core_ingestion, etc.) ✓
3. Identified E02-E07 have features as direct children ✓
4. Analyzed FolderScanner code (folder_scanner.go:114-146) ✓
5. Examined pattern matching code (patterns.go) ✓
6. Analyzed default patterns (defaults.go) ✓
7. Tested pattern matching manually ✓

### What We Found
- Scanner correctly walks nested directories ✓
- Scanner correctly finds epic ancestors ✓
- Pattern matching fails for `F##-xxx` format ✓
- Pattern matcher expects `E##-F##-xxx` format ✓
- Solution: Add second pattern ✓

---

## Key Insights

1. **Not a traversal issue:** The scanner already walks nested directories correctly via `filepath.Walk()`

2. **Pattern specificity:** The default feature pattern is too specific - it assumes all features have the epic number in their folder name

3. **Design works for E01:** The `MatchFeaturePattern()` function already accepts `parentEpicKey` parameter to handle features without embedded epic ID

4. **Simple fix:** Adding a second pattern leverages existing infrastructure perfectly

5. **Zero breaking changes:** New pattern is additive, doesn't affect E02-E07 or any other functionality

---

## Implementation Checklist

- [ ] Read EXECUTIVE_SUMMARY.md
- [ ] Read IMPLEMENTATION_PLAN.md
- [ ] Edit internal/patterns/defaults.go (line 23-28)
- [ ] Add test to internal/discovery/folder_scanner_test.go
- [ ] Run `make test`
- [ ] Test with wormwoodGM: `shark sync --dry-run --index --verbose`
- [ ] Commit changes
- [ ] Create pull request
- [ ] Verify with team

---

## Related Code Files

### Files Examined During Investigation
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner.go` - Scanner logic
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/patterns.go` - Pattern matching
- `/home/jwwelbor/projects/shark-task-manager/internal/patterns/defaults.go` - Default patterns
- `/home/jwwelbor/projects/wormwoodGM/docs/plan/E01-content-ingestion/` - Actual project structure
- `/home/jwwelbor/projects/wormwoodGM/docs/plan/E02-cli-content-submission/` - Working example
- `/home/jwwelbor/projects/wormwoodGM/.sharkconfig.json` - Project configuration

### Files to Modify
- `/home/jwwelbor/projects/shark-task-manager/internal/patterns/defaults.go` (PRIMARY)
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go` (TEST)

---

## Investigation Artifacts

All files are located in:
```
/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-18-feature-discovery-issue/
```

Files in this directory:
- `INDEX.md` (this file)
- `EXECUTIVE_SUMMARY.md` - Visual explanation
- `IMPLEMENTATION_PLAN.md` - Code changes
- `ACTUAL_ROOT_CAUSE.md` - Technical analysis
- `README.md` - Quick reference
- `analysis/` - Detailed analysis documents

---

## For Quick Review

**Read this order:**
1. EXECUTIVE_SUMMARY.md (5 min) - Understand the issue
2. IMPLEMENTATION_PLAN.md (10 min) - See the fix
3. ACTUAL_ROOT_CAUSE.md (15 min) - Verify the analysis

**Total time:** 30 minutes to fully understand the issue and fix

---

## Questions?

The investigation is thorough and complete. All documents are self-contained and provide context for understanding the issue and implementing the fix.

For implementation, refer to IMPLEMENTATION_PLAN.md which has:
- Exact line numbers
- Copy-paste ready code
- Test case code
- Verification steps
- Success criteria
