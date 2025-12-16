# Task: Update TaskRepository to Use Context

**Task ID**: E04-F09-T002
**Feature**: E04-F09 - Recommended Architecture Improvements
**Priority**: P0 (Critical)
**Estimated Effort**: 2 hours
**Status**: Todo

---

## Objective

Update TaskRepository implementation to use context.Context in all database operations, enabling request cancellation and timeout support.

## Background

Task E04-F09-T001 added `context.Context` to method signatures. This task implements context usage in the actual database calls.

## Acceptance Criteria

- [ ] All `Query()` calls replaced with `QueryContext(ctx, ...)`
- [ ] All `QueryRow()` calls replaced with `QueryRowContext(ctx, ...)`
- [ ] All `Exec()` calls replaced with `ExecContext(ctx, ...)`
- [ ] Context cancellation checked before expensive operations
- [ ] Transaction methods use `BeginTx(ctx, nil)`
- [ ] All helper methods (queryTasks) use context
- [ ] TaskRepository compiles without errors
- [ ] No functionality changes (behavior unchanged)

## Implementation Details

### File to Modify

```
internal/repository/task_repository.go (598 lines)
```

### Database Operation Changes

#### Pattern 1: Query Operations

**Before**:
```go
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    query := `SELECT id, feature_id, ... FROM tasks WHERE id = ?`

    task := &models.Task{}
    err := r.db.QueryRow(query, id).Scan(...)  // ← No context
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("task not found with id %d", id)
    }
    return task, err
}
```

**After**:
```go
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    // Check context cancellation before expensive operation
    if err := ctx.Err(); err != nil {
        return nil, fmt.Errorf("context error: %w", err)
    }

    query := `SELECT id, feature_id, ... FROM tasks WHERE id = ?`

    task := &models.Task{}
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)  // ← With context
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("task not found with id %d", id)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get task: %w", err)
    }
    return task, nil
}
```

#### Pattern 2: Multi-Row Queries

**Before**:
```go
func (r *TaskRepository) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
    query := `SELECT ... FROM tasks WHERE feature_id = ?`
    return r.queryTasks(ctx, query, featureID)  // ← Already has ctx
}

func (r *TaskRepository) queryTasks(ctx context.Context, query string, args ...interface{}) ([]*models.Task, error) {
    rows, err := r.db.Query(query, args...)  // ← No context
    if err != nil {
        return nil, fmt.Errorf("failed to query tasks: %w", err)
    }
    defer rows.Close()

    var tasks []*models.Task
    for rows.Next() {
        task := &models.Task{}
        err := rows.Scan(...)
        if err != nil {
            return nil, fmt.Errorf("failed to scan task: %w", err)
        }
        tasks = append(tasks, task)
    }
    return tasks, rows.Err()
}
```

**After**:
```go
func (r *TaskRepository) queryTasks(ctx context.Context, query string, args ...interface{}) ([]*models.Task, error) {
    // Check context before query
    if err := ctx.Err(); err != nil {
        return nil, fmt.Errorf("context error: %w", err)
    }

    rows, err := r.db.QueryContext(ctx, query, args...)  // ← With context
    if err != nil {
        return nil, fmt.Errorf("failed to query tasks: %w", err)
    }
    defer rows.Close()

    var tasks []*models.Task
    for rows.Next() {
        // Check context during iteration (for long result sets)
        if err := ctx.Err(); err != nil {
            return nil, fmt.Errorf("context cancelled during iteration: %w", err)
        }

        task := &models.Task{}
        err := rows.Scan(...)
        if err != nil {
            return nil, fmt.Errorf("failed to scan task: %w", err)
        }
        tasks = append(tasks, task)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating tasks: %w", err)
    }

    return tasks, nil
}
```

#### Pattern 3: Exec Operations

**Before**:
```go
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
    if err := task.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    query := `INSERT INTO tasks (...) VALUES (?, ?, ...)`

    result, err := r.db.Exec(query, task.FeatureID, ...)  // ← No context
    if err != nil {
        return fmt.Errorf("failed to create task: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get last insert id: %w", err)
    }

    task.ID = id
    return nil
}
```

**After**:
```go
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
    if err := task.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // Check context before write operation
    if err := ctx.Err(); err != nil {
        return fmt.Errorf("context error: %w", err)
    }

    query := `INSERT INTO tasks (...) VALUES (?, ?, ...)`

    result, err := r.db.ExecContext(ctx, query, task.FeatureID, ...)  // ← With context
    if err != nil {
        return fmt.Errorf("failed to create task: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get last insert id: %w", err)
    }

    task.ID = id
    return nil
}
```

#### Pattern 4: Transactions

**Before**:
```go
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
    // Start transaction
    tx, err := r.db.BeginTx()  // ← No context
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Get current status
    var currentStatus string
    err = tx.QueryRow("SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)  // ← No context
    if err == sql.ErrNoRows {
        return fmt.Errorf("task not found with id %d", taskID)
    }

    // Update status
    _, err = tx.Exec("UPDATE tasks SET status = ? WHERE id = ?", newStatus, taskID)  // ← No context
    if err != nil {
        return fmt.Errorf("failed to update task status: %w", err)
    }

    // Create history
    _, err = tx.Exec("INSERT INTO task_history (...) VALUES (?, ?, ?, ?)", taskID, currentStatus, newStatus, agent)  // ← No context
    if err != nil {
        return fmt.Errorf("failed to create history record: %w", err)
    }

    return tx.Commit()
}
```

**After**:
```go
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
    // Check context before transaction
    if err := ctx.Err(); err != nil {
        return fmt.Errorf("context error: %w", err)
    }

    // Start transaction with context
    tx, err := r.db.BeginTx(ctx, nil)  // ← With context
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Get current status - use context
    var currentStatus string
    err = tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
    if err == sql.ErrNoRows {
        return fmt.Errorf("task not found with id %d", taskID)
    }
    if err != nil {
        return fmt.Errorf("failed to get current task status: %w", err)
    }

    // Update status - use context
    _, err = tx.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", newStatus, taskID)
    if err != nil {
        return fmt.Errorf("failed to update task status: %w", err)
    }

    // Create history - use context
    _, err = tx.ExecContext(ctx, "INSERT INTO task_history (...) VALUES (?, ?, ?, ?)", taskID, currentStatus, newStatus, agent)
    if err != nil {
        return fmt.Errorf("failed to create history record: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

### Methods to Update

All 17 methods in TaskRepository:

1. ✅ `Create()` - Exec → ExecContext
2. ✅ `GetByID()` - QueryRow → QueryRowContext
3. ✅ `GetByKey()` - QueryRow → QueryRowContext
4. ✅ `ListByFeature()` - Uses queryTasks (update queryTasks)
5. ✅ `ListByEpic()` - Uses queryTasks (update queryTasks)
6. ✅ `FilterByStatus()` - Uses queryTasks (update queryTasks)
7. ✅ `FilterByAgentType()` - Uses queryTasks (update queryTasks)
8. ✅ `FilterCombined()` - Uses queryTasks (update queryTasks)
9. ✅ `List()` - Uses queryTasks (update queryTasks)
10. ✅ `Update()` - Exec → ExecContext
11. ✅ `UpdateStatus()` - Transaction with BeginTx, QueryRow, Exec
12. ✅ `BlockTask()` - Transaction with BeginTx, QueryRow, Exec
13. ✅ `UnblockTask()` - Transaction with BeginTx, QueryRow, Exec
14. ✅ `ReopenTask()` - Transaction with BeginTx, QueryRow, Exec
15. ✅ `Delete()` - Exec → ExecContext
16. ✅ `GetStatusBreakdown()` - Query → QueryContext
17. ✅ `queryTasks()` - Helper method, Query → QueryContext

### Context Check Pattern

Add context checks before expensive operations:

```go
// Check context before database operation
if err := ctx.Err(); err != nil {
    return nil, fmt.Errorf("context error: %w", err)
}
```

**When to check**:
- Before starting a transaction
- Before single-row queries (optional, QueryRowContext checks internally)
- Before multi-row queries
- During long result set iterations
- Before expensive calculations

**When NOT to check**:
- After every line (too verbose)
- In tight loops (only check periodically)
- When database driver already checks (QueryRowContext does this)

### DB Interface Update

Update the DB struct to support context:

```go
// internal/repository/repository.go

type DB struct {
    *sql.DB
}

func NewDB(db *sql.DB) *DB {
    return &DB{db}
}

// Add context-aware transaction method
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
    return db.DB.BeginTx(ctx, opts)
}

// Delegate to sql.DB for context-aware methods
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    return db.DB.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
    return db.DB.QueryRowContext(ctx, query, args...)
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    return db.DB.ExecContext(ctx, query, args...)
}
```

## Testing

### Manual Testing

```bash
# Verify compilation
go build ./internal/repository/task_repository.go

# Run specific repository tests (will fail until T008 updates tests)
go test ./internal/repository/ -run TestTask
```

### Context Cancellation Test (Manual)

Create temporary test to verify context handling:

```go
func TestTaskRepository_ContextCancellation(t *testing.T) {
    db := testdb.NewTestDB(t)
    defer db.Close()

    repo := NewTaskRepository(db)

    // Create cancelled context
    ctx, cancel := context.WithCancel(context.Background())
    cancel()

    // Should return context.Canceled error
    task, err := repo.GetByID(ctx, 1)

    assert.Nil(t, task)
    assert.Error(t, err)
    // Verify it's a context error
    assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context"))
}
```

## Dependencies

### Depends On
- E04-F09-T001: Add context.Context to repository interfaces

### Blocks
- E04-F09-T006: Update HTTP handlers to use request context
- E04-F09-T007: Update CLI commands to use context with timeout
- E04-F09-T008: Update all repository tests to use context

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Missing database call updates | Low | Grep for `r.db.Query(` patterns |
| Context check overhead | Very Low | Overhead is negligible (<1μs) |
| Transaction deadlock with context timeout | Low | Use reasonable timeouts (30s+) |

## Success Criteria

- ✅ All Query() → QueryContext()
- ✅ All QueryRow() → QueryRowContext()
- ✅ All Exec() → ExecContext()
- ✅ All BeginTx() → BeginTx(ctx, nil)
- ✅ Context checks added before expensive operations
- ✅ TaskRepository compiles without errors
- ✅ Manual context cancellation test passes

## Completion Checklist

- [ ] Update repository.go DB struct with context methods
- [ ] Update Create() method
- [ ] Update GetByID() method
- [ ] Update GetByKey() method
- [ ] Update Update() method
- [ ] Update Delete() method
- [ ] Update UpdateStatus() method (transaction)
- [ ] Update BlockTask() method (transaction)
- [ ] Update UnblockTask() method (transaction)
- [ ] Update ReopenTask() method (transaction)
- [ ] Update GetStatusBreakdown() method
- [ ] Update queryTasks() helper method
- [ ] Add context checks before expensive operations
- [ ] Verify compilation: `go build ./internal/repository/`
- [ ] Run manual context cancellation test
- [ ] Git commit: "Add context support to TaskRepository"
