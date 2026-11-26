package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
	"github.com/stretchr/testify/require"
)

func filenameTestEnvelope(t *testing.T, refs []workercontrol.EvidenceRef) []byte {
	t.Helper()
	raw, err := json.Marshal(refs)
	require.NoError(t, err)
	return []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":%s,"gate_result":{"schema_version":1,"summary":"approved"}}`, raw))
}

// Seed the exact review independently so a current-admission rejection cannot
// mask the QA reader's behavior on the same correctly hashed packet.
func seedFilenameReview(t *testing.T, root, run string, result []byte, notes *bugApprovalNotes, history *bugApprovalHistory) {
	t.Helper()
	gate := "code_review"
	digest, err := gaterun.ComputeOperationDigest("B001", "bug", gate, gate, result)
	require.NoError(t, err)
	dir, err := gaterun.RunDir(root, run)
	require.NoError(t, err)
	_, err = gaterun.CreateIdentity(dir, gaterun.RunIdentity{RunID: run, EntityKey: "B001", EntityType: "bug", SourceStatus: gate, Gate: gate, OperationDigest: digest})
	require.NoError(t, err)
	_, err = gaterun.CreateResult(dir, result)
	require.NoError(t, err)
	state := gaterun.NewOperationState(run, "B001", "bug", gate, gate, digest)
	state.AdmissionVersion = gatepersist.BoundAdmissionVersion
	require.NoError(t, state.MarkPersistenceComplete())
	require.NoError(t, state.MarkTransitionApplied())
	require.NoError(t, state.Save(dir))
	data, err := json.Marshal(bugApprovalNote{Gate: gate, Role: "success", Outcome: "pass", RunID: run, Digest: digest})
	require.NoError(t, err)
	meta := string(data)
	notes.items = append(notes.items, &models.EntityNote{ID: 2, Metadata: &meta})
	reason := gatepersist.BoundTransitionReason(gate, "pass", run, digest)
	history.items = append(history.items, &models.EntityHistory{ID: 2, FromStatus: &gate, ToStatus: "qa", Notes: &reason})
}

func TestBugReviewFilenameAdmission(t *testing.T) {
	runReviewAdmissionCases(t, "filename", 27)
}

func TestGroupedSourceAdmission(t *testing.T) {
	runReviewAdmissionCases(t, "grouped", 22)
}

func runReviewAdmissionCases(t *testing.T, suite string, count int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX byte filenames")
	}
	root := t.TempDir()
	plan := []byte("approved plan\n")
	const planPath = "docs/plan/plan.md"
	commitTestBugPlan(t, root, planPath, plan)
	commitTestBugPlan(t, root, "context\xea.txt", []byte("unchanged reviewer context\n"))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git/info/exclude"), []byte(".shark/\n"), 0600))
	base, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	baseline := strings.TrimSpace(string(base))
	planDigest := fmt.Sprintf("%x", sha256.Sum256(plan))
	approved := []workercontrol.EvidenceRef{{Kind: "bug_repair_plan", Pointer: planPath, Summary: "sha256:" + planDigest}, {Kind: "bug_review_baseline", Pointer: baseline}}
	planNotes, planHistory := seedReviewTestPlan(t, root, filenameTestEnvelope(t, approved), plan)
	_, file, _, _ := runtime.Caller(0)
	scripts := filepath.Join(filepath.Dir(file), "../../skills/shark-rider/skills/deep-review/scripts")
	cmd := exec.Command("python3", filepath.Join(filepath.Dir(file), "testdata/filename_receipts.py"), root, scripts, baseline, planDigest, suite)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	var cases []struct {
		Name          string
		ExpectedError string `json:"expected_error"`
		Refs          []workercontrol.EvidenceRef
	}
	require.NoError(t, json.Unmarshal(output, &cases))
	require.Len(t, cases, count)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			notes, history := planNotes, planHistory
			review := filenameTestEnvelope(t, append(approved, tc.Refs...))
			seedFilenameReview(t, root, "prior-"+tc.Name, review, &notes, &history)
			var scope workercontrol.EvidenceRef
			for _, r := range tc.Refs {
				if r.Kind == "bug_review_scope" {
					scope = r
				}
			}
			require.NotEmpty(t, scope.Pointer)
			for _, gate := range []string{"code_review", "qa"} {
				t.Run(gate, func(t *testing.T) {
					result, target := review, "qa"
					gateNotes, gateHistory := planNotes, planHistory
					if gate == "qa" {
						result, target = filenameTestEnvelope(t, append(approved, scope)), "resolved"
						gateNotes, gateHistory = notes, history
					}
					status := &fakeTransitioner{status: map[string]string{"B001": gate}}
					written := &fakeNoteWriter{}
					coord := gatepersist.NewCoordinator(written, fakeNoteReader{}, fakeHistoryReader{}, fakeStatusValidator{valid: map[string]bool{gate: true, target: true}}, status, status, &fakeLeaseReleaser{}, status)
					coord.Admission = NewBugGateAdmission(gateNotes, gateHistory)
					_, err := IngestGateResult(context.Background(), GateIngestRequest{EnvelopeBytes: result, Coordinator: coord, ProjectRoot: root, RunID: gate + "-" + tc.Name, EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: gate, Gate: gate, Session: gatepersist.Session{ID: "parent"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}, Outcomes: map[string]string{"pass": target}})
					switch {
					case suite == "grouped" && tc.ExpectedError != "":
						require.ErrorContains(t, err, tc.ExpectedError)
						if gate == "qa" {
							require.ErrorContains(t, err, "prior code review evidence")
						}
					case suite == "grouped", strings.HasSuffix(tc.Name, "complete"), strings.HasSuffix(tc.Name, "restored"), strings.HasSuffix(tc.Name, "-optional-omitted"):
						require.NoError(t, err)
						require.Equal(t, target, status.status["B001"])
						require.Len(t, written.notes, 1)
						return
					case strings.Contains(tc.Name, "-scope-"):
						require.ErrorContains(t, err, "candidate manifest mismatch")
						require.ErrorContains(t, err, `"caf\udce9.txt"`)
					default:
						require.ErrorContains(t, err, `"caf\udce9.txt"`)
						if strings.Contains(tc.Name, "-angles-") {
							require.ErrorContains(t, err, "angle b")
						}
						if gate == "qa" {
							require.ErrorContains(t, err, "prior code review evidence")
						}
					}
					require.Equal(t, gate, status.status["B001"])
					require.Empty(t, written.notes)
				})
			}
		})
	}
}
