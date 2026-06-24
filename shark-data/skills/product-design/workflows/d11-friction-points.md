# D11 — Friction Points

Produce the friction-point inventory (`D11-friction-points.md`) — a prioritized inventory of where and why the experience breaks down for users, derived from D10 touchpoints and D09 journey maps.

This is the final CX design artifact. Its outputs are what user testing (D12) exercises and what validated designs (D14) resolve.

> **Read first:** `context/design-patterns.md` — patterns referenced when describing friction.

**Builds on:** D10 touchpoints. Also read D09 journey maps — the emotional valley moments are starting candidates for friction.

## What a Friction Point Is

A **friction point** is a specific, observable place in the journey where the user experiences resistance — they slow down, make an error, seek help, use a workaround, or abandon the flow. It is not a feature gap ("there's no search") — it is an experienced problem ("users can't find what they need and resort to ctrl+F on long pages").

Friction has three components:
1. **Where** (touchpoint ID from D10)
2. **What happens** (the observable behavior or symptom)
3. **Why it matters** (persona impact and business consequence)

## Identification Steps

### 1. Start from D10 critical touchpoints and D09 valley moments

These are the highest-probability locations for friction. List them first.

### 2. Walk every non-critical touchpoint

For each D10 touchpoint with quality "Adequate" or "Broken" or "Missing," ask: what specifically causes friction here?

### 3. For each friction point, assess

- **Type:**
  - *Confusion* — the user doesn't know what to do or what happened
  - *Effort* — the user knows what to do but it takes too long or too many steps
  - *Error* — the user makes a mistake they can't easily recover from
  - *Trust gap* — the user is uncertain whether to proceed (security, privacy, accuracy concerns)
  - *Capability gap* — the user can't accomplish the task at all with current tools

- **Severity:** Critical (blocks progress) / High (causes abandonment or workaround) / Medium (causes frustration) / Low (minor annoyance)

- **Frequency:** Always / Often / Sometimes / Rarely

- **Persona impact:** Which personas experience this? (P1, P2…)

### 4. Score and prioritize

Priority score = Severity weight × Frequency weight × Persona reach

Use: Critical=4, High=3, Medium=2, Low=1 for severity; Always=4, Often=3, Sometimes=2, Rarely=1 for frequency.

The top 5 friction points (highest scores) are the non-negotiable targets for validation.

## Quality Criteria

- [ ] Every friction point traces to a D10 touchpoint ID.
- [ ] Type, severity, frequency, and persona impact are set for each.
- [ ] Top 5 are explicitly identified and explain why they score highest.
- [ ] No friction point describes a feature solution — only the problem.

## Output Template

```markdown
# D11 — Friction Points

*Derived from: D10-touchpoints.md, D09-journey-maps.md*

## Friction Point Inventory (Priority Order)

### FP-01 — [Short name] (Top 5)
- **Touchpoint:** [D10 touchpoint ID]
- **Type:** Confusion / Effort / Error / Trust gap / Capability gap
- **Description:** [Observable behavior or symptom — what the user does or experiences]
- **Severity:** Critical / High / Medium / Low
- **Frequency:** Always / Often / Sometimes / Rarely
- **Personas affected:** P1, P2…
- **Priority score:** [Severity weight × Frequency weight × persona count]
- **Business consequence:** [What this costs — abandonment, support load, conversion drop, etc.]

### FP-02 — [Short name] (Top 5)
[Same structure]

### FP-03 — [Short name]
[Same structure — continue for all friction points, ranked]

---

## Top 5 Summary

| ID | Name | Type | Score | Primary persona |
|---|---|---|---|---|
| FP-01 | [Name] | [Type] | [Score] | P1 |
| FP-02 | [Name] | [Type] | [Score] | P1, P2 |

These five are the test targets for D12 (user testing) and the design verdicts in D14.

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D11-friction-points.md` in the product-design directory.
