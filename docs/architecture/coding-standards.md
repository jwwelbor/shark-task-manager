# Coding Standards

**Project**: Shark Task Manager
**Stack**: Go 1.23.4, SQLite3/libsql, Cobra CLI, Viper config
**Priorities**: Correctness > Maintainability > Security > Performance

This file is the index and core Go coding standard for the codebase. It covers rules that apply to code quality independent of Shark workflow or agent behavior.

Project workflow, agent instructions, service-layer migration rules, and command usage belong in `CLAUDE.md` and `.claude/rules/`, not in coding standards.

## Standards Index

| Pathway | Use When | Standard |
|---------|----------|----------|
| Core Go | Writing any Go code | This file |
| Repositories and database | Writing SQL, scanning rows, managing persistence resources | [Repositories and Database](coding-standards/repositories-database.md) |
| Tests | Writing or changing Go tests | [Testing](coding-standards/testing.md) |
| Security | Handling input, SQL, filesystem paths, or error exposure | [Security](coding-standards/security.md) |

## Related Project Guidance

These are intentionally outside coding standards:

| Topic | Location |
|-------|----------|
| Agent entry point and repo navigation | [CLAUDE.md](../../CLAUDE.md) |
| Tool-neutral agent pointer | [AGENTS.md](../../AGENTS.md) |
| Build/test workflow and Shark task lifecycle | [.claude/rules/development-workflows.md](../../.claude/rules/development-workflows.md) |
| Architecture boundaries and service-layer rules | [.claude/rules/architecture.md](../../.claude/rules/architecture.md) |
| CLI command implementation rules | [.claude/rules/cli/commands.md](../../.claude/rules/cli/commands.md) |
| Service design rules | [.claude/rules/services/service-design.md](../../.claude/rules/services/service-design.md) |
| Test architecture rules | [.claude/rules/testing/architecture.md](../../.claude/rules/testing/architecture.md) |
| Durable architecture overview | [architecture-overview.md](architecture-overview.md) |

## Core Go Standards

### Formatting

**Rule**: Run `gofmt` on all Go code.

**Why**: Formatting should be mechanical and consistent.

**Enforce**: `make fmt`

```bash
make fmt
```

**Rule**: Packages need package-level documentation when they expose reusable behavior.

**Why**: Package docs make boundaries discoverable.

```go
// Package repository provides data access helpers for task entities.
package repository
```

### Naming

**Rule**: Use MixedCaps for Go identifiers. Do not use underscores in identifiers.

```go
// Good
type TaskRepository struct{}
func getByID() {}

// Bad
type task_repository struct{}
func get_by_id() {}
```

**Rule**: Test files must use the `_test.go` suffix.

```text
validator_test.go
```

**Rule**: Interface names should describe behavior. Use an `-er` name when it reads naturally, but do not force awkward names.

```go
type TaskReader interface {
    GetByKey(ctx context.Context, key string) (*models.Task, error)
}
```

Avoid `TaskRepositoryInterface`.

### Error Handling

**Rule**: Wrap errors with context at boundaries using `fmt.Errorf` and `%w`.

```go
if err := task.Validate(); err != nil {
    return fmt.Errorf("validate task: %w", err)
}
```

**Rule**: Define expected sentinel errors as package-level variables with an `Err` prefix.

```go
var ErrInvalidTaskStatus = errors.New("invalid task status")
```

**Rule**: Use typed errors when callers need structured branching.

```go
type ValidationError struct {
    Field string
    Msg   string
}
```

**Rule**: Check errors with `errors.Is` or `errors.As`. Never string-match error messages.

```go
if errors.Is(err, ErrInvalidTaskStatus) {
    return fmt.Errorf("status rejected: %w", err)
}
```

**Rule**: Never ignore errors. Use `_` only with a comment explaining why the error is impossible or intentionally irrelevant.

```go
id, err := result.LastInsertId()
if err != nil {
    return fmt.Errorf("get last insert id: %w", err)
}
```

### Context

**Rule**: `context.Context` is the first parameter for operations that can block, perform IO, or call another layer.

```go
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error)
```

**Rule**: Do not store `context.Context` in structs.

**Rule**: Use context for cancellation, deadlines, and request-scoped values only. Do not use context for dependency injection.

**Rule**: CLI commands and handlers should create or inherit a bounded context and pass it down.

```go
ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
defer cancel()
```

### Constants and Enums

**Rule**: Use typed constants for status values and other constrained sets. Avoid magic strings in comparisons.

```go
type TaskStatus string

const (
    TaskStatusTodo       TaskStatus = "todo"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusCompleted  TaskStatus = "completed"
)

if task.Status == TaskStatusInProgress {
    // ...
}
```

### Structs and Receivers

**Rule**: Struct tags must be explicit and consistent for serialized or persisted fields.

```go
type Task struct {
    ID     int64      `json:"id" db:"id"`
    Title  string     `json:"title" db:"title"`
    Status TaskStatus `json:"status" db:"status"`
}
```

Use response DTOs when API/CLI output needs differ from the domain model.

**Rule**: Use pointer receivers for methods that modify state or operate on large structs.

```go
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error
```

### Documentation

**Rule**: Exported functions and types must have doc comments that start with the exported name.

```go
// ValidateTaskKey validates the task key format.
func ValidateTaskKey(key string) error
```

### Code Hygiene

**Rule**: Prefer early returns over deep nesting.

```go
if err != nil {
    return err
}

// main path
```

**Rule**: Use named return values only when they clarify the API.

**Rule**: Avoid generic `utils` packages. Put helpers near the behavior they support unless there is a proven cross-cutting concept.

## Definition of Done

Before code is complete:

```bash
make fmt
make vet
make lint
make test
```

Do not suppress warnings or skip tests without a documented reason.
