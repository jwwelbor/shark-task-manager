---
inputs:
  - requirement_text: the requirement, user story, or spec section to decompose (prose)
  - intent_context: surrounding goals, success criteria, or parent requirement (optional)
  - domain_vocabulary: glossary of domain terms (optional)
  - existing_criteria: acceptance criteria already drafted, to refine or de-duplicate against (optional)
outputs:
  - acceptance_criteria: list of {criterion_id, statement, traces_to, class_hint}
  - open_questions: list of {question, location_in_requirement, why_it_blocks_testability}
  - hidden_decisions: list of vague terms that were made explicit, with the decision taken
---

# Workflow: Decompose a Requirement into Testable Criteria

## Purpose

Convert prose intent into a clean set of acceptance criteria that are unambiguous, testable, and
traceable. The deliverable is not a test matrix yet — it is the set of clear statements that the
matrix will later enumerate against. The most valuable output of this stage is often the list of
ambiguities it surfaces.

## Process

### Step 1: Read for Intent Before Reading for Detail

Read `requirement_text` once for what the requirement is *trying to achieve*, not its mechanics.
If `intent_context` is provided, read it first — the parent goal is the source of truth for what
"correct" means. A criterion that satisfies the literal words but violates the intent is a
decomposition failure.

Write a one-sentence statement of intent in your own words. If you cannot, the requirement is too
vague to decompose — record that as the first open question and stop.

### Step 2: Identify the Hidden Decisions

Scan the requirement for words that hide a decision. These are the words a test cannot be written
against until they are made concrete:

| Hidden-decision word | What it hides |
|---|---|
| "valid" / "invalid" | The exact membership rule for each set |
| "correctly" / "properly" | The exact expected output |
| "handles" / "manages" | The observable behavior (return, error, message, side effect) |
| "gracefully" | What the actor sees instead of a crash |
| "fast" / "responsive" | A number with a unit and a percentile |
| "secure" / "safe" | An enumerated set of threats that are defended against |
| "etc." / "and so on" | The members that were left unlisted |
| "appropriate" / "reasonable" | The judgement criterion |

For each hidden-decision word, either (a) resolve it from `intent_context` or `domain_vocabulary`
and record the decision in `hidden_decisions`, or (b) if it cannot be resolved, record it in
`open_questions`. Never guess silently — a silent guess becomes a defect later.

### Step 3: Split Compound Requirements

A single sentence frequently contains several requirements joined by "and", "or", "but",
"including", or a comma list. Split them. One criterion should assert one verifiable thing.

- "The form validates the email and saves the record" → two criteria (validation, persistence).
- "Accepts PNG, JPG, or WEBP up to 5 MB" → criteria for each accepted format, plus the size limit.

If splitting reveals an implicit dependency between the parts ("saves *only if* validation
passes"), capture that dependency as a precondition note on the dependent criterion.

### Step 4: Draft One Criterion per Verifiable Assertion

For each verifiable assertion, write a criterion in this shape:

```
GIVEN <precondition / starting state>
WHEN  <the action or trigger>
THEN  <the exact, observable expected outcome>
```

Rules for each criterion:

- The THEN must be observable and exact — a value, a state, a message, a side effect — not "works".
- Use `domain_vocabulary` terms precisely; do not introduce synonyms that subtly change meaning.
- Assign each criterion an ID (`AC-1`, `AC-2`, ...) and a `traces_to` reference to the
  sentence or goal it came from.
- Tag a `class_hint` describing the shape of the criterion, which tells the matrix step which
  enumeration technique to apply later:
  - **set** — accepts/rejects a membership of values (drives equivalence partitioning)
  - **range** — has an ordered numeric/size/length/date domain (drives boundary analysis)
  - **combination** — behavior depends on several conditions combining (drives a decision table)
  - **lifecycle** — an entity moves between states (drives state-transition enumeration)
  - **defensive** — asserts a protective property (drives threat-class enumeration)

### Step 5: Check Each Criterion Against the Decomposition Criteria

Apply the checks from `context/decomposition-criteria.md` to every criterion:

| Check | Question |
|---|---|
| Unambiguous | Would two readers write the same test from this? |
| Testable | Can a concrete input/output pair be written for it? |
| Traceable | Does it map to a sentence or goal in the requirement? |
| Atomic | Does it assert exactly one thing? |
| Outcome-specified | Is the expected result exact, not "correct"? |

Any criterion that fails a check is rewritten, split, or — if the failure is caused by missing
information — converted into an `open_question`.

### Step 6: Catch Open-Ended Defensive Criteria

Criteria with a `defensive` class hint ("must be immutable", "must be secure", "must reject bad
input") are the most common source of endless rework, because an unbounded property invites an
endless stream of new ways to violate it. For each defensive criterion, replace the open-ended
property with an enumerated model:

- Name the classes of threat or violation it must withstand (see the defensive section of
  `context/edge-case-catalog.md`).
- Restate the criterion as: "withstands the following enumerated classes: [list]".
- If you cannot enumerate the classes, the criterion is not yet testable — record it as an
  open question rather than passing an unbounded assertion downstream.

### Step 7: De-duplicate Against Existing Criteria

If `existing_criteria` was provided, compare each new criterion against it:

- **Duplicate** — drop the new one, note the existing ID it matches.
- **Refinement** — the new one is sharper; replace the old, note what changed.
- **Conflict** — the two disagree; record a conflict in `open_questions` for resolution.

### Step 8: Assemble the Output

Produce:

- `acceptance_criteria` — the clean, checked, de-duplicated list, each with ID, GIVEN/WHEN/THEN
  statement, `traces_to`, and `class_hint`.
- `hidden_decisions` — every vague term that was made explicit and the decision taken.
- `open_questions` — every unresolved ambiguity, with where it appears and why it blocks
  testability.

Hand `acceptance_criteria` to `workflows/build-test-matrix.md` to enumerate the full case set.

## What Good Decomposition Looks Like

**Good:**
- Every vague word is either resolved (with the decision recorded) or raised as a question.
- Compound requirements are split into atomic criteria.
- Expected outcomes are concrete and observable.
- Defensive properties are enumerated, not sloganized.

**Bad:**
- Rephrasing the requirement without removing any ambiguity.
- One giant criterion asserting five things.
- "Correctly handles errors" left intact.
- Silently assuming what "valid" means instead of asking.

## Quality Checklist

- [ ] Intent stated in one sentence (or flagged as too vague to decompose)
- [ ] Every hidden-decision word resolved or raised as an open question
- [ ] Compound requirements split into atomic criteria
- [ ] Each criterion uses GIVEN/WHEN/THEN with an exact, observable THEN
- [ ] Each criterion has an ID, a `traces_to`, and a `class_hint`
- [ ] Each criterion passes the unambiguous/testable/traceable/atomic/outcome checks
- [ ] Defensive criteria are enumerated, not open-ended
- [ ] De-duplicated against `existing_criteria` (if provided)
- [ ] Open questions captured rather than silently assumed
