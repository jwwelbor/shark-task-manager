# Bug: Discount amount is rounded to the nearest cent instead of down

`pricing.ApplyDiscount(subtotalCents, percent)` is documented to reduce
`subtotalCents` by `percent` and return the discounted total, "rounded
down." In practice the discount is currently rounded to the *nearest* cent,
which can overstate the discount by a cent compared to the documented
behavior. For example:

```go
total, _ := pricing.ApplyDiscount(999, 33) // want: 670, got: 669
```

`999 * 33 / 100 = 329.67`, which should floor to a discount of `329` cents
(total `670`), not round to `330` (total `669`).

Fix `ApplyDiscount` so the discount is always rounded down (floor), matching
its documented behavior, without changing behavior for inputs where the
discount is already an exact integer.
