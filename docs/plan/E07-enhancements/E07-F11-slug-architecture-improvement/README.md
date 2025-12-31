# E07-F11: Slug Architecture Improvement

**Epic**: E07 - Database & State Management Enhancements
**Feature**: Slug Architecture Improvement
**Status**: Planning
**Priority**: High
**Created**: 2025-12-30

---

## Overview

This feature addresses critical architectural violations where the database is not the single source of truth for metadata. Currently, slugs are computed on-the-fly, path resolution reads files, and file formats are inconsistent across entity types.

**Key Problem**: File system is treated as source of truth instead of database.

**Solution**: Add slug storage to database, implement database-first path resolution, and optionally standardize file formats.

---

## Quick Links

- **[Feature Overview](./feature-overview.md)** - Business value, technical requirements, implementation approach
- **[User Stories](./user-stories.md)** - 23 detailed user stories with acceptance criteria
- **[Prioritization](./prioritization.md)** - P0/P1/P2 breakdown and implementation roadmap
- **[Architecture Analysis](../../../../dev-artifacts/2025-12-30-slug-architecture-review/architectural-analysis.md)** - Detailed technical analysis
- **[Recommendations Summary](../../../../dev-artifacts/2025-12-30-slug-architecture-review/recommendations-summary.md)** - Executive summary

---

## At a Glance

### The Problem

1. **No slug column in database** - Slugs computed on-the-fly from titles
2. **Discovery reads files** - File system becomes source of truth
3. **PathBuilder reads files** - Path resolution requires file I/O
4. **Inconsistent formats** - Tasks use YAML, epics/features use markdown body

### The Solution

1. **Add slug column to database** - Store slugs, don't compute them
2. **Store slugs at creation time** - Generate once, store forever
3. **PathResolver reads database** - No file I/O for path resolution
4. **Flexible key lookup** - Support both `E05` and `E05-slug`
5. **Optional: YAML frontmatter** - Consistent file format (breaking change)

### The Impact

- **10x faster path resolution** (0.1ms vs 1ms)
- **Database is source of truth** (architectural integrity restored)
- **Better user experience** (shorter keys accepted)
- **Consistent patterns** (all entities follow same approach)

---

## Implementation Phases

### Phase 1: Database Schema (P0) - 2-3 hours
Add slug columns to epics, features, tasks tables. Create indexes. Backfill existing slugs from file_path.

**Stories**: S-1-1, S-1-2, S-1-3

---

### Phase 2: Slug Storage (P0) - 3-4 hours
Generate slugs at creation time and store in database. Update epic, feature, task creation logic.

**Stories**: S-2-1, S-2-2, S-2-3, S-2-4

---

### Phase 3: PathResolver (P0) - 4-5 hours
Implement database-first path resolution. Replace PathBuilder with PathResolver everywhere.

**Stories**: S-3-1, S-3-2, S-3-3

---

### Phase 4: Key Lookup Enhancement (P1) - 2-3 hours
Support both numeric (`E05`) and slugged (`E05-slug`) keys in all commands.

**Stories**: S-4-1, S-4-2, S-4-3, S-4-4

---

### Phase 5: File Format Conversion (P2) - 4-6 hours **OPTIONAL**
Convert epic/feature files to YAML frontmatter for consistency. Breaking change.

**Stories**: S-5-1, S-5-2, S-5-3, S-5-4, S-5-5

---

### Phase 6: Cleanup (P1) - 1-2 hours
Remove PathBuilder, update documentation, performance benchmarking, end-to-end validation.

**Stories**: S-6-1, S-6-2, S-6-3, S-6-4

---

## Prioritization Summary

### P0 - Must Have (Phases 1-3)
**Effort**: 9-12 hours | **Stories**: 10

Fixes architectural violations. Database becomes single source of truth. 10x performance improvement.

**CRITICAL**: Must complete all P0 work to restore architectural integrity.

---

### P1 - Should Have (Phases 4, 6)
**Effort**: 3-5 hours | **Stories**: 8

Better UX (flexible key lookup), technical debt cleanup, comprehensive documentation and testing.

**RECOMMENDED**: Complete after P0 for production-ready solution.

---

### P2 - Nice to Have (Phase 5)
**Effort**: 4-6 hours | **Stories**: 5

Consistent file format across all entities. Breaking change with migration required.

**OPTIONAL**: Can be deferred to separate epic or skipped entirely.

---

## Recommended Approach

### Minimum Viable: P0 Only (9-12 hours)
Complete Phases 1, 2, 3 to fix architectural violations and achieve 10x performance improvement.

### Recommended: P0 + P1 (12-17 hours)
Complete Phases 1, 2, 3, 4, 6 for complete solution with UX improvements and cleanup.

### Complete: P0 + P1 + P2 (16-23 hours)
Complete all phases including file format standardization (breaking change).

**Decision**: Recommend **P0 + P1** approach. Defer Phase 5 to future epic or skip entirely.

---

## User Story Summary

### 23 Total Stories

**By Priority**:
- P0: 13 stories (Phases 1-3)
- P1: 8 stories (Phases 4, 6)
- P2: 5 stories (Phase 5)

**By Size**:
- XS: 2 stories
- S: 8 stories
- M: 9 stories
- L: 4 stories
- XL: 0 stories

**By Phase**:
1. Database Schema: 3 stories
2. Slug Storage: 4 stories
3. PathResolver: 3 stories
4. Key Lookup: 4 stories
5. File Format (Optional): 5 stories
6. Cleanup: 4 stories

---

## Success Criteria

### When P0 is Complete
- ✅ Database has slug columns with indexes
- ✅ Existing entities have slugs backfilled
- ✅ New entities store slugs at creation
- ✅ Path resolution uses database (no file reads)
- ✅ 5-10x performance improvement achieved

### When P1 is Complete
- ✅ Both numeric and slugged keys work
- ✅ PathBuilder removed from codebase
- ✅ Documentation reflects new architecture
- ✅ Performance benchmarks validate improvement
- ✅ End-to-end tests cover all workflows

### When P2 is Complete (Optional)
- ✅ All entities use YAML frontmatter
- ✅ Discovery parses frontmatter consistently
- ✅ Slug validation during sync
- ✅ Zero data loss during conversion

---

## Risk Assessment

### Low Risk ✅
- Adding slug column (backward compatible)
- Backfilling slugs (one-time migration)
- Flexible key lookup (enhancement)

### Medium Risk ⚠️
- PathResolver refactor (internal change)
- Discovery updates (new validation logic)

### High Risk ⚠️⚠️
- File format conversion (breaking change)
- **Mitigation**: Make optional (Phase 5), provide migration script, dry-run mode

---

## Breaking Changes

### P0 and P1: No Breaking Changes
All work is backward compatible. Existing commands work unchanged.

### P2: Breaking Changes
File format conversion requires migration of existing files. Provide migration script with dry-run mode and backups.

**Recommendation**: Defer Phase 5 to avoid breaking changes.

---

## Dependencies

### Internal
- Epic E07 (Database & State Management)
- Existing repository pattern
- Existing slug generation logic

### External
- None (internal refactoring only)

---

## Out of Scope

1. Slug editing/renaming (slugs are immutable)
2. Custom slug support (always generated from title)
3. File path migration (files stay at current paths)
4. Multi-language slug support (English-centric)
5. Slug uniqueness enforcement (keys handle uniqueness)

---

## Next Steps

1. **Review Documents**
   - Feature overview
   - User stories
   - Prioritization

2. **Make Decisions**
   - Scope: P0 only, P0+P1, or P0+P1+P2?
   - Timeline: When to start?
   - Resources: Who will implement?

3. **Create Implementation Tasks**
   - Use `/task` command to generate agent-executable tasks
   - Start with Phase 1 (Database Schema)

4. **Begin Implementation**
   - Phase 1: Add slug columns and backfill
   - Phase 2: Store slugs at creation
   - Phase 3: Implement PathResolver

---

## Document Status

- ✅ **Feature Overview**: Complete
- ✅ **User Stories**: Complete (23 stories)
- ✅ **Prioritization**: Complete (P0/P1/P2 breakdown)
- ✅ **Architecture Analysis**: Complete (in dev-artifacts)
- ✅ **README**: Complete (this document)

**Ready for**: Stakeholder review and decision on scope

---

## Related Architecture Review

This feature is based on a comprehensive architecture review conducted on 2025-12-30:

- **[Architectural Analysis](../../../../dev-artifacts/2025-12-30-slug-architecture-review/architectural-analysis.md)** (15,000 words)
- **[Recommendations Summary](../../../../dev-artifacts/2025-12-30-slug-architecture-review/recommendations-summary.md)** (3,000 words)
- **[Architecture Diagrams](../../../../dev-artifacts/2025-12-30-slug-architecture-review/architecture-diagrams.md)** (ASCII diagrams)
- **[Implementation Examples](../../../../dev-artifacts/2025-12-30-slug-architecture-review/implementation-examples.md)** (Code samples)

---

## Contact

**Created By**: BusinessAnalyst Agent
**Date**: 2025-12-30
**Epic**: E07 - Database & State Management Enhancements
**Feature**: E07-F11 - Slug Architecture Improvement

For questions or clarifications, refer to the detailed documents in this directory.
