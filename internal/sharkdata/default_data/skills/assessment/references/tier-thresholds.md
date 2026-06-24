---
inputs:
  # This reference is consulted by the assessment skill during complexity_triage mode
  # (to characterize a tier) and during effort_estimation (to validate estimates against
  # tier expectations). It also documents customization guidance for projects that need
  # non-default thresholds.
  #
  # No runtime inputs — this reference is purely declarative.
outputs:
  # No outputs — informs scoring and routing decisions made by the caller.
---

# Complexity Tier Thresholds

Default tier definitions and project customization guidance for complexity-based workflow routing.

## Default Tier Assignments

### Tier Classification System

Based on total complexity score (sum of 9 dimensions, max 27 points):

| Tier | Score Range | Description | Typical Routing |
|------|-------------|-------------|-----------------|
| **SIMPLE** | 0-6 | Quick wins, minimal complexity | Skip BA/arch → straight to task generation |
| **STANDARD** | 7-15 | Normal features, moderate effort | Full workflow: BA → arch → tasks → autonomous build |
| **COMPLEX** | 16+ | Large features, high effort/risk | Full workflow → manual execution (no autonomous build) |

### Score Distribution

How scores typically distribute across the dimensions:

```
0-6 points (SIMPLE):
  Technical: 0-3 points    (existing patterns, no schema, internal)
  Execution: 0-3 points    (1-3 tasks, additive, <1 week)
  Example: Add helper function, update config

7-15 points (STANDARD):
  Technical: 3-8 points    (some new patterns, minor schema, new endpoints)
  Execution: 2-7 points    (4-7 tasks, some refactoring, 1-2 weeks)
  Example: New CRUD feature, API endpoint with tests

16+ points (COMPLEX):
  Technical: 8+ OR         (major new architecture)
  Execution: 8+ OR         (8+ tasks, high risk, 3+ weeks)
  Either dimension: 16+
  Example: Service layer refactoring, new authentication system
```

**Key insight**: a feature can be COMPLEX due to either technical novelty OR execution scope. Many borderline-COMPLEX features score most of their points in execution (high task count, behavior-preserving, multi-week effort).

---

## Tier Characterization

### SIMPLE Tier (0-6)

**Phases to skip**: business-analyst refinement, architecture design.
**Go to**: task generation (minimal spec).
**Autonomous build**: YES (if 1-2 tasks).

**Characteristics**:

- Single task or two quick tasks
- No new patterns introduced
- No schema changes
- No API changes
- Additive only (low risk)
- <1 week effort

**Example**: Add `--verbose` flag to a CLI command.

---

### STANDARD Tier (7-15)

**Phases to include**: BA refinement, architecture design.
**Go to**: task generation with PRD and arch docs.
**Autonomous build**: YES (if ≤7 tasks, additive or minor refactoring).

**Characteristics**:

- 4-7 tasks typical
- Uses existing patterns with adaptations
- Minor schema changes (add columns, indexes)
- New API endpoints
- Some dependencies (1-2 features)
- 1-2 weeks effort

**Example**: create new service with CRUD endpoints and tests.

---

### COMPLEX Tier (16+)

**Phases to include**: BA refinement, architecture design, extensive planning.
**Go to**: task generation with comprehensive specs.
**Autonomous build**: NO (manual execution with agent assistance).

**Characteristics**:

- 8+ tasks typical
- Major new patterns or architectures
- New tables/schemas or significant refactoring
- Complex integration (3+ dependencies)
- Behavior-preserving refactoring (high risk)
- 3+ weeks effort

**Example**: service-layer architecture refactoring.

**Why no autonomous build**:

- Context window limits (8+ tasks often exceeds AI context).
- Regression risk requires careful human oversight.
- Multi-week effort is too large for a single session.
- Needs strategic decision-making throughout.

**Alternative**: split into multiple STANDARD features when possible.

---

## Tier Boundary Analysis

### Score 6 vs 7 (SIMPLE ↔ STANDARD)

**Score 6** (highest SIMPLE):

- File Impact: 2 (4-10 files)
- Pattern Novelty: 0 (existing)
- Task Estimation: 2 (4-7 tasks)
- All others: 0
- **Routing**: skip BA/arch.

**Score 7** (lowest STANDARD):

- File Impact: 2 (4-10 files)
- Pattern Novelty: 1 (adapt existing)
- Task Estimation: 2 (4-7 tasks)
- All others: 0
- **Routing**: include BA/arch.

**Difference**: introduction of adapted patterns triggers full workflow for design review.

---

### Score 15 vs 16 (STANDARD ↔ COMPLEX)

**Score 15** (highest STANDARD):

- File Impact: 3 (10+ files)
- Task Estimation: 3 (8+ tasks)
- Regression Risk: 2 (behavior-preserving)
- Execution Effort: 2 (3-4 weeks)
- All others: 0
- **Routing**: full workflow → autonomous build (if feasible).

**Score 16** (lowest COMPLEX):

- Same as above + 1 more point anywhere.
- **Routing**: full workflow → manual execution.

**Difference**: a single additional point changes the autonomous build decision.

**Gray zone (14-16)**: consider:

- Task count: >10 tasks → likely too complex for autonomous.
- Regression risk: behavior-preserving → needs human oversight.
- Context: will it fit in AI context window?

---

## Customization Guidelines

### When to Adjust Thresholds

Projects may need custom thresholds based on:

1. **Team size**:
   - Solo developer: raise STANDARD threshold to 20 (handle more complexity).
   - Large team: lower COMPLEX threshold to 12 (split work earlier).

2. **Project maturity**:
   - Greenfield: SIMPLE=0-8, STANDARD=9-18, COMPLEX=19+.
   - Legacy refactoring: SIMPLE=0-4, STANDARD=5-12, COMPLEX=13+.

3. **Risk tolerance**:
   - High risk (production): SIMPLE=0-4, STANDARD=5-10, COMPLEX=11+.
   - Low risk (prototype): SIMPLE=0-10, STANDARD=11-20, COMPLEX=21+.

4. **Agent capability**:
   - Autonomous build reliable: keep defaults.
   - Autonomous build unreliable: lower STANDARD to 5-12, COMPLEX at 13+.

### Threshold Validation

After customizing thresholds, validate with historical features:

- [ ] Run complexity triage on 5-10 past features.
- [ ] Verify tier assignments match team intuition.
- [ ] Check that SIMPLE features could skip BA/arch.
- [ ] Check that STANDARD features fit autonomous build.
- [ ] Check that COMPLEX features needed manual execution.
- [ ] Adjust if >30% of features are misclassified.

### Common Misclassifications

**Symptom**: most features land in STANDARD (7-15).
**Cause**: threshold range too wide.
**Fix**: narrow STANDARD range (e.g. 8-12) or adjust scoring.

**Symptom**: no features are SIMPLE.
**Cause**: threshold too low or not creating small features.
**Fix**: raise SIMPLE threshold to 8 OR encourage smaller features.

**Symptom**: many features marked COMPLEX but completed quickly.
**Cause**: threshold too low or over-scoring.
**Fix**: raise COMPLEX threshold to 20 OR review scoring calibration.

---

## Tier Override Cases

### When to Override Tier Assignment

Sometimes the complexity score doesn't match reality. Override when:

1. **Strategic importance**: feature is simple but strategically critical → upgrade to STANDARD for full documentation.
2. **Learning opportunity**: feature is complex but team needs practice → downgrade to STANDARD with extra support.
3. **Risk mismatch**: score says STANDARD but involves production data → upgrade to COMPLEX for extra review.
4. **Prototype exception**: score says COMPLEX but it's throwaway code → downgrade to SIMPLE.

### Documenting Overrides

- Always record why the tier was overridden.
- Reference the score but explain the business context.
- Example rationale: "Score: 16/27 (COMPLEX), overriding to STANDARD for team learning."

---

## Autonomous Build Feasibility

### Decision Factors Beyond Complexity Score

Even STANDARD features may not be suitable for autonomous build.

**Disqualifiers** (recommend manual execution):

- Task count >10 (context window limits)
- Regression risk = 2 or 3 (needs human oversight)
- Execution effort >2 weeks (too long for single session)
- Circular dependencies (needs strategic thinking)
- Novel patterns + behavior-preserving (high-risk combination)

**Computation**:

```
Autonomous Build Feasible =
  (tier == STANDARD) AND
  (task_count <= 7) AND
  (regression_risk <= 1) AND
  (execution_effort <= 1) AND
  (no_circular_deps)
```

**Worked example**:

- Tier: STANDARD (score 15/27)
- Task count: 12 — fails
- Regression risk: 2 — fails
- Execution effort: 2 — fails
- **Result**: NOT feasible for autonomous build despite STANDARD tier.

**Recommendation**: stop the autonomous build attempt and route to manual execution.

---

## Tier Migration Patterns

### Feature Starts SIMPLE, Grows to STANDARD

**Example**: "Add user avatar."

- Initial triage: SIMPLE (2 tasks, 1 file).
- During implementation: discovered need for image processing, storage, API.
- Re-triage: STANDARD (6 tasks, multiple files, new endpoints).

**Action**: re-run complexity triage mid-development. Update the tier in metadata; the host workflow re-routes.

### Feature Starts COMPLEX, Splits to STANDARD

**Example**: large refactoring feature scoring 15/27.

- Initial triage: COMPLEX-borderline.
- Decision: split into 3 features.
- Each split: STANDARD (5/27, 6/27, 4/27).

**Action**: create new features and migrate tasks. Each split is autonomous-build-feasible on its own.

---

## Project-Specific Examples

### Example 1: CLI Task Manager Project

**Thresholds**: default (SIMPLE: 0-6, STANDARD: 7-15, COMPLEX: 16+).

**Typical distribution**:

- SIMPLE: 20% (helper functions, config updates, small fixes)
- STANDARD: 60% (new CLI commands, service methods, CRUD features)
- COMPLEX: 20% (architecture changes, service-layer refactoring)

**Customization**: none needed (defaults work well).

**Learnings**:

- Most CLI features land in STANDARD (7-12 points).
- Refactoring often pushes to COMPLEX (execution dimensions).
- Behavior-preserving work consistently scores 2+ on regression risk.

---

### Example 2: Microservices Platform

**Thresholds**: custom (SIMPLE: 0-4, STANDARD: 5-12, COMPLEX: 13+).

**Reasoning**:

- High integration complexity (multiple services).
- Behavior-preserving migrations common (high regression risk).
- Tighter thresholds reduce autonomous build attempts.

**Typical distribution**:

- SIMPLE: 10% (service config, feature flags)
- STANDARD: 40% (new endpoints, business logic)
- COMPLEX: 50% (cross-service features, migrations)

---

### Example 3: Frontend-Heavy Application

**Thresholds**: custom (SIMPLE: 0-8, STANDARD: 9-18, COMPLEX: 19+).

**Reasoning**:

- UI Complexity dimension often scores 2-3.
- Pattern Novelty lower (React patterns established).
- Can handle higher complexity scores.

**Typical distribution**:

- SIMPLE: 30% (component tweaks, styling)
- STANDARD: 60% (new pages, complex components)
- COMPLEX: 10% (design system overhaul, state management)

---

## Summary

**Key takeaways**:

1. Default thresholds (0-6, 7-15, 16+) work well for most projects.
2. Customize based on team size, risk tolerance, project maturity.
3. Tier determines workflow path, not just complexity perception.
4. Autonomous build feasibility requires additional factors beyond tier.
5. Override tiers when business context demands it.
6. Re-triage if feature scope changes significantly.

**Decision tree**:

```
Score calculated (0-27)
  │
  ├─ 0-6: SIMPLE → skip BA/arch, straight to tasks
  ├─ 7-15: STANDARD → full workflow, autonomous build if feasible
  └─ 16+: COMPLEX → full workflow, manual execution
      │
      └─ Consider: should this be split into multiple STANDARD features?
```

**Next steps**:

- Apply defaults initially.
- Collect data on 10+ features.
- Analyze misclassifications.
- Adjust thresholds if needed.
