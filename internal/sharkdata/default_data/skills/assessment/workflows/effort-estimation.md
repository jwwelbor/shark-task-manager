---
name: assessment-effort-estimation
mode: effort_estimation
---

# Workflow: Effort Estimation

**Purpose**: Estimate implementation effort for planning and capacity allocation.

## Process

### Step 1: Apply Heuristic Formula

```text
Effort (days) ~= (file_impact_score x 2) + (task_count x 1.5) + (pattern_novelty_score x 3)

If regression_risk_score == 2 (behavior-preserving): multiply by 1.5
```

Worked example:

- File Impact: 3 -> 6 days
- Task Count: 12 -> 18 days
- Pattern Novelty: 0 -> 0 days
- Base: 24 days
- Regression Risk: 2 -> multiply by 1.5
- Total: 36 days

### Step 2: Validate Against Tier Thresholds

Compare estimate to tier expectations:

- SIMPLE: <1 week
- STANDARD: 1-2 weeks
- COMPLEX: 3+ weeks

If the estimate does not match the assigned tier, flag it and reconsider both.

### Step 3: Apply Context Adjustments

Multiply the base estimate by:

- 1.2-1.5x for teams learning new technology
- 0.8-0.9x for teams with deep domain expertise
- 1.3-1.8x for refactoring work
- 2.0+x for greenfield architecture

Document the multiplier and the reason.

### Step 4: State Confidence Level

- **HIGH**: similar features exist, team has done this kind of work, low novelty
- **MEDIUM**: some unknowns, mix of familiar and novel patterns
- **LOW**: greenfield, behavior-preserving refactor, novel architecture, or all three

### Step 5: Document Estimate

```markdown
# Effort Estimate

**Complexity Score**: {score}/27 (tier: {tier})

## Estimation Inputs
- File Impact: {score}
- Task Count: {count}
- Pattern Novelty: {score}
- Regression Risk: {score}

## Calculation
Base Effort: {days} days
Adjustment: {factor}x ({reason})
**Total Effort**: {days} days ({weeks} weeks)

## Confidence Level
{HIGH|MEDIUM|LOW} — {rationale}

## Assumptions
- {assumption 1}

## Risks to Estimate
- {risk 1}
```

### Step 6: Return Structured Output

Return: `base_effort_days`, `adjusted_effort_days`, `confidence_level`, `assumptions`, `risks`, `estimate_report`.
