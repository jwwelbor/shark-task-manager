package repository

// aliases.go provides backward-compatible type and constructor aliases for
// repository sub-packages. All existing callers continue to use the
// repository.XxxRepository and repository.NewXxxRepository names unchanged.
//
// Phase 2 aliases: standalone entity repositories that have been moved into
// dedicated sub-packages (idea, bug, changecard, worksession, document,
// entitydoc, entityhistory, entityrel, search, templateenrich).
//
// Phase 3 aliases: note sub-package (EntityNoteRepository) extracted from root.
// Compile-time check verifies note.EntityNoteRepository satisfies NoteCreator.
//
// Phase 4 aliases: core entity repositories (epic, feature, task) extracted
// from root into dedicated sub-packages.

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
	"github.com/jwwelbor/shark-task-manager/internal/repository/search"
	taskpkg "github.com/jwwelbor/shark-task-manager/internal/repository/task"
	"github.com/jwwelbor/shark-task-manager/internal/repository/templateenrich"
	"github.com/jwwelbor/shark-task-manager/internal/repository/worksession"
)

// --- Idea ---

// IdeaRepository is an alias for idea.IdeaRepository.
type IdeaRepository = idea.IdeaRepository

// IdeaFilter is an alias for idea.IdeaFilter.
type IdeaFilter = idea.IdeaFilter

// NewIdeaRepository creates a new IdeaRepository. Existing callers are unaffected.
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

// NewBugRepository creates a new BugRepository. Existing callers are unaffected.
var NewBugRepository = bug.NewBugRepository

// --- ChangeCard ---

// ChangeCardRepository is an alias for changecard.ChangeCardRepository.
type ChangeCardRepository = changecard.ChangeCardRepository

// ChangeCardRepoFilter is an alias for changecard.ChangeCardRepoFilter.
type ChangeCardRepoFilter = changecard.ChangeCardRepoFilter

// ChangeCardStatusSummary is an alias for changecard.ChangeCardStatusSummary.
type ChangeCardStatusSummary = changecard.ChangeCardStatusSummary

// ChangeCardThroughputStats is an alias for changecard.ChangeCardThroughputStats.
type ChangeCardThroughputStats = changecard.ChangeCardThroughputStats

// NewChangeCardRepository creates a new ChangeCardRepository. Existing callers are unaffected.
var NewChangeCardRepository = changecard.NewChangeCardRepository

// --- WorkSession ---

// WorkSessionRepository is an alias for worksession.WorkSessionRepository.
type WorkSessionRepository = worksession.WorkSessionRepository

// SessionStats is an alias for worksession.SessionStats.
type SessionStats = worksession.SessionStats

// SessionAnalytics is an alias for worksession.SessionAnalytics.
type SessionAnalytics = worksession.SessionAnalytics

// NewWorkSessionRepository creates a new WorkSessionRepository. Existing callers are unaffected.
var NewWorkSessionRepository = worksession.NewWorkSessionRepository

// --- Document ---

// DocumentRepository is an alias for document.DocumentRepository.
type DocumentRepository = document.DocumentRepository

// NewDocumentRepository creates a new DocumentRepository. Existing callers are unaffected.
var NewDocumentRepository = document.NewDocumentRepository

// --- EntityDocument ---

// EntityDocumentRepository is an alias for entitydoc.EntityDocumentRepository.
type EntityDocumentRepository = entitydoc.EntityDocumentRepository

// NewEntityDocumentRepository creates a new EntityDocumentRepository. Existing callers are unaffected.
var NewEntityDocumentRepository = entitydoc.NewEntityDocumentRepository

// --- EntityHistory ---

// EntityHistoryRepository is an alias for entityhistory.EntityHistoryRepository.
type EntityHistoryRepository = entityhistory.EntityHistoryRepository

// NewEntityHistoryRepository creates a new EntityHistoryRepository. Existing callers are unaffected.
var NewEntityHistoryRepository = entityhistory.NewEntityHistoryRepository

// --- EntityRelationship ---

// EntityRelationshipRepository is an alias for entityrel.EntityRelationshipRepository.
type EntityRelationshipRepository = entityrel.EntityRelationshipRepository

// EntityRelTaskKeyAdapter is an alias for entityrel.EntityRelTaskKeyAdapter.
type EntityRelTaskKeyAdapter = entityrel.EntityRelTaskKeyAdapter

// NewEntityRelationshipRepository creates a new EntityRelationshipRepository. Existing callers are unaffected.
var NewEntityRelationshipRepository = entityrel.NewEntityRelationshipRepository

// NewEntityRelTaskKeyAdapter creates a new EntityRelTaskKeyAdapter. Existing callers are unaffected.
var NewEntityRelTaskKeyAdapter = entityrel.NewEntityRelTaskKeyAdapter

// --- Search ---

// SearchRepository is an alias for search.SearchRepository.
type SearchRepository = search.SearchRepository

// SearchResult is an alias for search.SearchResult.
type SearchResult = search.SearchResult

// EntitySearchResult is an alias for search.EntitySearchResult.
type EntitySearchResult = search.EntitySearchResult

// NewSearchRepository creates a new SearchRepository. Existing callers are unaffected.
var NewSearchRepository = search.NewSearchRepository

// --- TemplateEnrichment ---

// TemplateEnrichmentRepository is an alias for templateenrich.TemplateEnrichmentRepository.
type TemplateEnrichmentRepository = templateenrich.TemplateEnrichmentRepository

// NewTemplateEnrichmentRepository creates a new TemplateEnrichmentRepository. Existing callers are unaffected.
var NewTemplateEnrichmentRepository = templateenrich.NewTemplateEnrichmentRepository

// --- Note (Phase 3) ---

// EntityNoteRepository is an alias for note.EntityNoteRepository.
// Existing callers using repository.EntityNoteRepository are unaffected.
type EntityNoteRepository = note.EntityNoteRepository

// RejectionNoteMetadata is an alias for note.RejectionNoteMetadata.
type RejectionNoteMetadata = note.RejectionNoteMetadata

// RejectionHistoryEntry is an alias for note.RejectionHistoryEntry.
type RejectionHistoryEntry = note.RejectionHistoryEntry

// NewEntityNoteRepository creates a new EntityNoteRepository. Existing callers are unaffected.
var NewEntityNoteRepository = note.NewEntityNoteRepository

// --- Epic (Phase 4) ---

// EpicRepository is an alias for epicpkg.EpicRepository.
type EpicRepository = epicpkg.EpicRepository

// EpicDisplayDataRaw is an alias for epicpkg.EpicDisplayDataRaw.
type EpicDisplayDataRaw = epicpkg.EpicDisplayDataRaw

// FeatureProgressData is an alias for epicpkg.FeatureProgressData.
type FeatureProgressData = epicpkg.FeatureProgressData

// EpicRelationshipRepository is an alias for epicpkg.EpicRelationshipRepository.
type EpicRelationshipRepository = epicpkg.EpicRelationshipRepository

// NewEpicRepository creates a new EpicRepository. Existing callers are unaffected.
var NewEpicRepository = epicpkg.NewEpicRepository

// NewEpicRelationshipRepository creates a new EpicRelationshipRepository. Existing callers are unaffected.
var NewEpicRelationshipRepository = epicpkg.NewEpicRelationshipRepository

// --- Feature (Phase 4) ---

// FeatureRepository is an alias for featurepkg.FeatureRepository.
type FeatureRepository = featurepkg.FeatureRepository

// FeatureDisplayDataRaw is an alias for featurepkg.FeatureDisplayDataRaw.
type FeatureDisplayDataRaw = featurepkg.FeatureDisplayDataRaw

// FeatureRelationshipRepository is an alias for featurepkg.FeatureRelationshipRepository.
type FeatureRelationshipRepository = featurepkg.FeatureRelationshipRepository

// NewFeatureRepository creates a new FeatureRepository. Existing callers are unaffected.
var NewFeatureRepository = featurepkg.NewFeatureRepository

// NewFeatureRelationshipRepository creates a new FeatureRelationshipRepository. Existing callers are unaffected.
var NewFeatureRelationshipRepository = featurepkg.NewFeatureRelationshipRepository

// --- Task (Phase 4) ---

// NoteCreator is an alias for taskpkg.NoteCreator.
// Existing callers using repository.NoteCreator are unaffected.
type NoteCreator = taskpkg.NoteCreator

// TaskRepository is an alias for taskpkg.TaskRepository.
type TaskRepository = taskpkg.TaskRepository

// TaskDisplayDataRaw is an alias for taskpkg.TaskDisplayDataRaw.
type TaskDisplayDataRaw = taskpkg.TaskDisplayDataRaw

// TaskRelationshipRepository is an alias for taskpkg.TaskRelationshipRepository.
type TaskRelationshipRepository = taskpkg.TaskRelationshipRepository

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
// Existing callers are unaffected.
var NewTaskRepositoryWithNoteCreator = taskpkg.NewTaskRepositoryWithNoteCreator

// NewTaskRepositoryWithWorkflow creates a TaskRepository (workflow param ignored).
// Deprecated: Use NewTaskRepository instead.
var NewTaskRepositoryWithWorkflow = taskpkg.NewTaskRepositoryWithWorkflow

// NewTaskRelationshipRepository creates a new TaskRelationshipRepository. Existing callers are unaffected.
var NewTaskRelationshipRepository = taskpkg.NewTaskRelationshipRepository

// NewTaskHistoryRepository creates a new TaskHistoryRepository. Existing callers are unaffected.
// Deprecated: Use NewEntityHistoryRepository from entityhistory package instead.
var NewTaskHistoryRepository = taskpkg.NewTaskHistoryRepository //nolint:staticcheck

// Compile-time check: note.EntityNoteRepository must satisfy the NoteCreator interface
// from the task sub-package. This guarantees that the wiring in NewTaskRepository above
// is type-safe at compile time.
var _ NoteCreator = (*note.EntityNoteRepository)(nil)
