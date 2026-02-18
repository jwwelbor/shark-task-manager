package cli

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

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
	workflowSvc := GetWorkflowService()
	// TODO: noteRepo interface mismatch - EpicNoteRepository expects different signature
	return services.NewEpicService(epicRepo, workflowSvc, nil, featureRepo, taskRepo)
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
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	workflowSvc := GetWorkflowService()
	// TODO: noteRepo interface mismatch - FeatureNoteRepository expects different signature
	return services.NewFeatureService(featureRepo, workflowSvc, nil, taskRepo)
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
