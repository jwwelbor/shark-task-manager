// Package services — PlanHierarchyService enumerates direct children for
// one-level `shark plan <epic|feature>` selection.
//
// This is intentionally separate from CascadeService (used by keyed
// `shark next`/`shark run` cascade traversal): CascadeService's per-child
// query behavior is the exact 0e3f0103 contract those commands must keep
// unchanged, while PlanHierarchyService loads one hierarchy edge in a single
// set-oriented query and applies claim/dependency filtering in memory.
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	planhierarchyrepo "github.com/jwwelbor/shark-task-manager/internal/repository/planhierarchy"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// PlanHierarchyChild is the bounded direct-child evidence returned to
// one-level hierarchy planning.
type PlanHierarchyChild struct {
	Key            string
	Title          string
	Status         string
	EntityType     models.EntityType
	ExecutionOrder *int
	Priority       *int
}

// PlanHierarchyChildrenState describes the children under a planning parent
// at the moment planning is evaluated.
//
// Children contains the ordered subset that is currently claimable:
// non-terminal, unclaimed children whose hard dependencies are satisfied.
// TotalChildren and NonTerminalChildren retain the broader classification so
// callers can distinguish "nothing ready right now" from "all child work is
// finished".
type PlanHierarchyChildrenState struct {
	Children            []PlanHierarchyChild
	TotalChildren       int
	NonTerminalChildren int
}

// PlanHierarchySnapshotReader loads one direct hierarchy edge in one
// database query.
type PlanHierarchySnapshotReader interface {
	ReadDirectChildren(
		ctx context.Context,
		parentType, parentKey string,
		claimTTL time.Duration,
		evaluatedAt time.Time,
	) (planhierarchyrepo.Snapshot, error)
}

// PlanHierarchyClaimPolicy supplies the configured lease expiry policy
// without performing another database read.
type PlanHierarchyClaimPolicy interface {
	TTL() time.Duration
}

// PlanHierarchyWorkflowProvider is the narrow workflow interface the service
// needs to scope terminal-status checks per entity level.
type PlanHierarchyWorkflowProvider interface {
	ForLevel(level string) *workflow.Service
}

// PlanHierarchyService enumerates claimable direct children for one-level
// hierarchy planning. It owns no transactions and writes no state.
type PlanHierarchyService struct {
	snapshotReader PlanHierarchySnapshotReader
	workflowSvc    PlanHierarchyWorkflowProvider
	claimPolicy    PlanHierarchyClaimPolicy
}

// NewPlanHierarchyService constructs the production one-query hierarchy
// planning reader. Each DescribeChildren call performs one database query
// regardless of the number of direct children.
func NewPlanHierarchyService(
	snapshotReader PlanHierarchySnapshotReader,
	workflowSvc PlanHierarchyWorkflowProvider,
	claimPolicy PlanHierarchyClaimPolicy,
) *PlanHierarchyService {
	return &PlanHierarchyService{
		snapshotReader: snapshotReader,
		workflowSvc:    workflowSvc,
		claimPolicy:    claimPolicy,
	}
}

// DescribeChildren returns the ordered list of currently claimable direct
// children of (parentType, parentKey) plus summary counts that let callers
// tell whether a planning parent is truly finished versus merely waiting.
//
// Behavior:
//
//   - parentType == "epic": direct features, filtered to non-terminal,
//     unclaimed children.
//   - parentType == "feature": direct tasks, filtered to non-terminal,
//     unclaimed children whose hard dependencies are all terminal.
//   - any other parentType: no children — return (nil, nil). The caller
//     treats this as "no dispatchable child".
func (s *PlanHierarchyService) DescribeChildren(
	ctx context.Context,
	parentType, parentKey string,
) (PlanHierarchyChildrenState, error) {
	claimTTL := time.Duration(0)
	if s.claimPolicy != nil {
		claimTTL = s.claimPolicy.TTL()
	}
	snapshot, err := s.snapshotReader.ReadDirectChildren(
		ctx,
		parentType,
		parentKey,
		claimTTL,
		time.Now().UTC(),
	)
	if err != nil {
		return PlanHierarchyChildrenState{}, err
	}
	if !snapshot.ParentFound {
		return PlanHierarchyChildrenState{}, fmt.Errorf("%s %s not found", parentType, parentKey)
	}

	childWorkflow := s.workflowSvc.ForLevel(planHierarchyChildWorkflowLevel(parentType))
	children := make([]PlanHierarchyChild, 0, len(snapshot.Children))
	nonTerminal := 0
	for _, child := range snapshot.Children {
		if s.isTerminalStatus(childWorkflow, child.Status) {
			continue
		}
		nonTerminal++
		if child.Claimed || !planHierarchyDependenciesSatisfied(childWorkflow, child.Dependencies) {
			continue
		}
		children = append(children, PlanHierarchyChild{
			Key:            child.Key,
			Title:          child.Title,
			Status:         child.Status,
			EntityType:     child.EntityType,
			ExecutionOrder: child.ExecutionOrder,
			Priority:       child.Priority,
		})
	}
	return PlanHierarchyChildrenState{
		Children:            children,
		TotalChildren:       len(snapshot.Children),
		NonTerminalChildren: nonTerminal,
	}, nil
}

func planHierarchyChildWorkflowLevel(parentType string) string {
	switch parentType {
	case string(models.EntityTypeEpic):
		return workflow.LevelFeature
	case string(models.EntityTypeFeature):
		return workflow.LevelTask
	default:
		return ""
	}
}

func planHierarchyDependenciesSatisfied(
	childWorkflow *workflow.Service,
	dependencies []planhierarchyrepo.Dependency,
) bool {
	for _, dependency := range dependencies {
		if childWorkflow == nil || !childWorkflow.IsTerminalStatus(dependency.Status) {
			return false
		}
	}
	return true
}

// isTerminalStatus reports whether a status is terminal (no productive
// dispatch possible) for the given workflow level. A nil workflow is treated
// as "no terminal classification available" rather than panicking.
func (s *PlanHierarchyService) isTerminalStatus(wf *workflow.Service, status string) bool {
	if wf == nil {
		return false
	}
	return wf.IsTerminalStatus(status)
}
