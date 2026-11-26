package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
	"github.com/stretchr/testify/require"
)

// Exercise current ingestion and a separately seeded QA history with two
// raw blockers whose reconciled decisions each require a host observation.
func testReportOnlyObservations(t *testing.T, root string, scope bugCandidateScope, originalReport string, originalAngles []byte, scopeRefs string, planNotes bugApprovalNotes, planHistory bugApprovalHistory) {
	t.Helper()
	// Review artifacts live in the worker checkout. Only the parent run owns
	// the action observation and gate history, including when QA replays them.
	parent := filepath.Join(t.TempDir(), "parent")
	out, err := exec.Command("git", "-C", root, "worktree", "add", "--detach", parent, "HEAD").CombinedOutput()
	require.NoError(t, err, string(out))
	planRun := filepath.Join(".shark", "runs", "approved-plan")
	require.NoError(t, os.CopyFS(filepath.Join(parent, planRun), os.DirFS(filepath.Join(root, planRun))))
	identity := bugReviewIdentity{scope.CandidateCommit, scope.CandidateTree, scope.DiffSHA256, scope.RepairPlanSHA256}
	var angles map[string]bugAngle
	require.NoError(t, json.Unmarshal(originalAngles, &angles))
	packet := bugReviewDecisions{CandidateIdentity: identity, ReviewedFiles: scope.ChangedFiles}
	for index, label := range []string{"a", "b"} {
		path := []string{"change.txt", "leaf.txt"}[index]
		raw := bugAngleFinding{Severity: "blocker", File: reviewGitPath(path), Line: 1, Rule: "STANDARD", Summary: "Label spelling", Diagnosis: "Claimed mandatory spelling"}
		angle := angles[label]
		angle.Findings = []bugAngleFinding{raw}
		angles[label] = angle
		packet.Findings = append(packet.Findings, bugReviewDecision{
			FindingID: fmt.Sprintf("F-%d", index+1), File: raw.File, Line: raw.Line, Rule: raw.Rule,
			Summary: raw.Summary, Diagnosis: raw.Diagnosis, Severity: "nit", Disposition: "report_only", Reason: "The guide permits both spellings.",
		})
	}
	save := func(project, path, kind string, data []byte) string {
		full := filepath.Join(project, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0700))
		require.NoError(t, os.WriteFile(full, data, 0600))
		return fmt.Sprintf(`{"kind":%q,"pointer":%q,"summary":"sha256:%x"}`, kind, path, sha256.Sum256(data))
	}
	angleData := mustJSON(t, angles)
	angleRef := save(root, ".shark/triage-angles.json", "bug_review_angles", angleData)
	header, _, ok := strings.Cut(originalReport, "\n---\n")
	require.True(t, ok)
	header = strings.Replace(header, fmt.Sprintf("%x", sha256.Sum256(originalAngles)), fmt.Sprintf("%x", sha256.Sum256(angleData)), 1)
	header = strings.Replace(header, "**Verdict:** PASS", "**Verdict:** PASS-WITH-TRIAGE", 1)
	oldDecisions, err := os.ReadFile(filepath.Join(root, ".shark/report.md.decisions.json"))
	require.NoError(t, err)
	envelope := func(refs string) []byte {
		return []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[%s],"gate_result":{"schema_version":1,"summary":"checked"}}`, refs))
	}

	for _, trial := range []triageObservationTrial{
		{name: "observed-zeros"},
		{name: "qa-other-pointer"},
		{name: "qa-other-digest"},
		{name: "qa-extra-observation"},
		{name: "worker-copy-only", want: "open evidence"},
		{name: "other-parent-run", want: "original review run"},
		{name: "hyphenated-withdrawal"},
		{name: "omitted-hyphenated-observation", want: "missing report-only observation for F-2"},
		{"absent-edits", "edits", "omit", "F-1: report-only requires observed edits of zero"},
		{"null-edits", "edits", "null", "F-1: report-only requires observed edits of zero"},
		{"nonzero-edits", "edits", "one", "F-1: report-only requires observed edits of zero"},
		{"absent-writes", "tracker_writes", "omit", "F-1: report-only requires observed tracker_writes of zero"},
		{"null-writes", "tracker_writes", "null", "F-1: report-only requires observed tracker_writes of zero"},
		{"nonzero-writes", "tracker_writes", "one", "F-1: report-only requires observed tracker_writes of zero"},
		{name: "shared-decision-id", want: "duplicate report-only finding ID F-1"},
		{name: "empty-decision-id", want: "report-only finding ID is required"},
		{name: "missing-peer", want: "missing report-only observation for F-2"},
		{name: "duplicated-observation", want: "duplicate report-only observation for F-1"},
	} {
		t.Run(trial.name, func(t *testing.T) {
			decisions, actions := trial.packetAndActions(packet)
			decisionData := mustJSON(t, decisions)
			decisionRef := save(root, ".shark/"+trial.name+"/decisions.json", "bug_review_decisions", decisionData)
			bound := strings.Replace(header, fmt.Sprintf("%x", sha256.Sum256(oldDecisions)), fmt.Sprintf("%x", sha256.Sum256(decisionData)), 1)
			report := bound + "\n---\nScope: all changed files. Findings: F-1 at change.txt:1 and F-2 at leaf.txt:1 alleged a mandatory spelling. The guide permits both at each site. Both withdrawals are report-only. Verdict: PASS-WITH-TRIAGE.\n"
			reportRef := save(root, ".shark/"+trial.name+"/report.md", "bug_review_report", []byte(report))
			observation := mustJSON(t, map[string]any{"candidate_identity": identity, "observed_by": "parent", "actions": actions})
			reviewID := "observation-" + trial.name
			saveObservation := func(runID string) string {
				project := parent
				switch trial.name {
				case "worker-copy-only":
					project = root
				case "other-parent-run":
					runID += "-other"
				}
				return save(project, ".shark/runs/"+runID+"/parent-triage-observation.json", "bug_parent_triage_observation", observation)
			}
			observationRef := saveObservation(reviewID)
			sharedRefs := scopeRefs + "," + observationRef
			prior := envelope(strings.Join([]string{sharedRefs, reportRef, angleRef, decisionRef}, ","))
			gates := []string{"code_review", "qa"}
			if trial.name == "observed-zeros" || trial.name == "qa-other-pointer" {
				gates = append(gates, "qa-blocked")
			}
			for _, entry := range gates {
				t.Run(entry, func(t *testing.T) {
					gate := entry
					blocked := entry == "qa-blocked"
					if blocked {
						gate = "qa"
					}
					notes, history := planNotes, planHistory
					run, target, result := reviewID, "qa", prior
					if gate == "qa" {
						run, target = reviewID+"-qa", "resolved"
						// Current ingestion has its own operation directory. Seed QA
						// independently using copies of the original gate artifacts.
						if blocked {
							run += "-blocked"
						}
						seedID := run + "-prior"
						seedRef := saveObservation(seedID)
						seeded := []byte(strings.Replace(string(prior), observationRef, seedRef, 1))
						seedMaterialReviewAt(t, parent, root, seedID, seeded, &notes, &history)
						qaRefs := scopeRefs + "," + seedRef
						switch trial.name {
						case "qa-other-pointer":
							qaRefs = scopeRefs + "," + observationRef
						case "qa-other-digest":
							qaRefs = strings.Replace(qaRefs, fmt.Sprintf("%x", sha256.Sum256(observation)), strings.Repeat("0", 64), 1)
						case "qa-extra-observation":
							qaRefs += "," + observationRef
						}
						result = envelope(qaRefs)
						if blocked {
							target = "blocked"
							result = []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"blocked","evidence":[%s],"gate_result":{"schema_version":1,"summary":"QA input unavailable","no_kickback_reason":"Missing external input"}}`, qaRefs))
						}
						t.Cleanup(func() {
							stored, exists, err := gaterun.ReadResult(filepath.Join(parent, ".shark/runs", seedID))
							require.NoError(t, err)
							require.True(t, exists)
							require.Equal(t, seeded, stored)
							var ref workercontrol.EvidenceRef
							require.NoError(t, json.Unmarshal([]byte(seedRef), &ref))
							source := parent
							if trial.name == "worker-copy-only" {
								source = root
							}
							after, err := os.ReadFile(filepath.Join(source, ref.Pointer))
							require.NoError(t, err)
							require.Equal(t, observation, after)
						})
					}
					passTarget := target
					if blocked {
						passTarget = "resolved"
					}
					status := &fakeTransitioner{status: map[string]string{"B001": gate}}
					written := &fakeNoteWriter{}
					coord := gatepersist.NewCoordinator(written, fakeNoteReader{}, fakeHistoryReader{}, fakeStatusValidator{valid: map[string]bool{target: true, gate: true}}, status, status, &fakeLeaseReleaser{}, status)
					coord.Admission = NewBugGateAdmission(notes, history)
					_, err := IngestGateResult(context.Background(), GateIngestRequest{
						ProjectRoot: parent, WorkingDir: root, RunID: run, Gate: gate, SourceStatus: gate, Coordinator: coord, EnvelopeBytes: result,
						EntityKey: "B001", EntityType: models.EntityTypeBug, Session: gatepersist.Session{ID: "parent"},
						Outcomes: map[string]string{"pass": passTarget, "blocked": "blocked"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess, "blocked": gateresult.RoleBlocked},
						ParentObservationInjected: gate == "code_review",
					})
					want := trial.want
					if gate == "qa" && strings.HasPrefix(trial.name, "qa-") {
						want = "bug_parent_triage_observation"
					}
					if want != "" {
						require.ErrorContains(t, err, want)
						require.Empty(t, written.notes)
						require.Equal(t, gate, status.status["B001"])
						return
					}
					require.NoError(t, err)
					require.Len(t, written.notes, 1)
					require.Equal(t, target, status.status["B001"])
					stored, exists, err := gaterun.ReadResult(filepath.Join(parent, ".shark/runs", run))
					require.NoError(t, err)
					require.True(t, exists)
					require.Equal(t, result, stored)
				})
			}
		})
	}
}

type triageObservationTrial struct {
	name, field, mutation, want string
}

func (trial triageObservationTrial) packetAndActions(packet bugReviewDecisions) (bugReviewDecisions, []map[string]any) {
	decisions := packet
	decisions.Findings = append([]bugReviewDecision(nil), packet.Findings...)
	actions := []map[string]any{
		{"finding_id": "F-1", "action": "report_only", "edits": 0, "tracker_writes": 0},
		{"finding_id": "F-2", "action": "report_only", "edits": 0, "tracker_writes": 0},
	}
	switch trial.name {
	case "hyphenated-withdrawal", "omitted-hyphenated-observation":
		decisions.Findings[1].Disposition = "report-only"
		if trial.name == "omitted-hyphenated-observation" {
			actions = actions[:1]
		}
	case "shared-decision-id":
		decisions.Findings[1].FindingID = "F-1"
		actions = actions[:1]
	case "empty-decision-id":
		decisions.Findings[0].FindingID = ""
		actions[0]["finding_id"] = ""
	case "missing-peer":
		actions = actions[:1]
	case "duplicated-observation":
		actions[1] = actions[0]
	}
	switch trial.mutation {
	case "omit":
		delete(actions[0], trial.field)
	case "null":
		actions[0][trial.field] = nil
	case "one":
		actions[0][trial.field] = 1
	}
	return decisions, actions
}
