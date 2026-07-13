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
	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	sprintrepo "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	teamrunrepo "github.com/jwwelbor/shark-task-manager/internal/repository/teamrun"
	"github.com/jwwelbor/shark-task-manager/internal/repository/worksession"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/team"
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

// GetCascadeService returns a CascadeService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Used by `shark next` cascade resolution to enumerate dispatchable children
// (B029): the CLI command must NOT construct repositories directly, so this
// accessor wires the underlying task/epic/feature repositories at the CLI
// boundary and returns a service the command can call.
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

// GetTeamPlanner returns the read-only team planner wired to the same child,
// dependency, workflow, and claim sources used by ordinary Shark services.
// It deliberately constructs no command, dispatcher, or mutation path.
func GetTeamPlanner() *team.TeamPlanner {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for TeamPlanner: %v", err))
	}
	actionSvc, err := GetActionService(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to create action service for TeamPlanner: %v", err))
	}

	taskRepo := repository.NewTaskRepository(db)
	dependencyAdapter := team.NewDependencyAdapter(
		&teamLegacyDependencySource{repo: taskRepo},
		&teamRelationshipDependencySource{
			repo:     repository.NewEntityRelationshipRepository(db),
			registry: GetEntityRegistry(),
		},
	)
	planner, err := team.NewTeamPlanner(team.PlannerDeps{
		Children:     &teamCascadeChildReader{cascade: GetCascadeService()},
		Dependencies: dependencyAdapter,
		Dispatch:     newTeamDispatchStepResolver(actionSvc),
		Claims:       &teamClaimDiagnosticReader{claims: GetClaimService()},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create TeamPlanner: %v", err))
	}
	return planner
}

// GetTeamLedger returns the durable team ledger backed by the normalized
// team-run repository. The service owns validation and idempotency; this
// accessor only supplies the repository dependency.
func GetTeamLedger() *team.LedgerService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database for TeamLedger: %v", err))
	}
	return team.NewLedgerService(teamrunrepo.NewTeamRunRepository(db))
}

type teamCascadeChildReader struct {
	cascade *services.CascadeService
}

func (r *teamCascadeChildReader) ListChildren(ctx context.Context, rootType models.EntityType, rootKey string) ([]team.ChildSnapshot, error) {
	children, err := r.cascade.ListChildren(ctx, string(rootType), rootKey)
	if err != nil {
		return nil, err
	}
	out := make([]team.ChildSnapshot, 0, len(children))
	for _, child := range children {
		item := team.ChildSnapshot{
			Key:                child.Key,
			EntityType:         child.EntityType,
			Status:             child.Status,
			LegacyDependencies: valueOrEmpty(child.DependsOn),
		}
		if child.ExecutionOrder != nil {
			item.ExecutionOrder = *child.ExecutionOrder
		}
		if child.Priority != nil {
			item.Priority = *child.Priority
		}
		out = append(out, item)
	}
	return out, nil
}

type teamLegacyDependencySource struct {
	repo interface {
		GetByKey(context.Context, string) (*models.Task, error)
	}
}

func (s *teamLegacyDependencySource) ListLegacyDependencies(ctx context.Context, child team.ChildIdentity) (string, error) {
	if child.EntityType != models.EntityTypeTask {
		return "", nil
	}
	task, err := s.repo.GetByKey(ctx, child.Key)
	if err != nil {
		return "", fmt.Errorf("get legacy dependencies for %s: %w", child.Key, err)
	}
	if task == nil || task.DependsOn == nil {
		return "", nil
	}
	return *task.DependsOn, nil
}

type teamRelationshipDependencySource struct {
	repo     *repository.EntityRelationshipRepository
	registry *services.EntityRegistry
}

func (s *teamRelationshipDependencySource) ListRelationshipDependencies(ctx context.Context, child team.ChildIdentity) ([]team.DependencyEdge, error) {
	entityRepo, err := s.registry.GetRepository(child.EntityType)
	if err != nil {
		return nil, fmt.Errorf("resolve %s repository: %w", child.Key, err)
	}
	entity, err := entityRepo.GetByKey(ctx, child.Key)
	if err != nil {
		return nil, fmt.Errorf("resolve %s for relationships: %w", child.Key, err)
	}
	if entity == nil {
		return nil, fmt.Errorf("resolve %s for relationships: empty entity", child.Key)
	}
	rels, err := s.repo.GetOutgoing(ctx, child.EntityType, entity.GetID(), []models.EntityRelationshipType{models.EntityRelDependsOn})
	if err != nil {
		return nil, fmt.Errorf("list relationships for %s: %w", child.Key, err)
	}
	edges := make([]team.DependencyEdge, 0, len(rels))
	for _, rel := range rels {
		if rel == nil {
			return nil, fmt.Errorf("list relationships for %s: nil relationship", child.Key)
		}
		targetRepo, err := s.registry.GetRepository(rel.ToEntityType)
		if err != nil {
			return nil, fmt.Errorf("resolve dependency repository for %s: %w", child.Key, err)
		}
		target, err := targetRepo.GetByID(ctx, rel.ToEntityID)
		if err != nil {
			return nil, fmt.Errorf("resolve dependency for %s: %w", child.Key, err)
		}
		if target == nil {
			return nil, fmt.Errorf("resolve dependency for %s: empty entity", child.Key)
		}
		edges = append(edges, team.DependencyEdge{
			ChildKey:         child.Key,
			ChildType:        child.EntityType,
			DependencyKey:    target.GetKey(),
			DependencyType:   rel.ToEntityType,
			DependencyStatus: target.GetStatus(),
			Satisfied:        false,
			Resolved:         true,
			Source:           "relationship",
		})
	}
	return edges, nil
}

type teamClaimDiagnosticReader struct {
	claims *services.ClaimService
}

func (r *teamClaimDiagnosticReader) Diagnose(ctx context.Context, child team.ChildIdentity) (team.ClaimDiagnostic, error) {
	claim, err := r.claims.Get(ctx, string(child.EntityType), child.Key)
	if err != nil {
		return team.ClaimDiagnostic{}, fmt.Errorf("read claim for %s: %w", child.Key, err)
	}
	if claim == nil {
		return team.ClaimDiagnostic{}, nil
	}
	return team.ClaimDiagnostic{Claimed: true, ClaimSessionID: claim.SessionID, Reason: fmt.Sprintf("claimed by %s", claim.ClaimedBy)}, nil
}

type teamDispatchStepResolver struct {
	actionService action.ActionService
	transitioners map[models.EntityType]dispatch.EntityTransitioner
	placeholders  map[models.EntityType]dispatch.PlaceholderGenerator
}

func newTeamDispatchStepResolver(actionService action.ActionService) dispatch.DispatchStepResolver {
	return &teamDispatchStepResolver{
		actionService: actionService,
		transitioners: map[models.EntityType]dispatch.EntityTransitioner{
			models.EntityTypeTask:     GetTaskService(),
			models.EntityTypeFeature:  GetFeatureService(),
			models.EntityTypeEpic:     GetEpicService(),
			models.EntityTypeBug:      GetBugService(),
			models.EntityTypeChange:   GetChangeCardService(),
			models.EntityTypeTechDebt: GetTechDebtService(),
		},
		placeholders: map[models.EntityType]dispatch.PlaceholderGenerator{
			models.EntityTypeTask: teamPlaceholderGeneratorFunc(func(ctx context.Context, key string) (map[string]string, error) {
				entity, err := GetTaskService().GetTask(ctx, key)
				if err != nil {
					return nil, err
				}
				return config.TaskPlaceholders(entity), nil
			}),
			models.EntityTypeFeature: teamPlaceholderGeneratorFunc(func(ctx context.Context, key string) (map[string]string, error) {
				entity, err := GetFeatureService().GetFeature(ctx, key)
				if err != nil {
					return nil, err
				}
				return config.FeaturePlaceholders(entity), nil
			}),
			models.EntityTypeEpic: teamPlaceholderGeneratorFunc(func(ctx context.Context, key string) (map[string]string, error) {
				entity, err := GetEpicService().GetEpic(ctx, key)
				if err != nil {
					return nil, err
				}
				return config.EpicPlaceholders(entity), nil
			}),
			models.EntityTypeBug: teamPlaceholderGeneratorFunc(func(ctx context.Context, key string) (map[string]string, error) {
				entity, err := GetBugService().GetBug(ctx, key)
				if err != nil {
					return nil, err
				}
				return config.BugPlaceholders(entity), nil
			}),
			models.EntityTypeChange: teamPlaceholderGeneratorFunc(func(ctx context.Context, key string) (map[string]string, error) {
				entity, err := GetChangeCardService().GetChangeCard(ctx, key)
				if err != nil {
					return nil, err
				}
				return config.ChangeCardPlaceholders(entity), nil
			}),
			models.EntityTypeTechDebt: teamPlaceholderGeneratorFunc(func(ctx context.Context, key string) (map[string]string, error) {
				entity, err := GetTechDebtService().GetTechDebt(ctx, key)
				if err != nil {
					return nil, err
				}
				return config.TechDebtPlaceholders(entity), nil
			}),
		},
	}
}

type teamPlaceholderGeneratorFunc func(context.Context, string) (map[string]string, error)

func (f teamPlaceholderGeneratorFunc) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	return f(ctx, key)
}

func (r *teamDispatchStepResolver) Resolve(ctx context.Context, entityType models.EntityType, key string) (dispatch.DispatchStep, error) {
	transitioner, ok := r.transitioners[entityType]
	if !ok {
		return dispatch.DispatchStep{}, fmt.Errorf("resolve team dispatch step: unsupported entity type %q", entityType)
	}
	resolver, err := dispatch.NewDispatchStepResolver(dispatch.StepResolverDeps{
		Transitioner:     transitioner,
		Placeholders:     r.placeholders[entityType],
		ActionService:    r.actionService.ForEntity(action.NormalizeEntityType(string(entityType))),
		IsArchivedStatus: func(models.EntityType, string) bool { return false },
	})
	if err != nil {
		return dispatch.DispatchStep{}, fmt.Errorf("create team dispatch-step resolver: %w", err)
	}
	return resolver.Resolve(ctx, entityType, key)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
	return services.NewSprintService(sprintRepo, workflowSvc, sprintRepo, sprintRepo, cfg, db)
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
	return services.NewSprintAnalyticsService(analyticsRepo, sprintRepo)
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
