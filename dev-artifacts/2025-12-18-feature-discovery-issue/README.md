# Shark Feature Discovery Issue - Complete Investigation

**Date:** 2025-12-18
**Project:** wormwoodGM / shark-task-manager
**Issue:** `shark sync --index` discovers 0 features
**Status:** ROOT CAUSE IDENTIFIED ✓

## Quick Summary

Feature discovery fails because the FolderScanner is not receiving the project-specific pattern configuration from `.sharkconfig.json`. It uses hard-coded default patterns instead, which don't account for the nested organizational folder structure used in wormwoodGM.

**Root Cause Location:** `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go:55`

## Investigation Artifacts

### 1. DEBUG_SUMMARY.md
**Comprehensive overview of the entire investigation**
- Executive summary
- Investigation process with screenshots
- Directory structure comparison
- Code analysis
- Solution summary

**Start here for quick understanding of the issue.**

### 2. FINAL_ROOT_CAUSE.md
**Detailed technical analysis of the root cause**
- Problem statement
- Directory structure analysis (expected vs actual)
- Feature pattern configuration
- Code trace showing why features aren't found
- Critical discovery explaining the actual bug
- Root cause summary
- Files affected

**Read this for technical deep-dive.**

### 3. VERIFICATION_AND_FIX.md
**How to verify the bug and implement the fix**
- Current behavior (broken)
- Expected behavior (fixed)
- Why current code fails
- The fix (step-by-step)
- Testing approach
- Expected feature count in wormwoodGM
- Potential side effects
- Implementation order

**Use this to implement the fix.**

### 4. CODE_REFERENCES.md
**Complete file paths and code snippets**
- Root cause location with code
- Pattern initialization chain
- Pattern override logic
- Feature discovery logic
- Pattern matching implementation
- Discovery process entry point
- Test files (existing)
- Configuration file
- Project structure
- Sync engine integration
- Command entry point
- Summary of changes needed
- Verification steps

**Reference guide for developers.**

### 5. StructureComparison.md
**Directory structure comparison**
- Expected structure (what works)
- wormwoodGM actual structure (what breaks)
- Why features aren't discovered
- Discovery algorithm analysis
- Feature pattern expectations
- Why features aren't discovered (detailed trace)

**Useful for understanding the organizational pattern differences.**

### 6. ROOT_CAUSE_ANALYSIS.md
**Initial investigation document**
- Problem summary
- Root cause (hierarchical mismatch)
- Discovery code analysis
- Feature pattern expectations
- Verification section

**Early investigation notes.**

### 7. regex_test.go
**Test program for regex pattern matching**
- Standalone Go program for testing patterns
- Can be compiled and run to verify pattern behavior

**For pattern validation.**

## The Bug in One Sentence

The FolderScanner.Scan() function is called with `nil` pattern overrides instead of the `.sharkconfig.json` patterns, causing it to use hard-coded defaults that don't match wormwoodGM's nested feature organization.

## The Fix in One Sentence

Pass `e.patternConfig` instead of `nil` to `scanner.Scan()` at line 55 of `internal/sync/discovery.go`.

## Quick Reference

### Files Modified
1. `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go` (Line 55) - PRIMARY FIX
2. `/home/jwwelbor/projects/shark-task-manager/internal/sync/engine.go` - Add pattern config field/loading
3. `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner_test.go` - Add test for nested features

### Evidence
- Running: `shark sync --dry-run --index --verbose`
- Shows: `DEBUG: FolderScanner.Scan found 14 epics, 0 features`
- Expected: Should find ~40-50 features

### Impact
- Projects with nested feature organization (like wormwoodGM) cannot use discovery
- Workaround: Use `shark sync --pattern=task` for file-based sync instead

## Testing

### Unit Test Case Needed
Add test in `folder_scanner_test.go` for nested feature organization:
- Create intermediate folders (`01-foundation/`)
- Create feature folders under intermediate (`F01-...`, `F02-...`)
- Verify features are discovered despite nesting

### Integration Test
```bash
cd /home/jwwelbor/projects/wormwoodGM
shark sync --dry-run --index --verbose
# Before fix: 0 features
# After fix: 40-50 features (actual count)
```

## Files for Reference

### Key Locations in Codebase
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/folder_scanner.go` - Feature discovery algorithm
- `/home/jwwelbor/projects/shark-task-manager/internal/discovery/patterns.go` - Pattern matching
- `/home/jwwelbor/projects/shark-task-manager/internal/sync/discovery.go` - **BUG IS HERE**
- `/home/jwwelbor/projects/shark-task-manager/internal/sync/engine.go` - Sync engine
- `/home/jwwelbor/projects/wormwoodGM/.sharkconfig.json` - Project configuration

### Dates and References
- **Debug Investigation Date:** 2025-12-18
- **Project Working Directory:** `/home/jwwelbor/projects/wormwoodGM`
- **Shark Code Location:** `/home/jwwelbor/projects/shark-task-manager`
- **Workspace:** `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-18-feature-discovery-issue/`

## Next Steps

1. ✓ Identify root cause (DONE)
2. ✓ Document investigation (DONE)
3. [ ] Implement fix (in discovery.go line 55)
4. [ ] Update SyncEngine to store pattern config
5. [ ] Load patterns in discovery workflow
6. [ ] Add unit test for nested features
7. [ ] Test with wormwoodGM project
8. [ ] Run full test suite
9. [ ] Create PR

## Contact / Notes

All analysis documents are in this directory. The investigation is complete and ready for implementation.
