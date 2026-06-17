# Security Standards

Use these standards when handling external input, SQL, filesystem paths, or output boundaries.

## Input Validation

**Rule**: Validate all external input at API and CLI boundaries.

```go
func (t *Task) Validate() error {
    if err := ValidateTaskKey(t.Key); err != nil {
        return err
    }
    if t.Title == "" {
        return ErrEmptyTitle
    }
    return ValidateTaskStatus(string(t.Status))
}
```

**Rule**: Use explicit validation functions for structured keys and identifiers.

```go
var taskKeyPattern = regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`)

func ValidateTaskKey(key string) error {
    if !taskKeyPattern.MatchString(key) {
        return fmt.Errorf("%w: got %q", ErrInvalidTaskKey, key)
    }
    return nil
}
```

**Rule**: Validate enums before persistence or state transitions.

## SQL Security

**Rule**: Never concatenate user input into SQL queries.

```go
query := `SELECT * FROM tasks WHERE status = ?`
rows, err := r.db.QueryContext(ctx, query, status)
```

Avoid:

```go
query := fmt.Sprintf("SELECT * FROM tasks WHERE status = '%s'", status)
```

**Rule**: Prefer typed constants for constrained database values.

```go
type TaskStatus string

const TaskStatusInProgress TaskStatus = "in_progress"
```

## Filesystem Security

**Rule**: Validate and sanitize file paths before file operations.

```go
func validatePlanPath(path string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }

    baseDir, err := filepath.Abs("docs/plan")
    if err != nil {
        return err
    }

    rel, err := filepath.Rel(baseDir, absPath)
    if err != nil {
        return err
    }
    if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
        return errors.New("path outside docs/plan")
    }
    return nil
}
```

**Rule**: Use atomic file operations for generated project artifacts when overwrite or race behavior matters.

## Error Exposure

**Rule**: Preserve detailed errors internally, but map them deliberately at the CLI/API boundary.

Examples:

- CLI can show concise user-facing messages and return exit codes.
- HTTP should avoid leaking internal paths, SQL, or implementation details in responses.

## Security Checklist

- [ ] External input is validated at the boundary.
- [ ] Structured identifiers use explicit validators.
- [ ] SQL uses parameters only.
- [ ] Status and enum values use typed constants and validation.
- [ ] File paths are checked against an allowed base path.
- [ ] Internal error details are not leaked through API responses.
