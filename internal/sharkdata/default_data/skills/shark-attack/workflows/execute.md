# Execute Shark work with the chair-led protocol

## Goal

Compose role-aware selection, the ordinary Rider dispatch loop, and bounded
council handoff or escalation. Shark remains the authority for workflow state,
prompts, claims, leases, and history; this procedure adds no second runtime.

## Start work

1. For assigned work, invoke `/shark-rider run {ENTITY_KEY}`.
2. For role-aware self-pull, follow `pull-by-role.md`. It uses the existing
   workflow-role selector to choose one key, then invoke `/shark-rider run
   {ENTITY_KEY}` for that exact selection. The selection is not a claim,
   authorization, prompt, or replacement workflow engine.
3. The Rider parent calls `shark next {ENTITY_KEY} --json`, claims the
   returned concrete entity, and sends `response.prompt` unchanged to one
   worker. Do not reconstruct a prompt from `shark get`, roster data, local
   skills, or a persona name.

## Parent and worker boundary

The Rider parent owns claim, heartbeat, kickbacks, configured outcome advance,
and session release for the dispatched entity. The worker follows
`context/worker-ownership.md`: perform scoped craft, write bounded notes,
context, and evidence, and return a semantic outcome. It does not claim,
heartbeat, release, advance, or status-set the dispatched entity.

On `pause`, `archive`, `error`, an explicit human gate, or a missing outcome,
the Rider follows the ordinary `/shark-rider run` stop and blocker rules. Do
not retry an error blindly, fabricate a terminal state, or report partial work
as success. Apply any task kickback before the parent advances and release the
session on every success, failure, or exception path.

## Handoff and escalation

Before a refresh or material escalation, write a bounded decision, handoff, or
escalation using `communicate.md` and `escalate.md`. Include the root and child
scope, question, evidence, responsible role, requested decision, route, and
next owner. For a material scope, architecture, or quality question without a
configured policy, use `route: council-review` and recommend pause/review; do
not name a fixed human destination or guess the decision.

Store relative council artifact pointers and concise metadata only. Do not put
rendered prompts, credentials, access tokens, secrets, or unrestricted worker
transcripts in council records. Non-material questions remain a bounded
handoff or decision rather than an automatic escalation.

## Refresh and result

A refreshed coordinator first inspects ordinary Shark claim and history state,
then follows `resume.md` for matching `docs/council/decisions/`,
`docs/council/handoffs/`,
`docs/council/escalations/`, and inbox pointers. Resume from those bounded
records; do not require prior chat or create a separate run ledger.

The result is either one ordinary Rider run, a bounded no-item/no-role/conflict
outcome from role pull, or a durable escalation and pause/review recommendation.
The procedure reuses `pull-by-role.md`, `escalate.md`, and `resume.md`; it adds
no command, provider configuration, or workflow-state authority.
