---
name: audit-status-signal
description: Evaluate whether an existing status report or model is accurate, fresh, and decision-useful.
inputs:
  - tracked_objects
  - status_view
  - audiences
  - decisions_supported
  - update_sources
outputs:
  - signal_audit
---

# Audit Status Signal

1. **Map consumers to decisions.** Identify what the status view is supposed to help decide.
2. **Check completeness.** Verify each decision has enough fields to support it.
3. **Check freshness.** Compare update timestamps or evidence to freshness expectations.
4. **Check ambiguity.** Flag labels that can mean multiple things or hide blockers.
5. **Check redundancy.** Identify duplicated or conflicting fields.
6. **Report fixes.** Recommend field additions, removals, renames, or freshness rules.

## Report Template

```markdown
# Status Signal Audit

**Verdict:** USEFUL | USEFUL_WITH_WARNINGS | MISLEADING

| Finding | Severity | Evidence | Decision Impact | Recommendation |
|---|---|---|---|---|
| {finding} | {severity} | {evidence} | {impact} | {recommendation} |

## Missing Signals
- {signal}

## Stale Signals
- {signal}
```
