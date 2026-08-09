# Task: Require SKU values to be uppercase-only

`validate.SKU(value)` currently checks that a SKU is non-empty and contains
no whitespace, but it accepts SKUs containing lowercase letters.

Add a rule to `SKU` that rejects any value containing a lowercase letter,
returning a clear, descriptive error in that case, in addition to its
existing checks. Values that already pass the existing checks and contain
no lowercase letters should continue to be accepted.
