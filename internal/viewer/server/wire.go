package server

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	viewerapi "github.com/jwwelbor/shark-task-manager/internal/api/viewer"
	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/bug"
	"github.com/jwwelbor/shark-task-manager/internal/repository/changecard"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entitydoc"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entityrel"
	"github.com/jwwelbor/shark-task-manager/internal/repository/idea"
	repnote "github.com/jwwelbor/shark-task-manager/internal/repository/note"
	sprintrepo "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	tagrepo "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
	techdebtrepo "github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// loadTagEnforcementConfig reads .sharkconfig.json from projectRoot and
// returns a services.TagEnforcementConfig. If the file is missing or fails
// to parse, an empty fallback is returned so wiring never fails at startup.
func loadTagEnforcementConfig(projectRoot string) services.TagEnforcementConfig {
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	mgr := config.NewManager(configPath)
	cfg, err := mgr.Load()
	if err != nil || cfg == nil {
		return services.EmptyTagEnforcementConfig{}
	}
	return cfg
}

// loadMaintainerGate builds a maintainer.Gate from the project's
// .sharkconfig.json. When no config is present, NewFileGate receives nil
// and returns a gate that always denies — safe default for HTTP wiring
// since attach/detach paths never consume the gate.
func loadMaintainerGate(projectRoot string) maintainer.Gate {
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	mgr := config.NewManager(configPath)
	cfg, _ := mgr.Load()
	var mc *config.MaintainerConfig
	if cfg != nil {
		mc = cfg.Maintainer
	}
	return maintainer.NewFileGate(projectRoot, mc, mc.CacheWindow())
}

// Compile-time interface satisfaction checks for adapters.
var (
	_ services.WorkSessionRepository             = (*workSessionAdapter)(nil)
	_ services.TaskHistoryRepository             = (*taskHistoryAdapter)(nil)
	_ services.ViewerEntityDocRepository         = (*entityDocAdapter)(nil)
	_ services.ViewerIdeaRepository              = (*ideaAdapter)(nil)
	_ services.ViewerBugListRepository           = (*bugListAdapter)(nil)
	_ services.ViewerChangeCardListRepository    = (*changeCardListAdapter)(nil)
	_ services.ViewerEntityNoteRepository        = (*repnote.EntityNoteRepository)(nil)
	_ services.ViewerEntityDocByEntityRepository = (*entitydoc.EntityDocumentRepository)(nil)
)

// ideaAdapter adapts *idea.IdeaRepository to services.ViewerIdeaRepository.
type ideaAdapter struct {
	repo *idea.IdeaRepository
}

func (a *ideaAdapter) ListAll(ctx context.Context) ([]*models.Idea, error) {
	return a.repo.List(ctx, nil)
}

func (a *ideaAdapter) GetByKey(ctx context.Context, key string) (*models.Idea, error) {
	return a.repo.GetByKey(ctx, key)
}

// bugListAdapter adapts *bug.BugRepository to services.ViewerBugListRepository.
// terminalStatuses comes from workflow.Service.ForLevel(LevelBug).GetTerminalStatuses()
// and is injected at construction time so the adapter uses the project-specific
// terminal set rather than the hardcoded fallback.
type bugListAdapter struct {
	repo             *bug.BugRepository
	terminalStatuses []string
}

func (a *bugListAdapter) ListAll(ctx context.Context, includeTerminal bool) ([]*models.Bug, error) {
	return a.repo.List(ctx, &bug.BugListFilters{
		IncludeTerminal:  includeTerminal,
		TerminalStatuses: a.terminalStatuses,
	})
}

func (a *bugListAdapter) GetByKey(ctx context.Context, key string) (*models.Bug, error) {
	return a.repo.GetByKey(ctx, key)
}

// changeCardListAdapter adapts *changecard.ChangeCardRepository to services.ViewerChangeCardListRepository.
// terminalStatuses comes from workflow.Service.ForLevel(LevelChange).GetTerminalStatuses()
// and is injected at construction time so the adapter uses the project-specific
// terminal set rather than the hardcoded fallback.
type changeCardListAdapter struct {
	repo             *changecard.ChangeCardRepository
	terminalStatuses []string
}

func (a *changeCardListAdapter) ListAll(ctx context.Context, includeTerminal bool) ([]*models.ChangeCard, error) {
	return a.repo.List(ctx, &changecard.ChangeCardRepoFilter{
		IncludeTerminal:  includeTerminal,
		TerminalStatuses: a.terminalStatuses,
	})
}

func (a *changeCardListAdapter) GetByKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	return a.repo.GetByKey(ctx, key)
}

// sprintAnalyticsAdapter adapts *sprintrepo.SprintAnalyticsRepository to the
// services.SprintAnalyticsRepository interface. The concrete repository returns
// repository-layer DTO types, but the service layer uses service-owned DTOs so
// the services package stays repository-agnostic.
type sprintAnalyticsAdapter struct {
	repo *sprintrepo.SprintAnalyticsRepository
}

func (a *sprintAnalyticsAdapter) GetVelocityData(ctx context.Context, limit int) ([]services.AnalyticsVelocityRow, error) {
	rows, err := a.repo.GetVelocityData(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]services.AnalyticsVelocityRow, len(rows))
	for i, r := range rows {
		out[i] = services.AnalyticsVelocityRow{
			SprintKey:        r.SprintKey,
			SprintName:       r.SprintName,
			CompletedSize:    r.CompletedSize,
			UnsizedCompleted: r.UnsizedCompleted,
		}
	}
	return out, nil
}

func (a *sprintAnalyticsAdapter) GetSprintAssignedEntities(ctx context.Context, sprintID int64) ([]services.AnalyticsAssignedEntity, error) {
	entities, err := a.repo.GetSprintAssignedEntities(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	out := make([]services.AnalyticsAssignedEntity, len(entities))
	for i, e := range entities {
		out[i] = services.AnalyticsAssignedEntity{
			EntityType: e.EntityType,
			EntityID:   e.EntityID,
			AssignedAt: e.AssignedAt,
			RemovedAt:  e.RemovedAt,
			Size:       e.Size,
		}
	}
	return out, nil
}

func (a *sprintAnalyticsAdapter) GetCompletionEvents(ctx context.Context, sprintID int64, start, end time.Time) ([]services.AnalyticsCompletionEvent, error) {
	events, err := a.repo.GetCompletionEvents(ctx, sprintID, start, end)
	if err != nil {
		return nil, err
	}
	out := make([]services.AnalyticsCompletionEvent, len(events))
	for i, ev := range events {
		out[i] = services.AnalyticsCompletionEvent{
			EntityID:   ev.EntityID,
			EntityType: ev.EntityType,
			NewStatus:  ev.NewStatus,
			Timestamp:  ev.Timestamp,
		}
	}
	return out, nil
}

func (a *sprintAnalyticsAdapter) GetCycleTimeByPhase(ctx context.Context, sprintID int64) ([]services.AnalyticsPhaseTimeRow, error) {
	rows, err := a.repo.GetCycleTimeByPhase(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	out := make([]services.AnalyticsPhaseTimeRow, len(rows))
	for i, r := range rows {
		out[i] = services.AnalyticsPhaseTimeRow{
			Phase:       r.Phase,
			AverageDays: r.AverageDays,
		}
	}
	return out, nil
}

// entityDocAdapter adapts *entitydoc.EntityDocumentRepository to the
// services.ViewerEntityDocRepository interface, converting BulkDoc types.
type entityDocAdapter struct {
	repo *entitydoc.EntityDocumentRepository
}

func (a *entityDocAdapter) ListAll(ctx context.Context) ([]*services.BulkEntityDoc, error) {
	raw, err := a.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*services.BulkEntityDoc, len(raw))
	for i, d := range raw {
		out[i] = &services.BulkEntityDoc{
			EntityType: d.EntityType,
			EntityID:   d.EntityID,
			Title:      d.Title,
			FilePath:   d.FilePath,
		}
	}
	return out, nil
}

// workSessionAdapter adapts *repository.WorkSessionRepository to the services.WorkSessionRepository interface.
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
	MutationService   *viewerapi.MutationService
	EditSvc           *services.EditService
	// TagService is constructed once and injected into every entity service
	// for tag attach/detach and tag_required_for enforcement.
	TagService    *services.TagService
	SearchService *services.SearchService
}

// WireServices constructs all services with their dependencies.
// This is the dependency injection root for the HTTP API and the shark web CLI command.
//
// Parameters:
//   - db: database connection (shared across all services)
//   - projectRoot: path to project root for workflow config
//
// Returns:
//   - *ServiceContainer: all services ready to inject into handlers
func WireServices(db *repository.DB, projectRoot string) *ServiceContainer {
	// Step 1: Initialize workflow service (shared dependency)
	workflowSvc := workflow.NewService(projectRoot)

	// Step 2: Construct repositories (data access layer)
	taskRepo := repository.NewTaskRepository(db)
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

	// Step 3c: Construct the shared TagService once and inject it into every
	// entity service. Mirrors the CLI accessor so both entry points behave
	// identically for create-with-tag, attach/detach, and tag_required_for
	// enforcement. Empty-stub fallback disables enforcement if config load fails.
	tagEnforcementCfg := loadTagEnforcementConfig(projectRoot)
	tagGate := loadMaintainerGate(projectRoot)
	tagSvc := services.NewTagService(
		tagrepo.NewTagRepository(db),
		tagrepo.NewEntityTagRepository(db),
		tagGate,
		tagEnforcementCfg,
	)
	searchRepo := repository.NewSearchRepository(db)

	// Step 4: Construct entity-specific services
	taskService := services.NewTaskService(taskRepo, entitySvc, creatorSvc)
	taskService.SetEntityHistoryRepo(entityHistoryRepo)
	taskService.SetTagService(tagSvc)
	taskService.SetSearchIndexer(searchRepo)

	featureService := services.NewFeatureService(
		featureRepo, entitySvc,
		registry.MustGetRepository(models.EntityTypeFeature),
		taskRepo, epicRepo,
	)
	featureService.SetEntityHistoryRepo(entityHistoryRepo)
	featureService.SetTagService(tagSvc)
	featureService.SetSearchIndexer(searchRepo)

	// Wire FeatureService into TaskService for auto-reopen behavior
	taskService.SetFeatureService(featureService)

	// Wire cascade reopen dependencies
	taskService.SetCascadeDeps(db, featureRepo, epicRepo, entityHistoryRepo, entityHistoryRepo)
	featureService.SetCascadeDeps(db, epicRepo, entityHistoryRepo, entityHistoryRepo)

	epicService := services.NewEpicService(
		epicRepo,
		entitySvc,
		registry.MustGetRepository(models.EntityTypeEpic),
		featureRepo,
		taskRepo,
	)
	epicService.SetTagService(tagSvc)
	epicService.SetSearchIndexer(searchRepo)

	analyticsSvc := services.NewEpicAnalyticsService(epicRepo, taskRepo)
	epicService.SetAnalyticsService(analyticsSvc)

	bugService := services.NewBugService(
		bugRepoAdapter,
		entitySvc,
		registry.MustGetRepository(models.EntityTypeBug),
		epicRepo, featureRepo, taskRepo,
		projectRoot,
		tagSvc,
	)
	bugService.SetSearchIndexer(searchRepo)

	changeCardService := services.NewChangeCardService(
		changeCardRepoAdapter,
		entitySvc,
		registry.MustGetRepository(models.EntityTypeChange),
		epicRepo, featureRepo,
		projectRoot,
	)
	changeCardService.SetTagService(tagSvc)
	changeCardService.SetSearchIndexer(searchRepo)

	noteService, err := services.NewNoteService(noteRepo, registry)
	if err != nil {
		panic(fmt.Sprintf("failed to create NoteService: %v", err))
	}
	noteService.SetSearchIndexer(searchRepo)
	contextService, err := services.NewContextService(registry)
	if err != nil {
		panic(fmt.Sprintf("failed to create ContextService: %v", err))
	}
	resumeService, err := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo, registry)
	if err != nil {
		panic(fmt.Sprintf("failed to create ResumeService: %v", err))
	}

	// Step 5: Construct EntityRelationshipService for use by ViewerService.
	entityRelRepo := entityrel.NewEntityRelationshipRepository(db)
	entityRelSvc := services.NewEntityRelationshipService(entityRelRepo, taskRepo)

	// Step 5a: Construct SprintService and SprintAnalyticsService for Sprint-mode viewer data.
	sprintRepo := sprintrepo.NewSprintRepository(db)
	sprintAnalyticsRepo := sprintrepo.NewSprintAnalyticsRepository(db)
	sprintSvc := services.NewSprintService(sprintRepo, workflowSvc, sprintRepo, sprintRepo, nil, db)
	sprintAnalyticsSvc := services.NewSprintAnalyticsService(&sprintAnalyticsAdapter{repo: sprintAnalyticsRepo}, sprintRepo)

	techDebtRepo := techdebtrepo.NewTechDebtRepository(db)

	// Step 5b: Construct ViewerService for the read-only dashboard API.
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
		entityRelSvc,
		registry,
	)
	viewerService.WithEntityDocRepo(&entityDocAdapter{repo: entitydoc.NewEntityDocumentRepository(db)})
	viewerService.WithIdeaRepo(&ideaAdapter{repo: idea.NewIdeaRepository(db)})
	viewerService.WithBugListRepo(&bugListAdapter{
		repo:             bugRepoAdapter,
		terminalStatuses: workflowSvc.ForLevel(workflow.LevelBug).GetTerminalStatuses(),
	})
	viewerService.WithChangeCardListRepo(&changeCardListAdapter{
		repo:             changeCardRepoAdapter,
		terminalStatuses: workflowSvc.ForLevel(workflow.LevelChange).GetTerminalStatuses(),
	})
	viewerService.WithTechDebtRepo(techDebtRepo)
	viewerService.WithNoteRepo(repnote.NewEntityNoteRepository(db))
	viewerService.WithDocByEntityRepo(entitydoc.NewEntityDocumentRepository(db))
	viewerService.WithSprintService(sprintSvc)
	viewerService.WithSprintAnalyticsService(sprintAnalyticsSvc)
	// Wire tag service for decoration and filtering (REQ-F-015, T-E28-F06-002).
	// REQ-F-014: MaintainerGate is NOT passed to ViewerService (defense-in-depth).
	viewerService.WithTagService(tagSvc)

	// Step 5c: Construct MutationService for the viewer mutation API.
	mutationService := viewerapi.NewMutationService(
		epicService,
		featureService,
		taskService,
		noteService,
		entityRelSvc,
		viewerapi.NewRegistryEntityResolver(registry),
	)

	// Step 6: Construct EditService for the file-write endpoint.
	editService := services.NewEditService(projectRoot)

	// Step 7: Construct SearchService with TagService wired for tag post-filter
	// (REQ-F-011, spec §2.8.3). tagSvc is passed as the second argument so that
	// --tag filtering works on the search path.
	searchService := services.NewSearchService(searchRepo, tagSvc)

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
		MutationService:   mutationService,
		EditSvc:           editService,
		TagService:        tagSvc,
		SearchService:     searchService,
	}
}
