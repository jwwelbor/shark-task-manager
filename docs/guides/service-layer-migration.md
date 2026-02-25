# Service Layer Migration Guide

Comprehensive guide for migrating fat CLI commands to the service layer pattern.

## Overview

This guide walks you through refactoring existing CLI commands that contain business logic (fat controllers) into thin wrappers that call service methods. The goal is to move all business logic from CLI commands into the service layer.

**Target**: Reduce CLI command files from 100+ lines to 15-30 lines by extracting business logic into services.

## Migration Benefits

- **Reusability**: Service logic works for CLI, HTTP API, tests, and future entry points
- **Testability**: Service logic can be tested with mocks (no database needed)
- **Maintainability**: Business logic in one place, not scattered across commands
- **Clarity**: Commands are simple: parse → call → format

## Before You Start

### Prerequisites

1. Read `.claude/rules/services/service-design.md` - Understand service design principles
2. Read `.claude/rules/services/cli-integration.md` - See before/after examples
3. Identify the command to refactor (start with simple commands)

### Checklist Before Migration

- [ ] Command file exists and contains business logic (not just a thin wrapper)
- [ ] You understand what business logic is in the command
- [ ] You have tests for the command (or can write them)
- [ ] You have identified the service that should own this logic (TaskService, FeatureService, EpicService)

## 10-Step Migration Process

### Step 1: Analyze Current Command

**Goal**: Identify what business logic exists in the command.

**Actions**:
1. Open the command file (e.g., `internal/cli/commands/task.go`)
2. Find the command handler function (e.g., `runTaskStart`)
3. Identify business logic:
   - [ ] Workflow validation (status checks)
   - [ ] Dependency validation
   - [ ] Filtering/sorting logic
   - [ ] Key generation
   - [ ] File creation
   - [ ] Progress calculation
   - [ ] Status cascading
   - [ ] Complex conditionals
4. Note repository calls (these should move to service)
5. Note presentation logic (table/JSON formatting - this stays in command)

**Example Analysis:**

```go
// Command: runTaskStart (65 lines - TOO MUCH)
func runTaskStart(cmd *cobra.Command, args []string) error {
    // ✅ Stays: Argument parsing
    taskKey := args[0]

    // ❌ Move to service: Database access
    repoDb, _ := cli.GetDB(ctx)
    repo := repository.NewTaskRepository(repoDb)

    // ❌ Move to service: Data retrieval
    task, _ := repo.GetByKey(ctx, taskKey)

    // ❌ Move to service: Business validation
    if task.Status != "todo" && task.Status != "blocked" {
        return fmt.Errorf("cannot start task in status %s", task.Status)
    }

    // ❌ Move to service: Dependency check
    deps, _ := repo.GetTaskDependencies(ctx, task.Key)
    for _, dep := range deps {
        if dep.Status != "completed" {
            return fmt.Errorf("dependency %s not completed", dep.Key)
        }
    }

    // ❌ Move to service: Status update
    repo.UpdateStatus(ctx, task.ID, "in_progress", &agentID, nil)

    // ❌ Move to service: Cascade trigger
    calcService := status.NewCalculationService(repoDb, cfg)
    calcService.CascadeFromFeatureID(ctx, task.FeatureID)

    // ✅ Stays: Output formatting
    cli.Success("Task started")
    return nil
}
```

**Document findings**:
- Business logic: 40 lines (62%)
- Presentation logic: 5 lines (8%)
- Argument parsing: 5 lines (8%)
- Repository calls: 15 lines (23%)

### Step 2: Design Service Method

**Goal**: Define the service method signature and responsibilities.

**Actions**:
1. Choose service file (`task_service.go`, `feature_service.go`, etc.)
2. Design method signature:
   - Name: `StartTask`, `CompleteTask`, `ListTasks`, etc.
   - Inputs: Business-level parameters (task key, not ID)
   - Outputs: Domain model (`*models.Task`), error
3. Identify dependencies (repositories, other services)
4. Create DTO if input is complex (more than 3 parameters)

**Example Design:**

```go
// Service method signature
func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
    // Implementation steps:
    // 1. Get task from repository
    // 2. Validate current status allows starting
    // 3. Check dependencies are met
    // 4. Update status to in_progress
    // 5. Trigger status cascade (if needed)
    // 6. Return updated task
}
```

**Document design**:
- Method name: `StartTask`
- Inputs: `ctx context.Context, key string, agentID string`
- Outputs: `*models.Task, error`
- Dependencies: TaskRepository, workflow.Service
- Business rules: Status must be todo/blocked, dependencies must be completed

### Step 3: Create or Update Service Method

**Goal**: Implement the service method with all business logic from the command.

**Actions**:
1. Open service file (`internal/services/task_service.go`)
2. Add method to service struct
3. Move business logic from command to service:
   - Copy validation logic
   - Copy workflow checks
   - Copy repository calls
   - Copy orchestration logic
4. Add error wrapping with business context
5. Add godoc comments

**Example Implementation:**

```go
// StartTask transitions a task to in_progress (or appropriate start status).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to start
//   - agentID: optional agent identifier for tracking who started the task
//
// Returns:
//   - *models.Task: the updated task
//   - error: validation errors, workflow errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - WorkflowError: invalid transition (task not in a startable status)
//   - DependencyError: task dependencies not met
//   - RepositoryError: database update failed
func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
    // Step 1: Get task
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to start task %s: %w", key, err)
    }

    // Step 2: Validate workflow transition
    if err := s.workflowSvc.ValidateTransition(string(task.Status), "in_progress"); err != nil {
        return nil, fmt.Errorf("cannot start task %s in status %s: %w", key, task.Status, err)
    }

    // Step 3: Validate dependencies
    if err := s.ValidateDependencies(ctx, key, "in_progress"); err != nil {
        return nil, fmt.Errorf("dependencies not met for task %s: %w", key, err)
    }

    // Step 4: Update status
    if err := s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agentID, nil); err != nil {
        return nil, fmt.Errorf("failed to update task %s status: %w", key, err)
    }

    // Step 5: Reload task (optional, or update in-place)
    task, err = s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, err
    }

    // Step 6: Trigger cascade (future: move to separate method or event)
    // TODO: Implement cascade logic

    return task, nil
}
```

### Step 4: Create DTOs (If Needed)

**Goal**: For complex inputs (>3 parameters), create a DTO.

**Actions**:
1. Open or create `internal/services/task_dto.go`
2. Define input DTO struct
3. Add JSON tags for HTTP API compatibility
4. Add validation tags if using validator library

**Example DTO:**

```go
// CreateTaskInput contains the parameters for creating a new task.
type CreateTaskInput struct {
    // Required fields
    EpicKey    string `json:"epic_key" validate:"required"`
    FeatureKey string `json:"feature_key" validate:"required"`
    Title      string `json:"title" validate:"required,min=3"`

    // Optional fields
    AgentType      string   `json:"agent_type,omitempty"`
    Priority       int      `json:"priority,omitempty" validate:"min=1,max=10"`
    ExecutionOrder int      `json:"execution_order,omitempty"`
    DependsOn      []string `json:"depends_on,omitempty"`
    FilePath       string   `json:"file_path,omitempty"`
    CreateFile     bool     `json:"create_file,omitempty"`
    Force          bool     `json:"force,omitempty"`
}
```

**When to use DTOs:**
- Input has more than 3 parameters
- Input has optional parameters
- Input needs to be reusable across CLI and HTTP API
- Input validation is complex

### Step 5: Refactor Command to Call Service

**Goal**: Simplify command to thin wrapper.

**Actions**:
1. Open command file (`internal/cli/commands/task.go`)
2. Find command handler function
3. Replace business logic with service call
4. Keep argument parsing
5. Keep output formatting
6. Add error handling

**Example Refactoring:**

**Before (65 lines):**
```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    taskKey := args[0]

    repoDb, _ := cli.GetDB(ctx)
    repo := repository.NewTaskRepository(repoDb)

    task, _ := repo.GetByKey(ctx, taskKey)
    if task.Status != "todo" && task.Status != "blocked" {
        return fmt.Errorf("cannot start")
    }
    // ... 50 more lines of business logic ...

    cli.Success("Task started")
    return nil
}
```

**After (15 lines):**
```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    taskKey := args[0]
    agentID, _ := cmd.Flags().GetString("agent")

    // Step 2: Call service
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), taskKey, agentID)
    if err != nil {
        return err
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }

    cli.Success(fmt.Sprintf("Started task %s", task.Key))
    return nil
}
```

**Line reduction: 65 → 15 (77% reduction)**

### Step 6: Add Service Accessor (If Needed)

**Goal**: Create global accessor function for CLI commands.

**Actions**:
1. Open `internal/cli/services_global.go`
2. Add accessor function if not exists
3. Wire dependencies (DB, repositories, workflow service)

**Example Accessor:**

```go
// GetTaskService returns a TaskService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func GetTaskService() *services.TaskService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    taskRepo := repository.NewTaskRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewTaskService(taskRepo, workflowSvc, nil, nil)
}
```

### Step 7: Update Tests (Service Tests)

**Goal**: Test service logic with mocked repositories.

**Actions**:
1. Create or open `internal/services/task_service_test.go`
2. Define mock repositories
3. Write test for happy path
4. Write tests for error cases
5. Use table-driven tests for multiple scenarios

**Example Service Test:**

```go
func TestTaskService_StartTask(t *testing.T) {
    // Arrange: Create mocks
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{
                ID:     1,
                Key:    "E07-F01-001",
                Status: "todo",
            }, nil
        },
        UpdateStatusFunc: func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
            assert.Equal(t, int64(1), id)
            assert.Equal(t, models.TaskStatus("in_progress"), status)
            return nil
        },
    }

    mockWorkflow := &MockWorkflowService{
        ValidateTransitionFunc: func(from, to string) error {
            return nil // Allow transition
        },
    }

    svc := services.NewTaskService(mockRepo, mockWorkflow, nil, nil)

    // Act: Start task
    task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

    // Assert: Success
    assert.NoError(t, err)
    assert.NotNil(t, task)
    assert.Equal(t, models.TaskStatus("in_progress"), task.Status)
}
```

### Step 8: Update Tests (CLI Tests - Optional)

**Goal**: Update CLI tests to mock services instead of repositories.

**Actions**:
1. Open or create `internal/cli/commands/task_test.go`
2. Create mock service
3. Test command parsing and output formatting
4. Don't test business logic (that's in service tests)

**Example CLI Test:**

```go
func TestTaskStartCommand(t *testing.T) {
    // Create mock service
    mockSvc := &MockTaskService{
        StartTaskFunc: func(ctx context.Context, key string, agentID string) (*models.Task, error) {
            return &models.Task{
                Key:    "E07-F01-001",
                Status: "in_progress",
            }, nil
        },
    }

    // Inject mock (via test setup helper)
    cli.OverrideTaskService(mockSvc)
    defer cli.ResetServices()

    // Execute command
    err := runTaskStart(cmd, []string{"E07-F01-001"})

    // Assert: Command succeeded
    assert.NoError(t, err)
}
```

### Step 9: Run Validation

**Goal**: Ensure migration didn't break anything.

**Actions**:
1. Run service tests: `go test ./internal/services/ -v`
2. Run CLI tests: `go test ./internal/cli/commands/ -v`
3. Run all tests: `make test`
4. Run linting: `make lint`
5. Run formatting: `make fmt`
6. Build project: `make build`
7. Manual smoke test: Run the actual command with `./bin/shark`

**Validation Checklist:**
- [ ] All service tests pass
- [ ] All CLI tests pass (if exist)
- [ ] All repository tests pass
- [ ] Linting passes (`make lint`)
- [ ] Formatting is correct (`make fmt`)
- [ ] Project builds (`make build`)
- [ ] Manual smoke test works (run actual command)

### Step 10: Clean Up

**Goal**: Remove unused code and improve clarity.

**Actions**:
1. Remove unused repository imports from command file
2. Remove helper functions that moved to service
3. Update command documentation (godoc, Long description)
4. Update command examples if behavior changed
5. Commit changes with clear commit message

**Example Cleanup:**

```go
// Before: Imports repository
import (
    "github.com/jwwelbor/shark-task-manager/internal/repository"
    "github.com/jwwelbor/shark-task-manager/internal/cli"
)

// After: No repository import needed
import (
    "github.com/jwwelbor/shark-task-manager/internal/cli"
)
```

**Commit Message Example:**
```
refactor(cli): migrate task start command to service layer

Extracts business logic from runTaskStart into TaskService.StartTask:
- Workflow validation (status checks)
- Dependency validation
- Status update orchestration
- Status cascade triggering

Command is now a thin wrapper: parse → call → format.

Line reduction: 65 → 15 (77% reduction)

Related: E15-F01 (Service layer architecture refactoring)
```

## Migration Checklist Template

Use this checklist for each command you migrate:

```markdown
## Migration: [Command Name]

- [ ] Step 1: Analyzed current command (identified business logic)
- [ ] Step 2: Designed service method signature
- [ ] Step 3: Implemented service method with business logic
- [ ] Step 4: Created DTOs (if needed)
- [ ] Step 5: Refactored command to call service
- [ ] Step 6: Added service accessor (if needed)
- [ ] Step 7: Wrote service tests with mocks
- [ ] Step 8: Updated CLI tests (optional)
- [ ] Step 9: Ran validation (tests, lint, build)
- [ ] Step 10: Cleaned up unused code

### Metrics
- Lines before: XXX
- Lines after: XXX
- Reduction: XX%
- Business logic extracted: XX lines
- Tests added: XX
```

## Common Patterns

### Pattern 1: Simple Get Operation

**Command**: Just parse key and call service.

```go
func runTaskGet(cmd *cobra.Command, args []string) error {
    taskKey := args[0]

    svc := cli.GetTaskService()
    task, err := svc.GetTask(cmd.Context(), taskKey)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }

    // Format output
    cli.Success(fmt.Sprintf("Task: %s (%s)", task.Title, task.Status))
    return nil
}
```

**Service**: Just retrieve from repository.

```go
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get task %s: %w", key, err)
    }
    return task, nil
}
```

### Pattern 2: List with Filters

**Command**: Parse filters into DTO, call service.

```go
func runTaskList(cmd *cobra.Command, args []string) error {
    filters := parseTaskFilters(cmd, args) // Helper

    svc := cli.GetTaskService()
    tasks, err := svc.ListTasks(cmd.Context(), filters)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(tasks)
    }

    // Table formatting
    return cli.OutputTable(headers, formatTaskRows(tasks))
}
```

**Service**: Apply filters and sorting.

```go
func (s *TaskService) ListTasks(ctx context.Context, filters TaskFilters) ([]*models.Task, error) {
    // Get tasks from repository
    tasks, err := s.repo.List(ctx)
    if err != nil {
        return nil, err
    }

    // Apply filters (business logic)
    var filtered []*models.Task
    for _, t := range tasks {
        if matchesFilters(t, filters) {
            filtered = append(filtered, t)
        }
    }

    // Sort by execution order and priority (business logic)
    sort.Slice(filtered, func(i, j int) bool {
        if filtered[i].ExecutionOrder != filtered[j].ExecutionOrder {
            return filtered[i].ExecutionOrder < filtered[j].ExecutionOrder
        }
        return filtered[i].Priority > filtered[j].Priority
    })

    return filtered, nil
}
```

### Pattern 3: Create with Validation

**Command**: Parse input, call service, format response.

```go
func runTaskCreate(cmd *cobra.Command, args []string) error {
    input := parseCreateTaskInput(cmd, args) // Helper

    svc := cli.GetTaskService()
    task, err := svc.CreateTask(cmd.Context(), input)
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }

    cli.Success(fmt.Sprintf("Created task %s at %s", task.Key, task.FilePath))
    return nil
}
```

**Service**: Validate, generate key, create file, save to DB.

```go
func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, error) {
    // Validate epic exists
    epic, err := s.epicRepo.GetByKey(ctx, input.EpicKey)
    if err != nil {
        return nil, fmt.Errorf("epic not found: %w", err)
    }

    // Validate feature exists
    feature, err := s.featureRepo.GetByKey(ctx, input.FeatureKey)
    if err != nil {
        return nil, fmt.Errorf("feature not found: %w", err)
    }

    // Generate task key (business logic)
    taskKey, err := s.creatorSvc.GenerateTaskKey(ctx, epic.Key, feature.Key)
    if err != nil {
        return nil, err
    }

    // Create task model
    task := &models.Task{
        Key:            taskKey,
        Title:          input.Title,
        Status:         models.TaskStatus(s.workflowSvc.GetDefaultStatus()),
        EpicID:         epic.ID,
        FeatureID:      feature.ID,
        AgentType:      input.AgentType,
        Priority:       input.Priority,
        ExecutionOrder: input.ExecutionOrder,
    }

    // Validate model
    if err := task.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Create file (business logic)
    if err := s.creatorSvc.CreateTaskFile(ctx, task, input.FilePath); err != nil {
        return nil, err
    }

    // Save to database
    if err := s.repo.Create(ctx, task); err != nil {
        return nil, fmt.Errorf("failed to create task: %w", err)
    }

    return task, nil
}
```

## Troubleshooting

### Problem: Command still has 50+ lines after migration

**Cause**: Business logic not fully extracted to service.

**Solution**:
1. Re-analyze command with Step 1 (identify remaining business logic)
2. Move remaining validation/filtering/orchestration to service
3. Keep only: parse → call → format

### Problem: Service method has 10+ parameters

**Cause**: Input not bundled into DTO.

**Solution**:
1. Create DTO struct (Step 4)
2. Replace individual parameters with DTO
3. Example: `CreateTask(ctx, epic, feature, title, agent, priority, order, deps, file, create, force)` → `CreateTask(ctx, CreateTaskInput)`

### Problem: Tests still use real database

**Cause**: Not using mocks in service tests.

**Solution**:
1. Define mock repository (Step 7)
2. Inject mock into service constructor
3. Test service logic, not database queries

### Problem: Service has no error wrapping

**Cause**: Forgot to add business context to errors.

**Solution**:
1. Wrap repository errors: `fmt.Errorf("failed to get task %s: %w", key, err)`
2. Wrap workflow errors: `fmt.Errorf("cannot start task %s: %w", key, err)`
3. Add business context at each layer

### Problem: Command calls repository directly

**Cause**: Forgot to remove direct repository access.

**Solution**:
1. Replace `repo.GetByKey()` with `svc.GetTask()`
2. Remove repository import from command file
3. All repository calls must go through service

## Metrics

Track these metrics for each migration:

| Metric | Before | After | Target |
|--------|--------|-------|--------|
| Lines in command | 120 | 30 | <40 |
| Lines in service | 0 | 95 | >80 |
| Business logic in command | 70 | 0 | 0 |
| Repository calls in command | 5 | 0 | 0 |
| Service tests (count) | 0 | 6 | >5 |
| Test coverage (service) | 0% | 85% | >80% |

## Migration Priority

Migrate commands in this order:

1. **High Priority** (simple, high-value):
   - task get (5 lines of logic)
   - epic get (5 lines of logic)
   - feature get (5 lines of logic)

2. **Medium Priority** (moderate complexity):
   - task list (30 lines of logic)
   - task start (40 lines of logic)
   - task complete (35 lines of logic)

3. **Low Priority** (complex, low-frequency):
   - task create (80 lines of logic)
   - task next (60 lines of logic)
   - task block/unblock (40 lines of logic)

Start with high-priority commands to build confidence and establish patterns.

## Related Documentation

- **Service Design Principles**: `.claude/rules/services/service-design.md`
- **CLI Integration Patterns**: `.claude/rules/services/cli-integration.md`
- **Service Testing Patterns**: `.claude/rules/services/testing.md`
- **HTTP Integration Patterns**: `.claude/rules/services/http-integration.md`
- **Architecture Overview**: `.claude/rules/architecture.md`
