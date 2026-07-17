# Pull authorized work by role

## Goal

Let an existing coordinator delegate one worker-owned child only after the
workflow authorizes its role. This worker-owned child mode is not
`/shark-rider run`: the Rider parent owns the concrete entity's selection,
claim, session, transition, and release in that loop.

## Authority boundary

The workflow-resolved `agent_type` is the only role input to a pull. The
owning sprint procedure passes that value to
`SprintService.GetNextTask(ctx, agentType)` through its
`shark sprint next --agent=<type>` adapter. That service preserves the existing
priority/dependency order and excludes ineligible, blocked, or already-claimed
work.

Do not substitute a roster role, legacy `agent` assignment, or `model_tier` for
the workflow-resolved `agent_type`. Roster membership describes available
expertise; it does not grant claim or status authority. A roster's model
preference cannot select work or override workflow metadata.

The sprint result is an authorization to select one returned child only. It is
not a claimable dispatch response, a free-form role assignment, or a second
workflow engine. It carries no canonical prompt metadata or provider metadata;
those come only from the ordinary Rider dispatch path's `shark next` response.

## Procedure

1. Start from the workflow role already resolved for the worker. If no role is
   resolved, do not infer one from the roster; report the missing authority to
   the coordinator.
2. Use the existing Rider sprint pull procedure. Its owning adapter invokes
   `shark sprint next --agent=<type>` and keeps deterministic
   priority/dependency order in `SprintService.GetNextTask(ctx, agentType)`.
   Do not recreate selection logic in this skill.
3. Inspect the returned child key and workflow role. If no eligible child is
   returned, report that outcome without claiming unrelated work.
4. In worker-owned child mode, ask the owning claim path to claim exactly that child through
   `ClaimService.Claim`. The claim service owns session identity, conflict
   handling, leases, retries, and any atomic claim-next behavior. Never
   force-claim and never construct a detached claim from roster data.
5. Work only on the claimed child, then follow
   `context/worker-ownership.md` for the bounded evidence return to the parent
   coordinator. Do not hand this child session to `/shark-rider run`.

For a role-aware Rider self-pull, use the workflow-role selection boundary in
steps 1 through 3, then invoke `/shark-rider run <selected-key>`. Role-aware
Rider self-pull never claims or executes the returned `BacklogItemView`
directly. `/shark-rider run <selected-key>` calls `shark next <selected-key>
--json`; only its `response.entity_key` is claimable, and it dispatches exact
`response.prompt` with `response.provider` and `response.model` metadata. Do
not invoke this worker-owned claim step or the child-worker ownership contract
for a Rider-dispatched worker.

## Missing prerequisites and capability

For missing product gates, recommend bootstrap or escalation; do not guess
product decisions. If the required team capability is unavailable, state the
gap and use an explicit sequential fallback only when it is safe. Otherwise,
stop with an actionable capability gap. These recommendations preserve ordinary
`/shark-rider run` routing; this protocol never silently replaces ordinary
dispatch or adds a team runtime.

## Result

In worker-owned child mode, the worker either has one role-authorized child
claimed through the existing service or returns a bounded reason that it cannot
safely proceed. In `/shark-rider run`, the Rider parent owns the selected
entity's session instead. The workflow engine, sprint service, and claim
service remain the only routing and lease authorities.
