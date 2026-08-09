# Task: Reject a negative fee in ApplyFlatFee

`pricing.ApplyFlatFee(subtotalCents, feeCents)` adds a flat fee to a
subtotal and returns the new total. It does not currently validate that
`feeCents` is non-negative, silently accepting a negative fee and reducing
the total in a way that is not a meaningful checkout state.

Add validation so that `ApplyFlatFee` returns a clear, descriptive error
when `feeCents` is negative, without changing behavior for any
non-negative `feeCents`.
