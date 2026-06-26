---
name: clarification
description: Techniques for resolving ambiguous requirements. Detect ambiguity, surface hidden assumptions, and use question ladders to turn an unclear requirement into a precise, testable one. Use whenever a requirement, request, or spec is vague, underspecified, or open to multiple interpretations.
when_to_use: when a requirement is unclear, underspecified, or could be read more than one way — before designing, estimating, or implementing against it
version: 1.0.0
domain: requirements-clarification
inputs:
  - requirement_text: the raw requirement, request, ticket, or instruction to clarify
  - context: surrounding information (parent goal, domain, prior decisions) (optional)
  - constraints: known non-negotiables — deadlines, platforms, scope limits (optional)
  - audience: who can answer clarifying questions — end user, stakeholder, or none available (optional)
  - prior_clarifications: previously resolved questions to avoid re-asking (optional)
outputs:
  - selected_workflow: one of {detect-ambiguity, question-ladder, surface-assumptions}
  - ambiguity_report: structured list of {span, ambiguity_type, interpretations, why_it_matters, severity}
  - clarifying_questions: ordered questions, each tied to a specific ambiguity and the decision it unblocks
  - assumption_register: explicit assumptions with {assumption, confidence, blast_radius, validation_method}
  - clarified_requirement: the rewritten requirement once questions are answered or assumptions are accepted
---

# Clarification Skill

## Overview

This skill provides systematic techniques for turning a vague requirement into a precise, testable one. It treats ambiguity as a measurable property of text, not a vibe. The core moves are:

1. **Detect** — locate the exact spans that admit more than one reading
2. **Surface** — make the silent assumptions visible so they can be confirmed or denied
3. **Question** — ask the minimum set of questions, in the right order, to collapse the interpretations into one

The deliverable is always the same shape: an unclear requirement goes in, and either a *clarified requirement* or an *explicit set of surfaced assumptions* comes out. Nothing downstream — design, estimation, implementation, testing — should proceed on a requirement that still has unresolved high-severity ambiguity.

## When to Use This Skill

Use clarification workflows when:

- A requirement, ticket, or instruction can be read in more than one way
- Key terms are undefined or used loosely ("fast", "secure", "the user", "etc.")
- Scope boundaries are fuzzy — it is unclear what is in and what is out
- A request implies a goal but never states it
- You are about to make a decision whose correctness depends on an interpretation you have not confirmed
- An estimate, design, or implementation would have to *guess* to proceed

Do **not** reach for this skill when the requirement is already precise and testable. Over-clarifying a clear requirement wastes the answerer's attention and signals low judgment.

## Workflow Selection

Pick the workflow that matches where you are in the clarification process. Most real clarification runs use **detect-ambiguity first**, then branch into one or both of the others.

### Detect Ambiguity (start here)
**When**: You have a raw requirement and need to know *whether* and *where* it is unclear.
**Invoke**: `workflows/detect-ambiguity.md`
**Output**: An ambiguity report — a list of specific spans, each tagged with an ambiguity type and a severity.
**Use case**: Triage. Decide if clarification is even needed, and if so, where the effort should go.

### Surface Assumptions
**When**: The requirement reads as clear, but only because the reader is silently filling gaps. You want those gaps made explicit.
**Invoke**: `workflows/surface-assumptions.md`
**Output**: An assumption register — each assumption with confidence, blast radius, and how it could be validated.
**Use case**: No one is available to answer questions, or you want to proceed-with-assumptions and document exactly what you bet on.

### Question Ladder
**When**: You have located the ambiguities and someone *is* available to answer.
**Invoke**: `workflows/question-ladder.md`
**Output**: An ordered set of clarifying questions, each tied to a specific ambiguity and the decision it unblocks, plus the rewritten requirement once answered.
**Use case**: Live or asynchronous clarification with a user or stakeholder. Minimizes question count and round-trips.

## Context Files

All workflows draw on these shared references:

- `context/ambiguity-taxonomy.md` — the catalog of ambiguity types (referential, scope, quantifier, term-definition, intent, conditional, completeness) with detection cues and worked examples
- `context/question-templates.md` — reusable question forms for each ambiguity type, plus rules for sequencing and keeping question count low
- `context/clarification-criteria.md` — the bar a requirement must clear to count as "clarified", and severity rubric for prioritizing which ambiguities to chase

## Core Principles

1. **Ambiguity is local.** Tag specific spans of text, not the whole requirement. "It's vague" is not a finding; "the word *recent* has no defined window" is.

2. **Every question must earn its place.** Each clarifying question should map to (a) a concrete ambiguity and (b) a decision that changes depending on the answer. If both interpretations lead to the same action, do not ask — note it and move on.

3. **Severity drives effort.** A misread that quietly produces the wrong outcome is high severity. A misread that fails loudly and cheaply is low severity. Chase high-severity ambiguity first; let low-severity ambiguity ride as a documented assumption.

4. **Prefer the closed question late, the open question early.** Open questions ("what does *done* mean here?") expand the space; closed questions ("is X in scope: yes/no?") collapse it. Open early to understand, close late to commit.

5. **Always offer an escape hatch: proceed-with-assumptions.** When no answer is available, do not stall. Convert the open question into an explicit, documented assumption with a stated blast radius, and proceed. The assumption register is itself a valid output.

6. **A clarified requirement is testable.** The exit condition is not "we discussed it" — it is that the rewritten requirement could be handed to an independent party who would build or verify the same thing.

## Process Sketch

```
Raw requirement
   ↓
detect-ambiguity  → ambiguity_report (spans + types + severity)
   ↓
Any high-severity ambiguity?
   ├─ No  → requirement is clear enough → done
   └─ Yes ↓
          Is an answerer available?
          ├─ Yes → question-ladder    → clarified_requirement
          └─ No  → surface-assumptions → assumption_register (proceed-with-assumptions)
   ↓
Re-check against context/clarification-criteria.md
   ↓
clarified_requirement OR documented assumption_register
```

## Anti-Patterns to Avoid

- **Question dumping** — sending a wall of 15 questions instead of the 3 that actually matter
- **Asking what you can read** — asking the answerer something the surrounding context already states
- **Vague findings** — "this is confusing" instead of pointing at the exact span and reading
- **Clarifying forever** — chasing every theoretical ambiguity past the point where the requirement is testable
- **Silent assumptions** — proceeding on an interpretation without writing it down, so no one can challenge it
- **False precision** — rewriting the requirement to *sound* exact while smuggling in your own unverified guess

## Success Criteria

Clarification is complete when:

- Every high-severity ambiguity has been resolved by an answer or converted to a documented assumption
- The rewritten requirement is testable — an independent party would build/verify the same thing
- Each asked question changed an actual decision (no wasted questions)
- Remaining low-severity ambiguities are explicitly listed, not silently absorbed
- The output (clarified requirement or assumption register) can stand alone without you in the room

---

Select a workflow:
- Triage / locate ambiguity → `workflows/detect-ambiguity.md`
- Make silent assumptions explicit → `workflows/surface-assumptions.md`
- Ask and resolve → `workflows/question-ladder.md`
