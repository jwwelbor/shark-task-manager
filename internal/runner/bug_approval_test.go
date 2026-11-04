package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func (h bugApprovalHistory) GetHistory(context.Context, models.EntityType, string) ([]*models.EntityHistory, error) {
	return h.items, nil
}

func TestBugApprovalRequiresCommittedCurrentUninvalidatedReview(t *testing.T) {
	for _, mode := range []string{"valid", "missing", "unapproved", "invalidated", "partial", "tampered"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			plan := []byte("plan v1\n")
			require.NoError(t, os.WriteFile(filepath.Join(root, "plan.md"), plan, 0o600))
			hash := sha256.Sum256(plan)
			envelope := func(h [32]byte) []byte {
				return []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"plan.md","summary":"sha256:%x"}],"gate_result":{"schema_version":1,"summary":"approved"}}`, h))
			}
			prior := envelope(hash)
			dir, err := gaterun.RunDir(root, "approved-run")
			require.NoError(t, err)
			lock, err := gaterun.AcquireRunLock(dir, 0)
			require.NoError(t, err)
			require.NoError(t, lock.Release())
			digest, err := gaterun.ComputeOperationDigest("B001", "bug", "plan_review", "plan_review", prior)
			require.NoError(t, err)
			_, err = gaterun.CreateIdentity(dir, gaterun.RunIdentity{RunID: "approved-run", EntityKey: "B001", EntityType: "bug", SourceStatus: "plan_review", Gate: "plan_review", OperationDigest: digest})
			require.NoError(t, err)
			_, err = gaterun.CreateResult(dir, prior)
			require.NoError(t, err)
			state := gaterun.NewOperationState("approved-run", "B001", "bug", "plan_review", "plan_review", digest)
			require.NoError(t, state.MarkPersistenceComplete())
			if mode != "partial" {
				require.NoError(t, state.MarkTransitionApplied())
			}
			require.NoError(t, state.Save(dir))
			if mode == "tampered" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "result.json"), []byte(`{"kind":"final"}`), 0o600))
			}
			metadata, err := json.Marshal(map[string]string{"gate": "plan_review", "role": "success", "outcome_key": "pass", "run_id": "approved-run", "operation_digest": digest})
			require.NoError(t, err)
			meta := string(metadata)
			notes := bugApprovalNotes{items: []*models.EntityNote{{ID: 1, Metadata: &meta}}}
			if mode == "missing" {
				notes.items = nil
			}
			from, reason := "plan_review", "gate plan_review outcome pass"
			history := bugApprovalHistory{items: []*models.EntityHistory{{ID: 1, FromStatus: &from, ToStatus: "development", Notes: &reason}}}
			if mode == "invalidated" {
				history.items = append(history.items, &models.EntityHistory{ID: 2, ToStatus: "research"})
			}
			if mode == "unapproved" {
				plan = []byte("plan v2\n")
				require.NoError(t, os.WriteFile(filepath.Join(root, "plan.md"), plan, 0o600))
				hash = sha256.Sum256(plan)
			}
			status := &fakeTransitioner{status: map[string]string{"B001": "development"}}
			written := &fakeNoteWriter{}
			coord := gatepersist.NewCoordinator(written, fakeNoteReader{}, fakeHistoryReader{}, fakeStatusValidator{valid: map[string]bool{"development": true, "code_review": true}}, status, status, &fakeLeaseReleaser{}, status)
			coord.Admission = NewBugGateAdmission(notes, history)
			result, err := IngestGateResult(context.Background(), GateIngestRequest{
				EnvelopeBytes: envelope(hash), Coordinator: coord, ProjectRoot: root, RunID: "current-run",
				EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: "development", Gate: "development",
				Session: gatepersist.Session{ID: "parent"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess},
				Outcomes: map[string]string{"pass": "code_review"},
			})
			if mode == "valid" {
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
