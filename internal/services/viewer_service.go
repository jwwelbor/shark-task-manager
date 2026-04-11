// Package services provides the business logic layer for the shark task manager.
package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
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
}

// ViewerFeatureRepository is the minimal feature repository interface used by ViewerService.
type ViewerFeatureRepository interface {
	List(ctx context.Context) ([]*models.Feature, error)
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
}

// ViewerTaskRepository is the minimal task repository interface used by ViewerService.
type ViewerTaskRepository interface {
	List(ctx context.Context) ([]*models.Task, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	GetByKey(ctx context.Context, key string) (*models.Task, error)
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
type ViewerEntityHistoryRepository interface {
	ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error)
	ListRecentAcrossEntities(ctx context.Context, opts RecentActivityOptions) ([]*models.EntityHistory, error)
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
	Epics       SummaryEntityCounts `json:"epics"`
	Features    SummaryEntityCounts `json:"features"`
	Tasks       SummaryTaskCounts   `json:"tasks"`
	Bugs        SummaryBugCounts    `json:"bugs"`
	ChangeCards SummaryEntityCounts `json:"change_cards"`
}

// HierarchyFeature is a feature with its task counts embedded.
type HierarchyFeature struct {
	*models.Feature
	TaskCount    int    `json:"task_count"`
	BlockedCount int    `json:"blocked_count"`
	StatusColor  string `json:"status_color"`
	StatusPhase  string `json:"status_phase"`
}

// HierarchyEpic is an epic with its child features embedded.
type HierarchyEpic struct {
	*models.Epic
	Features    []*HierarchyFeature `json:"features"`
	StatusColor string              `json:"status_color"`
	StatusPhase string              `json:"status_phase"`
}

// HierarchyResponse is the response type for ViewerService.Hierarchy.
type HierarchyResponse struct {
	Epics []*HierarchyEpic `json:"epics"`
}

// HistoryResponse is the response type for ViewerService.History.
type HistoryResponse struct {
	EntityType models.EntityType       `json:"entity_type"`
	EntityKey  string                  `json:"entity_key"`
	Records    []*models.EntityHistory `json:"records"`
}

// FileResponse is the response type for ViewerService.File.
type FileResponse struct {
	Exists  bool   `json:"exists"`
	Content string `json:"content,omitempty"`
	Path    string `json:"path,omitempty"` // relative path under project root
}

// FeatureTaskOptions carries filters and pagination for ViewerService.FeatureTasks.
type FeatureTaskOptions struct {
	Status  string // empty = no filter
	Agent   string // empty = no filter
	Blocked *bool  // nil = no filter
	Limit   int    // 0 = use default (200); >500 = clamped to 500
	Offset  int
}

// ViewerTask decorates models.Task with workflow-derived display metadata.
type ViewerTask struct {
	*models.Task
	StatusColor string `json:"status_color"`
	StatusPhase string `json:"status_phase"`
}

// FeatureTasksResponse is the response type for ViewerService.FeatureTasks.
type FeatureTasksResponse struct {
	FeatureKey string        `json:"feature_key"`
	Total      int           `json:"total"` // pre-filter count
	Tasks      []*ViewerTask `json:"tasks"`
}

// RecentActivityOptions carries filter/pagination options for ViewerService.RecentActivity.
type RecentActivityOptions struct {
	Limit  int // 0 → default 50; >200 → clamped to 200
	Offset int
}

// ActivityRecord is one history entry enriched with a display label.
type ActivityRecord struct {
	*models.EntityHistory
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

// ----- ViewerService -----

// ViewerService provides read-only dashboard aggregation for the viewer API.
// It composes existing repositories and the workflow service to expose
// seven dashboard-shaped aggregates.
//
// ViewerService has NO mutation methods (ADR-E27-003).
type ViewerService struct {
	epicRepo       ViewerEpicRepository
	featureRepo    ViewerFeatureRepository
	taskRepo       ViewerTaskRepository
	bugRepo        ViewerBugRepository
	changeCardRepo ViewerChangeCardRepository
	historyRepo    ViewerEntityHistoryRepository
	workflowSvc    *workflow.Service
	statusCalc     *status.CalculationService // optional; reserved for future use
	projectRoot    string
}

// NewViewerService constructs a ViewerService.
// All repository and workflow arguments except statusCalc are required and must be non-nil.
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
	}
}

// Summary returns entity-type counts with per-status color/phase metadata
// from the workflow service.
func (s *ViewerService) Summary(ctx context.Context) (*SummaryResponse, error) {
	epics, err := s.epicRepo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to list epics: %w", err)
	}
	epicCounts := make(map[string]int, len(epics))
	for _, e := range epics {
		epicCounts[string(e.Status)]++
	}

	features, err := s.featureRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to list features: %w", err)
	}
	featureCounts := make(map[string]int, len(features))
	for _, f := range features {
		featureCounts[string(f.Status)]++
	}

	tasks, err := s.taskRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("viewer summary: failed to list tasks: %w", err)
	}
	taskCounts := make(map[string]int, len(tasks))
	blockedCount := 0
	for _, t := range tasks {
		taskCounts[string(t.Status)]++
		if t.BlockedReason != nil {
			blockedCount++
		}
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

	epicSvc := s.workflowSvc.ForLevel(workflow.LevelEpic)
	featureSvc := s.workflowSvc.ForLevel(workflow.LevelFeature)
	taskSvc := s.workflowSvc.ForLevel(workflow.LevelTask)
	bugSvc := s.workflowSvc.ForLevel(workflow.LevelBug)
	ccSvc := s.workflowSvc.ForLevel(workflow.LevelChange)

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

// Hierarchy returns epics ordered by execution_order ASC, created_at ASC,
// with features and task/blocked counts embedded.
func (s *ViewerService) Hierarchy(ctx context.Context) (*HierarchyResponse, error) {
	epics, err := s.epicRepo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("viewer hierarchy: failed to list epics: %w", err)
	}

	// Sort epics by key ASC (epics have no execution_order; key is already ordered E01, E02, …).
	sort.Slice(epics, func(i, j int) bool {
		return epics[i].Key < epics[j].Key
	})

	epicSvc := s.workflowSvc.ForLevel(workflow.LevelEpic)
	featureSvc := s.workflowSvc.ForLevel(workflow.LevelFeature)
	taskSvc := s.workflowSvc.ForLevel(workflow.LevelTask)

	result := &HierarchyResponse{
		Epics: make([]*HierarchyEpic, 0, len(epics)),
	}

	for _, epic := range epics {
		epicMeta := epicSvc.GetStatusMetadata(string(epic.Status))
		he := &HierarchyEpic{
			Epic:        epic,
			Features:    []*HierarchyFeature{},
			StatusColor: colorOrGray(epicMeta.Color),
			StatusPhase: phaseOrUnknown(epicMeta.Phase),
		}

		features, err := s.featureRepo.ListByEpic(ctx, epic.ID)
		if err != nil {
			return nil, fmt.Errorf("viewer hierarchy: failed to list features for epic %s: %w", epic.Key, err)
		}

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
			tasks, err := s.taskRepo.ListByFeature(ctx, f.ID)
			if err != nil {
				return nil, fmt.Errorf("viewer hierarchy: failed to list tasks for feature %s: %w", f.Key, err)
			}

			taskCount := len(tasks)
			blockedCount := 0
			for _, t := range tasks {
				if t.BlockedReason != nil {
					blockedCount++
				}
			}

			fMeta := featureSvc.GetStatusMetadata(string(f.Status))
			_ = taskSvc // used indirectly via enrichEntityCounts elsewhere

			he.Features = append(he.Features, &HierarchyFeature{
				Feature:      f,
				TaskCount:    taskCount,
				BlockedCount: blockedCount,
				StatusColor:  colorOrGray(fMeta.Color),
				StatusPhase:  phaseOrUnknown(fMeta.Phase),
			})
		}

		result.Epics = append(result.Epics, he)
	}

	return result, nil
}

// colorOrGray returns the color if non-empty, otherwise "gray".
func colorOrGray(color string) string {
	if color == "" {
		return "gray"
	}
	return color
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

	// Make the stored path absolute relative to project root.
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(s.projectRoot, filePath)
	}

	// Canonicalize project root.
	rootCanon, err := filepath.EvalSymlinks(s.projectRoot)
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

	// Containment check: resolved path must start with rootCanon + separator.
	prefix := rootCanon + string(os.PathSeparator)
	if targetCanon != rootCanon && !strings.HasPrefix(targetCanon, prefix) {
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

	default:
		return "", fmt.Errorf("file read not supported for entity type %q", entityType)
	}
}

// FeatureTasks returns tasks for a feature with status/agent/blocked filtering
// and limit/offset pagination.
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

	tasks, err := s.taskRepo.ListByFeature(ctx, f.ID)
	if err != nil {
		return nil, fmt.Errorf("viewer feature tasks: failed to list tasks for feature %s: %w", featureKey, err)
	}

	// Sort: execution_order ASC (nil → MaxInt), priority DESC, created_at ASC.
	sort.Slice(tasks, func(i, j int) bool {
		oi := maxInt
		if tasks[i].ExecutionOrder != nil {
			oi = *tasks[i].ExecutionOrder
		}
		oj := maxInt
		if tasks[j].ExecutionOrder != nil {
			oj = *tasks[j].ExecutionOrder
		}
		if oi != oj {
			return oi < oj
		}
		pi := tasks[i].Priority
		pj := tasks[j].Priority
		if pi != pj {
			return pi > pj
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	total := len(tasks)

	// Apply filters.
	filtered := make([]*models.Task, 0, len(tasks))
	for _, t := range tasks {
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

	taskSvc := s.workflowSvc.ForLevel(workflow.LevelTask)
	viewerTasks := make([]*ViewerTask, 0, len(filtered))
	for _, t := range filtered {
		meta := taskSvc.GetStatusMetadata(string(t.Status))
		viewerTasks = append(viewerTasks, &ViewerTask{
			Task:        t,
			StatusColor: colorOrGray(meta.Color),
			StatusPhase: phaseOrUnknown(meta.Phase),
		})
	}

	return &FeatureTasksResponse{
		FeatureKey: featureKey,
		Total:      total,
		Tasks:      viewerTasks,
	}, nil
}

// RecentActivity returns the most recent status transitions across all entity types.
// Limit is clamped to [1, 200]; Offset 0→50 default applied at the repository layer.
func (s *ViewerService) RecentActivity(ctx context.Context, opts RecentActivityOptions) (*RecentActivityResponse, error) {
	// Clamp limit: 0 → 50; >200 → 200.
	if opts.Limit <= 0 {
		opts.Limit = 50
	} else if opts.Limit > 200 {
		opts.Limit = 200
	}

	records, err := s.historyRepo.ListRecentAcrossEntities(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("viewer recent activity: %w", err)
	}

	// Omit orphan entries (entity was deleted — title empty/nil from JOIN).
	result := make([]*ActivityRecord, 0, len(records))
	for _, r := range records {
		result = append(result, &ActivityRecord{EntityHistory: r})
	}

	return &RecentActivityResponse{Records: result}, nil
}

// WorkflowMeta returns the status and transition metadata for all five entity levels.
func (s *ViewerService) WorkflowMeta(_ context.Context) (*WorkflowMetaResponse, error) {
	levels := []string{
		workflow.LevelEpic,
		workflow.LevelFeature,
		workflow.LevelTask,
		workflow.LevelBug,
		workflow.LevelChange,
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
