package pricing

import "testing"

// TestRoundedUnitPrice_RoundsUpOnRemainder is the held-back F2P oracle for
// corpus item pricing-rounded-unit-price-truncation (T-E40-F01-003). It
// fails at the fixture's base commit, where RoundedUnitPrice floors instead
// of rounding up, and passes once reference.patch restores ceiling
// division.
func TestRoundedUnitPrice_RoundsUpOnRemainder(t *testing.T) {
	got := RoundedUnitPrice(100, 3)
	want := 34
	if got != want {
		t.Fatalf("RoundedUnitPrice(100, 3) = %d, want %d", got, want)
	}
}
