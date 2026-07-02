---
name: assessment-complexity-triage
mode: complexity_triage
---

# Workflow: Complexity Triage

**Purpose**: Assign complexity tier (SIMPLE/STANDARD/COMPLEX) to inform workflow routing decisions.

**When to use**: Whenever a feature's complexity needs to be assessed at intake, after scope changes, or when deciding whether autonomous build is feasible.

## Process

### Step 1: Gather Context

From inputs, you should have: `feature_title`, `feature_description`, `epic_context`, optionally `codebase_summary` and `existing_task_count`.

If the codebase summary is missing and you need it for accurate scoring, request it from the host. The host is responsible for invoking research upstream when needed.

You should leave this step with a clear understanding of what the feature is, why it exists, and where it fits in the codebase.

### Step 2: Score Across 9 Dimensions

Score each dimension 0-3 points. See `../references/complexity-dimensions.md` for the full rubric and worked examples.

**Technical complexity (6 dimensions)**:

1. File Impact
2. Pattern Novelty
3. Data Model
4. API Surface
5. Cross-Feature Dependencies
6. UI Complexity

**Execution complexity (3 dimensions)**:

7. Task Estimation
8. Regression Risk
9. Execution Effort

### Step 3: Calculate Total Score

```text
Total Score = Sum of all 9 dimensions (max 27 points)
```

### Step 4: Assign Tier

Based on total score:

- **0-6 points**: SIMPLE tier
- **7-15 points**: STANDARD tier
- **16+ points**: COMPLEX tier

See `../references/tier-thresholds.md` for tier characterization and customization guidance.

### Step 5: Assess Autonomous Build Feasibility

Even STANDARD features may not be suitable for autonomous build. Flag `autonomous_build_feasible = false` when any of these hold:

- Task count > 10
- Regression risk >= 2
- Execution effort >= 2
- Circular dependencies present
- Novel patterns combined with behavior-preserving requirement

### Step 6: Generate Triage Report

Produce structured markdown with feature key/title, total score, tier, per-dimension scores with rationale, technical and execution subtotals, tier rationale, autonomous-build feasibility verdict, and recommendations.

Format:

```markdown
# Complexity Triage Report — {feature_title}

**Score**: {total}/27
**Tier**: {SIMPLE|STANDARD|COMPLEX}

## Dimension Scores

### Technical Complexity (6 dimensions)
1. File Impact: {score}/3 — {rationale}
2. Pattern Novelty: {score}/3 — {rationale}
3. Data Model: {score}/3 — {rationale}
4. API Surface: {score}/3 — {rationale}
5. Cross-Feature Dependencies: {score}/3 — {rationale}
6. UI Complexity: {score}/3 — {rationale}

### Execution Complexity (3 dimensions)
7. Task Estimation: {score}/3 — {count} tasks
8. Regression Risk: {score}/3 — {additive|behavior-preserving|breaking}
9. Execution Effort: {score}/3 — {time estimate}

**Technical Total**: {sum}/18
**Execution Total**: {sum}/9
**Overall Total**: {sum}/27

## Tier Assignment

**Assigned Tier**: {tier}
**Rationale**: {1-2 sentence explanation}

## Autonomous Build Feasibility

- Task count: {count} (threshold <=10)
- Regression risk: {score} (threshold <=1)
- Execution effort: {score} (threshold <=1)
- Circular dependencies: {yes|no}

**Recommendation**: {feasible | manual execution recommended | consider splitting}
```

### Step 7: Return Structured Output

Return: `complexity_score`, `tier`, `dimension_scores`, `triage_report`, `autonomous_build_feasible`, `tier_rationale`.
