---
paths: "cmd/server/**/*"
---

# HTTP API Integration Patterns with Service Layer

This rule is loaded when working with HTTP API server files.

## Overview

HTTP API handlers must be **thin wrappers** that:
1. Parse HTTP request (body, params, query strings, headers)
2. Call a service method
3. Format HTTP response (JSON, status codes, headers)

**All business logic belongs in the service layer**, not in HTTP handlers.

## The Three-Step Pattern

Every HTTP handler follows this exact pattern:

```go
func (h *TaskHandler) MyEndpoint(w http.ResponseWriter, r *http.Request) {
    // Step 1: Parse HTTP request
    key := chi.URLParam(r, "key")
    var req MyRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Step 2: Call service
    result, err := h.taskService.PerformOperation(r.Context(), key, req.Notes)
    if err != nil {
        h.handleError(w, err)
        return
    }

    // Step 3: Format HTTP response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(result)
}
```

**That's it. No conditionals, no business logic, no direct repository calls.**

## Handler Structure

### Handler Type with Service Injection

```go
// TaskHandler handles HTTP requests for task operations
type TaskHandler struct {
    taskService *services.TaskService
}

// NewTaskHandler creates a TaskHandler with injected service
func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
    return &TaskHandler{
        taskService: taskService,
    }
}
```

**Why Inject Services:**
- Testability: Easy to inject mocks in tests
- Flexibility: Can swap implementations without changing handlers
- Clarity: Dependencies are explicit in constructor

### Router Setup

```go
// In cmd/server/main.go
func main() {
    // Initialize dependencies
    db, _ := repository.InitDB()
    taskRepo := repository.NewTaskRepository(db)
    workflowSvc := workflow.NewService(projectRoot)
    taskService := services.NewTaskService(taskRepo, workflowSvc, nil, nil)

    // Create handler with service injection
    taskHandler := api.NewTaskHandler(taskService)

    // Setup routes
    r := chi.NewRouter()
    r.Route("/api/v1/tasks", func(r chi.Router) {
        r.Get("/", taskHandler.ListTasks)
        r.Post("/", taskHandler.CreateTask)
        r.Get("/{key}", taskHandler.GetTask)
        r.Patch("/{key}/start", taskHandler.StartTask)
        r.Patch("/{key}/complete", taskHandler.CompleteTask)
    })

    http.ListenAndServe(":8080", r)
}
```

## HTTP Handler Examples

### Example 1: Get Task (Read Operation)

**Handler (Thin Wrapper):**

```go
// GetTask retrieves a task by key
// GET /api/v1/tasks/{key}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    // Step 1: Parse request
    key := chi.URLParam(r, "key")

    // Step 2: Call service
    task, err := h.taskService.GetTask(r.Context(), key)
    if err != nil {
        h.handleError(w, err)
        return
    }

    // Step 3: Format response
    h.respondJSON(w, http.StatusOK, task)
}
```

**Response:**

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "key": "E07-F01-001",
  "title": "Implement user authentication",
  "status": "in_progress",
  "priority": 5,
  "agent_type": "backend",
  "created_at": "2026-02-17T10:00:00Z",
  "updated_at": "2026-02-17T11:30:00Z"
}
```

### Example 2: Start Task (Status Transition)

**Request DTO:**

```go
type StartTaskRequest struct {
    AgentID string `json:"agent_id"`
}
```

**Handler:**

```go
// StartTask transitions a task to in_progress
// PATCH /api/v1/tasks/{key}/start
func (h *TaskHandler) StartTask(w http.ResponseWriter, r *http.Request) {
    // Step 1: Parse request
    key := chi.URLParam(r, "key")

    var req StartTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Step 2: Call service (all business logic here)
    task, err := h.taskService.StartTask(r.Context(), key, req.AgentID)
    if err != nil {
        h.handleError(w, err)
        return
    }

    // Step 3: Format response
    h.respondJSON(w, http.StatusOK, task)
}
```

**Request:**

```bash
curl -X PATCH http://localhost:8080/api/v1/tasks/E07-F01-001/start \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "agent123"}'
```

**Response:**

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "key": "E07-F01-001",
  "title": "Implement user authentication",
  "status": "in_progress",
  "agent_id": "agent123",
  "updated_at": "2026-02-17T12:00:00Z"
}
```

### Example 3: List Tasks (Query with Filters)

**Request DTO:**

```go
type ListTasksRequest struct {
    EpicKey    string `json:"epic_key"`
    FeatureKey string `json:"feature_key"`
    Status     string `json:"status"`
    AgentType  string `json:"agent_type"`
    ShowAll    bool   `json:"show_all"`
}
```

**Handler:**

```go
// ListTasks retrieves tasks matching filters
// GET /api/v1/tasks?epic=E07&feature=F01&status=todo&agent=backend
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
    // Step 1: Parse query parameters
    filters := services.TaskFilters{
        EpicKey:    r.URL.Query().Get("epic"),
        FeatureKey: r.URL.Query().Get("feature"),
        Status:     r.URL.Query().Get("status"),
        AgentType:  r.URL.Query().Get("agent"),
        ShowAll:    r.URL.Query().Get("show_all") == "true",
    }

    // Step 2: Call service (filtering and sorting logic in service)
    tasks, err := h.taskService.ListTasks(r.Context(), filters)
    if err != nil {
        h.handleError(w, err)
        return
    }

    // Step 3: Format response
    h.respondJSON(w, http.StatusOK, tasks)
}
```

**Request:**

```bash
curl http://localhost:8080/api/v1/tasks?epic=E07&feature=F01&status=todo
```

**Response:**

```json
HTTP/1.1 200 OK
Content-Type: application/json

[
  {
    "key": "E07-F01-001",
    "title": "Task 1",
    "status": "todo",
    "priority": 5
  },
  {
    "key": "E07-F01-002",
    "title": "Task 2",
    "status": "todo",
    "priority": 3
  }
]
```

### Example 4: Create Task (Complex Input)

**Request DTO:**

```go
type CreateTaskRequest struct {
    EpicKey        string   `json:"epic_key" validate:"required"`
    FeatureKey     string   `json:"feature_key" validate:"required"`
    Title          string   `json:"title" validate:"required,min=3"`
    AgentType      string   `json:"agent_type,omitempty"`
    Priority       int      `json:"priority,omitempty" validate:"min=1,max=10"`
    ExecutionOrder int      `json:"execution_order,omitempty"`
    DependsOn      []string `json:"depends_on,omitempty"`
}
```

**Handler:**

```go
// CreateTask creates a new task
// POST /api/v1/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    // Step 1: Parse and validate request
    var req CreateTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Basic validation (structural only - business validation in service)
    if req.Title == "" || req.EpicKey == "" || req.FeatureKey == "" {
        h.respondError(w, http.StatusBadRequest, "Missing required fields")
        return
    }

    // Convert to service input DTO
    input := services.CreateTaskInput{
        EpicKey:        req.EpicKey,
        FeatureKey:     req.FeatureKey,
        Title:          req.Title,
        AgentType:      req.AgentType,
        Priority:       req.Priority,
        ExecutionOrder: req.ExecutionOrder,
        DependsOn:      req.DependsOn,
    }

    // Step 2: Call service (all business logic here)
    task, err := h.taskService.CreateTask(r.Context(), input)
    if err != nil {
        h.handleError(w, err)
        return
    }

    // Step 3: Format response
    h.respondJSON(w, http.StatusCreated, task)
}
```

**Request:**

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "epic_key": "E07",
    "feature_key": "F01",
    "title": "Implement JWT token validation",
    "agent_type": "backend",
    "priority": 8,
    "execution_order": 3
  }'
```

**Response:**

```json
HTTP/1.1 201 Created
Content-Type: application/json
Location: /api/v1/tasks/E07-F01-003

{
  "key": "E07-F01-003",
  "title": "Implement JWT token validation",
  "status": "todo",
  "epic_key": "E07",
  "feature_key": "F01",
  "agent_type": "backend",
  "priority": 8,
  "execution_order": 3,
  "created_at": "2026-02-17T12:15:00Z"
}
```

## Error Handling

### Error Response Format

**Standard Error Response:**

```go
type ErrorResponse struct {
    Error   string   `json:"error"`
    Message string   `json:"message"`
    Details []string `json:"details,omitempty"`
}
```

### Error Handler Helper

```go
// handleError translates service errors to HTTP responses
func (h *TaskHandler) handleError(w http.ResponseWriter, err error) {
    // Check error type and map to status code
    var notFoundErr *repository.NotFoundError
    if errors.As(err, &notFoundErr) {
        h.respondError(w, http.StatusNotFound, fmt.Sprintf("Task not found: %s", notFoundErr.Key))
        return
    }

    var workflowErr *workflow.TransitionError
    if errors.As(err, &workflowErr) {
        h.respondError(w, http.StatusUnprocessableEntity, workflowErr.Error())
        return
    }

    var validationErr *models.ValidationError
    if errors.As(err, &validationErr) {
        h.respondError(w, http.StatusBadRequest, validationErr.Error())
        return
    }

    // Generic server error
    h.respondError(w, http.StatusInternalServerError, "Internal server error")
}

// respondError formats and sends error response
func (h *TaskHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error:   http.StatusText(statusCode),
        Message: message,
    })
}
```

### HTTP Status Code Mapping

| Error Type | HTTP Status Code | Example |
|------------|------------------|---------|
| NotFoundError | 404 Not Found | Task doesn't exist |
| ValidationError | 400 Bad Request | Invalid input format |
| WorkflowError | 422 Unprocessable Entity | Invalid status transition |
| ConflictError | 409 Conflict | Duplicate key |
| DependencyError | 424 Failed Dependency | Dependencies not met |
| Generic Error | 500 Internal Server Error | Database failure |

**Example Error Responses:**

```json
// 404 Not Found
{
  "error": "Not Found",
  "message": "Task not found: E07-F01-999"
}

// 422 Unprocessable Entity
{
  "error": "Unprocessable Entity",
  "message": "Invalid transition from 'todo' to 'completed'"
}

// 400 Bad Request
{
  "error": "Bad Request",
  "message": "Validation failed",
  "details": [
    "title: cannot be empty",
    "priority: must be between 1 and 10"
  ]
}
```

## Response Helpers

### JSON Response Helper

```go
// respondJSON formats and sends JSON response
func (h *TaskHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    if data != nil {
        if err := json.NewEncoder(w).Encode(data); err != nil {
            // Log error but don't send another response (headers already sent)
            log.Printf("Failed to encode JSON response: %v", err)
        }
    }
}
```

### Location Header for Created Resources

```go
// CreateTask creates a new task
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    task, err := h.taskService.CreateTask(r.Context(), input)
    if err != nil {
        h.handleError(w, err)
        return
    }

    // Set Location header for created resource
    w.Header().Set("Location", fmt.Sprintf("/api/v1/tasks/%s", task.Key))
    h.respondJSON(w, http.StatusCreated, task)
}
```

## Middleware Patterns

### Service Injection via Context

```go
// Middleware to inject services into context
func ServiceMiddleware(taskService *services.TaskService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := context.WithValue(r.Context(), "taskService", taskService)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Handler retrieves service from context
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    svc := r.Context().Value("taskService").(*services.TaskService)
    task, _ := svc.GetTask(r.Context(), chi.URLParam(r, "key"))
    h.respondJSON(w, http.StatusOK, task)
}
```

**Note**: Direct injection via handler constructor is preferred over context injection for clarity.

### Request ID Middleware

```go
// RequestIDMiddleware adds request ID to context
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        ctx := context.WithValue(r.Context(), "requestID", requestID)
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Logging Middleware

```go
// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap response writer to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

        next.ServeHTTP(wrapped, r)

        log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

## Testing HTTP Handlers with Services

Handlers are tested with **mocked services**, not real databases.

**Pattern:**

```go
// Test handler with mocked service
func TestTaskHandler_GetTask(t *testing.T) {
    // Create mock service
    mockSvc := &MockTaskService{
        GetTaskFunc: func(ctx context.Context, key string) (*models.Task, error) {
            assert.Equal(t, "E07-F01-001", key)
            return &models.Task{
                Key:    "E07-F01-001",
                Title:  "Test Task",
                Status: "todo",
            }, nil
        },
    }

    // Create handler with mock
    handler := api.NewTaskHandler(mockSvc)

    // Create test request
    req := httptest.NewRequest("GET", "/api/v1/tasks/E07-F01-001", nil)
    req = req.WithContext(chi.NewRouteContext(context.Background(), chi.RouteParams{
        {Key: "key", Value: "E07-F01-001"},
    }))
    rec := httptest.NewRecorder()

    // Execute handler
    handler.GetTask(rec, req)

    // Assert response
    assert.Equal(t, http.StatusOK, rec.Code)

    var task models.Task
    json.NewDecoder(rec.Body).Decode(&task)
    assert.Equal(t, "E07-F01-001", task.Key)
    assert.Equal(t, "Test Task", task.Title)
}

// Test error handling
func TestTaskHandler_GetTask_NotFound(t *testing.T) {
    mockSvc := &MockTaskService{
        GetTaskFunc: func(ctx context.Context, key string) (*models.Task, error) {
            return nil, &repository.NotFoundError{Entity: "task", Key: key}
        },
    }

    handler := api.NewTaskHandler(mockSvc)
    req := httptest.NewRequest("GET", "/api/v1/tasks/E07-F01-999", nil)
    req = req.WithContext(chi.NewRouteContext(context.Background(), chi.RouteParams{
        {Key: "key", Value: "E07-F01-999"},
    }))
    rec := httptest.NewRecorder()

    handler.GetTask(rec, req)

    assert.Equal(t, http.StatusNotFound, rec.Code)

    var errResp api.ErrorResponse
    json.NewDecoder(rec.Body).Decode(&errResp)
    assert.Contains(t, errResp.Message, "E07-F01-999")
}
```

## Handler Responsibility Matrix

| Responsibility | HTTP Handler | Service | Repository |
|----------------|--------------|---------|------------|
| Parse HTTP request | ✅ | ❌ | ❌ |
| Parse JSON body | ✅ | ❌ | ❌ |
| Parse URL params | ✅ | ❌ | ❌ |
| Parse query strings | ✅ | ❌ | ❌ |
| Structural validation | ✅ | ❌ | ❌ |
| Business validation | ❌ | ✅ | ❌ |
| Workflow validation | ❌ | ✅ | ❌ |
| Database queries | ❌ | ❌ | ✅ |
| Transaction management | ❌ | ✅ | ❌ |
| Format JSON response | ✅ | ❌ | ❌ |
| Set HTTP headers | ✅ | ❌ | ❌ |
| Set status codes | ✅ | ❌ | ❌ |
| Error to status mapping | ✅ | ❌ | ❌ |

## API Versioning

**URL-based versioning:**

```go
r.Route("/api/v1", func(r chi.Router) {
    r.Get("/tasks", taskHandler.ListTasks)
    r.Post("/tasks", taskHandler.CreateTask)
})

r.Route("/api/v2", func(r chi.Router) {
    r.Get("/tasks", taskHandlerV2.ListTasks)  // New version
    r.Post("/tasks", taskHandlerV2.CreateTask)
})
```

**Header-based versioning:**

```go
func VersionMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        version := r.Header.Get("API-Version")
        if version == "" {
            version = "v1"  // Default
        }
        ctx := context.WithValue(r.Context(), "apiVersion", version)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Common Mistakes to Avoid

### Mistake 1: Doing Business Logic in Handlers

```go
// ❌ BAD: Validation in handler
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var req CreateTaskRequest
    json.NewDecoder(r.Body).Decode(&req)

    // ❌ Business validation in handler
    if req.Priority < 1 || req.Priority > 10 {
        h.respondError(w, http.StatusBadRequest, "Invalid priority")
        return
    }

    // ❌ Workflow check in handler
    if req.Status != "todo" && req.Status != "in_progress" {
        h.respondError(w, http.StatusBadRequest, "Invalid status")
        return
    }

    task, _ := h.taskService.CreateTask(r.Context(), input)
    h.respondJSON(w, http.StatusCreated, task)
}

// ✅ GOOD: Validation in service
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var req CreateTaskRequest
    json.NewDecoder(r.Body).Decode(&req)

    input := services.CreateTaskInput{/* ... */}
    task, err := h.taskService.CreateTask(r.Context(), input)
    if err != nil {
        h.handleError(w, err)  // Service returns validation errors
        return
    }

    h.respondJSON(w, http.StatusCreated, task)
}
```

### Mistake 2: Calling Repositories Directly

```go
// ❌ BAD: Handler calls repository
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    db, _ := repository.InitDB()
    repo := repository.NewTaskRepository(db)
    task, _ := repo.GetByKey(r.Context(), chi.URLParam(r, "key"))
    h.respondJSON(w, http.StatusOK, task)
}

// ✅ GOOD: Handler calls service
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    task, err := h.taskService.GetTask(r.Context(), chi.URLParam(r, "key"))
    if err != nil {
        h.handleError(w, err)
        return
    }
    h.respondJSON(w, http.StatusOK, task)
}
```

### Mistake 3: Not Handling Errors Properly

```go
// ❌ BAD: Generic error handling
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    task, err := h.taskService.GetTask(r.Context(), chi.URLParam(r, "key"))
    if err != nil {
        // ❌ All errors return 500
        h.respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
    h.respondJSON(w, http.StatusOK, task)
}

// ✅ GOOD: Type-based error handling
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    task, err := h.taskService.GetTask(r.Context(), chi.URLParam(r, "key"))
    if err != nil {
        h.handleError(w, err)  // Maps error types to appropriate status codes
        return
    }
    h.respondJSON(w, http.StatusOK, task)
}
```

### Mistake 4: Exposing Internal Details

```go
// ❌ BAD: Exposing database IDs
type TaskResponse struct {
    ID        int64  `json:"id"`          // ❌ Internal database ID
    EpicID    int64  `json:"epic_id"`     // ❌ Internal foreign key
    FeatureID int64  `json:"feature_id"`  // ❌ Internal foreign key
    Key       string `json:"key"`
    Title     string `json:"title"`
}

// ✅ GOOD: Business keys only
type TaskResponse struct {
    Key        string `json:"key"`         // ✅ Business key
    EpicKey    string `json:"epic_key"`    // ✅ Business key
    FeatureKey string `json:"feature_key"` // ✅ Business key
    Title      string `json:"title"`
}
```

## Related Documentation

- **Service Design**: `.claude/rules/services/service-design.md`
- **Service Testing**: `.claude/rules/services/testing.md`
- **CLI Integration**: `.claude/rules/services/cli-integration.md`
- **Error Handling**: `.claude/rules/go/error-handling.md`
- **Go Patterns**: `.claude/rules/go/patterns.md`
