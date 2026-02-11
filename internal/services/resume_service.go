package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ResumeEpicRepository defines the epic repository interface needed by ResumeService.
type ResumeEpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetContextData(ctx context.Context, epicID int64) (*string, error)
}

// ResumeFeatureRepository defines the feature repository interface needed by ResumeService.
type ResumeFeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetContextData(ctx context.Context, featureID int64) (*string, error)
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
}

// ResumeTaskRepository defines the task repository interface needed by ResumeService.
type ResumeTaskRepository interface {
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
}

// ResumeEntityNoteRepository defines the entity note repository interface needed by ResumeService.
type ResumeEntityNoteRepository interface {
	GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

// EpicResumeContext aggregates all context needed to resume work on an epic
type EpicResumeContext struct {
	Epic        *models.Epic         `json:"epic"`
	ContextData *models.ContextData  `json:"context_data,omitempty"`
	Notes       []*models.EntityNote `json:"notes,omitempty"`
	Features    []*FeatureSummary    `json:"features,omitempty"`
	TaskSummary *TaskRollup          `json:"task_summary,omitempty"`
}

// FeatureSummary provides a brief overview of a feature within an epic resume
type FeatureSummary struct {
	Key       string  `json:"key"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	TaskCount int     `json:"task_count"`
	Progress  float64 `json:"progress_pct"`
}

// TaskRollup provides aggregate task counts by status
type TaskRollup struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
}

// FeatureResumeContext aggregates all context needed to resume work on a feature
type FeatureResumeContext struct {
	Feature     *models.Feature      `json:"feature"`
	ContextData *models.ContextData  `json:"context_data,omitempty"`
	Notes       []*models.EntityNote `json:"notes,omitempty"`
	Tasks       []*TaskSummary       `json:"tasks,omitempty"`
	TaskSummary *TaskRollup          `json:"task_summary,omitempty"`
}

// TaskSummary provides a brief overview of a task within a feature resume
type TaskSummary struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// ResumeService provides context aggregation for resuming work on entities.
type ResumeService struct {
	epicRepo    ResumeEpicRepository
	featureRepo ResumeFeatureRepository
	taskRepo    ResumeTaskRepository
	noteRepo    ResumeEntityNoteRepository
}

// NewResumeService creates a new ResumeService with injected dependencies.
func NewResumeService(epicRepo ResumeEpicRepository, featureRepo ResumeFeatureRepository, taskRepo ResumeTaskRepository, noteRepo ResumeEntityNoteRepository) *ResumeService {
	return &ResumeService{
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
		noteRepo:    noteRepo,
	}
}

// GetEpicResume aggregates all context needed to resume work on an epic.
func (s *ResumeService) GetEpicResume(ctx context.Context, epicKey string) (*EpicResumeContext, error) {
	epic, err := s.epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("epic not found: %s: %w", epicKey, err)
	}

	resumeCtx := &EpicResumeContext{
		Epic: epic,
	}

	// Get context data
	contextJSON, err := s.epicRepo.GetContextData(ctx, epic.ID)
	if err == nil && contextJSON != nil && *contextJSON != "" && *contextJSON != "{}" {
		contextData, parseErr := models.FromJSON(*contextJSON)
		if parseErr == nil {
			resumeCtx.ContextData = contextData
		}
	}

	// Get notes
	notes, err := s.noteRepo.GetByEntity(ctx, models.EntityTypeEpic, epic.ID)
	if err == nil {
		resumeCtx.Notes = notes
	}

	// Get features with summaries
	features, err := s.featureRepo.ListByEpic(ctx, epic.ID)
	if err == nil {
		summaries := make([]*FeatureSummary, 0, len(features))
		for _, f := range features {
			tasks, taskErr := s.taskRepo.ListByFeature(ctx, f.ID)
			taskCount := 0
			if taskErr == nil {
				taskCount = len(tasks)
			}
			summaries = append(summaries, &FeatureSummary{
				Key:       f.Key,
				Title:     f.Title,
				Status:    string(f.Status),
				TaskCount: taskCount,
				Progress:  f.ProgressPct,
			})
		}
		resumeCtx.Features = summaries
	}

	// Build task rollup across all features
	allTasks, err := s.taskRepo.ListByEpic(ctx, epicKey)
	if err == nil {
		rollup := &TaskRollup{
			Total:    len(allTasks),
			ByStatus: make(map[string]int),
		}
		for _, t := range allTasks {
			rollup.ByStatus[string(t.Status)]++
		}
		resumeCtx.TaskSummary = rollup
	}

	return resumeCtx, nil
}

// GetFeatureResume aggregates all context needed to resume work on a feature.
func (s *ResumeService) GetFeatureResume(ctx context.Context, featureKey string) (*FeatureResumeContext, error) {
	feature, err := s.featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("feature not found: %s: %w", featureKey, err)
	}

	resumeCtx := &FeatureResumeContext{
		Feature: feature,
	}

	// Get context data
	contextJSON, err := s.featureRepo.GetContextData(ctx, feature.ID)
	if err == nil && contextJSON != nil && *contextJSON != "" && *contextJSON != "{}" {
		contextData, parseErr := models.FromJSON(*contextJSON)
		if parseErr == nil {
			resumeCtx.ContextData = contextData
		}
	}

	// Get notes
	notes, err := s.noteRepo.GetByEntity(ctx, models.EntityTypeFeature, feature.ID)
	if err == nil {
		resumeCtx.Notes = notes
	}

	// Get tasks with summaries
	tasks, err := s.taskRepo.ListByFeature(ctx, feature.ID)
	if err == nil {
		summaries := make([]*TaskSummary, 0, len(tasks))
		rollup := &TaskRollup{
			Total:    len(tasks),
			ByStatus: make(map[string]int),
		}
		for _, t := range tasks {
			summaries = append(summaries, &TaskSummary{
				Key:      t.Key,
				Title:    t.Title,
				Status:   string(t.Status),
				Priority: t.Priority,
			})
			rollup.ByStatus[string(t.Status)]++
		}
		resumeCtx.Tasks = summaries
		resumeCtx.TaskSummary = rollup
	}

	return resumeCtx, nil
}
