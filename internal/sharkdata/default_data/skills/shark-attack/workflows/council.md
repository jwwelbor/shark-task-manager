# Council: route and resolve material questions

## Goal

Define the boundary between a routine question (the E39 `Q###` loop
`route-question.md` implements) and a material question — one that needs
council-level adjudication, per epic architecture §4.5's council
communication contract (I-04). This file owns that boundary and, once a
question crosses it, owns the question end to end. It also carries forward
two procedures this restructure supersedes: the bounded council inbox and
acknowledgement rule, and the material-question routing procedure.

## The material threshold

A question is material when it trips any `context/operating-model.md`
Axis-1 Council criterion — specialists disagree, a cross-feature or
cross-epic contract is missing or inconsistent, the change has high blast
radius, the change is irreversible, or a worker cannot proceed safely from
project evidence — or when it is not scope-bounded and single-role within
one of three named impact areas: **scope** (product intent or feature/epic
scope), **architecture** (a structural or contract decision), or
**quality-gate** (an existing quality gate is blocked). Link to
`context/operating-model.md` for the Axis-1 criteria rather than
re-deriving them here.

A question that is scope-bounded, single-role, and touches none of those
three impact areas is routine. It never reaches this file.

## Category vocabulary → default path

Every worker-reported `category` value from
`context/worker-control-schema.yaml` (`product|requirements|architecture|
quality|process`) resolves to exactly one default path below. `product`,
`architecture`, and `quality` name the threshold's three impact areas, so
they default to this file; `requirements` and `process` default to the E39
`Q###` loop. The threshold statement above is each row's single tie-break
rule — it can move a `product`/`architecture`/`quality` question to routine
when it is scope-bounded and single-role, and it can move a
`requirements`/`process` question to material when it trips an Axis-1
criterion. No category is left without a default, and none is routable to
both paths without this shared rule deciding it.

| Category | Default path | Off-default when |
|---|---|---|
| `product` | Council (this file) | scope-bounded, single-role, no cross-entity impact → routine |
| `architecture` | Council (this file) | scope-bounded, single-role, no cross-entity impact → routine |
| `quality` | Council (this file) | scope-bounded, single-role, no cross-entity impact → routine |
| `requirements` | `route-question.md` (E39 `Q###`) | trips an Axis-1 criterion above → material |
| `process` | `route-question.md` (E39 `Q###`) | trips an Axis-1 criterion above → material |

## Routine path

A routine question creates no `docs/council/` artifact. Route it through
`route-question.md`'s full mint-through-resolve E39 `Q###` loop instead —
this file does not duplicate that procedure.

## Dispatch topology

Once a question is classified material, its resolution still applies the
execution topology `coordinate.md` resolved in its own step 2 —
`Sequential`, `Parallel with ownership`, or `Parallel with isolation` — the
same way `batch.md` does. See `context/operating-model.md` for the topology
definitions and degradation rule, and `execute-wave.md` for the parallel
wave mechanics; this file does not restate either.

## Route a material question

A material question routes through this file only — never through a second
escalation format or destination.

1. Check for `docs/product/escalation_triggers.md`. If it exists, apply its
   configured trigger and route rules instead of the default below.
2. Create one bounded artifact under `docs/council/escalations/` per
   `context/message-schema.md`'s artifact contract: artifact ID, trigger,
   requested decision, evidence, sender and recipient roles, root key,
   optional child key, route, status, and `next_action`.
3. When the policy file in step 1 is absent, set `status: unresolved`, set
   `route: council-review`, and recommend `pause/review`. Do not guess a
   product answer or name a fixed human destination.
4. Exactly one artifact file exists per material question. Its
   `artifact_id`, `trigger`, evidence, roles, and `created_at` never change
   after creation; resolving the question updates only that same file's
   `status`, `updated_at`, and `next_action` fields, and adds the
   resolution or a pointer to its decision artifact. Never create a second
   artifact for the same question and never delete the audit trail.
5. Do not create an escalation for a question this file's threshold
   classifies routine solely because the policy file in step 1 is absent.

## Council inbox and acknowledgement

Store in-flight council messages below `docs/council/inbox/<member-id>/`
per `context/message-schema.md`'s inbox contract. Keep the body to the
context needed to act — no rendered prompts, credentials, access tokens,
unrestricted worker output, absolute paths, or paths that escape
`docs/council/`.

After acting, write the result as a bounded artifact in
`docs/council/decisions/`, `handoffs/`, or `escalations/` (a resolution is
also filed under `escalations/`, per `context/message-schema.md`), following
that file's artifact-field contract.

Acknowledge or remove the inbox message only after the durable artifact is
present. Repeating an acknowledgement for an already acknowledged message is
a no-op.

Reuse an artifact ID only for byte-equivalent content. Report a conflicting
ID and preserve the first artifact; never overwrite it.

## Ownership boundary

Writing a council artifact does not claim work, release a root lease, or
advance a root workflow state. Use the existing Shark Rider, sprint, notes,
context, and claim procedures for those operations.

## Result

A routine question stays in the E39 `Q###` loop and creates no
`docs/council/` artifact. A material question produces exactly one
immutable-audit-trail decision/handoff/escalation record under
`docs/council/`, routed through this file alone. The owning coordinator
retains workflow-transition and root-lease authority throughout.
