# Change: replace the three-level priority scale with a 1-5 scale

Our task priority field currently accepts three string levels -- `"high"`,
`"medium"`, and `"low"` -- which isn't granular enough for how the team
actually triages work.

Please change the priority field to a numeric 1-5 scale instead, where 1 is
the most urgent and 5 is the least:

| Priority | Label      |
|----------|------------|
| 1        | critical   |
| 2        | high       |
| 3        | medium     |
| 4        | low        |
| 5        | trivial    |

Requirements:

- Tasks should be created and validated against the new 1-5 integer scale.
  A priority outside 1-5 must be rejected.
- We already have records stored under the old three-level scale that need
  to be converted, not just newly created tasks going forward. Convert each
  existing record's priority using the natural mapping onto the new scale
  (`"high"` -> 2, `"medium"` -> 3, `"low"` -> 4), preserving every other
  field on the record (title, status, due date).
- Existing behavior for anything unrelated to priority (task completion,
  overdue checks, title validation) must keep working exactly as before.
