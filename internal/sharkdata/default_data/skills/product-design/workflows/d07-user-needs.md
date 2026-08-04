# D07 — User Needs

Produce the user needs (`D07-user-needs.md`) by translating D06 user insights into structured, solution-agnostic need statements that the design phase can address.

> **Read first:** `context/design-patterns.md` — referenced across the UX/CX synthesis work.

**Builds on:** D06 user insights.

## What This Is

D07 bridges insights (what users experience) and personas (who users are). A **need statement** describes what a user is trying to accomplish and why, without specifying a solution. It is the job-to-be-done, not a feature request.

Format: **"When [situation], the user needs to [action/outcome] so that [reason/goal]."**

Good: "When reviewing a lengthy contract, the user needs to identify the riskiest clauses quickly so they can focus their attention where it matters most."

Bad: "The user needs a risk-highlighting feature." (This is a solution, not a need.)

## Derivation Steps

### 1. Read D06 fully

Extract the behavioral clusters, workaround behaviors, universal frustrations, and central tension.

### 2. Derive one need statement per behavioral pattern

For each cluster from D06, derive the primary need that drives their behavior. Then check: are there secondary needs within the same cluster that are meaningfully distinct?

Aim for 3–8 total need statements. More than 8 suggests you're describing features, not needs.

### 3. Score each need

For each need statement:
- **Frequency:** How many users have this need? (All / Most / Some / Few)
- **Intensity:** How much does it matter when they have it? (Critical / High / Medium / Low)
- **Currently served?** Is there an existing way to meet this need, even imperfectly? (Yes / Partially / No)
- **Initiative relevance:** Does D01's proposed solution directly address this? (Core / Adjacent / Out of scope)

### 4. Sequence by priority

Order needs with Core + Critical + Poorly-served first. These are the non-negotiables for D08 personas and D09 journey maps.

### 5. Confirm with the user

Before finalizing, present the top 3 needs to the human stakeholder:
> "Based on all the research, these are the three most critical user needs. Do any of these surprise you, or feel like we've missed something important?"

Use `AskUserQuestion`. Do not finalize without this confirmation — needs are a human-validated input, not a model inference.

## Record material Questions

When a prioritized user-need decision remains materially unresolved, use
`skills/question-management/SKILL.md` to create or reuse a linked Q###. Record
a non-material rationale in the needs artifact instead. Do not treat the
absence of `TBD` text as decision closure.

## Quality Criteria

- [ ] Every need statement is solution-agnostic (no product features named).
- [ ] Each need traces to at least one D06 insight or cluster.
- [ ] Needs are scored and sequenced by priority.
- [ ] Human stakeholder has confirmed the top-3 list.
- [ ] No need statement is just a rephrasing of a D05 theme — each must add the "so that" purpose.

## Output Template

```markdown
# D07 — User Needs

*Derived from: D06-user-insights.md*

## Need Statements (Priority Order)

### N01 — [Short name]
**Statement:** When [situation], the user needs to [action/outcome] so that [reason].

| Dimension | Value |
|---|---|
| Frequency | All / Most / Some / Few |
| Intensity | Critical / High / Medium / Low |
| Currently served | Yes / Partially / No |
| Initiative relevance | Core / Adjacent / Out of scope |

**Traces to D06:** [Cluster or pattern name]

---

### N02 — [Short name]
[Same structure]

---

## Stakeholder Confirmation

Confirmed by: [name] on [date]
Top-3 needs reviewed: N01, N02, N03
Notes: [Any amendments or pushback from the confirmation conversation]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D07-user-needs.md` in the product-design directory.
