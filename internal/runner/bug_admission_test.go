package runner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

func TestBugPlanAdmissionThroughGateIngestion(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	git("init")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	git("commit", "--allow-empty", "-m", "baseline")
	baseline := git("rev-parse", "HEAD")
	plan := []byte("accepted plan revision\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plan.md"), plan, 0o600))
	blob := git("hash-object", "-w", "plan.md")
	digest := sha256.Sum256(plan)
	cases := []struct {
		name, evidence string
		allowed        bool
	}{
		{"missing", `[]`, false},
		{"stale", `[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}]`, false},
		{"escaping", fmt.Sprintf(`[{"kind":"bug_repair_plan","pointer":"../plan.md","summary":"sha256:%x"}]`, digest), false},
		{"missing baseline", fmt.Sprintf(`[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"}]`, digest), false},
		{"unknown commit", fmt.Sprintf(`[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]`, digest), false},
		{"blob is not a commit", fmt.Sprintf(`[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"%s"}]`, digest, blob), false},
		{"current", fmt.Sprintf(`[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"%s"}]`, digest, baseline), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := "B001"
			status := &fakeTransitioner{status: map[string]string{key: "plan_review"}}
			notes := &fakeNoteWriter{}
			coord := gatepersist.NewCoordinator(notes, fakeNoteReader{}, fakeHistoryReader{}, fakeStatusValidator{valid: map[string]bool{"plan_review": true, "development": true}}, status, status, &fakeLeaseReleaser{}, status)
			coord.Admission = BugGateAdmission
			coord.BeforeTransition = PrepareBugApprovalSnapshot
			if tc.name == "current" {
				coord.BeforeTransition = func(ctx context.Context, req gatepersist.Request) error {
					if err := PrepareBugApprovalSnapshot(ctx, req); err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(root, "plan.md"), []byte("changed after snapshot"), 0o600)
				}
			}
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
			retained, exists, err := gaterun.ReadApprovedPlan(filepath.Join(root, ".shark", "runs", "run-current"))
			require.NoError(t, err)
			require.True(t, exists)
			require.Equal(t, plan, retained)
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
