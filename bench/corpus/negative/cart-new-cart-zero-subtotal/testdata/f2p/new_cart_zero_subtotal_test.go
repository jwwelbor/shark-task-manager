package cart

import "testing"

// TestCart_NewCartHasZeroSubtotal is the held-back F2P oracle for negative
// corpus item cart-new-cart-zero-subtotal (T-E40-F01-004, rejection branch
// (a)). It asserts a "bug" that does not exist: a freshly constructed Cart
// from New() already has a zero Subtotal and a zero ItemCount at the
// fixture's base commit, so this test is already green before any patch is
// applied. bench/scripts/admit.sh (T-E40-F01-006) must reject this
// candidate at check (a) — F2P already green at base — never admitting it.
func TestCart_NewCartHasZeroSubtotal(t *testing.T) {
	c := New()

	if got, want := c.Subtotal(), 0; got != want {
		t.Fatalf("Subtotal() = %d, want %d", got, want)
	}
	if got, want := c.ItemCount(), 0; got != want {
		t.Fatalf("ItemCount() = %d, want %d", got, want)
	}
}
