package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	sprintrepo "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/status"
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

// resumeSessionAdapter adapts *repository.WorkSessionRepository to the services.ResumeWorkSessionRepository interface.
type resumeSessionAdapter struct {
	repo *repository.WorkSessionRepository
}

func (a *resumeSessionAdapter) GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error) {
	return a.repo.GetByTaskID(ctx, taskID)
}

func (a *resumeSessionAdapter) GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*services.ResumeSessionStats, error) {
	stats, err := a.repo.GetSessionStatsByTaskID(ctx, taskID)
	if err != nil || stats == nil {
		return nil, err
	}
	return &services.ResumeSessionStats{
		TotalSessions:   stats.TotalSessions,
		TotalDuration:   stats.TotalDuration,
		AverageDuration: stats.AverageDuration,
		ActiveSession:   stats.ActiveSession,
	}, nil
}

func (a *resumeSessionAdapter) GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error) {
	return a.repo.GetActiveSessionByTaskID(ctx, taskID)
}

// sprintAnalyticsAdapter adapts *sprintrepo.SprintAnalyticsRepository to the
// services.SprintAnalyticsRepository interface. The concrete repository returns
// repository-layer DTO types (sprint.VelocityRow, etc.) but the service
// interface uses service-owned types (services.AnalyticsVelocityRow, etc.) so
// that the services package does not import the repository package.
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

// GetEpicService returns an EpicService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Pattern:
//   - Creates new service instance per call (lightweight, no shared state)
//   - Reuses global DB and workflow service (expensive to recreate)
//   - Panics on DB failure (fail-fast for CLI entry points)
//   - Optional dependencies (noteRepo, featureRepo) provided for full functionality
//
// Usage:
//
//	svc := cli.GetEpicService()
//	result, err := svc.TransitionStatus(ctx, "E07", "in_progress", opts)
func GetEpicService() *services.EpicService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	workflowSvc := GetWorkflowService()
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepositoryWithWorkflow(db, workflowSvc.GetWorkflow())
	docRepo := repository.NewDocumentRepository(db)
	enrichRepo := repository.NewTemplateEnrichmentRepository(db)
	entitySvc := GetEntityService()
	entityRepo := GetEntityRegistry().MustGetRepository(models.EntityTypeEpic)
	entityDocRepo := repository.NewEntityDocumentRepository(db)
	docAdapter := repository.NewPolymorphicDocRepoAdapter(entityDocRepo)
	svc := services.NewEpicService(epicRepo, entitySvc, entityRepo, featureRepo, taskRepo)
	svc.SetTracer(GetTracer("shark/services/epic"))
	svc.SetDocRepo(docAdapter)
	svc.SetWritableDocRepo(docRepo, entityDocRepo)
	svc.SetEnrichRepo(enrichRepo)
	// E28-F04 T-008: pass the shared *TagService so EpicService can
	// enforce `tag_required_for` on create and honour --tag on create/update.
	svc.SetTagService(GetTagService())
	svc.SetSizeEnforcement(getSizeEnforcement())

	// Wire the analytics sub-service explicitly to avoid lazy-init on every call.
	analyticsSvc := services.NewEpicAnalyticsService(epicRepo, taskRepo)
	svc.SetAnalyticsService(analyticsSvc)

	return svc
}

// GetFeatureService returns a FeatureService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Pattern:
//   - Creates new service instance per call (lightweight, no shared state)
//   - Reuses global DB and workflow service (expensive to recreate)
//   - Panics on DB failure (fail-fast for CLI entry points)
//   - Optional dependencies (noteRepo, taskRepo) provided for full functionality
//
// Usage:
//
//	svc := cli.GetFeatureService()
//	result, err := svc.TransitionStatus(ctx, "E07-F01", "in_progress", opts)
func GetFeatureService() *services.FeatureService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	workflowSvc := GetWorkflowService()
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepositoryWithWorkflow(db, workflowSvc.GetWorkflow())
	docRepo := repository.NewDocumentRepository(db)
	enrichRepo := repository.NewTemplateEnrichmentRepository(db)
	entitySvc := GetEntityService()
	entityRepo := GetEntityRegistry().MustGetRepository(models.EntityTypeFeature)
	entityDocRepo := repository.NewEntityDocumentRepository(db)
	docAdapter := repository.NewPolymorphicDocRepoAdapter(entityDocRepo)
	entityHistoryRepo := repository.NewEntityHistoryRepository(db)
	svc := services.NewFeatureService(featureRepo, entitySvc, entityRepo, taskRepo, epicRepo)
	svc.SetTracer(GetTracer("shark/services/feature"))
	svc.SetDocRepo(docAdapter)
	svc.SetWritableDocRepo(docRepo, entityDocRepo)
	svc.SetEnrichRepo(enrichRepo)
	svc.SetEntityHistoryRepo(entityHistoryRepo)
	// E28-F04 T-007: pass the shared *TagService so FeatureService can
	// enforce `tag_required_for` on create and honour --tag on create/update.
	svc.SetTagService(GetTagService())
	svc.SetSizeEnforcement(getSizeEnforcement())

	// Wire cascade reopen dependencies so that a feature regression automatically
	// reopens a terminal epic ancestor (AC-T3 / REQ-F-001).
	svc.SetCascadeDeps(db, epicRepo, entityHistoryRepo, entityHistoryRepo)

	// Wire the progress sub-service explicitly to avoid lazy-init on every call.
	progressSvc := services.NewFeatureProgressService(featureRepo, taskRepo, workflowSvc)
	svc.SetProgressService(progressSvc)

	return svc
}

// GetIdeaService returns an IdeaService instance.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetIdeaService()
//	idea, err := svc.CreateIdea(cmd.Context(), services.CreateIdeaInput{Title: "My idea"})
func GetIdeaService() *services.IdeaService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	ideaRepo := repository.NewIdeaRepository(db)
	svc, err := services.NewIdeaService(ideaRepo)
	if err != nil {
		panic(fmt.Sprintf("failed to create IdeaService: %v", err))
	}

	// E28-F04 T-010: wire the shared *TagService so IdeaService can
	// enforce `tag_required_for` on create and honour --tag on create/update.
	svc.SetTagService(GetTagService())
	svc.SetSizeEnforcement(getSizeEnforcement())

	return svc
}

// GetDisplayService returns a DisplayService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func GetDisplayService() *services.DisplayService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	workflowSvc := GetWorkflowService()
	deps := services.DisplayServiceDeps{
		EpicRepo:               repository.NewEpicRepository(db),
		FeatureRepo:            repository.NewFeatureRepository(db),
		TaskRepo:               repository.NewTaskRepositoryWithWorkflow(db, workflowSvc.GetWorkflow()),
		DocumentRepo:           repository.NewPolymorphicDocRepoAdapter(repository.NewEntityDocumentRepository(db)),
		NoteRepo:               repository.NewEntityNoteRepository(db),
		TaskRelRepo:            repository.NewEntityRelTaskKeyAdapter(db),
		TemplateEnrichmentRepo: repository.NewTemplateEnrichmentRepository(db),
	}
	return services.NewDisplayService(deps, workflowSvc)
}

// GetSearchService returns a SearchService instance.
// Creates a new instance each call with the global DB connection.
// TagService is wired so that --tag filtering works on the search path
// (REQ-F-011, spec §2.8.1).
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetSearchService()
//	results, err := svc.SearchAll(ctx, "query", "", nil)
func GetSearchService() *services.SearchService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	searchRepo := repository.NewSearchRepository(db)
	return services.NewSearchService(searchRepo, GetTagService())
}

// GetRecentService returns a RecentService instance.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func GetRecentService() *services.RecentService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	workflowSvc := GetWorkflowService()
	taskRepo := repository.NewTaskRepositoryWithWorkflow(db, workflowSvc.GetWorkflow())
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	bugRepo := repository.NewBugRepository(db)
	changeRepo := repository.NewChangeCardRepository(db)
	ideaRepo := repository.NewIdeaRepository(db)
	techDebtRepo := repository.NewTechDebtRepository(db)
	return services.NewRecentService(taskRepo, featureRepo, epicRepo, bugRepo, changeRepo, ideaRepo, techDebtRepo)
}

// GetStatusService returns a StatusService instance.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetStatusService()
//	dashboard, err := svc.GetDashboard(ctx, req)
func GetStatusService() *status.StatusService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	bugRepo := repository.NewBugRepository(db)
	ccRepo := repository.NewChangeCardRepository(db)
	return status.NewStatusService(db,
		status.WithBugRepository(bugRepo),
		status.WithChangeCardRepository(ccRepo),
	)
}
