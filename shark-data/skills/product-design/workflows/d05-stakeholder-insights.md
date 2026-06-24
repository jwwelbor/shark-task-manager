# D05 — Stakeholder Insights

Produce the stakeholder insights (`D05-stakeholder-insights.md`) by interviewing or synthesizing input from users and stakeholders.

> **Read first:** `context/research-techniques.md` — interview templates, synthesis methods.

**Builds on:** D01 vision statement (target user and problem anchor the interview scope). Also read D03 (market context) and D04 (feasibility constraints) if available.

## Approach — Choose by Evidence Available

Two methodology modes; pick based on what evidence the user actually has:

**Mode A: Live interview synthesis** — The user will conduct or has already conducted stakeholder/user interviews. Structure the questions in advance, then capture and synthesize the outputs.

**Mode B: Proxy synthesis** — No live interviews are available. Use existing inputs: support tickets, sales call notes, NPS comments, user surveys, prior research. Synthesize from these, noting what's inferred vs. directly stated.

Ask the user which evidence they have before choosing the mode.

## Mode A — Interview Guide

Provide these questions for the user to ask each participant. They are open-ended — avoid yes/no framing.

**About the current situation:**
- "Walk me through the last time you had to do [the problem D01 describes]. What happened?"
- "What tools or workarounds do you use today for this? What do you like or dislike about them?"
- "What's the most frustrating part of the current experience?"

**About priorities:**
- "If this problem were magically solved tomorrow, what would be different for you?"
- "What would make you trust a new solution enough to switch to it?"
- "What would make you NOT switch — what's the deal-breaker?"

**About context:**
- "Who else is involved when you deal with this problem — who else would be affected by a change?"
- "How often does this problem come up? When does it tend to happen?"

After interviews, ask the user to share their notes and synthesize using the template below.

## Mode B — Existing Research Synthesis

For each input source, capture:
- Source type (support tickets, sales notes, NPS, etc.)
- Volume / time range
- Top recurring themes (direct quotes preferred)
- What's notable about what's absent (things you'd expect to see but don't)

## Synthesis Steps

Whether Mode A or B:

1. **Cluster observations** into themes — group similar pain points, needs, or behaviors together.
2. **Score each theme** by frequency (how often it appears) and intensity (how strongly participants feel about it).
3. **Tag each theme** as: Core need / Nice-to-have / Table-stakes / Deal-breaker.
4. **Surface surprises** — things that weren't anticipated from D01 or D03.
5. **Identify gaps** — things D01 assumed about users that were not confirmed.

## Quality Criteria

- [ ] All observations are attributed to a source (participant ID, document, or data source) — not invented.
- [ ] Themes are ranked by frequency × intensity.
- [ ] At least one surprise or unexpected finding is captured (if none, explain why).
- [ ] Any D01 assumption that was contradicted is called out explicitly.

## Output Template

```markdown
# D05 — Stakeholder Insights

*Synthesis mode: A (live interviews) / B (existing research)*
*Sources: [list participants P1–Pn, or documents]*

## Research Summary

- **Participants / sources:** [count and brief description]
- **Date range:** [when research was conducted]

## Key Themes

### Theme 1: [Name] — [Core need / Nice-to-have / Table-stakes / Deal-breaker]
- **Frequency:** [n of N sources]
- **Intensity:** High / Medium / Low
- **Evidence:** "[Direct quote or paraphrase]" — P1; "[Quote]" — P3
- **Implication for the initiative:** [What this means for design]

### Theme 2: [Name]
[Same structure]

## Surprises

- [Something unexpected that wasn't in D01 or D03]

## D01 Assumption Checks

| D01 Assumption | Confirmed / Contradicted / Unresolved | Evidence |
|---|---|---|
| [Assumption from D01] | [Status] | [What we heard] |

## Gaps

- [Things we wanted to learn but couldn't — who could fill this gap?]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D05-stakeholder-insights.md` in the product-design directory.
