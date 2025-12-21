# Delete Command Bug Analysis

**Date:** 2025-12-20
**Status:** Waiting for Developer Fix

## Issue Summary

The `shark related-docs delete` command is not actually deleting document links from the database. The implementation is a stub that returns without calling any repository deletion methods.

## Root Cause

In `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/related_docs.go`, the `runRelatedDocsDelete` function (lines 218-340):

1. Validates the parent entity (epic, feature, task) exists
2. Returns immediately WITHOUT calling any repository unlink methods
3. Returns success JSON without actually removing the database link

### Current Problematic Code (lines 248-276 for epic example):

```go
// Handle epic parent
if epic != "" {
    e, err := epicRepo.GetByKey(ctx, epic)
    if err != nil {
        // ... handle error ...
        return nil
    }

    _ = e // Use variable to avoid unused error

    // MISSING: No call to docRepo.UnlinkFromEpic()

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "status": "unlinked",
            "title":  title,
            "parent": "epic",
        })
    }

    return nil
}
```

## Available Repository Methods

The `DocumentRepository` has all required unlink methods:

1. `UnlinkFromEpic(ctx context.Context, epicID, documentID int64) error` (line 165-175)
2. `UnlinkFromFeature(ctx context.Context, featureID, documentID int64) error` (line 177-187)
3. `UnlinkFromTask(ctx context.Context, taskID, documentID int64) error` (line 189-199)

## Required Fix

For each parent type (epic, feature, task), the delete command must:

1. Get the parent entity (already done)
2. Get the document by title (needs implementation - search by title in documents table)
3. Call the appropriate unlink method with parent ID and document ID
4. Handle idempotent delete (succeed even if link doesn't exist)

## Test Plan Ready

Prepared comprehensive test cases:
- Add document to epic, list to verify, delete, list to verify removal
- Regression tests for feature and task
- Idempotent delete test (delete non-existent)
- JSON output validation
- Full unit test suite validation

Awaiting developer fix before executing tests.
