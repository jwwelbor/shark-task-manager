# D08 — User Personas

Produce the user personas (`D08-user-personas.md`) — the most cross-cutting product-design artifact. Personas defined here are referenced verbatim in D09–D14 and throughout downstream specification and design work.

> **Read first:** `context/design-patterns.md` — referenced across the UX/CX synthesis work.

**Builds on:** D07 user needs and D06 user insights.

## What Makes a Good Persona

A persona is a behaviorally-grounded archetype, not a demographic average. It answers: "What kind of person is this, how do they think about the problem, and what do they need from us?"

**Anti-patterns to avoid:**
- Demographics without behavior ("35-year-old marketing manager") — demographics don't predict behavior
- Aspirational characters ("tech-savvy early adopter") — name what they actually do, not what we hope they are
- Too many personas (>4 for most products) — if you have 8 personas, most are noise
- Invented details — every attribute must trace to D06 clusters or D05 research

## Derivation Steps

### 1. Map D06 clusters to proto-personas

Each behavioral cluster from D06 is a candidate persona. Check: do any clusters represent the same underlying user type that just encounters the problem in slightly different contexts? Merge those.

Aim for 2–4 personas. One primary persona (the one the design optimizes for) and 1–3 secondary.

### 2. Build each persona from evidence

For each persona:
- Name the D06 cluster(s) it draws from
- Name the D07 need statements that are primary for this persona
- Describe their context, behavior, and emotional state — from research, not imagination
- Write the "day-in-the-life" scenario only around the problem domain — not their whole life

### 3. Designate primary vs. secondary

The **primary persona** is the one whose needs take precedence when design trade-offs arise. There is exactly one. Name it explicitly.

Secondary personas matter but don't drive decisions when there's a conflict with the primary.

### 4. Confirm with human stakeholder

Present the persona list and the primary designation. Ask:
> "Does this persona set feel right? Is there a type of user we've missed, or one here that doesn't really exist in practice?"

Use `AskUserQuestion`. Adjust based on feedback — the persona set is a human-validated input.

## Record material Questions

When a persona-priority or trade-off decision remains materially unresolved,
use `skills/question-management/SKILL.md` to create or reuse a linked Q###.
Record a non-material rationale in the persona artifact instead. Do not treat
the absence of `TBD` text as decision closure.

## Quality Criteria

- [ ] 2–4 personas only — no inflation.
- [ ] Exactly one primary persona designated.
- [ ] Every persona attribute traces to D06 cluster or D05 participant quote.
- [ ] Each persona maps to at least one D07 need statement by ID.
- [ ] No demographic-only labels without accompanying behavioral description.
- [ ] Human stakeholder confirmed the set.

## Output Template

```markdown
# D08 — User Personas

*Derived from: D06-user-insights.md, D07-user-needs.md*

## Persona Summary

| ID | Name | Type | Primary needs |
|---|---|---|---|
| P1 | [Name] | **Primary** | N01, N02 |
| P2 | [Name] | Secondary | N03 |
| P3 | [Name] | Secondary | N01, N04 |

---

## P1 — [Name] * Primary

**Behavioral archetype:** [One sentence: what defines how this person approaches the problem]

**Context:**
- [Where/when/why they encounter the problem]
- [Tools, processes, or environment they're working in]
- [Constraints on their time, authority, or information]

**Goals in this domain:**
- [What they're trying to accomplish — from D07 need statements]

**Frustrations (current state):**
- [From D06 / D05 evidence — cite source]

**What success looks like for them:**
- [What "done" feels like from their perspective]

**Decision-making style:** [How they evaluate and choose — cautious/fast, social/solo, data-driven/intuitive]

**Representative quote:** "[A quote from a D05 participant that captures this persona — or a synthesized quote clearly labeled as synthesized]"

**Traces to:** D06 [cluster name], D05 [participant IDs]
**Primary needs:** [D07 need IDs]

---

## P2 — [Name]

[Same structure, without * Primary]

---

## Stakeholder Confirmation

Confirmed by: [name] on [date]
Amendments from review: [any changes made]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
*Note: Persona IDs (P1, P2…) are stable references used verbatim in D09–D14 and all downstream specs.*
```

Save as `D08-user-personas.md` in the product-design directory.
