---
paths: "internal/services/**/*_test.go"
---

# Service Testing Patterns

This rule is loaded when working with service test files.

## Golden Rule

**Services are tested with MOCKED repositories. NEVER use real database in service tests.**

## Why Mock Repositories

| Benefit | Description |
|---------|-------------|
| **Speed** | In-memory mocks are 100x faster than database operations |
| **Isolation** | Test service logic independently of database behavior |
| **Determinism** | No flaky tests from database state or concurrency |
| **Clarity** | Mock expectations document exact repository interactions |
| **Flexibility** | Easy to simulate edge cases (errors, timeouts, race conditions) |

## Mock Repository Pattern

### Define Mock Interface

```go
// MockTaskRepository implements TaskRepository interface for testing
type MockTaskRepository struct {
    // Function fields for each interface method
    GetByKeyFunc     func(ctx context.Context, key string) (*models.Task, error)
    CreateFunc       func(ctx context.Context, task *models.Task) error
    UpdateFunc       func(ctx context.Context, task *models.Task) error
    UpdateStatusFunc func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error
    ListByFeatureFunc func(ctx context.Context, featureID int64) ([]*models.Task, error)
}

// Implement interface methods by delegating to function fields
func (m *MockTaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    if m.GetByKeyFunc != nil {
        return m.GetByKeyFunc(ctx, key)
    }
    return nil, fmt.Errorf("GetByKey not implemented in mock")
}

func (m *MockTaskRepository) Create(ctx context.Context, task *models.Task) error {
    if m.CreateFunc != nil {
        return m.CreateFunc(ctx, task)
    }
    return fmt.Errorf("Create not implemented in mock")
}

func (m *MockTaskRepository) Update(ctx context.Context, task *models.Task) error {
    if m.UpdateFunc != nil {
        return m.UpdateFunc(ctx, task)
    }
    return fmt.Errorf("Update not implemented in mock")
}

func (m *MockTaskRepository) UpdateStatus(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
    if m.UpdateStatusFunc != nil {
        return m.UpdateStatusFunc(ctx, id, status, agent, notes)
    }
    return fmt.Errorf("UpdateStatus not implemented in mock")
}

func (m *MockTaskRepository) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
    if m.ListByFeatureFunc != nil {
        return m.ListByFeatureFunc(ctx, featureID)
    }
    return nil, fmt.Errorf("ListByFeature not implemented in mock")
}
```

**Benefits of Function Field Pattern:**
- Inline test data per test case
- Easy to verify call counts and parameters
- No need for complex mocking frameworks
- Test code is self-contained and readable

### Mock Workflow Service

```go
// MockWorkflowService implements workflow validation for testing
type MockWorkflowService struct {
    ValidateStatusFunc     func(status string) error
    ValidateTransitionFunc func(from, to string) error
    GetDefaultStatusFunc   func() string
    IsTerminalStatusFunc   func(status string) bool
}

func (m *MockWorkflowService) ValidateStatus(status string) error {
    if m.ValidateStatusFunc != nil {
        return m.ValidateStatusFunc(status)
    }
    return nil
}

func (m *MockWorkflowService) ValidateTransition(from, to string) error {
    if m.ValidateTransitionFunc != nil {
        return m.ValidateTransitionFunc(from, to)
    }
    return nil
}

func (m *MockWorkflowService) GetDefaultStatus() string {
    if m.GetDefaultStatusFunc != nil {
        return m.GetDefaultStatusFunc()
    }
    return "todo"
}

func (m *MockWorkflowService) IsTerminalStatus(status string) bool {
    if m.IsTerminalStatusFunc != nil {
        return m.IsTerminalStatusFunc(status)
    }
    return status == "completed"
}
```

## Service Testing Examples

### Example 1: Test Get Operation (Simple Mock)

```go
func TestTaskService_GetTask(t *testing.T) {
    // Arrange: Create mock repository
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            // Verify service passes correct key
            assert.Equal(t, "E07-F01-001", key)

            // Return mock task
            return &models.Task{
                Key:    "E07-F01-001",
                Title:  "Test Task",
                Status: "todo",
            }, nil
        },
    }

    // Create service with mock
    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    // Act: Call service method
    task, err := svc.GetTask(context.Background(), "E07-F01-001")

    // Assert: Verify result
    assert.NoError(t, err)
    assert.NotNil(t, task)
    assert.Equal(t, "E07-F01-001", task.Key)
    assert.Equal(t, "Test Task", task.Title)
    assert.Equal(t, models.TaskStatus("todo"), task.Status)
}
```

### Example 2: Test Status Transition (Workflow Validation)

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
            // Verify correct parameters
            assert.Equal(t, int64(1), id)
            assert.Equal(t, models.TaskStatus("in_progress"), status)
            assert.NotNil(t, agent)
            assert.Equal(t, "agent123", *agent)
            return nil
        },
    }

    mockWorkflow := &MockWorkflowService{
        ValidateTransitionFunc: func(from, to string) error {
            // Verify transition validation is called
            assert.Equal(t, "todo", from)
            assert.Equal(t, "in_progress", to)
            return nil // Allow transition
        },
    }

    svc := services.NewTaskService(mockRepo, mockWorkflow, nil, nil)

    // Act: Start task
    task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

    // Assert: Task started successfully
    assert.NoError(t, err)
    assert.NotNil(t, task)
    assert.Equal(t, models.TaskStatus("in_progress"), task.Status)
}
```

### Example 3: Test Error Handling

```go
func TestTaskService_StartTask_NotFound(t *testing.T) {
    // Arrange: Mock returns NotFoundError
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return nil, &repository.NotFoundError{
                Entity: "task",
                Key:    key,
            }
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    // Act: Attempt to start non-existent task
    task, err := svc.StartTask(context.Background(), "E07-F01-999", "agent123")

    // Assert: Error is propagated correctly
    assert.Error(t, err)
    assert.Nil(t, task)

    var notFoundErr *repository.NotFoundError
    assert.True(t, errors.As(err, &notFoundErr))
    assert.Equal(t, "task", notFoundErr.Entity)
    assert.Equal(t, "E07-F01-999", notFoundErr.Key)
}

func TestTaskService_StartTask_InvalidTransition(t *testing.T) {
    // Arrange: Task in completed status (can't start)
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{
                ID:     1,
                Key:    "E07-F01-001",
                Status: "completed", // Already completed
            }, nil
        },
    }

    mockWorkflow := &MockWorkflowService{
        ValidateTransitionFunc: func(from, to string) error {
            // Reject transition
            return fmt.Errorf("invalid transition from '%s' to '%s'", from, to)
        },
    }

    svc := services.NewTaskService(mockRepo, mockWorkflow, nil, nil)

    // Act: Attempt invalid transition
    task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

    // Assert: Workflow error returned
    assert.Error(t, err)
    assert.Nil(t, task)
    assert.Contains(t, err.Error(), "invalid transition")
    assert.Contains(t, err.Error(), "completed")
}
```

### Example 4: Test List with Filtering (Complex Logic)

```go
func TestTaskService_ListTasks_WithFilters(t *testing.T) {
    // Arrange: Mock returns all tasks, service filters
    mockRepo := &MockTaskRepository{
        ListFunc: func(ctx context.Context) ([]*models.Task, error) {
            // Repository returns all tasks
            return []*models.Task{
                {Key: "E07-F01-001", Status: "todo", AgentType: "backend"},
                {Key: "E07-F01-002", Status: "in_progress", AgentType: "frontend"},
                {Key: "E07-F01-003", Status: "completed", AgentType: "backend"},
                {Key: "E07-F01-004", Status: "todo", AgentType: "qa"},
            }, nil
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    // Act: List tasks with filters
    tasks, err := svc.ListTasks(context.Background(), services.TaskFilters{
        Status:    "todo",
        AgentType: "backend",
    })

    // Assert: Filtering logic works correctly
    assert.NoError(t, err)
    assert.Len(t, tasks, 1) // Only one task matches both filters
    assert.Equal(t, "E07-F01-001", tasks[0].Key)
    assert.Equal(t, models.TaskStatus("todo"), tasks[0].Status)
    assert.Equal(t, "backend", tasks[0].AgentType)
}

func TestTaskService_ListTasks_ShowAll(t *testing.T) {
    mockRepo := &MockTaskRepository{
        ListFunc: func(ctx context.Context) ([]*models.Task, error) {
            return []*models.Task{
                {Key: "E07-F01-001", Status: "todo"},
                {Key: "E07-F01-002", Status: "completed"},
                {Key: "E07-F01-003", Status: "completed"},
            }, nil
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    // Act: List with ShowAll=false (default behavior)
    tasksFiltered, err := svc.ListTasks(context.Background(), services.TaskFilters{
        ShowAll: false,
    })

    // Assert: Completed tasks hidden
    assert.NoError(t, err)
    assert.Len(t, tasksFiltered, 1)
    assert.Equal(t, "E07-F01-001", tasksFiltered[0].Key)

    // Act: List with ShowAll=true
    tasksAll, err := svc.ListTasks(context.Background(), services.TaskFilters{
        ShowAll: true,
    })

    // Assert: All tasks returned
    assert.NoError(t, err)
    assert.Len(t, tasksAll, 3)
}
```

### Example 5: Test Dependency Validation

```go
func TestTaskService_ValidateDependencies(t *testing.T) {
    // Arrange: Mock repository returns task with dependencies
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{
                Key:       "E07-F01-002",
                DependsOn: []string{"E07-F01-001"}, // Has dependency
            }, nil
        },
        GetTaskDependentsFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
            if taskKey == "E07-F01-001" {
                return []*models.Task{
                    {Key: "E07-F01-001", Status: "completed"}, // Dependency satisfied
                }, nil
            }
            return nil, nil
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    // Act: Validate dependencies
    err := svc.ValidateDependencies(context.Background(), "E07-F01-002", "in_progress")

    // Assert: Dependencies are met
    assert.NoError(t, err)
}

func TestTaskService_ValidateDependencies_Blocked(t *testing.T) {
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{
                Key:       "E07-F01-002",
                DependsOn: []string{"E07-F01-001"},
            }, nil
        },
        GetTaskDependentsFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
            if taskKey == "E07-F01-001" {
                return []*models.Task{
                    {Key: "E07-F01-001", Status: "todo"}, // Dependency NOT satisfied
                }, nil
            }
            return nil, nil
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    // Act: Validate dependencies
    err := svc.ValidateDependencies(context.Background(), "E07-F01-002", "in_progress")

    // Assert: Dependency error returned
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "E07-F01-001")
    assert.Contains(t, err.Error(), "not completed")
}
```

### Example 6: Table-Driven Tests

```go
func TestTaskService_StartTask_Scenarios(t *testing.T) {
    tests := []struct {
        name           string
        initialStatus  string
        workflowError  error
        expectedStatus models.TaskStatus
        expectError    bool
        errorContains  string
    }{
        {
            name:           "start from todo",
            initialStatus:  "todo",
            workflowError:  nil,
            expectedStatus: "in_progress",
            expectError:    false,
        },
        {
            name:           "start from blocked",
            initialStatus:  "blocked",
            workflowError:  nil,
            expectedStatus: "in_progress",
            expectError:    false,
        },
        {
            name:          "cannot start from completed",
            initialStatus: "completed",
            workflowError: fmt.Errorf("invalid transition from 'completed' to 'in_progress'"),
            expectError:   true,
            errorContains: "invalid transition",
        },
        {
            name:          "cannot start from in_progress",
            initialStatus: "in_progress",
            workflowError: fmt.Errorf("task already in progress"),
            expectError:   true,
            errorContains: "already in progress",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange: Create mocks per test case
            mockRepo := &MockTaskRepository{
                GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
                    return &models.Task{
                        ID:     1,
                        Key:    "E07-F01-001",
                        Status: models.TaskStatus(tt.initialStatus),
                    }, nil
                },
                UpdateStatusFunc: func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
                    return nil
                },
            }

            mockWorkflow := &MockWorkflowService{
                ValidateTransitionFunc: func(from, to string) error {
                    return tt.workflowError
                },
            }

            svc := services.NewTaskService(mockRepo, mockWorkflow, nil, nil)

            // Act: Start task
            task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

            // Assert: Check result matches expected
            if tt.expectError {
                assert.Error(t, err)
                assert.Nil(t, task)
                if tt.errorContains != "" {
                    assert.Contains(t, err.Error(), tt.errorContains)
                }
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, task)
                assert.Equal(t, tt.expectedStatus, task.Status)
            }
        })
    }
}
```

## Testing Patterns

### Pattern 1: Verify Repository Calls

```go
func TestTaskService_CreateTask_CallsRepository(t *testing.T) {
    var capturedTask *models.Task

    mockRepo := &MockTaskRepository{
        CreateFunc: func(ctx context.Context, task *models.Task) error {
            // Capture task that was passed to repository
            capturedTask = task
            return nil
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    input := services.CreateTaskInput{
        EpicKey:    "E07",
        FeatureKey: "F01",
        Title:      "Test Task",
        Priority:   8,
    }

    task, err := svc.CreateTask(context.Background(), input)

    // Assert: Verify service called repository with correct data
    assert.NoError(t, err)
    assert.NotNil(t, capturedTask)
    assert.Equal(t, "Test Task", capturedTask.Title)
    assert.Equal(t, 8, capturedTask.Priority)
}
```

### Pattern 2: Simulate Repository Errors

```go
func TestTaskService_GetTask_DatabaseError(t *testing.T) {
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            // Simulate database connection error
            return nil, fmt.Errorf("database connection failed")
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    task, err := svc.GetTask(context.Background(), "E07-F01-001")

    // Assert: Error is propagated
    assert.Error(t, err)
    assert.Nil(t, task)
    assert.Contains(t, err.Error(), "database connection failed")
}
```

### Pattern 3: Test Context Cancellation

```go
func TestTaskService_GetTask_ContextCanceled(t *testing.T) {
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            // Check if context is canceled
            if ctx.Err() != nil {
                return nil, ctx.Err()
            }
            return &models.Task{Key: key}, nil
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    // Create canceled context
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately

    task, err := svc.GetTask(ctx, "E07-F01-001")

    // Assert: Context cancellation is handled
    assert.Error(t, err)
    assert.Nil(t, task)
    assert.Equal(t, context.Canceled, err)
}
```

### Pattern 4: Test Graceful Degradation

```go
func TestTaskService_BlockTask_NoteRepoNil(t *testing.T) {
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{
                ID:     1,
                Key:    "E07-F01-001",
                Status: "in_progress",
            }, nil
        },
        UpdateFunc: func(ctx context.Context, task *models.Task) error {
            assert.Equal(t, models.TaskStatus("blocked"), task.Status)
            return nil
        },
    }

    // Create service WITHOUT note repository
    svc := services.NewTaskService(mockRepo, nil, nil, nil)

    // Act: Block task (should work even without note repo)
    task, err := svc.BlockTask(context.Background(), "E07-F01-001", "Waiting on API")

    // Assert: Task blocked successfully (rejection note skipped gracefully)
    assert.NoError(t, err)
    assert.Equal(t, models.TaskStatus("blocked"), task.Status)
}
```

## Testing Anti-Patterns

### Anti-Pattern 1: Using Real Database

```go
// ❌ BAD: Service test using real database
func TestTaskService_StartTask(t *testing.T) {
    db := test.GetTestDB() // Real database
    repo := repository.NewTaskRepository(db)
    svc := services.NewTaskService(repo, nil, nil, nil)

    // This is a repository integration test, not a service unit test
}

// ✅ GOOD: Service test using mock
func TestTaskService_StartTask(t *testing.T) {
    mockRepo := &MockTaskRepository{...}
    svc := services.NewTaskService(mockRepo, nil, nil, nil)
    // Test service logic in isolation
}
```

### Anti-Pattern 2: Testing Repository Logic in Service Test

```go
// ❌ BAD: Testing repository's SQL queries
func TestTaskService_GetTask(t *testing.T) {
    // Don't test that repository constructs correct SQL
    // That's the repository's responsibility
}

// ✅ GOOD: Testing service orchestration
func TestTaskService_StartTask(t *testing.T) {
    // Test that service:
    // - Calls GetByKey to retrieve task
    // - Validates status transition via workflow service
    // - Calls UpdateStatus with correct parameters
}
```

### Anti-Pattern 3: Not Verifying Mock Interactions

```go
// ❌ BAD: Mock returns data but interactions not verified
func TestTaskService_StartTask(t *testing.T) {
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{Key: key, Status: "todo"}, nil
        },
        UpdateStatusFunc: func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
            // No assertions - just return nil
            return nil
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)
    svc.StartTask(context.Background(), "E07-F01-001", "agent123")
}

// ✅ GOOD: Verify mock was called with correct parameters
func TestTaskService_StartTask(t *testing.T) {
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            assert.Equal(t, "E07-F01-001", key) // Verify key
            return &models.Task{ID: 1, Key: key, Status: "todo"}, nil
        },
        UpdateStatusFunc: func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
            assert.Equal(t, int64(1), id)                         // Verify ID
            assert.Equal(t, models.TaskStatus("in_progress"), status) // Verify status
            assert.Equal(t, "agent123", *agent)                   // Verify agent
            return nil
        },
    }

    svc := services.NewTaskService(mockRepo, nil, nil, nil)
    task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")
    assert.NoError(t, err)
    assert.Equal(t, models.TaskStatus("in_progress"), task.Status)
}
```

## Test Organization

### File Structure

```
internal/services/
├── task_service.go           # Service implementation
├── task_service_test.go      # Service tests
├── task_dto.go               # DTOs
├── feature_service.go
├── feature_service_test.go
├── epic_service.go
├── epic_service_test.go
└── mocks_test.go             # Shared mock definitions
```

### Mock Reusability

Define commonly-used mocks in `mocks_test.go`:

```go
// mocks_test.go
package services

// MockTaskRepository is shared across all service tests
type MockTaskRepository struct {
    GetByKeyFunc     func(ctx context.Context, key string) (*models.Task, error)
    CreateFunc       func(ctx context.Context, task *models.Task) error
    UpdateFunc       func(ctx context.Context, task *models.Task) error
    UpdateStatusFunc func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error
}

// ... interface implementations ...

// MockWorkflowService is shared across all service tests
type MockWorkflowService struct {
    ValidateStatusFunc     func(status string) error
    ValidateTransitionFunc func(from, to string) error
}

// ... interface implementations ...
```

## Running Service Tests

```bash
# Run all service tests
go test ./internal/services/...

# Run specific service tests
go test ./internal/services/ -run TestTaskService

# Run with coverage
go test ./internal/services/ -cover

# Run with verbose output
go test ./internal/services/ -v

# Run specific test
go test ./internal/services/ -run TestTaskService_StartTask
```

## Test Coverage Goals

- **Service logic**: 80%+ coverage
- **Error paths**: 100% coverage (all error scenarios tested)
- **Edge cases**: 100% coverage (nil checks, empty lists, boundary values)
- **Happy paths**: 100% coverage (all successful scenarios)

## Related Documentation

- **Service Design**: `.claude/rules/services/service-design.md`
- **CLI Integration**: `.claude/rules/services/cli-integration.md`
- **HTTP Integration**: `.claude/rules/services/http-integration.md`
- **Testing Architecture**: `.claude/rules/testing/architecture.md`
- **Repository Testing**: `.claude/rules/testing/repository-tests.md`
- **CLI Testing**: `.claude/rules/testing/cli-tests.md`
