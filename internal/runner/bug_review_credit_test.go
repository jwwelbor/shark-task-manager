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
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

// creditFixture models B01: two independent leaves reviewed in round A, then a
// successor C that edits only leaf-b. Leaf-a may be credited from A's receipt.
type creditFixture struct {
	t        *testing.T
	root     string
	scope    bugCandidateScope
	predID   bugReviewIdentity
	notes    bugApprovalNotes
	history  bugApprovalHistory
	run      string
	digest   string
	delta    bugSuccessorDelta
	predTree string
	seq      int
}

func newCreditFixture(t *testing.T, mutate func(*creditFixture, func(args ...string) string)) *creditFixture {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("config", "user.name", "T")
	git("config", "user.email", "t@example.com")
	git("config", "commit.gpgsign", "false")
	write := func(name, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(body), 0600))
	}
	write("leaf-a.md", "base a\n")
	write("leaf-b.md", "base b\n")
	git("add", ".")
	git("commit", "-m", "base")
	base := git("rev-parse", "HEAD")
	write("leaf-a.md", "round A a\n")
	write("leaf-b.md", "round A b\n")
	git("commit", "-am", "A")
	f := &creditFixture{t: t, root: root, run: "review-a"}
	predCommit, predTree := git("rev-parse", "HEAD"), git("rev-parse", "HEAD^{tree}")
	write("leaf-b.md", "successor b\n")
	if mutate != nil {
		mutate(f, git)
	}
	git("commit", "-am", "C")
	f.scope = bugCandidateScope{BaseCommit: base, CandidateCommit: git("rev-parse", "HEAD"), CandidateTree: git("rev-parse", "HEAD^{tree}"),
		DiffSHA256: "dc", RepairPlanSHA256: "pc", ChangedFiles: reviewGitPaths{"leaf-a.md", "leaf-b.md"}}
	f.delta = bugSuccessorDelta{PredecessorCommit: predCommit, PredecessorTree: predTree, ChangedFiles: reviewGitPaths{"leaf-b.md"}}
	raw, err := json.Marshal(f.delta)
	require.NoError(t, err)
	f.scope.SuccessorDelta = raw
	f.predID = bugReviewIdentity{predCommit, predTree, "da", "pa"}
	f.predTree = predTree
	f.seedPredecessor(nil)
	return f
}

func (f *creditFixture) seedPredecessor(edit func(map[string]bugAngle)) {
	t := f.t
	f.seq++
	f.run = fmt.Sprintf("review-a-%d", f.seq)
	predScope := bugCandidateScope{BaseCommit: f.scope.BaseCommit, CandidateCommit: f.predID.Commit, CandidateTree: f.predID.Tree, DiffSHA256: "da", RepairPlanSHA256: "pa", ChangedFiles: reviewGitPaths{"leaf-a.md", "leaf-b.md"}}
	angles := map[string]bugAngle{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angles[name] = bugAngle{CandidateIdentity: f.predID, ReviewedFiles: reviewGitPaths{"leaf-a.md", "leaf-b.md"}, Findings: []bugAngleFinding{}}
	}
	if edit != nil {
		edit(angles)
	}
	save := func(name, kind string, v any) string {
		data, err := json.Marshal(v)
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(filepath.Join(f.root, ".shark"), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(f.root, ".shark", name), data, 0600))
		return fmt.Sprintf(`{"kind":%q,"pointer":".shark/%s","summary":"sha256:%x"}`, kind, name, sha256.Sum256(data))
	}
	refs := save("pred-scope.json", "bug_review_scope", predScope) + "," + save("pred-angles.json", "bug_review_angles", angles)
	result := []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[%s],"gate_result":{"schema_version":1,"summary":"checked"}}`, refs))
	f.notes, f.history = bugApprovalNotes{}, bugApprovalHistory{}
	seedMaterialReviewAt(t, f.root, "", f.run, result, &f.notes, &f.history)
	var meta bugApprovalNote
	require.NoError(t, json.Unmarshal([]byte(*f.notes.items[0].Metadata), &meta))
	f.digest = meta.Digest
}

func (f *creditFixture) packet(edit func(*bugAngle)) []byte {
	unit := bugCarriedUnit{Version: 1, Path: "leaf-a.md", Obligation: "file_local", Inputs: reviewGitPaths{}, DynamicLookup: new(bool),
		PredecessorCommit: f.delta.PredecessorCommit, PredecessorRun: f.run, PredecessorOperation: f.digest}
	angles := map[string]bugAngle{}
	id := bugReviewIdentity{f.scope.CandidateCommit, f.scope.CandidateTree, f.scope.DiffSHA256, f.scope.RepairPlanSHA256}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		angle := bugAngle{CandidateIdentity: id, ReviewedFiles: reviewGitPaths{"leaf-b.md"}, Findings: []bugAngleFinding{}, CarriedUnits: []bugCarriedUnit{unit}, CrossFileFresh: true}
		if edit != nil {
			edit(&angle)
		}
		angles[name] = angle
	}
	data, err := json.Marshal(angles)
	require.NoError(f.t, err)
	return data
}

func (f *creditFixture) check(packet []byte, credit *bugCredit) error {
	req := gatepersist.Request{EntityKey: "B001", EntityType: models.EntityTypeBug, RunDir: filepath.Join(f.root, ".shark", "runs", "current")}
	return validateBugReviewAnglesWith(packet, f.scope, map[bugFindingKey]bugReviewDecision{}, "PASS", credit.verifier(context.Background(), req, f.scope))
}

func (f *creditFixture) credit(endID int64) *bugCredit {
	return &bugCredit{notes: f.notes, history: f.history, normalize: func(s string) string { return s }, resolve: defaultBugSuccessTarget, endID: endID}
}

func TestBugCarriedCreditRequiresVerifiedPredecessorAndUnchangedLeaf(t *testing.T) {
	f := newCreditFixture(t, nil)
	// B01 positive: leaf-a is credited, leaf-b and cross-file work are fresh.
	require.NoError(t, f.check(f.packet(nil), f.credit(0)))
	// Counterfactual: without parent history readers the same packet cannot fund credit.
	require.ErrorContains(t, f.check(f.packet(nil), nil), "unavailable")
	// Legacy full-manifest packets are unaffected.
	require.NoError(t, f.check(f.packet(func(a *bugAngle) { a.CarriedUnits, a.ReviewedFiles = nil, reviewGitPaths{"leaf-a.md", "leaf-b.md"} }), nil))

	forged := f.packet(func(a *bugAngle) { a.CarriedUnits[0].PredecessorOperation = strings.Repeat("0", 64) })
	require.ErrorContains(t, f.check(forged, f.credit(0)), "not a parent-applied")
	require.ErrorContains(t, f.check(f.packet(func(a *bugAngle) { a.CrossFileFresh = false }), f.credit(0)), "cross-file")
	require.ErrorContains(t, f.check(f.packet(func(a *bugAngle) { a.CarriedUnits[0].Path = "leaf-b.md"; a.ReviewedFiles = reviewGitPaths{"leaf-a.md"} }), f.credit(0)), "changed since the predecessor")
	require.ErrorContains(t, f.check(f.packet(func(a *bugAngle) { a.CarriedUnits[0].Inputs = reviewGitPaths{"leaf-b.md"} }), f.credit(0)), "declared input")
	require.ErrorContains(t, f.check(f.packet(func(a *bugAngle) { *a.CarriedUnits[0].DynamicLookup = true }), f.credit(0)), "closed input list")
}

func TestBugCarriedCreditRejectsInterruptedHistoryAndUndeclaredTreeChange(t *testing.T) {
	f := newCreditFixture(t, nil)
	reject := fmt.Sprintf("gate qa outcome fail run r2 operation %s", strings.Repeat("a", 64))
	gate := "qa"
	f.history.items = append(f.history.items, &models.EntityHistory{ID: 3, FromStatus: &gate, ToStatus: "development", Notes: &reject})
	require.ErrorContains(t, f.check(f.packet(nil), f.credit(0)), "rejection or forced")
	// At QA replay the interval ends at the accepted successor, so a later entry is outside it.
	require.NoError(t, f.check(f.packet(nil), f.credit(2)))

	// The captured delta omits leaf-a although it differs between the two trees.
	drift := newCreditFixture(t, func(d *creditFixture, _ func(...string) string) {
		require.NoError(t, os.WriteFile(filepath.Join(d.root, "leaf-a.md"), []byte("silent edit\n"), 0600))
	})
	require.ErrorContains(t, drift.check(drift.packet(nil), drift.credit(0)), "differs between")
}

func TestBugCarriedCreditRejectsRecursiveAndUninspectedPredecessor(t *testing.T) {
	f := newCreditFixture(t, nil)
	f.seedPredecessor(func(angles map[string]bugAngle) {
		a := angles["a"]
		a.ReviewedFiles = reviewGitPaths{"leaf-b.md"}
		angles["a"] = a
	})
	require.ErrorContains(t, f.check(f.packet(nil), f.credit(0)), "no direct inspection")
	f.seedPredecessor(func(angles map[string]bugAngle) {
		a := angles["a"]
		a.CarriedUnits = f.packetUnits()
		angles["a"] = a
	})
	require.ErrorContains(t, f.check(f.packet(nil), f.credit(0)), "recursive")
}

func (f *creditFixture) packetUnits() []bugCarriedUnit {
	return []bugCarriedUnit{{Path: "leaf-a.md", Inputs: reviewGitPaths{}}}
}

func TestBugCarriedCreditSharedFixtureShapes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "skills", "shark-rider", "skills", "deep-review", "scripts", "testdata", "credit_cases.json"))
	require.NoError(t, err)
	var doc struct {
		ChangedFiles   reviewGitPaths  `json:"changed_files"`
		SuccessorDelta json.RawMessage `json:"successor_delta"`
		Cases          []struct {
			Name     string          `json:"name"`
			ErrorKey *string         `json:"error_key"`
			Angle    json.RawMessage `json:"angle"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(data, &doc))
	messages := map[string]string{
		"cross_file": "fresh cross-file", "closed_inputs": "closed input list", "version": "credit version",
		"identity": "predecessor identity", "path": "changed since the predecessor", "input": "declared input",
		"duplicate": "duplicated", "overlap": "both fresh and carried", "missing": "coverage incomplete",
	}
	scope := bugCandidateScope{ChangedFiles: doc.ChangedFiles, SuccessorDelta: doc.SuccessorDelta}
	accept := func(string, bugCarriedUnit) error { return nil }
	for _, c := range doc.Cases {
		t.Run(c.Name, func(t *testing.T) {
			var angle bugAngle
			require.NoError(t, json.Unmarshal(c.Angle, &angle))
			err := checkBugAngleCoverage("a", angle, scope, accept)
			if c.ErrorKey == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, messages[*c.ErrorKey])
		})
	}
}
