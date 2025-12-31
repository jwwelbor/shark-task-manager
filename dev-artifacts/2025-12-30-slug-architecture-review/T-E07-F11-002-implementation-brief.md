# T-E07-F11-002 Implementation Brief: Backfill Slugs from Existing File Paths

**Date**: 2025-12-30
**Task**: T-E07-F11-002
**Priority**: 10 (P0 - CRITICAL)
**Phase**: 1 - Database Schema
**Agent**: Backend Developer

---

## Context

Task T-E07-F11-001 has been completed and approved. The database schema now includes slug columns for epics, features, and tasks with appropriate indexes. Now we need to backfill slugs from existing file_path values.

### Database State After T-E07-F11-001

**Schema Changes Completed:**
- ✅ Slug column added to `epics` table (TEXT, nullable)
- ✅ Slug column added to `features` table (TEXT, nullable)
- ✅ Slug column added to `tasks` table (TEXT, nullable)
- ✅ Indexes created: `idx_epics_slug`, `idx_features_slug`, `idx_tasks_slug`
- ✅ Migration is idempotent and tested

**Database Location**: `/home/jwwelbor/projects/shark-task-manager/shark-tasks.db`

---

## Data Landscape Analysis

### Current Entity Counts

| Entity Type | Total Count | With file_path | Without file_path |
|-------------|-------------|----------------|-------------------|
| Epics       | 8           | 0              | 8                 |
| Features    | 39          | 11             | 28                |
| Tasks       | 278         | 278            | 0                 |

### Key Insights

1. **Epics**: All 8 epics have NULL file_path - cannot backfill slugs from file paths
2. **Features**: Only 11 of 39 features (28%) have file paths - can only backfill these 11
3. **Tasks**: All 278 tasks (100%) have file paths - can backfill all

### Sample File Path Patterns

**Features** (from actual data):
```
E10-F01 → docs/plan/E10-advanced-task-intelligence-context-management/E10-F01-task-activity-notes-system/feature.md
E10-F02 → docs/plan/E10-advanced-task-intelligence-context-management/E10-F02-task-completion-intelligence/feature.md
```
Expected slug extraction: `task-activity-notes-system`, `task-completion-intelligence`

**Tasks** (from actual data):
```
T-E04-F01-001 → /home/jwwelbor/projects/shark-task-manager/docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/tasks/T-E04-F01-001.md
T-E04-F01-002 → /home/jwwelbor/projects/shark-task-manager/docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema/tasks/T-E04-F01-002.md
```

**IMPORTANT**: Task file paths are absolute paths, not relative!

---

## Implementation Requirements

### Slug Extraction Patterns

#### Epic Slug Extraction
**Pattern**: Extract text between `E##-` and `/epic.md`
**Example**: `docs/plan/E05-task-mgmt-cli-capabilities/epic.md` → `task-mgmt-cli-capabilities`

**CURRENT STATUS**: All epics have NULL file_path, so no backfill possible.

#### Feature Slug Extraction
**Pattern**: Extract text after `E##-F##-` and before `/feature.md` or `/prd.md`
**Example**: `docs/plan/E10-.../E10-F01-task-activity-notes-system/feature.md` → `task-activity-notes-system`

**CURRENT STATUS**: 11 features with file_path, 28 without.

#### Task Slug Extraction
**Pattern**: Extract filename without .md extension
**Example**: `/path/to/tasks/T-E04-F01-001.md` → `T-E04-F01-001`

**IMPORTANT**: Task slugs should be the task KEY (e.g., `T-E04-F01-001`), NOT extracted from filename text.

**CURRENT STATUS**: All 278 tasks have file_path, can backfill all.

---

## Implementation Strategy

### Recommended Approach: Go Implementation

Use Go for better readability, error handling, and testing:

```go
// internal/repository/slug_backfill.go

func BackfillSlugs(ctx context.Context, db *sql.DB) (*BackfillReport, error) {
    // 1. Backfill features with file_path
    // 2. Backfill tasks with file_path
    // 3. Skip epics (no file_path data)
    // 4. Generate report
}
```

### Slug Extraction Logic

**Features**:
```go
// Extract from: "docs/plan/E10-.../E10-F01-task-activity-notes-system/feature.md"
// Result: "task-activity-notes-system"
func extractFeatureSlugFromPath(filePath string) (string, error) {
    // Find pattern E##-F##-{slug}/
    // Extract {slug} portion
}
```

**Tasks**:
```go
// For task T-E04-F01-001, slug should be the task key itself
func extractTaskSlugFromKey(taskKey string) string {
    return taskKey  // Task slug = task key
}
```

### SQL Alternative (Less Recommended)

If using SQL directly, handle edge cases carefully:
- NULL file_path values
- Different path formats (absolute vs relative)
- Missing file patterns

---

## Acceptance Criteria (from Task Spec)

### AC1: Epic Slug Extraction
- [x] Epics with file_path have slugs extracted correctly
- **ACTUAL**: No epics have file_path, so skip epic backfill

### AC2: Feature Slug Extraction
- [ ] Features with file_path have slugs extracted correctly
- [ ] Slug matches pattern: text after F##- and before /feature.md or /prd.md
- [ ] Example: E10-F01-task-activity-notes → "task-activity-notes"

### AC3: Task Slug Extraction
- [ ] Tasks with file_path have slugs extracted correctly
- [ ] Task slug = task key (e.g., T-E04-F01-001)
- [ ] All 278 tasks get slugs

### AC4: NULL Handling
- [ ] Entities with NULL file_path keep slug as NULL
- [ ] Entities with empty file_path keep slug as NULL
- [ ] No errors occur for missing file_path

### AC5: Verification Report
- [ ] Count of epics with slugs extracted (expect 0)
- [ ] Count of features with slugs extracted (expect ~11)
- [ ] Count of tasks with slugs extracted (expect 278)
- [ ] Count of entities with NULL slugs (expect 8 epics + 28 features = 36)
- [ ] Summary shows coverage percentage

---

## Testing Requirements

### Integration Tests

Create test in `internal/repository/slug_backfill_test.go`:

1. **Setup**: Create test database with sample entities
2. **Execute**: Run backfill function
3. **Verify**: Check slug values extracted correctly
4. **Edge Cases**: Test NULL file_path handling
5. **Idempotency**: Run twice, ensure same result

### Manual Verification

After implementation, verify in production database:

```bash
# Check feature slug extraction
sqlite3 shark-tasks.db "SELECT key, slug, file_path FROM features WHERE file_path IS NOT NULL LIMIT 5;"

# Check task slug extraction
sqlite3 shark-tasks.db "SELECT key, slug FROM tasks LIMIT 10;"

# Check NULL handling
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM epics WHERE slug IS NULL;"
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM features WHERE slug IS NULL;"
```

---

## Expected Outcomes

### After Successful Implementation

**Epics**:
- 8 epics with slug = NULL (no file_path to extract from)

**Features**:
- ~11 features with slug extracted from file_path
- ~28 features with slug = NULL (no file_path)

**Tasks**:
- 278 tasks with slug = task key

### Verification Commands

```bash
# Count backfilled slugs
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM features WHERE slug IS NOT NULL;"
# Expected: ~11

sqlite3 shark-tasks.db "SELECT COUNT(*) FROM tasks WHERE slug IS NOT NULL;"
# Expected: 278

# Verify patterns
sqlite3 shark-tasks.db "SELECT key, slug FROM features WHERE slug IS NOT NULL LIMIT 3;"
sqlite3 shark-tasks.db "SELECT key, slug FROM tasks LIMIT 5;"
```

---

## Files to Create/Modify

### New Files
- `internal/repository/slug_backfill.go` - Backfill logic
- `internal/repository/slug_backfill_test.go` - Tests
- `internal/cli/commands/migrate_slugs.go` - CLI command (if needed for T-E07-F11-003)

### Existing Files to Reference
- `internal/db/db.go` - Database connection and schema
- `internal/models/epic.go`, `feature.go`, `task.go` - Model definitions

---

## Related Tasks

**Depends On**:
- ✅ T-E07-F11-001 - Add slug columns to database schema (COMPLETED)

**Blocks**:
- T-E07-F11-003 - Create migration CLI command for slug columns

**Phase**: Phase 1 - Database Schema (P0 - CRITICAL)

---

## Implementation Notes

### Important Considerations

1. **Task Slugs = Task Keys**: Don't extract from filename, use the task key itself
2. **Absolute Paths**: Task file_path values are absolute, handle accordingly
3. **Feature Patterns**: Features use both `feature.md` and `prd.md` filenames
4. **NULL Safety**: Many entities have NULL file_path, handle gracefully
5. **Idempotency**: Should be safe to run multiple times

### Performance

- Batch updates for efficiency (e.g., prepare statement, execute multiple times)
- Transaction for atomicity (all-or-nothing update)
- Consider using a single UPDATE per entity type

---

## Developer Checklist

- [ ] Read this implementation brief thoroughly
- [ ] Understand the data landscape (0 epics, 11 features, 278 tasks)
- [ ] Create slug extraction functions with proper error handling
- [ ] Write comprehensive tests (integration tests required)
- [ ] Test on development database first
- [ ] Generate verification report
- [ ] Verify manually with sample queries
- [ ] Update task status in shark (`shark task start`, `shark task complete`)
- [ ] Document any deviations or issues encountered

---

## Success Criteria

**Definition of Done**:
- ✅ Backfill function implemented and tested
- ✅ ~11 features have slugs extracted from file_path
- ✅ All 278 tasks have slugs (= task keys)
- ✅ Entities without file_path have NULL slugs
- ✅ Integration tests pass
- ✅ Verification report shows expected counts
- ✅ No data loss or corruption
- ✅ Task T-E07-F11-002 marked complete in shark

---

**End of Implementation Brief**
