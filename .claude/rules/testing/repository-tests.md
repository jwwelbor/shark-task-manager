---
paths: "internal/repository/**/*_test.go"
---

# Repository Testing Patterns

This rule is loaded when working with repository test files.

## ✅ USE REAL DATABASE - MUST CLEAN UP

Repository tests verify database operations (CRUD, transactions, queries).

## Requirements

- Create test database using `test.GetTestDB()`
- Clean up test data BEFORE each test (DELETE existing records)
- Use `test.SeedTestData()` for consistent fixtures
- Never rely on data from previous tests
- Verify database constraints, triggers, indexes

## Standard Pattern

```go
func TestTaskRepository_Create(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewTaskRepository(db)

    // CRITICAL: Clean up existing data first
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

    // Seed fresh test data
    epicID, featureID := test.SeedTestData()

    // Run test
    task := &models.Task{
        Key:       "TEST-E07-F01-001",
        Title:     "Test Task",
        Status:    "todo",
        EpicID:    epicID,
        FeatureID: featureID,
    }

    err := repo.Create(ctx, task)
    if err != nil {
        t.Fatalf("Create() error = %v", err)
    }

    // Verify
    if task.ID == 0 {
        t.Error("expected ID to be set")
    }

    // Cleanup at end (optional, but good practice)
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
}
```

## Table-Driven Repository Tests

```go
func TestTaskRepository_GetByKey(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewTaskRepository(db)

    // Clean up
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

    // Seed test data
    epicID, featureID := test.SeedTestData()

    // Create test tasks
    tasks := []*models.Task{
        {Key: "TEST-E07-F01-001", Title: "Task 1", Status: "todo", EpicID: epicID, FeatureID: featureID},
        {Key: "TEST-E07-F01-002", Title: "Task 2", Status: "in_progress", EpicID: epicID, FeatureID: featureID},
    }

    for _, task := range tasks {
        if err := repo.Create(ctx, task); err != nil {
            t.Fatalf("Failed to create test task: %v", err)
        }
    }

    // Test cases
    tests := []struct {
        name    string
        key     string
        wantErr bool
    }{
        {"existing task", "TEST-E07-F01-001", false},
        {"non-existent task", "TEST-E07-F01-999", true},
        {"case insensitive", "test-e07-f01-001", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            task, err := repo.GetByKey(ctx, tt.key)
            if (err != nil) != tt.wantErr {
                t.Errorf("GetByKey() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && task == nil {
                t.Error("expected task, got nil")
            }
        })
    }

    // Cleanup
    for _, task := range tasks {
        database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
    }
}
```

## Testing Transactions

```go
func TestTaskRepository_UpdateWithTransaction(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewTaskRepository(db)

    // Clean up
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

    // Seed
    epicID, featureID := test.SeedTestData()

    // Create task
    task := &models.Task{
        Key:       "TEST-E07-F01-001",
        Title:     "Test Task",
        Status:    "todo",
        EpicID:    epicID,
        FeatureID: featureID,
    }

    if err := repo.Create(ctx, task); err != nil {
        t.Fatalf("Create() error = %v", err)
    }
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

    // Test transaction
    t.Run("successful transaction", func(t *testing.T) {
        err := repo.UpdateStatus(ctx, task.Key, "in_progress")
        if err != nil {
            t.Errorf("UpdateStatus() error = %v", err)
        }

        // Verify update
        updated, err := repo.GetByKey(ctx, task.Key)
        if err != nil {
            t.Fatalf("GetByKey() error = %v", err)
        }

        if updated.Status != "in_progress" {
            t.Errorf("expected status 'in_progress', got %s", updated.Status)
        }
    })

    t.Run("rollback on error", func(t *testing.T) {
        // This test would verify that errors cause rollback
        // and the database is left in a consistent state
    })
}
```

## Testing Database Constraints

```go
func TestTaskRepository_UniqueConstraint(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewTaskRepository(db)

    // Clean up
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'TEST-E07-F01-001'")

    // Seed
    epicID, featureID := test.SeedTestData()

    // Create first task
    task1 := &models.Task{
        Key:       "TEST-E07-F01-001",
        Title:     "Task 1",
        Status:    "todo",
        EpicID:    epicID,
        FeatureID: featureID,
    }

    err := repo.Create(ctx, task1)
    if err != nil {
        t.Fatalf("Create() error = %v", err)
    }
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task1.ID)

    // Try to create duplicate
    task2 := &models.Task{
        Key:       "TEST-E07-F01-001", // Same key
        Title:     "Task 2",
        Status:    "todo",
        EpicID:    epicID,
        FeatureID: featureID,
    }

    err = repo.Create(ctx, task2)
    if err == nil {
        t.Error("expected error for duplicate key, got nil")
    }

    // Verify error is about unique constraint
    if !strings.Contains(err.Error(), "UNIQUE") {
        t.Errorf("expected UNIQUE constraint error, got: %v", err)
    }
}
```

## Testing Progress Calculation

```go
func TestFeatureRepository_CalculateProgress(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    featureRepo := NewFeatureRepository(db)
    taskRepo := NewTaskRepository(db)

    // Clean up
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

    // Seed
    epicID, featureID := test.SeedTestData()

    // Create tasks with different statuses
    tasks := []*models.Task{
        {Key: "TEST-E07-F01-001", Status: "completed", EpicID: epicID, FeatureID: featureID},
        {Key: "TEST-E07-F01-002", Status: "completed", EpicID: epicID, FeatureID: featureID},
        {Key: "TEST-E07-F01-003", Status: "in_progress", EpicID: epicID, FeatureID: featureID},
        {Key: "TEST-E07-F01-004", Status: "todo", EpicID: epicID, FeatureID: featureID},
    }

    for _, task := range tasks {
        task.Title = "Test Task"
        if err := taskRepo.Create(ctx, task); err != nil {
            t.Fatalf("Failed to create task: %v", err)
        }
    }

    // Calculate progress
    progress, err := featureRepo.CalculateProgress(ctx, featureID)
    if err != nil {
        t.Fatalf("CalculateProgress() error = %v", err)
    }

    // Expected: 2 completed out of 4 = 50%
    expected := 50.0
    if progress != expected {
        t.Errorf("expected progress %v, got %v", expected, progress)
    }

    // Cleanup
    for _, task := range tasks {
        database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
    }
}
```

## Cleanup Best Practices

1. **Clean BEFORE test runs** (most important)
2. **Use defer for cleanup AFTER test** (good practice)
3. **Use specific WHERE clauses** (avoid deleting unrelated data)
4. **Use test prefixes** (e.g., `TEST-` for keys)

```go
func TestExample(t *testing.T) {
    database := test.GetTestDB()

    // Clean BEFORE (most important)
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

    // Create test data
    task := createTestTask()

    // Clean AFTER (good practice)
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

    // Test logic
}
```
