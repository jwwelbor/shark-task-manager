package pricing

import "testing"

// TestApplyDiscount_RoundsDown is the held-back F2P oracle for corpus item
// pricing-discount-rounding (T-E40-F01-002). It fails at the fixture's base
// commit, where ApplyDiscount rounds the discount to the nearest cent
// instead of the documented round-down behavior, and passes once
// reference.patch restores floor-rounding.
func TestApplyDiscount_RoundsDown(t *testing.T) {
	got, err := ApplyDiscount(999, 33)
	if err != nil {
		t.Fatalf("ApplyDiscount(999, 33) unexpected error: %v", err)
	}
	if want := 670; got != want {
		t.Fatalf("ApplyDiscount(999, 33) = %d, want %d (rounded down)", got, want)
	}
}
