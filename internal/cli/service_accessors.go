package cli

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
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
	return services.NewFeatureServiceWithRelationships(featureRepo, workflowSvc, noteRepo, taskRepo, docRepo, nil, epicRepo)
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
