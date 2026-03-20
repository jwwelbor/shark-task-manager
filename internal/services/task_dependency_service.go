package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskDependencyTaskRepository defines the task repository methods needed by TaskDependencyService.
type TaskDependencyTaskRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
	GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error)
}

// TaskDependencyService handles dependency management, relationship operations,
// and related-document linking for tasks.
// It is a focused sub-service extracted from TaskService to reduce its size.
type TaskDependencyService struct {
	repo            TaskDependencyTaskRepository
	depRepo         TaskDependencyRepository
	relQueryRepo    TaskRelationshipQueryRepository
	writableDocRepo TaskWritableDocumentRepository
	docSvc          *EntityDocumentService // shared document operations; built by SetWritableDocRepo
}

// NewTaskDependencyService creates a new TaskDependencyService.
//
// Parameters:
//   - repo: task repository for task lookups (required, panics if nil)
func NewTaskDependencyService(repo TaskDependencyTaskRepository) *TaskDependencyService {
	requireNonNil(repo, "TaskDependencyService requires a non-nil TaskDependencyTaskRepository")
	return &TaskDependencyService{repo: repo}
}

// SetDepRepo sets the dependency repository for Add/Remove/List dependency operations.
func (s *TaskDependencyService) SetDepRepo(depRepo TaskDependencyRepository) {
	s.depRepo = depRepo
}

// SetRelQueryRepo sets the relationship query repository for rich relationship queries.
func (s *TaskDependencyService) SetRelQueryRepo(relQueryRepo TaskRelationshipQueryRepository) {
	s.relQueryRepo = relQueryRepo
}

// SetWritableDocRepo sets the writable document repository for link/unlink operations.
func (s *TaskDependencyService) SetWritableDocRepo(writableDocRepo TaskWritableDocumentRepository) {
	s.writableDocRepo = writableDocRepo
	s.docSvc = NewEntityDocumentService(
		writableDocRepo,
		models.EntityTypeTask,
		writableDocRepo.LinkToTask,
		writableDocRepo.UnlinkFromTask,
		nil, // list is handled by TaskService via docRepo
		func(ctx context.Context, key string) (int64, error) {
			task, err := s.repo.GetByKey(ctx, key)
			if err != nil {
				return 0, err
			}
			return task.ID, nil
		},
	)
}

// ValidateDependencies checks if a task's dependencies are met for the given transition.
// Includes circular dependency detection using depth-first search.
func (s *TaskDependencyService) ValidateDependencies(ctx context.Context, key string, targetStatus string) error {
	dependents, err := s.repo.GetTaskDependents(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get dependencies for task %s: %w", key, err)
	}

	if len(dependents) == 0 {
		return nil
	}

	for _, dep := range dependents {
		if dep.Status != models.TaskStatus("completed") {
			return fmt.Errorf("dependency not met: task %s depends on %s which is in status %s (must be completed)", key, dep.Key, dep.Status)
		}
	}

	if err := s.detectCircularDependency(ctx, key, make(map[string]bool), make(map[string]bool)); err != nil {
		return fmt.Errorf("circular dependency detected: %w", err)
	}

	return nil
}

// detectCircularDependency uses depth-first search to detect cycles in the dependency graph.
func (s *TaskDependencyService) detectCircularDependency(ctx context.Context, taskKey string, visited map[string]bool, recStack map[string]bool) error {
	visited[taskKey] = true
	recStack[taskKey] = true

	dependents, err := s.repo.GetTaskDependents(ctx, taskKey)
	if err != nil {
		return err
	}

	for _, dep := range dependents {
		if !visited[dep.Key] {
			if err := s.detectCircularDependency(ctx, dep.Key, visited, recStack); err != nil {
				return err
			}
		} else if recStack[dep.Key] {
			return fmt.Errorf("cycle detected: %s → %s", taskKey, dep.Key)
		}
	}

	recStack[taskKey] = false
	return nil
}

// GetDependencyTree retrieves the full dependency tree for a task.
func (s *TaskDependencyService) GetDependencyTree(ctx context.Context, key string) (*DependencyTree, error) {
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency tree for task %s: %w", key, err)
	}

	tree := &DependencyTree{
		Task: &TaskNode{
			Key:         task.Key,
			Title:       task.Title,
			Status:      string(task.Status),
			Priority:    task.Priority,
			IsCompleted: task.Status == models.TaskStatus("completed"),
			IsBlocked:   task.Status == models.TaskStatus("blocked"),
			UpdatedAt:   task.UpdatedAt,
		},
		Dependencies: []*TaskNode{},
		Dependents:   []*TaskNode{},
		Blocked:      false,
		BlockedBy:    []string{},
		CanStart:     true,
		Depth:        0,
	}

	dependents, err := s.repo.GetTaskDependents(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	for _, dep := range dependents {
		depNode := &TaskNode{
			Key:         dep.Key,
			Title:       dep.Title,
			Status:      string(dep.Status),
			Priority:    dep.Priority,
			IsCompleted: dep.Status == models.TaskStatus("completed"),
			IsBlocked:   dep.Status == models.TaskStatus("blocked"),
			UpdatedAt:   dep.UpdatedAt,
		}
		tree.Dependencies = append(tree.Dependencies, depNode)

		if dep.Status != models.TaskStatus("completed") {
			tree.Blocked = true
			tree.BlockedBy = append(tree.BlockedBy, dep.Key)
			tree.CanStart = false
		}
	}

	return tree, nil
}

// AddDependency creates a dependency relationship between two tasks.
// The task identified by taskKey will depend on the task identified by depKey.
func (s *TaskDependencyService) AddDependency(ctx context.Context, taskKey, depKey string) error {
	if s.depRepo == nil {
		return fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	dep, err := s.repo.GetByKey(ctx, depKey)
	if err != nil {
		return fmt.Errorf("dependency task not found: %w", err)
	}

	if task.ID == dep.ID {
		return fmt.Errorf("task cannot depend on itself")
	}

	rel := &models.TaskRelationship{
		FromTaskID:       task.ID,
		ToTaskID:         dep.ID,
		RelationshipType: "depends_on",
	}

	if err := s.depRepo.Create(ctx, rel); err != nil {
		return fmt.Errorf("failed to add dependency from %s to %s: %w", taskKey, depKey, err)
	}

	return nil
}

// RemoveDependency removes a dependency relationship between two tasks.
func (s *TaskDependencyService) RemoveDependency(ctx context.Context, taskKey, depKey string) error {
	if s.depRepo == nil {
		return fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	dep, err := s.repo.GetByKey(ctx, depKey)
	if err != nil {
		return fmt.Errorf("dependency task not found: %w", err)
	}

	if err := s.depRepo.DeleteByTasksAndType(ctx, task.ID, dep.ID, "depends_on"); err != nil {
		return fmt.Errorf("failed to remove dependency from %s to %s: %w", taskKey, depKey, err)
	}

	return nil
}

// ListDependencies returns all tasks that the given task depends on.
func (s *TaskDependencyService) ListDependencies(ctx context.Context, taskKey string) ([]*models.Task, error) {
	if s.depRepo == nil {
		return nil, fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	rels, err := s.depRepo.GetOutgoing(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		return nil, fmt.Errorf("failed to list dependencies for %s: %w", taskKey, err)
	}

	tasks := make([]*models.Task, 0, len(rels))
	for _, rel := range rels {
		depTask, err := s.repo.GetByID(ctx, rel.ToTaskID)
		if err != nil {
			continue
		}
		tasks = append(tasks, depTask)
	}

	return tasks, nil
}

// UnlinkFile removes the typed relationship between two tasks.
func (s *TaskDependencyService) UnlinkFile(ctx context.Context, taskKey, relType, targetKey string) error {
	if s.depRepo == nil {
		return fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	target, err := s.repo.GetByKey(ctx, targetKey)
	if err != nil {
		return fmt.Errorf("target task not found: %w", err)
	}

	if err := s.depRepo.DeleteByTasksAndType(ctx, task.ID, target.ID, relType); err != nil {
		return fmt.Errorf("failed to unlink %s from %s (%s): %w", taskKey, targetKey, relType, err)
	}

	return nil
}

// UnlinkRelationships removes relationship links between tasks.
// If targetKeys is empty, all relationships of the given type are removed.
func (s *TaskDependencyService) UnlinkRelationships(ctx context.Context, taskKey, relType string, targetKeys []string) (int, error) {
	if s.depRepo == nil {
		return 0, fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return 0, fmt.Errorf("task not found: %w", err)
	}

	if len(targetKeys) == 0 {
		rels, err := s.depRepo.GetOutgoing(ctx, task.ID, []string{relType})
		if err != nil {
			return 0, fmt.Errorf("failed to get relationships for %s: %w", taskKey, err)
		}

		count := 0
		for _, rel := range rels {
			if err := s.depRepo.Delete(ctx, rel.ID); err != nil {
				return count, fmt.Errorf("failed to delete relationship %d: %w", rel.ID, err)
			}
			count++
		}
		return count, nil
	}

	count := 0
	for _, targetKey := range targetKeys {
		target, err := s.repo.GetByKey(ctx, targetKey)
		if err != nil {
			return count, fmt.Errorf("target task not found: %w", err)
		}

		if err := s.depRepo.DeleteByTasksAndType(ctx, task.ID, target.ID, relType); err != nil {
			return count, fmt.Errorf("failed to unlink %s from %s (%s): %w", taskKey, targetKey, relType, err)
		}
		count++
	}

	return count, nil
}

// CreateTypedRelationship creates a typed relationship between two tasks.
// For depends_on and blocks relationships, circular dependency detection is performed.
func (s *TaskDependencyService) CreateTypedRelationship(ctx context.Context, taskKey, targetKey, relType string) (*models.Task, error) {
	if s.depRepo == nil {
		return nil, fmt.Errorf("relationship repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task %s not found: %w", taskKey, err)
	}

	targetTask, err := s.repo.GetByKey(ctx, targetKey)
	if err != nil {
		return nil, fmt.Errorf("target task %s not found: %w", targetKey, err)
	}

	if relType == "depends_on" || relType == "blocks" {
		if err := s.depRepo.DetectCycle(ctx, task.ID, targetTask.ID, relType); err != nil {
			return nil, fmt.Errorf("circular dependency detected: %w", err)
		}
	}

	rel := &models.TaskRelationship{
		FromTaskID:       task.ID,
		ToTaskID:         targetTask.ID,
		RelationshipType: models.RelationshipType(relType),
	}

	if err := s.depRepo.Create(ctx, rel); err != nil {
		return nil, fmt.Errorf("failed to create %s relationship from %s to %s: %w", relType, taskKey, targetKey, err)
	}

	return targetTask, nil
}

// GetTaskRelationships retrieves all relationships for a task, optionally filtered by type.
func (s *TaskDependencyService) GetTaskRelationships(ctx context.Context, taskKey string, typeFilter []string) ([]RelationshipWithTask, error) {
	if s.relQueryRepo == nil {
		return nil, fmt.Errorf("relationship query repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	allRels, err := s.relQueryRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships for %s: %w", taskKey, err)
	}

	var result []RelationshipWithTask
	for _, rel := range allRels {
		if len(typeFilter) > 0 {
			found := false
			for _, ft := range typeFilter {
				if string(rel.RelationshipType) == ft {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		direction := "outgoing"
		relatedTaskID := rel.ToTaskID
		if rel.FromTaskID != task.ID {
			direction = "incoming"
			relatedTaskID = rel.FromTaskID
		}

		relatedTask, err := s.repo.GetByID(ctx, relatedTaskID)
		if err != nil {
			continue
		}

		result = append(result, RelationshipWithTask{
			RelationshipType: string(rel.RelationshipType),
			Direction:        direction,
			TaskKey:          relatedTask.Key,
			TaskTitle:        relatedTask.Title,
			TaskStatus:       string(relatedTask.Status),
		})
	}

	return result, nil
}

// GetTaskBlockedBy retrieves tasks that this task depends on (incoming dependencies).
func (s *TaskDependencyService) GetTaskBlockedBy(ctx context.Context, taskKey string) ([]RelationshipWithTask, error) {
	if s.relQueryRepo == nil {
		return nil, fmt.Errorf("relationship query repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	deps, err := s.relQueryRepo.GetOutgoing(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies for %s: %w", taskKey, err)
	}

	var result []RelationshipWithTask
	for _, rel := range deps {
		depTask, err := s.repo.GetByID(ctx, rel.ToTaskID)
		if err != nil {
			continue
		}
		result = append(result, RelationshipWithTask{
			RelationshipType: "depends_on",
			Direction:        "outgoing",
			TaskKey:          depTask.Key,
			TaskTitle:        depTask.Title,
			TaskStatus:       string(depTask.Status),
		})
	}

	return result, nil
}

// GetTaskBlocks retrieves tasks that depend on this task completing.
func (s *TaskDependencyService) GetTaskBlocks(ctx context.Context, taskKey string) ([]RelationshipWithTask, error) {
	if s.relQueryRepo == nil {
		return nil, fmt.Errorf("relationship query repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	incoming, err := s.relQueryRepo.GetIncoming(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming dependencies for %s: %w", taskKey, err)
	}

	outgoing, err := s.relQueryRepo.GetOutgoing(ctx, task.ID, []string{"blocks"})
	if err != nil {
		return nil, fmt.Errorf("failed to get explicit blocks for %s: %w", taskKey, err)
	}

	allBlocked := append(incoming, outgoing...)

	var result []RelationshipWithTask
	for _, rel := range allBlocked {
		var blockedTaskID int64
		var direction string
		if rel.FromTaskID != task.ID {
			blockedTaskID = rel.FromTaskID
			direction = "incoming"
		} else {
			blockedTaskID = rel.ToTaskID
			direction = "outgoing"
		}

		blockedTask, err := s.repo.GetByID(ctx, blockedTaskID)
		if err != nil {
			continue
		}
		result = append(result, RelationshipWithTask{
			RelationshipType: string(rel.RelationshipType),
			Direction:        direction,
			TaskKey:          blockedTask.Key,
			TaskTitle:        blockedTask.Title,
			TaskStatus:       string(blockedTask.Status),
		})
	}

	return result, nil
}

// LinkDocument links a document to a task, creating the document record if it doesn't exist.
// Delegates to the shared EntityDocumentService.
func (s *TaskDependencyService) LinkDocument(ctx context.Context, taskKey, title, path string) (*models.Document, error) {
	if s.docSvc == nil {
		return nil, fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.LinkDocumentByKey(ctx, taskKey, title, path)
}

// UnlinkDocument removes the link between a document and a task.
// This operation is idempotent: it succeeds even if the document is not linked.
// Delegates to the shared EntityDocumentService.
func (s *TaskDependencyService) UnlinkDocument(ctx context.Context, taskKey, title string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.UnlinkDocumentByKey(ctx, taskKey, title)
}
