# Pull authorized work by role

> **Compatibility note:** This document is a historical / compatibility reference.
> It no longer describes a sanctioned normal-path claim. The only sanctioned
> claim path is "Sanctioned path: Rider re-entry" below. Everything under
> "Historical reference: worker-owned child mode" documents a retired
> direct-claim procedure, retained for compatibility only — do not follow it
> as a normal procedure.

## Authority boundary

The workflow-resolved `agent_type` is the only role input to a pull. The
owning sprint procedure passes that value to
`SprintService.GetNextTask(ctx, agentType)` through its
`shark sprint next --agent=<type>` adapter. That read-only selector filters
non-terminal items by workflow-role eligibility before sorting and preserves
the existing priority/dependency order. It does not inspect lease state or
perform blocked/gate filtering: a returned item can still encounter a
workflow gate or a non-force claim conflict at the owning workflow or
claim-service path.

Do not substitute a roster role, legacy `agent` assignment, or `model_tier`
for the workflow-resolved `agent_type`. Roster membership describes available
expertise; it does not grant claim or status authority. A roster's model
preference cannot select work or override workflow metadata.

## Sanctioned path: Rider re-entry

This is the only sanctioned claim path. It supersedes the historical
worker-owned child mode described below.

1. Start from the workflow role already resolved for the worker. If no role
   is resolved, do not infer one from the roster; report the missing
   authority to the coordinator.
2. Select a candidate using the existing Rider sprint pull procedure. Its
   owning adapter invokes `shark sprint next --agent=<type>` and keeps
   deterministic priority/dependency order in
   `SprintService.GetNextTask(ctx, agentType)`. Do not recreate selection
   logic in this skill.
3. Inspect the returned child key and workflow role. If no eligible child is
   returned, report that outcome without claiming unrelated work.
4. Invoke `/shark-rider run <selected-key>`. Role-aware Rider self-pull never
   claims or executes the returned `BacklogItemView` directly.
   `/shark-rider run <selected-key>` calls `shark next <selected-key>
   --json`; only its `response.entity_key` is claimable, and it dispatches
   exact `response.prompt` with `response.provider` and `response.model`
   metadata. Do not invoke the historical worker-owned claim step below, or
   the child-worker ownership contract, for a Rider-dispatched worker.

## Historical reference: worker-owned child mode (compatibility only)

> Everything below this heading is retained for compatibility only. It
> describes a retired direct-claim procedure, not a sanctioned normal path.

### Goal (historical)

Let an existing coordinator delegate one worker-owned child only after the
workflow authorizes its role. This worker-owned child mode is not
`/shark-rider run`: the Rider parent owns the concrete entity's selection,
claim, session, transition, and release in that loop.

### Procedure (historical, compatibility only)

1. Start from the workflow role already resolved for the worker. If no role is
   resolved, do not infer one from the roster; report the missing authority to
   the coordinator.
2. Use the existing Rider sprint pull procedure. Its owning adapter invokes
   `shark sprint next --agent=<type>` and keeps deterministic
   priority/dependency order in `SprintService.GetNextTask(ctx, agentType)`.
   Do not recreate selection logic in this skill.
3. Inspect the returned child key and workflow role. If no eligible child is
   returned, report that outcome without claiming unrelated work.
4. In worker-owned child mode, ask the owning claim path to claim exactly that
   child through `ClaimService.Claim`. The claim service owns session
   identity, conflict handling, leases, retries, and any atomic claim-next
   behavior. Never force-claim and never construct a detached claim from
   roster data.
5. Work only on the claimed child, then follow
   `context/worker-ownership.md` for the bounded evidence return to the parent
   coordinator. Do not hand this child session to `/shark-rider run`.

### Missing prerequisites and capability (historical, compatibility only)

For missing product gates, recommend bootstrap or escalation; do not guess
product decisions. If the required team capability is unavailable, state the
gap and use an explicit sequential fallback only when it is safe. Otherwise,
stop with an actionable capability gap. These recommendations preserve
ordinary `/shark-rider run` routing; this protocol never silently replaces
ordinary dispatch or adds a team runtime.

### Result (historical, compatibility only)

In worker-owned child mode, the worker either has one role-authorized child
claimed through the existing service or returns a bounded reason that it
cannot safely proceed. The workflow engine, sprint service, and claim
service remain the only routing and lease authorities.
