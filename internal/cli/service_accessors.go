package cli

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/status"
)

// epicNoteAdapter adapts *repository.EntityNoteRepository to the services.EpicNoteRepository interface.
// The service interface uses (string, string) for entityType and documentPath,
// while the repository uses (models.EntityType, *string).
type epicNoteAdapter struct {
	repo *repository.EntityNoteRepository
}

func (a *epicNoteAdapter) CreateRejectionNote(
	ctx context.Context,
	entityType string,
	entityID int64,
	historyID int64,
	fromStatus, toStatus, reason, rejectedBy, documentPath string,
) error {
	var dp *string
	if documentPath != "" {
		dp = &documentPath
	}
	_, err := a.repo.CreateRejectionNote(ctx, models.EntityType(entityType), entityID, historyID,
		fromStatus, toStatus, reason, rejectedBy, dp)
	return err
}

// featureNoteAdapter adapts *repository.EntityNoteRepository to the services.FeatureNoteRepository interface.
// Same bridging logic as epicNoteAdapter.
type featureNoteAdapter struct {
	repo *repository.EntityNoteRepository
}

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

func (a *featureNoteAdapter) CreateRejectionNote(
	ctx context.Context,
	entityType string,
	entityID int64,
	historyID int64,
	fromStatus, toStatus, reason, rejectedBy, documentPath string,
) error {
	var dp *string
	if documentPath != "" {
		dp = &documentPath
	}
	_, err := a.repo.CreateRejectionNote(ctx, models.EntityType(entityType), entityID, historyID,
		fromStatus, toStatus, reason, rejectedBy, dp)
	return err
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
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	docRepo := repository.NewDocumentRepository(db)
	noteRepo := &epicNoteAdapter{repo: repository.NewEntityNoteRepository(db)}
	workflowSvc := GetWorkflowService()
	svc := services.NewEpicService(epicRepo, workflowSvc, noteRepo, featureRepo, taskRepo)
	svc.SetDocRepo(docRepo)
	svc.SetWritableDocRepo(docRepo)
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
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	noteRepo := &featureNoteAdapter{repo: repository.NewEntityNoteRepository(db)}
	docRepo := repository.NewDocumentRepository(db)
	workflowSvc := GetWorkflowService()
	svc := services.NewFeatureServiceWithRelationships(featureRepo, workflowSvc, noteRepo, taskRepo, docRepo, nil, epicRepo)
	svc.SetWritableDocRepo(docRepo)
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
	return status.NewStatusService(db)
}
