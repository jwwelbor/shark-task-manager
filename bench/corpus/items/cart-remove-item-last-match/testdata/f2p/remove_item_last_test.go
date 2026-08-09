package cart

import "testing"

// TestCart_RemoveItemRemovesLastMatch is the held-back F2P oracle for corpus
// item cart-remove-item-last-match (T-E40-F01-003). It fails at the
// fixture's base commit, where RemoveItem incorrectly refuses to remove a
// matching item when that item is the cart's last line item, and passes
// once reference.patch removes that erroneous condition.
func TestCart_RemoveItemRemovesLastMatch(t *testing.T) {
	c := New()
	c.AddItem(Item{SKU: "sku-1", Name: "Widget", Price: 500, Quantity: 2})
	c.AddItem(Item{SKU: "sku-2", Name: "Gadget", Price: 1200, Quantity: 1})

	if removed := c.RemoveItem("sku-2"); !removed {
		t.Fatal("RemoveItem(\"sku-2\") = false, want true")
	}
	if got, want := len(c.Items()), 1; got != want {
		t.Fatalf("len(Items()) = %d, want %d", got, want)
	}
	if got := c.Items()[0].SKU; got != "sku-1" {
		t.Fatalf("remaining item SKU = %q, want %q", got, "sku-1")
	}
}
