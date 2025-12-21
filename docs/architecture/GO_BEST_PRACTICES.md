# Go Best Practices Guide for Shark Task Manager

**Quick Reference**: Common Go patterns and idioms used in this project

---

## Table of Contents

1. [Test Organization](#test-organization)
2. [Dependency Injection in Go](#dependency-injection-in-go)
3. [SOLID Principles in Go](#solid-principles-in-go)
4. [Common Go Idioms](#common-go-idioms)
5. [Recommended Improvements](#recommended-improvements)

---

## Test Organization

### ✅ Go's Standard: Tests in Same Package

**This is correct and intentional!**

```
internal/repository/
├── task_repository.go              # Production code
├── task_repository_test.go         # Tests (same package)
├── task_lifecycle_test.go          # More tests (same package)
├── epic_repository.go              # Production code
└── epic_feature_integration_test.go # Tests (same package)
```

### Why Go Does This

1. **Compiler excludes test files from builds**
   - `*_test.go` files are never compiled into binaries
   - Zero risk of test code in production
   - No binary size increase

2. **Test private (unexported) functions**
   ```go
   // task_repository.go
   func (r *TaskRepository) helperFunction() { ... }  // unexported

   // task_repository_test.go (same package "repository")
   func TestHelperFunction(t *testing.T) {
       r.helperFunction()  // ✅ Can test private functions
   }
   ```

3. **Easy navigation**
   - Test is right next to code it tests
   - No jumping between `src/` and `test/` directories

### Black-Box Testing (When Needed)

If you want to test only public APIs (black-box testing):

```go
// task_repository_test.go
package repository_test  // ← Different package

import (
    "testing"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

func TestPublicAPI(t *testing.T) {
    // Can only access exported (public) functions
}
```

### Test File Naming

| Pattern | Purpose | Example |
|---------|---------|---------|
| `*_test.go` | Unit tests | `task_repository_test.go` |
| `*_integration_test.go` | Integration tests | `epic_feature_integration_test.go` |
| `*_benchmark_test.go` | Benchmarks | `query_performance_benchmark_test.go` |

### Build Tags for Integration Tests

Separate expensive tests:

```go
//go:build integration
// +build integration

package repository

func TestExpensiveIntegration(t *testing.T) { ... }
```

Run only integration tests:
```bash
go test -tags=integration ./...
```

---

## Dependency Injection in Go

### ✅ You're Already Using DI!

Go uses **constructor injection** instead of DI frameworks:

```go
// Define dependency
type TaskRepository struct {
    db *DB  // ← Dependency injected via constructor
}

// Constructor injects dependency
func NewTaskRepository(db *DB) *TaskRepository {
    return &TaskRepository{db: db}  // ← Injection happens here
}

// Usage in main.go
func main() {
    database := db.InitDB("shark-tasks.db")
    dbWrapper := repository.NewDB(database)

    // Inject dependencies
    taskRepo := repository.NewTaskRepository(dbWrapper)
    epicRepo := repository.NewEpicRepository(dbWrapper)

    // Use repositories
    tasks, err := taskRepo.ListByFeature(ctx, featureID)
}
```

### Comparison: Go vs Java/C#

| Feature | Java/C# (Spring/.NET) | Go (Manual DI) |
|---------|-----------------------|----------------|
| **DI Framework** | Required (@Autowired, etc.) | Not needed |
| **Configuration** | Annotations or XML | Constructor functions |
| **Compile-time Safety** | Partial | Full ✅ |
| **Explicit Dependencies** | Hidden in framework | Visible in code ✅ |
| **Learning Curve** | Medium-High | Low ✅ |

### Interface-Based DI (Recommended Enhancement)

**Current** (concrete dependency):
```go
type TaskRepository struct {
    db *DB  // ← Concrete type
}
```

**Recommended** (interface dependency):
```go
// Define interface
type Database interface {
    Query(query string, args ...interface{}) (*sql.Rows, error)
    QueryRow(query string, args ...interface{}) *sql.Row
    Exec(query string, args ...interface{}) (sql.Result, error)
    BeginTx() (*sql.Tx, error)
}

// Depend on interface
type TaskRepository struct {
    db Database  // ← Interface, can be mocked
}

// Concrete implementation
type sqliteDB struct {
    *sql.DB
}

func (db *sqliteDB) Query(...) { ... }
func (db *sqliteDB) QueryRow(...) { ... }
// ...

// Constructor returns interface
func NewTaskRepository(db Database) *TaskRepository {
    return &TaskRepository{db: db}
}
```

**Benefits**:
- Easy to mock for testing
- Can swap implementations (SQLite → PostgreSQL)
- Satisfies Dependency Inversion Principle

### Testing with Mocks

With interfaces, you can create test mocks:

```go
// Mock database for testing
type mockDB struct{}

func (m *mockDB) Query(...) (*sql.Rows, error) {
    // Return test data
    return nil, nil
}

func (m *mockDB) QueryRow(...) *sql.Row {
    // Return test data
    return nil
}

// Test uses mock
func TestTaskRepository(t *testing.T) {
    mockDB := &mockDB{}
    repo := NewTaskRepository(mockDB)  // ← Inject mock

    // Test without real database
    task, err := repo.GetByID(ctx, 1)
    // ...
}
```

### Popular DI Frameworks (Optional)

For large projects, consider:

1. **wire** (Google)
   - Compile-time code generation
   - No runtime reflection
   - Type-safe

   ```go
   //go:generate wire
   //+build wireinject

   func InitializeApp() (*App, error) {
       wire.Build(
           db.NewDB,
           repository.NewTaskRepository,
           repository.NewEpicRepository,
           NewApp,
       )
       return nil, nil
   }
   ```

2. **dig** (Uber)
   - Runtime DI container
   - Reflection-based
   - More flexible

   ```go
   container := dig.New()
   container.Provide(db.NewDB)
   container.Provide(repository.NewTaskRepository)
   container.Invoke(func(repo *TaskRepository) {
       // Use repo
   })
   ```

**Recommendation for this project**: Stick with manual DI. It's simple, explicit, and perfect for this size.

---

## SOLID Principles in Go

### S - Single Responsibility Principle ✅

**Your Code**: Each component has one job

```go
// ✅ GOOD: TaskRepository only handles task data access
type TaskRepository struct {
    db *DB
}

func (r *TaskRepository) Create(task *Task) error { ... }
func (r *TaskRepository) GetByID(id int64) (*Task, error) { ... }

// ✅ GOOD: Task model handles domain logic + validation
type Task struct {
    ID     int64
    Title  string
    Status TaskStatus
}

func (t *Task) Validate() error { ... }  // Validation belongs in model
```

**Anti-pattern** (not in your code):
```go
// ❌ BAD: God object doing too much
type TaskManager struct {
    // Handles DB, validation, HTTP, CLI, email, logging...
}
```

### O - Open/Closed Principle ✅

**Your Code**: Open for extension, closed for modification

```go
// ✅ Can add new repository methods without changing existing ones
func (r *TaskRepository) GetByKey(key string) (*Task, error) {
    // New method added without modifying Create(), GetByID(), etc.
}

// ✅ Can add new models without modifying existing code
type NewEntity struct { ... }  // Doesn't affect Task, Epic, Feature
```

**Enhancement**: Use interfaces for better extensibility

```go
// Define interface
type TaskRepository interface {
    Create(task *Task) error
    GetByID(id int64) (*Task, error)
}

// Can have multiple implementations
type SQLiteTaskRepository struct { ... }  // Current
type PostgresTaskRepository struct { ... }  // Future
type InMemoryTaskRepository struct { ... }  // Testing
```

### L - Liskov Substitution Principle 🔸

**Current**: Not applicable (no interfaces yet)

**With Interfaces**: Any implementation must be substitutable

```go
type TaskRepository interface {
    Create(ctx context.Context, task *Task) error
    GetByID(ctx context.Context, id int64) (*Task, error)
}

// Both implementations satisfy the contract
type SQLiteTaskRepo struct { ... }
type PostgresTaskRepo struct { ... }

// Can swap implementations
func ProcessTasks(repo TaskRepository) {
    // Works with ANY TaskRepository implementation
    task, _ := repo.GetByID(ctx, 1)
}
```

### I - Interface Segregation Principle 🔸

**Current**: Not applicable (no interfaces yet)

**Recommendation**: Keep interfaces small and focused

```go
// ✅ GOOD: Focused interfaces
type TaskReader interface {
    GetByID(ctx context.Context, id int64) (*Task, error)
    List(ctx context.Context) ([]*Task, error)
}

type TaskWriter interface {
    Create(ctx context.Context, task *Task) error
    Update(ctx context.Context, task *Task) error
    Delete(ctx context.Context, id int64) error
}

// Client only depends on what it needs
func ReadTasks(reader TaskReader) {
    // Doesn't need TaskWriter methods
}

// ❌ BAD: Fat interface
type TaskRepository interface {
    // 20+ methods - forces clients to depend on unused methods
    Create(...)
    Update(...)
    Delete(...)
    GetByID(...)
    GetByKey(...)
    ListByFeature(...)
    // ...
}
```

### D - Dependency Inversion Principle 🔸

**Current**: Depends on concrete types

```go
type TaskRepository struct {
    db *DB  // ← Concrete dependency
}
```

**Recommended**: Depend on abstractions (interfaces)

```go
// High-level module defines what it needs
type Database interface {
    Query(...) (*sql.Rows, error)
    QueryRow(...) *sql.Row
    Exec(...) (sql.Result, error)
}

// High-level module depends on abstraction
type TaskRepository struct {
    db Database  // ← Interface, not concrete type
}

// Low-level module implements interface
type sqliteDB struct {
    *sql.DB
}

func (db *sqliteDB) Query(...) { ... }
```

**Benefits**:
- TaskRepository doesn't know about SQLite specifics
- Can test with mock database
- Can swap database implementations

---

## Common Go Idioms

### 1. Error Handling ✅

**Your Code**: Explicit error checking everywhere

```go
// ✅ GOOD: Check errors explicitly
task, err := repo.GetByID(id)
if err != nil {
    return fmt.Errorf("failed to get task: %w", err)  // Wrap error
}

// ✅ GOOD: Use %w for error wrapping
return fmt.Errorf("database error: %w", err)
```

**Go Philosophy**: Errors are values, handle them explicitly

```go
// ✅ GOOD: Return error as value
func GetTask(id int64) (*Task, error) {
    // ...
    return nil, errors.New("task not found")
}

// ❌ BAD: Panic for regular errors (only use for programming errors)
func GetTask(id int64) *Task {
    // ...
    panic("task not found")  // ❌ Don't do this
}
```

### 2. Pointers for Modification ✅

**Your Code**: Receiver determines mutability

```go
// ✅ Value receiver: Doesn't modify
func (t Task) String() string {
    return t.Title
}

// ✅ Pointer receiver: Can modify
func (t *Task) Validate() error {
    if t.Title == "" {
        return ErrEmptyTitle
    }
    return nil
}
```

**Rule of Thumb**:
- Pointer receiver: If method modifies OR struct is large
- Value receiver: If method only reads AND struct is small

### 3. Constructor Functions ✅

**Your Code**: `New*` functions for initialization

```go
// ✅ GOOD: Constructor pattern
func NewTaskRepository(db *DB) *TaskRepository {
    return &TaskRepository{db: db}
}

// Usage
repo := NewTaskRepository(db)
```

**Benefits**:
- Ensures proper initialization
- Can add initialization logic
- Clear entry point

### 4. Defer for Cleanup ✅

**Your Code**: Deferred cleanup

```go
// ✅ GOOD: Defer Close
rows, err := r.db.Query(query, args...)
if err != nil {
    return nil, err
}
defer rows.Close()  // ← Ensures cleanup even if error

// ✅ GOOD: Defer Rollback
tx, err := r.db.BeginTx()
if err != nil {
    return err
}
defer tx.Rollback()  // ← Rollback if not committed

// Do work...
return tx.Commit()  // ← Commit overwrites Rollback (no-op)
```

### 5. Nil Slices vs Empty Slices

```go
// Both are valid, but have different meanings

// Nil slice: "No data, not initialized"
var tasks []*Task  // nil slice

// Empty slice: "Initialized, zero results"
tasks := []*Task{}  // empty slice

// In your code:
func (r *TaskRepository) List() ([]*Task, error) {
    var tasks []*Task  // nil slice
    // If no results, returns nil
    // If results found, appends and returns slice
    return tasks, nil
}

// Better: Always return empty slice, not nil
func (r *TaskRepository) List() ([]*Task, error) {
    tasks := make([]*Task, 0)  // Empty slice, not nil
    // Always returns valid slice (empty or populated)
    return tasks, nil
}
```

### 6. Context for Cancellation (Recommended)

**Current**: No context

```go
func (r *TaskRepository) GetByID(id int64) (*Task, error) { ... }
```

**Recommended**: Add context

```go
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*Task, error) {
    // Check context cancellation
    if ctx.Err() != nil {
        return nil, ctx.Err()
    }

    // Use context in database calls
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)
    return task, err
}

// Usage
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

task, err := repo.GetByID(ctx, 1)
```

**Benefits**:
- Request cancellation
- Timeout management
- Trace propagation

---

## Recommended Improvements

### Priority 1: Add Context ⭐⭐⭐

**Why**: Standard Go practice for I/O operations

```go
// Before
func (r *TaskRepository) GetByID(id int64) (*Task, error)

// After
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*Task, error)
```

**Implementation**:
1. Add `context.Context` as first parameter to all repository methods
2. Use `QueryContext()`, `QueryRowContext()`, `ExecContext()` instead of `Query()`, `QueryRow()`, `Exec()`
3. Check `ctx.Err()` before expensive operations

### Priority 2: Define Repository Interfaces ⭐⭐⭐

**Why**: Better testability, clearer contracts

```go
// internal/domain/repositories.go
package domain

type TaskRepository interface {
    Create(ctx context.Context, task *models.Task) error
    GetByID(ctx context.Context, id int64) (*models.Task, error)
    Update(ctx context.Context, task *models.Task) error
    Delete(ctx context.Context, id int64) error
    // ...
}

type EpicRepository interface { ... }
type FeatureRepository interface { ... }
```

**Implementation**:
1. Create `internal/domain/` package
2. Define interfaces for all repositories
3. Update repository constructors to return interfaces
4. Create mock implementations for testing

### Priority 3: Domain-Specific Errors ⭐⭐

**Why**: Better error handling, clearer semantics

```go
// internal/domain/errors.go
package domain

import "errors"

var (
    ErrTaskNotFound     = errors.New("task not found")
    ErrEpicNotFound     = errors.New("epic not found")
    ErrFeatureNotFound  = errors.New("feature not found")
    ErrInvalidStatus    = errors.New("invalid status transition")
    ErrDependencyNotMet = errors.New("dependency not satisfied")
)

// Repository returns domain errors
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*Task, error) {
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)
    if err == sql.ErrNoRows {
        return nil, domain.ErrTaskNotFound  // ← Domain error
    }
    return nil, fmt.Errorf("database error: %w", err)
}

// CLI handles specific errors
task, err := repo.GetByID(ctx, id)
if errors.Is(err, domain.ErrTaskNotFound) {
    fmt.Println("Task not found. Use 'shark task list' to see available tasks.")
    return
}
```

### Priority 4: Configuration Management ⭐

**Why**: No hardcoded values, environment-specific config

```go
// Current: Hardcoded
database, err := db.InitDB("shark-tasks.db")
port := "8080"

// Recommended: Environment variables
import "os"

dbPath := os.Getenv("DB_PATH")
if dbPath == "" {
    dbPath = "shark-tasks.db"  // Default
}

port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}

database, err := db.InitDB(dbPath)
```

**Or use Viper** (already in dependencies):
```go
import "github.com/spf13/viper"

func loadConfig() {
    viper.SetDefault("database.path", "shark-tasks.db")
    viper.SetDefault("server.port", "8080")
    viper.SetEnvPrefix("SHARK")
    viper.AutomaticEnv()

    dbPath := viper.GetString("database.path")
    port := viper.GetString("server.port")
}
```

---

## Quick Reference Summary

### Your Architecture: What's Right ✅

1. ✅ **Test organization** - Following Go convention perfectly
2. ✅ **Dependency injection** - Using constructor injection correctly
3. ✅ **Single Responsibility** - Clear separation of concerns
4. ✅ **Error handling** - Explicit checks everywhere
5. ✅ **Project structure** - Standard Go layout
6. ✅ **Repository pattern** - Clean data access layer
7. ✅ **Transactions** - Proper atomicity

### Low-Hanging Fruit Improvements 🔸

1. 🔸 **Add context.Context** to all I/O operations
2. 🔸 **Define repository interfaces** for better testing
3. 🔸 **Domain-specific errors** for clearer error handling
4. 🔸 **Extract configuration** to environment variables

### Not Issues ✅

1. ✅ Tests in same directory as code - **This is correct!**
2. ✅ No DI framework - **Go uses constructor injection!**
3. ✅ Manual SQL - **Better performance and control!**
4. ✅ Simple architecture - **Appropriate for project size!**

---

## Conclusion

Your Go code follows best practices and demonstrates good understanding of Go idioms. The recommended improvements (context, interfaces, errors) will make it more idiomatic and maintainable, but the current architecture is already solid and production-ready.

**Key Takeaway**: Go is different from Java/C#. What looks "missing" (DI framework, ORM, separate test directories) is actually intentional Go design philosophy favoring simplicity and explicitness.
