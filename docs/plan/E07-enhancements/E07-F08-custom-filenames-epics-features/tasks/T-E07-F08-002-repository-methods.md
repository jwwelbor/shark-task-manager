---
task_key: T-E07-F08-002
epic_key: E07
feature_key: E07-F08
title: Implement GetByFilePath and UpdateFilePath repository methods
status: created
priority: 2
agent_type: backend
depends_on: ["T-E07-F08-001"]
---

# Task: Implement GetByFilePath and UpdateFilePath repository methods

## Objective

Extend `EpicRepository` and `FeatureRepository` with methods to query entities by file path (for collision detection) and update file paths (for force reassignment).

## Context

**Why this task exists**: Custom filename support requires collision detection (checking if a file path is already claimed) and force reassignment (clearing a file path from one entity to assign it to another). Repository methods provide the database operations for these workflows.

**Design reference**:
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (lines 316-435)
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (lines 283-315)

## What to Build

Add the following methods to epic and feature repositories:

### Epic Repository (`internal/repository/epic_repository.go`)

1. **GetByFilePath**:
   - Purpose: Retrieve an epic by its file path for collision detection
   - Signature: `GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error)`
   - Query: `SELECT * FROM epics WHERE file_path = ?`
   - Returns `(nil, nil)` if no epic found (not an error)
   - Returns `(*Epic, nil)` if epic found
   - Returns `(nil, error)` on database failures

2. **UpdateFilePath**:
   - Purpose: Update or clear the file path for an epic
   - Signature: `UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error`
   - Query: `UPDATE epics SET file_path = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?`
   - Pass `nil` for `newFilePath` to set `file_path` to NULL
   - Returns error if epic not found (no rows updated)

### Feature Repository (`internal/repository/feature_repository.go`)

3. **GetByFilePath**:
   - Purpose: Retrieve a feature by its file path for collision detection
   - Signature: `GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error)`
   - Query: `SELECT * FROM features WHERE file_path = ?`
   - Returns `(nil, nil)` if no feature found
   - Returns `(*Feature, nil)` if feature found
   - Returns `(nil, error)` on database failures

4. **UpdateFilePath**:
   - Purpose: Update or clear the file path for a feature
   - Signature: `UpdateFilePath(ctx context.Context, featureKey string, newFilePath *string) error`
   - Query: `UPDATE features SET file_path = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?`
   - Pass `nil` for `newFilePath` to set `file_path` to NULL
   - Returns error if feature not found

## Success Criteria

- [ ] `EpicRepository.GetByFilePath` correctly queries epics by file path
- [ ] `EpicRepository.UpdateFilePath` correctly updates or clears epic file paths
- [ ] `FeatureRepository.GetByFilePath` correctly queries features by file path
- [ ] `FeatureRepository.UpdateFilePath` correctly updates or clears feature file paths
- [ ] All methods handle NULL values correctly (pointer semantics)
- [ ] Methods follow existing repository patterns (error wrapping, context usage)
- [ ] Unit tests pass for all four methods

## Validation Gates

1. **Unit Tests**:
   - Write tests in `internal/repository/epic_repository_test.go`:
     - `TestEpicRepository_GetByFilePath`
     - `TestEpicRepository_GetByFilePath_NotFound`
     - `TestEpicRepository_UpdateFilePath`
     - `TestEpicRepository_UpdateFilePath_Clear` (set to NULL)

   - Write tests in `internal/repository/feature_repository_test.go`:
     - `TestFeatureRepository_GetByFilePath`
     - `TestFeatureRepository_GetByFilePath_NotFound`
     - `TestFeatureRepository_UpdateFilePath`
     - `TestFeatureRepository_UpdateFilePath_Clear` (set to NULL)

2. **Error Handling**:
   - Methods return wrapped errors with context: `fmt.Errorf("get epic by file path: %w", err)`
   - Database errors are propagated correctly
   - "Not found" is not an error for `GetByFilePath` (returns nil)

3. **Query Performance**:
   - Verify queries use the `idx_epics_file_path` and `idx_features_file_path` indexes
   - Use SQLite `EXPLAIN QUERY PLAN` to confirm index usage

## Dependencies

**Prerequisite Tasks**:
- T-E07-F08-001 (database schema must include `file_path` columns and indexes)

**Blocks**:
- T-E07-F08-003 (CLI commands need these repository methods for collision detection)
- T-E07-F08-004 (CLI commands need these methods for force reassignment)

## Implementation Notes

### Query Pattern

Follow the existing repository pattern for database queries:

```go
func (r *EpicRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error) {
    var epic models.Epic
    query := `SELECT id, key, title, description, status, priority, business_value, file_path,
              created_at, updated_at FROM epics WHERE file_path = ?`

    err := r.db.GetContext(ctx, &epic, query, filePath)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil // Not found is not an error
        }
        return nil, fmt.Errorf("get epic by file path: %w", err)
    }

    return &epic, nil
}
```

### NULL Handling

Use pointer types for nullable parameters:

```go
func (r *EpicRepository) UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error {
    query := `UPDATE epics SET file_path = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?`

    result, err := r.db.ExecContext(ctx, query, newFilePath, epicKey)
    if err != nil {
        return fmt.Errorf("update epic file path: %w", err)
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("epic not found: %s", epicKey)
    }

    return nil
}
```

Passing `nil` for `newFilePath` sets the database column to NULL.

### Testing Strategy

Create test database with sample data:

```go
func TestEpicRepository_GetByFilePath(t *testing.T) {
    db := testdb.Setup(t)
    repo := repository.NewEpicRepository(db)

    // Create epic with custom file path
    customPath := "docs/roadmap/2025.md"
    epic := &models.Epic{
        Key: "E99",
        Title: "Test Epic",
        Status: "draft",
        Priority: "medium",
        FilePath: &customPath,
    }
    repo.Create(context.Background(), epic)

    // Test GetByFilePath
    found, err := repo.GetByFilePath(context.Background(), customPath)
    assert.NoError(t, err)
    assert.NotNil(t, found)
    assert.Equal(t, "E99", found.Key)
    assert.Equal(t, customPath, *found.FilePath)
}
```

## References

- **Backend Design**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (Section: Repository Methods)
- **Existing Repository Pattern**: `internal/repository/task_repository.go` (for consistency)
- **PRD Section**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (Section 4.3)
