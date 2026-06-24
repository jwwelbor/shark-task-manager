---
name: content-validation
description: Validate written artifacts for completeness, internal consistency, evidence quality, clarity, and readiness before they are used downstream. Use for spec, plan, report, design, or review content that needs a rubric-based quality pass without becoming a workflow gate.
when_to_use: when a document or generated artifact needs quality validation for completeness, coherence, traceability, and actionability before another activity relies on it
version: 1.0.0
domain: artifact-quality
inputs:
  - artifact_text: the content to validate
  - artifact_type: spec, plan, report, design, review, or other artifact category
  - expected_sections: required sections or structural expectations
  - source_material: upstream requirements, notes, or evidence the artifact should reflect
  - audience: who will consume the artifact
outputs:
  - selected_workflow: one of {validate-content-quality}
  - validation_report: rubric-based findings with severity and evidence
  - missing_content: required or implied content not present
  - consistency_issues: contradictions, duplicate claims, or unsupported assertions
  - recommended_edits: concrete improvements
---

# Content Validation Skill

This skill evaluates whether an artifact is ready to be trusted by a downstream reader. It is a content-quality method, not a lifecycle gate. It identifies gaps and recommends edits; the host decides what to do with the result.

## Workflow Selection

Use `workflows/validate-content-quality.md` for the full validation pass. It checks structure, completeness, consistency, evidence, clarity, traceability, and actionability.

## Validation Rubric

1. **Complete:** Required sections are present and substantive.
2. **Consistent:** Claims do not contradict each other or the supplied source material.
3. **Traceable:** Important claims can be traced to evidence, context, or explicit assumptions.
4. **Specific:** Recommendations and requirements are precise enough to act on.
5. **Audience-fit:** The artifact gives its intended reader the right level of detail.
6. **Decision-ready:** Open questions, risks, and next actions are clearly separated.

## Severity

- **Blocker:** Downstream work would likely make a wrong decision or fail due to the issue.
- **Major:** The artifact is usable only with additional interpretation or clarification.
- **Minor:** The artifact is mostly sound but should be improved for clarity or maintainability.
- **Info:** Optional polish or future improvement.

## Output Requirements

Every finding must include:

- The affected section or quoted span
- The rubric dimension violated
- Why it matters
- Severity
- Recommended edit
