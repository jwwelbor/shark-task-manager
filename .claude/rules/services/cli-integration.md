---
paths: "internal/cli/**/*"
---

# CLI Integration with Service Layer

CLI commands must be **thin wrappers**: parse args, call service, format output. **No business logic in commands.**

## The Three-Step Pattern

```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    // 1. Parse arguments
    taskKey := args[0]

    // 2. Call service
    svc := cli.GetTaskService()
    result, err := svc.PerformOperation(cmd.Context(), taskKey)
    if err != nil {
        return err
    }

    // 3. Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }
    cli.Success(fmt.Sprintf("Operation succeeded: %s", result.Key))
    return nil
}
```

## Before/After Example

**BEFORE (Fat Controller - 65 lines):**
```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    repoDb, _ := cli.GetDB(cmd.Context())
    repo := repository.NewTaskRepository(repoDb)
    task, _ := repo.GetByKey(ctx, args[0])

    // Business logic that belongs in service:
    if task.Status != "todo" && task.Status != "blocked" { ... }
    deps, _ := repo.GetTaskDependencies(ctx, task.Key)
    for _, dep := range deps { ... }
    repo.UpdateStatus(ctx, task.ID, "in_progress", &agentID, nil)
    calcService.CascadeFromFeatureID(ctx, task.FeatureID)
    // ...
}
```

**AFTER (Thin Wrapper - 15 lines):**
```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    taskKey := args[0]
    agentID, _ := cmd.Flags().GetString("agent")

    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), taskKey, agentID)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }
    cli.Success(fmt.Sprintf("Started task %s", task.Key))
    return nil
}
```

## Responsibility Matrix

| Responsibility | CLI Command | Service | Repository |
|----------------|:-----------:|:-------:|:----------:|
| Parse args/flags | x | | |
| Business validation | | x | |
| Workflow transitions | | x | |
| Key generation | | x | |
| File creation | | x | |
| Database queries | | | x |
| Transaction management | | x | |
| Format JSON/table output | x | | |
| Exit code mapping | x | | |

## Service Accessor Pattern

Commands access services via global accessors in `internal/cli/services_global.go`:

```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService() // Lazy init, shared DB, new instance per call
    result, err := svc.DoSomething(cmd.Context(), args[0])
    // ...
}
```

Do NOT construct services manually in commands. Use `cli.Get<Entity>Service()`.

## Error Handling

Prefer returning typed errors over calling `os.Exit()` directly (bypasses cleanup hooks):

```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), args[0], agentID)
    if err != nil {
        var notFoundErr *repository.NotFoundError
        if errors.As(err, &notFoundErr) {
            cli.Error(fmt.Sprintf("Task not found: %s", args[0]))
            return fmt.Errorf("exit code 1: %w", err)
        }
        return err
    }
    cli.Success(fmt.Sprintf("Task %s started", task.Key))
    return nil
}
```

Exit codes: 0=success, 1=not found, 2=DB error, 3=invalid state.

## Output Formatting

```go
// JSON
if cli.GlobalConfig.JSON { return cli.OutputJSON(result) }

// Table (presentation logic stays in commands)
return cli.OutputTable(headers, rows)

// Messages
cli.Success("...")  cli.Error("...")  cli.Warning("...")  cli.Info("...")
```

## Helper Functions for Complex Parsing

Extract argument parsing into helpers to keep handlers clean:

```go
func runTaskCreate(cmd *cobra.Command, args []string) error {
    input := parseCreateTaskInput(cmd, args) // Helper
    svc := cli.GetTaskService()
    task, err := svc.CreateTask(cmd.Context(), input)
    if err != nil { return err }
    cli.Success(fmt.Sprintf("Created task %s", task.Key))
    return nil
}
```

## Rules (Do NOT Violate)

1. **No validation in commands** — service validates
2. **No output formatting in services** — services return domain models
3. **No direct repository calls from commands** — always go through service
4. **No inline complex logic** — extract to helper functions
5. **Test commands with mocked services** — never real database

See `docs/guides/service-layer-migration.md` for migrating existing fat commands.
