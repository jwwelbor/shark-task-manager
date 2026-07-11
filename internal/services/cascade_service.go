// Package services — CascadeService enumerates dispatchable children for
// `shark next` cascade resolution.
//
// Terminal-status filtering delegates to workflow.Service.IsTerminalStatus
// so custom workflows that rename the terminal status (e.g. "shipped"
// instead of "completed") continue to filter correctly. Repository query
// order is passed through untouched.
package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// CascadeChild is a (key, entityType) pair the cascade resolver hands back to
// the CLI for recursion.
type CascadeChild struct {
	Key        string
	EntityType models.EntityType
}

// CascadeChildrenState describes the children under a cascade parent at the
// moment dispatch is evaluated.
//
// Children contains the ordered subset that is currently dispatchable:
// non-terminal children whose dependencies are satisfied. TotalChildren and
// NonTerminalChildren retain the broader classification so callers can
// distinguish "nothing ready right now" from "all child work is finished".
type CascadeChildrenState struct {
	Children            []CascadeChild
	TotalChildren       int
	NonTerminalChildren int
}

// CascadeTaskRepo is the narrow task repository interface the cascade service
// needs to enumerate tasks under a feature.
type CascadeTaskRepo interface {
	ListByFeatureKey(ctx context.Context, featureKey string) ([]*models.Task, error)
	GetTaskDependencies(ctx context.Context, taskKey string) ([]*models.Task, error)
}

// CascadeEpicLookup is the narrow epic repository interface the cascade
// service needs to resolve an epic key to its ID before listing features.
type CascadeEpicLookup interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
}

// CascadeFeatureLister is the narrow feature repository interface the cascade
// service needs to enumerate features under an epic.
type CascadeFeatureLister interface {
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
}

// CascadeWorkflowProvider is the narrow workflow interface the cascade service
// needs to scope terminal-status checks per entity level.
//
// Satisfied by *workflow.Service in production. Defined here so service tests
// can swap in a fake workflow without depending on the concrete workflow
// package.
type CascadeWorkflowProvider interface {
	ForLevel(level string) *workflow.Service
}

// CascadeService enumerates dispatchable children for cascade resolution.
//
// This is a thin orchestration service: it owns no transactions and writes no
// state. Its single responsibility is "given a parent (entityType, key),
// return the ordered, terminal-filtered list of children to recurse into."
type CascadeService struct {
	taskRepo    CascadeTaskRepo
	epicRepo    CascadeEpicLookup
	featureRepo CascadeFeatureLister
	workflowSvc CascadeWorkflowProvider
}

// NewCascadeService constructs a CascadeService.
//
// All four dependencies are required; passing nil for any of them will cause
// DescribeDispatchableChildren to panic the first time the corresponding branch
// is exercised. This is a deliberate fail-fast posture for a service that
// must produce correct dispatch decisions.
func NewCascadeService(
	taskRepo CascadeTaskRepo,
	epicRepo CascadeEpicLookup,
	featureRepo CascadeFeatureLister,
	workflowSvc CascadeWorkflowProvider,
) *CascadeService {
	return &CascadeService{
		taskRepo:    taskRepo,
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		workflowSvc: workflowSvc,
	}
}

// DescribeDispatchableChildren returns the ordered list of currently
// dispatchable children plus summary counts that let callers tell whether a
// cascade parent is truly finished versus merely waiting.
//
// Behavior:
//
//   - entityType == "feature": list tasks for the feature key, filter out any
//     whose status is terminal under the task-level workflow.
//   - entityType == "epic": resolve epic key → ID, list features under it,
//     filter out any whose status is terminal under the feature-level workflow.
//   - any other entityType (task, bug, change, tech-debt, …): no children —
//     return (nil, nil). The caller treats this as "no dispatchable child →
//     pause".
//
// Repository query order is preserved untouched so the caller picks "first
// in-progress task; else first todo by order" simply by iterating.
func (s *CascadeService) DescribeDispatchableChildren(ctx context.Context, entityType, key string) (CascadeChildrenState, error) {
	switch entityType {
	case "feature":
		tasks, err := s.taskRepo.ListByFeatureKey(ctx, key)
		if err != nil {
			return CascadeChildrenState{}, fmt.Errorf("failed to list tasks for feature %s: %w", key, err)
		}
		taskWf := s.workflowSvc.ForLevel(workflow.LevelTask)
		out := make([]CascadeChild, 0, len(tasks))
		nonTerminal := 0
		for _, t := range tasks {
			if s.isTerminalStatus(taskWf, string(t.Status)) {
				continue
			}
			nonTerminal++
			ready, err := s.dependenciesSatisfied(ctx, t.Key)
			if err != nil {
				return CascadeChildrenState{}, err
			}
			if !ready {
				continue
			}
			out = append(out, CascadeChild{Key: t.Key, EntityType: models.EntityTypeTask})
		}
		return CascadeChildrenState{
			Children:            out,
			TotalChildren:       len(tasks),
			NonTerminalChildren: nonTerminal,
		}, nil

	case "epic":
		epic, err := s.epicRepo.GetByKey(ctx, key)
		if err != nil {
			return CascadeChildrenState{}, fmt.Errorf("failed to get epic %s: %w", key, err)
		}
		features, err := s.featureRepo.ListByEpic(ctx, epic.ID)
		if err != nil {
			return CascadeChildrenState{}, fmt.Errorf("failed to list features for epic %s: %w", key, err)
		}
		featureWf := s.workflowSvc.ForLevel(workflow.LevelFeature)
		out := make([]CascadeChild, 0, len(features))
		nonTerminal := 0
		for _, f := range features {
			if s.isTerminalStatus(featureWf, string(f.Status)) {
				continue
			}
			nonTerminal++
			out = append(out, CascadeChild{Key: f.Key, EntityType: models.EntityTypeFeature})
		}
		return CascadeChildrenState{
			Children:            out,
			TotalChildren:       len(features),
			NonTerminalChildren: nonTerminal,
		}, nil
	}

	// Leaf entities (task, bug, change-card, tech-debt) have no children.
	// Returning (nil, nil) — not an error — lets the caller fall through to
	// "no dispatchable child → pause".
	return CascadeChildrenState{}, nil
}


// isTerminalStatus reports whether a status is terminal (no productive
// dispatch possible) for the given workflow level. Delegates to
// workflow.Service.IsTerminalStatus, which reads the configured terminal set
// from the per-level workflow YAML (special_statuses._complete_). Using the
// workflow service rather than a hardcoded literal list keeps cascade
// correctness aligned with custom workflows that rename terminal statuses
// (e.g. "shipped" instead of "completed"). See B028.
//
// A nil workflow is treated as "no terminal classification available" rather
// than panicking, mirroring the helper's previous behavior in cascade.go.
func (s *CascadeService) isTerminalStatus(wf *workflow.Service, status string) bool {
	if wf == nil {
		return false
	}
	return wf.IsTerminalStatus(status)
}

func (s *CascadeService) dependenciesSatisfied(ctx context.Context, taskKey string) (bool, error) {
	dependencies, err := s.taskRepo.GetTaskDependencies(ctx, taskKey)
	if err != nil {
		return false, fmt.Errorf("failed to list dependencies for task %s: %w", taskKey, err)
	}
	for _, dep := range dependencies {
		if dep.Status != models.TaskStatus("completed") && dep.Status != models.TaskStatus("archived") {
			return false, nil
		}
	}
	return true, nil
}
