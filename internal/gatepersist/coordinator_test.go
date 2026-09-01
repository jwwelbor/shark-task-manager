package gatepersist

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

const mainEntityKey = "T-E34-F05-999"

// fixtureResult builds a structurally-valid GateResult with one finding, one
// remediation sweep, one change impact, and one kickback targeting a
// different task entity — exercising all four persisted note kinds plus the
// kickback step in one fixture, per feature.md's "fixture GateResult with
// findings, kickbacks, sweeps, and impacts" verification requirement.
func fixtureResult() *gateresult.GateResult {
	return &gateresult.GateResult{
		SchemaVersion: gateresult.SchemaVersion1,
		Summary:       "two findings found, one fixed via sweep",
		Findings: []gateresult.Finding{
			{
				Severity:       "high",
				ClassKey:       "missing-error-wrap",
				ClassStatement: "errors are dropped instead of wrapped",
				Fingerprint:    "fp-001",
				AffectedIDs:    []string{"AC-1"},
				Disposition:    gateresult.DispositionFixed,
			},
		},
		Kickbacks: []gateresult.Kickback{
			{EntityKey: "T-E34-F05-100", TargetStatus: "blocked", Reason: "needs rework on the caller"},
		},
		RemediationSweeps: []gateresult.DefectClassSweep{
			{
				ClassKey:           "missing-error-wrap",
				ClassStatement:     "errors are dropped instead of wrapped",
				SearchedCount:      3,
				MatchingCount:      1,
				FixedCount:         1,
				DispositionedCount: 0,
				OpenCount:          0,
				Instances: []gateresult.SweepInstance{
					{Fingerprint: "fp-001", SitePointer: "internal/foo.go:12", Disposition: "fixed"},
				},
				Guard: gateresult.SweepGuard{
					Kind:                  "test",
					ImplementationPointer: "internal/foo_test.go:5",
					CounterfactualPointer: "internal/foo_test.go:20",
					Status:                "verified",
				},
				Status: "complete",
			},
		},
		ChangeImpacts: []gateresult.ChangeImpactSet{
			{
				SourceKind:    "tech_debt",
				SourceKey:     "TD-042",
				SourcePointer: "docs/plan/td/TD-042.md",
				ChangeSummary: "error wrapping helper introduced",
				Status:        "accounted",
			},
		},
	}
}

func envelopeFor(result *gateresult.GateResult) json.RawMessage {
	envelope := map[string]interface{}{
		"kind":                "final",
		"recommended_outcome": "kickback_rework",
		"gate_result":         result,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		panic(err)
	}
	return data
}

func baseRequest(t *testing.T, runDir string) Request {
	t.Helper()
	result := fixtureResult()
	return Request{
		RunDir:       runDir,
		RunID:        "run-001",
		EntityKey:    mainEntityKey,
		EntityType:   models.EntityTypeTask,
		SourceStatus: "in_review",
		Gate:         "code_review",
		Session:      Session{ID: "sess-1", Agent: "reviewer-agent"},
		EnvelopeJSON: envelopeFor(result),
		Result:       result,
		Role:         gateresult.RoleKickbackRework,
		OutcomeKey:   "kickback_rework",
		TargetStatus: "ready_for_qa",
	}
}

func newTestCoordinator(w *fakeWorld, v StatusValidator) *Coordinator {
	return NewCoordinator(w, w, w, v, w, w)
}

func defaultValidator() *fakeStatusValidator {
	return newFakeStatusValidator().allow(models.EntityTypeTask, "todo", "in_review", "ready_for_qa", "completed", "blocked")
}

func TestPersist_OrderingAndContent(t *testing.T) {
	runDir := t.TempDir()
	world := newFakeWorld()
	coord := newTestCoordinator(world, defaultValidator())
	req := baseRequest(t, runDir)
	req.RetirementConfirmed = true

	res, err := coord.Persist(context.Background(), req)
	if err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if !res.PersistenceComplete || !res.TransitionApplied {
		t.Fatalf("expected persistence complete and transition applied, got %+v", res)
	}
	if !res.Transitioned || res.ToStatus != "ready_for_qa" {
		t.Fatalf("expected main transition to ready_for_qa, got %+v", res)
	}
	if !res.LeaseReleased {
		t.Fatalf("expected lease released")
	}
	if len(res.CompletedSuboperations) != 5 {
		t.Fatalf("expected 5 completed suboperations (summary+finding+sweep+impact+kickback), got %d: %v",
			len(res.CompletedSuboperations), res.CompletedSuboperations)
	}

	// Ordering: gate-summary note, then finding note, then sweep note, then
	// impact note, then the kickback transition, then the main transition.
	notes := world.notes
	if len(notes) != 4 {
		t.Fatalf("expected 4 notes (summary, finding, sweep, impact), got %d", len(notes))
	}
	wantOrder := []models.NoteType{models.NoteType(noteTypeReview), models.NoteType(noteTypeReviewFinding), models.NoteType(noteTypeReference), models.NoteType(noteTypeReference)}
	for i, want := range wantOrder {
		if notes[i].NoteType != want {
			t.Errorf("note[%d] type = %q, want %q", i, notes[i].NoteType, want)
		}
	}
	// The two "reference" notes are distinguished by record_kind metadata.
	sweepMeta, _ := decodeMetadata(*notes[2].Metadata)
	if sweepMeta[metaRecordKind] != recordKindSweep {
		t.Errorf("notes[2] record_kind = %v, want %s", sweepMeta[metaRecordKind], recordKindSweep)
	}
	impactMeta, _ := decodeMetadata(*notes[3].Metadata)
	if impactMeta[metaRecordKind] != recordKindImpact {
		t.Errorf("notes[3] record_kind = %v, want %s", impactMeta[metaRecordKind], recordKindImpact)
	}
	if impactMeta[metaSourceKey] != "TD-042" {
		t.Errorf("impact note missing source_key metadata: %v", impactMeta)
	}

	findingMeta, _ := decodeMetadata(*notes[1].Metadata)
	for _, want := range []string{metaSeverity, metaClassKey, metaClassStatement, metaFingerprint, metaDisposition, metaParentSession} {
		if _, ok := findingMeta[want]; !ok {
			t.Errorf("finding note metadata missing %q: %v", want, findingMeta)
		}
	}
	if findingMeta[metaParentSession] != "sess-1" {
		t.Errorf("finding note parent_session = %v, want sess-1", findingMeta[metaParentSession])
	}

	// Kickback happened before the main transition: only one transition call
	// recorded main entity's target status, and the kickback entity's target
	// status differs.
	if got := world.statuses[historyKey(models.EntityTypeTask, "T-E34-F05-100")]; got != "blocked" {
		t.Errorf("kickback target status = %q, want blocked", got)
	}
	if got := world.statuses[historyKey(models.EntityTypeTask, mainEntityKey)]; got != "ready_for_qa" {
		t.Errorf("main entity status = %q, want ready_for_qa", got)
	}
	if len(world.transitionCalls) != 2 {
		t.Fatalf("expected 2 transition calls (kickback + main), got %d: %+v", len(world.transitionCalls), world.transitionCalls)
	}
	if world.transitionCalls[0].entityKey != "T-E34-F05-100" {
		t.Errorf("expected kickback transition before main transition, got %+v", world.transitionCalls[0])
	}
	if world.transitionCalls[1].entityKey != mainEntityKey {
		t.Errorf("expected main transition last, got %+v", world.transitionCalls[1])
	}

	if len(world.releaseCalls) != 1 {
		t.Fatalf("expected exactly 1 release call, got %d", len(world.releaseCalls))
	}
}

func TestPersist_FailureInjectionResumesWithoutDuplicates(t *testing.T) {
	runDir := t.TempDir()
	world := newFakeWorld()
	req := baseRequest(t, runDir)

	// Inject a "crash" right after the finding note is durably committed,
	// before operation-state.json would record it.
	findingContentStr := findingContent(req.Result.Findings[0])
	world.failNoteContent[noteTypeReviewFinding+"|"+findingContentStr] = true

	coord := newTestCoordinator(world, defaultValidator())
	if _, err := coord.Persist(context.Background(), req); err == nil {
		t.Fatalf("expected injected failure to surface as an error")
	}
	if len(world.notes) != 2 {
		t.Fatalf("expected the gate-summary and finding notes to have committed durably despite the injected error, got %d", len(world.notes))
	}

	// Resume: same run dir/run ID, same request. The already-durable summary
	// and finding notes must be reconciled, not rewritten.
	req2 := baseRequest(t, runDir)
	req2.RetirementConfirmed = true
	res, err := coord.Persist(context.Background(), req2)
	if err != nil {
		t.Fatalf("resume Persist: %v", err)
	}
	if !res.TransitionApplied || !res.LeaseReleased {
		t.Fatalf("expected resume to complete transition and release, got %+v", res)
	}
	if len(world.notes) != 4 {
		t.Fatalf("expected exactly 4 notes total after resume (no duplicates), got %d", len(world.notes))
	}
	if len(world.transitionCalls) != 2 {
		t.Fatalf("expected exactly 2 transition calls total (kickback + main), got %d", len(world.transitionCalls))
	}
}

func TestPersist_KickbackFailureInjectionResumesWithoutReapplying(t *testing.T) {
	runDir := t.TempDir()
	world := newFakeWorld()
	req := baseRequest(t, runDir)

	world.failTransitionTo[string(models.EntityTypeTask)+"|T-E34-F05-100|blocked"] = true

	coord := newTestCoordinator(world, defaultValidator())
	if _, err := coord.Persist(context.Background(), req); err == nil {
		t.Fatalf("expected injected kickback failure to surface as an error")
	}
	if got := world.statuses[historyKey(models.EntityTypeTask, "T-E34-F05-100")]; got != "blocked" {
		t.Fatalf("expected kickback status change to have committed despite injected error, got %q", got)
	}

	req2 := baseRequest(t, runDir)
	req2.RetirementConfirmed = true
	if _, err := coord.Persist(context.Background(), req2); err != nil {
		t.Fatalf("resume Persist: %v", err)
	}
	kickbackCalls := 0
	for _, c := range world.transitionCalls {
		if c.entityKey == "T-E34-F05-100" {
			kickbackCalls++
		}
	}
	if kickbackCalls != 1 {
		t.Fatalf("expected exactly 1 kickback transition call across both attempts (no reapplication), got %d", kickbackCalls)
	}
}

func TestPersist_ConflictingReplayRejected(t *testing.T) {
	runDir := t.TempDir()
	world := newFakeWorld()
	coord := newTestCoordinator(world, defaultValidator())

	req := baseRequest(t, runDir)
	if _, err := coord.Persist(context.Background(), req); err != nil {
		t.Fatalf("first Persist: %v", err)
	}

	// A different result under the same run_id/run dir must be rejected,
	// not silently applied over the accepted first-writer bytes.
	differentResult := fixtureResult()
	differentResult.Summary = "a completely different summary"
	req2 := req
	req2.Result = differentResult
	req2.EnvelopeJSON = envelopeFor(differentResult)

	_, err := coord.Persist(context.Background(), req2)
	if err == nil {
		t.Fatalf("expected conflicting replay to be rejected")
	}
	if !gaterun.IsConflict(err) {
		t.Fatalf("expected a gaterun.ConflictError, got %v", err)
	}
	if len(world.notes) != 4 {
		t.Fatalf("conflicting replay must not add or change any note, got %d notes", len(world.notes))
	}
}

func TestPersist_TransitionAppliedThenReleaseGatedOnRetirement(t *testing.T) {
	runDir := t.TempDir()
	world := newFakeWorld()
	coord := newTestCoordinator(world, defaultValidator())

	req := baseRequest(t, runDir)
	req.RetirementConfirmed = false
	res, err := coord.Persist(context.Background(), req)
	if err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if !res.TransitionApplied {
		t.Fatalf("expected transition applied even without retirement evidence")
	}
	if res.LeaseReleased || len(world.releaseCalls) != 0 {
		t.Fatalf("expected lease NOT released before retirement evidence, got %+v", res)
	}

	mainTransitions := 0
	for _, c := range world.transitionCalls {
		if c.entityKey == mainEntityKey {
			mainTransitions++
		}
	}
	if mainTransitions != 1 {
		t.Fatalf("expected exactly 1 main transition call, got %d", mainTransitions)
	}

	// A second call (retirement now confirmed) must release without
	// repeating the transition.
	req2 := baseRequest(t, runDir)
	req2.RetirementConfirmed = true
	res2, err := coord.Persist(context.Background(), req2)
	if err != nil {
		t.Fatalf("second Persist: %v", err)
	}
	if !res2.LeaseReleased {
		t.Fatalf("expected lease released on second call")
	}

	mainTransitions = 0
	for _, c := range world.transitionCalls {
		if c.entityKey == mainEntityKey {
			mainTransitions++
		}
	}
	if mainTransitions != 2 {
		t.Fatalf("expected the idempotent verify-call on resume (2 total main-transition calls, second a no-op), got %d", mainTransitions)
	}
	if world.statuses[historyKey(models.EntityTypeTask, mainEntityKey)] != "ready_for_qa" {
		t.Fatalf("expected main entity to remain at ready_for_qa, not re-transitioned elsewhere")
	}
}

func TestPersist_KickbackTargetingMainEntityRejected(t *testing.T) {
	runDir := t.TempDir()
	world := newFakeWorld()
	coord := newTestCoordinator(world, defaultValidator())

	req := baseRequest(t, runDir)
	req.Result.Kickbacks[0].EntityKey = mainEntityKey
	req.EnvelopeJSON = envelopeFor(req.Result)

	if _, err := coord.Persist(context.Background(), req); err == nil {
		t.Fatalf("expected a kickback targeting the main entity to be rejected")
	}
	if len(world.notes) != 0 {
		t.Fatalf("expected no partial mutation when kickback validation fails, got %d notes", len(world.notes))
	}
}

func TestPersist_InvalidKickbackTargetStatusRejectedWithoutPartialMutation(t *testing.T) {
	runDir := t.TempDir()
	world := newFakeWorld()
	// Validator does not allow "todo" for task -- only ready_for_qa/completed.
	v := newFakeStatusValidator().allow(models.EntityTypeTask, "in_review", "ready_for_qa", "completed")
	coord := newTestCoordinator(world, v)

	req := baseRequest(t, runDir)
	_, err := coord.Persist(context.Background(), req)
	if err == nil {
		t.Fatalf("expected invalid kickback target status to be rejected")
	}
	var kerr *KickbackValidationError
	if !asKickbackValidationError(err, &kerr) {
		t.Fatalf("expected a *KickbackValidationError, got %v (%T)", err, err)
	}
	if len(world.notes) != 0 || len(world.transitionCalls) != 0 {
		t.Fatalf("expected no writes at all before kickback validation, got %d notes, %d transitions", len(world.notes), len(world.transitionCalls))
	}

	// Sanity: the run directory itself must not have accepted a durable
	// result either -- validation runs after CreateResult in this
	// implementation, so document and assert that explicitly instead of
	// silently relying on it.
	if _, exists, _ := gaterun.ReadResult(runDir); !exists {
		t.Fatalf("expected result.json to exist (validation runs after create-once accept, by design)")
	}
	// But operation-state must never reach persistence_complete/transitioned.
	state, exists, err := gaterun.LoadOperationState(runDir)
	if err != nil {
		t.Fatalf("LoadOperationState: %v", err)
	}
	if exists && state.PersistenceState != gaterun.PersistenceStatePending {
		t.Fatalf("expected operation state to remain pending after a rejected kickback, got %q", state.PersistenceState)
	}
}

func asKickbackValidationError(err error, target **KickbackValidationError) bool {
	if e, ok := err.(*KickbackValidationError); ok {
		*target = e
		return true
	}
	return false
}

func TestPersist_RequiresRunDirEntityFileSystem(t *testing.T) {
	tmp := t.TempDir()
	// A run dir under a plain file (not a directory) must fail closed rather
	// than silently succeeding or corrupting an unrelated path.
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	world := newFakeWorld()
	coord := newTestCoordinator(world, defaultValidator())
	req := baseRequest(t, filepath.Join(blocker, "run-1"))
	if _, err := coord.Persist(context.Background(), req); err == nil {
		t.Fatalf("expected acquiring a run lock under a non-directory path to fail")
	}
}
