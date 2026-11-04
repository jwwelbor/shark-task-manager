package gatepersist

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

func bugAdmissionRequest(t *testing.T, root, runID, summary string) Request {
	t.Helper()
	dir, err := gaterun.RunDir(root, runID)
	require.NoError(t, err)
	result := &gateresult.GateResult{SchemaVersion: gateresult.SchemaVersion1, Summary: summary}
	envelope, err := json.Marshal(map[string]any{"kind": "final", "recommended_outcome": "pass", "gate_result": result})
	require.NoError(t, err)
	return Request{RunDir: dir, RunID: runID, EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: "plan_review", Gate: "plan_review", Session: Session{ID: "same-session"}, EnvelopeJSON: envelope, Result: result, Role: gateresult.RoleSuccess, OutcomeKey: "pass", TargetStatus: "development"}
}

func TestBoundAdmissionRejectsOtherRunsTargetTransition(t *testing.T) {
	root := t.TempDir()
	world := newFakeWorld()
	world.setStatus(models.EntityTypeBug, "B001", "plan_review")
	coord := newTestCoordinator(world, newFakeStatusValidator().allow(models.EntityTypeBug, "plan_review", "development"))
	b := bugAdmissionRequest(t, root, "run-b", "plan B")
	a := bugAdmissionRequest(t, root, "run-a", "plan A")
	calls := map[string]int{}
	coord.Admission = func(_ context.Context, r Request) error {
		calls[r.RunID]++
		if r.RunID == "run-b" && calls[r.RunID] == 2 {
			return errors.New("plan B changed")
		}
		return nil
	}
	_, err := coord.Persist(context.Background(), b)
	require.ErrorContains(t, err, "plan B changed")
	_, err = coord.Persist(context.Background(), a)
	require.NoError(t, err)
	_, err = coord.Persist(context.Background(), b)
	require.ErrorContains(t, err, "does not belong to this run")
	state, exists, err := gaterun.LoadOperationState(b.RunDir)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, gaterun.PersistenceStateComplete, state.PersistenceState)
}

func TestBoundAdmissionAllowsItsOwnCrashAfterTransition(t *testing.T) {
	root := t.TempDir()
	world := newFakeWorld()
	world.setStatus(models.EntityTypeBug, "B001", "plan_review")
	world.failTransitionTo["bug|B001|development"] = true
	coord := newTestCoordinator(world, newFakeStatusValidator().allow(models.EntityTypeBug, "plan_review", "development"))
	req := bugAdmissionRequest(t, root, "crash-run", "plan A")
	coord.Admission = func(context.Context, Request) error { return nil }
	_, err := coord.Persist(context.Background(), req)
	require.Error(t, err)
	coord.Admission = func(context.Context, Request) error { return errors.New("plan changed after transition") }
	result, err := coord.Persist(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.TransitionApplied)
}
