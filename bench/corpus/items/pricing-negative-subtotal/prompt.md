# Task: Reject a negative subtotal in ApplyDiscount

`pricing.ApplyDiscount(subtotalCents, percent)` validates that `percent` is
between 0 and 100, but does not validate `subtotalCents`. A negative
`subtotalCents` is currently accepted and silently produces a negative
"discounted" total, which is not a meaningful checkout state.

Add validation so that `ApplyDiscount` returns a clear, descriptive error
when `subtotalCents` is negative, without changing behavior for any
non-negative `subtotalCents`.
