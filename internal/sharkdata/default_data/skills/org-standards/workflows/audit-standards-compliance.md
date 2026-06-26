---
name: audit-standards-compliance
description: Compare artifacts against a standard and report concrete deviations.
inputs:
  - standards_document
  - artifacts
  - target_scope
outputs:
  - compliance_report
---

# Audit Standards Compliance

1. **Resolve scope.** Confirm the standard applies to each artifact.
2. **Extract requirements.** Turn the standard into a checklist of mandatory and recommended items.
3. **Inspect artifacts.** Record evidence for each pass or deviation.
4. **Rate severity.**
   - Blocker: violates a mandatory rule and creates correctness, security, or maintenance risk.
   - Warning: violates a recommended convention or creates avoidable inconsistency.
   - Info: stylistic drift with low impact.
5. **Recommend remediation.** Give the smallest change that brings the artifact back into alignment.

## Report Template

```markdown
# Standards Compliance Report

**Scope:** {scope}
**Verdict:** PASS | PASS_WITH_WARNINGS | FAIL

| Rule | Artifact | Severity | Evidence | Remediation |
|---|---|---|---|---|
| {rule} | {path} | {severity} | {evidence} | {fix} |

## Open Questions
- {question}
```
