# ADR-002: Outcome-Routed Review Kickbacks

**Date:** 2026-11-21
**Status:** Accepted

## Context

Review results can name child entities to reopen. The current kickback payload also supplies a concrete `target_status`, and the persistence coordinator can reopen a terminal child directly. That lets worker output choose a status outside the target entity's configured outcome route and can bypass E38-F09's open-Question hold. A parent lease establishes who may apply a child mutation; it does not select the child's next state.

## Decision

Use the review's configured outcome key (`Request.OutcomeKey`) as the route input for the reviewed entity and each named child kickback. The outcome key is distinct from its resolved semantic role (for example, `fail` may map to `kickback_rework`). Resolve child destinations from the target entity's configured `rework_outcomes` for its current state, including terminal states. A worker-supplied `target_status` remains required by GateResult v1 as a compatibility assertion and must match the resolved route; removing it requires a new contract version. Rider must not use `shark status set --force` for review kickbacks.

Use a dedicated routed-rework transition path, not a force bypass. Verify the parent entity/session lease in the same database transaction as each child status, history, and dependent-state write. For a fresh result, preflight all child routes and Question holds before target-store persistence. On resume, reconcile already-committed kickbacks by suboperation ID and history first, then preflight every pending route and Question before applying any pending kickback; use the recorded destination for completed operations and resolve pending operations from their current source. A pre-existing open or answering Question, missing route, or mismatched compatibility assertion fails closed before persistence. A Question or lease loss encountered after an authorized prefix leaves that prefix intact and prevents later child and parent transitions.

## Rationale

The workflow owns state routing. Semantic dispositions remain stable across projects even when their configured destination statuses differ. Explicit rework routes let a project define how a rejected review reopens a terminal child without granting the worker arbitrary status authority. The Question gate remains authoritative for both ordinary and review-triggered transitions.

## Consequences

**Positive:**
- Review results cannot select arbitrary child statuses.
- Route changes stay in workflow configuration instead of prompts.
- Open Questions stop child rework before the review advances its parent.
- The live parent lease authorizes each child write at its transaction boundary.
- Replay preserves the original child destination and never reroutes an applied kickback from its new state.

**Negative:**
- Projects that use child kickbacks must configure `rework_outcomes` for each applicable child state.
- GateResult v1 continues to require `target_status`; it is accepted only when it agrees with the resolved route.

## References

- [B077 repair plan revision 27](../../plan/bugs/B077.repair-plan.md)
- [E38-F09 AC-006 and D-006](../../plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/spec.md)
- [B077 owner disposition and architect consultation](../../plan/bugs/B077.repair-plan-evidence.md#revision-26-owner-disposition-and-route-design)
