package cli

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
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
	return a.repo.GetSessionAnalyticsByFeature(ctx, featureID, agentType)
}

func (a *workSessionAdapter) GetSessionAnalyticsByEpic(ctx context.Context, epicID int64, agentType *string) (*services.SessionAnalytics, error) {
	return a.repo.GetSessionAnalyticsByEpic(ctx, epicID, agentType)
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
	svc.SetDocRepo(docAdapter)
	svc.SetWritableDocRepo(docRepo, entityDocRepo)
	svc.SetEnrichRepo(enrichRepo)

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
	svc := services.NewFeatureService(featureRepo, entitySvc, entityRepo, taskRepo, epicRepo)
	svc.SetDocRepo(docAdapter)
	svc.SetWritableDocRepo(docRepo, entityDocRepo)
	svc.SetEnrichRepo(enrichRepo)

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
	return services.NewIdeaService(ideaRepo)
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
	return services.NewDisplayService(db, workflowSvc)
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
