# D10 — Touchpoints

Produce the touchpoint catalog (`D10-touchpoints.md`) — a catalog of every interaction point between the user and the product/service, derived from D09 journey stages.

> **Read first:** `context/design-patterns.md` — interaction patterns referenced when characterizing touchpoints.

**Builds on:** D09 journey maps. The touchpoint catalog is the input to friction analysis (D11) and to feature-level design.

## What This Is

D09 maps journeys at the stage level. D10 zooms in on each touchpoint — the specific moments of interaction — and characterizes them: who initiates, through what channel, with what expected outcome, and what's currently working or broken.

## Extraction Steps

### 1. Enumerate touchpoints from D09

For each journey stage in D09, extract each touchpoint listed under "Touchpoints." Give each a stable ID: `T-[stage initials]-[sequence]` (e.g., `T-ON-01` for first onboarding touchpoint).

### 2. Characterize each touchpoint

For each:
- **Channel:** Web app / Mobile / Email / API / Human (support, sales) / Physical
- **Initiator:** User-initiated / System-initiated / Mutual
- **Purpose:** What the user is trying to do at this moment
- **Current state:** Does this touchpoint exist today in a competitor or workaround? How does it work?
- **Quality signal:** If it exists, is it: Working well / Adequate / Broken / Missing entirely

### 3. Identify critical touchpoints

A touchpoint is **critical** if:
- It's the moment of highest emotional intensity in the journey (the valley or peak from D09)
- It's where users currently drop off or seek workarounds
- It's the gate that must succeed for the user to advance to the next stage

Mark critical touchpoints explicitly — these get priority attention in D11 and in feature design.

### 4. Map ownership

For each touchpoint, note: which team or system owns delivering it? (Frontend, backend, support, third-party, etc.) This becomes relevant during feature scoping.

## Quality Criteria

- [ ] Every touchpoint traces to a D09 stage.
- [ ] All touchpoints have stable IDs (T-XX-NN format).
- [ ] Critical touchpoints are marked and have a reason stated.
- [ ] No touchpoint is described as a feature — touchpoints describe the interaction, not the implementation.

## Output Template

```markdown
# D10 — Touchpoints

*Derived from: D09-journey-maps.md*

## Touchpoint Catalog

### Stage: [Stage name from D09]

#### T-[XX]-01 — [Touchpoint name]
- **Channel:** [Web / Mobile / Email / API / Human / Physical]
- **Initiator:** User / System / Mutual
- **Purpose:** [What the user needs to accomplish here]
- **Current state:** [How this works today, or "does not exist"]
- **Quality:** Working well / Adequate / Broken / Missing
- **Critical:** Yes / No — [reason if yes]
- **Owner:** [Team or system responsible]
- **Personas:** [P1 / P2 — which personas encounter this touchpoint]

#### T-[XX]-02 — [Touchpoint name]
[Same structure]

---

### Stage: [Next stage]
[Continue]

---

## Critical Touchpoints Summary

| ID | Name | Stage | Why critical | Current quality |
|---|---|---|---|---|
| T-XX-NN | [Name] | [Stage] | [Reason] | [Working/Broken/Missing] |

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D10-touchpoints.md` in the product-design directory.
