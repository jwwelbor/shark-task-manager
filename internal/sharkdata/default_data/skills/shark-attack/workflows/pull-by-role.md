# Pull authorized work by role

## Goal

Let a council member self-pull only work authorized for its role while keeping
workflow selection, sprint ordering, claims, and root dispatch ownership in
their existing Shark authorities.

## Authority boundary

The workflow-resolved `agent_type` is the only role input to a pull. The
owning sprint procedure passes that value to
`SprintService.GetNextTask(ctx, agentType)` through its
`shark sprint next --agent=<type>` adapter. That service performs read-only
selection: it preserves the existing priority/dependency order for eligible
non-terminal sprint work and does not exclude live claims.

Do not substitute a roster role, legacy `agent` assignment, or `model_tier` for
the workflow-resolved `agent_type`. Roster membership describes available
expertise; it does not grant claim or status authority. A roster's model
preference cannot select work or override workflow metadata.

The sprint result selects one returned child only. It includes the workflow
role and canonical prompt metadata needed by the owning dispatch path; it is
not a claim, a role authorization, a free-form role assignment, or a second
workflow engine. A direct local `shark claim` is a lease operation, not role
authorization.

## Procedure

1. Start from the workflow role already resolved for the worker. If no role is
   resolved, do not infer one from the roster. Return a no-role outcome to the
   parent Rider loop.
2. Use the existing Rider sprint pull procedure. Its owning adapter invokes
   `shark sprint next --agent=<type>` and keeps deterministic
   priority/dependency order in `SprintService.GetNextTask(ctx, agentType)`.
   This is read-only selection, not a claim. Do not recreate selection logic in
   this skill.
3. Inspect the returned child key, workflow role, and canonical prompt
   metadata. If no eligible child is returned, return a no-item outcome to the
   parent Rider loop without claiming unrelated work.
4. Ask the owning claim path to claim exactly that child through
   `ClaimService.Claim`. ClaimService owns session generation, expiry
   reclamation, claim conflict reporting, heartbeat, and session-scoped
   release. If a live lease wins the race, return a claim conflict outcome to
   the parent Rider loop. Never force-claim, select another role, steal a
   lease, or construct a detached claim from roster data.
5. Work only on the claimed child, then follow
   `context/worker-ownership.md` for the bounded evidence return to the parent
   coordinator.

## Return bounded outcomes

- **No role:** Return the missing workflow-role outcome. Do not infer a role
  from the roster, legacy `agent` assignment, actor identity, or `model_tier`.
- **No item:** Return the no-item outcome. Do not claim another item.
- **Claim conflict:** Return the conflict. Do not force-claim, retry with
  another role, or steal the live lease.
- **Workflow pause/gate:** For a workflow pause/gate, return the pause or gate
  outcome. Do not transition workflow state or release the dispatched parent
  lease.

The parent Rider loop owns these outcomes, the dispatched parent lease, and
workflow transitions.

## Missing prerequisites and capability

For missing product gates, recommend bootstrap or escalation; do not guess
product decisions. If the required team capability is unavailable, state the
gap and use an explicit sequential fallback only when it is safe. Otherwise,
stop with an actionable capability gap. These recommendations preserve ordinary
`/shark-rider run` routing; this protocol never silently replaces ordinary
dispatch or adds a team runtime.

## Result

The worker either has one role-selected child claimed through the existing
service or has returned a bounded reason that it cannot safely proceed. The
workflow engine, sprint service, and claim service remain the only routing and
lease authorities.
