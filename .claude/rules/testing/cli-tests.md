---
paths: "internal/cli/**/*_test.go"
---

# CLI Command Testing Patterns

This rule is loaded when working with CLI test files.

## ❌ NEVER USE REAL DATABASE - USE MOCKS

CLI command tests verify command logic, argument parsing, and output formatting.

## Requirements

- Create mock repository interfaces
- Inject mocks into command handlers
- Test command behavior without database
- Verify JSON/table output formatting
- Test error handling

## Mock Repository Pattern

```go
// Define mock interface
type MockTaskRepository struct {
    CreateFunc   func(ctx context.Context, task *models.Task) error
    GetFunc      func(ctx context.Context, key string) (*models.Task, error)
    UpdateFunc   func(ctx context.Context, task *models.Task) error
    DeleteFunc   func(ctx context.Context, key string) error
    ListFunc     func(ctx context.Context, filters map[string]interface{}) ([]*models.Task, error)
}

func (m *MockTaskRepository) Create(ctx context.Context, task *models.Task) error {
    if m.CreateFunc != nil {
        return m.CreateFunc(ctx, task)
    }
    return nil
}

func (m *MockTaskRepository) Get(ctx context.Context, key string) (*models.Task, error) {
    if m.GetFunc != nil {
        return m.GetFunc(ctx, key)
    }
    return nil, fmt.Errorf("not found")
}

// ... implement other methods ...
```

## Testing Command Execution

```go
func TestTaskCreateCommand(t *testing.T) {
    // Create mock repository
    mockRepo := &MockTaskRepository{
        CreateFunc: func(ctx context.Context, task *models.Task) error {
            // Set ID to simulate database behavior
            task.ID = 123
            return nil
        },
    }

    // Test command logic (without actually running cobra command)
    ctx := context.Background()
    task := &models.Task{
        Key:    "E07-F01-001",
        Title:  "Test Task",
        Status: "todo",
    }

    err := mockRepo.Create(ctx, task)
    if err != nil {
        t.Errorf("Create() error = %v", err)
    }

    if task.ID == 0 {
        t.Error("expected ID to be set")
    }
}
```

## Testing Command Arguments

```go
func TestTaskCommandArgs(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        wantErr bool
    }{
        {
            name:    "valid 3-arg format",
            args:    []string{"E07", "F01", "Task Title"},
            wantErr: false,
        },
        {
            name:    "valid 2-arg format",
            args:    []string{"E07-F01", "Task Title"},
            wantErr: false,
        },
        {
            name:    "missing title",
            args:    []string{"E07", "F01"},
            wantErr: true,
        },
        {
            name:    "invalid epic key",
            args:    []string{"INVALID", "F01", "Task Title"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Parse and validate arguments
            err := validateTaskCreateArgs(tt.args)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateTaskCreateArgs() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Testing JSON Output

```go
func TestTaskGetCommandJSON(t *testing.T) {
    // Create mock
    mockTask := &models.Task{
        Key:    "E07-F01-001",
        Title:  "Test Task",
        Status: "todo",
    }

    mockRepo := &MockTaskRepository{
        GetFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return mockTask, nil
        },
    }

    // Simulate command execution with --json flag
    cli.GlobalConfig.JSON = true

    // Get task
    task, err := mockRepo.Get(context.Background(), "E07-F01-001")
    if err != nil {
        t.Fatalf("Get() error = %v", err)
    }

    // Verify JSON serialization
    jsonData, err := json.Marshal(task)
    if err != nil {
        t.Fatalf("json.Marshal() error = %v", err)
    }

    // Parse back and verify
    var parsedTask models.Task
    if err := json.Unmarshal(jsonData, &parsedTask); err != nil {
        t.Fatalf("json.Unmarshal() error = %v", err)
    }

    if parsedTask.Key != mockTask.Key {
        t.Errorf("expected key %s, got %s", mockTask.Key, parsedTask.Key)
    }
}
```

## Testing Error Handling

```go
func TestTaskGetCommandNotFound(t *testing.T) {
    // Create mock that returns not found error
    mockRepo := &MockTaskRepository{
        GetFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return nil, &repository.NotFoundError{
                Entity: "task",
                Key:    key,
            }
        },
    }

    // Execute command
    task, err := mockRepo.Get(context.Background(), "E07-F01-999")

    // Verify error
    if err == nil {
        t.Fatal("expected error, got nil")
    }

    var notFoundErr *repository.NotFoundError
    if !errors.As(err, &notFoundErr) {
        t.Errorf("expected NotFoundError, got %T", err)
    }

    if task != nil {
        t.Error("expected nil task")
    }
}
```

## Testing with Multiple Mocks

```go
func TestTaskCompleteCommand(t *testing.T) {
    // Mock task repository
    mockTaskRepo := &MockTaskRepository{
        GetFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{
                Key:    "E07-F01-001",
                Status: "in_progress",
            }, nil
        },
        UpdateFunc: func(ctx context.Context, task *models.Task) error {
            return nil
        },
    }

    // Mock history repository
    mockHistoryRepo := &MockTaskHistoryRepository{
        CreateFunc: func(ctx context.Context, history *models.TaskHistory) error {
            return nil
        },
    }

    // Execute command logic
    ctx := context.Background()
    task, err := mockTaskRepo.Get(ctx, "E07-F01-001")
    if err != nil {
        t.Fatalf("Get() error = %v", err)
    }

    // Verify task is in correct state
    if task.Status != "in_progress" {
        t.Errorf("expected status 'in_progress', got %s", task.Status)
    }

    // Update status
    task.Status = "ready_for_review"
    if err := mockTaskRepo.Update(ctx, task); err != nil {
        t.Errorf("Update() error = %v", err)
    }

    // Record history
    history := &models.TaskHistory{
        TaskID:    task.ID,
        FromStatus: "in_progress",
        ToStatus:   "ready_for_review",
    }
    if err := mockHistoryRepo.Create(ctx, history); err != nil {
        t.Errorf("Create() error = %v", err)
    }
}
```

## Testing Validation

```go
func TestTaskValidation(t *testing.T) {
    tests := []struct {
        name    string
        task    *models.Task
        wantErr bool
    }{
        {
            name: "valid task",
            task: &models.Task{
                Key:      "E07-F01-001",
                Title:    "Valid Task",
                Status:   "todo",
                Priority: 5,
            },
            wantErr: false,
        },
        {
            name: "empty title",
            task: &models.Task{
                Key:      "E07-F01-001",
                Title:    "",
                Status:   "todo",
                Priority: 5,
            },
            wantErr: true,
        },
        {
            name: "invalid priority",
            task: &models.Task{
                Key:      "E07-F01-001",
                Title:    "Task",
                Status:   "todo",
                Priority: 15, // Invalid: must be 1-10
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.task.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Best Practices

1. **Never use real database** - Always use mocks
2. **Test behavior, not implementation** - Focus on inputs/outputs
3. **Use table-driven tests** - Cover multiple scenarios efficiently
4. **Test error paths** - Ensure errors are handled correctly
5. **Mock at the right level** - Mock repositories, not database
6. **Keep tests fast** - No I/O, no network, no database
7. **Make tests deterministic** - No random data, no time dependencies
8. **Clean up global state** - Reset `cli.GlobalConfig` after tests

## Example: Complete Command Test

```go
func TestTaskStartCommand(t *testing.T) {
    // Save original config
    origJSON := cli.GlobalConfig.JSON
    defer func() { cli.GlobalConfig.JSON = origJSON }()

    tests := []struct {
        name        string
        taskStatus  string
        expectError bool
        errorType   interface{}
    }{
        {
            name:        "start todo task",
            taskStatus:  "todo",
            expectError: false,
        },
        {
            name:        "task already in progress",
            taskStatus:  "in_progress",
            expectError: true,
            errorType:   &InvalidStateError{},
        },
        {
            name:        "task not found",
            taskStatus:  "",
            expectError: true,
            errorType:   &NotFoundError{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create mock based on test case
            var mockRepo *MockTaskRepository
            if tt.taskStatus == "" {
                // Not found scenario
                mockRepo = &MockTaskRepository{
                    GetFunc: func(ctx context.Context, key string) (*models.Task, error) {
                        return nil, &NotFoundError{Entity: "task", Key: key}
                    },
                }
            } else {
                // Found scenario
                mockRepo = &MockTaskRepository{
                    GetFunc: func(ctx context.Context, key string) (*models.Task, error) {
                        return &models.Task{
                            Key:    key,
                            Status: tt.taskStatus,
                        }, nil
                    },
                    UpdateFunc: func(ctx context.Context, task *models.Task) error {
                        return nil
                    },
                }
            }

            // Execute command logic
            err := startTask(context.Background(), mockRepo, "E07-F01-001")

            // Verify result
            if (err != nil) != tt.expectError {
                t.Errorf("startTask() error = %v, expectError %v", err, tt.expectError)
            }

            if tt.expectError && tt.errorType != nil {
                if !errors.As(err, &tt.errorType) {
                    t.Errorf("expected error type %T, got %T", tt.errorType, err)
                }
            }
        })
    }
}
```
