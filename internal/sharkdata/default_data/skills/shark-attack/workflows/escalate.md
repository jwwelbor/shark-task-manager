# Escalate a material council question

## Goal

Record an unresolved material question, keep its evidence durable, and route
it without inventing a fixed human destination.

## Escalate when

Escalate when evidence is missing, direction changes materially, specialists
disagree, or a process or quality blocker remains unresolved.

## Steps

1. Check for `docs/product/escalation_triggers.md`. If it exists, apply its
   configured trigger and route rules.

2. Create a bounded escalation artifact under `docs/council/escalations/`.
   Include an artifact ID, trigger, requested decision, evidence, sender and
   recipient roles, root key, optional child key, route, status, timestamps,
   and `next_action`.

3. For a material unresolved question when the policy file is absent, set
   `status: unresolved`, set `route: council-review`, and recommend
   `pause/review`. Do not guess a product answer or name a fixed human
   destination.

4. When the question is resolved, add the resolution or a reference to its
   decision artifact. Keep the original escalation and update its bounded
   metadata rather than deleting the audit trail.

5. Do not create an escalation for a non-material question solely because the
   policy file is absent. Record the reason for no escalation in the relevant
   decision or handoff when a durable record is useful.

## Result

The council has an actionable, structured pause point. The owning coordinator
retains workflow-transition and root-lease authority.
