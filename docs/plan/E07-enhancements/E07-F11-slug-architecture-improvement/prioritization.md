# Feature Prioritization: Slug Architecture Improvement

**Feature**: E07-F11 - Slug Architecture Improvement
**Epic**: E07 - Database & State Management Enhancements
**Created**: 2025-12-30

---

## Executive Summary

This document defines the prioritization strategy for implementing the slug architecture improvement feature. The work is divided into 6 phases with clear priority levels (P0, P1, P2) based on:

1. **Architectural integrity** - Fixing database-first violations
2. **Performance impact** - Measurable improvements
3. **User experience** - Usability enhancements
4. **Technical debt** - Code maintainability

**Total Effort**: 16-23 hours across 23 user stories

---

## Priority Definitions

### P0 - Must Have (Critical)
Work that **MUST** be completed to fix architectural violations and restore database-first principles. Without this work, the system violates core design principles and accumulates technical debt.

**Characteristics**:
- Fixes architectural violations
- Provides foundation for future work
- Low risk to implement
- High value to technical quality

### P1 - Should Have (Important)
Work that **SHOULD** be completed to improve user experience and clean up technical debt. Valuable but not blocking core functionality.

**Characteristics**:
- Enhances user experience
- Reduces technical debt
- Improves maintainability
- Can be deferred if needed

### P2 - Nice to Have (Optional)
Work that **COULD** be completed for consistency and better patterns, but involves breaking changes and can be deferred indefinitely.

**Characteristics**:
- Improves consistency
- Breaking changes required
- Can be deferred or skipped
- Lower immediate value

---

## P0 - Must Have Work

### Phase 1: Database Schema (2-3 hours)

**Priority**: **P0 - CRITICAL**

#### Stories
1. **S-1-1**: Add Slug Columns to Database (S)
2. **S-1-2**: Backfill Slugs from Existing File Paths (M)
3. **S-1-3**: Create Migration CLI Command (M)

#### Why P0?
- **Foundation for everything else**: All other work depends on slug columns existing
- **Fixes architectural violation**: Database must store all metadata (currently missing slugs)
- **Low risk**: Adding nullable columns is backward compatible
- **Immediate value**: Enables slug storage for new entities

#### What This Achieves
- Database can store slugs (currently only in file paths)
- Existing entities get slugs backfilled from file_path
- Migration is reproducible and testable
- Database schema matches architectural intent

#### Acceptance Criteria for Phase 1 Completion
- ✅ Slug columns exist in epics, features, tasks tables
- ✅ Indexes created on all slug columns
- ✅ Existing slugs extracted from file_path and stored
- ✅ Migration command works with dry-run mode
- ✅ Migration is idempotent (can run multiple times safely)

---

### Phase 2: Slug Storage (3-4 hours)

**Priority**: **P0 - CRITICAL**

#### Stories
1. **S-2-1**: Generate and Store Slug During Epic Creation (S)
2. **S-2-2**: Generate and Store Slug During Feature Creation (S)
3. **S-2-3**: Generate and Store Slug During Task Creation (S)
4. **S-2-4**: Test Slug Storage Across All Entity Types (M)

#### Why P0?
- **Starts capturing slugs**: New entities must store slugs to build database state
- **Makes database source of truth**: Slugs no longer computed on-the-fly
- **Low risk**: Addition to existing creation logic
- **Immediate value**: Every new entity gets a slug stored

#### What This Achieves
- All new epics store slugs at creation time
- All new features store slugs at creation time
- All new tasks store slugs at creation time
- Slugs are immutable (generated once, stored forever)
- File paths use stored slugs (consistent with database)

#### Acceptance Criteria for Phase 2 Completion
- ✅ Epic creation stores slug in database
- ✅ Feature creation stores slug in database
- ✅ Task creation stores slug in database
- ✅ File paths match database slugs
- ✅ Integration tests verify slug storage

---

### Phase 3: PathResolver (4-5 hours)

**Priority**: **P0 - CRITICAL**

#### Stories
1. **S-3-1**: Implement PathResolver Interface (M)
2. **S-3-2**: Replace PathBuilder with PathResolver in Commands (L)
3. **S-3-3**: Performance Testing for PathResolver (M)

#### Why P0?
- **Fixes architectural violation**: Path resolution must be database-driven, not file-driven
- **Eliminates file reads**: PathBuilder reads files, PathResolver reads database
- **Performance improvement**: 10x faster path resolution (0.1ms vs 1ms)
- **Completes migration**: With this, database is truly the source of truth

#### What This Achieves
- Path resolution reads from database only (no file I/O)
- PathBuilder is replaced everywhere
- 10x performance improvement for path operations
- Database is single source of truth for all metadata

#### Acceptance Criteria for Phase 3 Completion
- ✅ PathResolver implemented with database-first design
- ✅ All commands use PathResolver (PathBuilder not imported anywhere)
- ✅ Path resolution is 5-10x faster than before
- ✅ No file reads during path determination
- ✅ All tests pass with PathResolver

---

## P0 Summary

**Total P0 Effort**: 9-12 hours
**Total P0 Stories**: 10 stories

**When P0 is Complete**:
- ✅ Database is single source of truth for all metadata
- ✅ Slugs stored in database (not computed on-the-fly)
- ✅ Path resolution is database-driven (no file reads)
- ✅ 10x performance improvement achieved
- ✅ Architectural violation fixed

**P0 work MUST be completed together** - each phase builds on the previous. Do not skip any P0 phase.

---

## P1 - Should Have Work

### Phase 4: Key Lookup Enhancement (2-3 hours)

**Priority**: **P1 - IMPORTANT**

#### Stories
1. **S-4-1**: Support Numeric and Slugged Keys in Epic Repository (S)
2. **S-4-2**: Support Numeric and Slugged Keys in Feature Repository (S)
3. **S-4-3**: Support Numeric and Slugged Keys in Task Repository (S)
4. **S-4-4**: Update CLI Commands to Accept Both Key Formats (M)

#### Why P1?
- **User experience improvement**: Users can use `E05` instead of `E05-task-mgmt-cli-capabilities`
- **Backward compatible**: Enhancement, not breaking change
- **Low risk**: Simple string manipulation
- **High user value**: Makes CLI more usable

#### Why Not P0?
- Not architecturally critical (database-first is already restored by P0)
- Users can still use numeric keys (this just adds slugged key support)
- Can be deferred without blocking other work

#### What This Achieves
- Both `E05` and `E05-slug` work in all commands
- Better user experience (shorter keys accepted)
- Backward compatible (existing usage unchanged)
- Clear documentation about both formats

#### Acceptance Criteria for Phase 4 Completion
- ✅ Repositories support both numeric and slugged keys
- ✅ All CLI commands accept both formats
- ✅ Help text explains both formats work
- ✅ Error messages are clear for invalid keys

---

### Phase 6: Cleanup (1-2 hours)

**Priority**: **P1 - IMPORTANT**

#### Stories
1. **S-6-1**: Deprecate and Remove PathBuilder (S)
2. **S-6-2**: Update Project Documentation (M)
3. **S-6-3**: Performance Benchmarking and Reporting (M)
4. **S-6-4**: End-to-End Validation (L)

#### Why P1?
- **Reduces technical debt**: PathBuilder is deprecated after P0 Phase 3
- **Improves maintainability**: One way to resolve paths (PathResolver)
- **Documentation accuracy**: Docs should reflect actual implementation
- **Validation**: End-to-end tests ensure everything works

#### Why Not P0?
- PathBuilder is already replaced in P0 Phase 3 (just not deleted)
- Documentation can be updated later
- System works without this cleanup

#### What This Achieves
- PathBuilder code removed from codebase
- Documentation reflects new architecture
- Performance benchmarks demonstrate improvement
- Comprehensive end-to-end tests

#### Acceptance Criteria for Phase 6 Completion
- ✅ PathBuilder code deleted
- ✅ All documentation updated
- ✅ Performance benchmarks show 10x improvement
- ✅ End-to-end tests pass for all workflows

---

## P1 Summary

**Total P1 Effort**: 3-5 hours
**Total P1 Stories**: 8 stories

**When P1 is Complete**:
- ✅ Users can use either numeric or slugged keys
- ✅ PathBuilder is completely removed
- ✅ Documentation is accurate and complete
- ✅ Performance improvements are validated
- ✅ Technical debt is cleaned up

**P1 work SHOULD be completed** but can be deferred if time is constrained. Completing P0 work is sufficient to fix the architectural issues.

---

## P2 - Nice to Have Work

### Phase 5: File Format Conversion (4-6 hours) - OPTIONAL

**Priority**: **P2 - OPTIONAL**

#### Stories
1. **S-5-1**: Design YAML Frontmatter Template for Epics (XS)
2. **S-5-2**: Design YAML Frontmatter Template for Features (XS)
3. **S-5-3**: Create Migration Script to Convert Epic Files (L)
4. **S-5-4**: Create Migration Script to Convert Feature Files (L)
5. **S-5-5**: Update Discovery to Parse YAML Frontmatter (M)

#### Why P2?
- **Consistency improvement**: Makes all entities use same file format
- **Easier parsing**: YAML frontmatter is easier than markdown body parsing
- **Better validation**: Structured metadata is easier to validate

#### Why Not P1 or P0?
- **Breaking change**: Existing files must be converted
- **Current format works**: Markdown body parsing works fine, just less elegant
- **High risk**: File format migration can cause data loss if bugs exist
- **Can be deferred indefinitely**: System works perfectly without this

#### What This Achieves
- All entities (epics, features, tasks) use YAML frontmatter
- Consistent parsing logic across all entity types
- Easier validation and metadata extraction
- Better consistency with task files

#### Risks
- **High risk of data loss** if conversion has bugs
- **Breaking change** - old discovery won't work after conversion
- **User disruption** - requires migration of existing files
- **Effort vs value** - significant effort for consistency gain only

#### Mitigation
- Provide dry-run mode
- Create backups before conversion
- Phased rollout (convert one entity type at a time)
- Support both formats during transition (if needed)

#### Acceptance Criteria for Phase 5 Completion
- ✅ YAML frontmatter templates designed and documented
- ✅ Migration scripts with dry-run mode
- ✅ All epic files converted successfully
- ✅ All feature files converted successfully
- ✅ Discovery parses YAML frontmatter
- ✅ Validation compares file slug vs database slug

---

## P2 Summary

**Total P2 Effort**: 4-6 hours
**Total P2 Stories**: 5 stories

**When P2 is Complete**:
- ✅ All entities use consistent file format (YAML frontmatter)
- ✅ Parsing is simpler and faster
- ✅ Validation is easier

**P2 work is OPTIONAL** and can be skipped entirely or deferred to a future epic. The system works perfectly well with P0 complete and P1 providing polish.

---

## Implementation Roadmap

### Minimum Viable Implementation (P0 Only)

**Effort**: 9-12 hours
**Phases**: 1, 2, 3
**Stories**: 10 stories

**Result**: Database is single source of truth, 10x performance improvement, architectural violations fixed.

**Recommendation**: **This is the minimum required work.** All P0 phases must be completed.

---

### Recommended Implementation (P0 + P1)

**Effort**: 12-17 hours
**Phases**: 1, 2, 3, 4, 6
**Stories**: 18 stories

**Result**: Everything in P0 PLUS better UX, technical debt cleaned up, comprehensive documentation and validation.

**Recommendation**: **This is the recommended scope.** Provides complete solution with UX improvements and cleanup.

---

### Complete Implementation (P0 + P1 + P2)

**Effort**: 16-23 hours
**Phases**: 1, 2, 3, 4, 5, 6
**Stories**: 23 stories

**Result**: Everything in P0 + P1 PLUS consistent file format across all entities.

**Recommendation**: **Only do this if consistency is very important.** Phase 5 is a breaking change with significant risk. Consider deferring to a separate epic.

---

## Decision Matrix

| Criterion | P0 | P1 | P2 |
|-----------|----|----|-----|
| Fixes architectural violation | ✅ Yes | ❌ No | ❌ No |
| Performance improvement | ✅ Yes (10x) | ❌ No | ✅ Yes (2-3x discovery) |
| Breaking changes | ❌ No | ❌ No | ✅ Yes |
| User experience improvement | ❌ No | ✅ Yes | ❌ No |
| Technical debt reduction | ✅ Yes | ✅ Yes | ✅ Yes |
| Risk level | 🟢 Low | 🟢 Low | 🔴 High |
| Can be deferred | ❌ No | ✅ Yes | ✅ Yes |
| Backward compatible | ✅ Yes | ✅ Yes | ❌ No |

---

## Recommended Approach

### Stage 1: P0 Implementation (Week 1)
**Effort**: 9-12 hours

1. Implement Phase 1: Database Schema (2-3 hours)
2. Test migration on development database
3. Implement Phase 2: Slug Storage (3-4 hours)
4. Test entity creation end-to-end
5. Implement Phase 3: PathResolver (4-5 hours)
6. Performance testing and validation

**Gate**: All P0 acceptance criteria met, architectural violations fixed

---

### Stage 2: P1 Implementation (Week 2)
**Effort**: 3-5 hours

1. Implement Phase 4: Key Lookup Enhancement (2-3 hours)
2. Test both key formats in all commands
3. Implement Phase 6: Cleanup (1-2 hours)
4. Update documentation
5. Performance benchmarking
6. End-to-end validation

**Gate**: All P1 acceptance criteria met, UX improved, technical debt cleaned up

---

### Stage 3: P2 Consideration (Future)
**Effort**: 4-6 hours (if pursued)

**Decision Point**: Do we need consistent file formats?

**If YES**:
1. Design templates carefully
2. Implement migration scripts with dry-run
3. Test on subset of files
4. Create backups
5. Phased rollout (epics first, then features)
6. Update discovery and validation

**If NO**:
- Document decision to defer
- Close Phase 5 stories as "won't do"
- System works perfectly with current file formats

---

## Risk Mitigation Strategy

### P0 Risks (Low)
- **Risk**: Migration breaks existing database
- **Mitigation**: Idempotent migration, dry-run mode, backups, rollback plan

- **Risk**: PathResolver slower than expected
- **Mitigation**: Performance testing before full rollout, benchmark suite

### P1 Risks (Low)
- **Risk**: Key extraction logic has edge cases
- **Mitigation**: Comprehensive unit tests, fuzzing, error handling

- **Risk**: Documentation becomes outdated
- **Mitigation**: Review documentation as part of completion criteria

### P2 Risks (High)
- **Risk**: File conversion loses data
- **Mitigation**: Dry-run, backups, phased rollout, validation after conversion

- **Risk**: Discovery breaks on converted files
- **Mitigation**: Support both formats during transition, gradual migration

---

## Success Metrics

### P0 Success (Must Achieve)
- ✅ Database schema includes slug columns
- ✅ All new entities store slugs at creation
- ✅ Path resolution uses database (no file reads)
- ✅ 5-10x performance improvement in path resolution
- ✅ Zero data loss during migration

### P1 Success (Should Achieve)
- ✅ Both numeric and slugged keys work in all commands
- ✅ PathBuilder completely removed from codebase
- ✅ Documentation reflects actual architecture
- ✅ Performance benchmarks demonstrate improvement
- ✅ End-to-end tests cover all workflows

### P2 Success (Optional)
- ✅ All epic/feature files use YAML frontmatter
- ✅ Discovery parses frontmatter consistently
- ✅ Slug validation during sync
- ✅ Zero data loss during file format conversion

---

## Out of Scope (All Priorities)

The following are explicitly out of scope:

1. **Slug editing/renaming**: Slugs are immutable
2. **Custom slug support**: Slugs always generated from title
3. **File path migration**: Files stay at current paths
4. **Multi-language slug support**: English-centric normalization only
5. **Slug uniqueness enforcement**: Keys handle uniqueness, not slugs

---

## Approval & Sign-off

**Business Owner**: TBD
**Technical Lead**: TBD
**Recommended Scope**: **P0 + P1** (12-17 hours)
**Optional Scope**: **P2** (defer to future epic)

---

## Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2025-12-30 | 1.0 | BusinessAnalyst Agent | Initial prioritization document |
