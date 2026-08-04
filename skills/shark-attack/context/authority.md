# Parent, worker, and council authority boundary

## Purpose

Every role in the protocol — parent, worker, and council — has a distinct,
non-overlapping authority. This file is the single place that boundary is
stated; `workflows/route-question.md`, `workflows/council.md`, and
`context/worker-ownership.md` document the procedures that exercise it, not
a second copy of the rule.

## The three roles

- **Parent** — the Rider loop or an equivalent dispatching coordinator. It is
  the only role that ever claims, heartbeats, releases, or transitions Shark
  workflow state. It retains that authority for the whole life of a
  dispatched entity, including every consultation opened while that entity
  is claimed.
- **Worker** — the dispatched agent doing the requested craft. A worker
  never runs a Shark mutation command (ADR-005). It returns its work as
  bounded evidence and a semantic outcome; when a question needs a
  consultation, it returns that question as structured text in its control
  envelope (`context/worker-control-schema.yaml`) and stops — the parent
  materializes and routes it.
- **Council role** — a roster member (`context/roster-schema.yaml`) who
  facilitates or answers a routed question. Facilitating a decision,
  chairing a council session, or answering a routed question grants no
  claim, lease, or workflow-transition authority of its own. The parent
  still performs the one authorized transition once evidence and an outcome
  exist.

## During a live consultation the parent MUST

- keep the dispatched entity's own claim alive by heartbeat for the whole
  consultation. That lease is a separate concern from any lease the
  consultation itself opens on a routed Question, and both are owned
  exactly as their own procedures document.
- bound the consultation by the dispatched entity's remaining claim lease.
- never advance or set the dispatched entity's status while a
  `question_blocks` Question is open or answering. This is a structural
  hold, not a prose discipline: the gate is enforced at the status-advance
  path, at keyed dispatch, at the run preflight, and in the runner. The
  direct status-set path is the documented human escape hatch and has no
  place on this loop.

## On heartbeat failure or lease loss the parent MUST

- immediately stop active mutation workers;
- never deliver a council answer, integrate changes, or transition the
  entity under the lost authority;
- treat any handoff produced under a lost lease as context only — it
  supplies information for the next attempt but never restores authority on
  its own.

Recovery requires a fresh keyed dispatch of the affected entity followed by
a new, successful claim. Releasing a stolen or expired claim is necessary
but not sufficient — a session that no longer matches any live claim stays
rejected until a fresh claim exists under it.

## If a routed responder is silent, fails, or disappears

The parent pings once, interrupts or cancels the stale consultation where
the host supports it, and routes to one replacement responder. If no
qualified responder returns before the consultation deadline, the parent
stops write workers, records a bounded unresolved handoff, records the
blocker, and releases the lease. A claim must never be held indefinitely.

## Category → standing role

`context/worker-control-schema.yaml`'s question envelope carries a
`category` value: `product`, `requirements`, `architecture`, `quality`, or
`process`. `workflows/council.md` owns which *path* a category defaults to
(the routine Question loop or the council escalation procedure); this
section instead names which roster role is the standing authority for each
category once a responder is needed, so routing never falls to an
unassigned or ad hoc identity:

| Category | Standing role |
|---|---|
| `product` | Product Manager |
| `requirements` | Business Analyst |
| `architecture` | Architect |
| `quality` | Quality Analyst |
| `process` | Scrum Master |

A project roster may substitute its own member for a standing role's
built-in persona, but every category still resolves to exactly one role —
never to an unresolved or competing set of responders.

## Result

At every point in a consultation, exactly one role holds Shark workflow
authority (the parent), exactly one role does the requested craft (the
worker), and exactly one role answers or facilitates the question at hand
(the standing council role for its category). None of the three can acquire
another's authority by doing its own job well — a worker cannot claim by
returning a good answer, and a council role cannot transition status by
resolving a question. Only the parent's own claim, heartbeat, and
transition calls do that.
