package runner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

func TestBugPlanAdmissionThroughGateIngestion(t *testing.T) {
	root := t.TempDir()
	plan := []byte("accepted plan revision\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plan.md"), plan, 0o600))
	digest := sha256.Sum256(plan)
	cases := []struct {
		name, evidence string
		allowed        bool
	}{
		{"missing", `[]`, false},
		{"stale", `[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}]`, false},
		{"escaping", fmt.Sprintf(`[{"kind":"bug_repair_plan","pointer":"../plan.md","summary":"sha256:%x"}]`, digest), false},
		{"missing baseline", fmt.Sprintf(`[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"}]`, digest), false},
		{"current", fmt.Sprintf(`[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]`, digest), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := "B001"
			status := &fakeTransitioner{status: map[string]string{key: "plan_review"}}
			notes := &fakeNoteWriter{}
			coord := gatepersist.NewCoordinator(notes, fakeNoteReader{}, fakeHistoryReader{}, fakeStatusValidator{valid: map[string]bool{"plan_review": true, "development": true}}, status, status, &fakeLeaseReleaser{}, status)
			coord.Admission = BugGateAdmission
			envelope := []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":%s,"gate_result":{"schema_version":1,"summary":"review complete"}}`, tc.evidence))
			res, err := IngestGateResult(context.Background(), GateIngestRequest{EnvelopeBytes: envelope, Coordinator: coord, ProjectRoot: root, RunID: "run-" + tc.name, EntityKey: key, EntityType: models.EntityTypeBug, SourceStatus: "plan_review", Gate: "plan_review", Session: gatepersist.Session{ID: "parent"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}, Outcomes: map[string]string{"pass": "development"}})
			if !tc.allowed {
				require.Error(t, err)
				require.Empty(t, notes.notes)
				require.Equal(t, "plan_review", status.status[key])
				return
			}
			require.NoError(t, err)
			require.Equal(t, "development", res.ToStatus)
		})
	}
}

func TestBugAdmissionDoesNotBlockReworkOrOtherEntities(t *testing.T) {
	admit := NewBugGateAdmission(nil, nil)
	for _, req := range []gatepersist.Request{
		{EntityType: models.EntityTypeBug, Gate: "plan_review", Role: gateresult.RoleRouteRework},
		{EntityType: models.EntityTypeBug, Gate: "qa", Role: gateresult.RoleBlocked},
		{EntityType: models.EntityTypeTask, Gate: "code_review", Role: gateresult.RoleSuccess},
	} {
		require.NoError(t, admit(context.Background(), req))
	}
}
