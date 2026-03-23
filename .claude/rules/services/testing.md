---
paths: "internal/services/**/*_test.go"
---

# Service Testing Patterns

## Golden Rule

**Services are tested with MOCKED repositories. NEVER use real database in service tests.**

## Mock Pattern (Function Fields)

Define mocks with function fields — inline test data per case, no mocking framework needed.

```go
type MockTaskRepository struct {
    GetByKeyFunc     func(ctx context.Context, key string) (*models.Task, error)
    CreateFunc       func(ctx context.Context, task *models.Task) error
    UpdateFunc       func(ctx context.Context, task *models.Task) error
    UpdateStatusFunc func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error
    ListByFeatureFunc func(ctx context.Context, featureID int64) ([]*models.Task, error)
}

func (m *MockTaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    if m.GetByKeyFunc != nil { return m.GetByKeyFunc(ctx, key) }
    return nil, fmt.Errorf("GetByKey not implemented in mock")
}
// ... same delegation pattern for other methods
```

```go
type MockWorkflowService struct {
    ValidateTransitionFunc func(from, to string) error
    GetDefaultStatusFunc   func() string
}

func (m *MockWorkflowService) ValidateTransition(from, to string) error {
    if m.ValidateTransitionFunc != nil { return m.ValidateTransitionFunc(from, to) }
    return nil
}
```

Put shared mocks in `mocks_test.go` within the services package.

## Example: Status Transition Test

```go
func TestTaskService_StartTask(t *testing.T) {
    mockRepo := &MockTaskRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return &models.Task{ID: 1, Key: "E07-F01-001", Status: "todo"}, nil
        },
        UpdateStatusFunc: func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
            assert.Equal(t, int64(1), id)
            assert.Equal(t, models.TaskStatus("in_progress"), status)
            assert.Equal(t, "agent123", *agent)
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
    assert.NotNil(t, task)
    assert.Equal(t, models.TaskStatus("in_progress"), task.Status)
}
```

## Table-Driven Tests

```go
func TestTaskService_StartTask_Scenarios(t *testing.T) {
    tests := []struct {
        name          string
        initialStatus string
        workflowErr   error
        expectError   bool
        errorContains string
    }{
        {"start from todo", "todo", nil, false, ""},
        {"start from blocked", "blocked", nil, false, ""},
        {"cannot start from completed", "completed",
            fmt.Errorf("invalid transition"), true, "invalid transition"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &MockTaskRepository{
                GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
                    return &models.Task{ID: 1, Key: key, Status: models.TaskStatus(tt.initialStatus)}, nil
                },
                UpdateStatusFunc: func(ctx context.Context, id int64, status models.TaskStatus, agent, notes *string) error {
                    return nil
                },
            }
            mockWorkflow := &MockWorkflowService{
                ValidateTransitionFunc: func(from, to string) error { return tt.workflowErr },
            }

            svc := services.NewTaskService(mockRepo, mockWorkflow, nil, nil)
            task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

            if tt.expectError {
                assert.Error(t, err)
                assert.Nil(t, task)
                if tt.errorContains != "" { assert.Contains(t, err.Error(), tt.errorContains) }
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, task)
            }
        })
    }
}
```

## Key Testing Patterns

1. **Verify mock interactions** — assert parameters passed to repository methods
2. **Capture arguments** — use variables to capture what service sends to repos
3. **Simulate errors** — return `fmt.Errorf(...)` from mock to test error paths
4. **Test graceful degradation** — pass nil for optional dependencies, verify no panic
5. **Test context cancellation** — pass canceled context, verify error propagation

## Anti-Patterns

| Anti-Pattern | Fix |
|---|---|
| Real database in service tests | Use mock repositories |
| Testing SQL logic in service tests | That belongs in repository tests |
| No assertions on mock calls | Always verify parameters passed to mocks |
| Testing only happy path | Cover all error paths and edge cases |

## File Structure

```
internal/services/
├── task_service.go / task_service_test.go
├── feature_service.go / feature_service_test.go
├── epic_service.go / epic_service_test.go
├── task_dto.go
└── mocks_test.go          # Shared mock definitions
```

## Coverage Goals

- Service logic: 80%+
- Error paths: 100%
- Edge cases (nil, empty, boundaries): 100%

## Running Tests

```bash
go test ./internal/services/...              # All service tests
go test ./internal/services/ -run TestTaskService  # Specific service
go test ./internal/services/ -cover          # With coverage
```
