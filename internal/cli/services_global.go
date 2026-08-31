package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entityrel"
	planhierarchyrepo "github.com/jwwelbor/shark-task-manager/internal/repository/planhierarchy"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
	sprintrepo "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/repository/worksession"
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
		// B030: register sprint and idea adapters so polymorphic services
		// (NoteService, ContextService, EntityHistoryService, etc.) can
		// resolve S### and I-YYYY-MM-DD-## keys to their backing entities.
		c.registry.Register(models.EntityTypeSprint,
			services.NewSprintRepositoryAdapter(repository.NewSprintRepository(db)))
		c.registry.Register(models.EntityTypeIdea,
			services.NewIdeaRepositoryAdapter(repository.NewIdeaRepository(db)))
		c.registry.Register(models.EntityTypeQuestion,
			services.NewQuestionRepositoryAdapter(repository.NewQuestionRepository(db)))
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
			if cfg, cfgErr := GetConfig(); cfgErr == nil && cfg != nil {
				c.entityService.SetAdvanceGuard(cfg.GetAdvanceGuard(), repository.NewAdvanceGuardRepository(db))
			}
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
		svc.SetSearchIndexer(repository.NewSearchRepository(db))
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
	svc.SetAggregateMutationCoordinator(services.NewAggregateMutationCoordinator(repository.NewProgressMutationRepository(), GetWorkflowService()))

	// E28-F04 T-006: wire the shared *TagService so TaskService can enforce
	// `tag_required_for` on create and honour --tag on create/update.
	svc.SetTagService(GetTagService())
	svc.SetSearchIndexer(repository.NewSearchRepository(d.db))

	// Wire size enforcement (mirrors tag enforcement). When `size_required_for`
	// in .sharkconfig.json contains "task", CreateTask rejects calls without --size.
	svc.SetSizeEnforcement(getSizeEnforcement())

	// Wire entity history recording for polymorphic entity_history table.
	entityHistoryRepo := repository.NewEntityHistoryRepository(d.db)
	svc.SetEntityHistoryRepo(entityHistoryRepo)

	// Wire cascade reopen dependencies so that a task regression automatically
	// reopens terminal feature and epic ancestors (AC-T1 / REQ-F-001).
	epicRepo := repository.NewEpicRepository(d.db)
	svc.SetCascadeDeps(d.db, d.featureRepo, epicRepo, entityHistoryRepo, entityHistoryRepo)

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
	svc.SetAggregateMutationCoordinator(services.NewAggregateMutationCoordinator(repository.NewProgressMutationRepository(), GetWorkflowService()))
	svc.SetSearchIndexer(repository.NewSearchRepository(d.db))

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

	// Wire cascade reopen dependencies so that a task regression automatically
	// reopens terminal feature and epic ancestors (AC-T1 / REQ-F-001).
	entityHistoryRepo := repository.NewEntityHistoryRepository(d.db)
	svc.SetCascadeDeps(d.db, d.featureRepo, epicRepo, entityHistoryRepo, entityHistoryRepo)

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
	svc.SetAggregateMutationCoordinator(services.NewAggregateMutationCoordinator(repository.NewProgressMutationRepository(), GetWorkflowService()))
	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}
	svc.SetWritableDocRepo(docRepo, entityDocRepo, projectRoot)
	svc.SetEnrichRepo(enrichRepo)

	// E28-F05 T-010: wire TagService so GetTaskWithTags renders the Tags row
	// in rich-display (REQ-F-014 / AC-28c). Must match the wiring in GetTaskService().
	svc.SetTagService(GetTagService())
	svc.SetSizeEnforcement(getSizeEnforcement())
	svc.SetSearchIndexer(repository.NewSearchRepository(d.db))

	// Wire entity history recording for polymorphic entity_history table.
	entityHistoryRepo := repository.NewEntityHistoryRepository(d.db)
	svc.SetEntityHistoryRepo(entityHistoryRepo)

	// Wire cascade reopen dependencies so that a task regression automatically
	// reopens terminal feature and epic ancestors (AC-T1 / REQ-F-001).
	epicRepo := repository.NewEpicRepository(d.db)
	svc.SetCascadeDeps(d.db, d.featureRepo, epicRepo, entityHistoryRepo, entityHistoryRepo)

	// Wire sub-services for query delegation.
	querySvc := services.NewTaskQueryService(d.taskRepo)
	svc.SetQueryService(querySvc)

	// Wire history sub-service for sessions/analytics.
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
	svc.SetWritableDocRepo(docRepo, entityDocRepo, projectRoot)

	// E28-F04 T-009: wire the shared *TagService so ChangeCardService can
	// enforce `tag_required_for` on create and honour --tag on create/update.
	svc.SetTagService(GetTagService())
	svc.SetSizeEnforcement(getSizeEnforcement())
	svc.SetSearchIndexer(repository.NewSearchRepository(db))

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

	// E28-F04 T-005: pass the shared *TagService so BugService can enforce
	// `tag_required_for` on create and honour --tag on create/update.
	tagSvc := GetTagService()
	svc := services.NewBugService(bugRepo, entitySvc, entityRepo, epicRepo, featureRepo, taskRepo, projectRoot, tagSvc)
	svc.SetSizeEnforcement(getSizeEnforcement())
	svc.SetSearchIndexer(repository.NewSearchRepository(db))
	docRepo := repository.NewDocumentRepository(db)
	entityDocRepo := repository.NewEntityDocumentRepository(db)
	svc.SetWritableDocRepo(docRepo, entityDocRepo, projectRoot)
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

	// Pass the shared *TagService so TechDebtService can enforce
	// `tag_required_for` on create and honour --tag on create/update.
	tagSvc := GetTagService()
	svc := services.NewTechDebtService(tdRepo, entitySvc, entityRepo, projectRoot, tagSvc)
	svc.SetSizeEnforcement(getSizeEnforcement())
	svc.SetSearchIndexer(repository.NewSearchRepository(db))
	docRepo := repository.NewDocumentRepository(db)
	entityDocRepo := repository.NewEntityDocumentRepository(db)
	svc.SetWritableDocRepo(docRepo, entityDocRepo, projectRoot)
	return svc
}

// GetQuestionService returns the direct service for the bounded Question
// entity. It is intentionally constructed per call like the other simple
// entity accessors and contains no CLI concerns.
func GetQuestionService() *services.QuestionService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for QuestionService: %v", err))
	}
	svc, err := services.NewQuestionService(repository.NewQuestionRepository(db))
	if err != nil {
		panic(fmt.Sprintf("failed to create QuestionService: %v", err))
	}
	svc.SetHistoryRepo(repository.NewEntityHistoryRepository(db))
	svc.SetSearchIndexer(repository.NewSearchRepository(db))
	svc.SetClaimReader(GetClaimService())
	svc.SetFocusedReadDependencies(entityrel.NewEntityRelationshipRepository(db), GetEntityRegistry())
	svc.SetEntityTransitioner(GetEntityService(), GetEntityRegistry().MustGetRepository(models.EntityTypeQuestion))
	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}
	svc.SetProjectRoot(projectRoot)
	return svc
}

// GetQuestionBlocker returns the read-only direct Question gate used by
// dispatch and transition boundaries.
func GetQuestionBlocker() *services.QuestionBlocker {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for QuestionBlocker: %v", err))
	}
	blocker, err := services.NewQuestionBlocker(
		repository.NewEntityRelationshipRepository(db), GetEntityRegistry(), GetQuestionService(),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create QuestionBlocker: %v", err))
	}
	return blocker
}

// GetQuestionDocumentService returns the generic EntityDocumentService wired
// for Questions, used by `shark related-docs add/delete/list --question`.
// QuestionService itself does not expose LinkDocument/UnlinkDocument/
// ListDocuments wrapper methods the way BugService/ChangeCardService/
// TaskService do, so this accessor mirrors their construction (wiring
// through cli.GetDB/FindProjectRoot/GetEntityRegistry here rather than
// inline in the commands package) without changing the *ByKey call shape
// the Question document commands already use.
func GetQuestionDocumentService(ctx context.Context) *services.EntityDocumentService {
	db, err := GetDB(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to get database for question documents: %v", err))
	}
	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}
	entityRepo := GetEntityRegistry().MustGetRepository(models.EntityTypeQuestion)
	return services.NewEntityDocumentService(
		repository.NewDocumentRepository(db),
		repository.NewEntityDocumentRepository(db),
		services.EntityLookupFnFromRepo(entityRepo),
		projectRoot,
	)
}

// GetCascadeService returns a CascadeService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Used by the in-process `shark run` controller. Keyed `shark next` uses
// GetPlanHierarchyService so sequential and fork-emitting modes share one
// claim/dependency/order snapshot.
//
// Usage:
//
//	svc := cli.GetCascadeService()
//	state, err := svc.DescribeDispatchableChildren(ctx, "feature", "E07-F01")
func GetCascadeService() *services.CascadeService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	taskRepo := repository.NewTaskRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	workflowSvc := GetWorkflowService()
	return services.NewCascadeService(taskRepo, epicRepo, featureRepo, workflowSvc)
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

// GetSprintService returns a SprintService instance.
// Creates a new instance each call with the global DB and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// The single *sprint.SprintRepository satisfies SprintRepository,
// SprintAssignmentQueryRepository, and SprintCapacityRepository — no
// additional repository type is needed.
//
// Usage:
//
//	svc := cli.GetSprintService()
//	sprint, err := svc.CreateSprint(ctx, input)
func GetSprintService() *services.SprintService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	cfg, _ := GetConfig() // nil-safe; sprint_defaults read from .sharkconfig.json
	sprintRepo := repository.NewSprintRepository(db)
	workflowSvc := GetWorkflowService()
	// Pass db for CloseSprintWithCarryover transaction support (T-E19-F03-007)
	svc := services.NewSprintService(sprintRepo, workflowSvc, sprintRepo, sprintRepo, cfg, db)
	// B059: wire generic status-transition dispatch so sprints support
	// `shark next`/`shark status set|advance` like other entity types.
	svc.EnableWorkflowDispatch(GetEntityService(), services.NewSprintRepositoryAdapter(sprintRepo))
	// B044: wire claim-awareness so GetNextTask never hands out an
	// actively-claimed backlog item.
	svc.SetClaimReader(GetClaimService())
	// E19-F09: sprint selection shares the same read-only Question gate as
	// keyed dispatch so open Questions cannot be selected for a sprint wave.
	svc.SetQuestionBlocker(GetQuestionBlocker())
	portfolioSnapshot := portfoliorepo.NewRepository(db)
	svc.SetAdmissionService(services.NewSprintAdmissionService(
		services.NewPortfolioSprintAdmissionEvidenceReader(
			portfolioSnapshot,
			GetPortfolioAdviceService(),
			GetPortfolioPlanningService(),
			GetWorkflowService(),
		),
	))
	return svc
}

// GetSprintAnalyticsService returns a SprintAnalyticsService instance.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// SprintAnalyticsService is read-only and has no workflow validation dependencies.
// It is kept separate from SprintService to maintain single responsibility.
//
// Usage:
//
//	svc := cli.GetSprintAnalyticsService()
//	result, err := svc.GetVelocity(ctx, 5)
func GetSprintAnalyticsService() *services.SprintAnalyticsService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	analyticsRepo := &sprintAnalyticsAdapter{repo: sprintrepo.NewSprintAnalyticsRepository(db)}
	sprintRepo := repository.NewSprintRepository(db)
	svc := services.NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	svc.SetWorkflow(GetWorkflowService())
	return svc
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

// GetClaimService returns a ClaimService backed by the global DB connection.
// Creates a new instance per call (the underlying repo is stateless); the TTL
// is read from .sharkconfig.json when claim_ttl_seconds is set, otherwise from
// SHARK_CLAIM_TTL_SECONDS or services.DefaultClaimTTL.
// Panics on DB failure (fail-fast, matching the other CLI accessors).
func GetClaimService() *services.ClaimService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	var ttl *time.Duration
	if cfg, cfgErr := GetConfig(); cfgErr == nil && cfg != nil && cfg.ClaimTTLSeconds != nil {
		resolved := time.Duration(*cfg.ClaimTTLSeconds) * time.Second
		ttl = &resolved
	}
	svc := services.NewClaimService(claimrepo.NewRepository(db), ttl)
	svc.SetSessionLog(worksession.NewWorkSessionRepository(db))
	svc.SetTaskResolver(repository.NewTaskRepository(db))
	return svc
}

// GetPortfolioAdviceService returns a read-only portfolio advice service
// backed by the shared CLI database and configured workflows. This is the
// internal evidence input to bare `shark plan`'s epic selection.
func GetPortfolioAdviceService() *services.PortfolioAdviceService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	return services.NewPortfolioAdviceServiceFromSnapshot(
		portfoliorepo.NewRepository(db),
		GetClaimService(),
		GetWorkflowService(),
	)
}

// GetPlanHierarchyService returns the one-query direct-child reader used by
// one-level `shark plan <epic|feature>` selection and both keyed `shark next`
// cascade emission modes. GetCascadeService remains for `shark run`.
func GetPlanHierarchyService() *services.PlanHierarchyService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	return services.NewPlanHierarchyService(
		planhierarchyrepo.NewRepository(db),
		GetWorkflowService(),
		GetClaimService(),
		services.PlanHierarchyEdgeReaders{
			Relationships:    GetEntityRelationshipService(),
			Registry:         GetEntityRegistry(),
			TaskDependencies: repository.NewTaskRepository(db),
		},
	)
}

// GetPortfolioPlanningService returns the stateless selector used to turn
// portfolio evidence into a first executable epic-root layer for bare
// `shark plan`.
func GetPortfolioPlanningService() *services.PortfolioPlanningService {
	return services.NewPortfolioPlanningService()
}

// GetStandalonePlanningService returns the selector used by standalone
// collection roots such as `shark plan bugs`.
func GetStandalonePlanningService() *services.StandalonePlanningService {
	dependencies := services.NewStandaloneHardDependencyService(
		GetEntityRelationshipService(),
		GetEntityRegistry(),
		GetWorkflowService(),
	)
	return services.NewStandalonePlanningService(
		GetBugService(),
		GetChangeCardService(),
		GetTechDebtService(),
		GetClaimService(),
		dependencies,
	)
}
