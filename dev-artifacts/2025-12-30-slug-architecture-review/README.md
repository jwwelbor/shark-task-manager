# Slug Architecture Review - Complete Analysis

**Date**: 2025-12-30
**Reviewer**: Claude (Architect Agent)
**Status**: Ready for Review and Decision

---

## Overview

This directory contains a comprehensive architectural review of slug generation, path resolution, and file path management across epics, features, and tasks in the Shark Task Manager system.

The analysis identified critical architectural issues where the database is not the single source of truth, leading to:
- Performance overhead from file reads
- Inconsistent metadata storage patterns
- Complex path resolution logic
- Poor user experience with key requirements

---

## Documents in This Review

### 1. [architectural-analysis.md](./architectural-analysis.md) - Full Technical Analysis

**Length**: ~15,000 words
**Purpose**: Comprehensive deep-dive into current vs. proposed architecture

**Contents**:
- Executive summary of critical issues
- Current state analysis (database schema, file formats, path resolution)
- Root cause analysis (why database is not source of truth)
- Desired end state with proposed changes
- Migration strategy (6 phases)
- Code architecture recommendations
- Testing strategy
- Performance impact analysis
- Backward compatibility considerations

**Audience**: Technical stakeholders, implementation team

---

### 2. [recommendations-summary.md](./recommendations-summary.md) - Executive Summary

**Length**: ~3,000 words
**Purpose**: High-level summary for quick decision-making

**Contents**:
- TL;DR (3 sentences)
- Critical issues (P0 list)
- Proposed database schema changes
- Proposed file format (YAML frontmatter)
- Implementation phases (6 phases, 16-23 hours)
- Breaking changes and risks
- Performance improvements (10x faster path resolution)
- Decision points for stakeholders
- Quick reference diagrams

**Audience**: Product managers, tech leads, decision-makers

---

### 3. [architecture-diagrams.md](./architecture-diagrams.md) - Visual Architecture

**Length**: ~2,000 words (ASCII diagrams)
**Purpose**: Visual representation of current vs. proposed architecture

**Contents**:
- Current data flow (epic creation, discovery)
- Current problem visualization
- Proposed data flow (database-first)
- Proposed solution visualization
- Path resolution comparison (current vs. proposed)
- Key lookup flexibility diagram
- Migration flow diagram
- Performance impact diagrams
- Before/after summary table

**Audience**: All technical stakeholders (visual learners)

---

### 4. [implementation-examples.md](./implementation-examples.md) - Code Samples

**Length**: ~4,000 words (code-heavy)
**Purpose**: Concrete implementation guidance

**Contents**:
1. Database migration script (add slug columns, backfill)
2. PathResolver implementation (database-first resolution)
3. Updated EpicRepository (flexible key lookup)
4. Updated epic creation (store slug at creation)
5. Updated discovery (validate slug)
6. Updated models (add slug field)
7. CLI migration command
8. Testing examples

**Audience**: Implementation developers

---

## Quick Navigation

### For Decision-Makers
1. Start with [recommendations-summary.md](./recommendations-summary.md)
2. Review "Decision Points" section
3. Check "Risk Assessment" and "Breaking Changes"
4. Decide on:
   - File format (YAML vs markdown)
   - Migration timing (all-at-once vs phased)
   - Slug for tasks (include or skip)

### For Architects
1. Read [architectural-analysis.md](./architectural-analysis.md) in full
2. Review [architecture-diagrams.md](./architecture-diagrams.md) for visual context
3. Assess "Root Cause Analysis" section
4. Validate proposed schema changes
5. Review migration strategy

### For Implementers
1. Read [recommendations-summary.md](./recommendations-summary.md) for context
2. Study [implementation-examples.md](./implementation-examples.md) for code samples
3. Review [architectural-analysis.md](./architectural-analysis.md) "Proposed Architecture" sections
4. Follow implementation phases in order

### For Project Managers
1. Read [recommendations-summary.md](./recommendations-summary.md)
2. Note "Implementation Phases" (6 phases, 16-23 hours)
3. Review "Breaking Changes" section
4. Plan migration timeline based on phases

---

## Key Findings Summary

### Critical Issues (Must Fix)

1. **Database Is Not Single Source of Truth**
   - Discovery reads filesystem → updates database
   - Violates core architectural principle
   - Causes data integrity issues

2. **No Slug Column in Database**
   - Slugs computed on-the-fly from titles
   - Performance overhead (1ms per path resolution)
   - Non-deterministic paths if slug logic changes

3. **Inconsistent File Formats**
   - Tasks use YAML frontmatter ✅
   - Epics/features use markdown body ❌
   - Different parsing logic required

4. **Key Format Confusion**
   - Users must know `E05-task-mgmt-cli-capabilities`
   - Database only stores `E05`
   - Poor UX, confusing documentation

---

## Recommended Solution

### Database Changes

```sql
-- Add slug columns to all tables
ALTER TABLE epics ADD COLUMN slug TEXT;
ALTER TABLE features ADD COLUMN slug TEXT;
ALTER TABLE tasks ADD COLUMN slug TEXT;

-- Create indexes
CREATE INDEX idx_epics_slug ON epics(slug);
CREATE INDEX idx_features_slug ON features(slug);
CREATE INDEX idx_tasks_slug ON tasks(slug);

-- Backfill from file_path (one-time migration)
UPDATE epics SET slug = ... WHERE file_path IS NOT NULL;
UPDATE features SET slug = ... WHERE file_path IS NOT NULL;
UPDATE tasks SET slug = ... WHERE file_path IS NOT NULL;
```

### File Format Standardization

**Proposed**: YAML frontmatter for all entities (consistent with tasks)

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

### Path Resolution Refactor

**Current**: PathBuilder (computes slugs, reads files)
**Proposed**: PathResolver (reads from database only)

```go
// Current (bad)
pb := PathBuilder{}
path := pb.ResolveTaskPath(epicKey, featureKey, taskKey, title, ...)

// Proposed (good)
pr := PathResolver{db: db}
path, err := pr.ResolveTaskPath(ctx, taskKey)
```

**Performance**: 10x faster (0.1ms vs 1ms per call)

---

## Implementation Plan

### Phase 1: Database Schema (2-3 hours) ✅ FIRST
- Add slug columns to epics, features, tasks
- Create indexes
- Write migration script
- Backfill existing slugs from file_path

### Phase 2: Slug Storage (3-4 hours)
- Update epic creation to store slug
- Update feature creation to store slug
- Update task creation to store slug
- Test creation flow

### Phase 3: PathResolver (4-5 hours)
- Implement PathResolver interface
- Replace PathBuilder usage in all commands
- Test path resolution

### Phase 4: Key Lookup Enhancement (2-3 hours)
- Update GetByKey methods in repositories
- Support both numeric (`E05`) and slugged (`E05-slug`) keys
- Test CLI commands with both formats

### Phase 5: File Format Conversion (4-6 hours) - OPTIONAL
- Decide on file format (YAML vs markdown)
- Write conversion script
- Update discovery to parse new format
- Test sync operation

### Phase 6: Cleanup (1-2 hours)
- Remove PathBuilder code
- Update documentation
- Final end-to-end testing

**Total Estimated Effort**: 16-23 hours

---

## Decision Points

### Decision 1: File Format
**Question**: Use YAML frontmatter or keep markdown body format?

**Options**:
- A. YAML frontmatter (recommended) - Consistent with tasks, easier parsing
- B. Markdown body with slug field - Minimal change, human-readable

**Recommendation**: Option A for consistency and machine-readability

---

### Decision 2: Slug for Tasks
**Question**: Add slug column to tasks table?

**Options**:
- A. Add slug column (recommended) - Consistency across all entities
- B. Skip tasks - Already have slug in filename

**Recommendation**: Option A for consistency and future flexibility

---

### Decision 3: Migration Timing
**Question**: Implement all phases at once or phased rollout?

**Options**:
- A. All phases at once - Faster, riskier
- B. Phased rollout - Slower, safer

**Recommendation**: Option B (phased) to minimize risk and allow testing between phases

---

### Decision 4: Breaking Changes
**Question**: Accept breaking changes for file format conversion?

**Options**:
- A. Convert all files - Clean break, better long-term architecture
- B. Support both formats - Gradual migration, more complex code

**Recommendation**: Option A with migration script and dry-run mode

---

## Success Criteria

### Phase 1 Complete
- ✅ Slug columns exist in all tables
- ✅ Indexes created
- ✅ Existing slugs backfilled from file_path
- ✅ New entity creation stores slugs

### Phase 2 Complete
- ✅ All creation commands store slugs
- ✅ Slugs are immutable (don't change with title)
- ✅ File paths match database slugs

### Phase 3 Complete
- ✅ PathResolver implemented
- ✅ All commands use PathResolver
- ✅ No file reads for path determination
- ✅ 10x faster path resolution

### Phase 4 Complete
- ✅ GetByKey supports numeric keys (`E05`)
- ✅ GetByKey supports slugged keys (`E05-slug`)
- ✅ Both formats work in all CLI commands

### Phase 5 Complete (Optional)
- ✅ All epic/feature files use YAML frontmatter
- ✅ Discovery parses frontmatter
- ✅ Sync validates slug matches database

### Final Success
- ✅ Database is single source of truth
- ✅ Slugs stored and validated
- ✅ Consistent patterns across all entities
- ✅ No performance degradation
- ✅ Backward compatible (with migration)

---

## Risk Assessment

### Low Risk ✅
- Adding slug column (nullable, indexed)
- Backfilling slugs from file_path
- Flexible key lookup (backward compatible)

### Medium Risk ⚠️
- PathResolver refactor (internal change, needs testing)
- Discovery updates (validate against database)
- Performance impact (should improve, but verify)

### High Risk ⚠️⚠️
- File format conversion (breaking change)
- **Mitigation**: Migration script, dry-run mode, backups, phased rollout

---

## Next Steps

1. **Stakeholder Review** (1-2 days)
   - Present [recommendations-summary.md](./recommendations-summary.md) to team
   - Discuss decision points
   - Get consensus on approach

2. **Make Decisions** (1 day)
   - File format: YAML vs markdown
   - Migration timing: All-at-once vs phased
   - Breaking changes: Accept or defer

3. **Create Implementation Tasks** (1 day)
   - Break down phases into granular tasks
   - Assign owners and timelines
   - Set up test environment

4. **Start Phase 1** (2-3 hours)
   - Implement database migration
   - Test on development database
   - Backfill slugs

5. **Iterate Through Phases** (2-3 weeks)
   - Implement one phase at a time
   - Test thoroughly after each phase
   - Review and refine before next phase

---

## Questions for Review

1. **File Format**: Do we accept YAML frontmatter for epics/features?
2. **Migration Timing**: Phased rollout or all-at-once?
3. **Breaking Changes**: Can we convert existing files or must support both formats?
4. **Slug for Tasks**: Include slug column for tasks?
5. **Timeline**: Can we allocate 16-23 hours over 2-3 weeks?

---

## Contact

**Reviewer**: Claude (Architect Agent)
**Date**: 2025-12-30
**Review ID**: slug-architecture-2025-12-30

For questions or clarifications, refer to the detailed analysis documents in this directory.

---

## Document Status

- ✅ **architectural-analysis.md**: Complete (15,000 words)
- ✅ **recommendations-summary.md**: Complete (3,000 words)
- ✅ **architecture-diagrams.md**: Complete (2,000 words, ASCII diagrams)
- ✅ **implementation-examples.md**: Complete (4,000 words, code samples)
- ✅ **README.md**: Complete (this document)

**Total Analysis**: ~24,000 words, ready for stakeholder review.
