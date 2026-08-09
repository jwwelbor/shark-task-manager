# Bug: Reserving the exact available quantity fails

`inventory.Stock.Reserve(sku, quantity)` is supposed to succeed whenever
`quantity` is less than or equal to the SKU's currently available stock, and
fail with `ErrInsufficientStock` only when the reservation would exceed what
is available.

Right now, reserving a quantity that exactly equals the available stock
incorrectly fails with `ErrInsufficientStock`. For example:

```go
s := inventory.NewStock()
s.Set("sku-1", 5)

err := s.Reserve("sku-1", 5) // want: nil, got: ErrInsufficientStock
```

Reserving any amount strictly less than the available stock works correctly
today; only the exact-match case is affected.

Fix `Reserve` so that reserving a quantity equal to the available stock
succeeds and leaves `Available` at zero, without changing the behavior for
any other quantity.
