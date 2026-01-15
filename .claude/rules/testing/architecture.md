---
paths: "**/*_test.go"
---

# Testing Architecture

This rule is loaded when working with Go test files.

## ⚠️ TESTING GOLDEN RULE ⚠️

**ONLY repository tests should use the real database. Everything else MUST use mocked repositories.**

This rule is critical for:
- Test isolation (no data pollution between tests)
- Test speed (in-memory mocks are 100x faster)
- Test reliability (no flaky tests from database state)
- Parallel test execution (no database contention)

## Test Categories

### 1. Repository Tests (`internal/repository/*_test.go`)
**✅ USE REAL DATABASE - MUST CLEAN UP**

See `.claude/rules/testing/repository-tests.md` for details.

### 2. CLI Command Tests (`internal/cli/commands/*_test.go`)
**❌ NEVER USE REAL DATABASE - USE MOCKS**

See `.claude/rules/testing/cli-tests.md` for details.

### 3. Service Layer Tests (`internal/sync/*_test.go`, etc.)
**❌ NEVER USE REAL DATABASE - USE MOCKS**

These tests verify business logic and service orchestration.

**Requirements:**
- Mock all repository dependencies
- Test service logic in isolation
- Verify correct repository method calls
- Test error propagation and handling

### 4. Unit Tests (models, utils, parsers)
**❌ NO DATABASE - PURE LOGIC**

These test pure functions with no dependencies.

**Requirements:**
- No database, no file system, no network
- Test data transformations, validations, parsing
- Fast, deterministic, parallel-safe

## Test Organization

```
internal/
├── repository/
│   ├── task_repository.go
│   └── task_repository_test.go       # ✅ Uses real DB + cleanup
├── cli/commands/
│   ├── task.go
│   ├── task_test.go                  # ❌ Uses mocks only
│   └── mock_task_repository.go       # Mock interface
├── sync/
│   ├── sync.go
│   └── sync_test.go                  # ❌ Uses mocks only
└── models/
    ├── task.go
    └── task_test.go                  # ❌ Pure logic only
```

## Running Tests

```bash
make test              # Full suite
make test-coverage     # With coverage HTML report
go test -v ./...       # Manual verbose run
go test -run TestName  # Specific test

# Run only repository tests (with DB)
go test -v ./internal/repository

# Run only CLI tests (no DB, fast)
go test -v ./internal/cli/commands
```

## Test Database

- Location: `internal/repository/test-shark-tasks.db`
- Created by: `internal/test/testdb.go`
- Shared across repository tests (fast, avoids recreation)
- MUST be cleaned before each test to avoid pollution

## Testing Patterns

### Table-Driven Tests

Use table-driven tests for testing multiple cases:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid", false},
        {"empty input", "", true},
        {"too long", strings.Repeat("a", 1000), true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Subtests

Use `t.Run()` for organizing related checks:

```go
func TestTaskOperations(t *testing.T) {
    t.Run("creation", func(t *testing.T) {
        // Test task creation
    })

    t.Run("update", func(t *testing.T) {
        // Test task update
    })

    t.Run("deletion", func(t *testing.T) {
        // Test task deletion
    })
}
```

### Test Helpers

Extract common setup into helper functions:

```go
func setupTest(t *testing.T) (*TaskRepository, func()) {
    db := test.GetTestDB()
    repo := NewTaskRepository(db)

    // Cleanup function
    cleanup := func() {
        db.Exec("DELETE FROM tasks WHERE key LIKE 'TEST-%'")
    }

    // Run cleanup before test
    cleanup()

    return repo, cleanup
}

func TestSomething(t *testing.T) {
    repo, cleanup := setupTest(t)
    defer cleanup()

    // Test code here
}
```

### Testing Errors

```go
func TestErrorHandling(t *testing.T) {
    err := someOperation()

    // Check error occurred
    if err == nil {
        t.Fatal("expected error, got nil")
    }

    // Check error message
    if !strings.Contains(err.Error(), "expected text") {
        t.Errorf("unexpected error message: %v", err)
    }

    // Check error type
    var notFoundErr *NotFoundError
    if !errors.As(err, &notFoundErr) {
        t.Errorf("expected NotFoundError, got %T", err)
    }
}
```

## Common Testing Mistakes

### ❌ WRONG: CLI test using real database
```go
func TestTaskCommand(t *testing.T) {
    database := test.GetTestDB()  // DON'T DO THIS
    // This causes test pollution and flaky tests
}
```

### ✅ CORRECT: CLI test using mock
```go
func TestTaskCommand(t *testing.T) {
    mockRepo := &MockTaskRepository{...}
    // Test command logic in isolation
}
```

### ❌ WRONG: Repository test without cleanup
```go
func TestCreate(t *testing.T) {
    database := test.GetTestDB()
    repo := NewTaskRepository(db)
    repo.Create(ctx, task)  // Leaves data in DB
    // Next test will see this data!
}
```

### ✅ CORRECT: Repository test with cleanup
```go
func TestCreate(t *testing.T) {
    database := test.GetTestDB()
    repo := NewTaskRepository(db)

    // Clean first
    database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", task.Key)

    repo.Create(ctx, task)

    // Verify and cleanup
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
}
```
