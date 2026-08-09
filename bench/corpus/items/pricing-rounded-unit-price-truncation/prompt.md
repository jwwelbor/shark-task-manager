# Bug: RoundedUnitPrice rounds down instead of up

`pricing.RoundedUnitPrice(totalCents, quantity)` is documented to divide
`totalCents` by `quantity` and round the per-unit price **up** to the
nearest cent when the division has a remainder, so that
`quantity * RoundedUnitPrice(totalCents, quantity)` never undercharges the
original total.

Right now, `RoundedUnitPrice` rounds down (truncates) instead of up. For
example:

```go
pricing.RoundedUnitPrice(100, 3) // want: 34, got: 33
```

`34 * 3 = 102` covers the original 100 cents; `33 * 3 = 99` falls short by
one cent. When `totalCents` divides evenly by `quantity`, the result is
unaffected either way.

Fix `RoundedUnitPrice` so it rounds up whenever the division has a
remainder, without changing the result for an even division.
