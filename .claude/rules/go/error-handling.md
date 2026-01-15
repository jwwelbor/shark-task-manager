---
paths: "{internal,cmd}/**/*.go"
---

# Error Handling Patterns

This rule is loaded when working with Go source files.

## General Principles

- Always return errors explicitly
- Use `fmt.Errorf("context: %w", err)` for wrapping
- Never ignore errors with `_`; if unused, return them
- Provide context at each error handling point

## Error Wrapping

### Basic Wrapping
```go
result, err := someOperation()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Multi-Level Wrapping
```go
func Level3() error {
    if err := database.Query(); err != nil {
        return fmt.Errorf("query failed: %w", err)
    }
    return nil
}

func Level2() error {
    if err := Level3(); err != nil {
        return fmt.Errorf("level3 operation failed: %w", err)
    }
    return nil
}

func Level1() error {
    if err := Level2(); err != nil {
        return fmt.Errorf("level2 operation failed: %w", err)
    }
    return nil
}

// Error chain: "level2 operation failed: level3 operation failed: query failed: <original error>"
```

## Custom Error Types

### Define Custom Errors for Domain Logic
```go
// NotFoundError indicates an entity was not found
type NotFoundError struct {
    Entity string
    Key    string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.Entity, e.Key)
}

// ValidationError indicates validation failure
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// ConflictError indicates a conflict (e.g., duplicate key)
type ConflictError struct {
    Resource string
    Key      string
}

func (e *ConflictError) Error() string {
    return fmt.Sprintf("conflict: %s with key %s already exists", e.Resource, e.Key)
}
```

### Using Custom Errors
```go
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    // ... query database ...

    if err == sql.ErrNoRows {
        return nil, &NotFoundError{Entity: "task", Key: key}
    }

    if err != nil {
        return nil, fmt.Errorf("failed to get task: %w", err)
    }

    return task, nil
}
```

### Checking Error Types
```go
task, err := repo.GetByKey(ctx, "E07-F01-001")
if err != nil {
    var notFoundErr *NotFoundError
    if errors.As(err, &notFoundErr) {
        // Handle not found specifically
        return fmt.Errorf("task does not exist: %s", notFoundErr.Key)
    }

    // Handle other errors
    return fmt.Errorf("failed to get task: %w", err)
}
```

## Exit Codes

Shark uses consistent exit codes for CLI commands:
- **0**: Success
- **1**: Not found (entity doesn't exist)
- **2**: Database error
- **3**: Invalid state (e.g., trying to complete a task that's not in_progress)

### Exit Code Pattern
```go
func runCommand(cmd *cobra.Command, args []string) error {
    task, err := repo.GetByKey(ctx, args[0])
    if err != nil {
        var notFoundErr *repository.NotFoundError
        if errors.As(err, &notFoundErr) {
            cli.Error(fmt.Sprintf("Task not found: %s", args[0]))
            os.Exit(1) // Exit code 1 for not found
        }

        cli.Error(fmt.Sprintf("Database error: %v", err))
        os.Exit(2) // Exit code 2 for database error
    }

    if task.Status != "in_progress" {
        cli.Error("Task must be in_progress to complete")
        os.Exit(3) // Exit code 3 for invalid state
    }

    // ... rest of command logic ...
    return nil
}
```

## Error Context

### Add Context at Each Layer
```go
// Repository layer - technical context
func (r *TaskRepository) UpdateStatus(ctx context.Context, key string, status string) error {
    if err := r.db.Exec(query, status, key); err != nil {
        return fmt.Errorf("failed to update task status in database: %w", err)
    }
    return nil
}

// Service layer - business context
func (s *TaskService) CompleteTask(ctx context.Context, key string) error {
    if err := s.repo.UpdateStatus(ctx, key, "completed"); err != nil {
        return fmt.Errorf("failed to complete task %s: %w", key, err)
    }
    return nil
}

// Command layer - user-facing context
func runCompleteCommand(cmd *cobra.Command, args []string) error {
    if err := service.CompleteTask(cmd.Context(), args[0]); err != nil {
        return fmt.Errorf("unable to mark task as complete: %w", err)
    }
    return nil
}
```

## Sentinel Errors

For common, expected errors, define sentinel errors:

```go
var (
    ErrTaskNotFound     = errors.New("task not found")
    ErrInvalidStatus    = errors.New("invalid status transition")
    ErrDuplicateKey     = errors.New("duplicate key")
    ErrMissingRequired  = errors.New("missing required field")
)

// Usage
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    // ... query ...

    if err == sql.ErrNoRows {
        return nil, ErrTaskNotFound
    }

    return task, nil
}

// Checking sentinel errors
task, err := repo.GetByKey(ctx, key)
if errors.Is(err, repository.ErrTaskNotFound) {
    // Handle not found
}
```

## Error Aggregation

When collecting multiple errors:

```go
type ValidationErrors struct {
    Errors []error
}

func (ve *ValidationErrors) Error() string {
    var messages []string
    for _, err := range ve.Errors {
        messages = append(messages, err.Error())
    }
    return strings.Join(messages, "; ")
}

func (ve *ValidationErrors) Add(err error) {
    if err != nil {
        ve.Errors = append(ve.Errors, err)
    }
}

func (ve *ValidationErrors) HasErrors() bool {
    return len(ve.Errors) > 0
}

// Usage
func (t *Task) Validate() error {
    errs := &ValidationErrors{}

    if t.Title == "" {
        errs.Add(errors.New("title is required"))
    }

    if t.Priority < 1 || t.Priority > 10 {
        errs.Add(errors.New("priority must be between 1 and 10"))
    }

    if errs.HasErrors() {
        return errs
    }

    return nil
}
```

## Panic vs Error

**Prefer errors over panics**:
- Use `panic` only for unrecoverable errors (e.g., programming mistakes)
- Use `error` returns for all expected and unexpected runtime conditions

```go
// Good - returns error
func Divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// Bad - panics (unless this is truly unrecoverable)
func Divide(a, b int) int {
    if b == 0 {
        panic("division by zero")
    }
    return a / b
}
```

## Deferred Error Handling

When using defer with error-returning functions:

```go
func processFile(path string) (err error) {
    f, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }

    // Capture close error
    defer func() {
        if closeErr := f.Close(); closeErr != nil {
            // If we already have an error, wrap both
            if err != nil {
                err = fmt.Errorf("%v (also failed to close file: %w)", err, closeErr)
            } else {
                err = fmt.Errorf("failed to close file: %w", closeErr)
            }
        }
    }()

    // ... process file ...
    return nil
}
```
