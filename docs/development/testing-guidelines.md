# Testing Guidelines for Shark Task Manager

## Core Principle: Test the Right Layer

**ONLY test with the real database in:**
1. **Repository layer tests** - Testing SQL queries work correctly
2. **Integration tests** - Testing end-to-end workflows

**For everything else, MOCK the database calls.**

## Why Mock Instead of Using the Database?

You already know what the repository will return. Don't waste time:
- Creating test database entries
- Managing test data cleanup
- Dealing with test isolation issues
- Waiting for slow database operations

Instead, mock the repository and focus on testing YOUR logic.

## Testing Pattern by Layer

### 1. Repository Layer (Use Real Database)

**Purpose:** Verify SQL queries work correctly

```go
// internal/repository/feature_repository_test.go
func TestFeatureRepository_CalculateProgress(t *testing.T) {
    // ✅ Use real database - we're testing SQL
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewFeatureRepository(db)

    // Create test data with unique keys to avoid conflicts
    suffix := fmt.Sprintf("%d", time.Now().UnixNano())
    epic := &models.Epic{
        Key: fmt.Sprintf("E-TEST-%s", suffix),
        // ...
    }
    repo.Create(epic)

    // Test the SQL query
    progress, err := repo.CalculateProgress(epic.ID)
    // assert...
}
```

**Key Points:**
- Use `test.GetTestDB()` which returns a shared singleton database
- Use **unique keys based on timestamp** to avoid conflicts between tests
- Tests can run in parallel and multiple times without conflicts
- Don't use fixed keys like "E01" - they will conflict!

### 2. Service Layer (Mock Repositories)

**Purpose:** Test business logic, not SQL

```go
// internal/services/task_service_test.go
func TestTaskService_StartTask(t *testing.T) {
    // ✅ Mock the repository - we're testing service logic
    mockRepo := &MockTaskRepository{
        GetByIDFunc: func(id int64) (*models.Task, error) {
            return &models.Task{
                ID: id,
                Status: models.TaskStatusTodo,
                // ...
            }, nil
        },
        UpdateStatusFunc: func(id int64, status models.TaskStatus) error {
            return nil
        },
    }

    service := NewTaskService(mockRepo)

    // Test business logic
    err := service.StartTask(1)
    // assert...
}
```

**Why:** You already know `GetByID` will return a task. Mock it and test your service logic.

### 3. CLI Layer (Mock Services)

**Purpose:** Test command handling, argument parsing, output formatting

```go
// internal/cli/epic_test.go
func TestEpicGetCommand(t *testing.T) {
    // ✅ Mock the service - we're testing CLI behavior
    mockService := &MockEpicService{
        GetEpicByKeyFunc: func(key string) (*models.Epic, error) {
            return &models.Epic{
                Key: key,
                Title: "Test Epic",
                ProgressPct: 75.0,
            }, nil
        },
    }

    // Test CLI command
    runner := CliRunner()
    result := runner.Invoke(epicGetCmd, []string{"E01"})

    // Assert output formatting
    assert.Contains(t, result.Output, "75.0%")
}
```

**Why:** You already know the service will return an epic. Mock it and test command handling.

### 4. Integration Tests (Use Real Database)

**Purpose:** Test the entire stack works together

```go
// tests/integration/epic_workflow_test.go
func TestEpicToFeatureWorkflow(t *testing.T) {
    // ✅ Use real database - testing end-to-end
    database := test.GetTestDB()

    // Create epic via CLI
    runner := CliRunner()
    runner.Invoke(epicCreateCmd, []string{"--key", "E-INT-001", "--title", "Integration Test"})

    // Create feature
    runner.Invoke(featureCreateCmd, []string{"--epic", "E-INT-001", "--key", "E-INT-001-F01"})

    // Verify in database
    var count int
    database.QueryRow("SELECT COUNT(*) FROM features WHERE epic_id = ?", epicID).Scan(&count)
    assert.Equal(t, 1, count)
}
```

## Common Anti-Patterns to Avoid

### ❌ BAD: Testing Service Logic with Real Database
```go
// DON'T DO THIS
func TestCalculateEpicProgress(t *testing.T) {
    db := test.GetTestDB()
    repo := NewEpicRepository(db)
    service := NewEpicService(repo)

    // Creating real database entries just to test a calculation
    epic := &models.Epic{Key: "E01", ...}
    repo.Create(epic)
    // ...50 more lines of database setup...

    progress := service.CalculateProgress("E01")
    assert.Equal(t, 75.0, progress)
}
```

**Why Bad:** Slow, fragile, and testing SQL instead of business logic.

### ✅ GOOD: Mock the Repository
```go
func TestCalculateEpicProgress(t *testing.T) {
    mockRepo := &MockEpicRepository{
        GetProgressFunc: func(key string) (float64, error) {
            return 75.0, nil
        },
    }
    service := NewEpicService(mockRepo)

    progress := service.CalculateProgress("E01")
    assert.Equal(t, 75.0, progress)
}
```

**Why Good:** Fast, focused on service logic, no database needed.

### ❌ BAD: Using Fixed Keys in Repository Tests
```go
// DON'T DO THIS
func TestFeatureRepository(t *testing.T) {
    db := test.GetTestDB()
    repo := NewFeatureRepository(db)

    feature := &models.Feature{Key: "E01-F01", ...}  // ❌ Fixed key!
    repo.Create(feature)
    // Test will fail on second run due to UNIQUE constraint
}
```

### ✅ GOOD: Use Unique Keys
```go
func TestFeatureRepository(t *testing.T) {
    db := test.GetTestDB()
    repo := NewFeatureRepository(db)

    suffix := fmt.Sprintf("%d", time.Now().UnixNano())
    feature := &models.Feature{
        Key: fmt.Sprintf("E-TEST-%s-F01", suffix),  // ✅ Unique!
        ...
    }
    repo.Create(feature)
    // Test can run multiple times and in parallel
}
```

## Test Database Pattern

The project uses a **shared singleton test database** via `test.GetTestDB()`:

```go
// internal/test/testdb.go
var (
    testDB   *sql.DB
    dbOnce   sync.Once
)

func GetTestDB() *sql.DB {
    dbOnce.Do(func() {
        testDB, _ = db.InitDB("test-shark-tasks.db")
    })
    return testDB
}
```

**Implications:**
- Database is created once and shared across all tests
- Tests must use unique keys to avoid conflicts
- Database persists between test runs (not cleaned up automatically)
- Tests can run in parallel safely if keys are unique

## Quick Reference

| Layer | Use Real DB? | What to Test | Mock What? |
|-------|--------------|--------------|------------|
| Repository | ✅ Yes | SQL queries work | Nothing |
| Service | ❌ No | Business logic | Repository |
| CLI | ❌ No | Command handling, formatting | Service |
| Integration | ✅ Yes | Full stack works | Nothing |

## Summary

**The Golden Rule:** Only use the real database if you're testing that the DATABASE works (SQL queries). For everything else, mock it and test YOUR code.

This keeps tests:
- **Fast** - No database I/O
- **Focused** - Testing one thing at a time
- **Reliable** - No test data conflicts
- **Maintainable** - Easy to understand and modify
