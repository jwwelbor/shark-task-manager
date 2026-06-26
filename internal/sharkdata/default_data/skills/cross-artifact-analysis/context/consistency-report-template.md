# Consistency Report Template

```markdown
# Cross-Artifact Consistency Report

**Artifacts Compared:** {list}
**Verdict:** CONSISTENT | CONSISTENT_WITH_WARNINGS | DRIFTED

## Summary
{brief summary}

## Mismatches
| Severity | Dimension | Layers | Evidence | Remediation |
|---|---|---|---|---|
| {severity} | {dimension} | {layers} | {quoted evidence} | {fix} |

## Traceability Matrix
| Parent Requirement | Child Coverage | Status | Notes |
|---|---|---|---|
| {requirement} | {child refs} | covered/partial/missing | {notes} |

## Uncompared Artifacts
- {artifact and reason}
```
