# Decomposition Criteria

This document defines what makes an acceptance criterion good enough to test against. Apply these
checks during `decompose-requirement` (when drafting criteria) and again at the start of
`build-test-matrix` (before enumerating cases).

## The Five Properties of a Testable Criterion

A criterion is ready when it satisfies all five. A failure on any one sends the criterion back for
rewriting, splitting, or — if the failure is caused by missing information — into open questions.

### 1. Unambiguous

Two independent readers would write the same test from it. There is exactly one reasonable
interpretation.

- **Pass:** "A password shorter than 8 characters is rejected with the message 'Password too short'."
- **Fail:** "The password is validated." (Validated against what? What happens on failure?)

### 2. Testable

A concrete input/output pair can be written for it. If you cannot imagine the input you would feed
and the output you would assert, it is not testable.

- **Pass:** "Uploading a 6 MB file returns a 'file too large' rejection."
- **Fail:** "The system is performant." (No input, no measurable output.)

### 3. Traceable

It maps back to a specific sentence, goal, or parent requirement. Every criterion exists because
something asked for it. Orphan criteria are scope creep; missing traces are coverage gaps.

### 4. Atomic

It asserts exactly one verifiable thing. Compound criteria ("validates AND saves AND notifies")
hide partial failures — a test could pass on two of three and still report green.

### 5. Outcome-Specified

The expected result is a concrete value, state, message, or side effect — never "works",
"correctly", or "as expected".

- **Pass:** "Returns an empty list and a count of 0."
- **Fail:** "Returns the right result."

## Vague-Word Glossary

These words almost always hide a decision. When you find one, resolve it (and record the decision)
or raise it as an open question.

| Word | The decision it hides |
|---|---|
| valid / invalid | The exact membership rule of each set |
| correctly / properly / right | The exact expected output |
| handles / manages / processes | The observable behavior produced |
| gracefully | What the actor sees instead of a failure |
| fast / quick / responsive | A number, a unit, and a percentile |
| secure / safe / robust | An enumerated set of defended-against threats |
| appropriate / reasonable / sensible | The judgement rule being applied |
| etc. / and so on / among others | The unlisted members |
| supports | The specific operations and their outcomes |
| user-friendly | The specific observable usability behaviors |

## Open-Ended Property Anti-Pattern

Criteria that assert an unbounded protective property ("must be immutable", "must be secure",
"must never lose data") cannot be exhaustively tested, because the property invites an endless
stream of new ways to violate it. The fix is always the same: replace the open-ended property with
an **enumerated model** of the classes of violation it must withstand. See the defensive section of
`edge-case-catalog.md` for the standard classes.

A criterion that asserts an unbounded property without an enumerated model is treated as not yet
testable.

## Severity of Decomposition Defects

When reviewing a set of criteria, classify each problem:

- **Blocker** — the criterion is untestable as written (vague outcome, hidden decision unresolved,
  open-ended property). Cannot proceed to the matrix.
- **Major** — the criterion is testable but non-atomic or weakly traced. Should be fixed before
  the matrix to avoid ambiguous coverage.
- **Minor** — wording or terminology inconsistency that does not change the test that would be
  written. Fix opportunistically.

## Self-Check Questions

For any criterion, ask in order:

1. Can I write the input I would feed it? (Testable)
2. Can I write the exact output I would assert? (Outcome-specified)
3. Would another reader assert the same output? (Unambiguous)
4. Does it assert only one thing? (Atomic)
5. Where in the requirement did it come from? (Traceable)

A "no" to any question is a defect, not a judgement call.
