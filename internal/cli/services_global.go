package cli

import (
	"context"
	"fmt"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

var (
	globalNoteService *services.NoteService
	noteServiceOnce   sync.Once
	noteServiceErr    error

	globalContextService *services.ContextService
	contextServiceOnce   sync.Once
	contextServiceErr    error

	globalResumeService *services.ResumeService
	resumeServiceOnce   sync.Once
	resumeServiceErr    error
)

// GetNoteService returns the global NoteService, initializing it if needed.
func GetNoteService(ctx context.Context) (*services.NoteService, error) {
	noteServiceOnce.Do(func() {
		db, err := GetDB(ctx)
		if err != nil {
			noteServiceErr = fmt.Errorf("failed to get database for NoteService: %w", err)
			return
		}

		noteRepo := repository.NewEntityNoteRepository(db)
		epicRepo := repository.NewEpicRepository(db)
		featureRepo := repository.NewFeatureRepository(db)
		taskRepo := repository.NewTaskRepository(db)

		globalNoteService = services.NewNoteService(noteRepo, epicRepo, featureRepo, taskRepo)
	})

	if noteServiceErr != nil {
		return nil, noteServiceErr
	}
	return globalNoteService, nil
}

// GetContextService returns the global ContextService, initializing it if needed.
func GetContextService(ctx context.Context) (*services.ContextService, error) {
	contextServiceOnce.Do(func() {
		db, err := GetDB(ctx)
		if err != nil {
			contextServiceErr = fmt.Errorf("failed to get database for ContextService: %w", err)
			return
		}

		epicRepo := repository.NewEpicRepository(db)
		featureRepo := repository.NewFeatureRepository(db)
		taskRepo := repository.NewTaskRepository(db)

		globalContextService = services.NewContextService(epicRepo, featureRepo, taskRepo)
	})

	if contextServiceErr != nil {
		return nil, contextServiceErr
	}
	return globalContextService, nil
}

// GetResumeService returns the global ResumeService, initializing it if needed.
func GetResumeService(ctx context.Context) (*services.ResumeService, error) {
	resumeServiceOnce.Do(func() {
		db, err := GetDB(ctx)
		if err != nil {
			resumeServiceErr = fmt.Errorf("failed to get database for ResumeService: %w", err)
			return
		}

		epicRepo := repository.NewEpicRepository(db)
		featureRepo := repository.NewFeatureRepository(db)
		taskRepo := repository.NewTaskRepository(db)
		noteRepo := repository.NewEntityNoteRepository(db)

		globalResumeService = services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo)
	})

	if resumeServiceErr != nil {
		return nil, resumeServiceErr
	}
	return globalResumeService, nil
}

// GetTaskService returns a TaskService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Pattern:
//   - Creates new service instance per call (lightweight, no shared state)
//   - Reuses global DB and workflow service (expensive to recreate)
//   - Panics on DB failure (fail-fast for CLI entry points)
//   - Optional dependencies (creatorSvc, noteRepo) passed as nil for now
//
// Usage:
//
//	svc := cli.GetTaskService()
//	task, err := svc.StartTask(ctx, "E07-F01-001", "agent123")
func GetTaskService() *services.TaskService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	taskRepo := repository.NewTaskRepository(db)
	workflowSvc := GetWorkflowService()
	noteRepo := repository.NewEntityNoteRepository(db)

	// Wire taskcreation.Creator for task file creation support
	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}
	historyRepo := repository.NewTaskHistoryRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("") // use embedded templates
	renderer := templates.NewRenderer(loader)
	creatorSvc := taskcreation.NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, projectRoot, workflowSvc)

	return services.NewTaskService(taskRepo, workflowSvc, creatorSvc, noteRepo)
}

// GetTaskServiceWithDeps returns a TaskService with relationship and document repositories wired.
// Used by commands that need dependency/relationship management (unlink, deps) or document operations.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetTaskServiceWithDeps()
//	count, err := svc.UnlinkRelationships(ctx, taskKey, relType, targetKeys)
func GetTaskServiceWithDeps() *services.TaskService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	taskRepo := repository.NewTaskRepository(db)
	workflowSvc := GetWorkflowService()
	noteRepo := repository.NewEntityNoteRepository(db)

	// Wire taskcreation.Creator for task file creation support
	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}
	historyRepo := repository.NewTaskHistoryRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creatorSvc := taskcreation.NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, projectRoot, workflowSvc)

	relRepo := repository.NewTaskRelationshipRepository(db)
	docRepo := repository.NewDocumentRepository(db)
	sessionRepo := &workSessionAdapter{repo: repository.NewWorkSessionRepository(db)}

	svc := services.NewTaskServiceWithRelationships(taskRepo, workflowSvc, creatorSvc, noteRepo, docRepo, relRepo, sessionRepo)
	svc.SetDepRepo(relRepo)
	svc.SetRelQueryRepo(relRepo)
	svc.SetWritableDocRepo(docRepo)
	return svc
}

// ResetServices clears global service state. For testing only.
func ResetServices() {
	globalNoteService = nil
	noteServiceErr = nil
	noteServiceOnce = sync.Once{}

	globalContextService = nil
	contextServiceErr = nil
	contextServiceOnce = sync.Once{}

	globalResumeService = nil
	resumeServiceErr = nil
	resumeServiceOnce = sync.Once{}
}
