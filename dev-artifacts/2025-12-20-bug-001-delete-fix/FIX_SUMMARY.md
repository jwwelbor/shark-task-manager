# BUG-001 Fix Summary

## Problem Statement

The `shark related-docs delete` command was returning success messages but not actually removing document links from the database. This was a false-positive success bug.

### Root Cause

In `/internal/cli/commands/related_docs.go`, the `runRelatedDocsDelete` function:
1. Validated that the parent entity (epic, feature, or task) exists
2. BUT never called any `UnlinkFromEpic()`, `UnlinkFromFeature()`, or `UnlinkFromTask()` methods
3. Simply returned a success response without performing the actual deletion

### Example of Broken Behavior

```bash
$ shark related-docs add "Design Doc" docs/design.md --epic=E01
Document linked to epic E01

$ shark related-docs list --epic=E01
Related Documents:
  - Design Doc (docs/design.md)

$ shark related-docs delete "Design Doc" --epic=E01
Document unlinked from epic E01    # FALSE - not actually unlinked!

$ shark related-docs list --epic=E01
Related Documents:
  - Design Doc (docs/design.md)    # STILL LINKED - bug confirmed!
```

## Solution Implemented

### 1. Added GetByTitle Method to DocumentRepository

**File**: `/internal/repository/document_repository.go`

Added public method to look up documents by title alone (without requiring the file path):

```go
// GetByTitle retrieves a document by title only
func (r *DocumentRepository) GetByTitle(ctx context.Context, title string) (*models.Document, error) {
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

### 2. Fixed runRelatedDocsDelete to Actually Unlink Documents

**File**: `/internal/cli/commands/related_docs.go`

Updated the delete command handler to:
1. Look up the document by title using `GetByTitle()`
2. Call the appropriate `UnlinkFrom*()` method if document exists
3. Handle non-existent documents gracefully (idempotent behavior)

**Key Changes for Epic Handling** (similar for Feature and Task):

```go
// Handle epic parent
if epic != "" {
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

	// Look up the document by title
	doc, err := docRepo.GetByTitle(ctx, title)
	if err == nil {
		// Document exists, actually perform the unlinking
		if err := docRepo.UnlinkFromEpic(ctx, e.ID, doc.ID); err != nil {
			return fmt.Errorf("failed to unlink document: %w", err)
		}
	}
	// If document doesn't exist, delete is idempotent - succeed anyway

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status": "unlinked",
			"title":  title,
			"parent": "epic",
		})
	}

	fmt.Printf("Document unlinked from epic %s\n", epic)
	return nil
}
```

### 3. Added Unit Tests

**File**: `/internal/repository/document_repository_test.go`

Added two new test functions:

1. **TestGetByTitle**: Verifies document lookup by title works correctly
   - Creates a document
   - Retrieves it using `GetByTitle()`
   - Validates all fields match

2. **TestGetByTitleNotFound**: Verifies proper error handling
   - Attempts to retrieve non-existent document
   - Confirms error is returned

## Test Results

All tests pass, including:
- New `TestGetByTitle` - PASS
- New `TestGetByTitleNotFound` - PASS
- All existing document repository tests - PASS
- Full project test suite - PASS
- Project builds successfully with no errors

## Verification

### Before Fix
```
$ shark related-docs list --epic=E01 --json
[{"id": 1, "title": "Design Doc", "file_path": "docs/design.md", ...}]

$ shark related-docs delete "Design Doc" --epic=E01
Document unlinked from epic E01

$ shark related-docs list --epic=E01 --json
[{"id": 1, "title": "Design Doc", "file_path": "docs/design.md", ...}]  # BUG: Still there!
```

### After Fix
```
$ shark related-docs list --epic=E01 --json
[{"id": 1, "title": "Design Doc", "file_path": "docs/design.md", ...}]

$ shark related-docs delete "Design Doc" --epic=E01
Document unlinked from epic E01

$ shark related-docs list --epic=E01 --json
[]  # FIXED: Document actually unlinked!

$ shark related-docs delete "Design Doc" --epic=E01  # Idempotent
Document unlinked from epic E01  # Succeeds even though link doesn't exist
```

## Idempotent Delete Behavior

The fix ensures the delete command is idempotent (safe to call multiple times):

1. If parent entity doesn't exist → Success
2. If document doesn't exist → Success
3. If both exist but not linked → Success
4. If both exist and are linked → Unlinks and succeeds

This aligns with the design stated in the command documentation.

## Files Modified

1. `/internal/repository/document_repository.go`
   - Added `GetByTitle()` method (public)

2. `/internal/cli/commands/related_docs.go`
   - Updated `runRelatedDocsDelete()` to call `GetByTitle()` and `UnlinkFrom*()` methods
   - Added proper document lookup and actual unlinking
   - Added human-readable success messages

3. `/internal/repository/document_repository_test.go`
   - Added `TestGetByTitle()` test
   - Added `TestGetByTitleNotFound()` test

## Impact

- Fixes false-positive success messages
- Delete command now actually removes links from database
- Maintains idempotent behavior for safe repeated calls
- No breaking changes to public APIs
- All existing tests continue to pass
