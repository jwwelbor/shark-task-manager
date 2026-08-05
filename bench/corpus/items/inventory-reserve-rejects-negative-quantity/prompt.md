# Task: Reject a negative quantity in Reserve

`inventory.Stock.Reserve(sku, quantity)` is meant to decrement `sku`'s
available quantity by `quantity`, or fail if not enough stock is available.
It does not currently validate that `quantity` is non-negative. Calling
`Reserve` with a negative quantity is silently accepted and *increases*
available stock instead of reserving anything, which bypasses the intended
reservation semantics.

Add validation so that `Reserve` returns a clear, descriptive error when
`quantity` is negative and leaves the stock level unchanged, without
changing behavior for any non-negative `quantity`.
