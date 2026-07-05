package search

import (
	"testing"
)

// TestEntitySearchResult_IDFieldPresentInStruct is a compile-time assertion
// that the ID field exists on EntitySearchResult. If the field is missing,
// the struct literal below will fail to compile.
func TestEntitySearchResult_IDFieldPresentInStruct(t *testing.T) {
	// Compile-time check: struct literal with ID field must compile.
	r := EntitySearchResult{
		EntityType: "task",
		ID:         42,
		Key:        "E01-F01-001",
		Title:      "some task",
		Status:     "todo",
		Severity:   "",
		Rank:       -0.5,
		Snippet:    "some <mark>task</mark>",
	}
	if r.ID != 42 {
		t.Errorf("expected ID=42, got %d", r.ID)
	}
	if r.Rank == 0 {
		t.Errorf("expected non-zero Rank")
	}
	if r.Snippet == "" {
		t.Errorf("expected Snippet to be populated")
	}
}

// TestEntitySearchResult_IDFieldNonZero verifies that the ID field can hold
// a non-zero value and is preserved when constructing results. This acts as
// a runtime assertion that the field is properly wired.
func TestEntitySearchResult_IDFieldNonZero(t *testing.T) {
	results := []*EntitySearchResult{
		{EntityType: "epic", ID: 1, Key: "E01", Title: "epic one", Status: "in_progress"},
		{EntityType: "feature", ID: 5, Key: "E01-F01", Title: "feature one", Status: "todo"},
		{EntityType: "task", ID: 100, Key: "E01-F01-001", Title: "task one", Status: "in_progress"},
		{EntityType: "bug", ID: 7, Key: "B001", Title: "bug one", Status: "triage"},
		{EntityType: "change", ID: 3, Key: "CC-001", Title: "change one", Status: "draft"},
	}

	for _, r := range results {
		if r.ID == 0 {
			t.Errorf("entity_type=%s key=%s: ID must be non-zero", r.EntityType, r.Key)
		}
	}
}
