---
task_key: T-E07-F08-001
epic_key: E07
feature_key: E07-F08
title: Add file_path columns to epics and features tables
status: created
priority: 1
agent_type: backend
depends_on: []
---

# Task: Add file_path columns to epics and features tables

## Objective

Extend the database schema to support custom file paths for epics and features by adding nullable `file_path` columns to both tables with appropriate indexes for collision detection.

## Context

**Why this task exists**: E07-F05 introduced custom filenames for tasks. This feature extends that capability to epics and features, requiring database schema changes to store custom file paths.

**Design reference**:
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/03-data-design.md` (lines 77-129)
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (lines 249-270)

## What to Build

Update the database schema in `internal/db/db.go` to add:

1. **New Columns**:
   - Add `file_path TEXT` column to `epics` table (nullable)
   - Add `file_path TEXT` column to `features` table (nullable)

2. **Indexes**:
   - Create `idx_epics_file_path` on `epics(file_path)` for fast collision detection
   - Create `idx_features_file_path` on `features(file_path)` for fast collision detection

3. **Model Updates**:
   - Update `internal/models/epic.go`: Add `FilePath *string` field with `db:"file_path"` tag
   - Update `internal/models/feature.go`: Add `FilePath *string` field with `db:"file_path"` tag

## Success Criteria

- [ ] `epics` table has `file_path TEXT` column (nullable)
- [ ] `features` table has `file_path TEXT` column (nullable)
- [ ] Both tables have btree indexes on `file_path` columns
- [ ] `Epic` model struct includes `FilePath *string` field
- [ ] `Feature` model struct includes `FilePath *string` field
- [ ] Existing databases automatically upgrade when schema is applied (via `CREATE IF NOT EXISTS`)
- [ ] NULL values in `file_path` represent default file locations (backward compatibility)

## Validation Gates

1. **Schema Validation**:
   - Run `make build` successfully after changes
   - Create a test database and verify columns exist via SQLite CLI
   - Verify indexes are created: `.schema epics` and `.schema features`

2. **Model Validation**:
   - Epic and Feature structs compile without errors
   - Pointer types (*string) allow NULL representation

3. **Migration Safety**:
   - Existing epics/features remain queryable after schema update
   - No data loss or corruption occurs during column addition

## Dependencies

**Prerequisite Tasks**: None (foundation task)

**Blocks**: T-E07-F08-002, T-E07-F08-003, T-E07-F08-004, T-E07-F08-005 (all subsequent tasks need database schema in place)

## Implementation Notes

### Database Schema Pattern

The schema changes should follow SQLite's `ALTER TABLE` pattern:

```sql
-- Add columns
ALTER TABLE epics ADD COLUMN file_path TEXT;
ALTER TABLE features ADD COLUMN file_path TEXT;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_epics_file_path ON epics(file_path);
CREATE INDEX IF NOT EXISTS idx_features_file_path ON features(file_path);
```

However, since the project uses embedded schema strings with `CREATE IF NOT EXISTS`, add these directly to the table definitions in `internal/db/db.go`.

### Model Field Pattern

Use pointer types for nullable database columns:

```go
type Epic struct {
    // ... existing fields ...
    FilePath *string `db:"file_path"`
}
```

This allows `nil` to represent NULL in the database, preserving backward compatibility for epics/features using default locations.

### Testing Strategy

After implementation, test with:

```bash
# Build the binary
make build

# Initialize a test database
./bin/shark init --non-interactive --db=test-schema.db

# Verify schema
sqlite3 test-schema.db ".schema epics"
sqlite3 test-schema.db ".schema features"

# Check for indexes
sqlite3 test-schema.db "SELECT name FROM sqlite_master WHERE type='index' AND tbl_name IN ('epics', 'features');"
```

## References

- **Data Design**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/03-data-design.md`
- **Current Schema**: `internal/db/db.go` (lines 67-191)
- **PRD Section**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (Section 4.1)
