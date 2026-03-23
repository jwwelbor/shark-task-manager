# Error Handling

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 4 — Behavior Analysis

## Error Hierarchy

```mermaid
classDiagram
    class error {
        <<interface>>
        +Error() string
    }

    class NotFoundError {
        +Entity string
        +Key string
        +Error() string
    }

    class ValidationError {
        +Field string
        +Message string
        +Error() string
    }

    class TransitionError {
        +FromStatus string
        +ToStatus string
        +Reason string
        +Error() string
    }

    class BackwardReasonError {
        +FromStatus string
        +ToStatus string
        +Error() string
    }

    class FieldNotFoundError {
        +Field string
        +Error() string
    }

    error <|-- NotFoundError
    error <|-- ValidationError
    error <|-- TransitionError
    error <|-- BackwardReasonError
    error <|-- FieldNotFoundError
```

## Exit Code Mapping

| Code | Meaning | Error Type | Example |
|------|---------|-----------|---------|
| 0 | Success | nil | Command completed |
| 1 | Not found | NotFoundError | `shark get E99` |
| 2 | Database error | Generic error | Connection failure |
| 3 | Invalid state | TransitionError, ValidationError | Invalid workflow transition |
| 4 | Field not found | FieldNotFoundError | `--field nonexistent` |

## Error Wrapping Pattern

Errors are wrapped with context at each architectural layer:

```
Database Layer:
  "UNIQUE constraint failed: tasks.key"

Repository Layer:
  "failed to create task: UNIQUE constraint failed: tasks.key"

Service Layer:
  "failed to create task in epic E07: failed to create task: UNIQUE constraint failed"

Command Layer:
  "unable to create task: failed to create task in epic E07: ..."
```

**Pattern**: `fmt.Errorf("context description: %w", err)` using Go's error wrapping

## Sentinel Errors

Defined in `internal/models/validation.go` and `internal/repository/`:

```go
var (
    ErrInvalidEpicKey     = errors.New("invalid epic key format: must match ^E\\d{2}$")
    ErrInvalidTaskStatus  = errors.New("invalid task status")
    ErrInvalidAgentType   = errors.New("invalid agent type: cannot be empty")
    ErrInvalidPriority    = errors.New("invalid priority: must be between 1 and 10")
    ErrCircularDependency = errors.New("circular dependency detected")
    ErrSelfRelationship   = errors.New("task cannot have a relationship with itself")
)
```

## Validation Patterns

### Model Layer (Structural)

Location: `internal/models/validation.go`

- Non-empty title check
- Priority range (1-10)
- Key format regex validation
- Status non-empty check
- **Does NOT validate**: status against workflow config (that's the service layer's job)

### Service Layer (Business Rules)

Location: `internal/services/*.go`

- Status transition validity via `workflow.Service`
- Dependency satisfaction check
- Entity existence validation (epic exists, feature exists)
- Backward transition reason requirement
- Force flag reason requirement

### Database Layer (Constraints)

Location: `internal/db/db.go` (schema)

- UNIQUE constraints on entity keys
- FOREIGN KEY cascade (epic → features → tasks)
- CHECK constraints on priority, status values
- NOT NULL on required fields

## Error Response Formats

### CLI Error Output

```
Error: task not found: E07-F01-999
```

### CLI JSON Error Output (--json mode)

```json
{
  "error": "task not found: E07-F01-999"
}
```

### HTTP Error Response (planned)

```json
{
  "error": "Not Found",
  "message": "Task not found: E07-F01-999"
}
```

## Recovery Patterns

| Scenario | Pattern | Location |
|----------|---------|----------|
| Transaction failure | `defer tx.Rollback()` + explicit Commit | Service methods |
| DB connection failure | Panic (fail-fast for CLI) | `cli/db_global.go` |
| File write failure | Atomic write with O_EXCL | `fileops/writer.go` |
| Config load failure | Fall back to defaults | `config/config.go` |
| Workflow config missing | Use basic profile | `workflow/service.go` |
| Optional dependency nil | Graceful degradation (skip) | Service constructors |

## Logging Patterns

- **Verbose mode** (`--verbose`): Debug-level logging via `log.Printf`
- **Normal mode**: Errors only via `cli.Error()`
- **JSON mode**: Errors as JSON objects
- **No structured logging**: Uses standard `log` package, not structured logger

See also: [Business Logic](business-logic.md) | [Workflows](workflows.md) | [Decision Logic](decision-logic.md)
