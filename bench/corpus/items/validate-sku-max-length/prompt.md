# Task: Add a maximum length limit to SKU validation

`validate.SKU(value)` checks that a SKU is non-empty and contains no
whitespace, but currently accepts a SKU of any length.

Add a maximum SKU length of 40 characters. `SKU` should return a clear,
descriptive error when given a value longer than 40 characters, in addition
to its existing checks. Values of 40 characters or fewer that already pass
the existing checks should continue to be accepted.
