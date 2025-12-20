# Delete Command Bug Fix - Status Tracker

**Task:** Verify fix for `shark related-docs delete` command bug
**QA Agent:** Claude (Haiku 4.5)
**Current Date:** 2025-12-20

## Status Timeline

### Phase 1: Issue Analysis - COMPLETED
- [x] Identified bug location: `internal/cli/commands/related_docs.go` lines 218-340
- [x] Root cause: `runRelatedDocsDelete` is a stub that doesn't call unlink methods
- [x] Verified repository methods exist: `UnlinkFromEpic`, `UnlinkFromFeature`, `UnlinkFromTask`
- [x] Created issue analysis document
- [x] Created comprehensive test plan with 8 test cases + 3 regression tests

**Artifacts Created:**
- `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-20-delete-command-verification/issue-analysis.md`
- `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-20-delete-command-verification/test-plan.md`

### Phase 2: Waiting for Developer Fix - IN PROGRESS
**Status:** Awaiting developer implementation

**Required Fix:**
The `runRelatedDocsDelete` function must:
1. ✓ Get parent entity (epic/feature/task) - ALREADY DONE
2. Get document by title - NEEDS IMPLEMENTATION
3. Call appropriate unlink method (UnlinkFromEpic/Feature/Task)
4. Return success/failure

**Example Fix for Epic:**
```go
// Handle epic parent
if epic != "" {
    e, err := epicRepo.GetByKey(ctx, epic)
    if err != nil {
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "status": "unlinked",
                "parent": "epic",
            })
        }
        return nil
    }

    // Get document by title
    doc, err := docRepo.getByTitle(ctx, title)
    if err != nil {
        // Idempotent: succeed even if document doesn't exist
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "status": "unlinked",
                "title":  title,
                "parent": "epic",
            })
        }
        return nil
    }

    // Unlink the document
    if err := docRepo.UnlinkFromEpic(ctx, e.ID, doc.ID); err != nil {
        return fmt.Errorf("failed to unlink document from epic: %w", err)
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "status":      "unlinked",
            "document_id": doc.ID,
            "title":       doc.Title,
            "parent":      "epic",
        })
    }

    fmt.Printf("Document unlinked from epic %s\n", epic)
    return nil
}
```

**Note:** May need helper method to get document by title, or existing method can be used.

### Phase 3: Testing & Verification - PENDING

Once developer signals fix is ready:

1. Build fresh binary
2. Execute test plan (8 core + 3 regression tests)
3. Verify database state
4. Document results

---

## Key Testing Priorities

**CRITICAL TEST:** TC-02 - Delete Document from Epic
- Add document to epic
- Verify it appears in list
- Delete document
- **VERIFY IT DOES NOT APPEAR IN LIST**

This is the core bug being fixed. All other tests validate no regressions.

---

## Developer Handoff Notes

**Files Modified:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/related_docs.go`

**Function to Fix:** `runRelatedDocsDelete` (lines 218-340)

**Available Methods in DocumentRepository:**
- `UnlinkFromEpic(ctx context.Context, epicID, documentID int64) error`
- `UnlinkFromFeature(ctx context.Context, featureID, documentID int64) error`
- `UnlinkFromTask(ctx context.Context, taskID, documentID int64) error`
- May need: Method to get document by title

**Test Validation:** Unit tests should be created in `related_docs_test.go` for the delete function.

---

## Next Steps

1. **Developer:** Implement fix in `runRelatedDocsDelete`
2. **Developer:** Run unit tests locally
3. **Developer:** Signal fix is ready
4. **QA:** Build fresh binary
5. **QA:** Execute comprehensive test plan
6. **QA:** Verify database state
7. **QA:** Document final results

---

## Contact & Questions

QA Agent: Claude (Haiku 4.5)
Analysis Location: `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-20-delete-command-verification/`

When ready: Execute test plan and provide results.
