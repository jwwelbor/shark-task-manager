package inventory

import "testing"

// TestStock_ReserveExactAvailableStockSucceeds is the held-back F2P oracle
// for corpus item inventory-reserve-boundary (T-E40-F01-002). It fails at
// the fixture's base commit, where Reserve incorrectly rejects a
// reservation exactly equal to the available stock, and passes once
// reference.patch restores the correct boundary check.
func TestStock_ReserveExactAvailableStockSucceeds(t *testing.T) {
	s := NewStock()
	s.Set("sku-1", 5)

	if err := s.Reserve("sku-1", 5); err != nil {
		t.Fatalf("Reserve() unexpected error: %v", err)
	}
	if got, want := s.Available("sku-1"), 0; got != want {
		t.Fatalf("Available() = %d, want %d", got, want)
	}
}
