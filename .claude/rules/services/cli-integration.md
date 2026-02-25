---
paths: "internal/cli/**/*"
---

# CLI Integration Patterns with Service Layer

This rule is loaded when working with CLI command files.

## Overview

CLI commands must be **thin wrappers** that:
1. Parse arguments and flags
2. Call a service method
3. Format output (JSON/table) and handle errors

**All business logic belongs in the service layer**, not in CLI commands.

## The Three-Step Pattern

Every CLI command follows this exact pattern:

```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    taskKey := args[0]
    notes := cmd.Flags().GetString("notes")

    // Step 2: Call service
    svc := cli.GetTaskService()
    result, err := svc.PerformOperation(cmd.Context(), taskKey, notes)
    if err != nil {
        return err // Error formatting happens in Cobra or main.go
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }

    cli.Success(fmt.Sprintf("Operation succeeded: %s", result.Key))
    return nil
}
```

**That's it. No loops, no conditionals, no business logic.**

## Before/After Examples

### Example 1: Task Start Command

**BEFORE (Fat Controller - 65 lines)**

```go
// ❌ BAD: Business logic in CLI command
func runTaskStart(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    taskKey := args[0]

    // Get database
    repoDb, err := cli.GetDB(ctx)
    if err != nil {
        return fmt.Errorf("failed to get database: %w", err)
    }

    // Create repository
    repo := repository.NewTaskRepository(repoDb)

    // Get task (data access in command)
    task, err := repo.GetByKey(ctx, taskKey)
    if err != nil {
        cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
        os.Exit(1)
    }

    // Validate current status (business logic in command)
    if task.Status != "todo" && task.Status != "blocked" {
        cli.Error(fmt.Sprintf("Cannot start task in status '%s'. Task must be 'todo' or 'blocked'.", task.Status))
        os.Exit(3)
    }

    // Check dependencies (business logic in command)
    deps, err := repo.GetTaskDependencies(ctx, task.Key)
    if err != nil {
        return err
    }
    for _, dep := range deps {
        if dep.Status != "completed" {
            cli.Error(fmt.Sprintf("Dependency %s is not completed", dep.Key))
            os.Exit(3)
        }
    }

    // Get agent ID
    agentID := cmd.Flags().GetString("agent")

    // Update status (data access in command)
    if err := repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agentID, nil); err != nil {
        cli.Error(fmt.Sprintf("Failed to update task status: %v", err))
        os.Exit(2)
    }

    // Reload task
    task, err = repo.GetByKey(ctx, taskKey)
    if err != nil {
        return err
    }

    // Trigger cascade (business logic in command)
    configPath, _ := cli.GetConfigPath()
    cfg, _ := config.LoadWorkflowConfig(configPath)
    calcService := status.NewCalculationService(repoDb, cfg)
    calcService.CascadeFromFeatureID(ctx, task.FeatureID)

    // Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }

    cli.Success(fmt.Sprintf("Started task %s", task.Key))
    return nil
}
```

**AFTER (Thin Wrapper - 15 lines)**

```go
// ✅ GOOD: Thin wrapper calling service
func runTaskStart(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    taskKey := args[0]
    agentID, _ := cmd.Flags().GetString("agent")

    // Step 2: Call service (all business logic here)
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), taskKey, agentID)
    if err != nil {
        return err // Cobra handles error display and exit codes
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }

    cli.Success(fmt.Sprintf("Started task %s", task.Key))
    return nil
}
```

**What Moved to Service:**
- Workflow validation (status checks)
- Dependency validation
- Status update logic
- Cascade triggering
- Error wrapping with business context

**Line Count Reduction: 65 → 15 (77% reduction)**

### Example 2: Task List Command

**BEFORE (Fat Controller - 120 lines)**

```go
// ❌ BAD: Filtering and business logic in command
func runTaskList(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Get database
    repoDb, err := cli.GetDB(ctx)
    if err != nil {
        return err
    }

    // Parse filters from flags and args
    epicKey, _ := cmd.Flags().GetString("epic")
    featureKey, _ := cmd.Flags().GetString("feature")
    statusFilter, _ := cmd.Flags().GetString("status")
    agentType, _ := cmd.Flags().GetString("agent")
    showAll, _ := cmd.Flags().GetBool("show-all")

    // Parse positional args (business logic in command)
    if len(args) >= 1 {
        epicKey = args[0]
    }
    if len(args) >= 2 {
        if strings.Contains(args[1], "-") {
            featureKey = args[1]
        } else {
            featureKey = epicKey + "-" + args[1]
        }
    }

    // Create repository
    repo := repository.NewTaskRepository(repoDb)

    // Get all tasks (data access in command)
    var tasks []*models.Task
    if featureKey != "" {
        featureRepo := repository.NewFeatureRepository(repoDb)
        feature, err := featureRepo.GetByKey(ctx, featureKey)
        if err != nil {
            return err
        }
        tasks, err = repo.ListByFeature(ctx, feature.ID)
        if err != nil {
            return err
        }
    } else if epicKey != "" {
        tasks, err = repo.ListByEpic(ctx, epicKey)
        if err != nil {
            return err
        }
    } else {
        tasks, err = repo.List(ctx)
        if err != nil {
            return err
        }
    }

    // Filter tasks (business logic in command)
    var filtered []*models.Task
    for _, t := range tasks {
        // Status filter
        if statusFilter != "" && string(t.Status) != statusFilter {
            continue
        }

        // Agent filter
        if agentType != "" && t.AgentType != agentType {
            continue
        }

        // Hide completed unless --show-all
        if !showAll && t.Status == "completed" {
            continue
        }

        filtered = append(filtered, t)
    }

    // Sort tasks (business logic in command)
    sort.Slice(filtered, func(i, j int) bool {
        if filtered[i].ExecutionOrder != filtered[j].ExecutionOrder {
            return filtered[i].ExecutionOrder < filtered[j].ExecutionOrder
        }
        return filtered[i].Priority > filtered[j].Priority
    })

    // Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(filtered)
    }

    // Table formatting
    headers := []string{"Key", "Title", "Status", "Agent", "Priority"}
    var rows [][]string
    for _, t := range filtered {
        rows = append(rows, []string{
            t.Key,
            t.Title,
            string(t.Status),
            t.AgentType,
            fmt.Sprintf("%d", t.Priority),
        })
    }

    return cli.OutputTable(headers, rows)
}
```

**AFTER (Thin Wrapper - 30 lines)**

```go
// ✅ GOOD: Filtering and sorting in service
func runTaskList(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments into filters struct
    filters := parseTaskFilters(cmd, args) // Helper function

    // Step 2: Call service
    svc := cli.GetTaskService()
    tasks, err := svc.ListTasks(cmd.Context(), filters)
    if err != nil {
        return err
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(tasks)
    }

    // Table formatting (presentation logic stays in command)
    headers := []string{"Key", "Title", "Status", "Agent", "Priority"}
    var rows [][]string
    for _, t := range tasks {
        rows = append(rows, []string{
            t.Key,
            t.Title,
            string(t.Status),
            t.AgentType,
            fmt.Sprintf("%d", t.Priority),
        })
    }

    return cli.OutputTable(headers, rows)
}

// Helper: Parse flags and args into filters struct
func parseTaskFilters(cmd *cobra.Command, args []string) services.TaskFilters {
    epicKey, _ := cmd.Flags().GetString("epic")
    featureKey, _ := cmd.Flags().GetString("feature")
    status, _ := cmd.Flags().GetString("status")
    agentType, _ := cmd.Flags().GetString("agent")
    showAll, _ := cmd.Flags().GetBool("show-all")

    // Parse positional args
    if len(args) >= 1 {
        epicKey = args[0]
    }
    if len(args) >= 2 {
        featureKey = args[1]
        if !strings.Contains(featureKey, "-") {
            featureKey = epicKey + "-" + featureKey
        }
    }

    return services.TaskFilters{
        EpicKey:    epicKey,
        FeatureKey: featureKey,
        Status:     status,
        AgentType:  agentType,
        ShowAll:    showAll,
    }
}
```

**What Moved to Service:**
- Feature/epic key resolution
- Repository selection logic
- Task filtering by status/agent/completed
- Sorting by execution order and priority
- Business rules for default filters

**Line Count Reduction: 120 → 30 (75% reduction)**

### Example 3: Task Create Command

**BEFORE (Fat Controller - 180 lines)**

```go
// ❌ BAD: Validation, file creation, key generation in command
func runTaskCreate(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Parse args (complex parsing logic in command)
    var epicKey, featureKey, title string
    if len(args) == 3 {
        epicKey, featureKey, title = args[0], args[1], args[2]
    } else if len(args) == 2 {
        combined, title := args[0], args[1]
        parts := strings.Split(combined, "-")
        epicKey, featureKey = parts[0], parts[1]
    } else {
        epicKey, _ = cmd.Flags().GetString("epic")
        featureKey, _ = cmd.Flags().GetString("feature")
        title = args[0]
    }

    // Validate inputs (validation in command)
    if epicKey == "" || featureKey == "" || title == "" {
        return fmt.Errorf("epic, feature, and title are required")
    }

    // Get database
    repoDb, err := cli.GetDB(ctx)
    if err != nil {
        return err
    }

    // Validate epic exists (data access in command)
    epicRepo := repository.NewEpicRepository(repoDb)
    epic, err := epicRepo.GetByKey(ctx, epicKey)
    if err != nil {
        return fmt.Errorf("epic %s not found", epicKey)
    }

    // Validate feature exists (data access in command)
    featureRepo := repository.NewFeatureRepository(repoDb)
    feature, err := featureRepo.GetByKey(ctx, featureKey)
    if err != nil {
        return fmt.Errorf("feature %s not found", featureKey)
    }

    // Get next task number (business logic in command)
    taskRepo := repository.NewTaskRepository(repoDb)
    existingTasks, _ := taskRepo.ListByFeature(ctx, feature.ID)
    nextNum := len(existingTasks) + 1
    taskKey := fmt.Sprintf("T-%s-%s-%03d", epicKey, featureKey, nextNum)

    // Parse optional fields
    agentType, _ := cmd.Flags().GetString("agent")
    priority, _ := cmd.Flags().GetInt("priority")
    execOrder, _ := cmd.Flags().GetInt("order")
    filePath, _ := cmd.Flags().GetString("file")
    createFile, _ := cmd.Flags().GetBool("create")

    // Create task model (model creation in command)
    task := &models.Task{
        Key:            taskKey,
        Title:          title,
        Status:         models.TaskStatus("todo"),
        EpicID:         epic.ID,
        FeatureID:      feature.ID,
        AgentType:      agentType,
        Priority:       priority,
        ExecutionOrder: execOrder,
    }

    // Validate model (validation in command)
    if priority < 1 || priority > 10 {
        return fmt.Errorf("priority must be between 1 and 10")
    }

    // Create file (file operations in command)
    projectRoot, _ := pathresolver.FindProjectRoot()
    if filePath == "" {
        filePath = filepath.Join("docs/plan", epic.Key, feature.Key, taskKey+".md")
    }

    // Generate file content (content generation in command)
    tmpl, _ := templates.LoadTaskTemplate()
    content, _ := tmpl.Execute(task)

    // Write file (file I/O in command)
    writer := fileops.NewEntityFileWriter()
    result, err := writer.WriteEntityFile(fileops.WriteOptions{
        Content:         content,
        ProjectRoot:     projectRoot,
        FilePath:        filePath,
        EntityType:      "task",
        UseAtomicWrite:  true,
        CreateIfMissing: createFile,
    })
    if err != nil {
        return err
    }

    task.FilePath = result.AbsolutePath

    // Save to database (data access in command)
    if err := taskRepo.Create(ctx, task); err != nil {
        return fmt.Errorf("failed to create task: %w", err)
    }

    // Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }

    cli.Success(fmt.Sprintf("Created task %s", task.Key))
    return nil
}
```

**AFTER (Thin Wrapper - 25 lines)**

```go
// ✅ GOOD: All creation logic in service
func runTaskCreate(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments into input struct
    input := parseCreateTaskInput(cmd, args) // Helper function

    // Step 2: Call service (all logic here)
    svc := cli.GetTaskService()
    task, err := svc.CreateTask(cmd.Context(), input)
    if err != nil {
        return err
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }

    cli.Success(fmt.Sprintf("Created task %s at %s", task.Key, task.FilePath))
    return nil
}

// Helper: Parse flags and args into CreateTaskInput
func parseCreateTaskInput(cmd *cobra.Command, args []string) services.CreateTaskInput {
    // Parse positional args (3-arg, 2-arg, or flag format)
    epicKey, featureKey, title := parseTaskCreateArgs(args, cmd)

    // Parse optional flags
    agentType, _ := cmd.Flags().GetString("agent")
    priority, _ := cmd.Flags().GetInt("priority")
    execOrder, _ := cmd.Flags().GetInt("order")
    filePath, _ := cmd.Flags().GetString("file")
    createFile, _ := cmd.Flags().GetBool("create")
    force, _ := cmd.Flags().GetBool("force")

    return services.CreateTaskInput{
        EpicKey:        epicKey,
        FeatureKey:     featureKey,
        Title:          title,
        AgentType:      agentType,
        Priority:       priority,
        ExecutionOrder: execOrder,
        FilePath:       filePath,
        CreateFile:     createFile,
        Force:          force,
    }
}
```

**What Moved to Service:**
- Epic/feature validation
- Task key generation
- Next task number calculation
- File path resolution
- Template rendering
- File creation
- Database save
- Transaction management

**Line Count Reduction: 180 → 25 (86% reduction)**

## Command Responsibility Matrix

| Responsibility | CLI Command | Service | Repository |
|----------------|-------------|---------|------------|
| Parse CLI args | ✅ | ❌ | ❌ |
| Parse CLI flags | ✅ | ❌ | ❌ |
| Validate business rules | ❌ | ✅ | ❌ |
| Check workflow transitions | ❌ | ✅ | ❌ |
| Validate dependencies | ❌ | ✅ | ❌ |
| Key generation | ❌ | ✅ | ❌ |
| File creation | ❌ | ✅ | ❌ |
| Database queries | ❌ | ❌ | ✅ |
| Transaction management | ❌ | ✅ | ❌ |
| Error wrapping | ❌ | ✅ | ✅ |
| Format JSON output | ✅ | ❌ | ❌ |
| Format table output | ✅ | ❌ | ❌ |
| Exit code mapping | ✅ | ❌ | ❌ |

## Service Accessor Pattern

CLI commands access services via **global accessor functions** defined in `internal/cli/services_global.go`.

**Pattern:**

```go
// In internal/cli/services_global.go
func GetTaskService() *services.TaskService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    taskRepo := repository.NewTaskRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewTaskService(taskRepo, workflowSvc, nil, nil)
}

// In CLI command
func runMyCommand(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService() // Get service instance
    result, err := svc.DoSomething(cmd.Context(), args[0])
    if err != nil {
        return err
    }
    cli.Success("Done")
    return nil
}
```

**Why Global Accessors:**
- **Simplicity**: Commands don't manage service lifecycle
- **Consistency**: Same pattern across all commands
- **Lazy initialization**: Service created only when needed
- **Shared dependencies**: DB connection and workflow service reused
- **Testability**: Can be replaced with mocks via dependency injection in tests

**Pattern Comparison:**

```go
// ❌ BAD: Command constructs service manually
func runMyCommand(cmd *cobra.Command, args []string) error {
    db, _ := cli.GetDB(cmd.Context())
    repo := repository.NewTaskRepository(db)
    workflow := workflow.NewService(projectRoot)
    svc := services.NewTaskService(repo, workflow, nil, nil)
    // ... use service
}

// ✅ GOOD: Command uses global accessor
func runMyCommand(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService() // All wiring hidden
    // ... use service
}
```

## Error Handling in Commands

Commands translate service errors into appropriate **exit codes** and user-friendly messages.

**Note on `os.Exit()`**: Calling `os.Exit()` directly in Cobra command handlers bypasses Cobra's `PersistentPostRun` cleanup hooks and makes the command difficult to test. Prefer returning a typed error from the handler and letting the root command's `Execute()` error handler translate it to an exit code via `os.Exit()`. The pattern below shows typed error returns as the preferred approach.

**Pattern:**

```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), args[0], agentID)
    if err != nil {
        // Return typed errors — the caller maps these to exit codes
        var notFoundErr *repository.NotFoundError
        if errors.As(err, &notFoundErr) {
            cli.Error(fmt.Sprintf("Task not found: %s", args[0]))
            return fmt.Errorf("exit code 1: %w", err)
        }

        var workflowErr *workflow.TransitionError
        if errors.As(err, &workflowErr) {
            cli.Error(fmt.Sprintf("Invalid status transition: %v", err))
            return fmt.Errorf("exit code 3: %w", err)
        }

        // Generic error
        cli.Error(fmt.Sprintf("Error: %v", err))
        return fmt.Errorf("exit code 2: %w", err)
    }

    cli.Success(fmt.Sprintf("Task %s started", task.Key))
    return nil
}
```

**Exit Code Mapping:**
- **0**: Success
- **1**: Not found (entity doesn't exist)
- **2**: Database/system error
- **3**: Invalid state (business rule violation)

See `.claude/rules/go/error-handling.md` for complete error patterns.

## Output Formatting Patterns

### JSON Output

```go
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(result)
}
```

**Always use `cli.OutputJSON()`** for JSON formatting - it handles indentation and marshaling.

### Table Output

```go
headers := []string{"Key", "Title", "Status"}
var rows [][]string
for _, task := range tasks {
    rows = append(rows, []string{
        task.Key,
        task.Title,
        string(task.Status),
    })
}
return cli.OutputTable(headers, rows)
```

**Table formatting is presentation logic** - it belongs in commands, not services.

### Success/Error Messages

```go
cli.Success("Operation succeeded")        // Green checkmark + message
cli.Error("Operation failed")            // Red X + message
cli.Warning("Potential issue")           // Yellow warning + message
cli.Info("Additional information")       // Blue info + message
```

## Helper Function Pattern

For complex argument parsing, extract helper functions:

```go
// Main command handler (stays clean)
func runTaskCreate(cmd *cobra.Command, args []string) error {
    input := parseCreateTaskInput(cmd, args)
    svc := cli.GetTaskService()
    task, err := svc.CreateTask(cmd.Context(), input)
    if err != nil {
        return err
    }
    cli.Success(fmt.Sprintf("Created task %s", task.Key))
    return nil
}

// Helper: Parse args into DTO
func parseCreateTaskInput(cmd *cobra.Command, args []string) services.CreateTaskInput {
    epicKey, featureKey, title := parsePositionalArgs(args, cmd)

    agentType, _ := cmd.Flags().GetString("agent")
    priority, _ := cmd.Flags().GetInt("priority")

    return services.CreateTaskInput{
        EpicKey:   epicKey,
        FeatureKey: featureKey,
        Title:     title,
        AgentType: agentType,
        Priority:  priority,
    }
}

// Helper: Parse positional args (3-arg, 2-arg, or flag format)
func parsePositionalArgs(args []string, cmd *cobra.Command) (epic, feature, title string) {
    if len(args) == 3 {
        return args[0], args[1], args[2]
    }
    if len(args) == 2 {
        combined := args[0]
        parts := strings.Split(combined, "-")
        return parts[0], parts[1], args[1]
    }
    epic, _ = cmd.Flags().GetString("epic")
    feature, _ = cmd.Flags().GetString("feature")
    title = args[0]
    return
}
```

**Benefits:**
- Main handler stays focused (parse → call → format)
- Parsing logic is reusable
- Easier to test argument parsing separately
- Clear separation of concerns

## Migration Checklist for Existing Commands

When refactoring a fat CLI command to use services:

1. [ ] **Identify business logic** in the command (validation, workflow checks, filtering, sorting)
2. [ ] **Create or update service method** that encapsulates this logic
3. [ ] **Create DTO** for complex inputs (e.g., `CreateTaskInput`, `TaskFilters`)
4. [ ] **Extract helper function** for argument parsing if needed
5. [ ] **Replace repository calls** in command with service call
6. [ ] **Move validation logic** from command to service
7. [ ] **Move workflow checks** from command to service
8. [ ] **Keep presentation logic** in command (table formatting, JSON output)
9. [ ] **Update error handling** to translate service errors to exit codes
10. [ ] **Test command** with mocked service (not real database)
11. [ ] **Update service tests** to cover new logic
12. [ ] **Remove unused repository imports** from command file

See `docs/guides/service-layer-migration.md` for step-by-step migration guide.

## Testing CLI Commands with Services

Commands are tested with **mocked services**, not real databases.

**Pattern:**

```go
// Mock service
type MockTaskService struct {
    StartTaskFunc func(ctx context.Context, key string, agentID string) (*models.Task, error)
}

func (m *MockTaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
    if m.StartTaskFunc != nil {
        return m.StartTaskFunc(ctx, key, agentID)
    }
    return nil, fmt.Errorf("not implemented")
}

// Test command with mock
func TestTaskStartCommand(t *testing.T) {
    mockSvc := &MockTaskService{
        StartTaskFunc: func(ctx context.Context, key string, agentID string) (*models.Task, error) {
            assert.Equal(t, "E07-F01-001", key)
            assert.Equal(t, "agent123", agentID)
            return &models.Task{
                Key:    "E07-F01-001",
                Status: "in_progress",
            }, nil
        },
    }

    // Inject mock into command (via global accessor replacement in test setup)
    cli.OverrideTaskService(mockSvc) // Test helper
    defer cli.ResetServices()

    // Execute command
    err := runTaskStart(cmd, []string{"E07-F01-001"})
    assert.NoError(t, err)
}
```

See `.claude/rules/testing/cli-tests.md` for complete CLI testing patterns.

## Common Mistakes to Avoid

### Mistake 1: Doing Validation in Commands

```go
// ❌ BAD
func runTaskCreate(cmd *cobra.Command, args []string) error {
    priority, _ := cmd.Flags().GetInt("priority")
    if priority < 1 || priority > 10 {
        return fmt.Errorf("priority must be between 1 and 10")
    }
    // ...
}

// ✅ GOOD: Validation in service
func runTaskCreate(cmd *cobra.Command, args []string) error {
    input := parseCreateTaskInput(cmd, args)
    svc := cli.GetTaskService()
    return svc.CreateTask(cmd.Context(), input) // Service validates
}
```

### Mistake 2: Formatting Output in Services

```go
// ❌ BAD: Service returns formatted string
func (s *TaskService) GetTask(ctx context.Context, key string) (string, error) {
    task, _ := s.repo.GetByKey(ctx, key)
    return fmt.Sprintf("Task: %s (%s)", task.Title, task.Status), nil
}

// ✅ GOOD: Service returns domain model, command formats
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    return s.repo.GetByKey(ctx, key)
}

func runTaskGet(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    task, _ := svc.GetTask(cmd.Context(), args[0])
    cli.Success(fmt.Sprintf("Task: %s (%s)", task.Title, task.Status))
    return nil
}
```

### Mistake 3: Calling Repositories Directly from Commands

```go
// ❌ BAD: Command bypasses service
func runTaskList(cmd *cobra.Command, args []string) error {
    db, _ := cli.GetDB(cmd.Context())
    repo := repository.NewTaskRepository(db)
    tasks, _ := repo.List(cmd.Context())
    // ...
}

// ✅ GOOD: Command calls service
func runTaskList(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    tasks, _ := svc.ListTasks(cmd.Context(), filters)
    // ...
}
```

### Mistake 4: Complex Logic in Argument Parsing

```go
// ❌ BAD: Complex logic inline
func runTaskCreate(cmd *cobra.Command, args []string) error {
    var epicKey, featureKey, title string
    if len(args) == 3 {
        epicKey, featureKey, title = args[0], args[1], args[2]
    } else if len(args) == 2 {
        combined := args[0]
        parts := strings.Split(combined, "-")
        if len(parts) != 2 {
            return fmt.Errorf("invalid format")
        }
        epicKey, featureKey, title = parts[0], parts[1], args[1]
    } else {
        // ... more complex parsing
    }
    // ... rest of command
}

// ✅ GOOD: Extract to helper
func runTaskCreate(cmd *cobra.Command, args []string) error {
    input := parseCreateTaskInput(cmd, args) // Helper function
    svc := cli.GetTaskService()
    return svc.CreateTask(cmd.Context(), input)
}
```

## Related Documentation

- **Service Design**: `.claude/rules/services/service-design.md`
- **Service Testing**: `.claude/rules/services/testing.md`
- **HTTP Integration**: `.claude/rules/services/http-integration.md`
- **CLI Command Reference**: `.claude/rules/cli/commands.md`
- **CLI Patterns**: `.claude/rules/cli/patterns.md`
- **Migration Guide**: `docs/guides/service-layer-migration.md`
