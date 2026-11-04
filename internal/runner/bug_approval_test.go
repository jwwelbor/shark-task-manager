package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

type bugApprovalNotes struct{ items []*models.EntityNote }

func (n bugApprovalNotes) ListNotes(context.Context, models.EntityType, string, []string) ([]*models.EntityNote, error) {
	return n.items, nil
}

type bugApprovalHistory struct{ items []*models.EntityHistory }

type bugApprovalDocuments struct{ items []*models.Document }

func (d bugApprovalDocuments) ListRelatedDocumentsByKey(context.Context, string) ([]*models.Document, error) {
	return d.items, nil
}

func TestPrepareBugPlanApprovalRequiresLinkedDigestBoundReport(t *testing.T) {
	root := t.TempDir()
	plan := []byte("reviewed plan")
	planHash := sha256.Sum256(plan)
	report := []byte(fmt.Sprintf("plan_path: docs/plan/plan.md\nplan_sha256: sha256:%x\nindependent review\n", planHash))
	commitTestBugPlan(t, root, "docs/plan/plan.md", plan)
	require.NoError(t, os.WriteFile(filepath.Join(root, "review.md"), report, 0600))
	reportHash := sha256.Sum256(report)
	dir, err := gaterun.RunDir(root, "approval")
	require.NoError(t, err)
	req := gatepersist.Request{EntityType: models.EntityTypeBug, EntityKey: "B001", Gate: "plan_review", Role: gateresult.RoleSuccess, RunDir: dir,
		EnvelopeJSON: []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"docs/plan/plan.md","summary":"sha256:%x"},{"kind":"plan_review_report","pointer":"review.md","summary":"sha256:%x"}],"gate_result":{"schema_version":1,"summary":"approved"}}`, planHash, reportHash))}
	req.PlanRoot, req.PlanCommit, req.PlanBlob, err = bugPlanProvenance(context.Background(), req)
	require.NoError(t, err)
	req.RunID = "approval"
	require.ErrorContains(t, PrepareBugPlanApproval(context.Background(), req, bugApprovalDocuments{}), "not linked")
	linked := bugApprovalDocuments{items: []*models.Document{{FilePath: "review.md"}}}
	staleReport := []byte("plan_path: docs/plan/old.md\nplan_sha256: sha256:0000000000000000000000000000000000000000000000000000000000000000\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "review.md"), staleReport, 0600))
	staleHash := sha256.Sum256(staleReport)
	staleReq := req
	staleReq.EnvelopeJSON = []byte(strings.Replace(string(req.EnvelopeJSON), fmt.Sprintf("%x", reportHash), fmt.Sprintf("%x", staleHash), 1))
	require.ErrorContains(t, PrepareBugPlanApproval(context.Background(), staleReq, linked), "identify exactly the approved plan")
	require.NoError(t, os.WriteFile(filepath.Join(root, "review.md"), []byte("changed"), 0600))
	require.ErrorContains(t, PrepareBugPlanApproval(context.Background(), req, linked), "digest is stale")
	require.NoError(t, os.WriteFile(filepath.Join(root, "review.md"), report, 0600))
	require.NoError(t, PrepareBugPlanApproval(context.Background(), req, linked))
	retained, exists, err := gaterun.ReadApprovedPlan(dir)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, plan, retained)
}

func validBugApprovalMode(mode string) bool {
	return mode == "valid" || mode == "renamed" || mode == "later_incomplete" || mode == "digest_case"
}

func bugApprovalEnvelopeCase(mode string, hash [32]byte, envelope []byte) []byte {
	if mode != "digest_case" {
		return envelope
	}
	digest := fmt.Sprintf("%x", hash)
	return []byte(strings.Replace(string(envelope), digest, strings.ToUpper(digest), 1))
}

func appendLaterIncompleteApproval(t *testing.T, mode string, notes *bugApprovalNotes) {
	t.Helper()
	if mode != "later_incomplete" {
		return
	}
	data, err := json.Marshal(map[string]string{"gate": "plan_review", "role": "success", "outcome_key": "pass", "run_id": "unapplied-run", "operation_digest": strings.Repeat("f", 64)})
	require.NoError(t, err)
	metadata := string(data)
	notes.items = append(notes.items, &models.EntityNote{ID: 2, Metadata: &metadata})
}

func (h bugApprovalHistory) GetHistory(context.Context, models.EntityType, string) ([]*models.EntityHistory, error) {
	return h.items, nil
}

func TestBugApprovalRequiresCommittedCurrentUninvalidatedReview(t *testing.T) {
	for _, mode := range []string{"valid", "renamed", "later_incomplete", "digest_case", "source_changed", "snapshot_tampered", "snapshot_missing", "baseline_changed", "missing", "unapproved", "invalidated", "partial", "tampered", "provenance_tampered"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			plan := []byte("plan v1\n")
			commitTestBugPlan(t, root, "docs/plan/plan.md", plan)
			hash := sha256.Sum256(plan)
			envelope := func(h [32]byte, outcome, baseline string) []byte {
				return []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"%s","evidence":[{"kind":"bug_repair_plan","pointer":"docs/plan/plan.md","summary":"sha256:%x"},{"kind":"bug_review_baseline","pointer":"%s"}],"gate_result":{"schema_version":1,"summary":"approved"}}`, outcome, h, baseline))
			}
			priorOutcome := "pass"
			if mode == "renamed" {
				priorOutcome = "approved"
			}
			baseline := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			prior := envelope(hash, priorOutcome, baseline)
			dir, err := gaterun.RunDir(root, "approved-run")
			require.NoError(t, err)
			lock, err := gaterun.AcquireRunLock(dir, 0)
			require.NoError(t, err)
			require.NoError(t, lock.Release())
			planRoot, planCommit, planBlob, err := bugPlanProvenance(context.Background(), gatepersist.Request{RunDir: dir, EnvelopeJSON: prior})
			require.NoError(t, err)
			digest, err := gaterun.ComputeBoundOperationDigest("B001", "bug", "plan_review", "plan_review", prior, planRoot, planCommit, planBlob)
			require.NoError(t, err)
			require.NoError(t, retainBugPlanCommit(context.Background(), gatepersist.Request{RunDir: dir, RunID: "approved-run", PlanCommit: planCommit}))
			_, err = gaterun.CreateIdentity(dir, gaterun.RunIdentity{RunID: "approved-run", EntityKey: "B001", EntityType: "bug", SourceStatus: "plan_review", Gate: "plan_review", OperationDigest: digest, PlanRoot: planRoot, PlanCommit: planCommit, PlanBlob: planBlob})
			require.NoError(t, err)
			_, err = gaterun.CreateResult(dir, prior)
			require.NoError(t, err)
			_, err = gaterun.CreateApprovedPlan(dir, plan)
			require.NoError(t, err)
			if mode == "provenance_tampered" {
				identityPath := filepath.Join(dir, "identity.json")
				data, readErr := os.ReadFile(identityPath)
				require.NoError(t, readErr)
				var identity map[string]any
				require.NoError(t, json.Unmarshal(data, &identity))
				identity["plan_commit"] = strings.Repeat("b", 40)
				data, err = json.Marshal(identity)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(identityPath, data, 0600))
			}
			if mode == "snapshot_tampered" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "approved-plan.md"), []byte("different plan"), 0o600))
			}
			if mode == "snapshot_missing" {
				require.NoError(t, os.Remove(filepath.Join(dir, "approved-plan.md")))
			}
			state := gaterun.NewOperationState("approved-run", "B001", "bug", "plan_review", "plan_review", digest)
			state.AdmissionVersion = gatepersist.BoundAdmissionVersion
			require.NoError(t, state.MarkPersistenceComplete())
			if mode != "partial" {
				require.NoError(t, state.MarkTransitionApplied())
			}
			require.NoError(t, state.Save(dir))
			if mode == "tampered" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "result.json"), []byte(`{"kind":"final"}`), 0o600))
			}
			metadata, err := json.Marshal(map[string]string{"gate": "plan_review", "role": "success", "outcome_key": priorOutcome, "run_id": "approved-run", "operation_digest": digest})
			require.NoError(t, err)
			meta := string(metadata)
			notes := bugApprovalNotes{items: []*models.EntityNote{{ID: 1, Metadata: &meta}}}
			appendLaterIncompleteApproval(t, mode, &notes)
			if mode == "missing" {
				notes.items = nil
			}
			from, reason := "plan_review", gatepersist.BoundTransitionReason("plan_review", priorOutcome, "approved-run", digest)
			history := bugApprovalHistory{items: []*models.EntityHistory{{ID: 1, FromStatus: &from, ToStatus: "development", Notes: &reason}}}
			if mode == "invalidated" {
				history.items = append(history.items, &models.EntityHistory{ID: 2, ToStatus: "research"})
			}
			if mode == "unapproved" {
				plan = []byte("plan v2\n")
				require.NoError(t, os.WriteFile(filepath.Join(root, "docs/plan/plan.md"), plan, 0o600))
				hash = sha256.Sum256(plan)
			}
			if mode == "source_changed" {
				require.NoError(t, os.WriteFile(filepath.Join(root, "docs/plan/plan.md"), []byte("unreviewed edit"), 0o600))
			}
			if mode == "baseline_changed" {
				baseline = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			}
			status := &fakeTransitioner{status: map[string]string{"B001": "development"}}
			written := &fakeNoteWriter{}
			coord := gatepersist.NewCoordinator(written, fakeNoteReader{}, fakeHistoryReader{}, fakeStatusValidator{valid: map[string]bool{"development": true, "code_review": true}}, status, status, &fakeLeaseReleaser{}, status)
			coord.Admission = NewBugGateAdmission(notes, history)
			currentEnvelope := bugApprovalEnvelopeCase(mode, hash, envelope(hash, "pass", baseline))
			result, err := IngestGateResult(context.Background(), GateIngestRequest{
				EnvelopeBytes: currentEnvelope, Coordinator: coord, ProjectRoot: root, RunID: "current-run",
				EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: "development", Gate: "development",
				Session: gatepersist.Session{ID: "parent"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
				Outcomes: map[string]string{"pass": "code_review"},
			})
			if validBugApprovalMode(mode) {
				require.NoError(t, err)
				require.Equal(t, "code_review", result.ToStatus)
			} else {
				require.Error(t, err)
				require.Equal(t, "development", status.status["B001"])
				require.Empty(t, written.notes)
			}
		})
	}
}
