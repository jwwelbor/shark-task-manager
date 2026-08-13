package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// DisplayEpicRepository is the interface DisplayService needs for epic data access.
type DisplayEpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetFeatureProgressDataByEpic(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error)
	GetFeatureStatusRollup(ctx context.Context, epicID int64) (map[string]int, error)
	GetTaskStatusRollup(ctx context.Context, epicID int64) (map[string]int, error)
}

// DisplayFeatureRepository is the interface DisplayService needs for feature data access.
type DisplayFeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
	GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
}

// DisplayTaskRepository is the interface DisplayService needs for task data access.
type DisplayTaskRepository interface {
	GetTaskCountForFeature(ctx context.Context, featureID int64) (int, error)
	ListBlockedTasksByEpic(ctx context.Context, epicKey string, blockedStatuses []string) ([]*models.Task, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
}

// DisplayNoteRepository is the interface DisplayService needs for note data access.
type DisplayNoteRepository interface {
	GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

// DisplayTemplateEnrichmentRepository is the interface DisplayService needs for template enrichment data access.
type DisplayTemplateEnrichmentRepository interface {
	GetEpicEnrichment(ctx context.Context, epicID int64) (*config.TemplateEnrichmentData, error)
	GetFeatureEnrichment(ctx context.Context, featureID int64) (*config.TemplateEnrichmentData, error)
	GetTaskEnrichment(ctx context.Context, taskID int64) (*config.TemplateEnrichmentData, error)
}

// DisplayMode indicates whether an entity is in planning or aggregation mode
type DisplayMode string

const (
	// DisplayModePlanning means the entity has its own workflow status (not aggregating children)
	DisplayModePlanning DisplayMode = "planning"
	// DisplayModeAggregation means the entity derives progress from children
	DisplayModeAggregation DisplayMode = "aggregation"
)

// WorkflowPosition represents where an entity is in its workflow
type WorkflowPosition struct {
	Statuses      []string `json:"statuses"`
	CurrentIndex  int      `json:"current_index"`
	CurrentStatus string   `json:"current_status"`
}

// FeatureDisplayItem represents a feature in the epic's feature list
type FeatureDisplayItem struct {
	Feature   *models.Feature `json:"feature"`
	TaskCount int             `json:"task_count"`
	Mode      DisplayMode     `json:"display_mode"`
	Phase     string          `json:"phase,omitempty"`
}

// EpicDisplayInfo contains all data needed to render an epic's details
type EpicDisplayInfo struct {
	Epic *models.Epic `json:"epic"`
	Mode DisplayMode  `json:"display_mode"`

	// Planning mode fields
	Phase            string            `json:"phase,omitempty"`
	PhaseDescription string            `json:"phase_description,omitempty"`
	WorkflowPosition *WorkflowPosition `json:"workflow_position,omitempty"`

	// Aggregation mode fields
	Progress        float64              `json:"progress_pct,omitempty"`
	Features        []FeatureDisplayItem `json:"features,omitempty"`
	FeatureRollup   map[string]int       `json:"feature_status_rollup,omitempty"`
	TaskRollup      map[string]int       `json:"task_status_rollup,omitempty"`
	BlockedTasks    []*models.Task       `json:"impediments,omitempty"`
	ApprovalBacklog int                  `json:"approval_backlog_count,omitempty"`
	RelatedDocs     []*models.Document   `json:"related_documents,omitempty"`

	// Common fields
	Notes              []*models.EntityNote    `json:"notes,omitempty"`
	ContextData        *models.ContextData     `json:"context_data,omitempty"`
	ResolvedPath       string                  `json:"path,omitempty"`
	Filename           string                  `json:"filename,omitempty"`
	StatusSource       string                  `json:"status_source"`
	ValidTransitions   []string                `json:"valid_transitions,omitempty"`
	OrchestratorAction *config.PopulatedAction `json:"orchestrator_action,omitempty"`
}

// FeatureDisplayInfo contains all data needed to render a feature's details
type FeatureDisplayInfo struct {
	Feature *models.Feature `json:"feature"`
	Mode    DisplayMode     `json:"display_mode"`

	// Planning mode fields
	Phase            string            `json:"phase,omitempty"`
	PhaseDescription string            `json:"phase_description,omitempty"`
	WorkflowPosition *WorkflowPosition `json:"workflow_position,omitempty"`

	// Aggregation mode fields
	Tasks           []*models.Task     `json:"tasks,omitempty"`
	StatusBreakdown []StatusCountItem  `json:"status_breakdown,omitempty"`
	RelatedDocs     []*models.Document `json:"related_documents,omitempty"`

	// Common fields
	Notes              []*models.EntityNote    `json:"notes,omitempty"`
	ContextData        *models.ContextData     `json:"context_data,omitempty"`
	ResolvedPath       string                  `json:"path,omitempty"`
	StatusSource       string                  `json:"status_source"`
	ValidTransitions   []string                `json:"valid_transitions,omitempty"`
	OrchestratorAction *config.PopulatedAction `json:"orchestrator_action,omitempty"`
}

// StatusCountItem is a simplified status count for JSON output
type StatusCountItem struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// DisplayServiceDeps holds the repository dependencies for DisplayService.
// All fields use narrow interfaces to enable testing with mock implementations.
type DisplayServiceDeps struct {
	EpicRepo               DisplayEpicRepository
	FeatureRepo            DisplayFeatureRepository
	TaskRepo               DisplayTaskRepository
	DocumentRepo           config.DocumentRepository
	NoteRepo               DisplayNoteRepository
	TaskRelRepo            config.TaskRelationshipRepository
	TemplateEnrichmentRepo DisplayTemplateEnrichmentRepository
}

// DisplayService encapsulates the planning-vs-aggregation display logic.
// It determines whether an entity should show its own workflow position
// or aggregated child progress based on workflow configuration.
type DisplayService struct {
	deps            DisplayServiceDeps
	epicWorkflow    *config.WorkflowConfig
	featureWorkflow *config.WorkflowConfig
	workflowSvc     *workflow.Service
}

// NewDisplayService creates a new DisplayService with pre-wired dependencies.
// Callers are responsible for constructing and injecting all repository dependencies.
// It loads level-specific workflow configurations from workflowSvc.
func NewDisplayService(deps DisplayServiceDeps, workflowSvc *workflow.Service) *DisplayService {
	return &DisplayService{
		deps:            deps,
		epicWorkflow:    workflowSvc.ForLevel(workflow.LevelEpic).GetWorkflow(),
		featureWorkflow: workflowSvc.ForLevel(workflow.LevelFeature).GetWorkflow(),
		workflowSvc:     workflowSvc,
	}
}

// DetermineEpicDisplayMode checks the epic's current status against workflow config
// to decide if it should show planning or aggregation mode.
func (s *DisplayService) DetermineEpicDisplayMode(epic *models.Epic) DisplayMode {
	return s.determineDisplayMode(string(epic.Status), s.epicWorkflow)
}

// DetermineEpicDisplayModeByStatus checks an epic status string against workflow config
// to decide planning vs aggregation mode. No database access required.
func (s *DisplayService) DetermineEpicDisplayModeByStatus(status string) DisplayMode {
	return s.determineDisplayMode(status, s.epicWorkflow)
}

// GetEpicPhase returns the workflow phase for an epic status, or empty string if unavailable.
func (s *DisplayService) GetEpicPhase(status string) string {
	if s.epicWorkflow == nil {
		return ""
	}
	if meta, found := s.epicWorkflow.GetStatusMetadata(status); found {
		return meta.Phase
	}
	return ""
}

// DetermineFeatureDisplayMode checks the feature's current status against workflow config
// to decide if it should show planning or aggregation mode.
func (s *DisplayService) DetermineFeatureDisplayMode(feature *models.Feature) DisplayMode {
	return s.determineDisplayMode(string(feature.Status), s.featureWorkflow)
}

// GetFeaturePhase returns the workflow phase for a feature status, or empty string if unavailable.
func (s *DisplayService) GetFeaturePhase(status string) string {
	if s.featureWorkflow == nil {
		return ""
	}
	if meta, found := s.featureWorkflow.GetStatusMetadata(status); found {
		return meta.Phase
	}
	return ""
}

// determineDisplayMode is the core algorithm for deciding planning vs aggregation.
func (s *DisplayService) determineDisplayMode(currentStatus string, wfCfg *config.WorkflowConfig) DisplayMode {
	if wfCfg == nil {
		return DisplayModeAggregation
	}

	// Check if current status is in _aggregation_ special statuses
	aggStatuses, exists := wfCfg.SpecialStatuses[config.AggregationStatusKey]
	if exists {
		for _, aggStatus := range aggStatuses {
			if strings.EqualFold(aggStatus, currentStatus) {
				return DisplayModeAggregation
			}
		}
	}

	// Check is_planning from status metadata
	meta, found := wfCfg.GetStatusMetadata(currentStatus)
	if found && meta.IsPlanning {
		return DisplayModePlanning
	}

	// Default: aggregation (includes terminal statuses like completed)
	return DisplayModeAggregation
}

// BuildWorkflowPosition constructs the workflow position data for display.
func (s *DisplayService) BuildWorkflowPosition(currentStatus string, wfCfg *config.WorkflowConfig) *WorkflowPosition {
	if wfCfg == nil {
		return nil
	}

	// Build ordered status list from status_flow keys
	// Use a topological ordering approach: start from _start_ statuses and follow transitions
	ordered := buildOrderedStatuses(wfCfg)

	currentIdx := -1
	for i, st := range ordered {
		if strings.EqualFold(st, currentStatus) {
			currentIdx = i
			break
		}
	}

	return &WorkflowPosition{
		Statuses:      ordered,
		CurrentIndex:  currentIdx,
		CurrentStatus: currentStatus,
	}
}

// buildOrderedStatuses creates a linear ordering of statuses from the workflow.
// It follows the "happy path" (first transition at each step) from start to terminal.
func buildOrderedStatuses(wfCfg *config.WorkflowConfig) []string {
	if wfCfg == nil {
		return nil
	}

	// Find start status
	startStatuses, exists := wfCfg.SpecialStatuses[config.StartStatusKey]
	if !exists || len(startStatuses) == 0 {
		return nil
	}

	// Build the "happy path" by following first transitions
	seen := make(map[string]bool)
	var ordered []string
	current := startStatuses[0] //shark:ordered _start_ preserves the author's declaration order

	for {
		if seen[current] {
			break
		}
		seen[current] = true
		ordered = append(ordered, current)

		transitions, ok := wfCfg.StatusFlow[current]
		if !ok || len(transitions) == 0 {
			break
		}

		// Follow first transition (happy path), skip special statuses
		specialStatuses := map[string]bool{"blocked": true, "on_hold": true, "cancelled": true}
		next := ""
		for _, t := range transitions {
			if !specialStatuses[t] {
				next = t
				break
			}
		}
		if next == "" {
			break
		}
		current = next
	}

	return ordered
}

// GetEpicDisplayInfo assembles all data needed to display an epic.
func (s *DisplayService) GetEpicDisplayInfo(ctx context.Context, epicKey string) (*EpicDisplayInfo, error) {
	epic, err := s.deps.EpicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}

	mode := s.DetermineEpicDisplayMode(epic)

	info := &EpicDisplayInfo{
		Epic: epic,
		Mode: mode,
	}

	if mode == DisplayModePlanning {
		if err := s.populateEpicPlanningInfo(ctx, info); err != nil {
			return nil, err
		}
	} else {
		if err := s.populateEpicAggregationInfo(ctx, info); err != nil {
			return nil, err
		}
	}

	// Populate valid transitions
	info.ValidTransitions = s.workflowSvc.ForLevel(workflow.LevelEpic).GetValidTransitions(string(epic.Status))
	if info.ValidTransitions == nil {
		info.ValidTransitions = []string{}
	}

	// Populate orchestrator action for both modes
	info.OrchestratorAction = s.ResolveEpicAction(ctx, epic)

	return info, nil
}

// GetFeatureDisplayInfo assembles all data needed to display a feature.
func (s *DisplayService) GetFeatureDisplayInfo(ctx context.Context, featureKey string) (*FeatureDisplayInfo, error) {
	feature, err := s.deps.FeatureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	mode := s.DetermineFeatureDisplayMode(feature)

	info := &FeatureDisplayInfo{
		Feature: feature,
		Mode:    mode,
	}

	if mode == DisplayModePlanning {
		if err := s.populateFeaturePlanningInfo(ctx, info); err != nil {
			return nil, err
		}
	} else {
		if err := s.populateFeatureAggregationInfo(ctx, info); err != nil {
			return nil, err
		}
	}

	// Populate valid transitions
	info.ValidTransitions = s.workflowSvc.ForLevel(workflow.LevelFeature).GetValidTransitions(string(feature.Status))
	if info.ValidTransitions == nil {
		info.ValidTransitions = []string{}
	}

	// Populate orchestrator action for both modes
	info.OrchestratorAction = s.ResolveFeatureAction(ctx, feature)

	return info, nil
}

// populateEpicPlanningInfo fills in planning-mode fields for an epic.
// It also fetches features so they are visible even in planning mode.
func (s *DisplayService) populateEpicPlanningInfo(ctx context.Context, info *EpicDisplayInfo) error {
	status := string(info.Epic.Status)
	meta, found := s.epicWorkflow.GetStatusMetadata(status)

	info.StatusSource = "workflow"

	if found {
		info.Phase = meta.Phase
		info.PhaseDescription = meta.Description
	}

	info.WorkflowPosition = s.BuildWorkflowPosition(status, s.epicWorkflow)

	// Always fetch features so they're visible even in planning mode
	features, err := s.deps.FeatureRepo.ListByEpic(ctx, info.Epic.ID)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	info.Features = make([]FeatureDisplayItem, 0, len(features))
	for _, feature := range features {
		taskCount, err := s.deps.TaskRepo.GetTaskCountForFeature(ctx, feature.ID)
		if err != nil {
			taskCount = 0
		}

		featureMode := s.DetermineFeatureDisplayMode(feature)
		featurePhase := ""
		if featureMode == DisplayModePlanning {
			if meta, found := s.featureWorkflow.GetStatusMetadata(string(feature.Status)); found {
				featurePhase = meta.Phase
			}
		}

		info.Features = append(info.Features, FeatureDisplayItem{
			Feature:   feature,
			TaskCount: taskCount,
			Mode:      featureMode,
			Phase:     featurePhase,
		})
	}

	// Fetch notes (log and continue on error)
	if s.deps.NoteRepo != nil {
		notes, err := s.deps.NoteRepo.GetByEntity(ctx, models.EntityTypeEpic, info.Epic.ID)
		if err == nil {
			info.Notes = notes
		}
	}
	if info.Notes == nil {
		info.Notes = make([]*models.EntityNote, 0)
	}

	// Parse context data from epic's JSON context field
	if info.Epic.ContextData != nil && *info.Epic.ContextData != "" {
		cd, err := models.FromJSON(*info.Epic.ContextData)
		if err == nil {
			info.ContextData = cd
		}
	}

	// Related documents
	relatedDocs, err := s.deps.DocumentRepo.ListForEpic(ctx, info.Epic.ID)
	if err == nil {
		info.RelatedDocs = relatedDocs
	}
	if info.RelatedDocs == nil {
		info.RelatedDocs = make([]*models.Document, 0)
	}

	return nil
}

// ResolveEpicAction looks up the orchestrator action for an epic's current status.
// Returns nil if no action is defined for the status.
// Uses EpicPlaceholdersWithRelated to include related documents.
// This is a public method for use by CLI commands.
func (s *DisplayService) ResolveEpicAction(ctx context.Context, epic *models.Epic) *config.PopulatedAction {
	if s.epicWorkflow == nil || s.epicWorkflow.StatusMetadata == nil {
		return nil
	}

	status := string(epic.Status)
	meta, exists := s.epicWorkflow.StatusMetadata[status]
	if !exists || meta.OrchestratorAction == nil {
		return nil
	}

	// Fetch enrichment data (optional, graceful degradation)
	var enrichment *config.TemplateEnrichmentData
	if s.deps.TemplateEnrichmentRepo != nil {
		data, err := s.deps.TemplateEnrichmentRepo.GetEpicEnrichment(ctx, epic.ID)
		if err != nil {
			slog.Warn("Failed to fetch enrichment data for epic", "epic", epic.Key, "error", err)
		} else {
			enrichment = data
		}
	}

	// Use EpicPlaceholdersWithRelated to populate placeholders with related docs
	// Note: We don't have an epic relationship repository yet, so pass nil
	placeholders := config.EpicPlaceholdersWithRelated(epic, s.deps.DocumentRepo, nil, ctx, enrichment)

	// Display context (shark get): entity is already in this status, so show RESUME CONTEXT preamble.
	placeholders["is_resume"] = "true"

	return meta.OrchestratorAction.ToPopulatedAction(placeholders)
}

// populateEpicAggregationInfo fills in aggregation-mode fields for an epic.
// This delegates to existing repository methods to maintain backward compatibility.
func (s *DisplayService) populateEpicAggregationInfo(ctx context.Context, info *EpicDisplayInfo) error {
	epicID := info.Epic.ID

	info.StatusSource = "calculated"

	// Calculate progress from raw feature data (same business logic as EpicService.CalculateProgress)
	progressData, err := s.deps.EpicRepo.GetFeatureProgressDataByEpic(ctx, epicID)
	if err != nil {
		info.Progress = 0.0
	} else if len(progressData) == 0 {
		info.Progress = 0.0
	} else {
		var totalProgress float64
		featureWorkflow := s.workflowSvc.ForLevel(workflow.LevelFeature)
		for _, d := range progressData {
			if featureWorkflow.IsTerminalStatus(d.Status) {
				totalProgress += 100.0
			} else {
				totalProgress += d.ProgressPct
			}
		}
		info.Progress = totalProgress / float64(len(progressData))
	}

	// Get features
	features, err := s.deps.FeatureRepo.ListByEpic(ctx, epicID)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	// Build feature display items with per-feature mode determination
	info.Features = make([]FeatureDisplayItem, 0, len(features))
	for _, feature := range features {
		taskCount, err := s.deps.TaskRepo.GetTaskCountForFeature(ctx, feature.ID)
		if err != nil {
			taskCount = 0
		}

		featureMode := s.DetermineFeatureDisplayMode(feature)
		featurePhase := ""
		if featureMode == DisplayModePlanning {
			if meta, found := s.featureWorkflow.GetStatusMetadata(string(feature.Status)); found {
				featurePhase = meta.Phase
			}
		}

		info.Features = append(info.Features, FeatureDisplayItem{
			Feature:   feature,
			TaskCount: taskCount,
			Mode:      featureMode,
			Phase:     featurePhase,
		})
	}

	// Feature status rollup
	featureRollup, err := s.deps.EpicRepo.GetFeatureStatusRollup(ctx, epicID)
	if err == nil {
		info.FeatureRollup = featureRollup
	} else {
		info.FeatureRollup = make(map[string]int)
	}

	// Task status rollup
	taskRollup, err := s.deps.EpicRepo.GetTaskStatusRollup(ctx, epicID)
	if err == nil {
		info.TaskRollup = taskRollup
	} else {
		info.TaskRollup = make(map[string]int)
	}

	// Blocked tasks
	blockedStatuses := s.workflowSvc.ForLevel(workflow.LevelTask).GetStatusesByPhase("blocked")
	blockCount := 0
	for _, blockedStatus := range blockedStatuses {
		blockCount += info.TaskRollup[blockedStatus]
	}
	if blockCount > 0 {
		blockedTasks, err := s.deps.TaskRepo.ListBlockedTasksByEpic(ctx, info.Epic.Key, blockedStatuses)
		if err == nil {
			info.BlockedTasks = blockedTasks
		}
	}
	if info.BlockedTasks == nil {
		info.BlockedTasks = make([]*models.Task, 0)
	}

	// Approval backlog
	taskWorkflow := s.workflowSvc.ForLevel(workflow.LevelTask)
	for status, count := range info.TaskRollup {
		meta := taskWorkflow.GetStatusMetadata(status)
		if meta.Phase == "review" || meta.Phase == "approval" {
			info.ApprovalBacklog += count
		}
	}

	// Related documents
	relatedDocs, err := s.deps.DocumentRepo.ListForEpic(ctx, epicID)
	if err == nil {
		info.RelatedDocs = relatedDocs
	}
	if info.RelatedDocs == nil {
		info.RelatedDocs = make([]*models.Document, 0)
	}

	return nil
}

// populateFeaturePlanningInfo fills in planning-mode fields for a feature.
// It also fetches tasks so they are visible even in planning mode.
func (s *DisplayService) populateFeaturePlanningInfo(ctx context.Context, info *FeatureDisplayInfo) error {
	status := string(info.Feature.Status)
	meta, found := s.featureWorkflow.GetStatusMetadata(status)

	info.StatusSource = "workflow"

	if found {
		info.Phase = meta.Phase
		info.PhaseDescription = meta.Description
	}

	info.WorkflowPosition = s.BuildWorkflowPosition(status, s.featureWorkflow)

	// Always fetch tasks so they're visible even in planning mode
	tasks, err := s.deps.TaskRepo.ListByFeature(ctx, info.Feature.ID)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}
	info.Tasks = tasks

	// Fetch notes (log and continue on error)
	if s.deps.NoteRepo != nil {
		notes, err := s.deps.NoteRepo.GetByEntity(ctx, models.EntityTypeFeature, info.Feature.ID)
		if err == nil {
			info.Notes = notes
		}
	}
	if info.Notes == nil {
		info.Notes = make([]*models.EntityNote, 0)
	}

	// Parse context data from feature's JSON context field
	if info.Feature.ContextData != nil && *info.Feature.ContextData != "" {
		cd, err := models.FromJSON(*info.Feature.ContextData)
		if err == nil {
			info.ContextData = cd
		}
	}

	// Related documents
	relatedDocs, err := s.deps.DocumentRepo.ListForFeature(ctx, info.Feature.ID)
	if err == nil {
		info.RelatedDocs = relatedDocs
	}
	if info.RelatedDocs == nil {
		info.RelatedDocs = make([]*models.Document, 0)
	}

	return nil
}

// ResolveFeatureAction looks up the orchestrator action for a feature's current status.
// Returns nil if no action is defined for the status.
// Uses FeaturePlaceholdersWithRelated to include related documents.
// This is a public method for use by CLI commands.
func (s *DisplayService) ResolveFeatureAction(ctx context.Context, feature *models.Feature) *config.PopulatedAction {
	if s.featureWorkflow == nil || s.featureWorkflow.StatusMetadata == nil {
		return nil
	}

	status := string(feature.Status)
	meta, exists := s.featureWorkflow.StatusMetadata[status]
	if !exists || meta.OrchestratorAction == nil {
		return nil
	}

	// Fetch enrichment data (optional, graceful degradation)
	var enrichment *config.TemplateEnrichmentData
	if s.deps.TemplateEnrichmentRepo != nil {
		data, err := s.deps.TemplateEnrichmentRepo.GetFeatureEnrichment(ctx, feature.ID)
		if err != nil {
			slog.Warn("Failed to fetch enrichment data for feature", "feature", feature.Key, "error", err)
		} else {
			enrichment = data
		}
	}

	// Use FeaturePlaceholdersWithRelated to populate placeholders with related docs
	// Note: We don't have a feature relationship repository yet, so pass nil
	placeholders := config.FeaturePlaceholdersWithRelated(ctx, feature, s.deps.DocumentRepo, nil, enrichment)

	// Display context (shark get): entity is already in this status, so show RESUME CONTEXT preamble.
	placeholders["is_resume"] = "true"

	return meta.OrchestratorAction.ToPopulatedAction(placeholders)
}

// ResolveTaskAction looks up the orchestrator action for a task's current status.
// Returns nil if no action is defined for the status.
// Uses TaskPlaceholdersWithRelated to include related documents and tasks.
// This is a public method for use by CLI commands.
func (s *DisplayService) ResolveTaskAction(ctx context.Context, task *models.Task) *config.PopulatedAction {
	taskWorkflow := s.workflowSvc.ForLevel(workflow.LevelTask).GetWorkflow()
	if taskWorkflow == nil || taskWorkflow.StatusMetadata == nil {
		return nil
	}

	status := string(task.Status)
	meta, exists := taskWorkflow.StatusMetadata[status]
	if !exists || meta.OrchestratorAction == nil {
		return nil
	}

	// Fetch enrichment data (optional, graceful degradation)
	var taskEnrichment *config.TemplateEnrichmentData
	if s.deps.TemplateEnrichmentRepo != nil {
		data, err := s.deps.TemplateEnrichmentRepo.GetTaskEnrichment(ctx, task.ID)
		if err != nil {
			slog.Warn("Failed to fetch enrichment data for task", "task", task.Key, "error", err)
		} else {
			taskEnrichment = data
		}
	}

	// Use TaskPlaceholdersWithRelated to populate placeholders with related docs and tasks
	placeholders := config.TaskPlaceholdersWithRelated(ctx, task, s.deps.DocumentRepo, s.deps.TaskRelRepo, taskEnrichment)

	// Display context (shark get): entity is already in this status, so show RESUME CONTEXT preamble.
	placeholders["is_resume"] = "true"

	return meta.OrchestratorAction.ToPopulatedAction(placeholders)
}

// populateFeatureAggregationInfo fills in aggregation-mode fields for a feature.
func (s *DisplayService) populateFeatureAggregationInfo(ctx context.Context, info *FeatureDisplayInfo) error {
	featureID := info.Feature.ID

	info.StatusSource = "calculated"

	// Get tasks
	tasks, err := s.deps.TaskRepo.ListByFeature(ctx, featureID)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}
	info.Tasks = tasks

	// Status breakdown
	statusCounts, err := s.deps.FeatureRepo.GetTaskStatusBreakdown(ctx, featureID)
	if err == nil {
		info.StatusBreakdown = make([]StatusCountItem, 0)
		for status, count := range statusCounts {
			if count > 0 {
				info.StatusBreakdown = append(info.StatusBreakdown, StatusCountItem{
					Status: string(status),
					Count:  count,
				})
			}
		}
	}

	// Related documents
	relatedDocs, err := s.deps.DocumentRepo.ListForFeature(ctx, featureID)
	if err == nil {
		info.RelatedDocs = relatedDocs
	}
	if info.RelatedDocs == nil {
		info.RelatedDocs = make([]*models.Document, 0)
	}

	return nil
}
