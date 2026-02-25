package services

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
}

// ResumeEntityNoteRepository defines the entity note repository interface needed by ResumeService.
type ResumeEntityNoteRepository interface {
	GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

// ResumeSessionStats contains aggregated work session statistics for task resume context.
type ResumeSessionStats struct {
	TotalSessions   int
	TotalDuration   time.Duration
	AverageDuration time.Duration
	ActiveSession   bool
}

// ResumeWorkSessionRepository defines the work session repository interface needed by ResumeService.
type ResumeWorkSessionRepository interface {
	GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error)
	GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*ResumeSessionStats, error)
	GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error)
}

// TaskResumeContext aggregates all context needed to resume work on a task.
type TaskResumeContext struct {
	Task           *models.Task               `json:"task"`
	ContextData    *models.ContextData        `json:"context_data,omitempty"`
	Notes          []*models.EntityNote       `json:"notes,omitempty"`
	WorkSessions   []*models.WorkSession      `json:"work_sessions,omitempty"`
	SessionStats   *ResumeSessionStats        `json:"session_stats,omitempty"`
	ActiveSession  *models.WorkSession        `json:"active_session,omitempty"`
	Dependencies   []string                   `json:"dependencies,omitempty"`
	CompletionMeta *models.CompletionMetadata `json:"completion_metadata,omitempty"`
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
	sessionRepo ResumeWorkSessionRepository
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

// SetSessionRepo sets the optional work session repository for task resume support.
func (s *ResumeService) SetSessionRepo(repo ResumeWorkSessionRepository) {
	s.sessionRepo = repo
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

// GetTaskResume aggregates all context needed to resume work on a task.
func (s *ResumeService) GetTaskResume(ctx context.Context, taskKey string) (*TaskResumeContext, error) {
	task, err := s.taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %s: %w", taskKey, err)
	}

	resumeCtx := &TaskResumeContext{
		Task: task,
	}

	// Parse context data from task field
	if task.ContextData != nil && *task.ContextData != "" && *task.ContextData != "{}" {
		contextData, parseErr := models.FromJSON(*task.ContextData)
		if parseErr == nil {
			resumeCtx.ContextData = contextData
		}
	}

	// Get notes
	notes, err := s.noteRepo.GetByEntity(ctx, models.EntityTypeTask, task.ID)
	if err == nil {
		resumeCtx.Notes = notes
	}

	// Get work sessions and statistics (optional - degrade gracefully if sessionRepo is nil)
	if s.sessionRepo != nil {
		sessions, sessErr := s.sessionRepo.GetByTaskID(ctx, task.ID)
		if sessErr == nil {
			resumeCtx.WorkSessions = sessions
		}

		stats, statsErr := s.sessionRepo.GetSessionStatsByTaskID(ctx, task.ID)
		if statsErr == nil {
			resumeCtx.SessionStats = stats
		}

		activeSession, activeErr := s.sessionRepo.GetActiveSessionByTaskID(ctx, task.ID)
		if activeErr == nil && activeSession != nil {
			resumeCtx.ActiveSession = activeSession
		}
	}

	// Parse dependencies from task field
	if task.DependsOn != nil && *task.DependsOn != "" {
		deps := strings.Split(strings.Trim(*task.DependsOn, "[]"), ",")
		parsed := make([]string, 0, len(deps))
		for _, dep := range deps {
			dep = strings.Trim(strings.Trim(dep, "\""), " ")
			if dep != "" {
				parsed = append(parsed, dep)
			}
		}
		resumeCtx.Dependencies = parsed
	}

	// Build completion metadata if task is completed or ready for review
	if task.Status == models.TaskStatus("completed") || task.Status == models.TaskStatus("ready_for_review") {
		completionMeta := &models.CompletionMetadata{
			CompletedBy:      task.CompletedBy,
			CompletionNotes:  task.CompletionNotes,
			TestsPassed:      task.TestsPassed,
			TimeSpentMinutes: task.TimeSpentMinutes,
		}
		if task.VerificationStatus != nil {
			completionMeta.VerificationStatus = *task.VerificationStatus
		}
		if task.FilesChanged != nil && *task.FilesChanged != "" {
			if parseErr := completionMeta.FromJSON(*task.FilesChanged); parseErr == nil {
				resumeCtx.CompletionMeta = completionMeta
			}
		} else {
			resumeCtx.CompletionMeta = completionMeta
		}
	}

	return resumeCtx, nil
}
