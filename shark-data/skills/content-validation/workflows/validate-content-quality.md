---
name: validate-content-quality
description: Validate an artifact against completeness, consistency, evidence, clarity, traceability, and actionability criteria.
inputs:
  - artifact_text
  - artifact_type
  - expected_sections
  - source_material
  - audience
outputs:
  - validation_report
  - missing_content
  - consistency_issues
  - recommended_edits
---

# Validate Content Quality

1. **Identify the artifact contract.** Determine what the artifact type must contain and who relies on it.
2. **Check structure.** Compare actual sections to `expected_sections`; flag missing or empty sections.
3. **Check source alignment.** Compare major claims against `source_material`; flag unsupported or contradictory claims.
4. **Check internal consistency.** Look for conflicting terminology, duplicated decisions, or mismatched numbers.
5. **Check specificity.** Replace vague assertions with exact actors, conditions, outcomes, thresholds, or examples.
6. **Check actionability.** Ensure recommendations name the action, owner or audience, and decision impact.
7. **Produce the report.** Group findings by severity and provide concrete edits.

## Report Template

```markdown
# Content Validation Report

**Artifact:** {artifact_type}
**Verdict:** READY | READY_WITH_EDITS | NOT_READY

## Summary
{brief assessment}

## Findings
| Severity | Section | Dimension | Evidence | Recommended Edit |
|---|---|---|---|---|
| {severity} | {section} | {dimension} | {quote or summary} | {edit} |

## Missing Content
- {item}

## Consistency Issues
- {issue}

## Recommended Edits
1. {edit}
```
