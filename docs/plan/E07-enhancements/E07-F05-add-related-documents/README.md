# E07-F05 Add Related Documents Feature

## Overview

This directory contains the complete implementation plan for the **Add Related Documents** feature (E07-F05), which enables linking supporting documents (architecture diagrams, design docs, QA reports, API specs) to epics, features, and tasks.

## Documents in This Directory

### Implementation Plans (Start Here)

1. **QUICK-START.md** - Quick overview for getting started
   - Feature summary
   - Implementation sequence
   - Critical requirements
   - Common pitfalls

2. **EXECUTIVE-SUMMARY.md** - High-level overview
   - Quick timeline and effort estimates
   - Key requirements by category (Schema, Repository, CLI, Testing)
   - Risk mitigation strategies
   - Success criteria

3. **MASTER-IMPLEMENTATION-PLAN.md** - Comprehensive detailed plan
   - Complete 14-section implementation guide
   - Detailed task breakdown with steps and success criteria
   - Quality gates between waves
   - Integration testing approach
   - Risk assessment and mitigation
   - 50+ pages of detailed planning

4. **REQUIREMENT-TRACEABILITY-MATRIX.md** - Requirements mapping
   - Links every requirement to implementation task
   - Maps requirements to test cases
   - Acceptance criteria by scenario
   - Completion checklist

### Task Specifications

5. **tasks/T-E07-F05-001.md** - Database Schema Task
   - Design and implement documents database schema
   - 4 tables + migration function
   - 5-6 hours estimated

6. **tasks/T-E07-F05-002.md** - Repository Task
   - Implement DocumentRepository with 14 methods
   - CRUD, Link/Unlink, List operations
   - 8-10 hours estimated

7. **tasks/T-E07-F05-003.md** - CLI Commands Task
   - Implement shark related-docs command group
   - add, delete, list subcommands
   - 6-8 hours estimated

8. **tasks/T-E07-F05-004.md** - Integration Tests Task
   - Write 30+ integration tests
   - Test all schema and repository functionality
   - 8-10 hours estimated

9. **tasks/T-E07-F05-005.md** - CLI Tests Task
   - Write 25+ CLI command tests
   - Mock repositories (no real DB)
   - 8-10 hours estimated

### Feature Design

10. **../E07-F05-add-related-documents/prd.md** - Product Requirements Document
    - User personas and stories
    - Detailed requirements
    - Out-of-scope items
    - Success metrics

## Quick Navigation

### Starting Implementation?
1. Start with **QUICK-START.md** (10-minute read)
2. Read the relevant task file (T-E07-F05-001 through 005)
3. Reference **MASTER-IMPLEMENTATION-PLAN.md** for detailed steps
4. Use **REQUIREMENT-TRACEABILITY-MATRIX.md** to track progress

### Need a High-Level View?
1. Read **EXECUTIVE-SUMMARY.md** (15-minute read)
2. Check the timeline and effort estimates
3. Review key risks and mitigation strategies

### Validating Requirements?
1. Open **REQUIREMENT-TRACEABILITY-MATRIX.md**
2. Find your requirement (SCH-xxx, REP-xxx, CLI-xxx)
3. See which task implements it and which test validates it

### Looking for Detailed Implementation Steps?
1. Open **MASTER-IMPLEMENTATION-PLAN.md**
2. Find the relevant wave/task
3. Follow the step-by-step implementation guide

## Key Facts

**Feature Goal**: Enable linking supporting documents to epics, features, and tasks

**User Commands**:
```bash
shark related-docs add "OAuth Spec" "docs/oauth.md" --task=T-E01-F01-001
shark related-docs list --epic=E01
shark related-docs delete "OAuth Spec" --epic=E01
```

**Architecture**:
- 4-table database design (documents + 3 link tables)
- DocumentRepository with 14 methods
- CLI command group with 3 subcommands
- 30+ integration tests
- 25+ CLI tests

**Timeline**: ~2 weeks, ~50 hours total
- 5 sequential tasks
- Must complete in order (T-001 → T-002 → T-003 → T-004 → T-005)

**Complexity**: Medium
- Straightforward CRUD operations
- Standard repository pattern
- Standard CLI pattern
- Comprehensive testing required

## Requirements Summary

| Category | Count | Notes |
|----------|-------|-------|
| Schema Requirements | 10 | Tables, constraints, indexes, migration |
| Repository Requirements | 18 | 14 methods + error handling, transactions, queries |
| CLI Requirements | 18 | 3 commands, 6 flags, validation, output formats |
| Testing Requirements | 55+ | 30+ integration tests + 25+ CLI tests |
| **Total** | **49+** | Comprehensive coverage |

## Quality Gates

Between each task, there are quality gates:

1. **After T-001**: Schema verified, migration tested on clean + existing DB
2. **After T-002**: All 14 methods callable, lint passes, follows patterns
3. **After T-003**: Manual testing passes for all 3 commands
4. **After T-004**: 30+ tests passing, >90% coverage, no flaky tests
5. **After T-005**: 25+ tests passing, mocked repos, all commands validated

## Critical Implementation Constraints

These are non-negotiable:

1. **Parameterized queries only** - No SQL string concatenation
2. **Error wrapping** - `fmt.Errorf("context: %w", err)`
3. **Transaction support** - Multi-statement ops with `defer tx.Rollback()`
4. **Idempotent migrations** - Safe to run multiple times
5. **Foreign key constraints** - ON DELETE CASCADE properly tested
6. **Composite primary keys** - (parent_id, document_id) on link tables

## File Structure

**New files to create**:
```
internal/
├── models/document.go
├── repository/document_repository.go
├── repository/document_repository_test.go
└── cli/commands/related_docs.go
└── cli/commands/related_docs_test.go
```

**Files to modify**:
```
internal/
├── db/db.go (add migration function)
└── cli/commands/root.go (register command group)
```

## Testing Strategy

### T-004 Integration Tests (30+)
- Real database, test schema and repository
- CRUD operations (CreateOrGet, GetByID, Delete)
- Link/Unlink operations (6 methods)
- List operations (3 methods)
- Error conditions (FK, UNIQUE, not found)
- Cascade delete behavior
- Target: >90% coverage

### T-005 CLI Tests (25+)
- Mocked repositories (no real database)
- Command parsing and flag validation
- Parent existence validation
- Output formatting (table and JSON)
- Error handling and exit codes
- All tested without database access

## Implementation Order

**Sequential (must complete in order)**:
1. T-001: Schema (5-6 hours)
2. T-002: Repository (8-10 hours)
3. T-003: CLI Commands (6-8 hours)
4. T-004: Integration Tests (8-10 hours) - can scaffold while T-003 is in progress
5. T-005: CLI Tests (8-10 hours) - can scaffold while T-003 is in progress

## Success Criteria

- All 5 tasks completed
- 49+ requirements implemented
- 30+ integration tests passing
- 25+ CLI tests passing
- >90% code coverage
- `make test` passes all tests
- `make build` succeeds
- Manual end-to-end testing passes

## Related Documents

- **Epic PRD**: `/docs/plan/E07-enhancements/epic.md` (if available)
- **Feature PRD**: `/docs/plan/E07-enhancements/E07-F05-add-related-documents/prd.md`
- **Architecture Review**: Check for ARCH-*.md in feature folder

## Contact & Questions

For questions about implementation:
1. Check the relevant task file (T-E07-F05-001 through 005)
2. Review project patterns in internal/repository/ and internal/cli/commands/
3. Check CLAUDE.md for project-specific guidelines
4. Consult MASTER-IMPLEMENTATION-PLAN.md for detailed guidance

---

**Created**: 2025-12-20
**Status**: Ready for Implementation
**Complexity**: Medium (5 tasks, standard patterns, comprehensive testing)
**Estimated Duration**: 2 weeks with focused development

