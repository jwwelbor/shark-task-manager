---
paths: "internal/cli/**/*"
---

# CLI Patterns

This rule is loaded when working with CLI-related files.

## Command Layer Responsibility

CLI commands must be **thin wrappers** with three responsibilities only:
1. **Parse** arguments and flags
2. **Call** a service method (all business logic lives in `internal/services/`)
3. **Format** output (JSON, table, success/error messages)

**Commands must NOT**: contain business logic, call repositories directly, manage transactions, implement filtering/validation logic, or check workflow rules. See `@.claude/rules/architecture.md` for the full layering rules.

## CLI Output Patterns

### Check for JSON Output
- Use `cli.GlobalConfig.JSON` to check if JSON output is needed
- Always output JSON with indentation when requested: use `cli.OutputJSON(data)`
- Table output via `cli.OutputTable(headers, rows)` for human readability

### Output Functions
- `cli.Success(message)` - Success messages
- `cli.Error(message)` - Error messages
- `cli.Warning(message)` - Warning messages
- `cli.Info(message)` - Informational messages

### Example
```go
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(data)
}

// Human-readable table output
headers := []string{"Key", "Title", "Status"}
rows := [][]string{
    {"E07-F01-001", "Task Title", "todo"},
}
return cli.OutputTable(headers, rows)
```

## Root Command Structure

### Global Flags
Available to all commands:
- `--json`: Machine-readable JSON output (required for AI agents)
- `--no-color`: Disable colored output
- `--verbose` / `-v`: Enable debug logging
- `--db`: Override database path (default: `shark-tasks.db`)
- `--config`: Override config file path (default: `.sharkconfig.json`)

### Command Registration
Commands automatically register themselves via `init()` functions:

```go
func init() {
    cli.RootCmd.AddCommand(myCmd)
}
```

## Validation Patterns

Two levels of validation:
- **Model layer** (`internal/models/validation.go`): Basic structural validation (non-empty, range checks). Must NOT hardcode status lists or import workflow packages.
- **Service layer** (`internal/services/`): Business rule validation (workflow status checks, transition validity) via `workflow.Service`.

### Example
```go
// In service layer (correct place for business validation)
func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, error) {
    task := &models.Task{
        Title:  input.Title,
        Status: models.TaskStatus(s.workflowSvc.GetDefaultStatus()),
    }

    // Structural validation (model layer)
    if err := task.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Business validation (service layer)
    if err := s.workflowSvc.ValidateStatus(string(task.Status)); err != nil {
        return nil, err
    }

    return task, s.repo.Create(ctx, task)
}
```

## Key Format Flexibility

**All entity keys are case insensitive:**
- Epic keys: `E07`, `e07`, `E07-user-management`, `e07-user-management`
- Feature keys: `E07-F01`, `e07-f01`, `F01`, `f01`
- Task keys: `E07-F20-001`, `e07-f20-001` (short format), `T-E07-F20-001`, `t-e07-f20-001` (traditional)

**Short task key format (recommended):**
- Use `E07-F20-001` instead of `T-E07-F20-001`
- The `T-` prefix is optional and automatically normalized
- Both formats work identically in all commands

**Positional argument syntax:**
- Feature create: `shark feature create E07 "Feature Title"`
- Task create: `shark task create E07 F20 "Task Title"` or `shark task create E07-F20 "Task Title"`
- Legacy flag syntax still fully supported
