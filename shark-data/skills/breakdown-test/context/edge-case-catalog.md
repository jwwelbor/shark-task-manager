# Edge-Case Catalog

A systematic catalog of edge, boundary, and negative case classes to enumerate against during
`build-test-matrix`. The point of a catalog is to make edge-case discovery mechanical rather than
inspirational — you walk the list against each criterion instead of hoping to remember the cases.

For each criterion, go through every section below. Each applicable class becomes at least one row
in the matrix. Each clearly inapplicable class is marked "N/A — [reason]" so the omission is a
decision, not an oversight.

## 1. Boundary Classes (for ordered domains: numbers, sizes, lengths, dates)

For any minimum or maximum the criterion mentions or implies, enumerate the values around it:

- `min − 1` (just below — should be rejected)
- `min` (the boundary — should be accepted)
- `min + 1` (just inside)
- `max − 1` (just inside)
- `max` (the boundary — should be accepted)
- `max + 1` (just above — should be rejected)

Also consider domain-specific edges: zero, negative one, the largest representable value, dates at
period boundaries (start/end of day, month, year), leap-day, and timezone transitions.

## 2. Emptiness and Absence

- Empty collection / empty string / empty file.
- Null, missing, or omitted field where one is expected.
- Whitespace-only input where content is expected.
- A field present but set to a "no value" sentinel (zero, empty, default).

## 3. Multiplicity

- Exactly zero items.
- Exactly one item.
- Exactly the maximum allowed.
- One over the maximum.
- A very large quantity (orders of magnitude beyond the typical case).
- Duplicate items where uniqueness is expected.

## 4. Ordering and Timing (for domains with sequence or concurrency)

- First and last element of a sequence.
- Out-of-order arrival.
- Simultaneous or near-simultaneous actions on the same target.
- Repeated identical actions (does the second one duplicate, no-op, or error?).
- An action arriving before its precondition is met.

## 5. Format and Encoding

- Mixed case where case may matter.
- Leading/trailing whitespace.
- Unusual but legal characters (unicode, accents, emoji, right-to-left text).
- Maximum-length and minimum-length representations.
- Inputs that are valid in one interpretation but not another (ambiguous format).

## 6. Defensive / Threat Classes (for criteria asserting a protective property)

When a criterion asserts that something must be protected, enumerate one case per class rather than
one example. These are the classes that close an otherwise unbounded property:

- **Mutation** — attempts to change something that must stay fixed: direct change, change via a
  collection operation, change via a nested/contained value, change via a derived view.
- **Injection** — hostile content supplied through each input surface that reaches an interpreter,
  query, template, command, or serializer.
- **Bypass** — attempts to reach a protected outcome without passing the guard: skipping a step,
  forging a precondition, using an alternate entry path, replaying a prior valid request.
- **Exhaustion** — inputs that stress a limit: very large size, deep nesting, long-running or
  recursive structure, rapid repetition.
- **Confused state** — partially-completed, interrupted, or contradictory inputs that probe whether
  the protection holds when the actor is in an unexpected state.

For each enumerated class, write a negative row: the hostile input AND the exact expected
defensive response (rejection, unchanged state, error condition, sanitized output).

## 7. Interaction and Dependency

- The target does not exist / was already removed.
- A dependency the criterion relies on is unavailable or returns nothing.
- A related entity is in an incompatible state.
- The same input under two different surrounding states (does context change the outcome?).

## Using the Catalog

1. For each acceptance criterion, read its statement and `class_hint`.
2. Walk sections 1–7. For each class, decide: applicable or N/A.
3. For applicable classes, write a matrix row with a concrete input and an exact expected outcome.
4. For N/A classes, record a one-line reason.
5. The criterion's edge and negative coverage is complete when every section has been decided —
   covered or N/A — with no section silently skipped.

## A Note on Stopping

A catalog walk has a natural end: every section decided once. This is the antidote to the
"discover one more case forever" problem. If new cases keep appearing, they almost always belong to
a class that was marked N/A too quickly — revisit that class rather than appending ad-hoc cases.
