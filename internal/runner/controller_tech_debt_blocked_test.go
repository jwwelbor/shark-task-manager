// Covers T-E31-F01-009 / TC-020d: a tech-debt dispatch that returns "blocked"
// parks the item, and the run loop does not dispatch it again.
//
// Every status, action, outcome and outcome role below is read from the
// embedded tech-debt workflow, so a route change in tech-debt.yaml changes
// what these fakes do. Only the worker's own reply is supplied by the test.
package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// embeddedTransitioner serves the controller from a workflow service and shares
// one status map with the gate coordinator, so a second Run sees the status the
// first one wrote.
type embeddedTransitioner struct {
	*fakeTransitioner
	workflowSvc *workflow.Service
}

func (t embeddedTransitioner) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	status := t.workflowSvc.NormalizeStatus(t.status[key])
	var transitions []services.TransitionInfoWithAction
	for _, info := range t.workflowSvc.GetTransitionInfo(status) {
		transitions = append(transitions, services.TransitionInfoWithAction{TransitionInfo: info})
	}
	return &services.NextStatusInfo{
		CurrentStatus:        status,
		AvailableTransitions: transitions,
		IsTerminal:           t.workflowSvc.IsTerminalStatus(status),
		Outcomes:             t.workflowSvc.GetOutcomes(status),
		ResultContract:       t.workflowSvc.GetResultContract(status),
		OutcomeRoles:         t.workflowSvc.GetOutcomeRoles(status),
	}, nil
}

func (t embeddedTransitioner) TransitionStatus(_ context.Context, key, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
	from := t.status[key]
	t.status[key] = target
	return &services.TransitionResult{FromStatus: from, ToStatus: target}, nil
}

type embeddedStatusValidator struct{ workflowSvc *workflow.Service }

func (v embeddedStatusValidator) IsValidStatus(_ models.EntityType, status string) bool {
	return v.workflowSvc.IsValidStatus(status)
}

func embeddedActionService(workflowSvc *workflow.Service) *MockActionService {
	return &MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*config.PopulatedAction, error) {
			meta, ok := workflowSvc.GetWorkflow().GetStatusMetadata(status)
			if !ok || meta.OrchestratorAction == nil {
				return nil, nil
			}
			action := meta.OrchestratorAction
			return &config.PopulatedAction{
				Action:      action.Action,
				AgentType:   action.AgentType,
				Provider:    action.Provider,
				Model:       action.Model,
				Instruction: action.InstructionTemplate,
			}, nil
		},
	}
}

// blockingDispatcher replies "blocked" and counts dispatches. It errors after a
// few calls so that a workflow that loops fails instead of running on.
func blockingDispatcher(dispatched *int) *MockDispatcher {
	const maxDispatches = 3
	return &MockDispatcher{DispatchFunc: func(context.Context, DispatchInput) (*DispatchResult, error) {
		*dispatched++
		if *dispatched > maxDispatches {
			return nil, errors.New("item was dispatched repeatedly instead of parking")
		}
		return &DispatchResult{ExitCode: 0, Stdout: `{"kind": "final", "recommended_outcome": "blocked",` +
			` "evidence": [], "gate_result": {"schema_version": 1, "summary": "blocked",` +
			` "no_kickback_reason": "no quality gate commands are declared"}}`}, nil
	}}
}

func newTechDebtParkingController(t *testing.T, items *fakeTransitioner, dispatched *int) *RunController {
	t.Helper()
	workflowSvc := workflow.NewService("")
	techDebt := workflowSvc.ForLevel("tech_debt")
	transitioner := embeddedTransitioner{fakeTransitioner: items, workflowSvc: techDebt}
	coordinator := gatepersist.NewCoordinator(
		&fakeNoteWriter{}, fakeNoteReader{}, fakeHistoryReader{},
		embeddedStatusValidator{workflowSvc: techDebt},
		transitioner, transitioner, &fakeLeaseReleaser{}, transitioner,
	)
	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    embeddedActionService(techDebt),
		WorkflowSvc:  workflowSvc,
		Dispatchers:  map[string]AgentDispatcher{"anthropic": blockingDispatcher(dispatched)},
		GateIngest:   &GateIngestDeps{Coordinator: coordinator},
	})
	require.NoError(t, err)
	return ctrl
}

func TestE31F01_TC020d_BlockedTechDebtParksWithoutRedispatch(t *testing.T) {
	for _, from := range []string{"triaged", "in_progress"} {
		t.Run(from, func(t *testing.T) {
			const key = "TD-001"
			items := &fakeTransitioner{status: map[string]string{key: from}}
			dispatched := 0
			ctrl := newTechDebtParkingController(t, items, &dispatched)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			opts := RunOptions{ProjectRoot: t.TempDir(), RunID: "run-td-park-" + from, SessionID: "sess-1", EntityType: "tech_debt"}

			first, err := ctrl.Run(ctx, key, opts)
			require.NoError(t, err)
			assert.Equal(t, "paused", first.Outcome, "run error: %s", first.Error)
			assert.Equal(t, "blocked", first.FinalStatus)
			assert.Equal(t, "blocked", items.status[key])
			assert.Equal(t, 1, dispatched)

			opts.RunID += "-again"
			second, err := ctrl.Run(ctx, key, opts)
			require.NoError(t, err)
			assert.Equal(t, "paused", second.Outcome, "run error: %s", second.Error)
			assert.Equal(t, "blocked", items.status[key])
			assert.Equal(t, 1, dispatched, "a parked item must not be dispatched again")
		})
	}
}
