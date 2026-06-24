# D02 — Success Criteria

Produce the success criteria (`D02-success-criteria.md`) through structured elicitation with the human stakeholder.

> **Read first:** `context/measurement-frameworks.md` — SMART, OKRs, North Star, HEART, the per-metric schema, leading vs. lagging indicators.

**Builds on:** D01 vision statement. Read it first — every success criterion must trace to a D01 vision element.

## Operating Principles

1. **Outcomes over outputs.** "Login page deployed" is an output. "Support tickets for password issues drop 40%" is an outcome. Push until metrics describe what changes for users or the business.
2. **Falsifiable beats aspirational.** Every criterion must be something that could, in principle, fail. "Users will love it" is unfalsifiable. "≥40% weekly active users return within 7 days" is falsifiable.
3. **Balance leading and lagging.** At least one metric must be visible within weeks (leading indicator); at least one must capture the real business outcome over a longer horizon (lagging).
4. **Name the measurement method.** A metric without a measurement method is a wish. Confirm: how will we actually know this number?

## Elicitation Sequence

**1. Anchor to vision elements**
> "D01 describes [summarize the core outcome from D01]. What would have to be true, measured by data, for you to say this initiative succeeded?"

**2. Per metric: definition and measurement**
> "You said [metric X]. How would you measure that — what's the data source, and who owns pulling that number?"

**3. Baseline**
> "What's the baseline today? If we don't know, who will find out and by when?"

**4. Target and time horizon**
> "What's the target, and over what time window? (e.g., 'reach X within 6 months of launch')"

**5. Leading indicator check**
> "Is there a signal we'd see within the first 2–4 weeks that would tell us we're on track — even before the real outcome is measurable?"

**6. Non-success / failure definition**
> "If this initiative were to fail, what would the numbers look like? What's the floor below which we'd call it a loss?"

**7. Priority**
> "If you could only hit one of these, which one matters most? Rank them."

## Quality Criteria

- [ ] Every metric came from the user, not the model.
- [ ] Each metric has: definition, measurement method, baseline, target, time horizon, owner, data source.
- [ ] At least one leading indicator (visible within weeks).
- [ ] At least one lagging outcome (the real business win).
- [ ] A reasonable skeptic could prove each criterion failed.
- [ ] Metrics are ranked by priority.

## Output Template

```markdown
# D02 — Success Criteria

*Traces to: D01-vision-statement.md*

## Metrics

### [Metric Name] — Priority [1/2/3…]
- **Definition:** [What exactly this measures]
- **Type:** Leading / Lagging
- **Measurement method:** [How we get this number]
- **Data source:** [System, tool, or process]
- **Owner:** [Who is accountable for the measurement]
- **Baseline:** [Current value, or TBD — <owner> by <date>]
- **Target:** [Value] by [date/time-post-launch]
- **Failure floor:** [Value below which we call it a miss]
- **Traces to D01:** [Which vision element this evaluates]

### [Metric 2]
[Same schema]

## Non-Goals (What We Are Not Measuring)
- [Metric we explicitly decided not to track, and why]

## Review Cadence
- **Weekly during launch:** [who reviews what]
- **30/60/90-day checkpoint:** [who, what threshold triggers a pivot]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D02-success-criteria.md` in the product-design directory.
