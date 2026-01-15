---
paths: "internal/cli/**/*"
---

# CLI Patterns

This rule is loaded when working with CLI-related files.

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

### Model Validation
- Models have `Validate()` methods in `internal/models/validation.go`
- Validate at model layer BEFORE database operations
- Database constraints (CHECK, FOREIGN KEY) provide additional safety

### Example
```go
task := &models.Task{
    Title: title,
    // ... other fields
}

if err := task.Validate(); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

// Proceed with database operation
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
