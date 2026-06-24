# D06 — User Insights

Produce the user insights (`D06-user-insights.md`) by synthesizing all discovery research into a consolidated, cross-source picture of what users experience, think, and feel.

> **Read first:** `context/design-patterns.md` — referenced across the UX/CX synthesis work.

**Builds on:** D05 stakeholder insights (primary input). Also read D03 (market context) and D01 (target user). D06 is a synthesis artifact — it introduces no new primary research; it draws together what D03 and D05 revealed and sharpens it into findings the design phase can act on.

## What This Is (and Isn't)

D05 is raw stakeholder interview output — themes, quotes, surprise flags. D06 is the interpretation layer: "given everything we learned, what do we now know about how users actually experience this problem?" It is a step up in abstraction toward needs (D07) and personas (D08).

Do not repeat D05 verbatim. Synthesize. The reader of D06 should come away with a clear mental model of the user's world, not a summary of what we did.

## Synthesis Steps

### 1. Read all upstream sources

Read D01 (target user), D03 (market context and competitor behaviors), D05 (stakeholder themes). Note any tensions between what the market shows and what users said.

### 2. Cluster into behavioral segments

Group users by how they behave around the problem — not by demographics. Ask: "Are there meaningfully different ways people approach this?" If yes, these become the seeds of personas in D08.

For each cluster, describe:
- The triggering situation (when the problem surfaces for them)
- The current behavior (what they actually do)
- The emotional state (what they feel while doing it)
- The outcome they're trying to reach

### 3. Surface cross-cutting patterns

Across all clusters, identify:
- **Universal frustrations:** Pain points that appear regardless of segment
- **Segment-specific frustrations:** Pain points that only affect one cluster
- **Workaround behaviors:** Things users do that signal a missing capability
- **Moments of delight:** Things in current tools/processes that users actually like and would miss

### 4. Identify the tension

Most good product problems have a central tension — two things users want that currently work against each other (e.g., "they want speed but also trust"; "they want flexibility but not complexity"). Name it explicitly if it exists.

### 5. Flag open questions

What do we still not know about users that would meaningfully affect design decisions? These become research debts to address before or during validation.

## Quality Criteria

- [ ] Every insight traces to D05 evidence or D03 market observation — no new claims invented here.
- [ ] At least two behavioral clusters identified (if only one, justify why the user population is truly homogenous).
- [ ] The central tension named (or explicitly stated as absent with reasoning).
- [ ] Open questions listed with enough specificity that someone could design research to answer them.

## Output Template

```markdown
# D06 — User Insights

*Synthesized from: D05-stakeholder-insights.md, D03-market-research.md, D01-vision-statement.md*

## Behavioral Clusters

### Cluster A: [Descriptive name, not a demographic label]
- **Triggering situation:** [When this cluster encounters the problem]
- **Current behavior:** [What they actually do]
- **Emotional state:** [How they feel about it]
- **Desired outcome:** [What they're trying to reach]
- **Evidence:** [D05 theme IDs or participant quotes]

### Cluster B: [Name]
[Same structure]

## Cross-Cutting Patterns

### Universal frustrations (all clusters)
- [Frustration] — evidence: [source]

### Segment-specific frustrations
- Cluster A: [Frustration]
- Cluster B: [Frustration]

### Workaround behaviors
- [Behavior] — signals: [what capability is missing]

### Moments of delight (things to preserve)
- [What users currently like] — evidence: [source]

## Central Tension

[The core trade-off users face. E.g.: "Users want X but also Y, and current solutions force them to choose."]

Or: No significant central tension identified — [brief explanation].

## Open Questions

1. [Specific unknown] — impact on design: [high/medium/low]
2. [Specific unknown] — impact on design: [high/medium/low]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D06-user-insights.md` in the product-design directory.
