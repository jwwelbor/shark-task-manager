# Bug Fix Summary: Feature Update --path Not Displaying custom_folder_path

**Date:** 2026-01-02
**Task:** T-E07-F18-001
**Status:** Fixed
**Approach:** Test-Driven Development (TDD)

---

## Problem Statement

Users reported that `shark feature update --path` command appeared to do nothing - it ran without errors but didn't show the updated custom path when running `shark feature get`.

## Root Cause Analysis

The bug analysis document suspected the issue was in the repository layer or command handler, but investigation revealed:

1. **Repository Layer:** Working correctly ✅
   - The `Update()` method in `feature_repository.go` correctly includes `custom_folder_path` in the SQL UPDATE statement
   - Created test `TestFeatureRepository_UpdateCustomPath` which passed immediately, confirming repository works

2. **Command Handler:** Working correctly ✅
   - The `runFeatureUpdate` function correctly validates the path and sets `feature.CustomFolderPath`
   - Database verification showed the field WAS being updated

3. **Actual Bug:** Display layer missing field ❌
   - The `runFeatureGet` function was NOT including `custom_folder_path` in its JSON output
   - The `renderFeatureDetails` function was NOT including it in the formatted text output

## TDD Workflow

### 1. Write Test First
Created `TestFeatureRepository_UpdateCustomPath` in `/home/jwwelbor/projects/shark-task-manager/internal/repository/feature_repository_test.go`:
- Creates a test feature without custom path
- Updates it with `custom_folder_path = "docs/custom/test-location"`
- Verifies the database field is updated
- Verifies retrieval returns the correct value

### 2. Run Test - Expected to Fail
Test **passed immediately**, indicating the repository layer was working correctly.

### 3. Investigation
Manual testing revealed:
```bash
# Update works
./bin/shark feature update E07-F20 --path="docs/custom/test"

# Database is updated
sqlite3 shark-tasks.db "SELECT custom_folder_path FROM features WHERE key='E07-F20';"
# Result: docs/custom/test ✅

# But feature get doesn't show it
./bin/shark feature get E07-F20 --json | jq '.custom_folder_path'
# Result: null ❌
```

### 4. Implement Fix
**File:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/feature.go`

**Change 1:** Add `custom_folder_path` to JSON output (lines 561-582)
```go
// Output as JSON if requested
if cli.GlobalConfig.JSON {
    result := map[string]interface{}{
        "id":                 feature.ID,
        "epic_id":            feature.EpicID,
        "key":                feature.Key,
        "title":              feature.Title,
        "description":        feature.Description,
        "status":             feature.Status,
        "status_source":      statusSource,
        "status_override":    feature.StatusOverride,
        "progress_pct":       feature.ProgressPct,
        "path":               dirPath,
        "filename":           filename,
        "custom_folder_path": feature.CustomFolderPath,  // ← ADDED
        "created_at":         feature.CreatedAt,
        "updated_at":         feature.UpdatedAt,
        "tasks":              tasks,
        "status_breakdown":   statusBreakdown,
        "related_documents":  relatedDocs,
    }
    return cli.OutputJSON(result)
}
```

**Change 2:** Add `custom_folder_path` to formatted text output (lines 664-666)
```go
if feature.CustomFolderPath != nil && *feature.CustomFolderPath != "" {
    info = append(info, []string{"Custom Folder Path", *feature.CustomFolderPath})
}
```

### 5. Verify Fix

**Repository test:** ✅ Still passes
```bash
go test -v ./internal/repository -run TestFeatureRepository_UpdateCustomPath
# PASS
```

**All repository tests:** ✅ Pass
```bash
go test -v ./internal/repository
# PASS (all tests)
```

**End-to-end CLI test:** ✅ Works
```bash
# Create feature
./bin/shark feature create --epic=E07 "E2E Test Custom Path"
# Result: E07-F21

# Verify initial state
./bin/shark feature get E07-F21 --json | jq '.custom_folder_path'
# Result: null ✅

# Update with custom path
./bin/shark feature update E07-F21 --path="docs/e2e-test/custom"
# Result: Feature E07-F21 updated successfully

# Verify custom path is displayed
./bin/shark feature get E07-F21 --json | jq '.custom_folder_path'
# Result: "docs/e2e-test/custom" ✅
```

## Files Modified

1. `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/feature.go`
   - Added `custom_folder_path` to JSON output map (line 575)
   - Added `custom_folder_path` to formatted text info table (lines 664-666)

2. `/home/jwwelbor/projects/shark-task-manager/internal/repository/feature_repository_test.go`
   - Added `TestFeatureRepository_UpdateCustomPath` test (lines 330-400)

## Impact

- **Severity:** Medium (feature worked but appeared broken)
- **User Experience:** Fixed - users can now see their custom paths
- **Breaking Changes:** None
- **Backward Compatibility:** Full ✅

## Lessons Learned

1. **TDD revealed the real issue quickly:** Writing the test first showed the repository was fine, focusing investigation on display layer
2. **Silent failures are confusing:** The update worked but wasn't visible, leading to user confusion
3. **Always test the full user journey:** Database updates mean nothing if users can't see them

## Verification Checklist

- [x] Repository test passes
- [x] All repository tests pass
- [x] CLI JSON output shows custom_folder_path
- [x] CLI text output shows custom_folder_path
- [x] End-to-end workflow verified
- [x] No breaking changes
- [x] Task status updated to ready_for_code_review
