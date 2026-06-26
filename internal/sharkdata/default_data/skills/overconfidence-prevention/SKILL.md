---
name: overconfidence-prevention
description: Detect and reduce overconfident reasoning in analysis, plans, estimates, reviews, and recommendations. Use when conclusions may be based on weak evidence, hidden assumptions, premature certainty, or insufficient exploration of alternatives.
when_to_use: when an answer, plan, estimate, or review needs calibration against evidence strength, uncertainty, assumptions, and plausible alternatives
version: 1.0.0
domain: reasoning-calibration
inputs:
  - claim_or_recommendation: the conclusion, plan, estimate, or assertion to calibrate
  - evidence: observations, sources, tests, or constraints supporting the claim
  - decision_context: what decision depends on the claim
  - alternatives: competing explanations or options already considered
outputs:
  - selected_workflow: one of {calibrate-confidence}
  - calibration_report: evidence-strength assessment and confidence recommendation
  - assumptions: explicit assumptions with confidence and validation method
  - missing_evidence: information that would materially change confidence
  - safer_rewrite: calibrated wording or decision framing
---

# Overconfidence Prevention Skill

This skill helps an agent avoid sounding more certain than the evidence supports. It does not force indecision; it makes confidence proportional to evidence, risk, and reversibility.

## Workflow Selection

Use `workflows/calibrate-confidence.md` for any conclusion that needs a confidence check before it informs a decision.

## Calibration Checks

1. **Evidence strength:** Is the conclusion based on direct evidence, inference, analogy, or guesswork?
2. **Coverage:** Did the analysis inspect the relevant cases, files, stakeholders, or failure modes?
3. **Alternatives:** What else could explain the same observations?
4. **Assumptions:** Which premises are unstated, unverified, or inherited from prior work?
5. **Reversibility:** How costly is it if the conclusion is wrong?
6. **Wording:** Does the answer distinguish fact, inference, recommendation, and uncertainty?

## Confidence Labels

- **High:** Direct evidence covers the important cases; alternatives are weak or ruled out.
- **Medium:** Evidence is credible but partial; alternatives remain plausible.
- **Low:** Evidence is indirect, sparse, stale, or highly assumption-dependent.

## Output Standard

Always separate:

- What is known
- What is inferred
- What is assumed
- What remains uncertain
- What would change the recommendation
