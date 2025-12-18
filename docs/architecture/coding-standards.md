# Executive Summary (Code-Only)

**Project**: Shark Task Manager
**Stack**: Go 1.23.4, SQLite3, Cobra CLI, Viper config
**Tools**: gofmt, go vet, golangci-lint v1.55.2
**Priorities**: Correctness > Maintainability > Security > Performance

This document defines code-level standards for the Shark Task Manager codebase. All rules are enforceable locally through linters, formatters, and testing. These standards apply to Go code and focus on consistency, correctness, and secure-by-construction patterns.

## Universal Coding Standards

### Code Style and Formatting

**Rule**: Use `gofmt` for all Go code formatting
**Why**: Enforces consistent formatting across the codebase, eliminates style debates
**How to enforce**: Run `make fmt` before committing; integrate into pre-commit hook
**Example**:
```bash
# Before committing
make fmt
```

**Rule**: All packages must have package-level documentation
**Why**: Improves code discoverability and understanding
**How to enforce**: Review during PR; add to PR checklist
**Example**:
```go
✅ GOOD:
// Package repository provides data access layer with context support.
//
// All repository methods accept context.Context as the first parameter to support:
// - Request cancellation
// - Timeout management
// - Distributed tracing
package repository

❌ BAD:
package repository
```

### Naming Conventions

**Rule**: Use MixedCaps (camelCase/PascalCase) for all identifiers, never underscores
**Why**: Go convention; improves readability and consistency
**How to enforce**: golangci-lint with stylecheck enabled
**Example**:
```go
✅ GOOD:
type TaskRepository struct {}
func getByID() {}

❌ BAD:
type task_repository struct {}
func get_by_id() {}
```

**Rule**: Test files must use `_test.go` suffix
**Why**: Go convention; enables proper test discovery
**How to enforce**: Filename pattern check in PR review
**Example**:
```go
✅ GOOD: validator_test.go
❌ BAD: validator-test.go, test_validator.go
```

**Rule**: Interfaces should be named with -er suffix when describing behavior
**Why**: Go convention; makes interfaces self-documenting
**How to enforce**: Manual review; golangci-lint stylecheck
**Example**:
```go
✅ GOOD:
type TaskRepository interface {
    Create(ctx context.Context, task *Task) error
}

❌ BAD:
type TaskRepositoryInterface interface {
    Create(ctx context.Context, task *Task) error
}
```

### Error Handling

**Rule**: All errors must be wrapped with context using `fmt.Errorf` with `%w` verb
**Why**: Preserves error chain for debugging and error inspection
**How to enforce**: golangci-lint with errorlint enabled
**Example**:
```go
✅ GOOD:
if err := task.Validate(); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

❌ BAD:
if err := task.Validate(); err != nil {
    return err
}
```

**Rule**: Define sentinel errors as package-level variables with `Err` prefix
**Why**: Enables error checking with `errors.Is()`; clear error taxonomy
**How to enforce**: Manual code review
**Example**:
```go
✅ GOOD:
var (
    ErrInvalidEpicKey    = errors.New("invalid epic key format: must match ^E\\d{2}$")
    ErrInvalidTaskStatus = errors.New("invalid task status: must be todo, in_progress, blocked, ready_for_review, completed, or archived")
)

❌ BAD:
func validateKey(key string) error {
    return errors.New("invalid key")
}
```

**Rule**: Never ignore errors; use `_` only when documented why error is impossible
**Why**: Prevents silent failures and bugs
**How to enforce**: golangci-lint with errcheck enabled
**Example**:
```go
✅ GOOD:
id, err := result.LastInsertId()
if err != nil {
    return fmt.Errorf("failed to get last insert id: %w", err)
}

❌ BAD:
id, _ := result.LastInsertId()
```

### Context Usage

**Rule**: All repository methods must accept `context.Context` as first parameter
**Why**: Enables request cancellation, timeouts, and distributed tracing
**How to enforce**: Manual review; consistent pattern in repository layer
**Example**:
```go
✅ GOOD:
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    // Use ctx in database calls
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)
}

❌ BAD:
func (r *TaskRepository) GetByID(id int64) (*models.Task, error) {
    // No context support
}
```

**Rule**: CLI commands must create context with timeout
**Why**: Prevents hanging operations; enforces operation boundaries
**How to enforce**: Code review; check all CLI command entry points
**Example**:
```go
✅ GOOD:
func runTaskGet(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    task, err := repo.GetByKey(ctx, taskKey)
    // ...
}

❌ BAD:
func runTaskGet(cmd *cobra.Command, args []string) error {
    task, err := repo.GetByKey(context.Background(), taskKey)
    // ...
}
```

## Language-Specific Standards

### Go Standards

**Rule**: Use Go 1.23.4+ features; avoid deprecated APIs
**Why**: Leverage latest improvements; future-proof codebase
**How to enforce**: Documented in go.mod; reviewed during updates

**Rule**: Prefer table-driven tests over individual test functions
**Why**: Reduces code duplication; easier to add test cases
**How to enforce**: Code review; see existing test patterns
**Example**:
```go
✅ GOOD:
func TestValidatePattern_MissingRequiredCaptureGroups_Epic(t *testing.T) {
    tests := []struct {
        name    string
        pattern string
        wantErr bool
    }{
        {
            name:    "has number - valid",
            pattern: `^E(?P<number>\d{2})-.*$`,
            wantErr: false,
        },
        {
            name:    "missing all required groups - invalid",
            pattern: `^E\d{2}-[a-z]+$`,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePattern(tt.pattern, "epic")
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

❌ BAD:
func TestValidatePattern_ValidNumber(t *testing.T) {
    err := ValidatePattern(`^E(?P<number>\d{2})-.*$`, "epic")
    assert.NoError(t, err)
}

func TestValidatePattern_MissingGroups(t *testing.T) {
    err := ValidatePattern(`^E\d{2}-[a-z]+$`, "epic")
    assert.Error(t, err)
}
```

**Rule**: Use `testify/assert` for test assertions; `testify/require` for fatal assertions
**Why**: Provides clear, readable assertions with helpful error messages
**How to enforce**: Import pattern check in tests
**Example**:
```go
✅ GOOD:
require.NoError(t, err, "Setup must succeed")
assert.Equal(t, expected, actual, "Values should match")

❌ BAD:
if err != nil {
    t.Fatalf("Setup failed: %v", err)
}
if expected != actual {
    t.Errorf("Expected %v, got %v", expected, actual)
}
```

**Rule**: Use typed constants and enums; avoid magic strings
**Why**: Compile-time safety; prevents typos; improves autocomplete
**How to enforce**: Code review; look for string literals in comparisons
**Example**:
```go
✅ GOOD:
type TaskStatus string

const (
    TaskStatusTodo          TaskStatus = "todo"
    TaskStatusInProgress    TaskStatus = "in_progress"
    TaskStatusCompleted     TaskStatus = "completed"
)

if task.Status == TaskStatusInProgress {
    // ...
}

❌ BAD:
if task.Status == "in_progress" {
    // ...
}
```

**Rule**: Struct tags must use consistent naming: `json`, `db`
**Why**: Ensures proper serialization and database mapping
**How to enforce**: Code review; check struct definitions
**Example**:
```go
✅ GOOD:
type Task struct {
    ID        int64      `json:"id" db:"id"`
    Title     string     `json:"title" db:"title"`
    Status    TaskStatus `json:"status" db:"status"`
}

❌ BAD:
type Task struct {
    ID        int64
    Title     string
    Status    TaskStatus
}
```

**Rule**: Use pointer receivers for methods that modify state or are on large structs
**Why**: Avoids copying; allows mutation; consistent with Go conventions
**How to enforce**: golangci-lint with gocritic enabled
**Example**:
```go
✅ GOOD:
func (r *TaskRepository) Create(ctx context.Context, task *Task) error {
    // Repository method uses pointer receiver
}

func (t *Task) Validate() error {
    // Validation on struct type - doesn't modify but consistent
}

❌ BAD:
func (r TaskRepository) Create(ctx context.Context, task *Task) error {
    // Copies entire repository struct unnecessarily
}
```

**Rule**: Exported functions and types must have doc comments starting with the name
**Why**: Generates proper godoc documentation; aids IDE tooltips
**How to enforce**: golangci-lint with golint/revive enabled
**Example**:
```go
✅ GOOD:
// ValidateTaskKey validates the task key format
func ValidateTaskKey(key string) error {
    // ...
}

❌ BAD:
// Validates the task key format
func ValidateTaskKey(key string) error {
    // ...
}
```

## Framework-Specific Standards

### Cobra CLI Framework

**Rule**: Command structure must follow: verb + noun pattern
**Why**: Consistent CLI UX; predictable command discovery
**How to enforce**: Review command definitions
**Example**:
```go
✅ GOOD:
shark task list
shark task create
shark epic get E04

❌ BAD:
shark list-tasks
shark create_task
```

**Rule**: All commands must support `--json` flag for machine-readable output
**Why**: Enables automation and AI agent integration
**How to enforce**: Check all command RunE functions
**Example**:
```go
✅ GOOD:
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(tasks)
}

❌ BAD:
fmt.Printf("Tasks: %v\n", tasks)
```

**Rule**: Use `cobra.ExactArgs(N)` for commands requiring specific arg count
**Why**: Clear error messages; prevents misuse
**How to enforce**: Review command Args field
**Example**:
```go
✅ GOOD:
var taskGetCmd = &cobra.Command{
    Use:  "get <task-key>",
    Args: cobra.ExactArgs(1),
    RunE: runTaskGet,
}

❌ BAD:
var taskGetCmd = &cobra.Command{
    Use:  "get <task-key>",
    RunE: runTaskGet,
}
```

### SQLite Database

**Rule**: All database operations must use prepared statements or parameterized queries
**Why**: Prevents SQL injection; improves performance
**How to enforce**: Code review; grep for string concatenation in SQL
**Example**:
```go
✅ GOOD:
query := `SELECT * FROM tasks WHERE key = ?`
err := r.db.QueryRowContext(ctx, query, taskKey).Scan(...)

❌ BAD:
query := fmt.Sprintf("SELECT * FROM tasks WHERE key = '%s'", taskKey)
err := r.db.QueryRowContext(ctx, query).Scan(...)
```

**Rule**: Use transactions for multi-step operations
**Why**: Ensures atomicity; prevents partial state
**How to enforce**: Review operations that modify multiple tables
**Example**:
```go
✅ GOOD:
tx, err := r.db.BeginTxContext(ctx)
if err != nil {
    return fmt.Errorf("failed to begin transaction: %w", err)
}
defer tx.Rollback()

// Multiple operations
_, err = tx.ExecContext(ctx, updateQuery, args...)
_, err = tx.ExecContext(ctx, historyQuery, historyArgs...)

if err := tx.Commit(); err != nil {
    return fmt.Errorf("failed to commit: %w", err)
}

❌ BAD:
_, err = r.db.ExecContext(ctx, updateQuery, args...)
_, err = r.db.ExecContext(ctx, historyQuery, historyArgs...)
```

**Rule**: Always defer `Close()` and `Rollback()` immediately after resource acquisition
**Why**: Prevents resource leaks; ensures cleanup
**How to enforce**: Code review; look for missing defer statements
**Example**:
```go
✅ GOOD:
database, err := db.InitDB(dbPath)
if err != nil {
    return err
}
defer database.Close()

tx, err := r.db.BeginTxContext(ctx)
if err != nil {
    return err
}
defer tx.Rollback()

❌ BAD:
database, err := db.InitDB(dbPath)
if err != nil {
    return err
}
// Missing defer - potential resource leak
```

## Testing Standards (Code-Level)

### Test Organization

**Rule**: Test files must be in the same package as code under test
**Why**: Enables testing of unexported functions; maintains package boundaries
**How to enforce**: File organization review
**Example**:
```
✅ GOOD:
internal/patterns/validator.go
internal/patterns/validator_test.go

❌ BAD:
internal/patterns/validator.go
tests/patterns/validator_test.go
```

**Rule**: Use subtests with `t.Run()` for table-driven tests
**Why**: Provides clear test output; enables selective test execution
**How to enforce**: Code review; check test patterns
**Example**:
```go
✅ GOOD:
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        err := ValidatePattern(tt.pattern, "epic")
        // assertions
    })
}

❌ BAD:
for _, tt := range tests {
    err := ValidatePattern(tt.pattern, "epic")
    // assertions - no subtest isolation
}
```

### Test Coverage

**Rule**: All public functions must have test coverage
**Why**: Ensures correctness; prevents regressions
**How to enforce**: `make test-coverage` reports; PR review

**Rule**: Test both success and error paths
**Why**: Validates error handling; ensures robustness
**How to enforce**: Code review; coverage analysis
**Example**:
```go
✅ GOOD:
func TestCreate_Success(t *testing.T) {
    // Test successful creation
}

func TestCreate_ValidationError(t *testing.T) {
    // Test validation failure
}

func TestCreate_DatabaseError(t *testing.T) {
    // Test database failure
}

❌ BAD:
func TestCreate(t *testing.T) {
    // Only tests success path
}
```

### Integration Tests

**Rule**: Integration tests must use isolated test databases
**Why**: Prevents test interference; enables parallel execution
**How to enforce**: Check test setup; use `testdb.go` helper
**Example**:
```go
✅ GOOD:
func TestTaskRepository_Integration(t *testing.T) {
    db := testdb.Setup(t)
    defer testdb.Teardown(t, db)
    // Test uses isolated database
}

❌ BAD:
func TestTaskRepository_Integration(t *testing.T) {
    db, _ := sql.Open("sqlite3", "shark-tasks.db")
    // Uses production database
}
```

**Rule**: Clean up test databases after test completion
**Why**: Prevents disk space issues; ensures clean state
**How to enforce**: Check for cleanup in test teardown
**Example**:
```go
✅ GOOD:
// Makefile
test:
	@rm -f internal/repository/test-shark-tasks.db*
	@go test -v ./...
```

## Secure Coding Standards

### Input Validation

**Rule**: Validate all input at API boundaries using explicit validation functions
**Why**: Prevents injection attacks; ensures data integrity
**How to enforce**: Code review; check all user input paths
**Example**:
```go
✅ GOOD:
func (t *Task) Validate() error {
    if err := ValidateTaskKey(t.Key); err != nil {
        return err
    }
    if t.Title == "" {
        return ErrEmptyTitle
    }
    if err := ValidateTaskStatus(string(t.Status)); err != nil {
        return err
    }
    return nil
}

❌ BAD:
func (r *TaskRepository) Create(ctx context.Context, task *Task) error {
    // No validation - trusts input
    query := `INSERT INTO tasks (...) VALUES (...)`
    _, err := r.db.ExecContext(ctx, query, task.Key, task.Title, ...)
}
```

**Rule**: Use regex validation for structured keys and identifiers
**Why**: Prevents malformed input; enforces format constraints
**How to enforce**: Check validation.go patterns
**Example**:
```go
✅ GOOD:
var taskKeyPattern = regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`)

func ValidateTaskKey(key string) error {
    if !taskKeyPattern.MatchString(key) {
        return fmt.Errorf("%w: got %q", ErrInvalidTaskKey, key)
    }
    return nil
}

❌ BAD:
func ValidateTaskKey(key string) error {
    if len(key) < 10 {
        return errors.New("key too short")
    }
    return nil
}
```

### SQL Security

**Rule**: Never concatenate user input into SQL queries
**Why**: Prevents SQL injection attacks
**How to enforce**: Code review; grep for fmt.Sprintf with SQL
**Example**:
```go
✅ GOOD:
query := `SELECT * FROM tasks WHERE status = ?`
rows, err := r.db.QueryContext(ctx, query, status)

❌ BAD:
query := fmt.Sprintf("SELECT * FROM tasks WHERE status = '%s'", status)
rows, err := r.db.QueryContext(ctx, query)
```

**Rule**: Use type-safe enums for status values; validate before database operations
**Why**: Prevents invalid state transitions; type safety
**How to enforce**: Check for enum validation in repository methods
**Example**:
```go
✅ GOOD:
func ValidateTaskStatus(status string) error {
    validStatuses := map[string]bool{
        "todo":             true,
        "in_progress":      true,
        "blocked":          true,
        "ready_for_review": true,
        "completed":        true,
        "archived":         true,
    }
    if !validStatuses[status] {
        return fmt.Errorf("%w: got %q", ErrInvalidTaskStatus, status)
    }
    return nil
}

❌ BAD:
// No validation - accepts any string as status
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, status string) error {
    query := `UPDATE tasks SET status = ? WHERE id = ?`
    _, err := r.db.ExecContext(ctx, query, status, taskID)
}
```

### File System Security

**Rule**: Validate and sanitize file paths before file operations
**Why**: Prevents path traversal attacks
**How to enforce**: Code review; check all file path handling
**Example**:
```go
✅ GOOD:
// Validate path is within expected directory
func validateFilePath(path string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }

    baseDir, _ := filepath.Abs("docs/plan")
    if !strings.HasPrefix(absPath, baseDir) {
        return errors.New("path outside allowed directory")
    }
    return nil
}

❌ BAD:
func writeTaskFile(path string, content []byte) error {
    // No validation - vulnerable to path traversal
    return os.WriteFile(path, content, 0644)
}
```

## Review Rubric (YAML)

```yaml
rules:
  - id: "GO-STYLE-001"
    title: "Code formatted with gofmt"
    appliesTo:
      - "go"
    severity: "major"
    why: "Enforces consistent formatting"
    how_to_check: "Run make fmt"
    example_good: "make fmt"
    example_bad: "Manual formatting"

  - id: "GO-ERROR-001"
    title: "Errors wrapped with context"
    appliesTo:
      - "go"
    severity: "major"
    why: "Preserves error chain for debugging"
    how_to_check: "Check for fmt.Errorf with %w"
    example_good: 'fmt.Errorf("failed: %w", err)'
    example_bad: "return err"

  - id: "GO-ERROR-002"
    title: "No ignored errors"
    appliesTo:
      - "go"
    severity: "critical"
    why: "Prevents silent failures"
    how_to_check: "golangci-lint errcheck"
    example_good: "if err != nil { return err }"
    example_bad: "_, _ = result.LastInsertId()"

  - id: "GO-CTX-001"
    title: "Repository methods accept context"
    appliesTo:
      - "go"
    severity: "major"
    why: "Enables cancellation and timeouts"
    how_to_check: "Check first parameter is context.Context"
    example_good: "func (r *Repo) Get(ctx context.Context, id int64)"
    example_bad: "func (r *Repo) Get(id int64)"

  - id: "GO-CTX-002"
    title: "CLI commands create context with timeout"
    appliesTo:
      - "go"
    severity: "major"
    why: "Prevents hanging operations"
    how_to_check: "Check for context.WithTimeout in RunE"
    example_good: "ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)"
    example_bad: "ctx := context.Background()"

  - id: "GO-NAMING-001"
    title: "Use MixedCaps not underscores"
    appliesTo:
      - "go"
    severity: "major"
    why: "Go convention"
    how_to_check: "golangci-lint stylecheck"
    example_good: "taskRepository"
    example_bad: "task_repository"

  - id: "GO-TEST-001"
    title: "Use table-driven tests"
    appliesTo:
      - "go"
    severity: "minor"
    why: "Reduces duplication; easier to extend"
    how_to_check: "Review test structure"
    example_good: "tests := []struct{...}; for _, tt := range tests { t.Run(tt.name, ...) }"
    example_bad: "Multiple individual test functions"

  - id: "GO-TEST-002"
    title: "Use testify/assert and testify/require"
    appliesTo:
      - "go"
    severity: "minor"
    why: "Clear, readable assertions"
    how_to_check: "Check test imports"
    example_good: "assert.Equal(t, expected, actual)"
    example_bad: "if expected != actual { t.Error(...) }"

  - id: "GO-CONST-001"
    title: "Use typed constants for enums"
    appliesTo:
      - "go"
    severity: "major"
    why: "Compile-time safety; prevents typos"
    how_to_check: "Look for string literals in comparisons"
    example_good: "if status == TaskStatusInProgress"
    example_bad: 'if status == "in_progress"'

  - id: "GO-STRUCT-001"
    title: "Struct tags include json and db"
    appliesTo:
      - "go"
    severity: "major"
    why: "Proper serialization and DB mapping"
    how_to_check: "Review struct definitions"
    example_good: "ID int64 `json:\"id\" db:\"id\"`"
    example_bad: "ID int64"

  - id: "GO-DOC-001"
    title: "Exported items have doc comments"
    appliesTo:
      - "go"
    severity: "major"
    why: "Generates proper godoc"
    how_to_check: "golangci-lint golint/revive"
    example_good: "// ValidateTaskKey validates the task key format"
    example_bad: "// validates the key"

  - id: "CLI-CMD-001"
    title: "Commands support --json flag"
    appliesTo:
      - "go"
    severity: "major"
    why: "Enables automation"
    how_to_check: "Check RunE for cli.GlobalConfig.JSON"
    example_good: "if cli.GlobalConfig.JSON { return cli.OutputJSON(...) }"
    example_bad: "fmt.Printf(...)"

  - id: "CLI-CMD-002"
    title: "Use cobra.ExactArgs for fixed arg count"
    appliesTo:
      - "go"
    severity: "minor"
    why: "Clear error messages"
    how_to_check: "Review Command Args field"
    example_good: "Args: cobra.ExactArgs(1)"
    example_bad: "Args: nil"

  - id: "SQL-SEC-001"
    title: "Use parameterized queries"
    appliesTo:
      - "go"
    severity: "critical"
    why: "Prevents SQL injection"
    how_to_check: "Grep for fmt.Sprintf with SQL keywords"
    example_good: "query := `SELECT * FROM tasks WHERE key = ?`"
    example_bad: "query := fmt.Sprintf(\"SELECT * FROM tasks WHERE key = '%s'\", key)"

  - id: "SQL-TXN-001"
    title: "Use transactions for multi-step operations"
    appliesTo:
      - "go"
    severity: "major"
    why: "Ensures atomicity"
    how_to_check: "Review operations modifying multiple tables"
    example_good: "tx, err := db.BeginTxContext(ctx); defer tx.Rollback()"
    example_bad: "Multiple separate ExecContext calls"

  - id: "SQL-RES-001"
    title: "Defer Close() and Rollback() after acquisition"
    appliesTo:
      - "go"
    severity: "major"
    why: "Prevents resource leaks"
    how_to_check: "Check for defer after resource creation"
    example_good: "db, err := sql.Open(...); defer db.Close()"
    example_bad: "db, err := sql.Open(...) // no defer"

  - id: "VAL-INPUT-001"
    title: "Validate all input at API boundaries"
    appliesTo:
      - "go"
    severity: "critical"
    why: "Prevents injection; ensures integrity"
    how_to_check: "Check for Validate() calls"
    example_good: "if err := task.Validate(); err != nil { return err }"
    example_bad: "No validation before database operation"

  - id: "VAL-REGEX-001"
    title: "Use regex for structured key validation"
    appliesTo:
      - "go"
    severity: "major"
    why: "Enforces format constraints"
    how_to_check: "Check validation.go patterns"
    example_good: "taskKeyPattern.MatchString(key)"
    example_bad: "len(key) > 10"

  - id: "VAL-ENUM-001"
    title: "Validate enums before database operations"
    appliesTo:
      - "go"
    severity: "major"
    why: "Type safety; prevents invalid states"
    how_to_check: "Check for ValidateTaskStatus calls"
    example_good: "ValidateTaskStatus(string(status))"
    example_bad: "UPDATE tasks SET status = ? (no validation)"

  - id: "SEC-PATH-001"
    title: "Validate file paths before operations"
    appliesTo:
      - "go"
    severity: "critical"
    why: "Prevents path traversal"
    how_to_check: "Review file path handling"
    example_good: "strings.HasPrefix(absPath, baseDir)"
    example_bad: "os.WriteFile(userPath, content, 0644)"
```

## Reference Configs (Local Dev)

### .editorconfig

```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[*.{yml,yaml,json,md}]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab
```

### golangci-lint.yaml

```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable:
    - errcheck      # Check for unchecked errors
    - gosimple      # Simplify code
    - govet         # Go vet
    - ineffassign   # Detect ineffectual assignments
    - staticcheck   # Advanced static analysis
    - typecheck     # Type checking
    - unused        # Detect unused code
    - stylecheck    # Style checks (replaces golint)
    - errorlint     # Error wrapping checks
    - gocritic      # Opinionated checks
    - revive        # Drop-in replacement for golint

linters-settings:
  errcheck:
    check-blank: true
  govet:
    check-shadowing: true
  stylecheck:
    checks: ["all"]
  errorlint:
    errorf: true
```

### Makefile Integration

```makefile
.PHONY: fmt vet lint test test-coverage

# Format code
fmt:
	@go fmt ./...

# Run go vet
vet:
	@go vet ./...

# Lint code
lint:
	@golangci-lint run

# Run tests
test:
	@rm -f internal/repository/test-shark-tasks.db*
	@go test -v ./...

# Coverage
test-coverage:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
```

## Adoption Guide & Checklists

### PR Checklist

Before submitting a pull request:

- [ ] Run `make fmt` to format code
- [ ] Run `make vet` to check for issues
- [ ] Run `make lint` to check linter rules
- [ ] Run `make test` and ensure all tests pass
- [ ] Add tests for new functionality
- [ ] Update documentation if needed
- [ ] Verify errors are wrapped with context
- [ ] Check that all exported items have doc comments
- [ ] Ensure SQL queries use parameterized statements
- [ ] Validate input at API boundaries

### New Module Checklist

When creating a new module/package:

- [ ] Add package-level documentation
- [ ] Define sentinel errors as package-level `Err*` variables
- [ ] Create `*_test.go` file in same package
- [ ] Use table-driven tests for multiple test cases
- [ ] Define typed constants for enums
- [ ] Document all exported functions and types
- [ ] Add validation functions for custom types
- [ ] Use context for long-running operations

### API Endpoint/Repository Method Checklist

When adding new repository methods:

- [ ] Accept `context.Context` as first parameter
- [ ] Use parameterized queries (no string concatenation)
- [ ] Wrap errors with context using `fmt.Errorf` with `%w`
- [ ] Use transactions for multi-step operations
- [ ] Defer `Close()` and `Rollback()` immediately
- [ ] Validate input using explicit validation functions
- [ ] Add integration tests with isolated test database
- [ ] Test both success and error paths

### CLI Command Checklist

When adding new CLI commands:

- [ ] Use verb + noun naming pattern
- [ ] Support `--json` flag for machine-readable output
- [ ] Create context with timeout in RunE function
- [ ] Use `cobra.ExactArgs(N)` if fixed arg count required
- [ ] Provide clear usage examples in Long description
- [ ] Add appropriate flags with default values
- [ ] Handle errors gracefully with clear messages
- [ ] Exit with appropriate exit codes (0=success, 1=not found, 2=db error, 3=invalid state)

## One-Page Quickstart

### Files to Add

1. `.editorconfig` - Editor configuration for consistent formatting
2. `golangci-lint.yaml` - Linter configuration (optional but recommended)

### Commands to Run

```bash
# Install dependencies
make install

# Format code (always run first)
make fmt

# Check for issues
make vet

# Run linter (requires golangci-lint)
make lint

# Run tests
make test

# Generate coverage report
make test-coverage
```

### Adoption Order

1. **Format** - Run `make fmt` on entire codebase
2. **Lint** - Install golangci-lint and run `make lint`
3. **Refactor** - Fix critical issues (SQL injection, unchecked errors)
4. **Tests** - Add missing test coverage for critical paths
5. **Document** - Add doc comments to all exported items
6. **Integrate** - Add checks to PR workflow

### Quick Reference

| Category | Rule | Command |
|----------|------|---------|
| Formatting | Use gofmt | `make fmt` |
| Linting | Run golangci-lint | `make lint` |
| Testing | Table-driven tests | `make test` |
| Errors | Wrap with context | `fmt.Errorf("...: %w", err)` |
| Context | First param in repos | `func (r *Repo) Get(ctx context.Context, ...)` |
| SQL | Parameterized queries | `db.QueryContext(ctx, "SELECT * FROM tasks WHERE id = ?", id)` |
| Validation | At API boundaries | `if err := task.Validate(); err != nil { return err }` |
| Documentation | Exported items | `// ValidateTaskKey validates...` |
