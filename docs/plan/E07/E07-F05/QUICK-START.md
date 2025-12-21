# E07-F05 Quick Start Guide

## What Is This Feature?

Add the ability to link supporting documents (architecture diagrams, design docs, QA reports, API specs) to epics, features, and tasks via a new CLI command group.

**Command Examples**:
```bash
shark related-docs add "OAuth Spec" "docs/oauth.md" --task=T-E01-F01-001
shark related-docs list --epic=E01
shark related-docs delete "OAuth Spec" --epic=E01
```

---

## Implementation Overview

5 sequential tasks, ~50 hours total:

| Task | Hours | What | Status |
|------|-------|------|--------|
| T-001 | 5-6 | Database schema (4 tables, migration) | Todo |
| T-002 | 8-10 | DocumentRepository (14 methods) | Todo |
| T-003 | 6-8 | CLI commands (add/delete/list) | Todo |
| T-004 | 8-10 | Integration tests (30+ tests) | Todo |
| T-005 | 8-10 | CLI tests (25+ tests) | Todo |

**Sequential** - Must complete in order (T-001 before T-002, etc)

---

## Key Architecture Decisions

### Why 4 Tables?
- documents: Centralized metadata (id, title, file_path, created_at)
- epic_documents: Links to epics
- feature_documents: Links to features
- task_documents: Links to tasks

Reason: Type safety, better performance, clearer schema

### Why CreateOrGet Method?
Upsert pattern that reuses documents across multiple parents
- No duplicate documents with same title+path
- One call instead of create + get
- Idempotent behavior

### Why Separate CLI Tests with Mocks?
- No database required during CLI testing
- Faster execution
- Tests command logic independently
- Follows project pattern

---

## Implementation Sequence

### Wave 1: Database Schema (T-001)
1. Add migration function `migrateDocumentTables()` to `internal/db/db.go`
2. Create 4 tables with constraints
3. Add 5 indexes
4. Test on clean + existing database

**Quality Gate**: Schema verified with sqlite3 commands

### Wave 2: Repository (T-002)
1. Create Document model (internal/models/document.go)
2. Create DocumentRepository (internal/repository/document_repository.go)
3. Implement 14 methods:
   - CreateOrGet, GetByID, Delete (CRUD)
   - LinkToEpic, LinkToFeature, LinkToTask (Link)
   - UnlinkFromEpic, UnlinkFromFeature, UnlinkFromTask (Unlink)
   - ListForEpic, ListForFeature, ListForTask (List)

**Quality Gate**: All methods callable, lint passes

### Wave 3: CLI Commands (T-003)
1. Create command group (internal/cli/commands/related_docs.go)
2. Implement add command (flag validation, parent validation)
3. Implement delete command (idempotent behavior)
4. Implement list command (filtering support)

**Quality Gate**: Manual testing passes for all commands

### Wave 4: Integration Tests (T-004)
1. Create test file (internal/repository/document_repository_test.go)
2. Write 30+ tests covering:
   - CRUD operations
   - Link/Unlink operations
   - List operations
   - Error conditions (FK, UNIQUE constraints)
   - Cascade delete

**Quality Gate**: 30+ tests passing, >90% coverage

### Wave 5: CLI Tests (T-005)
1. Create test file (internal/cli/commands/related_docs_test.go)
2. Write 25+ tests with mocked repositories
3. Test all commands, flags, error cases, JSON output

**Quality Gate**: 25+ tests passing, mocked repos only

---

## Critical Implementation Details

### Must-Have Requirements
These are non-negotiable:

1. **Parameterized queries only** - No SQL string concatenation
2. **Error wrapping** - `fmt.Errorf("context: %w", err)`
3. **Transactions** - `defer tx.Rollback()` for multi-statement ops
4. **Idempotent migrations** - Use `IF NOT EXISTS` everywhere
5. **Foreign key constraints** - ON DELETE CASCADE tested
6. **Composite PKs** - (parent_id, document_id) on link tables

### Code Patterns to Follow

**Error Handling**:
```go
// Good
doc, err := repo.GetByID(ctx, id)
if err != nil {
    return fmt.Errorf("fetch document: %w", err)
}

// Bad
doc, err := repo.GetByID(ctx, id)
if err != nil {
    log.Fatal(err)  // No wrapping
}
```

**Parameterized Queries**:
```go
// Good
row := db.QueryRowContext(ctx, "SELECT * FROM documents WHERE id = ?", id)

// Bad
row := db.QueryRowContext(ctx, "SELECT * FROM documents WHERE id = " + strconv.Itoa(int(id)))
```

**Transactions**:
```go
// Good
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return fmt.Errorf("begin transaction: %w", err)
}
defer tx.Rollback()
// ... operations ...
if err := tx.Commit(); err != nil {
    return fmt.Errorf("commit: %w", err)
}

// Bad
tx, _ := db.BeginTx(ctx, nil)
// ... operations ...
tx.Commit()  // No error handling
```

---

## Testing Strategy

### T-004 Integration Tests (30+)

Test each method independently:

```go
func TestCreateOrGet(t *testing.T) {
    db := testdb.TestDB(t)
    repo := NewDocumentRepository(db)

    // Test 1: Create new
    doc, err := repo.CreateOrGet(ctx, "Title", "path.md")
    if err != nil { t.Fatal(err) }
    if doc.ID == 0 { t.Fatal("Expected ID") }

    // Test 2: Get existing
    doc2, _ := repo.CreateOrGet(ctx, "Title", "path.md")
    if doc.ID != doc2.ID { t.Errorf("Expected same ID") }
}
```

### T-005 CLI Tests (25+)

Test commands with mocked repositories:

```go
func TestAddCommand(t *testing.T) {
    mockRepo := &MockDocumentRepository{ /* ... */ }
    mockEpicRepo := &MockEpicRepository{
        Epics: map[string]*models.Epic{"E01": {...}},
    }

    cmd := createAddCmd(mockRepo, mockEpicRepo)
    cmd.SetArgs([]string{"add", "Title", "path", "--epic=E01"})

    err := cmd.Execute()
    if err != nil { t.Fatal(err) }
    if mockRepo.CreateOrGetCalls == 0 { t.Fatal("CreateOrGet not called") }
}
```

---

## Validation Checklist

Use this to verify implementation is correct:

### T-001 Schema
```bash
# Verify schema
sqlite3 shark-tasks.db ".schema documents"
sqlite3 shark-tasks.db ".schema epic_documents"
sqlite3 shark-tasks.db ".schema feature_documents"
sqlite3 shark-tasks.db ".schema task_documents"

# Verify indexes
sqlite3 shark-tasks.db ".indices"

# Test cascade delete
sqlite3 shark-tasks.db "PRAGMA foreign_keys;"  # Should show 1
```

### T-002 Repository
```bash
# Check it compiles
go build ./internal/repository

# Run linter
make lint

# Verify no SQL injection
grep -r "fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT" internal/repository/document_repository.go
# Should return 0 results
```

### T-003 CLI
```bash
# Build
make build

# Test manually
./bin/shark related-docs --help
./bin/shark related-docs add --help
./bin/shark related-docs delete --help
./bin/shark related-docs list --help

# Test commands
shark init --non-interactive
shark epic create "E01" --title="Test Epic"
shark related-docs add "Doc" "docs/doc.md" --epic=E01
shark related-docs list --epic=E01
```

### T-004 & T-005 Tests
```bash
# Run tests
make test

# Check coverage
make test-coverage
# Open coverage.html

# Run specific test
go test -v ./internal/repository -run TestDocument
go test -v ./internal/cli/commands -run TestRelated
```

---

## Common Pitfalls to Avoid

### Pitfall 1: Not Using IF NOT EXISTS
```go
// Bad
CREATE TABLE documents (...)

// Good
CREATE TABLE IF NOT EXISTS documents (...)
```

### Pitfall 2: Forgetting Error Wrapping
```go
// Bad
result, _ := db.Exec(...)

// Good
result, err := db.Exec(...)
if err != nil {
    return fmt.Errorf("execute SQL: %w", err)
}
```

### Pitfall 3: Not Testing Cascade Delete
```go
// Must verify in T-004
// Delete epic, check epic_documents is empty
// Delete feature, check feature_documents is empty
```

### Pitfall 4: CLI Tests Accessing Real Database
```go
// Bad - Don't do this in T-005
docRepo := repository.NewDocumentRepository(realDB)

// Good
docRepo := &MockDocumentRepository{...}
```

### Pitfall 5: Missing Parent Validation in CLI
```go
// Bad - Doesn't check if epic exists
shark related-docs add "Doc" "path" --epic=E99  // Succeeds with DB error

// Good - Validates first
if !epicRepo.Exists("E99") {
    return fmt.Errorf("Epic E99 not found")
}
```

---

## File Locations

### New Files
- `/internal/models/document.go` (T-002)
- `/internal/repository/document_repository.go` (T-002)
- `/internal/repository/document_repository_test.go` (T-004)
- `/internal/cli/commands/related_docs.go` (T-003)
- `/internal/cli/commands/related_docs_test.go` (T-005)

### Modified Files
- `/internal/db/db.go` (T-001 - add migration)
- `/internal/cli/commands/root.go` or equivalent (T-003 - register commands)

---

## Documentation References

1. **Full Master Plan**: MASTER-IMPLEMENTATION-PLAN.md
2. **Executive Summary**: EXECUTIVE-SUMMARY.md
3. **Requirement Traceability**: REQUIREMENT-TRACEABILITY-MATRIX.md
4. **Original Tasks**:
   - T-E07-F05-001.md (Schema)
   - T-E07-F05-002.md (Repository)
   - T-E07-F05-003.md (CLI)
   - T-E07-F05-004.md (Integration Tests)
   - T-E07-F05-005.md (CLI Tests)
5. **Feature PRD**: `/docs/plan/E07-enhancements/E07-F05-add-related-documents/prd.md`

---

## Project Standards

### Error Handling
- Wrap all errors: `fmt.Errorf("context: %w", err)`
- Use meaningful messages
- Exit codes: 0 (success), 1 (not found), 2 (DB error)

### Code Style
- Follow Go idioms (use `make fmt`)
- Use `context.Context` for all repository methods
- Use parameterized queries (? placeholders)

### Testing
- Use testdb.TestDB() for test databases
- Use defer tx.Rollback() for isolation
- Use table-driven tests for multiple cases
- Aim for >90% coverage

### Git
- Commit frequently with clear messages
- Only commit working code
- Do not delete database files

---

## Need Help?

1. Read the full MASTER-IMPLEMENTATION-PLAN.md
2. Check existing repository patterns (task_repository.go, epic_repository.go)
3. Check existing CLI patterns (task.go, epic.go in cli/commands/)
4. Review project testing patterns (task_repository_test.go)
5. Check CLAUDE.md for project-specific guidelines

---

**Status**: Ready for Implementation
**Created**: 2025-12-20
**Total Requirements**: 49 across 5 tasks
**Estimated Timeline**: 2 weeks with focused development

