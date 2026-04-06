package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/validation"
	"github.com/jwwelbor/shark-task-manager/internal/view"
)

// serviceContainer holds all lazily-initialized singleton services.
// Using a single struct makes ResetServices() safe: we swap the entire
// container atomically instead of reassigning individual sync.Once values
// (which would be a data race if any goroutine is mid-initialization).
type serviceContainer struct {
	registryOnce sync.Once
	registry     *services.EntityRegistry

	entityServiceOnce sync.Once
	entityService     *services.EntityService

	actionServiceOnce sync.Once
	actionService     config.ActionService
	actionServiceErr  error

	noteServiceOnce sync.Once
	noteService     *services.NoteService
	noteServiceErr  error

	contextServiceOnce sync.Once
	contextService     *services.ContextService

	resumeServiceOnce sync.Once
	resumeService     *services.ResumeService
	resumeServiceErr  error
}

// globalContainer is accessed only through loadContainer / storeContainer.
// Using atomic pointer operations ensures that a call to ResetServices()
// is immediately visible to any goroutine that subsequently calls a
// Get* function, without requiring a separate mutex.
//
//nolint:gochecknoglobals // Intentional package-level singleton for CLI entry points.
var globalContainer unsafe.Pointer // *serviceContainer

func init() {
	storeContainer(new(serviceContainer))
}

func loadContainer() *serviceContainer {
	return (*serviceContainer)(atomic.LoadPointer(&globalContainer))
}

func storeContainer(c *serviceContainer) {
	atomic.StorePointer(&globalContainer, unsafe.Pointer(c))
}

// GetEntityRegistry returns the global EntityRegistry, initializing it if needed.
// Uses sync.Once for thread-safe lazy initialization.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func GetEntityRegistry() *services.EntityRegistry {
	c := loadContainer()
	c.registryOnce.Do(func() {
		db, err := GetDB(context.Background())
		if err != nil {
			panic(fmt.Sprintf("failed to get database for EntityRegistry: %v", err))
		}

		c.registry = services.NewEntityRegistry()
		c.registry.Register(models.EntityTypeEpic,
			services.NewEpicRepositoryAdapter(repository.NewEpicRepository(db)))
		c.registry.Register(models.EntityTypeFeature,
			services.NewFeatureRepositoryAdapter(repository.NewFeatureRepository(db)))
		c.registry.Register(models.EntityTypeTask,
			services.NewTaskRepositoryAdapter(repository.NewTaskRepository(db)))
		c.registry.Register(models.EntityTypeBug,
			services.NewBugRepositoryAdapter(repository.NewBugRepository(db)))
		c.registry.Register(models.EntityTypeChange,
			services.NewChangeCardRepositoryAdapter(repository.NewChangeCardRepository(db)))
		c.registry.Register(models.EntityTypeTechDebt,
			services.NewTechDebtRepositoryAdapter(repository.NewTechDebtRepository(db)))
	})
	return c.registry
}

// GetEntityService returns the global EntityService, initializing it if needed.
// Uses sync.Once for thread-safe lazy initialization.
// Wires the optional RejectionNoteCreator for rejection notes during transitions.
func GetEntityService() *services.EntityService {
	c := loadContainer()
	c.entityServiceOnce.Do(func() {
		workflowSvc := GetWorkflowService()
		c.entityService = services.NewEntityService(workflowSvc)
		// Wire optional note repo for rejection notes during transitions
		db, err := GetDB(context.Background())
		if err == nil {
			c.entityService.SetNoteRepo(repository.NewEntityNoteRepository(db))
			c.entityService.SetHistoryRepo(repository.NewEntityHistoryRepository(db))
		}
	})
	return c.entityService
}

// GetActionService returns the global ActionService, initializing it if needed.
// Uses sync.Once for thread-safe lazy initialization.
// Returns (config.ActionService, error) because config.NewActionService can fail.
func GetActionService(ctx context.Context) (config.ActionService, error) {
	c := loadContainer()
	c.actionServiceOnce.Do(func() {
		projectRoot, err := FindProjectRoot()
		if err != nil || projectRoot == "" {
			projectRoot = "."
		}
		configPath := filepath.Join(projectRoot, ".sharkconfig.json")
		svc, err := config.NewActionService(configPath)
		if err != nil {
			c.actionServiceErr = fmt.Errorf("failed to create ActionService: %w", err)
			return
		}
		c.actionService = svc
	})

	if c.actionServiceErr != nil {
		return nil, c.actionServiceErr
	}
	return c.actionService, nil
}

// GetNoteService returns the global NoteService, initializing it if needed.
func GetNoteService(ctx context.Context) (*services.NoteService, error) {
	c := loadContainer()
	c.noteServiceOnce.Do(func() {
		db, err := GetDB(ctx)
		if err != nil {
			c.noteServiceErr = fmt.Errorf("failed to get database for NoteService: %w", err)
			return
		}

		noteRepo := repository.NewEntityNoteRepository(db)
		svc, svcErr := services.NewNoteService(noteRepo, GetEntityRegistry())
		if svcErr != nil {
			c.noteServiceErr = fmt.Errorf("failed to create NoteService: %w", svcErr)
			return
		}
		c.noteService = svc
	})

	if c.noteServiceErr != nil {
		return nil, c.noteServiceErr
	}
	return c.noteService, nil
}

// GetContextService returns the global ContextService, initializing it if needed.
// Unlike GetNoteService/GetResumeService, this never fails because its only
// dependency (GetEntityRegistry) panics on failure rather than returning an error.
func GetContextService() *services.ContextService {
	c := loadContainer()
	c.contextServiceOnce.Do(func() {
		svc, err := services.NewContextService(GetEntityRegistry())
		if err != nil {
			panic(fmt.Sprintf("failed to create ContextService: %v", err))
		}
		c.contextService = svc
	})
	return c.contextService
}

// GetResumeService returns the global ResumeService, initializing it if needed.
func GetResumeService(ctx context.Context) (*services.ResumeService, error) {
	c := loadContainer()
	c.resumeServiceOnce.Do(func() {
		db, err := GetDB(ctx)
		if err != nil {
			c.resumeServiceErr = fmt.Errorf("failed to get database for ResumeService: %w", err)
			return
		}

		epicRepo := repository.NewEpicRepository(db)
		featureRepo := repository.NewFeatureRepository(db)
		taskRepo := repository.NewTaskRepository(db)
		noteRepo := repository.NewEntityNoteRepository(db)
		sessionRepo := &resumeSessionAdapter{repo: repository.NewWorkSessionRepository(db)}

		svc, svcErr := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo, GetEntityRegistry())
		if svcErr != nil {
			c.resumeServiceErr = fmt.Errorf("failed to create ResumeService: %w", svcErr)
			return
		}
		svc.SetSessionRepo(sessionRepo)
		c.resumeService = svc
	})

	if c.resumeServiceErr != nil {
		return nil, c.resumeServiceErr
	}
	return c.resumeService, nil
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

// GetTechDebtService returns a TechDebtService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetTechDebtService()
//	td, err := svc.CreateTechDebt(ctx, input)
func GetTechDebtService() *services.TechDebtService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	tdRepo := repository.NewTechDebtRepository(db)

	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}

	entitySvc := GetEntityService()
	entityRepo := GetEntityRegistry().MustGetRepository(models.EntityTypeTechDebt)

	svc := services.NewTechDebtService(tdRepo, entitySvc, entityRepo, projectRoot)
	docRepo := repository.NewDocumentRepository(db)
	entityDocRepo := repository.NewEntityDocumentRepository(db)
	svc.SetWritableDocRepo(docRepo, entityDocRepo)
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
	tdRepo := repository.NewTechDebtRepository(db)
	return services.NewDashboardAnalyticsService(bugRepo, ccRepo, tdRepo)
}

// GetConfigService returns a ConfigService instance.
// ConfigService is stateless and cheap to create, so no singleton is needed.
//
// Usage:
//
//	svc := cli.GetConfigService()
//	report, err := svc.ValidateAllPatterns(cfg)
func GetConfigService() *services.ConfigService {
	return services.NewConfigService()
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

// resetEntityService resets only the entity service singleton within the current container.
// Called by ResetWorkflowService to invalidate the EntityService when the workflow changes.
// This targets only the entityService fields without replacing the entire container,
// preserving other initialized services (e.g. actionService).
//
// Note: This resets state within the current container by swapping to a fresh container
// that preserves non-entity-service singletons. For test simplicity, we reset the full
// container here, since ResetWorkflowService is only called in tests.
func resetEntityService() {
	storeContainer(new(serviceContainer))
}

// ResetServices replaces the global service container with a fresh one.
// All lazily-initialized singletons (EntityRegistry, EntityService, ActionService,
// NoteService, ContextService, ResumeService) are discarded and will be
// re-initialized on the next call.
//
// The swap is performed with a single atomic pointer store, which is safe to
// call concurrently: any goroutine that has already loaded the old container
// will continue using it until its current operation completes; subsequent
// callers will see the new (empty) container.
//
// For testing only. Do not call from production code.
func ResetServices() {
	storeContainer(new(serviceContainer))

	// Reset observability state for test isolation
	ResetObservability()
}
