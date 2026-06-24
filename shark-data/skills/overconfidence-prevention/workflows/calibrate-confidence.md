---
name: calibrate-confidence
description: Assess whether a claim or recommendation is calibrated to its evidence.
inputs:
  - claim_or_recommendation
  - evidence
  - decision_context
  - alternatives
outputs:
  - calibration_report
  - assumptions
  - missing_evidence
  - safer_rewrite
---

# Calibrate Confidence

1. **Restate the claim.** Make the conclusion explicit and narrow.
2. **Classify evidence.**
   - Direct observation
   - Test result
   - Source citation
   - Code or artifact inspection
   - Inference
   - Analogy
   - Assumption
3. **Map coverage.** List what was checked and what was not checked.
4. **Generate alternatives.** Name plausible competing explanations or options.
5. **Assess consequence.** Rate the cost of being wrong as low, medium, or high.
6. **Assign confidence.** Use high, medium, or low with rationale.
7. **Rewrite safely.** Replace overclaiming with calibrated language and next verification steps.

## Report Template

```markdown
# Confidence Calibration

**Claim:** {claim}
**Recommended Confidence:** HIGH | MEDIUM | LOW

## Evidence
| Evidence | Type | Strength | Notes |
|---|---|---|---|
| {evidence} | {type} | {strength} | {notes} |

## Assumptions
- {assumption} — confidence: {level}; validation: {method}

## Alternatives
- {alternative}

## Missing Evidence
- {missing evidence}

## Safer Rewrite
{calibrated wording}
```
