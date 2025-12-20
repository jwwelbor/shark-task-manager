# E07-F05 Implementation Plan - Executive Summary

## Quick Overview

**Feature Goal**: Add capability to link supporting documents (architecture diagrams, design docs, QA reports) to epics, features, and tasks.

**Scope**: 5 sequential tasks implementing a documents database system with CLI commands
- **T-001**: Database schema (4 tables + migration)
- **T-002**: DocumentRepository (14 methods)
- **T-003**: CLI commands (add, delete, list)
- **T-004**: Integration tests (30+ tests)
- **T-005**: CLI tests (25+ tests)

**Total Effort**: ~40-50 hours
**Risk Level**: Low-Medium (no complex algorithms, straightforward CRUD + CLI)

---

## Key Requirements Summary

### Schema (T-001)
4 tables with proper constraints:
- `documents`: (id, title, file_path, created_at, UNIQUE(title, file_path))
- `epic_documents`: link table with composite PK + CASCADE delete
- `feature_documents`: link table with composite PK + CASCADE delete
- `task_documents`: link table with composite PK + CASCADE delete
- 5 indexes on FK columns for performance

### Repository (T-002)
14 methods in DocumentRepository:
- **CRUD**: CreateOrGet (upsert), GetByID, Delete
- **Link**: LinkToEpic, LinkToFeature, LinkToTask (3 methods)
- **Unlink**: UnlinkFromEpic, UnlinkFromFeature, UnlinkFromTask (3 methods)
- **List**: ListForEpic, ListForFeature, ListForTask (3 methods)
- **Standards**: Error wrapping, parameterized queries, transactions

### CLI Commands (T-003)
Command group `shark related-docs` with 3 subcommands:
```bash
shark related-docs add <title> <path> --epic=<key>|--feature=<key>|--task=<key>
shark related-docs delete <title> [--epic|--feature|--task=<key>]
shark related-docs list [--epic|--feature|--task=<key>] [--json]
```

Key behaviors:
- Mutually exclusive parent flags (exactly one required for add)
- Parent existence validation before linking
- Idempotent delete (succeed even if already deleted)
- JSON output support for all commands

### Testing (T-004 & T-005)
- **30+ integration tests**: Schema validation, CRUD, FK constraints, cascade delete
- **25+ CLI tests**: Command parsing, flag validation, output formatting (mocked repos)
- Target coverage: >90% for repository code
- All tests isolated and deterministic

---

## Critical Implementation Constraints

### Must-Have (No Shortcuts)
1. **Parameterized queries only** - No string concatenation in SQL
2. **Proper error wrapping** - All DB errors: `fmt.Errorf("context: %w", err)`
3. **Transaction support** - Multi-statement operations wrapped with `defer tx.Rollback()`
4. **Idempotent migrations** - Safe to run multiple times, use `IF NOT EXISTS`
5. **Foreign key constraints** - ON DELETE CASCADE must be tested
6. **Composite primary keys** - (parent_id, document_id) on all link tables

### Quality Gates Between Tasks
- **After T-001**: Schema validation + manual constraint testing
- **After T-002**: All methods callable, lint passes, follows project patterns
- **After T-003**: Manual functional testing of all 3 commands
- **After T-004**: 30+ tests passing, >90% coverage, no flaky tests
- **After T-005**: 25+ tests passing, mocked (no DB access), all commands validated

---

## Task Sequence & Dependencies

### Sequential (Must Complete in Order)
```
T-001 Schema (5-6 hrs)
  ↓ requires schema to exist
T-002 Repository (8-10 hrs)
  ↓ requires repo to implement
T-003 CLI Commands (6-8 hrs)
  ↓ requires commands to test
T-004 Integration Tests (8-10 hrs) ← Can start scaffolding during T-003
  ↓ requires repo/CLI to test
T-005 CLI Tests (8-10 hrs) ← Can start scaffolding during T-003
  ↓
Feature Complete
```

### Parallel Opportunities
- T-004 test structure can be scaffolded while T-003 CLI is being developed
- T-005 test structure can be scaffolded while T-003 CLI is being developed
- Both test files can reference implementations in progress

### Start Conditions
- **T-001**: Can start immediately (no dependencies)
- **T-002**: After T-001 schema created and tested
- **T-003**: After T-002 repository working
- **T-004**: Anytime after T-002, use test fixtures to mock data
- **T-005**: Anytime after T-003, use mock repositories (no DB required)

---

## Risk Mitigation Matrix

| Risk | Mitigation |
|------|-----------|
| Migration idempotency issues | Test on clean + existing DB before release; use IF NOT EXISTS everywhere |
| Foreign key cascade affects wrong data | Test cascade manually; verify unrelated data unaffected |
| SQL injection in dynamic queries | Parameterized queries only; code review all SQL |
| Performance with large datasets | Indexes on all FK columns; test with 10k documents |
| Mock repos don't match real behavior | Keep mocks in sync with real repo; use same interface |
| T-002 takes longer | Start with core CRUD, add Link/Unlink methods after |
| Testing reveals design flaws | Get schema review before starting T-002 |

---

## Integration Testing Strategy

After all 5 tasks complete, validate with end-to-end scenarios:

### Scenario 1: Add-List-Delete Workflow
```bash
shark epic create "Foundation"  # Create E01
shark related-docs add "OAuth Spec" "docs/oauth.md" --epic=E01
shark related-docs list --epic=E01  # Shows OAuth Spec
shark related-docs delete "OAuth Spec" --epic=E01
shark related-docs list --epic=E01  # Empty
```

### Scenario 2: Document Reuse
```bash
shark related-docs add "Architecture" "docs/arch.md" --epic=E01
shark related-docs add "Architecture" "docs/arch.md" --feature=E01-F01
shark related-docs add "Architecture" "docs/arch.md" --task=T-E01-F01-001
# All show same document ID (reused)
shark related-docs delete "Architecture" --task=T-E01-F01-001
# Still linked to epic and feature
```

### Scenario 3: Error Handling
```bash
shark related-docs add "Doc" "path" --epic=E99  # Error: E99 not found
shark related-docs add "Doc" "path"  # Error: exactly one parent flag required
shark related-docs add "Doc" "path" --epic=E01 --feature=E01-F01  # Error: mutually exclusive
shark related-docs delete "NonExistent" --epic=E01  # Success (idempotent)
```

### Scenario 4: JSON Output
```bash
shark related-docs list --json | jq .  # Valid JSON
shark related-docs add "Doc" "path" --epic=E01 --json | jq .  # Valid JSON
shark related-docs delete "Doc" --epic=E01 --json | jq .  # Valid JSON
```

---

## Code Organization

### New Files to Create
```
internal/
├── models/document.go                        (T-002)
├── repository/document_repository.go         (T-002)
├── repository/document_repository_test.go    (T-004)
└── cli/commands/related_docs.go              (T-003)
```

### Existing Files to Modify
```
internal/
├── db/db.go                                  (T-001: Add migration function)
├── cli/commands/related_docs_test.go         (T-005)
└── commands/root.go or similar               (T-003: Register new command group)
```

---

## Estimated Timeline

| Task | Hours | Duration | Start After |
|------|-------|----------|------------|
| T-001 (Schema) | 5-6 | 1-2 days | Immediate |
| T-002 (Repository) | 8-10 | 2-3 days | T-001 complete |
| T-003 (CLI) | 6-8 | 2 days | T-002 complete |
| T-004 (Integration Tests) | 8-10 | 2-3 days | T-002 complete (parallel with T-003) |
| T-005 (CLI Tests) | 8-10 | 2-3 days | T-003 complete |
| **Total** | **40-50** | **~2 weeks** | |

---

## Success Criteria

### Feature-Level
- All 5 tasks completed with no blockers
- All CRUD operations working (create, read, delete)
- All link/unlink operations working
- All list operations with filtering working
- All 3 CLI commands working
- 30+ integration tests passing with >90% coverage
- 25+ CLI tests passing
- Manual end-to-end testing successful
- `make test` passes all tests
- `make build` succeeds without errors

### Per-Task
- **T-001**: Schema verified with sqlite3, migration tested clean + existing DB
- **T-002**: 14 methods callable, no lint errors, proper error wrapping
- **T-003**: All commands work, flag validation enforced, parent validation working
- **T-004**: 30+ tests, >90% coverage, isolated and deterministic
- **T-005**: 25+ tests, mocked repos, all commands validated

---

## Key Technical Decisions

### Why 4 Tables (Not 1)?
Separate link tables (epic_documents, feature_documents, task_documents) instead of single table because:
- Type safety with foreign keys
- Better performance with targeted queries
- Clearer schema semantics
- Easier to enforce constraints

### Why Composite Primary Keys?
(parent_id, document_id) composite PKs on link tables:
- Prevents duplicate links naturally
- No need for separate unique constraint
- Better query performance
- Matches project patterns

### Why CreateOrGet (Not Create + Get)?
Upsert pattern (CreateOrGet) for documents:
- Avoids duplicate documents with same title+path
- Enables document reuse across parents
- Cleaner API (one call, not two)
- Idempotent behavior

### Why Separate CLI Tests with Mocks?
(T-005) not full integration tests:
- No database required for CLI testing
- Faster test execution
- Tests command logic independently
- Follows project pattern (mock repositories)

---

## Handoff Checklist

When starting implementation:
- [ ] Read MASTER-IMPLEMENTATION-PLAN.md (this file is summary)
- [ ] Read all 5 task files (T-E07-F05-001 through 005)
- [ ] Read feature PRD
- [ ] Review existing repo patterns (internal/repository/)
- [ ] Review existing CLI patterns (internal/cli/commands/)
- [ ] Understand project error handling (fmt.Errorf with %w)
- [ ] Understand migration pattern (internal/db/db.go)
- [ ] Set up development workspace
- [ ] Start with T-001 (cannot skip)
- [ ] Complete quality gates between waves
- [ ] Perform end-to-end validation after all tasks

---

## Document References

- **Full Plan**: `/docs/plan/E07/E07-F05/MASTER-IMPLEMENTATION-PLAN.md`
- **Task 1**: `/docs/plan/E07/E07-F05/tasks/T-E07-F05-001.md`
- **Task 2**: `/docs/plan/E07/E07-F05/tasks/T-E07-F05-002.md`
- **Task 3**: `/docs/plan/E07/E07-F05/tasks/T-E07-F05-003.md`
- **Task 4**: `/docs/plan/E07/E07-F05/tasks/T-E07-F05-004.md`
- **Task 5**: `/docs/plan/E07/E07-F05/tasks/T-E07-F05-005.md`
- **Feature PRD**: `/docs/plan/E07-enhancements/E07-F05-add-related-documents/prd.md`

