package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// TaskQueryRepository defines the repository interface needed by TaskQueryService.
// This is a subset of TaskRepository focused on query operations.
type TaskQueryRepository interface {
	List(ctx context.Context) ([]*models.Task, error)
	ListByFeatureKey(ctx context.Context, featureKey string) ([]*models.Task, error)
	ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
	FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error)
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
	GetTaskDisplayDataRaw(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error)
}

// TaskQueryService handles list, filter, search, and display-data queries for tasks.
// It is a focused sub-service extracted from TaskService to reduce its size.
type TaskQueryService struct {
	repo TaskQueryRepository
}

// NewTaskQueryService creates a new TaskQueryService.
//
// Parameters:
//   - repo: task repository for query operations (required, panics if nil)
func NewTaskQueryService(repo TaskQueryRepository) *TaskQueryService {
	requireNonNil(repo, "TaskQueryService requires a non-nil TaskQueryRepository")
	return &TaskQueryService{repo: repo}
}

// ListTasks retrieves tasks matching the given filters.
func (s *TaskQueryService) ListTasks(ctx context.Context, filters TaskFilters) ([]*models.Task, error) {
	var tasks []*models.Task
	var err error

	switch {
	case filters.FeatureKey != "":
		tasks, err = s.repo.ListByFeatureKey(ctx, filters.FeatureKey)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for feature %s: %w", filters.FeatureKey, err)
		}
	case filters.EpicKey != "":
		tasks, err = s.repo.ListByEpic(ctx, filters.EpicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for epic %s: %w", filters.EpicKey, err)
		}
	default:
		tasks, err = s.repo.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}
	}

	// Apply filters
	var filtered []*models.Task
	for _, task := range tasks {
		if filters.Status != "" && string(task.Status) != filters.Status {
			continue
		}

		if filters.AgentType != "" {
			if task.AgentType == nil || *task.AgentType != filters.AgentType {
				continue
			}
		}

		if filters.Blocked && string(task.Status) != "blocked" {
			continue
		}

		if !filters.ShowAll && string(task.Status) == "completed" {
			continue
		}

		if filters.TitleSearch != "" {
			if !strings.Contains(strings.ToLower(task.Title), strings.ToLower(filters.TitleSearch)) {
				continue
			}
		}

		if filters.MinPriority > 0 && task.Priority < filters.MinPriority {
			continue
		}
		if filters.MaxPriority > 0 && task.Priority > filters.MaxPriority {
			continue
		}

		filtered = append(filtered, task)
	}

	sortTasks(filtered)

	return filtered, nil
}

// ListTasksWithPagination retrieves a paginated list of tasks matching the given filters.
func (s *TaskQueryService) ListTasksWithPagination(ctx context.Context, filters TaskFilters) ([]*models.Task, int, error) {
	allTasks, err := s.ListTasks(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	total := len(allTasks)

	start := filters.Offset
	if start > total {
		return []*models.Task{}, total, nil
	}

	if filters.Limit == 0 {
		return allTasks[start:], total, nil
	}

	end := start + filters.Limit
	if end > total {
		end = total
	}

	return allTasks[start:end], total, nil
}

// GetTasksByStatus groups tasks by status and returns count per status.
func (s *TaskQueryService) GetTasksByStatus(ctx context.Context, filters TaskFilters) (map[string]int, error) {
	filters.ShowAll = true

	tasks, err := s.ListTasks(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by status: %w", err)
	}

	statusMap := make(map[string]int)
	for _, task := range tasks {
		statusMap[string(task.Status)]++
	}

	return statusMap, nil
}

// GetTasksByAgent groups tasks by agent type and returns count per agent.
func (s *TaskQueryService) GetTasksByAgent(ctx context.Context, filters TaskFilters) (map[string]int, error) {
	tasks, err := s.ListTasks(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by agent: %w", err)
	}

	agentMap := make(map[string]int)
	for _, task := range tasks {
		if task.AgentType != nil {
			agentMap[*task.AgentType]++
		}
	}

	return agentMap, nil
}

// GetBlockedTasks returns all blocked tasks matching the given filters.
func (s *TaskQueryService) GetBlockedTasks(ctx context.Context, filters TaskFilters) ([]*models.Task, error) {
	filters.Blocked = true
	filters.ShowAll = true

	tasks, err := s.ListTasks(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked tasks: %w", err)
	}

	return tasks, nil
}

// SearchByFile finds tasks that reference or are related to the given file path.
func (s *TaskQueryService) SearchByFile(ctx context.Context, filePath string, filters TaskFilters) ([]*models.Task, error) {
	tasks, err := s.repo.FindByFileChanged(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks by file %s: %w", filePath, err)
	}

	if filters.Status != "" || filters.EpicKey != "" || filters.FeatureKey != "" {
		filtered := make([]*models.Task, 0, len(tasks))
		epicKeyUpper := strings.ToUpper(filters.EpicKey)
		featureKeyUpper := strings.ToUpper(filters.FeatureKey)
		for _, t := range tasks {
			if filters.Status != "" && string(t.Status) != filters.Status {
				continue
			}
			if epicKeyUpper != "" && !strings.Contains(strings.ToUpper(t.Key), epicKeyUpper) {
				continue
			}
			if featureKeyUpper != "" && !strings.Contains(strings.ToUpper(t.Key), featureKeyUpper) {
				continue
			}
			filtered = append(filtered, t)
		}
		return filtered, nil
	}

	return tasks, nil
}

// GetTaskDisplayData fetches all data needed to display a task in a single SQL query
// via the task_display_data view. This reduces round-trips from ~5 to 1, critical for
// Turso cloud databases where each round-trip costs ~150-200ms.
func (s *TaskQueryService) GetTaskDisplayData(ctx context.Context, task *models.Task) (*TaskDisplayData, error) {
	result := &TaskDisplayData{
		BlockedBy:     make([]RelationshipWithTask, 0),
		Blocks:        make([]RelationshipWithTask, 0),
		Relationships: make([]RelationshipWithTask, 0),
		Dependencies:  make([]*models.Task, 0),
		RelatedDocs:   make([]*models.Document, 0),
		Notes:         make([]*models.EntityNote, 0),
	}

	raw, err := s.repo.GetTaskDisplayDataRaw(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get display data for task %s: %w", task.Key, err)
	}

	blockedByRaw, err := unmarshalJSONArray[RelationshipWithTask](raw.BlockedByJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal blocked_by for task %s: %w", task.Key, err)
	}
	result.BlockedBy = blockedByRaw

	blocksRaw, err := unmarshalJSONArray[RelationshipWithTask](raw.BlocksJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal blocks for task %s: %w", task.Key, err)
	}
	result.Blocks = blocksRaw

	relationshipsRaw, err := unmarshalJSONArray[RelationshipWithTask](raw.RelationshipsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal relationships for task %s: %w", task.Key, err)
	}
	result.Relationships = relationshipsRaw

	depsRaw, err := unmarshalJSONArray[taskDependencyJSON](raw.DependenciesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dependencies for task %s: %w", task.Key, err)
	}
	for _, d := range depsRaw {
		result.Dependencies = append(result.Dependencies, &models.Task{BaseEntity: models.BaseEntity{Key: d.Key,
			Title: d.Title}, Status: models.TaskStatus(d.Status),
		})
	}

	docsRaw, err := unmarshalJSONArray[documentJSON](raw.DocumentsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal documents for task %s: %w", task.Key, err)
	}
	for _, d := range docsRaw {
		result.RelatedDocs = append(result.RelatedDocs, &models.Document{
			ID:       d.ID,
			Title:    d.Title,
			FilePath: d.FilePath,
		})
	}

	notesRaw, err := unmarshalJSONArray[noteJSON](raw.NotesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal notes for task %s: %w", task.Key, err)
	}
	for _, n := range notesRaw {
		result.Notes = append(result.Notes, &models.EntityNote{
			ID:         n.ID,
			EntityType: models.EntityTypeTask,
			EntityID:   task.ID,
			NoteType:   models.NoteType(n.NoteType),
			Content:    n.Content,
			CreatedBy:  n.CreatedBy,
			Metadata:   n.Metadata,
		})
	}

	return result, nil
}
