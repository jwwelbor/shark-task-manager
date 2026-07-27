// Package services — PlanHierarchyService enumerates direct children for
// one-level `shark plan <epic|feature>` selection and keyed `shark next`
// cascade traversal. It loads one hierarchy edge in a single set-oriented
// query and applies claim/dependency filtering in memory.
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	planhierarchyrepo "github.com/jwwelbor/shark-task-manager/internal/repository/planhierarchy"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
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

// PlanHierarchyEdge is one relationship endpoint of a plan candidate: the
// entity on the other side of the edge, its current status, and the
// relationship type that produced the edge.
//
// Status is reported raw. Terminal classification is deliberately left to the
// caller — a satisfied dependency is information a parallel-safety decision
// needs, so it must not be filtered out here.
type PlanHierarchyEdge struct {
	Key    string
	Status string
	Type   string
}

// PlanHierarchyWarningDanglingRelationship identifies a relationship row whose
// far endpoint no longer exists. The row is omitted from the edge buckets and
// reported as bounded evidence so callers cannot mistake the result for a
// complete relationship graph.
const PlanHierarchyWarningDanglingRelationship = "DANGLING_RELATIONSHIP"

// PlanHierarchyEdgeWarning describes one relationship row that could not be
// represented as an edge. It intentionally contains only stable row and
// endpoint identity; repository implementation details stay out of the wire
// contract.
type PlanHierarchyEdgeWarning struct {
	Code             string
	Direction        string
	RelationshipID   int64
	EndpointType     models.EntityType
	EndpointID       int64
	RelationshipType models.EntityRelationshipType
}

// PlanHierarchyEdges is one candidate's relationship neighbourhood, split by
// what the edge means for scheduling:
//
//   - DependsOn: work this candidate must wait for — outgoing depends_on plus
//     incoming blocks. This is the same pair StandaloneHardDependencyService
//     evaluates in HasUnresolvedHardDependency.
//   - Blocks: work waiting on this candidate — incoming depends_on plus
//     outgoing blocks, matching EntityRelationshipService.GetTaskBlocks.
//   - Links: every other relationship type in either direction (related_to,
//     follows, spawned_from, duplicates, references, linked_to) — advisory
//     context, never a scheduling constraint.
//
// Type is retained per edge because DependsOn and Blocks each aggregate two
// relationship types; the bucket says what the edge means, Type says how it
// was recorded.
type PlanHierarchyEdges struct {
	DependsOn []PlanHierarchyEdge
	Blocks    []PlanHierarchyEdge
	Links     []PlanHierarchyEdge
	Warnings  []PlanHierarchyEdgeWarning
}

// PlanHierarchyRelationshipReader is the entity-generic relationship surface
// used to load candidate edges. *EntityRelationshipService satisfies it.
type PlanHierarchyRelationshipReader interface {
	GetOutgoing(
		ctx context.Context,
		entityType models.EntityType,
		entityID int64,
		relTypes []models.EntityRelationshipType,
	) ([]*models.EntityRelationship, error)
	GetIncoming(
		ctx context.Context,
		entityType models.EntityType,
		entityID int64,
		relTypes []models.EntityRelationshipType,
	) ([]*models.EntityRelationship, error)
}

// PlanHierarchyEntityRegistry resolves an entity type to its repository so
// relationship endpoints of any entity type can be turned into key + status.
// *EntityRegistry satisfies it.
type PlanHierarchyEntityRegistry interface {
	GetRepository(entityType models.EntityType) (EntityRepository, error)
}

// PlanHierarchyTaskDependencyReader supplies the task-only legacy
// tasks.depends_on JSON edges. The polymorphic entity_relationships table does
// not own that column, and planhierarchy's own task query unions it, so
// candidate edges must union it too or the rider would see a dependency graph
// that disagrees with the dispatch filter that produced the candidates.
// *repository.TaskRepository satisfies it.
type PlanHierarchyTaskDependencyReader interface {
	GetTaskDependencies(ctx context.Context, taskKey string) ([]*models.Task, error)
	GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error)
}

// PlanHierarchyEdgeReaders bundles the dependencies DescribeChildEdges needs.
// Relationships and Registry are required for edge loading; TaskDependencies
// is only consulted for task-tier candidates.
type PlanHierarchyEdgeReaders struct {
	Relationships    PlanHierarchyRelationshipReader
	Registry         PlanHierarchyEntityRegistry
	TaskDependencies PlanHierarchyTaskDependencyReader
}

// PlanHierarchyService enumerates claimable direct children for one-level
// hierarchy planning. It owns no transactions and writes no state.
type PlanHierarchyService struct {
	snapshotReader PlanHierarchySnapshotReader
	workflowSvc    PlanHierarchyWorkflowProvider
	claimPolicy    PlanHierarchyClaimPolicy
	edgeReaders    PlanHierarchyEdgeReaders
}

// NewPlanHierarchyService constructs the production one-query hierarchy
// planning reader. Each DescribeChildren call performs one database query
// regardless of the number of direct children.
//
// edgeReaders is only used by DescribeChildEdges. Passing a zero value keeps
// DescribeChildren fully functional; DescribeChildEdges then fails loudly
// rather than reporting every candidate as edge-less.
func NewPlanHierarchyService(
	snapshotReader PlanHierarchySnapshotReader,
	workflowSvc PlanHierarchyWorkflowProvider,
	claimPolicy PlanHierarchyClaimPolicy,
	edgeReaders PlanHierarchyEdgeReaders,
) *PlanHierarchyService {
	return &PlanHierarchyService{
		snapshotReader: snapshotReader,
		workflowSvc:    workflowSvc,
		claimPolicy:    claimPolicy,
		edgeReaders:    edgeReaders,
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
//   - any unsupported parentType, or a missing parent, returns a not-found
//     error.
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

// DescribeChildEdges loads the dependency, blocker, and link edges of the
// given candidate keys so a caller stopping at a fork can decide which subset
// is safe to run in parallel.
//
// entityType is the type of the candidates themselves (not their parent), so
// this works at every tier: task-tier forks get task edges, feature-tier forks
// get feature edges, and any other registered entity type works the same way.
//
// The returned map is keyed by each entity's canonical stored key — the same
// value DescribeChildren reports as PlanHierarchyChild.Key — so callers can
// look candidates up directly regardless of the case or slug form they passed
// in. Candidates with no edges are present with a zero-value entry.
func (s *PlanHierarchyService) DescribeChildEdges(
	ctx context.Context,
	entityType string,
	keys []string,
) (map[string]PlanHierarchyEdges, error) {
	edges := make(map[string]PlanHierarchyEdges, len(keys))
	if len(keys) == 0 {
		return edges, nil
	}
	if s.edgeReaders.Relationships == nil || s.edgeReaders.Registry == nil {
		return nil, fmt.Errorf(
			"describe %s candidate edges: relationship reader and entity registry are required",
			entityType,
		)
	}
	resolvedType := models.EntityType(entityType)
	repo, err := s.edgeReaders.Registry.GetRepository(resolvedType)
	if err != nil {
		return nil, fmt.Errorf("resolve %s repository for candidate edges: %w", entityType, err)
	}
	for _, key := range keys {
		entity, err := repo.GetByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("resolve %s candidate %s for edges: %w", entityType, key, err)
		}
		if entity == nil {
			return nil, fmt.Errorf("resolve %s candidate %s for edges: entity is missing", entityType, key)
		}
		candidateEdges, err := s.describeEntityEdges(ctx, resolvedType, entity)
		if err != nil {
			return nil, err
		}
		edges[entity.GetKey()] = candidateEdges
	}
	return edges, nil
}

// describeEntityEdges classifies one entity's relationships into the
// scheduling-meaningful buckets documented on PlanHierarchyEdges.
func (s *PlanHierarchyService) describeEntityEdges(
	ctx context.Context,
	entityType models.EntityType,
	entity models.Entity,
) (PlanHierarchyEdges, error) {
	outgoing, err := s.edgeReaders.Relationships.GetOutgoing(ctx, entityType, entity.GetID(), nil)
	if err != nil {
		return PlanHierarchyEdges{}, fmt.Errorf(
			"load outgoing relationships for %s %s: %w", entityType, entity.GetKey(), err,
		)
	}
	incoming, err := s.edgeReaders.Relationships.GetIncoming(ctx, entityType, entity.GetID(), nil)
	if err != nil {
		return PlanHierarchyEdges{}, fmt.Errorf(
			"load incoming relationships for %s %s: %w", entityType, entity.GetKey(), err,
		)
	}

	edges := PlanHierarchyEdges{}
	for _, relationship := range outgoing {
		if relationship == nil {
			continue
		}
		edge, err := s.resolveEdgeEndpoint(
			ctx, relationship.ToEntityType, relationship.ToEntityID, relationship.RelationshipType,
		)
		if err != nil {
			if errors.Is(err, repoerr.ErrNotFound) {
				edges.Warnings = append(edges.Warnings, danglingRelationshipWarning(
					"outgoing",
					relationship,
					relationship.ToEntityType,
					relationship.ToEntityID,
				))
				continue
			}
			return PlanHierarchyEdges{}, err
		}
		// Outgoing: this entity depends on the target, or blocks the target.
		switch relationship.RelationshipType {
		case models.EntityRelDependsOn:
			edges.DependsOn = append(edges.DependsOn, edge)
		case models.EntityRelBlocks:
			edges.Blocks = append(edges.Blocks, edge)
		default:
			edges.Links = append(edges.Links, edge)
		}
	}
	for _, relationship := range incoming {
		if relationship == nil {
			continue
		}
		edge, err := s.resolveEdgeEndpoint(
			ctx, relationship.FromEntityType, relationship.FromEntityID, relationship.RelationshipType,
		)
		if err != nil {
			if errors.Is(err, repoerr.ErrNotFound) {
				edges.Warnings = append(edges.Warnings, danglingRelationshipWarning(
					"incoming",
					relationship,
					relationship.FromEntityType,
					relationship.FromEntityID,
				))
				continue
			}
			return PlanHierarchyEdges{}, err
		}
		// Incoming inverts the meaning: the source depends on this entity
		// (so this entity blocks it), or the source blocks this entity.
		switch relationship.RelationshipType {
		case models.EntityRelDependsOn:
			edges.Blocks = append(edges.Blocks, edge)
		case models.EntityRelBlocks:
			edges.DependsOn = append(edges.DependsOn, edge)
		default:
			edges.Links = append(edges.Links, edge)
		}
	}

	if err := s.appendLegacyTaskEdges(ctx, entityType, entity.GetKey(), &edges); err != nil {
		return PlanHierarchyEdges{}, err
	}

	edges.DependsOn = dedupePlanHierarchyEdges(edges.DependsOn)
	edges.Blocks = dedupePlanHierarchyEdges(edges.Blocks)
	edges.Links = dedupePlanHierarchyEdges(edges.Links)
	return edges, nil
}

func danglingRelationshipWarning(
	direction string,
	relationship *models.EntityRelationship,
	endpointType models.EntityType,
	endpointID int64,
) PlanHierarchyEdgeWarning {
	return PlanHierarchyEdgeWarning{
		Code:             PlanHierarchyWarningDanglingRelationship,
		Direction:        direction,
		RelationshipID:   relationship.ID,
		EndpointType:     endpointType,
		EndpointID:       endpointID,
		RelationshipType: relationship.RelationshipType,
	}
}

// appendLegacyTaskEdges unions the task-only tasks.depends_on JSON column into
// the edge buckets. GetTaskDependencies/GetTaskDependents already include the
// entity_relationships rows as well; dedupePlanHierarchyEdges collapses the
// resulting overlap.
func (s *PlanHierarchyService) appendLegacyTaskEdges(
	ctx context.Context,
	entityType models.EntityType,
	key string,
	edges *PlanHierarchyEdges,
) error {
	if entityType != models.EntityTypeTask || s.edgeReaders.TaskDependencies == nil {
		return nil
	}
	dependencies, err := s.edgeReaders.TaskDependencies.GetTaskDependencies(ctx, key)
	if err != nil {
		return fmt.Errorf("load task dependencies for %s: %w", key, err)
	}
	for _, dependency := range dependencies {
		if dependency == nil {
			continue
		}
		edges.DependsOn = append(edges.DependsOn, PlanHierarchyEdge{
			Key:    dependency.Key,
			Status: string(dependency.Status),
			Type:   string(models.EntityRelDependsOn),
		})
	}
	dependents, err := s.edgeReaders.TaskDependencies.GetTaskDependents(ctx, key)
	if err != nil {
		return fmt.Errorf("load task dependents for %s: %w", key, err)
	}
	for _, dependent := range dependents {
		if dependent == nil {
			continue
		}
		edges.Blocks = append(edges.Blocks, PlanHierarchyEdge{
			Key:    dependent.Key,
			Status: string(dependent.Status),
			Type:   string(models.EntityRelDependsOn),
		})
	}
	return nil
}

// resolveEdgeEndpoint turns a relationship endpoint into a key + status edge.
// The registry lookup is what makes this entity-generic: any registered entity
// type on the far side of an edge resolves the same way.
func (s *PlanHierarchyService) resolveEdgeEndpoint(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	relationshipType models.EntityRelationshipType,
) (PlanHierarchyEdge, error) {
	repo, err := s.edgeReaders.Registry.GetRepository(entityType)
	if err != nil {
		return PlanHierarchyEdge{}, fmt.Errorf("resolve %s edge repository: %w", entityType, err)
	}
	entity, err := repo.GetByID(ctx, entityID)
	if err != nil {
		return PlanHierarchyEdge{}, fmt.Errorf("resolve %s edge endpoint %d: %w", entityType, entityID, err)
	}
	if entity == nil {
		return PlanHierarchyEdge{}, fmt.Errorf("resolve %s edge endpoint %d: entity is missing", entityType, entityID)
	}
	return PlanHierarchyEdge{
		Key:    entity.GetKey(),
		Status: entity.GetStatus(),
		Type:   string(relationshipType),
	}, nil
}

// dedupePlanHierarchyEdges collapses repeated (key, type) pairs, keeping first
// occurrence so ordering stays deterministic. Duplicates are expected: the
// legacy task union and the relationship table describe some of the same edges.
func dedupePlanHierarchyEdges(edges []PlanHierarchyEdge) []PlanHierarchyEdge {
	if len(edges) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(edges))
	deduped := make([]PlanHierarchyEdge, 0, len(edges))
	for _, edge := range edges {
		identity := edge.Key + "\x00" + edge.Type
		if seen[identity] {
			continue
		}
		seen[identity] = true
		deduped = append(deduped, edge)
	}
	return deduped
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
