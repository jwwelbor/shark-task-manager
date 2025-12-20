# Task: Define Domain Errors

**Task ID**: E04-F09-T015
**Feature**: E04-F09 - Recommended Architecture Improvements
**Priority**: P2 (Medium)
**Estimated Effort**: 2 hours
**Status**: Todo

---

## Objective

Define typed domain errors in `internal/domain/errors.go` to replace generic error strings, enabling better error handling and more helpful user messages.

## Background

Currently, errors are returned as generic strings:
```go
return nil, fmt.Errorf("task not found with id %d", id)
return nil, fmt.Errorf("epic not found with key %s", key)
```

This makes it impossible to:
- Distinguish error types programmatically
- Handle specific errors differently
- Provide context-specific help messages
- Build error hierarchies

Domain errors solve this by defining typed, sentinel errors that can be checked with `errors.Is()`.

## Acceptance Criteria

- [ ] All domain errors defined in `internal/domain/errors.go`
- [ ] Errors follow Go error conventions (start with `Err` prefix)
- [ ] Each error has clear documentation
- [ ] Error categories organized logically
- [ ] Helper functions for common error checks provided
- [ ] Examples documented
- [ ] Compiles successfully

## Implementation Details

### File: internal/domain/errors.go

```go
package domain

import "errors"

// Domain Errors
//
// This file defines all business domain errors used throughout the application.
// Use these errors instead of creating new error strings in repositories or services.
//
// Error Handling Pattern:
//
//   // In repository
//   func (r *repo) GetByID(ctx context.Context, id int64) (*Task, error) {
//       if err == sql.ErrNoRows {
//           return nil, ErrTaskNotFound  // ← Return domain error
//       }
//       return nil, fmt.Errorf("database error: %w", err)  // ← Wrap other errors
//   }
//
//   // In CLI
//   task, err := repo.GetByID(ctx, id)
//   if errors.Is(err, domain.ErrTaskNotFound) {  // ← Check specific error
//       fmt.Println("Task not found. Use 'shark task list' to see available tasks.")
//       return nil
//   }
//   if err != nil {
//       return fmt.Errorf("failed to get task: %w", err)
//   }

// ==============================================================================
// Not Found Errors
// ==============================================================================

// ErrTaskNotFound indicates a task with the specified identifier does not exist.
//
// This error is returned when attempting to retrieve, update, or delete a task
// that cannot be found in the database.
//
// Common causes:
//   - Task was deleted
//   - Incorrect task key or ID
//   - Task belongs to different epic/feature
var ErrTaskNotFound = errors.New("task not found")

// ErrEpicNotFound indicates an epic with the specified identifier does not exist.
var ErrEpicNotFound = errors.New("epic not found")

// ErrFeatureNotFound indicates a feature with the specified identifier does not exist.
var ErrFeatureNotFound = errors.New("feature not found")

// ErrHistoryNotFound indicates no history records exist for the specified task.
var ErrHistoryNotFound = errors.New("task history not found")

// ==============================================================================
// Validation Errors
// ==============================================================================

// ErrInvalidTaskKey indicates the task key format is invalid.
//
// Task keys must follow the pattern: E[0-9]{2,}-F[0-9]{2,}-T[0-9]{3,}
// Examples: E04-F01-T001, E10-F05-T123
var ErrInvalidTaskKey = errors.New("invalid task key format")

// ErrInvalidEpicKey indicates the epic key format is invalid.
//
// Epic keys must follow the pattern: E[0-9]{2,}
// Examples: E01, E04, E100
var ErrInvalidEpicKey = errors.New("invalid epic key format")

// ErrInvalidFeatureKey indicates the feature key format is invalid.
//
// Feature keys must follow the pattern: E[0-9]{2,}-F[0-9]{2,}
// Examples: E04-F01, E10-F05
var ErrInvalidFeatureKey = errors.New("invalid feature key format")

// ErrEmptyTitle indicates a required title field is empty.
var ErrEmptyTitle = errors.New("title cannot be empty")

// ErrInvalidPriority indicates the priority value is out of valid range.
//
// Task priority must be between 1 (highest) and 10 (lowest).
var ErrInvalidPriority = errors.New("priority must be between 1 and 10")

// ErrInvalidStatus indicates an invalid status value was provided.
//
// Valid task statuses: todo, in_progress, blocked, ready_for_review, completed, archived
// Valid epic/feature statuses: draft, active, completed, archived
var ErrInvalidStatus = errors.New("invalid status value")

// ErrInvalidAgentType indicates an invalid agent type was provided.
//
// Valid agent types: frontend, backend, api, testing, devops, general
var ErrInvalidAgentType = errors.New("invalid agent type")

// ErrInvalidProgress indicates progress percentage is out of valid range.
//
// Progress must be between 0.0 and 100.0.
var ErrInvalidProgress = errors.New("progress must be between 0.0 and 100.0")

// ErrInvalidDependsOn indicates the depends_on field contains invalid JSON or task keys.
var ErrInvalidDependsOn = errors.New("depends_on must be valid JSON array of task keys")

// ==============================================================================
// Business Logic Errors
// ==============================================================================

// ErrInvalidStatusTransition indicates an illegal status transition was attempted.
//
// Examples of invalid transitions:
//   - todo → completed (must go through in_progress)
//   - completed → todo (use ReopenTask instead)
//   - archived → any other status (archived is final)
var ErrInvalidStatusTransition = errors.New("invalid status transition")

// ErrDependencyNotMet indicates a task cannot be started because its dependencies
// are not yet completed.
//
// Tasks with dependencies (depends_on field) must wait until all dependent tasks
// reach "completed" status before they can be started.
var ErrDependencyNotMet = errors.New("task dependency not satisfied")

// ErrCannotBlock indicates a task cannot be blocked in its current status.
//
// Only tasks in "todo" or "in_progress" status can be blocked.
var ErrCannotBlock = errors.New("cannot block task in current status")

// ErrAlreadyCompleted indicates an operation was attempted on a completed task
// that only applies to active tasks.
var ErrAlreadyCompleted = errors.New("task is already completed")

// ErrAlreadyArchived indicates an operation was attempted on an archived entity.
//
// Archived entities cannot be modified. They must be restored first.
var ErrAlreadyArchived = errors.New("entity is archived and cannot be modified")

// ErrBlockedReasonRequired indicates BlockTask was called without a reason.
//
// When blocking a task, a reason must always be provided to track why
// the task is blocked.
var ErrBlockedReasonRequired = errors.New("blocked reason is required when blocking task")

// ==============================================================================
// Database Constraint Errors
// ==============================================================================

// ErrDuplicateKey indicates a unique constraint violation.
//
// This typically occurs when attempting to create a task, epic, or feature
// with a key that already exists in the database.
var ErrDuplicateKey = errors.New("duplicate key constraint violation")

// ErrForeignKeyViolation indicates a foreign key constraint violation.
//
// This typically occurs when:
//   - Creating a task with non-existent feature_id
//   - Creating a feature with non-existent epic_id
var ErrForeignKeyViolation = errors.New("foreign key constraint violation")

// ErrCheckConstraintViolation indicates a check constraint was violated.
//
// This can occur when database-level validation fails, such as:
//   - Invalid status value
//   - Progress outside 0-100 range
//   - Priority outside 1-10 range
var ErrCheckConstraintViolation = errors.New("check constraint violation")

// ==============================================================================
// Concurrency Errors
// ==============================================================================

// ErrConcurrentModification indicates another process modified the entity
// between read and write operations.
//
// This can occur in concurrent scenarios where optimistic locking is used.
var ErrConcurrentModification = errors.New("entity was modified by another process")

// ErrDatabaseLocked indicates the database is locked by another process.
//
// SQLite uses file-level locking. This error can occur during concurrent writes.
// The operation should be retried after a brief delay.
var ErrDatabaseLocked = errors.New("database is locked")

// ==============================================================================
// Helper Functions
// ==============================================================================

// IsNotFoundError returns true if the error is any "not found" error.
//
// Useful for generic error handling when the specific entity type doesn't matter.
//
// Example:
//   if domain.IsNotFoundError(err) {
//       fmt.Println("The requested resource was not found")
//       return
//   }
func IsNotFoundError(err error) bool {
    return errors.Is(err, ErrTaskNotFound) ||
        errors.Is(err, ErrEpicNotFound) ||
        errors.Is(err, ErrFeatureNotFound) ||
        errors.Is(err, ErrHistoryNotFound)
}

// IsValidationError returns true if the error is a validation error.
//
// Useful for distinguishing validation errors from other error types.
//
// Example:
//   if domain.IsValidationError(err) {
//       fmt.Println("Invalid input:", err)
//       return
//   }
func IsValidationError(err error) bool {
    return errors.Is(err, ErrInvalidTaskKey) ||
        errors.Is(err, ErrInvalidEpicKey) ||
        errors.Is(err, ErrInvalidFeatureKey) ||
        errors.Is(err, ErrEmptyTitle) ||
        errors.Is(err, ErrInvalidPriority) ||
        errors.Is(err, ErrInvalidStatus) ||
        errors.Is(err, ErrInvalidAgentType) ||
        errors.Is(err, ErrInvalidProgress) ||
        errors.Is(err, ErrInvalidDependsOn)
}

// IsConstraintError returns true if the error is a database constraint violation.
//
// Example:
//   if domain.IsConstraintError(err) {
//       fmt.Println("Database constraint violation:", err)
//       return
//   }
func IsConstraintError(err error) bool {
    return errors.Is(err, ErrDuplicateKey) ||
        errors.Is(err, ErrForeignKeyViolation) ||
        errors.Is(err, ErrCheckConstraintViolation)
}

// ==============================================================================
// Error Wrapping Helpers
// ==============================================================================

// NotFoundError wraps ErrTaskNotFound with context about which task.
//
// Example:
//   return domain.TaskNotFoundError(taskKey)
//   // Returns: "task not found: E04-F01-T001"
func TaskNotFoundError(taskKey string) error {
    return fmt.Errorf("%w: %s", ErrTaskNotFound, taskKey)
}

// EpicNotFoundError wraps ErrEpicNotFound with context about which epic.
func EpicNotFoundError(epicKey string) error {
    return fmt.Errorf("%w: %s", ErrEpicNotFound, epicKey)
}

// FeatureNotFoundError wraps ErrFeatureNotFound with context about which feature.
func FeatureNotFoundError(featureKey string) error {
    return fmt.Errorf("%w: %s", ErrFeatureNotFound, featureKey)
}

// DuplicateKeyError wraps ErrDuplicateKey with context about which key.
func DuplicateKeyError(key string) error {
    return fmt.Errorf("%w: %s", ErrDuplicateKey, key)
}
```

### Error Categories

| Category | Errors | Purpose |
|----------|--------|---------|
| **Not Found** | Task, Epic, Feature, History | Entity doesn't exist |
| **Validation** | Key format, title, priority, status | Input validation failures |
| **Business Logic** | Status transitions, dependencies, blocking | Business rule violations |
| **Database** | Duplicate key, foreign key, check constraint | Database constraint violations |
| **Concurrency** | Concurrent modification, database locked | Concurrent access issues |

### Usage Examples

#### Example 1: Repository Returns Domain Error

```go
// internal/repository/sqlite/task.go
func (r *taskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)

    if err == sql.ErrNoRows {
        return nil, domain.ErrTaskNotFound  // ← Return domain error
    }
    if err != nil {
        return nil, fmt.Errorf("database error: %w", err)
    }

    return task, nil
}

// Or with context:
func (r *taskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    err := r.db.QueryRowContext(ctx, query, key).Scan(...)

    if err == sql.ErrNoRows {
        return nil, domain.TaskNotFoundError(key)  // ← Wrapped error with context
    }
    if err != nil {
        return nil, fmt.Errorf("database error: %w", err)
    }

    return task, nil
}
```

#### Example 2: CLI Handles Specific Error

```go
// internal/cli/commands/task.go
func (c *TaskCommands) ShowTask(taskKey string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    task, err := c.taskRepo.GetByKey(ctx, taskKey)

    // Check for specific error type
    if errors.Is(err, domain.ErrTaskNotFound) {
        fmt.Fprintf(os.Stderr, "Error: Task %s not found.\n", taskKey)
        fmt.Fprintf(os.Stderr, "\nUse 'shark task list' to see available tasks.\n")
        fmt.Fprintf(os.Stderr, "Or 'shark task list --feature=%s' to see tasks in this feature.\n", getFeatureKey(taskKey))
        return nil  // Exit gracefully with helpful message
    }

    if err != nil {
        return fmt.Errorf("failed to get task: %w", err)
    }

    // Display task...
    return nil
}
```

#### Example 3: Generic Not Found Handling

```go
// Generic handler for any entity
func handleEntityNotFound(err error, entityType, identifier string) {
    if domain.IsNotFoundError(err) {
        fmt.Fprintf(os.Stderr, "Error: %s '%s' not found.\n", entityType, identifier)
        return
    }

    // Other error handling...
}

// Usage
task, err := repo.GetByID(ctx, id)
if err != nil {
    handleEntityNotFound(err, "Task", taskKey)
    return
}
```

## Testing

### Unit Tests

```go
// internal/domain/errors_test.go
package domain_test

import (
    "errors"
    "testing"

    "github.com/jwwelbor/shark-task-manager/internal/domain"
    "github.com/stretchr/testify/assert"
)

func TestDomainErrors_Identity(t *testing.T) {
    // Verify errors have identity
    assert.NotNil(t, domain.ErrTaskNotFound)
    assert.NotNil(t, domain.ErrEpicNotFound)
}

func TestIsNotFoundError(t *testing.T) {
    tests := []struct {
        name string
        err  error
        want bool
    }{
        {"task not found", domain.ErrTaskNotFound, true},
        {"epic not found", domain.ErrEpicNotFound, true},
        {"feature not found", domain.ErrFeatureNotFound, true},
        {"other error", errors.New("other"), false},
        {"nil error", nil, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := domain.IsNotFoundError(tt.err)
            assert.Equal(t, tt.want, got)
        })
    }
}

func TestIsValidationError(t *testing.T) {
    assert.True(t, domain.IsValidationError(domain.ErrInvalidTaskKey))
    assert.True(t, domain.IsValidationError(domain.ErrEmptyTitle))
    assert.False(t, domain.IsValidationError(domain.ErrTaskNotFound))
}

func TestErrorWrapping(t *testing.T) {
    // Test wrapped errors can be unwrapped
    err := domain.TaskNotFoundError("E04-F01-T001")

    assert.True(t, errors.Is(err, domain.ErrTaskNotFound))
    assert.Contains(t, err.Error(), "E04-F01-T001")
}
```

## Dependencies

### Depends On
- E04-F09-T009: Create internal/domain package structure

### Blocks
- E04-F09-T016: Update repositories to return domain errors
- E04-F09-T017: Update CLI commands to handle domain errors

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Missing error types | Low | Review all error strings in codebase |
| Inconsistent error usage | Medium | Code review + linting |
| Too many error types | Low | Group related errors, use helper functions |

## Success Criteria

- ✅ All domain errors defined in errors.go
- ✅ Errors organized by category
- ✅ Helper functions provided
- ✅ Comprehensive documentation
- ✅ Examples included
- ✅ Unit tests pass
- ✅ Ready for repository integration (T016)

## Completion Checklist

- [ ] Write error definitions in `internal/domain/errors.go`
- [ ] Document each error with clear descriptions
- [ ] Implement helper functions (IsNotFoundError, IsValidationError, etc.)
- [ ] Implement wrapping helpers (TaskNotFoundError, etc.)
- [ ] Write unit tests for helper functions
- [ ] Verify compilation: `go build ./internal/domain/`
- [ ] Run tests: `go test ./internal/domain/`
- [ ] Git commit: "Define domain errors for better error handling"
