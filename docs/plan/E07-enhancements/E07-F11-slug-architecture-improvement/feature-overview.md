# Feature Overview: Slug Architecture Improvement

**Feature Key**: E07-F11
**Epic**: E07 - Database & State Management Enhancements
**Status**: Planning
**Priority**: High
**Created**: 2025-12-30

---

## Business Value

### Problem Statement

The current architecture violates a fundamental principle: **the database should be the single source of truth for all metadata**. Currently:

1. **Discovery reads filesystem to determine slugs** and updates the database (file system becomes source of truth)
2. **No slug column in database** - slugs are computed on-the-fly from titles, causing performance overhead and non-deterministic paths
3. **Inconsistent file formats** - tasks use YAML frontmatter while epics/features use markdown body parsing
4. **Key format confusion** - users must know full slugged keys (`E05-task-mgmt-cli-capabilities`) instead of just numeric keys (`E05`)

### Impact

**Data Integrity Issues**:
- File content becomes source of truth for metadata, violating database-first architecture
- If title changes, slug changes → file path changes → potential file orphaning
- No validation that computed path matches database file_path

**Performance Issues**:
- Unnecessary file I/O during path resolution and discovery (~1ms per path resolution)
- Slug generation happens every time a path is needed instead of once at creation
- Discovery must parse markdown files instead of structured frontmatter

**User Experience Issues**:
- Users must remember and type complex slugged keys for all operations
- Poor discoverability (is `E05` or `E05-task-mgmt-cli-capabilities` the "real" key?)
- Inconsistent patterns across epics, features, and tasks create confusion

**Technical Debt**:
- Multiple code paths for determining file locations
- Inconsistent patterns across entity types
- PathBuilder complexity with file reads and slug computation

### Expected Benefits

After implementing this feature:

1. **Database as Single Source of Truth**: All metadata (key, title, slug) stored and managed in database
2. **10x Faster Path Resolution**: 0.1ms (database read) vs 1ms (compute slug + build path)
3. **2-3x Faster Discovery**: YAML frontmatter parsing vs markdown parsing
4. **Better User Experience**: Both `E05` and `E05-slug` work in all commands
5. **Data Integrity**: Slugs validated against database, no drift between files and database
6. **Consistent Patterns**: Epics, features, tasks all use same approach

---

## Technical Requirements

### Database Schema Changes

Add `slug` column to all entity tables:

```sql
-- Epics table
ALTER TABLE epics ADD COLUMN slug TEXT;
CREATE INDEX idx_epics_slug ON epics(slug);

-- Features table
ALTER TABLE features ADD COLUMN slug TEXT;
CREATE INDEX idx_features_slug ON features(slug);

-- Tasks table (for consistency)
ALTER TABLE tasks ADD COLUMN slug TEXT;
CREATE INDEX idx_tasks_slug ON tasks(slug);
```

### File Format Standardization

**Current State**:
- Tasks: YAML frontmatter with `task_key` ✅
- Epics: Markdown body with `**Epic Key**: E05-slug` ❌
- Features: No key in file, discovered from folder name ❌

**Target State** (all entities):
```yaml
---
epic_key: E05
slug: task-mgmt-cli-capabilities
title: Task Management CLI - Extended Capabilities
status: active
priority: high
created_at: 2025-12-14
---

# Epic: Task Management CLI - Extended Capabilities

Content...
```

### Path Resolution Strategy

**Current**: PathBuilder reads files, computes slugs, builds paths
**Target**: PathResolver reads database only, uses stored slugs

```go
// Current (bad)
pb := PathBuilder{}
path := pb.ResolveTaskPath(epicKey, featureKey, taskKey, title, ...)

// Proposed (good)
pr := PathResolver{db: db}
path, err := pr.ResolveTaskPath(ctx, taskKey)
```

### Key Lookup Enhancement

Support both numeric and slugged keys in all GetByKey methods:

```go
// Both work:
shark epic get E05
shark epic get E05-task-mgmt-cli-capabilities

// Repository implementation:
func (r *EpicRepository) GetByKey(ctx, keyOrSlug string) (*Epic, error) {
    // Try exact match first
    // If not found and contains hyphen, extract numeric key
    // Return same epic
}
```

---

## Implementation Approach

### Six-Phase Implementation

#### Phase 1: Database Schema (P0 - MUST HAVE)
- Add slug columns to all tables
- Create indexes for performance
- Write migration script with backfill logic
- Test migration on development database

**Effort**: 2-3 hours
**Risk**: Low (nullable column, backward compatible)

#### Phase 2: Slug Storage (P0 - MUST HAVE)
- Update epic creation to generate and store slug
- Update feature creation to generate and store slug
- Update task creation to generate and store slug
- Test creation flow end-to-end

**Effort**: 3-4 hours
**Risk**: Low (addition, not replacement)

#### Phase 3: PathResolver (P0 - MUST HAVE)
- Implement PathResolver interface with database-first design
- Replace PathBuilder usage in all commands
- Test path resolution with default and custom paths
- Performance testing (verify 10x improvement)

**Effort**: 4-5 hours
**Risk**: Medium (internal refactor, needs thorough testing)

#### Phase 4: Key Lookup Enhancement (P1 - SHOULD HAVE)
- Update GetByKey methods in all repositories
- Support both numeric (`E05`) and slugged (`E05-slug`) keys
- Test all CLI commands with both formats
- Update help text and documentation

**Effort**: 2-3 hours
**Risk**: Low (backward compatible enhancement)

#### Phase 5: File Format Conversion (P2 - OPTIONAL)
- Convert epic/feature files to YAML frontmatter
- Update discovery to parse frontmatter
- Update sync to validate slug matches database
- Provide migration script with dry-run mode

**Effort**: 4-6 hours
**Risk**: High (breaking change for existing files)

#### Phase 6: Cleanup (P1 - SHOULD HAVE)
- Remove PathBuilder code
- Update all documentation
- Final end-to-end testing
- Performance benchmarking

**Effort**: 1-2 hours
**Risk**: Low (cleanup after validated migration)

**Total Estimated Effort**: 16-23 hours

---

## Prioritization

### P0 - Must Have (Phases 1-3)
These phases fix the architectural debt and restore database-first principles:

1. **Database Schema** - Foundation for everything else
2. **Slug Storage** - Start capturing slugs at creation time
3. **PathResolver** - Stop reading files for path resolution

**Why P0**: These fix the core architectural violation (database not being source of truth) and provide immediate performance benefits.

### P1 - Should Have (Phases 4, 6)
These phases improve user experience and clean up technical debt:

4. **Key Lookup Enhancement** - Better UX, backward compatible
6. **Cleanup** - Remove deprecated PathBuilder code

**Why P1**: Important for UX and maintainability, but not blocking core functionality.

### P2 - Nice to Have (Phase 5)
This phase standardizes file formats but is a breaking change:

5. **File Format Conversion** - YAML frontmatter for all entities

**Why P2**: Valuable for consistency and easier parsing, but can be deferred. Current markdown format works, just requires more complex parsing. This is purely a consistency/maintainability improvement.

---

## Success Metrics

### Functional Success Criteria

- ✅ All new epics/features/tasks store slugs in database at creation
- ✅ Path resolution happens without reading files
- ✅ Both numeric and slugged keys work in all commands
- ✅ Discovery validates file slugs against database slugs
- ✅ No data loss or corruption during migration

### Performance Success Criteria

- ✅ Path resolution improves from ~1ms to ~0.1ms (10x faster)
- ✅ Discovery improves by 2-3x when YAML frontmatter is used
- ✅ Database size increase is negligible (<100KB for 1000 records)

### Quality Success Criteria

- ✅ All unit tests pass
- ✅ All integration tests pass
- ✅ Migration tested on development database
- ✅ Backward compatibility maintained (except Phase 5 with opt-in migration)

---

## Testing Strategy

### Unit Tests

1. **Slug Generation**: Deterministic, unicode handling, truncation, special characters
2. **Path Resolution**: Epic/feature/task paths, custom paths, inheritance
3. **Key Extraction**: Numeric key from slugged key, edge cases

### Integration Tests

1. **Entity Creation**: Epic/feature/task with slug stored in database
2. **Discovery**: Validate file slug vs database slug, report conflicts
3. **Key Lookup**: Both numeric and slugged formats return same record
4. **Path Resolution**: No file reads, correct paths computed

### Migration Tests

1. **Schema Migration**: Columns added, indexes created
2. **Data Backfill**: Slugs extracted correctly from file_path
3. **File Format Conversion**: Metadata preserved, no data loss

---

## Breaking Changes & Mitigation

### Breaking Change: File Format Conversion (Phase 5 Only)

**Impact**: Existing epic/feature files must be converted to YAML frontmatter

**Mitigation**:
1. Provide migration script with dry-run mode
2. Backup files before conversion
3. Make Phase 5 optional - can be deferred
4. Support both formats during transition period (if needed)

### Non-Breaking Changes

All other phases are backward compatible:
- Adding slug column (nullable)
- Storing slugs at creation (new data only)
- PathResolver (internal implementation detail)
- Flexible key lookup (enhancement to existing functionality)

---

## Dependencies

### Internal Dependencies

- **Epic E07**: This is a feature within the Database & State Management epic
- **Existing Database Schema**: Must have epics, features, tasks tables
- **Existing File Structure**: Must have docs/plan/* hierarchy

### External Dependencies

- None - this is purely internal refactoring

### Technical Dependencies

- SQLite 3.x with support for ALTER TABLE
- Go 1.23.4+ with existing repository pattern
- Existing slug generation logic in `internal/slug/`

---

## Risk Assessment

### Low Risk ✅

- Adding slug column (nullable, indexed, backward compatible)
- Backfilling slugs from file_path (one-time migration)
- Flexible key lookup (backward compatible)

### Medium Risk ⚠️

- PathResolver refactor (internal change, needs thorough testing)
- Discovery updates (validate against database, new logic)
- Performance impact (should improve, but verify in real-world usage)

### High Risk ⚠️⚠️

- File format conversion (breaking change for existing files)
- **Mitigation**: Make optional (Phase 5), provide migration script, dry-run mode, backups

---

## Assumptions

1. **Database is primary data store**: All critical operations should use database as source of truth
2. **Slugs are deterministic**: Slug generation from title is deterministic and won't change
3. **Title changes are rare**: Once created, entity titles rarely change (slug immutability is acceptable)
4. **Performance matters**: 10x improvement in path resolution is valuable
5. **File system is secondary**: Files are representation of database state, not source of truth

---

## Out of Scope

The following are explicitly out of scope for this feature:

1. **Slug editing/renaming**: Slugs are immutable once created (title changes don't update slug)
2. **Slug uniqueness validation**: Slugs don't need to be globally unique (keys handle uniqueness)
3. **Custom slug generation**: Users can't provide custom slugs (always generated from title)
4. **File path migration**: Existing files stay at current paths (slug doesn't force file moves)
5. **Multi-language support**: Slug generation is English-centric (unicode normalization only)

---

## Future Considerations

Items to consider for future enhancements:

1. **Slug Regeneration Command**: Allow manual slug regeneration if needed
2. **Slug Validation Command**: Audit command to find slug/file path mismatches
3. **Custom Slug Support**: Allow users to override generated slug at creation
4. **File Path Synchronization**: Auto-move files if slug changes (with user confirmation)
5. **Slug History**: Track slug changes over time for audit trail

---

## Related Documents

- Architecture Analysis: `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-30-slug-architecture-review/architectural-analysis.md`
- Recommendations Summary: `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-30-slug-architecture-review/recommendations-summary.md`
- Architecture Diagrams: `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-30-slug-architecture-review/architecture-diagrams.md`
- Implementation Examples: `/home/jwwelbor/projects/shark-task-manager/dev-artifacts/2025-12-30-slug-architecture-review/implementation-examples.md`

---

## Approval & Sign-off

**Business Owner**: TBD
**Technical Lead**: TBD
**Estimated Effort**: 16-23 hours (M-L complexity)
**Target Start**: TBD
**Target Completion**: TBD

---

## Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2025-12-30 | 1.0 | BusinessAnalyst Agent | Initial feature overview created from architecture review |
