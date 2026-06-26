---
inputs:
  - acceptance_criteria: list of {criterion_id, statement, traces_to, class_hint} (from decompose-requirement, or pre-existing)
  - known_constraints: non-functional constraints relevant to the criteria (optional)
  - domain_vocabulary: glossary of domain terms (optional)
outputs:
  - test_matrix: list of {condition_id, criterion_ref, technique, precondition, input, expected_outcome, class}
  - edge_cases: enumerated boundary/empty/limit conditions per criterion
  - negative_cases: enumerated "this MUST NOT happen" conditions per criterion
  - coverage_notes: per-criterion note of which technique was applied and why
  - open_questions: ambiguities discovered during enumeration that block a row from being written
---

# Workflow: Build a Test Matrix from Acceptance Criteria

## Purpose

Take a set of clean acceptance criteria and enumerate, for each one, the full set of test
conditions: positive cases, edge/boundary cases, and negative cases — each with a concrete input
and an exact expected outcome. The output is a matrix a test author can translate directly,
without re-interpreting intent.

The discipline here is **enumeration by class, not by example**. You do not list a few cases that
come to mind; you choose a technique that forces the input/outcome surface to be fully partitioned,
then write one case per partition. This is what prevents the "we keep discovering one more case"
spiral.

## Process

### Step 1: Confirm Each Criterion Is Matrix-Ready

For each criterion, confirm it is unambiguous and has an exact expected outcome (per
`context/decomposition-criteria.md`). If a criterion is still vague, it cannot be enumerated —
record an open question and skip its rows rather than inventing an interpretation.

### Step 2: Choose an Enumeration Technique per Criterion

Use the criterion's `class_hint` (or infer it from the statement) to pick the technique that
forces full coverage. Each technique dictates *what* you must enumerate.

| Criterion shape | Technique | What it forces you to enumerate |
|---|---|---|
| **set** — accepts/rejects values | Equivalence partitioning | One representative per valid class + one per invalid class — not one per individual value |
| **range** — ordered numeric/size/length/date domain | Boundary analysis | min−1, min, min+1, … , max−1, max, max+1; plus empty/zero/negative where meaningful |
| **combination** — behavior depends on several conditions | Decision table | Every reachable combination of conditions × the resulting outcome — proves nothing falls through |
| **lifecycle** — entity moves between states | State-transition enumeration | Every legal transition + every illegal transition — proves no missing edge and no accidental shortcut |
| **defensive** — protective property | Threat-class enumeration | One case per class of violation (mutation, injection, bypass, exhaustion) drawn from `context/edge-case-catalog.md` |

A single criterion may need more than one technique (e.g., a range that is also defended against
out-of-range injection). Record the technique(s) chosen in `coverage_notes` with a one-line
rationale. A criterion with no technique annotation is treated as un-enumerated and sent back.

### Step 3: Enumerate Positive Conditions

For each criterion, write the rows that prove the intended behavior happens:

- One row per valid equivalence class / legal transition / reachable true-combination.
- Each row gets a concrete `input` (real values, not "some value") and an exact
  `expected_outcome` (real values/state/message, not "succeeds").
- Tag each row `class: positive`.

### Step 4: Enumerate Edge and Boundary Conditions

Walk `context/edge-case-catalog.md` against each criterion and write a row for every applicable
boundary or limit condition:

- Numeric/size/length/date boundaries (the BVA set from Step 2).
- Emptiness and absence (empty collection, empty string, null/missing field, zero).
- Multiplicity (exactly one, exactly the maximum, one over the maximum, very large).
- Ordering and timing (first, last, simultaneous, out-of-order) where the domain has order.

Tag each row `class: edge`. Collect them into `edge_cases` grouped by criterion. If a catalog
class clearly does not apply to a criterion, note "N/A — [reason]" rather than leaving a gap, so
the omission is a decision and not an oversight.

### Step 5: Enumerate Negative Conditions

For every criterion, write at least one row describing what MUST NOT happen and the exact
observable result when the forbidden input is supplied:

- Invalid members of each rejected equivalence class.
- Every illegal state transition (for lifecycle criteria).
- Each enumerated threat class (for defensive criteria).
- Each false-combination in a decision table that should be refused.

A negative row must state both the forbidden input AND the exact expected response (rejection
value, error condition, unchanged state, message) — "it should fail" is not a complete expected
outcome. Tag each row `class: negative`. Collect them into `negative_cases` grouped by criterion.

### Step 6: Cover Non-Functional Constraints

If `known_constraints` is provided, attach the relevant ones to the criteria they bound, and write
rows that make each constraint checkable with a concrete threshold:

- Performance → an input scale + a numeric limit + a percentile ("p95 under N ms at M items").
- Compatibility → the specific environments/versions the behavior must hold across.
- Reliability → the failure to inject and the expected recovery/retry/idempotent outcome.
- Security → the threat-class rows already enumerated in Step 5, made concrete.

A constraint with no row is an untested constraint — either write the row or record why coverage
is deferred in `coverage_notes`.

### Step 7: Assign IDs and Assemble the Matrix

Give every row a `condition_id` (`TC-AC1-01`, `TC-AC1-02`, …) and a `criterion_ref` back to its
acceptance criterion. Assemble `test_matrix` using the structure in
`context/test-matrix-template.md`. Every row must have: precondition, concrete input, exact
expected outcome, technique, and class.

### Step 8: Self-Check Coverage

Before finishing, verify:

- Every criterion has at least one positive, the applicable edge rows, and at least one negative.
- Every row has a concrete input and an exact expected outcome — no placeholders, no "works".
- Every criterion has a technique recorded in `coverage_notes`.
- Every applicable edge-case-catalog class is either covered or explicitly marked N/A.
- Any ambiguity that blocked a row is in `open_questions`, not silently resolved.

## What a Good Test Matrix Looks Like

**Good:**
- Cases are derived from a named technique, so coverage is provably complete at the chosen level.
- Inputs and outcomes are concrete enough that two authors would write equivalent tests.
- Negative and edge classes are enumerated, each with its exact expected response.
- Non-functional constraints have concrete, threshold-bearing rows.

**Bad:**
- A handful of happy-path rows and nothing else.
- "Input: invalid data, Expected: error" with no specifics.
- Boundary values omitted because the happy path "obviously works".
- A defensive criterion with one example instead of an enumerated threat model.

## Quality Checklist

- [ ] Every criterion confirmed matrix-ready (or its rows skipped with an open question)
- [ ] A technique chosen and recorded for every criterion
- [ ] Positive rows cover every valid class / legal transition / true-combination
- [ ] Edge rows written for every applicable catalog class (or marked N/A with reason)
- [ ] At least one negative row per criterion, each with an exact expected response
- [ ] Non-functional constraints attached and given threshold-bearing rows
- [ ] Every row has a unique ID, concrete input, and exact expected outcome
- [ ] Coverage self-check passed; open questions captured
