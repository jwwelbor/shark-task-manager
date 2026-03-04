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

		svc := services.NewNoteService(noteRepo, epicRepo, featureRepo, taskRepo)
		changeCardRepo := repository.NewChangeCardRepository(db)
		svc.SetChangeCardRepo(changeCardRepo)
		bugRepo := repository.NewBugRepository(db)
		svc.SetBugRepo(bugRepo)
		globalNoteService = svc
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

		svc := services.NewContextService(epicRepo, featureRepo, taskRepo)
		bugRepo := repository.NewBugRepository(db)
		svc.SetBugRepo(bugRepo)
		changeCardRepo := repository.NewChangeCardRepository(db)
		svc.SetChangeCardRepo(changeCardRepo)
		globalContextService = svc
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
	featureRepo *repository.FeatureRepository
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
	workflowSvc := GetWorkflowService()
	taskRepo := repository.NewTaskRepositoryWithWorkflow(db, workflowSvc.GetWorkflow())
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
		featureRepo: featureRepo,
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
//	task, err := svc.AdvanceTaskStatus(ctx, "E07-F01-001")
func GetTaskService() *services.TaskService {
	d := buildTaskServiceDeps()
	svc := services.NewTaskService(d.taskRepo, d.workflowSvc, d.creatorSvc, d.noteRepo)
	svc.SetFeatureRepo(d.featureRepo)
	svc.SetFeatureService(GetFeatureService())

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
	svc := services.NewTaskService(d.taskRepo, d.workflowSvc, d.creatorSvc, d.noteRepo)
	svc.SetFeatureRepo(d.featureRepo)
	svc.SetHistoryRepo(&taskHistoryAdapter{repo: d.historyRepo})
	svc.SetFeatureService(GetFeatureService())

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

	svc := services.NewTaskService(d.taskRepo, d.workflowSvc, d.creatorSvc, d.noteRepo)
	svc.SetDocRepo(docRepo)
	svc.SetRelRepo(relRepo)
	svc.SetSessionRepo(sessionRepo)
	svc.SetFeatureRepo(d.featureRepo)
	svc.SetDepRepo(relRepo)
	svc.SetRelQueryRepo(relRepo)
	svc.SetWritableDocRepo(docRepo)

	// Wire sub-services for query and dependency delegation.
	querySvc := services.NewTaskQueryService(d.taskRepo)
	svc.SetQueryService(querySvc)

	depSvc := services.NewTaskDependencyService(d.taskRepo)
	depSvc.SetDepRepo(relRepo)
	depSvc.SetRelQueryRepo(relRepo)
	depSvc.SetWritableDocRepo(docRepo)
	svc.SetDependencyService(depSvc)

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
	workflowSvc := GetWorkflowService()
	changeCardRepo := repository.NewChangeCardRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)

	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}

	return services.NewChangeCardService(changeCardRepo, workflowSvc, epicRepo, featureRepo, projectRoot)
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
	workflowSvc := GetWorkflowService()
	bugRepo := repository.NewBugRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	return services.NewBugService(bugRepo, workflowSvc, epicRepo, featureRepo, taskRepo)
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
