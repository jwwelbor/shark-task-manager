---
inputs:
  - ambiguity_report: output of detect-ambiguity (spans, types, severities)
  - requirement_text: the original requirement being clarified
  - audience: who will answer (end user, stakeholder, domain expert)
  - prior_clarifications: previously resolved points to avoid re-asking (optional)
outputs:
  - clarifying_questions: ordered questions, each tied to an ambiguity and the decision it unblocks
  - clarified_requirement: the rewritten requirement after answers are incorporated
  - residual_assumptions: any ambiguity intentionally left as a documented assumption
---

# Workflow: Question Ladder

**Purpose**: Convert a set of detected ambiguities into the *minimum* ordered set of questions that, once answered, collapse the requirement into a single testable reading.
**Use for**: Live or asynchronous clarification with someone who can answer.
**Output**: An ordered question set and the rewritten requirement.

## Overview

A question ladder is an *ordered* sequence — not a flat list — because answers cascade. The answer to a high-leverage question often dissolves three lower questions, or reframes them entirely. Asking in the wrong order wastes the answerer's attention and forces extra round-trips.

The two governing rules:

1. **Every question must change a decision.** If both possible answers lead you to do the same thing, do not ask.
2. **Ask the fewest questions that get you to testable.** The goal is not maximal understanding; it is enough understanding to proceed without guessing on anything high-severity.

## Process

### Phase 1: Filter — drop questions that don't change a decision

Take every finding from the ambiguity report and apply the decision test:

> For interpretation A vs B, do I take a *different action*?

- **No** → discard. Record it under residual notes ("resolved by indifference — both readings yield the same action").
- **Yes** → keep. This finding earns a question.

This step alone typically removes a third of candidate questions.

### Phase 2: Order by leverage

Sort the surviving findings so that **answering one shrinks the others**. Use this priority:

1. **Intent first.** A question that establishes the goal/why. Its answer often reframes or eliminates everything below it.
2. **Scope next.** What's in and out. Defines the boundary the rest live inside.
3. **Then structural / conditional.** Rules, branches, edge handling.
4. **Then definitions and quantifiers last.** Thresholds and term meanings are cheap to pin down once the goal and scope are fixed — and sometimes the goal makes the exact number obvious.

Within a tier, order by severity: high before medium.

### Phase 3: Shape each question

For each surviving finding, choose the question *form* using `../context/question-templates.md`. Key choices:

- **Open vs closed.** Use an **open** question when you don't yet understand the space ("What should happen when the upload fails midway?"). Use a **closed / multiple-choice** question when you have already enumerated the interpretations and just need a pick ("When a large file is uploaded, should the user (a) be warned only, or (b) confirm before processing?").
- **Offer the interpretations you already found.** Closed questions built from your A/B interpretations are faster to answer and confirm you understood the options. They also catch the case where the real answer is a *third* option you missed.
- **One decision per question.** Never bundle two ambiguities into one sentence — the answerer will address one and ignore the other.
- **Carry the "why it matters" lightly.** A one-line reason ("this changes whether the flow blocks") helps the answerer give a useful answer, but don't lecture.

### Phase 4: Attach the proceed-with-assumptions fallback

For each question, pre-write the assumption you will adopt **if it goes unanswered**. This does two things:
- It lets you proceed if the answerer goes silent (the ladder degrades gracefully into an assumption register — see `surface-assumptions.md`).
- Stating "if I don't hear back, I'll assume A" often prompts a faster correction than an open question would.

### Phase 5: Ask, then fold answers back in

Present the ladder (top-leverage first). As answers arrive:

1. Apply the answer to the requirement text.
2. **Re-scan** — an answer can introduce a new ambiguity ("process in background" answered, but now "notify when done" — notify how?). Add a rung if it is high-severity; otherwise note it as residual.
3. Stop when no high-severity ambiguity remains, even if lower-tier questions are still open. Those become residual assumptions.

### Phase 6: Rewrite to testable

Produce the clarified requirement. It must pass the bar in `../context/clarification-criteria.md`: an independent party reading it would build or verify the same thing. Quote the resolved decisions inline rather than referencing "as discussed".

## Output Format

```markdown
# Clarification Ladder: {requirement label}

## Questions (ask top-down)

1. [INTENT · high] {question}
   - Unblocks: {the decision this changes}
   - Form: open | closed
   - If unanswered, I will assume: {fallback}

2. [SCOPE · high] {question}
   - Unblocks: {decision}
   - Options: (a) {…} (b) {…}
   - If unanswered, I will assume: {fallback}

3. [DEFINITION · medium] {question}
   ...

## Dropped (no decision change)
- {finding} — both readings lead to the same action.

## Clarified Requirement
{rewritten, testable requirement with resolved decisions inline}

## Residual Assumptions
- {low-severity items left as documented assumptions}
```

## Worked Example

Carrying forward the upload example from `detect-ambiguity.md` (findings A1 large-file threshold, A2 warning-vs-confirm, A3 background scope):

```markdown
## Questions (ask top-down)

1. [INTENT · high] What problem is the warning solving — are we protecting the
   user from a slow/expensive operation, or protecting the system from load?
   - Unblocks: whether the warning must block (protect user's choice) or can be
     informational (protect system, but proceed anyway).
   - Form: open
   - If unanswered, I will assume: protecting the user → confirmation required.

2. [SCOPE · medium] Should background processing apply to ALL uploads, or only
   ones above the size threshold?
   - Unblocks: behavior for the common, non-large case (A3).
   - Options: (a) only large files (b) all uploads
   - If unanswered, I will assume: (a) only large files — narrower change.

3. [DEFINITION · high] What counts as "large"? Is there an existing size limit
   we should reuse, or a new threshold?
   - Unblocks: the exact trigger point for both warning and background (A1).
   - Form: closed-ish — expects a number or a reference to an existing limit.
   - If unanswered, I will assume: reuse the existing documented upload limit.

## Clarified Requirement
When a user uploads a file larger than {threshold from Q3}, present a
confirmation the user must accept before the upload is processed. Once accepted,
process that file asynchronously; uploads at or below the threshold continue to
process inline as today.
```

Note the ordering: intent (Q1) was asked first because its answer determines whether A2 is "warn" or "confirm" — answering Q1 *resolves* A2 without a separate question for it.

## Success Criteria

The ladder is complete when:
- [ ] Every question maps to a finding *and* a decision that its answer changes
- [ ] Questions that fail the decision test were dropped and recorded
- [ ] Questions are ordered by leverage (intent → scope → structure → definitions)
- [ ] Each question carries a proceed-with-assumptions fallback
- [ ] Answers were folded back in and the text re-scanned for new ambiguity
- [ ] The rewritten requirement is testable per the clarification criteria
- [ ] Residual (low-severity) items are listed as assumptions, not silently dropped

## Related

- `../context/question-templates.md` — question forms and sequencing rules
- `../context/clarification-criteria.md` — the "testable" bar and severity rubric
- `detect-ambiguity.md` — produces the input ambiguity report
- `surface-assumptions.md` — where unanswered rungs go
