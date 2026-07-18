# Execute a chair-led Shark Attack run

## Goal

Coordinate role-aware work around the ordinary `/shark-rider run <root>` loop
without adding a team runtime, command, claim store, or aggregate status.

## Procedure

1. Read the roster, durable council decisions, handoffs, unresolved escalations,
   and relevant inbox entries. Shark claims, history, and `shark next` remain
   the operational source of truth.
2. For assigned work, let the Rider parent call `shark next <root> --json`,
   claim the returned concrete entity, pass `response.prompt` unchanged to one
   worker, apply its configured outcome, then release the same session.
3. For a role-aware Rider self-pull, use the workflow-resolved role selection
   boundary in `pull-by-role.md`. Roster membership, responsibility prose,
   legacy assignment, and `model_tier` never grant authority. Use
   `shark sprint next --agent=<type>` only to select `selected-key`, then
   invoke `/shark-rider run <selected-key>`. Do not claim the `BacklogItemView`
   selection directly. The ordinary Rider loop calls `shark next <selected-key>
   --json`, then claims `response.entity_key`, dispatches exact
   `response.prompt` with `response.provider` and `response.model` metadata,
   handles the outcome, and releases that session. Do not invoke
   `pull-by-role.md`'s worker-owned claim step or hand a worker-owned child
   session into `/shark-rider run`.
4. A Rider-dispatched worker provides scoped craft and evidence only. The
   parent owns claim, heartbeat, notes, kickbacks, workflow transition, and
   release for its dispatched entity. The worker does not select a replacement
   entity or mutate the dispatched entity's lease or workflow state.
5. Record a bounded handoff or decision when a worker changes. For missing
   evidence, material scope/architecture/quality questions, or unresolved
   disagreement, follow `escalate.md`. Every escalation records the material
   question, evidence, responsible role, requested decision, route, and next
   owner; absent policy routes to `council-review`.
6. Stop on `pause`, `archive`, `error`, or an explicit human gate. A refreshed
   coordinator follows `resume.md`; do not create a second resume record or
   store prompts, credentials, transcripts, or unrestricted output.

## Result

The chair coordinates evidence and escalation while Shark and the Rider retain
the complete execution and state authority.

## Separate worker-owned child mode

The worker-owned child mode described by `pull-by-role.md` and
`context/worker-ownership.md` is not `/shark-rider run`. Use it only when an
existing coordinator explicitly owns delegated-child execution and its separate
lease lifecycle. Do not transfer its claim session into this parent loop.
