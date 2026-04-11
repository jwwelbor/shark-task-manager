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
	"fmt"

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
	analytics, err := a.repo.GetSessionAnalyticsByFeature(ctx, featureID, agentType)
	if err != nil || analytics == nil {
		return nil, err
	}
	return &services.SessionAnalytics{
		TotalSessions:          analytics.TotalSessions,
		TotalDuration:          analytics.TotalDuration,
		AverageDuration:        analytics.AverageDuration,
		MedianDuration:         analytics.MedianDuration,
		TasksWithSessions:      analytics.TasksWithSessions,
		TasksWithPauses:        analytics.TasksWithPauses,
		AverageSessionsPerTask: analytics.AverageSessionsPerTask,
		PauseRate:              analytics.PauseRate,
	}, nil
}

func (a *workSessionAdapter) GetSessionAnalyticsByEpic(ctx context.Context, epicID int64, agentType *string) (*services.SessionAnalytics, error) {
	analytics, err := a.repo.GetSessionAnalyticsByEpic(ctx, epicID, agentType)
	if err != nil || analytics == nil {
		return nil, err
	}
	return &services.SessionAnalytics{
		TotalSessions:          analytics.TotalSessions,
		TotalDuration:          analytics.TotalDuration,
		AverageDuration:        analytics.AverageDuration,
		MedianDuration:         analytics.MedianDuration,
		TasksWithSessions:      analytics.TasksWithSessions,
		TasksWithPauses:        analytics.TasksWithPauses,
		AverageSessionsPerTask: analytics.AverageSessionsPerTask,
		PauseRate:              analytics.PauseRate,
	}, nil
}

// taskHistoryAdapter adapts *repository.TaskHistoryRepository to the services.TaskHistoryRepository interface.
// The two HistoryFilters types have identical fields but different package paths, so an adapter is needed.
type taskHistoryAdapter struct {
	repo *repository.TaskHistoryRepository //nolint:staticcheck // Deprecated: will migrate to EntityHistoryRepository
}

func (a *taskHistoryAdapter) GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
	return a.repo.GetHistoryByTaskKey(ctx, taskKey) //nolint:staticcheck // Deprecated: will migrate to EntityHistoryRepository
}

func (a *taskHistoryAdapter) ListWithFilters(ctx context.Context, filters services.HistoryFilters) ([]*models.TaskHistory, error) {
	return a.repo.ListWithFilters(ctx, repository.HistoryFilters{ //nolint:staticcheck // Deprecated: will migrate to EntityHistoryRepository
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
	TaskService       *services.TaskService
	FeatureService    *services.FeatureService
	EpicService       *services.EpicService
	BugService        *services.BugService
	ChangeCardService *services.ChangeCardService
	NoteService       *services.NoteService
	ContextService    *services.ContextService
	ResumeService     *services.ResumeService
	ViewerService     *services.ViewerService
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
	historyRepo := repository.NewTaskHistoryRepository(db) //nolint:staticcheck // Deprecated: will migrate to EntityHistoryRepository

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

	// Step 3b: Construct EntityHistoryRepository for polymorphic history recording
	entityHistoryRepo := repository.NewEntityHistoryRepository(db)
	entitySvc.SetHistoryRepo(entityHistoryRepo)

	// Step 4: Construct entity-specific services
	// Rejection note creation is handled by EntityService (via SetNoteRepo above).
	taskService := services.NewTaskService(taskRepo, entitySvc, creatorSvc)
	taskService.SetEntityHistoryRepo(entityHistoryRepo)

	featureService := services.NewFeatureService(
		featureRepo, entitySvc,
		registry.MustGetRepository(models.EntityTypeFeature),
		taskRepo, epicRepo,
	)
	featureService.SetEntityHistoryRepo(entityHistoryRepo)

	// Wire FeatureService into TaskService for auto-reopen behavior
	taskService.SetFeatureService(featureService)

	// Wire cascade reopen dependencies into both services so that regressions
	// automatically reopen terminal ancestors (AC-T3 / REQ-F-001).
	// TaskService cascade: task regression → reopen feature + epic.
	taskService.SetCascadeDeps(db, featureRepo, epicRepo, entityHistoryRepo, entityHistoryRepo)
	// FeatureService cascade: feature regression → reopen epic.
	featureService.SetCascadeDeps(db, epicRepo, entityHistoryRepo, entityHistoryRepo)

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

	bugService := services.NewBugService(
		bugRepoAdapter,
		entitySvc,
		registry.MustGetRepository(models.EntityTypeBug),
		epicRepo, featureRepo, taskRepo,
		projectRoot,
	)

	changeCardService := services.NewChangeCardService(
		changeCardRepoAdapter,
		entitySvc,
		registry.MustGetRepository(models.EntityTypeChange),
		epicRepo, featureRepo,
		projectRoot,
	)

	noteService, err := services.NewNoteService(noteRepo, registry)
	if err != nil {
		panic(fmt.Sprintf("failed to create NoteService: %v", err))
	}
	contextService, err := services.NewContextService(registry)
	if err != nil {
		panic(fmt.Sprintf("failed to create ContextService: %v", err))
	}
	resumeService, err := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo, registry)
	if err != nil {
		panic(fmt.Sprintf("failed to create ResumeService: %v", err))
	}

	// Step 5: Construct ViewerService for the read-only dashboard API.
	// statusCalc is optional and passed as nil; it can be wired in a later iteration
	// when weighted-progress display is added to the viewer summary endpoint.
	viewerService := services.NewViewerService(
		epicRepo,
		featureRepo,
		taskRepo,
		bugRepoAdapter,
		changeCardRepoAdapter,
		entityHistoryRepo,
		workflowSvc,
		nil, // statusCalc: optional, not required for current viewer endpoints
		projectRoot,
	)

	// Step 6: Return container with all services
	_ = historyRepo // available for future wiring
	return &ServiceContainer{
		TaskService:       taskService,
		FeatureService:    featureService,
		EpicService:       epicService,
		BugService:        bugService,
		ChangeCardService: changeCardService,
		NoteService:       noteService,
		ContextService:    contextService,
		ResumeService:     resumeService,
		ViewerService:     viewerService,
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
