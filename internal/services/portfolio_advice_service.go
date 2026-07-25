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

// PortfolioAdviceWorkflowProvider supplies configured classifiers by entity level.
type PortfolioAdviceWorkflowProvider interface {
	ForLevel(level string) *workflow.Service
}

// PortfolioAdviceService assembles the read-only evidence used by bare shark plan.
type PortfolioAdviceService struct {
	snapshotSource PortfolioSnapshotSource
	claimFilter    PortfolioClaimFilter
	workflows      PortfolioAdviceWorkflowProvider
}

// NewPortfolioAdviceServiceFromSnapshot constructs the portfolio advice service
// from the same complete snapshot and claim-filter seams used in production.
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
	children      []portfoliorepo.ChildStateRow
	relationships []portfoliorepo.EpicRelationshipRow
	claims        []*models.EntityClaim
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

func assemblePortfolioAdvice(
	epics []*models.Epic,
	reads portfolioAdviceReads,
	workflows portfolioAdviceWorkflows,
) *models.PortfolioAdviceEnvelope {
	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = true
	advice.Prompt = portfolioAdvicePrompt

	states, byKey, allEpics := buildPortfolioEpicStates(epics, workflows.epic, advice)
	ownership := make(map[string]string)
	assemblePortfolioChildren(states, byKey, reads.children, ownership, workflows.feature, workflows.task, advice)
	assemblePortfolioRelationships(byKey, allEpics, reads.relationships, workflows.epic, advice)
	assemblePortfolioClaims(byKey, ownership, reads.claims, advice)
	finalizePortfolioEpics(states, advice)
	advice.Ordering = analyzePortfolioGraph(advice.Epics, advice.Relationships)
	sortPortfolioWarnings(advice.Warnings)
	return advice
}
