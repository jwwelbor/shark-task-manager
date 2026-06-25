# D03 — Market Research

Produce the market research (`D03-market-research.md`) by researching the competitive landscape, market size, and positioning context.

> **Read first:** `context/research-techniques.md` — competitive analysis frameworks, market sizing methods.

**Builds on:** D01 vision statement — read it to scope research against the target user and problem.

## Research Steps

### 1. Scope the Research

Read D01. Extract:
- Target user (§ Target User)
- The problem being solved (§ Problem)
- Any explicit competitors or alternatives mentioned (§ Differentiator)

These are the anchors. Don't research the broader market — research the market for this specific user and problem.

### 2. Competitive Landscape

Use `WebSearch` to identify direct and indirect competitors:
- **Direct:** Products solving the same problem for the same user
- **Indirect:** Workarounds, adjacent tools, or "do nothing" alternatives

For each competitor, document:
- Name and URL
- Target user and core value proposition
- Key strengths (what they do well)
- Key weaknesses or gaps (what they leave unaddressed)
- Pricing model (if public)
- Market position (leader / challenger / niche)

Aim for 3–6 direct competitors and 2–3 indirect. Don't pad with irrelevant players.

### 3. Market Size and Trends

Research:
- Estimated market size (TAM/SAM/SOM if findable — label the source)
- Key growth trends in the space
- Recent funding, acquisitions, or exits (signals of market activity)
- Regulatory or technology shifts that affect the space

Cite sources. If reliable data isn't findable, say so rather than guessing.

### 4. Positioning Gap

Based on the landscape, identify:
- What gap or underserved need exists that D01's proposed solution targets
- How competitors are positioned relative to each other (axes: e.g., price vs. power, breadth vs. depth, self-serve vs. enterprise)
- Where our initiative could differentiate

### 5. Key Risks from the Market

- Competitors likely to copy or respond
- Market timing risks (too early, too crowded, commoditizing)
- Distribution or channel challenges

## Quality Criteria

- [ ] All competitors are sourced (links included), not invented from training data.
- [ ] Every market size claim has a source (name + URL or "primary research").
- [ ] Positioning gap traces directly to D01's target user and problem.
- [ ] Risks are specific, not generic ("market is competitive" is not a risk).

## Output Template

```markdown
# D03 — Market Research

*Scoped from: D01-vision-statement.md*

## Competitive Landscape

### Direct Competitors

| Competitor | Target User | Core Value Prop | Strengths | Gaps | Pricing | Position |
|---|---|---|---|---|---|---|
| [Name]([URL]) | [who] | [what] | [+] | [−] | [model] | [leader/niche/etc] |

### Indirect Competitors / Alternatives

| Alternative | Why users choose it | Gap it leaves |
|---|---|---|
| [Name] | [reason] | [gap] |

## Market Size and Trends

- **TAM:** [figure] — Source: [name, URL, date]
- **Key trends:** [bullet]
- **Recent signals:** [funding, M&A, exits — with dates]
- **Regulatory/tech shifts:** [if applicable]

## Positioning Gap

[Describe the gap our initiative targets. What map of the space shows no one owning the position we're aiming for?]

## Market Risks

1. **[Risk]:** [Specific concern and why it's real]
2. **[Risk]:** [Same]

## Research Limitations

- [Things we couldn't verify — what would we need to do primary research to confirm?]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D03-market-research.md` in the product-design directory.
