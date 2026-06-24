# D01 — Vision Statement

Produce the vision statement (`D01-vision-statement.md`) through structured elicitation with the human stakeholder.

> **Read first:** `context/vision-anatomy.md` — components, anti-patterns, worked example.

**Builds on:** Nothing. This is the foundational product-design artifact; everything else traces back to it.

## Operating Principles

1. **One question per turn.** Use `AskUserQuestion`. Vision work rewards depth; rapid-fire questions get shallow answers.
2. **Echo and confirm.** After every 2–3 answers, paraphrase back. Misunderstandings caught here cost nothing; caught later they cost weeks.
3. **Push for the no.** If the user agrees with everything, widen the scope until they push back — the no locates the real boundary.
4. **Don't drift into solutioning.** If the user starts describing UI or tech stack, redirect: "Hold that for design — for the vision, why does that matter to the user?"
5. **Mark unknowns.** Anything unanswerable becomes `TBD — <owner> by <date>`.

## Elicitation Sequence

Work through these in order. Each answer informs the next question.

**1. The Problem**
> "What problem are you solving, and for whom? Describe the user and the situation they're in when the problem bites them."

**2. The Stakes**
> "Why does this problem matter — to the user, and to the business? What's the cost of leaving it unsolved?"

**3. The Proposed Solution (high-level)**
> "What kind of solution are you envisioning? Don't design it yet — just describe what changes for the user when it exists."

**4. The Differentiator**
> "What makes this different from what's already out there — whether that's a competitor, a workaround, or doing nothing?"

**5. The Target User**
> "Who specifically is the primary user? Be narrow — 'everyone' is not a user. What are their key characteristics, context, or behaviors?"

**6. The Scope**
> "What is explicitly in scope for this initiative? And what is explicitly out of scope?" (If they can't name out-of-scope items, push: "What are you tempted to include but won't?")

**7. Constraints**
> "What are the hard limits — budget, timeline, technology, regulatory, or team?"

**8. Assumptions and Risks**
> "What are you assuming to be true that hasn't been verified? What's the biggest risk to this initiative?"

## Quality Criteria

- [ ] Every section filled by the user, not inferred by the model.
- [ ] At least one explicit non-goal named.
- [ ] At least one constraint named.
- [ ] At least one assumption and one risk named.
- [ ] No section says "to be decided" without naming who decides and by when.

## Output Template

```markdown
# D01 — Vision Statement

## Problem
[The problem, for whom, in what context]

## Stakes
[Why it matters to user and business; cost of inaction]

## Solution Vision
[What changes for the user when this exists — not a design, a direction]

## Differentiator
[What makes this distinct from alternatives, workarounds, or doing nothing]

## Target User
[Narrow description of primary user — characteristics, context, behaviors]

## Scope

### In scope
- [item]

### Out of scope (explicit non-goals)
- [item]

## Constraints
- [Budget / timeline / tech / regulatory / team limits]

## Assumptions
- [Thing we're treating as true but haven't verified]

## Risks
- [Biggest threats to success]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D01-vision-statement.md` in the product-design directory.
