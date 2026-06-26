---
name: design-status-model
description: Define fields, meanings, update rules, and freshness expectations for a status-tracking model.
inputs:
  - tracked_objects
  - lifecycle_model
  - audiences
  - decisions_supported
  - update_sources
outputs:
  - status_model
  - staleness_policy
---

# Design Status Model

1. **Identify decisions.** List what each audience needs to decide from the status view.
2. **Define tracked objects.** Name the work items and relationships the model must represent.
3. **Separate dimensions.**
   - Lifecycle state
   - Progress
   - Health
   - Ownership
   - Freshness
   - Blockers
4. **Define fields.** For each field, specify purpose, value shape, source, update trigger, and consumer.
5. **Set staleness policy.** Define freshness windows by object type and decision risk.
6. **Remove redundant signals.** If two fields answer the same question, keep the one with clearer ownership.

## Field Template

```markdown
| Field | Purpose | Values | Source | Update Trigger | Stale After | Consumer Decision |
|---|---|---|---|---|---|---|
| {field} | {purpose} | {values} | {source} | {trigger} | {duration} | {decision} |
```
