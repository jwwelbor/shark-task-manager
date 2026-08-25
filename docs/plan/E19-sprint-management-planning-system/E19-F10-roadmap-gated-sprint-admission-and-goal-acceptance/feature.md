---
feature_key: E19-F10-roadmap-gated-sprint-admission-and-goal-acceptance
epic_key: E19
title: Roadmap-Gated Sprint Admission and Goal Acceptance
description: Prevent sprints from admitting or dispatching downstream work whose ancestor epic dependencies or current product-roadmap entry conditions are incomplete. Apply the policy to sprint add, readiness, planning output, and next selection; support a reasoned override for exceptional prerequisite work; and distinguish completed backlog items from an accepted executable Sprint Goal demonstration during closing.
---

# Roadmap-Gated Sprint Admission and Goal Acceptance

**Feature Key**: E19-F10-roadmap-gated-sprint-admission-and-goal-acceptance

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
Sprint assignment and readiness currently inspect direct task dependencies and capacity,
but do not reject work whose ancestor epic prerequisites or product-roadmap entry
conditions are incomplete. A sprint can therefore score as ready and drain its backlog
without demonstrating its declared product outcome.

### Solution
Gate sprint admission, readiness, and next-work selection on ancestor dependencies and
the current roadmap layer. Allow exceptional prerequisite work only through an explicit
reasoned override, expose the selected portfolio gate in planning output, and require a
Sprint Goal Review before closing.

### Impact
Sprint capacity advances a coherent, executable product increment, and completion of
assigned work is no longer confused with acceptance of the Sprint Goal.

## Triage Breadcrumb

- Apply ancestor epic dependency enforcement to `shark sprint add`, readiness, and
  sprint next/plan selection.
- Readiness must fail for assigned work in a downstream roadmap layer and must not be
  improved by filler added only to satisfy the three-item score threshold.
- Exceptional prerequisite work requires an explicit override and recorded reason.
- Sprint planning output must include the portfolio-selected epic and current gate.
- Planning should select one epic plus named direct blockers and require one executable
  Sprint Goal; unrelated bugs and debt remain deferred even when capacity is available.
- Closing must inspect or run the declared demo, record the golden-path step before and
  after, return the sprint to active when the goal is not demonstrated, and distinguish
  task completion from goal acceptance.
- E19-F09 remains the owner for consolidating the `shark plan` and compatibility
  `shark sprint next` selection surfaces; E34-F10 owns the reusable prompt guard.
- Detailed requirements, acceptance criteria, and task decomposition are intentionally
  deferred to this feature's normal workflow.

---

*Last Updated*: 2026-08-24
