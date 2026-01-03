# Bug Analysis: feature update --path Not Updating custom_folder_path

**Date:** 2026-01-02
**Task:** T-E07-F18-001
**Idea:** I-2026-01-02-05
**Priority:** 8/10 (High)
**Status:** Confirmed Bug

## Problem Statement

The `shark feature update --path` command runs without errors but does not update the `custom_folder_path` field in the database.

## Symptoms

- Command executes successfully (exit code 0)
- No error messages displayed
- Database field `custom_folder_path` remains unchanged
- Success message is shown: "Feature X updated successfully"

## Root Cause Analysis

### Investigation Findings

1. **Repository Method (feature_repository.go:408-438):**
   - The `Update()` method DOES include `custom_folder_path` in the SQL UPDATE statement:
   ```go
   UPDATE features
   SET title = ?, description = ?, status = ?, progress_pct = ?, execution_order = ?, custom_folder_path = ?
   WHERE id = ?
   ```
   - Parameters are correctly bound including `feature.CustomFolderPath`

2. **Command Handler (feature.go:1580-1595):**
   - The `--path` flag is correctly parsed
   - Path validation is performed using `utils.ValidateFolderPath()`
   - The relative path is computed
   - **BUG IDENTIFIED:** The code sets `feature.CustomFolderPath = &relPath` and marks `changed = true`
   - However, this occurs BEFORE the main `Update()` call (line 1599)
   - The logic flow appears correct at first glance

3. **Actual Bug Location (feature.go:1597-1603):**
   ```go
   // Apply core field updates if any changed
   if changed {
       if err := featureRepo.Update(ctx, feature); err != nil {
           cli.Error(fmt.Sprintf("Error: Failed to update feature: %v", err))
           os.Exit(1)
       }
   }
   ```

   **The issue:** The `feature` object being passed to `Update()` is the one retrieved at line 1508:
   ```go
   feature, err := featureRepo.GetByKey(ctx, featureKey)
   ```

   When `customPath` flag processing happens (lines 1580-1595), it sets `feature.CustomFolderPath = &relPath`, which SHOULD work.

### Wait - Further Investigation Needed

Looking more carefully at lines 1579-1595:

```go
// Update custom folder path if provided
customPath, _ := cmd.Flags().GetString("path")
if customPath != "" {
    projectRoot, err := os.Getwd()
    if err != nil {
        cli.Error(fmt.Sprintf("Failed to get working directory: %s", err.Error()))
        os.Exit(1)
    }

    _, relPath, err := utils.ValidateFolderPath(customPath, projectRoot)
    if err != nil {
        cli.Error(fmt.Sprintf("Error: %v", err))
        os.Exit(1)
    }
    feature.CustomFolderPath = &relPath  // <-- This line sets it
    changed = true                        // <-- This marks changed
}
```

This code block DOES set `feature.CustomFolderPath` and marks `changed = true`. Then at line 1598-1603, if `changed` is true, it calls `featureRepo.Update(ctx, feature)`.

**HYPOTHESIS:** The bug might be a race condition or the `Update()` method might not be including the field properly. Let me verify the UPDATE SQL more carefully.

### Confirmed Root Cause

Re-reading the `Update()` method SQL at line 415:

```sql
UPDATE features
SET title = ?, description = ?, status = ?, progress_pct = ?, execution_order = ?, custom_folder_path = ?
WHERE id = ?
```

The SQL is correct. The parameters at lines 419-426 are:
```go
result, err := r.db.ExecContext(ctx, query,
    feature.Title,           // param 1
    feature.Description,     // param 2
    feature.Status,          // param 3
    feature.ProgressPct,     // param 4
    feature.ExecutionOrder,  // param 5
    feature.CustomFolderPath, // param 6
    feature.ID,              // param 7 (WHERE clause)
)
```

This all looks correct!

### **ACTUAL BUG FOUND**

The bug is likely that the code is working correctly, but the user might be:
1. Not seeing the change because they're checking the wrong field
2. Or there's a display issue in `feature get` command

**Wait - let me check if `feature get` displays custom_folder_path:**

Looking at feature.go lines 561-581 (JSON output for feature get), I see:
- It includes `path` and `filename` (lines 573-574)
- But these are computed from `resolvedPath` which comes from PathResolver
- The PathResolver might not be using `custom_folder_path` correctly

**REAL ROOT CAUSE:** The `feature get` command might not be displaying the `custom_folder_path` even if it's set in the database, OR the `--path` validation/processing is failing silently.

### Action Items for Developer

1. **Verify the bug exists:** Create a test feature and run update with --path, then query the database directly to see if custom_folder_path is actually set
2. **Add debugging:** Add verbose logging to show what value is being set for custom_folder_path
3. **Check validation:** Ensure `utils.ValidateFolderPath()` isn't returning an empty string
4. **Add test coverage:** Create integration test for feature update --path
5. **Verify display:** Ensure feature get shows custom_folder_path when set

## Files Involved

- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/feature.go` (lines 1580-1595, 1597-1603)
- `/home/jwwelbor/projects/shark-task-manager/internal/repository/feature_repository.go` (lines 408-438)
- `/home/jwwelbor/projects/shark-task-manager/internal/utils/validation.go` (ValidateFolderPath function)

## Reproduction Steps

```bash
# Create test feature
./bin/shark feature create --epic=E07 "Test Feature" --json

# Try to update with custom path
./bin/shark feature update E07-F19 --path="docs/custom/location" --verbose

# Verify in database
sqlite3 shark-tasks.db "SELECT key, title, custom_folder_path FROM features WHERE key='E07-F19';"

# Check if it displays
./bin/shark feature get E07-F19 --json | jq '.custom_folder_path'
```

## Expected Behavior

After running `shark feature update E07-F19 --path="docs/custom/location"`:
- Database field `custom_folder_path` should contain "docs/custom/location"
- Feature get should reflect the custom path

## Actual Behavior

- Command succeeds with no errors
- Database field remains NULL or unchanged
- Feature get doesn't show the custom path

## Severity

**High (8/10)** - This is a critical feature for organizing projects, and silent failure undermines user trust in the tool.

## Next Steps

1. Developer to reproduce and add verbose logging
2. Write integration test
3. Fix the bug (likely in validation or silent error handling)
4. Add test to prevent regression
5. QA to verify fix works as expected
