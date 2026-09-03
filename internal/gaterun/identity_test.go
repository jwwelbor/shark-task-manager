package gaterun

import "testing"

func fullIdentity() RunIdentity {
	return RunIdentity{
		RunID:           "run-1",
		EntityKey:       "T-E01-F01-001",
		EntityType:      "task",
		SourceStatus:    "in_review",
		Gate:            "qa",
		OperationDigest: "digest-abc",
	}
}

func TestCreateIdentity_CreatesOnceAndIsIdempotentForSameEntity(t *testing.T) {
	dir := newRunDir(t)
	rec := fullIdentity()

	created, err := CreateIdentity(dir, rec)
	if err != nil {
		t.Fatalf("CreateIdentity (first): %v", err)
	}
	if !created {
		t.Error("created = false, want true on first call")
	}

	created, err = CreateIdentity(dir, rec)
	if err != nil {
		t.Fatalf("CreateIdentity (replay, same entity): %v", err)
	}
	if created {
		t.Error("created = true on replay, want false (idempotent)")
	}

	got, exists, err := ReadIdentity(dir)
	if err != nil {
		t.Fatalf("ReadIdentity: %v", err)
	}
	if !exists {
		t.Fatal("ReadIdentity: exists = false, want true")
	}
	if *got != rec {
		t.Errorf("ReadIdentity = %+v, want %+v", *got, rec)
	}
}

func TestCreateIdentity_RejectsRebindingToADifferentEntity(t *testing.T) {
	dir := newRunDir(t)
	original := fullIdentity()
	if _, err := CreateIdentity(dir, original); err != nil {
		t.Fatalf("CreateIdentity (original): %v", err)
	}

	foreign := original
	foreign.EntityKey = "T-E02-F02-002"
	_, err := CreateIdentity(dir, foreign)
	if err == nil {
		t.Fatal("CreateIdentity (foreign entity, same run_id) succeeded, want *ConflictError")
	}
	if !IsConflict(err) {
		t.Errorf("CreateIdentity (foreign entity) error = %v, want *ConflictError", err)
	}

	// The originally bound identity must be left untouched.
	got, exists, err := ReadIdentity(dir)
	if err != nil {
		t.Fatalf("ReadIdentity: %v", err)
	}
	if !exists {
		t.Fatal("ReadIdentity: exists = false, want true")
	}
	if *got != original {
		t.Errorf("ReadIdentity after rejected rebind = %+v, want unchanged %+v", *got, original)
	}
}

// TestCreateIdentity_RejectsRebindingOnDivergedReplayContext is the UAT
// round 3+4 (note #2926) regression: a SAME-entity CreateIdentity call that
// disagrees on SourceStatus, Gate, or OperationDigest must fail closed
// exactly like a different-entity rebind, never silently rebind the run to
// a new replay context. Before this fix's RunIdentity extension, this case
// could not even be represented — only EntityKey/EntityType were part of
// the bound record, so a diverged SourceStatus/Gate/digest was silently
// accepted.
func TestCreateIdentity_RejectsRebindingOnDivergedReplayContext(t *testing.T) {
	fields := []struct {
		name   string
		mutate func(rec *RunIdentity)
	}{
		{"source_status", func(rec *RunIdentity) { rec.SourceStatus = "blocked" }},
		{"gate", func(rec *RunIdentity) { rec.Gate = "code_review" }},
		{"operation_digest", func(rec *RunIdentity) { rec.OperationDigest = "digest-xyz" }},
	}
	for _, f := range fields {
		t.Run(f.name, func(t *testing.T) {
			dir := newRunDir(t)
			original := fullIdentity()
			if _, err := CreateIdentity(dir, original); err != nil {
				t.Fatalf("CreateIdentity (original): %v", err)
			}

			diverged := original
			f.mutate(&diverged)
			_, err := CreateIdentity(dir, diverged)
			if err == nil {
				t.Fatalf("CreateIdentity (diverged %s, same entity) succeeded, want *ConflictError", f.name)
			}
			if !IsConflict(err) {
				t.Errorf("CreateIdentity (diverged %s) error = %v, want *ConflictError", f.name, err)
			}

			got, exists, err := ReadIdentity(dir)
			if err != nil {
				t.Fatalf("ReadIdentity: %v", err)
			}
			if !exists {
				t.Fatal("ReadIdentity: exists = false, want true")
			}
			if *got != original {
				t.Errorf("ReadIdentity after rejected rebind = %+v, want unchanged %+v", *got, original)
			}
		})
	}
}

func TestCreateIdentity_RejectsEmptyFields(t *testing.T) {
	dir := newRunDir(t)
	cases := []RunIdentity{
		{RunID: "", EntityKey: "T-E01-F01-001", EntityType: "task", Gate: "qa", OperationDigest: "d"},
		{RunID: "run-1", EntityKey: "", EntityType: "task", Gate: "qa", OperationDigest: "d"},
		{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: "", Gate: "qa", OperationDigest: "d"},
		{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: "task", Gate: "", OperationDigest: "d"},
		{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: "task", Gate: "qa", OperationDigest: ""},
	}
	for _, rec := range cases {
		if _, err := CreateIdentity(dir, rec); err == nil {
			t.Errorf("CreateIdentity(%+v) succeeded, want error for missing field", rec)
		}
	}
}

func TestReadIdentity_NotExists(t *testing.T) {
	dir := newRunDir(t)
	rec, exists, err := ReadIdentity(dir)
	if err != nil {
		t.Fatalf("ReadIdentity: %v", err)
	}
	if exists {
		t.Error("exists = true, want false")
	}
	if rec != nil {
		t.Errorf("rec = %+v, want nil", rec)
	}
}

func TestVerifyRunIdentityOwner(t *testing.T) {
	rec := &RunIdentity{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: "task"}

	if err := VerifyRunIdentityOwner(rec, "T-E01-F01-001", "task"); err != nil {
		t.Errorf("matching identity: unexpected error: %v", err)
	}
	if err := VerifyRunIdentityOwner(rec, "T-E02-F02-002", "task"); err == nil {
		t.Error("mismatched entity_key: want error, got nil")
	}
	if err := VerifyRunIdentityOwner(rec, "T-E01-F01-001", "feature"); err == nil {
		t.Error("mismatched entity_type: want error, got nil")
	}
	if err := VerifyRunIdentityOwner(nil, "T-E01-F01-001", "task"); err == nil {
		t.Error("nil record: want error, got nil")
	}
}

// TestVerifyRunIdentity_FailsClosedOnReplayContextMismatch is the exact UAT
// round 3+4 (note #2926) defect reproduction: a result.json recovered under
// a DIFFERENT source_status/gate/digest than originally recorded, for the
// SAME entity, must be rejected — not silently accepted because only
// EntityKey/EntityType were checked.
func TestVerifyRunIdentity_FailsClosedOnReplayContextMismatch(t *testing.T) {
	rec := &RunIdentity{
		RunID:           "run-1",
		EntityKey:       "T-E01-F01-001",
		EntityType:      "task",
		SourceStatus:    "in_review",
		Gate:            "qa",
		OperationDigest: "digest-abc",
	}

	matching := RunIdentity{
		EntityKey:       "T-E01-F01-001",
		EntityType:      "task",
		SourceStatus:    "in_review",
		Gate:            "qa",
		OperationDigest: "digest-abc",
	}
	if err := VerifyRunIdentity(rec, matching); err != nil {
		t.Errorf("matching identity: unexpected error: %v", err)
	}

	cases := []struct {
		name string
		want RunIdentity
	}{
		{"entity_key", func() RunIdentity { w := matching; w.EntityKey = "T-E02-F02-002"; return w }()},
		{"entity_type", func() RunIdentity { w := matching; w.EntityType = "feature"; return w }()},
		{"source_status", func() RunIdentity { w := matching; w.SourceStatus = "blocked"; return w }()},
		{"gate", func() RunIdentity { w := matching; w.Gate = "code_review"; return w }()},
		{"operation_digest", func() RunIdentity { w := matching; w.OperationDigest = "digest-xyz"; return w }()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := VerifyRunIdentity(rec, c.want); err == nil {
				t.Errorf("mismatched %s: want error, got nil", c.name)
			}
		})
	}

	if err := VerifyRunIdentity(nil, matching); err == nil {
		t.Error("nil record: want error, got nil")
	}
}

// TestCreateResult_StillCreatedWhenIdentityAlreadyBound is a narrow
// regression guard proving CreateIdentity and CreateResult peacefully
// coexist as separate create-once sidecars under the same run directory —
// the UAT-3-1 fix's expected write order (identity first, then result) must
// not break result.json's own existing create-once contract.
func TestCreateResult_StillCreatedWhenIdentityAlreadyBound(t *testing.T) {
	dir := newRunDir(t)
	if _, err := CreateIdentity(dir, fullIdentity()); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	created, err := CreateResult(dir, []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	if !created {
		t.Error("created = false, want true")
	}
}
