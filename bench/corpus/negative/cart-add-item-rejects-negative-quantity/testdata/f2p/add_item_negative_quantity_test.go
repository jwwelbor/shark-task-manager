package cart

import "testing"

// TestCart_AddItemRejectsNegativeQuantity is the held-back F2P oracle for
// negative corpus item cart-add-item-rejects-negative-quantity
// (T-E40-F01-004, rejection branch (b)). It is an ordinary, well-formed
// red-at-base test: AddItem currently has no negative-Quantity guard, so
// an item with a negative Quantity is appended and ItemCount() goes
// negative. This test passes honestly against check (a) — it is genuinely
// red at base. What makes this candidate a rejection is its manifest
// entry's p2p_set (full_suite_no_exclusions in bench/corpus/corpus.yaml),
// which — unlike every other item's "default" — does not exclude
// pkg/inventory::TestStock_PermanentlyFailingRegressionProbe, a
// deliberately wrong assertion committed straight into the fixture at the
// frozen base commit. bench/scripts/admit.sh (T-E40-F01-006) MUST reject
// this candidate at check (b) — P2P-red-at-base — because its own P2P set
// already contains a failing test before any patch, and MUST NOT admit it.
func TestCart_AddItemRejectsNegativeQuantity(t *testing.T) {
	c := New()
	c.AddItem(Item{SKU: "sku-1", Name: "Widget", Price: 500, Quantity: -1})

	if got, want := c.ItemCount(), 0; got != want {
		t.Fatalf("ItemCount() = %d, want %d", got, want)
	}
}
