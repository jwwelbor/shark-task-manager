# Bug: A freshly created cart reports a non-zero subtotal

Reports from checkout QA suggest that `cart.New()` sometimes returns a Cart
whose `Subtotal()` and `ItemCount()` are non-zero before any item has been
added, which would make the checkout summary screen show a stale total.

Investigate `cart.New()`, `Cart.Subtotal()`, and `Cart.ItemCount()` and fix
whatever causes a brand-new, empty Cart to report a non-zero subtotal or
item count.
