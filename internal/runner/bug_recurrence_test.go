package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

// B04: a new fingerprint inside an accepted completed sweep's scope must cite
// that sweep, selected from the owning project's history, and renew the whole
// scope. Selection ignores incomplete sweeps and sweeps that were never applied.
func TestBugRecurrenceSelectsLatestAcceptedSweepFromHistory(t *testing.T) {
	root := t.TempDir()
	var notes bugApprovalNotes
	var history bugApprovalHistory
	sweepEnvelope := func(class string, status string, scope []string, fingerprints ...string) []byte {
		var instances []gateresult.SweepInstance
		for _, fp := range fingerprints {
			instances = append(instances, gateresult.SweepInstance{Fingerprint: fp, SitePointer: scope[0] + ":1", Disposition: "fixed"})
		}
		open := 0
		if status != "complete" {
			open = 1
		}
		sweep := gateresult.DefectClassSweep{ClassKey: class, ClassStatement: "s", SearchScope: scope, SearchedCount: 2, MatchingCount: len(instances), Instances: instances, FixedCount: len(instances) - open, OpenCount: open,
			Guard: gateresult.SweepGuard{Kind: "test", ImplementationPointer: "g", CounterfactualPointer: "c", Status: "verified"}, Status: status}
		return mustJSON(t, map[string]any{"kind": "final", "recommended_outcome": "pass", "evidence": []any{}, "gate_result": map[string]any{"schema_version": 1, "summary": "x", "remediation_sweeps": []any{sweep}}})
	}
	seed := func(run string, envelope []byte, apply bool) {
		seedMaterialReviewAt(t, root, "", run, envelope, &notes, &history)
		last := history.items[len(history.items)-1]
		to := "development"
		last.ToStatus = to
		if !apply {
			dir := filepath.Join(root, ".shark", "runs", run)
			state, exists, err := gaterun.LoadOperationState(dir)
			require.NoError(t, err)
			require.True(t, exists)
			state.PersistenceState = gaterun.PersistenceStatePending
			require.NoError(t, state.Save(dir))
		}
	}
	scope := []string{"internal/foo.go", "internal/sibling.go"}
	guardRecord := func(restoredExit int) {
		doc := map[string]any{
			"guard_pointer": "g", "class_key": "K", "site": scope[0] + ":1", "candidate": "cand", "enabled": true,
			"restored":  map[string]any{"exit_code": restoredExit, "flagged_site": scope[0] + ":1", "diagnostic": "guard flagged " + scope[0] + ":1"},
			"corrected": map[string]any{"exit_code": 0},
		}
		require.NoError(t, os.WriteFile(filepath.Join(root, "g"), []byte("package guard"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(root, "c"), mustJSON(t, doc), 0o600))
	}
	guardRecord(1)
	seed("sweep-old", sweepEnvelope("K", "complete", scope, "old-1"), true)
	seed("sweep-new", sweepEnvelope("K", "complete", scope, "new-1"), true)

	current := func(class, site, fingerprint string, evidence ...string) gatepersist.Request {
		return gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "current"), EntityKey: "B001", EntityType: models.EntityTypeBug, Gate: "code_review",
			Result: &gateresult.GateResult{Findings: []gateresult.Finding{{Severity: "blocker", ClassKey: class, Fingerprint: fingerprint, AffectedIDs: []string{site}, EvidencePointers: evidence, Disposition: "open"}}}}
	}
	record := func(name string, mutate func(map[string]any)) string {
		doc := map[string]any{"kind": "recurrence", "prior_run_id": "sweep-new", "prior_instance_fingerprint": "new-1", "class_key": "K", "scope": scope, "renewed_enumeration": scope}
		mutate(doc)
		data, err := json.Marshal(doc)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(root, name), data, 0o600))
		return name
	}
	ctx := context.Background()
	verify := func(req gatepersist.Request) error { return verifyBugRecurrence(ctx, req, history) }
	require.NoError(t, verify(current("K", "internal/sibling.go:9", "f-new", record("ok.json", func(map[string]any) {}))))
	for name, mutate := range map[string]func(map[string]any){
		"older sweep cited":           func(m map[string]any) { m["prior_run_id"], m["prior_instance_fingerprint"] = "sweep-old", "old-1" },
		"fingerprint not in sweep":    func(m map[string]any) { m["prior_instance_fingerprint"] = "other" },
		"sibling-only enumeration":    func(m map[string]any) { m["renewed_enumeration"] = []string{"internal/sibling.go"} },
		"scope narrowed":              func(m map[string]any) { m["scope"] = []string{"internal/sibling.go"} },
		"ordinary rework classifying": func(m map[string]any) { m["kind"] = "ordinary" },
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, verify(current("K", "internal/sibling.go:9", "f-new", record("bad.json", mutate))), "fingerprint f-new recurs inside accepted sweep sweep-new")
		})
	}
	t.Run("recurrence evidence omitted", func(t *testing.T) {
		require.ErrorContains(t, verify(current("K", "internal/sibling.go:9", "f-new")), "explicit recurrence evidence")
	})
	t.Run("other class or scope keeps ordinary handling", func(t *testing.T) {
		require.NoError(t, verify(current("OTHER", "internal/sibling.go:9", "f-new")))
		require.NoError(t, verify(current("K", "internal/other.go:3", "f-new")))
	})
	t.Run("same fingerprint is the existing recurrence path", func(t *testing.T) {
		require.NoError(t, verify(current("K", "internal/sibling.go:9", "new-1")))
	})
	t.Run("later incomplete or unapplied sweeps supply no authority", func(t *testing.T) {
		seed("sweep-incomplete", sweepEnvelope("K", "open", scope, "late-1"), true)
		seed("sweep-unapplied", sweepEnvelope("K", "complete", scope, "late-2"), false)
		require.NoError(t, verify(current("K", "internal/sibling.go:9", "f-new", record("still-ok.json", func(map[string]any) {}))))
	})
	t.Run("completed sweep without surviving guard evidence earns no historical credit", func(t *testing.T) {
		req := current("K", "internal/sibling.go:9", "f-new")
		require.ErrorContains(t, verify(req), "explicit recurrence evidence", "valid evidence credits the predecessor")
		guardRecord(0) // the guard no longer flags the restored defect
		require.NoError(t, verify(req), "unsupported sweep must supply no recurrence authority")
		cited := current("K", "internal/sibling.go:9", "f-new", record("cites-unsupported.json", func(map[string]any) {}))
		require.ErrorContains(t, verify(cited), "fails guard re-verification", "a recurrence may not cite an unsupported run")
		require.NoError(t, os.Remove(filepath.Join(root, "c")))
		require.NoError(t, verify(req), "missing evidence must supply no authority")
		guardRecord(1)
		require.ErrorContains(t, verify(req), "explicit recurrence evidence")
	})
	t.Run("unverifiable history blocks", func(t *testing.T) {
		broken := history
		broken.items = append(append([]*models.EntityHistory(nil), history.items...), &models.EntityHistory{ID: 99, ToStatus: "development", Notes: ptr(gatepersist.BoundTransitionReason("code_review", "pass", "missing-run", "0000000000000000000000000000000000000000000000000000000000000000"))})
		err := verifyBugRecurrence(ctx, current("K", "internal/sibling.go:9", "f-new", record("ok2.json", func(map[string]any) {})), broken)
		require.ErrorContains(t, err, "recurrence history unavailable")
		require.Error(t, verifyBugRecurrence(ctx, current("K", "internal/sibling.go:9", "f-new"), nil))
	})
}

// A historical guard record is bound to its own run's retained candidate: a
// record naming a different candidate earns no credit for that run's sweep.
func TestBugHistoricalGuardRecordIsBoundToItsRunCandidate(t *testing.T) {
	root := t.TempDir()
	scope := []string{"internal/foo.go"}
	site := scope[0] + ":1"
	scopeDoc := mustJSON(t, map[string]any{"candidate_commit": "reviewed"})
	require.NoError(t, os.WriteFile(filepath.Join(root, "scope.json"), scopeDoc, 0o600))
	digest := sha256.Sum256(scopeDoc)
	guard := func(candidate string) {
		doc := map[string]any{"guard_pointer": "g", "class_key": "K", "site": site, "candidate": candidate, "enabled": true,
			"restored":  map[string]any{"exit_code": 1, "flagged_site": site, "diagnostic": "flagged " + site},
			"corrected": map[string]any{"exit_code": 0}}
		require.NoError(t, os.WriteFile(filepath.Join(root, "g"), []byte("package guard"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(root, "c"), mustJSON(t, doc), 0o600))
	}
	sweep := gateresult.DefectClassSweep{ClassKey: "K", ClassStatement: "s", SearchScope: scope, SearchedCount: 1, MatchingCount: 1, FixedCount: 1,
		Instances: []gateresult.SweepInstance{{Fingerprint: "old-1", SitePointer: site, Disposition: "fixed"}},
		Guard:     gateresult.SweepGuard{Kind: "test", ImplementationPointer: "g", CounterfactualPointer: "c", Status: "verified"}, Status: "complete"}
	envelope := mustJSON(t, map[string]any{"kind": "final", "recommended_outcome": "pass",
		"evidence":    []any{map[string]any{"kind": "bug_review_scope", "pointer": "scope.json", "summary": "sha256:" + hex.EncodeToString(digest[:])}},
		"gate_result": map[string]any{"schema_version": 1, "summary": "x", "remediation_sweeps": []any{sweep}}})
	var notes bugApprovalNotes
	var history bugApprovalHistory
	seedMaterialReviewAt(t, root, "", "sweep-old", envelope, &notes, &history)
	history.items[len(history.items)-1].ToStatus = "development"
	req := gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "current"), EntityKey: "B001", EntityType: models.EntityTypeBug, Gate: "code_review",
		Result: &gateresult.GateResult{Findings: []gateresult.Finding{{Severity: "blocker", ClassKey: "K", Fingerprint: "f-new", AffectedIDs: []string{"internal/foo.go:9"}, Disposition: "open"}}}}
	ctx := context.Background()
	prior := func() int {
		got, _, err := acceptedBugSweeps(ctx, req, history)
		require.NoError(t, err)
		return len(got)
	}
	guard("reviewed")
	require.Equal(t, 1, prior(), "a record for the run's own candidate earns credit")
	guard("some-other-candidate")
	require.Equal(t, 0, prior(), "a record for a different candidate earns no credit")
}

func ptr(s string) *string { return &s }
