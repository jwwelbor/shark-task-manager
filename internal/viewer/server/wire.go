package server

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/bug"
	"github.com/jwwelbor/shark-task-manager/internal/repository/changecard"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entitydoc"
	"github.com/jwwelbor/shark-task-manager/internal/repository/idea"
	repnote "github.com/jwwelbor/shark-task-manager/internal/repository/note"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// Compile-time interface satisfaction checks for adapters.
var (
	_ services.WorkSessionRepository             = (*workSessionAdapter)(nil)
	_ services.TaskHistoryRepository             = (*taskHistoryAdapter)(nil)
	_ services.ViewerEntityDocRepository         = (*entityDocAdapter)(nil)
	_ services.ViewerIdeaRepository              = (*ideaAdapter)(nil)
	_ services.ViewerBugListRepository           = (*bugListAdapter)(nil)
	_ services.ViewerChangeCardListRepository    = (*changeCardListAdapter)(nil)
	_ services.ViewerTaskRelationshipRepository  = (*taskRelAdapter)(nil)
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
type bugListAdapter struct {
	repo *bug.BugRepository
}

func (a *bugListAdapter) ListAll(ctx context.Context) ([]*models.Bug, error) {
	return a.repo.List(ctx, nil)
}

func (a *bugListAdapter) GetByKey(ctx context.Context, key string) (*models.Bug, error) {
	return a.repo.GetByKey(ctx, key)
}

// changeCardListAdapter adapts *changecard.ChangeCardRepository to services.ViewerChangeCardListRepository.
type changeCardListAdapter struct {
	repo *changecard.ChangeCardRepository
}

func (a *changeCardListAdapter) ListAll(ctx context.Context) ([]*models.ChangeCard, error) {
	return a.repo.List(ctx, nil)
}

func (a *changeCardListAdapter) GetByKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	return a.repo.GetByKey(ctx, key)
}

// taskRelAdapter adapts *repository.DB to services.ViewerTaskRelationshipRepository.
// It runs a single bulk SQL query to fetch all task relationships with resolved keys,
// avoiding N+1 queries in the Hierarchy endpoint. No new per-entity GET endpoint is
// introduced — this data is embedded in the hierarchy payload only (AC-T3).
type taskRelAdapter struct {
	db *repository.DB
}

func (a *taskRelAdapter) ListAll(ctx context.Context) ([]*services.ViewerTaskRelationship, error) {
	const query = `
		SELECT
			tr.from_task_id,
			tr.to_task_id,
			tr.relationship_type,
			ft.key AS from_key,
			tt.key AS to_key
		FROM task_relationships tr
		INNER JOIN tasks ft ON ft.id = tr.from_task_id
		INNER JOIN tasks tt ON tt.id = tr.to_task_id
		ORDER BY tr.id ASC
		LIMIT 10000
	`
	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("taskRelAdapter: failed to list all relationships: %w", err)
	}
	defer rows.Close()

	var out []*services.ViewerTaskRelationship
	for rows.Next() {
		r := &services.ViewerTaskRelationship{}
		if err := rows.Scan(&r.FromTaskID, &r.ToTaskID, &r.RelType, &r.FromKey, &r.ToKey); err != nil {
			return nil, fmt.Errorf("taskRelAdapter: failed to scan relationship: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("taskRelAdapter: error iterating relationships: %w", err)
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
	EditSvc           *services.EditService
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
	viewerService.WithEntityDocRepo(&entityDocAdapter{repo: entitydoc.NewEntityDocumentRepository(db)})
	viewerService.WithIdeaRepo(&ideaAdapter{repo: idea.NewIdeaRepository(db)})
	viewerService.WithBugListRepo(&bugListAdapter{repo: bugRepoAdapter})
	viewerService.WithChangeCardListRepo(&changeCardListAdapter{repo: changeCardRepoAdapter})
	viewerService.WithTaskRelRepo(&taskRelAdapter{db: db})
	viewerService.WithNoteRepo(repnote.NewEntityNoteRepository(db))
	viewerService.WithDocByEntityRepo(entitydoc.NewEntityDocumentRepository(db))

	// Step 6: Construct EditService for the file-write endpoint.
	editService := services.NewEditService(projectRoot)

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
		EditSvc:           editService,
	}
}
