# Developer Fix Guide - Delete Command Bug

**File to Modify:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/related_docs.go`

**Function to Fix:** `runRelatedDocsDelete` (lines 218-340)

**Current Status:** Stub implementation that validates parent exists but doesn't delete

---

## Problem Analysis

The function currently:
1. ✓ Validates parent entity (epic/feature/task) exists
2. ✓ Returns success message
3. ✗ MISSING: Does NOT call unlink method from DocumentRepository
4. ✗ MISSING: Does NOT look up the document by title first

Result: Database link is never removed, causing data inconsistency.

---

## Available Repository Methods

The DocumentRepository has all needed unlink methods:

```go
// From internal/repository/document_repository.go

// Unlink methods (lines 165-199)
func (r *DocumentRepository) UnlinkFromEpic(ctx context.Context, epicID, documentID int64) error
func (r *DocumentRepository) UnlinkFromFeature(ctx context.Context, featureID, documentID int64) error
func (r *DocumentRepository) UnlinkFromTask(ctx context.Context, taskID, documentID int64) error

// Document lookup (can search by title)
// May need to add: getByTitle() method or use existing getByTitleAndPath()
```

---

## Implementation Guide

### Step 1: Add Document Lookup Helper (If Needed)

Check if DocumentRepository already has method to get document by title. If not, add:

```go
// In document_repository.go
func (r *DocumentRepository) getByTitle(ctx context.Context, title string) (*models.Document, error) {
    query := `
        SELECT id, title, file_path, created_at
        FROM documents
        WHERE title = ?
    `

    doc := &models.Document{}
    err := r.db.QueryRowContext(ctx, query, title).Scan(
        &doc.ID,
        &doc.Title,
        &doc.FilePath,
        &doc.CreatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("document not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get document: %w", err)
    }

    return doc, nil
}
```

Or use existing `getByTitleAndPath()` if path is known.

### Step 2: Fix Epic Handler

Replace the epic section (lines 248-276) with:

```go
// Handle epic parent
if epic != "" {
    // Get the epic
    e, err := epicRepo.GetByKey(ctx, epic)
    if err != nil {
        // Epic doesn't exist, but delete is idempotent - succeed anyway
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "status": "unlinked",
                "parent": "epic",
            })
        }
        return nil
    }

    // Create document repository (needed to unlink)
    docRepo := repository.NewDocumentRepository(dbWrapper)

    // Find the document by title
    doc, err := docRepo.getByTitle(ctx, title)  // May need to make this public or use different approach
    if err != nil {
        // Document doesn't exist, but delete is idempotent - succeed anyway
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "status": "unlinked",
                "title":  title,
                "parent": "epic",
            })
        }
        return nil
    }

    // CRITICAL FIX: Actually unlink the document
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

### Step 3: Fix Feature Handler

Replace the feature section (lines 278-303) with:

```go
// Handle feature parent
if feature != "" {
    // Get the feature
    f, err := featureRepo.GetByKey(ctx, feature)
    if err != nil {
        // Feature doesn't exist, but delete is idempotent - succeed anyway
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "status": "unlinked",
                "parent": "feature",
            })
        }
        return nil
    }

    // Create document repository
    docRepo := repository.NewDocumentRepository(dbWrapper)

    // Find the document by title
    doc, err := docRepo.getByTitle(ctx, title)
    if err != nil {
        // Document doesn't exist, but delete is idempotent - succeed anyway
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "status": "unlinked",
                "title":  title,
                "parent": "feature",
            })
        }
        return nil
    }

    // CRITICAL FIX: Actually unlink the document
    if err := docRepo.UnlinkFromFeature(ctx, f.ID, doc.ID); err != nil {
        return fmt.Errorf("failed to unlink document from feature: %w", err)
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "status":      "unlinked",
            "document_id": doc.ID,
            "title":       doc.Title,
            "parent":      "feature",
        })
    }

    fmt.Printf("Document unlinked from feature %s\n", feature)
    return nil
}
```

### Step 4: Fix Task Handler

Replace the task section (lines 305-330) with:

```go
// Handle task parent
if task != "" {
    // Get the task
    t, err := taskRepo.GetByKey(ctx, task)
    if err != nil {
        // Task doesn't exist, but delete is idempotent - succeed anyway
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "status": "unlinked",
                "parent": "task",
            })
        }
        return nil
    }

    // Create document repository
    docRepo := repository.NewDocumentRepository(dbWrapper)

    // Find the document by title
    doc, err := docRepo.getByTitle(ctx, title)
    if err != nil {
        // Document doesn't exist, but delete is idempotent - succeed anyway
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "status": "unlinked",
                "title":  title,
                "parent": "task",
            })
        }
        return nil
    }

    // CRITICAL FIX: Actually unlink the document
    if err := docRepo.UnlinkFromTask(ctx, t.ID, doc.ID); err != nil {
        return fmt.Errorf("failed to unlink document from task: %w", err)
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "status":      "unlinked",
            "document_id": doc.ID,
            "title":       doc.Title,
            "parent":      "task",
        })
    }

    fmt.Printf("Document unlinked from task %s\n", task)
    return nil
}
```

---

## Key Points

1. **Idempotent Delete:** Delete should succeed even if:
   - Parent doesn't exist
   - Document doesn't exist
   - Link already removed

2. **Document Lookup:** Must find document by title before unlinking
   - Consider: What if multiple documents have same title?
   - Current design assumes title is unique identifier (validate)

3. **Error Handling:**
   - Handle "not found" as success (idempotent)
   - Return actual errors from unlink operation

4. **JSON Output:** Include document_id in success response for client use

5. **User Feedback:** Print confirmation message for CLI users

---

## Testing Checklist Before Signaling Ready

- [ ] No compilation errors
- [ ] `make build` succeeds
- [ ] Logic handles all 3 parent types (epic, feature, task)
- [ ] Idempotent delete works (non-existent document)
- [ ] JSON output includes document_id
- [ ] Human-readable output works
- [ ] Unit tests still pass: `make test`
- [ ] Manual test works:
  ```bash
  # Add document
  shark related-docs add "TestDoc" docs/test.md --epic=E01

  # List to verify
  shark related-docs list --epic=E01
  # Should show TestDoc

  # Delete
  shark related-docs delete "TestDoc" --epic=E01

  # List to verify deletion
  shark related-docs list --epic=E01
  # Should NOT show TestDoc (CRITICAL)
  ```

---

## Notes for Implementation

### Document Lookup Approach

**Option A:** Make `getByTitle` public in DocumentRepository
```go
// In document_repository.go
func (r *DocumentRepository) GetByTitle(ctx context.Context, title string) (*models.Document, error) {
    // implementation
}
```

**Option B:** Use existing `getByTitleAndPath` method
- Requires knowing the document path
- May need different approach

**Option C:** Add new query that finds document link directly
- Query epic_documents/feature_documents/task_documents to find document_id
- Then fetch document record

Recommend **Option A** for consistency with other repository methods.

### Avoiding Code Duplication

All three handlers (epic, feature, task) follow same pattern:
1. Get parent entity
2. Find document by title
3. Call appropriate unlink method
4. Return success

Consider refactoring into helper functions if code duplication is an issue:
```go
func (r *DocumentRepository) unlinkDocumentFromParent(ctx context.Context,
    parentID, documentID int64, parentType string) error {

    switch parentType {
    case "epic":
        return r.UnlinkFromEpic(ctx, parentID, documentID)
    case "feature":
        return r.UnlinkFromFeature(ctx, parentID, documentID)
    case "task":
        return r.UnlinkFromTask(ctx, parentID, documentID)
    default:
        return fmt.Errorf("unknown parent type: %s", parentType)
    }
}
```

---

## Validation Queries

After implementing fix, verify with these SQLite queries:

```sql
-- Before delete: document link exists
SELECT * FROM epic_documents WHERE epic_id=1 AND document_id=1;
-- Should return 1 row

-- After delete: document link removed
SELECT * FROM epic_documents WHERE epic_id=1 AND document_id=1;
-- Should return 0 rows

-- Verify document record still exists (not deleted, only unlinked)
SELECT * FROM documents WHERE id=1;
-- Should still exist
```

---

## Questions to Ask Yourself

1. **Idempotency:** What happens if I delete same document twice?
   - Should succeed both times

2. **Data Consistency:** What if title is not unique?
   - Current design appears to assume title uniqueness
   - Verify with team before release

3. **Error vs Success:** When should delete return error vs succeed silently?
   - Unlink operation fails (DB error) = return error
   - Document not found = succeed (idempotent)
   - Parent not found = succeed (idempotent)

4. **JSON Contract:** What fields should JSON response include?
   - status: "unlinked" / "failed"
   - document_id: (for success)
   - title: (document title)
   - parent: ("epic" / "feature" / "task")
   - error: (for failures)

---

## When Ready

1. Commit changes locally
2. Run `make test` to verify unit tests pass
3. Create simple manual test
4. Signal to QA: "Fix is ready for verification"
5. QA will run comprehensive test plan

---

## Support

If stuck:
- Check existing LinkToX methods for pattern reference
- Review UnlinkFromX methods to understand expected parameters
- Look at related-docs add command for similar structure
- Run `make test` frequently to catch issues early

Good luck! The fix is straightforward - just need to add the unlink method calls.
