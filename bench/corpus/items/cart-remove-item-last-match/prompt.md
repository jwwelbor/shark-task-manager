# Bug: Removing the cart's last line item silently fails

`cart.Cart.RemoveItem(sku)` is supposed to remove the first line item whose
`SKU` matches `sku` and report `true`. If no item matches, it should leave
the cart unchanged and report `false`.

Right now, removal works correctly when the matching item is anywhere
except the last line item in the cart. When the matching item happens to be
the cart's last line item, `RemoveItem` incorrectly leaves it in place and
returns `false`, as if nothing matched. For example:

```go
c := cart.New()
c.AddItem(cart.Item{SKU: "sku-1", Name: "Widget", Price: 500, Quantity: 2})
c.AddItem(cart.Item{SKU: "sku-2", Name: "Gadget", Price: 1200, Quantity: 1})

removed := c.RemoveItem("sku-2") // want: true, got: false
// c.Items() still contains both items
```

Fix `RemoveItem` so it removes the first matching item regardless of its
position in the cart, without changing behavior when no item matches.
