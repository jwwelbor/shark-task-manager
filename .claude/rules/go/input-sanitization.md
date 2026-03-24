# Input Sanitization Patterns

This rule documents how Shark Task Manager sanitizes and validates all input before it reaches the database or business logic. Security-sensitive for onboarding.

## Overview

Shark uses a **layered sanitization approach**:

1. **Structural validation** at the model layer (`internal/models/validation.go`) — rejects malformed data
2. **Business validation** at the service layer via `workflow.Service` — enforces domain rules
3. **Parameterized queries** at the repository layer — prevents SQL injection
4. **Allowlist enums** for string fields that must be one of a fixed set of values

---

## Layer 1: Model Layer — Structural Validation

The `internal/models/validation.go` file provides reusable validators for all entity fields. These are called from `Validate()` methods on model structs before any data reaches services or repositories.

### Key Format Validation (Regex Allowlist)

Entity keys are restricted to exact patterns using compiled regular expressions:

```go
var (
    epicKeyPattern    = regexp.MustCompile(`^E\d{2}$`)          // E07, E99
    featureKeyPattern = regexp.MustCompile(`^E\d{2}-F\d{2}$`)   // E07-F01
    taskKeyPattern    = regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`) // T-E07-F01-001
    bugKeyPattern     = regexp.MustCompile(`^B\d{3}$`)           // B001
    changeCardKeyPattern = regexp.MustCompile(`^CC-\d{3}$`)      // CC-001
)

func ValidateEpicKey(key string) error {
    if !epicKeyPattern.MatchString(key) {
        return fmt.Errorf("%w: got %q", ErrInvalidEpicKey, key)
    }
    return nil
}
```

**Why regex allowlists?**
- Reject any character outside the defined set — no escaping needed
- Anchor patterns with `^` and `$` to prevent partial matches
- Compile once at package init for performance

### Whitespace Sanitization

Free-text fields must not be empty after trimming whitespace. This catches inputs like `"   "` that look non-empty but contain only whitespace:

```go
func ValidateAgentType(agentType string) error {
    trimmed := strings.TrimSpace(agentType)
    if trimmed == "" {
        return fmt.Errorf("agent type cannot be empty or whitespace-only")
    }
    if len(trimmed) > 100 {
        return fmt.Errorf("agent type too long: maximum 100 characters, got %d", len(trimmed))
    }
    return nil
}
```

**Pattern:** Always `strings.TrimSpace(input)` before checking `== ""`. Apply length limits (e.g., 100 characters) to prevent oversized inputs.

### Enum Allowlists

String fields that must be one of a predefined set use a `map[string]bool` allowlist:

```go
func ValidateNoteType(noteType string) error {
    validTypes := map[string]bool{
        "comment": true, "decision": true, "blocker": true,
        "solution": true, "reference": true, "implementation": true,
        "testing": true, "future": true, "question": true,
        "rejection": true, "requirement": true,
    }
    if !validTypes[noteType] {
        return fmt.Errorf("%w: got %q", ErrInvalidNoteType, noteType)
    }
    return nil
}
```

**Why map lookups instead of switch/case?**
- O(1) lookup regardless of the number of valid values
- Easy to add or remove valid values without restructuring logic
- The `%q` format verb in error messages quotes the input, making injected characters visible in logs

### JSON Array Validation

Fields that store JSON arrays (e.g., `depends_on`) are parsed with `encoding/json` to ensure they are structurally valid before reaching the database:

```go
func ValidateDependsOn(dependsOn string) error {
    if dependsOn == "" || dependsOn == "null" {
        return nil // Empty or null is valid
    }

    var deps []string
    if err := json.Unmarshal([]byte(dependsOn), &deps); err != nil {
        return fmt.Errorf("%w: %v", ErrInvalidDependsOn, err)
    }

    // Validate each element is a valid task key
    for _, dep := range deps {
        if err := ValidateTaskKey(dep); err != nil {
            return fmt.Errorf("invalid task key in depends_on: %w", err)
        }
    }

    return nil
}
```

**Key points:**
- Parse first, then validate individual elements
- Each element goes through the same key validation as direct inputs
- `"null"` is accepted as equivalent to empty (SQLite NULL compatibility)

---

## Layer 2: Service Layer — Business Rule Validation

Status strings are NOT validated against a hardcoded list in the model layer. This is intentional: the model layer cannot import the workflow package (circular dependency). Status validation happens at the service layer via `workflow.Service`:

```go
// BAD: hardcoding status values in models
func ValidateTaskStatus(status string) error {
    validStatuses := []string{"todo", "in_progress", "completed"} // NEVER DO THIS
    ...
}

// GOOD: structural check only in models
func ValidateTaskStatus(status string) error {
    if strings.TrimSpace(status) == "" {
        return fmt.Errorf("%w: status cannot be empty", ErrInvalidTaskStatus)
    }
    return nil
}

// GOOD: workflow-aware validation in services
func (s *TaskService) StartTask(ctx context.Context, key string) error {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return err
    }
    if err := s.workflowSvc.ValidateTransition(string(task.Status), "in_progress"); err != nil {
        return fmt.Errorf("cannot start task %s: %w", key, err)
    }
    return s.repo.UpdateStatus(ctx, task.ID, "in_progress", nil, nil)
}
```

**Rule:** Status values are config-driven. Never hardcode status names in model validators. Use `workflow.Service.ValidateStatus()` or `ValidateTransition()` at the service layer.

---

## Layer 3: Repository Layer — SQL Injection Prevention

**All** database queries use parameterized queries (prepared statements). String interpolation into SQL is never used.

### Correct Pattern

```go
// CORRECT: parameterized query — user input never touches SQL text
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    query := `
        SELECT id, key, title, status, created_at, updated_at
        FROM tasks
        WHERE key = ?
    `
    task := &models.Task{}
    err := r.db.QueryRowContext(ctx, query, key).Scan(
        &task.ID, &task.Key, &task.Title, &task.Status,
        &task.CreatedAt, &task.UpdatedAt,
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

### Anti-Patterns to Avoid

```go
// NEVER DO THIS: string interpolation into SQL
query := fmt.Sprintf("SELECT * FROM tasks WHERE key = '%s'", key) // SQL injection!
query := "SELECT * FROM tasks WHERE key = '" + key + "'"           // SQL injection!

// NEVER DO THIS: using Exec with string formatting instead of placeholders
db.Exec("UPDATE tasks SET title = '" + title + "' WHERE id = " + id) // SQL injection!
```

**SQLite placeholder syntax:** Use `?` for positional parameters. The `database/sql` package automatically handles escaping.

---

## Layer 4: CLI Input Handling

CLI inputs come from Cobra flag parsing. The patterns for sanitizing CLI inputs:

### String Flags

Use `strings.TrimSpace()` on any string flag before passing to services:

```go
func runTaskCreate(cmd *cobra.Command, args []string) error {
    title := strings.TrimSpace(args[len(args)-1])
    if title == "" {
        return fmt.Errorf("title cannot be empty")
    }

    agentType, _ := cmd.Flags().GetString("agent")
    agentType = strings.TrimSpace(agentType)

    // Pass trimmed values to service
    svc := cli.GetTaskService()
    task, err := svc.CreateTask(cmd.Context(), services.CreateTaskInput{
        Title:     title,
        AgentType: agentType,
    })
    ...
}
```

### Key Normalization

Entity keys are normalized to uppercase before lookup to support case-insensitive input:

```go
// Keys are normalized in the repository layer
// Both "e07-f01-001" and "E07-F01-001" resolve to the same task
key = strings.ToUpper(strings.TrimSpace(key))
```

---

## Validation Error Types

Validation errors use sentinel errors defined in `internal/models/validation.go` so callers can use `errors.Is()` to check specific failure types:

```go
var (
    ErrInvalidEpicKey    = errors.New("invalid epic key format: must match ^E\\d{2}$")
    ErrInvalidAgentType  = errors.New("invalid agent type: cannot be empty or whitespace-only")
    ErrEmptyTitle        = errors.New("title cannot be empty")
    ErrInvalidNoteType   = errors.New("invalid note type: ...")
    // ... see validation.go for full list
)

// Usage: Check for specific error type
err := models.ValidateEpicKey(key)
if errors.Is(err, models.ErrInvalidEpicKey) {
    // handle invalid key specifically
}
```

**Error message format:**
- Use `%w` to wrap sentinel errors: `fmt.Errorf("%w: got %q", ErrInvalidEpicKey, key)`
- Use `%q` to quote user-supplied input in error messages — this makes injected special characters visible in logs and avoids log injection

---

## Security Properties

| Threat | Mitigation |
|--------|------------|
| SQL Injection | Parameterized queries (`?` placeholders) via `database/sql` |
| Key format attacks | Regex allowlists anchored with `^` and `$` |
| Oversized inputs | Explicit length checks in validators (e.g., 100 char limit) |
| Whitespace-only inputs | `strings.TrimSpace()` before empty check |
| Invalid enum values | `map[string]bool` allowlist lookups |
| Malformed JSON | `json.Unmarshal` parse-and-validate before storage |
| Log injection | `%q` verb in error messages quotes all user input |
| Circular workflow bypasses | Status transitions validated via `workflow.Service`, not hardcoded |

---

## Checklist for New Input Fields

When adding a new field that accepts user input, verify:

- [ ] **String fields**: `strings.TrimSpace()` before empty check
- [ ] **Key fields**: Regex allowlist matching `^pattern$`
- [ ] **Enum fields**: `map[string]bool` allowlist
- [ ] **Numeric fields**: Range check (min/max)
- [ ] **JSON fields**: Parse with `encoding/json` before storage
- [ ] **Database queries**: Use `?` placeholders, never string interpolation
- [ ] **Error messages**: Quote user input with `%q` to prevent log injection
- [ ] **Length limits**: Apply reasonable maximum length to prevent oversized inputs

---

## Related Documentation

- **Go Patterns**: `.claude/rules/go/patterns.md` — general coding patterns
- **Error Handling**: `.claude/rules/go/error-handling.md` — error wrapping conventions
- **Model Validation**: `internal/models/validation.go` — full set of validation functions
- **Service Design**: `.claude/rules/services/service-design.md` — where business validation lives
