package cli

import (
	"context"
	"fmt"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/validation"
	"github.com/jwwelbor/shark-task-manager/internal/view"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
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
		sessionRepo := &resumeSessionAdapter{repo: repository.NewWorkSessionRepository(db)}

		svc := services.NewResumeService(epicRepo, featureRepo, taskRepo, noteRepo)
		svc.SetSessionRepo(sessionRepo)
		globalResumeService = svc
	})

	if resumeServiceErr != nil {
		return nil, resumeServiceErr
	}
	return globalResumeService, nil
}

// taskServiceDeps holds the shared dependencies for constructing a TaskService.
// Extracted to eliminate duplication across GetTaskService, GetTaskServiceWithHistory,
// and GetTaskServiceWithDeps which all wire the same base dependencies.
type taskServiceDeps struct {
	db          *repository.DB
	taskRepo    *repository.TaskRepository
	workflowSvc *workflow.Service
	noteRepo    *repository.EntityNoteRepository
	creatorSvc  *taskcreation.Creator
	historyRepo *repository.TaskHistoryRepository
}

// buildTaskServiceDeps constructs the shared dependencies for TaskService variants.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func buildTaskServiceDeps() taskServiceDeps {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	taskRepo := repository.NewTaskRepository(db)
	workflowSvc := GetWorkflowService()
	noteRepo := repository.NewEntityNoteRepository(db)

	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}
	historyRepo := repository.NewTaskHistoryRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	renderer := templates.NewRenderer(templates.NewLoader(""))
	creatorSvc := taskcreation.NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, projectRoot, workflowSvc)

	return taskServiceDeps{
		db:          db,
		taskRepo:    taskRepo,
		workflowSvc: workflowSvc,
		noteRepo:    noteRepo,
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
//	task, err := svc.StartTask(ctx, "E07-F01-001", "agent123")
func GetTaskService() *services.TaskService {
	d := buildTaskServiceDeps()
	return services.NewTaskService(d.taskRepo, d.workflowSvc, d.creatorSvc, d.noteRepo)
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
	svc := services.NewTaskService(d.taskRepo, d.workflowSvc, d.creatorSvc, d.noteRepo)
	svc.SetHistoryRepo(&taskHistoryAdapter{repo: d.historyRepo})
	return svc
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
	d := buildTaskServiceDeps()
	relRepo := repository.NewTaskRelationshipRepository(d.db)
	docRepo := repository.NewDocumentRepository(d.db)
	sessionRepo := &workSessionAdapter{repo: repository.NewWorkSessionRepository(d.db)}

	svc := services.NewTaskServiceWithRelationships(d.taskRepo, d.workflowSvc, d.creatorSvc, d.noteRepo, docRepo, relRepo, sessionRepo)
	svc.SetDepRepo(relRepo)
	svc.SetRelQueryRepo(relRepo)
	svc.SetWritableDocRepo(docRepo)
	return svc
}

// GetCriteriaService returns a CriteriaService instance.
// Creates a new instance each call with the global DB connection.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
//
// Usage:
//
//	svc := cli.GetCriteriaService()
//	count, err := svc.ImportCriteriaFromFile(ctx, "E07-F01-001")
func GetCriteriaService() *services.CriteriaService {
	db, err := GetDB(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to get database: %v", err))
	}
	criteriaRepo := repository.NewTaskCriteriaRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	return services.NewCriteriaService(criteriaRepo, taskRepo, featureRepo)
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
	return view.NewService(epicRepo, featureRepo, taskRepo)
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
