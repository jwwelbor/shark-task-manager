---
name: breakdown-test
description: Decompose a requirement or specification section into testable acceptance criteria. Turns prose intent into an enumerated test matrix of conditions, expected outcomes, edge cases, and negative cases that any agent can hand to an implementer or test author.
when_to_use: when a requirement, user story, spec section, or acceptance criterion needs to be broken down into concrete, verifiable test conditions before implementation or test authoring begins
version: 1.0.0
domain: requirements-decomposition
inputs:
  - requirement_text: the requirement, user story, or spec section to decompose (prose)
  - intent_context: surrounding goals, success criteria, or parent requirement that establishes intent (optional)
  - known_constraints: non-functional constraints (performance, security, compatibility limits) relevant to the requirement (optional)
  - domain_vocabulary: glossary of domain terms so conditions use precise language (optional)
  - existing_criteria: acceptance criteria already drafted, to refine or de-duplicate against (optional)
outputs:
  - selected_workflow: one of {decompose-requirement, build-test-matrix}
  - acceptance_criteria: list of unambiguous, testable criteria derived from the requirement
  - test_matrix: list of {condition_id, criterion_ref, technique, precondition, input, expected_outcome, class}
  - edge_cases: enumerated boundary, empty, and limit conditions per criterion
  - negative_cases: enumerated "this MUST NOT happen" conditions per criterion
  - open_questions: ambiguities or gaps surfaced during decomposition that need clarification
---

# Breakdown-Test Skill

This skill provides a disciplined method for turning a requirement written in prose into a
structured, testable matrix. The output is the bridge between "what we want" (intent) and
"what we will verify" (concrete conditions with expected outcomes).

The skill does no implementing, scheduling, or routing. It produces a decomposition artifact —
a set of acceptance criteria and a test matrix — that is independently useful to any agent or
person who later writes code, writes tests, or reviews work.

## Core Principle

A requirement is not testable until every word that hides a decision has been made explicit.
"Handles invalid input correctly" hides three decisions: what counts as invalid, what "handles"
means, and what "correctly" produces. Decomposition is the act of replacing those hidden
decisions with enumerated conditions and exact expected outcomes.

## When to Use This Skill

Use these workflows when you have:

- A user story or requirement that needs acceptance criteria before work starts
- A spec section whose phrasing is vague, broad, or open-ended
- Draft acceptance criteria you want to harden into a concrete test matrix
- A defensive or non-functional requirement ("must be secure", "must be fast") that needs an
  enumerated model rather than a slogan

## Workflow Selection

### Decompose a Requirement into Criteria
**When**: You have prose intent and need a clean set of testable acceptance criteria.
**Invoke**: `workflows/decompose-requirement.md`
**Output**: A list of unambiguous criteria plus surfaced ambiguities (open questions).
**Use case**: First pass — converting "what we want" into "what we can check."

### Build a Full Test Matrix
**When**: You have acceptance criteria (your own or `existing_criteria`) and need the full
enumeration of conditions, expected outcomes, edge cases, and negative cases.
**Invoke**: `workflows/build-test-matrix.md`
**Output**: A condition-by-condition test matrix ready to hand to a test author.
**Use case**: Second pass — turning each criterion into the exhaustive set of cases that prove it.

The two workflows compose: run `decompose-requirement` first to get clean criteria, then
`build-test-matrix` to enumerate every case. For an already-clean set of criteria, go straight
to `build-test-matrix`.

## Context Resources

All workflows draw on these reference files:

- `context/decomposition-criteria.md` — what makes a criterion unambiguous, testable, traceable, and complete
- `context/edge-case-catalog.md` — a systematic catalog of edge, boundary, and negative case classes to enumerate against
- `context/test-matrix-template.md` — the output template for the decomposition artifact

## Output Standards

Every decomposition produced by this skill must:

1. **Trace to intent** — each criterion maps back to a sentence or goal in the requirement.
2. **Be unambiguous** — two independent readers would write the same test from it.
3. **Specify exact outcomes** — expected results are concrete values or observable behavior, not "works".
4. **Cover the three classes** — positive (happy path), edge (boundary/limit), and negative (must-not-happen).
5. **Surface gaps, not paper over them** — ambiguities become `open_questions`, never silent assumptions.

## What This Skill Does NOT Do

- It does not write code or tests — it produces the specification those are built from.
- It does not decide order of work or who does it.
- It does not assume any particular tooling, framework, or runtime.
