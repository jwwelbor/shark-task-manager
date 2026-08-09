# Task: Reject negative quantities when adding a cart item

`Cart.AddItem(item)` currently appends any item to the cart, including one
with a negative `Quantity`, which lets `Cart.ItemCount()` go negative.

Change `AddItem` so an item with a negative `Quantity` is silently ignored
and never added to the cart. Items with a zero or positive `Quantity`
should continue to be added exactly as they are today.
