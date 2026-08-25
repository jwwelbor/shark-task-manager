---
feature_key: E34-F10-product-critical-path-guard-for-delivery-workflows
epic_key: E34
title: Product Critical-Path Guard for Delivery Workflows
description: Add a reusable product-delivery guard that makes Shark planning and execution prompts consult D01, D02, the product delivery roadmap, and a durable product critical-path artifact before selecting or dispatching work. Require explicit path gate, contribution, executable evidence, dependency, and side-quest disposition reporting across epic, feature, task, and sprint workflows.
---

# Product Critical-Path Guard for Delivery Workflows

**Feature Key**: E34-F10-product-critical-path-guard-for-delivery-workflows

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
Shark workflow prompts can optimize locally selected work without first proving that
the work advances the current product-roadmap gate. This permits downstream epics,
component-only evidence, and unrelated backlog fillers to consume delivery capacity
while the production golden path remains blocked.

### Solution
Introduce a durable product critical-path artifact and a reusable delivery guard for
planning and execution prompts. The guard must report the current gate, the last
passing production step, the proposed contribution, executable advancement evidence,
unresolved prerequisites, and the disposition of side work before dispatch.

### Impact
Planning remains tied to one executable next product step instead of backlog volume or
component-level completion.

## Triage Breadcrumb

- The intended project inputs are `docs/product/D01-vision-statement.md`,
  `docs/product/D02-success-criteria.md`,
  `docs/plan/product-delivery-roadmap.md`, and
  `docs/plan/product-critical-path.md`.
- The shared guard is needed in sprint planning/active/closing; epic
  assessment/decomposition/active; feature specification, test planning, task
  generation, task review, and approval; and task development completion reporting.
- Fixture, capture, hand-authored actor, contract-only, and component-suite evidence
  cannot satisfy a production product gate.
- E19-F10 owns runtime sprint admission and goal-acceptance enforcement. E34-F09
  remains the owner for override-drift visibility and downstream WWGM reconciliation.
- Detailed requirements, acceptance criteria, and task decomposition are intentionally
  deferred to this feature's normal workflow.

---

*Last Updated*: 2026-08-24
