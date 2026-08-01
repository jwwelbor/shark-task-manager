package repository

// aliases.go provides backward-compatible type and constructor aliases for
// repository sub-packages. All existing callers continue to use the
// repository.XxxRepository and repository.NewXxxRepository names unchanged.

import (
	"github.com/jwwelbor/shark-task-manager/internal/repository/bug"
	"github.com/jwwelbor/shark-task-manager/internal/repository/changecard"
	"github.com/jwwelbor/shark-task-manager/internal/repository/document"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entitydoc"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entityhistory"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entityrel"
	epicpkg "github.com/jwwelbor/shark-task-manager/internal/repository/epic"
	featurepkg "github.com/jwwelbor/shark-task-manager/internal/repository/feature"
	"github.com/jwwelbor/shark-task-manager/internal/repository/idea"
	"github.com/jwwelbor/shark-task-manager/internal/repository/note"
	"github.com/jwwelbor/shark-task-manager/internal/repository/question"
	"github.com/jwwelbor/shark-task-manager/internal/repository/search"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	taskpkg "github.com/jwwelbor/shark-task-manager/internal/repository/task"
	"github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/jwwelbor/shark-task-manager/internal/repository/templateenrich"
	"github.com/jwwelbor/shark-task-manager/internal/repository/worksession"
)

// --- Question ---

// QuestionRepository is an alias for question.QuestionRepository.
type QuestionRepository = question.QuestionRepository

// QuestionListFilter is an alias for question.QuestionListFilter.
type QuestionListFilter = question.QuestionListFilter

// NewQuestionRepository creates a new QuestionRepository.
var NewQuestionRepository = question.NewQuestionRepository

// --- Idea ---

// IdeaRepository is an alias for idea.IdeaRepository.
type IdeaRepository = idea.IdeaRepository

// IdeaFilter is an alias for idea.IdeaFilter.
type IdeaFilter = idea.IdeaFilter

// NewIdeaRepository creates a new IdeaRepository.
var NewIdeaRepository = idea.NewIdeaRepository

// --- Bug ---

// BugRepository is an alias for bug.BugRepository.
type BugRepository = bug.BugRepository

// BugListFilters is an alias for bug.BugListFilters.
type BugListFilters = bug.BugListFilters

// BugStatusSummary is an alias for bug.BugStatusSummary.
type BugStatusSummary = bug.BugStatusSummary

// BugResolutionStats is an alias for bug.BugResolutionStats.
type BugResolutionStats = bug.BugResolutionStats

// BugFeatureSummary is an alias for bug.BugFeatureSummary.
type BugFeatureSummary = bug.BugFeatureSummary

// NewBugRepository creates a new BugRepository.
var NewBugRepository = bug.NewBugRepository

// --- TechDebt ---

// TechDebtRepository is an alias for techdebt.TechDebtRepository.
type TechDebtRepository = techdebt.TechDebtRepository

// TechDebtFilters is an alias for techdebt.TechDebtFilters.
type TechDebtFilters = techdebt.TechDebtFilters

// NewTechDebtRepository creates a new TechDebtRepository.
var NewTechDebtRepository = techdebt.NewTechDebtRepository

// --- ChangeCard ---

// ChangeCardRepository is an alias for changecard.ChangeCardRepository.
type ChangeCardRepository = changecard.ChangeCardRepository

// ChangeCardRepoFilter is an alias for changecard.ChangeCardRepoFilter.
type ChangeCardRepoFilter = changecard.ChangeCardRepoFilter

// ChangeCardStatusSummary is an alias for changecard.ChangeCardStatusSummary.
type ChangeCardStatusSummary = changecard.ChangeCardStatusSummary

// ChangeCardThroughputStats is an alias for changecard.ChangeCardThroughputStats.
type ChangeCardThroughputStats = changecard.ChangeCardThroughputStats

// NewChangeCardRepository creates a new ChangeCardRepository.
var NewChangeCardRepository = changecard.NewChangeCardRepository

// --- Sprint ---

// SprintRepository is an alias for sprint.SprintRepository.
type SprintRepository = sprint.SprintRepository

// SprintListFilters is an alias for sprint.SprintListFilters.
type SprintListFilters = sprint.SprintListFilters

// NewSprintRepository creates a new SprintRepository.
var NewSprintRepository = sprint.NewSprintRepository

// --- WorkSession ---

// WorkSessionRepository is an alias for worksession.WorkSessionRepository.
type WorkSessionRepository = worksession.WorkSessionRepository

// SessionStats is an alias for worksession.SessionStats.
type SessionStats = worksession.SessionStats

// SessionAnalytics is an alias for worksession.SessionAnalytics.
type SessionAnalytics = worksession.SessionAnalytics

// NewWorkSessionRepository creates a new WorkSessionRepository.
var NewWorkSessionRepository = worksession.NewWorkSessionRepository

// --- Document ---

// DocumentRepository is an alias for document.DocumentRepository.
type DocumentRepository = document.DocumentRepository

// NewDocumentRepository creates a new DocumentRepository.
var NewDocumentRepository = document.NewDocumentRepository

// --- EntityDocument ---

// EntityDocumentRepository is an alias for entitydoc.EntityDocumentRepository.
type EntityDocumentRepository = entitydoc.EntityDocumentRepository

// NewEntityDocumentRepository creates a new EntityDocumentRepository.
var NewEntityDocumentRepository = entitydoc.NewEntityDocumentRepository

// --- EntityHistory ---

// EntityHistoryRepository is an alias for entityhistory.EntityHistoryRepository.
type EntityHistoryRepository = entityhistory.EntityHistoryRepository

// NewEntityHistoryRepository creates a new EntityHistoryRepository.
var NewEntityHistoryRepository = entityhistory.NewEntityHistoryRepository

// --- EntityRelationship ---

// EntityRelationshipRepository is an alias for entityrel.EntityRelationshipRepository.
type EntityRelationshipRepository = entityrel.EntityRelationshipRepository

// EntityRelTaskKeyAdapter is an alias for entityrel.EntityRelTaskKeyAdapter.
type EntityRelTaskKeyAdapter = entityrel.EntityRelTaskKeyAdapter

// NewEntityRelationshipRepository creates a new EntityRelationshipRepository.
var NewEntityRelationshipRepository = entityrel.NewEntityRelationshipRepository

// NewEntityRelTaskKeyAdapter creates a new EntityRelTaskKeyAdapter.
var NewEntityRelTaskKeyAdapter = entityrel.NewEntityRelTaskKeyAdapter

// EntityRelFeatureKeyAdapter is an alias for entityrel.EntityRelFeatureKeyAdapter.
type EntityRelFeatureKeyAdapter = entityrel.EntityRelFeatureKeyAdapter

// NewEntityRelFeatureKeyAdapter creates a new EntityRelFeatureKeyAdapter.
var NewEntityRelFeatureKeyAdapter = entityrel.NewEntityRelFeatureKeyAdapter

// EntityRelEpicKeyAdapter is an alias for entityrel.EntityRelEpicKeyAdapter.
type EntityRelEpicKeyAdapter = entityrel.EntityRelEpicKeyAdapter

// NewEntityRelEpicKeyAdapter creates a new EntityRelEpicKeyAdapter.
var NewEntityRelEpicKeyAdapter = entityrel.NewEntityRelEpicKeyAdapter

// --- Search ---

// SearchRepository is an alias for search.SearchRepository.
type SearchRepository = search.SearchRepository

// EntitySearchResult is an alias for search.EntitySearchResult.
type EntitySearchResult = search.EntitySearchResult

// NewSearchRepository creates a new SearchRepository.
var NewSearchRepository = search.NewSearchRepository

// --- TemplateEnrichment ---

// TemplateEnrichmentRepository is an alias for templateenrich.TemplateEnrichmentRepository.
type TemplateEnrichmentRepository = templateenrich.TemplateEnrichmentRepository

// NewTemplateEnrichmentRepository creates a new TemplateEnrichmentRepository.
var NewTemplateEnrichmentRepository = templateenrich.NewTemplateEnrichmentRepository

// --- Note ---

// EntityNoteRepository is an alias for note.EntityNoteRepository.
type EntityNoteRepository = note.EntityNoteRepository

// RejectionNoteMetadata is an alias for note.RejectionNoteMetadata.
type RejectionNoteMetadata = note.RejectionNoteMetadata

// RejectionHistoryEntry is an alias for note.RejectionHistoryEntry.
type RejectionHistoryEntry = note.RejectionHistoryEntry

// NewEntityNoteRepository creates a new EntityNoteRepository.
var NewEntityNoteRepository = note.NewEntityNoteRepository

// --- Epic ---

// EpicRepository is an alias for epicpkg.EpicRepository.
type EpicRepository = epicpkg.EpicRepository

// EpicDisplayDataRaw is an alias for epicpkg.EpicDisplayDataRaw.
type EpicDisplayDataRaw = epicpkg.EpicDisplayDataRaw

// FeatureProgressData is an alias for epicpkg.FeatureProgressData.
type FeatureProgressData = epicpkg.FeatureProgressData

// NewEpicRepository creates a new EpicRepository.
var NewEpicRepository = epicpkg.NewEpicRepository

// --- Feature ---

// FeatureRepository is an alias for featurepkg.FeatureRepository.
type FeatureRepository = featurepkg.FeatureRepository

// FeatureDisplayDataRaw is an alias for featurepkg.FeatureDisplayDataRaw.
type FeatureDisplayDataRaw = featurepkg.FeatureDisplayDataRaw

// NewFeatureRepository creates a new FeatureRepository.
var NewFeatureRepository = featurepkg.NewFeatureRepository

// --- Task ---

// NoteCreator is an alias for taskpkg.NoteCreator.
type NoteCreator = taskpkg.NoteCreator

// TaskRepository is an alias for taskpkg.TaskRepository.
type TaskRepository = taskpkg.TaskRepository

// TaskDisplayDataRaw is an alias for taskpkg.TaskDisplayDataRaw.
type TaskDisplayDataRaw = taskpkg.TaskDisplayDataRaw

// HistoryFilters is an alias for taskpkg.HistoryFilters.
// Deprecated: Use EntityHistoryFilters from entityhistory package instead.
type HistoryFilters = taskpkg.HistoryFilters //nolint:staticcheck

// TaskHistoryRepository is an alias for taskpkg.TaskHistoryRepository.
// Deprecated: Use EntityHistoryRepository from entityhistory package instead.
type TaskHistoryRepository = taskpkg.TaskHistoryRepository //nolint:staticcheck

// NewTaskRepository creates a TaskRepository with rejection note support automatically wired.
// This is the canonical public constructor for TaskRepository. It wires note.EntityNoteRepository
// as the NoteCreator so that rejection notes are persisted on forced status updates.
//
// Callers that do not need rejection note support can use NewTaskRepositoryWithNoteCreator(db, nil).
func NewTaskRepository(db *DB) *TaskRepository {
	noteRepo := note.NewEntityNoteRepository(db)
	return taskpkg.NewTaskRepositoryWithNoteCreator(db, noteRepo)
}

// NewTaskRepositoryWithNoteCreator creates a TaskRepository with explicit rejection note support.
var NewTaskRepositoryWithNoteCreator = taskpkg.NewTaskRepositoryWithNoteCreator

// NewTaskRepositoryWithWorkflow creates a TaskRepository (workflow param ignored).
// Deprecated: Use NewTaskRepository instead.
var NewTaskRepositoryWithWorkflow = taskpkg.NewTaskRepositoryWithWorkflow

// NewTaskHistoryRepository creates a new TaskHistoryRepository.
// Deprecated: Use NewEntityHistoryRepository from entityhistory package instead.
var NewTaskHistoryRepository = taskpkg.NewTaskHistoryRepository //nolint:staticcheck

// Compile-time check: note.EntityNoteRepository must satisfy the NoteCreator interface
// from the task sub-package. This guarantees that the wiring in NewTaskRepository above
// is type-safe at compile time.
var _ NoteCreator = (*note.EntityNoteRepository)(nil)
