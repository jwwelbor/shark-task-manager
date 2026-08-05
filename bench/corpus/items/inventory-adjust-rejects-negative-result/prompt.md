# Task: Reject a stock adjustment that would go negative

`inventory.Stock.Adjust(sku, delta)` applies a manual correction of `delta`
(positive or negative) to `sku`'s available quantity, for reconciling stock
counts outside the normal reserve/release flow. It does not currently
validate the result: an adjustment that would drive the resulting available
quantity below zero is silently applied, leaving `Available` negative.

Add validation so that `Adjust` returns a clear, descriptive error and
leaves the stock level **unchanged** when the resulting quantity would be
negative, without changing behavior for any adjustment that keeps the
result at zero or above.
