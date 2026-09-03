package gaterun

import "testing"

func TestCreateIdentity_CreatesOnceAndIsIdempotentForSameEntity(t *testing.T) {
	dir := newRunDir(t)
	rec := RunIdentity{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: "task"}

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
	original := RunIdentity{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: "task"}
	if _, err := CreateIdentity(dir, original); err != nil {
		t.Fatalf("CreateIdentity (original): %v", err)
	}

	foreign := RunIdentity{RunID: "run-1", EntityKey: "T-E02-F02-002", EntityType: "task"}
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

func TestCreateIdentity_RejectsEmptyFields(t *testing.T) {
	dir := newRunDir(t)
	cases := []RunIdentity{
		{RunID: "", EntityKey: "T-E01-F01-001", EntityType: "task"},
		{RunID: "run-1", EntityKey: "", EntityType: "task"},
		{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: ""},
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

func TestVerifyRunIdentity(t *testing.T) {
	rec := &RunIdentity{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: "task"}

	if err := VerifyRunIdentity(rec, "T-E01-F01-001", "task"); err != nil {
		t.Errorf("matching identity: unexpected error: %v", err)
	}
	if err := VerifyRunIdentity(rec, "T-E02-F02-002", "task"); err == nil {
		t.Error("mismatched entity_key: want error, got nil")
	}
	if err := VerifyRunIdentity(rec, "T-E01-F01-001", "feature"); err == nil {
		t.Error("mismatched entity_type: want error, got nil")
	}
	if err := VerifyRunIdentity(nil, "T-E01-F01-001", "task"); err == nil {
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
	if _, err := CreateIdentity(dir, RunIdentity{RunID: "run-1", EntityKey: "T-E01-F01-001", EntityType: "task"}); err != nil {
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
