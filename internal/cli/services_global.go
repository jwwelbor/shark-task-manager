package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/validation"
	"github.com/jwwelbor/shark-task-manager/internal/view"
)

var (
	globalRegistry *services.EntityRegistry
	registryOnce   sync.Once

	globalEntityService *services.EntityService
	entityServiceOnce   sync.Once

	globalActionService config.ActionService
	actionServiceOnce   sync.Once
	actionServiceErr    error

	globalNoteService *services.NoteService
	noteServiceOnce   sync.Once
	noteServiceErr    error

	globalContextService *services.ContextService
	contextServiceOnce   sync.Once

	globalResumeService *services.ResumeService
	resumeServiceOnce   sync.Once
	resumeServiceErr    error
)

// GetEntityRegistry returns the global EntityRegistry, initializing it if needed.
// Uses sync.Once for thread-safe lazy initialization.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func GetEntityRegistry() *services.EntityRegistry {
	registryOnce.Do(func() {
		db, err := GetDB(context.Background())
		if err != nil {
			panic(fmt.Sprintf("failed to get database for EntityRegistry: %v", err))
		}

		globalRegistry = services.NewEntityRegistry()
		globalRegistry.Register(models.EntityTypeEpic,
			services.NewEpicRepositoryAdapter(repository.NewEpicRepository(db)))
		globalRegistry.Register(models.EntityTypeFeature,
			services.NewFeatureRepositoryAdapter(repository.NewFeatureRepository(db)))
		globalRegistry.Register(models.EntityTypeTask,
			services.NewTaskRepositoryAdapter(repository.NewTaskRepository(db)))
		globalRegistry.Register(models.EntityTypeBug,
			services.NewBugRepositoryAdapter(repository.NewBugRepository(db)))
		globalRegistry.Register(models.EntityTypeChange,
			services.NewChangeCardRepositoryAdapter(repository.NewChangeCardRepository(db)))
	})
	return globalRegistry
}

// GetEntityService returns the global EntityService, initializing it if needed.
// Uses sync.Once for thread-safe lazy initialization.
// Wires the optional RejectionNoteCreator for rejection notes during transitions.
func GetEntityService() *services.EntityService {
	entityServiceOnce.Do(func() {
		workflowSvc := GetWorkflowService()
		globalEntityService = services.NewEntityService(workflowSvc)
		// Wire optional note repo for rejection notes during transitions
		db, err := GetDB(context.Background())
		if err == nil {
			globalEntityService.SetNoteRepo(repository.NewEntityNoteRepository(db))
			globalEntityService.SetHistoryRepo(repository.NewEntityHistoryRepository(db))
		}
	})
	return globalEntityService
}

// GetActionService returns the global ActionService, initializing it if needed.
// Uses sync.Once for thread-safe lazy initialization.
// Returns (config.ActionService, error) because config.NewActionService can fail.
func GetActionService(ctx context.Context) (config.ActionService, error) {
	actionServiceOnce.Do(func() {
		projectRoot, err := FindProjectRoot()
		if err != nil || projectRoot == "" {
			projectRoot = "."
		}
		configPath := filepath.Join(projectRoot, ".sharkconfig.json")
		svc, err := config.NewActionService(configPath)
		if err != nil {
			actionServiceErr = fmt.Errorf("failed to create ActionService: %w", err)
			return
		}
		globalActionService = svc
	})

	if actionServiceErr != nil {
		return nil, actionServiceErr
	}
	return globalActionService, nil
}

// GetNoteService returns the global NoteService, initializing it if needed.
func GetNoteService(ctx context.Context) (*services.NoteService, error) {
	noteServiceOnce.Do(func() {
		db, err := GetDB(ctx)
		if err != nil {
			noteServiceErr = fmt.Errorf("failed to get database for NoteService: %w", err)
			return
		}

		noteRepo := repository.NewEntityNoteRepository(db)
		globalNoteService = services.NewNoteService(noteRepo, GetEntityRegistry())
	})

	if noteServiceErr != nil {
		return nil, noteServiceErr
	}
	return globalNoteService, nil
}

// GetContextService returns the global ContextService, initializing it if needed.
// Unlike GetNoteService/GetResumeService, this never fails because its only
// dependency (GetEntityRegistry) panics on failure rather than returning an error.
func GetContextService() *services.ContextService {
	contextServiceOnce.Do(func() {
		globalContextService = services.NewContextService(GetEntityRegistry())
	})
	return globalContextService
}

// GetResumeService returns the global ResumeService, initializing it if needed.
func GetResumeService(ctx context.Context) (*services.ResumeService, error) {
	resumeServiceOnce.Do(func() {
		db, err := GetDB(ctx)
		if err != nil {
			resumeServiceErr = fmt.Errorf("failed to get database for ResumeService: %w", err)
			return
		}

		epicRepo := repository.NewEpicRepository(db)
		featureRepo := repository.NewFeatureRepository(db)
		taskRepo := repository.NewTaskRepository(db)
		noteRepo := repository.NewEntityNoteRepository(db)
		sessionRepo := &resumeSessionAdapter{repo: repository.NewWorkSessionRepository(db)}

		svc := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo, GetEntityRegistry())
		svc.SetSessionRepo(sessionRepo)
		globalResumeService = svc
	})

	if resumeServiceErr != nil {
		return nil, resumeServiceErr
	}
	return globalResumeService, nil
}

// taskServiceDeps holds the shared dependencies for constructing a TaskService.
// Extracted to eliminate duplication across GetTaskService, GetTaskServiceWithHistory,
// and GetTaskServiceWithDocs which all wire the same base dependencies.
type taskServiceDeps struct {
	db          *repository.DB
	taskRepo    *repository.TaskRepository
	featureRepo *repository.FeatureRepository
	entitySvc   *services.EntityService
	creatorSvc  *taskcreation.Creator
	historyRepo *repository.TaskHistoryRepository //nolint:staticcheck // Deprecated: will migrate to EntityHistoryRepository
}

// buildTaskServiceDeps constructs the shared dependencies for TaskService variants.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func buildTaskServiceDeps() taskServiceDeps {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	workflowSvc := GetWorkflowService()
	taskRepo := repository.NewTaskRepositoryWithWorkflow(db, workflowSvc.GetWorkflow())

	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}
	historyRepo := repository.NewTaskHistoryRepository(db) //nolint:staticcheck // Deprecated: will migrate to EntityHistoryRepository
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	renderer := templates.NewRenderer(templates.NewLoader(""))
	creatorSvc := taskcreation.NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, projectRoot, workflowSvc)

	return taskServiceDeps{
		db:          db,
		taskRepo:    taskRepo,
		featureRepo: featureRepo,
		entitySvc:   GetEntityService(),
		creatorSvc:  creatorSvc,
		historyRepo: historyRepo,
	}
}

// GetTaskService returns a TaskService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetTaskService()
//	task, err := svc.AdvanceTaskStatus(ctx, "E07-F01-001")
func GetTaskService() *services.TaskService {
	d := buildTaskServiceDeps()
	svc := services.NewTaskService(d.taskRepo, d.entitySvc, d.creatorSvc)
	svc.SetTracer(GetTracer("shark/services/task"))
	svc.SetFeatureRepo(d.featureRepo)
	svc.SetFeatureService(GetFeatureService())

	// Wire entity history recording for polymorphic entity_history table.
	entityHistoryRepo := repository.NewEntityHistoryRepository(d.db)
	svc.SetEntityHistoryRepo(entityHistoryRepo)

	// Wire sub-services for query delegation.
	querySvc := services.NewTaskQueryService(d.taskRepo)
	svc.SetQueryService(querySvc)

	return svc
}

// GetTaskServiceWithHistory returns a TaskService with the history repository wired.
// Used by commands that need to query task history (e.g., the history command).
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetTaskServiceWithHistory()
//	histories, err := svc.GetTaskHistory(ctx, "E07-F01-001")
func GetTaskServiceWithHistory() *services.TaskService {
	d := buildTaskServiceDeps()
	svc := services.NewTaskService(d.taskRepo, d.entitySvc, d.creatorSvc)
	svc.SetTracer(GetTracer("shark/services/task"))
	svc.SetFeatureRepo(d.featureRepo)
	svc.SetHistoryRepo(&taskHistoryAdapter{repo: d.historyRepo})
	svc.SetFeatureService(GetFeatureService())

	// Wire sub-services for query and history delegation.
	querySvc := services.NewTaskQueryService(d.taskRepo)
	svc.SetQueryService(querySvc)

	sessionRepo := &workSessionAdapter{repo: repository.NewWorkSessionRepository(d.db)}
	epicRepo := repository.NewEpicRepository(d.db)
	historySvc := services.NewTaskHistoryService(&taskHistoryAdapter{repo: d.historyRepo})
	historySvc.SetSessionRepo(sessionRepo)
	historySvc.SetFeatureRepo(d.featureRepo)
	historySvc.SetEpicRepo(epicRepo)
	svc.SetHistoryService(historySvc)

	return svc
}

// GetTaskServiceWithDocs returns a TaskService with document, session, and history repositories wired.
// Used by commands that need document operations, work sessions, or analytics.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetTaskServiceWithDocs()
//	doc, err := svc.LinkDocument(ctx, taskKey, title, path)
func GetTaskServiceWithDocs() *services.TaskService {
	d := buildTaskServiceDeps()
	docRepo := repository.NewDocumentRepository(d.db)
	entityDocRepo := repository.NewEntityDocumentRepository(d.db)
	docAdapter := repository.NewPolymorphicDocRepoAdapter(entityDocRepo)
	sessionRepo := &workSessionAdapter{repo: repository.NewWorkSessionRepository(d.db)}
	enrichRepo := repository.NewTemplateEnrichmentRepository(d.db)

	svc := services.NewTaskService(d.taskRepo, d.entitySvc, d.creatorSvc)
	svc.SetTracer(GetTracer("shark/services/task"))
	svc.SetDocRepo(docAdapter)
	svc.SetRelRepo(repository.NewEntityRelTaskKeyAdapter(d.db))
	svc.SetSessionRepo(sessionRepo)
	svc.SetFeatureRepo(d.featureRepo)
	svc.SetFeatureService(GetFeatureService())
	svc.SetWritableDocRepo(docRepo, entityDocRepo)
	svc.SetEnrichRepo(enrichRepo)

	// Wire entity history recording for polymorphic entity_history table.
	entityHistoryRepo := repository.NewEntityHistoryRepository(d.db)
	svc.SetEntityHistoryRepo(entityHistoryRepo)

	// Wire sub-services for query delegation.
	querySvc := services.NewTaskQueryService(d.taskRepo)
	svc.SetQueryService(querySvc)

	// Wire history sub-service for sessions/analytics.
	epicRepo := repository.NewEpicRepository(d.db)
	historySvc := services.NewTaskHistoryService(&taskHistoryAdapter{repo: d.historyRepo})
	historySvc.SetSessionRepo(sessionRepo)
	historySvc.SetFeatureRepo(d.featureRepo)
	historySvc.SetEpicRepo(epicRepo)
	svc.SetHistoryService(historySvc)
	svc.SetHistoryRepo(&taskHistoryAdapter{repo: d.historyRepo})

	return svc
}

// GetViewService returns a view.Service instance for viewing entity file paths.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetViewService()
//	filePath, err := svc.GetFilePath(ctx, parsedScope)
func GetViewService() *view.Service {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	svc := view.NewService(epicRepo, featureRepo, taskRepo)
	changeCardRepo := repository.NewChangeCardRepository(db)
	svc.SetChangeCardRepo(changeCardRepo)
	bugRepo := repository.NewBugRepository(db)
	svc.SetBugRepo(bugRepo)
	return svc
}

// GetValidationRunner returns a validation.Validator configured with all repositories.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	v := cli.GetValidationRunner()
//	result, err := v.Validate(ctx)
func GetValidationRunner() *validation.Validator {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	repoAdapter := validation.NewRepositoryAdapter(epicRepo, featureRepo, taskRepo)
	return validation.NewValidator(repoAdapter)
}

// GetChangeCardService returns a ChangeCardService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetChangeCardService()
//	card, err := svc.CreateChangeCard(ctx, input)
func GetChangeCardService() *services.ChangeCardService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	changeCardRepo := repository.NewChangeCardRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)

	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}

	entitySvc := GetEntityService()
	entityRepo := GetEntityRegistry().MustGetRepository(models.EntityTypeChange)

	svc := services.NewChangeCardService(changeCardRepo, entitySvc, entityRepo, epicRepo, featureRepo, projectRoot)
	docRepo := repository.NewDocumentRepository(db)
	entityDocRepo := repository.NewEntityDocumentRepository(db)
	svc.SetWritableDocRepo(docRepo, entityDocRepo)
	// No longer need: svc.SetEntityHistoryRepo(...) -- EntityService handles history
	return svc
}

// GetBugService returns a BugService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetBugService()
//	bug, err := svc.CreateBug(ctx, input)
func GetBugService() *services.BugService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	bugRepo := repository.NewBugRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}

	entitySvc := GetEntityService()
	entityRepo := GetEntityRegistry().MustGetRepository(models.EntityTypeBug)

	svc := services.NewBugService(bugRepo, entitySvc, entityRepo, epicRepo, featureRepo, taskRepo, projectRoot)
	docRepo := repository.NewDocumentRepository(db)
	entityDocRepo := repository.NewEntityDocumentRepository(db)
	svc.SetWritableDocRepo(docRepo, entityDocRepo)
	// No longer need: svc.SetEntityHistoryRepo(...) -- EntityService handles history
	return svc
}

// GetDashboardAnalyticsService returns a DashboardAnalyticsService instance.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetDashboardAnalyticsService()
//	result, err := svc.GetBugAnalytics(ctx)
func GetDashboardAnalyticsService() *services.DashboardAnalyticsService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	bugRepo := repository.NewBugRepository(db)
	ccRepo := repository.NewChangeCardRepository(db)
	return services.NewDashboardAnalyticsService(bugRepo, ccRepo)
}

// GetEntityHistoryService returns an EntityHistoryService instance.
// Creates a new instance each call (lightweight, no shared state).
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func GetEntityHistoryService() *services.EntityHistoryService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for EntityHistoryService: %v", err))
	}
	historyRepo := repository.NewEntityHistoryRepository(db)
	return services.NewEntityHistoryService(historyRepo, GetEntityRegistry())
}

// GetEntityRelationshipService returns an EntityRelationshipService instance.
// Creates a new instance each call (lightweight, no shared state).
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func GetEntityRelationshipService() *services.EntityRelationshipService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for EntityRelationshipService: %v", err))
	}
	repo := repository.NewEntityRelationshipRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	return services.NewEntityRelationshipService(repo, taskRepo)
}

// ResetServices clears global service state. For testing only.
func ResetServices() {
	globalActionService = nil
	actionServiceErr = nil
	actionServiceOnce = sync.Once{}

	globalNoteService = nil
	noteServiceErr = nil
	noteServiceOnce = sync.Once{}

	globalContextService = nil
	contextServiceOnce = sync.Once{}

	globalResumeService = nil
	resumeServiceErr = nil
	resumeServiceOnce = sync.Once{}

	// Reset registry
	globalRegistry = nil
	registryOnce = sync.Once{}

	// Reset entity service
	globalEntityService = nil
	entityServiceOnce = sync.Once{}

	// Reset observability state for test isolation
	ResetObservability()
}
