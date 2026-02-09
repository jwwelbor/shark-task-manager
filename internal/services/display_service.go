package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

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
	ResolvedPath string `json:"path,omitempty"`
	Filename     string `json:"filename,omitempty"`
	StatusSource string `json:"status_source"`
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
	ResolvedPath string `json:"path,omitempty"`
	StatusSource string `json:"status_source"`
}

// StatusCountItem is a simplified status count for JSON output
type StatusCountItem struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// DisplayServiceDeps holds the repository dependencies for DisplayService
type DisplayServiceDeps struct {
	EpicRepo     *repository.EpicRepository
	FeatureRepo  *repository.FeatureRepository
	TaskRepo     *repository.TaskRepository
	DocumentRepo *repository.DocumentRepository
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

// NewDisplayService creates a new DisplayService.
// It loads level-specific workflow configurations and falls back to defaults.
func NewDisplayService(db *repository.DB, workflowSvc *workflow.Service) *DisplayService {
	return &DisplayService{
		deps: DisplayServiceDeps{
			EpicRepo:     repository.NewEpicRepository(db),
			FeatureRepo:  repository.NewFeatureRepository(db),
			TaskRepo:     repository.NewTaskRepository(db),
			DocumentRepo: repository.NewDocumentRepository(db),
		},
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

// DetermineFeatureDisplayMode checks the feature's current status against workflow config
// to decide if it should show planning or aggregation mode.
func (s *DisplayService) DetermineFeatureDisplayMode(feature *models.Feature) DisplayMode {
	return s.determineDisplayMode(string(feature.Status), s.featureWorkflow)
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
	current := startStatuses[0]

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

		// Follow first transition (happy path), skip blocked/on_hold
		next := ""
		for _, t := range transitions {
			if !strings.Contains(t, "blocked") && !strings.Contains(t, "on_hold") && !strings.Contains(t, "cancelled") {
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
		s.populateEpicPlanningInfo(info)
	} else {
		if err := s.populateEpicAggregationInfo(ctx, info); err != nil {
			return nil, err
		}
	}

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
		s.populateFeaturePlanningInfo(info)
	} else {
		if err := s.populateFeatureAggregationInfo(ctx, info); err != nil {
			return nil, err
		}
	}

	return info, nil
}

// populateEpicPlanningInfo fills in planning-mode fields for an epic.
func (s *DisplayService) populateEpicPlanningInfo(info *EpicDisplayInfo) {
	status := string(info.Epic.Status)
	meta, found := s.epicWorkflow.GetStatusMetadata(status)

	info.StatusSource = "workflow"

	if found {
		info.Phase = meta.Phase
		info.PhaseDescription = meta.Description
	}

	info.WorkflowPosition = s.BuildWorkflowPosition(status, s.epicWorkflow)
}

// populateEpicAggregationInfo fills in aggregation-mode fields for an epic.
// This delegates to existing repository methods to maintain backward compatibility.
func (s *DisplayService) populateEpicAggregationInfo(ctx context.Context, info *EpicDisplayInfo) error {
	epicID := info.Epic.ID

	info.StatusSource = "calculated"

	// Calculate progress
	progress, err := s.deps.EpicRepo.CalculateProgress(ctx, epicID)
	if err != nil {
		progress = 0.0
	}
	info.Progress = progress

	// Get features
	features, err := s.deps.FeatureRepo.ListByEpic(ctx, epicID)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	// Build feature display items with per-feature mode determination
	info.Features = make([]FeatureDisplayItem, 0, len(features))
	for _, feature := range features {
		// Update progress
		if err := s.deps.FeatureRepo.UpdateProgress(ctx, feature.ID); err == nil {
			if updated, err := s.deps.FeatureRepo.GetByID(ctx, feature.ID); err == nil {
				feature = updated
			}
		}

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
	if blockCount, ok := info.TaskRollup["blocked"]; ok && blockCount > 0 {
		blockedTasks, err := s.deps.TaskRepo.ListBlockedTasksByEpic(ctx, info.Epic.Key)
		if err == nil {
			info.BlockedTasks = blockedTasks
		}
	}
	if info.BlockedTasks == nil {
		info.BlockedTasks = make([]*models.Task, 0)
	}

	// Approval backlog
	if approvalCount, ok := info.TaskRollup["ready_for_review"]; ok {
		info.ApprovalBacklog = approvalCount
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
func (s *DisplayService) populateFeaturePlanningInfo(info *FeatureDisplayInfo) {
	status := string(info.Feature.Status)
	meta, found := s.featureWorkflow.GetStatusMetadata(status)

	info.StatusSource = "workflow"

	if found {
		info.Phase = meta.Phase
		info.PhaseDescription = meta.Description
	}

	info.WorkflowPosition = s.BuildWorkflowPosition(status, s.featureWorkflow)
}

// populateFeatureAggregationInfo fills in aggregation-mode fields for a feature.
func (s *DisplayService) populateFeatureAggregationInfo(ctx context.Context, info *FeatureDisplayInfo) error {
	featureID := info.Feature.ID

	info.StatusSource = "calculated"

	// Update progress
	if err := s.deps.FeatureRepo.UpdateProgress(ctx, featureID); err == nil {
		if updated, err := s.deps.FeatureRepo.GetByID(ctx, featureID); err == nil {
			info.Feature = updated
		}
	}

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
