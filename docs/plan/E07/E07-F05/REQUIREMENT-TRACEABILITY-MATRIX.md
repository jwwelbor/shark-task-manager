# E07-F05 Requirement Traceability Matrix

This document maps every feature requirement to implementation tasks and test coverage.

## Schema Requirements → Tasks

| Req ID | Requirement | Task | Implementation Steps | Tested By | Status |
|--------|-------------|------|----------------------|-----------|--------|
| SCH-001 | documents table: id (PK), title, file_path, created_at | T-001 | 1.3 | T-004 Schema Test | Todo |
| SCH-002 | UNIQUE constraint on (title, file_path) | T-001 | 1.3 | T-004 Duplicate Doc Test | Todo |
| SCH-003 | epic_documents table: epic_id (FK), document_id (FK), PK(epic_id, doc_id) | T-001 | 1.4 | T-004 Link Test | Todo |
| SCH-004 | feature_documents table: feature_id (FK), document_id (FK) | T-001 | 1.5 | T-004 Link Test | Todo |
| SCH-005 | task_documents table: task_id (FK), document_id (FK) | T-001 | 1.6 | T-004 Link Test | Todo |
| SCH-006 | Composite PK on all link tables | T-001 | 1.4-1.6 | T-004 Duplicate Link Test | Todo |
| SCH-007 | ON DELETE CASCADE for all FK | T-001 | 1.7 | T-004 Cascade Test | Todo |
| SCH-008 | Indexes on all FK columns | T-001 | 1.8 | Schema verification | Todo |
| SCH-009 | Auto-migration (IF NOT EXISTS, idempotent) | T-001 | 1.2, 1.9-1.10 | T-001 Testing | Todo |
| SCH-010 | Performance: <100ms for 10k docs | T-001 | 1.8 | Load test (optional) | Todo |

## Repository Requirements → Tasks

| Req ID | Requirement | Task | Implementation | Tested By | Status |
|--------|-------------|------|---|-----------|--------|
| REP-001 | Document model struct (ID, Title, FilePath, CreatedAt) | T-002 | 2.1 | T-004 | Todo |
| REP-002 | Document validation (title, path required) | T-002 | 2.2 | T-004 | Todo |
| REP-003 | CreateOrGet(ctx, title, path) - upsert | T-002 | 2.4 | T-004 Test CreateOrGet | Todo |
| REP-004 | GetByID(ctx, id) | T-002 | 2.5 | T-004 Test GetByID | Todo |
| REP-005 | Delete(ctx, id) | T-002 | 2.6 | T-004 Test Delete | Todo |
| REP-006 | LinkToEpic(ctx, docID, epicID) with transaction | T-002 | 2.7 | T-004 Test LinkToEpic | Todo |
| REP-007 | LinkToFeature(ctx, docID, featureID) with transaction | T-002 | 2.8 | T-004 Test LinkToFeature | Todo |
| REP-008 | LinkToTask(ctx, docID, taskID) with transaction | T-002 | 2.9 | T-004 Test LinkToTask | Todo |
| REP-009 | UnlinkFromEpic(ctx, docID, epicID) | T-002 | 2.10 | T-004 Test UnlinkFromEpic | Todo |
| REP-010 | UnlinkFromFeature(ctx, docID, featureID) | T-002 | 2.11 | T-004 Test UnlinkFromFeature | Todo |
| REP-011 | UnlinkFromTask(ctx, docID, taskID) | T-002 | 2.12 | T-004 Test UnlinkFromTask | Todo |
| REP-012 | ListForEpic(ctx, epicID) | T-002 | 2.13 | T-004 Test ListForEpic | Todo |
| REP-013 | ListForFeature(ctx, featureID) | T-002 | 2.14 | T-004 Test ListForFeature | Todo |
| REP-014 | ListForTask(ctx, taskID) | T-002 | 2.15 | T-004 Test ListForTask | Todo |
| REP-015 | Error wrapping (fmt.Errorf with %w) | T-002 | 2.16 | Code Review | Todo |
| REP-016 | Parameterized queries (no SQL injection) | T-002 | 2.17 | Code Review | Todo |
| REP-017 | Transaction support (defer tx.Rollback) | T-002 | 2.18 | T-004 Transaction Test | Todo |
| REP-018 | Performance: <50ms for typical operations | T-002 | N/A (design) | Load test (optional) | Todo |

## CLI Command Requirements → Tasks

| Req ID | Requirement | Task | Implementation | Tested By | Status |
|--------|-------------|------|---|-----------|--------|
| CLI-001 | related-docs add command with title, path arguments | T-003 | 3.2 | T-005 AddCmd Test | Todo |
| CLI-002 | add: --epic flag | T-003 | 3.2 | T-005 AddCmd Epic Test | Todo |
| CLI-003 | add: --feature flag | T-003 | 3.2 | T-005 AddCmd Feature Test | Todo |
| CLI-004 | add: --task flag | T-003 | 3.2 | T-005 AddCmd Task Test | Todo |
| CLI-005 | add: Mutually exclusive parent flags (exactly 1 required) | T-003 | 3.3 | T-005 Flag Validation Test | Todo |
| CLI-006 | add: Validate parent exists before linking | T-003 | 3.4 | T-005 Parent Validation Test | Todo |
| CLI-007 | add: Success message with document ID | T-003 | 3.5 | T-005 Output Test | Todo |
| CLI-008 | add: Error messages with parent context | T-003 | 3.9 | T-005 Error Message Test | Todo |
| CLI-009 | delete command with title argument | T-003 | 3.8 | T-005 DeleteCmd Test | Todo |
| CLI-010 | delete: Optional parent flags | T-003 | 3.8 | T-005 Optional Flags Test | Todo |
| CLI-011 | delete: Idempotent behavior | T-003 | 3.9 | T-005 Idempotent Test | Todo |
| CLI-012 | delete: Confirmation message | T-003 | 3.10 | T-005 Output Test | Todo |
| CLI-013 | list command with optional parent filters | T-003 | 3.12 | T-005 ListCmd Test | Todo |
| CLI-014 | list: Table format by default | T-003 | 3.13 | T-005 Table Output Test | Todo |
| CLI-015 | list: JSON output with --json flag | T-003 | 3.14 | T-005 JSON Output Test | Todo |
| CLI-016 | All commands support --verbose flag | T-003 | 3.19 | T-005 Verbose Output Test | Todo |
| CLI-017 | Commands follow Cobra patterns | T-003 | 3.1, 3.17 | Code Review | Todo |
| CLI-018 | Exit codes (0=success, 1=not found, 2=error) | T-003 | 3.19 | T-005 Exit Code Test | Todo |

## Feature-Level User Stories → Implementation

| User Story | Requirement | Task | How Implemented |
|-----------|-------------|------|-----------------|
| Story 1: Link docs to tasks | REP-001...008, CLI-009...012 | T-002, T-003 | DocumentRepository + add command |
| Story 2: Link docs to features/epics | REP-001...008, CLI-009...012 | T-002, T-003 | LinkToFeature, LinkToEpic methods |
| Story 3: CLI commands for management | CLI-001...018 | T-003 | related-docs add/delete/list commands |
| Story 4: View docs in get commands | OUT OF SCOPE | Future | Integration with task/feature/epic get |
| Story 5: Path validation | NOT REQUIRED | Future | Optional file existence check |

## Testing Coverage Matrix

### T-004 Integration Test Cases (30+)

| Category | Test Cases | Coverage |
|----------|-----------|----------|
| CreateOrGet | 3 (new doc, existing doc, invalid inputs) | REP-003 |
| GetByID | 2 (valid, invalid) | REP-004 |
| Delete | 3 (valid, cascade, non-existent) | REP-005, SCH-007 |
| LinkToEpic | 3 (valid, invalid epic, duplicate) | REP-006, SCH-003, SCH-006 |
| LinkToFeature | 3 (valid, invalid feature, duplicate) | REP-007, SCH-004, SCH-006 |
| LinkToTask | 3 (valid, invalid task, duplicate) | REP-008, SCH-005, SCH-006 |
| UnlinkFromEpic | 2 (existing, non-existent) | REP-009 |
| UnlinkFromFeature | 2 (existing, non-existent) | REP-010 |
| UnlinkFromTask | 2 (existing, non-existent) | REP-011 |
| ListForEpic | 2 (with docs, without docs) | REP-012 |
| ListForFeature | 2 (with docs, without docs) | REP-013 |
| ListForTask | 2 (with docs, without docs) | REP-014 |
| **Total** | **30+** | **All Repository Methods** |

### T-005 CLI Test Cases (25+)

| Category | Test Cases | Coverage |
|----------|-----------|----------|
| Add Command | 7 (epic, feature, task, no flag, multiple flags, invalid parent, JSON) | CLI-001...008 |
| Delete Command | 5 (delete, idempotent, optional flags, JSON, confirmation) | CLI-009...012 |
| List Command | 8 (all docs, filter epic, filter feature, filter task, empty, table, JSON, sorting) | CLI-013...015 |
| Flag Validation | 3 (mutually exclusive, verbose, help) | CLI-016, CLI-017 |
| Error Handling | 2 (error messages, exit codes) | CLI-018 |
| **Total** | **25+** | **All CLI Commands** |

## Requirements by Priority

### P0 (Must-Have)
- [ ] Database schema (T-001)
- [ ] Repository CRUD operations (T-002)
- [ ] Repository Link/Unlink operations (T-002)
- [ ] CLI add command (T-003)
- [ ] CLI delete command (T-003)
- [ ] CLI list command (T-003)
- [ ] Integration tests >90% coverage (T-004)
- [ ] CLI command tests (T-005)

### P1 (Should-Have)
- [ ] Performance optimization (<100ms, <50ms)
- [ ] Cascade delete behavior (tested manually)
- [ ] Idempotent migrations
- [ ] Comprehensive error messages

### P2 (Could-Have)
- [ ] Document path validation
- [ ] External URL support
- [ ] Document version tracking

### Out of Scope
- [ ] Full-text search
- [ ] Document generation
- [ ] Integration with task/feature/epic views

## Acceptance Criteria by Scenario

### Scenario 1: Add Document to Task

**Given**:
- Task T-E01-F01-001 exists

**When**:
```bash
shark related-docs add "API Spec" "docs/api/spec.md" --task=T-E01-F01-001
```

**Then**:
- [ ] Document created in documents table (REP-003, SCH-001, SCH-002)
- [ ] Link created in task_documents table (REP-008, SCH-005)
- [ ] Success message shows document ID (CLI-007)
- [ ] Exit code 0 (CLI-018)

### Scenario 2: List Documents for Feature

**Given**:
- Feature E01-F01 has 3 documents linked

**When**:
```bash
shark related-docs list --feature=E01-F01
```

**Then**:
- [ ] Table shows all 3 documents (CLI-014)
- [ ] Columns: Title, Path, Created At
- [ ] Sorted by creation date
- [ ] Exit code 0 (CLI-018)

### Scenario 3: Delete Document Link

**Given**:
- Epic E01 has document "Architecture" linked

**When**:
```bash
shark related-docs delete "Architecture" --epic=E01
```

**Then**:
- [ ] Link removed from epic_documents (REP-009)
- [ ] Document remains in documents table (not deleted)
- [ ] Confirmation message displayed (CLI-012)
- [ ] Exit code 0 (CLI-018)

### Scenario 4: Foreign Key Constraint

**Given**:
- Non-existent epic E99

**When**:
```bash
shark related-docs add "Doc" "path" --epic=E99
```

**Then**:
- [ ] Error message: "Epic E99 not found" (CLI-006, CLI-008)
- [ ] Link not created
- [ ] Exit code 1 (CLI-018)
- [ ] Database unchanged

### Scenario 5: Idempotent Delete

**Given**:
- Document not linked to epic

**When**:
```bash
shark related-docs delete "NonExistent" --epic=E01
shark related-docs delete "NonExistent" --epic=E01  # Run twice
```

**Then**:
- [ ] Both commands succeed (REP-009 idempotent)
- [ ] Same exit code 0 both times (CLI-011, CLI-018)
- [ ] No error message

## Requirement Completion Checklist

Use this checklist to track implementation progress:

### T-001 Schema
- [ ] SCH-001: documents table
- [ ] SCH-002: UNIQUE constraint
- [ ] SCH-003: epic_documents table
- [ ] SCH-004: feature_documents table
- [ ] SCH-005: task_documents table
- [ ] SCH-006: Composite PKs
- [ ] SCH-007: CASCADE delete
- [ ] SCH-008: Indexes
- [ ] SCH-009: Idempotent migration
- [ ] SCH-010: Performance (optional)
- **Quality Gate**: Schema verified, migration tested

### T-002 Repository
- [ ] REP-001: Document model
- [ ] REP-002: Validation
- [ ] REP-003: CreateOrGet
- [ ] REP-004: GetByID
- [ ] REP-005: Delete
- [ ] REP-006: LinkToEpic
- [ ] REP-007: LinkToFeature
- [ ] REP-008: LinkToTask
- [ ] REP-009: UnlinkFromEpic
- [ ] REP-010: UnlinkFromFeature
- [ ] REP-011: UnlinkFromTask
- [ ] REP-012: ListForEpic
- [ ] REP-013: ListForFeature
- [ ] REP-014: ListForTask
- [ ] REP-015: Error wrapping
- [ ] REP-016: Parameterized queries
- [ ] REP-017: Transactions
- [ ] REP-018: Performance (optional)
- **Quality Gate**: All methods callable, lint passes

### T-003 CLI Commands
- [ ] CLI-001: add command
- [ ] CLI-002: --epic flag
- [ ] CLI-003: --feature flag
- [ ] CLI-004: --task flag
- [ ] CLI-005: Mutually exclusive flags
- [ ] CLI-006: Parent validation
- [ ] CLI-007: Success message
- [ ] CLI-008: Error messages
- [ ] CLI-009: delete command
- [ ] CLI-010: Optional flags
- [ ] CLI-011: Idempotent delete
- [ ] CLI-012: Confirmation
- [ ] CLI-013: list command
- [ ] CLI-014: Table format
- [ ] CLI-015: JSON output
- [ ] CLI-016: --verbose support
- [ ] CLI-017: Cobra patterns
- [ ] CLI-018: Exit codes
- **Quality Gate**: Manual testing passes

### T-004 Integration Tests
- [ ] 30+ test cases
- [ ] All CRUD tests
- [ ] All Link/Unlink tests
- [ ] All List tests
- [ ] Error condition tests
- [ ] Cascade delete test
- [ ] >90% coverage
- [ ] All tests passing
- **Quality Gate**: Coverage report, all tests green

### T-005 CLI Tests
- [ ] 25+ test cases
- [ ] Add command tests
- [ ] Delete command tests
- [ ] List command tests
- [ ] Flag validation tests
- [ ] Error handling tests
- [ ] JSON output validation
- [ ] Mock repositories (no DB)
- [ ] All tests passing
- **Quality Gate**: All tests passing, mocked repos

---

**Last Updated**: 2025-12-20
**Status**: Ready for Implementation
**Total Requirements**: 49 (18 Schema + 18 Repository + 18 CLI + 2 Testing)

