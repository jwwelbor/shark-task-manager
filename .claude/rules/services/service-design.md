---
paths: "internal/services/**/*"
---

# Service Layer Design

**Key Principle**: Fat services, thin controllers, dumb repositories.

Services orchestrate business logic, enforce rules, validate workflows, and coordinate repositories. They bridge entry points (CLI, HTTP) and data access.

## Design Rules

### 1. Single Responsibility

One service per domain aggregate:
- **TaskService** — lifecycle, transitions, dependencies
- **FeatureService** — progress, task summaries
- **EpicService** — feature rollups, impediments
- **NoteService**, **ContextService**, **ResumeService** — cross-cutting

### 2. Explicit Repository Interfaces

Define tailored interfaces per service, not generic CRUD:

```go
type TaskRepository interface {
    Create(ctx context.Context, task *models.Task) error
    GetByKey(ctx context.Context, key string) (*models.Task, error)
    Update(ctx context.Context, task *models.Task) error
    ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
    UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error
}
```

### 3. Method Signatures

- `context.Context` as first parameter (never store in structs)
- Accept business-level inputs (task key, not database ID)
- Return domain models (`*models.Task`), not primitives
- Use DTOs for >3 parameters:

```go
type CreateTaskInput struct {
    EpicKey, FeatureKey, Title, AgentType string
    Priority, ExecutionOrder              int
    DependsOn                             []string
}

func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, error)
```

| Use Case | DTO | Domain Model |
|----------|:---:|:------------:|
| Create/Update inputs | x | |
| Filter/query params | x | |
| Return values | | x |

### 4. Error Wrapping

Wrap errors with business context at each layer:

```go
func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("failed to start task %s: %w", key, err)
    }
    if err := s.workflowSvc.ValidateTransition(string(task.Status), "in_progress"); err != nil {
        return nil, fmt.Errorf("cannot start task %s in status %s: %w", key, task.Status, err)
    }
    // ...
}
```

### 5. Constructor Injection

```go
type TaskService struct {
    repo        TaskRepository
    workflowSvc *workflow.Service
    creatorSvc  *taskcreation.Creator  // Can be nil
    noteRepo    TaskNoteRepository     // Can be nil
}

func NewTaskService(repo TaskRepository, workflowSvc *workflow.Service,
    creatorSvc *taskcreation.Creator, noteRepo TaskNoteRepository) *TaskService {
    return &TaskService{
        repo: repo, workflowSvc: workflowSvc.ForLevel(workflow.LevelTask),
        creatorSvc: creatorSvc, noteRepo: noteRepo,
    }
}
```

Optional dependencies degrade gracefully:
```go
if s.noteRepo != nil && reason != "" {
    _ = s.noteRepo.CreateRejectionNote(ctx, ...)
}
```

### 6. No Output Formatting

Services return domain models. Never format for CLI/HTTP — that's the caller's job.

### 7. Transaction Ownership

Services own transaction boundaries. Inject `*repository.DB` when transactions are needed:

```go
func (s *TaskService) CompleteTaskWithNotes(ctx context.Context, key, notes string) (*models.Task, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil { return nil, fmt.Errorf("failed to begin transaction: %w", err) }
    defer tx.Rollback()

    task, err := s.repo.UpdateStatusTx(ctx, tx, key, "ready_for_review", &notes)
    if err != nil { return nil, err }
    if err := s.historyRepo.CreateTx(ctx, tx, task.ID, "in_progress", "ready_for_review"); err != nil { return nil, err }

    if err := tx.Commit(); err != nil { return nil, fmt.Errorf("failed to commit: %w", err) }
    return task, nil
}
```

### 8. Workflow Validation

Use `workflow.Service` — never hardcode status lists:

```go
// BAD:  return status == "todo" || status == "blocked"
// GOOD: return s.workflowSvc.IsValidTransition(status, "in_progress")
```

### 9. Entry-Point Agnostic

Same service works for CLI, HTTP, tests — no Cobra/HTTP dependencies in services.

## Naming Conventions

- Services: `TaskService`, `FeatureService` (noun + Service)
- Lifecycle methods: `StartTask`, `CompleteTask`, `ApproveTask`
- Queries: `ListTasks`, `GetDependencyTree`
- Validation: `ValidateStatus`, `ValidateDependencies`

## Anti-Patterns

### Fat Controllers
```go
// BAD: Business logic in CLI command
func runTaskStart(cmd *cobra.Command, args []string) error {
    repo := repository.NewTaskRepository(db)
    task, _ := repo.GetByKey(ctx, args[0])
    if task.Status != "todo" { return fmt.Errorf("...") }  // Business logic!
    repo.UpdateStatus(ctx, task.ID, "in_progress", nil, nil)
}

// GOOD: Thin wrapper
func runTaskStart(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), args[0], agentID)
    if err != nil { return err }
    cli.Success(fmt.Sprintf("Task %s started", task.Key))
    return nil
}
```

### Smart Repositories
Repositories must NOT contain workflow validation, progress calculation, or business rules — only data access.

### Bypassing Services
Commands must NOT call repository methods directly. Always go through a service.

### Anemic Services
Services should add value (validation, orchestration, error wrapping) — not just proxy repository calls.

### God Services
One service per aggregate. Don't create a single `ProjectService` with 100 methods.

## Related Documentation

- **Testing**: `.claude/rules/services/testing.md`
- **CLI Integration**: `.claude/rules/services/cli-integration.md`
- **HTTP Integration**: `.claude/rules/services/http-integration.md`
- **Architecture**: `.claude/rules/architecture.md`
