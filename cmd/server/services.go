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
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// Compile-time interface satisfaction checks for adapters.
// These adapters are prepared for full HTTP API wiring (F02).
var (
	_ services.WorkSessionRepository = (*workSessionAdapter)(nil)
	_ services.TaskHistoryRepository = (*taskHistoryAdapter)(nil)
)

// workSessionAdapter adapts *repository.WorkSessionRepository to the services.WorkSessionRepository interface.
// The repository returns *repository.SessionStats but the service interface expects *services.WorkSessionStats.
type workSessionAdapter struct {
	repo *repository.WorkSessionRepository
}

func (a *workSessionAdapter) GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error) {
	return a.repo.GetByTaskID(ctx, taskID)
}

func (a *workSessionAdapter) GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*services.WorkSessionStats, error) {
	stats, err := a.repo.GetSessionStatsByTaskID(ctx, taskID)
	if err != nil || stats == nil {
		return nil, err
	}
	return &services.WorkSessionStats{
		TotalSessions:   stats.TotalSessions,
		TotalDuration:   stats.TotalDuration,
		AverageDuration: stats.AverageDuration,
		MedianDuration:  stats.MedianDuration,
		ActiveSession:   stats.ActiveSession,
	}, nil
}

func (a *workSessionAdapter) GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error) {
	return a.repo.GetActiveSessionByTaskID(ctx, taskID)
}

func (a *workSessionAdapter) GetSessionAnalyticsByFeature(ctx context.Context, featureID int64, agentType *string) (*services.SessionAnalytics, error) {
	return a.repo.GetSessionAnalyticsByFeature(ctx, featureID, agentType)
}

func (a *workSessionAdapter) GetSessionAnalyticsByEpic(ctx context.Context, epicID int64, agentType *string) (*services.SessionAnalytics, error) {
	return a.repo.GetSessionAnalyticsByEpic(ctx, epicID, agentType)
}

// taskHistoryAdapter adapts *repository.TaskHistoryRepository to the services.TaskHistoryRepository interface.
// The two HistoryFilters types have identical fields but different package paths, so an adapter is needed.
type taskHistoryAdapter struct {
	repo *repository.TaskHistoryRepository
}

func (a *taskHistoryAdapter) GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
	return a.repo.GetHistoryByTaskKey(ctx, taskKey)
}

func (a *taskHistoryAdapter) ListWithFilters(ctx context.Context, filters services.HistoryFilters) ([]*models.TaskHistory, error) {
	return a.repo.ListWithFilters(ctx, repository.HistoryFilters{
		Agent:      filters.Agent,
		Since:      filters.Since,
		EpicKey:    filters.EpicKey,
		FeatureKey: filters.FeatureKey,
		OldStatus:  filters.OldStatus,
		NewStatus:  filters.NewStatus,
		Limit:      filters.Limit,
		Offset:     filters.Offset,
	})
}

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
	taskRepo := repository.NewTaskRepositoryWithWorkflow(db, workflowSvc.GetWorkflow())
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	noteRepo := repository.NewEntityNoteRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	// Step 2b: Construct taskcreation.Creator for task file creation
	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	renderer := templates.NewRenderer(templates.NewLoader("")) // use embedded templates
	creatorSvc := taskcreation.NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, projectRoot, workflowSvc)

	// Step 3: Construct EntityService and EntityRegistry (shared infrastructure)
	entitySvc := services.NewEntityService(workflowSvc)
	entitySvc.SetNoteRepo(noteRepo) // Wire rejection note creation into EntityService
	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeEpic, services.NewEpicRepositoryAdapter(epicRepo))
	registry.Register(models.EntityTypeFeature, services.NewFeatureRepositoryAdapter(featureRepo))
	registry.Register(models.EntityTypeTask, services.NewTaskRepositoryAdapter(taskRepo))
	bugRepoAdapter := repository.NewBugRepository(db)
	registry.Register(models.EntityTypeBug, services.NewBugRepositoryAdapter(bugRepoAdapter))
	changeCardRepoAdapter := repository.NewChangeCardRepository(db)
	registry.Register(models.EntityTypeChange, services.NewChangeCardRepositoryAdapter(changeCardRepoAdapter))

	// Step 4: Construct entity-specific services
	// Rejection note creation is handled by EntityService (via SetNoteRepo above).
	taskService := services.NewTaskService(taskRepo, entitySvc, creatorSvc)

	featureService := services.NewFeatureService(
		featureRepo, entitySvc,
		registry.MustGetRepository(models.EntityTypeFeature),
		taskRepo, epicRepo,
	)

	epicService := services.NewEpicService(
		epicRepo,  // Epic data access
		entitySvc, // Shared transition logic
		registry.MustGetRepository(models.EntityTypeEpic), // Polymorphic adapter
		featureRepo, // Child feature counting
		taskRepo,    // Task repository for impediment tracking
	)

	// Wire the analytics sub-service explicitly to avoid lazy-init on every call.
	analyticsSvc := services.NewEpicAnalyticsService(epicRepo, taskRepo)
	epicService.SetAnalyticsService(analyticsSvc)

	noteService := services.NewNoteService(noteRepo, registry)
	contextService := services.NewContextService(registry)
	resumeService := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo, registry)

	// Step 5: Return container with all services
	_ = historyRepo // available for future wiring
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
//     task, err := h.taskService.AdvanceTaskStatus(r.Context(), req.TaskKey)
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
