# Slug Extraction Bug Fix Summary

## Problem Identified

The slug backfill was incorrectly extracting feature keys (F01, F05, etc.) as epic slugs when epic folders didn't contain slugs.

### Bug Evidence

**Database showed incorrect epic slugs:**
```
E05 -> "F01" (should be NULL)
E07 -> "F05" (should be NULL)
E08 -> "F01" (should be NULL)
E10 -> "F01" (should be NULL)
```

### Root Cause

**Path Example:** `docs/plan/E08/E08-F01/tasks/...`
- Epic folder: `E08` (no slug - just key)
- Feature folder: `E08-F01` (contains feature key)

The extraction logic would:
1. Find first `E##-` pattern → `E08-`
2. Extract slug after `-` → Next `/` found
3. Extract `F01` as the epic slug ❌

## Solution Implemented

Added validation to `extractEpicSlugFromPath()` to reject feature key patterns.

### Validation Function

```go
// isFeatureKeyPattern checks if a string matches feature key pattern
func isFeatureKeyPattern(s string) bool {
    // Rejects: "F01", "F05", "F123" (exact matches)
    // Rejects: "F01-migrations", "F05-slug-name" (starts with F##-)
    // Allows: "task-mgmt-cli-core", "enhancements" (valid slugs)
}
```

### Validation Rules

1. Epic slug must NOT match `F\d\d` or `F\d\d\d` pattern
2. Epic slug must NOT start with `F\d-` or `F\d\d-` or `F\d\d\d-`
3. Only valid slugs like "task-mgmt-cli-core" are accepted
4. If validation fails, return empty string (NULL slug)

## Test Cases Added

```go
// Epic folder without slug - should NOT extract F01
{
    path: "docs/plan/E08/E08-F01/tasks/T-E08-F01-001.md",
    expected: "",  // Not "F01"
}

// Epic folder without slug - should NOT extract F01-migrations
{
    path: "docs/plan/E05/E05-F01-migrations/prd.md",
    expected: "",  // Not "F01-migrations"
}

// Epic with proper slug - should extract correctly
{
    path: "docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/tasks/...",
    expected: "task-mgmt-cli-core"  // Correct
}
```

## Results

### Before Fix
```
E05: "F01" ❌
E07: "F05" ❌
E08: "F01" ❌
E10: "F01" ❌
```

### After Fix
```
E05: NULL ✅ (correct - folder has no slug)
E07: "enhancements" ✅ (extracted from path)
E08: NULL ✅ (correct - folder has no slug)
E10: "advanced-task-intelligence-context-management" ✅ (extracted from path)
```

### Coverage Improvement

**Epics:**
- Before: 0/8 (0%) - with incorrect F## values
- After: 5/8 (62.5%) - with correct slugs or NULL

**Features:**
- Before: 11/39 (28%)
- After: 26/39 (66.7%) - from parent path extraction

**Tasks:**
- Stable: 1/278 (0.4%) - only tasks with slugs in filename

### Correct NULL Cases

Three epics correctly have NULL slugs because their folder names don't contain slugs:
- E05 (folder: `E05/`)
- E08 (folder: `E08/`)
- E09 (folder: `E09/`)

## Files Modified

1. `/internal/db/migrate_slug_backfill.go`
   - Added `isFeatureKeyPattern()` validation function
   - Updated `extractEpicSlugFromPath()` to validate extracted slugs

2. `/internal/db/migrate_slug_backfill_test.go`
   - Added 3 test cases for epic folders without slugs
   - All tests passing

## Testing

All tests pass:
```
✅ TestExtractEpicSlugFromPath - All 12 test cases
✅ TestExtractFeatureSlugFromPath - All 9 test cases
✅ TestExtractTaskSlugFromPath - All 7 test cases
✅ TestBackfillSlugsFromFilePaths - Integration test
```

## Status

🟢 **COMPLETE** - Bug fixed, tests passing, backfill verified on production database.
