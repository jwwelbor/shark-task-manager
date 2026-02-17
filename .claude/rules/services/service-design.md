---
paths: "internal/services/**/*"
---

# Service Layer Design Principles

This rule is loaded when working with service layer files.

## Overview

The service layer is the **primary business logic layer** in Shark Task Manager. Services orchestrate operations, enforce business rules, validate workflows, and coordinate between repositories. They are the bridge between entry points (CLI commands, HTTP API handlers) and data access (repositories).

**Key Principle**: Fat services, thin controllers, dumb repositories.

## Service Design Principles

### 1. Single Responsibility

Each service is responsible for one domain aggregate:

- **TaskService** - Task lifecycle, status transitions, dependency validation, blocking/unblocking
- **FeatureService** - Feature status transitions, progress rollup, task summaries
- **EpicService** - Epic status transitions, feature rollups, impediment tracking
- **NoteService** - Rejection notes and entity annotations
- **ContextService** - Entity context retrieval and aggregation
- **ResumeService** - Resume entity work with full context

**Anti-Pattern**: Creating a "service" that just wraps one repository method with no added logic.

### 2. Explicit Interfaces Over Generic Patterns

Services define **explicit repository interfaces** tailored to their needs, not generic CRUD interfaces.

**Good (Explicit):**
```go
// TaskRepository defines exactly what TaskService needs
type TaskRepository interface {
    // CRUD operations
    Create(ctx context.Context, task *models.Task) error
    GetByKey(ctx context.Context, key string) (*models.Task, error)
    Update(ctx context.Context, task *models.Task) error

    // Query operations specific to tasks
    ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
    ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)

    // Dependency operations (task-specific)
    GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error)

    // Status operations (repository handles DB update, service adds business logic)
    UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus,
                 agent *string, notes *string) error
}

// Concrete *repository.TaskRepository implements this interface
```

**Bad (Generic):**
```go
// DON'T: Generic repository interface loses type safety and clarity
type Repository[T any] interface {
    Create(ctx context.Context, entity T) error
    GetByID(ctx context.Context, id int64) (T, error)
    Update(ctx context.Context, entity T) error
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context, filters map[string]interface{}) ([]T, error)
}

// Hard to understand what operations are available
// Hard to mock specific behaviors in tests
// Loses compile-time type checking
```

**Why Explicit is Better:**
- Clear contract between service and repository
- Easy to mock in tests (only implement what you need)
- Self-documenting (interface shows exactly what operations exist)
- Compile-time safety (can't call methods that don't exist)
- No runtime reflection or type assertions

### 3. Context as First Parameter

All service methods accept `context.Context` as the first parameter for cancellation, timeouts, and request-scoped values.

**Pattern:**
```go
func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
    // Use ctx for repository calls
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, err
    }

    // Service orchestration here...
    return task, nil
}
```

**Never store context in struct fields** - it's request-scoped and should flow through call chains.

### 4. Business-Level Inputs and Outputs

Service methods accept **business-level inputs** (task key, not ID; human-readable values) and return **domain models**, not database primitives.

**Good:**
```go
func (s *TaskService) CompleteTask(ctx context.Context, key string, notes string) (*models.Task, error) {
    // ✅ Accepts task key (E07-F01-001), not database ID
    // ✅ Returns domain model *models.Task
    // ✅ Wraps errors with business context
}
```

**Bad:**
```go
func (s *TaskService) CompleteTask(ctx context.Context, id int64, notes string) error {
    // ❌ Exposes database ID (internal detail)
    // ❌ Returns error only (no updated task)
    // ❌ Caller can't get updated task without extra query
}
```

**Data Transfer Objects (DTOs):**

Use DTOs for complex inputs to avoid parameter explosion:

```go
// Good - DTO bundles related fields
type CreateTaskInput struct {
    EpicKey        string
    FeatureKey     string
    Title          string
    AgentType      string
    Priority       int
    ExecutionOrder int
    DependsOn      []string
    FilePath       string
    CreateFile     bool
    Force          bool
}

func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, error) {
    // Validation and creation logic
}

// Bad - parameter explosion, hard to maintain
func (s *TaskService) CreateTask(ctx context.Context, epicKey, featureKey, title, agentType string,
                                  priority, executionOrder int, dependsOn []string,
                                  filePath string, createFile, force bool) (*models.Task, error) {
    // Too many parameters, hard to call
}
```

**When to Use DTOs vs Domain Models:**

| Use Case | Use DTO | Use Domain Model |
|----------|---------|------------------|
| Create operations (partial data) | ✅ CreateTaskInput | ❌ |
| Update operations (partial fields) | ✅ TaskUpdates | ❌ |
| Filter/query operations | ✅ TaskFilters | ❌ |
| Return values | ❌ | ✅ *models.Task |
| Repository storage | ❌ | ✅ *models.Task |

### 5. Error Wrapping with Business Context

Services wrap repository errors with **business context** at each layer:

**Pattern:**
```go
func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
    // Repository layer returns technical error: "failed to get task: sql: no rows"
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        // Service wraps with business context
        return nil, fmt.Errorf("failed to start task %s: %w", key, err)
    }

    // Workflow validation adds business rule context
    if err := s.workflowSvc.ValidateTransition(string(task.Status), "in_progress"); err != nil {
        return nil, fmt.Errorf("cannot start task %s in status %s: %w", key, task.Status, err)
    }

    // Status update adds operation context
    if err := s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agentID, nil); err != nil {
        return nil, fmt.Errorf("failed to update task %s status: %w", key, err)
    }

    return task, nil
}
```

**Error Chain Example:**
```
Command: "unable to mark task as complete: ..."
Service: "failed to complete task E07-F01-001: ..."
Workflow: "invalid transition from 'todo' to 'completed'"
```

See `.claude/rules/go/error-handling.md` for complete error patterns.

### 6. Dependency Injection via Constructors

Services receive dependencies through **explicit constructor parameters**, not global singletons or service locators.

**Pattern:**
```go
// Service struct with dependency fields
type TaskService struct {
    repo        TaskRepository              // Required: data access
    workflowSvc *workflow.Service           // Required: workflow validation
    creatorSvc  *taskcreation.Creator       // Optional: task creation
    noteRepo    TaskNoteRepository          // Optional: rejection notes
}

// Constructor with explicit dependencies
func NewTaskService(
    repo TaskRepository,                    // Interface, not concrete type
    workflowSvc *workflow.Service,
    creatorSvc *taskcreation.Creator,       // Can be nil
    noteRepo TaskNoteRepository,            // Can be nil
) *TaskService {
    return &TaskService{
        repo:        repo,
        workflowSvc: workflowSvc.ForLevel(workflow.LevelTask), // Scope to task level
        creatorSvc:  creatorSvc,
        noteRepo:    noteRepo,
    }
}
```

**Benefits:**
- **Compile-time safety** - can't construct service without required dependencies
- **Testability** - easy to inject mocks
- **Clarity** - dependencies are explicit in constructor signature
- **No magic** - no reflection, no runtime dependency resolution

**Graceful Degradation for Optional Dependencies:**

```go
func (s *TaskService) BlockTask(ctx context.Context, key string, reason string) (*models.Task, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, err
    }

    // Update status (required operation)
    task.Status = models.TaskStatus("blocked")
    if err := s.repo.Update(ctx, task); err != nil {
        return nil, err
    }

    // Create rejection note (optional - degrade gracefully if noteRepo is nil)
    if s.noteRepo != nil && reason != "" {
        _ = s.noteRepo.CreateRejectionNote(ctx, models.EntityTypeTask, task.ID,
                                           0, string(task.Status), "blocked", reason, "", nil)
    }

    return task, nil
}
```

### 7. No Output Formatting in Services

Services **never format output** for CLI or HTTP. They return domain models and errors. Formatting is the caller's responsibility.

**Good (Service):**
```go
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    return s.repo.GetByKey(ctx, key)
    // ✅ Returns domain model
    // ✅ Caller decides how to format (JSON, table, etc.)
}
```

**Bad (Service):**
```go
func (s *TaskService) GetTask(ctx context.Context, key string, format string) (string, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return "", err
    }

    // ❌ Service shouldn't know about CLI output formatting
    if format == "json" {
        bytes, _ := json.Marshal(task)
        return string(bytes), nil
    }

    // ❌ Service shouldn't know about table formatting
    return fmt.Sprintf("Task: %s (%s)", task.Title, task.Status), nil
}
```

**Separation of Concerns:**
- **Service**: Business logic, validation, orchestration
- **CLI Command**: Argument parsing, output formatting (JSON/table), exit codes
- **HTTP Handler**: Request parsing, response formatting (JSON), status codes

### 8. Transaction Ownership

Services own **transaction boundaries**, not repositories. Repositories can accept `*sql.Tx` for participation in service-managed transactions.

**Pattern:**
```go
func (s *TaskService) CompleteTaskWithNotes(ctx context.Context, key string, notes string) (*models.Task, error) {
    // Service starts transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Rollback if not committed

    // Update task status (participates in transaction)
    task, err := s.repo.UpdateStatusTx(ctx, tx, key, "ready_for_review", &notes)
    if err != nil {
        return nil, err // Transaction rolled back automatically
    }

    // Create history record (participates in same transaction)
    if err := s.historyRepo.CreateTx(ctx, tx, task.ID, "in_progress", "ready_for_review"); err != nil {
        return nil, err // Transaction rolled back automatically
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }

    return task, nil
}
```

**Why Services Own Transactions:**
- Services know which operations must be atomic
- Repositories focus on single-table operations
- Business invariants enforced at service level
- Easier to test (mock transaction behavior at service level)

### 9. Workflow Validation at Service Layer

Services use `workflow.Service` to validate status transitions and enforce business rules.

**Pattern:**
```go
func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, err
    }

    // ✅ Validate transition via workflow service
    if err := s.workflowSvc.ValidateTransition(string(task.Status), "in_progress"); err != nil {
        return nil, fmt.Errorf("cannot start task: %w", err)
    }

    // ✅ Service enforces business rule (check dependencies met)
    if err := s.ValidateDependencies(ctx, key, "in_progress"); err != nil {
        return nil, fmt.Errorf("dependencies not met: %w", err)
    }

    // Perform update
    return s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agentID, nil)
}
```

**Never hardcode statuses in services:**

```go
// ❌ BAD: Hardcoded status list
func (s *TaskService) CanStart(status string) bool {
    return status == "todo" || status == "blocked"
}

// ✅ GOOD: Use workflow service
func (s *TaskService) CanStart(status string) bool {
    return s.workflowSvc.IsValidTransition(status, "in_progress")
}
```

### 10. Reusability Across Entry Points

Services are designed to be **entry-point agnostic** - they work equally well for CLI, HTTP API, gRPC, or test harnesses.

**Example: Same Service, Multiple Entry Points**

```go
// CLI Command (entry point 1)
func runTaskStart(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), args[0], agentID)
    if err != nil {
        return err
    }
    cli.Success(fmt.Sprintf("Task %s started", task.Key))
    return nil
}

// HTTP API Handler (entry point 2)
func (h *TaskHandler) StartTask(w http.ResponseWriter, r *http.Request) {
    key := chi.URLParam(r, "key")
    agentID := r.Header.Get("X-Agent-ID")

    task, err := h.taskService.StartTask(r.Context(), key, agentID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    json.NewEncoder(w).Encode(task)
}

// Test Harness (entry point 3)
func TestTaskService_StartTask(t *testing.T) {
    mockRepo := &MockTaskRepository{...}
    svc := services.NewTaskService(mockRepo, mockWorkflow, nil, nil)

    task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")
    assert.NoError(t, err)
    assert.Equal(t, "in_progress", task.Status)
}
```

**Same service logic, different presentation layers.**

## Service Naming Conventions

### Service Names

- Suffix: `Service` (e.g., `TaskService`, `FeatureService`, `EpicService`)
- Noun-based, not verb-based (e.g., `TaskService`, not `TaskManager`)
- One service per domain aggregate

### Method Names

- **CRUD**: `Create`, `Get`, `Update`, `Delete` (not `CreateTask`, `GetTask` - the service name already implies "Task")
- **Lifecycle**: `StartTask`, `CompleteTask`, `ApproveTask`, `ReopenTask` (verb + entity for clarity)
- **Queries**: `ListTasks`, `GetNextTask`, `GetDependencyTree` (verb + plural or descriptor)
- **Validation**: `ValidateStatus`, `ValidateDependencies` (verb + what is validated)
- **Actions**: `BlockTask`, `UnblockTask`, `TransitionStatus` (verb + entity + action)

**Example:**

```go
// Good naming
type TaskService struct { ... }

func (s *TaskService) Create(ctx context.Context, input CreateTaskInput) (*models.Task, error)
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error)
func (s *TaskService) ListTasks(ctx context.Context, filters TaskFilters) ([]*models.Task, error)
func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error)
func (s *TaskService) ValidateDependencies(ctx context.Context, key string, targetStatus string) error
```

## Service Testing Strategy

Services are tested with **mocked repositories**, not real databases.

**Pattern:**

```go
// Define mock in test file
type MockTaskRepository struct {
    GetByKeyFunc func(ctx context.Context, key string) (*models.Task, error)
    UpdateStatusFunc func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error
}

func (m *MockTaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    if m.GetByKeyFunc != nil {
        return m.GetByKeyFunc(ctx, key)
    }
    return nil, fmt.Errorf("not implemented")
}

func (m *MockTaskRepository) UpdateStatus(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
    if m.UpdateStatusFunc != nil {
        return m.UpdateStatusFunc(ctx, id, status, agent, notes)
    }
    return fmt.Errorf("not implemented")
}

// Test service logic with mocks
func TestTaskService_StartTask(t *testing.T) {
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
            assert.Equal(t, "todo", from)
            assert.Equal(t, "in_progress", to)
            return nil
        },
    }

    svc := services.NewTaskService(mockRepo, mockWorkflow, nil, nil)

    task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")
    assert.NoError(t, err)
    assert.Equal(t, "in_progress", task.Status)
}
```

See `.claude/rules/services/testing.md` for complete testing patterns.

## Common Anti-Patterns to Avoid

### 1. Fat Controllers (Business Logic in Commands)

**❌ Bad:**
```go
// CLI command with business logic
func runTaskStart(cmd *cobra.Command, args []string) error {
    db, _ := cli.GetDB(cmd.Context())
    repo := repository.NewTaskRepository(db)

    // ❌ Validation logic in command
    task, err := repo.GetByKey(cmd.Context(), args[0])
    if task.Status != "todo" && task.Status != "blocked" {
        return fmt.Errorf("cannot start task in status %s", task.Status)
    }

    // ❌ Business logic in command
    if err := repo.UpdateStatus(cmd.Context(), task.ID, "in_progress", nil, nil); err != nil {
        return err
    }

    cli.Success("Task started")
    return nil
}
```

**✅ Good:**
```go
// Thin command wrapper calling service
func runTaskStart(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), args[0], agentID)
    if err != nil {
        return err
    }
    cli.Success(fmt.Sprintf("Task %s started", task.Key))
    return nil
}
```

### 2. Smart Repositories (Business Logic in Repositories)

**❌ Bad:**
```go
// Repository with business logic
func (r *TaskRepository) CompleteTask(ctx context.Context, key string) (*models.Task, error) {
    task, _ := r.GetByKey(ctx, key)

    // ❌ Workflow validation in repository
    if task.Status != "in_progress" {
        return nil, fmt.Errorf("can only complete in-progress tasks")
    }

    // ❌ Progress calculation in repository
    feature, _ := r.GetFeature(ctx, task.FeatureID)
    progress := r.CalculateFeatureProgress(ctx, feature.ID)

    // ❌ Business rule in repository
    if progress < 50 {
        return nil, fmt.Errorf("feature must be at least 50% complete")
    }

    task.Status = "completed"
    return r.Update(ctx, task)
}
```

**✅ Good:**
```go
// Dumb repository (pure data access)
func (r *TaskRepository) UpdateStatus(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
    query := `UPDATE tasks SET status = ?, agent = ?, notes = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
    _, err := r.db.ExecContext(ctx, query, status, agent, notes, id)
    return err
}

// Service has business logic
func (s *TaskService) CompleteTask(ctx context.Context, key string, notes string) (*models.Task, error) {
    task, _ := s.repo.GetByKey(ctx, key)

    // ✅ Workflow validation in service
    if err := s.workflowSvc.ValidateTransition(string(task.Status), "completed"); err != nil {
        return nil, err
    }

    // ✅ Business rules in service
    if err := s.ValidateDependencies(ctx, key, "completed"); err != nil {
        return nil, err
    }

    // Repository does data access only
    return s.repo.UpdateStatus(ctx, task.ID, "completed", nil, &notes)
}
```

### 3. Bypassing Services (Commands Calling Repositories Directly)

**❌ Bad:**
```go
func runTaskList(cmd *cobra.Command, args []string) error {
    db, _ := cli.GetDB(cmd.Context())
    repo := repository.NewTaskRepository(db)

    // ❌ Command calls repository directly
    tasks, err := repo.List(cmd.Context())
    if err != nil {
        return err
    }

    // ❌ Filtering logic in command
    var filtered []*models.Task
    for _, t := range tasks {
        if t.Status != "completed" {
            filtered = append(filtered, t)
        }
    }

    cli.OutputJSON(filtered)
    return nil
}
```

**✅ Good:**
```go
func runTaskList(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()

    // ✅ Command calls service
    tasks, err := svc.ListTasks(cmd.Context(), services.TaskFilters{
        ShowAll: false, // Service handles filtering
    })
    if err != nil {
        return err
    }

    cli.OutputJSON(tasks)
    return nil
}
```

### 4. Anemic Services (Just Wrapping Repository Calls)

**❌ Bad:**
```go
// Service adds no value
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    return s.repo.GetByKey(ctx, key) // Just a wrapper, no business logic
}

func (s *TaskService) UpdateTask(ctx context.Context, task *models.Task) error {
    return s.repo.Update(ctx, task) // Just a wrapper, no validation
}
```

**✅ Good:**
```go
// Service adds business value
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to get task %s: %w", key, err) // Error wrapping
    }

    // Validate status is still valid (config might have changed)
    if err := s.workflowSvc.ValidateStatus(string(task.Status)); err != nil {
        return nil, fmt.Errorf("task %s has invalid status: %w", key, err)
    }

    return task, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, key string, updates TaskUpdates) (*models.Task, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, err
    }

    // ✅ Validation logic
    if updates.Priority != nil && (*updates.Priority < 1 || *updates.Priority > 10) {
        return nil, fmt.Errorf("priority must be between 1 and 10")
    }

    // ✅ Apply updates (business rule: only update non-nil fields)
    if updates.Title != nil {
        task.Title = *updates.Title
    }
    if updates.Priority != nil {
        task.Priority = *updates.Priority
    }

    // ✅ Model validation
    if err := task.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    return s.repo.Update(ctx, task)
}
```

### 5. God Services (One Service Doing Everything)

**❌ Bad:**
```go
// One service for all entities
type ProjectService struct {
    epicRepo    EpicRepository
    featureRepo FeatureRepository
    taskRepo    TaskRepository
    noteRepo    NoteRepository
}

func (s *ProjectService) CreateEpic(...) (*models.Epic, error)
func (s *ProjectService) CreateFeature(...) (*models.Feature, error)
func (s *ProjectService) CreateTask(...) (*models.Task, error)
func (s *ProjectService) GetEpic(...) (*models.Epic, error)
func (s *ProjectService) GetFeature(...) (*models.Feature, error)
func (s *ProjectService) GetTask(...) (*models.Task, error)
// ... 100 more methods
```

**✅ Good:**
```go
// Separate services per aggregate
type EpicService struct {
    repo EpicRepository
    workflowSvc *workflow.Service
}

type FeatureService struct {
    repo FeatureRepository
    workflowSvc *workflow.Service
}

type TaskService struct {
    repo TaskRepository
    workflowSvc *workflow.Service
    creatorSvc *taskcreation.Creator
}

// Each service focuses on one domain aggregate
```

## Related Documentation

- **Testing Services**: `.claude/rules/services/testing.md`
- **CLI Integration**: `.claude/rules/services/cli-integration.md`
- **HTTP Integration**: `.claude/rules/services/http-integration.md`
- **Error Handling**: `.claude/rules/go/error-handling.md`
- **Go Patterns**: `.claude/rules/go/patterns.md`
- **Architecture Overview**: `.claude/rules/architecture.md`
