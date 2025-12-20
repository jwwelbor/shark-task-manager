# E07-F05 Master Implementation Plan
## Add Related Documents Feature

**Plan Created**: 2025-12-20
**Feature Key**: E07-F05
**Epic**: E07 (Enhancements)
**Complexity**: Medium
**Estimated Effort**: 5 tasks, ~40-50 hours for complete implementation

---

## 1. FEATURE SUMMARY

### High-Level Goal
Enable users to link supporting documents (architecture diagrams, design docs, QA reports, API specifications) to epics, features, and tasks, creating structured relationships that make documentation discoverable and context-aware within the task management system.

### Problem Statement
Important supporting documents exist outside the task management system with no formal links to related epics, features, or tasks. This creates information silos where developers must hunt for relevant documents, and there's no structured way to track document-work relationships.

### Solution Overview
Implement a 4-table data model:
- **documents**: Centralized store for document metadata (id, title, file_path, created_at)
- **epic_documents**: Links documents to epics (many-to-many)
- **feature_documents**: Links documents to features (many-to-many)
- **task_documents**: Links documents to tasks (many-to-many)

Provide CLI command group `shark related-docs` with add/delete/list subcommands for managing associations.

### User Impact
- **Developers**: Quick access to relevant documentation for current task/feature
- **Product Managers**: Ability to link architecture and design docs to features/epics
- **Teams**: Structured, searchable relationships between work items and documentation

### Scope Boundaries
**In Scope**:
- Document metadata tracking (title, path)
- Links to epics, features, and tasks
- CLI commands for management
- Basic validation and error handling

**Out of Scope** (Future):
- Document content indexing or full-text search
- Document version tracking (git handles this)
- Automatic document generation
- Integration with task/feature/epic detail views

---

## 2. COMPREHENSIVE REQUIREMENT MATRIX

### Master Requirements Index

This matrix shows which task(s) implement each requirement:

| Req ID | Requirement | Category | Task(s) | Priority | Status |
|--------|-------------|----------|---------|----------|--------|
| **SCHEMA LAYER** | | | | | |
| SCH-001 | documents table (id, title, file_path, created_at) | Schema | T-001 | P0 | Todo |
| SCH-002 | UNIQUE constraint on (title, file_path) | Schema | T-001 | P0 | Todo |
| SCH-003 | epic_documents link table (epic_id, doc_id) | Schema | T-001 | P0 | Todo |
| SCH-004 | feature_documents link table (feature_id, doc_id) | Schema | T-001 | P0 | Todo |
| SCH-005 | task_documents link table (task_id, doc_id) | Schema | T-001 | P0 | Todo |
| SCH-006 | Composite PK on all link tables (parent_id, doc_id) | Schema | T-001 | P0 | Todo |
| SCH-007 | ON DELETE CASCADE for all foreign keys | Schema | T-001 | P0 | Todo |
| SCH-008 | Indexes on all foreign key columns | Schema | T-001 | P0 | Todo |
| SCH-009 | Auto-migration support (backward compatible) | Schema | T-001 | P0 | Todo |
| SCH-010 | Performance: <100ms queries for 10k docs | Schema | T-001 | P1 | Todo |
| **REPOSITORY LAYER** | | | | | |
| REP-001 | Document model struct (ID, Title, FilePath, CreatedAt) | Repository | T-002 | P0 | Todo |
| REP-002 | Document validation (title and path required) | Repository | T-002 | P0 | Todo |
| REP-003 | CreateOrGet(ctx, title, path) - upsert logic | Repository | T-002 | P0 | Todo |
| REP-004 | GetByID(ctx, id) - fetch by ID | Repository | T-002 | P0 | Todo |
| REP-005 | Delete(ctx, id) - delete with cascade | Repository | T-002 | P0 | Todo |
| REP-006 | LinkToEpic(ctx, docID, epicID) - transaction support | Repository | T-002 | P0 | Todo |
| REP-007 | LinkToFeature(ctx, docID, featureID) - transaction support | Repository | T-002 | P0 | Todo |
| REP-008 | LinkToTask(ctx, docID, taskID) - transaction support | Repository | T-002 | P0 | Todo |
| REP-009 | UnlinkFromEpic(ctx, docID, epicID) | Repository | T-002 | P0 | Todo |
| REP-010 | UnlinkFromFeature(ctx, docID, featureID) | Repository | T-002 | P0 | Todo |
| REP-011 | UnlinkFromTask(ctx, docID, taskID) | Repository | T-002 | P0 | Todo |
| REP-012 | ListForEpic(ctx, epicID) - fetch all linked docs | Repository | T-002 | P0 | Todo |
| REP-013 | ListForFeature(ctx, featureID) - fetch all linked docs | Repository | T-002 | P0 | Todo |
| REP-014 | ListForTask(ctx, taskID) - fetch all linked docs | Repository | T-002 | P0 | Todo |
| REP-015 | Proper error wrapping (fmt.Errorf with %w) | Repository | T-002 | P0 | Todo |
| REP-016 | Parameterized queries (prevent SQL injection) | Repository | T-002 | P0 | Todo |
| REP-017 | Transaction support for multi-statement operations | Repository | T-002 | P0 | Todo |
| REP-018 | Performance: <50ms for typical operations | Repository | T-002 | P1 | Todo |
| **CLI COMMANDS** | | | | | |
| CLI-001 | related-docs add command with title and path args | Commands | T-003 | P0 | Todo |
| CLI-002 | add: --epic flag for epic linking | Commands | T-003 | P0 | Todo |
| CLI-003 | add: --feature flag for feature linking | Commands | T-003 | P0 | Todo |
| CLI-004 | add: --task flag for task linking | Commands | T-003 | P0 | Todo |
| CLI-005 | add: Mutually exclusive parent flags (exactly 1 required) | Commands | T-003 | P0 | Todo |
| CLI-006 | add: Validate parent exists before linking | Commands | T-003 | P0 | Todo |
| CLI-007 | add: Success message with document ID | Commands | T-003 | P0 | Todo |
| CLI-008 | add: Error messages include parent context | Commands | T-003 | P0 | Todo |
| CLI-009 | delete command with title argument | Commands | T-003 | P0 | Todo |
| CLI-010 | delete: Optional parent flags | Commands | T-003 | P0 | Todo |
| CLI-011 | delete: Idempotent behavior (succeed even if not found) | Commands | T-003 | P0 | Todo |
| CLI-012 | delete: Confirmation message | Commands | T-003 | P0 | Todo |
| CLI-013 | list command with optional parent filters | Commands | T-003 | P0 | Todo |
| CLI-014 | list: Shows table format by default | Commands | T-003 | P0 | Todo |
| CLI-015 | list: JSON output with --json flag | Commands | T-003 | P0 | Todo |
| CLI-016 | All commands support --verbose flag | Commands | T-003 | P0 | Todo |
| CLI-017 | All commands follow Cobra patterns | Commands | T-003 | P0 | Todo |
| CLI-018 | Proper exit codes (0=success, 1=not found, 2=DB error) | Commands | T-003 | P0 | Todo |
| **TESTING** | | | | | |
| TEST-001 | Integration tests in document_repository_test.go | Testing | T-004 | P0 | Todo |
| TEST-002 | Test CreateOrGet creates new document | Testing | T-004 | P0 | Todo |
| TEST-003 | Test CreateOrGet returns existing for duplicates | Testing | T-004 | P0 | Todo |
| TEST-004 | Test GetByID with valid/invalid IDs | Testing | T-004 | P0 | Todo |
| TEST-005 | Test Delete with cascade to links | Testing | T-004 | P0 | Todo |
| TEST-006 | Test LinkToEpic creates valid link | Testing | T-004 | P0 | Todo |
| TEST-007 | Test LinkToEpic fails with invalid epic | Testing | T-004 | P0 | Todo |
| TEST-008 | Test LinkToFeature and LinkToTask similarly | Testing | T-004 | P0 | Todo |
| TEST-009 | Test duplicate links fail with UNIQUE constraint | Testing | T-004 | P0 | Todo |
| TEST-010 | Test all UnlinkFrom* operations | Testing | T-004 | P0 | Todo |
| TEST-011 | Test all ListFor* operations | Testing | T-004 | P0 | Todo |
| TEST-012 | Test empty results from List operations | Testing | T-004 | P0 | Todo |
| TEST-013 | 30+ integration test cases | Testing | T-004 | P0 | Todo |
| TEST-014 | >90% code coverage for repository | Testing | T-004 | P0 | Todo |
| TEST-015 | CLI command tests (25+ cases) | Testing | T-005 | P0 | Todo |
| TEST-016 | Test add command with all parent types | Testing | T-005 | P0 | Todo |
| TEST-017 | Test add validation (mutually exclusive flags) | Testing | T-005 | P0 | Todo |
| TEST-018 | Test add with invalid/non-existent parent | Testing | T-005 | P0 | Todo |
| TEST-019 | Test delete command idempotency | Testing | T-005 | P0 | Todo |
| TEST-020 | Test list command filtering | Testing | T-005 | P0 | Todo |
| TEST-021 | Test JSON output validity | Testing | T-005 | P0 | Todo |
| TEST-022 | Test error message clarity | Testing | T-005 | P0 | Todo |
| TEST-023 | Mock repositories (no real DB in CLI tests) | Testing | T-005 | P0 | Todo |

### Key Constraints & Requirements Summary

**Data Integrity**
- Foreign key constraints enforce referential integrity
- ON DELETE CASCADE prevents orphaned links
- UNIQUE constraints prevent duplicate documents
- Composite PKs on junction tables

**Performance**
- Schema queries: <100ms for 10k documents, 100k links
- Repository operations: <50ms typical
- Indexes on all foreign key columns

**Compatibility**
- Auto-migration system (backward compatible)
- Safe to run on existing databases
- Idempotent migration functions

**API/CLI Standards**
- Follows existing Cobra patterns
- Proper error wrapping (fmt.Errorf with %w)
- Context support for cancellation/timeouts
- Parameterized queries (no SQL injection)

---

## 3. TASK DEPENDENCY GRAPH

### Dependency Relationships

```
T-E07-F05-001 (Schema)
    ↓ (must complete first)
T-E07-F05-002 (Repository)
    ↓ (depends on repo implementation)
T-E07-F05-003 (CLI Commands) ← Also depends on Epic/Feature/Task repos
    ↓
T-E07-F05-004 (Integration Tests) ← Tests schema + repo
    ↓
T-E07-F05-005 (CLI Tests) ← Tests commands
    ↓
COMPLETE & INTEGRATED
```

### Sequential vs Parallel Opportunities

**Must Be Sequential**:
1. T-001 (Schema) - Foundation layer
2. T-002 (Repository) - Requires schema to exist
3. T-003 (CLI Commands) - Requires repository
4. T-004 (Repo Tests) - Must follow repo implementation
5. T-005 (CLI Tests) - Must follow CLI implementation

**Can Begin in Parallel** (after blocking task completes):
- T-004 and T-005 can be developed in parallel during T-003 development
- Test file structure can be scaffolded while implementation is in progress

**Testing Dependencies**:
- T-004 integration tests validate T-001 schema
- T-004 integration tests validate T-002 repository
- T-005 CLI tests validate T-003 commands
- T-005 tests mock all dependencies (no DB required)

---

## 4. DETAILED IMPLEMENTATION SEQUENCE

### WAVE 1: Database Schema Foundation (T-E07-F05-001)

**Task**: Design and implement documents database schema
**Priority**: P0
**Estimated Effort**: 4-6 hours
**Goal**: Create SQLite schema with all tables, constraints, indexes

#### Steps & Success Criteria

| Step | Description | Success Criteria | Validation |
|------|-------------|------------------|-----------|
| 1.1 | Analyze existing migration pattern in `internal/db/db.go` | Understand auto-migration system, idempotency requirements | Review migration functions in codebase |
| 1.2 | Create migration function `migrateDocumentTables()` | Function exists and is called from schema init | Function in db.go with conditional checks |
| 1.3 | Implement `documents` table | id (PK), title (NOT NULL), file_path (NOT NULL), created_at (DEFAULT NOW), UNIQUE(title, file_path) | `sqlite3 shark-tasks.db ".schema documents"` shows all columns |
| 1.4 | Implement `epic_documents` table | epic_id (FK), document_id (FK), created_at (DEFAULT NOW), PK(epic_id, document_id) | Schema check + FK constraint test |
| 1.5 | Implement `feature_documents` table | feature_id (FK), document_id (FK), created_at (DEFAULT NOW), PK(feature_id, document_id) | Schema check + FK constraint test |
| 1.6 | Implement `task_documents` table | task_id (FK), document_id (FK), created_at (DEFAULT NOW), PK(task_id, document_id) | Schema check + FK constraint test |
| 1.7 | Add ON DELETE CASCADE to all foreign keys | Deleting epic/feature/task removes related links | Manual test: delete epic, verify links removed |
| 1.8 | Create indexes for performance | idx_documents_title, idx_documents_file_path, idx_epic_documents_epic_id, idx_feature_documents_feature_id, idx_task_documents_task_id | `sqlite3 shark-tasks.db ".indices"` shows all indexes |
| 1.9 | Test on clean database | Fresh DB has all tables and constraints | Run `shark init --non-interactive` and verify schema |
| 1.10 | Test on existing database with data | Migration runs without errors, preserves existing data | Backup DB, run migration, verify data intact |

#### Quality Gates
- [ ] Migration function is idempotent (safe to run multiple times)
- [ ] All CREATE TABLE and CREATE INDEX use IF NOT EXISTS
- [ ] Foreign key constraints are enforced (PRAGMA foreign_keys = ON)
- [ ] ON DELETE CASCADE tested manually
- [ ] `make test` passes (no new test failures)
- [ ] Schema matches architecture review specification

#### Deliverables
- `internal/db/db.go` with `migrateDocumentTables()` function
- All four tables created with correct structure
- All indexes created for performance
- Schema verified via `sqlite3 shark-tasks.db .schema`

---

### WAVE 2: Repository Implementation (T-E07-F05-002)

**Task**: Implement DocumentRepository with CRUD and link operations
**Priority**: P0
**Estimated Effort**: 8-10 hours
**Goal**: Create complete repository with 14 methods for document management

#### Steps & Success Criteria

| Step | Description | Success Criteria | Validation |
|------|-------------|------------------|-----------|
| 2.1 | Create `internal/models/document.go` | Document struct with ID, Title, FilePath, CreatedAt fields, JSON/DB tags | Struct compiles, has validation method |
| 2.2 | Implement Document validation | Title and FilePath required and non-empty | Validation function called from constructor |
| 2.3 | Create `internal/repository/document_repository.go` | Constructor `NewDocumentRepository(db *DB)`, struct with db field | File exists, imports correct packages |
| 2.4 | Implement CreateOrGet method | Upsert logic - returns existing if title+path match, otherwise creates | Test: insert twice with same values, get same ID back |
| 2.5 | Implement GetByID method | SELECT by id, proper error handling | Test: valid ID returns document, invalid ID returns error |
| 2.6 | Implement Delete method | DELETE from documents, cascade handled by DB schema | Test: verify deleted, links removed |
| 2.7 | Implement LinkToEpic method | INSERT into epic_documents with FK check, transaction support | Test: link created, invalid epic fails with FK error |
| 2.8 | Implement LinkToFeature method | INSERT into feature_documents with FK check, transaction | Test: link created, constraints enforced |
| 2.9 | Implement LinkToTask method | INSERT into task_documents with FK check, transaction | Test: link created, constraints enforced |
| 2.10 | Implement UnlinkFromEpic method | DELETE from epic_documents for specific link | Test: link removed, others unaffected |
| 2.11 | Implement UnlinkFromFeature method | DELETE from feature_documents for specific link | Test: link removed, others unaffected |
| 2.12 | Implement UnlinkFromTask method | DELETE from task_documents for specific link | Test: link removed, others unaffected |
| 2.13 | Implement ListForEpic method | SELECT documents joined with epic_documents, error if parent not found | Test: returns correct documents, empty on no links |
| 2.14 | Implement ListForFeature method | SELECT documents joined with feature_documents | Test: returns correct documents |
| 2.15 | Implement ListForTask method | SELECT documents joined with task_documents | Test: returns correct documents |
| 2.16 | Add error wrapping to all methods | Every db error wrapped with fmt.Errorf("context: %w", err) | Code review: all errors wrapped |
| 2.17 | Use prepared statements/parameterized queries | No string concatenation for SQL | Code review: grep for string concat |
| 2.18 | Implement transaction support | Multi-statement ops wrapped in tx, defer rollback pattern | Code review: LinkTo* methods use transactions |

#### Quality Gates
- [ ] All 14 methods implemented and callable
- [ ] CreateOrGet idempotency verified (same inputs = same output)
- [ ] Foreign key constraints enforced (test with invalid ID)
- [ ] No SQL injection vulnerabilities (parameterized queries only)
- [ ] All errors wrapped with context
- [ ] Code compiles and passes lint check
- [ ] Methods follow project error handling patterns

#### Deliverables
- `internal/models/document.go` with Document struct and validation
- `internal/repository/document_repository.go` with 14 methods
- All methods properly error wrapped
- All CRUD and link operations working

#### Testing Plan (Deferred to T-004)
- Unit tests validate all 14 methods
- Integration tests with real database
- >90% code coverage for repository

---

### WAVE 3: CLI Commands Implementation (T-E07-F05-003)

**Task**: Implement related-docs CLI commands (add, delete, list)
**Priority**: P0
**Estimated Effort**: 6-8 hours
**Goal**: Create CLI command group with 3 subcommands for document management

#### Steps & Success Criteria

| Step | Description | Success Criteria | Validation |
|------|-------------|------------------|-----------|
| 3.1 | Create `internal/cli/commands/related_docs.go` | File exists with command structure | File compiles, imports correct packages |
| 3.2 | Implement relatedDocsAddCmd | Arguments: title (positional), path (positional), flags: --epic, --feature, --task | `shark related-docs add "Test" "path" --epic=E01` works |
| 3.3 | Implement add flag validation | Ensure exactly one parent flag (--epic, --feature, --task) | Test: no flags = error, multiple flags = error |
| 3.4 | Implement add parent validation | Check parent exists in DB before linking | Test: non-existent epic returns "Epic not found" error |
| 3.5 | Implement add logic | Call DocumentRepo.CreateOrGet(), then LinkTo*() | Test: document created and linked |
| 3.6 | Implement add success output | Display document ID and confirmation message | `shark related-docs add "Doc" "path" --epic=E01` shows ID |
| 3.7 | Implement add JSON output | Support --json flag with valid JSON structure | `--json` flag produces valid JSON |
| 3.8 | Implement relatedDocsDeleteCmd | Arguments: title (positional), flags: --epic, --feature, --task (optional) | `shark related-docs delete "Doc" --epic=E01` works |
| 3.9 | Implement delete idempotency | Succeed silently even if link doesn't exist | Test: delete same link twice, both succeed |
| 3.10 | Implement delete confirmation | Success message indicating link removed | Output shows confirmation |
| 3.11 | Implement delete JSON output | Support --json flag | `--json` flag produces valid JSON |
| 3.12 | Implement relatedDocsListCmd | No required args, optional parent flags | `shark related-docs list --epic=E01` works |
| 3.13 | Implement list table output | Default format shows title, path, created_at columns | Human-readable table output |
| 3.14 | Implement list JSON output | Support --json flag | `--json` flag produces valid JSON |
| 3.15 | Implement list filtering | --epic, --feature, --task filters to specific parent | Test: --epic filter shows only epic documents |
| 3.16 | Implement list "no documents" case | Return empty list (not error) if no matches | Empty list handled gracefully |
| 3.17 | Register commands in root | `related-docs add/delete/list` in command hierarchy | `shark related-docs` shows subcommands |
| 3.18 | Wire DocumentRepository | Each command handler has access to repo instance | Commands can call repo methods |
| 3.19 | Add error handling | Proper error messages with context | DB errors reported clearly |

#### Quality Gates
- [ ] All three commands work correctly
- [ ] Mutually exclusive flags enforced (add, delete show errors for conflicts)
- [ ] Parent validation prevents linking to non-existent items
- [ ] JSON output is valid and well-formatted
- [ ] Error messages are helpful and actionable
- [ ] `make build` succeeds without errors
- [ ] Exit codes correct (0=success, 1=not found, 2=error)

#### Deliverables
- `internal/cli/commands/related_docs.go` with all three commands
- Proper flag handling and validation
- Correct DocumentRepository wiring
- Error handling and user-friendly messages

#### Testing Plan (Deferred to T-005)
- CLI command tests with mocked repository (25+ cases)
- Test flag validation and error cases
- Test output formatting (table and JSON)

---

### WAVE 4: Integration Tests (T-E07-F05-004)

**Task**: Write integration tests for document repository operations
**Priority**: P0
**Estimated Effort**: 8-10 hours
**Goal**: Validate all schema and repository functionality with real database

#### Test Structure

```
document_repository_test.go
├── TestCreateOrGet
│   ├── CreateNewDocument
│   ├── CreateExistingDocument (duplicate title+path)
│   ├── InvalidInputs (empty title, empty path)
│   └── TableDriven tests for multiple scenarios
│
├── TestGetByID
│   ├── ExistingDocument
│   ├── NonExistentDocument
│   └── InvalidID format
│
├── TestDelete
│   ├── ValidDelete
│   ├── VerifyCascadeDelete
│   └── NonExistentID (idempotent)
│
├── TestLinkToEpic/Feature/Task
│   ├── ValidLink
│   ├── NonExistentParent (FK constraint)
│   ├── DuplicateLink (UNIQUE constraint)
│   └── Verify link in DB
│
├── TestUnlinkFromEpic/Feature/Task
│   ├── ExistingLink
│   ├── NonExistentLink
│   ├── VerifyOtherLinks unaffected
│   └── Idempotent behavior
│
└── TestListForEpic/Feature/Task
    ├── WithDocuments
    ├── WithoutDocuments (empty)
    ├── MultipleDocuments
    └── Correct sorting
```

#### Test Implementation Steps

| Step | Description | Success Criteria | Validation |
|------|-------------|------------------|-----------|
| 4.1 | Create test file `document_repository_test.go` | File exists with test package | File compiles |
| 4.2 | Implement test database setup | Use testdb.TestDB() pattern, fresh DB per test | Each test has isolated DB |
| 4.3 | Create test fixtures | Helper functions for test docs, epics, features, tasks | Fixtures create valid test data |
| 4.4 | Test CreateOrGet new document | Insert new doc, verify returned, check DB | Assertion: ID > 0, data in DB |
| 4.5 | Test CreateOrGet existing document | Insert twice with same values, get same ID | Assertion: both calls return same ID |
| 4.6 | Test CreateOrGet validation | Empty title/path should fail | Error returned for invalid input |
| 4.7 | Test GetByID valid | Fetch document by ID | Returns correct document |
| 4.8 | Test GetByID non-existent | Fetch invalid ID | Returns error (sql.ErrNoRows) |
| 4.9 | Test Delete success | Delete document, verify gone | SELECT returns no rows |
| 4.10 | Test Delete cascade | Delete epic, verify links removed | epic_documents table empty |
| 4.11 | Test Link operations | Create link with valid parent | Link exists in junction table |
| 4.12 | Test Link FK constraint | Create link with invalid parent | Foreign key error |
| 4.13 | Test Link duplicate | Create same link twice | UNIQUE constraint error |
| 4.14 | Test Unlink operations | Remove specific link | Junction table no longer has link |
| 4.15 | Test Unlink isolation | Unlink one doc, verify others unaffected | Other links still exist |
| 4.16 | Test List operations | Fetch documents for parent | Returns all linked docs |
| 4.17 | Test List empty | Parent with no documents | Returns empty slice (not error) |
| 4.18 | Test transaction rollback | Link fails mid-operation | Transaction rolled back |
| 4.19 | Add error assertion helpers | Reusable helpers for constraint validation | Helper functions in test file |
| 4.20 | Implement table-driven tests | Use struct slices for multiple test cases | Organized test structure |

#### Quality Gates
- [ ] 30+ test cases covering all functionality
- [ ] All methods tested (CreateOrGet, GetByID, Delete, all Link/Unlink/List)
- [ ] All error conditions tested (FK, UNIQUE, not found)
- [ ] >90% code coverage for repository
- [ ] Tests are isolated (no cross-test dependencies)
- [ ] Tests are deterministic (no flaky tests)
- [ ] All tests complete in <100ms total
- [ ] `make test` shows all passing

#### Deliverables
- `internal/repository/document_repository_test.go` with 30+ test cases
- Test database setup and fixtures
- Error assertion helpers
- >90% coverage report

---

### WAVE 5: CLI Command Tests (T-E07-F05-005)

**Task**: Write CLI command tests and validation
**Priority**: P0
**Estimated Effort**: 8-10 hours
**Goal**: Validate all CLI functionality with mocked dependencies

#### Test Structure

```
related_docs_test.go
├── TestAddCommand
│   ├── ValidAddEpic
│   ├── ValidAddFeature
│   ├── ValidAddTask
│   ├── NoParentFlag (error)
│   ├── MultipleParentFlags (error)
│   ├── NonExistentParent (error)
│   ├── InvalidArguments (missing args)
│   ├── JSONOutput
│   └── SuccessMessage
│
├── TestDeleteCommand
│   ├── ValidDelete
│   ├── IdempotentDelete
│   ├── OptionalParentFlags
│   ├── JSONOutput
│   └── ConfirmationMessage
│
├── TestListCommand
│   ├── ListAll
│   ├── FilterByEpic
│   ├── FilterByFeature
│   ├── FilterByTask
│   ├── EmptyResults
│   ├── TableFormat
│   ├── JSONOutput
│   └── MultipleResults
│
├── TestFlagValidation
│   ├── MutuallyExclusiveFlags
│   ├── VerboseOutput
│   └── JSONValidation
│
└── TestErrorHandling
    ├── ClearErrorMessages
    ├── ExitCodes
    └── ParentNotFound
```

#### Test Implementation Steps

| Step | Description | Success Criteria | Validation |
|------|-------------|------------------|-----------|
| 5.1 | Create test file `related_docs_test.go` | File exists with test package | File compiles |
| 5.2 | Create mock repositories | MockDocumentRepository, MockEpicRepository, etc. | Mocks return predictable values |
| 5.3 | Create test fixtures | Mock documents, epics, features, tasks | Fixtures set up correctly |
| 5.4 | Test add with --epic | Command creates document and link | Mock methods called correctly |
| 5.5 | Test add with --feature | Command creates document and link | Correct parent type used |
| 5.6 | Test add with --task | Command creates document and link | Correct parent type used |
| 5.7 | Test add no parent flag | Command fails with helpful error | Error message: "exactly one parent" |
| 5.8 | Test add multiple parent flags | Command fails with helpful error | Error message: "mutually exclusive" |
| 5.9 | Test add non-existent parent | Command fails, specific not found error | Error: "Epic E99 not found" |
| 5.10 | Test add missing arguments | Command fails when title or path missing | Usage message shown |
| 5.11 | Test add JSON output | Flag produces valid JSON | json.Unmarshal succeeds |
| 5.12 | Test add success message | Output includes document ID | Message contains ID |
| 5.13 | Test delete existing link | Link removed from mock repo | Unlink called with correct args |
| 5.14 | Test delete non-existent link | No error (idempotent) | Succeeds silently |
| 5.15 | Test delete confirmation | Success message shown | Message displayed |
| 5.16 | Test delete JSON output | Valid JSON produced | json.Unmarshal succeeds |
| 5.17 | Test list all documents | No filter shows all | Mock ListForEpic not called |
| 5.18 | Test list filter by epic | --epic shows only that epic's docs | ListForEpic called with correct ID |
| 5.19 | Test list filter by feature | --feature shows only that feature's docs | ListForFeature called with correct ID |
| 5.20 | Test list filter by task | --task shows only that task's docs | ListForTask called with correct ID |
| 5.21 | Test list empty results | No documents found | Empty list returned gracefully |
| 5.22 | Test list table format | Default output is table | Column headers present |
| 5.23 | Test list JSON output | --json produces valid JSON | json.Unmarshal succeeds |
| 5.24 | Test output capture | Stdout captured for verification | Output can be tested |
| 5.25 | Test exit codes | Correct codes for success/error/not found | 0, 1, 2 returned appropriately |

#### Quality Gates
- [ ] 25+ test cases covering all commands
- [ ] All flag combinations tested
- [ ] All error conditions tested (validation, not found, constraints)
- [ ] Output formatting validated (table and JSON)
- [ ] Mocked repositories (no real database access)
- [ ] JSON output validates with json.Unmarshal
- [ ] Error messages are helpful and actionable
- [ ] `make test` runs all tests successfully

#### Deliverables
- `internal/cli/commands/related_docs_test.go` with 25+ test cases
- Mock repositories (Document, Epic, Feature, Task)
- Test fixtures and helpers
- Clear test documentation

---

## 5. QUALITY GATES BETWEEN WAVES

### Gate 1: After T-001 (Schema) - Code Review & Schema Validation

**Checkpoints**:
- [ ] Migration function is idempotent (safe to run multiple times)
- [ ] All table definitions use IF NOT EXISTS
- [ ] Foreign key constraints properly set with ON DELETE CASCADE
- [ ] All indexes created for query optimization
- [ ] Schema matches PRD architecture review specification

**Validation Commands**:
```bash
# Verify schema exists
sqlite3 shark-tasks.db ".schema documents"
sqlite3 shark-tasks.db ".schema epic_documents"
sqlite3 shark-tasks.db ".schema feature_documents"
sqlite3 shark-tasks.db ".schema task_documents"

# Verify indexes
sqlite3 shark-tasks.db ".indices"

# Test constraints
sqlite3 shark-tasks.db "PRAGMA foreign_keys;"  # Should be ON
```

**Go/No-Go Decision**:
- **Go to T-002** if: Schema is correct, migration tested on clean + existing DB, `make test` passes
- **Redo T-001** if: Schema missing columns, constraints not enforced, migration is not idempotent

---

### Gate 2: After T-002 (Repository) - Code Review & Unit Testing

**Checkpoints**:
- [ ] All 14 methods implemented and callable
- [ ] CreateOrGet idempotency verified
- [ ] No SQL injection vulnerabilities (parameterized queries only)
- [ ] All errors wrapped with context
- [ ] Code compiles and passes lint
- [ ] Proper transaction handling in Link methods

**Validation**:
```bash
# Compilation
go build ./internal/repository

# Lint check
make lint

# Manual smoke test of methods
# (Will be formalized in T-004)
```

**Go/No-Go Decision**:
- **Go to T-003** if: All methods compile, no lint errors, code follows project patterns
- **Redo T-002** if: Missing methods, SQL injection vulnerability, improper error handling

---

### Gate 3: After T-003 (CLI) - Manual Testing & Code Review

**Checkpoints**:
- [ ] All three commands work (add, delete, list)
- [ ] Flag validation enforced (mutually exclusive, required)
- [ ] Parent existence validated before linking
- [ ] JSON output is valid
- [ ] Error messages are helpful
- [ ] Exit codes correct
- [ ] Commands registered in CLI hierarchy

**Validation**:
```bash
# Build and test manually
make build
./bin/shark related-docs --help
./bin/shark related-docs add --help
./bin/shark related-docs delete --help
./bin/shark related-docs list --help

# Manual functional tests (requires running test DB)
shark init --non-interactive
shark epic create "E01" # Create test epic
shark related-docs add "Test Doc" "docs/test.md" --epic=E01
shark related-docs list --epic=E01
shark related-docs delete "Test Doc" --epic=E01
```

**Go/No-Go Decision**:
- **Go to T-004** if: All commands work, manual tests pass, code follows patterns
- **Redo T-003** if: Commands missing, validation not working, poor error messages

---

### Gate 4: After T-004 (Integration Tests) - Test Coverage & Quality

**Checkpoints**:
- [ ] 30+ test cases written
- [ ] All schema and repository functionality tested
- [ ] All error conditions tested (FK, UNIQUE, not found)
- [ ] >90% code coverage for repository
- [ ] All tests passing with `make test`
- [ ] Tests are isolated and deterministic

**Validation**:
```bash
# Run all tests
make test

# Check coverage
make test-coverage
# Check coverage for document_repository.go

# Run specific test file
go test -v ./internal/repository -run TestDocument
```

**Go/No-Go Decision**:
- **Go to T-005** if: 30+ tests pass, >90% coverage, no flaky tests
- **Redo T-004** if: <30 tests, <85% coverage, tests are flaky or fail

---

### Gate 5: After T-005 (CLI Tests) - CLI Quality & Integration

**Checkpoints**:
- [ ] 25+ test cases written
- [ ] All commands tested (add, delete, list)
- [ ] All flags tested (--epic, --feature, --task, --json, --verbose)
- [ ] All error cases tested
- [ ] Mocked repositories (no DB access)
- [ ] JSON output validated
- [ ] All tests passing with `make test`

**Validation**:
```bash
# Run all CLI tests
make test

# Run specific test file
go test -v ./internal/cli/commands -run TestRelated

# Manual integration test
make build
shark related-docs list --json  # Should work without error
```

**Go/No-Go Decision**:
- **FEATURE COMPLETE** if: 25+ tests pass, all commands working, proper mocking
- **Redo T-005** if: <25 tests, real DB access in tests, missing error cases

---

## 6. INTEGRATION VALIDATION APPROACH

### End-to-End Feature Validation

Once all 5 tasks complete, perform comprehensive integration testing:

#### Test Scenario 1: Complete Add-List-Delete Workflow

```bash
# Setup
shark init --non-interactive
shark epic create "Foundation" --priority=high
# E01 created

shark feature create --epic=E01 "Authentication" --execution-order=1
# E01-F01 created

shark task create --epic=E01 --feature=E01-F01 \
  "Implement OAuth" --priority=5
# T-E01-F01-001 created

# Workflow: Add document to task
shark related-docs add "OAuth Spec" "docs/oauth-spec.md" \
  --task=T-E01-F01-001
# Expected: Document added, link created

# Verify document appears in list
shark related-docs list --task=T-E01-F01-001
# Expected: Table shows OAuth Spec document

# Delete link
shark related-docs delete "OAuth Spec" --task=T-E01-F01-001
# Expected: Confirmation message

# Verify deletion
shark related-docs list --task=T-E01-F01-001
# Expected: Empty (or no OAuth Spec listed)
```

#### Test Scenario 2: Document Reuse Across Multiple Parents

```bash
# Add same document to multiple parents
shark related-docs add "Architecture" "docs/architecture.md" --epic=E01
shark related-docs add "Architecture" "docs/architecture.md" --feature=E01-F01
shark related-docs add "Architecture" "docs/architecture.md" --task=T-E01-F01-001

# Verify reuse (same document ID)
shark related-docs list --epic=E01 --json
# Should show document with unique ID

shark related-docs list --feature=E01-F01 --json
# Should show same document ID

shark related-docs list --task=T-E01-F01-001 --json
# Should show same document ID

# Delete from one parent
shark related-docs delete "Architecture" --task=T-E01-F01-001

# Verify other links remain
shark related-docs list --epic=E01
# Should still show Architecture

shark related-docs list --feature=E01-F01
# Should still show Architecture

shark related-docs list --task=T-E01-F01-001
# Should NOT show Architecture
```

#### Test Scenario 3: Error Handling

```bash
# Test 1: Non-existent parent
shark related-docs add "Doc" "docs/doc.md" --epic=E99
# Expected: Error "Epic E99 not found"

# Test 2: Missing required flag
shark related-docs add "Doc" "docs/doc.md"
# Expected: Error "exactly one parent flag required"

# Test 3: Multiple parent flags
shark related-docs add "Doc" "docs/doc.md" --epic=E01 --feature=E01-F01
# Expected: Error "mutually exclusive"

# Test 4: Idempotent delete
shark related-docs delete "NonExistent" --epic=E01
# Expected: Success (no error)

shark related-docs delete "NonExistent" --epic=E01
# Expected: Success again (idempotent)
```

#### Test Scenario 4: JSON Output Validation

```bash
# All commands should support --json flag
shark related-docs list --epic=E01 --json | jq .
# Should be valid JSON

shark related-docs add "Doc" "docs/doc.md" --epic=E01 --json | jq .
# Should be valid JSON

shark related-docs delete "Doc" --epic=E01 --json | jq .
# Should be valid JSON
```

### Database Integrity Checks

After integration testing, validate database state:

```bash
# Count documents
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM documents;"

# Count links
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM epic_documents;"
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM feature_documents;"
sqlite3 shark-tasks.db "SELECT COUNT(*) FROM task_documents;"

# Verify no orphaned links
sqlite3 shark-tasks.db "
  SELECT * FROM epic_documents
  WHERE epic_id NOT IN (SELECT id FROM epics);
"
# Expected: 0 rows (no orphans)

# Verify uniqueness
sqlite3 shark-tasks.db "
  SELECT title, file_path, COUNT(*) as cnt
  FROM documents
  GROUP BY title, file_path
  HAVING cnt > 1;
"
# Expected: 0 rows (no duplicates)
```

---

## 7. RISK ASSESSMENT & MITIGATION

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| **Migration idempotency issues** | Medium | High | Test migration on clean + existing DB before release; use IF NOT EXISTS everywhere |
| **Foreign key cascade delete affects wrong data** | Low | High | Test cascade manually; verify unrelated data unaffected |
| **SQL injection in dynamic queries** | Low | Critical | Use parameterized queries only; code review all SQL |
| **Performance issues with large datasets** | Low | Medium | Create indexes on all FK columns; test with 10k documents |
| **Race conditions in transactions** | Low | Medium | Test concurrent access; use proper transaction boundaries |
| **Mock repositories don't match real behavior** | Medium | Medium | Keep mocks in sync with real repo; use same interface |
| **JSON output format inconsistency** | Low | Low | Define JSON schema; validate in tests |

### Schedule Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| **T-002 takes longer than estimated** | Medium | Medium | Start with core methods (CRUD), add Link/Unlink after |
| **Testing reveals design flaws** | Medium | High | Implement T-001 carefully; get schema review before T-002 |
| **Scope creep (integration with task/epic views)** | Medium | Medium | Stick to MVP (links only, no view integration) |

### Dependencies

| Dependency | Status | Mitigation |
|-----------|--------|-----------|
| SQLite WAL mode, FK constraints | Available | Verify PRAGMA settings |
| Cobra CLI framework | Available | Use existing patterns |
| Test database infrastructure | Available | Use testdb.TestDB() pattern |
| Epic/Feature/Task repositories | Available | Reuse existing instances |

---

## 8. REQUIREMENT-TO-IMPLEMENTATION MAPPING

### Schema Requirements (SCH-001 to SCH-010)

**Task**: T-E07-F05-001
**Implementation**:
- T-001 Step 1.3: documents table with UNIQUE(title, file_path)
- T-001 Step 1.4-1.6: Link tables with composite PKs
- T-001 Step 1.7: ON DELETE CASCADE
- T-001 Step 1.8: Indexes for FK columns
- T-001 Step 1.9-1.10: Migration testing

**Validation**: Schema validation gate after T-001

---

### Repository Requirements (REP-001 to REP-018)

**Task**: T-E07-F05-002
**Implementation**:
- T-002 Step 2.1-2.2: Document model and validation
- T-002 Step 2.4-2.5: CreateOrGet, GetByID, Delete (CRUD)
- T-002 Step 2.6-2.8: Link operations (LinkToEpic/Feature/Task)
- T-002 Step 2.9-2.11: Unlink operations
- T-002 Step 2.12-2.14: List operations
- T-002 Step 2.15-2.18: Error wrapping, parameterized queries, transactions

**Validation**: Integration test gate after T-004

---

### CLI Requirements (CLI-001 to CLI-018)

**Task**: T-E07-F05-003
**Implementation**:
- T-003 Step 3.1-3.7: Add command with validation and output
- T-003 Step 3.8-3.11: Delete command with idempotency
- T-003 Step 3.12-3.16: List command with filtering
- T-003 Step 3.17-3.19: Command registration, repository wiring, error handling

**Validation**: Manual testing gate after T-003, CLI test gate after T-005

---

### Testing Requirements (TEST-001 to TEST-023)

**Tasks**: T-E07-F05-004, T-E07-F05-005
**Implementation**:
- T-004: 30+ integration tests validating schema + repository
- T-005: 25+ CLI tests with mocked repositories

**Validation**: Test coverage gates after T-004 and T-005

---

## 9. TASK BREAKDOWN WITH ESTIMATED HOURS

### Time Allocation by Phase

| Phase | Task | Hours | Key Activities |
|-------|------|-------|-----------------|
| **Schema** | T-001 | 5-6 | Migration setup, 4 tables, indexes, testing |
| **Repository** | T-002 | 8-10 | Model, 14 methods, error wrapping, parameterized queries |
| **CLI** | T-003 | 6-8 | 3 commands, flag validation, parent validation, output formatting |
| **Integration Tests** | T-004 | 8-10 | Test setup, 30+ test cases, fixtures, >90% coverage |
| **CLI Tests** | T-005 | 8-10 | Mock setup, 25+ test cases, output validation |
| **Integration & QA** | Manual | 2-3 | End-to-end scenarios, database integrity checks |
| **TOTAL** | | **40-50** | Complete feature with comprehensive testing |

### Per-Task Breakdown

**T-E07-F05-001 (Schema)**: 5-6 hours
- 1 hour: Analyze existing migration patterns
- 1.5 hours: Implement 4 tables and constraints
- 1.5 hours: Add indexes and cascade configuration
- 1 hour: Test on clean and existing databases

**T-E07-F05-002 (Repository)**: 8-10 hours
- 1 hour: Create Document model and validation
- 3 hours: Implement CRUD methods (CreateOrGet, GetByID, Delete)
- 3 hours: Implement Link/Unlink methods (9 methods)
- 1.5 hours: Implement List methods (3 methods)
- 1 hour: Error wrapping and final review

**T-E07-F05-003 (CLI)**: 6-8 hours
- 1 hour: Command structure and add command setup
- 2 hours: Add command with flag validation and parent validation
- 1.5 hours: Delete and List commands
- 1.5 hours: Output formatting (table, JSON), error handling
- 1 hour: Command registration and manual testing

**T-E07-F05-004 (Integration Tests)**: 8-10 hours
- 1 hour: Test setup and fixtures
- 2 hours: CRUD test cases (CreateOrGet, GetByID, Delete)
- 3 hours: Link/Unlink/List test cases (18+ tests)
- 1.5 hours: Error condition tests (FK, UNIQUE constraints)
- 1.5 hours: Finalization, coverage reporting

**T-E07-F05-005 (CLI Tests)**: 8-10 hours
- 1 hour: Mock setup and test fixtures
- 2 hours: Add command tests (8+ tests)
- 2 hours: Delete and List command tests (12+ tests)
- 2 hours: Flag validation and error condition tests
- 1 hour: JSON output validation, finalization

---

## 10. SUCCESS CRITERIA & COMPLETION CHECKLIST

### Feature-Level Success Criteria

- [ ] **All 5 tasks completed** with no blockers
- [ ] **Database schema** created correctly with migrations
- [ ] **DocumentRepository** fully implemented with 14 methods
- [ ] **CLI commands** (add, delete, list) work correctly
- [ ] **30+ integration tests** passing with >90% coverage
- [ ] **25+ CLI tests** passing with mocked dependencies
- [ ] **Manual end-to-end testing** successful (4 scenarios)
- [ ] **Database integrity** verified (no orphans, proper constraints)
- [ ] **`make test`** passes with all tests green
- [ ] **`make build`** succeeds without errors
- [ ] **Code review** completed with all feedback addressed

### Per-Task Completion Criteria

**T-E07-F05-001 (Schema)**:
- [ ] Migration function created and tested
- [ ] All 4 tables created with correct columns and constraints
- [ ] All 5 indexes created for performance
- [ ] ON DELETE CASCADE tested and working
- [ ] Schema verified with sqlite3 commands
- [ ] Migration tested on clean and existing databases

**T-E07-F05-002 (Repository)**:
- [ ] Document model created with validation
- [ ] All 14 methods implemented
- [ ] All methods use parameterized queries
- [ ] All errors wrapped with context
- [ ] Transactions used for multi-statement operations
- [ ] Code compiles and passes lint
- [ ] Constructor properly receives DB instance

**T-E07-F05-003 (CLI)**:
- [ ] related-docs add command works
- [ ] related-docs delete command works
- [ ] related-docs list command works
- [ ] Flag validation enforced (mutually exclusive)
- [ ] Parent existence validated before linking
- [ ] JSON output works for all commands
- [ ] Error messages are helpful and clear
- [ ] Commands registered in CLI hierarchy
- [ ] `shark related-docs --help` shows all subcommands

**T-E07-F05-004 (Integration Tests)**:
- [ ] 30+ test cases written
- [ ] All CRUD operations tested
- [ ] All Link/Unlink operations tested
- [ ] All List operations tested
- [ ] Error conditions tested (FK, UNIQUE, not found)
- [ ] Cascade delete tested
- [ ] >90% code coverage for document_repository.go
- [ ] All tests isolated and deterministic
- [ ] All tests passing with `make test`

**T-E07-F05-005 (CLI Tests)**:
- [ ] 25+ test cases written
- [ ] All commands tested (add, delete, list)
- [ ] All flags tested (--epic, --feature, --task, --json, --verbose)
- [ ] Error cases tested (validation, not found, constraints)
- [ ] Mock repositories in place (no DB access)
- [ ] JSON output validated
- [ ] Exit codes verified (0, 1, 2)
- [ ] All tests passing with `make test`

---

## 11. DETAILED IMPLEMENTATION REFERENCE

### File Structure

```
internal/
├── db/
│   └── db.go                    (T-001: Add migrateDocumentTables())
│
├── models/
│   └── document.go              (T-002: Document struct + validation)
│
├── repository/
│   ├── document_repository.go               (T-002: 14 methods)
│   ├── document_repository_test.go          (T-004: 30+ tests)
│   └── ...existing repositories unchanged...
│
└── cli/
    └── commands/
        ├── related_docs.go                  (T-003: 3 commands)
        ├── related_docs_test.go             (T-005: 25+ tests)
        └── ...existing commands unchanged...
```

### Database Schema Summary

**documents table**:
```sql
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, file_path)
)
```

**epic_documents table**:
```sql
CREATE TABLE IF NOT EXISTS epic_documents (
    epic_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(epic_id, document_id),
    FOREIGN KEY(epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE CASCADE
)
```

Similar for feature_documents and task_documents.

**Indexes**:
```sql
CREATE INDEX IF NOT EXISTS idx_documents_title ON documents(title)
CREATE INDEX IF NOT EXISTS idx_documents_file_path ON documents(file_path)
CREATE INDEX IF NOT EXISTS idx_epic_documents_epic_id ON epic_documents(epic_id)
CREATE INDEX IF NOT EXISTS idx_feature_documents_feature_id ON feature_documents(feature_id)
CREATE INDEX IF NOT EXISTS idx_task_documents_task_id ON task_documents(task_id)
```

### API Contract Summary

**Add Document**:
```
Command: shark related-docs add <title> <path> --epic=<key> | --feature=<key> | --task=<key>

Response (success):
{
  "id": 123,
  "title": "Architecture Doc",
  "file_path": "docs/architecture.md",
  "created_at": "2025-12-20T10:30:00Z",
  "message": "Document linked to Epic E01"
}

Exit Code: 0
```

**Delete Document**:
```
Command: shark related-docs delete <title> [--epic=<key>] [--feature=<key>] [--task=<key>]

Response (success):
{
  "message": "Document link removed from Epic E01"
}

Exit Code: 0
```

**List Documents**:
```
Command: shark related-docs list [--epic=<key>] [--feature=<key>] [--task=<key>] [--json]

Response (table, default):
| Title         | File Path            | Created At          |
|---------------|----------------------|---------------------|
| Architecture  | docs/arch.md         | 2025-12-20 10:30:00 |
| API Spec      | docs/api/spec.md     | 2025-12-20 10:31:00 |

Response (JSON):
{
  "documents": [
    {
      "id": 123,
      "title": "Architecture",
      "file_path": "docs/arch.md",
      "created_at": "2025-12-20T10:30:00Z"
    }
  ]
}

Exit Code: 0
```

---

## 12. APPENDIX: TESTING PATTERNS

### Integration Test Pattern (T-004)

```go
func TestCreateOrGet(t *testing.T) {
    // Setup
    db := testdb.TestDB(t)
    repo := NewDocumentRepository(db)

    // Test: Create new document
    t.Run("CreateNewDocument", func(t *testing.T) {
        doc, err := repo.CreateOrGet(context.Background(), "Title", "path/doc.md")
        if err != nil {
            t.Fatalf("CreateOrGet failed: %v", err)
        }
        if doc.ID == 0 {
            t.Fatal("Expected non-zero ID")
        }
        if doc.Title != "Title" {
            t.Errorf("Expected title 'Title', got %q", doc.Title)
        }
    })

    // Test: Get existing document
    t.Run("GetExisting", func(t *testing.T) {
        doc1, _ := repo.CreateOrGet(context.Background(), "Title", "path/doc.md")
        doc2, _ := repo.CreateOrGet(context.Background(), "Title", "path/doc.md")

        if doc1.ID != doc2.ID {
            t.Errorf("Expected same ID, got %d and %d", doc1.ID, doc2.ID)
        }
    })
}
```

### CLI Test Pattern (T-005)

```go
func TestAddCommand(t *testing.T) {
    // Setup mocks
    mockDocRepo := &MockDocumentRepository{
        Documents: make(map[int64]*models.Document),
        NextID:    1,
    }
    mockEpicRepo := &MockEpicRepository{
        Epics: map[string]*models.Epic{
            "E01": {Key: "E01", ID: 1, Title: "Foundation"},
        },
    }

    // Test: Valid add
    t.Run("ValidAddEpic", func(t *testing.T) {
        cmd := createAddCmd(mockDocRepo, mockEpicRepo)
        // Set flags and args
        cmd.SetArgs([]string{"add", "Test Doc", "docs/test.md", "--epic=E01"})

        err := cmd.Execute()
        if err != nil {
            t.Fatalf("Command failed: %v", err)
        }

        // Verify mock was called
        if mockDocRepo.CreateOrGetCalls == 0 {
            t.Fatal("CreateOrGet not called")
        }
    })
}
```

---

## 13. HANDOFF CHECKLIST FOR DEVELOPERS

When starting implementation:

- [ ] Read this master plan document in full
- [ ] Review the 5 task files (T-E07-F05-001 through 005)
- [ ] Review the feature PRD (prd.md)
- [ ] Review existing repository patterns in internal/repository/
- [ ] Review existing CLI command patterns in internal/cli/commands/
- [ ] Understand project error handling patterns (fmt.Errorf with %w)
- [ ] Understand SQLite migration pattern in internal/db/db.go
- [ ] Understand project testing patterns (testdb, mocks)
- [ ] Set up development workspace for artifacts
- [ ] Start with T-001 (cannot start T-002 until T-001 done)
- [ ] Complete quality gates between waves
- [ ] Perform end-to-end validation after T-005

---

## 14. DOCUMENT METADATA

**Document Type**: Master Implementation Plan
**Feature**: E07-F05 (Add Related Documents)
**Created**: 2025-12-20
**Last Updated**: 2025-12-20
**Status**: Ready for Implementation
**Complexity**: Medium (5 tasks, ~50 hours, moderate technical depth)
**Priority**: P0 (MustHave)

**Related Documents**:
- Feature PRD: `/docs/plan/E07-enhancements/E07-F05-add-related-documents/prd.md`
- Task Files: `/docs/plan/E07/E07-F05/tasks/T-E07-F05-00[1-5].md`
- Epic PRD: `/docs/plan/E07-enhancements/epic.md` (if available)

