// Package services provides the business logic layer for the shark task manager.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/entityhistory"
	questionrepo "github.com/jwwelbor/shark-task-manager/internal/repository/question"
	"github.com/jwwelbor/shark-task-manager/internal/repository/task"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// maxInt is used as a sentinel "infinity" when sorting entities with optional execution_order.
const maxInt = int(^uint(0) >> 1)

// viewerFileSizeLimit is the maximum number of bytes read from an entity file.
// Reads are capped at this value + 1 so that the service can detect oversized
// files without reading them fully into memory.
const viewerFileSizeLimit = 2 * 1024 * 1024 // 2 MiB

// SecurityError is returned by ViewerService.File when a path-traversal attack
// is detected (i.e. the resolved file path lies outside the project root).
type SecurityError struct {
	Path string
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf("path %q is outside the project root (possible traversal attack)", e.Path)
}

// FileTooLargeError is returned by ViewerService.File when an entity file
// exceeds the 2 MiB read limit.
type FileTooLargeError struct {
	Path     string
	LimitMiB int
}

func (e *FileTooLargeError) Error() string {
	return fmt.Sprintf("file %q exceeds the %d MiB read limit", e.Path, e.LimitMiB)
}

// ----- Repository interfaces for ViewerService -----

// ViewerEpicRepository is the minimal epic repository interface used by ViewerService.
type ViewerEpicRepository interface {
	List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
}

// ViewerFeatureRepository is the minimal feature repository interface used by ViewerService.
type ViewerFeatureRepository interface {
	List(ctx context.Context) ([]*models.Feature, error)
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
}

// ViewerTaskRepository is the minimal task repository interface used by ViewerService.
type ViewerTaskRepository interface {
	List(ctx context.Context) ([]*models.Task, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	CountBlocked(ctx context.Context) (int, error)
	// ListByFeatureWithViewerRelationships returns tasks for a feature with pre-resolved relationship JSON.
	// Used by FeatureTasks (the lazy-load endpoint that replaced inline embedding in Hierarchy — see B017).
	ListByFeatureWithViewerRelationships(ctx context.Context, featureID int64) ([]*models.ViewerTaskWithRelationships, error)
	// CountsByFeature returns total and blocked task counts per feature in a single
	// aggregate query. Used by the hierarchy endpoint (B017) so full task rows are
	// never embedded in the hierarchy payload (E27-F02 REQ-F-002 lazy-load contract).
	CountsByFeature(ctx context.Context) (map[int64]task.FeatureTaskCounts, error)
	// FeatureIDsForTaskIDs returns the distinct feature IDs containing any of
	// the given task IDs. Used by the hierarchy tag-filter prune logic (B017):
	// when tasks are not embedded in the hierarchy, this maps tag-matched task
	// IDs back to their parent features so feature-level prune decisions can
	// still honor task-level tag matches.
	FeatureIDsForTaskIDs(ctx context.Context, taskIDs []int64) (map[int64]struct{}, error)
}

// ViewerBugRepository is the minimal bug repository interface used by ViewerService.
type ViewerBugRepository interface {
	CountByStatus(ctx context.Context) (map[string]int, error)
	CountBySeverity(ctx context.Context) (map[string]int, error)
}

// ViewerChangeCardRepository is the minimal change-card repository interface used by ViewerService.
type ViewerChangeCardRepository interface {
	CountByStatus(ctx context.Context) (map[string]int, error)
}

// ViewerEntityHistoryRepository is the entity-history query interface used by ViewerService.
// It exposes both per-entity lookup and the cross-entity recent-activity method
// that is implemented by T-E27-F02-001.
//
// ListRecentAcrossEntities uses the entityhistory package types directly so that the
// concrete *entityhistory.EntityHistoryRepository satisfies this interface without
// any adapter.
type ViewerEntityHistoryRepository interface {
	ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error)
	ListRecentAcrossEntities(ctx context.Context, opts entityhistory.ListRecentAcrossEntitiesOptions) ([]*entityhistory.RecentActivityRow, error)
}

// BulkEntityDoc is a document linked to an entity, returned by ViewerEntityDocRepository.ListAll.
type BulkEntityDoc struct {
	EntityType string
	EntityID   int64
	Title      string
	FilePath   string
}

// ViewerEntityDocRepository is the minimal interface for bulk-loading all entity-linked documents.
// It is optional — ViewerService degrades gracefully if nil.
type ViewerEntityDocRepository interface {
	ListAll(ctx context.Context) ([]*BulkEntityDoc, error)
}

// ViewerEntityNoteRepository is the minimal note repository interface used by ViewerService.Notes.
// It is optional — ViewerService degrades gracefully if nil (Notes returns empty slice).
type ViewerEntityNoteRepository interface {
	GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

// ViewerEntityDocByEntityRepository is the minimal interface for loading documents for a single entity.
// It is optional — ViewerService degrades gracefully if nil (RelatedDocs returns empty slice).
type ViewerEntityDocByEntityRepository interface {
	ListForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.Document, error)
}

// ViewerIdeaRepository is the minimal idea repository interface used by ViewerService.
// It is optional — ViewerService degrades gracefully if nil (ideas section omitted from Summary).
type ViewerIdeaRepository interface {
	ListAll(ctx context.Context) ([]*models.Idea, error)
	GetByKey(ctx context.Context, key string) (*models.Idea, error)
}

// ViewerBugListRepository lists all bugs for the hierarchy sidebar.
// It is optional — ViewerService degrades gracefully if nil (bugs section omitted from Hierarchy).
// Also used by resolveEntityID for bug history lookups.
//
// includeTerminal mirrors the CLI `bug list --all` flag: when false, terminal-status
// bugs (resolved, wont_fix, duplicate) are excluded at the repository layer.
type ViewerBugListRepository interface {
	ListAll(ctx context.Context, includeTerminal bool) ([]*models.Bug, error)
	GetByKey(ctx context.Context, key string) (*models.Bug, error)
}

// ViewerChangeCardListRepository lists all change cards for the hierarchy sidebar.
// It is optional — ViewerService degrades gracefully if nil (change cards section omitted from Hierarchy).
// Also used by resolveFilePath and resolveEntityID for change card lookups.
//
// includeTerminal mirrors the CLI `change list --all` flag: when false, terminal-status
// change cards (completed, declined) are excluded at the repository layer.
type ViewerChangeCardListRepository interface {
	ListAll(ctx context.Context, includeTerminal bool) ([]*models.ChangeCard, error)
	GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
}

// ViewerTechDebtRepository lists tech-debt entities for the hierarchy sidebar and dashboard summary.
// It is optional — ViewerService degrades gracefully if nil (tech-debt sections omitted).
// Also used by resolveFilePath and resolveEntityID for tech-debt lookups.
type ViewerTechDebtRepository interface {
	ListAll(ctx context.Context) ([]*models.TechDebt, error)
	GetByKey(ctx context.Context, key string) (*models.TechDebt, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
}

// ViewerQuestionRepository resolves Question records for the existing generic
// viewer read paths. It deliberately exposes no focused Question query API;
// F01 only extends the established key-based viewer surfaces.
type ViewerQuestionRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Question, error)
	List(ctx context.Context, filter questionrepo.QuestionListFilter) ([]*models.Question, error)
}

// TagReader is the narrow consumer contract that ViewerService needs from TagService.
// *services.TagService satisfies it. Defined here so the viewer service can be tested
// with an in-memory mock without importing the full tag package chain.
// (REQ-F-015, ADR-F06-3)
type TagReader interface {
	ListTags(ctx context.Context) ([]*models.Tag, error)
	EntityIDsByTags(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error)
	AttachedTagNamesByIDs(ctx context.Context, entityType models.EntityType, entityIDs []int64) (map[int64][]string, error)
}

// TagDTO is the narrow projection of models.Tag that the viewer exposes.
// Names are normalized lowercase-ASCII (enforced at F03 add-time).
// (REQ-F-001)
type TagDTO struct {
	Name string `json:"name"`
}

// TagsResponse is the response type for ViewerService.Tags.
// Tags is always a non-nil slice (may be empty) to satisfy AC-01 and ADR-F06-2.
type TagsResponse struct {
	Tags []TagDTO `json:"tags"`
}

// HierarchyOptions carries filter options for ViewerService.Hierarchy.
// Nil/zero values mean "no filter" — identical to pre-F06 behavior.
// (REQ-F-010, ADR-F06-5)
type HierarchyOptions struct {
	Tags            []string // empty → no filter (AC-04 still applies)
	IncludeTerminal bool     // mirror CLI --all: when false, hide terminal-status bugs/change cards at the repo layer
}

// FlatEntity is a lightweight summary of a non-hierarchical entity (bug, change card, tech debt, idea)
// used in the hierarchy sidebar flat sections.
type FlatEntity struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	StatusColor string   `json:"status_color"`
	Tags        []string `json:"tags"`           // NEW (REQ-F-003); always non-nil (ADR-F06-2)
	Size        *int     `json:"size,omitempty"` // E27-F11 (REQ-F-001, REQ-F-002, REQ-F-008)
	dbID        int64    // unexported; used for tag decoration and filter. Not serialized.
}

// ----- Request / Response types -----

// StatusColorInfo carries workflow-derived display metadata for a single status bucket.
type StatusColorInfo struct {
	Status         string  `json:"status"`
	Count          int     `json:"count"`
	Color          string  `json:"color"`
	Phase          string  `json:"phase"`
	ProgressWeight float64 `json:"progress_weight"`
}

// SummaryEntityCounts holds per-status counts for one entity type.
type SummaryEntityCounts struct {
	Total    int               `json:"total"`
	ByStatus []StatusColorInfo `json:"by_status"`
}

// SummaryTaskCounts extends SummaryEntityCounts with a blocked count.
type SummaryTaskCounts struct {
	SummaryEntityCounts
	BlockedCount int `json:"blocked_count"`
}

// SummaryBugCounts extends SummaryEntityCounts with severity counts.
type SummaryBugCounts struct {
	SummaryEntityCounts
	BySeverity map[string]int `json:"by_severity,omitempty"`
}

// SummaryResponse is the response type for ViewerService.Summary.
type SummaryResponse struct {
	Epics       SummaryEntityCounts  `json:"epics"`
	Features    SummaryEntityCounts  `json:"features"`
	Tasks       SummaryTaskCounts    `json:"tasks"`
	Bugs        SummaryBugCounts     `json:"bugs"`
	ChangeCards SummaryEntityCounts  `json:"change_cards"`
	TechDebts   *SummaryEntityCounts `json:"tech_debts,omitempty"`
	Ideas       *SummaryEntityCounts `json:"ideas,omitempty"`
}

// HierarchyDoc is a document linked to an epic or feature.
type HierarchyDoc struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

// HierarchyFeature is a feature with task counts and linked docs.
//
// Per E27-F02 REQ-F-002 / B017, full task data is NOT embedded in the hierarchy
// payload. Clients fetch tasks lazily via `GET /api/v1/viewer/features/{key}/tasks`
// when a feature is expanded. The hierarchy carries only `task_count` and
// `blocked_count` so the sidebar can render summary stats without bulk-loading
// every task on initial page load.
type HierarchyFeature struct {
	*models.Feature
	TaskCount    int             `json:"task_count"`
	BlockedCount int             `json:"blocked_count"`
	StatusColor  string          `json:"status_color"`
	StatusPhase  string          `json:"status_phase"`
	Docs         []*HierarchyDoc `json:"docs"`
	Tags         []string        `json:"tags"` // NEW (REQ-F-003); always non-nil (ADR-F06-2)
}

// HierarchyEpic is an epic with its child features and linked docs embedded.
type HierarchyEpic struct {
	*models.Epic
	Features    []*HierarchyFeature `json:"features"`
	StatusColor string              `json:"status_color"`
	StatusPhase string              `json:"status_phase"`
	Docs        []*HierarchyDoc     `json:"docs"`
	Tags        []string            `json:"tags"` // NEW (REQ-F-003); always non-nil (ADR-F06-2)
}

// HierarchyResponse is the response type for ViewerService.Hierarchy.
type HierarchyResponse struct {
	ProjectName string           `json:"project_name"`
	Epics       []*HierarchyEpic `json:"epics"`
	Questions   []*FlatEntity    `json:"questions,omitempty"`
	Bugs        []*FlatEntity    `json:"bugs,omitempty"`
	ChangeCards []*FlatEntity    `json:"change_cards,omitempty"`
	TechDebts   []*FlatEntity    `json:"tech_debts,omitempty"`
	Ideas       []*FlatEntity    `json:"ideas,omitempty"`
}

// HistoryResponse is the response type for ViewerService.History.
type HistoryResponse struct {
	EntityType models.EntityType       `json:"entity_type"`
	EntityKey  string                  `json:"entity_key"`
	Records    []*models.EntityHistory `json:"records"`
}

// NoteDTO is a single note item returned by ViewerService.Notes.
// Only the six fields documented in REQ-F-020 AC-020.3 are exposed;
// metadata and updated_at are intentionally omitted.
type NoteDTO struct {
	ID        int64  `json:"id"`
	NoteType  string `json:"note_type"`
	Content   string `json:"content"`
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
}

// NotesResponse is the response type for ViewerService.Notes.
// Notes is always a non-nil slice (may be empty) to satisfy AC-020.1.
type NotesResponse struct {
	EntityType models.EntityType `json:"entity_type"`
	EntityKey  string            `json:"entity_key"`
	Notes      []NoteDTO         `json:"notes"`
}

// RelatedDocDTO is a single related document returned by ViewerService.RelatedDocs.
type RelatedDocDTO struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	FilePath string `json:"file_path"`
}

// RelatedDocsResponse is the response type for ViewerService.RelatedDocs.
// Docs is always a non-nil slice (may be empty) to satisfy AC-021.1.
type RelatedDocsResponse struct {
	EntityType models.EntityType `json:"entity_type"`
	EntityKey  string            `json:"entity_key"`
	Docs       []RelatedDocDTO   `json:"docs"`
}

// FileResponse is the response type for ViewerService.File.
type FileResponse struct {
	Exists  bool   `json:"exists"`
	Content string `json:"content,omitempty"`
	Path    string `json:"path,omitempty"` // relative path under project root
}

// FolderFileEntry is one entry returned by ViewerService.FolderFiles.
type FolderFileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"` // relative to project root
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// FolderFilesResponse is the response type for ViewerService.FolderFiles.
type FolderFilesResponse struct {
	DirPath string             `json:"dir_path"` // relative to project root
	Entries []*FolderFileEntry `json:"entries"`
}

// NavFolder is one folder rendered as a top-level dashboard navigation group.
type NavFolder struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Path   string `json:"path"`
	Source string `json:"source"`
	Exists bool   `json:"exists"`
}

// NavFoldersResponse is the ordered navigation-folder contract for the viewer.
// Folders is always non-nil so JSON responses use [] rather than null.
type NavFoldersResponse struct {
	Folders []NavFolder `json:"folders"`
}

// FeatureTaskOptions carries filters and pagination for ViewerService.FeatureTasks.
type FeatureTaskOptions struct {
	Status  string // empty = no filter
	Agent   string // empty = no filter
	Blocked *bool  // nil = no filter
	Limit   int    // 0 = use default (200); >500 = clamped to 500
	Offset  int
	Tags    []string // NEW (REQ-F-008); empty → no tag filter
}

// ViewerTask decorates models.Task with workflow-derived display metadata and
// relationship data for the client-side dependency block (REQ-F-008).
type ViewerTask struct {
	*models.Task
	StatusColor   string                `json:"status_color"`
	StatusPhase   string                `json:"status_phase"`
	Relationships []ViewerRelatedEntity `json:"relationships"` // From entity_relationships (all cross-entity links)
	Tags          []string              `json:"tags"`          // NEW (REQ-F-003); always non-nil (ADR-F06-2)
}

// ViewerRelatedEntity represents a single relationship edge from a task's
// perspective, including the other entity's type and key so cross-entity
// links render correctly in the viewer UI.
type ViewerRelatedEntity struct {
	Direction        string                        `json:"direction"`         // "outgoing" | "incoming"
	RelationshipType models.EntityRelationshipType `json:"relationship_type"` // e.g. "depends_on", "related_to"
	EntityType       models.EntityType             `json:"entity_type"`       // e.g. "task", "bug", "feature"
	EntityKey        string                        `json:"entity_key"`        // resolved key of the related entity
}

// FeatureTasksResponse is the response type for ViewerService.FeatureTasks.
type FeatureTasksResponse struct {
	FeatureKey string        `json:"feature_key"`
	Total      int           `json:"total"` // pre-filter count
	Tasks      []*ViewerTask `json:"tasks"`
}

// RecentActivityOptions carries filter/pagination options for ViewerService.RecentActivity.
type RecentActivityOptions struct {
	// Limit caps the number of rows returned. 0 → default 50; >200 → clamped to 200.
	Limit int
	// Offset is reserved for future client-side pagination (not forwarded to the repository).
	Offset int
	// EntityType optionally restricts results to a single entity type (e.g. "task", "epic").
	// Empty string means all entity types are included.
	EntityType string
	// Since optionally restricts results to activity recorded after this time.
	// nil means no lower-bound time filter.
	Since *time.Time
}

// ActivityRecord is one recent activity entry returned by ViewerService.RecentActivity.
// It wraps entityhistory.RecentActivityRow which already includes the entity key and title
// from the INNER JOIN query executed by the repository.
type ActivityRecord struct {
	EntityType string    `json:"entity_type"`
	Key        string    `json:"key"`
	Title      string    `json:"title"`
	FromStatus string    `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status"`
	ChangedAt  time.Time `json:"changed_at"`
}

// RecentActivityResponse is the response type for ViewerService.RecentActivity.
type RecentActivityResponse struct {
	Records []*ActivityRecord `json:"records"`
}

// WorkflowStatusMeta is metadata for one status in a workflow level.
type WorkflowStatusMeta struct {
	Name           string  `json:"name"`
	Color          string  `json:"color"`
	Phase          string  `json:"phase"`
	ProgressWeight float64 `json:"progress_weight"`
	SprintBucket   *string `json:"sprint_bucket,omitempty"`
}

// WorkflowTransitionMeta describes one valid status transition.
type WorkflowTransitionMeta struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Direction string `json:"direction"` // "forward", "backward", "lateral"
}

// WorkflowLevelMeta holds status and transition metadata for one entity level.
type WorkflowLevelMeta struct {
	Level       string                   `json:"level"`
	Statuses    []WorkflowStatusMeta     `json:"statuses"`
	Transitions []WorkflowTransitionMeta `json:"transitions"`
}

// WorkflowMetaResponse is the response type for ViewerService.WorkflowMeta.
type WorkflowMetaResponse struct {
	Levels map[string]*WorkflowLevelMeta `json:"levels"`
}

// SprintOverviewResponse is the viewer-facing bundle for the Sprint Overview subview.
// It composes the current sprint identity, backlog state, readiness, capacity, and
// optional analytics summary into one read-only payload.
type SprintOverviewResponse struct {
	Sprint    *models.Sprint       `json:"sprint"`
	Backlog   *SprintBacklog       `json:"backlog"`
	Readiness *SprintReadiness     `json:"readiness"`
	Capacity  []CapacityRow        `json:"capacity"`
	Catalog   *SprintCatalog       `json:"catalog,omitempty"`
	Summary   *SprintSummaryResult `json:"summary,omitempty"`
}

// SprintReportResponse is the viewer-facing bundle for the Sprint Report subview.
// It composes burndown, velocity, and summary analytics for the current sprint.
type SprintReportResponse struct {
	Sprint   *models.Sprint       `json:"sprint"`
	Burndown *BurndownResult      `json:"burndown"`
	Velocity *VelocityResult      `json:"velocity"`
	Catalog  *SprintCatalog       `json:"catalog,omitempty"`
	Summary  *SprintSummaryResult `json:"summary"`
}

// SprintCatalog groups sprints for the viewer sidebar tree.
type SprintCatalog struct {
	Active   []*models.Sprint `json:"active"`
	Upcoming []*models.Sprint `json:"upcoming"`
	Archived []*models.Sprint `json:"archived"`
}

// ViewerSprintService is the narrow viewer-facing contract needed to compose sprint
// overview and planning data without importing the full CLI/service stack into tests.
type ViewerSprintService interface {
	ListSprints(ctx context.Context, filters *SprintListFilters) ([]*models.Sprint, error)
	GetSprint(ctx context.Context, key string) (*models.Sprint, error)
	GetSprintBacklog(ctx context.Context, sprintKey string, opts BacklogOptions) (*SprintBacklog, error)
	GetSprintReadiness(ctx context.Context, key string) (*SprintReadiness, error)
	GetSprintCapacity(ctx context.Context, key string) ([]CapacityRow, error)
	PlanSprint(ctx context.Context, key string) (*SprintPlanView, error)
}

// ViewerSprintAnalyticsService is the narrow viewer-facing contract needed to compose
// sprint reporting data without importing the full CLI/service stack into tests.
type ViewerSprintAnalyticsService interface {
	GetBurndown(ctx context.Context, sprintKey string) (*BurndownResult, error)
	GetVelocity(ctx context.Context, n int) (*VelocityResult, error)
	GetSummary(ctx context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error)
}

var (
	_ ViewerSprintService          = (*SprintService)(nil)
	_ ViewerSprintAnalyticsService = (*SprintAnalyticsService)(nil)
)

// ----- ViewerService -----

// ViewerService provides read-only dashboard aggregation for the viewer API.
// It composes existing repositories and the workflow service to expose
// seven dashboard-shaped aggregates.
//
// ViewerService has NO mutation methods (ADR-E27-003).
type ViewerService struct {
	epicRepo           ViewerEpicRepository
	featureRepo        ViewerFeatureRepository
	taskRepo           ViewerTaskRepository
	bugRepo            ViewerBugRepository
	changeCardRepo     ViewerChangeCardRepository
	historyRepo        ViewerEntityHistoryRepository
	entityDocRepo      ViewerEntityDocRepository         // optional; used by Hierarchy for linked docs
	noteRepo           ViewerEntityNoteRepository        // optional; used by Notes endpoint
	docByEntityRepo    ViewerEntityDocByEntityRepository // optional; used by RelatedDocs endpoint
	ideaRepo           ViewerIdeaRepository              // optional; used by Summary and Hierarchy
	bugListRepo        ViewerBugListRepository           // optional; used by Hierarchy for bug flat list
	changeCardListRepo ViewerChangeCardListRepository    // optional; used by Hierarchy for change card flat list
	techDebtRepo       ViewerTechDebtRepository          // optional; used by Summary and Hierarchy for tech-debt flat list
	questionRepo       ViewerQuestionRepository          // optional; used by generic key-based viewer reads
	tagSvc             TagReader                         // optional; used by Tags, Hierarchy, FeatureTasks (REQ-F-015)
	sprintSvc          ViewerSprintService               // optional; used by Sprint Overview and Plan
	sprintAnalyticsSvc ViewerSprintAnalyticsService      // optional; used by Sprint Report
	entityRelSvc       *EntityRelationshipService        // optional; retained for History/RelatedDocs lookups
	entityRegistry     *EntityRegistry                   // optional; retained for History/RelatedDocs lookups
	workflowSvc        *workflow.Service
	statusCalc         *status.CalculationService // optional; reserved for future use
	projectRoot        string
	browsableFolders   []config.BrowsableFolder
}

// NewViewerService constructs a ViewerService.
// All repository and workflow arguments except statusCalc, entityRelSvc, and entityRegistry
// are required and must be non-nil. entityRelSvc and entityRegistry are optional; when
// provided they are used for entity lookups in History and other endpoints. Relationship
// data in Hierarchy/FeatureTasks is now loaded via the viewer_task_relationships SQL view
// and no longer requires entityRelSvc or entityRegistry.
// Panics if any required dependency is nil (matching existing service constructors).
func NewViewerService(
	epicRepo ViewerEpicRepository,
	featureRepo ViewerFeatureRepository,
	taskRepo ViewerTaskRepository,
	bugRepo ViewerBugRepository,
	changeCardRepo ViewerChangeCardRepository,
	historyRepo ViewerEntityHistoryRepository,
	workflowSvc *workflow.Service,
	statusCalc *status.CalculationService,
	projectRoot string,
	entityRelSvc *EntityRelationshipService,
	entityRegistry *EntityRegistry,
) *ViewerService {
	requireNonNil(epicRepo, "ViewerService requires a non-nil EpicRepository")
	requireNonNil(featureRepo, "ViewerService requires a non-nil FeatureRepository")
	requireNonNil(taskRepo, "ViewerService requires a non-nil TaskRepository")
	requireNonNil(bugRepo, "ViewerService requires a non-nil BugRepository")
	requireNonNil(changeCardRepo, "ViewerService requires a non-nil ChangeCardRepository")
	requireNonNil(historyRepo, "ViewerService requires a non-nil EntityHistoryRepository")
	if workflowSvc == nil {
		panic("ViewerService requires a non-nil WorkflowService must not be nil")
	}
	return &ViewerService{
		epicRepo:       epicRepo,
		featureRepo:    featureRepo,
		taskRepo:       taskRepo,
		bugRepo:        bugRepo,
		changeCardRepo: changeCardRepo,
		historyRepo:    historyRepo,
		workflowSvc:    workflowSvc,
		statusCalc:     statusCalc,
		projectRoot:    projectRoot,
		entityRelSvc:   entityRelSvc,
		entityRegistry: entityRegistry,
	}
}

// WithEntityDocRepo wires the optional entity-document repository used by Hierarchy.
// Call after NewViewerService; safe to skip (docs section will be empty).
func (s *ViewerService) WithEntityDocRepo(r ViewerEntityDocRepository) {
	s.entityDocRepo = r
}

// WithNoteRepo wires the optional note repository; safe to skip (Notes returns empty slice when nil).
func (s *ViewerService) WithNoteRepo(r ViewerEntityNoteRepository) {
	s.noteRepo = r
}

// WithDocByEntityRepo wires the optional per-entity document repository; safe to skip (RelatedDocs returns empty slice when nil).
func (s *ViewerService) WithDocByEntityRepo(r ViewerEntityDocByEntityRepository) {
	s.docByEntityRepo = r
}

// WithIdeaRepo wires the optional idea repository used by Summary and Hierarchy.
// Call after NewViewerService; safe to skip (ideas fields omitted when nil).
func (s *ViewerService) WithIdeaRepo(r ViewerIdeaRepository) {
	s.ideaRepo = r
}

// WithBugListRepo wires the optional bug-list repository used by Hierarchy.
// Call after NewViewerService; safe to skip (bugs section omitted when nil).
func (s *ViewerService) WithBugListRepo(r ViewerBugListRepository) {
	s.bugListRepo = r
}

// WithChangeCardListRepo wires the optional change-card-list repository used by Hierarchy.
// Call after NewViewerService; safe to skip (change_cards section omitted when nil).
func (s *ViewerService) WithChangeCardListRepo(r ViewerChangeCardListRepository) {
	s.changeCardListRepo = r
}

// WithTechDebtRepo wires the optional tech-debt repository used by Summary, Hierarchy,
// History, File, and RelatedDocs. Call after NewViewerService; safe to skip.
func (s *ViewerService) WithTechDebtRepo(r ViewerTechDebtRepository) {
	s.techDebtRepo = r
}

// WithQuestionRepo wires the optional typed Question repository into the
// existing History, File, Notes, and RelatedDocs viewer paths.
func (s *ViewerService) WithQuestionRepo(r ViewerQuestionRepository) {
	s.questionRepo = r
}

// WithTagService wires the optional tag reader used by Tags, Hierarchy, and FeatureTasks.
// Call after NewViewerService; safe to skip — the service degrades gracefully:
// Tags() returns {tags: []}, all entity DTOs carry tags: [], tag filters are silently ignored.
// (REQ-F-015, ADR-F06-3)
func (s *ViewerService) WithTagService(r TagReader) *ViewerService {
	s.tagSvc = r
	return s
}

// WithSprintService wires the optional sprint service used by Sprint Overview and Plan.
// Call after NewViewerService; safe to skip if Sprint mode is not exposed.
func (s *ViewerService) WithSprintService(r ViewerSprintService) *ViewerService {
	s.sprintSvc = r
	return s
}

// WithSprintAnalyticsService wires the optional sprint analytics service used by Sprint Report.
// Call after NewViewerService; safe to skip if Sprint mode is not exposed.
func (s *ViewerService) WithSprintAnalyticsService(r ViewerSprintAnalyticsService) *ViewerService {
	s.sprintAnalyticsSvc = r
	return s
}

// WithBrowsableFolders wires optional configured sidebar folders. Path
// validation remains deferred to NavFolders, where it is checked against the
// current project root immediately before exposure.
func (s *ViewerService) WithBrowsableFolders(folders []config.BrowsableFolder) *ViewerService {
	s.browsableFolders = append([]config.BrowsableFolder(nil), folders...)
	return s
}

// Tags returns the full tag vocabulary for the viewer filter UI.
// The method delegates to TagReader.ListTags and reshapes each *models.Tag into a
// narrow TagDTO. Results are sorted alphabetically by name ascending.
//
// When tagSvc is nil (not wired), Tags degrades gracefully and returns {tags: []}
// with no error and no error-level log (REQ-F-015, AC-14, REQ-NF-011).
//
// OTel span: viewer_service.tags with tag.count attribute (REQ-NF-004).
func (s *ViewerService) Tags(ctx context.Context) (*TagsResponse, error) {
	ctx, span := otel.Tracer("shark/services/viewer").Start(ctx, "viewer_service.tags")
	defer span.End()

	if s.tagSvc == nil {
		span.SetAttributes(attribute.Int("tag.count", 0))
		return &TagsResponse{Tags: []TagDTO{}}, nil
	}

	tags, err := s.tagSvc.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer service: list tags: %w", err)
	}

	out := make([]TagDTO, len(tags))
	for i, t := range tags {
		out[i] = TagDTO{Name: t.Name}
	}
	// Sort alphabetically ascending (REQ-F-001, AC-02).
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	span.SetAttributes(attribute.Int("tag.count", len(out)))
	return &TagsResponse{Tags: out}, nil
}

func (s *ViewerService) resolveSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if s.sprintSvc == nil {
		return nil, fmt.Errorf("viewer sprint: sprint service not wired")
	}

	normalized := strings.ToUpper(strings.TrimSpace(key))
	if normalized != "" {
		sprintEntity, err := s.sprintSvc.GetSprint(ctx, normalized)
		if err != nil {
			return nil, fmt.Errorf("viewer sprint: failed to load sprint %q: %w", normalized, err)
		}
		return sprintEntity, nil
	}

	// Prefer an active sprint; fall back to planning if none exists.
	for _, status := range []string{"active", "planning"} {
		sprints, err := s.sprintSvc.ListSprints(ctx, &SprintListFilters{Status: status})
		if err != nil {
			return nil, fmt.Errorf("viewer sprint: failed to list %s sprint: %w", status, err)
		}
		if len(sprints) > 0 {
			return sprints[0], nil
		}
	}
	return nil, fmt.Errorf("sprint not found: no active or planning sprint found")
}

func (s *ViewerService) buildSprintCatalog(ctx context.Context) (*SprintCatalog, error) {
	if s.sprintSvc == nil {
		return nil, fmt.Errorf("viewer sprint: sprint service not wired")
	}

	activeStatuses := []string{"active", "closing"}
	upcomingStatuses := []string{"planning"}
	archivedStatuses := []string{"completed", "cancelled", "archived"}

	catalog := &SprintCatalog{
		Active:   []*models.Sprint{},
		Upcoming: []*models.Sprint{},
		Archived: []*models.Sprint{},
	}

	loadInto := func(statuses []string, target *[]*models.Sprint) error {
		for _, status := range statuses {
			sprints, err := s.sprintSvc.ListSprints(ctx, &SprintListFilters{Status: status})
			if err != nil {
				return fmt.Errorf("viewer sprint: failed to list %s sprints: %w", status, err)
			}
			*target = append(*target, sprints...)
		}
		return nil
	}

	if err := loadInto(activeStatuses, &catalog.Active); err != nil {
		return nil, err
	}
	if err := loadInto(upcomingStatuses, &catalog.Upcoming); err != nil {
		return nil, err
	}
	if err := loadInto(archivedStatuses, &catalog.Archived); err != nil {
		return nil, err
	}

	return catalog, nil
}

// SprintOverview returns the current sprint's operational bundle for the Overview subview.
// It composes the sprint identity, backlog/status buckets, readiness, capacity, and optional
// analytics summary from the existing sprint services.
func (s *ViewerService) SprintOverview(ctx context.Context, key string) (*SprintOverviewResponse, error) {
	sprintEntity, err := s.resolveSprint(ctx, key)
	if err != nil {
		return nil, err
	}
	catalog, err := s.buildSprintCatalog(ctx)
	if err != nil {
		return nil, err
	}

	// View:"grouped" ensures backlog.groups is always populated so the viewer sidebar
	// can aggregate items by status_category into the four display buckets (ready,
	// in_progress, blocked, done) via SPRINT_BUCKET_MAP.  Without this, active sprints
	// default to the "ordered" view which populates backlog.items instead of
	// backlog.groups, leaving all sidebar bucket counts at zero.
	backlog, err := s.sprintSvc.GetSprintBacklog(ctx, sprintEntity.Key, BacklogOptions{View: "grouped"})
	if err != nil {
		return nil, fmt.Errorf("viewer sprint overview: failed to load backlog: %w", err)
	}

	readiness, err := s.sprintSvc.GetSprintReadiness(ctx, sprintEntity.Key)
	if err != nil {
		return nil, fmt.Errorf("viewer sprint overview: failed to load readiness: %w", err)
	}

	capacity, err := s.sprintSvc.GetSprintCapacity(ctx, sprintEntity.Key)
	if err != nil {
		return nil, fmt.Errorf("viewer sprint overview: failed to load capacity: %w", err)
	}

	// Summary is only available for completed/archived sprints; skip it for others.
	var summary *SprintSummaryResult
	if s.sprintAnalyticsSvc != nil {
		st := string(sprintEntity.Status)
		if st == "completed" || st == "archived" {
			summary, err = s.sprintAnalyticsSvc.GetSummary(ctx, sprintEntity.Key, false)
			if err != nil {
				return nil, fmt.Errorf("viewer sprint overview: failed to load summary: %w", err)
			}
		}
	}

	return &SprintOverviewResponse{
		Sprint:    sprintEntity,
		Backlog:   backlog,
		Readiness: readiness,
		Capacity:  capacity,
		Catalog:   catalog,
		Summary:   summary,
	}, nil
}

// SprintPlan returns the planning bundle for the Plan subview.
// It delegates to the existing SprintService.PlanSprint composition.
func (s *ViewerService) SprintPlan(ctx context.Context, key string) (*SprintPlanView, error) {
	sprintEntity, err := s.resolveSprint(ctx, key)
	if err != nil {
		return nil, err
	}
	catalog, err := s.buildSprintCatalog(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.sprintSvc.PlanSprint(ctx, sprintEntity.Key)
	if err != nil {
		return nil, fmt.Errorf("viewer sprint plan: failed to load plan: %w", err)
	}
	view.Catalog = catalog
	return view, nil
}

// SprintReport returns the reporting bundle for the Report subview.
// It composes burndown, velocity, and summary analytics from the sprint analytics service.
func (s *ViewerService) SprintReport(ctx context.Context, key string) (*SprintReportResponse, error) {
	sprintEntity, err := s.resolveSprint(ctx, key)
	if err != nil {
		return nil, err
	}
	catalog, err := s.buildSprintCatalog(ctx)
	if err != nil {
		return nil, err
	}
	if s.sprintAnalyticsSvc == nil {
		return nil, fmt.Errorf("viewer sprint report: sprint analytics service not wired")
	}

	burndown, err := s.sprintAnalyticsSvc.GetBurndown(ctx, sprintEntity.Key)
	if err != nil {
		return nil, fmt.Errorf("viewer sprint report: failed to load burndown: %w", err)
	}

	velocity, err := s.sprintAnalyticsSvc.GetVelocity(ctx, 6)
	if err != nil {
		return nil, fmt.Errorf("viewer sprint report: failed to load velocity: %w", err)
	}

	summary, err := s.sprintAnalyticsSvc.GetSummary(ctx, sprintEntity.Key, false)
	if err != nil {
		if sprintEntity.Status == "completed" || sprintEntity.Status == "archived" {
			return nil, fmt.Errorf("viewer sprint report: failed to load summary: %w", err)
		}
		summary = nil // summary not yet available for in-progress sprints
	}

	return &SprintReportResponse{
		Sprint:   sprintEntity,
		Burndown: burndown,
		Velocity: velocity,
		Catalog:  catalog,
		Summary:  summary,
	}, nil
}

// Summary returns entity-type counts with per-status color/phase metadata
// from the workflow service.
func (s *ViewerService) Summary(ctx context.Context) (*SummaryResponse, error) {
	epicCounts, err := s.epicRepo.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to count epics: %w", err)
	}

	featureCounts, err := s.featureRepo.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to count features: %w", err)
	}

	taskCounts, err := s.taskRepo.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to count tasks: %w", err)
	}

	blockedCount, err := s.taskRepo.CountBlocked(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to count blocked tasks: %w", err)
	}

	bugCounts, err := s.bugRepo.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to count bugs by status: %w", err)
	}

	bugSeverity, err := s.bugRepo.CountBySeverity(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to count bugs by severity: %w", err)
	}

	ccCounts, err := s.changeCardRepo.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to count change cards by status: %w", err)
	}

	var techDebtCounts map[string]int
	if s.techDebtRepo != nil {
		techDebtCounts, err = s.techDebtRepo.CountByStatus(ctx)
		if err != nil {
			return nil, fmt.Errorf("viewer summary: failed to count tech debts by status: %w", err)
		}
	}

	epicSvc := s.workflowSvc.ForLevel(workflow.LevelEpic)
	featureSvc := s.workflowSvc.ForLevel(workflow.LevelFeature)
	taskSvc := s.workflowSvc.ForLevel(workflow.LevelTask)
	bugSvc := s.workflowSvc.ForLevel(workflow.LevelBug)
	ccSvc := s.workflowSvc.ForLevel(workflow.LevelChange)
	techDebtSvc := s.workflowSvc.ForLevel(workflow.LevelTechDebt)

	var ideaCounts *SummaryEntityCounts
	if s.ideaRepo != nil {
		ideas, err := s.ideaRepo.ListAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("viewer summary: failed to list ideas: %w", err)
		}
		raw := make(map[string]int, len(ideas))
		for _, i := range ideas {
			raw[string(i.Status)]++
		}
		ic := enrichIdeaCounts(raw)
		ideaCounts = &ic
	}

	return &SummaryResponse{
		Epics:    enrichEntityCounts(epicCounts, epicSvc),
		Features: enrichEntityCounts(featureCounts, featureSvc),
		Tasks: SummaryTaskCounts{
			SummaryEntityCounts: enrichEntityCounts(taskCounts, taskSvc),
			BlockedCount:        blockedCount,
		},
		Bugs: SummaryBugCounts{
			SummaryEntityCounts: enrichEntityCounts(bugCounts, bugSvc),
			BySeverity:          bugSeverity,
		},
		ChangeCards: enrichEntityCounts(ccCounts, ccSvc),
		TechDebts: func() *SummaryEntityCounts {
			if s.techDebtRepo == nil {
				return nil
			}
			counts := enrichEntityCounts(techDebtCounts, techDebtSvc)
			return &counts
		}(),
		Ideas: ideaCounts,
	}, nil
}

// enrichEntityCounts converts a raw status→count map into a SummaryEntityCounts
// with workflow-derived color, phase, and progress_weight fields.
func enrichEntityCounts(counts map[string]int, svc *workflow.Service) SummaryEntityCounts {
	result := SummaryEntityCounts{}
	ordered := svc.GetAllStatusesOrdered()

	// Build a set of statuses we actually have counts for.
	seen := make(map[string]bool)
	for status := range counts {
		seen[status] = true
	}

	// Emit in workflow order first, then any remaining unknown statuses.
	emit := func(st string, count int) {
		meta := svc.GetStatusMetadata(st)
		color := meta.Color
		phase := meta.Phase
		if color == "" {
			color = "gray"
		}
		if phase == "" {
			phase = "unknown"
		}
		pw := getProgressWeight(svc, st)
		result.ByStatus = append(result.ByStatus, StatusColorInfo{
			Status:         st,
			Count:          count,
			Color:          color,
			Phase:          phase,
			ProgressWeight: pw,
		})
		result.Total += count
	}

	for _, st := range ordered {
		if c, ok := counts[st]; ok && c > 0 {
			emit(st, c)
			seen[st] = false // mark emitted
		}
	}
	// Any statuses returned by the DB that aren't in the workflow config.
	for st, c := range counts {
		if seen[st] && c > 0 {
			emit(st, c)
		}
	}

	if result.ByStatus == nil {
		result.ByStatus = []StatusColorInfo{}
	}

	return result
}

// ideaStatusOrder defines the display order for idea status buckets.
var ideaStatusOrder = []string{"new", "on_hold", "converted", "archived"}

// ideaStatusColors maps idea status names to display hex colors.
// Ideas are not workflow-driven, so colors are static.
var ideaStatusColors = map[string]string{
	"new":       "#60a5fa", // blue
	"on_hold":   "#fbbf24", // amber
	"converted": "#34d399", // green
	"archived":  "#6b7280", // gray
}

// enrichIdeaCounts converts a raw idea status→count map into a SummaryEntityCounts
// using static color assignments (ideas are not workflow-driven).
func enrichIdeaCounts(counts map[string]int) SummaryEntityCounts {
	result := SummaryEntityCounts{}
	seen := make(map[string]bool, len(counts))
	for st := range counts {
		seen[st] = true
	}

	for _, st := range ideaStatusOrder {
		if c, ok := counts[st]; ok && c > 0 {
			color := ideaStatusColors[st]
			if color == "" {
				color = "#6b7280"
			}
			result.ByStatus = append(result.ByStatus, StatusColorInfo{
				Status: st,
				Count:  c,
				Color:  color,
			})
			result.Total += c
			seen[st] = false
		}
	}
	// Any statuses not in the predefined order.
	for st, c := range counts {
		if seen[st] && c > 0 {
			result.ByStatus = append(result.ByStatus, StatusColorInfo{
				Status: st,
				Count:  c,
				Color:  "#6b7280",
			})
			result.Total += c
		}
	}
	if result.ByStatus == nil {
		result.ByStatus = []StatusColorInfo{}
	}
	return result
}

// getProgressWeight retrieves the ProgressWeight for a status from the underlying
// workflow config. Falls back to 0.0 for unknown statuses.
func getProgressWeight(svc *workflow.Service, statusName string) float64 {
	wf := svc.GetWorkflow()
	if wf == nil {
		return 0.0
	}
	meta, ok := wf.GetStatusMetadata(statusName)
	if !ok {
		return 0.0
	}
	return meta.ProgressWeight
}

// parseViewerRelationships parses the relationships_json string produced by the
// viewer_task_relationships SQL view into a []ViewerRelatedEntity. Returns a
// non-nil empty slice on parse error or when jsonStr is empty / "[]".
func parseViewerRelationships(jsonStr string) []ViewerRelatedEntity {
	if jsonStr == "" || jsonStr == "[]" {
		return []ViewerRelatedEntity{}
	}
	var raw []struct {
		Direction        string `json:"direction"`
		RelationshipType string `json:"relationship_type"`
		EntityType       string `json:"entity_type"`
		EntityKey        string `json:"entity_key"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		fmt.Fprintf(os.Stderr, "viewer: failed to parse relationships JSON: %v\n", err)
		return []ViewerRelatedEntity{}
	}
	result := make([]ViewerRelatedEntity, 0, len(raw))
	for _, r := range raw {
		result = append(result, ViewerRelatedEntity{
			Direction:        r.Direction,
			RelationshipType: models.EntityRelationshipType(r.RelationshipType),
			EntityType:       models.EntityType(r.EntityType),
			EntityKey:        r.EntityKey,
		})
	}
	return result
}

// Hierarchy returns epics ordered by execution_order ASC, created_at ASC,
// with features and task/blocked counts embedded.
//
// opts.Tags: when non-empty and tagSvc is wired, the response is pruned so that
// only entities with the AND-intersection of all provided tags survive
// (ADR-F06-4, REQ-F-010).  When tagSvc is nil, opts.Tags is silently ignored
// and every entity DTO carries Tags: []string{} (REQ-F-015, ADR-F06-2).
//
// Tag decoration (REQ-F-004): at most one call to tagSvc.AttachedTagNamesByIDs
// per entity type present (epic, feature, task, question, bug, change, tech debt,
// idea).
func (s *ViewerService) Hierarchy(ctx context.Context, opts HierarchyOptions) (*HierarchyResponse, error) {
	// ── Step 1: Build the unfiltered tree and flat lists ──

	// Bulk-load all three entity types. Tasks include pre-resolved relationship JSON
	// from the viewer_task_relationships view — one query replaces N+1 per-task calls.

	epics, err := s.epicRepo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("viewer hierarchy: failed to list epics: %w", err)
	}

	allFeatures, err := s.featureRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer hierarchy: failed to list features: %w", err)
	}

	// B017 / E27-F02 REQ-F-002: hierarchy is lazy-load — only counts are returned
	// per feature, never full task rows. Tasks are fetched via the
	// /features/{key}/tasks endpoint on demand.
	taskCountsByFeature, err := s.taskRepo.CountsByFeature(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer hierarchy: failed to count tasks by feature: %w", err)
	}

	var techDebts []*models.TechDebt
	if s.techDebtRepo != nil {
		techDebts, err = s.techDebtRepo.ListAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("viewer hierarchy: failed to list tech debts: %w", err)
		}
		sort.SliceStable(techDebts, func(i, j int) bool {
			if !techDebts[i].CreatedAt.Equal(techDebts[j].CreatedAt) {
				return techDebts[i].CreatedAt.After(techDebts[j].CreatedAt)
			}
			if !techDebts[i].UpdatedAt.Equal(techDebts[j].UpdatedAt) {
				return techDebts[i].UpdatedAt.After(techDebts[j].UpdatedAt)
			}
			return techDebts[i].Key < techDebts[j].Key
		})
	}

	// Index features by epic ID.
	featuresByEpic := make(map[int64][]*models.Feature, len(allFeatures))
	for _, f := range allFeatures {
		featuresByEpic[f.EpicID] = append(featuresByEpic[f.EpicID], f)
	}

	// Sort epics by key ASC (epics have no execution_order; key is already ordered E01, E02, …).
	sort.Slice(epics, func(i, j int) bool {
		return epics[i].Key < epics[j].Key
	})

	epicSvc := s.workflowSvc.ForLevel(workflow.LevelEpic)
	featureSvc := s.workflowSvc.ForLevel(workflow.LevelFeature)
	techDebtSvc := s.workflowSvc.ForLevel(workflow.LevelTechDebt)

	// Bulk-load linked docs (optional — skipped when entityDocRepo is nil).
	docsByEpic := make(map[int64][]*HierarchyDoc)
	docsByFeature := make(map[int64][]*HierarchyDoc)
	if s.entityDocRepo != nil {
		bulkDocs, err := s.entityDocRepo.ListAll(ctx)
		if err == nil {
			for _, d := range bulkDocs {
				hd := &HierarchyDoc{Title: d.Title, Path: d.FilePath}
				switch d.EntityType {
				case "epic":
					docsByEpic[d.EntityID] = append(docsByEpic[d.EntityID], hd)
				case "feature":
					docsByFeature[d.EntityID] = append(docsByFeature[d.EntityID], hd)
				}
			}
		}
	}

	// ── Step 2: If tag filter requested, compute ID sets per entity type ──
	// Returns *UnregisteredTagError when a tag name is not in the vocabulary;
	// propagates unchanged to the caller (AC-T4, REQ-F-011).
	var idSets map[models.EntityType]map[int64]struct{}
	if len(opts.Tags) > 0 && s.tagSvc != nil {
		idSets, err = s.computeHierarchyTagIDSets(ctx, opts.Tags)
		if err != nil {
			return nil, err // propagate *UnregisteredTagError or repo error unchanged
		}
	}

	// ── Build result structure ──
	result := &HierarchyResponse{
		ProjectName: projectNameFromRoot(s.projectRoot),
		Epics:       make([]*HierarchyEpic, 0, len(epics)),
	}

	for _, epic := range epics {
		epicMeta := epicSvc.GetStatusMetadata(string(epic.Status))
		he := &HierarchyEpic{
			Epic:        epic,
			Features:    []*HierarchyFeature{},
			StatusColor: colorOrGray(epicMeta.Color),
			StatusPhase: phaseOrUnknown(epicMeta.Phase),
			Docs:        docsByEpic[epic.ID],
			Tags:        []string{}, // ADR-F06-2: always non-nil
		}
		if he.Docs == nil {
			he.Docs = []*HierarchyDoc{}
		}

		features := featuresByEpic[epic.ID]

		// Sort features by execution_order ASC (nil → MaxInt), then created_at ASC.
		sort.Slice(features, func(i, j int) bool {
			oi := maxInt
			if features[i].ExecutionOrder != nil {
				oi = *features[i].ExecutionOrder
			}
			oj := maxInt
			if features[j].ExecutionOrder != nil {
				oj = *features[j].ExecutionOrder
			}
			if oi != oj {
				return oi < oj
			}
			return features[i].CreatedAt.Before(features[j].CreatedAt)
		})

		for _, f := range features {
			// B017 / E27-F02 REQ-F-002: only counts, never full task data, in the
			// hierarchy payload. Frontend fetches tasks lazily via
			// /api/v1/viewer/features/{key}/tasks when a feature is expanded.
			counts := taskCountsByFeature[f.ID]

			fDocs := docsByFeature[f.ID]
			if fDocs == nil {
				fDocs = []*HierarchyDoc{}
			}

			fMeta := featureSvc.GetStatusMetadata(string(f.Status))
			he.Features = append(he.Features, &HierarchyFeature{
				Feature:      f,
				TaskCount:    counts.Total,
				BlockedCount: counts.Blocked,
				StatusColor:  colorOrGray(fMeta.Color),
				StatusPhase:  phaseOrUnknown(fMeta.Phase),
				Docs:         fDocs,
				Tags:         []string{}, // ADR-F06-2: always non-nil
			})
		}

		result.Epics = append(result.Epics, he)
	}

	// ── Flat sections: Questions, bugs, change cards, tech debt, ideas ──
	bugSvc := s.workflowSvc.ForLevel(workflow.LevelBug)
	ccSvc := s.workflowSvc.ForLevel(workflow.LevelChange)
	questionSvc := s.workflowSvc.ForLevel(workflow.LevelQuestion)

	if s.questionRepo != nil {
		questions, err := s.listHierarchyQuestions(ctx)
		if err != nil {
			return nil, fmt.Errorf("viewer hierarchy: failed to list questions: %w", err)
		}
		result.Questions = make([]*FlatEntity, 0, len(questions))
		for _, question := range questions {
			meta := questionSvc.GetStatusMetadata(string(question.Status))
			result.Questions = append(result.Questions, &FlatEntity{
				Key:         question.Key,
				Title:       question.Title,
				Status:      string(question.Status),
				StatusColor: colorOrGray(meta.Color),
				Tags:        []string{},
				Size:        question.Size,
				dbID:        question.ID,
			})
		}
	}

	if s.bugListRepo != nil {
		bugs, err := s.bugListRepo.ListAll(ctx, opts.IncludeTerminal)
		if err == nil {
			result.Bugs = make([]*FlatEntity, 0, len(bugs))
			for _, b := range bugs {
				meta := bugSvc.GetStatusMetadata(string(b.Status))
				result.Bugs = append(result.Bugs, &FlatEntity{
					Key:         b.Key,
					Title:       b.Title,
					Status:      string(b.Status),
					StatusColor: colorOrGray(meta.Color),
					Tags:        []string{}, // ADR-F06-2: always non-nil
					Size:        b.Size,     // E27-F11: sourced from BaseEntity.Size
					dbID:        b.ID,       // tracked for tag decoration / filter
				})
			}
		}
	}

	if s.changeCardListRepo != nil {
		ccs, err := s.changeCardListRepo.ListAll(ctx, opts.IncludeTerminal)
		if err == nil {
			result.ChangeCards = make([]*FlatEntity, 0, len(ccs))
			for _, cc := range ccs {
				meta := ccSvc.GetStatusMetadata(string(cc.Status))
				result.ChangeCards = append(result.ChangeCards, &FlatEntity{
					Key:         cc.Key,
					Title:       cc.Title,
					Status:      string(cc.Status),
					StatusColor: colorOrGray(meta.Color),
					Tags:        []string{}, // ADR-F06-2: always non-nil
					Size:        cc.Size,    // E27-F11: sourced from BaseEntity.Size
					dbID:        cc.ID,      // tracked for tag decoration / filter
				})
			}
		}
	}

	if len(techDebts) > 0 {
		result.TechDebts = make([]*FlatEntity, 0, len(techDebts))
		for _, td := range techDebts {
			meta := techDebtSvc.GetStatusMetadata(string(td.Status))
			result.TechDebts = append(result.TechDebts, &FlatEntity{
				Key:         td.Key,
				Title:       td.Title,
				Status:      string(td.Status),
				StatusColor: colorOrGray(meta.Color),
				Tags:        []string{}, // ADR-F06-2: always non-nil
				Size:        td.Size,    // E27-F11: sourced from BaseEntity.Size
				dbID:        td.ID,      // tracked for tag decoration / filter
			})
		}
	}

	if s.ideaRepo != nil {
		allIdeas, err := s.ideaRepo.ListAll(ctx)
		if err == nil {
			result.Ideas = make([]*FlatEntity, 0, len(allIdeas))
			for _, idea := range allIdeas {
				color, ok := ideaStatusColors[string(idea.Status)]
				if !ok {
					color = "#6b7280"
				}
				result.Ideas = append(result.Ideas, &FlatEntity{
					Key:         idea.Key,
					Title:       idea.Title,
					Status:      string(idea.Status),
					StatusColor: color,
					Tags:        []string{}, // ADR-F06-2: always non-nil
					Size:        idea.Size,  // E27-F11: sourced from Idea.Size (no BaseEntity embed)
					dbID:        idea.ID,    // tracked for tag decoration / filter
				})
			}
		}
	}

	// ── Step 3: Collect entity IDs per type for batch decoration ──
	// ── Step 4: Fetch tag names per type (REQ-F-004: ≤ 7 calls) ──
	if s.tagSvc != nil {
		tagsByEntity := s.fetchTagsForHierarchy(ctx, result)
		// ── Step 5: Walk tree; assign Tags fields (ADR-F06-2) ──
		applyTagsToHierarchy(result, tagsByEntity)
	}

	// ── Step 6: If filter requested, prune (ADR-F06-4) ──
	if idSets != nil {
		// B017 / E27-F02 REQ-F-002: since tasks are no longer in the hierarchy,
		// we resolve task-tag matches back to their parent feature IDs so the
		// prune step can keep features that have a tag-matching task even when
		// the feature itself isn't directly tagged.
		featuresWithTaggedTasks := make(map[int64]struct{})
		if taskMatchSet, ok := idSets[models.EntityTypeTask]; ok && len(taskMatchSet) > 0 {
			matchedTaskIDs := make([]int64, 0, len(taskMatchSet))
			for id := range taskMatchSet {
				matchedTaskIDs = append(matchedTaskIDs, id)
			}
			featureIDs, err := s.taskRepo.FeatureIDsForTaskIDs(ctx, matchedTaskIDs)
			if err != nil {
				return nil, fmt.Errorf("viewer hierarchy: map tagged tasks to features: %w", err)
			}
			featuresWithTaggedTasks = featureIDs
		}
		pruneHierarchy(result, idSets, featuresWithTaggedTasks)
	}

	return result, nil
}

// listHierarchyQuestions reads the finite Q001-Q999 namespace in repository
// pages. The hierarchy is the existing generic Viewer entity source, not a
// focused Question query route; it supplies only the FlatEntity projection.
func (s *ViewerService) listHierarchyQuestions(ctx context.Context) ([]*models.Question, error) {
	const pageSize = 100
	questions := make([]*models.Question, 0)
	for offset := 0; ; offset += pageSize {
		page, err := s.questionRepo.List(ctx, questionrepo.QuestionListFilter{Limit: pageSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		questions = append(questions, page...)
		if len(page) < pageSize {
			return questions, nil
		}
	}
}

// computeHierarchyTagIDSets calls EntityIDsByTags for each entity type in the hierarchy
// using the AND operator. Returns *UnregisteredTagError unchanged on invalid tag names.
// (REQ-F-010, AC-T4, TC-AC07-3)
func (s *ViewerService) computeHierarchyTagIDSets(ctx context.Context, tags []string) (map[models.EntityType]map[int64]struct{}, error) {
	entityTypes := []models.EntityType{
		models.EntityTypeEpic,
		models.EntityTypeFeature,
		models.EntityTypeTask,
		models.EntityTypeQuestion,
		models.EntityTypeBug,
		models.EntityTypeChange,
		models.EntityTypeTechDebt,
		models.EntityTypeIdea,
	}
	result := make(map[models.EntityType]map[int64]struct{}, len(entityTypes))
	for _, et := range entityTypes {
		ids, err := s.tagSvc.EntityIDsByTags(ctx, et, tags, TagQueryOpAnd)
		if err != nil {
			return nil, err // propagate *UnregisteredTagError unchanged
		}
		set := make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			set[id] = struct{}{}
		}
		result[et] = set
	}
	return result, nil
}

// fetchTagsForHierarchy collects entity IDs from the result tree (using the unexported
// dbID field on FlatEntity) and issues at most one batched AttachedTagNamesByIDs call
// one per entity type present in the response. Non-present types are skipped.
// Returns a nested map[EntityType]map[ID][]string for O(1) per-entity lookup.
// (REQ-F-004, AC-16, ADR-F06-1)
func (s *ViewerService) fetchTagsForHierarchy(ctx context.Context, resp *HierarchyResponse) map[models.EntityType]map[int64][]string {
	result := make(map[models.EntityType]map[int64][]string)

	fetch := func(et models.EntityType, ids []int64) {
		if len(ids) == 0 {
			return
		}
		m, err := s.tagSvc.AttachedTagNamesByIDs(ctx, et, ids)
		if err != nil {
			return // best-effort: decoration errors don't fail the endpoint
		}
		result[et] = m
	}

	// Hierarchical entities — IDs come from embedded model pointers.
	// B017 / E27-F02 REQ-F-002: hierarchy no longer carries task data, so task
	// tags are not decorated here; they are fetched alongside task rows by the
	// lazy /features/{key}/tasks endpoint.
	epicIDs := make([]int64, 0, len(resp.Epics))
	var featureIDs []int64
	for _, e := range resp.Epics {
		epicIDs = append(epicIDs, e.Epic.ID)
		for _, f := range e.Features {
			featureIDs = append(featureIDs, f.Feature.ID)
		}
	}
	fetch(models.EntityTypeEpic, epicIDs)
	fetch(models.EntityTypeFeature, featureIDs)

	// Flat entities — IDs come from the unexported dbID field set during construction.
	questionIDs := make([]int64, 0, len(resp.Questions))
	for _, question := range resp.Questions {
		if question.dbID != 0 {
			questionIDs = append(questionIDs, question.dbID)
		}
	}
	fetch(models.EntityTypeQuestion, questionIDs)

	bugIDs := make([]int64, 0, len(resp.Bugs))
	for _, b := range resp.Bugs {
		if b.dbID != 0 {
			bugIDs = append(bugIDs, b.dbID)
		}
	}
	fetch(models.EntityTypeBug, bugIDs)

	ccIDs := make([]int64, 0, len(resp.ChangeCards))
	for _, cc := range resp.ChangeCards {
		if cc.dbID != 0 {
			ccIDs = append(ccIDs, cc.dbID)
		}
	}
	fetch(models.EntityTypeChange, ccIDs)

	tdIDs := make([]int64, 0, len(resp.TechDebts))
	for _, td := range resp.TechDebts {
		if td.dbID != 0 {
			tdIDs = append(tdIDs, td.dbID)
		}
	}
	fetch(models.EntityTypeTechDebt, tdIDs)

	ideaIDs := make([]int64, 0, len(resp.Ideas))
	for _, idea := range resp.Ideas {
		if idea.dbID != 0 {
			ideaIDs = append(ideaIDs, idea.dbID)
		}
	}
	fetch(models.EntityTypeIdea, ideaIDs)

	return result
}

// applyTagsToHierarchy walks the result tree and assigns Tags to every DTO
// from the tagsByEntity lookup map. Always assigns []string{} when no tags
// are present (never nil) — ADR-F06-2.
func applyTagsToHierarchy(resp *HierarchyResponse, tagsByEntity map[models.EntityType]map[int64][]string) {
	epicMap := tagsByEntity[models.EntityTypeEpic]
	featureMap := tagsByEntity[models.EntityTypeFeature]

	for _, e := range resp.Epics {
		if tags, ok := epicMap[e.Epic.ID]; ok {
			e.Tags = tags
		}
		if e.Tags == nil {
			e.Tags = []string{}
		}
		for _, f := range e.Features {
			if tags, ok := featureMap[f.Feature.ID]; ok {
				f.Tags = tags
			}
			if f.Tags == nil {
				f.Tags = []string{}
			}
			// B017 / E27-F02 REQ-F-002: tasks are not embedded in the hierarchy,
			// so task tag decoration happens in the lazy FeatureTasks endpoint.
		}
	}

	// Flat entities: use unexported dbID for lookup.
	questionMap := tagsByEntity[models.EntityTypeQuestion]
	for _, question := range resp.Questions {
		if question.dbID != 0 {
			if tags, ok := questionMap[question.dbID]; ok {
				question.Tags = tags
			}
		}
		if question.Tags == nil {
			question.Tags = []string{}
		}
	}

	bugMap := tagsByEntity[models.EntityTypeBug]
	for _, b := range resp.Bugs {
		if b.dbID != 0 {
			if tags, ok := bugMap[b.dbID]; ok {
				b.Tags = tags
			}
		}
		if b.Tags == nil {
			b.Tags = []string{}
		}
	}
	ccMap := tagsByEntity[models.EntityTypeChange]
	for _, cc := range resp.ChangeCards {
		if cc.dbID != 0 {
			if tags, ok := ccMap[cc.dbID]; ok {
				cc.Tags = tags
			}
		}
		if cc.Tags == nil {
			cc.Tags = []string{}
		}
	}
	tdMap := tagsByEntity[models.EntityTypeTechDebt]
	for _, td := range resp.TechDebts {
		if td.dbID != 0 {
			if tags, ok := tdMap[td.dbID]; ok {
				td.Tags = tags
			}
		}
		if td.Tags == nil {
			td.Tags = []string{}
		}
	}
	ideaMap := tagsByEntity[models.EntityTypeIdea]
	for _, idea := range resp.Ideas {
		if idea.dbID != 0 {
			if tags, ok := ideaMap[idea.dbID]; ok {
				idea.Tags = tags
			}
		}
		if idea.Tags == nil {
			idea.Tags = []string{}
		}
	}
}

// pruneHierarchy removes from resp all entities that are not in idSets.
// Epics are pruned if they have no directly matching tag AND no surviving features.
// Features are pruned if they have no directly matching tag AND no task in the
// feature matches the tag filter. Flat entities are independently filtered using
// their unexported dbID.
//
// B017 / E27-F02 REQ-F-002: since tasks are no longer embedded in the hierarchy,
// task-level prune decisions are surfaced via the precomputed
// featuresWithTaggedTasks set (mapping task-tag matches back to parent
// features). Task task_count/blocked_count are NOT recomputed from the filter
// — they remain the unfiltered counts for the feature, and the lazy
// /features/{key}/tasks endpoint applies the same tag filter when fetching
// task rows.
// (ADR-F06-4, REQ-F-010, TC-AC06-1 through TC-AC06-5)
func pruneHierarchy(resp *HierarchyResponse, idSets map[models.EntityType]map[int64]struct{}, featuresWithTaggedTasks map[int64]struct{}) {
	epicMatchIDs := idSets[models.EntityTypeEpic]
	featureMatchIDs := idSets[models.EntityTypeFeature]

	prunedEpics := make([]*HierarchyEpic, 0, len(resp.Epics))
	for _, e := range resp.Epics {
		// Prune features first.
		prunedFeatures := make([]*HierarchyFeature, 0, len(e.Features))
		for _, f := range e.Features {
			// Feature survives if directly tagged OR has any tag-matching task.
			_, featureDirectly := featureMatchIDs[f.Feature.ID]
			_, hasMatchingTask := featuresWithTaggedTasks[f.Feature.ID]
			if featureDirectly || hasMatchingTask {
				prunedFeatures = append(prunedFeatures, f)
			}
		}
		// Epic survives if directly tagged OR has surviving features.
		_, epicDirectly := epicMatchIDs[e.Epic.ID]
		if epicDirectly || len(prunedFeatures) > 0 {
			e.Features = prunedFeatures
			prunedEpics = append(prunedEpics, e)
		}
	}
	resp.Epics = prunedEpics

	// Prune flat entities independently using dbID.
	questionMatchIDs := idSets[models.EntityTypeQuestion]
	if resp.Questions != nil {
		prunedQuestions := make([]*FlatEntity, 0, len(resp.Questions))
		for _, question := range resp.Questions {
			if question.dbID != 0 {
				if _, ok := questionMatchIDs[question.dbID]; ok {
					prunedQuestions = append(prunedQuestions, question)
				}
			} else {
				prunedQuestions = append(prunedQuestions, question) // no ID — pass through (shouldn't happen)
			}
		}
		resp.Questions = prunedQuestions
	}

	bugMatchIDs := idSets[models.EntityTypeBug]
	if resp.Bugs != nil {
		prunedBugs := make([]*FlatEntity, 0, len(resp.Bugs))
		for _, b := range resp.Bugs {
			if b.dbID != 0 {
				if _, ok := bugMatchIDs[b.dbID]; ok {
					prunedBugs = append(prunedBugs, b)
				}
			} else {
				prunedBugs = append(prunedBugs, b) // no ID — pass through (shouldn't happen)
			}
		}
		resp.Bugs = prunedBugs
	}

	ccMatchIDs := idSets[models.EntityTypeChange]
	if resp.ChangeCards != nil {
		prunedCC := make([]*FlatEntity, 0, len(resp.ChangeCards))
		for _, cc := range resp.ChangeCards {
			if cc.dbID != 0 {
				if _, ok := ccMatchIDs[cc.dbID]; ok {
					prunedCC = append(prunedCC, cc)
				}
			} else {
				prunedCC = append(prunedCC, cc)
			}
		}
		resp.ChangeCards = prunedCC
	}

	tdMatchIDs := idSets[models.EntityTypeTechDebt]
	if resp.TechDebts != nil {
		prunedTechDebts := make([]*FlatEntity, 0, len(resp.TechDebts))
		for _, td := range resp.TechDebts {
			if td.dbID != 0 {
				if _, ok := tdMatchIDs[td.dbID]; ok {
					prunedTechDebts = append(prunedTechDebts, td)
				}
			} else {
				prunedTechDebts = append(prunedTechDebts, td)
			}
		}
		resp.TechDebts = prunedTechDebts
	}

	ideaMatchIDs := idSets[models.EntityTypeIdea]
	if resp.Ideas != nil {
		prunedIdeas := make([]*FlatEntity, 0, len(resp.Ideas))
		for _, idea := range resp.Ideas {
			if idea.dbID != 0 {
				if _, ok := ideaMatchIDs[idea.dbID]; ok {
					prunedIdeas = append(prunedIdeas, idea)
				}
			} else {
				prunedIdeas = append(prunedIdeas, idea)
			}
		}
		resp.Ideas = prunedIdeas
	}
}

// colorOrGray returns the color if non-empty, otherwise "gray".
func colorOrGray(color string) string {
	if color == "" {
		return "gray"
	}
	return color
}

// synthesizeIdeaContent looks up an idea by key and returns a synthesized
// markdown FileResponse built from the idea's stored fields (title, description,
// notes, status, priority). Ideas have no file_path in the database.
func (s *ViewerService) synthesizeIdeaContent(ctx context.Context, key string) (*FileResponse, error) {
	if s.ideaRepo == nil {
		return &FileResponse{Exists: false}, nil
	}
	idea, err := s.ideaRepo.GetByKey(ctx, key)
	if err != nil {
		return &FileResponse{Exists: false}, nil
	}
	return &FileResponse{
		Exists:  true,
		Content: synthesizeIdeaMarkdown(idea),
		Path:    "ideas/" + idea.Key,
	}, nil
}

// synthesizeIdeaMarkdown builds a markdown document from an idea's stored fields.
func synthesizeIdeaMarkdown(idea *models.Idea) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n\n", idea.Title)
	fmt.Fprintf(&sb, "**Key**: `%s`  \n", idea.Key)
	fmt.Fprintf(&sb, "**Status**: %s  \n", idea.Status)
	if idea.Priority != nil {
		fmt.Fprintf(&sb, "**Priority**: %d  \n", *idea.Priority)
	}
	sb.WriteString("\n")
	if idea.Description != nil && *idea.Description != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(*idea.Description)
		sb.WriteString("\n\n")
	}
	if idea.Notes != nil && *idea.Notes != "" {
		sb.WriteString("## Notes\n\n")
		sb.WriteString(*idea.Notes)
		sb.WriteString("\n")
	}
	return sb.String()
}

// phaseOrUnknown returns the phase if non-empty, otherwise "unknown".
func phaseOrUnknown(phase string) string {
	if phase == "" {
		return "unknown"
	}
	return phase
}

// History returns the status-change history for the entity identified by key.
// The entity type is detected from the key format.
// Returns a NotFoundError-wrapped error if the key cannot be resolved to an entity.
func (s *ViewerService) History(ctx context.Context, key string) (*HistoryResponse, error) {
	entityType, err := detectEntityType(key)
	if err != nil {
		return nil, fmt.Errorf("viewer history: %w", err)
	}

	entityID, err := s.resolveEntityID(ctx, entityType, key)
	if err != nil {
		return nil, fmt.Errorf("viewer history: %w", err)
	}

	records, err := s.historyRepo.ListByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("viewer history: failed to list history for %s %s: %w", entityType, key, err)
	}

	return &HistoryResponse{
		EntityType: entityType,
		EntityKey:  key,
		Records:    records,
	}, nil
}

// Notes returns the notes for the entity identified by key.
// The entity type is detected from the key format.
// Returns a NotesResponse with an empty (non-nil) Notes slice when noteRepo is nil
// or the entity has no notes — satisfying AC-020.1 and AC-T2.
func (s *ViewerService) Notes(ctx context.Context, key string) (*NotesResponse, error) {
	entityType, err := detectEntityType(key)
	if err != nil {
		return nil, fmt.Errorf("viewer notes: %w", err)
	}

	normalizedKey := strings.ToUpper(strings.TrimSpace(key))
	out := &NotesResponse{
		EntityType: entityType,
		EntityKey:  normalizedKey,
		Notes:      []NoteDTO{},
	}

	if s.noteRepo == nil {
		return out, nil
	}

	entityID, err := s.resolveEntityID(ctx, entityType, key)
	if err != nil {
		return nil, fmt.Errorf("viewer notes: %w", err)
	}

	raw, err := s.noteRepo.GetByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("viewer notes: failed to load notes for %s %s: %w", entityType, key, err)
	}

	// Repo returns ASC; reverse for DESC order (AC-020.2 / AC-T1).
	out.Notes = make([]NoteDTO, 0, len(raw))
	for i := len(raw) - 1; i >= 0; i-- {
		n := raw[i]
		createdBy := ""
		if n.CreatedBy != nil {
			createdBy = *n.CreatedBy
		}
		out.Notes = append(out.Notes, NoteDTO{
			ID:        n.ID,
			NoteType:  string(n.NoteType),
			Content:   n.Content,
			CreatedBy: createdBy,
			CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return out, nil
}

// RelatedDocs returns the related documents for the entity identified by key.
// The entity type is detected from the key format.
// Returns a RelatedDocsResponse with an empty (non-nil) Docs slice when
// docByEntityRepo is nil or the entity has no docs — satisfying AC-021.1 and AC-T3.
func (s *ViewerService) RelatedDocs(ctx context.Context, key string) (*RelatedDocsResponse, error) {
	entityType, err := detectEntityType(key)
	if err != nil {
		return nil, fmt.Errorf("viewer related-docs: %w", err)
	}

	normalizedKey := strings.ToUpper(strings.TrimSpace(key))
	out := &RelatedDocsResponse{
		EntityType: entityType,
		EntityKey:  normalizedKey,
		Docs:       []RelatedDocDTO{},
	}

	if s.docByEntityRepo == nil {
		return out, nil
	}

	entityID, err := s.resolveEntityID(ctx, entityType, key)
	if err != nil {
		return nil, fmt.Errorf("viewer related-docs: %w", err)
	}

	raw, err := s.docByEntityRepo.ListForEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("viewer related-docs: failed to load docs for %s %s: %w", entityType, key, err)
	}

	out.Docs = make([]RelatedDocDTO, 0, len(raw))
	for _, d := range raw {
		out.Docs = append(out.Docs, RelatedDocDTO{
			ID:       d.ID,
			Title:    d.Title,
			FilePath: d.FilePath,
		})
	}

	return out, nil
}

// detectEntityType infers the entity type from the key format using the keys package
// validation helpers. The check order ensures short task keys (E##-F##-###) are
// tested before feature keys (E##-F##), which is a strict prefix.
func detectEntityType(key string) (models.EntityType, error) {
	upper := strings.ToUpper(strings.TrimSpace(key))
	switch {
	case keys.IsShortTaskKey(upper) || keys.IsTaskKey(upper):
		return models.EntityTypeTask, nil
	case keys.IsFeatureKey(upper):
		return models.EntityTypeFeature, nil
	case keys.IsEpicKey(upper):
		return models.EntityTypeEpic, nil
	case keys.IsBugKey(upper):
		return models.EntityTypeBug, nil
	case keys.IsChangeCardKey(upper):
		return models.EntityTypeChange, nil
	case keys.IsTechDebtKey(upper):
		return models.EntityTypeTechDebt, nil
	case keys.IsQuestionKey(upper):
		return models.EntityTypeQuestion, nil
	}
	return "", fmt.Errorf("unrecognized entity key format: %q", key)
}

// resolveEntityID looks up the entity by key and returns its database ID.
func (s *ViewerService) resolveEntityID(ctx context.Context, entityType models.EntityType, key string) (int64, error) {
	switch entityType {
	case models.EntityTypeEpic:
		epics, err := s.epicRepo.List(ctx, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to look up epic %q: %w", key, err)
		}
		upper := strings.ToUpper(key)
		for _, e := range epics {
			if strings.ToUpper(e.Key) == upper {
				return e.ID, nil
			}
		}
		return 0, fmt.Errorf("epic not found: %q", key)

	case models.EntityTypeFeature:
		f, err := s.featureRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("failed to look up feature %q: %w", key, err)
		}
		return f.ID, nil

	case models.EntityTypeTask:
		t, err := s.taskRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("failed to look up task %q: %w", key, err)
		}
		return t.ID, nil

	case models.EntityTypeBug:
		if s.bugListRepo == nil {
			return 0, fmt.Errorf("bug history lookup not available")
		}
		b, err := s.bugListRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("failed to look up bug %q: %w", key, err)
		}
		return b.ID, nil

	case models.EntityTypeChange:
		if s.changeCardListRepo == nil {
			return 0, fmt.Errorf("change card history lookup not available")
		}
		cc, err := s.changeCardListRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("failed to look up change card %q: %w", key, err)
		}
		return cc.ID, nil

	case models.EntityTypeTechDebt:
		if s.techDebtRepo == nil {
			return 0, fmt.Errorf("tech debt history lookup not available")
		}
		td, err := s.techDebtRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("failed to look up tech debt %q: %w", key, err)
		}
		if td == nil {
			return 0, fmt.Errorf("tech debt %q not found", key)
		}
		return td.ID, nil

	case models.EntityTypeQuestion:
		if s.questionRepo == nil {
			return 0, fmt.Errorf("question history lookup not available")
		}
		question, err := s.questionRepo.GetByKey(ctx, key)
		if err != nil {
			return 0, fmt.Errorf("failed to look up question %q: %w", key, err)
		}
		if question == nil {
			return 0, fmt.Errorf("question not found: %q", key)
		}
		return question.ID, nil

	default:
		return 0, fmt.Errorf("unsupported entity type for history lookup: %q", entityType)
	}
}

// File reads an entity's markdown file, guarding against path traversal with
// filepath.EvalSymlinks and enforcing a 2 MiB size limit.
//
// Returns:
//   - FileResponse{Exists:false} (no error) when the file is absent or the entity has no file_path.
//   - SecurityError when the resolved path lies outside the project root.
//   - FileTooLargeError when the file exceeds 2 MiB.
//   - FileResponse{Exists:true, Content:..., Path:relPath} on success.
func (s *ViewerService) File(ctx context.Context, key string) (*FileResponse, error) {
	// Ideas have no file_path; synthesize markdown from stored fields.
	if keys.IsIdeaKey(strings.ToUpper(strings.TrimSpace(key))) {
		return s.synthesizeIdeaContent(ctx, key)
	}

	entityType, err := detectEntityType(key)
	if err != nil {
		return nil, fmt.Errorf("viewer file: %w", err)
	}

	filePath, err := s.resolveFilePath(ctx, entityType, key)
	if err != nil {
		return nil, fmt.Errorf("viewer file: %w", err)
	}
	if filePath == "" {
		return &FileResponse{Exists: false}, nil
	}

	// Resolve project root to an absolute path so that EvalSymlinks produces
	// absolute canonical paths (necessary when projectRoot is "." or similar).
	absRoot, err := filepath.Abs(s.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("viewer file: failed to resolve project root: %w", err)
	}

	// Make the stored path absolute relative to project root.
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(absRoot, filePath)
	}

	// Canonicalize project root.
	rootCanon, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Errorf("viewer file: failed to canonicalize project root: %w", err)
	}

	// Canonicalize the target path.
	targetCanon, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileResponse{Exists: false}, nil
		}
		return nil, fmt.Errorf("viewer file: failed to canonicalize file path: %w", err)
	}

	if !isContained(rootCanon, targetCanon) {
		return nil, &SecurityError{Path: targetCanon}
	}

	// Open and read with size limit.
	f, err := os.Open(targetCanon)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileResponse{Exists: false}, nil
		}
		return nil, fmt.Errorf("viewer file: failed to open %q: %w", targetCanon, err)
	}
	defer f.Close()

	limited := io.LimitReader(f, int64(viewerFileSizeLimit)+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("viewer file: failed to read %q: %w", targetCanon, err)
	}

	if len(raw) > viewerFileSizeLimit {
		return nil, &FileTooLargeError{Path: targetCanon, LimitMiB: 2}
	}

	relPath, err := filepath.Rel(rootCanon, targetCanon)
	if err != nil {
		relPath = filePath
	}

	return &FileResponse{
		Exists:  true,
		Content: string(raw),
		Path:    relPath,
	}, nil
}

// FileByPath reads an arbitrary file within the project root by its relative path.
// This is used for linked documents (related-docs) that are not entity spec files.
//
// Returns:
//   - FileResponse{Exists:false} (no error) when the file does not exist.
//   - SecurityError when the resolved path lies outside the project root.
//   - FileTooLargeError when the file exceeds 2 MiB.
//   - FileResponse{Exists:true, Content:..., Path:relPath} on success.
func (s *ViewerService) FileByPath(ctx context.Context, filePath string) (*FileResponse, error) {
	// Reject absolute paths and obvious traversal.
	if filepath.IsAbs(filePath) {
		return nil, &SecurityError{Path: filePath}
	}

	// Resolve project root to an absolute path so that EvalSymlinks produces
	// absolute canonical paths (necessary when projectRoot is "." or similar).
	absRoot, err := filepath.Abs(s.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("viewer file by path: failed to resolve project root: %w", err)
	}

	absPath := filepath.Join(absRoot, filePath)

	// Canonicalize project root.
	rootCanon, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Errorf("viewer file by path: failed to canonicalize project root: %w", err)
	}

	// Canonicalize the target path.
	targetCanon, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileResponse{Exists: false}, nil
		}
		return nil, fmt.Errorf("viewer file by path: failed to canonicalize file path: %w", err)
	}

	if !isContained(rootCanon, targetCanon) {
		return nil, &SecurityError{Path: targetCanon}
	}

	// Open and read with size limit.
	f, err := os.Open(targetCanon)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileResponse{Exists: false}, nil
		}
		return nil, fmt.Errorf("viewer file by path: failed to open %q: %w", targetCanon, err)
	}
	defer f.Close()

	limited := io.LimitReader(f, int64(viewerFileSizeLimit)+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("viewer file by path: failed to read %q: %w", targetCanon, err)
	}

	if len(raw) > viewerFileSizeLimit {
		return nil, &FileTooLargeError{Path: targetCanon, LimitMiB: 2}
	}

	relPath, err := filepath.Rel(rootCanon, targetCanon)
	if err != nil {
		relPath = filePath
	}

	return &FileResponse{
		Exists:  true,
		Content: string(raw),
		Path:    relPath,
	}, nil
}

// resolveFilePath retrieves the file_path stored on the entity.
// Returns empty string if the entity has no associated file.
func (s *ViewerService) resolveFilePath(ctx context.Context, entityType models.EntityType, key string) (string, error) {
	switch entityType {
	case models.EntityTypeEpic:
		epics, err := s.epicRepo.List(ctx, nil)
		if err != nil {
			return "", fmt.Errorf("failed to look up epic %q: %w", key, err)
		}
		upper := strings.ToUpper(key)
		for _, e := range epics {
			if strings.ToUpper(e.Key) == upper {
				if e.FilePath != nil {
					return *e.FilePath, nil
				}
				return "", nil
			}
		}
		return "", fmt.Errorf("epic not found: %q", key)

	case models.EntityTypeFeature:
		f, err := s.featureRepo.GetByKey(ctx, key)
		if err != nil {
			return "", fmt.Errorf("failed to look up feature %q: %w", key, err)
		}
		if f.FilePath != nil {
			return *f.FilePath, nil
		}
		return "", nil

	case models.EntityTypeTask:
		t, err := s.taskRepo.GetByKey(ctx, key)
		if err != nil {
			return "", fmt.Errorf("failed to look up task %q: %w", key, err)
		}
		if t.FilePath != nil {
			return *t.FilePath, nil
		}
		return "", nil

	case models.EntityTypeBug:
		if s.bugListRepo == nil {
			return "", nil
		}
		b, err := s.bugListRepo.GetByKey(ctx, key)
		if err != nil {
			return "", fmt.Errorf("failed to look up bug %q: %w", key, err)
		}
		if b == nil {
			return "", nil
		}
		if b.FilePath != nil {
			return *b.FilePath, nil
		}
		return "", nil

	case models.EntityTypeChange:
		if s.changeCardListRepo == nil {
			return "", nil
		}
		cc, err := s.changeCardListRepo.GetByKey(ctx, key)
		if err != nil {
			return "", fmt.Errorf("failed to look up change card %q: %w", key, err)
		}
		if cc.FilePath != nil {
			return *cc.FilePath, nil
		}
		return "", nil

	case models.EntityTypeTechDebt:
		if s.techDebtRepo == nil {
			return "", nil
		}
		td, err := s.techDebtRepo.GetByKey(ctx, key)
		if err != nil {
			return "", fmt.Errorf("failed to look up tech debt %q: %w", key, err)
		}
		if td == nil {
			return "", nil
		}
		if td.FilePath != nil {
			return *td.FilePath, nil
		}
		return "", nil

	case models.EntityTypeQuestion:
		if s.questionRepo == nil {
			return "", nil
		}
		question, err := s.questionRepo.GetByKey(ctx, key)
		if err != nil {
			return "", fmt.Errorf("failed to look up question %q: %w", key, err)
		}
		if question != nil && question.FilePath != nil {
			return *question.FilePath, nil
		}
		return "", nil

	default:
		return "", fmt.Errorf("file read not supported for entity type %q", entityType)
	}
}

// FeatureTasks returns tasks for a feature with status/agent/blocked/tag filtering
// and limit/offset pagination.
//
// Tag filter (REQ-F-008): when opts.Tags is non-empty and tagSvc is wired, the task
// list is intersected with the tag-matched ID set BEFORE the other filters and
// pagination. Total reflects the post-tag-filter count.
//
// Tag decoration (REQ-F-005): one AttachedTagNamesByIDs call after pagination covers
// only the IDs in the returned page.  When tagSvc is nil, opts.Tags is silently
// ignored and every ViewerTask carries Tags: []string{} (REQ-F-015, ADR-F06-2).
func (s *ViewerService) FeatureTasks(ctx context.Context, featureKey string, opts FeatureTaskOptions) (*FeatureTasksResponse, error) {
	// Clamp limit: 0 or negative → 200; >500 → 500.
	limit := opts.Limit
	if limit <= 0 {
		limit = 200
	} else if limit > 500 {
		limit = 500
	}

	// Verify feature exists.
	f, err := s.featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("viewer feature tasks: feature %q not found: %w", featureKey, err)
	}

	tasksWithRels, err := s.taskRepo.ListByFeatureWithViewerRelationships(ctx, f.ID)
	if err != nil {
		return nil, fmt.Errorf("viewer feature tasks: failed to list tasks for feature %s: %w", featureKey, err)
	}

	// Sort: execution_order ASC (nil → MaxInt), priority DESC, created_at ASC.
	sort.Slice(tasksWithRels, func(i, j int) bool {
		oi := maxInt
		if tasksWithRels[i].ExecutionOrder != nil {
			oi = *tasksWithRels[i].ExecutionOrder
		}
		oj := maxInt
		if tasksWithRels[j].ExecutionOrder != nil {
			oj = *tasksWithRels[j].ExecutionOrder
		}
		if oi != oj {
			return oi < oj
		}
		pi := tasksWithRels[i].Priority
		pj := tasksWithRels[j].Priority
		if pi != pj {
			return pi > pj
		}
		return tasksWithRels[i].CreatedAt.Before(tasksWithRels[j].CreatedAt)
	})

	// ── Tag filter (REQ-F-008): applied BEFORE other filters and pagination ──
	// When tagSvc is nil, opts.Tags is silently ignored (REQ-F-015).
	if len(opts.Tags) > 0 && s.tagSvc != nil {
		taggedIDs, err := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeTask, opts.Tags, TagQueryOpAnd)
		if err != nil {
			return nil, err // propagate *UnregisteredTagError unchanged
		}
		idSet := make(map[int64]struct{}, len(taggedIDs))
		for _, id := range taggedIDs {
			idSet[id] = struct{}{}
		}
		tagFiltered := make([]*models.ViewerTaskWithRelationships, 0, len(tasksWithRels))
		for _, t := range tasksWithRels {
			if _, ok := idSet[t.Task.ID]; ok {
				tagFiltered = append(tagFiltered, t)
			}
		}
		tasksWithRels = tagFiltered
	}

	total := len(tasksWithRels)

	// Apply status/agent/blocked filters.
	filtered := make([]*models.ViewerTaskWithRelationships, 0, len(tasksWithRels))
	for _, t := range tasksWithRels {
		if opts.Status != "" && !strings.EqualFold(string(t.Status), opts.Status) {
			continue
		}
		if opts.Agent != "" {
			if t.AgentType == nil || !strings.EqualFold(*t.AgentType, opts.Agent) {
				continue
			}
		}
		if opts.Blocked != nil {
			isBlocked := t.BlockedReason != nil
			if *opts.Blocked != isBlocked {
				continue
			}
		}
		filtered = append(filtered, t)
	}

	// Pagination.
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(filtered) {
		filtered = nil
	} else {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[offset:end]
	}

	// Build ViewerTask slice; relationship data is pre-resolved from the
	// viewer_task_relationships SQL view — zero additional DB calls per task.
	taskSvc := s.workflowSvc.ForLevel(workflow.LevelTask)
	viewerTasks := make([]*ViewerTask, 0, len(filtered))
	for _, t := range filtered {
		meta := taskSvc.GetStatusMetadata(string(t.Status))
		rels := parseViewerRelationships(t.RelationshipsJSON)
		viewerTasks = append(viewerTasks, &ViewerTask{
			Task:          t.Task,
			StatusColor:   colorOrGray(meta.Color),
			StatusPhase:   phaseOrUnknown(meta.Phase),
			Relationships: rels,
			Tags:          []string{}, // ADR-F06-2: always non-nil; will be decorated below
		})
	}

	// ── Post-pagination decoration (REQ-F-005): one call for the page IDs only ──
	if s.tagSvc != nil && len(viewerTasks) > 0 {
		pageIDs := make([]int64, 0, len(viewerTasks))
		for _, t := range viewerTasks {
			pageIDs = append(pageIDs, t.Task.ID)
		}
		tagMap, err := s.tagSvc.AttachedTagNamesByIDs(ctx, models.EntityTypeTask, pageIDs)
		if err == nil {
			for _, t := range viewerTasks {
				if tags, ok := tagMap[t.Task.ID]; ok {
					t.Tags = tags
				}
				if t.Tags == nil {
					t.Tags = []string{}
				}
			}
		}
		// On decoration error, Tags fields remain []string{} (best-effort)
	}

	return &FeatureTasksResponse{
		FeatureKey: featureKey,
		Total:      total,
		Tasks:      viewerTasks,
	}, nil
}

// RecentActivity returns the most recent status transitions across all entity types.
// Limit is clamped to [1, 200]; zero defaults to 50.
func (s *ViewerService) RecentActivity(ctx context.Context, opts RecentActivityOptions) (*RecentActivityResponse, error) {
	// Clamp limit: 0 → 50; >200 → 200.
	if opts.Limit <= 0 {
		opts.Limit = 50
	} else if opts.Limit > 200 {
		opts.Limit = 200
	}

	// Map service DTO → repository options.
	repoOpts := entityhistory.ListRecentAcrossEntitiesOptions{
		Limit:      opts.Limit,
		EntityType: opts.EntityType,
		Since:      opts.Since,
	}

	rows, err := s.historyRepo.ListRecentAcrossEntities(ctx, repoOpts)
	if err != nil {
		return nil, fmt.Errorf("viewer recent activity: %w", err)
	}

	// Map repository rows → ActivityRecord response type.
	result := make([]*ActivityRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, &ActivityRecord{
			EntityType: row.EntityType,
			Key:        row.Key,
			Title:      row.Title,
			FromStatus: row.FromStatus,
			ToStatus:   row.ToStatus,
			ChangedAt:  row.ChangedAt,
		})
	}

	return &RecentActivityResponse{Records: result}, nil
}

// WorkflowMeta returns the status and transition metadata for all entity levels.
func (s *ViewerService) WorkflowMeta(_ context.Context) (*WorkflowMetaResponse, error) {
	levels := []string{
		workflow.LevelEpic,
		workflow.LevelFeature,
		workflow.LevelTask,
		workflow.LevelBug,
		workflow.LevelChange,
		workflow.LevelTechDebt,
	}

	response := &WorkflowMetaResponse{
		Levels: make(map[string]*WorkflowLevelMeta, len(levels)),
	}

	for _, levelName := range levels {
		svc := s.workflowSvc.ForLevel(levelName)
		statuses := svc.GetAllStatusesOrdered()

		// Build ordinal map for direction computation.
		ordinal := make(map[string]int, len(statuses))
		for i, st := range statuses {
			ordinal[st] = i
		}

		statusMetas := make([]WorkflowStatusMeta, 0, len(statuses))
		for _, st := range statuses {
			meta := svc.GetStatusMetadata(st)
			pw := getProgressWeight(svc, st)
			statusMetas = append(statusMetas, WorkflowStatusMeta{
				Name:           st,
				Color:          colorOrGray(meta.Color),
				Phase:          phaseOrUnknown(meta.Phase),
				ProgressWeight: pw,
				SprintBucket:   meta.SprintBucket,
			})
		}

		transitions := make([]WorkflowTransitionMeta, 0)
		for _, from := range statuses {
			targets := svc.GetValidTransitions(from)
			for _, to := range targets {
				fromOrd, fromKnown := ordinal[from]
				toOrd, toKnown := ordinal[to]
				direction := "lateral"
				if fromKnown && toKnown {
					switch {
					case toOrd > fromOrd:
						direction = "forward"
					case toOrd < fromOrd:
						direction = "backward"
					}
				}
				transitions = append(transitions, WorkflowTransitionMeta{
					From:      from,
					To:        to,
					Direction: direction,
				})
			}
		}

		response.Levels[levelName] = &WorkflowLevelMeta{
			Level:       levelName,
			Statuses:    statusMetas,
			Transitions: transitions,
		}
	}

	return response, nil
}

// projectNameFromRoot returns the folder name from projectRoot.
// Handles ".", "..", "/", and empty string gracefully.
func projectNameFromRoot(projectRoot string) string {
	abs, err := filepath.Abs(projectRoot)
	if err != nil || abs == "" {
		return "Project"
	}
	name := filepath.Base(abs)
	if name == "." || name == "/" || name == "" {
		return "Project"
	}
	return name
}

// canonicalProjectRoot holds the absolute and EvalSymlinks-resolved forms of
// the project root used by viewer folder operations.
type canonicalProjectRoot struct {
	absPath   string
	canonical string
}

// resolvedProjectPath is a project-relative path after canonical resolution.
// Exists is false when the requested target is unavailable, including through
// a dangling symlink.
type resolvedProjectPath struct {
	canonical string
	exists    bool
}

// canonicalProjectRoot resolves the service project root once for a folder
// operation. Callers must use isContained before exposing or reading a target.
func (s *ViewerService) canonicalProjectRoot(operation string) (canonicalProjectRoot, error) {
	absPath, err := filepath.Abs(s.projectRoot)
	if err != nil {
		return canonicalProjectRoot{}, fmt.Errorf("%s: failed to resolve project root: %w", operation, err)
	}
	canonical, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return canonicalProjectRoot{}, fmt.Errorf("%s: failed to canonicalize project root: %w", operation, err)
	}
	return canonicalProjectRoot{absPath: absPath, canonical: canonical}, nil
}

// resolveRelativePath resolves a project-relative target. Missing targets are
// resolved through their deepest existing ancestor so callers can still apply
// containment checks before treating them as unavailable.
func (r canonicalProjectRoot) resolveRelativePath(relativePath string) (resolvedProjectPath, error) {
	absPath := filepath.Join(r.absPath, filepath.Clean(relativePath))
	canonical, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		return resolvedProjectPath{canonical: canonical, exists: true}, nil
	}
	if !os.IsNotExist(err) {
		return resolvedProjectPath{}, fmt.Errorf("canonicalize target: %w", err)
	}
	return r.resolveMissingPath(absPath)
}

// resolveMissingPath rebuilds a missing target under its canonical existing
// ancestor. This preserves symlink resolution for missing descendants.
func (r canonicalProjectRoot) resolveMissingPath(absPath string) (resolvedProjectPath, error) {
	ancestor, err := deepestExistingAncestor(absPath)
	if err != nil {
		return resolvedProjectPath{}, err
	}
	ancestorCanonical, err := filepath.EvalSymlinks(ancestor)
	if os.IsNotExist(err) {
		// Lstat identifies a dangling symlink as an existing ancestor, while
		// EvalSymlinks correctly reports that its target is unavailable. Treat
		// both the link and any descendant as unavailable so folder browsing
		// retains its stable empty-result contract. A resolvable symlink still
		// reaches the containment check below its caller and therefore cannot
		// escape the project root. Rebase the unavailable placeholder under the
		// canonical root so it shares the namespace used by containment checks
		// when the configured project root is itself a symlink.
		relative, relErr := filepath.Rel(r.absPath, absPath)
		if relErr != nil {
			return resolvedProjectPath{}, fmt.Errorf("resolve unavailable target relative to project root: %w", relErr)
		}
		return resolvedProjectPath{canonical: filepath.Join(r.canonical, relative)}, nil
	}
	if err != nil {
		return resolvedProjectPath{}, fmt.Errorf("canonicalize existing ancestor: %w", err)
	}
	remainder, err := filepath.Rel(ancestor, absPath)
	if err != nil {
		return resolvedProjectPath{}, fmt.Errorf("resolve missing target: %w", err)
	}
	return resolvedProjectPath{canonical: filepath.Join(ancestorCanonical, remainder)}, nil
}

// deepestExistingAncestor returns the nearest existing path at or above path.
func deepestExistingAncestor(path string) (string, error) {
	for {
		_, err := os.Lstat(path)
		if err == nil {
			return path, nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect path %q: %w", path, err)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("could not find existing ancestor for %q", path)
		}
		path = parent
	}
}

// navFolder describes one candidate response item before path validation.
type navFolder struct {
	id, label, path, source string
}

// buildNavFolder validates a folder candidate and returns an empty rejection
// reason when it is safe to expose. Containment remains single-sourced in
// isContained.
func buildNavFolder(root canonicalProjectRoot, candidate navFolder) (NavFolder, string, error) {
	path := strings.TrimSpace(candidate.path)
	if path == "" {
		return NavFolder{}, "path is empty", nil
	}
	if filepath.IsAbs(path) {
		return NavFolder{}, "path is absolute", nil
	}
	cleanPath := filepath.Clean(path)
	resolved, err := root.resolveRelativePath(cleanPath)
	if err != nil {
		return NavFolder{}, "", err
	}
	if !isContained(root.canonical, resolved.canonical) {
		return NavFolder{}, "path escapes project root", nil
	}
	if resolved.exists {
		info, err := os.Stat(resolved.canonical)
		if err != nil {
			return NavFolder{}, "", fmt.Errorf("inspect folder target: %w", err)
		}
		if !info.IsDir() {
			return NavFolder{}, "path is not a directory", nil
		}
	}
	cleanPath = filepath.ToSlash(cleanPath)
	label := candidate.label
	if label == "" {
		base := filepath.Base(cleanPath)
		label = strings.ToUpper(base[:1]) + base[1:]
	}
	return NavFolder{ID: candidate.id, Label: label, Path: cleanPath, Source: candidate.source, Exists: resolved.exists}, "", nil
}

// NavFolders returns the fixed documentation folders followed by valid
// user-configured folders. It validates every candidate against the canonical
// project root immediately before exposing it to the viewer.
func (s *ViewerService) NavFolders(ctx context.Context) (*NavFoldersResponse, error) {
	_ = ctx

	root, err := s.canonicalProjectRoot("nav folders")
	if err != nil {
		return nil, err
	}

	response := &NavFoldersResponse{Folders: make([]NavFolder, 0, 2+len(s.browsableFolders))}
	for _, candidate := range []navFolder{
		{id: "architecture", label: "Architecture", path: "docs/architecture", source: "builtin"},
		{id: "product", label: "Product", path: "docs/product", source: "builtin"},
	} {
		folder, rejected, err := buildNavFolder(root, candidate)
		if err != nil {
			return nil, fmt.Errorf("nav folders: validate built-in %q: %w", candidate.path, err)
		}
		if rejected != "" {
			return nil, fmt.Errorf("nav folders: built-in %q rejected: %s", candidate.path, rejected)
		}
		response.Folders = append(response.Folders, folder)
	}
	for _, folder := range s.browsableFolders {
		candidate := navFolder{id: filepath.ToSlash(filepath.Clean(strings.TrimSpace(folder.Path))), label: folder.Label, path: folder.Path, source: "config"}
		responseFolder, rejected, err := buildNavFolder(root, candidate)
		if err != nil {
			slog.Warn("viewer nav folder rejected", "path", folder.Path, "reason", err.Error())
			continue
		}
		if rejected != "" {
			slog.Warn("viewer nav folder rejected", "path", folder.Path, "reason", rejected)
			continue
		}
		response.Folders = append(response.Folders, responseFolder)
	}
	return response, nil
}

// readFolderEntries reads a canonical directory. A directory that disappears
// after path resolution is reported as unavailable rather than an error.
func readFolderEntries(canonicalPath string) ([]os.DirEntry, bool, error) {
	entries, err := os.ReadDir(canonicalPath)
	if err == nil {
		return entries, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("read directory %q: %w", canonicalPath, err)
}

// folderFileEntries converts directory entries into the viewer response DTOs.
func folderFileEntries(relDir string, entries []os.DirEntry) []*FolderFileEntry {
	result := make([]*FolderFileEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		var size int64
		if err == nil && !entry.IsDir() {
			size = info.Size()
		}
		result = append(result, &FolderFileEntry{
			Name:  entry.Name(),
			Path:  filepath.ToSlash(filepath.Join(relDir, entry.Name())),
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}
	return result
}

// emptyFolderFiles returns the stable empty-directory response contract.
func emptyFolderFiles(relPath string) *FolderFilesResponse {
	return &FolderFilesResponse{DirPath: relPath, Entries: []*FolderFileEntry{}}
}

// FolderFiles lists the immediate children of the directory at relPath within the project root.
// It performs the same path-containment security check as FileByPath.
// Hidden files (starting with ".") and common noise entries are included.
// Returns a SecurityError if relPath escapes the project root.
func (s *ViewerService) FolderFiles(ctx context.Context, relPath string) (*FolderFilesResponse, error) {
	if filepath.IsAbs(relPath) {
		return nil, &SecurityError{Path: relPath}
	}

	root, err := s.canonicalProjectRoot("folder files")
	if err != nil {
		return nil, err
	}
	resolved, err := root.resolveRelativePath(relPath)
	if err != nil {
		return nil, fmt.Errorf("folder files: resolve directory path: %w", err)
	}
	if !isContained(root.canonical, resolved.canonical) {
		return nil, &SecurityError{Path: resolved.canonical}
	}
	if !resolved.exists {
		return emptyFolderFiles(relPath), nil
	}

	entries, exists, err := readFolderEntries(resolved.canonical)
	if err != nil {
		return nil, fmt.Errorf("folder files: %w", err)
	}
	if !exists {
		return emptyFolderFiles(relPath), nil
	}

	relDir, err := filepath.Rel(root.canonical, resolved.canonical)
	if err != nil {
		relDir = relPath
	}

	return &FolderFilesResponse{DirPath: relDir, Entries: folderFileEntries(relDir, entries)}, nil
}
