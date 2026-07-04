package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// EpicAnalyticsRepository defines the subset of EpicRepository needed by EpicAnalyticsService.
// This interface is satisfied by *repository.EpicRepository.
type EpicAnalyticsRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetFeatureProgressDataByEpic(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error)
	GetFeatureStatusRollup(ctx context.Context, epicID int64) (map[string]int, error)
	GetTaskStatusRollup(ctx context.Context, epicID int64) (map[string]int, error)
	GetEpicDisplayDataRaw(ctx context.Context, epicID int64) (*repository.EpicDisplayDataRaw, error)
}

// EpicAnalyticsTaskRepository defines the subset of task repository needed by EpicAnalyticsService.
// This interface is satisfied by *repository.TaskRepository.
type EpicAnalyticsTaskRepository interface {
	ListBlockedTasksByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
}

// EpicAnalyticsService provides analytics and display-data computation for epics.
// It is a focused sub-service extracted from EpicService to reduce file size.
//
// Responsibilities:
//   - Progress calculation (weighted and completion-based)
//   - Feature rollup and task status rollup
//   - Impediment/blocked task retrieval
//   - Epic health assessment
//   - Display data aggregation (single-query view)
type EpicAnalyticsService struct {
	repo     EpicAnalyticsRepository
	taskRepo EpicAnalyticsTaskRepository // optional; degrades gracefully if nil
}

// NewEpicAnalyticsService creates a new EpicAnalyticsService.
//
// Parameters:
//   - repo: epic repository for analytics queries (required)
//   - taskRepo: task repository for blocked-task queries (optional; pass nil to degrade gracefully)
func NewEpicAnalyticsService(repo EpicAnalyticsRepository, taskRepo EpicAnalyticsTaskRepository) *EpicAnalyticsService {
	requireNonNil(repo, "EpicAnalyticsService requires a non-nil EpicAnalyticsRepository")
	return &EpicAnalyticsService{
		repo:     repo,
		taskRepo: taskRepo,
	}
}

// CalculateProgress computes epic progress from raw feature data.
// Business rule: completed/archived features count as 100% progress regardless
// of their stored progress_pct value. All other features use their stored
// progress_pct. Epic progress is the average across all features.
// Returns 0 if the epic has no features.
func (s *EpicAnalyticsService) CalculateProgress(ctx context.Context, epicID int64) (float64, error) {
	data, err := s.repo.GetFeatureProgressDataByEpic(ctx, epicID)
	if err != nil {
		return 0, fmt.Errorf("failed to get feature progress data: %w", err)
	}

	if len(data) == 0 {
		return 0, nil
	}

	var totalProgress float64
	activeFeatures := 0
	for _, d := range data {
		// Skip cancelled features — they don't count toward epic progress
		if d.Status == "cancelled" {
			continue
		}
		activeFeatures++
		if d.Status == "completed" || d.Status == "archived" {
			totalProgress += 100.0
		} else {
			totalProgress += d.ProgressPct
		}
	}

	if activeFeatures == 0 {
		return 0, nil
	}

	return totalProgress / float64(activeFeatures), nil
}

// GetProgress retrieves progress metrics for an epic.
func (s *EpicAnalyticsService) GetProgress(ctx context.Context, key string) (*EpicProgressInfo, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	progressPct, err := s.CalculateProgress(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate progress for epic %s: %w", key, err)
	}

	featureRollup, err := s.repo.GetFeatureStatusRollup(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature rollup for epic %s: %w", key, err)
	}

	taskRollup, err := s.repo.GetTaskStatusRollup(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task rollup for epic %s: %w", key, err)
	}

	totalFeatures := 0
	for _, count := range featureRollup {
		totalFeatures += count
	}

	return &EpicProgressInfo{
		EpicKey:       key,
		ProgressPct:   math.Round(progressPct*100) / 100,
		TotalFeatures: totalFeatures,
		TaskRollup:    taskRollup,
	}, nil
}

// GetFeatureRollup aggregates feature statuses for an epic.
func (s *EpicAnalyticsService) GetFeatureRollup(ctx context.Context, key string) (*FeatureRollup, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	statusCounts, err := s.repo.GetFeatureStatusRollup(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature rollup for epic %s: %w", key, err)
	}

	totalFeatures := 0
	for _, count := range statusCounts {
		totalFeatures += count
	}

	return &FeatureRollup{
		EpicKey:       key,
		TotalFeatures: totalFeatures,
		StatusCounts:  statusCounts,
	}, nil
}

// GetTaskStatusRollup aggregates task statuses across all features in an epic.
func (s *EpicAnalyticsService) GetTaskStatusRollup(ctx context.Context, key string) (map[string]int, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	rollup, err := s.repo.GetTaskStatusRollup(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status rollup for epic %s: %w", key, err)
	}
	return rollup, nil
}

// GetImpediments returns blocked tasks that impede epic progress.
// Degrades gracefully if taskRepo is nil (returns empty slice).
func (s *EpicAnalyticsService) GetImpediments(ctx context.Context, key string) ([]*Impediment, error) {
	if s.taskRepo == nil {
		return []*Impediment{}, nil
	}

	blockedTasks, err := s.taskRepo.ListBlockedTasksByEpic(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get impediments for epic %s: %w", key, err)
	}

	impediments := make([]*Impediment, 0, len(blockedTasks))
	now := time.Now()
	for _, task := range blockedTasks {
		ageDays := 0
		if task.BlockedAt.Valid {
			ageDays = int(now.Sub(task.BlockedAt.Time).Hours() / 24)
		} else {
			ageDays = int(now.Sub(task.UpdatedAt).Hours() / 24)
		}
		impediments = append(impediments, &Impediment{
			TaskKey:  task.Key,
			Title:    task.Title,
			Status:   string(task.Status),
			Priority: task.Priority,
			AgeDays:  ageDays,
		})
	}

	return impediments, nil
}

// GetBlockedTasks returns the raw blocked tasks that impede epic progress.
// Unlike GetImpediments which returns DTO objects, this returns the full model objects.
// Degrades gracefully if taskRepo is nil (returns empty slice).
func (s *EpicAnalyticsService) GetBlockedTasks(ctx context.Context, key string) ([]*models.Task, error) {
	if s.taskRepo == nil {
		return []*models.Task{}, nil
	}
	blockedTasks, err := s.taskRepo.ListBlockedTasksByEpic(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked tasks for epic %s: %w", key, err)
	}
	return blockedTasks, nil
}

// GetHealth analyzes the health of an epic based on blocked tasks and feature status.
// Degrades gracefully if taskRepo is nil (returns healthy).
func (s *EpicAnalyticsService) GetHealth(ctx context.Context, key string) (*EpicHealthInfo, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	health := &EpicHealthInfo{
		EpicKey: key,
		Status:  "healthy",
	}

	// Check for blocked tasks
	impediments, err := s.GetImpediments(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze health for epic %s: %w", key, err)
	}

	if len(impediments) >= 2 {
		health.Status = "critical"
		health.Reasons = append(health.Reasons, fmt.Sprintf("%d blocked tasks", len(impediments)))
	} else if len(impediments) == 1 {
		health.Status = "warning"
		health.Reasons = append(health.Reasons, "1 blocked task")
	}

	// Check for high-priority blocked tasks
	for _, imp := range impediments {
		if imp.Priority <= 3 && health.Status != "critical" {
			health.Status = "critical"
			health.Reasons = append(health.Reasons, fmt.Sprintf("high-priority task %s is blocked", imp.TaskKey))
		}
	}

	return health, nil
}

// GetEpicDisplayData fetches all data needed to display an epic in a single SQL query
// via the epic_display_data view. This reduces round-trips from ~8 to 1, critical for
// Turso cloud databases where each round-trip costs ~150-200ms.
func (s *EpicAnalyticsService) GetEpicDisplayData(ctx context.Context, epic *models.Epic, projectRoot string) (*EpicDisplayData, error) {
	result := &EpicDisplayData{
		Epic:              epic,
		FeatureTaskCounts: make(map[int64]int),
		StatusBreakdowns:  make(map[int64]map[models.TaskStatus]int),
		FeatureRollup:     make(map[string]int),
		TaskStatusRollup:  make(map[string]int),
		BlockedTasks:      make([]*models.Task, 0),
		RelatedDocs:       make([]*models.Document, 0),
		Notes:             make([]*models.EntityNote, 0),
	}

	// Resolve epic path without re-fetching the epic (non-fetching version)
	result.RelPath = resolveEpicPathFromLoaded(epic, projectRoot)

	// Single query via the epic_display_data view — 1 round-trip
	raw, err := s.repo.GetEpicDisplayDataRaw(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get display data for epic %s: %w", epic.Key, err)
	}

	// Unmarshal features
	featuresRaw, err := unmarshalJSONArray[featureJSON](raw.FeaturesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal features for epic %s: %w", epic.Key, err)
	}
	features := make([]*models.Feature, 0, len(featuresRaw))
	for _, fj := range featuresRaw {
		statusOverride := fj.StatusOverride != 0
		f := &models.Feature{BaseEntity: models.BaseEntity{ID: fj.ID,
			Key:         fj.Key,
			Title:       fj.Title,
			Slug:        fj.Slug,
			Description: fj.Description,

			FilePath:    fj.FilePath,
			ContextData: fj.ContextData}, Status: models.FeatureStatus(fj.Status),
			StatusOverride: statusOverride,
			ProgressPct:    fj.ProgressPct,
			EpicID:         epic.ID,
		}
		if fj.ExecutionOrder != nil {
			f.ExecutionOrder = fj.ExecutionOrder
		}
		features = append(features, f)
	}
	result.Features = features

	// Unmarshal task status breakdowns
	breakdownsRaw, err := unmarshalJSONArray[taskBreakdownJSON](raw.TaskBreakdownJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal task breakdowns for epic %s: %w", epic.Key, err)
	}
	for _, b := range breakdownsRaw {
		if _, ok := result.StatusBreakdowns[b.FeatureID]; !ok {
			result.StatusBreakdowns[b.FeatureID] = make(map[models.TaskStatus]int)
		}
		result.StatusBreakdowns[b.FeatureID][models.TaskStatus(b.Status)] = b.Count
	}

	// Derive task counts per feature from breakdowns (no separate query needed)
	for featureID, breakdown := range result.StatusBreakdowns {
		total := 0
		for _, count := range breakdown {
			total += count
		}
		result.FeatureTaskCounts[featureID] = total
	}

	// Unmarshal blocked tasks
	blockedRaw, err := unmarshalJSONArray[blockedTaskJSON](raw.BlockedTasksJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal blocked tasks for epic %s: %w", epic.Key, err)
	}
	for _, bt := range blockedRaw {
		task := &models.Task{BaseEntity: models.BaseEntity{ID: bt.ID,

			Key:   bt.Key,
			Title: bt.Title}, FeatureID: bt.FeatureID,

			Status:        models.TaskStatus(bt.Status),
			BlockedReason: bt.BlockedReason,
		}
		result.BlockedTasks = append(result.BlockedTasks, task)
	}

	// Unmarshal related documents
	docsRaw, err := unmarshalJSONArray[documentJSON](raw.DocumentsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal documents for epic %s: %w", epic.Key, err)
	}
	for _, d := range docsRaw {
		doc := &models.Document{
			ID:       d.ID,
			Title:    d.Title,
			FilePath: d.FilePath,
		}
		result.RelatedDocs = append(result.RelatedDocs, doc)
	}

	// Unmarshal notes
	notesRaw, err := unmarshalJSONArray[noteJSON](raw.NotesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal notes for epic %s: %w", epic.Key, err)
	}
	for _, n := range notesRaw {
		note := &models.EntityNote{
			ID:         n.ID,
			EntityType: models.EntityTypeEpic,
			EntityID:   epic.ID,
			NoteType:   models.NoteType(n.NoteType),
			Content:    n.Content,
			CreatedBy:  n.CreatedBy,
			Metadata:   n.Metadata,
		}
		result.Notes = append(result.Notes, note)
	}

	// Compute feature rollup in-memory
	for _, f := range features {
		result.FeatureRollup[string(f.Status)]++
	}

	// Compute task status rollup in-memory
	for _, breakdown := range result.StatusBreakdowns {
		for status, count := range breakdown {
			result.TaskStatusRollup[string(status)] += count
		}
	}

	// Compute epic progress from stored feature ProgressPct values
	if len(features) > 0 {
		var totalProgress float64
		for _, f := range features {
			if status.IsFeatureCompletedStatus(f.Status) {
				totalProgress += 100.0
			} else {
				totalProgress += f.ProgressPct
			}
		}
		result.Progress = math.Round(totalProgress/float64(len(features))*100) / 100
	}

	return result, nil
}

// EpicDisplayData bundles all data needed to display an epic's details.
// Fetched via a single SQL view query (epic_display_data) for optimal Turso performance.
type EpicDisplayData struct {
	Epic              *models.Epic
	Features          []*models.Feature
	FeatureTaskCounts map[int64]int
	StatusBreakdowns  map[int64]map[models.TaskStatus]int
	FeatureRollup     map[string]int
	TaskStatusRollup  map[string]int
	BlockedTasks      []*models.Task
	RelatedDocs       []*models.Document
	Notes             []*models.EntityNote
	Progress          float64
	RelPath           string
}

// JSON helper types for unmarshaling the epic_display_data view columns.

type featureJSON struct {
	ID             int64   `json:"id"`
	Key            string  `json:"key"`
	Title          string  `json:"title"`
	Slug           *string `json:"slug"`
	Description    *string `json:"description"`
	Status         string  `json:"status"`
	StatusOverride int     `json:"status_override"`
	ProgressPct    float64 `json:"progress_pct"`
	ExecutionOrder *int    `json:"execution_order"`
	FilePath       *string `json:"file_path"`
	ContextData    *string `json:"context_data"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type taskBreakdownJSON struct {
	FeatureID int64  `json:"feature_id"`
	Status    string `json:"status"`
	Count     int    `json:"cnt"`
}

type blockedTaskJSON struct {
	ID            int64   `json:"id"`
	FeatureID     int64   `json:"feature_id"`
	Key           string  `json:"key"`
	Title         string  `json:"title"`
	Status        string  `json:"status"`
	BlockedReason *string `json:"blocked_reason"`
	BlockedAt     *string `json:"blocked_at"`
}

type documentJSON struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	FilePath string `json:"file_path"`
}

type noteJSON struct {
	ID        int64   `json:"id"`
	NoteType  string  `json:"note_type"`
	Content   string  `json:"content"`
	CreatedBy *string `json:"created_by"`
	Metadata  *string `json:"metadata"`
	CreatedAt string  `json:"created_at"`
}

// unmarshalJSONArray is a generic helper to unmarshal a JSON array string into a slice of T.
func unmarshalJSONArray[T any](raw string) ([]T, error) {
	if raw == "" || raw == "null" || raw == "[]" {
		return []T{}, nil
	}
	var result []T
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// deriveEpicStatusFromFeatures determines the appropriate epic status based on
// the breakdown of child feature statuses. This mirrors the logic used by the
// status.CalculationService but operates without that package dependency.
//
// Rules:
//   - No features -> keep current status
//   - Non-cascade parent statuses -> keep current status
//   - Cascade parent statuses with any non-terminal child feature -> keep current status
//   - Cascade parent statuses with all child features terminal -> advance one configured step
func deriveEpicStatusFromFeatures(featureCounts map[models.FeatureStatus]int, current models.EpicStatus, workflowSvc *workflow.Service) models.EpicStatus {
	total := 0
	for _, count := range featureCounts {
		total += count
	}
	if total == 0 {
		return current
	}

	if workflowSvc == nil {
		return current
	}

	epicWorkflow := workflowSvc.ForLevel(workflow.LevelEpic)
	if !epicWorkflow.HasOrchestratorAction(string(current), config.ActionCascade) {
		return current
	}

	featureWorkflow := workflowSvc.ForLevel(workflow.LevelFeature)
	terminalChildren := 0
	for status, count := range featureCounts {
		if featureWorkflow.IsTerminalStatus(string(status)) {
			terminalChildren += count
		}
	}
	if terminalChildren != total {
		return current
	}

	if nextStatus, ok := epicWorkflow.GetSingleNextStatus(string(current)); ok {
		return models.EpicStatus(nextStatus)
	}

	return current
}

// resolveEpicPathFromLoaded returns the relative file path for an epic without
// re-fetching the epic from the database. This is used by GetEpicDisplayData
// to avoid the extra query that ResolveEpicPath performs.
func resolveEpicPathFromLoaded(epic *models.Epic, projectRoot string) string {
	if epic.FilePath != nil && *epic.FilePath != "" {
		return *epic.FilePath
	}

	slug := ""
	if epic.Slug != nil && *epic.Slug != "" {
		slug = *epic.Slug
	} else {
		slug = strings.ToLower(epic.Key)
	}

	return filepath.Join("docs", "plan", epic.Key+"-"+slug, "epic.md")
}
