# Epic E07 Architecture Review - Executive Summary

**Date**: 2025-12-18
**Epic**: E07 - Enhancements
**Features Reviewed**: 7
**Overall Status**: ✅ All features architecturally approved

---

## Quick Reference

| Feature | Complexity | Risk | Effort | Priority |
|---------|-----------|------|--------|----------|
| E07-F02: Task Title Positional | Simple | LOW | 2-3h | 1 (Quick win) |
| E07-F01: Remove Agent Requirement | Simple | LOW | 4-6h | 2 (Quick win) |
| E07-F07: Discovery Integration | Complex | MEDIUM | 16-20h | 3 (Foundation) |
| E07-F03: Move Task Files | Moderate | MEDIUM | 8-10h | 4 (Needs F07) |
| E07-F06: Force Status | Moderate | MEDIUM | 8-10h | 5 (Parallel) |
| E07-F04: Execution Order | Moderate | LOW | 6-8h | 6 (Parallel) |
| E07-F05: Related Documents | Complex | MEDIUM | 12-16h | 7 (Most complex) |

**Total Effort**: 56-73 hours (7-9 developer days)

---

## Implementation Roadmap

### Phase 1: Quick Wins (Week 1)
**Goal**: Deliver immediate UX improvements

- **E07-F02**: Make task title positional argument
  - Simple CLI change, matches epic/feature pattern
  - Breaking change but easy to communicate

- **E07-F01**: Remove agent requirement
  - Reduces friction in task creation
  - Supports custom templates

### Phase 2: Discovery Foundation (Week 2)
**Goal**: Enable advanced features

- **E07-F07**: Epic Index Discovery Integration
  - Wires existing discovery package to sync command
  - Enables epic-index.md parsing and conflict resolution
  - Unblocks F03 (slug resolution needed)

### Phase 3: File Organization (Week 2-3)
**Goal**: Improve project structure

- **E07-F03**: Move task files to feature folder
  - Creates hierarchical docs structure
  - Depends on F07 for slug resolution

### Phase 4: Database Enhancements (Week 3-4)
**Goal**: Add power-user features

- **E07-F06**: Allow force status on task
  - Administrative override for status changes
  - Cascading feature → tasks updates

- **E07-F04**: Implementation order
  - Adds execution_order field to guide work sequence
  - Can run parallel with F06

### Phase 5: Advanced Features (Week 4-5)
**Goal**: Complete the epic

- **E07-F05**: Add related documents
  - Most complex schema changes
  - Many-to-many relationships
  - Comprehensive CLI commands

---

## Key Architecture Decisions

### 1. Optional Agent Field (F01)
**Decision**: Remove required validation, allow NULL
**Impact**: Simpler, more flexible workflow
**Risk**: None - backward compatible

### 2. Positional Title Argument (F02)
**Decision**: Accept breaking change for consistency
**Impact**: Better UX, matches epic/feature commands
**Risk**: Breaking change - needs migration guide

### 3. Hierarchical Task Files (F03)
**Decision**: Place tasks under feature/tasks/ folder
**Impact**: Better organization, easier navigation
**Risk**: Requires slug resolution (depends on F07)

### 4. Execution Order as Integer (F04)
**Decision**: Nullable integer field, not JSON
**Impact**: Simple sorting, standard SQL patterns
**Risk**: None - can evolve later if needed

### 5. Normalized Document Storage (F05)
**Decision**: Separate documents table with junction tables
**Impact**: Proper many-to-many, document reuse
**Risk**: Complex schema but correct design

### 6. Force Flag Pattern (F06)
**Decision**: Use --force flag, not separate commands
**Impact**: Clear intent, standard CLI pattern
**Risk**: Need audit trail (forced=true in history)

### 7. Discovery Opt-In (F07)
**Decision**: Explicit --index flag, not auto-detect
**Impact**: Predictable behavior, user control
**Risk**: None - backward compatible

---

## Database Schema Changes

### Coordinated Migration Script Needed

```sql
-- E07-enhancements.sql (unified migration)

-- F04: Execution order
ALTER TABLE features ADD COLUMN execution_order INTEGER NULL;
ALTER TABLE tasks ADD COLUMN execution_order INTEGER NULL;

-- F05: Related documents
CREATE TABLE documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, file_path)
);

CREATE TABLE epic_documents (
    epic_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (epic_id, document_id),
    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE TABLE feature_documents (
    feature_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (feature_id, document_id),
    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE TABLE task_documents (
    task_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (task_id, document_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

-- F06: Force tracking (optional)
ALTER TABLE task_history ADD COLUMN forced BOOLEAN DEFAULT FALSE;
```

---

## Dependency Graph

```
F02 (Title Positional) ────────────┐
                                   │
F01 (Remove Agent Req) ────────────┤
                                   │
                                   ├──> [No dependencies]
                                   │
F07 (Discovery Integration) ───────┘
    │
    └──> F03 (Move Task Files) [Needs slug resolution]

F06 (Force Status) ─────┐
                        ├──> [Can run in parallel]
F04 (Execution Order) ──┘

F05 (Related Docs) ─────> [Most complex, last]
```

---

## Risk Assessment

### Overall Risk: LOW to MEDIUM

**Low Risk Features** (F01, F02, F04):
- Simple changes
- Standard patterns
- Well-understood implementations

**Medium Risk Features** (F03, F05, F06, F07):
- F03: Slug resolution dependency
- F05: Complex schema with foreign keys
- F06: Transaction complexity in cascade
- F07: Integration complexity, performance concerns

**Mitigation Strategies**:
1. **F03**: Implement after F07 to leverage discovery
2. **F05**: Comprehensive transaction testing
3. **F06**: Robust rollback testing
4. **F07**: Performance benchmarks on large projects

---

## Success Criteria

### Technical Success
- [ ] All 7 features implemented and tested
- [ ] Database migrations execute cleanly
- [ ] Integration tests pass for feature combinations
- [ ] Performance benchmarks meet targets (F07 < 5s for 200 features)
- [ ] Backward compatibility maintained (existing features work)

### User Success
- [ ] F02 migration guide published
- [ ] F07 conflict resolution UX is clear
- [ ] Documentation updated for all new features
- [ ] CLI help text accurate and helpful

### Quality Gates
- [ ] Unit test coverage > 80%
- [ ] Integration tests for cross-feature workflows
- [ ] No SQL injection vulnerabilities
- [ ] All errors properly wrapped with context
- [ ] golangci-lint passes with no warnings

---

## Next Steps

1. **Approve Architecture** ✅ (This document)
2. **Generate Implementation Tasks**
   - Use `/task` command to generate tasks for each feature
   - Follow recommended implementation order
3. **Create Unified Migration Script**
   - Single migration for all schema changes
4. **Implement Phase 1** (F02, F01)
   - Quick wins to build momentum
5. **Implement Phase 2** (F07)
   - Foundation for dependent features
6. **Continue Sequential Implementation**
   - Follow dependency graph
7. **Integration Testing**
   - Test feature combinations
8. **Release Preparation**
   - Migration guide for F02
   - Performance validation for F07
   - Documentation updates

---

## Recommendations

### Immediate Actions
1. ✅ Approve this architecture review
2. Generate implementation tasks using shark CLI
3. Assign features to implementation sprints
4. Set up performance benchmarking for F07

### Development Practices
1. Implement in recommended order (respect dependencies)
2. Use unified migration script for schema changes
3. Write integration tests for feature combinations
4. Document breaking changes (F02) clearly

### Testing Strategy
1. Unit tests for each feature independently
2. Integration tests for F03+F07, F04+F06
3. Performance tests for F07 (large epic-index files)
4. End-to-end workflow tests with all features

---

## Contact

For questions or clarification on architectural decisions:
- Review full details in `E07-ARCHITECTURE-REVIEW.md`
- Architecture Decision Records (ADRs) documented in review
- Implementation guidance provided for each feature

---

**Status**: ✅ Architecture Review Complete
**Recommendation**: Proceed to task generation and implementation
