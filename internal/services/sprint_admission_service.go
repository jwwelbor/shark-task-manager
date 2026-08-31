package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

type SprintAdmissionState string

const (
	SprintAdmissionAllowed    SprintAdmissionState = "allowed"
	SprintAdmissionBlocked    SprintAdmissionState = "blocked"
	SprintAdmissionOverridden SprintAdmissionState = "overridden"
)

type SprintAdmissionReasonCode string

const (
	SprintAdmissionReasonAncestorDependency SprintAdmissionReasonCode = "ancestor_dependency_unmet"
	SprintAdmissionReasonOutsidePortfolio   SprintAdmissionReasonCode = "outside_portfolio_gate"
)

type SprintAdmissionCandidate struct {
	Key     string
	EpicKey string
}

type SprintAdmissionEvidence struct {
	PortfolioEpicKey string
	Candidates       map[string]SprintAdmissionCandidate
	UnmetAncestors   map[string][]string
}

type SprintAdmissionEvidenceReader interface {
	ReadSprintAdmissionEvidence(ctx context.Context) (*SprintAdmissionEvidence, error)
}

// PortfolioSprintAdmissionEvidenceReader adapts the existing set-oriented
// portfolio snapshot into the narrower admission evidence contract.
type PortfolioSprintAdmissionEvidenceReader struct {
	snapshotSource PortfolioSnapshotSource
	advisor        *PortfolioAdviceService
	planner        *PortfolioPlanningService
	workflows      PortfolioAdviceWorkflowProvider
}

func NewPortfolioSprintAdmissionEvidenceReader(
	snapshotSource PortfolioSnapshotSource,
	advisor *PortfolioAdviceService,
	planner *PortfolioPlanningService,
	workflows PortfolioAdviceWorkflowProvider,
) *PortfolioSprintAdmissionEvidenceReader {
	requireNonNil(snapshotSource, "Sprint admission evidence requires a portfolio snapshot source")
	requireNonNil(advisor, "Sprint admission evidence requires a portfolio advisor")
	requireNonNil(planner, "Sprint admission evidence requires a portfolio planner")
	requireNonNil(workflows, "Sprint admission evidence requires workflow metadata")
	return &PortfolioSprintAdmissionEvidenceReader{snapshotSource: snapshotSource, advisor: advisor, planner: planner, workflows: workflows}
}

func (r *PortfolioSprintAdmissionEvidenceReader) ReadSprintAdmissionEvidence(ctx context.Context) (*SprintAdmissionEvidence, error) {
	advice, err := r.advisor.Advise(ctx)
	if err != nil {
		return nil, fmt.Errorf("read portfolio admission advice: %w", err)
	}
	plan := r.planner.Plan(advice)
	// Zero eligible roots (between epics, all blocked, or portfolio paused) and
	// multiple tied roots are both normal portfolio states, not errors. Zero
	// roots means no portfolio gate is currently active, so admission evidence
	// carries an empty PortfolioEpicKey and evaluateSprintAdmissionEvidence
	// skips the outside-portfolio-gate check entirely. A tie resolves
	// deterministically to the lexicographically lowest root key, mirroring
	// the "one active gate" mental model already used elsewhere for parallel
	// ties.
	portfolioEpicKey := ""
	switch {
	case len(plan.RootKeys) == 1:
		portfolioEpicKey = plan.RootKeys[0]
	case len(plan.RootKeys) > 1:
		tied := append([]string(nil), plan.RootKeys...)
		sort.Strings(tied)
		portfolioEpicKey = tied[0]
	}
	snapshot, err := r.snapshotSource.ReadSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("read portfolio admission snapshot: %w", err)
	}
	epicWorkflow := r.workflows.ForLevel(workflow.LevelEpic)
	if epicWorkflow == nil {
		return nil, fmt.Errorf("read portfolio admission snapshot: epic workflow is unavailable")
	}
	return sprintAdmissionEvidenceFromSnapshot(snapshot, portfolioEpicKey, epicWorkflow)
}

func sprintAdmissionEvidenceFromSnapshot(snapshot portfoliorepo.Snapshot, portfolioEpicKey string, epicWorkflow *workflow.Service) (*SprintAdmissionEvidence, error) {
	evidence := &SprintAdmissionEvidence{
		PortfolioEpicKey: portfolioEpicKey,
		Candidates:       make(map[string]SprintAdmissionCandidate, len(snapshot.Epics)+len(snapshot.Children)),
		UnmetAncestors:   make(map[string][]string),
	}
	for _, epic := range snapshot.Epics {
		if epic != nil {
			evidence.Candidates[epic.Key] = SprintAdmissionCandidate{Key: epic.Key, EpicKey: epic.Key}
		}
	}
	for _, child := range snapshot.Children {
		evidence.Candidates[child.EntityKey] = SprintAdmissionCandidate{Key: child.EntityKey, EpicKey: child.EpicKey}
	}
	for _, relationship := range snapshot.Relationships {
		if relationship.RelationshipType != models.EntityRelDependsOn {
			continue
		}
		if relationship.FromKey == nil || relationship.ToKey == nil || relationship.ToStatus == nil {
			return nil, fmt.Errorf("read portfolio admission snapshot: dependency relationship has an unavailable endpoint")
		}
		if epicWorkflow.IsTerminalStatus(*relationship.ToStatus) && !epicWorkflow.GetStatusMetadata(*relationship.ToStatus).ExcludeFromProgress {
			continue
		}
		evidence.UnmetAncestors[*relationship.FromKey] = append(evidence.UnmetAncestors[*relationship.FromKey], *relationship.ToKey)
	}
	for epicKey := range evidence.UnmetAncestors {
		sort.Strings(evidence.UnmetAncestors[epicKey])
	}
	return evidence, nil
}

type SprintAdmissionDecision struct {
	CandidateKey      string                    `json:"candidate_key"`
	PortfolioEpicKey  string                    `json:"portfolio_epic_key"`
	UnmetAncestorKeys []string                  `json:"unmet_ancestor_keys"`
	State             SprintAdmissionState      `json:"state"`
	ReasonCode        SprintAdmissionReasonCode `json:"reason_code,omitempty"`
}

type SprintAdmissionService struct {
	evidenceReader SprintAdmissionEvidenceReader
}

func NewSprintAdmissionService(evidenceReader SprintAdmissionEvidenceReader) *SprintAdmissionService {
	requireNonNil(evidenceReader, "SprintAdmissionService requires an evidence reader")
	return &SprintAdmissionService{evidenceReader: evidenceReader}
}

func (s *SprintAdmissionService) Evaluate(ctx context.Context, candidateKey string) (SprintAdmissionDecision, error) {
	evidence, err := s.evidenceReader.ReadSprintAdmissionEvidence(ctx)
	if err != nil {
		return SprintAdmissionDecision{}, fmt.Errorf("read sprint admission evidence: %w", err)
	}
	if evidence == nil {
		return SprintAdmissionDecision{}, fmt.Errorf("read sprint admission evidence: empty evidence")
	}
	return evaluateSprintAdmissionEvidence(evidence, candidateKey)
}

// EvaluateCandidates evaluates a bounded candidate set from one evidence
// snapshot. Read-only planning consumers use this method to avoid one
// portfolio read per candidate.
func (s *SprintAdmissionService) EvaluateCandidates(ctx context.Context, candidateKeys []string) ([]SprintAdmissionDecision, error) {
	evidence, err := s.evidenceReader.ReadSprintAdmissionEvidence(ctx)
	if err != nil {
		return nil, fmt.Errorf("read sprint admission evidence: %w", err)
	}
	if evidence == nil {
		return nil, fmt.Errorf("read sprint admission evidence: empty evidence")
	}
	decisions := make([]SprintAdmissionDecision, 0, len(candidateKeys))
	for _, candidateKey := range candidateKeys {
		decision, err := evaluateSprintAdmissionEvidence(evidence, candidateKey)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

func evaluateSprintAdmissionEvidence(evidence *SprintAdmissionEvidence, candidateKey string) (SprintAdmissionDecision, error) {
	candidate, ok := evidence.Candidates[candidateKey]
	if !ok {
		return SprintAdmissionDecision{}, fmt.Errorf("read sprint admission evidence: candidate %q is unavailable", candidateKey)
	}

	decision := SprintAdmissionDecision{
		CandidateKey:      candidateKey,
		PortfolioEpicKey:  evidence.PortfolioEpicKey,
		UnmetAncestorKeys: append([]string(nil), evidence.UnmetAncestors[candidate.EpicKey]...),
		State:             SprintAdmissionAllowed,
	}
	sort.Strings(decision.UnmetAncestorKeys)
	if len(decision.UnmetAncestorKeys) > 0 {
		decision.State = SprintAdmissionBlocked
		decision.ReasonCode = SprintAdmissionReasonAncestorDependency
		return decision, nil
	}
	if evidence.PortfolioEpicKey != "" && candidate.EpicKey != evidence.PortfolioEpicKey {
		decision.State = SprintAdmissionBlocked
		decision.ReasonCode = SprintAdmissionReasonOutsidePortfolio
	}
	return decision, nil
}

func (d SprintAdmissionDecision) WithOverride(override *models.SprintAdmissionOverride) SprintAdmissionDecision {
	if override != nil && d.State == SprintAdmissionBlocked {
		d.State = SprintAdmissionOverridden
	}
	return d
}
