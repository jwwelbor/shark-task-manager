package inventory

import "testing"

// TestStock_ReserveRejectsNegativeQuantity is the held-back F2P oracle for
// corpus item inventory-reserve-rejects-negative-quantity (T-E40-F01-003).
// It fails at the fixture's base commit, where Reserve silently accepts a
// negative quantity and thereby increases available stock instead of
// rejecting the call, and passes once reference.patch adds the validation.
func TestStock_ReserveRejectsNegativeQuantity(t *testing.T) {
	s := NewStock()
	s.Set("sku-1", 5)

	if err := s.Reserve("sku-1", -3); err == nil {
		t.Fatal("Reserve(\"sku-1\", -3) expected error, got nil")
	}
	if got, want := s.Available("sku-1"), 5; got != want {
		t.Fatalf("Available() = %d, want %d (unchanged after rejected reservation)", got, want)
	}
}
