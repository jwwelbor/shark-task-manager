# Worker ownership boundary

## Select the execution mode before claiming

The worker-owned child mode is not `/shark-rider run`. Use it only when an
existing coordinator explicitly delegates an authorized child and owns that
separate child-lease lifecycle. Do not hand its session to the Rider loop.

For `/shark-rider run`, the Rider parent calls `shark next` and claims the
returned concrete entity before dispatching `response.prompt`. A role-aware
self-pull supplies only a selected key to `/shark-rider run`; it never claims
or executes the `BacklogItemView` selection directly. A Rider-dispatched worker
never claims, heartbeats, releases, or selects a replacement entity. It returns
bounded evidence and a semantic outcome for the parent to persist and route.

## Worker-owned child mode: parent coordinator owns the root

The parent coordinator retains the root lease and is the only actor that may
perform a root heartbeat, root release, or root workflow transition. A child
worker must not call root `status set`, force-claim a root or another child,
advance root status, or otherwise mutate the dispatched root workflow state.

## Worker-owned child mode: child worker may act only within its authorization

After the existing sprint and claim authorities return an authorized child, the
worker may:

- Read bounded state for the root and its authorized child.
- Claim, heartbeat and release only its own child lease through the owning
  claim path.
- Write a scoped council artifact or Shark evidence pointer for that child.
- Return a semantic outcome and bounded evidence pointer to the parent
  coordinator.

The worker may not claim an unrelated child, write outside its scoped artifact
path, mutate another worker's lease, or choose a status transition. The parent
uses the returned evidence and semantic outcome to decide any configured root
or child workflow advance.

## Worker-owned child mode: required handoff

Return only the information the parent needs to continue safely:

- root key and authorized child key;
- the child claim/session identity when available;
- semantic outcome, such as completion, blocked, or needs review;
- bounded evidence/artifact pointers and a concise reason; and
- any blocker that requires coordinator or council review.

Do not return rendered prompts, credentials, access tokens, unrestricted worker
output, secret-bearing file content, or arbitrary filesystem paths. Report an
attempted out-of-scope action as a bounded blocker rather than bypassing the
authority boundary.

## Result

In worker-owned child mode, the child lease is managed only by its owner,
evidence remains scoped and safe, and the parent coordinator can retain its
root lease and perform the single authorized workflow transition. In
`/shark-rider run`, the Rider parent owns the dispatched entity's lease and
workflow transition from selection through release.
