# Direct: single-worker dispatch

## Goal

Execute work that `coordinate.md` classified as `Direct` coordination
level: one bounded entity, one worker. Direct dispatch performs no
topology-selection step of its own — one worker is the entire dispatch
shape, so there is no wave to shape and no ownership or isolation
evidence to check. Use `batch.md`/`execute-wave.md` instead once more
than one worker is dispatched concurrently for related work.

## Procedure

1. Confirm the classified work is a single dispatchable key — a task,
   feature, epic, bug, change-card, or tech-debt item resolved through
   keyed dispatch (see the route-based workflow guide's claim/session
   lease section) or through a role-aware self-pull
   (`pull-by-role.md`'s sanctioned path).
2. Run the ordinary Rider dispatch loop against that one key: claim,
   hand off the returned prompt to exactly one worker, apply the
   configured outcome, release. `skills/shark-rider/verbs/run.md` owns
   this loop's mechanics; do not restate them here.
3. If that loop instead returns a fork response, the fork is not a
   Direct dispatch — it is a selection among several independently
   dispatchable children. Re-classify each chosen candidate through
   `coordinate.md` rather than assuming Direct still applies to it.

## Result

Exactly one worker is claimed, dispatched, and released for the
classified entity. No topology axis is consulted, because a single-worker
dispatch has no wave to shape.
