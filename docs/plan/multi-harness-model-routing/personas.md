# E41 stakeholder responsibilities

The [epic PRD](./epic.md#stakeholder-impact) describes the impact on each role. A person can hold several roles. The distinction matters when a workflow author requests a route that the operator has not approved.

| Stakeholder | Decision or responsibility | Evidence they need |
|---|---|---|
| Workflow author | Classify each step and declare exceptions or review requirements. | Preview naming the selected route and which input won. |
| Project operator | Approve bindings, credentials, permissions, budgets and rollout. Reconcile work when the previous worker might have written. | A check that distinguishes installation, compatibility and entitlement, plus per-attempt failure and retirement evidence. |
| Gate owner | Decide whether a review can count for a specific candidate under the declared separation policy. | Candidate identity, every contributing producer and observed reviewer identity for the same attempt. |
| Benchmark maintainer | Select comparable profiles and determine whether E40 can accept a rerun for comparison. | Frozen workload and plan digests, versioned bindings and worker-correlated route evidence. |

The [user journeys](./user-journeys.md) show when each role acts.
