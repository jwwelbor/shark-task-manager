---
paths: "{internal,cmd}/**/*.go"
---

# Go Coding Patterns

This rule is loaded when working with Go source files.

## Error Handling

- Always return errors explicitly; use `fmt.Errorf("context: %w", err)` for wrapping
- Exit codes: 0 (success), 1 (not found), 2 (DB error), 3 (invalid state)
- Never ignore errors with `_`; if unused, return them

### Error Wrapping Example
```go
result, err := someOperation()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Custom Error Types
```go
// Define custom error types when needed
type NotFoundError struct {
    Entity string
    Key    string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.Entity, e.Key)
}
```

## Database Transactions

- Use `tx.Rollback()` with defer for all multi-statement operations
- Atomic status updates wrap multiple queries in a single transaction
- `task_history` records created automatically with triggers for status changes

### Transaction Pattern
```go
func (r *Repository) UpdateWithTransaction(ctx context.Context, id int64) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Rollback if not committed

    // Perform multiple operations
    if err := r.operation1(ctx, tx, id); err != nil {
        return err
    }

    if err := r.operation2(ctx, tx, id); err != nil {
        return err
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

## Validation

Two levels of validation:
- **Model layer** (`internal/models/validation.go`): Basic structural validation only (non-empty checks, range checks). Models must NOT import workflow or config packages.
- **Service layer**: Workflow-aware validation (valid statuses, transitions) via `workflow.Service`. This is where business rule validation belongs.
- Database constraints (CHECK, FOREIGN KEY) provide additional safety

### Model Validation Pattern (Structural Only)
```go
func (t *Task) Validate() error {
    if strings.TrimSpace(t.Title) == "" {
        return errors.New("title cannot be empty")
    }

    if t.Priority < 1 || t.Priority > 10 {
        return errors.New("priority must be between 1 and 10")
    }

    if strings.TrimSpace(string(t.Status)) == "" {
        return errors.New("status cannot be empty")
    }

    // NOTE: Status validity is checked at the service layer via
    // workflow.Service.ValidateStatus(), NOT here. Do NOT hardcode
    // status lists in models.
    return nil
}
```

### Service Validation Pattern (Business Rules)
```go
func (s *TaskService) StartTask(ctx context.Context, key string) error {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return err
    }

    // Workflow-aware validation belongs in services
    if err := s.workflowSvc.ValidateStatus(string(task.Status)); err != nil {
        return err
    }

    if !s.workflowSvc.IsValidTransition(string(task.Status), "in_progress") {
        return fmt.Errorf("cannot start task in status %s", task.Status)
    }

    return s.repo.UpdateStatus(ctx, key, "in_progress")
}
```

## Repository Pattern

### Repository Structure
```go
type TaskRepository struct {
    db *db.DB
}

func NewTaskRepository(db *db.DB) *TaskRepository {
    return &TaskRepository{db: db}
}

// CRUD methods
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
    // Implementation
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    // Implementation
}

func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
    // Implementation
}

func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
    // Implementation
}

// Query methods
func (r *TaskRepository) GetByStatus(ctx context.Context, status string) ([]*models.Task, error) {
    // Implementation
}

func (r *TaskRepository) List(ctx context.Context, filters map[string]interface{}) ([]*models.Task, error) {
    // Implementation
}
```

### Using Prepared Statements
```go
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    query := `
        SELECT id, key, title, status, created_at, updated_at
        FROM tasks
        WHERE key = ?
    `

    task := &models.Task{}
    err := r.db.QueryRowContext(ctx, query, key).Scan(
        &task.ID,
        &task.Key,
        &task.Title,
        &task.Status,
        &task.CreatedAt,
        &task.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, &NotFoundError{Entity: "task", Key: key}
    }

    if err != nil {
        return nil, fmt.Errorf("failed to get task: %w", err)
    }

    return task, nil
}
```

## Dependency Injection

- Services created with injected repositories: `NewTaskService(repo, workflowSvc)`
- Repositories created with injected DB: `NewTaskRepository(db *DB)`
- No DI framework; constructor injection is explicit and compile-safe
- Wiring happens at entry points (CLI init, HTTP server setup), not in command handlers

### Constructor Pattern
```go
// Service constructor - receives repository dependencies
func NewTaskService(repo *TaskRepository, workflowSvc *workflow.Service) *TaskService {
    return &TaskService{
        repo:        repo,
        workflowSvc: workflowSvc,
    }
}

// Repository constructor - receives DB dependency
func NewTaskRepository(db *db.DB) *TaskRepository {
    return &TaskRepository{db: db}
}

// Wiring at entry point (not in individual commands)
repo := repository.NewTaskRepository(db)
workflowSvc := workflow.NewService(projectRoot)
taskSvc := services.NewTaskService(repo, workflowSvc)
```

### Layering Rules
- **CLI commands** call services, never repositories directly
- **Services** call repositories and other services
- **Repositories** call the database only
- Business logic belongs in services, not commands or repositories

## Context Usage

- Always pass `context.Context` as first parameter
- Use context for cancellation, timeouts, and request-scoped values
- Never store context in structs

### Context Pattern
```go
func (r *Repository) ProcessTask(ctx context.Context, taskID int64) error {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Use context with database operations
    task, err := r.GetByID(ctx, taskID)
    if err != nil {
        return err
    }

    // Process task
    return r.Update(ctx, task)
}
```

## Interface Design

- Keep interfaces small and focused
- Define interfaces at the point of use (consumer side)
- Accept interfaces, return structs

### Interface Pattern
```go
// Define interface near where it's used
type TaskGetter interface {
    GetByID(ctx context.Context, id int64) (*models.Task, error)
}

// Function accepts interface
func ProcessTask(ctx context.Context, getter TaskGetter, id int64) error {
    task, err := getter.GetByID(ctx, id)
    if err != nil {
        return err
    }

    // Process task
    return nil
}

// Concrete type implements interface
type TaskRepository struct {
    db *db.DB
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    // Implementation
}
```

## Testing Patterns

See `.claude/rules/testing/architecture.md` and `.claude/rules/go/testing.md` for Go testing patterns.
