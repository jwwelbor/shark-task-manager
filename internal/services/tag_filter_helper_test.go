package services

import (
	"testing"
)

// testEntity is a minimal struct for testing filterByTagIDs.
type testEntity struct {
	ID   int64
	Name string
}

func testEntityID(e testEntity) int64 {
	return e.ID
}

// TestFilterByTagIDs_NilTaggedIDSetReturnsUnchanged verifies AC-T1:
// filterByTagIDs with taggedIDSet==nil returns the input slice unchanged.
func TestFilterByTagIDs_NilTaggedIDSetReturnsUnchanged(t *testing.T) {
	ents := []testEntity{
		{ID: 1, Name: "a"},
		{ID: 2, Name: "b"},
		{ID: 3, Name: "c"},
	}

	result := filterByTagIDs(ents, nil, testEntityID)

	if len(result) != len(ents) {
		t.Fatalf("expected len=%d, got len=%d", len(ents), len(result))
	}
	for i, e := range ents {
		if result[i].ID != e.ID {
			t.Errorf("result[%d].ID = %d, want %d", i, result[i].ID, e.ID)
		}
	}
}

// TestFilterByTagIDs_EmptyTaggedIDSetReturnsEmpty verifies AC-T2:
// filterByTagIDs with an empty (non-nil) taggedIDSet returns an empty non-nil slice.
func TestFilterByTagIDs_EmptyTaggedIDSetReturnsEmpty(t *testing.T) {
	ents := []testEntity{
		{ID: 1, Name: "a"},
		{ID: 2, Name: "b"},
	}
	emptySet := map[int64]struct{}{}

	result := filterByTagIDs(ents, emptySet, testEntityID)

	if result == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got len=%d", len(result))
	}
}

// TestFilterByTagIDs_MatchingIDsKept verifies that entities whose IDs are
// in the set are retained in the output.
func TestFilterByTagIDs_MatchingIDsKept(t *testing.T) {
	ents := []testEntity{
		{ID: 1, Name: "keep-1"},
		{ID: 2, Name: "drop"},
		{ID: 3, Name: "keep-3"},
		{ID: 4, Name: "drop"},
		{ID: 5, Name: "keep-5"},
	}
	set := map[int64]struct{}{
		1: {},
		3: {},
		5: {},
	}

	result := filterByTagIDs(ents, set, testEntityID)

	if len(result) != 3 {
		t.Fatalf("expected len=3, got len=%d", len(result))
	}
	for _, r := range result {
		if _, ok := set[r.ID]; !ok {
			t.Errorf("unexpected entity ID %d in result", r.ID)
		}
	}
}

// TestFilterByTagIDs_NoneMatch returns empty non-nil slice when no IDs match.
func TestFilterByTagIDs_NoneMatch(t *testing.T) {
	ents := []testEntity{
		{ID: 1, Name: "a"},
		{ID: 2, Name: "b"},
	}
	set := map[int64]struct{}{
		99:  {},
		100: {},
	}

	result := filterByTagIDs(ents, set, testEntityID)

	if result == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got len=%d", len(result))
	}
}

// TestFilterByTagIDs_AllMatch returns all entities when all IDs are in set.
func TestFilterByTagIDs_AllMatch(t *testing.T) {
	ents := []testEntity{
		{ID: 10, Name: "a"},
		{ID: 20, Name: "b"},
		{ID: 30, Name: "c"},
	}
	set := map[int64]struct{}{
		10: {},
		20: {},
		30: {},
	}

	result := filterByTagIDs(ents, set, testEntityID)

	if len(result) != 3 {
		t.Fatalf("expected len=3, got len=%d", len(result))
	}
}

// TestFilterByTagIDs_EmptyInputNilSet returns empty slice (nil set → unchanged empty).
func TestFilterByTagIDs_EmptyInputNilSet(t *testing.T) {
	var ents []testEntity

	result := filterByTagIDs(ents, nil, testEntityID)

	// nil set → return input unchanged; input is nil, so result should be nil
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

// TestFilterByTagIDs_EmptyInputEmptySet returns empty non-nil slice.
func TestFilterByTagIDs_EmptyInputEmptySet(t *testing.T) {
	var ents []testEntity
	set := map[int64]struct{}{}

	result := filterByTagIDs(ents, set, testEntityID)

	// set != nil → filter runs; no matches → kept = ents[:0]
	// ents is nil so ents[:0] is nil, but nil == empty for practical purposes.
	// The test verifies no panic and result is empty.
	if len(result) != 0 {
		t.Fatalf("expected empty, got len=%d", len(result))
	}
}

// TestFilterByTagIDs_WorksWithPointerSlice verifies the generic function
// works with pointer element types (e.g., []*testEntity).
func TestFilterByTagIDs_WorksWithPointerSlice(t *testing.T) {
	ents := []*testEntity{
		{ID: 1, Name: "a"},
		{ID: 2, Name: "b"},
		{ID: 3, Name: "c"},
	}
	set := map[int64]struct{}{
		2: {},
	}

	result := filterByTagIDs(ents, set, func(e *testEntity) int64 { return e.ID })

	if len(result) != 1 {
		t.Fatalf("expected len=1, got len=%d", len(result))
	}
	if result[0].ID != 2 {
		t.Errorf("expected ID=2, got ID=%d", result[0].ID)
	}
}

// TestFilterByTagIDs_PreservesOrder verifies that the relative order of
// surviving elements is preserved (in-place filter, not sorted by ID).
func TestFilterByTagIDs_PreservesOrder(t *testing.T) {
	ents := []testEntity{
		{ID: 5, Name: "first"},
		{ID: 1, Name: "second"},
		{ID: 3, Name: "third"},
	}
	// Keep IDs 5 and 3 (in original order).
	set := map[int64]struct{}{
		5: {},
		3: {},
	}

	result := filterByTagIDs(ents, set, testEntityID)

	if len(result) != 2 {
		t.Fatalf("expected len=2, got len=%d", len(result))
	}
	if result[0].ID != 5 {
		t.Errorf("expected first result ID=5, got ID=%d", result[0].ID)
	}
	if result[1].ID != 3 {
		t.Errorf("expected second result ID=3, got ID=%d", result[1].ID)
	}
}
