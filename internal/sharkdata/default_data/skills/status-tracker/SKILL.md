---
name: status-tracker
description: Design and audit project-status tracking models, including state fields, progress signals, staleness indicators, owner visibility, and handoff health. Use when defining what status information should be captured or evaluating whether a status view accurately represents work.
when_to_use: when a project, feature, task, or operational process needs a clear status model, progress indicators, staleness detection, or status-report quality review
version: 1.0.0
domain: project-status-modeling
inputs:
  - tracked_objects: entities or work items whose status must be represented
  - lifecycle_model: known phases, states, or handoff points
  - audiences: people or agents consuming the status view
  - decisions_supported: decisions the status information should enable
  - update_sources: where status evidence comes from
outputs:
  - selected_workflow: one of {design-status-model, audit-status-signal}
  - status_model: fields, meanings, allowed values, and update rules
  - staleness_policy: freshness thresholds and stale-signal behavior
  - signal_audit: findings about missing, misleading, or redundant status signals
---

# Status Tracker Skill

This skill defines what status information should exist and how to judge whether it is trustworthy. It focuses on the domain model for visibility and handoffs. It does not prescribe a workflow engine or perform state transitions.

## Workflow Selection

### Design Status Model

Use `workflows/design-status-model.md` when creating or revising the fields and signals that describe work state.

### Audit Status Signal

Use `workflows/audit-status-signal.md` when evaluating an existing status view or report for accuracy, completeness, and freshness.

## Principles

1. **Status supports decisions.** Track only fields that help someone decide what to do next.
2. **Separate state from health.** State says where work is; health says whether that state is okay.
3. **Make staleness visible.** Old status can be worse than no status.
4. **Use observable evidence.** Prefer timestamps, completion markers, blockers, and handoff records over vague labels.
5. **Avoid duplicate truth.** A field should have a clear source of authority.

## Common Fields

- Current state or phase
- Owner or responsible role
- Last meaningful update
- Blockers and dependencies
- Progress signal
- Health indicator
- Next expected action
- Due or review date

## Output Standard

For every proposed field, define:

- Purpose
- Allowed values or shape
- Source of truth
- Update trigger
- Staleness threshold
- Consumer decision supported
