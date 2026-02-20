package main

// This file demonstrates the service wiring pattern for HTTP API handlers.
// Unlike CLI commands which use global accessors (cli.GetTaskService()),
// HTTP servers explicitly construct services at startup and inject them into handlers.
//
// This pattern provides:
// - Explicit dependency graph visible in main()
// - No global state (except DB connection)
// - Easy to test handlers with mock services
// - Clear lifecycle management
//
// Example usage in main.go:
//
//	func main() {
//	    // 1. Initialize database
//	    db, err := repository.InitDB("shark-tasks.db")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer db.Close()
//
//	    // 2. Wire up services
//	    services := WireServices(db, ".")
//
//	    // 3. Create handlers with service dependencies
//	    taskHandler := api.NewTaskHandler(services.TaskService)
//	    featureHandler := api.NewFeatureHandler(services.FeatureService)
//	    epicHandler := api.NewEpicHandler(services.EpicService)
//
//	    // 4. Set up routes
//	    http.HandleFunc("/api/tasks", taskHandler.List)
//	    http.HandleFunc("/api/tasks/start", taskHandler.Start)
//	    // ... more routes
//
//	    // 5. Start server
//	    log.Fatal(http.ListenAndServe(":8080", nil))
//	}

import (
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// ServiceContainer holds all initialized services for HTTP handlers.
// This provides a clean way to pass services to handlers without global state.
type ServiceContainer struct {
	TaskService    *services.TaskService
	FeatureService *services.FeatureService
	EpicService    *services.EpicService
	NoteService    *services.NoteService
	ContextService *services.ContextService
	ResumeService  *services.ResumeService
}

// WireServices constructs all services with their dependencies.
// This is the dependency injection "root" for the HTTP API.
//
// Parameters:
//   - db: database connection (shared across all services)
//   - projectRoot: path to project root for workflow config
//
// Returns:
//   - *ServiceContainer: all services ready to inject into handlers
//
// Design notes:
//   - Constructs services in dependency order
//   - Reuses repositories (no need for separate instances per service)
//   - Workflow service initialized once for all entity services
//   - Optional dependencies (creatorSvc, docRepo, relRepo) set to nil for now
func WireServices(db *repository.DB, projectRoot string) *ServiceContainer {
	// Step 1: Initialize workflow service (shared dependency)
	workflowSvc := workflow.NewService(projectRoot)

	// Step 2: Construct repositories (data access layer)
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	noteRepo := repository.NewEntityNoteRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	// Step 2b: Construct taskcreation.Creator for task file creation
	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	renderer := templates.NewRenderer(templates.NewLoader("")) // use embedded templates
	creatorSvc := taskcreation.NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, projectRoot, workflowSvc)

	// Step 3: Construct domain services (business logic layer)
	taskService := services.NewTaskService(
		taskRepo,    // Task data access
		workflowSvc, // Workflow validation
		creatorSvc,  // Task file creation
		noteRepo,    // Rejection note tracking (implements TaskNoteRepository)
	)

	featureService := services.NewFeatureService(
		featureRepo, // Feature data access
		workflowSvc, // Workflow validation
		nil,         // TODO: Note repository (FeatureNoteRepository interface differs)
		taskRepo,    // Child task counting for warnings
		epicRepo,    // Epic lookup for CreateFeature
	)

	epicService := services.NewEpicService(
		epicRepo,    // Epic data access
		workflowSvc, // Workflow validation
		nil,         // TODO: Note repository (EpicNoteRepository interface differs)
		featureRepo, // Child feature counting for warnings
		taskRepo,    // Task repository for impediment tracking
	)

	// Step 4: Construct supporting services
	noteService := services.NewNoteService(noteRepo, epicRepo, featureRepo, taskRepo)
	contextService := services.NewContextService(epicRepo, featureRepo, taskRepo)
	resumeService := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo)

	// Step 5: Return container with all services
	return &ServiceContainer{
		TaskService:    taskService,
		FeatureService: featureService,
		EpicService:    epicService,
		NoteService:    noteService,
		ContextService: contextService,
		ResumeService:  resumeService,
	}
}

// Example handler structure (for documentation purposes):
//
// type TaskHandler struct {
//     taskService *services.TaskService
// }
//
// func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
//     if taskService == nil {
//         panic("taskService is required")
//     }
//     return &TaskHandler{taskService: taskService}
// }
//
// func (h *TaskHandler) Start(w http.ResponseWriter, r *http.Request) {
//     // 1. Parse request
//     var req struct {
//         TaskKey string `json:"task_key"`
//         AgentID string `json:"agent_id"`
//     }
//     if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//         http.Error(w, "invalid request", http.StatusBadRequest)
//         return
//     }
//
//     // 2. Call service (all business logic in service layer)
//     task, err := h.taskService.StartTask(r.Context(), req.TaskKey, req.AgentID)
//     if err != nil {
//         http.Error(w, err.Error(), http.StatusInternalServerError)
//         return
//     }
//
//     // 3. Format response
//     w.Header().Set("Content-Type", "application/json")
//     json.NewEncoder(w).Encode(task)
// }
//
// Key differences from CLI pattern:
//   - CLI: Uses global accessors (cli.GetTaskService())
//   - HTTP: Explicit injection via NewTaskHandler(taskService)
//   - CLI: Service created per command invocation
//   - HTTP: Service created once at server startup, reused across requests
//   - CLI: Panics on DB failure (fail-fast)
//   - HTTP: Returns errors to client, server keeps running
