package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

const portfolioAdvicePrompt = "Inspect docs/product/cross-epic-integration-map.md when it exists.\n" +
	"Treat this envelope's state, relationships, blockers, and active work as the live Shark authority; treat product documents only as intent and decision context.\n" +
	"Respect hard precedence before considering priority, business value, progress, and continuity from active work; do not convert those fields into an undocumented weighted score.\n" +
	"Recommend exactly one eligibility=eligible epic key, give the decisive \"why now\" evidence, and compare it with the strongest eligible alternative.\n" +
	"If evidence_complete=false, no eligible root exists, or evidence contradicts, report the condition and the next evidence or relationship fix instead of guessing; when no eligible root exists, no root can be recommended from current Shark state.\n" +
	"End at advice. Do not claim, dispatch, or advance the root."

// PortfolioAdviceEpicReader is the read-only epic surface used by portfolio advice.
type PortfolioAdviceEpicReader interface {
	List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
}

// PortfolioAdviceSnapshotReader is the set-oriented read surface used by portfolio advice.
type PortfolioAdviceSnapshotReader interface {
	ListChildStates(ctx context.Context) ([]portfoliorepo.ChildStateRow, error)
	ListEpicRelationships(ctx context.Context) ([]portfoliorepo.EpicRelationshipRow, error)
}

// PortfolioAdviceClaimReader is the non-mutating claim surface used by portfolio advice.
type PortfolioAdviceClaimReader interface {
	ListActiveReadOnly(ctx context.Context, evaluatedAt time.Time) ([]*models.EntityClaim, error)
}

// PortfolioAdviceWorkflowProvider supplies configured classifiers by entity level.
type PortfolioAdviceWorkflowProvider interface {
	ForLevel(level string) *workflow.Service
}

// PortfolioAdviceService assembles the read-only evidence returned by bare shark next.
type PortfolioAdviceService struct {
	epics     PortfolioAdviceEpicReader
	snapshot  PortfolioAdviceSnapshotReader
	claims    PortfolioAdviceClaimReader
	workflows PortfolioAdviceWorkflowProvider

	snapshotSource PortfolioSnapshotSource
	claimFilter    PortfolioClaimFilter
}

// NewPortfolioAdviceService constructs a portfolio advice service from read-only dependencies.
func NewPortfolioAdviceService(
	epics PortfolioAdviceEpicReader,
	snapshot PortfolioAdviceSnapshotReader,
	claims PortfolioAdviceClaimReader,
	workflows PortfolioAdviceWorkflowProvider,
) *PortfolioAdviceService {
	requireNonNil(epics, "PortfolioAdviceService requires a non-nil epic reader")
	requireNonNil(snapshot, "PortfolioAdviceService requires a non-nil snapshot reader")
	requireNonNil(claims, "PortfolioAdviceService requires a non-nil claim reader")
	requireNonNil(workflows, "PortfolioAdviceService requires a non-nil workflow provider")
	return &PortfolioAdviceService{epics: epics, snapshot: snapshot, claims: claims, workflows: workflows}
}

// NewPortfolioAdviceServiceFromSnapshot constructs the production one-query
// portfolio advice path.
func NewPortfolioAdviceServiceFromSnapshot(
	snapshotSource PortfolioSnapshotSource,
	claimFilter PortfolioClaimFilter,
	workflows PortfolioAdviceWorkflowProvider,
) *PortfolioAdviceService {
	requireNonNil(snapshotSource, "PortfolioAdviceService requires a non-nil snapshot source")
	requireNonNil(claimFilter, "PortfolioAdviceService requires a non-nil claim filter")
	requireNonNil(workflows, "PortfolioAdviceService requires a non-nil workflow provider")
	return &PortfolioAdviceService{
		snapshotSource: snapshotSource,
		claimFilter:    claimFilter,
		workflows:      workflows,
	}
}

// Advise returns portfolio evidence without mutating Shark state.
func (s *PortfolioAdviceService) Advise(ctx context.Context) (*models.PortfolioAdviceEnvelope, error) {
	if err := portfolioAdviceContextError(ctx, nil); err != nil {
		return nil, err
	}
	evaluatedAt := time.Now().UTC()

	if s.snapshotSource != nil {
		return s.adviseFromSnapshot(ctx, evaluatedAt)
	}

	epics, err := s.listPortfolioEpics(ctx)
	if err != nil {
		return nil, err
	}

	workflows, err := s.portfolioWorkflows()
	if err != nil {
		return nil, err
	}

	reads, err := s.readPortfolioAdviceEvidence(ctx, evaluatedAt)
	if err != nil {
		return nil, err
	}
	advice := assemblePortfolioAdvice(epics, reads, workflows)
	if err := portfolioAdviceContextError(ctx, nil); err != nil {
		return nil, err
	}
	return advice, nil
}

func (s *PortfolioAdviceService) adviseFromSnapshot(
	ctx context.Context,
	evaluatedAt time.Time,
) (*models.PortfolioAdviceEnvelope, error) {
	snapshot, err := s.snapshotSource.ReadSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("assemble portfolio advice snapshot: %w", err)
	}
	workflows, err := s.portfolioWorkflows()
	if err != nil {
		return nil, err
	}
	reads := portfolioAdviceReads{
		children:      snapshot.Children,
		relationships: snapshot.Relationships,
		claims:        s.claimFilter.FilterActiveReadOnly(snapshot.Claims, evaluatedAt),
	}
	advice := assemblePortfolioAdvice(snapshot.Epics, reads, workflows)
	if err := portfolioAdviceContextError(ctx, nil); err != nil {
		return nil, err
	}
	return advice, nil
}

type portfolioAdviceWorkflows struct {
	epic    *workflow.Service
	feature *workflow.Service
	task    *workflow.Service
}

type portfolioAdviceReads struct {
	children        []portfoliorepo.ChildStateRow
	childErr        error
	relationships   []portfoliorepo.EpicRelationshipRow
	relationshipErr error
	claims          []*models.EntityClaim
	claimErr        error
}

func (s *PortfolioAdviceService) listPortfolioEpics(ctx context.Context) ([]*models.Epic, error) {
	epics, err := s.epics.List(ctx, nil)
	if err == nil {
		return epics, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("assemble portfolio advice: %w", ctxErr)
	}
	return nil, fmt.Errorf("list portfolio epics: %w", err)
}

func (s *PortfolioAdviceService) portfolioWorkflows() (portfolioAdviceWorkflows, error) {
	configured := portfolioAdviceWorkflows{
		epic:    s.workflows.ForLevel(workflow.LevelEpic),
		feature: s.workflows.ForLevel(workflow.LevelFeature),
		task:    s.workflows.ForLevel(workflow.LevelTask),
	}
	if configured.epic == nil || configured.feature == nil || configured.task == nil {
		return portfolioAdviceWorkflows{}, fmt.Errorf("assemble portfolio advice: workflow configuration is unavailable")
	}
	return configured, nil
}

func (s *PortfolioAdviceService) readPortfolioAdviceEvidence(
	ctx context.Context,
	evaluatedAt time.Time,
) (portfolioAdviceReads, error) {
	reads := portfolioAdviceReads{}
	reads.children, reads.childErr = s.snapshot.ListChildStates(ctx)
	if err := portfolioAdviceContextError(ctx, reads.childErr); err != nil {
		return portfolioAdviceReads{}, err
	}
	reads.relationships, reads.relationshipErr = s.snapshot.ListEpicRelationships(ctx)
	if err := portfolioAdviceContextError(ctx, reads.relationshipErr); err != nil {
		return portfolioAdviceReads{}, err
	}
	reads.claims, reads.claimErr = s.claims.ListActiveReadOnly(ctx, evaluatedAt)
	if err := portfolioAdviceContextError(ctx, reads.claimErr); err != nil {
		return portfolioAdviceReads{}, err
	}
	return reads, nil
}

func assemblePortfolioAdvice(
	epics []*models.Epic,
	reads portfolioAdviceReads,
	workflows portfolioAdviceWorkflows,
) *models.PortfolioAdviceEnvelope {
	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = reads.childErr == nil && reads.relationshipErr == nil && reads.claimErr == nil
	advice.Prompt = portfolioAdvicePrompt

	states, byKey, allEpics := buildPortfolioEpicStates(epics, workflows.epic, advice)
	ownership := make(map[string]string)
	applyPortfolioChildEvidence(states, byKey, reads, ownership, workflows, advice)
	applyPortfolioRelationshipEvidence(states, byKey, allEpics, reads, workflows.epic, advice)
	applyPortfolioClaimEvidence(byKey, ownership, reads, advice)
	finalizePortfolioEpics(states, advice)
	if reads.relationshipErr == nil {
		advice.Ordering = analyzePortfolioGraph(advice.Epics, advice.Relationships)
	}
	sortPortfolioWarnings(advice.Warnings)
	return advice
}

func applyPortfolioChildEvidence(
	states []*portfolioEpicAssembly,
	byKey map[string]*portfolioEpicAssembly,
	reads portfolioAdviceReads,
	ownership map[string]string,
	workflows portfolioAdviceWorkflows,
	advice *models.PortfolioAdviceEnvelope,
) {
	if reads.childErr == nil {
		assemblePortfolioChildren(states, byKey, reads.children, ownership, workflows.feature, workflows.task, advice)
		return
	}
	advice.Warnings = append(advice.Warnings, portfolioEvidenceWarning(
		models.PortfolioWarningChildStateUnavailable,
		"portfolio descendant state is unavailable; eligibility requiring child evidence is unknown",
	))
	markPortfolioEligibilityUnknown(states)
}

func applyPortfolioRelationshipEvidence(
	states []*portfolioEpicAssembly,
	byKey map[string]*portfolioEpicAssembly,
	allEpics map[string]*models.Epic,
	reads portfolioAdviceReads,
	epicWorkflow *workflow.Service,
	advice *models.PortfolioAdviceEnvelope,
) {
	if reads.relationshipErr == nil {
		assemblePortfolioRelationships(byKey, allEpics, reads.relationships, epicWorkflow, advice)
		return
	}
	advice.Warnings = append(advice.Warnings, portfolioEvidenceWarning(
		models.PortfolioWarningRelationshipStateUnavailable,
		"portfolio relationship state is unavailable; relationship-dependent eligibility is unknown",
	))
	markPortfolioEligibilityUnknown(states)
}

func applyPortfolioClaimEvidence(
	byKey map[string]*portfolioEpicAssembly,
	ownership map[string]string,
	reads portfolioAdviceReads,
	advice *models.PortfolioAdviceEnvelope,
) {
	if reads.claimErr == nil {
		assemblePortfolioClaims(byKey, ownership, reads.claims, advice)
		return
	}
	advice.Warnings = append(advice.Warnings, portfolioEvidenceWarning(
		models.PortfolioWarningClaimStateUnavailable,
		"portfolio active-claim state is unavailable; active work evidence is incomplete",
	))
}

func markPortfolioEligibilityUnknown(states []*portfolioEpicAssembly) {
	for _, state := range states {
		state.eligibilityUnknown = true
	}
}
