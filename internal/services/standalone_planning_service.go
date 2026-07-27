package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// StandalonePlanCollection identifies a supported root collection for
// collection-scoped shark next selection.
type StandalonePlanCollection string

const (
	// StandalonePlanBugs selects bug roots by severity.
	StandalonePlanBugs StandalonePlanCollection = "bugs"
	// StandalonePlanChangeCards selects change-card roots by numeric priority.
	StandalonePlanChangeCards StandalonePlanCollection = "change-cards"
	// StandalonePlanTechDebt selects tech-debt roots by severity.
	StandalonePlanTechDebt StandalonePlanCollection = "tech-debt"
)

// StandalonePlanCandidate is one claimable standalone root.
type StandalonePlanCandidate struct {
	Key        string
	EntityType models.EntityType
}

// StandalonePlan groups candidates into ordered priority tiers. Candidates
// within one tier may be resolved together; later tiers are considered only
// when earlier tiers contain no workflow-ready dispatch.
type StandalonePlan struct {
	Collection StandalonePlanCollection
	Layers     [][]StandalonePlanCandidate
}

// StandalonePlanBugReader is the bug list surface used by next planning.
type StandalonePlanBugReader interface {
	ListBugs(ctx context.Context, filters BugFilters) ([]*models.Bug, error)
}

// StandalonePlanChangeReader is the change-card list surface used by next planning.
type StandalonePlanChangeReader interface {
	ListChangeCards(ctx context.Context, filters ChangeCardFilters) ([]*models.ChangeCard, error)
}

// StandalonePlanTechDebtReader is the tech-debt list surface used by next planning.
type StandalonePlanTechDebtReader interface {
	ListTechDebts(ctx context.Context, filters TechDebtFilters) ([]*models.TechDebt, error)
}

// StandalonePlanClaimReader supplies non-mutating claimability checks.
type StandalonePlanClaimReader interface {
	IsClaimable(ctx context.Context, entityType, entityKey string) (bool, error)
}

// StandalonePlanDependencyReader checks unresolved hard relationships for one
// standalone candidate.
type StandalonePlanDependencyReader interface {
	HasUnresolvedHardDependency(ctx context.Context, entityType models.EntityType, entityID int64) (bool, error)
}

// StandalonePlanningService selects standalone roots without dispatching or
// mutating workflow state.
type StandalonePlanningService struct {
	bugs         StandalonePlanBugReader
	changes      StandalonePlanChangeReader
	techDebt     StandalonePlanTechDebtReader
	claims       StandalonePlanClaimReader
	dependencies StandalonePlanDependencyReader
}

// NewStandalonePlanningService constructs a standalone collection planner.
func NewStandalonePlanningService(
	bugs StandalonePlanBugReader,
	changes StandalonePlanChangeReader,
	techDebt StandalonePlanTechDebtReader,
	claims StandalonePlanClaimReader,
	dependencies StandalonePlanDependencyReader,
) *StandalonePlanningService {
	requireNonNil(bugs, "StandalonePlanningService requires a non-nil bug reader")
	requireNonNil(changes, "StandalonePlanningService requires a non-nil change-card reader")
	requireNonNil(techDebt, "StandalonePlanningService requires a non-nil tech-debt reader")
	requireNonNil(claims, "StandalonePlanningService requires a non-nil claim reader")
	requireNonNil(dependencies, "StandalonePlanningService requires a non-nil dependency reader")
	return &StandalonePlanningService{
		bugs: bugs, changes: changes, techDebt: techDebt, claims: claims, dependencies: dependencies,
	}
}

// Plan returns dependency-satisfied, claimable roots grouped by stored
// priority tier. Production list readers exclude terminal roots; callers apply
// final workflow-action readiness filtering.
func (s *StandalonePlanningService) Plan(ctx context.Context, collection StandalonePlanCollection) (StandalonePlan, error) {
	ranked, err := s.listRankedCandidates(ctx, collection)
	if err != nil {
		return StandalonePlan{}, err
	}
	claimable, err := s.filterClaimable(ctx, ranked)
	if err != nil {
		return StandalonePlan{}, err
	}
	return StandalonePlan{Collection: collection, Layers: groupStandaloneCandidates(claimable)}, nil
}

type rankedStandaloneCandidate struct {
	StandalonePlanCandidate
	id        int64
	rank      int
	createdAt time.Time
}

func (s *StandalonePlanningService) listRankedCandidates(
	ctx context.Context,
	collection StandalonePlanCollection,
) ([]rankedStandaloneCandidate, error) {
	switch collection {
	case StandalonePlanBugs:
		return s.listRankedBugs(ctx)
	case StandalonePlanChangeCards:
		return s.listRankedChangeCards(ctx)
	case StandalonePlanTechDebt:
		return s.listRankedTechDebt(ctx)
	default:
		return nil, fmt.Errorf("unsupported standalone next collection %q", collection)
	}
}

func (s *StandalonePlanningService) listRankedBugs(ctx context.Context) ([]rankedStandaloneCandidate, error) {
	bugs, err := s.bugs.ListBugs(ctx, BugFilters{})
	if err != nil {
		return nil, fmt.Errorf("plan next bugs: %w", err)
	}
	out := make([]rankedStandaloneCandidate, 0, len(bugs))
	for _, bug := range bugs {
		if bug == nil {
			continue
		}
		out = append(out, newRankedStandaloneCandidate(
			bug.ID, bug.Key, models.EntityTypeBug, severityRank(string(bug.Severity)), bug.CreatedAt,
		))
	}
	return out, nil
}

func (s *StandalonePlanningService) listRankedChangeCards(ctx context.Context) ([]rankedStandaloneCandidate, error) {
	cards, err := s.changes.ListChangeCards(ctx, ChangeCardFilters{})
	if err != nil {
		return nil, fmt.Errorf("plan next change-cards: %w", err)
	}
	out := make([]rankedStandaloneCandidate, 0, len(cards))
	for _, card := range cards {
		if card == nil {
			continue
		}
		rank := card.Priority
		if rank <= 0 {
			rank = 11
		}
		out = append(out, newRankedStandaloneCandidate(
			card.ID, card.Key, models.EntityTypeChange, rank, card.CreatedAt,
		))
	}
	return out, nil
}

func (s *StandalonePlanningService) listRankedTechDebt(ctx context.Context) ([]rankedStandaloneCandidate, error) {
	items, err := s.techDebt.ListTechDebts(ctx, TechDebtFilters{})
	if err != nil {
		return nil, fmt.Errorf("plan next tech-debt: %w", err)
	}
	out := make([]rankedStandaloneCandidate, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, newRankedStandaloneCandidate(
			item.ID, item.Key, models.EntityTypeTechDebt, severityRank(string(item.Severity)), item.CreatedAt,
		))
	}
	return out, nil
}

func newRankedStandaloneCandidate(
	id int64,
	key string,
	entityType models.EntityType,
	rank int,
	createdAt time.Time,
) rankedStandaloneCandidate {
	return rankedStandaloneCandidate{
		StandalonePlanCandidate: StandalonePlanCandidate{Key: key, EntityType: entityType},
		id:                      id,
		rank:                    rank,
		createdAt:               createdAt,
	}
}

func (s *StandalonePlanningService) filterClaimable(
	ctx context.Context,
	candidates []rankedStandaloneCandidate,
) ([]rankedStandaloneCandidate, error) {
	out := make([]rankedStandaloneCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		blocked, err := s.dependencies.HasUnresolvedHardDependency(ctx, candidate.EntityType, candidate.id)
		if err != nil {
			return nil, fmt.Errorf("check dependencies for %s %s: %w", candidate.EntityType, candidate.Key, err)
		}
		if blocked {
			continue
		}
		claimable, err := s.claims.IsClaimable(ctx, string(candidate.EntityType), candidate.Key)
		if err != nil {
			return nil, fmt.Errorf("check claim for %s %s: %w", candidate.EntityType, candidate.Key, err)
		}
		if claimable {
			out = append(out, candidate)
		}
	}
	return out, nil
}

// StandaloneHardDependencyRelationshipReader is the relationship surface used
// to evaluate hard standalone dependencies.
type StandaloneHardDependencyRelationshipReader interface {
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

// StandaloneHardDependencyRegistry resolves relationship endpoints.
type StandaloneHardDependencyRegistry interface {
	GetRepository(entityType models.EntityType) (EntityRepository, error)
}

// StandaloneHardDependencyWorkflowProvider supplies terminal classifiers.
type StandaloneHardDependencyWorkflowProvider interface {
	ForLevel(level string) *workflow.Service
}

// StandaloneHardDependencyService evaluates outgoing depends_on and incoming
// blocks relationships for standalone next selection.
type StandaloneHardDependencyService struct {
	relationships StandaloneHardDependencyRelationshipReader
	registry      StandaloneHardDependencyRegistry
	workflows     StandaloneHardDependencyWorkflowProvider
}

// NewStandaloneHardDependencyService constructs a hard-dependency evaluator.
func NewStandaloneHardDependencyService(
	relationships StandaloneHardDependencyRelationshipReader,
	registry StandaloneHardDependencyRegistry,
	workflows StandaloneHardDependencyWorkflowProvider,
) *StandaloneHardDependencyService {
	requireNonNil(relationships, "StandaloneHardDependencyService requires a non-nil relationship reader")
	requireNonNil(registry, "StandaloneHardDependencyService requires a non-nil entity registry")
	requireNonNil(workflows, "StandaloneHardDependencyService requires a non-nil workflow provider")
	return &StandaloneHardDependencyService{relationships: relationships, registry: registry, workflows: workflows}
}

// HasUnresolvedHardDependency reports whether a standalone root must wait for a
// non-terminal dependency or blocker.
func (s *StandaloneHardDependencyService) HasUnresolvedHardDependency(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
) (bool, error) {
	outgoing, err := s.relationships.GetOutgoing(
		ctx, entityType, entityID, []models.EntityRelationshipType{models.EntityRelDependsOn},
	)
	if err != nil {
		return false, fmt.Errorf("list outgoing depends_on relationships: %w", err)
	}
	for _, relationship := range outgoing {
		if relationship == nil {
			continue
		}
		terminal, err := s.isTerminalEntity(ctx, relationship.ToEntityType, relationship.ToEntityID)
		if err != nil {
			return false, err
		}
		if !terminal {
			return true, nil
		}
	}

	incoming, err := s.relationships.GetIncoming(
		ctx, entityType, entityID, []models.EntityRelationshipType{models.EntityRelBlocks},
	)
	if err != nil {
		return false, fmt.Errorf("list incoming blocks relationships: %w", err)
	}
	for _, relationship := range incoming {
		if relationship == nil {
			continue
		}
		terminal, err := s.isTerminalEntity(ctx, relationship.FromEntityType, relationship.FromEntityID)
		if err != nil {
			return false, err
		}
		if !terminal {
			return true, nil
		}
	}
	return false, nil
}

func (s *StandaloneHardDependencyService) isTerminalEntity(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
) (bool, error) {
	repository, err := s.registry.GetRepository(entityType)
	if err != nil {
		return false, fmt.Errorf("resolve %s dependency repository: %w", entityType, err)
	}
	entity, err := repository.GetByID(ctx, entityID)
	if err != nil {
		return false, fmt.Errorf("resolve %s dependency %d: %w", entityType, entityID, err)
	}
	if entity == nil {
		return false, fmt.Errorf("resolve %s dependency %d: entity is missing", entityType, entityID)
	}
	classifier := s.workflows.ForLevel(string(entityType))
	if classifier == nil {
		return false, fmt.Errorf("resolve %s dependency %d: workflow is unavailable", entityType, entityID)
	}
	return classifier.IsTerminalStatus(entity.GetStatus()), nil
}

func groupStandaloneCandidates(candidates []rankedStandaloneCandidate) [][]StandalonePlanCandidate {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].rank != candidates[j].rank {
			return candidates[i].rank < candidates[j].rank
		}
		if !candidates[i].createdAt.Equal(candidates[j].createdAt) {
			return candidates[i].createdAt.Before(candidates[j].createdAt)
		}
		return candidates[i].Key < candidates[j].Key
	})

	layers := make([][]StandalonePlanCandidate, 0)
	lastRank := -1
	for _, candidate := range candidates {
		if len(layers) == 0 || candidate.rank != lastRank {
			layers = append(layers, []StandalonePlanCandidate{})
			lastRank = candidate.rank
		}
		index := len(layers) - 1
		layers[index] = append(layers[index], candidate.StandalonePlanCandidate)
	}
	return layers
}

func severityRank(severity string) int {
	// Bug and tech-debt severities share the same stored values, so BugSeverity
	// is the comparison vocabulary for both collections.
	switch models.BugSeverity(severity) {
	case models.BugSeverityCritical:
		return 1
	case models.BugSeverityHigh:
		return 2
	case models.BugSeverityMedium:
		return 3
	case models.BugSeverityLow:
		return 4
	default:
		return 5
	}
}
