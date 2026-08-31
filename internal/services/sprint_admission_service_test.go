package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSprintAdmissionEvidenceReader struct {
	evidence *SprintAdmissionEvidence
	err      error
}

func (r stubSprintAdmissionEvidenceReader) ReadSprintAdmissionEvidence(context.Context) (*SprintAdmissionEvidence, error) {
	return r.evidence, r.err
}

func TestSprintAdmissionService_EvaluateBlocksUnmetAncestor(t *testing.T) {
	service := NewSprintAdmissionService(stubSprintAdmissionEvidenceReader{evidence: &SprintAdmissionEvidence{
		PortfolioEpicKey: "E01",
		Candidates: map[string]SprintAdmissionCandidate{
			"T-E02-F01-001": {Key: "T-E02-F01-001", EpicKey: "E02"},
		},
		UnmetAncestors: map[string][]string{"E02": {"E01"}},
	}})

	decision, err := service.Evaluate(context.Background(), "T-E02-F01-001")

	require.NoError(t, err)
	assert.Equal(t, SprintAdmissionBlocked, decision.State)
	assert.Equal(t, SprintAdmissionReasonAncestorDependency, decision.ReasonCode)
	assert.Equal(t, []string{"E01"}, decision.UnmetAncestorKeys)
}

func TestSprintAdmissionService_EvaluateBlocksCandidateOutsidePortfolio(t *testing.T) {
	service := NewSprintAdmissionService(stubSprintAdmissionEvidenceReader{evidence: &SprintAdmissionEvidence{
		PortfolioEpicKey: "E01",
		Candidates:       map[string]SprintAdmissionCandidate{"T-E02-F01-001": {Key: "T-E02-F01-001", EpicKey: "E02"}},
		UnmetAncestors:   map[string][]string{},
	}})

	decision, err := service.Evaluate(context.Background(), "T-E02-F01-001")

	require.NoError(t, err)
	assert.Equal(t, SprintAdmissionBlocked, decision.State)
	assert.Equal(t, SprintAdmissionReasonOutsidePortfolio, decision.ReasonCode)
}

func TestSprintAdmissionService_EvaluateFailsClosedWhenEvidenceUnavailable(t *testing.T) {
	service := NewSprintAdmissionService(stubSprintAdmissionEvidenceReader{err: errors.New("database unavailable")})

	_, err := service.Evaluate(context.Background(), "T-E02-F01-001")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "admission evidence")
}

func TestValidSprintOverrideReasonBoundaries(t *testing.T) {
	assert.False(t, validSprintOverrideReason("                   "))
	assert.False(t, validSprintOverrideReason("1234567890123456789"))
	assert.True(t, validSprintOverrideReason("12345678901234567890"))
	assert.True(t, validSprintOverrideReason(strings.Repeat("x", 500)))
	assert.False(t, validSprintOverrideReason(strings.Repeat("x", 501)))
}

func TestSprintService_AddEntityToSprintRejectsBlockedCandidateBeforeAssignment(t *testing.T) {
	added := false
	repo := &MockSprintRepository{
		GetByKeyFunc: func(context.Context, string) (*models.Sprint, error) {
			return &models.Sprint{ID: 1, Key: "S001", Status: "planning"}, nil
		},
		GetTaskIDByKeyFunc: func(context.Context, string) (int64, error) { return 10, nil },
		AddAssignmentFunc: func(context.Context, *models.SprintAssignment) error {
			added = true
			return nil
		},
	}
	service := NewSprintService(repo, workflow.NewService(""), nil, nil, nil)
	service.SetAdmissionService(NewSprintAdmissionService(stubSprintAdmissionEvidenceReader{evidence: &SprintAdmissionEvidence{
		PortfolioEpicKey: "E01",
		Candidates:       map[string]SprintAdmissionCandidate{"T-E02-F01-001": {Key: "T-E02-F01-001", EpicKey: "E02"}},
		UnmetAncestors:   map[string][]string{"E02": {"E01"}},
	}}))

	_, _, err := service.AddEntityToSprint(context.Background(), AddEntityInput{SprintKey: "S001", EntityKey: "T-E02-F01-001"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), string(SprintAdmissionReasonAncestorDependency))
	assert.False(t, added)
}

// TC-006: selection and the sprint-next compatibility projection must omit a
// blocked first-ranked candidate through the shared admission evaluator.
func TestSelectSprint_TC006_SkipsRoadmapBlockedCandidate(t *testing.T) {
	order1, order2 := 1, 2
	svc := newGetNextTaskTestService(t, []*sprint.BacklogItem{
		{EntityType: "task", Key: "task-A", Title: "Blocked", Status: "todo", SprintOrder: &order1, AssignedAt: time.Now()},
		{EntityType: "task", Key: "task-B", Title: "Allowed", Status: "todo", SprintOrder: &order2, AssignedAt: time.Now()},
	})
	svc.SetAdmissionService(NewSprintAdmissionService(stubSprintAdmissionEvidenceReader{evidence: &SprintAdmissionEvidence{
		PortfolioEpicKey: "E01",
		Candidates: map[string]SprintAdmissionCandidate{
			"task-A": {Key: "task-A", EpicKey: "E02"},
			"task-B": {Key: "task-B", EpicKey: "E01"},
		},
		UnmetAncestors: map[string][]string{"E02": {"E01"}},
	}}))

	selection, err := svc.SelectSprint(context.Background(), SprintSelectionInput{SprintKey: "S001", Limit: 2})

	require.NoError(t, err)
	require.Len(t, selection.Items, 1)
	assert.Equal(t, "task-B", selection.Items[0].Key)
}

// TC-005: a blocked assigned item is a circuit breaker; otherwise healthy
// capacity, count, sizing, and agent factors cannot offset it.
func TestSprintService_GetSprintReadiness_TC005_BlockedAdmissionForcesOverallZero(t *testing.T) {
	ctx := context.Background()
	svc := makeReadinessSvc(&models.Sprint{ID: 24, Key: "S024", Status: "planning"}, []sprint.AssignmentWithSize{
		{EntityType: "task", Key: "task-blocked", Title: "Blocked", Size: sizePtrR(5), AgentType: agentPtrR("backend")},
		{EntityType: "task", Key: "task-healthy-1", Title: "Healthy 1", Size: sizePtrR(5), AgentType: agentPtrR("backend")},
		{EntityType: "task", Key: "task-healthy-2", Title: "Healthy 2", Size: sizePtrR(5), AgentType: agentPtrR("frontend")},
		{EntityType: "task", Key: "task-healthy-3", Title: "Healthy 3", Size: sizePtrR(5), AgentType: agentPtrR("frontend")},
	}, []*models.SprintCapacity{{SprintID: 24, AgentType: "backend", CapacityPoints: 10}, {SprintID: 24, AgentType: "frontend", CapacityPoints: 10}})
	svc.SetAdmissionService(NewSprintAdmissionService(stubSprintAdmissionEvidenceReader{evidence: &SprintAdmissionEvidence{
		PortfolioEpicKey: "E01",
		Candidates: map[string]SprintAdmissionCandidate{
			"task-blocked":   {Key: "task-blocked", EpicKey: "E02"},
			"task-healthy-1": {Key: "task-healthy-1", EpicKey: "E01"},
			"task-healthy-2": {Key: "task-healthy-2", EpicKey: "E01"},
			"task-healthy-3": {Key: "task-healthy-3", EpicKey: "E01"},
		},
		UnmetAncestors: map[string][]string{"E02": {"E01"}},
	}}))

	readiness, err := svc.GetSprintReadiness(ctx, "S024")

	require.NoError(t, err)
	assert.Equal(t, 0, readiness.OverallScore)
	require.Len(t, readiness.Factors, 7)
	assert.Equal(t, "Roadmap admission", readiness.Factors[6].Name)
	assert.Equal(t, 0, readiness.Factors[6].Score)
	assert.Contains(t, readiness.Factors[6].Detail, "task-blocked")
}

// TestPortfolioSprintAdmissionEvidenceReader_NoEligibleEpicAllowsAllCandidates
// covers the gap the reviewer flagged: no test exercised the real evidence
// reader with anything but the one-root happy path. With zero eligible
// portfolio roots (e.g. both epics terminal), the planner reports
// PauseReason "no_eligible_epic" and RootKeys == []. That is a normal
// between-epics state, not an error, so the reader must return usable
// evidence with an empty PortfolioEpicKey rather than failing every caller
// (shark sprint add/next/readiness/plan).
func TestPortfolioSprintAdmissionEvidenceReader_NoEligibleEpicAllowsAllCandidates(t *testing.T) {
	source := &stubPortfolioSnapshotSource{snapshot: portfoliorepo.Snapshot{
		Epics: []*models.Epic{
			portfolioTestEpic(1, "E01", "Shipped one", "shipped_custom", models.PriorityHigh, nil),
			portfolioTestEpic(2, "E02", "Shipped two", "shipped_custom", models.PriorityHigh, nil),
		},
	}}
	reader := newTestPortfolioSprintAdmissionEvidenceReader(t, source)

	evidence, err := reader.ReadSprintAdmissionEvidence(context.Background())

	require.NoError(t, err)
	require.NotNil(t, evidence)
	assert.Equal(t, "", evidence.PortfolioEpicKey)

	// With no active portfolio gate, a candidate under any epic is not
	// blocked for being "outside the portfolio" — only unmet ancestor
	// dependencies still block.
	decision, err := evaluateSprintAdmissionEvidence(&SprintAdmissionEvidence{
		PortfolioEpicKey: evidence.PortfolioEpicKey,
		Candidates:       map[string]SprintAdmissionCandidate{"T-E09-F01-001": {Key: "T-E09-F01-001", EpicKey: "E09"}},
		UnmetAncestors:   map[string][]string{},
	}, "T-E09-F01-001")
	require.NoError(t, err)
	assert.Equal(t, SprintAdmissionAllowed, decision.State)
}

// TestPortfolioSprintAdmissionEvidenceReader_TieResolvesToLowestKey covers
// the parallel-tie case: multiple equally-eligible, equally-prioritized
// epics produce PortfolioPlan.RootKeys with len > 1 ("parallel_tie"). The
// reader must not error; it resolves deterministically to the
// lexicographically lowest tied root key.
func TestPortfolioSprintAdmissionEvidenceReader_TieResolvesToLowestKey(t *testing.T) {
	source := &stubPortfolioSnapshotSource{snapshot: portfoliorepo.Snapshot{
		Epics: []*models.Epic{
			portfolioTestEpic(3, "E03", "Third", "active_custom", models.PriorityHigh, nil),
			portfolioTestEpic(2, "E02", "Second", "active_custom", models.PriorityHigh, nil),
		},
	}}
	reader := newTestPortfolioSprintAdmissionEvidenceReader(t, source)

	evidence, err := reader.ReadSprintAdmissionEvidence(context.Background())

	require.NoError(t, err)
	require.NotNil(t, evidence)
	assert.Equal(t, "E02", evidence.PortfolioEpicKey)
}

// TestPortfolioSprintAdmissionEvidenceReader_SingleRootHappyPath is the
// existing one-root case, exercised through the real advisor/planner
// pipeline (not a stub evidence reader) so it sits alongside the zero- and
// tie-root coverage above.
func TestPortfolioSprintAdmissionEvidenceReader_SingleRootHappyPath(t *testing.T) {
	source := &stubPortfolioSnapshotSource{snapshot: portfoliorepo.Snapshot{
		Epics: []*models.Epic{
			portfolioTestEpic(1, "E01", "Only eligible epic", "active_custom", models.PriorityHigh, nil),
		},
	}}
	reader := newTestPortfolioSprintAdmissionEvidenceReader(t, source)

	evidence, err := reader.ReadSprintAdmissionEvidence(context.Background())

	require.NoError(t, err)
	require.NotNil(t, evidence)
	assert.Equal(t, "E01", evidence.PortfolioEpicKey)
}

func newTestPortfolioSprintAdmissionEvidenceReader(
	t *testing.T,
	source *stubPortfolioSnapshotSource,
) *PortfolioSprintAdmissionEvidenceReader {
	t.Helper()
	workflows := portfolioTestWorkflows()
	advisor := NewPortfolioAdviceServiceFromSnapshot(source, &stubPortfolioClaimFilter{}, workflows)
	planner := NewPortfolioPlanningService()
	return NewPortfolioSprintAdmissionEvidenceReader(source, advisor, planner, workflows)
}
