package inventory

import "testing"

// TestStock_AdjustRejectsNegativeResult is the held-back F2P oracle for
// corpus item inventory-adjust-rejects-negative-result (T-E40-F01-003). It
// fails at the fixture's base commit, where Adjust silently accepts a
// correction that would drive available stock negative, and passes once
// reference.patch adds the validation.
func TestStock_AdjustRejectsNegativeResult(t *testing.T) {
	s := NewStock()
	s.Set("sku-1", 2)

	if err := s.Adjust("sku-1", -5); err == nil {
		t.Fatal("Adjust(\"sku-1\", -5) expected error, got nil")
	}
	if got, want := s.Available("sku-1"), 2; got != want {
		t.Fatalf("Available() = %d, want %d (unchanged after rejected adjustment)", got, want)
	}
}
